// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sync provides basic synchronization primitives such as mutual
// exclusion locks. Other than the Once and WaitGroup types, most are intended
// for use by low-level library routines. Higher-level synchronization is
// better done via channels and communication.
//
// Values containing the types defined in this package should not be copied.
//
// Package sync 提供了基础同步原语，如互斥锁。
// 除了 Once 和 WaitGroup 类型，大多数类型都适用于低级库例程（go routines）。
// 通过 channels 和 communication 可以更好地完成更高级别的同步。
//
// 定义在此包中的类型值不应该被复制。
package sync

import (
	"internal/race"
	"sync/atomic"
	"unsafe"
)

func throw(string) // provided by runtime // 由运行时提供

// A Mutex is a mutual exclusion lock.
// The zero value for a Mutex is an unlocked mutex.
//
// A Mutex must not be copied after first use.
//
// Mutex 是一个互斥锁。
// Mutex 的零值是一个 unlocked 状态的互斥锁。
//
// 在第一次使用后，一定不能复制 Mutex。
//
// IMP: 接收者为值时会发生隐式复制，如下所示
// func (t T) Do()
// 接收者必须为指针，如下所示
// func (t *T) Do()
//
// IMP: state 每一位的含义
// 31            3 2 1 0
// |             | | | |
// 1 1 1 ... 1 1 1 1 1 1
// \_____________/ | | |
// (4)等待队列长度 (3)(2)(1)
// (1) mutexLocked   -- mutex 是否被上锁，1 为被上锁，0 为未被上锁。
// (2) mutexWoken    -- 正常模式下，通知 Unlock 不唤醒其他被阻塞的 goroutines。
// (3) mutexStarving -- 是否处于饥饿模式。
// (4) 等待队列长度    -- 等待队列中 goroutine 的数量。
type Mutex struct {
	state int32
	sema  uint32
}

// A Locker represents an object that can be locked and unlocked.
//
// Locker 代表一个可以被上锁和解锁的对象。
type Locker interface {
	Lock()
	Unlock()
}

const (
	// mutexLocked = 1，mutex 被上锁
	mutexLocked = 1 << iota // mutex is locked
	// mutexWoken = 2
	mutexWoken
	// mutexStarving = 4
	mutexStarving
	// mutexWaiterShift = 3
	mutexWaiterShift = iota

	// Mutex fairness.
	//
	// Mutex can be in 2 modes of operations: normal and starvation.
	// In normal mode waiters are queued in FIFO order, but a woken up waiter
	// does not own the mutex and competes with new arriving goroutines over
	// the ownership. New arriving goroutines have an advantage -- they are
	// already running on CPU and there can be lots of them, so a woken up
	// waiter has good chances of losing. In such case it is queued at front
	// of the wait queue. If a waiter fails to acquire the mutex for more than 1ms,
	// it switches mutex to the starvation mode.
	//
	// In starvation mode ownership of the mutex is directly handed off from
	// the unlocking goroutine to the waiter at the front of the queue.
	// New arriving goroutines don't try to acquire the mutex even if it appears
	// to be unlocked, and don't try to spin. Instead they queue themselves at
	// the tail of the wait queue.
	//
	// If a waiter receives ownership of the mutex and sees that either
	// (1) it is the last waiter in the queue, or (2) it waited for less than 1 ms,
	// it switches mutex back to normal operation mode.
	//
	// Normal mode has considerably better performance as a goroutine can acquire
	// a mutex several times in a row even if there are blocked waiters.
	// Starvation mode is important to prevent pathological cases of tail latency.
	//
	// 互斥公平
	//
	// Mutex 可以处于两种操作模式：正常和饥饿。
	// 在正常模式下，等待的 goroutine 按 FIFO 顺序排队，但是一个被唤醒的等待 goroutine 并没用
	// 拥有互斥锁，并且与所有新到的 goroutines 竞争所有权。新到的 goroutines 有一个优势 -- 它
	// 们已经在 CPU 上运行，并且它们的数量可以很多，所以一个被唤醒的等待 goroutine 很有可能会失
	// 败。在这种情况下，它将排在等待队列的前面。如果一个等待 goroutine 获取互斥锁失败超过 1ms，
	// 它会将互斥锁切换到饥饿模式。
	//
	// 在饥饿模式下，互斥锁的所有权直接从解锁的 goroutine 移交给等待队列前面的 goroutine。即使互
	// 斥锁被解锁，新到的 goroutine 也不会尝试获取互斥锁，并且不会尝试自旋。相反，它们将排在等待队
	// 列的末尾。
	//
	// 如果等待的 goroutine 获得到互斥锁的所有权并且观察到下面两条规则中的任何一条，它会将互斥锁切
	// 换回正常操作模式。
	// (1) 它是队列中最后一个等待的 goroutine。
	// (2) 它等待的时间小于 1ms。
	//
	// 正常模式具有相当好的性能，因为一个 goroutine 可以连续多次获取互斥锁，即使存在被阻塞的其他在
	// 等待的 goroutine。
	starvationThresholdNs = 1e6
)

// Lock locks m.
// If the lock is already in use, the calling goroutine
// blocks until the mutex is available.
//
// Lock 将 m 上锁。
// 如果 lock 已经在使用，调用的 goroutine 将阻塞到 mutex 可用。
func (m *Mutex) Lock() {
	// Fast path: grab unlocked mutex.
	//
	// 快速途径：获取 unlocked 状态到 mutex。
	// IMP: m.state 等于 0 时，将 m.state 置为 1（mutexLocked），完成上锁过程。
	if atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked) {
		if race.Enabled {
			race.Acquire(unsafe.Pointer(m))
		}
		return
	}

	var waitStartTime int64
	starving := false
	awoke := false
	iter := 0 // 自旋次数，用于 runtime_canSpin 判断自旋是否有意义
	old := m.state
	for {
		// Don't spin in starvation mode, ownership is handed off to waiters
		// so we won't be able to acquire the mutex anyway.
		//
		// 在饥饿模式下，不能自旋，CPU 所有权移交给等待的 goroutine，所以我们将不能获得到互斥锁。
		//
		// old&(mutexLocked|mutexStarving) == mutexLocked 判断是否为饥饿模式，饥饿模式下为 false。
		//runtime_canSpin(iter) 判断是否能进行自旋。
		if old&(mutexLocked|mutexStarving) == mutexLocked && runtime_canSpin(iter) {
			// Active spinning makes sense.
			// Try to set mutexWoken flag to inform Unlock
			// to not wake other blocked goroutines.
			//
			// 主动自旋是有意义的。
			// 尝试设置 mutexWoken 标志以通知 Unlock 不唤醒其他被阻塞的 goroutines。
			//
			// !awoke 此 goroutine 未设置 mutexWoken。
			// old&mutexWoken == 0 此前，没有其他 goroutine 设置 mutexWoken。
			// old>>mutexWaiterShift != 0 判断等待队列是否为空。
			// atomic.CompareAndSwapInt32(&m.state, old, old|mutexWoken) &m.state 未被其他 goroutine 更新。
			if !awoke && old&mutexWoken == 0 && old>>mutexWaiterShift != 0 &&
				atomic.CompareAndSwapInt32(&m.state, old, old|mutexWoken) {
				awoke = true
			}
			runtime_doSpin()
			iter++
			old = m.state
			continue
		}
		new := old
		// Don't try to acquire starving mutex, new arriving goroutines must queue.
		//
		// 不试图获取饥饿模式的互斥锁，新到的 goroutines 必须排队。
		//
		// 正常模式下，new 上锁。
		if old&mutexStarving == 0 {
			new |= mutexLocked
		}
		// 已经上锁或处于饥饿模式，当前的 goroutine 加入等待队列。
		if old&(mutexLocked|mutexStarving) != 0 {
			new += 1 << mutexWaiterShift
		}
		// The current goroutine switches mutex to starvation mode.
		// But if the mutex is currently unlocked, don't do the switch.
		// Unlock expects that starving mutex has waiters, which will not
		// be true in this case.
		//
		// 当前的 goroutine 将互斥锁切换到饥饿模式。
		// 但是如果互斥锁当前处于 unlocked 状态，就不做切换。
		// TSK: Unlock 期待饥饿互斥锁有等待者，但在这种情况下不会。
		if starving && old&mutexLocked != 0 {
			new |= mutexStarving
		}
		if awoke {
			// The goroutine has been woken from sleep,
			// so we need to reset the flag in either caseo.
			if new&mutexWoken == 0 {
				throw("sync: inconsistent mutex state")
			}
			new &^= mutexWoken
		}
		if atomic.CompareAndSwapInt32(&m.state, old, new) {
			if old&(mutexLocked|mutexStarving) == 0 {
				break // locked the mutex with CAS
			}
			// If we were already waiting before, queue at the front of the queue.
			queueLifo := waitStartTime != 0
			if waitStartTime == 0 {
				waitStartTime = runtime_nanotime()
			}
			runtime_SemacquireMutex(&m.sema, queueLifo)
			starving = starving || runtime_nanotime()-waitStartTime > starvationThresholdNs
			old = m.state
			if old&mutexStarving != 0 {
				// If this goroutine was woken and mutex is in starvation mode,
				// ownership was handed off to us but mutex is in somewhat
				// inconsistent state: mutexLocked is not set and we are still
				// accounted as waiter. Fix that.
				if old&(mutexLocked|mutexWoken) != 0 || old>>mutexWaiterShift == 0 {
					throw("sync: inconsistent mutex state")
				}
				delta := int32(mutexLocked - 1<<mutexWaiterShift)
				if !starving || old>>mutexWaiterShift == 1 {
					// Exit starvation mode.
					// Critical to do it here and consider wait time.
					// Starvation mode is so inefficient, that two goroutines
					// can go lock-step infinitely once they switch mutex
					// to starvation mode.
					delta -= mutexStarving
				}
				atomic.AddInt32(&m.state, delta)
				break
			}
			awoke = true
			iter = 0
		} else {
			old = m.state
		}
	}

	if race.Enabled {
		race.Acquire(unsafe.Pointer(m))
	}
}

// Unlock unlocks m.
// It is a run-time error if m is not locked on entry to Unlock.
//
// A locked Mutex is not associated with a particular goroutine.
// It is allowed for one goroutine to lock a Mutex and then
// arrange for another goroutine to unlock it.
//
// Unlock 将 m 解锁。
// 如果在解锁 m 前未被上锁，将会产生一个运行时错误。
// 一个被上锁的 Mutex 没有和特定的 goroutine 关联起来。
// 允许一个 goroutine 锁定 Mutex，然后安排另一个 goroutine 解锁。
func (m *Mutex) Unlock() {
	if race.Enabled {
		_ = m.state
		race.Release(unsafe.Pointer(m))
	}

	// Fast path: drop lock bit.
	//
	// 快速途径：减去 mutexLocked。
	new := atomic.AddInt32(&m.state, -mutexLocked)
	if (new+mutexLocked)&mutexLocked == 0 {
		throw("sync: unlock of unlocked mutex")
	}
	if new&mutexStarving == 0 {
		old := new
		for {
			// If there are no waiters or a goroutine has already
			// been woken or grabbed the lock, no need to wake anyone.
			// In starvation mode ownership is directly handed off from unlocking
			// goroutine to the next waiter. We are not part of this chain,
			// since we did not observe mutexStarving when we unlocked the mutex above.
			// So get off the way.
			if old>>mutexWaiterShift == 0 || old&(mutexLocked|mutexWoken|mutexStarving) != 0 {
				return
			}
			// Grab the right to wake someone.
			new = (old - 1<<mutexWaiterShift) | mutexWoken
			if atomic.CompareAndSwapInt32(&m.state, old, new) {
				runtime_Semrelease(&m.sema, false)
				return
			}
			old = m.state
		}
	} else {
		// Starving mode: handoff mutex ownership to the next waiter.
		// Note: mutexLocked is not set, the waiter will set it after wakeup.
		// But mutex is still considered locked if mutexStarving is set,
		// so new coming goroutines won't acquire it.
		runtime_Semrelease(&m.sema, true)
	}
}

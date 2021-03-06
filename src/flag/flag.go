// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package flag implements command-line flag parsing.

	Usage

	Define flags using flag.String(), Bool(), Int(), etc.

	This declares an integer flag, -flagname, stored in the pointer ip, with type *int.
		import "flag"
		var ip = flag.Int("flagname", 1234, "help message for flagname")
	If you like, you can bind the flag to a variable using the Var() functions.
		var flagvar int
		func init() {
			flag.IntVar(&flagvar, "flagname", 1234, "help message for flagname")
		}
	Or you can create custom flags that satisfy the Value interface (with
	pointer receivers) and couple them to flag parsing by
		flag.Var(&flagVal, "name", "help message for flagname")
	For such flags, the default value is just the initial value of the variable.

	After all flags are defined, call
		flag.Parse()
	to parse the command line into the defined flags.

	Flags may then be used directly. If you're using the flags themselves,
	they are all pointers; if you bind to variables, they're values.
		fmt.Println("ip has value ", *ip)
		fmt.Println("flagvar has value ", flagvar)

	After parsing, the arguments following the flags are available as the
	slice flag.Args() or individually as flag.Arg(i).
	The arguments are indexed from 0 through flag.NArg()-1.

	Command line flag syntax

	The following forms are permitted:

		-flag
		-flag=x
		-flag x  // non-boolean flags only
	One or two minus signs may be used; they are equivalent.
	The last form is not permitted for boolean flags because the
	meaning of the command
		cmd -x *
	where * is a Unix shell wildcard, will change if there is a file
	called 0, false, etc. You must use the -flag=false form to turn
	off a boolean flag.

	Flag parsing stops just before the first non-flag argument
	("-" is a non-flag argument) or after the terminator "--".

	Integer flags accept 1234, 0664, 0x1234 and may be negative.
	Boolean flags may be:
		1, 0, t, f, T, F, true, false, TRUE, FALSE, True, False
	Duration flags accept any input valid for time.ParseDuration.

	The default set of command-line flags is controlled by
	top-level functions.  The FlagSet type allows one to define
	independent sets of flags, such as to implement subcommands
	in a command-line interface. The methods of FlagSet are
	analogous to the top-level functions for the command-line
	flag set.
*/
/*
	flag 包实现了命令行标志解析。

	用法

	使用 flag.String()、Bool()、Int() 等函数定义标志。

	这声明了一个 int 型标志，-flagname，储存在指针 ip 中，类型为 *int。
		import "flag"
		var ip = flag.Int("flagname", 1234, "help message for flagname")
	如果你愿意，你可以使用 Var() 函数将标志绑定到一个变量上。
		var flagvar int
		func init() {
			flag.IntVar(&flagvar, "flagname", 1234, "help message for flagname")
		}
	或者你可以创建满足 Value 接口的自定义标志（带有指针接收者）并通过下面的函数将它们结合起来
		flag.Var(&flagVal, "name", "help message for flagname")
	对于此类标志，默认值只是变量的初始值。

	在定义完所有标志后，调用
		flag.Parse()
	将命令行解析到定义的标志上。

	然后标志可以直接使用。如果你使用标志本身，它们都是指针。如果你将其绑定到变量上，它们就是值。
		fmt.Println("ip has value ", *ip)
		fmt.Println("flagvar has value ", flagvar)

	解析后，通过 flag.Args() 可以获取非标志参数切片或者通过 flag.Arg(i) 获取指定非标志参数。
	参数索引为 0 到 flag.NArg()-1。

	命令行标志语法

	下面的格式是允许的：

		-flag
		-flag=x
		-flag x  //  仅限非布尔标志
	可以使用一个或两个破折号，它们是等价的。
	最后一种的格式不允许使用布尔标志，因为命令的含义
		cmd -x *
	其中 * 是 Unix shell 通配符，如果存在文件叫 0、false 等，它将会发生更改。你必须使用
	-flag=false 的格式来关闭布尔标志。

	标志在第一个非标志参数（"-" 是一个非标志参数）之前或终结符 "--" 之后停止解析。

	int 型标志可以接受 1234、0664、0x1234 以及负数。
	bool 型标志可以接受：
		1、0、t、f、T、F、true、false、TRUE、FALSE、True、False
	duration 型标志可以接受对 time.ParseDuration 有效对任何输入。

	默认的命令行标志集由顶层函数控制。FlagSet 类型允许定义独立对标志集，例如在命令行接口中
	实现子命令。FlagSet 的方法类似于命令行标志集的顶层函数。
*/
package flag

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrHelp is the error returned if the -help or -h flag is invoked
// but no such flag is defined.
//
// ErrHelp 在标志 -help 或 -h 被调用，或者使用了没有定义的标志时返回此错误。
var ErrHelp = errors.New("flag: help requested")

// -- bool Value
type boolValue bool

func newBoolValue(val bool, p *bool) *boolValue {
	*p = val
	return (*boolValue)(p)
}

func (b *boolValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	*b = boolValue(v)
	return err
}

func (b *boolValue) Get() interface{} { return bool(*b) }

func (b *boolValue) String() string { return strconv.FormatBool(bool(*b)) }

func (b *boolValue) IsBoolFlag() bool { return true }

// optional interface to indicate boolean flags that can be
// supplied without "=value" text
//
// 可选的接口，用于指明 bool 型标志，它在没有 "=value" 文本被提供
type boolFlag interface {
	Value
	IsBoolFlag() bool
}

// -- int Value
type intValue int

func newIntValue(val int, p *int) *intValue {
	*p = val
	return (*intValue)(p)
}

func (i *intValue) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, strconv.IntSize)
	*i = intValue(v)
	return err
}

func (i *intValue) Get() interface{} { return int(*i) }

func (i *intValue) String() string { return strconv.Itoa(int(*i)) }

// -- int64 Value
type int64Value int64

func newInt64Value(val int64, p *int64) *int64Value {
	*p = val
	return (*int64Value)(p)
}

func (i *int64Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 64)
	*i = int64Value(v)
	return err
}

func (i *int64Value) Get() interface{} { return int64(*i) }

func (i *int64Value) String() string { return strconv.FormatInt(int64(*i), 10) }

// -- uint Value
type uintValue uint

func newUintValue(val uint, p *uint) *uintValue {
	*p = val
	return (*uintValue)(p)
}

func (i *uintValue) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, strconv.IntSize)
	*i = uintValue(v)
	return err
}

func (i *uintValue) Get() interface{} { return uint(*i) }

func (i *uintValue) String() string { return strconv.FormatUint(uint64(*i), 10) }

// -- uint64 Value
type uint64Value uint64

func newUint64Value(val uint64, p *uint64) *uint64Value {
	*p = val
	return (*uint64Value)(p)
}

func (i *uint64Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 64)
	*i = uint64Value(v)
	return err
}

func (i *uint64Value) Get() interface{} { return uint64(*i) }

func (i *uint64Value) String() string { return strconv.FormatUint(uint64(*i), 10) }

// -- string Value
type stringValue string

func newStringValue(val string, p *string) *stringValue {
	*p = val
	return (*stringValue)(p)
}

func (s *stringValue) Set(val string) error {
	*s = stringValue(val)
	return nil
}

func (s *stringValue) Get() interface{} { return string(*s) }

func (s *stringValue) String() string { return string(*s) }

// -- float64 Value
type float64Value float64

func newFloat64Value(val float64, p *float64) *float64Value {
	*p = val
	return (*float64Value)(p)
}

func (f *float64Value) Set(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	*f = float64Value(v)
	return err
}

func (f *float64Value) Get() interface{} { return float64(*f) }

func (f *float64Value) String() string { return strconv.FormatFloat(float64(*f), 'g', -1, 64) }

// -- time.Duration Value
type durationValue time.Duration

func newDurationValue(val time.Duration, p *time.Duration) *durationValue {
	*p = val
	return (*durationValue)(p)
}

func (d *durationValue) Set(s string) error {
	v, err := time.ParseDuration(s)
	*d = durationValue(v)
	return err
}

func (d *durationValue) Get() interface{} { return time.Duration(*d) }

func (d *durationValue) String() string { return (*time.Duration)(d).String() }

// Value is the interface to the dynamic value stored in a flag.
// (The default value is represented as a string.)
//
// If a Value has an IsBoolFlag() bool method returning true,
// the command-line parser makes -name equivalent to -name=true
// rather than using the next command-line argument.
//
// Set is called once, in command line order, for each flag present.
// The flag package may call the String method with a zero-valued receiver,
// such as a nil pointer.
//
// Value 是操作存储在标志中动态值的接口。（默认值表示为字符串）
//
// 如果 Value 的 IsBoolFlag() bool 方法返回 true，命令行解析器将使 -name 等效于 -name=true，
// 所以使用 -name 而不是使用 -name=true 形式的命令行参数。
//
// 对于每个存在的标志，以命令行的顺序调用一次 Set。
// flag 包可以使用零值接受者（例如空指针）调用 String 方法。
type Value interface {
	String() string
	Set(string) error
}

// Getter is an interface that allows the contents of a Value to be retrieved.
// It wraps the Value interface, rather than being part of it, because it
// appeared after Go 1 and its compatibility rules. All Value types provided
// by this package satisfy the Getter interface.
//
// Getter 是一个允许获取 Value 内容的接口。它包含 Value 接口，而不是它的一部分，因为它出现
// 在 Go 1 之后以及兼容性规则。此包提供的所有 Value 类型都满足 Getter 接口。
type Getter interface {
	Value
	Get() interface{}
}

// ErrorHandling defines how FlagSet.Parse behaves if the parse fails.
//
// ErrorHandling 定义了 FlagSet.Parse 解析失败后的行为。
type ErrorHandling int

// These constants cause FlagSet.Parse to behave as described if the parse fails.
//
// 如果解析失败，FlagSet.Parse 的行为与常量描述的相同。
const (
	// 返回描述性错误。
	ContinueOnError ErrorHandling = iota // Return a descriptive error.
	// 调用 os.Exit(2)。
	ExitOnError // Call os.Exit(2).
	// 使用描述性错误调用 panic。
	PanicOnError // Call panic with a descriptive error.
)

// A FlagSet represents a set of defined flags. The zero value of a FlagSet
// has no name and has ContinueOnError error handling.
//
// FlagSet 表示一组定义的标志。FlagSet 的零值没有名称并且错误处理等级为 ContinueOnError。
type FlagSet struct {
	// Usage is the function called when an error occurs while parsing flags.
	// The field is a function (not a method) that may be changed to point to
	// a custom error handler. What happens after Usage is called depends
	// on the ErrorHandling setting; for the command line, this defaults
	// to ExitOnError, which exits the program after calling Usage.
	//
	// Usage 当解析标志发生错误时会调用此函数。
	// 此字段是一个函数（不是一个方法），它可以被更改指向一个自定义的错误处理函数。
	// 在 Usage 被调用后会发生什么依赖于 ErrorHandling 的设置，对于命令行，默认设置是
	// ExitOnError，它将会在调用 Usage 后退出。
	Usage func()

	name   string
	parsed bool
	actual map[string]*Flag
	formal map[string]*Flag
	// 标志后的参数（更多请看 111 行注释）
	args          []string // arguments after flags
	errorHandling ErrorHandling
	// nil 意味着是 stderr，使用 out() 访问器
	output io.Writer // nil means stderr; use out() accessor
}

// A Flag represents the state of a flag.
// Flag 表示标志的状态。
type Flag struct {
	// Name 为它出现在命令行上的名称
	Name string // name as it appears on command line
	// Usage 是帮助信息
	Usage string // help message
	// Value 为集合
	Value Value // value as set
	// 默认值（为文本），提供给帮助信息使用
	DefValue string // default value (as text); for usage message
}

// sortFlags returns the flags as a slice in lexicographical sorted order.
//
// sortFlags 返回字典序排列的标志切片。
func sortFlags(flags map[string]*Flag) []*Flag {
	list := make(sort.StringSlice, len(flags))
	i := 0
	for _, f := range flags {
		list[i] = f.Name
		i++
	}
	list.Sort()
	result := make([]*Flag, len(list))
	for i, name := range list {
		result[i] = flags[name]
	}
	return result
}

// Output returns the destination for usage and error messages. os.Stderr is returned if
// output was not set or was set to nil.
//
// Output 返回用法和错误信息的写入位置。如果输出未被设置或被设置为 nil 时，返回 os.Stderr。
func (f *FlagSet) Output() io.Writer {
	if f.output == nil {
		return os.Stderr
	}
	return f.output
}

// Name returns the name of the flag set.
//
// Name 返回标志集的名称。
func (f *FlagSet) Name() string {
	return f.name
}

// ErrorHandling returns the error handling behavior of the flag set.
//
// ErrorHandling 返回标志集的错误处理行为。
func (f *FlagSet) ErrorHandling() ErrorHandling {
	return f.errorHandling
}

// SetOutput sets the destination for usage and error messages.
// If output is nil, os.Stderr is used.
//
// SetOutput 设置用法和错误信息的写入位置。如果输出为 nil，将使用 os.Stderr。
func (f *FlagSet) SetOutput(output io.Writer) {
	f.output = output
}

// VisitAll visits the flags in lexicographical order, calling fn for each.
// It visits all flags, even those not set.
//
// VisitAll 以字典序访问标志，并为每个标志调用 fn。它会访问所有标志，即使用户未设置它。
func (f *FlagSet) VisitAll(fn func(*Flag)) {
	for _, flag := range sortFlags(f.formal) {
		fn(flag)
	}
}

// VisitAll visits the command-line flags in lexicographical order, calling
// fn for each. It visits all flags, even those not set.
//
// VisitAll 以字典序访问命令行标志，并为每个标志调用 fn。它会访问所有标志，即使用户未设置它。
func VisitAll(fn func(*Flag)) {
	CommandLine.VisitAll(fn)
}

// Visit visits the flags in lexicographical order, calling fn for each.
// It visits only those flags that have been set.
//
// Visit 以字典序访问标志，并为每个标志调用 fn。它只访问被用户设置过的标志。
func (f *FlagSet) Visit(fn func(*Flag)) {
	for _, flag := range sortFlags(f.actual) {
		fn(flag)
	}
}

// Visit visits the command-line flags in lexicographical order, calling fn
// for each. It visits only those flags that have been set.
//
// Visit 以字典序访问命令行标志，并为每个标志调用 fn。它只访问被用户设置过的标志。
func Visit(fn func(*Flag)) {
	CommandLine.Visit(fn)
}

// Lookup returns the Flag structure of the named flag, returning nil if none exists.
//
// Lookup 返回 name 标志对应到 Flag 结构体，如果不存在返回 nil。
func (f *FlagSet) Lookup(name string) *Flag {
	return f.formal[name]
}

// Lookup returns the Flag structure of the named command-line flag,
// returning nil if none exists.
//
// Lookup 返回命令行标志中 name 标志对应到 Flag 结构体，如果不存在返回 nil。
func Lookup(name string) *Flag {
	return CommandLine.formal[name]
}

// Set sets the value of the named flag.
//
// Set 设置 name 标志的值。
func (f *FlagSet) Set(name, value string) error {
	flag, ok := f.formal[name]
	if !ok {
		return fmt.Errorf("no such flag -%v", name)
	}
	err := flag.Value.Set(value)
	if err != nil {
		return err
	}
	if f.actual == nil {
		f.actual = make(map[string]*Flag)
	}
	f.actual[name] = flag
	return nil
}

// Set sets the value of the named command-line flag.
//
// Set 设置命令行标志中 name 标志的值。
func Set(name, value string) error {
	return CommandLine.Set(name, value)
}

// isZeroValue determines whether the string represents the zero
// value for a flag.
//
// isZeroValue 判断字符串是否为标志的零值。
func isZeroValue(flag *Flag, value string) bool {
	// Build a zero value of the flag's Value type, and see if the
	// result of calling its String method equals the value passed in.
	// This works unless the Value type is itself an interface type.
	//
	// 构造一个标志 Value 类型的零值，并检验用其 String 方法得到的结果和传入的 value 是否相等。
	// 除非 Value 类型本身是一个接口类型，否则此方法有效。
	// TSK:// IMP:
	typ := reflect.TypeOf(flag.Value)
	var z reflect.Value
	if typ.Kind() == reflect.Ptr {
		z = reflect.New(typ.Elem())
	} else {
		z = reflect.Zero(typ)
	}
	return value == z.Interface().(Value).String()
}

// UnquoteUsage extracts a back-quoted name from the usage
// string for a flag and returns it and the un-quoted usage.
// Given "a `name` to show" it returns ("name", "a name to show").
// If there are no back quotes, the name is an educated guess of the
// type of the flag's value, or the empty string if the flag is boolean.
//
// UnquoteUsage 从用法信息中提取引号中的 name，并将它和去除引号的用法信息返回。
// 给出 "a `name` to show"，将返回 "name" 和 "a name to show"。
// 如果没有引号，则该名称是对标志值类型有依据的猜测。
// 如果该标志是布尔值，则为空字符串。
func UnquoteUsage(flag *Flag) (name string, usage string) {
	// Look for a back-quoted name, but avoid the strings package.
	//
	// 查找引号中的 name，但要避免使用 strings 包。
	usage = flag.Usage
	for i := 0; i < len(usage); i++ {
		if usage[i] == '`' {
			for j := i + 1; j < len(usage); j++ {
				if usage[j] == '`' {
					name = usage[i+1 : j]
					usage = usage[:i] + name + usage[j+1:]
					return name, usage
				}
			}
			// 只有一个引号，进入类型 name 的情况。
			break // Only one back quote; use type name.
		}
	}
	// No explicit name, so use type if we can find one.
	//
	// 没有明确的 name，所以使用我们可以找到的类型。
	name = "value"
	switch flag.Value.(type) {
	case boolFlag:
		name = ""
	case *durationValue:
		name = "duration"
	case *float64Value:
		name = "float"
	case *intValue, *int64Value:
		name = "int"
	case *stringValue:
		name = "string"
	case *uintValue, *uint64Value:
		name = "uint"
	}
	return
}

// PrintDefaults prints, to standard error unless configured otherwise, the
// default values of all defined command-line flags in the set. See the
// documentation for the global function PrintDefaults for more information.
//
// PrintDefaults 打印集合中所有已定义的命令行标志的默认值到标志错误输出，除非另有配置。
// 更多信息请查看全局函数 PrintDefaults 的文档。
func (f *FlagSet) PrintDefaults() {
	f.VisitAll(func(flag *Flag) {
		// 前面有两个空格，看下面两条注释
		s := fmt.Sprintf("  -%s", flag.Name) // Two spaces before -; see next two comments.
		name, usage := UnquoteUsage(flag)
		if len(name) > 0 {
			s += " " + name
		}
		// Boolean flags of one ASCII letter are so common we
		// treat them specially, putting their usage on the same line.
		//
		// 单个 ASCII 码字母的 bool 型标志是如此常见。我们特殊对待此类标志，将它们的
		// 用法信息在同一行输出。
		// 看上一条注释可以知道格式为，空格、空格、'-'、字母。
		if len(s) <= 4 { // space, space, '-', 'x'.
			s += "\t"
		} else {
			// Four spaces before the tab triggers good alignment
			// for both 4- and 8-space tab stops.
			//
			// 前有四个空格对于 4 个或 8 个空格的 tab 符都能有更好的对齐效果。
			s += "\n    \t"
		}
		s += strings.Replace(usage, "\n", "\n    \t", -1)

		if !isZeroValue(flag, flag.DefValue) {
			if _, ok := flag.Value.(*stringValue); ok {
				// put quotes on the value
				//
				// 值中存在引号
				s += fmt.Sprintf(" (default %q)", flag.DefValue)
			} else {
				s += fmt.Sprintf(" (default %v)", flag.DefValue)
			}
		}
		fmt.Fprint(f.Output(), s, "\n")
	})
}

// PrintDefaults prints, to standard error unless configured otherwise,
// a usage message showing the default settings of all defined
// command-line flags.
// For an integer valued flag x, the default output has the form
//	-x int
//		usage-message-for-x (default 7)
// The usage message will appear on a separate line for anything but
// a bool flag with a one-byte name. For bool flags, the type is
// omitted and if the flag name is one byte the usage message appears
// on the same line. The parenthetical default is omitted if the
// default is the zero value for the type. The listed type, here int,
// can be changed by placing a back-quoted name in the flag's usage
// string; the first such item in the message is taken to be a parameter
// name to show in the message and the back quotes are stripped from
// the message when displayed. For instance, given
//	flag.String("I", "", "search `directory` for include files")
// the output will be
//	-I directory
//		search directory for include files.
//
// PrintDefaults 打印集合中所有已定义的命令行标志的默认值到标志错误输出，除非另有配置。
// 对于 int 型值的标志 x，默认输出的格式为
//	-x int
//		usage-message-for-x (default 7)
// 除了单字节名称的 bool 型标志外，所有标志的用法信息将以单独的行显示。对于 bool 型标志，
// 类型会被省略。并且，如果标志名是一个字节的，则所有信息出现在同一行。如果默认值是类型的零
// 值，则括号中的默认值会被省略。列出的类型（这里是 int）可以通过在标志用法的字符串中放置
// 一个带有引号的名称来代替。信息中的第一个匹配项将被当作参数名显示在消息中，并且显示时将去
// 除消息中的引号。例如，给定
//	flag.String("I", "", "search `directory` for include files")
// 输出结果为
//	-I directory
//		search directory for include files.
func PrintDefaults() {
	CommandLine.PrintDefaults()
}

// defaultUsage is the default function to print a usage message.
//
// defaultUsage 是打印用法信息的默认方法。
func (f *FlagSet) defaultUsage() {
	if f.name == "" {
		fmt.Fprintf(f.Output(), "Usage:\n")
	} else {
		fmt.Fprintf(f.Output(), "Usage of %s:\n", f.name)
	}
	f.PrintDefaults()
}

// NOTE: Usage is not just defaultUsage(CommandLine)
// because it serves (via godoc flag Usage) as the example
// for how to write your own usage function.

// Usage prints a usage message documenting all defined command-line flags
// to CommandLine's output, which by default is os.Stderr.
// It is called when an error occurs while parsing flags.
// The function is a variable that may be changed to point to a custom function.
// By default it prints a simple header and calls PrintDefaults; for details about the
// format of the output and how to control it, see the documentation for PrintDefaults.
// Custom usage functions may choose to exit the program; by default exiting
// happens anyway as the command line's error handling strategy is set to
// ExitOnError.
//
// NOTE: Usage 不仅仅是默认用法（CommandLine），因为它（通过 godoc flag Usage）还可以做为如何编写自
// 己的 usage 函数的示例。

// Usage 打印用法信息到 CommandLine.output（默认为 os.Stderr），它包含了所有命令行定义的标志。当解析
// 标志出现错误时会调用此函数。此函数是一个变量，由此它可以被改变指向一个自定义的函数。默认情况下，它会打印
// 一个简单的标题并调用 PrintDefaults。有关输出格式及其控制方法的详细信息，请看 PrintDefaults 的文档。
// 自定义的 usage 函数可以选择退出程序。默认情况下会退出程序，因为命令行的错误处理策略为 ExitOnError。
var Usage = func() {
	fmt.Fprintf(CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	PrintDefaults()
}

// NFlag returns the number of flags that have been set.
func (f *FlagSet) NFlag() int { return len(f.actual) }

// NFlag returns the number of command-line flags that have been set.
func NFlag() int { return len(CommandLine.actual) }

// Arg returns the i'th argument. Arg(0) is the first remaining argument
// after flags have been processed. Arg returns an empty string if the
// requested element does not exist.
func (f *FlagSet) Arg(i int) string {
	if i < 0 || i >= len(f.args) {
		return ""
	}
	return f.args[i]
}

// Arg returns the i'th command-line argument. Arg(0) is the first remaining argument
// after flags have been processed. Arg returns an empty string if the
// requested element does not exist.
func Arg(i int) string {
	return CommandLine.Arg(i)
}

// NArg is the number of arguments remaining after flags have been processed.
func (f *FlagSet) NArg() int { return len(f.args) }

// NArg is the number of arguments remaining after flags have been processed.
func NArg() int { return len(CommandLine.args) }

// Args returns the non-flag arguments.
func (f *FlagSet) Args() []string { return f.args }

// Args returns the non-flag command-line arguments.
func Args() []string { return CommandLine.args }

// BoolVar defines a bool flag with specified name, default value, and usage string.
// The argument p points to a bool variable in which to store the value of the flag.
func (f *FlagSet) BoolVar(p *bool, name string, value bool, usage string) {
	f.Var(newBoolValue(value, p), name, usage)
}

// BoolVar defines a bool flag with specified name, default value, and usage string.
// The argument p points to a bool variable in which to store the value of the flag.
func BoolVar(p *bool, name string, value bool, usage string) {
	CommandLine.Var(newBoolValue(value, p), name, usage)
}

// Bool defines a bool flag with specified name, default value, and usage string.
// The return value is the address of a bool variable that stores the value of the flag.
func (f *FlagSet) Bool(name string, value bool, usage string) *bool {
	p := new(bool)
	f.BoolVar(p, name, value, usage)
	return p
}

// Bool defines a bool flag with specified name, default value, and usage string.
// The return value is the address of a bool variable that stores the value of the flag.
func Bool(name string, value bool, usage string) *bool {
	return CommandLine.Bool(name, value, usage)
}

// IntVar defines an int flag with specified name, default value, and usage string.
// The argument p points to an int variable in which to store the value of the flag.
func (f *FlagSet) IntVar(p *int, name string, value int, usage string) {
	f.Var(newIntValue(value, p), name, usage)
}

// IntVar defines an int flag with specified name, default value, and usage string.
// The argument p points to an int variable in which to store the value of the flag.
func IntVar(p *int, name string, value int, usage string) {
	CommandLine.Var(newIntValue(value, p), name, usage)
}

// Int defines an int flag with specified name, default value, and usage string.
// The return value is the address of an int variable that stores the value of the flag.
func (f *FlagSet) Int(name string, value int, usage string) *int {
	p := new(int)
	f.IntVar(p, name, value, usage)
	return p
}

// Int defines an int flag with specified name, default value, and usage string.
// The return value is the address of an int variable that stores the value of the flag.
func Int(name string, value int, usage string) *int {
	return CommandLine.Int(name, value, usage)
}

// Int64Var defines an int64 flag with specified name, default value, and usage string.
// The argument p points to an int64 variable in which to store the value of the flag.
func (f *FlagSet) Int64Var(p *int64, name string, value int64, usage string) {
	f.Var(newInt64Value(value, p), name, usage)
}

// Int64Var defines an int64 flag with specified name, default value, and usage string.
// The argument p points to an int64 variable in which to store the value of the flag.
func Int64Var(p *int64, name string, value int64, usage string) {
	CommandLine.Var(newInt64Value(value, p), name, usage)
}

// Int64 defines an int64 flag with specified name, default value, and usage string.
// The return value is the address of an int64 variable that stores the value of the flag.
func (f *FlagSet) Int64(name string, value int64, usage string) *int64 {
	p := new(int64)
	f.Int64Var(p, name, value, usage)
	return p
}

// Int64 defines an int64 flag with specified name, default value, and usage string.
// The return value is the address of an int64 variable that stores the value of the flag.
func Int64(name string, value int64, usage string) *int64 {
	return CommandLine.Int64(name, value, usage)
}

// UintVar defines a uint flag with specified name, default value, and usage string.
// The argument p points to a uint variable in which to store the value of the flag.
func (f *FlagSet) UintVar(p *uint, name string, value uint, usage string) {
	f.Var(newUintValue(value, p), name, usage)
}

// UintVar defines a uint flag with specified name, default value, and usage string.
// The argument p points to a uint variable in which to store the value of the flag.
func UintVar(p *uint, name string, value uint, usage string) {
	CommandLine.Var(newUintValue(value, p), name, usage)
}

// Uint defines a uint flag with specified name, default value, and usage string.
// The return value is the address of a uint variable that stores the value of the flag.
func (f *FlagSet) Uint(name string, value uint, usage string) *uint {
	p := new(uint)
	f.UintVar(p, name, value, usage)
	return p
}

// Uint defines a uint flag with specified name, default value, and usage string.
// The return value is the address of a uint variable that stores the value of the flag.
func Uint(name string, value uint, usage string) *uint {
	return CommandLine.Uint(name, value, usage)
}

// Uint64Var defines a uint64 flag with specified name, default value, and usage string.
// The argument p points to a uint64 variable in which to store the value of the flag.
func (f *FlagSet) Uint64Var(p *uint64, name string, value uint64, usage string) {
	f.Var(newUint64Value(value, p), name, usage)
}

// Uint64Var defines a uint64 flag with specified name, default value, and usage string.
// The argument p points to a uint64 variable in which to store the value of the flag.
func Uint64Var(p *uint64, name string, value uint64, usage string) {
	CommandLine.Var(newUint64Value(value, p), name, usage)
}

// Uint64 defines a uint64 flag with specified name, default value, and usage string.
// The return value is the address of a uint64 variable that stores the value of the flag.
func (f *FlagSet) Uint64(name string, value uint64, usage string) *uint64 {
	p := new(uint64)
	f.Uint64Var(p, name, value, usage)
	return p
}

// Uint64 defines a uint64 flag with specified name, default value, and usage string.
// The return value is the address of a uint64 variable that stores the value of the flag.
func Uint64(name string, value uint64, usage string) *uint64 {
	return CommandLine.Uint64(name, value, usage)
}

// StringVar defines a string flag with specified name, default value, and usage string.
// The argument p points to a string variable in which to store the value of the flag.
func (f *FlagSet) StringVar(p *string, name string, value string, usage string) {
	f.Var(newStringValue(value, p), name, usage)
}

// StringVar defines a string flag with specified name, default value, and usage string.
// The argument p points to a string variable in which to store the value of the flag.
func StringVar(p *string, name string, value string, usage string) {
	CommandLine.Var(newStringValue(value, p), name, usage)
}

// String defines a string flag with specified name, default value, and usage string.
// The return value is the address of a string variable that stores the value of the flag.
func (f *FlagSet) String(name string, value string, usage string) *string {
	p := new(string)
	f.StringVar(p, name, value, usage)
	return p
}

// String defines a string flag with specified name, default value, and usage string.
// The return value is the address of a string variable that stores the value of the flag.
func String(name string, value string, usage string) *string {
	return CommandLine.String(name, value, usage)
}

// Float64Var defines a float64 flag with specified name, default value, and usage string.
// The argument p points to a float64 variable in which to store the value of the flag.
func (f *FlagSet) Float64Var(p *float64, name string, value float64, usage string) {
	f.Var(newFloat64Value(value, p), name, usage)
}

// Float64Var defines a float64 flag with specified name, default value, and usage string.
// The argument p points to a float64 variable in which to store the value of the flag.
func Float64Var(p *float64, name string, value float64, usage string) {
	CommandLine.Var(newFloat64Value(value, p), name, usage)
}

// Float64 defines a float64 flag with specified name, default value, and usage string.
// The return value is the address of a float64 variable that stores the value of the flag.
func (f *FlagSet) Float64(name string, value float64, usage string) *float64 {
	p := new(float64)
	f.Float64Var(p, name, value, usage)
	return p
}

// Float64 defines a float64 flag with specified name, default value, and usage string.
// The return value is the address of a float64 variable that stores the value of the flag.
func Float64(name string, value float64, usage string) *float64 {
	return CommandLine.Float64(name, value, usage)
}

// DurationVar defines a time.Duration flag with specified name, default value, and usage string.
// The argument p points to a time.Duration variable in which to store the value of the flag.
// The flag accepts a value acceptable to time.ParseDuration.
func (f *FlagSet) DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	f.Var(newDurationValue(value, p), name, usage)
}

// DurationVar defines a time.Duration flag with specified name, default value, and usage string.
// The argument p points to a time.Duration variable in which to store the value of the flag.
// The flag accepts a value acceptable to time.ParseDuration.
func DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	CommandLine.Var(newDurationValue(value, p), name, usage)
}

// Duration defines a time.Duration flag with specified name, default value, and usage string.
// The return value is the address of a time.Duration variable that stores the value of the flag.
// The flag accepts a value acceptable to time.ParseDuration.
func (f *FlagSet) Duration(name string, value time.Duration, usage string) *time.Duration {
	p := new(time.Duration)
	f.DurationVar(p, name, value, usage)
	return p
}

// Duration defines a time.Duration flag with specified name, default value, and usage string.
// The return value is the address of a time.Duration variable that stores the value of the flag.
// The flag accepts a value acceptable to time.ParseDuration.
func Duration(name string, value time.Duration, usage string) *time.Duration {
	return CommandLine.Duration(name, value, usage)
}

// Var defines a flag with the specified name and usage string. The type and
// value of the flag are represented by the first argument, of type Value, which
// typically holds a user-defined implementation of Value. For instance, the
// caller could create a flag that turns a comma-separated string into a slice
// of strings by giving the slice the methods of Value; in particular, Set would
// decompose the comma-separated string into the slice.
//
// Var 使用指定的 name 和 usage 字符串定义一个标志。标志的类型和值由第一个参数表示，它通常包括用户
// 自定义的 Value 实现，类型为 Value。例如，调用者可以创建一个标志，通过给切片提供 Value 的方法，
// 将逗号分隔的字符串转化成字符串切片。尤其是 Set 能将逗号分隔的字符串分解成切片。
func (f *FlagSet) Var(value Value, name string, usage string) {
	// Remember the default value as a string; it won't change.
	//
	// 记住默认值是一个字符串，它不会改变。
	flag := &Flag{name, usage, value, value.String()}
	_, alreadythere := f.formal[name]
	if alreadythere {
		var msg string
		if f.name == "" {
			msg = fmt.Sprintf("flag redefined: %s", name)
		} else {
			msg = fmt.Sprintf("%s flag redefined: %s", f.name, name)
		}
		fmt.Fprintln(f.Output(), msg)
		// 仅在使用相同的名称声明标志时才会发生
		panic(msg) // Happens only if flags are declared with identical names
	}
	if f.formal == nil {
		f.formal = make(map[string]*Flag)
	}
	f.formal[name] = flag
}

// Var defines a flag with the specified name and usage string. The type and
// value of the flag are represented by the first argument, of type Value, which
// typically holds a user-defined implementation of Value. For instance, the
// caller could create a flag that turns a comma-separated string into a slice
// of strings by giving the slice the methods of Value; in particular, Set would
// decompose the comma-separated string into the slice.
//
// Var 使用指定的 name 和 usage 字符串定义一个标志。标志的类型和值由第一个参数表示，它通常包括用户
// 自定义的 Value 实现，类型为 Value。例如，调用者可以创建一个标志，通过给切片提供 Value 的方法，
// 将逗号分隔的字符串转化成字符串切片。尤其是 Set 能将逗号分隔的字符串分解成切片。
func Var(value Value, name string, usage string) {
	CommandLine.Var(value, name, usage)
}

// failf prints to standard error a formatted error and usage message and
// returns the error.
//
// failf 打印格式化的错误和用法信息到输出，并返回错误。
func (f *FlagSet) failf(format string, a ...interface{}) error {
	err := fmt.Errorf(format, a...)
	fmt.Fprintln(f.Output(), err)
	f.usage()
	return err
}

// usage calls the Usage method for the flag set if one is specified,
// or the appropriate default usage function otherwise.
//
// usage 如果指定了 Usage 方法，则使用调用标记集的 Usage 方法，否则调用相应的默认
// usage 函数。
func (f *FlagSet) usage() {
	if f.Usage == nil {
		f.defaultUsage()
	} else {
		f.Usage()
	}
}

// parseOne parses one flag. It reports whether a flag was seen.
//
// parseOne 解析一个标志。它还返回是否找到标志。
func (f *FlagSet) parseOne() (bool, error) {
	if len(f.args) == 0 {
		return false, nil
	}
	s := f.args[0]
	if len(s) < 2 || s[0] != '-' {
		return false, nil
	}
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		// "--" 终止标志
		if len(s) == 2 { // "--" terminates the flags
			f.args = f.args[1:]
			return false, nil
		}
	}
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, f.failf("bad flag syntax: %s", s)
	}

	// it's a flag. does it have an argument?
	//
	// 它是一个标志。它是否有参数？
	f.args = f.args[1:]
	hasValue := false
	value := ""
	// '=' 不会是第一个，即标志不会为 "-=xx"，上面已做过判断
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			value = name[i+1:]
			hasValue = true
			name = name[0:i]
			break
		}
	}
	m := f.formal
	flag, alreadythere := m[name] // BUG
	if !alreadythere {
		// 特殊情况：打印帮助信息
		if name == "help" || name == "h" { // special case for nice help message.
			f.usage()
			return false, ErrHelp
		}
		return false, f.failf("flag provided but not defined: -%s", name)
	}

	// 特殊情况：不需要参数
	if fv, ok := flag.Value.(boolFlag); ok && fv.IsBoolFlag() { // special case: doesn't need an arg
		if hasValue {
			if err := fv.Set(value); err != nil {
				return false, f.failf("invalid boolean value %q for -%s: %v", value, name, err)
			}
		} else {
			if err := fv.Set("true"); err != nil {
				return false, f.failf("invalid boolean flag %s: %v", name, err)
			}
		}
	} else {
		// It must have a value, which might be the next argument.
		//
		// 它必须有一个值，值可能是下一个参数。
		if !hasValue && len(f.args) > 0 {
			// value is the next arg
			//
			// 值是下一个参数
			hasValue = true
			value, f.args = f.args[0], f.args[1:]
		}
		if !hasValue {
			return false, f.failf("flag needs an argument: -%s", name)
		}
		if err := flag.Value.Set(value); err != nil {
			return false, f.failf("invalid value %q for flag -%s: %v", value, name, err)
		}
	}
	if f.actual == nil {
		f.actual = make(map[string]*Flag)
	}
	f.actual[name] = flag
	return true, nil
}

// Parse parses flag definitions from the argument list, which should not
// include the command name. Must be called after all flags in the FlagSet
// are defined and before flags are accessed by the program.
// The return value will be ErrHelp if -help or -h were set but not defined.
//
// Parse 从参数列表中解析标志定义，参数列表中不应该包含命令名称。必须在所有标志被定义后，
// 以及标志被程序访问前被调用。如果设置了 -help 或 -h 或者使用了未定义的标志，则返回值将
// 为 ErrHelp。
func (f *FlagSet) Parse(arguments []string) error {
	f.parsed = true
	f.args = arguments
	for {
		seen, err := f.parseOne()
		if seen {
			continue
		}
		if err == nil {
			break
		}
		switch f.errorHandling {
		case ContinueOnError:
			return err
		case ExitOnError:
			os.Exit(2)
		case PanicOnError:
			panic(err)
		}
	}
	return nil
}

// Parsed reports whether f.Parse has been called.
func (f *FlagSet) Parsed() bool {
	return f.parsed
}

// Parse parses the command-line flags from os.Args[1:]. Must be called
// after all flags are defined and before flags are accessed by the program.
//
// Parse 从 os.Args[1:] 解析命令行标志。必须在所有标志被定义后，以及标志被程序访问前被
// 调用。
func Parse() {
	// Ignore errors; CommandLine is set for ExitOnError.
	//
	// 忽略错误；CommandLine 被是在为 ExitOnError。
	CommandLine.Parse(os.Args[1:])
}

// Parsed reports whether the command-line flags have been parsed.
func Parsed() bool {
	return CommandLine.Parsed()
}

// CommandLine is the default set of command-line flags, parsed from os.Args.
// The top-level functions such as BoolVar, Arg, and so on are wrappers for the
// methods of CommandLine.
//
// CommandLine 是默认的命令行标志集，从 os.Args 解析。
// 顶层方法，如 BoolVar、Arg 等，是 CommandLine 方法的包装器。
var CommandLine = NewFlagSet(os.Args[0], ExitOnError)

func init() {
	// Override generic FlagSet default Usage with call to global Usage.
	// Note: This is not CommandLine.Usage = Usage,
	// because we want any eventual call to use any updated value of Usage,
	// not the value it has when this line is run.
	//
	// 覆盖通用 FlagSet 默认 Usage 函数，调用全局 Usage 函数。
	// IMP: 此处是很高级的用法
	// Note: 不使用 CommandLine.Usage = Usage 是因为我们希望所有调用最终使用的都是
	// Usage 的更新值，而不是运行此行时的值。
	CommandLine.Usage = commandLineUsage
}

func commandLineUsage() {
	Usage()
}

// NewFlagSet returns a new, empty flag set with the specified name and
// error handling property. If the name is not empty, it will be printed
// in the default usage message and in error messages.
//
// NewFlagSet 返回一个新的、空的、带有指定名称和错误处理级的标志集。如果名称不为空，
// 它将在默认的用法信息和错误信息中被输出出来。
func NewFlagSet(name string, errorHandling ErrorHandling) *FlagSet {
	f := &FlagSet{
		name:          name,
		errorHandling: errorHandling,
	}
	f.Usage = f.defaultUsage
	return f
}

// Init sets the name and error handling property for a flag set.
// By default, the zero FlagSet uses an empty name and the
// ContinueOnError error handling policy.
//
// Init 给一个标志集设置名称和错误处理级。默认情况下，FlagSet 的零值使用一个空的名称和
// ContinueOnError 错误处理级。
func (f *FlagSet) Init(name string, errorHandling ErrorHandling) {
	f.name = name
	f.errorHandling = errorHandling
}

# Annotate Go SDK

以前总是断断续续地看了写 Go 的源码，但时间一长就不容易记得清白了。

所以我希望将所看的所想的记录下来，与大家分享。大家有什么想说的，也十分欢迎提 [Issues](https://github.com/lizebang/annotate-go-sdk/issues/new) 及提 PR。

我所看的是 Go1.11 版本，这是 Go 的 [LICENSE](./GO-LICENSE)。

## Prepare

我使用的是 vscode，你们也可以使用它，同时推荐安装 [Go](https://marketplace.visualstudio.com/items?itemName=ms-vscode.Go)、[todo-highlight](https://marketplace.visualstudio.com/items?itemName=wayou.vscode-todo-highlight) 和 [todo-tree](https://marketplace.visualstudio.com/items?itemName=Gruntfuggly.todo-tree) 这三个插件。

## Workspace Settings

为了让 vscode 能正常跳转，不跳到 `$GOROOT`，请在 Workspace Settings 将 `go.goroot` 设置为本目录。

注意：vscode 可能会提示需要升级 go tools，此时请忽略。

## Extension Settings

我的 `todo-highlight` 和 `todo-tree` 设置如下：

```settings
	// todohighlight
	"todohighlight.keywords": [
		{
			"text": "TODO:",
			"color": "#000",
			"backgroundColor": "#ffbd2a",
			"overviewRulerColor": "rgba(255,189,42,0.8)"
		},
		{
			"text": "FIXME:",
			"color": "#000",
			"backgroundColor": "#f06292",
			"overviewRulerColor": "rgba(240,98,146,0.8)"
		},
		{
			"text": "NOTE:",
			"color": "#000",
			"backgroundColor": "#00F0F0",
			"overviewRulerColor": "rgba(240,98,146,0.8)"
		},
		{
			"text": "TS:",
			"color": "#000",
			"backgroundColor": "#aa00aa",
			"overviewRulerColor": "rgba(240,98,146,0.8)"
		},
		{
			"text": "IMP:",
			"color": "#000",
			"backgroundColor": "#a287f4",
			"overviewRulerColor": "rgba(240,98,146,0.8)"
		}
	],

	// todo-tree
	"todo-tree.defaultHighlight": {
		"foreground": "green",
		"background": "white",
		"type": "none"
	},
	"todo-tree.tags": ["TODO:", "FIXME:", "NOTE:", "TS:", "IMP:"],
	"todo-tree.customHighlight": {
		"TODO:": {},
		"FIXME:": {},
		"NOTE:": {},
		"TS:": {},
		"IMP:": {}
	},
```

`TS` 和 `IMP` 是我自己定义的，它们的含义是：

- `TS:` translate 翻译
- `IMP:` important 重要

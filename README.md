# Annotate Go SDK

以前总是断断续续地看了写 Go 的源码，但时间一长就不容易记得清白了。

所以我希望将所看的所想的记录下来，与大家分享。大家有什么想说的，也十分欢迎提 [Issues](https://github.com/lizebang/annotate-go-sdk/issues/new) 及提 PR。

我所看的是 Go1.11 版本，这是 Go 的 [LICENSE](./GO-LICENSE)。

## Prepare

我使用的是 vscode，你们也可以使用它，同时推荐安装 [Go](https://marketplace.visualstudio.com/items?itemName=ms-vscode.Go) 和 [todo-tree](https://marketplace.visualstudio.com/items?itemName=Gruntfuggly.todo-tree) 这两个插件。

## Settings

`todo-tree` 设置如下：

```settings
  // todo-tree
  "todo-tree.tags": ["TODO:", "FIXME:", "BUG:", "NOTE:", "TS:", "IMP:", "TSK:"],
  "todo-tree.customHighlight": {
    "BUG:": {
      "icon": "bug",
      "type": "tag",
      "opacity": 100,
      "foreground": "#000000",
      "background": "#e11d21",
      "iconColour": "#e11d21"
    },
    "FIXME:": {
      "icon": "tools",
      "type": "tag",
      "opacity": 100,
      "foreground": "#000000",
      "background": "#fbca04",
      "iconColour": "#fbca04"
    },
    "TODO:": {
      "icon": "check",
      "type": "tag",
      "opacity": 100,
      "foreground": "#000000",
      "background": "#0ffa16",
      "iconColour": "#0ffa16"
    },
    "NOTE:": {
      "icon": "note",
      "type": "tag",
      "opacity": 100,
      "foreground": "#000000",
      "background": "#0052cc",
      "iconColour": "#0052cc"
    },
    "TSK:": {
      "icon": "tasklist",
      "type": "tag",
      "opacity": 100,
      "foreground": "#000000",
      "background": "#d455d0",
      "iconColour": "#d455d0"
    },
    "IMP:": {
      "icon": "issue-opened",
      "type": "tag",
      "opacity": 100,
      "foreground": "#000000",
      "background": "#aa00aa",
      "iconColour": "#aa00aa"
    },
    "TS:": {
      "icon": "sync",
      "type": "tag",
      "opacity": 100,
      "foreground": "#000000",
      "background": "#d2b48c",
      "iconColour": "#d2b48c"
    }
  },
```

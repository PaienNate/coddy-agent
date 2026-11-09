# 键盘

控制台 TUI 与 Web 界面输入框的按键，按代码中的绑定方式：控制台表格来自 `external/cli/app.go` 和 `external/cli/tui/editor.go`，Web 界面表格来自 `external/ui/src/ui/chat/Composer.tsx` 及其打开的组件。控制台用 `/hotkeys` 打印自己的简短版本。

## 控制台

按键会被解析为 `external/cli/tui/keys.go` 的 `ctrl+x` / `shift+enter` / `alt+backspace` 记法。模态框打开时，只有模态框接收键盘输入；编辑器建议菜单打开时，它优先接收导航键。

| 按键 | 操作 |
|---|---|
| enter | 空闲时发送；轮次进行中时排入 `agent.queue_mode`（首次使用会询问默认值） |
| shift+enter / ctrl+j | 换行（对于不上报 shift+enter 的终端，紧挨 enter 前的反斜杠也会拆行） |
| escape | 若建议菜单打开则关闭它；否则停止正在运行的 `!!` 命令；否则停止等待浏览器的 `/login` 登录；否则中断正在运行的轮次（`HandleSessionCancel`） |
| ctrl+c | 清空编辑器；编辑器为空时，2 秒内按两次退出 |
| ctrl+d | 编辑器为空时退出 |
| ctrl+l | 模型选择器 |
| F1 | 内置文档（[内置文档](../features/built-in-docs.md#控制台帮助)）；在终端把 F1 留给自己的地方使用 `/docs` |
| ctrl+p / ctrl+shift+p | 在已配置的模型之间向前 / 向后循环 |
| shift+tab | 循环切换推理级别（带 `reasoning_levels` 的模型） |
| ctrl+o | 展开头部提示、上一次工具输出（当 `write` / `edit` / `apply_patch` 参数仍在流式传输时，为实时草稿）以及上一次 `!!` 块 |
| ctrl+t | 折叠或展开思考块 |
| up / down | 在草稿首行 / 末行时浏览提示词历史；否则移动光标 |
| ctrl+v / alt+v | 将剪贴板中的图片粘贴为 `[image #N]` 标记；在 X 显示环境下，它会先在标记窗口中打开（[粘贴和标记图片](../features/picture-markup.md)） |
| tab | 轮次进行中、有草稿且没有建议菜单打开时，以与 Enter 相反的模式排队；否则打开补全或插入制表符 |
| 1 / 2 | 仅当首次使用的排队问题显示在状态行时：将 Steer / After turn 保存为 Enter 模式并排队该消息；escape 会把草稿放回 |
| 草稿开头的 `/`，任意位置的 `@` | 输入时打开命令菜单和提及菜单（[提及](../features/mentions.md#在控制台中)） |
| tab / enter | 提及菜单打开时：采用高亮行；文件夹或 `@session:` 会让菜单保持打开并显示其内容 |
| escape | 提及菜单打开时：关闭它 |

在 `/tasks` 浮层内（[后台任务](../features/background-tasks.md#在控制台中)）：

| 按键 | 操作 |
|---|---|
| up / down | 在任务之间移动 |
| enter | 打开光标下的任务：其命令和输出的最后几行 |
| s | 停止光标下的任务；若有打开的任务且仍在运行，则停止打开的任务 |
| r | 重新读取任务，以及打开任务的输出 |
| escape | 离开打开的任务；在列表上时，关闭浮层 |

在 F1 和 `/docs` 打开的帮助内（[内置文档](../features/built-in-docs.md#控制台帮助)）：

| 按键 | 操作 |
|---|---|
| 字母、backspace / shift+backspace | 编辑搜索（在列表中） |
| up / down | 在条目之间移动；在页面上时，滚动一行 |
| enter | 打开条目并定位到其章节；在页面上时，滚动一行 |
| pgup / pgdn / space | 将页面滚动一屏 |
| home / end | 页面顶部 / 底部 |
| tab / shift+tab | 页面的下一节 / 上一节 |
| n / p | 文档的下一页 / 上一页 |
| / | 从页面返回搜索 |
| escape | 离开页面；在列表上时，关闭帮助 |
| F1 / ctrl+c | 关闭帮助 |

草稿内的编辑按键：

| 按键 | 操作 |
|---|---|
| left / ctrl+b, right / ctrl+f | 一个字符 |
| alt+left / ctrl+left / alt+b, alt+right / ctrl+right / alt+f | 一个词 |
| home / ctrl+a, end / ctrl+e | 行首 / 行尾 |
| backspace / shift+backspace, delete | 向后 / 向前一个字符 |
| ctrl+w / alt+backspace | 删除光标前的词 |
| alt+d / alt+delete | 删除光标后的词 |
| ctrl+u, ctrl+k | 删除到行首 / 行尾 |

菜单、选择器和提示：

| 按键 | 位置 | 操作 |
|---|---|---|
| up / down | 建议菜单、每个选择器 | 移动高亮 |
| enter / tab | 建议菜单 | 应用高亮行；在斜杠命令上按 enter 会一步应用并提交，因此 `/resume` + enter 会直接打开选择器 |
| escape | 建议菜单 | 关闭它 |
| 输入的字符、backspace / shift+backspace | 模型、模式、主题和恢复选择器 | 过滤列表 |
| enter | 选择器、权限提示、问题 | 选择高亮选项（权限提示会提供智能体发送的选项） |
| escape / ctrl+c | 选择器、权限提示、问题 | 取消；取消的权限提示会拒绝该调用 |
| space | 带 `multiple: true` 的问题 | 切换高亮选项 |
| escape | 问题的自定义答案编辑器 | 返回选项列表 |

在 X 显示环境下，粘贴的图片会打开的标记窗口（[粘贴和标记图片](../features/picture-markup.md#标注窗口)）：

| 按键 | 操作 |
|---|---|
| p / o / r / a / t / c | 选择铅笔、椭圆、矩形、箭头、文本或裁剪工具 |
| enter | 完成：附加标记后的图片；正在输入标签时，完成该标签 |
| escape | 取消：不附加任何内容；正在输入标签时，丢弃该标签 |
| backspace | 正在输入标签时，删除其最后一个字符 |
| ctrl+z | 丢弃正在输入的标签，否则撤销上一步 |

来自代码的平台说明：单独的 Escape 在本地会在 10 ms 后解析，在设置了 `SSH_CONNECTION` 或 `SSH_TTY` 时则为 100 ms（`external/cli/tui/terminal.go`）；紧随其后跟着 `/` 的 Escape 对两者都不等待，并且绝不会被读作 alt+`/`，控制台会先接收 Escape，再接收斜杠命令，因此一次动作就能关闭选择器并输入下一条命令；反斜杠加 enter 的换行是为不上报 shift+enter 的终端而存在；在非空编辑器上，`ctrl+c` 会清空它，因此退出需要空编辑器并再按一次；终端在按住 Shift 时报告为编码按键的字符（xterm 的 modifyOtherKeys，控制台会启用它，以及 Kitty 的按键报告）会作为文本输入，而不是被读作快捷键，因此输入不会在大写单词的第一个字母处停止。

## Web 界面

输入框是一个普通的 `textarea`；此处未列出的按键保留其浏览器含义。

| 按键 | 位置 | 操作 |
|---|---|---|
| Enter | 输入框，任何带键盘的设备，包括窄窗口 | 空闲时发送；轮次运行时，将文本和附件排入已配置的 `agent.queue_mode`（首次使用会询问偏好哪种模式） |
| Tab | 轮次运行时的输入框，有草稿且没有选择器打开 | 以与 Enter 偏好相反的模式排入文本和附件 |
| Shift+Enter | 输入框 | 换行（浏览器默认行为，不拦截） |
| Ctrl+Enter / Alt+Enter | 输入框 | 在光标处换行，替换选区（浏览器不会插入任何内容，因此由输入框执行） |
| Cmd+Enter | 输入框 | 发送，同 Enter |
| Enter | 输入框，仅触摸设备（没有悬停指针，指针为粗略类型：手机） | 换行；发送由按钮完成，因为手机键盘没有 Shift+Enter |
| Enter, Tab, arrows, Escape, Ctrl+Z | 输入框，输入法正在组合时（keydown 携带 `isComposing`，或在 Safari 中于 `compositionend` 后 100 ms 内的 keyCode 229） | 属于输入法：不会发送任何内容，斜杠、`@`、命令选项或行范围选择器都不会采用某一行、移动其高亮或关闭，Ctrl+Z 不会恢复改进提示词之前的草稿；任何其他 keyCode 229（Android 键盘为普通按键发送的）会像普通按键一样操作选择器，不过带它的 Enter 仍然不会发送 |
| ArrowUp / ArrowDown | 斜杠、`@` 或命令选项菜单打开 | 移动高亮行，在两端循环 |
| Tab | 斜杠菜单打开 | 应用高亮命令 |
| Tab | `@` 提及菜单打开 | 应用高亮行，轮次运行时也如此 |
| Tab | 命令选项菜单打开（`/compact --model`、`/goal -r`） | 将高亮选项、模型或推理级别放入草稿 |
| Enter | 斜杠、`@` 或命令选项菜单打开 | 应用但不发送；文件夹或 scheme 行会让 `@` 菜单保持打开，而 `--model` 会打开模型列表 |
| Escape | 斜杠、`@`、命令选项或行范围选择器打开 | 关闭选择器；针对该提及，`@path:N-M` 选择器会保持关闭，直到草稿发生变化 |
| Escape | 上下文明细浮层打开 | 关闭它 |
| Escape | 输入框在聊天上方展开 | 将其折回文本高度；下一次 Escape 会在有编辑打开时退出编辑（[Web 界面](../surfaces/web-ui.md#输入框中的长提示词)） |
| Ctrl+Z / Cmd+Z | 输入框，紧接改进提示词之后 | 恢复改进前的草稿，仅一次 |
| Enter / Space | 上下文环按钮获得焦点 | 打开或关闭明细 |
| Enter / Escape | 模型菜单过滤器（后端超过五个时显示） | 选择第一个匹配项 / 关闭菜单 |
| Escape | 输入框的菜单（模式、模型、推理、权限）、环境、文件夹或分支菜单 | 关闭它，无论焦点在哪里 |
| Escape | 文件夹浏览器（打开文件夹…） | 新建文件夹行打开时折叠它，否则像取消一样关闭浏览器 |
| Escape | 设置中的技能安装搜索，已输入查询 | 清空它并移除市场结果；下一次 Escape 属于抽屉 |
| / | 设置中，文本字段或下拉选择框之外 | 将光标放入设置搜索并显示其结果；在窄屏幕上且某个章节打开时，先返回磁贴；空字段中的 `/` 徽章会在字段未聚焦时显示该按键，且在所有宽度下都如此，仅触摸设备没有徽章（[Web 界面](../surfaces/web-ui.md#设置搜索)） |
| ArrowUp / ArrowDown, Enter | 显示结果的设置搜索 | 移动高亮结果，打开它 |
| Escape | 设置搜索 | 显示结果时，清空查询；在字段中且结果隐藏时，清空查询；下一次 Escape 属于抽屉 |
| Escape | 侧栏打开的屏幕：历史记录、调度器、聊天上方的集群、文档、设置 | 像它的 × 一样关闭它，返回其下方内容；一次一步：调度器中打开的任务或其运行记录、设置中打开的行，以及手机或平板上的设置章节会先返回，而打开的菜单、选择器、提示或组合框列表，或含文本的搜索，会先于屏幕接收该按键（[Web 界面](../surfaces/web-ui.md#布局)） |
| Enter | 问题提示 | 当每个问题都有答案后发送答案；在输入框中输入时，它仍属于输入框 |
| Escape | 问题提示 | 跳过问题，除非聊天上方打开了侧栏的某个屏幕，或输入框的某个选择器接收了该按键 |
| Escape / Tab | 确认对话框 | 取消 / 将焦点保持在对话框内 |
| Ctrl+Shift+F / Cmd+Shift+F | 打开的聊天，包括输入框 | 打开文件窗口，或关闭它（[Web 界面](../surfaces/web-ui.md#工作区文件)） |
| Escape | 文件窗口 | 过滤器有文本时清空它，否则关闭窗口；其下方的停靠栏会留给下一次 Escape |
| Escape | 编辑窗口 | ⋮ 菜单打开时将其收起，树过滤器有文本时清空它，否则关闭窗口；丢弃问题打开时，改为回答该问题并离开窗口（[Web 界面](../surfaces/web-ui.md#编辑窗口)） |
| P / O / R / A / T / C | 标记编辑器，标签之外 | 选择铅笔、椭圆、矩形、箭头、文本或裁剪工具（[粘贴和标记图片](../features/picture-markup.md#在-web-界面中)） |
| Enter / Escape | 标记编辑器 | 完成 / 取消；在标签中，完成它 / 丢弃它 |
| Ctrl+Z / Cmd+Z | 标记编辑器 | 丢弃正在输入的标签，否则撤销上一步 |
| F1 | 任意位置 | 打开文档阅读器，或关闭它（[内置文档](../features/built-in-docs.md#web-界面阅读器)） |
| / | 文档阅读器，字段之外 | 将光标放入其搜索框 |
| ArrowUp / ArrowDown, Enter, Escape | 文档搜索框 | 移动选中的命中项，打开并定位到其章节，清空搜索；在空框中，Escape 关闭阅读器 |

停止轮次没有按键：它是输入框栏中的停止按钮。输入 `@` 时会打开 `@` 菜单（[提及](../features/mentions.md#在-web-界面中)），而在文件后输入 `:` 会将其变成行范围选择器；两者都不是绑定。

<!-- docsgen:source sha256=ce25c41e565347c0 -->

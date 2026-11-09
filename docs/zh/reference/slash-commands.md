# 斜杠命令

每个界面上的内置命令，以及技能如何变成命令。斜杠命令是六类事物之一：设置命令（`/model`、`/reasoning`、`/think`、`/nothink`、`/toolset`、`/agent`、`/plan`、`/ask`、`/permissions`），会话管理器在每个界面上从提示词开头取走，并在任何轮次开始前应用；控制台或 Web 输入框的客户端命令（`/tasks`、`/docs`、`/mcp` 等），会打开其界面，且永远不会到达智能体；确定性内置命令（`/compact`、`/export`、`/plugin`、`/worktree`），智能体在提示词变成消息之前识别并运行，不经过模型的一个轮次；目标命令（`/goal`），由会话管理器在每个界面上应答，显示、暂停和清除不运行轮次，而设置目标和 `/goal resume` 会启动一个轮次，并且需要智能体模式；Telegram 机器人命令，由适配器处理；或技能，智能体会将其正文写入调用它的消息中，作为输入文本之后的附件。无论会话模式是什么，设置命令、客户端命令、内置命令和机器人命令都会运行，因为它们是操作员输入，而不是工具调用。

## 命令

| 命令 | 界面 | 作用 | 详情 |
|---|---|---|---|
| `/model <id> [--once\|--count=N]` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses`、Telegram、Pachca | 切换会话的模型，或下一轮次或 N 个轮次的模型。在控制台中单独输入，或从 Web 界面的 `/` 菜单中选择时，会打开模型选择器；在 Telegram 中单独输入时，会打开一个基于已配置的 `models` 的内联键盘。 | [会话设置](../features/session-settings.md#命令) |
| `/reasoning <level\|off\|default> [--once\|--count=N]`，别名 `/effort` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses`、Telegram、Pachca | 设置推理级别；在提供商支持时，`off` 关闭思考，`default` 返回模型自身的级别。在控制台中单独输入，或从 Web 界面的菜单选择时，会打开推理选择器。 | [会话设置](../features/session-settings.md#关闭思考) |
| `/think [level] [--once\|--count=N]`，`/nothink [--once\|--count=N]`（别名 `/no_think`） | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses`、Telegram、Pachca | 打开思考（使用模型的默认级别，或指定的级别）或关闭思考。 | [会话设置](../features/session-settings.md#关闭思考) |
| `/agent`，`/plan`，`/ask` `[--once\|--count=N]` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses`、Telegram、Pachca | 切换运行模式。 | [运行模式](../features/modes.md#在每个界面上切换) |
| `/permissions <ask\|accept_edits\|bypass> [--once\|--count=N]` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses` | 设置本会话中工具在何时请求批准；持久化在会话的 `session.json` 中，因此重启后仍然保留；`--once` 和 `--count=N` 不会保留。在控制台中单独输入，或从 Web 界面的菜单选择时，会打开权限选择器。不是 Telegram 命令：机器人会自行批准其聊天智能体。 | [会话设置](../features/session-settings.md#从权限对话框切换) |
| `/toolset <name\|default> [--once\|--count=N]` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses`、Telegram、Pachca | 选择本会话被提供工具时所依据的 `toolsets[]` 预设；`default` 提供该模式的所有工具。保存在会话的 `session.json` 中。仅当配置中存在预设时才会在菜单中列出；在控制台中单独输入，或从 Web 界面的菜单选择时，会打开工具集选择器。 | [会话设置](../features/session-settings.md#工具集) |
| `/resume [id or title]` | 控制台；Telegram（机器人的管理员） | 控制台：在当前文件夹的会话上打开选择器；所选会话替换当前会话。Telegram：在服务器保留的每个会话上打开内联键盘，或者，如果后面带文字，则打开其 id 或标题所指的会话。 | [会话](../features/sessions.md#恢复)，[Telegram 网关](../surfaces/gateway.md#命令) |
| `/new` | 控制台 | 在同一文件夹中启动新会话。 | [控制台](../surfaces/console.md#命令与按键) |
| `/login [provider]` | 控制台 | 为这台机器设置模型提供商，并将该提供商及其模型添加到 `config.yaml`。单独输入时，会打开提供商选择器，这是 Web 界面首次运行界面的控制台对应版本，全新的控制台也会自行打开它：NeuralDeep、OpenCode、OpenAI、Anthropic、Codex、Devin、OpenAI-compatible 以及你自己的 ACP 智能体，NeuralDeep 排在第一位。指定名称可跳过选择器：`neuraldeep`、`codex`、`devin` 或 `opencode`，或 `config.yaml` 列出的提供商。控制台会立即在新配置上运行。不能通过 `--remote` 使用。 | [控制台](../surfaces/console.md#登录提供商-login) |
| `/theme` | 控制台 | 在深色和浅色调色板之间选择。 | [控制台](../surfaces/console.md#标志) |
| `/hotkeys` | 控制台 | 打印键位列表。 | [键盘](keyboard.md) |
| `/queue [list\|drop <n>\|mode <n> <steer\|after_turn>\|clear]` | 控制台 | 列出等待中的消息、更改一个模式、将一条消息取回输入框，或清空队列。也可对远程服务器使用。 | [消息队列](../features/message-queue.md#在控制台中) |
| `/usage` | 控制台 | 强制重新读取当前提供商账户用量并打印明细；在 `--remote` 下，读取的是服务器自己的密钥。 | [控制台](../surfaces/console.md#命令与按键) |
| `/tasks` | 控制台 | 在编辑器位置打开会话的后台任务：正在运行什么、已完成什么、按 **enter** 查看某个任务的输出、按 **s** 停止某个任务。在 `--remote` 下，它会列出并停止服务器的进程。 | [后台任务](../features/background-tasks.md#在控制台中) |
| `/mcp` | 控制台、Web 界面、Telegram | 控制台：打开服务器列表，包含连接状态、工具数量、服务器和工具开关，以及在 `mcp.project_trust: ask` 下的项目信任控制。Web 界面：打开设置 → MCP 服务器，无论命令后面是什么。Telegram：列出服务器，并且只为受信任条目提供启用/禁用按钮；在群组中需要提及（`/mcp@botname`）。 | [MCP](../features/mcp.md#管理-api-与-ui) |
| `/docs [words or page]` | 控制台、Web 界面 | 打开 Coddy 的内置文档，与 F1 打开的是同一界面：目录、按词语搜索，或打开一个页面（`/docs features/mentions#completion`）并定位到其章节。页面可通过其地址或确切标题指定；其他任何词语都会作为搜索。在 Web 界面中，输入框在浏览器中运行它，不会有任何内容到达智能体。`/help` 是控制台中的同一命令。 | [内置文档](../features/built-in-docs.md#控制台帮助) |
| `/quit`，`/exit` | 控制台 | 退出。 | [控制台](../surfaces/console.md#命令与按键) |
| `/compact [-m\|--model <id>] [-r\|--reasoning <level>] [instructions]` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses` | 总结较早的历史，并逐字保留最近的轮次；命令后的词语会引导摘要器，`--model`（`-m`）指定撰写本次摘要的模型（一个 `models[].model`、不带提供商名称的模型名，或能精确匹配一个模型的部分名称），`--reasoning`（`-r`）指定撰写时使用的级别。仅在 `compaction.enable` 为 true 时列出；手动压缩始终强制执行。 | [上下文压缩](../features/compaction.md#compact-命令) |
| `/goal [-m\|--model <id>] [-r\|--reasoning <level>] [--then-file <path>] <objective>`，`/goal`，`/goal pause`，`/goal resume [-m\|--model <id>] [-r\|--reasoning <level>]`，`/goal clear` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses`、Telegram、Pachca | 设置会话目标并立即开始处理；第二个模型检查每个轮次，并继续未完成的工作，直到目标达成、需要你介入或预算耗尽。单独输入 `/goal` 会显示它（Web 界面和控制台会打开目标菜单），`pause` 和 `resume` 会停止并重新启动继续执行，`resume` 还可接受阻止目标的内容（[风险门禁](../features/session-supervisor.md#风险门禁)），`clear` 会移除它。`--model` 和 `--reasoning` 像 `/compact --model` 一样选择检查目标的模型和级别。`--then-file` 会串联目标：一旦目标达成，该文件的文本会成为新会话的目标（仅 `coddy serve`）。在消息平台中，只有机器人的管理员可以更改它。 | [会话目标与监督器](../features/session-supervisor.md) |
| `/export [md\|html\|json\|jsonl] [path] [--no-tools] [--no-thinking]` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses` | 将对话记录写入会话工作区；目录会接收 `coddy-export-<timestamp>.<ext>`，文件名会按扩展名设置格式。在 `--remote` 下，文件落在服务器上。 | [会话导出](../features/session-export.md#用法) |
| `/plugin marketplace add\|list\|update\|remove`，`/plugin install <plugin>@<marketplace>`，`/plugin update [<name>]`，`/plugin install\|remove\|enable\|disable` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses` | 管理技能插件和市场，是 `coddy plugin` 的聊天对应版本：添加市场不会安装任何内容，`install <plugin>@<marketplace>` 会安装其中的一个插件，`marketplace update` 读取列表并报告更新的版本，`update` 会为已安装的内容安装它们。批准项目市场（`marketplace trust`）交由终端和设置处理。 | [技能](../features/skills.md#plugin-命令cli-与聊天中的-plugin) |
| `/worktree <branch> [--from-head]`，`/worktree <path>`，`/worktree exit [--keep \| --remove [--force]]`，`/worktree` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses` | 将聊天移动到某个分支的 git worktree 中（没有则打开一个），或按文件夹移动到一个现有 worktree，或移回主检出，这是 `worktree_open` 和 `worktree_exit` 的操作员对应版本。`exit` 会移除不包含自身内容的 worktree，并保留其中有工作的 worktree；`--keep` 仍会保留它，`--remove` 会移除它或拒绝，`--force` 会丢弃未提交的更改，并保留为快照。单独使用时，它会说明聊天在哪里工作，以及仓库的每个 worktree 处于什么状态。Web 界面面板对话框的答案就是这些命令。 | [Git worktree](../features/worktrees.md) |
| `/start`，`/help` | Telegram、Pachca | 机器人的问候语和命令列表。 | [Telegram 网关](../surfaces/gateway.md#命令) |
| `/context` | Telegram、Pachca | 按类别显示聊天会话的上下文窗口使用情况。 | [Telegram 网关](../surfaces/gateway.md#命令) |
| `/clear` | Telegram、Pachca | 为聊天启动一个新会话（在群组中，机器人只从其管理员处接受该命令，和设置命令一样）；旧会话仍保留在磁盘上。在 Telegram 中，`/resume` 会将它带回；Pachca 没有 `/resume`，可通过 Web 界面或控制台继续它。 | [Telegram 网关](../surfaces/gateway.md#会话生命周期)，[Pachca 网关](../surfaces/pachca.md#命令) |
| `/<skill>` | 控制台、Web 界面、ACP 编辑器、`POST /v1/responses` | 运行技能：完整的 `SKILL.md` 正文会写入调用它的消息中，作为输入文本之后的附件，后续轮次会重放它。 | [技能](../features/skills.md#技能如何应用) |

从代码中可以得出四个边界：

- Telegram 适配器会应答自己的命令，将 `/goal` 以及除 `/permissions` 之外的每个设置命令传递给会话（`isSettingsCommand` 在 `external/gateway/telegram/bot.go` 中），并丢弃任何其他以 `/` 开头的消息，因此 `/compact`、`/export`、`/worktree` 和技能在那里不可达；
- Pachca 适配器以其较小的集合执行同样的操作（`knownCommand` 在 `external/gateway/pachca/dispatch.go` 中，包含 `/goal`）：发送到那里的 `/compact`、`/export`、`/worktree`、`/resume` 和 `/mcp` 会被丢弃；
- 子智能体从不运行内置命令：以 `/export` 开头的子提示词对子智能体来说是一项普通任务（`internal/agent/react.go`）；
- 大多数控制台客户端命令只存在于控制台中；`/mcp` 还会打开 Web 设置和 Telegram 菜单。设置命令在所有地方都相同，因为会话管理器中的一个解析器和一个设置器服务于每个界面。

## 每个界面从哪里获取其列表

| 界面 | 列表来源 | 备注 |
|---|---|---|
| 控制台 | 一个固定的客户端列表（`resume`、`new`、`theme`、`hotkeys`、`queue`、`usage`、`tasks`、`mcp`、`goal`、`docs`、`quit`）与服务器通过 ACP `available_commands_update` 通知公布的各行合并，其中包含设置命令（`slashCatalog` 在 `external/cli/app.go` 中）。 | 输入 `/` 会打开建议菜单；在建议上按 Enter 会应用它并一次性提交。 |
| Web 界面 | 输入框中的两组内容：来自 `GET /coddy/commands` 的内置命令，加上本地 `/docs` 和 `/mcp`，以及来自 `GET /coddy/slash-commands` 的技能，通过 `prefix` 分页并筛选，并通过 `X-Coddy-Session-ID` 限定到会话工作区。服务器内置命令行带有其 `kind`（`setting` 或 `action`）、参数 `hint`、其 `aliases`，对于设置命令，还会带有它更改的 `setting` 及其 `choices` 或固定的 `value`。 | 当服务器宣布配置重新加载时，两个服务器列表都会重新读取（`event: config_reloaded` 在 `GET /coddy/events` 上）。选择 `/model`、`/reasoning` 或 `/permissions` 会打开该选择器；`/agent`、`/plan` 或 `/ask` 会切换模式；`/mcp` 会打开设置 → MCP 服务器。其他行会插入普通的 `/name` 标记。一旦输入了 `/compact` 或 `/goal`，输入框会补全它们的选项和值：`--model`（`-m`）来自已配置的模型，`--reasoning`（`-r`）来自草稿的 `--model` 所指模型的级别（[输入框命令选项](../surfaces/web-ui.md#输入框命令选项)）。 |
| ACP 编辑器 | `available_commands_update` 在 `session/new` 和 `session/load` 之后：先是设置命令，然后是其他内置命令，再后是按名称排序的技能；行带有 `name`（不带斜杠）、`description`，对于设置命令，还带有 `/compact`、`/goal` 和 `/worktree`，`input.hint` 及其参数和标志。与某个内置命令或其别名同名的技能会被排除。 | 同一个函数 `session.BuiltinCommandRows`（位于 `internal/session/commands.go` 中）为 HTTP 端点和通知提供数据，因此两者永远不会不一致：`compact` 仅在压缩启用时存在，`goal`、`export`、`plugin` 和 `worktree` 始终存在。 |
| Telegram | `start`、`help`、`model`、`mcp`、`agent`、`plan`、`ask`、`context`、`goal`、`resume` 和 `clear`，在启动时通过 `setMyCommands` 注册。 | 它们会出现在客户端的命令菜单中；`/reasoning`、`/think` 和 `/nothink` 在输入时可用，但不在菜单中。 |
| Pachca | `start`、`help`、`clear`、`model`、`context`、`goal` 以及设置命令，但不包括 `/permissions`，在输入时识别（`knownCommand` 在 `external/gateway/pachca/dispatch.go` 中）。 | 这类机器人没有 Pachca 命令菜单；在群聊中，命令需要提及机器人，和那里的任何消息一样。参见 [Pachca 网关](../surfaces/pachca.md#命令)。 |

## 技能如何变成命令

加载器接受的每个技能文件都是一个命令：

- 标识符由 `skills.CanonicalCommandName` 从文件位置推导：文件夹内的 `SKILL.md` 取文件夹名（`skills/code-review/SKILL.md` 变成 `/code-review`），而位于 `skills.dirs` 条目根目录的 Markdown 文件取其文件名主干（`skills/deploy.md` 变成 `/deploy`）；
- frontmatter 的 `name` 字段设置技能的显示名称，而命令标识符仍保持由路径推导出的那个；
- 当两个文件映射到同一个标识符时，按 `skills.dirs` 顺序排在前面的文件胜出（`skills.ListSkills`），并且在配置中被禁用的技能根本不会加载；
- 命令旁显示的描述是 frontmatter 的 `description`，如果没有，则是正文的第一行纯文本，最多 160 个字符。

目录也会到达模型：系统提示词带有一个 `## Slash commands` 块，列出每个 `/name` 及其描述（`skills.BuildSlashCatalogMarkdown`），并且在 `skills.auto_discovery` 开启时，模型可以通过 `load_skill` 工具自行拉取完整正文（[工具](tools.md)）。

其中十个命令在全新安装中无需下载任何内容就已存在，即 `/configure-coddy`、`/crossreview`、`/rpa-init`、`/rpa-feat`、`/rpa-bugfix`、`/rpa-gen-rules`、`/chinese-code-review`、`/chinese-commit-conventions`、`/chinese-documentation` 和 `/chinese-git-workflow`，这是二进制写入 `${CODDY_HOME}/skills` 的[标准交付](../features/skills.md#标准交付)。

## 提示词中的命令如何解析

设置命令首先由 `session.ParseSettingsCommands` 取走，发生在轮次锁之前，也发生在文本变成消息之前：

- 只有输入文本的开头才算数；句子中间的设置命令是普通文本，提及的附件或技能正文永远不会被读取；
- 每个命令的值及其 `--once`、`--count=N` 或 `--count N` 标志，是命令所在行中其名称之后的词语；标志可以位于值之前或之后；
- 命令可以串联，位于同一行或连续行（`/model x --once /nothink --once review this`），不属于任何命令的第一个词会开始提示词，该提示词按原样保留，并且本身可以是 `/compact`、`/export` 或技能；
- 仅由命令组成的提示词不会运行轮次，也不会在模型历史或对话记录中留下任何内容：Web 界面和控制台会在各自的选择器和页脚上显示更改，而编辑器、Telegram 机器人和 HTTP 客户端会收到每次更改的通知作为应答；
- 名称匹配不区分大小写，包括别名，并且优先于同名技能。

五个内置命令会在整个提示词上识别（`parseCompactCommand`、`parseGoalCommand`、`parsePluginCommand`、`parseExportCommand` 和 `parseWorktreeCommand` 在 `internal/agent` 中）：

- 去除首尾空白后，文本必须恰好是 `/compact`、`/goal`、`/plugin`、`/export` 或 `/worktree`，或以其中一个开头并后跟空白；句子中间的内置命令不是命令；
- `/goal <text>` 会替换目标和继续次数；单独输入 `/goal` 会显示其状态，`/goal clear` 会移除它；选项 `--model` 和 `--reasoning` 放在最前面，和在 `/compact` 之后一样；
- 在 `/compact` 之后，选项放在最前面（`--model <id>`、`--model=<id>` 或 `-m <id>`，以及 `--reasoning <level>`、`--reasoning=<level>` 或 `-r <level>`；未知选项或没有值的选项会以用法行应答并且不压缩任何内容），从第一个不是选项的词开始，所有内容都是摘要器的指令，除其周围空白外逐字保留；只有 `--` 这类词、`-m` 和 `-r` 是选项，因此以列表项开头的指令仍然是指令；
- `/plugin` 之后的词语会被拆分为子命令及其参数；
- 在 `/export` 之后，`--no-tools` 和 `--no-thinking` 可以出现在任何位置，第一个剩余的词在指定格式时就是格式，其余用单个空格连接起来就是目标路径；
- 在 `/worktree` 之后，选项可以出现在任何位置，另一个词是 `exit`、一个路径（以 `/`、`./`、`../` 或 `~` 开头）或分支；`--from-head` 与分支搭配，`--remove` 与 `exit` 搭配，`--force` 与 `--remove` 搭配，其他任何内容都会以原因和用法行应答，并且不移动任何内容；
- 命令文本会作为用户行持久化，因此对话记录会显示它，结果则存储为助手消息。

技能由 `skills.ParseInvokedCommandNames` 在用户输入的文本上逐行查找（绝不会在提及带来的附件上查找）：

- 围栏代码块内的行以及以 `>` 开头的行会被跳过；
- 在剩余的行上，正则表达式 `invokedMidLineSlashRE`、`(?:^|[\t ])\/([a-zA-Z0-9][a-zA-Z0-9_-]*)` 会取出每个位于行首或紧跟空格或制表符之后的 `/name`，因此句子中间的 `/review` 会被计入，而 `x/foo`、`path/to/file` 和 `https://` 不会；
- 旧版 SPA 草稿的选择器形式 `[/name](coddy-skill:name)` 也会在两个名称一致时被接受；
- 每个名称只保留一次，按出现顺序；没有匹配技能的名称会作为纯文本保留，因此提示词中一个孤立的 `/tmp` 是无害的；
- 消息发送时，每个匹配技能的正文会写入其中一次，作为文本之后的附件（`<coddy_attachment path="skill:name" kind="skill">`，`invokedSkillBlocks` 在 `internal/agent/react.go` 中）。后续轮次会带着发送时的正文重放该消息，因此提供商的缓存前缀得以保持，模型也会保留给它的指令；对话记录仍会显示输入时的消息，Web 界面和重新打开的控制台会从气泡中丢掉正文。轮次中途排队的后续消息会以相同方式调用技能（[提及](../features/mentions.md#提及与提示词缓存)）。

控制台的客户端命令会在上述任何处理之前分派：草稿中第一个以空白分隔的字段，去掉开头的 `/` 后，会与固定列表匹配，未知名称会作为普通提示词回退给智能体（`dispatchSlash` 在 `external/cli/slash.go` 中）。

<!-- docsgen:source sha256=d349ac2060b1110d -->

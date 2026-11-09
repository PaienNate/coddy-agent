# 会话

会话就是一次对话：对话记录、它运行所在的工作目录、它使用的模式、模型和权限模式、它的待办清单、上传到其中的文件，以及它产生的工具活动。每个界面 - 控制台、Web 界面、ACP 编辑器、Telegram 网关、调度器运行、子智能体 - 都处理同一种会话，并以相同方式存储，因此在一个地方开始的对话可以在另一个地方继续。本页涵盖磁盘上的会话包、恢复、编辑消息时的历史回退、待办清单、子会话以及 `coddy sessions` 命令。将对话记录写入文件请参阅[会话导出](session-export.md)。

## 磁盘上的会话包

每个会话都是会话根目录下的一个目录，即 `$CODDY_HOME/sessions/<id>/`（默认为 `~/.coddy/sessions/`）。根目录会随 `config.yaml` 中的 `sessions.dir` 移动，或随 `coddy serve`、`coddy acp`、控制台以及 `coddy sessions` 动词上的 `--sessions-dir` 移动（[config.yaml 参考](../reference/config.md#sessions)）；进程在启动时创建它，无法创建时则启动失败。

ID 为 `sess_` 后跟 24 个十六进制字符，无论会话由谁启动：一次控制台运行、一个浏览器标签页、信使网关上的聊天、另一个会话派生的子智能体运行、调度器任务自身的会话及其下的运行。会话 ID 不会报告人当时坐在哪里，因此下游任何东西都无法据此分支。ID 同时用作文件夹名，所以只接受字母、数字、`_` 和 `-`。

由另一个会话派生的会话存储在其内部，位于 `<parent>/subagents/<child>/`，而该子会话的子会话再深一层，因此会话根目录列出的是人启动的对话，每个会话包携带它委派的工作。调度器任务的会话也以同样方式工作：任务的运行是它的子会话。会话包是什么 - 聊天、委派运行、任务会话、调度运行 - 记录在其 `session.json` 中，而不是名称中。

| 路径 | 内容 |
|---|---|
| `session.json` | id、`cwd`、`mode`、会话的模型和推理覆盖、会话的权限模式、标题（派生或由用户固定）、其归档到的 `tags`、是否为 `pinned`（带有 `pinnedAt` 和 `pinnedRank`）以及是否为 `archived`（带有 `archivedAt`）、`updatedAt`、由 `SessionStart` 钩子移交的上下文、调度器或子智能体会话包的运行元数据，以及未读圆点背后的活动计数器 |
| `messages.json` | 模型看到的对话记录：`user`、`assistant` 和 `tool` 行，包括压缩摘要；钩子以其路径作为 `transcript_path` 接收 |
| `assets/` | 通过输入框上传的文件，以只读保存，PNG 预览位于 `thumbnails/` 下 |
| `todos/active.md` | 当前待办清单；`todos/archive/` 保留被替换和归档的清单 |
| `plans/<slug>.plan.md` | 在计划模式中编写的设计计划 |
| `tool_calls/<id>/` | 每次工具调用的 `args.json`、`result.md` 和 `meta.json`，以便在流式传输显示预览后获取完整结果。ID 属于模型提供商，因此仅当它是普通单一名称时才会用作文件夹名（字母、数字、`_`、`.`、`-`，最多 128 个字符）；其他任何情况 - 分隔符、路径穿越、长于文件名的 ID - 改为存储在其摘要下，而无论哪种情况，`meta.json` 都携带该调用的 `toolCallId`。会自行编写 ID 的模型（如 `functions.write:0`）可能给新调用一个该会话较早调用已经携带的 ID：新调用随后会得到提供商的 ID，并附加第一个可用后缀（`_2`、`_3`），每个界面和对话记录都以该 ID 认识它 |
| `stats.json` | 已完成模型调用的 token 总数 |
| `background/<task_id>/` | 每个后台任务和子智能体运行的记录与输出日志（[长期记忆](memory.md)） |
| `subagents/<child id>/` | 此会话派生的每个子会话的会话包：一个 `spawn_agent` 子会话，每个子会话都是具有相同布局的独立会话 |
| `ui_log.json`, `permission_grants.json`, `pending_permission.json` | 对话记录中显示的通知行、使用 "始终允许" 批准的命令和写入目标、通过 HTTP 等待回答的权限提示 |
| `pending_plan_context.json` | 移交给计划运行所启动轮次的设计计划文本，保留到该轮次结束。因权限提示而暂停的轮次会在回答到达后继续，有时在另一个进程中继续，并再次渲染相同的系统提示词（[计划模式](modes.md)） |

压缩和结果逐出从不重写会话包：二者都是在请求发往模型时构建的投影，而 `messages.json` 保留每一原始行（[上下文压缩](compaction.md)）。

## 会话记住什么

模式（`agent`、`plan` 或 `ask`）、模型和推理级别是会话范围的覆盖项，持久化在 `session.json` 中。权限模式也属于会话：新会话以任何界面上最后选择的模式启动（保存在 `${CODDY_HOME}/permission-mode.json` 中，`ask` 直到做出任何选择），保持该模式直到被切换，并把它持久化为 `sessionPermissionMode`，因此重启后每个会话都会以自己的模式回来。由旧版本保存、文件中没有自身模式的会话在再次打开时采用当前默认值（子智能体的子会话在其自己的 `session.json` 中把其收窄后的模式记录在 `permissionMode` 下，作为其运行时所处模式的记录，永远不会被读回）。四者都通过同一个设置器更改 - ACP 的 `session/set_mode` 和 `session/set_config_option` 方法、输入框的选择器（`PATCH /coddy/sessions/{id}` 搭配 `selectedModelId`、`selectedReasoning`、`mode` 或 `permissionMode`）、每个界面上的设置命令以及控制台的 `ctrl+l` - 而仅针对接下来几轮的更改永远不会写入磁盘（[会话设置](session-settings.md)）。为某个会话切换的权限模式也会成为新会话启动时使用的模式。工作目录也会被记录：`coddy -c`、控制台选择器和 `coddy sessions list --cwd` 都会据此过滤。Web 界面在第一条消息之前选择文件夹、git 分支和可选 worktree，并在对话记录有消息后锁定它们（[Web 界面](../surfaces/web-ui.md#每会话工作区文件夹--分支--worktree)）。标题在固定之前从第一条提示词派生，可在聊天标题栏中内联固定，或使用 `PATCH /coddy/sessions/{id}` 和 `title` 固定；同一路由承载下文描述的 `tags` 和 `archived` 标志。

## 标签与归档

已经积累了数月的历史应当被收窄，而不是靠滚动浏览，而会话包上的两个字段承载这一点。

**标签**是会话归档所用的简短扁平词汇表：小写，内部空白转为连字符，去除重复项，最多八个，每个最多三十二个字符。没有人必须发明它们。为新聊天命名的调用 `POST /coddy/describe` 会在同一答案中向模型索取短语*和*一到三个主题标签，而固定标题的 `PATCH` 会一并携带它们 - 因此归档不产生额外请求，忽略该指令的模型只是像以前一样为聊天命名。每条第一条消息都会发给模型，无论多短：`git status` 是可用的标题，且完全没有归档，而为其跳过该调用会使最短的对话成为唯一没有标签的对话。之后可以编辑它们：带 `tags` 的 `PATCH /coddy/sessions/{id}` 会整体替换该集合，空数组会清除它，省略该字段则保持原样。列表会按它们过滤（`GET /coddy/sessions?tags=backend,ui` 保留携带其中**任意**标签的会话），搜索会在标题和工作区之外匹配它们，History 会按它们分组。

**没有人写过内容的对话不会被列出。** 会话在第一条提示词之前就会被打开 - 控制台启动时带一个会话，编辑器打开一个线程，Web 聊天应用它选中的文件夹 - 而一个没有提示词就被留下的会话没有任何可回来的内容。History、`GET /coddy/sessions`、`coddy sessions list`、`coddy -c`、控制台选择器、ACP `session/list` 以及 `@session:` 补全都会排除不含消息的会话，除非它被固定；History 会保留首轮仍在准备中的新聊天，因此该聊天从第一次发送起就在列表中。在提示词之前失败的打印运行，在请求打印运行的任何地方仍会被列出，因为脚本可能需要查看它。删除整个历史或归档仍会移除这样的会话包。控制台更进一步，只在其第一条提示词时写入会话（[控制台](../surfaces/console.md)）。

**运行命令的第一条消息**按命令的作用命名。单独输入的 `/rpa-init` 只是一个模型只能重复或猜测的标记，因此对于每个命名技能或聊天工作区内置操作的 `/name` 词，为聊天命名的调用会列出该命令在其目录中携带的描述，模型根据该描述以及命令周围的词语为聊天命名。仅由命令名组成的答案不会被当作标题：会采用命令周围的词语，或在命令单独输入时采用输入的命令本身。消息开头的设置命令（`/model`、`/plan`）用于配置会话，不是工作内容，因此会从消息中移除；仅由这些内容组成的消息不会命名任何东西，聊天保持其现有名称。当调用正在进行时，Web 界面会在聊天标题栏和该聊天的 History 行上显示闪烁占位符，而不是第一条消息（[Web 界面](../surfaces/web-ui.md#会话列表)）。

**模型也会归档会话。** 运行了数小时的对话很少仍是其第一条消息所命名的那一个，而智能体正是注意到这一点的角色：`session_describe` 工具读取当前归档并写入调用指定的部分 - `title` 重命名会话，`tags` 替换整个集合，`add_tags` 和 `remove_tags` 就地更改它并保留其余部分（[工具](../reference/tools.md#会话自身的归档)）。它通过与其他所有内容相同的折叠方式写入，因此答案报告的是存储后的拼写而不是请求的拼写，空的 `title` 会取消固定，并把会话恢复为第一条消息派生的短语。每次写入都是会话自身的一个步骤，合并也包括在内，因此片刻之前从 Web 界面归档的标签不会被工具的 `add_tags` 丢弃，答案的 `changed` 字段来自这些写入，而不是来自比较会话在此之前和之后的样子。

**固定**会把对话保持在每个列表的顶部，无论按什么排序：带 `pinned` 的 `PATCH /coddy/sessions/{id}` 会设置该标志并打上 `pinnedAt`，而只在一种顺序下有效的固定算不上固定。固定项是操作者手动维护的一个列表 - `pinnedRank` 记录它们被拖入的顺序（`POST /coddy/sessions/pins/reorder`），新的固定项位于已有固定项之上，取消固定会忘记其位置，因此再次固定是一个新的固定项，而不是回到旧位置。

**归档**会把对话从工作列表中移出，但不从磁盘上移除。带 `archived` 的 `PATCH /coddy/sessions/{id}` 会设置该标志并打上 `archivedAt`；默认列表会排除这些会话，`archived=only` 是归档，`archived=all` 是全部。其他一切不变：已归档会话的恢复、导出和回退与之前完全相同。清空归档是一个请求，即带 `{"scope":"archived"}` 的 `POST /coddy/sessions/bulk-delete`，而 `all` 范围仍然表示整个历史，包括归档。

**会话来自哪里**也会被记录：`session.json` 携带一个 `origin`，对于在此主机上打开的对话为空，对于信使网关持有的对话为 `gateway:<messenger>`（目前是 `gateway:telegram`），对于一次性打印模式的一次运行为 `print`（`coddy -p`、`coddy -i`）。创建会话的界面会在会话包的第一次写入之前写入它，之后没有任何东西重写它，因此从 Web 界面重新打开 Telegram 聊天不会重新标记它，用 `--session-id <id> -p` 从脚本继续某人的聊天仍使其保持为聊天。`GET /coddy/sessions?origin=local`、`?origin=gateway` 和 `?origin=print` 区分这三者。

**打印运行不碍事。** 脚本和其他智能体一天会调用 `coddy -p` 几十次 - 交叉审查把一份简报发给多个审查者，流水线为每个文件运行一个阶段 - 而每次运行都是一个像其他会话一样存储的会话，因此脚本可以继续它（`coddy -c -p`），或在超时后按 ID 拾取它。人用来选择对话的列表会排除它们，就像排除调度器运行和子智能体子会话一样：History、控制台的 `/resume` 选择器和交互式 `coddy -c`、ACP `session/list`、Telegram 的 `/resume` 键盘、`@session:` 补全以及不带 `include_print=true` 的 `GET /coddy/sessions`。History 在其环境过滤器的 **CLI runs** 下显示它们，而 **Settings → Sessions** 会把它们与其他会话一起列出，并标记终端符号，同时通过其 **Source** 过滤器收窄，那里也是清理它们的地方。`coddy sessions list` 也会列出它们，因为脚本正是读取它来查找一次运行，而 `--origin print` 只保留它们。之后不需要任何东西的运行会传入 `--ephemeral` 标志，其会话在运行结束时被删除（[控制台](../surfaces/console.md#一次性打印模式--p--prompt)）。

这三者都在 Web 界面中可见（[Web 界面](../surfaces/web-ui.md#会话列表)）：History 有一个菜单用于环境、归档状态、分组（日期、文件夹或标签，标题可折叠）和顺序，而 **Settings -> Sessions** 中的会话表可按归档过滤、按列排序，并有一个用于清空归档的按钮。

## Git worktree

Coddy 为会话打开的 worktree 会放到 `<repo>/.coddy/worktrees/<branch>/` - 位于检出目录内，与其他项目本地 Coddy 状态（`.coddy/rules`、`.coddy/agents`、`.coddy/mcp.json`）并列，就像 Claude Code 和 Codex 保存它们的方式一样。分支名通过替换路径不能容纳的字符映射为文件夹名，因此 `feature/login` 会变成 `feature-login`。

该根目录带有自己的 `.gitignore`，其中包含 `*`，在创建第一个 worktree 时写入，并在它丢失时随时放回。它位于 worktree 旁边而不是内部，并忽略其下所有内容，包括忽略文件本身，因此该文件夹不会进入**主**检出目录的 `git status`，也不必向仓库自己的忽略列表添加任何内容。它不会隐藏工作：在 worktree 内部，`git status` 会按预期报告你的编辑。该路径上已有的忽略文件属于操作者，永远不会被重写 - 如果它所说的内容不覆盖 worktree，它们就保持可见，那便是操作者的决定。

`.coddy` 是仓库内容，因此检出目录可以将其作为指向别处的符号链接发布。解析到检出目录之外的 worktree 根目录会被拒绝，而不是被跟随；读起来像 git 选项（`--detach`）或映射不到可用文件夹名的分支名会在 git 看到它之前被拒绝。

位于检出目录内的另一面是，该文件夹在主工作副本中是一个被忽略的路径，因此 `git clean` 会触及它。在 git 2.47 上实测：`-xdf` 会跳过 worktree 本身（"skipping repository"），但确实会删除 `.gitignore`，而 Coddy 会在下一个 worktree 时再次写入它；`-xdff` 会删除整个文件夹，并把条目作为 `prunable` 留下。在采用任一操作之前先运行 `git worktree list`，在其中一个执行之后运行 `git worktree prune`。

智能体的 `worktree_open` 工具和 Web 界面输入框上方面板中的 worktree 复选框使用同一位置。二者都获取 `origin`，并从其最新的默认分支开始一个新的功能分支（该工具在带 `base: "head"` 时也会从当前提交开始）- 只存在于 `origin` 上的分支会从它那里以跟踪方式检出（另一个远程的分支，命名为 `<remote>/<branch>`，同样从该远程检出），已存在的本地分支会在其顶端被重用 - 且二者都不会把默认分支作为功能 worktree 打开。该工具会把活动会话移入 worktree，而 Web 选择器只在第一条消息之前可用。完整流程见 [Git worktree](worktrees.md)，`POST /coddy/sessions/{id}/workspace` 见 [HTTP API](../reference/http-api.md)。

## 一个存储，每个界面

`coddy serve` 为 HTTP API、Web 界面、Telegram 网关和调度器在同一个存储上运行一个会话管理器，因此与机器人的聊天是一个普通会话，History 抽屉会在它进行时列出并打开它，而在一个界面上开始的轮次会镜像到正在观看同一会话的浏览器中（[coddy serve](../operate/serve.md)、[Telegram 网关](../surfaces/gateway.md)）。控制台和 `coddy acp` 各自打开同一个存储；使用 `--remote` 时，它们处理服务器上的会话，而不是本地存储。集群中继会聚合其节点的会话列表（[集群](../operate/swarm.md)）。

一个会话一次只运行一个轮次。轮次锁是会话包上的文件锁，因此第二个进程 - 当 `coddy acp` 持有轮次时的 `coddy serve` - 会被拒绝，而不是交错执行（通过 HTTP 返回 `409`），取消还会向会话包写入一个标记，持有轮次的进程会在轮询之间拾取它。不同会话并行流式传输，各自位于自己的锁之后（[Web 界面](../surfaces/web-ui.md#并行会话与生成取消)）。

一类 HTTP 请求在此存储之外运行：不带 `X-Coddy-Session-ID` 调用 `models[].model` 选择器是无状态的，不会留下会话包（[HTTP API](../reference/http-api.md#model配置文件与直接完成)）。

通过 `--remote` 连接到同一服务器的浏览器和控制台可以停止轮次，或添加和移除[排队的后续消息](message-queue.md)，包括另一个客户端启动的轮次。它们在重新连接时会恢复活动和队列状态；丢失流连接并不意味着会话已停止。成功的取消响应会确认请求，而轮次的完成会确认其锁已释放。HTTP 配置在获取该锁时、工作区准备之前安装取消钩子；在该窗口中的停止操作在运行器被接纳时仍然有效。共享权限/问题答案以及远程控制台中的实时外部轮次对话记录不属于这些控制。仅共享会话包的独立本地进程仍然有独立的队列和待回答通道。

## 恢复

| 界面 | 方式 |
|---|---|
| 控制台 | `coddy -c`（`--continue`）重新打开为当前文件夹记录的最近会话，排除打印运行，若不存在则失败；`--session-id <id>` 重新打开该会话，或在该 ID 下创建一个；`--resume` 和 `/resume` 打开一个选择器，按标题和最后更新列出该文件夹的会话，也排除打印运行；`-c -p "..."` 以打印模式继续，包括上一次打印运行。退出时会打印 ID 以及恢复它的命令。在 `--remote` 下，它们都处理服务器上的列表，而不带文件夹过滤（[控制台](../surfaces/console.md)）。 |
| Telegram | `/resume` 在服务器保留的会话上打开内联键盘，最新的在前，显示标题并标记聊天自己的会话；`/resume <id or title>` 继续词语所指的会话，并在多个匹配时提供键盘。聊天在 `gateway_sessions.json` 中绑定到该选择，因此网关重启后仍然保持（[Telegram 网关](../surfaces/gateway.md#命令)）。 |
| Web 界面 | **History** 抽屉（`#/history`）按最后活动列出会话，固定项在前，除非过滤菜单另有说明否则按文件夹分组，并让归档不挡路；搜索会遍历标题、文件夹、标签和第一条提示词，当会话正在工作时该行带一个圆点，当它在你关注别处时完成时带另一个圆点，当它等待权限或回答时带一个问号；一行会打开 `#/s/<id>`，仅凭该 URL 就能让会话回来（[Web 界面](../surfaces/web-ui.md#会话列表)）。 |
| ACP 编辑器 | `session/load` 恢复 `session.json` 和 `messages.json`，重放轮次和工具调用摘要，当 `todos/active.md` 存在时发送 `plan` 更新，然后发送命令目录；`session/list` 列出会话包，并可带可选的 `cwd` 过滤器，该过滤器无论文件夹路径如何拼写都会匹配；只要配置了存储，就会通告 `loadSession`。`coddy acp --session-id <id>` 使下一个 `session/new` 重新打开该会话包，或在该名称下创建一个（[ACP 协议](../reference/acp-protocol.md#sessionload)）。 |
| HTTP | `GET /coddy/sessions`（`limit`、`cursor`、`q`、`include_scheduler`、`include_subagents`、`include_print`、`include_activity`）、`GET /coddy/sessions/{id}/messages`，以及在 `POST /v1/responses` 上使用 `X-Coddy-Session-ID` 继续会话（[HTTP API](../reference/http-api.md)）。 |

![History 抽屉](../../assets/screenshot-fullhd-history.png)
*起始屏幕上的 History 抽屉：固定的对话在最前，每个文件夹一组，标签位于标题下方，过滤按钮位于搜索旁边。*

列表按 `updatedAt` 排序，最新的在前；当有内容被持久化时时间戳会更新 - 一个轮次、一个固定的标题 - 而仅仅为读取而加载会话包时不会更新。重新打开会话会使用 `source: resume` 再次运行 `SessionStart` 钩子（[钩子](hooks.md#事件)）。

## 编辑消息时回退

编辑已经发送的消息会就地重写对话。Web 界面中用户气泡下方的编辑控件会把该消息重新加载到输入框；发送它会以该用户消息的从 0 开始的索引调用 `POST /coddy/sessions/{id}/rewind`，服务器会在该点截断对话记录 - 其后的消息以及它们产生的 UI 日志行会离开对话记录，它们的工具调用记录会移入撤销快照，被移除调用的待处理权限提示会被丢弃 - 并把编辑后的文本作为同一会话的下一个轮次发送。没有同级对话，也没有可导航的内容：编辑点之后所回答的内容会离开对话记录，这正是更正的含义，并且只有最后一次编辑可以撤回（见下文）。

当消息位于输入框中时，Web 界面会提示这一点：字段上方的横幅会指明该消息，气泡会有轮廓，发送后将移除的行会变淡。按 Escape 或横幅上的叉号会离开编辑，并放回编辑控件替换掉的草稿；发送编辑也会把该草稿放回。

已发送的编辑可以撤回。回退会把它在会话包中切掉的尾部保留下来（`rewind_undo/`：消息、它们的 UI 日志行以及工具调用记录），而 `POST /coddy/sessions/{id}/rewind/undo` 会恢复它：编辑后的提示词启动的轮次会先被取消，切点之后的所有内容会被丢弃，对话再次回到编辑之前的样子。只要 `GET /coddy/sessions/{id}/messages` 携带 `rewindUndo`，撤销就保持可用 - 直到编辑后的提示词之后出现第二个提示词、另一次回退替换它，或一次压缩重写保留的前缀。被移除轮次更改的文件不会被还原：撤销恢复的是对话，不是工作区。

当轮次正在进行时，`POST /coddy/sessions/{id}/rewind` 会以 **409** 被拒绝，而子智能体子会话对它以及其他一切操作都是只读的（[Web 界面](../surfaces/web-ui.md#消息编辑与历史回溯)）。用户消息索引只计算真实用户轮次 - 压缩摘要行带有用户角色但不是轮次，而 `background_wake` 行是轮次。同一服务器上的每个浏览器标签页都会在 `GET /coddy/events` 上听到 `session_rewound` 并重新获取；使用 `--remote` 附加的控制台、ACP 客户端或信使网关不会订阅它，并会在下次加载时读取截断后的历史。被丢弃轮次派生的子会话会将其会话包保留在 `subagents/` 下 - 运行对话记录会保留以供审计，即使产生它们的轮次已经消失。

## 待办清单

`coddy_todo_*` 工具（`plan_read`、`plan_replace`、`plan_archive`、`item_add`、`item_update`、`item_remove`、`item_move`，状态为 `pending`、`in_progress`、`completed`、`failed` 和 `cancelled`）维护会话的清单。每次更改都会镜像到 `todos/active.md`，并作为 `plan` 更新发布给客户端，且当列表非空时，系统提示词会携带一个 `### Current todo checklist` 块，在轮次的每次模型调用之前刷新，因此轮次中途勾选的条目会立即可见。列表为空时没有这样的块，且 `coddy_todo_plan_read` 不会返回空结果，而是回答一条说明没有活动计划的通知，并指明 `coddy_todo_plan_replace`。有些模型在收到空响应时会重复同一调用，直到[循环防护](../operate/security.md#循环防护与上限)停止轮次。`item_update`、`item_remove` 和 `item_move` 对空列表返回 `plan is empty` 错误。Ask 模式不提供待办工具。

替换由两条规则约束。当任何条目不是 `completed` 时，`coddy_todo_plan_replace` 会被拒绝 - 先完成这些行或归档 - 且当每一行都完成时，先前的 `active.md` 会在写入新列表之前移动到 `todos/archive/todo-<nanos>.md`。`coddy_todo_plan_archive` 会把未完成的行标记为完成，写入 `todos/archive/plan_<unix_seconds>.md` 并清除活动列表。通过 HTTP，同一列表是 `GET` 和 `PUT /coddy/sessions/{id}/plan`，以及 `POST /coddy/sessions/{id}/plan/archive`。对对话记录的每次读取 `GET /coddy/sessions/{id}/messages` 都会以 `todoPlan` 携带该列表，Web 界面会在最新消息下方显示它（[Web 界面](../surfaces/web-ui.md#对话记录下方的待办清单)）。

在计划模式中编写的设计计划是单独的文件，即 `plans/<slug>.plan.md`，它们本身永远不会填充待办清单（[ACP 协议](../reference/acp-protocol.md#设计方案计划模式)）。

## 子会话与运行会话包

子智能体运行是一个子会话：位于 `<parent>/subagents/<child>/` 下的真实会话包，其 `session.json` 携带 `subagentRun`、`parentSessionId`、`subagentName`、`subagentTaskId` 和 `subagentDepth`。子会话不会出现在任何默认列表中 - History、`GET /coddy/sessions`、`coddy sessions list`、`coddy -c`、ACP `session/list` - 且是只读对话记录：针对它的提示词、回退或权限回答都会被拒绝，并指明父会话（[子智能体](subagents.md#子会话)）。调度器任务拥有一个**任务会话**（`session.json` 中的 `schedulerRun` 和 `schedulerJobId`，从输入框列表中隐藏，且自身为只读），任务的每次运行都是它的子会话 - 一个子会话包，其 `session.json` 还携带 `schedulerJobId`、`schedulerTrigger` 和 `schedulerFireSlot`。`scheduler.retain_sessions` 为每个任务保留最新的已完成运行，删除任务会删除其会话树（[调度器](../operate/scheduler.md)）。

## 删除会话

在 History 行的 ⋮ 菜单中选择 **Delete**，经过一次确认后，以及 `DELETE /coddy/sessions/{id}`，都会移除会话树：该会话及其派生的每个子会话，先停止它们的后台任务，最深的会话包最先移除。正在运行的轮次会被取消并等待完成，然后才移除任何内容。一次删除多个（包括归档）可从 **Settings → Sessions** 进行。命令行上没有删除动词；对于没有进程持有的会话，手动删除目录是等效的。

## 导入 OpenCode v2 会话

**设置 → OpenCode** 用于配置远程 OpenCode v2 服务器：API 地址、HTTP Basic Auth
用户名（默认 `opencode`）和密码。**测试连接** 会读取 `/api/info`、
`/api/project` 和项目的会话页；**导入全部会话** 会作为后台任务执行迁移，并在同一面板
显示进度。密码是只写的，连接保存在 `${CODDY_HOME}/opencode.json`（仅所有者可读），
不写入 `config.yaml`。

导入是单向的，且只使用 API：读取 `/api/project`、`/api/session` 和
`/api/experimental/session/:id/export`，绝不打开 OpenCode 数据库。同样的迁移也可以
不用 Web 界面执行：

```bash
coddy sessions import --from opencode \
  --url https://opencode.example.com:4096 \
  --username opencode --password-stdin \
  --all-projects --dry-run
```

去掉 `--dry-run` 才会写入会话包；`--save` 会保存连接供以后使用。服务器必须是
OpenCode v2，且能从此 Coddy 进程访问。如果会话记录的工作区已不存在，它会作为只读归档
导入，并带有 `opencode-missing-workspace` 标签。子会话保留 `subagents/` 下的父子布局，
提示词文件和图片写入会话的 `assets/`，最新任务快照保存在
`${CODDY_HOME}/opencode-import/jobs/`，因此重启后会把任务报告为中断，而不是静默丢失。
已存在的会话包默认跳过，除非给出
`--replace`。导入器不会复制 API key、权限、插件或快照，旧工具调用只是历史，绝不会
重新执行。导入的 OpenCode 工具调用会改写成 Coddy 的工具名和参数形状（`bash` 变成
`run_command`，`filePath` 变成 `path`，OpenCode 的毫秒 shell 超时换算成秒）；Coddy
没有对应工具时，调用及其结果会被丢弃，避免续聊时模仿不存在的工具。

## sessions CLI

```bash
coddy sessions list [--sessions-dir <path>] [--cwd <filter>] [--origin local|gateway|print]
coddy sessions export <session-id> [--format md|html|json|jsonl] [--out <path>] [--no-tools] [--no-thinking] [--sessions-dir <path>]
```

`list` 会为每个存储的会话打印一行制表符分隔的记录 - `SESSION_ID`、`UPDATED_AT`、`CWD`、`TITLE` - 最新的在前，然后是 `(total N)`；调度器运行和子会话会被排除，打印运行会被列出，`--cwd` 只保留以该绝对目录保存的会话，`--origin` 只保留某个界面的会话。`export` 会把已存储会话的对话记录写入文件，并接受唯一 ID 前缀代替完整 ID；格式、路径规则和裁剪选项见[会话导出](session-export.md)。

## 测试

- `features/` 中的可执行规格：`print_sessions.feature`（打印运行会被标记并排除在选择器之外，`-c -p` 会继续它，交互式 `-c` 会跳过它，`--ephemeral` 不留下任何内容；框架 `external/cli/bdd_print_sessions_test.go`）、`acp_session_integration.feature`（重新打开的会话包会在 `session/new` 响应之后重放其对话记录；框架 `internal/session/bdd_acp_session_test.go`）、`session_rewind.feature`（对话记录在编辑后的消息处被截断；`external/httpserver/bdd_rewind_test.go`）、`session_rewind_undo.feature`（撤销会带回原始轮次；同一框架）、`session_export.feature` 和 `session_export_cli.feature`（[会话导出](session-export.md)）。
- 单元测试：`internal/session/filesystem_test.go`（会话包布局和列表）、`internal/session/rewind_test.go`（就地截断和产物清理）、`internal/session/rewind_undo_test.go`（保留的尾部、撤销何时结束、恢复的 UI 日志和工具调用记录）、`external/cli/continue_test.go`（`-c` 解析），以及 SPA 中的 `external/ui/src/ui/messages/userMsgIndices.test.ts`（编辑或回退所指的用户消息索引，一次唤醒也算作一个轮次）。

<!-- docsgen:source sha256=32744aaffa72cbe4 -->

# 工具

模型可调用的内置工具、它们的参数、权限以及暴露它们的模式。名称、参数名和权限标志取自 `internal/tools` 下的构造函数以及可选包；`internal/tools.NewRegistryFor` 由这些构造函数构建注册表，而 `internal/agent.ToolSetForMode` 在 `internal/agent/toolsets.go` 中决定每种模式向模型发送哪些定义。Agent 模式不受限制；plan 和 ask 带有固定允许列表，并且二者都会在执行时重新检查其列表，因此从历史重放的调用无法越过它。这些模式背后的理由见[运行模式](../features/modes.md#三种模式)；如何添加工具见[自定义工具](../contributing/custom-tools.md)。

## 权限

权限列使用 `internal/agent/react.go` 中的门控在会话权限模式下应用的类别（在另选之前为 `ask`，`accept_edits`、`bypass`；见[安全与信任](../operate/security.md#权限模式与提示)）：

- `none`：从不提示；
- `write`：文件写入类，在 `ask` 下会提示，除非该路径已在本会话中获得授权；在 `bypass` 下自动批准，并且在 `accept_edits` 下于工作区内自动批准；
- `command`：`run_command`，除非命令被 `tools.command_allowlist` 或会话授权覆盖，否则会提示（包括提示可存储的、程序范围的授权，见[后台任务](../features/background-tasks.md#权限)）；`bypass` 会跳过提示；
- `config`：`config_commit` 和 `config_rollback`，在 `ask` 和 `accept_edits` 下会提示，因为提交可以启动 MCP 进程并改变权限策略本身；只有 `bypass` 会跳过它；
- `http`：`http_request`，在 `ask` 和 `accept_edits` 下会提示，除非目标位于 `tools.http_request.allowlist`，或已在本会话中获批，并且请求携带的所有内容（本地文件、代理、未经检查的证书，以及在 `ask` 下它保存的文件）也随之获批；`bypass` 会跳过它（[HTTP 请求](../features/http-requests.md#权限)）；
- `always`：在这些类别之外设置 `RequiresPermission` 的工具，权限模式不会改变它们，只有应答 `allow` 的 PreToolUse 钩子会跳过提示。

## 文件

当 `browser.enable` 为 true 时，浏览器工具会在 Agent 模式下提供，且位于使用 `browser` 标签构建的二进制中。它可以访问连接到此服务器的浏览器、连接到本机服务器的浏览器（在其 `browser.serve_remote` 下）以及 `browser.remotes` 的服务器。

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `browser` | 通过 Coddy for Browser 扩展在用户浏览器中工作：列出已连接的浏览器和可访问的标签页，打开、读取并对标签页执行操作，在页面与工作区之间移动文件（[Coddy for Browser](../features/browser.md)） | `action`、`browser_id`、`tab_id`、`params`、`path` | 始终，外加扩展中的网站批准 | agent |

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `read` | 以文本形式读取文件，可选的从 1 开始的行范围，或列出目录；其他编码（UTF-16、旧代码页）的文本会转换为 UTF-8。PNG、JPEG、GIF 或 WebP 文件（根据内容判断）会作为图片展示给具有 `models[].multimodal: true` 的模型，最大 3.75 MB、每边 8000 像素，并且仅限完整到文件末尾的文件，GIF 以其第一帧展示，Web 界面和 Telegram 聊天也会显示它（[图片](../features/images.md)）；任何其他二进制文件（PDF、归档文件），以及提供给没有 `multimodal` 的模型的图片，都会连同其类型和大小被拒绝 | `path`、`offset`、`limit`、`recursive`、`show_hidden`、`keep` | 无 | agent、plan、ask |
| `keep_result` | 将已读取的页面或 grep 结果标记为有用，以便结果逐出时保留它（[上下文压缩](../features/compaction.md#结果驱逐)） | `path`、`offset`、`limit`、`pattern` | 无 | agent、plan、ask |
| `glob` | 按 glob 模式查找文件，最新优先，最多 100 个路径 | `pattern`、`path` | 无 | agent、plan、ask |
| `grep` | 对文件内容进行正则表达式搜索 | `pattern`、`path`、`glob`、`case_sensitive`、`max_results`、`keep` | 无 | agent、plan、ask |
| `print_tree` | 打印类似 `tree` 的目录树 | `path`、`depth` | 无 | agent、plan、ask |
| `edit` | 替换精确的文本范围；保留行尾 | `path`、`oldString`、`newString`、`replaceAll` | 写入 | agent |
| `write` | 创建或覆盖文件；会创建父目录 | `path`、`content` | 写入 | agent |
| `apply_patch` | 应用统一 diff 或 Codex V4A 补丁 | `path`、`patch` | 写入 | agent |
| `mkdir` | 创建目录；`parents` 的作用类似于 `mkdir -p` | `path`、`parents` | 写入 | agent |
| `rmdir` | 删除空目录 | `path` | 写入 | agent |
| `touch` | 创建空文件或刷新其修改时间 | `path`、`create_parents` | 写入 | agent |
| `rm` | 删除文件，或使用 `recursive` 删除整棵树 | `path`、`recursive` | 写入 | agent |
| `mv` | 移动或重命名；会创建目标父目录 | `src`、`dst` | 写入 | agent |

`write`、`edit`、`apply_patch` 以及 `mv` 跨磁盘进行的复制会以原子方式替换文件：新内容写入同一文件夹中的临时文件，在写入并刷新后，该临时文件会取代目标文件。中途失败的写入（磁盘已满）会使文件保持原样，模型读到的错误会说明这一点。进程被终止时，要么留下旧内容，要么留下全部新内容；在重命名前被终止的进程会在文件旁留下一个名为 `<name>.<random>~` 的临时文件。新文件保留旧文件的权限（其模式、Linux 上的 POSIX ACL、其 DACL、Windows 上的隐藏、系统和未编入索引属性及其创建时间），而其所有者会变为运行 Coddy 的用户。符号链接会被穿透写入：链接保留，它指向的文件发生变化。具有多个硬链接的文件会以写入时使用的名称成为新文件，其他名称保留旧内容，结果行会说明这一点。命名管道或设备会就地写入，指向尚不存在文件的链接也是如此，因为它没有旧内容需要保留。

## Shell 与后台任务

`background_*` 工具仅在启用 `tools.background`（默认）时注册，`preview_server` 还在启用 `tools.preview_server` 时注册。`canvas_show` 和 `canvas_inspect` 在启用 `tools.canvas`（默认）时注册。指南见[后台任务](../features/background-tasks.md#模型可见的接口)。

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `run_command` | 通过主机 shell 在工作区中运行 shell 命令；可选的 `cwd` 仅为该命令更改目录。`background: true` 返回任务 id，并在可用时默认于完成时唤醒智能体。超过超时时间的前台命令会被移交给池，并且默认也会在完成时唤醒；`notify_on_finish: false` 会禁用任一唤醒 | `command`、`cwd`、`permission_rationale`、`timeout_seconds`、`background`、`notify_on_finish`、`expected_seconds` | 命令 | agent、plan |
| `share_file` | 将工作区中的一个普通、非符号链接文件复制到当前持久化会话中，作为不可变的可下载产物。该副本按内容寻址，上限为 25 MiB（每个会话 100 MiB），并且不会进入模型提示词内容或对话记录导出。子智能体无法使用它。 | `path` | 始终 | agent |
| `worktree_open` | 将会话移入持有某分支的 [worktree](../features/worktrees.md)，或为其打开一个（获取 origin；从默认分支创建新分支，或使用 `base: "head"` 从会话所在的提交创建）；改用 `path` 时，进入现有的 worktree，也可以进入分离的 worktree，不创建也不获取任何内容。一个 `branch` 若写成 `<remote>/<name>`，会打开跟踪该远程分支的本地 `<name>`。新 worktree 会获得主检出目录的 `.worktreeinclude` 所指定的被忽略文件的副本，列在 `included` 中 | `branch` 或 `path`，`base` | 始终 | agent |
| `worktree_list` | 列出会话仓库的 worktree，主检出目录在前，并说明每个 worktree 的状态：分支、最后一次提交、未提交和未推送的更改、相对默认分支的领先和落后、已锁定或已消失、哪一个是会话的，以及其他会话分别在哪个 worktree 中工作 | 无 | 无 | agent、plan、ask |
| `worktree_exit` | 将会话从其 [worktree](../features/worktrees.md) 移回主检出目录。`action` `auto`（默认）会在 worktree 不包含自身内容时删除该 worktree 及其分支，否则保留它，并在 `kept` 中说明原因；`keep` 保留它；`remove` 会删除它或拒绝：当存在未提交更改时会拒绝删除，除非使用 `discard_changes`（这些更改会作为快照保存在 `refs/coddy/worktree-snapshots/` 下，`git stash apply --index` 可将其恢复），并且在提交未被任何分支、远程分支或标签持有时始终拒绝，在 worktree 内有已初始化子模块或自身 git 仓库时拒绝，在 worktree 已锁定时拒绝，在本进程的另一个会话或后台任务正在其中工作时拒绝，以及在 Coddy 未打开的 worktree 上拒绝；仅当另一个分支、远程分支或标签持有其每个提交时，分支也会一并删除 | `action`、`discard_changes` | 始终 | agent |
| `background_list` | 列出会话的后台任务，包含状态、已用时间和预估 | 无 | 无 | agent、plan |
| `background_output` | 返回任务捕获的 stdout 和 stderr | `task_id`、`tail_lines` | 无 | agent、plan |
| `background_wait` | 等待任务完成，受默认值和最大值限制 | `task_id`、`timeout_seconds` | 无 | agent、plan |
| `background_stop` | 终止任务及其生成的所有内容 | `task_id` | 无 | agent、plan |
| `background_reap` | 查找并终止本会话中比启动它们的运行存活更久的进程 | 无 | 始终 | agent |
| `preview_server` | 将项目的一个目录作为静态网站提供在空闲的 localhost 端口上，并返回地址供用户打开；该服务器是一个后台任务，运行直到被停止（[预览服务器](../features/preview-server.md)） | `path`、`timeout_seconds` | 无 | agent |
| `canvas_show` | 将工作区中的 HTML 页面放到画布上，画布是一个 `.canvas` 文件（JSON Canvas，Obsidian 的格式），Web 界面会在其文件窗口中打开它，可作为设备尺寸或自定义尺寸的实时框架，也可将框架指向该页面；没有 `file` 时，页面会进入聊天的画布或新的 `canvas/<folder>.canvas`，而将 `.canvas` 文件作为 `path` 会令该文件成为聊天的画布（[画布](../features/canvas.md)） | `path`、`file`、`title`、`device`、`width`、`height`、`root`、`frame` | 无 | agent |
| `canvas_inspect` | 读取运行画布框架的浏览器所报告的内容：是否已加载、其标题、控制台错误和警告、未捕获的异常；等待关于当前文件的报告；`file` 指定画布名称，默认为聊天的画布 | `file`、`frame`、`wait_seconds` | 无 | agent |
| `ssh_run_command` | 通过 SSH 在远程主机上运行命令，先使用 agent 套接字，然后使用密钥文件（[配置](../getting-started/configuration.md#ssh-远程执行)） | `host`、`command`、`port`、`timeout_seconds`、`permission_rationale` | 始终 | agent |

## Web

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `websearch` | 一次搜索多个引擎并合并结果，报告每个引擎的结果 | `query`、`page`、`max_results`、`site` | 无 | agent、plan、ask |
| `webfetch` | 下载公共页面并以 Markdown 返回其正文；私有网络和 localhost 会被拒绝，包括每一次重定向 | `url`、`timeout_seconds`、`max_chars` | 无 | agent、plan、ask |
| `http_request` | 发送任意 HTTP 或 HTTPS 请求，类似 curl，并返回状态行、标头和正文，或将正文保存到文件（[HTTP 请求](../features/http-requests.md)） | `url`、`method`、`query`、`headers`、以下之一 `body` / `body_base64` / `body_file` / `json` / `form` / `form_data`、`output_file`、`follow_redirects`、`proxy`、`verify_tls`、`timeout_seconds`、`permission_rationale` | http | agent |

`websearch` 会询问 `tools.websearch.engines` 中指定的引擎（默认先 Brave 后 Bing），并在结果旁报告每个引擎的结果，因此返回挑战页面的引擎会被点名，而不会被计为未找到任何内容；如果所有引擎都被拒之门外，搜索会失败，而不是返回空列表。存在哪些引擎、相关性门控如何丢弃无关结果集，以及如何将其指向你自己的 SearXNG：[Web 搜索](../features/web-search.md)。

`http_request` 是处理其他所有 HTTP 事务的工具：API、localhost 上的服务、上传和下载。它不拒绝任何地址，这就是它受门控的原因；`webfetch` 共享其客户端，但在 SSRF 防护、固定的 `GET` 和 Markdown 转换方面有所不同（[HTTP 请求](../features/http-requests.md#同一客户端上的-webfetch)）。操作员希望在每个请求中携带的标头（例如浏览器 `User-Agent`）会放入 `tools.http_request.default_headers`，只有 `http_request` 会发送它，而调用自身的 `headers` 会覆盖它（[默认标头](../features/http-requests.md#默认请求头)）。

## 交互、技能与子智能体

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `question` | 向用户提出一个或多个多项选择题并等待答案；问题也接受用户输入的答案，除非它将 `custom` 设为 `false`，因此模型不会自行提供“其他”选项 | `questions[]`（属于 `header`）、`question`、`options[]`（`label`、`description`）、`multiple`、`custom`（默认 `true`） | 无（该轮次等待用户） | agent、plan、ask |
| `load_skill` | 按名称加载已编目技能的完整说明；仅在 `skills.auto_discovery` 开启时注册（[技能](../features/skills.md)） | `name` | 无 | agent、plan、ask |
| `coddy_docs_search` | 搜索 Coddy 自带的、内置于二进制中的文档，对章节使用 BM25，支持英语、俄语或中文（[内置文档](../features/built-in-docs.md#智能体的工具)） | `query`、`limit`、`lang` | 无 | agent、plan、ask |
| `coddy_docs_read` | 读取该文档的页面或章节，或在没有页面时读取其目录，使用该轮次的语言，除非 `lang` 指定了语言；长页面会分成多个部分 | `page`、`offset`、`lang` | 无 | agent、plan、ask |
| `spawn_agent` | 将自包含任务委托给子智能体，等待其报告或将其作为后台任务运行。分离的子进程在可用时默认于完成时唤醒父进程；`notify_on_finish: false` 会禁用它。`resume` 会在其自己的子会话中继续已完成的运行，而不是从头开始，`isolation: "worktree"` 会在其自己的 git worktree 中运行子进程（[子智能体](../features/subagents.md#spawn_agent-工具)） | `agent`、`prompt`、`description`、`model`、`reasoning`、`background`、`expected_seconds`、`timeout_seconds`、`notify_on_finish`、`isolation`、`resume` | 自身无；子进程自己的调用会通过父进程提示 | agent、plan（plan 模式父进程的子进程保持 plan 模式）；在 ask 中隐藏，在 `subagents.max_depth` 处，除非会话自身的定义声明了 `spawns` 允许列表，以及不存在运行时的情况，例如定时运行 |
| `switch_model` | 仅在用户要求时更改模型或推理级别；默认从该会话的下一个请求开始应用，或使用 `scope: turn` 仅应用于本轮（[会话设置](../features/session-settings.md#按请求更改模型)） | `model`、`reasoning`、`scope`（`session` 或 `turn`） | 无 | agent、plan、ask；从不提供给子智能体 |

## 待办清单

会话的计划文档以清单形式呈现，持久化在捆绑包中，并作为 ACP `plan` 更新显示（[会话](../features/sessions.md#待办清单)）。仅 Agent 模式。

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `coddy_todo_plan_read` | 读取清单但不更改；没有清单时，会回答一条说明没有活动清单的备注，并指出 `coddy_todo_plan_replace`，从不返回空结果 | 无 | 无 | agent |
| `coddy_todo_plan_replace` | 从 Markdown 替换整个清单 | `markdown` | 无 | agent |
| `coddy_todo_plan_archive` | 完成清单：未完成项标记为已完成，计划归档 | 无 | 无 | agent |
| `coddy_todo_item_add` | 添加一项，位于末尾或某个索引之后 | `content`、`status`、`after_index` | 无 | agent |
| `coddy_todo_item_remove` | 按从零开始的索引移除一项 | `index` | 无 | agent |
| `coddy_todo_item_update` | 更改一项的内容或状态 | `index`、`content`、`status` | 无 | agent |
| `coddy_todo_item_move` | 移动一行 | `from_index`、`to_index` | 无 | agent |

## 设计计划

计划文档位于会话捆绑包内的 `plans/<slug>.plan.md`（[运行模式](../features/modes.md#plan-模式与计划文档)、[ACP 协议](acp-protocol.md#设计方案计划模式)）。

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `plan_write` | 写入或替换计划文档 | `slug`、`content` | 无 | agent、plan |
| `plan_list` | 列出会话的计划文档 | 无 | 无 | agent、plan |
| `plan_read` | 读取一个计划文档 | `slug` | 无 | agent、plan |
| `plan_exit` | 将会话从 plan 模式切换到 agent 模式 | 无 | 无 | agent（不在 plan 允许列表中） |

## 上下文

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `compact_context` | 将较早的历史折叠为摘要，使会话持续适配模型的上下文窗口（[上下文压缩](../features/compaction.md#模型可以请求压缩)） | `instructions`、`model`（仅用于此次调用的摘要器，仅在用户指定时传入：已配置的模型 id、不带提供商名称的模型名称，或能精确匹配一个模型的部分名称） | 无 | agent、plan；不在 ask 中，ask 的工具保持只读；当 `compaction.enable` 为 false 时隐藏 |

## 会话自身的归档

会话列出时所依据的标题及其分组标签（[会话](../features/sessions.md#标签与归档)）。该工具归档的会话就是它运行所在的会话；子智能体会归档自己的子会话，绝不会归档父会话。

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|------|---------|-------------------|------------|-------|
| `session_describe` | 读取会话的标题和标签，并可更改任一者；无参数调用时仅报告它们 | `title`、`tags`、`add_tags`、`remove_tags` | 无 | agent、plan；不在 ask 中，ask 的工具保持只读 |

## 自配置

对实时 `config.yaml` 的分阶段编辑（[config.yaml 参考](config.md#智能体自配置)）。只有在界面接入了配置重载器时才会提供编辑系列；没有重载器时，`config_get` 是唯一剩下的成员。

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `config_get` | 按点分 uci 风格路径读取经过脱敏的值；`.` 表示整个文件 | `path` | 无 | agent、plan |
| `config_changes` | 列出提交将应用的分阶段命令 | 无 | 无 | agent、plan |
| `config_set` | 针对活动文件暂存类似 uci 的命令：`set <path>=<value>`、`add_list <path>=<value>`、`del_list <path>=<value>`、`delete <path>` | `commands[]` | 无 | agent |
| `config_revert` | 丢弃分阶段命令，全部或某个路径下的命令 | `path` | 无 | agent |
| `config_commit` | 验证批次，为之前的文件创建快照并应用；热重载正在运行的进程 | 无 | 配置 | agent |
| `config_rollback` | 恢复上一次提交写入的快照 | 无 | 配置 | agent |

## 调度器

使用 `scheduler` 标签编译进来，并且仅在调度器启用时注册（`internal/tools/scheduler_hook.go`、`external/scheduler/tools/register.go`）。`job_id` 是文件基本名；每个针对单个任务的工具接受 `scope`、`user`（默认，`${CODDY_HOME}/scheduler`）或 `project`（`<session cwd>/.coddy/scheduler`，仅在操作员批准后才运行的任务；没有工具会批准它）；任务字段即 frontmatter 中的字段（[调度器](../operate/scheduler.md#工具启用调度器时)）。仅 Agent 模式。

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `coddy_scheduler_jobs_list` | 列出会话工作区以及调度器运行的工作区中的用户任务和项目任务，并注明各自的范围和信任状态 | `include_body` | 无 | agent |
| `coddy_scheduler_job_get` | 加载一个任务 | `scope`、`job_id` | 无 | agent |
| `coddy_scheduler_job_runs` | 列出任务的运行，最新优先：任务会话下的任务、保存对话记录的运行会话、触发器、状态和时间 | `scope`、`job_id`、`limit` | 无 | agent |
| `coddy_scheduler_job_create` | 创建任务文件 | `scope`、`job_id`、`description`、`schedule`、`paused`、`cwd`、`model`、`mode`、`agent`、`permission_mode`、`body` | 始终 | agent |
| `coddy_scheduler_job_replace` | 替换任务的所有字段 | `scope`、`job_id`、`description`、`schedule`、`paused`、`cwd`、`model`、`mode`、`agent`、`permission_mode`、`body` | 始终 | agent |
| `coddy_scheduler_job_patch` | 仅更改给定字段，可选择重命名任务 | `scope`、`job_id`、`new_job_id`、`description`、`schedule`、`paused`、`cwd`、`model`、`mode`、`agent`、`permission_mode`、`body` | 始终 | agent |
| `coddy_scheduler_job_pause`、`coddy_scheduler_job_resume` | 设置或清除 `paused` | `scope`、`job_id` | 始终 | agent |
| `coddy_scheduler_job_delete` | 删除任务、其 `.state` 附属文件及其运行历史，前提是没有正在进行的运行 | `scope`、`job_id` | 始终 | agent |
| `coddy_scheduler_job_run` | 立即启动一次运行，作为任务会话下的后台智能体任务；返回任务和运行会话 | `scope`、`job_id` | 始终 | agent |
| `coddy_scheduler_job_cancel` | 停止正在进行的任务运行 | `scope`、`job_id` | 始终 | agent |

## 记忆子智能体

在启用 `memory` 标签和 `memory.enable` 时，模型会通过 `spawn_agent` 按需委托给记忆子智能体，并使用 `agent: "memory"`，这是一个在后台任务池中运行的普通子智能体，而这些是它的工具：它们只会注册到该子进程的注册表中（`internal/agent/memory_hooks.go`、`external/memory/tools`），因此主模型永远不会看到它们。仅进行回忆的子进程（ask 模式轮次）会获得前三个，其他每个子进程获得全部六个。路径为 `scope:relative/path.md`（[长期记忆](../features/memory.md#工具)）。

| 工具 | 用途 | 参数（简） | 权限 | 模式 |
|---|---|---|---|---|
| `coddy_memory_search` | 根据查询对所选根目录下的笔记进行排序 | `query`、`scope` | 无 | 记忆子进程 |
| `coddy_memory_list` | 列出路径下一层的目录和笔记 | `path` | 无 | 记忆子进程 |
| `coddy_memory_read` | 读取一条笔记 | `path` | 无 | 记忆子进程 |
| `coddy_memory_mkdir` | 在范围内创建嵌套文件夹 | `path` | 无 | 记忆子进程 |
| `coddy_memory_save` | 写入或覆盖笔记 | `title`、`body`、`scope`、`relative_path` | 无 | 记忆子进程 |
| `coddy_memory_delete` | 删除笔记或文件夹及其下所有内容，绝不会删除根目录 | `path` | 无 | 记忆子进程 |

## MCP 工具

每个已连接 MCP 服务器的每个已启用工具都会以 `server__tool` 名称加入同一个函数调用列表，使用服务器自己的输入 schema，描述前缀为 `[server]`（`internal/mcp/client.go`；[MCP 服务器](../features/mcp.md#工具命名空间)）。由于 `__` 是分隔符，服务器名称不得包含它。这些定义会在 agent 和 plan 模式下追加，从不在 ask 模式下追加；在 ask 模式下，对这类名称的调用会在执行时被拒绝；在 plan 模式下，名称会通过该检查，MCP 层会像在 agent 模式下一样决定该调用。名称包含 `__` 的调用会绕过注册表：智能体会将其路由到所属服务器，并对结果应用 `default` 输出限制。MCP 调用没有内置权限提示；服务器是否可以启动由工作区信任决定，而按服务器和按工具的禁用开关会限制正在运行的服务器可以做什么（[MCP 服务器](../features/mcp.md#权限模型)）。

这类调用会像每个内置工具一样被读作一个动作，而不是其注册表 id：对话记录行和控制台框会显示 *正在调用 `browser_navigate`（MCP 服务器 `playwright`）*（俄语：*запускаю browser_navigate на MCP-сервере playwright*），实时状态行将其目标用在 `playwright/browser_navigate` 上，因为通用动词无法说明该调用的作用。其他智能体使用的 `mcp__server__tool` 拼写也以相同方式读取。除标签外，该行还会指明调用作用于什么，来自其 `path`、`url` 或 `name` 参数，与内置工具一样；如果服务器以其他方式命名参数，则回退到第一个可读作标签的参数，即单行内容，而不是正文（[Web 界面](../surfaces/web-ui.md#工具调用卡片内置-spa当前)）。

## 工具调用前运行的内容

操作员钩子会在权限门控以及权限模式之前看到每次调用，包括 MCP 工具（[钩子](../features/hooks.md#钩子如何作答)）。应答 `permissionDecision: deny` 的 `PreToolUse` 钩子会阻止该调用；它永远不会执行，模型会读取 `blocked by hook: <reason>` 作为工具结果。`allow` 会跳过工具原本会引发的提示，`ask` 即使在 `accept_edits` 或 `bypass` 下也会强制提示，而 `updatedInput` 会在调用运行前替换参数对象。多个匹配的钩子都会运行，且限制性最强的决定胜出。

一个 `toolsets[]` 预设会通过 `tools` 和 `disallowed_tools` 缩小会话可用的工具范围；模型会命名其预设，会话可以使用 `/toolset` 选择另一个预设（[工具集](../getting-started/configuration.md#工具集)），这适用于不应携带它从不调用的工具 schema 的小窗口本地模型。上面的名称就是这些列表采用的名称，MCP 工具会以 `server__tool` 或 `server__*` 匹配。对于产生该请求时未提供的工具，其调用会在模式自身检查之后、任何钩子或权限提示之前被拒绝，结果为 `error: tool "<name>" is not offered: this session runs with the toolset "<name>" ...`。

子智能体的工具集是从父级缩小而来的（[子智能体](../features/subagents.md#能力如何收窄)）：定义中的 `tools` 和 `disallowed_tools` 只会移除名称，而 `question`、五个配置编辑工具和 `plan_exit` 会从每个子进程中排除，因此子进程既不能询问用户，也不能重写配置，也不能离开 plan 模式。结果在到达模型之前会受 `tools.output_limits` 限制，对内置工具和 MCP 工具都是如此。

<!-- docsgen:source sha256=3c5fc6a2b4883503 -->

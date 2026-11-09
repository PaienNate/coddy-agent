# 调度器

## 概览

调度器是一个可选的、类似 cron 的运行器。它从两个固定位置读取扁平的 `*.md` 任务文件（YAML frontmatter 加 Markdown 正文）：你自己的任务位于 **`${CODDY_HOME}/scheduler`**，仓库携带的项目任务位于 **`<workspace>/.coddy/scheduler`**，并在每个受信任任务到期时启动它。一次运行是一个**后台智能体任务**：与 `spawn_agent` 调用启动的任务相同（[子智能体](../features/subagents.md)），是后台任务池中类型为 `agent` 的任务（[后台任务](../features/background-tasks.md)），由子会话支撑，二者都挂在一个按任务划分的会话之下。该任务会话就是任务的运行历史，Web 界面的运行面板显示它的方式，与聊天任务面板显示聊天任务的方式相同。调度器仅在 **`scheduler`** 构建标签下编译进来。

组成部分：

- **`external/scheduler`**：**`Serve`** 入口 **`coddy serve`** 在 **`scheduler.enable`** 下运行；控制台和 **`coddy acp`** 用 **`--scheduler`** 启动同一个守护进程。
- **`external/scheduler/daemon`**：UTC 分钟 tick 和**运行时**：什么会启动一次运行、停止它、知道任务是否正在运行、应用保留策略并保留任务会话。
- **`external/scheduler/storage`**：两个根目录中的扁平任务发现、项目任务的包含规则、YAML frontmatter、UTC cron、**`.state`** sidecar、项目任务的批准（**`scheduler-trust.json`**）以及守护进程扫描的工作区（**`scheduler-workspaces.json`**）。
- **`external/scheduler/service`**（**`schedservice`**）：共享 CRUD、运行记录行、HTTP 和工具负载（不与 **`internal/tools`** 形成循环）。
- **`external/scheduler/tools`**：**`schedtools`** 注册 **`coddy_scheduler_*`** 工具（每个工具一个 `*.go` 文件，位于 **`tools/`** 下），当 **`scheduler.enable`** 为 true 时。
- **`internal/agent/scheduled_run.go`**：**`RunScheduledJob`**，即运行本身，运行在 **`spawn_agent`** 使用的子运行执行器上。

cron 解析器使用 **五个字段**（**minute hour day month weekday**）并采用 **UTC**。触发点按 **UTC 分钟边界**（秒 **0**、纳秒 **0**）求值，类似 **crond**：守护进程每个 UTC 分钟唤醒一次，读取任务文件，仅当该分钟匹配表达式且 **`.state`** 检查点严格早于该分钟时才启动任务。因此 **`* * * * *`** **每个 UTC 分钟最多运行一次**。诸如 **`*/2 * * * *`** 的步长字段使用与 vixie cron 相同的分钟网格（UTC 分钟 **0,2,4,…**）；**`*/3 * * * *`** 使用 UTC **0,3,6,…**。

任务文件中 **`schedule`** 字段的更改会在下一次扫描时读取（无需重启）。

## 构建

- 仅调度器：`go build -tags=scheduler ./cmd/coddy`
- HTTP 和调度器：`go build -tags=http,scheduler ./cmd/coddy`（为内嵌 SPA 添加 `,ui`，使用 `http`）

## 启用

当配置中的 **`scheduler.enable: true`** 启用时，或当你传入 **`coddy serve --scheduler`**、**`coddy acp --scheduler`** 或 **`coddy --scheduler`**（控制台）时，调度器守护进程和工具处于活动状态。守护进程需要构成这些运行的会话管理器，因此它会在该管理器存在后启动；单纯的集群中继永远不会运行它。

**`/coddy/scheduler`** 下的 REST 路由需要 **`-tags=http,scheduler`**；参见 **`docs/reference/http-api.md`**。

## 任务所在位置

![调度器抽屉，显示三个任务，其中一个已暂停](../../assets/screenshot-fullhd-scheduler.png)

*调度器抽屉：三个任务及其下次运行时间，一个已暂停，一个显示上次运行结果，每行都有“运行记录”和“立即运行”*

![任务编辑器：cron 提示、模式、模型、子智能体和权限模式，以及 Markdown 正文](../../assets/screenshot-fullhd-scheduler-job.png)

*任务编辑器：cron 提示、模式、模型、一次运行的子智能体和权限模式、Markdown 正文，以及底部的“运行记录”、“暂停”和“删除”*

任务是**直接**位于两个文件夹之一的 **`*.md`** 文件；发现任务时不使用嵌套子目录。两个文件夹都不可配置。

- **用户任务**位于 **`${CODDY_HOME}/scheduler`**。它们属于你，并且始终受信任。
- **项目任务**位于 **`<workspace>/.coddy/scheduler`**，并随仓库一起传播，因此团队可以把计划中的工作放在代码旁边。它们只有在受信任后才会运行（见下一节）。

旧的 **`scheduler.dir`** 键不再被读取。`coddy -t` 会将其报告为已移动的键，下一次启动会把该文件夹中的任务（连同它们的 **`.state`** sidecar）复制到 **`${CODDY_HOME}/scheduler`**，并从 `config.yaml` 中移除该键，同时把旧文件保留为 `config.yaml.bak-<time>`。如果主目录中已有字节完全相同的任务，则跳过它（当主目录没有检查点时，会携带其检查点）；如果已有字节不同的任务，则复制为 **`<id>-migrated.md`**（若该名称已被占用，则复制为 **`<id>-migrated-2.md`** 及后续名称），因此不会有任务悄悄停止。旧文件夹保持原样。

任务 id 在两个作用域中唯一：项目任务不能使用用户任务的 id，用户任务也不能使用调度器已知工作区中项目任务的 id（创建和重命名时都会返回 **409**）。如果项目文件仍以冲突的 id 出现，则会被列为 **冲突**，并且在重命名之前不会运行。

每个任务都有一个 sidecar：用户任务的 **`basename.state`** 位于 **`basename.md`** 旁边，项目任务的 **`${CODDY_HOME}/scheduler/.projects/<key>/<job_id>.state`**（键由工作区派生），因此调度器写入的任何内容都不会落到检出目录中。它是一个小型 JSON 记录，包含 cron 检查点（**`last_scheduled_utc`**），以及任务运行过一次后的**任务会话** id（**`session_id`**）。两个写入方都会保留这两个字段：写入检查点不会丢失会话指针，第一次运行也不会丢失检查点。通过 API 或编辑器重命名任务时，sidecar 会随文件移动，因此运行历史会跟随重命名。没有指针的 sidecar（被手动删除，或由旧版 Coddy 写入）不会使历史成为孤儿：守护进程会在会话根目录中按它命名的任务 id 找到任务会话，并再次记录该指针。

守护进程**在运行开始之前**写入检查点，使用触发它的 **UTC 分钟起点**，因此下一个 UTC 分钟 tick 不会在运行仍在进行时再次触发同一分钟。检查点以**原子方式**写入（同目录中的临时文件加重命名）。如果还没有检查点（或旧版构建留下了 1980 年之前的过期时间戳），第一次运行会依据挂钟时间和五字段表达式按 **vixie 风格**计时，而不是从 Unix 纪元开始补跑积压。对于给定的 Coddy 主目录，只应让**一个**长期运行的进程启用调度器：两个守护进程读取相同任务可能会重复触发。

可选的 YAML frontmatter **`paused: true`** 会跳过 cron tick 和 **`POST …/run`**，直到恢复。

## 项目任务与信任

项目任务是仓库内容，它会按定时器以你的权限运行智能体，因此它受到与项目 MCP 服务器、子智能体定义和钩子相同的信任门控。**`scheduler.project_trust`** 设置策略：

- **`ask`**（默认）：项目任务会被列出，但只有你为该工作区批准它之后才会运行（按 cron 或手动）。批准是 **`${CODDY_HOME}/scheduler-trust.json`** 中的一条回执，绑定到工作区、任务 id 和文件 sha256，因此之后发生变化的任务（一次 `git pull`、一次手动编辑）会再次等待批准。链接的 git worktree 共享其主检出的批准（[Git worktree](../features/worktrees.md#worktree-与其主检出共享的内容)）；
- **`allow`**：项目任务像你自己的任务一样运行；
- **`deny`**：项目任务会被列出，但永远不会运行。

**你通过 Coddy 创建**的项目任务（Web 界面、从会话发起的 REST API，或 `coddy_scheduler_job_create` 配合 `scope: "project"`）会针对写入的字节立即获得批准：是你创建了它。同一个文件通过 git 到达另一个人时，不会为他们获得批准。通过 Coddy 进行的编辑只有在编辑前任务已受信任时，才会让任务保持受信任；暂停或编辑随检出而来的任务不会批准它。没有工具会批准项目任务：批准是人的决定，在 Web 界面或通过 REST 进行。

![调度器抽屉：一个全局任务，以及一个随仓库而来、等待批准的项目任务](../../assets/scheduler/project-jobs-needs-approval-dark-1280.png)

*调度器抽屉：一个全局任务，以及一个随仓库而来、等待批准的项目任务*

![项目任务的批准区块：它来自哪里、为什么它不运行，以及批准绑定到的文件](../../assets/scheduler/project-job-approve-dark-1280.png)

*项目任务的批准区块：它来自哪里、为什么它不运行，以及盾牌批准的确切文件*

抽屉会先在 **全局** 下列出用户任务，再在 **此项目** 下列出聊天所在工作区的项目任务，然后列出调度器运行的每个其他工作区的项目任务。不会运行的行会说明原因（**需要批准**、**已拒绝**、**id 冲突**、**无效**）；等待批准的任务在“运行”的位置显示一个盾牌，点击会用其原始文件打开任务。那里的盾牌会批准所显示的确切字节（如果摘要不再匹配文件，服务器会拒绝），而在已批准的任务上则会撤回批准。新任务默认是 **全局**，除非你选择 **此项目**，这需要该项目中有一个聊天。

每个受信任的任务，无论是用户任务还是项目任务，都在进程的唯一守护进程及其唯一任务池中、在唯一的 **`scheduler.max_queue`** 和 **`scheduler.timeout`** 下运行。守护进程会扫描：

1. 用户任务；
2. 其自身工作目录中的项目任务；
3. 每个已获批准工作区的项目任务；
4. 会话向调度器询问过的每个工作区的项目任务（列在 **`${CODDY_HOME}/scheduler-workspaces.json`** 中），这就是任务在 **`allow`** 下运行的方式，在那里任何东西都不会被批准。

其中每一项都按其项目计数：链接的 git worktree 算作其主检出的同一文件夹。因此项目任务会从主检出按 cron 运行一次，而 worktree 中的副本会列为未调度。

信任是在对文件的一次读取中决定的，发生在写入 cron 检查点之前，而运行也由同一次读取构成，因此中途被替换的文件绝不会在它未获得的批准下运行。守护进程会按内容记录一次它跳过了某个任务及其原因。

项目任务的包含规则：**`.coddy`**、**`.coddy/scheduler`** 和任务文件必须是真实文件夹和文件，而不是链接（链接会被列为 **无效**，且永远不会被写入）；如果某个工作区的项目文件夹会位于 **`${CODDY_HOME}`** 内部（Coddy 在 `$HOME` 下以默认主目录启动），则该工作区没有项目任务；项目任务的 **`cwd`** 必须为空（其工作区）、是一个保持在工作区内部的相对路径，或是一个在解析链接后指向该工作区或其中文件夹的绝对路径（在项目任务出现之前编写的任务经常带有这样的路径）。

## 任务文件格式

Frontmatter 字段：

- **`description`**（string）：简短的人类可读摘要
- **`schedule`**（string）：五字段 crontab，UTC
- **`cwd`**（string，可选）：对于用户任务，空表示 Coddy 进程的 cwd，相对路径在运行时相对于它解析；对于项目任务，空表示其工作区，并且只接受工作区内部的路径，相对或绝对均可
- **`model`**（string，可选）：本次运行的会话模型覆盖
- **`mode`**（string，可选）：**`agent`**、**`plan`** 或 **`ask`**（默认 **`agent`**）
- **`agent`**（string，可选）：子智能体定义名称（**`subagents.dirs`**，见[子智能体](../features/subagents.md)）：运行会在其角色、工具允许列表、模型、推理级别和权限收窄下进行。定义的 `reasoning` 会在运行使用的模型（任务的 `model`，否则定义中的，否则配置的）提供该级别时生效，如同一次 spawn；一个它不提供的级别会被丢弃，并在智能体日志中给出警告，而 `reasoning: default` 保留模型自身的级别。项目作用域的定义需要像任何 spawn 一样具有信任回执；命名未批准定义的任务不会启动，手动运行会说明这一点。定义的 `isolation` 不适用：任务在其 `cwd` 中运行。空值会以该模式的完整工具集运行一个通用智能体。
- **`permission_mode`**（string，可选）：**`ask`**、**`accept_edits`** 或 **`bypass`**：运行在无需询问的情况下可以做什么。空值为 **`bypass`**，这是无人值守默认值，也是调度器一直以来的行为；在 **`agent`** 中命名的定义只能收窄它。计划运行中没有人回答提示，因此在 **`ask`** 或 **`accept_edits`** 下，受门控的调用会被**拒绝**，绝不会等待；拒绝会告诉运行没有人可以被询问，因此它会报告自己无法做什么，而不是声称操作员拒绝了（[子智能体](../features/subagents.md)，分离运行）。
- **`paused`**（bool，可选）：为 true 时任务不执行

正文：用作该调度器运行的一次性用户指令的 Markdown。

运行的系统提示词会说明这是一个无人值守启动的计划任务（带有任务 id 和 cron 槽位或 "手动"），没有人会回答问题，其最终消息会保留为运行的记录；当任务命名定义时，还会包含该定义的角色。技能、规则和环境块会像在任何会话中一样存在，并且会通过工作区信任门控为任务的 cwd 拨号配置的 MCP 服务器，除非任务以 ask 模式运行（该模式不提供 MCP 工具），或其定义不允许其中任何服务器的工具。若定义命名了某个服务器的一个工具（`srv__lookup`）或一个可达该服务器的模式，则允许该服务器。

## 运行

每个任务拥有一个普通会话，即**任务会话**，在第一次运行时创建并记录在 sidecar 中。它本身从不运行轮次：针对它的提示词会被拒绝（通过 HTTP 返回 **409**，并指出任务），它会排除在历史记录和所有默认列表之外（**`GET /coddy/sessions?include_scheduler=true`** 会列出它），它存在的目的是让任务的运行有一个父级。一次运行是该会话的子项，存储在 `spawn_agent` 子项存储的位置：

```
<sessions root>/<job session>/
  session.json                 schedulerRun: true, schedulerJobId: <job_id>
  background/<task_id>/        the run's task record and progress log
  subagents/<run session>/     the run's transcript (subagentRun, schedulerJobId, schedulerTrigger)
```

运行的任务行携带 **`kind: "agent"`**、标签 **`<job_id> · cron 2026-09-18 10:00 UTC`**（或 **`· manual …`**）以及 **`agent: {"name": <job_id or its definition>, "session_id": <run session>}`**。其进度日志就是任务面板为任何子智能体运行显示的内容（`→ tool`、`✓ tool`、`[assistant]` 行，结尾的 `=== subagent report ===` 块）；其对话记录是运行会话，只读，并带有一条通知，指出任务并链接回任务的运行记录。状态来自任务池：**`running`**、**`succeeded`**、**`failed`**（轮次以错误结束）、**`timed_out`**（达到 **`scheduler.timeout`**）、**`stopped`**（已取消）、**`orphaned`**（进程在运行进行中死亡）。

- **每个任务一次只运行一个。** 守护进程会在任何东西启动之前预留任务，因此 cron tick 和手动运行不能同时启动它；在运行进行中到来的到期槽位会被跳过并记录日志。
- **`scheduler.max_queue`** 限制所有任务中同时进行的运行数量。超过上限的 tick 会跳过该槽位的任务并给出警告；超过上限的手动运行会返回 **409**。
- **`scheduler.timeout`** 是运行的硬性限制；任务池会像任何任务一样将其上限设为 **`tools.background.max_timeout_seconds`**，当调度器限制高于该上限时，守护进程会在启动时警告。
- **提供商连接中断不会结束运行。** 没有人盯着计划运行，因此像子智能体一样，它会连续承受最多五次对其提供商的失败调用，而聊天轮次在两次后就会放弃：在默认 **`agent.llm_retry_base_ms`** 下约为六分钟，当 **`scheduler.timeout`** 留出空间时。它会把每次重连写入进度日志（**`↻ provider failed (...); reconnecting in 20s (attempt 2 of 5)`**，[子智能体](../features/subagents.md#提供商连接断开时)）。
- **取消**会停止运行的任务；运行会被记录为 **`stopped`**。
- **保留。** **`scheduler.retain_sessions`**（默认 **5**）为每个任务保留那么多个**已完成**运行，按开始时间取最新；当一次运行完成时，较旧的运行会丢失其任务记录和对话记录。运行面板中的 **清除**（**`DELETE /coddy/scheduler/jobs/{job_id}/runs`**）会丢弃所有已完成运行。删除任务会删除其任务会话及其下的每次运行。
- **运行之间不会在内存中保留任何东西。** 已完成运行的会话会退役，其池条目会释放；当面板请求时，记录会从 bundle 中恢复。运行留下的工作（一个转入后台的 `run_command`、它生成的子智能体）会在运行的轮次返回时停止，如同任何子项一样。
- **重启。** 进程死亡时正在进行的运行会显示为 **`orphaned`**；检查点会阻止其槽位再次触发。

计划运行位于 spawn 深度 **0**：在 **`subagents.max_depth`** 内，它可以生成自己的子智能体，这些子智能体会像任何 spawn 一样计入 **`subagents.max_concurrent`**。它永远不会被已完成任务唤醒（对于它启动的一切，`notify_on_finish` 都关闭）。

运行是调度器的工作，而不是对话：它会排除在历史记录之外，并且在运行时会计入导航栏的 **调度器** 项，而绝不会计入 **历史记录**。无论调度器是否打开，计数都会跟随每次运行的开始和结束。

在此设计之前发布的调度器运行（顶层 **`sched_`** bundle）仍会从历史记录中隐藏，并且不会列为运行；不需要时请手动删除它们。

守护进程日志保持简短（**`slog`**：**`scheduler_run_spawn`** 和 **`scheduler_run_finish`**，包含任务、运行会话、任务和状态）；完整跟踪位于运行会话中。

## Web 界面中的运行

任务按作用域分组（见“项目任务与信任”）。每个任务行都会显示其上次运行（任务面板的状态点在名称行上，运行结束后，其状态和时钟会显示在描述下方，布局类似历史记录的一行；正在运行的任务只显示其脉动点和“停止”按钮）以及一个 **运行记录** 控件，用于打开任务的运行记录：任务会话的后台任务面板，停靠在任务编辑器停靠的位置（编辑器页脚有相同控件）。路由：**`#/scheduler/jobs/<job_id>/runs`**；命名某次运行的链接（**`.../runs/<task_id>`**）会打开面板并展开该运行的卡片，然后丢弃该 id。每次运行都是一张卡片：第一行是状态点和任务，第二行是运行开始的时间、运行了多久，以及完成后结束的时间（按你的时区），右侧是模型、其推理级别和消耗的 token。运行如何以及何时启动（手动或按 cron，槽位以 UTC 表示）以及完整的开始和结束时间是卡片的悬停文本。进行中的运行会从任务行停止，其按钮在运行时为“停止”，因此其卡片没有停止按钮；已完成的运行位于 **已完成** 计数器下；点击会在原处打开卡片，显示运行的进度日志、运行时长以及 **显示对话记录**，后者会打开运行会话。可以同时打开多张卡片。当有运行进行中时，面板每 2.5 秒轮询一次 **`GET /coddy/sessions/{job session}/background-tasks`**，否则每 15 秒轮询一次，与聊天面板完全一样。

![任务的运行面板：一次运行进行中，一次已完成](../../assets/scheduler/runs-panel-running-dark-1280.png)

*任务的运行面板：一次运行进行中，一次已完成*

![打开的已完成运行：其进度日志和报告](../../assets/scheduler/runs-detail-finished-dark-1280.png)

*打开的已完成运行：其进度日志和报告*

![运行的对话记录，只读，并带有返回任务运行记录的链接](../../assets/scheduler/run-transcript-readonly-dark-1280.png)

*运行的对话记录，只读，并带有返回任务运行记录的链接*

## HTTP API

启用 **`-tags=http,scheduler`** 时，**`GET /coddy/scheduler/jobs`**（行携带 **`scope`**、**`workspace`**、**`trust`**、**`running`**、**`session_id`** 和 **`last_run`**；没有会话头或 **`cwd`** 时，它只列出用户任务）、任务 CRUD（**`scope=project`** 寻址项目任务，在 **`X-Coddy-Session-ID`** 中会话的工作区里创建）、用于项目任务的 **`trust`** / **`untrust`**、**`pause`** / **`resume`**、**`run`**（**202** 以及 **`task_id`**、**`session_id`** 和 **`run_session_id`**）、**`cancel`**、**`GET …/runs`** 和 **`DELETE …/runs`** 镜像 **`schedservice`** 层；运行面板本身轮询会话的后台任务路由。**`503`**，如果 **`scheduler.enable`** 为 false。只有链接 **scheduler** 时，OpenAPI 才会合并这些路径（比较 **`external/httpserver/scheduler_http.go`** 与 **`scheduler_http_stub.go`**）。

## 工具（启用调度器时）

- 每个按任务划分的工具都接受 **`scope`**：**`user`**（默认）或 **`project`**，即会话工作区的项目任务
- **`coddy_scheduler_jobs_list`**：列出任务（**`include_body`** 可选）：用户任务、会话工作区的项目任务以及调度器运行的工作区的项目任务，每个都带有其 **`trust`**
- **`coddy_scheduler_job_get`**：一个任务 JSON，包含 **`body`**
- **`coddy_scheduler_job_create`** / **`coddy_scheduler_job_replace`** / **`coddy_scheduler_job_patch`**（可选的 **`new_job_id`** 会重命名任务文件；**`agent`** 和 **`permission_mode`** 是与其他字段一样的字段）
- **`coddy_scheduler_job_delete`**
- **`coddy_scheduler_job_pause`** / **`coddy_scheduler_job_resume`**
- **`coddy_scheduler_job_run`**：手动运行（不会推进 cron **`.state`**）；返回任务和运行会话
- **`coddy_scheduler_job_cancel`**
- **`coddy_scheduler_job_runs`**：一个任务的运行记录行

旧名称 **`coddy_scheduler_list`**、**`read`**、**`write`**、**`delete`**、**`validate`** 已移除。

## 示例任务（分钟 tick）

```md
---
description: "Minute tick"
schedule: "* * * * *"
cwd: ""
mode: agent
---

In the session working directory run

bash -lc 'date -u +%FT%TZ > tick.txt'
```

一个在只读定义下、具有收窄权限模式的任务：

```md
---
description: "Nightly audit"
schedule: "0 3 * * *"
mode: agent
agent: reviewer
permission_mode: accept_edits
---

Review yesterday's commits and write the findings to docs/audit/<date>.md.
```

另请参见 **`docs/reference/http-api.md`**（调度器表）、**`docs/getting-started/configuration.md`**（**`scheduler`** 键）、**`docs/plans/scheduler-runs.md`**（设计记录）以及 **`examples/README.md`**（Python 框架 **`http_e2e_scheduler_api`**、**`http_e2e_scheduler_agent`**、**`acp_e2e_scheduler_agent`**）。

<!-- docsgen:source sha256=b76999f77e29d026 -->

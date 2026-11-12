# ReAct 智能体：设计规范

## 在 Coddy 中的角色

Coddy 被建模为**框架加执行引擎**。本文档规定了该引擎 -

- `internal/agent` 中的 **ReAct 循环**，它将提示词和工具转化为流式传输的轮次，
- 默认的**编码智能体**行为 - 工具注册表、`agent`、`plan` 和 `ask` 模式、权限门禁。

同一框架可以使用更窄的工具界面或不同的客户端（自动化，而不仅是 IDE）。

下面的 ReAct 流程和消息约定对于任何使用 ACP 的会话都保持稳定。

## 什么是 ReAct？

ReAct（推理 + 行动）是一种智能体范式，其中 LLM 在以下环节之间交替：
- **思考** - 关于下一步做什么的内部推理
- **行动** - 调用工具或产生输出
- **观察** - 接收行动的结果

参考：https://arxiv.org/abs/2210.03629

## ReAct 循环实现

### 系统提示词结构

模板为 **`internal/prompts/agent.md`**、**`plan.md`** 和 **`ask.md`**（默认内嵌，或通过 **`prompts.dir`** 覆盖）。它们使用 Go 的 **`text/template`**。

![Coddy 发给模型的请求，自上而下包括：工具定义、系统消息的各个块、历史与轮次上下文，每部分都标出了会使其重新渲染的因素](../../assets/diagrams/system-prompt.png)
*请求的每个块及其重新渲染事件：当这些事件都不发生时，系统消息在会话的每个请求中都是相同的字节，只有历史之后的轮次上下文会在每一步变化。*

渲染顺序大致与 markdown 文件对应如下：

```
You are Coddy, an AI coding agent ...   identity line, see "Agent identity" below
Working directory: {{.CWD}}
{{.SubagentRole}}                       ## Your role as a subagent, in a child session only
## Mode: Agent | Plan | Ask             capabilities, how to work, how to plan
                                        (plan mode lists {{.DiscardedPlans}} here)
{{.Subagents}}                          the subagent catalog, when the session may spawn
## Available tools
{{.Tools}}                              MCP tools among them, as server__tool
{{.Skills}}                             ## Slash commands: the skill catalog and the load_skill hint;
                                        a /name body rides in the user message, load_skill returns one
{{.PlanContext}}                        agent mode: the plan an approved turn implements
{{.Rules}}                              AGENTS.md and DESIGN.md of the agent home, then of the
                                        workspace, then the rules that always apply
## Project instructions
{{.Instructions}}                       the files of instructions.files
## Session memory
{{.Memory}}                             the session's notes

<environment_context>                   appended outside the template
<os>...</os>
<arch>...</arch>
<shell>...</shell>
</environment_context>
## Hook context                         appended: SessionStart and this turn's UserPromptSubmit
...                                     appended: the surface block of the web UI or a messenger
```

环境块追加在可配置模板之外，因此操作系统和 shell 的事实不会被自定义提示词意外遗漏。

### 系统提示词在一轮内冻结

系统消息**每轮渲染一次**，由 **`buildSystemPromptParts`**（**`internal/agent/system_prompt.go`**）完成，随后 ReAct 循环的每一步都会逐字节地发回同一条消息。步骤之间没有任何东西会重写 **`messages[0]`**，除非 **`prompts.dir`** 下的模板会打印时钟或清单本身（见下一节末尾）。如果一轮因权限提示而停止，当回答恢复该轮时，它会再次渲染（**`ResumeAfterPermission`**）。

原因是提供商的提示词缓存。提供商按**前缀**缓存请求 - 先是工具定义，然后是从开头开始的消息 - 与上一个请求不同的第一个字节会丢弃其后缓存的一切。系统消息位于整个对话之前，因此包含秒的实时时钟，或工具刚刚改写过的清单行，过去会让五万 token 的历史在每次请求时都看起来是新的。阿里巴巴的网关按 1024 token 的块进行缓存；第一个块内的差异会使 `cached_tokens` 保持为零，无论后面跟着多少相同的历史。

### 轮次上下文块

一轮运行期间确实会变化的内容，会放在重放历史之后，作为一条由 **`buildTurnContext`**（**`internal/agent/turn_context.go`**）构建的尾随 `user` 消息：

```
<turn_context>
Runtime state refreshed by Coddy for this step. It is not a message from the user; do not answer it, just take it into account.

## Current UTC time
2026-09-15T11:39:03Z

## Current todo checklist
- [ ] ...
</turn_context>
```

- **实时时钟**，始终存在 - 在本轮提示词渲染时打一次时间戳，并由该轮每一步复用，因为通道会重新发出一个没有产生任何内容的步骤，而该重放必须与失败的请求逐字节一致（[循环步骤](#循环步骤)下的*空助手重新发出*）；
- **待办清单**（来自 **`internal/tools/todo.FormatPlanMarkdown`** 对 **`session.Plan`** 的 markdown），当会话有时 - 因此本轮中的 `coddy_todo_*` 调用会在下一步立即反映出来；
- 此会话中**仍在运行的后台任务**，每行一条，使用 **`background_list`** 的措辞（`backgroundTasksSection`，[background-tasks.md](../features/background-tasks.md)），这样模型无需调用就能知道它留下了什么在运行。已完成任务和系统任务会被排除，当 `tools.background.enable` 为 false 时不写入该章节。

该块从不持久化：它在 **`provider.Stream`** 发送边界处追加，紧邻结果驱逐投影，循环持续追加的工作消息切片从不会看到它。因此请求中只有最后几百个 token 未被缓存；它们背后的对话是缓存命中。

**工具调用激活的规则** - 一个 glob 规则，其模式匹配该调用读取或写入的文件，以及它进入的文件夹中嵌套的 `AGENTS.md` 和 `DESIGN.md` - 不在该块中。它附在**该调用的结果**中（**`toolCallRules`**，**`internal/agent/rules_activation.go`**），保存在对话记录行自身的 **`Rules`** 字段中，因此界面显示的输出和结果驱逐所替换的输出保持为工具自身的输出，并且只在同一发送边界处与内容合并（**`withToolRules`**）。它与结果一起写入一次，之后每个请求都会逐字节重放；如果放在块里，它会在每一步再次发送，并会被下一轮的系统提示词纳入。模型已经可以用相同文本读到的规则 - 在更早的结果或消息中仍会发送 - 不会再次附加；被压缩折叠掉的规则，或其文件此后已更改的规则，会在下一次匹配调用中回来，而对规则文档本身的调用不会作为规则返回它。系统提示词的规则块和指令块本身按会话的规则代次渲染一次（**`standingParts`**，**`session.RulesPrompt`**），因此在会话中途编辑 **`AGENTS.md`** 也不会改变 `messages[0]`；压缩会开始下一代。参见[规则与提示词缓存](../features/rules.md#规则与提示词缓存)。

**`UTCNow`** 和 **`TodoList`** 仍可供 **`prompts.dir`** 下的模板使用，该模板仍可渲染它们 - 代价是每个请求都会失去该缓存。这样的模板在历史之后不会获得时钟和清单；它的块携带没有模板会打印的唯一章节，即正在运行的后台任务，并在没有此类任务时省略。

### 另一半：结果驱逐

冻结系统消息只是稳定前缀的一半。读取、grep 以及 `glob`、`print_tree`、`websearch` 和 `webfetch` 的列表结果驱逐（**`internal/agent/result_eviction.go`**）会把占位符写入重放历史的**中间**，这同样会从该点起使缓存失效。使用默认滑动工作窗口时，这过去几乎在每一步都会发生，因此即使提示词已冻结，对话每次都会被重新处理。**`compaction.result_eviction.start_percent`**（默认 50）与 **`start_tokens`**（默认 60000）会推迟投影，直到对话达到模型 **`max_context_tokens`** 的该比例与此 token 数中较小者；低于该值时，历史完全按提供商已有的样子发出。门槛只统计对话，从不计入系统提示词、工具定义或规则。该决定基于**未修剪**的消息进行衡量，因此修剪不会把估计值推回标记以下，从而使投影在两种形态之间来回切换。提供商因过大而拒绝的请求会在该轮剩余时间内打开门禁（**`Agent.evictionForced`**）：一旦提供商拒绝读取前缀，门禁所保护的缓存就无关紧要了。一轮内的折叠（**`compact_in_turn.go`**）也是同样的权衡：它只在压缩阈值处重写历史，而它写入的摘要行以提示词开头，因此重写后的前缀仍以模型被要求的内容开头。参见 [compaction.md](../features/compaction.md)。

### 读取缓存命中

提供商将请求的缓存份额报告为 **`usage.prompt_tokens_details.cached_tokens`**（OpenAI 兼容）或 **`cache_read_input_tokens`**（Anthropic）。Coddy 将其作为 **`llm.Response.CachedInputTokens`** 携带，它是 **`llm.Response.InputTokens`** 的一部分，按 OpenAI 的 **`prompt_tokens`** 方式计算整个提示词：Anthropic 的 **`input_tokens`**、**`cache_creation_input_tokens`** 和 **`cache_read_input_tokens`** 会相加。它按调用记录两者：

```
export CODDY_LOG_LEVEL=debug   # or logger.levels: {agent: debug}
# msg="llm call usage" input_tokens=6044 cached_input_tokens=5120 output_tokens=12
```

长会话的第二步显示 **`cached_input_tokens`** 接近零，意味着有东西正在重写前缀。大多数 OpenAI 兼容服务器完全省略该字段，然后它会读为零，但并不意味着未命中。

### 智能体身份

**Coddy 发送的每个系统提示词都会在开头点明产品名。** 该句子位于 **`internal/prompts/identity.go`** 中，形式为 **`prompts.Identity`**（**`"You are Coddy, an AI coding agent."`**），而 **`prompts.WithIdentity`** 负责把它放在那里。

为什么存在：LLM 网关无法通过线路协议区分一个 OpenAI 兼容客户端和另一个，因此网关通过将**系统提示词的开头**与已知产品表进行匹配来归属流量（**`"You are Claude Code…"`**，**`"You are Cline…"`**）。Coddy 过去以通用句子开头，因此在那类分析中不可见。将 **`prompts.Identity`** 视为已发布的契约：网关以 **`you are coddy`** 子串为键，改写它会在网关跟上之前悄悄将 Coddy 从其报告中移除。

应用位置：

- **`buildSystemPrompt`**（`internal/agent/system_prompt.go`），上下文分解之前的最后一步，覆盖智能体、计划和询问模式、用户自己的 **`prompts.dir`** 模板以及渲染回退；
- **`buildCompactionRequest`**（`internal/agent/compact.go`），摘要器是独立请求，有自己的系统提示词；
- 辅助 HTTP 提示词：聊天标题生成（`external/httpserver/coddy_coddy.go`）和提示词增强（`external/httpserver/enhance_prompt.go`）；记忆子智能体的模板（`external/memory/prompts/memory_agent.md`）由 `buildSystemPromptParts` 通过其子会话的 `PromptTemplate` 渲染，因此它会自行以身份行开头。

测试锁定的两个属性（`internal/prompts/identity_test.go`，`internal/agent/identity_prompt_test.go`）：

1. 标记落在**前 220 个字符**内，因为这是网关检查的前缀，很长的 **`{{.CWD}}`** 不能把它挤出去；
2. 该行**恰好出现一次**。内置模板已经以 **`You are Coddy, …`** 开头，因此 **`WithIdentity`** 会检测到标记并原样返回它们，而不是在顶部叠加第二条身份行。

自定义 **`prompts.dir`** 模板**不需要**点明 Coddy：该行会自动为它前置。

### 通过函数调用 API 进行工具调用

现代 LLM 支持原生函数/工具调用。智能体使用此方式而不是基于文本的 ReAct 提示：

1. 工具被定义为 JSON Schema 对象并传递给 LLM API
2. LLM 返回结构化的工具调用请求（不是原始文本）
3. 智能体执行请求的工具
4. 结果作为 `tool` 角色消息追加到对话中
5. LLM 在上下文中结合工具结果继续推理

这种方法比文本解析更可靠，并受到所有主要提供商支持（OpenAI、Anthropic、Ollama 及兼容模型）。

### 对话消息结构

```
messages: [
  { role: "system",    content: <system_prompt> },
  { role: "user",      content: <user_prompt> },
  { role: "assistant", content: "", tool_calls: [{ id: "call_1", name: "read", args: {...} }] },
  { role: "tool",      tool_call_id: "call_1", content: <file_contents> },
  { role: "assistant", content: "", tool_calls: [{ id: "call_2", name: "write", args: {...} }] },
  { role: "tool",      tool_call_id: "call_2", content: "OK" },
  { role: "assistant", content: <final_answer> }
]
```

### 循环步骤

```
1. BUILD_MESSAGES
   - Load applicable skills and project rules for current context (separate prompt sections)
   - Build system prompt once for the turn (`buildSystemPromptParts`: template + TemplateData; the built-in templates print neither the clock nor the todo checklist, the turn context block carries them)
   - Skill bodies are already in the history: on Run entry, before the user turn is persisted, Run appends the body of each skill the typed text invokes as `/name` to that message as a `<coddy_attachment kind="skill">` element (`invokedSkillBlocks`), so later requests replay the same bytes and the prefix cache holds. Surfaces leave the element out of the transcript (`mention.ForDisplay`, `stripCoddyAttachments.ts`); queued follow-ups get their skill bodies the same way.
   - Prepend system to session history (user turn already persisted on Run entry)

2. LLM_CALL
   - Send messages + tool definitions to LLM provider
   - Receive response: may contain text + tool_calls

3. STREAM_RESPONSE
   - For each text chunk: send session/update(agent_message_chunk)
   - For each tool_call: send session/update(tool_call, status=pending)

4. EXECUTE_TOOLS (if any tool calls)
   - For each tool_call sequentially inside one assistant message:
     a. Send session/update(tool_call_update, status=in_progress)
     b. If requires permission: session/request_permission -> wait for response
     c. Execute tool (built-in or MCP)
     d. Send session/update(tool_call_update, status=completed|failed, content=result)
     e. Append tool result to conversation history

5. REFRESH_TURN_CONTEXT
   - messages[0] is not rewritten: the system message stays as the turn rendered it, and what moved since (wall clock, todo checklist, running background tasks) is rebuilt by `buildTurnContext` and appended after the history at the send boundary
   - A template under `prompts.dir` that prints `{{.UTCNow}}` or `{{.TodoList}}` is the exception: it is re-rendered before every call, and its block carries no clock or checklist
   - An automatic compaction or a workspace switch re-renders the system message for the new history or workspace, and a turn resumed after a permission prompt renders it again on entry; no other step of the turn does
   - Next loop iteration repeats from step 2

6. CHECK_COMPLETION
   - If no tool calls in last response -> DONE (stopReason: end_turn)
   - If turn_count >= max_turns -> DONE (stopReason: max_turns;
     the turn's stop notice names the key that set the limit). max_turns defaults
     to 165; an explicit 0 disables the step limit
   - Otherwise -> back to step 2

   Loop guard (**`agent.loop_guard`**, default on) can end the turn earlier:
   - A streamed response repeating the same passage **`loop_stream_repeat_cycles`** times
     in a row (answer text or reasoning) has its stream cancelled. The repeated run is
     stripped from the stored message, so it is never replayed to the model.
   - A tool call repeated **`loop_tool_repeat_limit`** times with identical canonical
     arguments is not executed; the model gets a result explaining why.
   - Either case first nudges the model to change course, up to **`loop_nudge_max`**
     times, then -> DONE (stopReason: agent_refused) with a notice.

   Recovery attempts and transport retries draw from the **`llm_retry_max`**
   shared budget (default 3). The initial model request for a step is not
   charged; each extra attempt or recovery, the provider recovery excepted,
   consumes one slot; the budget resets
   when the model makes progress (a tool call or an answered followup) — it is
   not a lifetime cap on all LLM calls in a turn. Per-strategy limits apply
   independently of the budget:

   - **Transport retries.** The resilient wrapper repeats on HTTP 429, 408,
     500, 502, 503, 504 and 529 (overloaded), on an overload the provider
     reports inside the stream (an error object whose decoded `type` or `code`
     says so: `service_unavailable_error`, `overloaded_error`,
     `server_is_overloaded`, or a numeric 503 or 529; the words of its message
     decide nothing), and on transport failures that left no output with the
     caller; each retry consumes one slot. The Codex, Anthropic and
     OpenAI-compatible streams read the same fields, and an overload that
     names no number counts as a 503. Any other in-band error is read by its
     numeric `code` (a number or a numeric string) on each of those streams:
     it is retried when the code is one of those statuses and is final
     otherwise, whatever status or connection phrase its message holds. A
     status is also read from the text of
     an untyped error, as a number with no digit in front of it (`529 `, not
     the end of `1529 tokens`). A cancellation, an unknown host, and any
     failure after output was emitted stay final. The Responses API's own
     failure events, `response.failed` (its error object rides on the
     response) and the flat `error` event (its `code` is a field of the
     event), are read by the same fields, so an overload there counts as a
     503 as well; any other failure they report is an untyped error, named by
     its `code` in front of its message and read by that text as above.
   - **First-token re-issue.** A streamed call the first-token guard
     (**`llm_first_token_timeout_ms`**) cuts with **nothing produced** is
     re-issued at most once, using a normal **`max_turns`** iteration. The
     re-issue is skipped after output was emitted or the caller cancelled.
     Consumes one slot.
   - **Empty-assistant re-issue.** A step that returns neither answer text nor
     a tool call is replayed once as the identical request; the empty assistant
     turn is removed from the LLM-facing message slice (the transcript keeps
     it, with signed thinking preserved). A reply stopped at `max_tokens` is
     not replayed: the same request would hit the same cap, and the
     output-limit recovery below handles it. Consumes one slot.
   - **Wording nudges.** If the re-issue also comes back empty, up to
     two nudges (**`maxEmptyAssistantContinuations`**) are sent, each consuming
     one slot and a normal **`max_turns`** iteration. The plain replay uses
     a normal iteration too; transport retries stay inside that iteration.
     The nudge projection also removes the empty assistant message from the
     LLM-facing history. If an auto-compaction rebuilds that history between
     attempts, the pending recovery projection and the nudges that still stand
     at its end are restored; a provider recovery rebuilds nothing, it runs the
     request that failed again. Signed reasoning stays in the transcript. After the budget or nudge limit is exhausted the turn
     ends with **`StopReasonRefused`**.
   - **Output-limit recovery.** A step that stops at `max_tokens`
     (`finish_reason: "length"`) does not end the turn and is never replayed,
     because the same request would hit the same cap. It is answered with a
     message appended at the end of the history, local to the request and
     never persisted, chosen by what the step left behind (output_limit.go):
     - Nothing to act on (no visible text, no tool call - only thinking, or
       nothing): **`outputLimitNudge`** - keep the reasoning short, split a
       large file write or edit into several smaller tool calls. The empty step
       stays in the transcript and is left out of every request of the turn.
     - Visible text that stops partway: **`outputLimitContinueNudge`** -
       continue exactly where it stopped without repeating, keep the step
       short. The text stays in the transcript and in the history, because the
       user read it. The continuation is the next assistant message, and the
       cut one is stored marked `truncated` (`llm.Message.Truncated`), both
       exactly as streamed, the whitespace at the join included
       (`session.StripAssistantContent`); a truncated text takes no placement
       markers of shared files, and the message that completes the answer
       closes with them.
       No event marks the seam: a live surface appends the chunks to the
       message it is drawing. These readers of the transcript join the two
       with nothing between, passing over a step that came back empty between
       them (`session.FollowsTruncated`): every history built from the
       transcript - the next turn's, and a rebuild within the turn after a
       compaction (`foldSplitAnswers` in `buildMessages`); a reloaded
       transcript in the
       web UI (`transcriptFromMessages.ts`), which keeps them apart only where
       the stream drew them apart - the continuation began with thinking of its
       own, or a notice stands between them; and the answer a turn hands over - a subagent's report,
       the reply of a non-streaming request, the Stop hook's
       `last_assistant_message` (`session.ContinuedAnswer`, without the
       placement markers). A paged read of the history never starts or ends a
       page between them (`session.PageMessages`). A replay to the console, an
       ACP client or a `--remote` console and an export show the parts as the
       messages they are stored as. Telegram and Pachca read
       nothing back: they stream the whole answer into one message. The goal
       supervisor's digest of a turn (`goal_digest.go`) does not join them: it
       reads each assistant message on its own, so the parts before the last
       are steps of the turn and the last part alone is its closing claim.
       Assistant messages that follow each other without that mark (the text
       before a cut-off tool call and the step after it, an answer a provider
       failure cut off and the step run again after it) are joined in the
       history with a blank line, so a later turn never sends two assistant
       messages in a row, which chat templates that insist on alternation
       refuse; a step with signed reasoning keeps its message. An answer of its
       own a provider failure cut off is not marked `truncated`: the stream
       ended wherever the failure hit it, and the step run again starts the
       answer over as often as it goes on from the last character. A part of a
       continued answer a provider failure cut off is marked: the step run
       again is asked to go on from it, so it stays one text with the rest.
       The requests of the turn itself are not touched: the local message
       stands between the two there.
     - A last tool call the output limit cut off (**`cutOffToolCall`**): its
       arguments are there and are not a whole JSON document. Empty arguments
       are a call that takes none and runs. **`cutOffToolCallNudge`** names the tool (a
       plain identifier only), says the call was cut off and not run, and asks
       for several smaller calls. Only the last call can be cut; an earlier
       call that does not parse is a model error and runs, and fails, as
       before. The cut call never runs - no permission prompt, no hooks - and
       is left out of the stored assistant message, so the tool-call pairing
       stays whole and no provider is sent half a JSON document. Its pending
       row closes as `failed` and its record in the tool-call store says why.
       The calls before it were written whole and run in order; their results
       come first, then the message.

     The history only grows at its end, so the cached prefix is untouched. The
     loop keeps one ordered list of the messages that stand at the end of the
     working history (**`standing`**): every recovery appends the message it
     sent, in the words it was sent in, and a step that leaves text or a tool
     call in the history clears it, since what came before was answered. A
     history rebuilt from the transcript between the attempts (an
     auto-compaction, or the compaction after a refused request) puts that list
     back (**`restoreRecoveries`**) and leaves the empty steps out again. At most
     **`maxOutputLimitRecoveries`** (2) cut-off steps in a row. Each consumes
     one slot and a normal **`max_turns`** iteration (on the last iteration the
     turn ends on the output limit, not on the step limit), so
     `llm_retry_max: 0` turns it off. A follow-up and a step that is not cut
     off reset the count; a cut-off step that also ran whole calls does not,
     though those calls restart the allowance. Subagents and scheduled runs
     take the same path. When the retries are spent the turn ends with
     **`StopReasonMaxTokens`** and a notice (**`outputLimitNotice`**).
   - **Provider recovery.** A call that failed because of the provider's lane
     (**`llm.IsTransientProviderError`**: an answer with HTTP 408, 500, 502,
     503, 504 or 529; an in-band error, on each of those three streams, that
     is an overload by the fields above or that carries one of those
     statuses as its numeric `code` - an in-band error of another kind, such
     as an `api_error` that names no status, stays final; a cut or
     silent stream - an event cut inside its JSON included, a stream frame that
     is not JSON at all not - output already emitted or not; a 429 is left to
     the wrapper and the limit wait)
     does not end the turn. The answer text and
     reasoning already streamed are kept as an assistant message without tool
     calls, a **`notice`** row goes to the UI log, and after a pause
     (**`providerRecoveryDelay`**: 5x then 20x **`llm_retry_base_ms`**, or a
     longer **`Retry-After`**, capped at 2 minutes) the step runs again on a
     fresh allowance as the request that failed, byte for byte, so the
     provider's cached prefix holds (issue #520): the working history is not
     rebuilt from the transcript, which would bring back the steps that came
     back empty and drop the messages the recoveries sent. When text was kept,
     the kept answer and an LLM-facing nudge to continue are appended to it.
     The nudge stands at the end of the history like the messages of the other
     recoveries (and like the loop guard's nudge), so a compaction before the
     step runs again puts it back after the kept answer instead of ending the
     request on that answer. At most **`maxProviderRecoveries`** (2) in a row; a successful call resets
     the count. Uses a normal **`max_turns`** iteration. It takes no slot of
     the shared budget: its own count bounds it, and `llm_retry_max: 0` turns
     it off.
   - **Overflow recovery.** A call the provider refused as larger than the
     model's window (**`llm.IsContextOverflow`**) with nothing streamed yet
     compacts the turn and runs the step again, once per step
     (**`maxOverflowRecoveries`**; a successful call resets it): earlier turns
     when a regular split fits the limit the refusal revealed, otherwise the
     steps of the turn itself, behind the prompt (**`internal/agent/compact_in_turn.go`**).
     It forces result eviction past its **`start_percent`** and **`start_tokens`** gate for the rest
     of the turn (**`Agent.evictionForced`**), writes a **`notice`** row, and
     uses a normal **`max_turns`** iteration. A second refusal of the same
     step, a fold that fails or has nothing to fold, and
     **`compaction.auto_enable: false`** end the turn with
     **`contextOverflowError`**.

   An explicit **`llm_retry_max: 0`** disables all of the above except the
   overflow recovery, which is a compaction and not a retry of the same
   request. The
   `loop_guard`, Stop hooks, fallback models, and `wait_for_limit_reset` are
   independent policies, each governed by their own settings.

   At `logger.level: debug`, each LLM call completion logs
   `msg="llm call finished"` with `provider_attempts` (total inner adapter
   calls including transport-layer retries), `transport_retries`, `call_reason`
   (one of `step`, `empty_reissue`, `empty_nudge`, `output_limit_nudge`,
   `first_token_retry`, `provider_recovery`, `loop_guard`, `quota_reset_wait`,
   `stop_hook`, `queued_followup`, `context_overflow_recovery`), and
   `retries_remaining` (budget slots left after this call). Completion and usage
   records also carry `session` and a process-monotonic `call_id`. With the agent
   debug component enabled, `request_latency.go` emits correlated local, HTTP
   and dispatch timings without request content. See
   [A long session becomes slower](../getting-started/troubleshooting.md#a-long-session-becomes-slower)
   for the stage boundaries and the bounded synthetic comparison.

   Two guards bound a streamed call that stops answering. The first-token guard
   (**`llm_first_token_timeout_ms`**, 90 s) cuts a call that produced nothing;
   the stream idle guard (**`llm_stream_idle_timeout_ms`**, five minutes, in
   **`internal/llm/transport.go`**) cuts a response whose server sent nothing
   for that long after its first bytes. The text the user already watched
   stream in is persisted like a truncation, and the turn ends with the stall
   named. Neither guard applies to a **`stream: false`** model, whose answer
   arrives in one piece: **`providers[].timeout_ms`** is the only bound on
   such a call, next to the HTTP/2 liveness pings that close a connection
   whose far side stopped answering.

7. FINAL_RESPONSE
   - Send session/prompt response with stopReason
```

## 模式特定行为

### 智能体模式

内嵌的 **`agent.md`** 描述智能体行为（质量、shell、待办、git worktree）。待办相关指令引用 **`Tools`** 中呈现的 **`coddy_todo_plan_*`** 和 **`coddy_todo_item_*`** 工具。**Git worktrees** 块将 **`.coddy/worktrees/<branch>`** 指定为 worktree 的放置位置，与 HTTP 界面的工作区切换使用同一目录（[会话](../features/sessions.md#git-worktree)）；没有它，模型会自行选择位置并在仓库根目录留下一个未跟踪文件夹。

代表性内置工具（不包括带 MCP 命名空间的工具）：

- `read`、`keep_result`、`write`、`edit`、`apply_patch`、`glob`、`grep`、`print_tree`
- `websearch`、`webfetch`、`http_request`
- `run_command` 和后台工具（`background_list`、`background_output`、`background_wait`、`background_stop`、`background_reap`）
- 文件系统变更：**`mkdir`**、**`rm`**、**`rmdir`**、**`touch`**、**`mv`**（部分可能需要权限路径）
- 会话清单：**`coddy_todo_plan_read`**、**`coddy_todo_plan_replace`**、**`coddy_todo_plan_archive`**、**`coddy_todo_item_add`**、**`coddy_todo_item_remove`**、**`coddy_todo_item_update`**、**`coddy_todo_item_move`**
- 所有 MCP 服务器工具（名称为 **`serverName__toolName`**）

### 计划模式

内嵌的 **`plan.md`** 使默认的 **registry** 界面面向读取（对外提供的集合中没有内置写入或 **coddy** 待办工具）。**`run_command`** 以及来自已配置服务器的所有 **MCP** 工具仍可用于检查。

暴露给 LLM 的代表性内置工具（registry 允许列表）：

- `read`、`keep_result`、`glob`、`grep`、`print_tree`
- `websearch`、`webfetch`
- `run_command`，带 `background_list`、`background_output`、`background_wait`、`background_stop`
- `plan_write`、`plan_list`、`plan_read`

完整列表位于 **`internal/agent.ToolSetForMode("plan")`**（[运行模式](../features/modes.md#plan)）。

外加 MCP 工具（**`serverName__toolName`**）。允许列表也在执行时强制执行，因此从历史重放的调用如果指定了列表之外的工具，会被拒绝并给出计划模式通知，而不是被执行。当准备交付实现工作时，提示词会指示将客户端切换到 **`agent`** 模式。

### 询问模式

内嵌的 **`ask.md`** 描述一个只读助手：它从仓库和网络回答，从不改变任何东西。registry 允许列表（**`internal/agent.ToolSetForMode("ask")`**）为 **`read`**、**`keep_result`**、**`glob`**、**`grep`**、**`print_tree`**、**`websearch`**、**`webfetch`**、**`question`**、**`load_skill`**、**`coddy_docs_search`** 和 **`coddy_docs_read`**；没有 shell，没有计划、待办或配置工具，没有 **`spawn_agent`**，并且**从不**追加 **MCP** 工具。与计划模式一样，允许列表也在执行时强制执行，因此从历史重放的调用会被拒绝并给出只读通知；ask 在此处拒绝 MCP 工具名称，而 plan 会让它们通过到 MCP 层。计划提及或 **`runPlanSlug`** 元数据在询问模式下永远不会启动计划运行。子智能体子级永远不会在询问模式下运行，除非其父级的轮次已经处于询问模式，而该模式无法生成子智能体；在会话只读时生成的记忆子级会构建为仅回忆，其六个工具中的三个只读工具为（**`memory.ToolNames(true)`**，[memory.md](../features/memory.md)）。

## 内置工具规范

内置工具、它们的参数、每个工具所需的权限以及提供它的模式列在[工具](../reference/tools.md)中。提供给模型的 schema 是 **`internal/tools`** 和可选包下每个工具构造函数的 **`InputSchema`**；**`internal/tools.NewRegistryFor`** 从中构建注册表。

## 计划更新格式

客户端接收 `session/update` 通知，其 **`sessionUpdate`** 字段等于 **`plan`**，携带结构化的 **`entries`**。当会话持久化开启时，待办列表工具还会将活动清单以 **`todos/active.md`** 持久化到捆绑包中（对应 **`FormatPlanMarkdown`** / **`ParsePlanMarkdown`**）。

示例载荷：

```json
{
  "sessionUpdate": "plan",
  "entries": [
    { "content": "Read current auth module", "priority": "high", "status": "pending" },
    { "content": "Analyze JWT requirements", "priority": "high", "status": "pending" },
    { "content": "Write new auth implementation", "priority": "medium", "status": "pending" },
    { "content": "Update tests", "priority": "low", "status": "pending" }
  ]
}
```

计划条目会随智能体推进而更新：
```json
{ "content": "Read current auth module", "priority": "high", "status": "completed" }
```

## ReAct 循环中的错误处理

- LLM API 错误：从 `agent.llm_retry_max` 每步共享预算中取用（默认 3；显式 0 禁用）；传输重试、首 token 重新发出、无答案恢复和输出上限恢复共享同一池；预算在工具取得进展时重置，并且不是每轮所有 LLM 调用的生命周期上限
- LLM 流停滞（在最初的字节之后 `llm_stream_idle_timeout_ms` 内没有字节）：持久化部分答案，以指明停滞的方式使轮次失败；仅当没有交付任何内容时，才在剩余共享预算内重试
- 工具执行错误：将错误作为观察返回，让 LLM 决定下一步
- 权限被拒绝：返回 "permission denied" 观察
- 工具超时：在配置的超时后返回 "timeout" 观察
- 上下文过长：总结较早消息，并使用摘要继续
- 已取消：中止所有操作，返回 `cancelled` 停止原因

<!-- docsgen:source sha256=663d0f5f2ccb61e6 -->

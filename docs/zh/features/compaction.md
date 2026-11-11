# 上下文压缩

模型读取的上下文窗口是有限的，而一个持续工作的会话会超出它：每一次交互、每一条工具结果和每一个文件分页都留在对话记录里。压缩让长会话保持在窗口之内，同时不丢失记录。较早的历史被折叠进一段生成的摘要，模型读取这段摘要来替代原始的行，而最近的轮次保持原样，另外还有一个更廉价的机制，把模型已经越过的文件分页和搜索结果折叠起来。这两者都是在请求发往模型时构建的投影：磁盘上的对话记录保留每一条原始消息和每一份完整结果。

![一次压缩如何运行：什么触发它、CompactSession 的各个步骤、切点落在哪里、摘要器读取什么、摘要器链及其各个阶段，以及之后发生什么](../../assets/diagrams/compaction-flow.png)
*每一次触发都以 `CompactSession` 结束：它在一个边界处切分，让 `PreCompact` 钩子有机会否决，把边界之前的历史分阶段地通过摘要器链折叠，并在那里插入一行摘要。行的链及其合并见 [摘要构成一条链](#摘要构成一条链)。*

## /compact 命令

```text
/compact [-m|--model <id>] [-r|--reasoning <level>] [instructions]
```

内置的 `/compact` 在每个提示词界面上都以 `/export` 和 `/plugin` 的方式运行 - 包括控制台、ACP 编辑器、Web 界面输入框和 `POST /v1/responses`：它在文本成为消息之前就被识别，不占用主模型的一个轮次。只有当 `compaction.enable` 为真时，它才出现在命令目录里（`GET /coddy/commands`、ACP 的 `available_commands_update`）。命令之后的任何内容都会作为附加指令交给摘要器，因此 `/compact focus on the file paths and the failing test` 会引导摘要保留什么。Ask 模式不会限制它：该命令是操作者动作，位于只读边界之外。

`--model <id>`（或 `--model=<id>`）为那一次压缩指定摘要器，无论 `compaction.model` 怎么说：`/compact --model hub/qwen3-coder keep the file paths`。该 id 可以是任意大小写的 `models[].model`、不带提供商前缀的模型名（`hub/qwen3-coder` 用 `qwen3-coder`，即使旁边还有一个更长的 `hub/qwen3-coder-next`），或者是某个 id 的一部分，只要它恰好匹配一个已配置的模型 - 当只有一个模型的 id 中含有 `qwen` 时，`--model qwen` 就够了。被指定的模型会排在摘要器链的头部，而已配置的链保持在它后面，因此某个提供商宕机时仍会按 [历史大于一次摘要请求](#历史大于一次摘要请求) 所述回退。匹配不到任何模型、或匹配到多个模型的名字不会压缩任何东西，而是回复已配置的 id。`--reasoning <level>`（或 `--reasoning=<level>`）设置撰写摘要时使用的推理级别：可以是摘要器模型提供的级别 - 若 `--model` 指定了模型则用该模型，否则用 `compaction.model`，再否则用会话的 - 或者用 `default` 表示模型自身的级别。链中不提供该级别的回退模型按自己的级别运行。`-m` 和 `-r` 是短写法：`/compact -m qwen -r high keep the file paths`。选项在前：第一个不是选项的词开始指令，此后的指令可以随意提及 `--model`；只有 `--` 词、`-m` 和 `-r` 是选项，因此写成列表的指令（`- keep the paths`）仍是指令。在 Web 界面中，输入框会补全选项及其值 - 模型，以及 `--model` 所指定模型的级别 - 参见 [Web 界面页面](../surfaces/web-ui.md#输入框命令选项)。

手动压缩是强制的。它会折叠现存的一切：当配置的保留轮次数目导致没有可摘要的内容时，它用更少的保留轮次重试，直到为零，因此即使是很短的对话也能压缩。命令文本会作为用户行持久化，以便对话记录显示它，而回复只有一行：

```text
Context compacted: 14 message(s) summarized, 6 kept verbatim. Summarizer: hub/qwen3-coder.
Nothing was compacted: unknown model "qwn" (configured: openai/gpt-4o, hub/qwen3-coder). Usage: /compact [-m|--model <id>] [-r|--reasoning <level>] [instructions]. ...
Nothing to compact: there is no earlier conversation to summarize yet, or nothing new since the last compaction.
Compaction is disabled in the configuration (compaction.enable: false).
```

通过 HTTP 时，同样的动作是 `POST /coddy/sessions/{id}/compact`，可带一个可选请求体 `{"instructions": "...", "model": "...", "reasoning": "..."}`（`model` 对应命令的 `--model`，`reasoning` 对应其 `--reasoning`）；它返回摘要、消息计数以及撰写它的 `model`；压缩被禁用时返回 `400`；`model` 未指定单一的已配置模型，或 `reasoning` 是摘要器不提供的级别时返回对应错误；`409` 表示某个轮次正占用会话、会话正在被删除，或这是只读的子会话（[HTTP API](../reference/http-api.md)）。它作为一个轮次运行，因此 `GET /coddy/events` 会宣告其开始和结束，而正在查看该会话的浏览器标签页会重新加载发生变化的内容。

## 自动压缩

智能体会把上下文用量与会话模型的窗口进行比较。在某次模型调用报告了 `input_tokens` 之后，下一次检查会使用该计数加上对自那次调用以来新增内容的估算。当提供商未报告计数时，估算会包括系统提示词、工具定义与 schema、规则、技能、MCP、消息文本、工具参数、推理和图片。与旧的 runes/4 规则相比，该估算为代码和非 ASCII 文本预留了更多空间。达到 `compaction.threshold_percent`（默认 80）时，较早的历史会在下一次模型调用之前被压缩：在一个轮次的第一次调用之前、在权限答复之后、在轮次之间工具结果使上下文增长时，以及在首次调用新选定的、窗口可能更小的模型之前（[会话设置](session-settings.md#发送命令时会发生什么)）。把 `compaction.auto_enable: false` 设为 false 可保留手动压缩而禁用该触发器。任何失败 - 摘要器错误、钩子否决 - 都会被记录，轮次继续未经压缩地运行。

### 上下文窗口

每个读取方都以相同方式解析窗口 - 包括触发器、控制台上下文百分比背后的 `usage_update`，以及 Web 界面绘制上下文环所依据的 `GET /v1/models` 的 `max_context_tokens` - 因此环显示的就是触发器测量的：

1. 模型条目的 `max_context_tokens`；
2. 提供商的模型列表为该模型报告的窗口：`limit.context`（NeuralDeep 中枢）、`context_length`（OpenRouter）、`max_model_len`（vLLM）、`max_context_length`（LM Studio）、`max_input_tokens`（Anthropic）、`max_context_tokens`（另一个 Coddy 的 `GET /v1/models`）或 `context_window`（Codex 目录，其中它是 Codex 自身所使用的窗口，如今它所服务的每个模型都是 272000；它旁边更大的 `max_context_window` 只是 Codex 允许其自身 `model_context_window` 设置将其提升到的上限，Coddy 不读取它）。该列表会为 `neuraldeep`、`devin`、`codex` 和 `anthropic` 提供商读取，以及为带有显式 `api_base` 的 `openai` 提供商读取，从不用于 api.openai.com，因为它的列表不携带窗口。它在一个轮次开始时、在提供模型列表时以及会话切换到该模型时读取，一个轮次对从未应答过的列表最多等待三秒 - 轮次中途切换模型后的那一步在测量前也以同样方式等待 - 并且其可信时间为一小时；读取失败后会在五分钟后重试，或在设置中登录或登出后于下一次读取时重试；
3. 128000。

当提供商未报告窗口，或报告的窗口大于部署实际提供的窗口（以更小上下文启动的本地服务器）时，设置 `max_context_tokens`。

### 少数几个长轮次

当窗口容得下 `keep_recent_turns` 个用户轮次时，自动压缩会原样保留这么多轮次。当容不下时 - 例如由少数几个很长的智能体轮次组成的会话，上下文在没有很多提示词的情况下就超出了窗口 - 它会保留更少，最少保留到正在被回答的那个提示词，而它绝不会折叠该提示词。这个尾部的规模在 token 上同样有上限：`keep_tail_tokens`（默认 15000）保留能容得下的最新轮次，因此宽窗口上的压缩落在约 15k，而不是落在它本应回收的那份窗口份额上。

当窗口中只有该提示词时，就没有更早的轮次可折叠，而一个由单个提示词后跟几十个工具步骤组成的轮次 - 其中许多是并行的 `glob`、`grep` 或 `read` 调用，每个都返回数千字节 - 可能单凭自己就超出一个小窗口。此时触发器会折叠 **该轮次较早的步骤**。一个步骤是一条助手消息及其之后的工具结果，因此十个并行调用算一个步骤，一批调用绝不会在其调用与结果之间被切开。切点之前的步骤会像任何其他压缩一样交给同一个摘要器 - 同样的模型链、对大于一次请求的历史使用同样的阶段、同样的 `PreCompact` 和 `PostCompact` 钩子（它们看到的触发原因是 `auto`） - 而最新的步骤保持原样。

正在被回答的提示词不能进入摘要：一个不知道被问了什么的模型什么也答不出来。因此，替代这些步骤的摘要行以该提示词开头，逐字保留，然后是一行说明其上方文本是什么，然后是摘要：

```text
<the prompt, verbatim>

The text above is the message this turn started with, verbatim. Coddy compacted the conversation before this point, including the earlier steps of the work on it. Summary of the compacted part:

<the summary>
```

这一行不说明该消息是谁写的，因为并非每个轮次都以你输入的提示词开头：[会话监督者](session-supervisor.md)用它自己的指令开始其目标轮次，而一个完成的后台任务会用一条通知唤醒一个轮次。它们各自在其开头几个词中就表明自己是什么，因此这一行对这三种情况可以保持一致。

当存在更早压缩的摘要行时，该行跟在它们之后，并开启模型对该轮次所读取的内容，因此之后每个请求都会携带该提示词。如果提示词本身是一段粘贴的日志，它会在中间被截断到窗口的十分之一（至少 300 token），并附带一个标记说明丢掉了多少字符；对话记录保留完整文本，附加在提示词上的图片留在原始消息上，不会复制到该行。当该轮次再次被折叠时，会在第一行之后新增一行，只摘要它之后的步骤；提示词留在第一行的开头，因此窗口中只保存一份它。如果一个轮次被折叠多次，以致其行超出了它们在窗口中的份额，这些行会被合并，合并后的行以该提示词开头来替代它们（[摘要构成一条链](#摘要构成一条链)）。

在轮次运行期间你排队的消息是该轮次的后续消息，而不是它自己的轮次，Coddy 会在对话记录中标记它（`messages.json` 中的 `follow_up`）；由 [Stop 钩子](hooks.md#事件) 提交的后续消息也算一个。该标记用于区分正在被回答的提示词与其后的后续消息，即使某条后续消息逐字重复了提示词，而且它在 Coddy 重启后仍然保留。因此行首的文本始终是该轮次开始时的那条消息。落入被折叠步骤中的后续消息会与它们一起被摘要，而处于最新步骤中的后续消息则作为普通消息留在原处。轮次的步骤从提示词开始计数，自动压缩绝不会在后续消息处切分，因此它无法把提示词折叠进普通摘要。在标记存在之前写入的对话记录没有标记，其最后一条用户消息被视为提示词。

保留多少步骤由预算决定，上限为四个步骤。折叠的目标取两者中较小者：触发压缩的窗口份额的一半（`threshold_percent: 80` 时为窗口的 40%），以及每个请求除对话之外所携带的内容（系统提示词、工具定义、规则）加上 `keep_tail_tokens`。在宽窗口上起作用的是后者：1M 窗口配 120k 规则时目标是 135k，而不是 419k，这正是真正回收窗口与窗口依旧那么大之间的区别。随后目标再减去每个请求除对话之外所携带的内容，减去窗口中已有的摘要行（[摘要构成一条链](#摘要构成一条链)），再减去行首的提示词。能容得下的最新步骤会保留，最多到上限，且最新的步骤始终保留。若没有这样的目标，一次保留了四个大步骤的折叠会使请求仍高于阈值，并在紧接着的下一步再次折叠。摘要本身不计入预算：目标留出的那一半用于支付它。只有一个步骤的轮次没有更早的内容可折叠，该轮次会记录一次并继续。折叠以及下文所述的恢复没有自己的开关：它们是自动压缩的一部分，`compaction.auto_enable: false` 会随之一并关闭它们。

当每个请求除对话之外所携带的内容（系统提示词、工具定义、规则）或最新步骤对窗口来说太大时，一次折叠可能使请求仍高于阈值，而下一步又会再次折叠：每一步都产生一次摘要器调用和一次摘要的摘要。因此每次自动折叠之后，Coddy 会估算下一个请求，当它仍然达到或高于阈值时，该轮次不再进行自动折叠；日志会带数字记录一次。轮次剩余部分交给下文所述的对被拒请求的恢复。触发器使用提供商对上次请求的自身计数来测量上下文，因此该估算以及上述折叠的目标都以同样方式缩放：按提供商为该请求所计数到的每个 Coddy 估算 token 的实际 token 数缩放。否则，计数比估算更密的提供商会看到一次折叠保留了实际已超过真实阈值的步骤。对于一个停止折叠的轮次，补救办法是更大的窗口、更高的 `threshold_percent`，或更少的工具和规则。

#### 当提供商拒绝请求时

阈值是相对于 Coddy 所知的窗口来测量的。服务器实际提供的可能小于该窗口（以更小上下文启动的本地服务器）、计数比 Coddy 的估算更密，或收到一个放不进剩余空间的步骤。当提供商因为请求放不进模型窗口而拒绝它时，轮次并不会就此结束：Coddy 会压缩该轮次并再次请求同一步骤，一次。

- 它通过 `context_length_exceeded` 代码来识别该拒绝，无论应答状态是什么，也通过 OpenAI 兼容服务器、Anthropic、llama.cpp、vLLM 和 Gemini 的措辞来识别，措辞仅在 400、404、413 或 422 应答中识别，或在不带任何状态的错误中识别（流中的一帧、Codex 失败事件）：提及 token 的速率限制（429）或故障（5xx）不算，而 "too many tokens" 只在有状态表明请求被拒绝的地方才算。OpenAI 客户端从其错误中丢弃的响应体 - vLLM 的扁平 `{"object":"error",...}`、纯文本、Codex 的 `{"detail": ...}` - 会从响应中读取。
- 已经向客户端流式传输了内容的请求绝不会再次请求，无论因何被拒。
- 折叠的目标是在该拒绝所揭示的限制之下，以 Coddy 自己的单位计：窗口被下调到提供商指定的限制，按提供商计数比 Coddy 估算多出的比例缩放，且绝不高于被拒请求的估算值。目标仍是该限制下阈值份额的同一半。下限是完全没有步骤，而不是一个步骤：请求已超过限制，因此如果其他都放不下，连最新步骤也会被去掉。唯一的例外是输出限制截断、模型被要求继续的应答：其各部分及其后的空步骤保持原样，因为模型从文本的最后一个词继续，而轮次交付的应答会与各部分合并。当轮次中除了该应答别无他物时，不折叠任何内容，拒绝结束该轮次。
- 当窗口中有更早的轮次，而进行中的轮次从其提示词开始能放进剩余空间时，Coddy 按常规压缩的方式折叠更早的轮次，提示词保持为普通消息。否则它按上述方式在轮次内部折叠。
- [结果驱逐](#结果驱逐) 会立即为该轮次剩余部分启动，无论 `start_percent` 和 `start_tokens` 说什么：该门槛保护提供商的提示词缓存，而拒绝过请求的提供商对于它不会读取的内容的缓存毫无用处。下一轮次开始时门槛恢复。
- 重试对每个步骤只有一次。成功的调用会重新开始计数，因此两个不同的步骤可以各自恢复，而同一步骤再次被拒则结束该轮次。会话日志会收到一条通知，说明请求被拒并已再次请求，智能体日志记录大小（`estimatedTokens`、`providerTokens`、`providerLimit`、`limitTokens`）。压缩期间的 Stop 会以停止方式结束轮次；关闭或截止时间会以拒绝的解释和 `(the compaction after the refusal was interrupted: ...)` 结束它。

当没有任何可折叠的内容、钩子否决了压缩、摘要器失败、更小的请求也被拒，或 `compaction.auto_enable` 为 false 时，轮次会以一个说明此事的错误结束，替代提供商裸的应答：

```text
context window exceeded: the provider refused the request because it does not fit the model's context window (Coddy estimated about 38000 tokens; the window is 49152 tokens; the provider counted 51402 tokens against a limit of 49152); run /compact, start a new session or split the task into smaller steps: provider "local" (http://127.0.0.1:8080/v1): openai stream: ...
```

该消息给出 Coddy 自己对请求的估算及其测量所依据的窗口，在提供商的应答包含大小信息时补充这些大小，并以提供商的原始应答结尾。在一次没有帮助的压缩之后，它还会显示 `Coddy compacted this turn and asked again, but the provider refused the smaller request too`。当提供商的限制低于 Coddy 所工作的窗口，或窗口从未设置而默认为 128000 时，它还会告诉你把 `models[].max_context_tokens` 设置为服务器实际提供的窗口，以便下一次压缩更早开始。提供商的状��保留在错误上，因此 HTTP API 会以和之前相同的 400 或 404 答复客户端。`/compact` 会折叠历史，包括正在被回答的提示词。[结果驱逐](#结果驱逐) 会在对话达到 `start_percent` 与 `start_tokens` 门槛后，把最旧的目录列表和网页结果从请求中去掉，因此一连串很长的 `glob` 和 `print_tree` 调用能更久地适应小窗口（issue #490）。

## 模型可以请求压缩

阈值是在调用之前做出的猜测；模型才是知道自己刚刚读到了什么的那一方。它粘贴的构建日志、它不再需要的文件、返回远超预期的搜索 - 模型看到上下文在填满，可以用 `compact_context` 工具自行折叠历史，而不必等触发器或等操作者输入 `/compact`：

```json
{"instructions": "keep the file paths and the failing test", "model": "hub/qwen3-coder"}
```

`instructions` 是可选的，完全像 `/compact` 之后的文本一样引导摘要。`model` 也是可选的，对应命令的 `--model`：工具描述会列出已配置的模型，因此提示词里的 "compact the context with qwen" 会变成一次指定该模型的调用，而匹配不到任何模型的名字会作为工具的错误连同已配置的 id 返回，供模型纠正。携带该调用的助手消息本身绝不会被折叠，因此它的结果总是回应一个提供商能看到的调用，即使在模型在会话的第一个轮次就进行压缩。该调用表现得像手动压缩 - 它折叠现存的一切 - 而工具返回与命令相同的那一行，因此模型能读回它做了什么，并在缩短后的历史上继续：循环在下一次调用之前会从折叠后的对话记录重建请求。当折叠把正在被回答的提示词一并带走时 - 在会话的第一个轮次，没有其他可折叠的内容 - 提示词会和其余内容一样进入摘要：该工具是模型自己的选择，而让长轮次的提示词保持原样的是轮次内部的自动折叠。该工具在智能体模式和计划模式中提供，当 `compaction.enable` 为 false 时完全隐藏。Ask 模式不会获得它，因为那里提供的每个工具都是只读的；操作者的 `/compact` 和自动触发器在该模式下仍然有效。

## 历史大于一次摘要请求

在此之前，头部会先去重：会话中已经逐字出现过的行会被丢弃，只在原处保留第一份副本，并且每个条目说明有多少次重复被去掉。会话通过重复自己来填满窗口 - 每次尝试后都粘贴同样的构建日志、同一个文件被读取十几次 - 而摘要器从第二份副本中什么也学不到，却要为此付出全额（issue #273）。短行保持原样：一个右花括号和一个裸数字是结构，丢掉它们会破坏摘要必须读取的代码。

以前折叠是一次调用：把对话的整个头部放进一次对摘要器的请求中。当会话接近它所依据的窗口时这还行，而恰恰在压缩最重要的时候它就不行了。一个远超其窗口的会话 - 一个不断读取大文件的模型、一个因为窗口未知而从未触发的自动触发器 - 到达 `/compact` 时，历史是摘要器自身窗口的数倍，提供商就会拒绝该请求：

```text
compaction LLM call: 400 Bad Request: this model's maximum context length is exceeded
```

这会让会话卡住：太大而无法发送，而唯一能缩小它的调用又发不出去。

摘要器拒绝也不是终点。`compaction.fallback_models` 会按顺序列出尝试过的模型，在前者失败时使用下一个，而会话自身的模型始终在链中，且只出现一次：在最后，除非 `compaction.model` 或该列表已经给了它更早的位置 - 因此指向已宕机、过载或消失的部署的 `compaction.model` 不再会让一个满的会话无路可走（issue #247）。

这样的历史现在分阶段折叠。每个阶段携带迄今已折叠的所有内容的摘要，加上下一段对话记录，两者都按摘要器自身的上下文窗口来度量（当它指定了另一个模型时，按 `compaction.model` 的窗口），并返回一份覆盖两者的摘要。最后一个阶段的答复就是进入对话记录的内容，因此一次花了七次调用的压缩会让会话看起来与只花一次调用的完全一样。被提供商以过大为由拒绝的阶段会用一半的对话记录重试，每个摘要器最多三次。即使单独也太大的一条条目会以省略中间、保留头尾的方式发送 - 折叠会取得进展，而不是停在那条它本就要折叠掉的消息上。任何措辞都无法识别的 400 或 413 会在该阶段继续之前被截断一次，因为一个措辞 Coddy 未曾见过的服务器不能让一个满的会话无路可走。任何其他拒绝 - 密钥被拒、故障、限制 - 与大小无关，在一半大小时也会以同样方式失败，因此该阶段会立即转到链中的下一个摘要器。各阶段按自动触发器所使用的估算来度量（三个 ASCII 字符折算一个 token，其他每个字符一个 token），每次重试都会以 warn 级别记录为 `compaction pass refused as too large`（对无法识别的拒绝的单次截断记录为 `compaction pass refused with a status that can mean too large`），并附带折叠估算的 token 数和提供商报告的 token 数，这能区分窗口设得过高与估算不足。`POST /coddy/sessions/{id}/compact` 会把各阶段报告为 `steps`。

## 会话运行期间显示什么

每一次压缩，无论以何种方式启动，都会绘制工具调用所绘制的那一行：开始时宣告为 `compact_context`，多阶段折叠运行期间随所在阶段更新（`compacting context: pass 3 of 7`），并以它折叠了什么作为结束。在此之前，会话在摘要器运行的整个期间只是安静下去，而一次对大历史进行多次调用的折叠会让人很难熬。

![压缩运行期间的 compact_context 行](../../assets/compaction/compact-row-running-dark-1280.png)
*从输入框发送的 `/compact`：该行说明摘要器撰写期间上下文正在被压缩，位于它即将折叠的轮次下方。截取自运行中的界面。*

压缩为自己绘制的那一行是实时的：它会送达正在观察该会话的任何人 - 发送 `/compact` 的输入框、第二个浏览器标签页、`--remote` 上的控制台 - 而之后留在对话记录中的是压缩摘要行。模型请求的压缩则不同，因为那里的行已经存在：`compact_context` 调用会复用它，因此各阶段会出现在下达它们的调用之下，并且它会像任何其他工具调用一样留在对话记录中。

## 保留什么、摘要什么

边界是第 `keep_recent_turns` 条最近的用户消息（默认 2）；如果它会保留的轮次超过 `keep_tail_tokens`（默认 15000），边界就向前移动：保留能容进该预算的最新轮次，最多 `keep_recent_turns` 个，至少一个，因此宽窗口留下 15k 的尾部，而不是自身的一份份额。从该边界起的所有内容 - 用户轮次、各轮次之后的回复和工具活动 - 保持原样。它之前的一切作为扁平化的对话记录发送给摘要器，工具调用渲染为带标签的行，并在模型的视野中被插入到边界处的一行摘要取代。更早写下的摘要不会被再次摘要（[摘要构成一条链](#摘要构成一条链)）。`keep_recent_turns: 0` 会摘要整个窗口，无论尾部预算怎么说；`keep_tail_tokens: 0` 关闭绝对上限，只由轮次数量决定。当窗口中只有正在被回答的那个轮次时，边界改为该轮次的一个步骤，如 [少数几个长轮次](#少数几个长轮次) 所述。自动边界始终是在正在被回答的提示词之前开始的用户轮次：你在该轮次期间排队的消息是它的一部分，压缩不会在那里切分。

摘要会向 `compaction.model` 请求（若已设置），否则向会话的模型请求，使用固定的系统提示词，依次要求：用户的目标与约束、已做出的决定与被否决的方法、工作的状态、重要的确切路径、名称、命令和值，以及未决问题与后续步骤。下文的结果驱逐投影会先应用于历史，因此模型已经越过的分页、搜索或列表不会被完整摘要。

工具结果或消息所携带的项目规则不会被摘要：被摘要折叠掉的规则会在下一次触及匹配路径的工具调用或提及时回来，而保留的尾部仍携带的规则，只有当其文件发生变化时才会再次出现。在轮次内写入的行会重复当时发送的提示词，因此提及附加到提示词上的规则被视为已送达，不会第二次附加；当提示词的截断把这样一条规则截断时，下一次匹配会完整地附加它。压缩也是重新读取常驻规则的时刻 - `AGENTS.md` 与 `DESIGN.md` 成对规则、始终适用的规则以及 `instructions.files` 的文件 - 因此会话期间对它们的编辑会进入下一次系统提示词（[规则与提示词缓存](rules.md#规则与提示词缓存)）。

### 摘要构成一条链

一次压缩只折叠最后一行摘要之后的内容，并在它之后添加自己的行；之前写下的行保持原样。模型在系统提示词之后读取它们全部（[系统提示词结构](../contributing/react-agent.md#系统提示词结构)），从最旧到最新，然后是原样保留的轮次。摘要器也会读取更早的行，把它们放在对话记录之前的一个独立章节中，作为它不得重复的上下文，它们的大小会从一次摘要请求可携带的容量中扣除。它们最多占一半：当放不下时，其中最旧的在这次请求中被略过，而链本身保持原样。

![压缩摘要链：磁盘上的对话记录、模型读取的内容、下一个块如何写入以及最早的块如何合并](../../assets/diagrams/compaction-chain.png)
*每次压缩追加一个摘要块，并且绝不重写更早的块：模型读取系统提示词、从最旧到最新的块，然后是原样的尾部，而超出窗口三分之一的链会把其最早的块合并成一行。*

像 Coddy 以前那样在每次压缩时重述上一份摘要，会一次接一次地重述，磨损最早轮次的细节，并且改变每个请求的开头，因此提供商的提示词缓存从第一条消息起就丢失了。有了链，两次压缩之间的请求在最后一行摘要结束之前以相同的字节开头，而压缩只在它之后添加。

链每次压缩增加一行。当其行占据会话模型上下文窗口的三分之一以上时，最早的那些会被合并：摘要器把它们重述为一行，重新开启这条链，插入在它所替代的最后一行之后，并且合并到使链降到该份额一半所需的行数，合并行按其重述的最长行计入；对摘要器的要求是不超过该长度的行。两者之间的距离使合并很少发生，因为合并确实会改变请求的开头。刚刚写入的行绝不会被合并。正在被回答的那个轮次的行最后才合并，仅当更早的行不够时 - 一个长轮次被一次又一次折叠 - 此时合并行以该轮次的提示词开头，就像该轮次的第一行那样，因此窗口中仍只保存一份它。失败的合并让链保持原样；压缩本身已完成，其结果会说明这一点。自上一次以来没有任何内容的 `/compact` 没有可折叠的内容，不会重述链。

在链存在之前被压缩的会话在其窗口开头有一行摘要，它被视作链的第一行读取：下一次压缩在它之后添加。

## 摘要行与上下文估算

摘要行是一条标记为 `compaction_summary` 的用户角色消息，以 `The earlier conversation was compacted. Summary of the compacted part:` 开头。压缩写入的每一行也标记为 `compaction_block`（合并行不是），而前面有行的则以 `The conversation after the summary above was compacted as well. Summary of the compacted part:` 开头。模型的窗口是这条链 - 每一行摘要回溯到最近一个不带 `compaction_block` 的行（合并行，或在链存在之前写入的行），或当它们全都带有该标记时回溯到第一行 - 然后是最后一行之后的消息；这些行覆盖的消息仅为了对话记录而保留在 `messages.json` 中。轮次内部的折叠写入的行以提示词开头，其后是 `The text above is the message this turn started with, verbatim. Coddy compacted the conversation before this point, including the earlier steps of the work on it. Summary of the compacted part:` 这一行（[少数几个长轮次](#少数几个长轮次)）。该行以相同方式标记。当链的最后一行之后是一条助手消息时，它代表携带其提示词的那个轮次，因此下一个提示词会把它之后的步骤作为更早的轮次折叠。所有读取某行的代码都在其中找到相同的边界：该行的第一次出现把开头消息与摘要分开。[会话监督者](session-supervisor.md)把其前面的部分标记为 `[the message this turn started with]`，并把其后的部分读作工作者的主张，各自单独截断，因此长提示词不会把摘要挤出其证据摘要，长摘要也不会把提示词挤出去。本身引用了前言行的那段提示词是边界唯一可能误读的输入。`PreCompact` 钩子在两个触发器之前运行并可否决它，`PostCompact` 钩子接收摘要（[钩子](hooks.md#事件)）。它们只为有内容可折叠的压缩运行：高于阈值但没有更早内容可折叠的步骤不再唤醒它们。

压缩之后，上下文估算会重新计算并作为 `usage_update` 发布，带有 `used` 和 `size`，这就是输入框的上下文环和控制台页脚的上下文百分比所显示的；`GET /coddy/sessions/{id}/stats` 按类别返回相同的分解。共享会话的每个客户端都读取更小的那个数字：从自己的流中发送该轮次的标签页、从 `GET /coddy/sessions/{id}/composer-stream` 观察该轮次的另一个标签页（相同的帧），以及仅从 `turn_ended` 到达时重新加载的统计中查看该会话的标签页。压缩折叠保留尾部之前的轮次，因此当上下文的主体位于较早轮次时数字会下降；一条很大的最后消息会保持原样，直到更新的轮次把它推过 `keep_recent_turns`。

Context 弹出框底部、其列表下方，说明压缩何时自行开始：配置的阈值及其折算的 token 数，例如 **Auto-compact at 80% · 104.8k**（默认 `threshold_percent`），或 **Auto-compact is off** 加 `auto_enable: false`。它旁边的 **Compact now** 启动与 `/compact` 相同的 `POST /coddy/sessions/{id}/compact` 动作，显示进度，并在完成时刷新对话记录和上下文指示器。

![带手动压缩动作和已配置阈值的 Context 弹出框](../../assets/compaction/context-popover-threshold-dark-1280.png)
*在 1280px 下从运行中的 Web 界面渲染的 Context 弹出框。*

Web 界面还会从任一流中的 `usage_update.size` 更新会话的上下文窗口。如果提供商列表在 `/v1/models` 返回 128000 回退值之后到达，环会采用所报告的窗口而无需重新加载。统计刷新会保留该实时窗口；选择不同模型或保存配置会使用模型列表，直到新的用量更新到达。为一个会话接收的窗口绝不会改变另一个会话的环。

![使用回退窗口的 Context](../../assets/compaction/context-window-before-dark-1280.png)
*在提供商应答之前：同样的 120000 token 占回退窗口的 93.8%。用替代提供商从运行中的界面截取。*

![使用迟到的提供商窗口的 Context](../../assets/compaction/context-window-after-dark-1280.png)
*在流报告 262144 token 的窗口之后：45.8%，无需重新加载。窄布局：[之前](../../assets/compaction/context-window-before-dark-390.png) 与 [之后](../../assets/compaction/context-window-after-dark-390.png)。*

## 对话记录显示什么

在 Web 界面中，摘要是一个标记为 **context compacted** 的折叠行，样式类似思考展开项，其正文渲染摘要 - 即现在模型上下文中的内容。正文从前言之后开始，因此在轮次内写入的行不会在那里重复其提示词：提示词仍是轮次顶部的用户行。该行位于边界处、原样保留的轮次上方，而不是聊天末尾；对一个短会话进行强制压缩后，它是最后一行。`/compact` 命令及其单行回复是普通的用户行和助手行。`GET /coddy/sessions/{id}/messages` 返回每一条原始行，摘要携带 `compaction_summary: true`（当它延续链时还带 `compaction_block: true`），而导出会把它写为 `compaction_summary` 条目（[会话导出](session-export.md)）。ACP 客户端以及随之而来的控制台在重放时看到的是一个普通行，作为以 `The earlier conversation was compacted.` 开头的用户消息，而延续链的行则以 `The conversation after the summary above was compacted as well.` 开头。在轮次内写入的行会从其前言开始重放，作为以 `The text above is the message this turn started with` 开头的用户消息，因为它前面的提示词已经在其自己的位置重放过，客户端会把它显示两次。

## 结果驱逐

对大文件分页、运行大范围搜索或列出十几个文件夹，否则会把每个结果都钉在上下文里直到会话结束。结果驱逐在构建请求时把模型已经越过的结果折叠为简短的占位符，因此对提供商来说工具调用及其结果仍然成对，而主体已经不在。两组工具会经过它，各有自己的窗口：`read` 和 `grep`，以及列表工具 `glob`、`print_tree`、`websearch` 和 `webfetch`：

```text
[evicted: internal/agent/react.go lines 1-200, not marked as useful; re-read if this range is needed again]
[evicted: grep "maybeAutoCompact" in internal/agent, not marked as useful; re-run the search if needed]
[evicted: internal/agent/react.go was modified after this read; re-read for current contents]
[evicted: glob "**/*.go" in internal/agent (312 lines); re-run if needed]
[evicted: print_tree of internal/session (87 lines); re-run if needed]
[evicted: websearch "godog parallel scenarios" (24 lines); re-run if needed]
[evicted: webfetch https://example.com/docs (140 lines); re-fetch if needed]
[evicted: glob "**/*.go" in internal/agent is stale after internal/agent/new.go was modified; re-run if needed]
```

当一个 `read` 或 `grep` 结果位于工作窗口之内 - 即最近 `keep_recent` 个候选（默认 2，因此一次读取和一次 grep 可以相互比较）- 或模型将其标记为有用时，它就会保留：`keep: true` 在 `read` 或 `grep` 调用本身上，或在之后一次命名该分页（`path`，可选 `offset` 和 `limit`）或该搜索（`pattern`，可选 `path`）的 `keep_result` 调用中。`keep_result` 不读取任何内容；历史中记录的调用就是那个钉。成功写入文件（`write`、`edit`、`apply_patch`、`mv`、`rm`、`touch`、`mkdir`、`rmdir`；不是被拒或失败的调用）会使该文件更早的每次读取以及搜索根或命中触及它的每次 grep 过期，无论是否被钉：它们会以 "was modified" 占位符折叠，以便模型重新读取当前内容。大小等于或小于 `min_result_bytes`（2000）的结果从不是候选。`read`、`grep` 和 `keep_result` 的描述会告诉模型，未标记的结果会折叠为占位符并在写入后过期，以及如何钉住一个结果；各模式的提示词也这么说，并补充说明折叠的结果可以通过重复该调用回来。`keep_result` 在每个模式中都提供，包括 ask 模式。列表工具的描述不提驱逐：模型只以占位符的形式遇到列表占位符，该占位符会指明要重复的调用。

列表重新请求的成本很低，因此它没有标记，其窗口按步骤计数。一个步骤是一条助手消息及其所发出每个调用的结果：十个并行的 `glob` 和 `print_tree` 调用算一个步骤。含有列表的最近三个步骤的结果保持完整；更早步骤的结果折叠为占位符，重复该模式、查询或 URL，并说明结果有多少行。含有列表的最新步骤始终保留其结果，因此模型刚请求的列表能到达它，而不是在下一次请求中被折叠并再次请求。两个窗口相互独立：列表不占用读取和 grep 的 `keep_recent` 窗口中的位置，读取也不会把列表挤出步骤窗口。`glob` 或 `print_tree` 结果也会过期，然后即使在窗口内也会以 "stale after" 占位符折叠，因为列表可能已不再与磁盘匹配。在 `glob` 或 `print_tree` 所列文件夹之下的任何成功写入、对该文件夹本身或其周围文件夹的写入（在其下写入、编辑、创建或删除文件，移动或删除该文件夹）都会使该列表过期。搜索或分页描述的是网络，不是工作区，没有任何写入会使它过期。大小等于或小于 `min_result_bytes` 的结果在这里也不是候选。过期比较路径时区分大小写，Windows 除外，因此在大小写不敏感的 macOS 卷上，以不同大小写拼写的写入不会使更早的读取、grep 或列表过期。

`run_command` 和 MCP 结果绝不会被驱逐：命令或远程服务器第二次可能给出不同应答，或者可能因运行而改变了某些东西，因此占位符可能隐藏重复调用也无法带回的内容。列表工具及其三步窗口是固定的：开关是 `result_eviction.enable`，而 `start_percent` 与 `start_tokens` 决定它们何时开始，就像对读取和 grep 一样。

驱逐是对同一历史的第二层投影：持久化的对话记录完整保留每一份结果，而 `GET /coddy/sessions/{id}/tool-calls/{toolCallId}` 提供它。它补充了 `tools.output_limits`，即结果产生时对其大小设限的每工具行数上限（[config.yaml 参考](../reference/config.md#toolsoutput_limits)）。

## 配置

```yaml
compaction:
  enable: true             # master switch: the command and the automatic trigger
  auto_enable: true        # false keeps manual compaction; no threshold trigger, no fold inside a turn, no retry after a refusal
  threshold_percent: 80    # auto-compact at this percent of the model's context window (1..100)
  keep_recent_turns: 2     # user turns kept verbatim; 0 summarises everything
  keep_tail_tokens: 15000  # absolute ceiling on that tail; 0 leaves keep_recent_turns alone
  model: ""                # models[].model for the summariser; empty = the session's model
  fallback_models: []      # summarisers tried in order when the one before fails; the session's model is always in the chain
  result_eviction:
    enable: true           # read, grep and the listings of glob, print_tree, websearch and webfetch
    keep_recent: 2         # most recent read/grep results kept intact
    min_result_bytes: 2000 # results at or below this size are never evicted
    start_percent: 50      # evict only once the context reaches this percent of the window
    start_tokens: 60000    # ...or this many tokens of conversation, whichever comes first
```

| 键 | 默认值 | 含义 |
|---|---|---|
| `enable` | `true` | 是否启用压缩：命令、REST 路由和自动触发器 |
| `auto_enable` | `true` | 自动压缩：阈值触发器、长轮次较早步骤的折叠，以及提供商以过大为由拒绝请求后的压缩与重试；为 false 时手动操作仍然可用 |
| `threshold_percent` | `80` | 自动触发器，以模型 [上下文窗口](#上下文窗口) 的百分比表示 |
| `keep_recent_turns` | `2` | 原样保留的用户轮次（及其后的活动） |
| `model` | `""` | 当会话的模型不应摘要自己的历史时，用于摘要器的 `models[].model` |
| `fallback_models` | `[]` | 依次尝试的摘要器，在前者失败时使用下一个；会话的模型始终在链中，除非它已有更早的位置，否则在最后（[历史大于一次摘要请求](#历史大于一次摘要请求)） |
| `result_eviction.enable` | `true` | 折叠被取代的 `read`、`grep` 和列表结果 |
| `result_eviction.keep_recent` | `2` | 保留为工作窗口的最近 `read` 和 `grep` 结果 |
| `result_eviction.min_result_bytes` | `2000` | 大小等于或小于此值的结果保持原样 |
| `result_eviction.start_percent` | `50` | 驱逐开始前上下文必须有多满；`0` 表示从第一个结果起就驱逐 |
| `result_eviction.start_tokens` | `60000` | 同一门槛的对话 token 形式；两者中较小者开始驱逐；`0` 只保留 `start_percent` |

`start_percent` 之所以存在，是因为驱逐和提供商的提示词缓存方向相反。占位符被写入 **重放历史的中部**，而提供商按前缀缓存请求，因此一个被折叠的结果会丢弃其后每条消息的缓存副本。在滑动工作窗口下，这几乎在每一步都会发生，于是长对话会以全价重新处理，只为省下窗口尚未短缺的几千个 token。因此在该标记以下，历史完全按提供商已有的样子发出；在该标记以上，空间比缓存更重要。`start_tokens`（默认 60000）是同一门槛的 token 形式：宽窗口的一半远在死结果真正开始碍事之前，因此驱逐在窗口的 `start_percent` 与 `start_tokens` 中较小者处开始。门槛只测量对话：系统提示词、工具定义和规则不计入，因为它们无法被驱逐，而很大的 `AGENTS.md` 否则会在第一个结果处就打开门槛。没有 `max_context_tokens` 的模型条目无法被测量，会像以往一样从第一个结果起驱逐读取和 grep；此类条目的列表会等待会话所工作的窗口的 `start_percent`，即 [上下文窗口](#上下文窗口) 所解析的那个窗口。关于同一故事的另一半 - Coddy 出于同样原因不再放入系统提示词的内容 - 见 [react-agent.md](../contributing/react-agent.md) 中的 *The turn context block*。

带类型和验证的字段表在 [config.yaml 参考](../reference/config.md#compaction) 中；这些键是普通设置，可在 Web 界面设置中的 **Context compaction** 标签页编辑，该标签页跟在 **ReAct loop** 之后，因为它就是决定向模型发送什么内容的同一个循环（`#/settings/compaction`）。阈值所依据的窗口属于模型条目：它的 `max_context_tokens`，否则是其提供商报告的窗口，再否则是 128000（[上下文窗口](#上下文窗口)）。

## 测试

- `features/` 中的可执行测试规格：`context_compaction.feature`（保留的轮次、下一次请求中的摘要、经 ACP 的更小上下文、分阶段折叠的历史及客户端观察到的行、被以过大为由拒绝并减少内容再次请求的摘要请求、在 Coddy 压缩该轮次并再次请求后提供商仍以大于窗口为由拒绝请求时以解释结束的轮次，以及模型通过 `compact_context` 折叠自己的历史；框架 `internal/agent/bdd_compaction_test.go`）、`context_in_turn_compaction.feature`（一个由并行读取组成的长轮次，经多次折叠仍留在小窗口内，每个请求都携带提示词，该轮次中输入的后续消息绝不取代提示词，监督者启动的目标轮次其摘要行不把其指令称为用户请求，提供商以过大为由拒绝的请求被压缩并再次请求，以及宽窗口上的折叠落在尾部预算上；框架 `internal/agent/bdd_in_turn_compaction_test.go`）、`context_compaction_command.feature`（提示词界面和 REST 端点上的 `/compact`；`external/httpserver/bdd_compaction_http_test.go`）、`context_compaction_auto.feature`（高于阈值的提示词在回复之前压缩，而没有 `max_context_tokens` 的模型在其提供商报告的窗口处压缩，即 `GET /v1/models` 所显示的窗口；`external/httpserver/bdd_compaction_auto_test.go`）、`context_result_eviction.feature`（被标记的分页和搜索保留，未标记的折叠，旧的目录列表折叠而最新步骤保留，输出限制；`internal/agent/bdd_result_eviction_test.go`）。
- 单元测试：`internal/session/compaction_test.go`（拆分索引与可见窗口、摘要行的链及其起点、链和对其而言过长的链的 `@session:` 摘要，后者保留最新的行以及长轮次的请求）、`internal/agent/compact_chain_test.go`（在链之前写入的行由一个块延续、向摘要器展示且不重述的更早的行、链之后无内容可折叠、自第一次以来无内容的第二次 `/compact`、把更早的行作为最新一段发送给更小的摘要器）、`internal/agent/compact_merge_test.go`（一次合并重述多少行、进行中轮次的行最后合并、合并行去哪里、一次合并之后不会立即再次合并、失败的合并使压缩仍算完成、一个被折叠二十次的长轮次在其份额内保留且提示词只有一份、更早的行适配更小的摘要器、大于新窗口的链在压缩后合并、与其重述的行一样长的合并行仍使链保持在目标上、当合并带走携带轮次提示词的那一行时该提示词被计入）、`internal/session/compaction_turn_test.go`（轮次开头提示词在后续消息之后被找到及其标记在重启后仍保留、含并行批次的轮次的步骤边界绝不会被切开、自动拆分绝不在后续消息处切分、轮次内行的布局及所有读取方共享的边界、其 ACP 重放和监督者的摘要）、`internal/remote/follow_test.go`（控制台远程模式中的相同重放）、`internal/agent/compact_in_turn_test.go`（预算下边界的选择、受尾部预算限制的目标、自动触发器和恢复的下限、恢复偏好的常规拆分、提示词上限及其标记、跨链式折叠的一份提示词、触发器对轮次内折叠的回退、按提供商计数缩放的目标、一次仍高于阈值之后不再自动折叠、无内容可折叠时 `PreCompact` 钩子不运行、行的提示词所携带的规则）、`internal/agent/context_overflow_test.go`（消息及随之结束的轮次、拒绝所揭示的限制、每步骤一次的重试及其重置、流式输出后不重试、恢复期间的取消、强制驱逐及其范围、关闭恢复的开关）、`internal/agent/prompt_cache_compaction_test.go`（一个长轮次：在驱逐门槛以下历史只增长，在门槛以上只有列表结果变成占位符，而轮次内部的折叠是唯一的另一种重写；属于 `make test-cache`）、`internal/session/context_window_test.go`（解析顺序、询问哪些提供商、列表缓存及其有界等待）、`internal/llm/model_list_test.go`（列表的窗口字段）、`internal/llm/context_overflow_test.go`（OpenAI 和 Anthropic 客户端的拒绝按到达时读取、流内帧和仅有文本的错误、什么不算溢出、从拒绝中读出的大小、不重试）、`internal/agent/react_test.go`（阈值、更少的保留轮次、恢复的轮次）、`internal/agent/compact_fold_test.go`（阶段预算及其下限、阶段在哪里切分、因单独放不下而被省略的条目、对被以过大为由拒绝的某阶段的重试以及对任何其他拒绝的故障转移，正在被回答的提示词在 `keep_recent_turns: 0` 之后仍保留，以及尾部被削减到能容进 `keep_tail_tokens` 的轮次）、`internal/agent/result_eviction_test.go`（钉、过期、占位符；列表的步骤窗口及作为一步的扇出、它们在所列文件夹下写入后的过期、它们对会话所工作窗口与 `start_tokens` 的门槛、门槛只按对话测量、生成历史上的工具调用配对）、`internal/config/compaction_test.go`（默认值和验证、压缩与驱逐的绝对预算）。
- 针对真实提供商的实机测试：
  - `examples/httpserver/http_e2e_compact_auto.py` 启动自己的 `coddy serve`，使用没有 `max_context_tokens` 的模型，检查 Web 界面的窗口、流的 `usage_update` 和触发器三者一致，以及会话在没有 `/compact` 的情况下压缩；
  - `examples/httpserver/http_e2e_compact_clients.py` 扮演一个会话的三个客户端 - 发送方、输入框流上的第二个标签页、统计上的空闲查看者 - 并检查它们在 `/compact` 之后、在 `POST .../compact` 之后（在 `GET /coddy/events` 上宣告）以及在一次自动压缩之后都读到上下文用量下降；
  - `examples/cli/cli_e2e_compact.py` 在 pty 中驱动控制台，并检查页脚的上下文百分比在 `/compact` 之后下降，并在下一轮次保持较低。

<!-- docsgen:source sha256=8dcb32ec5149c3b7 -->

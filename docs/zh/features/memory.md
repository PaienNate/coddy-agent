# 长期记忆

在 LLM 意义上，记忆是任何进入上下文的内容：聊天历史是短期部分，并随会话结束而结束。Coddy 中的长期记忆是磁盘上的一组 Markdown 和纯文本笔记，由专门的子智能体 **记忆子智能体** 查阅和更新。主智能体像委托给任何子智能体一样委托给它：在捆绑的 `memory` 定义下，`spawn_agent` 调用会启动一个子智能体，它拥有自己的会话包、自己的对话记录和自己的任务日志（[子智能体](subagents.md)、[后台任务](background-tasks.md)）。主智能体从不看到记忆工具；它只看到子智能体的报告，作为该 `spawn_agent` 调用的结果，而笔记比任何会话都存续更久。

记忆**按需运行**：在一轮中，直到模型决定请求，什么也不会发生。不会为每条消息启动后台运行，而从未提及笔记的一轮不会为记忆付出任何代价。

## 记忆子智能体做什么

子智能体把父级提示词作为其任务，并由拥有指南的记忆根目录的 [guides](#指南) 打开，同时拥有自己的系统提示词（`external/memory/prompts/memory_agent.md`，嵌入在二进制中，因此更改它意味着重新构建）。它在两种模式中选择一种，同一运行中绝不两者兼用：

- **Recall** 为主助手加载上下文：它搜索笔记、列出文件夹并读取文件，然后在 `Already on disk` 下报告纯事实，并可选在 `Not in notes` 下报告 - 没有路径，没有文件名 - 或当没有匹配时恰好报告 `(no memory hits)`。Recall 是子智能体不确定时的默认模式，并且当第一次搜索一无所获时，它会用英文关键词和文件夹列表重试；
- **Persist** 在父级要求记住、保存、忘记或删除某内容时，或当从对话写入持久笔记是明显意图时，更新笔记。它先读取现有笔记以避免重复，在第一次保存到某个文件夹前创建该文件夹，并报告它验证、保存、跳过或删除了什么。密钥、令牌和密码绝不写入。

提示词告诉子智能体，回答请求是主助手的职责：其报告只携带笔记所包含的内容。子智能体的最后一条助手消息就是报告，它作为 `spawn_agent` 工具结果到达父级，像任何其他委托一样。任务在 32 KiB 处截断，这是 `spawn_agent` 提示词的界限。

## 模型如何调用它

记忆子智能体是一个捆绑的子智能体定义，`internal/subagents/bundled/memory.md`，在面向模型的目录中以名称 `memory` 列出。模型通过 `spawn_agent` 使用 `agent: "memory"` 以及说明要回忆或持久化什么的提示词来访问它：

```text
spawn_agent  agent: "memory"
             prompt: "what do the notes say about our deployment target?"
```

两个开关决定是否提供该名称：二进制必须使用 `memory` 构建标签构建（发布二进制和 Docker 镜像带有它；从源码构建时，`make build TAGS="... memory ..."`，见 [从源码构建](../contributing/build.md)），并且 `memory.enable` 在 `config.yaml` 中必须为 true。只要任一开关关闭，该定义就**从目录中隐藏**，因此模型永远不会看到一个会被拒绝的启动，而一个仍然命名 `memory` 的 `spawn_agent` 会带着原因被拒绝。没有该标签时，记忆键会被接受，但不会运行任何东西。操作者自己的名为 `memory` 的定义会像任何其他内置定义一样替换捆绑定义。

启动的子智能体是一个**普通的 `agent` 类后台任务**，不是特殊的系统任务：它像任何子智能体运行一样由 `background_list` 列出，可以用 `background_output` 读取并用 `background_wait` 等待，它计入 `subagents.max_concurrent`，并且像任何委托一样触发 `SubagentStart` 和 `SubagentStop` 钩子（[钩子](hooks.md)）。它以智能体模式运行，只有六个记忆工具，没有其他东西：没有 shell、没有文件系统工具、没有 MCP 服务器。它不能在自己的 worktree 中运行；在创建检出之前启动就会被拒绝。其超时来自定义的前置元数据（`timeout_seconds: 300`），像每个任务一样受 `tools.background.max_timeout_seconds` 限制。

在智能体和计划模式下，子智能体拥有完整工具集。在 **ask 模式**下，运行仅限回忆：子智能体获得搜索、列出和读取工具，其提示词说明保存不可用，而它仍然请求的保存会在执行前被拒绝，因此只读会话不能更改已存储的内容。

当 `memory.model` 指定了已配置的模型时，子智能体在该模型上运行，否则在会话的模型上运行。其补全上限为 `copilot_max_tokens`，其 ReAct 轮次上限为 `recall_max_turns` 和 `persist_max_turns` 中较大者。一个宕机或过载的模型不必让运行付出代价：`memory.fallback_models` 是在 `memory.model` 之后尝试的链，而会话自己的模型是最后手段，无论它是否被列出。当调用在产生任何输出之前失败（模型宕机、未授权、配额不足）时，链会移到下一个模型；已经在文本交付后中断的流则遵循普通重试策略，因此不会有答案被流式传输两次（[issue #247](https://github.com/coddy-project/coddy-agent/issues/247)）。

子智能体的系统提示词是模板 `external/memory/prompts/memory_agent.md`，内置于二进制中。`memory.additional_prompt` 在记忆角色之后、工具列表之前向其中添加你自己的一个章节，**操作者指令**：一个用你的话固定行为的地方，例如“只处理笔记；绝不要回答任务本身”，或指定笔记所使用的语言。主智能体从不看到该文本，ask 模式也以同样方式呈现它。`memory.additional_prompt_max_chars` 会在那么多字符处截断更长的文本，在每次读取被截断文本的启动时于智能体日志中给出警告，并在 `coddy -t` 中产生一项发现；`0` 保持其完整（[issue #266](https://github.com/coddy-project/coddy-agent/issues/266)）。

## 提示词缓存

提供商按请求的前缀缓存请求：先是工具定义，然后是从开头开始的消息，而与上一个请求不同的第一个字节会让其后缓存的一切付出代价（[ReAct 智能体](../contributing/react-agent.md#系统提示词在一轮内冻结)）。记忆子智能体是一个单独的对话，拥有自己的简短提示词；它与父级不共享前缀，也不从父级缓存中取任何东西。记忆根目录的指南属于它的任务，即其系统提示词和工具定义之后的一条消息，因此编辑指南不会移动提供商为记忆对话缓存的任何内容。

在父级一侧，报告是 `spawn_agent` 调用的结果，因此它像任何其他工具结果一样追加到历史末尾，并且永不移动缓存前缀。主智能体的系统提示词只携带会话笔记（模板的 `{{.Memory}}` 槽位，`agent.md`、`plan.md`、`ask.md`；见 [配置](../getting-started/configuration.md)），因此每次调用都不同的回忆不会触及 `messages[0]`。

## 观察一次运行

启动的子智能体是一次普通的子智能体运行，因此它在任何子智能体运行被观察的地方被观察。Web 界面的任务面板将其列为带有 `memory` 标签的卡片（标签来自智能体名称，就像 `spawn_agent` 子智能体携带其智能体名称一样），标题为父级传入的提示词；**显示对话记录**以只读方式打开子会话，输入框被替换为返回聊天的链接。打开的卡片显示子智能体的进度日志：其工具调用及其文本，然后是 `=== subagent report ===` 块。已完成的运行留在抽屉的已完成列表中，并随会话的其他任务一起清除；删除会话会连同其记忆子智能体一起移除。

面向模型的池工具像任何其他智能体任务一样看到该运行：`background_list` 列出它，`background_output`、`background_wait` 和 `background_stop` 接受其 id。线路上没有单独的 `memory_run` 更新，也没有每轮状态行：该运行通过普通的 `agent` 类任务行和 `spawn_agent` 工具结果报告。

![任务面板中有一个正在进行的记忆运行：记忆标签和已用时间](../../assets/memory/memory-tasks-running-dark-1280.png)

*任务面板中有一个正在进行的记忆运行：`memory` 标签和已用时间*

![就地打开的一个已完成记忆运行的卡片：子智能体的日志和报告块](../../assets/memory/memory-tasks-detail-dark-1280.png)

*已完成的运行：子智能体的日志和报告块*

![从运行中以只读方式打开的记忆子智能体对话记录](../../assets/memory/memory-child-transcript-dark-1280.png)

*从运行中以只读方式打开的记忆子智能体对话记录：任务、搜索和报告*

## 边界

- **时间。** 一次运行受定义的 `timeout_seconds`（300）限制，池像每个任务一样受 `tools.background.max_timeout_seconds` 限制。达到该限制会停止子智能体，并将任务记录为 `timed_out`。
- **并发。** 该运行像任何子智能体一样计入 `subagents.max_concurrent`，因为它就是一个子智能体。
- **能力。** 子智能体拥有六个其父级没有的工具，这是子智能体工具集不会收窄父级工具集的唯一地方：该集合在代码中固定，只授予标记为 `memory` 类型的子智能体，并且只接触两个笔记根目录，没有其他东西。
- **隔离。** 启动拒绝 worktree 隔离：记忆子智能体在父级的工作区中运行，绝不在自己的检出中运行。
- **钩子。** `SubagentStart` 和 `SubagentStop` 像任何委托一样为记忆子智能体触发，并且在子智能体内部，每个普通事件都会为其自己的轮次触发。载荷的 `subagent` 块携带 `"kind": "memory"`，即用于匹配的键（[钩子](hooks.md)）。
- **退出。** 在持久化中途停止的运行会丢失其笔记。

## 工具

| 工具 | 参数 | 回忆 | 持久化 |
|---|---|---|---|
| `coddy_memory_search` | `query`, `scope` (`global`, `project`, `both`) | 是 | 是 |
| `coddy_memory_list` | `path` (`global:` 或 `project:notes`；一层) | 是 | 是 |
| `coddy_memory_read` | `path` | 是 | 是 |
| `coddy_memory_mkdir` | `path` | | 是 |
| `coddy_memory_save` | `title`, `body`, `scope`, 可选 `relative_path` | | 是 |
| `coddy_memory_delete` | `path` (一个文件，或包含其下所有内容的文件夹；绝不是根目录) | | 是 |

路径是 `scope:relative/path.md` - `global:preferences.md`、`project:architecture/api.md` - 笔记正文内的链接应使用相同形式（或使用具有此类目标的 Markdown 链接），以便它们在两个根目录之间保持无歧义。搜索使用 [内置文档搜索](built-in-docs.md) 的 BM25F 排序器，将所选根目录下的每个笔记与查询进行排序，返回最多 `max_search_hits` 个片段，每个片段最多 1,200 个字符，最佳优先，并且旨在作为入口点：子智能体用 `coddy_memory_read` 打开命中项，并跟随其中的链接。没有 `relative_path` 的保存会获得一个从标题派生的平面文件名，保留任何文字体系的字母；有它时，笔记会落入该文件夹。

### 搜索如何排序

笔记有两个字段：其路径（文件夹和文件名，去掉 `.md` 或 `.txt`）和其正文。路径中的一个词说明笔记是关于什么的，因此它按正文中一个词的三倍计入。权重是代码中的常量，不是设置。

- **词干提取。** 词在词干提取后匹配，英语、俄语和中文都一样，因此 `configs` 会找到写着 `config` 的笔记，俄语查询会找到词的另一种形式（`настройка` 找到 `настройки`），中文按相邻两字匹配（`安装` 能找到包含“安装”的笔记）；
- **稀有词权重更高。** 少数笔记拥有的词胜过大多数笔记都有的词，重复一个词每次增加的权重更少，因此包含查询中每个词的笔记排名高于多次包含一个高频词的笔记；
- **前缀。** 三个字母或更长的查询词也会找到以它开头的词：`deploy` 找到 `deployment`，排在写着 `deploy` 的笔记之后；
- **功能词不计分。** `how`、`to` 和 `the`（及其俄语对应词）会从查询中丢弃，因此 `how to deploy the relay` 会优先排序关于中继部署的笔记，并且不会返回关于共享 `how to the` 的通用操作指南笔记。仅由功能词组成的查询什么也找不到；
- **顺序。** 分数相同的笔记按路径排序，命中项携带其分数；
- **不是指南。** 作用域根目录下的 `AGENTS.md` 绝不是命中项（[指南](#指南)）；子文件夹中的一个是普通笔记。

索引由每次调用读取的文件构建。笔记在调用之间会变化，且存储很小，因此没有什么需要保留或失效。

笔记正文最多包含 `memory.max_note_chars` 个字符（默认 900；是字符，不是字节，因此西里尔文或其他非拉丁文字体系的笔记与英文笔记拥有同样空间）。工具定义说明了上限，因此子智能体在调用前就知道它，其提示词告诉它将较长主题拆分为多个笔记。超过上限的正文会在字符边界处截断，以 `...` 行结尾，并且保存会回答 `saved as <path> (warning: body truncated to 900 characters)` 而不是普通的 `saved as <path>`，因此截断绝不会无声发生（[issue #429](https://github.com/coddy-project/coddy-agent/issues/429)）。`0` 移除该上限。

## 存储布局

| 根目录 | 位置 | 笔记的作用域 |
|---|---|---|
| 全局 | `memory.dir`，或未设置时的 `$CODDY_HOME/memory`（`~/.coddy/memory`）；`${CODDY_HOME}` 和 `~` 会展开 | 由每个会话共享 |
| 项目 | `<session cwd>/.coddy/memory`，不可配置 | 会话运行所在的工作区 |

只有 `.md` 和 `.txt` 文件计入。鼓励使用嵌套文件夹进行主题分组，子智能体在保存到树的新分支之前会调用 `coddy_memory_mkdir`。两个根目录都是普通目录：它们可以用任何编辑器编辑，项目根目录可以随仓库提交或被忽略，视笔记的需要而定。

两个根目录对链接的处理不同。全局根目录是你的：其中的笔记或文件夹可以是指向任何地方的链接，例如一个同步文件夹，Coddy 会跟随它。项目根目录随检出而来，因此其中的链接绝不会指向其外部。Coddy 通过 `os.Root` 到达它：只有当链接的目标保持在内部时，才会跟随其中的链接，并且 `.coddy` 和 `.coddy/memory` 必须是真实文件夹，而不是链接。克隆仓库中指向 `~/.aws/credentials` 的链接笔记既不会被读取，也不会被搜索找到，因此它永远不会到达模型，保存也绝不会落入指向另一个项目的链接文件夹中。删除这样的链接会移除链接，并保留它所指向的内容。本身是链接的项目根目录会被拒绝，并给出说明这一点的错误；对两个根目录的搜索仍会从全局根目录回答。

## 指南

子智能体仅通过搜索、列出和读取，加上其固定提示词来了解笔记。记忆根目录根部的 `AGENTS.md` 是告诉它你如何保存笔记的地方：一个起始索引笔记、每个主题一个文件夹、命名约定、绝不要保存什么。`memory.additional_prompt` 对每个工作区做同样的事；一个指南属于一个根目录。

- **位置。** `AGENTS.md` 位于全局根目录顶部（`$CODDY_HOME/memory/AGENTS.md`，或 `memory.dir` 下）以及项目根目录顶部（`<workspace>/.coddy/memory/AGENTS.md`）。文件存在就是开关，没有对应设置，而子文件夹中同名文件是普通笔记；
- **两个层级，全局优先。** 每次运行开始时，子智能体的任务先以全局指南开头，然后是项目指南，然后是提示词，每个指南都在说明其层级和路径的标题下。没有指南的层级不添加任何内容：没有标题，没有占位符。两个层级保持分开：两者都不覆盖或合并到另一个中，子智能体的提示词告诉它对自身根目录的笔记遵循每个指南；
- **像任何笔记一样读取。** 指南经过与 `coddy_memory_read` 相同的存储，因此项目指南获得项目根目录其余部分的保护：一个是通向 `.coddy/memory` 外部的链接，一个命名管道或其他非常规文件，或一个磁盘拒绝打开的文件（无权限、循环链接），都不会被读取，智能体日志中的警告会说明这一点，运行会在没有它的情况下继续；
- **大小。** 无论文件多大，Coddy 读取指南的前 8 KiB，并在文件还有后续内容时以说明这一点的标记结束。保持它是地图，并将细节放在它指向的笔记中；子智能体用 `coddy_memory_read` 和 `coddy_memory_list` 打开它们，或用 `coddy_memory_search` 找到它们；
- **不在提示词中。** 指南搭载在任务中，绝不在系统提示词或模板中：指南中的 `{{` 是纯文本，编辑指南不会更改子智能体的系统提示词或工具定义；
- **被引用，因此引用保持为文本。** 每一行指南都作为 Markdown 引用写入任务（其前有 `> `），被引用的行对 `@` 提及语法而言是散文。项目指南随检出而来，因此其中的 `@~/.aws/credentials` 或 `@/abs/path` 不会向子智能体的请求附加任何文件；
- **不是搜索命中项。** `coddy_memory_search` 不会返回根目录 `AGENTS.md`，因为子智能体已经拥有它。子智能体仍然可以列出并读取它，并在指南要求时在持久化运行中写入它。

两个指南都存在时，任务如下所示。

```text
Memory guide (global:AGENTS.md):
> <the global file, every line quoted>

Memory guide (project:AGENTS.md):
> <the project file, every line quoted>

Task from the parent:
<the prompt the parent passed>
```

没有指南时，任务就是其前导之后的提示词。

## 通过 HTTP 浏览笔记

同时使用 `http` 和 `memory` 构建的二进制会将会话的两个根目录作为 `/coddy/sessions/{id}/memory/*` 下的树提供；普通 `http` 构建会在那里回答 `404`。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/coddy/sessions/{id}/memory/tree` | 不带 `root` 时，为根目录 `global` 和 `workspace`；带 `root` 和可选 `path` 时，为下一层的 `.md` 和 `.txt` 子项 |
| GET | `/coddy/sessions/{id}/memory/file` | `root` 和 `path`；UTF-8 内容 |
| PUT | `/coddy/sessions/{id}/memory/file` | `{"root","path","content"}` |
| POST | `/coddy/sessions/{id}/memory/dir` | `{"root","path"}` 创建文件夹 |
| DELETE | `/coddy/sessions/{id}/memory/file` | `root` 和 `path`；一个文件，或递归删除文件夹；对根目录本身返回 `400` |

`workspace` 是该会话 cwd 的项目根目录。这些路由通过与记忆工具相同的存储到达两个根目录，因此上述存储布局规则在这里也适用：通过其拼写或通过工作区根目录中的链接离开某个根目录的路径，会用 `400` 和原因回答，`.coddy/memory` 是链接的工作区的每条路由也是如此。会话自己的笔记（`session.json` 中的 `agentMemory`）不能在这里编辑。这些路由是 Web 界面笔记中 [长期记忆](../surfaces/web-ui.md#长期记忆) 下写明的契约；记忆键本身是 Web 界面设置的 **记忆** 选项卡。

## 配置

```yaml
memory:
  enable: false
  model: ""                # models[].model for the memory subagent only; empty = the session's model
  fallback_models: []      # tried in order when the model above them fails before answering
  dir: ""                  # global root; empty = ${CODDY_HOME}/memory
  recall_max_turns: 6      # the child's round cap is the larger of the two
  persist_max_turns: 12
  copilot_max_tokens: 4096 # completion cap of the memory model's calls
  max_search_hits: 8
  max_note_chars: 900      # longest body one saved note may have, in characters; 0 = no cap
  additional_prompt: ""    # your own instructions for the memory subagent only; the main agent never sees them
  additional_prompt_max_chars: 0 # cut that text at so many characters (a warning is logged); 0 keeps it whole
```

| 键 | 默认值 | 含义 |
|---|---|---|
| `enable` | `false` | 是否公开记忆子智能体（需要 `memory` 构建标签） |
| `model` | `""` | 将子智能体固定到一个 `models[].model`；主智能体不受影响 |
| `fallback_models` | `[]` | 当上面的模型在回答前失败时按顺序尝试；会话自己的模型是最后手段，无论它是否被列出 |
| `dir` | `""` | 全局根目录 |
| `recall_max_turns`, `persist_max_turns` | `6`, `12` | 限制子智能体的 ReAct 轮次；有效上限是两者中较大者 |
| `copilot_max_tokens` | `4096` | 记忆模型调用的补全上限 |
| `max_search_hits` | `8` | `coddy_memory_search` 返回的片段数 |
| `max_note_chars` | `900` | 由 `coddy_memory_save` 保存的一个笔记可拥有的最长正文，以字符计；更长的会在工具的回答中带警告截断；`0` 移除上限 |
| `additional_prompt` | `""` | 你为记忆子智能体提供的自己的指令，渲染为其 **操作者指令** 章节；主智能体从不看到它们 |
| `additional_prompt_max_chars` | `0` | 在那么多字符处截断 `additional_prompt`，在智能体日志中给出警告，并在 `coddy -t` 中产生一项发现；`0` 保持完整 |

字段表在 [config.yaml 参考](../reference/config.md#memory) 中；`config.example.yaml` 携带相同的带注释块。Web 界面在 **设置 → 记忆** 下编辑相同的键，分组为 **模型和存储**、其下的 **回退记忆模型** 列表、**指令** 和 **限制**。

![设置 → 记忆：模型、回退模型、指令和限制](../../assets/memory/memory-settings-dark-1280.png)

*设置 → 记忆：模型和回退模型、操作者的附加指令及其上限，以及轮次和令牌限制及笔记大小上限*

## 成本与延迟

记忆只在模型请求时消耗令牌。从未启动记忆子智能体的一轮完全不会运行记忆模型；这样做的一轮会为一次子对话付费：子智能体自己的系统提示词（模板、环境块、六个工具定义）及其轮次，至少一次模型调用，使用工具时更多。用 `memory.model` 固定一个小而快的模型可以保持回忆成本低，同时主智能体仍使用你为它选择的模型。指南在每次运行时最多为每个根目录向子智能体的任务增加 8 KiB。

## 测试

- `features/memory_subagent.feature`（框架 `internal/agent/bdd_memory_test.go`，`-tags memory`）：捆绑定义和子捆绑下的启动、作为 `spawn_agent` 结果的报告、ask 模式的仅回忆子智能体、一次持久化、一个在字符上限内完整保留的西里尔文笔记、回退链、池准入、子智能体搜索的排序、其任务顶部的指南。`features/memory_http.feature`（`external/httpserver/bdd_memory_http_test.go`，`-tags http,memory`）：只读子对话记录和基于 REST 的笔记路由。
- 单元测试：`external/memory/agent_test.go`（模板、任务界限、指南：两个层级、一个层级、大小截断、花括号保持字面、链接后的指南）、`internal/agent/prompt_cache_memory_test.go`（子智能体的系统消息和工具定义在指南不同的运行间完全相同）、`external/memory/storage/storage_test.go`（搜索排序：英语和俄语的词干提取、稀有词、路径词、前缀、功能词、作用域和限制；片段按字符截断、嵌套写入、来自非拉丁标题的文件名、遍历、删除、链接目标）、`external/memory/storage/storage_links_test.go`（一个笔记、一个指南、一个文件夹和作为其外链接的项目根目录、一个保持在内部的链接、在全局根目录中跟随的链接）、`external/memory/tools/register_test.go`（工具定义）、`external/memory/tools/mem_save_test.go`（以字符计的笔记上限、截断警告、`max_note_chars`）、`external/httpserver/memory_http_test.go`（基于 REST 的遍历）。
- `examples/acp/acp_e2e_memory.py`、`examples/httpserver/http_e2e_memory.py` 和 `examples/cli/cli_e2e_memory.py` 驱动真实模型：预先植入的全局笔记会被回忆并塑造回复，一次 `remember` 轮次会持久化一个笔记，ACP 框架随后在一个全新会话中询问该事实，因此答案只能来自磁盘，还有一个步骤中只有植入的全局 `AGENTS.md` 能导向回答的笔记，并且运行的记录会在界面显示它的地方被检查（HTTP 上的 `memory` 任务行、用于 ACP 和控制台的 `<session>/subagents/` 下的子捆绑）。

<!-- docsgen:source sha256=a7923c9bb3b41fee -->

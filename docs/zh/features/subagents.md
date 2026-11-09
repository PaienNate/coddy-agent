# 子智能体

漫长的调查或并行展开（审查三个模块、按包运行测试矩阵、研究四个库）会填满操作者正在观看的那一个对话。**子智能体**是一个子级智能体运行，拥有自己的上下文窗口、自己的会话包，以及操作者在 markdown 文件中编写的角色提示词。父级交给它一个自包含的任务，继续工作，只取回子级的最终报告。

子智能体运行是**后台任务池中的一个任务**（`docs/features/background-tasks.md`）。调度、超时、任务面板、REST 界面、关闭时排空，或 `<session>/background/<task_id>/` 下的持久化，没有任何一点是子智能体特有的；新增的是第二种任务（`kind: agent`）、它背后的子会话，以及启动它的工具。

![一次 spawn_agent 调用会发生什么：定义来自哪里、运行时按什么顺序执行检查、子级的工具如何收窄、运行如何进入父级包、权限中继与恢复](../../assets/diagrams/subagents-spawn.png)
*一次 `spawn_agent` 调用与一个正在运行的子会话之间会发生什么：每项检查都可以用工具错误拒绝，只有通过最后一项检查，子级才会作为父级后台池中的任务启动。*

## 模型何时委派

可能生成子智能体的会话会在其系统提示词中得到一个 `## Subagents` 章节：`spawn_agent` 做什么、委派何时划算（会淹没上下文的工作，或可以用 `background: true` 并行运行的独立片段）、如何收集分离运行，以及它可以使用的定义目录。该指导明确说明，一步任务直接完成，并且新的子级以**空上下文**开始 - 提示词必须携带一切，因为子级看不到父级对话的任何内容。恢复的子级是例外：它保留自己的对话记录，因此其提示词只说明接下来要做什么（[恢复运行](#恢复运行)）。只有子级的最终消息会返回，而且用户永远看不到它，所以父级被告知要在自己的回复中重述重要内容。

该章节在 `agent` 和 `plan` 模式下呈现（规划器像 Claude Code 的 Explore 那样展开调查；计划模式父级的子级会被强制进入计划模式），并且从不在 `ask` 模式下呈现，该模式是只读的，不委派任何内容。对于无法生成子智能体的会话，该章节会被省略，工具会被隐藏：功能被禁用、界面没有会话管理器、轮次在 `ask` 模式下运行，或会话已经处于 `subagents.max_depth` - 除非其自身定义声明了 `spawns` 允许列表，这会保留该工具并将目录收窄到列出的名称（[深度](#能力如何收窄)）。

## 子智能体周围的钩子

操作者钩子（`docs/features/hooks.md`）会跟随智能体进入其子级。`SubagentStart` 在子级启动前于父级中触发（它可以拒绝生成，或交给子级一段前置到其任务中的上下文），`SubagentStop` 在子级轮次结束时于父级中触发，并带有其结果和报告。在子级内部，每个普通事件都会为子级自己的轮次触发，负载携带一个 `subagent` 块，指明子级、其父会话及其深度，因此钩子可以区别对待委派的工作。

## 定义文件

### 目录与优先级

定义会在四个默认目录中搜索，始终读取，优先级最低者在前，然后在 `subagents.dirs` 的额外目录中搜索；后面的目录会**按名称**替换前面的目录：

1. `${HOME}/.agents/agents` - 用户作用域，其他智能体在用户主目录中共享的文件夹；
2. `${CWD}/.agents/agents` - 项目作用域，工作区的共享文件夹；
3. `${CODDY_HOME}/agents` - 用户作用域，操作者自己的 Coddy 文件；
4. `${CWD}/.coddy/agents` - 项目作用域，工作区的 Coddy 文件；
5. `subagents.dirs` 的每个条目，按顺序，在默认目录之后，并优先于它们赢得名称（默认为空）。

该链与技能相同（[技能](skills.md#目录布局)）：除非 `subagents.dirs` 指定它，否则不会读取 Claude Code 的 `.claude/agents`（`"${CWD}/.claude/agents"`；其 frontmatter 原样加载）。一个被命名两次的目录只读取一次，在它最后的位置。`${HOME}`、`${CODDY_HOME}`、`${CWD}` 和前导 `~` 会展开；相对条目会相对于会话 cwd 解析。四个内置定义位于每个目录之下，因此同名用户文件会替换内置定义。

作用域按**规范路径**决定：展开后的目录和会话 cwd 都会经过 MCP 信任存储使用的同一规范化（绝对路径、解析符号链接、清理）。位于规范 cwd 处或其下的目录是**项目作用域**，并遵循 `subagents.project_trust`（见下文）；其他所有内容都是**用户作用域**。因此，通过符号链接到达的工作区仍然拥有其 `.coddy/agents`。

在一个目录内，每个 `*.md` 文件递归地都是一个定义；以点开头的文件和目录会被跳过。也接受 `<dir>/<name>/AGENT.md`，并且其默认名称来自**目录**，而不是 `AGENT` 词干。文件按字典序访问，第一个声明某个名称的文件获胜；重复项会被跳过并记录警告，没有 frontmatter 的文件、没有 `description` 的文件、名称、模式或权限模式无效的文件，或超过大小上限的文件也会被跳过并记录警告。

### Frontmatter

只有 `description` 是必需的。`name` 默认使用文件词干（对于 `AGENT.md` 则是目录），并转为小写。未知键会被忽略，因此为其他智能体编写的文件也能加载。

| 字段 | 别名 | 含义 |
|---|---|---|
| `name` | | 匹配 `[a-z0-9][a-z0-9_-]*` 的标识符；即模型传给 `spawn_agent` 的内容。 |
| `description` | | 向父模型显示的一行，以便它选择智能体；在目录中截断为 200 个字符。 |
| `model` | | 子级的 `models[].model` id。未知 id 会回退到父级的模型，并在智能体日志中记录警告，在任务的输出日志中记录一条说明。 |
| `reasoning` | `effort` | 子级模型提供的推理级别，`off` 或 `default`。模型不提供的级别会回退到其默认值，并在智能体日志中记录警告。 |
| `mode` | | `agent` 或 `plan`。空值继承父级的模式；只读父级（`plan`，以及 `ask`，如果生成曾经源自那里）始终强制使用自己的模式。父级模式是其轮次启动时的模式，而不是实时会话模式，因此轮次中途发生的模式切换不能拓宽子级。 |
| `tools` | | 允许列表：YAML 列表或逗号分隔字符串（`tools: read, grep`）。条目是精确工具名、裸 `*`，或 `prefix*` 模式，因此 `context7__*` 允许某个 MCP 服务器的每个工具。空表示父级拥有的一切。 |
| `disallowed_tools` | `disallowedTools` | 使用相同语法的拒绝列表；优先于 `tools`。 |
| `permission_mode` | `permissionMode` | `ask`、`accept_edits` 或 `bypass`。接受 Claude Code 拼写（`default` 和 `prompt` 表示 `ask`，`acceptEdits` / `accept-edits`，`bypassPermissions` / `bypass-permissions` / `dontask`）。它只能**收窄**父级的有效模式。 |
| `max_turns` | `maxTurns` | 子级 ReAct 轮数的上限。`0` 使用 `subagents.max_turns`，然后是 `agent.max_turns`。 |
| `timeout_seconds` | | 定义自身对单次运行的硬限制（见下面的超时优先级）。 |
| `background` | | `true` 强制进行分离运行，即使模型请求的是前台运行。 |
| `isolation` | | `worktree`：子级在自己的 git worktree 中工作（[自己的 worktree](#自己的-worktree)）。`none` 或无值：它在父级的文件夹中工作。任何其他值都会拒绝该文件。 |
| `hidden` | | 将定义排除在面向模型的目录和“未知智能体”列表之外。它仍然可以按名称生成，CLI 列表和 HTTP 目录会显示它并带有 `hidden` 标记。 |
| `spawns` | | 生成允许列表：使用 `tools` 模式语法的子智能体名称的 YAML 列表或逗号分隔字符串（`explore`、`peer-*`、`*`）。非空时会做两件事：子级只能委派给匹配的名称，并且可以越过 `subagents.max_depth` 一层这样做，因此处于上限的编排器仍然可以生成其审查者。仅对 `builtin` 和 `user` 作用域生效 - 项目文件的 `spawns` 会被解析并显示在目录中，但运行时会忽略它，因此签入的定义不能拓宽深度防护。 |

### 正文即角色

frontmatter 之后的一切都是子级的角色块。子级的系统提示词以固定前言开始（`## Your role as a subagent`：你是子智能体 *name*，由父级生成以完成一个自包含任务，你看不到父级对话的任何内容，不能向用户提问，只有你的最终消息会到达父级，因此以简洁报告结束），后面原样跟随正文。提示词的其余部分是普通智能体或计划模板，因此技能、规则和环境块与任何会话中一样存在。

一个最小定义：

```markdown
---
name: reviewer
description: Reviews a diff or a module for defects and reports findings with file paths and lines.
tools: read, glob, grep, print_tree, run_command
permission_mode: ask
max_turns: 20
---
You review code. Read before you judge, cite `path:line` for every finding,
separate defects from style remarks, and end with a short verdict.
```

### 边界

加载器和工具防止定义树淹没提示词：每个目录最多 **200** 个定义（其余会被跳过并记录警告），超过 **256 KiB** 的文件会被跳过，超过 **64 KiB** 的角色正文会被截断并带标记，`description` 截断为 **200** 个字符，超过 **32 KiB** 的 `spawn_agent` `prompt` 会被拒绝。任务标签为 `agent <name>: <description>`，其中 `description` 是调用自身的参数，回退到提示词的第一行；整个标签上限为 **60** 个字符，这是任务池对命令标签的规则。

每个加载的定义都是一个**不可变值**，携带其作用域、其路径以及文件字节的 SHA-256 **摘要**。文件只读取一次；信任、工具集和角色都基于同一个值决定，绝不再读。

## 内置定义

四个定义以嵌入形式随附，因此操作者编写文件之前委派即可工作。它们都以作用域 `builtin` 和路径 `(embedded)` 列出。

- **`general`** - 一个具有**父级工具集**的通用工作者（自身没有 `tools` 限制），用于多步骤任务、研究，或可以并行运行的独立工作单元。其角色告诉它先阅读再更改任何内容，保持编辑最小，在任务需要时验证，绝不重试操作者未授予的权限，并报告它做了什么以及父级仍必须决定什么。
- **`explore`** - 一个只读探索者，用于在提出更改之前定位文件、符号和用法并收集证据。其工具列表恰好是 `read`、`keep_result`、`glob`、`grep`、`print_tree`、`websearch`、`webfetch`、`load_skill`、`coddy_docs_search`、`coddy_docs_read`、`background_list`、`background_output`、`background_wait`：没有 `run_command`，没有写入，没有 MCP 工具。仅计划模式本身不会是只读的（它仍然提供 shell），这就是为什么该列表被明确写出。由于该集合中没有任何内容是 MCP 工具，`explore` 子级永远不会拨号 MCP 服务器。
- **`crossreview`** - 一个 `hidden` 编排器，将一份审查简报发送给外部代码智能体 CLI（`run_command` 后台任务）和内部 `explore` 子级，使它们彼此不可见，等待每个答案，对照代码验证每项发现，并独自决定修复什么、什么不值得修复以及拒绝什么；其报告是父级转述的最终结论。这就是 `spawns` 存在的原因：其定义声明了 `spawns: [explore]`，因此它在深度上限处保留 `spawn_agent`，并仅委派给那一种审查者。会话通过捆绑的 `/crossreview` 技能到达它，该技能处理检测、智能体和模型问题、审查者名册以及简报。
- **`memory`** - 长期记忆管理者，仅在带有 `memory` 标记且 `memory.enable` 开启的构建中列出（否则加载器会省略它，因此模型永远不会看到闸门会拒绝的生成）。委派给它会在任务之前召回并持久化长期笔记：针对全局和项目记忆根的六个 `coddy_memory_*` 工具（只读会话中的三个读取工具）、记忆提示词模板、带回退链的记忆模型，以及记忆根目录的 `AGENTS.md` 指南 - 生成调用的提示词就是任务。记忆根之外的一切都不在它的触及范围内，它不能在 worktree 中运行，因为这些根属于会话的工作区。参见[长期记忆](memory.md)。

## 作用域与项目信任

随检出一起到达的定义会指导子级的工具，而权限闸门并不覆盖子级能做的一切：`websearch` 和 `webfetch` 不询问任何人，MCP 调用绕过提示，而且模型可以自行委派，因为 `description` 会邀请它这样做。因此，**项目作用域**定义像项目本地 MCP 声明一样被信任：通过一份绑定到工作区和文件内容的带外回执，而不是通过聊天内提示（树中的每个发送者都会在 `permission_mode: bypass` 下自动允许提示，因此提示可能没有任何人能授予）。

`subagents.project_trust` 采用与 `mcp.project_trust` 相同的词汇：

- **`deny`** - 完全不读取项目作用域目录。智能体发现、`coddy agents list` 和 HTTP 目录会省略它们；测试证明这些文件保持未读取。
- **`ask`**（默认） - 项目作用域定义会加载，并在 `(canonical workspace, name, digest)` 存在回执之前列为 `needs_approval`（链接的 git worktree 使用其主检出的回执：[Git worktrees](worktrees.md#worktree-与其主检出共享的内容)）。在没有回执的情况下生成一个定义会在运行时的已解析定义上被**拒绝**：工具结果说该定义来自一个未为此工作区批准的项目文件，并列出批准它的方式，因此模型可以告诉操作者并在之后重试。提示词中的目录块仅以**名称**列出这样的条目，并附有固定通知，说明它正在等待批准以及如何批准；在回执存在之前，文件的描述会被保留，因此不受信任的检出无法将它自己的任何一行放入父级的系统提示词。已批准和用户作用域定义的描述在渲染前会被展平为一行。
- **`allow`** - 项目作用域定义的行为像用户作用域定义。

内置定义和用户作用域文件始终是 `trusted`。无论作用域如何，`permission_mode`、`tools` 和 `disallowed_tools` 只会收窄父级的能力，`spawns` 对项目作用域被忽略，并且定义自身不携带钩子、命令或 MCP 声明，因此其中没有任何内容会在加载时启动进程。

回执位于 **`~/.coddy/subagents-trust.json`**（`<home>/subagents-trust.json`），它是 `mcp-trust.json` 的兄弟文件而不是同一个文件，因此 MCP 批准永远不会被读取为智能体批准，反之亦然：

```json
{
  "version": 1,
  "workspaces": {
    "/home/me/project": [
      { "name": "reviewer", "digest": "3f2a…", "path": "/home/me/project/.coddy/agents/reviewer.md", "approved_at": "2026-09-03T10:15:00Z" }
    ]
  }
}
```

重写已批准文件会改变其摘要，回执将不再匹配，因此在下一次生成时会被拒绝，直到该内容被批准；项目的另一个检出批准的版本也算已批准。批准一个名称会替换同一检出中该名称先前的回执，并保留项目其他检出的回执。存储会在每次决策时重新读取文件，因此从终端授予的批准无需重启即可到达正在运行的智能体的下一次生成。

批准界面：

- **CLI**：`coddy agents list [--cwd DIR]` 打印工作区（在链接的 git worktree 中还有为其保留批准的主检出）、有效策略和带有作用域、信任状态和标记的目录，随后在项目定义等待批准时给出提示；`coddy agents trust <name> [--cwd DIR]` 先打印有效声明（文件、模型、模式、权限模式、工具列表、摘要、回执路径），然后为磁盘上当前的文件记录回执；`coddy agents untrust <name> [--cwd DIR]` 撤回它。内置或用户作用域名称无需批准，命令会说明这一点。`--cwd` 默认为进程工作目录，按 `coddy mcp` 解析。
- **HTTP**（`coddy serve`）：`GET /coddy/subagents?cwd=<dir>` 返回目录；`POST /coddy/subagents/{name}/trust` 和 `POST /coddy/subagents/{name}/untrust` 以正文 `{"cwd": "<dir>"}` 写入和移除回执。`cwd` 必须是绝对路径，并默认为服务器自己的工作目录；信任正文可以添加目录显示的 `digest`，而此后被重写的文件会被以 409 拒绝，而不是在未看见的情况下批准。目录行携带 `scope`、`trust`（`trusted` / `needs_approval`）、布尔值 `trusted` 和 `needs_approval`、`digest`、`path`、`builtin`、`hidden`，以及定义声明的边界（`tools`、`disallowed_tools`、`permission_mode`、`timeout_seconds`、`max_turns`、`background`、`spawns` 和 `role_bytes`，即其角色正文的大小），以便客户端可以显示定义声明的内容。文件未声明的边界会缺席，这意味着继承；角色正文本身永远不会被提供。错误是 `{"error":{"message"}}` JSON。详情见 `docs/reference/http-api.md`。
- Web UI 中的**设置 → 子智能体**列出定义并批准它们。该标签页以 `subagents.enable` 开关开始，然后是**定义目录**和 `subagents` 配置表单的其余部分，位于其自己的**子智能体设置**块中，它们下面是屏幕上会话工作区的目录（`spawn_agent` 针对会话自己的 cwd 解析定义，因此列出的就是该工作区）。每个定义显示其名称、作用域徽章、纯文本描述及其文件；名称前的折叠箭头会展开定义声明的内容：模型、模式、权限模式、工具列表、超时、轮次上限、是否始终分离运行，以及其指令的大小（文件省略的边界读作继承）。在 `ask` 下，项目定义携带 MCP 标签页的盾牌：点击它会为行显示的文件记录回执（其摘要随请求发送，因此期间被重写的文件会被拒绝），再次点击则撤回回执。仍在等待回执的定义还会携带琥珀色**需要批准**徽章，其工具提示会说明盾牌和 `coddy agents trust <name>`。
- **策略**：你已经信任的检出可以通过设置 `subagents.project_trust: allow`，在 `config.yaml` 中，于 Web UI 的**设置 → 子智能体**下（`subagents` 配置章节：策略和池边界），或通过捆绑的 `configure-coddy` 技能来运行其定义而无需回执，该技能记录该键，以便智能体可以暂存 `set subagents.project_trust=allow` 并通过普通的权限门控配置提交来提交它。策略在表单保存时应用。

![设置，子智能体标签页：会话工作区的定义，一个等待批准并带有其盾牌](../../assets/subagents/settings-subagents-catalog-dark-1280.png)

*设置，子智能体标签页：会话工作区的定义，一个等待批准，带有批准它的盾牌*

## `spawn_agent` 工具

| 参数 | 含义 |
|---|---|
| `agent` | 定义名称（必需）。 |
| `prompt` | 任务，自包含：目标、相关路径、约束，以及报告必须包含的内容（必需，最多 32 KiB）。 |
| `description` | 三到五个词命名任务；成为任务标签和子会话标题。 |
| `background` | 立即返回任务 id，而不是等待报告。默认 `false`。带有 `background: true` 的定义会强制开启。 |
| `expected_seconds` | 模型自己的估计；驱动状态指示器，并且在未给出超时时驱动硬超时 - 与后台化的 `run_command` 相同的建议语义。 |
| `timeout_seconds` | 运行的硬限制。 |
| `model` | 为子级配置的模型 id，覆盖定义的 `model` 和父级的模型。配置不认识的 id 会被拒绝，并列出已配置的 id。 |
| `reasoning` | 子级的推理级别：其模型提供的级别，`off` 或 `default`；覆盖定义的 `reasoning`。模型不提供的级别会被拒绝。 |
| `notify_on_finish` | 对于后台运行：子级完成时用结果唤醒父级，默认在有唤醒器可用时；显式设置 `false` 以禁用它（[后台任务](background-tasks.md)）。前台生成会被强制**关闭**，其报告已经通过工具结果返回；子级进行的任何生成也一样。 |
| `isolation` | `worktree`：子级在自己的 git worktree 中工作（[自己的 worktree](#自己的-worktree)），无论其定义怎么说。没有它时，由定义的 `isolation` 决定。 |
| `resume` | 继续此会话的一个已完成运行，而不是启动新的子级：其任务 id（`bg_…`）或来自较早结果的子会话 id（`sess_…`）。子级保留其对话记录，并将 `prompt` 作为下一条消息；`agent` 必须指定同一个子智能体（[恢复运行](#恢复运行)）。 |

该工具在 `subagents.enable` 开启时注册，并在 `agent` 和 `plan` 模式下提供，绝不在 `ask` 模式下提供。它需要**自己的权限提示**：启动子级本身不会改变你文件中的任何内容（隔离的子级会获得自己的 worktree 和分支，如果它没有在那里留下任何内容，这些就会消失），子级进行的每个工具调用都会自行门控，项目信任在运行时钩子内、任何内容启动之前决定。

**前台**生成（默认）会阻塞工具调用，直到子级的轮次结束，并在信封中返回报告：

```
<subagent task="bg_3" session="sess_9f1c…" agent="explore" status="succeeded" turns="4">
<![CDATA[
…the child's last assistant message…
]]>
</subagent>
The user did not see this report: restate what matters in your own reply. The full transcript is session sess_9f1c… (Tasks panel → Show transcript).
```

`status` 是任务池对该任务的裁决（`succeeded`、`failed`、`timed_out`、`stopped`）；当它不是 `succeeded` 时，会有一行说明，并告诉模型相应地对待报告，以错误结束的运行会指明错误。达到 `max_turns` 或未产生最终消息的子级会被标记为 `failed` 并附上该原因，即使其轮次返回时没有错误；它在到达 `max_turns` 途中写的文本会作为报告返回，原因会说明它不是结论。`turns` 是此次运行写入子级对话记录的助手轮数（恢复运行只计算自己的轮数）。报告被包裹在 CDATA 中，因此子级写的任何内容都不能破坏信封。

**后台**生成会立即返回：

```
Started subagent explore as background task bg_3 (child session sess_9f1c…).
Hard timeout 30m.
You will be woken with the outcome when it finishes, so you can end your turn now.
```

最后一行是在有东西可以唤醒父级时的默认值。`coddy -p` 不运行唤醒器，子级的对话记录会随其轮次关闭，因此那里的行会说明没有东西会唤醒模型，并且任务不会记录唤醒（[后台任务](background-tasks.md#哪个进程唤醒智能体)）。显式 `notify_on_finish: false` 也会让任务保持安静。从这里开始，运行是普通任务：`background_list` 显示它，`background_output` 流式传输子级的进度日志，`background_wait` 为其阻塞并返回以报告块结尾的日志，`background_stop` 取消子级。收集已完成的报告或停止任务会防止冗余唤醒。

拒绝会作为工具错误返回，并指明适用的旋钮：未知名称（附可见定义列表）、名称超出生成定义的 `spawns` 允许列表（会指明该列表）、没有回执的项目文件（附批准命令）、达到 `subagents.max_depth`、提示词超过 32 KiB、已有 `subagents.max_concurrent` 个运行在途、任务池自身的每会话限制（`tools.background.max_concurrent`），以及任务池正在为关闭而排空。使用 `subagents.enable: false` 时，该工具根本不会注册。没有会话管理器的界面永远不会被告知该工具，而任何调用都会回答此会话中不可用子智能体。

## 能力如何收窄

子级可以做的一切都在生成时从父级派生，并且只能收缩。

- **模式。** 只读父级（`plan`、`ask`）始终产生处于其自身模式的子级。否则，定义的 `mode` 适用，为空时则使用父级的模式。父级模式是生成轮次被接纳时的模式：在轮次运行期间发生的 `session/set_mode` 既不会改变子级的模式，也不会改变与子级求交的父级工具集。
- **权限模式绝不会拓宽。** 子级以父级有效模式和定义请求中更严格者运行，在 `ask` < `accept_edits` < `bypass` 的尺度上。在 `ask` 父级下请求 `bypass` 的定义会得到 `ask`；空请求则继承。
- **工具集交集。** 子级的有效集在生成时计算一次：父级自己的可调用集（其模式集，以及父级自身是子级时的其自身有效集），与子级模式的集相交，与定义的允许列表相交，减去定义的拒绝列表，减去强制排除项。MCP 工具名是集合的一部分，并按精确名称或 `prefix*` 匹配，因此子级只有当父级拥有某个 MCP 服务器且定义允许它时才会看到它。模式的集包含该模式提供的 MCP 工具：计划模式下的子级保留其定义允许的父级 MCP 工具，询问模式下的子级没有这些（[运行模式](modes.md#plan)）。
- **模型的工具集。** 子级运行所用的模型（父级的、定义的 `model`，或调用指定的）可以在其 `models[]` 行上指定工具集（[工具集](../getting-started/configuration.md#工具集)）。它在子级运行时进一步收窄有效集：其模型获得交集，超出交集的调用会在权限闸门之前被拒绝。父级自己的工具集，由其模型选择或指定，不属于上述交集，因为它限制的是向父级显示的内容，而不是子级是什么。如果某次生成的模型工具集让子级没有工具，该生成会被拒绝，并给出指明模型和工具集的消息，计划运行也受同一规则约束。
- **强制排除项。** `question`（子级不能询问用户；`RequestQuestion` 始终拒绝）、`config_set`、`config_changes`、`config_commit`、`config_revert`、`config_rollback`（子级不能重写智能体自己的配置）、`plan_exit`（子级不能代表操作者离开计划模式）、`switch_model`（子级在其被给定的模型上运行）、`worktree_open` 和 `worktree_exit`（子级停留在其启动的文件夹中），以及对处于深度限制的子级使用 `spawn_agent`。
- **强制执行两次。** 向子级模型通告的定义会过滤到该集合，并且每个调用都会在权限闸门运行之前再次检查，包括 MCP 调用：幻觉或重放的 `run_command` 会以 `tool run_command is not available to this subagent` 回答，并且永远不会执行。
- **深度。** `subagents.max_depth` 限制嵌套。默认的 `1` 允许会话生成无法继续生成的子级；在限制处的子级不会获得该工具，超过限制的生成会被拒绝。显式的 `0` 在所有地方禁止生成。存在一个有界例外：声明 `spawns` 的定义会在上限处保留 `spawn_agent`，并且可以恰好越过上限一层委派给列表上的名称 - `spawns` 允许列表会在每个深度限制名称，而深度超过上限的子级永远不会获得该工具，因此该例外只换来恰好一代额外深度（捆绑的 `crossreview` 编排器就是它存在的原因）。
- **模型。** 子级继承父会话正在运行的模型，除非定义的 `model` 指定了已配置的 `models[].model` id；未知 id 会保留父级的模型，并且回退会同时记录在智能体日志和任务的输出日志中。
- **轮次。** 定义的 `max_turns`，否则 `subagents.max_turns`，否则 `agent.max_turns`。
- **MCP。** 只有当子级的有效集可以包含 MCP 名称时，才会为它拨号服务器。已配置服务器会像新会话一样，通过工作区信任闸门为子级的 cwd 重新解析，父级由 ACP 客户端提供的服务器会根据父级保留的声明重新拨号。子级拥有并关闭自己的客户端；不从父级借用任何内容，因此父级重载或遗忘不会在运行中途切断子级。

## 自己的 worktree

除非隔离另有说明，子级在父级的文件夹中工作，因此其编辑会落在父级编辑旁边，也会落在每个其他并行运行的子级编辑旁边。在其定义中带有 `isolation: worktree`，或在 `spawn_agent` 调用中带有 `"isolation":"worktree"` 时，子级会在自己的 git worktree 中工作：

- worktree 在主检出的 `.coddy/worktrees/` 下创建，位于新分支 `subagent/<name>-<id>` 上（子会话 id 的末尾），基于父级检出所在的提交，无需 fetch，也无需 `origin`。父级未提交的更改不在那里。主检出的 `.worktreeinclude` 指定的文件会被复制进去（[git 不携带的文件](worktrees.md#git-不携带的文件)）；
- 在其检出的子文件夹中工作的父级会把 worktree 的同一子文件夹交给子级。子级的规则、技能和 MCP 服务器都是该文件夹的，主检出批准的内容在那里也成立（[worktree 与主检出共享什么](worktrees.md#worktree-与其主检出共享的内容)）；
- 运行结束时，会先停止它自己的后台任务，然后如果子级没有在那里留下任何内容，就会连同其分支一起移除 worktree：没有未提交更改（编辑过的 `.worktreeinclude` 副本也算），完全没有提交，即使有推送或另一个分支持有它，也没有其他会话或后台任务在其中工作。有工作内容的 worktree 会保留。报告会指明 worktree 和分支，并说明它们是否保留，以便父级审查那里的更改、合并分支或用 `worktree_open` 打开它；
- 智能体任务携带 `worktree` 和 `branch`（`GET /coddy/sessions/{id}/background-tasks`），后台生成会在其第一个答案中指明它们；
- 恢复的运行会在其较早运行工作过的 worktree 中继续，位于子级留下的任意分支上，如果该运行的结束移除了它，则会在同一分支上再次检出。在父级文件夹中工作过的运行会留在那里：在其恢复时请求隔离会被拒绝。

隔离需要一个至少有一次提交的 git 仓库。在其他任何地方，生成都会在任何内容启动之前被拒绝，并且不会留下任务、限制器槽位或子会话。worktree 属于仓库，而不是父级的文件夹，因此移动到同一仓库另一个 worktree 的父级仍然可以恢复子级。

## 子智能体的权限提示

处于 `ask` 或 `accept_edits` 模式的子级仍然会碰到权限闸门，而子级没有自己的客户端。因此每次生成都会创建一个**中继**，它拥有父级的发送者和进行生成的父级轮次（工具调用上下文），以及子级自己的取消：

- 在生成轮次存活期间，请求会以**父级的会话 id** 和标题 `[subagent <name>] Run: <tool>` 转发到父级的客户端，因此 ACP 编辑器、控制台模态框、Web UI 或 Telegram 聊天会在父级对话中显示它。等待也会监视子级，因此被停止的子级会以拒绝解除阻塞；在提示仍在屏幕上时结束的父级轮次会撤回该副本，并将提示交给下面的代理，因此它会在对话仍被读取的地方被回答一次；
- 一旦生成轮次返回 - **分离**运行的正常状态 - 请求永远不会触碰已完成轮次的传输：它会前往界面的**分离权限代理**（见下文），而在没有代理存在时，它会立即被拒绝，并给出子级报告的原因，而不是声称用户说不行（`permission not granted: this subagent is running detached ...`）；
- **每个父会话最多一个提示在途**：仲裁器会串行化该父级的每个子级的请求，这也是 HTTP 待处理权限记录（每个会话一个）所能表示的内容。两个子级同时询问会一个接一个地提示；
- 由后续轮次生成的子级携带该轮次的上下文，因此较早的分离子级的已完成轮次不会拒绝它。

操作者自己的设置如何与之结合。处于 `bypass` 的父会话会让父级以及每个**继承**该模式的子级静默，除了钩子的 `ask` 和[决策检查](decisions.md)的提示。收窄到 `ask` 或 `accept_edits` 的定义是另一种情况：中继会在每个转发的请求上盖上子级自己的有效模式，而每个可以自动允许的发送者（ACP 服务器、HTTP 桥、控制台、打印模式、Telegram 网关）都会根据该标记而不是父级的会话来决定。这样的子级会在有人能回答的地方被提示（SPA、控制台、`--remote` 控制台或 ACP 客户端、本地 ACP 编辑器、Telegram 聊天，后者用按钮询问），并在无人能回答的地方被拒绝（打印模式、非流式 HTTP 轮次）。标记在向上传递时只会变得更严格：孙级的提示会穿过两个中继，中间子级不能重新拓宽它，而 `PreToolUse` 钩子用 `ask` 强制的提示即使在一个 `bypass` 子级中也会保留该 `ask`。转发的提示只提供**允许**和**拒绝**；“始终”答案会被保留，因为子级的一次运行是一个轮次，常设授权无法比它活得更久。转发的提示要么实时回答，要么根本不回答：HTTP 桥不会将它们写入父级的待处理权限记录，因此页面重载或重新连接无法恢复一个提示（子级自己的超时或父级轮次的结束会撤回它），而父级自己的闸门会为自己保留该记录。

### 分离运行：提示比轮次活得更久

`background: true` 是对一个比启动它的轮次活得更久的运行的请求，因此闸门也必须比它活得更久。当生成轮次消失后，中继会将提示交给**代理**（`agent.DetachedPermissionBroker`，与 `SubagentRuntime` 相同的接缝形状）并在其上阻塞。就像 Claude Code 中的后台子智能体一样，提示会出现在人们正在阅读的对话中，并指明提问的子智能体；**拒绝**会拒绝那一个调用，子智能体会继续：

| 界面 | 分离子级的提示出现在哪里 |
|---|---|
| Web UI（带 HTTP 界面的 `coddy serve`） | 在父会话聊天的末尾，与内联提示相同的卡片中，子智能体在卡片头部被指明 |
| 控制台 | 权限模态框，标题为 `[subagent <name>] ...`，轮次之间也会出现；当另一个模态框打开时到达的提示会等待它 |
| 带 `--remote` 的控制台 | 同一模态框，针对此控制台打开的会话，由服务器的事件流宣告 |
| Telegram（带网关的 `coddy serve`） | 在拥有父会话的聊天中显示一条带有**允许**和**拒绝**按钮的消息 |
| Pachca（带网关的 `coddy serve`） | 在拥有父会话的聊天中显示同一条带有**允许**和**拒绝**按钮的消息；只有机器人的管理员可以回答 |
| `coddy acp`、打印模式、计划运行 | 无处：子级会立即被拒绝，并给出上述原因并报告它 |

- **`coddy serve` 会同时向每个界面提供提示。** 共享会话管理器的运行器会将运行时交给每个轮次作为其代理，每个界面在出现时提供自己（`serve.Runtime.AddDetachedPermissionApprover`）：HTTP 服务器，以及针对作为其聊天的会话的 Telegram 或 Pachca 机器人。第一个答案会解决提示，其他所有副本都会被撤回：Web 卡片离开聊天，远程控制台的模态框关闭，Telegram 消息显示 *No longer waiting*。无法显示提示的界面会置身事外；当没有任何界面可以时，子级会以其原因被拒绝。
- **当轮次结束时仍在父级屏幕上的提示会以同样方式移动。** 后台子级通常会在父级仍在写回复时开始询问，而答案可能不会在该回复结束前到来；携带提示的流会随轮次消失，因此中继会撤回父级客户端显示的副本 - 控制台取下其模态框，HTTP 桥停止等待，Telegram 消息显示 *No longer waiting* - 并通过代理提出提示，在那里它作为分离提示被回答一次。只有在没有代理存在时，子级才会被拒绝，并给出上述原因。在 Web UI 中，已完成轮次的内联卡片会随其流一起退役，而聊天末尾的卡片才是回答它的那个。
- **Web UI** 从被阻塞运行的后台任务行读取提示（`GET /coddy/sessions/{id}/background-tasks` 上的 `pending_permission`），并在 `GET /coddy/events` 的 `event: subagent_permission` 上重新读取这些行（阶段 `asked` 和 `settled`），任务轮询作为回退。答案发往 `POST /coddy/sessions/{child}/permission`。没有新路由：权限中心以 `(session id, tool call id)` 而不是活动轮次为键，处理程序在保持子级对话记录只读的防护之前查询它。通过 `--remote` 连接的控制台读取相同事件（连接时快照会重复仍在等待的提示）并发布到同一路由。
- **Telegram** 会无需询问地允许聊天自己的智能体请求，但会就子智能体的请求 - 在轮次期间和之后 - 用内联按钮询问。只有会话提问的那个人可以回答：在具有单独会话的群组中，另一个成员的点击会被忽略。
- 提示**不会**占用父级的仲裁器槽位。该槽位存在是因为父级持有单个待处理记录，而分离提示以其自己的子会话为键，可以等待数分钟；兄弟的活动提示不得排在其后。“始终”答案在这里也会被保留。
- 不会持久化任何内容。等待者是一个 goroutine，因此提示不能比提出它的进程活得更久；重启后运行已消失，抽屉会将其任务显示为孤立。
- 等待以运行自己的上下文结束，任务池的超时、显式停止和关闭都会取消它。等待中的运行仍然计入 `subagents.max_concurrent` 和父级的 `tools.background.max_concurrent`，这是它正在做什么的真相，而无人回答的提示会保持运行直到其硬超时。

![后台子智能体在其父聊天末尾请求权限](../../assets/subagents/chat-subagent-permission-dark-1280.png)

*后台子智能体在其父聊天末尾请求权限*

## 并发与超时

一次生成适用两个上限，每个拒绝都会指明自己的键：

- **`subagents.max_concurrent`**（默认 4）是**进程级**的：可以同时运行的子级 LLM 循环数量，无论由哪个会话启动。它是一个可调整大小的信号量，会拒绝而不是排队，符合任务池的“拒绝，不排队”契约；消息告诉模型用 `background_wait` 或 `background_list` 等待并重试。在负载下降低上限永远不会停止正在运行的子级；新生成会被拒绝，直到计数降到新限制以下。该值在每次生成时重新读取，因此配置提交无需重启即可生效。
- **`tools.background.max_concurrent`** 是任务池的**每会话**任务限制，子级的任务注册在**父级的**会话下，因此子智能体运行会像其后台命令一样计入父级的任务。因该限制已满而被拒绝的生成会指明 `tools.background.max_concurrent`，以便模型区分两个上限。

单次运行的硬限制按以下顺序解析：调用的 `timeout_seconds`，然后是定义的 `timeout_seconds`，然后是 `expected_seconds × 3` 下限为 60 秒，然后是 `subagents.default_timeout_seconds`（1800）。结果受 `tools.background.max_timeout_seconds` 限制，与后台命令完全一样。达到它会取消子级并将任务记录为 `timed_out`，这是失败，不是成功。

## 提供商连接断开时

子智能体无人值守地工作：运行时没有人阅读其对话记录，也没有人能在其中输入“继续”。其模型提供商通道的失败（远程主机关闭的连接、重置、代理的 5xx、流变得沉默）是通道的失败，不是任务的失败，因此子级会像交互轮次那样挺过去（[提供商在轮次中途失败](../getting-started/troubleshooting.md#提供商在轮次中途失败)），只是时间更长。它已经写下的文本会留在其对话记录中，暂停后该步骤会再次运行，并要求模型从停止处继续。交互轮次会连续挺过两次失败调用，并在第三次时结束，因为阅读它的人决定接下来发生什么。子智能体和计划运行会连续挺过五次，并在第六次时结束，在默认 `agent.llm_retry_base_ms` 下暂停 5 秒、20 秒、80 秒、2 分钟和 2 分钟：当运行有余地时大约六分钟。每次恢复会占用运行的一个轮次（`max_turns`），而运行的硬超时会缩短暂停并结束运行为 `timed_out`，因此超时短或轮次限制低的运行会更早放弃。成功的调用会重新开始计数。记忆子智能体保留交互预算，因为模型委派给它并作为工具结果等待其报告，因此它会像交互轮次一样挺过同样的两次失败；`agent.llm_retry_max: 0` 会在所有地方关闭恢复。

Windows 使用 Winsock 自己的代码报告死连接，`WSAECONNRESET`（`wsarecv: An existing connection was forcibly closed by the remote host.`）和 `WSAECONNABORTED`（`An established connection was aborted by the software in your host machine.`）。两者都算作连接已死，就像 Linux 上的 `connection reset by peer`（[issue #389](https://github.com/coddy-project/coddy-agent/issues/389)）。

每次重连都是任务进度日志的一行，位于子级已写文本之后，因此等待中断结束的运行不会读起来像挂起：

```
subagent general (task bg_5, session sess_e20a…) starting
[assistant] Reading the files
↻ provider failed (provider "cockpit" (http://localhost:62557/v1): openai stream: read tcp 127.0.0.1:53107->127.0.0.1:62557: wsarecv: An existing connection was forcibly closed by the remote host.); reconnecting in 5s (attempt 1 of 5)
[assistant] …the rest of the answer…
```

### 恢复运行

提供商停机时间超过上述时间、运行的超时或其轮次限制会在没有报告的情况下结束运行，其对话记录会保留它做过的一切。父级不会从空上下文对同一任务启动第二个子智能体：将 `resume` 设置为运行的 task id（或其子会话 id）的 `spawn_agent` 会继续同一个子级。子会话从其包重新打开，`prompt` 成为其在较早对话记录之后的下一条消息，并且该运行是同一子会话上父级的新任务（新的 `bg_…` id）：

```json
{"agent": "general", "resume": "bg_5", "prompt": "The provider is back. Go on from where you stopped."}
```

- 该运行像新生成一样，根据当前定义来决定：其信任回执、模式、权限模式、工具集、推理级别和超时。`agent` 必须指定该运行所属的子智能体。子级保留其标题，模型保留其较早运行工作过的模型，除非调用指定另一个；此后从配置中移除的模型会像未知模型一样被替换，任务日志会同时指明两者。较早运行被允许的任何内容都不会延续：其答案记录的授权会留在后面，恢复的运行会再次询问。
- 对话记录按其离开时的状态继续，并做一处修复：它持有但没有结果的调用（运行在一批调用中途被停止或超时，或进程在其中一个调用期间死亡）会被回答为没有记录结果，它可能运行过，也可能没有，因此模型在再次进行调用之前会检查当前状态。提供商会拒绝携带没有结果的调用的请求，因此没有该答案，恢复的运行会在第一步失败。
- 对话记录不在磁盘上的运行会在任何内容启动之前被拒绝：其子会话从未创建（运行在途中失败），或子级此后被删除。
- 子级只在其工作过的工作区中继续。会话移动到另一个文件夹后，恢复会被拒绝，因为对话记录的路径和结果属于旧工作区；在那里为该任务启动新的子智能体。拥有自己 worktree 的子级会在其中继续，因此对于该子级，会话只需要在同一仓库中工作（[自己的 worktree](#自己的-worktree)）。
- 只有此会话的已完成运行会被恢复。仍在途的运行会被拒绝（先用 `background_wait` 等待它或用 `background_stop` 停止它），任何不是此会话已完成子级的内容也会被拒绝。计划运行是其 job 会话的子级，绝不是聊天的子级。
- 报告、轮数计数和信封属于恢复的运行：较早运行给出的答案不会被当作没有写任何内容的运行的报告。任务日志以 `subagent <name> (task bg_…, session sess_…) resuming` 开头，后台恢复会回答 `Resumed subagent <name> as background task bg_…`。
- 未成功的运行会在父级阅读的地方说明如何继续。前台信封和任务日志的报告块携带 `call spawn_agent with resume="bg_5" and agent "general" …`，如果发生了这种情况，则位于 `The run was cut by a failure of the model provider, not by its task.` 之后，失败后台运行唤醒的轮次会指明同一调用。父级系统提示词中的目录也会说明。

界面仍然不能向子级发送消息：除生成它的父级外，对话记录对所有人保持只读（[子会话](#子会话)）。

## 子会话

生成会创建一个带有普通会话 id 的子会话，该 id 在任务池参与之前生成，因此第一个任务快照已经携带它；恢复的运行会重新打开它所继续运行的子会话（[恢复运行](#恢复运行)）。关于该 id 的任何内容都不说明会话是被委派的 - 它与聊天获得的是同一个 **`sess_<24 hex>`** - 标记它的是包所在的位置及其元数据所说的内容。子级是一个**位于父级内部**的真实会话包，位于 **`<sessions root>/<parent>/subagents/<child>/`**，以 `session/new` 构建会话的相同方式构建（技能、规则目录、持久化），其标题固定为任务标签，其 `session.json` 携带 `subagentRun: true`、`parentSessionId`、`subagentName`、`subagentTaskId` 和 `subagentDepth`。生成自己子级的子级会再嵌套一层。子级运行时被注册为活动会话，因此对话记录读取会从活动状态提供；完成后，包会像任何已关闭会话一样提供对话记录。

- **从历史中隐藏。** 子会话不进入每个默认列表：Web UI 历史、`GET /coddy/sessions`、`coddy sessions list`、`coddy -c` 和 ACP `session/list`。`GET /coddy/sessions?include_subagents=true` 会包括它们。
- **只读对话记录。** 界面不能向子级发送消息；只有生成它的父级可以用 `spawn_agent` `resume` 继续它（[恢复运行](#恢复运行)）。任何针对子会话且不是子级自己任务轮次的提示（输入框 `POST /v1/responses`、ACP `session/prompt`、控制台提示、运行计划请求、后台唤醒器）都会以 `subagent sessions are read-only transcripts: <child> belongs to <parent>` 被拒绝；通过 HTTP 则是 **409**，并指明父级。SPA 渲染子级的对话记录时，输入框会被替换为链接回父聊天的通知。
- **从任务面板。** 智能体任务的卡片以智能体名称作为标签，以运行的描述作为标题，其元信息行指明子级运行的模型及其调用迄今花费的 token - 输入和输出合计，在提供商报告之前对在途调用进行估算 - 因此消耗大量上下文的运行会在运行时显示出来。`GET /coddy/sessions/{id}/background-tasks` 的行以 `agent.model`、`agent.input_tokens` 和 `agent.output_tokens` 携带相同数字，任务记录会保留最终数字。无论折叠还是打开，它都显示**显示对话记录**，路由到 `#/s/<child id>`；聊天中 `spawn_agent` 调用的卡片不会重复它。卡片输出框中的实时进度日志在运行时就是报告；对话记录显示子级的工具调用及其最终答案。
- **删除级联。** `DELETE /coddy/sessions/{id}` 通过一条路径移除整棵树：请求的会话加上每个后代，通过遍历其包的 `subagents/` 文件夹从根到叶找到。首先停止并等待每个节点的代表任务（子级的任务位于其父级之下，因此这会同时到达正在运行的子级和正在运行的后代），然后是每个节点的任何剩余任务，只有之后才会最深优先移除包，请求的会话最后。在此之前，任何节点的活动轮次都会被取消并等待，删除期间到达的轮次会被拒绝；忽略取消并超过稳定超时（15 秒）的轮次会中止删除且不移除任何内容（HTTP `409`）。树在标记后会重新扫描，直到没有新后代出现，因此删除开始时创建的子级会随它一起移除，而不是成为孤儿。此后没有任何内容写入已移除的包。

## 计划运行

调度器启动的运行（[调度器](../operate/scheduler.md)）是同一类子级：job 会话下 kind 为 `agent` 的任务、带对话记录的子会话、相同的进度日志、相同的退役和相同的只读规则。不同之处是父级 - 一个从不运行轮次的 job 会话，而不是聊天 - 以及告知子级的内容：其角色块说明它是一个无人值守启动的计划 job，其 `session.json` 携带 `schedulerJobId` 和 `schedulerTrigger`，对话记录的通知链接回 job 的运行而不是父聊天。job 可以指定一个定义（其 frontmatter 中的 `agent:`）以在其角色、工具、模型、推理级别和权限收窄下运行，具有与生成相同的信任规则和相同的级别规则（当运行的模型提供时应用）；没有定义时，运行拥有其模式的完整工具集。它位于深度 0，因此在 `subagents.max_depth` 内它可以生成自己的子级，这些子级像任何生成一样计入 `subagents.max_concurrent`。

## 生命周期规则

- **每次运行一个轮次。** 运行恰好是子级通过会话管理器正常路径的一个提示词轮次，因此它会获取轮次锁、报告活动边、遵守跨进程取消，并像任何会话一样持久化。恢复的运行是同一子会话的又一个轮次，从其包重新打开，并在结束时再次退役（[恢复运行](#恢复运行)）。
- **退役会先结算子级自己的工作。** `general` 子级继承后台化的 `run_command`，并在深度限制之下继承 `spawn_agent`，因此它可以在轮次返回时留下正在运行的命令或分离的孙级。退役的子级是只读对话记录，没有剩余轮次来收集它们，因此在活动条目被丢弃之前，子级拥有的每个任务都会被停止并等待（孙级的句柄会取消该孙级，其自身退役会以相同方式递归）。已完成的任务记录留在子级的包中。此结算所需的时间，受任务池每个任务 3 秒停止宽限的限制，计入子级自己的硬超时。然后关闭子级的 MCP 客户端并丢弃其活动条目；包留在磁盘上。
- **不会代表子级唤醒任何人。** 子级进行的生成永远不会携带 `notify_on_finish`，而针对子会话的唤醒轮次会被只读防护拒绝，因此退役前后完成的任务不会启动轮次。
- **父级轮次被取消（停止）。** 前台子级在从父级工具调用派生的上下文上运行，因此它会随父级一起被取消，工具结果会报告最终状态。分离子级在从工具调用分离的上下文上运行，因为该上下文在工具返回的那一刻结束；它按设计继续运行，并从任务面板或用 `background_stop` 停止。
- **关闭与排空。** 服务器排空会像命令一样停止智能体任务：每个句柄取消其子级，子级退役并释放其限制器槽位。任务池排空后不会启动任何新内容。
- **每条退出路径都会释放。** 任务池拒绝、创建子会话失败（记录为 `failed` 任务）、超时、停止、父级取消、运行中的 panic（被恢复并报告为 `failed`，附 panic 文本）以及正常完成都会经过同一次幂等完成，因此槽位永远不会泄漏，也永远不会被释放两次。
- **父级被遗忘或重载。** 子级有自己的状态和自己的 MCP 客户端，因此它所依赖的任何内容都不会在其下方被关闭。

## 任务行与输出日志

智能体任务行（`background_list`、`GET .../background-tasks`、持久化的 `meta.json`）携带 `kind: "agent"`、标签 `agent <name>: <description>`、父级的 `session_id`、生成调用的 `tool_call_id`（因此对话记录行保留其活动 chip），以及 `agent: {"name": "<name>", "session_id": "sess_…"}`。它不报告 pid：其背后的句柄不是操作系统进程，因此幸存者探测永远不会把它误认为进程。

子级永远不会写入父级的流。其进度作为紧凑日志行进入任务的输出接收器，这就是 `background_output` 和面板在运行时显示的内容：

```
subagent explore (task bg_3, session sess_9f1c…) starting
→ grep
✓ grep
[assistant] The runtime is wired in four places…
=== subagent report ===
agent: explore | task: bg_3 | session: sess_9f1c… | outcome: end_turn | turns: 4 | duration: 41s
--- report ---
…the child's last assistant message…
```

`✗ <tool> (failed)` 标记失败或被拒绝的工具调用，包括超出子级工具集的调用，`↻` 标记提供商失败后的重连（[提供商连接断开时](#提供商连接断开时)）。`outcome` 是子级的停止原因（`end_turn`、`cancelled`、`failed` 或其他 ACP 停止原因），`turns` 是此运行写入的助手轮数（与前台信封携带的计数相同），当运行以报告结束时，报告之前会有一行 `error:`。

## 记忆子智能体

**记忆子智能体**会召回并持久化长期笔记（[长期记忆](memory.md)）。它是一个普通的捆绑定义，由模型通过 `spawn_agent` 以 `agent: "memory"` 像任何其他委派一样生成：任务池中 kind 为 `agent` 的任务、父级包内的子会话、相同的对话记录和日志，`SubagentStart` 和 `SubagentStop` 触发，运行计入并发限制并对面向模型的任务池工具可见，其卡片按智能体名称标记为 `memory`。没有任何东西会自行启动它；当模型判断笔记值得读取或写入时，它会委派给它。有几点使它不同于手写定义：

- 除非二进制带有 `memory` 标记且 `memory.enable` 开启，否则加载器会省略它，因此模型永远不会看到闸门会拒绝的生成；
- 它有六个其父级没有的工具，这是子级集合只会收窄这一规则的唯一例外：该集合由记忆包在代码中固定，并且只授予此子级，绝不通过定义文件授予，而且它只能触及两个笔记根。其模式始终是 `agent`；询问模式轮次会将其收窄为三个召回工具；
- 其模型和回退链来自 `memory.model` 和 `memory.fallback_models`，而不是定义的 `model`，其任务在生成调用的提示词之前携带记忆根的 `AGENTS.md` 指南；
- 它拒绝 `isolation: worktree`，因为两个笔记根属于会话的工作区，而自己的 worktree 会让它们无法触及。

## 远程模式

子智能体位于会话管理器所在之处。当控制台或 `coddy acp` 处于 `--remote` 模式（`docs/surfaces/console.md`、远程模式）时，管理器、子会话、任务池任务和信任回执都在 `coddy serve` 主机上：

- 定义从**服务器**的文件夹读取（服务器用户的 `~/.agents/agents`、服务器主目录的 `${CODDY_HOME}/agents`、服务器上会话 cwd 的 `.agents/agents` 和 `.coddy/agents`，然后是它的 `subagents.dirs`）；
- 项目定义**在服务器上**批准：在该主机上运行 `coddy agents trust <name> --cwd <workspace>`，或使用 bearer token 运行 `POST /coddy/subagents/{name}/trust`。本地 `coddy agents` 子命令只读写本地主目录，对 `--remote` 一无所知；
- 子级的权限提示像父级的一样传输：中继在父会话下转发它们，HTTP 桥发出 `permission` SSE 事件（即使服务器上的父会话以 `bypass` 运行，因为子级自己的模式才是决定因素），远程控制台或 ACP 客户端显示带有 `[subagent <name>]` 前缀的提示，并通过 `POST /coddy/sessions/{parent}/permission` 回答；
- 在生成轮次结束后才询问的**分离**子级也会到达远程控制台：服务器在 `GET /coddy/events` 上宣告提示，控制台为其打开的会话打开模态框，并用 `POST /coddy/sessions/{child}/permission` 回答。同一提示会在该主机提供的 SPA 的父聊天中等待，如果会话是 Telegram 聊天，也会在 Telegram 聊天中等待；第一个答案获胜。远程 ACP 客户端不会显示它，在有人回答之前，运行会等待直到其硬超时；
- `spawn_agent` 调用及其报告像任何工具调用一样流回，因此控制台状态行在远程模式下也会显示 `Running a subagent`（其上方的工具框指明是哪一个）；子级的对话记录留在服务器上，并从同一主机提供的 SPA（任务面板、显示对话记录）或从 `GET /coddy/sessions/{child id}/messages` 读取；
- `subagents.*` 设置是服务器的：在该主机的 SPA 设置中编辑它们，或从远程会话使用 `configure-coddy` 技能（配置工具在服务器上运行）；`max_concurrent` 适用于每个会话的下一次生成，无论哪个轮次保存了它；
- 回执以**服务器端**会话工作区为键，对于由远程控制台或 ACP 客户端创建的会话，这就是服务器的默认 cwd（`GET /coddy/sessions` 行的 `cwd` 字段，或带会话头的 `GET /coddy/workspace/context`）；远程控制台页脚显示本地文件夹，而不是该路径。一个可复制粘贴的批准：

  ```bash
  curl -sS -X POST "$CODDY_URL/coddy/subagents/marker-reporter/trust" \
    -H "Authorization: Bearer $CODDY_REMOTE_TOKEN" -H 'Content-Type: application/json' \
    -d '{"cwd": "/srv/checkouts/project"}'
  ```

- 连接断开会让服务器轮次、其前台子级和任何打开的提示在服务器端继续运行（轮次与请求分离）；远程控制台报告流错误，轮次结束后 `/resume` 显示结果，而在服务器撤回提示后回答的提示会被忽略，而不是使轮次失败；
- 唤醒轮次（分离生成或后台命令上的 `notify_on_finish`）在服务器上运行：在绑定到会话的 Telegram 聊天中，或在会话的输入框中继上，Web UI 和通过 `--remote` 跟随轮次的控制台可以回答其权限提示；两者都未启动时，其中的门控工具调用会被拒绝，除非服务器自己的权限模式是绕过（[后台任务](background-tasks.md#在-coddy-serve-下)）。

可执行检查是 `features/remote_client.feature` 中的场景 "A subagent's permission prompt reaches the remote client even when the server bypasses its own"、实时 `examples/acp/acp_e2e_remote_subagents.py` 以及 `examples/cli/cli_e2e_remote.py` 的子智能体步骤。

## 配置

所有旋钮都是 `subagents:` 下的普通 `config.yaml` 键；字段表在 `docs/reference/config.md`（章节 `subagents`），Web UI 在**设置 → 子智能体**下编辑它们，捆绑的 `configure-coddy` 技能可以通过暂存配置工具更改它们。

```yaml
subagents:
  enable: true
  dirs: []                    # extra folders after the four defaults, e.g. "${CWD}/.claude/agents"
  project_trust: ask          # ask | allow | deny
  max_concurrent: 4           # child runs in flight across the whole process
  max_depth: 1                # 1: children cannot spawn; 0: nobody spawns
  default_timeout_seconds: 1800
  max_turns: 0                # 0 follows agent.max_turns
```

`tools.background` 仍然限制运行所在的任务池：其每会话 `max_concurrent`、`max_timeout_seconds` 上限，以及进度日志的 `output_buffer_bytes`。

## 测试

- 快乐路径是由 godog 运行的 Gherkin 规范：`features/subagents.feature`（向模型提供的定义、前台和后台生成、父级流的隔离、深度、权限收窄、项目信任拒绝和回执、`deny`、由子级继承的客户端提供 MCP 服务器、权限中继和仲裁器、分离子级的提示到达代理或在没有代理时以原因被拒绝、只读子会话、子级自己任务的结算、并发上限；框架 `internal/agent/bdd_subagents_test.go`，在真实 `session.Manager` 上运行脚本化提供商）、`features/subagents_worktree.feature`（子级在自己的 worktree 中：当子级没有在那里留下任何内容时，worktree 连同其分支被移除；当它留下工作时，保留并在报告中指明；并且调用和定义都可以请求它）、`features/subagents_reconnect.feature`（远程主机强制关闭其连接的子级会重连并完成，子级比交互轮次等待更多失败调用，父级恢复因提供商中断而失败的子级，而不是生成第二个；相同的框架，使用会断开连接的子级提供商）、`features/subagents_detached_prompts.feature`（分离提示提供给 `coddy serve` 的每个界面，第一个答案在其他地方撤回它；`internal/serve/bdd_detached_prompts_test.go`）、`features/subagents_http.feature`（智能体任务行、隐藏和包含的子会话、活动和已完成的子对话记录、带有定义声明边界和信任路由的目录、在事件流上宣告并通过子会话回答的分离提示、树删除；`external/httpserver/bdd_subagents_test.go`、`-tags http`）、`features/subagents_web_ui.feature`（只读子智能体标签页、后台子智能体的提示在其父聊天中回答；`external/ui/bdd_subagents_ui_test.go` 运行 Vitest 场景，`-tags http,ui`）、`features/subagents_catalog.feature`（跨作用域和信任状态的 CLI 列表；`internal/subagents/bdd_catalog_test.go`）、`features/gateway_telegram_subagent_permission.feature`（子智能体在轮次期间和之后在 Telegram 聊天中询问；`external/gateway/telegram/bdd_subagent_permission_test.go`、`-tags gateway`），以及在 `features/cli_tui.feature` 中的 `Running a subagent` 状态行、后台子智能体在轮次后通过模态框询问，以及等待在另一个提示后面的提示（`external/cli/bdd_cli_tui_test.go`、`-tags cli`），附加控制台使用 `features/cli_remote.feature`（`external/cli/bdd_cli_remote_test.go`），以及 `features/subagents_crossreview.feature`（一个协调器定义通过其 `spawns` 允许列表越过深度上限一层委派，而允许列表拒绝其外的名称；相同的 `internal/agent` 框架）。
- 边缘情况是代码旁边的普通单元测试：frontmatter 别名和逗号分隔工具、`AGENT.md` 命名、重复项、超大和无效文件、优先级和规范作用域、摘要和回执存储、目录中的边界、权限收窄、工具集交集和 MCP 模式、限制器（`internal/subagents`）；`Launch` 排序和 `Snapshot.Agent` 持久化（`internal/bgtask`）；子会话元数据、只读防护、列表过滤、树删除以及恢复子级的重新打开及其拒绝（`internal/session`）；生成钩子的退出路径、超时、接收器、CDATA 信封、中继的分离路径和拒绝原因、重连预算及其日志行、恢复拒绝和恢复运行自己的报告、深度边界处、其下以及项目作用域下的 `spawns` 允许列表（`internal/agent`）；瞬时分类中的 Winsock 连接错误（`internal/llm`）；运行时的提供、其撤回和扇出（`internal/serve`）；发布的提示、其事件、重放和取消，以及目录路由的 JSON 错误（`external/httpserver`）；模态框队列（`external/cli/gates_test.go`）；附加控制台宣告的提示（`internal/remote/detached_prompts_test.go`）；来自错误人员的点击、撤回的请求和会话的聊天（`external/gateway/telegram`、`external/gateway/sessionstore`）；配置默认值和验证（`internal/config`）。在 SPA 中：`settings/subagentCatalog.test.ts`、`settings/subagentsApi.test.ts`、`settings/SubagentsSection.test.tsx`、`chat/SubagentPermissionCard.test.tsx`、`chat/ChatScreen.test.tsx`、`chat/serverEvents.test.ts` 和 `tasks/taskStatus.test.ts`。
- 针对真实模型的端到端：`examples/acp/acp_e2e_subagents.py`、`examples/httpserver/http_e2e_subagents.py` 和 `examples/cli/cli_e2e_subagents.py`，接入三个运行器。每个都会将 fixture `examples/agents_fixture/.coddy/agents/marker-reporter.md` 复制到工作区，预先批准它（CLI 和 ACP 运行使用 `coddy agents trust`，HTTP 使用 `POST /coddy/subagents/{name}/trust`），以便运行保持无人值守，写入一个标记文件，并要求模型委派读取它。它们断言父级包下的 `agent` 任务、父级内部带有父链接的子包、子对话记录中的标记、报告块中以及父级最终答案中的标记；HTTP 脚本还会驱动任务行、只读子对话记录、带和不带 `include_subagents` 的会话列表，以及每个定义声明边界的目录，然后运行第二个 fixture `echo-runner`（`permission_mode: ask`、`background: true`）：父级的流被读取到结尾而不回答，子级的提示作为 `pending_permission` 在任务行上找到，并由 `GET /coddy/events` 重放，用 `POST /coddy/sessions/{child}/permission` 回答，宣告为已解决，运行以命令的输出结束。

## 范围之外

后续事项，故意不属于此更改：向正在运行的子级发送消息、定义级钩子和 MCP 服务器、从 Web UI 编辑定义（定义仍然是你编写的文件；**设置 → 子智能体**、`coddy agents trust` 和 HTTP 路由只批准它）、任务池满时排队而不是拒绝，以及子级的实时 SSE 中继（对话记录改为从活动状态或包读取）。包含所考虑替代方案的设计记录是 `docs/plans/subagents.md`。

## 截图

子智能体标签页显示在[作用域与项目信任](#作用域与项目信任)下，后台子智能体的提示显示在其父聊天中的[分离运行](#分离运行提示比轮次活得更久)下。

![对话记录中的 spawn_agent 工具卡片](../../assets/subagents/spawn-agent-card-dark-1280.png)

*对话记录中的 spawn_agent 工具卡片*

![任务面板中有正在进行的子智能体运行](../../assets/subagents/tasks-panel-agent-running-dark.png)

*任务面板中有正在进行的子智能体运行*

![已完成运行的卡片就地打开：显示对话记录、日志和报告](../../assets/subagents/tasks-detail-agent-finished-dark.png)

*已完成运行的卡片就地打开：摘要上的显示对话记录、其下的进度日志和报告*

![从任务行以只读方式打开的子会话](../../assets/subagents/child-transcript-readonly-dark.png)

*从任务行以只读方式打开的子会话*

<!-- docsgen:source sha256=d1df667a693742ff -->

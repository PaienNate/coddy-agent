# 运行模式

会话以三种模式之一运行：`agent`、`plan` 或 `ask`。模式决定模型在某一轮次中会获得哪些工具，并且对于 `plan` 和 `ask`，还决定哪些调用可以运行。本页列出每种模式允许的内容、Plan 模式如何将其文档交给实现、Plan 和 Ask 模式在调用执行时如何强制执行，以及如何在每个界面上切换。模式与权限模式（`ask`、`accept_edits`、`bypass`）不同，后者决定智能体在命令或文件写入之前何时提示你；两者会结合使用。

## 三种模式

### `agent`

默认模式，用于实现、重构、调试和运行各类任务。模型会获得注册表中的所有内置工具：另外两种模式的读取工具，加上文件写入与编辑、shell 及其后台任务、待办清单、暂存配置工具、计划工具和 `spawn_agent`，以及每个已连接 MCP 服务器的工具。这些调用中哪些会先提示你，由权限模式决定，而不是运行模式。

### `plan`

用于设计文档、规格说明以及变更之前的调查。模型会获得一个固定的允许列表：

- 用于仓库的 `read`、`keep_result`、`glob`、`grep` 和 `print_tree`；
- 用于 Web 的 `websearch` 和 `webfetch`；
- `run_command`，搭配 `background_list`、`background_output`、`background_wait` 和 `background_stop`，这样无需等待慢速的只读命令；
- `question`，以及只读的 `config_get` 和 `config_changes`（暂存和提交配置更改仍留在 Agent 模式）；
- 用于计划文档的 `plan_write`、`plan_list` 和 `plan_read`；
- `load_skill`，以及 `spawn_agent`，其子智能体仍留在 Plan 模式；
- 用于 Coddy 自身文档的 `coddy_docs_search` 和 `coddy_docs_read`（[内置文档](built-in-docs.md)）；
- 已连接 MCP 服务器的工具。

没有内置文件写入，也没有待办工具：计划准备好后，实现在 Agent 模式中进行。当调用运行时，同一列表会再次强制执行（参见[执行时的 Plan 和 Ask 模式](#执行时的-plan-和-ask-模式)）。

### `ask`

用于关于代码库的问题、审查、诊断和 Web 研究。模型会获得 `read`、`keep_result`、`glob`、`grep`、`print_tree`、`websearch`、`webfetch`、`question`、`load_skill`、`coddy_docs_search` 和 `coddy_docs_read`，仅此而已：没有 shell，没有计划、待办或配置工具，没有 `spawn_agent`，没有 MCP 工具。当调用运行时，同一列表会再次强制执行，MCP 工具名会在那里被拒绝（参见[执行时的 Plan 和 Ask 模式](#执行时的-plan-和-ask-模式)）。

### 共同点

允许列表固定在 `internal/agent.ToolSetForMode` 中；Agent 模式不受限制。`load_skill` 仅在 `skills.auto_discovery` 开启时存在，`spawn_agent` 仅在 `subagents.enable` 开启时存在；子智能体定义自身的 `mode` 仅在 Agent 模式父级下适用。每种模式都会从内嵌的 `agent.md`、`plan.md` 或 `ask.md` 渲染自己的系统提示词，可通过 `prompts.dir` 用 `prompts.agent_prompt`、`plan_prompt` 和 `ask_prompt` 替换（[配置参考](../reference/config.md#prompts)）。工具集，即会话通过 `/toolset` 选择的工具集或其模型命名的工具集，可以进一步收窄提供范围（[工具集](../getting-started/configuration.md#工具集)）；模式的列表始终先适用，工具集绝不会向其添加工具。该选择会与会话一起存储在 `session.json` 中。

![Web 界面输入框中的模式菜单](../../assets/modes-menu-dark-1280.png)

*Web 界面输入框的 Mode 菜单列出 Agent、Plan 和 Ask；模型、推理级别和权限模式紧随其后排列在同一行。*

## Plan 模式与计划文档

在 Plan 模式中，模型用 `plan_write` 记录其设计，该工具会将 `plans/<slug>.plan.md` 写入会话包：YAML frontmatter（`name`、`overview`、`todos`）和 markdown 正文。`plan_list` 和 `plan_read` 会查找并重新读取这些文件。每次 `plan_write` 之后，Coddy 会发布一个 `plan` 会话更新，其清单条目来自 frontmatter，因此任何 ACP 客户端都会显示预览，并在 `messages.json` 中为 Web 界面持久化一行 `plan_document`。

Web 界面会在聊天栏中将该行渲染为计划文档卡片：

- **折叠**：标题（`name`，否则为 slug，文件路径作为工具提示）、一行描述，以及页脚中的 **放弃** 和 **运行计划**。
- **展开**：默认显示渲染后的 markdown；眼睛切换按钮会切换到 markdown 行编辑器，其中包含不含 frontmatter 的正文，并在最后一次按键后约 600 毫秒通过 `PUT /coddy/sessions/{id}/plans/{slug}` 自动保存。该窗格会随文档增长，而不是在内部滚动。
- **放弃**（同一路由上的 `DELETE`）会将计划标记为已放弃：卡片仍留在对话记录中，呈弱化状态，其控件被禁用，并且该计划会从 Plan 模式的系统提示词中移除。
- **运行计划**会在下一个提示词中发送 `metadata.runPlanSlug`（通过 ACP 的 `_meta` `coddy.dev/runPlanSlug`）：Coddy 将会话切换到 Agent 模式，把计划正文注入系统提示词并运行该轮次。该切换与会话的 `/agent` 一样：通过 `--once` 或 `--count=N`、`/plan` 或 `/ask` 设置的临时模式覆盖会随其结束，因此计划在 Agent 模式下执行，绝不在 Plan 或 Ask 工具集下执行（[会话设置](session-settings.md)）。处于 Ask 模式的会话会拒绝运行计划（见下文）。会话待办列表不会由计划填充。计划正文会在该轮次仍可继续的期间一直保留：因权限提示而停止的轮次会在答案到达时重新渲染其系统提示词，因此文本保存在会话包中（`pending_plan_context.json`），并在该轮次结束后释放。
- 在子智能体子级的只读对话记录中，卡片没有页脚，其编辑器为只读。

对于没有该钩子的客户端，可移植的做法是切换到 Agent 模式，并在提示词中提及 `@plans/<slug>.plan.md`。详情：[Web 界面](../surfaces/web-ui.md#计划文档卡片计划模式对话记录)、[ACP 协议](../reference/acp-protocol.md#设计方案计划模式)。

![Web 界面中的计划文档卡片，带有“放弃”和“运行计划”](../../assets/plan-document-card-dark-1280.png)

*卡片 `plan_write` 在 Plan 模式聊天中留下的内容：计划名称和摘要，以及“放弃”和“运行计划”；下方模型的回答会重复该计划。*

## 执行时的 Plan 和 Ask 模式

两种受限模式都限制模型可获得的工具，并且都在工具调用即将执行时再次对照其列表进行检查。模型可以重放先前在其他模式下记录的历史中的调用，或命名一个从未展示给它的工具，而提供商也可以忽略给它的列表。集合之外的调用不会执行：

- 在 Ask 模式中，工具结果为 `error: tool "write" is not available in Ask mode because it is not read-only`，在 Plan 模式中为 `error: tool "write" is not available in Plan mode because it is outside the plan tool set; the user can switch to Agent mode to use it`。模型会带着该拒绝重新提示；
- 通过 HTTP 时，流会将该调用作为已取消传输，并带上相同文本；
- 用 *"始终允许"* 批准这样一个待处理的调用不会记录授权，并且当权限回答恢复该调用时会再次运行同一检查，因此在会话切换到 Plan 或 Ask 模式后批准的写入也会被拒绝。

两种模式在 MCP 工具上有所不同。Ask 不提供任何 MCP 工具，因此会拒绝每个 `server__tool` 名称。Plan 提供已连接服务器的工具，因此 MCP 名称会通过模式检查，并由 MCP 层像在 Agent 模式中一样决定：对未连接服务器的调用，或对已被其禁用开关关闭的工具的调用，会在那里被拒绝（[MCP 服务器](mcp.md#权限模型)）。

Ask 模式边界的其余部分源于其只读工具集。在轮次开始前，针对 `POST /v1/responses` 且带 `model` `ask` 的 `metadata.runPlanSlug` 会以 `409` 应答，而通过 ACP 的同一钩子会被以错误拒绝。对 `@plans/<slug>.plan.md` 的提及会作为阅读材料内联，而不是运行；在由 `/ask --once` 或 `--count=N` 置于 Ask 模式的轮次中，以及在 `/agent --once` 借给 Ask 会话的轮次中也是如此。记忆子智能体只回忆，从不保存。`spawn_agent` 不会被提供，因此 Ask 轮次无法委派。作为提示词输入的确定性操作员命令（`/compact`、`/plugin`）不在此边界内。正常路径是 `features/plan_mode.feature`、`features/ask_mode.feature` 和 `features/ask_mode_http.feature`。

![Ask 模式下的输入框](../../assets/modes-pill-hero-dark-1280.png)

*Ask 模式下的输入框：胶囊按钮采用该模式的绿色轮廓，Plan 模式采用橙色轮廓。*

## 在每个界面上切换

| 界面 | 方式 |
|---|---|
| Web 界面 | 输入框中的 **Mode** 胶囊按钮，位于 **Model** 旁边，或输入/从 `/` 菜单中选择 `/agent`、`/plan`、`/ask`；胶囊按钮的选择作为 `POST /v1/responses` 的顶层 `model` 传递；胶囊按钮显示屏幕上会话的模式，新聊天以 `agent` 启动（[会话设置](session-settings.md#web-界面)） |
| 控制台 | 在聊天中输入 `/agent`、`/plan` 或 `/ask`，或启动时使用 `--mode agent\|plan\|ask`，还可与 `-c`、`--resume` 和 `-p` 组合；在 `--remote` 模式下，该命令会切换服务器的会话 |
| ACP | 使用 `session/set_config_option`，搭配 `configId` `mode` 和 `value` `agent`、`plan` 或 `ask`（首选），或旧式 `session/set_mode` 搭配 `modeId`；智能体以 `current_mode_update` 和 `config_option_update` 应答，`session/new` 会在 `configOptions` 和 `modes` 中宣告这三种模式 |
| HTTP API | 在 `POST /v1/responses` 或 `POST /v1/chat/completions` 上将 `model` 设为 `agent`、`plan` 或 `ask`；`GET /v1/models` 使用 `owned_by` `coddy` 列出这三者，`metadata.model` 选择后端 |
| Telegram | `/agent`、`/plan` 或 `/ask` |
| 调度器 | 在任务文件 frontmatter 中设置 `mode:`，省略时为 `agent` |

每条命令也接受 `--once` 或 `--count=N`，用于为下一轮或接下来 N 轮切换模式，然后返回会话自身的模式：`/plan --once how would you split this package?` 规划一次回答，并让会话留在 Agent 模式。这些命令通过 ACP 和 HTTP 发送时也以相同方式工作，作为提示词文本的开头发送（[会话设置](session-settings.md)）。

一次性运行也以同样方式选择模式：`coddy --mode ask -p "..."` 在不触碰工作区的情况下作答。界面指南：[控制台（TUI）](../surfaces/console.md)、[Web 界面](../surfaces/web-ui.md)、[ACP 协议](../reference/acp-protocol.md)、[HTTP API](../reference/http-api.md)、[Telegram 网关](../surfaces/gateway.md)、[调度器](../operate/scheduler.md)。

<!-- docsgen:source sha256=44d44bb607ec81eb -->

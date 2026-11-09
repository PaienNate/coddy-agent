# 会话设置

会话依赖四项设置运行：模型、推理级别、[运行模式](modes.md) 和权限模式；当配置中存在工具集预设时，还有第五项：工具集。其中每一项都可以在对话本身中更改，只需在任意界面输入斜杠命令，即可在会话剩余时间内生效，或仅在接下来几轮内生效。权限对话框可以让会话退出 `ask`，而智能体会在用户要求时更改模型或推理级别，绝不会自行更改。所有更改都会经过会话管理器中的同一个设置器（`Manager.ApplySessionSettings`、`internal/session/settings.go`），而每个正在观察该会话的界面，包括浏览器标签页、控制台和编辑器，都会显示新值。

## 命令

| 命令 | 更改项 | 参数 |
|---|---|---|
| `/model <id>` | 模型 | 已配置的模型 ID |
| `/reasoning <level\|off\|default>`（别名 `/effort`） | 推理级别 | 模型提供的级别；`off` 关闭思考，`default` 回到模型自身的级别 |
| `/think [level]` | 开启思考 | 可选；不带参数时使用模型的默认级别，否则使用 `medium` |
| `/nothink`（别名 `/no_think`） | 关闭思考 | 无 |
| `/agent`、`/plan`、`/ask` | 运行模式 | 无 |
| `/permissions <ask\|accept_edits\|bypass>` | 权限模式：工具何时请求批准 | 模式；不带参数时控制台会打开选择器 |
| `/toolset <name\|default>` | 会话据以获取工具的工具集 | 一个 `toolsets[]` 预设；`default` 表示该模式的所有工具；不带参数时控制台会打开选择器 |

`/mode` 已不存在，在任何界面上都不再提供：在菜单中它紧邻 `/model`，只差一次按键就会选错命令，而如今每种模式都有自己的命令。

每条命令都可以在自己的词项中接受 `--once` 或 `--count=N`，从而将设置改为仅对下一轮或接下来 N 轮（最多 50 轮）生效，而不是对整个会话生效：

```text
/model rpa/qwen3.8-27b --count=3
/nothink --once
/plan --once how would you split this package?
```

命令只会在输入内容的开头被解析，且可以有多条依次出现，位于同一行或连续多行。第一个不属于任何命令的词会开始提示词，而该提示词本身可以是 `/compact`、`/export` 或一个技能。句子中间的命令会被当作普通文本，提及或技能正文引入的内容也一样。设置命令优先于同名技能。命令注册表位于 `internal/session/commands.go`。

## 会话范围还是接下来几轮

不带标志的更改属于会话，并持续到下一次更改。带有 `--once` 或 `--count=N` 的更改按你发起的轮次计数：你发送的每个提示词都会从每个覆盖项中消耗一轮，而该轮在其整个持续期间都保持这些值，包括在排队的后续消息之后以及在权限提示之后执行的各个步骤。后台唤醒和子智能体的轮次不消耗轮次。针对会话的更改会清除该设置的轮次覆盖，包括正在运行的轮次，因此最终生效的是你最后输入的命令。运行已保存的计划（**"运行计划"**，`runPlanSlug`）也属于这种模式更改：它会把会话切换到智能体模式并结束模式覆盖，`/plan` 或 `/ask`，因此即使 `/plan --count=N` 还有剩余轮次，计划也会在智能体模式下执行。处于询问模式的会话拒绝运行计划，而一个仅被 `/ask` 轮次提及的计划仍只是阅读材料（[运行模式](modes.md#plan-模式与计划文档)）。

会话的模型、推理级别（包括关闭思考）和模式保存在其 `session.json` 中。其权限模式也保存在那里（`sessionPermissionMode`），因此一个为无人值守运行而切换到 `bypass` 的会话，在控制台或 `coddy serve` 重启后仍处于 `bypass`。权限模式不在 `config.yaml` 中：新会话以任意界面上最后选择的模式启动，包括通过 `/permissions` 切换的会话、选择器、权限对话框或 `PATCH /coddy/sessions/{id}`，在尚未做出任何选择之前则使用 `ask`。Coddy 将该选择保存在 `${CODDY_HOME}/permission-mode.json` 中（[安全与信任](../operate/security.md#权限模式与提示)）。会话会保持自己的模式，直到它自身被切换，因此之后在另一个会话中做出的选择不会改变它；带有 `--once` 或 `--count=N` 的更改也不是一种选择：它不会影响新会话启动时所处的模式，控制台的 `--permission-mode` 启动标志同样如此，该标志只设置其运行所打开的会话。它的轮次覆盖保存在进程内存中：重启会遗忘它们。在此之前，它们会一直随会话保留，即使没有任何界面持有该会话也是如此：控制台会在 `/new` 和 `/resume` 时释放会话，而回到该会话时，其权限模式和覆盖项也会恢复。

## 发送命令时会发生什么

一条只有命令的消息不会运行任何轮次。设置会被应用，而模型看不到任何内容。Web 界面和控制台会在各自的选择器和页脚中显示更改；编辑器、Telegram 机器人以及 HTTP 客户端会把每项更改的简短通知作为对该消息的回答返回，例如 `Model: rpa/qwen3.8-27b for the next 3 turns`。对话记录不会保留这次交互的任何内容（[对话记录显示的内容](#对话记录显示的内容)）。

如果消息在命令之后还有文本，则会应用会话范围的更改，并在应用轮次范围更改的情况下将文本作为一个轮次运行。该覆盖项会随该提示词一起传递，并且只有在轮次被接纳后才会安装，因此第二个标签页或后台唤醒无法抢先占用它。

当某个轮次正在运行时，会话范围的命令会立即生效，其后的任何文本都会作为普通后续消息进入[消息队列](message-queue.md)。带有文本的轮次范围命令此时会被拒绝（通过 HTTP 返回 `409 turn_scoped_follow_up`），因为排队消息可能在投递前更改模式或被取消，所以该命令没有稳定的目标轮次。新的模型或推理级别会从正在运行的轮次的下一次模型请求开始使用，绝不会在流式传输中途切换：更改发生时正在流式传输的回答会由启动它的模型完成，并以该模型的名义存储，而对话记录会从下一个回答开始标明新模型。权限模式会在每次工具调用时读取；新的运行模式从下一轮开始生效。已经在运行的子智能体会保持其启动时获得的设置。

切换到的模型会立即读取其上下文窗口，包括其提供商的模型列表所报告的窗口，因此下一个请求，即正在运行的轮次的下一个步骤或下一个提示词的第一个请求，会以该窗口为衡量基准。当上下文超过新窗口的 `compaction.threshold_percent` 时，会在该请求之前进行压缩（[压缩](compaction.md#上下文窗口)）。

## 关闭思考

只有在模型条目显式设置 `allow_reasoning_off: true` 时，才会提供 `/nothink` 和 `/reasoning off`。默认值为 `false`：Coddy 无法推断某个提供商/模型部署是否会遵循该请求，因此操作员只有在验证该部署后才会启用此选项。

- OpenAI 兼容服务器上的 Qwen3 模型会接收 `chat_template_kwargs.enable_thinking: false`；
- 带有思考预算的 Anthropic 模型不会接收思考块，而 Claude Opus 4.7、4.8 和 5、Sonnet 5 以及 Haiku 5.5 会接收 `thinking: {"type": "disabled"}`；
- Codex 后端会接收 `none`；
- 另一个提供商/模型部署会使用其现有的 `off` 映射。

Claude Opus 5.5、Sonnet 5.5 和 Fable 模型始终会思考：在 `type: anthropic` 提供商上，即使启用了选择加入，它们也从不提供 `off`，而降低级别才是让它们减少思考的方式。

未启用选择加入时，该命令会被拒绝，并会列出模型确实提供的级别。显式为空的 `reasoning_levels: []` 即使设置了选择加入，也会隐藏选择器和 `off`。当你切换到不提供已存储级别的模型时，会话会以该模型的默认级别运行，而选择器和快照会显示实际使用的级别。

## 工具集

一个 `toolsets[]` 预设会收窄会话可获得的工具范围，适用于窗口较小的本地模型，这类模型不应承载其永远不会调用的工具的 schema（[工具集](../getting-started/configuration.md#工具集)）。模型会指定其会话运行所使用的工具集，而 `/toolset <name>` 会为此会话选择另一个工具集；`/toolset default` 不选择任何工具集，因此无论其模型指定了什么，会话都会获得其模式下的所有工具。所选项保存在会话的 `session.json` 中，并且能在重启和切换模型后保留；新会话以其模型的工具集启动。`/model <id> /toolset <name>` 可以在同一行切换两者。

只有当配置中存在预设时，才会提供工具集：此时会话设置的快照会带有 `toolset` 和 `toolsetChoices`，Web 输入框会在权限标签之后显示 **"工具集"** 标签，控制台的 `/toolset` 会打开一个选择器，其页脚会显示预设名称，编辑器则会看到 **"工具集"** 选项。没有预设时，这些都不会出现；而向配置添加预设后无需重启即可带来它们：服务器会重新发布每个活动会话的设置。

## 从权限对话框切换

当会话处于询问状态（`ask`）时，属于会话本身的权限提示会多提供一个选项：**"本次会话绕过"**（`allow_session_bypass`），对于文件写入还会提供 **"本次会话允许编辑"**（`allow_session_accept_edits`）。该选择会批准此次调用并切换会话，因此同一轮次的其余部分会在没有提示的情况下运行；与任何会话切换一样，它也会成为新会话启动时所处的模式。由子智能体中继的提示、由钩子强制产生的提示，以及 `config_commit` 或 `config_rollback` 都不会提供这些选项：提交可能重写权限策略本身，因此其提示不会切换会话。这两个在 `ask` 和 `accept_edits` 下都会询问；处于 `bypass` 的会话，无论它是如何进入该模式的，都会像其他所有调用一样在没有提示的情况下运行它们。

![Web 界面中的一个 run_command 提示，包含"允许"、"始终允许"、"始终允许 echo"、一个红色的"本次会话绕过"以及"拒绝"](../../assets/session-settings/session-settings-permission-dialog-dark-1280.png)

*处于 `ask` 的会话的权限提示：会话切换选项以红色显示在"拒绝"之前。*

## 按请求更改模型

智能体只有在用户在当前对话中提出请求时，才会使用 `switch_model` 更改其模型或推理级别。它不会自行为了困难步骤或常规工作选择不同的模型或级别，并且只有在用户指定摘要器时，才会为 `compact_context` 指定一个摘要器。子智能体则不同：智能体可以让子智能体使用与其自身不同的模型或推理级别运行，这是预期行为。系统提示词只会在提供了 `switch_model` 的轮次中描述它。相关设置有三个来源：

- `switch_model` 工具会从其下一个请求开始设置模型、推理级别或两者。用户请求默认对整个会话生效，类似 `/model`；只有当用户将更改限制为本轮或本任务时，`scope: turn` 才适用。其描述会列出已配置的模型及其级别，因此选择是在现有选项中做出的。只要有可切换的内容（多个模型，或一个带有推理级别的模型），它在每种模式下都会提供，且绝不会提供给子智能体；
- 技能 frontmatter 中的 `model` 和 `reasoning`（别名 `effort`）会在该轮次的剩余时间内生效，无论你是输入了 `/skill`，还是模型调用了 `load_skill`，除非该轮次已经运行在由命令或 `switch_model` 为其设置的值上（[技能](skills.md#yaml-frontmatter)）；
- `spawn_agent` 接受为子智能体设置的 `model` 和 `reasoning`，智能体可以自行选择它们；子智能体定义则接受 `reasoning`，紧邻 `model`；子智能体的模型依次取参数、定义中的值、父级的模型（[子智能体](subagents.md#spawn_agent-工具)）。

配置未提供的模型 ID 或级别，在 `switch_model` 或 `spawn_agent` 调用中会是一个错误，需由模型自行纠正。在技能或定义文件中，它会是智能体日志中的警告，而会话自身的值保持不变。

## 对话记录显示的内容

你所做的更改会显示在界面上保存其设置的位置：Web 界面中输入框的选择器及其覆盖项行、控制台的页脚、编辑器的选项。对话记录不会为此增加一行，无论它来自选择器、命令、权限对话框，还是启动时的 `--model` 和 `--mode`，也包括 `coddy -p`：这是你做出的更改，并且已经显示在屏幕上。

智能体自行做出的更改才是对话记录会注明的内容：其 `switch_model` 调用，或某个 frontmatter 中指定了模型或推理级别的技能。Web 界面会在该轮次末尾显示一行 **SYSTEM**，控制台会打印一行，两者都会包含更改文本，例如 `Model: rpa/qwen3.8-27b for this session` 或 `Reasoning: off for the rest of this turn`。

![一个用户请求另一个模型的轮次：switch_model 行、由新模型写出的回答，以及一行 SYSTEM："Model: stub/qwen3.8-27b for this session"](../../assets/session-settings/session-settings-agent-switch-dark-1280.png)

*智能体按请求切换了模型；结束该轮次的 SYSTEM 行是设置更改留下的唯一一行。*

由早期版本保存的会话会在其 `ui_log.json` 中保留每次更改的通知。Web 界面不会显示智能体不可能做出的那些更改：模式、权限模式、针对若干轮次的更改，以及在没有 `switch_model` 调用的轮次中为会话设置的模型或级别。

## 各界面上的行为

### Web 界面

输入框的 **"模式"**、**"权限"** 和 **"模型"** 选择器会按服务器所持有的值显示会话设置。进入会话时，会根据服务器通过 `GET /coddy/sessions/{id}/messages` 返回的快照显示该会话自己的模式、模型、级别（关闭思考运行的会话会显示 **"关闭"**）以及权限模式。起始页所选的内容，以及记住该选择的 cookie `coddy_llm_model` 和 `coddy_llm_reasoning`，仅是新聊天的默认值：它们绝不会替换屏幕上某个会话的值，因此也绝不会随下一条消息带入该会话。反过来同样成立：新聊天会以默认模型和级别在 **"智能体"** 模式下启动，无论刚离开的聊天使用的是什么模式、模型和级别，也无论是否通过"后退"按钮到达起始页；在那里选择的模式会以该模式创建聊天。提示词会携带模式（`POST /v1/responses` 的顶层 `model`）、`metadata.model` 和 `metadata.reasoning`，服务器会保留收到的内容，因此当已打开的会话仍在加载、选择器仍显示刚离开的会话时，**"发送"** 和 Enter 会等待该会话自己的快照；发往正在运行轮次的消息仍会加入队列。读取失败会重试，只有服务器返回 404 时才会将该地址作为新聊天打开。对于没有自身级别、且其模型未指定 `reasoning_default` 的会话，会显示其运行所在的级别，即 `medium`，否则显示模型的第一个级别，并且在下一条消息中不发送任何级别，因此它不会被固定到无人选择的级别。**"推理"** 菜单仅在其逻辑模型设置 `allow_reasoning_off: true` 时提供 **"关闭"**。正在运行的轮次所持有的设置（`--once`、`--count=N`）会保留在覆盖项行中，而选择器会保留会话自身的值。权限标签显示 **"先询问"**、**"接受编辑"** 或 **"绕过"**，最后一项为红色；选择器旁边的一行会列出为接下来几轮更改的内容。该标签的工具提示会指出新会话启动时所处的模式。在起始屏幕上，聊天尚未拥有会话之前，该标签会显示那个模式，即在任意界面上最后选择的模式：`GET /coddy/info` 会指出它，而每个设置快照都会将其作为 `configuredPermissionMode` 携带，因此在另一个标签页或控制台中切换的会话也会移动该标签。在那里选择的模式会作为 `/permissions <mode>` 随第一条消息发送，除非它正是那个模式，因此当新会话以 `bypass` 启动时，显式选择的 **"先询问"** 会被保留。当页面不知道该模式时（服务器未指出它，或读取失败），标签所显示的模式会随第一条消息发送，因此第一轮会按照操作员所看到的内容运行。在其他任何地方做出的更改，包括命令、对话框、用户请求的 `switch_model` 调用、同一会话上的控制台，都会以 `event: session_settings` 到达标签页并移动选择器。尚未看到最新更改的浏览器在发送时不会撤销它：请求会携带 `metadata.settingsVersion`，而较旧版本不会改动会话的设置。

![从对话框执行绕过后的输入框：选择器以红色的"绕过"标签结尾，后面是"neuraldeep/qwen3.6-35b-a3b，剩余 2 轮"](../../assets/session-settings/session-settings-composer-bypass-dark-1280.png)

*会话切换到绕过模式并为两轮更改模型后的输入框：选择器和覆盖项行会显示这两项更改，而对话记录不会为它们增加一行。*

输入 `/` 会列出命令及其参数。选择 `/model`、`/reasoning` 或 `/permissions` 会打开对应的选择器，选择 `/agent`、`/plan` 或 `/ask` 会切换模式；如果带值输入并发送，该命令会像其他任何提示词一样发往服务器。

!["/p" 的斜杠菜单：/plan、带有 ask|accept_edits|bypass 参数的 /permissions，以及 /plugin](../../assets/session-settings/session-settings-slash-menu-dark-1280.png)

*斜杠菜单会列出每条命令的参数及其 `--once|--count=N` 标志。*

### 控制台

控制台通过同一个解析器运行相同的命令。单独的 `/model`、`/reasoning` 或 `/permissions` 会打开其选择器。页脚会在文件夹旁边显示权限模式，对于 `bypass` 使用警告色，并在模型下方显示一行轮次覆盖项。两者都属于屏幕上显示的会话：在 `/new` 或 `/resume` 之后，页脚会显示所进入会话的权限模式和覆盖项。

![控制台在 /permissions bypass 以及串联的 /model --once 和 /reasoning --count=3 之后：对话记录中没有行，页脚中显示 bypass，以及 "next turn: model stub/coddy-mini • next 3 turns: reasoning high"](../../assets/session-settings/session-settings-console-footer-dark.png)

*控制台页脚：权限模式，以及下一轮和接下来三轮所更改的内容。*

启动时的 `--model`、`--mode` 和 `--permission-mode` 会为控制台打开的会话设置相同的设置，而 `--permission-mode` 不会影响新会话启动时所处的模式。在 `--remote` 下，每条命令都会更改服务器端的会话，包括权限模式（[控制台](../surfaces/console.md#远程模式---remote)）。

### ACP 编辑器

编辑器会将命令作为提示词文本发送，`available_commands_update` 会列出它们及其参数提示（`input.hint`）。推理选项会在 ACP 的 `thought_level` 类别中公布，客户端可将其渲染为思考控件；权限模式则在 `_permission_mode` 中公布，这是 Coddy 自有类别，客户端会将其显示为普通选择器（[编辑器](../surfaces/editors.md)）。

### Telegram

机器人会将 `/model <id>`、`/reasoning`、`/think`、`/nothink`、`/agent`、`/plan` 和 `/ask` 传递给会话；单独一条命令会以其通知作为回答，而单独的 `/model` 仍会打开其键盘。`/resume` 会用恢复后的会话所运行的模型和推理级别来回答（当会话及其模型都未指定时使用 `default`），这是它自己的值，而不是机器人上最后选择的模型，后者只有全新聊天才会以之启动。`/permissions` 不是机器人命令：机器人会自行批准其聊天智能体的工具（[Telegram 网关](../surfaces/gateway.md#命令)）。

### HTTP

`PATCH /coddy/sessions/{id}` 接受 `selectedModelId`、`selectedReasoning`、`mode` 和 `permissionMode`，并通过 `turns` 表示更改持续这么多轮，同时以会话的 `settings` 快照作为回答。`GET /coddy/sessions/{id}/messages` 返回相同的快照，`GET /coddy/commands` 会列出命令及其类型、参数提示，以及每条命令为会话接受的值；仅由命令组成的提示词会以 `POST /v1/responses` 上的 `coddy_meta.settings_only` 作为回答（[HTTP API](../reference/http-api.md)）。

快照：

```json
{
  "sessionId": "sess_abc123def456",
  "version": 42,
  "model": "rpa/qwen3.8-27b",
  "reasoning": "high",
  "reasoningChoices": ["low", "medium", "high", "off"],
  "mode": "agent",
  "permissionMode": "bypass",
  "configuredPermissionMode": "ask",
  "overrides": [{ "setting": "model", "value": "rpa/qwen3.8-mini", "turnsLeft": 2 }]
}
```

`version` 会为一个进程中各会话的快照排序；客户端会保留其见过的最高值。`configuredPermissionMode` 是新会话启动时所处的模式，即在任意界面上最后选择的模式（在尚未做出任何选择之前为 `ask`）；该字段沿用了该模式来自配置时的名称。带有 `active: true` 的覆盖项由正在运行的轮次持有。

## 相关

[运行模式](modes.md)、[安全与信任](../operate/security.md#权限模式与提示)、[斜杠命令](../reference/slash-commands.md)、[工具](../reference/tools.md)、[消息队列](message-queue.md)。

<!-- docsgen:source sha256=58152e9870d929f5 -->

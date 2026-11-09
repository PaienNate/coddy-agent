# 作为模型的 ACP 智能体

一个 `type: acp` 提供商运行一个本地程序，该程序通过 stdio 使用 [Agent Client Protocol](https://agentclientprotocol.com)：`claude-agent-acp`、`codex-acp`、`opencode acp`、`zcode-acp-server`、另一个 `coddy acp`，并将其作为模型提供服务。你可以像选择任何其他模型一样在模型选择器中选择它，每个界面都会在答案流式传输时显示：控制台、Web 界面、ACP 编辑器和 Telegram 网关。

模型背后的智能体是一个完整的智能体。它自带系统提示词、自己的工具以及它拥有的任何沙箱，并在会话的工作区中运行它们。Coddy 将对话发送给它，显示它报告的内容，并在它请求权限时询问你。Coddy 自己的工具、规则和技能不参与其中：该智能体从不调用 Coddy 工具，因此在此类模型上的一个轮次就是一次请求和一次回答。

## 配置

```yaml
providers:
  - name: claude
    type: acp
    acp:
      command: claude-agent-acp
      modes: {plan: plan, ask: plan}
models:
  - model: claude/default
```

`acp.command` 是程序，在 `PATH` 上查找，`acp.args` 是它的参数。提供商名称后的模型 id 选择该智能体自己的某个模型：`default` 保留智能体启动时使用的模型，其他任何值都必须是该智能体提供的模型（设置会列出它们，见[设置与试运行](#设置与试运行)）。模型 id 可以包含斜杠，因为 `coddy acp` 就是这样命名其模型的（`coddy/neuraldeep/qwen3.8-27b`）。

| 智能体 | `command` / `args` | 登录 | `modes.plan` / `modes.ask` |
| --- | --- | --- | --- |
| Claude Code | `claude-agent-acp` | Claude CLI 登录 | `plan` |
| Codex | `codex-acp` | `auth_method: api-key` 搭配 `OPENAI_API_KEY`（位于 `env` 中），或 Codex CLI 的 ChatGPT 登录 | `read-only` |
| OpenCode | `opencode`、`[acp]` | OpenCode 自己的提供商 | `plan` |
| ZCode | `zcode-acp-server`（带 `process: per_session`） | ZCode 自己的提供商文件 | `plan` |
| Coddy | `coddy`、`[acp]` | 它自己的 `config.yaml`（为它提供一个自己的 `CODDY_HOME`，位于 `env` 中） | `plan` / `ask` |

作为智能体的另一个 Coddy 读取它自己的 `config.yaml`，并提供其中列出的模型。本页的截图运行的就是这一行。

```yaml
providers:
  - name: coddy
    type: acp
    acp:
      command: coddy
      args: [acp]
      env:
        CODDY_HOME: ~/.coddy-agent
      modes: {plan: plan, ask: ask}
models:
  - model: coddy/default
  - model: coddy/neuraldeep/qwen3.8-27b
```

命令、参数、`cwd` 或 `env` 值中的前导 `~` 表示你的主目录。

`acp` 块的其他键：

- `env` - 添加到智能体启动环境的变量。智能体会继承 Coddy 的环境，因此 Coddy 拥有的 `ANTHROPIC_API_KEY` 或 `OPENAI_API_KEY` 可能优先于智能体自己的登录；在这里将该变量设置为某个值或空字符串，以选择智能体看到的内容。
- `cwd` - 为智能体及其打开的每个会话固定一个文件夹。当它为空时，每个会话都在其所服务的 Coddy 会话的工作区中打开。任何值中的 `${CWD}` 就是该工作区，此时智能体会按工作区各启动一次。
- `auth_method` - 智能体启动后，Coddy 传给 ACP `authenticate` 调用的身份验证方法 id，适用于在客户端完成身份验证前拒绝会话的智能体。带 API 密钥的 `codex-acp` 需要 `api-key`。
- `permissions` - `ask`（默认）、`allow` 或 `deny`，见[权限](#权限)。
- `modes` - 每个 Coddy 模式对应的智能体自身模式，见[计划与询问模式](#计划与询问模式)。
- `process` - `shared`（默认）或 `per_session`，见[智能体进程](#智能体进程)。
- `idle_timeout_ms` - 智能体在最后一次调用后保持运行的时间；0 表示六十分钟。

在提供商行上，`proxy` 会作为环境传递给智能体：`inherit`（默认）保持 Coddy 的代理变量不变，包括 `ALL_PROXY`；`none` 会移除它们；代理 URL 会设置 `HTTPS_PROXY` 和 `HTTP_PROXY`，将 `NO_PROXY` 设为环回地址并移除 `ALL_PROXY`，从而让智能体自己的本地流量保持直连。智能体是否遵守它们取决于智能体本身。`timeout_ms` 为正数时，会限制每个提示词。`api_base`、`api_key` 和 `api_key_command` 不会被读取。

## 智能体上的会话

一个 Coddy 会话绑定到智能体的一个 ACP 会话。第一轮会打开它并发送到此为止的对话；之后的每一轮只发送新消息，智能体保留自己的历史记录，包括工具结果。Coddy 的系统提示词不会发送：智能体有自己的系统提示词，而 Coddy 的提示词描述的是智能体无法调用的工具。Coddy 在每一轮添加的运行时块（时间、待办清单）会在消息之后发送。

当历史记录不再延续智能体所持有的内容时，对话会在新的 ACP 会话中重新开始，并在其第一个提示词中包含完整历史：在回退、编辑消息、压缩、工作区变更之后，或智能体进程重新启动之后。在回答完成后再次询问同一件事（重新生成）也会打开新会话。

智能体在工作时报告的内容：

- 其文本作为回答流式传输，其推理作为推理流式传输；
- 其每一次工具调用都是一行工具记录，就像 Coddy 自己的工具那样，标题是它处理的对象（`Read README.md`、`read notes.txt`），并带有传给它的内容和它产生的内容。智能体仅以工具名称宣布的调用，会在它指明所处理的文件或命令时，或在调用结束时，获得对应行。这些行是智能体自己的工具：Coddy 不运行其中任何一个，也不把它们发送给任何模型；
- 智能体结束回答时若没有任何文本，会显示 `(the agent finished without a message)`；若未达到普通结束就停止，则会说明原因，例如 `(the agent stopped: it reached its output limit)`。

![智能体的回答位于其两个工具调用下方，测试运行已展开并显示其命令和输出](../../assets/acp-provider/acp-provider-answer-dark-1280.png)

*智能体的一个回答：它的工具调用各自成行，测试运行已展开并显示其命令和输出。*

回答会保留每次调用输出的前几行，而会话会保留完整输出，就像 Coddy 自己的工具那样：重新加载后，这些行会在控制台和加载该会话的编辑器中恢复，单击即可查看某一行的完整输出。

智能体会写一行、运行一个工具、再写下一行，而所有这些都属于同一个回答。这些行就位于事情发生的位置：对话记录会记录每次调用被宣布时回答已写了多少，并在那里切分回答，因此重新加载后回读的一轮看起来与运行时一样。由较旧版本 Coddy 写入的会话没有此类标记，其调用会像当时那样位于整个回答之前。

已完成轮次在其回答下方留下的那一行，即运行了多久、写了多少，在这里不包含生成速度。智能体会围绕自己的等待过程流式传输回答，因此它每秒写多少并不能说明模型的情况。

模型 id 通过智能体的模型选项选择智能体的模型；当智能体的推理选项提供同名值时，会话的推理级别会设置该选项；会话的模式通过 `acp.modes` 选择智能体的模式。某一轮未指定的值会回到会话启动时的值，因此较早轮次的选择绝不会一直保留。

智能体工作时写入的消息会进入[消息队列](message-queue.md)，而 **steer** 消息会被立即读取：这样的轮次是一次调用，其中没有 Coddy 的步骤，因此其步骤边界就是智能体的提示词。Coddy 会中断该提示词，并携带该消息重新开始这一轮，就像带有等待队列的 Stop 手动执行的那样。智能体保留自己的 ACP 会话，并从已完成的部分继续，其到目前为止的回答仍留在对话记录中。要求等待该轮次的消息不会改变任何事：智能体会先完成。

Stop 会按协议要求取消智能体的提示词。智能体到目前为止的工作仍保留在回答中，下一条消息会在同一会话中继续，而被取消的提示词绝不会再次发送：会重复该提示词的请求将改为失败，因为智能体可能已经对其采取了行动。当智能体在十秒内未确认取消时，Coddy 会关闭其 ACP 会话（下一条消息会从对话记录开始一个新会话），并再给该提示词二十秒来结束；此后它会停止智能体进程，而在 `process: shared` 下，这还会结束它服务的其他会话的提示词。

## 计划与询问模式

在计划与询问模式下，会话承诺不写入。智能体的工具不是 Coddy 的工具，因此 Coddy 通过将智能体置于其自身的某个只读模式来履行该承诺：`acp.modes.plan` 和 `acp.modes.ask` 指定该模式（上表列出了每个智能体的模式）。如果未设置会话模式对应的键，该轮次会被拒绝，并给出指明该键的消息，`coddy --dry-run` 也会对此发出警告。

除此之外，智能体在计划或询问模式下发送的每个权限请求都会被拒绝，无论 `acp.permissions` 怎么设置。对于不询问就写入的智能体（Claude 的 `bypassPermissions`、ZCode 的 `yolo`，或你之前始终允许的工具），只能靠该模式来约束。

## 权限

当智能体在运行某个工具之前询问时，由 `acp.permissions` 决定答案：

- `ask` - 由你回答，使用与 Coddy 自己的工具相同的提示，在持有该会话的任何界面上进行。该提示位于智能体调用的行下方（控制台和编辑器将其标题标为 `Agent:` 以及调用标题），显示智能体关于该调用的说明，并提供三个选项：允许、在智能体提供时始终允许、拒绝。Web 界面将它们标为 Allow、Allow always 和 Reject；其他界面显示智能体给它们的名称。在 `bypass` 权限模式下，请求无需询问即被允许；在 `accept_edits` 下，编辑请求也是如此；
- `allow` - 每个请求都会被允许一次；
- `deny` - 每个请求都会被拒绝。

![Web 界面中智能体写入请求的权限提示，带有 Allow、Allow always 和 Reject 选项](../../assets/acp-provider/acp-provider-permission-dark-1280.png)

*智能体请求写入文件：该提示位于调用行下方，就像 Coddy 自己的工具一样。*

无人回答的请求会被拒绝，而不是被取消，因此智能体会继续其轮次。Allow always 是智能体自己的 "always"：智能体将其保留在自己一侧，而不是 Coddy 一侧。

Telegram 机器人会不经询问就批准其聊天智能体的工具调用，因为设置该机器人的管理员已经这样决定了。智能体的请求不属于这些调用，因此机器人会像对待子智能体一样，在聊天中用按钮询问，由点击决定。

## 智能体不做什么

智能体只在会话内回答。不属于会话轮次的调用永远不会到达它：

- 压缩会在其链中跳过 `acp` 模型，并需要将 `compaction.model` 设为另一个模型；自动压缩完全不会在 `acp` 模型上运行，因为智能体保留并压缩自己的上下文；
- 新聊天的标题和标签使用 `agent.model`（若未设置，则使用第一个配置的模型）；当该模型是 `acp` 模型时，它们使用按字母顺序第一个不是该类型的模型；
- 目标检查需要将 `supervisor.model` 设为另一个模型，而记忆子智能体需要 `memory.model`：否则，对记忆子智能体的委派不会在智能体上运行；
- Web 界面输入框中的 Improve prompt 使用按字母顺序第一个不是智能体的模型；
- 将 `/v1/chat/completions` 和 `/v1/responses` 与 `acp` 模型作为直接模型使用时，会返回 400；智能体配置文件（`agent`、`plan`、`ask`）可与其配合使用；
- 消息平台限制的轮次（发送者不是机器人管理员）会被拒绝：该限制约束的是 Coddy 的工具，而不是智能体的工具。

智能体报告的 token 计数会保留用于统计；智能体上的会话的上下文环并不测量智能体的上下文。

## 智能体进程

一个智能体进程服务于提供商行的每个会话（当块中的某个值指定 `${CWD}` 时，每个工作区一个），每个会话都在自己的 ACP 会话中。它在第一次调用时启动，在最后一次调用后保持 `idle_timeout_ms`，并随 Coddy 停止（`coddy serve`、`coddy acp`、控制台）。死掉的进程会在下一次调用时重新启动，其会话会从对话记录重新开始。

`process: per_session` 为每个 Coddy 会话提供独立进程。`zcode-acp-server` 需要它才能让两个会话同时回答：每个进程一次只运行一个提示词。

一个智能体可以是另一个 Coddy（`coddy acp`），而后者可以有自己的 `acp` 行。每个 Coddy 会将 `CODDY_ACP_DEPTH` 加一后传给它启动的智能体；自身 `CODDY_ACP_DEPTH` 为 3 的 Coddy 不会启动任何智能体，因此相互启动的智能体链会在三层后结束。

智能体不能调用协议的任何文件系统和终端方法（`fs/*`、`terminal/*`）：Coddy 不会声明这两种能力，智能体使用自己的工具读取、写入和运行命令。

## 设置与试运行

在 "设置 → LLM 提供商" 中，类型为 `acp` 的行会显示 ACP 智能体块，而不是 API 字段。

![设置，LLM 提供商：类型为 acp 的行，运行 coddy acp 并使用自己的 CODDY_HOME](../../assets/acp-provider/acp-provider-settings-dark-1280.png)

*一个 `acp` 行，运行 `coddy acp`：命令、其参数以及智能体自己的 `CODDY_HOME`。*

该行末尾的 "模型" 列表会向已保存行的智能体询问其模型：请先保存该行，因为表单不会启动尚未保存的程序。`default` 是智能体启动时使用的模型，其他则是它提供的模型。与每个提供商行一样，加号会将 `<row>/<id>` 添加到逻辑模型，勾选标记则会再次将其移除。

![设置，LLM 提供商：coddy 行的模型列表，包含 default、neuraldeep/qwen3.8-27b 和 neuraldeep/gemma-4-31b](../../assets/acp-provider/acp-provider-models-dark-1280.png)

*`coddy acp` 提供的模型。`default` 和 `neuraldeep/qwen3.8-27b` 位于逻辑模型中，加号添加的是 `neuraldeep/gemma-4-31b`。*

该行的逻辑模型与其他模型一样，是输入框选择器中的一个模型，显示为模型名称，并在其工具提示中显示完整 id。

![输入框的模型选择器，带有 default 和 qwen3.8-27b，这是 coddy 行的两个逻辑模型](../../assets/acp-provider/acp-provider-model-picker-dark-1280.png)

*模型选择器中有 `coddy` 行的两个逻辑模型。*

`coddy --dry-run` 会启动每个智能体，打开一个会话以读取其模型，检查已配置的模型 id 是否在其中，并对缺失的 `modes.plan` 或 `modes.ask` 发出警告。

## 故障排查

- `authentication required; set acp.auth_method to one of: ...` - 智能体要求客户端对其进行身份验证。请单独登录该智能体，或设置它列出的方法（使用它在 `env` 中读取的键）。
- `acp.modes.ask of provider "..." is not set`（或 `acp.modes.plan`） - 会话处于该模式；请为其设置智能体的只读模式。
- `the agent's last prompt in this conversation ended without a clear outcome` - 在提示词被取消或失败且智能体可能已采取行动之后，某个请求重复了最后一条消息，而 Coddy 没有再次发送它。请发送一条新消息。
- `exited` 以及智能体的 stderr：智能体停止了。下一次调用会重新启动它；请手动运行命令以查明原因。
- 来自 ZCode 的 `Session is not active`：两个会话同时向一个 ZCode 进程发出请求；请设置 `process: per_session`。

<!-- docsgen:source sha256=f934274a4b090923 -->

# `config.yaml` 参考

`~/.coddy/config.yaml` 的逐字段参考。叙述性文档（文件发现、`.env`、提供商指南）见 [配置](../getting-started/configuration.md)。

一部机器可读的 [JSON Schema](../../../internal/config/config.schema.json) 随本参考一同提供，发布于 **https://coddy.dev/config.schema.json** 并嵌入二进制文件；在命令行中，`coddy -t` / `--test-config` 会依据它校验文件（见 [配置](../getting-started/configuration.md#从命令行检查文件)）。将编辑器的 YAML 语言服务器指向它，即可获得自动补全和拼写检查：

```yaml
# yaml-language-server: $schema=https://coddy.dev/config.schema.json
```

VS Code（配合 YAML 扩展）、Zed、Neovim 和 Helix 会自动读取此注释，Coddy 也会把它写入其保存的每个 `config.yaml`（见 [配置](../getting-started/configuration.md)）；JetBrains IDE 不读取它，需要改为在 **JSON Schema Mappings** 下注册该 URL。该 schema 由 `internal/config/docs_schema_test.go` 中的 `TestDocsConfigSchemaMatchesStructs` 与 Go 配置结构体保持同步。可选的三态字段（例如 `compaction.enable`、`models[].stream`、`tools.output_limits.*`）除具体值外也接受 `null`：`null` 表示未设置，与省略该键完全相同。Coddy 保存文件时会直接省略未设置的字段 - 它从不自行写入 `null`，并且仅在某个值不同于内置默认值时才添加该键，因此手工维护的文件可以保持精简。

除非标记为 **required**，每个字段都是可选的；空的 `config.yaml`（或完全没有）是有效的，并使用内置默认值。任何字符串值都可以用 `${VAR_NAME}` 引用环境变量（在加载文件时展开）。要在值中保留 **字面量 `$`**（例如 `$2y$10$…` 这样的机密），将其写成双份 `$$` - 对于 `proxy` 字段，UI 会自动这样做。`${CODDY_HOME}` 由加载器展开；`${CWD}` 会保留在已加载的值中，并由读取该路径的组件按会话展开，进程级 `sessions.dir`、`memory.dir` 和 `logger.file` 除外（见 [配置](../getting-started/configuration.md#环境变量引用)）。

## 智能体自配置

智能体会话提供一组有类型的配置工具，具有分阶段、类似 uci 的语义：

- `config_get` 从活动 YAML 文件中读取点分路径。机密形态字段（包括 `api_key_command` 以及任何以 `_api_key` 结尾的键，例如 `tools.websearch.brave_api_key`）以及环境和标头值（`env` 或 `headers` 条目、`default_headers` 值）会以 `<redacted>` 返回；HTTP API 的 MCP 服务器列表也以同样方式显示两个 mcp.json 文件中的值。
- `config_set` **暂存** UCI 风格命令（`set`、`add_list`、`del_list`、`delete`），而不改动文件。未知的 schema 路径以及会使配置无效的命令会在暂存时被拒绝。回显的命令列表会将机密形态的值掩码为 `<redacted>`；暂存存储保留原始值。
- `config_changes` 列出提交将应用的暂存命令（机密已脱敏）。
- `config_commit` 应用暂存批次：校验、将先前的文件快照到 `config.yaml.prev`（若配置文件先前不存在则为空文档，因此首次提交仍可回滚）、原子写入，并热重载技能、规则、内置工具和 MCP 信任策略。由于提交可能启动 MCP 进程（更改了 `mcp.project_trust`）并更改权限策略本身，它会在 `ask` 和 `accept_edits` 两种模式下请求工具权限 - 只有处于 `bypass` 的会话会跳过对话框 - 并且提示会列出暂存命令，机密已脱敏。还会指示智能体先请求用户确认保存。如果运行时重载失败，文件会被恢复且暂存命令保留；如果连该恢复也失败，暂存列表会保持已消费状态，以免盲目重试将其重放。
- `config_revert` 丢弃暂存命令（全部，或某个路径下的命令）。
- `config_rollback` 将提交前快照恢复到活动文件之上（两者交换，因此第二次回滚会撤销第一次）并热重载。它带有与 `config_commit` 相同的权限策略，智能体在调用前会警告用户。

命令和路径采用类似 OpenWrt 的 `uci` CLI 的点分形式，并用选择器指定具名序列条目：

| 命令 | 含义 |
|---|---|
| `set agent.max_turns=40` | 设置映射字段 |
| `set providers[name=openrouter]={"type":"openai"}` | 按标量字段选择序列对象；设置时若不存在则追加 |
| `add_list skills.dirs=/opt/skills` | 追加序列条目 |
| `del_list skills.dirs=/opt/skills` | 移除匹配的序列条目 |
| `delete providers[name=openrouter]` | 删除字段或条目 |
| `skills.dirs.0`（路径形式） | 序列索引 |

根路径（`.` 或 `/`）为只读。对象和数组的值使用 JSON；字符串类型字段接受字面文本。暂存命令会持久化在会话包中，因此可在重启和 HTTP 权限恢复后继续存在。

捆绑的 `/configure-coddy` 技能向智能体教授此语法、先确认后提交的工作流，以及 MCP 服务器和技能的安全发现/安装工作流；它还包含面向智能体的配置区域目录，任何 schema 变更都必须与本参考一同更新。进程级监听器变更可能仍需要重启相关命令；热重载明确保证针对当前会话的智能体配置、技能、规则、内置工具和 MCP 信任策略。MCP 服务器不属于此文件（见 [`mcp`](#mcp)）。

## 字段参考

由 `internal/config/config.schema.json`（嵌入二进制、发布于 <https://coddy.dev/config.schema.json> 的 schema）以及加载器应用的默认值生成，生成器为 `make docs`；当此块过期时 `make docs-check` 会失败。类型按 schema 声明的方式显示（`string or null` 是三态字段），默认值是空 `config.yaml` 所解析到的结果，描述是 `coddy -t` 为该键打印的 `doc:` 行。不适合放入表格行的说明见 [说明](#说明)。

<!-- docsgen:config:start -->
### `providers`

上游 LLM 供应商的 API 凭据和传输选择。当 api_key 为空时，运行时读取 NAME_API_KEY 环境变量（NAME 是提供商名称的大写形式，连字符映射为下划线）；type 为 "codex" 的行使用 ChatGPT 登录并且不读取密钥，Kimi 订阅也不读取 API 密钥。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `providers` | 对象列表 |  | 上游 LLM 供应商的 API 凭据和传输选择。当 api_key 为空时，运行时读取 NAME_API_KEY 环境变量（NAME 是提供商名称的大写形式，连字符映射为下划线）；type 为 "codex" 的行使用 ChatGPT 登录并且不读取密钥，Kimi 订阅也不读取 API 密钥。 |
| `providers[].name` | 字符串 |  | 用作 models[].model 第一段的逻辑 id。ASCII 字母、数字、连字符、下划线；必须以字母开头。 |
| `providers[].type` | 字符串，可选值：`openai`, `deepseek`, `anthropic`, `neuraldeep`, `opencode`, `codex`, `devin`, `kimi`, `acp` |  | 此提供商条目的通信协议。官方 DeepSeek API 使用 deepseek，以获得思考控制、提示词缓存统计和 V4 Flash/Pro 预设；其他 OpenAI 兼容端点使用 openai。 |
| `providers[].mode` | 字符串，可选值：`""`, `subscription`, `platform` |  | Kimi 凭据模式。省略表示订阅。 |
| `providers[].region` | 字符串，可选值：`""`, `global`, `china` |  | Kimi 区域。省略表示全球。 |
| `providers[].kimi_identity` | 字符串 |  | 此 Kimi 提供商行的稳定随机标识，用于隔离受管凭据。 |
| `providers[].api_base` | 字符串 |  | 可选的基 URL 覆盖。对于 type "openai"，包含 /v1（例如 http://localhost:11434/v1）；对于 type "anthropic"，为兼容 Anthropic 的网关（默认 https://api.anthropic.com）。对于 type "neuraldeep"，它选择部署：https://api.neuraldeep.ru/v1（俄罗斯，默认）或 https://api.neuraldeep.tech/v1（国际镜像）；任何其他值都回退到默认值。对于 type "opencode"，它选择网关：https://opencode.ai/inference/go/v1（Go 订阅，默认）或 https://opencode.ai/inference/v1（按量付费 Console）；任何其他值都回退到默认值。对于 type "kimi"，订阅拒绝 api_base；Platform 请求使用由 region 选择的端点，而不是 api_base。对于 type "codex" 和 "devin" 会被忽略，它们始终使用其官方端点；对于 type "acp" 也会被忽略，其智能体在此机器上运行。 |
| `providers[].user_agent` | 字符串，可选值：`""`, `openai`, `anthropic`, `coddy` | "" | 用于 openai 和 anthropic 类型的客户端身份。留空（默认值）、openai 或 anthropic 会保留当前提供商 SDK 身份（OpenAI/Go 或 Anthropic/Go）；coddy 会发送 Coddy/<running version>。适用于补全和模型列表。其他提供商类型会忽略此项。 |
| `providers[].api_key` | 字符串 |  | 提供商密钥。字面密钥、在加载时展开的 "${ENV}" 引用，或为空以在 LLM 调用时读取 NAME_API_KEY。解析顺序：api_key -> api_key_command 的 stdout -> NAME_API_KEY。对于 type "devin"，它是 Devin 会话令牌，设置后优先于浏览器登录和 Devin CLI 登录。对于 type "opencode"，显式密钥优先于 `coddy providers login opencode` 存储的登录。对于 type "kimi"，Platform 使用 Platform API 密钥；订阅拒绝 api_key 并且不使用 NAME_API_KEY。对于 type "codex" 会被忽略，它使用 ChatGPT 登录；对于 type "acp" 也会被忽略，其智能体自行登录。 |
| `providers[].api_key_command` | 字符串 |  | 可选的凭据辅助命令。当 api_key 为空时，它通过检测到的主机 shell（Windows 上为 pwsh、powershell 或 cmd；其他平台为 bash 或 sh）运行，并将去除首尾空白后的 stdout 用作密钥。失败时，解析会回退到 NAME_API_KEY。对于 type "kimi"，Platform 可以使用此命令；订阅拒绝它。对于 type "codex" 和 "acp" 绝不会运行，因为它们不接收密钥。 |
| `providers[].proxy` | 字符串 | inherit | inherit（默认值，与省略该键相同）、none，或代理 URL：http://,、https://,、socks5:// 或 socks5h://。inherit 遵循 Coddy 进程环境指定的代理，即 HTTPS_PROXY、HTTP_PROXY 和 NO_PROXY，从不用于回环地址；不会读取 ALL_PROXY。none 直接连接并忽略这些变量。代理 URL 会让此行的每个请求都通过该代理，包括 NO_PROXY 和回环地址；使用 SOCKS 时，由代理解析主机名。该设置覆盖提供商的每个请求：其补全、模型列表、账户用量和登录。对于 type "acp"，智能体自行发出请求：inherit 会保持 Coddy 环境的代理变量不变，包括 ALL_PROXY；代理 URL 会作为 HTTPS_PROXY 和 HTTP_PROXY 传递给它，并将 NO_PROXY 设为回环地址且移除 ALL_PROXY；none 会从其环境中移除代理变量，而智能体是否遵循它们取决于智能体。 |
| `providers[].timeout_ms` | 整数 | 0 | 对发往此提供商的每个 LLM HTTP 请求的可选限制，包括流式响应体读取；对于 type "acp"，则限制智能体回答的每个提示词。0（默认值）不设置客户端超时，因此在大型上下文上缓慢的提示词处理绝不会被提前截断。 |
| `providers[].usage_limits_panel` | 布尔值 或 null | true | 显示此提供商的账户用量面板（控制台页脚行和 /usage、Web 界面中的用量章节和横幅），并为此读取提供商的用量端点（type 为 neuraldeep 时为 GET /v1/limits，type 为 codex 时为 Codex 后端的用量端点，type 为 devin 时为席位管理状态 RPC）。对于 type kimi，订阅用量读取 /usages，而 Platform 没有用量来源。省略或 true 会保持面板开启；false 会隐藏它并停止此行的这些读取。只有 type 具有用量来源的提供商才会受到影响。 |
| `providers[].acp` | 对象 或 null |  | type 为 "acp" 的提供商所运行的智能体：一个在 stdio 上使用 Agent Client Protocol 的本地程序，在第一次调用时启动，并为此行的每个会话保留（当 process: per_session 时每个会话一个，当某个值命名 ${CWD} 时每个工作区一个），在无调用达到 idle_timeout_ms 后停止。仅在 type 为 acp 的行上读取；参见 https://coddy.dev/docs/features/acp-provider. |
| `providers[].acp.command` | 字符串 |  | 智能体程序（在 PATH 中查找），例如 claude-agent-acp、codex-acp、opencode 或 coddy。type acp 必填。 |
| `providers[].acp.args` | 字符串列表 |  | 命令之后的参数，例如 opencode 和 coddy 的 [acp]。 |
| `providers[].acp.env` | 字符串映射 |  | 添加到 Coddy 传递给智能体的环境中的变量，例如智能体读取的密钥。值、参数、命令或 cwd 中的 ${CWD} 是会话的工作区，此时智能体会按每个工作区启动一次。 |
| `providers[].acp.cwd` | 字符串 |  | 智能体进程和每个 ACP 会话的固定文件夹。为空（默认值）时，会在其服务的 Coddy 会话的工作区中打开每个会话。 |
| `providers[].acp.auth_method` | 字符串 |  | Coddy 在 initialize 之后传给 ACP authenticate 调用的身份验证方法 id，用于在客户端完成身份验证之前拒绝会话的智能体（codex-acp：api-key，并在 env 中带有 OPENAI_API_KEY）。为空则不调用任何方法：智能体使用其自身已有的登录。 |
| `providers[].acp.permissions` | 字符串，可选值：`ask`, `allow`, `deny` | ask | 当智能体请求运行其某个工具的权限时 Coddy 的答复：ask 通过会话的权限提示将请求转发给用户（在权限模式为 bypass 的会话中，以及 accept_edits 下的编辑，会不经询问直接允许），allow 批准一次，deny 拒绝。在 plan 和 ask 模式下，每个请求都会被拒绝。 |
| `providers[].acp.modes` | 对象 |  | 针对每个 Coddy 会话模式，智能体自身的模式（ACP 模式 id，例如 plan 或 read-only）。空的 modes.agent 会保持智能体的默认模式。modes.plan 和 modes.ask 必须命名智能体的只读模式；没有它时，该模式下的轮次会被拒绝。 |
| `providers[].acp.modes.agent` | 字符串 |  | 会话处于 agent 模式时智能体的模式。 |
| `providers[].acp.modes.plan` | 字符串 |  | 会话处于 plan 模式时智能体的模式。 |
| `providers[].acp.modes.ask` | 字符串 |  | 会话处于 ask 模式时智能体的模式。 |
| `providers[].acp.idle_timeout_ms` | 整数 | 0 | 在最后一次调用之后经过这么长时间就停止智能体进程；下一次调用会再次启动它，并将对话重放到新会话中。0（默认值）表示六十分钟。 |
| `providers[].acp.process` | 字符串，可选值：`shared`, `per_session` | shared | shared（默认值）：一个智能体进程服务此行的每个会话，每个会话都在其自己的 ACP 会话中。per_session：每个 Coddy 会话都有自己的智能体进程，适用于每个进程一次只运行一个提示词的智能体（zcode-acp-server）。 |

### `models`

智能体和界面可选择的有名称的模型条目。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `models` | 对象列表 |  | 智能体和界面可选择的有名称的模型条目。 |
| `models[].model` | 字符串 |  | "provider_name/api_model_id"：第一段路径必须匹配 providers[].name；其余部分发送到 LLM API（其本身可以包含斜杠）。 |
| `models[].kimi_metadata` | 对象 或 null |  | 绑定到此模型引用、提供商模式和区域的 Kimi 目录协议。 |
| `models[].kimi_metadata.model` | 字符串 |  | 目录中列出的完整 provider/model 引用。 |
| `models[].kimi_metadata.protocol` | 字符串，可选值：`kimi`, `anthropic`, `openai_responses` |  | 为此模型报告的协议。 |
| `models[].kimi_metadata.mode` | 字符串，可选值：`subscription`, `platform` |  | 用于列出该模型的提供商模式。 |
| `models[].kimi_metadata.region` | 字符串，可选值：`global`, `china` |  | 用于列出该模型的提供商区域。 |
| `models[].max_tokens` | 整数 |  | 每条助手消息的补全令牌上限。codex 模型绝不会发送它（Codex 后端拒绝 max_output_tokens），因此在那里它不构成限制，coddy -t 和 --dry-run 会对这样的值发出警告。 |
| `models[].temperature` | 数字 |  | 采样温度（0 = 确定性；越高 = 越随机）。 |
| `models[].max_context_tokens` | 整数 | 0 | 模型的上下文窗口（以令牌计）：Web 界面上下文环、控制台上下文百分比和自动压缩触发器据此衡量。0 会在提供商报告时从提供商的模型列表中读取它（NeuralDeep hub、Anthropic、Codex 和 Devin 目录、vLLM、OpenRouter、LM Studio），否则为 128000。 |
| `models[].multimodal` | 布尔值 | false | 除文本外，模型还接受图像/文件输入；界面会为此模型显示文件附件按钮，并且 read 会向它展示 PNG、JPEG、GIF 或 WebP 文件中的图片，而不是将其作为二进制拒绝。 |
| `models[].reasoning_levels` | 字符串列表 或 null |  | 覆盖为此模型提供的推理级别。省略则根据模型 id 自动检测（gpt-5* -> minimal,low,medium,high；OpenAI o 系列、gpt-oss*、qwen3* 和 Claude 扩展思考模型 -> low,medium,high），并去掉聚合器放在其前面的供应商（qwen/qwen3.8-flash 按 qwen3.8-flash 读取）。显式空列表会隐藏选择器。设置界面通过其 "Fetch reasoning levels" 按钮从 GET /coddy/config/reasoning-levels 填充此字段。 |
| `models[].reasoning_default` | 字符串 |  | 为新聊天预选的推理级别；必须是解析出的级别之一，否则会被忽略。 |
| `models[].allow_reasoning_off` | 布尔值 | false | 在此模型的推理选择器中显示 "Off"。仅当此提供商/模型部署遵循 Coddy 用于禁用推理的提供商特定请求时才启用它；Coddy 无法自动验证该能力。 |
| `models[].stream` | 布尔值 或 null | true | 用于与此模型通信的传输方式。省略（或 true）以通过 SSE 流式传输。false 会发出一个阻塞式补全请求并一次性交付整个答案，适用于处理事件流不佳的服务器或代理。对于 type 为 codex 的提供商会拒绝，因为其后端仅支持流式传输。 |
| `models[].toolset` | 字符串 |  | 此模型上的会话据以提供工具的 toolsets[] 预设名称，除非会话使用 /toolset <name> 选择另一个。不存在或为空则提供该模式会提供的每个工具。可使用它让小型上下文的本地模型免于接收其从不调用的工具的 schema。 |

### `agent`

主智能体循环的默认值（模型 id 和安全上限）。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `agent.queue_mode` | 字符串，可选值：`steer`, `after_turn` |  | 运行中的轮次期间按 Enter 的首选排队模式：steer 加入下一个 ReAct 步骤，after_turn 在当前答案之后开始新轮次。不存在时，交互式界面会在首次使用时询问。 |
| `agent.model` | 字符串 |  | 可选的默认 models[].model id。交互式界面（Web 界面、控制台）按会话选择模型；未指定模型的无人值守调用（coddy -p、coddy acp、没有模型选择器的 API 请求）需要设置它，当它为空时它们会报告 "no model configured"。 |
| `agent.max_turns` | 整数 | 165 | 一个提示词轮次中 ReAct 步骤（LLM 调用）的上限。默认 165；设为 0 可禁用上限。达到上限的轮次会以一条指明该上限的通知结束。 |
| `agent.llm_retry_max` | 整数 或 null | 3 | 由传输重试、空答案恢复、首令牌重新发出和输出限制恢复（在 max_tokens 处被截断的步骤）共享的额外尝试次数，直到工具取得进展或出现新的后续消息。0 禁用这些重试。循环防护、Stop 钩子、回退模型和配额重置等待有各自独立的限制。 |
| `agent.llm_retry_base_ms` | 整数 | 1000 | LLM 重试之间的初始退避时间，以毫秒计。服务器提供的暂停（Retry-After-Ms / Retry-After 响应头、"Limit resets at" / "retry in Ns" 正文短语）会覆盖指数退避，上限为 60 秒。 |
| `agent.llm_min_interval_ms` | 整数 | 0 | 连续 LLM 调用之间的最小间隔，以毫秒计，包括重试尝试（0 禁用节奏控制；例如在严格的免费套餐上为 12000）。 |
| `agent.llm_first_token_timeout_ms` | 整数 或 null | 90000 | 流式 LLM 调用在被轮次取消之前可以保持静默多长时间（API 挂起防护）。显式 0 禁用该防护；阻塞式（stream: false）传输绝不会受到防护。 |
| `agent.llm_stream_idle_timeout_ms` | 整数 或 null | 300000 | 流式 LLM 响应在发送其首批字节之后可以多长时间不发送任何内容，之后流会因停滞而被切断（答案中途挂起防护）：已交付的文本会保留，轮次会以指明停滞的方式结束。显式 0 禁用该防护；等待第一个字节属于 llm_first_token_timeout_ms，并且阻塞式（stream: false）传输绝不会受到防护。 |
| `agent.loop_guard` | 布尔值 或 null | true | 失控循环防护：切断退化为自我重复的流式响应，并阻止使用相同参数反复调用的工具。 |
| `agent.loop_tool_repeat_limit` | 整数 或 null | 2 | 在循环防护介入之前，连续 ReAct 响应中连续出现的相同工具调用（相同名称、相同规范参数）的次数。0 禁用工具重复检查。 |
| `agent.loop_stream_repeat_cycles` | 整数 或 null | 5 | 在一个流式响应内，流被切断之前连续相同的输出周期数。0 禁用流检查。 |
| `agent.loop_nudge_max` | 整数 或 null | 1 | 在一个轮次被循环防护以通知停止之前，可以多少次将其推回正轨。 |
| `agent.wait_for_limit_reset` | 布尔值 | false | 等待已触发的用量限制解除并重新发出调用，而不是以提供商的错误结束轮次。适用于顶层轮次，其提供商在 Retry-After 响应头或 429 本身的文本中指定了超出重试预算的暂停；等待期间轮次锁和客户端流保持打开，因此默认关闭。 |
| `agent.wait_for_limit_reset_max_ms` | 整数 或 null | 14400000 | 一个轮次等待限制所花费的最长时间总计，以毫秒计，包括重试包装器自身因限制而进行的休眠；超过该时间的暂停会立即以错误结束轮次。显式 0 表示从不等待。 |

### `prompts`

覆盖内置系统提示词模板（Go text/template）。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `prompts.dir` | 字符串 | "" | 包含提示词模板文件的目录。为空时使用内嵌默认值。支持 ~ 和 ${CWD}（渲染时的会话 cwd）。 |
| `prompts.agent_prompt` | 字符串 | agent.md | prompts.dir 中用于 agent 模式的文件名。 |
| `prompts.plan_prompt` | 字符串 | plan.md | prompts.dir 中用于 plan 模式的文件名。 |
| `prompts.ask_prompt` | 字符串 | ask.md | prompts.dir 中用于 ask 模式的文件名。 |

### `instructions`

操作者添加到系统提示词的文件（AGENTS.md 约定），位于每个会话本来就会读取的文档之后：先是智能体主目录的 AGENTS.md 和 DESIGN.md，然后是会话文件夹中的这些文件，再是工具进入的文件夹中的嵌套文件，这些文件都没有在此命名，也无法关闭。提示词中已有的文件不会被读取两次。参见 https://coddy.dev/docs/features/rules。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `instructions.files` | 字符串列表 | [] | 额外的指令文件，按列出的顺序追加在 AGENTS.md 和 DESIGN.md 文档之后；默认空。${CODDY_HOME}、${CWD} 和开头的 ~ 会展开；绝对路径条目按原样读取，相对条目则相对于会话 CWD 解析。命名了提示词已包含的文档的条目会被跳过。 |

### `skills`

扫描技能的目录（SKILL.md 和根目录下的 .md/.mdc 文件）。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `skills.dirs` | 字符串列表 | [] | 额外的技能目录，在四个默认文件夹之后读取，并且优先级高于它们。默认文件夹始终会被读取，优先级从低到高：${HOME}/.agents/skills（与每个智能体共享，npx skills 和 npx skillsbd 会安装到那里）、项目的 .agents/skills、${CODDY_HOME}/skills（Coddy 自带的技能和已安装的技能）、项目的 .coddy/skills；然后按顺序读取这些条目。在多个目录中发现的技能会从此顺序中的最后一个目录获取，重复命名的目录会在其最后出现的位置读取。${CODDY_HOME} 在加载文件时展开，${HOME} 和 ~ 展开到你的主文件夹，${CWD} 和相对路径相对于会话的工作区（包括新聊天所选的文件夹）。 |
| `skills.project_trust` | 字符串，可选值：`ask`, `allow`, `deny` | ask | 项目 .coddy/marketplaces.json 的信任策略，该文件会随检出一起传递："ask"（默认）会将其来源和市场排除在每次同步之外，直到操作者为该工作区批准该确切条目；"allow" 会像对待操作者自己的 ${CODDY_HOME}/marketplaces.json 一样对待它们；"deny" 从不使用它们（它们会列为已关闭）。它们安装的内容会进入 ${CODDY_HOME}/skills。项目的技能文件夹（.coddy/skills、.agents/skills）不受影响。技能市场不在 config.yaml 中声明：它们位于 ${CODDY_HOME}/marketplaces.json 和项目的 .coddy/marketplaces.json 中，旧的 skills.sources 列表会在加载时移入主目录文件。参见 https://coddy.dev/docs/features/skills#project-marketplaces-and-trust。 |
| `skills.auto_discovery` | 布尔值 或 null | true | 提供由模型驱动的 load_skill 工具，使智能体在请求匹配时自行将目录中技能的完整指令拉入轮次，而不是要求显式的 /name。默认为 true。 |

### `rules`

从 ${CODDY_HOME}/rules（操作者自己的规则，在每个工作区中应用）以及从会话 CWD 下的一个项目文件夹中发现规则文件，该文件夹是 .coddy/rules、.agents/rules、.cursor/rules、.claude/rules、.codex/rules 中第一个包含规则文件的文件夹，并且在向下到达工具或提及所及文件的路径上的每个子文件夹中都采用相同方式，这些规则各自应用于其自己的子文件夹；.mdc 文件是 Cursor 规则，.md 文件是 Claude Code 规则。AGENTS.md 和 DESIGN.md 文档在此意义上不是规则，这些设置也不会关闭它们。参见 https://coddy.dev/docs/features/rules。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `rules.auto_discover` | 布尔值 或 null | true | 自动扫描规则文件夹：${CODDY_HOME}/rules、会话 CWD 的项目文件夹，以及工具或提及所及子文件夹的规则文件夹。AGENTS.md 和 DESIGN.md 文档无论哪种方式都会被读取。 |
| `rules.systems` | 字符串列表 | [] | 限制加载哪些规则系统：user（${CODDY_HOME}/rules）、coddy、agents-dir（.agents/rules）、cursor、claude、codex。空表示全部。被排除的项目文件夹会从链中移除；在允许的文件夹中，读取第一个包含规则文件的文件夹。agents 过去用于切换嵌套的 AGENTS.md 文件，现在不再影响它们（它们始终会被读取）；它仍会被接受，而仅包含 agents 的列表不会加载任何规则文件夹。 |

### `mcp`

不绑定到单个服务器条目的 MCP 设置。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `mcp.project_trust` | 字符串，可选值：`ask`, `allow`, `deny` | ask | 项目本地 <cwd>/.coddy/mcp.json 的信任策略，该文件随检出一起携带："ask" 让其中的服务器保持冷状态，直到操作者针对该工作区批准该确切声明；"allow" 自动启动它们（仅限受信任的工作区）；"deny" 永不加载它们。 |
| `mcp.idle_timeout_seconds` | 整数 或 null | 300 | 不再被任何会话持有的 MCP 服务器在 Coddy 停止它之前继续运行的秒数：项目服务器在其工作区的最后一个会话释放它之后，ACP 客户端发送的服务器在发送它的最后一个会话关闭之后。在此期间接管该服务器的会话会发现它仍在运行。随进程启动的 coddy serve、console 和 coddy acp 的全局服务器不受影响，而被关闭、不再获批、被移除或以不同方式声明的服务器会立即停止。0 表示最后一个会话释放后立即停止该服务器。 |

### `tools`

内置工具的文件系统与 shell 策略。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `tools.command_allowlist` | 字符串列表 |  | 永远不需要权限的命令。一个条目精确覆盖其命令，或以空格后跟参数的形式覆盖；一行中的多条命令（&&、\|\|、;、&、管道、换行）只有在每一条都被覆盖时才无提示运行，而命令替换或重定向总是会询问。在 PowerShell 和 cmd.exe 上，一个条目只覆盖不含操作符的命令。"*" 允许所有命令。 |
| `tools.ssh_connect_timeout` | 整数 | 30 | SSH 连接（ssh_run_command 工具）的 TCP 拨号超时，单位为秒。 |
| `tools.output_limits` | 对象 |  | 每个工具结果或错误可返回进 LLM 上下文的最大行数。每个启用的限制还会施加每次调用 64 KiB 的字节安全上限，使超长的单行无法绕过它。0 为该工具禁用两项限制。未设置的字段回退到内置默认值。 |
| `tools.output_limits.read` | 整数 或 null | 1000 | 读取文件分页或目录列表的最大行数。 |
| `tools.output_limits.grep` | 整数 或 null | 200 | grep 返回的 path:line:content 记录的最大数量。 |
| `tools.output_limits.glob` | 整数 或 null | 300 | glob 返回的路径的最大数量。 |
| `tools.output_limits.print_tree` | 整数 或 null | 400 | 目录树的最大行数。 |
| `tools.output_limits.run_command` | 整数 或 null | 500 | shell 命令合并的 stdout+stderr 最大行数。 |
| `tools.output_limits.ssh_run_command` | 整数 或 null | 500 | 远程 SSH 命令合并的 stdout+stderr 最大行数。 |
| `tools.output_limits.webfetch` | 整数 或 null | 800 | 抓取页面 markdown 的最大行数。 |
| `tools.output_limits.websearch` | 整数 或 null | 200 | 搜索结果的最大行数。 |
| `tools.output_limits.default` | 整数 或 null | 1000 | 适用于上面未列出的任何工具，包括 MCP 工具。0 表示不限。 |
| `tools.background` | 对象 |  | 智能体在会话任务池中以分离方式运行、而不阻塞轮次的命令。任务池随 coddy 进程一同终止；每个任务的元数据与捕获的输出保留在会话包中，被重启中断的任务会报告为孤立。 |
| `tools.background.enable` | 布尔值 或 null | true | 在 run_command 上提供后台选项，并暴露后台任务工具。 |
| `tools.background.max_concurrent` | 整数 | 5 | 一个会话可同时运行的后台任务数量。0 使用默认值。 |
| `tools.background.default_timeout_seconds` | 整数 | 900 | 对未显式指定超时且无时长估算的任务的硬性限制。0 使用默认值。 |
| `tools.background.max_timeout_seconds` | 整数 | 3600 | 施加到任何请求的或由估算得出的超时上的上限。0 使用默认值。 |
| `tools.background.output_buffer_bytes` | 整数 | 262144 | 每个任务的输出有多少保留在内存中供状态指示器使用。完整日志仍会写入会话包。0 使用默认值。 |
| `tools.preview_server` | 对象 |  | preview_server 工具：智能体在项目目录上以空闲端口启动的静态文件服务器，使操作者可以在浏览器中打开 HTML、JS 和 CSS 成果。该服务器是会话的后台任务，运行到被停止、会话被删除或 coddy 退出为止。 |
| `tools.preview_server.enable` | 布尔值 或 null | true | 提供 preview_server 工具。该服务器是后台任务，因此 tools.background.enable: false 也会将它关闭。 |
| `tools.preview_server.host` | 字符串 | 127.0.0.1 | 服务器绑定的地址，不含端口：端口始终是空闲端口。任何非回环地址都会把所服务的目录暴露给能够访问该地址的人。 |
| `tools.preview_server.public_host` | 字符串 | "" | 写入智能体所分发 URL 中的主机，用于运行在操作者浏览器之外（容器、远程机器）的 coddy。为空则使用绑定主机。 |
| `tools.canvas` | 对象 |  | Web 界面的画布：工作区的 .canvas 文件在“文件”窗口中显示为一块由实时 frame 组成的板，每个 frame 都是工作区的一个页面，运行在沙箱中，由智能体通过 canvas_show 放入，或由用户从“文件”窗口放入；canvas_inspect 读取浏览器所看到的内容。参见 https://coddy.dev/docs/features/canvas。 |
| `tools.canvas.enable` | 布尔值 或 null | true | 提供 canvas_show 和 canvas_inspect 工具，提供 frame 服务并在 Web 界面中显示画布。 |
| `tools.canvas.allow_cdn` | 布尔值 或 null | true | 允许 frame 从常见 CDN 加载库、样式和字体：cdn.jsdelivr.net、unpkg.com、cdnjs.cloudflare.com、cdn.tailwindcss.com、fonts.googleapis.com 和 fonts.gstatic.com。 |
| `tools.canvas.allowed_sources` | 字符串列表 |  | frame 可以从中加载和连接的更多主机，形式为 https://host 或 https://*.host，可带可选的端口和路径。画布上的页面可以将其文件夹下读取到的内容发送给其中任何一个。 |
| `tools.websearch` | 对象 |  | websearch 工具询问哪些搜索引擎、其结果按什么顺序合并，以及它可为询问它们花费什么。每个引擎都在结果旁报告自己的结果，因此被阻止的后端会被点名，而不是被算作“网上没有内容”。 |
| `tools.websearch.engines` | 字符串列表 | ["brave","bing"] | 要询问的搜索引擎，按合并顺序排列。未设置时先询问 brave 再询问 bing。默认不询问 "ddg" 和 "google"：从服务器上实测，DuckDuckGo 对每个查询都返回反机器人插页，而 Google 在浏览器中渲染其结果。"searxng" 需要 searxng_url。"exa"、"firecrawl"、"parallel"、"tinyfish" 和 "tavily" 是 OpenCode 兼容的提供商；支持 keyless 时密钥可选。 |
| `tools.websearch.engine_timeout_seconds` | 整数 | 8 | 一个引擎在被报告为不可用之前可花费的秒数。0 使用默认值。 |
| `tools.websearch.total_timeout_seconds` | 整数 | 20 | 整个搜索可花费的秒数，无论询问多少个引擎。0 使用默认值。 |
| `tools.websearch.max_concurrent_engines` | 整数 | 4 | 同时询问多少个引擎。0 使用默认值。 |
| `tools.websearch.snippet_chars` | 整数 | 320 | 单条结果描述的最大字符数。0 使用默认值。 |
| `tools.websearch.cache_ttl_seconds` | 整数 | 300 | 一个引擎的回答在被再次询问之前可复用的秒数，使重复搜索不会重复发送请求。0 使用默认值；负值会关闭缓存。 |
| `tools.websearch.searxng_url` | 字符串 | "" | 你自己的 SearXNG 实例的基础地址，通过其 JSON API 询问（在其 settings.yml 中启用 json 格式）。自托管的聚合器是被抓取的引擎遭拒时的持久答案；localhost 和局域网地址是特意允许的。 |
| `tools.websearch.brave_api_key` | 字符串 | "" | Brave Search API 订阅令牌。有它时 brave 引擎使用官方 JSON API 而非读取公开结果页面，后者在 Brave 重新部署时没有可破坏的解析器。为空则读取 BRAVE_API_KEY 环境变量（也包括来自 ${CODDY_HOME}/.env 的），因此密钥无需存储在此处。 |
| `tools.websearch.deepseek_api_key` | 字符串 | "" | DeepSeek Search API 密钥。deepseek 引擎询问 DeepSeek 的 Search API（POST /v1/tools/search），它没有可抓取的公开页面，因此没有密钥时该引擎不可用。为空则读取 DEEPSEEK_API_KEY 环境变量（也包括来自 ${CODDY_HOME}/.env 的），因此密钥无需存储在此处。 |
| `tools.websearch.exa_api_key` | 字符串 | "" | Exa 搜索 MCP 密钥。为空则读取 EXA_API_KEY，或使用 Exa 的 keyless 层。 |
| `tools.websearch.firecrawl_api_key` | 字符串 | "" | Firecrawl 搜索 MCP 密钥。为空则读取 FIRECRAWL_API_KEY，或使用其 keyless 层。 |
| `tools.websearch.parallel_api_key` | 字符串 | "" | Parallel 搜索 MCP 密钥。为空则读取 PARALLEL_API_KEY，或使用其 keyless 层。 |
| `tools.websearch.tinyfish_api_key` | 字符串 | "" | TinyFish 搜索 MCP 密钥。为空则读取 TINYFISH_API_KEY，或使用其免费 keyless 层。 |
| `tools.websearch.tavily_api_key` | 字符串 | "" | Tavily 搜索 API 密钥。为空则读取 TAVILY_API_KEY，或使用其 keyless 层。 |
| `tools.http_request` | 对象 |  | http_request 工具（智能体的 curl）的策略：它无需询问即可到达的目的地，以及每个请求都会发送的请求头。在 ask 和 accept_edits 权限模式下，除非请求的目的地已在此处允许或在会话中获批，否则会询问操作者；bypass 从不询问。参见 https://coddy.dev/docs/features/http-requests。 |
| `tools.http_request.allowlist` | 字符串列表 | [] | 请求无需询问即可到达的目的地：主机（api.github.com）、子域通配符（*.example.com），二者均可带可选的 :port，或来源（http://localhost:8080），或地址前缀（https://api.example.com/v1/）。"*" 允许所有目的地。一个条目还覆盖请求上传的文件和未校验的证书；代理需要自己的条目，而响应保存到的文件遵循写入策略。 |
| `tools.http_request.default_headers` | 字符串映射 | {} | 每次 http_request 调用都会发送的请求头，除非调用自身指定了该请求头：针对拒绝工具的站点的浏览器 User-Agent，内部服务所期望的 Accept。它们取代工具自身的默认值（其 coddy-agent User-Agent），调用的请求头取代它们 - 调用中的空值会移除其一 - 而此处的空值会让该请求头不出现在每个未设置它的调用中。名称由字母、数字、- 和 _ 组成，以字母开头。Host、Content-Type、Content-Length 和 Transfer-Encoding 描述单个请求，会被拒绝，逐跳请求头（Connection、Keep-Alive、Proxy-Connection、TE、Trailer、Upgrade）以及 Proxy-Authorization 同样会被拒绝，因为 https 请求会经由代理隧道把它们带到每个来源。这些请求头会发往该工具到达的每个目的地，因此此处的凭据会到达所有目的地；权限提示会点名它们，并仅显示描述客户端的那一个（User-Agent、Accept……）的值，config_get 会对模型隐藏它们的值，而 webfetch、URL 提及和模型提供商从不发送它们。参见 https://coddy.dev/docs/features/http-requests#default-headers。 |

### `subagents`

用户定义的子智能体，模型可通过 spawn_agent 委派给它。定义是带 YAML frontmatter 的 markdown 文件；每次运行都是父会话的一个后台任务，拥有自己的子会话和对话记录。参见 https://coddy.dev/docs/features/subagents。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `subagents.enable` | 布尔值 或 null | true | 注册 spawn_agent 工具，并在系统提示词中列出子智能体目录。 |
| `subagents.dirs` | 字符串列表 | [] | 额外的定义目录，在四个默认文件夹之后读取，且优先级高于它们。默认目录始终会被读取，优先级从低到高：${HOME}/.agents/agents、项目的 .agents/agents、${CODDY_HOME}/agents、项目的 .coddy/agents；然后是这些条目，按其顺序。在多个目录中找到的同一定义，取此顺序中的最后一个，重复列出的目录在其最后出现的位置读取。${CODDY_HOME} 在加载文件时展开，${HOME} 和 ~ 展开为你的主文件夹，${CWD} 和相对路径相对于会话的工作区解析。工作区内的目录属于项目范围，遵循信任策略。 |
| `subagents.project_trust` | 字符串，可选值：`ask`, `allow`, `deny` | ask | 在工作区内找到的定义的信任策略，这些定义随检出一起携带："ask" 加载它们，但在操作者针对该工作区批准该确切文件之前拒绝启动（设置 -> 子智能体中的盾牌、在运行 coddy 的机器上执行 coddy agents trust，或用会话工作区作为 cwd 调用 POST /coddy/subagents/{name}/trust）；"allow" 将它们视同操作者自己的文件；"deny" 永不读取它们。 |
| `subagents.max_concurrent` | 整数 | 4 | 整个进程可同时处于运行中的子智能体运行数量，无论由哪个会话启动。超过限制的启动会被拒绝，而不是排队。0 使用默认值。 |
| `subagents.max_depth` | 整数 或 null | 1 | 启动可嵌套的深度：1 允许会话启动子智能体，但后者不能再启动；0 在所有地方禁止启动。省略则使用默认值。 |
| `subagents.default_timeout_seconds` | 整数 | 1800 | 对定义和调用都未给出超时的单次运行的硬性限制。受 tools.background.max_timeout_seconds 限制。0 使用默认值。 |
| `subagents.max_turns` | 整数 | 0 | 子智能体可执行的 ReAct 轮数。0 则遵循 agent.max_turns。 |

### `hooks`

在会话生命周期节点运行的操作者命令：工具调用前后、提交提示词时、智能体停止时、会话开始时以及压缩前后。定义是 Claude Code 形态的 JSON 文件（~/.coddy/hooks.json、.coddy/hooks.json、.claude/settings*.json）。参见 https://coddy.dev/docs/features/hooks。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `hooks.enable` | 布尔值 或 null | true | 是否加载并运行钩子。 |
| `hooks.files` | 字符串列表 | ["${CODDY_HOME}/hooks.json","${CWD}/.claude/settings.json","${CWD}/.claude/settings.local.json","${CWD}/.coddy/hooks.json"] | 定义文件，优先级从低到高；每个匹配的钩子都会运行。${CODDY_HOME} 和 ${CWD} 会展开；相对条目相对于会话 cwd 解析。工作区内的文件属于项目范围，遵循 project_trust；Claude Code 设置文件中只读取 hooks 键。 |
| `hooks.project_trust` | 字符串，可选值：`ask`, `allow`, `deny` | ask | 在工作区内找到的钩子文件的信任策略，这些文件随检出一起携带："ask" 列出它们，但在操作者在运行 coddy 的机器上针对该工作区批准该确切文件之前不运行任何内容（在那里执行 coddy hooks trust，或用会话工作区作为 cwd 调用 POST /coddy/hooks/trust）；"allow" 将它们视同操作者自己的文件；"deny" 永不读取它们。 |
| `hooks.default_timeout_seconds` | 整数 | 60 | 对定义未给出超时的单个钩子进程的硬性限制。0 使用默认值。 |
| `hooks.stop_loop_limit` | 整数 | 5 | 每轮中 Stop 钩子可让智能体重新工作的次数。0 使用默认值。 |
| `hooks.max_output_chars` | 整数 | 10000 | 单个钩子可交给模型或用户的上下文、消息和理由的上限；更长的值会被截断并加标记。0 使用默认值。 |

### `logger`

进程日志级别、输出、格式和文件轮转。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `logger.level` | 字符串，可选值：`debug`, `info`, `warn`, `warning`, `error` | info | 写入已配置输出的最低严重级别（"warning" 可作为 "warn" 的别名接受）。 |
| `logger.levels` | 对象列表 |  | 按组件的最低严重级别。组件是记录在其 "component" 属性中携带的点分名称（gateway、gateway.telegram、gateway.pachca、session、agent、scheduler、provider.acp）；父名称覆盖其下嵌套的一切，最长的已配置前缀优先。没有组件的记录遵循 "level"。 |
| `logger.levels[].component` | 字符串 |  | 点分组件名称，例如 "gateway.telegram"。 |
| `logger.levels[].level` | 字符串，可选值：`debug`, `info`, `warn`, `warning`, `error` |  | 该组件的最低严重级别。 |
| `logger.outputs` | 字符串列表 | ["stderr"] | 日志记录的去向；stdout、stderr、file 的任意组合。为空表示仅 stderr。 |
| `logger.file` | 字符串 |  | 文件接收器的路径；当 outputs 包含 "file" 时为必填。支持 ${CODDY_HOME}。 |
| `logger.format` | 字符串，可选值：`text`, `json` | text | 日志记录格式。 |
| `logger.rotation` | 对象 |  | 文件接收器的基于大小的轮转。 |
| `logger.rotation.max_size_mb` | 整数 | 0 | 文件达到此大小（MB）后轮转；0 禁用基于大小的轮转。 |
| `logger.rotation.max_files` | 整数 | 0 | max_size_mb > 0 时保留的轮转备份数量。 |

### `sessions`

持久化会话包的存储位置。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `sessions.dir` | 字符串 | "" | 会话根目录。为空则解析为 ${CODDY_HOME}/sessions。支持 ${CODDY_HOME} 和 ~。 |

### `compaction`

总结较早的对话历史，使长会话持续适配模型的上下文窗口。手动压缩命令，加上在模型上下文窗口某一百分比处自动触发。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `compaction.enable` | 布尔值 或 null | true | 压缩的总开关（手动命令和自动触发）。默认为 true。 |
| `compaction.auto_enable` | 布尔值 或 null | true | 自动压缩：达到 threshold_percent 时，较早的轮次被折叠为摘要，或者，当窗口中没有更早的轮次时，折叠正在回答的这一轮的较早步骤（此时摘要行以提示词本身逐字开头，最多保留最近四个步骤）；以及提供商以超出其上下文窗口为由拒绝的请求会被压缩一次并再次询问。设为 false 可保留手动 /compact 和“上下文”弹出框操作，但不启用上述任何行为。默认为 true。 |
| `compaction.threshold_percent` | 整数 | 80 | 当估计的上下文用量达到有效模型上下文窗口的这一百分比（1..100）时触发自动压缩：即其 max_context_tokens，否则为其提供商的模型列表所报告的窗口，否则为 128000 - 也就是 Web 界面上下文圆环所显示的窗口。 |
| `compaction.keep_recent_turns` | 整数 或 null | 2 | 保留多少最近的用户轮次（每轮包含其后的智能体回复和工具活动）逐字不变；只有该边界之前的历史会被总结。0 表示总结整个窗口。当窗口中的用户轮次不多于此值时，压缩保留得更少：自动触发保留到正在回答的提示词为止，手动命令保留到零。 |
| `compaction.keep_tail_tokens` | 整数 或 null | 15000 | 压缩逐字保留的尾部在估算 token 上的绝对上限：能容得下的最新轮次会保留，最多 keep_recent_turns 个，至少一个（手动压缩折叠全部内容时为零个）。它限制宽窗口留下的尾部，否则折叠之后的每个步骤都要再次为它付费。0 表示关闭该上限，只由 keep_recent_turns 决定。 |
| `compaction.model` | 字符串 | "" | 可选，用于总结调用的 models[].model。为空则使用会话的有效模型。 |
| `compaction.fallback_models` | 字符串列表 |  | 当前一个失败时按顺序尝试的总结器模型（models[].model id）。无论是否列出，会话自身的模型都是最后手段，因此压缩不会因某个不可达的总结器而被阻塞。 |
| `compaction.result_eviction` | 对象 |  | 在构建 LLM 请求时把已被取代的工具结果折叠为简短占位符（持久化的对话记录从不重写），使分页读取大文件、大范围搜索或目录列表的扇出无法在后续每个轮次中钉住失效的行。read 和 grep 结果在模型标记它们（keep_result，或 keep:true）或它们属于最近的结果时保留；glob、print_tree、websearch 和 webfetch 的列表在包含其一的最新三个步骤内保留（一个步骤是一条助手消息及其所有工具调用的全部结果）。一次写入会使其更改所覆盖的较早读取、grep 和文件列表失效。 |
| `compaction.result_eviction.enable` | 布尔值 或 null | true | 工具结果驱逐的总开关（read、grep 及列表工具）。默认为 true。 |
| `compaction.result_eviction.keep_recent` | 整数 或 null | 2 | 多少最近的、可驱逐的 read 和 grep 结果（读取分页、grep 转储）保持完整作为工作窗口。0 表示不保留。列表结果不计入其中。默认值 2 使一个 read 和一个 grep 同时保持有效；设为 1 时，比较两个结果的模型会不断重新获取被另一个驱逐的那个。 |
| `compaction.result_eviction.min_result_bytes` | 整数 或 null | 2000 | 小于或等于此大小的结果永远不会被逐出（太小，不值得使用占位符）。0 会使每个结果都成为候选对象。 |
| `compaction.result_eviction.start_percent` | 整数 或 null | 50 | 仅当估算上下文达到模型上下文窗口的该百分比时才开始逐出。低于该值时，重放的历史记录会原样发送，因此提供商的提示词缓存会保留它；占位符出现在历史记录中间会使其后每个缓存 token 失效。0 表示从第一个结果开始逐出。没有 max_context_tokens 的模型的读取和 grep 结果会从第一个结果开始逐出；列表结果则按其提供商报告的窗口或默认 128000 来衡量。 |
| `compaction.result_eviction.start_tokens` | 整数 或 null | 60000 | 逐出门槛的绝对上限：逐出在窗口的 start_percent 与这么多对话 token 中较小者处开始。对话的测量不计请求自身的开销（系统提示词、工具定义、规则），这些都无法逐出，因此很大的 AGENTS.md 不会在第一个结果处就打开门槛。0 表示只保留 start_percent。 |

### `memory`

可选的记忆子智能体（实现位于 external/memory；运行时通过 memory.enable 启用）。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `memory.enable` | 布尔值 | false | 通过 spawn_agent 向模型暴露内置的记忆子智能体（需要 memory 构建标签）。 |
| `memory.model` | 字符串 | "" | 记忆子智能体运行所用的确切 models[].model id；为空则使用会话的模型。 |
| `memory.fallback_models` | 字符串列表 |  | 当排在前面的模型在应答前失败时，按顺序尝试的记忆子智能体模型（models[].model id）。会话自身的模型始终作为最后手段，无论是否列出，因此一个无法访问的部署不会拖垮记忆运行。 |
| `memory.dir` | 字符串 | "" | 长期记忆根目录。为空时解析为 ${CODDY_HOME}/memory。支持 ${CODDY_HOME} 和 ~。 |
| `memory.recall_max_turns` | 整数 | 6 | 与 persist_max_turns 一起限制记忆子智能体的 ReAct 轮数；子智能体的上限取两者中较大的值。 |
| `memory.persist_max_turns` | 整数 | 12 | 与 recall_max_turns 一起限制记忆子智能体的 ReAct 轮数；子智能体的上限取两者中较大的值。 |
| `memory.copilot_max_tokens` | 整数 | 4096 | 记忆模型调用的补全 token 上限。 |
| `memory.max_search_hits` | 整数 | 8 | memory_search 返回的最大片段数。 |
| `memory.max_note_chars` | 整数 或 null | 900 | coddy_memory_save 保存的单条笔记正文的最大长度，以字符（而非字节）计。显式设置为 0 会移除上限。 |
| `memory.additional_prompt` | 字符串 | "" | 仅给记忆子智能体的操作者指令：其系统提示词中的一个章节，主智能体永远不会看到。为空则不添加任何内容。 |
| `memory.additional_prompt_max_chars` | 整数 | 0 | additional_prompt 的字符数上限；更长的文本会在该处截断，智能体日志会说明，coddy -t 会报告。0 表示无上限。 |

### `httpserver`

OpenAI 兼容 HTTP API 的默认值（仅由使用 -tags http 构建的二进制使用；嵌入式 SPA 需要 -tags http,ui）。参见 https://coddy.dev/docs/reference/http-api。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `httpserver.enable` | 布尔值 或 null | true | 在此进程中提供 HTTP API（以及嵌入式 SPA）。省略表示 true；在仅轮询 messenger 或为集群提供中继的节点上设置为 false。 |
| `httpserver.host` | 字符串 | 127.0.0.1 | HTTP API 的绑定地址。为空表示 127.0.0.1；设置为 0.0.0.0 以接受来自其他机器的连接。 |
| `httpserver.port` | 整数 | 0 | 当 coddy serve 未传递 -P/--port 时使用的默认监听端口。0 回退到 12345。 |
| `httpserver.max_body_bytes` | 整数 | 0 | POST /v1/chat/completions 和 POST /v1/responses 接受的最大请求体，以字节计。0 表示内置的 512 MiB。更大的请求体将以 413 和 request_too_large 错误拒绝。 |
| `httpserver.max_client_tools` | 整数 | 0 | 客户端在 POST /v1/chat/completions 和 POST /v1/responses 上向直接模型提供的最多工具数。0 表示内置的 128，即 OpenAI 自身的限制。更多工具会以 400 和 too_many_tools 错误拒绝，该错误会指明数量和此设置。对于其 MCP 服务器提供更多工具的客户端，可提高此值；提供商自身的限制仍会生效。 |
| `httpserver.max_client_tool_schema_bytes` | 整数 | 0 | 在 POST /v1/chat/completions 和 POST /v1/responses 上，单个客户端工具的参数 schema 最大大小，以字节计。0 表示内置的 256 KiB。更大的会以 400 和 tool_parameters_too_large 错误拒绝。 |
| `httpserver.auth_token` | 字符串 | "" | HTTP API 的可选 bearer 凭据。为空表示不进行身份验证（历史默认值）。使用 "${ENV}" 引用可将密钥排除在文件之外；GET /coddy/config 永远不会回显它。优先使用 --auth-token / CODDY_HTTP_TOKEN，以将其完全排除在 config.yaml 之外。参见 https://coddy.dev/docs/operate/remote。 |
| `httpserver.login` | 对象 |  | 可选的浏览器密码登录，使网络上的服务器不会被任何发现该端口的人读取。除非此处或 CODDY_HTTP_USER / CODDY_HTTP_PASSWORD 中存在账户，否则关闭。API 客户端继续使用 auth_token。参见 https://coddy.dev/docs/operate/remote。 |
| `httpserver.login.enable` | 布尔值 或 null |  | 显式开启或关闭登录表单。省略时跟随凭据：此处或环境中的账户会启用它。设置为 false 时，即使变量仍存在，也会保持表单关闭。 |
| `httpserver.login.mode` | 字符串，可选值：`password` | password | 浏览器进行身份验证的方式。仅实现了 "password"；保留该键是为了在不移动配置的情况下添加受信任代理模式。 |
| `httpserver.login.user` | 字符串 | "" | 登录表单的账户名。支持 "${ENV}" 引用，例如 "${CODDY_HTTP_USER}"。 |
| `httpserver.login.password_hash` | 字符串 | "" | 密码的 argon2id 哈希，采用 PHC 形式（"$argon2id$v=19$m=...$..."），由 `coddy serve set-password` 写入。GET /coddy/config 永远不会回显它，从设置屏幕保存时会保留它。请将明文密码放在 CODDY_HTTP_PASSWORD 中，而不是这里。 |
| `httpserver.login.session_ttl_hours` | 整数 | 0 | 浏览器保持登录的时长。0 表示浏览器关闭时丢弃 cookie；服务器仍会在 30 天后自行使会话过期，因为永久保留的记录不是会话。 |
| `httpserver.public_docs` | 布尔值 | false | 启用身份验证时，保持 /docs 和 /openapi.* 无需 token 即可访问。 |
| `httpserver.allow_insecure` | 布尔值 | false | 表明无身份验证的服务器是有意开放的：消除关于它的两条启动警告 - 非环回绑定，以及允许未列出的页面的 CORS（allow_loopback 或 "*"） - 以及镜像这些警告的 --dry-run 发现，并关闭对环回上外来 Host 和跨源写入的拒绝。 |
| `httpserver.cors` | 对象 |  | 跨源访问，使另一个源上的浏览器 UI 可以调用此 API（例如将内置 UI 指向远程服务器）。Bearer 身份验证仍然适用：CORS 决定浏览器是否把响应展示给页面，而从不决定服务器是否给出响应。 |
| `httpserver.cors.enable` | 布尔值 | false | 处理 CORS 预检，并为允许的源发出 Access-Control-* 头。 |
| `httpserver.cors.allow_loopback` | 布尔值 | false | 还允许从浏览器自身机器提供的任何页面：主机为 localhost、*.localhost 名称、127.0.0.0/8 或 [::1] 的 http 或 https 源，任意端口。这是远程 coddy serve 的笔记本电脑场景，其中 Web 界面来自笔记本电脑自身的 coddy serve，且其端口会变化。比 "*" 更窄，并且和 "*" 一样，其安全性仅取决于 API 背后的 token 或登录。 |
| `httpserver.cors.allowed_origins` | 字符串列表 | [] | 允许调用 API 的确切源，例如 "http://localhost:5173". 单个 "*" 允许任何源。 |
| `httpserver.remotes` | 对象列表 | [] | 在 UI 环境选择器中提供、并由 `coddy --remote <name>` 解析的远程 coddy serve 服务器和集群中继。条目可以携带要出示的 token；如果没有，UI 会按远程在浏览器中保存 token，控制台会读取 --remote-token 或 CODDY_REMOTE_TOKEN。 |
| `httpserver.remotes[].name` | 字符串 |  | 远程的显示标签。为空时：Web 界面显示远程报告的名称（中继的 swarm.name，否则为其运行所在的主机名；智能体的主机名），否则显示其地址。 |
| `httpserver.remotes[].url` | 字符串 |  | 远程 coddy serve 服务器或中继的基础 URL，例如 "https://box.example:12345". |
| `httpserver.remotes[].token` | 字符串 |  | 向此远程出示的 bearer token：中继的客户端 token（swarm.auth_token）或服务器的 httpserver.auth_token，通常是 ${ENV} 引用。可选且需主动启用：每个读取此配置的浏览器都会收到它，并且当未提供 --remote-token 时，coddy --remote <name> 会使用它。挂载在中继下的节点（<url>/swarm/nodes/<name>）采用中继的 token。 |

### `swarm`

无状态中继，节点注册到其中，并可链接到其他中继。中继本身需要 -tags swarm 和 swarm.enable；下面的加入列表由任何 coddy serve 进程遵循，这正是普通智能体可通过中继访问的原因。参见 https://coddy.dev/docs/operate/swarm。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `swarm.enable` | 布尔值 | false | 在此进程中运行集群中继。与 swarm.join 无关，后者是智能体注册到父中继的方式。 |
| `swarm.host` | 字符串 | "" | 当 CLI 未传递 --swarm-host 时中继的绑定地址。为空回退到 0.0.0.0。 |
| `swarm.port` | 整数 | 0 | 中继的监听端口。0 回退到 12346。 |
| `swarm.name` | 字符串 | "" | 此中继在拓扑视图、/swarm/info 和子节点路径中的标签。为空：其运行所在的主机名。 |
| `swarm.auth_token` | 字符串 | "" | 客户端向此中继出示的 bearer 凭据。中继使用每个节点自己的凭据访问该节点，因此绑定在非环回地址且未认证的中继是整个集群的敞开大门；除非设置了 allow_insecure，否则服务器拒绝启动。永不回显。 |
| `swarm.pairing_tokens` | 字符串列表 | [] | 节点注册时必须出示的凭据。为空会关闭注册，除非设置了 insecure_open_registration。永不回显。 |
| `swarm.allow_insecure` | 布尔值 | false | 允许在没有客户端 token 的情况下绑定到非环回地址。 |
| `swarm.insecure_open_registration` | 布尔值 | false | 允许任何调用方在没有配对 token 的情况下注册节点。仅用于开发。 |
| `swarm.allow_private_upstreams` | 字符串列表 | [] | 节点可以通告的主机，即使它们解析到环回或私有地址范围；否则这些会被拒绝，以免注册把中继变成对其自身网络的探测。 |
| `swarm.cors` | 对象 |  | 中继 API 的跨源访问。SPA 从构造上就是从另一个源访问中继，因此通常必须开启。Allow-Headers 包含 Last-Event-ID，以便 SSE 流可通过中继恢复。 |
| `swarm.cors.enable` | 布尔值 | false | 处理 CORS 预检，并为允许的源发出 Access-Control-* 头。 |
| `swarm.cors.allow_loopback` | 布尔值 | false | 还允许从浏览器自身机器提供的任何页面：主机为 localhost、*.localhost 名称、127.0.0.0/8 或 [::1] 的 http 或 https 源，任意端口。笔记本电脑的 Web 界面访问此中继，无论其自身的 coddy serve 占用了哪个端口。比 "*" 更窄；客户端 token 仍然适用。 |
| `swarm.cors.allowed_origins` | 字符串列表 | [] | 允许调用 API 的确切源，例如 "http://localhost:5173". 单个 "*" 允许任何源。 |
| `swarm.tls` | 对象 |  | 通过 HTTPS 提供中继。两个文件要么都设置，要么都不设置。最低 TLS 1.2；证书是启动时状态，因此轮换证书需要重启。 |
| `swarm.tls.cert_file` | 字符串 | "" | PEM 证书链。 |
| `swarm.tls.key_file` | 字符串 | "" | PEM 私钥。 |
| `swarm.lease_ttl_seconds` | 整数 | 0 | 注册在没有心跳的情况下存活的时长。0 回退到 90；节点按其三分之一进行刷新。 |
| `swarm.fanout_timeout_seconds` | 整数 | 0 | 聚合调用的每节点截止时间。较慢的节点会降级为警告，而不会阻塞应答。0 回退到 3。 |
| `swarm.upstreams` | 对象列表 | [] | 手动配置而非注册的节点，用于无法运行加入循环的智能体。 |
| `swarm.upstreams[].name` | 字符串 |  | 节点名称；会成为 URL 路径段，因此只能包含字母、数字、下划线和连字符。 |
| `swarm.upstreams[].url` | 字符串 |  | 中继为访问该节点而拨号的源。 |
| `swarm.upstreams[].kind` | 字符串，可选值：`""`, `agent`, `relay` | "" | agent（默认）或 relay。 |
| `swarm.upstreams[].token` | 字符串 | "" | 中继向此节点出示的 bearer 凭据。永不回显。 |
| `swarm.upstreams[].dial` | 对象 |  | 此链路如何到达另一端。中继预期位于不同网络，因此代理跳转和私有证书颁发机构在这里很常见。 |
| `swarm.upstreams[].dial.proxy` | 字符串 | "" | 通过代理路由连接：http、https、socks5 或 socks5h。为空回退到标准环境变量。永不回显。 |
| `swarm.upstreams[].dial.ca_file` | 字符串 | "" | 用于验证证书为私下签名的对端的证书颁发机构。 |
| `swarm.upstreams[].dial.insecure_skip_verify` | 布尔值 | false | 接受任何证书。仅用于实验环境；每次使用都会记录日志。 |
| `swarm.join` | 对象列表 | [] | 此进程在启动时注册到的父中继。无论此进程是否运行自己的中继都会遵循，这正是中继可以链式连接的原因。 |
| `swarm.join[].url` | 字符串 |  | 父中继源。 |
| `swarm.join[].name` | 字符串 | "" | 要在该中继中声明的名称。为空回退到主机名。 |
| `swarm.join[].pairing_token` | 字符串 | "" | 授权注册的凭据。永不回显。 |
| `swarm.join[].advertise_url` | 字符串 | "" | 中继可以访问此进程的位置。留空以选择隧道传输：此进程向外拨号，并通过该连接反向提供其 API，当网络不接受入站连接时，这是唯一的进入方式。 |
| `swarm.join[].token` | 字符串 | "" | 此进程自身的 bearer 凭据，交给中继，以便其代理时进行身份验证。优先使用专为中继签发的凭据。永不回显。 |
| `swarm.join[].labels` | 字符串映射 |  | 在拓扑视图中显示的自由形式标签。 |
| `swarm.join[].dial` | 对象 |  | 此链路如何到达另一端。中继预期位于不同网络，因此代理跳转和私有证书颁发机构在这里很常见。 |
| `swarm.join[].dial.proxy` | 字符串 | "" | 通过代理路由连接：http、https、socks5 或 socks5h。为空回退到标准环境变量。永不回显。 |
| `swarm.join[].dial.ca_file` | 字符串 | "" | 用于验证证书为私下签名的对端的证书颁发机构。 |
| `swarm.join[].dial.insecure_skip_verify` | 布尔值 | false | 接受任意证书。仅限实验环境；每次使用都会记录日志。 |

### `ui`

控制内置的单页 UI（仅在以 -tags http,ui 构建的二进制中有意义）。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `ui.enable` | 布尔值 或 null | true | 在 GET / 提供内嵌 SPA。默认值为 true；设为 false 可运行仅 API 的服务器（配置后 API 仍需要 httpserver.auth_token）。 |

### `scheduler`

Cron 驱动的定时任务（仅在以 -tags scheduler 构建的二进制中使用）。任务是位于 ${CODDY_HOME}/scheduler（用户任务）和 <workspace>/.coddy/scheduler（项目任务，批准后才会运行）中的平铺 *.md 文件，带 YAML frontmatter；使用 UTC 的五字段 crontab。一次运行是该任务自身会话下的一个后台智能体任务，而这个会话就是该任务的运行历史。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `scheduler.enable` | 布尔值 | false | 运行调度器守护进程并暴露 coddy_scheduler_* 工具。也可通过 coddy serve --scheduler（或 coddy acp -scheduler）按进程强制启用。 |
| `scheduler.max_queue` | 整数 | 10 | 所有任务同时正在进行的运行数；超过上限的到期时槽会被跳过，直到有运行完成，而超过上限的手动运行会被拒绝。 |
| `scheduler.timeout` | 字符串 | 30m | 单次运行的实际时间限制，使用 Go 时长表示（例如 "30m"、"1h30m"）；后台任务池仍会以 tools.background.max_timeout_seconds 为其设上限。 |
| `scheduler.project_trust` | 字符串，可选值：`ask`, `allow`, `deny` | ask | 项目任务的信任策略；项目任务指随检出一起携带的 <workspace>/.coddy/scheduler 中的 *.md 文件："ask" 会列出它们，但只有在操作员为该项目工作区批准了该确切文件后才会运行（调度器抽屉中的盾牌，或从该工作区中的会话发送 POST /coddy/scheduler/jobs/{job_id}/trust?scope=project）；操作员通过 Coddy 创建的任务会立即获得批准；"allow" 会像操作员自己的任务一样运行它们；"deny" 则永不运行它们。参见 https://coddy.dev/docs/operate/scheduler#project-jobs-and-trust。 |
| `scheduler.retain_sessions` | 整数 | 5 | 每个 job_id 保留的已完成运行（其任务记录和对话记录位于任务会话下）；当有运行完成时，较旧的运行会被移除。 |

### `gateways`

即时通讯机器人适配器（仅在以 -tags gateway 构建的二进制中使用，或以 -tags gateway.telegram / gateway.pachca 构建以使用单个适配器；由 coddy serve 与所有其他已启用的子系统一起启动）。参见 https://coddy.dev/docs/surfaces/gateway 和 https://coddy.dev/docs/surfaces/pachca。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `gateways.telegram` | 对象 |  | Telegram 机器人适配器。 |
| `gateways.telegram.enable` | 布尔值 | false | 在此 coddy serve 进程中轮询 Telegram。 |
| `gateways.telegram.token` | 字符串 | "" | 来自 @BotFather 的机器人令牌。留空则读取 TELEGRAM_BOT_TOKEN 环境变量（例如通过 ~/.coddy/.env）。 |
| `gateways.telegram.proxy` | 字符串 | inherit | inherit（默认值，等同于省略该键）、none 或代理 URL：http://, https://, socks5:// 或 socks5h://，读取方式与 providers[].proxy 相同。inherit 会遵循 Coddy 进程环境中的代理相关变量 HTTPS_PROXY、HTTP_PROXY 和 NO_PROXY；对回环地址绝不使用代理；不读取 ALL_PROXY。none 会直接连接机器人并忽略这些变量。代理 URL 会将每个 Bot API 请求都通过该代理发送；使用 SOCKS 时由代理解析主机名。 |
| `gateways.telegram.rich_messages` | 布尔值 | false | 使用 Bot API 10.1 Rich Messages（原生 Markdown、流式传输的思考占位符、可折叠的工具列表）。不受支持时会回退到旧版格式。 |
| `gateways.telegram.admins` | 整数列表 |  | 具有提升权限的 Telegram 用户 ID；管理员始终通过访问检查。 |
| `gateways.telegram.default_access` | 字符串 | all | 未设置覆盖的聊天所使用的回退访问级别："all"、"admins" 或 "group:<name>"。 |
| `gateways.telegram.default_isolation` | 字符串，可选值：`individual`, `shared`, `admin` | individual | 群聊的回退会话隔离方式："individual"（每个用户一个会话）、"shared"（所有用户共用一个会话）、"admin"（仅管理员，共用会话）。 |
| `gateways.telegram.user_groups` | 对象列表 |  | 具名用户 ID 集合，可在访问字段中以 group:<name> 引用。 |
| `gateways.telegram.user_groups[].name` | 字符串 |  | 以 group:<name> 引用的组名。 |
| `gateways.telegram.user_groups[].user_ids` | 整数列表 |  | 此组中的 Telegram 数字用户 ID。 |
| `gateways.telegram.chats` | 对象列表 |  | 按聊天设置的覆盖项；群组和超级群的 chat_id 为负数。 |
| `gateways.telegram.chats[].chat_id` | 整数 |  | Telegram 聊天 ID（群组和超级群为负数）。 |
| `gateways.telegram.chats[].isolation` | 字符串，可选值：`individual`, `shared`, `admin` |  | 按聊天设置的会话隔离覆盖项。 |
| `gateways.telegram.chats[].access` | 字符串 |  | 按聊天设置的访问覆盖项："all"、"admins" 或 "group:<name>"。 |
| `gateways.telegram.mini_app` | 对象 |  | 使此 coddy serve 的 Web 界面成为该机器人的 Telegram Mini App。Web 界面会自行适配 Telegram；这些键告诉机器人它所在的位置，以便它设置菜单按钮，并用一个按钮响应 /app，该按钮会打开该聊天自己的对话。Telegram 仅通过 https 打开 Mini App，因此 Web 界面必须发布在 TLS 代理之后，并开启登录（httpserver.login）。参见 https://coddy.dev/docs/surfaces/gateway#mini-app。 |
| `gateways.telegram.mini_app.url` | 字符串 | "" | Web 界面对外提供服务的公共 https 地址，例如 https://coddy.example.com/.。纯 http 仅接受回环主机（例如 tgfake 这样的离线测试台）。不能包含片段：Telegram 会把其启动参数放在那里。留空：机器人不提供 Mini App，并且不会改动在 @BotFather 中设置的菜单按钮。 |
| `gateways.telegram.mini_app.menu_button` | 布尔值 或 null | true | 让机器人的菜单按钮（位于消息输入框旁）打开 Web 界面。关闭此项、清空 url，或 Web 界面不要求登录时，只要该按钮仍打开机器人设置的地址，机器人就会恢复它此前替换掉的菜单按钮（在 @BotFather 中设置的 Mini App，否则为命令列表）。 |
| `gateways.pachca` | 对象 |  | Pachca 集成机器人适配器：读取机器人的事件历史，并通过 Pachca REST API 响应。请在机器人的出站 webhook 设置中开启 "Save events history"（events_history_enabled）。参见 https://coddy.dev/docs/surfaces/pachca。 |
| `gateways.pachca.enable` | 布尔值 | false | 在此 coddy serve 进程中运行 Pachca 机器人。 |
| `gateways.pachca.token` | 字符串 | "" | Pachca 集成机器人的访问令牌。留空则读取 PACHCA_BOT_TOKEN 环境变量（例如通过 ~/.coddy/.env）。 |
| `gateways.pachca.proxy` | 字符串 | inherit | inherit（默认值，等同于省略该键）、none 或代理 URL：http://, https://, socks5:// 或 socks5h://，读取方式与 providers[].proxy 相同。inherit 会遵循 Coddy 进程的 HTTPS_PROXY、HTTP_PROXY 和 NO_PROXY；none 直接连接；代理 URL 会将每个 Pachca API 请求都通过该代理发送。 |
| `gateways.pachca.poll_interval_seconds` | 整数 | 2 | 读取事件历史的频率，单位为秒（1 到 60）。Pachca 对该方法的响应速率约为每两秒五个请求。 |
| `gateways.pachca.admins` | 整数列表 |  | 具有提升权限的 Pachca 用户 ID；管理员始终通过访问检查。 |
| `gateways.pachca.default_access` | 字符串 | all | 未设置覆盖的聊天所使用的回退访问级别："all"、"admins" 或 "group:<name>"。 |
| `gateways.pachca.default_isolation` | 字符串，可选值：`individual`, `shared`, `admin` | individual | 群聊的回退会话隔离方式："individual"（每个用户一个会话）、"shared"（所有用户共用一个会话）、"admin"（仅管理员，共用会话）。 |
| `gateways.pachca.user_groups` | 对象列表 |  | 具名用户 ID 集合，可在访问字段中以 group:<name> 引用。 |
| `gateways.pachca.user_groups[].name` | 字符串 |  | 以 group:<name> 引用的组名。 |
| `gateways.pachca.user_groups[].user_ids` | 整数列表 |  | 此组中的 Pachca 用户 ID。 |
| `gateways.pachca.chats` | 对象列表 |  | 按 Pachca 聊天 ID 设置的按聊天覆盖项（每个话题都有自己的聊天 ID）。 |
| `gateways.pachca.chats[].chat_id` | 整数 |  | Pachca 聊天 ID。 |
| `gateways.pachca.chats[].isolation` | 字符串，可选值：`individual`, `shared`, `admin` |  | 按聊天设置的会话隔离覆盖项。 |
| `gateways.pachca.chats[].access` | 字符串 |  | 按聊天设置的访问覆盖项："all"、"admins" 或 "group:<name>"。 |

### `browser`

Coddy for Browser：扩展连接到 HTTP 服务器，让聊天通过 browser 工具在用户的浏览器中工作，并在其侧边面板中显示 Web 界面；其他 Coddy 进程和集群的聊天通过此服务器访问浏览器。需要带有 browser 构建标签的二进制文件；默认关闭。参见 https://coddy.dev/docs/features/browser。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `browser.enable` | 布尔值 | false | 向此进程的聊天提供 browser 工具。需要 browser.token、browser.remotes 或两者。 |
| `browser.token` | 字符串 | "" | 扩展密钥，至少 32 个字符，独立于 HTTP API 的凭据。凭借它，HTTP 服务器接受扩展（/browser/v1/connect），并且它还用于将扩展的侧边面板登录到 Web 界面，因此要像 API 的令牌一样保密。只写字段。使用环境引用以避免将其写入 YAML。 |
| `browser.serve_remote` | 布尔值 | false | 允许其他 Coddy 进程驱动此处连接的浏览器：本机上的控制台或编辑器、集群中的一个节点。它们使用此服务器的 API 凭据调用 /coddy/browser/v1，并且扩展会在每个此类聊天的第一个操作之前询问用户。需要 browser.token。 |
| `browser.remotes` | 字符串列表 | [] | 此进程的聊天除自身之外还可以使用其浏览器的 Coddy 服务器：httpserver.remotes 中的名称，或服务器地址，例如集群中继的节点挂载（https://relay:8443/swarm/nodes/home），其令牌的查找方式与 --remote 相同。该服务器需要 browser.serve_remote。 |

### `decisions`

在权限门控做出决定之前，询问 NeuralDeep decisions API 某条 shell 命令是否安全：安全的命令无需询问即可运行；不安全的命令在每种模式下都会询问，包括 bypass 模式；在无人能应答的情况下则拒绝。参见 https://coddy.dev/docs/features/decisions。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `decisions.enable` | 布尔值 | false | 在权限门控做出决定之前，将操作者未显式允许的 run_command 和 ssh_run_command 调用与 decisions 端点进行核对。安全命令无需询问即可运行；不安全的命令，或检查未给出裁决的命令，在每种模式下都会询问，包括 bypass 模式，裁决会放在提示中，任何模式或界面都不会自行应答该提示，无人能应答时则拒绝，拒绝结果作为工具结果返回，以便会话记录。由允许列表、对恰好该命令的会话授权或钩子的 allow 批准的命令不进行检查，操作者在提示中批准的命令也不检查；程序范围或前缀授权不会使命令免于检查。需要在类型为 neuraldeep 的第一个提供商行上提供 NeuralDeep 凭据：其 api_key 或 api_key_command、其 NAME_API_KEY 环境变量（对于名为 neuraldeep 的行或没有行时，为 NEURALDEEP_API_KEY），或其存储的 hub 登录信息。 |
| `decisions.model` | 字符串，可选值：`frida-decisions`, `clef-flash` | frida-decisions | 要询问的 decisions 模型：frida-decisions（编码器前向，最多 512 个 token 的命令文本，每次请求约 20 ms）或 clef-flash（最多 8192 个 token，约 150 ms）。 |
| `decisions.threshold` | 数字 | 0.5 | 不安全选项的概率达到或超过该值时，命令被视为不安全。0 使用默认值（0.5）；较低值更严格，较高值会让边缘命令通过。仅返回 unsafe 的答案按 1 计算。 |

### `metrics`

Prometheus 导出器：在独立端口上提供 GET /metrics，由 coddy serve 运行。仅在以 -tags metrics 构建的二进制中生效；其他二进制会拒绝 metrics.enable。默认关闭。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `metrics.enable` | 布尔值 | false | 在此进程中运行导出器。需要 metrics 构建标签。 |
| `metrics.host` | 字符串 | "" | 导出器的绑定地址。留空则回退到 127.0.0.1，仅本机可访问。 |
| `metrics.port` | 整数 | 0 | 导出器的监听端口。0 会回退到 12347。 |
| `metrics.token` | 字符串 | "" | 抓取时必须在 Authorization 请求头中提供的 Bearer 凭据。${ENV} 引用会在加载时展开。留空表示不进行身份验证，因此在绑定到非回环地址前应设置一个。绝不会回显。 |

### `supervisor`

使用第二个模型在每轮后检查会话目标（/goal），在限定范围内继续未完成的工作，并监视停滞或循环的轮次；参见 https://coddy.dev/docs/features/session-supervisor。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `supervisor.enable` | 布尔值 | false | 即使没有 /goal，也会根据最新请求检查普通轮次。无论此开关如何设置，会话目标都会被检查。 |
| `supervisor.model` | 字符串 | "" | 用于检查工作的已配置 models[].model。留空则使用会话自身的模型，此时它会评判自己的工作；另一家族的模型是更好的评判者。 |
| `supervisor.verify` | 布尔值 或 null | true | 在目标关闭前确认已满足的判定：goal-verifier 子智能体使用只读工具读取工作区，并检查每一项要求。未设置表示 true。 |
| `supervisor.stall_seconds` | 整数 或 null | 300 | 目标轮次可以没有任何更新地持续多少秒，之后看门狗会切断它并启动恢复轮次。权限提示、正在运行的工具和正在运行的后台任务会暂停计时器。0 表示禁用。 |
| `supervisor.max_nudges` | 整数 或 null | 2 | 每次监督器运行中，在停滞、工具循环或轮次失败后可用的恢复轮次数。 |
| `supervisor.max_continuations` | 整数 或 null | 10 | 一个目标可以使用的自动继续轮次数；用完后，该目标会获得一个收尾轮次，并以受限状态停止。 |
| `supervisor.loop_repeat` | 整数 或 null | 3 | 相同工具操作（相同调用、相同结果）的重复循环可以出现多少轮，之后看门狗会切断该轮次。0 表示禁用循环检测。 |
| `supervisor.token_budget` | 整数 或 null | 0 | 一个目标的各轮次可以消耗的未缓存输入 token 与输出 token 的总数；0 表示不设上限。 |
| `supervisor.max_chain_sessions` | 整数 或 null | 10 | 一个目标链（/goal --then-file）可以依次打开的会话数；当最后一个会话的目标得到满足时，该链停止。0 会关闭目标链。 |

### `toolsets`

工具列表的具名预设。models[] 行通过 toolset 命名一个，会话可以使用 /toolset <name> 选择一个，既可单独使用，也可在同一行中跟在 /model 之后。预设只会收窄范围：模式、子智能体定义或开关已经隐藏的工具仍保持隐藏，对集合之外工具的调用会被拒绝且不会出现权限提示。它是上下文优化，不是安全边界。

| 键 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `toolsets` | 对象列表 |  | 工具列表的具名预设。models[] 行通过 toolset 命名一个，会话可以使用 /toolset <name> 选择一个，既可单独使用，也可在同一行中跟在 /model 之后。预设只会收窄范围：模式、子智能体定义或开关已经隐藏的工具仍保持隐藏，对集合之外工具的调用会被拒绝且不会出现权限提示。它是上下文优化，不是安全边界。 |
| `toolsets[].name` | 字符串 |  | 预设名称：字母、数字、'.'、'_' 和 '-'。default 是保留名称：会话选择它会获得该模式提供的每个工具。 |
| `toolsets[].tools` | 字符串列表 |  | 允许列表：工具名称、裸 *，或 prefix*（context7__* 允许该 MCP 服务器的每个工具）。不存在或为空则允许该模式会允许的每个工具。coddy -t 会对不匹配任何工具的条目发出警告。 |
| `toolsets[].disallowed_tools` | 字符串列表 |  | 在 tools 之后应用的拒绝列表，使用相同的名称和模式；同时出现在两个列表中的名称会被移除。coddy -t 会对不匹配任何工具的条目发出警告。 |
<!-- docsgen:config:end -->

## 说明

### `providers`

LLM 后端列表（`[]config.ProviderConfig`、`internal/config/providers.go`）。

密钥解析顺序：`api_key` → `api_key_command` stdout → `NAME_API_KEY` 环境变量。`codex` 行使用 ChatGPT 登录且不读取它们中的任何一个。

```yaml
providers:
  - name: openai
    type: openai
    api_key: "${OPENAI_API_KEY}"
  - name: local
    type: openai
    api_base: "http://localhost:11434/v1"
    api_key: "~"
  - name: neuraldeep
    type: neuraldeep
    api_key: "${NEURALDEEP_API_KEY}"
    # usage_limits_panel: false  # hide the account usage panel for this row
  - name: codex
    type: codex # use Sign In with ChatGPT in the bundled web UI; no api_key needed
```

#### 将 llama.cpp 作为 OpenAI 兼容提供商

`llama-server` 可作为 `type: openai` 提供商（`api_base: "http://host:8080/v1"`）使用。推荐的启动标志：

- `--jinja` - 在服务器上启用模型的聊天模板，这是 **工具调用** 所必需的。没有它，llama.cpp 会静默忽略 `tools` 参数，智能体循环会退化为纯文本答案。
- `-c <n>` - 将上下文窗口设置为足够大以容纳智能体提示词（系统提示词加工具 schema 加历史；16k 是实际最低值，越大越好）。当请求超过服务器上下文时，llama.cpp 报告 `the request exceeds the available context size` - 提高 `-c` 或修剪 `max_context_tokens`。

截至 2025 年的 llama.cpp 构建在流中途失败时会报告非标准 SSE `error:` 字段；Coddy 理解该方言和当前的 `data: {"error": ...}` 形状，并在错误中显示服务器消息。

对于 `type: codex`，在捆绑的 Web UI 中打开 **设置 → LLM 提供商** 并选择 **Sign In with ChatGPT**，或为 ACP 和无头设置运行等效的终端命令：

```bash
coddy providers login codex    # device flow: prints a URL and a one-time code, then waits
coddy providers list           # every provider with the credential its requests actually use,
                               # plus a warning for an openai row that has neither api_base
                               # nor a credential (its requests leave for api.openai.com)
coddy providers logout codex   # removes the Coddy-managed credential (leaves the Codex CLI login alone)
```

（直到 1.0.11，同一流程还有一个仅 codex 的命令 `coddy codex login | status | logout`。它已被上面的命令取代，这些命令覆盖每个后端；`coddy providers list` 报告 `codex status` 过去报告的内容。）

成功登录还会将订阅目录发布到 `config.yaml`，方式类似 `coddy providers login neuraldeep` 发布其层级模型（设置按钮仅存储凭据）：它会在缺失时添加 `codex` 提供商行，为 Codex 列出的每个目录模型添加一个 `models[]` 条目（Codex 从其自己的选择器中隐藏的 id，例如 `gpt-reserve` 和 `codex-auto-review`，会被排除），并在尚未设置任何内容时设置 `agent.model` - Codex 自己排名最高的模型。文件中已有的任何内容都不会被重写：现有的提供商行、已列出的模型和已选择的 `agent.model` 都会原样保留，因此重复登录是无操作。唯一的例外是提供商无法回答的 `agent.model` - 示例配置中的 `openai` 行且其密钥未设置 - 登录会将其移动到它添加的 codex 模型，并说明这一点。`--no-config` 仅存储凭据。条目不携带 `max_tokens`，因为 Codex 后端拒绝 `max_output_tokens`。正在运行的 `coddy serve` 保持其已加载配置，因此请重启它以拾取新模型。

两条路径使用同一存储；提供商名称是参数，因此当 `config.yaml` 定义多个时，`coddy providers login <name>` 会针对特定的 codex 行（名称不是 `codex` 的行要么存在于 `config.yaml` 中，要么由登录用 `--type codex` 创建）。每个 codex 行保留自己的登录；Codex CLI 登录仅作为唯一 codex 行的替代，或在多个行中名为 `codex` 的行的替代（[一个提供商类型的多个配置文件](../getting-started/configuration.md#同一提供商类型的多个配置文件)）。Coddy 使用官方设备授权流程，并将可刷新凭据存储在 `$CODDY_HOME/providers/<provider-name>/codex-auth.json`，具有限制性文件权限；令牌从不进入 `config.yaml`。`api_key`、`api_key_command` 和 `api_base` 对 Codex 忽略（留在 codex 行上的 `api_key_command` 从不运行），而 `proxy` 适用于 OAuth 和提供商请求。Codex 模型没有 Coddy 可设置的输出上限：`Unsupported parameter` 是后端对 `max_output_tokens` 的回答，因此 codex 模型上的 `models[].max_tokens` 不限制任何内容。它仍被接受 - 设置表单会在每个模型行上预置它 - 但 `coddy -t`、`--dry-run` 和 `coddy serve` 的启动日志会警告它，而在 `POST /v1/chat/completions` 上为 codex 模型发送的 `max_tokens` 会以 `400` 拒绝。`max_context_tokens` 是另一回事：上下文环和自动压缩所衡量的窗口，不是提供商强制的限制。在 codex 模型上未设置时，它是 Codex 目录为该模型报告的窗口 `context_window`（它今天服务的每个模型为 272000）（[上下文窗口](../features/compaction.md#上下文窗口)）。模型选择器使用保存的令牌从官方 Codex 后端读取目录。后端仅从模型的 `minimal_client_version` 起向 Codex 发布提供每个模型，因为较旧的 Codex 二进制无法运行它；Coddy 通过自己的循环和工具运行每个模型，因此它作为 Codex 发布 0.999.0 请求，远高于门控命名的发布，并列出账户被提供的每个模型，包括最新的，无论安装了哪个 Codex CLI（如果有）。Codex 从其自己的选择器中隐藏的模型不在列表中。如果不存在 Coddy 管理的凭据，Coddy 仍与 `~/.codex/auth.json`（或 `$CODEX_HOME/auth.json`）中的 Codex CLI 登录兼容。Codex 请求始终指向官方后端；进程级 `CODDY_CODEX_BASE_URL` 是唯一的覆盖（测试和自托管网关），因此设置文档永远不能自行重定向 OAuth 令牌。

Codex 只是模型后端：智能体保留 Coddy 自己的系统提示词、工具目录、权限和 ReAct 循环，ChatGPT 凭据仅用于认证 Responses 调用（`features/codex_auth.feature` 在 HTTP 和 ACP 界面上都固定这一点）。

**令牌生命周期。** 访问令牌会在过期前不久透明刷新，刷新后的令牌会写回其来源文件。当凭据是 Codex CLI 登录（对于该登录服务的唯一行）时，该文件就是 `~/.codex/auth.json` 本身 - 与 `codex` CLI 读取的文件相同，因此两个工具可共用一个登录，Coddy 执行的刷新对 CLI 可见（反之亦然）。Coddy 管理的凭据在 `$CODDY_HOME/providers/<name>/` 下就地刷新，从不触碰 CLI 登录。注销（`coddy providers logout codex`，或设置中的 **Sign Out**）仅移除 Coddy 管理的文件。

**启动报告。** 当至少配置了一个 `type: codex` 提供商时，`coddy acp` 和 `coddy serve` 会在启动时为每个提供商记录一行 `codex credential`：凭据来自何处以及访问令牌还有效多久。缺少凭据、不可用的 `auth_mode`，或已过期且无剩余刷新令牌的令牌会记录为 **warning** 并指出 `coddy providers login <name>`；已过期但可刷新的令牌只是信息行，因为下一个请求会续期。没有 codex 提供商的设置不记录任何内容。

**推理。** Codex 后端提供 `gpt-5*` 和 `gpt-6*` 模型 id，但仅接受 `none`、`low`、`medium`、`high` 和 `xhigh`，因此 codex 支持的模型在其他提供商提供 `minimal` 的地方提供 **`none`**（显式 `reasoning_levels: [minimal]` 也会重映射）。推理轮次请求摘要（`summary: auto`），以便思考流式进入 UI；并请求加密推理（`include: reasoning.encrypted_content`），以便模型自己的思维链在同一轮次的下一个请求中原样重放 - 与 Codex CLI 使用的流程相同。重放的推理会标记产生它的模型，并在会话切换模型时跳过。这些项以不透明方式存储在 `messages.json`（`reasoning_signature`，每个助手轮次约 1 KB），且不由 `GET /coddy/sessions/{id}/messages` 暴露。

### `models`

逻辑模型列表（`[]config.ModelEntry`、`internal/config/models.go`）。

```yaml
models:
  - model: "openai/gpt-5.6-terra"
    max_tokens: 8192
    reasoning_default: medium
    multimodal: true
  - model: "openai/gpt-5"
    max_tokens: 8192
    reasoning_default: medium
  - model: "local/qwen3-30b"
    max_tokens: 8192
    stream: false
```

**非流式模型。** `stream: false` 改变一件事：coddy 发送单个阻塞 `POST /chat/completions` 而不是请求 SSE，并将完成的答案一次性交给运行时其余部分。对于 Anthropic 模型，请求是 `POST /v1/messages` 且不带 `stream`。其截止时间随 `max_tokens` 增长（每 128000 token 一小时，从十分钟到一小时），因此客户端无论 `max_tokens` 是什么都会发送它。下游一切不变，因此对话记录、工具调用和会话包看起来相同；不同之处在于模型完成前不显示任何内容，思考行不显示实时持续时间。开启前有两个后果值得了解。在阻塞调用期间按 **Stop** 会丢失整个答案，因为服务器尚未发送任何内容，而流式轮次会保留已到达的 token。请求 SSE 流的客户端仍会得到流 - 该开关管理到 LLM 的连接，而不是到客户端的连接 - 这就是为什么流式 HTTP 响应现在每 15 秒携带 keepalive 注释，以免代理丢弃静默数分钟的轮次。

该开关对 `type: codex` 提供商拒绝。Codex Responses 后端没有阻塞模式，因此在那里遵循该键意味着仍然流式传输，只是假装不流式。

### `agent`

ReAct 循环设置（`config.Agent`、`internal/config/agent.go`）。

### `prompts`

系统提示词模板覆盖（`config.Prompts`、`internal/config/prompts.go`）。模板字段在 [配置](../getting-started/configuration.md#完整配置-schema) 中记录。

### `instructions`

操作员添加到提示词中的指令文件（`config.Instructions`、`internal/config/instructions.go`）。默认列表为空，且只做添加：智能体主目录、会话文件夹以及工具进入的文件夹的 `AGENTS.md` 和 `DESIGN.md` 无论是否命名都会读取，其文件按列出顺序排在这些之后。命名提示词已包含文件（例如会话文件夹自己的 `AGENTS.md`）的条目会被跳过。见 [rules.md](../features/rules.md#agentsmd-与-designmd)。

### `skills`

技能发现（`config.Skills`、`internal/config/skills.go`）。

### `rules`

规则发现（`config.Rules`、`internal/config/rules.go`），来自工作区文件夹和操作员自己的 `${CODDY_HOME}/rules`。见 [rules.md](../features/rules.md)。

### `mcp`

不绑定到单个服务器条目的 MCP 设置（`config.MCP`、`internal/config/mcp.go`）。

服务器本身不在 `config.yaml` 中声明。它们位于两个 Cursor 兼容文件中，一个以服务器名称为键的单一 `mcpServers` 对象（`env` 和 `headers` 是 JSON 对象，每工具开关使用 `disabledTools`）：用户全局的 `${CODDY_HOME}/mcp.json`（默认 `~/.coddy/mcp.json`，类似 Cursor 的 `~/.cursor/mcp.json`，"global" 范围）和项目本地的 `<workspace>/.coddy/mcp.json`（"local" 范围），后者在名称上优先于全局文件，其条目需要工作区批准后才能启动。旧的在 `config.yaml` 中的 `mcp_servers` 列表会在下次加载时移入 `${CODDY_HOME}/mcp.json`，并保留该文件的 `config.yaml.bak-<time>` 副本，`coddy -t` 会将其报告为警告。见 [mcp.md](../features/mcp.md)。

为 [issue #80](https://github.com/coddy-project/coddy-agent/issues/80) 添加。
批准记录在 `~/.coddy/mcp-trust.json` 中，以规范工作区路径（链接的 git worktree 使用其主检出路径）和承载命令的声明（传输、命令、参数、env、url、标头）的摘要为键，因此重写已批准的条目会再次询问。使用 `coddy mcp trust <name>`、`POST /coddy/mcp/{name}/trust` 或 **设置 → MCP 服务器** 中的盾牌按钮批准。该标签页也会编辑此策略本身（`POST /coddy/mcp/project-trust`），因此它不会渲染为单独的设置部分。见 `docs/features/mcp.md`。

### `tools`

权限策略（`config.Tools`、`internal/config/tools.go`）。

#### `tools.output_limits`

工具结果或错误可返回到 LLM 上下文的最大行数（`config.ToolOutputLimits`）。每个启用的行限制还会应用硬性 **64 KiB 每次调用字节上限**，防止压缩文件、base64 负载或单行 MCP 响应绕过防护。`0` 禁用该工具的两个限制；未设置字段回退到内置默认值。截断输出以标记结束，告诉模型如何获取其余部分（`read` 的 `offset`/`limit`，`grep` 的较窄模式，`websearch` 的 `page`）。

#### `tools.background`

后台执行的边界（`config.ToolBackground`）。后台化的 `run_command` 返回任务 id 而不是输出；`background_list`、`background_output`、`background_wait` 和 `background_stop` 稍后收集结果。任务池位于运行中的 `coddy` 进程内：每个任务将其元数据和捕获输出镜像到 `background/<task_id>/` 下的会话包中，被重启中断的任务报告为 `orphaned`，而不是仍在运行。任何整数字段上的 `0` 表示“使用默认值”。见 `docs/features/background-tasks.md`。

### `subagents`

子智能体（`config.Subagents`、`internal/config/subagents.go`）：模型通过 `spawn_agent` 工具委派的子智能体。定义是带 YAML frontmatter 的 markdown 文件（`name`、`description`、`model`、`reasoning`、`mode`、`tools`、`disallowed_tools`、`permission_mode`、`max_turns`、`timeout_seconds`、`background`、`hidden`），其正文是子智能体的角色。每次运行都是父会话的后台任务，拥有自己的子会话和对话记录，因此 `background_list` / `background_output` / `background_wait` / `background_stop`、Tasks 面板和 `GET /coddy/sessions/{id}/background-tasks` 都能看到它。`max_concurrent`、`default_timeout_seconds` 和 `max_turns` 上的 `0` 表示“使用默认值”；`max_depth` 是例外，省略它使用默认值 `1`，因为显式 `0` 会在所有地方禁止生成。见 `docs/features/subagents.md`。

项目范围定义的批准记录在 `~/.coddy/subagents-trust.json` 中，以规范工作区路径（链接的 git worktree 使用其主检出路径）、定义名称和文件摘要为键，因此编辑已批准的文件会再次询问。定义中的 `permission_mode`、`tools` 和 `disallowed_tools` 只能在每个范围中收窄父智能体可做的事情。

### `hooks`

钩子（`config.Hooks`、`internal/config/hooks.go`）：在会话生命周期点运行的操作员命令。钩子从 stdin 读取一个 JSON 文档，并以退出码加 stdout 上的可选 JSON 回答；`PreToolUse` 钩子可以无论权限模式如何拒绝工具调用、越过提示批准它、强制提示、重写其参数或添加上下文，而 `PostToolUse` / `PostToolUseFailure` 钩子可以添加反馈。定义是 Claude Code 形状的 JSON 文件。每个整数键上的 `0` 表示“使用默认值”。见 `docs/features/hooks.md`。

项目范围文件的批准记录在 `~/.coddy/hooks-trust.json` 中，以规范工作区路径（链接的 git worktree 使用其主检出路径）、工作区相对文件路径和文件摘要为键，因此编辑已批准的文件会再次询问。

### `logger`

日志记录（`config.Logger`、`internal/config/logger.go`）。设置后，ACP 标志 `--log-level`、`--log-output`、`--log-file`、`--log-format` 会覆盖这些。

`levels` 一次提高或降低一个子系统的详细程度，因此追查不工作的 Telegram 命令不再意味着将整个进程调到 `debug` 并从其他所有内容中读取它：

```yaml
logger:
  level: "info"
  levels:
    - component: "gateway.telegram"
      level: "debug"
```

组件是子系统标记其日志记录器的点分名称，并且会保留在该子系统写入的每条记录上，因此文件也可以在事后按子系统过滤。父名称覆盖其下所有嵌套项，最长的已配置前缀胜出：`gateway` 到达 `gateway.telegram`，除非该名称有自己的条目。来自进程未标记部分的记录没有组件，并遵循 `level`。将同一组件配置两次是错误，而不是静默胜出者。

`--log-level` 接受相同规范作为逗号分隔列表，这就是在 systemd 下运行的操作员可以仅为一次重启提高一个子系统而无需编辑文件的方式：

```bash
coddy serve --log-level "info,gateway.telegram=debug"
```

裸 `--log-level debug` 仅设置根级别，并保持已配置条目不变；命名组件的规范会替换它们，因此该标志是记录内容的完整陈述。

### `sessions`

会话包存储（`config.Sessions`、`internal/config/sessions.go`）。

### `compaction`

上下文压缩（`config.Compaction`、`internal/config/compaction.go`）：总结较早的对话历史，使长会话持续适配模型上下文窗口。适用于手动 compact 命令和自动阈值触发。

#### `compaction.result_eviction`

在构建 LLM 请求时（`config.ResultEviction`），将未标记的 `read`/`grep` 工具结果折叠为短占位符，因此分页大文件或运行宽泛搜索不能在后续每个轮次中钉住死行。当模型标记它（`keep_result` 工具，或调用上的 `keep: true`）或它位于最近工作窗口内时，结果会保留；对文件的文件系统写入会使覆盖它的较早 read/grep 失效。持久化会话包完整保留每个结果。

### `memory`

长期记忆子智能体（`config.MemoryConfig`、`internal/config/memory.go`；实现在 `external/memory`，`memory` 构建标签）：捆绑的 `memory` 子智能体定义，模型通过 `spawn_agent` 按需委派，其模型和回退链、召回和持久化边界以及笔记存储（[长期记忆](../features/memory.md)）。

### `httpserver`

OpenAI 兼容 HTTP API 默认值（`config.HTTPServerConfig`、`internal/config/http.go`；`http` 构建标签）。见 [http-api.md](http-api.md)。

### `swarm`

无状态中继，节点注册到其中，并可链接到其他中继（`config.SwarmConfig`、`internal/config/swarm.go`；服务器端需要 `swarm` 构建标签，尽管 `swarm.join` 由每个 `coddy serve` 进程遵循）。见 [swarm.md](../operate/swarm.md)。

### `ui`

嵌入式 Web UI（`config.UIConfig`、`internal/config/ui.go`；仅与 `-tags http,ui` 一起有意义）。

### `scheduler`

Cron 调度器（`config.SchedulerConfig`、`internal/config/scheduler.go`；`scheduler` 构建标签）。任务文件格式在 [配置](../getting-started/configuration.md#调度器可选构建) 中描述。

### `gateways`

信使网关（`config.GatewayConfig`、`internal/config/gateway.go`；`gateway` 或 `gateway.telegram` 构建标签；使用 `coddy serve` 运行）。见 [gateway.md](../surfaces/gateway.md)。

#### `gateways.telegram`

## 相关环境变量

这些变量控制配置发现本身，而不是单个字段（见 [配置](../getting-started/configuration.md#配置文件位置与路径)）：

| 变量 | 等效标志 | 含义 |
|---|---|---|
| `CODDY_HOME` | `--home` | 智能体状态目录（默认 `~/.coddy`）。 |
| `CODDY_CWD` | `--cwd` | 默认会话工作目录。 |
| `CODDY_CONFIG` | `--config` | `config.yaml` 的显式路径。 |
| `CODDY_SWARM_TOKEN` | `--swarm-auth-token` | `coddy serve` 运行的中继的客户端凭据（见 [`swarm`](#swarm)）。 |
| `CODDY_SWARM_PAIRING_TOKEN` | `--swarm-pairing-token` | 该中继的注册凭据（见 [`swarm`](#swarm)）。 |
| `NAME_API_KEY` | - | 每个提供商的 API 密钥回退（见 [`providers`](#providers)）。 |
| `TELEGRAM_BOT_TOKEN` | - | Telegram 机器人令牌回退（见 [`gateways.telegram`](#gatewaystelegram)）。 |

<!-- docsgen:source sha256=55e329f3692b9b51 -->

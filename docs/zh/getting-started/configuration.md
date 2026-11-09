# 配置参考

本页是叙述性指南。两个配套文件涵盖完整键列表：

- **[config-reference.md](../reference/config.md)** - 逐字段表格：类型、默认值、环境变量回退、必填/可选、示例。
- **[config.schema.json](../../../internal/config/config.schema.json)** - 用于编辑器自动补全和校验的 JSON Schema（draft-07），发布在 **https://coddy.dev/config.schema.json**，因此编辑器无需检出代码即可解析它，并且它被嵌入二进制文件，使 `coddy -t` 按同一文档检查文件（见[从命令行检查文件](#从命令行检查文件)）。任何带 YAML 语言服务器的编辑器（VS Code YAML 扩展、Zed、Neovim、Helix）都会在文件带有以下头部行时，在你输入时校验键和值：

```yaml
# yaml-language-server: $schema=https://coddy.dev/config.schema.json
```

**Coddy 会自行写入该行。** 每次重写 `config.yaml` 的保存——设置屏幕（`PUT /coddy/config`）、技能来源、智能体自身的 `config_set` / `config_commit`——都会在文件没有头部时添加该头部，并保留你自己选择的 `$schema`（固定标签、本地路径）不动。同样的保存会保留你的注释，包括被注释掉的键、键已有的顺序，以及你书写每个未被更改值的方式（见[环境变量引用](#环境变量引用)）。保存会写入文件已有的键，以及任何确实与内置默认值不同的内容——从未设置的可选字段会被完全省略，而不是写成 `null`，因此将整节保持注释状态的文件会保持原样，提供商或模型条目只保留其明确命名的字段。JetBrains IDE 不读取该头部；如果 `config.yaml` 在那里未被校验，请在 **设置 - 语言与框架 - 架构和 DTD - JSON 架构映射** 下手动映射同一 URL。VS Code 可以在不修改文件的情况下获得相同配置：

```json
"yaml.schemas": { "https://coddy.dev/config.schema.json": ["**/.coddy/config.yaml"] }
```

该 schema 由 `TestDocsConfigSchemaMatchesStructs`（`internal/config/docs_schema_test.go`）与 Go 配置结构体保持同步；当新增或重命名配置字段而未更新 schema 时，CI 会失败。从 `coddy.dev` 提供的副本是该文件的逐字镜像，保存在[站点仓库](https://github.com/coddy-project/coddy-project.github.io)中。

## 配置文件位置与路径

解析后的位置使用环境变量和标志（见 README）。简而言之：

- **`CODDY_HOME`** - 智能体状态目录。默认 **`~/.coddy`**。存放 `config.yaml`、`sessions/`、`skills/`、Coddy 管理的提供商凭据（位于 `providers/` 下），以及使用可选 cron 调度器时的 **`scheduler/`**。
- **`CODDY_CWD`** - 当 `session/new` 发送空的 `cwd` 时默认的文件系统 cwd。默认是启动时进程的工作目录。设置时与 **`--cwd`** 标志含义相同。
- **`CODDY_CONFIG`** - `config.yaml` 的显式路径。与 **`--config`** 相同。
- **`KIMI_CODE_HOME`** - Kimi CLI 状态目录，当 Coddy 没有托管凭据时，由符合条件的 `type: kimi` 订阅行使用。默认 `~/.kimi-code`；见 [Kimi](../features/kimi.md)。
- **`CODEX_HOME`** - Codex CLI 状态目录，当不存在 Coddy 管理的凭据时，由 **`type: codex`** 提供商读取：唯一的 codex 行，或在多行中名为 **`codex`** 的行（[同一提供商类型的多个配置文件](#同一提供商类型的多个配置文件)）。默认 **`~/.codex`**。
- **`CODDY_CODEX_BASE_URL`** - 覆盖 Codex 后端端点（默认 **`https://chatgpt.com/backend-api/codex`**）。有意保持在进程级：**`api_base`** 对 **`type: codex`** 仍被忽略，因此设置文档不能重定向 ChatGPT OAuth 令牌。供可执行规范及自托管 Codex 网关使用。
- **`CODDY_DEVIN_CLI_CREDENTIALS`** - Devin CLI 的 **`credentials.toml`**，当不存在 Coddy 管理的登录时，由 **`type: devin`** 提供商读取。默认 **`~/.local/share/devin/credentials.toml`**（或位于 **`$XDG_DATA_HOME`** 下）。**`CODDY_DEVIN_API_SERVER_URL`**、**`CODDY_DEVIN_WEBAPP_URL`** 和 **`CODDY_DEVIN_API_URL`** 为整个进程移动 Devin 端点，用于测试环境和测试；见 [Devin](../features/devin.md)。

如果未给出 **`--config`**，加载器使用 **`$CODDY_HOME/config.yaml`**（默认主目录 **`~/.coddy`**）。如果该文件缺失，则应用内置默认值（不报错）。进程当前工作目录中的 **`config.yaml`** 不会自行被读取：它可能随代码检出而来，而配置可以运行命令（**`providers[].api_key_command`**、**`providers[].acp.command`**），并指定密钥和提示词的去向（**`providers[].api_base`**、**`swarm.join`**）。如果启动时发现一个这样的文件而主目录中没有，日志会说明，**`coddy -t`** 会指出它。要使用该文件，请用 **`--config`** 或 **`CODDY_CONFIG`** 指定它，或将其移动到主目录。

当主文件存在但无效（YAML 解析或校验错误）时，加载器会自动从同一目录中的 **`config.yaml.bak`** 恢复（见 **`internal/config/recovery.go`**）。每次成功加载后，服务器会写入 **`config.yaml.bak`**。HTTP **`PUT /coddy/config`** 路由（见 **`docs/reference/http-api.md`**）在覆盖前也会将当前文件快照到 **`config.yaml.bak`**，因此重新加载失败时可以回滚。

该文件保存提供商密钥和令牌，因此 Coddy 会将其写入 `config.yaml`、其备份和 `mcp.json` 为仅所有者可读（模式 `0600`），无论文件之前是什么模式。手动或旧版本创建的文件会保持其模式，直到 Coddy 保存它；当机器上的其他用户可以读取它时，`coddy -t` 会发出警告，`chmod 600 ~/.coddy/config.yaml ~/.coddy/config.yaml.bak` 会立即关闭它。

`coddy acp` 子命令还接受 **`--home`**（覆盖 `CODDY_HOME`）、**`--sessions-dir`** 和 **`--session-id`**。YAML 中可选的 **`sessions.dir`** 在未设置 **`--sessions-dir`** 时覆盖会话根目录（默认 **`$CODDY_HOME/sessions`**）。

## 从命令行检查文件

每个加载 `config.yaml` 的命令也接受 `-t`（长格式 `--test-config`）：裸 `coddy -t`、`coddy cli -t`、`coddy acp -t` 和 `coddy serve -t`。该标志检查该命令将加载的文件——`--config PATH` 和 `--home DIR` 像启动时一样精确选择它，`~/.coddy/.env` 会先被加载并展开 `${VAR}` 引用——然后退出，不启动任何内容。它从不写入：正常加载对损坏文件执行的 `config.yaml.bak` 恢复（见上文）不会运行，因此告知你的文件就是磁盘上的文件。该标志适用于所有构建，包括没有 `cli` 标签的精简构建。

检查分两个阶段。首先，文档会按上面的 JSON Schema 校验，即编辑器使用的同一 schema，已嵌入二进制。这能捕获加载器静默接受的错误：`config.yaml` 被宽松解码，因此未知或拼写错误的键（`enable` 写成 `enabled`）会被忽略而不是拒绝，形状错误的值、超出枚举或范围的值、缺失的必填键或重复键会在之后表现为异常行为。然后，加载器自身的规则在解析后的文档上运行：模型指定了不存在的提供商、`file` 输出没有 `logger.file`、`models` 中缺少 `agent.model`。每个问题都会打印为 `file:line:column: what is wrong`，并带一行缩进的 `fix:` 说明如何纠正，以及（如果 schema 中有）一行 `doc:` 携带字段描述：

```text
$ coddy serve -t
/home/me/.coddy/config.yaml:13:3: httpserver.enabled: unknown key "enabled" (the loader ignores it, so it has no effect)
    fix: did you mean "enable"? keys allowed under httpserver: allow_insecure, auth_token, cors, enable, host, port, public_docs, remotes
    doc: Serve the HTTP API (and the embedded SPA) in this process. Omitted means true; set false on a node that only polls a messenger or relays a swarm.
/home/me/.coddy/config.yaml:16:10: logger.level: "verbose" is not an allowed value
    fix: use one of debug, info, warn, warning, error
    doc: Minimum severity written to the configured outputs ("warning" is accepted as an alias of "warn").
/home/me/.coddy/config.yaml:18:11: warning: subagents.enable: "yes" is read as the boolean true, but the schema and editors expect true or false
    fix: write enable: true (true or false, unquoted)
/home/me/.coddy/config.yaml: 2 errors, 1 warning
config test failed
```

文件有错误时退出状态为 1，否则为 0，因此该标志适合放在部署脚本中 `coddy serve restart` 之前。警告（标记为 `warning:`）从不会导致检查失败：它们标记加载器仍会读取但 schema 和编辑器拒绝的写法——布尔值写成 `yes`，整数写成 `40.0`——文件没有 `# yaml-language-server:` 头部、机器上其他用户可读的文件（修复方法是关闭它的 `chmod 600`），以及提供商从不发送的设置：由 `codex` 提供商提供的模型上的 `max_tokens` 不会限制任何内容，因为 Codex 后端不接受输出上限。加载器仍会接受该项，因为设置表单在其添加的每个模型行上都会填入 `max_tokens`，并且 `coddy serve` 会在启动时于警告中说明它。文件缺失是错误，因为该标志的存在就是为了检查启动将使用的文件。形似密钥的键（`api_key`、`auth_token`、`pairing_tokens`）下的值绝不会在消息中回显。

完全无法解析的文件与值只是错误的文件处理方式不同。解析器报告它正在读取的块开始的行，而在带有注释头部的文件中，这可能是错误上方很远的空行，因此检查会重新读取文件，找到使其停止解析的行并改为报告该行。启动时打印同一行，因此 `coddy -t` 和 `coddy serve` 会将你送到同一位置。

编辑器留在文件中的内容不属于配置。Windows 上写入的文件以回车和换行结束行，并可能在第一个换行前带有字节顺序标记；两者在读取时都会被丢弃，因此标记后的 `# yaml-language-server:` 头部仍会被找到，发现仍会指出编辑器显示的行，保存会恢复文件自身的行尾。Coddy 不读取的一种形式是 UTF-16——记事本的 "Unicode"——它会按此报告，并建议使用 UTF-8，而不是作为语法错误报告。

## 试运行：探测文件指向的内容

`--dry-run` 在执行与 `--test-config` 相同的检查后，查看文件描述的世界。每个接受 `-t` 的命令也接受它：`coddy --dry-run`、`coddy cli --dry-run`、`coddy acp --dry-run`、`coddy serve --dry-run`，其中 `--config` 和 `--home` 像启动时一样选择文件。静态检查先运行，有错误的文件会停在那里——探测损坏文件命名的内容只会让第一个错误淹没在其后果中。文件干净时，配置会在无副作用的情况下加载（不写入或恢复 `config.yaml.bak`）并探测：

- **memory** - `memory.additional_prompt` 长于 `memory.additional_prompt_max_chars` 会在该键处产生警告：memory 子智能体读取被截断的文本；
- **paths** - `sessions.dir`、`logger.file`、`memory.dir` 和调度器的任务文件夹在缺失时只要可以创建就没问题（进程在启动时创建它们），而当常规文件挡路时则是错误；`prompts.dir` 必须存在，其中缺失的模板是警告；你写入的 `skills.dirs`、`subagents.dirs` 和 `hooks.files` 条目在缺失时是警告，而缺失的默认值保持安静；存在的钩子文件必须能解析；`instructions.files` 条目在其文件不存在、无法读取、是文件夹或为空时是警告，但相对路径或 `${CWD}` 条目在检查运行的工作区中缺失时会被跳过（另一个工作区可能携带它）；`swarm.tls` 必须能加载，且每个 `dial.ca_file` 必须持有证书；
- **LLM providers** - 每个提供商会请求其模型列表，这在一次请求中检验地址、代理和凭据（包括 `coddy providers login` 凭据）；指向供应商官方端点且没有任何可呈现内容的提供商会在不发送请求的情况下报告。然后每个 `models[]` 条目会对照该列表检查：服务器未命名的模型是警告，因为某些服务器提供的模型多于列表。`codex` 模型上的 `max_tokens` 无论提供商如何回答都是警告，因为没有请求会携带它；
- **MCP servers** 中的 `~/.coddy/mcp.json` - stdio 服务器的可执行文件会像生成时一样在 `PATH` 中解析，但不实际生成；远程服务器会被请求任意 HTTP 响应，并带上其头部。项目本地的 `.coddy/mcp.json` 声明不会被联系：它们位于工作区信任门之后；
- **Telegram** - 当 `gateways.telegram.enable` 为 true 时，令牌会通过 `gateways.telegram.proxy`（如设置）对照 Bot API（`getMe`）检查；报告会指出机器人；
- **remotes** - 每个 `httpserver.remotes[]` URL 都会被请求响应（宕机时是警告，因为它仅按需使用），并且控制台或 `acp` 运行的 `--remote` 目标必须接受令牌；
- **仅 `coddy serve`** - 配置和类型化标志启用的子系统会像启动时一样解析（此二进制未构建的界面是错误，而不是静默跳过），每个监听地址会绑定一次并释放，因此另一个进程占用的端口会连同设置它的行一起指出，并且 `swarm.join` 中的中继以及中继挂载的上游会通过其拨号设置到达。

单独使用时该标志很安静：它打印问题——每个 `warning` 和 `error` 都带文件中的位置和修复方法——以及末尾一行状态，因此健康的设置只回答该行，部署脚本只需读取一项：

```text
$ coddy --dry-run
dry run: 0 errors, 0 warnings, 4 ok
```

出问题时，问题先出现，状态行仍会结束报告；退出状态为 1，最后一行显示 `dry run failed`：

```text
$ coddy --dry-run
warning  skills.dirs[0]: /home/me/.coddy/skills does not exist
         at /home/me/.coddy/config.yaml:22:10
         fix: create it or remove the entry; a ${CWD} entry is resolved per session, so a folder missing here may exist in another workspace
warning  skills.dirs[1]: /opt/team-skills does not exist
         at /home/me/.coddy/config.yaml:22:34
         fix: create it or remove the entry; a ${CWD} entry is resolved per session, so a folder missing here may exist in another workspace
error    mcp.json[tickets]: command "ticket-mcp" not found in PATH
         fix: install it or write an absolute path as the command of tickets in /home/me/.coddy/mcp.json
warning  models[local/llama-4]: not in the model list of provider local (the server may still serve it)
         at /home/me/.coddy/config.yaml:11:5
         fix: check the model id; the provider lists gpt-oss-20b, qwen3.6-35b
error    providers[gpu]: cannot reach http://127.0.0.1:18732/v1: dial tcp 127.0.0.1:18732: connect: connection refused
         at /home/me/.coddy/config.yaml:6:5
         fix: check api_base and that the server is running
dry run: 2 errors, 3 warnings, 4 ok
dry run failed
```

添加 `--test-config` 可查看全貌：首先是配置检查报告（与 `-t` 打印的相同，包括 `valid`），然后是每个探测，包括通过的探测，因此报告会显示实际尝试了什么以及针对哪个地址：

```text
$ coddy --dry-run --test-config
/home/me/.coddy/config.yaml: valid
ok       sessions.dir: /home/me/.coddy/sessions will be created at first start
warning  skills.dirs[0]: /home/me/.coddy/skills does not exist
         at /home/me/.coddy/config.yaml:22:10
         fix: create it or remove the entry; a ${CWD} entry is resolved per session, so a folder missing here may exist in another workspace
warning  skills.dirs[1]: /opt/team-skills does not exist
         at /home/me/.coddy/config.yaml:22:34
         fix: create it or remove the entry; a ${CWD} entry is resolved per session, so a folder missing here may exist in another workspace
ok       mcp.json[context7]: command "npx" resolves to /usr/bin/npx
error    mcp.json[tickets]: command "ticket-mcp" not found in PATH
         fix: install it or write an absolute path as the command of tickets in /home/me/.coddy/mcp.json
ok       providers[local]: openai at http://127.0.0.1:18731/v1 lists 2 models
         at /home/me/.coddy/config.yaml:3:5
ok       models[local/qwen3.6-35b]: listed by provider local
warning  models[local/llama-4]: not in the model list of provider local (the server may still serve it)
         at /home/me/.coddy/config.yaml:11:5
         fix: check the model id; the provider lists gpt-oss-20b, qwen3.6-35b
error    providers[gpu]: cannot reach http://127.0.0.1:18732/v1: dial tcp 127.0.0.1:18732: connect: connection refused
         at /home/me/.coddy/config.yaml:6:5
         fix: check api_base and that the server is running
skipped  models[gpu/qwen3.6-35b]: provider gpu failed
dry run: 2 errors, 3 warnings, 4 ok
dry run failed
```

`ok` 和 `skipped` 行不带修复；`warning` 从不会使运行失败；`error` 会。未通过静态检查的文件总会显示，无论给出了哪些标志：在修复之前无法探测其他内容。网络探测并发运行，每个限制为十秒，因此死服务器只需等待一次，而不是每个模型一次。密钥不会回显：Telegram 令牌在任何错误文本中都会被掩码，提供商密钥绝不会打印。`CODDY_TELEGRAM_API_BASE` 将 Telegram 探测和机器人本身指向另一个 Bot API 源：自托管服务器，或离线测试环境 [tgfake](../surfaces/gateway.md#针对假-bot-api-调试)。

## 完整配置 Schema

智能体名称、标题和构建版本不能在此配置。它们在二进制中固定，并在 ACP `initialize`（`internal/acp` 和 `internal/version`）期间报告。

```yaml
# LLM backends (Go: []config.ProviderConfig, internal/config/providers.go)
# Each providers[].name must match ^[a-zA-Z][a-zA-Z0-9_-]*$ (ASCII letter first, then letters, digits, hyphen, underscore).
# api_key may be a literal, "${ENV}" expanded when the file loads, or empty to read NAME_API_KEY at LLM call time
# (NAME is the provider name in uppercase with hyphens mapped to underscores, for example rpa -> RPA_API_KEY).
# api_key_command (optional): when api_key is empty, this command is run via the detected host shell and its trimmed stdout is
# used as the key (credential helper, like git/docker helpers or AWS credential_process). It lets a provider fetch
# short-lived or login-issued keys without storing a static secret. On failure resolution falls back to NAME_API_KEY.
# Resolution order: literal api_key -> api_key_command stdout -> NAME_API_KEY env.
# A codex row reads none of the three and runs no helper: it signs in with ChatGPT (coddy providers login).
providers:
  - name: "openai"
    type: "openai"
    api_key: "${OPENAI_API_KEY}"
    # api_base: ""                    # optional override for OpenAI-compatible base URL
    # api_key_command: "my-cli print-token"  # host shell: pwsh/powershell/cmd on Windows; bash/sh elsewhere
    # proxy: none                     # route of this row: inherit (default), none, or a proxy URL
    # timeout_ms: 300000               # optional bound on each LLM request incl. streamed read (0 = no client timeout)

  - name: "anthropic"
    type: "anthropic"
    api_key: "${ANTHROPIC_API_KEY}"

  - name: "neuraldeep"
    type: "neuraldeep"
    api_key: "${NEURALDEEP_API_KEY}"

  # In the bundled web UI, select codex and use Sign In with ChatGPT. Tokens are
  # stored at $CODDY_HOME/providers/codex/codex-auth.json, not in config.yaml.
  - name: "codex"
    type: "codex"

  # `coddy providers login devin` signs in to a Devin account in the browser
  # (or reuses the Devin CLI login with --devin-cli) and adds this row and one
  # model per family. The session token lives under $CODDY_HOME/providers/devin/.
  - name: "devin"
    type: "devin"

  - name: "local"
    type: "openai"
    api_base: "http://localhost:11434/v1"
    api_key: "~"

  - name: "deepseek"
    type: "openai"
    api_base: "https://api.deepseek.com/v1"
    api_key: "${DEEPSEEK_API_KEY}"

# Logical models (Go: []config.ModelEntry, internal/config/models.go).
# Each model value is "provider_name/api_model_id". The first path segment must match providers[].name.
# The same string is the ACP model selector and agent.model default.
models:
  - model: "openai/gpt-5.6-terra"
    max_tokens: 8192
    reasoning_default: medium     # reasoning models take a level, not a temperature
    multimodal: true              # accepts images/files; UI shows file attachment button

  - model: "anthropic/claude-3-5-sonnet-20241022"
    max_tokens: 8192
    temperature: 0.2
    multimodal: true

  - model: "openai/gpt-5"
    max_tokens: 8192
    reasoning_default: medium     # level pre-selected for new chats (composer reasoning selector)
    # reasoning_levels: [low, high]  # optional override of offered levels; [] hides the selector
    # allow_reasoning_off: true       # optional: add Off only after verifying this deployment honours the disable-reasoning request

  - model: "local/qwen2.5-coder:14b"
    max_tokens: 4096
    temperature: 0.1

  - model: "deepseek/deepseek-coder-v2"
    max_tokens: 8192
    temperature: 0.1

  - model: "neuraldeep/default"
    max_tokens: 8192
    temperature: 0.2

  - model: "codex/gpt-5.6-sol"
    max_tokens: 8192

  - model: "devin/claude-sonnet-5"
    reasoning_levels: [low, medium, high, xhigh, max]  # each level is a variant of the family
    reasoning_default: medium

  - model: "local/qwen3.6-35b"
    max_tokens: 4096
    max_context_tokens: 49152     # a small window: offer only the tools a coding session calls
    toolset: coding               # a preset of toolsets below; a session can pick another with /toolset

# Named presets of the tools a session is offered (see "Toolsets" below)
toolsets:
  - name: coding
    tools: [read, write, edit, apply_patch, glob, grep, print_tree, keep_result, run_command,
            background_output, background_wait, background_stop, compact_context]
    # disallowed_tools: [ssh_run_command, http_request]  # removes names after tools; alone it keeps the rest

# ReAct loop settings (Go: config.Agent, internal/config/agent.go)
agent:
  model: "openai/gpt-5.6-terra"  # optional default LLM until the client overrides per session;
                               # unset, interactive surfaces pick a model per session, while
                               # coddy -p / coddy acp / API calls without a model report "no model configured"
  max_turns: 165               # ReAct iterations per prompt, recoveries included; 0 explicitly disables the limit
  llm_retry_max: 3             # shared per-step budget: transport retries + no-answer and output-limit recoveries
                               # (default 3; 0 disables these retries, not separately configured continuations)
  llm_retry_base_ms: 1000      # initial backoff between LLM retries; a server-provided
                               # pause (Retry-After-Ms / Retry-After headers, "Limit resets
                               # at" / "retry in Ns" body phrases) overrides the backoff,
                               # capped at 60s
  llm_min_interval_ms: 0       # min gap between consecutive LLM calls, retries included; e.g. 12000 on strict free tiers
  llm_first_token_timeout_ms: 90000  # cancel a silent streamed LLM call after this long (0 disables the guard)
  llm_stream_idle_timeout_ms: 300000 # cut a streamed answer that sends nothing for this long after its first bytes,
                                     # keeping the text already delivered (0 disables the guard; blocking models are never guarded)
  wait_for_limit_reset: false        # wait for a hit usage limit to lift and re-issue the call (off: the turn ends with the error)
  wait_for_limit_reset_max_ms: 14400000  # total wait per turn (4 h), the retry wrapper's sleeps on a limit included; under 60 s it also bounds ordinary 429 retries; 0 never waits
  loop_guard: true             # stop a response that repeats itself, and a tool called over and over with identical args
  loop_tool_repeat_limit: 2    # identical calls in successive ReAct responses before the guard steps in (0 disables)
  loop_stream_repeat_cycles: 5 # identical output cycles in one stream before it is cut (0 disables)
  loop_nudge_max: 1            # nudges before the guard stops the turn with a notice

# System prompt templates
prompts:
  # Empty dir = use embedded defaults. Otherwise a directory containing the files named below.
  #
  # Whatever you put here, Coddy prepends its identity line ("You are Coddy, ...") unless the
  # template already opens with it — gateways attribute traffic by the start of the system
  # prompt. Source: internal/prompts/identity.go, rationale: docs/contributing/react-agent.md (Agent identity).
  #
  # Go text/template data. Fields in internal/prompts/loader.go. YAML shape is config.Prompts in internal/config/prompts.go.
  #   {{.CWD}}      - session working directory
  #   {{.Tools}}    - markdown list of tool names and short descriptions for the current mode
  #   {{.Skills}}   - markdown block for active skills (omit section when empty via {{if .Skills}})
  #   {{.Rules}}    - the AGENTS.md and DESIGN.md of the agent home and of the workspace, then the always-on rules
  #   {{.Instructions}} - the files of instructions.files. A template that prints neither block still gets the
  #                   documents and those files: in {{.Instructions}} when it has it, otherwise appended after it
  #   {{.TodoList}} - current session todo checklist as markdown lines (empty until coddy todo tools update state)
  #   {{.Memory}}   - session notes. The memory subagent's report is not rendered here: it comes back as the tool result of the spawn_agent call that delegated to it
  #   {{.UTCNow}}   - date and time in UTC (RFC3339), refreshed whenever the system prompt is rendered
  #
  # Built-in templates order: Tools, Skills, Memory (session notes).
  # They deliberately render neither {{.TodoList}} nor {{.UTCNow}}: both move between the steps of a
  # turn, and the system prompt is what the provider's prompt cache keys the whole conversation on.
  # Coddy sends the clock and the checklist after the history instead, in a <turn_context> block,
  # and a rule a tool call activates rides in that call's result. Your own template may still render them, at the cost of that cache.
  # See docs/contributing/react-agent.md (The turn context block).
  dir: ""
  agent_prompt: "agent.md"     # optional; default agent.md
  plan_prompt: "plan.md"       # optional; default plan.md
  ask_prompt: "ask.md"         # optional; default ask.md

# Session bundle storage (Go: config.Sessions, internal/config/sessions.go)
sessions:
  # Empty = default $CODDY_HOME/sessions. Supports ${CODDY_HOME} and ~ in path.
  dir: ""

# Context compaction (Go: config.Compaction, internal/config/compaction.go).
# Summarizes history older than the keep-recent boundary into one transcript row;
# later LLM prompts replay only the summary plus the kept tail. Trigger manually
# with the built-in /compact command (optional -m/--model <id> and -r/--reasoning <level>
# for that one summary, then optional trailing summarizer instructions)
# or automatically at threshold_percent of the model's context window
# (models[].max_context_tokens, else the window the provider reports, else 128000).
compaction:
  enable: true             # master switch (manual command and automation)
  auto_enable: true        # false disables only automatic compaction
  threshold_percent: 80    # auto-compact trigger, 1..100, a percent of the context window
  keep_recent_turns: 2     # last N user turns stay verbatim; 0 summarizes everything
  model: ""                # models[].model for the summarizer; empty = session model
  fallback_models: []      # tried in order when the summarizer above them fails; the session
                           # model is the last resort whether or not it is listed

# Optional long-term memory subagent (Go: config.MemoryConfig, internal/config/memory.go; logic in external/memory).
# Linked with the memory build tag; enable at runtime with memory.enable. The model then delegates to the memory
# subagent on demand through spawn_agent with agent: "memory" (docs/features/memory.md).
memory:
  enable: false
  # Exact id from models[]. The memory subagent runs on it; the main assistant model is unaffected.
  # Example: "rpa/qwen3.6-35b-a3b". Empty means the session's model.
  model: ""
  dir: "" # long-term memory root; empty = $CODDY_HOME/memory. Supports ${CODDY_HOME} and ~ when set.
  recall_max_turns: 6   # the child's round cap is the larger of the two
  persist_max_turns: 12
  copilot_max_tokens: 4096
  max_search_hits: 8
  max_note_chars: 900   # longest body one saved note may have, in characters; 0 = no cap
  additional_prompt: ""          # your own instructions for the memory subagent only; the main agent never sees them
  additional_prompt_max_chars: 0 # cut additional_prompt at this many characters (a warning is logged); 0 = no cap

# Skills directories (Go: config.Skills, internal/config/skills.go)
skills:
  # Four folders are always read, lowest -> highest priority, whatever this
  # key says (a lower folder wins a skill name over the ones above it):
  #   ${HOME}/.agents/skills    - your skills shared with every agent (npx skills / npx skillsbd)
  #   .agents/skills            - the project's skills shared with every agent
  #   ${CODDY_HOME}/skills      - Coddy's own: the standard delivery, installed skills
  #   .coddy/skills             - the project's skills for Coddy
  # dirs only ADDS directories, read after the four and stronger than them.
  # ${HOME} and ~ are your home, ${CWD} and a relative path the session's workspace.
  dirs:
    - "~/my-team-skills"

# Rules (Go: config.Rules, internal/config/rules.go)
# One project folder is read under the session CWD: the first of .coddy/rules,
# the shared .agents/rules, .cursor/rules, .claude/rules and .codex/rules that
# holds a rule file, so another agent's mirror of the same rules is not loaded
# twice. A subfolder's own rule folders are read the same way when a tool reaches
# a file under them, and apply to that subfolder only. Your own ~/.coddy/rules
# joins them in every workspace. The AGENTS.md and
# DESIGN.md documents - yours in ~/.coddy, the workspace's, and the nested ones
# of the folders a tool enters - are read whatever these keys say and have no
# key here. .mdc files are read as Cursor rules, .md files as Claude Code rules.
# The rules that always apply go into {{.Rules}} in the system prompt (separate
# from skills); a rule scoped to paths arrives with the tool result or message
# that touches a matching path. See docs/features/rules.md.
rules:
  auto_discover: true
  systems: []   # optional: user, coddy, agents-dir, cursor, claude, codex ("agents" no longer affects the AGENTS.md files; alone it loads no rule folder)

# MCP servers are not declared here: they live in ~/.coddy/mcp.json (every
# session) and <workspace>/.coddy/mcp.json (that project, once approved), a
# Cursor-compatible "mcpServers" object. An old mcp_servers list is moved into
# ~/.coddy/mcp.json on the next start. See docs/features/mcp.md.

# Tool configuration (Go: config.Tools, internal/config/tools.go)
# The permission mode (ask, accept_edits, bypass) is not configured here: a new
# session starts in the mode chosen last on any surface (/permissions, the composer
# chip, the permission dialog, --permission-mode, ACP session/set_config_option
# "permission_mode"), kept in ${CODDY_HOME}/permission-mode.json; ask until one is
# chosen. See docs/operate/security.md.
# tools:
#   # TCP dial timeout for SSH connections in seconds (default: 30).
#   ssh_connect_timeout: 30

# Subagents (Go: config.Subagents, internal/config/subagents.go). Child agents the model spawns with spawn_agent
# from markdown definitions; each run is a background task with its own child session. See docs/features/subagents.md.
# subagents:
#   enable: true
#   dirs: []                      # extra folders after ~/.agents/agents, .agents/agents,
#                                 # ${CODDY_HOME}/agents and .coddy/agents, which are always read
#   project_trust: ask            # ask (approve project files once per project) | allow | deny
#   max_concurrent: 4             # subagent runs in flight across the whole process
#   max_depth: 1                  # 1 = children cannot spawn further; 0 = nobody spawns
#   default_timeout_seconds: 1800 # hard limit when the definition and the call give none
#   max_turns: 0                  # 0 follows agent.max_turns

# Hooks (Go: config.Hooks, internal/config/hooks.go). Your own commands at lifecycle points of a session,
# defined in JSON files of Claude Code's shape; project files need a one-time approval. See docs/features/hooks.md.
# hooks:
#   enable: true
#   files: ["${CODDY_HOME}/hooks.json", "${CWD}/.claude/settings.json", "${CWD}/.claude/settings.local.json", "${CWD}/.coddy/hooks.json"]
#   project_trust: ask            # ask (approve project files once per project) | allow | deny
#   default_timeout_seconds: 60   # per hook process when the definition gives no timeout
#   stop_loop_limit: 5            # Stop-hook continuations per turn
#   max_output_chars: 10000       # cap on what one hook hands to the model or the user

# HTTP OpenAI gateway (only with go build -tags=http). Embedded SPA on / needs -tags=http,ui too. See docs/reference/http-api.md
# httpserver:
#   host: "127.0.0.1"
#   port: 8080

# Cron scheduler (only with go build -tags=scheduler). UTC crontab; flat *.md jobs in ${CODDY_HOME}/scheduler
# and, once approved, in <workspace>/.coddy/scheduler.
# scheduler:
#   enable: false
#   project_trust: ask
#   max_queue: 10
#   timeout: "30m"
#   retain_sessions: 5  # max completed run session dirs kept per job_id (default 5)

# Prometheus exporter (Go: config.MetricsConfig, internal/config/metrics.go; only with go build -tags=metrics).
# GET /metrics on a port of its own, run by coddy serve. See docs/operate/monitoring.md
# metrics:
#   enable: false
#   host: "127.0.0.1"   # empty falls back to it; bind 0.0.0.0 only together with a token
#   port: 12347         # 0 falls back to it
#   token: ""           # bearer credential a scrape must present; ${ENV} is expanded

# Logging (Go: config.Logger, internal/config/logger.go)
logger:
  level: "info"           # debug | info | warn | error
  # Raise or lower one subsystem on its own. A component is the dotted name a
  # record carries in its "component" field (gateway, gateway.telegram,
  # session, agent, scheduler); a parent name covers what is nested under it
  # and the longest match wins. Omitted = every record follows level above.
  levels:
    - component: "gateway.telegram"
      level: "debug"
  # Where records go: any combination of stdout, stderr, file. Omitted or empty = stderr only.
  outputs: []
  # Path for the file sink; required when outputs includes file.
  file: ""
  # text (default) or json
  format: "text"
  rotation:
    max_size_mb: 0        # 0 = no size-based rotation
    max_files: 0          # rotated backups to keep when max_size_mb > 0
```

设置后，ACP 标志会覆盖相同的调节项：**`--log-level`**、**`--log-output`**（stdout、stderr、文件、两者）、**`--log-file`**、**`--log-format`**。空标志值保留 YAML（或内置）默认值。**`--log-level`** 还接受以逗号分隔列表形式给出的按组件规范（**`--log-level "info,gateway.telegram=debug"`**）；裸级别会保留已配置的 **`levels`** 条目不动，而命名组件的规范会替换它们。

如果较旧的双字段风格在 **`logger`** 下设置了 **`file`**，但没有 **`outputs`**，加载器会展开为 **`stderr`** 加 **`file`**，以便文件日志生效。

## SSH 远程执行

内置 `ssh_run_command` 工具允许智能体通过 SSH 在远程主机上运行命令——无需外部 `ssh` 二进制（通过 `golang.org/x/crypto/ssh` 的纯 Go 实现）。唯一可配置项是 `tools.ssh_connect_timeout`（TCP 拨号超时，默认 30 秒）。

**认证顺序：**
1. **SSH agent** —— 如果 `SSH_AUTH_SOCK` 已设置且可访问，则首先使用 agent。这涵盖 YubiKeys、1Password SSH agent、gpg-agent 和标准 `ssh-agent` 设置。
2. **密钥文件** —— Coddy 始终在当前操作系统用户的 `~/.ssh` 目录中查找。尝试的密钥名称顺序为：`id_ed25519`、`id_rsa`、`id_ecdsa`、`id_dsa`。受口令保护的密钥会被静默跳过。

两种来源同时生效——如果 agent 可用且拥有密钥，当 agent 拒绝时文件仍可作为回退。

**主机密钥验证** - 根据运行调用的会话的权限模式自动派生：
- 除 `bypass` 外的任何模式 **（默认）** —— 新主机在首次连接时自动添加到 `~/.ssh/known_hosts`（TOFU）；如果已知主机的密钥已更改，旧条目会被新条目替换。
- `bypass` —— 禁用主机密钥验证（适用于临时虚拟机或 CI 环境）。

**工具 schema** —— `ssh_run_command` 接受：
| 字段 | 类型 | 必填 | 描述 |
|---|---|---|---|
| `host` | string | 是 | `user@hostname` —— 用户必填 |
| `command` | string | 是 | 在远程主机上运行的 Shell 命令 |
| `port` | integer | 否 | SSH 端口（默认：22） |
| `timeout_seconds` | integer | 否 | 命令超时时间（秒） |
| `permission_rationale` | string | 否 | 权限对话框中显示的文本 |

该工具需要用户权限（与 `run_command` 相同），并返回合并的 stdout + stderr。

## HTTP 网关（可选构建）

**`httpserver`** 键（在 `internal/config/http.go` 中为 `config.HTTPServerConfig`）除非使用带 **`-tags http`** 构建的二进制，否则会被忽略。当 **`coddy serve`** 仍为内置标志默认值（`0.0.0.0` 和 `12345`）时，它设置默认 **`host`** 和 **`port`**。见 **`docs/reference/http-api.md`**。

### Web 界面登录（`httpserver.login`）

默认关闭。它会关闭位于网络上的服务器的浏览器界面，因此对话记录、工具输出和配置编辑器不会被发现该端口的人读取。请用命令写入账户，而不是手动写入：

```bash
coddy serve set-password --user pasha
```

```yaml
httpserver:
  host: 0.0.0.0
  auth_token: "${CODDY_HTTP_TOKEN}"   # unchanged: the credential API clients present
  login:
    enable: true                       # omit to follow the credentials; false wins over everything
    user: "pasha"
    password_hash: "$$argon2id$$v=19$$..."   # argon2id, written by the command above
    session_ttl_hours: 720                   # 0 = the browser drops the cookie on close (the server still expires its record after 30 days)
```

手动写入此文件的哈希需要将每个 `$` 加倍（`$$argon2id$$v=19$$...`），因为文件加载时 `$NAME` 会被展开为环境引用。命令会为你完成；`coddy -t` 在发现不再能解析的哈希时会指出问题。`user` 或 `password_hash` 中的 `${VAR}` 引用会像这里的其他值一样工作，并且在从设置屏幕保存后仍作为引用保留；要完全将凭据排除在文档之外，请改用 `CODDY_HTTP_USER` / `CODDY_HTTP_PASSWORD`。

账户也可以仅来自环境——`CODDY_HTTP_USER` 和 `CODDY_HTTP_PASSWORD`，见下文 `.env` 章节——这是容器或 systemd 单元的方式。表单供浏览器使用；`coddy --remote`、`coddy acp --remote`、swarm 中继和每个脚本仍然出示 bearer 令牌。完整行为：[HTTP API](../reference/http-api.md#web-ui-登录可选)、[远程模式](../operate/remote.md#登录表单)。

## 调度器（可选构建）

### MCP 项目信任

**`mcp.project_trust`** 键决定项目本地的 **`<workspace>/.coddy/mcp.json`** 是否可以启动其服务器：**`ask`**（默认）会保持它们，直到操作员批准该工作区的每个声明，**`allow`** 会自动启动它们，**`deny`** 从不加载它们。传递 **`coddy acp --mcp-project-trust <value>`** 或 **`coddy serve --mcp-project-trust <value>`** 可为单个进程覆盖它，CI 任务和容器入口点使用它而不是编辑配置文件。未知值会导致启动失败。为[问题 #80](https://github.com/coddy-project/coddy-agent/issues/80)添加；完整指南见 [docs/features/mcp.md](../features/mcp.md)。

**`scheduler`** 键（在 `internal/config/scheduler.go` 中为 `config.SchedulerConfig`）仅在使用 **`-tags scheduler`** 构建时使用。在 YAML 中设置 **`scheduler.enable: true`**，或传递 **`coddy acp -scheduler`** / **`coddy serve -scheduler`**，以在不编辑配置文件的情况下为该进程设置 **`scheduler.enable`**。

任务是两个固定文件夹中的扁平 **`*.md`** 文件：你自己的任务在 **`${CODDY_HOME}/scheduler`**，仓库携带的项目任务在 **`<workspace>/.coddy/scheduler`**，只有在 **`scheduler.project_trust`** 下受信任后才会运行（默认 `ask`；你通过 Coddy 创建的项目任务会立即获批，见[调度器](../operate/scheduler.md#项目任务与信任)）。旧的 **`scheduler.dir`** 键不再读取：下次启动会将其任务复制到 **`${CODDY_HOME}/scheduler`** 并移除它。每个文件都有 YAML frontmatter，包含 **`description`**、**`schedule`**（五个 cron 字段，**UTC**）、可选的 **`cwd`**（默认是启动 **`coddy`** 的目录，对于项目任务则是其工作区，其 cwd 必须保持在其中）、**`model`**、**`mode`**（`agent`、`plan` 或 `ask`）、可选的 **`agent`**（运行所依据的子智能体定义）、可选的 **`permission_mode`**（`ask`、`accept_edits` 或 `bypass`；空为 `bypass`，即无人值守默认值）、可选的 **`paused`**（为 true 时，cron 和手动运行会被跳过，直到恢复）。Markdown 正文是运行的一次性指令，该运行是任务自身会话下的后台智能体任务（[调度器](../operate/scheduler.md)）。一个 sidecar **`basename.state`**（最后触发的槽位和任务会话 id）对于用户任务位于 **`basename.md`** 旁边，对于项目任务位于 **`${CODDY_HOME}/scheduler/.projects/`** 下。

**`retain_sessions`**（默认 **5**）限制每个 **`job_id`** 保留多少个**已完成**运行（其任务记录和对话记录，位于任务会话下）；较旧的运行会在运行完成时移除。**`max_queue`** 限制所有任务中正在进行的运行数，**`timeout`** 是一次运行的硬限制（后台任务池仍将其限制在 **`tools.background.max_timeout_seconds`**）。

当调度器实际启用时，**`coddy_scheduler_*`** 工具涵盖列出或获取、创建或替换或修补、删除、暂停或恢复、手动运行、取消，以及列出运行元数据（**`coddy_scheduler_jobs_list`**、**`coddy_scheduler_job_get`**、**`coddy_scheduler_job_create`**、**`coddy_scheduler_job_replace`**、**`coddy_scheduler_job_patch`**、**`coddy_scheduler_job_delete`**、**`coddy_scheduler_job_pause`**、**`coddy_scheduler_job_resume`**、**`coddy_scheduler_job_run`**、**`coddy_scheduler_job_cancel`**、**`coddy_scheduler_job_runs`**）。使用 **`-tags=http,scheduler`** 时，相同操作以 REST 形式存在于 **`/coddy/scheduler`** 下（见 **`docs/reference/http-api.md`**）。

## 指标导出器（可选构建）

**`metrics`** 键（在 `internal/config/metrics.go` 中为 `config.MetricsConfig`）仅在带 **`-tags metrics`** 构建的二进制中生效，发布二进制包含它。任何其他二进制会在启动时拒绝 **`metrics.enable: true`** 并指出该标签。带有该标签时，**`coddy serve`** 在 **`metrics.enable`** 为 true 时与其他子系统一起运行导出器，并在 **`metrics.host`** 和 **`metrics.port`** 上提供 **`GET /metrics`**（默认 `127.0.0.1:12347`）。**`metrics.token`** 是抓取必须出示的 bearer 凭据：在绑定到非回环地址前设置一个，`coddy serve` 会在启动时对此警告，`--dry-run` 会报告它。设置屏幕在带该标签的构建中显示 **Metrics** 标签页。抓取配置、指标列表和警报规则见[监控](../operate/monitoring.md)。

## 消息网关（`gateways`）

需要带 **`-tags gateway.telegram`**（仅 Telegram）或 **`-tags gateway`**（所有适配器）构建的二进制。`coddy serve` 子命令读取此块。

```yaml
# Messenger gateways (external/gateway/; build with -tags gateway.telegram or -tags gateway).
# Full guide: docs/surfaces/gateway.md
gateways:
  telegram:
    # Set to true to activate the Telegram adapter when coddy serve starts.
    enable: false

    # Bot token from @BotFather. Never hard-code; always use an env reference.
    token: "${TELEGRAM_BOT_TOKEN}"

    # How the bot reaches the Bot API: inherit (the default) follows HTTPS_PROXY,
    # none connects directly, or a proxy URL (http, https, socks5, socks5h).
    # proxy: "socks5h://127.0.0.1:1080"

    # Telegram user IDs with admin privileges.
    # Admins bypass every access check and can always interact with the bot.
    admins: []
    # Example:
    # admins: [98874093]

    # Default access level for chats without a per-chat override.
    #   "all"          - anyone who can write to the chat
    #   "admins"       - only user IDs listed in admins
    #   "group:<name>" - only users in the named user_groups entry (admins always pass)
    default_access: "all"

    # Default session isolation mode for group chats without a per-chat override.
    #   "individual"   - each group member gets their own session
    #   "shared"       - all members share one session
    #   "admin"        - only admins can interact; all admins share one session
    default_isolation: "individual"

    # Named sets of user IDs for group-based access control.
    user_groups: []
    # Example:
    # user_groups:
    #   - name: "devs"
    #     user_ids: [111222333, 444555666]

    # Per-chat overrides. chat_id is negative for groups and supergroups.
    chats: []
    # Example:
    # chats:
    #   - chat_id: -1001234567890
    #     isolation: "individual"
    #     access: "all"
    #   - chat_id: -1009876543210
    #     isolation: "admin"
    #     access: "admins"
```

`token` 在 `enable: true` 时于启动时校验。`proxy` 是可选的，读取方式类似提供商的代理：空或 `inherit` 遵循环境的代理（`HTTPS_PROXY`、`HTTP_PROXY`、`NO_PROXY`），`none` 直接连接，URL 则通过该代理连接（[提供商代理](#提供商代理)）。其他字段如果省略则应用默认值：`default_access: "all"`、`default_isolation: "individual"`。

完整配置指南、运行说明以及如何为其他消息应用添加适配器，见 **[docs/surfaces/gateway.md](../surfaces/gateway.md)**。

## `.env` 文件

如果 **`$CODDY_HOME/.env`** 存在，它会在启动时、`config.yaml` 解析**之前**被读取。这样你可以将所有密钥保存在一处，而无需触碰 shell 配置文件或 Docker compose 环境块。

```sh
# ~/.coddy/.env
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
TELEGRAM_BOT_TOKEN=8992982910:AAF...
CODDY_HTTP_TOKEN=a-long-random-token       # bearer credential for API clients
CODDY_HTTP_USER=pasha                      # web UI sign-in account...
CODDY_HTTP_PASSWORD=correct-horse-battery-staple   # ...enables the form on its own
```

`CODDY_HTTP_USER` 和 `CODDY_HTTP_PASSWORD` 是唯一一对完全不被 `config.yaml` 引用的值：密码在服务器启动时进行哈希，文件永远不会看到任一值，从设置屏幕保存也无法将它们写入其中。它们优先于文件中的账户，`httpserver.login.enable: false` 会在它们仍设置时关闭表单。

然后在 `config.yaml` 中照常引用它们：

```yaml
providers:
  - name: openai
    type: openai
    api_key: "${OPENAI_API_KEY}"
```

**规则：**

- 进程环境中**已设置**的变量**绝不会被覆盖**——进程环境始终优先。`.env` 仅是回退。
- 缺失的 `.env` 会被静默忽略（不是错误）。
- 文件相对于有效的 `CODDY_HOME` 解析（默认 `~/.coddy`，或来自 `--home` / `CODDY_HOME` 环境变量的路径）。

**支持的语法：**

| 行形式 | 示例 |
|-----------|---------|
| `KEY=value` | `OPENAI_API_KEY=sk-abc` |
| `export KEY=value` | `export DEBUG=true` |
| 双引号值 | `MSG="hello world"` |
| 单引号值 | `PATH='no escape \n here'` |
| `"…"` 中的转义序列 | `NOTE="line1\nline2"` → 真实换行 |
| 行内注释（未加引号） | `KEY=val # this is ignored` |
| 注释行 | `# full-line comment` |

进程环境中已有的值（例如由 shell、Docker、systemd 设置）优先，绝不会被 `.env` 更改。

## 环境变量引用

任何配置值都可以使用 `${VAR_NAME}` 语法引用环境变量。
智能体在启动时解析这些引用。

**值中的字面 `$`。** 在手写的 `config.yaml` 中，每个字面美元符号都写成 `$$`。
例如，`$2y$10$…` 写成 `$$2y$$10$$…`。否则加载器会将 `$2y` 和 `$10` 等片段视为环境引用并更改值。设置会将 API 密钥、令牌、代理 URL 和默认 HTTP 头部值视为字面文本，并在保存时转义其美元符号。请原样输入这些值，不要将 `$` 加倍。要在凭据字段中使用环境引用，请将其放入 YAML，而不是设置中。

**保存会保留引用。** 加载后的配置持有引用解析后的内容，因此设置界面使用密钥本身和绝对路径。保存时，文件中写成 `${VAR}`、`${CODDY_HOME}/...` 或 `~/...` 的值会按该方式写回，只要它仍加载为正在保存的值——无论是在单个值（`memory.dir`）还是列表条目（`skills.dirs`、`subagents.dirs`、`hooks.files`、`instructions.files`）中；只有你在屏幕上更改的值才会替换引用。因此，保存在环境或 `~/.coddy/.env` 中的键绝不会因为无关的保存而落入 `config.yaml`，而未触碰表单的保存会按文件拼写的方式写回每个值。进程在读取文件后更改的内容也是如此：命令行标志、`coddy serve` 填入的中继监听地址或来自环境的配对令牌，除非你在屏幕上更改该值，否则不会写入 `config.yaml`；你打开表单后另一次保存更改的值也不会被你的保存放回。当保存时磁盘上的文件无法加载（手动编辑损坏、文件被删除），保存会写入服务器运行的配置，一如既往。缩进和空行不会保留：保存写入的文件缩进两个空格，节之间没有空行。

两个占位符不是环境变量：

- **`${CODDY_HOME}`** - 解析后的 `CODDY_HOME` 目录，在读取文件时替换。
- **`${CWD}`** - **会话**工作目录。它在读取文件时**不会**被替换：它保留在加载后的值中，任何使用该路径的内容都会针对发起请求的会话展开它——技能加载、子智能体和钩子发现、提示词模板（`prompts.dir`）、MCP 服务器命令、参数、URL、环境和头部。因此一个 **`coddy serve`** 进程可以服务多个工作区，而根植于项目的会话会看到该项目的 **`${CWD}/.coddy/skills`**（或你写入的任何条目，例如 **`${CWD}/.agents/skills`**），无论服务器从哪个目录启动。只有进程作用域的位置（**`sessions.dir`**、**`scheduler.dir`**、**`memory.dir`**、**`logger.file`**）在加载时针对默认工作目录（**`CODDY_CWD`**）展开 **`${CWD}`**，因为没有会话拥有它们。

名为 **`CWD`** 的环境变量不会替换该占位符（不带花括号的裸 `$CWD` 仍是普通环境引用，和以前一样），并且 **`GET /coddy/config`**、设置界面和 **`config_get`** 会按书写原样报告该条目。该占位符仅在上面列出的字段中生效；在任何其他字符串值中它会保持原样（提示词模板改用 `{{.CWD}}`）。1.0.5 及更早版本在读取文件时将 **`${CWD}`** 替换为进程目录，因此在该版本中进行的设置保存可能在你写入 **`${CWD}/.agents/skills`** 的位置存储了绝对路径，例如 **`/home/you/.agents/skills`**；请手动放回占位符以获得按会话解析。

## 模型提供商参考

提供商 **`type`** 值与 **`internal/llm.NewProvider`** 匹配：**`openai`**、**`anthropic`**、**`neuraldeep`**、**`opencode`**、**`codex`**、**`devin`**、**`acp`**。**`type: acp`** 行运行一个使用 Agent Client Protocol 的本地智能体（Claude Code、Codex、OpenCode、ZCode、另一个 Coddy），并将其作为模型提供；它接受 **`acp`** 块（**`command`**、**`args`**、**`env`**、**`modes`**、...）而不是 **`api_base`** 和 **`api_key`**——见[作为模型的 ACP 智能体](../features/acp-provider.md)。

**`type: opencode`** 默认使用 Go 订阅网关；**`api_base: https://opencode.ai/inference/v1`** 选择按量付费 Console。使用 **`coddy providers login opencode`** 登录账户（设备码和在浏览器中选择组织），或使用 **`--api-key -`** 存储服务账户密钥。存储的账户令牌会通过提供商代理自动刷新。显式的 **`api_key`** 仍优先于登录。Coddy 按模型和网关选择推理 API；见 [OpenCode](../features/opencode.md)。

YAML 拆分：

- **`providers`**：**`name`**（唯一）、**`type`**、**`api_key`**、可选的 **`api_base`**（提供商 SDK 的基础 URL 覆盖：对于 **`type: openai`**，是 OpenAI 兼容端点或没有 **`/v1`** 的 Ollama 主机；对于 **`type: anthropic`**，是 Anthropic 兼容网关/中继；对于 **`type: neuraldeep`**，它选择部署，**`https://api.neuraldeep.ru/v1`** 或 **`https://api.neuraldeep.tech/v1`**，任何其他值都会回退到第一个）、可选的 **`proxy`**（该行每个请求的路由：默认 **`inherit`**，直接连接为 **`none`**，或 **`http://`**、**`https://`**、**`socks5://`** 或 **`socks5h://`** 代理 URL；见[提供商代理](#提供商代理)）、可选的 **`usage_limits_panel`**（布尔值，默认 **`true`**；**`false`** 会在所有界面上隐藏该行的账户用量面板，并停止其背后的用量读取，目前对 **`type: neuraldeep`**、**`type: codex`** 和 **`type: devin`** 有意义）。
- **`models`**：**`model`**（字符串 **`provider_name/api_model_id`**，会话选择器和 **`agent.model`** 值；第一段命名 **`providers[].name`**，其余部分是 API 模型 id）、**`max_tokens`**、**`temperature`**、可选的 **`max_context_tokens`**（模型的上下文窗口：Web 界面上下文环、控制台上下文百分比和自动压缩所衡量的对象；为 0 时，如果提供商报告，则从其模型列表读取，否则为 128000——见[上下文压缩](../features/compaction.md#上下文窗口)）、可选的 **`multimodal`**（布尔值，默认 **`false`**；当 **`true`** 表示模型接受图像/文件输入时——UI 仅在输入框中为该模型显示文件附件按钮，并且 [`read`](../reference/tools.md#文件) 会向此类模型展示图像文件中的图片，而不是拒绝它，见[图像](../features/images.md)）、可选的 **`reasoning_levels`**（字符串列表；覆盖为此模型提供的推理级别——省略时会从 API 模型 id 自动检测：**`gpt-5*`** 和 **`gpt-6*`** → **`minimal,low,medium,high`**，OpenAI **`o`** 系列、**`gpt-oss*`**、**`qwen3*`**（qwen3、qwen3.5、qwen3.6、qwen3.8、...）和 Claude 扩展思考模型 → **`low,medium,high`**；显式空列表会隐藏输入框推理选择器）、可选的 **`reasoning_default`**（新聊天预选的级别；必须是已解析级别之一）、可选的 **`allow_reasoning_off`**（布尔值，默认 **`false`**；仅当操作员已验证其提供商/模型部署遵守 Coddy 的提供商特定禁用推理请求时，才会为该模型的选项添加 **`off`**），以及可选的 **`toolset`**（**`toolsets[]`** 预设的名称，用于缩小此模型上会话可用的工具，适用于小窗口本地模型——见[工具集](#工具集)）。推理级别映射到 OpenAI **`reasoning_effort`** 和 Anthropic 扩展思考 **`budget_tokens`**。拒绝思考预算的 Claude 模型——Opus 4.7、4.8、5 和 5.5，Sonnet 5 和 5.5，Haiku 5.5 以及 Fable 模型——改用自适应思考：级别作为 **`thinking.display: "summarized"` 旁的 **`output_config.effort` 发送，因此聊天会显示其推理，而没有级别时模型保持自己的默认值。它们不接受固定思考预算，因此较小的 **`max_tokens`** 上限会按给定值保留，已配置的 **`temperature`** 不会出现在其请求中。Opus 5.5、Sonnet 5.5 和 Fable 模型始终思考：在 **`type: anthropic`** 提供商上，即使有 **`allow_reasoning_off`**，也绝不会为它们提供 **`off`**。对于 OpenAI 兼容提供商上的 **`qwen3*`** 模型，请求还会携带 **`chat_template_kwargs`** **`{"enable_thinking": true}`**，以便聊天模板思考开关保持开启。Codex 后端拒绝 **`max_output_tokens`**，因此不会为 **`codex`** 提供商发送 **`max_tokens`**；它也拒绝其 **`gpt-5*`** 和 **`gpt-6*`** id 通常暗示的 **`minimal`** 层级，因此 codex 支持的模型改为提供 **`none`**（在输入框选择器和 **`GET /v1/models`** 中）。推理会请求摘要（**`summary: auto`**）以便思考流式传输，并请求加密推理（**`include: reasoning.encrypted_content`**），以便思维链在工具调用之间像 Codex CLI 那样重放。令牌生命周期和启动凭据报告见 [config-reference.md](../reference/config.md)。

### 工具集

每个请求都携带会话提供的每个工具的 schema。智能体模式下的会话提供约 44 个工具：它们的 `tools` 数组约 11,700 个令牌，系统提示词打印的工具列表另有 4,600 个（上下文计量器的估计，由 `make test-perf` 测量）。大窗口模型从不会注意到。具有 49k 窗口的本地模型会在第一条消息之前将三分之一花在编码会话从不调用的工具上。

工具集是 `toolsets` 下工具列表的命名预设。`models[]` 行命名其会话运行所使用的工具集，会话也可以为自己选择另一个：

```yaml
toolsets:
  - name: coding
    tools: [read, write, edit, apply_patch, glob, grep, print_tree, keep_result, run_command,
            background_output, background_wait, background_stop, compact_context]
  - name: no-remote
    disallowed_tools: [ssh_run_command, http_request]   # every tool except these two

models:
  - model: "local/qwen3.6-35b"
    max_context_tokens: 49152
    toolset: coding
  - model: "local/qwen3.6-27b"
    toolset: coding                     # the same preset, written once
  - model: "codex/gpt-5.6-sol"          # no toolset: every tool, as before
```

`coding` 的十三个工具将请求的 `tools` 数组从 44 个工具和约 11,700 个令牌减少到 13 个工具和约 3,800 个，并将系统提示词的工具列表从约 4,600 个令牌减少到约 1,400 个：每个请求大约少 11,000 个令牌。第三个模型上的会话发送它一直发送的内容。

- **语法。** `tools` 是允许列表，`disallowed_tools` 是在其后应用的工具拒绝列表；同时出现在两者中的名称会被移除。条目是[工具参考](../reference/tools.md)中的精确工具名、裸 `*` 或 `prefix*` 模式。MCP 工具命名为 `server__tool`，因此 `context7__*` 允许一个服务器的所有工具，而未命名任何 MCP 模式的允许列表不提供任何 MCP 工具。匹配方式与[子智能体定义](../features/subagents.md#定义文件)用于其自身 `tools` 和 `disallowed_tools` 的方式相同。名称由字母、数字、`.`、`_` 和 `-` 组成，`default` 是保留的。
- **空表示全部。** 列表缺失、为空或空白的预设会提供该模式会提供的每个工具，未命名任何工具集的模型也是如此。`disallowed_tools: ["*"]` 是不提供任何工具的方式：此时请求完全不携带工具，该轮次是普通聊天。
- **适用哪个工具集。** 会话自己的选择优先：`/toolset <name>`、Web 输入框的工具集芯片、控制台的 `/toolset` 选择器、编辑器的 **Toolset** 选项，或带 `selectedToolset` 的 `PATCH /coddy/sessions/{id}`。`default` 不选择任何预设，因此无论其模型命名什么，会话都会获得其模式提供的每个工具。没有选择时，会话使用其模型命名的工具集运行。选择属于会话：它保存在其 `session.json` 中，在重启和切换模型后仍保留，而新会话启动时不带选择。`/toolset coding --once`（或 `--count=N`）会将其保持那么多轮次，`/model local/qwen3.6-35b /toolset coding` 会在一行中切换两者。模型命名了配置中不存在的工具集是配置错误；选择了已移除的预设会回退到模型的工具集。
- **它只会缩小范围。** 模式（Plan 和 Ask 提供固定集合）、配置（`tools.background.enable: false`、`subagents.enable: false`）或子智能体定义已经隐藏的工具，无论预设命名什么都会保持隐藏。也不会强制加入任何内容：循环不需要工具，压缩、`/compact` 和目标检查不是工具调用。
- **它遵循设置。** 请求会获得会话设置为其所构建模型解析出的工具集的工具。切换模型（`/model`、模型选择器、`switch_model`）或切换工具集会从下一个请求开始改变集合，在一轮中间也是如此；当集合移动时，系统提示词会随之重新渲染；在两个没有工具集的模型之间切换不会改变请求中的任何内容。更改预设或模型工具集的 `config_commit` 或 `config_rollback` 会从下一个请求开始做同样的事。如果新模型的提供商无法构建，该轮次会继续使用前一个模型及其集合。对于给定工具集，集合和提示词在每个请求上都相同，因此提供商的提示词缓存会继续工作。
- **调用会再次检查。** 调用会针对产生它的请求的集合进行检查。对该请求未提供的工具调用——从另一个工具集下写入的历史重放，或凭空编造——会以 `error: tool "write" is not offered: this session runs with the toolset "coding" ...` 应答，显示为已取消，并且绝不会到达钩子或权限提示。批次中间切换，或改变该轮次工具集的 `config_commit` 或 `config_rollback`，会从下一个请求开始改变集合：同一批次中其后的调用仍会在它们被提供时使用的集合下运行。在其轮次结束后恢复调用的权限应答（稍后在 Web 界面中给出的批准）会针对会话当时运行所使用的工具集进行检查，如果该工具集不提供该工具则以相同方式拒绝，并且不存储 "allow always" 授权。
- **提示词随之变化。** 系统提示词中的工具列表持有与请求相同的工具。当不提供 `spawn_agent` 时会省略子智能体名册，当不提供该工具时会省略调用 `load_skill` 的提示。内置提示词的说明仍会按名称提及某些工具；未被提供某个工具的模型会忽略这些行。
- **子智能体。** 子智能体在一个模型上运行——父级的、定义的 `model`，或调用命名的模型——并获得其有效集合（[交集](../features/subagents.md#能力如何收窄)），再被该模型命名的工具集缩小。父级自己的工具集限制父级看到的内容，而不是子智能体获得的内容，因此小型本地编排器可以将工作交给没有工具集的模型上的子智能体。如果生成时模型的工具集未给子智能体留下任何工具，则会拒绝并给出命名该模型和工具集的消息。工具集不会决定为子智能体或其他任何地方启动哪些 MCP 服务器：模型不需要的服务器会在其声明中关闭（[MCP 开关](../features/mcp.md#启用禁用开关)）。调度器任务遵循相同规则，而工具由运行时固定的 memory 子智能体不受影响。
- **拒绝工具的提供商。** 另一侧的服务器对单个请求可携带多少工具有自己的上限：OpenAI 的是 128，Coddy 服务器的是 `httpserver.max_client_tools`，其 `GET /v1/models` 在 `limits` 下公布（[HTTP API](../reference/http-api.md#端点摘要)）。其 MCP 服务器将其推过该上限的会话过去会因提供商的裸 `400` 而失去该轮次。现在它会以双方各自计数的结果以及发送更少工具的方法结束：使用 `/toolset` 选择本节中的预设、[关闭 MCP 服务器](../features/mcp.md#启用禁用开关)，或在远端是你拥有的 Coddy 服务器时提高上限。对某个工具 `parameters` schema 大于远端可接受范围的拒绝会改为指出该工具。不会在丢弃工具的情况下重新发送任何内容：模型获得哪些工具由你决定。
- **不是安全边界。** 工具集决定模型看到什么，以节省其上下文。会话可以做什么由模式、`tools.permission_mode`、[钩子](../features/hooks.md)和信任门决定。
- **拼写错误。** `coddy -t` 和 `coddy --dry-run` 会警告未匹配任何工具的预设条目，并给出行号，服务器在启动时记录相同内容。它仍是警告：构建会省略其构建时未包含标签的工具（调度器的），而 MCP 工具仅在其服务器连接时存在，因此带 `__` 的条目绝不会被报告。
- **在哪里显示。** 只有当配置中有预设时，界面才会提供工具集：Web 输入框中 **Permissions** 之后的 **Toolset** 芯片、控制台的 `/toolset` 选择器和页脚的 `tools <name>`、编辑器的 **Toolset** 选项。添加到配置中的预设会使它们无需重启即可出现。**Settings** 有一个用于预设的 **Toolsets** 标签页，逻辑模型行的 **Tools** 块会命名该模型的工具集（[设置：提供商模型](../surfaces/web-ui.md#设置提供商模型)）。

### 提供商代理

**`providers[].proxy`** 选择提供商行发出的每个请求的路由：其补全、模型列表、账户用量、OAuth/设备登录和令牌刷新，以及登出撤销。

- 无值，或 **`inherit`**（默认）：Coddy 进程环境命名的代理。**`HTTPS_PROXY`** 涵盖 **`https://`** 地址，**`HTTP_PROXY`** 涵盖 **`http://`** 地址，**`NO_PROXY`** 列出直连的主机，回环地址绝不会走代理。**`ALL_PROXY`** 不会被读取。Coddy 一直如此行为：空值从不意味着直接连接，无论该字段的旧描述如何。
- **`none`**：直接连接。该行会忽略这些变量，因此可直接访问的提供商在机器代理损坏、过期或缓慢时仍能工作——例如检查 TLS 的公司代理、本地转发器、另一套设置遗留的变量。
- 代理 URL（**`http://`**、**`https://`**、**`socks5://`**、**`socks5h://`**）：该行的每个请求都会通过该代理，包括 **`NO_PROXY`** 主机和回环地址。使用任一 SOCKS 方案时，代理会解析主机名。凭据放在 URL 中（**`http://user:pass@host:3128`**），其中的 **`$`** 会在设置保存后保留（见[环境变量引用](#环境变量引用)）。

```yaml
providers:
  - name: corp          # needs the machine's proxy: nothing to set
    type: openai
    api_key: "${CORP_API_KEY}"
  - name: local         # reachable directly, HTTPS_PROXY or not
    type: openai
    api_base: http://192.168.1.20:8000/v1
    proxy: none
  - name: remote        # a proxy of its own, whatever the environment says
    type: anthropic
    api_key: "${ANTHROPIC_API_KEY}"
    proxy: socks5h://127.0.0.1:1080
```

该设置仅属于其所在行，因此具有不同路由的行可以并存在一个文件中。Telegram 网关的 **`gateways.telegram.proxy`** 对机器人自身的请求接受相同的值（[Telegram 网关](../surfaces/gateway.md#代理)）。它涵盖 Coddy 进程发送的请求；浏览器中打开的登录页面以及 **`api_key_command`** 所做的任何事情都不在其范围内。更改后的值会应用于该行在其之后启动的请求；已经等待浏览器的登录会按其开始时的路由完成。环境变量每个进程只读取一次，因此更改它们需要重启。关键字接受任意大小写，并以小写写回；其他任何不是代理 URL 的内容都是配置错误，并会指出接受的值（`http_request` 调用的 `direct` 不在其中）。在 Web 界面中，该行会在 **Proxy URL** 字段上方显示 **Ignore system proxy** 开关，它会写入 **`none`**（[Web 界面](../surfaces/web-ui.md#设置提供商代理)）。当提供商无法访问时，`coddy --dry-run` 会在其提示中说明请求所走的路由——该行的代理、直接连接，或环境命名的代理——并省略凭据。

### `openai`
标准 OpenAI API。支持当前推理系列（`gpt-6-astra`、`gpt-5.6-sol`、`gpt-5.6-terra`、`gpt-5.6-luna`、`gpt-5.5`）以及较旧的 `o` 系列和 `gpt-4` id。

提供商需要 **`api_key`**。可选的 **`proxy`** 路由此行的请求（[提供商代理](#提供商代理)）。**`models[].model`** 字符串必须以该提供商 **`name`** 和斜杠开头，然后是 OpenAI API 模型 id，例如 **`openai/gpt-5.6-terra`**。还要为推理 id 设置 **`max_tokens`**，为非推理 id 设置 **`temperature`**。

### `anthropic`
Anthropic API。支持：`claude-3-5-sonnet-*`、`claude-3-5-haiku-*`、`claude-3-opus-*`

提供商需要 **`api_key`**。可选的 **`api_base`** 覆盖 Anthropic API 基础 URL（默认 **`https://api.anthropic.com`**），例如 Anthropic 兼容网关或中继。可选的 **`proxy`** 路由此行的请求（[提供商代理](#提供商代理)）。像 **`anthropic/claude-3-5-sonnet-20241022`** 一样使用 **`models[].model`**，再加上 **`max_tokens`**、**`temperature`**。

### `neuraldeep`
通过其 OpenAI 兼容端点访问 NeuralDeep API。

凭据来自 hub 登录或普通密钥。**`coddy providers login neuraldeep`** 通过设备流程登录：它会打印一个短代码和确认该代码的 hub 页面，因此浏览器可以在任何机器上——笔记本电脑、手机——而 Coddy 运行在通过 SSH 访问的服务器上。当此机器有自己的浏览器时，它会为你打开该页面。**`--browser`** 要求较旧的回环回调；该方式只会在本机运行的浏览器中完成。登录会等待确认，直到 hub 的十五分钟截止时间，Ctrl-C 会提前结束。无论哪种方式，它都会将 hub 颁发的密钥存储在 **`$CODDY_HOME/providers/<name>/neuraldeep-auth.json`** 下，并将该层级的模型追加到配置；控制台中的 **`/login`** 运行相同的设备流程，捆绑的 Web 界面在提供商行上提供 **Sign In with NeuralDeep**。显式的 **`api_key`**（或 **`api_key_command`** / **`NEURALDEEP_API_KEY`**）始终优先于存储的登录。

当具有用量来源的提供商的模型处于活动状态时，控制台页脚、远程控制台和 HTTP API 会显示账户用量，在会话开始时和每轮之后刷新；见 **`docs/surfaces/console.md`**（页脚、`/usage`）和 **`docs/reference/http-api.md`**（**`GET /coddy/providers/{name}/usage`**）。目前的来源：**`type: neuraldeep`** 读取 hub 的只读 **`GET /v1/limits`**（会话和周窗口的已用百分比及重置时间、以卢布计的钱包、带重置时间的命中限制）；**`type: codex`** 使用保存的 ChatGPT OAuth 令牌读取 Codex 后端的用量端点（按上游持续时间显示计划和百分比窗口，以及功能范围的条目）；**`type: devin`** 读取 Devin API 服务器的席位管理状态 RPC（计划，以及配额计划的每日/每周配额窗口已用百分比或 ACU 计划的 ACU 计量）。使用该行自己的凭据，绝不显示美元数字，也不编造请求计数。面板默认开启；该行上的 **`usage_limits_panel: false`**（设置 → LLM 提供商中的 **Usage limits panel** 开关）会在所有界面上隐藏它，并停止该行的用量读取，适用于共享屏幕或不属于你监视的账户。

同一 API 由两个部署提供：**`https://api.neuraldeep.ru/v1`** 用于俄罗斯，**`https://api.neuraldeep.tech/v1`** 用于其他地方。**`api_base`** 选择一个——留空则为第一个，任何不是两者之一的值都会回退到它（启动警告会说明）。选择随凭据一起传递：登录会转到 **`hub.neuraldeep.ru`** 或 **`hub.neuraldeep.tech`** 以匹配，因此在登录前选择端点（**`coddy providers login neuraldeep --api-base https://api.neuraldeep.tech/v1`**，或设置中的端点下拉菜单）。使用 **`--api-base`** 登录还会将现有提供商行移动到该端点（除非 **`--no-config`**），因此该行和密钥一致；在设置中，登录会遵循表单中选定的下拉菜单，在保存之前。一个 hub 铸造的密钥不会被另一个认可；当存储的登录和所选端点不一致时，Coddy 会在启动时警告，设置行会实时显示相同警告。**`CODDY_NEURALDEEP_BASE_URL`** 和 **`CODDY_NEURALDEEP_HUB_URL`** 仍会为测试环境和测试重定向整个进程，并且它们优先于配置。可选的 **`proxy`** 路由此行的请求，包括 hub 登录和用量读取（[提供商代理](#提供商代理)）。像 **`neuraldeep/qwen3.6-35b-a3b`** 一样使用 **`models[].model`**，再加上 **`max_tokens`**、**`temperature`**。

### `devin`
Devin（Cognition）账户的模型，按照 Devin CLI 到达它们的方式访问。

**`coddy providers login devin`** 通过浏览器登录（PKCE，类似 **`devin auth login`**）：Devin 页面会将浏览器送回本机的回环端口，通过 SSH 时你改为将其最终地址粘贴到终端。**`--devin-cli`** 重用 Devin CLI 已持有的登录，不打开浏览器。会话令牌存储在 **`$CODDY_HOME/providers/<name>/devin-auth.json`** 下；没有它时，提供商会回退到 Devin CLI 的 **`credentials.toml`**，显式的 **`api_key`**（或 **`api_key_command`** / **`DEVIN_API_KEY`**）优先于两者。登录会为每个系列添加一个模型，例如 **`devin/claude-opus-5`**，系列变体作为其 **`reasoning_levels`**：级别 **`high`** 作为 **`claude-opus-5-high`** 发送。**`api_base`** 会被忽略；可选的 **`proxy`** 路由登录、目录和聊天（[提供商代理](#提供商代理)）。完整说明，包括级别如何映射到变体以及如何选择输出上限，见 [Devin](../features/devin.md)。

### 同一提供商类型的多个配置文件

行就是配置文件：多行可以共享同一类型，每行都有自己的名称，并各自保留自己的登录、模型和用量。三个 ChatGPT 账户是三个 **`codex`** 行，两个 NeuralDeep 账户是两个 **`neuraldeep`** 行：

```yaml
providers:
  - name: "codex"
    type: "codex"
  - name: "codex-work"
    type: "codex"
  - name: "neuraldeep"
    type: "neuraldeep"
  - name: "nd-tech"
    type: "neuraldeep"
    api_base: "https://api.neuraldeep.tech/v1"
models:
  - model: "codex-work/gpt-5.5"
  - model: "nd-tech/qwen3.6-35b-a3b"
```

每行分别登录：设置中其行上的 Sign In 按钮，或终端中的 **`coddy providers login <name>`**。config.yaml 尚未列出的行会在 **`--type`** 命名该类型时由其登录创建（**`coddy providers login codex-work --type codex`**）。登录落在 **`$CODDY_HOME/providers/<name>/`** 下，该行的模型是 **`<name>/<model id>`**，该行的 **`<NAME>_API_KEY`** 变量（**`ND_TECH_API_KEY`**）仅属于该行；codex 行完全不读取密钥变量。

Codex CLI 登录（**`~/.codex/auth.json`**、`CODEX_HOME`）和 Devin CLI 登录各自是一个账户，因此每个代表一行而无需自己登录：其类型的唯一行，或当有多行时，名为 **`codex`** 的行（**`devin`**）。其他每一行都自行登录，而不是悄悄使用该账户——向只有 **`chatgpt`** 一行且该行使用 Codex CLI 登录的设置中添加第二个 codex 行，会使 **`chatgpt`** 也未登录，直到它登录或被重命名为 **`codex`**。启动日志、**`coddy --dry-run`**、**`coddy providers list`** 和设置行会指出这样的行以及 CLI 登录所服务的行，`--devin-cli` 会拒绝 Devin CLI 登录不服务的行。

显示时这些行也保持分开：用量面板标题和控制台的 **`/usage`** 会在品牌旁命名该行（**`Codex · codex-work`**），除非该行以其类型命名；NeuralDeep 登录会在 hub 上以其行标记密钥（**`coddy @ host (nd-tech)`**）。

### 本地 OpenAI 兼容服务器（Ollama、llama.cpp、LM Studio）
使用 **`type: openai`**，并将 **`api_base`** 设置为已包含 **`/v1`** 的 OpenAI 兼容基础 URL，例如 Ollama 的 **`http://localhost:11434/v1`**。

### OpenRouter 风格聚合器（OpenRouter、RouterAI）
复制 OpenRouter API 的聚合器是一个 **`type: openai`** 行，其 **`api_base`** 为（OpenRouter 用 **`https://openrouter.ai/api/v1`**，RouterAI 用 **`https://routerai.ru/api/v1`**）。其模型 id 前面带有供应商：

```yaml
providers:
  - name: "openrouter"
    type: "openai"
    api_base: "https://openrouter.ai/api/v1"
    api_key: "${OPENROUTER_API_KEY}"
models:
  - model: "openrouter/qwen/qwen3.8-flash"
    reasoning_default: "medium"
```

Coddy 会从供应商背后的模型读取此类 id 的推理级别，因此 **`openrouter/qwen/qwen3.8-flash`** 提供 **`qwen3.8-flash`** 的级别。聚合器的模型列表会命名每个模型接受的请求参数（**`supported_parameters`**），Coddy 会连同模型的上下文窗口一起读取它。当列表命名统一 **`reasoning`** 对象而不是 **`reasoning_effort`** 时，所选级别会作为 **`reasoning: {"effort": "<level>"}`** 发送，**`off`** 作为 **`reasoning: {"enabled": false}`** 发送；Qwen3 的 **`chat_template_kwargs`** 开关只会发送给其列表命名它或未列出任何内容的服务器。模型的推理会在响应的 **`reasoning`** 字段中返回，并像其他推理一样显示在聊天中，聚合器为其报告的令牌数（**`usage.completion_tokens_details.reasoning_tokens`**）显示在其持续时间旁（[Web 界面](../surfaces/web-ui.md#对话记录消息类型)）。

### 兼容提供商客户端身份

对于 OpenAI 兼容和 Anthropic 兼容提供商，省略
`providers[].user_agent` 或将其留空以保持提供商 SDK 身份
（`OpenAI/Go` 或 `Anthropic/Go`）。显式的 `openai` 和 `anthropic` 值在切换提供商类型后也会保持当前协议 SDK 身份。
选择 `coddy` 以发送 `Coddy/<running version>`（在未标记构建中为 `dev`）。
该设置适用于阻塞和流式请求以及模型列表，包括未保存的提供商预览。其他提供商类型会忽略它。
在设置中，打开提供商，展开 **Advanced settings** 并选择 **User-Agent**。
保存提供商以将选择应用于新调用。

<!-- docsgen:source sha256=ea22bc82474aec90 -->

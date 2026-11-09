# 故障排除

当[快速开始](quickstart.md)的某个步骤没有按说明进行时该检查什么：找不到二进制文件、配置无法加载、提供商拒绝密钥、端口被占用、构建中缺少某个界面、机器人或钩子保持静默。每个章节都会说明症状、常见原因和修复方法，最后一节列出在寻求帮助前应收集哪些信息。除非某个章节另有说明，所有平台上的命令都相同。

## 安装后找不到二进制文件

**症状。** 在运行安装程序的终端中，`coddy: command not found`，或者编辑器报告无法启动 `coddy acp`。

**原因。** `install.sh` 会将二进制文件放入 `~/.local/bin`，并向登录 shell 的 rc 文件写入一个带保护条件的代码块，将该目录加入 `PATH`；你安装时使用的终端尚未重新读取该文件。在 Windows 上，`install.ps1` 会将 `%LOCALAPPDATA%\Programs\coddy` 添加到用户 `PATH`，同样，安装终端看不到它。编辑器和其它框架可能通过 `cmd /c` 或 `sh -c` 启动智能体，而完全没有用户 `PATH`。

**修复。** 打开一个新终端，或刷新当前终端：

```bash
source ~/.zshrc            # or ~/.bashrc
export PATH="$HOME/.local/bin:$PATH"
coddy -v
```

```powershell
$env:Path = [Environment]::GetEnvironmentVariable("Path","User") + ";" + [Environment]::GetEnvironmentVariable("Path","Machine")
coddy -v
```

为 ACP 客户端提供二进制文件的绝对路径，而不是依赖 `PATH`（`/home/you/.local/bin/coddy`、`%LOCALAPPDATA%\Programs\coddy\coddy.exe`）。当存在多个副本时，`which coddy` 会指出 shell 运行的是哪一个；用 `coddy update` 更新那一个。对于非 root 用户，从克隆仓库执行 `make install` 会落到同一个 `~/.local/bin`。详情：[安装](install.md)。

## 配置无法加载，或 `coddy -t` 报告错误

**症状。** 某个命令以 `load config: ...` 退出，`coddy -t` 打印以文件路径开头的行，或者你写入的设置没有生效。

**原因。** 加载器会读取 `--config` 或 `$CODDY_HOME/config.yaml`（默认 `~/.coddy`），当该文件不存在时使用内置默认值运行。当前目录中的 `config.yaml` 除非你显式指定，否则不会被读取，因此在错误位置编辑的文件不会产生任何变化。它的解码很宽松：未知或拼写错误的键（用 `enabled` 而非 `enable`）会被忽略而不是拒绝，形状错误的值之后会表现为异常行为。解析失败的文件在正常加载时会从 `config.yaml.bak` 恢复；下面的检查不会这样做，因此它报告的是磁盘上的文件。

**修复。** 按照失败命令加载文件的方式来运行检查；`--config PATH` 和 `--home DIR` 会像启动时那样选择它：

```bash
coddy -t
coddy serve -t
```

每个问题都会打印为 `file:line:column: what is wrong`，带一行缩进的 `fix:`，并且在模式提供相应内容时，还会有一行 `doc:`。语法错误会指出导致文件解析停止的那一行，而不是解析器归咎的那一行，启动时也会打印同一行。出错时退出状态为 1；警告（布尔值上的 `yes`、缺少 `# yaml-language-server:` 头）永远不会使检查失败，而文件缺失则属于错误。当文件没有问题后，再进一步：

```bash
coddy --dry-run                 # the problems and one status line
coddy --dry-run --test-config   # the config report and every probe
```

`--dry-run` 会探测文件中所命名的内容：目录、每个提供商的模型列表、stdio MCP 服务器的可执行文件、Telegram token，以及对于 `coddy serve` 的监听地址。报告格式与规则：[配置](configuration.md#从命令行检查文件)。

## 提供商拒绝密钥或模型 ID

**症状。** 一个轮次以提供商返回的认证错误或未知模型错误结束；`coddy --dry-run` 报告 `providers[<name>]: cannot reach ...` 或 `models[<id>]: not in the model list of provider <name>`；`coddy providers list` 打印以 `!` 开头的行。

**原因。** 必须有三件事对齐。每个 `models[].model` 的第一段必须等于一个 `providers[].name`（字母、数字、连字符和下划线，并以字母开头），并且 `agent.model` 必须是 `models` 条目之一；否则加载器会报告 `models[...]: unknown provider`。密钥按固定顺序解析：文件加载时展开的字面量 `api_key` 或 `${VAR}` 引用，然后是 `api_key_command` 的标准输出，再然后是 `NAME_API_KEY` 环境变量，其中 `NAME` 是提供商名称的大写形式，并将连字符转换为下划线（`my-proxy` 读取 `MY_PROXY_API_KEY`）。对未设置变量的引用会展开为空，请求会在没有凭据的情况下发出。没有 `api_base` 的 `type: openai` 行会与 OpenAI 本身通信；本地或第三方服务器需要带有 `/v1` 的 `api_base`，并且模型 ID 必须是该服务器列出的 ID。

**修复。**

```bash
coddy providers list     # every provider with the credential source requests actually use
coddy --dry-run          # asks each provider for its model list, checks every models[] entry
```

`providers list` 会警告某个 `openai` 行指向官方端点却没有任何可出示的凭据；`--dry-run` 会指出它尝试的地址，并且对于服务器未列出的模型，会指出它确实列出的 ID（这是警告，因为有些服务器提供的模型比列出的更多）。将密钥放入 `~/.coddy/.env`（`OPENAI_API_KEY=sk-...`），它会在文件之前被读取，且永远不会覆盖 shell。密钥被拒绝的 NeuralDeep 行可用 `coddy providers login neuraldeep` 修复，Codex 行可用 `coddy providers login codex` 修复。字段参考：[config.yaml 参考](../reference/config.md#providers)。

## 端口 12345 被占用，或 API 未提供服务

**症状。** `coddy serve` 因绑定错误退出，`coddy serve --dry-run` 报告 `httpserver` 地址以及设置它的那一行，或者进程运行时 `http://127.0.0.1:12345/` 拒绝连接。

**原因。** 地址来自 `-H` / `-P`，然后是 `httpserver.host` / `httpserver.port`，再然后是 `127.0.0.1:12345`。另一个进程占用了端口，通常是先前启动的 `coddy serve --daemon`，或者此进程通过 `httpserver.enable: false` 或 `--http=false` 关闭了 HTTP API。默认绑定为回环地址，因此按设计会拒绝另一台机器上的浏览器。

**修复。**

```bash
coddy serve status            # a daemon already up? then: coddy serve stop
coddy serve --dry-run         # binds each listen address once and releases it
coddy serve -P 12346          # or httpserver.port in the file
```

要从其他机器访问，请绑定更宽的地址并要求 token：`coddy serve -H 0.0.0.0 --auth-token <secret>`（也可以是 `CODDY_HTTP_TOKEN` 或 `httpserver.auth_token`）；没有 token 时，进程会警告它可在未经认证的情况下访问。监听地址是运行中的进程唯一无法采用的设置：在 `--daemon` 下，工作进程会自行在新地址上重启（退出状态 75 会请求调度器提供替代进程），`coddy serve install` 的 systemd 用户服务也是如此；在前台运行时，请自行重启。在 `--daemon` 下，被占用的端口会在键入命令的终端中报告，并重试直到释放。参见 [coddy serve 与守护进程](../operate/serve.md) 和 [HTTP API](../reference/http-api.md#cli-标志)。

## 构建中缺少某个界面

**症状。** `coddy serve` 因 `<key> is true but this binary has no <surface> support (rebuild with -tags <tag>)` 拒绝启动；直接运行 `coddy` 会打印用法而不是打开控制台，`coddy cli` 回答 `interactive console is not built in`；未启用任何内容的 `coddy serve` 会说 `no subsystem is enabled`。

**原因。** 控制台、HTTP API、Web 界面、调度器、记忆子智能体、消息网关、集群中继和 Prometheus 导出器都是 Go 构建标签。普通的 `make build`（或 `go build`，或 `go install ...@latest`）会生成不含这些内容的精简 ACP 二进制文件。`coddy serve` 会将二进制文件无法运行的已启用子系统视为错误而不是警告，因此原本会静默离线的机器人会按名称被拒绝。

**修复。** 使用发布构建：GitHub Release 归档、`.deb` 和 `.rpm` 包以及 Homebrew cask 都使用所有标签构建，Docker 镜像包含 API、Web 界面、调度器、记忆子智能体、控制台和网关；或者自行构建：

```bash
make build TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop"
```

`ui` 需要 Node.js 和 npm（Makefile 会先运行 `ui-build`）。`coddy serve --dry-run` 会在不启动任何内容的情况下报告此二进制文件未构建的界面。标签参考：[从源码构建](../contributing/build.md#构建标签参考)。

## Web 界面能响应 API，但没有页面

**症状。** `GET /` 返回一个简短的纯文本 404，而 `/docs/`、`/v1/models` 以及 API 的其余部分正常工作。

**原因。** 要么二进制文件链接了 `http` 但没有链接 `ui`（单页应用在构建时从 `make ui-build` 用 npm 生成的资源中嵌入），要么 `ui.enable: false` 有意以仅 API 模式运行服务器。

**修复。** 使用两个标签重新构建，至少使用 `make build TAGS="http ui"`，并确保 `PATH` 上有 Node.js 和 npm；检查文件中的 `ui.enable`。重新构建后，普通浏览器刷新即可加载新资源。发布二进制文件包含界面。参见 [从源码构建](../contributing/build.md) 和 [HTTP API](../reference/http-api.md)。

## MCP 服务器无法启动

**症状。** 会话启动，但警告将某个服务器列为已跳过，并且其工具从模型的列表中缺失；在 **设置 -> MCP 服务器** 下，该服务器显示一个盾牌按钮。

**原因。** 在 `<workspace>/.coddy/mcp.json` 中声明的服务器随检出内容一起到达，因此在默认策略 `mcp.project_trust: ask` 下，它会保持冷状态，直到你为该工作区批准那个确切的声明；警告会指出服务器、工作区、摘要以及批准它的命令，而重写已批准的条目会撤销批准。来自 `config.yaml` 和 `~/.coddy/mcp.json` 的服务器不受此限制，但它们的命令仍必须在 `PATH` 上，而被标记为 `disabled: true`（在 mcp.json 中为 `"disabled": true`）的服务器不会连接。启动失败的服务器永远不会阻塞会话：它会带着警告继续。

**修复。**

```bash
coddy mcp list               # scope, trust state and the command of every merged server
coddy mcp trust <name>       # prints the declaration, then records the receipt
coddy --dry-run              # resolves stdio commands in PATH, contacts remote servers
```

`coddy mcp list` 和 `trust` 接受 `--cwd DIR` 以用于另一个工作区。通过 HTTP，同样的决定是 `POST /coddy/mcp/{name}/trust` 和 Web 界面中的盾牌。`coddy serve --mcp-project-trust allow`（或 `coddy acp`）会为单个进程覆盖该策略，这正是 CI 任务所使用的。批准记录位于 `~/.coddy/mcp-trust.json`。指南：[MCP 服务器](../features/mcp.md#面向项目本地服务器的工作区信任)。

## Telegram 机器人保持静默

**症状。** 机器人从不回答，或者点击 `/model` 或 `/resume` 没有任何变化。

**原因。** 按出现频率排序：二进制文件没有网关标签（启动错误会指出该标签）；`gateways.telegram.enable` 为 true，但无法解析出 token（网关拒绝启动，并提示设置 `gateways.telegram.token` 或 `TELEGRAM_BOT_TOKEN`）；发送者不被允许（`default_access: admins` 或 `group:<name>` 会静默丢弃其他所有人，并且 `admins` 必须包含你的数字 Telegram 用户 ID）；消息位于群组中且没有对机器人说话（在群组中，它只响应 @mention、对其自身消息的回复或 `/clear`）；另一个进程正在轮询同一个机器人，而 Telegram 只会将每个更新交给一个长轮询。文本得到回答但点击被忽略，且 `debug` 处没有任何内容，问题出在订阅：Telegram 会记住机器人最后请求的 `allowed_updates`，而一个曾经在另一个框架下运行过的 token 可能只订阅了消息。Coddy 在每次轮询时都会请求 `message` 和 `callback_query`，因为它自己遇到过这个问题；`curl https://api.telegram.org/bot<token>/getWebhookInfo` 会显示当前生效的内容。

**修复。** `coddy --dry-run` 会根据 Bot API 检查 token 并指出机器人。已连接的机器人会在 `info` 记录 `telegram bot connected`。对于被丢弃的更新，请提高该组件的日志级别：

```bash
coddy serve --log-level "info,gateway.telegram=debug"
```

或在文件中 `logger.levels` 下进行相同设置。在 `debug` 下，每个更新都会被记录，并附带其被丢弃的原因（访问被拒绝、仅管理员聊天、未对机器人说话的群组消息、队列已满）。`warn` 处静默且 `debug` 处没有任何内容，意味着更新从未到达：检查 token、访问列表和其他轮询者。指南：[Telegram 网关](../surfaces/gateway.md#调试聊天)。

要将 Telegram 与机器人分开排查，请在设置 `CODDY_TELEGRAM_API_BASE` 的情况下，让机器人针对 [tgfake](https://github.com/EvilFreelancer/tgfake) 的假 Bot API 运行：你从本机上的页面发送消息，每次 Bot API 调用都会被列出，并且可以按需注入故障。指南：[针对假 Bot API 进行调试](../surfaces/gateway.md#针对假-bot-api-调试)。

## 钩子或子智能体定义被忽略

**症状。** `<workspace>/.coddy/hooks.json`（或 Claude Code 的 `.claude/settings.json`）中的钩子从不运行，对话记录会显示一条关于钩子文件被暂扣的通知；`spawn_agent` 回答某个定义来自未为此工作区批准的项目文件；`coddy hooks list` 或 `coddy agents list` 显示 `needs_approval`。

**原因。** 位于会话 cwd 处或之下的文件属于项目范围，遵循 `hooks.project_trust` 和 `subagents.project_trust`。在默认的 `ask` 下，它们会被解析并列出，但在运行 Coddy 的机器上为该工作区批准那个确切文件之前，不会运行或生成任何内容；批准记录绑定到文件摘要，因此编辑它会导致再次询问。你自己的文件（`~/.coddy/hooks.json`、`${CODDY_HOME}/agents`）永远不需要批准。`hooks.enable: false` 或 `subagents.enable: false` 会完全关闭该功能，只会读取 Claude Code 设置文件的 `hooks` 键，并且 `spawn_agent` 在询问模式下永远不会提供。

**修复。**

```bash
coddy hooks list [--cwd DIR]
coddy hooks trust <file>        # e.g. .coddy/hooks.json
coddy agents list [--cwd DIR]
coddy agents trust <name>
```

两个 `trust` 命令都会打印它们即将批准的内容，并记录批准记录（`~/.coddy/hooks-trust.json`、`~/.coddy/subagents-trust.json`）。从远程控制台或 ACP 客户端进行批准时，批准应位于服务器上：使用 `POST /coddy/hooks/trust` 和 `POST /coddy/subagents/{name}/trust`，并将会话工作区作为 `cwd`。Web 界面的 "设置 > 子智能体" 会列出这些定义，并用其行上的盾牌批准项目定义，这会在服务器上为屏幕上聊天所在的工作区记录批准记录。你已经信任的检出内容可以在 `project_trust: allow` 下运行。指南：[钩子](../features/hooks.md#项目文件与信任)、[子智能体](../features/subagents.md#作用域与项目信任)。

## 项目市场未同步

**症状。** 项目在 `.coddy/marketplaces.json` 中声明的源或市场未被使用：`coddy plugin marketplace update` 以类似 `? team/skills: declared by <workspace>/.coddy/marketplaces.json, not approved for this workspace` 的行结束，`plugin install <plugin>@<name>` 说该市场未获批准，并且 "设置 > 技能" 显示该行带有一条备注且其检查按钮被禁用。

**原因。** 该文件随检出内容一起到达，并决定安装什么到 `${CODDY_HOME}/skills` 中，因此其条目遵循 `skills.project_trust`：在默认的 `ask` 下，条目只有在你为该工作区批准后才会生效，而批准绑定到该条目，因此重写它的检出内容会导致再次询问。在 `deny` 下，这些条目会被列为已关闭，且永远不会使用。

**修复。** 在工作区的终端中批准它，或者使用 "设置 > 技能" 中其行上的盾牌：

```bash
coddy plugin marketplace list            # what is declared, and what waits for approval
coddy plugin marketplace trust <name>    # a marketplace by name, a source by its address
coddy plugin install <source>            # a source; a plugin of a marketplace: coddy plugin install <plugin>@<name>
```

批准本身不会安装任何内容：安装是最后一条命令。

聊天中的 `/plugin` 无法批准。你已经信任的检出内容可以在 `skills.project_trust: allow` 下运行。指南：[技能](../features/skills.md#项目市场与信任)。

## 轮次因使用限制而停止

**症状。** 轮次以提供商的限制错误结束；控制台页脚显示 `limit reached (resets 20:59)`，对话记录显示 `Usage limit reached`。

**原因。** 如果 `429` 指明的重置时间超出了重试预算可覆盖的范围，默认情况下会结束轮次。`agent.wait_for_limit_reset` 默认关闭，因为等待中的轮次会全程持有轮次锁并保持客户端流打开。

**修复。** 在控制台中，`/usage` 会打印账户明细：每个窗口及其百分比和重置时间、冷却时间、钱包。等待重置，或用 `/model` 切换模型。要让轮次自行等待，请设置：

```yaml
agent:
  wait_for_limit_reset: true
  # wait_for_limit_reset_max_ms: 14400000   # at most 4 h per turn, the default
```

顶层轮次随后会等待重置，每 20 秒报告一次 `Usage limit reached · resuming at 20:59`，并重新发出相同的调用；按 Escape 会像任何轮次一样停止等待。中心拒绝的密钥会在页脚显示 `key rejected`，并可通过 `coddy providers login neuraldeep` 修复，或在控制台中通过 `/login neuraldeep` 修复。参见 [控制台 (TUI)](../surfaces/console.md) 和 [`agent` 参考](../reference/config.md#agent)。

## 提供商回答前轮次因网络错误失败

**症状。** 轮次以 `LLM error: provider "<name>" (<address>): ...` 结束，消息以 `net/http: TLS handshake timeout`、`dial tcp ...: i/o timeout`、`connect: no route to host` 或 `connection reset by peer` 结尾。

**原因。** 与提供商的连接未能建立，或在任何输出到达前被切断：链路饱和或抖动（同一台机器上的后台任务中有大文件下载就足够）、VPN 重新连接、代理停滞。此类请求从未到达服务器，因此 Coddy 会重试，退避从 `agent.llm_retry_base_ms` 开始并翻倍，每次重试从 `agent.llm_retry_max` 的每步共享预算中占用一个名额（默认 3）。该预算在同一传输重试、首次 token 重新发出，以及同一步骤的连续无回答和输出限制恢复之间共享；当模型取得进展时会重置。只有当所有尝试都失败时，错误才会到达轮次。同一类的两种失败不会这样重试：完全无法解析的主机名（`no such host`），这是 `api_base` 错误而不是瞬时故障；以及已经显示文本后流被切断，因为重放会再次显示该文本。当重试耗尽，或流在文本之后中断时，轮次本身仍会继续：参见 [提供商在轮次中途失败](#提供商在轮次中途失败)。

**修复。** 检查错误中的地址是否是你想要的地址，然后检查它是否能从本机得到响应：

```bash
coddy --dry-run          # asks every provider for its model list at the configured address
```

当重试后错误仍然出现时，说明链路中断的时间超过了退避覆盖范围。对于不稳定的链路，提高预算；对于只能通过代理访问的提供商，在其行上设置 `proxy`：

```yaml
agent:
  llm_retry_max: 5
  llm_retry_base_ms: 2000
providers:
  - name: my-proxy
    type: openai
    api_base: https://llm.example.com/v1
    proxy: socks5h://127.0.0.1:1080
```

相反的情况同样常见：机器在 `HTTPS_PROXY` 中指定了代理（公司代理、本地转发器、从其他设置遗留的变量），而该代理停滞或破坏握手，同时提供商本身可以直连。除非另有说明，每一行都会遵循该变量，因此在应直连的行上设置 `proxy: none`；其他行保留代理。当提供商无法访问时，`coddy --dry-run` 会指出环境所选择的代理。更多内容见 [提供商代理](configuration.md#提供商代理)。

```yaml
providers:
  - name: local
    type: openai
    api_base: http://192.168.1.20:8000/v1
    proxy: none
```

字段参考：[`agent`](../reference/config.md#agent)、[`providers`](../reference/config.md#providers)。

## 轮次从不回答，或以 `stream stalled` 结束

**症状。** 模型开始回答，文本在句子中间停止；一段时间后轮次以 `LLM error: provider "<name>" (<address>): ... stream stalled: no data from the model for 5m0s` 结束，已经到达的文本保留在对话记录中。或者一次性运行（`coddy -p`）长时间没有任何输出，当它被终止时，会话日志显示 `generation was interrupted before producing a response (the model had been silent for 25m0s)`。

**原因。** 提供商接收了请求，然后停止发送。链路上没有任何信息说明它是否仍在工作：网关后面的工作进程卡住、代理或 VPN 隧道丢失了远端却没有关闭连接、网关忘记了请求。Coddy 在三个地方限制这种等待：

- 首 token 守卫，`agent.llm_first_token_timeout_ms`（90 秒），会切断没有产生任何内容的流式调用，并重新发出一次；该地址通常代表一组部署，下一次尝试会落到另一个成员上；
- 流空闲守卫，`agent.llm_stream_idle_timeout_ms`（5 分钟），会切断服务器在首批字节之后长时间没有发送任何内容的流式回答，并保留已到达的内容。该步骤随后会在暂停后再次运行，就像提供商在轮次中途的任何失败一样（[下文](#提供商在轮次中途失败)），并且只有当这也没有帮助时，轮次才会以指出停滞的错误结束；
- HTTP/2 活性 ping 会关闭对端在大约 45 秒内停止响应的连接，并重复请求；这正是区分死隧道和慢模型的方式。

两个守卫都不适用于配置了 `stream: false` 的模型：它的回答一次性到达，因此该调用唯一的限制是 `providers[].timeout_ms`，而从外部终止的运行会报告模型已沉默多长时间。

**修复。** 守卫在健康但缓慢的模型上触发，是需要提高的值，而不是缺陷：在小服务器上处理大段简述，可能会在首个 token 之前花几分钟进行提示词处理；当服务器抢占请求时，长回答可能会暂停一段时间。将守卫设置为部署所需的值，或设置为 `0` 以将其关闭，并为阻塞式模型改为设置请求时限：

```yaml
agent:
  llm_first_token_timeout_ms: 600000   # ten minutes before the first token
  llm_stream_idle_timeout_ms: 600000   # ten minutes of silence mid-answer
providers:
  - name: slow-hub
    type: openai
    api_base: https://llm.example.com/v1
    timeout_ms: 3600000                # one hour for a stream: false model's whole request
```

启用 `-log-level debug` 后，控制台会为每次模型调用向其日志写入一行 `llm call finished`（Coddy 主目录下的 `logs/cli.log`；`coddy serve` 会将其记录到 `logger.outputs` 指向的位置），包含调用持续时间、到首次回调的时间（`first_chunk`）和回调次数。回调可能只包含使用量元数据；它不一定是模型的第一个内容。

字段参考：[`agent`](../reference/config.md#agent)、[`providers`](../reference/config.md#providers)。

## 长会话变慢

启用 `agent` 调试组件，以区分本地工作与提供商调用，例如使用 `-log-level 'info,agent=debug'`。每次模型调用都有一个 `session` 和一个进程单调的 `call_id`。现有的 `llm call finished` 和 `llm call usage` 记录携带相同的键。

这些额外记录包含计时元数据，绝不包含消息正文、URL、标头、凭据或工具参数：

- `llm request latency` 报告 `context_refresh_ms`、`outbound_projection_ms`（驱逐、规则、图像和轮次上下文投影）、`pairing_ms` 和 `model_call_ms`。可选的 `first_meaningful_ms`、`first_reasoning_ms`、`first_text_ms` 和 `first_tool_ms` 用于区分内容回调和仅使用量回调。上下文刷新不包括初始系统提示词组装和压缩本身；
- `llm http request latency` 每次调用最多报告八个 HTTP 请求，包含 `request_bytes`（ContentLength，未知时为 `-1`）、`start_ms`、`round_trip_ms` 以及观察到的传输里程碑。`sdk_prepare_ms` 衡量 Codex SDK/客户端准备，包括 JSON 编码，而不是纯序列化。`connection_ms` 是连接获取时间；`wrote_request_ms` 和 `first_byte_ms` 是从 HTTP 传输入口开始的偏移量。`round_trip_ms` 在响应头或传输错误处结束，而不是流式回答结束处。协议、状态和连接复用仅在观察到时出现；
- `llm tool dispatch latency` 报告从提供商返回到进入第一个执行的工具调用之间的 `completion_to_dispatch_ms`，在限制、钩子或权限检查之前。它包括中间核算、持久化、规则发现和诊断日志记录，不包括工具执行或权限等待。

这些区间会重叠；不要将它们相加。首字节时间包括客户端、网络和服务器工作，不是服务器推理时间。响应头和流元数据可能在模型内容之前到达。ACP 和自定义的非 HTTP 提供商没有 HTTP 里程碑。没有智能体调试日志记录时，不会创建收集器或额外记录。重试共享该调用的收集器；总计和丢弃记录计数在超过其八条记录上限后仍可见。

要进行有界的实时比较，请从仓库根目录运行[合成 Go 示例](../../../examples/diagnostics/request-latency)。它使用 `config.LoadReadOnly` 以及正常的提供商/认证/代理路径，从不使用已存储的会话。选择一个已配置的流式模型，它同时提供两个请求的推理级别，并具有显式的上下文窗口。

```bash
mkdir -p dist/request-latency
go run ./examples/diagnostics/request-latency -live \
  -config "$HOME/.coddy/config.yaml" -model codex/gpt-6.1-sol \
  > dist/request-latency/live.jsonl
```

固定计划最多进行 12 次调用。上下文对按短/长/长/短顺序排列，每个都立即进行一次相同重复；推理保持为 `xhigh`。最后两对固定短上下文，并比较 `high` 与 `xhigh`。默认值为 64 KiB 和 1,800,000 归档字节、8 MiB 的消息内容总预算、120 秒调用期限和 30 分钟总时限。对于更小的已配置窗口，请减小 `-short-bytes` 和 `-long-bytes`。窗口检查使用每 token 两字节的夹具估算加上余量，而不是分词器。提供商拒绝、无效回答或过多可见输出会停止运行且不重试。

JSONL 包含计时/token 元数据和回答验证结果，绝不包含模型文本。重复会观察缓存复用；它们不会强制某种缓存状态。比较实际输入、缓存、输出和推理 token，而不仅是夹具字节。推理级别使用单独的适配器实例，因此 Codex 会话头可能不同。Codex 不支持请求的 `-max-tokens` 输出上限；示例将可见文本/推理限制为 8192 字节，并在超出时取消，而隐藏推理仅受期限限制。这衡量的是一个合成模型任务，而不是整个智能体循环，也不是真实会话缓存行为的原因。

## 提供商在轮次中途失败

**症状。** 回答在句子中间停止，会话日志显示类似 `The provider failed mid-turn (... server error 500: litellm.MidStreamFallbackError ...). The turn went on after a 5s pause (recovery 1 of 2).` 的通知。随后轮次会自行继续。在此之前，此类轮次会以错误结束，并等待用户输入 "continue"（[issue #246](https://github.com/coddy-project/coddy-agent/issues/246)）。

**原因。** 失败的是提供商的通道，而不是请求：代理返回 5xx 且其回退也失败、连接被切断、流变得静默。远程主机关闭的连接在 Linux 上显示为 `connection reset by peer`，在 Windows 上显示为 `wsarecv: An existing connection was forcibly closed by the remote host.`；两者都是切断（[issue #389](https://github.com/coddy-project/coddy-agent/issues/389)）。提供商过载也是一种通道失败（[issue #537](https://github.com/coddy-project/coddy-agent/issues/537)）。HTTP `529`，或提供商在已经开始的流中发送的过载错误（例如 `codex stream: received error while streaming: {... "code":"server_is_overloaded" ...}` 或 Anthropic 的 `overloaded_error`），在尚未显示任何文本时会先像 `503` 一样重试。当这些重试耗尽，或过载在文本之后到达时，轮次会采取下文描述的恢复措施。以前，此类错误会立即结束轮次。Codex 后端和 OpenCode 网关在 Responses API 的另外两个事件中发送同样的过载：`response.failed` 和一个扁平的 `error` 事件，提供商的 SDK 会原样交给 Coddy。Coddy 按相同字段读取它们，并以相同方式重试调用；一旦重试耗尽，错误会显示为 `codex stream: server error 503: server_is_overloaded: ...`（[issue #602](https://github.com/coddy-project/coddy-agent/issues/602)）。以前，它会显示为 `codex stream: server_is_overloaded: ...` 并在不重试的情况下结束轮次。一个已经成帧但在其 JSON 内部结束的流事件也是切断（[issue #384](https://github.com/coddy-project/coddy-agent/issues/384)）：修复前，它会以解码器自身的文本 `codex stream: unexpected end of JSON input` 或 `openai stream: undecodable SSE frame: {"choices":...` 出现，并结束轮次；现在它会显示为 `stream truncated: an event arrived with incomplete JSON (unexpected end of JSON input)`（Codex 和 Anthropic 也会将切断拼写为 `invalid character '\n' in string`）或在 OpenAI 兼容服务器上显示为 `stream truncated: an event arrived with incomplete JSON (undecodable SSE frame: {"choices":...)`，保留已经显示的文本并采取相同的恢复措施。`undecodable SSE frame:` 后跟完全不是 JSON 的文本（HTML 错误页面）意味着服务器或代理向流中写入了别的东西；轮次会以该错误结束，且不会重试。一个 Codex 轮次在模型安静约五秒后以 `stream truncated: an event arrived with incomplete JSON (unexpected end of JSON input)` 失败，一次又一次恢复，这根本不是切断：Codex 后端在模型静默时发送 `: keep-alive` 注释，而读取器把该注释当成了没有任何内容的事件。现在会跳过注释和没有数据的帧，就像在 OpenAI 兼容服务器上一直做的那样。弹性包装器仅在没有任何内容到达用户时重试调用，因此第一批文字之后的失败，或超过其退避时间的失败，会到达轮次。轮次将其视为断路器：

- 用户已经看到的文本会保留在对话记录中，并要求模型从停止处继续，而不是重新开始；
- 第一次恢复前的暂停是 `agent.llm_retry_base_ms` 的五倍（默认 5 秒），之后每次恢复前再延长四倍（20 秒、80 秒），或者采用提供商在 `Retry-After` 中要求的暂停时间（如果更长），最多 2 分钟；
- 连续两次恢复后，断路器打开，轮次以提供商的错误结束；而一次成功的调用会再次关闭它。子智能体和定时运行没有人可以告诉它们继续，因此最多可连续挺过五次（当其硬超时和轮次限制留有余地时约为六分钟），并将每次重连写入其任务日志（[子智能体](../features/subagents.md#提供商连接断开时)）。即使在 `coddy -p` 中，顶层轮次也保留两次：运行它的人会看到错误并决定是否再次运行；
- 提供商拒绝的请求（4xx）永远不会重复，使用量或速率限制（429）留给退避和 [`wait_for_limit_reset`](#轮次因使用限制而停止)，暂停期间按 Stop 会以停止结束轮次（期限会以提供商的错误结束轮次），而 `agent.llm_retry_max: 0` 会连同所有其他重试一起关闭恢复。

**修复。** 对于几秒内即可恢复的通道，无需采取任何措施。对于持续失败的通道，请检查提供商状态，或将模型放到具有更健康部署的网关后面。更长时间的故障是提高 `agent.llm_retry_base_ms` 的理由，这会拉长暂停时间。以这种方式失败的子智能体会保留其对话记录，其报告会告诉父级使用 `spawn_agent` `resume` 继续它，而不是在同一任务上启动第二个子智能体（[恢复运行](../features/subagents.md#恢复运行)）。

## 轮次以 `context window exceeded` 结束

**症状。** 轮次以 `context window exceeded: the provider refused the request because it does not fit the model's context window (...)` 结束，并且提供商自己的回答会结束该消息，例如 `404 Not Found "Context limit is 49152 tokens; prompt=51402 leaves 0 output tokens, below the minimum 16"` 或 `This model's maximum context length is 8192 tokens`。在此消息之前，此类轮次会以 `LLM error:` 和仅提供商的文本结束（[issue #490](https://github.com/coddy-project/coddy-agent/issues/490)）。

**原因。** 请求超出了模型或其背后服务器的窗口。Coddy 会在压缩阈值处折叠长轮次的早期步骤，并且当提供商以请求过大为由拒绝时，会压缩该轮次并再次询问一次，因此这条消息意味着那没有帮助：没有可折叠的内容（提示词和一个步骤）、`PreCompact` 钩子否决了压缩、摘要器失败，或提供商也拒绝了更小的请求。消息之前的日志行会说明是哪种情况，当运行了压缩时，消息本身会说明 `Coddy compacted this turn and asked again, but the provider refused the smaller request too`。当 `compaction.auto_enable` 或 `compaction.enable` 为 false 时，会立即出现相同的消息。将窗口设置得高于服务器实际运行的值会增加这种情况的可能性：触发器依据 `max_context_tokens` 进行衡量，而以更小上下文启动的本地服务器会在触发器触发之前拒绝请求。该消息会显示 Coddy 对请求的估计、它所依据的窗口，以及当提供商的回答包含这些数字时提供商自己的数字。

**修复。** 运行 `/compact`，它会将历史与当时正在回答的提示词一起折叠，然后继续任务；或启动新会话；或拆分任务，使一个轮次读取的内容更少。当提供商的限制低于消息显示的窗口时，将模型条目的 `max_context_tokens` 设置为提供商的数字，以便自动压缩及时开始（[上下文窗口](../features/compaction.md#上下文窗口)）。如果单个步骤带来的内容超过窗口容量，请降低产生它的工具的输出限制（[`tools.output_limits`](../reference/config.md#toolsoutput_limits)）。字段参考：[`models`](../reference/config.md#models)、[`compaction`](../reference/config.md#compaction)；折叠的工作原理见[少量长轮次](../features/compaction.md#少数几个长轮次)。

## 轮次在任务完成前停止

**症状。** 智能体在任务未完成时停止工作，最后一条回答下方的通知会说明原因：`Stopped after 40 steps, the step limit set by agent.max_turns. ...`，或 `The answer was cut off at the model's output limit (max_tokens). ...`。控制台会打印同一行，`coddy -p` 会将其写入 stderr，而 Telegram 聊天会将其作为独立消息收到（[issue #255](https://github.com/coddy-project/coddy-agent/issues/255)）。

**原因。** 是限制结束了轮次，而不是模型。`agent.max_turns` 默认将一个轮次的 ReAct 步骤上限设为 165。仅当你明确要禁用上限时，才将其设置为 `0`；子智能体的限制来自其定义中的 `max_turns`，然后是 `subagents.max_turns`。模型上的 `max_tokens` 会限制一次回答。

在 `max_tokens` 处停止的步骤不会立即结束轮次。重放相同请求会触及相同上限，因此智能体会在历史末尾添加一条短消息并再次询问。消息内容取决于步骤停止的位置：

- 当没有文本也没有工具调用时，思考模型将整个上限都花在了推理上。智能体会将该空步骤排除在下一次请求之外，消息要求模型保持推理简短，并将大型文件写入或编辑拆分为多个更小的工具调用；
- 当在回答中途时，文本会保留在历史中，因为你已经读过它。消息要求模型从停止处精确继续，不要重复已写内容，并保持步骤简短。续写会在文本之后流式传输。对话记录会完全按照流式传输的样子保留两条助手消息，而下一轮次的请求、Web 界面重新加载、子智能体的报告以及非流式请求的回复会将它们合并为它们本应是一段文本；
- 当在工具调用中途时，参数被截断：它们不是完整的 JSON 文档。智能体不会运行该调用，也不会要求你批准它，其行会以失败关闭，对话记录和下一次请求都不会携带它。模型在此之前完整写出的调用会照常运行。消息会指出工具，说明该调用被截断且未运行，并要求模型将其重新作为多个更小的调用。

智能体最多连续询问两次。每次重试会占用 `agent.llm_retry_max` 额外尝试中的一个以及 `agent.max_turns` 的一个步骤，而 `agent.llm_retry_max: 0` 会关闭恢复。未被截断的步骤会重新开始计数，你发来的新消息也会如此。一个也被运行了完整调用的截断步骤则不会：那些调用会重新开始 `agent.llm_retry_max` 配额，但该步骤仍然被截断了。在 `agent.max_turns` 允许的最后一步上，没有空间进行重试，轮次会以输出限制通知结束。子智能体和定时运行没有人可以告诉它们继续，因此也会以相同方式恢复。只有当模型在这些重试之后再次被截断时，轮次才会以该通知结束。

**修复。** 发送消息让智能体继续，或提高通知所指的限制。重试后的输出限制通知意味着模型的 `max_tokens` 对于其工作方式来说太小：思考模型需要为推理和回答留出空间。

```yaml
agent:
  max_turns: 240        # a larger step limit for this task
models:
  - model: openai/gpt-5.6-terra
    max_tokens: 32000
```

仅当有意禁用步骤上限时，才设置 `max_turns: 0`。

## `brew` 说 `No Cask with this name exists`

**症状。** `brew install --cask https://github.com/coddy-project/coddy-agent/releases/latest/download/coddy.rb` 打印 `Cask 'coddy' is unavailable: No Cask with this name exists` 并提供一个名称相似的 cask。或者 `brew install --cask coddy` 因 `Refusing to load cask coddy-project/coddy-agent/coddy from untrusted tap coddy-project/coddy-agent` 停止。

**原因。** 当前的 Homebrew 只会从 tap 中加载 cask。它会把 URL 或指向 `.rb` 文件的路径读取为 cask 名称，而没有 tap 携带该名称的 cask。非 Homebrew 自身的 tap 会保持不受信任，直到你信任它或通过完整名称从其中安装。

**修复。** 通过完整名称从 Coddy 的 tap 安装；Homebrew 会添加该 tap 并信任该 cask：

```bash
brew install coddy-project/coddy-agent/coddy
```

旧版 Homebrew 从 URL 安装的 Coddy 不属于任何 tap，因此 `brew upgrade` 永远不会为它提供新版本。从该 tap 重新安装；`brew uninstall` 会将 `~/.coddy` 保留在原位：

```bash
brew uninstall --cask coddy
brew install coddy-project/coddy-agent/coddy
```

参见[安装](install.md#macoshomebrew)。

## 会话磁盘几乎已满

**症状。** **设置 → 系统** 会在会话磁盘的仪表旁显示 **磁盘不足** 或 **磁盘已满**，并且 `examples/prometheus/coddy-rules.yml` 的 `CoddyDiskLow` 规则会在五分钟后触发。**原因。** 当存放 `sessions.dir`（除非另行设置，否则为 `~/.coddy/sessions`）的磁盘可用空间低于 512 MiB 这一固定值时，两者都会发出警告。会话包（其中包含上传到其中的文件及其工具输出）以及无人轮转的日志文件是增长的内容。**修复。** 在 **设置 → 会话** 中勾选旧会话并删除它们，或清空归档（[会话](../features/sessions.md)）；`coddy sessions list` 只会列出它们，而没有进程持有的会话文件夹可以手动删除。使用 `logger.rotation.max_size_mb` 和 `logger.rotation.max_files` 限制 `logger.file` 的文件，并裁剪守护进程的 `~/.coddy/logs/serve.log`，因为每次重启都会追加。页面会在 15 秒内重新读取磁盘。

## `coddy update` 拒绝覆盖由包管理器安装的二进制文件

**症状。** `coddy update` 不做任何更改，指出拥有该安装的包，打印包管理器命令并以 1 退出；或者它指向 `brew upgrade`。

**原因。** 由 `apt`、`dnf` 或 Homebrew 放到磁盘上的 `coddy` 会被该工具的数据库跟踪。就地覆盖 `/usr/bin/coddy` 会使数据库描述一个已消失的构建，而下一次 `apt upgrade` 会悄悄把旧版本放回去。`coddy update` 会询问 `dpkg-query -S` 和 `rpm -qf` 文件归谁所有，并将 `Caskroom` 或 `Cellar` 路径识别为 Homebrew 的路径。

**修复。**

```bash
sudo coddy update -y                                    # fetches the .deb or .rpm, verifies it, hands it to the package manager
sudo apt-get install ./coddy_<newer>_linux_amd64.deb    # or: sudo dnf install ./coddy_<newer>_linux_amd64.rpm
brew upgrade --cask coddy                               # the cask; a formula install takes: brew upgrade coddy
```

`~/.local/bin` 下的副本或你自己的构建不受任何此类影响，并会继续自我更新。参见[更新](update.md#由包管理器拥有的安装)。

## Termux 说 `has unexpected e_type: 2` 或 `Bad system call`

**症状。** 在 Android 上的 Termux 中，`coddy` 会立即因 `error: ".../coddy" has unexpected e_type: 2` 停止。或者它启动了，但提供商请求因 `x509: certificate signed by unknown authority` 或 `lookup ... on [::1]:53: ... connection refused` 失败，或者进程因 `Bad system call` 死亡。

**原因。** 那是 Linux 构建。面向 Android 10 或更高版本的 Termux 会通过 Android 的链接器启动每个程序，而该链接器只加载位置无关的可执行文件，Linux 构建是静态的。在 Termux 直接启动它的情况下，它会在 Android 没有的 Linux 路径中查找证书和 `/etc/resolv.conf`，并且某些 Android 版本会禁止它发起的某个系统调用。

**修复。** 安装 Android 构建，`coddy_X.Y.Z_android_arm64.tar.gz`（在 x86_64 上为 `_amd64`），安装脚本在 Termux 中会选择它。再次运行脚本会就地替换 Linux 二进制文件：

```bash
curl -fsSL https://coddy.dev/install.sh | bash
coddy -v
```

在安装期间保持打开的 Termux 会话中，只有在 `source ~/.bashrc` 之后或在新会话中才能找到 `coddy`。参见 [Android (Termux)](android.md)。

## Docker 镜像在 arm64 主机上因 `exec format error` 停止

**症状。** 在 Raspberry Pi、arm64 服务器或任何其他 `aarch64` 主机上，`docker run ghcr.io/coddy-project/coddy-agent` 会立即以 `exec /bin/coddy: exec format error` 退出，尽管 `docker image inspect` 报告 `Architecture: arm64`。注册了 `qemu-user-static` 的主机仍会在模拟下缓慢运行该镜像。

**原因。** 从 0.8.3 到 1.2.85 发布的镜像的 `linux/arm64` 变体携带了 x86-64 的 `/bin/coddy`：Dockerfile 中 `ARG TARGETARCH` 上的默认值替换了 BuildKit 正在为其构建的平台（[issue #482](https://github.com/coddy-project/coddy-agent/issues/482)）。无论内部是什么二进制文件，Docker 都会用请求的平台标记变体。

**修复。** 拉取 1.2.85 之后发布的镜像，或在主机上使用 BuildKit 从仓库构建一个（参见 [Docker](docker.md#先决条件)）。要查看变体携带了什么，请将二进制文件复制出来，因为镜像没有 shell：

```bash
docker pull ghcr.io/coddy-project/coddy-agent:latest
c=$(docker create ghcr.io/coddy-project/coddy-agent:latest)
docker cp "$c:/bin/coddy" ./coddy-from-image && docker rm "$c"
file ./coddy-from-image   # ARM aarch64 on an arm64 host
```

从源码构建时，如果第一条 `FROM` 因 `failed to parse platform` 停止，说明运行在旧版构建器上：安装 `buildx` 插件，或使用 `docker buildx build` 构建。参见[检查镜像的平台](docker.md#检查镜像的平台)。

## Windows 注意事项

- **路径。** 二进制文件是 `%LOCALAPPDATA%\Programs\coddy\coddy.exe`；配置和会话位于 `%USERPROFILE%\.coddy\` 下。使用 `$env:USERPROFILE`，而不是 `$HOME`，因为后者在 Windows PowerShell 和 Git Bash 之间有所不同。
- **Shell。** `run_command`、`api_key_command` 和钩子会按以下优先级通过 `pwsh`、然后是 Windows PowerShell、再然后是 `cmd.exe` 运行；Unix 构建选择 `bash`，然后是 `sh`。请为 PowerShell 编写命令。
- **`make`。** Makefile 需要类 Unix shell：请从 Git Bash（或 WSL、MSYS2）运行它，而不是从 `cmd` 或 PowerShell。`ui` 标签还需要 `PATH` 上有 Node.js 和 npm。
- **`npm --prefix`。** 当 `make ui-build` 因 `npm error enoent ... open '...\package.json'` 失败时，机器上的 npm 错误处理了 `--prefix`。在其目录内构建界面，然后让 `make` 获取预构建资源：

  ```bash
  (cd external/ui && npm install && npm run build:go)
  make build TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop"
  ```

- **从脚本更新。** `coddy update -y` 会不经询问地安装，并且不会启动 Coddy，因此适合没有控制台的 CI 步骤。
- **编辑 `config.yaml`。** Windows 换行符和记事本写入的字节顺序标记会被读取并丢弃，保存时会把文件的换行符恢复，因此在任何 Windows 编辑器中编辑该文件都是安全的。例外是编码：保存为 **Unicode**（UTF-16）而不是 **UTF-8** 的文件会按名称被拒绝（`the file is UTF-16 LE text, not UTF-8`），必须重新保存。记事本的 "另存为" 对话框在 "保存" 按钮旁边有编码选项；大多数其他编辑器将正确的编码称为 "UTF-8 without BOM"。
- **编辑器。** 使用绝对 `coddy.exe` 路径配置 ACP 客户端（见第一节）。

## 收集诊断信息

在寻求帮助之前，收集读者在不需要机密信息的情况下复现该设置所需的内容：

```bash
coddy -v                          # the version (dev when the build embedded none)
coddy -t                          # the config report; api_key and token values are never echoed
coddy --dry-run --test-config     # every probe, the ones that passed included
coddy providers list              # providers and the credential source each one uses
coddy mcp list                    # MCP servers, scope and trust state
coddy hooks list                  # hook files and their trust state
coddy agents list                 # subagent definitions and their trust state
coddy serve status                # the daemon: pid, config, log and worker
```

日志在每个长时间运行的命令上都遵循相同的开关：`--log-level` 接受裸级别（`debug`、`info`、`warn`、`error`）或提高某个组件的规范，例如 `info,gateway.telegram=debug`；`coddy acp` 和 `coddy serve` 还接受 `--log-output stdout|stderr|file|both`、`--log-file` 和 `--log-format text|json`。控制台从不记录到终端：其记录写入 `<home>/logs/cli.log`（`--log-file` 会移动它们）。守护进程会追加写入 `~/.coddy/logs/serve.log`。要通过协议进行复现：

```bash
coddy acp --log-level debug
```

相同的设置会持久保存在 `config.yaml` 中的 `logger` 下（`level`、`levels`、`outputs`、`file`、`format`）。参考：[配置](configuration.md)、[`logger`](../reference/config.md#logger)。

<!-- docsgen:source sha256=b17c6441bf157c48 -->

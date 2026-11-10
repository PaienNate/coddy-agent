# 环境变量

`coddy` 二进制读取的每一个变量，按领域分组，并列出读取它的文件以及解释相关功能的页面。优先级在所有地方都相同：命令行标志优先于环境变量，环境变量优先于 `config.yaml` 和内置默认值。在查阅下面任何内容之前，有两个机制会向环境提供变量。`$CODDY_HOME/.env` 在启动时读取（`internal/config/dotenv.go`），并且只设置尚未设置的变量，因此 shell、Docker 或 systemd 总是优先于该文件。`config.yaml` 中的 `${VAR}` 引用会在文件加载时从进程环境展开（`internal/config/expand.go`）；`$$` 表示一个字面美元符号，`${CWD}` 是会话占位符，而不是环境引用。仅为测试套件、示例框架和 CI 而存在的变量列在最后，不属于产品界面的一部分。

## 路径与配置

| 变量 | 读取方 | 含义 | 文档位置 |
|---|---|---|---|
| `CODDY_HOME` | `internal/config/paths.go` | 状态目录：`config.yaml`、`sessions/`、`skills/`、`providers/` 下的提供商凭据、信任回执、`.env`。默认 `~/.coddy`；`--home` 标志优先于它。 | [配置](../getting-started/configuration.md#配置文件位置与路径) |
| `CODDY_CWD` | `internal/config/paths.go` | 当客户端未发送时，会话的工作目录，在启动时转为绝对路径。默认：进程工作目录；`--cwd` 标志优先。 | [配置](../getting-started/configuration.md#配置文件位置与路径) |
| `CODDY_CONFIG` | `internal/config/paths.go` | 要加载的配置文件。默认 `$CODDY_HOME/config.yaml`；`--config` 标志优先。 | [配置](../getting-started/configuration.md#配置文件位置与路径) |

`config.yaml` 内的 `${CODDY_HOME}` 在读取文件时会被替换为解析后的 home（`internal/config/expand.go`），无论变量本身包含什么。

## 提供商与密钥

提供商的密钥按以下顺序解析：字面量 `api_key`、`api_key_command` 的标准输出，然后是变量 `NAME_API_KEY`，其中 `NAME` 是提供商名称转换为大写并将连字符替换为下划线（`rpa` 变成 `RPA_API_KEY`，`my-lab` 变成 `MY_LAB_API_KEY`）。该名称由 `internal/config/providers.go` 中的 `config.ProviderAPIKeyEnvVarName` 构建，而每个报告凭据来源的界面（`coddy providers list`、通过 HTTP 的 NeuralDeep 登录、使用量缓存）都会调用同一个函数。`codex` 行不解析任何密钥：它使用 ChatGPT 登录，不读取这三者中的任何一个，也不运行 `api_key_command`。

| 变量 | 读取方 | 含义 | 文档位置 |
|---|---|---|---|
| `NAME_API_KEY` | `internal/config/providers.go`（`EffectiveAPIKey`） | 当 `api_key` 和 `api_key_command` 都没有结果时，提供商 `name` 的回退密钥。在构建请求时读取，而不是在文件加载时读取。命令安全检查读取其 neuraldeep 行中的那一个，当没有配置这样的行时读取 `NEURALDEEP_API_KEY`。 | [配置](../getting-started/configuration.md#完整配置-schema)、[命令安全](../features/decisions.md#凭据) |
| `OPENAI_API_KEY` | `internal/config/config.go` | 除了针对名为 `openai` 的提供商的 `NAME_API_KEY` 规则之外：当文件完全没有配置 `providers` 和 `models` 时，加载器会使用此密钥合成一个 `openai` 提供商，其模型为 `openai/gpt-5.4`。 | [配置](../getting-started/configuration.md#完整配置-schema)、[Docker](../getting-started/docker.md#docker-compose) |
| `ANTHROPIC_API_KEY` | `internal/config/config.go` | 针对 `anthropic` 提供商的相同引导，使用 `anthropic/claude-sonnet-4-6`，在 `OPENAI_API_KEY` 为空时尝试。 | [配置](../getting-started/configuration.md#完整配置-schema) |
| `BRAVE_API_KEY` | `internal/config/tools.go`（`ToolSettings`） | 当 `tools.websearch.brave_api_key` 为空时的 Brave Search API 密钥；有了密钥，`brave` 引擎会使用官方 API 而不是公共结果页面。在轮次构建其工具设置时读取，因此无需写入 `config.yaml`。 | [Web 搜索](../features/web-search.md#brave-search-api-密钥) |
| `DEEPSEEK_API_KEY` | `internal/config/tools.go`（`ToolSettings`） | 当 `tools.websearch.deepseek_api_key` 为空时的 DeepSeek Search API 密钥。`deepseek` 引擎没有可抓取的公共页面，因此没有密钥时它不可用。在轮次构建其工具设置时读取，因此无需写入 `config.yaml`。 | [Web 搜索](../features/web-search.md#引擎) |
| `EXA_API_KEY` | `internal/config/tools.go`（`ToolSettings`） | 当 `tools.websearch.exa_api_key` 为空时的 Exa 搜索密钥；可选，两者都为空时使用 Exa 的 keyless 层。 | [Web 搜索](../features/web-search.md#引擎) |
| `FIRECRAWL_API_KEY` | `internal/config/tools.go`（`ToolSettings`） | 当 `tools.websearch.firecrawl_api_key` 为空时的 Firecrawl 搜索密钥；可选，否则使用 keyless。 | [Web 搜索](../features/web-search.md#引擎) |
| `PARALLEL_API_KEY` | `internal/config/tools.go`（`ToolSettings`） | 当 `tools.websearch.parallel_api_key` 为空时的 Parallel 搜索密钥；可选，否则使用 keyless。 | [Web 搜索](../features/web-search.md#引擎) |
| `TINYFISH_API_KEY` | `internal/config/tools.go`（`ToolSettings`） | 当 `tools.websearch.tinyfish_api_key` 为空时的 TinyFish 搜索密钥；可选，否则使用免费 keyless 层。 | [Web 搜索](../features/web-search.md#引擎) |
| `TAVILY_API_KEY` | `internal/config/tools.go`（`ToolSettings`） | 当 `tools.websearch.tavily_api_key` 为空时的 Tavily 搜索密钥；可选，否则使用 keyless。 | [Web 搜索](../features/web-search.md#引擎) |
| `TELEGRAM_BOT_TOKEN` | `internal/config/gateway.go`（`EffectiveToken`） | 当 `gateways.telegram.token` 为空时的机器人令牌。 | [Telegram 网关](../surfaces/gateway.md#快速开始telegram) |
| `PACHCA_BOT_TOKEN` | `internal/config/gateway.go`（`PachcaGatewayConfig.EffectiveToken`） | 当 `gateways.pachca.token` 为空时的 Pachca 机器人令牌。 | [Pachca 网关](../surfaces/pachca.md#配置) |
| `KIMI_CODE_HOME` | `internal/llm/kimi_cli_auth.go` | 用于符合条件的订阅回退的 Kimi CLI 状态目录。默认 `~/.kimi-code`；Coddy 读取访问令牌但不修改它。 | [Kimi](../features/kimi.md#登录与-cli-回退) |
| `CODEX_HOME` | `internal/llm/codex_auth.go` | Codex CLI 状态目录，当不存在由 Coddy 管理的登录时，`type: codex` 提供商会回退到它的 `auth.json`。默认 `~/.codex`。 | [配置](../getting-started/configuration.md#配置文件位置与路径) |
| `CODDY_CODEX_BASE_URL` | `internal/llm/codex_auth.go` | 为整个进程覆盖 Codex 后端端点；对于 `type: codex`，`api_base` 被有意忽略，因此配置文件永远无法重定向 OAuth 令牌。账户使用量读取会从同一端点推导其主机。 | [配置](../getting-started/configuration.md#配置文件位置与路径) |
| `CODDY_NEURALDEEP_HUB_URL` | `internal/llm/neuraldeep_auth.go` | 在 `coddy providers login neuraldeep` 期间签发密钥并响应使用量面板的中心，覆盖从 `api_base` 推导出的部署。 | [配置](../getting-started/configuration.md) |
| `CODDY_NEURALDEEP_BASE_URL` | `internal/llm/neuraldeep_auth.go` | 每个 NeuralDeep 请求都发往的 API 基础地址，覆盖 `api_base`。用于测试环境和测试。 | [控制台](../surfaces/console.md#测试) |
| `CODDY_OPENCODE_BASE_URL` | `internal/llm/opencode.go` | OpenCode 的推理和目录基础地址，覆盖 `api_base`（Go 或 Console）。用于测试环境和测试。 | [OpenCode](../features/opencode.md#两个网关) |
| `CODDY_OPENCODE_CONSOLE_URL` | `internal/llm/opencode_oauth.go` | 用于设备登录、账户发现和令牌刷新的 Console 基础地址。默认为 `https://opencode.ai/console`。用于测试环境和测试；独立于推理覆盖。 | [OpenCode](../features/opencode.md#使用账户登录) |
| `CODDY_DEVIN_API_SERVER_URL` | `internal/llm/devin_auth.go` | 每个 `type: devin` 请求都发往的 Devin API 服务器（用户 JWT、模型目录、聊天、账户使用量读取）以及登录代码交换的回退，作用于整个进程。对于 `type: devin`，`api_base` 被忽略，因此配置文件无法重定向会话令牌。用于测试环境和测试。 | [Devin](../features/devin.md#测试台与测试) |
| `CODDY_DEVIN_WEBAPP_URL`、`CODDY_DEVIN_API_URL` | `internal/llm/devin_auth.go` | 在 `coddy providers login devin` 期间，登录页面主机（`https://app.devin.ai`）以及将登录代码交换为会话令牌的 Devin API（`https://api.devin.ai`）。用于测试环境和测试。 | [Devin](../features/devin.md#测试台与测试) |
| `CODDY_ACP_DEPTH` | `internal/llm/acp_pool.go` | 由 Coddy 在 `type: acp` 提供商所启动智能体的环境中设置：比它自身的值多一（不存在表示 0）。以这种方式启动的 Coddy（`acp` 行后面的 `coddy acp`）在值为 3 时拒绝启动自己的智能体，因此互相启动的智能体会停止。不应当手动设置。 | [作为模型的 ACP 智能体](../features/acp-provider.md#智能体进程) |
| `CODDY_DEVIN_CLI_CREDENTIALS` | `internal/llm/devin_auth.go` | 当不存在由 Coddy 管理的登录时，`type: devin` 提供商回退到的 Devin CLI `credentials.toml`。默认：`$XDG_DATA_HOME/devin/credentials.toml` 或 `~/.local/share/devin/credentials.toml`，以存在者为准。 | [Devin](../features/devin.md#会话令牌来自哪里) |
| `SSH_AUTH_SOCK` | `internal/tools/ssh/agent.go` | `ssh_run_command` 在密钥文件之前尝试的 SSH 代理套接字；未设置或不可达表示仅使用密钥文件。 | [配置](../getting-started/configuration.md#ssh-远程执行) |
| `BROWSER`、`NO_BROWSER` | `internal/platform/browser.go` | `coddy providers login` 是否可以打开浏览器，以及使用什么打开。`BROWSER` 指定打开器（`none` 表示退出；`%` 模板或 `:` 列表会被忽略，转而尝试 `PATH` 上的打开器）；设置 `NO_BROWSER` 表示退出。在 Linux 上必须设置 `DISPLAY` 或 `WAYLAND_DISPLAY`，并且 SSH 会话（设置了 `SSH_CONNECTION`、`SSH_TTY` 或 `SSH_CLIENT`）在所有平台上都被视为没有浏览器。 | [安全与信任](../operate/security.md#机密) |
| `HTTP_PROXY`、`HTTPS_PROXY`、`NO_PROXY` | `internal/netx/dial.go`（Go 的 `http.ProxyFromEnvironment`） | 当 `internal/netx` 所服务的部分没有设置自己的 `proxy` 时，为其构建的 HTTP 传输所使用的代理；显式的 `proxy` 会替换它们。`http_request` 和 `webfetch` 也会遵循它们，除非调用传入了自己的 `proxy`（或 `direct`），LLM 提供商行的每个请求以及 Telegram 机器人也是如此，除非 `providers[].proxy` 或 `gateways.telegram.proxy` 是 `none` 或代理 URL。小写拼写（`http_proxy`、`https_proxy`、`no_proxy`）也会被读取；当同一变量的两种拼写都设置时，大写者优先，对于 `internal/netx` 拨号的集群隧道与旁边的请求一样。`ALL_PROXY` 不会被读取。 | [集群](../operate/swarm.md#加密与代理)、[提供商代理](../getting-started/configuration.md#提供商代理)、[Telegram 网关](../surfaces/gateway.md#代理) |

## 远程、HTTP 与集群

| 变量 | 读取方 | 含义 | 文档位置 |
|---|---|---|---|
| `CODDY_HTTP_TOKEN` | `cmd/coddy/serve.go` | 当 `--auth-token` 不存在时，`coddy serve` 在 `/v1/*` 和 `/coddy/*` 上要求的 bearer 令牌；`httpserver.auth_token` 排在两者之后。来自标志或环境变量的令牌在热重载后仍然有效，而不会被写入文件。 | [远程模式](../operate/remote.md#令牌)、[HTTP API](http-api.md#身份验证可选) |
| `CODDY_HTTP_USER` | `external/httpserver/serve.go` | Web 界面登录表单的账户名。与 `CODDY_HTTP_PASSWORD` 一起时，它会自行启用该表单，就像 `CODDY_HTTP_TOKEN` 启用 bearer 门禁一样，并且优先于文件中的 `httpserver.login.user`。显式的 `httpserver.login.enable: false` 仍会关闭该表单。 | [远程模式](../operate/remote.md#登录表单)、[HTTP API](http-api.md#web-ui-登录可选) |
| `CODDY_HTTP_PASSWORD` | `external/httpserver/serve.go` | `CODDY_HTTP_USER` 背后的密码，明文形式。它在服务器启动时用 argon2id 哈希，并且永远不会写入 `config.yaml`，这就是为什么 `$CODDY_HOME/.env` 是它的自然归属。它没有标志：命令行上的密码在 `ps` 中可见。 | [远程模式](../operate/remote.md#登录表单)、[HTTP API](http-api.md#web-ui-登录可选) |
| `CODDY_REMOTE_TOKEN` | `internal/remote/resolve.go` | 当 `--remote-token` 和匹配的 `httpserver.remotes` 条目的 `token` 都没有给出令牌时，控制台和 `coddy acp` 随 `--remote` 发送的令牌。 | [远程模式](../operate/remote.md#令牌) |
| `CODDY_SWARM_TOKEN` | `external/swarm/serve.go`，在 `cmd/coddy/serve.go` 中读取 | 当 `--swarm-auth-token` 不存在时，客户端向中继出示的 bearer 令牌；`swarm.auth_token` 排在两者之后。 | [集群](../operate/swarm.md#安全性)、[config.yaml 参考](config.md#相关环境变量) |
| `CODDY_SWARM_PAIRING_TOKEN` | `external/swarm/serve.go`，在 `cmd/coddy/serve.go` 中读取 | 当 `--swarm-pairing-token` 不存在时，节点注册时出示的凭据；每次重载时追加到 `swarm.pairing_tokens`。 | [集群](../operate/swarm.md#安全性)、[config.yaml 参考](config.md#相关环境变量) |
| `CODDY_PACHCA_API_BASE` | `internal/config/gateway.go`，在 `external/gateway/pachca/bot.go` 和 `internal/dryrun/network.go` 中读取 | Pachca 网关和 `--dry-run` 探测代替 `https://api.pachca.com/api/shared/v1` 所通信的 API 基础地址：测试服务器或本机上的测试环境。 | [Pachca 网关](../surfaces/pachca.md#运行) |
| `CODDY_TELEGRAM_API_BASE` | `internal/config/gateway.go`，在 `external/gateway/telegram/bot.go` 和 `internal/dryrun/network.go` 中读取 | Telegram 网关和 `--dry-run` 探测代替 `https://api.telegram.org` 所通信的 Bot API 源：自托管的 Bot API 服务器，或离线测试环境 [tgfake](../contributing/tgfake.md)。 | [Telegram 网关](../surfaces/gateway.md#针对假-bot-api-调试)、[配置](../getting-started/configuration.md#试运行探测文件指向的内容) |
| `CODDY_SERVE_ROLE` | `internal/serve/daemon.go` | 由 `coddy serve --daemon` 在它重新执行的进程上设置为 `dispatcher` 或 `worker`，以便每一半都知道自己是哪一个。`service` 告诉前台的 `coddy serve`，服务管理器会再次启动它，因此移动监听地址的配置变更会以状态 75 结束它，而不是等待手动重启；`coddy serve install` 的 systemd 单元会设置它，你自己的监督进程也可以设置。不要手动设置 `dispatcher` 或 `worker`。 | [自我重启](../operate/serve.md#自行重启) |
| `CODDY_UI_THEME`、`CODDY_UI_WALLPAPER` | `internal/config/ui_defaults.go`，在 `GET /coddy/info` 中提供 | 每个浏览器启动时的 Web 界面默认值：`CODDY_UI_THEME` 指定没有 `coddy_ui_theme` cookie 的浏览器打开时使用的主题，`CODDY_UI_WALLPAPER` 指定当浏览器尚未选择自己的背景时显示的服务器壁纸库中的文件。访问者自己的选择始终优先。 | [Web 界面](../surfaces/web-ui.md) |

## 更新

| 变量 | 读取方 | 含义 | 文档位置 |
|---|---|---|---|
| `CODDY_UPDATE_NOTES` | `internal/update/notes.go` | `0`、`false`、`no` 或 `off` 会阻止 `coddy update` 打印旧版本和新版本之间的发布说明，其效果与 `--no-notes` 相同。 | [更新](../getting-started/update.md#变更内容) |

## 传递给钩子的变量

钩子进程继承 Coddy 的整个环境，并在此基础上获得这些变量（`internal/hooks/runner.go`）。它们是为钩子设置的，Coddy 从不读取它们。

| 变量 | 读取方 | 含义 | 文档位置 |
|---|---|---|---|
| `CODDY_PROJECT_DIR`、`CLAUDE_PROJECT_DIR` | 钩子进程 | 会话工作目录，以两个名称提供，以便 Claude Code 钩子无需修改即可工作。 | [钩子](../features/hooks.md#钩子收到什么) |
| `CODDY_SESSION_ID` | 钩子进程 | 事件所属的会话。 | [钩子](../features/hooks.md#钩子收到什么) |
| `CODDY_HOOK_EVENT` | 钩子进程 | 事件名称，例如 `PreToolUse`。 | [钩子](../features/hooks.md#钩子收到什么) |
| `CODDY_HOME` | 钩子进程 | 解析后的状态目录（如果已知）。 | [钩子](../features/hooks.md#钩子收到什么) |

通过 stdio 启动的 MCP 服务器会获得 Coddy 的环境，加上其定义中的 `env` 映射（`internal/mcp/client.go`），其值经过了与配置其余部分相同的 `${VAR}` 展开。

## 控制台与终端

| 变量 | 读取方 | 含义 | 文档位置 |
|---|---|---|---|
| `CODDY_LANG` | `internal/docs/lang.go`（`LangFromEnv`） | 控制台 F1 帮助、`coddy docs` 以及控制台轮次所读取文档的语言（`ru`、`en` 或 `zh`，语言环境的任意拼写），优先于下面的语言环境变量；这是在未设置任何这些变量的系统上选择语言的方式，包括 Windows。`coddy docs --lang` 优先于它。 | [内置文档](../features/built-in-docs.md#语言) |
| `LC_ALL`、`LC_MESSAGES`、`LANG` | `internal/docs/lang.go`（`LangFromEnv`） | 终端的语言环境，当 `CODDY_LANG` 未设置时按该顺序读取：第一个被设置的会选定文档语言（`ru_RU.UTF-8` 是俄语，`zh_CN.UTF-8` 是中文；`C`、未翻译的语言或未设置则为英语）。 | [内置文档](../features/built-in-docs.md#语言) |
| `COLORFGBG` | `external/cli/run.go` | 与 `--theme auto` 一起时，`7` 或 `15` 的背景会选择浅色主题；其他任何值都会回退到深色。 | [控制台](../surfaces/console.md#标志) |
| `COLORTERM` | `external/cli/theme.go` | `truecolor` 或 `24bit` 会将控制台调色板切换为 24 位色。 | [控制台](../surfaces/console.md) |
| `SSH_CONNECTION`、`SSH_TTY` | `external/cli/tui/terminal.go` | 单独的 Escape 在 SSH 上 100 毫秒后解析，本地为 10 毫秒。 | [键盘](keyboard.md) |
| `COLUMNS` | `internal/skills/commands.go` | `coddy skills list` 表的宽度；默认 100，小于 40 的值会被忽略。 | [技能](../features/skills.md) |

## Android (Termux)

仅由 Android 构建读取（`internal/platform/android.go`、`android_init.go`），Termux 会在那里导出它们。无需手动设置。

| 变量 | 读取方 | 含义 | 文档位置 |
|---|---|---|---|
| `TERMUX__PREFIX`、`PREFIX` | `internal/platform/android.go` | Termux 前缀，两者中第一个包含绝对路径的；默认 `/data/data/com.termux/files/usr`。Coddy 从中取得 `etc/tls/cert.pem` 和 `tmp`，以及其 `bin` 中用于 `/bin/...` 或 `/usr/bin/...` 路径的程序。 | [Android (Termux)](../getting-started/android.md#coddy-在-android-上所做适配) |
| `TERMUX_APP__DATA_DIR`、`TERMUX_APP__LEGACY_DATA_DIR` | `internal/platform/android.go` | 以其两个名称表示的应用数据目录。当 Coddy 本身是这样启动时，其下的程序会通过 `/system/bin/linker64` 启动；没有这些变量时，前缀所在的目录代表它。 | [Android (Termux)](../getting-started/android.md#coddy-在-android-上所做适配) |
| `SSL_CERT_FILE` | Go 的 `crypto/x509` | 当未设置且文件存在时设为 `$PREFIX/etc/tls/cert.pem`，以便 Termux CA 包在 Android 的存储旁边被信任。 | [Android (Termux)](../getting-started/android.md#coddy-在-android-上所做适配) |
| `TMPDIR` | Go 的 `os.TempDir` | 当未设置时设为 `$PREFIX/tmp`：Go 的 Android 默认值 `/data/local/tmp` 对应用不可写。 | [Android (Termux)](../getting-started/android.md#coddy-在-android-上所做适配) |
| `TERMUX_EXEC__PROC_SELF_EXE` | 子进程 | 对于 Coddy 通过系统链接器启动的程序，设置为该程序的路径，此时 `/proc/self/exe` 不会命名它；对于其他每个子进程都会丢弃。 | [Android (Termux)](../getting-started/android.md#coddy-在-android-上所做适配) |

## Docker 与 compose

这些变量由 compose 文件和 `Dockerfile` 读取，而不是由二进制读取。在容器内，镜像会设置 `CODDY_HOME=/home/user/.coddy`、`CODDY_CWD=/workspace` 和 `CODDY_CONFIG=/home/user/.coddy.yaml`，并且两个 compose 文件都用挂载的 `/home/user/.coddy/config.yaml` 覆盖 `CODDY_CONFIG`。

| 变量 | 读取方 | 含义 | 文档位置 |
|---|---|---|---|
| `CODDY_IMAGE` | `docker-compose.yml` | 要运行的镜像；默认 `ghcr.io/coddy-project/coddy-agent:latest`。 | [Docker](../getting-started/docker.md#docker-compose) |
| `CODDY_COMMAND` | `docker-compose.yml`、`docker-compose.dev.yml` | 容器的命令行；默认 `serve -H 0.0.0.0 -P 12345`。 | [Docker](../getting-started/docker.md#docker-compose) |
| `CODDY_HTTP_BIND` | 两个 compose 文件 | 端口发布到的主机地址；默认 `127.0.0.1`，`0.0.0.0` 会将其开放到网络。 | [Docker](../getting-started/docker.md#docker-compose) |
| `CODDY_HTTP_PORT` | 两个 compose 文件 | 发布到容器 12345 端口的主机端口。 | [Docker](../getting-started/docker.md#docker-compose) |
| `CODDY_CONFIG`、`CODDY_CWD`、`CODDY_HOME` | 两个 compose 文件 | 在主机上，分别挂载为配置文件、工作区和状态目录的路径（默认 `./config.yaml`、`./workspace`、`./coddy_home`）。 | [Docker](../getting-started/docker.md#docker-compose) |
| `OPENAI_API_KEY`、`ANTHROPIC_API_KEY`、`DEEPSEEK_API_KEY`、`TELEGRAM_BOT_TOKEN` | 两个 compose 文件 | 从主机传递到容器中，未设置时为空。 | [Docker](../getting-started/docker.md#docker-compose) |
| `CODDY_VERSION`、`CODDY_BUILD_TAGS` | `docker-compose.dev.yml` | 本地构建镜像的标签和逗号分隔的构建标签。 | [Docker](../getting-started/docker.md#docker-compose) |

## 开发、测试与 CI

已发布的二进制不会读取其中任何一个。它们为贡献者而存在，列出来是为了让在代码树中执行 grep 时不会出现未记录的名称。

| 变量 | 读取方 | 含义 | 文档位置 |
|---|---|---|---|
| `CODDY_HOOK_LINT`、`CODDY_HOOK_FORMAT`、`CODDY_HOOK_DOCS`、`CODDY_HOOK_TESTS`、`CODDY_HOOK_SKIP` | `scripts/checks.sh`，由 `.githooks/pre-commit` 运行 | 提交门禁：`CODDY_HOOK_LINT=0` 跳过 linter，`CODDY_HOOK_FORMAT=0` 跳过 SPA 的 Prettier 检查（当提交暂存 SPA 文件时运行），`CODDY_HOOK_DOCS=0` 跳过文档检查，`CODDY_HOOK_TESTS=fast|full|matrix` 添加测试，`CODDY_HOOK_SKIP=1` 绕过门禁。 | [智能体说明](../../../AGENTS.md) |
| `CODDY_UI_BACKEND` | `external/ui/vite.config.ts` | 在开发 SPA 时，Vite 开发服务器将 `/v1`、`/coddy`、`/swarm`、`/docs` 和 `/openapi.*` 代理到的 `coddy serve` 源。 | [资源索引](../../assets/INDEX.md) |
| `CODDY_LIVE_MODEL`、`NEURALDEEP_API_KEY` | `internal/agent/live_neuraldeep_test.go` | 启用针对 NeuralDeep 端点的实时探测；没有两者时测试会被跳过。 | 文件头 |
| `NEURALDEEP_API_KEY` | `internal/llm/decisions_live_test.go`、`internal/agent/decisions_live_e2e_test.go` | 启用针对真实决策端点的实时探测：传输探测和完整轮次端到端探测，没有密钥时两者都会被跳过。 | [命令安全](../features/decisions.md#测试) |
| `CODDY_BIN`、`CODDY_CONFIG`、`CODDY_HOME`、`CODDY_CWD`、`SESSION_ROOT`、`SESSION_ID`、`BASE_URL`、`MODEL`、`WORK_DIR`、`CODDY_CHAT_PROFILE`、`NEURALDEEP_API_KEY`、`RPA_API_KEY` | `examples/` 下的框架脚本 | 端到端脚本驱动哪个二进制、配置、服务器和模型；每个脚本的文档字符串会列出自己的内容。 | [示例](../../../examples/README.md) |
| `CODDY_TGFAKE_BIN` | `examples/gateway/tg_e2e_offline.sh`、`examples/shared/wake_e2e_common.py`、`examples/cli/capture_wake.py`、`external/ui/scripts/telegram-mini-app-check.mjs`；由 **Telegram 测试环境 (tgfake)** CI 任务设置 | 预构建的 tgfake 二进制，用于启动，而不是在 `go.mod` 固定的版本上构建 `github.com/EvilFreelancer/tgfake/cmd/tgfake`。 | [使用 tgfake 的集成测试](../contributing/tgfake.md) |
| `GITHUB_TOKEN` | `.github/workflows/*.yaml` | 发布、镜像和合并时打标签任务所使用的 GitHub Actions 令牌。二进制不会读取它：`coddy update` 以匿名方式与发布 API 通信。 | 工作流文件 |
| `GO_WANT_MCP_HELPER`、`GO_WANT_MCP_MARKER`、`GO_WANT_TRUST_MARKER`、`GO_WANT_CONFIG_RELOAD_MCP`、`CODDY_MCP_MARKER_FILE`、`MCP_HELPER_TOKEN`、`TRUST_MARKER_FILE` | `internal/mcp`、`internal/session` 和 `external/httpserver` 中的 MCP 和信任测试 | 让测试二进制将自身重新执行为一个桩 MCP 服务器，并报告它运行的位置。 | 测试文件 |
| `CODDY_NETX_PROXY_PARITY` | `internal/netx` 测试 | 让测试二进制将自身重新执行为子进程，以比较 `netx` 拨号隧道所使用的代理与 `net/http` 发送旁边请求所使用的代理，且每个代理变量的两种拼写都已设置。 | 测试文件 |
| `CODDY_TEST_DOTENV_A`、`CODDY_TEST_DOTENV_B`、`CODDY_TEST_DOTENV_C`、`CODDY_EXPAND_TEST`、`CODDY_CHECK_TEST_PORT` | `internal/config` 测试 | `.env` 加载器、`${VAR}` 展开和 `-t` 检查的测试夹具。 | 测试文件 |
| `CODDY_TEST_HOME`、`CODDY_TEST_WORKSPACE`、`CODDY_TEST_SESSIONS_ROOT`、`CODDY_TEST_SKILLS_DIR`、`CODDY_TEST_PREFERRED_SESSION_ID`、`CODDY_TEST_ACP_HELPER`、`CODDY_TEST_MCP_HELPER`、`CODDY_TEST_HTTP_SETTINGS_MCP_HELPER`、`CODDY_TEST_SLOW_MCP_STARTED`、`CODDY_PROJECT_MCP_STARTED` | `cmd/coddy`、`internal/agent`、`internal/mcp` 和 `external/httpserver` 下的 BDD 框架和桩辅助程序 | 将重新执行自身的测试二进制指向场景的临时 home、工作区和会话根目录，并让桩 MCP 服务器报告它已启动。 | 测试文件 |
| `RELEASE_SIGNING_KEY` | `cmd/releasesign`、`.github/workflows/release-binaries.yaml` | 为发布的 `SHA256SUMS` 签名的密钥；仓库机密，绝不是运行中的 Coddy 的变量。 | [构建](../contributing/build.md#发布签名) |
| `SEC_SCANNERS`、`SEC_FAIL_TRIVY`、`SEC_FAIL_SEMGREP`、`SEC_FAIL_GOVULNCHECK`、`GOVULNCHECK_TAGS`、`SEC_DOCKER`、`SEMGREP_CONFIGS` | `scripts/security-scan.sh` | `make security` 运行哪些 AppSec 扫描器，门禁处于什么状态（trivy 漏洞/机密发现为 `CRITICAL`，semgrep 为关闭，govulncheck 为被调用的易受攻击函数），govulncheck 加载代码所用的构建标签，是否允许固定的 docker 回退，以及 semgrep 加载哪些注册表包。 | [AppSec 扫描](../contributing/security-scanning.md) |
| `SEMGREP_APP_TOKEN` | `scripts/security-scan.sh`，传递给 semgrep | 可选的 Semgrep 注册表令牌：解除匿名速率限制并启用 Pro 规则。Fork 的拉取请求永远看不到它。 | [AppSec 扫描](../contributing/security-scanning.md) |

<!-- docsgen:source sha256=e492af1e16cd6dd0 -->

# Devin

`type: devin` 提供商为 Coddy 提供 Devin（Cognition）账户中的模型：Claude、GPT、Gemini、SWE、Kimi、GLM 以及该账户计划所服务的其余系列。你按照 Devin CLI 的方式通过浏览器登录，或复用 `devin auth login` 已持有的登录。这里 Devin 只是后端：智能体始终是 Coddy 自己的，在控制台、Web 界面、ACP 编辑器和 Telegram 网关等每个界面上都使用自己的系统提示词、工具、权限门和 ReAct 循环。

用量会像 Devin CLI 会话一样计费到 Devin 账户。

## 登录

```bash
coddy providers login devin
```

该命令会在浏览器中打开 Devin 登录页面并等待。你登录后，页面会将浏览器送回此机器上的 `http://127.0.0.1:<port>/callback`，Coddy 在那里接收授权码，将其换取会话令牌（PKCE，与 `devin auth login` 所做的交换相同），并通过用它签发用户 JWT 来验证令牌。然后它会打印账户并写入三项内容：

- 会话令牌写入 `$CODDY_HOME/providers/devin/devin-auth.json`，模式为 `0600`；
- 提供商行以及账户可用的每个系列一个模型写入 `config.yaml`；
- 写入 `agent.model`，当配置中还没有该配置项，或已有的配置项运行在一个无法响应的提供商上时（示例配置的 `openai` 行且其密钥未设置）。

```text
$ coddy providers login devin
Open https://app.devin.ai/auth/cli/continue?...
Sign in there; the page then sends the browser back to this machine.
If the browser runs on another machine, that last page will not load: copy its address from the address bar and paste it here.
Waiting for the sign-in to finish...
Signed in as Jane <jane@example.com>. Credential stored at /home/jane/.coddy/providers/devin/devin-auth.json
Updated /home/jane/.coddy/config.yaml: provider devin, agent.model devin/claude-opus-5, 39 models (devin/claude-opus-5, devin/claude-fable-5-1, devin/claude-sonnet-5, ...)
```

在通过 SSH 访问的机器上，没有可打开的浏览器，而且笔记本电脑上的浏览器也无法访问服务器的环回端口。在笔记本电脑上打开打印出的地址，登录；当浏览器落到无法加载的 `127.0.0.1` 页面时，从地址栏复制该地址并粘贴到终端：它携带授权码，登录将以相同方式完成。不是此类地址的一行会被拒绝并给出提示，等待继续。等待会在十分钟后结束，或按 Ctrl-C 提前结束。

在控制台中，`/login devin` 执行相同的登录并写入相同的三项内容，随后控制台无需重启即可运行新模型；另一台机器上的浏览器地址会粘贴到编辑器中（[控制台](../surfaces/console.md#登录提供商-login)）。

### 从设置登录

**设置 → LLM 提供商**下的 Devin 行提供 **使用 Devin 登录**。它通过 `POST /coddy/providers/{name}/devin-auth/device` 启动相同的 PKCE 流程，使用当前未保存表单中的代理，并打开验证页面。当浏览器无法访问 Coddy 的环回回调时，将完整回调地址粘贴到登录链接下方的字段中。该行随后显示账户和 **退出登录**。令牌保留在 Coddy 服务器的 `$CODDY_HOME/providers/<name>/devin-auth.json`；**全部保存**只写入提供商行，绝不写入令牌。

`--no-config` 存储令牌，且不修改 `config.yaml`。第二次登录只添加缺失的内容：文件中已有的行和模型保持你写入时的样子。

### 复用 Devin CLI 登录

```bash
coddy providers login devin --devin-cli
```

安装并登录 Devin CLI 后，无需浏览器：该命令通过用 CLI 自身登录签发用户 JWT 来检查它，并将模型目录发布到 `config.yaml`。Devin CLI 登录是一个账户，因此只服务一行：唯一的 devin 行，或存在多个时名为 `devin` 的行（[同一提供商类型的多个配置文件](../getting-started/configuration.md#同一提供商类型的多个配置文件)）；`--devin-cli` 拒绝另一行，该行需使用自己的登录。令牌保留在 Devin CLI 存放它的位置；Coddy 在每次请求时读取它，且从不写入该文件。因此 `devin auth logout` 也会结束 Coddy 的登录。

## 会话令牌来自哪里

请求使用以下第一个持有令牌的来源：

1. 该行的 `api_key`、`api_key_command`，或 `DEVIN_API_KEY` 变量（对于名为 `devin` 的行；名称遵循通常的 `NAME_API_KEY` 规则）。值是 Devin 会话令牌，`devin-session-token$...`；裸令牌会获得该前缀；
2. Coddy 管理的登录，`$CODDY_HOME/providers/<name>/devin-auth.json`；
3. Devin CLI 登录，用于其服务的那一行（唯一的 devin 行，或多个中名为 `devin` 的行），`credentials.toml` 位于 `$XDG_DATA_HOME/devin` 或 `~/.local/share/devin` 下（macOS 上还有 `~/Library/Application Support/devin`，Windows 上为 `%LOCALAPPDATA%\devin`）。`CODDY_DEVIN_CLI_CREDENTIALS` 显式指定该文件。

`coddy providers list` 显示某行使用的是哪一个来源，并对令牌做掩码：

```text
  devin (devin): signed in to Devin (jane@example.com, devin-session-token$…Q2xw)
```

`coddy providers logout devin` 只删除 Coddy 管理的文件。当存在 Devin CLI 登录时，该行仍通过它工作，命令会说明这一点。会话本身在 Devin 侧保持有效，直到你在那里结束它。

启动时，`coddy acp` 和 `coddy serve` 的 HTTP 服务器会为每个 devin 行记录一行 `devin credential`，指明它将使用的来源（`DEVIN_API_KEY` 风格的变量在设置时，因为它排在第一位），或警告没有来源，并且对于 Devin CLI 登录不服务的行，说明它服务哪一行。

## 模型与推理级别

Devin 模型目录将模型的每个变体列为自己的 id：Claude Opus 5 以 `claude-opus-5-low`、`-medium`、`-high`、`-xhigh` 和 `-max` 的形式出现，外加付费变体，例如 `-fast`。Coddy 每个系列提供一个模型，并通过推理级别到达各变体，因此登录会写入如下条目：

```yaml
models:
  - model: devin/claude-opus-5
    reasoning_levels: [low, medium, high, xhigh, max]
    reasoning_default: medium
    max_context_tokens: 1000000
    multimodal: true
```

在 `devin/claude-opus-5` 上以 `high` 级别进行的一轮会作为 `claude-opus-5-high` 发送。级别、默认值、上下文窗口和图像支持都按系列来自目录；只有一个变体的系列会得到 `reasoning_levels: []`，且其推理选择器会被隐藏。当某个系列有 `none` 变体时，选择器会在 `none` 旁边提供 `off`，两者都会选中该变体。系列缺少的级别会回退到该系列的默认变体；`minimal` 会采用 `minimal`、`none` 或 `low` 中该系列拥有的第一个。

任何完整的变体 id 也可以作为模型使用，例如 `devin/claude-opus-5-high-fast`：目录已知的 id 会按原样发送，无论推理级别如何。目录未列出的 id 也是如此，此时由 Devin API 服务器对其作出响应。

模型条目上的 `max_tokens` 会限制输出上限；没有它时，使用变体在目录中的自身限制。当提示词加上上限无法放入上下文窗口时，该上限会被降低，因为服务器会以不透明的内部错误响应超出部分，且任何重试都无法修复。除非有推理级别生效，否则会发送模型上配置的 `temperature`；没有推理级别时，请求会携带 `1.0`，与 Devin 客户端所做的一样。

思考内容会像任何其他提供商一样流式传输到对话记录中。带签名的推理块（Claude、Gemini）会与助手消息一起保留，并在同一变体的下一次请求中重放，因此模型会在工具调用之间延续自己的思维链。

## 账户用量

`type: devin` 行还会报告账户配额，读取自 Devin API 服务器的席位管理 `GetUserStatus` RPC - 与编辑器所做的调用相同 - 使用该行自己的会话令牌且**不**使用用户 JWT：只要请求元数据携带用户 JWT，服务器就会从其响应中省略 `plan_status`（配额字段），因此此读取是唯一不能签发 JWT 的 Devin 调用。响应通过共享用量面板到达每个界面：控制台页脚和 `/usage`、Web 界面上下文浮层和横幅、`GET /coddy/providers/{name}/usage` 以及 `provider_usage` 更新（[控制台](../surfaces/console.md#视觉模型)、[配置](../getting-started/configuration.md)）。

面板显示的内容就是计划所报告的内容：

- **配额**计划读取 `day` 和 `week` 计量器及其重置时间；服务器报告的是*剩余*百分比，因此面板将其显示为已用。隐藏某个配额窗口的计划会跳过该计量器；
- **ACU** 计划从已用/限制对中读取单个 `acu` 计量器，没有重置时钟；
- **credits** 计划、未知计费策略，或仅指明计划的状态，只读作计划本身 - Coddy 不会虚构服务器未发送的计量器、请求计数或重置，缺失的配额会读作 `quota unavailable`，而不是 `0%`。

计划的配额并不是唯一能阻止一轮的东西。模型可能因自身速率限制而拒绝一轮 - `HTTP 429 resource_exhausted: Reached free model rate limit ... Your limit will reset in 20 minutes` - 并且 `GetUserStatus` 的任何字段都不携带该限制，因此当该模型拒绝每个请求时，`day` 和 `week` 计量器仍读作剩余 100%。Devin 在拒绝时不发送 `Retry-After` 头，而是将窗口重新打开的时刻写入消息中。Coddy 会读取该短语，因此重置时间会到达面板，且该轮不会以一个裸错误结束；同时 [`agent.wait_for_limit_reset`](../reference/config.md#agent) 会等待该限制结束并重新发出相同调用，面板中会显示倒计时。在遇到此类限制之前，没有任何东西会显示它，因为账户不报告其计数器。

该行上的 `usage_limits_panel: false` 会在所有界面隐藏面板，并停止 `GetUserStatus` 读取。该协议未由 Cognition 发布，因此读取遵循编辑器当前的线上格式（3.10.31 描述符），并可能在 Devin 改动它时失效。

## 配置

```yaml
providers:
  - name: devin
    type: devin
    # api_key: devin-session-token$...   # optional, wins over the logins
    # proxy: none                        # the route of every request of the row

models:
  - model: devin/claude-sonnet-5
    reasoning_levels: [low, medium, high, xhigh, max]
    reasoning_default: medium
```

`api_base` 对 `type: devin` 被忽略：会话令牌只会发往 Devin API 服务器。服务器来自登录本身：由专用部署服务的账户会将其与令牌一起记录，而用户 JWT 可将聊天迁移到账户自己的服务器。`proxy` 会统一路由登录、目录和聊天（[提供商代理](../getting-started/configuration.md#提供商代理)）。

`coddy --dry-run` 会向每个 devin 行询问其目录，并对照系列列表检查已配置的模型；其中的完整变体 id 会作为警告报告，因为系列列表不命名变体。

## 工作原理

该提供商使用 Devin CLI 和 Devin Desktop 与 Devin API 服务器通信的协议：带 protobuf 消息的 Connect-RPC。会话令牌会被换取为用户 JWT（`GetUserJwt`，缓存至其过期前不久），目录来自 `GetCliModelConfigs`（按账户缓存三十分钟），而一轮是一次 `GetChatMessage` 服务器流，由 gzip 压缩帧组成，携带文本、思考内容、工具调用片段、用量和停止原因。Coddy 的系统提示词作为请求的系统提示词传输，其工具作为工具定义传输，工具结果作为工具消息传输。用量会计入提示词缓存：报告的输入会加上未缓存部分、缓存写入和缓存读取，而缓存读取会报告为缓存输入。

Protobuf 字符串必须是有效 UTF-8，服务器会以 `HTTP 400 invalid_argument: an internal error occurred` 拒绝违反该规则的请求，无论原因如何都是相同响应。因此，Coddy 会在发送前将每个不是 UTF-8 的字节序列替换为 U+FFFD，无论它来自哪里：`grep` 匹配项、来自旧编码文件的命令输出，或非图像附件。JSON 提供商会从 JSON 编码器获得相同的替换。

该协议未由 Cognition 发布。Coddy 遵循官方客户端发送的内容，因此 Devin 一侧的改动可能会破坏该提供商，直到 Coddy 跟进。

服务器在任何文本到达调用方之前以错误结束的流，会按通常规则重试（速率限制、`resource_exhausted`，计为 HTTP 429）；一旦文本已流式传输，就不会重试，因此不会重复显示任何增量。在结束帧之前被截断的流会保留其已交付的文本，并丢弃未完成答案的工具调用。

## 测试台与测试

`CODDY_DEVIN_API_SERVER_URL`、`CODDY_DEVIN_WEBAPP_URL` 和 `CODDY_DEVIN_API_URL` 会为整个进程替换 API 服务器、登录页面和授权码交换；`internal/devinfake` 是离线测试台，模拟这三者，而 `features/devin_provider.feature` 在没有账户和网络的情况下，针对它驱动登录、Devin CLI 登录、完整智能体轮次以及一个非 UTF-8 的工具结果。

<!-- docsgen:source sha256=0923f6495860e454 -->

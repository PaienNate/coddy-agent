# 即时通讯网关

即时通讯网关让你可以直接从 Telegram 等聊天应用驱动 Coddy 智能体。智能体运行与 HTTP UI 或 ACP 模式相同的 ReAct 循环、工具和技能，网关只是传输层。

本页介绍 Telegram 机器人以及网关的通用部分。Pachca（Пачка）集成机器人有单独页面：[Pachca 网关](pachca.md)。

## 目录

- [概览](#概览)
- [构建标签](#构建标签)
- [快速开始（Telegram）](#快速开始telegram)
- [配置参考](#配置参考)
  - [访问级别](#访问级别)
  - [会话隔离模式](#会话隔离模式)
  - [按聊天覆盖](#按聊天覆盖)
  - [用户组](#用户组)
- [运行网关](#运行网关)
- [调试聊天](#调试聊天)
- [机器人交互模型](#机器人交互模型)
  - [私聊](#私聊)
  - [回复](#回复)
  - [群聊](#群聊)
  - [命令](#命令)
- [即时通讯需要什么，以及在哪里说明](#即时通讯需要什么以及在哪里说明)
- [编写新适配器](#编写新适配器)
  - [1. 实现 Adapter 接口](#1-实现-adapter-接口)
  - [2. 添加 serve_name.go 及其桩](#2-添加-serve_namego-及其桩)
  - [3. 实现 acp.UpdateSender](#3-实现-acpupdatesender)
  - [4. 添加构建标签](#4-添加构建标签)
  - [5. 在 coddy serve 中注册子系统](#5-在-coddy-serve-中注册子系统)
- [聊天和浏览器中的同一会话](#聊天和浏览器中的同一会话)
- [Mini App](#mini-app)
- [被唤醒的轮次会进入聊天](#被唤醒的轮次会进入聊天)
- [用户发送的照片和文件](#用户发送的照片和文件)
- [智能体查看过的图片](#智能体查看过的图片)
- [会话生命周期](#会话生命周期)
- [安全说明](#安全说明)

---

## 概览

```
Telegram / Pachca / future messengers
         │  polling / events history
         ▼
  external/gateway/          ← build tag: gateway | gateway.telegram | gateway.pachca
    Hub (goroutine per adapter, auto-restart)
         │
         ▼
  sessionstore               ← maps chat+user context → Coddy session ID
         │                     /clear replaces the stored ID, /resume binds it
         │                     to a session chosen from the server's list
         ▼
  session.Manager            ← shared with coddy acp / coddy serve
    HandleSessionPromptWithSender(...)
         │
         ▼
  ReAct agent loop           ← identical to HTTP and ACP paths
  (tools, skills, MCP, LLM)
         │
         ▼
  Sender (per-message)       ← buffers agent output, sends back to chat
```

多个网关（目前是 Telegram 和 Pachca）运行在同一进程中，各自作为独立子系统，拥有自己的会话映射，但共用同一个会话管理器。

它们也与 `coddy serve` 启动的其他所有内容共享该管理器。一个进程，一个
`session.Manager`：聊天对话就是普通的 Coddy 会话，发生时可被列出，
并可在 Web 界面中打开。

---

## 构建标签

| 标签 | 包含 |
|-----|----------|
| `gateway.telegram` | 仅 Telegram 适配器 |
| `gateway.pachca` | 仅 Pachca 适配器 |
| `gateway` | 所有适配器（Telegram 和 Pachca） |

`gateway` 是推荐完整集合的一部分，因此发布的二进制、软件包和已发布镜像都包含它。仅在需要更精简的二进制时单独构建：

```bash
# Telegram only
make build TAGS="gateway.telegram"

# Pachca only
make build TAGS="gateway.pachca"

# All gateways
make build TAGS="gateway"

# The full set (what a release ships)
make build TAGS="http ui scheduler memory cli gateway swarm metrics browser"
```

在 `config.yaml` 中启用但未编译进二进制的机器人会导致 `coddy serve` 启动错误，并指出该标签；其他所有子命令不受影响。

---

## 快速开始（Telegram）

**第 1 步 - 创建机器人**

在 Telegram 上与 [@BotFather](https://t.me/BotFather) 对话：

```
/newbot
```

保存它返回的 token。**切勿将 token 提交到版本控制。** 将其存储在环境变量中：

```bash
export TELEGRAM_BOT_TOKEN="<your-token>"
```

**第 2 步 - 找到你的 Telegram 用户 ID**

向 [@userinfobot](https://t.me/userinfobot) 发送任意消息。它会回复你的用户 ID。该 ID 会成为 `admins` 列表条目。

**第 3 步 - 添加网关配置**

在 `~/.coddy/config.yaml`（或 `config.yaml` 所在的任何位置）中添加：

```yaml
gateways:
  telegram:
    enable: true
    token: "${TELEGRAM_BOT_TOKEN}"
    admins: [98874093]           # your Telegram user ID
    default_access: "all"
    default_isolation: "individual"
```

**第 4 步 - 构建并运行**

```bash
make build TAGS="gateway.telegram"
./build/coddy serve --config ~/.coddy/config.yaml
```

打开 Telegram，找到你的机器人，发送消息。智能体会在同一聊天中回复。

---

## 配置参考

所有网关配置都位于 `config.yaml` 中的 `gateways` 键下。使用捆绑 UI 运行 `coddy serve` 时，相同字段可在 **设置 → 网关 → Telegram** 下编辑；`gateways` 块通过 `GET`/`PUT /coddy/config` 往返读写，因此在 UI 中保存设置会保留它（机器人 token 会完整显示，仅在可信网络上使用）。

```yaml
gateways:
  telegram:
    enable: false
    # Bot token. Optional: leave empty (or omit) to read it from the TELEGRAM_BOT_TOKEN
    # environment variable (e.g. via .env), the same way provider api_key falls back to
    # NAME_API_KEY. When telegram is enabled but no token can be resolved, the gateway
    # logs a warning and skips the bot instead of failing config validation.
    token: "${TELEGRAM_BOT_TOKEN}"

    # How the bot reaches the Bot API (see "Proxy" below). Left out, or inherit,
    # it follows HTTPS_PROXY, HTTP_PROXY and NO_PROXY of the Coddy process;
    # none connects directly; a URL (http, https, socks5, socks5h) goes through it.
    # proxy: none
    # proxy: "socks5h://127.0.0.1:1080"
    # proxy: "http://proxy.example.com:3128"

    # Bot API 10.1 Rich Messages (see "Rich Messages" below). Default false.
    rich_messages: true

    # Telegram user IDs with elevated privileges.
    # Admins always pass every access check regardless of default_access.
    admins: []

    # Default access level for chats without a per-chat override.
    # Values: "all" | "admins" | "group:<name>"
    default_access: "all"

    # Default session isolation for group chats without a per-chat override.
    # Values: "individual" | "shared" | "admin"
    default_isolation: "individual"

    # Named user groups for group-level access control.
    user_groups:
      - name: "devs"
        user_ids: [111222333, 444555666]

    # Per-chat overrides (optional). chat_id is negative for groups/supergroups.
    chats:
      - chat_id: -1001234567890
        isolation: "individual"
        access: "all"
      - chat_id: -1009876543210
        isolation: "admin"
        access: "admins"
```

### 代理

`proxy` 决定机器人如何访问 Bot API，其读取方式类似提供商的 `proxy`（[提供商代理](../getting-started/configuration.md#提供商代理)）：

- 无值，或 `inherit`（默认）：Coddy 进程环境指定的代理 - `HTTPS_PROXY` 用于 `https://api.telegram.org`，`NO_PROXY` 用于直连的主机，绝不用于回环地址；不读取 `ALL_PROXY`。空字段从不表示直连：在设置了 `HTTPS_PROXY` 的机器上，机器人会走该代理；
- `none`：直连，机器人忽略这些变量；
- 代理 URL：每个 Bot API 请求都通过它。

```yaml
gateways:
  telegram:
    proxy: none                       # the machine's proxy cannot reach Telegram
    # proxy: "socks5h://127.0.0.1:1080"  # or "http://proxy.example.com:3128"
```

支持的方案：`http`、`https`、`socks5`、`socks5h`；使用任一 SOCKS 方案时，代理会解析主机名。在 Web 界面中，该字段是 **忽略系统代理** 开关，它会写入 `none`，位于 **代理 URL** 输入框上方（**设置 → 网关**，Telegram 块）。`coddy --dry-run` 会按机器人的方式，通过同一路由询问 `getMe`。

### 富消息

设置 `rich_messages: true` 以使用 [Bot API 10.1 富消息](https://core.telegram.org/bots/api#rich-messages)传输，而不是旧版 Telegram Markdown 子集：

```yaml
gateways:
  telegram:
    rich_messages: true
```

| 方面 | 旧版（默认） | `rich_messages: true` |
|--------|------------------|------------------------|
| 最终消息 | `mdToTelegram` 将标题/表格降级为纯文本 | 智能体的原生 Markdown 通过 `sendRichMessage` 原样发送 - 标题、表格、任务列表、围栏代码、脚注、LaTeX 都会渲染 |
| 流式传输（私聊） | 对实时消息进行渐进式 `editMessageText` | 临时 `sendRichMessageDraft` 预览（30 秒，带动画） |
| 工具活动 | `⚙️ toolname…` 行，从最终消息中删除 | 流式传输期间显示实时 `<tg-thinking>` 占位符，**并且** 最终消息中为每个已执行工具显示一个折叠的 `<details>` 块（名称 + 输出，失败时 `❌`） |
| 会话看到的内容 | 用户输入的内容 | 用户输入的内容 |
| 系统提示词块 | 旧版子集，为该轮次明确写出 | 完整 GFM 渲染；在手机上保持简短 |

**行为说明：**

- **群聊**不会获得草稿流式传输（`sendRichMessageDraft` 仅限私聊）；机器人在轮次结束后发送最终的 `sendRichMessage`，工作时显示正在输入指示器。
- **草稿是临时的** - 它们约 30 秒后过期且永远不会持久化；轮次由单独的 `sendRichMessage` 最终确定。Bot API 10.1 中没有 `editRichMessage`。
- **优雅回退** - 如果富发送失败（例如 Bot API 服务器不支持 10.1），网关会自动回退到旧版格式化发送，因此机器人绝不会沉默。
- 需要实现 Bot API 10.1 的 Bot API 服务器。

### 访问级别

| 值 | 谁可以交互 |
|-------|-----------------|
| `all` | 任何能向聊天写入的人 |
| `admins` | 仅 `admins` 中列出的用户 ID |
| `group:<name>` | 指定 `user_groups` 条目的成员（管理员始终通过） |

每条传入消息都会检查访问权限。被拒绝的消息会被静默丢弃。

### 会话隔离模式

仅适用于**群组和超级群组聊天**。无论此设置如何，私聊始终按用户隔离。

| 模式 | 会话范围 |
|------|--------------|
| `individual` | 每个群成员获得自己的私聊会话 |
| `shared` | 群组所有成员共享一个会话 |
| `admin` | 仅管理员用户可交互；所有管理员共享一个会话 |

示例：团队聊天中的共享 DevOps 机器人使用 `shared`。添加到群组中的个人助理使用 `individual`。

### 按聊天覆盖

使用 `chats` 为特定聊天覆盖 `isolation` 和 `access`：

```yaml
gateways:
  telegram:
    default_access: "all"
    default_isolation: "individual"
    chats:
      - chat_id: -1001111111111   # team group: shared session, all members
        isolation: "shared"
        access: "all"
      - chat_id: -1002222222222   # private project: admins only
        isolation: "admin"
        access: "admins"
```

### 用户组

定义 Telegram 用户 ID 的命名组，并在 `access` 中引用它们：

```yaml
gateways:
  telegram:
    user_groups:
      - name: "devs"
        user_ids: [111, 222, 333]
      - name: "qa"
        user_ids: [444, 555]
    chats:
      - chat_id: -1001234567890
        access: "group:devs"    # only devs (+ admins) can use the bot here
```

---

## 运行网关

网关不是独立命令。`coddy serve` 会运行配置启用的每个子系统，机器人就是其中之一：

```yaml
gateways:
  telegram:
    enable: true
```

```bash
coddy serve [flags]
```

| 标志 | 默认值 | 描述 |
|------|---------|-------------|
| `--config` | `$CODDY_HOME/config.yaml` | 配置文件路径 |
| `--home` | `~/.coddy` | 智能体状态目录（`CODDY_HOME`） |
| `--cwd` | 进程 cwd | 默认会话工作目录 |
| `--sessions-dir` | `$CODDY_HOME/sessions` | 会话捆绑包存储位置 |
| `--log-level` | 来自配置 | `debug\|info\|warn\|error`，或带按组件覆盖的逗号分隔规范，例如 `info,gateway.telegram=debug`（见[调试聊天](#调试聊天)） |
| `--gateway` | 来自配置 | 本次运行覆盖 `gateways.telegram.enable` |
| `--http=false` | — | 运行机器人而不提供 HTTP API |

典型生产调用：

```bash
coddy serve \
  --config /etc/coddy/config.yaml \
  --home /var/lib/coddy \
  --sessions-dir /var/lib/coddy/sessions
```

该进程还会在 `127.0.0.1:12345` 上提供 Web 界面，因为 `httpserver.enable` 默认为 true。仅机器人是 `--http=false`，或文件中的 `httpserver.enable: false`。

如果二进制在构建时没有网关标签，启用机器人会导致启动错误并指出该标签，而不是一个没人阅读的警告。

进程会阻塞，直到 `SIGINT` 或 `SIGTERM`。每个适配器在自己的 goroutine 中运行，出错时自动重启（5 秒退避）。关闭时，轮询器先停止接收新消息，已经在生成的轮次最多有 20 秒完成，因此回答不会在半句话时被切断。

**使用 Docker Compose** - 仓库的 compose 文件运行 `coddy serve`，因此启用机器人只是挂载的 `config.yaml` 中的一行，而不是第二个容器：

```bash
export TELEGRAM_BOT_TOKEN="<bot-token>"          # or leave it in $CODDY_HOME/.env
docker compose -f docker-compose.dev.yml up -d --build
docker compose -f docker-compose.dev.yml logs -f coddy   # expect: "telegram bot connected"
```

仍然可以将机器人作为独立于 Web 界面的服务运行 - 两个
`coddy serve` 进程，一个带 `--http=false`，一个带 `--gateway=false` -
但它们会再次拥有独立的会话管理器，聊天对话
也不再在浏览器中实时可见。除非需要隔离，否则优先使用一个进程。


---

## 调试聊天

一个似乎什么都没做的命令 - 点击 `/model` 后模型没有变化，机器人从不回答的消息 - 在 `info` 不会留下任何痕迹。该级别记录成功的内容（连接、会话加载和清除、应用了模型或模式）以及失败到足以发出警告的内容；被静默丢弃的更新，或从未到达的点击，都不在两个列表中。在 `debug` 使用 `gateway.telegram` 则会记录整个路径：每条更新到达时（`text_len` 统计用户写入的内容，包括照片或文件的图注，`media` 指出随附内容），某条更新为何被丢弃（访问被拒绝、仅管理员聊天、群消息未发给机器人、工作队列已满、未知命令、空消息）或因在聊天中回答而被拒绝（机器人不接收的媒体、超过下载限制的文件、不读取图像的模型），每个识别的命令，机器人构建的每个菜单及其所属会话，以及每个回调解析到的模型或模式以及是否已应用。

只提高该组件的日志级别，保持进程其余部分不变：

```yaml
logger:
  level: "info"
  levels:
    - component: "gateway.telegram"
      level: "debug"
```

在 systemd 下进行单次重启时，该标志携带相同规范，无需编辑配置文件：

```bash
coddy serve --log-level "info,gateway.telegram=debug"
```

每条记录都保留其 `component` 属性，因此混合子系统的文件仍可过滤：

```bash
grep '"component":"gateway.telegram"' /var/log/coddy/coddy.log
```

成功生效的切换会在 `info` 报告，因此确认信息会出现在日志中而无需提高日志级别：`telegram: model applied` 和 `telegram: mode applied` 指出会话和新值，`telegram: session resumed` 指出聊天离开的会话和移动到的会话。到达机器人但失败的点击会在 `warn` 记录原因，同样可见：会话无法加载时 `telegram: callback session`，按钮指定的模型不再配置时 `telegram: callback model unknown`，继续按钮指定的会话在发送键盘后已被删除时 `telegram: callback session unknown`，所选捆绑包无法加载时 `telegram: resume session`，管理器拒绝更改时 `telegram: set model`，以及用户发送的照片或文件无法从 Telegram 获取时 `telegram: file download failed`（记录指出文件，绝不指出其地址，因为地址携带机器人 token）。在 `debug` 级别，`telegram: resume menu` 记录选择器的每一页及聊天当前会话，`telegram: resume query` 记录 `/resume` 后的词语以及匹配的会话数。在 `warn` 没有日志且 `debug` 也没有任何内容，意味着更新从未到达 - 检查机器人 token、ACL，以及是否有另一个进程在轮询同一机器人，因为 Telegram 每条更新只投递给一个长轮询。

### 针对假 Bot API 调试

日志告诉你机器人对更新做了什么；它不能让你在没有手机、token 和回答背后的模型的情况下发送更新。[tgfake](https://github.com/EvilFreelancer/tgfake) 可以做到，这是本仓库用于测试其机器人的离线 Telegram 测试台，也是一个独立项目：一个替身 Bot API 服务器，会响应网关调用的每个方法（`getMe`、带真实长轮询的 `getUpdates`、`sendMessage`、`editMessageText`、`answerCallbackQuery`、Bot API 10.1 的 `sendRichMessage` 和 `sendRichMessageDraft` 等），保留收到的聊天，并提供一个页面，让你成为聊天中的人 - 机器人的内联键盘就是按钮。使用 `--llm` 时，它还会提供一个脚本化模型，因此整个测试台完全无需网络即可运行。获取发布二进制（[安装脚本](https://github.com/EvilFreelancer/tgfake#install)或发布页面），或从此检出的根目录运行其固定版本，使用 `go tool tgfake`：

```bash
tgfake --llm --llm-strip-tag turn_context --llm-delay 50ms           # a release binary: Bot API + model on 127.0.0.1:18790
go tool tgfake --llm --llm-strip-tag turn_context --llm-delay 50ms   # the same, the version go.mod pins
```

`--llm-strip-tag turn_context` 告诉模型跳过 Coddy 附加到每个请求的 `<turn_context>` 块，因此它会回答用户写入的内容。将 `coddy serve` 指向测试台，使用 **`CODDY_TELEGRAM_API_BASE`**，`--dry-run` 探测也会遵循该来源，并给它 stub 作为提供商：

```yaml
providers:
  - name: stub
    type: openai
    api_base: "http://127.0.0.1:18790/v1"
    api_key: "sk-tgfake"            # any key; the stub checks none
models:
  - model: stub/tgfake-demo
agent:
  model: stub/tgfake-demo
httpserver:
  enable: false                     # the stand is the bot alone; drop this to watch the chat in the web UI too
gateways:
  telegram:
    enable: true
    token: "123456:fake"            # any token; the fake accepts all of them
    admins: [4242]                  # the page's user: /resume, approvals and settings are an admin's
logger:
  levels:
    - component: gateway.telegram
      level: debug
```

```bash
export CODDY_TELEGRAM_API_BASE=http://127.0.0.1:18790   # PowerShell: $env:CODDY_TELEGRAM_API_BASE="http://127.0.0.1:18790"
coddy serve --dry-run --config stand.yaml               # ok  gateways.telegram: token accepted by the Bot API, bot @tgfake_bot
coddy serve --gateway --http=false --config stand.yaml  # telegram: api base override ... telegram bot connected
```

然后打开 `http://127.0.0.1:18790/`，输入 `hello`，发送 `/model` 并点击一个模型（向 `models` 添加第二个条目，让键盘提供选择），尝试一个设置命令，例如 `/plan --once What can you do?`，并在日志从左侧填充时阅读右侧的 Bot API 调用。

![tgfake 的深色模式聊天页面：左侧是用户一侧的聊天，机器人的 /model 键盘显示为按钮，一条 /plan --once 消息得到回答，机器人发出的每个 Bot API 调用列在右侧](../../assets/tgfake-chat-dark-1280.png)

*tgfake 的聊天页面：脚本化模型回答问候，`/model` 键盘应用了点击，在计划模式下发送了一条仅一轮的消息，一条裸 `/ask` 的通知，右侧是机器人发出的每个 Bot API 调用，`getUpdates` 轮询被隐藏。*

同一页面也是 HTTP API，脚本或编码智能体可以驱动它（完整参考是 tgfake 的[模拟 API](https://github.com/EvilFreelancer/tgfake/blob/main/docs/sim-api.md)）：

| 路由 | 主体 / 回答 |
|-------|---------------|
| `POST /sim/message` | `{"chat_id": 4242, "user_id": 4242, "username": "alice", "text": "hello"}`；群组路径使用 `chat_type: group`、`mention: true` 和 `reply_to_message_id`。开头的 `/word` 会变成 `bot_command` 实体。 |
| `POST /sim/callback` | `{"chat_id": 4242, "label": "stub/coddy-mini"}` 按文本点击按钮（忽略 `✓` 前缀），或 `{"message_id": 4, "data": "model:stub/coddy-mini"}`。 |
| `GET /sim/chat/4242` | 对话记录：消息、每次编辑后的键盘、草稿、`typing`；`grep` 使用 `?format=text`。 |
| `GET /sim/outbox?method=sendMessage&since=10` | 每个 Bot API 调用及其参数和回答；脚本使用 `/sim/outbox/count?method=...`。 |
| `POST /sim/fault` | `{"method": "sendMessage", "code": 429, "retry_after": 2, "times": 1}` 使下一个 `sendMessage` 像洪泛一样失败；`"method": "*"` 使一切失败直到 `DELETE /sim/fault`；`"contains": "<details>"` 将故障缩小到参数包含该文本的调用，即 Telegram 拒绝一个实体而不是该方法。 |
| `POST /sim/webapp/launch` | `{"chat_id": 4242, "url": "https://coddy.example.com/"}` 像客户端一样打开 Mini App 并回答 `{url, init_data}`：地址的片段中包含启动参数，启动数据用机器人 token 签名。没有 `url` 时，它打开聊天的菜单按钮，只有私聊才有该按钮。 |
| `POST /sim/reset` | 忘记聊天、发件箱、故障和菜单按钮。更新 ID 继续增长，因此轮询机器人不会混乱。 |

假服务器在 Telegram 严格的地方也严格。不改变任何内容的编辑、编辑从未发送的消息、超过 4096 字符的文本、回复聊天中不存在的消息（除非 `allow_sending_without_reply` 表示仍要发送）、回答假服务器从未发出的回调查询、对已发出查询的第二次回答（一个查询只接受一个回答，因此在点击被确认后通过警报告知的失败永远不会到达用户）、`inline_keyboard` 不是数组的 `reply_markup`（包括 `null`，Go 库的空键盘会编码成它）、`callback_data` 超过 64 字节的键盘（`BUTTON_DATA_INVALID`）、没有动作的按钮，以及私聊之外的 `web_app` 按钮或指向既不是 https 也不是本机纯 http 的地址的按钮，都会以 Telegram 自己的错误被拒绝。测试台在一处更严格：它拒绝带多个动作的按钮，而 Telegram 会将其读作第一个动作。能在测试台上工作的键盘就能在聊天中工作。

它还会像 Telegram 一样记住 `allowed_updates`。一个曾经在另一个框架下运行的机器人 token 可能只订阅了消息，而未指定类型的轮询会继承这一点：文本会到达，键盘点击会在任何人看到之前被丢弃。因此网关每次轮询都请求 `message` 和 `callback_query`。`GET /sim/state` 显示生效的订阅；tgfake 的 `pkg/server` 的 `Options.AllowedUpdates` 在陈旧订阅下启动机器人，轮询功能就是这样重现该情况。在真实机器人上，`getWebhookInfo` 报告相同字段。订阅在创建更新时应用：更改它会保留已排队的更新，但无法恢复创建时被排除的事件。

该页面也是 [Mini Apps](#mini-app) 的 Telegram 客户端。点击 `web_app` 按钮，或机器人设为 Mini App 的菜单按钮（消息字段左侧），会在聊天旁边的手机框架中打开应用，而该页面就是它的 Telegram，就像 web.telegram.org 一样：它用机器人 token 签名启动数据（最近一次 Bot API 调用的 token，或 `--token`），回答视口、安全区域和主题请求，在应用请求时显示 **Back**，用应用设置的颜色绘制其头部，并在手机下方记录双向的每个事件。手机上方的控件可切换尺寸、展开和半开之间的窗口（框架会像手机一样裁剪全高 WebView）、Telegram 主题、全屏内边距和 Android 键盘。要在那里打开 Web 界面，让测试台的 `coddy serve` 提供它并公布它：

```yaml
httpserver:
  allow_insecure: true                 # no sign-in on the stand, advertised on purpose
gateways:
  telegram:
    mini_app:
      url: "http://127.0.0.1:12345/"   # plain http passes on a loopback host
```

然后发送 `/app` 并点击 **Open in Coddy**。同一主机上的页面和 Web 界面对 cookie 而言是一个站点，因此框架会像手机的 WebView 一样保持登录；`localhost` 上的页面与 `127.0.0.1` 上的 Web 界面是跨站框架，就像 Telegram Web 一样。`npm run check:telegram` 从脚本驱动同一测试台（[Web 界面页面](web-ui.md)上的 *Checking the Telegram Mini App*）。

![tgfake 的聊天页面，Mini App 已打开：左侧是聊天和机器人的回答，中间是半开手机框架中的 Web 界面，模式表覆盖其停靠的输入框，上方是带 Back 和 Close 的 Telegram 头部，右侧是 Bot API 调用](../../assets/tgfake-mini-app-dark-1280.png)

*tgfake 的聊天页面，Web 界面作为机器人的 Mini App 半开：模式表和输入框停留在手机显示的部分，聊天在其上方。*

富消息预览会在最后一次成功修订后 30 秒过期。聊天页面和 `/sim/chat/{id}` 停止显示过期草稿；读取聊天或写入另一个草稿也会从其存储中移除过期条目。发件箱保留调用以供调试，持久消息会保留。

`--llm-answer`（可重复）按轮次脚本化模型的回复，`--llm-script
rules.json` matches them by substring (`[{"match": "weather", "answer":
"Sunny."}]`），如果两者都没有，模型会回显提示词 - 用户的消息，而不是 `<turn_context>` 块，只要给出 `--llm-strip-tag
turn_context`。规则是可靠选择：会话从第一条消息派生的标题是另一次模型调用，因此答案列表会比聊天显示提前一步。规则也可以让模型行动：使用 `"tool": {"name": "run_command",
"arguments": {...}}` 时，匹配轮次的第一个请求会获得该工具调用，携带其结果的请求会获得规则的 `answer` - 足以启动后台命令，或无需模型即可满足权限提示。流式回答每个 `--llm-delay` 到达一个词，足够长，可让实时 `editMessageText` 路径或带 `rich_messages: true` 的草稿路径运行。

**`examples/gateway/tg_e2e_offline.sh`** 一次完成上述所有操作 - 启动 tgfake（`CODDY_TGFAKE_BIN` 中的二进制，否则构建 `go.mod` 固定的版本），写入临时 home，针对它启动 `coddy serve`，发送 `hello` 并检查回复，然后用 `/clear` 离开会话，从 `/resume` 键盘返回并检查下一条消息落在该捆绑包中，最后要求智能体 "启动测试" - 工具规则用 `notify_on_finish` 在后台启动一个失败命令 - 并等待[被唤醒的轮次](#被唤醒的轮次会进入聊天)到达聊天，先出现提示再出现回答，进程中没有任何 HTTP 服务器 - `TG_E2E_KEEP=1` 让测试台保持运行并打印页面 URL。它也可以在 Windows 上的 Git Bash 中运行。

该变量不仅用于假服务器：自托管 Bot API 服务器（`telegram-bot-api` 用于大文件或本地网络）也以同样方式指向，`coddy serve --dry-run` 会在机器人启动前确认哪个服务器回答。

---

## 机器人交互模型

### 私聊

每个与机器人开始私聊的用户都会获得自己的隔离会话。无需配置。每条消息都是给机器人的，无论是否提及。

### 回复

回复某条消息的人是在询问该消息。智能体会在用户写入的内容前收到被回复消息的引用，其第一行指出作者：

```
> Anna:
> the build is red since noon

why?
```

这在私聊和群聊中都有效，无论回复的是机器人自己的回答还是别人的消息（在群组中，同时提及机器人）。仅回复下的提及会让智能体处理被引用的消息。引用是用户消息的一部分，因此 Web 界面中的对话记录读起来与聊天一致；设置命令永远不会被引用。模型每轮都会被告知该约定（[即时通讯需要什么](#即时通讯需要什么以及在哪里说明)）。

### 群聊

群组是许多人交谈的地方，因此机器人**仅在**被明确寻址时**响应**：

1. **@提及**机器人的消息（`@coddy_agent_bot hello`）
2. 对先前机器人消息的**直接回复**

命令遵循相同规则：在群组中，机器人命令或设置命令需要提及，Telegram 将其写作 `/clear@coddy_agent_bot`（群组的命令菜单会插入该形式），或回复机器人的消息。群组中的裸 `/clear` 留给其中的人。

### 非管理员可以做什么

机器人会看到每条消息的作者，其管理员（`gateways.telegram.admins`）可以做任何事。访问规则允许的其他任何人只能获得对话，不能获得任何改变智能体的内容：

- 该轮次只能调用读取工具（`read`、`grep`、`glob`、`print_tree`），且限制在会话工作目录内、智能体 home 之外、内置文档、Web 搜索、会话计划和待办列表以及反问；其他任何内容 - 配置、`switch_model`、子智能体、worktree、调度器、后台任务、服务器、shell 命令、写入、获取页面、每个 MCP 工具 - 都会被拒绝，并告知模型原因；
- `@` 提及仅附加工作目录中的文件，不附加网页；
- `/plugin` 只回答其用法屏幕，`/plugin list`：安装、更新或移除技能会改变智能体 home 的每个会话所运行的内容；
- 每个需要批准的操作都会询问机器人，无论会话的权限模式如何，机器人都会拒绝；管理员在共享会话中授予的 "始终允许" 不适用；
- 权限按钮仅由管理员回答，在会话共享的群组中也是如此；
- 被唤醒的轮次和后台子智能体的请求采用对话所属者的权限：私聊或独立群会话的人，绝不是共享群组的人。

模型看到的工具定义保持不变，因此提供商的提示词缓存在同一共享会话中的管理员和其他所有人之间保持有效。因此，`admins` 列表为空的机器人只允许所有人聊天；`coddy serve --dry-run` 会发出警告。

在群组中，更改设置的是管理员：设置命令（`/agent`、`/plan`、`/ask`、`/think`、`/nothink`、`/reasoning`、`/model <id>`）、`/model` 菜单及其点击、`/clear` 和 `/resume`。其他任何人会收到 *Only the bot's admins can change settings in this chat.*，且不会发生任何变化；在私聊中，每个人仍然更改自己的会话。`/mcp` 开关会更改整个智能体的配置，因此在每个聊天中点击它们都是管理员的事，`/resume` 也是如此，它会到达服务器保留的每个会话，包括操作员自己的会话。

当 `isolation` 为 `admin` 时，机器人还会忽略所有不在 `admins` 列表中的人。

### 命令

| 命令 | 可用对象 | 效果 |
|---------|-------------|--------|
| `/start` | 所有用户 | 问候和快速介绍。 |
| `/help` | 所有用户 | 列出所有可用命令。 |
| `/model [id]` | 所有被允许的用户（群组中仅管理员） | 单独使用时打开内联键盘以切换活动 LLM 模型（来自配置的 `models` 列表）；带 id 时立即切换到该模型。管理员的选择也是机器人下一个全新聊天启动时使用的模型；其他人的选择保留在自己的会话中。 |
| `/mcp` | 所有被允许的用户（其开关：管理员） | 列出全局和项目 MCP 服务器及其状态和工具数量；当服务器关闭而其状态是信任判定时，也会显示 `off`（`checkout · needs_approval · off · 0 tools`）。按钮启用或禁用已为工作区信任的服务器；无人批准的项目服务器没有按钮，因为项目信任通过 CLI、控制台或 Web 界面授予。失败的点击会在菜单消息第一行说明原因，位于重新绘制的菜单上方；如果无法读取服务器列表，则单独显示且没有按钮。 |
| `/agent`、`/plan`、`/ask` | 所有被允许的用户（群组中仅管理员） | 切换会话模式。 |
| `/reasoning <level>`、`/think [level]`、`/nothink` | 所有被允许的用户（群组中仅管理员） | 设置推理级别，或在模型提供商支持时打开或关闭思考。不在命令菜单中。 |
| `/context` | 所有被允许的用户 | 显示当前会话上下文窗口使用情况，按类别分解（对话、系统提示词、工具定义、规则、技能、MCP）。 |
| `/goal <objective>`、`/goal`、`/goal pause`、`/goal resume`、`/goal clear` | 管理员（`admins`）更改，所有被允许的用户显示 | 设置会话目标并开始处理、显示、暂停、恢复或移除它（[会话目标与监督器](../features/session-supervisor.md)）。监督器启动的每一轮都会在上一轮回答后用单独消息宣布。 |
| `/resume [id or title]` | 管理员（`admins`） | 继续另一个会话。单独使用时打开内联键盘，列出服务器保留的会话，最新在前，每页八个，当前聊天自己的会话有标记；点击将聊天绑定到所选会话。后面带词时，立即恢复其 id 为这些词、id 以这些词开头或标题包含这些词（后两者不区分大小写）的会话；多个匹配会以键盘返回，无匹配则用消息回答。回复会指出恢复的会话运行的模型和推理级别，即其自己的设置。离开的会话保持加载。 |
| `/app` | 管理员（`admins`） | 仅在设置了 `gateways.telegram.mini_app.url` 时：以 **Open in Coddy** 回答，该按钮在 Web 界面中打开聊天对话 - 私聊中作为 Mini App，群组中作为链接（[Mini App](#mini-app)）。 |
| `/clear` | 所有被允许的用户（群组中仅管理员） | 为当前用户/聊天上下文启动新会话。旧会话从内存中移除（持久化历史仍保存在磁盘上）；`/resume` 将其恢复。 |

当监督器在聊天中启动延续时，机器人会在下一个回答前单独发布一条 `🔁` 说明，包含剩余工作。同一提示词会保存为单独的对话记录行，因此另一个界面上的 `/resume` 也能看到它。

设置命令接受 `--once` 或 `--count=N` 以仅对后续消息更改设置，并且它们后面可以跟一条消息：`/plan --once how would you split this package?` 计划一个回答并让聊天保持其模式。单独的命令不运行轮次，并会用一行说明更改了什么。`/permissions` 不是机器人命令：机器人自己批准其聊天智能体的工具，因此会话的权限模式只会改变其他监视该会话的界面所询问的内容（[会话设置](../features/session-settings.md)）。

---

## 即时通讯需要什么，以及在哪里说明

即时通讯有自己的方言和自己的屏幕形态。Coddy 在网关所属的两个地方说明这两者，二者都不会触及会话保留的对话。Telegram 机器人还会将用户客户端报告的语言（`language_code`）作为轮次语言传递，因此该轮次的 `@coddy:` 提及附加且智能体读取的文档，在 Coddy 文档包含该语言时就是该语言（[内置文档](../features/built-in-docs.md#语言)）。

### 针对该轮次告知模型

在聊天轮次运行之前，适配器会向会话传递一段**系统提示词**，描述如何向此聊天回答：Telegram 渲染的强调、它不渲染的标题、表格会变成一堵竖线墙、标识符应放在反引号中，因为裸 `_` 会开启斜体，以及聊天在手机上是窄列。它位于 `external/gateway/telegram/prompt.go`，`rich_messages: true` 会发送不同的内容 - 在那里聊天会完整渲染 GitHub 风格 Markdown，因此唯一值得说的是手机屏幕需要多少内容。

该块属于**轮次**，不属于会话（`session.PromptRunOpts.SurfaceSystemPrompt`）。其中没有任何内容被持久化，因此对话记录保存的是对话，而不是运行它的界面，同一会话上的浏览器轮次构建时不包含它。这确实意味着不同界面的提示词前缀不同，因此从其他地方跟随而来的轮次不会重用其缓存前缀。这是让每个集成自己说话、而不是让核心了解即时通讯的刻意代价。

### 回答在发出时会被渲染

模型并不总是遵守，而显示原始 `##` 的聊天比网关悄悄修复过的回答更糟，因此回复在离开时也会在 `external/gateway/telegram/markdown.go` 中转换。

对于旧版发送：ATX 标题和 `**bold**` 变成 `*bold*`，`__x__` 变成 `_x_`，星号项目符号变成 `•`，表格被展平为普通行，水平线变为分隔线。围栏块和内联代码段在任何规则运行前被放到一边，并原样放回，因此块内的 Go `**p` 或 `# comment` 会按模型写入的样子到达聊天。实时流式预览发送时不使用解析模式 - 半句话就是半个标记 - 因此它获得相同转换，但强调标记被丢弃而不是显示为标点。如果 Telegram 仍然拒绝解析消息，发送器会在没有解析模式的情况下重发：正文中的多余星号只会损失格式，绝不会损失回复。

使用 `rich_messages: true` 时没有需要降级的内容 - 智能体的 Markdown 原样发出 - 回退路径是上面的旧版渲染。

### 添加集成

两部分都由新适配器编写：一个 `prompt.go` 说明其即时通讯需要什么，以及一个用于其语法的渲染器。`internal/` 中没有任何内容会了解它。

---

## 编写新适配器

例如，要在 Telegram 旁添加 Discord 适配器，请遵循此模式。

### 1. 实现 Adapter 接口

创建 `external/gateway/discord/bot.go`：

```go
//go:build gateway || gateway.discord

package discord

import (
    "context"
    "fmt"

    "github.com/EvilFreelancer/coddy-agent/external/gateway"
    "github.com/EvilFreelancer/coddy-agent/external/gateway/access"
    "github.com/EvilFreelancer/coddy-agent/external/gateway/sessionstore"
    "github.com/EvilFreelancer/coddy-agent/internal/config"
)

type Bot struct {
    cfg   *config.DiscordGatewayConfig  // add to config.GatewayConfig
    store *sessionstore.Store
    // ... discord client, session runner, logger
}

func New(cfg *config.DiscordGatewayConfig, runner SessionRunner, cwd string, log *slog.Logger) *Bot {
    return &Bot{cfg: cfg, store: sessionstore.New()}
}

// Name satisfies gateway.Adapter.
func (b *Bot) Name() string { return "discord" }

// Start connects and polls. Must block until ctx is cancelled.
func (b *Bot) Start(ctx context.Context) error {
    // connect discord client
    // poll or use websocket events
    // for each message: call b.handleMessage(ctx, msg)
    return nil
}
```

`SessionRunner` 接口（`external/gateway/telegram/bot.go`）是你从会话管理器需要的内容：

```go
type SessionRunner interface {
    EnsureHTTPSession(ctx context.Context, sessionID string, defaultCWD string) (*session.State, error)
    HandleSessionPromptWithSender(ctx context.Context, params acp.SessionPromptParams, sender acp.UpdateSender, opts *session.PromptRunOpts) (*acp.SessionPromptResult, error)
    ForgetLiveSession(sessionID string)
    HandleSessionSetConfigOption(ctx context.Context, params acp.SessionSetConfigOptionParams) (*acp.SessionSetConfigOptionResult, error)
    HandleSessionList(ctx context.Context, params acp.SessionListParams) (*acp.SessionListResult, error)
    Cfg() *config.Config
}
```

`session.Manager` 已经满足此接口，直接传递即可。`HandleSessionSetConfigOption` 是 `/model` 内联键盘调用的内容，设置命令（`/agent`、`/model <id> --once` 等）通过 `HandleSessionPromptWithSender` 以提示词文本到达会话，其管理器会从消息开头取下它们；`HandleSessionList` 是 `/resume` 向聊天提供的内容；`Cfg()` 返回已加载的配置（由 `/model` 用于列出可用模型）。

### 2. 添加 serve_name.go 及其桩

每个适配器由 `external/gateway` 中自己的函数启动，位于适配器标签下的文件中，并为不带该标签的构建提供桩。Telegram 对是 `serve_telegram.go` / `serve_telegram_stub.go`，Pachca 对是 `serve_pachca.go` / `serve_pachca_stub.go`：

```go
//go:build gateway || gateway.discord

package gateway

// DiscordAvailable reports whether this binary carries the Discord adapter.
const DiscordAvailable = true

// ServeDiscord builds the Discord bot and runs it until ctx is cancelled.
func ServeDiscord(ctx context.Context, opts Options) error {
    bot := discord.New(&opts.Cfg.Gateways.Discord, opts.Mgr, opts.DefaultCWD,
        logger.Component(opts.Log, logger.ComponentGatewayDiscord), storePath, opts.Mirror)
    NewHub(logger.Component(opts.Log, logger.ComponentGateway), bot).Start(ctx)
    return nil
}
```

桩（`//go:build !(gateway || gateway.discord)`）设置 `DiscordAvailable = false` 并返回指出标签的错误。给适配器自己的会话映射文件：两个 `sessionstore.Store` 实例重写一个文件会互相撤销写入。

### 3. 实现 acp.UpdateSender

每个消息分发都需要一个实现三个方法的 `Sender`：

```go
type UpdateSender interface {
    SendSessionUpdate(sessionID string, update interface{}) error
    RequestPermission(ctx context.Context, params acp.PermissionRequestParams) (*acp.PermissionResult, error)
    RequestQuestion(ctx context.Context, params acp.QuestionRequestParams) (*acp.QuestionResult, error)
}
```

- `SendSessionUpdate` 接收流式事件：`acp.MessageChunkUpdate` 在 `update.Content.Text` 中携带文本增量；`acp.ToolCallUpdate` 是工具启动通知。缓冲文本块，并在智能体轮次后的 `Flush()` 中作为单条消息发送。
- `RequestPermission` 应自动批准聊天智能体自己的请求（管理员是特意配置机器人的）：返回 `&acp.PermissionResult{Outcome: "allow", OptionID: "allow"}`。强制提示（`params.Forced`：钩子的 `ask`、决策检查未清除的命令）以及带有子智能体自己模式的请求（低于 `bypass`，`params.EffectivePermissionMode`）不是管理员可以放行的：`permission.ChatApproves` 会将它们区分开；像 `external/gateway/telegram/permission.go` 一样询问聊天，或拒绝它。
- `RequestQuestion` 可以将问题文本发送到聊天并返回空回答，或实现适当的基于回复的流程。

参见 `external/gateway/telegram/sender.go` 获取工作参考。

### 4. 添加构建标签

遵循现有模式：

- `external/gateway/discord/*.go` → `//go:build gateway || gateway.discord`
- `external/gateway/discord/*_test.go` → 相同约束
- 桩（如需要）→ `//go:build !(gateway || gateway.discord)`

将标签添加到共享文件的约束中 - `gateway.go`、`hub.go`、`access`、`sessionstore`、`proxyutil` - 以及 `Makefile` 中的 `TEST_TAG_SETS`。

### 5. 在 coddy serve 中注册子系统

在 `internal/serve/subsystem.go` 中为机器人添加 `serve.Kind`，并在 `cmd/coddy/serve.go` 的 `subsystems()` 中添加描述符：其 `ConfigKey`、`BuildTag`、`Available: gateway.DiscordAvailable`、`NeedsSessions: true`、针对重建机器人会以不同方式读取的每个设置的 `Fingerprint`，以及调用 `gateway.ServeDiscord` 的 `Run`。作为独立子系统的机器人会在其设置更改时单独重建，并且在没有其标签的二进制中启用时会按名称被拒绝。向横幅（`printServeBanner`）添加一行，并向 `internal/dryrun` 添加 token 探测。

---

## 聊天和浏览器中的同一会话

同时开启 `httpserver.enable` 和 `gateways.telegram.enable` 时，Telegram 对话和 Web 界面是同一会话的两个视图。

- **聊天会话会出现在浏览器中。** 聊天对话是带有普通 `sess_` id 的普通会话 - 人坐在哪里并不决定对话背后的会话 - 因此 `GET /coddy/sessions` 会将其列在终端或浏览器标签页中启动的会话旁边，打开它会加载相同的对话记录。
- **聊天轮次运行时会流式传输到浏览器。** 网关将其轮次发布到会话的输入框中继 - HTTP 服务器自己的轮次和被唤醒轮次使用的同一中继 - 因此监视该会话的标签页会看到 token 到达，而不是事后才看到。
- **浏览器观看；聊天回答。** 会话更新会分发到两个界面，但权限请求和问题只发送到聊天，因为只有聊天有人在阅读。监视者只是旁观者。唯一例外是轮次结束后才询问的后台子智能体：该提示会同时提供给聊天和浏览器，先回答者胜出。
- **继续在两个方向都有效。** 在浏览器中回复，Telegram 中的下一个 `/context` 会显示它；在 Telegram 中回复，浏览器下次加载时会有它。一次只运行一个轮次：会话的轮次锁是文件锁，因此浏览器轮次进行中到达的消息会用忙碌通知回答，而不是交错运行。
- **已经在被监视的轮次不会被接管。** 如果会话上有浏览器轮次正在运行，到达的聊天消息不会接管其流 - 聊天消息稍后仍会得到忙碌回答。

如果从浏览器删除会话，`gateway_sessions.json` 中的聊天映射仍指向该 id；下一条消息找不到捆绑包，并会在其下启动新的对话记录。对话会重置，这正是删除它的含义。

## Mini App

机器人可以将 Web 界面作为其 [Mini App](https://core.telegram.org/bots/webapps) 打开：点击消息字段旁的菜单按钮，或点击 `/app` 回答的按钮，会在 Telegram 内打开 Web 界面，并位于聊天自己的对话上。该对话就是浏览器显示的会话（[上文](#聊天和浏览器中的同一会话)），因此聊天和 Mini App 是它的两个视图。Web 界面在 Telegram 内做什么，在 [Web 界面页面](web-ui.md)的 *Telegram Mini App* 下描述。

Telegram 从用户手机打开 Mini App，且仅通过 https，因此 Web 界面必须在那里可访问：

1. 将 `coddy serve` 发布在 TLS 代理（Caddy、nginx、隧道）后面，使用诸如 `https://coddy.example.com/` 的公共地址，并将 `httpserver.host` 保持在其后的回环地址上。
2. 使用 `coddy serve set-password`，或 `CODDY_HTTP_USER` 和 `CODDY_HTTP_PASSWORD` 开启登录。菜单按钮会显示给每个与机器人打开私聊的人，无论 `default_access` 说什么，因此机器人不会宣传一个什么都不要求的 Web 界面：没有登录且没有 token 时，它不会动菜单按钮，`/app` 说明缺少什么，`coddy serve --dry-run` 发出警告。`httpserver.allow_insecure: true` 告诉它你是故意公开发布 Web 界面的。
3. 命名地址：

```yaml
gateways:
  telegram:
    mini_app:
      url: https://coddy.example.com/
      menu_button: true     # the default
```

只有机器人的管理员（`gateways.telegram.admins`）才能从聊天打开 Web 界面。Mini App 无需密码即可让管理员登录。Telegram 用机器人 token 签名启动数据；服务器检查签名，并且每个启动最多一小时且只接受一次。`${CODDY_HOME}/telegram-launches.json` 记录跨重启接受的启动。重用已接受的启动数据会返回 401。重放记录存储失败会返回 500，且不发放会话或 cookie。记录被替换之前的失败不会将启动标记为已使用，因此修复存储后用相同启动数据重试。如果替换后目录同步失败，启动可能已被标记为已使用，并在内存中继续受重放保护；如果重试返回 401，请重新打开 Mini App。成功登录会打开一个会话（`POST /coddy/auth/telegram`），它也能在仅 token 网关后工作，并在 token 更改或用户不再是管理员时结束。其他任何点击菜单按钮的人会在普通登录表单上方看到 *Only the bot's admins can open Coddy from Telegram.*（当服务器没有表单时则单独显示），`/app` 也会以同样方式回答他们。Web 界面是整个智能体；聊天是其他人得到的内容。

每次启动时，机器人随后会将其菜单按钮指向 `url`，标签为 **Coddy**，并将 `/app` 添加到其命令列表。`/app` 以 **Open in Coddy** 回答：在私聊中是一个打开聊天对话作为 Mini App 的按钮（`url?session=<id>`），在群组中是指向同一地址的普通链接，因为 Telegram 仅允许私聊中的 Mini App 按钮。在聊天有对话之前，按钮会打开开始屏幕。

机器人从 @BotFather 借用菜单按钮并归还它。第一次时，它会在 `gateway_sessions.json` 中保留它替换的按钮，当 Telegram 不说那是哪个按钮时，它会不动菜单按钮直到下次启动。当你设置 `menu_button: false`、清空 `url` 或 Web 界面停止要求登录时，它会放回该按钮，只要该按钮仍然打开机器人设置的地址。你自己的按钮如果打开与机器人相同地址也会回来，除非 Web 界面停止要求登录：那样聊天会显示命令。你在此期间在 @BotFather 中更改的按钮会保持你离开时的样子。机器人在启动时确定按钮，因此关闭机器人（`gateways.telegram.enable: false`）会让按钮继续打开 Web 界面：要收回它，设置 `menu_button: false` 并启动机器人一次。没有 `url` 时，机器人永远不会碰菜单按钮，因此你在 @BotFather 中手动接入的 Mini App 会继续工作：Web 界面会适应打开它的任何 Telegram。

`coddy serve --dry-run` 向 `url` 询问 Web 界面，并说明机器人何时会阻止它。Telegram Web（web.telegram.org）在另一个站点的框架中运行 Mini App，浏览器会丢弃登录 cookie；登录屏幕随后会说明这一点，并在自己的标签页中链接到 Coddy。要在没有 Telegram 和手机的情况下尝试所有这些，请从离线测试台打开 Mini App（[针对假 Bot API 调试](#针对假-bot-api-调试)）。

## 被唤醒的轮次会进入聊天

聊天中的智能体可以在后台启动长命令或子智能体并结束其轮次：当任务结束时，进程会唤醒智能体，除非调用设置了 `notify_on_finish: false`（[后台任务](../features/background-tasks.md#任务完成时唤醒智能体)）。被唤醒的轮次属于绑定到会话的聊天，因此机器人会在那里通过聊天自己的发送器运行它，完全像用户发送的消息一样：

- 聊天首先会收到自己的提示，位于回答上方：
  `🔔 Woken by a finished background task: bg_3 make test, failed, exit 2, 1m 30s`
  （多个任务一起结束时每个任务一行）；
- 然后回答像任何其他回答一样流式传输并最终确定，使用相同的界面提示词和相同的 Markdown 渲染；
- 监视会话的浏览器会通过输入框中继跟随它，聊天的权限规则适用：聊天智能体被允许它请求的内容，子智能体则在聊天中被询问。

这不需要 HTTP 服务器。在 `coddy serve` 下，进程拥有唤醒器，并先将每个被唤醒的轮次提供给拥有对话的界面 - 当聊天绑定到会话时就是机器人 - 然后才提供给 HTTP 服务器；`coddy serve --gateway --http=false` 仍会唤醒聊天。没有聊天绑定的会话（聊天用 `/clear` 或 `/resume` 移走）不属于机器人：它在 Web 界面的中继中运行，或完全没有界面而通过管理器运行，聊天听不到它的任何内容。机器人仅在连接时接收被唤醒的轮次；发现聊天自己的轮次仍在运行时会等待它，就像它在每个界面上所做的一样。

## 用户发送的照片和文件

用户可以私聊发送照片或文档给机器人，带或不带图注，或者在群组中发给机器人。当会话的模型读取图像（`models[].multimodal: true`）时，机器人会从 Telegram 下载文件并像 Web 界面传递附件一样交给会话：文件保存在会话资产中（只读），提示词指出其位置，图片随轮次展示给模型，图注就是用户询问的内容（[图像](../features/images.md)）。照片以最大尺寸到达，作为文件发送的图片仍然是图片。文本文件会为模型写出，最多前 64 KB；不是文本的文件、PDF 或归档会被指名，智能体用其工具从资产中打开它。图注的工作方式与文本相同：其中的命令就是命令（图注为 `/plan --once` 的照片会在计划模式下运行该轮次，图注为 `/clear` 的照片会启动新会话），提及机器人会在群组中寻址，作为回复发送的照片或文件会在图注前引用被回复的消息。

相册是一个轮次。机器人在最后一项后等待一秒以接收下一项，然后按发送顺序一起发送相册的每张图片，其图注也按相同顺序；在群组中，其中任何一项的提及都会寻址整个相册。相册之后立即发送的消息不会超过它：相册先走，然后才是消息。

机器人不接收的内容，会在聊天中说明，而不是静默丢弃：

- 视频、语音或视频消息、音频文件、动画、贴纸、位置、联系人或投票会用一条说明回答，无论带不带图注；作为文件发送的视频或声音会作为文件处理；
- 超过 20 MB 的文件，即机器人从 Telegram 下载的最大值，会在下载任何内容前被拒绝；
- 模型不读取图像的会话会被要求用 `/model` 选择一个支持图像的模型，且不会下载任何内容；
- 下载失败会在聊天中报告，以便用户重新发送文件。

下载通过 `gateways.telegram.proxy` 进行，像机器人的每个请求一样，从 Bot API 来源的文件地址发起；使用 `--local` 启动的自托管 Bot API 服务器会用自己的磁盘路径回答，因此通过它发送的文件会被报告为未下载。非管理员也可以发送文件；此类轮次的智能体不会读取工作目录之外的任何内容，因此它只看到展示给它的图片和文本文件，无法从资产中打开 PDF。

## 智能体查看过的图片

当聊天智能体读取图像文件且会话的模型读取图像（`models[].multimodal: true`）时，或调用返回图片的 MCP 工具时，机器人会在调用完成后立即将该图片作为照片发送到聊天中，并以名称作为图注（[图像](../features/images.md)）；一次调用的图片以最多十张的相册发送，最后一张相册后剩余的作为照片发送。MCP 工具属于机器人的管理员，因此只有管理员的轮次会调用。用户会在回答到来前看到智能体查看过的内容，在私聊和群组中都是如此。照片是 Coddy 与会话一起保留的副本，因此即使工作区文件此后已更改，它仍是展示给模型的图片。没有富消息时，回答增长所在的实时消息是在调用运行前发送的，因此一旦发布照片，机器人会丢弃它，回答会作为对问题的新回复出现在照片下方；在 Telegram 允许时会删除旧消息。

Telegram 拒绝作为照片的图片 - 边长过长、形状过窄、聊天不允许照片 - 会作为文档发送；速率限制、途中失败或关于聊天本身的拒绝不会作为文档重试。Telegram 拒绝作为照片的相册会逐张发送，因此每张仍可作为文档发送。只有聊天自己的会话会发送图片：子智能体读取的内容保留在其会话中，Web 界面会在那里显示。没有 `multimodal` 的模型会被拒绝 `read` 会展示给它的图片，因此不会发送任何内容，`read` 会报告原因；MCP 工具返回的图片仍会发送，因为工具无论如何都返回了它。

## 会话生命周期

```
User message arrives
        │
        ▼
sessionstore.SessionKey(gateway, chatID, userID, isolationMode, isGroup)
        │  returns a string key like "tg:chat:-100:user:42"
        ▼
store.Get(key)
        │  returns existing session ID (from gateway_sessions.json),
        │  or mints a new one on first use
        ▼
manager.EnsureHTTPSession(ctx, sessionID, cwd)
        │  loads from disk if persisted, creates fresh otherwise
        ▼
manager.HandleSessionPromptWithSender(ctx, params, sender, nil)
        │  runs the ReAct loop; sends stream events to Sender
        │  Sender streams tokens: first chunk → new Telegram message;
        │  subsequent chunks → progressive editMessageText (throttled);
        │  tool executions → shows "⚙️ toolname…" indicator in the live message.
        ▼
sender.Flush()
        │  replaces the live streaming message with the final formatted text
        │  (markdown.go renders the answer for Telegram: headings, double-star
        │   bold and tables into the legacy subset, fenced code untouched)
        ▼
Session bundle written to disk ($CODDY_HOME/sessions/<id>/)
```

**会话存储持久化** - 键→会话 ID 映射持久化在 `$CODDY_HOME/sessions/` 内的 `gateway_sessions.json` 中（与会话捆绑包同一目录）。重启时机器人会重新加载此文件并无缝继续现有对话。同一文件携带一个保留的 `$last_model` 条目：操作员在此网关上最后选择的模型（`/model` 键盘或输入的会话范围 `/model <id>`）。**全新**会话 - 聊天首次联系，或 `/clear` 后的新会话 - 会从它开始而不是 `agent.model`；在网关首次使用时，它会从按字母顺序第一个配置的模型开始。已经选择过模型的会话，包括有选择但尚无消息的会话，会保留它。

**`/clear` 流程：**

```
store.Reset(key)   → replaces stored session ID in gateway_sessions.json
manager.ForgetLiveSession(oldID)   → drops the in-memory session (disk persists)
Next message → EnsureHTTPSession creates a fresh session for the new ID
```

旧会话文件仍以旧 ID 保存在磁盘上。使用 `coddy sessions list` 检查它们，或在聊天中使用 `/resume` 返回其中一个。

**`/resume` 流程：**

```
manager.HandleSessionList(...)     → the sessions the server keeps, newest first
                                     (no folder filter: a chat has no cwd of its own)
/resume            → inline keyboard, one button per session: the title, or the
                     id of a session without one, then its age; the chat's own
                     session is marked; eight per page with Prev/Next
/resume <query>    → the session whose id the query is (byte for byte), or
                     whose id starts with it or whose title contains it (case-
                     insensitive); one match is resumed at once, several come
                     back as the keyboard
manager.EnsureHTTPSession(ctx, chosenID, cwd)   → loads the bundle first, so a
                                                   bundle that cannot be read is
                                                   reported here and nothing changes
store.Bind(key, chosenID)   → replaces the stored id in gateway_sessions.json
Next message → runs in the resumed session
```

聊天来源的会话保持加载。`/resume` 是切换，不是结束 - 聊天可以直接回来 - 而 `/clear` 表示对话已结束，从内存中丢弃它属于那里。点击一个比其会话存活的键盘 - 自显示列表后已从 Web 界面删除 - 不会绑定任何内容，并会在聊天中作为对键盘的回复回答。它不是警报：机器人会在每次点击到达时确认，而 Telegram 每次点击只接受一个回答。

---

## 安全说明

- **Token 暴露** - 切勿将机器人 token 提交到版本控制。在 YAML 中使用 `"${TELEGRAM_BOT_TOKEN}"`，并在启动前导出变量。
- **权限** - 网关会自动批准聊天智能体自己的工具权限请求，以便无人值守工作。如果要限制智能体可以运行哪些 shell 命令，请在 `config.yaml` 中限制 `tools.command_allowlist`。定义将其权限模式缩小到 `bypass` 以下的子智能体、`PreToolUse` 钩子的 `ask`，以及[决策检查](../features/decisions.md)未清除的命令，不会被放行：机器人会在聊天中用 **Allow** / **Reject** 按钮询问，并在子智能体询问时指出它 - 在轮次期间，以及后台子智能体在轮次结束后 - 只有其会话询问的人才能回答（在具有独立会话的群组中，其他成员的点击会被忽略，并保留所有者的按钮可用）。消息确定后会显示 *Allowed*、*Denied* 或 *No longer waiting*。
- **访问控制** - 对于只应响应特定用户集的机器人，设置 `default_access: "admins"`。开放机器人（`default_access: "all"`）会响应任何能向聊天写入的 Telegram 用户。
- **`/resume` 列出服务器的每个会话** - 包括在控制台或浏览器中启动的会话，因此在每个聊天中它都是管理员命令，私聊也是如此：不在 `gateways.telegram.admins` 中的发送者会收到仅管理员提示。从群组继续会话的管理员会将该会话的对话记录放在模型面前，并将其回答放在群组面前。对于多人都可写入的机器人，仍应保持 `default_access` 范围狭窄：在默认情况下 - 每个权限自动批准、shell 不受限制 - 磁盘上的捆绑包本来就在被允许用户的可及范围内。
- **Mini App** - 菜单按钮会显示给每个与机器人打开私聊的人，无论 `default_access` 说什么，点击它会到达 Web 界面地址。因此机器人仅在 Web 界面要求登录或 token，或 `httpserver.allow_insecure: true` 表示故意开放时，才会宣传它。Telegram 签名的启动数据（`initData`）仅让机器人的管理员登录，且仅通过服务器：`POST /coddy/auth/telegram` 用机器人 token 检查签名，每次启动只接受一次且最多一小时，并拒绝任何不在 `gateways.telegram.admins` 中的人，后者随后像在浏览器中一样登录。Web 界面本身会将启动数据排除在其地址之外，并且绝不将其作为身份。
- **网络** - 网关使用 Telegram 长轮询（不是 webhook）。无需开放入站端口。Mini App 是你选择的例外：其 `url` 必须能从打开它的人那里访问。

<!-- docsgen:source sha256=8dd9194a403218c9 -->

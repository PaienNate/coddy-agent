# 远程模式

一台机器上的 `coddy serve` 可以由另一台机器驱动。控制台和 `coddy acp` 接受 `--remote`，Web 界面在其侧栏底部有一个环境菜单，三者都通过服务器的 HTTP API 通信：智能体、它的工具、会话和权限策略位于服务器上，而客户端负责渲染对话记录并回答提示词。本页是用户指南；包含设计理由以及被推迟部分的设计记录见 [远程控制设计](../../plans/remote-control.md)。

## 什么是远程模式

| 客户端 | 如何连接 | 需要服务器提供什么 |
|---|---|---|
| 控制台（`coddy`、`coddy cli`、`coddy -p "..."`） | `--remote <target>` | HTTP API（`http` 构建标签，`httpserver.enable`，默认开启） |
| 通过 ACP 的编辑器（`coddy acp`） | `--remote <target>` | 同上 |
| Web 界面 | 侧栏底部的环境菜单 | 同上，外加服务器上的 CORS |

使用 `--remote` 时，本地进程不保留会话存储，不运行调度器，也不运行智能体循环；每个会话调用都通过 `/v1/*` 和 `/coddy/*` 代理到服务器，流式传输的答案通过 SSE 返回，并被转换为控制台和 ACP 客户端已经理解的更新。规范：`features/remote_client.feature` 和 `features/cli_remote.feature`；实时检查为 `examples/cli/cli_e2e_remote.py`、`examples/acp/acp_remote.py` 和 `examples/httpserver/http_e2e_remote.py`。

## 命名远程

`--remote` 接受三种写法：

- 来自客户端自己的 `config.yaml` 中 `httpserver.remotes` 的**配置名称**（不区分大小写匹配）；
- 裸 **`host:port`**，会在前面加上 `http://`；
- 完整的 **`http://` 或 `https://` URL**，可选带路径前缀。

地址必须是裸源：URL 中的查询字符串、片段或用户凭据都会被拒绝。命名远程包含一个名称、一个 URL，以及在你选择将其保存在那里时要出示的令牌：

```yaml
httpserver:
  remotes:
    - name: nas02
      url: "https://nas02.example:12345"
    - name: office-relay
      url: "http://relay.lan:12346"
      token: "${CODDY_RELAY_TOKEN}"   # optional, see The token below
```

```bash
coddy --remote nas02                 # by name
coddy --remote 192.168.1.20:12345    # bare host:port, plain http
coddy acp --remote https://nas02.example:12345
```

同一个 `remotes` 列表就是服务器在其 Web 界面的环境菜单中提供的列表，因此该条目有双重用途：把它放在你输入所用机器的配置中。集群中继也以同样的方式列出；菜单会识别它，并列出其背后的智能体（[环境菜单](#环境菜单)）。

## 令牌

在服务器上，设置令牌之前身份验证处于关闭状态。`config.yaml` 中的 `httpserver.auth_token`（支持 `${ENV}` 展开）、`coddy serve` 上的 `--auth-token`，或 `CODDY_HTTP_TOKEN` 变量会将其开启；此后每个 `/v1/*` 和 `/coddy/*` 路由都要求 `Authorization: Bearer <token>`，否则返回 `401`。SPA 外壳及其静态资源保持公开，以便浏览器加载页面并请求令牌；`/docs` 和 `/openapi.*` 受保护，除非 `httpserver.public_docs: true`。`GET /coddy/config` 从不返回令牌（它只报告 `auth_configured`），通过标志或环境提供的令牌会在 `PUT /coddy/config` 热重载后继续存在，而不会被写入文件。绑定到非回环地址且没有令牌的服务器会在启动时记录警告，除非 `httpserver.allow_insecure: true`。

令牌是 API 客户端出示的凭据。浏览器没有可输入令牌的字段，因此仅凭令牌会让 Web 界面显示 `Unauthorized (401)`；下面的登录表单是浏览器在这一道关卡中的另一半。

在客户端，令牌按以下顺序取第一个：

- `--remote-token`；
- 目标所属 `httpserver.remotes` 条目的 `token`：所指定的条目、其地址对应的条目，或目标作为节点挂载所属的中继条目（`<relay>/swarm/nodes/<name>`，它采用中继的客户端令牌）；
- `CODDY_REMOTE_TOKEN`。

条目的令牌优先于变量，因为它绑定到一个目标，而变量不绑定任何目标。它是可选的，把它保存在文件中是一种有代价的选择：每个读取此服务器配置（`GET /coddy/config`）的浏览器都会收到它，就像收到提供商的 `api_key` 一样，任何能读取该文件的人也会收到。将其写为 `${ENV}` 引用，让机密本身存放在 `.env` 中。不写它时，Web 界面会在其 **连接到…** 表单中接收令牌，并仅将其保存在浏览器中（`localStorage`，按远程分别保存），而控制台会读取标志或变量。

`coddy serve` 自身没有 TLS，因此发送到非回环 `http://` 地址的令牌会以明文传输；控制台和 `coddy acp` 在即将发生这种情况时会打印警告。在服务器前面放置一个终止 TLS 的反向代理，或通过 SSH 隧道访问它，这样地址会保持在回环上：

```bash
ssh -N -L 12345:127.0.0.1:12345 nas02   # in one terminal
coddy --remote 127.0.0.1:12345          # in another
```

## 登录表单

网络上的 `coddy serve` 对任何发现该端口的人都是可读的：对话记录、工具输出、工作区浏览器、配置编辑器。`httpserver.login` 用一个操作员账户将其关闭，而浏览器正是通过该表单登录。它默认关闭；未提及它的配置行为与之前完全相同。

```bash
coddy serve set-password --user pasha
```

该命令会要求输入两次密码，将 argon2id 哈希原地写入 `config.yaml`（注释和键顺序会保留），并开启该表单。此后，匿名浏览器会看到登录屏幕而不是应用，在通过验证之前，`/v1/*` 或 `/coddy/*` 下的任何内容都不可读。页面本身及其静态资源保持公开，以便登录屏幕能够加载；`/docs` 照旧遵循 `public_docs`。

对于容器或 systemd 单元，账户可以改为存放在环境变量中，即 `CODDY_HTTP_USER` 和 `CODDY_HTTP_PASSWORD`（通常在 `$CODDY_HOME/.env` 中）。它们会自行启用该表单，优先于文件中的账户，并且永远不会被写入文件。显式设置 `login.enable: false` 会在变量仍然设置的情况下关闭该表单。

```yaml
httpserver:
  host: 0.0.0.0
  auth_token: "${CODDY_HTTP_TOKEN}"   # for coddy --remote, coddy acp --remote, a relay, scripts
  login:
    enable: true
    user: "pasha"
    password_hash: "$$argon2id$$v=19$$..."
    session_ttl_hours: 720            # 0 = until the browser closes
```

**两种凭据不可互换。** 表单供浏览器使用；所有不是浏览器的东西，包括 `coddy --remote`、`coddy acp --remote`、挂载此节点的集群中继、Python 框架、Swagger 的 **授权**，仍然出示 bearer 令牌。仅用密码关闭服务器而不设置 `auth_token` 正是会破坏这些功能的原因，`coddy serve` 会在启动时的日志中说明这一点。当除浏览器之外的任何东西要与服务器通信时，请同时设置两者。

登录会设置一个 `HttpOnly`、`SameSite=Strict` cookie，它能打开令牌所能打开的相同路由，包括流式传输。退出登录（导航栏底部的条目）会丢弃服务器上的会话，因此在别处复制的 cookie 也会停止工作。会话保存在内存中：从页面保存设置会保留它们，重启进程会结束它们，轮换密码会结束它开启的每个会话。完整行为，包括针对 cookie 认证写入的 CSRF 规则以及对错误密码的限流，见 [HTTP API](../reference/http-api.md#web-ui-登录可选)。

`coddy serve` 仍然没有自己的 TLS，因此在非回环 `http://` 页面中输入的密码会像令牌一样以明文传输。在它前面放置一个终止 TLS 的反向代理，或通过上面的 SSH 隧道访问它。

## 为 Web 界面配置 CORS

从一个源加载 UI 并调用另一个源 API 的浏览器，需要远程服务器选择启用 CORS：

```yaml
httpserver:
  auth_token: "${CODDY_HTTP_TOKEN}"
  cors:
    enable: true
    allow_loopback: true          # the laptop's own coddy serve, on any loopback address and port
    allowed_origins: ["https://my-ui.example"]   # exact origins elsewhere; or ["*"]
  remotes:                       # optional: offered in this server's own UI
    - name: "prod box"
      url: "https://box.example:12345"
```

开启 `cors.enable` 后，来自允许源的预检请求会收到 `204`，其中包含 `Access-Control-Allow-Origin`（回显该源，或在配置后为 `*`）和 `Access-Control-Allow-Headers: Authorization, Content-Type, X-Coddy-Session-ID`；未列出的源不会收到 CORS 头，浏览器会将其报告为被阻止的请求。有三个设置说明允许哪些源。`allowed_origins` 精确指定它们，而 `http://localhost:12345`、`http://127.0.0.1:12345` 和 `http://localhost:5173` 是三个不同的源。`allow_loopback` 允许来自浏览器本机的每个页面，即主机为 `localhost` 的 `http` 或 `https` 源、`*.localhost` 名称、`127.0.0.0/8` 或 `[::1]`，任意端口，这正是笔记本电脑场景：Web 界面来自笔记本电脑自己的 `coddy serve`，其端口会随安装而变化；源会被回显，绝不会放宽为 `*`。`"*"` 允许任何位置的任何页面，并且由于列表会先被读取，其中的 `"*"` 也会用 `*` 回应回环源，无论 `allow_loopback` 是否开启。无论三者中哪一个允许该页面，bearer 令牌仍适用于真实请求：CORS 决定浏览器是否向页面显示答案，而不是服务器是否给出答案，因此 `allow_loopback` 和 `"*"` 的安全性仅取决于 API 背后的令牌或登录表单；当其中任一开启且服务器不要求凭据时，`coddy serve` 会在启动时警告，`--dry-run` 会报告它（[安全与信任](security.md#http-表面)）。集群中继有自己的 CORS，即 `swarm.cors`，使用相同的三个键：在别处提供并与中继通信的页面需要在那里加入其源，而中继背后的节点由中继的 CORS 应答，绝不会由节点自身的 CORS 应答。

![CORS 使回环地址上的页面无法访问远程时显示的健康警报](../../assets/remote/env-banner-cors-loopback-dark-1280.png)

*由笔记本电脑自己的 coddy serve 提供的页面：警报会指出精确源列表以及允许该页面使用任意端口的 `allow_loopback` 开关*

由于 `EventSource` 无法发送头，两个 SSE 订阅路由 `GET /coddy/sessions/{id}/composer-stream` 和 `GET /coddy/events` 也接受 `?access_token=`；捆绑的 UI 改为获取这些流，因此其头会生效，令牌不会落入 URL。参考：[HTTP API](../reference/http-api.md#身份验证可选)。

## 环境菜单

环境是 Web 界面侧栏底部的条目，位于“设置”之下、“退出登录”之上：当页面驱动自己的服务器时显示笔记本电脑图标，当它驱动远程时显示两个相互指向的尖括号，并用一个圆点表示该主机是否应答，工具提示会标明 **本地** 或活动远程。在平板电脑或手机上，它是顶栏图标之一，绝不会折叠到 **更多** 后面。它的菜单在侧栏旁边打开（窄屏上为从底部弹出的面板），包含一个 **环境** 部分，其中有“本地”，以及一个 **远程** 部分，其中有服务器配置的 `remotes` 加上 **连接到…**（名称、URL、令牌）。该列表在菜单打开时从页面自己的服务器读取，并在该服务器的配置被替换时再次读取，因此通过保存设置或智能体的 `config_commit` 添加的远程无需重新加载页面就会出现。没有 `name` 的条目会以远程报告的名称显示，即中继的名称（除非 `swarm.name` 设置了名称，否则就是其主机名）或智能体的主机名，旁边显示其地址；如果它没有报告任何内容，则仅显示其地址。选择一个条目会立即连接，并在其上重新启动应用，因此会话列表、模型和默认值会从所选后端重新读取：在两个远程之间就地切换，无需重新加载页面；切换到本地或从本地切换出去则需重新加载。选择页面已经所在的条目会重新读取所有内容，这是该远程宕机后的返回方式；SPA 外壳始终来自本地源，因此即使远程宕机，**本地** 仍可从菜单访问。

![侧栏旁边的环境菜单，显示一个中继、其智能体以及尚不能使用的远程](../../assets/env-menu-states-dark-1280.png)

*一个中继及其下方列出的智能体，一个需要其令牌的中继，一个被 CORS 阻止而无法从此页面访问的中继，以及一个不应答的中继*

菜单打开时会询问每个远程是否可以使用，圆点表示答案：绿色表示它应答并接受令牌，红色表示不可以，琥珀色表示正在询问。红色条目会在其下方的一行中说明原因：

- **不应答**：该地址没有监听任何内容，或地址错误；
- **拒绝令牌**：智能体需要其 `httpserver.auth_token`，中继需要其客户端令牌（`swarm.auth_token`）；对于配置中没有令牌的远程，**输入令牌** 会打开已为其填好的连接表单；
- **被 CORS 阻止**：有东西应答，但浏览器不让此页面获得答案，这就是从浏览器看 CORS 关闭（或缺少此源）时的样子。该行会指出此页面的源以及要将其加入的键：中继上的 `swarm.cors.allowed_origins`，coddy serve 上的 `httpserver.cors.allowed_origins`；并且，当此页面本身位于回环地址（笔记本电脑自己的 `coddy serve`）时，还会指出允许其使用任意端口的 `allow_loopback` 开关。

中继通过其公共 `GET /swarm/info` 被识别：它不提供自己的 `/v1`，因此询问的不是它的模型目录。它的令牌在 `GET /swarm/nodes` 上检查，一旦被接受，菜单会列出其背后的智能体，深入每一跳，因此节点只需点击一次即可到达，而不必藏在中继的映射后面（[集群，UI](swarm.md#界面)）。

活动环境在加载时、每 30 秒以及窗口获得焦点时会以同样方式被询问；当其无法使用时，横幅会说明它是三种情况中的哪一种，指出修复它的设置，并提供 **切换到本地**，而不是留下空屏幕。选择以及输入到表单中的每个远程的令牌保存在浏览器中（`coddy_env`、`coddy_env_tokens`）；来自配置的令牌不会复制到那里。文件夹最近记录按环境保留，每个环境离开时所处的屏幕也按环境保留：切换回本地会返回到你离开时所在的对话，而远程会打开在你离开的位置或其主页。

## 各部分在哪里运行

| 在服务器上 | 在客户端上 |
|---|---|
| ReAct 循环和模型调用，使用服务器的提供商和密钥 | 渲染对话记录、思考、工具框、计划更新、令牌和上下文统计 |
| 文件和 shell 工具，在服务器的工作区中 | 模型选择器，从服务器的 `GET /v1/models` 填充 |
| MCP 服务器、钩子、子智能体及其信任回执 | 权限和问题提示词，通过 `POST /coddy/sessions/{id}/permission` 和 `.../question` 回答 |
| 会话捆绑包、配置文件以及智能体的自配置提交 | 取消（`POST /coddy/sessions/{id}/cancel`）、`ctrl+o` 获取完整工具输出 |

远程控制台或 ACP 客户端创建的会话位于服务器的默认工作目录中（`coddy serve --cwd`、`CODDY_CWD`，否则就是服务器启动时所在的目录）；远程控制台页脚仍显示本地文件夹。Web 界面可以通过其文件夹标签将会话移动到服务器端的另一个文件夹。`/export` 写入服务器的工作区。`/resume`、`-c` 和 `--session-id` 作用于服务器的会话列表，本地文件夹过滤器不适用。设置命令（`/model`、`/reasoning`、`/agent`、`/plan`、`/ask`、`/permissions`，带 `--once` 或 `--count=N`）通过 Web 界面使用的同一个 `PATCH /coddy/sessions/{id}` 更改服务器的会话，包括权限模式，并且该会话的每个客户端都会看到更改（[会话设置](../features/session-settings.md)）。控制台的 `/reasoning [level]` 和 `shift+tab` 会将所选的推理级别持久化到服务器会话上。启动横幅显示 `remote: <url>`，退出提示会打印一条包含 `--remote` 的重连命令。

信任决策也属于服务器。项目 MCP 服务器、钩子文件或子智能体定义在该主机上获得批准，可以使用那里的 CLI（`coddy mcp trust`、`coddy hooks trust`、`coddy agents trust`），或通过带 bearer 令牌的 API（`POST /coddy/mcp/{name}/trust`、`POST /coddy/hooks/trust`、`POST /coddy/subagents/{name}/trust`，每个都以服务器端工作区作为 `cwd`）；本地子命令不接受 `--remote`。子智能体的权限提示词会在父会话下到达远程客户端，前缀为 `[subagent <name>]`，即使父会话运行在 `bypass` 中也是如此。参见 [子智能体](../features/subagents.md#远程模式)。

## 限制

- 每个服务器一个 bearer 令牌，没有用户或角色：持有它的人拥有整个 API，包括配置编辑器和工作区文件。
- `coddy serve` 内部没有 TLS；加密来自代理或隧道。
- 远程客户端切换的权限模式（`/permissions`、`--permission-mode`、对话框中的会话开关）会作用于服务器会话，并对该会话的每个客户端生效，还会随该会话保存在服务器上，在 `coddy serve` 重启后仍然保留。它也会成为该服务器上新会话启动时使用的模式。
- 连接断开后，服务器上的轮次、其子项以及任何打开的提示词会继续运行；`/resume` 会在轮次结束后显示结果，对服务器已经撤回的提示词的回答会被忽略。在轮次中途退出控制台会短暂等待取消到达服务器。
- 一个会话一次只能进行一个轮次：当第一个轮次持有锁时，第二个客户端向同一会话发送提示词会收到 `409`。
- 只有当该服务器通过其 `cors`（中继上为 `swarm.cors`）允许 UI 的源时，Web 界面才能访问远程：在 `allowed_origins` 中指定精确源，或对由笔记本电脑自己的 `coddy serve` 提供的页面使用 `allow_loopback`；当阻碍正是这一点时，菜单会在远程条目行中说明。

## 从命令行检查远程

```bash
coddy --dry-run --remote nas02
curl -sS -H "Authorization: Bearer $CODDY_REMOTE_TOKEN" https://nas02.example:12345/v1/models
```

`--dry-run`（在控制台上以及 `coddy acp` 上）会运行配置检查，然后探测文件所指向的内容，包括 `--remote` 目标，并在探测失败时以状态 1 退出。每个配置的远程都会用其条目的令牌进行探测；中继会被识别为中继，而指向中继根路径的 `--remote` 会被拒绝，并给出应改用其智能体的挂载（`<relay>/swarm/nodes/<name>`），因为中继本身不驱动任何东西。`curl` 行是 UI 的状态圆点向智能体发出的第一个请求。在服务器端，`coddy serve --dry-run` 会报告它将绑定的监听地址以及端口是否已被占用。参见 [coddy serve 和守护进程](serve.md)。

## 历史

设计记录 [远程控制设计](../../plans/remote-control.md) 记录已发布的方向，即经过身份验证的 API、CORS、环境选择器以及 `--remote` 背后的 Go 客户端，也记录了未实现的部分：服务器内部的 TLS，以及带智能体注册表的反向隧道，后者后来由 [集群中继](swarm.md) 通过其反向 HTTP/2 隧道覆盖。

<!-- docsgen:source sha256=838461967b51f472 -->

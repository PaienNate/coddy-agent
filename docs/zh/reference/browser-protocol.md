# 浏览器协议

`coddy serve` 与 [Coddy for Browser](../features/browser.md) 扩展之间的通信线路。有两个仓库实现它：本仓库 `external/browser` 中的端点与 `browser` 工具，以及 [coddy-project/coddy-for-browser](https://github.com/coddy-project/coddy-for-browser) 中的扩展。协议的一处改动会同时落在两者上，本页也随之更新。

当前版本为 **1**。

## 传输

扩展向 `coddy serve`（其归属服务器）的 HTTP 服务器打开一条 WebSocket 并保持打开。服务器从不连接浏览器：Coddy 向它请求的一切都沿着这条套接字下行，因此 Coddy 能到达一个它没有路由的浏览器。当 `browser.enable` 开启时，端点以 `browser.token` 应答，否则以 `404` 应答。

| | |
| --- | --- |
| 地址 | HTTP 服务器上的 `ws://<host>:<port>/browser/v1/connect`，即 Web 界面的地址；位于 HTTPS 代理之后时为 `wss://` |
| 源 | `chrome-extension://<id>`（Chrome、Yandex Browser）或 `moz-extension://<uuid>`（Firefox）；服务器会以任何其他 `Origin` 头或缺失该头来拒绝升级 |
| 消息 | 每个文本帧一个 JSON-RPC 2.0 对象，双向皆然 |
| 大小 | 在 `initialize` 得到应答之前为 16 KiB，之后为 8 MiB |
| 存活 | 扩展每 20 s 发送一次 `$/ping`；服务器会关闭静默 60 s 的套接字 |

端点的路由位于 HTTP API 自身的关卡之外：扩展自己证明密钥，在 `initialize` 中以及每次文件传输时都是如此。另一个 Coddy 进程的对话从不使用这条套接字；它通过归属服务器的 API 到达浏览器（[其他 Coddy 进程](#其他-coddy-进程)）。

双方都会发送请求、应答和通知。Coddy 的请求 id 是 48 个十六进制数字组成的字符串；扩展的 id 是它自己的。对另一方从未发送过的 id 的应答会被忽略。

## 握手

首条消息是扩展的 `initialize` 请求。密钥随它一同传递，因为浏览器无法在 WebSocket 上设置头。服务器对任何其他首条消息都以 `-32600` 应答并关闭套接字。

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": 1,
    "key": "<browser.token>",
    "browser": { "id": "6f1c0a9e-2d4b-4b8e-9a51-3c7e0f2d9b10", "name": "Chrome - laptop", "version": "0.1.0" },
    "actions": ["open_tab", "close_tab", "release", "snapshot", "screenshot", "navigate", "click", "type", "press", "hover", "scroll", "drag", "upload_file", "download_file"],
    "grant": "b0f3a5d2-..."
  }
}
```

- `browser.id` 在一次安装中保持不变：8 到 80 个字母、数字和 `-` 字符。使用相同 id 的第二个连接会替换第一个连接，第一个连接中仍在等待的请求会以未知结果失败；
- `browser.name`（1 到 100 个字符）在连接多个浏览器时用于区分它们；
- `actions` 列出扩展实现了哪些能力。Coddy 不会发送其他动作；
- `grant` 指明扩展当前的选择，与 `browser/tabs` 的作用相同，因此在该标签页列表之前的命令会携带正确的选择。

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": 1,
    "server": { "name": "coddy", "version": "1.4.0" },
    "limits": { "maxFileBytes": 33554432 }
  }
}
```

| 错误 | 含义 |
| --- | --- |
| `-32001` | 密钥与 `browser.token` 不匹配。 |
| `-32002` | 另一个协议版本；`data.supported` 列出服务器支持的版本。 |
| `-32600` | 首条消息不是 `initialize`。 |
| `-32602` | 参数无效，其中包括格式错误的浏览器 id 或名称。 |

出错后服务器会关闭套接字。应答之后紧接着，扩展会发送 `browser/tabs`，空列表也不例外。

## 标签页

扩展决定 Coddy 能到达哪些标签页，并在列表变化时完整上报它们。

```json
{
  "jsonrpc": "2.0",
  "method": "browser/tabs",
  "params": {
    "tabs": [
      { "id": 41, "url": "https://example.com/", "title": "Example Domain" },
      { "id": 77, "url": "https://example.org/report", "title": "Report", "sessionId": "sess_8f2c..." }
    ],
    "grant": "b0f3a5d2-..."
  }
}
```

- 没有 `sessionId` 的标签页是共享的：任何对话都可以在其中操作，其第一个动作会把它移入该对话。带有 `sessionId` 的标签页属于该对话，服务器在发送前会拒绝其他对话对它的命令；
- 至多 100 个标签页，仅限 `http` 和 `https` 页面，URL 至多 8192 个字符，标题至多 1000 个字符。违反这些规则的列表会终止连接；
- `grant` 指明当前的选择。用户按下 Stop 时扩展会替换它，并拒绝携带更旧选择的命令。空的 `grant` 会保留前一个选择；
- 扩展在应答改变了列表的命令之前先发送 `browser/tabs`，因此对 `open_tab` 的应答绝不会指名 Coddy 尚不知道的标签页。

在 Coddy for Browser 中，该列表就是浏览器中其标签页组的内容：共享的 **Coddy** 组，以及每个对话一个以其标题命名的组。

## 命令

Coddy 为对话的每个动作发送一个 `browser/command` 请求。

```json
{
  "jsonrpc": "2.0",
  "id": "9a1f...",
  "method": "browser/command",
  "params": {
    "action": "click",
    "tabId": 41,
    "params": { "ref": "4c1d9e2a:7" },
    "grant": "b0f3a5d2-...",
    "session": { "id": "sess_8f2c...", "title": "Check the release notes" }
  }
}
```

`params` 保存动作自身的参数，与对话的模型所写的一致。`session.title` 为标签页组和页内指示器指明对话。`tabId` 在 `open_tab` 和 `release` 中省略。

另一个 Coddy 进程的对话携带 `session.node`，即它所在机器的自称名称，其 `session.id` 为 `<node>/<session id>`；归属服务器的会话 id 绝不携带 `/`。扩展在此类对话的第一个动作之前询问用户，并把该节点加入其标签页组的标题。

```json
"session": { "id": "laptop/sess_31d0...", "title": "Plan the trip", "node": "laptop" }
```

| 动作 | 参数 | 结果 |
| --- | --- | --- |
| `open_tab` | `url` | 新标签页的 `tabId`、`url`、`title`，在其加载完成后；该标签页起始于该对话的组 |
| `close_tab` | - | `closed` |
| `release` | - | `released`：该对话有多少标签页回到了共享组 |
| `snapshot` | - | `url`、`title`、`text`（至多 20000 个字符）、`elements`（至多 500 个：`ref`、`tag`、`name`、`x`、`y`，以及适用时的 `role`、`type`、`href`、`value`、`checked`、`disabled`、`offscreen`）、`viewport` |
| `screenshot` | - | `dataUrl`（`data:image/png;base64,...`，一个图像像素对应一个 CSS 像素）、`width`、`height` |
| `navigate` | `url`，或 `back` / `forward` | `url`，加载后的 `title` |
| `click` | `ref`，或 `x` 和 `y`；`button`（`left`、`right`、`middle`）、`count`（1 到 3） | `ok`；当输入经由页面完成时为 `trusted: false` |
| `type` | `text`；`ref`、`clear` | `ok` |
| `press` | `key`（`Enter`、`Tab`、`Escape`、`Backspace`、`Delete`、方向键、`Home`、`End`、`PageUp`、`PageDown`、`Space`、一个字母或数字，配合 `Ctrl+`、`Alt+`、`Shift+`、`Meta+`） | `ok` |
| `hover` | `ref`，或 `x` 和 `y` | `ok` |
| `scroll` | `dx`、`dy`；`x`、`y`（省略时为视口中心） | `ok` |
| `drag` | `x`、`y`、`to_x`、`to_y` | `ok` |
| `upload_file` | 文件输入框的 `ref`、`transfer_id`、`filename` | `filename`、`bytes` |
| `download_file` | `url`、`transfer_id`、`filename` | `bytes`、`filename` |

- 坐标为视口的 CSS 像素。`ref` 来自同一页面最近一次的 `snapshot`，并在页面变化时失效；
- 除 `open_tab`、`close_tab` 和 `release` 之外的每个结果还携带 `tab`：动作之后该标签页的 `id`、`url` 和 `title`；
- 没有浏览器调试器的扩展（Firefox）会在页面自身中派发 `click`、`type`、`press`、`hover`、`scroll` 和 `drag` 的事件，并在其结果中加上 `trusted: false`：页面可以将它们与人的输入区分开；
- 动作失败时返回错误应答，代码为 `-32000`，附带对话可读取的消息，以及一个位于 `not_shared`、`other_chat`、`stopped`、`chat_declined`、`chat_prompt_timeout`、`site_declined`、`cross_origin_frame`、`site_prompt_timeout`、`page_changed`、`cancelled`、`unsupported`、`invalid_params`、`too_large`、`transfer_failed`、`download_failed` 之中的 `data.reason`。

Coddy 等待应答 2 分钟，等待 `download_file` 10 分钟。当对话的轮次被取消或等待超时，Coddy 发送 `$/cancelRequest` 并向对话报告结果未知；它绝不会自行重复动作。

```json
{ "jsonrpc": "2.0", "method": "$/cancelRequest", "params": { "id": "9a1f..." } }
```

## 网站批准

在某个网站（协议、主机和端口）上的第一个动作之前，扩展会在它自己的窗口中，或在打开时的侧边面板中询问用户：允许此对话、始终允许，或拒绝。`open_tab`、`navigate` 和 `download_file` 会询问它们要前往的站点。问题在 110 s 内未得到回答时，命令以 `site_prompt_timeout` 失败；被拒绝时以 `site_declined` 失败。该问题绝不会到达 Coddy：用户的回答留在浏览器中。

携带 `session.node` 的对话会先被询问一次：允许此对话，或拒绝（`chat_declined`、`chat_prompt_timeout`）。"始终允许" 仅覆盖归属服务器的对话，因此每个新网站都会为另一进程的对话再次询问，而 Stop 会收回对此类对话的允许。

始终允许的站点列表以 `localhost`、`127.0.0.1` 和 `[::1]` 的主机条目开头（`*://<host>:*`，任意协议和端口），用户可以像撤销其他条目一样撤销它们。主机条目在每个本地名称下都不覆盖归属服务器自身的端口，而携带 `session.node` 的对话会像其他对话一样询问本地源。用户可以事先添加条目：一个源、`*://<host>:<port>`、一个范围 `*://<host>:<lo>-<hi>`，或在每个端口上的一种协议（`http://<host>:*`），也可以一次允许所有网站。只有指明某一个端口的条目才覆盖归属服务器自身的端口。

## 文件

`upload_file` 和 `download_file` 不在套接字上携带文件内容。Coddy 为调用的时长开启一张一次性票据，并发送其 id 作为 `transfer_id`；扩展在同一服务器上通过 HTTP 传输字节，以密钥作为 bearer token。

| 请求 | 方向 | 应答 |
| --- | --- | --- |
| `GET /browser/v1/files/<transfer_id>` | Coddy 到浏览器：要在输入框上设置的文件 | 带字节的 `200` |
| `POST /browser/v1/files/<transfer_id>` | 浏览器到 Coddy：它下载的字节 | `204` |

一张票据在 2 分钟内服务一次请求，之后应答 `410`；密钥错误时应答 `401`。文件至多 32 MiB。扩展使用浏览器的 cookie 获取下载，并拒绝重定向，因此对话需传入最终地址。

## 通知

| 方法 | 方向 | 参数 | 含义 |
| --- | --- | --- | --- |
| `browser/tabs` | 扩展到 Coddy | `tabs`、`grant` | Coddy 可到达的标签页，完整列表。 |
| `browser/stop` | 扩展到 Coddy | - | 用户按下了 Stop：所有仍在等待的命令失败，且在下一个 `browser/tabs` 之前不共享任何标签页。 |
| `$/ping` | 扩展到 Coddy | - | 保持套接字存活，Chrome 的 service worker 也随之存活。 |
| `browser/turnEnded` | Coddy 到扩展 | `sessionId` | 对话的轮次已结束；扩展移除其工作标记。Coddy 只向持有该对话标签页的浏览器发送它。 |
| `$/cancelRequest` | Coddy 到扩展 | `id` | 停止具有该 id 的命令。 |

一方会忽略它不认识的通知，并以 `-32601` 应答它不认识的请求，因此任一方都可以在一个协议版本内先行升级。

## 侧边面板

扩展的侧边面板在框架中显示归属服务器的 Web 界面。它通过套接字申请票据。

```json
{ "jsonrpc": "2.0", "id": 7, "method": "panel/open", "params": {} }
```

```json
{ "jsonrpc": "2.0", "id": 7, "result": { "ticket": "3f9c...64 hex digits" } }
```

一张票据在一分钟内仅可用于一次登录，并绑定到为其签发的密钥。面板加载 `<server>/?coddy_panel=<ticket>&coddy_panel_origin=<extension origin>`（`chrome-extension://<id>` 或 `moz-extension://<uuid>`）；Web 界面从地址中取出这两者，并在 `POST /coddy/auth/browser` 兑换票据以换取它自己的会话 cookie，该 cookie 在 `browser.token` 变化时结束（[HTTP API](http-api.md#web-ui-登录可选)）。

框架和面板之后通过 `postMessage` 通信，各自仅向对方的源发送，且各自仅接受来自该源上对方窗口的消息。

| 消息 | 来自 | 字段 | 含义 |
| --- | --- | --- | --- |
| `{source: "coddy-web-ui", type: "turn"}` | Web 界面 | `requestId`、`sessionId`、`title` | 在发送轮次之前：人正在从哪个标签页书写？ |
| `{source: "coddy-for-browser", type: "turn"}` | 面板 | `requestId`、`tab`（`browser_id`、`tab_id`、`url`、`title`）或 `null`、`error` | 该标签页，已位于对话的标签页组中。Web 界面将其作为 `POST /v1/responses` 的 `metadata.browser` 发送。 |
| `{source: "coddy-web-ui", type: "theme"}` | Web 界面 | `theme`：`scheme` 以及作为计算颜色的 `bg`、`surface`、`border`、`text`、`muted`、`accent`，`rgb()` 或 `color(srgb ...)` | 面板的条带采用 Web 界面的主题；任何其他值都会被拒绝。 |
| `{source: "coddy-for-browser", type: "open"}` | 面板 | `sessionId` | 显示服务器的此对话（`#/s/<sessionId>`）。只接受形如 `sess_...` 的 id，且仅在 Web 界面显示面板自身服务器时。 |

Web 界面等待面板的应答 3 s，若无应答则在无标签页的情况下发送轮次。Web 界面另一环境的对话不会获得标签页。

在归属服务器的对话所持有的标签页上打开时，面板会在该对话上加载 Web 界面：`<server>/?coddy_panel=...&coddy_panel_origin=...#/s/<session id>`。

## 其他 Coddy 进程

不在归属服务器进程中运行的对话 - 同一台机器上的控制台或编辑器、集群的一个节点 - 在 `browser.serve_remote` 开启时，通过归属服务器的 HTTP API 到达其浏览器，位于服务器常规凭据之后。通过集群中继时，这些路由以 `<relay>/swarm/nodes/<node>/coddy/...` 承载。

| 请求 | 作用 |
| --- | --- |
| `GET /coddy/browser/v1/browsers?node=&session_id=` | 已连接的浏览器及其标签页，按该对话所见 |
| `POST /coddy/browser/v1/call` | 一个动作：`browser_id`、`action`、`tab_id`、`params`、`caller`（`node`、`session_id`、`title`），以及 `file`（`name`、base64 形式的 `data`），用于 `upload_file` 和 `download_file` |

归属服务器通过套接字将该动作连同 `session.id` `<node>/<session id>` 和 `session.node` 下行发送，并以 `result` 应答，对于 `download_file` 则应答 `file` 中的字节。被拒绝或失败的动作以 `409` 应答并附带扩展的消息。调用方通过丢弃其请求来取消动作，归属服务器会发送 `$/cancelRequest`。归属服务器看不到远程对话的轮次在何处结束，因此在对话最后一次调用后 45 s 为其发送 `browser/turnEnded`。字段和状态码见 [HTTP API](http-api.md)。

<!-- docsgen:source sha256=3970dde76402c0bb -->

# HTTP 请求

`http_request` 工具是智能体的 curl。模型自行选择方法、查询参数和每一个请求头，发送任意类型的请求体或工作区文件，并按照 `curl -i` 打印的方式来读取响应。智能体用它调用 API、检查自己刚在 localhost 上启动的开发服务器，以及上传或下载文件。

`webfetch` 仍然是用来阅读文章的工：它接收一个公开 URL 并以 Markdown 返回正文。两者共用一个 HTTP 客户端 - `webfetch` 就是契约更窄的 `http_request`，见[下文](#同一客户端上的-webfetch)。

由于模型自行构造的请求可能会改变别处的东西，也可能携带智能体读到的任何内容，因此它要经过权限门禁：在 `ask` 下，操作者会在任何内容发出之前看到请求发往哪里、携带了什么。

## 一次请求及其响应

```json
{
  "method": "PATCH",
  "url": "https://api.example.com/items/7?verbose=1",
  "query": { "tag": ["red", "blue"] },
  "headers": { "Authorization": "Bearer ${TOKEN}", "X-Trace": "abc-123" },
  "json": { "name": "lamp" }
}
```

该请求发往 `https://api.example.com/items/7?verbose=1&tag=red&tag=blue`，携带 JSON 请求体 `{"name":"lamp"}` 和 `Content-Type: application/json`。响应如下：

```text
HTTP/2.0 200 OK
Content-Type: application/json
Date: Wed, 16 Sep 2026 12:00:00 GMT

{"id":7,"name":"lamp"}
```

![Web 界面对话记录中一次完成的 http_request 调用](../../assets/http-requests/http-request-answer-dark-1280.png)

*对话记录中的调用及其响应：首先是原因、请求头和发出的请求体，然后是响应状态、响应头和响应体。*

4xx 或 5xx 状态与其他响应并无不同，不是失败的调用：模型读取状态行并决定怎么做。只有在没有收到任何响应时调用才失败 - 地址无法解析、连接被拒绝、证书未通过验证、超时耗尽 - 或者请求体无法保存到 `output_file`，或者参数在发送任何内容之前就被拒绝。

## 参数

| 参数 | 含义 |
|----------|---------|
| `url` | 必填。一个绝对的 `http://` 或 `https://` URL；查询参数可以写在其中。 |
| `method` | 任意方法标识，发送时会转为大写：`GET`、`POST`、`PUT`、`PATCH`、`DELETE`、`HEAD`、`OPTIONS` 或自定义方法。默认为 `GET`，给出请求体时为 `POST`。 |
| `query` | 追加到 `url` 中已有参数之后的查询参数。值可以是字符串、数字、布尔值，或用于重复名称的这些类型的数组。 |
| `headers` | 按名称给出的请求头。见[请求头](#请求头)。 |
| `body` | 原始请求体，按原样发送。 |
| `body_base64` | 二进制请求体，base64 编码；未设置时按 `Content-Type: application/octet-stream` 处理。 |
| `body_file` | 作为整个请求体发送的本地文件；未设置时其内容类型跟随其扩展名 `Content-Type`。 |
| `json` | 任意 JSON 值，使用 `Content-Type: application/json` 压缩后发送。本身持有 JSON 对象或数组的字符串会作为该文档发送。 |
| `form` | 作为 `application/x-www-form-urlencoded` 请求体发送的字段；值的处理方式同 `query`。 |
| `form_data` | 按顺序排列的 `multipart/form-data` 各部分：每个部分有一个 `name`，以及一个 `value` 或一个本地 `file`，还可选 `filename` 和 `content_type`。 |
| `output_file` | 将响应体保存到该路径而不是返回它。 |
| `follow_redirects` | 跟随仍留在同一源上的重定向。默认 `false`。见[重定向](#重定向)。 |
| `proxy` | 用于本次请求的代理，或 `direct`。见[代理与证书](#代理与证书)。 |
| `verify_tls` | `false` 接受任意服务器证书，类似 `curl -k`。默认 `true`。 |
| `timeout_seconds` | 整个交换（包括读取请求体）的时间上限。默认 30，最多 600。 |
| `permission_rationale` | 显示在权限提示顶部的文本。 |

一次请求最多携带一个载荷：`body`、`body_base64`、`body_file`、`json`、`form` 或 `form_data`。同时给出其中两个会被拒绝，并在错误中同时指明两者。`body_file`、`form_data[].file` 和 `output_file` 中的路径会相对于会话的工作目录解析，不存在的文件会在发送任何内容之前使调用失败。

multipart 请求体在发送前就已排布好，文件作为已知大小的部分，因此请求携带精确的 `Content-Length` - 拒绝分块上传的服务器也会接受 - 并且文件内容是流式发送而不是读入内存。

## 请求头

在 `headers` 中指定的请求头会替换工具原本在该名称下要发送的任何内容：它自己的 `User-Agent`、载荷的 `Content-Type`、地址的 `Host`，以及操作者配置的[默认请求头](#默认请求头)。空值会移除工具原本要发送的请求头：`"User-Agent": ""` 不发送任何内容，`"Content-Type": ""` 让 JSON 请求体不带该头。

有三个请求头是被检查而非直接透传的：

- `Content-Length` 必须等于请求体的大小 - 省略它则自动计算；
- `Transfer-Encoding` 只接受 `chunked`，表示发送请求体时不带长度；
- `Host` 无法移除，只能替换。

该工具不自行发送 `Accept-Encoding`。当请求自身要求 `gzip` 或 `deflate` 时，请求体会被解码以供阅读，并按到达时的样子保存到 `output_file`。

## 默认请求头

有些请求头属于每个请求，而不属于某一次调用。托管前端后面的站点可能只向浏览器提供其文件：`https://match3.drobek.online/app.webmanifest` 对工具自身的 `User-Agent` 返回 `415 Unsupported Media Type` 和一个 HTML 页面，而对带 manifest 的 Chrome `User-Agent` 则正常响应。内部服务可能要求每次调用都带某个 `Accept` 或客户端请求头。`tools.http_request.default_headers` 可以一次性指定这类请求头：

```yaml
tools:
  http_request:
    default_headers:
      User-Agent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
      Accept: application/json
```

工具按三层构建请求的请求头，后一层优先：

1. 工具自身的默认值：`coddy-agent` `User-Agent`，以及载荷的 `Content-Type`，后者无法由 `default_headers` 指定（见下文）；
2. `default_headers`；
3. 该次调用的 `headers`。

调用中指定的请求头会替换已配置的那个，无论调用如何拼写其名称，而调用中的空值会移除它：`"User-Agent": ""` 之后完全不再发送 `User-Agent`。`default_headers` 中的空值会让该请求头从所有未设置它的调用中消失，因此 `User-Agent: ""` 会让工具不再自我标识。没有该键时，请求与之前完全一样地发出。

`Host`、`Content-Type`、`Content-Length` 和 `Transfer-Encoding` 会被拒绝：工具从每次调用中推导它们 - 其地址、其载荷以及载荷的框架 - 需要其他值的调用可在自己的 `headers` 中设置。逐跳请求头亦然 - `Connection`、`Keep-Alive`、`Proxy-Connection`、`TE`、`Trailer` 和 `Upgrade` - 它们描述的是单个连接而非客户端；为每个请求设置它们会破坏这些请求（HTTP/2 会拒绝携带 `Upgrade` 的请求）。`Proxy-Authorization` 同样会被拒绝：`https` 请求以隧道方式经过代理，并把其请求头在隧道内带到源站，因此在此设置的代理凭据会抵达每一个目的地；它应放在代理地址中（`HTTPS_PROXY`，或某次调用的 `proxy` 中的 `http://user:password@host:3128`）。名称须符合所有在用请求头的形状 - 字母、数字、`-` 和 `_`，以字母开头 - 因为它同时还是配置路径的一段，在那里点号会将其拆分，数字会被当作列表位置；调用仍可在自己的 `headers` 中发送任何其他名称。同一请求头的两种拼写以及包含换行符的值也会被拒绝。加载器会在启动时拒绝这样的配置，`coddy -t` 会在其报告中指出该请求头。

已配置的请求头的作用如同调用写出了它：例如这里的 `Accept-Encoding: gzip`，会让每个请求体都被解码以供阅读，并按到达时的样子保存到 `output_file`。

只有 `http_request` 会发送这些请求头。`webfetch`、`@https://...` 提及所读取的页面，以及发往模型提供商的请求，都保持各自的做法。

在 Web 界面中，这个映射就是设置里 **Tools and permissions** 选项卡上 **HTTP requests** 的 **Default headers** 块：每个请求头一行，包含其名称和值，行下方有 **Add**。

![Web 界面设置中 http_request 的默认请求头](../../assets/http-requests/http-request-default-headers-settings-dark-1280.png)

*Tools and permissions 选项卡的 HTTP requests 块：先是允许列表，然后是每个默认请求头一行。*

### 提示词显示的内容

权限提示会把已配置的请求头列在将要发出的请求头之中，并在单独一行中指明它们，以便操作者将其与模型写出的请求头区分开；被配置以空值省略的请求头也会在那里标明，写作 `User-Agent (not sent)`：

```text
GET https://match3.drobek.online/app.webmanifest
Headers:
  Accept: application/json
  User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36
Headers from tools.http_request.default_headers: Accept, User-Agent
```

对于表明客户端身份或可接受内容的请求头，提示会显示其配置值 - `User-Agent`、`Accept` 系列、`Cache-Control`、`Pragma`、`DNT`、`Origin`、`Upgrade-Insecure-Requests`，以及浏览器添加的 `Sec-Ch-Ua` 客户端提示和 `Sec-Fetch-` 元数据。其他任何已配置的请求头只显示其名称，值显示为 `<redacted>`：凭据可以藏在任何名称下（`X-Auth`、`X-Session`），而且该提示还会到达 Telegram 聊天、通知钩子以及任何看屏幕的人。模型在调用本身中写出的请求头按原样显示，与它在对话记录中的样子一致。

请求头是配置的一部分，授予不会记录它们：获得批准或在允许列表中的目的地，其请求会带上 `default_headers` 当时的任何内容。在 `coddy serve` 重启后仍在等待的提示，按其展示的请求来作答：当答案到达时 `default_headers` 已发生变化，调用会用现在将要发送的请求重新询问，而不是按旧答案执行。

### 凭据

该工具的每个请求都携带这些请求头，发往模型用它抵达的每一个目的地：你原本想的那项服务、搜索找到的页面、提示词注入所要求的地址。在此设置的 `Authorization`、`Cookie` 或 API 密钥请求头会发往所有这些目的地，在 `bypass` 下或对允许列表中的目的地都无需提示。除非这正是意图（例如在一台只与某个服务通信的机器上），否则请把凭据排除在 `default_headers` 之外；否则让调用携带凭据，提示会针对它所要抵达的那一个目的地显示它。

`config_get` 向模型显示配置了哪些默认请求头，而从不显示其值，就像它对待 MCP 服务器的请求头值那样 - 空值除外，空值表示该请求头被省略；文件和设置界面会显示它们。模型按其读到的值原样回填的 `<redacted>`，会被拒绝，而不会覆盖该请求头。

## 响应

首先是状态行，然后是按名称排序的响应头，一个空行以及响应体。关于响应体未展示内容的说明以方括号形式附在末尾。

当响应体是文本时就以文本显示：按其 `Content-Type` 声明的字符集解码（`windows-1251` 到达时即可读），或在不含 NUL 字节且为有效 UTF-8 时按 UTF-8 显示。其他内容则以描述代替转储：

```text
[binary body: 48213 bytes, image/png; pass output_file to save it]
```

响应中最多读入 1 MiB 的响应体，被截断时会有说明，而到达模型的内容还会被 `tools.output_limits.default` 进一步截断 - 因此较大的或二进制的响应体应放入 `output_file`。下载内容先写到目标位置旁边，完成后再重命名到位，因此失败或过大的下载（超过 1 GiB）绝不会以该名称留下部分文件；响应中会说明 `[body: saved 48213 bytes to /path/logo.png]`。

## 重定向

除非调用传入 `follow_redirects: true`，否则不跟随重定向。3xx 响应会连同其 `Location` 一起返回，由模型决定是否前往。

使用 `follow_redirects: true` 时，只有当重定向仍留在请求的源上 - 相同的协议、主机和端口 - 或把同一主机上的 `http` 升级为 `https` 时才会被跟随。指向其他源的重定向会被保留：响应就是该重定向，并附有一条说明指出它指向的地址，要抵达它需要发起新的调用，该调用会自行经过权限门禁。因此对某个服务的批准绝不会被该服务本身带到另一个服务。`307` 或 `308` 会重新发送请求体，包括文件。

## 代理与证书

没有 `proxy` 时，请求会像 curl 一样使用 Coddy 进程所在环境指定的代理（`HTTPS_PROXY`、`HTTP_PROXY`、`NO_PROXY`）；发往 `localhost` 或环回地址的请求从不经过代理。`proxy` 可为单个请求覆盖这一点：

- `http://host:port`、`https://host:port`、`socks5://host:port` 或 `socks5h://host:port` 会通过该代理发送请求，URL 中的凭据（`http://user:pass@host:3128`）用于向它认证；`socks5h` 通过代理解析目的地的名称；
- `direct` 忽略环境中的代理，直接连接到目的地。

`verify_tls: false` 接受任意服务器证书 - 自签名的开发服务器、私有 CA 后面的内部服务。提示会以大写字母说明这一点。

## 权限

`http_request` 的门禁方式与 `run_command` 相同：是否需要询问取决于会话的权限模式、`tools.http_request.allowlist` 以及本会话中先前批准的内容。

| 模式 | 会询问什么 |
|------|-----------|
| `bypass` | 什么都不问。 |
| `accept_edits` | 目的地不在允许列表中也未获批准，或携带了该目的地的批准未覆盖内容的请求。 |
| `ask` | 同上，此外还有 `output_file` 未获批准的请求：保存响应就是一次文件写入。 |

提示显示的是将要发出的请求，而不是其 JSON 参数：

```text
Upload the quarterly report

POST https://api.example.com/upload?v=2
Headers:
  Authorization: Bearer abc
  Content-Type: multipart/form-data; boundary=<generated>
  User-Agent: coddy-agent/1.0 (+https://github.com/EvilFreelancer/coddy-agent)
Body: multipart form, 20611 bytes
  doc: the file /home/me/project/report.pdf (20480 bytes)
  note = Q3
Proxy: http://10.0.0.2:3128
Saves the response body to /home/me/project/out/receipt.json

An always answer also approves the files it uploads, the proxy and the file it saves for requests to https://api.example.com.
```

![Web 界面中一次 http_request 上传的权限卡片](../../assets/http-requests/http-request-permission-dark-1280.png)

*Web 界面卡片：将要发出的请求、上传的文件，以及可为它能留下的每种授予各一个按钮。*

工具本会拒绝的参数仍然会询问，并在提示末尾给出调用将失败的原因。配置添加的请求头会与其他请求头一同列出，并在单独一行中指明（见[提示词显示的内容](#提示词显示的内容)）。

### 一次答复批准了什么

对话框提供四个选项：

- **Allow** - 仅此一次请求；
- **Always allow `https://api.example.com/upload`** - 本会话余下时间内对该地址的所有后续请求，无论其方法和查询参数（当地址是源的根时不提供）；
- **Always allow `https://api.example.com`** - 对该源上任何内容的所有后续请求；
- **Reject**。

批准一个目的地并不会批准之后发往它的请求所携带的内容，因为那些内容从未展示过。"always" 答复会记录该目的地连同本次请求携带的内容，后续请求只有在它携带的一切都已被记录时才被放行：

- 本地文件（`body_file`、`form_data` 文件部分）- 针对该源批准；同一文件发往另一个源，或另一个文件发到这里，都会重新询问；
- 代理 - 针对该源批准；另一个代理会重新询问；
- 未检查的证书（`verify_tls: false`）- 针对该源批准；
- `output_file` 路径 - 按该路径批准，与文件写入的 "Allow always" 方式相同。

授予保存在会话包（`permission_grants.json`）中，并在会话重启后保留；新会话开始时没有任何授予。

### 允许列表

`tools.http_request.allowlist` 指定从不需要询问的目的地：

```yaml
tools:
  http_request:
    allowlist:
      - api.github.com               # a host, any scheme and port
      - "*.internal.example.com"     # its subdomains, not the domain itself
      - localhost:8080               # a host and a port
      - http://127.0.0.1:3000        # an origin
      - https://api.example.com/v1/  # an address prefix
```

带路径的条目覆盖该地址及其下的一切：`https://api.example.com/v1` 覆盖 `/v1` 和 `/v1/items`，但不覆盖 `/v10`；`https://api.example.com/v1/` 只覆盖 `/v1/` 下的内容。`"*"` 允许所有目的地。无法匹配的条目 - 不支持的协议、凭据、查询参数、没有协议的路径、出现在最左侧标签以外的通配符 - 是配置错误，由 `coddy -t` 连同其索引报告。

允许列表中的目的地也覆盖发往它的请求所上传的文件和未检查的证书：该条目是操作者本人对该服务的信任声明。它不覆盖代理，代理是自身独立的目的地，需要自己的条目；也不覆盖 `ask` 下的 `output_file`，那遵循写入策略。

### 私有地址

与 `webfetch` 不同，`http_request` 不拒绝 localhost 或私有网络：抵达智能体正在构建的服务正是该工具存在的理由之一。权限门禁才是模型与这些地址之间的屏障，因此在 `bypass` 下，或允许列表中包含 `"*"` 时，模型可以抵达主机能抵达的任何东西 - 包括云元数据地址 `169.254.169.254`，与运行 `curl` 的 `run_command` 所能做到的相同。请把 `bypass` 留给可信环境，并按名称将无人值守运行所需的服务加入允许列表。

## 同一客户端上的 webfetch

`webfetch` 会为普通 `GET` 其 URL 构建与 `http_request` 相同的请求，并通过同一客户端发送，只是策略更窄：

- 地址会经过 SSRF 防护检查 - 不允许 localhost、`.local` 名称、私有地址、链路本地地址或元数据地址，URL 中不允许凭据 - 每个重定向在被跟随前也同样检查；
- 每个重定向都会被跟随，最多十次；
- 传输层请求 gzip 并解码；
- 非 2xx 状态是错误，页面上限为 4 MiB，响应会经过可读性处理转为 Markdown；
- 不发送[默认请求头](#默认请求头)：请求只携带工具自身的 `User-Agent`。

它不需要权限，因为除了 URL 之外，模型无法操控上述任何内容。

## 相关

[工具](../reference/tools.md) - 每个内置工具、其参数及其权限类别；
[安全与信任](../operate/security.md#权限模式与提示) - 权限模式以及各模式会询问什么；
[Web 搜索](web-search.md) - `websearch`，以及用于阅读页面的 `webfetch`；
[config.yaml 参考](../reference/config.md) - `tools.http_request.allowlist`、`tools.http_request.default_headers` 和 `tools.output_limits`。

<!-- docsgen:source sha256=7b786c2a610c9132 -->

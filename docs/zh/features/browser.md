# Coddy for Browser

Coddy for Browser 让你的 Coddy 的聊天在你自己的浏览器中工作 - Chrome、Yandex Browser 或 Firefox：打开标签页、读取页面、点击、输入、截图、在页面与工作区之间移动文件。模型在你的服务器上运行，[Coddy for Browser](https://github.com/coddy-project/coddy-for-browser) 扩展在你共享的标签页中执行其操作。扩展的侧边面板会在你查看的页面旁显示你 Coddy 的 Web 界面，任何其他界面也可以驱动浏览器：Web 界面、控制台、编辑器、Telegram 聊天、定时任务、另一台机器上你的 [swarm](../operate/swarm.md) 的一个节点。

## 工作原理

扩展连接到 `coddy serve` 的 HTTP 服务器，即其主服务器，并保持一个 WebSocket 打开。Coddy 将聊天的每个操作作为 JSON-RPC 请求通过该套接字发送下去，扩展则作出应答。服务器从不连接到浏览器，因此位于 NAT 之后笔记本电脑上的浏览器，与服务器本身的浏览器工作方式相同。另一个 Coddy 进程的聊天 - 同一台机器上的控制台或编辑器、集群的一个节点 - 通过主服务器的 HTTP API 到达浏览器。通信协议在 [浏览器协议](../reference/browser-protocol.md) 中描述。

![Coddy for Browser：扩展连接到其主服务器的套接字、该服务器的路由、其他进程的聊天，以及一个操作的路径](../../assets/diagrams/browser-flow.png)

*浏览器、其主服务器与其他进程的聊天如何连接，以及一个操作经过什么。*

## 设置服务器

扩展的端点与 `browser` 工具是 `browser` 构建标签的一个模块。发布二进制、Docker 镜像和软件包都包含它；未包含它构建的二进制会拒绝 `browser.enable` 并指出该标签。侧边面板显示 Web 界面，因此二进制还需要 `ui` 标签，同样的构建也包含它。

生成至少 32 个字符的密钥，例如使用 `openssl rand -hex 32`，将其保存在 `<home>/.env` 中作为 `CODDY_BROWSER_KEY`，并在 `config.yaml` 中开启该模块。

```yaml
browser:
  enable: true
  token: ${CODDY_BROWSER_KEY}
```

扩展连接到 HTTP 服务器（`httpserver.enable`），地址与 Web 界面使用的地址相同。从 **设置** 保存该块，或重启 `coddy serve`；新密钥会断开持有旧密钥的扩展。当 `browser.enable` 为 true 时，`browser` 工具会提供给聊天。

当你在设置中输入密钥时，Coddy 会在保存和重新加载时保留其中的美元符号。手写 YAML 中的环境引用（例如 `${CODDY_BROWSER_KEY}`）仍然有效，并且能在无关的设置保存后保留。如果环境展开已经更改了已保存的密钥，请在设置中重新输入原始密钥。

在浏览器所在的机器上运行 `coddy serve`。另一台机器上的服务器需要在其前面配置 HTTPS，一个会传递 WebSocket 升级的反向代理：扩展仅对 `localhost`、`127.0.0.1` 和 `[::1]` 使用普通 HTTP。不接受任何入站连接的机器使用 `swarm.join` 加入你的集群，其他节点通过中继到达其浏览器（[其他 Coddy 进程的聊天](#其他-coddy-进程的聊天)）。

该密钥独立于 `httpserver.auth_token`。它打开扩展的套接字及其文件传输，侧边面板的 Web 界面使用套接字分发的一次性票据登录，因此持有密钥的人拥有该服务器的 Web 界面。请像对待 HTTP API 的令牌一样对它保密。

## 安装扩展

扩展已发布到两个商店：

| 商店 | 浏览器 |
| --- | --- |
| [Chrome Web Store](https://chromewebstore.google.com/detail/coddy-for-browser/pdppgelbaehcedjpgfonemkmglpjifni) | Chrome 116 及更高版本、Yandex Browser、Edge 和其他 Chromium 浏览器 |
| [Firefox Add-ons](https://addons.mozilla.org/firefox/addon/coddy-for-browser/) | Firefox 139 及更高版本 |

Web 界面的 **设置 → 浏览器** 提供两者，首先是你当前所用浏览器的商店，旁边是该浏览器需要的密钥，因此连接一个浏览器的全部内容都在该页面上。

扩展的每次 [发布](https://github.com/coddy-project/coddy-for-browser/releases) 也附带其软件包，其 CI 的每次运行都会将它们保留为产物，用于商店尚未发布的版本或你自己的构建：

| 软件包 | 浏览器 |
| --- | --- |
| `coddy-for-browser-chrome-<version>.zip` | Chrome 116 及更高版本、Yandex Browser 和其他 Chromium 浏览器 |
| `coddy-for-browser-firefox-<version>.zip` | Firefox 139 及更高版本，未签名 |

- **Chrome**：在该版本发布到 Chrome Web Store 后从那里安装，或将归档解压到你保留的文件夹，打开 `chrome://extensions`，开启 **开发者模式**，按 **加载已解压的扩展程序** 并选择该文件夹。
- **Yandex Browser**：同样使用 Chrome Web Store，或使用同一归档并在 `browser://extensions` 上执行相同步骤。
- **Firefox**：在该版本发布到 addons.mozilla.org 后从那里安装。正式版 Firefox 只安装 Mozilla 签名的附加组件，因此未签名归档会从 `about:debugging#/runtime/this-firefox` 作为临时附加组件加载（**临时载入附加组件**），Firefox 关闭时会将其移除；Firefox Developer Edition 和 Nightly 在 `about:config` 中将 `xpinstall.signatures.required` 设为 `false` 后会保持安装。

若要从源码构建，请在扩展的检出目录中运行 `npm run build`，并加载 `dist/chrome` 或 `dist/firefox`。扩展的 README 中有每种浏览器的详细信息。

## 连接浏览器

Web 界面的 **设置 → 浏览器** 会引导整个过程：两个商店、地址和密钥，以及服务器看到的内容。按 **生成**，保存设置，卡片会等待扩展；一旦浏览器到达，它会列出它们，每个都按在其选项中给定的名称及其报告的版本列出。它读取 `GET /coddy/browser/setup`，其中不包含密钥和标签页。在那里生成的密钥是服务器自有字母表中 48 个字符，远超配置要求的 32 个；`openssl rand -hex 32` 会为手动配置的服务器生成一个。

对大多数人来说，连接浏览器就是本节的全部内容，因此 **向其他 Coddy 进程开放这些浏览器** 和 **其他服务器的浏览器** 折叠在卡片下方的 **在 Coddy 进程之间共享浏览器** 中。[其他 Coddy 进程的聊天](#其他-coddy-进程的聊天) 说明了这两个设置的作用。

扩展的设置页面会在安装后打开，之后可从侧边面板的设置按钮打开。输入服务器地址（其 Web 界面的地址，本地服务器为 `http://127.0.0.1:12345`）、密钥以及此浏览器的名称，然后按 **保存并连接**。

![扩展的设置页面，已连接到 Coddy](../../assets/browser/browser-settings-dark-1280.png)

*Coddy for Browser 的设置页面已连接到本地 Coddy。*

多个浏览器可以连接到一台服务器，每个都使用自己的名称。聊天会列出它们并选择一个；仅连接一个浏览器时，会直接使用它而不询问。

## 侧边面板

扩展的工具栏按钮会打开其侧边面板：主服务器的 Web 界面，包含 Web 界面的一切 - 模型、附件、批准模式、历史记录以及 Coddy 提出的问题。Firefox 在其侧栏中显示它。Yandex Browser 没有用于扩展的侧边面板，因此在那里，按钮会在浏览器窗口右边缘的独立窗口中打开同一页面。面板使用通过其套接字获得的一次性票据登录，因此无需在那里输入密码。

*Chrome 中侧边面板的录屏：被要求搜索时，聊天在 DuckDuckGo 中输入，在 go.dev 上打开 Go 发布历史并回答，扩展在访问每个新网站前询问。文件在仓库中为 [browser-panel.mp4](../../assets/video/browser-panel.mp4)。*

在发送消息之前，Web 界面会向面板询问你正在查看的标签页，面板会将该标签页移动到聊天的标签页组中。关于 "此页面" 的请求会在你面前的页面上生效，聊天中未指定标签页的 `browser` 调用会作用于它。

![Chrome 中侧边面板里有 Coddy 的聊天，旁边是它处理过的页面](../../assets/browser/browser-panel-chrome-dark-1280.png)

*Chrome 中的侧边面板：扩展的条带显示当前标签页和持有它的聊天，Web 界面位于其下。该标签页位于紫色的 Coddy 组中，工具栏中的扩展图标会打开面板。*

Web 界面上方的条带是扩展自己的。它显示当前标签页及其归属，带有 **共享** 和 **收回** 以及一个打开扩展设置的按钮，并采用 Web 界面的主题。连接状态显示在设置页面上；当没有连接时，面板会显示一张通往那里的卡片，而不是 Web 界面。面板中的聊天是普通的 Coddy 会话，会列在 Web 界面的所有位置。Web 界面切换到的其他环境的聊天不会获得标签页：该标签页位于面板自身服务器的浏览器中。

当在主服务器某个聊天持有的标签页上打开时，面板会显示该聊天。控制台、编辑器、定时任务或 API 在那里启动的轮次会像在任何聊天中一样流式传输到面板中，并且在你转到 Web 界面的其他地方后，条带中的聊天名称会再次显示它。另一个 Coddy 进程的聊天位于其自己的服务器上，因此面板会指出它，但无法显示它。

扩展的问题 - 网站、另一台机器的聊天 - 会出现在面板顶部、Web 界面之上，聊天会在其周围保持可见。

## 共享标签页

面板外的聊天可以到达你共享的标签页。在侧边面板的条带中按 **共享**，或将标签页拖入 **Coddy** 组；**收回** 会将其移出组。扩展设置页面上的 **断开连接** 会取消每个操作并将每个标签页移出组；标签页保持打开。Coddy 正在处理的页面上的停止按钮执行相同操作并保持连接。

## 从聊天中工作

用文字提出请求，例如 "打开 coddy-project/coddy-agent 的发布页面并总结最新发布"。聊天会调用 [`browser`](../reference/tools.md) 工具：

- `list` 显示聊天可到达的浏览器及其可使用的标签页，标记为 `yours` 或 `busy`（另一个聊天的）；
- `select` 在连接多个浏览器时为聊天记住一个浏览器；
- `open_tab`、`close_tab` 和 `release` 在聊天的组中打开标签页、关闭其一个标签页并将其标签页交回共享组；
- `snapshot` 将页面读取为文本以及最多 500 个带引用的元素，`screenshot` 为读取图像的模型附加标签页可见部分的图像；
- `navigate`、`click`、`type`、`press`、`hover`、`scroll` 和 `drag` 作用于页面。在 Chrome 和 Yandex Browser 中，点击和按键会经过浏览器的调试器，因此页面会将它们作为可信输入接收，与来自人的输入相同。Firefox 不向扩展提供调试器，因此在那里扩展会在页面自身中派发事件，调用结果会说明 `"trusted": false`；
- `upload_file` 和 `download_file` 在页面与工作区之间移动文件。

来自 `snapshot` 的引用会持续到页面更改，因此聊天在导航后会再次读取页面。`browser` 工具会像其他有副作用的工具一样请求权限：除非会话在 `bypass` 权限模式下运行，否则聊天会确认每次调用。

在网站上的第一个操作之前，扩展会在其自己的窗口中，或在侧边面板打开时在侧边面板中询问你：是为此聊天允许、始终允许，还是拒绝。答案留在浏览器中；页面无法回答它，Coddy 也无法回答。设置页面列出始终允许的网站并可撤销它们。

始终允许的网站列表以本机开头 - `localhost`、`127.0.0.1` 和 `[::1]`，任意端口 - 因此聊天可与本地开发服务器工作而无需询问；像任何其他网站一样在设置页面撤销条目。这些条目排除了 Coddy 服务器本身在其端口的每个本地名称下的情况：其 Web 界面是批准聊天自身请求的地方，因此聊天只有在你允许该地址后才会在那里操作。另一台机器的聊天会像任何其他网站一样询问本地页面。

设置页面还会在扩展询问之前接收地址：一个源（`https://example.com`）、主机的一个端口（`localhost:3000`）、一个端口范围（`10.0.0.5:8000-8100`）或每个端口（`example.com:*`）。那里的一个开关会为主服务器的聊天允许每个网站。该开关以及针对每个端口或范围的条目都不覆盖 Coddy 服务器；带其单个端口的条目会覆盖。

![扩展询问 Coddy 是否可以在某个网站上操作](../../assets/browser/browser-approval-dark-420.png)

*扩展在 Coddy 于新网站上操作之前提出的问题。*

当聊天在标签页中工作时，标签页会发光并显示停止按钮，聊天组的标题以沙漏开头。两者都会在聊天的轮次结束时消失。

![Coddy 正在处理的标签页，带有发光效果和停止按钮](../../assets/browser/browser-working-dark-1280.png)

*Coddy 在其中工作时的标签页。*

*Web 界面中一个聊天的录屏，该聊天位于其自己的窗口中，在旁边的 Chrome 中工作：扩展在其自己的窗口中询问网站，标签页在紫色的 Coddy 组中打开，在该标签页上打开的侧边面板会跟随同一聊天。文件在仓库中为 [browser-remote.mp4](../../assets/video/browser-remote.mp4)。*

## 其他 Coddy 进程的聊天

一个浏览器有一个主服务器，即其扩展连接到的服务器。在其它任何地方运行的聊天通过该服务器的 HTTP API（`/coddy/browser/v1`）并使用该服务器的普通 API 凭据到达它。主服务器需选择加入。

```yaml
browser:
  enable: true
  token: ${CODDY_BROWSER_KEY}
  serve_remote: true
```

- **同一台机器上的控制台或编辑器。** `coddy` 和 `coddy acp` 读取与它们旁边的 `coddy serve` 相同的 `config.yaml`，并且在 `serve_remote` 开启时，它们的 `browser` 工具会使用 `httpserver.auth_token` 到达位于其 `httpserver.host` 和 `port` 的服务器。使用其他 `-H`、`-P` 或 `--auth-token` 标志启动的服务器则改在 `browser.remotes` 中命名。
- **另一台机器。** 在运行聊天的进程的 `browser.remotes` 中列出主服务器：`httpserver.remotes` 的名称，或一个地址。集群的一个节点通过中继对它的挂载到达，并使用中继在 `httpserver.remotes` 中条目的令牌，如同 [`--remote`](../operate/remote.md)。

```yaml
httpserver:
  remotes:
    - name: relay
      url: https://relay.example:8443
      token: ${CODDY_RELAY_TOKEN}
browser:
  enable: true
  remotes:
    - https://relay.example:8443/swarm/nodes/home
```

浏览器所在的机器使用密钥运行 `coddy serve`，`serve_remote` 和 `swarm.join` 将其命名为 `home`；另一台机器的聊天随后会将其浏览器列为 `<browser id>@home` 并按该名称选择它。仅使用其他服务器浏览器的进程不需要自己的密钥。

浏览器旁的人做决定。通过 API 到来的聊天的第一个操作会在扩展中询问，并指出该聊天及其运行的机器：允许此聊天，或拒绝。网站的 "始终允许" 仅覆盖主服务器的聊天，因此来自另一台机器的聊天会询问每个新网站，侧边面板会在聊天旁指出该机器。停止和断开连接会收回授予来自其他机器聊天的一切。

![扩展询问另一个 Coddy 的聊天是否可以使用浏览器](../../assets/browser/browser-panel-remote-dark-420.png)

*在另一台机器上运行的聊天在其第一个操作之前询问。*

持有主服务器 API 凭据的人已经可以在那里运行轮次，也可以通过中继，因此该 API 不会增加超出该凭据所拥有权限的权限。主服务器无法看到远程聊天的轮次在哪里结束：其标签页的工作标记会在最后一次操作后 45 秒消失。

## 文件

`upload_file` 读取工作区中的文件（`path`，相对路径），并将其设置到 `params.ref` 指定的文件输入上。`download_file` 在浏览器中携带其 cookie 获取 `params.url`，并将字节写入工作区的新文件；已存在的路径会被拒绝。文件最大为 32 MiB。下载不会跟随重定向，因此聊天会传递最终地址。字节通过一次性票据在扩展与其主服务器之间移动，绝不通过扩展指定的服务器路径，而另一个进程的聊天会在其对主服务器的调用中携带它们。

## 限制

- 已测试 Chrome、Yandex Browser 和 Firefox。其他 Chromium 浏览器加载 Chrome 软件包；
- 仅页面的顶层框架：框架内的元素通过截图和坐标到达；
- Chrome 和 Yandex Browser 在扩展操作时会显示其 "已开始调试此浏览器" 栏；它在最后一次操作后 20 秒分离；
- 在 Firefox 中，页面可以区分 Coddy 的输入与人的输入，检查这一点的网站（某些编辑器和登录表单）会忽略它。那里的后退或前进会先移动，然后对新网站的下一个操作会询问它；
- 关闭的浏览器、重启的扩展或丢失的连接会使操作结果未知。Coddy 从不自行重复操作；聊天会先检查页面；
- 一台服务器的每个浏览器使用一个密钥：持有密钥的人可以用另一个浏览器的名称连接；
- 扩展不会为你输入密码、解决 CAPTCHA 或登录。

## 验证构建

```bash
go test -tags=browser ./external/browser/...
go test -tags=http,browser ./external/httpserver -run Browser
```

测试使用假扩展（`external/browser/browsertest`）通过真实 WebSocket 驱动真实端点。`features/browser.feature` 包含该工具的正常路径，其中包括通过主服务器 API 的另一个进程的聊天；`features/browser_panel.feature` 包含在真实智能体上使用脚本化模型的侧边面板 Web 界面。`TestLiveExtension` 在真实 Chromium 中针对端点和工具运行真实扩展：将 `CODDY_FOR_BROWSER_DIR` 指向扩展的已构建检出目录，并让 Node 能访问 Playwright。扩展自己的 CI 会在 Chromium 中运行其 Chrome 软件包，并在 Firefox 中针对服务器模拟运行其 Firefox 软件包。

```bash
CODDY_FOR_BROWSER_DIR=../coddy-for-browser go test -tags=browser ./external/browser/... -run TestLiveExtension -v
```

<!-- docsgen:source sha256=4d83e5a7657526e5 -->

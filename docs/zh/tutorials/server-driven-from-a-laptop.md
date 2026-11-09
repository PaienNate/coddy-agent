# 从笔记本电脑驱动服务器

在服务器上运行 `coddy serve`，并为其配置工作区和后端模型；笔记本电脑上只保留客户端。控制台、基于 ACP 的编辑器以及 Web 界面都连接同一台服务器，会话也保存在那里。完整指南见 [远程模式](../operate/remote.md)；下文各参数的来源见 [控制台](../surfaces/console.md)、"远程模式（`--remote`）"和 [身份验证](../reference/http-api.md#身份验证可选)。

1. **为服务器设置令牌，并绑定到非 loopback 地址。** Bearer 认证默认关闭。请通过环境变量或 `--auth-token` 设置令牌，而不是将其写入配置文件；在 `httpserver.auth_token` 中使用 `${ENV}` 引用同样可行。`host` 默认取 `127.0.0.1`，因为一个进程会启动所有已启用的子系统，所以仅仅为了接入机器人，不应顺带把 API 暴露到网络。`cors` 供第 4 步中的浏览器使用。

   ```yaml
   httpserver:
     host: "0.0.0.0"
     port: 12345
     auth_token: "${CODDY_HTTP_TOKEN}"
     cors:
       enable: true
       allow_loopback: true   # the laptop's UI is served by its own coddy serve: any loopback address, any port
   ```

   ```bash
   export CODDY_HTTP_TOKEN="a-long-random-string"   # or a line in ~/.coddy/.env
   coddy serve --dry-run    # binds the address once, names a port somebody else holds
   coddy serve --daemon     # foreground `coddy serve` under systemd or Docker
   ```

   启动横幅会说明哪些地址可达、以何种方式可达：`httpserver  http://0.0.0.0:12345  (bearer auth)`。在没有令牌的情况下绑定非 loopback 地址会在启动时记录一条警告；当客户端通过明文 http 向非 loopback 地址发送令牌时也会发出警告：当通信路径经过你无法控制的网络时，请在服务器前面加上 TLS，反向代理即可。

   笔记本电脑上不需要有任何可达的服务，因此处于 NAT 之后或地址会变化的笔记本电脑同样可用：控制台主动拨号服务器（第 2 步），浏览器从笔记本电脑自身的 `coddy serve` 拨号（第 4 步），而加入集群的笔记本电脑会在其 `swarm.join` 条目中省略 `advertise_url` 并拨号中继，由中继沿该连接把请求送回（[集群，两种传输方式](../operate/swarm.md#两种传输方式)）。`allow_loopback` 正是让第 4 步不受笔记本电脑的地址拼写和端口影响的原因：`http://localhost:12345`、`http://127.0.0.1:12345` 和 Vite 的 `http://localhost:5173` 是三个来源，而该开关会把它们全部放行。它的安全性仅取决于本步骤所设置的令牌（[安全与信任](../operate/security.md)）。

2. **从笔记本电脑的控制台驱动它。** `--remote` 接受裸 `host:port`（协议默认 http）、完整 URL，或笔记本电脑自身 `config.yaml` 中 `httpserver.remotes` 下所列远程项的名称。令牌依次取自 `--remote-token`、该远程项条目中的 `token`（如果存在），然后是 `CODDY_REMOTE_TOKEN`。

   ```bash
   export CODDY_REMOTE_TOKEN="a-long-random-string"
   coddy --dry-run --remote box.example:12345   # the target has to accept the token
   coddy --remote box.example:12345
   coddy -p "What changed in the last release?" --remote box.example:12345
   ```

   ```yaml
   httpserver:
     remotes:
       - name: "box"
         url: "https://box.example:12345"
   ```

   使用上面的条目后，只需 `coddy --remote box` 即可。该轮次在服务器的工作区中运行：工具框、思考过程和 token 计数会流式返回，权限与提问弹窗通过服务器作答，`/resume`、`-c` 和 `--session-id` 从服务器的会话列表中选择，横幅显示 `remote: <url>`。`--permission-mode` 和 `/permissions` 切换服务器的会话，此后在服务器上新建的会话都将以该模式启动。有一项被有意禁止远程使用：`!!` 本地 shell（因为工作区不在这台机器上）。

3. **从编辑器驱动它。** `coddy acp` 接受同样的两个参数，因此诸如 Zed 之类的 ACP 客户端可以将其智能体命令指向远程服务器；请把 `CODDY_REMOTE_TOKEN` 放入编辑器的环境变量中，或传入 `--remote-token`。

   ```bash
   coddy acp --remote box.example:12345
   ```

4. **从浏览器驱动它。** 环境项位于侧栏最下方：当页面驱动自身服务器时显示为笔记本电脑，当它驱动远程服务器时显示为两个相向的尖括号，并用一个圆点表示该主机是否有响应，其工具提示会显示当前使用的环境。其菜单列出提供该页面的 `coddy serve` 的远程项（`httpserver.remotes`）以及 `Connect to…`，后者接受名称、URL 和 bearer 令牌；选择某个条目会立即连接并重新加载，而在此处输入的令牌会留在浏览器中。`httpserver.remotes` 条目可以自带令牌（`token: "${BOX_TOKEN}"`），这样每个打开此页面的浏览器都会从配置中获取该令牌（[令牌](../operate/remote.md#令牌)）。远程服务器必须通过 `httpserver.cors` 允许该页面的来源，这正是第 1 步所设置的；当携带令牌的跨域 `GET /v1/models` 成功时，每个条目上的圆点会变绿，若为红色，则其下方的行会说明是服务器无响应、拒绝令牌，还是将此外源排除在 CORS 之外（[跨域访问与远程 UI](../reference/http-api.md#跨源访问与远程-ui)、[Web 界面](../surfaces/web-ui.md)）。直接打开服务器自身的地址则会加载无需令牌的 SPA 外壳，因为该外壳是公开的，页面会在调用 API 之前请求令牌。

5. **记住哪些内容留在服务器上。** 会话、信任凭据以及子智能体的子会话都归服务器所有，因此项目文件在服务器上审批：在服务器本身上使用 `coddy mcp trust`、`coddy hooks trust` 和 `coddy agents trust`，或通过 API 使用 `POST /coddy/mcp/{name}/trust`、`POST /coddy/hooks/trust` 和 `POST /coddy/subagents/{name}/trust`。连接断开后服务器上的该轮次仍会继续运行；`/resume` 会在其结束后显示结果。

<!-- docsgen:source sha256=249305851498f302 -->

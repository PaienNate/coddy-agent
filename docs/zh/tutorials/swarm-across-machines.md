# 跨机器的集群

**目标。** 多台机器上的工作节点，一个它们都加入的中继，以及你的笔记本电脑用自己的 Web 界面和控制台驱动其中任意一个：中继出现在笔记本电脑的环境菜单中，其智能体列在它下面，一键进入某个工作节点，整个集群的地图只需一次点击。除了中继的端口之外，其他都不必可达：工作节点主动向外拨号连接到它。

**环境。** 每台机器上都安装 Coddy 1.2.36 或更高版本（[安装](../getting-started/install.md)），并带有发布二进制所携带的 `swarm` 标签。每个工作节点都能访问的模型，其密钥放在该工作节点上：模型调用和工具都在工作节点所在之处运行。背景知识见 [集群](../operate/swarm.md) 和 [远程模式](../operate/remote.md)；在同一台机器上用 Docker 实现同样的形态见 [Docker 中的中继及其节点](swarm-relay-and-nodes.md)。

示例使用以下机器；请把你自己机器的主机名和地址替换进去：

| 角色 | 机器 | 其他机器使用的地址 |
|---|---|---|
| 中继 | `relay` | `relay.lan`，端口 12346 |
| 工作节点 | `gpu01` | 无需 |
| 工作节点 | `gpu02` | 无需 |
| 笔记本电脑 | 你的机器 | 无需 |

## 1. 三个令牌

中继由令牌保护，而不是由网络保护（[安全](../operate/swarm.md#安全性)）。在任意机器上生成三个令牌并留待下面的步骤使用：

```bash
openssl rand -hex 24   # client token: what the laptop presents to use the relay
openssl rand -hex 24   # pairing token: what a worker presents to join it
openssl rand -hex 24   # one worker token per worker: its own API's bearer
```

客户端令牌可以打开中继后面的每一个节点，因此要像对待整个机群的密码一样对待它。

## 2. 中继

在 `relay` 上，`~/.coddy/.env` 保存这些令牌（`chmod 600`）：

```bash
SWARM_CLIENT_TOKEN=<client token>
SWARM_PAIRING_TOKEN=<pairing token>
```

而 `~/.coddy/config.yaml` 只运行中继。`cors` 放行你打开笔记本电脑 Web 界面时的页面地址，因为该页面将在浏览器中与中继通信；`allow_loopback` 覆盖笔记本电脑自身 `coddy serve` 可能用来提供该页面的所有回环地址和端口：

```yaml
httpserver:
  enable: false

swarm:
  enable: true
  host: "0.0.0.0"
  port: 12346
  name: "office"
  auth_token: "${SWARM_CLIENT_TOKEN}"
  pairing_tokens: ["${SWARM_PAIRING_TOKEN}"]
  cors:
    enable: true
    allow_loopback: true   # http://localhost:12345, http://127.0.0.1:12345, any other loopback port
```

```bash
coddy serve --dry-run   # the address is free, the tokens and the file are right
coddy serve install     # a systemd user service; or `coddy serve` in a terminal
curl -s http://relay.lan:12346/swarm/info
```

`/swarm/info` 不需要令牌，通过 `node_count: 0` 输出 `"swarm": true`。来源会被精确比较，所以请列出你打开笔记本电脑页面的每一个地址；中继会自行拾取此文件的修改。

## 3. 工作节点

在每个工作节点上，`~/.coddy/.env`：

```bash
OPENAI_API_KEY=sk-...
SWARM_PAIRING_TOKEN=<pairing token>
NODE_TOKEN=<this worker's token>
```

以及 `~/.coddy/config.yaml` - 一个普通的 `coddy serve`，带有一个模型、用于其自身 API 的 bearer 令牌，以及一个 `swarm.join` 条目。这里没有 `advertise_url`，因此工作节点主动拨号连接中继，并沿着该连接被反向驱动；它不需要开放任何端口：

```yaml
providers:
  - name: openai
    type: openai
    api_key: "${OPENAI_API_KEY}"

models:
  - model: "openai/gpt-5.6-terra"

agent:
  model: "openai/gpt-5.6-terra"

httpserver:
  auth_token: "${NODE_TOKEN}"

swarm:
  join:
    - url: "http://relay.lan:12346"
      name: "gpu01"                     # gpu02 on the second worker
      pairing_token: "${SWARM_PAIRING_TOKEN}"
      token: "${NODE_TOKEN}"            # the same value as httpserver.auth_token
```

```bash
coddy serve --dry-run   # the relay is reachable from here
coddy serve install
```

中继的日志会为每个工作节点产生一行 `swarm node registered` 和一行 `swarm tunnel established`，并且在任意机器上：

```bash
curl -s -H "Authorization: Bearer <client token>" http://relay.lan:12346/swarm/nodes
```

会列出 `gpu01` 和 `gpu02`，附带 `"transport": "tunnel"` 和 `"online": true`。

## 4. 笔记本电脑

笔记本电脑为 Web 界面运行自己的 `coddy serve`（[快速开始](../getting-started/quickstart.md)），默认监听 `http://localhost:12345`。中继作为远程项加入其 `~/.coddy/config.yaml`，客户端令牌不写入文件：

```yaml
httpserver:
  remotes:
    - name: "office"
      url: "http://relay.lan:12346"
      token: "${CODDY_RELAY_TOKEN}"
```

```bash
echo 'CODDY_RELAY_TOKEN=<client token>' >> ~/.coddy/.env
```

`token` 那一行是可选的。加上它，环境菜单和 `coddy --remote office` 就会自动使用令牌而不再询问，并且每个打开这台笔记本电脑页面的浏览器都能从配置中获得该令牌；不加它，则需要在每个浏览器的 **Connect to…** 中手动输入一次令牌（[令牌](../operate/remote.md#令牌)）。让智能体添加远程项同样可行：它会暂存相同的条目并提交，菜单在下次打开时就会显示新的远程项。

![笔记本电脑的环境菜单，显示中继及其工作节点](../../assets/swarm/across-machines-env-menu-dark-1280.png)

*笔记本电脑环境菜单中的中继：绿色，其工作节点列在下方*

从侧栏底部打开环境菜单。`office` 显示为绿点并标注 `relay · 2 agents`，其下方是 `gpu01` 和 `gpu02`。点击 `gpu01`：页面会经由中继重新加载到该工作节点上 - 其会话出现在 History 中，其文件夹和模型出现在输入框中，其配置出现在 Settings 中。你启动的轮次就在 `gpu01` 上运行。

侧栏中的 **Swarm** 条目会在不离开该工作节点的情况下打开中继的地图。你的笔记本电脑位于顶部，以其主机名标注，中继在其下，工作节点再在其下；你当前所在的工作节点被圈出。点击 `gpu02` 即可切换到那里 - 地图保持打开，改为圈出 `gpu02`，页面不会重新加载 - 点击笔记本电脑可返回 Local，点击中继则可连接到中继本身。

![在工作节点上方打开的集群地图](../../assets/swarm/map-over-node-dark-1280.png)

*在 `gpu01` 上方的地图：连接出发的笔记本电脑、中继、两个工作节点*

在中继本身上，其 Settings 编辑的是中继自身的部署配置 - 其中也包括 CORS 来源 - 保存会立即生效（[中继自身的 Settings](../operate/swarm.md#设置位于何处)）。

## 5. 控制台

笔记本电脑的控制台通过中继的挂载点到达工作节点。`httpserver.remotes` 中的中继条目携带令牌，挂载在该中继下的节点会沿用同一个令牌，因此无需任何标志：

```bash
coddy --dry-run --remote http://relay.lan:12346/swarm/nodes/gpu01
coddy --remote http://relay.lan:12346/swarm/nodes/gpu01
coddy -p "What is in the workspace?" --remote http://relay.lan:12346/swarm/nodes/gpu02
```

`coddy --dry-run --remote office` 会显示 `office` 是一个中继，并打印其智能体的挂载点，因为中继自身不驱动任何东西，而控制台需要一个节点。`coddy acp` 为编辑器接收相同的 `--remote`（[使用远程节点](swarm-remote-nodes.md)）。

## 常见故障

- **中继的圆点是红色，并说响应被 CORS 拦截。** 中继未放行笔记本电脑页面所在的地址。中继下方的那一行会指出该确切来源，对于位于回环地址上的页面，还会给出那个开关：在中继上、在其文件中或在其自身的 Settings 中设置 `swarm.cors.allow_loopback: true`，即可放行任意端口上的所有回环来源；或者把确切来源加入 `swarm.cors.allowed_origins`，注意 `localhost` 和 `127.0.0.1` 是两个不同的来源。
- **圆点是红色，并说中继拒绝该令牌。** 笔记本电脑所提供的令牌不是中继的 `swarm.auth_token`（或 `--swarm-auth-token`）。如果笔记本电脑的条目中有 `token`，就在那里修正；如果没有，中继下方的 **Enter token** 会打开已为其填好内容的表单。
- **圆点是红色，并说中继无响应。** 从笔记本电脑看，该地址上没有监听：请从笔记本电脑检查 `curl http://relay.lan:12346/swarm/info`、中继的 `host`，以及两者之间的防火墙。
- **中继是绿色，但没有列出任何工作节点。** 没有工作节点加入：工作节点的日志会说明其注册是否被拒绝。配对令牌是精确比较的。
- **列出了工作节点，但在那里的轮次以 401 失败。** 工作节点 `swarm.join` 条目中的 `token` 不是它自己的 `httpserver.auth_token`；中继在代理时会出示该值。
- **`coddy --remote office` 毫无进展。** `office` 是中继，它不提供任何会话服务。请把 `--remote` 指向某个工作节点的挂载点；试运行会列出它们。
- **添加到笔记本电脑配置中的远程项没有出现在菜单里。** 该文件在菜单打开时才会被读取，所以请关闭菜单再重新打开；笔记本电脑的服务器必须已加载该文件，而它在编辑后几秒内会自行完成加载。
- **保存中继设置后，工作节点会短暂消失。** 保存会重建中继，工作节点会在几秒内重新注册并重新打开其隧道；更换新的监听地址则需要重启中继。

<!-- docsgen:source sha256=8d0c8f9eeb80dcbb -->

# 作为集群节点的 Android 手机

https://github.com/user-attachments/assets/e7786835-0676-47be-8b42-477630dfa998

*一段关于 1.3.0 的两分钟录屏：Android 手机上 Termux 中的 Coddy 加入一个中继，笔记本电脑从其环境菜单连接到该中继，手机出现在地图画布上，运行 `getprop` 并写入一条笔记，Files 窗口通过中继打开手机的工作区。该文件在仓库中名为 [android-swarm.mp4](../../assets/video/android-swarm.mp4)。*

**目标。** 一部在 Termux 中运行 Coddy 的手机加入你在笔记本电脑或服务器上运行的中继，然后你像对待任何其他节点一样，从中继的 Web 界面或控制台驱动它：提示词、在手机上运行的工具、权限提示、手机自己的会话。手机自己打开连接，因此不需要公网地址、不需要开放端口，也不需要 VPN；位于运营商 NAT 之后的移动网络手机，与连接你 Wi-Fi 的手机以相同方式加入。

**环境。** 一台用于运行中继的机器，装有 Coddy 1.2.31 或更高版本：一台连接同一 Wi-Fi 的笔记本电脑，或一台具有公网地址的服务器。一部装有 [Termux](https://termux.dev) 的 Android 手机，64 位，Android 7.0 或更高版本。一个手机可访问的模型及其密钥：手机自己执行模型调用和工具，中继只负责承载流量。背景信息见 [Android（Termux）](../getting-started/android.md) 和[集群](../operate/swarm.md)；带有节点的中继的结构见 [Docker 中的中继及其节点](swarm-relay-and-nodes.md)。

## 1. 中继

两个令牌决定谁可以使用中继以及谁可以加入中继（[安全](../operate/swarm.md#安全性)）。在中继所在的机器上生成它们：

```bash
mkdir -p ~/.coddy
printf 'SWARM_CLIENT_TOKEN=%s\nSWARM_PAIRING_TOKEN=%s\n' "$(openssl rand -hex 16)" "$(openssl rand -hex 16)" >> ~/.coddy/.env
chmod 600 ~/.coddy/.env
```

中继的 `~/.coddy/config.yaml` 只运行中继：

```yaml
httpserver:
  enable: false

swarm:
  enable: true
  host: "0.0.0.0"
  port: 12346
  name: "home"
  auth_token: "${SWARM_CLIENT_TOKEN}"
  pairing_tokens: ["${SWARM_PAIRING_TOKEN}"]
```

```bash
coddy serve
```

中继监听机器的所有地址，因此手机可通过该机器在 Wi-Fi 上的地址访问它，例如 `http://192.168.1.10:12346`。在手机的浏览器中，`http://192.168.1.10:12346/swarm/info` 无需令牌即可响应，并表明中继已启动。

如果中继位于手机可通过互联网访问的服务器上，则需要证书（`swarm.tls`，[加密与代理](../operate/swarm.md#加密与代理)），并且手机通过 `https://...` 加入：令牌在请求头中传输，而移动网络可能会让明文 HTTP 经过会重新封装它的代理，隧道无法在这种情况下存活（[两种传输](../operate/swarm.md#两种传输方式)）。

## 2. Termux 中的 Coddy

在手机上，在 Termux 中：

```bash
pkg install curl tmux
curl -fsSL https://coddy.dev/install.sh | bash
source ~/.bashrc
coddy -v
```

该脚本会获取适用于手机处理器的 Android 构建（[安装](../getting-started/android.md#安装)）。

## 3. 手机的配置

手机是一个普通的 `coddy serve`，带有一个模型、一个用于其自身 API 的令牌，以及一个 `swarm.join` 条目。没有 `advertise_url`，因此手机拨号到中继，中继再沿着该连接反向驱动它（隧道传输）。在 Termux 中 `~/.coddy/config.yaml`：

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
  enable: true
  auth_token: "${NODE_TOKEN}"

swarm:
  join:
    - url: "http://192.168.1.10:12346"
      name: "pixel"
      pairing_token: "${SWARM_PAIRING_TOKEN}"
      token: "${NODE_TOKEN}"
```

密钥放入与其相邻的 `~/.coddy/.env`：模型密钥、中继的配对令牌，以及手机自己的一个令牌，中继在向手机转发请求时会出示该令牌。

```bash
cat >> ~/.coddy/.env <<'EOF'
OPENAI_API_KEY=sk-...
SWARM_PAIRING_TOKEN=<the relay's pairing token>
NODE_TOKEN=<a token for this phone>
EOF
chmod 600 ~/.coddy/.env
```

来自[配置](../getting-started/configuration.md)的任何提供商都以相同方式工作。`name` 会成为手机在中继上的路径，因此请给每部手机各自的路径。

在启动任何内容之前，试运行会检查文件、模型和中继：

```bash
coddy serve --dry-run
```

## 4. 启动并保持运行

```bash
termux-wake-lock
tmux new -s coddy 'coddy serve'
```

用 **Ctrl+B** 分离，然后按 **D**；`tmux attach -t coddy` 会把日志带回来。`coddy serve --daemon` 是不使用 tmux 的替代方案（[coddy serve](../operate/serve.md)）。

启动后的几秒内，手机的日志会显示带有 `transport=tunnel` 的 `swarm: joining relay`，中继的日志会显示 `swarm node registered`，以及针对 `pixel` 的 `swarm tunnel established`。在中继所在的机器上：

```bash
export T=<the relay's client token>
curl -s -H "Authorization: Bearer $T" http://127.0.0.1:12346/swarm/nodes
```

列表中会列出带有 `"transport": "tunnel"` 和 `"online": true` 的 `pixel`。

Android 会停止它认为空闲的应用的进程。唤醒锁可让 Termux 在屏幕关闭时继续运行；还要在电池设置中让 Termux 不受限制运行，并在 Android 12 及更高版本上解除 Termux 文档中描述的后台进程限制（[运行 coddy serve](../getting-started/android.md#运行-coddy-serve)）。

## 5. 驱动手机

**浏览器。** 打开中继，`http://192.168.1.10:12346/`。页面提示中继需要一个令牌：从侧栏底部打开环境菜单，选择 **连接到…**，为它提供一个名称、中继的地址和客户端令牌，然后点击 **连接**。集群地图会在中继下显示 `pixel`，并将其标记为向外拨号的节点。

![以手机作为向外拨号节点的中继集群地图](../../assets/swarm/android-node-map-dark-1280.png)

*中继 `home`，其下是手机 `pixel`；虚线是手机打开的连接。*

点击 `pixel`，历史记录抽屉、输入框和设置便都属于手机：环境的工具提示显示 `pixel`，文件夹是 Termux 主目录。在那里开始的轮次在手机上运行：它的模型调用、它在 Termux 的 `bash` 中执行的命令、它的文件。权限提示会在浏览器中发给你。

![手机上的命令正在浏览器中等待批准](../../assets/swarm/android-node-permission-dark-1280.png)

*当被问及在哪个设备上运行时，智能体想要在手机上 Termux 的 `bash` 中运行 `getprop`，而批准请求会在浏览器中发出。*

**控制台。** 手机的挂载点像任何其他远程节点一样是一个基础 URL（[使用远程节点](swarm-remote-nodes.md)）：

```bash
coddy --remote http://192.168.1.10:12346/swarm/nodes/pixel --remote-token "$T"
coddy -p "Which Android version is this? Use getprop." --remote http://192.168.1.10:12346/swarm/nodes/pixel --remote-token "$T"
```

手机的工作区是启动 `coddy serve` 所在的 Termux 主目录。手机的共享存储（下载内容、照片）在 `termux-setup-storage` 之后变得可访问，该操作会向 Android 请求一次权限；之后会话的文件夹可以指向 `~/storage/shared` 之下。

## 常见问题

- **中继在启动时退出，并给出关于在没有客户端令牌的情况下绑定非环回地址的消息。** `SWARM_CLIENT_TOKEN` 没有到达它：该文件是运行中继的账户的 `~/.coddy/.env`。
- **手机始终不出现。** 它的配对令牌不是中继的 `pairing_tokens` 之一，或者手机无法访问该地址：网络正常时，Termux 中的 `curl http://192.168.1.10:12346/swarm/info` 会响应。访客 Wi-Fi 经常会把设备隔开。
- **屏幕关闭时手机会掉线。** 缺少唤醒锁，或电池设置仍限制 Termux；中继会在节点的租约到期后将其标记为离线，而进程再次运行时它会自行恢复。
- **通过手机挂载点发出的请求返回 401。** `swarm.join` 中的 `token` 不是手机的 `httpserver.auth_token`；正因如此，两者在上面的配置中都读取 `NODE_TOKEN`。
- **在移动网络上，`swarm tunnel established` 永远不会跟随 `swarm node registered`。** 手机和中继之间的某个东西会重新封装 HTTP。为中继配置 TLS，并加入 `https://...`。
- **两部手机声明同一个名称。** 只有第一部能保留它：中继通过每个租约的密钥来证明名称，而不是通过共享的配对令牌。请给每部手机起不同的名称。

<!-- docsgen:source sha256=f0785dadbfa44806 -->

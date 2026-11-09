# 使用远程节点

**目标。** 从控制台、编辑器和浏览器驱动一个位于中继之后的节点，就像驱动本地的 `coddy serve` 一样：提示词、权限提示、会话、模型选择器。并了解什么留在节点上、什么会传输，以及哪个凭据用于打开什么。

**环境。** 一个至少有一个节点的中继，例如 [Docker 中的中继及其节点](swarm-relay-and-nodes.md) 的演示环境，位于 `http://127.0.0.1:12346`，客户端令牌在 `$T` 中。通用远程指南是 [远程模式](../operate/remote.md)；本页是其中与集群相关的部分。

## 1. 什么是挂载

中继将每个节点挂载在 `/swarm/nodes/<name>/` 下，一个挂载承载节点的整个 API：`/v1/*`、`/coddy/*`、只读的集群路由，以及后续的 `/swarm/nodes/*` 跳。它是一个带路径前缀的普通基础 URL，而这正是 `--remote` 所接受的，因此每个已经能与 `coddy serve` 通信的客户端都能无需改动地通过中继与节点通信。中继使用节点自己的令牌代表你向节点进行身份验证；你只持有中继的客户端令牌。

## 2. 控制台

```bash
export T=change-me-client
coddy --remote http://127.0.0.1:12346/swarm/nodes/node-a --remote-token "$T"
```

启动横幅显示 `remote: <url>`；会话、其工作目录和模型目录属于 `node-a`。`-p` 用于一次性回答，`-c` 用于继续该节点的最新会话，`/resume` 作用于节点的会话列表：

```bash
coddy -p "List the files in the workspace." --remote http://127.0.0.1:12346/swarm/nodes/node-a --remote-token "$T"
coddy -c --remote http://127.0.0.1:12346/swarm/nodes/node-a --remote-token "$T"
```

位于第二个中继之后的节点只是一条更长的路径，仅此而已：`--remote http://127.0.0.1:12346/swarm/nodes/inner/swarm/nodes/node-c`（[中继链](swarm-multi-hop.md)）。

在输入任何内容之前，试运行会说明路径是否正确以及节点是否有响应：

```bash
coddy --dry-run --remote http://127.0.0.1:12346/swarm/nodes/node-a --remote-token "$T"
```

令牌也可以来自 `CODDY_REMOTE_TOKEN`，你经常使用的中继地址可以在自己的 `config.yaml` 中通过 `httpserver.remotes` 命名一次，然后按名称访问。该条目也可以携带客户端令牌，最好作为 `${ENV}` 引用，然后该中继下的节点挂载无需标志即可使用它（[命名远程](../operate/remote.md#命名远程)、[令牌](../operate/remote.md#令牌)）。

## 3. 编辑器

`coddy acp` 接受同样的两个标志，因此 Zed、VS Code、Obsidian 或脚本可以从编辑器的输入框驱动节点，使用节点的模式、模型、权限提示和技能：

```bash
coddy acp --remote http://127.0.0.1:12346/swarm/nodes/node-a --remote-token "$T"
```

将编辑器配置为运行该命令行，而不是裸的 `coddy acp`（[编辑器](../surfaces/editors.md)）。

## 4. 浏览器

打开中继本身，`http://127.0.0.1:12346/`。使用 `ui` 标签构建的中继（已发布的镜像就是如此）会在自己的地址上提供 Web 界面；它会要求客户端令牌，并打开集群地图。点击一个节点：历史记录抽屉列出该节点的会话，输入框显示其工作目录和模型，在那里启动的一轮会在该节点上运行。**Swarm** 条目在你进入某个节点后仍留在侧栏中，因此另一个节点只需点击一次，地图上的每个节点都会显示它持有多少会话、有多少轮正在运行，以及其中是否有一轮正在等待你。

从由其他 `coddy serve` 提供的 Web 界面（例如笔记本电脑上的那个）来看，中继只是另一个环境：该服务器的 `httpserver.remotes` 中的一个条目，或侧栏底部的环境项 **Connect to…**：中继的地址和客户端令牌。中继必须通过 `swarm.cors` 接纳该页面的来源：在 `allowed_origins` 中精确接纳，或对回环地址上的页面使用 `allow_loopback: true`，无论笔记本电脑的 `coddy serve` 占用了哪个端口。一旦它接受令牌，环境菜单就会列出其智能体，因此一个节点只需点击一次；[跨机器的集群](swarm-across-machines.md) 从零开始演示该设置。

![通过两个中继转发的权限提示](../../assets/swarm/permission-two-relays-dark-1280.png)

*来自位于两个中继之后的节点的权限提示，在浏览器中作答*

## 5. 什么在哪里运行

模型调用、工具、工作区、MCP 服务器、钩子和子智能体都在节点上运行，使用节点的提供商和密钥。客户端渲染对话记录并回答提示。节点提出的权限请求或问题会经过每一跳传回启动该轮次的客户端，并在那里作答，无论是在控制台、编辑器还是浏览器中。

信任在节点上决定。随节点上的检出一起到达的项目 MCP 服务器、钩子文件或子智能体定义会在该主机上获得批准，使用那里的 CLI（`coddy hooks trust ...`）或通过挂载使用令牌（`POST /swarm/nodes/node-a/coddy/hooks/trust`）；本地子命令不接受 `--remote`（[安全与信任](../operate/security.md)）。

## 6. 凭据与加密

一个凭据打开整个集群：任何持有中继客户端令牌的人都能控制该中继可到达的每个节点，并传递性地经过每一跳。此版本中没有按节点划分的客户端权限，因此客户端令牌是一扇覆盖整个集群的门，并应如此处理：按中继铸造，通过用新值重启中继来轮换，并且绝不在私有网络之外通过明文 HTTP 发送。

当 `swarm.tls` 指定证书和密钥时，中继会使用 TLS（[加密与代理](../operate/swarm.md#加密与代理)）；节点从不需要自己的 TLS，因为它只通过通往中继的隧道被访问。没有 TLS 时，通过 SSH 隧道访问中继，并将地址保持在回环上：

```bash
ssh -N -L 12346:127.0.0.1:12346 relay-host      # in one terminal
coddy --remote http://127.0.0.1:12346/swarm/nodes/node-a --remote-token "$T"
```

## 容易出错的地方

- **`--remote` 在连接前被拒绝。** 目标必须是裸源加路径前缀：没有查询字符串、没有片段、URL 中没有凭据。令牌放入 `--remote-token`、远程条目的 `token` 或 `CODDY_REMOTE_TOKEN`。
- **每个请求都出现 `401`。** 在应当使用中继客户端令牌的地方使用了节点自己的令牌。通过挂载时，你出示中继的令牌；中继出示节点的令牌。
- **浏览器中的状态点显示为红色，但令牌正确。** Web 界面是从另一个 `coddy serve` 打开的，而该中继不接纳页面的来源。改为打开中继自己的地址，当页面位于回环地址时设置 `swarm.cors.allow_loopback: true`，或在 `swarm.cors.allowed_origins` 中列出确切的来源。
- **从笔记本电脑开启的绕过会保持开启。** `/permissions bypass` 会更改该节点的会话，对它的所有客户端都如此，并且会随该会话保留，即使节点重启也如此。无人值守运行结束后，用 `/permissions ask` 将其切回；每个会话的默认值应放在节点的 `config.yaml` 中。
- **笔记本电脑断网后，对话记录在轮次中途停止。** 该轮次会在节点上继续运行；它结束后，`/resume` 或 `-c` 会显示结果，而对节点已经撤回的提示所作的回答会被忽略。
- **第二个客户端在同一会话上得到 `409`。** 一个会话一次只能进行一轮；第一个客户端持有锁，直到其轮次结束。

<!-- docsgen:source sha256=7ad332ed905cc47f -->

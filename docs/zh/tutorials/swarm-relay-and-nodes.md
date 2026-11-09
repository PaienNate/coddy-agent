# Docker 中的中继和节点

**目标。** 在容器中运行多个 Coddy 节点，在它们前面放一个中继，在一个地方列出集群中的每个会话，并从你的笔记本电脑通往任意节点。除中继的端口外不暴露任何东西：节点主动向外拨号连接到中继，因此它们不需要发布端口，也不需要入站路由。

**环境。** 在一台机器上使用带 Compose v2 的 Docker(一旦 `RELAY_URL` 指向真实地址，同样的文件可跨机器使用)、已发布的镜像 `ghcr.io/coddy-project/coddy-agent`(它带有 `swarm` 标签)，以及节点能够访问的模型。示例为节点提供一个 OpenAI 密钥；来自 [Configuration](../getting-started/configuration.md) 的任何提供商都以相同方式工作，包括本地 OpenAI 兼容服务器。背景见 [Swarm](../operate/swarm.md)；本页是运行一个集群的最短路径。

## 1. 文件

一个文件夹中的四个文件。令牌是中继的整个安全模型([Security](../operate/swarm.md#安全性))，因此在任何内容离开你的机器之前，请替换 `.env` 中的三个值。

`.env`:

```bash
SWARM_CLIENT_TOKEN=change-me-client      # what a person or a client presents to the relay
SWARM_PAIRING_TOKEN=change-me-pairing    # what a node presents to register
NODE_A_TOKEN=change-me-node-a            # each node's own bearer, handed to the relay on join
NODE_B_TOKEN=change-me-node-b
WORKER_TOKEN=change-me-worker
OPENAI_API_KEY=sk-...
```

`relay.yaml`，中继的配置。它除了中继之外不运行任何东西：

```yaml
httpserver:
  enable: false

swarm:
  enable: true
  host: "0.0.0.0"
  port: 12346
  name: "outer"
  auth_token: "${SWARM_CLIENT_TOKEN}"
  pairing_tokens: ["${SWARM_PAIRING_TOKEN}"]
```

`node.yaml`，由每个节点共享。节点是一个普通的 `coddy serve`，带有模型、其自身 API 上的 bearer 令牌，以及一个 `swarm.join` 条目。没有 `advertise_url`，因此节点自行打开到中继的连接，并沿该连接被驱动(隧道传输):

```yaml
providers:
  - name: openai
    type: openai
    api_key: "${OPENAI_API_KEY}"

models:
  - model: "openai/gpt-5.6-terra"
    max_tokens: 8192

agent:
  model: "openai/gpt-5.6-terra"

httpserver:
  enable: true
  auth_token: "${NODE_TOKEN}"

swarm:
  join:
    - url: "${RELAY_URL}"
      name: "${NODE_NAME}"
      pairing_token: "${SWARM_PAIRING_TOKEN}"
      token: "${NODE_TOKEN}"
```

`docker-compose.yml`:

```yaml
x-node: &node
  image: ghcr.io/coddy-project/coddy-agent:latest
  command: serve -H 0.0.0.0 -P 12345
  working_dir: /workspace
  depends_on: [relay]

services:
  relay:
    image: ghcr.io/coddy-project/coddy-agent:latest
    command: serve
    environment:
      CODDY_HOME: /home/user/.coddy
      CODDY_CONFIG: /home/user/.coddy/config.yaml
      SWARM_CLIENT_TOKEN: ${SWARM_CLIENT_TOKEN}
      SWARM_PAIRING_TOKEN: ${SWARM_PAIRING_TOKEN}
    ports: ["12346:12346"]
    volumes:
      - ./relay.yaml:/home/user/.coddy/config.yaml:ro
      - relay_home:/home/user/.coddy

  node-a:
    <<: *node
    environment:
      CODDY_HOME: /home/user/.coddy
      CODDY_CONFIG: /home/user/.coddy/config.yaml
      CODDY_CWD: /workspace
      RELAY_URL: http://relay:12346
      NODE_NAME: node-a
      NODE_TOKEN: ${NODE_A_TOKEN}
      SWARM_PAIRING_TOKEN: ${SWARM_PAIRING_TOKEN}
      OPENAI_API_KEY: ${OPENAI_API_KEY-}
    volumes:
      - ./node.yaml:/home/user/.coddy/config.yaml:ro
      - ./workspace/node-a:/workspace
      - node_a_home:/home/user/.coddy

  node-b:
    <<: *node
    environment:
      CODDY_HOME: /home/user/.coddy
      CODDY_CONFIG: /home/user/.coddy/config.yaml
      CODDY_CWD: /workspace
      RELAY_URL: http://relay:12346
      NODE_NAME: node-b
      NODE_TOKEN: ${NODE_B_TOKEN}
      SWARM_PAIRING_TOKEN: ${SWARM_PAIRING_TOKEN}
      OPENAI_API_KEY: ${OPENAI_API_KEY-}
    volumes:
      - ./node.yaml:/home/user/.coddy/config.yaml:ro
      - ./workspace/node-b:/workspace
      - node_b_home:/home/user/.coddy

volumes:
  relay_home:
  node_a_home:
  node_b_home:
```

有两个细节很重要。配置文件以只读方式挂载在用于主目录的命名卷之上，因为主目录是节点保存其会话以及按租约生成的密钥的地方，该密钥让它在重启后重新认领自己的名称。而节点不发布任何端口：一切都通过中继到达它们。

## 2. 启动它

```bash
mkdir -p workspace/node-a workspace/node-b
docker compose up -d
docker compose logs relay
```

中继日志在每个节点 `serve` 启动后的几秒内，会以每个节点一行 `swarm node registered` 和一行 `swarm tunnel established` 结束。

## 3. 检查它

`/swarm/info` 是开放的，中继上的其他一切都要求客户端令牌：

```bash
export T=change-me-client
curl -s http://127.0.0.1:12346/swarm/info
curl -s -H "Authorization: Bearer $T" http://127.0.0.1:12346/swarm/nodes
curl -s -H "Authorization: Bearer $T" http://127.0.0.1:12346/swarm/topology
```

`node_count` 为 2，节点列表列出 `node-a` 和 `node-b`，其 `"transport": "tunnel"` 和 `"online": true`，拓扑图将二者绘制在中继之下。

节点挂载在 `/swarm/nodes/<name>/` 之下，因此节点自身的 API 只差一个路径前缀，普通客户端无需改动即可工作：

```bash
curl -s -H "Authorization: Bearer $T" http://127.0.0.1:12346/swarm/nodes/node-a/v1/models
coddy --dry-run --remote http://127.0.0.1:12346/swarm/nodes/node-a --remote-token "$T"
coddy -p "Say hello in one line." --remote http://127.0.0.1:12346/swarm/nodes/node-a --remote-token "$T"
```

试运行会报告节点提供的模型，一次性提示词由 `node-a` 的模型作答。之后，聚合列表会显示该会话及其路由：

```bash
curl -s -H "Authorization: Bearer $T" 'http://127.0.0.1:12346/swarm/sessions'
```

每一行在会话 id 旁边带有 `node_path`(此处为 `["node-a"]`)，因为 id 是按节点选择的，确实会冲突。

在浏览器中，`http://localhost:12346/` 是中继自身的 Web 界面：它要求输入客户端令牌，然后打开到带有两个节点的集群地图。点击一个节点，History 抽屉、输入框和设置都属于该节点，中继在中间而没有任何一方感知到它([The UI](../operate/swarm.md#界面))。

## 4. 更多节点

没有 `name` 的节点以其主机名注册，在 Compose 中主机名就是容器 id，因此没有 `NODE_NAME` 的服务可以扩缩：

```yaml
  worker:
    <<: *node
    environment:
      CODDY_HOME: /home/user/.coddy
      CODDY_CONFIG: /home/user/.coddy/config.yaml
      CODDY_CWD: /workspace
      RELAY_URL: http://relay:12346
      NODE_TOKEN: ${WORKER_TOKEN}
      SWARM_PAIRING_TOKEN: ${SWARM_PAIRING_TOKEN}
      OPENAI_API_KEY: ${OPENAI_API_KEY-}
    volumes:
      - ./worker.yaml:/home/user/.coddy/config.yaml:ro
      - ./workspace/workers:/workspace
```

`worker.yaml` 就是去掉 `name:` 行的 `node.yaml`。然后：

```bash
docker compose up -d --scale worker=3
curl -s -H "Authorization: Bearer $T" http://127.0.0.1:12346/swarm/nodes
```

又出现三个智能体，以其容器 id 命名。共享令牌和工作区的 worker 是可互换的，这正是扩缩服务的目的；你按名称寻址的节点则保留自己的服务。

另一台机器上的节点是同样的 `node.yaml`，只需将 `RELAY_URL` 指向中继的公共地址，在该机器上无需开放任何东西：隧道是从它这一侧打开的。

## 常见出错情况

- **中继在启动时退出，消息提到在没有客户端令牌的情况下绑定到非回环地址。** `SWARM_CLIENT_TOKEN` 为空。位于 `0.0.0.0` 上的中继拒绝以开放方式运行；`swarm.allow_insecure` 会为实验环境覆盖这一点，仅此而已。
- **某个节点始终不出现。** 它的配对令牌与中继的某个 `pairing_tokens` 不匹配；节点日志显示注册被拒绝。令牌是精确比较的，包括空格。
- **节点出现了，但通过其挂载发出的请求返回 401。** 节点 `swarm.join` 条目中的 `token` 不是节点自身的 `httpserver.auth_token`。中继在代理时会出示该令牌，因此两个值必须相同。
- **`swarm tunnel established` 始终跟不上 `swarm node registered`。** 中继前面的某个东西重新分帧了 HTTP：一个七层代理或一个仅支持 HTTP/2 的终止器。隧道需要原始的端到端连接([Two transports](../operate/swarm.md#两种传输方式))；请将中继放在 TCP 透传之后，或用 `swarm.tls` 给它自己的 TLS。
- **两个容器认领同一个名称。** 只有第一个保留它：名称由中继生成的按租约密钥来证明，而不是由共享的配对令牌证明。给每个具名服务它自己的 `NODE_NAME`，或者去掉名称并让主机名生效。
- **带有 `advertise_url` 的节点被拒绝。** 在 Docker 中，服务名解析到一个私有范围，除非 `swarm.allow_private_upstreams` 指定了主机，否则中继不会拨号到该范围。在 Compose 网络内部，隧道无论如何都是正确的传输方式。
- **在 `docker compose restart relay` 之后，列表会有一段时间为空。** 注册表存在于内存中；在节点再次签到之前，`/swarm/info` 会报告 `registry_warming: true`，而它们会在租约的三分之一以内(默认 30 秒)签到。
- **模型列表为空且提示词失败。** 节点没有它能访问的提供商：检查其环境中的 `OPENAI_API_KEY`，或从容器内部检查本地服务器的地址。

<!-- docsgen:source sha256=ac82d5d0485a9b74 -->

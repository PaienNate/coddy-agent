# 中继链

**目标.** 在第一个中继之后再加一个中继，并让它拥有自己的节点，这样连接到外层中继的单个客户端就能访问并列出所有内容：整个集群呈树状或环状的中继结构。这就是把分散在各网络中的机群连接成一个整体、且每个中继只知道自己直接子节点的方式。

**环境.** [Docker 中的中继及其节点](swarm-relay-and-nodes.md)所在的实验环境，已运行。在其上新增两个服务。相关机制见[环与路由](../operate/swarm.md#环与路由)。

## 1. 思路

从父节点的视角看，中继就是一个节点。一个既运行中继（`swarm.enable`）、又在 `swarm.join` 中列出父节点的进程，会像智能体一样以 `kind: relay` 注册进该父节点。链接所需的全部就是这些：不需要第二套机制，也不需要任何一跳去知道链路有多深。

## 2. 文件

`inner.yaml`，一个要加入外层中继的中继。其 join 条目中的 `token` 是它自己的客户端令牌，因为外层中继在把请求代理进它时，出示的正是这个令牌：

```yaml
httpserver:
  enable: false

swarm:
  enable: true
  host: "0.0.0.0"
  port: 12346
  name: "inner"
  auth_token: "${INNER_CLIENT_TOKEN}"
  pairing_tokens: ["${SWARM_PAIRING_TOKEN}"]
  join:
    - url: "http://relay:12346"
      name: "inner"
      pairing_token: "${SWARM_PAIRING_TOKEN}"
      token: "${INNER_CLIENT_TOKEN}"
```

`.env` 中的两行：

```bash
INNER_CLIENT_TOKEN=change-me-inner
NODE_C_TOKEN=change-me-node-c
```

`docker-compose.yml` 中新增两个服务，与现有服务并列。`node-c` 是一个普通节点，其 relay 为 `inner`：

```yaml
  inner:
    image: ghcr.io/coddy-project/coddy-agent:latest
    command: serve
    environment:
      CODDY_HOME: /home/user/.coddy
      CODDY_CONFIG: /home/user/.coddy/config.yaml
      INNER_CLIENT_TOKEN: ${INNER_CLIENT_TOKEN}
      SWARM_PAIRING_TOKEN: ${SWARM_PAIRING_TOKEN}
    volumes:
      - ./inner.yaml:/home/user/.coddy/config.yaml:ro
      - inner_home:/home/user/.coddy
    depends_on: [relay]

  node-c:
    <<: *node
    environment:
      CODDY_HOME: /home/user/.coddy
      CODDY_CONFIG: /home/user/.coddy/config.yaml
      CODDY_CWD: /workspace
      RELAY_URL: http://inner:12346
      NODE_NAME: node-c
      NODE_TOKEN: ${NODE_C_TOKEN}
      SWARM_PAIRING_TOKEN: ${SWARM_PAIRING_TOKEN}
      OPENAI_API_KEY: ${OPENAI_API_KEY-}
    volumes:
      - ./node.yaml:/home/user/.coddy/config.yaml:ro
      - ./workspace/node-c:/workspace
      - node_c_home:/home/user/.coddy
    depends_on: [inner]
```

以及另外两个具名卷，`inner_home` 和 `node_c_home`。内层中继不发布端口：它通过外层中继被访问。

## 3. 启动并检查

```bash
mkdir -p workspace/node-c
docker compose up -d
docker compose logs inner
```

内层中继的日志会显示它正在监听，随后是向 `relay` 的 `swarm: joining relay`，然后是 `node-c` 注册进它。从外层中继出发，使用外层客户端令牌：

```bash
curl -s -H "Authorization: Bearer $T" http://127.0.0.1:12346/swarm/nodes
curl -s -H "Authorization: Bearer $T" http://127.0.0.1:12346/swarm/topology
```

外层中继的节点列表以 `"kind": "relay"` 标出 `inner`；拓扑中也在其之后带有 `node-c`，路由为两跳，因为拓扑遍历会向每个子中继询问它自己的子节点。

挂载通过把路径写出来而组合，每一跳剥离自己的前缀：

```bash
curl -s -H "Authorization: Bearer $T" \
  http://127.0.0.1:12346/swarm/nodes/inner/swarm/nodes/node-c/v1/models
coddy -p "Say hello in one line." \
  --remote http://127.0.0.1:12346/swarm/nodes/inner/swarm/nodes/node-c --remote-token "$T"
```

提示词从 `node-c` 得到回答，穿过两个中继，且只使用外层中继的令牌：外层中继用 `inner.yaml` 中的令牌认证到 `inner`，而 `inner` 则用该节点自己的令牌认证到 `node-c`。

聚合列表也以相同方式递归。询问子中继时会要它的聚合结果而非它的节点列表，因此 `node-c` 上的会话会以 `"node_path": ["inner", "node-c"]` 出现在外层中继上：

```bash
curl -s -H "Authorization: Bearer $T" 'http://127.0.0.1:12346/swarm/sessions'
```

在外层中继的 Web 界面中，地图会把内层中继画成一个框，其下带有 `node-c`；进入 `node-c` 会通过两跳驱动它。

## 4. 环

一个节点，或一个中继，可以加入两个父节点。在 `swarm.join` 中再加一个条目，它就会注册进两者；外层中继于是知道通往它的两条路径。这就是一个环，而且是被允许的：路由来自广度优先遍历，因此会报告最短的那条，其余则作为该节点拓扑条目上的 `alternates` 出现，当短跳失效时客户端就在那里进行故障转移。请求会携带它途经的中继 id，因此绕环遍历会终止而不是空转。

## 常见问题

- **内层中继已注册，但通过 `/swarm/nodes/inner/...` 的请求返回 401。** 内层中继自己 join 条目中的 `token` 不是内层中继的 `auth_token`。外层中继在把请求转发进内层中继时会出示该值；两者必须相同。
- **`node-c` 已注册进 `inner`，但外层中继的 `/swarm/nodes` 未列出它。** 这是设计使然：中继的节点列表是它的直接子节点。`node-c` 在 `/swarm/topology` 和 `/swarm/sessions` 中，可通过两跳挂载访问。
- **外层中继的拓扑对内层中继报告警告。** 内层中继已知但当前无法访问：它的隧道断了，或在重启 `inner` 后其租约过期了。它会在租约时间的三分之一内重新注册；持续存在的警告意味着两者之间存在网络问题。
- **挂载路径在几跳后失效。** 手写路径受与扇出相同的跳数预算限制，因此客户端无法无限地绕环遍历。缩短路由：从距离节点最近的中继去寻址它。
- **一个环只显示节点一次，而非两次。** 正确：集群会按身份（`agent_uuid`）对节点去重，保留较短的路由，并把另一条列为备用。

<!-- docsgen:source sha256=ed926bb72f89f916 -->

# 使用 Prometheus 监控

`coddy serve` 可以将提供商调用、工具调用、轮次、等待人工处理的提示词、提供商配额、集群状态、后台任务以及会话存储磁盘，作为 Prometheus 指标在独立端口导出。本页介绍启用导出器、抓取、指标、随附告警规则、Grafana 仪表板以及 Web 界面中的 System 页面。

## 导出器是什么

导出器是 [`coddy serve`](serve.md) 的辅助子系统，默认关闭：单独启用它永远不够（仅启用 `metrics` 的配置会被拒绝），并且随时可以停止。它需要 `metrics` 构建标签，发布二进制、Docker 镜像和 Linux 软件包都带有该标签；不带该标签的二进制会拒绝 `metrics.enable: true` 并指出该标签。

- 它在自己的监听器上提供 `GET /metrics`，与 [HTTP API](../reference/http-api.md) 不共享路由或令牌。
- 计数随进程启动而开始，无论 `metrics.enable` 如何设置，因此稍后启用的导出器也会显示之前发生的情况。计数器随进程重启归零。
- 只有 `coddy serve` 会导出，控制台或 `coddy acp` 不会。

## 启用它

```yaml
metrics:
  enable: true
  host: 127.0.0.1   # the default; empty falls back to it
  port: 12347       # the default; 0 falls back to it
  token: "${CODDY_METRICS_TOKEN}"
```

这些键位于 [配置参考](../reference/config.md#metrics)。设置了令牌后，不带 `Authorization: Bearer <token>` 的抓取会得到 `401`；scheme 不区分大小写匹配，令牌中的 `${ENV}` 会在文件加载时展开。Settings 屏幕的 **Metrics** 标签页（`#/settings/metrics`）编辑相同的键；其令牌字段从不显示已存储的值，留空会保留原值。

![Settings 屏幕的 Metrics 标签页，宽布局](../../assets/monitoring/metrics-settings-dark-1280.png)

*Settings 屏幕的 Metrics 标签页，宽 1280 px，深色主题*

![Settings 屏幕的 Metrics 标签页，窄布局](../../assets/monitoring/metrics-settings-dark-390.png)

*同一标签页，宽 390 px，深色主题*

[配置更改](serve.md#接收配置更改) 会应用到正在运行的进程，且保留计数器：新令牌从下一次抓取开始生效，新的 `metrics.host` 或 `metrics.port` 会原地重建监听器，`metrics.enable` 会启动或停止它。无法绑定的监听器只会停止导出器（`subsystem failed`，日志中带 `subsystem=metrics`），Prometheus 会看到目标不可用（`CoddyDown`）。在迁移地址前运行 `coddy serve --dry-run`：它会报告另一个进程在 `metrics.port` 占用的端口。

## 绑定到回环之外

导出器监听 `127.0.0.1:12347`，只有本机可以访问。这些数字不包含对话内容（参见[从不导出的内容](#从不导出的内容)），但会显示负载和提供商使用情况。要从另一台机器抓取，将 `metrics.host` 设为可访问的地址（`0.0.0.0` 表示全部）并设置 `metrics.token`。没有令牌时，绑定到回环之外（除 `localhost`、`127.x.x.x` 和 `::1` 之外的任何地址）会记录启动警告，清除此类导出器令牌的重新加载会再次记录该警告，并且 `coddy serve --dry-run` 会在 `metrics.host` 发出警告。

在容器中，回环是容器自身的回环，发布的端口无法访问它：在有令牌的情况下绑定 `0.0.0.0` 并发布端口 `12347`。镜像参见 [Docker](../getting-started/docker.md#镜像默认包含的内容)，令牌保护的内容参见[安全与信任](security.md#指标导出器)。

## 抓取它

```yaml
scrape_configs:
  - job_name: coddy
    authorization:
      type: Bearer
      credentials_file: /etc/prometheus/coddy-metrics-token
    static_configs:
      - targets: ["coddy.example.net:12347"]
```

位于回环且没有令牌的导出器不需要 `authorization` 块。[随附规则](#随附告警规则) 监视 `up{job=~"coddy.*"}`，因此让任务名以 `coddy` 开头，以便 `CoddyDown` 覆盖它。先手动检查：

```bash
curl -s -H "Authorization: Bearer $CODDY_METRICS_TOKEN" http://127.0.0.1:12347/metrics | head
```

它对 `200` 返回暴露文本；令牌缺失或错误时对 `401` 返回 `WWW-Authenticate: Bearer realm="coddy-metrics"`；除 `/metrics` 之外的任何路径返回 `404`；当两个抓取正在运行或一个抓取超过 10 秒时又来了第三个抓取，则返回 `503`。

## 它导出什么

抓取不会向提供商、机器人 API 或 MCP 服务器询问任何内容。`go_*` 和 `process_*` 指标族来自 `github.com/VictoriaMetrics/metrics`，因此集合取决于平台。Coddy 为每个指标族添加通用 HELP 行，并让它们保持无类型。表格列出 Coddy 自己的指标族。

### 提供商调用

| 指标族 | 类型 | 含义 |
|--------|------|---------|
| `coddy_llm_attempts_total{provider,model,outcome,attempt}` | counter | 提供商调用尝试，包含重试：`attempt` 为 `first` 或 `retry`。 |
| `coddy_llm_attempt_duration_seconds{provider}` | histogram | 尝试持续时间，包含整个流。桶：0.5、1、2.5、5、10、30、60、120、300、600 秒。 |
| `coddy_llm_time_to_first_token_seconds{provider}` | histogram | 从流式尝试开始到第一个带文本、推理或工具调用的块的时间。桶：0.25、0.5、1、2、4、8、15、30、60 秒。 |
| `coddy_llm_input_tokens_total{provider,model}` | counter | 提供商报告的输入 token。 |
| `coddy_llm_cached_input_tokens_total{provider,model}` | counter | 缓存的输入 token，是输入 token 的子集。 |
| `coddy_llm_output_tokens_total{provider,model}` | counter | 提供商报告的输出 token。 |

一次 **尝试** 是重试包装器对提供商发起的一次调用，因此被重试的调用每次尝试计数一次：第一次尝试为 `attempt="first"`，之后每次为 `attempt="retry"`，`first` 次尝试会计入这些调用。模型列表、用量读取和登录不是尝试。只要存在响应，token 就来自该尝试的响应。仅当尝试向调用方流式传输了内容时才观察首个 token 的时间；非流式调用以及在产生任何内容前结束的流没有该时间。

每次尝试恰好落入一个 `outcome`：`ok`、`canceled`（调用方停止了它）、`timeout`（截止时间已到或流停滞）、`rate_limited`（HTTP 429）、`auth`（401 或 403）、`context_overflow`（提示词不适合模型的上下文窗口）、`client_error`（任何其他 4xx）、`server_error`（5xx、流被截断、拨号失败、瞬态传输错误或提供商在其流中报告的过载）以及 `error`（其他所有情况）。

### 工具调用

| 指标族 | 类型 | 含义 |
|--------|------|---------|
| `coddy_tool_calls_total{tool,outcome}` | counter | 智能体决定进行的工具调用，涵盖进程中的每个会话，包括子智能体的会话。 |
| `coddy_tool_call_duration_seconds{tool}` | histogram | 工具运行了多久，针对实际运行的调用；钩子不计入。桶：0.1、0.5、1、5、15、60、300 秒。 |

`tool` 是内置工具的名称（`read`、`run_command`、`spawn_agent`，[工具参考](../reference/tools.md) 列出了它们），`mcp` 表示 MCP 服务器的任何工具，`other` 表示智能体不认识的名称，例如模型编造的名称。一次调用会落入一个 `outcome`：`ok`（工具返回了结果）、`error`（返回了错误）、`refused`（从未运行：模式、工具集、子智能体的工具集、钩子或安全检查拒绝了它）或 `denied`（在权限提示中被人拒绝）。没有人回答的提示词，由于承载它的流已结束，不算结果：调用在恢复并运行时才计数。

### 进程

| 指标族 | 类型 | 含义 |
|--------|------|---------|
| `coddy_turns_started_total`, `coddy_turns_ended_total` | counter | 此进程中开始和结束的轮次。 |
| `coddy_turn_duration_seconds` | histogram | 已结束轮次的持续时间，从会话变为忙碌的那一刻算起。桶：1、5、15、30、60、120、300、600、1800、3600 秒。 |
| `coddy_config_reloads_total` | counter | 实时配置的替换次数。 |
| `coddy_build_info{version,revision,goversion,goos,goarch,tags}` | gauge | 正在运行的构建；值为 1。 |
| `coddy_subsystem_up{subsystem}` | gauge | 对于进程正在运行或正在启动的子系统为 1，对于已启用但未运行的子系统为 0；导出器不包含在内。 |
| `coddy_restart_required` | gauge | 当某项已保存的更改只有新进程才能应用、并等待一个不会自行执行的重新启动时为 1：例如迁移了 HTTP 或集群地址，或某个界面在未使用会话存储启动的进程中运行轮次（[自行重启](serve.md#自行重启)）。 |
| `coddy_turns_running` | gauge | 有轮次正在运行的会话数。 |
| `coddy_turn_oldest_running_seconds` | gauge | 运行时间最长的轮次的时长，没有轮次运行时为 0。等待权限或用量限制是合理的，因此不能证明卡死。 |
| `coddy_background_tasks{kind,status}` | gauge | 按类型（`command`、`agent`、`server`）和状态（`queued`、`running`）统计的模型后台任务。 |
| `coddy_sessions_disk_available_bytes`, `coddy_sessions_disk_size_bytes` | gauge | 会话存储所在磁盘上非 root 写入者可用的空闲空间及其大小；读取失败或进程没有会话存储时不存在。 |

一个 **轮次** 是智能体在一个会话上的一次运行，可发生在进程的任何界面上，包括由已完成的后台任务唤醒的一次运行。

### 等待、配额与集群

| 指标族 | 类型 | 含义 |
|--------|------|---------|
| `coddy_prompts_pending{kind}` | gauge | 等待人工回答的提示词：`permission`（权限提示）或 `question`（`question` 工具）。 |
| `coddy_provider_quota_used_ratio{provider,window}` | gauge | 提供商账户某个用量窗口已使用的比例，从 0 到 1。 |
| `coddy_provider_quota_reset_timestamp_seconds{provider,window}` | gauge | 该窗口重新开始的时间，Unix 时间；提供商未指明时间时不存在。 |
| `coddy_swarm_joins{relay,transport,state}` | gauge | 此进程注册到父中继（`swarm.join`）的情况，按中继主机、传输方式（`direct`、`tunnel`）以及最后一次尝试的状态（`online`、`offline`）分组。 |
| `coddy_swarm_nodes{kind,transport,state}` | gauge | 此进程的中继已知的节点，按类型（`agent`、`relay`）、传输方式和状态分组；仅在运行中继的进程中存在。 |

提示词在请求它的调用等待期间计数，包括子智能体的提示词。在回答前流已结束的提示词不在任何调用中等待，因此只有在恢复时才再次计数。等待提示词的轮次不是卡死，等待中的提示词可将 `CoddyTurnLongRunning` 与卡住的轮次区分开。

配额指标族从会话管理器为各界面用量面板保留的缓存中读取，因此抓取不会向提供商询问任何内容，数值的新旧程度与各界面最后一次读取时相同（轮次结束时会再次读取）。它们存在于具有用量读取器的提供商类型：`neuraldeep`、`codex` 和 `devin`，并启用 `providers[].usage_limits_panel`。`window` 是 `session`、`day`、`week`、`acu` 和 `window-<seconds>` 之一，在相同长度的第二个窗口之后为 `-secondary`。最后一次读取失败的快照会被省略，而不是用旧数字报告。

### 标签及其预算

`provider` 是 `providers[]` 中提供商行的名称，`model` 是在 `models[]` 中声明的模型的 API 模型 id，不带提供商前缀（`gpt-5`，而不是 `openai/gpt-5`）。其他任何内容都是 `other`：未声明的提供商为 `provider="other", model="other"`，已声明提供商的未声明模型为 `model="other"`。本身名为 `other` 的提供商行或模型 id 与该回退值共享其序列。

标签在观察到尝试时根据实时配置决定。前 16 个提供商名称和前 64 个已命名提供商与模型对保留自己的标签，之后每个都是 `other`。已接纳的名称会保持接纳，因此重新加载、重命名或删除永远不会从计数器中移除序列，重启会重新开始预算。达到上限时，`coddy_llm_*` 指标族合计约 2,100 个序列。配额指标族只携带已接纳的提供商名称。

启动时以及每次配置重新加载时，每个已接纳的提供商都会获得 `coddy_llm_attempts_total`，对首次尝试和重试的每种结果都为零，三个 token 计数器为零，每个已声明模型一次，`other` 一次，以及空的 `coddy_llm_attempt_duration_seconds` 和 `coddy_llm_time_to_first_token_seconds`，因此对首次尝试的 `increase()` 会计算该尝试。`coddy_turn_duration_seconds` 从一开始就存在，且为空；工具指标族没有种子。

### 读取数字

一次抓取会依次读取各指标族，因此它们描述的是略有不同的时刻：在同一次抓取中，`coddy_llm_attempts_total` 与 `coddy_llm_attempt_duration_seconds` 的 `_count` 可能不同，已开始轮次减去已结束轮次不一定等于 `coddy_turns_running`。只有直方图的 `_count` 等于其自身的 `+Inf` 桶。使用 `rate()` 或 `increase()` 在窗口内比较指标族，而不是比较两个瞬间。

## 随附告警规则

[`examples/prometheus/coddy-rules.yml`](../../../examples/prometheus/coddy-rules.yml) 在一个名为 `coddy` 的组中包含七条告警；使用 `rule_files` 加载它：

| 告警 | 严重级别 | 触发条件 |
|-------|----------|------------|
| `CoddyDown` | critical | `up{job=~"coddy.*"}` 连续 5 分钟为 0 |
| `CoddyDiskLow` | warning | `coddy_sessions_disk_available_bytes` 连续 5 分钟低于 512 MiB |
| `CoddyProviderFailing` | warning | 15 分钟内某个提供商的尝试中超过 20% 以 `server_error`、`timeout`、`auth` 或 `rate_limited` 结束，且在超过 10 次尝试中如此，持续 10 分钟 |
| `CoddyTurnLongRunning` | info | `coddy_turn_oldest_running_seconds` 超过一小时 |
| `CoddyRestartRequired` | warning | `coddy_restart_required` 连续 10 分钟为 1 |
| `CoddySwarmJoinDown` | warning | 到中继的注册连续 5 分钟为 `offline` |
| `CoddyProviderQuotaHigh` | warning | 提供商账户的某个用量窗口连续 5 分钟使用超过 90% |

CI 会在每个拉取请求上测试它们。[`examples/prometheus/coddy-rules.test.yml`](../../../examples/prometheus/coddy-rules.test.yml) 包含每条规则必须触发和必须保持静默的用例；使用 `promtool test rules examples/prometheus/coddy-rules.test.yml` 运行它们。

## Grafana 仪表板

[`examples/grafana/coddy-dashboard.json`](../../../examples/grafana/coddy-dashboard.json) 为单个进程或整个集群绘制上述所有指标族：集群概览和每个进程一行、轮次、工具调用、等待中的提示词和提供商配额、集群的中继和注册、模型调用及首个 token 时间和重试、token、后台任务和子系统，以及 Go 和进程指标。在 Grafana 中导入它（**Dashboards → New → Import**），并选择 Prometheus 数据源；它面向 Grafana 12 制作。

![Coddy 仪表板，覆盖三个智能体和集群的一个中继](../../assets/monitoring/grafana-dashboard-dark-1280.png)

*覆盖三个智能体和集群一个中继的仪表板，按目标的 `node` 标签分组*

顶部的变量用于筛选和拆分每个面板：

- **Job** 和 **Process** 选择抓取任务和目标（`instance`）；
- **Provider** 缩小模型调用和 token 面板的范围；
- **Group by** 指定按组面板拆分所用的标签：`instance`、`job` 或抓取配置放在目标上的任何标签，例如 `node`、`role` 或 `site`。该列表从目标中读取，因此在那里添加的标签会自动出现。

[`examples/grafana/docker-compose.yaml`](../../../examples/grafana/docker-compose.yaml) 在主机网络上启动 Prometheus 和 Grafana，并配置好[随附规则](#随附告警规则)和仪表板。在 [`examples/grafana/targets.yml`](../../../examples/grafana/targets.yml) 中列出你的导出器，每个 `coddy serve` 一个条目，并带上 **Group by** 列表应提供的标签：

```yaml
- targets: ["10.0.0.11:12347"]
  labels: {node: alpha, role: agent}
- targets: ["10.0.0.10:12347"]
  labels: {node: relay-1, role: relay}
```

```bash
CODDY_METRICS_TOKEN=... docker compose -f examples/grafana/docker-compose.yaml up -d
```

Prometheus 在 `http://127.0.0.1:9090` 上响应，Grafana 在 `http://127.0.0.1:3000` 上响应，未登录时为只读（编辑使用 `admin` / `admin`）。`PROMETHEUS_PORT`、`GRAFANA_PORT` 和 `CODDY_TARGETS`（另一个 targets 文件）可移动它们。targets 文件更改时，Prometheus 会重新读取它。

## System 页面

Web 界面中的 **Settings → System** 面向没有 Prometheus 和 Grafana 的安装：它一眼展示一个进程，无需任何设置。Prometheus 和[仪表板](#grafana-仪表板)仍是历史、告警以及该页面不绘制的指标族（提供商配额、工具调用、轮次持续时间、重试、首个 token 时间和中继的节点数）的所在。该页面只保留不超过一小时的历史，且不发送告警。

它显示版本和构建标签、进程的启动方式、每个子系统的状态和地址（导出器所在行说明抓取是否需要令牌）、会话磁盘相对于 `CoddyDiskLow` 监视的 512 MiB 线的状态、轮次、等待回答的提示词和当前运行的后台任务，并在有事项需要关注时显示徽章，包括已离线达到 `CoddySwarmJoinDown` 所等待的 5 分钟的中继注册。参见 [System](../surfaces/web-ui.md#系统设置标签页)。

![Settings → System，显示三个需要关注的事项，位于最近一小时的图表以及安装、子系统、磁盘、活动和模型提供商之上](../../assets/web-ui/system-attention-dark-1280.png)

*Settings → System，显示有待重启、指标导出器已停止以及一个中继不可达。*

该页面读取 `GET /coddy/status`（[HTTP API](../reference/http-api.md#系统状态)），它位于 API 自身凭据之后：指标令牌无法打开它，API 令牌也无法打开 `/metrics`。使用 `metrics` 标签构建的二进制会添加 **Last hour** 图表和 **Model providers since start** 表，无论 `metrics.enable` 如何设置；没有该标签的二进制则两者都没有。采样器每 15 秒读取导出器自己的计数器，方式与抓取写入时相同，并在内存中保留 240 个样本，即一小时。尝试序列在 `attempt`（首次和重试）上求和，与规则相同。它不自行计数，不需要抓取，重启后从空开始。

## 从不导出的内容

导出器报告计数、持续时间和状态，从不报告提示词、模型输出、工具参数或结果、权限提示或问题的文本、会话 id 或标题、工作区文件夹、节点名称、MCP 服务器或其工具的名称，或对话的其他内容。每个标签值要么是封闭集合的成员（结果、任务或提示词类型、状态、子系统名称、内置工具名称、配额窗口、集群类型、传输方式或状态、构建标签、Go 版本、操作系统、架构），要么是来自你配置的名称（提供商行、已声明模型 id、`swarm.join` 中中继的主机），因此请确保这些名称不含任何你不想放入仪表板的内容。令牌从不导出，`GET /coddy/config` 仅报告是否设置了令牌。

## 中继和其他进程

每个 `coddy serve` 都在自己的端口上有自己的导出器，包括中继：中继进程导出与其他任何进程相同的指标族（它的 `coddy_subsystem_up{subsystem="swarm"}`、它的构建以及它的 Go 和进程指标；其轮次和 token 指标族保持为零）以及 `coddy_swarm_nodes`，即它已知的节点。每个加入中继的进程都会导出 `coddy_swarm_joins`。中继不会承载其背后节点的导出器：挂载会拒绝 `/metrics`。直接抓取每个节点，将一个集群的节点放入一个 Prometheus 任务，并为每个节点设置一个目标，再为目标添加标签（`node`、`role`），供仪表板的 **Group by** 使用。

通过隧道传输方式加入其中继的节点（`swarm.join[].advertise_url` 为空）位于不接受入站连接的网络中，因此 Prometheus 无法从外部访问其导出器。在此类节点旁以 agent 模式运行 Prometheus，在回环上抓取它，并使用 `remote_write` 将样本发送出去。

<!-- docsgen:source sha256=d9fe69675311aff4 -->

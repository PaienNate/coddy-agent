# Web 搜索

`websearch` 工具会同时向多个搜索引擎发起查询，并合并返回的结果，而 `webfetch` 工具会下载单个页面并以 Markdown 返回。智能体正是通过这两者读取仓库之外的内容。

搜索引擎对待程序的方式与对待浏览器的方式并不相同。有的会用验证页面作答，有的在浏览器中渲染结果、返回给你的却是一份空文档，还有的会在遇到它不喜欢的查询时，返回一整页格式规整、却完全关于另一个主题的结果。一份掩盖了上述情况的合并答案比没有答案更糟：模型会把空列表读成"网上没有关于这个的内容"，然后停止查找。因此每个引擎都会在结果旁报告自己的结果状态。

## 返回内容

```json
{
  "query": "golang context cancellation",
  "page": 1,
  "engines": [
    { "engine": "brave", "status": "ok", "results": 12, "took_ms": 620 },
    { "engine": "bing", "status": "blocked", "reason": "decoy: none of the results share a word with the query", "took_ms": 710 }
  ],
  "results": [
    {
      "title": "Canceling in-progress operations - The Go Programming Language",
      "url": "https://go.dev/doc/database/cancel-operations",
      "description": "You can manage in-progress operations by using Go context.Context...",
      "source": "brave"
    }
  ],
  "hint": "More results are available: call websearch again with page incremented, or narrow the query."
}
```

每个引擎都会带有四种结果之一。

`ok` - 引擎作出了应答，且其行已被解析。`results` 统计它向合并答案贡献了多少条，因此另一个引擎已经提供过的页面不会被重复计数；
`empty` - 引擎返回了一个解析器能理解的结果页，但页面上没有任何内容。这是唯一一种真正意味着"那些关键词下网上什么都没有"的结果；
`blocked` - 引擎返回的不是结果页：非 2xx 状态、验证挑战、解析器已无法识别的布局，或一批与查询无关的行。`reason` 会说明是哪种；
`error` - 根本无法连接到该引擎。

当**所有**引擎都被拦截时，调用会失败而不是返回空列表，错误信息会列出每个引擎及其原因。整个设计的关键正在于此：搜索失败会被报告为搜索失败，而不是作为关于世界的一个事实。

`cached: true` 标记的是复用了近期相同搜索的答案，而非重新查询。

### 在 Web 界面中

对话记录中的搜索行呈现的是它找到的内容，而不是上面的 JSON。它的标题会指明查询以及该次搜索使用的每个参数 - 页码、请求的行数（调用未指定时为 15）以及站点（如果有的话）- 其下第一行说明每个引擎如何作答：作出应答的引擎给出数量，未作答的引擎给出结果状态及其原因。命中结果以链接形式跟随其后，每条摘要各占一行。打开该行会一次性加载完整答案：该行只保留输出的前几行，这些行由引擎报告填充，因此查看结果时没有 **More** 控件需要点击。

![已展开的搜索行：标题中的查询及其参数、引擎报告、每条命中作为一个链接](../../assets/web-search/web-search-row-open-dark-1280.png)

*已展开的搜索行：标题中的查询及其参数、每个引擎如何作答、每条命中作为一个链接*

## 引擎

| 引擎 | 是否需要密钥 | 备注 |
|--------|-----------|-------|
| `brave` | 否（可选） | 默认引擎，也是合并顺序中的第一个。读取公开结果页；若提供 API 密钥（`brave_api_key` 或 `BRAVE_API_KEY`），则改用官方 Search API，不存在会失效的解析器。 |
| `bing` | 否 | 默认引擎，第二个。会对它不喜欢的客户端返回一组无关结果，这会被下面的相关性闸门捕获。 |
| `searxng` | 否，但需要 `searxng_url` | 你自己的实例，通过其 JSON API 查询。可靠之选：不受针对其所有者的速率限制，也永远不会收到诱饵结果。 |
| `deepseek` | 是（`deepseek_api_key` 或 `DEEPSEEK_API_KEY`） | [DeepSeek Search API](https://api-docs.deepseek.com/guides/search)，默认禁用。需要密钥，且仅支持第 1 页。凭据缺失以及对后续页面的请求都会报告为 `blocked`。查询（包括任何 `site:` 限制）必须包含 1-128 个 Unicode 字符。 |
| `ddg` | 否 | 默认不查询。从服务器发起时，其两个端点都会对每个查询返回 HTTP 202 反爬页面，表现为 `blocked`；在 DuckDuckGo 仍能正常服务你的地方，它没有结果的查询会表现为 `empty`。 |
| `google` | 否 | 默认不查询。提供给未获批准客户端的页面不含任何自然结果，无论使用何种用户代理；其结果在浏览器中渲染。 |

结果按引擎的列出顺序合并，并按规范化后的 URL 去重 - 通过 `www`、经 `http`、带营销参数（`utm_*`、`fbclid`、`gclid` 及其同类）或以斜杠结尾到达的同一页面算作一行，而交给模型的 URL 是其引擎返回的那个。用于选择内容而非标识来源的参数会保留，因此同一文件的两个修订版本是两个结果。

### 相关性闸门

Bing 会对它不喜欢的客户端返回一个标题呼应查询、而十行结果却关于其他内容的页面：关于 Go 的问题给出披萨配料，关于某个 GitHub 仓库的问题给出旅游网站，下一次请求又是另一个主题。下游的任何环节都无法把这样的一行与真实结果区分开。

因此，一批结果会作为整体来判定，且仅当它大到足以构成一个结果页时才判定。当引擎返回至少五行，且其中没有一行与查询有任何关系时，该引擎并未对查询作出应答，整批结果会以 `status: blocked`、`reason: decoy` 被丢弃。

查询词是否算作命中判定得相当宽松：作为行中的一个词、作为一个词的一部分（关于 `OOM` 的问题可由关于 `OOMKilled` 的页面来命中），以及词尾折叠，使 `cancellation` 能匹配 `cancel`。这项检验的一切都倾向于保留结果，因为它的错误将是丢弃一个真实答案 - 而它所针对的诱饵按任何标准都与查询毫无共同之处。

判定以整批为单位也是同样的道理。单独一行未命中任何查询词是正常的：一个标题为 "Terminating programs" 的页面，对于关于终止进程的问题是很好的答案。一整页都是这样就不正常了。

## 配置

```yaml
tools:
  websearch:
    engines: [brave, bing]        # merge order; brave, bing, ddg, google, searxng, deepseek
    engine_timeout_seconds: 8     # per-engine budget
    total_timeout_seconds: 20     # budget for the whole search
    max_concurrent_engines: 4     # how many engines are asked at once
    snippet_chars: 320            # per-result description cap
    cache_ttl_seconds: 300        # reuse an engine answer for this long; negative = off
    searxng_url: ""               # e.g. http://localhost:8080
    brave_api_key: ""             # official Brave Search API; or BRAVE_API_KEY
    deepseek_api_key: ""          # DeepSeek Search API; the deepseek engine needs a key
```

要使用 DeepSeek，请在 `engines` 中添加 `deepseek`，并设置 `deepseek_api_key` 或导出 `DEEPSEEK_API_KEY`。已配置的密钥优先。API 以 Bearer 认证接收 `POST /v1/tools/search`，JSON 正文包含 `query` 和 `count`。它在第 1 页最多返回 `max_results` 行。后续页面会跳过 DeepSeek 而不发出 HTTP 请求，因此它无法在其他引擎的结果之前重复第一页的行。仅使用 DeepSeek 时，提示会要求使用更窄的查询，而不是翻到下一页。

![设置中的 DeepSeek Search API 密钥](../../assets/web-search-deepseek-settings-dark-1280.png)

*选择了 DeepSeek 的网页搜索设置，其 API 密钥字段留空以使用 `DEEPSEEK_API_KEY`。*

加载器不认识的引擎名属于配置错误，而不是被静默跳过的后端；`searxng` 缺少 `searxng_url` 也会以同样方式被拒绝 - `coddy -t` 会连同所在行一起报告这两者。

### Brave Search API 密钥

密钥不必存放在 `config.yaml` 中。当 `brave_api_key` 为空时，引擎会读取 `BRAVE_API_KEY` 环境变量，因此密钥可以来自 shell、容器或 `~/.coddy/.env`：

```bash
echo 'BRAVE_API_KEY=BSA...' >> ~/.coddy/.env
```

文件中的密钥优先于该变量，`brave_api_key: ${BRAVE_API_KEY}` 的用法与其他引用相同。两种方式都不会泄露密钥：`config_get` 和 `coddy -t` 会将其打印为 `<redacted>`，而从设置界面保存时，会把 `${BRAVE_API_KEY}` 引用原样写回为引用，把空字段写回为空，绝不会写回环境提供的密钥。该变量在轮次构建其工具设置时读取；向正在运行的 `coddy serve` 的 `.env` 添加密钥，会在重启后生效，因为 `.env` 在启动时读取。

### 你自己的 SearXNG

自托管的 [SearXNG](https://docs.searxng.org/) 是对抓取型引擎各种失败方式的统一答案。它自己聚合各引擎，不受针对其所属机器的速率限制，也永远不会收到诱饵结果。

```yaml
tools:
  websearch:
    engines: [searxng, brave]
    searxng_url: http://localhost:8080
```

该实例必须提供 JSON API，而其默认配置并不提供：在其 `settings.yml` 的 `search.formats` 中添加 `json`。此处设置错误会返回 HTTP 403，表现为带有该原因的 `blocked`，而不是空结果列表。

与交给 `webfetch` 的 URL 不同，这个地址来自你自己的配置而非来自模型，而自托管实例通常监听 `localhost` 或某个局域网地址 - 因此私网范围是有意允许的：拒绝它们恰恰会拒绝这个引擎存在的意义所在的那类部署。

被拒绝的是链路本地范围，那里没有 SearXNG 在监听，而云元数据服务在监听。主机名会在该检查之前解析，因为指向那里的名称与字面地址一样能到达它，而且该引擎从不跟随重定向：你审查的是你配置的地址，而不是它转发到的任何地方。

### 缓存

一个引擎对一个查询的答案会被记住 `cache_ttl_seconds`（默认 300），因此模型对同一搜索查询两次时不会让引擎回答两次。

失败会被记住的时间要短得多，且两类失败各不相同。验证页面具有粘性 - 引擎对下一次调用也会返回它 - 因此保留 30 秒：足以阻止单个轮次内的重试，又短到能察觉到引擎的恢复。完全无法连接的引擎会保留 5 秒，这能省下一个轮次内的重复请求，又不会掩盖已经恢复的主机。

DeepSeek 的缓存条目包含 API 密钥的 SHA-256 指纹，绝不包含密钥本身。更换密钥会开启新的缓存条目，因此缓存中的认证失败不会在凭据变更后继续存在。成功的 DeepSeek 响应限制为 4 MiB，且必须包含 `results` 数组；格式错误的答案会报告 `blocked`，而不是空搜索。

## 参数

| 参数 | 含义 |
|----------|---------|
| `query` | 必填。搜索词，使用答案可能所用的语言。 |
| `page` | 结果页码，从 1 开始。沿同一结果列表继续向下翻。 |
| `max_results` | 本页返回的行数；默认 15，最多 25。 |
| `site` | 限制到单个域，例如 `go.dev`。会折叠进各引擎自己的语法。 |

## 读取页面

`websearch` 返回的是摘要，而摘要不是答案。`webfetch` 接收一个 URL，通过可读性提取以 Markdown 返回文章，拒绝私网和 localhost，并在跟随每个重定向之前按相同规则检查它 - 参见[工具](../reference/tools.md)。它通过 [`http_request`](http-requests.md) 的客户端发送请求，后者是用于调用 API 或任何其他不属于读取页面的操作的工具。

## 相关

[工具](../reference/tools.md) - 每个内置工具及其参数；
[配置](../getting-started/configuration.md) - `config.yaml` 所在的位置以及如何检查它；
[运行模式](modes.md) - `websearch` 和 `webfetch` 在每种模式下都会提供，包括只读的 `ask`；
[HTTP 请求](http-requests.md) - `http_request`，智能体的 curl，以及 `webfetch` 所基于的客户端。

<!-- docsgen:source sha256=c081528189644e39 -->

# 内置文档

Coddy 自带文档。本文档的每个页面都会在构建时编译进二进制，因此人们阅读的页面和智能体查找的页面都是运行中的这个二进制自身的文档：不会向站点发送请求，文本与程序之间不会有偏差，而且在没有网络的机器上也能工作。

同一批页面可从每个界面打开：

| 位置 | 方式 | 得到什么 |
| --- | --- | --- |
| Web 界面 | 侧边栏中的 **文档**、**F1**、输入框中的 `/docs [words or page]`，或地址 `#/docs/<page>#<section>` | 一个阅读器：目录、页面、页面章节、搜索、上一页和下一页、**询问智能体** |
| 控制台 | **F1**，或 `/docs [words or page]` | 覆盖在编辑器上的帮助屏幕：搜索，然后在找到的章节处打开页面 |
| Shell | `coddy docs list`、`coddy docs search <words>`、`coddy docs show <page>[#section]` | 目录、最匹配的章节、以 Markdown 表示的页面或章节 |
| 智能体 | 工具 `coddy_docs_search` 和 `coddy_docs_read` | 在每种模式下搜索和阅读 |
| 提示词 | `@coddy:<page>#<section>` | 附加到消息的页面或章节 |

## 页面和章节

页面以其在 `docs/` 下的路径来标识，不带 `.md`：`features/mentions`、`reference/config`。这个名字在所有地方都是它的地址：`@coddy:features/mentions`、`coddy docs show features/mentions`、Web 阅读器的 `#/docs/features/mentions`，以及 `https://coddy.dev/docs/features/mentions` 处的公开副本。章节则加上 GitHub 为其标题生成的锚点：`features/mentions#completion`。

当按名称请求某个页面或章节时，Coddy 也接受人们书写它的其他方式：带或不带 `docs/` 和 `.md` 的文件路径，一个 `coddy:` 或 `@coddy:` 链接，一个 `coddy.dev/docs` 地址，仅当一个页面拥有该文件名时的页面文件名（`mentions`），以及页面标题（`Mentions`）。没有任何匹配的名称会得到最接近的页面作为回答；页面没有的章节，会得到它已有的章节。

在从二进制中读出的页面内，指向另一个页面的链接写作 `coddy:<page>#<section>`。Web 阅读器会就地打开它，智能体将其传给 `coddy_docs_read`。在 Web 界面中，会话里任何位置提到的页面都会打开阅读器：你发送的消息中的 `@coddy:<page>#<section>`，以及回答中的 `coddy:` 链接或 `@coddy:` 引用，这就是智能体被要求把你指向某个页面的方式。

图片、视频和仓库文件的链接指向二进制构建时所基于的发布版本的 GitHub（开发构建则为 `main`）：文本在二进制中，截图和录像不在，因此联网时它们会显示，不联网时则回退到图注。GitHub 从附件嵌入的录像会在阅读器中从其在仓库中的副本播放。

## 语言

文档以英文编写，并翻译成俄文和简体中文：每个页面在 `docs/ru/` 和 `docs/zh/` 下都有相同路径的译文，二进制中同时包含三者。阅读器中没有语言切换。每个界面都会显示它已经知道你使用的语言：

| 界面 | 显示的语言 |
| --- | --- |
| Web 界面（以及 Telegram Mini App） | 界面语言，**设置 → 外观**或浏览器语言 |
| Telegram 和其他 messenger | 该用户的 Telegram 客户端所报告的语言（`language_code`），否则是他们书写的语言 |
| 控制台、`coddy docs` | `CODDY_LANG`，否则是终端的区域设置：`LC_ALL`，然后是 `LC_MESSAGES`，再是 `LANG`；对单条命令使用 `coddy docs --lang ru` |
| 智能体的工具 | 调用所指定的 `lang`，否则是该轮次的语言：界面的语言，若没有则是用户编写提示词所用的语言 |
| 提示词中的 `@coddy:` | 该轮次所来自界面的语言，否则是你编写提示词所用的语言 |

某个界面的语言没有译文时（德语终端、乌克兰语 Telegram 客户端），会把选择留给提示词的语言，而两者都不匹配的提示词则按英文阅读。页面在三种语言中保持同一地址，章节也是如此：俄文或中文标题会响应同一位置英文标题的锚点，因此 `coddy:features/mentions#completion`、`@coddy:features/mentions#completion` 或 `#/docs/features/mentions#completion` 无论使用哪种语言都会打开同一章节，智能体用一种语言写下的链接会在另一种语言中打开。俄文页面的公开副本是 `https://coddy.dev/ru/docs/<page>`，中文页面的公开副本是 `https://coddy.dev/zh/docs/<page>`。

## 搜索

搜索对章节而非页面进行排序，因此命中会落在给出答案的段落上。它用 BM25 在三个字段上为每个章节评分：页面标题、章节标题和正文，其中标题和页面标题的权重高于正文，因此指名某一章的查询会找到该章。经过词干提取后，匹配不区分大小写：英文使用轻量词干提取器（`sessions` 能找到 `session`），俄文使用 Snowball 词干提取器（`сессиями` 能找到 `сессия`，`ё` 读作 `е`），中文按相邻两字切分（`安装` 能找到包含“安装”的章节），标识符会按整体及其组成部分匹配（`max_turns` 和 `turns` 都能找到 `agent.max_turns`），三个字母或更长的词也会找到以它开头的词，因此逐字母输入的查询在完成之前就能找到页面。一个页面最多显示三个章节，因此长页面不会把其他页面挤出去。每个命中都带有摘要片段：该章节文本中包含最多匹配词的连续片段，并标记这些词。

索引在第一次有搜索时构建，耗时约十分之一秒，并在进程生命周期内保留。

## Web 界面阅读器

![阅读器：左侧为目录，中间为页面，右侧为页面章节](../../assets/built-in-docs/reader-page-dark-1280.png)
*阅读器在 1280 px 下：显示正在阅读的页面的目录、页面本身，以及标记出正在阅读章节的页面章节列表。*

侧边栏中的 **文档** 会在上次停留的页面上打开阅读器，**F1** 可从应用中的任何位置打开和关闭它，**Escape** 会像 × 一样关闭它，输入框中的 `/docs` 会执行它在控制台中的行为（`/docs telegram proxy` 会在该搜索上打开阅读器，`/docs features/mentions#completion` 会在该章节上打开，且不会向智能体发送任何内容），而诸如 `#/docs/features/mentions#completion` 这样的地址会直接打开该章节，可在新标签页中打开，也可从别人发送的链接打开。每个页面、章节和搜索结果都是真实链接：浏览器的后退和前进会在已读页面之间移动，中键单击会在新标签页中打开一个页面，章节标题旁的 `#` 就是该章节的地址。带搜索的页眉固定不动，页面在其下方滚动；右侧栏会跟随正在阅读的章节。

![搜索框下方的搜索结果，匹配词已标记](../../assets/built-in-docs/reader-search-dark-1280.png)
*页眉中的搜索框会在你输入时于其下方列出找到的章节。Up 和 Down 选择一个，Enter 会在其章节处打开它，Escape 清除搜索，再按一次 Escape 关闭阅读器。*

- **像读书一样阅读。** 每个页面底部都按目录顺序链接上一页和下一页。
- **询问智能体**会打开一个新聊天，并提及该页面（`@coddy:<page>#<section>`，如果有正在阅读的章节，则提及该章节）。当页面上选中了文本时，该按钮会就所选内容提问：聊天打开时会引用所选内容并提及它所在的章节，准备好提问。
- **图片**会在点击时覆盖在页面上打开，并适应窗口大小；`+`、`-` 和按钮可将缩放提高到 300%，点击图片会在适应大小和双倍大小之间切换，缩放后的图片可滚动。Escape 关闭它。
- 页面上方的 **coddy.dev** 会打开同一页面的公开副本，便于分享给没有 Coddy 的人。
- `/` 会将光标放入搜索框。

阅读器会跟随应用的主题和语言：将界面切换为俄文或中文后，目录、页面和搜索都会以该语言显示。在窄屏幕上，目录会折叠成页面上方的一个按钮。

## 控制台帮助

![控制台帮助：带有选中章节摘要片段的搜索列表](../../assets/cli-tui/19-docs-search.png)
*F1，然后 `telegram proxy`：找到的章节，选中的章节及其地址和摘要片段。*

**F1** 会在编辑器的位置打开帮助并显示目录；输入即搜索。**Enter** 会打开选中的条目，并滚动到其章节。有些终端会保留 F1 自用（GNOME Terminal 会打开自己的帮助）；`/docs` 会打开同一屏幕，`/docs telegram proxy` 会带上该搜索，`/docs features/mentions#completion` 则直接定位到该章节。

![控制台帮助：在找到的章节处打开的页面](../../assets/cli-tui/20-docs-page.png)
*在章节处打开的页面：阅读器在页面中的位置，以及按键。*

| 按键 | 在列表中 | 在页面上 |
| --- | --- | --- |
| 字母、Backspace | 编辑搜索 | - |
| Up、Down | 选择一个条目 | 滚动一行 |
| Enter | 在其章节处打开 | 滚动一行 |
| PgUp、PgDn、Space | - | 滚动一屏 |
| Home、End | - | 顶部、底部 |
| Tab、Shift+Tab | - | 下一个章节、上一个章节 |
| `n`、`p` | - | 下一页、上一页 |
| `/` | - | 返回搜索 |
| Escape | 关闭帮助 | 返回列表 |
| F1、Ctrl+C | 关闭帮助 | 关闭帮助 |

图片显示为它们的图注。帮助读取控制台所运行的二进制，因此在 `--remote` 下它会显示控制台自身的文档，而不是服务器的文档。它使用终端的语言（见[语言](#语言)）；控制台自身的文字保持英文。

## 智能体的工具

当问题涉及 Coddy 本身时，智能体会去查它的文档：如何设置 Telegram 机器人、`config.yaml` 的某个键有什么作用、询问模式允许哪些工具，而不是凭记忆回答：

| 工具 | 参数 | 返回 |
| --- | --- | --- |
| `coddy_docs_search` | `query`（英文、俄文或中文）、`limit`（默认 8，最多 20）、`lang`（`en`、`ru` 或 `zh`） | 最匹配的章节，每个都带有可读取的引用、页面标题和章节标题以及摘要片段 |
| `coddy_docs_read` | `page`（引用、`coddy:` 链接或标题；省略则获取目录）、`offset`、`lang` | 页面或章节，以及它的引用、行范围和公开地址 |

未指定 `lang` 并且在轮次语言中没有找到任何内容的搜索，会尝试查询的语言，因此，在由编辑器以英文启动的轮次中，俄文或中文查询仍然能找到相应语言的章节。

两者都不需要权限，并在智能体、计划和询问模式中提供，也提供给子智能体和内置的 `explore` 子智能体。在聊天中，一次调用会按其行为显示，并在旁边带上查询或页面：Web 界面中显示为 *正在搜索文档* 和 *正在阅读文档*，控制台中显示为 *搜索文档中* 和 *阅读文档中*。页面按约 24 千字符分块读取：长页面会以继续阅读的行和它的章节列表结尾，因此模型读取它需要的章节，而不是整个页面。

当要把用户指向某个页面时，智能体会写提及 `@coddy:<page>#<section>` 或 `coddy:` 链接，而不是地址：两者都会在作出回答的那个二进制自身的阅读器中打开页面。公开地址 `https://coddy.dev/docs/<page>` 用于离开 Coddy 的内容，唯一的命令行写法是二进制所响应的写法，`coddy docs show <page>[#section]` 与 `coddy docs search` 和 `coddy docs list` 并列。

## 提及页面

提示词中的 `@coddy:` 会将一个页面或其中一个章节附加到消息：

```text
set this up for me like @coddy:surfaces/gateway#proxy says, the machine needs socks5
```

页面或章节及其子章节会作为 `<coddy_attachment kind="doc">`（[提及](mentions.md)）随消息传递，最多 64 千字符，使用该轮次的语言（见[语言](#语言)）：较长的参考页面会以其开头部分到达，并带有继续阅读的行和它的章节，模型用 `coddy_docs_read` 读取它需要的部分。自动补全在每个界面都可用：`@coddy:` 列出页面，`@coddy:prox` 按名称查找页面、按词语查找章节，`@coddy:features/mentions#` 列出该页面的章节。标题使用界面的语言；插入的是三种语言共享的地址。

## 从 Shell 使用

```bash
coddy docs                                   # the contents
coddy docs search telegram proxy --limit 3   # the best sections
coddy docs show features/mentions#completion # one section, as Markdown
coddy docs show features/mentions --lang ru  # the same page in Russian
coddy docs show features/mentions --lang zh  # the same page in Chinese
```

`coddy docs` 在每个构建中都可用，包括未包含控制台的构建，且不需要配置。Shell 补全会提供 `coddy docs show` 之后的页面。

## 二进制包含的内容

`docs/nav.yaml` 各组的页面会被嵌入（`docs/embed.go`），包括英文、俄文（`docs/ru/`）和中文（`docs/zh/`）；`docs/plans/` 下的设计记录、`docs/assets/` 下的图片，以及映射保留在仓库根目录的页面（贡献指南、Web 界面设计契约、智能体笔记）不会被嵌入。二进制中会加入英文、俄文和中文三种 Markdown。`docs/` 下的新组文件夹必须加入三种语言的嵌入模式：`internal/docs` 的测试会在二进制三种语言都未包含的映射页面中失败。

实现：`internal/docs`（库、章节、搜索、读取、链接重写；`lang.go` 用于语言，`stem_ru.go` 用于俄文词干提取器，`rank.go` 用于中文双字切分）、`internal/tools/docs_search.go` 和 `docs_read.go`、`internal/session/mentions.go`（`resolveDoc`）和 `mention_search.go`（`docCandidates`）、`external/httpserver/docs_http.go`（`GET /coddy/docs`、`/coddy/docs/page`、`/coddy/docs/search`、[HTTP API](../reference/http-api.md)）、`external/ui/src/ui/docs/`、`external/cli/docs_modal.go` 和 `cmd/coddy/docs.go`。规范：`features/builtin_docs.feature`。

## 相关

- [提及](mentions.md) - 提示词可以指向的其他内容；
- [工具](../reference/tools.md) - 模型可以调用的每个工具；
- [Web 界面](../surfaces/web-ui.md) 和 [控制台](../surfaces/console.md) - 阅读器和帮助所在的界面；
- [编写文档](../contributing/documentation.md) - 页面如何编写、翻译和检查。

<!-- docsgen:source sha256=38469ed830d3850e -->

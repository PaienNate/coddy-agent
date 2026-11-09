# 编写文档

`docs/` 下的文档如何组织，对 Coddy 的改动必须如何在文档中体现，截图、示意图和视频如何出现在页面上，以及哪些检查守护这一切。一句话的规则：用户会注意到的能力，或对页面所描述的任何内容的改动，只有当页面在同一个拉取请求中说明它时，才算完成。

## 布局

| 文件夹 | 存放内容 | 页面类型 |
|--------|------------------|-----------|
| `docs/getting-started/` | 安装、配置、更新、故障排查 | 指南，按顺序读一次 |
| `docs/surfaces/` | 每种与智能体对话的方式一页：控制台、Web 界面、编辑器、Telegram | 界面页面 |
| `docs/operate/` | 将其作为服务运行、远程模式、集群、调度器、安全 | 运维指南 |
| `docs/features/` | 每种能力一页：模式、会话、规则、技能、子智能体、钩子、MCP、后台任务、压缩、记忆、导出、内置文档 | 功能页面 |
| `docs/reference/` | 完整列表：CLI、`config.yaml`、环境变量、斜杠命令、键盘、工具、HTTP API、ACP | 参考页面，在代码是事实来源之处生成 |
| `docs/tutorials/` | 面向任务的指南，每个教程一页 | 教程 |
| `docs/contributing/` | Coddy 如何构建、测试、设计和编写文档 | 贡献者页面 |
| `docs/plans/` | 设计记录，按当时做出的决定记录 | 内部：不在映射中，冻结，不随之后的改名重写 |
| `docs/assets/` | 页面嵌入的内容：截图、视频、品牌文件 | 见[资源索引](../../assets/INDEX.md) |
| `docs/ru/` | 俄语翻译：相同路径上的相同页面，翻译后的映射和 schema 描述 | 见[翻译](#翻译) |
| `docs/zh/` | 简体中文翻译：相同路径上的相同页面，翻译后的映射和 schema 描述 | 见[翻译](#翻译) |

全部内容的映射是 [`docs/nav.yaml`](../../nav.yaml)：每个页面及其分组、标题和一行摘要。[`docs/README.md`](../README.md)（中心页）由它生成，发布在站点根目录的 `llms.txt` 和 `llms-full.txt` 也是如此，文档站点的侧边栏也将如此。不在映射中的页面，对读者和智能体来说就不存在，`make docs-check` 会说明这一点。`docs/plans/` 的设计记录是唯一的例外：它们是内部的，不进入映射。

映射中的页面随二进制一起发布（[内置文档](../features/built-in-docs.md)）：`docs/embed.go` 嵌入 `nav.yaml` 和各分组文件夹的 Markdown，Web 界面的阅读器、控制台的 F1 帮助、`coddy docs` 以及智能体的 `coddy_docs_*` 工具都读取那份副本。由此对写作者有两点要求。`docs/` 下新增的分组文件夹必须加入 `docs/embed.go` 中的模式，否则 `internal/docs` 中的 `TestDefaultCarriesEveryPageOfTheMap` 会失败。而且页面会在仓库之外被阅读：指向其他页面的相对链接在任何地方都有效（它们会变成 `coddy:<slug>#<anchor>`），指向仓库文件的图片和链接会解析到发布标签处的 GitHub，因此图片需要一条能独立成立的图注，离线读者和控制台看到的就是它，而不是图片本身。

页面移动时其地址也随之移动：没有重定向占位页，所以旧链接会失效，coddy.dev 地址（`coddy.dev/docs/<slug>`）跟随映射。离开仓库的链接使用那种形式，二进制、schema 和内置技能会打印当前地址。

## 页面类型

每个页面以 `# <title>` 开头（标题来自 `nav.yaml`），然后是一段引言，说明页面涵盖什么、面向谁，之后是 H2 章节。英文（其翻译在 `docs/ru/` 下为俄语、在 `docs/zh/` 下为简体中文，见[翻译](#翻译)），沿用现有页面的语气，本应使用破折号处使用带空格的普通连字符，带语言标注的围栏代码块，相对链接（`../features/hooks.md`，同一页面的标题用 `#anchor`）。不使用 YAML frontmatter：GitHub 会把它渲染成表格。

**功能页面**（`docs/features/`）：它是什么以及智能体何时使用它；它在各个界面上的样子，在解释它的段落旁附上 Web 界面或控制台的截图；如何配置它，链接 [config.yaml 参考](../reference/config.md#字段参考) 中的字段行，而不是重复它们；涉及的工具、命令或文件，链接相应的参考行；限制和有意缺失的内容；测试所在的位置。

**界面页面**（`docs/surfaces/`）：如何启动它；带截图的布局；该界面提供什么（模式、模型、权限、命令、按键）以及它与其他界面的不同之处；远程使用；截图和测试。

**运维指南**（`docs/operate/`）：命令、配置键、重新加载和重启时会发生什么、安全说明、故障排查指引。

**参考页面**（`docs/reference/`）：一段引言和表格。生成的块位于 `<!-- docsgen:NAME:start -->` 和 `<!-- docsgen:NAME:end -->` 标记之间；它们周围的正文是手写的，并在重新生成后保留。切勿编辑标记内部。

**教程**（`docs/tutorials/<name>.md`）：一个从目标到可用结果的任务。页面说明目标和它假设的环境，用可复制的确切命令和 YAML 为步骤编号，以一个证明结果的检查结束，并列出容易出错的地方；它链接到功能页面了解原因，链接到参考了解细节。

## 改动必须包含什么

- **新能力**要在 `features/`、`surfaces/` 或 `operate/` 下有一页（或所属页面中拥有该领域的一个章节），在 `nav.yaml` 中有一条目，在列出同类内容的参考中有一行（工具对应 `tools.md`，命令对应 `slash-commands.md`，变量对应 `environment-variables.md`，键对应 `keyboard.md`，键通过 schema 描述对应生成的 `config.md`），如果是新用户选择 Coddy 的理由之一，还要在 README 的 "What it does" 列表中有相应一行。
- **改变的行为**要更新描述它的页面。用 `git grep -n '<key or command>' docs/` 找到它；旧文本会一直传授给下一个读者，无论人或智能体，直到它消失。
- **Web 界面或控制台功能**要展示出来，而不只是描述：按下面的截图配方，在解释段落旁嵌入截图。改变截图所展示内容的改动要重新截图。
- **工作跨越多个部分的模块** - 步骤流水线、生命周期、诸如提供商提示词缓存的不变量、跨包或跨进程的数据 - 要在描述它的页面上用示意图展示，对示意图所展示内容的改动要重新渲染它（[示意图](#示意图)）。
- **移除的功能**要连同它的页面、它在 `nav.yaml` 中的条目和它的资源一起移除；不留下重定向占位页。
- **重命名的键、命令或标志**要按配对工作流规则的步骤 9（[Cursor](../../../.cursor/rules/workflow.mdc)、[Claude Code](../../../.claude/rules/workflow.md)）所述，扫过整棵树，包括文档。

配对工作流规则承载与步骤 10 相同的契约。Cursor 和 Claude Code 读取它们原生的表示；Codex、OpenCode 和 ZCode 项目适配器附加 Cursor 表示，并由 `make test-agent-rules` 验证。

## 翻译

文档被翻译成俄语和简体中文。`docs/ru/<group>/<page>.md` 和 `docs/zh/<group>/<page>.md` 分别是 `docs/<group>/<page>.md` 的翻译，名称和路径相同，各自语言目录下的 `README.md` 翻译中心页。`docs/<lang>/nav.yaml` 保存每个分组和页面的翻译标题与摘要，`docs/<lang>/config-descriptions.yaml` 保存该语言字段表所打印的配置 schema 的翻译描述；两者的每个条目都在 `en:` 下保留它所翻译的英文文本。设计记录、资源以及映射保留在仓库根目录的页面不翻译：翻译页面嵌入相同的截图。每个界面都会显示它认为读者使用的语言（[内置文档](../features/built-in-docs.md#语言)），因此落后于其英文页面的翻译是有人会读到的缺陷。

对英文页面的改动要在同一个拉取请求中把同样的改动带入其翻译：

- 翻译保留其源文件的标题，数量相同、顺序相同、层级相同，因为一个地址通过其位置在三种语言中命名同一个章节；
- 代码块、行内代码、命令、键、路径和程序输出保持原样，Web 界面的控件用对应语言的词典标签命名（`external/ui/src/ui/i18n/messages/ru.ts` 或 `zh.ts`）；
- 链接按英文页面的写法复制：`make docs` 使它们适配翻译所在文件夹，为资源或仓库文件加上额外的 `../`，并把英文标题的锚点转换为翻译标题的锚点；
- 一旦翻译与英文页面所说一致，`make docs-stamp PAGES="docs/<lang>/<group>/<page>.md"` 就在翻译的最后一行 `<!-- docsgen:source sha256=... -->` 记录英文页面的摘要。该戳记会拒绝自 `HEAD` 以来未改变的翻译；当英文改动无需翻译时（例如修正的拼写错误），`UNCHANGED=1` 会接受它。

新的或重写的翻译正文在盖戳之前，由未撰写它的审阅者阅读。一份包含每个改动段落原文与译文的简报会发给 `/crossreview`，或另一个至少三名彼此保持盲审的审阅者组成的法定人数；主持审阅的人对照英文页面检查每项发现并独自决定，第二轮、必要时第三轮且绝不再多，阅读重写的段落。步骤在翻译规则中的 *Cross-review of a translation* 下。

新页面带来它在 `docs/ru/nav.yaml` 和 `docs/zh/nav.yaml` 中的条目及对应翻译；新的或更改的 schema 描述带来它在 `docs/ru/config-descriptions.yaml` 和 `docs/zh/config-descriptions.yaml` 中的条目。`make docs-check` 会在以下情况失败：缺少翻译；`docs/ru/` 中某个文件没有翻译任何内容；翻译的标题、摘要或描述落后于其英文文本；标题与源文件不同；戳记落后于英文页面（附带显示自那时起其中有何变化的命令）；以及规则禁止的俄语排版：长破折号或短破折号、法式引号、弯引号、水平分隔线、俄语措辞规则禁止的词、图片的替代文本仍为英文。提交前钩子把过时的戳记报告为警告，CI 报告为错误，因此英文页面的提交可以先于其翻译，而拉取请求不可以。

俄语和中文的写作规则、术语表以及命令都在配对文档翻译规则中（[Cursor](../../../.cursor/rules/docs-translation.mdc)、[Claude Code](../../../.claude/rules/docs-translation.md)）。

## 截图

截取真实界面，绝不用效果图，也绝不用不再匹配的旧图片。

- **Web 界面**：`make build TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop"`，在一次性 `CODDY_HOME` 上以可展示的路径运行 `coddy serve`，默认使用 Playwright 以 1280x800 在深色主题下截图，页面讨论窄布局时宽 390，讨论主题时用浅色。绝不要截取提供商详情面板：它会完整渲染 `api_key` 值。`examples/` 框架和配对 UI 验证规则（[Cursor](../../../.cursor/rules/ui-verification.mdc)、[Claude Code](../../../.claude/rules/ui-verification.md)）描述了这套装置。
- **控制台**：在隔离的 Xvfb 显示上的真实 Konsole 窗口，用 XTEST 驱动（demo-videos 仓库中的 `demo-videos/rig/stage_konsole.sh`），裁剪到使用的行。真实模型无法按需重现的状态用另一种方式截取，不使用 Konsole：脚本在 pty 中驱动 `coddy cli`，用 **pyte** 模拟屏幕，并通过无头 chromium 把单元格网格渲染成 PNG，自己架设 HTTP 端点，因此既不需要提供商也不需要密钥 - `examples/cli/capture_usage.py` 用于用量页脚，`examples/cli/capture_subagent.py` 用于加载了技能的委派运行，`examples/cli/capture_tasks.py` 用于运行中轮次的状态行和 `/tasks` 覆盖层，`examples/cli/capture_settings.py` 用于会话设置命令之后的页脚。
- **命名与放置**：页面需要不止一两张图片时用 `docs/assets/<page-or-feature>/<state>-<theme>-<width>.png`，否则放在顶层的 `docs/assets/<feature>-<state>-<theme>-<width>.png`。短横线命名，不含问题或拉取请求编号，小于 1 MB。
- **在页面上**：一行图片后跟一行斜体图注，放在解释图片所展示内容的段落旁。每个视图和状态一张图片；页面比较两者时用并排两张图片的表格。

  ```markdown
  ![The Tasks panel with a subagent run in progress](../assets/subagents/tasks-panel-agent-running-dark.png)

  *The Tasks panel with a subagent run in progress*
  ```

作为拉取请求证据拍摄的截图（前后对比、每种主题、每种宽度）属于拉取请求：把它们拖进 GitHub 上的描述，GitHub 会将它们存到 `user-attachments` 下，或者推送到孤儿分支 `screenshots` 并链接原始文件。它们绝不放在 `docs/assets/` 下。

## 示意图

描述工作跨越多个部分的模块（步骤流水线、生命周期、诸如提供商提示词缓存的不变量、跨包或跨进程的数据）的页面，在正文旁用示意图展示它。[摘要形成链](../features/compaction.md#摘要构成一条链) 和 [系统提示词结构](react-agent.md#系统提示词结构) 是例子。看到各部分及其间箭头的读者能一遍读懂正文，同一张图也能放到幻灯片上。

- **源文件与图片。** 示意图是 `docs/assets/diagrams/` 下链接共享 `diagram.css` 和 `diagram.js` 的 HTML 页面，以及由它渲染出的同名 PNG。文档页面嵌入该 PNG。改动时编辑 HTML 并运行 `make diagrams`，它需要无头 Chromium 和 Noto Sans 字体（`NAMES="compaction-chain"` 渲染一张示意图），然后提交两个文件。`make docs-check` 会在图片缺少源文件以及源文件缺少图片时失败；它无法判断图片比源文件旧，所以每次编辑后都要渲染。
- **外观。** 每张示意图都遵循 `compaction-chain.html`：一个标题、一行说明图片展示什么、编号面板、在有帮助处使用真实名称（配置键、文件、函数、路由），以及说明原因的简短注释。`diagram.css` 固定每张示意图中每种颜色的含义，`diagram.js` 绘制方框之间的箭头。图片以 CSS 尺寸的两倍渲染，并保持在 1 MB 以下。
- **语言。** 标签为英文。俄语页面嵌入相同的 PNG，并翻译其替代文本和图注。
- **准确性。** 示意图像正文一样做出论断。每一张都对照代码检查，而不仅对照页面；对示意图所展示行为的改动要编辑其源文件，并在同一个拉取请求中重新渲染。
- **不用 Mermaid。** 文档示意图不写成 Mermaid 围栏：Mermaid 自行排版图片，方框旁没有空间放注释，而且围栏无法放到幻灯片上。
- **在页面上。** 图片放在解释同一内容的段落旁，带有替代文本和一行斜体图注，与截图一样。

## 视频

`docs/assets/video/` 下的六个演示（控制台、Web 界面、通过 ACP 的 Zed 和 VS Code、集群、作为集群节点的 Android 手机）是用 demo-videos 仓库中的装置录制的：在 Xvfb 上用 XTEST 脚本拍摄，用 `post.py` 做后期制作（缩放、字幕、快捷键徽章、片头和片尾），然后为仓库重新编码：

```bash
ffmpeg -i take.mp4 -vf "scale=1280:-2" -c:v libx264 -preset slow -crf 31 -pix_fmt yuv420p -movflags +faststart -an docs/assets/video/<name>.mp4
```

视频保持在 4 MB 以下、四分钟以内；功能片段二十秒比两分钟更好。展示已变更界面的视频要重新录制或移除。

GitHub 只为作为 GitHub 附件上传的文件渲染内嵌播放器：提交到仓库的文件在其文件页上仍是一个链接，`<video>` 标签会被清理器丢弃。因此页面携带两样东西：单独一行的附件 URL（`https://github.com/user-attachments/assets/<uuid>`），GitHub 会把它变成播放器；其下是一行斜体图注，带有指向 `docs/assets/video/` 中副本的链接，这是资源检查所统计的，也是文档站点将嵌入的。要获得该 URL，把文件拖进本仓库任意问题或拉取请求的评论框，等待上传完成，复制它插入的 URL，然后发布评论：附件在为其上传的评论公开之前一直保持私有，因此未发布的实际上传只对上传者播放，对其他人都不播放。附件可以是同一次拍摄的带解说剪辑（画外音和背景音乐）；`docs/assets/video/` 中的副本保持无声，大小限制适用于它。

## 资源索引

[`docs/assets/INDEX.md`](../../assets/INDEX.md) 说明文件夹中应放什么、截图配方和品牌文件，并带有生成的清单：每个资源及其大小和引用它的文件。`make docs-check` 会在资源无人引用时失败，因此文件要么被页面、README、`DESIGN.md`、`Dockerfile` 或截图脚本使用，要么被删除。

## 生成的页面

`internal/docsgen`（通过 `cmd/docsgen` 运行）从 `nav.yaml` 生成嵌入式配置 schema 以及完整标签二进制的 `--help` 屏幕：

| 文件 | 块 | 来源 |
|------|-------|--------|
| `docs/README.md` | `docsgen:nav` | `nav.yaml`：分组、页面、摘要 |
| `llms.txt`、`llms-full.txt`（站点根目录，**不**保存在本仓库中） | 整个文件 | `nav.yaml` 和页面本身；`docs/plans/` 在映射之外，因此两个文件都不包含设计记录 |
| `docs/reference/config.md` | `docsgen:config` | `internal/config/config.schema.json` 描述和加载器的默认值（`config.DocDefaults`） |
| `docs/reference/cli.md` | `docsgen:cli` | `coddy --help` 以及每个带标志集的命令的 `--help` |
| `docs/assets/INDEX.md` | `docsgen:assets` | `docs/assets` 下的文件及其引用 |
| `docs/ru/README.md`、`docs/ru/reference/config.md`、`docs/ru/reference/cli.md` | 相同的块 | `docs/ru/nav.yaml`、带 `docs/ru/config-descriptions.yaml` 的 schema、相同的 `--help` 屏幕 |
| `docs/ru/` 下每个页面的链接 | 仅目标 | 英文页面的链接经过调整以适应翻译所在文件夹 |

```bash
make docs            # regenerate everything (builds the full-tag binary first, which needs Node for the ui tag)
make docs-fast       # everything but the CLI reference, no binary and no Node needed
make docs-check      # regenerate into memory and fail on drift, missing pages, broken links, unused assets
make docs-stamp PAGES="docs/ru/features/mcp.md"  # record that a translation follows its English page
```

`make docs-check` 在 CI 中作为 **Documentation** 作业运行，并在触及 `docs/`、README、`AGENTS.md`、`DESIGN.md`、`CONTRIBUTING.md` 或配置 schema 的提交的提交前钩子中运行（在那里不构建 CLI：`go run ./cmd/docsgen -skip-cli`）。链接检查按 GitHub 风格对照目标页面的标题解析每个相对链接和图片以及每个 `#anchor`；围栏代码块被忽略。

对 `docs/` 的遍历会询问 git 排除了什么并跳过它，因此 `.gitignore` 保持在本仓库之外的文件夹既不是映射中缺失的页面，也不是失效链接的来源 - `docs/` 下的本地草稿没有代价。只有未跟踪的文件才算数，因此排除规则碰巧匹配的已跟踪页面仍然是页面，`docs/plans/` 继续通过它自己的规则而非这条规则离开映射。没有 git 或在仓库之外时，不进行过滤，整棵树像以前一样被遍历。

## 站点

映射中的每个页面在 coddy.dev 上都有稳定地址，因此二进制、schema、内置技能、文章和其他人的链接可以在不指出本仓库路径的情况下命名一个页面，而且那里没有任何重复：

| 地址 | 会发生什么 |
|---------|--------------|
| `https://coddy.dev/docs/<slug>` | 访问者落在 GitHub 上的该页面；`#fragment` 在跳转后保留 |
| `https://coddy.dev/docs/<slug>.md` | main 分支上该页面的原始 Markdown |
| `https://coddy.dev/docs/` | 中心页，GitHub 上的 `docs/README.md` |
| `https://coddy.dev/llms.txt`、`https://coddy.dev/llms-full.txt` | 每次运行由 `nav.yaml` 和页面构建，直接写入站点检出目录；索引链接 `main` 上的原始 Markdown |
| `https://coddy.dev/ru/docs/<slug>`、`.md`、`https://coddy.dev/ru/docs/` | 对 `docs/ru/` 下的俄语页面同理；地址中的英文锚点通过 `docs-anchors-ru.json` 转换为俄语锚点 |
| `https://coddy.dev/ru/llms.txt`、`https://coddy.dev/ru/llms-full.txt` | 索引和俄语页面的拼接 |
| `https://coddy.dev/config.schema.json` | 配置 schema，由 `make site-schema` 发布 |

slug 是页面在 `docs/` 下的路径，去掉 `.md`（`getting-started/install`、`reference/config`）；`docs/` 之外的页面以其文件名命名（`CONTRIBUTING`）。GitHub Pages 没有服务端重定向，但它会为每个不是文件的地址提供 `404.html`，因此站点的 `404.html` 加载 `docs-redirect.js`，这是一个读取路径并转发浏览器的生成脚本（`window.coddyDocsTarget` 是纯映射，无需导航即可测试）。响应带有 404 状态，这对人来说没问题，对智能体则无关紧要，它们从 `llms.txt` 获得原始地址；站点前面 Cloudflare 区域上的重定向规则把同一映射变成真正的 301（Rules、Redirect Rules、dynamic：当路径以 `/docs/` 开头时，以状态 301 重定向到 `concat("https://github.com/coddy-project/coddy-agent/blob/main/docs/", substring(http.request.uri.path, 6), ".md")`，第二条规则针对 `.md` 后缀指向 raw.githubusercontent.com）。`internal/docsgen` 从 `nav.yaml` 渲染该层：

```bash
make site-docs        # write docs-redirect.js, docs-anchors-ru.json and both languages' llms files into the site checkout beside this one (SITE_REPO=... if elsewhere)
make site-docs-check  # report drift without writing
```

<!-- docsgen:NAME:start -->

<!-- docsgen:NAME:end -->

<!-- docsgen:source sha256=... -->

`llms.txt` 和 `llms-full.txt` **不保存在本仓库中**。它们是每个页面的拼接，每次运行都会重建，因此索引中的副本在触及任何页面的每个分支中都会冲突 - 两个人编辑两个不同的页面都会重新生成同一个文件。`make docs` 只把它们渲染到站点层；`.gitignore` 覆盖旧路径，以防过时的本地副本回来。

站点仓库是 `coddy-project.github.io`。对页面、`nav.yaml` 或 schema 的改动随同一个拉取请求在那里发布；站点提交在 coddy-agent 改动合并之后进行，因为 `llms.txt` 按页面在 `main` 上的路径命名页面。离开仓库的链接使用 `coddy.dev/docs/<slug>` 形式，绝不使用 GitHub 路径。

## 设计记录

`docs/plans/` 保存较大功能背后的计划和决定，按撰写时的样子，包括日期等一切。阅读它们是为了了解原因，而不是做法，当代码演进时它们不会被重写：别处的改名不会动它们，只有当页面移动时才修复链接目标。新计划是以功能命名的新文件；当计划带有日期时，该日期是做出决定的那一天。

它们是内部的，是 `docs/` 中位于 `nav.yaml` 之外的唯一部分：中心页不提供它们，`llms.txt` 和 `llms-full.txt` 都不包含它们。指向记录的页面直接链接它（`../plans/hooks.md`），这些链接像其他链接一样被检查。

## Web 界面设计契约

仓库根目录的 [`DESIGN.md`](../../../DESIGN.md) 是嵌入式 SPA 的视觉和行为契约：令牌、布局、组件行为、本地化、开发工作流。改变组件外观或行为的对 `external/ui/src/` 的改动要在同一改动中更新 `DESIGN.md`，当用户能看到差异时还要更新界面页面 [Web 界面](../surfaces/web-ui.md)。品牌文件（徽标、favicon、社交预览）在资源索引中描述。

<!-- docsgen:source sha256=7f9711f13417ad66 -->

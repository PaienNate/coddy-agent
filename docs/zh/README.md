# Coddy 文档

Coddy 是一个封装在单个静态 Go 二进制中的通用智能体：一个带有文件系统和 shell 工具的 ReAct 循环，MCP 服务器、项目规则、技能、子智能体、钩子、后台任务、cron 调度器和长期记忆，可从终端控制台、带 OpenAI 兼容 HTTP API 的内嵌 Web 界面、通过 Agent Client Protocol 接入的编辑器，或 Telegram 机器人驱动。每个界面共享 `~/.coddy` 下的同一批会话。

初次接触？先读[快速开始](getting-started/quickstart.md)，然后读你所使用界面的页面。本导航由 `make docs` 从 [nav.yaml](../nav.yaml) 生成；同一份列表还供 <https://coddy.dev/llms.txt> 和 <https://coddy.dev/llms-full.txt> 使用，面向阅读文档的智能体，每次运行都会构建，并从站点发布，而不是保留在此处。页面如何组织，以及对 Coddy 的改动必须如何带入这些页面，见[编写文档](contributing/documentation.md)。

<!-- docsgen:nav:start -->
## 入门

安装 Coddy，为它配置模型，完成首次运行，并保持更新。

- [快速开始](getting-started/quickstart.md) - 在五分钟内从全新安装到获得第一个回答，涵盖控制台、浏览器和编辑器。
- [安装](getting-started/install.md) - 单行安装器、发布归档、Linux .deb 和 .rpm 包、Homebrew、Android 上的 Termux、Windows 路径、手动放置。
- [配置](getting-started/configuration.md) - config.yaml 的位置，如何用 -t 和 --dry-run 检查它，提供商和模型，SSH 远程执行，.env 文件。
- [更新](getting-started/update.md) - coddy update、发布资产、由包管理器接管的安装，以及变更报告。
- [Docker](getting-started/docker.md) - GHCR 镜像、docker compose、卷和环境变量、端口 12345 上内置的 Web 界面。
- [Homebrew](getting-started/homebrew.md) - cask 与 formula 的对比，哪个 Homebrew 仓库承接什么，homebrew/core 提交。
- [Android（Termux）](getting-started/android.md) - 面向 Termux 的 Android 构建，为什么 Linux 归档无法在那里启动，以及 Coddy 在设备上适配了什么。
- [故障排除](getting-started/troubleshooting.md) - 当二进制不在 PATH 中、配置无法加载、提供商拒绝密钥、端口被占用或构建缺少某个界面时，应检查什么。

## 界面

在终端、浏览器、编辑器或即时通讯工具中，使用同一个智能体和相同的会话。

- [控制台（TUI）](surfaces/console.md) - 在终端中直接运行 coddy，布局、按键、斜杠命令、本地 shell、打印模式和远程模式。
- [Web 界面](surfaces/web-ui.md) - 由 coddy serve 提供的嵌入式单页应用，包含会话、输入框、模式和模型、附件、设置、主题和语言。
- [桌面窗口](surfaces/desktop.md) - coddy desktop，将 Web 界面作为应用窗口、聊天作为标签页，记住的端口、按键，以及各平台为窗口提供的功能。
- [Android 应用](surfaces/android.md) - 在手机上运行完整服务器的 APK：环回 WebView、内置 BusyBox、带测试按钮的 SSH 凭据，以及构建产物。
- [编辑器（ACP）](surfaces/editors.md) - Zed、VS Code、Obsidian 和脚本作为 coddy acp 的 ACP 客户端，以及它们与其他界面共享的内容。
- [Telegram 网关](surfaces/gateway.md) - Telegram 机器人适配器、设置、访问级别、会话隔离、富消息、在浏览器中实时查看同一聊天，以及编写新适配器。
- [Pachca 网关](surfaces/pachca.md) - Pachca (Пачка) 集成机器人、在工作区中的设置、事件历史轮询、群组、话题和私聊、命令、限制。

## 运维

将 Coddy 作为服务运行，从其他地方访问它，并限制它可以执行的操作。

- [coddy serve 与守护进程](operate/serve.md) - 每个启用的子系统共用一个进程，systemd 用户服务（serve install 和 uninstall），带 status、stop 和 restart 的 --daemon，以及配置重载。
- [监控](operate/monitoring.md) - coddy serve 的 Prometheus 导出器，在容器中启用和绑定它、抓取配置、每个指标、随附的告警规则，以及永远不会被导出的内容。
- [远程模式](operate/remote.md) - 通过 --remote 从控制台、ACP 或 Web 界面驱动远程 coddy serve，令牌、CORS 和环境菜单。
- [集群](operate/swarm.md) - 中继和节点、挂载、聚合会话列表、环和路由、反向隧道。
- [调度器](operate/scheduler.md) - Cron 任务文件、UTC 触发规则、作为任务会话下的后台智能体任务运行、运行面板、调度器工具和 REST。
- [安全与信任](operate/security.md) - 智能体可以执行什么以及如何限制它，权限模式，对 MCP 服务器、钩子和子智能体的项目信任，令牌和 CORS，以及哪些内容未沙箱化。

## 功能

智能体能做什么，以及每项能力如何配置。

- [运行模式](features/modes.md) - agent、plan 和 ask，每种模式允许哪些工具，以及如何在每个界面上切换。
- [会话设置](features/session-settings.md) - 在对话中通过 /model、/reasoning、/think、/nothink、/agent、/plan、/ask 和 /permissions 切换模型、推理级别、模式和权限模式，可作用于当前会话或后续轮次，还包括权限对话框中的会话开关，以及模型切换本身。
- [会话目标与监督器](features/session-supervisor.md) - /goal 让智能体持续工作，直到第二个模型根据证据判断目标已达成，并由只读验证器确认；还包括续接、预算、针对停滞和循环的看门狗，以及阻止 docker 和迁移变更的风险门控。
- [会话](features/sessions.md) - 磁盘上的会话包、恢复、编辑消息时回退历史、待办列表、sessions CLI。
- [Git worktree](features/worktrees.md) - 从 origin 创建新功能分支、会话移动、Web 输入框和项目分组。
- [规则与指令](features/rules.md) - 来自 .coddy、.agents、.cursor、.claude 和 .codex 文件夹的规则，智能体进入某文件夹时的 AGENTS.md 和 DESIGN.md，你自己的配对和智能体主目录中的规则，指令文件，按扩展名区分的方言，激活方式。
- [技能](features/skills.md) - 作为斜杠命令的 SKILL.md 包、skills.dirs、注册表，以及 plugin 命令。
- [子智能体](features/subagents.md) - spawn_agent、定义文件、内置项、项目信任凭据、能力收窄、子会话。
- [钩子](features/hooks.md) - 采用 Claude Code 的 hooks.json 形式的生命周期钩子、事件、匹配器、stdin 载荷、退出码和 JSON 回答、项目信任。
- [MCP 服务器](features/mcp.md) - 通过 stdio、可流式 HTTP 和 SSE 连接 MCP 服务器、mcp.json 文件、工作区信任、启用开关、管理 API。
- [Kimi](features/kimi.md) - Kimi Code 订阅和 Platform 配置、区域端点、CLI 凭据、用量和安全。
- [Devin](features/devin.md) - 使用 Devin (Cognition) 账户的模型：浏览器登录或 Devin CLI 登录，每个家族一个模型，推理级别映射到其变体，以及会话令牌来自哪里。
- [OpenCode](features/opencode.md) - 使用 OpenCode Go 和 Console 网关，通过账户登录或服务账户密钥：模型协议、客户端标头和网关选择。
- [作为模型的 ACP 智能体](features/acp-provider.md) - 将本地 ACP 智能体（Claude Code、Codex、OpenCode、ZCode、另一个 Coddy）作为模型提供：acp 块、每个 Coddy 会话对应一个智能体会话、其权限请求、plan 和 ask 模式、智能体进程。
- [Web 搜索](features/web-search.md) - websearch 会询问哪些搜索引擎，被阻止的后端如何报告，而不是被计为没有找到结果，相关性门控，以及将其指向你自己的 SearXNG。
- [HTTP 请求](features/http-requests.md) - http_request 工具：任意方法、标头、正文、多部分上传和下载、每次请求的代理和证书检查、配置为每个请求添加的标头，以及显示请求发往何处和携带什么的权限提示。
- [Coddy for Browser](features/browser.md) - 通过 Coddy for Browser 扩展，将你自己的浏览器（Chrome、Yandex Browser 或 Firefox）作为智能体的双手：browser 工具、标签组、网站批准、文件、侧边栏中的 Web 界面、其他进程和集群的聊天。
- [提及](features/mentions.md) - 在提示词中用 @ 指向对象：磁盘任意位置的文件、行范围、文件夹、其他会话、规则、子智能体、网页，在消息中一次性解析，并在每个界面上提供补全。
- [图像](features/images.md) - 模型查看图片：为标记为多模态的模型读取 PNG、JPEG、GIF 或 WebP 文件、MCP 工具返回的截图、限制、发送给提供商的内容，以及 Web 界面和 Telegram 聊天中的预览。
- [粘贴并标注图片](features/picture-markup.md) - 将复制的图片粘贴到提示词中：控制台中的 ctrl+v、Web 界面中的粘贴，以及首先打开的标注编辑器，支持铅笔、形状、箭头、文本、裁剪、旋转和撤销。
- [内置文档](features/built-in-docs.md) - 二进制中的这份文档：Web 界面的阅读器、控制台中的 F1、coddy docs、智能体的 coddy_docs 工具和 @coddy:<page> 提及，使用 BM25 搜索，不涉及网站。
- [消息队列](features/message-queue.md) - 在智能体工作时编写后续消息、运行中的轮次何时读取它、撤回一条消息、输入框和控制台界面、队列路由。
- [后台任务](features/background-tasks.md) - 分离的命令、任务池、超时、接管长时间前台命令、程序级权限授予。
- [预览服务器](features/preview-server.md) - 在空闲的 localhost 端口上提供项目目录，以便在浏览器中打开 HTML、JS 和 CSS 工作成果；服务器作为后台任务，它拒绝提供的内容，绑定主机。
- [画布](features/canvas.md) - Web 界面文件窗口中 .canvas 面板上的实时页面：canvas_show 和 canvas_inspect、指向元素请求更改、框架运行的沙箱，以及每块面板都是工作区的 Obsidian .canvas 文件，按 Obsidian 写入的方式读写。
- [上下文压缩](features/compaction.md) - /compact 和在阈值处自动总结、保留的近期轮次、使用 keep_result 的结果逐出。
- [长期记忆](features/memory.md) - 模型通过 spawn_agent 委托的按需记忆子智能体，它回忆和保存什么、它的工具、存储布局、配置和成本。
- [命令安全（decisions）](features/decisions.md) - NeuralDeep decisions 检查在权限门控之前询问 shell 命令是否安全：安全命令无需提示即可运行，不安全命令在每种模式下都会被询问，包括 bypass 模式，而在无人能回答时会被拒绝。
- [会话导出](features/session-export.md) - /export 和 coddy sessions export、格式、路径规则、裁剪选项、JSON 文档。

## 参考

完整列表，在代码作为事实来源之处从代码生成。

- [CLI 参考](reference/cli.md) - coddy 二进制的每个命令、动词和标志，由 --help 生成。
- [config.yaml 参考](reference/config.md) - config.yaml 的每个字段及其类型、默认值和描述，由 schema 生成，外加自配置工具。
- [环境变量](reference/environment-variables.md) - 二进制读取的每个变量、它覆盖什么，以及在哪里有文档记录。
- [斜杠命令](reference/slash-commands.md) - 每个界面上的内置命令，以及技能如何成为命令。
- [键盘](reference/keyboard.md) - 控制台 TUI 和 Web 界面输入框的按键。
- [工具](reference/tools.md) - 模型可以调用的内置工具、它们的参数、权限，以及公开它们的模式。
- [HTTP API](reference/http-api.md) - OpenAI 兼容端点和 /coddy REST 界面、身份验证、会话和标头、OpenAPI 文档。
- [ACP 协议](reference/acp-protocol.md) - coddy acp 如何实现 Agent Client Protocol、方法、通知、权限和问题请求。
- [浏览器协议](reference/browser-protocol.md) - coddy serve 与 Coddy for Browser 扩展之间的 JSON-RPC 通信：握手、标签页、命令、网站批准、文件传输、侧边栏，以及供其他 Coddy 进程使用的 API。

## 教程

面向任务的指南，每篇都是从目标到可用结果的完整路径，包含可复制的配置和命令、验证是否成功的检查，以及常见出错点。

- [CI 中的 Coddy](tutorials/coddy-in-ci.md) - 一个在每个拉取请求上运行 coddy -p 的工作流文件，密钥放在 secret 中，答案放在任务日志中。
- [团队 Telegram 机器人](tutorials/telegram-bot-for-a-team.md) - 使用 coddy serve 为团队提供一个机器人：令牌、谁可以与它对话、群组隔离，以及在浏览器中实时查看同一聊天。
- [为智能体设置项目](tutorials/project-set-up-for-agents.md) - 一个教会任何智能体如何工作的仓库：规则、AGENTS.md、技能、一个 MCP 服务器和信任凭据。
- [从笔记本电脑驱动服务器](tutorials/server-driven-from-a-laptop.md) - 在拥有模型和工作区的机器上运行 coddy serve，通过控制台、编辑器和浏览器从笔记本电脑驱动。
- [在 VS Code Copilot 中将 Coddy 作为模型](tutorials/coddy-as-a-model-in-vs-code.md) - 将运行中的 coddy serve 注册到 chatLanguageModels.json，使其模型和智能体出现在 Copilot 的模型选择器中，每个请求一个会话，并受智能体的权限门控。
- [你自己的技能](tutorials/a-skill-of-your-own.md) - 一个 SKILL.md 如何在每个界面上成为斜杠命令，从第一个文件到注册表安装。
- [跨机器的集群](tutorials/swarm-across-machines.md) - 多台机器上的工作节点加入同一个中继，笔记本电脑通过自己的 Web 界面和控制台驱动任意节点，中继及其智能体出现在环境菜单中，地图一键可达。
- [Docker 中的中继和节点](tutorials/swarm-relay-and-nodes.md) - 一个 Compose 环境，包含中继和主动拨出的节点、令牌、检查、从控制台和浏览器挂载节点，以及通过添加无名称工作节点来扩容。
- [中继链](tutorials/swarm-multi-hop.md) - 第一个中继后面的第二个中继及其自己的节点、两跳挂载、递归会话列表、环和备用节点。
- [使用远程节点](tutorials/swarm-remote-nodes.md) - 从控制台、编辑器和浏览器驱动中继后面的节点，什么在哪里运行，以及哪个凭据打开什么。
- [作为集群节点的 Android 手机](tutorials/swarm-android-nodes.md) - 在 Termux 中运行 Coddy 的手机加入笔记本电脑或服务器上的中继，并从该中继的 Web 界面和控制台进行驱动，手机上无需开放端口。

## 贡献

Coddy 如何构建、测试、编写文档和发布。

- [贡献指南](../../CONTRIBUTING.md) - 开发环境、分支和拉取请求流程，以及每次更改必须携带的内容。
- [编写文档](contributing/documentation.md) - 页面类型、导航地图、截图和视频、资产索引、生成的参考，以及守护它们的检查。
- [架构](contributing/architecture.md) - 系统设计和组件概览、包边界、会话模式、目录结构。
- [从源码构建](contributing/build.md) - 前提条件、make build、TAGS 与 go build -tags、发布二进制和分发包。
- [使用 tgfake 进行集成测试](contributing/tgfake.md) - 机器人测试所针对的假 Telegram Bot API：go.mod 固定的版本、进程内测试套件、端到端脚本、CI 任务，以及迁移到新的 tgfake。
- [AppSec 扫描](contributing/security-scanning.md) - 通过一个脚本在本地和 CI 中运行 Trivy、semgrep 和 govulncheck、严重性门控、固定的 Go 工具链、分类和抑制。
- [自定义工具](contributing/custom-tools.md) - 向注册表添加内置工具、其 schema 和权限接线，并附一个完整示例。
- [ReAct 智能体](contributing/react-agent.md) - 循环设计、系统提示词结构、工具调用契约和特定模式的行为。
- [Web 界面设计](../../DESIGN.md) - 嵌入式 SPA 的设计令牌、布局和组件契约。
- [Codex 钩子](contributing/codex-hooks.md) - Cursor 规则如何到达在此仓库上工作的 Codex CLI 会话。
- [OpenCode 钩子](contributing/opencode-hooks.md) - 将 Cursor 规则确定性地传递到在此仓库上工作的 OpenCode 会话。
- [ZCode 钩子](contributing/zcode-hooks.md) - 将 Cursor 规则确定性地传递到在此仓库上工作的 ZCode 会话。
- [Kimi Code 钩子](contributing/kimi-code-hooks.md) - 将 Cursor 规则确定性地传递到在此仓库上工作的 Kimi Code 会话。
- [语法高亮审计](contributing/syntax-highlighting-audit.md) - NeuralDeep 对八个主题中代码高亮的审计、其夹具以及如何重新运行。
- [智能体说明](../../AGENTS.md) - 面向编码智能体的仓库地图和贡献者说明。
<!-- docsgen:nav:end -->

<!-- docsgen:source sha256=99985c583d8526a2 -->

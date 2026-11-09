# 交互式控制台 TUI (`coddy` / `coddy cli`)

https://github.com/user-attachments/assets/446b3966-1d45-4e5d-99cf-b5b02db3e860

*一段两分钟的 1.3.0 控制台会话录像：头部显示项目的 `AGENTS.md` 与技能，底部显示账户用量，模型选择器，模型读取的一张图片，该功能的 git worktree，一个由监督者检查的 `/goal` 与目标菜单，在轮次运行期间作为引导排入的后续消息，一个 `/commit` 技能，F1 上的内置文档以及 `coddy -c`。该文件在仓库中为 [console.mp4](../../assets/video/console.mp4)。*

控制台界面是一个终端 UI，运行在其他所有界面共用的同一套机制之上：`session.Manager`、智能体运行器以及 ACP 会话更新。智能体侧没有任何内容是控制台专属的 - 这个 TUI 是 ACP、HTTP 和 Telegram 网关之外的第四个 `UpdateSender`。

构建：`make build TAGS=cli`（或任何包含 `cli` 的 tag 集合；推荐的完整二进制为 `make build TAGS="http ui scheduler memory cli gateway swarm metrics browser"`）。在没有该 tag 的构建中，`coddy cli` 说明如何重新构建，而裸 `coddy` 会继续打印用法。

启动：在终端上裸跑 `coddy`（stdin 和 stdout 都必须是 tty - 管道和 CI 会保留用法约定），显式使用 `coddy cli [flags]`，或使用路由到控制台的标志式快捷方式：`coddy -c` 继续此文件夹中最近的会话，`coddy -p "..."` 运行一个非交互式提示词（`coddy -p -` 和 `coddy -i FILE` 从 stdin 或文件读取）。启动在终端进入 raw 模式之前运行：配置、会话存储、技能和规则文件夹，然后是第一帧。技能只从其文件夹读取，从不从 `marketplaces.json` 的市场读取，因此安装数百个技能只需不到一秒，而一个无响应的市场不产生任何开销（[Skills](../features/skills.md#技能何时被读取)）。配置的 MCP 服务器**不**在这条路径上，无论它们由什么启动 - 配置中指定的程序、通过 `npx` 运行的 npm 包、通过 streamable HTTP 或 SSE 连接的远程服务器：它们在控制台绘制完成后在后台连接，无论是新会话还是通过 `coddy -c` 或 `/resume` 恢复的会话，全部同时进行，每个都有 20 秒的上限，底部会统计它们（`MCP 2/5`）直到每个都响应。在此之前发送的提示词会在状态行上等待其工具列表（`Connecting MCP servers`），按 Escape 会像其他步骤一样结束等待。失败或被信任门拦截的服务器会作为可见对话记录中的一行说明一次，并附带应对措施。未按时响应的服务器会在下一个提示词时再试一次 - `npx` 包的首次启动可能因安装而超出上限 - 而因其他原因失败或失败两次的服务器，在其 `/mcp` 中的开关、重新加载或新会话之前不会再被拨号。恢复会话会在对话记录清空后恢复其当前 MCP 通知。全局配置的服务器随控制台启动，在其第一个会话请求它们之前就已启动，并在整个运行期间保持运行，因此 `/new` 和 `/resume` 会发现它们已连接，而不是重新启动它们；项目服务器是其工作区所有会话共用的一個进程（[MCP](../features/mcp.md#共享服务器)）。没有任何东西读取工作区树：嵌套的 `AGENTS.md` 文件按需读取，从工具进入的文件夹开始（`docs/features/rules.md`），因此在主目录中打开的控制台（仅 macOS 的 `~/Library` 就有数十万条目）会立即绘制其帧，而不是看起来卡死。底部的 git 分支出于同样的原因以三秒为上限读取。启动期间第一次 ctrl+c 会取消启动；第二次会以默认方式结束进程，而不是被吞掉。

退出控制台（双击 ctrl+c、ctrl+d、`/quit` 或 `/exit`）会在终端恢复后打印一个恢复提示：

```
session: sess_1a2b3c4d
continue: coddy cli --session-id sess_1a2b3c4d  (or: coddy -c)
```

会话在其第一个提示词时写入磁盘。启动后未发送任何提示词就关闭的控制台 - 或在发送任何内容之前用 `/new` 离开 - 不会在会话根目录中留下文件夹，也不会在 History 中留下行，并且不打印提示：没有可继续的内容。在第一个提示词之前更改的设置（模型、模式、权限模式）保存在内存中，并随它一起写入。


## 登录提供商 (`/login`)

全新安装没有 Coddy 可与之对话的模型提供商：还没有 `config.yaml`（Homebrew、Linux 软件包、手动解包的二进制），或者 `install.sh` 写入了示例配置，其 `openai` 行从 `OPENAI_API_KEY` 读取密钥。当没有任何提供商行可以响应时 - 没有在 `api_key`、`api_key_command` 或其 `NAME_API_KEY` 变量中有密钥、存储的登录、`acp` 智能体，或在 OpenAI 或 Anthropic 兼容行上有自己的 `api_base` - 头部会以警告色说明并指向 `/login`，因该原因失败的轮次后也会跟随同一行。因会话的模型运行在没有密钥或登录的提供商上而失败的轮次后，会有一行指出该提供商及出路：对于可登录的提供商使用 `/login <name>`，对于接受密钥的提供商使用其密钥，对于其他提供商的模型使用 ctrl+l。

![全新安装的首次启动：头部指向 /login](../../assets/screenshot-console-login-hint.png)

*没有 `config.yaml` 的首次启动：警告色的行指向 `/login`，底部不显示模型。*

### 提供商选择器

启动时没有可响应提供商的控制台会在第一个提示词之前自行打开选择器，就像 Web 界面在首次运行时打开自己的选择器一样（[Web UI](web-ui.md#首次运行引导)）。它以相同的顺序提供相同的提供商，打开时所在的行也相同：

| 行 | 是什么 | 控制台要求什么 |
|---|---|---|
| **NeuralDeep** | 任一 hub 上的 API 密钥或账户登录 | 先选 hub，然后设备登录 |
| **OpenCode** | 带账户登录的 OpenCode Go 或 Console | 先选网关，然后设备登录 |
| **OpenAI** | GPT 模型与 OpenAI API | API 密钥 |
| **Anthropic** | Claude 模型 | API 密钥 |
| **Codex** | 通过 OAuth 使用你的 ChatGPT 订阅 | 设备登录，当本机已有登录时则无需操作 |
| **Devin** | 带浏览器登录的 Devin 模型 | 浏览器登录，当本机已有登录时则无需操作 |
| **Kimi** | Kimi Code 订阅或 Kimi Platform API 密钥 | 先选服务与区域，然后设备登录或 API 密钥 |
| **OpenAI-compatible** | Ollama、LM Studio、DeepSeek、Groq 或任何你自己的端点 | 端点地址，然后 API 密钥 |
| **ACP agent** | 本机运行的你自己的智能体：Claude Code、Codex、Gemini | 使用 ACP 通信的命令 |

Kimi 提供全球（`kimi.ai`，默认）和中国（`kimi.com`）两个变体的 Kimi Code 和 Kimi Platform。选择器要求 Platform 密钥。对于订阅，它使用符合条件的现有登录或启动设备登录。具有其他模式或区域的现有 `kimi` 行保持不变。符合条件的现有 Kimi CLI 登录从 `KIMI_CODE_HOME`（默认 `~/.kimi-code`）读取且不做修改；其来源与 Coddy 管理的登录是分开的。提供商请求，包括登录，都使用该行的代理。

![Kimi 服务选择器在控制台中](../../assets/console/kimi-picker-dark-1280.png)

*控制台在两个区域提供两种服务。*

方向键移动，输入过滤，Enter 选中该行，Escape 收起选择器：头部继续指向 `/login`，它会再次打开它，因此它没有任何一次性屏幕的特性。需要登录的行会运行下面的登录；需要密钥的行在其没有自己的端点时会先被询问端点，然后要求密钥；ACP 智能体只被要求提供启动它的命令，因为智能体自己登录，Coddy 不为其存储凭证。密钥提示处的空回答不会更改任何凭证：已经持有 `config.yaml` 的行会保留其 `api_key` 或其 `api_key_command`，而两者都没有的行会读取其 `NAME_API_KEY` 变量，这就是没有密钥的本地服务器的运作方式。然后 Coddy 按照该行的询问方式向提供商询问它提供什么 - 使用端点和它已经携带的凭证，除非设置指定了另一个端点 - 并提供该列表；所选模型成为 `models[]`、会话的模型，以及当之前没有指定任何可运行的模型时成为 `agent.model`。无法访问的提供商会说明原因，改为手动指定模型，因此设置永远不会被列表卡住。

Codex 或 Devin 登录是本机上每个程序的同一个账户，因此选择器在询问任何内容之前会先查找一个：如果 Codex CLI 的 `auth.json` 或 Devin CLI 的凭证已在磁盘上，它会说明该提供商已登录并直接进入模型，除你选择的行和模型外不写入任何内容。另外两种登录属于它们取自的 hub，而选择器刚刚询问了要使用哪个 hub，因此 NeuralDeep 和 OpenCode 每次都会登录。只有登录才算数：`DEVIN_API_KEY` 或 `config.yaml` 中的密钥让该行可以响应，但不算登录，选择器仍会登录。一个 CLI 登录是一个账户，服务于一行，即其类型唯一的行，或在多个共享它时，以该类型命名的行（[同一提供商类型的多个配置文件](../getting-started/configuration.md#同一提供商类型的多个配置文件)）。选择器只写入那一行、`codex` 或 `devin`，因此它找到的登录始终是为它准备的，无论 `config.yaml` 还列出了什么；旁边你自己的行继续为自己登录。

`config.yaml` 已列出的提供商会保留选择器未询问的一切 - 凭证助手、代理、超时 - 因为只写入设置所回答的内容。当选择器等待提供商列出其模型时，`/login` 会说明这一点，而不是打开第二个列表，此时出现的权限或问题提示会保留屏幕：设置停止并说明如何再次启动它。

![全新安装首次启动时的提供商选择器](../../assets/cli-tui/login-provider-picker-dark-1280.png)

*首次启动时的选择器：NeuralDeep 在首位并高亮，八个提供商按 Web 界面显示的相同顺序排列。*

### `/login`

`/login` 随时重新打开该选择器；在其后跟一个名称会跳过选择器并在不离开控制台的情况下运行 `coddy providers login` 的登录：

| 命令 | 登录到 | 方式 |
|---|---|---|
| `/login` | 上面的选择器 | 方向键、Enter、Escape |
| `/login neuraldeep` | NeuralDeep | 设备代码：打开 hub 的页面，确认代码 |
| `/login codex` | Codex（ChatGPT） | 设备代码 |
| `/login devin` | Devin | 浏览器登录；页面会将浏览器送回本机 |
| `/login opencode` | OpenCode | 设备代码，然后在页面上选择组织 |
| `/login <name>` | `config.yaml` 列出的提供商 | 其类型的登录 |

控制台打印要打开的页面和要确认的代码，并在本机有浏览器时在本机浏览器中打开该页面；通过 SSH 时它仅打印地址，供任何浏览器使用，包括手机浏览器。Escape 停止登录。浏览器在另一台机器上运行的 Devin 登录从该浏览器被送回地址完成：将其粘贴到编辑器中并按 Enter。选择器询问要登录哪个 NeuralDeep 部署，俄罗斯或国际镜像；`/login neuraldeep` 采用 `neuraldeep` 行的 `api_base` 指定的那个，当其未指定时采用俄罗斯部署（[NeuralDeep](../getting-started/configuration.md#neuraldeep)）。

登录成功后，控制台将该提供商行及账户提供的每个模型添加到 `config.yaml`，并在未设置时添加到 `agent.model`，与 `coddy providers login` 添加的目录相同。然后它立即加载新配置：底部显示模型，ctrl+l 列出新模型，头部行消失。当 `agent.model` 运行在无法响应的提供商上时 - 示例配置的 `openai` 行密钥未设置 - 登录会将 `agent.model` 移到它登录的提供商，控制台也以同样方式移动会话并说明；在可响应提供商上的会话保留其模型，ctrl+l 或 `/model` 选择新模型之一。`coddy providers login` 以同样方式移动 `agent.model`。Web 界面中的设置则一次添加一个模型，每个旁边有加号（[Web UI](web-ui.md#设置提供商模型)）。其他类型的行，例如 `openai` 或 `anthropic`，接受密钥，`/login` 会说明这一点；`acp` 智能体自行登录。通过 `--remote` 时配置属于服务器，因此 `/login` 会将你送到那里。

![选择器登录后的控制台：hub 的页面和代码、Signed in to NeuralDeep、config.yaml 新增的内容，以及底部显示新模型及其用量](../../assets/cli-tui/login-signed-in-dark-1280.png)

*在全新安装中从选择器选择 NeuralDeep：页面和代码、账户、`config.yaml` 中的提供商及其五个模型，以及底部 `neuraldeep/qwen3.8-27b` 及账户用量。*


## 实时文件草稿

当模型准备 `write`、`edit` 或 `apply_patch` 时，工具卡片显示 **Generating**、收到后的路径、已解码的内容字节和行数，以及原始参数字节。最后一个计数器在编辑流式传输其搜索文本或另一个参数时持续变化。这些是已接收的数量，不是百分比，也不是对最终文件大小的预测。轮次的 token 估算包含到达时的参数片段；调用结束时提供商用量取代该估算，不会再次计算完整的参数。

![一次生成文件参数的写入调用](../../assets/cli-tui/tool-progress-collapsed-dark-1280.png)

*文件尚未写入：模型仍在生成其参数。*

`ctrl+o` 显示草稿的最新六行，上限为 1 KiB。对于 `edit` 这是 `newString`；对于 `apply_patch` 是补丁文本。预览不是已执行的结果，其他工具不暴露参数预览。更新限制为每次调用每秒五次，并有一次最终刷新。相同的元数据通过 ACP 和 HTTP 传输到远程控制台。OpenAI 兼容、Anthropic、Codex 和 Devin 流提供参数片段；没有片段的提供商仍显示工具名称并在完成时报告其参数。

![展开的文件草稿](../../assets/cli-tui/tool-progress-expanded-dark-1280.png)

*限定长度的尾部在生成时更新；终端控制序列会被清理。*

执行仍然等待完整调用和现有权限门。取消未完成的草稿不会写入部分文件。执行后，卡片显示正常结果，`ctrl+o` 加载该结果。

在没有提供商或凭证的情况下捕获此状态：

```bash
make build TAGS=cli
CLI_E2E_COLS=148 CHROME=/path/to/chromium python3 examples/cli/capture_tool_progress.py /tmp/coddy-draft-shots
```


## 视觉模型

布局复刻了 pi 编码智能体的 TUI（pi-mono `b1efcf7d7`，v0.84.2，MIT，Mario Zechner - 见下面的署名说明），颜色来自 coddy SPA 调色板（`external/ui/src/styles/themes/`）；本文档是 coddy 控制台的视觉契约。

从上到下：

- **头部**：`coddy`（粗体强调）+ 暗色版本；一行暗色提示（`escape interrupt · ctrl+c/ctrl+d clear/exit · / commands · ctrl+o more`）；当没有提供商可以响应时，一行警告色指向 `/login`（[登录](#登录提供商-login)）；`[Context]`（会话提示词携带的文档：你的和工作区中存在的 `AGENTS.md` 和 `DESIGN.md`，然后是 `instructions.files` 添加的文件，每个一次；之后，在警告色中，一个 `instructions.files` 条目，其文件会话无法读取，显示为 `<path> (not read: <reason>)`，见 [Rules](../features/rules.md#更多指令文件)）和 `[Skills]`（已加载的技能，捆绑的在前）。`ctrl+o` 展开完整提示列表并添加 `[Rules]` 和 `[MCP]` 章节；`[MCP]` 列出此工作区的会话启动的服务器：`~/.coddy/mcp.json` 中启用的以及项目的 `.coddy/mcp.json` 中信任门允许运行的。
- **对话记录**：用户消息为全宽背景框；助手 markdown（标题、粗体/斜体、行内代码、带边框的 ``` 围栏、`│ ` 引用、列表、制表符表格、OSC 8 链接）；斜体灰色思考块（用 `ctrl+t` 折叠）；工具调用为背景着色的框（待定 → 成功绿色调 / 错误红色调），粗体标题指明调用作用于什么（`read <path>`、`$ command`、`load_skill <skill>`、`Searching the docs <query>`、`Reading the docs <page>`、`spawn_agent <subagent> · <task> · background · timeout 300s`，其中子智能体之后的每个部分仅在调用传递了它时出现），预览上限为 10 行，`... (ctrl+o to expand)` 从 `sessions/<id>/tool_calls/` 读取完整结果。`spawn_agent` 框还携带委派本身：在标题下，暗色斜体，子智能体收到的提示词，在 10 个书面行或 600 个字符中先到者处截断，带有 `... (ctrl+o for the whole prompt)`。子智能体的报告作为框体落在其下方，因此任务和答案作为一个块阅读。
- 生成速度跟随输出 token 数，例如 `1.2k tokens · 9.8 tok/s`。每次模型调用启动一个新窗口：200 个 token，步长 50，然后当调用已写入 500 后，500 个 token，步长 100。在流式传输期间，速度在调用的第 200 个 token 处出现，并在调用的其余部分保持，因此短于该值的调用仅在完成行中显示它。它每五秒刷新一次，包括部分步骤；停滞的流保留最后的值。测量包括文本、推理和工具参数，从第一个输出增量到最后一个，不包括工具执行、权限等待和首个 token 的时间。实时 token 计数是估算值。当轮次完成、取消或失败时，仅当至少一次模型调用有可测量的生成间隔时才保存说明。它将所有调用的总输出 token 与仅测量调用的平均值配对；重新打开会话时恢复它。阻塞（非流式）提供商和 ACP 智能体模型没有可测量的按模型流式边界，因此它们的实时输出计数没有速度。以单个输出块交付的流式调用也没有可测量的生成间隔：其 token 和时间都不贡献于速度平均值，但其输出仍计入轮次总数。没有可测量的调用间隔时，实时输出计数器仍然工作，但不会产生速度或持久化的生成说明。低于 2 的速度使用两位小数，低于 10 使用一位，10 或以上不使用。
- **状态**：轮次运行时的盲文旋转指示器 `⠋⠙⠹...`，80 毫秒，带实时状态行。该行以轮次自身的数字开头：轮次已运行多久、模型在其中生成了多少 token、当前有多少后台任务在运行 - `15m 08s · 13.5k tokens · 1 running task · Thinking…`。在第一个 token 之前，它仅有时钟和短语（`57s · Waiting for the model`），任务仅在运行某些东西时出现。在会话配置的 MCP 服务器仍在连接时发送的提示词会在 `Connecting MCP servers` 上打开，带有自己的计数器，并在它们响应后回到等待模型。token 是智能体的 `turn_progress` 更新：已完成调用的提供商数字加上在途调用的估算，因此计数在答案流式传输时移动；通过 `--remote` 连接的控制台接收相同的更新。然后是当前步骤的短语，对于运行模型之外内容的步骤，有自己的计数器（`2m 05s · 1.2k tokens · Running a command · 45s`, `Running a subagent · 40s`，当 `spawn_agent` 调用在途时）；思考、响应和等待由轮次时钟覆盖。该行携带阶段和**步骤不作用于的任何内容** - 没有命令，没有路径，没有 URL：调用作用于什么由行上方的工具框命名一次，因此每个短语都是独立完整的（`Running a command`, never `Running` 等待命令跟随它）。普通等待随时间升级：`Waiting for the model` → `The model is taking longer than usual`（15 秒）→ `Still no response from the server`（60 秒）。当权限或问题模态打开时，该行显示 `Waiting for your approval` / `Waiting for your answer`，**没有**步骤计数器（没有东西在运行），批准后它返回门控工具并重新开始计数器。短语表位于 `external/cli/status.go`（SPA 的 `liveStatus.ts` 的 Go 孪生）。
- **计划组件**：编辑器上方的当前待办条目（`✓` 完成、`◐` 活动、`○` 待定、`✗` 失败）。
- **编辑器**：全宽 `─` 规则之间的多行输入（当缓冲区持有 `!!` 命令时为绿色）；带粘性列的自动换行感知光标移动；提示历史（在边缘上/下，上限 100）；大量粘贴折叠为 `[paste #N +K lines]` 标记；滚动内容显示 `─── ↑ N more ───` 边框。自动补全：首行的 `/` 命令和任何位置的 `@` 提及，`tab` 强制裸词上的路径补全。`@` 列表使用与 Web 界面相同的搜索（`GET /coddy/mentions`，本地时在进程内）：整个工作区针对输入内容排名，因此名称片段可在树中的任何位置找到文件；索引在提及开始时重建，因此控制台启动后写入的文件会被提供，删除的文件会消失；以 `/`、`~`、`./` 或 `../` 开头的查询会浏览该文件夹，磁盘上任何位置；`@session:`、`@rule:`、`@agent:` 和 `@coddy:`（内置文档的页面）列出这些种类。被截断的列表会在其滚动行上说明，`(3/50 of 1204, type to narrow)`。文件夹或 scheme 行保持列表打开；文件以空格结束提及，当其路径含空格时加引号（带引号的文件夹在光标前关闭其引号，因此即使后面没有文件，文本也会命名它）。提及可以将文件缩小到行范围，`@Dockerfile:21-31` 或 `@f.go#L21-31`，包括绝对路径。在远程模式下，列表来自运行会话的服务器。语法、每种类型附加的内容和限制在 [Mentions](../features/mentions.md) 中。
- **底部**：两行暗色。第一行左侧为 `cwd (git-branch) [• plan|ask] [• MCP N/M] [• N tasks running (/tasks)] [• accept edits|bypass]`，右侧为 `(provider) model [• reasoning] [• tools <preset>]`，只要模型有推理级别就显示推理级别；第二行左侧为 `↑in ↓out N.N%/ctx`（会话的 token 及它填充上下文窗口的多少），在模型下方为下面描述的账户用量。当权限模式不是 `ask` 时，它关闭第一行的左侧，`bypass` 以警告色显示，因此批准一切的会话永远不会看起来像会询问的会话；它是屏幕上会话的模式，`/new` 和 `/resume` 将其与轮次覆盖行一起移到进入的会话。为若干轮次更改的设置会在第二行下方添加一行强调色，`next turn: model x • next 3 turns: reasoning high` (`this turn`，当运行中的轮次持有它时）。运行任务注释在启动任务的轮次结束后仍保留，此时统计它们的状态行已消失。MCP 计数仅在配置的服务器在第一帧后仍在连接时存在 - 已从正在拨号的服务器中连接，持有的项目声明不计入 - 并在每个都响应后离开该行。当第一行放不下时，注释保持完整：路径先让位，然后模型被截断。第二行右侧在活动模型的提供商报告账户用量时显示（目前：`neuraldeep`，从 hub 的 `GET /v1/limits` 读取；`codex`，从 Codex 后端的用量端点读取；`devin`，从席位管理状态 RPC 读取）：`Pro • 3h 3% • week 7% • wallet -1 229 ₽`、计划、每个计量窗口的百分比**已用**，以及 NeuralDeep 钱包密钥的账户自身卢布余额。重置时间不放在底部：`/usage` 以你的时钟打印它们（24 小时内的时刻，一周内的星期，超出则为日期和时间），达到限制时也是如此。NeuralDeep 仅在其大于零时显示日窗口；Codex 和 Devin 显示它们报告的每个窗口，包括零窗口。Codex 窗口按上游持续时间标记（`5h`、`week`，以及诸如 `Fast model · week` 的功能范围条目）；Devin 配额计划读取 `day` 和 `week`（其剩余百分比显示为已用），ACU 计划读取 `acu`，没有可报告内容的来源读取 `quota unavailable`，而不是编造数字。达到 80% 或以上的窗口会变为警告色，对话记录通知会说 `You've used 82% of your NeuralDeep 3h limit · resets 20:59`，每个窗口和周期一次；达到限制时窗口被替换为错误色的 `limit reached (resets 20:59)`（`rate limited (retry in 42s)` for a per-minute block, `key blocked`, `wallet empty` or `account blocked` for the ones no clock lifts) and posts `Usage limit reached` once. With `agent.wait_for_limit_reset` 开启时，遇到重试永远无法覆盖的限制（`429` 命名一个遥远的重置）的轮次会等待它而不是失败：实时状态行读取 `Usage limit reached · resuming at 20:59`，没有运行计数器，底部保留 hub 的数字，限制解除时同一调用再次运行（Esc 像任何轮次一样停止等待）。NeuralDeep 无限制选项（Qwen ∞）上的模型读取 `∞ volume`。被拒绝的密钥读取 `neuraldeep: key rejected, /login neuraldeep`；当 hub 无法访问时，最后的数字保留并带有 `(stale)`。在窄终端上，钱包、日、周和计划按此顺序离开。数字在会话开始时及每轮结束后从会话管理器到达（`provider_usage` 更新，在 `--remote` 上相同）；在 `/model` 后，控制台向其后端请求新提供商的数字（热时来自缓存），窗口重置过后它请求重新读取，当后端按其节奏下限推迟刷新时，控制台在答案如此说明时再次读取缓存。否则不轮询。提供商行上的 `usage_limits_panel: false` 关闭面板：该行保持隐藏，`/usage` 会说明，并且该行不会向 hub 发送请求；配置重新加载会重新读取缓存，因此关闭的面板无需重启即可跟随。设计记录：`docs/plans/neuraldeep-usage.md`。

![原生控制台中完成的轮次，带有输出 token 和平均生成速度](../../assets/generation-speed/generation-speed-console-completed-dark-1280.png)

*原生控制台在编辑器上方保留已完成轮次的输出计数和平均生成速度*

渲染是 pi 的内联主屏幕模型：与前一帧的行差异、同步输出（`ESC[?2026h/l`）、逐行 SGR + OSC 8 重置、用于 IME 硬件光标放置的零宽 APC 光标标记、16 毫秒渲染节流，按键后立即渲染。

## 命令与按键

斜杠命令：设置命令 `/model`、`/reasoning`（`/effort`）、`/think`、`/nothink`、`/toolset`、`/agent`、`/plan`、`/ask` 和 `/permissions`，每个都带有 `--once` 或 `--count=N` 仅用于接下来的轮次（[Session settings](../features/session-settings.md)）；客户端 `/resume`、`/new`、`/login`、`/theme`、`/hotkeys`、`/queue`、`/usage`、`/tasks`、`/mcp`、`/docs`、`/quit`（`/exit`），以及一个裸 `/goal`，它会打开目标菜单；服务器驱动的 `/compact`、`/goal <objective>` 及其 `pause`、`resume` 和 `clear` 形式、`/export`、`/plugin`、`/worktree`，以及每个已加载的技能（来自 ACP 可用命令目录）。裸 `/model`、`/reasoning`、`/toolset` 或 `/permissions` 打开其选择器；带有值时命令由会话管理器应用，底部显示更改，后面跟有消息的命令应用于该消息启动的轮次。对话记录仅在智能体自行做出更改时获得一行，例如其 `switch_model` 调用后的 `Model: stub/coddy-mini for this session`。`/mode` 已移除：模式有自己的命令。`/toolset` 仅在配置有工具集预设时才在菜单中，其选择器列出预设和 `default`；底部在模型和级别之后显示会话运行所用的预设（`• tools coding`）。添加到配置的预设无需重启即可带来两者（[Toolsets](../getting-started/configuration.md#工具集)）。

**图片** - **ctrl+v**（在终端保留 ctrl+v 用于自身粘贴的地方为 **alt+v**）取剪贴板中的图片并将 `[image #N]` 标记放入编辑器；持有该标记的提示词将图片带给标记为 `multimodal` 的模型。有 X 显示时，图片首先在标记窗口中打开 - 铅笔、椭圆、矩形、箭头、文本、裁剪、旋转、撤销 - 完成后作为标记的图片附加。从 Web 界面或 Telegram 发送提示词时附带的图片也会显示为 `[image #N]`，而 Web 界面会将此处粘贴的图片显示为图片（[Pasting and marking up pictures](../features/picture-markup.md)）。

**模型记忆** — 控制台是独立的界面。会话范围的选取（`/model`、选择器，或提示词前的 `/model <id>`）记忆在 `$CODDY_HOME/console-state.json`（`{"last_model": "..."}`）中，每个**新**会话（`/new`、全新启动）从它开始，而不是 `agent.model`。在控制台的第一次使用时 - 还没有记住的选取 - 新会话从按字母顺序第一个配置的模型开始。重新打开和恢复的会话保留其捆绑包中保存的模型，`--model` 是每次调用的覆盖，不会重写记忆，`coddy -p` 总是遵循配置。

![/permissions bypass 和链式 /model --once 与 /reasoning --count=3 之后的控制台：底部显示 bypass，轮次覆盖行，对话记录中无行](../../assets/session-settings/session-settings-console-footer-dark.png)

*在 `/permissions bypass` 和 `/model stub/coddy-mini --once /reasoning high --count=3` 之后：底部显示 `bypass`，模型下方的轮次覆盖，对话记录中无内容。*

在斜杠建议上按 Enter 会一次性应用并提交。`/export [md|html|json|jsonl] [path]` writes the transcript into the workspace (`docs/features/session-export.md`)；在 `--remote` 下文件落在服务器上。`/usage` 强制重新读取活动提供商的账户用量（在面板关闭的行上，它改为指明 `usage_limits_panel` 开关）并将明细打印为暗色块，以品牌为标题，除非该行以其类型命名，则还加上行名（`Codex · codex-work`）：每个窗口有十格条、其百分比、计数器和重置时间、实时每分钟请求数、冷却、钱包及最近 30 天的支出、当节奏下限推迟读取时的 `refresh in Ns (pacing)` 行，以及快照的年龄。来源只报告它拥有的字段：Codex 和 Devin 打印窗口和计划，从不打印 rpm、冷却或钱包，而没有可报告内容的来源打印 `quota unavailable` rather than inventing numbers. Under `--remote` 时读取服务器自己的密钥，因此那里的 `key rejected` 行是信息性的（在服务器上登录）。

`/mcp` 打开服务器列表，包含范围、状态和工具计数，`off` 位于一个已关闭且状态为信任裁决的服务器旁边。Enter 打开服务器的控件以启用/禁用、项目信任和单个工具。只有在 `mcp.project_trust: ask` 下才为项目服务器提供授予和撤销，与 Web 界面的盾牌一样。在批准之前，控制台在选项上方打印整个声明 - 传输、命令行或 URL、其变量和标头的名称、工作区和文件 - 并且批准以指纹命名该声明，因此期间重写的检出会被拒绝而不是批准。禁用项目服务器会在操作员主目录下存储一个开关，而不是编辑检出，开关立即到达活动会话：该服务器连接或关闭，其他服务器继续运行。相同的菜单通过 MCP 管理路由在 `--remote` 上工作。

![控制台中 /mcp 服务器列表，显示一个禁用的全局服务器和一个不受信任的项目服务器](../../assets/mcp/mcp-console-dark-1280.png)

*`/mcp` 在打开服务器控件之前显示两个范围和信任状态。*

会话目标（[Session goal and supervisor](../features/session-supervisor.md)）在控制台中显示为对话记录行、底部注释和菜单。`/goal <objective>` 像任何提示词一样进入会话管理器：它设置目标并立即开始工作，`/goal pause`、`/goal resume` 和 `/goal clear` 也是如此。监督者启动的轮次是对话记录中的暗色行，而不是模型读取的指令 - `◎ Goal set: ...`、`◎ Goal continuation 1 of 10: <what the check found>` with the remaining work under it, `◎ Goal resumed: ...` - 实时以及会话恢复时，监督者自行做出的更改（`Goal blocked: <its question>`、`Goal complete: ...`）是一行，对于等待操作员的目标为警告色。当有目标时，底部在第一行注释之后命名它：`◎ goal active 1/10 (/goal)` in the accent colour, `blocked` in the warning colour, `complete`，绿色为 complete，`paused` 和 `limited` 为弱化色；在窄终端上，注释先丢弃命令，然后丢弃计数，然后路径被截断为零。裸 `/goal` 在编辑器位置打开目标菜单：目标、状态及原因、上次检查及其裁决、验证者是否确认了它以及它发现剩余的工作、监督者维护的清单，以及数字 - 已使用的 `supervisor.max_continuations` 续接次数、检查、目标轮次运行的时间和它们花费的 token 对照 `supervisor.token_budget`。当监督者处理目标时提供 **Pause**，暂停、阻塞或超出预算时提供 **Resume**（它发送 `/goal resume`，开始一个轮次），**Clear** 在移除目标前询问。菜单在打开期间跟随目标。在 `--remote` 下菜单通过 `GET`、`PATCH` 和 `DELETE /coddy/sessions/{id}/goal` 读取和更改目标，底部跟随服务器的 `session_goal` 事件，因此在浏览器中暂停的目标也会在此显示。

![控制台中带有被阻塞的会话目标：目标行、阻塞通知、目标菜单和底部注释](../../assets/session-supervisor/goal-console-blocked-dark-1280.png)

*监督者续接了一次然后因问题阻塞的目标：对话记录中的行、编辑器上方的目标菜单和底部的 `◎ goal blocked (/goal)`。*

`/tasks` 在编辑器位置打开会话的后台任务（[Background tasks](../features/background-tasks.md#在控制台中)）。智能体一直拥有 `background_list`、`background_output` 和 `background_stop`；这是操作员一侧的同一池。每个任务是一行：状态标记、说明其背后是什么的标签（命令为 `shell`，子智能体运行则为智能体的名称，委派给记忆子智能体为 `memory`）、标题 - 命令，或智能体被要求做的事 - 及其进展（`1m 08s · est. 5m 00s`，结束后为 `1m 30s`），带有智能体运行的模型和 token（`44s · qwen3.8-27b · 88.7k tokens`），最新的在前，就像 Web 界面的 Tasks 面板列出它们那样。结束时将唤醒智能体的运行中任务在其行中显示 `wakes the agent`，Web 界面的卡片在那里有它的铃。任务如何结束是它的标记（`✓`、`✗`、`■`）；打开的任务用文字说明。**enter** 打开光标下的任务：它如何以退出代码和持续时间结束（`failed · exit 2 · 1m 30s`）、其命令、智能体运行的子会话、它结束时的错误（除非那只是再次退出代码），以及其输出的最后几行，在任务运行时再次读取，结束时再读一次，读取它最后打印的内容。一次输出读取在途，如同列表读取，因此慢速服务器不会收集它们的队列。**s** 停止光标下的任务或打开的任务，包括进程组；**r** 再次读取所有内容；**escape** 先离开打开的任务，然后离开覆盖层。在 `--remote` 下，行、输出和停止都通过服务器的 REST 路由，因此覆盖层管理智能体运行的机器上的进程。当覆盖层打开、轮次运行或任务运行时每 2.5 秒刷新列表，否则每 15 秒刷新一次；轮次之间底部继续说明还有多少任务在运行。

智能体启动的后台任务在结束时会在本控制台中唤醒它，除非调用设置了 `notify_on_finish: false`（[Background tasks](../features/background-tasks.md#任务完成时唤醒智能体)）。

**F1** 在编辑器位置打开 Coddy 自己的文档，从二进制中读取（[Built-in documentation](../features/built-in-docs.md#控制台帮助)）：输入搜索章节，**enter** 在章节处打开一个，**tab** 在章节之间移动，**n** 和 **p** 翻页，**escape** 返回。`/docs [words or page]` 在终端保留 F1 用于自身的地方（GNOME Terminal 会这样）打开同一屏幕，在搜索上或直接在一个页面上：`/docs features/mentions#completion`。页面使用终端语言、`CODDY_LANG` 或区域设置（`LC_ALL`、`LC_MESSAGES`、`LANG`），控制台轮次附加和智能体读取的文档也是如此。

![F1 上的控制台帮助：搜索找到的章节](../../assets/cli-tui/19-docs-search.png)

*F1，然后 `telegram proxy`：找到的章节，选中的章节及其地址和片段*
被唤醒的轮次像输入的轮次一样运行 - 状态行、队列、在权限模态中询问的门控工具 - 并且在操作员消息本应出现的位置不显示任何东西：智能体的答案跟随上一轮，作为继续的工作，实时以及当 `/resume` 重放会话时。`/tasks` 是任务说明它的地方：运行时为 `wakes the agent`，其结束已启动轮次时为 `woke the agent`。在轮次、`!!` 命令或会话切换进行中到达的唤醒会等待其结束；对于控制台已用 `/new` 或 `/resume` 离开的会话，唤醒会等待操作员回到该会话，并且一行暗色文字会说明它在哪里等待一次。`coddy -p` 不运行唤醒器，因此那里的工具告诉模型没有东西会唤醒它。

在轮次运行时提交第一次会在状态行上询问默认队列模式：**1** 引导，**2** 轮次之后，**Esc** 将草稿放回。答案保存为 `agent.queue_mode`，消息被排队；用 **Tab** 发送的消息仍以另一模式进入，在答案之前写的第二条消息加入第一条。此后 **Enter** 使用该模式，**Tab** 使用另一模式：`steer` 到达下一个 ReAct 步骤，而 `after_turn` 在答案之后作为单独的提示词运行（[Message queue](../features/message-queue.md)）。等待中的消息显示在输入上方，为 `/queue` 命令编号：

```
queued messages (2) · /queue to manage
1. [steer] check the Windows path too
2. [after_turn] and skip the integration suite
```

`/queue` 列出它们，`/queue mode <n> steer|after_turn` 更改一个，`/queue drop <n>` 取回一个并将其文本放入输入，在自那以后输入的任何内容之前，`/queue clear` 清空队列。轮次读取的消息，以及轮次后消息在其提示词开始时，在对话记录中显示为你的消息，提及带来的文件折叠为提及。`escape` 取消轮次及其未读的引导消息；轮次后消息保持等待而不自动启动。在 `--remote` 下，这些控件也适用于另一个客户端在同一 `coddy serve` 上启动的轮次：**Enter** 为该轮次排队文本，**Escape** 请求取消。成功的取消响应确认请求；服务器可能仍在释放轮次。

远程控制台在打开和重新连接时读取会话的活动和队列，并通过 `GET /coddy/events` 接收更改。普通队列交付保持观察到的最高版本。服务器重启后，只有最新的新鲜快照，没有中间队列更新，可以降低该版本（包括降至零）。恢复还隔离延迟的重启前回复和通知，因此它们无法带回旧行。即使重启的服务器已在运行轮次，这也有效。失败的读取或交叉的快照会被重新读取，最多三次尝试和短暂延迟；更新的刷新或关闭会停止旧的恢复。队列读取有自己的超时预算，因此慢速活动读取无法用完它。服务器活动与控制台自己的提示词请求保持分离：观察另一个客户端的轮次不会启动或完成该请求。这些控件不订阅另一个客户端的实时对话记录，也不转移其权限/问题对话框。

如果轮次在 Enter 和队列准入之间结束，控制台会恢复提交的文本以及任何更新的草稿，而不是自动启动另一个提示词。一旦会话空闲，再次发送它。

![远程控制台，带有用于外部启动轮次的 Stop 和队列控件](../../assets/message-queue/stop-queue-remote-tui-dark-1280.png)

*控制外部启动轮次的远程控制台：Enter 排队后续消息，Escape 请求取消。*

此共享队列要求同一服务器进程的客户端。仅共享会话目录的裸本地控制台或本地 ACP 进程有自己的队列和应答通道；跨进程取消使用捆绑包的取消标记。一次性 `-p/--prompt` 调用者保留其现有的提示词和非交互式权限/问题行为。

`/reasoning` 无参数打开一个选择器，包含活动模型的可用级别；选择值会在会话上持久化该级别。`/reasoning <level>` 立即选择并持久化该级别。`shift+tab` 循环相同的级别并以相同方式持久化其选择。没有推理级别的模型报告推理不可用；不支持的直接值报告可选择的级别。

智能体自配置与通过 ACP 和 HTTP 时一样工作：每个轮次提供分阶段配置工具（`config_get`、`config_set`、`config_changes`、`config_commit`、`config_revert`、`config_rollback`；见 `docs/reference/config.md` 中的 **Agent self-configuration**），提交或回滚会热重新加载运行中的控制台，因此模型目录（`ctrl+l`、`ctrl+p`）、底部以及头部的 `[Context]`、`[Skills]`、`[Rules]` 和 `[MCP]` 章节无需重启即可跟随新文件。MCP 服务器不在该文件中：控制台监视 `~/.coddy/mcp.json`，其编辑在几秒内到达会话（[MCP](../features/mcp.md#在-coddy-之外所做的编辑)）。`-p/--prompt` 提供相同的工具；在 `--remote` 下服务器拥有重新加载。

| 按键 | 操作 |
|-----|--------|
| enter | 空闲时发送；轮次期间，将草稿排入 `agent.queue_mode`（首次会询问哪种） |
| tab | 在轮次期间有草稿时，以另一模式排队 |
| shift+enter / ctrl+j | 换行（反斜杠+enter 也可拆分） |
| escape | 中断运行中的轮次（`HandleSessionCancel`），或停止 `!!` 命令或 `/login` 登录 |
| ctrl+c | 清空编辑器；2 秒内两次退出 |
| ctrl+d | 编辑器为空时退出 |
| ctrl+l | 模型选择器 |
| F1 | 内置文档：搜索、阅读、翻页（也可用 `/docs`） |
| ctrl+p / ctrl+shift+p | 循环配置的模型 |
| shift+tab | 循环并持久化会话推理级别（具有 `reasoning_levels` 的模型） |
| ctrl+o | 展开头部提示 + 上次工具输出 + 上次 `!!` 块 |
| ctrl+t | 折叠/展开思考块 |
| up / down | 在首行/末行显示历史；否则移动光标 |

## 本地 shell (`!!`)

以 `!!` 开头的提交行不是提示词。Coddy 使用 `run_command` 所用的 shell 在工作区中运行其余部分，命令及其输出都不会显示给模型、持久化到会话捆绑包或计入上下文窗口。这是 pi 的 bash 模式前缀；pi 的另一半，`!`，将输出反馈给模型，仍然推迟。

- 前缀仅在提交行的开头计数；当缓冲区持有命令时编辑器规则变为绿色。要发送真正以 `!!` 开头的提示词，转义一次：`\!!careful` 以 `!!careful` 到达模型；
- 输出流式传输到其自己的对话记录块：折叠时的最后 10 行，`ctrl+o` 上的完整捕获，命令失败时的结束 `exit N`。输出解码并去除控制序列，与工具输出完全相同，并且只保留最后 256 KiB（块会说明它丢弃了多少），因此 `tail -f` 既不会增大控制台也不会减慢它；
- 没有权限模态，`ask` / `accept_edits` / `bypass` 不适用：输入命令的操作员是主体，而不是模型。计划模式限制智能体的工具，而不是你面前的终端；
- `escape` 杀死运行中的命令及其引导的进程组，退出控制台也是如此，因此没有任何东西继续打印到 coddy 刚刚归还的终端。命令有意分离的工作（`cmd &`、`nohup`）完全像在你自己的 shell 中一样比它更持久：一旦命令本身退出，控制台就没有什么可停止的了。当 kill 完全无法回收其目标时，第二次 `escape` 会归还控制台。没有超时，也不收养到后台任务池 - 该池是有意对智能体可见的；
- 一次一个：轮次运行时拒绝 `!!` 行，而命令运行时控制台拒绝提示词、另一个 `!!` 以及每个模态（`/new`、`/resume`、`/permissions`、`/theme`、`ctrl+l`），每个都有状态行说明 - 它们都不排队。模态会吞掉 `escape`，这是唯一停止命令的键；
- 命令从空设备读取，而不是从终端读取：交互式程序（`!!python`）看到 EOF，而不是从编辑器偷取按键；
- `--remote` 拒绝它。会话工作区位于服务器上，因此在本机运行命令会悄悄地触碰不同的树；
- `-p/--prompt` 不解释前缀：一次性提示词原样发送给模型。

该块仅属于运行中的控制台。用 `-c`、`--session-id` 或 `/resume` 重新打开会话不会重放它，因为 `!!` 命令的任何内容都不会写入磁盘。

模态在打开时替换编辑器：权限请求（选项列表来自智能体的 `permission.Options`）、问题工具（通过空格单选或多选，以及自由文本 **Custom answer...**，除非问题将 `custom` 设为 `false`；输入的文本加入旁边勾选的选项）、模型/推理/权限/主题/会话选择器（`→ ` 光标、输入过滤、`(i/n)` 滚动指示器）。

选择器在打开时以及每次调整大小时将其列表大小调整为终端大小，因此短屏幕 - 手机、分屏 - 仍显示选择是关于什么的。其边框、标题、信任提示或 `/goal` 在列表上方打印的主体、帮助行、滚动指示器和模态下方的底部保留其行；列表占据剩余部分，低至单行，并滚动其余部分。

后台子智能体在其 spawned 的轮次结束后继续工作，当它需要权限时通过相同的模态询问，标题为 `[subagent <name>] …`，在轮次之间也是如此。当屏幕上已有另一个权限或问题时到达的权限或问题会等待它而不是替换它，而询问者放弃的提示词（子智能体被停止或超时）会被撤下。子智能体的提示词在其 spawned 的轮次结束时仍然打开，会随该轮次关闭并立即作为子智能体自己的重新打开，因此它只被回答一次。一次性打印模式随其轮次退出，因此那里的后台子智能体被拒绝并附带原因。

问题模态将每一行花在选项标签上，并在列表下方打印高亮选项的描述，按整个宽度换行，因此句子长度的答案保持可读，而不是在列边界处被截断。当问题被回答时，其工具块显示带有选定答案的问题（`→ answer`、对于被驳回的为 `→ (no answer)`），而不是工具交给模型的 JSON。

## 标志

`--config --home --cwd --sessions-dir` 与其他子命令镜像。`-t/--test-config` 根据模式和加载器的规则检查控制台将加载的配置文件，打印每个问题及其行号和修复方法，并在错误时以状态 1 退出（见 [config.md](../getting-started/configuration.md#从命令行检查文件)）；它在没有 `cli` tag 的精简构建中也能工作。`--dry-run` 更进一步：在该检查之后，它探测文件指向的内容 - 它命名的目录、每个提供商的模型列表（地址和凭证在一个请求中）、MCP 命令、Telegram 令牌、给定时的 `--remote` 目标 - 并在探测失败时以状态 1 退出。单独使用时它仅打印问题和状态行；与 `--test-config` 一起使用时它打印配置检查报告和每个探测（见 [config.md](../getting-started/configuration.md#试运行探测文件指向的内容)）。`--session-id <id>` 重新打开（或创建）该会话并重放其对话记录。`-c/--continue` 重新打开此文件夹记录的最新会话，排除一次性打印模式的运行，只有 `-c -p` 会继续它们（不存在时报错；与 `--session-id` 和 `--resume` 互斥）。`--resume` 先打开会话选择器，在你选择之前不创建任何内容，打印运行也从中排除（与 `--session-id` 互斥；`--model`、`--mode` 和 `--permission-mode` 应用于选择器选择的任何会话）。`--model`、`--mode agent|plan|ask` 和 `--permission-mode ask|accept_edits|bypass` 在 UI 启动之前通过经验证的管理器配置选项 API 应用，在所有启动模式中（交互式、`--continue`、`--resume` 和 `--prompt`），并且它们像在会话中进行的切换一样写入会话，包括权限模式，因此会话在再次打开时保留它们，重启后也是如此；权限模式保留在该会话中，不会成为新会话启动的模式，因为从命令行运行是它自己的任务。`--theme dark|light|auto` (auto falls back COLORFGBG → dark). `--plain` 禁用终端查询、modifyOtherKeys、标题和 OSC 8，以实现确定性自动化。日志记录被强制从终端转移到 `<home>/logs/cli.log`（`--log-file`、`--log-level`）。`--log-level` 接受裸级别或逗号分隔的规范，单独提升一个子系统，例如 `--log-level "info,agent=debug"`（见 [config-reference.md](../reference/config.md#logger)）。

## 一次性打印模式 (`-p/--prompt`)

`coddy -p "..."`（或 `coddy cli --prompt "..."`）在没有终端的情况下运行单个智能体轮次：助手文本流式传输到 stdout，诊断信息到 stderr，进程在错误或取消轮次时以非零退出。不需要 tty，因此适合脚本和 cron。轮次持久化为标记为打印运行的会话（`session.json` 中的 `origin: print`），`-c -p "..."` 继续它 - token、文件和工具状态完全像交互式控制台一样延续。人们从中挑选对话的列表不包括打印运行：Web 界面的 History 直到选择其 **CLI runs** 过滤器、`/resume`、交互式 `coddy -c` 和编辑器的 `session/list`（[Sessions](../features/sessions.md#标签与归档)）。`coddy sessions list` 仍列出它们，因此超时的运行可通过其文件夹找到并用 `--session-id <id> -p "..."` 继续。运行重新打开的会话（`-c`、现有的 `--session-id`）保留其原有的来源。权限请求非交互式解决：`bypass` 允许，除了强制提示（钩子的 `ask`、决策检查未清除的命令），其他任何东西都拒绝调用并在 stderr 上注明。问题工具返回空答案。`--model`、`--mode`、`--permission-mode`、`--session-id` 和 `--continue` 都与 `--prompt` 组合；`--resume` 不可以（它需要交互式选择器）。

`--ephemeral` 在运行结束时删除运行的会话，无论它如何结束：包括失败的设置、失败的轮次和被取消的轮次。会话在运行期间存在，因此子智能体、后台命令和工具记录像在任何运行中一样工作，并且删除会停止运行留下的一切。崩溃可能会留下会话，像每个打印运行一样隐藏。删除失败是 stderr 上的警告，并将退出状态留给轮次。在 `--remote` 下服务器删除它。该标志在没有一次性提示词时被拒绝，并且与 `-c` 或 `--session-id` 一起时也被拒绝，其会话不是运行自己的可移除内容。

### 从文件或 stdin 获取提示词

提示词不必通过命令行传递，那里的操作系统限制单个参数（Linux 上为 128 KiB），而 `$(< file)` 会丢失文件的尾随换行符。coddy 自己读取它：

```bash
coddy -p - < brief.md          # the prompt on stdin
cat brief.md | coddy -p        # the same: a bare -p reads a pipe
coddy -i brief.md --mode ask   # the prompt from a named file
coddy -p -i brief.md           # the same, -p only asks for print mode
```

提示词恰好来自一个地方：`-p` 之后的文本、stdin（`-p -`、`-i -`，或当 stdin 不是终端时的裸 `-p`），或 `-i` 命名的文件。终端上的 `-p -` 在 stderr 上说明它从它读取提示词，并接受输入直到 ctrl+d。本身是 coddy 标志的提示词（`-c`、`--mode`）需要 `-p=-c` 拼写，因为裸 `-p` 后跟标志会从 stdin 读取提示词。它完全按读取的内容发送，包括行尾、尾随换行符和引号，并像输入的文本一样读取：其中的 `@path` 提及附加该文件，前导 `/skill` 或 `/model ...` 像在编辑器中一样工作。相对 `-i` 路径从命令运行的目录解析，就像任何 shell 参数一样（`--cwd` 移动会话，而不是路径）。符号链接会被跟随，像 `-i <(make report)` 这样的 FIFO 会被读取到结束，目录会被拒绝。

### 提示词下管道输入的数据

当提示词来自命令行或 `-i`，且 stdin 是管道或重定向的文件时，coddy 读取 stdin 到结束并在提示词之后附加它，就像 `codex exec` 追加管道 stdin 一样：

```bash
git diff origin/main...HEAD | coddy -p "Review this change" --mode ask
make test 2>&1 | coddy -p "Why does the build fail?"
coddy -p "Summarize the incidents" < app.log
```

模型在其位置读取为 `<coddy_attachment path="stdin" name="stdin" kind="stdin">`, and a transcript shows `[stdin]`。它是数据：其中的 `@` 不读取文件也不获取页面，其中的 `/command` 不运行任何东西，因此来自你无法控制的源的文本属于这里，而不是提示词。stderr 上的一行说明附加了多少，空白输入不附加任何内容。终端、`/dev/null` 和 stdin 上的套接字永远不会被附加；仅当提示词本身要来自 stdin 时才读取它们（`-p -`，并且对于终端以外的所有情况，裸 `-p`，因此生成 coddy 的程序可以写入提示词）。

这改变了现有 `coddy -p "..."` 在其 stdin 携带其他内容时发送的内容。在那里传递 `--no-stdin`（或重定向 `< /dev/null`）：

- 在 `while read -r f; do ...; done < list` 循环中，第一次运行会读取列表的其余部分；
- 在 `ssh host coddy -p ...` 后面，它保持 stdin 打开（`ssh -n` 也可以）；
- 在将作业脚本通过 stdin 提供给 shell 的 CI 运行器中，就像 GitLab 的运行器那样：没有该标志，coddy 会读取脚本的其余部分，发送给模型，其后的行将永远不会运行；
- 在从 stdin 接收 ref 行的 git 钩子中（`pre-push`、`pre-receive`），以及在使用 `docker run -i` 启动的容器中；
- 在 Windows PowerShell 中，它没有 `<` 重定向。

保持静默 3 秒的管道会在 stderr 上得到一行，说明运行正在等待输入，运行继续等待：像 `make` 这样的慢速生产者仍会附加其输出，而无人关闭的管道会显示为消息而不是无声挂起。

### 被拒绝的内容

输入在配置加载之前、会话存在之前以及任何请求之前被读取和检查，因此被拒绝的运行不发送任何内容并以 1 退出：

- 两个提示词（`-p "text" -i brief.md`、`-p - -i brief.md`）、重复的 `-p` 或 `-i`、空的 `-p ""`，或标志后留下的词语（`coddy -p fix the bug` 停在 "the"：给提示词加引号）；
- 缺失或不可读的文件，或目录；
- 不是 UTF-8（错误会指明字节偏移）或包含 NUL 字节的文本。UTF-8 字节顺序标记会被丢弃，带字节顺序标记的 UTF-16 会被解码；任何其他编码，包括没有标记的 UTF-16，需要先转换（`iconv -f cp1251 -t utf-8`）；
- 超过 8 MiB 的提示词输入，包括提示词文件和 stdin。不会为了适应而截断任何内容。限制是 coddy 读取的内容；那么大的提示词是否适合取决于模型的上下文窗口。

在 `--remote` 下客户端读取所有内容并将提示词作为请求的 `input` 发送，管道数据作为种类为 `stdin` 的字面附件发送。服务器会修剪 `input` 周围的空白，因此提示词自身的前导和尾随换行符不会在传输中保留，而附件逐字节到达。提示词文件中的 `@` 提及在服务器上、其工作区中解析。

## 远程模式 (`--remote`)

`--remote <target>` 将控制台（交互式和 `-p` 打印运行）指向远程 `coddy serve` 服务器，而不是在进程内运行智能体。目标是配置的远程名称（`httpserver.remotes`）、裸 `host:port`（scheme 默认为 http）或完整的 http(s) URL。承载令牌是 `--remote-token` 中的第一个、匹配 `httpserver.remotes` 条目的 `token`（配置的中继下的节点挂载采用中继的）和 `CODDY_REMOTE_TOKEN`。同一对标志在 `coddy acp` 上工作，因此 ACP 编辑器也可以驱动远程 coddy。

轮次在服务器上、其工作区中执行。对于此控制台启动的轮次，对话记录、工具框、思考、计划更新、token 和上下文统计通过 SSE 流式传回；权限和问题模态通过服务器的 REST 端点回答；`ctrl+o` 从服务器获取完整工具输出。模型选择器列出远程目录（`GET /v1/models`），并在服务器标记为 `default` 的行上显示没有自己模型的会话（服务器的 `agent.model`，或当 `agent.model` 命名了它未列出的模型时其第一个 `models` 行），`/resume`、`-c` 和 `--session-id` 对服务器的会话列表操作（本地文件夹过滤器不适用）。设置命令通过浏览器使用的相同 `PATCH /coddy/sessions/{id}` 更改服务器的会话，包括权限模式：`/permissions`、`--permission-mode` 和对话框的会话切换都到达服务器，底部跟随服务器的 `session_settings` 事件，以及当会话加载时服务器应答的快照。在服务器拥有会话之前所做的更改会被保留，并作为命令行在第一个提示词之前发送。`/reasoning` 和 `shift+tab` 在服务器会话上持久化选定的推理级别。会话仅在服务器上持久化；启动横幅显示 `remote: <url>`，退出提示打印包含 `--remote` 的重连命令。

子智能体在远程主机上运行：定义、信任收据和子会话都是服务器的。在那里批准项目定义（`coddy agents trust` on the server, or `POST /coddy/subagents/{name}/trust`）；本地 `coddy agents` 子命令不采用 `--remote`。子智能体的权限提示像父智能体的一样到达远程控制台，前缀为 `[subagent <name>]`，子智能体运行时状态行读取 `Running a subagent`；承担任务的子智能体由行上方的工具框命名，状态行从不重复它。轮次结束后才询问的后台子智能体也会到达控制台：服务器在其事件流上宣布提示词，控制台为其打开的会话打开模态，回答子会话；先在浏览器或聊天中回答，模态关闭。重新连接后，控制台协调完整的待处理请求快照：离线时回答的提示词关闭，而仍在等待的请求保持打开且没有重复模态。中断的快照不会驳回待处理请求。底部显示本地文件夹；信任收据以服务器端会话工作区为键（对于控制台创建的会话，为服务器的默认 cwd）。断开的连接会使服务器轮次及其子智能体保持运行；`/resume` 在结束时显示结果，对服务器已撤回提示词的回答会被忽略。服务器在控制台打开的会话中自行唤醒的轮次 - 完成的 `notify_on_finish` 任务 - 在会话的输入框中继上被跟随，由事件流上的 `background_wake` 宣布 - 或者，对于打开会话（`/resume`、`--session-id`）而该轮次已在运行的控制台，由会话活动的 `backgroundWake` 宣布，重新连接后由事件流的快照宣布，该快照在显示的最后一帧之后拾起同一轮次：答案和权限提示像它启动的轮次一样到达控制台，先在浏览器中回答的提示词再次关闭。控制台自己的唤醒器在 `--remote` 下保持关闭：任务在服务器的池中运行，服务器唤醒智能体。在轮次中退出控制台会短暂等待远程取消到达服务器。见 `docs/features/subagents.md`，Remote mode。

## 子智能体定义 (`coddy agents`)

三个子命令管理智能体可以委派给的子智能体定义（`docs/features/subagents.md`）。它们不需要构建 tag，像 `coddy mcp` 和 `coddy rules` 一样：

```
coddy agents list [--cwd DIR]
coddy agents trust <name> [--cwd DIR]
coddy agents untrust <name> [--cwd DIR]
```

`list` 打印工作区（在链接的 git worktree 中还有其批准所保留的主检出）和有效的 `subagents.project_trust`，然后以表格形式打印目录，包含 `NAME`、`SCOPE`（`builtin`、`user`、`project`）、`TRUST`（`trusted` 或 `needs_approval`）、`FLAGS`（`hidden`、`model=…`、`mode=…`）、`DESCRIPTION` 和 `PATH`（内置为 `(embedded)`）、`(total N)` 行，以及项目定义等待批准时的提示。`trust` 先打印有效声明（文件、模型、模式、权限模式、工具列表、摘要、收据路径），然后为磁盘上当前的文件记录收据，以规范工作区（链接的 git worktree 使用其主检出的）、名称和文件摘要为键，位于 `<home>/subagents-trust.json`；内置或用户范围名称无需批准，命令会说明。`untrust` 撤回收据。`--cwd` 默认为进程工作目录，像 `coddy mcp` 一样解析。在 `subagents.project_trust: deny` 下项目文件根本不列出，在 `allow` 下它们不需要收据。

在控制台中，`spawn_agent` 调用显示为工具框，其标题命名承担任务的子智能体（`spawn_agent explore · find every caller`），其下渲染子智能体收到的提示词，子智能体的报告在返回时作为主体添加；子智能体运行时状态行读取 `Running a subagent` 及其实耗计数器，框标题是名称所在之处。

![控制台对话记录中的委派运行](../../assets/cli-tui/13-subagent-delegation.png)

*一个已加载的技能和一次委派运行：框标题命名技能和子智能体及其任务和超时；暗色斜体块是子智能体收到的提示词，子智能体的报告紧随其后*

子智能体的权限请求，在其 spawned 的轮次仍然存活时，在父聊天中打开通常的模态，标题前缀为 `[subagent <name>]`。子会话是只读对话记录：`-c` 从不选择它们，发送给它们的提示词被拒绝并附带命名父会话的消息。

## 钩子定义文件 (`coddy hooks`)

三个子命令管理会话将加载的钩子定义文件（`docs/features/hooks.md`）。它们不需要构建 tag，像 `coddy agents` 一样：

```
coddy hooks list [--cwd DIR]
coddy hooks trust <file> [--cwd DIR]
coddy hooks untrust <file> [--cwd DIR]
```

`list` 打印工作区（在链接的 git worktree 中还有其批准所保留的主检出）和有效的 `hooks.project_trust`，然后是一个表格，包含 `FILE`（收据使用的名称：项目文件为工作区相对路径，操作员自己的为绝对路径）、`SCOPE`（`user`、`project`）、`TRUST`（`trusted` 或 `needs_approval`）和 `HOOKS`（诸如 `PreToolUse(run_command); PostToolUse(*)` 的摘要，或无法解析的文件为 `invalid: <error>`）、`(total N)` 行，以及项目文件等待批准时的提示。`trust` 打印它即将批准的钩子（事件、匹配器、命令）并为磁盘上当前的文件记录收据，以规范工作区（链接的 git worktree 使用其主检出的）、文件及其摘要为键，位于 `<home>/hooks-trust.json`；用户范围文件无需批准，命令会说明，无效文件会被拒绝。`untrust` 撤回收据。`--cwd` 默认为进程工作目录，像 `coddy mcp` 一样解析。在 `hooks.project_trust: deny` 下项目文件根本不列出，在 `allow` 下它们不需要收据。远程控制台（`--remote`）改为在服务器上通过 `POST /coddy/hooks/trust` 批准，因为收据必须位于运行钩子的机器上。

## 安全

所有来自渲染器外部的文本 - 模型输出、工具预览、标题、文件名、技能和 MCP 名称 - 在显示前都会被清理：ESC、C0 控制（换行和制表符除外）、DEL 和 C1 控制都会去除，因此模型或工具提供的转义序列永远无法到达终端。渲染行中唯一的控制序列是渲染器生成的样式。

权限模式 `ask` 将每个请求渲染为模态；`bypass` 完全像 ACP `serverRef` 路径一样短路。`!!` 前缀按设计没有权限门（见 **Local shell**）：它只能从提交的编辑器缓冲区到达，永远不能从模型输出到达，因此模型写入的任何内容都无法通过它启动命令。项目本地 MCP 服务器仍通过工作区信任门；等待批准的服务器保持断开，可通过 `coddy mcp list` 看到（用 `coddy mcp trust <name>` 批准）。

## 截图

![全新控制台的启动行、头部、编辑器和底部](../../assets/screenshot-console-start.png)

*带头部、`[Context]` 和 `[Skills]` 章节的启动行，编辑器，以及显示模型提供商账户用量的底部*

![ctrl+l 模型选择器](../../assets/screenshot-console-models.png)

*ctrl+l 模型选择器：每个配置的模型，当前模型用箭头标记*

![第一帧，底部显示 MCP 计数，服务器仍在连接](../../assets/cli-tui/21-mcp-connecting.png)

*第一帧：控制台已绘制并在其 MCP 服务器连接时接受按键，底部统计它们（`MCP 1/2`）*

![在服务器响应之前发送的提示词：状态行读取 Connecting MCP servers](../../assets/cli-tui/22-mcp-connecting-turn.png)

*在服务器响应之前发送的提示词在状态行上等待其工具列表*

![九行终端中的 ctrl+l 模型选择器](../../assets/cli-tui/23-short-terminal-selector.png)

*九行终端：列表让出行，以便边框、标题、高亮模型、`(6/8)` 滚动指示器、帮助行和底部都留在屏幕上*

![带账户窗口的用量底部](../../assets/cli-tui/09-usage-footer.png)

*带账户窗口的用量底部*

![窗口填满时的底部警告](../../assets/cli-tui/10-usage-warning.png)

*窗口填满时的底部警告*

![达到限制：轮次等待重置](../../assets/cli-tui/11-usage-blocked.png)

*达到限制：轮次等待重置*

![重置后轮次恢复](../../assets/cli-tui/12-usage-resuming.png)

*重置后轮次恢复*

![运行中轮次的状态行：2s、64 tokens、1 running task、Responding](../../assets/cli-tui/14-turn-progress.png)

*运行中轮次的状态行以其时钟、在其中生成的 token 和运行中的后台任务开头；底部也命名任务*

![列出运行中命令的 /tasks 覆盖层](../../assets/cli-tui/15-tasks-overlay.png)

*`/tasks`：会话的后台任务在编辑器位置*

![在 /tasks 覆盖层中打开的任务：其命令和输出的最后几行](../../assets/cli-tui/16-tasks-output.png)

*用 enter 打开的任务：命令、其输出的最后几行，以及停止它的 `s`*

![后台唤醒后的 /tasks 覆盖层：运行中的构建唤醒智能体，失败的测试运行唤醒了它](../../assets/cli-tui/17-tasks-wake.png)

*唤醒后：智能体的答案跟随其上一轮，中间没有任何东西，`/tasks` 说明哪个任务唤醒了它以及哪个将会*

![“@ment”的提及列表：一个文件夹和四个从整个树中排名的文件，每个都有其种类](../../assets/cli-tui/18-mention-list.png)

*`@ment`：整个工作区针对片段排名，每行的种类，以及列表容纳的五十个之外匹配了多少*

存在两套截图，它们回答不同的问题。

`docs/assets/screenshot-console-*.png` 是真实 **Konsole** 窗口中运行中控制台的照片（宽 1920 像素，裁剪到使用的行）：带 `[Context]` / `[Skills]`（`screenshot-console-start.png`）的启动行和头部、`ctrl+l` 模型选择器（`screenshot-console-models.png`），以及带工具框、思考块和底部计数器的已完成轮次（`screenshot-console-chat.png`），加上全新安装的首次启动，其头部指向 `/login`（`screenshot-console-login-hint.png`，在空的 `CODDY_HOME` 上）。在 README 和网站上使用这些：它们展示用户在终端模拟器中实际看到的内容。它们在一个干净的 `CODDY_HOME` 上拍摄，带有两个自己的技能和一个带 `AGENTS.md` 的小项目，因此头部列出新用户看到的内容。

`docs/assets/cli-tui/` 保存由每个功能一个 `examples/cli/capture_*.py` 脚本从 pyte 缓冲区渲染的 PNG，每个脚本针对脚本化的替身模型驱动共享的 e2e 驱动程序。这些是颜色和单元格布局的参考，不是营销图像；重新生成 chrome 已更改的那些。`examples/cli/capture.py` 渲染基本状态（启动、斜杠菜单、模型选择器、一个轮次）以供你在自己的机器上比较；它们都没有被提交。四个用量状态（`09-usage-footer`、`10-usage-warning`、`12-usage-resuming`、`11-usage-blocked`）来自 `examples/cli/capture_usage.py`，它在 `CODDY_NEURALDEEP_BASE_URL` 后面放置一个假的 hub `GET /limits`，加上一个聊天完成，回答一个为等待轮次命名遥远重置的 `429`，因此不需要真实密钥；只保留它们的 PNG。`13-subagent-delegation` 以同样方式来自 `examples/cli/capture_subagent.py`：本地 OpenAI 兼容端点脚本化一个 `load_skill` 调用、一个 `spawn_agent` 调用、子智能体的报告和父智能体的答案，因此截图既不需要提供商也不需要密钥。`18-mention-list` 来自 `examples/cli/capture_mentions.py`，它在临时文件夹中、带临时主目录、以空和 git 下布局此仓库的文件，并针对一个从不被询问任何东西的提供商输入 `@ment`。`session-settings-console-footer-dark` 来自 `examples/cli/capture_settings.py`：`/permissions bypass`，然后链式 `/model --once /reasoning --count=3`，临时主目录也代替 `HOME`，因此头部仅列出捆绑的技能。`login-provider-picker-dark-1280` 和 `login-signed-in-dark-1280` 都来自 `examples/cli/capture_login.py`：全新控制台在空 `CODDY_HOME` 上自行打开的选择器，然后从中选择 NeuralDeep，hub 和 API 基地址指向一个本地替身，它回答设备登录、账户、层级的模型和用量，因此既不需要 hub 也不需要密钥。`23-short-terminal-selector` 来自 `examples/cli/capture_short_terminal.py`，它在九行 pty 中针对仅列出其模型的提供商打开 `ctrl+l` 选择器；`CODDY_BIN` 和 `CLI_E2E_ROWS` 将同一脚本指向另一个二进制或另一个高度，这就是修复前状态在修复后状态旁被捕获的方式。


## 测试

- 单元 + BDD：`go test -tags=cli ./...`；happy-path 规范是 `features/cli_tui.feature`，由 `external/cli/bdd_cli_tui_test.go` 运行（存根运行器、假终端 - 无 LLM、无 pty）。提供商用量场景在 `CODDY_NEURALDEEP_BASE_URL` 后面放置一个假 hub `GET /limits`（进程范围，因此包保持顺序），并用 `SetProviderUsageClock` 驱动管理器的节奏时钟；`external/cli/usage_test.go` 固定底部措辞、丢弃顺序、阻塞者副本、hub 字符串的清理和重置计时器。
- 一次性输入：`features/cli_prompt_input.feature`（`external/cli/bdd_prompt_input_test.go`）在测试进程中使用管道或文件作为其 stdin 运行 `cli.Run`，针对记录每个请求的脚本化模型，并逐字节检查提示词、stdin 附件和对话记录标签；`external/cli/prompt_input_test.go` 涵盖标志规则、stdin 种类、解码和限制，并断言被拒绝的输入永远不会到达模型。`examples/cli/cli_e2e_print_input.py` 使用构建的二进制和真实 shell 做同样的事：管道、文件、200 KiB 提示词、`while read` 循环，以及 pty 上的裸 `-p`。
- 真实 pty，无模型：`examples/cli/cli_e2e_startup.py` 在 pty 中打开构建的二进制（pexpect + pyte），等待第一帧，在编辑器中输入，用 ctrl+c 清除并用第二次退出，然后检查恢复提示和退出状态；第二个案例配置一个从不响应的 stdio MCP 服务器（`sleep 600`），并检查第一帧仍在几秒内出现，底部带有 `MCP 0/1`，以及控制台仍通过双击 ctrl+c 离开；第三个添加 300 个已安装技能和 `marketplaces.json` 中一个接受连接但从不响应的源（issue #319），并检查第一帧在两秒内出现、输入的键被回显且该源从未被联系。这两个是 CI 强制执行的启动界限。启动本身的性能是测量而非门控：`make bench-cli-startup` 在同一 pty 中跨技能集计时第一帧和输入探测的回显，`make bench-cli-startup-real` 在操作员 `~/.coddy` 的私有副本上做同样的事，包括 MCP 服务器，报告每个服务器何时稳定以及哪个失败（`docs/plans/console-mcp-startup.md` 保留更改所引用的数字）。CI 在 Linux 测试矩阵的 `cli` 作业中以及在 `macos-latest` 上运行它（作业 `test-macos`，它还在 macOS 上运行平台包和控制台套件），因为 Go 套件从不打开 pty，而控制台的终端路径正是主机之间不同的地方。
- 真实 pty 上各种 MCP 服务器：`examples/cli/cli_e2e_mcp_servers.py` 配置一个原生程序（由脚本从 Go 编译并按路径运行）、一个通过真实 `npx -y` 从本地文件夹启动的 npm 包（不询问注册表）、一个通过 streamable HTTP 的远程服务器和一个通过 SSE 的远程服务器，并提供一个脚本化模型，依次调用每个的工具。它检查第一帧、答案中的四个 token、模型从每个服务器收到的内容，以及 `/mcp` 中列为已连接的四个服务器。它需要 `PATH` 上的 `go` 和 `npx`；CI 在 Linux 测试矩阵的 `cli` 作业中运行它。`features/cli_tui.feature` 在 Go 套件中为一个程序、streamable HTTP 和 SSE 持有相同的故事，`features/mcp_tool_calls.feature` 通过 HTTP 界面的真实 ReAct 轮次运行每种类型，包括 `npx`。
- 真实 pty 上的消息队列，无模型：`examples/cli/cli_e2e_queue.py` 提供一个脚本化模型，其第一个答案等待脚本，并检查首次使用问题和保存的 `agent.queue_mode`、Tab 排队另一模式、`/queue drop` 将消息放回输入，以及一旦答案被释放，引导消息在轮次中读取，延迟消息在其后回答，显示在两个答案之间。CI 在 Linux 测试矩阵的 `cli` 作业中运行它。
- 实时 e2e：`./examples/test_cli.sh` 默认在 pty（pexpect + pyte，仅 Linux）中针对 `neuraldeep/qwen3.8-27b` 驱动真实二进制 - 见 `examples/README.md`。

交互式 `--remote` 控制台的停止/队列回归清单：

- 打开另一个客户端启动的轮次的会话：Enter 排队文本而不是尝试第二个提示词；Escape 发送取消。
- 在后续消息等待时重新连接：队列读取恢复它们而没有另一次变更，较旧的 HTTP/SSE 版本不会替换较新的版本。
- 观察外部轮次开始或结束：其活动不会完成控制台自己的提示词请求或附加外部对话记录读取器。
- 确认一次性 `-p` 调用和权限/问题模态保留其现有的所有权和处理。

## 已知的 v1 与 pi 的差异

编辑器中的撤销/yank-pop 和跳转模式、kitty 键盘协议协商、内联图像、pi 的 `!` bash 模式（将命令输出反馈给模型的那一半：`!!` 按 **Local shell** 中所述发布，而 `!` 需要模型可见的对话记录记录并保持推迟）、pi 的 `/settings` 界面，以及会话选择器的范围/排序/重命名控件。主题切换会重建 chrome；已渲染的对话记录行保持其颜色直到下一个会话。

## 署名

TUI 渲染模型和视觉设计移植自 [`pi-mono`](https://github.com/badlogic/pi-mono) `packages/tui` 和 `packages/coding-agent` 交互模式（MIT License，Copyright (c) 2025 Mario Zechner），提交 `b1efcf7d7`。`external/cli/tui` 中的 Go 实现是对文档行为的独立重写。

<!-- docsgen:source sha256=9a573923bc1499b2 -->

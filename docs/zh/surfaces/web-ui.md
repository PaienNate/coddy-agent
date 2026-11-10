# Coddy 嵌入式 UI 规范

本页记录最初的 UI 需求和预期最终状态。它是功能规范与设计契约，每个界面的截图都紧邻规定该界面的章节。视觉 token 与组件约定见 [DESIGN.md](../../../DESIGN.md)。

https://github.com/user-attachments/assets/604d01bb-b730-4ca5-9955-a7a593891054

*一段两分钟的 1.3.0 Web 界面录屏：模型选择器、模型读取的一张图片、一个把工作移入 git worktree 并由主管确认的 `/goal`、带每个文件补丁的编辑窗口、文件窗口，以及从斜杠菜单启用的 `/commit` 技能。该文件位于仓库中的 [web-ui.mp4](../../assets/video/web-ui.mp4)。*

## 约束

- UI 以静态资源形式随 `coddy` 二进制嵌入发布（构建标记 `http`）。
- 运行时对 UI 不进行认证，也不检查 API 密钥。
- UI 必须通过 `coddy serve` 的同源工作。
- UI 本地化由注册表驱动；默认是英语，当前发布 **俄语（RU）** 和 **简体中文（ZH）**，可从 **设置 → 外观 → 语言** 选择（见下文）。
- Favicon 与 [coddy.dev](https://coddy.dev/) 一致（**`/coddy-favicon.svg`**，与 **`docs/assets/coddy-logo-mark-flat.svg`** 相同的标记，另有随 SPA 嵌入的 PNG/ICO 回退）。

## 外观（主题 + 语言）

![设置，外观选项卡：主题色块和语言选择器](../../assets/screenshot-fullhd-settings-appearance.png)

*设置，外观选项卡：主题色块和语言选择器*

![简化模式：设置覆盖在聊天之上，面板为纯色，没有阴影或模糊](../../assets/web-ui/simple-mode-setting-dark-1280.png)

*简化模式：设置覆盖在聊天之上，面板为纯色，没有阴影或模糊*

![应用于对话的俄语词典：聊天头部、工具行和输入框](../../assets/localization-ru-wide.png)

*应用于对话的俄语词典：聊天头部、工具行和输入框*

- **默认：** 首次访问使用深色主题；语言从 **`navigator.language`** 解析（如果是俄语则为 RU，如果是中文则为 ZH，否则为 EN）。
- **主题 cookie：** **`coddy_ui_theme`**，包含八个主题 id（**`dark`**、**`light`**、**`midnight`**、**`solarized-dark`**、**`monokai`**、**`nord`**、**`rose-pine`**、**`neuraldeep`**；路径 **`/`**，**`SameSite=Lax`**，有效期 1 年 `Max-Age`）。
- **主题选择器：** **设置**（**`#/settings`**）→ **外观** → 主题色块网格（**`data-testid="theme-swatch-<id>"`** 位于 **`appearance-theme-picker`** 内）。选择立即生效，且仅在客户端生效（不保存配置）。
- **部署默认值：** **`CODDY_UI_THEME`** 指定未携带 cookie 的浏览器打开时使用的主题，而 **`CODDY_UI_WALLPAPER`** 指定在用户选择背景之前显示的服务器壁纸库文件；两者都通过 **`GET /coddy/info`** 传递，并且仅在浏览器尚未做出自己的选择时生效。
- **简化模式：** 主题网格下方的开关（**`data-testid="appearance-simple-switch"`**），用于较慢的机器：所有主题中的面板均为纯色，没有阴影、模糊、装饰性渐变或动画。所选壁纸仍会连同其暗化效果可见。旁边的 (i) 会说明它关闭了什么。该选择是 cookie **`coddy_ui_simple`**，和主题一样仅在客户端生效，并在首次绘制前应用。开始屏幕上变化的词在普通和简化模式下都跟随主题强调色。
- **壁纸：** 位于简化模式开关下方、语言选择器上方（**`data-testid="appearance-wallpaper"`**）：一个紧凑的服务器图库，带有上传和删除控件；此浏览器会记住所选背景和暗化程度（[壁纸](#壁纸)）。
- **语言选择器：** 一个原生 select，位于**壁纸区块下方**（**`data-testid="appearance-language-select"`**），包含 **自动**（从 **`navigator.language`** 解析，不存储 cookie），其后是 **`locales.ts`** 中注册的每个区域设置。当前注册表渲染 **English**、**Русский** 和 **中文**；更改 select 会立即应用区域设置。
- **通知开关：** 位于语言选择器下方（**`data-testid="appearance-notifications-switch"`**）：当 Coddy 在后台时发送系统通知，保存在 cookie **`coddy_ui_notify`** 中（[通知](#通知)）。
- **语言 cookie：** **`coddy_ui_lang`** 存储已注册的区域设置 id（当前为 **`en`**、**`ru`** 或 **`zh`**），标志与主题 cookie 相同。选择 **自动** 会清除它。加载时的解析顺序：URL 中的 **`?lang=<registered-id>`**（也会持久化到 cookie）> cookie > **`navigator.language`**。切换会设置 **`document.documentElement.lang`** 并重新渲染，无需重新加载。文档会跟随它：阅读器会以该语言请求其页面（**`lang=`** 位于 **`/coddy/docs*`** 上），并且每个轮次都会指明该语言（**`metadata.lang`**），因此智能体也会以该语言读取并附加 Coddy 的文档（[内置文档](../features/built-in-docs.md#语言)）。不会向配置写入任何内容。
- **i18n 引擎：** **`external/ui/src/ui/i18n/`**（**`translate`/`t`**，区域设置存储，**`I18nProvider`** + **`useT()`**）。**`locales.ts`** 是受支持 id、选择器标签和词典的唯一注册表；选择器生成、区域设置校验、启动引导和一致性测试都由它派生。**`main.tsx`** 在 **`I18nProvider`** 中包装应用以及共享确认提供程序。**`useT()` 在提供程序之外回退到 `translate`**，因此组件在测试中无需包装即可渲染；默认英语值与此前硬编码字面量完全一致。
- **区域设置维护：** 添加区域设置需要其词典外加一个 **`locales.ts`** 条目。每个已注册词典都必须在同一个补丁中添加或更改相同的键和插值标记；**`messagesParity.test.ts`** 会同时强制这两点。
- **复数文案：** 带计数的字符串使用 **`translatePlural`** / **`tp(key, count)`**，每个 CLDR 类别对应一个词典条目（**`key.one`**、**`key.few`**、**`key.many`**、**`key.other`**），因此俄语会按数字对名词进行变格，而不是回退到单一形式；中文只使用 **`key.other`**。每个区域设置必须恰好提供其自身 **`Intl.PluralRules`** 产生的类别；一致性测试会按区域设置派生该集合。
- **覆盖范围：** 外观 + 设置界面已翻译（设置外壳、各章节、MCP、技能、CodexAuth、ModelField/Picker、Combobox），由 schema 驱动的设置字段标签和描述也已翻译（见下文），对话界面也已翻译：导航栏、主视觉标题、输入框（模式、模型选择器、附件、斜杠/@ 菜单、环境和文件夹模态框）、消息渲染（思考、工具调用、压缩、复制控件）、权限和问题提示、计划文档卡片、历史侧边栏、调度器抽屉和任务编辑器、后台任务面板、环境健康横幅，以及带拓扑图的集群屏幕。草稿、聊天和调度器任务的共享破坏性确认也已翻译。
- **Schema 字段本地化：** 由服务器 JSON Schema 渲染的设置章节（提供商、模型、智能体、主管、工具、子智能体、钩子、记忆、压缩、调度器、日志记录器、网关、会话，以及提示词选项卡的部分内容）通过 **`settings/schemaI18n.ts`** 在客户端本地化其字段标签和描述：词典键由章节 id 和点分字段路径确定性地派生（**`settings.schema.<section>.<path>.label` / `.desc`**，提示词选项卡的部分内容，从 `prompts` 和 `instructions` 键折叠进来，作为 **`settings.schema.system.<child>.<path>`**）。词典未定义的键会回退到 schema 自身的英语文本，因此未映射或新添加的服务器字段绝不会泄漏原始键。数组项行会为其嵌套字段继承域，但行标签保留自己的回退，因此包围的 fieldset 图例和描述不会逐行重复。
- **设置一次显示一个选项卡**：外观和技能与配置章节是同一列表中的选项卡。
- **持久化：** 切换主题会写入 cookie 并设置 **`document.documentElement.dataset.theme`**；重新加载必须保留所选主题。
- **CSS 契约：** **`--text`** 和 **`--bg`** 在 **`[data-theme="light"]`** 上分别为 **`#18181b`** 和 **`#f8f8fa`**；玻璃面板使用 **`rgba(255, 255, 255, 0.9)`**（而非深色着色）。深色默认值保留在 **`:root`** / **`[data-theme="dark"]`** 上。

## 首次运行引导

![较早的提供商选择器，在 NeuralDeep 上打开](../../assets/web-ui/onboarding-provider-dark-1280.png)

*全新 Coddy 主页上的提供商选择器，在 NeuralDeep 上打开（深色，1280 px）。*

![手机上的同一选择器，每行一张卡片](../../assets/web-ui/onboarding-provider-dark-390.png)

*手机层级上的较早选择器（深色，390 px）。*

- **何时打开。** 每次页面加载都会读取一次 `GET /coddy/onboarding/status`。当 config.yaml 不存在、没有配置提供商或模型、不存在有效的默认模型，或该模型的提供商没有可用的凭据来源时，选择器会打开。默认值是 `agent.model`，或按字母顺序第一个 `models[]` 行（当 `agent.model` 为空时）。智能体不使用的无密钥提供商只会出现在信息性 `missing_api_keys` 列表中，不会重新打开选择器。带有自身 `api_base` 的本地兼容 OpenAI 端点可以在没有密钥的情况下就绪。状态检查从不运行 `api_key_command`，也不会调用提供商。
- **提供商预设。** NeuralDeep、OpenCode、OpenAI、Anthropic、Codex、Devin、Kimi、兼容 OpenAI 和 ACP 智能体，按此顺序排列，对话框打开时所在的卡片 NeuralDeep 放在最前。整个目录在一个可滚动的网格中显示在屏幕上：桌面端显示两行，手机端显示四行，下面一行从边缘露出，上下边缘都根据其外侧的内容向面板内淡出，顶部边缘在最初几行滑入其下时显现，底部边缘在接近末尾时变淡，因此很明显还有更多内容，而在读到表单第一个字之前，目录仍不会填满对话框。滚轮或滚动条可到达其余部分。任何宽度下每行两张卡片，包括手机，每行固定高度，以便窗口显示完整卡片。一张卡片涵盖所有兼容 OpenAI 的端点，包括本地 Ollama 或 LM Studio，ACP 卡片接收此机器上运行的智能体的命令，写入 `providers[].acp.command` 且完全不写入凭据。控制台的选择器以相同顺序提供相同目录，并打开同一张卡片（[控制台](console.md#提供商选择器)）；两者都从同一份列表读取，即 `internal/providerpreset` 及其 TypeScript 对应文件 `onboarding/providerConfig.ts`，并由 Go 测试保证二者一致。每个预设都携带提供商名称、类型、推荐模型，以及在相关情况下携带端点和环境变量键占位符。OpenCode 提供 Go 和 Console 网关以及现有的设备登录；Devin 提供浏览器 PKCE 登录，以及供无法访问 Coddy 机器的浏览器使用的回调地址字段。两者也都接受显式密钥，显式密钥优先于已存储的登录。兼容 OpenAI 的预设没有自己的端点，因此在其 API 基础 URL 填写之前，**保存并继续** 保持禁用；Ollama 在其本地端点上不需要密钥，也不会为它写入占位密钥。
- **Kimi。** 服务选择器提供 Kimi Code 订阅和 Kimi Platform API 密钥，分别用于 `kimi.ai`（全球，默认）或 `kimi.com`（中国）。订阅提供设备登录；Platform 接受密钥。表单会显示订阅凭据来自 Coddy 还是 Kimi CLI。后者在 `KIMI_CODE_HOME` 下为只读（默认 `~/.kimi-code`）。当前表单代理会应用于登录。返回的 `kimi_identity` 属于提供商行，必须与其一起保存。
- **高级设置。** 首次运行不必回答的内容折叠在 **高级设置** 后面，这是一个与设置中的提供商表单所用同类 fieldset，其主体保持此表单的左边缘。它包含代理、登录提供商的 **API 密钥**，以及预设打开自有端点时的 **API 基础 URL**：NeuralDeep 的两个枢纽和 OpenCode 的两个网关是对可用默认值的细化，而不是首先要回答的问题。兼容 OpenAI 的预设将其端点保留在表单中，因为它没有端点，而 **保存** 会等待一个端点。首次运行从不需要打开它；打开后，它包含一个普通的 **代理 URL** 字段，在那里输入的内容是该行发出的每个请求（包括登录）的路由。留空则遵循系统代理。**忽略系统代理** 开关不在这里：为单行关闭系统代理属于调优，而调优位于 **设置 → LLM 提供商**。
- **模型和连接检查。** 对话框不要求选择模型。它按表单中该行的当前状态读取 `POST /coddy/providers/models`，未保存的提供商和仅存在于服务器上的登录也以同样方式列出其模型；该行会在提供商提供预设自身推荐模型时从该模型开始，在提供商未提供时从它确实列出的第一个模型开始，在提供商没有列出任何模型时从推荐模型开始。托管登录一连接就会获取，而 **保存并继续**（对话框唯一的操作）在尚未获取时自行获取一次，这样操作者引入的端点就不会被写入一个它从未提供过的模型。该获取就是过去需要手动点击 **测试连接** 按钮来请求的连接检查。列出的模型会将其上下文窗口、目录 `max_tokens` 和图像支持带入模型条目。之后可在输入框或 **设置 → LLM 提供商** 中更改模型；无法访问的提供商在此处不算错误，该行仍会被写入。
- **一次仅一种进入方式。** 对话框只显示一个主要操作，具体是哪一个取决于提供商。登录账户时，会在操作行中原先放置 **保存并继续** 的位置放置 **使用 NeuralDeep 登录**，且没有其他需要回答的内容；将密钥输入 **高级设置** 会把操作交还给 **保存并继续**，因为密钥意味着决定不使用账户流程；连接后仅保留账户行，不包含属于 **设置 → LLM 提供商** 的提示说明或 **退出登录**；登录运行期间，该位置以一行文字显示等待，而不是显示不可用按钮；账户连接后，按钮变为 **保存并继续**，设备代码及其链接消失，剩下的是标明账户的那一行。接受密钥的提供商会从一开始显示 **保存并继续**。两个同等权重的按钮，一个在字段旁，一个在底部，既没有表明两者都需要，也没有表明顺序：其 **API 密钥** 移到 **高级设置** 中代理旁，方便已有密钥的人使用，也不妨碍没有密钥的人。无登录的提供商（OpenAI、Anthropic、任何兼容 OpenAI 的端点）将密钥保留为表单本身的字段；Codex 完全没有密钥，ACP 智能体则两者都不携带。只要两者同时存在，显式密钥仍优先于已存储的登录。
- **离开。** 对话框没有跳过按钮，也没有测试连接按钮；**保存并继续** 是唯一操作。点击其外部会关闭它，导览和 **设置 → 外观 → 重新开始引导** 的行为与之前相同。
- **保存。** 校验和 PUT 走常规 `/coddy/config` 流程。所选提供商和模型会合并到对话框打开时读取的文档中：其他提供商、模型和 `agent.max_turns` 会被保留。托管的 NeuralDeep、OpenCode 或 Devin 登录会在保存的行中省略 `api_key` 和 `api_key_command`，以免旧密钥遮蔽它；Codex 从不写入这两个字段中的任何一个。随后应用会刷新其模型元数据。

![引导式导览的第一步](../../assets/web-ui/onboarding-tour-dark-1280.png)

*导览聚焦设置（深色，1280 px）。*

- **引导式导览。** 保存提供商，或点击对话框外部将其关闭，会启动一次教练标记导览，每个浏览器一次，在 localStorage 中记录为 `coddy.tourSeen`。步骤遵循首次对话发生的顺序：新建聊天按钮、历史、设置，然后是工作区标签、输入框、模型选择器和发送。调度器不属于首次对话，因此没有步骤。缺失的锚点会被跳过，聚光灯跟随调整大小和滚动，并支持后退、下一步、跳过和 Escape。
- **重新开始。** **设置 → 外观 → 重新开始引导** 会关闭抽屉、清除导览标志并再次打开提供商选择器。在集群中继上不提供此功能，因为其设置没有智能体引导。

![设置 → 外观中的重新开始引导](../../assets/web-ui/onboarding-restart-dark-1280.png)

*通知设置下方的重新开始引导（深色，1280 px）。*

自动检查：`features/onboarding_status.feature` 以及 `external/httpserver/bdd_onboarding_test.go` 和 `onboarding_http_test.go`；针对回调、回调粘贴、代理和取消路径的 `devin_auth_http_test.go`；用于选择器、非破坏性配置合并、本地化文案和导览的引导 Vitest 套件。

## 会话目标

当会话有目标时，输入框上方的板面上会出现一个目标标记，位于 git 的 `+A −D` 计数左侧，颜色为目标状态的颜色（进行中、已暂停、需要你、已完成、预算用尽）；指向它会显示状态和目标。它仅在 `/goal` 设置了目标后出现，此时位于 git 仓库之外的聊天也会获得该板面，并标明文件夹。它会打开目标菜单：完整目标、状态及其原因、主管的上次检查（当只读验证器确认时带有 **已验证** 标记）、需求清单、后续轮次、检查、活跃时间和 token，然后是检查它的模型（**检查者**），旁边是检查的推理级别（**推理**：`/goal -r` 选择的级别，否则为模型默认级别；没有级别的模型不显示推理），以及以图标表示的操作，并在提示中命名：**暂停**、**继续**、**编辑**（将 `/goal <new objective>` 连同同一检查器发送，这会替换目标并开始工作）和 **清除**（会先询问）。从输入框发送一个单独的 `/goal` 会打开同一菜单且不发送任何内容；没有目标时，它会提供一个用于设置目标的字段。主管启动的轮次会显示为单行目标行，而不是你的消息，实时显示并在重新加载后仍保留。**设置 → 会话目标** 用于编辑 `supervisor` 键。参见[会话目标与主管](../features/session-supervisor.md)。
## 设置：压缩模型

在 **设置 → 上下文压缩** 中，**摘要模型** 提供一个可搜索的下拉菜单，列出当前设置文档中配置的逻辑模型，包括未保存的模型编辑。选择一个模型或手动输入标识符。清空该字段以使用会话模型进行摘要。

![压缩模型下拉菜单](../../assets/compaction-model-open-dark-1280.png)

*在设置 → 上下文压缩中配置的摘要模型（深色，1280 px）。*

## 设置：Codex OAuth

- 在 **设置 → LLM 提供商** 中，类型为 **`type: codex`** 的行会隐藏通用的 **API base URL**、**API key** 和 **API key command** 字段，并显示 **Sign In with ChatGPT**。登录在保存之前即可使用，在新行上以及类型选择器刚切换到 codex 的已保存行上同样如此；后者只保留其 **`proxy`** 用于登录。
- 该按钮会启动 **`POST /coddy/providers/{name}/codex-auth/device`**，打开返回的官方验证页面，显示一次性代码，并轮询 **`GET .../device/{loginID}`** 直到完成或失败。如果浏览器阻止自动打开标签页，所显示的链接仍然可用。
- 连接状态来自 **`GET /coddy/providers/{name}/codex-auth`**。服务器 Codex CLI 登录不提供服务的行（多个 codex 行：该登录仅代表名为 **`codex`** 的行）在登录之前不会连接，字段会指明该登录所服务的行（状态的 **`cli_login_row`**）。**Sign Out** 仅通过 **`DELETE`** 删除 Coddy 管理的凭据，并取消仍在等待确认的登录，因此之后在保持打开的标签页中批准它不会存储任何内容；再次按下 **Sign In** 会以相同方式替换待处理的登录。服务器端的 Codex CLI 登录仍可能显示为兼容性连接。
- OAuth 令牌永远不会进入设置文档或浏览器。它们由服务器存储在 **`$CODDY_HOME/providers/<name>/codex-auth.json`** 下。

在 **设置 → LLM 提供商** 中，Kimi 订阅行提供设备登录，并将来源显示为 Coddy 或 Kimi CLI。**Sign out** 仅删除管理的凭据；如需退出 CLI，请在 Kimi CLI 中操作。表单会在该行保留返回的身份，已完成的登录可能需要 **Save all** 才能连接到已保存的配置。使用当前未保存的代理。平台行改用 API 密钥。

## 设置：NeuralDeep 登录

![登录后的 NeuralDeep 提供商行，正在使用 hub 签发的密钥](../../assets/settings-neuraldeep-signin-dark-1280.png)

*登录后的 NeuralDeep 提供商行，正在使用 hub 签发的密钥*

![端点下拉菜单警告：存储的登录来自另一个部署](../../assets/settings-neuraldeep-endpoint-mismatch-wide.png)

*端点下拉菜单警告：存储的登录来自另一个部署*

![提供商设置中的 Devin 浏览器登录](../../assets/web-ui/settings-devin-signin-dark-1280.png)

*带浏览器登录的 Devin 提供商行（深色，1280 px）。*

- 在 **设置 → LLM 提供商** 中，类型为 **`type: neuraldeep`** 的行会将自由文本 **API base URL** 替换为两个官方部署的下拉菜单，即 **`https://api.neuraldeep.ru/v1`**（俄罗斯）和 **`https://api.neuraldeep.tech/v1`**（国际镜像），写入 **`providers[].api_base`**；所选内容还决定下面的登录与哪个 hub 通信，并且登录会跟随表单中下拉菜单的当前值（设备启动会携带所选的端点），因此无需先保存。这也适用于类型选择器刚切换到 neuraldeep 的已保存行，例如 **`openai`** 行（**`config.example.yaml`** 随附的）：登录不会因该行保存时的类型而被拒绝，只有该行的 **`proxy`** 会延续到它，因为其端点和密钥属于另一种类型。存储的 **`api_base`** 若不是两者之一，会在字段下方标记并被服务器忽略。该行保留手动 **API key** 字段，并在端点选择器下方显示 **Sign In with NeuralDeep**（**`NeuralDeepAuthField`**）。登录是免粘贴的替代方案；显式密钥始终优先，控件会说明这一点，而不是假装登录已激活（来自状态端点的 **`source`**）。当胜出的密钥是服务器的 **`<NAME>_API_KEY`** 变量时（**`OPENAI_API_KEY`**，对于名为 **`openai`** 的行），注释会指明该变量，而不是指向空的 **API key** 字段。
- 在 **设置 → LLM 提供商** 中，OpenCode 行在 Go/Console 网关选择器下方提供 **Sign In with OpenCode**。设备代码保留在表单中 **Open sign-in page** 旁边，当操作员要求时它会打开 Console，因此可以在批准之前比较两个代码。在 Console 中确认代码和组织；然后该行会显示组织和 **Sign Out**。登录使用当前未保存的代理；该登录服务于两个网关，选择器决定该行的请求发往何处。**Save All** 保留行设置；令牌保留在服务器上。参见 [OpenCode](../features/opencode.md#从设置登录)；
- 同一部分中的 Devin 行提供 **Sign In with Devin**。PKCE 登录使用当前未保存的代理并打开验证页面。另一台机器上的浏览器可通过将完整的回调地址粘贴到登录链接下方的字段中来完成；然后该行会显示账户和 **Sign Out**。令牌保留在 Coddy 服务器上。参见 [Devin](../features/devin.md#从设置登录)；
- 该按钮会启动 **`POST /coddy/providers/{name}/neuraldeep-auth/device`**（hub 的 RFC 8628 设备流程，用于客户端 **`coddy`**；浏览器和 Coddy 服务器可能位于不同机器），打开返回的预填代码的门户页面，显示一次性代码，并轮询 **`GET .../device/{loginID}`** 直到完成或失败。
- 连接状态来自 **`GET /coddy/providers/{name}/neuraldeep-auth`**（仅掩码密钥），针对当前选择的端点读取（**`?api_base=`**）。当存储的登录由另一个部署的 hub 签发时（**`hub`** 与 **`endpoint_hub`** 不同），状态下方的注释会说使用它的请求会被拒绝，并要求为此端点重新登录；当显式密钥遮蔽登录时，该注释会被抑制。如果在此期间更改端点，已在进行中的登录会继续轮询（密钥来自流程启动时的 hub），随后读取的状态会标记不匹配。**Sign Out** 会尽力在 hub 上撤销密钥，然后通过 **`DELETE`** 删除本地凭据。
- 密钥永远不会进入设置文档或浏览器；它由服务器存储在 **`$CODDY_HOME/providers/<name>/neuraldeep-auth.json`** 下。层级模型从提供商表单末尾的 **Models** 列表添加到 **Logical models** 下，该列表在保存行之前使用此登录获取提供商目录（参见 [Settings: provider models](#设置提供商模型)）；CLI 流程（**`coddy providers login neuraldeep`**）和控制台的 **`/login`** 会自动将它们全部追加到配置中。

## 设置：提供商代理

![设置为直接连接的提供商行，其代理 URL 字段被禁用](../../assets/settings-provider-proxy-dark-1280.png)

*设置为直接连接的提供商行，其代理 URL 字段被禁用*

- **设置 → LLM 提供商** 中的每一行（包括 codex）都在 **Proxy URL** 字段上方带有 **Ignore system proxy** 开关（**`ProxySettingField`**）。两者都会编辑 **`providers[].proxy`**，即该行每个请求的路由：其补全、模型列表、账户用量、OAuth/设备登录和令牌刷新，以及退出登录撤销（[Provider proxy](../getting-started/configuration.md#提供商代理)）。**Sign In** 在 **Save all** 之前使用表单中的当前值：缺失值保留已保存行的路由，清空的字段遵循系统代理，`none` 表示直接连接。这包括刚切换到另一提供商类型的已保存行。Telegram 机器人的 **`gateways.telegram.proxy`** 读取方式相同，并在 **设置 → 网关** 的 Telegram 块中具有相同的一对（[Telegram gateway](gateway.md#代理)）。
- 该开关写入 **`none`**：该行直接连接，并忽略 Coddy 进程的 **`HTTPS_PROXY`**、**`HTTP_PROXY`** 和 **`NO_PROXY`**。开启时，URL 字段被禁用并显示 **Direct connection**。
- 关闭开关后，字段中的 URL 会将该行的每个请求通过该代理发送，而空字段（或存储的 **`inherit`**）显示 **Follows the system proxy**，即默认路由。
- 关闭开关会恢复该行在开启前所持有的内容，只要表单处于打开状态。文档只保留一个值，因此在 **Save** 之后，设置为 **`none`** 的行不再记得它所替换的 URL。
- 该值按原样通过 **`GET`** / **`PUT /coddy/config`** 传递。既不是关键字也不是代理 URL 的值会在 **Save** 时被拒绝，错误会指明可接受的值。

## 设置：提供商模型

![提供商的表单：Provider settings、折叠的 Advanced settings、Models 列表](../../assets/settings-provider-models-dark-1280.png)

*提供商的表单以该提供商所公布的内容结尾：加号会将 id 归入 Logical models，对勾标记已存在的项并再次将其移除，琥珀色 id 是提供商不再公布的项。*

- 提供商的表单（**设置 → LLM 提供商 → 行**）分为三块：**Provider settings**（id、类型、端点、密钥、登录、用量开关）、**Advanced settings**（凭据命令、代理、请求超时），在点击其折叠箭头之前保持折叠，以及 **Models**。
- **Models** 列出提供商所公布的内容，通过 **`POST /coddy/providers/models`** 使用表单当前所持有的行原样获取，因此未保存的行、仅存在于磁盘上的登录以及文档中尚不存在的提供商，都会像已保存行一样列出其模型（issue #335）。列表在表单打开时获取，当该行变为另一提供商时（在其 id 或类型停止变化之后）再次获取；对端点、密钥或代理的编辑会等待 **Models** 标题旁的刷新图标，它会以该行当前的状态立即获取。自动获取从不发送凭据命令，因此命令不会在输入一半时运行：刷新图标是唯一会运行它的获取方式。
- 每一行是模型 id、其右侧的上下文窗口（**`131k`**，工具提示中为准确计数）以及一个开关。窗口是提供商报告的那个；对于提供商未报告窗口的模型，则是 Coddy 的模型目录（见下文）中该 id 所对应的窗口。加号会将 **`provider/id`** 添加到同一未保存文档中的 **Logical models**，并将该窗口写入新模型的 **Context window**；当目录中该 id 的窗口相同时，其 **Max tokens** 数值也会一并带入，对于读取图像的模型则会开启 **Multimodal**。对勾表示模型已存在，点击它会再次移除该模型。在 **Save all** 之前不会到达 `config.yaml`。Logical models 在此提供商下列出但提供商不再公布的 id 会以琥珀色显示、带对勾，原因在其工具提示中。标题下方的行仅在没有任何内容可列出时出现：获取期间、出错时、答案为空时，或对于没有 id 和类型的行。超过八个模型时，筛选字段会缩小列表范围。
- 模型目录是一张包含 OpenAI、Anthropic 和 OpenCode 模型的表，每个模型都有上下文窗口、初始最大令牌数以及是否读取图像。它由 [models.dev](https://models.dev) 构建并嵌入二进制文件，因此 Coddy 运行时不会为其下载任何内容，比构建更新的模型在后续发布之前没有数值。对于提供商未报告窗口或报告与目录相同窗口的模型（如 Anthropic 的列表），列表会读取它，并同时采用两个数值或都不采用，因为最大令牌数是针对目录自身的窗口计算出来的。对于提示词和答案共享窗口的模型，它最多为窗口的五分之一，即高于默认压缩阈值的部分，因为当提示词和最大令牌数不能同时适配窗口时，提供商会拒绝请求。具有自身输入预算的模型会获得其完整输出限制。会话从不读取目录：添加时带有数值的模型会将其作为自身的设置保留，而逻辑模型的 **Context window** 字段显示提供商报告的内容。
- 打开的行是地址的一部分：**`#/settings/providers?id=demo`**、**`#/settings/models?id=demo%2Fqwen3.8-27b`**。打开行会写入它，返回会移除它，重命名行会重写它，编辑地址会打开它所指向的行。搜索会添加 `&field=<anchor>`，用于它带你前往的设置（[Settings: search](#设置搜索)）。指向不存在行的地址会显示列表并丢弃该名称。当行打开时，抽屉的头部会显示 **Provider settings**（或 **Model settings**），后面有一个返回列表的返回箭头，在桌面和手机上都是如此。
- **设置 → Logical models → 行** 分为四块：**Model**（提供商、模型 id、上下文窗口、多模态输入）、**Generation**（最大令牌数、温度、流式响应）、**Reasoning**（层级和默认值）以及 **Tools**（此模型上的会话获取其工具所使用的工具集）。**Provider** 是文档中 `providers[].name` 行的组合框（尚未列出的名称仍可输入），**Model id** 是普通字段；两者在第一个斜杠处组合为 `provider/id`。从提供商所公布的内容中选择 id 是在提供商自己的表单中完成的。
- **Context window (tokens)** 也会询问提供商。保持为 0 时，模型会依据提供商列表所报告的窗口进行测量（否则为 128000），因此该字段会将该数值显示为其占位符，例如 **`131072, reported by demo`**，并且不写入任何内容。旁边的刷新图标会再次询问并将数值写入 `max_context_tokens`，从而将其固定；当提供商未报告该 id 的窗口或请求失败时，字段下方的注释会加以说明。

![逻辑模型表单：普通模型 id 和提供商报告的上下文窗口](../../assets/settings-model-form-context-window-dark-1280.png)

*逻辑模型是提供商加上其 API 期望的 id；上下文窗口来自提供商。*

- **Tools** 指明模型的工具集（**`models[].toolset`**，参见 [Toolsets](../getting-started/configuration.md#工具集)）：**Toolsets** 标签页的预设列表加上 **None: every tool**。预设不再拥有的工具集仍会保留在选项中，因此表单从不会在操作员背后更改它，配置检查随后会报告它。
- **设置 → Toolsets** 保存预设（**`toolsets[]`**），每个预设一行，以其 **Name** 命名：先是 **Allowed tools**，然后是 **Disallowed tools**，各自是一个名称列表，带有 **Add** 和垃圾桶。条目是自由文本，写为确切的工具名、`*` 或 `prefix*`，MCP 工具写为 `server__tool`（`context7__*` 是该服务器的所有工具）。没有端点会列出构建中的工具，因此表单不提供建议，与任何工具都不匹配的名称会像其他名称一样保存；`coddy -t` 会对此发出警告。在设置中添加的预设从两个空列表开始，这会提供该模式所能提供的一切，而再次清空的列表会从 `config.yaml` 中省略。重命名预设会在命名它的模型上重命名它。

![Toolsets 标签页：一个带有小型本地模型允许工具的预设](../../assets/settings-toolsets-dark-1280.png)

*一个工具集：小型本地模型所需工具的白名单，以及在其之后应用的黑名单。*

- **Usage limits panel** 开关（`providers[].usage_limits_panel`）仅为具有用量来源的提供商类型（`neuraldeep`、`codex`、`devin`）渲染；对于其他类型，它不会改变任何内容。
- 在桌面上，设置抽屉宽 680px：足以让 **Appearance** 上的三张主题卡片排成一行，并为表单提供可读的行宽。
- 自动化检查：**`ProviderModelList.test.tsx`**（使用表单当前持有的行且不带命令进行获取，仅针对新 id 或类型重新获取，添加和移除，过期行，错误行，上下文徽章，筛选，包含规则）、**`ModelField.test.tsx`**、**`ContextWindowField.test.tsx`**（报告的窗口作为占位符，显式获取写入它，针对已更改 id 的迟到答案被丢弃）、**`SettingsArraySection.test.tsx`**（双向的地址，返回链接）、**`SettingsSection.test.tsx`**（提供商表单的三个块、逻辑模型表单的四个块、其工具集和 Toolsets 标签页的英语和俄语版本、添加时播种的报告窗口、按提供商类型的用量开关、行表单在标签页切换后不保留）、**`features/provider_models_web_ui.feature`**（**`external/ui/bdd_provider_models_ui_test.go`**）、**`features/toolsets_web_ui.feature`**（**`external/ui/bdd_toolsets_ui_test.go`**），在 API 侧还有 **`features/provider_models_fetch.feature`** 与 **`external/httpserver/providers_models_http_test.go`**。在 **390px** 下的实机检查：提供商表单仅垂直滚动（`.settings-scroll` 的 `scrollWidth` 等于 `clientWidth`），长 id 会省略显示。

## 设置：标签页和表单布局

- 标签页按组排列：**Appearance**、**Sessions** 和 **System**（只读页面，参见 [System](#系统设置标签页)）；模型来自何处（**LLM providers**、**Logical models**、**Toolsets**）；运行它们的循环（**ReAct loop**、**Session goal**、**Context compaction**、**Memory**）；智能体能做什么（**Command safety (decisions)**、**Tools and permissions**、**MCP servers**、**Skills**、**Subagents**、**Hooks**）；在没有人员于输入框时运行它们的内容（**Scheduler**、**Gateways**）；以及进程的运行（**Logger**、**Metrics**、**Prompts**）。**Prompts** 保存提示词模板和指令文件，并且始终位于最后。**Metrics**（Prometheus 导出器）仅由带有 `metrics` 标签构建的二进制文件提供；任何其他二进制文件都会省略该标签页，而其表单的保存会保持 `config.yaml` 中的 `metrics` 章节原样。
- **Sessions** 在已存储会话表格上方编辑会话包的存储位置（**Storage**，`sessions.dir`）；表格本身会立即生效，存储路径通过 **Save all** 保存。
- 标签页自身的 **Enabled** 开关会打开表单，位于每个块之上（Subagents、Hooks、Memory、Context compaction、Scheduler）。其他字段按含义放在字段集中，例如 **ReAct loop** 上的 **Model and turns**、**Retries**、**Stream timeouts**、**Loop guard** 和 **Usage limits**；列表（定义目录、回退模型、组件层级）是它们旁边的独立块。在嵌套块内部，列表会保留其框架（Telegram 管理员、用户组和按聊天覆盖）。
- 值列表就是其输入项，每个都带有一个垃圾桶按钮，下方是 **Add**：没有逐行标签，行与行之间也没有分隔线。对象列表的条目是独立的框架，没有编号标题。
- 值的映射（例如 **HTTP requests** 的 **Default headers**，`tools.http_request.default_headers`）每个条目一行：名称和值并排，一个垃圾桶按钮，行下方是 **Add**。没有名称的行会保留在屏幕上但不进入保存的配置；空值会被保存，对于 header 而言意味着"省略此 header"。
- 空路径字段会显示默认值指向何处（`${CODDY_HOME}/sessions`、`${CODDY_HOME}/memory`）或写入示例。
- 自动化检查：**`SchemaForm.groups.test.tsx`**（开关优先，兜底组，组放置在其第一个字段处，后续键保留，列表）、**`SchemaForm.map.test.tsx`**（映射的行按文档顺序，就地编辑，没有名称的行，两行同名，从外部替换文档）、**`settingsSections.test.ts`**（顺序，Prompts 最后，Logger、Gateways 和 Scheduler 标签页，Sessions 标签页拥有 `sessions`）、**`TestUISchemaRootPropertyOrder`**（`internal/config/jsondto_schema_test.go`）、**`settingsDrawerRhythmCss.test.ts`**（列表行与其输入项一样高）。
## 设置：搜索

![1280 px 的设置抽屉：分区栏顶部的搜索字段中填有查询 proxy，内容列中是匹配的设置，并标出单词 proxy；第一个高亮项是演示提供商在 LLM providers 下的 **Proxy URL**，即 Advanced settings](../../assets/web-ui/settings-search-results-dark-1280.png)

*宽抽屉上的查询 `proxy`：字段位于分区栏顶部，匹配的设置显示在标签页的位置。第一个结果是 `demo` 提供商的 **Proxy URL**，它被高亮；最后三个是通过各自的描述匹配到的。*

![390 px 手机上的设置：搜索字段中填有查询 proxy，位于磁贴开始处，匹配的设置显示在其下方、磁贴的位置，演示提供商的 Proxy URL 首先高亮](../../assets/web-ui/settings-search-results-dark-390.png)

*手机上的同一查询：字段位于磁贴的位置上方，结果占据该位置。`demo` 提供商的 **Proxy URL** 首先高亮。*

- **字段。** 一个搜索字段可找到任意标签页中的一项设置：一个字段、一组字段、一个标签页，或 **LLM providers** 或 **Logical models** 中的一行。在宽度为 1200px 或更大的抽屉上，它是分区栏中的第一项；在更窄的屏幕上（平板、手机、Telegram Mini App），它位于磁贴上方。`/` 在焦点位于文本框或选择框之外且没有打开对话框时，会把光标放入其中（在复选框或按钮上也可用）。在窄屏幕上打开某个分区时，`/` 会先返回磁贴，并在字段绘制完成后聚焦它。空字段内的 `/` 徽标会说明哪个键执行此操作：它在字段为空且没有焦点时显示，适用于所有宽度（跳转后焦点位于设置上，因此徽标会回来），当字段有焦点或包含文本时隐藏，纯触摸设备从不显示它。首次读取设置期间，该字段被禁用。
- **匹配的内容。** 查询中的每个词都必须出现在设置自身的名称、其配置键、其周围名称（其标签页、其行以及它所在的分组）或它的描述中，并按页面绘制它们的方式匹配。大小写无关，**ё** 可匹配 **е**；除此之外不做折叠，也不做模糊匹配。`max_turns` 和 `agent.max_turns` 可找到 **Max turns**，`timeout_ms` 可找到超时字段，且在每种语言中都如此。除配置键外，只匹配页面语言，因为匹配项会以标记形式显示，而另一种语言的文本不在屏幕上：在俄语页面上，`таймаут` 会找到俄语名称，不会搜索英语名称。匹配的词会在结果中标记。
- **顺序与限制。** 结果会显示设置的名称、它所在的位置（标签页、行和分组，用 `›` 连接）及其配置键。当匹配的词只出现在第二名称中时，该名称会以中点跟在主名称之后（`· Provider` 表示模型的 id 输入框）。仅通过描述找到的设置还会显示描述，并从该词稍前处开始。结果分为三组，每组按抽屉中的顺序排列：自身名称或配置键包含所有词的设置（标签页和分组在前），然后是需借助周围名称的设置，最后是需借助描述的设置。最多显示 50 条结果，随后是一行统计其余数量的文字：**还有 N 项设置匹配 - 添加一个词以缩小列表**。没有任何匹配的查询会显示 **没有匹配的设置；请尝试其他词或更少的词**。屏幕阅读器会在输入暂停后，一次性读出找到的数量、已显示的结果和统计出的其余数量。
- **行。** **LLM providers** 或 **Logical models** 的字段只存在于打开的行内，因此搜索会在每一行中按行名找到它一次：`proxy` 会找到每个拥有代理的提供商的代理。没有名称的行以及共享同一名称的行不会被提供，因为地址通过行名打开行，而服务器拒绝保存两个同名行。某行未绘制的字段（Codex 提供商的 API 密钥、没有用量来源的类型的用量面板）不会为该行提供。
- **字段分组。** 一组字段本身就是一个结果，可通过其名称找到：例如某个提供商的 **Advanced settings** 或 **ReAct loop** 的各组、一个嵌套对象或列表，以及提供商行或模型行表单中的各组。选择其中一项会打开其标签页和行，在它本身折叠时展开它，滚动到它，围绕整个分组绘制描边，并将焦点放在该分组上。
- **手写分组。** 带有由 API 填充的手绘分组的标签页，可通过该分组的标题及其 (i) 后的描述找到，绝不会通过分组内容找到：Appearance 上的 **Theme**、**Language** 和 **Notifications**，**Session management**、**MCP discovery** 和 **MCP servers**，**Skill auto-discovery**、**Marketplaces** 和 **Installed skills**，Subagents 上的 **Definitions**，提供商行的 **Models** 分组，以及 System 上的 **This installation**、**Subsystems**、**Disk** 和 **Activity**。`mcp.project_trust` 会找到 **MCP discovery**，`skills.auto_discovery` 会找到 **Skill auto-discovery**。会话表的筛选器、MCP 工具栏和编辑器、Skills 安装框以及每条记录（服务器、已安装技能、子智能体定义、存储的会话、提供商列出的模型）都不会被搜索。
- **结果会打开什么。** 点击或轻触结果，或使用 ArrowUp 和 ArrowDown 移动后按 Enter。标签页会打开；若结果有对应行，则行也会打开；折叠的分组会展开（提供商的 **Advanced settings**）；设置会滚动到面板中部（高于面板的分组按顶部对齐，因此其名称和首批字段可见，且 12px 的外边距可避免描边在面板边缘被裁切）；其周围会绘制一圈描边，持续 1.6 秒，在浅色覆盖层上以 2px 强调色轮廓距设置 3px 呈现，并淡入淡出（在减少动态效果下保持静止）；焦点会移动。使用鼠标时，焦点会移到与设置同名的控件：**Proxy URL** 会把光标放入 URL 输入框，而不是旁边的 **Ignore system proxy** 开关；模型的 id 输入框会聚焦 id 输入，而不是 **Provider**。在手机或任何纯触摸设备上，以及对于行或分组，焦点会改为移到设置的容器上，这样键盘不会在设置上方升起。作为标签页的结果没有可绘制描边的容器：标签页会在顶部打开，其面板获得焦点。再次选择同一结果会再次执行所有这些操作。另一个标签页中的结果会卸载当前打开的标签页，就像点击分区栏一样，因此标签页在表单之外保留的本地工作（正在编辑的 MCP 服务器 JSON、等待代码的设备登录）也会以同样方式丢失。
- **保留的查询。** 选择后查询会保留，结果会隐藏，因此不会遮住设置。点击或轻触字段，或在文本框外按 `/`，结果会重新出现并高亮所选项；然后按 ArrowDown 和 Enter 可打开下一项。结果显示时点击字段内，会使高亮结果保持在原位。清空字段（使用其清除按钮或删除文本）会像按 Escape 一样清除搜索。点击分区栏或轻触磁贴会隐藏结果并保留查询。关闭抽屉会忘记查询。结果显示时，标题显示为 **Settings** 且没有返回箭头，即使位于打开的行上也是如此。
- **指向设置的链接。** 选择结果还会把该设置写入地址，因此可以立即发送：`#/settings/agent?field=agent%3Amax_turns`，或列表行中的设置为 `#/settings/providers?id=demo&field=providers%3Aproxy`（`id=` 仅用于 **LLM providers** 或 **Logical models** 中的行，锚点会进行 URL 编码）。地址会就地重写，不新增历史记录。作为标签页的结果不会写入 `field=`。当你转到别处（点击分区栏或磁贴、标题栏的返回箭头、另一行、关闭抽屉）时，地址会丢失该设置，下一次选择会替换它。要分享设置，请选择其结果并复制地址。打开这样的地址（粘贴的链接、重新加载、中继自己的 Settings、Telegram Mini App）会打开标签页和行，并像选择结果一样带你到该设置；页面不认识的锚点（来自另一版本的链接）只会打开标签页和行。地址命名的是设置，绝不是查询。
- **Escape 和 Back。** 结果显示时，无论焦点在哪里，Escape 都会清除查询。在字段中、结果隐藏且字段内有查询时，Escape 会清除查询。否则该键按抽屉的规则处理（[Layout](#布局)）。在 Telegram Mini App 中，Back 会执行相同步骤：先清除结果正在显示的查询，然后从分区返回磁贴，然后像之前一样关闭 Settings。结果显示时，其下方的标签页不响应任何按键，不绘制提示或标签编辑器，并保留未保存的本地工作（MCP 服务器的 JSON、等待代码的设备登录、Skills 安装查询），这些工作会在结果消失后恢复。
- **值。** 搜索读取名称、配置键和描述，绝不读取值：URL、令牌、提示词，或标头的名称或值都找不到任何内容；密钥的名称可搜索，而其值不可搜索。唯一读取的值是两个列表中各行的名称，而这些名称列表本来就会显示。表单隐藏的键不会被搜索。搜索和选择结果不会触碰表单：未保存的更改和高亮的 Save 保持原样。
- **二进制无法运行的标签页。** 抽屉根据服务器提供的 schema 绘制，服务器会省略构建二进制时未包含的功能设置：**Metrics** 仅存在于使用 `metrics` 构建标签构建的二进制中。此类标签页不会被提供，其设置也不会出现在结果中，尽管设置文档仍保留其键，表单会按原样写回它们。
- **中继。** 中继自己的 Settings 也以同样方式搜索，搜索范围是中继的标签页（**Appearance**、**Swarm relay**、**Logger**；它没有 **System** 标签页）；通过中继访问的节点则使用其自己的 schema 搜索。
- **Telegram 的 Notifications。** 在 Telegram Mini App 内，Appearance 标签页不会绘制 **Notifications** 分组，因此搜索也不会提供 Notifications 结果。
- 自动检查：**`settingsSearch.test.ts`**（词、三个分组和限制、标记的词、描述行、第二名称、没有名称或共享名称的行、值从不建立索引、英语和俄语、按名称、键和描述找到 Metrics 标签页、服务器省略的标签页不会通过任何内容找到、System 标签页及其四个分组在未索引任何状态值的情况下找到、测量包含 40 个提供商和 200 个模型的文档）、**`SettingsSection.test.tsx`**（将绘制结果与所提供 schema 上的索引进行比较，逐个标签页、逐行，在英语、俄语和 Telegram 内：每个框一次、每个框的名称和描述、通过三种变异证明丢失条目、过期条目和名称变更会使其失败）、**`scheduler/hashRoute.test.ts`** 和 **`App.settingsLink.test.tsx`**（地址在有行和无行时写入并读回、链接打开其设置、来自另一版本的链接）、**`SchemaForm.groups.test.tsx`**（每个字段的框、没有对应项的对象列表条目、折叠分组在内部某项被显示时展开）、**`components/useEscapeCloses.test.tsx`**（隐藏窗格中的层，`PanePortal`）、**`Settings.test.tsx`**（`/` 在宽屏和窄屏上、输入、两次选择同一结果、提供商行及其折叠分组、鼠标和触摸的焦点、保留未保存编辑、Escape 的步骤、Telegram 的 Back、显示结果下的提示、标签编辑器和 Skills 安装结果、索引构建后编辑的内联列表、无匹配、状态行、俄语、中继，以及使用不包含 Metrics 的二进制时的 Metrics 标签页、延迟到达的分块后显示的 System 分组）、**`settingsFieldsetCss.test.ts`**（减少动态效果下仍保留的描边、与文档搜索共享的规则）、**`TestUISchemaTestdataIsCurrent`**（`internal/config/jsondto_schema_test.go`：比较所读取的所提供 schema 的已校验副本）以及 **`features/web_ui_settings_search.feature`**。浏览器检查：`npm run check:settings-search`（由 CI 在 Chromium 中运行，位于 [设置：打开与保存](#设置打开与保存) 下），以及 [检查 Telegram Mini App](#检查-telegram-mini-app) 的 `settings` 场景（需手动运行）。

## 设置：打开与保存

![保存后 Save 全绿，标签页上方没有一行文字](../../assets/settings-save-done-dark-1280.png)

*一次成功的保存：Save 按钮变绿两秒，屏幕上其他内容不变。*

![尚未保存的编辑：Save 突出显示，按钮前一行显示 Unsaved changes](../../assets/settings-unsaved-dark-1280.png)

*尚未保存的编辑：Save 突出显示，按钮前一行说明了这一点。*

![浅色主题中的相同状态](../../assets/settings-unsaved-light-1280.png)

*浅色主题中的相同状态。*

- 抽屉会立即打开其地址指定的标签页。应用会保留此页面中首次打开 Settings 时读取的 schema 和配置，因此之后打开时会在第一帧就用该副本绘制标签页，而不是先绘制空抽屉再重建。该副本会跟随服务器：每次 **`config_reloaded`** 以及事件流的每次重连都会在后台重新读取它。
- 页面首次打开时还没有可绘制的内容。它会把请求的标签页显示为骨架屏：灰色字段、分区栏中的灰色行，或窄屏上的灰色磁贴，绝不会用 Appearance 标签页替代显示。
- 没有未保存编辑的表单会在较新副本到达时接受它，因此智能体或另一个浏览器所做的更改会显示出来。含未保存编辑的表单会保留这些编辑；**Reload** 会采用服务器上的内容并丢弃它们（在 Reload 进行期间输入过的字段会保留已输入的内容）。
- 保存会立即将其写入的内容放入保留的副本中，因此保存后马上关闭并重新打开抽屉，即使尚未再次读取服务器，也会显示已保存的值。
- **Save all** 会在按钮本身上报告成功：按钮变绿两秒，不显示任何文字行。服务器拒绝的保存会在标签页上方用文字说明原因，按钮保持原样。
- 表单中的更改只有在 **Save all** 后才会生效，抽屉会在等待期间说明这一点：按钮突出显示，填充强调色并带有环绕描边，按钮前一行显示 **Unsaved changes**。这也包括通过 **Sign In with ChatGPT** 或 **Sign In with NeuralDeep** 登录的提供商：登录会立即存储凭据，但提供商行会随保存到达 `config.yaml`。将编辑恢复原状（重新输入某个值、清空字段、再次关闭开关）不算编辑，按钮会恢复自己的外观。
- 在有未保存编辑时关闭抽屉（使用关闭按钮、Escape 或点击旁边）会先询问：**Save and close** 或 **Keep editing**；**Reload** 仍是丢弃这些编辑的方式。重新加载或在有未保存编辑时关闭页面会让浏览器请求确认。切换到另一个环境不会询问：编辑属于正在离开的服务器，**Save all** 绝不会把它们发送到另一台服务器（如果页面重载到另一台服务器的过程被阻止，则会拒绝保存并提示重新加载）。在保存进行期间输入的编辑相对于该保存写入的内容仍为未保存，**Save and close** 会让抽屉继续打开并停留在该编辑上。
- 保存会把更改写入 `config.yaml`。你未触碰的值会保留其拼写：`${CODDY_HOME}/memory`、`~/.agents/skills`、`${OPENAI_API_KEY}`、引号、写在一行中的列表；提供商或模型条目只保留它指定的字段（重命名的条目也是如此），而进程与文件不同的运行方式（命令行标志、`coddy serve` 填入的中继地址）不会写入其中。缩进和空行不会保留：文件会以两个空格缩进返回，各章节之间没有空行。参见 [配置](../getting-started/configuration.md#环境变量引用)。
- 你未触碰的内容会以当前浏览器读取的内容为基准来衡量：文档会指明其来源配置，因此当另一个浏览器、智能体或手动编辑在你打开表单后保存了某些内容时，你的保存会保留那些值，只写入你的更改。你编辑过的列表会按你离开时的样子写入。
- 自动检查：**`Settings.unsaved.test.tsx`** 与 **`features/web_ui_settings_unsaved.feature`**（编辑会高亮 Save 直到保存、将编辑恢复原状、在有未保存编辑时关闭和按 Escape、离开页面、不会保存到另一台服务器、保存运行期间的编辑和开关、较新副本到达已恢复原状编辑的表单）、**`App.settingsUnsaved.test.tsx`**（点击背景会先询问）、**`settingsDirty.test.ts`**（什么算作编辑，包括空标头值）、**`Settings.test.tsx`**（第二次打开从副本绘制且不发起请求、宽屏和窄屏上的骨架屏以及首次读取失败后的骨架屏、重新加载会到达未触碰的表单而绝不会到达未保存编辑、绿色 Save 按钮和错误行、绿色保存后的失败保存、保存运行期间到达的编辑和较新副本、在提供商之后以列表打开的 Logical models、首次打开时指向某一行的深层链接、保存后马上重新打开时显示已保存的值、Reload 期间输入的编辑）、**`settingsConfigStore.test.ts`**（每次打开只读取一次、重新加载通知、按顺序应用响应、读取失败时保留副本、比失败读取更旧的副本会再次读取、已保存文档作为副本）、**`settingsSaveStateCss.test.ts`**（减少动态效果下保留的绿色和高亮状态、没有成功行、骨架屏从主题文本颜色着色）、**`features/config_save_round_trip.feature`** 与 **`external/httpserver/bdd_config_round_trip_test.go`**（没有编辑的保存会逐字节保持文件原样、一次编辑更改一行、从较早读取配置的浏览器保存不会撤销另一次保存）以及 **`internal/config/yaml_comments_test.go`** 中的保存测试。浏览器检查：**`npm run check:settings-unsaved`** 针对一次性的 `coddy serve`（**`scripts/settings-unsaved-check.mjs`**：高亮的 Save 及其前一行适合页脚，且 Save 可被轻触，从 320px 手机横竖两个方向到 1280px 桌面，在英语和俄语中；它不写入任何内容）。
- 搜索的浏览器检查：**`npm run check:settings-search`**（`scripts/settings-unsaved-check.mjs search`，以及与 Telegram 检查共享的断言在 `scripts/settings-search-assert.mjs` 中）在真实浏览器中驱动智能体和中继，使用英语和俄语，从 1280px 桌面到手机横竖两个方向。它会搜索配置键（`max_turns`、`proxy`、`temperature`、`pairing`），这些键在每种语言中都能匹配，并按锚点选择每个结果；设置必须被描边、位于其滚动区域的可见部分内并持有焦点（使用鼠标时是与设置同名的控件，在触摸设备上是未聚焦文本控件的容器），包括提供商折叠的 **Advanced settings** 中的代理，此时光标位于 URL 输入框，以及选项指定的模型行中的设置；并且在中继自己的页面上，`pairing` 会到达 **Swarm relay** 标签页。它还会对查询的生命周期进行核对：选择后保留，并通过 `/` 或轻触恢复且选中所选项；对于 `proxy`，名称匹配先于描述匹配；结果显示时点击字段会保持高亮，而按 Enter 会打开它；清空字段会清除搜索；标签页在结果下方保持其滚动位置；Escape 会逐层后退（结果、行的后退步骤、字段中的查询、抽屉）。它还会用 `field=` 检查地址、打开链接、搜索分组以及 `/` 徽标。它会在 System 分块被延迟 1,500 ms 时，按名称找到 **System** 标签页，并按标题找到其中一个分组：分块到达后，该分组会被描边并聚焦；如果在此之前离开，则会取消跳转。测试环境是两个 coddy 服务器，各自拥有自己的 `HOME` 和 `CODDY_HOME`，且不调用任何模型。设置 `CODDY_BIN` 而未设置 `CODDY_SERVE_URL` 时，脚本会从该二进制（使用 `TAGS="http ui swarm"` 构建；`scripts/settings-search-stands.mjs`）在空闲端口和临时主目录上启动两者，通过环境提供中继的客户端令牌，并在退出时停止两者，包括失败时。智能体有一个没有密钥的 `openai` 提供商 `demo` 和三个逻辑模型（`demo/qwen3-27b`、`demo/qwen3-8b`、`demo/llama-3`）。使用 `CODDY_SERVE_URL` 和 `CODDY_RELAY_URL` 时，它会使用你启动的测试环境：一个拥有提供商 `demo` 和两个或更多逻辑模型的智能体，以及一个中继（`swarm.enable: true`），该中继即使在回环地址上也要求客户端令牌，脚本会从 `CODDY_RELAY_TOKEN` 获取；在打开浏览器之前，它会检查两个测试环境，并在智能体未提供其 schema、未设置 `CODDY_RELAY_URL` 或它未作为中继响应、或中继拒绝令牌时，以一条普通消息停止。CI 会在 Chromium 中于 Tests on PR 作业针对 `http,scheduler,ui,memory,cli,swarm`（`CODDY_BIN="$PWD/build/coddy" npm --prefix external/ui run check:settings-search`）的步骤中运行该检查。WebKit（`CODDY_ENGINE=webkit`）和 `settings` 的 `npm run check:telegram` 场景仍需手动执行：当搜索、抽屉布局、隐藏标签页机制或 Mini App 的 Back 发生变化时运行它们，并在拉取请求中报告命令、引擎和结果。
## 设置：字段说明

![Max tokens 的说明显示在其 (i) 上方的提示中](../../assets/settings-field-hint-dark-1280.png)

*将鼠标悬停在字段名旁的 (i) 上，会在其上方并覆盖页面显示该字段的含义。*

- 设置表单只显示字段名和控件，仅此而已。字段的含义藏在名称右侧的 **(i)** 后面：悬停它（或用 Tab 聚焦到它），说明就会显示在它上方；在触摸屏上轻点它。移开、按 Escape、在别处轻点或滚动表单都会关闭说明。字段组（提供商的 **Models** 列表、Skills 的 **Marketplaces**、**MCP discovery**、Subagents 的 **Definitions**，以及生成表单中的对象和列表区块）也以同样的方式携带说明，位置在图例旁。
- 提示绘制在整个页面之上，而不是设置面板内部，因此永远不会被面板裁剪，也永远不会给面板添加滚动条。报告状态的文本（拉取结果、登录码、关于已存储值的警告）仍留在表单中其控件下方。
- 自动化检查：**`FieldHint.test.tsx`**。实时检查：在 **1280px** 下悬停一个 (i)，在 **390px** 下轻点一个 (i)；提示是 `<body>` 的子元素，位于 (i) 上方 8px 处并以其为中心，且 `.settings-scroll` 保持 `scrollWidth` 等于 `clientWidth`。

## 设置：布尔开关字段

- 设置表单中的每个开/关选项都通过共享的 **`SwitchField`**（**`external/ui/src/ui/settings/SwitchField.tsx`**）渲染：**`Switch`** 控件和一个标签单元格位于同一个两列网格中（**`.settings-switch-field`**）。这涵盖 **`SchemaForm`** 中由 schema 驱动的布尔值（例如 **Logical models → Multimodal** 和 **Stream responses**、**Tools and permissions → Background tasks → Enabled**、**Gateways → Telegram** 的各项标志）以及 **Skills → Skill auto-discovery** 行。
- 标签与开关处于同一水平线（垂直中心偏差在 1px 以内，间距 8px），字段的说明是标签右侧的 (i)，与其他所有字段一致。
- 标签是真正的 **`<label for>`**：点击文本会切换开关；点击 (i) 只会打开说明。开关由该标签命名（**`aria-labelledby`**）；当可见文本是状态文案（**Enabled** / **Disabled**）时，该行会传入显式的 **`ariaLabel`**，其优先级更高。
- 自动化检查：**`SwitchField.test.tsx`**（结构与 CSS 规则）、**`SchemaForm.switch.test.tsx`**、**`SkillsSection.autoDiscovery.test.tsx`**。实时检查：在 **1280px** 和 **390px** 下打开一个模型表单，对每个 **`[role=switch]`** 比较开关的 **`getBoundingClientRect()`**、**`.settings-switch-field-label`** 及其 **`.field-hint`**（中心偏差 ≤ 1px）。

## 实时重载配置

![带选项卡导航的设置面板，ReAct loop 选项卡](../../assets/screenshot-fullhd-settings.png)

*带选项卡导航的设置面板，ReAct loop 选项卡：其字段按含义分组到 fieldset 中*

- Coddy 会热重载自身的配置，而且不只来自此页面：智能体的 **`config_commit`** / **`config_rollback`** 工具会在轮次中途重写配置，安装技能会重写配置，另一个浏览器标签页可能正在保存设置表单。SPA 从配置派生的任何内容，包括输入框的 **Model** 选择器、**`multimodal`** 附件按钮、斜杠命令名称，都只在启动时读取一次，否则会一直保持陈旧直到页面重新加载（issue **#161**）。
- 每当有交互式 `question` 等待开始或结束时，**`GET /coddy/events`** 都会携带 **`event: session_question_pending`**，其中仅包含 **`{object, sessionId}`**。该事件和 `questionPending` 字段是拥有该交互式等待的 Coddy 服务器进程的进程本地状态；**`event: config_reloaded`** 在每次实时配置替换后执行。**`chat/serverEvents.ts`** 将这些转换为可选的 **`onQuestionPending`** 和 **`onConfigReloaded`** 回调，通过该标签页与其他标签页共享的流传递（见 [跨标签页共享的服务器事件](#跨标签页共享的服务器事件)）；对于前者，**`App.tsx`** 会重新读取 History；对于后者，它会递增 **`configEpoch`**。两个由配置派生的拉取，即 **`GET /v1/models`** 和 **`GET /coddy/slash-commands`**，都依赖这个计数器，因此它们会一起重新读取。设置中的 **Save** 按钮通过 **`onConfigSaved`** 递增同一个计数器，这就是为什么本地保存和远程替换的行为完全一致。
- 该事件不携带模型列表：发生变化的内容已经在那两个端点之后，服务器只在新配置生效后才发布该事件，因此重新读取不会读到即将下线的旧配置。
- 当事件流本身不可用时（旧版服务器、吞掉 SSE 的代理），没有其他机制会重新读取模型列表：页面会回退到它此前完全一样的行为，保持陈旧直到重新加载。这里的流是一种优化，而非保证，也没有为此增加额外的轮询。
- 该事件还会重新读取 **Settings** 抽屉保留的配置副本（见 [设置：打开与保存](#设置打开与保存)）。已打开的表单只有在自身没有编辑时才会接受它：在未保存的编辑下重新拉取会丢弃工作，因此这些编辑会保留，而表单中的 **Reload** 控件是刻意丢弃它们的方式。
- 自动化检查：**`serverEvents.test.ts`**（事件到达回调，且在没有回调时也无害）、**`features/config_reload_broadcast.feature`** 与 **`external/httpserver/bdd_config_reload_test.go`**（公告到达每个已打开的客户端，且在其上读取的模型列表已包含新模型），以及 **`external/httpserver/events_http_test.go`**（帧格式、防止公告未实际发生的替换的保护，以及公告时刻斜杠命令列表是新鲜的）。

## 登录（`httpserver.login`）

除非服务器有账户，否则关闭；没有账户的服务器渲染结果与以往完全一致。

![Web 界面的登录屏幕：深色主题下带用户名和密码字段的卡片](../../assets/webui-sign-in-dark-1280.png)

*配置了 `httpserver.login` 时，匿名浏览器看到的登录屏幕。*

- **决定因素：** **`AuthGate`**（**`external/ui/src/ui/auth/AuthGate.tsx`**，在 **`main.tsx`** 中包装 **`<App/>`**）在启动时调用一次 **`GET /coddy/auth/me`**。没有 **`authenticated`** 的 **`login_required`** 会渲染 **`SignInScreen`** 而不是应用；其他情况渲染应用。在答案尚未返回期间，它渲染一个空的 **`.auth-boot`** 背景而非应用，因此对话记录永远不会先显示再被拽走。
- **无法应答的服务器**，即网络错误，或来自在此功能存在之前构建的 `coddy serve` 的 **404**，会被视为“此处没有登录”，因此旧版服务器可以继续与新页面配合工作。
- **该屏幕**（**`SignInScreen.tsx`**、**`.auth-screen`** / **`.auth-shell`** / **`.auth-card`**）是整个页面，而不是覆盖在模糊对话记录上的对话框：它后面没有任何可看的东西。字标位于卡片上方，居中且在其外部；卡片包含标题、用户名、密码、一行错误文案（**`role="alert"`**）以及一个在两个字段都填写前禁用的提交按钮，标题下方没有解释性文本。每个字符串都是字典键（**`auth.signIn.*`**），卡片只使用主题令牌，因此全部八种主题和三种语言都能承载它。
- **字标**是矩形 logo（**`src/assets/coddy-logo-wordmark.svg`**，浅色主题下为 **`-light.svg`**），内联到 bundle 中而非拉取，因此屏幕可在一次请求中绘制完成。其 alt 文本已本地化（**`auth.signIn.logoAlt`**）。
- **登录成功后**页面会重新加载：其上的每个列表、流和缓存响应都是由没有会话的浏览器拉取的。
- **页面打开期间会话结束**（过期、轮换、在另一个标签页中退出登录）会让屏幕重新出现：本地源的 **`fetch`** shim 会报告任何 **`/v1/*`** 或 **`/coddy/*`** 调用（**`/coddy/auth/*`** 路由除外）返回 **401**，然后门禁会重新读取状态。
- **退出登录**是导航栏的最后一项（**`data-testid="nav-sign-out"`**），仅在配置了表单且此浏览器已通过它时显示；其工具提示会显示账户名。它发送 **`/coddy/auth/logout`** 并**仅在服务器确实丢弃会话时**重新加载：cookie 是 HttpOnly，因此如果页面在被拒绝后清除自身状态，重新加载会直接把它重新登录，按钮看起来会像坏了而不是被拒绝。
- **Coddy for Browser 的侧边面板会自行登录。** 在扩展框架中带票据（**`?coddy_panel=`**）打开的页面会在绘制屏幕之前将该票据发送到 **`POST /coddy/auth/browser`**，每个页面一次，并获得自己的会话 cookie（[Coddy for Browser](../features/browser.md#侧边面板)）。
- **远程环境不在此处设门禁。** 远程环境使用环境选择器中的 bearer 令牌跨源访问，而此源的 cookie 不会随这些调用一起传递；在远程模式下 **`AuthGate`** 会直接透传。

服务器行为、cookie 和 CSRF 规则：[HTTP API](../reference/http-api.md#web-ui-登录可选)。视觉契约：[DESIGN.md](../../../DESIGN.md)。

## 将 Web 界面安装为应用

Web 界面是一个浏览器可以安装的 Web 应用（issue [#508](https://github.com/coddy-project/coddy-agent/issues/508)）。它携带 Web 应用清单（`/manifest.webmanifest`）、用于主屏幕、程序坞和任务栏的图标，以及一个 service worker（`/sw.js`）。已安装的 Coddy 会在自己的窗口中打开，拥有自己的图标，其标题栏采用 **Settings → Appearance** 中所选主题的颜色。

- **桌面端 Chrome 和 Edge** - 地址栏右端的安装图标，或浏览器菜单中的安装项。
- **Android 上的 Chrome** - 浏览器菜单中的 **Add to Home screen** 或 **Install app**。
- **macOS 上的 Safari** - **File → Add to Dock**。
- **iPhone 和 iPad 上的 Safari** - **Share → Add to Home Screen**。

浏览器只在安全页面上提供安装：`https://`，或运行 `coddy serve` 的机器上的 `http://localhost` 和 `http://127.0.0.1`。通过其网络地址以纯 http 访问的服务器（`http://192.168.1.10:12345`）仍然是普通页面；请将其发布在 TLS 代理之后，如同 [Telegram Mini App](gateway.md#mini-app) 那样。

已安装的应用是同一个页面。它与为其提供服务的服务器通信，保留安装它的浏览器的登录状态（iPhone 和 iPad 上的主屏幕应用保留自己的登录状态，因此你需要在其中再次登录），每个环境、远程和中继在其中都像在标签页中一样工作。service worker 不缓存任何内容：每个请求仍然发往服务器，新构建会在应用下次加载时到达，服务器宕机时应用会显示标签页会显示的错误。

## 通知

![设置，Appearance 选项卡：语言选择器下方的系统通知开关](../../assets/web-ui/notifications-setting-dark-1280.png)

*Settings → Appearance 中语言选择器下方的系统通知开关。*

当 Coddy 处于后台时（另一个标签页、另一个窗口、应用最小化），浏览器可以告诉你智能体需要你。在 **Settings → Appearance → Notifications** 中开启它。该开关会向浏览器请求权限，选择仅保存在此浏览器中（cookie `coddy_ui_notify`），与主题一样。

一个标签页会在以下情况下就其关注的聊天、屏幕上的聊天以及你从该标签页发送过提示词的每个聊天发出通知：

- 一轮结束时：**The agent finished its turn.**，或失败轮次结束时的错误；
- 智能体请求运行工具的权限时：**Permission needed:** 以及该工具；
- 智能体向你提问时：其第一行；
- 聊天的后台子智能体在该聊天轮次结束后请求权限时。

通知会携带聊天的标题，点击它会把标签页带到前台并打开该聊天。当你正在查看 Coddy 时不显示任何内容：在其标签页在聚焦窗口中可见时不显示，在应用的另一个窗口拥有焦点时也不显示。同一浏览器中关注同一聊天的多个标签页只显示一条通知。Telegram 对话、调度器运行或子智能体自身会话的轮次不会被公告，除非标签页显示的是它们的聊天。

限制：

- 通知需要安全页面，与安装一样；在纯 http 下开关为关闭状态并说明原因。
- 由浏览器的权限决定。当浏览器阻止该站点的通知时，开关保持关闭并说明在哪里允许（浏览器的站点设置）。关闭开关会停止通知，并保持权限不变。
- 必须有一个 Coddy 标签页打开，处于后台或最小化。所有标签页都关闭时不会公告任何内容：到达已关闭浏览器的通知（Web Push）目前不在计划中，贡献者可以遵循的设计是 [docs/plans/pwa-push-notifications.md](../../plans/pwa-push-notifications.md)。
- iPhone 和 iPad 上的 Safari 只向添加到主屏幕并从中打开的 Coddy 显示通知。
- 在 [Telegram Mini App](gateway.md#mini-app) 内部没有开关：Telegram 不会为 Mini App 的页面显示任何通知。

代码位于 `external/ui/src/ui/pwa/`（`notifications.ts` 显示通知，`attentionTracker.ts` 决定哪些事件值得通知，`serviceWorker.ts` 注册 worker）和 `external/ui/public/sw.js`；场景是 `features/web_ui_installable_app.feature`。

## 壁纸

![设置，Appearance 选项卡，含四张内置壁纸、选中的紫色图片和暗化滑块](../../assets/web-ui/wallpaper-setting-dark-1280.png)

*紧凑画廊显示四张内置图片、选中的紫色壁纸、上传和清除控件，以及暗化滑块。*

**Settings → Appearance → Wallpaper** 直接显示画廊，无需额外的打开按钮。方形上传图标会打开浏览器的文件选择器，并将所选图片上传到选中的服务器。方形清除图标会清除此浏览器的背景并恢复主题背景；它不会删除服务器上的文件。每个浏览器记住自己选中的背景和暗化程度。

- **上传：** 服务器按 SHA-256 比较文件的字节。如果这些字节已存在，它会返回该文件的名称，即使所选文件的名称不同。否则它会向排序后的画廊添加一张新图片。名称冲突但字节不同时会获得新名称，而不替换现有文件。上传不会改变已选中的背景及其暗化程度。未选中背景时，它会选中上传的图片。私密保存在此浏览器中的图片在页面打开时不会上传；只有通过上传控件选择文件才会将其发送到服务器。
- **选择与删除：** 点击缩略图将该图片用作背景。画廊在桌面外壳上为三列，在较窄宽度下为两列，并在每个缩略图下方显示其文件名。活动边框标记选中项，没有单独的重复预览。缩略图在接近视口时加载，若浏览器不支持 `IntersectionObserver` 则立即加载。每个缩略图右上角都有一个无边框的删除叉号，位于 30px 控件内，点击区域为 40px，无背景或边框。它在图块悬停时或叉号本身获得键盘焦点时出现，在没有悬停的屏幕上保持可见。它会打开 **Delete "filename"?**，含 **Yes** 和 **No**；**Yes** 获得初始焦点。服务器文件仅在确认后删除。删除选中的图片也会清除此浏览器的背景。删除另一张图片不会改变当前背景。
- **显示与暗化：** 图片位于页面后方的固定层上。原生 CSS `center / cover` 保持其宽高比，填满视口并在中心裁剪多余部分。对话在图片上方滚动，不会移动它。它覆盖视口而非页面，因此在手机上很长的对话不会将其拉伸。暗化滑块将主题的画布颜色覆盖在其上，从 0 到 90 %（默认为 40 %，cookie `coddy_ui_wallpaper_dim`）。简化模式保留图片和暗化；玻璃、阴影和界面动画保持禁用。
- **图片：** Coddy 不施加任何文件大小、尺寸、像素或帧数限制。服务器识别图像头并流式传输原始文件，不进行像素解码，也不将整个文件读入内存。浏览器必须能够读取所选图片；读取失败会显示错误。
- **浏览器存储：** 所选图片的字节保存在 IndexedDB 中。浏览器在存储事务提交后应用新的选择。写入失败会显示错误并保留之前的背景和选中指示。后续选择或环境切换会取代较早的请求。没有 IndexedDB 时，选择仅在页面生命周期内有效。
- **内置图片：** 在配置了 home 的首次 HTTP 服务器启动时，在开始监听之前，Coddy 会在画廊之外的 `${CODDY_HOME}/.wallpapers-initialized` 中记录其一次性决定。如果 `${CODDY_HOME}/wallpapers` 不存在，它会创建该目录并将四张嵌入的图片复制到其中：`dunes.png`、`hills.png`、`night.png` 和 `coddy-purple.png`。不会自动选中任何背景。如果画廊已存在，即使为空，Coddy 也不会更改它并记录该决定。一旦常规标记存在，Coddy 不会恢复已删除的图片或已删除的画廊。标记在复制之前创建，因此中断的首次复制可能留下不完整的包，并且不会重试。初始化错误会记录为警告，HTTP 服务器继续启动。图片是嵌入的；不会进行外部下载。
- **服务器与身份验证：** 画廊使用所选服务器上的 `${CODDY_HOME}/wallpapers`（通常是 `~/.coddy/wallpapers`），按名称排序并区分大小写比较。隐藏文件、链接和子目录会被省略。列表请求不会创建缺失的目录，并返回空画廊；上传会在需要时创建它。首次启动初始化与列表请求是分开的。加载中、空和读取失败有各自的消息。列出、缩略图、选择、上传和删除都遵循活动环境，包括远程服务器或通过 swarm 中继的节点，使用普通的 bearer 头或浏览器 cookie 身份验证，图片 URL 中不包含令牌。服务器不联系任何外部图片服务。见[壁纸 API](../reference/http-api.md#壁纸画廊)。

![画廊上方的壁纸删除确认，Yes 已聚焦](../../assets/web-ui/wallpaper-gallery-dark-1280.png)

*删除确认在画廊上方打开。Yes 具有初始焦点；服务器文件仅在确认后删除。*

代码位于 `external/ui/src/ui/theme/backdropPrefs.ts`（浏览器存储和暗化）、`wallpaperGallery.ts`（画廊读取、上传和删除）、`BackdropSettings.tsx`（控件）以及样式表中的壁纸规则；场景是 `features/web_ui_wallpaper.feature` 和 `features/web_ui_wallpaper_gallery.feature`。
## 环境（本地 / 远程服务器）

- **导航栏项：** 环境是导航栏（**`nav/EnvironmentSwitcher.tsx`**、**`data-testid="nav-environment"`**）底部的一项，位于“设置”之下、**退出登录**之上（若存在），否则为最后一项；输入框没有环境标签。其图标是主机：本浏览器自身服务器为笔记本电脑，另一台主机为两个相互指向的 V 形箭头、一个略高于另一个，活动环境的圆点在其角落；工具提示显示 **`Environment: <host>`**，即页面自身服务器所运行机器的短主机名（对 **`devbox.example.org`** 为 **`devbox`**，读取自 **`GET /coddy/info`**；未知前为 **本地**），或远程的名称；宽导航栏在图标旁显示名称，超过导航栏宽度时以省略号截断，完整主机名在其工具提示中。在平板或手机的顶部栏上，它是图标之一，绝不会折叠到 **更多** 后面。它打开一个浮层菜单（**`data-testid="env-menu"`**，模式菜单族；堆叠外壳上为底部面板），包含 **环境** 区（**本地**）和 **远程** 区（已配置的远程 + **`Connect to…`**）。在桌面上，菜单在导航栏旁打开，其底部与该项底部齐平，向上增长并保持在窗口内；其高度等于该项底部上方空间，超出则滚动，因为中继的智能体以及无法使用的远程下方的提示可能超出窗口。包含各种远程的菜单见[远程模式](../operate/remote.md#环境菜单)。
- **选择即连接：** 选择 **本地** 或某个远程会**立即**连接（无确认步骤）；不会针对每次选择弹出令牌提示。在两个远程之间，应用会在新远程上原地重新启动，不重新加载页面（**`EnvScope`**、**`onEnvironmentSwitch`**）；切换到本地或从本地切出时，页面会重新加载。**持有者令牌**来自远程的 `httpserver.remotes` 条目（若其中带有），否则在 **`Connect to…`**（名称 / URL / 令牌）中输入，并按远程在浏览器中记住。若某个远程拒绝其令牌且配置中没有令牌，会提供 **输入令牌**，打开已为其填好的该表单。
- **每个环境离开时的位置：** 切换环境会重新启动应用，因此路由按环境记忆（**`coddy_env_routes`**）：回到本地时，应用打开本地离开时所在的对话；远程则打开其离开时的位置或其主页；中继的映射绝不会随应用进入普通智能体。
- **可达性圆点：** 菜单打开时会探测每个远程（**`env/remoteProbe.ts`**）：**绿色**表示有响应并接受令牌，**红色**表示没有，**琥珀色**表示正在探测；**本地**始终为绿色。红色远程会在其下方一行说明原因：没有响应、拒绝令牌（智能体的 **`httpserver.auth_token`**、中继的 **`swarm.auth_token`**），或浏览器不让页面看到其响应，这正是排除此源的 CORS 设置在浏览器中的表现（该行会指明源和两个键，并且当页面自身位于回环地址（本机的 `coddy serve`）上时，还会指出可在任意端口接纳它的 `allow_loopback` 开关，即 **`isLoopbackOrigin`**（位于 **`env/loopbackOrigin.ts`**））。智能体会被询问 **`GET /v1/models`**；中继不提供 **`/v1`**，通过其公开的 **`GET /swarm/info`** 识别，并在 **`GET /swarm/nodes`** 上检查其令牌，而无 CORS 请求可区分有响应的远程与无响应的远程。
- **中继的智能体：** 一旦中继接受令牌，其智能体会列在其下（从 **`GET /swarm/topology`** 起每一跳深度），每个都是菜单项，可通过中继进入该节点。
- **没有名称的条目**以远程报告的名称显示（**`reportedName`**，位于 **`env/remoteProbe.ts`**）：中继的名称来自其公开的 **`GET /swarm/info`**（除非 **`swarm.name`** 设置了名称，否则为其主机名），智能体的主机名来自 **`GET /coddy/info`**。其地址保留在行右侧；没有报告名称时，地址就是名称，且不会重复显示。
- **用途：** 将 UI 指向远程、已在运行的 **`coddy serve`** 服务器或集群中继，或使用本地服务器。提供的远程来自本地服务器的 **`httpserver.remotes`**（**`[{name, url, token?}]`**），在菜单打开时读取，并在每次配置重载时再次读取；**`Connect to…`** 接受临时的名称/URL/令牌。
- **客户端状态：** 活动环境保存在 **`localStorage`** 键 **`coddy_env`** 中；各远程的令牌在 **`coddy_env_tokens`** 中。绝不持久化到服务器配置；无认证的远程留空。工作区 **文件夹最近记录** 按环境命名空间隔离（**`envStorageSuffix()`**），因此每个远程记住自己的最后路径；**模型**和默认值在重载后来自远程的 **`GET /v1/models`**。
- **机制：** 全局 **`fetch`** 垫片（**`external/ui/src/ui/env/remoteEnv.ts`**，安装在 **`main.tsx`** 中）会将同源 API 请求（**`/v1/*`**、**`/coddy/*`**、**`/openapi*`**）重写到所选远程的基础 URL，并添加 **`Authorization: Bearer <token>`**。本地模式是透明直通。选择某个条目会持久化选择并重新启动应用（两个远程之间原地重启，切换到本地或从本地切出时重新加载），以便所有状态都从所选后端重新获取；SPA 外壳始终从本地源加载，因此即使远程宕机，你也始终可以从标签切回 **本地**。
- **浏览器自行加载的图片：** **`<img>`** 从不经过垫片，因此服务器以 API 路径命名的图片，即用户消息所带图片的缩略图（**`preview_url`**）及其原图（**`url`**），会向页面自身源请求，且不带令牌：通过集群中继时，该源不提供 **`/coddy/`** 路由，图片就会损坏。在远程环境中，这些图片改为通过它获取（**`useApiImageSrc`** 位于 **`external/ui/src/ui/env/apiImage.ts`**，与垫片使用相同的 **`remoteApiRequest`** 映射：基础 URL 在前，令牌放在 **`Authorization`** 头中，绝不放 URL），并从对象 URL 显示，图片离开屏幕时释放该 URL；在字节到达之前，图片没有 **`src`**，而不是显示损坏图标。在本地源，以及 **`blob:`** / **`data:`** / 其他地址，URL 按原样使用。灯箱打开时也以同样方式读取原图。
- **CORS：** 远程必须通过 **`httpserver.cors`** 允许 UI 的源：在 **`allowed_origins`** 中精确允许，或使用 **`allow_loopback: true`**（当 UI 由本机自己的 `coddy serve` 在回环地址上提供时，无论端口）（见 [http-api.md](../reference/http-api.md)）。SSE 重新附加（**`GET /coddy/sessions/{id}/composer-stream`**）通过获取方式加载（而非 `EventSource`），因此适用持有者头；该路由也接受 **`?access_token=`**，供外部 `EventSource` 客户端使用。
- **故障呈现（问题 #60）：** 对不可达 / 被拒绝 / TLS 或 DNS 失败 / 被 CORS 阻止的远程发起的 `fetch()` 会以 `TypeError` 拒绝（无 `Response`）；发送流程最后的 `catch` 现在会将这种情况与用户自己的 `AbortError` 区分开，并发出错误 `system_notice`（**`remoteSendErrorMessage`**），可读的 `401/403` 会得到针对认证的消息（**`remoteHttpErrorMessage`**），而不是裸状态。纯辅助函数位于 **`external/ui/src/ui/env/remoteErrors.ts`**。
- **活动环境健康状态（问题 #60、#401）：** 共享监视器（**`external/ui/src/ui/env/activeHealth.ts`**，在 **`main.tsx`** 中启动）在加载时、每 30 秒以及窗口获得焦点时探测*所选*环境，使用与菜单相同的探测（**`probeRemote`**）：智能体的 **`GET /v1/models`**，中继的 **`GET /swarm/info`** 和 **`GET /swarm/nodes`**。输入框标签圆点由该健康状态驱动（绿色在线 / 红色离线 / 琥珀色检查中，**`.env-status`**），并且当活动远程离线、拒绝其令牌或以浏览器不让页面看到的方式响应时，**`EnvHealthBanner`** 会显示带 **切换到本地** 操作的持续警报，因此应用绝不会在死后端上静默渲染为空。警报会说明是三种情况中的哪一种，并指出修复它的设置：智能体或中继的令牌键，或 CORS 键及此页面的源；对于回环地址上的页面，还会指出旁边的 `allow_loopback` 键（通过中继到达的节点由中继的 CORS 应答）。在堆叠外壳上，导航栏是固定在页面顶部的栏，警报挂在其下；其高度（**`--coddy-env-banner-h`**）会将页面下移，因此栏绝不会遮住警报。再次询问时，按间隔、在获得焦点时，或再次选择同一远程时，远程会保持其判定，直到有应答改变它，因此警报不会在探测期间闪烁消失；只保留最新一次询问的应答，配置中轮换的令牌会立即用于询问。

## 布局

![宽导航栏在 1920 px 下带标签](../../assets/nav-rail-wide-1920.png)

*宽导航栏在 1920 px 下带标签；其底部的环境项显示页面服务器所运行的机器*

![同一外壳在 390 px 下：导航栏变成顶部栏](../../assets/nav-topbar-mobile-390.png)

*同一外壳在 390 px 下：导航栏变成顶部栏，起始屏幕适配宽度*

![移动外壳上的聊天](../../assets/screenshot-mobile-chat.png)

*手机上的聊天：输入框上方的面板以药丸形式显示 git 计数，选择器标签一行，发送按钮就位*

桌面布局

- **品牌**仅为**排版**（**Coddy** 和 **agent**）。品牌文字前**没有**圆形徽标或图标，无论旧参考图是否包含圆圈。
- 桌面导航是**垂直面板**，在**右**边缘有圆角（不是全高居中胶囊）。在 **`min-width: 1920px`** 上，宽导航栏标题包含一个带**水平线**的图标，**仅**用于**折叠**为窄导航栏，而不是作为全局导航抽屉。
- 左侧导航栏从品牌下方的 **历史** 打开**聊天历史**；点击品牌进入**起始屏幕**（**新对话**），**新建对话**（**`data-testid="nav-new-chat"`**）也是如此：窄导航栏中品牌下方的加号、顶部栏中其后的加号，以及宽导航栏中 **历史** 上方的带标签行。
- **品牌**、**历史**、**调度器**（若有链接）、**设置** 以及 **历史** 列表中的每一行都使用真实片段 **`href`** 值（**`#/`**、**`#/history`**、**`#/scheduler`**、**`#/scheduler/new`**（新任务编辑器）、**`#/settings`**、**`#/s/<sessionId>`**），因此**中键点击**或 **Ctrl/Cmd 点击**会在同源的新浏览器标签页中打开，而另一个标签页可以继续流式传输。
- 会话列表在**所有**断点和导航栏宽度下**始终**是带背景遮罩的**抽屉浮层**（**没有**会缩小聊天区域的、位于导航栏旁的嵌入式列）。面板标题和相关界面文案使用 **历史**。在堆叠外壳上，该面板与调度器和设置的抽屉保持相同带区：距两侧 10 px，位于顶部条下方，距底部或屏幕自身底部内边距 14 px（若有）。其头部关闭控件和 Escape 是返回对话的方式。
- 可选导航栏**窄版与宽版**（图标加标签）仅在 **`min-width: 1920px`** 时启用，保存在 **`coddy_nav_rail`** cookie 中（默认 **`narrow`**）
- 主聊天区域显示流式助手输出
- 右栏不在当前里程碑范围内

宽屏

- **`min-width: 1920px`** 可启用导航栏加宽控件和由 cookie 支持的布局（**见 DESIGN.md**）。**历史** 仍是紧邻测量出的导航列的**浮动抽屉**（**`--rail-shell-track-width`**）；不要用静态像素常量为宽导航栏固定 **`left`**。

移动布局

- 在移动端，左侧导航栏变成顶部栏以保留水平空间；顶部栏在视口顶部为 **`position: fixed`**（**`shell-main`** 以 **`--coddy-mobile-top-inset`** 填充），而 **`body`** 滚动聊天。
- 在手机上，品牌占两行，**Coddy** 在 **agent** 上方，如同桌面上的窄导航栏。
- 在 **`max-width: 1199px`** 上（手机和平板皆然），**设置** 抽屉对其堆叠的每个带区保持一个内边距 **14px**：**设置** 头部、其下的状态带（仅在加载失败或保存被拒绝时显示）、分区磁贴、打开的分区以及重新加载 / 保存页脚，都从同一边缘开始并结束。通过测量 `getBoundingClientRect().left`（`.settings-tile` 与 `.settings-body`）并对照 `.settings.drawer` 来核对，二者必须一致。

手机布局（**`max-width: 599px`**）

外壳按窗口宽度分为四档：**手机** 最宽至 **599px**，**平板** 从 **600** 到 **1199px**，**桌面** 从 **1200px** 起，**宽屏** 从 **1920px** 起（该表为 **`DESIGN.md`** 中的 **布局网格**）。平板和桌面已能容纳所有控件；手机不能，因此最窄档有自己的一块 **`styles/shell/_phone.scss`**，其中内容不适用于 **599px** 以上。**599px** 是移动平台用于其紧凑窗口类的边界：所有竖屏手机都在其下（Galaxy S21 为 360，iPhone 13 mini 为 375，Honor X9d 为 393），最小的竖屏平板则在其上。

![375 px 手机上的聊天：以 MCP 工具命名的工具行在对话记录内换行，长链接会断开](../../assets/phone-transcript-mcp-row-dark-375.png)

*375 px 下的聊天：MCP 工具行让其标签换行，并将目标和耗时移到其下，答案中的长链接会断开而不是撑宽页面。*

- **顶部栏。** 品牌占两行，**Coddy** 在 **agent** 上方，其后是新对话的加号；图标为 **40px**，间距 **4px**，并靠右边缘。品牌旁放不下的内容会折叠到 **更多** 按钮（三个点）后面：**历史**、**集群** 和环境始终留在栏中，其他项随着空间增加按 **设置**、**调度器**、**文档**、**退出登录** 的顺序回来，菜单将折叠项列为 **文档**、**调度器**、**设置**，以及分隔符下的 **退出登录**。选择、按 **Escape** 或点击外部会关闭它；折叠面板打开时按钮高亮。40px 图标下，从约 290px 起可容纳五项，因此在大多数手机上菜单只会出现在中继或极窄屏幕上。分界由 **`nav/navOverflow.ts`** 定义，仅由堆叠外壳上的 **`NavRail`** 测量。
- **输入框、面板。** 输入框卡片没有标签行：工作区在其上方的面板上，一行，文件夹和分支以省略号收缩。在手机和任何触摸屏上，面板的选择项和 worktree 复选框高 **36px**，git 计数保留其带边框的纤细外观，并在其上有一个不可见的 **40px** 点击区域，方便手指点击；改进提示词的魔杖位于字段右上角。
- **输入框、选择器标签。** 多模态模型的附加按钮，然后模式、模型、推理和权限（所有宽度下均为此顺序）形成一条条带，在右边缘淡出下横向滚动；上下文环和 **发送** 从不收缩，因此任何标签都不会被画在其下。长模型名以省略号结尾。
- **起始屏幕。** 主视觉是一个网格轨道，不能超出其容器（**`grid-template-columns: minmax(0, 1fr)`**），因此页面绝不会横向滚动，标题保持居中。
- **对话记录。** 在堆叠外壳上，对话记录就是页面，因此比它宽的行曾使整个页面横向滚动：在 360px 手机上，以 MCP 工具命名的工具行曾使其滚动 400px，答案中的长链接或标识符也是如此。现在在任何宽度下，对话记录中都没有比它更宽的内容：带有目标的工具行若其标签在本行没有空间，会将目标、失败标记和耗时一起移到其下；比行更长的标签会在其中换行；正文、链接和行内代码在必须时断行，而表格和代码块保持其宽度并在自己的框中滚动。工具行在被点击前保持折叠。
- **设置。** 分区磁贴保持两列，分区名称换行成两行而不是以省略号结尾：手机没有悬停，因此被截断的名称在其工具提示后面无法触及。
- **文本字段** 在手机上以及任何仅触摸设备上至少为 **16px**（**`(any-hover: none) and (any-pointer: coarse)`**）：iOS Safari 会在聚焦时将页面缩放进入更小的字段，且不会缩回。输入框及其高亮镜像会一起变化，因为镜像必须保持 textarea 的度量。
- viewport meta 带有 **`interactive-widget=resizes-content`**，因此当屏幕键盘打开时 Android 浏览器会缩小布局，停靠的输入框会留在其上方。忽略它的浏览器（iOS Safari）只会缩小可视视口；此时停靠块会按键盘覆盖的量抬升（**`--coddy-keyboard-inset`**，测量自 **`visualViewport`**）。
- **不会自行弹出键盘。** 打开起始屏幕或聊天，或关闭历史，不会在手机或平板（仅触摸设备）上聚焦输入框，因为聚焦的字段会打开屏幕键盘；当你点击字段时它才会出现。桌面窗口无论宽窄都会保持焦点。

在 **360**、**375**、**393** 和 **430px** 下，在起始屏幕和聊天中、菜单关闭和历史打开时进行实时检查：**`document.documentElement.scrollWidth`** 等于 **`clientWidth`**，**发送** 按钮的矩形不与任何 **`.composer-tab`** 相交，品牌的矩形不与顶部栏的任何图标相交。在 **600**、**834** 和 **1280px** 下，布局分别为平板和桌面。对话记录在网格的每个宽度下由 **`npm run check:overflow`** 测量（见下文 **在网格的每个宽度下检查对话记录**）。CSS 契约由 **`phoneLayoutCss.test.ts`**、**`transcriptWrapCss.test.ts`** 和 **`layoutGridCss.test.ts`** 固定，后者会在样式表询问但网格未列出的宽度上失败；场景为 **`features/web_ui_phone.feature`**。

Telegram Mini App（问题 [#320](https://github.com/coddy-project/coddy-agent/issues/320)）

![Telegram Mini App 中的对话，半开状态，模式表面板覆盖在停靠的输入框上](../../assets/web-ui/telegram-mini-app-mode-dark-390.png)

*半开 Mini App 中的对话：模式面板和输入框留在 Telegram 显示的部分内。*

![从 Telegram 浅色主题打开的 Mini App 起始屏幕：Telegram 的头部和 Web 界面均为浅色](../../assets/web-ui/telegram-mini-app-start-light-390.png)

*从 Telegram 浅色主题打开且未在 Coddy 中选择主题：Web 界面跟随它，Telegram 的头部也是如此。*

Telegram 机器人可以通过其菜单按钮、`web_app` 按钮或链接，将 Web 界面作为其 Mini App 打开。当 `gateways.telegram.mini_app.url` 指定 Web 界面的公开 https 地址时，Telegram 网关会设置这一点（网关页面上的 [Mini App](gateway.md#mini-app)）；你也可以在 @BotFather 中手动将菜单按钮指向 Web 界面。在 Telegram 内，Web 界面会按下述方式变化，而普通浏览器看不到其中任何变化。

- **不加载来自 telegram.org 的脚本。** Web 界面自己处理 Telegram 的 [Web 事件](https://core.telegram.org/api/web-events)（`ui/telegram/bridge.ts`）：通过移动端和桌面应用注入的 `TelegramWebviewProxy`，或使用 Telegram Web 的父框架，其消息发送到父框架的源，并且只从该源接收父框架的消息。`index.html` 不从其他源加载任何内容，因此阻止 telegram.org 的网络不会拖慢任何人的页面。
- **启动参数。** Telegram 将它们放入地址中（`#tgWebAppData=...&tgWebAppVersion=...`）。Web 界面会在其他任何内容运行之前读取它们（`ui/telegram/launch.ts`），将它们保存在标签页的 `sessionStorage` 中，因此在 Telegram 内重新加载仍保持为 Mini App，并从地址中移除它们，以便路由器看到自己的路由，且从页面复制的链接不携带签名的启动数据。`?session=<id>`，即机器人 `/app` 给出的链接，在 Mini App 中和普通浏览器中都会打开该对话。启动数据从不代表用户：你仍像在浏览器中一样登录。
- **窗口。** Web 界面将 Mini App 扩展到全高，关闭会在滚动对话记录时折叠它的垂直滑动（Telegram 的头部仍会折叠它），并将 Telegram 的头部、背景和底部栏设为主题的颜色，且在主题变化时再次设置。在你于 **设置 → 外观** 中选择主题之前，它会跟随 Telegram 的浅色或深色主题。
- **返回。** Telegram 的返回按钮作用如 **Escape**：关闭最上方的菜单、面板、对话框或屏幕；在没有任何打开内容的对话中，它回到起始屏幕。它绝不会跳过智能体向你提出的问题。在没有任何打开内容的起始屏幕上，Telegram 显示 **关闭**，Android 的返回键会关闭 Mini App。
- **半开窗口。** Telegram 可能只显示 Mini App 的顶部。停靠的输入框、菜单和选择器（底部面板，与手机上各处相同）、确认对话框和图片查看器会升入 Telegram 显示的部分，抬升量为被隐藏部分或屏幕键盘中较大者，对话记录的最新消息保持在输入框上方。历史、设置和其他抽屉保持原位，其头部和关闭按钮在视野内。在全屏下，顶部栏会避开状态栏和 Telegram 自己的按钮。
- **Telegram Web。** web.telegram.org 会在另一个站点的框架中运行 Mini App，而浏览器不会保留登录 cookie。在服务器接受而浏览器丢弃登录之后，登录屏幕会说明这一点，并链接到单独标签页中的 Coddy。移动端和桌面应用为 Mini App 提供自己的窗口，其中登录与浏览器中一样有效。

该模块为 `external/ui/src/ui/telegram/`，带有自己的样式表 `telegram.css`；场景为 `features/web_ui_telegram_mini_app.feature`。要在没有 Telegram 和手机的情况下查看它，请从离线测试台打开：[针对假 Bot API 进行调试](gateway.md#针对假-bot-api-调试) 会手动展示测试台，而 `npm run check:telegram` 会端到端驱动它（见下文 *检查 Telegram Mini App*）。

标题链接

- GitHub 链接到 `https://github.com/coddy-project/coddy-agent`（**新标签页**，`rel=noopener`）。
- API 文档链接到 `/docs/`（**新标签页**，`rel=noopener`）。
- 页面正在通信的服务器版本，位于右侧（**`GET /coddy/info`**）：发布版显示 **`v1.2.36`** 并链接到 GitHub 上的发布说明，发布之间的构建按原样显示。在远程环境中或通过中继到达的节点内，它是该服务器的版本，而不是页面的版本。
- 三者位于起始屏幕的页脚（**GitHub** | **API 文档** | **v1.2.36**，**`chat/HeroFooter.tsx`**），聊天打开后即消失；导航栏不携带其中任何一项。

窄导航栏工具提示（桌面）

- 当导航栏**没有**宽标签时，**悬停工具提示**会强化图标含义（例如加号上的 **新建对话**、历史图标上的 **历史**）。**带宽标签的导航栏**会隐藏这些工具提示；标签本身就是可供性。
- 打开 **历史** 后，如果指针仍悬停在导航栏上，历史触发器的工具提示**不得**保持可见（见 **DESIGN.md**）。

用 Escape 关闭屏幕

- 导航栏中除 **退出登录** 外的每一项都会在聊天上方打开一个屏幕：**历史**、**调度器**、**集群**、**文档** 和 **设置**。**Escape** 会像其 **×** 一样关闭屏幕上的那一个，地址会回到其下方的内容（聊天、起始屏幕，或阅读器打开时的来源），因此重新加载不会让该屏幕回来。聊天上方的集群屏幕会像其背景遮罩一样关闭；在中继上，集群映射是主页并保持不变。
- 它一次撤销一步。离按键更近的内容先接收：打开的菜单（历史行的 ⋮、筛选菜单、手机上的 **更多**、输入框的 **模式**、**模型**、**推理** 和 **权限** 菜单、环境、文件夹和分支菜单）、选择器或输入框的上下文浮层、文件夹浏览器、正在重命名的聊天标题、设置字段的 (i) 提示、设置组合框的打开列表、技能安装搜索下的市场结果（Escape 清除搜索）、有文本时的文档搜索（第一次 Escape 清除它，下一次关闭阅读器）、结果显示或字段中有查询时的设置搜索（第一次 Escape 清除查询，下一次是抽屉的；[设置：搜索](#设置搜索)）、确认对话框、图片查看器。有自己步骤的屏幕会先执行该步骤再关闭，即其头部箭头所执行的步骤：调度器中打开的任务或其运行记录会回到任务列表；设置中打开的行（提供商、模型）会回到其列表；手机或平板上的设置分区会回到磁贴。
- 关闭了屏幕的 Escape 不会继续深入：其下方聊天的问题卡片会保留其问题（没有屏幕打开时，Escape 像以前一样跳过它们，而关闭了输入框某个选择器的 Escape 不会）。按住键、带修饰键的 Escape 以及结束输入法组合输入的 Escape 不会关闭任何内容。
- 该规则为每个屏幕只写一次（**`nav/railEscape.ts`**，由 **`App.tsx`** 使用导航栏每个屏幕的表调用，并由导航栏项进行类型检查），因此以后添加到导航栏的屏幕无需自己的处理器即可在 Escape 时关闭，且若表中没有条目就无法通过类型检查。菜单、浮层或对话框通过 **`useEscapeCloses`**（**`components/useEscapeCloses.ts`**）在 Escape 时关闭，它会在其下方的屏幕听到按键之前先占用该键。契约：**`DESIGN.md`**（**Escape 与导航栏的屏幕**）；测试 **`nav/railEscape.test.tsx`**、**`App.railEscape.test.tsx`**；场景 **`features/web_ui_rail_escape.feature`**。
## 会话

- 会话 ID 仅在从新聊天发送第一条消息后由客户端生成。
- 会话 ID 持久化在 URL 片段中。
  - 推荐格式 `#/s/<sessionId>`
- 未发送的输入框文本可作为仅客户端的草稿会话保留。
  - 草稿会话使用 `#/draft/<draftId>`，并存储在 `localStorage` 下的 `coddy_draft_sessions_v1` 中。
  - 历史记录行显示 `Draft:` 标题前缀。
- 会话 ID 在聊天传输中通过 `X-Coddy-Session-ID` 请求头发送。
- 会话 ID 校验匹配 `internal/session/ValidateFolderSessionID`。
- 会话持久化文件位于会话目录下，并在删除会话时一并删除。
  - `tool_calls/` 工具调用历史
  - `stats.json` token 使用总量

### 并行会话与生成取消

- 多个会话可以**同时流式传输**，每个都有自己的 **`POST /v1/responses`** 和 **`X-Coddy-Session-ID`**。应用为每个会话保留**影子**对话记录，这样快速切换 hash 不会误路由 SSE 更新；参见 **`pickStreamMutationBase`**，位于 **`external/ui/src/ui/chat/streamMutationBase.ts`** 中。
- 影子对话记录是**有界的**：非固定会话的 LRU 为 **3** 个（**`ShadowTranscriptCache`** 在 **`external/ui/src/ui/chat/sessionTranscriptCache.ts`** 中）。正在查看的会话以及任何有实时流的会话永远不会被逐出；被逐出的会话会在下次访问时重新获取，与现在相比没有额外请求。消息行和 **`Markdown`** 已记忆化（**`React.memo`**、**`useStableHandler`**），因此 token 流式传输时未更改的行会跳过重新渲染（**`MessageList`** 映射对话记录窗口的切片，见**长会话**；计划、权限和问题行不记忆化）。对话记录行不携带 **`content-visibility`**：让屏幕外的行跳过布局会按回退猜测给打开屏幕之上的每一行定尺寸，因此在长会话中滚动端口短了约六分之一，并且向上滚动时随着真实高度到达位置会被向下推。参见 **`DESIGN.md`**（**多会话流式传输与停止**）。
- **服务器活动与本地流读取器是分开的。** UI 在会话打开和重连时读取 **`GET /coddy/sessions/{id}/activity`**，无论筛选或分页的**历史记录**列表是否包含该会话。当此标签页自己的 POST 正在等待准入时，**`chat/useSessionTurnActivity`** 会跳过活动与队列水合。运行中的轮次即使没有本地读取器也保持**停止**和排队可用；丢失或关闭读取器并不能证明轮次已结束。与当前标签页待处理或已准入的 POST 重叠的 **`turn_ended`** 需要新的 REST 活动确认，UI 才会宣布空闲。
- **停止生成**总是调用 **`POST /coddy/sessions/{id}/cancel`**，参数为 **`{"continue_queue":true}`**，并在发送后立即中止此标签页对该轮次的流式 **`fetch`**，而不是等到回答之后。浏览器在其所有标签页之间为每个主机保留六条 HTTP/1.1 连接；打开几个 Coddy 标签页时，事件流和轮次流可能占满每一条连接，而取消请求会等待此读取器持有的连接。轮次在没有读取器的情况下仍在服务器上运行。失败的请求会显示错误，保持运行状态并使 **停止** 可用于重试，标签页会通过输入框中继重新加入轮次。启用 **`queueContinued: true`** 时，UI 会通过输入框中继重新附着到同一轮次生成，而无需等待另一个 **`turn_started`**。它不会在本地排空或重新提交排队消息。没有该标志时，成功即确认完全取消；UI 会等待服务器的轮次完成或报告空闲的新活动快照，然后才将会话视为已停止。现有的生成保护仍然保留。当 token 已经到达时，服务器会为该轮次持久化**部分**助手 **`content`**。**`GET /coddy/sessions/{id}/messages`** 可能短暂返回较旧的快照；当响应只是前缀时，UI 会与本地影子或可见行**合并**（**`mergeTranscriptPreferLocalSuffix`**、**`keepLocalTranscriptIfServerEmpty`**，位于 **`external/ui/src/ui/chat/transcriptServerSnapshot.ts`** 中）。仅在失败的加载针对**当前查看的**会话时，获取失败才会清除对话记录，这样停止不会清空聊天。

- **工具调用保留其结果。** 当停止在排队调用开始前中断轮次时，其保存的结果是 `cancelled` / **未执行**。已在运行的调用保留其实际结果：取消不会回滚副作用。对于缺少结果的旧对话记录，发送给提供商的请求会得到一个固定占位符，警告**未知结果**；旧对话记录及其 UI 行保持不变。在新用户消息之后到达的权限应答是过期的：它会清除该门控，在可能时将调用记录为已取消，并且不会运行工具或重写较新的消息。格式错误或有歧义的调用 ID 和结果会产生本地诊断，而不是提供商 `400`。使用 **`/compact`** 或启动新会话来恢复，不要盲目重放结果未知且有副作用的调用。

![取消失败时轮次仍在运行，仍可重试](../../assets/message-queue/stop-queue-cancel-failed-dark-1280.png)

*取消失败；轮次仍在运行，停止仍可用于再次尝试。*

会话标题

- UI 在聊天头部显示会话标题。
- 标题缺失时，UI 显示 `New chat`。
- 标题可就地编辑。失焦时 UI 通过 `PATCH /coddy/sessions/{id}` 保存。
- 当新聊天正在命名时，头部在标题将出现的位置显示闪烁条，而不是第一条消息（[会话列表](#会话列表)）。

### 每会话模型

![设置中模型行的流式开关](../../assets/settings-model-stream-toggle-dark-1280.png)

*设置中模型行的流式开关*

- **新聊天**在 cookie `coddy_llm_model` 命名了已配置后端时从该 cookie 默认**模型**，否则使用按字母排序的第一行 YAML。已发送消息中键入的 `/model <id>` 在此界面上也是一次选择，并写入同一个 cookie（轮次作用域的 `--once` / `--count` 形式不会）。
- **打开会话**会从 **`settings`**（属于 **`GET /coddy/sessions/{id}/messages`**）的快照恢复**模型**（会话自己的模型；顶层 `model` 命名运行中轮次所持有的模型），绝不从 cookie 恢复：cookie 和起始页的选择只是新聊天的默认值。
- 更改**模型**会写入 cookie（作为下一次**新聊天**的默认值）并在活动会话上 `PATCH` `selectedModelId`。ReAct 轮次仍发送 `metadata.model`，在 `POST /v1/responses` 上。
- **多模型 / 长名称**，后端 ID 是 `vendor/model`。配置了多个供应商时，菜单会把行分组在大写供应商标题下，每行只显示模型名称（完整 ID 保留在行工具提示中）。供应商保持配置顺序，每个供应商的模型按字母顺序读取，版本按数字比较（`gpt-5.6-luna`、`gpt-5.10`、`gpt-6-astra`），因此后来添加到 `models[]` 的模型会落在名称所属的位置，而不是其分组底部；筛选器中的 **Enter** 会按该顺序选择第一行。在桌面端，列表以约 5 行上限滚动。当后端**超过 5 个**时，顶部会出现一个**筛选输入框**（自动聚焦），匹配供应商、模型名称或完整 ID（不区分大小写）；**Enter** 选择第一个匹配项，**Escape** 关闭，空结果会显示 "没有模型匹配 …" 提示。输入框的每个菜单（**模式**、**模型**、**推理**、**权限**）都在 **Escape** 时关闭，无论是否有筛选器，也无论焦点在哪里。筛选/分组/阈值逻辑在 `chat/llmModelMenu.ts` 中（在 `llmModelMenu.test.ts` 中有单元测试；菜单接线由 `ComposerModelMenu.test.tsx` 覆盖）。
- **移动端底部面板**，在窄屏/移动端外壳中（`max-width: 1199px` 外壳栈断点），**模式** / **模型** / **推理**菜单会作为**全宽底部面板**在变暗遮罩上打开，与斜杠（`/`）和 `@` 选择器相同的模式，而不是拥挤的锚定下拉菜单。筛选和分组在面板内仍然生效。桌面端保留锚定下拉菜单。

### 每会话推理级别

![输入框中的推理级别下拉菜单，级别从提供商获取](../../assets/reasoning-levels-dark-1280.png)

*输入框中的推理级别下拉菜单，级别从提供商获取*

- 只有当活动模型暴露 **`reasoning_levels`**（来自 **`GET /v1/models`**）时（例如 gpt-5 / o-series / Claude thinking 等推理模型），输入框中才会在**模型**旁出现**推理**选择器。级别从 `models[].reasoning_levels` 派生（未设置时从模型 ID 自动检测），并通过 **`ModelInfo.reasoningLevels`** → **`llmReasoningLevels`**（位于 **`App.tsx`** 中）传播到 **`Composer`**。
- **新聊天**会从 cookie `coddy_llm_reasoning` 默认级别，然后是模型的 `reasoning_default`，再是 `medium`（或第一个提供的级别）。**打开会话**会从快照的 `settings.reasoning` 恢复它，绝不从 cookie 恢复；一个关闭思考运行的会话，当其逻辑模型启用了 `allow_reasoning_off: true` 时显示**关闭**，而一个自身没有级别、模型也没有 `reasoning_default` 的会话显示 `medium` 或第一个级别，在进入时以及之后每次快照后都如此，直到选择级别前不会发送它。菜单会为快照指定它的会话提供**关闭**。切换到不提供当前级别的模型时，会将其钳制到有效级别（参见 **`pickReasoningLevel`**，位于 **`chat/reasoningSelection.ts`** 中）。
- 更改级别会写入 cookie，并在活动会话上 `PATCH` `selectedReasoning`；ReAct 轮次还会发送 `metadata.reasoning`，在 `POST /v1/responses` 上，以便全新会话在第一轮就应用它。

### 会话设置：权限标签、轮次覆盖、实时镜像

![从对话框切换到绕过后的输入框：选择器以红色“绕过”标签结尾，后面是 "neuraldeep/qwen3.6-35b-a3b, 2 turns left"](../../assets/session-settings/session-settings-composer-bypass-dark-1280.png)

*会话从权限提示切换到绕过、且模型已更改两轮后的输入框。*

- 输入框镜像会话的设置快照（[会话设置](../features/session-settings.md)）：在 **`settings`** 于 **`GET /coddy/sessions/{id}/messages`** 上以及 **`PATCH /coddy/sessions/{id}`** 应答上，并在 **`event: session_settings`** 于轮次流和 **`GET /coddy/events`** 上。`chat/sessionSettings.ts` 解析它；`App.tsx` 为每个会话保留最高的 `version`（`isNewerSettings`），因此沿两条连接到达的同一更改，或在较晚更改事件之后到达的应答，只会应用一次且绝不会回滚。
- **模式**、**模型**和**推理**会跟随任何地方所做的更改，包括键入的命令、权限对话框、模型的 `switch_model`、控制台或同一会话上的编辑器。每个 `POST /v1/responses` 都携带 `metadata.settingsVersion`，即该标签页最后应用的版本；如果此后发布了更新的快照，服务器会忽略该标签页的 `model` / `reasoning` / 模式，因此过期的标签页无法撤销它尚未看到的更改。
- **模式**、**模型**和**推理**属于屏幕上的会话。打开会话时会显示其快照的模式（`settings.mode`）；新聊天（**新聊天**、返回到 `#/` 的返回按钮、历史记录中的草稿、服务器回答 **404** 的会话 ID）会以 **Agent** 开始，使用新聊天的默认模型和级别，绝不会使用刚离开的聊天的模式，而在起始屏幕上选择的模式会以该模式创建聊天。当打开的会话正在加载时，选择器仍显示离开的会话，提示词会携带它们的模式、`metadata.model` 和 `metadata.reasoning`，却没有已打开会话的 `settingsVersion`，因此服务器会保留它们：**发送**和**Enter**会等待，直到应用该会话自己的快照，草稿保留在输入框中。读取失败会在 1、3、9 秒后重试，之后只要聊天保持打开就每 30 秒重试一次；只有服务器的 **404** 会把该地址作为新聊天打开：历史记录只保存一页，因此其中缺失的会话很可能仍然存在。发给运行中轮次的消息仍会加入队列，而队列不携带它们中的任何一个。契约：`App.settingsPerSession.test.tsx`、`Composer.sessionLoading.test.tsx`、`features/web_ui_session_settings.feature`。
- **权限标签**（`data-testid="composer-permission"`）是该行的最后一个选择器，位于**模型**和**推理**之后：**先询问**、**接受编辑**或**绕过**，最后一个为红色并带发光，其工具提示命名新会话开始时的模式（*新会话开始于：绕过*）。其菜单调用 `PATCH` `permissionMode`，会话的新模式也会成为新会话开始时的模式。在新聊天中（起始屏幕或草稿，会话尚不存在），标签会命名新会话开始时的模式，即在任何界面上最后选择的模式：**`permissionMode`**（属于 **`GET /coddy/info`**），通过活动环境（本地服务器、远程、位于中继挂载后的节点）读取，并在每次 `config_reloaded` 和每次 `GET /coddy/events` 重连时再次读取，并由每个更新的 `event: session_settings` 的 `configuredPermissionMode` 移动，无论它涉及哪个会话。那里的选择会被保留，并作为 `/permissions <mode>` 行在第一条消息之前发送，除非它就是该模式；当该模式未知时（尚无应答、服务器不说明），选择总是会发送，因此当新会话开始于 `bypass` 时，显式的**先询问**绝不会被丢弃；没有选择时，会发送标签显示的模式，因此第一轮绝不会在页面无法命名的模式下运行。服务器从未接受的第一条消息会保留该选择，以便下次尝试进入同一会话。契约：`App.permissionDefault.test.tsx`。
- **工具集标签**（`data-testid="composer-toolset"`）在配置有工具集预设时跟在权限标签之后，否则不存在：它命名会话运行所用的工具集（无则为**所有工具**），其菜单列出预设和**所有工具**，选择会作为 `PATCH` `selectedToolset` 发出（[工具集](../getting-started/configuration.md#工具集)）。对于打开的会话，它读取快照的 `toolset` 和 `toolsetChoices`；在新聊天中，它从 **`toolsets`**（属于 **`GET /coddy/info`**）读取预设，并命名所选模型的工具集（**`toolset`**，在其 **`GET /v1/models`** 行上），直到选择另一个，那里的选择会作为 `metadata.toolset` 随第一条消息发出。两者都会在 `config_reloaded` 时再次读取，因此在设置中添加的预设无需重新加载即可带来该标签。契约：`App.toolset.test.tsx`、`Composer.test.tsx`。

![小型本地模型上的起始屏幕：权限标签之后，工具集标签显示 coding，其菜单已打开](../../assets/session-settings/composer-toolset-dark-1280.png)

*在命名 coding 工具集的模型上新建聊天：该标签在权限标签之后，菜单列出预设和所有工具。*

- **覆盖行**（`data-testid="composer-overrides"`）列出下一轮次更改的内容（`stub/qwen3.8-demo, 2 turns left`、`plan this turn`），完整列表在其工具提示中；在窄屏外壳中它会以省略号收缩，而不是把发送按钮挤出栏外。
- 仅包含设置命令的提示词不会运行轮次：流以 `coddy_meta.settings_only` 结束，标签页会丢弃其乐观的用户和助手行，并重新读取对话记录。更改会显示在选择器和覆盖行上，对话记录不会为此增加行。
- **SYSTEM** 通知行仅由智能体自身所做的更改留下：其 `switch_model` 调用、技能的 frontmatter（`Model: stub/qwen3.8-27b for this session`）。用户选择的内容，包括选择器、命令、权限对话框、控制台或 `coddy -p` 启动会话时使用的模型和模式，已经显示在选择器上；由较早版本保存的会话为此类更改保留的行不会显示。

### 设置：逻辑模型的推理级别

**设置 -> 逻辑模型 -> 推理级别**的功能检查清单
（`ReasoningLevelsField.tsx`、`useReasoningLevels.ts`）：

- 该字段拥有 `models[].reasoning_levels` 的三种状态，并在状态行中命名当前状态：**键缺失**（从模型 ID 自动检测）、`[]`（此模型的输入框**推理**选择器被隐藏），以及**非空列表**（仅提供这些级别）。通用数组编辑器无法表达第一种状态，因此通过设置添加的模型否则将永远无法回到自动检测。
- **获取推理级别**使用表单中当前的 ID 调用 `GET /coddy/config/reasoning-levels?model=<id>&provider_type=<type>`，该条目不必已保存，并用网关检测到的内容填充列表，采用输入框所应用的相同 Codex 重映射（`minimal` 变为 `none`）。`provider_type` 是 ID 指向的提供商行的类型，取自正在编辑的设置文档而不是已保存的配置，因此尚未保存或类型刚更改的提供商会按 **保存** 后的方式解析。在存在模型 ID 之前，该按钮被禁用。
- **没有**推理族的模型 ID 不会改动该字段，并会说明这一点。在那里写入 `[]` 会被解读为显式退出并隐藏选择器，这与按钮的请求相反。失败的请求会就地报告错误，也不会改动该字段。
- 状态行在重复获取反馈之前描述操作员正在查看的列表：一旦存在级别（获取或手动添加），它会被读作覆盖值，而 "未检测到任何内容" / 错误消息仅在键仍然缺失时适用。重新键入模型 ID，或手动编辑列表（添加、更改、删除级别），会清除该反馈并放弃任何仍在进行中的应答；在 ID 更改后、手动编辑后或行从列表中删除后到达的应答会被丢弃，而不会覆盖操作员较新的选择；确实到达的应答会通过字段最新的 `onChange` 写入，因此在请求待处理期间编辑的兄弟字段（例如**流式响应**开关）会保留其新值（`useReasoningLevels` 为每个请求编号，因此在字段继续变化后才答复的请求会解析为 `null`）。
- 当键存在时会出现**使用自动检测**，并移除该键，因此下一次保存会省略 `reasoning_levels` 并恢复检测。手动移除最后一个级别是有意达到 `[]` 退出状态的方式。
- `[]` 退出和自动检测默认值在设置保存后双向保留：`ModelEntry.ReasoningLevels` 和 `ModelJSON.ReasoningLevels` 是 `*[]string`，因此省略的键在写入的 `config.yaml` 中保持省略，而不是被序列化为 `reasoning_levels: []`。

### 每会话工作区（文件夹 / 分支 / worktree）

![打开文件夹对话框，页脚左侧有新文件夹和显示隐藏项](../../assets/ui-folder-picker/folder-picker-after-dark-1280.png)

*打开文件夹对话框，页脚左侧有新文件夹和显示隐藏项*

![新文件夹的内联名称行](../../assets/ui-folder-picker/folder-picker-new-row-dark-1280.png)

*新文件夹的内联名称行*

![手机上的打开文件夹对话框，在历史记录打开的位置打开](../../assets/ui-folder-picker/folder-picker-phone-dark-390.png)

*手机上的打开文件夹对话框，在历史记录打开的位置打开*

- 在第一条消息之前，文件夹、分支和 worktree 是**输入框上方面板中的选择项**（面板选择模式内的 `WorkspaceChips.tsx`，辅助函数在 `chat/workspaceContext.ts` 中）：**文件夹**（worktree 时为主项目基名和路径，否则为当前文件夹）、**分支**（当前 git 分支）和 **worktree 复选框**，后两者仅当工作区是 git 仓库时出现。输入框卡片本身没有标签行，因此字段从顶部开始。会话 cwd 仍位于其 worktree 内。
- **一行**：这些选择项是面板自己的项目；文件夹和分支会以省略号收缩，悬停时以及其菜单打开时，较浅的底色会标记选择项，会话开始前没有 git 计数。在手机上以及触摸屏上，选择项和复选框高 **36px**，git 计数通过其上方不可见的 **40px** 命中区域保持纤细外观，因此手指可以点中它们。
- 每当查看的会话变化时，上下文都会从 **`GET /coddy/workspace/context`** 加载，使用 **`X-Coddy-Session-ID`**；没有会话时显示服务器默认 cwd。
- **起始屏幕记住什么**：此浏览器中上次为新聊天选择的文件夹（cookie `coddy_workspace_dir`，每个环境一个），否则为服务器默认文件夹，以及 worktree 复选框（cookie `coddy_worktree`），对所有带 git 的文件夹使用一个选择。离开到起始屏幕的聊天不会把自己的任何内容交给它：不交文件夹、不交链接的 worktree、不交分支。显示的分支是文件夹当前所在的分支，在起始屏幕打开时以及页面重新获得焦点时会再次读取，因此在终端中执行的 `git switch` 会显示出来；可以在那里为新聊天更改它。
- **新聊天会跟随它选择的文件夹**：第一条消息之前没有会话，选择保存在 `App.tsx` 中（`pendingWorkspaceRef`，镜像在 `pendingWorkspacePath` 状态中），并在首次发送时应用。输入框在你键入时列出的所有内容都是针对该文件夹请求的：每个 cwd 作用域请求（`/coddy/slash-commands`、`/coddy/mentions`、`POST /coddy/mentions/check`、`/coddy/workspace/file`）都携带 **`cwd=<folder>`**，位于 **`X-Coddy-Session-ID`** 旁（辅助函数 **`chat/workspaceScope.ts`**），因此 `/` 菜单列出该文件夹的 `.coddy/skills`、其子智能体的 `@agent:` 以及其文件的 `@`。在输入框和对话记录中将 `/name` 标记为技能的已知技能名称会跟随同一文件夹：另一次选择会立即丢弃它们并再次询问，而为已离开的文件夹返回的应答会被丢弃；较早选择的较慢预览绝不会覆盖较晚的选择。设置 → 技能和子智能体列出同一文件夹。
- **第一条消息会立即显示**（issue #357）：聊天在发送它的那一帧就打开它，其下方实时行显示 `Preparing the session…`，同时应用起始屏幕上选择的文件夹和分支，服务器引入会话的 MCP 服务器。无法应用的文件夹或分支不会被静默放弃：不会向另一个工作区发送任何内容，起始屏幕会按发送消息时的状态返回，选择项、文本和文件都在原位，并在输入框下方说明原因（`The workspace could not be prepared: … The message was not sent.`），因此再次发送会再次应用这些选择。即使在选择项应用期间你去了另一个聊天，文本也会在起始屏幕等待，而从未启动的聊天不会留在历史记录中。
- **只选择一次**：文件夹 + 分支 + worktree 在对话开始前设置。一旦对话记录中有消息（`workspaceLocked`），选择项会让位给[运行中聊天的面板](#输入框上方的信息条)，该面板标明聊天的工作位置；服务器对 `POST .../workspace` 回答 **409**，已在进行的轮次也会回答 **409**。运行中的聊天仍可通过轮次移动：智能体的 `worktree_open` 和 `worktree_exit`，以及 `/worktree` 命令，包括用 `/worktree exit` 返回主检出（[Git worktrees](../features/worktrees.md)）。
- **文件夹选择项**会打开**最近**菜单（Claude Desktop 风格）：来自 `localStorage` `coddy_workspace_recents_v1`（`chat/workspaceRecents.ts`）的 MRU 文件夹，当前工作区用 **✓** 标记，长最近列表有本地筛选字段，然后底部是 `Open folder…`，它会打开**文件夹浏览器模态框**（**`WorkspaceFolderModal.tsx`**），其数据来自 **`GET /coddy/workspace/folders?path=`**：行会导航进入文件夹，`..` 向上，**打开**选择当前浏览的文件夹，**取消**关闭。文件夹列表是对话框唯一的滚动端口：它是唯一允许收缩的子元素（`min-height: 0`），因此 **取消** / **打开** 在较短的浏览器窗口中仍可到达，而不会被对话框的高度上限裁剪；越过最后一个文件夹的滚轮手势会留在列表中，而不是滚动后面的页面。已在 WebKit 中用 `external/ui/scripts/webkit-scroll-check.mjs` 验证（见下文）。
- **尺寸**：从平板宽度起，居中的卡片宽 560px（窄窗口更小）；在手机上，它在历史记录打开的位置打开，作为顶栏下方相同的玻璃面板，因此文件夹列表获得屏幕的完整高度。头部绘制方式与历史记录和调度器的头部相同，字段、行和按钮使用应用的字体。
- **文件夹链接**：列表中的目录符号链接在其文件夹图标上带有箭头，并在名称后显示其目标（`→ /path/to/target`），因此链接不会看起来像普通文件夹；点击它会浏览链接的路径。
- **新建文件夹**（页脚左侧的方形 **+**，使用调度器添加按钮的颜色，紧随其后的是**显示隐藏**开关；**取消** / **打开** 位于右侧）会在**路径字段和列表之间**打开一个内联名称行，它是列表的兄弟元素，不是列表中的行，因此绝不会在你眼皮底下滚走，并且在列表本身已缩到没有的短窗口中仍保持完整。**Enter** 或**创建文件夹**会发送 `POST /coddy/workspace/folders` `{"path": <browsed folder>, "name"}` POST；对话框随后显示服务器应答的列表，即**新文件夹**，因此**打开**会立即选中它。**Escape** 或该行的 **×** 会放弃它；没有打开行时，**Escape** 会像**取消**一样关闭对话框。按钮在驱动器层级被禁用（没有可创建的目录），并且在该行已打开时也被禁用；**创建文件夹**在键入名称前保持禁用。已被占用的名称（**409**）会让该行保持打开并保留键入文本并说明原因，任何其他失败也会如此，不会创建任何内容，浏览的文件夹也不会改变。
- **离开驱动器（Windows）**，从驱动器根目录执行 `..` 会打开**驱动器层级**（`?path=:drives:`，响应中的 `drives:true`）：每个卷一行（`C:`、`D:`、…），其上方没有 `..`，并且**打开**被禁用，因为它是导航位置，不是工作区。**路径行是可编辑字段**（`workspace-modal-path`）：键入或粘贴路径并按 **Enter** 会跳转到那里，来自资源管理器*复制为路径*的两侧引号会被去除（`cleanPathInput`），当字段持有未访问的路径时，主按钮显示**前往**而不是**打开**，因此粘贴的路径绝不会被误认为正在打开的文件夹。浏览器从 `pathParent(ctx.path)` 开始，这会保留当前驱动器（过去会将 Windows 路径折叠为 `/`）。选择会调用 `POST /coddy/sessions/{id}/workspace` `{"path"}`，会话 cwd 会切换并持久化；技能、项目规则、斜杠命令、已配置的 MCP 服务器（通过其信任门控为新工作区重新连接，旧工作区的关闭）以及 SessionStart 钩子上下文会从新 cwd 重新派生。
- **分支选择项**会打开分支列表（当前分支在前并标记为已选中），长列表有本地筛选。选择会向 `{"branch", "worktree": <checkbox>}` 发送 POST：默认就地检出，或当复选框开启时在 `<repo>/.coddy/worktrees/<branch>/` 下创建专用功能 worktree。创建 worktree 会获取 `origin`，从最新的 `origin/HEAD` 默认分支创建新分支，并拒绝默认分支或跟踪它的分支。没有复选框时，选择已在另一个 worktree 中检出的分支会跳转到那里（包括返回主检出）。
- **分支列表是最新的**：每次打开时，列表显示它已知的分支（本地分支以及上次获取后的远程分支），并在后台获取仓库的每个已配置远程（`POST /coddy/workspace/fetch`，即新聊天选择的文件夹或会话的文件夹）。筛选字段右端的加载指示器表示获取正在运行；行仍留在屏幕上并可在此期间选择，因此菜单保持打开时的尺寸。获取返回后，列表会显示获取后 git 所拥有的分支，因此自上次获取以来推送的分支无需终端即可加入列表，而其远程上已删除的分支会离开列表（同名本地分支会保留，它是你的）。在获取返回前选择的分支会保持为选择。只有尚无分支可显示的列表在等待时会显示**正在从远程获取分支…**。获取是 `git fetch --all --prune` 且不包含标签：它只移动远程跟踪引用，绝不检出、重置、合并或触碰工作树、索引、本地分支或标签。它使用仓库自己的凭据（凭据助手、ssh agent），并且绝不会在终端或 askpass 对话框中询问凭据；没有 git 二进制文件时，只能刷新不需要凭据或从 agent 获取 ssh 密钥的远程。整个刷新限制为 20 秒，超过即视为失败。
- **仅在远程上的分支**与本地分支位于同一列表，在当前分支之后按一个字母顺序排列，按本地分支的方式命名（`feature/search`，而不是 `origin/feature/search`），并在行首用云朵标记；你本地已有的分支只列一次，作为本地分支，多个远程拥有的名称也只列一次。选择一个会创建跟踪远程分支的本地分支（如果 origin 有则跟踪 origin 的，否则跟踪按名称排序的第一个远程的），按复选框所述就地或在工作树中创建；在第一条消息之前，面板会显示该选择，并在聊天开始时创建分支。
- **刷新失败**（无网络、登录被拒、远程不响应）会保留原有列表，并在其上方用琥珀色提示说明，**无法刷新远程分支。列表可能已过时。**，其下方是 git 原因的第一行，完整原因在提示的工具提示中。工作区仍可使用：列出的分支仍可选择。

![获取远程分支时的分支列表：它已知的分支仍会列出并可选择，筛选字段右端有一个小加载指示器](../../assets/ui-branch-menu/branch-menu-refreshing-dark-1280.png)

*获取远程分支时，列表保持不变，筛选字段中的加载指示器标记刷新*

![刷新后的分支列表：先是当前分支，然后本地分支和仅存在于远程的分支按一个字母顺序列出，每个仅远程行开头有云朵](../../assets/ui-branch-menu/branch-menu-remote-dark-1280.png)

*刷新后的分支列表：仅存在于远程的分支用云朵标记*

![刷新失败：列表上方有带 git 原因的琥珀色提示，缓存的分支仍会列出并可选择](../../assets/ui-branch-menu/branch-menu-fetch-failed-dark-1280.png)

*刷新失败时会在警告下保留缓存列表*

- **Worktree 复选框**（`composer-worktree-checkbox`，真实为 `input[type=checkbox]`）是 worktree 首选项；当会话已在链接的 worktree 内运行时，它显示为已选中且禁用。
- **历史记录文件夹分组**使用每个 git 会话行上作为 `repoRoot` 返回的主检出路径。位于 `main` 中的对话及其 worktree 中的对话会出现在同一个项目标题下，而每个会话保留自己的 cwd。参见 [Git worktrees](../features/worktrees.md)。
- **会话前（草稿/主页）**：选择项存储在客户端，通过 `GET /coddy/workspace/context?path=` 预览，并在首次发送时、`POST /v1/responses` 之前应用到新会话 ID。从项目对话返回起始屏幕时，会将该对话的文件夹保留为下一次聊天的预览；切换到另一个会话会丢弃待处理的选择项。
- 错误（缺少文件夹 **400**、git 冲突 / 工作区锁定 **409**）会保留当前标签；上下文会重新获取以保持真实。
- 自动化检查：`chat/workspaceContext.test.ts`、`chat/workspaceRecents.test.ts`（辅助函数）、`chat/WorkspaceChips.test.tsx`（标签、菜单、打开的分支列表背后的刷新、警告、远程行、模态框、锁定）、`App.workspaceHome.test.tsx`（刷新请求的是哪个文件夹，以及刷新期间选择的分支会保持为选择）、`features/web_ui_menus.feature` 中刷新期间分支列表的场景；后端行为在 `features/workspace_switching.feature`（godog）中指定为可执行规范，获取本身在 `internal/gitws/remotes_test.go` 中（每个远程、prune、不触碰本地、refspec 守卫、截止时间、两种 git 后端）。
## 会话列表

![按文件夹分组的历史记录，标题旁有一个加号](../../assets/sessions-history-grouping-dark-1280.png)

*按文件夹分组：标题即文件夹名，旁边的加号会在该工作区中新建聊天。置顶行位于顶部 "置顶" 组下*

![置顶分组，正通过行本身拖动](../../assets/sessions-history-pinned-dark-1280.png)

*置顶会话在每个模式下都作为一个分组排在最前，通过行本身拖动排序；被移动的行会淡出，它将要落到的行会被标出*

![一个用斜杠命令启动的新聊天正在命名：标题处有闪烁条，标签处有两条](../../assets/session-title-naming-dark-1280.png)

*用 `/rpa-init` 启动的聊天，在 describe 调用推导其名称时：其历史记录行用标题和标签的占位符代替命令。*

![一个会话的行菜单](../../assets/sessions-history-row-menu-dark-1280.png)

*每行一个控件：置顶、重命名和标签，然后一条分隔线，以及两个把会话移出列表的操作，其中删除使用危险色*

![一个会话的标签编辑器](../../assets/sessions-history-tag-editor-dark-1280.png)

*一个会话的标签：每个标签片上都有叉号，一个输入框会提供此历史已经归入的词语，以及输入内容将被存储为的折叠拼写*

![历史记录过滤器菜单](../../assets/sessions-history-filters-dark-1280.png)

*四行，每行写明其问题与当前生效的答案；只有偏离默认值的值才会着色，选项在行旁展开。这里选择了 **CLI 运行**，列表只保留由脚本通过 `coddy -p` 启动的运行*

![手机上的历史记录过滤器菜单](../../assets/sessions-history-filters-phone-dark-390.png)

*在手机上，菜单两侧都没有空间容纳列表，因此选项会在菜单内、其所在行下方折叠展开，且不超出屏幕*

![删除聊天前的共享确认对话框](../../assets/confirm-delete-chat-dark-1280.png)

*删除聊天前的共享确认对话框*

- **历史** 面板通过 `GET /coddy/sessions` 列出会话（仍是 **抽屉**，不是常驻的第二列）。
- 分页使用 `limit` 和 `cursor`，并通过 **无限滚动** 加载更早的行。
- 可选的 **`q`** 查询字符串（**标题、工作区路径、某个标签，或首条 **`user`** 消息内容**，不区分大小写的子串；**不是** 全聊天搜索）。搜索输入更新使用客户端防抖。
- **决定列表显示什么的一切都在一个控件里**：搜索行右端的滑块按钮会打开一个包含四行的菜单（**`SessionsFilterMenu.tsx`**）。它和搜索框放在一起，因为两者都会收窄下方列表；抽屉头部只保留关闭按钮。每一行都写明其问题与当前生效的答案，选项在其旁边展开：用鼠标悬停在某行，或点击、轻点该行，前一行会折叠；点击鼠标刚打开的那一行则保持打开。菜单渲染到文档中，而不是渲染到会裁剪溢出内容的抽屉里，因此子菜单可以越过抽屉边缘。列表会从拥有完整宽度的一侧打开，优先右侧，靠近窗口边缘时改用左侧；在手机上，两侧都没有足够空间，因此选项会在菜单内、其所在行下方折叠展开，轻点某行会打开或折叠它。菜单的宽度永远不会超过其控件左侧剩余的空间；有列表折叠在其中时，菜单会滚动，而不是溢出屏幕底部。
  - **状态**：**活跃**（默认）、**已归档**、**全部**，对应 **`archived`** 查询参数。它排在最前，因为这是最常被问及的问题。
  - **环境**：**全部**、**本地**、**网关**、**CLI 运行**，然后每个已配置远程各一行。前四项会收窄当前所读取服务器的列表（**`origin`**）：所有会话、在此主机上打开的会话、消息网关所持有的聊天，或一次性打印模式的运行记录（**`coddy -p`**），后者会被其他每一行排除（[会话](../features/sessions.md#标签与归档)）。它们是**过滤器**，因此不会重新加载任何内容。远程行是**开关**：它会把整个应用指向该服务器，与侧栏的环境菜单所做的是同一件事，而那确实会重新加载：来源过滤器会随其他所有内容一起重置。当只有一个可选项时，这一节会完全省略。
  - **分组方式**：**无**、**日期**、**文件夹**（默认，因为人们记住一段会话，更多是靠它涉及哪个检出目录，而不是它发生在哪一天）、**标签**。
  - **排序方式**：**最后活动**（默认）、**创建日期**、**名称**：**`sort`** 参数，每种都采用适合其值类型的自然方向。

  一条分隔线把前两行与后两行分开：前一组决定*列出什么*，后一组决定*如何排列*。只有当某行的值**不是**默认值时，才会用**强调色**绘制；如果菜单里每一行都着色，就无法说明哪些条件被收窄了。这四项都会在重新加载后通过各自的 cookie 记住（**`coddy_sessions_group`**、**`_status`**、**`_origin`**、**`_sort`**）：下一次页面加载就会忘记的过滤器不算设置。
- **分组**是客户端职责：服务器返回一个扁平且有序的页面，由抽屉决定标题落在何处，因此切换分组不会产生请求，也绝不会重排服务器在组内排好的顺序。标题是该分组桶自身的名称，后面跟一个插入符，点击名称执行的就是折叠；没有任何内容落入的桶不会绘制，带有三个标签的会话会在三个标签下都列出。
  - **日期** 按本地日历日分桶：**今天**、**昨天**、**过去 7 天**、**过去 30 天**、**更早**，以及最后的 **无日期**，用于没有时间戳的会话包。
  只有 **文件夹** 标题带有 **+**；日期或标签不是放置会话的地方。

  - **文件夹** 以完整路径为键并显示文件夹名，因此两个都名为 `one` 的检出目录会保持区分；当同一个名称确实出现在多个标题中时，每个标题都会在其下方写出完整路径，因为仅凭名称无法区分它们。没有工作区的会话排在最后。文件夹标题还带有 **+**，用于启动一个已指向该工作区的新聊天；该选择会经过与输入框的文件夹标签片相同的会话前路径，因此服务器会为新会话解析该文件夹当前的 git 分支。
  - **标签** 会把无标签会话排在最后。
- **对一个会话可执行的操作都收在其 ⋮ 后面**：**置顶**、**重命名** 和 **标签**，这三个会改变行的位置或它对自己的说明；然后分隔线下方是 **归档** 和 **删除**，这两个会把会话移出列表，其中删除使用危险色。每个操作一个图标会让标题各损失一个按钮的宽度，还会让删除只差一次误点；在菜单里，这些操作有空间显示文字。同一时间只打开一个菜单，**Escape** 会关闭它；菜单通过 portal 渲染到抽屉之外，因此不会在边缘被截断（靠近窗口底部时，它会翻转到该行上方）。
- **置顶会话在顶部是一个分组**，标题为 **置顶**，并由一条分隔线隔开，在每个分组模式下都是如此：它们是由操作者手动维护的单一列表，而不是贯穿每个分组的一条带。同一会话既出现在顶部*又*出现在其文件夹内，会带来拖动时到底移动哪一个的问题。
- **置顶项通过拖动行本身重新排序**：用鼠标按下并移动它（普通点击仍会打开会话）；用手指按住该行片刻，然后拖动（快速滑动仍会滚动列表）。拖拽由**指针事件**驱动（HTML5 拖放永远不会从触摸开始），列表显示放置将落到的位置，而不是让一行跟随指针；这样做在滚动后仍然有效，也不会产生合成层。放置后的顺序通过 **`POST /coddy/sessions/pins/reorder`** 写入，它会接收**整个**顺序，新置顶项会排在已有置顶项**上方**。该行不会在客户端移动：顺序以服务器的答复为准，因此变更后会重新读取列表。
- **已归档会话在列表中会变暗，且无法写入**：其行会变灰，打开它会把输入框替换为一条说明它已归档的通知，以及一个将其移出的按钮。服务器端不会拒绝任何内容：归档只是一个架子，不是锁；但若把输入框留在那里，就会诱使用户发出提示词，从而悄悄撤销操作者自己的*暂不*。输入框通过 **`GET /coddy/sessions/{id}/messages`** 得知这一点，其中带有 **`archived`**：会话列表会跳过归档，因此屏幕上的会话可能不在客户端持有的任何页面中。
- **归档**会把会话移出工作列表而不删除它（**`PATCH`** 配合 **`archived`**）。该行会立刻离开列表，请求在其后发出，列表不会再次读取，因此滚动已加载的行以及列表中的位置都保持不变，下一个要归档的会话就在那里。被拒绝的请求会把该行放回原处，并在该行本身上说明（*该会话未被归档*），因为到那时列表顶部（列表错误显示的位置）通常已不在视野中；重试或关闭历史时，这条注记会消失。请求完成时已经在途的列表加载无法把该行放回，而且每从已加载部分归档一个会话，下一页就会提前一行开始，因此不会有会话落在两页之间。已归档行会在标题旁带有归档**标记**，表示会话所处的状态，而不是其标签中的一个标签；同一个菜单项会把它放回。归档当前屏幕上的会话会立刻把输入框替换为同样的归档通知，放回它则会恢复输入框；该标志会跟随 **`PATCH`** 的答复，因此不必等待下一次对话记录加载。**状态** 过滤器会记住其设置，因此被搁置的会话会一直避开，直到操作者主动请求它。
- 一行的**标签**会在标题下方渲染为小标签片（标题独占第一行）。它们由标题生成提出，在行菜单的 **标签** 编辑器中手动编辑，并由模型自己的 `session_describe` 工具写入（[会话](../features/sessions.md#标签与归档)）。
- **正在命名的聊天**，即新聊天首条消息的 `POST /coddy/describe` 调用，会在其标题将出现的位置显示一个闪烁条，在历史记录行和聊天头部都是如此，并在其标签将出现的位置显示两个短条，而不是首条消息（一个 `/rpa-init` token）或 *新聊天*。调用一答复，两者就会立即取得名称和标签；空答复、失败或 30 秒内没有答复也会结束占位符，而稍后到达的名称仍会落位。在 `prefers-reduced-motion` 下，这些条保持静止。运行命令的首条消息会按该命令的作用命名（[会话](../features/sessions.md#标签与归档)）。
- **标签编辑器是显示标签的两处位置中的同一个组件**（**`SessionTagEditor.tsx`**）：每个标签片上都有叉号，一个输入框会提供此历史已经使用的标签（最常使用在前，排除自身，前缀匹配优先），**Enter** 提交所输入的内容，方向键和 **Enter** 采纳建议，空输入框上按 **Backspace** 删除最后一个标签片，**Escape** 关闭。没有保存按钮：每次更改都会立即成为一次 **`PATCH`**。行会在请求**之前**取得新集合，因此在第一次请求仍在途中时做出的第二个手势会基于它继续，而不是撤销它；服务器答复的集合（折叠为小写、空白转为连字符、最多八个）随后会替换它，因此标签片不会在一次刷新后改变拼写，而被拒绝的写入会恢复该行原本携带的内容。正在输入内容的折叠形式仅在不同于所输入内容时显示在输入框下方。
- 指示器
  - 一个脉动圆点，即强调色未读圆点再暗三分之一，会出现在每个轮次仍在运行的行上，包括当前打开的会话。它也会出现在**仍有后台任务在途且没有轮次运行**的行上，因为脱离的工作会比启动它的轮次活得更久，而没有该标记的行会被读作已完成；圆点的工具提示会说明它属于两者中的哪一种。该计数来自会话列表的 **`backgroundRunning`**（模型自己的任务，不含已完成项，不含系统事务），列表会在轮次事件和抽屉轮询时重新读取，因此在轮次**之外**开始或结束的任务会在下一次轮询时改变圆点，而不是立刻改变。等待权限或问题的轮次会改为显示该标记。
  - 每个在其他方面空闲的行都有透明轮廓环，因此其标题不会与分组标题混在一起。只有当某个会话在并非当前活动聊天期间完成时，才会用强调色圆点替换它。在简化模式下，该已完成未读圆点为半透明；其他所有会话标记保持通常外观。
  - 以真实错误结束的轮次会记录 **`lastErrorSeq`**。当其代次比会话的读取游标更新时，历史记录会显示实心红点；打开会话会确认它，并把标记变为红色轮廓。该轮廓会一直保留，直到之后某个成功轮次清除错误状态。未见的错误及其已确认的轮廓，都优先于通用的强调色未读完成圆点。
  - 当会话正在等待用户权限或交互式问题时，会出现问号图标。它保持静止：轮次是在等待人，而不是在取得进展。问题待处理事件会让历史记录立即重新读取会话列表；全局事件从不携带问题文本，问题文本保留在输入框流中。

![带有空闲、未见错误和已查看错误状态点的历史记录行](../../assets/sessions-history-state-dots-dark-1280.png)

*历史记录分组让每一行在视觉上都有依托：已完成的会话有中性轮廓，未见的失败轮次有实心红点，已查看的失败有红色轮廓。*
- CRUD
  - 通过 `PATCH /coddy/sessions/{id}` 设置 `title` 来重命名。
  - 通过 `DELETE /coddy/sessions/{id}` 删除。
  - 新建聊天从主屏幕开始。会话 id 仅在首次发送时创建。

会话重命名体验

- 标题重命名可在聊天头部完成，也可通过该行 **⋮** 菜单中的 **重命名** 完成，后者会把行标题变成一个已选中名称的输入框。
- 失去焦点时，界面会通过 `PATCH /coddy/sessions/{id}` 保存；**Enter** 保存，**Escape** 保持标题不变。

会话删除体验

- 删除是该行 **⋮** 菜单中的一行，单独分隔并使用危险色。
- 点击删除会显示一个确认对话框，然后调用 `DELETE /coddy/sessions/{id}`。此处对话框打开时焦点在 **删除** 上，因此 **Enter** 会完成点击所开始的事：行菜单中的 **删除** 只做一件事，而对话框询问的也正是这一件事。在其他所有地方，共享对话框仍以 **取消** 为焦点打开，因为在那些地方，误按 Enter 绝不能确认没人想确认的事情（**`initialFocus`** 在 **`confirm({...})`** 上）。
- 如果被删除的会话**不是**主聊天中当前显示的会话，则将其从列表中移除（并从服务器刷新），并**保持历史抽屉打开**。不要更改 URL，也不要清空仍留在屏幕上的会话的对话记录。
- 如果被删除的会话**就是**当前显示的会话，则导航到**新聊天**（空白起始屏幕，清除会话 hash），**关闭**历史抽屉，并像普通主页切换那样清除与输入框相关的状态。
- 一次删除多个会话，以及查看每个会话的开销，都在 **设置 -> 会话**（见下文）。历史仍然是*打开*会话的地方。

## 设置：会话管理

![会话管理表格，当前打开的会话受到保护](../../assets/sessions-management-table-dark-1280.png)

*所有已存储内容，包括归档和打印运行：一行说明它位于何处以及归在什么下面，终端符号位于由 `coddy -p` 启动的运行标题之前，其标签后的 **+** 打开与历史相同的编辑器，搜索下方的两个选择框收窄归档与来源，列标题对整份列表排序，搜索旁的两个图标是仅有的破坏性控件*

**设置 -> 会话**（**`#/settings/sessions_manager`**、**`SessionsManager.tsx`**，纯辅助函数位于 **`sessions/sessionManagerRows.ts`**）把已存储历史呈现为表格，而不是需要滚动的列表，放在一个 **会话管理** 字段集中，其 (i) 说明该表的用途。它与外观一样是客户端标签页：表格通过 **`/coddy/sessions`** 读取和移除会话包，因此它会在配置 schema 加载完成前渲染。其 id 是 **`sessions_manager`**，因为 **`sessions`** 已经是配置键：存储目录；schema 就绪后，该标签页会在表格上方（**存储**）编辑它，并通过 **全部保存** 保存。

- **行**来自 **`GET /coddy/sessions?include_stats=true&include_print=true`**，每次 50 条，并带有 **加载更多** 按钮。每行显示标题，下方显示其**工作区**，会话覆盖的**模型**（若从未覆盖则为 **`default`**，表示当时 **`agent.model`** 是什么就是什么），**消息数**，**总 token 数**（输入和输出在单元格工具提示中），以及**创建** / **更新**日期（精确时刻在工具提示中）。在 Coddy 记录创建时间戳之前存储的会话包会显示 **—**，而不是根据稍后保存编造出的日期。
- **搜索** 使用与历史抽屉相同的 **`q`** 过滤器：标题、工作区、标签或首条用户消息，不区分大小写，并在输入时防抖。
- **排序** 由列标题完成：**会话**（标题）、**消息数**、**Token 数**、**创建时间** 和 **更新时间** 都是按钮，已排序列带有插入符和 **`aria-sort`**。点击已排序列会反转顺序；点击其他列时，会从该类型值自然的方向开始，日期或计数从最大值开始，标题从首字母开始。顺序会传给服务器（**`sort`** 和 **`order`**），并应用于**分页之前的整个过滤后列表**，因此 **加载更多** 会继续排序结果，而不是重新排序某一页。
- **归档** 是搜索旁的选择框：**工作列表**（默认）、**归档**、**全部**。已归档行带有中性的 **已归档** 徽章，其工具提示说明它何时被搁置。会话从**历史**归档，而不是在这里；此标签页用于查看归档内容并清空归档。
- **来源** 是第二个选择框：**所有来源**（默认）、**本地**、**网关**、**CLI 运行**（**`origin`**）。与历史不同，表格会一次性打印模式的运行记录（**`coddy -p`**）与其他记录一起列出，因为这里正是清理它们的地方；这样的行会在标题前带一个终端符号，其工具提示显示 **CLI 运行**（[会话](../features/sessions.md#标签与归档)）。两个删除范围也都会触及打印运行。
- **删除只有两个范围，绝不会有更多**：搜索框旁的垃圾桶图标会移除**已勾选**的行（**`POST /coddy/sessions/bulk-delete`** 及其 id），旁边的归档盒图标会清空**归档**（**`scope: "archived"`**，并在 **`except`** 中指定当前打开的会话，因此即使对于由服务器解析的范围，下面的保护仍然成立），两者都位于共享确认对话框之后。每个按钮做什么由其**工具提示**和无障碍名称说明，而不是显示在按钮表面的标签；唯一绘制的文字是第一个按钮上的**选择计数**徽章，因为工具提示无法一眼展示这一点。已勾选的行是操作者能看到的内容；归档是由服务器解析的范围，因为归档可能包含比当前页更多的内容，确认对话框会用文字说明这一点。不存在第三个能触及更广范围的按钮。
- **标签**在标题下方渲染为标签片。点击其中一个会把表格收窄到归在该标签下的会话（**`tags`**），工具栏下方的一行会说明当前显示哪个标签，并提供清除它的链接。
- 一行**没有自己的删除操作**：勾选框和那一个按钮就是每行的全部界面，因此破坏性点击的范围从不会有歧义。
- **清空页面** 是表头复选框加上那一个按钮。当行数超过一页时，先 **加载更多**；表格下方的摘要行说明已列出多少行、已勾选多少行。
- **你打开的会话受到保护**，不受这里任何删除影响，包括归档范围：其行会高亮并标记为 **打开**，其勾选框被禁用，工具提示说明原因，表头复选框会跳过它，清空归档时也会按名称保留它，即使它正是被归档的那个。表格不能在你脚下把聊天拿走；请先关闭它或切换到另一个会话，然后从**历史**中删除它。
- **选择跟随表格显示的内容**：表头复选框会勾选和取消勾选已渲染的行，隐藏已勾选行的搜索会一并带走其勾选（清除搜索后该行会以未勾选状态回来）。破坏性操作永远不会触及屏幕外的行，勾选也不会因为曾在视野外保留而稍后重新出现。
- 无法移除的会话（其树中的某个轮次仍在运行）会在工具栏下方连同原因被**报告**，其行会保留。其他会话仍然已被删除：请求会分别以 **`deleted`** 和 **`failed`** 答复。
- 列表在**每次删除后重新读取**；不会重新加载任何内容。如果面板背后屏幕上的会话正是被删除的会话之一，聊天会重置为新聊天，设置会保持打开在此标签页。
- 表格是该标签页中唯一可水平滚动的元素，因此狭窄的外壳会滚动列，而不是滚动页面。
## 系统（设置标签页）

**设置 → 系统**（**`#/settings/status`**、**`SystemSection.tsx`**）是一个只读页面，介绍 Web UI 所连接的安装：它是什么、如何启动、运行着什么、会话磁盘还有多少空间，以及哪些事项需要注意。它是头部组中的一个客户端标签页，位于 **会话** 之后，在 schema 加载前绘制，并存在于每个 `http ui` 构建中；中继自身的主页没有此标签页。它读取 **`GET /coddy/status`**（[HTTP API](../reference/http-api.md#系统状态)），因此通过中继访问的节点和远程环境显示的是那台服务器，而不是提供该页面的服务器。页面上的任何内容都不会修改设置，也不会将设置标记为未保存。首次打开该标签页时，它会作为独立分块加载。

![1280 px 下的设置 → 系统：一条提示带显示 3 项需要注意、等待重启、指标导出器已停止和中继不可达，说明保存的地址将在 Coddy 重启后生效，过去一小时的四张图表，然后是安装、子系统（含一行关于中继的说明）、磁盘、活动和模型提供商](../../assets/web-ui/system-attention-dark-1280.png)

*设置 → 系统，有三项需要注意：一个等待重启的已保存 HTTP 地址；指标导出器无法占用其端口并已停止；一个节点已超过一小时未注册到的中继，其名称出现在子系统行下方的警告行中。提示带下方的区块将依次说明。*

- **摘要。** 第一个区块。存在问题时，它会显示 **N 项需要注意** 并列出这些项，每一项都会将焦点移到其所在区块：**磁盘空间不足**（为零时为 **磁盘已满**）、等待重启的项、已停止的子系统、持续失败的提供商、**中继不可达**（它会将焦点移到 **子系统** 标题）。没有问题时完全不显示摘要：下面的区块已经说明了情况。**部分**（某个区块无法读取，例如磁盘）和 **陈旧**（上次刷新失败）总会说明缺少什么或哪些内容已过期，并且部分状态会与问题同时出现。它是页面中唯一的实时区域。
- **重启提示。** 运行中的进程无法采用的已保存监听地址（`httpserver.host`、`httpserver.port`、`swarm.host`、`swarm.port`）会显示 **已保存的地址 `{address}` 在 Coddy 重启后生效。**，并给出该地址；另一项需要新进程的已保存更改，即在未带会话存储启动的进程中启用了会运行轮次的界面，会显示 **已保存的更改在 Coddy 重启后生效。** 两者都链接到 [应用已保存的地址](../operate/serve.md#应用已保存的地址)。当受监督的进程将重启进入无法启动的设置时，提示会显示 **已保存的设置无法启动：** 以及拒绝这些设置的检查所给出的消息。
- **此安装。** 版本和修订号（发布版本会链接到其发布说明）、构建标签、平台和 Go 版本、已运行时长、**启动者**（systemd 服务、后台守护进程或前台）以及配置文件，主文件夹写作 `~`，并有一个复制其路径的按钮。不显示其他路径。
- **子系统。** 配置启用的每个子系统各占一行：其状态（运行中、启动中、已停止，或保留：已关闭但仍在运行）及其地址，一个已保存、会在当前运行地址之后等待重启的地址。导出器所在行会说明抓取是否需要令牌，但绝不会显示令牌。向中继（`swarm.join`）的注册若已离线 5 分钟，会为每个中继添加一条警告行：**未注册到中继 `{relay}`，持续 `{duration}`**。
- **磁盘。** 一个仪表（`role="meter"`）用于显示存放会话的磁盘，即可用空间相对于其总大小。低于 512 MiB 时它会发出警告，这也是 [Monitoring](../operate/monitoring.md#随附告警规则) 的 `CoddyDiskLow` 规则所监视的那一行；读取失败时会显示 **不可用**。不显示主目录所在卷。[故障排查](../getting-started/troubleshooting.md#会话磁盘几乎已满) 列出了可释放的内容。
- **活动。** 正在运行的轮次及其最旧轮次的持续时间、等待回答的提示词（权限提示和问题），以及按类型划分的未完成后台任务（命令、子智能体、服务器）。
- **最近一小时** 和 **自启动以来的模型提供商** 只存在于使用 `metrics` 标签构建的二进制中，无论 `metrics.enable` 怎么说；没有它时，两者都不会绘制，也不会用占位符代替。**最近一小时** 是以 15 秒为步长的四张图表：每分钟模型尝试次数及提供商错误、尝试耗时、每分钟输入和输出 token，以及正在运行的轮次及后台任务。成功的尝试用主题的绿色绘制，错误用其红色绘制，因此每个主题都保留自己的调色板。折线中的空缺表示没有值的样本，不足一秒的时间以毫秒显示。表格给出自进程启动以来每个提供商的尝试次数、提供商错误百分比、平均耗时和最近一次失败。序列保存在内存中，重启会清除；[Monitoring](../operate/monitoring.md#system-页面) 说明了它们的来源。在手机上，每张图表都是带最新值的一行迷你折线图。

  ![浅色主题下的设置 → 系统，没有需要报告的事项：不显示摘要，先是最近一小时的四张图表，然后是安装、子系统、磁盘、活动以及模型提供商表格](../../assets/web-ui/system-clear-light-1280.png)

  *浅色主题下没有需要处理的事项，经过一小时的模型流量后：每张图表都覆盖一小时，并从样本开始处开始，因此运行不足一小时的进程会把线画在绘图区右侧。表格给出每个提供商的尝试次数、错误、平均耗时和最近一次失败。*

  ![390 px 手机上的设置 → 系统：包含三项的注意提示带、已保存地址提示，以及每张图表作为带最新值的一行迷你折线图](../../assets/web-ui/system-attention-dark-390.png)

  *手机上的同一个标签页：每张图表都是带最新值的一行迷你折线图。*

  ![在未启用 metrics 标签构建的二进制中的设置 → 系统：安装、子系统、磁盘和活动区块，没有图表或提供商表格](../../assets/web-ui/system-nometrics-dark-1280.png)

  *未使用 `metrics` 标签构建的二进制中的标签页：没有图表，没有提供商表格，也没有用占位符代替它们。*

- **注意徽标。** 侧栏条目会在其标签后显示项目数量，手机卡片在其右上角显示数量；卡片的描述会用 **N 项需要注意** 代替其简介。数量为零、遇到 404、401 或读取失败时没有徽标。该数字是标签页所显示状态文档中 `attention` 列表的长度，因此两者一致；离线中继只计一次；浏览器没有自己的规则。设置会在打开时读取一次状态，随后徽标跟随标签页和 **全部保存** 所发起的读取：没有任何轮询，设置关闭时也不会读取。通过中继挂载时，它是节点的数量；在远程环境中，则是远程环境的数量。

  ![390 px 手机上的设置：卡片网格中系统卡片标有 3，并用 3 项需要注意 代替其简介](../../assets/web-ui/system-badge-dark-390.png)

  *手机上的卡片网格：系统卡片带有数字，并在其简介位置用文字说明该数字。*

- **刷新。** 打开时刷新，之后在标签页显示且页面可见时每 15 秒刷新一次。设置打开时进行的读取若不到 5 秒，会作为该标签页的第一次读取；否则标签页会立即读取。页脚处的 **更新于 N 秒前** 不是实时更新的。
- **入口。** 点击 **全部保存** 后，抽屉会读取一次状态，因此等待重启的已保存更改会立刻显示提示；响应 404 的服务器则不显示任何内容。
- **降级状态。** 没有该路由的服务器（404）会显示 **此服务器不上报其状态** 并给出其版本。被拒绝的读取（401）会说明这一点。位于中继之后且离线的节点会显示为离线，并给出最后一次见到它的时间：对于主动拨号连接到中继的节点，就是该连接结束的时刻。刷新失败会保留上次的值并置灰，摘要会显示陈旧。只有问题会用警告色调绘制摘要，一切正常时不绘制摘要，其他每种状态都是弱化提示。文档中缺少的区块会显示 **不可用**，绝不会显示为零。

  ![服务器停止后的设置 → 系统：提示带显示无法刷新并显示最后读取的值，每个值都置灰](../../assets/web-ui/system-stale-dark-1280.png)

  *标签页加载后服务器停止了。下一次刷新失败，因此提示带说明显示的是最后读取的值，标签页继续将这些值保留在屏幕上，并置灰显示。*

- **搜索。** 该标签页可通过其名称和描述找到，其四个始终绘制的区块可通过标题找到；不会搜索任何值，也不会搜索 **最近一小时** 和 **模型提供商** 标题，因为它们取决于构建。区块搜索结果会打开该标签页，滚动到标题，为其绘制环形高亮并将焦点移到其上；如果分块加载较晚，则会等待；在它到达前离开会取消跳转（[设置：搜索](#设置搜索)）。
- **无障碍。** 摘要项会将焦点移到其区块标题，仪表具有名称和值，每张图表都是带有指标及其最新值标签的图像。
- 自动化检查：**`SystemSection.test.tsx`**（每种摘要和降级状态、仪表、没有 `metrics` 时不显示图表、焦点目标、隐藏时停止轮询、远程环境）、**`systemStatusApi.test.ts`**（按顺序应用回答、切换环境、共享读取、复用窗口）、**`Settings.test.tsx`**（徽标：其读取、其数字、为零或 404 后不显示、卡片和侧栏的无障碍文本、中继的主页）、**`features/system_status.feature`** 和 **`bdd_diagrams_test.go`**（分块保持在 20,000 B 以下，且 `Sparkline` 不进入 `app.js`）。

## 跨标签页共享的服务器事件

每个标签页都需要 **`GET /coddy/events`**：轮次开始和结束、队列变化、账户用量、问题等待状态的转换以及配置重新加载都会到达那里。`session_question_pending` 通知在两个生命周期边界上只携带 `{object, sessionId}`。该通知和 `questionPending` 是拥有交互式等待的 Coddy 服务器进程的进程本地内容；SPA 会重新读取会话行以获得 `questionPending`，而问题文本仍保留在每轮次的输入框流中。在纯 HTTP/1.1 下，浏览器会为其所有标签页保持到同一主机的六个连接，因此当每个标签页各自持有一个流时，六个打开的标签页就会占用全部连接，任何标签页都无法发送请求：不是提示词，不是 Stop，也不是历史读取。现在同一环境的标签页共享单个连接。

- **SharedWorker** - 在浏览器支持 SharedWorker 的情况下（桌面 Chrome、Edge、Firefox 和 Safari 16+，包括纯 http 和 https），由 `events-worker.js` 持有连接。每个标签页连接一个端口并发送 hello；第一个 hello 打开流，最后一个标签页离开时关闭它。远程环境会在 hello 中传入其地址和 bearer token，因为页面的 fetch shim 无法进入 worker。
- **Web Locks 和 BroadcastChannel** - 在没有 SharedWorker 的情况下，标签页通过 `navigator.locks` 选出一个自己的代表；持有锁的标签页持有流，并通过 `BroadcastChannel` 中继它。当该标签页关闭时，队列中的下一个会取得锁，并在自己的连接建立前报告流已断开。Web Locks 仅存在于安全上下文中（https、`localhost`、`127.0.0.1`）。
- **每个标签页一个流** - 两者都没有时，或者 worker 加载失败或五秒内未响应时，标签页会保留自己的连接，就像以前的每个标签页一样。这是许多打开的标签页仍可能占用全部六个连接的唯一情况：在不支持 SharedWorker 的浏览器中，通过纯 http 访问网络地址。
- **范围** - 标签页仅在一个环境内共享：worker、锁和通道以本地服务器命名，或以远程地址及其令牌指纹（绝不是令牌本身）命名，再加上协议编号，使不同构建的标签页永远不会相遇。
- **标签页听到的内容** 与它自己建立连接时会收到的内容相同：连接状态、每个运行中轮次一个 `turn_started`、`ready`，然后是实时事件。加入已启动流的标签页会收到正在运行的轮次以及单独发给它的 `ready`，因为 `ready` 会让标签页核对其会话并放弃任何待处理的 Stop，而其他标签页没有需要核对的内容。worker 或选出的标签页收到的 401 会被传递出去，因此每个标签页都会返回登录屏幕。
- **代码** - `chat/sharedServerEvents.ts` 选择传输方式，`chat/serverEventsHub.ts` 是连接的所有者以及每个标签页中的接收者，`chat/eventsWorker.ts` 与 `chat/eventsWorkerHost.ts` 是 worker，`chat/serverEvents.ts` 解析流。Vite 将 worker 输出为 `events-worker.js`，与 `app.js` 放在一起，二进制会将其与其余 shell 一起嵌入。测试：`chat/sharedServerEvents.test.ts`，以及 `App.stopQueue.test.tsx` 中的三标签页场景。

## 聊天传输

- 主要传输方式是 `POST /v1/responses`。
- `stream: true` 使用 SSE。

模式选择

- UI 允许用户从 `GET /v1/models` 中选择模式（至少包括 `agent`、`plan` 和 `ask`）。
- 所选模式会作为 `model` 字段在 `POST /v1/responses` 中发送。

SSE 载荷

- 默认 SSE 行流式传输类似 OpenAI 的增量。
- 命名的 SSE 事件
  - `tool_call`
  - `tool_call_update`
  - `plan`
  - `token_usage`（一次已完成模型调用的计数器；当提供商报告时，**`reasoningTokens`** 是其输出中用于推理的部分）
  - `usage_update`（当前模型上下文的 `used` / `size`；压缩后会再次发出）
  - 默认（无 `event:`）：聊天补全分块增量，包括 `delta.content` 和可选的 `delta.reasoning_content`，后者携带模型的推理内容，无论提供商将其放在哪个字段中（`reasoning_content`、`reasoning` 或 `thinking`）

## 长会话

![读取上方页面时的长对话记录顶部，深色主题，1280 px](../../assets/transcript-earlier-loading-dark-1280.png)

*读取上方页面时的长对话记录顶部，深色主题，1280 px*

一个会话可以增长到数千条消息，而 Web UI 过去会在显示任何内容之前读取、映射并渲染全部消息：一个 3306 条消息的会话在桌面端耗时 7.5 秒，并产生 5 秒的任务；在速度减慢六倍的 CPU 上耗时 41 秒，并产生 32 秒的任务，也就是 issue #338 中的旧手机，页面中有 9.4 万个元素。现在 Web UI 在对话上维护一个滑动窗口。

- **先显示最新一页。** 打开对话时会读取 `GET /coddy/sessions/{id}/messages?limit=60`，即约六十条消息组成的页面，从该轮次的提示词开始，并且只读取该页工具调用的预览（`GET .../tool-calls?from=&to=`）。短会话会在这一次读取中完整到达。响应中的 `window` 说明该页所在的位置，行号也从它开始编号，因此 id 和编辑索引与完整读取所给出的一致（[HTTP API](../reference/http-api.md)）。
- **屏幕上是有界切片。** 对话记录渲染其持有的最后若干行，并在每一帧向读者接近的边缘增长几行，因此没有任何一帧会一次性布局很长一段。另一侧超出视图一屏以上的行会在超过约 130 行后被丢弃，但读者正在其中输入的行以及最新若干行除外：当那里有权限提示或问题提示等待回答时，这些最新行会保留其选项和已输入文本，无论读者向上滚动多远；读者所在的行保持不动：窗口会在可见区域上方每次变化后自行校正滚动位置，因为 WebKit 不会锚定滚动。最新一行被渲染时，窗口会跟随新输出；滚动到底部按钮会把留在很上方的窗口带回最新行，并立即落在那里。
- **向上读取更早的页面。** 在已持有内容的顶部，会读取上方页面，即 `?limit=80&before=<the first message held>`，并放到前面。对话记录顶部的一个控件代表上方内容：**显示更早的消息**（供键盘使用的按钮；滚动到它也会执行相同操作）、读取页面时的 **正在加载更早的消息…**，以及读取失败后的 **重试**，除非按下它，否则不会重复。
- **在最新消息处放手。** 一旦读者回到最新消息，且会话中没有正在运行的内容：没有轮次、没有正在发送的回答、没有等待中的提示，向上读取的较早页面就会被丢弃，一个经过多轮增长起来的窗口会从最新页重新开始；在轮次结束时也会如此，无论读者位于最新消息，还是会话在后台运行，因为一个接一个运行轮次的会话很少会长时间空闲。被丢弃的内容远在屏幕上方，而屏幕仍停留在最新消息。
- **重新加载读取窗口，而不是历史。** 轮次结束、权限回答、中继漏掉帧：每一种都会从窗口的第一条消息重新读取实时窗口（`?from=`），较早页面不参与其中。窗口下方的历史被截断：来自另一个界面的回退，由 `session_rewound` 宣布，或被察觉为历史比窗口更短，会从最新页重新开始窗口。
- **编辑会回退到正确的提示词。** 提示词上的编辑控件会标明服务器所知的索引：窗口之前的提示词（`window.turnsBefore`）加上它在所持有提示词中的位置。
- **多个浏览器、标签页和界面。** 每个标签页持有自己的窗口，服务器不保存关于它的状态，因此一个会话可以同时在多个浏览器、标签页和控制台中打开。另一个标签页运行的轮次会像以前一样流式传入当前标签页；深入历史的读者会在它运行时保持位置，并在最新消息处找到它。控制台和 `--remote` 客户端读取历史时不使用窗口，会获得全部历史，一如既往。
- **通过集群中继。** 每次读取都会在查询字符串中携带其页面，中继的挂载会将其原样转发给节点，通过隧道时也是如此（[集群](../operate/swarm.md)）。
- **代价是什么。** 页面内查找只能看到已渲染的行。打开后又滚动到窗口外很远的工具卡片，在回到窗口时会关闭（计划文档会保留其状态）。早于分页读取的服务器会返回全部历史，Web UI 随后将其作为一个页面持有。

在同样的 3306 条消息会话上，于手机视口中以 CPU x6 运行，最新消息现在 2.8 秒内出现在屏幕上，包含页面加载，最长任务耗时 0.9 秒，页面持有约 600 个元素和 6 MB 堆内存；在整个历史中快速上滑时，一半帧保持在 50 ms 以下。**`npm run check:transcript`** 会测量它（见下方 **检查长对话记录**）。
## 对话记录滚动到底部

![输入框上方的滚动到底部按钮，深色主题，1280 px](../../assets/scroll-to-bottom-visible-dark-1280.png)

*输入框上方的滚动到底部按钮，深色主题，1280 px*

在长聊天中向上滚动会让最新消息留在屏幕外，而拖回滚动条是唯一的下行方式。输入框上方的圆形控件只需按一下即可完成。

- **显示条件** - 当滚动视口位于距末尾 `TRANSCRIPT_BOTTOM_THRESHOLD_PX`（**80px**）以内时，对话记录会跟随新输出。按钮恰在该条件不再成立时出现，并在重新成立时消失，因此看到它意味着对话记录在可视区域下方还有内容。它在一个已挂载节点上淡入淡出（进入 **~0.14s**，离开 **0.12s**）；隐藏时具有 `inert`，并退出 Tab 顺序。
- **位置** - 位于输入框自身列内（`.chat-bottom-inner`），贴着其右边缘，在停靠块上方 `10px`；在输入框下方时，它位于展开控件上方，两者相距 10px，并与改进提示词魔棒的竖线居中对齐。每个外壳都使用同一组坐标：绝对定位的桌面停靠区、`1200px` 下方的 `position: fixed` 输入框，以及后台任务面板预留的内嵌区域。
- **跳转** 按距离耗时 **220-460ms**，采用 ease-out 曲线：快速离开，最后几个像素逐渐稳定，而不是戛然而止。它逐帧驱动（`transcriptJumpDurationMs` / `easeTranscriptJump`，位于 `chat/transcriptScrollPosition.ts`），而非交给 `scrollTo({ behavior: "smooth" })`，因此在所有引擎中手感一致。到达后会重新启用跟随，本轮剩余内容又会自动滚动。
- **读者已滚开时的流式传输** - 位置不动，按钮保留，因为距末尾的距离只会增大。轮次中途启动的跳转会在每一帧重新读取末尾，因此它会落到最新消息，而不是按下按钮时对话记录的末尾位置。
- **读者始终优先** - 滚轮、手指或按动滚动条都会让滚动就地停止；如果停在未到末尾处，按钮会立刻回来。
- **仅向下** - 按钮绝不会让聊天向上移动。在手机上，末尾是相对于屏幕键盘上方可见区域来衡量的；当键盘覆盖页面而不是压缩页面时（iOS Safari），输入框和按钮会升到键盘上方；过去按钮会留在键盘下方，并把页面向上推键盘的高度。
- **两个滚动表面** - 宽外壳滚动 `.chat-scroll`，窄外壳滚动文档；同一个模块为两者读取距离和末尾位置，一次读取驱动跟随标志、按钮和跳转。
- **`prefers-reduced-motion: reduce`** 一步将对话记录置于末尾，并去掉按钮的淡入淡出。
- **无障碍名称和工具提示** 均为 `chat.scrollToBottom`；空白首屏从不渲染它。
- **打开另一个会话** 会落到其最新消息，无论上一个会话停在哪里：历史记录选择、指向 `#/s/<id>` 的链接，或在一个聊天中读到很靠上之后按返回按钮，都不再以旧偏移打开下一个会话（issue #342）。会话 id 变化时，进行中的跳转会被取消并重新启用跟随，且发生在新行到达之前。

## 输入框中的长提示词

![停靠的输入框为长提示词在聊天上方展开，深色主题，1280 px](../../assets/composer-expanded-dark-1280.png)

*停靠的输入框为长提示词在聊天上方展开，其控件位于标题下方一步，深色主题，1280 px*

输入框的字段会随文本增长（issue #342）。它从一直以来的高度开始，起始页为五行，聊天中为两行且最小 76px，并逐行增长到 **八行**，或可见视口的 **40%**（取较小者，例如打开键盘的手机），然后滚动。删除文本会使其缩回；窗口变窄、键盘打开和页面缩放都会重新适配。规则是 **`chat/composerHeight.ts`**，由 **`chat/useComposerFieldHeight.ts`** 应用。

- **展开** - 在聊天中，一个圆形控件位于输入框上方 10px，并与改进提示词魔棒的竖线居中对齐，大小与滚动到底部按钮相同；显示时，后者位于同一竖线上、其上方 10px（`data-testid="composer-expand"`，**展开消息输入框** / **收起消息输入框**，`aria-pressed`），让字段占据标题下方整个聊天区域，用于编写或粘贴长提示词；其箭头会展开到四角，展开时再合拢。展开后，标题、控件和卡片彼此相距一步，即从窗口边缘到顶部栏的距离（手机和平板为 10px，桌面为 14px），并且跳转到最新消息的功能会隐藏。
- **展开字段中的选择器** - 技能、提及和命令选择器在字段内部打开，位于字段底部、输入框栏上方，并保持在其高度内；在手机和平板上，无论是否展开，它们都不会使背后内容变暗。
- **屏幕键盘** - 展开的字段结束于键盘上方：在键盘压缩页面的地方（Android），窗口更短；在键盘覆盖页面的地方（iOS Safari），输入框会升到键盘上方并保持在可见区域内，位于标题下方，或位于页面被上推时可见区域顶部下方。在打开键盘时手机太小、容不下字段最小高度的情况下，字段保持最小高度。
- **折回** - 使用同一控件、字段中的 **Escape**（在 Escape 离开编辑之前：折回不会丢失内容）、发送或排队提示词，以及打开另一个聊天。文本保持不变。
- **起始页** 会随文本增长，但没有展开控件：卡片位于页面中央，周围留有空间。

## 引用对话记录

![答案中的选区提供“引用”，以及已发送提示词显示其引用内容，深色主题，1280 px](../../assets/transcript-quote-dark-1280.png)

*答案中的选区提供“引用”；已发送提示词显示其引用的段落，深色主题，1280 px*

在对话记录任意位置选择文本，答案、提示词、工具卡片皆可；选区稳定后，其旁边会出现一个 **引用** 按钮（`data-testid="transcript-quote"`，**`chat/TranscriptQuoteButton.tsx`**）。使用鼠标时，它位于选区上方；在触摸屏上位于下方，因为浏览器自带的选择菜单占据上方空间。它保持在聊天标题和输入框之间，并在选区折叠或滚出视野时离开。聊天标题、输入框以及对话记录之外的任何内容都不提供此功能；只读子智能体对话记录或已归档聊天也不提供，因为它们没有输入框。

- **按下后的操作** - 该段落会以 Markdown 引用行形式进入草稿（`> `，空行使用裸 `>`），接在已有内容之后，两侧各有一个空行，下方留一个空行用于书写；选区被清除，光标移到草稿末尾（字段会在应用可聚焦它的地方获得焦点，因此在手机上不会，因为焦点会打开键盘）。再次引用时，段落会按选取顺序累积（issue #342）。
- **草稿是纯文本** - 引用是草稿的一部分，因此任何保留、排队或恢复草稿的机制也会保留其引用，删除引用就是编辑文本。模型会把它读作 Markdown 引用；文档阅读器的 **询问所选内容** 也以同样方式引用（**`chat/quoteDraft.ts`**）。
- **在对话中** - 已发送提示词会将其引用的行显示为左侧带竖条的引用块，色调比周围文本更淡（`data-testid="user-message-quote"`）。只有行首的标记后跟空格或什么都不跟，才会开启引用，因此 `>=5` 仍是文本。复制和编辑控件仍会原样返回发送时的提示词。

## 输入框主要操作（`#btn-send`）

![输入框字段右上角的改进提示词魔棒](../../assets/composer-improve-prompt-dark-1280.png)

*字段右上角的改进提示词魔棒，草稿有文本时点亮*

上下文环和明细弹出框

- **悬停** 在 **`.composer-context-tip-host`** 上：紧凑工具提示（百分比、输入/输出/总计、最大上下文）不变。
- **单击** 在宽视口中于环旁打开 **`ContextBreakdownPopover`**（**`context-breakdown-menu--portal`**）；在堆叠外壳（**`max-width: 1199px`**）上，它使用与斜杠 / **`@`** 选择器相同的底部抽屉，并覆盖其自己的变暗遮罩（**`context-breakdown-menu--sheet`**，**`slash-sheet-backdrop`**；选择器的背景是透明的）。**Escape** 或 **关闭** 可关闭；关闭后悬停工具提示恢复。
- 图例键映射到 **`contextBreakdown`** 上的 **`GET /coddy/sessions/{id}/stats`**（`systemPrompt`，`toolDefinitions`，`rules`，`skills`，`mcp`，`conversation`）。实时 **`usage_update`** SSE 会立即替换显示的总计（包括在 `/compact` 或自动压缩之后），随后 UI 刷新详细统计。Vitest：**`Composer.test.tsx`**（`click context ring opens breakdown popover`）和 **`consumeComposerSse.order.test.ts`**（`usage_update replaces the displayed current context after compaction`）。

形状和字形

- 该控件位于上下文环的**右侧**（**`.composer-icon`** 在 **`Composer.tsx`** 上）。
- 点击目标是一个**完美圆形**：**宽度**和**高度**相等，**`border-radius: 50%`**，**`box-sizing: border-box`**（当前在 **`styles/composer/_send.scss`** 中为 **42×42px**）。除非视觉规范明确再次更改，否则**不要**为此控件交付圆角方形或超椭圆。
- **播放**（**空闲**，草稿非空）：一个 SVG 三角形，**17x17px**，位于 **`composer-send-glyph`** 内，其路径向右偏离中心以实现视觉平衡。
- **停止**（**会话有活动轮次时，草稿为空**）：实心方块 **`.composer-stop-square`**（**14x14px**，居中于 **42px** 圆内）。仍在右侧的 **`composer-bar-actions`** 中，紧邻上下文环，即使此标签页没有流读取器。
- **排队**（**会话有活动轮次时，草稿或附件存在**）：**播放**字形在强调色填充上回归（**`.composer-run-icon--queue`**，**`data-queue="true"`**）。清空字段和附件会带回**停止**。参见下文 **输入框消息队列**。
- 当文本区域仅含空白**且未附加文件**时为空闲**禁用**状态（**`:disabled`** 在 **`composer-send-play`** 上）；仅附件即可解锁发送（参见 **输入框文件附件（多模态）**）。

行为

- **Enter** 在空闲时提交，在会话有活动轮次时将草稿排队，适用于每台带键盘的设备，包括窄于 **1200px** 的桌面窗口；**`Shift+Enter`** 插入换行。**`Ctrl+Enter`** 或 **`Alt+Enter`** 也插入换行，例外是在仅包含 **`````**, where it opens an empty fenced code block and puts the caret inside its monospace field; arrow keys naturally leave the block through its line boundary. What Enter does follows the input device, not the width: on a **touch-only** device (**`(any-hover: none) and (any-pointer: coarse)`**, a phone) Return inserts a newline and the primary button sends or queues, because a phone keyboard has no Shift+Enter; **`Cmd+Enter`** still sends from an attached keyboard. The textarea's **`enterkeyhint`** says the same to the on-screen keyboard (**`send`**, or **`enter`** on a touch-only device). Enter that confirms an input-method candidate (**`isComposing`**, **`keyCode` 229**) never sends. The rule is **`chat/composerEnter.ts`**. No key the input method is composing with reaches the rest of the composer either (the check opens the textarea's **`onKeyDown`** in **`Composer.tsx`**): the slash, **`@`**, command option and line-range pickers take no row, move no highlight and do not close on it, and Ctrl+Z does not restore the draft from before Improve prompt. A keydown counts as composing when it carries **`isComposing`**, or when it is a **`keyCode` 229** keydown within 100 ms of **`compositionend`** 的行上，即 Safari 在结束组合输入后发送的键。任何其他 229（Android 键盘为普通按键发送）都会像普通按键一样操作选择器，而上面的发送规则仍会拒绝携带它的 Enter。
- **停止生成** 始终发送带 **`{"continue_queue":true}`** 的 **`POST /coddy/sessions/{id}/cancel`**，并立即中止该标签页自己的读取器，因此请求不会等待该读取器持有的连接。失败仍可见且可重试，标签页会重新加入正在运行的轮次。**`queueContinued: true`** 使标签页重新附加到同一生成；否则取消会等待服务器确认空闲。部分助手持久化和对话记录合并遵循 [并行会话和生成取消](#并行会话与生成取消)；队列资格和回退遵循 [消息队列](../features/message-queue.md#轮次边界与-stop)。
- **服务器从未接收的发送** 会回到输入框：文本（位于此后输入的任何内容之前）和附加文件。这涵盖按状态拒绝（**409** 忙、**4xx** 或 **5xx**）、传输途中失败的请求（上传中断、连接被拒绝）以及浏览器无法读取的附件（选取后磁盘上的文件被移动或更改、云照片不在设备上），在后一种情况下不会发送任何内容。乐观气泡离开对话记录，红色通知说明原因，括号中给出浏览器的原因或服务器的消息（`Failed to fetch`，`request body too large`），因此从未到达服务器、因而没有留下日志的失败仍能与服务器停止区分开来。传输途中失败的请求之后不会再次读取对话记录，因为该读取会抹掉通知；如果服务器最终确实接收了该轮次，活动刷新会附加到它，并在其结束时重新加载对话记录。规则是 **`App.tsx`** 中的 **`streamResponses`**。
- **改进提示词**：紧凑的 **24×24px** 魔棒按钮（**`data-testid="composer-enhance-btn"`**）位于**字段右上角**，在面板接缝下方 8px、距卡片边缘 12px（两者看起来一致），因此字段及其占位符从卡片顶部开始，第一行与魔棒齐平；文本会避开它，上方面板中 git 的计数结束于其右边缘。它不在下方输入框栏中。它有 `title` 和无障碍名称 **`Improve prompt`**，草稿为空时以及请求或生成活动期间禁用，调用 **`POST /coddy/enhance-prompt`**，并且仅在成功时替换草稿。**Ctrl+Z** / **⌘Z** 恢复改进前的草稿；失败时保持不变并显示内联错误。

回归

- 自动化 UI 检查（**Playwright MCP** 或 **`@playwright/test`**）可以断言 **`#btn-send`** **`offsetWidth`** **≈** **`offsetHeight`** 以及计算出的 **`border-radius`** **≥ 一半** **`min(width,height)`**（在亚像素容差内）。

## 输入框消息队列

智能体工作时，输入框保持可用：后续消息可以引导下一个 ReAct 步骤，或在当前答案之后等待单独的提示词。完整行为：[消息队列](../features/message-queue.md)。

- **排队** - 第一条后续消息会询问 **Enter** 应使用哪种模式，并将其保存到 `agent.queue_mode`。**Tab** 使用另一种模式。主要控件使用 Enter 模式。POST 正文携带 **`text`**、**`mode`** 和可选的 **`inline_files`**；输入框会清空文本和附件。在第一个答案之前通过 **Tab** 发送的消息，在模式选定后仍会以另一种模式进入。**Tab** 属于打开的斜杠或提及选择器，即使是空选择器。**409 no_active_turn** 会回退为普通提示词；其他拒绝会恢复草稿和文件。
- **列表** — 排队行在输入框上方渲染为 **`.composer-queue-item`**，显示模式；对于带图像的消息，还显示回形针及其数量（队列只命名图像，绝不携带其字节）。单击模式可通过 **PATCH** 切换。带边框的 **×** 通过 **DELETE** 移除该行，并将其文本以及来自答案 **`inline_files`** 的图像返回输入框；**404** 表示智能体已先读取它。完全取消后，保留的轮次后行以及已由被接受的继续请求预留的未读批次仍然可见。
- **停止但有排队消息** - **停止生成** 会为同一服务器进程中运行且符合条件的普通配置提示词请求继续。一旦当前运行展开，且其部分助手内容和工具结果稳定，服务器会按原始 FIFO 顺序将所有预留消息作为单独的用户行追加，包括重复项和保留的图像，无论队列模式如何。新的子运行会在相同的准入和生成下回答它们，只有一个写入者，没有合成或合并的用户提示词。空队列或不支持的路径会回退为完全取消。在预留批次被完整读取之前再次停止会取消轮次并保留该未读批次。错误、外部取消和关闭不会自动恢复它。后续消息遵循正常队列规则，等待状态在进程重启后仍会丢失。完整资格和默认客户端行为见 [消息队列](../features/message-queue.md#轮次边界与-stop)。
- **打开并重新连接** - UI 会在读取所选会话的活动快照时一并读取 **`GET /coddy/sessions/{id}/queue`**。现有等待消息会出现，而无需另一次队列变更、活动轮次读取器，或 **历史** 中的匹配行。
- **保持同步** - 同一 **`coddy serve`** 的客户端会接收带有整个队列的 **`event: message_queue`**，以及一个 **`version`**，经由轮次流和 **`GET /coddy/events`** 传递。HTTP 响应携带相同快照。**`QueueDeliveryOrder`** 保留最新版本并围栏重启恢复。被消费的引导消息会以 **`event: user_message`** 到达智能体读取它的位置，轮次后消息也会以同样方式在其自身提示词开始的位置到达，因此第二个答案绝不会从第一个答案继续运行。
- **范围** - 这些是停止和队列控件，不是共享会话总线。现有对话记录中继以及权限/问题所有权保持不变。共享会话目录的独立本地控制台或 ACP 进程共享取消标记，但不共享其内存队列或答案通道。
- **占位符** — 生成时，字段显示 **`composer.placeholderQueue`**，而不是空闲占位符。
- **附件随排队消息一起传递。** 图像会到达多模态模型；实时气泡会立即显示图片，从会话资产（**`chat/queuedUserMessage.ts`**）打开它，轮次后读取的对话记录会带来其缩略图。
- Vitest：**`composerQueue.test.tsx`**，**`queuedUserMessage.test.ts`**，**`App.stopQueue.test.tsx`**（取回消息、文本和图像）。

功能回归检查清单：

| 场景 | 预期 |
| --- | --- |
| 打开一个正在运行但不在当前历史搜索或页面中的会话 | 活动和队列读取会恢复停止/队列控件以及等待消息。 |
| 在已有等待消息时重新连接，或丢失本地轮次读取器 | 队列会自行填充，无需新的变更；控件跟随服务器活动，而不是读取器生命周期。 |
| 让 `/cancel` 失败，然后重试 | 错误保持可见，标签页通过中继重新加入正在运行的轮次，停止仍然可用。当 `queueContinued` 为 true 时，成功重试会重新附加到同一生成；否则服务器完成会确认空闲。 |
| 在一个正在运行的会话上打开三个标签页，队列为空（到主机的六个 HTTP/1.1 连接），在任意一个中按停止 | 取消请求会到达服务器，轮次在每个标签页中停止，且没有 "Could not stop generation" 错误。 |
| 通过普通 http 打开同一服务器的六个空闲标签页，然后从任意一个发送提示词 | 提示词会立即发送：标签页共享一个事件连接，在一个标签页中启动的轮次会在查看该会话的每个标签页中显示停止。 |
| 在一次 SSE 队列投递越过该读取后，再投递队列 GET 响应 | 已投递列表获胜；已移除或已消费的消息不会重新出现。重放的 `turn_started` 事件会保留列表和版本。 |
| 重启服务器，然后通过新的 GET 恢复队列 | 读取期间没有投递时，较低版本（包括 `0`）会替换陈旧状态；来自较早恢复纪元的变更响应以及自身/中继流队列帧会被忽略。 |
| 在此标签页的下一个 POST 待处理或已准入时，投递旧的 `turn_ended` | UI 会通过新的 REST 确认活动，而不是将新轮次标记为空闲；活动和队列填充会在准入期间等待。 |
| 在活动或队列读取待处理时切换会话 | 迟到的响应不会改变新会话的控件或队列。 |
| 在符合条件的配置提示词运行期间，队列中有混合模式消息、重复项和图像时按停止 | 部分答案和工具结果会先稳定，然后按原始 FIFO 顺序出现单独的用户行；同一已准入轮次通过中继回答它们，无需另一个客户端 POST 或 `turn_started`。 |
| 在被接受的批次被读取之前再次按停止 | 完全取消获胜，未读的已接受批次会被保留，不会自动继续。 |
| 在已保存计划运行、直接完成、权限恢复或监督命令期间按停止 | 服务器使用完全取消；它不会确认队列继续。 |
## 输入框文件附件（多模态）

![已发送的图片从会话包中恢复为缩略图](../../assets/composer-attachment-thumbnail-dark-1280.png)

*已发送的图片从会话包中恢复为缩略图*

- 回形针按钮（**`data-testid="composer-file-input"`** 隐藏 `<input type="file">` 由可见图标按钮触发）仅当活动模型具有 **`multimodal: true`**（来自 **`GET /v1/models`**）时出现在输入框中。该标志由 YAML 配置中的 **`models[].multimodal`** 派生，并经由 **`ModelInfo.multimodal`** → **`llmModelMultimodal`**（位于 **`App.tsx`** 中）→ **`Composer`** prop 传播。
- 除回形针选择器外，文件还通过另外两条入口路径进入 **`attachedFiles`**，两者都以 **`llmModelMultimodal`** 为前提：
  - **`textarea#composer`** 中的**剪贴板粘贴**：图片项（`kind === "file"`、`image/*`）会被附加，并取消默认粘贴；纯文本粘贴不受影响。粘贴的图片会获得确定性名称 **`pasted-<n>.<ext>`**（浏览器会将每个剪贴板图片命名为 `image.png`）。粘贴的一张图片会首先在**标注编辑器**中打开（**`ui/markup/ImageMarkupDialog.tsx`**）：**完成**会将其作为 PNG 附加，**取消**或 **Escape** 不会附加任何内容（[粘贴并标注图片](../features/picture-markup.md#在-web-界面中)）。同时粘贴多张图片，以及浏览器无法在画布上绘制时，会直接附加。
  - **拖放**到 **`.composer-card`**：拖入的文件像选择器选择一样附加；文件拖过卡片上方时，卡片会显示 **`.composer-card--dragover`** 放置目标提示。
- 当模型**不是**多模态时，粘贴/拖放会被拒绝，并显示临时行内提示 **`.composer-attach-hint`**（`role="status"`，约 4 秒后自动清除），而不会附加文件。
- `image/*` 附件是**预览卡片**，而非标签：图片填满 **128×96** 卡片（输入框侧为 **`.composer-attachment-card`**，已发送气泡中为 **`.msg-user-file-card`**，`object-fit: cover`，12px 圆角），以便操作者看到附加了什么。悬停会高亮卡片，光标为 **`zoom-in`**，点击会在共享查看器中打开图片（**`ui/components/ImageLightbox.tsx`**：缩放级别、**+ / - / 0**、拖动平移、滚轮和双指缩放、**Esc**、传送到 `document.body`）。非图片文件和锁定的编辑模式标签保留带文件名的图标标签。在卡片或查看器中右键单击图片，会提供 **"复制图片"** 和 **"保存图片"**（[Workspace files](#工作区文件) 描述了该菜单）。
  - 在输入框中，卡片的**移除**控件（**`composer.removeAttachment`**）位于右上角，悬停时淡入；它保留在 Tab 顺序中，并在 **`:focus-visible`** 上可见，移除操作绝不会打开查看器。查看器打开的图片是本地对象 URL - 文件本身，全尺寸。左上角的**铅笔**（**`composer.markupAttachment`**）会在标注编辑器中打开图片；完成会用标注后的 PNG 替换附件，取消则保持原样。
  - **已发送的用户气泡**会先在卡片中渲染乐观的 **`previewUrl`** blob，然后在持久化后替换为后端的 **`files[].preview_url`**；此时会撤销 blob URL。卡片打开的是 **`files[].url`**，即来自会话包的原始字节；在该字段存在之前发送的消息会打开其 **`preview_url`**，而不会失去点击。重新加载对话框会通过 **`GET /coddy/sessions/{id}/messages`** 恢复两者。
- 当所选模型为多模态时，**仅附件发送**有效：即使草稿为空，**发送**（按钮或 **Enter**）也会因有附件而解锁，并提交 **`onSend("", files)`**；服务器接受空字符串 `input` 与 `inline_files` 一起。如果用户切换到非多模态模型，现有标签仍会显示并带有 **`.composer-attachment-chip--disabled`**，仅附件发送变为禁用，文本发送会省略并保留这些文件。
- HTTP 处理程序会针对生效的 YAML 模型独立过滤 **`inline_files`**。这样，当 **`multimodal`** 为 false 时，自定义或过期的客户端无法转发或持久化文件。
- 附加文件保存在 **`Composer`** 上的 **`attachedFiles: File[]`** 状态中。预览标签会显示在输入框上方，展示文件名和类型图标（图片则为缩略图）。每个文件在附加时都会被复制到内存中（**`chat/attachmentCopy.ts`**）：拖入或粘贴的文件可能只是磁盘上的路径，而 Spectacle 等截图工具会在下次截图时删除其临时文件，这曾导致较早的图片无法显示或发送。无法读取的文件会被移除，并给出指明它的提示，其他文件则保留。
- 发送时，**`App.tsx`** 通过 **`FileReader`** 将每个文件读取为 data URL，并将 **`inline_files: [{name, data_url}]`** 包含在 **`POST /v1/responses`** 请求体中。
- **Agent / plan / ask 轮次**：当生效模型为多模态时，服务器将每个文件写入 **`~/.coddy/sessions/<id>/assets/`**（权限 **`0o444`**），并向用户消息注入 **`<coddy_session_assets>`** XML 块，以便智能体能够 **`read`** 或 **`cp`** 这些路径。文本文件也会原样写给模型，最多到其前 64 KB；既不是图片也不是文本的文件（PDF、压缩包）只会被命名，绝不会以文本形式写出。重复的资源名会获得 **`_1`**、**`_2`** 后缀（参见 `internal/session/assets.go` **`SavePartsToAssets`**）。
- **直接 YAML 模型轮次**：对于多模态模型，每个文件会先保存到会话资源目录下，然后成为内联发送给提供商的 **`image_url`** 内容部分。
- 对于任何模式，可解码的 PNG/JPEG/GIF 上传都会在 **`assets/thumbnails/`** 中获得一个只读 PNG 预览，限制为 **160 px**。**`GET /coddy/sessions/{id}/messages`** 返回 **`files`** 元数据、**`preview_url`** 和 **`url`**；**`GET /coddy/sessions/{id}/assets/{name}/thumbnail`** 提供生成的预览，**`GET /coddy/sessions/{id}/assets/{name}`** 提供原始字节，但**仅当这些字节嗅探为图片时** - 绝不由文件名决定，因此没有其他附件会离开会话包。用户气泡通过 **`stripCoddyAttachmentsForUserDisplay`** 剥离 XML 标注，并仅将 **`parseSessionAssetFiles`** 用作旧版回退。
- 在设置中保存 **`PUT /coddy/config`** 后，**`App.tsx`** 会递增 **`configEpoch`** → 重新获取 **`/v1/models`**，从而无需重新加载页面即可让附件按钮出现或消失。同一计数器也会被 **`GET /coddy/events`** 上的 **`event: config_reloaded`** 递增，因此在此页面之外所做的更改具有相同效果（参见 **实时配置重新加载**）。

| 用例 | 预期 | 自动化检查 |
|------|----------|-----------------|
| FA1 | 回形针仅在 `llmModelMultimodal` 为 true 时可见 | `Composer.test.tsx` |
| FA2 | 发送后文件标签在用户气泡中渲染 | `stripCoddyAttachments.test.ts` |
| FA3 | 标签通过 `parseSessionAssetFiles` 在重新加载后保留 | `stripCoddyAttachments.test.ts` |
| FA4 | 粘贴图片会将其作为 `pasted-<n>.<ext>` 标签附加（仅多模态） | `Composer.test.tsx` |
| FA5 | 使用非多模态模型粘贴/拖放会显示 `composer-attach-hint` 且不附加任何内容 | `Composer.test.tsx` |
| FA6 | 将文件拖放到 `.composer-card` 会附加它们并切换 `composer-card--dragover` | `Composer.test.tsx` |
| FA7 | `image/*` 附件会渲染 `composer-attachment-card` 预览；非图片标签保留图标 | `Composer.test.tsx` |
| FA8 | 仅附件即可解锁 Send/Enter 并提交 `onSend("", files)` | `Composer.test.tsx` |
| FA9 | 已发送气泡从 `previewUrl` 渲染 `msg-user-file-thumb`（仅图片）；仅元数据条目保留图标 | `UserMessage.test.tsx`、`optimisticUserFiles.test.ts` |
| FA10 | 切换到非多模态会保持标签禁用，且文本发送会省略它们 | `Composer.test.tsx` |
| FA11 | 后端缩略图元数据替换乐观 blob，并在重新加载后恢复 | `sessionMessageFiles.test.ts`、`transcriptServerSnapshot.test.ts`、`server_test.go` |
| FA12 | 预览卡片会在输入框和已发送气泡中放大打开图片；已发送气泡打开 `url` 并回退到 `preview_url` | `Composer.test.tsx`、`UserMessage.test.tsx` |
| FA13 | 移除输入框预览卡片会丢弃附件且不打开任何内容 | `Composer.test.tsx` |
| FA14 | `GET .../assets/{name}` 提供原始图片字节，并拒绝任何嗅探结果不是图片的内容 | `coddy_session_asset_test.go` |

## 输入框斜杠技能与镜像光标

权威叙述和视觉令牌位于 **`DESIGN.md`**（斜杠选择器、镜像契约、验证表）。本节是用于回归的功能契约。

接线与草稿

- **`textarea#composer`** 仅保存**纯文本**。被调用的技能显示为 **`/<name>`** 令牌（选择器选择后带一个空格）。UI **不得**在草稿中持久化 **`[/<name>](coddy-skill:<name>)`**。
- **`POST /v1/responses`** 上的首个用户轮次携带与输入框值相同的纯斜杠令牌（请求体中技能不进行客户端 Markdown 注入）。

选择器与分段

- **Commands** 组列出 **`GET /coddy/commands`** 中的内置命令及其参数提示（**`.slash-row-hint`**）。选择 **`/model`**、**`/reasoning`** 或 **`/permissions`** 会打开对应的输入框选择器，而不是插入文本；选择 **`/agent`**、**`/plan`** 或 **`/ask`** 会切换模式；带值输入的设置命令会作为提示词文本发送，并由服务器应用。
- 在输入框中输入 **`/mcp`** 会打开 Settings → MCP servers，而不发送轮次；命令后的词会被忽略，与控制台一致，而仅以它开头的词（`/mcpx`）是普通提示词。该命令在本地 Commands 组中显示在 `/docs` 旁边；它可从主页输入框和会话中使用。
- 菜单可见性和 **`prefix`** 派生自 **`external/ui/src/ui/skills/draftSlash.ts`** 中的 **`slashMenuDraftAtCaret`**（位于行首或 **`/`** 前有空白，可选后缀，不在围栏或块引用内）。
- 从外部清空的草稿 - 通过 **Send** 发送、在轮次中排队、作为浏览器命令运行 - 会随之关闭斜杠、**`@`** 和选项菜单：textarea 不会为其父级设置的值触发 change 事件，因此输入框会自行监视草稿（**`Composer.test.tsx`**、**`features/web_ui_menus.feature`**）。
- 镜像高亮在 **`external/ui/src/ui/skills/segmentComposerSlashSpans.ts`** 中使用 **`segmentComposerSlashSpans`**（支持行中的 **`/`**；**`x/foo`** 不是命令令牌）。

镜像与光标对齐

- 非空草稿：textarea 文本绘制为**透明**；**`.composer-mirror-inner`** 显示可见行，包括 **`.composer-skill-chip-inline`**（**`data-testid="composer-skill-chip"`**）。
- 输入框标签**不得**使用会改变行内宽度的水平 **padding**、**margin** 或 **border**。使用 **`box-shadow`** 作为轮廓。**`font-family`**、**`font-size`**、**`line-height`**、**`font-weight`**、标签上的 **`letter-spacing`** 和 **`#composer`** 必须匹配，以便光标对齐。样式表在 textarea 和镜像上均预留滚动条槽（**`scrollbar-gutter: stable`**），镜像随字段自身滚动并采用其高度，并在窗口大小变化和屏幕密度变化时重新对齐（**`DESIGN.md`**，*输入框镜像与光标同步*）。

对话记录与输入框

- **`user_message`** 气泡仅渲染**纯文本**（**`msg-user-body`**、**`white-space: pre-wrap`**）。没有 Markdown 管道；已知的 `/skill` 令牌和工作区 `@mention` 令牌是紧凑的复制控件，点击时会复制其字面源。诸如 **`/path/to`** 的斜杠令牌和 YAML 块完全保持持久化时的样子，保留换行。
- 输入框镜像标签（**`composer-skill-chip`**）**仅**在编辑 **`#composer`** 时应用，不在对话记录中应用。
- 持久化的用户轮次可能以 **`coddy_attachment`** XML 形式携带已水合附件，包含 **`path`**、**`name`** 和 CDATA 文件正文（**`internal/agent`**）。**`stripCoddyAttachmentsForUserDisplay`** 会将每个 XML 块替换为紧凑的 **`@path`**，**仅当**该路径尚未作为 **`@`** 提及出现在周围文本中时（**避免重复**，因为持久化轮次已经在用户文本中重复了 **`@`**，再加上已水合块）。

验证用例

| ID | 预期 | 主要自动化检查 |
| --- | --- | --- |
| UC1 | **`asdfasf /find-skills asdfasdf`** 一个标签，普通 **`textarea.value`** | **`external/ui/src/ui/chat/Composer.test.tsx`**（`composer highlights plain slash token as chip while editing`） |
| UC2 | 空白后行中菜单打开 | **`draftSlash.test.ts`**（`slashMenuDraftAtCaret works after whitespace mid-line`） |
| UC3 | **`x/foo`** 对 **`/foo`** 没有标签 | **`segmentComposerSlashSpans.test.ts`**（`segmentComposerSlashSpans skips letter before slash`） |
| UC4 | 行首 **`/foo`** 标签 | **`segmentComposerSlashSpans.test.ts`**（`segmentComposerSlashSpans line start slash`） |
| UC5 | 旧版粘贴上的 **`stripCoddySkillMarkdownLinks`** | **`segmentComposerSlashSpans.test.ts`**（`stripCoddySkillMarkdownLinks restores plain slash token`） |
| UC6 | 用户气泡保持 **`hi /demo there`** 为纯文本（无 **`coddy-skill-span`**） | **`UserMessage.test.tsx`** |
| UC7 | 多行 YAML / 路径在 **`user-message-body`** 中保持 **`\\n`** 布局 | **`UserMessage.test.tsx`** |
| UC7b | 仅显示 **`slugSlashes`**（普通 **`/`** 与旧版混合） | **`segmentComposerSlashSpans.test.ts`**（`slugSlashesForUserBubbleMarkdown …`；仅输入框/旧版，不在对话记录中） |
| UC8 | 实时 **`coddy serve`**：**`fontFamily`** 与 **`#composer`** 的一致性标签，填充后行尾的光标 **`selectionStart === value.length`** | **Playwright MCP** **`browser_evaluate`** 在 **`make build TAGS="http ui"`** 之后 |
| UC9 | 用户气泡隐藏 **`coddy_attachment`** 正文，仅显示 **`@path`** | **`UserMessage.test.tsx`**、**`stripCoddyAttachments.test.ts`** |

## 输入框命令选项

一旦 **`/compact`** 打开草稿，输入框就会补全该命令所需的内容。其后的两个短横线（**`/compact --`**）会提供选项 **`--model`** 和 **`--reasoning`**（单独的 **`-`** 不提供任何选项，因为指令可以是一个列表）；在 **`--model `** 或 **`--model=`**（或短形式 **`-m`**）之后，列表包含已配置的模型（输入框模型选择器提供的 id，**`props.llmModels`**），并随着 id 的输入通过不区分大小写的子串匹配收窄，这是服务器解析名称所用规则中最宽泛的一条（完整 id 或不带提供商的模型名优先胜出，参见[上下文压缩](../features/compaction.md#compact-命令)）。**ArrowDown** / **ArrowUp** 移动高亮，**Enter** 和 **Tab** 将该行放入草稿并在其后加一个空格，**Escape** 关闭列表并保持草稿不变。选择 **`--model`** 会立即打开模型。

在 **`--reasoning `**（或 **`-r`**）之后，列表包含 **`default`** 以及草稿中 **`--model`** 所指模型的级别。**`/goal`** 接受同样的两个选项：**`--model`**，从已配置模型补全；以及 **`--reasoning`**，从 **`default`** 和草稿中 **`--model`** 所指模型的级别补全（按服务器的方式解析：精确 id，否则是包含它的那个 id），或者当草稿未指定模型时，从会话的模型补全（**`commandReasoningChoices`**）。级别来自 **`GET /v1/models`**（每一行的 **`reasoning_levels`**，**`llmReasoningLevelsByModel`**）。参见[会话目标与监督器](../features/session-supervisor.md#命令)。

![在输入框下输入 /compact --model 后的模型列表](../../assets/compact-model-picker-open-dark-1280.png)

*输入框正在补全 `--model` 的值（Dark，1280 px）。*

![在输入框下输入 /compact -m luna -r 后的推理级别](../../assets/compact-reasoning-picker-open-dark-1280.png)

*输入框正在用模型 `-m` 命名的级别补全 `-r`（Dark，1280 px）。*

- 该列表是选择器外壳的第三面（**`.slash-menu`**，堆叠外壳上的底部面板），**`data-testid="command-arg-menu"`**，行 **`command-arg-row-<id>`**；它不需要请求。
- 可见性、被替换范围和已输入前缀来自 **`external/ui/src/ui/skills/draftCommandArg.ts`** 中的 **`commandArgDraftAtCaret`**，它镜像 **`parseCompactCommand`**（**`internal/agent/compact.go`**）和 **`ParseGoalCommand`**（**`internal/session/goal_command.go`**）：命令打开草稿，选项优先出现，第一个不是选项的词会开始指令或目标，在那里不进行补全。单独的 **`/compact `** 或 **`/goal `** 不会打开任何内容，因此 **Enter** 仍会发送命令。
- 测试：**`draftCommandArg.test.ts`**、**`Composer.commandArg.test.tsx`**。

## 输入框 **`@`** 提及

- **`textarea#composer`** 保留纯 **`input`**，包括每个字面 **`@`** 提及，并且 **`POST /v1/responses`** 按输入原样发送该文本：服务器在发送消息时解析提及（**`internal/session/mentions.go`**，语法在 **`internal/mention`** 中），与控制台、ACP 编辑器和 Telegram 机器人使用的是同一个解析器。输入框不再从草稿派生 **`attachments`**；**`extractAtFileAttachments`**（**`external/ui/src/ui/skills/draftAt.ts`**）仅向最近选择提供数据。提及附加的内容及其限制：[提及](../features/mentions.md)。
- **`@`** 菜单会询问 **`GET /coddy/mentions`**（**`q`** = **`@`** 之后的文本，**`limit=50`**；选择器打开后的第一个查询会加上 **`refresh=1`**，因此刚刚写入的文件也会被提供）。行是 **`MentionRow`**s（**`external/ui/src/ui/skills/mentionRows.ts`**）：**`kind`**（**`file`**、**`directory`**、**`session`**、**`rule`**、**`agent`**、**`plan`**、**`doc`**、**`scheme`**），**`insert`**（替换 **`@`** 加上查询的文本），**`label`**、**`detail`** 和 **`continue`**。每一行都以种类标签开头（**`.mention-kind`**，元种类使用强调色）；**`scheme`** 行是 **`@session:`**、**`@rule:`**、**`@agent:`** 或 **`@coddy:`**，并将搜索收窄到该种类。
- 选择一行会将 **`@`** 加上查询替换为 **`insert`**。**`continue`** 行（文件夹、方案提示、**`@rule:`** 提供的文件夹）不会添加空格，并会在其新查询上保持选择器打开 - **`@src/`** 列出 **`src/`** 包含的内容；任何其他行都会以空格结束提及，当路径包含空格时使用引号（**`@"my notes.md"`**）。带引号的文件夹（**`@"my notes/`**）会在光标前闭合其引号，因此即使后面没有文件，草稿也会命名该文件夹，下一次带引号的选择会接管该引号（**`mentionRows.ts`** 中的 **`applyMentionRow`**）。同一草稿的第二个答案 - 最近选择之后服务器返回的行、索引构建期间的重试 - 会保留箭头移动到的行。**`Composer`** 会在完成选择后推迟两个 **`updatePickerMenus`** tick，以免下拉菜单在尾随空格上立即重新打开。
- **ArrowDown** / **ArrowUp** 移动高亮（**`is-active`**、**`aria-selected`**，滚动到可见）；**Enter** 和 **Tab** 取用高亮行。选择器在轮次运行时也可用，因此排队的后续消息也可以提及文件。
- 以 **`/`**、**`~`**、**`./`** 或 **`../`** 开头的查询会在服务器上浏览该文件夹，可在磁盘上的任意位置；无方案查询会对会话工作区进行排名（先文件名，再路径段，再按顺序的字母），并合并名称匹配的规则、子智能体和计划。当服务器匹配到的数量超过列表可容纳时，**`.mention-more`**（**`data-testid="mention-more"`**）会在标题行显示 **`50 of 1204, type to narrow`**，无论列表滚动多远，该标题行都会留在屏幕上；当工作区的第一个索引仍在构建时，选择器每 400 ms 再次询问，最多五次。
- 空的 **`@`** 前缀（光标紧跟在 **`@`** 后）会先显示来自 **`localStorage`** 的最近选择（**`workspaceAtRecents`**），以 **`sessionId`** 为键（或在第一个分配 id 之前为 **`__no_session__`**），然后显示服务器对空查询的答案：三个方案提示和工作区顶部。最近条目来自文件和文件夹选择，以及成功配置文件发送时的 **`extractAtFileAttachments`**（当客户端生成或服务器轮换 **`X-Coddy-Session-ID`** 时，**`migrateWorkspaceAtRecents`** 会合并）。
- 草稿词法分析器（**`atMenuDraftAtCaret`**）会在行首或空白、左括号或引号之后的 **`@`** 上打开菜单；它接受 **`~`** 和 **`+`**、带引号前缀（**`@"my no`**，允许空格直到引号闭合）以及紧跟在 **`session`**、**`rule`** 或 **`agent`** 之后的 **`:`**。围栏代码块、行内代码段和 Markdown 块引用行会抑制它，与 **`draftSlash`** 保持一致（**`inMarkdownFenceBeforeCaret`**、**`blockquoteLine`**）。
- 镜像 **`@`** 样式使用 **`segmentComposerMirrorSpans`**（**`composer-at-chip-inline`**、**`data-testid="composer-at-chip"`**）。**`listAtPathSpans`**（**`draftAt.ts`**，基于 **`parseMentions`**，是 **`internal/mention`** 的 **`Parse`** 的孪生）会找到令牌 - 路径、文件夹、**`@session:<id>`**、**`@rule:<name>`**、**`@agent:<name>`**、带引号路径、网页 - 并且只有当服务器表示发送时会附加它时，令牌才会成为标签：输入停止后 150 ms，输入框会将草稿发送到 **`POST /coddy/mentions/check`**，它会以空跑方式运行解析器（不读取、不列出、不获取任何内容），并逐个令牌回答可解析的部分（**`typed`**）及其命名的内容。因此 **`npm install @google/genai`** 中的 **`@google/genai`** 保持为文本，**`compare @src/a.go b.go`** 仅将 **`@src/a.go`** 标记为标签，尚未得到回答的令牌保持为文本；在选择器中选择的行会立即成为标签。光标处的草稿在选择器于其上打开时保留其标签；光标之后仍在草稿内的文本会留在活动令牌上，直到 **`atMenuDraftAtCaret`** 词法分析器跳出。**`draftAt.test.ts`** 读取 **`internal/mention/testdata/grammar_cases.json`**，即 Go 语法测试所针对的用例。
- 零匹配的查询会让选择器保持打开（**`Nothing matches`**），而不是折叠菜单（**`composer-at-chip-inline`** 会隐藏 **`atNoMatch`**，与过期过滤器相同的 **`atIdx`**、**`prefix`**）。
- 堆叠外壳视口（**`(max-width: 1199px)`**）会将提及和斜杠选择器渲染为 **`slash-menu--sheet`**，以便面板在手机上可用，位于透明背景（**`slash-sheet-backdrop--clear`**）之上，点击外部会关闭它，且不会使任何内容变暗。
- 用户气泡会折叠消息发送时携带的每个 **`<coddy_attachment>`**（**`stripCoddyAttachmentsForUserDisplay`**，是 **`mention.ForDisplay`** 的孪生）：回到带来它的提及（**`mention`** 属性，否则 **`path`**），或者当文本已经带有该提及时回到空；**`/skill`** 的正文和提及路径拉入的规则不会显示任何内容。扫描会遍历 CDATA 节，因此包含 **`</coddy_attachment>`** 的文件不会将其尾部泄漏到气泡中。

### 行范围（**`@path:N-M`**）

- 提及可以将文件收窄到**从 1 开始、包含两端**的行范围：**`@Dockerfile:21-31`**，或者编辑器与代码托管平台所写的形式：**`#L21-31`**、**`#L21-L31`**、**`#L21`**、**`#21-31`**。**`listAtPathSpans`** 会吸收后缀，因此镜像会将整个令牌标记为一个 **`composer-at-chip-inline`**；范围必须结束令牌（**`:21-31x`** 保持为普通文本）并且 **`1 <= start <= end`**。**`internal/mention/grammar.go`** 在服务器端携带相同语法，两个测试套件共享其字面量（**`internal/mention/testdata/grammar_cases.json`**）。
- 只有这些行会到达模型。范围作为 **`#L<start>-<end>`** 片段附着在 **`acp.Resource.URI`** 上（**`internal/session/promptfiles.go`** 中的 **`lineRangeURI`** / **`sliceLines`**），并且 **`resourceAttachmentXML`**（**`internal/agent/mentions.go`**，基于 **`mention.Attachment`**）会将其转换为 **`<coddy_attachment path="..." name="..." lines="21-31">`**。超过最后一行的结束位置会被钳制。文件无法满足的范围（为零、倒置或起始位置超过最后一行）绝不会被扩大到整个文件：显式的 **`attachments[]`** 范围或带有此类 **`#L`** 片段的客户端 **`resource`** 会被拒绝（**`ErrLineRange`**，HTTP **400**），而输入到提示词文本中的范围会附加一条说明，指出文件有多少行。**`lines`** 标签只会为确实是该切片的正文写入；**`source.literal`** 或字节偏移（**`source.start`** / **`end`**）正文会不带它传输。选择器预览是面板打开时拍摄的快照；到达模型的行会在发送时从磁盘读取，切换会话会丢弃预览。**`stripCoddyAttachmentsForUserDisplay`** 会将此类块折叠回 **`@path:N-M`**；普通提及绝不会覆盖同一路径的范围提及，反之亦然。
- 在路径后输入 **`:`** 会自行关闭提及选择器（**`:`** 在方案提示之外不是 **`MENU_PATH_CHAR`**），并在其位置打开**行范围选择器**：**`external/ui/src/ui/skills/draftAtRange.ts`** 中的 **`atRangeDraftAtCaret`** / **`replaceAtRangeSuffix`** / **`highlightedRange`**，面板 **`data-testid="at-range-picker"`** 通过与 **`@`** 菜单相同的 portal / **`slash-menu--sheet`** 外壳渲染。它会预览文件（**`GET /coddy/workspace/file`**，每个路径一次获取，仅限工作区文件：在绝对路径后输入的范围仍会到达模型，但没有预览），并在输入数字时高亮 **`at-range-line--sel`**；只有起始位置而没有结束位置时，会高亮那一行。
- 输入框文本仍是唯一输入 - 没有数字字段。在桌面上，行是按钮：**`mousedown`** 锚定范围，在行上拖动会扩展范围，每一步都会通过 **`replaceAtRangeSuffix`** 重写后缀（**`preventDefault`** 会将焦点保留在 textarea 中）。在 **`isMobileShell`** 上，行会渲染为普通 **`div`**s，没有指针处理程序 - 手机没有可用于拖动的鼠标，因此范围是键入的。
- 无法解析的路径会让面板保持关闭，因此散文中的 **`@user:1-2`** 绝不会打开空面板；已确定的路径会被记住，因此下一个数字不会重新获取任何内容。**`Escape`** 会关闭面板，并在草稿继续之前对该提及抑制它；**`Enter`** 保持原样并且仍会发送。

| 用例 | 预期 | 自动化检查 |
| --- | --- | --- |
| AR1 | **`@f.go:21-31`** 作为一个令牌成为标签，并且只附加那些行 | **`draftAt.test.ts`**、**`internal/mention/mention_test.go`**（共享 **`grammar_cases.json`**）、**`features/at_line_range_mention.feature`** |
| AR2 | **`:21`**、**`:21-31x`**、**`:31-21`**、**`:0-5`** 不是范围 | **`draftAt.test.ts`**、**`internal/mention/mention_test.go`** |
| AR3 | 冒号打开选择器；数字移动高亮 | **`Composer.test.tsx`**、**`draftAtRange.test.ts`** |
| AR4 | 桌面拖动写入范围；移动端行不是按钮 | **`Composer.test.tsx`** |
| AR5 | 无法解析的路径会让面板保持关闭 | **`Composer.test.tsx`** |
| AR6 | 用户气泡显示 **`@path:N-M`** 而不是附件正文 | **`stripCoddyAttachments.test.ts`** |

| 用例 | 预期 | 自动化检查 |
| --- | --- | --- |
| AT1 | 路径内的空格（ **`readme copy.md`**、**`@"my notes.md"`** ）在选择器草稿中可用，并在服务器端解析 | **`draftAt.test.ts`**、**`features/mentions.feature`** |
| AT2 | 选择器会询问 **`GET /coddy/mentions`**（打开时为 **`refresh=1`**），命名每种类型并说明匹配了多少个 | **`Composer.test.tsx`**、**`TestCoddyMentionsGet`** |
| AT3 | 方向键移动高亮；**Enter** 会将其带空格插入，轮次运行时也可用；文件夹会保持选择器打开 | **`Composer.test.tsx`** |
| AT4 | **`session/prompt`** 文本中的 **`@`** 会单独解析（当 **`attachments`** 或 **`resource`** 已有正文时不重复） | **`TestHydratePromptContentBlocksExpandsAtInText`**、**`features/mentions.feature`** |
| AT5 | 气泡会将附件折叠回带来它们的提及，且能识别 CDATA | **`stripCoddyAttachments.test.ts`**、**`internal/mention/mention_test.go`** |
## 对话记录消息类型

![展开在答案上方的思考块](../../assets/thinking-block-open-dark-1280.png)

*展开在答案上方的思考块*

![带有工具卡片和答案的长对话记录](../../assets/transcript-long-dialog-1280.png)

*带有工具卡片和答案的长对话记录*

聊天对话记录渲染的是一个扁平的 UI 消息块列表。每个块都有一个 `type` 和一组最少量的必填字段。

- `user_message`
  - 纯用户输入文本（**不含 Markdown**；**`pre-wrap`** 保留换行）。
- `background_wake`
  - 一个无人输入过的轮次的第一行：模型启动的后台任务结束了，服务器唤醒了智能体（**`notify_on_finish`**，默认开启）（[后台任务](../features/background-tasks.md#被唤醒的轮次是什么样)）。它渲染为**空**，既不是用户气泡也不是提示条：智能体的答案接着上一轮出现，仿佛工作仍在继续，而 Tasks 面板中该任务卡片上的铃铛说明了是什么唤醒了它（*Woke the agent when it ended*）。
  - 在轮次流式传输期间由中继的 **`background_wake`** 帧构建，重新加载后则由消息的 **`background_wake`** 字段构建（**`chat/backgroundWake.ts`** 中的 **`parseBackgroundWakeTasks`** 会读取这两种形态）。它像用户消息一样开启一个轮次：实时状态行从它开始计时，下一次编辑会按照服务器统计用户消息的方式统计它，而被唤醒后失败的轮次不提供重试，因为没有输入内容可以再次发送。
- `thinking`
  - 将模型推理渲染为一条轻量的折叠行。
  - 状态 `in_progress` 显示标签 `thinking...` 和一个运行中的计时器。
  - 状态 `completed` 显示标签 `thinking` 以及实测时长（未测量时为无）并保留文本以供查看。
  - 当提供商报告推理的开销时，token 数跟在时长之后：实时状态行使用简短计数，其工具提示中显示精确数字。该计数在轮次流式传输期间来自结束模型调用的 **`token_usage`** 事件的 **`reasoningTokens`**，重新加载后则来自助手消息的 **`reasoning_tokens`**。OpenAI 兼容服务器将其报告为 **`usage.completion_tokens_details.reasoning_tokens`**，Codex 后端报告为 **`output_tokens_details.reasoning_tokens`**；Anthropic 和 Devin 不单独列出该项，因此它们的行不显示计数。
  - 对于不返回推理文本、只报告其开销的提供商（聚合器后面的 OpenAI `o` 系列模型），也会得到一行，带计数但不带文本。它会在从服务器读取对话记录后出现：轮次结束时，或在轮次进行中重新加载时，此时文本在该行之后仍会继续流式传输；展开后它会说明提供商未返回推理文本。
  - 一个轮次中可能出现多个 `thinking` 块（工具调用后推理可以恢复）。
- `tool_call`
  - 单条工具执行行，与 **thinking** / **memory** 使用相同的折叠外观（**chevron**、**`thinking-label`**、时长使用 **`thinking-dur`** 或 **`-`**）。
  - 标签表示智能体正在做什么，而不是它调用的函数：工具 id 通过 **`tool.name.*`** 目录翻译（**`reading a file`** / **`читаю файл`**、**`running a command`** / **`выполняю команду`**）。MCP 服务器提供的工具由其 `server__tool` id 通过 **`tool.name.mcp`** 命名，即 **`calling browser_navigate on the MCP server playwright`** / **`запускаю browser_navigate на MCP-сервере playwright`**，`mcp__server__tool` 的拼写也以同样方式读取；其他任何没有条目的工具保留其原始 id。**`read`** 通过一个工具提供文件和目录，因此只有当参数能证明时它才显示 **`browsing a directory`** / **`просматриваю директорию`**（**`recursive`**、**`show_hidden`**，或以分隔符结尾的路径）。
  - 在标签旁边，**`.tool-summary-target`** 指明该调用作用于的单一对象，即它读取的路径、它运行的命令、它加载的技能，取自实时状态行所用的同一个 **`toolCallTargetText`**。MCP 服务器自行命名其参数，因此当它的某次调用不包含 **`path`** / **`filePath`** / **`file_path`** / **`url`** / **`name`** 中的任何一项时，就会改为显示第一个读起来像标签的参数：一个非空的单行值，至多 120 个字符，绝不是正文。Coddy 工具则继续命名其文档所述的参数。标签、目标和时长共用一行（**`.thinking-head`**）：让步的是目标，它以省略号裁切，并在 **`title`** 中携带完整值，因此长路径绝不会让该行折到第二行。对文件一部分的 **`read`** 会在路径之后附加它所占的行，写法与提及一致：**`offset`** 120 和 **`limit`** 61 时显示 **`read.go:120-180`**，仅有限制时显示 **`read.go:1-40`**，偏移一直读到末尾时显示 **`read.go:240-`**；读取整个文件时只显示路径。路径让步于省略号，范围则绝不如此，工具提示同时携带两者。目标、**(failed)** 标记和时长是一个组：当标签给该组留下的行内空间太少时（MCP 工具的标签会用一句话同时指明服务器和工具），整个组会移到标签下方，而比行更长的标签会在行内换行；该行绝不会比对话记录更宽。没有单独目标时，如果放得下，时长会跟在换行后标签的最后一个词之后，否则正常换行。
  - 当某种写法更短时，路径会**相对于工作所在目录**显示（`relativeToolTarget`）。根目录是会话目录及其工作区的各个 worktree，最先匹配最深的，因此 worktree 内的文件会相对于该 worktree 而非它所属的检出目录来读取。只重写真实路径，绝不重写命令、模式、url 或名称，绝对路径仍保留在该行的 `title` 中以及展开的卡片中。实时状态行完全不指明路径：它只携带步骤的阶段，不携带它作用于的任何对象。
  - 在 **`pending`** 或 **`in_progress`** 期间，摘要标签使用 **`...`** 后缀（例如 **`reading a file...`**）。**`startedAtMs`** 会驱动一个实时时长，直到工具结束。
  - 当结构化预览和返回的正文同时存在时，它们相接并共用外侧圆角，形成一个连续的执行卡片；两者之间没有间隙、重复的边框、标题或分隔线。正文不带有 **Result** 标签：它是该调用下方唯一的内容。
  - **失败**的调用会在其摘要行上说明这一点：**`.tool-failed-marker`**（**`data-testid="tool-failed-marker"`**）会在目标与时长之间以主题的删除红色渲染 **(failed)** / **(ошибка)**，因此即使该行仍处于折叠状态，失败也能被读到，而不是作为展开正文内的一个彩色圆点。
  - 详情复用权限卡片中针对具体工具的预览，但不含批准操作。**read**、**grep**、**glob** 和 **print_tree** 会获得紧凑的结构化参数预览；未知工具保留带样式的等宽回退。**run_command** / **ssh_run_command** 将命令放在卡片内的嵌入块中，置于 **`$`** 提示符之后，其上方标题为本地调用指明服务器自身的解释器（来自 **`GET /coddy/workspace/context`** 的 **`shell`**，保存在 **`chat/hostShell.ts`** 中；服务器未报告时使用通用标签，对于远程的 **ssh_run_command** 也是如此），复制控件位于同一行（对话记录中为 **`data-testid="tool-preview-copy"`**，权限卡片上为 **`permission-prompt-copy`**），而不是在卡片标题中。**不接收任何参数**的调用没有可预览的输入：它不会渲染空的 **`{}`** 正文，而是仅渲染共享栏（**`data-testid="tool-action-preview"`**）：操作标签、待办状态标记，以及 **Done** / **Running…** / **Failed** / **Cancelled**，因此它读起来就像计划卡片的同类。**load_skill** 完全不显示参数卡片，因为摘要行已经指明了技能。大型的 **`write`** / **`write_file`** 代码预览以及 **`apply_patch`** / **`edit`** 差异使用共享的实测视口：**More…** 仅在真正溢出时出现，保持卡片高度并启用内部滚动，而 **Less** 会再次裁切正文并将其返回顶部。命令块以及其他任何没有该切换控件的预览都不会被限制高度，在手机上也是如此，因此长命令会在其输出上方换行，而不是溢出覆盖到输出上。通用的返回正文保持为纯文本。已完成的 **`load_skill`** 调用会渲染 Markdown；受支持的结构化工具会渲染下文所述的专用卡片。失败的调用保留原始等宽面板。若被截断（19 行内容加 **`...`**），则应用限制高度的结果视口（约 20 行），并隐藏 **overflow-y** 直到点击 **More…**；**More…**（**`data-testid="tool-result-more"`**）会执行 **GET `/coddy/sessions/{id}/tool-calls/{toolCallId}`**，然后在相同高度下启用 **overflow-y auto** 并变为 **Less**（**`data-testid="tool-result-less"`**）；**Less** 会在 **fullResultText** 仍保留在内存中时恢复被裁切的预览，无需第二次 GET。预览和结果控件都使用共享的左对齐 **`tool-overflow-toggle`** 标签按钮。

![工具调用标签](../../assets/web-ui/tool-label-wrap-dark-390.png)

*没有单独目标时，时长跟在标签的最后一行之后。*

## 输入点旁边的实时状态

在整个运行中的轮次期间，包括流式文本，一个 Coddy 标志的工作指示器会携带一条实时状态行（`TypingDotsMessage.tsx`，纯派生逻辑在 `chat/liveStatus.ts`，视觉约定在 `DESIGN.md`）。

该指示器用 14px 的 SVG（`coddy-work-indicator`）取代三个跳动的点。其裁切后的标志 `viewBox` 在视觉上与工作区 `FolderIcon` 相匹配。根 SVG 保持静止，暗淡的内环（`coddy-work-indicator-ring`）保持固定。外层的 C 轮廓（`coddy-work-indicator-c`）会旋转，并从一段短弧增长为接近完整的圆，然后再收缩，如此持续循环。CSS 动画 `coddy-work-orbit` 使轮廓旋转；`coddy-work-arc` 改变其 `stroke-dasharray` 和 `stroke-dashoffset`。它没有底板、背景或阴影，行高保持不变。在 `prefers-reduced-motion: reduce` 下以及不可见的空闲槽位中，指示器使用静态的 Coddy C 及固定内环。该行保留其 `typing-dots` 和 `typing-dots-status*` 钩子；计时器、状态短语和任务计数均不变。

![实时状态行旁边的 Coddy 标志工作指示器](../../assets/work-indicator-dark-1280.png)

*外层的 C 围绕固定的暗淡内环旋转、增长和收缩*

![一个运行中的轮次：18s、99 tokens、1 个运行中的任务、Writing the answer](../../assets/turn-progress-live-line-dark-1280.png)

*运行中轮次的实时行：轮次时钟、已生成的 token、运行中的后台任务以及智能体正在做什么*

![新聊天室的第一条消息立即出现在屏幕上，实时行显示 Preparing the session，同时会话的 MCP 服务器正在启动](../../assets/session-start-preparing-dark-1280.png)

*新聊天室在其第一条消息后三秒：消息已在屏幕上，聊天室已命名，实时行显示会话正在准备中，同时其 MCP 服务器正在启动*

![没有待办清单的工作状态，位于最后一条答案下方、输入框上方](../../assets/web-ui/live-status-spacing-working-dark-1280.png)

*没有清单的工作状态，位于输入框上方*

- 行首是该轮次自身的数字：轮次已运行多久、模型在该轮次中已生成多少 token、当前有多少后台任务正在运行（`15m 08s · 13.5k tokens · 1 running task · Thinking…`）。在第一个 token 之前，该行只有时钟和短语（`57s · Waiting for the model`），任务段仅在有事运行时出现。它是一个按钮：点击会打开 [Tasks 面板](../features/background-tasks.md#界面)。
- 在宽度不超过 599 px 的手机上，任务计数显示为 `1 task` 或 `2 tasks`，轮次结束后也是如此。俄语使用 `1 задача`、`2 задачи`、`5 задач`。该按钮保留其完整的无障碍名称。平板和台式机保留完整的运行任务标签。
![手机上紧凑的任务计数](../../assets/web-ui/phone-task-count-dark-390.png)

*手机的实时行，任务计数位于时间和 token 旁边。*

- 生成速度跟随输出 token 计数，例如 `1.2k tokens · 9.8 tok/s`。每次模型调用都会开启一个新的窗口：200 token 时步长为 50 token，一旦该调用写过 500 token 后则变为 500 token 步长为 100 token。流式传输期间，速度在该调用的第 200 个 token 时出现，并在该调用余下时间保持，因此短于该长度的调用只在完成后的行中显示速度。它每五秒刷新一次，包括部分步长；停滞的流保持最后一个值。该测量包括文本、推理和工具参数，从第一个输出增量到最后一次，且不包括工具执行、权限等待和首个 token 的耗时。实时 token 计数为估计值。当轮次结束、被取消或失败时，如果轮次写有输出，则会保存一条图注。它将所有调用的输出 token 总数与可用时已测量调用的平均值配对；重新打开会话会恢复它。阻塞式（非流式）提供商和 ACP 智能体模型没有可测量的按模型流式边界，因此其实时输出计数没有速度。以单个输出块交付的流式调用也没有可测量的生成区间：其 token 和耗时都不计入速度平均值，但其输出仍计入轮次总数。没有可测量的调用区间时，实时输出计数器仍然工作，但持久化的生成图注没有速度。低于 2 的速度使用两位小数，低于 10 使用一位，10 及以上不使用小数。

![流式传输期间的生成速度](../../assets/generation-speed/generation-speed-live-dark-1280.png)

*实时行在输出 token 计数之后立即显示生成速度*

- 已结束的轮次保留同一行，但没有步骤短语：静止的 Coddy 标记、轮次运行了多久、token 和平均速度（`1m 21s · 1.2k tokens · 9.8 tok/s`）。将鼠标悬停其上可查看其结束时间。当该轮次启动的任务仍在运行时，同一行会以这些任务的计数结尾，标记则继续旋转。

![已结束轮次的行](../../assets/generation-speed/generation-speed-completed-dark-1280.png)

*已结束轮次的行在重新加载后仍然保留*

- 当轮次已结束而其启动的后台任务仍在运行时，指示器停留在对话记录末尾，只显示计数，不显示其他内容（`1 running task`）：没有时钟也没有 token，因为没有轮次可计时。它打开同一个面板，并在最后一个任务完成时立即消失。
- 轮次之间有一个空的、透明的槽位，为实时行保留空间。无论有没有清单，实时行都使用相同的内边距，与周围对话记录行的间距一致。当清单显示时，该槽位和仅显示任务的行位于清单上方。
- 时钟和 token 由服务器提供：智能体在轮次流上发布 `turn_progress`（[HTTP API](../reference/http-api.md)），并在 `GET /coddy/sessions/{id}/activity` 之后保持相同的数字，重新加载的标签页会读取它，因此时钟会从原来的位置继续运行。`chat/turnProgress.ts` 融合这两个来源：一次读数以其轮次的服务器 `startedAt` 来指认该轮次，并按该轮次在服务器时钟上的年龄来确定时间，因此较新的读数无论来自哪个方向都获胜：在流的最后一帧之前读取、之后到达的活动响应会被丢弃，而被节流的标签页晚几秒处理的响应仍然是同一个轮次。当轮次结束以及任何发现会话空闲的活动读数时，标签页都会丢弃这些数字，因此下一轮永远不会从上一轮的时钟开始。早于 `turn_progress` 的服务器会让时钟从轮次的用户消息开始计数，且没有 token。
- 当前步骤的短语，以及当步骤运行的不是模型而是工具调用时的步骤计数器（`2m 05s · 1.2k tokens · Running a command · 45s`）。等待、思考和写作由轮次时钟涵盖。在手机上，该行占两行：轮次的数字紧挨指示器，短语在其下方，因此轮次结束时这些数字仍停留在原处。
- **该行只携带阶段，不携带步骤作用于的任何对象。** 没有路径、没有命令、没有模式、没有 url：步骤作用于什么只在实时行上方的对话记录行中指明一次。因此每个短语都自身完整，如 `Running a command`，绝不会是像 `Running` 那样等待跟上命令的片段，而读起来像片段的短语会在每个语言环境的字典检查中失败（`external/ui/src/ui/i18n/statusPhrases.test.ts`）。对行来说过长的短语会以省略号裁切，而不是让行换行。
- 优先级：未解决的权限提示 → 未解决的问题提示 → 正在运行的工具调用（`in_progress` 调用优先于稍后宣布的 `pending` 调用）→ 进行中的思考（`Thinking…`）→ 作为轮次最新一行的答案文本（`Writing the answer`）→ 等待模型。该行始终在指示器旁携带一个短语。（委派给记忆子智能体的操作在此处显示为其 `spawn_agent` 工具调用；`Working with memory` 短语属于子级自己的 `coddy_memory_*` 工具，而不属于父轮次。）
- 两种提示状态都**不**渲染步骤计数器：操作者做决定时没有东西在运行。轮次时钟继续计时。
- 单纯的等待会随时间升级：`Waiting for the model` → `The model is taking longer than usual`（15 s，`typing-dots-status--slow`）→ `Still no response from the server`（60 s）。
- 在那之前，自其消息以来未显示任何内容的轮次会显示 `Preparing the session…`，直到服务器报告它正在与其模型通信：轮次的第一个 `turn_progress` 在轮次引入其 MCP 服务器并等待其模型的上下文窗口期间携带 `preparing` 阶段（`chat/liveStatus.ts` 中的 `withPreparingPhase`）。此期间的等待不会升级，因为慢的不是模型。
- Coddy 未定义的工具采用通用的 `Running a tool`，它无法说明 MCP 服务器提供的调用，因此那一条短语会改为指明服务器和工具：`Calling browser_navigate on the MCP server playwright`。那是该步骤的身份，而不是它的参数；调用作用于什么在其上方的对话记录行中读取，该行也会指明服务器和工具。
- 派生逻辑会回溯到最后一个 `user_message`，因此已结束轮次中过期的 `in_progress` 行绝不会驱动该标签。短语表的控制台对应版本位于 `external/cli/status.go`。
### 折叠的工具调用

同一段对话记录里相邻的工具调用在至少有两个时会折叠成 **使用了 N 个工具** 一行。第二行按类别计数 - Shell、读取、搜索、编辑/写入、网络、子智能体、其他 - 失败调用保留失败标记。调用全部结束后组默认收起，有调用正在运行或失败时展开；展开后仍是原来的逐调用行，单独一个调用保持原样。权限提示、question、子智能体、待办、调度或计划卡片、后台任务、图片或共享文件不放进组里，因为折叠会藏住读者可能需要操作的内容。

## 工具调用卡片（内置 SPA，当前）

![编辑操作的权限卡片：差异，然后是 Allow、Allow always、两个会话开关和 Reject](../../assets/screenshot-tool-previews-dark.png)

*编辑操作的权限卡片：即将应用的差异，然后是各个答复、Reject 之前的两个会话开关*

![一个较长的工具结果折叠在 More 之后，并用 Less 展开](../../assets/screenshot-tool-previews-overflow-dark.png)

*一个较长的工具结果折叠在 More 之后，并通过 Less 展开*

调度器工具（`coddy_scheduler_*`）拥有自己的卡片，而不是显示其 JSON：栏中会标明任务及其发生的情况（*已恢复*、*运行已开始*、*未在运行*、*已创建*、*已更新*、*已删除*），读取或创建的任务会按其字段显示，包括描述、带可读说明的调度、状态、下次运行、模式、模型、文件夹，其中指令以 Markdown 呈现；任务列表每个任务一行，而某个任务的运行记录每次运行一行，并显示其状态和持续时间。

其他所有以已知形状作答的内置工具，以及 MCP 服务器的每个工具，也都有卡片（`StructuredToolCard.tsx`，解析位于 `chat/structuredToolDisplay.ts`），而不是在其为模型编写的文本之上显示 JSON 参数：

- `http_request` 会标明它实际抓取的地址（包括查询串），以及所有会改变它去向或它所信任内容的东西：调用的标头，其中携带凭据的标头会被遮蔽（`tools.http_request.default_headers` 添加到每个请求的标头不是该调用的参数；其权限提示会列出它们）、正文（其类型、文件或表单字段，以及它发送的 JSON 或文本）、隐藏凭据的代理、以失败颜色显示的已跳过证书检查、已跟随的重定向、超时、输出文件和理由。响应（状态、遮蔽后的标头、等宽字体正文、工具备注）位于一条分隔线下方；
- `switch_model` 会显示模型、推理级别及其生效时长，按工具在切换后的报告呈现；
- `background_*` 工具会为每个任务显示一行，其状态采用任务面板中的措辞，而 `background_output` / `background_wait` 以等宽字体显示输出；`preview_server` 会将其地址显示为链接，并显示任务及停止时间；
- `coddy_docs_search` 会将每个命中项链接到文档阅读器中，而 `coddy_docs_read` 会以带链接和行范围的方式呈现它读取的章节；
- `plan_list` / `plan_read` / `plan_write` 会显示计划列表、某个计划的名称及其正文上方的概览，以及写入操作所写的计划；`session_describe` 显示标题、标签和更改内容；`config_*` 工具会将其答复显示为字段，并显示服务器已遮蔽的值；`coddy_memory_*` 工具会将 `.md` 笔记显示为文档，并将搜索命中项显示为行；`keep_result` 和 `compact_context` 会将其参数显示为字段；
- MCP 服务器的调用会在栏中显示其服务器和工具，将参数显示为字段，并按答复的形状显示答复：JSON 对象显示为字段，其他 JSON 缩进显示，Markdown 显示为文档，其他内容显示为等宽文本。数字和 JSON 按服务器写入的原样显示，绝不从解析后的值回印：超过 2^53 的 id（例如 `12345678901234567891`）不会被舍入，重复键会为每个值显示一行，而诸如 `\u00e9` 的转义会保留在缩进的 JSON 中（`chat/jsonSource.ts`）。参数以及 `http_request` 的 JSON 正文同样如此。

失败或取消的调用，以及参数无法解析的调用，会保留原始面板。被截断的答复会在相同的 **More…** / **Less** 控件下限制卡片正文的高度，从而让栏保持在视野内。

![深色对话记录中的结构化卡片：通过代理发出的 http 请求、后台任务列表和某个任务的输出](../../assets/web-ui/tool-structured-cards-dark-1280.png)

*一个 http 请求、后台任务列表和某个任务的输出*

![读取 screenshot.png：图片预览显示在该行下方，然后是答复](../../assets/read-image-preview-dark-1280.png)

*模型看到的图片，预览显示在读取操作所在行的下方*

返回图片的调用（图像文件的 `read`、MCP 工具的截图（[图片](../features/images.md)））会在其行或 MCP 卡片下方预览该图片：每张图片一张卡片，大小与发送的附件相同，在行收起时可见，点击会在文档阅读器所用的查看器中打开原图。轮次运行期间，卡片来自调用最终更新的 `_meta.coddy.images`；重新加载后，则来自工具行的 `files`；字节像附件一样经由环境传输，因此远程服务器或中继也能显示它。它是 Coddy 随会话保留的副本，因此之后被覆盖的文件仍会按调用返回时的样子预览。卡片会显示副本的 160 px 缩略图；没有缩略图的图片（服务器无法解码的 WebP，或超过 1600 万像素、解码开销过大的图片）会用原图填充卡片。失败的调用不显示卡片。

![文档搜索和读取卡片以及会话归档卡片](../../assets/web-ui/tool-structured-documents-dark-1280.png)

*一次文档搜索、它读取的章节以及会话的标题和标签*

![对话记录中的 MCP 调用：参数显示为字段，JSON 答复在分隔线下方显示为字段，Markdown 答复显示为文档](../../assets/web-ui/tool-mcp-cards-dark-1280.png)

*MCP 调用：参数显示为字段，JSON 答复在其下方，Markdown 答复在更下方*

![窄屏上的结构化卡片](../../assets/web-ui/tool-structured-cards-dark-390.png)

*390px 下的相同卡片：栏会换行，而不是裁切地址*

![对话记录中的调度器调用：读取的任务按其字段显示，恢复和运行按其结果显示](../../assets/scheduler-tool-cards-dark-1280.png)

*对话记录中的调度器调用：读取的任务按其字段显示，恢复和运行按其结果显示*

`share_file` 为智能体提供了一种显式且受权限控制的方式，用于将报告或其他工作区文件交还给操作员。成功调用会在其工具行下方渲染可下载的文件卡片。浏览器下载的是保留在会话包中的不可变副本，因此该链接不会跟随之后的工作区更改；在远程环境中，同一操作会通过已配置的、使用 bearer 认证的 API 路由获取。缺失的副本会保持可见并显示为不可用，而不会变成损坏的图片或链接。

当后续助手答复引用已验证的共享文件时，这些文件的卡片会随该答复内联渲染。相邻的文件标记会形成一条响应式行，在窄屏上换行，并在下一个段落前保留 16px 间距；若该行结束答复，则会在答复的复制按钮和时间上方保留它们各自的 8px 间距。卡片尺寸为 134x140px（手机上最多 118x132px，每行两个）。图片构件会从已验证的内联预览 URL 显示缩略图；非图片文件保留其类型图块。每张卡片都保留相同的操作菜单：**Mention in chat** 会将文件路径作为 `@` 提及放入输入框，**Copy** 会打开一个子菜单，用于复制绝对路径、相对路径或文件名，**Download** 则保存副本。在桌面端，子菜单会在指针下的菜单旁打开，可通过点击和右方向键打开，左方向键或 Escape 会返回 **Copy**；在手机、平板或触摸屏上，菜单是一个列表，三种复制方式位于 **Copy** 标题下。其 **⋮** 位于右上角，与左上角的扩展名徽标齐平，在指针悬停时显示，并且在手机或其他没有悬停的屏幕上始终显示；菜单会在其下方打开（靠近窗口底部时则在其上方），保持在窗口内，页面滚动时跟随卡片，并会在再次按下 **⋮**、在别处按下、按 **Escape**、按 **Tab**（焦点从 **⋮** 移走），或卡片滚动到视野外（滚出窗口，或滚到聊天标题或输入框下方）时关闭。在比菜单更矮的窗口中，其项目会滚动。右键单击卡片会打开同一菜单。

单击卡片会在 [文件窗口](#工作区文件) 的标签页中打开该卡片复制来源的文件，单击图片缩略图也会如此。仅当文件位于聊天的工作区中时，卡片才会打开它：对话记录的列表会说明每个来源的位置（**`sourceState`**，[HTTP API](../reference/http-api.md)），卡片在单击时还会再检查一次。当文件已从工作区删除时，卡片会显示 **File deleted**，名称带删除线，单击不会打开任何内容，且 **Mention in chat** 被禁用，但仍可复制路径并下载副本；此时图片缩略图会在共享图片灯箱中打开副本。另一个文件夹中的文件（文件共享后聊天已移到 worktree）也无法从卡片打开；其菜单会说明该文件不在当前聊天的工作区中。

![共享文件卡片的来源已从工作区删除：名称带删除线，并在其下显示 File deleted](../../assets/web-ui/shared-file-deleted-dark-1280.png)

*来源已删除的共享文件：卡片会说明这一点，副本仍可从其菜单下载*

![答复中内联的共享文件卡片，其操作菜单在触发按钮下打开，Copy 子菜单在其旁边](../../assets/web-ui/shared-file-menu-open-dark-1280.png)

*答复中内联的共享文件卡片，其操作菜单在触发按钮下打开，Copy 子菜单在其旁边*

`spawn_agent` 有专用的参数卡片：智能体图标和名称、可选描述、带标签的超时徽标，以及用于多行提示词的内嵌面板。提示词初始为紧凑预览；**More…** 会打开一个有界、可滚动的视口，**Less** 会恢复预览，因此较长的子智能体指令不会占据对话记录；能够完整显示的提示词没有 **More…**，并且随着卡片大小调整，该控件会跟随提示词的溢出情况。子智能体的对话记录可从任务面板中该任务卡片上的 **Show transcript** 打开；调用卡片没有此类链接。超时是以秒为单位的所提供的执行限制，与工具标题旁的已用时长分开显示。布局会在窄屏上换行，并跟随当前浅色/深色主题。历史参数被截断的调用会针对每个不完整预览加载一次完整参数，包括正在运行的调用；格式错误的参数或获取失败会保留普通参数预览。卡片标签会跟随当前的英语/俄语 UI 区域设置。

**History** 和 **Scheduler** 的活动轮次徽标位于项目图标的右上角，覆盖在图标上，并适用于栏的每种形态：仅图标、带标签（宽版）以及手机 **More** 菜单中的行；它从不跟随标签。History 会计入活动聊天及其子智能体的轮次，同时仍将子对话记录排除在 History 列表本身之外；调度器运行只计入 **Scheduler**。Scheduler 的计数会在页面打开时读取，并在每个轮次的开始和结束时再次读取，无论 Scheduler 是否打开。

正常路径位于 `features/spawn_agent_card.feature`，由 `http,ui` godog 框架通过 `SpawnAgentCard.test.tsx` 中的 React DOM 测试运行。边界情况覆盖无效参数、缺失/无效超时、转义后的提示词文本以及历史记录获取失败。

权威行为与 **`DESIGN.md`** 工具时间线以及本清单一致。

| 关注点 | 当前行为 |
| --- | --- |
| 组件 | **`ToolCallMessage.tsx`** 至 **`thinking-row coddy-tool-call-row`**、**`details.thinking-details.coddy-tool-details`**、**`data-testid`**：**`tool-details-{toolCallId}`** |
| 摘要 | 与 **thinking** 相同的模式（**`thinking-summary`**、**`thinking-left`**、**`thinking-chevron`**），其中 **`thinking-label`**、**`.tool-summary-target`**、失败状态 **`.tool-failed-marker`** 和 **`thinking-dur`** 位于 **`.thinking-head`** 内，并按上文所述换行；在后台运行的 **`run_command`** 上，该时长是任务的时钟（**`data-testid="tool-bgtask-elapsed-<id>"`**）；**`aria-label="Tool summary"`** |
| 标签 | **`toolDisplayName`**（**`messages/toolDisplayName.ts`**）覆盖 **`tool.name.*`** 字典项；**`server__tool`** 名称（接受 **`mcp__`** 前缀）会通过 **`tool.name.mcp`** 读作 *calling {tool} on the MCP server {server}*，任何其他未知 id 会回退到自身。**`background: true`** 在 **`run_command`** 上会选择 **`tool.name.run_command_background`**（*running a command in the background* / *выполняю команду в фоне*），仅从完整参数读取 |
| 参数 | 共享 **`PermissionToolPreview`**（无批准操作；唯一的复制控件是 shell 命令块内部的控件，**`data-testid="tool-preview-copy"`**）；较大的 **write** / **write_file**、**apply_patch** 和 **edit** 正文会保留经过测量的 **More…**（**`data-testid="tool-preview-more"`**）/ **Less**（**`data-testid="tool-preview-less"`**）溢出控件 |
| 结果 | 通用输出使用 **`div.tool-call-result-card`**、**`aria-label="Tool result"`**，内部使用 **`pre.tool-result-pre`** 且没有标题行；已完成的结构化 todo 和 **`plan_exit`** 卡片会抑制多余的样板结果；**`load_skill`** 渲染 Markdown，受支持的工具和 MCP 调用使用 **`StructuredToolCard`**（**`data-testid="structured-tool-card"`**） |
| Markdown | 不用于工具 **result** 或 **user** 气泡；**assistant** 仍按下文使用 Markdown |
| 列表合并 | **`App.tsx`** **`loadMessages`** 会将 **`GET /coddy/sessions/{id}/tool-calls`** 行合并为 **`resultText`**、**`resultWasTruncated`**、计时 |
| 全文 | 首个结果 **More…**，或对任意状态下恢复的 **`apply_patch`** / **`write`** / **`write_file`** / **`edit`** 卡片自动进行不完整参数恢复，即 **`GET /coddy/sessions/{id}/tool-calls/{toolCallId}`**，使用 JSON **`result`** 和 **`args`**（同一对象包含 **`meta`**）。对话记录协调绝不会用截断的 200 字符 **`argsPreview`** 替换完整参数（**`pickRicherToolArgs`**），因此实时卡片在权限答复前后都保留完整预览 |
| 图片 | **`ToolImagePreviews.tsx`** 位于行下方，在 **`details`** 之外：**`.tool-images`**，每张图片一个 **`button.tool-image-card`**（**`data-testid="tool-image-open"`**，缩略图 **`tool-image-thumb`**），可打开 **`ImageLightbox`**；仅在调用完成时，从该行的 **`images`** 显示（实时为 **`_meta.coddy.images`**，重新加载后为工具行的 **`files`**） |
| CSS | **`styles/chat/_tool-rows.scss`**：透明 **`.coddy-tool-call-body`**、**`.tool-call-result-card`**、**`thinking-details:not([open])` body hidden**、**`.tool-images`** / **`.tool-image-card`**，外加上述结果视口 / 切换类；共享的 **`.permission-preview*`** 位于 **`_permission.scss`** 中，**`_tool-cards.scss`** 位于其旁边 |

- `assistant_message`
  - 该轮次的最终助手输出文本，位于工具调用之后。
  - 答复下方不带复制控件和时间，无论是否完成：该行曾在轮次结束时出现，并把 todo 清单和实时行向下推了自身高度。答复的工具提示会说明答复的写入时间。用户消息保留其复制控件、编辑控件和时间。

## 工具权限卡片

内联批准门由 **PermissionPromptSection** 和 **PermissionPromptPreview** 实现。

- 仅对待处理的权限请求渲染卡片。只读工具仅渲染其正常时间线行；没有信息性的无需批准卡片、对勾或说明性句子。
- 标题：人类可读的操作问题加一个原始工具 ID 徽标。预览标题保留给路径、shell 或操作范围，以免重复工具名称。
- 操作使用服务器提供的标签，保持不变（**Allow**、**Allow always**、可选的 **Always allow `<program>`**、**Reject**）。选项列表从 SSE 负载渲染，因此第四个按钮除了布局外无需客户端更改。
- 当会话发起询问时，服务器会在 **Reject** 之前添加会话开关：**Bypass for this session**（**`allow_session_bypass`**，以红色绘制，**`permission-prompt-btn--session-bypass`**），对于文件写入还有 **Allow edits for this session**（**`allow_session_accept_edits`**，**`permission-prompt-btn--session-edits`**）。两者都会批准调用并切换会话，权限标签则依据设置事件更新（[会话设置](../features/session-settings.md#从权限对话框切换)）。
- 程序级选项仅在单次普通调用中针对 **run_command** 到达客户端。其标签已经指明了确切的授权（**`curl`**、**`git status`**），因此卡片必须原样渲染，而不是重新推导程序名。
- 通过 **toolCallId** 将提示与对应的 **tool_call** 匹配，并优先使用该行的 **argsText**；回退到权限负载中的 **Arguments:** 内容。
- 服务器自行唤醒的轮次也会在这里询问。其提示词通过该标签页所跟随的输入框中继到达，会作为会话的待处理提示词持久化（因此重新加载会像其他提示词一样恢复它），任何客户端（此标签页、另一个标签页、跟随该轮次的控制台）给出的第一个答复都会将其确定下来。
- **apply_patch** 和 **edit** 会渲染旧/新行号栏，以及跟随主题的添加/删除/上下文行。其他文件系统修改工具和 **run_command** 使用紧凑的结构化预览，而不是 JSON。
- 折叠预览会在布局后测量。仅当 **scrollHeight > clientHeight** 时显示 **More…**；保持视口有界，将其切换为内部垂直滚动，并将按钮改为 **Less**。返回折叠状态会恢复裁切并重新测量溢出。共享按钮左对齐；在手机上最小高度为 **36px**。
- 恢复的写入权限提示会与其他文件系统修改工具一起包含 **rm** 和 **rmdir**。

自动检查：

- **external/ui/src/ui/chat/permissionToolPreview.test.ts**
- **external/ui/src/ui/chat/PermissionPromptSection.test.tsx**
- **external/ui/src/ui/messages/MessageList.test.tsx**

## 问题卡片

![问题卡片：两个问题，模型提供的选项，以及每个问题下方用于填写你自己答案的字段](../../assets/web-ui/question-card-own-answer-dark-1280.png)

*问题会以 Other… 结尾，即一个用于填写你自己答案的字段，除非模型将其关闭*

当智能体使用 `question` 工具向你提问时，其行下方的卡片会在各字母后列出每个问题的选项，并以 **Other…**（一个用于填写你自己答案的字段）结束问题，单选题和多选题皆如此。该字段遵循问题的 `custom` 标志，该标志默认开启，因为模型提供的选项常常无法表达你的意思。仅当答案必须是其某个选项时（例如是/否确认），模型才会通过 `custom: false` 将其关闭，此时问题只列出其选项。在多选题中，你输入的文本会与你勾选的选项合并。**Continue** 或 Return 发送答案，**Skip** 或 Escape 跳过问题，卡片上方的工具行会保留完整提议，并标记你选择的字母。

不带该标志的问题也会提供该字段，因此省略该标志的旧版本服务器会得到相同的卡片。控制台会以 **Custom answer...** 提供相同答案（[控制台](console.md)）。

自动检查：

- **external/ui/src/ui/chat/QuestionPromptSection.test.tsx**
- **external/ui/src/ui/messages/ToolCallMessage.test.tsx**
- **features/web_ui_question_own_answer.feature**
## 消息编辑与历史回溯

后续提示词在气泡上方预留 34px，与其下方的页脚空间相匹配（控件占 28px，再加 6px 间隙）。算上对话记录的行间距，普通文本提示词在桌面端上下各留 44px，在堆叠外壳上为 46px；附件则会额外占用各自的空间。第一个逻辑提示词保持其原有间距，即使更早的历史位于渲染窗口之外也是如此。

![后续提示词的间距，以及 Copy、编辑消息和时间页脚](../../assets/web-ui/user-message-controls-dark-1280.png)

*后续提示词的上下空间相互匹配；Copy 位于圆形箭头编辑控件之前。*

编辑已发送的消息会就地改写会话：历史回溯到该消息，编辑后的文本在同一会话中发送，不产生兄弟会话，也无需任何导航。

- 每个用户气泡都带有一个编辑控件（**`.msg-user-edit`**、**`data-testid="user-message-edit"`**，工具提示与无障碍名称均为 **编辑消息**），位于 **`.msg-user-foot`** 中。其图标是一个简洁的逆时针圆形箭头，采用与 Copy 相同的 **`.msg-copy-icon-btn`** 样式，始终可见并参与布局。页脚顺序依次为可选的 **撤销编辑**、**复制消息**、**编辑消息**，最后是时间。编辑控件会把该消息重新载入输入框草稿（附件标签从持久化的 session-assets 注释中恢复），并记录正在编辑的、从 0 开始的 **user** 消息索引。在编辑后的消息发送之前，它不会回溯历史。
- 当某条消息被载入以供编辑时，输入框会予以提示：卡片上方的横幅（**`.composer-edit-banner`**、**`data-testid="composer-edit-banner"`**）会标明它（**正在编辑消息**、其文本的开头、*发送后将把会话回溯到该消息*），发送按钮的无障碍名称变为 **发送编辑**，该气泡被描边（**`.msg-user-stack--editing`**），其后的每一行都会变淡，因为发送恰好会移除这些行。在输入框中按 **Escape**（在没有任何选择器或弹出层占用它之后）或点击横幅的叉号（**`data-testid="composer-edit-cancel"`**）会退出编辑，并还原编辑控件此前替换掉的草稿和附件；另一条消息的编辑控件会把编辑切换到那条消息，并保留同一份草稿以便返回。发送编辑后也会归还该草稿，因此在使用编辑控件之前写下的后续内容不会丢失。
- 发送该草稿会调用 **`POST /coddy/sessions/{id}/rewind`**，并带上 **`{"userMessageIndex"}`**。草稿和编辑状态会保持到回溯完成：请求失败会以一条 UI 日志的错误行呈现，不会丢失任何内容。
- 在 **200** 时，客户端丢弃影子对话记录以及被移除尾部中持久化的权限提示，从 **`GET .../messages`** 重新加载保留的前缀，并把编辑后的文本作为同一会话的下一轮发送。
- 服务器在编辑点截断 **`messages.json`**，将被移除轮次的 **`tool_calls/`** 条目和 **`ui_log`** 行移入撤销快照（**`rewind_undo/`**，一旦后续提示词或再次回溯将其弃用便会被移除），并清除其工具调用已离开对话记录的待处理权限提示。**`session_rewound`** 会在 **`GET /coddy/events`** 上告知其他持有该会话的每个标签页或界面，以相同方式重新加载。
- 已发送的编辑可以在 **`GET .../messages`** 携带 **`rewindUndo`** 期间被撤回：编辑后的提示词在其页脚带有 **撤销编辑**（**`.msg-user-undo-edit`**、**`data-testid="user-message-undo-edit"`**），横幅位置显示 **消息已编辑** 以及 **撤销**（**`data-testid="composer-undo-banner"`**，会因该次编辑被其叉号隐藏；提示词上的控件仍然保留）。撤销会在编辑后的轮次仍在运行时将其停止，调用 **`POST /coddy/sessions/{id}/rewind/undo`**，并重新加载对话记录；被拒绝则是一条错误行。发送另一条提示词会结束撤销，与服务器端的行为一致。被移除轮次所做的文件更改不会被还原，两个控件都会说明这一点。

![编辑已发送消息时输入框上方的编辑横幅](../../assets/web-ui/message-edit-banner-dark-1280.png)

*编辑已发送的消息：横幅会标明它，气泡被描边，发送后将移除的轮次会变淡。*

![编辑发送后的撤销横幅](../../assets/web-ui/message-edit-undo-dark-1280.png)

*编辑发送后，提示词上的撤销和横幅中的撤销会将会话恢复到原样。*

自动化检查：

- **internal/session/rewind_test.go**（就地截断、产物清理、拒绝）
- **features/session_rewind.feature** + **external/httpserver/bdd_rewind_test.go**（端点的正常路径）
- **external/ui/src/ui/messages/userMsgIndices.test.ts**（编辑所指向的索引，唤醒也算作一轮）
- **external/ui/src/ui/messages/MessageList.test.tsx**（当对话记录只保留长历史末尾时编辑所指向的索引，参见 **长会话**）
- **external/ui/src/ui/messages/UserMessage.test.tsx**（编辑控件的可见性、编辑标记、撤销编辑）
- **external/ui/src/ui/chat/Composer.editBanner.test.tsx**（编辑与撤销横幅、Escape、发送按钮的名称）
- **external/ui/src/ui/App.messageEdit.test.tsx**（取消操作会还原草稿，撤销会还原会话）
- **internal/session/rewind_undo_test.go**、**features/session_rewind_undo.feature** 和 **external/httpserver/coddy_rewind_undo_test.go**（保留的尾部、撤销何时结束、停止编辑后的轮次）

## 后台任务面板

截图：`docs/assets/screenshot-fullhd-tasks.png`。

该面板停靠在**会话内部**、对话记录的右侧（`.bgtasks-panel`），而不是外壳抽屉：任务属于启动它的那个聊天。在 `min-width: 1200px` 上，聊天列只让出面板实际覆盖的宽度，因此在宽窗口中打开面板时，对话记录和输入框会保持原位。路由为 `#/s/<sessionId>/tasks`，因此重新加载会同时恢复聊天和面板；关闭时会写回 `#/s/<sessionId>`。指定某个任务的链接（`#/s/<sessionId>/tasks/<task_id>`）仍会打开对应卡片，随后地址会去掉该 id。其后端为 `/coddy/sessions/{id}/background-tasks*`（参见 `docs/features/background-tasks.md`）。

- 它通过**轮询**而非监听 SSE，因为后台任务的存活时间会超过启动它的轮次：只要有任务在运行就每 2.5 秒轮询一次，否则每 15 秒一次。对不可达服务器的轮询会产生正常的错误结果，绝不会出现未处理的拒绝。
- **每个任务都是一张卡片**（`TaskCard`）：状态圆点、标签（`taskTag`：`shell`、智能体名称、`memory`、`server`）、标题（`taskTitle`：命令，或智能体运行的描述，去掉池写入的 `agent <name>:` 前缀）以及一行元信息（`taskMetaLine`：运行时为已用时间对比预估时间，结束后为 `20s · 12:50`，结果体现为圆点的颜色，并以文字形式仅出现在展开卡片的页脚中）。智能体运行会在元信息行右侧带上模型及其调用所消耗的 token（`agentUsage`：`agent.model` 用其简称，`agent.input_tokens` 加上 `agent.output_tokens`），当两者放不下时整体落到状态下方；完整模型 id 以及输入 / 输出拆分是展开器的 `title`。在这两者下方，卡片会在单独的一行上提供进入该任务的入口，且至多只有一个：智能体运行带有**显示对话记录**（`.bgtask-card-transcript`、`data-testid="bgtask-open-transcript-<id>"`），它会像从历史记录中选择一样，在 `#/s/<child id>` 打开子会话；正在运行的预览服务器带有其地址（`.bgtask-card-link`、`data-testid="bgtask-link-<id>"`、`target="_blank"`、`rel="noopener noreferrer"`），会在新标签页中打开该页面，地址较长时整体换行而非截断。两者都无需展开卡片即可使用，并且与 Stop 一样，它们都位于卡片自身的点击表面之上，因此绝不会触发展开。在该行携带 `agent.session_id` 之前，显示对话记录是纯文本，并带有一个说明原因的 title，按住修饰键点击该链接会在新标签页中打开聊天；已停止的服务器不携带地址，因为它所指的页面已不存在。运行中的卡片位于顶部，没有自己的标题，**已完成 N** 计数器上方的一切都是运行中的，并会添加 Stop、标题之后的**铃铛**（`.bgtask-notify`、`data-testid="bgtask-notify-<id>"`、`title`，无障碍名称为*结束时唤醒智能体*，当任务携带 `notify_on_finish` 时），以及仅在模型提供 `expected_seconds` 时才出现的进度条。当该行携带 `woke_agent`（其结束启动了一个轮次）时，已完成的卡片会保留铃铛，并显示*结束时唤醒了智能体*：被唤醒的轮次在对话记录中不显示任何内容，因此卡片是 Web 界面说明是什么唤醒了智能体的地方。
- **卡片本身就是一个控件，并就地展开。** 展开按钮铺展在卡片的摘要之上，因此点击任意位置都会展开它，摘要会在指针悬停时着色；Stop 是位于该表面之上的同级元素，绝不会切换卡片。展开的卡片会显示带有复制控件的命令（智能体运行和预览服务器背后都没有 shell，因此那里什么都不显示）、错误信息（除非它只是重复退出码，`taskErrorText`）、带有独立滚动条的输出框，以及任务完成后显示结果和退出码的页脚（`Failed · Exit code 2`；智能体运行或预览服务器没有退出码：池为它们生成的退出码是合成的）。任务运行了多久以及进入任务的入口属于摘要，不会在卡片内部再次说明。可以同时展开任意数量的卡片，每张卡片都通过面板的 `loadOutput` 读取自己的输出（在其任务运行期间又是每 2.5 秒一次，结束时再读取一次）；没有详情窗格。哪些卡片处于展开状态属于面板的状态，而非地址：路由为 `#/s/<id>/tasks`，仍指定任务的链接（`#/s/<id>/tasks/<task_id>`）会通过面板的 `focus` prop 打开该卡片，随后地址会去掉 id。该指针会标明它所属的聊天，且仅可使用一次（`onFocusHonoured`）：每个会话都从 `bg_1` 开始为任务编号，而面板会随抽屉一起卸载，因此外壳若保留指针，就会在下次打开时于屏幕上当前所在的任意聊天中再次打开一张卡片。
- **已完成 N** 是一个计数器；展开它会显示相同的卡片，最多渲染 40 张，并附上一条说明，指明哪些仍保留在磁盘上（已展开的卡片会显示在其所在位置，因此指向长会话早期任务的链接不会打开一张没人能看到的卡片），而任务刚刚结束的已展开卡片会随之打开该分区。**清除** 会丢弃该会话的已完成历史。
- 无论在实际运行的卡片之间还是在已完成历史内部，排序都纯粹按开始时间进行，最新的在前。
- **展开器**是粘性聊天头部右边缘的 **任务** 按钮（`chat-views-tasks`），而不是导航栏条目。它从第一条消息起就渲染为一个带圆点的方块，一旦聊天有任务（`1 / 3`），就会在圆点旁以任意宽度显示 `running / total`，并且是一个带有 `aria-pressed` 的开关。当某个轮次运行时，实时状态行会列出正在运行的任务并打开同一个面板。两者的计数都通过 `countTasks`（`tasks/taskStatus.ts`）进行，它会排除系统任务，即运行时为其自身启动的运行（目标验证器）；对[记忆子智能体](../features/memory.md)的委派属于普通智能体任务，会被计入。对话记录下方不渲染任何内容。
- 在 `max-width: 1199px` 上，面板占满屏幕，卡片的内边距增大，Stop 增大到 30px，输出框增大到 46vh。
- 启动任务的对话记录 `run_command` 行读起来与其他命令行一样：标签说明它是一次后台运行（*在后台运行命令*），时长位置显示任务的滴答时钟，而不是该调用毫无意义的 `0ms`。结果**不会**出现在该行上：状态、预估时间、退出码和错误都在此面板中该任务的卡片上查看。展开该行会保留时钟，且不会添加任何任务控件：Stop 在任务的卡片上。由同一轮询驱动。
- 无需展开卡片即可读取智能体任务：模型、token、已用时间和**显示对话记录**都在折叠的卡片上。展开它只会增加子级的实时进度日志，并以 `=== subagent report ===` 块结尾。预览服务器的读取方式相同：其地址在折叠的卡片上，展开它只会增加请求日志。

自动化检查：

- **external/ui/src/ui/tasks/taskStatus.test.ts**（计时、进度、超时、轮询节奏、按开始时间排序、分组、智能体任务辅助函数）
- **external/ui/src/ui/tasks/BackgroundTasksPanel.test.tsx**（每个任务统一的卡片形态、标签与标题、卡片作为单一控件且 Stop 独立、就地展开显示命令、复制、输出和页脚、可同时展开多张卡片、外壳指向的卡片、任务运行时重新读取输出、已完成计数器、清除、折叠卡片上的显示对话记录且只出现一次、空状态和错误状态）
- **external/ui/src/ui/tasks/api.test.ts**（路径、请求头、离线降级）
- **external/ui/src/ui/chat/ChatHeader.test.tsx**（视图按钮：按此顺序为文件和任务，且没有 Edits 按钮；任务是一个带圆点的方块，随后是圆点以及不计系统任务的 `running / total`；文件图标单独位于其方块中以及工具提示；`aria-pressed`）、**external/ui/src/ui/changes/workingCopy.test.tsx**（每个聊天读取一次 git 报告，在服务器通知时再次读取，一次完成的工具调用以及页面焦点）和 **ChatScreen.test.tsx**（开关、仅在 git 报告有更改时显示在底板上的计数、对话记录下方无任何内容）
- **external/ui/src/ui/tasks/backgroundTaskCss.test.ts**（面板停靠、标签、铺展的点击表面且 Stop 位于其上、悬停着色、有边界的输出框、头部控件、手机上的实时行、减少动态效果）
- **external/ui/src/ui/messages/ToolCallMessage.test.tsx**（后台行：其标签、时长位置中的任务时钟，以及没有任何结果泄漏到该行）

### 子智能体定义

**设置 > 子智能体** 是一个类似技能的混合标签页（在 `settingsSections.ts` 中区段类型为 `subagents`，`SubagentsSection.tsx`）：`subagents` 配置区段的 schema 驱动表单（`enable`、`dirs`、`project_trust`、`max_concurrent`、`max_depth`、`default_timeout_seconds`、`max_turns`；标签来自 `settings.schema.subagents.*`）会随文档其余部分一同保存，其下方有一个 **定义** 字段集，列出屏幕上聊天所在工作区的 `GET /coddy/subagents` 目录（`chatWorkspacePath` 来自 `App.tsx`：会话的文件夹，或在第一条消息之前选定的文件夹；若没有，则服务器针对其默认工作区作答）。该标签页不会打印该路径。

- 每一行都复用 MCP 列表的外观：名称、作用域徽章（`built in` / `yours` / `from the project`）、`hidden`、作为纯文本的描述以及文件。
- 在 `project_trust: ask` 下，项目定义会带有 MCP 标签页的盾牌（`subagent-trust-<name>`）：未获批准时为琥珀色，一旦获准则为普通样式。点击它会以 `POST /coddy/subagents/{name}/trust` 为该工作区批准该文件，并指明该行所显示的 `digest`，因此若检出此后重写了该文件，则会被拒绝（409），错误会显示在列表上方；点击已批准的盾牌会撤回回执。无论哪种操作之后，列表都会重新读取。仍在等待回执的定义还会带有琥珀色的 `needs approval` 徽章，其工具提示会指明盾牌和 `coddy agents trust <name>`。内置定义和用户定义，以及 `allow` 或 `deny` 下的每个定义，都没有盾牌。
- **声明的边界**，在每一行上折叠（`settings/subagentCatalog.ts` 中的 `subagentDeclaredFacts`）：模型、模式、权限、工具、拒绝项、超时、最大轮次、分离运行和指令大小，每个未声明的边界都显示为继承。长路径和工具列表会在面板内部换行（`.settings-subagents-section` 规则），而不是撑宽面板。

![设置中的子智能体目录](../../assets/subagents/settings-subagents-catalog-dark-1280.png)

后台子智能体在生成它的轮次结束后需要权限时，会在其父会话的聊天中询问：提示词等待在对话末尾，使用与内联提示词相同的卡片，子智能体名称在其头部（`SubagentPermissionCards`、`chat/SubagentPermissionCard.tsx`）。聊天会从会话后台任务行上的 `pending_permission` 读取它，在 `GET /coddy/events` 的 `subagent_permission` 事件上重新读取这些行（任务轮询是后备方案），并针对**子**会话以 `POST /coddy/sessions/{child}/permission` 作答。若提示词先在其他地方被回答（通过 `--remote` 附加的控制台、Telegram 聊天），它会在下一次读取时从聊天中消失。参见 `docs/features/subagents.md`（分离运行）。子级在父级仍在回复时提出的提示词，最初会作为该轮次的内联卡片显示；如果该轮次在其被回答之前结束，中继会撤回那份副本，并在聊天末尾再次提出该提示词，因此内联卡片会随其流一同退场，而末尾的卡片才是负责作答的那张。

多个请求会保持对话记录的 10px 间距。当你跟随聊天末尾时，新请求会滚动进入视野；阅读较早消息时则会保持你的位置。常规的任务刷新不会移动视口。

![两张带标准对话记录间距的后台权限卡片](../../assets/subagents/chat-permissions-spaced-dark-1280.png)

*同一轮次的两个后台子智能体在父聊天末尾请求权限，暗色主题，宽 1280px。*

![后台子智能体在其父聊天中请求权限](../../assets/subagents/chat-subagent-permission-dark-1280.png)

自动化检查：

- **external/ui/src/ui/settings/subagentCatalog.test.ts**（继承和声明的事实、格式化、作用域徽章键）
- **external/ui/src/ui/settings/subagentsApi.test.ts**（查询中的工作区、规范化目录、服务器错误消息、离线）
- **external/ui/src/ui/settings/SubagentsSection.test.tsx**（带有作用域、描述和文件的行；在 ask 下项目定义的盾牌批准文件并显示摘要以及撤回回执；被拒绝的批准；在 allow 或 deny 下以及在 内置或用户定义上均无盾牌；需要批准的徽章；折叠在展开项后的声明边界；加载失败；俄语文案）
- **external/ui/src/ui/settings/subagentsCatalogCss.test.ts**（目录不会超出面板、事实标签列、琥珀色徽章）
- **external/ui/src/ui/settings/SettingsSection.test.tsx**（子智能体类型保留其表单并询问会话工作区）
- **external/ui/src/ui/chat/SubagentPermissionCard.test.tsx**（针对子会话作答、仅显示等待中的任务且最早的在前、无等待时不显示任何内容、标题前缀、俄语文案）
- **external/ui/src/ui/chat/ChatScreen.test.tsx**（提示词等待在父聊天末尾，作答后会重新读取任务）
- **external/ui/src/ui/chat/serverEvents.test.ts**（`subagent_permission` 帧会标明它所属的聊天）
- **external/ui/src/ui/chat/relayedPermissionPrompts.test.ts**（已结束轮次为其子智能体中继的未解决提示词会随其流一同退场；父级自身的提示词则保留）

### 钩子

**设置 > 钩子** 是一个 schema 驱动的对象标签页（`settings-tab-hooks`）：`hooks` 配置区段（`enable`、`files`、`project_trust`、`default_timeout_seconds`、`stop_loop_limit`、`max_output_chars`），带有本地化的标签和简介（`settings.section.hooks.*`、`settings.schema.hooks.*`），并以 `SchemaExampleConfigJSON` 的默认值作为占位符。定义本身存放在 JSON 文件中（`docs/features/hooks.md`）；该标签页编辑的是从何处读取它们以及如何信任项目文件。

![设置中的钩子标签页](../../assets/screenshot-fullhd-settings-hooks.png)

被搁置的项目钩子文件会以**通知级别的系统行**出现在对话记录中：`GET /coddy/sessions/{id}/messages` 在 `uiLog` 中携带它，并带有 `level: "notice"`，SPA 使用与错误行相同的 `SystemNoticeMessage` 渲染它（`system_notice` 对话记录项，`level: "notice"`），但采用更柔和的蓝色调，用 `role="status"` 而非 `role="alert"`，带复制控件，并且**没有重试控件**，即使该行是最后一项也是如此。任何其他级别的行都保持不可见，而不是被错误渲染。

![被搁置的钩子文件通知](../../assets/screenshot-hooks-notice-dark.png)

- **external/ui/src/ui/messages/SystemNoticeMessage.test.tsx**（通知行：status 角色、notice 类、无重试）
- **external/ui/src/ui/settings/settingsSections.test.ts**（`hooks` 配置标签页的翻译标签和简介）
## 聊天的视图

聊天在对话之外有三个视图：它的**文件**、**后台任务**和**编辑**。文件和后台任务是聊天顶部的一排按钮：**Files** 和右边缘的 **Tasks**。它们在所有宽度下都是图标，就像手机一直显示的那样：Files 只有图标本身，位于与顶栏按钮同样大小的方形区域内。聊天有任务之前，Tasks 是同一方形区域里的一个点；有任务后，按钮会变宽，并在点的旁边显示正在运行的数量与总数，有工作在执行时该点会亮起。每个按钮都在工具提示中显示完整名称，Files 按钮还会显示其快捷键。按钮可打开对应视图，再次按下则收起；当前显示的视图按钮会变亮，包括其文字、边框和背景，就像被指向时那样，而强调色保留给正在运行的任务。后台任务在聊天旁的停靠区打开，文件则在聊天上方的窗口中打开。编辑在页头没有按钮：打开它们的是[输入框上方的信息条](#输入框上方的信息条)上的 git 计数。单词旁的每个图标（Tasks 的点、信息条上的仓库和分支）都对齐其小写字母的中线，而不是大写字母的中线：这些单词大多是小写，居中到该行时图标会显得高出一个像素。

![运行中的聊天：顶部的视图按钮 Files 和 Tasks，以及输入框上方标明仓库、分支和 git 计数的信息条](../../assets/views-toolbar-dark-1280.png)

*聊天顶部的视图按钮，以及其输入框上方的信息条*

在手机上：[390 px](../../assets/views-toolbar-dark-390.png)。`Ctrl+Shift+F`（在 Mac 上为 `Cmd+Shift+F`）可在聊天中的任意位置打开和关闭 Files 窗口，包括输入框。

## 画布

工作区中的 `.canvas` 文件会在 Files 窗口中以画板形式在其独立标签页中打开：实时框架，每个都是运行在沙箱中的工作区页面，文件中的卡片、分组和边，以及画板下方的提示词栏，本轮次的权限卡和问题卡位于该栏上方。当智能体将页面放到画布上时（手机上从不），Files 窗口会自行打开到该画板，来源包括对话记录中的 canvas_show 卡片，以及窗口 **More** 菜单中针对 HTML 文件的 **Open on canvas**；其地址是窗口的地址，`#/s/<id>/files?path=<file>.canvas`。画板左上角切换 **Interact** 和 **Comment**，右上角是缩放控件，与图片查看器和集群地图相同（减号、级别、加号；滚轮在指针处缩放），而 **More** 中的 **Preview** 改为显示文件的 JSON。显示画板时，窗口不透明，其覆盖的聊天会去掉玻璃效果：移动画板上方或下方的模糊会在每一帧重新绘制。指南见[画布](../features/canvas.md)。

![浅色主题下 Files 窗口中的 .canvas 文件：同一页面的两个框架，画板下方的提示词栏](../../assets/canvas/canvas-board-light-1280.png)

*Files 窗口中的 .canvas 文件，浅色主题。*

## 工作区文件

**Files** 窗口像文档阅读器那样，在聊天上方显示会话的工作区：左侧是树，从树中打开的文件作为右侧的标签页。可通过聊天页头的 **Files** 按钮、`Ctrl+Shift+F`，或点击已发送消息中的工作区提及、文件工具的路径、共享文件的卡片，或回答中提到的路径来打开。其地址保留文件和一个可选行号，`#/s/<id>/files?path=notes/readme.md&line=15`，因此刷新后会在该处打开。按 Escape 或关闭按钮可返回聊天，并将焦点还给打开窗口的元素；无论停靠区显示什么，它都保留在窗口下方，在手机或平板上也是如此，点击窗口旁边会以同样方式关闭。重新打开时，窗口会显示此前打开的文件，每个文件停在其行号处，地址则指向当前显示的那个。

回答中提到的路径也以同样方式打开。无论写成内联代码（`internal/foo.go:12`）、链接目标，还是写在正文中（internal/foo.go），一旦服务器确认它是聊天工作区中的文件（**`POST /coddy/sessions/{id}/workspace/stat`**），它就会变成链接。该链接会在标签页中打开文件，若路径包含行号则定位到该行（`:12`、`#L12`），其地址就是窗口自身的地址，因此中键点击会在新的浏览器标签页中打开文件。正文中的路径需要包含文件夹和扩展名，因为正文里到处都是有斜杠的单词；内联代码则可以只是文件名（`README.md`、`Makefile`）。文件夹、工作区之外的路径、工作区中不存在的名称，以及仍在到达的回答中的所有路径，都保持原样，这类内联代码在点击时仍会复制自身。只有聊天中才会查询路径；文档和画布卡片会将它们显示为文本。

![回答中的路径是 Files 窗口的链接：带行号的内联代码路径、正文中的路径和一个链接](../../assets/web-ui/answer-file-links-dark-1280.png)

*回答中的路径：内联代码、正文和链接会在 Files 窗口的标签页中打开；不是文件的代码保持原样*

![Files 窗口：工作区树旁边有两个打开的文件，README.md 正以带行号的源代码形式显示](../../assets/workspace-files-window-dark-1280.png)

*聊天上方的 Files 窗口：树中文件夹在前，打开的文件作为标签页*

更多截图：[浅色主题，1280 px](../../assets/workspace-files-window-light-1280.png)和[手机，390 px](../../assets/workspace-files-window-dark-390.png)。

树先列出文件夹，再列出文件，并在打开每个文件夹时加载它。树上方的过滤器会**按名称搜索整个工作区**，包括没人打开过的文件夹，使用与输入框的 `@` 选择器相同的索引；按 Escape 清除。窗口的 **⋮** 菜单可显示隐藏文件（点文件、`node_modules`、`vendor`），让长行换行，在 SVG 或 HTML 文件绘制的内容与其源代码之间切换（**Preview**），并针对当前显示的文件重新加载、复制其路径或下载它。显示隐藏文件不会限制访问：显式的工作区路径仍然可以打开隐藏文件。符号链接和特殊文件会以 ↗ 列出，但不会打开。页头左侧的树开关可收起树，右侧的箭头可将窗口展开到整个屏幕。打开的文件在页面存续期间保持打开，因此关闭再重新打开窗口时，它们仍在原处。放不下的标签页可用滚轮或滑动横向滚动，当前显示的标签页会被带入视野。

打开的文件就是其内容，上面没有别的东西：标签页会命名它，因此窗口不再显示第二个名称、大小或时间，也不在文件上方切换视图。文本文件显示带编号的行窗口，行号与其文本对齐；地址或链接指定的行会滚动到该行，但不会标记，因为目前文件只能读取。长文件可连续滚动，不受旧的 512 KiB 文件限制：当您接近屏幕上内容的末尾时，会读取接下来的行，因此没有需要点击的分页。Markdown 与其他文本一样：逐行显示其源代码，因此其中的图片或 HTML 只是一行待读取的内容，不会加载或运行任何东西。栅格图像有适应大小和实际大小控件；超过 20 MiB 的图像仍可下载。

右键点击图片（触摸屏上长按）会打开一个菜单，其中包含 **Copy image** 和 **Save image**，此处以及 Web 界面显示图片的所有其他地方都如此：聊天中的附件或工具图片、图表、文档以及全屏查看器。**Copy image** 会将图片以 PNG 形式放入剪贴板，SVG 则按其自身尺寸绘制；只有在浏览器允许页面写入图片的地方才会提供此功能，而从另一主机通过纯 HTTP 提供的页面则不允许。**Save image** 会下载图片，即工作区中以其自身名称命名的文件。按 Escape 关闭菜单，并让下方的查看器保持打开。来自其他站点的图片会保留浏览器自带的菜单。

![Files 窗口，以及右键点击图片打开的菜单：Copy image 和 Save image](../../assets/workspace-files-window-image-menu-dark-1280.png)

*Files 窗口中图片的菜单*

源代码使用聊天代码块的语法进行着色，采用当前主题的颜色，因此聊天会着色的语言（Dart、Elixir、Haskell、Scala、PowerShell、`Dockerfile`、`Jenkinsfile`、`CMakeLists.txt` 等等）在窗口中也同样着色；语法由文件的扩展名或名称选择，从不猜测。屏幕上的行作为一个整体文本着色，因此跨行的块注释或字符串在每一行都保持其颜色。长于 4096 个字符的行，例如压缩后的 bundle，保持为纯文本；屏幕上超过 200,000 个字符时，文本将不着色显示。

![Files 窗口将工作区的 SVG 绘图显示为图片](../../assets/workspace-files-window-svg-dark-1280.png)

*SVG 以图片打开；⋮ 菜单中的 **Preview** 会将其切换为源代码*

**SVG** 以图片打开，**HTML** 文件以源代码打开；⋮ 菜单中的 **Preview** 会将此类文件切换到另一种视图，浏览器会为该类型的每个文件记住此选择。SVG 的图片是其字节的图像，因此其中的脚本永不运行，它链接到的任何内容也不会加载。HTML 页面会在沙箱框架中绘制，该框架不运行脚本、不提交表单、不打开窗口，也不从网络加载任何内容：其内联样式和嵌入的 `data:` 图片会显示，而脚本、样式表链接和相对图片不会显示，指向页面外部的链接也不起作用。无论主题如何，页面都在白色画布上绘制，就像浏览器标签页会显示的那样。长于 2,000,000 个字符的页面不会被绘制；其源代码仍会显示。

![Files 窗口在沙箱框架中绘制工作区的 HTML 报告](../../assets/workspace-files-window-html-preview-dark-1280.png)

*开启 **Preview** 的 HTML 页面：其自身样式生效，脚本永不运行*

音频和视频使用原生控件和字节范围，包括在已认证的远程服务器上以及通过集群中继，中继会将文件的临时签名地址传送到节点，但不为其背书（[集群](../operate/swarm.md)）。地址在一小时后过期；**Reload** 会续期。

**PDF** 会在浏览器自带的查看器中打开，即它用于从链接打开的 PDF 的查看器（计算机上的 Chrome、Edge、Firefox 和 Safari 都有）。其字节通过与图片相同的已认证读取器传输，最大 50 MiB，因此远程服务器和集群中继也可用。没有自带查看器的浏览器，例如 Android 上的 Chrome，会将 PDF 作为下载提供，窗口无法读取的 PDF 也是如此。其他二进制格式可以安全下载。

![Files 窗口在浏览器自带的查看器中显示一页 PDF](../../assets/workspace-files-window-pdf-dark-1280.png)

*浏览器自带查看器中的 PDF*

返回页面或完成一次工具调用时，会再次检查打开的文件和打开的文件夹。未更改的文件不会再次读取，因此文本保持在原处，声音或影片继续播放；已更改的文件会显示 *此文件已更改。预览已刷新。* 并从顶部重新开始，而不是混合不同版本的行。文件夹会保留 **Load more files** 添加的行，已消失的文件夹会折叠收起。过滤器只提供工作区中的文件，从不提供其上或其磁盘其他位置的路径。选定的远程环境会应用于文件预览和会话图片缩略图。在手机上，窗口一次显示树或文件：在树中选择文件后，文件会取代树的位置，树开关可将树带回。

## 聊天的编辑

聊天的编辑是 git 为其文件夹报告的内容：每个与 `HEAD` 不同、无论是否已暂存的已跟踪文件，以及每个 git 不忽略的新文件。不会按轮次记录任何内容，因此由 `edit` 工具、shell 命令或 Coddy 旁边的编辑器写入的文件读起来都一样，而不在 git 仓库中的文件夹没有可显示的编辑。PATH 上没有 `git` 二进制文件时，服务器通过其内置实现（go-git）作答；该实现不检测重命名，因此暂存的重命名会读作一次删除加一次添加。

- 数据来自 `GET /coddy/sessions/{id}/changes`。每个聊天一份副本（`changes/workingCopy.ts`）服务于显示它的每个视图：输入框上方信息条上的 git 计数和编辑窗口。它在聊天打开时、任意聊天的每一轮次之后（`GET /coddy/events` 上的 `event: turn_ended`，无论哪个界面运行了该轮次：聊天共享文件夹，因此另一个聊天的轮次也会推动这一个）、丢弃之后（`event: session_changes`）、一批工具调用完成后不久，以及页面重新获得焦点时读取，因为文件夹可能已在别处被编辑。读取失败时，屏幕上会保留上一次的结果，因为重启中的服务器不能看起来像干净的文件夹；编辑窗口在尚无内容可显示时会说明读取失败的原因。
- 最多读取 500 个新文件，每个 2 MB；编辑窗口会说明遗漏了多少个。嵌套在该文件夹中的仓库是一个条目，正如 git 列出的那样：其中的任何内容都不会被读取或丢弃。git 忽略的文件永远不会出现。
- 当会话在仓库的子文件夹中运行时，只报告该子文件夹，路径相对于它。

### 输入框上方的信息条

![链接 worktree 中运行中的聊天：与输入框顶部相连的信息条标明仓库、worktree 的分支，以及其右边缘带框的 git 计数](../../assets/workspace-bar-worktree-dark-1280.png)

*在链接 worktree 中运行的聊天的输入框上方信息条*

在第一条消息之前，信息条提供文件夹、分支和 worktree 作为选择（[按会话的工作区](#每会话工作区文件夹--分支--worktree)）；会话开始前没有 git 计数。聊天开始运行后，它工作的地方就是事实而非选择，信息条会标明它（`WorkspaceBar.tsx`）。信息条看起来像一条排队中的消息，并与输入框卡片的顶部边缘相连，卡片的顶部边框就是它们之间的接缝。它左侧标明仓库（链接 worktree 中为主检出目录的名称，工具提示中为完整路径）和分支；右边缘在文件夹有未提交更改时显示 git 的 `+A −D`。计数位于浅色边框中，因此看起来像一个按钮；悬停时边框和背景变亮，点击可打开编辑窗口。在链接 worktree 中运行的聊天会显示相同的分支图标，工具提示会标明 worktree 的文件夹。任何宽度下都没有 worktree 退出按钮或离开对话框。使用 `/worktree exit --keep`、`/worktree exit --remove` 或 `/worktree exit --remove --force` 返回，或让智能体使用 `worktree_exit`（[Git worktrees](../features/worktrees.md)）。智能体的 `worktree_open` 或 `worktree_exit` 一完成，信息条就会在轮次仍在运行时重新读取聊天的工作位置；在聊天的每一轮次之后也会如此，因此无需刷新即可跟随这些工具和 `/worktree` 命令。信息条是 git 的：在不属于任何 git 仓库的文件夹中运行的聊天没有信息条，其输入框卡片保留圆角顶部。计数的右边缘与下方改进提示词的魔杖对齐；在手机或触摸屏上，计数保持纤细外观，并在其上叠加一个不可见的 40px 点击区域。

### 编辑窗口

编辑只有一个视图：聊天上方的窗口（`EditsView.tsx`），在一个可滚动文档中容纳文件夹未提交更改的所有差异。它的框架和页头与 [Files 窗口](#工作区文件)相同，位置相同，与浏览器窗口边缘的距离也相同：页头左侧是树开关和标题 **Edits**，其下是文件夹名称和 git 的 `+A −D`；右侧是 **⋮** 菜单、展开按钮和关闭按钮。菜单包含 **Side by side**（分栏视图，在手机或平板上不提供，那里每个差异都内联显示）、**Collapse all diffs** / **Expand all diffs** 和 **Discard all**。输入框上方信息条上的 git 计数可打开窗口；Escape、关闭按钮或点击其旁边会收起窗口。其地址为 `#/s/<id>/changes`，因此刷新后会再次打开。编辑没有停靠区界面：聊天旁的停靠区容纳后台任务，并在窗口下方保持原样。

左侧的树只列出已更改的文件，即 git 在它们所在文件夹中报告的未提交更改，每行末尾带有 git 的字母（**A** 新增，**M** 修改，**D** 删除）。差异按相同顺序排列。其过滤器按路径缩小列表；在那里选中的文件会将差异滚动到它，并且在其上方差异仍在加载时保持不动；滚动时，树会标记顶部的文件，当列表很长时，树自身会滚动以保持该行可见。在手机上，窗口打开时显示差异，树开关会将列表覆盖到差异上。

文件章节有一个粘性页头：名称可折叠差异，然后是复制路径和丢弃，始终可见，文件末尾右侧是其 `+a −b`。行号所在的单元格宽度与文件中最长的行号一致，因此每一行的代码，包括删除行或添加行，都从同一边缘开始；在变更内部，删除行在前，添加行在后，正如 git 的写法。块之间是无文字的间隔，而不是 `N unmodified lines` 填充行。在无可显示的文件夹中会说明：干净，或不在 git 仓库中。

![编辑窗口：左侧树中为已更改文件，右侧每个已更改文件一个章节](../../assets/session-changes-review-window-dark-1280.png)

*文件夹未提交更改上方的编辑窗口：已更改文件的树，旁边是一个可滚动文档中的所有差异*

文件页头中的 **Discard** 会将该文件恢复到 `HEAD`，菜单中的 **Discard all** 会丢弃每一项更改。两者都会先在共享确认对话框中询问，该对话框会说明文件是回到上次提交的内容，还是如果它是该提交之后新增的，则会被删除；在那里按 Escape 会回答问题并让窗口保持打开。然后页面会 POST `.../changes/revert`，带有 `{"paths":[...]}` 或 `{"all":true}`：已跟踪文件会从 `HEAD` 取回其内容和索引条目，`HEAD` 不包含的文件会被删除，连同它留下的任何空文件夹，git 忽略的文件永远不会被触碰。在该文件夹中运行的轮次，无论在本次聊天还是在该处工作的另一个聊天中，都会拒绝丢弃（**409**）；git 不再报告的文件（在此期间已提交或已放回）也是如此，此时请求的任何部分都不会应用。不提供提交：窗口只读取，丢弃是它唯一的写入操作。

代码由 `lowlight` 着色（highlight.js 位于树式 API 之后，与 `rehype-highlight` 提供给 Markdown 渲染器的是同一引擎），因此样式表中已有的 `hljs-*` 样式会生效，查看器从不注入标记。语法通过 `diffLanguage.ts` 从文件扩展名得出；未知扩展名会渲染为纯文本，而不是进行猜测。

详情路由只读取它被请求的文件（`gitws.WorktreeChangeFor`）：查看器一次加载一个补丁，而每次请求都解析整个集合意味着每次请求都要为每个已更改文件启动一个 git 子进程。新文件只有在 git 命名它之后才会被读取，因此该路由不能指向任意文件。当 git 的报告发生变化时，窗口只会重新读取状态发生变化的文件。

自动化检查：

- **external/ui/src/ui/changes/workingCopy.test.tsx**（聊天的每个视图只读取一次；根据服务器的消息、工具调用和页面的焦点进行读取；读取失败；不在仓库中的文件夹；另一个环境）
- **external/ui/src/ui/chat/WorkspaceBar.test.tsx**（仓库、分支及其工具提示中的 worktree、计数及其语言）
- **external/ui/src/ui/changes/sessionChangesText.test.ts**（路径拆分）和 **plurals.test.ts**（按区域设置的文件计数）
- **external/ui/src/ui/changes/diffRows.test.ts**（统一和分栏行构建、不均匀的片段、块间间隔）
- **external/ui/src/ui/changes/fileTree.test.ts**（目录分组和单子节点链折叠）
- **external/ui/src/ui/changes/EditsView.test.tsx**（Files 窗口的框架和页头，菜单：并排、全部折叠、全部丢弃；仅已更改文件的树及其过滤器；滚动到并标记的选中项；复制路径、跳过横幅、无 git 提示、着色开启和关闭、仅重新读取已移动的文件、在提问后丢弃一个文件及其之后的所有内容）
- **external/ui/src/ui/changes/diffLanguage.test.ts** + **highlightLine.test.ts**（语法选择，以及着色会精确重现该行）
- **internal/linediff**（统一差异和行统计；线性空间中的 Myers O(ND) 算法，因此大文件中的分散编辑仍保持为分散编辑，旧的 LCS 表在超过大小上限后不得不弃用，并将此类文件报告为整体重写。搜索受 `snakeBudget` 限制，只有既巨大又几乎完全不同的配对才可能耗尽；该配对会回退为整体替换）
- **internal/gitws**（`changes_test.go`、`backend_test.go`：在 git 二进制文件存在以及从 PATH 中隐藏时分别测试每个场景：工作副本状态、未跟踪文件和读取上限、.gitignore、保留空白的 blob 读取、嵌套仓库、丢弃文件、子文件夹或全部内容，以及在触碰任何内容之前拒绝更改集之外的路径；仅在二进制文件存在时测试重命名）
- **features/session_changes.feature**（端到端测试仓库：一轮次编辑文件并由 git 报告，智能体之外进行的编辑也是如此，查看器读取差异，先丢弃一个文件，然后丢弃全部内容）

### 子智能体对话记录

子会话是只读的：`GET /coddy/sessions/{id}/messages` 返回 `subagent {parentSessionId, name, taskId}` 和 `readOnly: true`，针对它的每个提示词都会被 409 拒绝。SPA 会读取这两个字段（普通会话中不存在），使用常规消息渲染器渲染对话记录，并用通知替换输入框（`SubagentReadOnlyNotice`）："子智能体 `<name>` 的只读对话记录。提示词请发送到父聊天。"，并带有指向 `#/s/<parentSessionId>` 的 **Open parent chat** 链接。其下方的行会标明子会话的有效模型及其括号中的推理级别，`codex/gpt-6.1-sol (High)`，以及是由 spawn、定义还是模型默认值选择的（消息中的 `selectedReasoning` 会回答）；一个提供级别但未选择级别的模型会显示 `(Default)`，即提供商自己的选择，而没有级别的模型没有括号。重试、消息编辑以及计划卡的 **Run plan** / **Discard** 在此类会话中会被隐藏（处理函数根本不会传入，因此 `plan_document` 卡片渲染时没有页脚，其 Markdown 编辑器为只读），聊天页头显示 "Subagent `<name>`"，因为子会话没有 History 行来命名它。子会话在 History 中隐藏，子会话 id 没有任何特殊之处，因此 shell 会获取从 Tasks 面板或通过 URL 打开的任何 id，并让消息端点作答。

自动化检查：

- **external/ui/src/ui/chat/subagentTranscript.test.ts**（标记解析，裸 `readOnly`）
- **external/ui/src/ui/chat/SubagentReadOnlyNotice.test.tsx**（有名称和无名称的文案、父链接 href、同标签页打开与修饰键点击）
- **external/ui/src/ui/chat/ChatScreen.test.tsx**（在停靠布局和主视觉布局中通知替换输入框）
- **external/ui/src/ui/chat/subagentReadOnlyCss.test.ts**（通知和链接使用主题令牌）
- **external/ui/src/ui/chat/PlanDocumentSection.test.tsx**（没有操作处理函数的卡片没有页脚、编辑器只读且不自动保存）
- **external/ui/src/ui/messages/MessageList.test.tsx**（只读对话记录上的计划卡渲染时不带 Run plan 和 Discard）
- **external/ui/src/ui/i18n/messagesParity.test.ts**（新键存在于每个字典中）
- **external/ui/src/ui/settings/settingsSections.test.ts**（`subagents` 配置标签页的已翻译标签和简介，该标签页是保留其 schema 键的混合标签页）
## 实时 token 用量

- 智能体工作时，UI 必须显示 token 计数器。
- 收到 SSE 事件 `token_usage` 时更新计数器。
- 更新粒度以完成的后端模型调用为单位，而不是每生成一个 token。
- UI 在重启后通过 `GET /coddy/sessions/{id}/stats` 恢复 token 计数器。

## 提供商账户用量

![上下文计数器下方的用量弹出框](../../assets/ui-usage/usage-popover-dark-1280.png)

*上下文计数器下方的用量弹出框*

![提供商行上的用量面板开关](../../assets/ui-usage/settings-usage-panel-dark-1280.png)

*提供商行上的用量面板开关*

![某个窗口已用尽：输入框会报告重置时间](../../assets/ui-usage/usage-blocked-dark-1280.png)

*某个窗口已用尽：输入框会报告重置时间*

![用量部分中的 Codex 订阅](../../assets/subscription-usage/codex-usage-dark-1280.png)

*用量部分中的 Codex 订阅：带时长标签的窗口和一项按功能划分的条目*

![用量部分中的 Devin 订阅](../../assets/subscription-usage/devin-usage-dark-1280.png)

*用量部分中的 Devin 订阅：每日和每周配额计量条*

![手机上的 Codex 用量部分](../../assets/subscription-usage/codex-usage-dark-390.png)

*作为手机底部面板的 Codex 用量部分*

![手机上的 Devin 用量部分](../../assets/subscription-usage/devin-usage-dark-390.png)

*作为手机底部面板的 Devin 用量部分*

- 当选中的模型的提供商报告账户用量时（目前为 `neuraldeep`、`codex` 和 `devin`），**上下文弹出框**（发送按钮旁的上下文圆环）末尾会有一个**用量部分**，方式类似 Claude Desktop 在上下文窗口下列出套餐限制：提供商与套餐（品牌旁的行显示 **`Codex · codex-work · Plus`**，除非该行按其类型命名，以便区分同一类型的多个配置），每个计量窗口一个计量条，附带按浏览器时钟显示的重置时间、以 UI 语言显示的标签和已用百分比，钱包以卢布显示，并在有变化时给出说明：达到限制及其重置时间（或无法通过等待解除的封锁原因）、模型处于提供商的无限量选项、登录被拒绝、读取过期、某一轮次正在等待重置。显示的窗口是来源所报告的窗口：NeuralDeep 的会话/周/日窗口，Codex 的带时长标签的窗口（`5h`、`week`，以及诸如 `Fast model · week` 的按功能划分的条目），Devin 的 `day`/`week` 配额计量条或其 `ACU` 计量条；没有配额可报告的来源会显示套餐并附上 `quota unavailable` 说明，绝不会凭空造出计量条。只有 NeuralDeep 会在日计量条为零时将其隐藏并报告钱包。达到 80 % 时，计量条变为琥珀色，输入框上方会出现一条**横幅**，内容为 `You've used 85% of your NeuralDeep 3h limit · resets 20:59`，可按提供商行、窗口和周期关闭；发生封锁时，横幅变为错误色调：定时封锁显示 `Usage limit reached · Resets 20:59`，空钱包、被封锁的密钥或账户以及速率限制会说明其原因；智能体等待重置时，横幅显示 `Usage limit reached · Auto-resuming at 20:59`。横幅在所有主题下都不透明，因此输入框下方滚动的聊天内容绝不会透出横幅；当它出现在一个原本停在最新消息的聊天上方时，聊天会上移，以保持该消息可见。
- 该行的**用量限制面板**开关位于设置 → LLM 提供商（`providers[].usage_limits_panel`，默认开启），它会隐藏该部分和横幅，并停止其背后的读取：此时路由对 `unsupported` 返回 `disabled: true`，钩子会丢弃它为那一行显示的快照。
- 数据来自 **`GET /coddy/providers/{name}/usage`**（会话打开、模型切换、所查看会话每完成一轮后、某个窗口重置后读取一次、服务器推迟刷新时读取一次缓存），以及轮次之间来自 **`event: provider_usage`**、位于 **`GET /coddy/events`** 上的数据。除此之外不会轮询；快照按服务器的读取时间排序，因此较慢的响应绝不会带回更旧的数字。视觉契约：**`DESIGN.md`**（**上下文弹出框用量部分和用量横幅**）；设计记录 **`docs/plans/neuraldeep-usage.md`**。

## Markdown 渲染

- 通用工具输出保持原始等宽文本；受支持的文档和结构化卡片会在 **`ToolCallMessage`** 内渲染 Markdown 或字段。
- **用户**消息为纯文本，保留换行（**`UserMessage`**）。
- **助手**消息可包含 Markdown。
- UI 渲染 Markdown，支持围栏代码块和语法高亮。
- 扩展语言注册表覆盖 [63 种语言的 NeuralDeep 审计](../contributing/syntax-highlighting-audit.md)，包括 Pascal/Delphi、GML、汇编、PowerShell、GDScript、HLSL、WGSL、COBOL 和 VBA。确切标签和限制请参阅该审计：PL/SQL/OpenCL/CUDA 使用基础语言着色，而 UnrealScript/TADS/URQ 仍为纯文本。捕获的响应会在无凭据的情况下离线测试。
- `vue` 围栏会将组件标记以及普通 `<script>` / `<style>` 内容按 JavaScript / CSS 高亮。Vue 插值以及 `lang="ts"`、SCSS 或其他预处理器没有专门的 Vue 感知解析。
- `postcss` 围栏使用 CSS 高亮器，包括选择器、属性、数字和注释。插件特定的 PostCSS 语法可能保持未着色。
- 用语言标记围栏（例如 `js`、`css`、`html`、`json`、`ts`、`python` 或 `go`）即可启用高亮。未标记或不受支持的语言保持纯文本。答案流式传输时高亮也有效。
- 切换主题时，代码颜色会立即跟随全部八种外观主题。每个主题都会定义共享的 **`--syntax-*`** 调色板，位于其 `external/ui/src/styles/themes/` 下的文件中；无需单独的语法主题设置。
- 每个代码块都有一个复制按钮，仅复制该代码块的内容。
- Mermaid 和 SVG 围栏会绘制为图片，LaTeX 会排版为公式：参见 [图表与公式](#图表与公式)。
- 来自其他站点的图片或视频（`![](https://...)`）在消息渲染时不会被获取：它会显示为一个按钮，带有其 alt 文本和主机名，**从 <host> 加载图片**，点击后才会加载。若立即渲染，浏览器会在答案一出现时就发送该地址及其路径或查询参数所携带的内容，而答案可能遵循来自 README、网页或工具结果的指令。本服务器的图片、相对路径图片和 `data:` 图片会立即加载。canvas 卡片也是如此；随 Coddy 一起提供的文档阅读器会立即加载其截图和视频。文本中包含形状数据的 Mermaid 块（`A@{ img: "https://..." }`）在绘制时可能获取某个地址，因此它会显示源码和 **绘制图表**；其他所有图表都会立即绘制，并且回复的指令或 front matter 不能设置字体、主题、样式表或 HTML 标签。

![Mermaid 授权提示](../../assets/web-ui/mermaid-consent-dark-1280.png)

*带有外部图片的图表会等待点击后才绘制。*

## 图表与公式

包含图表或公式的答案会以绘制后的形式显示，而不是源码。这涵盖智能体在被要求“画流程图”或“显示公式”时写出的内容：Mermaid 围栏、SVG 围栏，以及美元符号之间或 LaTeX 自身定界符之间的 LaTeX。Web 界面和 Telegram Mini App 会绘制它们；控制台、ACP 客户端和即时通讯机器人保留文本。

![答案中绘制的 Mermaid 流程图，带有 MMD 和 SVG 按钮以及其头部的源码开关](../../assets/web-ui/diagrams-flowchart-dark-1280.png)
*答案中的 Mermaid 围栏：默认绘制，源码只需点击一下。*

![一个长 Mermaid 图表的源码，在其框内滚动，源码开关点亮，复制按钮位于文本角落](../../assets/web-ui/diagrams-source-dark-1280.png)
*按下源码开关：图表的文本及其复制按钮。*

- **绘制内容。** 标记为 **`mermaid`** 的围栏（流程图、时序图、类图、状态图、ER 图、甘特图、饼图、思维导图以及 Mermaid 绘制的其他内容）和标记为 **`svg`** 的围栏。任何其他标签仍为代码块：一个 **`xml`** 围栏即使其中包含 `<svg>`，也不会被绘制。
- **该块。** 头部行会标明类型（**Mermaid 图表**、**SVG 图片**），右侧带有可保存的格式和源码开关。**MMD** 保存 Mermaid 源码（**`diagram.mmd`**），**SVG** 保存绘制后的图片（**`diagram.svg`**）；SVG 围栏只有 **SVG**，用于保存其源码（**`image.svg`**）。源码开关是一个尖括号图标，默认关闭，因此看到的是图片；按下后会显示源码，复制按钮位于文本右上角，源码超过约二十行时会在其框内滚动。点击图片会在图像查看器中打开，支持缩放和平移。
- **错误。** Mermaid 无法解析的图表会显示其源码，并在头部下方给出原因（**无法绘制图片：第 3 行解析错误：... 预期 ...，实际为 ...**），源码开关保持按下并禁用。如果渲染器完全无法加载（离线、页面比二进制更旧），该块会显示源码并提示 **无法加载图表渲染器：请重新加载页面**。
- **流式传输。** 仍在到达的围栏会在首次出现时绘制，然后在文本 300 毫秒未变化后再次绘制，因此不会在半写图表每个 token 上运行 Mermaid。
- **主题。** 图表从当前外观获取所有颜色，并在主题更改时重新绘制：节点使用强调色着色，边使用文本颜色，饼图扇区、图表系列和 git 分支使用主题的强调色和语法色，且每种颜色都与表面保持足够距离，使扇区突出并且其标签可读（4.5:1）。这在全部八种主题中都成立。SVG 按原样绘制。

![深色主题中绘制的甘特图和饼图，扇区使用主题的强调色和语法色](../../assets/web-ui/diagrams-charts-dark-1280.png)
*图表采用主题颜色：深色主题...*

![浅色主题中相同的甘特图和饼图](../../assets/web-ui/diagrams-charts-light-1280.png)
*...以及浅色主题。*

- **安全。** 图片是一个 `<img>`，其中 SVG 作为 `data:` URL：以这种方式绘制的 SVG 不运行脚本、不加载任何网络内容，也无法为页面设置样式，因此模型编写的 SVG 会按原样显示。Mermaid 以 **`securityLevel: "strict"`** 运行。

![由 KaTeX 排版的行内和显示公式，每个显示块上都有源码开关](../../assets/web-ui/diagrams-formulas-dark-1280.png)
*行内和显示公式；同一段落中的价格仍为文本。*

- **公式。** **`$...$`** 和 **`\(...\)`** 是行内公式；**`$$...$$`**、**`\[...\]`** 和 **`math`** 围栏是显示公式。KaTeX 会排版它们（不使用 **`\href`**、**`\url`** 或 **`\includegraphics`**：不受信任的命令已关闭），并在页面中自行构建公式节点（**`katex.render`**），因此绝不会解析由答案生成的 HTML 字符串。显示公式与图表有相同的头部，但只有源码开关（源码视图带有复制按钮），当宽度超过对话记录时会横向滚动。行内公式会将源码显示为工具提示，并在点击（或 Enter / Space）时复制它（包括美元符号）。引用或列表项中的 **`\[...\]`** 会留在其行内（按行内排版），而不会破坏引用或列表，围栏代码或缩进代码中的 LaTeX 定界符保持原样；一对定界符绝不会跨越空行、新列表项或引用深度变化。正文中的转义方括号（`a\[0\]`、`\(see below\)`）保持为文本。
- **不是数学的美元符号。** 单个美元符号的跨度只有在以下情况才是公式：开头美元符号后没有空格，结尾美元符号前没有空格，且两侧外侧没有字母或数字接触任一美元符号（Pandoc 规则），并且它不以 **`${NAME}`** 开头。因此，“它花了 $5 和 $10”、**`${CODDY_HOME}/hooks.json`**、**`$HOME$PATH`** 以及 argon2 哈希都保持为文本。代码跨度和围栏内的美元符号永远不是数学。错误的 TeX 会显示 KaTeX 自己的红色源码，而不是错误提示。
- **智能体知道。** Web 界面发送的每一轮都会标明它（**`metadata.surface: "webui"`**），服务器会在该轮的系统提示词中添加一个块：Web 界面会绘制 **`mermaid`** 和 **`svg`** 围栏并排版 LaTeX，因此当图表、图形或公式比文字表达更多时，它们就是答案，颜色则交给主题处理。该块属于该轮，绝不会被存储；控制台、远程控制台、即时通讯机器人和 API 客户端不会获得它，并继续用文本回答。一个会话若在 Web 界面和另一个界面之间轮流使用，则会在切换的那一轮因缓存未命中而付出代价。
- **位置。** Web 界面渲染 Markdown 的所有位置：答案、思考、工具卡片、计划卡片和文档阅读器。
- **加载。** Mermaid 和 KaTeX 不属于 **`app.js`**。它们是 **`/chunks/`** 下的单独文件（带内容哈希的名称，像 SPA 其余部分一样嵌入二进制），在页面首次显示图表或公式时获取，之后由浏览器永久缓存。既没有图表也没有公式的聊天不会下载任何额外内容。
- **代码。** **`ui/markdown/DiagramBlock.tsx`**（该块）、**`ui/markdown/pictureRender.ts`**（Mermaid 的配置、主题调色板、SVG 尺寸和缓存）、**`ui/markdown/MathFormula.tsx`**（KaTeX）、**`ui/markdown/mathDelimiters.ts`**（LaTeX 定界符和美元符号规则）、**`ui/markdown/renderers.ts`**（按需导入）。测试：**`ui/markdown/DiagramsAndMath.test.tsx`**、**`ui/markdown/mathDelimiters.test.ts`**、**`features/web_ui_diagrams.feature`**；浏览器检查：**`npm run check:diagrams`**（参见 [检查图表和公式](#检查图表和公式)）。

## Markdown 行编辑器（共享）

实现为 **`MarkdownLineEditor`**（`external/ui/src/ui/markdown/MarkdownLineEditor.tsx`）。用于：

- 调度器任务 **`body (markdown)`**（`SchedulerJobEditorSheet`，默认 **`minRows`** **10**）。
- 计划文档卡片的 Markdown 模式（`PlanDocumentSection`，**`minRows`** **4**，类 **`md-line-editor--plan`**）。

行为（参见 **`DESIGN.md`**，**Markdown 行编辑器**）：

- 占满父容器宽度；编辑器高度随内容变化（最小逻辑行数）；内部 **`textarea`** 上没有滚动条。
- 行号槽为每个**逻辑**行显示一个编号（以 `\n` 分隔）。换行后的视觉行会在行号槽中留下**空白**单元格（不重复编号）。
- 光标所在逻辑行：高亮覆盖该行的**所有**视觉行；活动行号着色。
- 换行测量使用隐藏探针，其字体和文本宽度与 textarea 相同；视觉行数 = **`ceil(height / lineHeight)`**。
- 长的不可断 token 会换行（**`overflow-wrap: anywhere`**）；编辑器内部没有水平滚动。

自动化检查：

- `external/ui/src/ui/markdown/MarkdownLineEditor.test.tsx`
- `external/ui/src/ui/markdown/markdownLineGutter.test.ts`

## 计划文档卡片（计划模式对话记录）

对话记录类型 **`plan_document`** 会在主聊天栏（而不是右侧栏）中渲染 **`PlanDocumentSection`**。

数据与 API：

- 持久化在 **`messages.json`** 中；水合字段包括 **`slug`**、**`name`**、**`overview`**、**`content`**，可选的 **`body`**、**`path`**、**`discarded`**。
- 正文编辑：使用 **`PUT /coddy/sessions/{id}/plans/{slug}`** 和 **`{ "body": "<markdown>" }`**（防抖自动保存）。
- 丢弃：**`DELETE /coddy/sessions/{id}/plans/{slug}`** 设置 **`discarded: true`**；卡片仍然可见，控件禁用。
- 运行计划：客户端触发实现运行（元数据 / 提示词；参见 **`docs/reference/acp-protocol.md`**）。

UI 要求：

- 折叠时：标题、单行描述、页脚中的 **丢弃** 和 **运行计划**；标题 **`title`** 的工具提示 = 已知时的计划文件绝对路径。
- 展开时：默认 **预览**（通过 **`Markdown`** 渲染 Markdown）；眼睛开关切换到 **`MarkdownLineEditor`**。
- 对于预览和 Markdown，内容窗格都会随文档长度增长（窗格上**没有**内部最大高度滚动）。
- 展开的桌面端（**`min-width: 640px`**）：标题行和操作按钮共享顶行；正文在下方占满宽度。
- 编辑器正文排除 YAML front matter（客户端 **`planEditorBody`**）；预览使用相同的正文文本。
- 只读对话记录（子智能体子会话）：**`MessageList`** 既不传递 **`onPlanDocumentRun`** 也不传递 **`onPlanDocumentDiscard`**；此时卡片渲染时不带页脚（**`.plan-document-card--readonly`**），Markdown 编辑器为只读，且不会安排自动保存。每个页脚按钮仅在其自身处理程序存在时出现。

自动化检查：

- `external/ui/src/ui/chat/PlanDocumentSection.test.tsx`
- `external/ui/src/ui/messages/MessageList.test.tsx`（处理程序转发、只读对话记录）

## 对话记录下方的待办清单

![最新消息下方的待办清单：两项已完成，进行中的项已标记，一项待处理](../../assets/web-ui/todo-checklist-running-dark-1280.png)

*最新消息下方的待办清单*

当会话保留的待办清单中仍有未完成工作时，对话的最新一端会显示整个列表，位于实时状态行下方、输入框上方。每一项都带有其工具行使用的标记：已完成项为勾选方框，待处理项为空方框，进行中项为半填充圆。进行中的项还带有自己的静止标记，即一个内带圆点的强调色圆环，因此列表能说明智能体当前所在项，而不会分散注意力。桌面端卡片宽度与其文本相同，且在其他卡片宽度范围内；手机端则占满整列。无论折叠还是展开，卡片都使用对话记录的行间距，不增加额外间距。

- **列表在存在待处理或进行中项时显示。** 一旦所有项都已完成、失败或取消，它只保留到关闭最后一项的那一轮结束：已完成列表的空闲聊天不显示任何内容，尽管会话会保留该列表，直到智能体替换或归档它。
- **每次读取对话记录时都会附带列表**（**`todoPlan`**，属于 **`GET /coddy/sessions/{id}/messages`**），因此重新加载的标签页会立即显示它，运行中轮次的 **`plan`** 帧会在智能体勾掉项时移动它。设计计划若由 **`plan_write`** 在同一事件上发布，则不是该清单，不会在此显示。
- **点击列表头部**（**待办列表**，带有已完成项数量）会将列表折叠到仅显示进行中的项；若没有进行中的项，则只保留头部。再次点击会展开，浏览器会为每个聊天保留该选择。
- **状态变化、新项、折叠和列表离开都会平滑移动，而不是跳变。** 在减少动态效果下，每个状态都会立即出现。
- **列表属于对话记录的最新一端：** 读者向上滚动到长会话的较早页面后，一旦回到末尾就会再次看到它。
- 自动化检查：**`messages/TodoChecklistMessage.test.tsx`**、**`messages/todoChecklistCss.test.ts`**、**`App.todoChecklist.test.tsx`**、**`features/todo_checklist_web_ui.feature`**。
## 待办清单（工具卡片）

![待办工具的清单：一次调用替换计划，然后一项标记为进行中](../../assets/todo-tool-preview-dark.png)

*待办工具的清单：一次调用替换计划，然后一项标记为进行中*

- **`coddy_todo_*`** 工具将其清单显示为对话记录行的预览（**`chat/todoToolPreview.ts`**）。Web 界面不读取 **`GET /coddy/sessions/{id}/plan`**（**`PUT`**、**`POST .../plan/archive`**）：对话记录下方的列表随对话记录读取一起返回，而这些路由保留给其他客户端。
- 与上方的 **`plan_document`** 对话记录卡片不同。

## 长期记忆

记忆树根目录

- `global`
- `workspace`

树 API

- `GET /coddy/sessions/{id}/memory/tree`
  - 不带 `root` 时返回根列表。
  - 带 `root` 和可选的 `path` 时列出子项。
- 仅列出 `.md` 和 `.txt` 文件。
- 必须拒绝路径遍历。

文件 API

- `GET /coddy/sessions/{id}/memory/file` 读取。
- `PUT /coddy/sessions/{id}/memory/file` 写入。

## MCP 服务器（设置标签页）

设置 -> MCP 服务器标签页的功能清单（`MCPSection.tsx`，章节类型 `mcp`；视觉契约见 `DESIGN.md`）。截图：`docs/assets/screenshot-fullhd-settings-mcp.png`（一个已连接的全局服务器，加上一个等待工作区批准的项目本地服务器）：

- **`GET /coddy/mcp`** 支撑该列表：合并后的全局 `~/.coddy/mcp.json` + 项目 `./.coddy/mcp.json` 服务器（config.yaml 未声明任何服务器），每一项都有 `source`（`global` / `local` 作用域徽章）、`origin`（`home` / `project`）和 `source_path`（真实文件，即徽章工具提示所指的名称）、探测结果 `status`，以及其工具清单。
- 每个服务器的状态圆点：已连接（绿色）、错误（红色，工具提示显示探测错误）、已禁用（灰色）、未知传输类型（琥珀色，`unsupported`）、等待工作区批准（琥珀色，`needs_approval`）、被 `mcp.project_trust: deny` 拒绝（红色，`denied`）。
- 一行以其折叠箭头和状态圆点开头，自身没有其他字形符号；展开后的工具和信任说明从名称所在位置开始。在手机上，控件会换行到名称及其命令行下方，因此名称能完整显示，而不是在四个 40px 按钮旁只剩几个字母。
- 该标签页包含**两个字段集**：**MCP 发现**（`.mcp-discovery-box`）位于 **MCP 服务器**（`.mcp-servers-box`）上方。发现区承载 `mcp.project_trust` 策略（`mcp-project-trust` 选择器，`POST /coddy/mcp/project-trust`）以及为什么项目条目受到门控的说明；它本身不是独立的设置章节，因为它恰好管理其下所列的服务器，并且与该标签页的其余部分一样，在更改时即持久化，而不加入“全部保存”。
- 项目本地行的工作区信任（`gated: true`）：盾牌按钮（`mcp-trust-{name}`）会提交 `POST /coddy/mcp/{name}/trust|untrust`，该批准会携带该行的 `fingerprint`，因此自列出以来被检出内容重写过的声明会被拒绝（`409`，显示为标签页的错误）而不是被批准；`needs_approval` 行会带有说明（`mcp-trust-note-{name}`），其中包含声明它所在的 `source_path`，以及批准所覆盖的声明（`.mcp-trust-facts`，来自 `declarationFacts`，位于 `mcpServerJson.ts` 中）：传输、`runs`（command + args）或 `contacts`（url）、环境变量和请求头的**名称**、其值读取的 Coddy 进程变量（`reads`，来自该行的 `reads`）以及工作区。值永远不会被渲染。盾牌**仅在 `ask` 下**渲染（`showsTrustControl`，位于 `mcpServerJson.ts` 中）：`allow` 无论如何都会启动每个项目服务器，而 `deny` 一个也不启动，因此没有可按服务器提供的决定。这样的行不会被探测，因此不列出工具；命令行保持可见，因为它是操作者批准的内容。`global` 行没有盾牌，且在 `denied` 下禁用。
- 服务器开关切换 `POST /coddy/mcp/{name}/enable|disable`；全局条目的开关会持久化到定义它的文件中，项目条目的开关会持久化到 `<home>/mcp-overrides.json`，而不改动检出内容。活动会话会连接或关闭那一个服务器，并让其他服务器继续运行。
- 服务器无法构建的列表（`GET /coddy/mcp` 返回错误，或请求从未到达）会在列表上方报告（`mcp-load-error`），并附上服务器消息；当 `mcp-overrides.json` 无法解析时，该消息会指明文件，而不是显示为“未配置服务器”；失败的刷新会保留标签页已显示的行。
- 展开一行会列出带有逐工具开关的工具（`POST /coddy/mcp/{name}/tools/{tool}/enable|disable`）；服务器禁用时，工具开关会锁定。
- 每一行都可编辑，两个作用域皆然。删除会调用 `DELETE /coddy/mcp/{name}`，编辑会就地打开 JSON 编辑器卡片，并将作用域固定为所属文件。
- 列表不携带环境变量或请求头值，仅在其位置显示 `<redacted>`，因此该行的编辑器会从占位符开始，并在 JSON 下方说明它们的作用（`mcp-editor-values-hint`）：保持原样的项保留存储值，新值会替换它，删除的键会被删除。保存会发送该行的 `fingerprint`（`PUT /coddy/mcp/{name}?scope=…&fingerprint=…`）；如果此时文件中已有另一个声明，服务器会返回 `409`，卡片保持打开并显示 `mcp.error.saveChanged`，列表会重新读取，且该条目必须重新打开才能编辑。
- 添加服务器会打开编辑器，预填 Cursor 风格的条目模板和本地/全局作用域选择器（默认本地）；保存会在客户端验证后发出 `PUT /coddy/mcp/{name}?scope=local|global`（`mcpServerJson.ts`：JSON 对象，需要 `command` 或 `url`，名称不得包含 `__`、空格或路径分隔符）。
- 刷新会通过 `GET /coddy/mcp?refresh=1` 重新探测所有服务器。
- 列表刷新从不会卸载列表（仅初始加载时显示占位符），因此抽屉的滚动位置得以保留。
- 该标签页不参与设置文档的“全部保存”流程。

## 技能市场（设置标签页）

**设置 > 技能** 保留其表单（自动发现、`skills.dirs`，以及位于其独立 **市场发现** 字段集中的 `skills.project_trust`，随其余设置一起保存）和已安装技能，并在两者之间有一个 **市场** 字段集（`MarketplacesEditor.tsx`，辅助函数在 `settings/marketplaces.ts`）。与 MCP 标签页一样，它由 API 驱动，从不属于设置文档：它列出的内容声明在 `~/.coddy/marketplaces.json`、项目的 `.coddy/marketplaces.json` 和 Coddy 自身中，通过 `GET /coddy/skills/sources` 为屏幕上显示的会话读取（`X-Coddy-Session-ID`），并且每个操作都会立即生效。

- 每个条目一行，按条目生效顺序排列：键（市场的名称、源的地址）、类型徽章（源为 **所有插件**，市场为 **目录**，其地址位于名称下方）和来源徽章（**内置**、**你的**、**来自项目**），其工具提示会指明声明文件。
- 内置目录 `rpa-skills`（其地址 `EvilFreelancer/rpa-skills` 位于名称下方）带有一个不可点击的盾牌（始终受信任）和一个禁用的删除按钮；它会像任何行一样检查更新。
- 在 `skills.project_trust: ask` 下，项目行带有 MCP 盾牌：条目等待批准时为琥珀色，批准后为普通样式。点击它会提交 `POST /coddy/skills/sources/trust` 并携带该行的 `fingerprint`（自那以后被检出内容重写过的条目会以 409 拒绝，消息显示在列表上方），对于已批准的条目则提交 `untrust`；无论哪种情况之后都会重新读取列表。仍在等待的行会变暗，说明哪个文件声明了它，并且在批准前既不会读取也不会安装，其检查按钮处于禁用状态。在 `deny` 下，其说明表示策略已将其关闭，并且没有盾牌；在 `allow` 下，项目行生效，也没有盾牌。
- 一行的检查按钮会检查该行，`POST /coddy/skills/sync?source=<key>`；**市场** 标题旁的刷新图标会检查每一行，方式类似于提供商的 **模型** 列表刷新，`POST /coddy/skills/sync`；两者都不会安装任何内容。两者都会刷新已安装列表并闪现对勾，列表上方的一行会说明检查发现了多少更新，或说明每个已安装技能都是最新的（检查一行之后，则是从该行安装的每个技能），检查无法读取的列表会在错误行中指明，而不会被称为已是最新，失败的刷新会说明这一点并保持按钮可用。被暂扣的行的检查处于禁用状态，其标题会先请求批准，或者在 `deny` 下指明策略。
- 一行不会安装任何东西：源在添加时整体安装（或使用 `coddy plugin install <source>`），目录中的插件则从安装搜索中安装。
- 删除是 `DELETE /coddy/skills/sources?source=<key>&origin=<row origin>`：它只编辑该行对应的文件，因此移除你的条目永远不会影响项目已检入的副本。
- 添加字段用 `POST /coddy/skills/sources` 和 `sync: true` 声明一个源，因此它在添加时会整体安装，安装到你的文件中，或者在使用 **此项目** 时安装到项目的文件中，并在写入时即为该工作区批准它。**此项目** 在聊天拥有会话后可用：在第一条消息之前，服务器会采用它自己的默认文件夹，而不是屏幕上显示的文件夹，因此选择器会停留在你的文件上。
- 在浏览器中保留为草稿的聊天在服务器上没有会话：此时列表及其操作不会发送 `X-Coddy-Session-ID`，并针对服务器的默认工作区作出响应（MCP 标签页也如此）。保存设置会重新读取列表，因此各行会跟随更改后的 `skills.project_trust`；子智能体目录对 `subagents.project_trust` 也如此。
- 检查之后，每个已安装且发布了更新版本的技能都会带有一个下载按钮（`POST /coddy/skills/{name}/update`），列表上还有一个覆盖所有这些技能的 **更新 N 个技能** 按钮（`POST /coddy/skills/update`），仅在有待更新内容时显示。两者都不会安装未安装的技能。
- **已安装技能** 的安装搜索会在字段获得焦点时读取生效的市场，并在字段下列出与输入内容匹配且未安装的插件；列表打开时会滚动到可见区域，因为该字段位于设置面板中较低的位置。无法读取的市场或源会在列表中指明，而不会被静默略过。
- 安装搜索（`GET /coddy/skills/available`）、更新检查（`GET /coddy/skills/updates`）以及两种更新都会发送到同一会话，因此在那里获得批准的项目市场会被提供，而在那里被暂扣的市场既不会被检查，也不会从中更新。
- 在手机上，行的控件会换行到键下方，添加字段会占满整个宽度。

![设置，技能：市场列表](../../assets/skills/skills-marketplaces-dark-1280.png)

自动化检查：

- **external/ui/src/ui/settings/MarketplacesEditor.test.tsx**（包含类型、来源和文件的行，内置行的禁用盾牌和删除，带说明的暂扣项目行，禁用的检查和盾牌，带指纹的批准及其撤回，被拒绝的批准，被拒绝的行，带作用域和安装的添加，按键移除，对一行和对所有行（标题的刷新图标）的检查及其发现，俄语文案）；**external/ui/src/ui/settings/SkillsSection.workspace.test.tsx**（一次发现较新版本的检查会提供 **更新 N 个技能**，它会为查看的会话更新这些技能）
- **external/httpserver/server_test.go**（`TestCoddySkillsSourcesSyncDelete`、`TestCoddySkillsNewRoutesEdgeCases`：路由、作用域、对重写条目返回 409、对同步暂扣条目返回 400）
- **features/skills_marketplace.feature**、**features/skills_marketplace_trust.feature**（项目市场等待批准，然后安装到 Coddy 主目录；检查报告较新版本且不安装任何内容；更新会安装它）

## 集群屏幕

![带有中继、节点和环的集群屏幕](../../assets/swarm/map-dark-1280.png)

*集群屏幕：本机、所连接的中继，以及一跳和两跳外的节点*

指南：`docs/operate/swarm.md`。视觉契约：`DESIGN.md`（**集群屏幕**）。

- **集群** 侧栏入口仅在 `GET /swarm/info` 有响应的地方出现，因此普通智能体永远不会显示它。它位于侧栏底部，在 **文档** 和 **设置** 上方。
- 在中继上，集群地图**就是**主屏幕：没有输入框，没有 `ChatScreen`，没有历史记录入口，也没有调度器入口，因为中继本身不持有任何会话。环境菜单在那里也保留在侧栏底部，在地图的空状态和错误状态下也是如此，因此不可用的中继不会移除更改环境的方式。
- **点击地图上的节点会切换到该节点**，并让地图继续覆盖在其上，该节点此时带有环；接下来要做什么是下一次点击的事。地图下方没有节点列表，也没有筛选标签：从某个节点出发，每个普通屏幕（聊天、历史记录、调度器、设置、工作区）都针对它工作，而侧栏中的 **集群** 会返回中继。应用当前所在的节点，以及中继自身地图上的中继，都不是按钮。
- 在两个远程之间切换不会重新加载页面：应用会就地在新远程上重新开始，地图立即根据其最后画面绘制，侧栏保持不变。
- 地图会为应用当前所在的节点加上环（在中继本身上，就是中继的卡片），并从所连接的中继向其绘制一条连通的强调路径；该路径之外的一切都会退到背景中。该路径上的每个中继都有轮廓，当前中继具有更强的当前节点轮廓。悬停另一个节点会预览点击会带你到哪里，其工具提示会说明绘制内容所示的信息：节点是什么、它是否向外拨号、它正在做什么。地图仅在出现故障时（*离线*、*无路由*）在节点下方写文字，并在卡片上统计中继的链接；连线不带箭头，仅当中继以另一个名称认识该节点时，连线才会被命名。
- 每个节点都会表明它正在做什么：空闲时显示会话数量，有轮次进行中时显示运行数量，当那里有内容等待权限提示时显示 *需要回答*。运行中的节点会脉动，等待中的节点以不同方式脉动，通往运行中节点的跳线带有移动虚线。所有这些都来自 `GET /swarm/sessions`，并且在 `prefers-reduced-motion` 下全部停止。
- 搜索在中继上运行，而不是在浏览器中运行，因此它可以到达此浏览器无法拨号的节点。匹配的会话仅在存在查询时作为行显示在地图下方；一行会在其节点上打开该会话。未响应的节点会作为警告列在地图上方，而不是被丢弃。
- 在手机上（低于 1200 px），屏幕会打开在顶栏下方、变暗背景之上，因此点击可以到达地图、搜索和节点；点击顶栏自身的条目仍会离开它。
- **树** 是默认地图布局。**图** 是一种确定性的有根自上而下图，用于环和交叉链接，以中继为根，或在绘制时以发起连接的本地计算机为根。它柔和地遵循路由深度，而不是使用僵硬的跳线行，并使用平滑连线。选择树或图是仅存在于浏览器中的状态，位于 localStorage 键 `coddy_swarm_layout`；已保存的旧版 `star` 值会迁移到图，缺失、被阻止或无效的值会选择树。服务器配置不会更改。
- 地图以适应状态开始。其树/图选择器和缩放控件的按钮，包括 **缩小**、级别和 **放大**，与图片查看器和 `.canvas` 看板使用的控件相同，在每个视口宽度下高度均为 28px；操作者缩放或平移后，级别会以百分比显示缩放比例，地图处于适应状态时显示适应图标，点击它会适应图形。滚轮缩放跟随指针，即使在适应状态下拖动也能沿两个轴平移，双指捏合可缩放，并且拖动/捏合手势不会打开节点。获得焦点时，**`+`** / **`=`**、**`-`** 和 **`0`** 分别放大、缩小和适应。适应、中继更改或布局更改会重置相机；轮询只会重新适应未被触碰的相机，否则会在变化后的边界内保留操作者的平移和缩放。
- 使用 `-tags "swarm ui"` 构建时，中继会在自己的地址上提供此 SPA；没有 `ui` 标签时，其根路径会说明如何重新构建。

![390 px 下的集群布局和缩放按钮](../../assets/web-ui/swarm-layout-controls-compact-dark-390.png)

*布局和缩放按钮具有相同高度。*

## 文档屏幕

![1280 px 下的文档阅读器](../../assets/built-in-docs/reader-page-dark-1280.png)

*文档阅读器：目录、页面、页面中的章节*

指南：`docs/features/built-in-docs.md`。视觉契约：`DESIGN.md`（**文档屏幕**）。

- 侧栏中的 **文档**（位于设置上方）、应用内任意位置的 **F1**，或地址 **`#/docs/<page>#<section>`** 会在集群屏幕所使用的同一玻璃停靠容器中打开阅读器（**`ui/docs/DocsView.tsx`**），两者共用同一宽度（**`--coddy-dock-width`**）。无论当前激活的是哪个环境，页面都从页面来源服务器读取：在远程、节点或中继上，阅读器仍显示本地二进制文件自身的文档。提供自己页面的中继没有文档，因此那里的侧栏会省略 **文档**。侧栏入口会重新打开阅读器上次停留的页面；仅 **`#/docs`** 会以 **`replaceState`** 停留在目录的第一页，因此“后退”不会返回到空阅读器。再次按 **F1**、**Escape** 或 **×** 控件会关闭它，并返回到打开它的位置（聊天、集群屏幕、调度器）；点击背景也会关闭它。
- 输入框中的 **`/docs [words or page]`** 是控制台命令，在浏览器中运行（**`ui/docs/docsCommand.ts`**、**`Composer`** **`onDocsCommand`**）：草稿会被清空，且不会发送任何内容，即使有轮次正在运行也如此。单独使用时，它会重新打开这本书；带有 **`/`**、**`#`** 或 scheme 的参数，或 **`GET /coddy/docs/page`** 解析到的页面确切标题，会打开该页面并定位到其章节；任何其他词语会打开阅读器，并已键入搜索内容且展开其命中结果（**`searchSeed`**）。斜杠菜单的“命令”组会在服务器命令旁列出 **`/docs`**。
- 数据来自 **`GET /coddy/docs`**（目录）、**`GET /coddy/docs/page`**（含标题和相邻项的一个页面）和 **`GET /coddy/docs/search`**（**`ui/docs/api.ts`**），像其他所有路由一样通过环境垫片，因此远程环境会显示它所通信的二进制文件的文档。
- 每个页面、章节、命中项和相邻项都是真正的 **`href`**：跟随其中一个会添加历史记录条目（“后退”和“前进”在已读页面之间移动），中键点击会打开新标签页，章节标题后的 **`#`** 就是该章节的地址。任何渲染后的 Markdown（页面或智能体的回答）中的 **`coddy:<page>#<section>`** 链接都会变成 **`#/docs/<page>#<section>`**（在 **`scheduler/hashRoute.ts`** 中为 **`docsHrefFromCoddyLink`**，由 **`markdown/Markdown.tsx`** 使用）。粘贴地址中的格式错误转义会按键入内容保留，而不会使路由器崩溃。
- 标题会获得服务器计算出的锚点，按级别和文本配对（在 **`ui/docs/docsReader.ts`** 中为 **`assignHeadingIds`**），然后阅读器会滚动到地址所指的章节。**本页内容** 会随页面滚动跟随正在阅读的章节；低于 1280 px 时它会被省略，低于 1200 px 时目录会折叠成页面之上的 **目录** 按钮。在那里，搜索和页面会横跨整个版面，而 **本页内容** 仍保留：从 900 px 起它像桌面端一样位于页面右侧，低于该宽度时它会折叠成 **目录** 下方的第二个按钮，章节链接会再次将其关闭。在桌面端，搜索结束于文本结束的位置，包括页面的经典滚动条。在目录中，组内的页面位于组标题右侧。
- 页头位于页面的各列上：标题在目录上方，搜索框在正文上方，**询问智能体** 和关闭控件在大纲上方。页头不滚动：页面在其下方的正文中滚动（**`.docs-body`**），因此其滚动条从搜索框下方开始。阅读器的宽度不会超过其三列（1350 px），并保持居中，因此在宽窗口中大纲会与正文对齐。
- 搜索框（**`/`** 会使其获得焦点；阅读器打开时键盘焦点在页面上，因此方向键和翻页键会滚动页面）在键入时进行搜索，在最后一次按键后 120 ms，并在自身下方显示命中结果，同时目录保持不变：页面 › 章节，以及标记了匹配词的摘录。它是一个组合框：向上和向下移动选择（**`aria-activedescendant`**），Enter 打开选中的命中项并收起列表，Escape 清除搜索；在空框中，Escape 关闭阅读器。
- 点击页面中的图像会在所有内容之上打开它（**`ui/components/ImageLightbox.tsx`**，渲染到正文中）：先适应窗口，**`+`** / **`-`** / 按钮会从 100% 缩放到 300%，点击图像会在适应和 200% 之间切换，**`0`** 会再次适应，Escape 或关闭控件会关闭它。缩放后的图像可用鼠标、笔或一根手指**拖动**（光标为张开的手，握住时闭合），图像上的**滚轮**会围绕指针缩放，而不是滚动舞台或其后页面（在适应状态的图像上向下滚轮不起作用），在触摸屏上，两指**捏合**会在适应大小和 300% 之间连续缩放，并保持两指之间的点不动；移动过的按压会平移，且不会同时切换缩放，因此原地按压的点击仍然有效。按钮和按键保持 100 / 150 / 200 / 300 的阶梯，并在捏合留下的位置取上或下一级。这些控件是画布和集群地图共享的缩放控件（**`ui/components/ZoomControls.tsx`**），位于查看器栏中关闭控件旁边，滚轮遵循相同规则（**`ui/components/zoom.ts`**：500 像素滚轮使图像放大一倍）。页面中的视频（文件为 **`.mp4`**、**`.webm`** 或 **`.mov`** 的 Markdown 图像，这正是服务器对 GitHub 附件行的处理结果）会在发布时从 GitHub 获取的 **`<video>`** 中播放。
- **`@coddy:<page>#<section>`** 是已发送消息（**`UserMessage.tsx`**）和回答（**`markdown/remarkDocMentions.ts`**）中指向阅读器的链接，按提及的语法读取（**`ui/docs/docMentions.ts`**）；代码仍然是代码。
- **询问智能体** 会启动一个新聊天（在 **`App.tsx`** 中为 **`askAboutDocs`**），其草稿会提及其页面或正在阅读的章节；如果在页面上选中了文本，它会引用所选内容并提及选择所在的章节（**`askDraftFor`**、**`sectionAnchorAt`**）。该按钮通过在鼠标按下时不获取焦点来保留选择，并在下一页加载时禁用；屏幕上的页面会变暗保留，直到新页面到达。在中继上，由于没有聊天，该按钮不会显示。

![移动端缩放控件](../../assets/web-ui/zoom-controls-dark-390.png)

*缩放控件在移动布局中保持紧凑。*
## Swagger

- Swagger UI 在 `/docs/` 下提供。
- OpenAPI 规范在 `/openapi.yaml` 和 `/openapi.json` 下提供。
- Swagger UI 资源必须内嵌，不使用 CDN。

## 开发工作流

- 在 `external/ui/src/` 下编辑 TypeScript 源代码。
- 使用 `npm --prefix external/ui run dev` 迭代而无需重新构建 Go 二进制。
- 使用 `npm --prefix external/ui run build:go` 构建并同步内嵌资源。
- **`make build TAGS="http ui"`** 在链接内嵌 bundle 之前运行 UI 构建步骤（**make ui-build**）。

### 在没有 Mac 的情况下复现 Safari 报告

Playwright 随附 Safari 所基于的 WebKit 构建，其版本跟随 Safari（**`playwright install webkit`** 为 Safari **26.x** 拉取 WebKit **26.x**），因此 Safari 布局报告可以在 Linux 上复现。**`external/ui/scripts/webkit-scroll-check.mjs`** 在该引擎中驱动一个运行中的 **`coddy serve`**，并断言文件夹浏览器对话框在短视口下的滚动不变量：没有任何布局超出对话框的高度上限，操作按钮位于对话框内，列表在滚轮手势下滚动，过度滚动停留在对话框内。它还会在服务器自身的文档和一个内容拥挤的文档上打开 **设置 → 系统**（`#/settings/status`），将面板滚动到末尾，并检查最后一个块、页脚和复制按钮都在视野内，以及每个摘要项都会让其块获得焦点；**`CODDY_ENGINE=chromium`** 在 Chromium 中运行它。

```bash
cd external/ui && npm i --no-save playwright && npx playwright install webkit
```

```bash
CODDY_URL=http://127.0.0.1:12345 CODDY_FOLDER=/a/folder/with/many/subdirs npm --prefix external/ui run check:webkit
```

**`CODDY_ENGINE=chromium`** 在 Chromium 中运行相同的断言，从而将仅 WebKit 的回归与所有引擎共有的布局缺陷区分开。该脚本 **不属于 `make test`**：它需要下载浏览器和一个运行中的服务器。**`npm ci`** 和 **`make ui-build`** 会清除未保存的 **`playwright`** 安装，因此重新构建后要重新运行安装命令行。

### 对照标签检查折叠箭头

该箭头是一个 SVG，其墨迹居中于其 viewBox（**`DESIGN.md`**，*Chevron*），因为作为文本字形时，其墨迹位于平台字体放置它的位置，而箭头高于其标签的报告不断出现。标记实际落在何处是布局事实，jsdom 没有布局，因此 vitest 测试套件无法回答。**`external/ui/scripts/chevron-align-check.mjs`** 在真实引擎中测量它：对于思考行、工具行和任务面板的 **Finished N** 开关，它比较箭头的墨迹中心与标签文本的墨迹中心（在文本节点上使用 **`Range`**，因此元素的 padding 或其自身行高都无法隐藏错误），并在二者相差超过 **1px** 时以非零状态退出。

它驱动 **`src/chevron-align-check.html`**，这是一个测试台，从真实组件对照真实样式表挂载那三个界面，因此它需要一个 **`vite`** 开发服务器，完全不需要后端。

```bash
cd external/ui && npm i --no-save playwright && npx playwright install chromium
```

```bash
cd external/ui && npx vite --port 5241 &
CODDY_UI_URL=http://127.0.0.1:5241 npm --prefix external/ui run check:chevron
```

**`CODDY_ENGINE=webkit`**（或 **`firefox`**）在另一个引擎中运行相同的测量，**`CODDY_CHEVRON_TOLERANCE_PX`** 会提高允许值。与上面的 WebKit 框架一样，这个 **不属于 `make test`**：它是手动检查，在变更涉及箭头、箭头所在的行或周围的字体时运行。

### 检查 diff 的行号

编辑窗口中 diff 的每个行号单元格都与文件最长数字一样宽（**`DESIGN.md`**，*聊天记录的编辑*），因为按自身数字确定大小的单元格会把被删除或添加的行的代码（其另一个单元格为空白）放在上下文的左边几个像素处。单元格绘制多宽是布局事实，jsdom 没有布局，因此 vitest 测试套件无法回答。**`external/ui/scripts/diff-gutter-check.mjs`** 在真实引擎中测量它：在统一视图和拆分视图的每一列中，它读取每一行代码的起始位置，并在两行相差超过 **0.5px** 或行号不适合其单元格时以非零状态退出。

它驱动 **`src/diff-gutter-check.html`**，这是一个测试台，从真实组件对照真实样式表挂载编辑窗口的一个文件，带有删除和添加的行以及两到四位的数字，因此它需要一个 **`vite`** 开发服务器，完全不需要后端。

```bash
cd external/ui && npm i --no-save playwright && npx playwright install chromium
```

```bash
cd external/ui && npx vite --port 5242 &
CODDY_UI_URL=http://127.0.0.1:5242 npm --prefix external/ui run check:diff-gutter
```

**`CODDY_ENGINE=webkit`**（或 **`firefox`**）在另一个引擎中运行相同的测量，**`CODDY_DIFF_GUTTER_TOLERANCE_PX`** 会提高允许值。它 **不属于 `make test`**：当变更涉及编辑窗口的 diff 行、其行号或其字体时运行它。

### 检查每种缩放下的输入框光标

输入框的 textarea 保留光标并将其字形绘制为透明；其下方的镜像绘制带有 chip 的草稿（**`DESIGN.md`**，*输入框镜像与光标同步*）。当二者以不同方式布局文本时（另一个宽度的文本框在另一个单词处换行，滚动偏移吸附到另一个设备像素而移动行），光标会远离用户阅读的文本。这在浏览器的页面缩放之下以及窗口被拖到另一个密度的显示器后都会出现，在 Chrome 和 Yandex Browser 中都是如此。**`external/ui/scripts/composer-caret-check.mjs`** 在真实引擎中测量它：对于每种情况，它都会对输入区截两次图，第一次隐藏镜像并将 textarea 的字形绘制为红色，然后绘制镜像的字形，并在任何像素不同时失败。这些情况包括：1280 px 下的新聊天和停靠聊天，以及 390 px 下的手机；设备缩放因子 1 到 3；页面缩放 110% 到 175%（用根元素的 CSS **`zoom`** 模拟，它像页面缩放一样将滚动条保持在设备像素中）；一个会滚动和一个不会滚动的草稿；用滚轮将输入区滚回一部分；输入后更改缩放；以及在 Chromium 中，在打开的页面下方更改密度而不调整窗口大小。

它驱动 **`src/composer-caret-check.html`**，这是一个测试台，从真实组件对照真实样式表挂载真实的 **`Composer`**，因此它需要一个 **`vite`** 开发服务器，完全不需要后端。

```bash
cd external/ui && npm i --no-save playwright && npx playwright install chromium
```

```bash
cd external/ui && npx vite --port 5241 &
CODDY_UI_URL=http://127.0.0.1:5241 npm --prefix external/ui run check:caret
```

**`CODDY_ENGINE=webkit`**（或 **`firefox`**）在另一个引擎中运行它，**`CODDY_BROWSER_PATH`** 将其指向已安装的 Chromium（包括 Yandex Browser 的二进制文件），**`CODDY_CARET_TOLERANCE`** 会提高允许不同的像素数。Firefox 会跳过输入后更改缩放：它在 textarea 中保留小数滚动偏移，并且当只有 CSS **`zoom`** 变化时会对 div 的偏移进行舍入，而它自己的页面缩放、密度变化不会这样做。与本页的其他框架一样，它是手动检查，在变更涉及输入框的输入区、其镜像或周围的字体时运行。

### 检查长提示词和引用

输入框的输入区增高多少、展开后的输入框停在哪里、引用按钮位于何处以及对话在哪里打开，都是 jsdom 看不到的布局事实（issue #342；**`DESIGN.md`**，*输入框高度与展开*、*从对话记录引用*、*对话记录滚动到底部按钮*）。**`external/ui/scripts/long-prompts-check.mjs`** 针对一个运行中的 **`coddy serve`**，在 1280 px 下用鼠标、在 390x844、320x568 和 844x390 下用手指测量它们：输入区随草稿增高，并在其上限处停止（八行、可见视口的 40%、绝不高到让停靠块骑到聊天标题栏上方）；展开控件位于输入框右边缘上方；在手机和平板上，顶部栏位于窗口边缘下方一个顶部节奏步长处，标题位于栏下方一个步长处；展开时，标题、控件和卡片彼此相距一个步长，输入区在按压过程中保持焦点（手机键盘保持打开），技能选择器在输入区底部打开且不使任何内容变暗；折叠时，输入区恢复为其文本；对于会缩小页面的键盘和会覆盖页面的键盘（视觉视口的替代物），展开后的输入框结束于其上方；在最后一个回答中的选区会在标题栏与停靠块之间的区域内提供引用，用鼠标时在其上方，用手指时在其下方，按压会将引用追加到草稿；在读者滚动到第一个对话的顶部后，另一个对话会在其最新消息处打开；展开控件在魔杖的垂直线上位于输入框上方 10px，跳到最新消息在输入框上方 10px；页面从不横向滚动。

它需要一个带有两个会话的服务器，每个会话都超过两屏，并且某个回答中包含一个段落：即 **`GET /coddy/sessions`** 的前两个，或 `CODDY_SESSIONS` 中的 id。如果测试台无法展示某项检查所测量的内容，它会以退出状态 2 停止并说明原因，而不是在空无一物的情况下通过。

```bash
cd external/ui && npm i --no-save playwright && npx playwright install chromium webkit
```

```bash
CODDY_URL=http://127.0.0.1:18081 npm --prefix external/ui run check:long-prompts
```

**`CODDY_ENGINE=webkit`** 在 WebKit 中运行它，在那里点击引用过去不会做任何事（被取消的 **`pointerdown`** 在那里会让触摸失去其点击）。**`CODDY_BROWSER_PATH`** 将 Chromium 指向已安装的浏览器。当变更涉及输入框的高度、展开控件、引用按钮或对话记录的跳转时运行它。

### 检查长对话记录

长对话是否能立即打开并滚动而不跳动，是 jsdom 无法回答的布局和时间问题。**`external/ui/scripts/transcript-window-check.mjs`** 针对真实二进制回答它：它将一个包含 3306 条消息的会话（约 9 MB，即 issue #338 的大小）和一个短会话写入临时 home 目录，提供自己的脚本化 OpenAI 兼容模型，启动一个带有 Web 界面的 **`coddy serve`** 节点和一个挂载它的 **`coddy serve --swarm`** 中继，并在 CPU 节流下（默认 x4）驱动 Chromium。它检查并以表格形式打印：

- **打开** 在 390 和 1280 px 下：第一次读取的是最新一页，最新消息在预算内出现在屏幕上并位于底部，没有任务阻塞页面超过该预算，DOM 持有有界的切片，堆保持较小；
- **向上滚动** 浏览历史：更旧的页面到达，读者下方的行移动不超过舍入误差（在关闭引擎自身的滚动锚定时也是如此，这正是 WebKit 的情况），帧时间、窗口保持有界，没有行 id 重复；滚动到底部按钮随后返回最新消息，并且在下次向上滚动时重新读取更旧的页面，这些页面此前已被释放；
- **堆叠外壳** 在 390 px 下，此时文档滚动；
- **一次编辑** 对较旧页面上提示词的编辑会指出服务器已知的索引（回退被拦截）；
- **一次失败的读取** 对上方页面的失败读取会等待 **重试**，而不是每一帧都再次请求，**重试** 会读取它；
- **模型提出的一个问题** 保持挂载，答案已被选中，同时读者向上翻到历史很远再返回；
- **一个短会话** 会被完整读取，每一行都可到达；
- **两个浏览器** 在同一会话上：一个向上读到很远，另一个运行一个轮次然后回退它，读者无需完整读取即可跟随两者；
- **一个集群中继**：通过中继的挂载点进行相同的读取。

```bash
make build TAGS="http ui swarm"
npm --prefix external/ui i --no-save playwright && npx --prefix external/ui playwright install chromium
CODDY_BIN=build/coddy npm --prefix external/ui run check:transcript
```

**`CODDY_CPU_THROTTLE`** 设置减速（**`1`** 将其关闭），**`CODDY_BUDGET_SCALE`** 为慢速机器倍增时间预算，**`CODDY_SCENARIOS`** 运行部分场景（**`open`**、**`scroll`**、**`phone`**、**`edit`**、**`retry`**、**`prompt`**、**`short`**、**`two-browsers`**、**`swarm`**），**`CODDY_ENGINE=webkit`** 在 WebKit 中运行它们（不含节流、长任务和堆读取，这些是 Chromium 的），**`CODDY_BROWSER_PATH`** 指向已安装的 Chromium，**`CODDY_E2E_KEEP=1`** 让测试台继续运行。CI 在测试矩阵的 **`http,scheduler,ui,memory,cli,swarm`** 任务中以全速运行它。当变更涉及对话记录窗口、消息或工具调用路由，或对话记录行渲染的内容时运行它。

### 通过集群中继检查消息队列

排队消息是否以正确模式到达模型，以及每个浏览器是否都能看到它，是真实二进制在确实有轮次运行时要回答的问题。**`external/ui/scripts/queue-modes-check.mjs`** 回答它：它提供一个脚本化 OpenAI 兼容模型，让其开场回答保持打开，直到脚本释放它们，启动一个没有 **`agent.queue_mode`** 的 **`coddy serve`** 节点和一个挂载它的 **`coddy serve --swarm`** 中继，并通过中继的挂载点驱动 Chromium，同时第二个浏览器直接在节点上观看同一会话。它检查：

- **首次使用问题** 会针对轮次期间写入的第一条消息出现，其答案通过中继以 **`agent.queue_mode`** 保存在节点的 **`config.yaml`** 中；
- **Tab** 会用一张图片排队另一种模式，队列会命名该图片且从不携带其字节，另一个浏览器能看到这两行；
- **模式切换**（PATCH）和**取回消息**（其文本和图片返回草稿）通过中继生效；
- **回答之后**，模型读取引导消息，然后将延迟消息作为带有其图片的独立提示词读取，实时对话记录会在其回答上方显示延迟消息；
- **图片通过中继加载**：轮次之后和重新加载之后的缩略图，灯箱中的原图，且不会向中继自身的源请求资源，也没有 URL 携带令牌；
- **停止** 会让轮次之后的消息保持等待而不启动它，它会在下一个回答之后运行；
- **设置 → ReAct 循环** 显示已保存的模式。

```bash
make build TAGS="http ui swarm"
npm --prefix external/ui i --no-save playwright && npx --prefix external/ui playwright install chromium
CODDY_BIN=build/coddy npm --prefix external/ui run check:queue
```

**`CODDY_SHOTS_DIR`** 还会在直接访问的第二个节点上截取[消息队列](../features/message-queue.md)的截图，**`CODDY_BROWSER_PATH`** 指向已安装的 Chromium，**`CODDY_PORT_BASE`** 移动其四个端口，**`CODDY_E2E_KEEP=1`** 让测试台继续运行。CI 在测试矩阵的 **`http,scheduler,ui,memory,cli,swarm`** 任务中运行它，位于对话记录检查之后。

### 通过集群中继检查聊天的视图

视图按钮、停靠区和文件窗口由 **`external/ui/scripts/files-check.mjs`** 针对真实二进制进行检查：它提供一个脚本化模型，其第一个回答通过 `write` 工具写入一个文件，启动一个要求 bearer 令牌的 **`coddy serve`** 节点，以及一个用自己的客户端令牌挂载它的 **`coddy serve --swarm`** 中继，并仅通过中继的挂载点驱动浏览器。它检查：

- **视图按钮** 在标题旁按该顺序显示文件与任务，且没有编辑按钮，文件是一个 18px 图标，任务是一个点，在桌面上各自单独位于 36px 的方块中，悬停时提示框显示在窗口内（文件按钮带有其快捷键），当前显示的视图对应按钮为按下状态，更亮且不使用强调色，再次按下会将其收起；
- **输入框上方的板条** 会标明仓库和分支，与输入框卡片的顶部相接且左右边缘相同，在其右边缘携带 git 的计数且没有轮廓，悬停时会使计数的背景变亮；干净文件夹中的聊天会有板条但没有计数；
- **编辑窗口** 从计数处打开，显示该轮次写入的文件及其地址，Escape 会将其收起；**停靠区** 显示后台任务，任何地方都没有标签条；每个位于文字旁的图标在所有宽度下都对齐其小写字母的中部；
- **丢弃** 编辑窗口中的文件会先询问，通过中继进行，当没有剩余内容时窗口会显示文件夹已干净，计数会从板条上消失；
- **文件窗口** 在聊天上方打开且筛选器获得焦点，先列出文件夹，按名称找到向下三层文件夹中的文件，将文件作为标签打开且文件上方没有任何遮挡，将 README 显示为其源代码且不加载图片，通过中继加载工作区的一张图片，展开到侧栏上方，按 Escape 关闭，在 `Ctrl+Shift+F` 上打开和关闭并保留其标签，并在对应行打开对话中的提及；
- **窗口绘制的类型**：跨三行的注释会在每一行上着色；SVG 作为通过中继读取的图片打开，其脚本从不运行，其指定的信标也从不被获取，关闭 **预览** 会显示其源代码；HTML 文件作为其源代码打开，**预览** 会在一个带有 `sandbox=""` 的框架中绘制它并填满文件的位置，其自身样式生效，其脚本从不运行，其指定的任何内容都不会被获取；PDF 从 `blob:` 地址在浏览器自己的查看器中打开，位于没有沙箱的框架中（此次运行为此使用完整的 Chromium，因为无头 shell 没有 PDF 查看器，在没有查看器时改为检查下载通知）；
- **声音通过中继播放**，仅从其签名地址播放，该地址在没有头部的情况下响应范围请求，伪造的地址会被节点拒绝，**下载** 会按其名称保存文件；
- **在手机上**，窗口适应屏幕，显示通过其地址打开的文件且没有树，文件和任务是标题旁的 40 px 方块，一个图标和一个点，板条保持与输入框卡片相接且计数位于其右边缘；
- **从另一个源**（节点自己的 Web 界面，以中继作为其环境）中，窗口通过中继读取，带有 `If-None-Match` 的 `HEAD` 通过中继的预检，`ETag` 可读，且打开的文件会被重新验证；
- **系统标签页**，分别针对通过中继的节点、直接打开的节点以及切换器中选择的远程环境，各自对照该服务器自身的 `GET /coddy/status`，以及一个已停止的节点：离线并显示其最后可见时间，先前打开的标签页变灰且陈旧。每个测试台都使用自己的 `HOME` 和 `TMPDIR` 运行，不继承 `CODDY_*` 变量，并显式设置 `--cwd`；
- **在网格的每一层**（600 到 1920 px），英语和俄语下，窗口都适应视口，树位于文件旁，视图按钮在一行中位于标题旁，两个方块的高度与顶部栏按钮相同。

```bash
make build TAGS="http ui swarm"
npm --prefix external/ui i --no-save playwright && npx --prefix external/ui playwright install chromium
CODDY_BIN=build/coddy npm --prefix external/ui run check:files
```

**`CODDY_ENGINE=webkit`** 在 WebKit 中运行它，**`CODDY_SHOTS_DIR`** 还会在直接访问的第三个节点上截取本页面的截图，**`CODDY_BROWSER_PATH`** 指向已安装的 Chromium，**`CODDY_PORT_BASE`** 移动其四个端口，**`CODDY_E2E_KEEP=1`** 让测试台继续运行。CI 在测试矩阵的 **`http,scheduler,ui,memory,cli,swarm`** 任务中运行它，位于消息队列检查之后。

### 检查 Telegram Mini App

`npm run check:telegram`（`external/ui/scripts/telegram-mini-app-check.mjs`）在没有 Telegram 的情况下将 Web 界面作为 Telegram Mini App 打开。它启动 [tgfake](../contributing/tgfake.md) 及其脚本化模型，以及一个真实的 `coddy serve`，其机器人会与测试台通信，并驱动测试台的聊天页面，该页面在其手机框架中打开机器人的 `/app` 按钮。机器人的 `admins` 会命名测试台的聊天，因为 Web 界面只对机器人的管理员开放。它会检查菜单按钮和 `/app`；Mini App 向客户端请求的内容（展开、关闭滑动、标题颜色、最后就绪）；不含启动数据的地址；半开窗口，在对话记录停在其最新消息时进入（输入框和该消息在视野内，没有任何内容宽于页面，聊天标题仍然粘性）；可见部分内的模式面板和启动屏幕的上下文面板；返回；Telegram 的浅色主题；以及跨站框架中的登录提示（测试台页面位于 `localhost`，Web 界面位于 `127.0.0.1`）。`settings` 场景在其自己的上下文中检查[设置搜索](#设置搜索)，一个为它创建的 390x844 触屏手机，并在依赖它之前断言仅触控媒体查询。它打开设置，搜索 `max_turns` 并选择结果：该设置被圈出，其框获得焦点，因此不会弹出键盘。从该章节返回后，磁贴仍保留，查询仍在字段中，结果隐藏；点击字段会再次显示结果，返回会清除查询并离开磁贴；再按一次返回会关闭设置。重新打开设置后，外观磁贴的标签会找到 **主题** 块，而没有任何东西能找到 **通知**，因为 Mini App 不绘制通知。CI 不运行此场景；当搜索或 Mini App 的返回发生变化时手动运行它。`system` 场景在 390 px 下从其磁贴打开 **设置 → 系统**，并对照测试台自身的 `GET /coddy/status` 检查其四个块，没有横向滚动。

```bash
make build TAGS="http ui gateway"
cd external/ui
npm i --no-save playwright && npx playwright install chromium   # once
CODDY_BIN=../../build/coddy npm run check:telegram
```

`CODDY_BROWSER_PATH` 指向已安装的 Chromium 而不是 Playwright 的下载版本，`CODDY_SCENARIOS` 运行 `bot`、`open`、`half`、`sheets`、`theme`、`signin` 和 `settings` 中的部分场景，`CODDY_SCREENSHOTS` 写入本页面和网关页面展示的截图，`CODDY_E2E_KEEP=1` 让测试台继续运行以供查看。

### 检查图表和公式

Mermaid 是否绘制、浏览器是否解码图片、KaTeX 的字体是否到达以及图片是否适合对话记录，都是真实引擎的事实。**`external/ui/scripts/diagram-check.mjs`** 打开 **`src/diagram-check.html`**，这是一个测试台，它在 **360** 和 **1280px** 下、以全部八种主题，从真实组件对照真实样式表挂载一个回答，其中包含六个 Mermaid 图（流程图、序列图、思维导图、甘特图、饼图和类图）、一个 SVG、一个损坏的图、行内和显示公式以及一个价格，并在以下情况失败：图片未绘制或伸出列外、损坏的图未指明其错误、公式未排版、KaTeX 字体未加载、价格变成了公式、页面横向滚动、Mermaid 和 KaTeX 未按需获取，或任何内容被记录为错误。

```bash
cd external/ui && npx vite --port 5247 &
CODDY_UI_URL=http://127.0.0.1:5247 npm --prefix external/ui run check:diagrams
```

该测试台还会截取 **`?theme=<id>`**、**`?lang=ru`** 和 **`?case=<name>`**（测试台的一个回答）的截图。**`CODDY_BROWSER_PATH`** 和 **`CODDY_ENGINE=webkit`** 的用法与上述检查相同。它 **不属于 `make test`**：当变更涉及 Markdown 渲染器、其样式或 chunk 的构建时运行它。

### 通过集群中继检查画布

画布由 **`external/ui/scripts/canvas-check.mjs`** 针对真实二进制进行检查：一个脚本化模型将工作区的一个商店页面放到画布上，编辑其脚本并读取浏览器的报告，一个要求 bearer 令牌的节点位于带有自己客户端令牌的中继之后，浏览器仅通过中继的挂载点到达该节点。它会检查：文件窗口会在页面所进入的 `.canvas` 文件上自行打开，提示栏位于板下方；页面的模块脚本在不透明源中运行，父级和 API 不可及；在评论模式中选择会变成聊天的提示词；页面的控制台错误会到达 `canvas_inspect`；一次编辑只会重新加载文件发生移动的那个框架；滚轮会缩放板而不在其上再次绘制任何内容，并且关闭窗口模糊，在大约一帧屏幕时间内完成；导航离开的页面会说明这一点，其旧地址停止工作；作为独立标签页打开的框架地址仍处于沙箱中，并被打开的节点拒绝；窗口和板控件在英语和俄语下适应网格的每一层；以及在窗口中打开的 Obsidian `.canvas` 文件会成为聊天的画布：其页面运行，其卡片、组和边被绘制，拖动的卡片和调整大小的框架会在文件中移动，并保留 Coddy 不认识的键，在别处编辑该文件会显示在打开的板上，**预览** 会显示文件的 JSON。

```bash
make build TAGS="http ui swarm"
cd external/ui && npm i --no-save playwright
CODDY_BROWSER_PATH=/usr/bin/chromium npm run check:canvas
```

**`CODDY_SHOTS_DIR`** 写入本页面和画布指南的截图，**`CODDY_ENGINE=webkit`** 在 WebKit 中运行它。它 **不属于 `make test`**：当变更涉及 `external/ui/src/ui/canvas/`、`internal/canvas`、画布路由或中继的挂载点时运行它。

### 在网格的每个宽度下检查对话记录

比对话记录更宽的行会在堆叠外壳上使整个页面横向滚动，而行是否适合是 jsdom 无法回答的布局事实。**`external/ui/scripts/phone-overflow-check.mjs`** 在字体加载完毕且行动画结束后，在网格的验证宽度（**360**、**375**、**393**、**430**、**599**、**600**、**834**、**1199**、**1280px**）下、以两种随附语言，在真实引擎中测量它：页面不得横向滚动，除读者可横向滚动的盒子（表格、代码块；仅裁剪的盒子不能作为借口）内部外，任何内容都不得伸出 **`.messages-inner`**；每个工具行都将其持续时间保持在行内，并将其目标、标记和持续时间保持在一行；任何目标都不得窄于其前 24px；文件某一部分的 **`read`** 的行范围无论其路径多长都完整显示；带有短标签的行在每种宽度下都保持一行。只要其中任何一项不满足，它就会以非零状态退出。

脚本以一个与宽度无关的检查结束。在模拟的 iPhone 上，它从页面中移除 **`navigator.clipboard`**，因为从另一台机器通过纯 HTTP 提供的页面缺少它，然后点击代码块、行内代码和用户消息的复制控件。页面一开始处于滚动状态，控件一开始获得焦点。每次点击控件中心都必须命中控件本身。除非每个控件都将其完整文本放到粘贴到字段时能找到的位置，从该文本在 16px 或更大的只读文本区域中的选区复制它，并且不留下文本区域，同时保持焦点、滚动位置和缩放与点击时发现的一致，否则检查失败。除非代码块和行内代码显示 **已复制**，否则也会失败。Playwright 的 WebKit 运行在 Mac 上，而不是 iOS 上，因此该检查无法看到 iOS 如何选择文本或缩放进入字段。

它驱动 **`src/phone-overflow-check.html`**，这是一个测试台，从真实组件对照真实样式表挂载过去会溢出的行，包括以 MCP 工具命名的工具行（一个用带有不中断令牌的 JSON 文档回答，失败的带有长持续时间）、一条长命令、一个输出为一条不中断行的内置工具、一个带有表格、代码块、长链接和长标识符的回答、一条由一个不中断单词组成的用户消息，位于带有聊天边距和其 920px 桌面条纹的列中，每个工具行都打开，因此它需要一个 **`vite`** 开发服务器且不需要后端。

```bash
cd external/ui && npx vite --port 5241 &
CODDY_UI_URL=http://127.0.0.1:5241 npm --prefix external/ui run check:overflow
```

Playwright 安装与上方折叠箭头检查相同。**`CODDY_BROWSER_PATH`** 将脚本指向已安装的浏览器（**`/usr/bin/chromium`**），而不是 Playwright 下载的浏览器，**`CODDY_ENGINE=webkit`** 在 WebKit 中运行它。它 **不属于 `make test`**：当变更涉及对话记录的行、Markdown 样式或网格时运行它。

### 检查共享文件卡片

共享文件的卡片会裁剪溢出它的内容（其缩略图到其角落），因此其操作菜单会渲染到文档中，并从 **⋮** 处放置（**`DESIGN.md`**，*共享文件卡片*）；当菜单位于卡片内部时，点击 **⋮** 不会绘制任何内容。菜单是否可见且可到达，以及标记落在何处，都是 jsdom 无法回答的布局事实。**`external/ui/scripts/artifact-card-check.mjs`** 在真实引擎中测量它们，在宽和窄的桌面窗口以及模拟手机（触控、无悬停）上，手机为竖持的 **390**、**360** 和 **320px**，以及侧放的 **844** 和 **568px**，以两种随附语言：每张卡片都是其设计尺寸，**⋮** 与扩展名徽标齐平并在卡片上与其镜像，其点在其内部居中，它在指针下显示，并且在触控屏上始终显示；单击或点击会在窗口内打开一个菜单，其每一项都在其中心响应指针，且没有任何一项被截断；菜单获得焦点，第二次按下、在其他地方按下以及 **Escape**（会把焦点交回 **⋮**）会将其关闭；右键单击会打开它；在窗口底部附近它会向上打开；它会随着滚动跟随其卡片，并在卡片离开窗口或某个层覆盖 **⋮** 时关闭（正如聊天的粘性标题和停靠的输入框所做的那样）；比菜单矮的窗口会将其保持在内部并滚动其项；以卡片结尾的回答下方的复制按钮只在其下方保留自身边距；页面从不横向滚动。只要其中任何一项不满足，它就会以非零状态退出。

它驱动 **`src/artifact-card-check.html`**，这是一个测试台，从真实组件对照真实样式表挂载一行卡片，这些卡片内联在回答中、一个位于 **`share_file`** 行下方，另一个在长页面底部结束一个回答，因此它需要一个 **`vite`** 开发服务器且不需要后端。

```bash
cd external/ui && npx vite --port 5243 &
CODDY_UI_URL=http://127.0.0.1:5243 npm --prefix external/ui run check:artifacts
```

Playwright 安装与上方折叠箭头检查相同。**`CODDY_BROWSER_PATH`** 和 **`CODDY_ENGINE=webkit`**（或 **`firefox`**）的用法与上述检查相同，**`CODDY_ARTIFACT_TOLERANCE_PX`** 会提高 **1px** 的允许值。它 **不属于 `make test`**：当变更涉及共享文件卡片、其菜单或其样式时运行它。

### 检查简化模式

**设置 → 外观** 的模式是计算样式的问题，而 jsdom 没有计算样式。**`npm run check:simple`** 打开开关、重新加载并更改主题，然后在 1280、800 和 390 px 下以每种主题读取启动屏幕、对话记录、运行中的轮次、历史抽屉、设置和文档的每个元素。它会在出现柔和阴影、模糊、装饰性渐变、遮罩、过渡、运行中的动画或对话记录可透视的面板时失败。选定的壁纸会保留其原生图像和调暗效果；它不会恢复玻璃、阴影或界面动画。它会启动自己的 **`coddy serve`** 和一个脚本化模型，因此不需要提供商。

```bash
make build TAGS="http ui"
CODDY_BIN="$PWD/build/coddy" npm --prefix external/ui run check:simple
```

Playwright 安装与上方折叠箭头检查相同。**`CODDY_BROWSER_PATH`** 的用法与上述检查相同，**`CODDY_SHOTS_DIR`** 会在那里写入运行的截图。

**`npm run bench:simple`** 测量该模式节省了什么：它在关闭和打开该模式的情况下运行相同场景，并报告帧率、长帧和浏览器的 CPU 时间，分别在所有核心和一个核心上。
## UI 测试场景

这些场景旨在通过 Playwright 针对 Vite 开发服务器进行自动化。

- 桌面导航没有宽度切换
  - 假设视口宽度为 1200-1919px
  - 当应用加载时
  - 那么 `data-testid="nav-history"` 可见
  - 并且不存在 `.rail-toggle-width`

- Escape 会关闭导航栏打开的画面
  - 假设聊天已在 `#/s/<sessionId>` 打开
  - 当用户从导航栏打开 **文档**、**设置**、**历史** 或 **调度器** 并按 Escape 时
  - 那么该画面关闭，地址再次为 `#/s/<sessionId>`
  - 当 **设置** 的某一行处于打开状态（`#/settings/models?id=<model>`）且用户按 Escape 时
  - 那么该行的表单返回列表，下一次按 Escape 会关闭设置
  - 当文档搜索中有文本且用户在其中按 Escape 时
  - 那么搜索被清空且阅读器保持打开；下一次按 Escape 会将其关闭

- 会话仅以抽屉形式出现
  - 假设任意桌面视口
  - 当应用加载时
  - 那么 `data-testid="sessions"` 不可见
  - 当用户点击 `data-testid="nav-history"`
  - 那么 `data-testid="sessions"` 变为可见
  - 当用户点击 `data-testid="sessions-close"`
  - 那么会话抽屉隐藏

- 移动端使用顶栏和单行品牌
  - 假设视口宽度至多为 1199px
  - 当应用加载时
  - 那么导航宽度切换不存在
  - 并且导航栏高度为 78px
  - 并且仍可从顶栏中的 **历史** 打开会话

- 手机适配起始画面和聊天
  - 假设视口宽度为 360、390 或 430px
  - 当起始画面加载时，以及再次在聊天中时
  - 那么 `document.documentElement.scrollWidth` 等于 `clientWidth`
  - 并且 `#btn-send` 的矩形不与 `.composer-tab` 的任何可见部分相交
  - 并且 `.rail-brand` 的矩形不与任何顶栏图标相交
  - 并且每个顶栏图标都位于胶囊内；低于约 270px 时，顶栏显示历史、设置和更多，而更多列出文档和调度器
  - 并且 `textarea#composer` 计算为 16px
  - 当视口为 768 或 1280px 时
  - 那么输入框各行换行，顶栏看起来与手机区块之前一样

- Enter 跟随输入设备
  - 假设桌面浏览器窗口宽 390px
  - 当用户输入草稿并按 Enter 时
  - 那么草稿被发送
  - 当用户在草稿中间按 Ctrl+Enter 时
  - 那么插入符处出现换行，且不发送任何内容
  - 假设是仅触控设备（Playwright 上下文带有 `isMobile` 和 `hasTouch`）
  - 当用户按 Return 时
  - 那么插入换行，并由发送按钮发送

- 工具调用在重启后保留
  - 假设一个会话已执行工具调用
  - 当用户重新加载页面时
  - 那么工具调用卡片在对话记录中可见
  - 并且展开工具卡片会显示其预览，输出拼接在其下方，或显示工具的结构化卡片，且不带批准按钮
  - 并且如果服务器将预览标记为已截断，则 **更多…** 然后 **收起** 的行为与上表一致；如果未截断，则没有溢出切换行，且结果面板上没有 **`tool-result-viewport--tall`**

- 工具结果截断（Playwright MCP）
  - 假设一个已持久化的会话，其磁盘上的工具输出超过预览行数上限
  - 当用户打开工具卡片并点击 **更多…** 时
  - 那么按钮变为 **收起**，完整行可在同一最大高度可滚动面板内查看，且 **`.tool-result-viewport--scroll`** 的 **`scrollHeight`** 大于 **`clientHeight`**
  - 当用户点击 **收起** 时
  - 那么预览显示以 **`...`** 结尾的截断文本，**`overflow-y`** 在 **`.tool-result-viewport--clip`** 上隐藏，并且 **更多…** 再次出现

- Token 使用量在重启后保留
  - 假设一个会话具有非零 token 使用量
  - 当用户重新加载页面时
  - 那么上下文环的工具提示会显示持久化的输入、输出和总计

- Tasks 面板中的 memory 子智能体（Playwright MCP）
  - 假设 **`memory.enable: true`** 在 **`coddy serve`** 进程上，并且全局或工作区记忆下至少有一个 Markdown 文件，以便召回可以运行
  - 当模型通过 **`spawn_agent`** 使用 **`agent: "memory"`** 委派给 memory 子智能体时
  - 那么 Tasks 面板显示一个标记为 **memory** 的普通智能体任务，Tasks 按钮会将其计入数量，且其卡片会打开子对话记录
  - 当用户打开 Tasks 面板时
  - 那么一个标记为 **`memory: <first line of the message>`** 的任务带有 **memory** 标签（**`bgtask-tag-<id>`**），其卡片提供 **显示对话记录** 以打开只读子会话而无需打开该任务，其打开的卡片显示子日志以 **`=== subagent report ===`** 和交付行结尾

对于针对实时网关的 Playwright MCP，先启动 **`make build TAGS="http ui memory"`**，然后使用一次性的 **`--home`** 启动 **`./build/coddy serve`**，以便配置可以启用 memory；打开 **`http://127.0.0.1:<port>/`**，导航到一个会话，发送提示词，打开 Tasks 面板，并断言该次运行所在行上的 **memory** 徽标。

### 提供商客户端身份

在 **设置 → LLM 提供商** 中打开一个 OpenAI 兼容或 Anthropic 兼容的提供商，
并展开 **高级设置**。**User-Agent** 提供协议 SDK 身份
（`OpenAI/Go` 或 `Anthropic/Go`，默认值）和 **Coddy/version**。后者会为
阻塞和流式传输请求以及模型发现发送 `Coddy/<running version>`。
模型列表使用尚未保存的选择；保存提供商以将其应用到新的模型调用。
其他提供商类型会隐藏该选择器。

![OpenAI 兼容提供商的 User-Agent 选择器](../../assets/web-ui/provider-user-agent-openai-dark-1280.png)

*已选择 Coddy 的 OpenAI 兼容提供商。*

![Anthropic 兼容提供商的 User-Agent 选择器](../../assets/web-ui/provider-user-agent-anthropic-dark-1280.png)

*Anthropic 兼容提供商默认提供其自己的 SDK 身份。*

<!-- docsgen:source sha256=e8810f24145369c5 -->

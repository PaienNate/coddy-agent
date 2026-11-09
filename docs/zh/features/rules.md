# 规则与指令

Coddy 从两个位置读取常驻指令，并将其注入系统提示词，与技能分开（**`{{.Skills}}`**）。

- **工作区**：会话工作目录及其 `AGENTS.md` 的规则文件夹，以及智能体到达后其子文件夹的规则文件夹；这些内容随检出一起传递，并适用于任何打开它的人；
- 你的**智能体主目录**（`~/.coddy`）：`AGENTS.md`、`DESIGN.md` 和 `rules/`，紧邻 `config.yaml`；它们属于运行 Coddy 的人，并在每个工作区中适用。

两者都无需配置。`AGENTS.md` 与 `DESIGN.md` 文档分为三层，始终会被读取，且没有设置可以关闭它们：你的、工作区的，以及智能体所操作文件夹的（[AGENTS.md 与 DESIGN.md](#agentsmd-与-designmd)）。始终适用的规则通过 **`{{.Rules}}`** 到达模型，而按路径限定的规则会随工具结果或首次触及匹配路径的消息到达。`instructions.files` 默认为空，只会在文档之后添加文件，位于 **`{{.Instructions}}`** 中。

![规则与 AGENTS.md 如何到达模型：一个项目规则文件夹加上你的规则文件夹、决定规则何时适用的头部、各部分在请求中的落点，以及规则生成](../../assets/diagrams/rules-context.png)
*系统消息只包含始终适用的内容，并在一次规则生成期间保持不变；由路径或提及带来的规则只随对话传递一次。*

## 提示词顺序

在渲染后的系统消息中，从上到下依次为：

1. 工具
2. 技能
3. 计划上下文（取决于模式）
4. **规则**：你的 `AGENTS.md` 与 `DESIGN.md`、工作区的规则，然后是始终适用的规则
5. **项目指令**：`instructions.files` 的文件，按列出顺序
6. 会话记忆

`prompts.dir` 下的模板决定其各块的位置，但不决定文档是否到达：系统会检查其渲染结果是否包含这些内容，因此，一个既不输出 `{{.Rules}}` 也不输出 `{{.Instructions}}` 的模板，或仅在为假的条件输出它们的模板，仍会获得文档和 `instructions.files` 的文件（若模板有指令槽位则放入其中，否则放入紧随其后的 "## Project instructions" 块中）。规则不会被移动：始终适用的规则只出现在模板输出 `{{.Rules}}` 的位置，而源中未命名 `{{.Rules}}` 的模板也不会获得按路径限定的规则；用 `@rule:name` 命名的规则仍会到达。无法读取或渲染的模板会由一个简短的回退提示词替代，文档、始终适用的规则和指令文件都会跟在它之后。

### 规则与提示词缓存

提供商按请求前缀进行缓存，而系统消息位于每个请求的开头：那里有一个字节发生变化，就会丢弃其后整个对话的缓存副本。因此，系统消息中的规则块只包含不依赖会话所触及内容的部分 - `AGENTS.md` 与 `DESIGN.md` 对，以及始终适用的规则 - 并且它会在会话启动时**渲染一次**，之后每一轮都逐字节复用。会话期间由智能体或你编辑过的 `AGENTS.md` 也不会改变它：该块会在一次[压缩](compaction.md)之后重新读取（压缩本来就会重写缓存历史），在配置重新加载或工作区切换之后，以及在新的进程中打开会话时重新读取。要立即把编辑过的文件交给模型，请提及它：`@AGENTS.md` 会将其当前文本附加到你的消息中，在对话中位于系统消息保留到下一次生成之前的旧文本之后。

兼容 OpenAI 的提供商会自行缓存足够长的前缀。Anthropic Messages API 只缓存请求标记的内容，因此一个 **`type: anthropic`** 请求会标记两个位置：系统提示词的末尾（这也涵盖其之前发送的工具定义），以及下一个请求将携带的最后一轮。轮次上下文以及 Coddy 为单个请求添加的提示位于该标记之后，因为后续请求不会再重复它们。下一次模型调用随后会以输入价格的一小部分从缓存中读取系统提示词、工具和早先轮次，并且只为新增内容支付缓存写入费用；token 用量中的 `cachedInputTokens` 显示读取了多少内容。在 **`type: openai`** 行中以聚合器名称出现的 Claude 模型，如 OpenRouter 所列的 `anthropic/...`，也以相同方式缓存，因此其请求会携带相同的两个标记，作为消息文本上的 `cache_control`。

按路径限定的规则从不进入系统消息。它随对话传递，而对话本来就会增长，并且只消耗自身长度一次：

- 读取或写入匹配文件的**工具调用**会在**该调用的结果**中、输出之后、一行引导语之下将其带入；嵌套的 `AGENTS.md` 会随进入其文件夹的调用以同样方式到达，子文件夹规则文件夹中的规则则随读取或写入其适用文件的调用到达；
- **提及**会将其作为附件带入**用户消息**（`kind="rule"`）：用户命名的仅提及规则，以及被提及文件或文件夹激活的受路径门控规则、子文件夹规则文件夹中的规则或嵌套的 `AGENTS.md`。见[提及](mentions.md#提及与提示词缓存)。

无论哪种方式，它都只到达**一次**：当携带相同文本的早先结果或消息仍被发送给模型时，后续匹配不会附加任何内容。当压缩将该结果或消息折叠进摘要，或规则文件此后已更改时，下一次匹配的工具调用或提及会再次带来该规则及其当前文本。读取或写入规则文档本身的调用 - `read internal/api/AGENTS.md`、对 `.cursor/rules/go.mdc` 的编辑 - 不会把该文档作为规则再返回：输出已经携带其文本，或者模型正在写入它。Web 界面、控制台和编辑器中显示的结果是工具自身的输出；规则只会随其传给模型。

## 发现

当 `rules.auto_discover` 为 true（默认）时，Coddy 读取你自己的文件夹以及工作区的**一个**项目文件夹（其子文件夹的规则文件夹会在智能体到达它们时稍后读取：[子文件夹的规则文件夹](#子文件夹的规则文件夹)）：

| 系统 | 路径 | 说明 |
|--------|------|-------|
| `user` | `~/.coddy/rules/` | **你的**，不是项目的：在每个工作区都会读取，从不属于检出的内容。其文件夹内同一路径的项目规则（包括扩展名）会覆盖它 |
| `coddy` | `.coddy/rules/` | Coddy 自己的文件夹，最先查看的项目文件夹 |
| `agents-dir` | `.agents/rules/` | 工具中立的 `.agents/` 树，位于 `.agents/skills` 旁边：面向所有参与该项目工作的智能体的规则 |
| `cursor` | `.cursor/rules/` | Cursor 的文件夹，采用 Coddy 自身规则所用的 `.mdc` 方言 |
| `claude` | `.claude/rules/` | Claude Code 的文件夹 |
| `codex` | `.codex/rules/` | 最后；Codex 将其 `*.rules` 命令策略放在这里，这不是提示词规则，因此只计入 `.md` 和 `.mdc` 文件 |
| `agents` | 嵌套 `**/AGENTS.md` 与 `**/DESIGN.md` | 不是规则文件夹：读取 `rules.auto_discover` 与 `rules.systems` 所述内容（[AGENTS.md 与 DESIGN.md](#agentsmd-与-designmd)）。遵循 [agents.md](https://agents.md/) 约定，以及紧邻它的设计说明；**按需读取**，只从工具进入的文件夹读取，从不遍历（见“激活”）；隐藏目录、`node_modules`、`vendor` 不会被进入 |

项目文件夹按表格顺序形成一条链：Coddy 读取**第一个包含规则文件的文件夹**，不再读取其后的文件夹。每个智能体都将自己的规则保存在自己的文件夹中，而与多个智能体协作的项目会在每个文件夹中保留相同规则 - `.cursor/rules/workflow.mdc` 及其 Claude Code 镜像 `.claude/rules/workflow.md` - 因此读取每个文件夹会把每条规则交给模型两次。带有 `.coddy/rules/` 的项目只从那里读取；在 `.agents/rules/` 中为每个智能体编写规则的项目则从那里读取；否则 Coddy 借用 Cursor 的文件夹，若也没有 Cursor 文件夹，则借用 Claude Code 的。不含 `.md` 或 `.mdc` 文件的文件夹 - 空文件夹，或仅包含 Codex 的 `*.rules` - 不计入，链会继续前进。

每个被读取的文件夹都会递归扫描 `.md` 和 `.mdc` 文件；这些是小型规则文件夹，会话启动时不会读取工作区的其他内容。规则以其文件在文件夹内的路径命名，不含扩展名：`.coddy/rules/frontend/style.mdc` 即 `frontend/style`，因此不同文件夹中同名的文件是不同的规则，另一个智能体的镜像也按该路径匹配（[命名规则](#命名规则)）。你自己的文件夹会加入被读取的任一个项目文件夹，当文件在两者中具有相同路径时，项目文件胜出。规则文件夹中的链接会被跟随，被链接的文件夹或文件也会被包含，在工作区文件夹和子文件夹的规则文件夹中都是如此。指向同一规则文件夹内某个文件夹的链接会被跳过，因此其规则保留真实位置的名称；到达两次的文件夹只读取一次；规则文件夹的读取深度最多为 16 层、10,000 个条目。由于会跟随链接，随检出提供的规则文件夹可以引入会话能读取的任何文件：对于你不信任的检出，应像检查其代码一样检查其规则文件夹。嵌套文档以完整路径为键，因此它们绝不会相互合并。

嵌套的 `AGENTS.md` 文件**按需读取**，方式如同 Codex 读取其工作文件夹的 `AGENTS.md` 链；紧邻的 `DESIGN.md` 会随其一起读取 - 一个描述自身的文件夹，无论写的是两者中的哪一个，都会被读取；工作区根目录和智能体主目录读取的也是同一对文件。不会遍历树来查找它们：文件系统工具调用或附加的 `file://` 路径首次指向某个路径时，从项目根（不含）向下到该路径目录的文件夹链上的每个此类文档都会被读取，并随该调用的结果或附加该路径的消息到达。读取 `a/b/c/f.go` 会拉入 `a/AGENTS.md`、`a/b/AGENTS.md` 和 `a/b/c/AGENTS.md`，以及这些文件夹中同时含有 `DESIGN.md` 的那个；无人进入的同级文件夹永远不会被打开，会话启动后写入的文件会在工具进入其文件夹的那一刻被读取。隐藏目录、`node_modules` 和 `vendor` 会终止链。`run_command` 不会激活任何内容：shell 字符串无法可靠地归因到目录。

这正是让带有随附同级检出的仓库保持可用的原因 - 45 个嵌套文件无条件下加载会在第一个问题之前消耗约 131k token 的系统提示词 - 也是让以主目录为锚点的会话不卡住的原因：一次对 `~` 的遍历（仅 `~/Library` 下，macOS 主目录就有数十万个条目，且在文件夹访问提示之后冷读取）曾使控制台在第一帧之前卡住。超过 256 KiB 的正文会被截断，与前两层一样。

**根**的一对文件不属于这一集合：它是文档的第二层，始终位于系统提示词中（[AGENTS.md 与 DESIGN.md](#agentsmd-与-designmd)）。

### 子文件夹的规则文件夹

一个包含多个项目的文件夹，每个项目都是带有自身规则的仓库（包含 `mobile`、`db`、`backend` 等的分组文件夹），会在各自适用的位置获得各自的规则。子文件夹的规则文件夹按需读取，如同嵌套的 `AGENTS.md`：文件系统工具调用首次以某个文件为目标，或提及、附加的 `file://` 路径命名某个文件或文件夹时，从项目根（不含）向下到该文件所在目录、或到该文件夹的链上每个文件夹，都会按与根相同的项目文件夹链进行探测，并读取第一个包含规则文件的文件夹。子项目的 `.cursor/rules` 与其 Claude Code 镜像不会同时读取。隐藏目录、`node_modules` 和 `vendor` 会终止链；调用列出或搜索的文件夹不会读取任何内容，会话启动时也不会遍历任何内容。

子文件夹中的规则适用于该子文件夹及其下所有内容，绝不适用于其上方或旁边：

- 其模式锚定在自身文件夹：`db/.coddy/rules/` 中的 `globs: migrations/**` 匹配 `db/migrations/0001.sql`，不匹配 `db/` 之外的任何内容；
- 若某规则在根处会从第一轮起适用（无模式的 `alwaysApply: true`、无 `paths` 的 `.md` 规则、无 frontmatter 的文件），则改为适用于其文件夹下的每个文件，并随首次读取或写入其中一个文件的调用到达；
- 它从不进入系统消息：它会随首次到达它的工具结果或消息到达一次，并在压缩后或其文件发生变化后再次到达，如同 glob 规则（[规则与提示词缓存](#规则与提示词缓存)）；
- 它以其完整路径标识，因此 `db/.cursor/rules/workflow.mdc` 与根处的 `workflow.mdc` 都会适用，且互不替代；
- 那里的手动规则（无模式、`alwaysApply: false`）在提示词点名它时到达模型：若它位于子文件夹的 `.coddy/rules/` 中，则通过 `@rule:<folder>/<name>` 到达（[命名规则](#命名规则)）；无论它位于哪个规则文件夹，都可通过提及其文件到达。

它们是规则文件夹，因此 `rules.auto_discover: false` 会连同工作区的规则文件夹一起关闭它们，`rules.systems` 会以同样方式收窄它们的链，而不含 `{{.Rules}}` 的模板不会获得其中任何规则。你自己的文件夹 `~/.coddy/rules/` 只针对工作区读取一次，绝不针对子文件夹读取。子文件夹的技能、MCP 服务器和钩子不会被读取：它们会运行代码，并留在工作区信任门禁之后。

工作区自身仓库的 git worktree 会像任何其他检出一样被读取。链绝不会进入隐藏文件夹下的 worktree（`.coddy/worktrees/`、`.claude/worktrees/`），而在 worktree 中打开的会话会将其规则文件夹作为工作区的规则文件夹读取。添加到工作区普通子文件夹中的 worktree（`git worktree add wt/feature`）会作为子项目读取，因此其工作区规则副本会出现在工作区自身规则旁边：请将 worktree 放在隐藏文件夹中或工作区之外。

### 命名规则

规则在提示词中通过其地址命名。目录中的规则 - 工作区的规则文件夹和你的 `~/.coddy/rules/` - 通过其名称寻址，即其文件在规则文件夹内的路径，不含扩展名：`.coddy/rules/style.mdc` 对应 `@rule:style`，`.coddy/rules/frontend/style.mdc` 对应 `@rule:frontend/style`。子文件夹自身 Coddy 文件夹中的规则会在前面加上子文件夹，从工作区书写：`db/.coddy/rules/deploy.mdc` 对应 `@rule:db/deploy`，`db/.coddy/rules/ops/deploy.mdc` 对应 `@rule:db/ops/deploy`。工作区的规则文件夹及其子文件夹的 `.coddy/rules/` 构成一棵地址树，子文件夹的规则文件夹会在发送命名它的消息时读取。

- 当某地址在两个位置命名规则时，更靠近工作区的所有者胜出：目录（工作区的规则文件夹与你的规则文件夹）优先于子项目，因此工作区规则文件夹中的文件夹 `db/` 优先于子项目中的 `db`，`db` 的规则 `ops/deploy` 优先于子项目 `db/ops` 的规则 `deploy`。补全会提供该规则。
- 在子文件夹链终止的子文件夹中，该引用不命名任何内容：工作区本身、工作区之外的文件夹、隐藏文件夹、`node_modules`、`vendor`，以及沿途指回工作区或其上级文件夹的链接。
- 子文件夹下其他智能体的规则文件夹（`.cursor/rules`、`.claude/rules` 等）不会响应它；提及其文件会附加文本。
- 不含 `/` 的名称也会找到目录中以其为最后一段的规则，前提是恰好只有一条规则具有该段：`@rule:lint` 会找到 `solo/lint`，而当 `frontend/style` 与 `backend/style` 同时存在时，`@rule:style` 不命名任何内容。单独的 `@name` 对仅提及规则也是如此。
- 地址由 ASCII 字母、数字、`_`、`-` 和 `.` 组成的段书写，各段均不以点开头，并由单个 `/` 连接。

`@rule:` 的补全会遍历该树。它会在规则之前提供已输入文件夹下一层的文件夹，作为保持选择器打开的步骤：规则文件夹中的文件夹与保留 `.coddy/rules/` 的子文件夹同等对待，若二者同名则只显示一个步骤。步骤之后是地址以该文件夹开头的规则，因此 `@rule:` 列出目录，`@rule:db/` 列出 `db` 以及工作区文件夹 `db/` 的规则，而 `@rule:sty` 会找到地址匹配 `sty` 的每条规则。本身不含 `.coddy/rules/` 的子文件夹（项目分组）需要手动输入（`@rule:apps/`）。`rules.auto_discover: false` 以及不含 `coddy` 的 `rules.systems` 列表会关闭子文件夹的地址。

CLI：`coddy rules list [--cwd DIR]` 会打印发现的目录：来源文件夹（`SOURCE`）、每个文件读取所用的方言（`FORMAT`）、激活模式（`APPLY`：`auto` 或 `mention`）、规则是否出现在每个提示词中（`ALWAYS`：无模式且无目录作用域的自动规则），以及什么会激活其他规则（`ACTIVATES ON`）。表格下方，`Project rules folder:` 指出项目规则的来源文件夹，`Not read:` 指出链中更下方同样包含规则的文件夹 - 通常是另一个智能体对相同规则的副本 - `Only in a folder not read:` 列出那些文件夹中被读取文件夹没有同名文件的文件，即会话缺少的规则，而不是其已有规则的镜像。`Could not read:` 指出链中存在但无法读取的文件夹；链越过了它。嵌套的 `AGENTS.md` 和 `DESIGN.md` 文件以及子文件夹的规则文件夹不在表格中，因为列出它们意味着遍历工作区；表格下方的两行说明它们按需读取。

```text
12 rule(s) under .
Project rules folder: .cursor/rules
Not read: .claude/rules (one project folder is read: the first of .coddy/rules, .agents/rules, .cursor/rules, .claude/rules, .codex/rules that holds a rule file)
```

## 规则文件格式

文件扩展名决定读取规则所用的方言。两种方言在每个文件夹中都被接受，因此 `.agents/rules/` 可以并排存放 Cursor 与 Claude Code 规则，而从 `.cursor/rules/` 或 `.claude/rules/` 复制来的文件仍保持其含义。

| 扩展名 | 格式 | Frontmatter 键 | 不含 `alwaysApply` 且不含模式的头部 |
|-----------|--------|------------------|----------------------------------------------------|
| `.mdc` | Cursor（[docs](https://cursor.com/docs/rules)） | `description`、`globs`（逗号分隔字符串或列表）、`alwaysApply` | 手动：Cursor 将 `alwaysApply` 默认为 false，因此正文仅在 **`@name`** 时进入 |
| `.md` | Claude Code（[docs](https://code.claude.com/docs/en/memory#organize-rules-with-clauderules)） | `paths`（glob 列表）、可选的 `description` | 无条件加载，与 Claude Code 一致 |

两种方言共享：

- 无 frontmatter 的文件从第一轮起生效。
- 模式（`globs` 或 `paths`；任一键在任一方言中均被接受）对规则进行门控：当匹配文件首次进入场景时，正文随带来它的工具结果或消息到达模型。这就是 Cursor 的自动附加与 Claude Code 的路径作用域；Coddy 对携带模式的 `alwaysApply: true` 规则应用同样的门控，以保持提示词精简。
- 显式的 `alwaysApply` 是不含模式头部的总开关：`true` 立即生效，`false` 为仅提及。
- CRLF 行尾、带引号或不带引号的值、`[flow, lists]`、`- item` 列表以及末尾的 `# comments` 都会被读取。

Cursor 的 `.mdc` 头部经常不是有效 YAML：`globs: **/*.go` 以 YAML 别名起始字符开头。Coddy 会先按 YAML 读取头部，再回退到能识别这些键的行读取器，因此此类文件会保留其 description、globs 和 `alwaysApply`，而不会被当作没有 frontmatter 处理。

```markdown
---
description: HTTP layer conventions
globs: external/httpserver/**/*.go, docs/reference/http-api.md
alwaysApply: false
---
```

```markdown
---
description: HTTP layer conventions
paths:
  - "external/httpserver/**/*.go"
  - "docs/reference/http-api.md"
---
```

上面两个头部在 `http-layer.mdc` 与 `http-layer.md` 中含义相同：一旦附加或触及其中某个文件，就会自动附加。

### Glob 模式

模式遵循 Cursor 与 Claude Code：`*` 匹配单个路径段内的内容，`**` 匹配任意数量的目录，`{a,b}` 匹配备选项，`[abc]` 匹配字符类。它们锚定在项目根（会话 cwd），或对于子文件夹规则文件夹中的规则，锚定在其自身文件夹：文件按其相对于该文件夹的路径匹配，因此 `internal/**/*.go` 匹配 `internal/agent/react.go` 但不匹配 `external/x.go`，`*.md` 仅命名根目录中的 markdown 文件（任意深度请使用 `**/*.md`），而工作区之外的文件，如同级检出或模块缓存，绝不会匹配任何内容。在 `.mdc` 文件中，多个模式用逗号分隔（`docs/**/*.md, README.md`）；花括号内的逗号（`*.{ts,tsx}`）不会分隔。

### 从更早版本升级

磁盘上已有的规则文件在此次发布后可能改变模式；`coddy rules list` 会显示结果。

- 项目的规则文件夹不再合并：只读取 `.coddy/rules`、`.agents/rules`、`.cursor/rules`、`.claude/rules` 和 `.codex/rules` 中第一个包含规则文件的文件夹。若某项目过去在比如 `.cursor/rules` 和 `.claude/rules` 中保存不同规则，现在只会得到 Cursor 的规则；`coddy rules list` 会在表格下方指出被跳过的文件夹。将 Coddy 应读取的内容放入 `.coddy/rules` 或 `.agents/rules`，或用 `rules.systems` 收窄链。
- 规则文件夹中的规则按其在其中的路径命名（`frontend/style`），不同文件夹中同名的文件都会被读取；此前，同名的第一个文件胜出，且每条规则仅响应其文件名。当没有其他规则具有该最后一段名称时，`@rule:style` 仍会找到 `frontend/style`；仅提及规则使用单独的 `@style` 时也是如此。
- `@rule:` 引用接受 `/`：`@rule:<folder>/<name>` 命名子文件夹中的规则，因此 `@rule:deploy/rollback`，过去会附加规则 `deploy` 并将 `/rollback` 留作文本，现在命名规则 `rollback`（属于 `deploy/.coddy/rules/`）。
- 子文件夹的规则文件夹过去从不读取；现在当智能体到达其下的文件时会读取，且其中每条规则适用于其所属子文件夹（[子文件夹的规则文件夹](#子文件夹的规则文件夹)）。没有设置可以单独关闭它们：`rules.auto_discover: false` 也会关闭工作区的规则文件夹，而 `rules.systems` 会同样收窄工作区及每个子文件夹的链。
- 受模式门控的规则以及嵌套的 `AGENTS.md`，一旦匹配后不再移入系统提示词：它们会随带来其路径的工具结果或消息到达一次，并在压缩后再次到达。
- `AGENTS.md`、`DESIGN.md` 以及 `instructions.files` 的文件会在会话启动时，以及压缩、配置重新加载或工作区切换之后读取，而不是每一轮都读取。提及 `@AGENTS.md` 可立即将编辑交给模型。
- `AGENTS.md` 与 `DESIGN.md` 层不再依赖任何设置：`rules.auto_discover: false`、不含 `agents` 的 `rules.systems` 列表，以及不含 `{{.Rules}}` 的 `prompts.dir` 模板，过去会丢弃嵌套文档（这样的模板还会丢弃主目录中的那一对文件）；现在它们在所有情况下都会传入。`instructions.files` 的默认值改为空列表，而不是 `["AGENTS.md", "DESIGN.md"]`，并且你写入的列表会在文档之后添加文件，而不是替换默认值；命名这些文档的条目会被跳过，因此现有列表仍可继续工作。`instructions.files` 的文件现在会在 256 KiB 处截断；更早的发布版本会完整发送。
- `alwaysApply: false` 与 `globs` 一起使用时，一旦匹配文件被附加或读取就会自动附加。更早的发布版本会将此类规则保持为仅提及。去掉 `globs` 可将规则保持为手动。
- 不含 `alwaysApply` 的 `.md` 规则遵循 Claude Code：不含 `paths` 时无条件生效，含之时受路径门控。更早的发布版本将其视为仅提及。写入 `alwaysApply: false` 可将 `.md` 规则保持为手动。
- 像 `*.go` 这样不带目录的模式仅匹配项目根目录中的文件。更早的发布版本会在任意深度按文件名匹配它；若要如此，请写入 `**/*.go`。
- 现在会读取不是有效 YAML 的 Cursor 头部（`globs: **/*.go`）。更早的发布版本会丢弃整个头部，因此这些规则总是启用且没有 glob；它们现在会遵循其 `alwaysApply`，并像其他规则一样等待其 glob。

## 激活

| 规则 | 行为 |
|------|----------|
| 模式（`globs` / `paths`），任意 `alwaysApply` | 在匹配文件首次进入场景时**到达一次**：以匹配文件为目标的文件系统工具调用（`read`、`edit`、`write` 等）会在其结果中带来正文；被提及的文件或文件夹（`@src/app.go`、编辑器的 `file://` 附件）则在该用户消息中带来正文。调用列出或搜索的文件夹（对目录执行 `grep`、`glob`）不会带来 glob 规则；读取或写入其中匹配文件的调用才会。压缩将其折叠掉后，或其文件已更改并再次被读取后，下一次匹配会再次带来它 |
| 不含模式的 `alwaysApply: true` | 在会话中立即生效 |
| 不含模式的 `alwaysApply: false` | **从不**自动包含。其正文随命名它的用户消息传递：**`@ruleName`** 或 **`@rule:ruleName`** |
| 无 `alwaysApply`、无模式、`.mdc` | 仅提及（Cursor 的默认值） |
| 无 `alwaysApply`、无模式、`.md` | 立即生效（Claude Code 无条件加载它） |
| 无 frontmatter | 立即生效 |
| 子文件夹规则文件夹中的规则 | 随其文件夹按需读取（[子文件夹的规则文件夹](#子文件夹的规则文件夹)）。在其文件夹下某个文件被首次调用或提及时到达**一次**，前提是该文件匹配其模式（若无模式，则为该处的任意文件）；并在压缩后或其文件变化后再次到达。那里的手动规则在提示词命名它时到达：位于子文件夹 `.coddy/rules/` 中的规则通过 `@rule:<folder>/<name>` 到达，任何规则都可通过提及其文件到达 |
| 嵌套 `AGENTS.md`、`DESIGN.md` | 无论 `rules.auto_discover`、`rules.systems` 和模板怎么说，都按需读取。其目录内的首次文件系统工具调用会将它（以及其上方文件夹链上的所有此类文档）读入该调用的结果；首次提及其中的路径则会读入该用户消息；如同 glob 规则，它只到达一次，并在压缩后或文件变化后再次到达 - 会话期间所做的编辑会随该文件夹中的下一次调用到达模型。系统提示词已包含的文件（在 `instructions.files` 中命名，或通过链接到达）不会再次到来。对于没有工具进入且没有提及命名的文件夹，不会读取任何内容 |

仅提及规则使用 **`@rule:name`**，名称为规则在其规则文件夹内的路径（不含扩展名），或不含 `/` 的名称使用单独的 **`@name`**（文件夹顶层的规则，或没有其他规则共享其最后一段的规则）；`@rule:name` 也会按名称附加目录中的任何其他规则，而 **`@rule:<folder>/<name>`** 会附加 `<folder>/.coddy/rules/` 中的规则（[命名规则](#命名规则)）。它们**不是**斜杠命令，也不会出现在技能目录中。`run_command` 不会激活任何内容：shell 字符串无法可靠地归因到路径。

## AGENTS.md 与 DESIGN.md

文件夹用一对文件描述自身：`AGENTS.md` 与 `DESIGN.md`，Coddy 总是按以下顺序分三层读取它们：

1. **`~/.coddy/AGENTS.md`**，然后是 **`~/.coddy/DESIGN.md`** - 你的智能体主目录，在每个工作区中都会读取，因此你的要求会在检出开始描述自身之前出现在模型面前；
2. 会话文件夹的 **`AGENTS.md`**，然后是 **`DESIGN.md`**；
3. 从会话文件夹下方一直到智能体所操作文件的链上每个文件夹的那一对文件，在文件系统工具或提及首次到达该文件时读取，方式如同 Codex 读取其工作文件夹的 `AGENTS.md` 链，并附加到该工具结果或消息（[激活](#激活)）。

存在哪个文件就读取哪个；只有 `DESIGN.md` 的主目录只会贡献该文件，不会贡献其他内容，缺失文件则不产生开销。前两层位于系统提示词中，第三层稍后随对话到达，因为会话启动后系统消息不会变化（[规则与提示词缓存](#规则与提示词缓存)）。每个文件在 256 KiB 处截断，隐藏文件夹、`node_modules` 或 `vendor` 会终止链。

这些都不需要配置，也无法关闭：`config.yaml` 中没有任何键命名这些文件，`rules.auto_discover: false` 和 `rules.systems` 只涉及规则文件夹，而 `prompts.dir` 下的模板无论输出什么都会获得文档（[提示词顺序](#提示词顺序)）。不应看到任何文档的会话应在没有这类文档的文件夹中运行，并且其智能体主目录也没有这些文档。链从会话文件夹开始：其上方的内容都不会读取，即使会话是在 git 仓库的某个子文件夹中打开的，也不会读取该仓库的根；会话文件夹之外的文件不会带来任何文档。

**每个文件一次。** 一个文件在每次规则生成中至多到达模型一次，以其真实路径识别：链接会被解析，并且在 Windows 和 macOS 上忽略路径大小写。按层顺序首次出现者胜出，因为其文本已位于提示词中，位于任何会重复它的内容之上：

- 在 `~/.coddy` 本身中打开的会话只会读取该对文件一次，作为你的文件；
- 指向你的文件的项目 `AGENTS.md` 就是你的文件，只读取一次；
- 命名层已携带的文档的 `instructions.files` 条目不会添加任何内容，因此仍列出 `AGENTS.md` 与 `DESIGN.md` 的配置会像以前一样工作；
- 系统提示词已通过 `instructions.files` 携带的嵌套文档，或因为该文件夹就是智能体主目录而携带的嵌套文档，在工具进入其文件夹时不会再次附加。

系统提示词的两层会在会话启动时读取，并在其规则有效期间保留：编辑会在下一次压缩、配置重新加载或工作区切换后到达提示词，或通过 `@AGENTS.md` 立即到达。被压缩折叠掉的嵌套文档会随下一次到达其文件夹的工具调用或提及再次到来；这不算重复。

## 你自己的指令与规则

检出会描述自身：如何构建、遵循什么约定、*在其中*哪些命令危险。你希望每个会话都具备的内容 - 回答所用语言、你在项目间保持的风格、你绝不希望运行的命令 - 不是项目的事务，也不应放在其文件中。这些内容位于你的智能体主目录 `~/.coddy` 中，并在每个工作区中读取：

| 文件 | 以何种形式到达提示词 | 读取时机 |
|------|-----------------------|-----------|
| `~/.coddy/AGENTS.md`、`~/.coddy/DESIGN.md` | 文档的第一层，位于项目自身的那对文件之上（[AGENTS.md 与 DESIGN.md](#agentsmd-与-designmd)） | 会话启动时，以及压缩后再次读取，在每个工作区中 |
| `~/.coddy/rules/*.md`、`*.mdc` | 如同任何项目规则：始终适用时为 **`{{.Rules}}`**，否则随带来其路径的工具结果或消息，或命名它们的消息到达 | 按其自身的 frontmatter（始终启用、受 glob 门控或 `@mention`） |

两者都无需配置。写入文件就是打开开关，删除它就是关闭开关，`config.yaml` 中没有任何键提及它们：

```bash
mkdir -p ~/.coddy/rules
$EDITOR ~/.coddy/AGENTS.md
```

你的一对文件会先被读取，因此你的要求会在检出开始描述自身之前出现在模型面前，项目自身的 `AGENTS.md` 与 `DESIGN.md` 紧随其后 - 它们会共同加入，互不替代。写两者之一或两者都写；不存在的文件不会产生开销。

你的规则是普通规则文件，遵循与项目规则相同的方言、glob 和激活模式 - `~/.coddy/rules/` 下带有 `globs: **/*.go` 的 `go.mdc` 会在会话打开的任何项目中等待 Go 文件，因为 glob 锚定在工作区，而不是规则来源的文件夹。`coddy rules list` 会以来源 `user` 显示它们，并在表格下方指出该文件夹。你的文件夹会与链所选的任一个项目文件夹一同读取，绝不会替代它。项目仍有最终决定权：当项目文件和你的某个文件在其规则文件夹内处于同一路径（包括扩展名）时，项目文件胜出。

这里的任何内容都不会与项目文件受到不同的信任：两者都是引导模型的文本，且都不能自行运行任何内容（见[安全](../operate/security.md)）。区别在于作者是谁 - 你自己的文件夹不需要信任凭据，因为其中不会随 `git clone` 到达任何内容。

### 更多指令文件

`instructions.files` 命名会话在上述所有文档之后读入 **`{{.Instructions}}`** 的文件，读取时机与前两层相同。它默认为空且只做添加 - 无论它是否命名这些文档，文档都会被读取 - 因此团队或机器可以列出其特有的内容：

```yaml
instructions:
  files:
    - "/srv/agents/house-style.md"  # an absolute path, shared by a fleet
    - "${CWD}/docs/conventions.md"  # this workspace
    - "${CODDY_HOME}/team.md"       # next to your config.yaml
```

开头的 `~` 会展开，配置理解的另外两个占位符也会展开：`${CODDY_HOME}` 是智能体主目录（`~/.coddy`），`${CWD}` 是读取该文件的会话的工作区。绝对条目按原样读取，相对条目则相对于会话工作目录解析。文件按列出顺序读取，位于文档末尾（"## Project instructions" 块的末尾，会话记忆之前），每个文件位于一个 `### <path>` 标题下，标题按其命名文档的方式命名它：模型知道某段文字来自哪个文件，因此 "follow `infrastructure.md`" 不会让它去读取一个它已经持有的文件。文件不在工作区中的相对条目或 `${CWD}` 条目会被静默跳过 - 这正是单个列表可以服务于并非都携带相同文件的工作区的方式。在每个工作区中都命名同一文件的条目（绝对路径、`~`、`${CODDY_HOME}`）是多个拥有自己配置的智能体共享一组指令的方式；当其文件无法读取时 - 对于此进程它不存在，例如文件夹未挂载到其容器中；在 Coddy 运行所用账户下可能无法读取；它是文件夹或为空 - 会话仍会在没有它的情况下启动，但会说明：日志中会有一条带有条目、路径和原因的警告 `instructions file not read`，该文件在[控制台](../surfaces/console.md)头部的 `[Context]` 下标记为 `(not read: <reason>)`，并在 `coddy --dry-run` 中为该条目显示一行警告（[试运行](../getting-started/configuration.md#试运行探测文件指向的内容)）。存在但无法读取的工作区文件也会以相同方式报告。每个文件都在 256 KiB 处截断，与文档一样。提示词已携带的文件不会读取两次。此处列出但在会话启动时缺失的某个子文件夹的 `AGENTS.md` 或 `DESIGN.md`，仍会在其文件夹出现后作为嵌套文档随其到达；启动时缺失的任何其他文件会在下一次生成（压缩、配置重新加载或工作区切换）时读取。

## 生成规则

使用 **`/rpa-gen-rules`**，它是[标准交付](skills.md#标准交付)中的一项技能，因此在全新安装中就已存在。它会先读取规范、文档和代码，并根据所发现的内容推导规则，而不是要求你描述项目：从较低层依赖向上的分层架构、BDD/TDD 交付，以及用于有意镜像的 Rules Sync。

根 `AGENTS.md` 承载通用基线。Cursor `.mdc` 与 Claude Code `.md` 规则是配对的本地表示，具有等价的正文和激活意图。当请求 Codex 支持时，该技能会在 `.codex/` 下安装一个故障开放的项目钩子，直接读取 `.cursor/rules/*.mdc`；它既不会创建手动 Codex 索引，也不会创建规则正文的第三份副本。关键约束保留在根策略或可执行强制中，其他宿主集成仅在仓库已经记录并测试它们时才会保留。

Coddy 读取一个项目文件夹，即链中第一个包含规则的文件夹（[发现](#发现)）：该技能为 Cursor 和 Claude Code 设置的项目会从 `.cursor/rules/` 读取，其 Claude Code 镜像会保持不动，而不会被加载两次。为 Coddy 自身编写的规则放入 `.coddy/rules/`，若每个智能体都应读取它们，则放入 `.agents/rules/`。

没有 `coddy rules generate` CLI 子命令。

## 上下文明细（UI）

在每个智能体轮次之后，Coddy 会按类别估算 token（`systemPrompt`、`toolDefinitions`、`rules`、`skills`、`mcp`、`conversation`），并在 **`GET /coddy/sessions/{id}/stats`** 上以 `contextBreakdown` 暴露它们。输入框上下文环会在点击时打开一个明细弹出框。`rules` 统计本页描述的所有常驻于系统提示词的内容：`AGENTS.md` 与 `DESIGN.md` 文档、始终适用的规则，以及 `instructions.files` 的文件。

## 配置

```yaml
instructions:
  # Files added after the AGENTS.md and DESIGN.md documents, which are always
  # read and are not named here. Read in the order listed; ${CODDY_HOME}, ${CWD}
  # and ~ expand, an absolute entry is read as it stands, a relative one
  # resolves against the session working directory. Empty by default.
  files: []

rules:
  auto_discover: true
  # Optional filter: user, coddy, agents-dir, cursor, claude, codex. A project
  # folder left out is not part of the chain; of the ones admitted, the first
  # that holds a rule file is read. "agents" is still accepted and changes
  # nothing, the nested AGENTS.md files being always read; a list holding only
  # "agents" still reads no rule folder, an empty list reads them all.
  systems: []
```

## 参考

- [Cursor Rules](https://cursor.com/docs/rules)
- [Claude `.claude/rules`](https://code.claude.com/docs/en/memory#organize-rules-with-clauderules)
- [Codex 钩子](https://developers.openai.com/codex/hooks)
- [Codex `AGENTS.md`](https://developers.openai.com/codex/guides/agents-md)
- 实现：`internal/rules/*`（文件夹链与子文件夹的规则文件夹，`Factory.Nested`，位于 `factory.go`；方言与 glob 匹配位于 `markdown.go`；目录作用域位于 `scope.go`；文档层与去重位于 `project_docs.go` 和 `dockey.go`；嵌套链位于 `agents.go`）、`instructions.files` 的条目位于 `internal/session/instructions_load.go`、常驻部分的按会话渲染位于 `internal/session/rules_load.go` 和 `internal/agent/rules_prompt.go`、它们在模板中的位置位于 `internal/agent/system_prompt.go`；工具路径激活以及工具结果携带的规则位于 `internal/agent/rules_activation.go` 和 `internal/tools/fs/toolpaths.go`
- 规范：`features/rules_one_folder.feature`、`features/rules_agents_dir.feature`、`features/agents_md_scoping.feature`、`features/rules_nested_folders.feature`、`features/global_instructions.feature`，以及提示词缓存组 `features/prompt_cache_rules.feature` 和 `features/prompt_cache_prefix.feature`（`make test-cache`）；会话在此仓库自身规则上的上下文开销由 `BenchmarkContextOfThisRepositoryRules` 测量（`make test-perf BENCH=Context`）
- 针对真实模型的端到端测试：`examples/cli/cli_e2e_rules.py`（通过控制台验证项目 glob 规则）和 `examples/cli/cli_e2e_global_instructions.py`（在隔离的 `CODDY_HOME` 中验证 `AGENTS.md` 与一条规则，工作区为空）

<!-- docsgen:source sha256=2fe457fef4c9d6cb -->

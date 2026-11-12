# 技能

技能是可复用的指令包，为智能体扩展斜杠命令、领域知识和专用工作流。它们为系统提示词中的 **`{{.Skills}}`** 块以及呈现给 ACP 客户端和 HTTP 界面的斜杠命令目录提供支持。

> **项目规则**（`.coddy/rules`、`.cursor/rules` 等）是一种独立机制，作为 **`{{.Rules}}`** 注入。不要把规则放进 `skills.dirs`。参见 [rules.md](rules.md)。

---

## 标准交付

Coddy 在二进制文件中内置了一组技能，并在首次安装时将它们写入 **`${CODDY_HOME}/skills`**。此过程不会下载任何内容：全新安装在首次运行时就拥有这些技能，离线也可用；如果某个技能会读取其 `SKILL.md` 旁边的文件，它会在磁盘上找到这些文件，而不是指向一个不存在的目录。

| 技能 | 作用 |
|-------|--------------|
| **`/configure-coddy`** | 在你要求时更改 Coddy 自身配置：设置、提供商、模型、日志、权限、MCP 服务器和技能。它会验证上游来源，用带类型的 `config_get` / `config_set` 工具暂存 uci 风格编辑，并且只在你确认后提交，因此 `config_commit` 会一步应用并热重载这些更改；`config_rollback` 会回到提交前快照。从不回显密钥。 |
| **`/crossreview`** | 向一组彼此不可见的评审者发送一份评审简报，包括外部控制台代码智能体（Claude Code、Codex、Coddy、Cursor Agent、Devin、OpenCode、Gemini CLI、Qwen Code、Kimi、Koda）和内部的 `explore` 子智能体；该过程通过随附的 `crossreview` 协调器进行，协调器收集每个答案，针对代码验证每个发现，并独自决定修复什么。首次运行（以及之后的 `/crossreview:setup`）会检测已安装的 CLI，询问要使用哪些智能体及其哪些模型，并将名单保留在项目中的 `.coddy/crossreview.json`，或保留在所有项目中的 `${CODDY_HOME}/crossreview.json`，早期版本也把它保存在那里。同一个技能可从 [EvilFreelancer/crossreview](https://github.com/EvilFreelancer/crossreview) 安装到其他智能体中。 |
| **`/rpa-init`** | 预热仓库上下文：记录基线，读取代码、文档和测试代码，按项目文档设置开发环境，运行测试，区分通过/失败/不可用/受阻，并写一份基于证据的简短报告。无需简报。 |
| **`/rpa-feat`** | 通过五道门添加功能：经确认的规格写入 `docs/superpowers/specs/`，经确认的计划写入 `docs/superpowers/plans/`，RED-GREEN-REFACTOR 并为每个完成的任​​务写一行账本，对每个任务的 diff 做独立审查，最后给出收尾菜单。构建步骤不变：意图、小任务、独立子智能体、完整测试套件、文档与示例、质量门控和证据式完成。需要描述要构建的内容。 |
| **`/rpa-bugfix`** | 通过同一份账本和同样的审查门修复 bug：证据、单一可信根因、失败复现测试、最窄修复、记录任务、独立审查、完整测试套件、质量门控和简短报告。需要提供 bug 信息：预期行为与实际行为，以及如何复现。 |
| **`/rpa-gen-rules`** | 根据仓库实际包含的内容，为 Cursor、Claude Code 和 Codex 编写或刷新项目的智能体规则。参见 [rules.md](rules.md#生成规则)。 |
| **`/chinese-code-review`** | 中文代码审查参考：既不回避问题又不让人下不来台的表达方式、分级标注（必须修复/建议修改/仅供参考/问题）、中英混排注释和常见审查反模式。仅显式调用。 |
| **`/chinese-commit-conventions`** | 中文 Conventional Commits 参考：类型与范围约定、subject/body 规则、破坏性变更、Issue 关联，以及 commitlint、husky 和 conventional-changelog 配置。仅显式调用。 |
| **`/chinese-documentation`** | 中文技术文档参考：中英文空格、全半角标点、术语处理、API 与 README 模板，以及机翻味反模式。仅显式调用。 |
| **`/chinese-git-workflow`** | 国内 Git 工作流参考：Gitee、Coding.net、极狐 GitLab、CNB 的接入方式、分支模型、提交规范、CI/CD 示例和 PR/MR 模板。仅显式调用。 |

四个 `chinese-*` 技能是参考资料，不是自动工作流。它们的描述要求模型只在操作者显式调用时加载，因此不会和工作流技能争夺上下文。

写入后，它们就是你主目录中的普通技能：可以编辑、`coddy skills disable <name>`、删除，或从市场更新。交付机制会做什么、不会做什么，记录在 **`${CODDY_HOME}/skills-delivery.json`** 中；该文件位于技能文件夹旁边，而不是其中，因此删除该文件夹不会让下次启动变成首次安装：

- 它从未交付过的技能会被写入：首次安装时写入每个技能，之后发布版本添加到交付中的技能也会写入；
- 你删除的技能会保持删除状态，下次启动不会再次写入；你删除的整个技能文件夹中的技能也不会再次写入；
- 比发布版本更旧的副本会被**替换**，因此 `coddy update` 会带来更新的技能。完全没有声明版本的副本会被视为更旧，它早于这些技能开始携带版本号，因此也会被替换；
- 更新的副本会完全保持原样。这也是保留编辑的方式：把你主目录中副本的 `metadata.version` 提高到发布版本所携带的版本号之上，否则下一个提高自身版本号的发布版本会覆盖它；
- Coddy 无法读取的副本（例如因权限不足而无法读取的 `SKILL.md`）会原封不动。它既不算缺失，也不算无版本，交付机制不会判断它无法打开的内容。

当交付的技能位于磁盘上时，它会被视为从发布它的目录安装（[见下文](#随附的市场)）：`coddy plugin marketplace update` 会报告那里发布了更新版本，`coddy plugin update` 会安装它。你删除的交付技能则不从任何地方安装：任何启动、检查或更新都不会再次写入它。只有删除 `skills-delivery.json` 本身才会这样；没有回执，下次启动会把主目录当作首次安装，并写入交付中所有缺失的技能。早期版本把回执保存在技能文件夹内的 `.bundled.json` 中；下次启动会把它移出，并保留其内容。

如果 `skills-delivery.json` 存在但不可读，交付机制会在该次运行中停止并说明情况，而不是猜测：如果把它读成空记录，就会声称从未交付过任何内容，并把你删除过的每个技能都写回来。替换技能时，旧副本会被重命名到一边，新副本被放入原位；如果进程在两者之间被杀，会留下备份而没有技能，下次运行会把它放回。

Coddy 无法写入的主目录（只读镜像、受限账户）不是错误：二进制文件中的副本会以只读方式代为响应，只有需要其 `SKILL.md` 旁边文件的技能（`references/`，或 `crossreview` 的 `scripts/`）才会注意到差异。你删除的交付技能不在其中：回执说明它已被交付，因此二进制文件不会再次把它作为你无法删除的技能提供。

`crossreview` 位于它自己的仓库中，因此同一个技能可在 Claude Code、Codex 或 Cursor 中工作，并由 **`make skills-vendor`** 内置到 `internal/skills/bundled/`（它的 `SKILL.md`、`references/` 和 `scripts/`）。`rpa-*` 工作流则在本仓库中维护，最初从其上游技能打包而来，以便 Coddy 把意图、计划、TDD、子智能体和验证门控融合进去；`scripts/bundled-skills.json` 将它们标记为本地技能，并说明 `crossreview` 的来源。

### 产物与门禁

两个流程技能把工作写到 superpowers 工作流约定的路径上。被任一工具处理过的仓库只保留一棵树，任一工具都能接着另一方的计划继续。

```text
docs/superpowers/specs/<日期>-<主题>-design.md   操作者在写代码前确认的规格
docs/superpowers/plans/<日期>-<功能>.md          操作者在执行前确认的计划
.superpowers/sdd/<计划名>/progress.md            账本：每个完成的任务一行
                              task-<n>-brief.md  从计划中切出的一个任务
                              task-<n>-tests.log 该任务测试命令的完整输出
                              review-<base>..<head>.diff   一个范围的审查包
```

`.superpowers/` 会写自己的 `.gitignore`（内容为 `*`），因此这棵树不会出现在 `git status` 中，也不会改动检出里被跟踪的忽略文件。账本第一行写明计划，每个完成的任务追加一行 `Task <n>: complete (commits <base>..<head>, tests: <命令> → <最后一行>)`。

五道门，每一道都是停下，而不是一个小标题：

1. **规格。** 技能写下结果、非目标、可选方案与验收标准，然后等待回答。小到不值得单独写规格的改动，把同样的内容并入计划的第一节，也仍然在这里停下。
2. **计划。** 计划为每个任务写明一个行为、一个测试和一条验证命令，在 **Review Focus** 下列出测试没有覆盖的失效模式，并做一次自检。然后再等一次。
3. **证据。** 每个任务以简报和基准提交开始，以该任务的测试命令结束。只有命令以 0 退出时，账本行才会出现。
4. **审查。** 范围 `base..HEAD` 变成一个审查包，由全新的审查子智能体按技能旁的简报（`reviewer-prompt.md`）阅读。Critical 和 Important 的发现用一轮修复，每条都先有失败测试；Minor 的发现作为延后项记入账本。
5. **收尾。** 本地合并、推送并开 pull request，或保留分支。只有第一种会清理 worktree，而且只在合并结果通过测试之后。

**`sdd`** 工具承载这些产物。它接收计划文件和一个动作：`workspace` 返回计划目录，`brief` 切出一个任务，`start` 做同样的事并返回基准提交，`done` 运行该任务的测试命令并记录结果，`review` 为一个范围写审查包。它运行命令的方式与 `run_command` 相同，只有 `done` 可能请求授权：已批准的测试命令也覆盖它。调用它的技能不需要 shell、CLI 或 `PATH` 条目，因此在控制台、HTTP、ACP 和子智能体中流程一致。

账本能在上下文压缩或新会话之后保留下来。先读 `progress.md` 再判断还剩什么：没有账本行的任务不算完成，无论转录里看起来如何。

2.1.0 是第一个带门禁的版本。`${CODDY_HOME}/skills` 中的 2.0.0 副本会被上面的交付规则替换；你自己的修改可以通过把 `metadata.version` 提高到高于发布版本来保留。

### 随附的市场

**`rpa-skills`**（`EvilFreelancer/rpa-skills`）是上游 `rpa-*` 技能所发布到的目录，它是一个**系统市场**：它在效果上等同于交付技能的存在方式，不需要在任何 `marketplaces.json` 中声明，也不需要为它写入任何文件，并且在每个工作区中受信任。它和你自己添加的市场一样是目录，因此它本身不会安装任何内容；在一台没人配置过的机器上，该集合的其余部分只差一条命令：

```bash
coddy plugin marketplace update rpa-skills   # read its list, report newer versions of what is installed
coddy plugin marketplace list rpa-skills     # the plugins it lists, the installed ones marked
coddy plugin install logika@rpa-skills       # install one of them
coddy plugin update                          # install the newer versions the read reported
```

它只是一个地址，仅此而已：Coddy 只会在你要求时联系它。它更新的是磁盘上的技能：交付的技能被视为从它安装，而你删除的技能，无论交付的还是安装的，都不会被任何检查或更新重新安装。因为没有文件声明它，所以也没有文件可把它移除：`coddy plugin marketplace remove` 会按名称或地址拒绝它（除非你的某个 `marketplaces.json` 文件恰好也命名了它，在这种情况下，它会从文件中移除该冗余条目，并告诉你市场本身仍然保留），`DELETE /coddy/skills/sources` 会返回 400，而**设置 → 技能 → 市场**会显示其所在行，带有无法点击的盾牌和禁用的删除按钮。要摆脱它发布的技能，请停用该技能（`coddy skills disable <name>`）或删除该技能（`coddy plugin remove <name>`），而不是删除该目录。

`GET /coddy/skills/sources` 会把它列在最前面，作为具有 `origin: "system"` 的市场 `rpa-skills`，并在 `system` 下写出其地址，客户端据此知道哪些行没有移除控件。

![设置，技能：市场列表，其中包含内置源、一个你自己的源和一个等待批准的项目源](../../assets/skills/skills-marketplaces-dark-1280.png)

*设置 → 技能 → 市场：内置目录 `rpa-skills` 及其始终受信任的盾牌，一个来自你自己文件的源和目录，以及一个项目声明的源，后者会保持等待，直到盾牌为此工作区批准它。*

## 从哪里获取技能

### skills.sh - 社区注册表

开放的智能体技能生态位于 **[https://skills.sh](https://skills.sh)**。技能就是带 `SKILL.md` 文件的普通 GitHub 仓库，兼容支持该格式的智能体（Cursor、Codex、Claude Code、Coddy 等）。

通过 **`npx skills`** 安装（需要 Node.js）：

```bash
# Search the registry
npx skills find [query]

# Install a skill globally into ~/.agents/skills/
npx skills add <owner/repo@skill>

# Update all installed skills
npx skills update

# Check for updates
npx skills check
```

全局技能会落到 **`~/.agents/skills/`**，与任何读取该目录的智能体共享。

### skillsbd - Coddy 精选注册表

**[https://neuraldeep.ru/skills](https://neuraldeep.ru/skills)** 是 **skillsbd** 注册表，专为 Coddy 精选。安装其 CLI：

```bash
npm install -g skillsbd
```

关键命令：

```bash
# Search the registry
npx skillsbd search [query]

# Install a skill
npx skillsbd install <name>

# List installed skills
npx skillsbd list
```

来自 skillsbd 的技能默认也会安装到 **`~/.agents/skills/`**，因此 Coddy 会通过默认的 `skills.dirs` 自动获取它们。

你也可以通过 Coddy Web 界面浏览和安装：**设置 → 技能** 中 **已安装技能** 的安装搜索。

---

## 市场与源

Coddy 可以自行获取技能，无需任何外部 CLI，来源可以是 **GitHub 仓库**、**git URL**，或指向 [agents-standard](https://agents.md) `marketplace.json` 的 **http(s) URL**。Git 源在可用时使用 Git 可执行文件，否则使用内置 go-git 后端；远程暂存位于 `${CODDY_HOME}/tmp` 下，因此最小镜像只需要可写的 Coddy 主目录，而不需要系统 `/tmp`。它可以获取什么，由两个形状相同的文件声明，绝不会在 `config.yaml` 中声明：

| 文件 | 写入者 | 生效时间 |
|------|------------|--------------|
| `${CODDY_HOME}/marketplaces.json` (`~/.coddy/marketplaces.json`) | 你：`coddy skills add`、`coddy plugin marketplace add`、设置 → 技能 | 立即，在所有工作区 |
| 项目的 `.coddy/marketplaces.json` | 项目，由项目提交；`coddy skills add --project`、设置中的 **此项目** | 按 `skills.project_trust` 允许的方式生效（[见下文](#项目市场与信任)） |

```json
{
  "sources": [
    "artwist-polyakov/polyakov-claude-skills",
    "owner/repo@v1.2",
    "https://github.com/owner/single-skill.git"
  ],
  "marketplaces": [
    { "name": "neuraldeep", "source": "https://neuraldeep.ru/skapi/marketplace.json" }
  ]
}
```

- **源**会整体安装：`plugin install <source>` 会安装它发布的每个插件（没有 `marketplace.json` 的仓库中的每个技能），安装的是当时发布的内容，而 `plugin update` 会让已安装内容保持最新。源可以是 GitHub `owner/repo`（用 `@ref` 固定分支或标签）、git URL 或 `marketplace.json` URL。
- **市场**是目录，以其 `marketplace.json` 给出的 `name` 被记住：`plugin marketplace add` 读取其列表，其插件用 `plugin install <plugin>@<name>` 逐个安装，更新只会触及这些插件。

声明其中任何一种都不会下载任何内容，之后也不会有任何内容自行安装：`coddy plugin marketplace update`（也包括 `coddy skills sync`、设置中 **市场** 标题的刷新图标以及 `POST /coddy/skills/sync`）会重新读取列表，并报告已安装技能中存在更新版本的那些；`coddy plugin update` 会安装这些版本，`coddy plugin install` 会安装某个插件或某个源。不会自动获取任何内容。条目安装的所有内容都会进入 `${CODDY_HOME}/skills`，无论它由哪个文件声明，因此项目的市场会把其技能带入你的 Coddy 主目录，而不是带入检出目录。内置的 [`EvilFreelancer/rpa-skills`](#随附的市场) 位于这两个文件旁边。两个文件都声明的条目只计一次，并视为你的条目；市场名称会保留你的文件给它的源。

`coddy skills add <src>` 在你的文件中声明源，`--project` 在项目文件中声明源，`coddy plugin install <src>` 安装它；`coddy plugin marketplace add <src>` 向你的文件添加市场。**设置 → 技能 → 市场** 会列出会话工作区的每个条目，并显示它是什么（**所有插件** 或 **目录**）以及在哪里声明（**内置**、**你的**、**来自项目**）；其添加字段会声明一个源并立即安装，安装到你的文件中；如果选择 **此项目**（且聊天已有会话），则安装到项目文件中。每一行都可以单独检查更新和移除，**市场** 标题的刷新图标会检查全部行。移除某一行只会编辑该行对应的文件；`coddy plugin marketplace remove` 会从两个文件中移除该条目。

### 项目市场与信任

项目的文件会随 `git clone` 一起到达，就像项目的 `.coddy/mcp.json` 及其子智能体定义一样，其中的条目决定什么代码会进入你的 Coddy 主目录。因此，它只会按 **`skills.project_trust`** 允许的方式生效：

- **`ask`**（默认）：该条目会被列出，标记为等待批准，在你为此工作区批准之前既不会被读取，也不会提供安装。可在工作区终端中用 `coddy plugin marketplace trust <name | source>` 批准，或在设置 → 技能中点击其所在行的盾牌。`coddy plugin marketplace list` 和 `plugin marketplace update` 会列出被它们拦下的条目以及批准它们的命令。批准不会安装任何东西：`coddy plugin install <source>` 安装其声明的源，`plugin install <plugin>@<name>` 安装某个市场中的插件。批准绑定到工作区（关联的 git worktree 共享其主检出的批准：[Git worktrees](worktrees.md#worktree-与其主检出共享的内容)）以及条目（其类型、名称和源），并保存在 `${CODDY_HOME}/skills-trust.json` 中；检出重写某个条目后会再次询问，`coddy plugin marketplace untrust <key>` 或盾牌会撤回它。它已安装的技能会保留，直到你移除它们。
- **`allow`**：项目条目立即生效。
- **`deny`**：项目条目永不生效；它们会作为被策略关闭的条目列出。

你自己写入项目的条目（`coddy skills add <src> --project`、设置中的 **此项目**）会在写入时为此工作区获得批准。聊天中的 `/plugin` 无法批准条目：模型可以在聊天中输入，因此命令会改为指向终端和盾牌。

### 从 config.yaml 迁移

早期版本把源保存在 `config.yaml` 中的 `skills.sources` 下，把添加的市场保存在 `${CODDY_HOME}/skills/.marketplaces.json` 中。两者都会自行迁移到 `${CODDY_HOME}/marketplaces.json`：

- 此版本首次加载 `config.yaml` 时，`skills.sources` 中的源会按顺序追加到文件中（它已经列出的源，无论拼写如何，以及内置源都不会重复），并且 `config.yaml` 会被重写为不含该键，旧文件作为 `config.yaml.bak-<timestamp>` 保留在旁边，迁移及其携带内容会记录日志；
- 首次读取市场时，`.marketplaces.json` 记住的市场会被添加到文件中，除非文件已经声明了该名称或源。`.marketplaces.json` 会保留，只保存 Coddy 从每个市场读取到的插件列表。

迁移遵循 `mcp_servers` 的规则（[MCP](mcp.md#从-configyaml-迁移)）：如果要移动到的文件不可读，则不会移动任何内容，保存时会保留该键；位于流式映射中的键（`skills: {sources: [...], ...}`）会移动其源，并保留该键供你删除。`coddy -t` 会把文件中仍然存在的 `skills.sources` 键报告为已迁移，并在此处给出指引。

### `plugin` 命令（CLI 与聊天中的 `/plugin`）

插件和市场管理使用同一个命令界面，在 `coddy plugin ...` CLI 和内置 `/plugin` 聊天命令中完全一致（一种确定性斜杠命令，不经过 LLM 轮次运行，类似 `/compact`）。它接受 Claude Code 接受的词：

```bash
coddy plugin marketplace add <owner/repo | url>          # add a marketplace: read its plugin list, install nothing
coddy plugin install <plugin>@<marketplace>              # install one plugin of an added marketplace
coddy plugin marketplace update [<marketplace>]          # read the lists again and report newer versions, install nothing
coddy plugin update [<name>]                             # install the newer versions of installed skills, or of one
coddy plugin marketplace list [<marketplace>]            # marketplaces and sources, or the plugins of one marketplace
coddy plugin marketplace remove <marketplace | source>   # remove a marketplace or a source; installed skills stay
coddy plugin marketplace trust <marketplace | source>    # approve a project entry for this workspace (terminal only)
coddy plugin marketplace untrust <marketplace | source>  # withdraw that approval
coddy plugin install <owner/repo | url>                  # install every skill a source publishes
coddy plugin remove <name>                               # delete an installed skill (bundled = read-only)
coddy plugin enable <name>   |   plugin disable <name>   # toggle a skill
coddy plugin list                                        # installed skills with versions
```

在聊天中，同样的词跟在 `/plugin` 后。例如，从 [neuraldeep.ru](https://neuraldeep.ru) 目录安装一个技能：

```text
/plugin marketplace add https://neuraldeep.ru/skapi/marketplace.json
/plugin install yandex-wordstat@neuraldeep
```

在终端中：

```bash
coddy plugin marketplace add https://neuraldeep.ru/skapi/marketplace.json
coddy plugin install yandex-wordstat@neuraldeep
```

- **`marketplace add`** 读取市场的 `marketplace.json`，并以它给出的 `name` 记住该市场（上面的 `neuraldeep`），同时记录它发布的插件列表；它不会安装其中任何插件。再次添加会刷新该列表，并回答该市场已添加。没有 `marketplace.json` 的源，或其 `marketplace.json` 未给出名称的源，无法添加；请改用 `plugin install <source>` 整体安装它。市场会在你的 `${CODDY_HOME}/marketplaces.json` 中声明；它读取的插件列表保存在 `${CODDY_HOME}/skills/.marketplaces.json` 中。
- **`install <plugin>@<marketplace>`** 安装生效市场中的一个插件，按该名称查找：可以是你的市场，也可以是项目声明且你已批准的市场。它会先再次读取市场，因此市场在添加后发布的插件无需更新即可安装。市场未添加、仍在等待批准，或插件不在其列表中，都会报错并说明要运行什么。
- **`marketplace update <marketplace>`** 再次读取市场（或按地址读取某个源），保留读取到的列表，并列出从该市场安装且存在更新版本的技能；它不会安装或更新任何内容。不带名称时，它会对所有生效的市场和源执行此操作，包括内置的 `rpa-skills`，`coddy skills sync` 也是如此。等待批准的项目条目不会被读取；它会连同批准它的命令一起列出。
- **`update [<name>]`** 安装更新版本：不带名称时，安装所有已安装技能中存在更新版本的技能；带名称时，安装该技能，或安装该名称对应插件下已安装的技能。它只写入已安装的技能：用 `plugin remove` 移除的技能不会重新安装，即使同一插件的另一个技能被更新也不会；版本高于已发布版本的副本会保持原样。带名称时，如果某个技能的源不发布版本，则会重新安装该技能，因为没有任何信息说明哪个副本更新。标准交付的技能在磁盘上时，视为从 `rpa-skills` 安装。
- **`install <owner/repo | url>`**（无 `@`）会安装源发布的每个技能，安装的是当时发布的内容，包括你移除的技能：它会在你的 `marketplaces.json` 中声明该源，除非工作区的某个文件已经声明了它（项目源需已批准）。没有其他命令会整体安装源；`update` 会让它安装的内容保持最新。
- **`marketplace list`** 探测每个生效的市场和源，并报告它是**有效市场**（agents standard，带有名称、版本和插件数量）、**没有 marketplace.json** 的仓库（直接发现技能），还是**不可达**，并说明其声明位置；等待批准的项目条目会连同批准它的命令一起列出，而不会被探测。带名称时，它会列出该市场的插件并标记已安装的插件。
- **`marketplace trust <key>`** 为终端所在的工作区批准一个项目条目，并打印它批准了什么：声明该条目的文件、它读取的地址以及批准所绑定的摘要。`untrust` 会撤回它。在聊天中，两者都会回答要运行的命令。

`marketplace remove <marketplace>` 会从声明该市场的文件中移除市场（你的文件或项目文件），并连同同地址的源以及它拥有的项目批准一起移除；从它安装的技能会保留，直到 `plugin remove <name>`。`install` 和 `marketplace add` 只触及它们命名的对象：另一个不可达的源不会使它们失败。

`plugin marketplace add` 只添加市场；用 `plugin install <plugin>@<marketplace>` 逐个安装其插件，或用 `plugin install <owner/repo | url>` 一次安装所有插件。

较低层的 `coddy skills` 命令仍用于技能文件本身：

```bash
coddy skills list                                      # all skills (with a VERSION column)
coddy skills enable <name>  |  disable <name>
coddy skills add <src> [--project]  |  sync  |  remove <name>   # declare a source, read the lists, delete
```

三个界面保持能力一致，选择最适合的即可：

- **CLI**：`coddy plugin ...`（以及 `coddy skills ...`）。
- **聊天**：`/plugin ...` 命令。
- **Web 界面**：**设置 → 技能 → 市场**（向你的文件或项目文件添加一个源，并立即安装；用盾牌批准项目条目；用某一行的检查按钮检查该行，或用 **市场** 标题的刷新图标检查全部；移除）；检查发现的新版本会显示为每个技能的下载按钮（工具提示中为 *下载更新*），以及覆盖全部技能的 **更新 N 个技能** 按钮。那里的安装搜索也会提供生效中的市场的插件。

### 版本与更新

市场的 `marketplace.json` 可以为每个插件声明一个 `version`（语义化版本），技能的 `SKILL.md` frontmatter 也可以在其 Agent Skills `metadata` 映射下携带自己的版本：

```yaml
---
name: rpa-feat
metadata:
  version: 1.0.1
description: ...
---
```

顶层 `version:` 键是较早技能使用的形式，仍会被读取；如果文件同时包含两者，`metadata.version` 优先。Coddy 会把已安装版本记录在 `${CODDY_HOME}/skills/.remote.json` 锁文件中，并显示在 `coddy skills list`、`coddy plugin list`、HTTP 技能行以及设置界面中。`coddy plugin marketplace update`（或设置中 **市场** 标题的刷新图标，由 `POST /coddy/skills/sync` 和 `GET /coddy/skills/updates` 支持）会重新读取每个源的清单，并报告哪些技能在上游存在更新版本。`coddy plugin update`（**更新 N 个技能** 按钮，`POST /coddy/skills/update`）会安装全部更新，`coddy plugin update <name>`（每个技能的下载按钮，工具提示中为 *下载更新*，`POST /coddy/skills/{name}/update`）会安装其中一个，只重写该插件已安装的技能。无版本插件不显示版本，也永远不会被标记为有更新（避免误报）。

如果仓库为多个智能体发布清单（用于 Codex 的 `.agents/plugins/marketplace.json` 旁边是用于 Claude Code 的 `.claude-plugin/marketplace.json`），则从第一个清单读取；如果某个插件没有版本，它会采用另一个清单给出的版本，因此一个 Codex 清单没有 `version` 字段的目录（例如 `rpa-skills`）仍会报告其版本。

以 [zip 归档](#以-zip-归档发布的插件)发布、且其条目没有声明 `version` 的插件，会以根据归档本身生成的版本记录：`sha256:` 加上其 SHA-256 的前 12 个十六进制数字，因此归档变化时锁文件也会变化。当市场条目声明归档的 `sha256` 时，更新检查会比较两者，任何变化都算作更新（摘要没有顺序）。两者都没有声明的条目不会被标记；`plugin update <name>` 仍会再次下载它并记录新版本。

### 源如何解析

1. `owner/repo` 简写和 git URL 会被浅克隆并刷新。Coddy 在存在时使用 Git CLI，否则使用内置 go-git 后端；API URL 会作为 JSON 下载。
2. 如果仓库（或 API 响应）是 agents-standard **市场**（`.agents/plugins/marketplace.json` 或 `.claude-plugin/marketplace.json`），则解析列出的每个插件：
   - **外部**源（`{"source":"github","repo":"owner/repo"}` / `{"source":"url","url":"…","ref":"…"}`）会被克隆；
   - **归档**源（`{"source":"archive","url":"https://…/plugin.zip","sha256":"…"}`）会被下载并解包，不使用 git（见[下文](#以-zip-归档发布的插件)）；
   - **相对**源（`"./plugins/foo"`）会从市场仓库内部读取。
3. 如果**没有清单**，则直接扫描仓库中的 `SKILL.md`。
4. 每个发现的技能目录（根级 `SKILL.md`、`skills/<name>/`、`.claude/skills/<name>/` 或 `plugins/<p>/skills/<s>/`）都会被复制，连同其同级的 `scripts/`、`references/`、`examples/`，复制到 `${CODDY_HOME}/skills/<name>/`，由正常的加载器在那里读取。

来源信息记录在 `${CODDY_HOME}/skills/.remote.json` 中。由于同步的技能位于普通技能目录中，`enable`/`disable` 会像处理其他技能一样处理它们；`remove` 会删除副本，任何检查或更新都不会把它装回来；`plugin install <source>` 会，因为它会安装源发布的所有内容。

私有仓库使用其选定的 Git 后端可以访问的凭据。无 Git 环境通过 go-git 支持 HTTPS 和 `file://` 源；SSH 源需要 go-git 在该进程中可以使用的认证机制。HTTP(S) 市场只能指定 HTTP(S) 外部插件 URL，这些 URL 会经过与 API URL 和插件归档相同的 SSRF 防护。操作者明确选择的本地或 SSH 市场可以通过其匹配的非 HTTP 传输解析插件。

### 以 zip 归档发布的插件

市场可以以 zip 归档而不是 git 仓库的形式发布插件，条目形状采用 Claude Code 读取的格式：

```json
{
  "name": "demo",
  "description": "A skill packed by its marketplace",
  "source": {
    "source": "archive",
    "url": "https://example.com/plugins/demo.zip",
    "sha256": "<64 hex digits, optional>"
  }
}
```

只有 `"source": "archive"` 会选择此路径：地址以 `.zip` 结尾的 `url` 源仍会用 git 克隆。安装此类插件（`coddy plugin install https://example.com/marketplace/demo.json`、聊天中 `/plugin` 后的相同词，或设置中的安装搜索）绝不会启动 git：

- **仅 https。** 归档地址以及沿途的每个重定向都必须是 `https`，并且每个都会先经过 SSRF 防护，因此回环、私有、共享（`100.64.0.0/10`）和链路本地地址以及云元数据端点都会在建立任何连接之前被拒绝。
- **sha256。** 当条目声明了 sha256 时，它必须是 64 个十六进制数字，并且每次下载都必须与下载的字节匹配；否则安装会被取消，不写入任何内容。如果市场声明的 sha256 与其提供的归档不匹配，该插件的每次安装都会失败，直到市场修复该条目。
- **限制。** 每个归档最多下载 50 MiB、解包 200 MiB、包含 10,000 个条目。
- **条目。** 带 `..` 元素、绝对路径或盘符的条目，符号链接、设备、管道或套接字条目，以及出现两次的文件，都会拒绝整个归档；每个条目都会在写入第一个条目之前接受检查，并且不会在临时文件夹之外写入任何内容。macOS 归档工具添加的 `__MACOSX` 文件夹会以同样方式检查，然后跳过。
- **插件根目录及其技能。** 插件根目录是归档顶层，或包裹所有内容的单一文件夹（即 GitHub 打包仓库的方式）。当它携带插件清单 `.claude-plugin/plugin.json`（或 `.codex-plugin/plugin.json`）时，只安装清单指定的技能：其 `skills` 字段列出的文件夹（相对于根目录的路径或路径列表，每个路径要么是技能文件夹，要么是技能文件夹的文件夹；离开根目录的路径会被忽略），或在它未列出任何内容时 `skills/` 下的文件夹，或者当 `skills/` 也没有包含技能文件夹时，如果插件根目录本身有 `SKILL.md`，则安装该根目录：即一个本身就是单个技能的插件，名称取该 `SKILL.md` 的 `name`，否则取插件的名称。插件中其他位置（其测试数据或示例中）的 `SKILL.md` 不会被当作技能，技能必须是文件夹：扁平的 `skills/<name>.md` 不算技能文件夹。没有清单的归档会像克隆一样搜索每个 `SKILL.md`；git 插件也会以这种方式搜索，因此其根目录中的 `SKILL.md` 也会被发现。
- **可执行脚本。** 归档标记为可执行的文件会以 `0755` 写入，其他文件以 `0644` 写入。

锁文件会在市场旁边记录归档地址，以及 [版本与更新](#版本与更新) 中描述的版本。

---

## 目录布局

![设置，技能选项卡：自动发现、额外技能目录、项目市场策略、市场与已安装技能](../../assets/screenshot-fullhd-settings-skills.png)

*设置，技能选项卡：自动发现、skills.dirs 的额外目录和项目市场策略、包含内置市场的市场列表，以及已安装技能，其中包括随附的技能*

每个工作区都会读取四个技能文件夹，无论 `config.yaml` 说什么，然后读取 `skills.dirs` 的额外目录。它们按此顺序读取，**文件夹在列表中越靠后，优先级越高**：如果某个技能的名称在多个文件夹中出现，则采用最后一个文件夹中的版本。

| 顺序 | 文件夹 | 内容 |
|-------|--------|---------------|
| 1（最弱） | `${HOME}/.agents/skills/` | 你与每个智能体共享的技能；`npx skills` 和 `npx skillsbd` 安装到这里 |
| 2 | 项目的 `.agents/skills/` | 项目与每个处理它的智能体共享的技能 |
| 3 | `${CODDY_HOME}/skills/`（默认 `~/.coddy/skills/`） | Coddy 自己的技能：标准交付、市场与插件安装、从设置安装的技能 |
| 4 | 项目的 `.coddy/skills/` | 项目供 Coddy 使用的技能 |
| 5（最强） | `skills.dirs` 的每个条目，按其顺序 | 你自己添加的目录，例如团队文件夹 |

因此，项目的 `.agents/skills` 会覆盖你的 `~/.agents/skills`，Coddy 的文件夹会覆盖两个 agents 文件夹，项目的 `.coddy/skills` 会覆盖 Coddy 自己的文件夹，而 `skills.dirs` 中的目录会覆盖全部四个。四个默认文件夹无法通过配置去掉：`skills.dirs` 只会增加它们，没有该键的配置只读取这四个默认文件夹。

```yaml
skills:
  dirs:
    - "~/my-team-skills"        # read after the four defaults, wins a name over them
    - "${CWD}/tools/skills"     # a folder of the workspace, like a relative "tools/skills"
```

- `${HOME}` 和 `~` 在加载配置文件时展开为主目录，`${CODDY_HOME}` 也会展开。`${CWD}`，以及写成相对路径的条目（`.agents/skills`、`tools/skills`），指的是**会话工作区**中的文件夹，而不是服务器启动所在目录中的文件夹。
- 被命名两次的文件夹（在 `skills.dirs` 中再次写出的默认文件夹，就像此布局之前编写的配置那样；或项目打开在你的主目录中，导致 `${HOME}/.agents/skills` 和项目的 `.agents/skills` 相同）只会读取一次，**在它最后出现的位置**。因此，在 `skills.dirs` 中命名默认文件夹会把它移到列在它前面的目录之下。指向已在列表中的文件夹的链接，视为该文件夹本身。
- 不存在的文件夹会被安静跳过，因此对于既没有 `.agents/skills` 也没有 `.coddy/skills` 的项目，默认文件夹不会带来任何开销。
- `coddy skills list` 会按此顺序在 **Search roots:** 下列出它读取的文件夹；设置 → 技能会显示每个技能及其来源文件。

技能可以是指向磁盘上任意位置的符号链接，包括项目外部：链接到 `.coddy/skills` 或 `.agents/skills` 的技能文件夹（`ln -s ~/shared-skills/review .coddy/skills/review`），以同样方式链接的单个 `.md` 技能文件，本身是链接的 `SKILL.md`，或整个技能文件夹（`.agents/skills`、`.coddy/skills`、`~/.agents/skills`）作为指向技能文件夹的链接。每种形式都会被列出，用 `/name` 调用，并提供给 `load_skill`；模型会通过链接路径（`Skill directory: <project>/.coddy/skills/review`）得知技能的文件夹，因此技能指定的文件（`scripts/`、`references/`）会从那里读取和运行。指向不存在位置的链接会被跳过。从设置 → 技能删除这样的技能会移除链接，并保留它指向的内容；当整个技能目录是链接时，被删除的是其中的技能文件夹，与任何文件夹一样。

`${CWD}` 由会话解析，而不是由进程解析。从任意目录启动的 `coddy serve` 服务器（例如从 `$HOME` 启动的用户服务）会向每个工作区为该项目的会话提供项目本地技能：在创建会话时选择文件夹（输入框的工作区选择器、`POST /coddy/sessions/{id}/workspace`，或带 `cwd` 的 ACP `session/new`）。一旦对话有了消息，工作区就固定下来，因此运行中的聊天会保留它启动时所在文件夹的技能。`GET /coddy/slash-commands` 和 `GET /coddy/skills` 会通过 **`X-Coddy-Session-ID`** 获取会话；没有它时，它们会在 **`cwd`** 查询中描述文件夹；两者都没有时，则使用服务器默认工作区，`coddy skills list` 对其运行所在目录打印的也是这个工作区。

Web 界面中的新聊天在第一条消息之前没有会话，因此开始屏幕上选择的文件夹会改为作为这些请求的 **`cwd`** 查询传递。选择 `data` 并输入 `/` 时，会立即列出 `data/.agents/skills` 和 `data/.coddy/skills` 的项目技能，位于全局技能旁边；选择一个没有这些技能的文件夹，它们就会离开菜单，而另一个工作区的 `/name` 不再在输入框或对话记录的消息中标记为技能。设置 → 技能会列出同一文件夹的项目技能；第一条消息发送后，会话就会在其中创建，因此该轮次会加载相同的技能。

---

## 支持的文件格式

### `subdir/SKILL.md`（推荐）

每个目录一个技能。兼容标准智能体技能布局和 `npx skills`：

```
~/.agents/skills/
  code-review/
    SKILL.md
  docker-helper/
    SKILL.md
```

### 技能目录中的根级 `.md` / `.mdc`

在 `skills.dirs` 条目根目录下的扁平文件也会注册为技能（文件名主干成为斜杠名称）。

### YAML frontmatter

每个技能文件都必须有一个 frontmatter 块，其中恰好包含两个字段，两个字段都必填且非空：

```markdown
---
name: code-review
description: One-line summary shown in the slash-command catalog and UI.
---

# Code Review

Full skill body here...
```

**`name`** 设置规范斜杠命令标识符（例如 `/code-review`）。设置后，它会覆盖从文件系统推导出的名称。`description` 会在目录和设置 → 技能面板中显示。

`routes` 是一行"何时使用"，它把技能放进系统提示词的路由提示。声明了它的技能会在斜杠命令目录之后一起列出，作为在创建、修改、调试或审查代码之前要加载的流程，因此模型不必等到自己的斜杠命令就能进入工作流：

```markdown
---
name: rpa-feat
description: Coddy 功能工作流，包含规格、计划、账本和审查。
routes: 用户要求新功能、行为变更或重构
---
```

没有 `routes` 的技能只会通过它的斜杠命令和 `load_skill` 提示提供。

两个可选字段决定技能运行时所使用的内容。`model` 指定已配置的模型 id，`reasoning`（别名 `effort`）指定该模型提供的一个级别，`off` 或 `default`；当技能被调用时，无论是通过提示词中的 `/code-review`，还是通过模型的 `load_skill`，它们都会在该轮次剩余时间内生效，会话自身的设置会在下一轮恢复（[会话设置](session-settings.md#按请求更改模型)）：

```markdown
---
name: code-review
description: Review the pending change for correctness.
model: rpa/qwen3.8-27b
reasoning: high
---
```

---

## 启用或停用而不卸载

```bash
coddy skills list              # show all skills with enabled/disabled status
coddy skills disable <name>    # skip a skill without removing it
coddy skills enable <name>     # re-enable
```

停用状态存储在 `~/.coddy/skills/.disabled` 中（纯文本，每行一个名称）。

---

## 编写自己的技能

在任意位置创建一个目录，并添加 `SKILL.md`：

```markdown
---
name: my-skill
description: Short description shown in the catalog.
---

# My skill

Instructions the agent will follow when this skill is active.
```

然后把这个目录放入某个默认文件夹（`~/.agents/skills/`、项目的 `.agents/skills/`、`~/.coddy/skills/`、项目的 `.coddy/skills/`），或将其父目录添加到 `config.yaml` 中的 `skills.dirs`。

在运行中的智能体会话里，通过暂存配置工具（`config_set` + `config_commit`）提交对 `skills.dirs` 或 `skills.auto_discovery` 的更改，会立即重建技能目录。外部安装器（例如 `coddy plugin install` 或 `npx skills add`）会更改磁盘上的文件，因此随后要对 `set skills.dirs=[...]` 执行一次幂等提交，以刷新运行中的加载器。仅在 `marketplaces.json` 中声明源，不会获取或安装任何内容。

要与其他智能体共享，请发布到 GitHub 并在 [skills.sh](https://skills.sh) 上列出，或提交到 [neuraldeep.ru/skills](https://neuraldeep.ru/skills)。

---

## 技能何时被读取

会话在启动时从磁盘读取技能（控制台的第一个会话、`session/new` 或通过 ACP 的 `session/load`、Web 界面中的新聊天），在工作区变化时以及配置重新加载时也会读取；此过程不会通过网络读取任何内容。

- **启动时读取，包括正文**：工作区读取的文件夹（[目录布局](#目录布局)）中的每个 `SKILL.md`（以及根级 `.md` / `.mdc` 技能文件），每个技能约 0.1 ms，因此安装一千个技能会给控制台首帧增加约 100 ms；二进制内置的技能保存在内存中。
- **启动时从不读取**：`marketplaces.json` 文件中的源和市场。源的清单或仓库只会在有人请求时获取，包括 `coddy plugin marketplace update`、`coddy plugin update`、`coddy skills sync` 和 `/plugin`、设置 → 技能（市场列表的检查、安装搜索、某个技能的下载按钮、**更新 N 个技能**）、`GET /coddy/skills/updates`，因此大型市场或没有响应的源永远不会延迟会话。

可能拖慢启动的是配置的 MCP 服务器；控制台会在首帧之后连接它们（[MCP 服务器](mcp.md#mcp-服务器生命周期)）。

## 技能如何应用

在每个 `session/prompt`，智能体会：

1. 使用会话从其工作区文件夹中读取的技能（[技能何时被读取](#技能何时被读取)）。
2. 所有已加载（并启用）的技能始终处于激活状态，其正文可作为斜杠命令使用，并按需注入。
3. 构建 **`{{.Skills}}`** 系统提示词块：列出所有技能的斜杠命令目录，以及任何始终激活或通过 glob 匹配、且名称**尚未**出现在目录中的技能的完整正文。
4. 在用户输入的文本中查找 `/name` 调用，并把每个匹配技能的正文**追加到用户消息中**，作为输入文本之后的 `<coddy_attachment path="skill:name" kind="skill">` 元素。该消息会连同正文进入**会话历史**，因此后续轮次会重放相同字节：提供商的缓存前缀得以保留，模型会保留给它的指令，直到压缩把该消息并入摘要（[提及与提示词缓存](mentions.md#提及与提示词缓存)）。对话记录会显示输入时的消息，因为 Web 界面会丢弃 `kind="skill"` 元素，重新打开的控制台或 ACP 编辑器收到的历史重放也是如此。轮次期间排队的后续消息也会以同样方式获得其技能正文。

模型自己通过 `load_skill` 工具加载的正文（在 `skills.auto_discovery` 开启时提供）会作为该调用的结果返回，并像任何其他工具结果一样留在会话历史中。

无论哪种方式，磁盘上的技能文件夹到达时都会以一行开头：`Skill directory: <path>`，即读取其 `SKILL.md` 的文件夹。技能通过相对路径指定自己的文件，如 `scripts/`、`references/`；如果没有这一行，模型就必须在磁盘上搜索它们。标准交付中从二进制读取的技能（Coddy 无法写入的主目录）会指向交付写入 `${CODDY_HOME}/skills` 的副本（如果存在）；磁盘上没有这样的副本时则没有这一行。

ACP 客户端会收到 `available_commands_update`，其时机在 `session/new` 与 `session/load` 之后。HTTP 界面会查询 `GET /coddy/slash-commands` 以进行自动补全。

---

## 参考资料

- 实现：`internal/skills/`，接线位于 `internal/session/`、`internal/agent/system_prompt.go`、`internal/agent/react.go`
- 配置参考：[config.md](../getting-started/configuration.md) → `skills`
- 规则（独立机制）：[rules.md](rules.md)
- Web 界面：设置 → 技能（需要 `coddy serve`）

<!-- docsgen:source sha256=9b46f2caf4abe74c -->

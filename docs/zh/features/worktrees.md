# Git worktree

Coddy 可以为会话提供自己的功能分支和工作副本。会话的工作目录会移入该 worktree，因此其下一次 `run_command`、文件工具、规则查找和项目范围的 MCP 连接都会使用该目录，而无需手动 `cd`。

在智能体对话中，请 Coddy 在功能分支上工作。`worktree_open` 工具接收 `{"branch":"feature/login"}`，并将会话移入持有该分支的 worktree，或为其打开一个：它会获取 `origin`，从 `origin/HEAD` 读取默认分支，并根据最新的 `origin/<base>` 提交创建新分支。仅存在于 `origin` 上的分支会从 `origin/<branch>` 检出并建立跟踪；本地已存在的分支会在其自身顶端复用，已在某个 worktree 中检出的分支则复用该 worktree。该工具会拒绝默认分支本身，以及跟踪 `origin/<base>` 的本地分支。

新分支也可以从会话所在位置开始：使用 `"base":"head"` 时，工具会从会话检出所在的提交开始创建分支，因此下一个分支会堆叠在进行中的工作之上，即使会话本身就在某个 worktree 中工作也是如此。已存在的分支无论 `base` 说什么，都会保留其自身顶端。两种情况下，新 worktree 都会放在主检出的 `.coddy/worktrees/` 下，绝不会放在打开它的那个 worktree 内部。使用 `{"path":"..."}` 而不是分支时，`worktree_open` 会进入一个已存在的 worktree，其文件夹为绝对路径或相对于主检出。它不会创建或获取任何内容，因此可以到达处于分离 HEAD 的 worktree，或另一个工具打开的 worktree；主检出本身则由 `worktree_exit` 负责。

`worktree_list` 会显示仓库的每个 worktree，首先是主检出，以及每个 worktree 的状态：其分支和最后一次提交、未提交和未推送的更改、领先和落后于 origin 默认分支的提交、是否被锁定或其文件夹是否已消失、会话在哪个 worktree 中工作，以及其他会话分别在哪个 worktree 中工作。它不会移动任何内容，并且在 Plan 和 Ask 模式下也会提供。为了不移动过去而更仔细地查看某个 worktree，智能体会在其中使用 `run_command` 及其 `cwd` 运行 `git`。

当 worktree 中的工作完成时，`worktree_exit` 会将会话移回主检出，其下一次工具调用会在那里运行。worktree 会发生什么取决于 `action`。`{"action":"auto"}` 是默认值，当 worktree 不包含自身内容时，它会移除该 worktree 及其分支：没有未提交的更改（被忽略的文件不算，除非 `.worktreeinclude` 命名的文件与主检出的不同，[下文](#git-不携带的文件)），并且没有任何其他分支、远程分支或标签持有的提交，没有其他人在其中工作，以下也没有任何情况拒绝其移除。否则它会保留 worktree，会话仍会返回，答案会在 `kept` 中说明原因。因此，为了四处查看而打开的 worktree 会在返回时消失，而包含工作的 worktree 会留待以后处理。`{"action":"keep"}` 无论如何都会将 worktree 及其分支留待以后处理；使用同一分支的 `worktree_open` 会回到该 worktree。`{"action":"remove"}` 会移除 worktree，或拒绝离开。它会拒绝有未提交更改的 worktree，除非 `discard_changes` 为 `true`；并且，无论 `discard_changes` 怎么说，它都会拒绝以下 worktree：位于没有任何分支、远程分支或标签持有的提交上的 worktree（带有自身工作的分离 HEAD）、已初始化子模块的 worktree（它们自己的提交位于该 worktree 的 git 目录中）、未跟踪文件夹中包含自身 git 仓库的 worktree（快照只会保留指向其提交的指针）、被锁定的 worktree、本 Coddy 进程的另一个活动会话或任一会话的后台任务仍在其中工作的 worktree（同一 worktree 中另一个 Coddy 进程的聊天不会被看到），以及 Coddy 未打开的 worktree（位于 `.coddy/worktrees/` 之外）。被丢弃的更改不会丢失：在 worktree 移除之前，Coddy 会像 `git stash` 那样保存它们，即索引和文件，保存在 `refs/coddy/worktree-snapshots/` 下，并在答案中命名该引用；在分支上执行 `git stash apply --index <ref>` 会将它们按原样恢复，包括使用 `--force` 添加到 `.gitignore` 之后的文件。被忽略的文件（例如构建输出）会随 worktree 一起移除。只有当不会丢失任何内容时，分支才会随 worktree 一起移除：即另一个分支、远程分支或标签持有该分支的每一个提交，就像合并到默认分支、已推送或堆叠在另一个分支下的分支那样。带有自身提交的分支会保留，答案中会说明这一点。拒绝会让会话留在原处。如果某个会话的 worktree 文件夹已消失，被手动删除或被另一个聊天删除，那么同一个调用会让该会话回到主检出，git 也会忘记该 worktree。与 `worktree_open` 一样，该工具会在当前权限模式下请求批准，仅在智能体模式下运行，并且绝不会提供给子智能体：子智能体改为通过隔离获得自己的 worktree（[在自有 worktree 中的子智能体](#在自有-worktree-中的子智能体)）。

操作员可以在控制台、Web 界面和 ACP 编辑器中使用 `/worktree` 命令执行同样的操作，而无需模型的一个轮次。`/worktree <branch>` 会将聊天移入该分支的 worktree；如果不存在，则打开一个（`--from-head` 会在当前提交处启动一个新分支）；`/worktree <path>` 会将其移入一个已存在的 worktree（路径以 `/`、`./`、`../` 或 `~` 开头，并从聊天的文件夹读取）；`/worktree exit` 会将聊天带回主检出，并在该 worktree 不包含自身内容时移除它，正如该工具的 `auto` 所做的那样（`--keep` 仍会保留它，`--remove` 会移除它或拒绝，而带有 `--remove` 的 `--force` 会丢弃其未提交的更改，并将其保存为快照）；单独使用 `/worktree` 会说明聊天在哪里工作以及仓库的每个 worktree 位于何处，正如 `worktree_list` 所做的那样。该命令会进行与这些工具相同的检查和拒绝，但作为操作员输入，它不请求批准，并在所有会话模式下工作。消息机器人会像对待其他内置项一样将其丢弃，而机器人对非其管理员的用户所限制的轮次会拒绝它。

创建需要一个 `origin` 远程，并在当前工具权限模式下获得批准；复用已存在的 worktree 既不需要网络，也不需要 `origin`。拒绝或获取失败会使会话留在原目录。该工具报告的其他错误包括：无效的分支名称、解析不到任何内容的 `origin/HEAD`、该分支已在主检出中检出，以及磁盘上已存在某个 worktree 路径。cwd 位于 git 子模块内的会话会将 `repo_root` 解析为空，并在历史记录中归组到它们自己的文件夹下。

主检出只会通过安全的快进推进到获取到的默认提交：该提交必须已经是祖先，并且已检出的默认分支必须拥有干净的 worktree。脏的或已分叉的主检出会保持不变；新的功能分支仍会从获取到的远程提交开始。智能体的轮次上下文会命名主检出路径和默认分支。当某个命令必须在主检出中执行一次调用时，请使用带有 `cwd` 的 `run_command`；省略 `cwd` 会使用会话的 worktree。

Coddy 将 worktree 保存在 `<main checkout>/.coddy/worktrees/` 下；例如，`feature/login` 会变成 `feature-login`。该父文件夹中的 `.gitignore` 会从主检出的状态中隐藏这些 worktree。项目规则、技能、钩子和配置的 MCP 服务器会为新 cwd 重新加载，项目 MCP 声明会通过工作区信任门检查；主检出已批准的内容在那里仍然有效，其运行的项目 MCP 服务器会被共享，而不是再次启动（[worktree 与其主检出共享的内容](#worktree-与其主检出共享的内容)）。会话会记录其新的 cwd 以供后续轮次使用。

仅存在于 `origin` 之外的远程上的分支也会以相同方式打开：通过其名称（`fix/typo`，当 `origin` 也有该分支时从 `origin` 获取，否则从按名称排序的第一个拥有该分支的远程获取），或通过其远程跟踪名称（`upstream/fix/typo`）来选择远程。无论哪种方式，都会在远程分支处创建本地 `fix/typo` 并跟踪它；已存在且已跟踪它的本地 `fix/typo` 会被复用，而跟踪其他内容的本地分支会被拒绝，而不是静默打开。该工具的答案会命名本地分支。

在 Web 界面中，在第一条消息之前，使用输入框上方面板上的 **Worktree** 复选框选择功能分支。相同的获取、分支检查和 worktree 放置规则适用。分支列表在打开之前会获取每个远程，并列出仅存在于某个远程上的分支，将其与本地分支一起显示，并用云标记，因此从另一台机器推送的分支无需终端即可打开；请参阅[按会话划分的工作区](../surfaces/web-ui.md#每会话工作区文件夹--分支--worktree)。聊天运行后，面板会显示主项目名称和功能分支，分支的工具提示会命名 worktree；历史记录会将主检出及其 worktree 归组到该项目下。会话仍在 worktree 内运行。对话开始后，Web 工作区选择会让位于该面板；智能体工具可以在轮次期间移动其自己的会话。

该面板在任何宽度下都没有 worktree 退出按钮或离开对话框。使用 `/worktree exit --keep`、`/worktree exit --remove` 或 `/worktree exit --remove --force` 返回主检出，或让智能体使用 `worktree_exit`。上述移除检查和快照仍然适用。当 `worktree_open` 或 `worktree_exit` 完成时，面板会在轮次仍在运行时重新读取工作文件夹，并在聊天的每个轮次之后重新读取，因此它无需重新加载即可跟随这些工具和 `/worktree`。

![历史记录归组在主项目下，输入框上方的面板命名项目及 worktree 的分支](../../assets/git-worktree/worktree-project-group-dark-1920.png)

*功能会话在其 worktree 中运行，而面板和历史记录命名主项目。*

`GET /coddy/workspace/context` 会报告 `repo_root`（主检出）和 `base_branch`（来自 `origin/HEAD`），以及当前的 `path` 和 `branch`。会话列表行带有 `repoRoot` 以用于分组。有关这些界面，请参阅 [Web 界面](../surfaces/web-ui.md#每会话工作区文件夹--分支--worktree) 和 [HTTP API](../reference/http-api.md)。

如果之后使用 `git worktree remove` 移除了受管理的 worktree，则指向 `<main>/.coddy/worktrees/` 内部的已保存会话会改为针对主检出打开。历史记录会立即将其归组到那里；原始记录的 worktree 路径会保留，直到你明确选择另一个工作区。

## git 不携带的文件

新的 worktree 是其分支的全新检出，因此 git 忽略的文件不会出现在那里：项目运行所用的 `.env`、本地的 `.coddy/mcp.json`、`secrets/` 下的密钥。请将它们列在主检出根目录的 `.worktreeinclude` 中，每行一个模式，使用 `.gitignore` 语法，这样 Coddy 创建的每个 worktree 都会获得一份副本，副本包含匹配某个模式且被 git 忽略的文件：

```gitignore
# .worktreeinclude
.env
.coddy/mcp.json
secrets/
```

该约定来自 Claude Code，因此已经拥有该文件的仓库在两者中都以相同方式工作。只会复制被忽略的文件：git 跟踪的文件会随分支而来，而 git 未忽略的未跟踪文件无论模式如何命名都不会被包含。只有当新 worktree 的分支也忽略某个文件时，才会复制该文件，因此早于忽略规则的分支不会获得可能被意外提交的副本。副本会保留文件的模式（仅你可读的 `.env` 仍会如此），并且绝不会覆盖 worktree 中已有的文件。链接不会被复制，其文件夹在任一侧检出中为链接的文件也不会被复制；复制创建的文件夹会获得主检出文件夹的模式。复制发生在创建 worktree 时，无论是通过 `worktree_open`、`/worktree` 还是 Web 界面的 **Worktree** 复选框，子智能体的 worktree 也是如此。`worktree_open` 会在 `included` 中列出它复制的内容。没有该文件时不会复制任何内容。在 worktree 中更改过的副本，或 include 文件命名但只有该 worktree 拥有的文件，都算作其自身的工作：使用 `auto` 离开会保留该 worktree，而其他被忽略的文件（例如构建输出）会随之移除。

## 在自有 worktree 中的子智能体

在父文件夹中编辑文件的子智能体会将其更改与父级的更改混在一起，而两个并行工作的子智能体会将各自的更改相互混合。启用隔离的子智能体则会在自己的 worktree 中工作：定义在其 frontmatter 中使用 `isolation: worktree` 请求隔离，父模型在 `spawn_agent` 调用中使用 `"isolation":"worktree"`。子智能体会获得一个新分支 `subagent/<name>-<id>`，位于父级检出所在的提交处，从该提交创建，无需获取，也不需要 `origin`，并位于主检出的 `.coddy/worktrees/` 下的一个 worktree 中，同时复制 `.worktreeinclude` 文件。父级的未提交更改不在那里。如果父级在其检出的子文件夹中工作，则会将 worktree 的同一子文件夹交给子智能体。

运行结束时，如果子智能体没有留下任何内容，worktree 会随其分支一起移除：没有任何 `worktree_exit` 的 `auto` 会因此保留它，完全没有提交（甚至不是推送或另一个分支持有的提交），并且没有其他会话或后台任务在其中工作。更改了文件或进行了提交的运行会保留 worktree，父级读取的报告会命名其文件夹和分支：父级在那里审查更改、合并分支，或使用 `worktree_open` 打开它。智能体任务会将它们记录为 `worktree` 和 `branch`。恢复的运行会在其先前运行工作过的 worktree 中继续，处于子智能体留下的任何分支上，并且当那次运行的结束移除了 worktree 时，会重新检出同一分支。曾在父文件夹中工作的运行不会因恢复而被移入 worktree。隔离需要一个带有提交的 git 仓库：在其他任何地方，启动生成之前都会被拒绝。请参阅[子智能体](subagents.md#自己的-worktree)。

## worktree 与其主检出共享的内容

链接的 worktree 与其主检出是同一个项目，因此 Coddy 对该项目作出的决定在其每个 worktree 中都有效。只有 worktree 本身添加或更改的内容会再次询问：

- 项目配置的批准。在主检出中批准的项目 MCP 声明、项目子智能体定义、项目钩子文件、项目市场条目和项目调度器任务，在其 worktree 中也会被视为已批准。回执仍绑定到已批准内容的摘要，因此分支添加或更改的声明、文件或条目会等待其自己的批准。每个检出都会保留自己的回执，因此在 worktree 中批准分支的版本不会影响主检出对其版本的批准，而合并后内容相同的分支不会询问任何内容。在一个检出的文件中删除或重命名任务，或移除市场条目，只会移除该检出的回执，而在任一检出中显式撤销会为整个项目撤销。移除 worktree 不会撤回其中已批准的任何内容：其回执会保留在项目中，直到显式撤销；
- 你对项目 MCP 服务器的开关（`<home>/mcp-overrides.json`）：在主检出中关闭的服务器会在其 worktree 中关闭；
- 项目 MCP 服务器。声明未命名 `${CWD}` 的项目服务器会在每个检出中运行相同的命令，因此主检出及其 worktree 共享一个进程，将会话移入 worktree 不会重启它。声明通过 `${CWD}`（`node ${CWD}/server.js`）的服务器会获得其自己检出的路径，并在每个 worktree 中运行一次：这就是测试分支中正在开发的 MCP 服务器的方式。声明中的相对路径会从启动 Coddy 的文件夹读取，而不是从检出读取；
- 项目调度器任务。调度器扫描项目，而不是检出：项目任务会从主检出按 cron 运行一次，无论聊天在多少个 worktree 中打开了调度器。worktree 中的任务副本会列为未调度，并且可以从那里的聊天手动运行；分支添加的任务会在分支合并后按 cron 运行。

worktree 内的文件夹对应主检出的同一文件夹：`<worktree>/pkg` 到 `<main checkout>/pkg`。在 worktree 中，`coddy mcp list`、`coddy agents list` 和 `coddy hooks list` 会添加一行，命名批准所保留的主检出，而 `GET /coddy/mcp`、`GET /coddy/subagents` 和 `GET /coddy/hooks` 的 `workspace` 字段也会命名它。

只有当 git 双向链接某个文件夹时，该文件夹才算作 worktree：其 `.git` 文件命名仓库的 worktree 存储，而该存储也会反向命名该文件夹。复制或写入另一个文件夹的 `.git` 文件，例如作为链接的 `.git`，不会为该文件夹提供任何内容。如果某个文件夹在主检出中的对应项是指向其他位置的链接，或位于那里嵌套的仓库中，则该文件夹会保留自己的批准；裸仓库的 worktree 文件夹也是如此。将其存储保留在主检出之外的仓库（`--separate-git-dir`）的 worktree 与主检出不共享任何内容：当该存储命名为 `.git` 时，它们共享持有该存储的文件夹的批准，否则各自保留自己的批准。在 Coddy 共享批准之前，worktree 在其自身路径下记录的批准不会被迁移，因此该 worktree 会再次请求只有它批准过的内容。

<!-- docsgen:source sha256=3d520c66b75e2c76 -->

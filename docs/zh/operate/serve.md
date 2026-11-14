# 将 Coddy 作为守护进程运行

`coddy serve` 会运行配置所启用的每个子系统 - HTTP API 与内嵌 Web 界面、消息网关、swarm 中继、cron 调度器、浏览器扩展的端点 - 都在一个进程中，由同一个会话管理器管理。它们中哪些会启动由 `config.yaml` 决定；参见[配置参考](../reference/config.md)以及各界面指南（[HTTP API](../reference/http-api.md)、[网关](../surfaces/gateway.md)、[swarm](swarm.md)、[调度器](scheduler.md)、[Coddy for Browser](../features/browser.md)）。

还有一个子系统与它们并列提供服务：Prometheus 导出器，位于 `metrics.enable` 下（[监控](monitoring.md)）。它是辅助性的。它从不被视为让进程保持运行的界面，因此仅启用 `metrics` 的配置会像不启用任何内容的配置一样被拒绝，而关闭它始终可行。

这些子系统共享的不只是管理器。智能体启动的后台任务在结束时将其唤醒（`notify_on_finish`，默认开启），无论运行哪些子系统：进程拥有唤醒器，并把每个被唤醒的轮次交给绑定到该会话的 Telegram 或 Pachca 聊天，否则交给 HTTP 服务器，否则通过管理器本身运行（[后台任务](../features/background-tasks.md#在-coddy-serve-下)）。MCP 服务器以同样方式共享：全局配置中的服务器随进程启动，并为每个子系统的每个会话提供服务，包括子智能体和计划运行；项目的服务器按工作区运行一次，与其 git worktrees 共享（[MCP](../features/mcp.md#共享服务器)）。

本页讲的是如何让该进程保持运行并保持最新。

![coddy serve 如何运行：分发器、systemd 或终端让一个进程保持运行，其运行时与监督程序运行每个已启用的子系统，配置更改要么就地应用，要么以一个新进程来应对](../../assets/diagrams/serve-runtime.png)
*一个进程、一个会话管理器、每个子系统一个 goroutine：监督程序能就地应用的更改会在那里应用，而移动后的监听地址会让工作进程以状态 75 结束，以便新进程绑定它。*

## 在前台运行

```bash
coddy serve
```

进程占用终端，打印它启动了什么，并在 Ctrl-C 时停止。在 `systemd`、`supervisord`、Docker 或任何其他已经接管进程生命周期的环境下，应使用这种形式 - 那些监督程序会自行重启进程，在它们下面再叠一层只会让第一层看不到失败。在 Linux 上，`coddy serve install` 会为你设置该 systemd 单元，可能是打包的单元，也可能是它写入的单元（下一节）。

### Windows 通知区域

在 Windows 上，`coddy serve` 在 HTTP 界面启用时会把 Coddy 图标放进通知区域；`coddy serve --daemon` 背后的 worker 也一样。这个图标只是服务器的便捷入口，不是第二个进程：如果当前会话没有交互式桌面，服务器仍会继续运行。

它的菜单有三项：

- **打开页面**：在默认浏览器中打开 `http://127.0.0.1:<port>/`。绑定 `0.0.0.0` 这类通配地址时，这个链接会换成 loopback；显式命名的绑定地址保持不变。
- **打开项目**：打开 Windows 文件夹选择框，以所选文件夹为工作区创建一个新聊天，并打开该聊天。最近聊天的工作区列在同一项下面，因此这个 home 用过的文件夹只差一次点击。
- **退出**：通过与 Ctrl+C 相同的 supervisor 关闭流程停止进程。

菜单里还有 **注册右键菜单**，默认不勾选。勾选后会在 `HKCU\Software\Classes` 下给文件夹、
文件夹背景和驱动器添加 **用 Coddy 打开**，取消勾选会移除它们；不需要管理员权限。该入口
运行 `coddy open <文件夹>`，通过一个 loopback 控制端点联系正在运行的服务，端点 token 以
仅所有者可读的权限保存在 `${CODDY_HOME}/serve/open.json`。Windows 11 会把这种经典菜单项
放在 **显示更多选项** 里。

托盘只属于 `coddy serve`；`coddy desktop` 保留自己的窗口和托盘行为。

## 在 Linux 上作为 systemd 用户服务

```bash
coddy serve install                      # once, as the user the service is for
journalctl --user -u coddy.service -f    # its log
coddy serve uninstall                    # take it away again
```

在带有 systemd 的 Linux 上，`coddy serve` 可以作为 *用户* 服务运行：它属于一个账户，以该账户的权限及其 `~/.coddy` 运行，在崩溃后恢复，并且在启用 [lingering](#注销后与启动时) 后，可在注销后继续运行并在启动时启动。`coddy serve install` 会将其放置到位，`coddy serve uninstall` 会将其移除。两者都以该服务所属用户的身份运行，绝不在 `sudo` 下运行。

在 macOS 和 Windows 上没有 systemd，两个命令都会说明这一点；`coddy serve --daemon` 是那里的可行路径。在 Linux 上，`install` 需要做什么取决于 Coddy 的安装方式：

| 安装方式 | 单元 | `install` 会如何处理它 |
|---|---|---|
| 使用 `.deb` 或 `.rpm` | `/usr/lib/systemd/user/coddy.service`，已安装且**未启用** | 为你的账户启用它 |
| 安装脚本、发布归档、Linux 上的 Homebrew、本地构建 | 无 | 为运行 `install` 时使用的二进制写入 `~/.config/systemd/user/coddy.service`，然后启用它 |

软件包不会为任何人启用该单元：一台机器上哪些账户运行服务器由各自决定，安装时打印的消息会说明这一点。安装脚本不安装单元，并以同样的方式指向 `coddy serve install` 结束。

无论哪种方式，`install` 都会：

1. 检查 `~/.coddy/config.yaml`，方式与 [`--test-config`](../reference/cli.md) 相同，并在出错时停止；
2. 当同一主目录下有 `coddy serve --daemon` 运行时拒绝执行，因为两者会绑定同一端口；
3. 放置单元（如上），并放置一个 drop-in `~/.config/systemd/user/coddy.service.d/coddy-install.conf`，其中包含你运行它时所用 shell 的 `PATH`（没有 `PATH` 的 shell 会保留先前运行写入的 drop-in）；
4. 创建 `~/Coddy`；
5. 运行 `systemctl --user daemon-reload`、`enable` 和 `restart`，等待两秒，并报告服务是否保持运行，同时给出显示其日志的命令。

再次运行 `install` 是安全的，这也是服务接收更改的方式：在 `coddy update` 或软件包升级后、二进制移动后、你的 `PATH` 更改后。它会重写自己写入的内容，重启服务，并且绝不覆盖不是它写入的单元或 drop-in。

### 服务运行时使用什么

- **工作目录 `~/Coddy`。** `coddy serve` 的工作目录是每个未指定工作目录而打开的会话的工作区：在尚未选择文件夹时启动的 Web 界面聊天、调度器任务、Telegram 对话。服务会获得自己的文件夹，因此智能体写入的内容会落在这里，而不是落在 `~/.coddy` 中配置和提供商密钥旁边。该文件夹缺失时，单元会创建它。在项目文件夹上打开的会话照常在该文件夹中工作。
- **智能体主目录 `~/.coddy`。** 服务读取 `~/.coddy/config.yaml` 和 `~/.coddy/.env`。用户管理器不会继承你的 shell 环境，因此在 `.bashrc` 中导出的 `CODDY_HOME` 不会到达它；`install` 看到时会说明这一点。
- **你的 shell 的 `PATH`。** 用户管理器以裸系统 `PATH` 启动服务，而如果智能体找不到你的终端能找到的 `go`、`node` 或 `python`，就会在第一次构建时失败。该 drop-in 会把 `PATH` 的绝对条目交给服务，即运行 `install` 时使用的那些。
- **日志。** 输出进入 `journalctl --user -u coddy.service`，当 `logger.file` 指定时也会写入文件。
- **重启。** `Restart=on-failure` 会恢复崩溃的进程。移动监听地址的配置更改 - Web 界面端口，从其自己的设置屏幕更改 - 会以状态 75 结束进程，systemd 会在新地址上再次启动它：单元设置 `CODDY_SERVE_ROLE=service`，它告诉 `coddy serve` 会有什么来启动它（[自行重启](#自行重启)）。
- **停止。** systemd 给 `coddy serve` 40 秒（在 `SIGTERM` 之后）完成进行中的轮次，然后杀死它。

其他任何内容 - 另一个工作目录、更多环境变量、一个标志 - 都放进你自己的 drop-in，`install` 和 `uninstall` 不会碰它：

```bash
systemctl --user edit coddy.service
```

### 注销后与启动时

用户管理器在其用户有会话时运行，因此当你注销最后一个会话时服务会停止，并在下次登录时再次启动。要在没有会话时保持其运行并在启动时启动，管理员需为该账户启用一次 lingering：

```bash
sudo loginctl enable-linger <user>
```

`install` 会在 lingering 关闭时打印该行。Coddy 从不更改它。

### 控制服务

systemd 拥有该服务。`coddy serve status | stop | restart` 的动词属于 `--daemon`，当它们反而找到 systemd 服务时会说明这一点：

```bash
systemctl --user status coddy.service
systemctl --user restart coddy.service
systemctl --user stop coddy.service       # until the next login, or the next boot
```

不要在同一账户下于它旁边运行 `coddy serve --daemon`：两者会绑定同一端口。守护进程运行时 `install` 会拒绝，而通过 `~/.coddy` 执行 `--daemon` 时，若服务正在运行则会拒绝（对另一个主目录只会警告，因为那个服务可能监听别处）。守护进程动词还会提到一个已启用但已停止的服务，它会在下次登录时恢复。

`install` 和 `uninstall` 通过账户的总线与该账户的用户管理器通信。使用 `su` 或 `sudo -u` 打开的 shell 会保留调用者的会话，因此无法从中访问管理器，两个命令都会说明这一点；请改为以该用户身份登录（ssh、桌面会话、`machinectl shell <user>@`）。

### 移除服务

```bash
coddy serve uninstall
```

停止并禁用 `coddy.service`，移除单元和 `install` 写入的 drop-in，并重新加载用户管理器。打包的单元会留在软件包放置它的位置，处于禁用状态。你自己编写的单元或 drop-in 会被禁用并保留。`~/.coddy`（配置、会话）和 `~/Coddy`（工作区）不会被触碰。

在移除 `.deb` 或 `.rpm` 之前运行它。卸载软件包会移除单元文件，但无法进入每个账户的 `~/.config`，因此启用了该服务的账户会保持启用，服务会继续运行已删除的二进制，直到停止。来自另一个 Coddy 安装的 `coddy serve uninstall` 仍会清除它；如果没有这样的安装，卸载过程会打印应作为该用户运行什么命令：

```bash
systemctl --user disable --now coddy.service
rm -f ~/.config/systemd/user/coddy.service.d/coddy-install.conf
```

## 在后台运行

```bash
coddy serve --daemon      # or -d
```

该命令会脱离终端并返回。它留下的是一个**分发器**：一个在**工作进程**中运行各子系统，并在该工作进程因任何非被告知停止的原因消失时启动新工作进程的进程。从某个界面逃逸的 panic、内存不足被杀、随网络一同死亡的监听器 - 它们都以同样方式结束：用一个根据当前磁盘上的配置和二进制构建的新进程来替代。

```
coddy serve 1.0.19 is running in the background
  pid     4711
  config  /home/you/.coddy/config.yaml
  log     /home/you/.coddy/logs/serve.log
```

该命令在返回前会等待第一个工作进程，因此无法启动的配置 - 其他人占用的端口、此二进制未构建的子系统 - 会在输入命令的终端中报告，而不仅仅出现在尚无人跟踪的日志里。在这种情况下，分发器会继续重试；它仍在运行，并且会如此说明。

状态存放在智能体主目录下：

| 路径 | 它是什么 |
|------|------------|
| `~/.coddy/serve.json` | 分发器的记录：pid、版本、配置、日志、启动时使用的参数，以及它当前拥有的工作进程 |
| `~/.coddy/logs/serve.log` | 两部分写入的所有内容，跨重启追加 |

### 重启节奏

失败的工作进程会在 **1 秒** 后重启，然后 2、4、8 秒，最多 **30 秒**。一旦工作进程保持运行达 **一分钟**，等待时间会回到 1 秒，因此上周糟糕的一小时不会拖慢今天的恢复。

分发器从不放弃。宕机一天的依赖会每 30 秒重试一次，直到它恢复；需要人处理的问题在那个人查看时仍会是坏的，原因就在日志里。

### 控制它

```bash
coddy serve status
coddy serve stop
coddy serve restart
```

这三个命令都接受 `--home DIR`（或 `CODDY_HOME`）来指明它们针对哪个安装。

`status` 回答两个不同的问题，因为分发器在运行并不能说明是否有任何内容正在被提供：

```
coddy serve 1.0.19 is running
  pid     4711
  since   2026-09-10T11:26:03+03:00
  config  /home/you/.coddy/config.yaml
  log     /home/you/.coddy/logs/serve.log
  args    -H=0.0.0.0 -swarm=true
  worker  4713
```

`restart` 会使用记录中保存的参数恢复守护进程，因此以 `-H 0.0.0.0 --swarm` 启动的守护进程会作为该守护进程恢复，而不是作为默认守护进程。它们会以规范的 `-name=value` 形式记录，这就是为什么 `status` 会那样显示它们，而不是按输入时的样子显示。

`stop` 会先停止工作进程，并给它最多 30 秒完成正在做的事情，因此仍在生成的轮次不会在句子中途被切断。

## 先检查配置

```bash
coddy serve -t            # or --test-config; --home and --config pick the file as for a start
```

该标志会根据已发布的 JSON Schema 和加载器规则检查此命令将加载的文件，打印每个问题及其行号和修复方法，并在有错误时以状态 1 退出。不会启动任何内容，也不会写入任何内容，因此它适合放在部署脚本中，紧挨在 `coddy serve restart` 之前。报告格式以及什么算作警告在 [config.md](../getting-started/configuration.md#从命令行检查文件) 中描述。

```bash
coddy serve --dry-run     # the same check, then probe what the file points at
```

`--dry-run` 会运行该检查，然后，当文件干净时，探测它所描述的世界：配置和所输入标志启用的子系统（此二进制未构建的界面会被报告，而不是启动）、它们将绑定的监听地址（另一个进程占用的端口会连同设置它的行一起指明）、每个提供商的模型列表、将配置的模型与该列表对照、stdio MCP 服务器的可执行文件、用 Bot API 校验 Telegram 机器人令牌、用 Pachca 校验 Pachca 机器人令牌及其作用域、`swarm.join` 中的中继以及中继挂载的上游，以及配置指定的目录和文件。探测失败时退出状态为 1。单独使用时，它只打印问题和一行状态；`coddy serve --dry-run --test-config` 会打印配置检查报告和每个探测结果。报告及其规则在 [config.md](../getting-started/configuration.md#试运行探测文件指向的内容) 中描述。

## 接收配置更改

运行中的进程会监视它加载的文件。在其**外部**所做的更改会像从设置屏幕保存一样生效：

- 在另一个终端中运行 `coddy providers login neuraldeep` 会添加提供商及其模型，每个已打开浏览器中的模型选择器会在几秒内拥有它们；
- 操作员手动编辑 `config.yaml`；
- 部署过程放入一个新文件。

接下来会发生什么取决于移动了什么：

| 更改 | 效果 |
|--------|--------|
| 模型、提供商、技能、权限、大多数设置 | 实时配置被替换；`GET /coddy/events` 携带 `config_reloaded`，已打开的客户端会重新读取（参见 [SPA 说明](../surfaces/web-ui.md)） |
| Telegram 或 Pachca 令牌（每个机器人单独重建，另一个继续运行）、调度器的目录或超时、中继的 `swarm` 设置中除地址之外的部分、导出器的地址（`metrics.host` / `port`，其计数器保留） | 仅该子系统就地重建 |
| `metrics.token` | 不会重建任何内容：导出器在每次抓取时读取令牌 |
| 某个子系统的 `enable` | 它会被启动（使用新配置给它的地址），或被停止 |
| `mcp.project_trust` | 每个活动会话会按新策略重新连接其配置的 MCP 服务器；移到 `deny` 后，项目服务器会停止 |
| `mcp.idle_timeout_seconds` | 从此之后无人持有的服务器会等待新时间后才停止 |
| 监听地址（`httpserver.host` / `port`，`swarm.host` / `port`） | 在分发器下，进程会在新地址上重启；在前台运行时，会记录为需要重启；[设置 → 系统](#应用已保存的地址) 会显示其中任一情况 |

MCP 服务器不在 `config.yaml` 中：进程以同样方式监视 `~/.coddy/mcp.json`，对它的编辑会启动新增或启用的服务器，在不再有会话持有后停止被移除或关闭的服务器，并启动声明已更改的服务器（使用新声明），而其他服务器保留其进程（[MCP](../features/mcp.md#在-coddy-之外所做的编辑)）。

界面在构造时只读取一次的其他所有内容 - 例如节点的 HTTP 服务器所做的 `swarm.join` 注册 - 仍然需要你主动重启，`coddy serve restart` 或 Ctrl-C 后再次启动。只有地址会被自行采纳，因为它是操作员从该地址正在提供服务的屏幕中更改的那一项。

无法解析的文件，或在编辑器写入期间短暂消失的文件，不会影响运行中的配置，并会在日志中报告。注释和键顺序不是设置，因此只移动这些内容的编辑不会改变任何东西，也不会宣告任何东西。

标志优先于文件，重新加载时与启动时一样。以 `--gateway` 或 `-H 0.0.0.0` 启动的守护进程，在其他人保存无关设置时会保留它们。进程自己进行的保存 - 从设置屏幕、MCP 标签页，或智能体的 `config_commit` 和 `config_rollback` - 也会在应用标志后生效。无法应用标志的保存会被报告为错误，运行中的配置保持原样。

### 自行重启

监听地址是运行中的进程无法采纳的一项更改：监听器正是调用方正在通话所经过的东西，在他们脚下移动它会丢弃请求移动的那条请求。背后有分发器时，工作进程会退出并请求被替换，替换者会绑定新地址 - 这就是操作员如何移动他们正在输入其设置屏幕的那台服务器本身的端口。

该请求的退出状态是 **75**（`EX_TEMPFAIL`），因此对 Coddy 一无所知的监督程序会按原本意图理解它：本次运行结束，值得启动另一次。前台运行的 `coddy serve` 只有在通过其环境中的 `CODDY_SERVE_ROLE=service` 被告知会有什么再次启动它时，才会为此退出；否则它会记录重启已到期，并保留旧监听器。`coddy serve install` 使用的单元会设置该变量，以及 `SuccessExitStatus=75` 和 `RestartForceExitStatus=75`，因此重启不会被记录为失败。你自己的单元或监督程序也需要该变量。

仅作为 swarm 中继启动的进程不会打开会话存储，因此 HTTP 服务器、网关和调度器 - 运行智能体轮次的界面 - 无法在其中启动；将它们之一启用的重新加载会以同样方式请求新进程，或说明重启已到期。需要新进程的更改会在其中任何内容被应用之前先被权衡：它会经过新进程所运行的预检，而预检会拒绝的更改（例如此二进制未构建的界面）会被整体拒绝。不会停止、启动或重建任何内容，日志会说明原因，并且进程不会被重启到无人能启动的配置中。

### 应用已保存的地址

运行中的进程无法采纳的已保存更改会显示在 **设置 → 系统** 中。对于监听地址，**子系统** 行会列出正在使用的地址和已保存的地址，提示内容为 **保存的地址 {address} 将在 Coddy 重启后生效。** 同样的提示，措辞为 **保存的更改将在 Coddy 重启后生效。**，也涵盖在没有会话存储的进程中启用的运行轮次的界面（HTTP 服务器、网关或调度器），例如仅作为 swarm 中继。同一页面上的 **启动方式** 会说明适用以下哪种情况：

- **systemd 服务**：进程以状态 75 退出，systemd 会自行按已保存的设置启动它。要自行重启它，请运行 `systemctl --user restart coddy.service`。
- **后台守护进程**：分发器会自行替换工作进程。要自行重启它，请运行 `coddy serve restart`。另一个进程占用的端口会按上文的 [节奏](#重启节奏) 重试。
- **前台运行**：没有任何东西会再次启动进程，因此请停止它并再次运行 `coddy serve`。你自己的单元或监督程序需要在其环境中设置 `CODDY_SERVE_ROLE=service` 才能像第一种情况那样工作。

受监督的进程只会重启到新进程能够启动的设置。当新进程运行的检查会拒绝已保存的配置时，不会发生重启：旧监听器保持运行，页面会显示 **保存的设置无法启动：** 后接检查消息。更正设置后，一旦文件通过检查，进程会自行重启。保存前运行 `coddy serve --dry-run`，可查看拒绝原因，或另一个进程占用的端口。

## 使用哪种形式

| 情况 | 形式 |
|-----------|------|
| 带有 systemd 的 Linux，应比你的登录更持久的服务器 | `coddy serve install`（systemd 用户服务） |
| 笔记本电脑、开发机、没有 systemd 的机器上的 shell | `coddy serve --daemon` |
| `supervisord`、`runit` | `coddy serve` 在前台运行，并使用 `CODDY_SERVE_ROLE=service`，让它们重启它 |
| Docker、Kubernetes | `coddy serve` 在前台作为 PID 1；编排器重启容器 |

软件包会附带一个不会启用的用户单元，并且不附带系统单元：Coddy 的状态是按用户存放在 `~/.coddy` 下的，因此系统守护进程将需要一个主目录和一个没人能编辑的配置。参见[安装](../getting-started/install.md)。

## 捕获线上协议

当提供商返回意外结果时，Web UI 可以显示实际发送到线路上的内容，即请求和响应、头与正文，位于 **设置 → 系统** 页面。开关是 **`httpserver.raw_capture`**：

```yaml
httpserver:
  raw_capture: true
```

默认关闭，因为捕获的内容与机密相邻，并且会给服务器的每次提供商调用带来一点开销。开启后：

- 此服务器发出的每个提供商请求和响应都**仅保存在内存中**，即一个容纳最近 100 次交换的有界环形缓冲区，绝不写入磁盘，并在进程停止时丢弃；
- 每个正文随传输层读取而记录，仅保留前 256 KiB；携带凭据的头 - **Authorization**、**Proxy-Authorization**、**Cookie**、**Set-Cookie** 以及 `x-api-key` 这类厂商密钥头 - 会从两个方向剥离，URL 中携带凭据的查询参数也会被遮蔽，因此被捕获的凭据不会留在环形缓冲区中；
- **设置 → 系统** 页面上的 **原始捕获** 块会列出这些交换（方法、URL、状态，以及请求和响应头与正文），**清除** 按钮会清空环形缓冲区；关闭开关后，整个块不再显示。

捕获会跟随实时配置，因此将其打开的重新加载会安装记录器，而将其关闭的重新加载会停止传输层捕获，无需重启。停止 HTTP API 本身也会同样停止捕获并清空环形缓冲区。

<!-- docsgen:source sha256=0628ed324b21f361 -->

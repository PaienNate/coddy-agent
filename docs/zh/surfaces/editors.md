# 编辑器（ACP）

两段录屏展示了编辑器通过 ACP 驱动 Coddy。

https://github.com/user-attachments/assets/4fb167a1-ddd8-4a82-8f63-4ea0d156998f

*1.3.0 上的 Zed，两分钟：Coddy 发布的各个选择器、由模型读取的一张图片、在全新 worktree 中处理的 `/goal`、一条通过安全检查且无需提示即放行的 shell 命令，以及 `/commit`。该文件位于仓库中的 [zed-acp.mp4](../../assets/video/zed-acp.mp4)。*

https://github.com/user-attachments/assets/6335b103-d6fb-40c3-975f-9db24b89fa94

*1.3.0 上配合 ACP 客户端扩展的 VS Code，两分钟：各个选择器、accept-edits 模式下的命令安全检查、带编辑卡片和 shell 卡片的 `/goal`、`/commit`、从树和一个 worktree 恢复的会话。该文件位于仓库中的 [vscode-acp.mp4](../../assets/video/vscode-acp.mp4)。*

Coddy 是一个 ACP 服务器。`coddy acp` 通过自己的 stdin 和 stdout 使用 [Agent Client Protocol](https://agentclientprotocol.com/)，因此，实现了 ACP 客户端一侧的编辑器、插件或脚本只要启动该二进制，就能得到与控制台和 Web 界面相同的智能体：相同的模式和模型、相同的权限策略、相同的技能，以及 `$CODDY_HOME` 下相同的会话包。Coddy 中没有任何编辑器专属的内容 - Zed、VS Code 和 Python 框架都以相同方式连接。

## ACP 是什么

ACP 规范了客户端与智能体进程通信的方式：客户端启动智能体，向其 stdin 写入以换行分隔的 JSON-RPC 请求，并从其 stdout 读取响应和 `session/update` 通知；stderr 承载日志，绝不承载协议。Coddy 实现智能体一侧 - `initialize`、`session/new`、`session/load`、`session/resume`、`session/list`、`session/delete`、`session/prompt`、`session/cancel`、`session/set_mode` 和 `session/set_config_option` - 并回调客户端以获得 `session/request_permission`、`elicitation/create`、`fs/read_text_file` 和 `fs/write_text_file`，并以 `session/request_question` 作为未实现 elicitation 的客户端的回退方案。只要配置了会话存储（对 `coddy acp` 而言始终如此），`initialize` 就会声明 `loadSession`，以及一个空的 `authMethods` 列表：没有登录步骤。

协议契约 - 每个方法、通知与载荷 - 见 [ACP 协议参考](../reference/acp-protocol.md)。本页讲的是如何将客户端连接到它。

## 命令行

```bash
coddy acp [flags]
```

| 参数 | 含义 |
|---|---|
| `--home DIR` | 智能体状态目录（`CODDY_HOME`，默认 `~/.coddy`）：配置、会话、技能、信任凭据 |
| `--config PATH` | `config.yaml`（`CODDY_CONFIG`，否则 `<home>/config.yaml`） |
| `--cwd DIR` | 当客户端发送空的 `cwd` 时，会话的默认工作目录（`CODDY_CWD`，默认为进程的 cwd）；在 `session/new` 中传入路径的编辑器则使用该路径 |
| `--sessions-dir DIR` | 会话根目录（否则取配置中的 `sessions.dir`，再否则 `~/.coddy/sessions`） |
| `--session-id ID` | 下一个 `session/new` 在该包存在时重新打开它，否则创建一个新包，其文件夹名为 `ID` |
| `--mcp-project-trust ask\|allow\|deny` | 本进程中针对工作区 `.coddy/mcp.json` 的策略；覆盖 `mcp.project_trust` |
| `--remote TARGET`、`--remote-token TOKEN` | 将整个协议代理到远程 `coddy serve`（见[驱动远程服务器](#驱动远程服务器)） |
| `--log-level`、`--log-output`、`--log-file`、`--log-format` | 日志记录；默认输出为 stderr，stdout 必须留给协议使用 |
| `--scheduler` | 在本进程中运行 cron 调度器（需要 `scheduler` 构建标签） |
| `--skills-auto-discovery=false` | 为本进程关闭由模型驱动的 `load_skill` 工具 |
| `-t` / `--test-config`、`--dry-run` | 检查配置（并探测其指向的内容）后退出；见[配置](../getting-started/configuration.md) |

配置客户端时请使用二进制的**绝对路径**，而不要依赖 `PATH`：某些框架会通过 `cmd /c` 或 `sh -c` 启动智能体，而不带上用户的 `PATH`。一行安装脚本会把二进制放到 Linux 和 macOS 上的 `~/.local/bin/coddy`，以及 Windows 上的 `%LOCALAPPDATA%\Programs\coddy\coddy.exe`（见[安装](../getting-started/install.md#windows)）。每个额外参数，例如 `--home` 或 `--remote`，都放入客户端参数列表中 `acp` 之后。

## Zed

Zed 通过 `agent_servers` 设置连接外部智能体。打开 **Agent Settings**，进入 **External Agents**，点击 **Add Agent** 并选择 **Add Custom Agent**：Zed 会打开 `settings.json`，其中有一个待填写的 `agent_servers` 条目。为 Coddy 添加一个条目：

```json
{
  "agent_servers": {
    "Coddy": {
      "type": "custom",
      "command": "/home/you/.local/bin/coddy",
      "args": ["acp"]
    }
  }
}
```

条目上可选的 `env` 对象会为该进程设置变量 - 当编辑器要使用 `~/.coddy` 之外的主目录时设置 `CODDY_HOME`，或设置不在你登录环境中的提供商密钥。Zed 无需重启即可识别该改动；在智能体面板新线程菜单的 *External Agents* 下选择 **Coddy**。

Coddy 自身的模式（`agent` / `plan` / `ask`）、其配置的模型及其权限策略会出现在 Zed 的输入框中，其技能则以斜杠命令的形式出现。权限提示是 Zed 自己的批准对话框：请求中携带 Coddy 提供的选项，而 Zed 发送的答复会按协议规定的嵌套形式读取。

## VS Code

任何 ACP 客户端扩展都可以；Coddy 的 [VS Code 演示视频](../../assets/video/vscode-acp.mp4)中用的是 **ACP Client** 扩展（`formulahendry.acp-client`）。用与 Zed 相同的方式让扩展指向该二进制：命令为 `coddy` 的绝对路径，参数为 `acp` 加上任意标志。扩展面板能显示的内容就是协议所承载的内容 - `session/new` 返回的模式、模型和权限模式选择器，某一轮次的工具调用与文件改动，斜杠命令，以及 `session/list` 和 `session/load` 提供给它的会话。命令在哪里配置取决于扩展本身；Coddy 只能看到它被启动为的那个进程。

## Obsidian 及其他客户端

Coddy 的 README 将 Obsidian 列在客户端之中。仓库里并没有 Obsidian 专属代码：实现 ACP 客户端一侧的插件或应用与其他客户端一样连接 - 它启动 `coddy acp` 并通过管道讲 JSON-RPC。只要客户端能运行命令并逐行读取其 stdout，剩下的就是协议参考。

## 脚本

`examples/acp/acp_e2e_todo.py` 是参考客户端：一个以换行分隔的 JSON-RPC 框架，它会驱动真实的轮次并检查待办列表、工具调用以及智能体写入的文件。请把它当作你自己客户端的模板，而不是把 `echo` 行拼接到管道里。手写客户端必须具备的四种行为，摘自参考文档的 [Stdio 客户端](../reference/acp-protocol.md#stdio-客户端coddy-特定) 章节：

- 当处理器不返回任何内容时，成功响应可能省略 `result`（`session/set_mode` 就是如此）：把带有你的请求 `id` 且没有 `method` 的任何一行视为完成；
- 发出 `session/prompt` 之后，在最终响应之前会到达许多 `session/update` 通知；逐行读取，直到你的 `id` 返回，并在其间用携带相同 `id` 的响应来答复 `session/request_permission` 和 `elicitation/create`（对于未实现 elicitation 的客户端，则是 `session/request_question`）；
- 当 stdout 不是终端时，它可能是块缓冲的：像该框架那样，用 `stdbuf -oL -eL` 或等效方式包装该二进制；
- 未完成的请求是异步分派的；如果你的客户端假定顺序，请在消费完上一个响应之前不要发送下一个 RPC。

整个 ACP 测试套件用 `./examples/build_coddy.sh && ./examples/test_acp.sh` 针对 `examples/config.demo.yaml` 运行；`CODDY_BIN`、`CODDY_CONFIG`、`SESSION_ROOT` 和 `SESSION_ID` 会覆盖每个脚本所用的内容（见 `examples/README.md`）。一行式的健全性检查完全不需要脚本：

```bash
echo '{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{}}}' | coddy acp
```

## 编辑器显示的内容

编辑器关于 Coddy 渲染的一切都来自协议，因此每个客户端得到的都是同一套内容：

| 客户端中的内容 | 来源 |
|---|---|
| 模式选择器（`agent`、`plan`、`ask`） | `session/new` 和 `session/load` 的 `mode` 配置项（外加旧版的 `modes` 字段）；通过 `session/set_config_option` 或 `session/set_mode` 更改 |
| 模型选择器 | `model` 配置项，`config.yaml` 中的每个 `models[].model` 一行，默认为 `agent.model`；列表为空时不出现 |
| 推理或思考控制 | `thought_level` 类别中的 `reasoning` 配置项，带有会话模型所提供的级别（仅当其逻辑模型设置了 `allow_reasoning_off: true` 时，`off` 才排在最后）；没有级别的模型不出现该项 |
| 权限策略（`ask`、`accept_edits`、`bypass`） | `_permission_mode` 类别中的 `permission_mode` 配置项，未做特殊处理的客户端会把它显示为普通选择器；该模式属于会话自身并持久化在其 `session.json` 中，因此智能体重启后仍然保留；新会话以任意界面（包括此选项）上最后选择的模式启动（在做出任何选择之前为 `ask`） |
| 斜杠命令 | `session/new` 和 `session/load` 之后的 `available_commands_update`：设置命令（`model`、`reasoning`、`think`、`nothink`、`agent`、`plan`、`ask`、`permissions`，其参数提示在 `input.hint` 中），然后是 `compact`、`export` 和 `plugin`，再然后是来自 `skills.dirs` 的每个技能。设置命令从提示词文本中执行，`--once` 或 `--count=N` 仅对后续轮次生效（[会话设置](../features/session-settings.md)） |
| 权限提示 | `session/request_permission`，选项为 Allow、Allow always 和 Reject（对于普通 shell 命令还有程序级的 `allow_always_<program>`，在 `ask` 下会话会切换到 `allow_session_bypass`，对于文件写入则切换到 `allow_session_accept_edits`）；以 `ask`（命令与写入）或 `accept_edits`（命令）发送；起决定作用的是会话自身的模式。在 `bypass` 下，只有两种提示会到达客户端：一种是由 `PreToolUse` 钩子通过 `ask` 强制触发的，另一种是针对 [命令安全检查](../features/decisions.md) 未能放行的 shell 命令（Allow、针对该确切命令的 Allow always、Reject）；检查判定为安全的命令会无需提示直接运行 |
| 会话的名称 | 每当名称变动时发送 `session_info_update`：由其首条消息推导出的名称、智能体固定下来的名称、其他界面所做的重命名。`title` 始终在协议中出现 - 重命名会携带新名称，清除名称则携带 `null` |
| 删除、恢复以及额外的工作区根目录 | `session/delete` 删除一个会话及其下嵌套的所有内容（正在运行的轮次会先被停止）；`session/resume` 在不重放对话记录的情况下恢复会话，适用于自己保存一份历史副本的客户端；`session/new`、`session/load` 和 `session/resume` 接受 `additionalDirectories`，即 `cwd` 之外的额外工作区根目录，`session/list` 则按行回报该列表 |
| 提问 | 当客户端声明了 `clientCapabilities.elicitation.form` 时，来自 `question` 工具的 `elicitation/create`，渲染为每个问题一个字段的表单；必须用其某个选项来回答的问题是选择器，允许用户自行输入的问题则是自由文本，选项作为建议。客户端以字段值作答。未实现 elicitation 的客户端会得到 `session/request_question`，其中每个问题都带有 `custom`、`true`（除非模型将其关闭），答案则是每个问题一个选项数组 - 与任何选项标签都不匹配的条目即为用户输入的文本 |
| 工具调用、计划检查清单、token 与账户用量、轮次的时钟与生成的 token | `tool_call` / `tool_call_update`、`plan`、`token_usage`、`provider_usage`、`turn_progress` 更新 |
| 无人输入的轮次：智能体启动的后台任务结束并将其唤醒（`notify_on_finish`，默认开启） | `background_wake` 更新，随后是同样的唤醒，以引用形式的智能体文本行（`> Woken by a finished background task: bg_3 make test, failed, exit 2, 1m 30s`）出现在答案开头，位于任何 `session/prompt` 之外；其中的受限工具会以 `session/request_permission` 询问（[后台任务](../features/background-tasks.md#任务完成时唤醒智能体)） |
| 会话列表与恢复 | `session/list`（可选的 `cwd` 过滤，无论文件夹路径如何拼写都能匹配；每行包含 `sessionId`、`cwd`、`title`、`updatedAt`）以及 `session/load`，后者会重放对话记录，然后重新发送命令目录 |

当智能体委派给子智能体时，子智能体自身的更新永远不会到达编辑器：它们会写入后台任务的日志。客户端代表子智能体收到的唯一内容就是权限提示，它以父级的 `sessionId` 投递，标题前缀为 `[subagent <name>]`。编辑器只能在提示轮次期间被询问，因此对于在轮次返回之后才发问的后台子智能体，提示不会被发送给它：在 `coddy acp` 下，子智能体会被拒绝，并附上它回报的原因；而在 `--remote` 下，提示会改为在服务器上等待（其 Web 界面、连接到它的控制台、该会话的 Telegram 聊天）。客户端自己在 `session/new` 中声明的 MCP 服务器会按声明连接；来自工作区 `.coddy/mcp.json` 的服务器则先经过信任门禁（见[安全与信任](../operate/security.md)）。

## 与控制台和 Web 界面共享的会话

`coddy acp` 将每个会话作为一个包存储在 `$CODDY_HOME/sessions/<sessionId>/` 下 - `session.json`、`messages.json`、`assets/`、`todos/active.md`、`plans/` - 使用的存储与控制台和 `coddy serve` 打开的是同一个。在相同的主目录下，在 Zed 中启动的线程就是 `coddy sessions list` 中的一行、Web 界面历史中的一行以及控制台 `/resume` 选择器中的一行，并且可以从其中任意一处继续；编辑器通过 `session/list` 看到其他界面为它所打开的文件夹创建的会话。配置文件、技能、规则、MCP 服务器、钩子和子智能体定义也以相同方式共享，因为它们都是从该主目录和会话的工作目录中读取的。

唯一的要求是各进程对主目录达成一致：从启动器启动的编辑器可能不会继承你 shell 导出的 `CODDY_HOME`，因此要么在客户端的 `env` 块中设置它，要么在参数中传入 `--home`。`--sessions-dir` 和 `sessions.dir` 会以相同方式移动所有界面的存储。`coddy acp --session-id <id>` 从命令行重新打开一个包；`session/load` 则从客户端做同样的事。

## 驱动远程服务器

```bash
coddy acp --remote nas02 --remote-token "$CODDY_REMOTE_TOKEN"
```

使用 `--remote` 时，该进程不保留任何存储，不运行调度器，也不运行智能体循环：每个会话调用都被代理到远程 `coddy serve`，其流式答案会被转换回 ACP 更新，权限提示和提问则通过服务器的 REST 路由来回答。目标可以是一个已配置的远程名称、一个 `host:port` 或一个 `http(s)://` URL；token 依次来自 `--remote-token`、匹配的 `httpserver.remotes` 条目的 `token` 或 `CODDY_REMOTE_TOKEN`。会话只持久化在服务器上，`--session-id` 仍然指定要在那里重新打开的包，模型选择器列出服务器的目录。远程服务器的编辑器条目就是同样的条目，只需把这两个标志追加到 `args`。细节、token 与 CORS：[远程模式](../operate/remote.md)。

## 故障排查

- **编辑器无法启动智能体。** 在客户端配置中使用二进制的绝对路径；`PATH` 通常不会被继承。在 Windows 上，安装脚本的路径是 `%LOCALAPPDATA%\Programs\coddy\coddy.exe`。
- **智能体启动了却没有任何返回。** 运行 `coddy acp -t` 检查配置，再运行 `coddy acp --dry-run` 探测提供商、MCP 命令、路径，以及（配合 `--remote`）服务器。两者都会打印每个问题及其所在行，并在不启动任何东西的情况下退出。
- **需要查看进程在做什么。** 在参数中加入 `--log-level debug`；日志默认写到 stderr，`--log-output file --log-file <path>` 则改为写入文件。绝不要把日志写到 stdout，客户端会把它当作协议来读取。
- **项目 MCP 服务器无法启动。** 它在等待批准：用 `coddy mcp list` 查看它，用 `coddy mcp trust <name>` 批准它，或者对你信任的代码检出用 `--mcp-project-trust allow` 启动进程。
- **编辑器没有为该文件夹列出任何会话，尽管控制台里有一些。** 编辑器会针对它所打开的文件夹请求 `session/list`（VS Code 的 ACP Client 发送第一个工作区文件夹），因此只有工作目录为该文件夹的会话才会出现：在子文件夹或另一个代码检出中启动的控制台不在其中，而不带 `--cwd` 的 `coddy sessions list` 会显示全部内容。无论文件夹路径如何拼写都能匹配 - 以逻辑路径存储的符号链接代码检出也能通过物理路径找到，且 Windows 和 macOS 忽略大小写；更早的版本比较这两个字符串，因而漏掉了控制台以同一文件夹的另一种拼写存储的会话。该列表在展开智能体节点时获取，并通过 **ACP: Refresh Sessions** 刷新。
- **编辑器与 shell 看到的会话不同。** 这两个进程使用了不同的主目录；在客户端的 `env` 块中设置 `CODDY_HOME`，或传入 `--home`。
- **客户端的权限答复被忽略。** Coddy 既读取编辑器发送的嵌套结果，也读取扁平形式；`cancelled` 结果或 `reject` 选项会拒绝该调用，其他任何内容都会允许。请检查客户端返回的是哪一种 `optionId`。

更一般的故障 - `PATH` 中缺少二进制、提供商拒绝密钥、端口被占用 - 见[故障排查](../getting-started/troubleshooting.md)。

<!-- docsgen:source sha256=c3c8cb937ebd482e -->

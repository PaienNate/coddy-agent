# 钩子

钩子是你自己的命令，Coddy 会在会话的生命周期节点运行它：工具调用之前、之后，以及（见事件表）一轮中的其他节点。该命令从 stdin 读取一个 JSON 文档，做它想做的事，并用退出码和 stdout 上可选的 JSON 作答。因此它可以阻止一次调用、越过权限提示直接批准、重写参数、给模型额外上下文，或仅仅记录发生了什么。

钩子以你的权限运行，在任何权限提示之前，对每一次匹配的调用运行。这正是该功能的意义，也是随检出一起到达的文件会被暂留到你批准为止的原因（见 [项目文件与信任](#项目文件与信任)）。请审查你启用的每一个钩子。

定义使用 Claude Code 的文件形态，事件名、负载字段和应答字段也遵循它，因此为 Claude Code（或使用相同形态的 Codex）编写的钩子在 Coddy 中无需改动即可用于所覆盖的事件。包含六个智能体对比的设计记录见 `docs/plans/hooks.md`。

![会话与一轮时间线上的钩子事件、哪些文件会运行、一个钩子进程及其退出码，以及同一事件上的多个钩子如何合并](../../assets/diagrams/hooks-lifecycle.png)
*每个事件在何处触发及其钩子能决定什么、Coddy 从哪些文件读取钩子、一个钩子进程如何作答，以及多个钩子的应答如何合并。*

## 定义文件

Coddy 读取 `hooks.files` 中列出的文件，优先级从低到高。默认列表：

| 文件 | 范围 | 说明 |
|---|---|---|
| `~/.coddy/hooks.json` | 用户 | 你自己的文件（`${CODDY_HOME}/hooks.json`） |
| `<workspace>/.claude/settings.json` | 项目 | Claude Code 兼容：只读取其 `hooks` 键 |
| `<workspace>/.claude/settings.local.json` | 项目 | 同上 |
| `<workspace>/.coddy/hooks.json` | 项目 | 工作区自己的文件 |

`${CODDY_HOME}`、`${CWD}` 和开头的 `~` 会展开；相对条目按会话 cwd 解析。不存在的文件会被跳过。来自每个文件的每个匹配钩子都会运行；优先级只决定目录顺序和运行顺序（先用户文件，然后按列表顺序的项目文件，最后是文件内的定义顺序）。

范围按规范路径（绝对路径，解析符号链接）决定：位于会话 cwd 或其下的文件属于**项目范围**，遵循 `hooks.project_trust`；其他所有文件属于**用户范围**，始终运行。

### 文件形态

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "run_command",
        "hooks": [
          { "type": "command", "command": ".coddy/hooks/guard.sh", "timeout": 30 }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "edit|write|apply_patch",
        "hooks": [
          { "type": "command", "command": "gofmt -l ." }
        ]
      }
    ]
  }
}
```

三个层级：**事件**（`PreToolUse`）、**匹配器组**（适用于哪些工具）和运行的**处理器**。没有 `matcher` 的组，或 `"*"` 的组，适用于该事件的每一次出现。

处理器字段：

| 字段 | 含义 |
|---|---|
| `type` | `command` 是唯一会运行的类型。Claude Code 的 `http`、`prompt`、`agent` 和 Codex 的 `mcp_tool` 会被解析，但会跳过并给出警告，在目录中显示为不支持。 |
| `command` | 命令行。没有 `args` 时，它通过宿主 shell 运行（bash 或 sh；Windows 上为 pwsh、PowerShell 或 cmd，与 `run_command` 使用的检测相同）。 |
| `args` | 可选。当该键存在时，命令直接用这些参数启动，不经过 shell（Claude Code 的 exec 形式）。 |
| `commandWindows` | 可选，当 Coddy 在 Windows 上运行时替代 `command`（Codex 的键；也接受 `command_windows`）。 |
| `timeout` | 秒。默认 `hooks.default_timeout_seconds`（60）。超时的钩子会连同其整个进程组被终止。 |
| `async` | `true` 会分离运行钩子：其输出被忽略，且它永远不能阻止或决定（因此异步处理器上的 `failClosed` 会被丢弃并给出警告）。整个进程中最多同时运行 8 个分离钩子；其余等待槽位。 |
| `failClosed` | `true` 会把崩溃、超时、被该轮取消所中断，或无效输出变成阻止，而不是非阻塞错误（也接受 `fail_closed`）。用于策略钩子。 |

`statusMessage`、`shell`、`once` 和 `additionalContextLimit` 会被接受并忽略。Claude Code 的 `if` 过滤器会被忽略并记录警告：钩子会对每一次匹配调用运行，因此依赖 `if` 的策略必须自行检查参数。

### 匹配器

Claude Code 的规则适用：

- 空或 `*` 匹配所有内容；
- 仅由字母、数字、`_`、`-`、空格、`,` 和 `|` 组成的值是精确名称或精确名称列表：`run_command`、`edit|write`、`Edit, Write`；
- 其他任何内容都是非锚定正则表达式（Go 语法）：`^run_.*`、`mcp__filesystem__.*`。无效表达式永不匹配。

工具事件会把匹配器与工具名比较。Coddy 自己的名称是模型看到的名称（`run_command`、`read`、`write`、`edit`、`apply_patch`、`glob`、`grep`、`webfetch`、`websearch`、`http_request`、`spawn_agent`、`question`，……）。MCP 工具是 `server__tool`，也匹配 `mcp__server__tool` 拼写。Claude Code 和 Codex 使用的名称会作为别名接受，因此 `Bash` 匹配 `run_command`，`Edit` 和 `Write` 匹配 `edit`，`write` 和 `apply_patch`，`Read` 匹配 `read`，`Task` 和 `Agent` 匹配 `spawn_agent`，`WebFetch` 和 `WebSearch` 匹配 `webfetch` 和 `websearch`。参数名不翻译：`tool_input` 携带 Coddy 的字段（`command` 对应 `run_command`；`path` 和 `content` 对应 `write`；文件系统工具对应 `path`、`pattern` 等），因此从 Claude Code 移植的脚本若读取 `tool_input.file_path`，必须读取 `tool_input.path`。

## 事件

| 事件 | 触发时机 | 匹配对象 | 能否阻止 |
|---|---|---|---|
| `SessionStart` | `session/new`（`startup`）、`session/load` 或重新打开（`resume`），以及工作区切换 - `POST /coddy/sessions/{id}/workspace`、选择文件夹、跳转 worktree 或就地切换分支 - （`workspace`），同步地，在会话返回之前 | 来源 | 否；仅上下文 |
| `UserPromptSubmit` | 当用户提交提示词时，在内置的 `/compact`、`/plugin`、`/export` 和 `/worktree` 命令被识别之后，且在提示词成为消息之前 | 无 | 是：提示词被拒绝 |
| `PreToolUse` | 在工具调用运行之前，在模式与子智能体检查之后、权限提示之前，无论权限模式如何；当通过 HTTP 持久化的权限被恢复时，它会针对提示所展示的参数再次运行（拒绝仍然拒绝；再次更改这些参数的钩子会取消调用，因为应答覆盖的是用户看到的内容）。提示所展示的参数会先被持久化，写入失败会取消调用而不是提示；恢复时若持久化参数缺失或不可读，则失败并保留待处理的提示以便重试，而拒绝不需要 bundle 中的任何内容 | 工具名 | 是：拒绝，或强制或跳过提示 |
| `PostToolUse` | 工具无错误返回之后 | 工具名 | 否；仅反馈和上下文 |
| `PostToolUseFailure` | 工具返回错误之后（权限拒绝或钩子拒绝之后不会） | 工具名 | 否；仅上下文 |
| `Stop` | 当 ReAct 循环即将以 `end_turn` 结束该轮时（取消、错误或轮次上限时不会） | 无 | 是：智能体被送回继续工作 |
| `PreCompact` | 在 `/compact`（`manual`）或自动压缩（`auto`）之前 | 触发器 | 是：压缩被否决 |
| `PostCompact` | 压缩之后 | 触发器 | 否 |
| `SubagentStart` | 在父会话中，当 `spawn_agent` 即将启动子智能体时，在定义及其信任已解析之后 | 子智能体名 | 是：生成被拒绝 |
| `SubagentStop` | 在父会话中，当子智能体的轮次结束、其报告到达父会话之前 | 子智能体名 | 否 |
| `Notification` | 当权限提示即将发送到客户端时 | 通知类型（`permission_prompt`） | 否 |

每个事件也会在子会话内为该子会话自己的轮次触发（工具调用、提示词、停止、压缩），负载中的 `subagent` 块会指明子会话；`SubagentStart` 和 `SubagentStop` 在父会话中触发。

## 钩子收到什么

stdin 上有一个 JSON 对象。会话字段在前，事件字段在后：

```json
{
  "session_id": "sess_5f1c0d2e9a7b4c3d8e6f1a2b",
  "hook_event_name": "PreToolUse",
  "cwd": "/home/op/project",
  "transcript_path": "/home/op/.coddy/sessions/sess_5f1c0d2e9a7b4c3d8e6f1a2b/messages.json",
  "permission_mode": "ask",
  "mode": "agent",
  "model": "openai/gpt-5",
  "turn": 3,
  "tool_name": "run_command",
  "tool_input": { "command": "rm -rf build" },
  "tool_use_id": "call_01"
}
```

| 字段 | 含义 |
|---|---|
| `session_id` | 调用所属的会话 |
| `hook_event_name` | 事件 |
| `cwd` | 会话工作目录（也是钩子的工作目录） |
| `transcript_path` | 会话的 `messages.json`；持久化关闭时为空 |
| `permission_mode` | `ask`、`accept_edits` 或 `bypass` |
| `mode` | `agent`、`plan` 或 `ask` |
| `model` | 生效的模型 id |
| `turn` | 用户轮次索引 |
| `subagent` | 在子会话内存在：`{"name", "parent_session_id", "depth"}` |
| `tool_name`、`tool_input`、`tool_use_id` | 工具事件：工具、其参数对象、调用 id |
| `tool_response` | `PostToolUse`：模型接收的文本 |
| `error` | `PostToolUseFailure`：错误文本 |
| `duration_ms` | `PostToolUse` 和 `PostToolUseFailure`：工具耗时 |
| `source` | `SessionStart`：`startup`、`resume` 或 `workspace`（工作区切换；必须只在真实会话启动时运行的钩子只匹配 `startup`/`resume` - 见下方 SessionStart 说明） |
| `prompt` | `UserPromptSubmit`：提示词文本 |
| `stop_hook_active`、`last_assistant_message` | `Stop`：Stop 钩子在此轮是否已把智能体送回工作，以及助手的最终文本 |
| `trigger`、`custom_instructions` | `PreCompact`：`manual` 或 `auto`，以及 `/compact` 之后的文本 |
| `trigger`、`summary` | `PostCompact`：触发器和生成的摘要（截断至 4,000 个字符） |
| `agent_name`、`agent_session_id`、`prompt`、`background` | `SubagentStart`：定义名、子会话 id、任务提示词以及该运行是否分离 |
| `agent_name`、`agent_session_id`、`task_id`、`status`、`report`、`turns` | `SubagentStop`：子智能体的结果（`end_turn`、`cancelled`、`failed`，……）、其最终报告（截断至 4,000 个字符）及其助手轮数 |
| `notification_type`、`message`、`tool_name`、`tool_input`、`tool_use_id` | `Notification`：`permission_prompt`，包含提示词正文和等待应答的调用 |

该进程还会在其环境中获得 `CODDY_PROJECT_DIR` 和 `CLAUDE_PROJECT_DIR`（会话 cwd）、`CODDY_SESSION_ID`、`CODDY_HOOK_EVENT` 和 `CODDY_HOME`，这些是在 Coddy 自身环境之上额外提供的。

## 钩子如何作答

| 退出码 | 效果 |
|---|---|
| `0`，stdout 为空 | 无决定：适用普通流程。沉默永远不会批准一个原本会询问的调用。 |
| `0`，stdout 是 JSON 对象 | 应用下面的字段 |
| `0`，其他 stdout | 纯文本；在工具事件上被忽略（在提示词和会话事件上会成为上下文） |
| `2` | 阻止：调用被拒绝，原因依次使用 JSON `reason`（若有）、stderr、通用原因 |
| 其他任何情况、崩溃、超时、被该轮取消所中断、无效 JSON | 非阻塞错误：记录日志，并在每个会话中记录一次为通知行；事件会像钩子不存在一样继续，除非处理器有 `failClosed: true`，这会将其变成阻止，并以该错误为原因。钩子被取消所中断的调用不会运行。 |

JSON 字段：

```json
{
  "continue": true,
  "stopReason": "shown to the user when continue is false",
  "systemMessage": "shown to the user as a notice row in the transcript, never to the model",
  "decision": "block",
  "reason": "why",
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "why",
    "updatedInput": { "command": "echo replaced" },
    "additionalContext": "a note the model reads next to the result"
  }
}
```

- `continue: false` 会在当前工具批次之后结束该轮；`stopReason` 会展示给用户。在 `PreToolUse` 上，调用也会被拒绝；在 `UserPromptSubmit` 上，提示词被拒绝；在 `PreCompact` 上，压缩被否决；在 `Stop` 上，该轮直接结束。
- `PreToolUse`：`permissionDecision` 为 `deny`（调用不执行；原因作为工具结果 `blocked by hook: <reason>` 发给模型）、`allow`（跳过权限提示）或 `ask`（强制显示提示，即使在 `bypass` 中、在即时通讯机器人中和在计划运行中，这些情况原本会不经询问就批准调用；无人可作答时，调用被拒绝）。`updatedInput` 会在调用运行前替换整个参数对象；工具调用卡片显示重写后的参数。带 `reason` 的顶层 `decision: "block"`（或 `"deny"`）是 deny 的旧式拼写。
- `PostToolUse`：`decision: "block"` 配合 `reason` 会把 `Hook feedback: <reason>` 追加到工具结果；工具已经运行，不会撤销任何东西。
- `additionalContext` 在三个工具事件上会作为 `Hook context: ...` 追加到工具结果。
- `UserPromptSubmit`：`decision: "block"` 配合 `reason`（或退出码 2）拒绝提示词：它不会被加入对话记录，模型不会被调用，该轮以 `prompt rejected by hook: <reason>` 结束，SPA 会将其显示为错误行。`additionalContext` 和纯 stdout 会成为**钩子上下文块**（见下）中该轮的部分。
- `Stop`：`decision: "block"` 配合 `reason`（或退出码 2）把智能体送回工作。原因加上任何 `additionalContext` 会作为下一条用户消息提交，并以 `[Stop hook] ` 为前缀持久化，以便对话记录解释续接；循环在同一轮中继续；钩子在下一次 stop 时看到 `stop_hook_active: true`。每轮最多 `hooks.stop_loop_limit` 次续接（5），之后该轮结束；ReAct 轮次上限仍然适用。
- `SessionStart`：`additionalContext` 和纯 stdout 会存储在会话上（`session.json` 中的 `hookContext`），并在该会话每个系统提示词的钩子上下文块中呈现；恢复会重新运行钩子并替换存储的文本，当不再有钩子运行时清除它。工作区切换会以来源 `workspace` 重新运行新工作区的钩子，原因相同：存储的上下文不再描述会话离开的文件夹。`systemMessage` 会成为通知行。
- `Stop`：只有在 ReAct 轮次还有剩余迭代时才提交后续消息；在最后一次迭代时，该轮会结束，而不是留下悬空消息。
- `PreCompact`：`decision: "block"` 配合 `reason`（或退出码 2）否决压缩：`/compact` 以 `compaction blocked by hook: <reason>` 失败，自动压缩会为该检查跳过，该轮在未压缩的情况下继续。
- `SubagentStart`：`decision: "block"` 配合 `reason`（或退出码 2）拒绝生成；`spawn_agent` 工具结果读作 `spawn of subagent "<name>" blocked by hook: <reason>`，且不会创建子会话。`additionalContext` 会作为 `Hook context: ...` 前置到子智能体的任务提示词。
- `SubagentStop` 和 `Notification` 是观察性的：会报告 `systemMessage` 和错误，决定会被忽略。向聊天发送 ping 或弹出桌面通知的钩子应带有 `"async": true`，以免权限提示被它延迟。

**钩子上下文块**是系统提示词中追加在环境块之后的 `## Hook context` 章节（因此自定义 `prompts.dir` 模板也会携带它）：先是来自 `SessionStart` 的会话级文本，然后是来自 `UserPromptSubmit` 的轮次级文本。轮次级部分不持久化；会话级部分会持久化。

多个匹配钩子按目录顺序依次运行；每个都看到前一个重写后的输入，而且即使出现拒绝，它们也都会运行，因此审计钩子能看到每一次调用。决定按最严格者胜出合并（`deny` > `ask` > `allow`）；每个 `additionalContext` 都按顺序保留。钩子交出的文本上限为 `hooks.max_output_chars` 个字符（10,000），超出会截断并带标记。stdout 和 stderr 各自最多捕获 256 KiB；超过该限制的 JSON 应答会报告为被截断，而不是无效。

## 配置

```yaml
hooks:
  enable: true
  files:
    - "${CODDY_HOME}/hooks.json"
    - "${CWD}/.claude/settings.json"
    - "${CWD}/.claude/settings.local.json"
    - "${CWD}/.coddy/hooks.json"
  project_trust: ask          # ask | allow | deny
  default_timeout_seconds: 60
  stop_loop_limit: 5
  max_output_chars: 10000
```

| 键 | 默认值 | 含义 |
|---|---|---|
| `enable` | `true` | 是否加载并运行钩子 |
| `files` | 上面的四个 | 定义文件，优先级从低到高 |
| `project_trust` | `ask` | 项目范围文件可以做什么：`ask`（列出，暂留到批准）、`allow`（像你自己的文件一样运行）、`deny`（从不读取） |
| `default_timeout_seconds` | `60` | 当定义未给出 `timeout` 时，按处理器设置 |
| `stop_loop_limit` | `5` | 每轮中 `Stop` 钩子最多可把智能体送回工作多少次 |
| `max_output_chars` | `10000` | 钩子交给模型或用户的每段文本的字符上限 |

这些键是普通的 `config.yaml` 键，因此设置页面和内置的 `configure-coddy` 技能可以更改它们。定义会在每一轮开始时重新读取：编辑文件会在下一轮生效，无需重启。

## 项目文件与信任

工作区内的钩子文件随检出一起到达。在默认策略 `ask` 下，它会被解析并列出，但在运行 Coddy 的机器上，你为该工作区批准该确切文件之前，它的任何钩子都不会运行。回执绑定到文件的摘要，因此编辑已批准的文件会撤回批准并再次询问。当命令指名工作区的文件时 - 直接运行的脚本（`.coddy/hooks/check.sh`、`"$CLAUDE_PROJECT_DIR"/.claude/hooks/check.sh`、`"$PWD/check.sh"`）或交给解释器（`python3 scripts/check.py`） - 摘要也覆盖这些文件的内容，因此编辑这样的脚本，或添加一个命令指名但检出中原本没有的脚本，也会再次询问；在一轮中被重写的脚本在该轮剩余时间内不会运行。Coddy 会在每个处理器启动前检查摘要，包括在异步处理器获得执行槽位之后。摘要按命令的词进行，因此请像阅读钩子文件一样阅读它看不到的内容：脚本自身加载的内容，以及运行时构建的路径。字面量 `cd` 会被跟随。命令通过重定向写入的文件不计入；仅作为参数指名的文件会计入，因此每当日志变化时 `tee hook.log` 都会再次询问。你自己的文件（用户范围）永远不需要批准。

第一个发现被暂留文件的轮次会在会话中记录一条通知（SPA 会在该轮内将其显示为系统行，位置在被暂留文件被发现处，每个会话和每个文件一次，没有重试控件；智能体日志也带有同一行），这样你就会知道钩子存在但被暂留：

| 深色 | 浅色 |
|---|---|
| ![被暂留的钩子文件通知，深色](../../assets/screenshot-hooks-notice-dark.png) | ![被暂留的钩子文件通知，浅色](../../assets/screenshot-hooks-notice-light.png) |

批准途径：

```
coddy hooks list [--cwd DIR]
coddy hooks trust <file> [--cwd DIR]
coddy hooks untrust <file> [--cwd DIR]
```

`list` 会打印工作区（在链接的 git worktree 中还会打印保留其批准的主检出）、生效的 `hooks.project_trust`，以及每个文件一行的表：`FILE`（回执使用的名称：项目文件为工作区相对路径，你自己的文件为绝对路径）、`SCOPE`、`TRUST`（`trusted` 或 `needs_approval`）及其钩子摘要（`PreToolUse(run_command); PostToolUse(*)`，无法解析的文件为 `invalid: <error>`），然后在项目文件等待批准时给出提示。`trust` 会打印它即将批准的钩子，并在 `~/.coddy/hooks-trust.json` 中记录回执，以规范工作区路径为键（链接的 git worktree 使用其主检出的路径：[Git worktrees](worktrees.md#worktree-与其主检出共享的内容)）、文件及其摘要；用户范围文件不需要批准，命令会说明这一点，无效文件无法批准。`untrust` 会撤回回执。`--cwd` 默认为进程工作目录。

通过 HTTP，`GET /coddy/hooks?cwd=<absolute path>` 返回相同目录，其中每个处理器为一行；`POST /coddy/hooks/trust` / `POST /coddy/hooks/untrust` 配合正文 `{"cwd": ..., "file": ".coddy/hooks.json"}` 会记录或撤回回执；`cwd` 必须是会话的服务端工作区，因此此路由是远程控制台或 ACP 客户端的批准路径，它们的本地 `coddy hooks trust` 会在错误的机器上写回执。目录的策略和配置键可在 Settings > Hooks 中更改，或使用内置的 `configure-coddy` 技能更改（对于你信任的检出使用 `set hooks.project_trust=allow`；在 `allow` 下项目文件不需要回执，在 `deny` 下它们从不被读取）。

回执文件是 `mcp-trust.json` 和 `subagents-trust.json` 的兄弟文件，绝不与它们共享：一种批准绝不能读成另一种。写入是一个事务：同一路径的每个存储共享的进程内锁、与其他进程共享的文件锁（`hooks-trust.json.lock`），以及通过唯一临时文件进行的原子替换，因此来自 CLI 和 HTTP 服务器的批准不会互相丢失。

## 示例

阻止破坏性 shell 命令，无论权限模式如何（`~/.coddy/hooks.json`）：

```json
{
  "hooks": {
    "PreToolUse": [
      { "matcher": "run_command", "hooks": [{ "type": "command", "command": "~/.coddy/hooks/no-rm-rf.sh", "failClosed": true }] }
    ]
  }
}
```

```bash
#!/bin/sh
# ~/.coddy/hooks/no-rm-rf.sh
command=$(jq -r '.tool_input.command // ""')
case "$command" in
  *"rm -rf"*)
    printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"rm -rf is not allowed here"}}'
    ;;
esac
exit 0
```

在会话目标的[风险门](session-supervisor.md#风险门禁)只能在轮次结束后报告的步骤运行之前拒绝它们：更改容器或卷的 docker 命令，以及对 git 已跟踪迁移的更改（`~/.coddy/hooks.json`）：

```json
{
  "hooks": {
    "PreToolUse": [
      { "matcher": "run_command", "hooks": [{ "type": "command", "command": "~/.coddy/hooks/no-docker-changes.sh", "failClosed": true }] },
      { "matcher": "edit|write|apply_patch|rm|mv", "hooks": [{ "type": "command", "command": "~/.coddy/hooks/no-migration-edits.sh", "failClosed": true }] }
    ]
  }
}
```

```bash
#!/bin/sh
# ~/.coddy/hooks/no-docker-changes.sh
command=$(jq -r '.tool_input.command // ""') || exit 1
# Join continued lines, then one command per line, lower case, no quote characters or backslashes.
# Each step runs on its own: a pipe would hide the failure of every tool but the last.
joined=$(printf '%s\n' "$command" | awk '{ if (sub(/\\$/, " ")) printf "%s", $0; else print }') || exit 1
lines=$(printf '%s\n' "$joined" | tr 'A-Z;&|' 'a-z\n\n\n') || exit 1
commands=$(printf '%s\n' "$lines" | tr -d "\\\\'\"") || exit 1
# A word is a run of [a-z0-9_./-] read from its last /; $w is a character outside a word.
w='[^a-z0-9_./-]'
# grep exits with 1 when nothing matches, which lets the call through, and with more on an error.
dockers=$(printf '%s\n' "$commands" | grep -E "(^|$w|/)docker(-compose)?($|$w)")
case $? in
  0) ;;
  1) exit 0 ;;
  *) exit 1 ;;
esac
printf '%s\n' "$dockers" | grep -Eq "(^|$w|/)(up|down|rm|stop|restart|create|run|kill|prune)($|$w)"
case $? in
  0) printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"docker commands that change containers or volumes are not allowed here"}}' ;;
  1) ;;
  *) exit 1 ;;
esac
exit 0
```

```bash
#!/bin/sh
# ~/.coddy/hooks/no-migration-edits.sh
input=$(cat)
for key in path src dst; do
  file=$(printf '%s' "$input" | jq -r --arg key "$key" '.tool_input[$key] // empty') || exit 1
  case "/$file" in
    */migrations|*/migrations/*|*/alembic/versions|*/alembic/versions/*|*/db/migrate|*/db/migrate/*|*/db/changelog|*/db/changelog/*)
      git ls-files --error-unmatch -- "$file" >/dev/null 2>&1
      status=$?
      if [ "$status" -eq 0 ]; then
        printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"committed migrations are not changed here; add a new migration"}}'
        exit 0
      elif [ "$status" -ne 1 ]; then
        exit 1
      fi
      ;;
  esac
done
exit 0
```

第一个脚本像风险门那样读取命令：它会连接续行，去掉大小写、引号字符和反斜杠，在 `;`、`&`、`|` 和换行处把行切分为命令，并拒绝包含 `docker` 或 `docker-compose` 且包含 `up`、`down`、`rm`、`stop`、`restart`、`create`、`run`、`kill` 或 `prune` 之一的命令。第二个脚本读取 `path`，对于 `mv` 同时读取 `src` 和 `dst`，因此移走已跟踪的迁移，以及用另一个文件覆盖一个迁移，都会被拒绝。它只拒绝 git 跟踪的文件，因此新的迁移仍然可以写入。对于 git 不跟踪的路径，`git ls-files --error-unmatch` 以 1 退出，只有该状态才让调用通过。当 `jq` 无法解析输入时，两个脚本都以 1 退出。第一个脚本在 `awk`、`tr` 或 `grep` 缺失或失败时也以 1 退出，每个步骤单独运行，因为管道只会报告最后一个工具的状态；没有匹配时 `grep` 以 1 退出，只有该状态才让调用通过。第二个脚本在 git 以任何其他方式失败时以 1 退出（git 缺失、没有仓库、索引不可读）。`failClosed: true` 会把每个这样的退出变成阻止，因此既不会让无法解析的输入、缺失的工具，也不会让无法回答的 git 放行调用。钩子在每次调用运行前看到它，而风险门在轮次结束后读取结果；钩子适用于每个会话，而不仅适用于某个目标。两个脚本都不能捕获所有拼写：两者都匹配文本，因此第一个还会拒绝 `echo docker compose down`，并漏掉隐藏在类似 `./deploy.sh` 的脚本中的命令，第二个只看到工具调用所指名的文件，而看不到 shell 命令重写的文件。

在每次编辑后运行格式化器，并告诉模型它改了什么：

```json
{
  "hooks": {
    "PostToolUse": [
      { "matcher": "edit|write|apply_patch", "hooks": [{ "type": "command", "command": "~/.coddy/hooks/gofmt.sh" }] }
    ]
  }
}
```

```bash
#!/bin/sh
path=$(jq -r '.tool_input.path // ""')
case "$path" in
  *.go)
    if out=$(gofmt -l "$path" 2>&1) && [ -n "$out" ]; then
      gofmt -w "$path"
      printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"gofmt reformatted %s"}}' "$path"
    fi
    ;;
esac
exit 0
```

将每次工具调用记录到文件而不影响流程（`async` 钩子不会拖慢轮次）：

```json
{
  "hooks": {
    "PreToolUse": [
      { "hooks": [{ "type": "command", "command": "cat >> ~/.coddy/hook-audit.jsonl", "async": true }] }
    ]
  }
}
```

## Windows

Shell 形式的命令会通过 `run_command` 检测到的 shell（pwsh、PowerShell 或 cmd）运行，因此请用该 shell 的语法编写，或提供 `commandWindows`。Exec 形式的处理器（存在 `args`）需要真正的可执行文件，而不是 `.cmd` shim。超时会终止钩子的整个进程树。

## 与 Claude Code 的差异

- 只有 `command` 处理器会运行；`http`、`prompt`、`agent` 和 `mcp_tool` 会被跳过。
- `tool_input` 携带 Coddy 的参数名。
- `if` 过滤器会被忽略。
- 匹配的钩子按顺序运行，而不是并行运行，因此 `updatedInput` 会确定性地串联。
- `PreToolUse` 的 `allow` 会跳过 Coddy 的权限提示；不存在它可以从属的拒绝规则。
- 项目文件由摘要绑定的回执带外批准，绝不由聊天内提示批准。

## 测试

- `features/` 中的可执行规范：`hooks_tool_calls.feature`（工具事件、负载、退出码契约、Claude Code 别名）、`hooks_project_trust.feature`（暂留和批准的项目文件、策略、通知行、HTTP 目录和批准路由）、`hooks_turn_lifecycle.feature`（提示词拒绝和上下文、停止循环、会话启动上下文、压缩否决和摘要）、`hooks_subagents.feature`（子会话内的钩子、`SubagentStart` / `SubagentStop`、权限提示通知）。godog 框架位于 `internal/agent/bdd_hooks_test.go`、`bdd_hooks_subagents_test.go` 和 `external/httpserver/bdd_hooks_test.go`；钩子进程是通过 `internal/hooks/hooktest` 重新执行的测试二进制本身，因此不涉及 shell 脚本，运行器也在 Windows 上覆盖（`internal/hooks` 在 Windows CI 任务中）。
- 单元测试：`internal/hooks/hooks_test.go`（文件形态、匹配器、加载器范围、运行器契约）、`internal/hooks/trust_test.go`（回执、目录）、`internal/config/hooks_test.go`、`cmd/coddy/hooks_test.go`。
- 在每个界面上针对真实模型进行端到端测试，并接入三个运行器：`examples/acp/acp_e2e_hooks.py`（通过 stdio 的 ACP；轮次之间使用 `coddy hooks list` 和 `coddy hooks trust`）、`examples/httpserver/http_e2e_hooks.py`（`GET /coddy/hooks` 带有被暂留的项目文件，`uiLog` 中的 `notice` 行，`POST /coddy/hooks/trust`，第二轮运行已批准的钩子，`POST /coddy/hooks/untrust`）以及 `examples/cli/cli_e2e_hooks.py`（带有相同记录器钩子的控制台 TUI，会话的 `ui_log.json` 中的通知，通过 CLI 批准）。每个测试都通过用户范围钩子记录真实 `run_command` 的每个 `PreToolUse` 和 `PostToolUse` 负载，检查项目范围钩子保持暂留，批准它，并检查同一会话的下一轮会运行它。

<!-- docsgen:source sha256=b6a5cf564c9675f8 -->

# 会话导出（`/export`）

内置的 `/export` 命令会将当前对话写入会话工作区中的文件。它的运行方式与 `/compact` 和 `/plugin` 相同：以确定性的方式运行，不产生模型轮次，在所有提示词界面（控制台 TUI、ACP 编辑器、Web 界面输入框、`POST /v1/responses`）上均可使用。该命令遵循 qwen-code 的 `/export`；裁剪选项遵循 Kilo Code 等基于 OpenCode 的工具的导出对话框。

## 用法

```text
/export [md|html|json|jsonl] [path] [--no-tools] [--no-thinking]
```

| 命令 | 结果 |
|---------|--------|
| `/export` | `coddy-export-<timestamp>.md`，位于工作区 |
| `/export json` | `coddy-export-<timestamp>.json`，位于工作区 |
| `/export md chat.md` | `chat.md` |
| `/export chat.html` | `chat.html`；格式由扩展名决定 |
| `/export notes/` | markdown 写入 `notes/`，不存在时创建 |
| `/export md chat.md --no-tools --no-thinking` | 仅对话文本 |

回复会给出文件相对于工作区的名称，以及其完整路径。时间戳为 UTC，例如 `coddy-export-2026-09-06T12-34-56Z.md`。生成的名称通过独占创建进行预留，因此当名称已被占用时（同一秒内的第二次导出，甚至来自共享该工作区的另一个会话），文件会在扩展名前获得下一个可用的 `-2`、`-3` 等后缀。命令后的单词以空白分隔，因此路径中的连续空格或制表符会合并为一个空格。

### 格式

当 `/export` 后的第一个单词表示一种格式时，该单词就是格式：`md`（或 `markdown`）、`html`、`json`、`jsonl`。没有该单词时，由目标的扩展名决定（`.md`、`.markdown`、`.html`、`.htm`、`.json`、`.jsonl`）；空目标或目录会导出 markdown。既不是格式也不是现有目录的裸词会被拒绝，因此 `/export yaml` 会报告支持的格式，而不会创建名为 `yaml` 的文件。

### 路径

- 相对路径会相对于会话工作区解析（即输入框的工作区标签或控制台页脚中显示的文件夹）。绝对路径在指向该工作区内部时会被接受。
- 现有目录、以 `/` 结尾的路径，或 `.`，会使用生成的文件名。其他任何路径都用于指定文件名；缺失的父目录会被创建。
- 通过 `..` 或符号链接离开工作区的目标会被拒绝。每个文件系统步骤都通过在工作区上打开的 `os.Root` 运行：Go 会在每次操作时拒绝指向根目录之外的符号链接，因此在预检查与写入之间出现的链接也会被拒绝。根据 `os.Root` 约定，不会检查工作区之下的文件系统边界和绑定挂载。
- 显式名称会写入新的临时文件，然后重命名覆盖目标，因此现有的常规文件会被替换，且绝不会保留更宽的权限。目标是目录、符号链接或任何其他非常规文件时会被拒绝。
- 在 POSIX 系统上，文件以仅所有者权限创建（`0600`）。Windows 没有等效的模式位：在那里文件会继承其目录的 ACL。

### 选项

- `--no-tools` 省略工具调用及其结果。只调用了工具的助手行会随之消失。
- `--no-thinking` 省略模型的推理。
- 选项可以出现在命令后的任意位置。未知的 `--option` 会被拒绝，并显示用法行。

## 文件包含的内容

导出内容在 `/export` 行进入对话记录之前构建，因此文件包含截至该命令的对话。命令及其回复随后会像其他所有内置命令一样被追加；若要包含它们，请再次导出。

**头部。** 会话 id、标题（已固定，或根据第一个提示词推导）、工作区路径、当工作区是检出目录时的 git 分支、模型、第一条消息的时间、导出时间、消息数和用户轮次数，以及会话的 `stats.json` 存在时的 token 总数。

**条目**，按对话记录顺序：

| 类型 | 内容 |
|------|---------|
| `user` | 提示词文本，原样保留。已填充的 `@` 提及（`<coddy_attachment>` 块）和上传文件会简化为路径与名称的附件列表；其正文不会被复制，它们留下的空隙会被闭合。 |
| `assistant` | 推理、文本，以及通过 `tool_call_id` 与其结果配对的工具调用；返回图片的调用（对图片文件使用 `read`、MCP 工具的图片块、[Images](images.md)）会按随会话资产保留的副本将其列在 `pictures` 下。图片或上传文件的副本会以会话目录当前所在位置命名，即使该目录在文件保存后发生了移动。没有文本、推理或调用的行（已取消的轮次）会被跳过。 |
| `tool_result` | 其调用不在对话记录中的结果。 |
| `compaction_summary` | 由 `/compact` 或自动压缩插入的摘要行。 |
| `plan_document` | 包含 slug、名称和文件路径的计划文档行。 |

## 格式

**Markdown**（`md`）：先是一个头部列表，然后是 `## User` 和 `## Assistant · <model> · <time>` 章节。推理位于折叠的 `<details>` 块中，每个工具调用变成 `### Tool call: <name>`，其中包含带围栏的输入（格式化后的 JSON）和结果。当内容本身包含反引号时，围栏会增长，因此工具输出中的代码保持完整。

**HTML**（`html`）：一个自包含页面，使用内联 CSS，通过 `prefers-color-scheme` 提供浅色和深色配色，不含脚本。每个值都会转义，文本以预换行块显示；助手回复中的 markdown 会原样保留，而不进行渲染。

**JSON**（`json`）：单个文档。

```json
{
  "version": 1,
  "session": {
    "id": "sess_...", "title": "...", "cwd": "/path", "git_branch": "main", "model": "openai/gpt-5.6-terra",
    "started_at": "2026-09-06T10:00:00Z", "exported_at": "2026-09-06T12:00:00Z",
    "message_count": 8, "user_turns": 3,
    "token_usage": {"input_tokens": 1200, "output_tokens": 300, "total_tokens": 1500}
  },
  "entries": [
    {"type": "user", "created_at": "...", "text": "...", "attachments": [{"path": "README.md", "name": "README.md"}]},
    {"type": "assistant", "created_at": "...", "model": "...", "reasoning": "...", "text": "...",
     "tool_calls": [{"id": "call_1", "name": "read", "input": {"path": "README.md"}, "result": "..."},
                    {"id": "call_2", "name": "read", "input": {"path": "shot.png"}, "result": "...",
                     "pictures": [{"path": "/home/me/.coddy/sessions/<id>/assets/shot-1a2b3c4d5e6f7a8b.png", "name": "shot.png"}]}]},
    {"type": "compaction_summary", "created_at": "...", "model": "...", "text": "..."},
    {"type": "plan_document", "created_at": "...", "text": "...", "plan": {"slug": "...", "name": "...", "path": "..."}}
  ]
}
```

空字段会被省略，但有一个例外：当工具没有返回任何内容时，`tool_calls[].result` 以 `""` 的形式存在；当对话记录中没有该调用的结果时（已取消或仍在运行的轮次），该字段不存在。`tool_calls[].input` 是调用的 JSON 参数；当模型生成的参数不是有效 JSON 时，则为 JSON 字符串。`tool_result` 条目带有 `tool_call_id` 和 `text`。

**JSON Lines**（`jsonl`）：第一行是头部，`{"type":"session","version":1,...}`，字段与上面的 `session` 相同；之后每一行是一个条目。

## 从 shell 使用：`coddy sessions export`

无需运行中的智能体也可使用同样的导出，就像 `coddy plugin` 与 `/plugin` 成对出现一样：

```bash
coddy sessions export <session-id> [--format md|html|json|jsonl] [--out <path>] [--no-tools] [--no-thinking] [--sessions-dir <path>]
```

- `<session-id>` 是来自 `coddy sessions list` 的文件夹 id，或恰好匹配一个已存储会话的前缀；有歧义的前缀会列出候选。
- 默认情况下，文件会落在当前目录。`--out` 可指定任意位置的文件或目录（包括绝对路径）；缺失的目录会被创建。格式遵循 `--format`，否则依据 `--out` 扩展名，再否则为 markdown，规则与聊天命令相同。
- `--no-tools` 和 `--no-thinking` 会像聊天选项一样裁剪文档。`--sessions-dir` 指向另一个会话根目录。
- 该命令会打印 `Session exported to <format>: <full path> (<n> transcript entries)`，并在会话未知、格式错误或写入失败时以非零状态退出。标志可以位于 id 之前或之后。
- 文档与聊天导出相同：标题、模型和 git 分支来自存储的 `session.json` 及其指定的工作区，token 总数来自其 `stats.json`。

## 命令列出的位置

`skills.BuiltinCommands` 将 `export` 与 `compact` 和 `plugin` 一起发布，因此无需额外接线即可到达 ACP `available_commands_update` 目录、`GET /coddy/commands`、输入框的 **Commands** 组，以及处于 `--remote` 模式的控制台。

## 限制

- 文件会落在运行智能体的机器上。当 `coddy serve` 位于另一台主机，或控制台处于 `--remote` 模式时，请在服务器的工作区中查找；回复会打印完整路径。
- Telegram 网关只响应它自己的命令（`/clear`、`/help`、`/model`、`/context`、`/resume`、`/start`）和设置命令（`/agent`、`/plan`、`/ask`、`/reasoning`、`/think`、`/nothink`），并丢弃其他斜杠命令，因此 `/export` 在那里不可用。
- 子智能体会话从不运行内置命令：以 `/export` 开头的子提示词对子智能体来说是一个普通任务。

## 代码

- `internal/session/transcript_export.go`：文档（`BuildExportDocument`）、渲染器（`RenderExport`）、目标解析（`ResolveExportRequest`、`ResolveExportTarget`）以及受保护的写入（`WriteExportFile`、`ExportSession`）。
- `internal/agent/export_command.go`：解析与回复；`Agent.Run` 在 ReAct 循环之前分派该命令。
- `cmd/coddy/sessions.go`：`coddy sessions export` 子命令（`PrepareExportOutput` 将 `--out` 映射到输出根目录，`FileStore.ResolveSessionID` 接受唯一 id 前缀）；规范 `features/session_export_cli.feature`，框架 `cmd/coddy/bdd_sessions_export_test.go`。
- `features/session_export.feature`：正常路径规范，由 `external/httpserver/bdd_session_export_test.go` 通过 `POST /v1/responses` 驱动（在 `-tags http` 下运行）；转义、符号链接、未知格式和选项由代码旁边的单元测试覆盖。

<!-- docsgen:source sha256=c99d50a4c6e4b0c4 -->

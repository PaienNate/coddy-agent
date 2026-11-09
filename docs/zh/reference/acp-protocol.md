# ACP 协议参考

## 概述

Coddy 将 ACP 实现为**框架的线协议契约**。ACP 标准化客户端（例如编辑器、脚本或编排器）如何与智能体进程通信。默认配置呈现的是一个**编码智能体**，但传输层和 RPC 方法是通用的框架接口 - 初始化、会话生命周期、`session/prompt`、权限流程和 MCP 相关选项。

参考：https://agentclientprotocol.com/protocol/overview

## 传输

所有消息均为通过 **stdin/stdout** 发送的、以换行分隔的 JSON 对象。

```
stdin  -> messages from Client to Agent
stdout -> messages from Agent to Client (responses + notifications)
stderr -> agent logs (not protocol messages)
```

## 消息类型

### 请求（客户端到智能体）

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "session/prompt",
  "params": { ... }
}
```

### 响应（智能体到客户端）

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": { ... }
}
```

### 错误响应

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32600,
    "message": "Invalid request"
  }
}
```

### 通知（智能体到客户端，不期望响应）

```json
{
  "jsonrpc": "2.0",
  "method": "session/update",
  "params": { ... }
}
```

## Stdio 客户端（Coddy 特定）

通过管道驱动 **`coddy acp`** 的手写脚本应实现以下行为。参考框架：**`examples/acp/acp_e2e_todo.py`**。

1. **`result` 和 `omitempty` 为 nil** - JSON-RPC 成功负载会以省略 `result` 的方式生成：当 Go 处理程序返回 **`nil`** 指针（例如 **`session/set_mode`**）时。响应行可能只包含 **`jsonrpc`**、**`id`**，既不包含 **`result`** 也不包含 **`error`**。将任何具有匹配 **`id`** 且没有 **`method`** 的对象视为你未完成请求的完成。
2. **交错的 `session/update`** - 在 **`session/prompt`** 之后，智能体会在最终响应之前流式发送许多通知。逐行读取 stdout，直到你请求 **`id`** 对应的行到达；在此期间处理 **`session/request_permission`** 或 **`elicitation/create`**（对于不支持 elicitation 的客户端为 **`session/request_question`**），通过写入具有相同 **`id`** 的客户端响应来回应。
3. **Stdout 缓冲** - 当 stdout 不是 TTY 时，输出可能会块缓冲。用 **`stdbuf -oL -eL`**（或等效工具）包装二进制文件，以便各行在写入时出现。
4. **并发请求处理程序** - 未完成的请求会异步分派。如果你的客户端假设严格顺序，请在上一个请求的响应被消费之前不要发送第二个 RPC。

## 协议流程

```
Client                          Agent
  |                               |
  |-------- initialize ---------->|
  |<------- initialize resp ------|
  |                               |
  |-------- session/new --------->|
  |<------- session/new resp -----|
  |                               |
  |-------- session/prompt ------>|
  |<------- session/update -------|  (notifications: plan, available_commands_update, chunks, tool_calls)
  |<------- session/update -------|
  |<------- session/update -------|
  |<------- session/prompt resp --|  (stopReason: end_turn)
  |                               |
```

## 方法

### `initialize`

协商协议版本并交换能力。

**请求参数：**
```json
{
  "protocolVersion": 1,
  "clientCapabilities": {
    "fs": {
      "readTextFile": true,
      "writeTextFile": true
    },
    "terminal": true,
    "elicitation": {
      "form": {}
    }
  },
  "clientInfo": {
    "name": "example-acp-client",
    "title": "Example ACP Client",
    "version": "1.0.0"
  }
}
```

渲染表单 elicitation 的客户端会在 `clientCapabilities.elicitation.form` 中声明这一点。Coddy 随后以 `elicitation/create` 而非 Coddy 专属的 `session/request_question` 来提问，因此任何实现该标准的编辑器都会显示这些问题。

**响应结果：**
```json
{
  "protocolVersion": 1,
  "agentCapabilities": {
    "loadSession": true,
    "sessionCapabilities": {
      "list": {},
      "delete": {},
      "additionalDirectories": {},
      "resume": {}
    },
    "promptCapabilities": {
      "image": false,
      "audio": false,
      "embeddedContext": true
    },
    "mcpCapabilities": {
      "http": true,
      "sse": false
    }
  },
  "agentInfo": {
    "name": "coddy-agent",
    "title": "Coddy Agent",
    "version": "0.1.0"
  },
  "authMethods": []
}
```

### `session/new`

创建新的会话。

**请求参数：**
```json
{
  "cwd": "/home/user/project",
  "additionalDirectories": ["/home/user/other"],
  "mcpServers": [
    {
      "name": "my-mcp",
      "command": "/path/to/mcp-server",
      "args": ["--stdio"],
      "env": []
    }
  ]
}
```

`additionalDirectories` 指定会话在 `cwd` 之外工作的额外工作区根目录。每个路径都是绝对路径，命名非绝对路径条目的请求会被拒绝；该列表是完整列表：省略它或发送空列表的请求会以无额外根目录启动会话。根目录与会话一起保留，在 `session/list` 上回报，并在 Coddy 判断路径是否为会话可到达路径时被视为工作区的一部分 - 包括提及解析器以及将受约束轮次限制在其工作区内的检查。

**响应结果：**

Coddy 同时返回 **会话配置选项**（现代 ACP 客户端首选）和旧版 **`modes`** 字段以保持兼容。支持 `configOptions` 的客户端应使用它们进行模式和模型选择。

```json
{
  "sessionId": "sess_abc123def456",
  "configOptions": [
    {
      "id": "mode",
      "name": "Session mode",
      "description": "Agent runs tools; Plan focuses on design without execution.",
      "category": "mode",
      "type": "select",
      "currentValue": "agent",
      "options": [
        {
          "value": "agent",
          "name": "Agent",
          "description": "Execute tasks with full tool access"
        },
        {
          "value": "plan",
          "name": "Plan",
          "description": "Plan and design without code execution"
        }
      ]
    },
    {
      "id": "model",
      "name": "Model",
      "description": "LLM used for this session.",
      "category": "model",
      "type": "select",
      "currentValue": "openai/gpt-5.6-terra",
      "options": [
        {
          "value": "openai/gpt-5.6-terra",
          "name": "gpt-5.6-terra",
          "description": "openai"
        }
      ]
    },
    {
      "id": "reasoning",
      "name": "Reasoning",
      "description": "Controls the reasoning effort used for this session.",
      "category": "thought_level",
      "type": "select",
      "currentValue": "medium",
      "options": [
        { "value": "low", "name": "low" },
        { "value": "medium", "name": "medium" },
        { "value": "high", "name": "high" }
      ]
    },
    {
      "id": "permission_mode",
      "name": "Permission mode",
      "description": "Controls when the agent asks for user approval before running tools.",
      "category": "_permission_mode",
      "type": "select",
      "currentValue": "ask",
      "options": [
        {
          "value": "ask",
          "name": "Ask",
          "description": "Always ask before running commands or writing files"
        },
        {
          "value": "accept_edits",
          "name": "Accept edits",
          "description": "Auto-approve file writes inside the workspace; ask before running commands"
        },
        {
          "value": "bypass",
          "name": "Bypass",
          "description": "Never ask for permission"
        }
      ]
    }
  ],
  "modes": {
    "currentModeId": "agent",
    "availableModes": [
      {
        "id": "agent",
        "name": "Agent",
        "description": "Execute tasks with full tool access"
      },
      {
        "id": "plan",
        "name": "Plan",
        "description": "Plan and design without code execution"
      },
      {
        "id": "ask",
        "name": "Ask",
        "description": "Answer questions from the repository without changing anything"
      }
    ]
  }
}
```

`model` 选项仅在智能体配置中的 `models` 列表非空时存在。在通过此选项更改之前，有效默认模型为 `agent.model` - 这可以是用户自己的选择，也可以是交互式客户端在新建会话上打上的、记住的界面选择（控制台和 Web 界面各自保留自己的选择；参见[配置](../getting-started/configuration.md)）。每个列出的 `value` 与 YAML 中的 `models[].model` 字符串（`provider_name/api_model_id`）匹配。

### `session/load`

通过 `sessionId` 重新加载已持久化的会话。智能体恢复 `session.json` 和 `messages.json`，根据请求重建技能和 MCP 连接，通过 `session/update` 重放先前的用户和助手轮次（以及工具调用摘要），发送 `plan` 更新（如果 `todos/active.md` 存在），并在响应上线后发送 `available_commands_update`。重放的用户消息按输入时的样子读取：其提及带来的附件会折叠回这些提及（`@src/app.go:3-5`），绝不会是其正文。

按照 ACP 的要求，重放先于此处的响应，这是安全的，因为客户端自己命名了会话。通过 **`session/new`**（`coddy acp --session-id <id>`）重新打开捆绑包则相反：客户端只能从响应中得知 id，因此重放会等待它。任何更早写入的内容都会到达客户端尚未注册的会话。

**请求参数**（按 ACP 要求，`cwd`、`sessionId` 和 `mcpServers` 为必填）：

```json
{
  "sessionId": "sess_abc123def456",
  "cwd": "/home/user/project",
  "additionalDirectories": [],
  "mcpServers": []
}
```

这里的 `additionalDirectories` 是已加载会话的完整额外根目录列表，与 `session/new` 上相同：省略或空列表会加载无额外根目录的会话。

**响应结果：** 与 `session/new` 类似，返回 `modes` 和 `configOptions`。

### `session/resume`

在不重放对话记录的情况下恢复已存储的会话。保留自己历史副本的客户端 - 已经列出并显示过该会话的客户端 - 会恢复它并获得如何驱动它的信息，而不是之前说过什么。`session/load` 才是执行重放的那个。

**请求参数**（按 ACP 要求，`cwd` 和 `sessionId` 为必填）：

```json
{
  "sessionId": "sess_abc123def456",
  "cwd": "/home/user/project",
  "additionalDirectories": ["/home/user/other"],
  "mcpServers": []
}
```

**响应结果：** 与 `session/new` 类似，返回 `modes` 和 `configOptions`。之前没有承载历史的 `session/update`。

通告为 `sessionCapabilities.resume`。已经活动的会话会就地恢复而不是重建，这可以避免运行中的轮次及其捆绑包最终出现两个写入者。

### `session/delete`

删除会话及其下嵌套的每个捆绑包 - 记录在其中的子智能体运行也会随之删除。会话上运行的轮次会被取消并先等待完成，树中的子智能体任务会在其捆绑包被删除之前停止，因此不会有任何东西重新创建刚被删除的内容。

**请求参数：**
```json
{
  "sessionId": "sess_abc123def456"
}
```

**响应结果：** `{}`

通告为 `sessionCapabilities.delete`。这是编辑器在 `session/list` 的一行上调用时使用的方法。

### `session/list`

列出配置的会话根目录下找到的已持久化会话（参见 README），最新的在前。响应中每个条目包含 `sessionId`、`cwd`、`title` 和 `updatedAt`；子智能体运行和调度器运行的子会话会被省略。

可选的 `cwd` 会将列表缩小到一个工作区。它命名的是文件夹，而不是字符串：路径会被清理，文件夹存在时会解析其符号链接，Windows 和 macOS 会以不区分大小写的方式比较。会话保留其客户端给出的 cwd，而两个客户端经常以不同方式拼写同一个文件夹 - 控制台存储符号链接检出目录的逻辑 `$PWD`，而编辑器发送其解析出的物理路径，Windows 客户端可能会将盘符小写 - 因此由一个客户端创建的会话仍会为另一个客户端列出。父文件夹或子文件夹的会话不会包括在内：在工作区子文件夹中启动的控制台不属于该工作区的会话。`coddy sessions list --cwd` 和 `GET /coddy/sessions?cwd=` 应用相同规则。

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "session/list",
  "params": { "cwd": "/home/user/project" }
}
```

当列表非空时，每一行都携带会话创建时被赋予的 `additionalDirectories`，因此显示会话的客户端无需加载它即可知道其工作根目录。该列表是完整列表，对于没有根目录的会话则省略。

### 磁盘布局（Coddy）

当进程以可写的会话根目录启动时（默认 **`$CODDY_HOME/sessions`**），每个捆绑包都是 `<root>/<sessionId>/`，包含：

- `session.json` - id、cwd、模式、模型覆盖、推理级别、普通会话被切换到的权限模式（`sessionPermissionMode`）或子智能体子会话运行时所处的权限模式（`permissionMode`，一条永远不会被读回的记录）、智能体记忆、派生或固定标题（`titlePinned`）、时间戳、可选的 **`activitySeq`** / **`readActivitySeq`** / **`lastErrorSeq`**，用于跨 HTTP 界面的输入框活动和错误状态同步（最后一个是最近一次真实轮次失败的活动代次，清除时为零）
- `messages.json` - LLM 消息历史（角色 user、assistant、tool）
- `assets/` - 保留用于未来会话范围内的文件
- `todos/active.md` - 从计划工具同步的当前待办清单
- `todos/archive/todo-<nanos>.md` - 已完成列表被替换时的归档列表
- `plans/<slug>.plan.md` - 设计方案文件（YAML frontmatter + markdown 正文），在计划模式下通过 **`plan_write`** 写入

配置了存储时，服务器始终通告 **`loadSession`**（`coddy acp` 和 **`coddy serve`** 在启动时打开一个 **`FileStore`**）。

### 设计方案（计划模式）

计划模式仅使用标准 ACP。智能体使用工具 **`plan_write`** / **`plan_list`** 保存文件（不是 JSON-RPC 扩展）。

在 **`plan_write`** 之后，Coddy 发布：

- **`session/update`**，其中包含来自文件 frontmatter 的 `sessionUpdate: "plan"` 和清单 `entries`（供任何 ACP 客户端预览）
- 在该更新上发布 **`_meta`**：`coddy.dev/planSlug`、`coddy.dev/planKind: "design"`（可选加入；用于区分设计方案与实时待办清单更新）
- 为捆绑 UI 在 **`messages.json`** 中持久化一行 **`plan_document`**（当模型总结计划时，也会在聊天中作为助手 markdown 可见）

**运行计划**（开始实现）无需自定义 `_coddy/*` 方法：

1. **Coddy 感知** - `session/prompt` 使用 `_meta`：

```json
{
  "sessionId": "sess_…",
  "prompt": [{ "type": "text", "text": "Implement the plan." }],
  "_meta": { "coddy.dev/runPlanSlug": "my-feature" }
}
```

Coddy 切换到 **agent** 模式，将计划正文注入系统提示词，并运行该轮次。会话待办（`todos/active.md`）**不会**从设计方案自动填充。在 **ask** 模式下，该钩子会被拒绝并报错，`@plans/<slug>.plan.md` 提及仅作为阅读材料内联：请先切换模式，再运行。

2. **可移植方式** - 客户端将 `mode` 设置为 **agent**，然后 `session/prompt` 引用 `@plans/<slug>.plan.md` 或类似 *实现 my-feature 计划* 的文本。

HTTP **`POST /v1/responses`** 通过 JSON **`metadata.runPlanSlug`** 接受相同的钩子（捆绑 UI）。方案文件的 CRUD 仅在 **`/coddy/sessions/{id}/plans`** 下通过 HTTP 提供（不属于核心 ACP）。

### `session/prompt`

发送用户消息，启动 ReAct 循环。

**请求参数：**
```json
{
  "sessionId": "sess_abc123def456",
  "prompt": [
    {
      "type": "text",
      "text": "Refactor the auth module to use JWT"
    }
  ],
  "_meta": {
    "coddy.dev/runPlanSlug": "optional-slug-for-run-plan"
  }
}
```

**响应结果：**
```json
{
  "stopReason": "end_turn"
}
```

停止原因：`end_turn` | `max_tokens` | `max_turns` | `agent_refused` | `cancelled`

**提及和上下文块。** Coddy 通告 `promptCapabilities.embeddedContext`，因此 Zed 等客户端会将其提及菜单选择的内容作为文本旁边的上下文块发送：

- 带有 `text` 的 `resource` 按发送时的内容附加（Zed 会包含未保存的编辑）。`file://` 的块通过其路径命名 - 当它位于会话的 `cwd` 中时使用相对路径 - 行片段（`#L10-20`、`#L10:20`、`#L10-L20`、`#L10`）会用这些行标记附件；诸如 Zed 的 `?symbol=` 之类的查询会被丢弃；
- 不带 `text` 的 `resource` 会从磁盘读取，位置由客户端指定：文件、文件的行范围或文件夹列表。缺失的文件或超出末尾的行会使提示词失败，因为客户端明确请求了它们；
- 指向本地文件或文件夹的 `resource_link`（每个 ACP 智能体都必须接受的块）会像提及一样被读取；Coddy 自身无法打开的链接 - 编辑器内部 URI、Web 地址 - 会以命名它的行到达模型。

在 `text` 块中输入的 `@` 提及会按每个界面的解析方式解析 - 磁盘上任意位置的文件、文件夹、行范围、`@session:<id>`、`@rule:<name>`、`@agent:<name>`、`@coddy:<page>#<section>`、网页 - 成为同一条用户消息的附件（[提及](../features/mentions.md)）。提及绝不会使提示词失败：不命名任何内容的提及会保留为普通文本。

### 子智能体运行和子会话（Coddy 特定）

当智能体委派给子智能体（`docs/features/subagents.md`）时，协议层面没有任何变化。父级的 `tool_call` / `tool_call_update` 行携带 `spawn_agent` 调用及其结果；子级在自己的会话中运行，**其更新永远不会到达 ACP 客户端**：子级的进度会进入后台任务的输出日志，因此编辑器永远不会收到它未创建的会话 id 的 `session/update`。客户端可以代表子级收到的唯一消息是在生成轮次仍在进行时的 `session/request_permission`；它会带着**父级**的 `sessionId` 和 `toolCall.title` 到达，其前缀为 `[subagent <name>]`，并像其他任何请求一样被应答。该轮次返回后，子级的请求会被拒绝，不会到达客户端。

子会话是只读对话记录。`session/list` 会省略它们，`session/load` 会像任何其他捆绑包一样重放一个子会话，而对子会话 id 调用 `session/prompt` 会返回一个命名父级的错误（`subagent sessions are read-only transcripts: sess_… belongs to sess_…`）；运行计划的 `_meta` 钩子也受同一保护。id 本身不会说明哪个是哪个 - 捆绑包的 `subagentRun` 元数据会说明。

### `session/cancel`

取消正在进行的提示词轮次（通知）。

```json
{
  "jsonrpc": "2.0",
  "method": "session/cancel",
  "params": {
    "sessionId": "sess_abc123def456"
  }
}
```

对于可写的会话捆绑包，Coddy 还会写入一个小的磁盘取消信号，以便另一个 **`coddy`** 进程（例如 **`coddy serve`** 在 **`coddy acp`** 运行该轮次时）可以在轮次期间轮询间隔之间观察到协作式取消。当会话加载在此进程中时，进程内轮次仍会通过相同的 **`TurnCtx`** 取消钩子结束。

### `session/set_mode`

在智能体模式之间切换（旧版 API）。当客户端支持会话配置选项时，优先使用带有 `configId` `mode` 的 `session/set_config_option`。

**请求参数：**
```json
{
  "sessionId": "sess_abc123def456",
  "modeId": "plan"
}
```

**响应结果：** `null`

当模式更改时，智能体还会发送带有 `config_option_update` 的 `session/update`，以便使用配置选项的客户端保持同步（例如，如果未设置会话模型覆盖，显示的默认模型可能会变化）。

### `session/set_config_option`

更改会话配置选项（ACP 会话配置选项）。支持的选项：**`mode`**、**`model`**、**`reasoning`**（当会话的模型具有级别时提供）、**`toolset`**（当配置具有 `toolsets[]` 预设时提供：这些预设和 `default`，[工具集](../getting-started/configuration.md#工具集)）、**`permission_mode`**。每次更改都会经过与设置命令及其他界面相同的设置器，因此观察该会话的浏览器或控制台会看到它（[会话设置](../features/session-settings.md)）。

**请求参数：**
```json
{
  "sessionId": "sess_abc123def456",
  "configId": "permission_mode",
  "value": "accept_edits"
}
```

有效的 `permission_mode` 值：`ask` | `accept_edits` | `bypass`。模式是会话自己的，并持久化在其 `session.json` 中，因此智能体重启后会保留它。新会话会以任何界面上最后选择的模式启动，包括此选项，并在尚未选择任何内容时以 `ask` 启动：Coddy 将该选择保存在 `${CODDY_HOME}/permission-mode.json` 中，而 `config.yaml` 没有对应键（[安全与信任](../operate/security.md#权限模式与提示)）。该选项在 `_permission_mode` 类别中通告，推理级别在 `thought_level` 中，工具集在 `_toolset` 中。

仅对后续轮次生效的更改没有配置选项：将设置命令作为提示词文本的开头发送（`/model <id> --once review this`、`/nothink --count=3`），智能体会在轮次开始前将其取走。

**响应结果：** 带有更新后 `currentValue` 字段的完整 `configOptions` 数组：

```json
{
  "configOptions": [ ... ]
}
```

未知的 `configId` 或该选项下未列出的 `value` 会产生 JSON-RPC 错误（无效参数）。

## 通知（智能体 -> 客户端）

全部通过 `session/update` 方法发送，并带有 `sessionUpdate` 判别字段。

### `plan` - 智能体执行计划

```json
{
  "sessionUpdate": "plan",
  "entries": [
    { "content": "Read auth module", "priority": "high", "status": "pending" },
    { "content": "Design JWT structure", "priority": "high", "status": "pending" },
    { "content": "Implement changes", "priority": "medium", "status": "pending" }
  ]
}
```

### `available_commands_update` - 斜杠命令

在 **`session/new`** 和 **`session/load`** 之后，Coddy 列出其斜杠命令。内置命令位于列表开头，与 **`GET /coddy/commands`** 相同的行：设置命令（**`model`**、**`reasoning`**、**`think`**、**`nothink`**、配置具有工具集预设时的 **`toolset`**、**`agent`**、**`plan`**、**`ask`**、**`permissions`**），然后在启用压缩时是 **`compact`**、**`export`** 和 **`plugin`**。技能随后出现，来自与 **`GET /coddy/slash-commands`** 相同的 **`ListSkills`** 管道，且没有与内置命令或其别名同名的技能。注册会话的响应会先于此通知写入，因此客户端不会将目录作为未知会话的更新而丢弃。行使用 ACP 的 **`name`**、**`description`**，对于设置命令，使用 **`input.hint`** 及其参数和标志（[斜杠命令](https://agentclientprotocol.com/protocol/slash-commands)）。每当目录变化时，智能体可以重复此通知。

```json
{
  "sessionUpdate": "available_commands_update",
  "availableCommands": [
    { "name": "model", "description": "Switch the model for this session, or for the next turns with --once / --count=N", "input": { "hint": "<model id> [--once|--count=N]" } },
    { "name": "demo", "description": "Runs the demo checklist" }
  ]
}
```

### `agent_message_chunk` - 文本响应块

```json
{
  "sessionUpdate": "agent_message_chunk",
  "content": {
    "type": "text",
    "text": "I'll start by reading the current auth module..."
  }
}
```

### `tool_call` - 工具调用开始

```json
{
  "sessionUpdate": "tool_call",
  "toolCallId": "call_001",
  "title": "Reading auth.go",
  "kind": "read",
  "status": "pending"
}
```

### `tool_call_update` - 工具调用状态更新

```json
{
  "sessionUpdate": "tool_call_update",
  "toolCallId": "call_001",
  "status": "completed",
  "content": [
    {
      "type": "content",
      "content": { "type": "text", "text": "File contents: ..." }
    }
  ]
}
```

工具调用状态：`pending` | `in_progress` | `completed` | `failed` | `cancelled`

工具调用类型是 ACP `ToolKind` 枚举的值，Coddy 不会发送其他值：`edit` 用于 `write`、`edit`、`apply_patch`、`mkdir`、`touch`、`config_commit` 和 `config_rollback`；`delete` 用于 `rm` 和 `rmdir`；`move` 用于 `mv`；`execute` 用于 `run_command` 和 `ssh_run_command`；`search` 用于 `glob`、`grep` 和 `coddy_docs_search`；`fetch` 用于 `webfetch`、`websearch` 和 `http_request`；`read` 用于其他读取工具；`other` 用于其他所有工具，包括 MCP 工具。相同的类型会进入 `session/request_permission` 的 `toolCall` 中。

### Coddy 特定的会话更新：`token_usage`、`usage_update`、`provider_usage`

Coddy 发送四种 ACP 规范未定义的 `session/update` 类型；忽略未知类型的客户端仍可正常工作。

- **`turn_progress`** 携带轮次时钟和输出 token。可选的 `tokensPerSecond` 报告生成速度；`completed: true` 标记最终更新，包括已停止或失败的轮次；存在时，速率为已测量模型调用的平均值。保存的平均值会在会话加载时重放。阻塞（非流式）提供商、ACP 智能体模型以及以单一输出块交付的调用不会为速度平均值贡献 token 或时间；输出计数仍包括它们。没有可测量的调用间隔时，实时输出计数仍然可用，但速率会被省略，且不会保存生成图注。存在平均值时，其保存的图注会将所有调用的总输出 token 与仅已测量调用的速率配对。控制台和 Web 界面也使用相同的计算。
- 每次模型调用完成后发送 **`token_usage`**：`inputTokens`、该调用的 `outputTokens`，以及整个轮次累计的 `totalTokens`。`usageKnown` 表示该调用的用量是否已知；`false` 表示即使数值字段非零，它们也不构成完整用量。`true` 包含显式报告的零。该字段为兼容旧发送方而可选。该标志描述本次调用，而不是本轮次较早调用中 `totalTokens` 的完整性。可选的 `sessionUsageKnown` 报告整个会话的累计用量是否完整。一旦某次调用的用量未知，即使后续调用具有 `usageKnown: true`，它也仍为 `false`。较旧的服务器可能省略该字段；为兼容性，客户端将其缺失视为已知。`inputTokens` 是该调用的完整提示词，包括由提供商提示词缓存提供的部分；该部分为 `cachedInputTokens`，提供商未报告时省略。`reasoningTokens` 是该调用的 `outputTokens` 中用于推理的部分，提供商未细分时省略。
- 当上下文窗口占用变化时发送 **`usage_update`**（模型调用、手动或自动压缩）：以 token 为单位的 `used` 和 `size`。
- **`provider_usage`**：会话模型提供商背后的账户配额，因此客户端可以绘制状态栏，就像控制台页脚中模型下方的配额一样。在会话就绪时以及每轮次后发送（绝不发送给子智能体子会话），前提是提供商已有用量来源且该行的面板已开启（`providers[].usage_limits_panel`，默认 true）；目前有来源的类型是 `neuraldeep`（hub 的只读 `GET /v1/limits`）、`codex`（Codex 后端的用量端点）和 `devin`（席位管理状态 RPC）。快照是账户范围的，并由管理器缓存（20 秒，读取之间至少 15 秒）；相对时长（`resetInSec`、`retryInSec`、`rate.resetInSec`）在交付时会按快照的年龄进行校正。永远不会出现美元金额和凭据。HTTP 客户端从 `GET /coddy/providers/{name}/usage` 读取相同结构（参见 `docs/reference/http-api.md`）。开启 `agent.wait_for_limit_reset` 时，如果某个轮次遇到重试永远无法覆盖的限制，它会在等待期间每 20 秒自行发送相同的更新，并带有 `blocked`、`retryAt`、`retryInSec` 和 `resuming: true`；然后重新发出调用，下一次轮次结束的快照会替换该更新：`{"sessionUpdate": "provider_usage", "provider": "neuraldeep", "providerType": "neuraldeep", "blocked": true, "retryAt": "2026-09-06T20:59:59Z", "retryInSec": 767, "resuming": true}`。

```json
{
  "sessionUpdate": "provider_usage",
  "provider": "neuraldeep",
  "providerType": "neuraldeep",
  "observedAt": "2026-09-06T17:47:02Z",
  "fetchedAt": "2026-09-06T17:47:10Z",
  "plan": "pro",
  "keyName": "coddy",
  "windows": [
    {"id": "session", "label": "3h", "used": 407, "limit": 15000, "remaining": 14593,
     "usedPercent": 2.71, "resetsAt": "2026-09-06T17:59:59Z", "resetInSec": 777},
    {"id": "week", "label": "week", "used": 9981, "limit": 150000, "remaining": 140019,
     "usedPercent": 6.65, "resetsAt": "2026-09-07T00:00:00Z", "resetInSec": 22378},
    {"id": "day", "label": "day", "usedPercent": 0, "resetsAt": "2026-09-07T00:00:00Z", "resetInSec": 22378}
  ],
  "rate": {"used": 2, "limit": 120, "remaining": 118, "resetInSec": 58},
  "wallet": {"balanceRub": -1229.24, "spentRub30d": 2000.74},
  "blocked": false,
  "unlimitedModels": ["qwen3.6-35b-a3b"],
  "blockedModels": [{"model": "kimi-k2.6", "blocker": "kimi_budget_exhausted",
                     "retryAt": "2026-10-09T20:15:41Z", "retryInSec": 2860119}]
}
```

`blocked: true` 带有 `blockers`（`session_exhausted`、`week_exhausted`、`rpm_exhausted`、`session_cooldown`、`abuse_cooldown`、`daily_capacity_exhausted`、`key_blocked`、`key_cap_blocked`、`wallet_empty`、`user_blocked`），对于有时间限制的那些，还带有 `retryAt` / `retryInSec`。`unlimited: true` 标记没有容量窗口的密钥；`unlimitedModels` 列出在计量密钥上绕过这些窗口的上游模型 id（客户端会比较模型选择器中第一个 `/` 之后的部分）。`blockedModels` 是镜像情况：该密钥现在**不能**调用的模型 id，每个都带有其 `blocker`，并且当门控有时间限制时，还带有 `retryAt` / `retryInSec`。这种门控覆盖目录的一部分而不是聊天类别，因此 `blocked` 保持 `false`，账户会为其他每个模型应答 - 仅读取 `blocked` 的客户端会显示账户健康，而对所选模型的每个请求都被拒绝。按匹配 `unlimitedModels` 的方式匹配该列表。读取失败会保留先前的窗口，并带有 `stale: true` 和 `error`（`unavailable`、`invalid`），但被拒绝的密钥除外：`error: unauthorized` 不带窗口到达，因为用 hub 不再认可的密钥读取的数字已不再是该账户的数字。`unsupported: true` 由针对没有来源的提供商类型的 REST 路由应答，并且对于用量限制面板已关闭的行（`providers[].usage_limits_panel: false`），会与 `disabled: true` 一起应答；这样的行永远不会被读取，也永远不会为其发送更新。

### `background_wake` - 无人输入的轮次

模型使用 `notify_on_finish` 启动的任务在结束时唤醒智能体（[后台任务](../features/background-tasks.md#任务完成时唤醒智能体)）。在 `coddy acp` 下，被唤醒的轮次以客户端作为发送方运行：其 `session/update` 通知会在客户端发送的任何 `session/prompt` 之外到达，其中的受门控工具会像其他任何工具一样通过 `session/request_permission` 询问。该轮次以此更新开始，在第一条消息持久化之前发送并取而代之：该消息是模型读取的指令，不会作为 `user_message_chunk` 发送，无论是实时还是在 `session/load` 上。

| 字段 | 含义 |
|---|---|
| `tasks` | 轮次报告的每个任务，按完成顺序 |
| `tasks[].id` | 任务 id（`bg_3`） |
| `tasks[].kind` | `command` 或 `agent` |
| `tasks[].label` | 命令，或子智能体运行的描述 |
| `tasks[].agent` | 智能体运行背后的子智能体定义 |
| `tasks[].status` | `succeeded`、`failed`、`timed_out` 或 `stopped` |
| `tasks[].exitCode` | 命令的退出码；智能体运行的是池的退出码，不说明任何问题 |
| `tasks[].durationMs` | 任务运行了多长时间 |
| `tasks[].error` | 池记录了某些内容时，出了什么问题 |

```json
{"sessionUpdate": "background_wake", "tasks": [{"id": "bg_3", "kind": "command", "label": "make test", "status": "failed", "exitCode": 2, "durationMs": 90000, "error": "exit status 2"}]}
```

仅渲染标准更新的编辑器会显示没人要求的答案，因此 `coddy acp` 会在更新之后，以答案开头的引用 `agent_message_chunk` 的相同唤醒形式跟随，实时和在 `session/load` 上：

```text
> Woken by a finished background task: bg_3 make test, failed, exit 2, 1m 30s
```

`coddy acp --remote` 会在 `session/load` 上以相同方式重放服务器的唤醒轮次；它不会实时跟随服务器的唤醒轮次。

### `session_info_update` - 会话被重命名

每当会话列出的标题变化时发送：其第一条消息派生的名称、智能体用 `session_describe` 固定的名称、操作员在另一个界面上进行的重命名。与智能体保持会话列表同步的客户端会随之一并重命名，而不是显示创建时的名称。

```json
{
  "sessionUpdate": "session_info_update",
  "title": "Ship the release",
  "updatedAt": "2026-10-24T09:12:33Z"
}
```

`title` 始终在线上 - 重命名携带新名称，而不再有人命名的会话携带 `null`，这会清除名称而不是保留它。`updatedAt` 是智能体宣布更改的时间。

会话的首次公告会在客户端注册它之后、在 `session/new`、`session/load` 或 `session/resume` 响应之后发出：该窗口内的重命名涉及客户端尚未见过的会话，会与其他内容一起发布，而不是提前发布。

### `current_mode_update` - 模式已更改

```json
{
  "sessionUpdate": "current_mode_update",
  "currentModeId": "agent"
}
```

### `config_option_update` - 会话配置选项已更改

在 `session/set_config_option` 之后、在 `session/set_mode` 之后，或每当智能体更新会话配置选项以匹配运行时状态时发送（以便 UI 保持对齐）。

```json
{
  "sessionUpdate": "config_option_update",
  "configOptions": [
    {
      "id": "mode",
      "name": "Session mode",
      "category": "mode",
      "type": "select",
      "currentValue": "agent",
      "options": [ ... ]
    },
    {
      "id": "model",
      "name": "Model",
      "category": "model",
      "type": "select",
      "currentValue": "openai/gpt-5.6-terra",
      "options": [ ... ]
    },
    {
      "id": "permission_mode",
      "name": "Permission mode",
      "category": "_permission_mode",
      "type": "select",
      "currentValue": "accept_edits",
      "options": [ ... ]
    }
  ]
}
```

## 权限请求（智能体 -> 客户端，期望响应）

当 `permission_mode` 为 `ask`（命令和写入）或 `accept_edits`（命令，以及工作区之外或进入 `.git` 文件夹的写入）时发送这些请求。在 `bypass` 下，智能体仅对没有任何模式能自行应答的提示词发送 `session/request_permission`：`PreToolUse` 钩子的 `ask`，以及 [决策检查](../features/decisions.md) 未放行的 shell 命令。通过 `session/set_config_option` 或在提示词中发送 `/permissions` 命令来设置模式；新会话以最后选择的模式启动。

在 `ask` 下，属于会话本身的请求会在“拒绝”之前多出两个类型为 `allow_always` 的选项：`allow_session_bypass`（**本次会话绕过**）以及，对于文件写入，`allow_session_accept_edits`（**允许本次会话编辑**）。选择其中之一会批准该调用并切换会话的权限模式，智能体会通过 `config_option_update` 宣布。从子智能体中继的请求、由钩子强制的请求以及 `config_commit` / `config_rollback` 不携带这些选项。

```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "session/request_permission",
  "params": {
    "sessionId": "sess_abc123def456",
    "toolCall": {
      "toolCallId": "call_002",
      "title": "Run: go build ./...",
      "kind": "execute",
      "status": "pending",
      "content": [
        { "type": "text", "text": "Execute: go build ./..." }
      ]
    },
    "options": [
      { "optionId": "allow", "name": "Allow", "kind": "allow_once" },
      { "optionId": "allow_always", "name": "Allow always", "kind": "allow_always" },
      { "optionId": "reject", "name": "Reject", "kind": "reject_once" }
    ]
  }
}
```

**响应：** 协议将结果嵌套在其自己的对象中，Zed 等编辑器发送的就是这种形式：

```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "result": {
    "outcome": { "outcome": "selected", "optionId": "allow" }
  }
}
```

被关闭的请求会以 `{ "outcome": { "outcome": "cancelled" } }` 应答。Coddy 也接受其自身界面和某些编辑器扩展发送的扁平形式（`{"outcome": "selected", "optionId": "allow"}`），并以相同方式读取两者：除非结果为 `cancelled` 或所选 `optionId` 为 `reject`，否则调用继续。选择 `allow_always`（或程序范围的 `allow_always_<program>` 选项）还会存储会话授权，因此同一命令不会再次询问。

## Elicitation（智能体 -> 客户端，期望响应）

由 **`question`** 工具使用。与权限请求相同的入站 JSON-RPC 模式（客户端必须用相同的 **`id`** 回复）。

通告了 `clientCapabilities.elicitation.form` 的客户端会通过 **`elicitation/create`** 以标准方式被询问，此类客户端会将其渲染为表单。从未声明这一点的客户端会继续收到下面描述的 Coddy 专属 **`session/request_question`**，因此不会向无法渲染表单的客户端发送表单。

```json
{
  "jsonrpc": "2.0",
  "id": 11,
  "method": "elicitation/create",
  "params": {
    "sessionId": "sess_abc123def456",
    "toolCallId": "call_003",
    "message": "Pick a stack",
    "mode": "form",
    "requestedSchema": {
      "type": "object",
      "properties": {
        "q1": {
          "type": "string",
          "title": "Pick a stack",
          "enum": ["Go", "Rust"]
        }
      },
      "required": ["q1"]
    }
  }
}
```

每个问题一个表单字段，按工具提问的顺序命名为 `q1`、`q2` 等。问题达到十个或更多时数字会补零（`q01`、`q02`），这样按 JSON 对象顺序渲染字段的客户端仍会按提问顺序显示它们。问题如何成为字段：

| 问题 | 字段 |
| --- | --- |
| 有选项，且答案必须是其中之一（`custom: false`） | 一个选择器：没有选项描述自身时为 `enum`，有选项描述自身时为 `oneOf` |
| 有选项，可选择多个（`multiple: true`） | 一个多选 `array`，其 `items` 携带相同选项；当用户也可以用自己的话回答时（`custom`，默认），旁边会有一个以其命名并添加了 `_other` 的可选文本字段（`q1_other`），且两个字段都不是必填 |
| 有选项，且用户可以用自己的话回答（`custom`，默认） | 自由文本，选项标签列在字段描述中作为建议 |

表单无法在同一字段上同时提供选择器和脱离选择器的出口，因此允许用户自行回答的问题会变为自由文本，而不是封闭选择器。这保留了模型依赖的契约：**与任何选项标签都不匹配的答案就是用户输入的文本**，用户写的任何内容都不会因为不是提供的标签之一而被拒绝。

**响应：**
```json
{
  "jsonrpc": "2.0",
  "id": 11,
  "result": {
    "action": "accept",
    "content": { "q1": "Rust" }
  }
}
```

`action` 为 `accept`、`decline` 或 `cancel`。在 `accept` 上，`content` 的值成为答案 - 单值字段一个条目，多选每个选择一个条目，如果用户输入了内容，则随后是其 `_other` 字段的文本。Coddy 会按 ACP 对智能体的要求再次检查这些值。必须用其某个选项回答的问题只保留它提供的标签，无论客户端放行了什么。被拒绝或取消的表单会以没有任何选择的方式应答，工具会将其报告给模型为未回答的问题，而不是失败。

### `session/request_question` - 回退方案

Coddy 自己的问题请求，保留给未实现 elicitation 的客户端。负载是工具提问时的问题，答案是对每个问题的一个选择数组，其中与任何选项标签都不匹配的条目是用户输入的文本。

```json
{
  "jsonrpc": "2.0",
  "id": 11,
  "method": "session/request_question",
  "params": {
    "sessionId": "sess_abc123def456",
    "requestId": "q_1730000000000",
    "toolCallId": "call_003",
    "questions": [
      {
        "question": "Pick a stack",
        "options": [{ "label": "Go" }, { "label": "Rust" }]
      }
    ]
  }
}
```

**响应：**
```json
{
  "jsonrpc": "2.0",
  "id": 11,
  "result": {
    "answers": [["Go"]]
  }
}
```

## 客户端文件系统方法

智能体可以在客户端上调用这些方法（如果客户端支持）：

### `fs/read_text_file`

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "fs/read_text_file",
  "params": { "path": "/absolute/path/to/file.go" }
}
```

### `fs/write_text_file`

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "fs/write_text_file",
  "params": {
    "path": "/absolute/path/to/file.go",
    "content": "package main\n..."
  }
}
```

<!-- docsgen:source sha256=d2d59d1f3355d63b -->

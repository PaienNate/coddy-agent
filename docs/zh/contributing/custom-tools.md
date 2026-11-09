# 自定义工具指南

## 概述

智能体使用一个**工具注册表**（`internal/tools`）向 LLM 暴露各项能力。智能体能执行的每一个动作——读取文件、运行命令、搜索代码——都以工具的形式实现。本指南说明如何将新的内置工具直接添加到智能体二进制中。

如果你希望在不修改智能体源码的情况下添加工具，请改用 [MCP 服务器](../features/mcp.md)。

---

## 工具系统的工作方式

```
User prompt
    │
    ▼
ReAct loop (internal/agent/react.go)
    │
    ├── tools.NewRegistryFor(cfg) registers builtins (internal/tools/export.go)
    │
    ├── Registry.AllToolDefinitions()
    │
    ├── FilterToolDefinitions(..., ToolSetForMode(mode))  ← internal/agent/toolsets.go
    │
    ├── (agent or plan mode, never ask) append MCP tool definitions
    │
    ├── narrow to the session's toolset (its /toolset pick, else models[].toolset)  ← internal/agent/toolsets.go
    │
    ├── passes tool definitions to LLM via provider.Stream()
    │
    └── on tool_call in LLM response:
            │
            ├── permission check (ACP)
            │
            └── registry.Execute(name, argsJSON, env)  →  tool.Execute()
```

LLM 会收到一份工具**定义**列表（名称、描述、JSON Schema）。当 LLM 决定调用某个工具时，智能体会执行它，并将结果回填到对话中。

---

## `Tool` 结构体

每个工具都是 `tools.Tool` 类型的值，它是 `tooling.Tool` 的类型别名，定义于
[`internal/tooling/tool.go`](../../../internal/tooling/tool.go)：

```go
type Tool struct {
    Definition llm.ToolDefinition

    // RequiresPermission indicates the tool needs user approval before running.
    RequiresPermission bool

    Execute func(ctx context.Context, argsJSON string, env *Env) (string, error)
}
```

### `llm.ToolDefinition`

这是 LLM 所看到的内容：

```go
type ToolDefinition struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    InputSchema interface{} `json:"input_schema"` // JSON Schema object
}
```

- `Name` - 唯一的 snake_case 标识符，例如 `fetch_url`、`git_log`
- `Description` - 纯英文描述；LLM 据此决定何时调用该工具
- `InputSchema` - 标准的 [JSON Schema](https://json-schema.org/)，描述工具的各个参数

### `tools.Env`

`Env` 结构体会被传入每一次 `Execute` 调用，并提供会话上下文：

```go
type Env struct {
    CWD              string   // session working directory
    PermissionMode   string   // "ask", "accept_edits", or "bypass"
    CommandAllowlist []string // commands that skip permission checks

    // Plan/todo support (always populated by the ReAct agent):
    SessionID string                      // current session ID
    Sender    acp.UpdateSender            // sends session/update to connected ACP client
    GetPlan   func() []acp.PlanEntry      // read current todo list
    SetPlan   func([]acp.PlanEntry)       // replace todo list
}
```

以 `env.CWD` 作为相对路径的基准。使用 `resolvePath(path, env.CWD)`（包内私有
辅助函数）将用户提供的路径转换为绝对路径。

如果你的工具希望将计划状态推送到客户端，请调用 `sendPlanUpdate(env, entries)` ——这是
`internal/tools/todo.go` 中的一个包内私有辅助函数，它在发送前会对 `Sender` 做空值检查。

---

## 分步指南：创建一个内置工具

### 1. 创建工具构造函数

在 `internal/tools/` 中新增一个 `.go` 文件（若主题相关，也可添加到已有文件中）。

```go
package tools

import (
    "context"
    "fmt"
    "net/http"

    "github.com/EvilFreelancer/coddy-agent/internal/llm"
)

// fetchURLTool returns a tool that performs an HTTP GET request.
func fetchURLTool() *Tool {
    return &Tool{
        Definition: llm.ToolDefinition{
            Name:        "fetch_url",
            Description: "Perform an HTTP GET request and return the response body as text.",
            InputSchema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "url": map[string]interface{}{
                        "type":        "string",
                        "description": "Full URL to fetch, e.g. https://example.com/api/data",
                    },
                },
                "required": []string{"url"},
            },
        },
        RequiresPermission: false,
        Execute:            executeFetchURL,
    }
}
```

### 2. 定义参数结构体

```go
type fetchURLArgs struct {
    URL string `json:"url"`
}
```

### 3. 实现 `Execute` 函数

```go
func executeFetchURL(ctx context.Context, argsJSON string, _ *Env) (string, error) {
    args, err := parseArgs[fetchURLArgs](argsJSON)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, args.URL, nil)
    if err != nil {
        return "", fmt.Errorf("fetch_url: invalid request: %w", err)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("fetch_url: %w", err)
    }
    defer resp.Body.Close()

    body := make([]byte, 0, 4096)
    buf := make([]byte, 512)
    for {
        n, readErr := resp.Body.Read(buf)
        body = append(body, buf[:n]...)
        if readErr != nil {
            break
        }
    }

    return fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(body)), nil
}
```

`Execute` 的关键规则：
- 对真正的错误（参数错误、IO 失败等）返回 `("", error)`
- 当命令执行了但产生了非成功结果时返回 `(errorMessage, nil)` ——这样
  LLM 能看到失败并决定下一步怎么做
- 对取消和超时使用 `context.Context`
- 使用 `parseArgs[T]` 将 JSON 参数反序列化为带类型的结构体

### 4. 在 `NewRegistryFor()` 中注册该工具

打开 `internal/tools/export.go`，在 `NewRegistryFor()` 内添加 `r.Register(...)`（如果你是在扩展文件系统工具包，也可以从 `internal/tools/fs/register.go` 这样的辅助函数中注册）。

```go
func NewRegistryFor(cfg *config.Config) *Registry {
    r := tooling.NewRegistry()
    // ...
    r.Register(fetchURLTool()) // <-- add here
    registerSchedulerTools(r, cfg)
    return r
}
```

就这么简单。重新构建（`go build ./...`）后，LLM 就能看到新工具。

---

## 工具字段参考

### 计划模式与智能体模式的暴露差异

针对模式的可见性**并非**在 `Tool` 结构体上配置。相反，`internal/agent/toolsets.go`
定义了一个 `ToolSet` 允许列表：

- **`agent`** 模式使用**空**的 `ToolSet`，即**不做任何过滤**——注册表中的每个工具都会被公开，并且 MCP 工具会被追加在后面。
- **`plan`** 模式会将**注册表**中的内置工具过滤到**固定的允许列表**（`read_file`、`list_dir`、`search_files`、`websearch`、`webfetch`、`run_command`）。其他内置工具（写入、todo 工具、调度器、记忆等）仍会保留注册以维持执行一致性，但会**对 LLM 隐藏**。来自已连接服务器的 **MCP** 工具定义在该过滤之后**仍会被追加**，接线方式与智能体模式相同。

当你新增一个应当对计划模式安全的内置工具时，请将它的名字追加到 `internal/agent/toolsets.go` 中的 `planToolNames`，并扩展 `internal/agent/toolsets_test.go` 中的测试。

### `RequiresPermission`

当 `true` 时，ReAct 循环会在调用工具之前暂停，并向 ACP 客户端发送一条
`session/request_permission` 通知。用户必须先批准该调用，执行才会继续。

对以下情况使用 `RequiresPermission: true`：
- 执行外部进程或 shell 命令的工具
- 在明显只读流程之外删除或不可逆地修改数据的工具

网络工具在其实现自带防护措施时（例如 `webfetch` 上的 SSRF 检查与响应大小限制），仍可使用 `RequiresPermission: false`。对于请求由模型自由构造的网络工具，例如 `http_request`，应设置 `RequiresPermission: true`，并在审批关卡（`permission.HTTPRequestAllowedWithSession`）中为其设立独立分支，由该分支决定目标地址以及请求携带的内容。

对以下情况使用 `RequiresPermission: false`：
- 只读操作
- 工作目录内的写入（可由 `env.RequirePermissionForWrites` 控制）

---

## `InputSchema` 的 JSON Schema

`InputSchema` 是一个标准的 JSON Schema 对象。最常见的模式：

### 必填字符串字段

```go
"url": map[string]interface{}{
    "type":        "string",
    "description": "URL to fetch",
},
```

### 带默认值的可选整数字段

```go
"timeout_seconds": map[string]interface{}{
    "type":        "integer",
    "description": "Timeout in seconds (default: 30)",
},
```

### 枚举字段

```go
"method": map[string]interface{}{
    "type":        "string",
    "enum":        []string{"GET", "POST", "PUT", "DELETE"},
    "description": "HTTP method",
},
```

### 布尔字段

```go
"follow_redirects": map[string]interface{}{
    "type":        "boolean",
    "description": "Follow HTTP redirects (default: true)",
},
```

### 标记必填字段

```go
"required": []string{"url", "method"},
```

---

## 完整示例：`git_log` 工具

下面是一个完整、可用于生产环境的示例，该工具在工作目录中运行 `git log`
并返回格式化后的摘要。

**`internal/tools/git.go`：**

```go
package tools

import (
    "bytes"
    "context"
    "fmt"
    "os/exec"

    "github.com/EvilFreelancer/coddy-agent/internal/llm"
)

func gitLogTool() *Tool {
    return &Tool{
        Definition: llm.ToolDefinition{
            Name:        "git_log",
            Description: "Show recent git commit history for the working directory repository.",
            InputSchema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "limit": map[string]interface{}{
                        "type":        "integer",
                        "description": "Number of commits to show (default: 10)",
                    },
                    "branch": map[string]interface{}{
                        "type":        "string",
                        "description": "Branch name (default: current branch)",
                    },
                },
            },
        },
        RequiresPermission: false,
        Execute:            executeGitLog,
    }
}

type gitLogArgs struct {
    Limit  int    `json:"limit"`
    Branch string `json:"branch"`
}

func executeGitLog(ctx context.Context, argsJSON string, env *Env) (string, error) {
    args, err := parseArgs[gitLogArgs](argsJSON)
    if err != nil {
        return "", err
    }

    limit := 10
    if args.Limit > 0 {
        limit = args.Limit
    }

    cmdArgs := []string{
        "log",
        fmt.Sprintf("--max-count=%d", limit),
        "--oneline",
        "--decorate",
    }
    if args.Branch != "" {
        cmdArgs = append(cmdArgs, args.Branch)
    }

    cmd := exec.CommandContext(ctx, "git", cmdArgs...)
    cmd.Dir = env.CWD

    var out bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &out

    if err := cmd.Run(); err != nil {
        return fmt.Sprintf("git log failed: %v\n%s", err, out.String()), nil
    }

    result := out.String()
    if result == "" {
        return "(no commits)", nil
    }
    return result, nil
}
```

然后在 `export.go` 中：

```go
r.Register(gitLogTool())
```

---

## 检查清单

在提交新工具之前，请确认：

- [ ] 工具名称唯一，并遵循 `snake_case` 约定
- [ ] 描述清楚地说明了该工具的作用以及何时使用它
- [ ] 所有必填字段都列在 `InputSchema.required` 中
- [ ] 可选字段具有合理的默认值，并在描述中注明
- [ ] 如果该工具必须在**计划**模式中可见，请将其名字添加到 `internal/agent/toolsets.go` 中的 `planToolNames`，并扩展 `internal/agent/toolsets_test.go`
- [ ] 对于具有破坏性的 shell 命令或敏感写入，将 `RequiresPermission` 设置为 `true`
- [ ] `Execute` 在参数解析失败时返回 `("", error)`
- [ ] `Execute` 在运行时失败时返回 `(errorMessage, nil)`，以便 LLM 能看到这些失败
- [ ] 工具构造函数已在 `NewRegistry()` 中注册
- [ ] `go build ./...` 和 `go test ./...` 均通过

---

## 内置计划 / Todo 工具

七个 **`coddy_`** 工具（`internal/tools/todo`）驱动检查清单。计划变更会以
**`plan`** 载荷发出 `session/update`。它们会为每个会话注册，但**只在 `agent` 模式下对 LLM 公开**（计划模式通过 `ToolSet` 隐藏它们）。

| 工具名称 | 用途 |
|-----------|---------|
| **`coddy_todo_plan_read`** | 返回当前活动计划的 markdown 检查清单渲染结果（`{}`）；当没有活动计划时，返回一行注明 **`coddy_todo_plan_replace`** 的说明，而不是空字符串。 |
| **`coddy_todo_plan_replace`** | 替换整个计划（`markdown`）。只要有任何一行未完成就会被拒绝，除非你先执行 **`coddy_todo_plan_archive`**。已完成的列表在替换前会归档 **`todos/active.md`**。 |
| **`coddy_todo_plan_archive`** | 将所有未完成的行标记为 **`completed`**，当 **`SessionDir`** 时写入 **`todos/archive/plan_<unix_seconds>.md`**，清空内存中的计划，发出空的 **`plan`**。 |
| **`coddy_todo_item_add`** | 追加或插入（`content`，可选 `status`，可选 `after_index`；`-1` 表示前插）。 |
| **`coddy_todo_item_remove`** | 删除一行（`index`）。 |
| **`coddy_todo_item_update`** | 变更（`index` 加上 **`content`** 和/或 **`status`**）。状态枚举为 `pending`、`in_progress`、`completed`、`failed`、`cancelled`。 |
| **`coddy_todo_item_move`** | 重排序（`from_index`、`to_index`；索引按删除语义之后应用）。 |

整体替换示例：

```json
{
  "markdown": "- [ ] Read existing code\n- [ ] Write tests\n- [ ] Implement feature\n- [ ] Update docs"
}
```

状态推进示例：

```json
{ "index": 0, "status": "in_progress" }
```

支持 ACP 的编辑器会将计划状态映射到各自的界面。Markdown 标记大致对应如下：

| 符号 | 状态 |
|--------|--------|
| `[ ]` | pending |
| `[>]` | in_progress |
| `[x]` | completed |
| `[!]` | failed |
| `[-]` | cancelled |

---

## 替代方案：MCP 服务器

如果你希望在不修改智能体源码的情况下添加工具（例如针对特定项目的工具、
第三方集成，或用其他语言编写的工具），请使用
[MCP 服务器集成](../features/mcp.md)。

MCP 工具在运行时以 `serverName__toolName` 为前缀注册，并遵循与内置工具相同的
`RequiresPermission`/计划允许列表模型。

<!-- docsgen:source sha256=a75d34d45479529e -->

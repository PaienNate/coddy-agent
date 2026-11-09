# OpenCode

`type: opencode` 提供商为 Coddy 提供 [OpenCode](https://opencode.ai) 网关的模型，即 **Go** 订阅或按量付费的 **Console**（Zen），可使用账户登录或服务账户密钥。这里 OpenCode 只是后端：智能体仍是 Coddy 自己的，带着它的系统提示词、工具、权限门禁和 ReAct 循环，在每个界面上都是如此，包括控制台、Web 界面、ACP 编辑器和 Telegram 网关。

OpenCode 维护一个编码模型目录（Grok、GPT、Claude、Qwen、GLM、Kimi、DeepSeek、MiniMax、MiMo、Gemini 等），并通过适合每个模型的线协议来提供它。Coddy 按模型和网关选择该协议。Console 的 Jev 决策模型使用单独的 System One API，对 Coddy 来说不是对话模型。

## 使用账户登录

在运行 Coddy 的机器上的终端中运行登录命令。

```bash
coddy providers login opencode
```

Coddy 会打印一个短代码并打开 OpenCode Console 账户页面。登录、确认该代码并选择你的组织。你可以在另一台机器的浏览器中打开打印出的 URL。这遵循 [OpenCode 客户端的设备方法](https://github.com/anomalyco/opencode/blob/f394d9d410bc6bb7cafcebfc5e3180a9b80ebed9/packages/core/src/plugin/provider/opencode.ts)。

在控制台中，`/login opencode` 运行相同的登录，并将提供商行和网关的模型写入 `config.yaml`（[Console](../surfaces/console.md#登录提供商-login)）。

登录会将访问令牌和刷新令牌存储在 `$CODDY_HOME/providers/opencode/opencode-auth.json`，模式为 `0600`。除非给出 `--no-config`，它还会将提供商行、网关的模型和一个 `agent.model` 添加到 `config.yaml`；最后一项会在其为空或运行在无法应答的提供商上时添加。使用 `--type opencode` 登录另一个命名配置文件，或使用 `--api-base https://opencode.ai/inference/v1` 将 Console 而非 Go 写入该行。使用 `--no-config` 时该行保持原样，因此在那里 `--api-base` 会被拒绝。

在每次目录或推理请求之前，Coddy 会刷新将在 60 秒内过期的令牌，并以原子方式保存轮换后的令牌。请求在 `x-org-id` 中携带所选组织。被撤销的刷新令牌需要重新登录。`coddy providers list` 会显示账户组织，而不暴露令牌。存储的凭据在每个 Coddy 界面上都有效。正在运行的服务器会为已加载的提供商行读取它；添加新行后请重启服务器。

显式的 `api_key`、`api_key_command` 或该行的 `NAME_API_KEY` 仍优先于存储的登录。`coddy providers logout opencode` 会删除本地文件。在 Console 中撤销账户会话或服务账户密钥，以在服务器上结束它。

设备授权、轮询和令牌刷新使用 `client_id: coddy` 标识此应用程序。

### 从设置登录

在 **设置 → LLM 提供商** 中，打开 OpenCode 行，并点击网关选择器下方的 **使用 OpenCode 登录**。Coddy 会在表单中显示设备代码，并在那里等待。**打开登录页面** 会打开 Console，其中会为你正在批准的设备显示相同的代码，请在批准前比较两者。该表单绝不会自行打开那个页面，因此当 Console 要求输入代码时，代码就在屏幕上。然后选择你的组织。确认后表单会显示组织；**退出登录** 会移除本地登录，并取消待处理的尝试。

该登录使用表单中当前显示的代理，包括新行或保存前切换到 OpenCode 的已保存行。它服务于两个网关，因此仅由网关选择器决定该行的请求发往何处。点击 **全部保存** 以将这些提供商设置保留给聊天和未来的令牌刷新。使用模型列表旁的刷新图标，以存储的登录读取目录。令牌保留在服务器上，绝不会出现在浏览器或配置 YAML 中。

## 使用服务账户密钥

1. 登录 [OpenCode Console](https://opencode.ai/console)。
2. 订阅 **Go**（或 **Go Plus**），或为 **Console** 添加额度，并添加你的账单信息。
3. 创建服务账户密钥，并将其设置到提供商行上。

```yaml
# yaml-language-server: $schema=https://coddy.dev/config.schema.json
providers:
  - name: opencode
    type: opencode
    api_key: "${OPENCODE_API_KEY}"   # or a literal key, or empty to read OPENCODE_API_KEY
    # api_base: https://opencode.ai/inference/v1   # Console; omit for the Go default
models:
  - model: opencode/glm-5.3
  - model: opencode/gpt-6-luna
  - model: opencode/claude-haiku-5-5
agent:
  model: opencode/glm-5.3
```

空的 `api_key` 会读取该行的 `NAME_API_KEY` 环境变量，然后在该变量未设置时使用存储的凭据。名为 `opencode` 的行会读取 `OPENCODE_API_KEY`。`${OPENCODE_API_KEY}` 形式会在文件加载时展开；裸空值会在调用时读取。

### 存储服务账户密钥

```bash
coddy providers login opencode --api-key -
```

`-` 从秘密终端提示或标准输入中的一行读取密钥。`--api-key KEY` 直接提供它。该密钥使用与账户登录相同的凭据文件和配置更新。先前存储的 API 密钥仍受支持。

## 两个网关

`api_base` 选择推理网关。账户登录对两个网关使用相同的 Console 设备授权服务；访问权限取决于所选组织及其订阅或额度。存储的登录属于账户，而不属于网关，正如在 OpenCode 客户端中，Go 和 Console 是同一个 `OPENCODE_API_KEY` 上的两个提供商：仅由该行的 `api_base` 决定其请求到达哪个网关，而一次登录可同时服务于 Go 行和 Console 行。

| 网关 | `api_base` | 说明 |
|---------|-----------|------------|
| **Go**（默认） | `https://opencode.ai/inference/go/v1` | Go / Go Plus 月度订阅 |
| **Console** | `https://opencode.ai/inference/v1` | 按量付费额度（Zen） |

空的 `api_base` 表示 Go。既不指向任一网关的值会回退到 Go，因此 OpenCode 密钥绝不会被发送到任意主机。在 Web 界面的 **设置 → LLM 提供商** 中，该行显示网关选择器，而不是自由文本 base。`CODDY_OPENCODE_BASE_URL` 会将整个流程移到另一个地址，用于测试台或测试。

## 每个模型的线协议

OpenCode 网关通过不同协议提供不同模型，其 `/models` 列表不提供任何提示，只有 id。Coddy 根据模型 id 进行选择：

| 协议 | 端点 | 模型系列 |
|----------|----------|----------------|
| OpenAI Responses | `/openai/v1/responses` | `grok-*`、`gpt-*`、`muse-*` |
| Anthropic | `/anthropic/v1/messages` | `claude-*`、`qwen*`、`minimax-m3`（在 Go 上） |
| Google | `/google/v1beta/models/<id>:generateContent` | `gemini-*` |
| OpenAI 兼容 | `/openai/v1/chat/completions` | `glm-*`、`kimi-*`、`deepseek-*`、`longcat-*`、`mimo-*`、`hy*`、`space-bunny`、免费聊天模型和未知 id |

这些路径在 Go 上遵循 `https://opencode.ai/inference/go`，在 Console 上遵循 `https://opencode.ai/inference`。目录仍位于该前缀下的 `/v1/models`。Console 通过 Chat Completions 提供 `qwen3.8-max` 和每个 `minimax-*` 模型；Go 通过 Anthropic 提供 `qwen3.8-max` 和 `minimax-m3`。较旧的 MiniMax 模型使用 Chat Completions。

此表不认识的 id 会回退到 Chat Completions。该映射遵循 [OpenCode Go 文档](https://opencode.ai/v2/docs/console/go/) 和 [Console 模型端点](https://opencode.ai/v2/docs/console/models/)；当 OpenCode 更改某个模型的协议时，请更新 `opencodeProtocolForGateway`，它位于 `internal/llm/opencode.go` 中。

推理级别按模型协议以 OpenAI Responses 的 `reasoning.effort`、Anthropic 思考预算、OpenAI 兼容的 `reasoning_effort` 或 Google 的 `thinkingConfig.thinkingLevel` 发送。Gemini 通过 `:streamGenerateContent?alt=sse` 进行流式传输；Coddy 在工具调用之间保留其已签名的响应部分，并在流在结束原因之前被截断时保留部分文本。支持内联图片；远程图片 URL 会作为附件命名，而不会抓取它们。

## 必需的客户端标头

该网关受到滥用监控，并要求每个客户端表明自身身份，并将一次对话保持在一条路由上。Coddy 在所有四种协议的推理和目录请求中发送这些标头：

- `User-Agent: coddy/<version>`，它自己的名称，而不是通用 SDK 或 HTTP 库的名称；
- `x-opencode-session: <session id>`，对话的稳定 id，以便 OpenCode 优化路由并复用其提示词缓存。不是对话的调用（模型列表、`--dry-run` 探测）会发送生成的 id。

Coddy 通过该行自己的传输将两个标头注入推理和目录请求。账户请求还使用其 `User-Agent` 标识 Coddy。

只会发送此行的 OpenCode 凭据。诸如 `OPENAI_API_KEY` 和 `ANTHROPIC_API_KEY` 的 SDK 默认值会被丢弃，并且在转发密钥之前会拒绝重定向到另一个源。

## 代理

`providers[].proxy` 会按与其他任何提供商完全相同的方式路由每个 OpenCode 请求，包括设备授权、账户和组织发现、令牌刷新、所有四种协议上的补全以及模型列表：`inherit` 遵循环境的代理，`none` 直接连接，URL 则通过该代理。参见 [提供商代理](../getting-started/configuration.md#提供商代理)。

## 使用限制

OpenCode 按每月美元预算（带有 5 小时和每周子限额）计量每个模型，你可以在 [控制台](https://opencode.ai/console) 中查看。Coddy 没有 OpenCode 账户用量读取器，因此不会为 `opencode` 行显示用量面板。模型的上下文窗口使用 `max_context_tokens`，然后使用可用的目录元数据，最后使用 128000。

## 测试

- 单元测试：`internal/llm/opencode_test.go` 检查两个网关的精确推理路径、标头、凭据隔离、重定向拒绝和存储的凭据。`opencode_google_test.go` 覆盖 Gemini 工具对话、已签名部分、用量和截断的流。
- `features/provider_proxy.feature` 覆盖所有四种 API 系列、模型发现、账户登录、令牌刷新以及通过每个代理设置进行的登录配置更新。`opencode_oauth_test.go` 验证 HTTPS CONNECT 路由、令牌轮换、并发刷新和授权失败；`cmd/coddy/opencode_test.go` 检查终端流程。
- Responses 传输与 Codex 后端（`internal/llm/responses_stream.go`）共享；`internal/llm/codex_test.go` 覆盖它。

### 设置截图

这些截图显示了连接到本地 Console 测试台的运行中的界面。

| 状态 | `1280px` | `390px` |
| --- | --- | --- |
| 之前 | ![之前 - 宽屏](../../assets/opencode-login/before-wide.jpg) | ![之前 - 窄屏](../../assets/opencode-login/before-narrow.jpg) |
| 准备登录 | ![准备登录 - 宽屏](../../assets/opencode-login/after-idle-wide.jpg) | ![准备登录 - 窄屏](../../assets/opencode-login/after-idle-narrow.jpg) |
| 等待确认 | ![等待确认 - 宽屏](../../assets/opencode-login/after-pending-wide.jpg) | ![等待确认 - 窄屏](../../assets/opencode-login/after-pending-narrow.jpg) |
| 已登录 | ![已登录 - 宽屏](../../assets/opencode-login/after-connected-wide.jpg) | ![已登录 - 窄屏](../../assets/opencode-login/after-connected-narrow.jpg) |

## 另请参阅

- [配置](../getting-started/configuration.md) - 提供商、模型和凭据顺序。
- [环境变量](../reference/environment-variables.md) - `OPENCODE_API_KEY`、`CODDY_OPENCODE_BASE_URL`、`CODDY_OPENCODE_CONSOLE_URL`。
- [配置参考](../reference/config.md) - `providers[]` 和 `models[]` 字段。

<!-- docsgen:source sha256=72c2542cfef0871e -->

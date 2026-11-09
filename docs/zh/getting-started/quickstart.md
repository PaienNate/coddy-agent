# 快速开始

本页把一台全新的机器带到第一个回答：安装二进制、给它配一个模型、检查文件，然后在终端、浏览器和编辑器中与智能体对话。其余每个页面都假设这些步骤已经完成。预计花费五分钟和一个 API 密钥。

## 1. 安装

每个操作系统一行； [安装指南](install.md) 中有 Linux 软件包、Homebrew cask、安装器选项和手动放置方式。

**Linux / macOS**

```bash
curl -fsSL https://coddy.dev/install.sh | bash
```

**Windows (PowerShell)**

```powershell
irm https://coddy.dev/install.ps1 | iex
```

安装器把 `coddy` 放到 `PATH`(Unix 上是 `~/.local/bin`,Windows 上是 `%LOCALAPPDATA%\Programs\coddy`)，并在文件缺失时从发布版本的 `config.example.yaml` 创建 `~/.coddy/config.yaml`。运行安装器的那个终端还没有重新读取自己的配置文件，所以请新开一个终端并检查：

```bash
coddy -v
```

Debian、Ubuntu、Fedora 及其衍生发行版可以改用 `.deb` 或 `.rpm`，容器则以 `coddy serve` 运行同一个二进制([Docker](docker.md))。如果找不到该命令，请参阅 [故障排查](troubleshooting.md#安装后找不到二进制文件)。

## 2. 给它配一个模型

最短的路径既不需要密钥也不需要编辑文件：运行 `coddy` 并输入 `/login`。单独一个 `/login` 会登录 NeuralDeep,`/login codex`、`/login devin` 和 `/login opencode` 则登录其余各家；控制台会打印页面和在浏览器中确认的代码，然后把该提供商及其提供的每个模型写入 `config.yaml` 并基于它们运行([从控制台登录](../surfaces/console.md#登录提供商-login))。本节剩下的内容讲的是手动配置提供商。

Coddy 需要知道的一切都存放在 `~/.coddy/config.yaml`(Windows 上是 `%USERPROFILE%\.coddy\config.yaml`)中。安装器创建的文件是 `config.example.yaml` 的副本，每个章节都在注释中作了解释；对于一个 OpenAI 兼容提供商，最少需要三个块：

```yaml
providers:
  - name: openai
    type: openai
    api_key: "${OPENAI_API_KEY}"

models:
  - model: "openai/gpt-5.6-terra"
    max_tokens: 8192
    reasoning_default: medium

agent:
  model: "openai/gpt-5.6-terra"
```

有三条规则把这些块串在一起。提供商的 `name` 会成为每个模型 id 的前缀，所以 `models[].model` 就是 `<provider name>/<model id as the API knows it>`。`agent.model` 必须是 `models` 条目之一。而 `api_key` 要么是密钥本身，要么是文件加载时展开的 `${VAR}` 引用，要么留空，此时会在调用时读取 `OPENAI_API_KEY`(提供商名称大写加上 `_API_KEY`)。

把密钥放到引用能找到的位置：shell 中，或 `~/.coddy/.env` 中，后者在文件被解析前读取，并且永远不会覆盖 shell 已设置的变量：

```bash
export OPENAI_API_KEY="sk-..."
# or, once and for all:
echo 'OPENAI_API_KEY=sk-...' >> ~/.coddy/.env
```

任何 OpenAI 兼容服务器都能配合 `type: openai` 和一个包含 `/v1` 的 `api_base` 使用；本地的 Ollama 或 llama.cpp 不需要真实密钥：

```yaml
providers:
  - name: local
    type: openai
    api_base: "http://localhost:11434/v1"
    api_key: "~"
```

Anthropic、NeuralDeep(用 `coddy providers login neuraldeep`，或在控制台用 `/login` 代替密钥)和 Codex(ChatGPT OAuth)在 [配置](configuration.md) 中有介绍；Devin 账号用 `coddy providers login devin` 登录([Devin](../features/devin.md)),OpenCode Go 或 Console 账号用 `coddy providers login opencode` 登录([OpenCode](../features/opencode.md))；在控制台中 `/login devin` 和 `/login opencode` 起相同作用。

## 3. 检查文件

有两个标志能在一切开始之前告诉你文件是否正确，每个加载该文件的命令都接受它们：

```bash
coddy -t          # the file against the schema and the loader's rules
coddy --dry-run   # the same, then the world it points at
```

`-t` 会打印 `<path>: valid`，或把每个问题打印为 `file:line:column: what is wrong` 并在其下附一行 `fix:`，有错误时以退出码 1 结束。`--dry-run` 会继续向每个提供商请求其模型列表，这一次请求就同时检验了地址和凭据，并把每个 `models[]` 条目与该列表对照检查；干净的配置会以一行 `dry run: 0 errors, 0 warnings, N ok` 作答。详情见：[检查文件](configuration.md#从命令行检查文件) 和 [试运行](configuration.md#试运行探测文件指向的内容)。

## 4. 控制台

在一个项目目录中，运行裸命令：

```bash
cd ~/src/my-project
coddy
```

这会打开交互式控制台：输入一个请求，按 Enter，然后看着工具框、思考块和回答流式输出。在默认权限模式下，智能体在执行命令或写入文件之前会先询问。Escape 中断当前轮次，`/quit` 或 `/exit`(或连按两次 Ctrl-C)退出，`coddy -c` 重新打开此文件夹最近的会话。

![一轮结束后的控制台](../../assets/screenshot-console-chat.png)

*控制台中一个已完成的轮次：`read` 工具框、思考块、回答，以及带 token 和上下文计数器、模型和其提供商账号用量的页脚。*

同一个智能体也可以在没有终端的情况下运行，用于脚本或 cron 任务：

```bash
coddy -p "Explain in three sentences how tasks are saved in this repository"
coddy --mode ask -p "Which files read config.yaml?"
git diff | coddy --mode ask -p "Review this change"
coddy -i brief.md
```

`-p` 把回答流式输出到 stdout，把诊断信息发送到 stderr，出错时以非零码退出，并把该轮次作为普通会话持久化；`--mode ask` 使其保持只读。管道输入 `coddy -p "..."` 的数据会随提示词一同传入，对命令行来说过长的提示词可以用 `-i` 从文件读取，或用 `-p -` 从 stdin 读取([单次打印模式](../surfaces/console.md#一次性打印模式--p--prompt))。控制台能做的一切都在 [控制台(TUI)](../surfaces/console.md) 中。

## 5. 浏览器

```bash
coddy serve
```

```text
coddy serve 1.2.31
  httpserver  http://127.0.0.1:12345  (no auth)
```

打开 **http://127.0.0.1:12345/**. 空白屏幕就是一个新对话：在输入框中选择模式、模型、推理级别和权限模式，输入，发送。无论由哪个界面创建，会话都存放在同一个 `~/.coddy` 下，所以控制台和浏览器共享同一份历史。设置在 `#/settings`,OpenAI 兼容 API 在 `/v1/*`，其 Swagger UI 在 `/docs/`。

![Web 界面中的一个会话](../../assets/screenshot-fullhd-chat.png)

*Web 界面中的一个会话，展开的 "编辑文件" 行显示了它所应用的差异。*

服务器故意只绑定回环地址。要从另一台机器访问它，请绑定更宽的地址并要求令牌：`coddy serve -H 0.0.0.0 --auth-token <secret>`(或文件中的 `httpserver.host` 和 `httpserver.auth_token`)。`coddy serve --daemon` 让它在后台保持运行，`coddy serve status | stop | restart` 控制它；参见 [coddy serve 与守护进程](../operate/serve.md)。

## 6. 通过 ACP 使用编辑器

`coddy acp` 通过 stdio 使用 Agent Client Protocol，这正是 Zed、VS Code、Obsidian 和脚本所用的协议。对于 Zed，在 `settings.json` 中添加一个条目，写入二进制的绝对路径(`which coddy` 打印的那个；编辑器并不总能继承你的 `PATH`):

```json
"agent_servers": {
  "Coddy": {
    "type": "custom",
    "command": "/home/you/.local/bin/coddy",
    "args": ["acp"]
  }
}
```

然后在智能体面板的新建线程菜单中选择 *External Agents* 下的 **Coddy**。Coddy 的模式(`agent`、`plan`、`ask`)、其配置的模型和权限策略会出现在 Zed 的输入框中，其技能会以斜杠命令的形式出现。这些会话同样是 `~/.coddy` 中的那些。VS Code、其他 ACP 客户端和参考脚本客户端在 [编辑器](../surfaces/editors.md) 中；协议本身在 [ACP 协议](../reference/acp-protocol.md) 中，可运行的示例位于 `examples/acp/` 下。

## 下一步

- [配置](configuration.md) - 每种提供商类型、`.env` 文件、环境变量和 SSH 远程执行；
- [运行模式](../features/modes.md) - agent、plan 和 ask 各自允许什么，以及如何在每个界面上切换；
- [技能](../features/skills.md)、[规则](../features/rules.md) 和 [MCP 服务器](../features/mcp.md) - 项目如何教会智能体它的约定和工具；
- [Telegram 网关](../surfaces/gateway.md) - 从即时通讯软件访问同样的会话；
- [更新](update.md) - `coddy update`，以及当包管理器掌管该二进制时会发生什么；
- [故障排查](troubleshooting.md) - 当上面的某一步没有按所写的那样进行时。

<!-- docsgen:source sha256=df7f331199a4f2a2 -->

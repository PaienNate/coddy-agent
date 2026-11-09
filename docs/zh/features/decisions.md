# 命令安全（decisions）

开启检查后，Coddy 会在权限门控做出决定之前，向 NeuralDeep decisions 端点询问某条 shell 命令是否安全：使用与 hub 其余部分相同的 `sk-` 密钥，但占用单独的 `POST /v1/decisions` 配额。答案为安全或不安全。安全命令无需权限提示即可运行。不安全的命令在每种模式下都会被询问，提示词中包含判定结果，`bypass` 也不例外：在那里，提示词是操作员仍能做出决定的唯一位置，因此被检查拦下的命令在操作员按下 **允许** 后仍可运行，并且智能体会在同一轮次读取其输出。在无人能回答提示词的地方，例如调度任务、`coddy -p`、父轮次已结束且没有可询问界面的子智能体，命令会被拒绝。拒绝会作为该调用的结果返回，即 `command rejected as unsafe: the decisions model frida-decisions classified the command as dangerous (unsafe at 0.99, threshold 0.50); ask the operator or use a safer alternative`，因此会话对话记录会记录它，Web 界面中的工具调用卡片会以已取消状态显示它，模型会读取它并可以选择另一条路径。

该检查默认关闭。可在 设置 → **命令安全（决策）** 中开启，或在 `config.yaml` 中开启：

```yaml
decisions:
  enable: true
  model: frida-decisions   # or clef-flash
  threshold: 0.5           # unsafe once p(unsafe) reaches this
```

## 检查何时运行

该检查覆盖 `run_command` 以及 `ssh_run_command` 的远程命令，并且只检查操作员未明确允许的命令：

| 命令 | `bypass` | `ask`、`accept_edits` |
|---|---|---|
| 安全 | 运行 | 无提示运行 |
| 不安全，或检查未给出判定 | 会被询问，提示词中包含判定结果；在无人能回答时被拒绝 | 会被询问，提示词中包含判定结果；在无人能回答时被拒绝 |
| 由 `tools.command_allowlist`、会话对恰好该命令的“始终允许”授权，或钩子的 `allow` 允许 | 运行，不检查 | 运行，不检查 |

检查引发的提示词是任何发送方都不会自行回答的提示词：在任何界面上的 `bypass`，以及会批准自己智能体的消息机器人，都会把它留给一个人。其选项包括一次 **允许**、**始终允许**（针对恰好该命令的会话授权，此后该命令会被信任且不再检查；对于 `ssh_run_command`，不会存储任何内容，因此其作用等同于 **允许**）和 **拒绝**，后者会像任何其他提示词一样拒绝该调用（`permission denied by user`）；在 `ask` 中，也会提供任何提示词的会话开关。只有 `tools.command_allowlist` 和对恰好该命令的授权能让命令通过检查：同一命令加上参数后，或普通提示词的程序级授权所覆盖的命令，都会再次被检查，因此 `rm -rf build /` 仍会被检查，即使在 `rm -rf build` 上按下 **始终允许** 之后也是如此。这也解释了为什么检查的提示词不提供始终允许整个程序的选项，而普通命令的提示词会提供：这样的授权不会记住检查所读取的任何内容。

在有人能回答的界面上（Web 界面、控制台、编辑器），检查的提示词会像任何其他提示词一样等待回答，因此在 `bypass` 中运行的轮次（包括受监督的目标）会在被标记的命令处停止，直到有人回答或停止它。

有几种调用保持原样：

- `PreToolUse` 钩子的 `ask` 保留其提示词，并且该命令不会被检查；
- 操作员在权限提示词中批准的命令不会再次被检查，无论是在全新调用中，还是在已回答提示词的恢复中，都是如此；
- 非机器人管理员的消息用户所在轮次会保留每个提示词供机器人拒绝，因此检查绝不会让此类轮次中的命令运行；
- 前台和后台命令（`background: true`）都会在进程启动前接受检查，子智能体子项会在相同的门控下以其继承的配置运行。

[Telegram](../surfaces/gateway.md) 和 [Pachca](../surfaces/pachca.md) 机器人会自行批准其聊天中自己智能体的提示词，但不会批准检查的提示词：它们会在聊天中用按钮询问，就像询问子智能体的提示词一样，并且只有机器人的管理员可以回答（在其他任何人的轮次中，提示词会被拒绝，如上所述）。通过 HTTP 启动的计划运行没有可询问的人，因此会拒绝该命令。对于 `PreToolUse` 钩子通过 `ask` 强制产生的提示词（[钩子](hooks.md)），情况也是如此：钩子正是通过这种方式让操作员运行它宁愿不未经询问就放行的命令。

按照设计，MCP 工具不会被检查：服务器的工具之所以运行，是因为操作员声明了该服务器，并且对于项目带来的服务器，还信任了它（[MCP 服务器](mcp.md)），而工具调用中没有任何内容能说明服务器用它执行的操作是否安全。文件工具也不会被检查。

## 问题与判定

发送到端点的状态是命令文本加上它将在何处运行：`run_command` 的工作目录、`user@host`（以及 22 以外的端口）所对应的 `ssh_run_command`；问题是一个单独的 `choice`，带有两个有描述的选项：`safe`（常规开发命令：读取、构建、测试、安装，只更改可重新构建的项目文件）和 `unsafe`（大范围或强制删除、磁盘擦除、数据库删除、强制推送、将下载内容管道到 shell、任何超出项目的不可逆操作）。端点会回答所选项以及每个选项的概率；当 `unsafe` 的概率达到 `decisions.threshold`（默认 0.5，即两个选项互换的临界点）时，Coddy 将该命令计为不安全，低于该值则计为安全：阈值是操作员的调节旋钮，而不是端点自己的选择，因此一个在 0.7 时选择了不安全的答案，在 0.9 的阈值下仍计为安全，而一个临界值 0.6 在默认阈值下计为不安全。不带概率的裸答案计为确定。`rm -rf /` 在两个模型上的得分约为 p(unsafe)=0.99；普通的 `echo` 或 `go test ./...` 会被归类为安全。

## 模型

`decisions.model` 会在端点提供的两个模型之间选择：

| 模型 | 状态预算 | 典型延迟 | 每次请求的配额单位 |
|---|---|---|---|
| `frida-decisions`（默认） | 512 个 token 的命令文本 | ~20 ms | 1 |
| `clef-flash` | 8192 个 token | ~150 ms | 4 |

`frida-decisions` 是一次不生成内容的编码器传递，这就是它既是默认选项又是廉价选项的原因；`clef-flash` 是更大的生成式模型，适用于上下文很长的命令。该检查使用 decisions 配额桶，与聊天类别分开：余额由 hub 上的 `GET /v1/decisions/quota` 提供。

该预算是硬性限制。hub 会截断比模型可读取长度更长的状态，并报告截断；而对命令开头部分的判定并不能说明其余部分如何，破坏性部分可能就在那里。因此，Coddy 会把模型只读取了一部分的命令视为检查未给出判定的命令：提示词会说该命令对模型来说太长，而在无人能回答的地方，结果也会这样说明，并告诉智能体将其拆分为更短的命令，或使用文件工具写文件，而不是使用 heredoc。模型已经判定为不安全的开头部分计为不安全。如果操作员的智能体运行长命令，可将 `decisions.model` 设置为 `clef-flash`。

## 凭据

该检查解析凭据的方式与通过 neuraldeep 提供商发出的聊天请求完全相同：显式的 `providers[].api_key`（或 `api_key_command`），位于第一个 `neuraldeep` 行上，然后使用该行的 `NAME_API_KEY` 环境变量（`NEURALDEEP_API_KEY` 用于名为 `neuraldeep` 的行），再使用该行存储的 hub 登录信息（`coddy providers login <name>`）；`api_base` 选择部署（`.ru` 或 `.tech`），并且 `providers[].proxy` 适用。该行可以不存在：此时检查使用名称 `neuraldeep`，因此 `NEURALDEEP_API_KEY` 或由 `coddy providers login neuraldeep` 存储的登录信息就足够了。

## 端点无法回答时

每条命令会向端点询问一次，一次请求最多等待五秒。当它未给出判定时（没有凭据、密钥被拒绝、请求被拒绝（钱包为空、不提供所请求的模型）、速率限制、网络故障或服务器错误、问题不可能得到的答案，或命令对模型来说太长），该命令计为不安全：会带着提示词中的原因询问；在无人能回答的地方，则不会执行，并在结果中给出原因（`command not executed: the decisions safety check gave no verdict (...)`）。已开启的安全网不会静默地放行；要运行不带它的命令，请关闭 `decisions.enable`。

## 测试

离线测试套件会代替端点：`internal/llm/decisions_test.go` 在 `httptest` 服务器上固定请求形状、答案形状和错误类型，`internal/agent/decisions_test.go` 加上 `features/decisions_check.feature` 会驱动门控本身：每种模式下上表的每一行、不是判定的答案，以及检查绝不能触碰的调用。两个实时探测针对真实端点运行，除非设置了 `NEURALDEEP_API_KEY`，否则都会跳过：传输探测 `internal/llm/decisions_live_test.go`（`rm -rf /` 不安全，`echo` 安全，两个模型）和端到端探测 `internal/agent/decisions_live_e2e_test.go`，即在无人回答提示词的界面上以 bypass 模式运行一个完整轮次，其脚本化模型请求 `rm -rf /` 和一个 `echo`：破坏性调用在对话记录中返回为已拒绝，而回显会运行：

```
NEURALDEEP_API_KEY=... go test ./internal/agent -run TestLiveDecisionsE2E -count=1 -v
```

![设置抽屉中的 "Command safety (decisions)" 标签页](../../assets/decisions/decisions-settings-dark-1280.png)

*设置 → 命令安全（决策）：启用开关默认关闭，位于“模型和阈值”字段集上方，其中的 (i) 说明了这两个字段*

<!-- docsgen:source sha256=5367f8a04b5eb515 -->

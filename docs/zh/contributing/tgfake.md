# 使用 tgfake 进行集成测试

Coddy 的 Telegram 机器人会针对 [tgfake](https://github.com/EvilFreelancer/tgfake) 进行测试，这是一个伪造的 Telegram Bot API，带有聊天页面、模拟 API 和脚本化的 OpenAI 兼容模型。它过去作为 `internal/tgfake` 和 `cmd/tgfake` 位于此处；现在它已是一个独立项目，以二进制文件形式发布并作为 Go 模块导入，因此任何机器人都可以在同一个沙箱中测试。本页介绍贡献者运行这些测试、阅读失败信息以及将 Coddy 迁移到新版 tgfake 所需了解的内容。

## 一个版本，三种使用方式

`go.mod` 只固定一次 tgfake，本仓库中的所有内容都使用该版本：

```text
require github.com/EvilFreelancer/tgfake vX.Y.Z
tool github.com/EvilFreelancer/tgfake/cmd/tgfake
```

| 位置 | 如何使用 tgfake |
|-------|--------------------|
| `external/gateway/telegram` 测试（`-tags gateway`） | `pkg/server` 在 `httptest` 上以进程内方式运行，作为 `tgfake` 导入，并通过 `fakeapi_test.go` 打开；键盘类型使用 `pkg/botapi`。每个需要 Telegram 的网关测试都会经过它。 |
| `internal/agent`、`external/httpserver`、`external/cli` 和网关测试套件 | 使用 `pkg/llmstub`，即脚本化模型，并设置 `StripTags: []string{"turn_context"}`，使规则匹配用户写下的内容，而不是 Coddy 附加到每个请求的块。 |
| `examples/gateway/tg_e2e_offline.sh`、`examples/shared/wake_e2e_common.py`、`examples/cli/capture_wake.py`、`npm run check:telegram` | `tgfake` 二进制文件：`CODDY_TGFAKE_BIN` 指定的那个，否则从固定模块构建（`go build github.com/EvilFreelancer/tgfake/cmd/tgfake`）。它们用 `--llm-model coddy-demo --llm-strip-tag turn_context` 启动它。 |
| CI，任务 **Telegram 测试台（tgfake）** | 固定版本的发布二进制文件，由 tgfake 自带的 GitHub Action 安装，然后让 `tg_e2e_offline.sh`（带 `TG_E2E_METRICS=1`）针对真实的 `coddy serve` 运行。 |

Go 测试只需要 `go test`：该模块会像任何其他依赖一样被下载。只有脚本会启动进程。

## 运行测试

进程内测试套件是普通运行的一部分：

```bash
go test -tags gateway ./external/gateway/telegram/
```

```bash
make test
```

端到端运行会启动二进制文件和一个真实的 `coddy serve`，并将其机器人指向该测试台：一条消息得到回复，`/clear` 和 `/resume` 回到同一会话，后台任务会将智能体唤醒到聊天中。

```bash
make build TAGS=gateway
```

```bash
examples/gateway/tg_e2e_offline.sh
```

`TG_E2E_METRICS=1` 还会启用 Prometheus 导出器并在聊天后抓取它；它需要带有 `TAGS="gateway metrics"` 构建的二进制文件，正如 CI 构建的那样，而没有它时，脚本只需要一个 `gateway` 构建。

`TG_E2E_KEEP=1` 会让测试台保持运行，并打印聊天页面地址，你可以在那里继续与机器人对话。要运行预构建的二进制文件而不是自行构建，请安装固定版本的发布包并指定它：

```bash
go list -m -f '{{.Version}}' github.com/EvilFreelancer/tgfake
```

```bash
curl -sSfL https://raw.githubusercontent.com/EvilFreelancer/tgfake/vX.Y.Z/scripts/install.sh | sh -s -- -b ./build vX.Y.Z
```

```bash
CODDY_TGFAKE_BIN=./build/tgfake examples/gateway/tg_e2e_offline.sh
```

同一个变量会提供给后台唤醒框架（`examples/test_httpserver.sh`、`examples/test_acp.sh`、控制台的 `cli_e2e_background_wake.py`）以及 Mini App 检查（`npm run check:telegram`，参见 [Web 界面页面](../surfaces/web-ui.md)）。

要手动操作一个测试台，请从仓库根目录启动固定版本，并遵循 [针对伪造 Bot API 进行调试](../surfaces/gateway.md#针对假-bot-api-调试)：

```bash
go tool tgfake --llm --llm-strip-tag turn_context
```

## 在 CI 中

`.github/workflows/tests-on-pr.yaml` 的 **Telegram 测试台（tgfake）** 任务是集成层：

1. 它读取 `go.mod` 固定的版本，使用 `go list -m`，并在工作流以不同 ref 使用 tgfake 的 action 时失败，这样两者就不会发生偏移；
2. `EvilFreelancer/tgfake@vX.Y.Z` 为该运行器安装该版本的发布二进制文件，并根据发布的校验和进行检查；
3. 它构建 `coddy`，使用 `gateway` 和 `metrics` 标签，并运行 `examples/gateway/tg_e2e_offline.sh`，同时设置 `CODDY_TGFAKE_BIN` 和 `TG_E2E_METRICS=1`。

不涉及令牌、模型，也不涉及下载之外的网络。进程内测试套件像其他所有测试一样在测试矩阵中运行。

## 值得了解的默认值

- 测试台扮演 `@tgfake_bot`，名字为 `tgfake`。轮询功能将其机器人命名为 `coddy_fake_bot` / `Coddy Fake`，通过 `Options`；其他测试采用默认值。
- 脚本化模型报告 `tgfake-demo`。Coddy 的测试和脚本请求 `coddy-demo`（Go 中为 `Model`，命令行上为 `--llm-model`），因为它们的配置指定了 `stub/coddy-demo`。
- 如果没有 `--llm-strip-tag turn_context`（或 Go 中的 `StripTags`），模型会将 `<turn_context>` 块视为提示词，且没有规则会匹配。
- 默认人员是用户 4242 `alice`，其私聊具有相同的 id。

## 更改 tgfake

需要 Telegram 以新方式行为的测试，例如网关开始调用的方法、它必须承受的拒绝，属于对 tgfake 的更改，而不是在这里编写的处理器。其仓库记录了服务器如何镜像 Telegram（`AGENTS.md`、`.claude/rules/telegram-fidelity.md`）。

1. 在 tgfake 中连同其测试一起进行更改，合并并推送标签 `vX.Y.Z`；其发布工作流会发布二进制文件。
2. 在这里，切换到新版本并让工具保持同步：

   ```bash
   go get github.com/EvilFreelancer/tgfake@vX.Y.Z
   ```

   ```bash
   go mod tidy
   ```

3. 将 `uses: EvilFreelancer/tgfake@...`（在 `tests-on-pr.yaml` 中）改为同一个标签，然后运行 `make test` 和 `examples/gateway/tg_e2e_offline.sh`。

要同时处理两者，请将一个位于两个检出之外的 workspace 文件指向两棵树，并将其用于本仓库的命令。workspace 还需要 `replace` 行，因为 `go.mod` 指定了本地树所代替的版本：

```text
go 1.26.0

use (
	./coddy-agent
	./tgfake
)

replace github.com/EvilFreelancer/tgfake vX.Y.Z => ./tgfake
```

```bash
GOWORK=$PWD/../coddy-tgfake.work go test -tags gateway ./external/gateway/telegram/
```

通过 `go build` 构建的脚本会以同样的方式选取本地树。该 workspace 文件永远不会被提交：CI 和所有其他检出都使用固定版本的发布包。

<!-- docsgen:source sha256=d9117e6a969e0277 -->

# CI 中的 Coddy

一个向智能体提出一个问题并读取答案的流水线步骤：审查拉取请求、总结 diff、检查文档是否仍与代码一致。控制台的打印模式，`coddy -p`（[控制台](../surfaces/console.md)，"一次性打印模式"），在没有终端的情况下运行一个轮次，将助手文本流式传输到 stdout，将诊断信息流式传输到 stderr，并在出错时以非零状态退出。发布二进制包含控制台；精简构建会用重新构建提示来回答 `-p`。

1. **在运行器上安装二进制。** 一行安装脚本（[安装](../getting-started/install.md)）会将 `coddy` 放入 `~/.local/bin`，并在不存在示例文件时根据示例文件创建 `~/.coddy/config.yaml`。把该目录加入 `PATH`，供后续步骤使用。

   ```bash
   curl -fsSL https://coddy.dev/install.sh | bash
   export PATH="$HOME/.local/bin:$PATH"
   coddy -v
   ```

2. **为任务提供独立的主目录。** `CODDY_HOME` 是配置、会话和日志所在的位置（默认 `~/.coddy`）。将其指向临时目录，这样一个任务的对话记录就绝不会泄漏到下一个任务，运行器被丢弃时也不会留下任何东西。加载器会读取 `$CODDY_HOME/config.yaml`，并在其之前读取 `$CODDY_HOME/.env`，以获取 shell 未设置的变量；`CODDY_CONFIG` 指定别处的配置文件，`--cwd`（或 `CODDY_CWD`）设置会话的工作区，在任务中就是检出目录。任务的每个会话都落在 `$CODDY_HOME/sessions` 下。

   ```bash
   export CODDY_HOME="$(mktemp -d)"
   ```

3. **编写一个密钥来自环境的配置。** 提供商的 `api_key` 可以是一个 `${VAR}` 引用，在文件加载时展开；也可以留空，此时密钥会在调用时从 `NAME_API_KEY` 读取 - 提供商名称转为大写，并将连字符变为下划线，因此 `openai` 读取 `OPENAI_API_KEY`，`team-gateway` 读取 `TEAM_GATEWAY_API_KEY`。无论哪种方式，密钥都保留在 CI 密钥存储中，绝不会出现在文件里。固定模型和轮次上限：`agent.max_turns` 限制一个步骤的成本。如果完全没有 `config.yaml`，仅靠 `OPENAI_API_KEY` 或 `ANTHROPIC_API_KEY` 会给出默认提供商和模型，但任务应固定自己的配置。[配置](../getting-started/configuration.md) 包含所有键。

   ```yaml
   providers:
     - name: openai
       type: openai
       api_key: "${OPENAI_API_KEY}"   # or leave it empty: OPENAI_API_KEY is read when the model is called

   models:
     - model: "openai/gpt-5.4-mini"
       max_tokens: 16384

   agent:
     model: "openai/gpt-5.4-mini"
     max_turns: 20
   ```

4. **在消耗任何 token 之前对配置做门禁检查。** `coddy -t` 会依据内嵌 schema 和加载器规则检查启动时将加载的文件，打印每个问题及其行号，并打印一行 `fix:`，有错误时以 1 退出，不启动任何内容（[从命令行检查文件](../getting-started/configuration.md#从命令行检查文件)）。`coddy --dry-run` 运行同样的检查，然后向每个提供商询问其模型列表，这会用一个请求同时检验地址和密钥，因此错误的密钥会在这里于具体的某一行失败，而不是在审查中途变成 401（[试运行](../getting-started/configuration.md#试运行探测文件指向的内容)）。两个标志在每个构建中都可用。

   ```bash
   coddy -t          # the file: schema and loader rules
   coddy --dry-run   # the file, then the providers it names
   ```

5. **运行提示词。** `-p` 接收提示词。`--mode ask` 将该轮次限制为只读工具（`read`、`glob`、`grep`、`print_tree`、`websearch`、`webfetch`、`question`、`load_skill`、`coddy_docs_search`、`coddy_docs_read`：没有 shell、没有文件写入、没有 MCP 工具），这对审查来说是合适的形态，也意味着不会出现权限提示。`--model` 选择一个已配置的模型，`--cwd` 指定工作区。答案进入 stdout，其他所有内容进入 stderr，因此重定向 stdout 会得到一个可以作为评论发布的文件。

   ```bash
   git diff origin/main...HEAD | coddy -p "Review this change. Read the files it touches for context, report defects with path:line, and end with one line: VERDICT: ok or VERDICT: needs work." --mode ask > review.md
   ```

   diff 通过 stdin 传入，并作为附件随提示词一起传递，就像 `codex exec` 接收管道 stdin 一样，因此模型无需工具调用即可获得它；stderr 上的一行会说明附加了多少内容。对于命令行来说过长的提示词（Linux 上操作系统将单个参数限制为 128 KiB）可放入文件 `coddy -i review-brief.md`，或放在 stdin 上 `coddy -p - < review-brief.md`，而文件提示词也以同样方式接收 diff：`git diff | coddy -i review-brief.md`。当某个步骤的 stdin 属于其他东西时 - 对文件列表的 `while read` 循环、`ssh`，以及 GitLab 运行器上的每个任务（它通过 stdin 将任务脚本提供给 shell） - 则传入 `--no-stdin`，以便不附加任何内容（[一次性打印模式](../surfaces/console.md#一次性打印模式--p--prompt)）。

   应当编辑文件的步骤改为在 `agent` 模式下运行，其权限请求随后无需人工即可得到答复：`--permission-mode bypass` 允许一切，`accept_edits` 批准文件写入并拒绝 shell 命令，而 `ask`（新会话在选定模式之前启动时所在的状态）会拒绝每个请求并在 stderr 上给出说明。提问工具会得到空答案。后续步骤使用 `coddy -c -p "..."` 继续同一会话，因此第二个问题能看到第一个问题的文件和工具结果；它还会保留第一步的模型、模式和权限模式。`--permission-mode` 仅设置其运行的会话：继续该会话的步骤会随会话保留模式，而新会话会以会话内最后选择的模式启动（在做出任何选择之前为 `ask`），无论之前的步骤传入了什么。如果继续的步骤不应在 `bypass` 下运行，则再次传入 `--permission-mode`。如果某个步骤的会话不会再被任何人读取，则传入 `--ephemeral`，运行结束时它会删除该会话。

6. **读取退出状态。**

   | 命令 | 退出码 0 | 退出码 1 |
   |---|---|---|
   | `coddy -p "..."` | 该轮次完成并打印了答案 | 配置未加载、提供商失败、某个标志指定了未知模型或模式、提示词输入被拒绝（两个提示词、文件缺失、文本不是 UTF-8、超过 8 MiB），或轮次被取消 |
   | `coddy -t` | 文件有效（警告不会使其失败） | 文件有错误或不存在 |
   | `coddy --dry-run` | 每个探测都通过 | 文件有错误或某个探测失败 |

   `-p` 的状态报告的是进程，而不是判定结果：即使轮次停在 `agent.max_turns`，仍然以 0 退出。让提示词以上面那样的显式行结尾，并让流水线用 grep 查找它。

7. **决定检出内容可以执行什么。** `.coddy/mcp.json`、`.coddy/hooks.json` 和 `.coddy/agents/` 随仓库到达，并在默认 `ask` 策略下保持，直到操作员批准它们，而在运行器上没人会这么做。对于你拥有的仓库，传入 `--mcp-project-trust allow`（控制台、`acp` 和 `serve` 都接受它），或在任务的配置中将 `mcp.project_trust`、`hooks.project_trust` 和 `subagents.project_trust` 设为 `allow`。对于在来自陌生人的拉取请求上运行的任务，将它们保留为 `ask`，或设置 `deny`，这样分叉就无法选择运行器执行什么。[MCP 服务器](../features/mcp.md)、[钩子](../features/hooks.md) 和 [子智能体](../features/subagents.md) 描述了这三个门禁。

组合为一个 GitHub Actions 任务：

```yaml
name: Review
on: [pull_request]

jobs:
  review:
    runs-on: ubuntu-latest
    env:
      CODDY_HOME: ${{ runner.temp }}/coddy
      OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - name: Install Coddy
        run: |
          curl -fsSL https://coddy.dev/install.sh | bash
          echo "$HOME/.local/bin" >> "$GITHUB_PATH"

      - name: Write the config
        run: |
          mkdir -p "$CODDY_HOME"
          cat > "$CODDY_HOME/config.yaml" <<'EOF'
          providers:
            - name: openai
              type: openai
              api_key: "${OPENAI_API_KEY}"
          models:
            - model: "openai/gpt-5.4-mini"
              max_tokens: 16384
          agent:
            model: "openai/gpt-5.4-mini"
            max_turns: 20
          EOF

      - name: Check the config and the provider
        run: coddy --dry-run

      - name: Review the change
        run: |
          git diff origin/main...HEAD | coddy -p "Review this change. Read the files it touches for context, report defects with path:line, and end with one line: VERDICT: ok or VERDICT: needs work." --mode ask | tee review.md
          grep -q 'VERDICT: ok' review.md
```

<!-- docsgen:source sha256=dc31e67cda913766 -->

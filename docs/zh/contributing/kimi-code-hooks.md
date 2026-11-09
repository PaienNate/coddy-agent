# 面向项目规则的 Kimi Code 钩子

本页面面向在本仓库上运行 Kimi Code 的贡献者。它是 [Codex 钩子](codex-hooks.md) 与 [ZCode 钩子](zcode-hooks.md) 的 Kimi Code 对应版本：三者都通过各自客户端的生命周期钩子读取相同的 `.cursor/rules/*.mdc` 文件。

## 为什么需要钩子

Kimi Code 会读取仓库的 `AGENTS.md`，但它不会解释 `.cursor/rules/*.mdc` 中的 `globs` 和 `alwaysApply` 字段。要求模型在每次编辑前打开相关规则只是一种建议性做法，可能会被跳过。

`.kimi-code/hooks/attach_rules.py` 直接读取该 frontmatter，并把规则正文交给模型。Cursor 目录树是适配器的输入，而不是通用的策略格式，适配器不会另行维护一份规则正文的 Kimi Code 副本。

## 何时触发什么

针对 2.3.2 版本验证，Kimi Code 为钩子提供了两种触达模型的方式。它会把 `UserPromptSubmit` 钩子打印的 `message` 追加到用户的提示词中，并把 `PreToolUse` 钩子给出的拒绝调用理由作为该调用的结果返回。Kimi Code 会丢弃 `SessionStart` 钩子打印的内容，以及在放行调用时 `PreToolUse` 钩子打印的内容，因此适配器两者都不使用。

| 事件 | 匹配器 | 钩子的行为 |
|---|---|---|
| `UserPromptSubmit` | 无 | 把 frontmatter 中含有 `alwaysApply: true` 的每条规则追加到会话的第一条提示词 |
| `PreToolUse` | `^(Edit\|Write)$` | 拒绝第一次触及尚未发送的作用域规则所覆盖文件的调用，并将这些规则作为理由；模型会重复该调用，重复的调用会被放行 |
| `PostCompact` | 无 | 忘记已发送过哪些规则，使它们在压缩之后再次送达 |

这三个条目运行同一个脚本，脚本根据 JSON 载荷中的 `hook_event_name` 进行分支判断。Kimi Code 会在 `tool_input.path` 中传入 `Edit` 或 `Write` 调用的路径。该脚本也能识别其他常见的路径字段（包括嵌套字段），并从会话所在文件夹解析相对路径。

一次拒绝最多携带 45,000 个字符，因为 Kimi Code 会把更长的工具结果移入文件，只向模型显示其开头部分。当某次调用的规则更长时，每次尝试会携带下一部分，待所有部分都送达后该调用才会放行。对于本仓库的一个 Go 文件，这通常需要两次拒绝。

## 启用

Kimi Code 只从**用户级**的 `~/.kimi-code/config.toml`（或 `$KIMI_CODE_HOME/config.toml`）读取钩子声明，并在每个项目中从会话所在文件夹运行它们。因此仓库以片段形式提供 `.kimi-code/hooks.toml` 供合并，而不是作为 Kimi Code 自行加载的配置文件。

1. 将 `.kimi-code/hooks.toml` 中的三个 `[[hooks]]` 条目追加到用户级 `config.toml`，保留其中已有的钩子。不要用该片段替换整个配置。
2. 运行 `kimi doctor` 检查该文件，然后启动一个新的 Kimi Code 会话。

与 Codex 钩子类似，每条命令都会运行 `"$(git rev-parse --show-toplevel)/.kimi-code/hooks/attach_rules.py"`，即会话所在检出目录中的脚本。因此一组条目即可覆盖本仓库的每个检出和 worktree，也包括在子文件夹中启动的会话。

每条命令都以 `|| exit 0` 结尾。在其他项目中，或在 git 仓库之外，Python 无法打开该脚本并以退出码 2 退出，而 Kimi Code 会把退出码 2 解读为对提示词或编辑的拒绝。`|| exit 0` 会将其转换为一个什么都不做的钩子。

Kimi Code 通过 `sh` 运行这些命令，在 Windows 上则通过 `cmd.exe`。`cmd.exe` 不会展开 `$(...)`，因此在 Windows 上该钩子什么也不做。

## 工作原理

- **frontmatter 解析器与其他 Python 适配器一致。** 它支持标量和 YAML 列表形式的 `globs`、带引号的逗号、转义、注释以及多行流式列表，且不依赖第三方 YAML 库。
- **glob 匹配为自行实现。** `**/` 表示零个或多个目录，`*` 限定在单个路径段内，单独的 `**` 可跨越任意内容。检出目录内的绝对路径会转换为仓库相对路径，相对路径从会话所在文件夹解析；最终位于检出目录之外的路径会被忽略。
- **每条规则在下次压缩之前最多送达某会话一次。** 状态存放在 `<tempdir>/kimi-code-attach-rules-<repository-hash>/<session_id>.json` 之下，因此多个克隆之间不会相互抑制。恢复的会话会保留其状态，而 `/new` 或 `/clear` 会以新 id 启动会话。并发进程通过按会话加的锁来串行化送达，状态更新使用原子替换。
- **只有在记录成功之后才会发出拒绝。** 如果状态无法加锁、写入或读回，编辑会在没有作用域规则的情况下放行，因此重试不会再因相同的规则被拒绝。此时常开规则仍会被追加到提示词中，并且可能重复出现。
- **它以放行方式失败。** 格式错误的 JSON、无法读取的规则或不可写的状态目录都不会拒绝任何内容。该钩子只负责传递指令，不强制执行任何内容。

## 验证与调试

运行仓库的适配器契约测试，它覆盖 Kimi Code 以及其他宿主，并按照 Kimi Code 2.3.2 的方式读取钩子的输出：

```bash
make test-agent-rules
```

在检出目录的根目录，用合成的 Kimi Code 载荷驱动真实脚本。载荷中没有 `cwd` 时，脚本会把自身的工作文件夹当作会话的文件夹。

```bash
echo '{"hook_event_name":"UserPromptSubmit","session_id":"kimi-probe","prompt":"hello"}' | python3 .kimi-code/hooks/attach_rules.py
echo '{"hook_event_name":"PreToolUse","session_id":"kimi-edit-probe","tool_name":"Edit","tool_input":{"path":"cmd/coddy/providers.go"}}' | python3 .kimi-code/hooks/attach_rules.py
```

第一条命令会打印一个 JSON 对象，其 `message` 中存放常开规则。第二条命令会打印一次拒绝：`hookSpecificOutput.permissionDecision` 为 `deny`，`permissionDecisionReason` 中存放规则。重复执行以获取下一部分，直到它不再打印任何内容。输出为空且退出码为 0 意味着没有 glob 匹配、该规则在该会话中已发送过，或输入未被理解。另一次探测请使用新的会话 id，把 `KIMI_CODE_RULES_STATE_DIR` 设置为一个临时目录，或删除 `<tempdir>/kimi-code-attach-rules-*` 以清除状态。`KIMI_CODE_RULES_DIR` 可以指向检出目录内的一个 fixture 规则树，用于隔离测试。

更改用户级配置后，请在**新的** Kimi Code 会话中验证，而不要假定已有会话已重新加载其钩子。第一条提示词会显示一个包含常开规则的 `UserPromptSubmit hook` 块，而对 Go 文件的第一次编辑会在执行前被以作用域规则为由拒绝。

## 与其他适配器的关系

该脚本与 `.zcode/hooks/attach_rules.py` 共享其 frontmatter 解析器、glob 匹配、锁和状态文件：两者都通过 JSON 字段接收被编辑的路径，而不需要 Codex 的 `apply_patch` 头部。送达方式不同，因为 Kimi Code 既不读取 `additionalContext`，也不读取 `SessionStart` 的输出。与 [OpenCode 插件](opencode-hooks.md) 类似，该适配器会拒绝一次写入，以便引入作用域规则。修改适配器时请保持共享部分一致，并通过 `make test-agent-rules` 覆盖每个被修改的适配器。

上游参考：[Kimi Code 钩子](https://www.kimi.com/code/docs/en/kimi-code-cli/customization/hooks.html) 和 [配置文件](https://www.kimi.com/code/docs/en/kimi-code-cli/configuration/config-files.html)。

<!-- docsgen:source sha256=165357bea49be016 -->

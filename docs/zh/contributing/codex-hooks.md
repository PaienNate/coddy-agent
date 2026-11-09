# 用于项目规则的 Codex 钩子

本页讲的是**针对本仓库运行 Codex CLI 的贡献者**。它不涉及 Coddy 自身的规则发现机制，那是一项产品功能，记录于 [Rules](../features/rules.md) - 那篇描述的是 `coddy` 二进制如何将 `{{.Rules}}` 注入其系统提示词。

## 为什么需要钩子

Codex **每个会话只解析一次**其指令链，从仓库根目录向下遍历到它启动时所在的目录，并每个目录最多拼接一个 `AGENTS.md`。有两点影响在此很重要：

- 在仓库根目录启动的会话永远不会加载嵌套的 `AGENTS.md`，无论它随后编辑哪些文件；
- 不存在基于 glob 的附加机制。Codex 没有 `.cursor/rules/*.mdc` 中 `globs` 字段的对应物，因此作用域只能通过文件在树中的位置来表达。

本仓库在 `.cursor/rules/*.mdc` 下保留了 Cursor 的原生表示，Codex 适配器直接读取这些文件。此前手工维护的索引只是让模型去打开另一个文件；那只是建议性的，会被跳过。

`.codex/hooks/attach_rules.py` 通过将规则正文直接注入会话弥合了这一缺口，因此合规性不再依赖模型主动选择遵循某个索引。

## 何时触发

| 事件 | 匹配器 | 注入内容 |
|---|---|---|
| `SessionStart` | `startup\|resume\|clear\|compact` | 每个 frontmatter 中含有 `alwaysApply: true` 的规则 |
| `PreToolUse` | `^(apply_patch\|Edit\|Write)$` | `globs` 覆盖待处理补丁中某个文件的规则 |

配置位于 `.codex/hooks.json`。两个处理函数运行同一个脚本；它根据钩子负载中的 `hook_event_name` 分支。

对本仓库而言，`russian-wording` 和 `merged-branch-cleanup` 在会话启动时到达。Go、HTTP、网关和 UI 主题规则是路径作用域的，按需附加。一次涉及 `external/httpserver/server.go` 的编辑会引入 `api-layer`、`architecture`、`code-style`、`implementation-order`、`testing` 和 `workflow`；涉及 `external/ui/src/` 的编辑还会引入 `ui-spa` 和 `ui-verification`。

## 工作原理

项目适配器自己解析 `.mdc` frontmatter（`description`、标量或列表形式的 `globs`，以及 `alwaysApply`）。带引号的逗号、成对的单引号、双引号转义，以及合法的标量、单行或多行 flow-list 和 block-list 形式中的 YAML 注释，都会按需解码或去除。它以 `.cursor/rules/` 作为输入，不为规则正文保留单独的 Codex 副本。新增 Cursor 规则文件无需适配器接线。以下要点值得了解：

- **Glob 匹配是手写的。** `fnmatch` 不可用，因为它的 `*` 也会跨越 `/`，这导致 `external/httpserver/**/*.go` 匹配不到 `external/httpserver/server.go`。脚本将 glob 转换为正则表达式，其中 `**/` 变成“零个或多个目录”,`*` 保持在单个路径段内，而单独的 `**` 可跨任意内容。
- **路径来自补丁头和结构化工具字段。** 会读取 `*** Add File:`、`*** Update File:`、`*** Delete File:` 和 `*** Move to:`，以及诸如 `file_path`、`path`、`source` 和 `destination` 等已知 JSON 字段。绝对路径会被转换为仓库相对路径，仓库之外的路径会被忽略。
- **每个作用域规则在每个活动会话上下文中最多注入一次。** 状态是 `<tempdir>/codex-attach-rules-<repository-hash>/<session_id>.json` 下的一个 JSON 文件，因此不同克隆之间不会相互抑制。`resume` 保留作用域去重；`startup`、`compact` 和 `clear` 会将状态重置为常开集合，以便作用域规则在需要时能再次附加。并发进程通过每会话锁将投递串行化，状态更新使用原子替换。规则 id 只有在上下文输出成功后才会被持久化，因此输出失败时它们仍可在后续尝试中生效。如果锁或持久化不可用，匹配的规则上下文仍会被输出，去重可能暂时退化为重复投递。
- **失败时放行。** stdin 上格式错误的 JSON、无法解析的规则文件，或不可写的状态目录，都会以退出码 0 结束且无输出。损坏的规则永远不会阻塞编辑。

## 上下文预算

Codex 默认将模型可见的钩子输出上限设为约 2500 个 token，然后把剩余部分**溢出**到 `<temp_dir>/hook_outputs/`，改为向模型展示首尾预览。因此两个处理函数都会提高 `additionalContextLimit`，会话启动时为 8000，每个补丁为 6000。这些是上限而非预留量；实际负载远低于它们。

添加规则时请记住这一点。该预算与会话中所有其他钩子和插件共享，过大的常开上下文会降低模型表现，而不是帮助它。

## 信任

Codex 要求非托管命令钩子在运行前获得明确批准，并依据**钩子定义的哈希**记录该批准。未获批准的钩子会被跳过且不会报硬错误，这与钩子未触发看起来完全一样。

- 在每个克隆中运行一次 `/hooks`，以审查并信任两个处理函数。
- 对 `attach_rules.py` 或 `hooks.json` 做**任何**修改后都要再次运行它。
- 只有在 `.codex/` 层受信任时，项目本地钩子才会加载。在不受信任的检出中，Codex 仍会加载用户级和系统级钩子，但不会加载这些。

这与 Coddy 对项目本地 `.coddy/mcp.json` 施加的工作区信任门禁一致（参见 [MCP Integration](../features/mcp.md)）：仅仅打开仓库不应就此授予其代码执行权限。

## 验证与调试

首先运行仓库适配器契约测试：

```bash
make test-agent-rules
```

它覆盖标量和 YAML 列表形式的 `globs`、结构化编辑路径、提供商规则激活、OpenCode 投递，以及不存在手工 Codex 索引这一点。

可以用合成负载手动驱动该脚本。它从 stdin 读取 JSON 并向 stdout 写出 JSON，因此无需 Codex 会话：

```bash
echo '{"hook_event_name":"PreToolUse","session_id":"probe","tool_input":{"command":"*** Begin Patch\n*** Update File: external/httpserver/server.go\n*** End Patch"}}' | python3 .codex/hooks/attach_rules.py
```

会话启动行为：

```bash
echo '{"hook_event_name":"SessionStart","session_id":"probe","source":"startup"}' | python3 .codex/hooks/attach_rules.py
```

空输出是有效结果，意味着三种情况之一：没有 glob 匹配、该规则在本会话中已发送过，或输入未被理解。清除去重状态以重新测试：

```bash
rm -rf "${TMPDIR:-/tmp}"/codex-attach-rules-*
```

如果钩子在真实会话中没有任何产出，请先检查 `/hooks`。信任通常是原因。

## 添加或修改规则

像以前一样编辑 `.cursor/rules/*.mdc`。新增或重命名主题时钩子无需改动，因为它会在每次事件时读取该目录。

- 只有当规则的 frontmatter 带有 `globs` 或 `alwaysApply: true` 时，Codex 才能触达该规则；
- 在同一次改动中更新对应的 Claude Code 版本，并保持正文等价；
- 在修改规则 frontmatter 或任一适配器文件后运行 `make test-agent-rules`。

## 不要与 Codex `.rules` 混淆

Codex 还将 "rules" 一词用于第二种无关机制：写在活动配置层旁边 `rules/` 目录下、用 Starlark 编写的 `.rules` 文件，它们决定**哪些 shell 命令可以在沙箱之外运行**。那是执行策略而非指令，并且通过以下命令校验：

```bash
codex execpolicy check --pretty --rules ~/.codex/rules/default.rules -- go test ./...
```

本仓库不包含 `.codex/rules/` 目录；命令策略属于每位开发者的个人事务，位于 `~/.codex/rules/`。请注意，只有当 `bash -lc` 脚本是由 `&&`、`||`、`;` 或 `|` 连接的简单词语链时，Codex 才会将其拆分为独立命令。包含 `VAR=x`、`$(...)`、通配符或控制流的脚本会作为一个不透明调用进行匹配，这就是自动记录的允许列表条目在之后的任何命令中往往不可用的原因。

## 其他智能体中的等效设置

| 智能体 | 常开规则 | 作用域规则 |
|---|---|---|
| Cursor | `.cursor/rules/*.mdc` 中的 `alwaysApply: true` | 同一 frontmatter 中的 `globs` |
| Claude Code | `CLAUDE.md`（此处符号链接到 `AGENTS.md`） | 原生 `.claude/rules/*.md` 路径作用域 |
| Codex | `AGENTS.md` 加上 `SessionStart` 钩子 | `PreToolUse` 钩子，即本页 |

上游参考：[Hooks](https://developers.openai.com/codex/hooks) 和 [Custom instructions with AGENTS.md](https://developers.openai.com/codex/guides/agents-md)。

<!-- docsgen:source sha256=0978ba409a7d580e -->

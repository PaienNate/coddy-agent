# 用于项目规则的 ZCode 钩子

本页面面向**在本仓库上运行 ZCode 客户端的贡献者**。它是[用于项目规则的 Codex 钩子](codex-hooks.md)的 ZCode 对应版本；两者都通过各自客户端的钩子机制，将相同的 `.cursor/rules/*.mdc` 文件提供给模型。

## 为什么需要钩子

ZCode 每个会话加载一次其指令文件（`AGENTS.md`），从工作目录向上搜索至项目根目录。由此产生的两个后果与 Codex 的情况相同：

- 在仓库根目录启动的会话永远不会加载嵌套的 `AGENTS.md`；
- 不存在基于 glob 的附加机制。ZCode 没有 `.cursor/rules/*.mdc` 中 `globs` 字段的等价物，因此作用域只能通过文件在目录树中的位置来表达。

本仓库在 `.cursor/rules/*.mdc` 下保留原生 Cursor 表示，ZCode 适配器直接读取这些文件。Cursor 目录树是适配器的输入，而不是通用策略格式。让模型指向另一个文件并要求其读取，仍然只是建议性的，可能被跳过。

`.zcode/hooks/attach_rules.py` 通过将规则正文直接注入会话，弥补了作用域规则的缺口，而无需维护单独的 ZCode 副本。

## 触发时机

| 事件 | 匹配器 | 注入内容 |
|---|---|---|
| `SessionStart` | `startup\|resume\|clear\|compact` | frontmatter 包含 `alwaysApply: true` 的每条规则 |
| `PreToolUse` | `^(Edit\|Write\|MultiEdit\|ApplyPatch)$` | 其 `globs` 覆盖待处理工具调用中某个文件的规则 |

配置位于 `hooks.events` 下的 `.zcode/config.json` 中。配置文件钩子默认处于禁用状态，因此配置中设置了 `hooks.enabled: true`。两个处理程序运行同一个脚本；脚本根据钩子负载中的 `hook_event_name` 进行分支。`ApplyPatch` 别名包含在 `PreToolUse` 匹配器中，因为 ZCode 会将 `Write`/`Edit` 设为 `ApplyPatch` 的别名。

对于本仓库，`russian-wording` 和 `merged-branch-cleanup` 会在会话启动时到达。Go、HTTP、网关和 UI 主题规则是路径作用域的，并按需附加。一次涉及 `external/httpserver/server.go` 的编辑会引入 `api-layer`、`architecture`、`code-style`、`implementation-order`、`testing` 和 `workflow`；涉及 `external/ui/src/` 的编辑还会引入 `ui-spa` 和 `ui-verification`。

## 工作原理

项目适配器自行解析 `.mdc` frontmatter（`description`、标量或列表 `globs`，以及 `alwaysApply`）。在有效的标量、单行或多行 flow-list 以及 block-list 形式中，带引号的逗号、双写的单引号、双引号转义和 YAML 注释会按需被解码或剥离。它读取 `.cursor/rules/` 作为输入，并且不保留规则正文的单独 ZCode 副本。新的 Cursor 规则文件无需适配器接线。值得了解的要点：

- **Glob 匹配为手工实现。** `fnmatch` 不可用，因为其 `*` 也会跨越 `/`，这导致 `external/httpserver/**/*.go` 无法匹配 `external/httpserver/server.go`。该脚本将 glob 转换为正则表达式，其中 `**/` 变为“零个或多个目录”，`*` 保持在单个段内，而单独的 `**` 可跨越任意内容。
- **路径来自结构化工具字段。** ZCode 向钩子传递一个 JSON 对象，因此脚本会遍历 `tool_input`，并保留已知携带路径的键（`file_path`、`path`、`notebook_path`、`source`、`destination`、`new_path`、`old_path` 等）下的字符串值。Codex 对应版本也接受这些字段，并额外解析 `apply_patch` 头部。嵌套路径和绝对路径在匹配前会被规范化。
- **每个作用域规则在每个活动会话上下文中最多注入一次。** 状态是 `<tempdir>/zcode-attach-rules-<repository-hash>/<session_id>.json` 下的一个 JSON 文件，以仓库和会话 id 为键，因此克隆之间不会互相抑制。`resume` 保留作用域去重；`startup`、`compact` 和 `clear` 会将状态重置为始终启用的集合。并发进程通过每会话锁串行化投递，状态更新使用原子替换。规则 id 仅在上下文输出成功后才会持久化，因此输出失败会使其仍可在后续尝试中投递。如果锁定或持久化不可用，匹配的规则上下文仍会输出，去重可能会临时退化为重复投递。
- **它以失败开放方式运行。** stdin 上格式错误的 JSON、无法解析的规则文件，或不可写的状态目录，都会以退出码 0 结束且不产生输出。损坏的规则永远不会阻止编辑。

## 启用

在设置 `hooks.enabled: true` 之前，ZCode 配置文件钩子处于禁用状态，因此已提交的 `.zcode/config.json` 已经启用了运行器。与 Codex 不同，这里没有逐钩子的信任门禁：带有 `enabled: true` 的工作区配置会无条件运行，这正是仓库共享设置的预期行为。在 ZCode 中打开该仓库，两个处理程序即处于活动状态；无需其他操作。

## Windows 解释器

钩子命令调用的是 `python`，而不是 `python3`。在 Windows 上，`python3` 名称通常是 Microsoft Store 存根，不打印任何内容并以非零状态退出，而真正的解释器是 `python`（或 `py` 启动器）。在类 Unix 系统上，`python` 通常也会存在；如果系统仅提供 `python3`，请将 `.zcode/config.json` 中的命令指向它。`${ZCODE_PROJECT_DIR}` 模板变量在 ZCode 会话内会展开为仓库根目录。

## 验证与调试

首先运行仓库适配器契约测试：

```bash
make test-agent-rules
```

它覆盖标量和 YAML 列表 `globs`、结构化编辑路径、提供商规则激活、OpenCode 投递，以及不存在手动 Codex 索引的情况。

使用合成负载手动驱动该脚本。它从 stdin 读取 JSON 并向 stdout 写入 JSON，因此无需 ZCode 会话：

```bash
echo '{"hook_event_name":"PreToolUse","tool_name":"Edit","session_id":"probe","tool_input":{"file_path":"external/httpserver/server.go"}}' | python .zcode/hooks/attach_rules.py
```

会话启动行为：

```bash
echo '{"hook_event_name":"SessionStart","session_id":"probe","source":"startup"}' | python .zcode/hooks/attach_rules.py
```

空输出是有效答案，表示以下三种情况之一：没有 glob 匹配、该规则已在本会话中发送过，或输入未被理解。清除去重状态以重新测试：

```bash
rm -rf "${TMPDIR:-/tmp}"/zcode-attach-rules-*
```

要针对一次性规则目录和状态进行探测，而不触及会话共享的默认值，请设置 `ZCODE_RULES_DIR` 和 `ZCODE_RULES_STATE_DIR`。

## 添加或更改规则

像以前一样编辑 `.cursor/rules/*.mdc`。添加或重命名主题时，钩子无需更改，因为它会在每次事件时读取该目录。

- 只有当规则的 frontmatter 带有 `globs` 或 `alwaysApply: true` 时，该规则才能从 ZCode 访问；
- 在同一次更改中更新有意保留的 Claude Code 对应版本，并保持正文等价；
- 更改规则 frontmatter 或任一 Python 适配器后，运行 `make test-agent-rules`。

## 与 Codex 钩子的关系

此脚本与 `.codex/hooks/attach_rules.py` 共享相同的目标及大部分逻辑（`parse_rule`、`glob_to_regex`、每会话状态、`SessionStart`/`PreToolUse` 拆分、失败开放）。两者都接受已知的结构化路径字段；Codex 变体还会额外解析 `apply_patch` 头部。保持共享逻辑同步，并通过 `make test-agent-rules` 覆盖两个适配器。

上游参考：ZCode 钩子和配置位于 `~/.zcode/cli/config.json`（用户作用域）和 `<repo>/.zcode/config.json`（工作区作用域）下；支持的事件包括 `SessionStart`、`UserPromptSubmit`、`PreToolUse`、`PermissionRequest`、`PostToolUse`、`PostToolUseFailure` 和 `Stop`。

<!-- docsgen:source sha256=9172be100b98e68f -->

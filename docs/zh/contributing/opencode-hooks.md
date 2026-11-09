# 面向项目规则的 OpenCode 钩子

本页面面向在本仓库上运行 OpenCode 的贡献者。它不描述 Coddy 自身的规则发现机制：`coddy` 二进制通过 `{{.Rules}}` 注入这些规则，详见 [规则](../features/rules.md)。

## 为什么需要插件

OpenCode 会读取根目录的 `AGENTS.md`，因此仓库地图和始终相关的贡献者说明已经能够进入会话。OpenCode 也可以从 `opencode.json` 加载指令 glob，但每个匹配的指令文件都会被合并进提示词；Cursor 按文件生效的 `globs` 语义不会应用于 `.mdc` frontmatter。

本仓库在 `.cursor/rules/*.mdc` 中保留了一份原生 Cursor 表示。位于 `.opencode/plugins/project-rules.js` 的项目插件直接读取该 frontmatter，并且只附加当前会话所需的规则。Cursor 目录树是适配器的输入，而不是通用的策略格式，也不会为规则正文额外维护一份 OpenCode 副本。

OpenCode 会自动加载 `.opencode/plugins/` 下的 JavaScript 和 TypeScript 文件。此插件不需要 `opencode.json` 条目或安装任何包。它只使用 OpenCode 运行时提供的、兼容 Node 的 API。

## 钩子映射

| OpenCode 钩子 | 行为 |
|---|---|
| `experimental.chat.system.transform` | 在每次模型请求中附加所有 `alwaysApply: true` 规则以及该会话当前激活的作用域规则。 |
| `tool.execute.before` | 从工具参数和补丁头部中提取文件路径，然后激活 `globs` 与这些路径匹配的规则。 |
| `experimental.session.compacting` | 将选中的规则加入压缩上下文，使其在续接摘要中保持明确。 |
| `session.deleted` 事件 | 释放被删除会话在内存中的作用域规则集合。 |

读取受规则约束的文件会为其作用域规则在下次模型请求中激活。这通常意味着规则会在模型编辑该文件之前就已就位。

OpenCode 稳定的 `tool.execute.before` 钩子可以阻止或修改工具调用，但无法向当前调用添加模型可见的消息。因此，如果 `edit`、`write`、`apply_patch` 或其他可识别的写入工具是首个暴露某条作用域规则的操作，插件会故意拒绝该调用一次。错误信息会要求 OpenCode 重试；随后的模型请求会通过 `experimental.chat.system.transform` 收到新激活的规则，重复的编辑则会被允许。这样一来，首次写入就是确定性的，而无需依赖模型去打开某个被引用的规则文件。

## 规则解析与匹配

该实现从 Cursor frontmatter 中读取 `description`、`globs` 和 `alwaysApply`。`globs` 可以是标量、单行或多行流式列表，也可以是 YAML 块列表；带引号的逗号、成对的单引号、双引号转义和 YAML 注释都会被正确解码，而不会截断列表。一个 glob 转换器会将 `*` 限制在单个路径段内，并将 `**/` 视为零个或多个目录，因此 `external/httpserver/**/*.go` 既能匹配 `external/httpserver/server.go`，也能匹配更深层的文件。

工具参数会被递归检查。可识别的路径键包括 `filePath`、`path`、`filename`、`target` 及其常见变体。统一补丁负载可识别 `*** Add File:`、`*** Update File:`、`*** Delete File:` 和 `*** Move to:` 头部。仓库之外的绝对路径会被忽略。

每个相关钩子都会重新读取规则，因此编辑或新增 `.mdc` 文件无需改动插件接线。格式错误或无法读取的规则文件会被跳过，而不会使其余规则失效。

## 会话状态与压缩

作用域激活状态由 OpenCode 的插件进程保存在内存中，并以会话 id 为键。它能在常规压缩后继续存在，并在 OpenCode 报告 `session.deleted` 时被清除。重启 OpenCode 会清空此缓存。重启后，读取受规则约束的文件会再次激活其规则；直接进行首次编辑则可能再次触发一次性的重试保护。

插件会刻意在每次系统变换中都加入选中的规则，而不是跟踪其文本是否已经发送过。压缩之后，模型请求彼此独立，而重复的系统上下文是持久约束的可靠放置位置。

## 验证

仓库的适配器测试套件覆盖了标量与 YAML 列表 frontmatter、常驻注入、递归 Cursor glob 匹配、由 `read` 触发的激活、首次写入重试、补丁路径提取、Python 钩子一致性以及会话清理：

```bash
make test-agent-rules
```

若只想迭代 OpenCode 部分，请运行：

```bash
make test-opencode-rules
# or: node --test .opencode/tests/project-rules.test.js
```

这些测试不需要提供商连接，也不需要安装 OpenCode。要验证真实的集成效果，请在本仓库中启动一个新的 OpenCode 会话，让它读取 `external/httpserver/server.go`，并使用 OpenCode 调试日志检查下一次模型请求。

## 兼容性与安全

系统变换和压缩钩子目前被 OpenCode 标记为 `experimental`。升级 OpenCode 时，请重新运行单元测试套件并做一次真实的冒烟测试。

项目插件是可执行代码，OpenCode 会自动加载它们。打开不受信任的检出内容时，请检查 `.opencode/plugins/`。此插件不会启动命令、访问网络或加载第三方依赖；它只读取当前 worktree 内的规则文件，并在内存中保留每个会话的规则名称。

上游参考：[OpenCode 规则](https://opencode.ai/docs/rules/) 和 [OpenCode 插件](https://opencode.ai/docs/plugins/)。

<!-- docsgen:source sha256=4be3b5256f338025 -->

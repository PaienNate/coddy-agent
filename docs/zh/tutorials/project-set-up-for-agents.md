# 为智能体设置项目

一个仓库可以承载智能体在其中出色工作所需的一切：约定、工作流、工具服务器和防护栏。所有这些都存放在检出目录下的文件里，因此与代码一同纳入版本控制，并随每次克隆一起到达。下面是接下来的步骤所生成的目录树，以及每个文件的作用：

```text
my-project/
  AGENTS.md                       # the second document layer, in every prompt
  .coddy/
    rules/
      go-style.md                 # a rule, attached once a Go file comes into play
    skills/
      release-notes/
        SKILL.md                  # /release-notes, project-local
    mcp.json                      # project MCP servers, held until approved
    hooks.json                    # lifecycle hooks, held until approved
    hooks/
      gofmt.sh                    # the script hooks.json runs
    agents/
      reviewer.md                 # a subagent definition, held until approved
```

1. **根目录下的 `AGENTS.md`。** 它会被无条件读入每一次提示词，作为你自己的文档之后的第二层文档（[AGENTS.md 与 DESIGN.md](../features/rules.md#agentsmd-与-designmd)），它旁边的 `DESIGN.md` 也是如此。让它保持为一份简短的地图：东西放在哪里，如何构建和测试，一次变更必须携带什么。树中更深处的嵌套 `AGENTS.md` 文件按需读取，在工具进入其所在文件夹的那一刻读取，并且从不主动遍历。

2. **`.coddy/rules/` 中的规则。** `.md` 文件是 Claude Code 规则（`paths` 会为其设置门控，没有它们时则是无条件的）；`.mdc` 文件是 Cursor 规则（`description`、`globs`、`alwaysApply`）。每个规则文件夹都接受这两种方言（[规则文件格式](../features/rules.md#规则文件格式)）。Coddy 只读取一个项目文件夹：项目中有 `.coddy/rules` 时就用它，否则用 `.agents/rules`、`.cursor/rules`、`.claude/rules` 和 `.codex/rules` 中第一个存有规则文件的，因此为其他智能体镜像的规则不会被重复加载（[发现](../features/rules.md#发现)）。本身就是独立项目的子文件夹也可以保留一个规则文件夹：当智能体在那里访问到文件时就会读取它，并且只作用于该子文件夹（[子文件夹的规则文件夹](../features/rules.md#子文件夹的规则文件夹)）。带门控的规则只会到达模型一次，伴随首次触及匹配文件的工具结果或消息。

   本教程中的一切都是项目在说话。*你*希望从每个会话、每个仓库中得到什么 - 用来回答的语言、你永远不想被运行的命令 - 属于你自己而不是这个检出目录：把它们放进 `~/.coddy/AGENTS.md` 和 `~/.coddy/rules/`，在那里它们处处适用，也不需要别人认同（[你自己的指令和规则](../features/rules.md#你自己的指令与规则)）。

   ```markdown
   ---
   description: Go coding standards
   paths:
     - "**/*.go"
   ---

   Write comments in English. Wrap errors with fmt.Errorf("context: %w", err).
   Run `make lint` before you report a change as done.
   ```

   `coddy rules list` 会打印出目录清单，包含来源文件夹、每个文件读取时所用的方言、它是否出现在每一次提示词中，以及是什么激活了其余文件。

3. **`.coddy/skills/<name>/SKILL.md` 中的技能。** `.coddy/skills` 是 Coddy 始终读取的四个文件夹中最强的一个，因此项目技能会覆盖来自 `~/.coddy/skills`、项目的 `.agents/skills` 或 `~/.agents/skills` 的同名技能；只有你添加到 `skills.dirs` 的目录排在它之后（[技能](../features/skills.md)）。目录名就是斜杠命令，这里是 `/release-notes`；编写该文件是本页最后一个配方。在项目中运行 `coddy skills list` 即可看到它为那个目录列出。

4. **`.coddy/mcp.json` 中的 MCP 服务器**（[MCP 服务器](../features/mcp.md)）。该文件采用 Cursor 的结构，一个以名称为键的 `mcpServers` 对象；仅有 URL 的条目是可流式传输的 HTTP 服务器，而 `args` 中的 `${CWD}` 是会话工作区。

   ```json
   {
     "mcpServers": {
       "filesystem": {
         "command": "npx",
         "args": ["-y", "@modelcontextprotocol/server-filesystem", "${CWD}"]
       },
       "docs": { "url": "https://mcp.example.com/mcp" }
     }
   }
   ```

   由于该文件随检出目录一起到达，将会是仓库（而非操作者）来选择会话启动时执行的命令。在默认的 `mcp.project_trust: ask` 下，这些条目保持冷状态，直到为该工作区批准为止（[项目本地服务器的工作区信任](../features/mcp.md#面向项目本地服务器的工作区信任)）：

   ```bash
   coddy mcp list                # scope, trust state and the command of every merged server
   coddy mcp trust filesystem    # prints the declaration once more, then records the receipt
   ```

   `~/.coddy/mcp-trust.json` 中的凭据以工作区和声明的摘要为键，因此重写条目会再次询问；`config.yaml` 和 `~/.coddy/mcp.json` 属于你，永远不会被门控。

5. **`.coddy/hooks.json` 中的钩子**（[钩子](../features/hooks.md)）。该文件使用 Claude Code 的结构。这个钩子在每次编辑后运行 `gofmt`，并告诉模型它改动了什么：

   ```json
   {
     "hooks": {
       "PostToolUse": [
         {
           "matcher": "edit|write|apply_patch",
           "hooks": [{ "type": "command", "command": ".coddy/hooks/gofmt.sh" }]
         }
       ]
     }
   }
   ```

   ```bash
   #!/bin/sh
   # .coddy/hooks/gofmt.sh - runs in the session cwd with the call's JSON on stdin
   path=$(jq -r '.tool_input.path // ""')
   case "$path" in
     *.go)
       if out=$(gofmt -l "$path" 2>&1) && [ -n "$out" ]; then
         gofmt -w "$path"
         printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"gofmt reformatted %s"}}' "$path"
       fi
       ;;
   esac
   exit 0
   ```

   钩子以你的权限运行，且在任何权限提示之前运行，因此随克隆而来的文件会被解析并列出，但在获得批准之前不会运行任何东西（[项目文件与信任](../features/hooks.md#项目文件与信任)）；发现它的第一个轮次会在会话中记录一条通知。检查这些命令，然后：

   ```bash
   coddy hooks list
   coddy hooks trust .coddy/hooks.json
   ```

6. **`.coddy/agents/reviewer.md` 中的子智能体**（[定义文件](../features/subagents.md#定义文件)）。只有 `description` 是必需的；`name` 默认为文件主干名。`tools`、`disallowed_tools` 和 `permission_mode` 只能收窄父级原本可以做的事，而正文是子级的角色。

   ```markdown
   ---
   name: reviewer
   description: Reviews a diff or a module for defects and reports findings with file paths and lines.
   tools: read, glob, grep, print_tree, run_command
   permission_mode: ask
   max_turns: 20
   ---
   You review code. Read before you judge, cite `path:line` for every finding,
   separate defects from style remarks, and end with a short verdict.
   ```

   项目定义遵循 `subagents.project_trust`，并拥有自己的凭据（[作用域与项目信任](../features/subagents.md#作用域与项目信任)）：

   ```bash
   coddy agents list
   coddy agents trust reviewer
   ```

7. **每个克隆批准一次，或设置一条策略。** 上面三项批准是按工作区和按文件摘要的，因此全新的克隆或被编辑过的文件会再次询问。对于你自己拥有的检出目录，在 `config.yaml` 中把 `mcp.project_trust`、`hooks.project_trust` 和 `subagents.project_trust` 设为 `allow` 即可跳过凭据；`deny` 则完全不读取项目文件。在 `--remote` 下，批准属于服务器，也就是代码将要运行的地方（下一个配方）。

<!-- docsgen:source sha256=145dce66f0de88c1 -->

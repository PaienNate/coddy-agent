# 你自己的技能

技能是一个 `SKILL.md` 文件，由 frontmatter 和一段指令正文组成；它会在每个界面上成为一个斜杠命令，而它的正文只有在被调用时才会送达模型。参考资料见 [Skills](../features/skills.md)，尤其是 [Writing your own skill](../features/skills.md#编写自己的技能) 和 [Supported file formats](../features/skills.md#支持的文件格式)。

1. **创建技能包。** 每个技能占一个目录，位于 `~/.coddy/skills/` 下；目录名就是斜杠命令，所以要保证 `name` 与它一致。`description` 是目录和设置页面显示的那一行，可选的 `version` 会由 `coddy skills list` 显示，并用于检测从某个来源安装的技能的更新。

   ```markdown
   ---
   name: release-notes
   description: Turns the commits since the last tag into release notes grouped by area.
   version: 1.0.0
   ---

   # Release notes

   Run `git describe --tags --abbrev=0` to find the last tag, then `git log <tag>..HEAD --oneline`.
   Group the commits by scope, drop merge commits, write one line per change in the imperative
   mood, and end with a "Breaking changes" section, or "None" when there are none.
   ```

   保存为 `~/.coddy/skills/release-notes/SKILL.md`，即 Coddy 始终读取的四个文件夹中的第三个。按优先级从低到高依次为：`~/.agents/skills/`（你自己的，与 `npx skills` 及其他智能体共享）、项目的 `.agents/skills/`、`~/.coddy/skills/`、项目的 `.coddy/skills/`；当两个技能同名时，靠后的文件夹胜出，任何其他文件夹都可以按顺序加到 `config.yaml` 中的 `skills.dirs` 里，排在这四个之后，且优先级高于它们。

2. **在目录中看到它。** `coddy skills list` 会打印搜索根目录，以及一张包含技能、其版本、`enabled` 或 `disabled` 和描述的表格。

   ```bash
   coddy skills list
   ```

3. **把它当作斜杠命令找到。** 在控制台编辑器的第一行输入 `/`，建议列表中就会包含 `/release-notes`；Web 界面的输入框会从 `GET /coddy/slash-commands` 中列出它，而 ACP 编辑器在 `session/new` 之后会通过 `available_commands_update` 收到它。当一条消息中包含 `/release-notes` 时，正文会被写入该消息一次，作为你文本之后的附件，模型会在该轮次及其后的轮次中在那里读取它；对话记录显示的仍是你输入的消息原文。当 `skills.auto_discovery` 保持默认值时，只要请求与描述匹配，模型也可以通过 `load_skill` 工具自行引入该技能。

4. **测试它。** 加载器会在每个提示词时重新扫描 `skills.dirs`，因此编辑、保存后再次发送即可；不涉及重启。打印模式可以在带有标签的仓库中提供一个快速循环：

   ```bash
   cd ~/src/my-project
   coddy -p "/release-notes for everything since the last tag"
   ```

5. **关闭它而不删除它。** `coddy skills disable release-notes` 会保留文件并去掉该命令；`coddy skills enable release-notes` 会把它恢复。状态以每行一个名称的形式保存在 `~/.coddy/skills/.disabled` 中。

   ```bash
   coddy skills disable release-notes
   coddy skills enable release-notes
   ```

6. **分享它。** 把该目录发布到 GitHub 仓库。其他人用 `coddy plugin install owner/repo` 安装它，`coddy plugin update` 会把他们带到你之后提升的版本，而在 [skills.sh](https://skills.sh) 上的一条列表能让它通过 `npx skills find` 被找到。

<!-- docsgen:source sha256=a879a05ca7ac1bbd -->

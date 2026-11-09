# 团队 Telegram 机器人

一个 `coddy serve` 进程轮询一个机器人，在你允许的聊天中回答，并把每段对话保留为普通会话，同一进程上的 Web 界面可以查看并继续这些会话。完整参考见 [Telegram 网关](../surfaces/gateway.md)；本教程演示团队群聊的设置。

1. **获取带网关的二进制。** 发布二进制、软件包和镜像都带有 `gateway` 标签。从源码构建时，可以只构建 Telegram 适配器，或构建完整集合：

   ```bash
   make build TAGS="gateway.telegram"
   # or the set a release ships
   make build TAGS="http ui scheduler memory cli gateway swarm metrics browser"
   ```

   在未带该标签构建的二进制中启用机器人会导致启动错误并指出该标签，因此不匹配不会被忽略。

2. **获取令牌和 ID。** 使用 `/newbot` 在 [@BotFather](https://t.me/BotFather) 创建机器人并保存令牌。向 [@userinfobot](https://t.me/userinfobot) 询问你的 Telegram 用户 ID，以及你希望加入允许列表的每位队友的 ID。不要将令牌放进 `config.yaml`：`~/.coddy/.env` 会在文件加载前读取，且进程环境始终优先于它。

   ```text
   TELEGRAM_BOT_TOKEN=8992982910:AAF...
   ```

3. **在配置中描述团队**（[配置参考](../surfaces/gateway.md#配置参考)）。

   ```yaml
   gateways:
     telegram:
       enable: true
       token: "${TELEGRAM_BOT_TOKEN}"   # may be omitted: an empty token reads TELEGRAM_BOT_TOKEN
       admins: [123456789]              # your own user id; admins pass every access check
       default_access: "admins"         # all | admins | group:<name>
       default_isolation: "shared"      # individual | shared | admin (group chats only)
       user_groups:
         - name: "devs"
           user_ids: [111222333, 444555666]
       chats:
         - chat_id: -1001234567890      # the team's group: every dev, one shared session
           access: "group:devs"
           isolation: "shared"
   ```

   `default_access` 指定在没有覆盖设置的聊天中谁可以与机器人对话：`admins` 只回应列出的 ID，`all` 回应任何能向该聊天写入的人，`group:<name>` 回应那个 `user_groups` 条目的成员（管理员始终通过）。被拒绝的消息会被静默丢弃。`default_isolation` 仅适用于群聊，私聊始终是每个用户一个会话，并选择会话范围：`individual` 为每位成员提供各自的会话，`shared` 为整个群聊提供一个所有人共同延续的会话，`admin` 只回应管理员并让他们共享一个会话。`chats` 按聊天覆盖这两项设置；群组的 `chat_id` 为负数。在 Bot API 10.1 服务器上，添加 `rich_messages: true` 以启用原生 Markdown 和可折叠的工具块。

4. **检查文件，然后在后台启动**（[coddy serve 与守护进程](../operate/serve.md)）。

   ```bash
   coddy serve --dry-run   # checks the token against the Bot API and names the bot
   coddy serve --daemon    # or -d
   coddy serve status
   ```

   `--daemon` 会分离并留下一个调度器，只要工作进程因 `coddy serve stop` 之外的任何原因死亡，调度器就会重启它。该命令会在返回前等待第一个工作进程，因此无法启动的配置会在输入该命令的终端中报告。记录位于 `~/.coddy/serve.json`，日志位于 `~/.coddy/logs/serve.log`；`coddy serve restart` 会用它启动时的参数将其恢复。在 `systemd`、Docker 或其他已经掌管进程生命周期的监管程序下，改为在前台运行 `coddy serve`，并让该监管程序重启它。每个机器人令牌运行一个进程：Telegram 将每个更新交给一个长轮询，因此第二个轮询器会从第一个那里抢走消息。仅运行机器人、不包含其他内容时，使用 `--http=false` 或 `httpserver.enable: false`。

5. **在浏览器中打开同一个聊天**（[聊天与浏览器中的同一会话](../surfaces/gateway.md#聊天和浏览器中的同一会话)）。在开启 `httpserver.enable`（默认开启）的情况下，同一进程在 `http://127.0.0.1:12345/` 提供 Web 界面。聊天对话是普通会话，与浏览器标签页或终端启动的会话类型相同，ID 类型也相同：它们会列在会话栏中，机器人正在回答的轮次会流式传输到正在观看它的标签页，而在浏览器中输入的回复会在下一条消息时出现在聊天历史中。权限提示留在聊天中，网关会自行回答它们，因此机器人可以无人值守地工作。所以，给该进程一个你可以承受其被改动的工作区（`coddy serve --cwd DIR`，否则就是它启动时所在的目录）。每个会话只运行一个轮次；当浏览器轮次正在进行时到达的消息会收到忙碌提示，而不是交错执行。要从另一台机器访问界面，绑定 `httpserver.host: "0.0.0.0"` 并按下一篇教程设置令牌。

6. **从聊天中使用它**（[机器人交互模型](../surfaces/gateway.md#群聊)）。在私聊中，每条消息都会被回应。在群聊中，只有当机器人被 @ 提及、消息回复它自己的某条消息，或发给 `/clear` 时，机器人才会响应；在 `admin` 隔离下，它还会忽略 `admins` 之外的任何人。`/agent`、`/plan` 和 `/ask` 切换会话模式，`/model` 打开一个基于已配置模型的内联键盘，`/context` 显示上下文窗口中被占用的内容，`/clear` 为该用户或聊天启动一个新会话；旧会话包仍保留在磁盘上，`coddy sessions list` 仍会显示它。更改令牌会就地重建网关，切换 `enable` 会在无需重启的情况下启动或停止它；当点击似乎没有作用时，提高某个组件的日志级别：`coddy serve --log-level "info,gateway.telegram=debug"`（[调试聊天](../surfaces/gateway.md#调试聊天)）。

<!-- docsgen:source sha256=da9ff995b4b6e934 -->

# 在 VS Code Copilot 中将 Coddy 作为模型

VS Code Copilot 通过 `chatLanguageModels.json`（管理模型，然后选择自定义端点）接受任何兼容 OpenAI 的端点作为聊天模型。把它指向一个正在运行的 `coddy serve`，就能让 coddy 的模型以及 coddy 的智能体出现在 Copilot Chat 的模型选择器中。Copilot 读取的流是 [两种流式传输方言](../reference/http-api.md#两种流方言) 中描述的严格版本；模型 id 就是 `GET /v1/models` 列出的那些（[HTTP API](../reference/http-api.md)）。

1. **在 VS Code 能访问到的地址上提供服务。** 默认绑定是端口 12345 上的回环地址，对于同一台机器上的 VS Code 来说已经足够。如果 VS Code 在别处，就按照 [从笔记本电脑驱动服务器](server-driven-from-a-laptop.md) 的做法绑定并保护服务器，并把令牌以 `apiKey` 的形式交给 VS Code。

   ```bash
   coddy serve
   curl -s http://127.0.0.1:12345/v1/models | jq -r '.data[].id'   # the ids step 2 takes
   ```

2. **注册端点。** 每个模型 id 对应一个条目。`models[].model` 类型的 id 表示由 coddy 代替提供商：Copilot 自己的工具会被提供给模型，其调用会回到 Copilot，由 Copilot 在编辑器中执行，因此 VS Code 的 Agent 模式的表现与直接对接提供商时一致；把 `toolCalling` 设为开启，当该条目在 `config.yaml` 中有 `multimodal: true` 时再开启 `vision`，这正是让图片能够通过的原因。`agent` 类型的 id 表示把 coddy 的 ReAct 智能体当作模型：它用自己的工具读取并编辑 `coddy serve` 启动时所在的工作区，忽略 Copilot 提供的工具，Copilot 只能看到最终答案，因此在该条目上保持 `toolCalling` 关闭，并从 VS Code 的 Ask 模式使用它。

   ```json
   {
     "name": "Coddy",
     "vendor": "customendpoint",
     "apiKey": "coddy",
     "apiType": "chat-completions",
     "models": [
       {
         "id": "neuraldeep/qwen3.8-27b",
         "name": "Coddy · Qwen 3.8 27B",
         "url": "http://127.0.0.1:12345/v1/chat/completions",
         "toolCalling": true,
         "vision": true,
         "maxInputTokens": 128000,
         "maxOutputTokens": 8192
       },
       {
         "id": "agent",
         "name": "Coddy · agent",
         "url": "http://127.0.0.1:12345/v1/chat/completions",
         "toolCalling": false,
         "maxInputTokens": 128000,
         "maxOutputTokens": 8192
       }
     ]
   }
   ```

   Copilot 会在每个请求中发送它自己的 `temperature`，而直接模型会用这个值替代配置中设定的值。推理模型可能会拒绝它：`codex` 模型总是拒绝（coddy 在调用它之前会先返回 `400`），OpenAI 的推理模型和 Anthropic 的扩展思考同样会拒绝 Copilot 发送的值，而 vLLM 上的 Qwen3 思考则接受它。为每个进行推理的模型在其条目上设置 `"thinking": true` - 一个 `codex` 模型，或 `GET /v1/models` 行中带有 `reasoning_levels` 的模型 - 然后 Copilot 就会像对待任何推理模型那样不再发送 `temperature`。

   Bearer 认证默认关闭，任何 `apiKey` 字符串都能通过 VS Code 的表单校验；设置 `httpserver.auth_token` 后，`apiKey` 就是那个令牌。Copilot 每次请求都会发送完整对话，因此每个请求在服务器上都是一个独立会话，除非在模型上设置 `requestHeaders` 条目把 `X-Coddy-Session-ID` 固定为某个 `sess_<hex>` id，这样就能让各轮次保持在同一个对话记录中。

3. **注意智能体的权限关卡。** `agent` 模型在其会话的权限模式下运行工具，而权限提示无法从 Copilot 中回答：该轮次会一直等待一个永远不会到来的答复。新会话会以服务器上最后选择的模式启动，因此当工作区是你放心让智能体操作的时候，先在那里选择一次 `bypass` - 在该服务器的任意会话中执行 `/permissions bypass`，从 Web 界面或控制台都可以 - 或者在服务器启动前把该文件写入服务器的 `CODDY_HOME`（默认是 `~/.coddy`）。否则就保持 `ask`，并在 Web 界面中回答该提示，此时该轮次会在其会话下保持活动状态。

   ```bash
   echo '{"permissionMode": "bypass"}' > ~/.coddy/permission-mode.json
   ```

4. **确认它确实生效了。** 在 Copilot Chat 中选择 "Coddy · Qwen 3.8 27B" 并随便问点什么：答案会流式返回，而在推理模型下，思考过程会显示在 Copilot 自己的折叠区域中，因为 `reasoning_content` 是它会读取的一个字段。从终端发出同样的请求，可以看到 Copilot 所消费的流，其中一个 choice 在 `[DONE]` 之前以 `stop` 结束：

   ```bash
   curl -sN http://127.0.0.1:12345/v1/chat/completions -H 'Content-Type: application/json' \
     -d '{"model":"neuraldeep/qwen3.8-27b","messages":[{"role":"user","content":"hello"}],"stream":true}' | tail -4
   ```

   Copilot 中出现 "Response contained no choices." 表示服务器没有完成任何一个 choice：请检查 `coddy serve` 是否为流式输出 `finish_reason` 的版本（`coddy -v`，1.1.5 或更高版本），并确认两者之间没有任何东西改写该流。

<!-- docsgen:source sha256=ca07bd6399ce6554 -->

# Kimi

将 Kimi Code 订阅或 Kimi Platform API 密钥用作 Coddy 的模型提供商。

![提供商选择器中的 Kimi Code](../../assets/web-ui/kimi-onboarding-dark-1280.png)

*选择 Kimi 时默认使用 kimi.ai 上的 Kimi Code。*

![手机上提供商选择器中的 Kimi Platform](../../assets/web-ui/kimi-onboarding-platform-dark-390.png)

*Platform 模式要求提供 API 密钥。*

## 配置

一个 `type: kimi` 提供商有两种凭据模式。省略 `mode` 表示 `subscription`，用于 Kimi Code；`platform` 使用 Kimi Platform API 密钥。省略 `region` 表示 `global`；为中国大陆设置 `region: china`。逻辑模型需单独配置。

```yaml
providers:
  - name: kimi
    type: kimi
    mode: subscription
    region: global
```

对于 Platform，设置 `mode: platform` 并提供 `api_key` 或 `api_key_command`。订阅模式会拒绝 `api_key`、`api_key_command` 和 `api_base`。Platform 密钥不是 Kimi Code OAuth 令牌。

| 区域 | Kimi Code 授权 | Kimi Code API | Platform API |
| --- | --- | --- | --- |
| `global` | `https://auth.kimi.ai` | `https://api.kimi.ai/coding/v1` | `https://api.moonshot.ai/v1` |
| `china` | `https://auth.kimi.com` | `https://api.kimi.com/coding/v1` | `https://api.moonshot.cn/v1` |

模型目录为每个模型附带一个协议。对于订阅模型，Coddy 支持 Kimi、Anthropic 和 OpenAI Responses 传输；Platform 推理目前仅接受 Kimi 协议。目录元数据与模型、模式和区域绑定。协议不匹配时需要验证，而不是静默更改传输。

Kimi Code 只接受 temperature 为 1，无论哪个模型，都会以 HTTP 400 拒绝任何其他值。因此，对于写在 `models[]` 行上的 `temperature`，Coddy 会将其排除在订阅请求之外。API 客户端在请求本身上发送的 temperature 仍会到达端点，除非该值为 1，否则端点会拒绝它。Platform 是 Moonshot API，接受该参数，因此配置的 temperature 会到达它。

## 登录与 CLI 回退

Kimi Code 使用 OAuth 设备授权。运行 `coddy providers login kimi` 或从提供商表单登录。按照验证链接操作并输入显示的代码。对于 Web 界面中未保存的提供商，请在登录后保存设置，以便该提供商保留其身份。Platform 模式使用密钥，不使用此流程。

当不存在 Coddy 管理的凭据时，符合条件的订阅行可以读取所选区域的 Kimi CLI 访问令牌。符合条件的行是唯一的 Kimi 行，或者在存在多个时名为 `kimi` 的行。无效或不确定的托管凭据不会回退到 CLI 账户。Coddy 读取 CLI 凭据时不会刷新、修改或删除它。如果令牌过期或服务器以 HTTP 401 拒绝它，请在 Kimi CLI 中续订登录，或通过 Coddy 登录。过期时间未知的令牌会一直被尝试，直到服务器拒绝它。当 CLI 是唯一来源时，请从 Kimi CLI 本身退出登录。

CLI 凭据目录为 `KIMI_CODE_HOME`，未设置时为 `~/.kimi-code`。全球和中国大陆凭据分别占用其 `credentials` 目录下的不同文件。

## 用量与代理

当服务器提供时，订阅用量有四个独立窗口：五小时、七天、每月总计和每月代码。缺失的窗口保持未知。Booster 钱包数据会被省略；它不会显示为卢布余额。Platform 模式不提供订阅用量。

提供商请求使用 `providers[].proxy`，包括设备授权、轮询、刷新、模型发现、推理和用量。省略它以继承进程代理，设置 `none` 以直接连接，或提供代理 URL。

## 凭据安全

托管凭据存储在 `$CODDY_HOME/providers/kimi-identities/<identity>/` 下、`config.yaml` 之外，并按区域使用不同文件。在 Unix 上，这些文件仅所有者具有权限；在 Windows 上，它们继承存储目录的 ACL。提供商行中的非机密身份可防止删除后重建的行继承旧的托管登录。丢弃未保存的草稿可能会留下未使用的身份目录；在自行删除某个目录之前，请检查已配置的身份和活动草稿。Coddy 不会自动删除这些目录。将凭据目录视为机密。订阅目标受限于所选区域；不要将 OAuth 令牌发送到 Platform 端点。如果刷新结果不确定，请重新登录，而不是重试可能已轮换的令牌。来自不同进程的令牌更新使用共享锁。Windows 不保证目录更改在突然断电后仍然保留。

<!-- docsgen:source sha256=4de0ab4c34127299 -->

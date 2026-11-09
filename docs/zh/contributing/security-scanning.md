# AppSec 扫描

*开源扫描器（trivy、semgrep、govulncheck）如何在本地和 CI 中运行，报告落在哪里，如何阅读它们，以及门禁如何收紧。关于智能体自身的安全模型——它可以执行什么、权限模式、项目信任——请参阅[安全与信任](../operate/security.md)。*

Issue [#374](https://github.com/coddy-project/coddy-agent/issues/374) 要求在仓库中引入开源 AppSec 扫描器。只有一个脚本是调用它们的唯一位置，因此一个使流水线失败的发现，可以用一条命令在开发者机器上复现：

```bash
make security          # every scanner, the CI gate behaviour
make sec-trivy         # trivy only
make sec-semgrep       # semgrep only
make sec-govulncheck   # govulncheck only
make sec-report        # every scanner, gate off — findings never fail, errors still do
```

每个 target 都是围绕 `scripts/security-scan.sh` 的薄 `SEC_*` 包装；当你需要 target 未命名的组合时，直接调用该脚本。

## 运行内容

- **trivy**——对检出目录进行文件系统扫描：依赖漏洞（`go.mod`、`package-lock.json` 等）、密钥以及 IaC 错误配置（Dockerfile）。错误配置的发现会被收集，但位于**门禁之外**——它们属于策略问题，而非漏洞。
- **semgrep**——使用 `SEMGREP_CONFIGS` 中的 registry 规则包进行 SAST（默认 `p/golang p/typescript`）。目前仅报告：积压量足够大，第一天就设门禁会阻塞每一个拉取请求。
- **govulncheck**——将 Go 漏洞数据库与代码对照，使用所有已发布的构建标签，但排除 `ui`（`GOVULNCHECK_TAGS`，来自 `make print-lint-tags-no-ui`：`ui` 嵌入的 SPA 资源不存在于全新检出中，而该标签不会增加任何漏洞可以藏身的 Go 代码）。它读取调用图，因此能区分代码调用的易受攻击函数与仅存在于所需模块中的函数，而 trivy 做不到；它还会评判构建所用工具链的标准库，而 trivy 完全看不到这一点。正是它捕捉到了从 `ssh` 工具触达的 `x/crypto/ssh` 拒绝服务，以及一个使用缺少一年标准库修复的 Go 构建的发布（参见 [The Go toolchain](#go-工具链)）。

## 运行器与版本

如果 `PATH` 上存在扫描器二进制，则使用它；否则运行固定的 docker 镜像（`scripts/security-scan.sh` 顶部的 `aquasec/trivy`、`semgrep/semgrep`、`tag@sha256:` 常量）。govulncheck 没有自己的镜像：固定的 Go 镜像（`GO_IMAGE`）使用 `go run` 在 `GOVULNCHECK_VERSION` 中运行发布版本，并将其模块和构建缓存保存在 `dist/security/.cache/` 下。`SEC_DOCKER=0` 禁止回退，适用于 docker 不可用的机器。

要升级扫描器：拉取新镜像，取其摘要（`docker images --digests`），更新常量，运行 `make security` 并与先前的摘要对比——规则或数据库更新可以在没有提交的情况下改变发现，因此升级应各自作为小型拉取请求。

## 报告

所有内容都落在 `dist/security/`（已 gitignore）：

- `trivy.json`、`semgrep.json`——完整发现，所有严重级别
- `govulncheck.json`——govulncheck 的 JSON 流：每个公告及触达它的追踪（`govulncheck -tags <tags> ./...` 以文本形式打印相同内容）
- `trivy.sarif`、`semgrep.sarif`——供 GitHub 代码扫描使用；trivy 的 SARIF 排除密钥，因为代码扫描会持久化匹配到的文本
- `summary.md`——严重级别计数，即 CI 任务摘要所显示的内容
- `.cache/`——扫描器缓存，使重复运行保持快速

## 门禁

报告与门禁是分开的：扫描始终写入完整报告，然后脚本根据 JSON 统计严重级别并决定退出码。

| 开关 | 值 | 默认 | 含义 |
|---|---|---|---|
| `SEC_FAIL_TRIVY` | `UNKNOWN\|LOW\|MEDIUM\|HIGH\|CRITICAL\|off` | `CRITICAL` | 在达到或高于该级别的漏洞/密钥发现上失败 |
| `SEC_FAIL_SEMGREP` | `ERROR\|WARNING\|INFO\|off` | `off` | 同上，使用 semgrep 自己的严重级别词汇 |
| `SEC_FAIL_GOVULNCHECK` | `symbol\|package\|module\|off` | `symbol` | 在代码调用的易受攻击函数（`symbol`）上失败，也在仅导入的易受攻击包（`package`）上失败，或在它所需的任何易受攻击模块（`module`）上失败 |
| `GOVULNCHECK_TAGS` | 逗号分隔的构建标签 | 除 `ui` 外的所有已发布标签 | govulncheck 编译代码树所使用的标签 |
| `SEC_SCANNERS` | `trivy`、`semgrep`、`govulncheck` 的任意逗号列表 | 全部三个 | 运行哪些扫描器 |
| `SEMGREP_CONFIGS` | 空格分隔的 `--config` 值 | `p/golang p/typescript` | Registry 规则包（浮动别名——发现可能随规则更新而变化） |
| `SEMGREP_APP_TOKEN` | token | 未设置 | 可选；解除 registry 速率限制并启用 Pro 规则。fork 拉取请求永远看不到它——那里的任务按设计保持仅报告 |
| `SEC_DOCKER` | `0\|1` | `1` | 允许 docker 回退 |

未知的开关值、缺失的工具、docker 无法运行或扫描崩溃都属于操作性错误，会使运行失败——包括在 `sec-report` 下，此时只有*发现*被宽恕。

漏洞数据库本身并未固定：本地与 CI 只有在解析到相同 DB 状态时才完全一致，因此在将 CI 失败归咎于环境之前，请先在本地重新运行。

govulncheck 没有严重级别，因此其门禁改为按深度计数。默认仅在代码调用的易受攻击函数上失败；位于无人导入的模块中的公告（例如无修复的已停止维护的 `golang.org/x/crypto/openpgp`）以 `module` 级别报告并保持信息性。

## Go 工具链

`go.mod` 携带两行：`go 1.26.0`，编译模块的最旧 Go，以及 `toolchain go1.27.3`，在默认 `GOTOOLCHAIN=auto` 下构建所链接的最旧发布版本。CI 从 `go` 行安装 Go（`actions/setup-go` 使用 `go-version-file`），然后 `go` 命令切换到 `toolchain` 发布版本（`GOTOOLCHAIN=auto`，官方 Go 的默认值），因此测试任务、发布归档和软件包链接的正是该发布版本的标准库。切换只会向上：Docker 镜像（其构建阶段设置 `GOTOOLCHAIN=auto`，因为官方镜像固定 `local`）和本地构建在更新的情况下使用 `golang:1.27` 镜像或已安装的 Go，这只会增加修复。用 `GOTOOLCHAIN=local` 保留的工具链完全不会切换，并链接其自己的发布版本，最低到 `go` 行。

没有 `toolchain` 行时，每个构建都正好使用 `go` 行的发布版本：1.25 时代的归档使用 `go1.25.0` 构建，缺少其之后发布的每一项标准库安全修复，而使用该工具链运行的 govulncheck 列出了代码调用的其中 32 项（`crypto/tls`、`crypto/x509`、`net/http`、`net/url`、`os.Root`、`archive/tar` 等）。

govulncheck 评判 `go` 所选工具链的标准库，因此当某个 Go 补丁发布修复了代码可触达的漏洞时，每周扫描会失败。修复是一个单行拉取请求：

```bash
go mod edit -toolchain=go1.27.N   # the patch release that carries the fix
make sec-govulncheck              # green again
```

在同一变更中将 `scripts/security-scan.sh` 的 `GO_IMAGE` 升级到相同发布版本，这样 docker 回退不会在每次运行时下载工具链。仅在代码需要更新语言或库时才提升 `go` 行。

## 分类与抑制

- **`.trivyignore.yaml`**——已接受的 CVE/错误配置发现，每个条目带有 `statement` 和 `expired_at` 日期，以便列表会腐烂，而不是永远赦免新问题。
- **`trivy.yaml`**——针对全新检出中不存在的路径（`node_modules`、`dist` 等）使用 `scan.skip-dirs`。Trivy 不遵守 `.gitignore`；没有它，本地扫描会遍历 CI 永远看不到的目录树。支持密钥规则抑制，但使用与已接受 CVE 相同的过期时间和书面理由；尽可能优先使用 `skip-dirs`/`skip-files` 规则或修复。
- **`.semgrepignore`**——保持最小（`.venv`）；semgrep 自行遵守 `.gitignore`。内联抑制为 `// nosemgrep`（Go/TS）或 `# nosemgrep`（YAML/shell），始终带理由注释，最好带上规则 id（`// nosemgrep: <rule-id>`）。semgrep 会在其 SARIF 中保留被抑制的发现并带有 `suppressions` 条目，而代码扫描无论如何都会将其作为警报提出，因此 `scripts/security-scan.sh` 将那些排除在 `semgrep.sarif` 之外，正如 `semgrep.json` 和门禁已经做的那样。
- 测试夹具携带故意伪造的凭据（`sk-test`、`pictures_test.go` 中的 PEM 块、`examples/` 中的存根密钥）。它们是预期内的发现；通过上述忽略文件接受它们，而不是"修复"夹具。

## CI：`.github/workflows/security.yaml`

拉取请求、推送到 `main`、每周一 cron 以及手动触发。该任务调用 `make security`，将两个 SARIF 文件上传到代码扫描（使用不同的 `category` 值），将 `dist/security/` 作为工件附加，并将严重级别摘要打印到任务摘要中，即使在门禁失败时也是如此。

fork 拉取请求会被正常扫描和门禁，但其 SARIF 上传会被跳过——fork token 无法写入 `security-events`，且不受信任的代码绝不能将发现推入此仓库的代码扫描历史。工件仍为维护者携带发现。工作流的任务（`AppSec scanners`）应被标记为分支保护中的必需项，以便门禁阻止合并。

## 逐步收紧

门禁旨在随积压的减少而收紧，每一步都是自己的拉取请求，并带有自己的分类：

1. `SEC_FAIL_TRIVY=CRITICAL` ← 现在
2. `SEC_FAIL_TRIVY=HIGH`
3. `SEC_FAIL_SEMGREP=ERROR`，然后 `WARNING`
4. 一旦其发现被分类，将 `misconfig` 和更广泛的 semgrep 规则包（`p/bash`、`p/yaml`、`p/dockerfile`、`p/react`）纳入门禁

<!-- docsgen:source sha256=fca94ec4dc1c013c -->

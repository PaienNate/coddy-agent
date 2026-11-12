# 从源码构建 Coddy

本页是本地构建的详细参考。简短版本请参阅根目录 **README** 中的 [安装](../../../README.md#install)。

## 先决条件

- **Go** - **1.26** 或更新版本，即 [`go.mod`](../../../go.mod) 的 `go` 系列。紧随其后的 `toolchain` 系列（当前为 **`go1.27.3`**）是构建在默认 `GOTOOLCHAIN=auto` 下链接的最旧版本：CI 和发布归档恰好使用该版本构建，而较旧的本地 `go` 会在首次使用时下载它。更新的本地 Go，或 Dockerfile 起始的 `golang:1.27` 镜像中更新的补丁版本，会用自身构建，这只会增加修复。只有通过 `GOTOOLCHAIN=local` 保留的工具链才会用较旧的发布版构建，最低可到 `go` 系列。当 Go 补丁版本修复了代码可触达的漏洞时，提高 `toolchain` 系列：`go mod edit -toolchain=go1.27.N`（参见 [AppSec 扫描](security-scanning.md#go-工具链)）。
- **Git** - 在可用时，Makefile 会从标签或 `git describe` 嵌入版本字符串。
- **Node.js 和 npm** - 当你同时使用 **`http`** 和 **`ui`** 构建时需要，因为 Makefile 会运行 **`ui-build`**（参见 [`Makefile`](../../../Makefile)）来生成 **`go:embed`** 所获取的资源，并且需要 **`make test`** 和 **`make lint`**，它们运行 SPA 的 vitest 套件及其 TypeScript 检查。

可选：

- **`golangci-lint` v2.x**（使用 Go **1.27** 或更新版本构建）- 用于 **`make lint`**。CI 使用 **`golangci/golangci-lint-action@v9`**（v6 仅支持 golangci-lint v1，v8 及更高版本需要 golangci-lint v2.1.0 或更新版本）。

## 推荐的完整二进制（HTTP、UI、调度器、内存、控制台）

使用 **`memory`** 构建以链接长期记忆（`external/memory`）。在配置中通过 **`memory.enable`** 在运行时启用行为（参见 [`external/memory/README.md`](../../../external/memory/README.md)）。

**HTTP 网关**、**嵌入式 SPA**、**调度器**、**内存**、**消息网关**（**`gateway`** - 参见 [`docs/surfaces/gateway.md`](../surfaces/gateway.md)），以及 **交互式控制台**（**`cli`**，在终端上直接运行 **`coddy`** - 参见 [`docs/surfaces/console.md`](../surfaces/console.md)）由 Go 构建标签控制。对于与默认 **Docker** 镜像匹配并包含所有可选功能的单个二进制：

```bash
make build TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop"
```

输出：**`build/coddy`**。

等效的 **`go build`**（当你使用 **`ui`** 时，在 `ui-build` 之后；或使用 **`make build`**，当 **`TAGS`** 同时包含 **`http`** 和 **`ui`** 时，它会自动运行 **`ui-build`**）：

```bash
make ui-build   # only when using -tags=...,ui,... with http; Makefile runs this for you on `make build`
VERSION="$(make -s print-version)"
REVISION="$(make -s print-revision)"
go build -tags=http,ui,scheduler,memory,cli,gateway,swarm,metrics,browser \
  -ldflags "-X github.com/EvilFreelancer/coddy-agent/internal/version.Version=${VERSION} -X github.com/EvilFreelancer/coddy-agent/internal/version.Revision=${REVISION}" \
  -o build/coddy \
  ./cmd/coddy/
```

[**Dockerfile**](../../../Dockerfile) 采用相同思路：通过 **`BUILD_TAGS`** 使用逗号分隔的标签（默认 **`http,scheduler,ui,memory,gateway,cli,swarm,metrics`**），并在版本 **`X`** 标志之外，用 **`-ldflags "-s -w ..."`** 去除调试符号。

## 安装到你的 PATH

**`make install`** 会把 **`build/coddy`** 复制到你的 **`PATH`** 上：

- 如果 **`build/coddy`** 已存在（例如在 **`make build TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop"`** 之后），则按原样安装，不会重新构建。
- 如果二进制缺失，**`make install`** 会先运行 **`make build TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop"`**。

- **root** - **`/usr/local/bin/coddy`**，手册页位于 **`/usr/local/share/man/man1`**
- **非 root** - **`~/.local/bin/coddy`**（确保该目录在 **`PATH`** 上），手册页位于 **`~/.local/share/man/man1`**

```bash
make build TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop"
make install
```

## 分发包

**`make deb`** 和 **`make rpm`** 构建发布版发布的 Linux 软件包；**`make brew`** 为其中一个渲染 Homebrew cask。

```bash
make deb
make rpm
```

输出：**`dist/coddy_<version>_linux_<arch>.deb`** 和 **`.rpm`**。可调项：

| 变量 | 默认值 | 作用 |
|----------|---------|------|
| **`PKG_ARCHS`** | 主机 **`GOARCH`** | 要打包的架构，例如 **`"amd64 arm64"`** |
| **`PKG_TAGS`** | **`http ui scheduler memory cli gateway swarm metrics browser desktop`** | 打包二进制的构建标签 |
| **`DIST_DIR`** | **`dist`** | 软件包的输出位置 |

```bash
make deb PKG_ARCHS="amd64 arm64"
make rpm PKG_TAGS="http cli"      # lean binary, no npm step
```

该配方是 **`packaging/nfpm.yaml`**，由 **`scripts/build-packages.sh`** 驱动，它会将手册页（**`packaging/man/coddy.1`**）、shell 补全（**`packaging/completions/`**）、systemd 用户单元（**`packaging/systemd/coddy.service`**）、维护者脚本（**`packaging/scripts/postinstall.sh`**、**`preremove.sh`**）、**`config.example.yaml`** 和 **`LICENSE`** 暂存到一个目录，并在其上运行 [nfpm](https://nfpm.goreleaser.com/)。nfpm 不是模块依赖：脚本在 **`PATH`** 上有 **`nfpm`** 时使用它，否则用 **`go run`** 获取固定版本，因此无需预先安装任何东西。

用户单元会被安装但不会启用，安装后消息会说明这一点；每个想要该服务的用户运行 **`coddy serve install`**。不会创建系统服务、系统账户或 **`/etc`** 下的文件，因为每个用户都将状态保存在 **`~/.coddy`** 下。该单元不是手写的：**`serve.PackagedUnitFile()`**（**`internal/serve/systemd.go`**）会渲染它，这与为脚本安装写入单元的是同一个函数；在 **`packaging/systemd/coddy.service`** 与其逐字节匹配之前，**`TestPackagedUnitIsTheRenderedOne`** 会失败。然后 **分发包** CI 任务会检查两种包格式都携带该文件和维护者脚本。

这些软件包建议安装 **`tmux`**，但不要求任何东西。Coddy 在没有它时也能运行，而它的控制台在其中有良好运行，会话可以比关闭的终端或断开的 SSH 连接存活更久。建议项会被指明但绝不会被安装 - **`apt`** 和 **`zypper`** 在安装期间列出它，**`dnf`** 将其保留在元数据中（**`rpm -q --suggests`**） - 因此选择权仍归用户。**分发包** CI 任务会检查 deb 和 rpm 都建议它，且既不要求也不推荐它。

版本字符串由 **`scripts/package-version.sh`** 针对两种格式进行规范化 - rpm 禁止版本中出现 **`-`**，而 dpkg 会将最后一个读作 Debian 修订版本的开头，因此 **`1.0.8-5-gb6b7d31-dirty`** 会打包为 **`1.0.8+5.gb6b7d31.dirty`**，两者都接受它，并且都会将其排序在 **`1.0.8`** 之后。

### Homebrew cask

```bash
make brew VERSION=1.0.11
```

**`scripts/build-homebrew-cask.sh`** 会用两个 macOS 归档的版本和 SHA-256 填充 **`packaging/homebrew/coddy.rb.tmpl`**，并写入 **`dist/coddy.rb`**。当归档位于 **`DIST_DIR`** 中时（发布任务中正是这种情况，就在交叉编译之后），它会对这些归档计算哈希；否则从该 GitHub 发布的 **`SHA256SUMS`** 读取其校验和 - cask 会固定校验和，因此只能为归档存在的版本渲染。

当前 Homebrew 拒绝通过路径给出的 cask，因此会在一次性的本地 tap 中试用渲染出的文件。在装有 Homebrew 的 Mac 上：

```bash
make brew-cask-check                                    # dist/coddy.rb from a local tap
make brew-cask-check BREW_TAP=coddy-project/coddy-agent # the published cask, by name
```

**`scripts/check-homebrew-cask.sh`** 会安装 cask，对其运行 **`brew style`**，检查 **`coddy -v`** 打印其版本，检查二进制解析到 **`Caskroom`** 中且不留下 **`com.apple.quarantine`** 属性，检查手册页和两个补全都已链接，并检查 **`coddy update`** 命名为 **`brew upgrade --cask coddy`**，然后运行它。最后它会卸载 cask 并移除其添加的 tap；**`--keep`** 会保留两者。它绝不会开启 Homebrew 的开发者模式：它会将本地 tap 创建为普通目录，而不是运行 **`brew tap-new`**，并且只为那一条命令向 **`brew style`** 提供 **`HOMEBREW_DEVELOPER`** 变量，因为不带该变量运行开发者命令会永久开启该模式。

cask 会链接 **`coddy`**、**`coddy.1`** 和两个补全脚本，这就是发布版 **`darwin`** 和 **`linux`** 归档在二进制旁边携带这些文件的原因。它声明 **`depends_on macos: :monterey`**：Go 1.25 及更高版本构建的二进制需要 macOS 12。**`postflight_steps`** 步骤会从二进制中移除 Homebrew 在下载时设置的隔离属性；该二进制为临时签名且未公证，因此 Gatekeeper 不会阻止其首次运行。它在 **`caveats`** 中推荐 **`tmux`**，而不是声明它：cask 的 **`depends_on`** 没有可选形式，而 Coddy 在没有 tmux 时也能运行，因此 cask 只命名它而不安装任何东西。

### Homebrew tap

用户通过 tap [coddy-project/homebrew-coddy-agent](https://github.com/coddy-project/homebrew-coddy-agent) 获取 cask，使用 **`brew install coddy-project/coddy-agent/coddy`** 安装。其 **`Casks/coddy.rb`** 是生成的，绝不手工编辑。

```bash
make brew-publish VERSION=1.0.11
```

**`scripts/publish-homebrew-tap.sh`** 会克隆 tap，用渲染出的 cask 替换 **`Casks/coddy.rb`**，并推送名为 **`coddy X.Y.Z`** 的提交。tap 只会向前推进：比 tap 当前携带版本更旧的发布版 cask 会被跳过，而与 tap 中完全相同的 cask 不会产生任何变化，因此为旧标签重新运行发布任务不会将其回退。若推送在与另一个发布任务的竞争中失败，则会从 tap 的新头部重试。

发布任务会在资产上传后运行它，使用部署密钥的私钥部分，该密钥只能写入 tap，不能做其他事，来源于机密 **`HOMEBREW_TAP_DEPLOY_KEY`**；GitHub 的 SSH 主机密钥来自 **`api.github.com/meta`**。没有该机密时，发布步骤会在上传后报错停止，tap 会停留在上一个发布版，直到再次运行该任务；在没有它的情况下手动运行，脚本会使用你自己的 git 凭据推送。随后，**Homebrew tap** 工作流（[`homebrew.yaml`](../../../.github/workflows/homebrew.yaml)）会在装有当前 Homebrew 的 macOS runner 上按名称安装 cask，并检查它携带刚发布的版本或更新版本。同一工作流会在触及 cask、脚本或自身的拉取请求上运行，其中它还会安装检出内容所渲染的 cask。它也会手动运行，并每周自行运行，以捕获停止安装该 tap 的 cask 的 Homebrew 发布。

组织所有者只需设置一次密钥。首先在组织设置中的 "Member privileges"、"Deploy keys" 下，允许组织的仓库使用部署密钥。然后生成一对密钥，并为其公钥部分授予对该 tap 的写权限。

```bash
ssh-keygen -t ed25519 -N "" -C "coddy-agent release workflow" -f ./tap_key
gh repo deploy-key add tap_key.pub --repo coddy-project/homebrew-coddy-agent --allow-write --title "coddy-agent release workflow"
```

将私钥部分存储为本仓库的机密，并删除两个文件。

```bash
gh secret set HOMEBREW_TAP_DEPLOY_KEY --repo coddy-project/coddy-agent < tap_key
rm tap_key tap_key.pub
```

无需其他操作：**`tag-on-merge.yaml`** 会使用 **`secrets: inherit`** 调用发布工作流，而 Actions 会自行提供 **`GITHUB_TOKEN`**。新密钥以相同方式替换旧密钥；之后从 tap 中移除旧的部署密钥。

### Homebrew formula

```bash
make brew-formula VERSION=1.0.13
make brew-check VERSION=1.0.13
```

**`scripts/build-homebrew-formula.sh`** 会用该标签 **源码** 归档的版本和 SHA-256 填充 **`packaging/homebrew/coddy-formula.rb.tmpl`**，并写入 **`dist/formula/coddy.rb`**。它会下载归档以计算哈希，因此版本必须是已发布的标签。

该文件是 [homebrew/core](https://github.com/Homebrew/homebrew-core) 拉取请求所携带的产物。Homebrew 将开源命令行软件作为从源码构建的 formula 路由到那里，并将 homebrew/cask 保留给原生应用和仅二进制软件，因此上面的 cask 是我们自己的渠道，而 formula 是提交物。该 formula 会构建发布标签集，这就是 **`node`** 与 **`go`** 一起成为构建依赖的原因：嵌入式 SPA 是生成的，而不是提交的。

**`scripts/check-homebrew-submission.sh`**（**`make brew-check`**）是预检 - 令牌可用性、知名度阈值、发布版以及渲染出的 formula。当有内容阻止提交时，它会以非零状态退出。完整路径，包括目前阻止自我提交的知名度计算，见 [homebrew.md](../getting-started/homebrew.md)。

这些软件包安装什么，以及它们如何与 **`coddy update`** 交互，已记录在 [install.md](../getting-started/install.md#linux-软件包debrpm) 和 [update.md](../getting-started/update.md#由包管理器拥有的安装) 中。

## 从 GitHub Releases 更新

有关 **`coddy update`**、发布资产名称以及它与 **`make install`** 的区别，请参阅 **[docs/getting-started/update.md](../getting-started/update.md)**。

## 精简构建（聚焦 ACP，更小的二进制）

普通的 **`make build`**（空的 **`TAGS`**）会省略 **`external/httpserver`**、嵌入式 UI、**`external/scheduler`** 和 **`external/memory`**。你仍会获得 **`coddy acp`**、核心工具和 MCP。

```bash
make build
```

当你只需要 stdio ACP，并希望更少依赖且没有 **`npm`** 步骤时使用。

## 桌面窗口（`coddy desktop`）

该窗口是普通二进制的一个动词，而 `desktop` 标签位于 `FULL_TAGS` 中，因此每个发布二进制都有它。`make desktop` 是设置了该标签的 `make build`：

```bash
make desktop            # build/coddy with the shipped tags
./build/coddy desktop
```

它会在智能体主目录记住的环回端口上提供 Web 界面，并以应用程序模式在其中打开 Chromium 系浏览器，使用位于 `<home>/desktop/profiles/` 下自己的浏览器配置文件。该界面在一个已经携带 HTTP 服务器和 SPA 的二进制中仅增加 0.44 MiB，因为它驱动窗口所用的 X11 库（`jezek/xgb`）已经因控制台的剪贴板图片而存在；只有托盘图标的 DBus 客户端是新增的。

容器镜像会构建相同集合（`BUILD_TAGS` 在 `Dockerfile` 中以及 `.github/workflows/docker-build-push.yaml` 中），因此 `coddy` 无论来自哪里都是同一个程序，即使容器中没有任何显示设备可用于打开窗口。该命令本身需要 `http` 和 `ui` 与 `desktop` 同时存在 - 它会在 SPA 上打开 - 而缺少三者中任何一个的构建会用列出这些标签的那一行回答 `coddy desktop`。它是什么以及如何表现：[桌面窗口](../surfaces/desktop.md)。

shell 的浏览器检查（`external/ui` 中的 `npm run check:desktop`）会驱动真实二进制：标签页、快捷键组合、从 Coddy 指向外部的链接、菜单，以及重启后仍保留的标签页列表。

桌面应用程序是一个独立的 Linux 软件包，并且不携带二进制：它提供将 Coddy 放入应用程序启动器的菜单项和图标，并依赖 `coddy` 提供程序。

```bash
make deb-desktop        # dist/coddy-desktop_<version>_linux_<arch>.deb
make rpm-desktop        # the same as an .rpm
```

它们接受相同的 `PKG_ARCHS` 和 `DIST_DIR` 可调项，并且不构建任何东西，因为该软件包中没有二进制。配方是 `packaging/nfpm-desktop.yaml`，由 `scripts/build-desktop-packages.sh` 驱动，它会暂存菜单项（`packaging/desktop/coddy-desktop.desktop`）、图标（`packaging/desktop/coddy-desktop.svg` 以及 `packaging/desktop/icons/` 下的各个尺寸，使用 `rsvg-convert` 从其渲染并提交：Web 界面的 favicon 在应用程序菜单提供的 32 和 48 像素下，标记周围留白太多而难以辨认）和 `LICENSE`。该软件包不启用任何东西，没有维护者脚本，并依赖 `coddy`。

## Android（Termux）

```bash
make android TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop desktop"   # build/coddy-android-arm64, -amd64
make check-android                                              # go vet with GOOS=android, test files included
```

Android 构建使用相同源码，但带有 **`GOOS=android`** 和 cgo，由 Android NDK 针对 arm64 和 x86_64 链接 Bionic：一个位置无关的可执行文件，它将 **`/system/bin/linker64`** 命名为其解释器，并且只需要 **`libc`**、**`libdl`** 和 **`liblog`**。这是 Termux 能启动的一种形态，因为 Termux 会通过 Android 的链接器运行每个程序，而该链接器会用 **`has unexpected e_type: 2`** 拒绝静态的 **`GOOS=linux`** 二进制；Bionic 为二进制提供 libc 为任何程序提供的内容：链接器下的自身参数以及 Android 的解析器。**`GOOS=android`** 还会让 Go 避开 Android 的 seccomp 过滤器所禁止的系统调用。无论如何，Go 仅通过 C 工具链链接 android/amd64，而没有 cgo 的构建会为每个主机请求 **`127.0.0.1:53`**，因此 **`internal/platform/android_nocgo.go`** 会在编译时阻止它。用户看到的内容见 [Android（Termux）](../getting-started/android.md)。

**`scripts/android-cc.sh arm64|amd64`** 会打印两个目标都使用的 NDK clang，面向 API 24（Android 7.0，Termux 运行的最低版本）：它从 **`ANDROID_NDK_HOME`**、**`ANDROID_NDK_ROOT`** 或 **`ANDROID_NDK`** 获取 NDK，这些由 GitHub runner 设置，否则使用 SDK 下最新的一个（**`sdkmanager "ndk;27.3.13750724"`** 安装 runner 携带的版本）。

在 Termux 之外构建的二进制不携带 Termux 应用于其自身 Go 的任何补丁，因此 **`internal/platform`** 会在运行时进行其余调整。当链接器启动二进制时，**`android_init.go`** 会保留二进制路径，因为此时 **`/proc/self/exe`** 命名链接器，并将 **`SSL_CERT_FILE`** 和 **`TMPDIR`** 指向 Termux 前缀；**`platform.AdaptCommand`** 会以 **`termux-exec`** 的方式启动子进程，当 Coddy 本身以该方式启动时通过链接器，并在从该前缀取得的 shebang 中使用 **`/usr/bin/env`** 和 **`/bin/sh`**。每个构建 **`exec.Cmd`** 的地方都会在 **`Start`** 之前调用 **`AdaptCommand`**，而 **`TestEverySpawnSiteAdaptsTheCommand`** 会在没有这样做的位置失败。

这些决策是 **`internal/platform/android.go`** 中的普通函数，由 **`features/android_termux.feature`** 和 **`internal/platform/android_test.go`** 在任何 Unix 主机上保持。**`//go:build android`** 背后的文件仅由 **`make check-android`** 以及拉取请求检查中的 **Android 交叉构建** 任务编译，该任务还会构建两个发布二进制，检查每个都是位置无关的、命名 Android 链接器且不需要 Android 未提供的库，并将它们作为 **`coddy-android-arm64`** 和 **`coddy-android-amd64`** 产物上传，以便在发布前在设备上试用。

## 版本字符串（`LDFLAGS`，`print-version`）

Makefile 设置：

```text
LDFLAGS := -X github.com/EvilFreelancer/coddy-agent/internal/version.Version=$(VERSION) -X github.com/EvilFreelancer/coddy-agent/internal/version.Revision=$(REVISION)
```

**`VERSION`** 从 git 解析（**HEAD** 处的标签，否则 **`git describe`**，再否则 **`dev`**）。打印下一次 **`make build`** 将嵌入的相同值：

```bash
make -s print-version
```

**`REVISION`** 是 **HEAD** 指向的提交（**`git rev-parse --verify -q HEAD`**），在 git 之外以及首次提交之前为空，**`make -s print-revision`** 会打印它。构建会传递该提交本身，因为 Go 1.26 及更早版本会从最近的 **`.git`** *目录* 填充它们自己的 **`vcs.revision`**，而在 git worktree 中 **`.git`** 是一个文件。Go 1.27 由 `toolchain` 系列选择，会填充 worktree 自己的提交；对于通过 **`GOTOOLCHAIN=local`** 保留的旧工具链，构建仍会命名它。Coddy 与其他一些智能体工具一样，将其 worktree 放在检出目录内的文件夹中，因此在此类 worktree 中构建的二进制会在自身版本旁报告外层检出的提交，位于 **`coddy_build_info`** 系列以及 **Settings → System** 上。未带该标志构建的二进制（普通的 **`go build`**）会继续读取 **`vcs.revision`**。**`scripts/build-packages.sh`** 会传递两个值；发布工作流、[**Dockerfile**](../../../Dockerfile) 和 Homebrew formula 仅传递版本，因为它们从干净克隆、没有 **`.git`** 的镜像上下文或源码 tarball 构建，在这些场景中工具链的值是正确的或不存在。

与 **`make build`** 对齐的手动单行命令：

```bash
go build \
  -tags=http,ui,scheduler,memory,cli,gateway,swarm,metrics,browser \
  -ldflags "-X github.com/EvilFreelancer/coddy-agent/internal/version.Version=$(make -s print-version) -X github.com/EvilFreelancer/coddy-agent/internal/version.Revision=$(make -s print-revision)" \
  -o build/coddy \
  ./cmd/coddy/
```

## **`TAGS` 与 `go build -tags`**

在 **`Makefile`** 中，**`TAGS`** 是 **空格分隔** 的：

```bash
make build TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop"
```

**`go build`** 期望 **逗号分隔** 的列表（无空格）：

```bash
go build -tags=http,ui,scheduler,memory,cli,gateway,swarm,metrics,browser ...
```

这些标签的顺序无关紧要。

## 构建标签参考

| 标签 | 启用内容 | 文档 |
|-----|---------|----------------|
| **`memory`** | 长期记忆子智能体；与 **`http`** 一起时启用 **`/coddy/sessions/{id}/memory/*`** REST；通过 **`memory.enable`** 切换运行时行为 | [`external/memory/README.md`](../../../external/memory/README.md) |
| **`http`** | OpenAI 形态的 REST 网关，`coddy serve` 在 **`httpserver.enable`**、**`/docs`**、**`/openapi.yaml`** 下运行 | [`docs/reference/http-api.md`](../reference/http-api.md) · [`external/httpserver/`](../../../external/httpserver) |
| **`ui`** | 在 **`/`** 上嵌入 SPA（需要 **`http`**；仅使用 **`http`** 时 **`/`** 返回 **404**） | [`docs/surfaces/web-ui.md`](../surfaces/web-ui.md) · [`DESIGN.md`](../../../DESIGN.md) |
| **`scheduler`** | 调度器守护进程钩子、**`coddy_scheduler_*`** 工具；与 **`http`** 一起时启用 **`/coddy/scheduler`** REST | [`docs/operate/scheduler.md`](../operate/scheduler.md) · [`external/scheduler/README.md`](../../../external/scheduler/README.md) |
| **`cli`** | 交互式控制台 TUI：在终端上直接运行 **`coddy`** | [`docs/surfaces/console.md`](../surfaces/console.md) · [`external/cli/`](../../../external/cli) |
| **`gateway.telegram`** | Telegram 机器人适配器，由 **`coddy serve`** 在 **`gateways.telegram.enable`** 下启动；按用户/群组会话、访问控制 | [`docs/surfaces/gateway.md`](../surfaces/gateway.md) · [`external/gateway/`](../../../external/gateway) |
| **`gateway`** | 所有消息适配器（**`gateway.telegram`** 的超集；包括未来的 Discord、Slack 适配器） | [`docs/surfaces/gateway.md`](../surfaces/gateway.md) |
| **`swarm`** | 聚合节点的无状态中继，由 **`coddy serve`** 在 **`swarm.enable`** 下启动 | [`docs/operate/swarm.md`](../operate/swarm.md) · [`external/swarm/`](../../../external/swarm) |
| **`metrics`** | Prometheus 导出器，由 **`coddy serve`** 在 **`metrics.enable`** 下启动（默认关闭）；在自己的端口上提供 **`/metrics`** | [`docs/operate/monitoring.md`](../operate/monitoring.md) · [`external/metrics/`](../../../external/metrics) |
| **`desktop`** | 桌面窗口：**`coddy desktop`** 命令、HTTP 服务器为打开窗口的进程挂载在 `/desktop/` 的标签页 shell，以及打开它的浏览器后端。需要 **`http`** 和 **`ui`** 与它同时存在；位于 **`FULL_TAGS`** 中，且不在容器镜像中 | [`docs/surfaces/desktop.md`](../surfaces/desktop.md) · [`external/desktop/`](../../../external/desktop) |

**`make test`** 是快速运行：整棵树运行一次，并编译进所有可选模块（**`http,ui,scheduler,memory,cli,gateway,swarm,metrics,browser,desktop`**）。**`make test-matrix`** 会遍历每种组合（[`Makefile`](../../../Makefile) 中的 **`TEST_TAG_SETS`** 列表）；CI 会在每个拉取请求上运行该矩阵，每种组合一个任务。`cli` 任务还会启动构建出的二进制，并通过真实 pty 驱动控制台（`examples/cli/cli_e2e_startup.py`），而 **Telegram stand (tgfake)** 任务会针对 [tgfake](tgfake.md) 的发布二进制运行 Telegram 机器人（`examples/gateway/tg_e2e_offline.sh`；它运行的二进制使用 `gateway metrics` 构建，脚本会在聊天后抓取导出器）。**Docker image** 任务会为 **`linux/amd64`** 和 **`linux/arm64`** 构建运行时镜像，这是发布版推送的平台，并检查每个变体都携带自身平台的二进制（**`make check-image`**，参见 [检查镜像的平台](../getting-started/docker.md#检查镜像的平台)）。**`make test-race`** 会在 Go 竞态检测器下运行整棵树，使用除 `ui` 之外的所有标签；CI 会在每个拉取请求的 **Race detector** 任务中运行它，`GOFLAGS=-count=3 make test-race` 会在本地重复每个测试，以查找偶发竞态。可以单独运行两个组：**`make test-cache`** 是提示词缓存组（`features/prompt_cache_*.feature` 规范和 `TestPromptCache*` 测试，属于 `make test`：每个请求发送给提供商的内容，在一个会话中逐字节保持一致），而 **`make test-perf`** 是性能组（`perf_test.go` 文件的基准测试，从不属于 `make test`：一轮重复工作的耗时，以及会话基于本仓库自身规则所产生的上下文成本，分别为 `system-tokens`、`request-tokens` 和 `duplicate-rules`）。

在 Windows 上，通过 Git Bash 运行 Make。构建和安装目标使用 Go 的可执行文件后缀（`coddy.exe`），包括 `make docs` 运行的二进制。仓库通过 `.gitattributes` 将文本文件保持为 LF，包括 CSS 源契约输入和嵌入式 Markdown。使用 `GOFLAGS=-p=4 make test` 为套件的子 MCP 和 PowerShell 进程留出启动时间。Windows CI 任务会使用随附标签运行规则适配器、SPA 和完整的 Go 套件。测试夹具在 JSON 和 YAML 中转义原生路径，打包检查接受 CRLF 检出。POSIX 模式断言在 Windows 上不适用；通过 Unix 目录或文件权限诱发失败的测试会在那里跳过（以及以 root 运行时）。仅当 Windows 拒绝符号链接权限时，符号链接检查才会跳过；启用开发者模式或运行提升权限的终端来执行它。Docker 资产检查还通过从索引读取链接模式，覆盖带 `core.symlinks=false` 的 Git 检出。删除仍被 `os.Root` 持有的目录的预览测试仅适用于 POSIX；Windows 需要先停止该预览，再移除其根目录。

文档检查接受 CRLF 代码围栏，并使用 LF 行尾测量 SVG 文本，因此 Windows 检出转换不会改变生成的清单。

## 发布二进制（CI）

对于 **`main`** 上的每个 SemVer git 标签 **`X.Y.Z`**，[**发布二进制**](../../../.github/workflows/release-binaries.yaml) 工作流（与 Docker CI 分开）会将归档上传到匹配的 **GitHub Release**：

| 归档 | 平台 |
|---------|----------|
| **`coddy_X.Y.Z_linux_amd64.tar.gz`** | Linux x86_64 |
| **`coddy_X.Y.Z_linux_arm64.tar.gz`** | Linux arm64 |
| **`coddy_X.Y.Z_android_arm64.tar.gz`**、**`coddy_X.Y.Z_android_amd64.tar.gz`** | Termux 下的 Android arm64 和 x86_64（参见 [上文](#androidtermux)） |
| **`coddy_X.Y.Z_windows_amd64.zip`** | Windows x86_64（**`coddy.exe`**） |
| **`coddy_X.Y.Z_windows_arm64.zip`** | Windows arm64（**`coddy.exe`**） |
| **`coddy_X.Y.Z_windows_arm64.zip`** | Windows arm64（**`coddy.exe`**） |
| **`coddy_X.Y.Z_darwin_amd64.tar.gz`** | macOS Intel |
| **`coddy_X.Y.Z_darwin_arm64.tar.gz`** | macOS Apple Silicon |
| **`coddy_X.Y.Z_linux_amd64.deb`**、**`coddy_X.Y.Z_linux_arm64.deb`** | Debian、Ubuntu 及其衍生版 |
| **`coddy_X.Y.Z_linux_amd64.rpm`**、**`coddy_X.Y.Z_linux_arm64.rpm`** | Fedora、RHEL、openSUSE 及其衍生版 |
| **`coddy.rb`** | 此标签 macOS 归档的 Homebrew cask；同一文件会提交到 tap [coddy-project/homebrew-coddy-agent](https://github.com/coddy-project/homebrew-coddy-agent) |
| **`SHA256SUMS`** | 上述每个归档和软件包的校验和 |
| **`SHA256SUMS.minisig`** | 该列表的签名（[发布签名](#发布签名)） |

**`.tar.gz`** 归档在二进制旁边携带手册页和 shell 补全；软件包会包装同一任务刚构建的 Linux 二进制，而不是自行编译，因此同一标签的 **`.deb`**、**`.rpm`** 和 **`.tar.gz`** 包含逐字节相同的可执行文件。

标签匹配完整功能集：**`http`**、**`ui`**、**`scheduler`**、**`memory`**、**`cli`**、**`gateway`**、**`swarm`**、**`metrics`**。标签存在后手动运行：

```bash
gh workflow run "Release binaries" --ref X.Y.Z -f tag=X.Y.Z
```

### 发布签名

该工作流通过 **`cmd/releasesign`** 使用仓库机密 **`RELEASE_SIGNING_KEY`** 中的密钥对 **`SHA256SUMS`** 签名，而 **`coddy update`** 会对照 **`releaseSigningKeys`** 中的公钥检查签名（**`internal/update/checksum.go`**）。签名采用 minisign 格式，因此 **`minisign -Vm SHA256SUMS -P <public key>`** 会验证它，并且其受信任注释会命名标签，因此旧发布版的已签名列表无法冒充新发布版。从 **`firstSignedRelease`** 起的发布版在没有有效签名时会被拒绝。

如果机密缺失或其密钥不被更新器信任，官方发布任务会在上传资产前停止。签名命令会在写入 **`SHA256SUMS.minisig`** 之前，对照随附公钥验证签名。空的公钥列表不会关闭强制验证。没有该机密的复刻可以发布未签名版本；官方更新器不会验证另一个仓库的发布版。

要生成密钥，请运行第一条命令，将其打印的公钥放入 **`releaseSigningKeys`**，然后存储机密：

```bash
go run ./cmd/releasesign keygen coddy-release-signing.key
gh secret set RELEASE_SIGNING_KEY --repo coddy-project/coddy-agent < coddy-release-signing.key
```

GitHub 不会归还机密，因此请将该文件离线保存，或接受丢失的密钥只能被替换而无法恢复。要替换密钥，请在由旧密钥签名的发布版中，将其公钥放在旧公钥之前发布，并在更新者拥有该构建后切换机密：二进制只信任其携带的密钥。

签名可以阻止任何能够上传发布资产的人，这正是泄露的令牌或另一个任务中被攻破的步骤所能做的事。它不能阻止任何能够使用该机密运行本仓库工作流的人。

## AppSec 扫描

**`make security`** 会在检出内容上运行项目的安全门禁 - trivy（依赖漏洞、机密；错误配置仅报告）、semgrep（SAST）和 govulncheck（代码可触达的 Go 漏洞，包括固定工具链的标准库）- 使用与拉取请求上 [**安全扫描**](../../../.github/workflows/security.yaml) 工作流相同的版本和阈值。**`make sec-report`** 会扫描但不会因发现而失败。报告会落在 **`dist/security/`** 中。运行器、门禁可调项和分诊：[AppSec 扫描](security-scanning.md)。

## 来自上游的 **`go install`**

```bash
go install github.com/EvilFreelancer/coddy-agent/cmd/coddy@latest
```

这会编译模块默认内容，**不包含** 你的本地 **`TAGS`**。对于已知功能集（HTTP、UI、调度器、内存、控制台、消息网关），请克隆仓库并使用 **`make build TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop"`**（或如上所述使用 **`go build -tags=...`**）。

<!-- docsgen:source sha256=a0fb1fcd772b8293 -->

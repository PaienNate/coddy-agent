# 安装 Coddy

安装脚本和落地页：**https://coddy.dev/**

## 一行安装

**Linux / macOS**

```bash
curl -fsSL https://coddy.dev/install.sh | bash
```

**Windows (PowerShell)**

```powershell
irm https://coddy.dev/install.ps1 | iex
```

**Android (Termux)**：同一个 **`install.sh`** 会获取 Android 构建；参见
[Android (Termux)](#android-termux)。

当 **`~/.coddy/config.yaml`** 缺失时，从发布 **`config.example.yaml`** 中创建它。

在 Linux 和 macOS 上，该脚本安装的不只是二进制文件。发布归档在 **`coddy`** 旁边还带有 man
手册页以及 bash 和 zsh 补全，因此它们会被放入同一前缀的 **`share`** 目录
（**`~/.local/bin`** -> **`~/.local/share`**），而用户级安装随后会向你的登录 shell 的 rc 文件
写入一个带防护标记的块：

| 该块的作用 | 原因 |
|---|---|
| 将安装目录加入 **`PATH`** | 新开的 shell 从未听说过 **`~/.local/bin`** |
| 将数据目录加入 **`MANPATH`** | 这样 **`man coddy`** 才能找到手册页 |
| 将补全目录前置到 zsh 的 **`fpath`** 并注册 **`_coddy`** | zsh 只搜索系统目录 |
| source bash 补全文件 | **`bash-completion`** 可能未安装 |

**`coddy update`** 会连同二进制一起刷新这些文件，因此 **`man coddy`** 和 Tab 补全
永远不会落后于正在运行的发布版本（参见 [update.md](update.md#手册页及其旁边的补全脚本)）。

该块每次运行时都会在其标记之间被重写，而不是追加，因此重新运行安装程序绝不会
重复它。用 **`--no-shell-setup`** 可以跳过它。系统前缀
（**`--install-dir /usr/local/bin`**）完全不会写入该块：那些目录本就已在
**`PATH`**、man 搜索路径以及 zsh 默认的 **`fpath`** 中。

```bash
source ~/.zshrc   # or open a new terminal
coddy -v
```

该脚本不安装 systemd 单元。在 Linux 上，若要让 **`coddy serve`** 作为
你账户的服务持续运行，请在配置中已有提供商密钥后运行一次 **`coddy serve install`**：它会为
脚本所安装的二进制写入 **`~/.config/systemd/user/coddy.service`**，启用它，并在
**`~/Coddy`** 中启动它。脚本结束时也会说明这一点。参见
[服务指南](../operate/serve.md#在-linux-上作为-systemd-用户服务)。

## Linux 软件包（deb、rpm）

每次发布都会在归档文件旁边为 **x86_64** 和 **arm64** 发布一个 **`.deb`** 和一个 **`.rpm`**，
因此 Coddy 可以像系统其余部分那样安装，也可以同样方式移除。
在你管理的机器上，优先选择这种方式而不是安装脚本：文件由
包数据库跟踪，man 手册页和 shell 补全已为你配置好，并且 `coddy update`
知道不要与你的包管理器冲突。

**Debian、Ubuntu 及衍生版**

```bash
curl -fsSLO https://github.com/coddy-project/coddy-agent/releases/latest/download/coddy_1.0.10_linux_amd64.deb
sudo apt-get install ./coddy_1.0.10_linux_amd64.deb
```

**Fedora、RHEL、openSUSE 及衍生版**

```bash
curl -fsSLO https://github.com/coddy-project/coddy-agent/releases/latest/download/coddy_1.0.10_linux_amd64.rpm
sudo dnf install ./coddy_1.0.10_linux_amd64.rpm
```

把 **`1.0.10`** 替换为你想要的发布版本，在 64 位 ARM 上把 **`amd64`** 替换为 **`arm64`**。
[发布页面](https://github.com/coddy-project/coddy-agent/releases) 列出了每个标签
发布了什么，旁边的 **`SHA256SUMS`** 也涵盖这些软件包：

```bash
sha256sum -c --ignore-missing SHA256SUMS
```

这些软件包建议安装 **`tmux`**，但并不会安装它。Coddy 无需它即可运行，而它的控制台
在其中运行良好：那里的会话可以在终端关闭或 SSH 连接断开后继续存在。**`apt`**
会将它列在 *Suggested packages* 下；用 **`sudo apt-get install tmux`** 或
**`sudo dnf install tmux`** 添加它。

### 软件包安装了什么

| 路径 | 内容 |
|------|------|
| **`/usr/bin/coddy`** | 完整二进制（**`http`**、**`ui`**、**`scheduler`**、**`memory`**、**`cli`**、**`gateway`**、**`swarm`**、**`metrics`**） |
| **`/usr/share/man/man1/coddy.1.gz`** | **`man coddy`** |
| **`/usr/share/bash-completion/completions/coddy`** | bash 补全 |
| **`/usr/share/zsh/site-functions/_coddy`** | zsh 补全 |
| **`/usr/lib/systemd/user/coddy.service`** | 用于 **`coddy serve`** 的 systemd 用户单元，已安装但**未启用** |
| **`/usr/share/doc/coddy/config.example.yaml`** | **`~/.coddy/config.yaml`** 的起点 |
| **`/usr/share/doc/coddy/LICENSE`**、**`copyright`** | 许可证 |

该软件包不会为任何人启用该单元，也没有系统服务、没有系统账户，
**`/etc`** 下也没有任何内容。配置、会话、技能和凭据都保留在调用用户的
**`~/.coddy`** 中，因此一个已安装的软件包就能服务机器上的每个用户，各自拥有
自己的状态，而运行什么 - 控制台、HTTP 网关、基于 ACP 的编辑器、服务 - 仍由
每个用户自己决定。安装时打印的消息会说明该单元已存在但未启用；
**`coddy serve install`**，以该服务所属用户的身份运行，会启用并启动它（参见
[服务指南](../operate/serve.md#在-linux-上作为-systemd-用户服务)）。

### 首次运行

```bash
mkdir -p ~/.coddy
cp /usr/share/doc/coddy/config.example.yaml ~/.coddy/config.yaml
# set a provider key in ~/.coddy/config.yaml
coddy
```

或者跳过复制：运行 **`coddy`** 并输入 **`/login`**，它会登录 NeuralDeep（或
**`/login codex`**、**`devin`**、**`opencode`**），并把该提供商及其模型写入
**`~/.coddy/config.yaml`**（[控制台](../surfaces/console.md#登录提供商-login)）。

### 升级与移除

通过包管理器升级，或者让 Coddy 以 root 身份为你获取发布软件包 - 两者
结果相同，而以普通用户身份运行 **`coddy update`** 会明确告知，而不是悄悄
替换由软件包管理的文件（参见 [update.md](update.md#由包管理器拥有的安装)）：

```bash
sudo apt-get install ./coddy_<newer>_linux_amd64.deb   # or dnf install ./...rpm
sudo coddy update -y                                   # downloads and installs the package
coddy serve install                                    # if you run the service: restart it on the new binary
```

正在运行的服务会一直保留它启动时所用的二进制，直到它重新启动，而这正是最后一行
所做的；升级过程会打印同样的提醒。

```bash
coddy serve uninstall        # first, as each user that enabled the service
sudo apt-get remove coddy    # or: sudo dnf remove coddy
rm -rf ~/.coddy ~/Coddy      # only if you also want the sessions, config and workspace gone
```

软件包移除无法触及每个账户的 **`~/.config`**，因此它会让已启用的
服务保持启用状态，并打印清除它的命令
（[移除服务](../operate/serve.md#移除服务)）。

没有可订阅的 apt 或 dnf 仓库：软件包是发布资产，因此新版本
会在你安装较新文件或运行 **`sudo coddy update`** 时到来，而不是来自后台
**`apt upgrade`**。

### 自行构建软件包

```bash
make deb
make rpm
```

参见 [build.md](../contributing/build.md#分发包)。

## macOS（Homebrew）

```bash
brew install coddy-project/coddy-agent/coddy
```

该 cask 位于 Coddy 自己的 tap，[coddy-project/homebrew-coddy-agent](https://github.com/coddy-project/homebrew-coddy-agent)，
每次发布都会更新它。该命令会添加该 tap、信任这一个 cask，并从最新发布的 macOS 归档中安装
**`coddy`** 二进制，另外还有 **`man coddy`** 以及 bash 和 zsh
补全。它需要 macOS 12 或更高版本。它不会安装 **`tmux`**：其注意事项会在
安装后以及通过 **`brew info --cask coddy`** 打印，推荐使用 **`brew install tmux`** 以获得能在终端关闭后
继续存在的控制台会话。它们还会给出针对 **`~/.zshrc`** 的两行内容，让 zsh 加载
补全：macOS 自带的 zsh 不运行任何 **`compinit`**，没有它，Tab 就不会补全
**`coddy`**（[Homebrew](homebrew.md#当前安装)）。

升级跟随发布版本：

```bash
brew upgrade --cask coddy
```

Homebrew 7 在第三方 tap 被信任之前不会从中加载任何内容。使用完整名称安装会让
Homebrew 信任该 cask，这就是后续命令可以使用短名称 **`coddy`** 的原因。要信任
整个 tap，请运行一次 **`brew trust coddy-project/coddy-agent`**；此后 **`brew install --cask coddy`** 也可以
按名称使用。移除通过 Homebrew 进行：

```bash
brew uninstall --cask coddy      # brew zap --cask coddy also removes ~/.coddy
```

Homebrew 会为每个 cask 下载设置 quarantine 属性，而 Coddy 二进制未
经过公证，因此该 cask 会在安装后从 **`coddy`** 移除该属性，macOS 才会允许
首次运行通过。如果 macOS 仍然阻止首次运行，请自行移除它：
**`xattr -d com.apple.quarantine "$(which coddy)"`**。

本页面过去给出的命令 **`brew install --cask https://github.com/coddy-project/coddy-agent/releases/latest/download/coddy.rb`**
已不再有效：当前的 Homebrew 会拒绝通过 URL 或文件路径给出的 cask，并回复
**`No Cask with this name exists`**。[故障排查](troubleshooting.md#brew-说-no-cask-with-this-name-exists)
会把以那种方式安装的 Coddy 迁移到该 tap。

在从未添加过该 tap 的 Mac 上，**`brew install coddy`** 需要 Coddy 位于 Homebrew 自己的
某个仓库中，而这一点仍未解决。Homebrew 会把开源命令行软件以从源码构建的 formula 形式
路由到 **homebrew/core**，因此该 tap 是我们自己的分发渠道，而不是
一次提交 - 完整路径，包括当前阻碍它的因素，都在 [homebrew.md](homebrew.md) 中。

**`coddy update`** 能识别 Homebrew 安装，并指回 **`brew upgrade`**，而不是
替换 Homebrew 跟踪的文件：对于此 cask 使用 **`brew upgrade --cask coddy`**，对于
formula 使用 **`brew upgrade coddy`**。Homebrew 拒绝在 **`sudo`** 下运行，因此那里没有
特权捷径。

## Android (Termux)

```bash
pkg install curl
curl -fsSL https://coddy.dev/install.sh | bash
```

在 Termux 中，脚本会获取适用于 Android 的构建，在 64 位 ARM 设备上为 **`coddy_X.Y.Z_android_arm64.tar.gz`**，
在 x86_64 上为 **`coddy_X.Y.Z_android_amd64.tar.gz`**。Linux 归档在那里无法
启动：面向 Android 10 或更高版本的 Termux 会通过 Android 的链接器运行每个
程序，而它会以 **`has unexpected e_type: 2`** 拒绝静态可执行文件。其余安装过程与 Linux 相同：**`~/.local/bin`**、
**`~/.local/share`** 中的 man 手册页和补全，以及 **`~/.bashrc`** 中的块。设备上
的不同之处（Coddy 启动的程序、证书、在后台运行 **`coddy serve`**）
在它自己的页面上：[Android (Termux)](android.md)。

## 安装之后

```bash
export PATH="$HOME/.local/bin:$PATH"
coddy -v
# edit ~/.coddy/config.yaml, or run coddy and type /login
coddy serve            # in this terminal
coddy serve --daemon   # in the background, restarted if it dies
coddy serve install    # Linux: as a systemd user service
```

在带 systemd 的 Linux 上，**`coddy serve install`** 会改为将它作为你账户的用户服务运行：
它会启用软件包安装的单元，或为安装脚本放置的二进制写入一个单元，并让它
在 **`~/Coddy`** 中开始工作。**`coddy serve uninstall`** 会再次移除它。关于日志、在注销后
保持其运行以及移除它，参见
[服务指南](../operate/serve.md#在-linux-上作为-systemd-用户服务)。**`coddy serve --daemon`** 是没有
systemd 时的途径。

## Windows

### 安装位置

| 内容 | 路径 |
|------|------|
| 二进制 | `%LOCALAPPDATA%\Programs\coddy\coddy.exe` |
| 配置 | `%USERPROFILE%\.coddy\config.yaml` |
| 会话 / 记忆 | `%USERPROFILE%\.coddy\sessions\` |

用户目录是 **`$env:USERPROFILE`**（`%USERPROFILE%`），**不是** `$HOME` —— `$HOME` 在 Windows PowerShell 和 Git Bash 环境中并不可靠（Git Bash 的 `$HOME` 可能与 `%USERPROFILE%` 不同）。

### 当前会话中的 PATH

`install.ps1` 会把二进制目录加入**用户** `PATH`。新终端会自动获取它；
你安装时所用的终端**不会**。要么打开一个新终端，要么就地刷新：

```powershell
$env:Path = [Environment]::GetEnvironmentVariable("Path","User") + ";" + [Environment]::GetEnvironmentVariable("Path","Machine")
```

（如果你有 Chocolatey，`refreshenv` 也可以。）

### 编辑器 / 智能体集成：使用绝对路径

一些框架会通过 `cmd /c` 或 `sh -c` 启动 **`coddy acp`**，并且不会继承用户 `PATH`。为了避免 "command not found" 这样的配置问题，请用绝对路径配置客户端：

```text
%LOCALAPPDATA%\Programs\coddy\coddy.exe
```

### 包管理器

Scoop / winget 清单尚未发布 —— 跟踪于 [issue #42](https://github.com/coddy-project/coddy-agent/issues/42)。在此之前，请使用上面的 `install.ps1` 并用 `coddy update -y` 升级。

## Docker

```bash
docker compose pull && docker compose up -d
```

参见 [Docker](docker.md)。

## 升级

```bash
coddy update -y
```

参见 [update.md](update.md)。

## 从源码构建

参见 [build.md](../contributing/build.md) 以及 README 的 **Other installation methods** 章节。

<!-- docsgen:source sha256=45b2fa2b21c1f2c8 -->

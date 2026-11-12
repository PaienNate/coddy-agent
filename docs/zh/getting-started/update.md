# 更新 Coddy

使用 **`coddy update`** 从 [GitHub Releases](https://github.com/coddy-project/coddy-agent/releases) 下载官方发布二进制，并替换你正在运行的 **`coddy`** 可执行文件。

在 Windows 上，Coddy 会从系统临时目录启动一个短生命周期的辅助程序。该辅助程序等待 `coddy update` 退出，保留已安装可执行文件的备份并将其替换。辅助程序随后不会启动 Coddy：只有当你自己运行 `coddy` 时，控制台才会打开。辅助程序的状态行会在 `update` 命令返回后继续显示在同一个 `cmd.exe` 或 PowerShell 控制台中，因此 `coddy update` 报告成功的时间点是下载已暂存，而不是替换已经发生。

每个下载都会在解包或安装任何内容之前，对照发布所公布的 `SHA256SUMS` 资产进行校验，并且该清单还会由二进制内置的公钥对照其签名 `SHA256SUMS.minisig` 进行校验，因此即使有人能上传发布资产，也无法同时上传一份匹配的清单。从 1.3.68 起的发布如果缺少其中任一文件、签名无法验证，或签名对应的是另一个 tag，都会被拒绝。更早的发布若没有这些文件，会给出提示并仍然安装；名称中包含 `--repo` 的 fork 发布也是如此，它有自己的密钥。签名采用 minisign，因此 `minisign -Vm SHA256SUMS -P <public key>` 会手动校验它。

## 会安装什么

CI 会在每个 SemVer tag **`X.Y.Z`** 上为每个平台发布一个归档包，以及 Linux 发行版软件包：

| 归档包 | 平台 |
|---------|----------|
| **`coddy_X.Y.Z_linux_amd64.tar.gz`** | Linux x86_64 |
| **`coddy_X.Y.Z_linux_arm64.tar.gz`** | Linux arm64 |
| **`coddy_X.Y.Z_android_arm64.tar.gz`** | Android arm64，在 Termux 下（[Android](android.md)） |
| **`coddy_X.Y.Z_android_amd64.tar.gz`** | Android x86_64，在 Termux 下 |
| **`coddy_X.Y.Z_windows_amd64.zip`** | Windows x86_64（**`coddy.exe`**） |
| **`coddy_X.Y.Z_windows_arm64.zip`** | Windows arm64（**`coddy.exe`**） |
| **`coddy_X.Y.Z_windows_arm64.zip`** | Windows arm64（**`coddy.exe`**） |
| **`coddy_X.Y.Z_darwin_amd64.tar.gz`** | macOS Intel |
| **`coddy_X.Y.Z_darwin_arm64.tar.gz`** | macOS Apple Silicon |
| **`coddy_X.Y.Z_linux_amd64.deb`** | Debian、Ubuntu 及衍生版，x86_64 |
| **`coddy_X.Y.Z_linux_arm64.deb`** | Debian、Ubuntu 及衍生版，arm64 |
| **`coddy_X.Y.Z_linux_amd64.rpm`** | Fedora、RHEL、openSUSE 及衍生版，x86_64 |
| **`coddy_X.Y.Z_linux_arm64.rpm`** | Fedora、RHEL、openSUSE 及衍生版，arm64 |

Linux、Android 和 macOS 归档包会在二进制旁边附带手册页以及 bash 和 zsh 补全脚本；软件包会把这些相同文件安装到系统期望的位置，参见[安装](install.md#linux-软件包debrpm)。每次发布还会公布 **`coddy.rb`**，即该 tag 下两个 macOS 归档包的 Homebrew cask，并将同一文件提交到 tap [coddy-project/homebrew-coddy-agent](https://github.com/coddy-project/homebrew-coddy-agent)，**`brew install coddy-project/coddy-agent/coddy`** 和 **`brew upgrade`** 正是从这里读取它。

每个二进制都使用完整的 tag 集合构建：**`http`**、**`ui`**、**`scheduler`**、**`memory`**、**`cli`**、**`gateway`**、**`swarm`**、**`metrics`**、**`browser`** 和 **`desktop`**，与 **`make build TAGS="http ui scheduler memory cli gateway swarm metrics browser desktop"`** 和默认 Docker 镜像相同。发布流水线参见[从源码构建](../contributing/build.md#发布二进制ci)。

## 会替换哪个文件

**`coddy update`** 会解析正在运行的可执行文件（跟随符号链接）并覆盖该路径。在 Android 上，Termux 可能通过系统链接器启动 Coddy，此时替换的是传给链接器的二进制，而不是链接器本身。示例：

- 以普通用户身份执行 **`make install`** 后，通常是 **`~/.local/bin/coddy`**。
- 运行 **`./build/coddy update`** 时，它会更新仓库中的 **`build/coddy`**。

这与 **`make install`** 不同，后者始终复制到 **`~/.local/bin`** 或 **`/usr/local/bin`**。要更新 **`PATH`** 上的二进制，请调用 **`which coddy`** 打印的同一条 **`coddy`**。

## 手册页及其旁边的补全脚本

Linux、Android 和 macOS 归档包会在二进制旁边附带 **`coddy.1`**、**`coddy.bash`** 和 **`coddy.zsh`**，安装脚本会把它们放入二进制前缀的 **`share`** 目录（**`~/.local/bin`** -> **`~/.local/share`**，**`/usr/local/bin`** -> **`/usr/local/share`**；参见[安装](install.md)）。**`coddy update`** 会在可执行文件之后立即刷新它在那里找到的每个此类文件，因此 **`man coddy`** 和 Tab 补全描述的是正在运行的发布版本。停留在上一个发布版本的补全脚本会继续提供二进制已不再拥有的命令，这就是为什么在二进制已经更新后，某个补全脚本仍会列出 **`http`** 和 **`gateway`**，却没有 **`serve`**（[issue #188](https://github.com/coddy-project/coddy-agent/issues/188)）。

不会创建任何内容：从未安装过的文件（**`--no-shell-setup`**、手动复制的二进制）会保持原样，而位于 **`bin`** 目录之外的可执行文件（构建树、裸下载）没有可配对的 **`share`** 目录。如果发布早于归档包包含这些文件的时期，已安装的副本会保持原样，并会说明这一点。无法写入的文件会在二进制安装完成后报告，并且命令以非零状态退出。

## 已经在运行的服务器

**`coddy update`** 会替换磁盘上的文件。已经在运行的 **`coddy serve`** 会一直保留它启动时使用的二进制，直到重启：systemd 用户服务通过 **`coddy serve install`** 重启（它还会把自己写入的 unit 指向已移动的二进制，参见[服务指南](../operate/serve.md#在-linux-上作为-systemd-用户服务)），守护进程则通过 **`coddy serve restart`** 重启。软件包升级会打印同样的提醒。

## 由包管理器拥有的安装

由 **`apt`** 或 **`dnf`** 放到磁盘上的 **`coddy`** 会逐文件列在包数据库中。原地覆盖 **`/usr/bin/coddy`** 会让该数据库仍然描述一个已不存在的构建，下一次 **`apt upgrade`** 或 **`dnf reinstall`** 会悄悄把旧版本放回来，而 **`dpkg --verify`** 会报告一个没人想要的校验和不匹配。因此 **`coddy update`** 不会替换由软件包管理的可执行文件。它会询问 **`dpkg-query -S`** 和 **`rpm -qf`** 谁拥有它即将写入的文件，并走另外两条路径之一：

- **作为普通用户**：它不做任何更改，指出拥有该安装的软件包，并打印升级它的命令（**`sudo apt-get install --only-upgrade coddy`**、**`sudo dnf upgrade coddy`**、...）。退出码为 **1**，以便脚本察觉；
- **作为 root**：它会下载适用于此架构的 **`.deb`** 或 **`.rpm`**，像任何其他资产一样对照已签名的 **`SHA256SUMS`** 进行验证，然后把它交给找到的包管理器（**`apt-get`**、**`apt`**、**`dnf`**、**`yum`**、**`zypper`**，否则为 **`dpkg`** 或 **`rpm`**）。包数据库保持正确。

**Homebrew** 在 macOS 和 Linux 上也能被识别：解析后位于 **`Caskroom`** 或 **`Cellar`** 目录中的可执行文件属于 brew。这里没有特权路径（Homebrew 拒绝在 **`sudo`** 下运行），因此 root 和普通用户都会被指向 **`brew upgrade`**。匹配到两个标记中的哪一个决定使用哪个标志：**`Caskroom`** 路径是 cask，使用 **`brew upgrade --cask coddy`**；**`Cellar`** 路径是 formula，使用 **`brew upgrade coddy`**。这一区别并非表面功夫，在 formula 安装上执行 **`--cask`** 会失败，因为不存在该名称的 cask 可供升级。

```bash
sudo coddy update -y
```

所有权是向包数据库询问的，而不是根据路径猜测，因此你从 **`/usr/bin`** 中复制出来的二进制会继续自我更新，发行版自己重新构建的 Coddy 会像我们的一样被识别，而在 **`~/.local/bin`** 下的构建不会受到这些机制的任何影响。

在此流水线之前发布的版本有归档包但没有软件包。root 会被告知这一点，并被指回包管理器，而不是拿到一个会覆盖由软件包管理的文件的归档包。

## 命令

检查是否有更新的发布（如果已是最新则退出码为 **0**，如果存在更新的 **`X.Y.Z`** 则退出码为 **1**）：

```bash
coddy -v
coddy update --check
```

安装最新发布（除非 **`-y`**，否则提示 **`[y/N]`**）：

```bash
coddy update
coddy update -y
```

安装特定 tag：

```bash
coddy update --version 0.9.3
coddy update --version 0.9.3 -y
```

覆盖 GitHub 仓库（默认 **`coddy-project/coddy-agent`**）：

```bash
coddy update --repo coddy-project/coddy-agent
```

为更早发布编写的脚本会传入 **`--no-restart`**，它仍会被接受，且没有效果。

安装时不显示变更报告（参见[变更内容](#变更内容)），适用于只需要安装输出行的脚本：

```bash
coddy update -y --no-notes
CODDY_UPDATE_NOTES=0 coddy update -y
```

所有标志：

```bash
coddy update --help
```

## 变更内容

![更新后的报告：每个被跳过的发布及其说明和比较链接](../../assets/update-report-dark-1920.png)

*更新后的报告：每个被跳过的发布及其说明和比较链接*

更新完成后，**`coddy update`** 会说明带来了哪些变化。它会列出从正在运行的版本到刚安装版本之间的每个发布，从最旧到最新，每个发布都包含发布日期和发布说明，并以整个范围的 GitHub 比较链接结束（[issue #195](https://github.com/coddy-project/coddy-agent/issues/195)）：

```console
Installed 1.0.37 (/home/user/.local/bin/coddy)

Changes since 1.0.35:
1.0.36 (2026-09-11)
  - fix(update): refresh the man page and the shell completions beside the binary (#193)
1.0.37 (2026-09-11)
  - feat(config): --dry-run probes what config.yaml points at before anything starts (#194)
Full changelog: https://github.com/coddy-project/coddy-agent/compare/1.0.35...1.0.37
```

说明是 GitHub 保存的发布正文，针对终端做了裁剪：每个发布中自动生成的 *What's Changed* 标题、作者尾注和 *Full Changelog* 页脚会被去掉，pull request 编号保留为 **`(#N)`**，手写正文则将其章节保留为普通标题。报告最多 20 行，因此间隔较长时会以 **`... N more lines`** 和比较链接结束，该链接一次性覆盖所有发布。

报告会在所有会安装内容的路径上打印，包括归档包、Windows 辅助程序交接，以及作为 root 的 **`.deb`** / **`.rpm`** 路径，从不在 **`--check`** 上打印，也不会在 Coddy 已是最新时打印。没有可用于计数的发布的构建（**`coddy -v`** 会打印 **`dev`**），以及向后回退的 **`--version`**，会得到刚安装发布的说明及其页面链接，而不是一个范围。

获取列表是对 GitHub API 的第二个请求，限时 15 秒，且不会导致更新失败：离线、被限流或返回的不是列表时，报告会缩减为仅剩 **`Full changelog:`** 这一行。**`--no-notes`** 会完全跳过它，环境中的 **`CODDY_UPDATE_NOTES=0`**（也包括 **`false`**、**`no`** 或 **`off`**）也会如此，适用于只需要安装输出行的脚本和 CI 步骤。

## 版本比较

**`coddy -v`** 可能会显示 git describe 字符串（例如 **`0.9.2-5-gb6b7d31-dirty`**）。**`coddy update`** 会将开头的 **`X.Y.Z`** 前缀与发布 tag 进行比较。本地 **`dev`** 构建会被视为比任何已发布的 SemVer 发布更旧。

## 其他升级路径

| 方法 | 使用场景 |
|--------|-------------|
| **`coddy update`** | 你已经在磁盘上有一个发布二进制，想要下一个（或特定）GitHub 发布。 |
| **`make install`** | 你从克隆构建，并想要在 **`PATH`** 上得到 **`build/coddy`**。 |
| **`make build TAGS="..."`** | 你需要发布中没有的自定义 tag 或本地更改。 |
| **Docker** | 在 [GHCR](https://github.com/coddy-project/coddy-agent/pkgs/container/coddy-agent) 上使用 **`docker compose pull`** / 镜像 tag **`X.Y.Z`**。 |
| **`go install ...@latest`** | 无需发布资产的快速安装；仅默认模块 tag（除非从源码构建，否则没有 **`http`** / UI）。 |
| **`apt` / `dnf` / `zypper`** | 你安装了 **`.deb`** 或 **`.rpm`**；安装更新的软件包文件，或运行 **`sudo coddy update`** 让它为你获取。 |
| **`brew upgrade --cask coddy`** | 你安装了 Homebrew cask（可执行文件解析后位于 **`Caskroom`**）。 |
| **`brew upgrade coddy`** | 你安装了 Homebrew formula（可执行文件解析后位于 **`Cellar`**）。 |

## 限制

- 仅支持发布表中列出的平台；其他平台会得到明确的错误。
- 在 Windows 上，Coddy 最多等待 30 秒，让另一个进程释放可执行文件。权限失败会更快报告，并指出目录：安装到 `Program Files` 需要提升权限的控制台。如果无法安装更新，当前可执行文件会保留在原位；如果更新后的二进制无法启动，Coddy 会恢复备份。
- Windows 辅助程序会通过它刚刚安装的 Coddy 删除自身。安装早于该交接的发布（`coddy update --version` 向后回退）会使辅助程序留在 `%TEMP%`，直到下一次 `coddy update` 将其清理。
- 资产下载会在临时连接失败后恢复（最多三次尝试）。GitHub 支持 Coddy 用于恢复的 HTTP range 请求；不支持 range 的服务器会从头重新下载，而在错误偏移处恢复的服务器会使下载失败，而不是安装一个拼接的归档包。
- **`coddy update`** 需要向 **`api.github.com`** 和资产 CDN（GitHub 发布下载）发起出站 HTTPS 连接。
- 软件包路径仅限 Linux（Homebrew 安装会被报告，Coddy 从不会升级它们），并且包管理器以非交互方式运行，因此会移除另一个软件包的升级会失败，而不是询问。在这种情况下，请自行运行包管理器。
- **`$CODDY_HOME`** 下的配置不会被修改；只会替换二进制，或承载它的软件包。

<!-- docsgen:source sha256=048072a557c38fbb -->

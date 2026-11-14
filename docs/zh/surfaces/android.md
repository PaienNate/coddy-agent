# Android 应用

Android APK 是手机上的完整 Coddy 服务器：同一个二进制、同一个会话存储和同一个
Web 界面，运行在应用沙箱里。它不是指向别人服务器的外壳 - 应用在环回地址上启动
**`coddy serve`**，其 WebView 加载的就是这个页面。构建位于 **`android/`**，
workflow **`.github/workflows/android-apk.yaml`** 在每次运行时产出 debug APK。

## 它是什么

- **`libcoddy.so`** - 由 `make android` 构建的 Go 二进制。Android 10 及更高版本
  禁止执行应用数据目录中的文件，而 `lib/<abi>` 下的原生库是可执行的，因此二进制
  放在 `jniLibs` 中，并由 `packaging.jniLibs.useLegacyPackaging = true` 保持为
  真实文件。
- **`libbusybox.so`** - 固定版本的静态 BusyBox，见下一节。
- **`CoddyService`** - 运行 `libcoddy.so serve -H 127.0.0.1 -P 8765` 的前台服务，
  因此即使用户切到其他应用，回合仍会继续。它的通知就是为此付出的代价。
- **`MainActivity`** - 一个加载 `http://127.0.0.1:8765/` 的 WebView，启动页会轮询
  `/coddy/info`，直到服务器应答。

对外只暴露环回地址：提供商、远程 MCP 服务器和 SSH 主机都是主动外连，因此手机
不需要入站端口，也不需要局域网地址。

## 构建与安装

CI 在每个触及 `android/` 的拉取请求上以及 `workflow_dispatch` 时构建 debug APK；
从运行记录下载 **`coddy-android-apk`**，用
`adb install coddy-android-<sha>-debug.apk` 安装，或直接在手机上打开该文件。
发布版构建需要你自己的签名密钥。

本地构建（需安装 Android SDK、NDK、Go 和 Node）：

```bash
android/scripts/build-apk.sh                       # arm64 与 x86_64
ANDROID_ARCHS=arm64 android/scripts/build-apk.sh   # 仅手机
```

脚本执行 `make android TAGS="http ui scheduler memory"`，把二进制放入 `jniLibs`，
获取 BusyBox，然后运行 `gradle :app:assembleDebug`。

## 首次启动

应用在首次启动时写入 **`CODDY_HOME/config.yaml`**，绑定到环回地址。随后 Web 界面的
引导流程会询问提供商 - 与桌面服务器一致 - 之后设置页面编辑的仍是同一个文件。

状态保存在应用沙箱内：`files/coddy` 是 `CODDY_HOME`（配置、会话、SSH 凭据），
`files/workspace` 是默认工作目录，`cache/tmp` 是 `TMPDIR`，因为 Android 上 Go 的
默认值 `/data/local/tmp` 不可写。

## SSH 凭据

**设置 → SSH** 保存 **`ssh_run_command`** 使用的主机，存放在
**`<CODDY_HOME>/ssh-credentials.json`**，权限 **0600**。每个主机可以带密码、一条
以其去空白 stdout 作为密码的凭据命令、私钥路径，或什么都不带 - 此时工具一如既往
地回退到 SSH agent 和 `~/.ssh` 下的密钥。密码认证同时应答 keyboard-interactive，
基于 PAM 的服务器正是通过它索取同一个密码。

页面永远不会收到密码：列表只报告 **has_password**，以空密码保存主机会保留原密码。
**测试**按钮用与工具相同的认证链连接主机并立即关闭，不执行任何命令；连接被拒时
会带回原因文字，页面原样显示。尚未保存的凭据同样可以测试。

## BusyBox

Android 自带的工具集（toybox）随系统版本变化，而且缺少 shell 回合常用的大部分
命令。APK 内置静态链接的 BusyBox（Alpine 的 `busybox-static`，在
**`android/scripts/fetch-busybox.sh`** 中按版本和 SHA-256 固定），首次启动时为每个
名称在 `files/bin` 建立一条符号链接，服务器把它放在 `PATH` 最前面：`sed`、`awk`、
`grep`、`find`、`xargs`、`tar`、`gzip`、`sort`、`uniq`、`tr`、`cut`、`diff`、
`mktemp`、`timeout`、`nc`、`base64`、`xxd`、`md5sum` 以及 BusyBox 的 `ash`。
`httpd`、`telnetd` 这类守护进程有意不链接。

BusyBox 以 GPLv2 发布，作为独立进程运行（与 MIT 许可的二进制构成 mere
aggregation）。许可证文本随 APK 分发，源码即 BusyBox 1.36.1 发行版，记录在
**`android/THIRD_PARTY.md`**。

应用是 BusyBox 环境而非 GNU：需要 GNU 语义的 `sed -i`，或应用未内置的工具
（`git`、`jq`、`python`、`curl`），应在远程主机上通过 SSH 或 MCP 服务器执行。
读取网页使用内置的 **`webfetch`** 工具，而不是本地 `curl`。

## 限制

- **环回端口没有身份验证。** 同一手机上的其他应用可以访问 `127.0.0.1:8765`。
  服务器只绑定环回，应用是唯一预期的客户端；加令牌或配对步骤是下一步加固，
  不是本构建的承诺。
- **固定端口 8765。** 手机上的第二个 Coddy 服务器会等待这一个停止。
- **不能自更新。** `coddy update` 替换磁盘上的二进制；这里的二进制是 APK 的一部分，
  因此升级就是安装新的 APK。
- **工作区是应用私有的。** 其外的文件需要后续工作；手机共享存储不会挂载进会话。

<!-- docsgen:source sha256=279e172628a5b383 -->

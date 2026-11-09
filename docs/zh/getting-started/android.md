# Android（Termux）

Coddy 可在 Android 上通过 [Termux](https://termux.dev) 运行，使用的是为 Android 构建的发布版本：
**`coddy_X.Y.Z_android_arm64.tar.gz`** 面向 64 位 ARM（aarch64），即手机和平板电脑的处理器；**`coddy_X.Y.Z_android_amd64.tar.gz`** 面向 x86-64（x86_64），Android 在模拟器、Chromebook 和 PC 上运行的就是该架构。它包含完整的 Coddy：控制台、带 Web 界面的 **`coddy serve`**、Telegram 机器人以及 **`coddy acp`**。需要 Android 7.0 或更高版本，与 Termux 本身的要求相同。

## 安装

安装脚本会识别 Termux，并获取适用于设备处理器的 Android 构建：

```bash
pkg install curl
curl -fsSL https://coddy.dev/install.sh | bash
```

它会将 **`coddy`** 安装到 **`~/.local/bin`**，将 man 手册页和补全安装到
**`~/.local/share`**，并把将它们加入 **`PATH`** 的配置块写入 **`~/.bashrc`**，Termux 的登录 shell 会读取该文件。再次运行脚本会替换它此前安装的二进制文件，包括之前尝试遗留在那里的 Linux 构建。打开一个新会话，或重新加载该文件，然后检查：

```bash
source ~/.bashrc
coddy -v
```

手动操作时，将 **`X.Y.Z`** 替换为所需的发布版本，并在
**`uname -m`** 显示 **`x86_64`** 时将 **`arm64`** 替换为 **`amd64`**：

```bash
curl -fsSLO https://github.com/coddy-project/coddy-agent/releases/download/X.Y.Z/coddy_X.Y.Z_android_arm64.tar.gz
tar -xzf coddy_X.Y.Z_android_arm64.tar.gz
install -m 0755 coddy "$PREFIX/bin/coddy"
```

**`$PREFIX/bin`** 已在 Termux 的 **`PATH`** 中。请将二进制文件放在 Termux 自己的存储中，如上述两个示例所示：Android 挂载共享存储（**`/sdcard`**、**`~/storage`**）时不允许从其中执行任何文件。

首次运行需要像在其他系统上一样，在 **`~/.coddy/config.yaml`** 中配置提供商密钥；参见
[配置](configuration.md)。

## Linux 归档为何无法运行

Linux 归档中包含的是静态 Linux 可执行文件，而 Termux 无法在所有环境中启动它。
Android 10 及更高版本拒绝授予以它们为目标的应用从其自身数据目录执行文件的权限，因此，以这些版本为目标的 Termux（例如 Google Play 构建）会通过 Android 的链接器 **`/system/bin/linker64 <program>`** 启动每个程序。该链接器仅加载位置无关的可执行文件，会拒绝静态可执行文件：

```text
error: "/data/data/com.termux/files/home/.local/bin/coddy" has unexpected e_type: 2
```

在 Termux 直接执行文件的环境中（F-Droid 和 GitHub 构建），Linux 二进制文件可以启动，但它依赖的 Linux 路径缺失：它找不到 CA 证书和 **`/etc/resolv.conf`**，
因此每次向提供商发出的请求都会失败；而在 seccomp 过滤器拒绝
**`faccessat2`** 系统调用的 Android 版本上，它首次查找程序时就会被以 **`Bad system call`** 终止。

Android 归档使用 **`GOOS=android`** 构建，并由 Android NDK 链接到 Bionic（Android 的 C 库）：这是一个位置无关的可执行文件，将 **`/system/bin/linker64`**
指定为解释器，仅需要每个 Android 都带有的库（**`libc`**、**`libdl`**、
**`liblog`**），两种 Termux 都能启动它。与任何 Termux 程序一样，它通过 Android 本身解析主机名，因此网络的 DNS、Private DNS 以及 VPN 的 DNS 都会生效。本页其余部分介绍该构建在运行时的不同行为。

## Coddy 在 Android 上所做适配

无需任何配置：Coddy 在启动时会读取 Termux 环境。

- **它启动的程序。** **`run_command`**、后台任务、钩子、基于 stdio 的 MCP 服务器、git 和
  ripgrep 都会按照 Termux 启动自身程序的方式启动。在 Termux 经由
  **`/system/bin/linker64`** 的地方，Coddy 也会如此，脚本会交给其解释器执行；首行指定 **`/usr/bin/env`** 或 **`/bin/sh`** 的脚本会获得同名的 Termux 程序。
  这正是 **`termux-exec`** 库为 Termux 自身程序所做的；Coddy 与任何 Go
  程序一样，通过该库无法看到的系统调用来启动进程，因此它自行完成同样的工作。
  shell 命令在 Termux 的 **`bash`** 中运行，行为与在你的会话中一致。
- **它自身的路径。** 通过链接器启动时，Coddy 从命令行而非
  **`/proc/self/exe`**（该值指向链接器）找到自己的二进制文件，因此 **`coddy update`** 和
  **`coddy serve --daemon`** 会指向正确的文件。
- **证书。** 除 Android 的系统存储外，Coddy 还信任 Termux 的 **`ca-certificates`** 包
  **`$PREFIX/etc/tls/cert.pem`** 中的证书束。它通过
  **`SSL_CERT_FILE`** 实现，因此你自己导出的证书束会优先。
- **临时文件。** **`TMPDIR`** 回退到 **`$PREFIX/tmp`**：Android 的默认值
  **`/data/local/tmp`** 对应用不可写。
- **打开浏览器。** 诸如 **`coddy providers login`** 的登录流程会使用 **`termux-open-url`** 打开手机的浏览器，
  浏览器会访问 Coddy 在设备回环地址上监听的 回调。
- **更新。** **`coddy update`** 会下载新版本的 Android 归档并就地替换
  二进制文件；参见 [更新](update.md)。

## 运行 `coddy serve`

**`coddy serve`** 与其他地方一样工作，Web 界面会在手机的浏览器中通过 **`http://127.0.0.1:12345/`** 打开。Android 没有 systemd，因此 **`coddy serve install`** 不可用：请在 Termux 会话中、在 **`tmux`**（**`pkg install tmux`**）中，或使用 **`coddy serve --daemon`** 在后台运行服务器（参见 [coddy serve](../operate/serve.md)）。

Android 会停止它认为空闲的应用的进程。当服务器需要保持运行时，请获取 Termux 的唤醒锁（**`termux-wake-lock`**，或从 Termux 通知中获取），并在电池设置中允许 Termux 不受限制地运行。Android 12 及更高版本还会在超过限制时终止应用的 后台进程（"phantom process killer"，幻影进程杀手）；Termux 的文档介绍了如何解除该限制。

手机也可以作为集群的一个节点：它拨号连接笔记本电脑或服务器上的中继，中继的 Web 界面和控制台会像驱动其他机器一样驱动它，手机上无需开放端口。完整路径见 [将 Android 手机用作集群节点](../tutorials/swarm-android-nodes.md)。

## 限制

- 仅发布 64 位构建，即 arm64 和 x86_64：不支持 32 位 ARM 或 x86 的 Android 设备。
- 如果某个程序本身是静态可执行文件，那么在 Termux 通过链接器启动程序的环境中它无法运行，无论由谁启动；这是 Android 的规则，错误同样是 **`unexpected e_type`**。

## 自行构建

**`make android`** 会像为 **`make build`** 构建时一样，使用 **`TAGS`** 交叉编译 **`build/coddy-android-arm64`** 和 **`build/coddy-android-amd64`**，而 **`make check-android`** 会对 Android 构建进行类型检查。两者都需要 Android NDK，它会将二进制文件链接到 Bionic；参见
[从源码构建](../contributing/build.md#androidtermux)。

<!-- docsgen:source sha256=a6179313a35653f6 -->

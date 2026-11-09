# Homebrew

Coddy 目前通过 Homebrew 安装，来自它自己的一个 tap，尚未进入 Homebrew 自己的某个仓库。本页区分这两件事：用户现在运行什么，以及提交到 Homebrew 需要什么。

## Coddy 应属于哪个仓库

Homebrew 有两个官方 tap，并按包类型在二者之间引导软件。[Adding Software to Homebrew](https://docs.brew.sh/Adding-Software-to-Homebrew#choose-a-package-type) 将 Homebrew 能从源码构建的开源命令行软件作为 formula 发送到 **homebrew/core**，并将 **homebrew/cask** 保留给原生 macOS 应用以及专有或仅二进制软件。

Coddy 采用 MIT 许可证，仅提供命令行界面，并使用 Go 和 Node 从源码构建，因此提交目标是 **[homebrew/core](https://github.com/Homebrew/homebrew-core) 作为 formula**，而不是 homebrew/cask。[Acceptable Casks](https://docs.brew.sh/Acceptable-Casks#appropriate-package-type) 从另一面陈述了同一规则，并补充说，被 homebrew/core 拒绝本身并不能使软件有资格进入 homebrew/cask。

这是对 [#160](https://github.com/coddy-project/coddy-agent/issues/160) 中原计划的更正，原计划将提交目标定在 homebrew/cask。

我们自己的 **cask** 不会消失。它安装某个发布版的预构建 macOS 归档，无需 Go、无需 Node、无需构建，而且它正是我们的 tap 所承载、落地页所安装的内容。cask 对这种工作是正确形态；只是对 Homebrew 自己的仓库来说是错误形态。

## 当前安装

```bash
brew install coddy-project/coddy-agent/coddy
```

tap [coddy-project/homebrew-coddy-agent](https://github.com/coddy-project/homebrew-coddy-agent) 承载一个文件，即 **`Casks/coddy.rb`**。每次发布都会用它两个 macOS 归档的校验和渲染该文件，将其作为 **`coddy.rb`** 附加到发布版，并提交到 tap（**`scripts/publish-homebrew-tap.sh`**），因此 **`brew upgrade --cask coddy`** 会看到每个新发布版。给定完整名称，Homebrew 会添加该 tap 并信任那一个 cask。Homebrew 7 在获得信任之前不会从第三方 tap 加载任何其他内容；**`brew trust coddy-project/coddy-agent`** 会信任整个 tap，之后短名称便可在每条命令中使用。

发布资产本身不再能安装。在 **`HOMEBREW_DEVELOPER`** 模式之外，当前 Homebrew 只会从 tap 加载 cask：它会把 URL 读作 cask 名称，因此 **`brew install --cask <URL>`** 会以 **`No Cask with this name exists`** 结束（[#269](https://github.com/coddy-project/coddy-agent/issues/269)），并且它会用 **`Homebrew requires casks to be in a tap`** 拒绝本地文件。该资产仍保留给任何想要该文件的人；tap 才是安装途径。

每次发布后，**Homebrew tap** 工作流会在装有当前 Homebrew 的 macOS 运行器上运行该命令并检查结果；它还会每周运行，并在更改 cask 的拉取请求上运行（[build.md](../contributing/build.md#homebrew-tap)）。

该 cask 是为 Homebrew 7 编写的：补全通过 **`bash_completion`** 和 **`zsh_completion`** 提供，唯一的安装步骤是一个 **`postflight_steps`** 块，而不是 Ruby **`postflight`** 块，并且 **`url`** 不带有 **`verified:`**。该步骤会移除 **`com.apple.quarantine`**，Homebrew 会在每次 cask 下载时设置它：二进制是临时签名的，并未公证，而该属性存在时，Gatekeeper 会阻止其首次运行。公证需要 Apple Developer ID，本项目没有。该 cask 要求 macOS 12 或更高版本，因为由 Go 1.25 及更高版本构建的二进制无法在更旧的发布版上运行。

zsh 补全落在 **`$(brew --prefix)/share/zsh/site-functions/_coddy`**。Homebrew 安装器会添加到 **`~/.zprofile`** 的 **`brew shellenv`** 会把该文件夹加入 zsh 的 **`fpath`**，但 zsh 只有在 **`compinit`** 运行后才会读取补全，而 macOS 自带的 zsh 不会运行它。因此安装后，Tab 不会补全 **`coddy`**，直到 **`~/.zshrc`** 包含下面两行；caveats 也会打印这两行：

```zsh
fpath=(/opt/homebrew/share/zsh/site-functions $fpath)   # /usr/local on an Intel Mac
autoload -Uz compinit && compinit
```

Oh My Zsh 和其他框架会自行运行 **`compinit`**；在那里，只要 **`brew shellenv`** 先运行就足够了。如果 zsh 警告存在不安全目录，**`compaudit | xargs chmod g-w`** 会修复其权限，而调用了 **`compinit -C`** 的设置会跳过对新补全的检查：移除 **`~/.zcompdump*`** 并打开一个新终端。安装脚本会把同样的块写入 **`~/.zshrc`** 中供其自己的文件夹使用；cask 不得改动你的 dotfiles，因此它改为打印这些行。**Homebrew tap** 工作流会检查，在 **`compinit`** 运行后，zsh 是否从该文件夹注册 **`_coddy`**。

该 cask 只安装 Coddy，并在其 caveats 中推荐 **`tmux`**（**`brew install tmux`**）。cask 的 **`depends_on`** 没有可选形式，因此在其中声明 tmux 会为所有人安装它，而 Coddy 在没有它的情况下也能运行；deb 和 rpm 也以同样方式建议安装它。

该 tap 也是 Homebrew 官方仓库拒绝的任何内容的文档化归属地（[Package Acceptance Policy](https://docs.brew.sh/Package-Acceptance-Policy#scope-and-third-party-taps)），因此无论下面的提交结果如何，它都仍然有用。

## 该 formula

```bash
make brew-formula VERSION=1.0.13
```

**`scripts/build-homebrew-formula.sh`** 会用该标签源码归档的版本和 SHA-256 填充 **`packaging/homebrew/coddy-formula.rb.tmpl`**，并写入 **`dist/formula/coddy.rb`**。它会下载归档以计算哈希，因此版本必须是已发布的标签。

formula 构建发布二进制所承载的内容，即 **`http ui scheduler memory cli gateway swarm metrics browser desktop`**，因此 `brew install` 和发布归档具有相同的功能集。这就是为什么 **`node`** 是与 **`go`** 并列的构建依赖：嵌入的 SPA 是生成而非提交的，而 **`go:embed`** 需要这些文件在链接承载它们的二进制之前存在。它运行 **`npm ci`**，而不是 **`npm install`**，因为 [Acceptable Formulae](https://docs.brew.sh/Acceptable-Formulae#versioned-and-verifiable-sources) 禁止在构建期间解析会变化的依赖集。

该 formula 省略了 **`tmux`**。[Formula Cookbook](https://docs.brew.sh/Formula-Cookbook#specifying-other-formulae-as-dependencies) 不允许 formula 的可选和推荐依赖出现在 homebrew/core 中，并且 caveats 只保留给 Homebrew 打包本身改变的内容，因此该推荐仍留在 cask 和本站的这些页面中。

它的 **`test do`** 块会运行 **`coddy -v`**、**`coddy sessions list`** 和 **`coddy skills list`**，这三条命令不需要网络，也不需要配置，并且如果二进制无法访问 **`~/.coddy`**，它们就会失败。

**`coddy update`** 已经拒绝覆盖 Homebrew 拥有的文件，并改为打印 brew 命令，这正是 Coddy 满足 [Acceptable Formulae 关于自更新软件的规则](https://docs.brew.sh/Acceptable-Formulae#self-updating-software) 的方式。formula 安装会落在 **`Cellar`**，并用 **`brew upgrade coddy`** 升级；cask 会落在 **`Caskroom`**，并使用 **`brew upgrade --cask coddy`**。Coddy 根据自身可执行文件的路径区分二者，参见 [update.md](update.md#由包管理器拥有的安装)。

## 预检

```bash
make brew-check VERSION=1.0.13
```

**`scripts/check-homebrew-submission.sh`** 会回答 Homebrew 验收标准中可从外部读取的部分：**`coddy`** token 在两个仓库中是否空闲、仓库是否达到知名度阈值、发布版是否存在，以及渲染后的 formula 是否能解析、是否指明该发布版、是否固定校验和，并且不带有 **`bottle`** 块（BrewTestBot 会添加这些块，提交者不会）。当某些内容阻止提交时，它会以非零状态退出，并在末尾打印仅限 macOS 的步骤。

### 知名度是当前未解决的阻塞项

[Package Acceptance Policy](https://docs.brew.sh/Package-Acceptance-Policy#notability) 设定了两个阈值，适用哪一个取决于谁发起拉取请求：

| 提交者 | 阈值 |
|---|---|
| 与项目无关联的任何人 | 30 个 fork、30 个 watcher **或** 75 个 star |
| 仓库所有者提交自己的项目 | 90 个 fork、90 个 watcher **或** 225 个 star |

在 1.0.13 时，仓库有 139 个 star、14 个 fork 和 2 个 watcher。这满足第一行，但不满足第二行，因此**自我提交尚不符合条件**，而其他人提交则符合。针对当前数字运行预检；它会进行这项算术运算，并说明仓库还差多少。

两个阈值都不是承诺。Homebrew 的政策明确表示，满足文档化标准并不保证被接受，同样，缺少某一项标准也不强制拒绝：当指标不能真实反映实际使用时，例外情况应在拉取请求中论证，而不是当作一个待勾选的选项。

## 提交

此后的所有内容都需要一台已安装 Homebrew 的主机，并且应使用 macOS：**`brew audit`** 和 **`brew style`** 是审查者运行的内容，而 formula 必须在其主要面向的平台上得到验证。预检会打印这些命令，并填好路径。

```bash
brew tap --force homebrew/core
cd "$(brew --repository homebrew/core)"
git checkout -b coddy origin/HEAD
cp /path/to/coddy-agent/dist/formula/coddy.rb Formula/c/coddy.rb
brew install --build-from-source --verbose --debug coddy
brew test coddy
brew audit --strict --online --new coddy
brew style coddy
```

然后将分支推送到 **Homebrew/homebrew-core** 的 fork，并针对它发起拉取请求。该 fork 是 Homebrew 仓库的 fork，而不是我们仓库的 fork；[How to Open a Homebrew Pull Request](https://docs.brew.sh/How-To-Open-a-Homebrew-Pull-Request) 是 Homebrew 期望的流程，而 **`brew bump-formula-pr`** 用于后续版本提升，不用于新的 formula。

[homebrew/core 的 CONTRIBUTING](https://github.com/Homebrew/homebrew-core/blob/master/CONTRIBUTING.md) 允许在若干条件下提交借助 AI 或 LLM 准备的拉取请求，这些条件会影响这项工作如何交接（[Responsible AI Usage](https://docs.brew.sh/Responsible-AI-Usage) 说明了理由）：

- 拉取请求必须披露使用了 AI 或 LLM，并说明使用了哪个工具或模型；
- 提交者必须在请 Homebrew 审查之前，审查所有生成的内容；
- 任何提交都不得将工作归因于 AI 或 LLM 作为作者、共同作者、提交者或签署人，包括通过 `Assisted-by` 或 `Co-developed-by` trailer；
- 提交者必须亲自回答维护者的问题和审查意见，不得使用 AI 或 LLM；
- 非维护者同一时间只能有一个由 AI 协助的拉取请求处于打开状态。

因此，formula、本页以及围绕它们的脚本在这里准备，而提交本身由一个人发起并为其辩护。每个包只发起一个拉取请求，并且在推送更新后不要 squash 提交。

## 接受之后

formula 的 **`livecheck`** 块会跟踪 GitHub 发布版，因此 Homebrew 自己的自动化会发起版本提升，而提交是一次性工作，不是每次发布都要做的工作。我们这边变化很小：**`brew install coddy`** 会在一台从未添加该 tap 的 Mac 上开始工作，文档会优先介绍它而不是该 tap，并且 formula 安装会从 **`coddy update`** 报告 **`brew upgrade coddy`**，它目前已经这样做了。该 tap 的 cask 用户如何迁移到 formula，将随该变化一起决定。

<!-- docsgen:source sha256=cd8299de6ff8c836 -->

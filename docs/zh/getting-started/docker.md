# Docker

在精简的 **`scratch`** 镜像中以 **`coddy serve`** 方式运行 Coddy。默认镜像内置了嵌入式 Web 界面(**`ui`** 构建标签)、OpenAI 兼容 REST(**`http`**)、调度器(**`scheduler`**)、长期记忆(**`memory`**)、控制台(**`cli`**)、消息网关(**`gateway`**)、集群中继(**`swarm`**)以及 Prometheus 导出器(**`metrics`**)。

相关文件：

- [`Dockerfile`](../../../Dockerfile) - 多阶段构建(**Node** UI 产物、**Go** 二进制、**`scratch`** 运行时)
- [`docker-compose.yml`](../../../docker-compose.yml) - 运行 **`ghcr.io/coddy-project/coddy-agent`**(默认 **`docker compose`**)
- [`docker-compose.dev.yml`](../../../docker-compose.dev.yml) - 从源码构建、发布端口 **12345**、卷
- [`.dockerignore`](../../../.dockerignore) - 保持上下文小巧；切勿将带有密钥的 **`config.yaml`** 提交
- [`examples/httpserver/docker.sh`](../../../examples/httpserver/docker.sh) - 自动化冒烟测试

已发布的镜像：**[GHCR 上的 coddy-agent](https://github.com/coddy-project/coddy-agent/pkgs/container/coddy-agent)**(`ghcr.io/coddy-project/coddy-agent`)。CI 在 SemVer 标签上构建 **multi-arch** 清单(**`linux/amd64`**、**`linux/arm64`**)，并在适当时推送浮动别名(**`latest`**、**`MAJOR.MINOR`**、**`MAJOR`**) - 参见 [`.github/workflows/docker-build-push.yaml`](../../../.github/workflows/docker-build-push.yaml)。

在 Apple Silicon 或 arm64 Linux 主机上，照常拉取镜像；Docker 会自动选择 **`arm64`**，而该变体携带 **`aarch64`** 二进制，CI 会在推送前从镜像中读取它(参见 [检查镜像的平台](#检查镜像的平台))。要显式固定平台：

```bash
docker pull --platform linux/arm64 ghcr.io/coddy-project/coddy-agent:latest
docker pull --platform linux/amd64 ghcr.io/coddy-project/coddy-agent:latest
```

不使用 Docker 的一般构建说明 - **[docs/contributing/build.md](../contributing/build.md)**。

## 先决条件

- **Docker**，带 **Compose V2**(**`docker compose`**，而不仅是旧版的 **`docker-compose`**)
- 一个以只读方式挂载进容器的 **`config.yaml`**(从 **`config.example.yaml`** 开始)。不要提交密钥。
- 对于 Web 界面，需要一台能访问已发布主机端口的机器上的浏览器(默认 **12345**)
- 要从 [`Dockerfile`](../../../Dockerfile) 构建镜像(**`docker-compose.dev.yml`**、**`make check-image`**)，需要 **BuildKit**:Docker Engine **23** 或更新版本，并带有 **`buildx`** 插件(**`docker buildx version`** 应答),Docker 官方软件包和 Docker Desktop 均自带；Debian 和 Ubuntu 的 **`docker.io`** 需要在其旁边安装 **`docker-buildx`**。构建阶段运行在 **`$BUILDPLATFORM`** 上，而只有 BuildKit 会设置它，因此旧版构建器会在第一个 **`FROM`** 处停止，并给出 **`failed to parse platform`**

## Docker Compose

Compose 是运行已发布 GHCR 镜像或本地构建镜像的推荐方式。两个文件都定义了名为 **`coddy`** 的**单一服务** - 没有数据库或 Redis 边车。进程以 **`coddy serve`** 绑定到容器内的 **`0.0.0.0:12345`**;Compose 将该端口映射到主机。

| 文件 | 何时使用 |
|------|-------------|
| [`docker-compose.yml`](../../../docker-compose.yml) | 从 GHCR 拉取发布版本(**`docker compose pull`**) - 日常和类生产运行，无需在主机上构建 |
| [`docker-compose.dev.yml`](../../../docker-compose.dev.yml) | 从仓库 **`build:`** [`Dockerfile`](../../../Dockerfile) - 对 Coddy 进行改造、复现 CI，或运行 **`examples/httpserver/docker.sh`** |

Compose V2 会合并同目录下可选的 **`docker-compose.override.yml`**(按惯例被 git 忽略)，因此你可以固定镜像标签、更改端口或添加 **`environment`**，而无需编辑受跟踪的文件。

### `coddy` 服务运行什么

| 设置 | 默认 compose | 开发 compose |
|---------|-----------------|-------------|
| **镜像** | **`${CODDY_IMAGE:-ghcr.io/coddy-project/coddy-agent:latest}`** | **`coddy-agent:${CODDY_VERSION:-dev}`**(本地构建) |
| **命令** | 镜像 **`CMD`**: **`serve -H 0.0.0.0 -P 12345`**，可用 **`CODDY_COMMAND`** 覆盖 | 相同 |
| **发布端口** | **`${CODDY_HTTP_BIND:-127.0.0.1}:${CODDY_HTTP_PORT:-12345}:12345`**：仅主机的回环地址 | 相同 |
| **工作目录** | **`/workspace`**(**`CODDY_CWD`**) | 相同 |

**健康检查** - 每 5 秒执行一次 **`coddy --version`**(一旦二进制响应，容器即为 "healthy";HTTP 就绪状态是另一回事 - 在 **`up`** 之后使用 **`curl /v1/models`**)。

**日志** - 使用 **`json-file`** 驱动，每个文件 **`max-size: 100k`**，以避免在长时间运行的主机上日志无限增长。

### 主机路径与环境变量

绑定挂载(在 **`up`** 之前用环境变量覆盖主机路径):

| 主机(默认) | 容器 | 变量 |
|----------------|-------------|----------|
| **`./config.yaml`** | **`/home/user/.coddy/config.yaml`**(只读) | **`CODDY_CONFIG`** |
| **`./workspace`** | **`/workspace`** | **`CODDY_CWD`** |
| **`./coddy_home`** | **`/home/user/.coddy`** | **`CODDY_HOME`** |

在容器内固定(由 compose **`environment`** 设置):

| 变量 | 值 | 作用 |
|----------|-------|------|
| **`CODDY_CONFIG`** | **`/home/user/.coddy/config.yaml`** | 将加载器指向已挂载的文件；仅镜像本身设置 **`/home/user/.coddy.yaml`**，位于 home 卷之外 |
| **`CODDY_HOME`** | **`/home/user/.coddy`** | 会话、技能、调度器存储 |
| **`CODDY_CWD`** | **`/workspace`** | 新会话和工具路径的默认工作目录 |

提供商密钥可以从主机 shell 注入(可选，未设置时为空):

- **`OPENAI_API_KEY`**、**`ANTHROPIC_API_KEY`**、**`DEEPSEEK_API_KEY`**

优先通过配置或你的编排器挂载密钥；不要提交真实密钥。

### 容器用户

镜像以非特权用户运行 Coddy,uid 和 gid 均为 **1000**，绝不以 root 运行。这是大多数 Linux 桌面的第一个用户，因此 Compose 从当前目录挂载的文件夹对你和容器来说都保持可写。当文件夹属于其他用户时，为 Compose 设置 **`CODDY_UID`** 和 **`CODDY_GID`**(服务会设置 **`user: "${CODDY_UID:-1000}:${CODDY_GID:-1000}"`**)，或将 **`--user`** 传给 **`docker run`**:

```bash
CODDY_UID=$(id -u) CODDY_GID=$(id -g) docker compose up -d
```

此变更之前的镜像以 root 运行，因此它们写入的 home 目录归 root 所有，非特权进程无法写入。先将文件夹交给该用户，然后启动新镜像：

```bash
sudo chown -R 1000:1000 coddy_home workspace
```

镜像检查健康状态的方式与 Compose 相同，即启动二进制(**`HEALTHCHECK`** **`/bin/coddy --version`**):scratch 镜像没有 shell 或 HTTP 客户端来向服务器本身询问。

### Compose 命令

**已发布镜像**(从仓库根目录或你存放 **`config.yaml`**、**`workspace/`**、**`coddy_home/`** 的任意目录):

```bash
docker compose pull
docker compose up -d
docker compose ps
docker compose logs -f coddy
curl -sS http://127.0.0.1:12345/v1/models | head
docker compose restart coddy
docker compose down
```

无需编辑 YAML 即可固定 GHCR 标签：

```bash
export CODDY_IMAGE=ghcr.io/coddy-project/coddy-agent:0.2.0
docker compose pull
docker compose up -d
```

仅更改主机端口：

```bash
export CODDY_HTTP_PORT=8080
docker compose up -d
# UI: http://127.0.0.1:8080/
```

两个文件都只将端口发布到主机的回环地址，因此 API 只能从该机器访问，其他任何地方都无法访问。若没有凭据，任何能访问该端口的人都拥有整个 API - 提供商密钥、工作区、智能体 - 因此在你用 **`CODDY_HTTP_BIND`** 将其开放到网络之前，请先设置 **`httpserver.auth_token`** 或登录([身份验证](../reference/http-api.md#身份验证可选)):

```bash
export CODDY_HTTP_BIND=0.0.0.0
docker compose up -d
```

**从源码构建**(开发 compose 文件):

```bash
docker compose -f docker-compose.dev.yml build coddy
docker compose -f docker-compose.dev.yml up -d --build
docker compose -f docker-compose.dev.yml logs -f coddy
docker compose -f docker-compose.dev.yml down
```

开发文件上的可选构建参数：

```bash
export CODDY_VERSION="$(git describe --tags --dirty 2>/dev/null || echo dev)"
export CODDY_BUILD_TAGS="http,scheduler,ui,memory,gateway"
docker compose -f docker-compose.dev.yml build coddy
```

**`CODDY_BUILD_TAGS`** 必须保持逗号分隔且**不含空格**，与 **`go build -tags=`** 匹配。开发文件默认 **`http,scheduler,ui,memory,gateway`**，以便构建的镜像能运行消息网关；去掉 **`gateway`** 可以精简它。单独的 [`Dockerfile`](../../../Dockerfile) 会构建完整集合(参见 [镜像默认包含的内容](#镜像默认包含的内容))。

### 添加消息网关

机器人不是单独的容器或单独的命令。**`coddy serve`** 运行已挂载的 **`config.yaml`** 所启用的每个子系统，因此启用机器人只需在该文件中添加一行：

```yaml
gateways:
  telegram:
    enable: true
```

```bash
# Build from source: the dev image already includes the `gateway` tag.
export TELEGRAM_BOT_TOKEN="<bot-token>"          # or leave in $CODDY_HOME/.env
docker compose -f docker-compose.dev.yml up -d --build
docker compose -f docker-compose.dev.yml logs -f coddy   # expect: "telegram bot connected"
```

随后机器人和 Web 界面共享同一个会话管理器，因此 Telegram 对话就是一个会话，你可以在 **`http://localhost:12345`** 打开它并在其运行时观察。当你需要不同的标志时，**`CODDY_COMMAND`** 仍会覆盖命令行。

注意事项：

- **已发布的 GHCR 镜像带有 `gateway` 标签**(CI [`docker-build-push.yaml`](../../../.github/workflows/docker-build-push.yaml) 设置 **`BUILD_TAGS=http,scheduler,ui,memory,cli,gateway,swarm,metrics,browser,desktop`**)，因此带 **`CODDY_COMMAND=gateway`** 的 **`docker-compose.yml`** 可在原版镜像上运行；无需自定义 **`CODDY_IMAGE`**。
- 机器人令牌从 **`TELEGRAM_BOT_TOKEN`**(两个 compose 文件都会传递)或 **`$CODDY_HOME/.env`** 读取；请勿将其放入 git。
- 如果你的 **`gateways.telegram.proxy`** 指向主机本地代理(例如 **`socks5://127.0.0.1:7890`**)，则无法从容器内部访问 - 请改用 **`host.docker.internal`**，或在 **`docker-compose.override.yml`** 中添加 **`network_mode: host`**。
- 网关模式不使用入站端口(Telegram 长轮询)；映射的 **`12345`** 完全不被使用。

### 覆盖示例

在 **`docker-compose.yml`** 旁边创建 **`docker-compose.override.yml`**(Compose 会自动加载):

```yaml
services:
  coddy:
    image: ghcr.io/coddy-project/coddy-agent:0.2.0
    ports:
      - "8080:12345"
    environment:
      OPENAI_API_KEY: ${OPENAI_API_KEY}
```

使用覆盖文件来指定机器特定的端口、固定标签或额外的 **`environment`** - 请勿将密钥放入 git。

### 升级与回滚

```bash
export CODDY_IMAGE=ghcr.io/coddy-project/coddy-agent:NEW_TAG
docker compose pull
docker compose up -d
```

主机上的会话和配置(**`./coddy_home`**、已挂载的 **`config.yaml`**)在镜像替换后仍然存在。如果新二进制未通过健康检查，请检查 **`docker compose logs coddy`** 并将 **`CODDY_IMAGE`** 回滚到之前的标签。

## 快速开始(Compose + Web 界面)

从本仓库的检出副本(或你存放 **`config.yaml`**、**`workspace/`** 和 **`coddy_home/`** 的任意文件夹):

**1. 准备配置和目录**

```bash
cp config.example.yaml config.yaml
mkdir -p workspace coddy_home
```

编辑 **`config.yaml`**：在 **`providers`** 和 **`models`** 下至少配置一个条目，并将 **`agent.model`** 设置为列表中某个模型 id。你可以将 **`api_key`** 留空，改为通过 compose **`environment`** 传入 **`OPENAI_API_KEY`**(或提供商 **`name`** 的 **`NAME_API_KEY`**)。

可选：在 YAML 中将 **`httpserver.host`** 设置为 **`0.0.0.0`**，并将 **`httpserver.port`** 设置为 **12345**。容器 **`CMD`** 已运行 **`coddy serve -H 0.0.0.0 -P 12345`**，因此即使文件中省略了 **`httpserver`**，标志也会生效。

**2. 启动 Coddy**(来自 GHCR 的已发布镜像):

```bash
docker compose pull
docker compose up -d
```

**3. 使用内置界面连接**

在主机(与 Docker 同一台机器，除非你转发了端口)的浏览器中打开：

```text
http://127.0.0.1:12345/
```

你将获得：

| URL | 用途 |
|-----|---------|
| **`/`** | 嵌入式 SPA(聊天输入框、会话、对话记录中的工具) |
| **`/#/settings`** | 实时 **`config.yaml`** 编辑器(**`GET/PUT /coddy/config`**) |
| **`/docs/`** | Swagger UI |
| **`/v1/models`** | 模型和模式列表(也被 SPA 的模型选择器使用) |

界面中典型首次使用流程：

1. 确认页面加载(静态资源来自 **`go:embed`**)。
2. 在输入框工具栏中选择一个**后端模型**(除 **agent**、**plan** 和 **ask** 配置文件外，每行都来自你 YAML 中的 **`models`** 列表)。
3. 如需要，在 **agent**、**plan** 和 **ask** 模式之间切换(会话操作配置文件；**ask** 为只读)。
4. 输入消息并发送。界面以 **`stream: true`** 调用 **`POST /v1/responses`**，并显示流式传输的助手输出。
5. 由工具创建或编辑的文件落在已挂载的工作区下(**主机 `./workspace`** → 容器 **`/workspace`**、**`CODDY_CWD`**)。

会话和技能状态持久保存在主机的 **`./coddy_home`** 下(容器内为 **`CODDY_HOME`**)。

**4. 健全性检查(可选)**

```bash
curl -sS http://127.0.0.1:12345/v1/models | head
docker compose logs -f coddy
```

**安全：** 容器运行 **`coddy serve -H 0.0.0.0`**，这正是映射端口可访问的原因；在容器之外，绑定的默认值是回环地址。没有 **`httpserver.auth_token`**(或 **`--auth-token`** / **`CODDY_HTTP_TOKEN`**)时，API 没有身份验证。请像对待任何管理 API 一样对待端口 **12345** - 绑定到 localhost、使用防火墙，或在前面放置带 TLS 和身份验证的反向代理以供远程访问。

对于本地 **`Dockerfile`** 构建，使用 **`docker-compose.dev.yml`** - 参见上文的 [Docker Compose](#docker-compose)。

## 镜像默认包含的内容

**`Dockerfile`** **`ARG BUILD_TAGS`** 默认为 **`http,scheduler,ui,memory,gateway,cli,swarm,metrics,browser,desktop`**(逗号分隔，含义与 **`go build -tags=`** 相同)，与发布二进制携带的集合相同。

- **`http`** - **`coddy serve`** 和 REST 网关(参见 **[docs/reference/http-api.md](../reference/http-api.md)**)。
- **`ui`** - **`/`** 上的嵌入式 SPA(需要 **`http`**)。
- **`scheduler`** - 调度器子系统(**[docs/operate/scheduler.md](../operate/scheduler.md)**)。
- **`memory`** - 长期记忆子智能体和会话记忆 REST(**[external/memory/README.md](../../../external/memory/README.md)**)；通过 **`memory.enable`** 切换运行时行为。
- **`gateway`** - Telegram 和 Pachca 机器人(**[docs/surfaces/gateway.md](../surfaces/gateway.md)**)，各自位于其 **`enable`** 键下。
- **`cli`** - 交互式控制台(**[docs/surfaces/console.md](../surfaces/console.md)**)，用于 **`docker run -it ... coddy`**。
- **`swarm`** - 聚合节点的中继(**[docs/operate/swarm.md](../operate/swarm.md)**)，位于 **`swarm.enable`** 下。
- **`metrics`** - Prometheus 导出器(**[docs/operate/monitoring.md](../operate/monitoring.md)**)，在设置 **`metrics.enable`** 之前关闭。它监听回环地址，这是已发布容器端口无法到达的：要从容器外部抓取它，请将 **`metrics.host`** 设置为 **`0.0.0.0`** 并同时设置 **`metrics.token`**，并发布 **`metrics.port`**(默认 12347)。
- **`desktop`** - **`coddy desktop`**，即作为本机窗口的 Web 界面(**[docs/surfaces/desktop.md](../surfaces/desktop.md)**)。镜像中没有任何东西运行它：容器没有显示器，该命令也找不到可打开的浏览器。之所以内置它，是为了让二进制无论来自何处都相同。
- **`browser`** - [Coddy for Browser](../features/browser.md) 扩展在 HTTP 服务器上的端点以及 `browser` 工具，在设置带 **`browser.token`** 的 **`browser.enable`** 之前关闭。扩展连接到已发布的 HTTP 端口，即 Web 界面的地址；位于容器所在机器之外的浏览器需要在其前面放置 HTTPS，因为扩展仅对同一机器上的服务器使用纯 HTTP。

要构建**不含**记忆或嵌入式界面的镜像，可通过 **`docker compose` `args`** 或 **`docker build --build-arg`** 覆盖 **`BUILD_TAGS`**(例如 **`http,scheduler,ui`** 或 **`http,scheduler`**)。

Compose 的卷和环境变量详细信息见 [Docker Compose](#docker-compose)。在没有 **`CODDY_CONFIG`** 的裸机安装中，加载器优先使用 **`$CODDY_HOME/config.yaml`**(参见 **`docs/getting-started/configuration.md`**)。

## Dockerfile 各阶段如何工作

1. **`ui-builder`(Node)** - 在 **`external/ui`** 下运行 **`npm ci`** 和 **`npm run build:go`**，生成静态产物，当 **`BUILD_TAGS`** 中包含 **`ui`** 时，将其复制到 Go 树中以供 **`go:embed`** 使用。它在构建平台上运行(**`FROM --platform=$BUILDPLATFORM`**)：产物对每个平台都相同，因此多架构构建只需原生构建一次。
2. **`build`(Go)** - 同样在构建平台上运行并进行交叉编译：**`CGO_ENABLED=0`**、来自 BuildKit **`TARGETOS`/`TARGETARCH`** 的 **`GOOS`/`GOARCH`**(CI 构建 **`linux/amd64`** 和 **`linux/arm64`**)、带 **`-trimpath`** 和 **`-ldflags "-s -w -X ...Version=..."`** 的 **`go build -tags="$BUILD_TAGS"`**，写入 **`/out/coddy`**，为 HTTPS 客户端复制 **`ca-certificates.crt`**。**`TARGETOS`** 和 **`TARGETARCH`** 有意不声明默认值：默认值会替换 BuildKit 传入的值，而 **`linux/arm64`** 镜像就曾以这种方式发布了 **x86-64** 二进制([issue #482](https://github.com/coddy-project/coddy-agent/issues/482))。
3. **`scratch`** - 仅包含二进制、CA 证书包和用户写入的文件夹(**`/home/user`**、**`/workspace`**、**`/tmp`**，归 **1000:1000** 所有);**`USER 1000:1000`**、**`HEALTHCHECK`** **`/bin/coddy --version`**、**`ENTRYPOINT`** **`/bin/coddy`**、默认 **`CMD`** **`serve -H 0.0.0.0 -P 12345`**([容器用户](#容器用户))。市场同步和插件安装不需要 `/tmp` 或 Git 可执行文件：远程暂存保存在可写的 `${CODDY_HOME}/tmp` 目录下，Git 源则回退到 go-git。HTTPS 和 `file://` 源在该路径下可用；SSH 源需要 go-git 在容器中可用的身份验证机制。

## 检查镜像的平台

Docker 会为多架构镜像的每个变体标注它所请求的平台，无论其中的二进制是什么，而且带有 **`qemu-user-static`** 的主机甚至会运行架构错误的二进制。只有二进制自身的 ELF 头才能说明它面向哪个平台，而检查读取的正是这一点：

```bash
make check-image
make check-image IMAGE_PLATFORMS=linux/arm64
```

**`make check-image`**([`scripts/check-image.sh`](../../../scripts/check-image.sh))为 **`IMAGE_PLATFORMS`** 中的每个平台构建镜像(默认 **`linux/amd64,linux/arm64`**，即发布所推送的)，一次一个平台，将每个文件系统导出到 **`dist/image/<os>_<arch>/`**，除非 **`/bin/coddy`** 处是该平台的二进制，否则失败。主机自身平台的二进制也会被运行，并必须打印其构建时的版本。它只需要带 **`buildx`** 的 Docker，别无其他：构建阶段原生运行并交叉编译，因此 **amd64** 主机无需模拟即可构建 **arm64** 变体，普通的 **`docker`** 驱动即可胜任。每个拉取请求的 **Docker 镜像**任务都会运行同一脚本，发布工作流在推送前也会运行它。

不使用 Docker 时，**`make test`** 对 Dockerfile 本身持有相同的功能：[`features/docker_image_platforms.feature`](../../../features/docker_image_platforms.feature) 按 BuildKit 解析其平台参数的方式读取它，并在 Go 阶段将为与正在构建的平台不同的平台编译，或将在模拟下运行时失败。

## 自动化冒烟测试

```bash
./examples/httpserver/docker.sh
```

该脚本构建一个临时 **`config.yaml`**，用 **`docker-compose.dev.yml`** 启动 **`coddy`**，等待 **`/v1/models`**，然后运行 **`examples/httpserver/http_smoke_gateway.py`**。

<!-- docsgen:source sha256=5f10a4159a86ce6d -->

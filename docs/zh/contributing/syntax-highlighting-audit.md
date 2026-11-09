# NeuralDeep 语法高亮审计

## 方法

于 2026-09-10 针对 NeuralDeep（`qwen3.8-27b-noreason`）运行。为 63 个语言名称各请求一个简短的围栏代码示例。未强制指定语言标签：下方标签即模型返回的内容。最初的 HTTP 429 响应通过带暂停的顺序重试处理；全部 63 个初始示例最终都收到了。另一次 URQ 澄清返回了 `UNKNOWN`。

响应在离线测试中通过真实的 React Markdown 渲染器重放。测试检查保留的源文本、受支持标签的高亮 token 跨度，以及不受支持标签的安全纯文本。生成的程序未被执行或编译，因此这是一次渲染 / 语言标签审计，而非语义正确性认证。fixtures 或本报告中不存储任何凭据。

## 结果

- 在此次扩展之前（PR 已包含 Vue/PostCSS）：63 个响应块中有 30 个包含语法跨度。
- 之后：63 个中有 60 个包含语法跨度；其余三个仍为可读的纯文本。
- 专用额外语法：GDScript、HLSL、WGSL、COBOL、T-SQL、VBA，另加 20 个额外捆绑语法。
- PL/SQL、OpenCL 和 CUDA 使用 SQL、C 和 C++ 基础语法；方言特有的构造可能保持未着色。
- NeuralDeep 将 Transact-SQL 示例标记为 `sql`；该响应正确使用了通用 SQL。显式的 `tsql`、`transactsql`、`transact-sql` 和 `t-sql` 标签会选择专用语法。
- Prolog 响应仅包含不带引号的原子和注释。捆绑的 Prolog 语法会为其注释着色，但将这些原子保持为纯文本；变量、字符串和数字另通过确定性测试单独检查。
- UnrealScript 和 TADS 没有配置语法。URQ 被误认为 GraphQL 查询；在指定 Universal RipSoft Quest / UrqW 后，模型返回了 `UNKNOWN`。具有误导性的初始 URQ 响应被保留为纯文本回归 fixture，并不能作为 URQ 语言支持的证据。[UrqW 项目及文档](https://urqw.github.io/UrqW/)。

## 各语言观察

| 请求的语言 | 返回的围栏标签 | 之前：token 跨度 | 之后：token 跨度 | 备注 |
| --- | --- | ---: | ---: | --- |
| React (JSX) | `jsx` | 9 | 9 | 语法已启用 |
| Java | `java` | 16 | 16 | 语法已启用 |
| TypeScript | `typescript` | 20 | 20 | 语法已启用 |
| PHP | `php` | 14 | 14 | 语法已启用 |
| Python | `python` | 9 | 9 | 语法已启用 |
| C++ | `cpp` | 15 | 15 | 语法已启用 |
| x86 汇编（Intel 语法） | `asm` | 0 | 35 | 语法已启用 |
| C# | `csharp` | 7 | 7 | 语法已启用 |
| Go | `go` | 9 | 9 | 语法已启用 |
| Rust | `rust` | 11 | 11 | 语法已启用 |
| C | `c` | 15 | 15 | 语法已启用 |
| XHTML | `xhtml` | 23 | 23 | 语法已启用 |
| XML | `xml` | 22 | 22 | 语法已启用 |
| XSLT | `xslt` | 0 | 29 | 语法已启用 |
| XSL | `xml` | 19 | 19 | 语法已启用 |
| Pascal | `pascal` | 0 | 14 | 语法已启用 |
| Object Pascal | `pascal` | 0 | 15 | 语法已启用 |
| GameMaker Language (GML) | `gml` | 0 | 9 | 语法已启用 |
| Common Lisp | `lisp` | 0 | 9 | 语法已启用 |
| Prolog | `prolog` | 0 | 2 | 此示例中仅注释着色；原子保持纯文本 |
| Delphi | `delphi` | 0 | 12 | 语法已启用 |
| Swift | `swift` | 15 | 15 | 语法已启用 |
| Dart | `dart` | 0 | 12 | 语法已启用 |
| Ruby | `ruby` | 6 | 6 | 语法已启用 |
| Kotlin | `kotlin` | 11 | 11 | 语法已启用 |
| Ada | `ada` | 0 | 13 | 语法已启用 |
| Objective-C | `objectivec` | 7 | 7 | 语法已启用 |
| VBA | `vba` | 0 | 13 | 语法已启用 |
| MATLAB | `matlab` | 0 | 8 | 语法已启用 |
| Elixir | `elixir` | 0 | 13 | 语法已启用 |
| Scala | `scala` | 0 | 12 | 语法已启用 |
| SQL | `sql` | 13 | 13 | 语法已启用 |
| Perl | `perl` | 12 | 12 | 语法已启用 |
| Fortran | `fortran` | 0 | 16 | 语法已启用 |
| COBOL | `cobol` | 0 | 16 | 语法已启用 |
| Scheme | `scheme` | 0 | 10 | 语法已启用 |
| Tcl | `tcl` | 0 | 11 | 语法已启用 |
| F# | `fsharp` | 0 | 9 | 语法已启用 |
| OCaml | `ocaml` | 0 | 13 | 语法已启用 |
| Visual Basic .NET | `vbnet` | 13 | 13 | 语法已启用 |
| Haskell | `haskell` | 0 | 10 | 语法已启用 |
| SVG | `svg` | 21 | 21 | 语法已启用 |
| GDScript (Godot) | `gdscript` | 0 | 10 | 语法已启用 |
| Lua | `lua` | 12 | 12 | 语法已启用 |
| Bash | `bash` | 11 | 11 | 语法已启用 |
| POSIX sh | `sh` | 9 | 9 | 语法已启用 |
| Zsh | `zsh` | 9 | 9 | 语法已启用 |
| PowerShell | `powershell` | 0 | 17 | 语法已启用 |
| R | `r` | 15 | 15 | 语法已启用 |
| Haxe | `haxe` | 0 | 14 | 语法已启用 |
| Oracle PL/SQL | `plsql` | 0 | 8 | 仅基础语言高亮 |
| Transact-SQL | `sql` | 15 | 15 | 模型返回了 sql；显式 tsql 也受支持 |
| UnrealScript | `unrealscript` | 0 | 0 | 纯文本：无语法 |
| GLSL | `glsl` | 0 | 12 | 语法已启用 |
| HLSL | `hlsl` | 0 | 9 | 语法已启用 |
| OpenCL C | `opencl` | 0 | 13 | 仅基础语言高亮 |
| CUDA C++ | `cuda` | 0 | 12 | 仅基础语言高亮 |
| TADS 3 | `tads` | 0 | 0 | 纯文本：无语法 |
| URQ | `urql` | 0 | 0 | 模型返回了错误语言（GraphQL）；澄清结果为 UNKNOWN |
| WGSL | `wgsl` | 0 | 18 | 语法已启用 |
| Vue 单文件组件 | `vue` | 20 | 20 | HTML/XML 加普通内嵌 JS/CSS |
| PostCSS | `postcss` | 6 | 6 | 语法已启用 |
| React TypeScript (TSX) | `tsx` | 10 | 10 | 语法已启用 |

## 回归 fixture 与维护

- `external/ui/src/ui/markdown/neuraldeepLanguageFixtures.json`：63 个原始响应，包括语言标签和预期的回退行为。
- `external/ui/src/ui/markdown/syntaxLanguages.test.tsx`：离线重放以及常见替代标签检查。无需账户或网络。
- `external/ui/src/ui/markdown/syntaxLanguages.ts`：单一显式语法注册表和别名映射，由聊天的代码块以及 Files 和 edits 窗口共享。自动检测仍处于禁用状态。
- `external/ui/src/ui/markdown/grammars/README.md`：六个内置语法的上游修订和许可证。

采用这一广泛语法集后，生产 JS bundle 从约 308.64 kB 增至 419.16 kB gzip（+110.52 kB，包含分发的许可证声明）。语法在本地捆绑；渲染不会从 CDN 获取语法代码。用户未要求每种语言的截图，因此审计以测试和此表格形式记录。

## 验证

- UI 测试套件：158 个文件中共 1,169 项测试通过。
- 实际浏览器渲染：全部 63 个响应均保留源文本；60 个包含着色 token 跨度，三个保持纯文本。十四种主题/视口组合（七种主题、两种宽度）均无页面水平溢出。
- 生产 Vite 构建和嵌入资源同步通过；全部六个第三方许可证声明均保留在输出的 JavaScript 中。
- 仓库范围的 TypeScript 检查报告了既有的无关错误，包括未更改的 Markdown 组件回调/组件类型定义；新的注册表、fixtures 和语法声明未报告类型错误。

- 定向 HTTP/UI 嵌入和 BDD 测试通过，`http,ui` Go 二进制构建成功。
- `make test` 被重新运行，并在未更改的 Windows 失败上停止：配置日志路径、对话记录导出路径、集群文件权限以及配置暂存/回滚。后续标签组合未达到。

## 截图

从构建的 `coddy serve` 及嵌入的 Web 界面，在 1280 x 900 下于发布 1.3.0 捕获：一个聊天轮次由脚本化模型以固定的 JavaScript、CSS、HTML 和 JSON 回复作答，因此未使用实时提供商或用户对话。全部七种主题也在 390 x 900 下检查：页眉、对话记录列和输入框边缘在两种宽度下均对齐，页面没有水平溢出。

| 主题 | 之后 |
| --- | --- |
| dark | ![dark](../../assets/syntax-highlighting/after-dark-1280.png) |
| light | ![light](../../assets/syntax-highlighting/after-light-1280.png) |
| midnight | ![midnight](../../assets/syntax-highlighting/after-midnight-1280.png) |
| solarized-dark | ![solarized-dark](../../assets/syntax-highlighting/after-solarized-dark-1280.png) |
| monokai | ![monokai](../../assets/syntax-highlighting/after-monokai-1280.png) |
| nord | ![nord](../../assets/syntax-highlighting/after-nord-1280.png) |
| rose-pine | ![rose-pine](../../assets/syntax-highlighting/after-rose-pine-1280.png) |

<!-- docsgen:source sha256=16d23e98b2fdf4c0 -->

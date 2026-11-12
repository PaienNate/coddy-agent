# RPA workflow gates implementation plan

> **面向 AI 代理的工作者：** 必需子技能：使用 subagent-driven-development（推荐）或 executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 给现有的 `/rpa-feat`、`/rpa-bugfix` 补上产物与门禁：`sdd` 工具（workspace/brief/start/done/review）、`.superpowers/sdd/` 账本、`docs/superpowers/{specs,plans}` 落盘、每任务审查、三个收尾选项，以及按技能声明自动路由。

**架构：** 新包 `internal/sdd` 承担全部文件与 git 行为（无 agent 依赖、可单测）；`internal/tools/sdd.go` 是模型调用的工具壳；`internal/agent/react.go` 的权限链加一个分支，只让 `done` 走审批；技能正文改为调用工具并把每一步写成"停"；frontmatter 的 `routes:` 驱动系统提示词里的路由行。

**技术栈：** Go，`tooling.Tool`/`Registry`，`gitws`，`internal/skills` 的前置元数据解析，embedded bundled skills，godog 只用于已有的功能规格（本计划不新增 BDD）。

**规格：** `docs/plans/rpa-workflow-gates.md`（实现者两份都读）

## 全局约束

- 产物路径与格式逐字兼容：`docs/superpowers/specs/`、`docs/superpowers/plans/`、`.superpowers/sdd/<plan>/`、账本表头 `# SDD ledger — plan: <plan>`、完成行 `Task <n>: complete (commits <base7>..<head7>, tests: <cmd> → <last line>)`、审查包三段。
- 不新增 CLI 子命令，不新增斜杠命令；打包面（man、completions、`topLevelCommands`）不动。
- 不改 `rpa-init`、`rpa-gen-rules`；不引入上游技能名或正文。
- 技能正文、代码注释、文档正文用英文；`docs/ru`、`docs/zh` 同 PR 更新并 `make docs-stamp`。
- `done` 只有在测试命令退出码为 0 时才写账本；失败时账本一字不改。
- 提交信息不带任何模型/代理署名。

## 审查重点（Review Focus）

- 计划文件名相同而目录不同，两个计划不能共用一个工作区 —— `TestWorkspaceDisambiguatesSameBasename`.
- 围栏代码块里的 `### Task 3：` 不能被当成任务标题 —— `TestBriefIgnoresHeadingsInsideFences`.
- 测试命令失败时账本不变，且工具返回失败 —— `TestDoneRecordsNothingWhenTheCommandFails`.
- `sdd` 的 `review` 遇到空范围或 base 不是 head 祖先时拒绝写文件 —— `TestReviewRefusesABogusRange`.
- 权限：`done` 在 ask 模式下要审批，其他动作永不审批；bypass 模式永不审批 —— `TestSDDPermissionOnlyForDone`, `TestSDDPermissionBypassNeverAsks`.
- 没有技能声明 `routes:` 时，系统提示词保持原句，不出现空的"load the matching skill" —— `TestSkillsPromptKeepsTheGenericHintWithoutRoutes`.

---

### Task 1: `internal/sdd` 的 workspace 与 brief

**文件：**
- 创建：`internal/sdd/workspace.go`、`internal/sdd/brief.go`、`internal/sdd/workspace_test.go`、`internal/sdd/brief_test.go`
- 修改：`internal/gitws/gitws.go`（导出 `Run`）

- [ ] **步骤 1：写失败测试**

```go
func TestWorkspaceCreatesThePlanDirectory(t *testing.T)
// temp git repo (git init + one commit)，plan 在 repo/docs/plans/x.md
// 期望：返回 <root>/.superpowers/sdd/x，目录存在，plan-path 内容是 docs/plans/x.md
//       .superpowers/sdd/.gitignore 内容是 "*"

func TestWorkspaceDisambiguatesSameBasename(t *testing.T)
// docs/alpha/x.md 与 docs/beta/x.md 两次调用，期望第二个是 x-beta，互不覆盖

func TestWorkspaceReusesItsOwnDirectory(t *testing.T)
// 同一 plan 调两次返回同一路径，plan-path 不变

func TestBriefSlicesOneTask(t *testing.T)
// 计划含 "### Task 1: A" / "### Task 2: B"，Brief(plan, 2) 只含 B 那一节

func TestBriefReadsChineseHeadings(t *testing.T)
// "### 任务 2：B" 能被切出

func TestBriefIgnoresHeadingsInsideFences(t *testing.T)
// 任务 1 的代码块里有 "### Task 2: not-a-task"，Brief(plan, 1) 仍包含它、Brief(plan, 2) 返回 ErrTaskNotFound

func TestBriefRejectsAMissingTask(t *testing.T)
// n=9 → errors.Is(err, ErrTaskNotFound)
```

- [ ] **步骤 2：运行测试确认失败**

运行：`go test ./internal/sdd -run 'TestWorkspace|TestBrief' -v`
预期：FAIL，包不存在或 `Workspace`/`Brief` 未定义。

- [ ] **步骤 3：实现**

`internal/gitws/gitws.go` 加：

```go
// Run executes git in dir and returns its trimmed stdout.
func Run(dir string, args ...string) (string, error) { return runGit(dir, args...) }
```

`internal/sdd/workspace.go`：

```go
func Workspace(repoRoot, planArg string) (string, error)
```

规则与脚本一致：`slug = basename(planArg) 去掉 .md`；`planID` = plan 的物理绝对路径相对 repoRoot 的形式（不在 repo 下则用绝对路径）；目录 `repoRoot/.superpowers/sdd/<slug>`，若 `plan-path` 存在且不等于 planID，依次试 `<slug>-<parentDir>`、`<slug>-<parentDir>-2..`；写 `plan-path`；写 `repoRoot/.superpowers/sdd/.gitignore`（内容 `*`）；返回绝对路径。

`internal/sdd/brief.go`：

```go
var ErrTaskNotFound = errors.New("task not found in the plan")

func Brief(planArg string, n int) (string, error)
func WriteBrief(dir, planArg string, n int) (path string, lines int, err error)
```

切分规则：逐行扫描，` ``` ` 翻转围栏状态；围栏外遇到 `^#+\s+(Task|任务)\s*<k>(\D|$)` 时，`k == n` 开始收集、否则停止收集（与现有脚本的 `intask` 语义一致）。`WriteBrief` 写到 `dir/task-<n>-brief.md`，返回行数。

- [ ] **步骤 4：运行测试确认通过**

运行：`go test ./internal/sdd -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/gitws/gitws.go internal/sdd/
git commit -m "feat(sdd): resolve the plan workspace and slice task briefs"
```

---

### Task 2: 账本与审查包

**文件：**
- 创建：`internal/sdd/ledger.go`、`internal/sdd/review.go`、`internal/sdd/ledger_test.go`、`internal/sdd/review_test.go`

- [ ] **步骤 1：写失败测试**

```go
func TestCompleteCreatesTheLedgerWithItsHeader(t *testing.T)
// ledger 不存在 → Complete 之后第一行是 "# SDD ledger — plan: docs/plans/x.md"

func TestCompleteAppendsTheExactLine(t *testing.T)
// 固定 base/head/command/lastLine，断言追加行为
// "Task 2: complete (commits <base7>..<head7>, tests: go test ./... → ok example) "

func TestCompleteKeepsEarlierLines(t *testing.T)
// 已有 Task 1 行，再 Complete 2，两行都在且顺序不变

func TestReviewPackageWritesTheThreeSections(t *testing.T)
// temp repo 两个 commit，断言文件含 "# Review package: <base>..<head>"、
// "## Commits"、"## Files changed"、"## Diff"，文件名 review-<base7>..<head7>.diff

func TestReviewRefusesABogusRange(t *testing.T)
// base 不是 head 祖先、以及空范围，两种都返回 ErrBadRange，且不写文件
```

- [ ] **步骤 2：运行测试确认失败**

运行：`go test ./internal/sdd -run 'TestComplete|TestReview' -v`
预期：FAIL。

- [ ] **步骤 3：实现**

```go
func Complete(repoRoot, dir, planArg string, n int, base, command, lastLine string) (line string, err error)
func ReviewPackage(repoRoot, dir, base, head string) (path string, commits int, size int64, err error)

var ErrBadRange = errors.New("review range is empty or not rooted at base")
```

`Complete`：用 `gitws.Run(repoRoot, "rev-parse", "--short=7", base)` 与 `... "HEAD"` 取短 sha；ledger 不存在时先写表头；追加一行；返回整行。
`ReviewPackage`：先用 `MergeBase` 语义校验（`git merge-base --is-ancestor base head` 与 `git rev-list --count base..head`），任一失败返回 `ErrBadRange` 且不创建文件；否则组装三段写文件。

- [ ] **步骤 4：运行测试确认通过**

运行：`go test ./internal/sdd -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/sdd/
git commit -m "feat(sdd): write the ledger and review packages"
```

---

### Task 3: `sdd` 工具

**文件：**
- 创建：`internal/tools/sdd.go`、`internal/tools/sdd_test.go`
- 修改：`internal/tools/export.go`（注册）、`internal/agent/react.go`（权限分支）、`internal/agent/decisions.go`（`ExtractSDDCommand` 或复用）
- 测试：`internal/agent/sdd_permission_test.go`（或并入既有 permission 测试文件）

- [ ] **步骤 1：写失败测试**

```go
func TestSDDToolSchemaHasTheFiveActions(t *testing.T)
// Definition.Name == "sdd"；action 枚举恰好 {workspace,brief,start,done,review}

func TestSDDWorkspaceAndStartRunAgainstATempRepo(t *testing.T)
// env.CWD 指向 temp repo；workspace 返回目录；start 返回 "brief:" 与 "base:"

func TestSDDDoneRecordsNothingWhenTheCommandFails(t *testing.T)
// command = "go test ./..." 用一个必失败的包 → 账本不存在/不变，工具返回 error
// 再用一个必成功的包 → 账本出现 Task 1: complete

func TestSDDPermissionOnlyForDone(t *testing.T)
// 权限链：action=start → requiresPerm=false；action=done 且命令未授权 → true；
// 命令已在 sessCmdGrants 里 → false

func TestSDDPermissionBypassNeverAsks(t *testing.T)
```

- [ ] **步骤 2：运行测试确认失败**

运行：`go test ./internal/tools ./internal/agent -run 'TestSDD' -v`
预期：FAIL，`SDDTool` 未定义。

- [ ] **步骤 3：实现**

`internal/tools/sdd.go`：

```go
const SDDToolName = "sdd"

func SDDTool() *tooling.Tool
```

Schema：`action`（enum，必填）、`plan`（string，必填）、`task`（integer）、`base`（string）、`command`（string）、`head`（string）。执行步骤：

1. `repoRoot, err := gitws.Run(env.CWD, "rev-parse", "--show-toplevel")`；失败则报"not a git repository"。
2. `dir, err := sdd.Workspace(repoRoot, plan)`。
3. 按 action 分派；`done` 用 `platform.CurrentShell().Command(command)` 执行，输出写 `dir/task-<n>-tests.log`，打印末 5 行；`rc != 0` 时返回 error 带上退出码，不写账本。

描述文字要点（会被模型读到）：五个动作各一句；`done` 强调"只有测试命令通过才记账"。

`internal/tools/export.go` 在 `skills.auto_discovery` 分支里注册；`internal/agent/decisions.go` 加：

```go
// sddCommandAndAction reads the action and command of an sdd call.
func sddCommandAndAction(argsJSON string) (action, command string, ok bool)
```

`internal/agent/react.go` 权限链加分支（放在 `run_command` 分支之后）：

```go
} else if tc.Name == tools.SDDToolName {
    action, cmd, ok := sddCommandAndAction(tc.InputJSON)
    requiresPerm = ok && action == "done"
    if requiresPerm && env.PermissionMode == config.PermModeBypass {
        requiresPerm = false
    } else if requiresPerm && permission.CommandAllowedWithSession(env, sessCmdGrants, cmd) {
        requiresPerm = false
    }
}
```

- [ ] **步骤 4：运行测试确认通过**

运行：`go test ./internal/tools ./internal/agent -run 'TestSDD' -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/tools/sdd.go internal/tools/sdd_test.go internal/tools/export.go internal/agent/decisions.go internal/agent/react.go internal/agent/sdd_permission_test.go
git commit -m "feat(tools): add the sdd workflow tool"
```

---

### Task 4: `routes:` 与自动路由

**文件：**
- 修改：`internal/skills/loader.go`（`Skill.Routes`、frontmatter）、`internal/skills/bundled.go`（解析）、`internal/agent/system_prompt.go`（路由行）
- 测试：`internal/skills/loader_test.go`、`internal/agent/system_prompt_test.go`（没有就创建）

- [ ] **步骤 1：写失败测试**

```go
func TestSkillReadsItsRoutesLine(t *testing.T)
// frontmatter "routes: the user asks for a new feature" 出现在 Skill.Routes

func TestSkillsPromptNamesTheDeclaredRoutes(t *testing.T)
// 两个技能带 routes，输出含 "load the flow skill that matches" 与两个名字

func TestSkillsPromptKeepsTheGenericHintWithoutRoutes(t *testing.T)
// 都不带 routes → 输出等于今天那句 loadSkillHint，且不含 "flow skill"
```

- [ ] **步骤 2：运行测试确认失败**

运行：`go test ./internal/skills ./internal/agent -run 'TestSkill.*Routes|TestSkillsPrompt' -v`
预期：FAIL。

- [ ] **步骤 3：实现**

`frontmatter` 加 `Routes string \`yaml:"routes"\``；`Skill` 加 `Routes string`；`loadFile`/`parseSkillBytes` 两处填值。
`internal/agent/system_prompt.go`：`buildSkillsPromptMarkdown` 里，若有技能声明 Routes，则用

```
Before you create, change, debug or review code, load the flow skill that matches and follow it: `rpa-feat` - <routes>; `rpa-bugfix` - <routes>.
```

替代 `loadSkillHint`（按目录顺序稳定排序）；否则保持原句。行只随规则代际变化，不逐轮变动。

- [ ] **步骤 4：运行测试确认通过**

运行：`go test ./internal/skills ./internal/agent -run 'TestSkill.*Routes|TestSkillsPrompt' -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/skills/loader.go internal/skills/bundled.go internal/skills/loader_test.go internal/agent/system_prompt.go internal/agent/system_prompt_test.go
git commit -m "feat(skills): route to the flow skill from its frontmatter"
```

---

### Task 5: 重写两个技能

**文件：**
- 修改：`internal/skills/bundled/rpa-feat/SKILL.md`（2.0.0 → 2.1.0）、`internal/skills/bundled/rpa-bugfix/SKILL.md`（2.0.0 → 2.1.0）
- 创建：`internal/skills/bundled/rpa-feat/reviewer-prompt.md`
- 测试：`internal/skills/bundled_test.go`

- [ ] **步骤 1：写失败测试**

```go
func TestTheFlowSkillsDeclareTheirGatesAndRoutes(t *testing.T)
// 两个技能：Version == "2.1.0"；Routes 非空；正文含 "sdd start"、"sdd done"、
// "sdd review"、".superpowers/sdd/"、"docs/superpowers/plans/"
// rpa-feat 另含 "docs/superpowers/specs/"
```

- [ ] **步骤 2：运行测试确认失败**

运行：`go test ./internal/skills -run TestTheFlowSkillsDeclareTheirGatesAndRoutes -v`
预期：FAIL（版本仍是 2.0.0，正文没有 sdd）。

- [ ] **步骤 3：改技能正文**

`rpa-feat`：保留十步骨架与既有措辞，插入五个门（spec → 批准；plan + Review Focus → 批准；每任务 `sdd start`/`sdd done`；每任务 `sdd review` + 审查者；收尾三选项），每处都写成"停下，等用户回答"。frontmatter 加：

```yaml
routes: the user asks for a new feature, a behavior change, or a refactor
```

`rpa-bugfix`：保留"先有可信根因、先有复现测试"，加同一个账本与 G3/G4/G5；一任务的计划文件作为锚点；`routes: the user reports a bug, a regression, or unexpected behavior`。

`reviewer-prompt.md`：自写英文审查 brief（读审查包、按 file:line 报 Critical/Important/Minor、列出"不予判断"的项、给明确结论）。

- [ ] **步骤 4：运行测试确认通过**

运行：`go test ./internal/skills -v`
预期：PASS（既有 bundled 测试也全绿）。

- [ ] **步骤 5：Commit**

```bash
git add internal/skills/bundled/rpa-feat internal/skills/bundled/rpa-bugfix
git commit -m "feat(skills): gate the rpa flows on artifacts and approvals"
```

---

### Task 6: 文档

**文件：**
- 修改：`docs/features/skills.md`、`docs/ru/features/skills.md`、`docs/zh/features/skills.md`

- [ ] **步骤 1：写英文段**

- 交付表里 `/rpa-feat`、`/rpa-bugfix` 两行补上"门"与产物；
- 新节 **Artifacts and gates**：树状图、账本行示例、`sdd` 五个动作一段、"替换老副本"的说明；
- 一句"没有 `sdd` 工具也能跑，只是回到无账本的手工流程"，作为回滚路径。

- [ ] **步骤 2：俄文、中文同步并盖戳**

```bash
make docs-stamp PAGES="docs/ru/features/skills.md docs/zh/features/skills.md"
```

- [ ] **步骤 3：生成与检查**

```bash
make docs
make docs-check
go test ./internal/config ./internal/skills ./internal/tools ./internal/sdd ./internal/agent
```

预期：与本次无关的既有 docs 红灯之外无新增问题；Go 全绿。

- [ ] **步骤 4：Commit**

```bash
git add docs/features/skills.md docs/ru/features/skills.md docs/zh/features/skills.md docs/reference/config.md
git commit -m "docs(features): describe the rpa artifacts and gates"
```

---

## 验收

- `go test ./internal/sdd ./internal/tools ./internal/skills ./internal/agent` 全绿。
- 在一个临时仓库里手工走一遍：`sdd workspace` → `start` → 故意让测试失败（账本无新行）→ 修好再 `done`（账本出现完整行）→ `review` 生成包含三段 diff 的文件。
- 老 home 里的 `rpa-feat`（2.0.0）在启动后被替换为 2.1.0；`skills-delivery.json` 记下新版本。
- 不带 `routes:` 的技能不改变系统提示词；两个流程技能在场时出现路由行。

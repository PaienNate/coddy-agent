# Compaction landing implementation plan

> **面向 AI 代理的工作者：** 必需子技能：使用 subagent-driven-development（推荐）或 executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 让压缩留下的尾巴和结果驱逐的起点由绝对 token 数决定，而不是由模型窗口的百分比决定，使 1M 窗口既能当 1M 用、又不会把 500k 尾巴重复付费。

**架构：** 三个数值落点各自加一个绝对上限，与既有百分比取更小者：`compaction.keep_tail_tokens`（压缩保留的尾巴）、回合内折点（改由 `requestOverhead() + keep_tail_tokens` 封顶）、`compaction.result_eviction.start_tokens`（驱逐门限，并且门限只量对话、不量请求开销）。触发阈值 `threshold_percent` 不动。

**技术栈：** Go，`internal/config` 的 YAML + 内嵌 JSON Schema，`internal/agent` 的压缩与结果驱逐，godog BDD 场景，`make docs` 生成字段表。

**规格：** `docs/plans/compaction-landing.md`（本计划实现的规格；执行者两份都读）

## 全局约束

- 不改 `compaction.threshold_percent` 的语义：它仍是真实窗口的百分比，1M 窗口就按 1M 触发。
- 新键全部可回滚：`keep_tail_tokens: 0` 和 `start_tokens: 0` 精确恢复旧算术，不需要回退代码。
- 持久化的 transcript 永不重写：折叠只插入摘要行，驱逐只改发给模型的投影。
- 保留的尾巴最少是 1 轮（自动触发）或 0 轮（`/compact` + force）；即使超过预算也不折掉正在回答的 prompt。
- `keep_recent_turns: 0` 仍然表示"全折"，显式 0 优先于 `keep_tail_tokens`。
- 代码注释与文档用英文；`docs/ru/` 的同一页在同一 PR 更新，并用 `make docs-stamp PAGES=...` 重新盖戳。
- 提交信息不带任何模型或代理署名。

## 审查重点（Review Focus）

- 尾巴超预算但只剩最后一轮时，仍保留那一轮而不是折掉正在回答的 prompt —— `TestRegularCompactionKeepsThePromptWhenItAloneExceedsTheTailBudget`。
- `keep_recent_turns: 0` 与 `keep_tail_tokens: 15000` 同时出现时，0 胜出（手动全折不被 resurrect）—— `TestKeepRecentTurnsZeroWinsOverTheTailBudget`。
- 请求开销（rules 120k）大于目标时，回合内折叠仍保留最新一步并且不反复折 —— `TestInTurnTargetBelowTheRequestOverheadKeepsTheFloor`。
- 驱逐门限不再被 rules 触发（否则每条结果都被折叠、缓存每步重写）—— `TestEvictionGateIgnoresTheRequestOverhead`。
- 窗口无限大或未声明时算术不溢出、不除零 —— `TestEvictionGateWithoutAWindow` 与 `TestInTurnTargetWithoutAWindow`。

---

### 任务 1：配置面（两个新键）

**文件：**
- 修改：`internal/config/compaction.go`
- 修改：`internal/config/config.schema.json`（`compaction` 与 `compaction.result_eviction` 两处 properties）
- 修改：`internal/config/ui_schema.go`（`compaction` 段落的 `intProp` 与键顺序列表）
- 测试：`internal/config/compaction_test.go`

- [ ] **步骤 1：编写失败的测试**

在 `internal/config/compaction_test.go` 追加：

```go
func TestCompactionKeepTailTokens(t *testing.T) {
	// unset -> 15000
	// explicit 0 -> 0 (disabled)
	// explicit 40000 -> 40000
	// negative -> Validate() error naming compaction.keep_tail_tokens
}

func TestResultEvictionStartTokens(t *testing.T) {
	// unset -> 60000
	// explicit 0 -> 0 (disabled)
	// explicit 120000 -> 120000
	// negative -> Validate() error naming compaction.result_eviction.start_tokens
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/config -run 'TestCompactionKeepTailTokens|TestResultEvictionStartTokens' -v`
预期：FAIL，`EffectiveKeepTailTokens` / `EffectiveStartTokens` 未定义。

- [ ] **步骤 3：实现**

`internal/config/compaction.go`：

```go
const CompactionDefaultKeepTailTokens = 15000
const ResultEvictionDefaultStartTokens = 60000

// Compaction
KeepTailTokens *int `yaml:"keep_tail_tokens"`   // nil = default, 0 = no absolute cap

// ResultEviction
StartTokens *int `yaml:"start_tokens"`          // nil = default, 0 = percent only
```

```go
func (c *Compaction) EffectiveKeepTailTokens() int
func (r *ResultEviction) EffectiveStartTokens() int
```

两者都返回默认值（15000 / 60000）当指针为 nil，返回 `*v` 当已设置。
`Validate` 两处都拒绝 `< 0` 和 `> 10_000_000`，错误文案带完整键路径。

`config.schema.json` 的两个新属性：`keep_tail_tokens` 类型 `["integer","null"]`，`"default": 15000`，`"minimum": 0`，`"maximum": 10000000`，description 说明"压缩保留的尾巴的绝对上限，0 关闭，与 keep_recent_turns 取更小者"；`start_tokens` 同形，默认 60000，description 说明"驱逐门限的绝对上限，与 start_percent 取更小者，且门限只量对话、不量系统提示与 rules"。

`ui_schema.go`：在 `threshold_percent`/`keep_recent_turns` 旁加 `intProp("Keep tail tokens", ...)` 与 `intProp("Start tokens", ...)`，并把两个键加进该段落的顺序列表（`keep_tail_tokens` 紧跟 `keep_recent_turns`，`start_tokens` 紧跟 `start_percent`）。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/config -v`
预期：PASS，且既有的 `TestCompactionDefaults`、`TestResultEvictionDefaults` 不改一行仍通过。

- [ ] **步骤 5：Commit**

```bash
git add internal/config/compaction.go internal/config/config.schema.json internal/config/ui_schema.go internal/config/compaction_test.go
git commit -m "feat(config): add absolute compaction and eviction budgets"
```

---

### 任务 2：常规压缩按尾巴预算选边界

**文件：**
- 修改：`internal/agent/compact.go`（`CompactSession` 的非 `InTurn` 分支，当前 `keep` / `minKeep` / `CompactionSplitIndexUpTo` 循环）
- 测试：`internal/agent/compact_fold_test.go`

- [ ] **步骤 1：编写失败的测试**

用既有 `compactTestAgent` 与 `compactTestProvider`（见 `compact_fold_test.go` 顶部）建会话，每个 exchange 的消息正文用 `strings.Repeat("word ", N)` 控制在几十 token：

```go
func TestRegularCompactionKeepsOnlyTheTurnsThatFitTheTailBudget(t *testing.T)
// 4 个 exchange，两两一轮的用户消息；keep_recent_turns: 2，keep_tail_tokens 调到只够 1 轮
// 断言：下一次 LLM 请求包含最后 1 轮，不包含更早的 3 轮

func TestRegularCompactionKeepsThePromptWhenItAloneExceedsTheTailBudget(t *testing.T)
// keep_tail_tokens: 10，最后一轮本身远超它；断言仍保留最后 1 轮，且没有 error

func TestKeepRecentTurnsZeroWinsOverTheTailBudget(t *testing.T)
// KeepRecentTurns: 0, KeepTailTokens: 15000, 手动 CompactSession(Force: true)
// 断言：请求里没有任何原始 exchange，只有摘要行
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/agent -run 'TestRegularCompactionKeepsOnlyTheTurnsThatFit|TestRegularCompactionKeepsThePromptWhenItAlone|TestKeepRecentTurnsZeroWins' -v`
预期：FAIL，尾巴仍然按 `keep_recent_turns` 保留。

- [ ] **步骤 3：实现**

把"从 keep 往下找到第一个可用边界"的循环改成"从 keep 往下找到第一个既可切又装得下的边界"：

```go
keepTail := a.cfg.Compaction.EffectiveKeepTailTokens()
idx, ok := 0, false
for k := keep; k >= minKeep; k-- {
    candidate, found := session.CompactionSplitIndexUpTo(msgs, k, last)
    if !found {
        continue
    }
    if k == minKeep || keepTail <= 0 || conversationTokens(project()[candidate-viewOffset:], a.modelReadsImages()) <= keepTail {
        idx, ok = candidate, true
        break
    }
}
```

`k == minKeep` 的短路就是"超过预算也保留"的地板。`keepRecentTurns: 0` 经既有的 `if keep < minKeep` 钳制后按原样工作。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/agent -run 'TestRegularCompactionKeepsOnlyTheTurnsThatFit|TestRegularCompactionKeepsThePromptWhenItAlone|TestKeepRecentTurnsZeroWins' -v`
预期：PASS。随后 `go test ./internal/agent -run 'Compaction|Compact'` 全绿（既有场景未设置新键，默认 15000 可能改变它们的期望尾巴——若有测试失败，逐条检查它是否在断言"保留 2 轮"，是则改用 `keep_tail_tokens: 0` 明确禁用绝对上限，而不是放宽默认值）。

- [ ] **步骤 5：Commit**

```bash
git add internal/agent/compact.go internal/agent/compact_fold_test.go
git commit -m "feat(agent): cut the compaction tail by an absolute budget"
```

---

### 任务 3：回合内折点由绝对预算封顶

**文件：**
- 修改：`internal/agent/compact_in_turn.go`（`planInTurn`、新增 `inTurnTargetWithTail`）
- 测试：`internal/agent/compact_in_turn_test.go`

- [ ] **步骤 1：编写失败的测试**

```go
func TestInTurnTargetCappedByTheTailBudget(t *testing.T)
// inTurnTargetWithTail(limit=1_048_576, threshold=80, keepTail=15_000, overhead=120_000) == 135_000

func TestInTurnTargetWithoutATailBudget(t *testing.T)
// keepTail=0 -> inTurnTargetTokens(limit, threshold) 的旧值

func TestInTurnTargetBelowTheRequestOverheadKeepsTheFloor(t *testing.T)
// overhead=200_000, keepTail=15_000 -> target 仍是 215_000，planInTurn 的 budget 转负，
// chooseInTurnSplit 取地板（最新一步），不返回 error

func TestInTurnTargetWithoutAWindow(t *testing.T)
// limit=0 -> inTurnTargetTokens 为 0，封顶后仍为 0，不 panic
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/agent -run 'TestInTurnTarget' -v`
预期：FAIL，`inTurnTargetWithTail` 未定义。

- [ ] **步骤 3：实现**

```go
// inTurnTargetWithTail is the size an in-turn fold aims the next request at:
// the share of the window the trigger leaves, or the request's own overhead plus
// the tail budget when that is smaller. 0 disables the absolute cap.
func inTurnTargetWithTail(limit, thresholdPercent, keepTailTokens, overhead int) int
```

`planInTurn` 用它替换 `target := inTurnTargetTokens(...)`，随后保留既有的 `providerScale` 缩放与 `overhead` 扣除，其余预算计算不动。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/agent -run 'TestInTurnTarget|TestChooseInTurnSplit|TestCompactSessionInTurn' -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/agent/compact_in_turn.go internal/agent/compact_in_turn_test.go
git commit -m "feat(agent): cap the in-turn fold aim at the tail budget"
```

---

### 任务 4：结果驱逐的绝对门限，且只量对话

**文件：**
- 修改：`internal/agent/result_eviction.go`（`evictionDue`、`listingEvictionDue`、新增 `evictionGate`）
- 测试：`internal/agent/result_eviction_test.go`

- [ ] **步骤 1：编写失败的测试**

```go
func TestEvictionGateIsTheSmallerOfPercentAndTokens(t *testing.T)
// window=1_048_576, percent=50, startTokens=60_000 -> gate 60_000
// window=128_000, percent=50, startTokens=60_000 -> gate 60_000
// window=100_000, percent=20, startTokens=60_000 -> gate 20_000

func TestEvictionGateIgnoresTheRequestOverhead(t *testing.T)
// 上下文分解里 Rules=120_000、Conversation=10_000，窗口 262_144，percent=50，startTokens=60_000
// 断言 prunedForLLM 不驱逐（旧实现在 262k 窗口下会把 120k rules 算成 46%，从第一条结果就驱逐）

func TestEvictionGateWithoutAWindow(t *testing.T)
// max_context_tokens=0 且 startTokens=0 -> 仍然驱逐（保留今天的兜底）

func TestEvictionGateZeroTokensKeepsThePercentOnly(t *testing.T)
// startTokens=0, window=200_000, percent=50 -> gate 100_000
```

并把既有的 `TestListingGateCountsTheSystemPromptOnce` 改写成 `TestListingGateDoesNotCountTheSystemPrompt`：断言投影里带不带 system 消息，门限判断结果相同。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/agent -run 'TestEvictionGate|TestListingGate' -v`
预期：FAIL。

- [ ] **步骤 3：实现**

```go
// evictionGate is the number of conversation tokens at which eviction starts:
// the smaller of percent of the window and the absolute start budget. 0 means
// from the first result.
func evictionGate(percent, window, startTokens int) int
```

`evictionDue` 与 `listingEvictionDue` 都改为：

```go
gate := evictionGate(re.EffectiveStartPercent(), window, re.EffectiveStartTokens())
if gate <= 0 { return true }
return conversationTokens(msgs, a.modelReadsImages()) >= gate
```

`window <= 0` 时保持既有兜底（`gate = 0` → always）。`listingEvictionDue` 里剥掉开头 system 消息的代码与注释删除：门限已不再量请求开销，那条"提前一整个 system prompt"的理由不成立。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/agent -run 'TestEvictionGate|TestListingGate|TestPrunedForLLM' -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/agent/result_eviction.go internal/agent/result_eviction_test.go
git commit -m "feat(agent): gate result eviction by an absolute conversation budget"
```

---

### 任务 5：宽窗口落点的场景锁

**文件：**
- 修改：`features/context_in_turn_compaction.feature`
- 修改：`internal/agent/bdd_in_turn_compaction_test.go`（`inTurnLongTurn`、`inTurnFeatureState`、三个新 step、注册表）
- 修改：`internal/agent/bdd_in_turn_compaction_test.go:553` 附近的 `sc.Step` 注册

- [ ] **步骤 1：编写失败的场景**

在 feature 末尾追加：

```gherkin
Scenario: A fold on a wide window lands at the tail budget
  Given a model window of 1048576 tokens
  And auto-compaction triggers at 5 percent of the window
  And the tail budget is 15000 tokens
  And result eviction is off, so only compaction can keep the turn inside the window
  And a model that reads 4 files in parallel at each of 20 steps and then answers
  When the user sends one prompt
  Then the turn was folded once
  And every request after the fold carried at most the tail budget in conversation tokens
```

- [ ] **步骤 2：运行场景验证失败**

运行：`go test ./internal/agent -run TestContextInTurnCompactionFeature -v`
预期：FAIL，`auto-compaction triggers at 5 percent of the window` 未定义。

- [ ] **步骤 3：实现**

`inTurnLongTurn` 加 `thresholdPercent int` 与 `keepTailTokens int`；`run` 里写进 `cfg.Compaction.ThresholdPercent` 与 `cfg.Compaction.KeepTailTokens`（都只在非 0 时设置，`KeepTailTokens` 用局部变量取地址）。`inTurnFeatureState` 加对应字段与 setter。

新 step `every request after the fold carried at most the tail budget in conversation tokens`：对 `provider.accepted` 里**每一条**带有 `CompactionSummary` 或 `CompactionBlock` 行的请求（折叠之后发出的那些），减去它自己的 system 与工具 schema 估算（`requestTokens(messages, nil)` 与 `requestTokens(messages, defs)` 之差），断言剩下的会话部分不超过 `keepTailTokens + 3000`（摘要行与被 10% 份额封顶的 prompt 的余量）。

- [ ] **步骤 4：运行场景验证通过**

运行：`go test ./internal/agent -run TestContextInTurnCompactionFeature -v`
预期：PASS；同一命令在把 `keepTailTokens` 临时改为 0 时应 FAIL（证明场景确实锁住了绝对预算，而不是碰巧过的）——验证后改回。

- [ ] **步骤 5：Commit**

```bash
git add features/context_in_turn_compaction.feature internal/agent/bdd_in_turn_compaction_test.go
git commit -m "test(agent): lock the fold landing on a wide window"
```

---

### 任务 6：文档与生成物

**文件：**
- 修改：`docs/features/compaction.md`（折叠、回合内、结果驱逐、参考表四处）
- 修改：`docs/ru/features/compaction.md`（同四处）
- 修改：`docs/contributing/react-agent.md:142` 与 `:430`（`start_percent` 的说明）
- 生成：`docs/reference/config.md`（`make docs`）
- 视需要：`internal/config/schema_ui_defaults.go`（若 schema 默认值有清单）

- [ ] **步骤 1：更新英文页**

- 折叠一节：尾巴由"最新的若干轮"改为"最新的、装得进 `keep_tail_tokens` 的若干轮，最多 `keep_recent_turns` 轮，最少 1 轮"；给出 1M 窗口 + 15000 的算例。
- 回合内一节：目标改为 `min(窗口 40%, 请求开销 + 15000)`，说明 rules 120k 时落在 ~135k。
- 结果驱逐一节：门限改为 `min(start_percent × 窗口, start_tokens)`，并说明只量对话、不量系统提示与 rules，给出 1M→60k、262k→60k、128k→60k 三个算例。
- 参考表加两行：`keep_tail_tokens` 默认 15000、`result_eviction.start_tokens` 默认 60000。

- [ ] **步骤 2：更新俄文页并重新盖戳**

按仓库的俄文措辞规则（`worktree` 保留，agent→агентный，不用长破折号）翻译同样的四处，然后：

```bash
make docs-stamp PAGES=features/compaction.md
```

- [ ] **步骤 3：重新生成并检查**

```bash
make docs
make docs-check
git diff --stat docs/reference/config.md
```

预期：`config.md` 的两行新字段出现在 compaction 表里，`docs-check` 无 stale 块、无缺页、无坏链。

- [ ] **步骤 4：运行仓库门禁**

```bash
make lint
go test ./internal/config ./internal/agent
```

预期：两项都通过（`make lint` 含 `gofmt -s`、golangci-lint 与 SPA 的 tsc）。

- [ ] **步骤 5：Commit**

```bash
git add docs/features/compaction.md docs/ru/features/compaction.md docs/contributing/react-agent.md docs/reference/config.md
git commit -m "docs(features): document the absolute compaction and eviction budgets"
```

---

## 验收

- `go test ./internal/config ./internal/agent` 全绿。
- `make docs-check` 与 `make lint` 通过。
- 用 `/opt/coddy/config.yaml` 的模型（1,048,576 窗口）跑一个多步回合，日志里 `auto-compacted the turn in progress` 的 `requestTokensAfter` 落在 `requestOverhead + keep_tail_tokens` 附近，而不是 40 万量级。
- 把 `keep_tail_tokens` 与 `start_tokens` 都设为 0，行为与改动前一致（可用 `git stash` 前后对比一次 `requestTokensAfter`）。

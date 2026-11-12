# Compaction landing: an absolute tail budget

Design record, 2026-11-11. First of six parts (see *Remaining parts* at the
end); this one carries the money.

A session on a declared 1,048,576-token window grew to 522k input tokens per
step over 41 steps, and neither guard fired: `compaction.threshold_percent` (80)
puts the trigger at 838,861 and `result_eviction.start_percent` (50) puts the
eviction gate at 524,288. A turn that large paid for its tail 41 times, which
is where the bill came from.

The trigger is not the defect. A 1M window should be usable as a 1M window; a
provider that serves it is not lying. The defect is that **every number that
decides how small a compaction leaves the conversation is a share of that
window**, so the guards scale with the window they exist to protect against:

- the in-turn fold aims the next request at `threshold_percent × 50%` of the
  window - 419k on a 1M window (`internal/agent/compact_in_turn.go`);
- `result_eviction.start_percent` waits for half the window - 524k - before it
  reclaims superseded read and grep results.

A compaction that lands at 419k has not reclaimed anything worth the summarizer
call it cost. What is missing is a ceiling in tokens: the same fold on a 128k
window leaves 51k, which is small; on a 1M window it leaves 419k, which is not.

## 1. What this changes, in one line

A compaction leaves a **fixed-size** tail (15k by default) instead of a share of
the model's window, and tool result eviction starts at a **fixed size** too.
`threshold_percent` keeps its meaning: a 1M window is still used as 1M.

## 2. Invariants

- Nothing here moves the trigger. `threshold_percent` stays a share of the real
  window; a session is allowed to grow to 80% of 1M before it is folded.
- Every number that decides the size of what a compaction leaves behind is
  absolute, or the smaller of an absolute ceiling and the old share.
- An unset key keeps today's behaviour, except where a new default is the point
  (see 3.4).
- The persisted transcript is never rewritten; folds still only insert a summary
  row, and eviction still only changes the LLM projection.

## 3. The change

### 3.1 `compaction.keep_tail_tokens`

New integer, default `15000`, `0` disables it (turn count alone decides, which is
today's behaviour).

The kept tail of a **regular or manual** compaction is the newest user turns
whose estimated size fits the budget, never more than `keep_recent_turns` and
never fewer than the floor (one turn for the automatic trigger, none for
`/compact` with `force`). Concretely, the split search that today starts at
`keep_recent_turns` and walks down only when a boundary does not exist also
walks down while the tail after the boundary is over the budget:

```
for k := keep; k >= minKeep; k-- {
    idx, ok := CompactionSplitIndexUpTo(msgs, k, last)
    if !ok { continue }
    if k == minKeep || tailTokens(idx) <= budget { return idx }
}
```

`minKeep` is returned even when its tail is over budget: folding the prompt
being answered is worse than a tail that is too large, and the trigger will run
again on the next step.

The size is measured like every other budget in compaction: `conversationTokens`
over the **projected** tail (after result eviction, images counted at the same
rate), so a tail full of evictable reads is not blamed for tokens the request
will not carry.

`keep_recent_turns: 0` still means "summarize the whole window" and wins over
`keep_tail_tokens`: an explicit 0 asks for no tail, and a new key must not
resurrect one.

### 3.2 The in-turn aim

`compact_in_turn.go` currently aims the next request at
`limit × threshold_percent × inTurnAimPercent/10000`. It becomes:

```
target = min(limit × threshold × inTurnAimPercent/10000, requestOverhead() + keep_tail_tokens)
```

`requestOverhead()` is the part of the request no fold can remove: the system
prompt, the tool definitions, the rules (including an `AGENTS.md`), skills, MCP
and subagent definitions. On the reported session that overhead is ~120k of
rules, so the new target is ~135k instead of 419k and the kept steps are what
fits in 15k. The budget passed to `chooseInTurnSplit` is unchanged otherwise: no
step at all when the overhead alone exceeds the target, which the existing
"fold spent" logic already handles.

With `keep_tail_tokens: 0` the aim is exactly today's formula.

### 3.3 `compaction.result_eviction.start_tokens`

New integer, default `60000`, `0` disables it (percent alone decides).

The gate becomes:

```
conversationTokens >= min(start_percent × window / 100, start_tokens)
```

measured on the conversation **without** `requestOverhead()`. Two changes in
one:

- the absolute ceiling is what makes eviction useful on a wide window;
- the overhead leaves the measurement, because it cannot be evicted. Today the
  gate counts the rules: on a 262k window a 120k rules document is 46% of the
  gate before the conversation has said anything, so eviction fires from the
  first result - the cache churn `start_percent` exists to prevent. Measured on
  the conversation alone, the gate means the same thing at every window size.

Smaller windows are therefore gated later than today (a 128k window today opens
at ~44k of conversation with 20k of overhead; it will open at 60k), and wide
windows earlier (1M: 60k instead of 500k). Both are intended: eviction is about
how much conversation the request carries.

### 3.4 Defaults, validation, schema

| key | default | bounds |
|---|---|---|
| `compaction.keep_tail_tokens` | 15000 | 0 .. 10,000,000; 0 disables |
| `compaction.result_eviction.start_tokens` | 60000 | 0 .. 10,000,000; 0 disables |

Both go into `internal/config/compaction.go`, `internal/config/config.schema.json`,
`internal/config/ui_schema.go` (Settings form descriptions, and the section's
key list), and the generated `docs/reference/config.md` through `make docs`.

The 15000 is OpenCode's default `compaction.keep.tokens` (v2.2.7,
`packages/core/src/session/compaction.ts:269`), the number the porting request
named; 60000 is chosen so that a 128k window reaches it at roughly the point
today's 50% gate does once a normal prompt and tool set are subtracted.

### 3.5 What does not change

- Overflow recovery (`internal/agent/context_overflow.go`): still one per step,
  still fed by the provider's reported limit and prompt count, still the best
  informed path in the tree.
- The multi-pass fold, the repeated-line dedupe and the summarizer chain.
- `keep_recent_turns` as a cap, and its default of 2.
- The prompt at the start of an in-turn summary row and its 10% share.
- The trigger, the context ring and the `usage_update` numbers.

## 4. What the reported session becomes

1M window, 120k of rules, threshold 80%, `keep_tail_tokens: 15000`:

| | before | after |
|---|---|---|
| in-turn fold aim | 419,430 | 135,000 |
| kept tail after a fold | up to 419k | ~15k |
| eviction gate | 524,288 | 60,000 |
| request per step, worst case | ~522k | ~135k |

The trigger still fires at 838,861: the session still uses its window, it just
does not pay a 500k tail forty times on the way there. A turn that reaches 838k
folds once and lands at ~135k.

## 5. Tests

- `internal/agent/compact_test.go` (and `compact_in_turn_test.go`): a split
  search that walks down for the budget; a floor that stays even when over
  budget; `keep_recent_turns: 0` winning over a non-zero budget; `force`.
- `internal/agent/compact_in_turn_test.go`: the aim is the smaller of the two
  numbers; overhead above the target leaves the floor; `keep_tail_tokens: 0`
  restores the old aim.
- `internal/agent/result_eviction_test.go`: the gate is the smaller of percent
  and tokens; overhead does not trip it; `start_tokens: 0` restores percent-only.
- `internal/config`: defaults, validation, schema acceptance of both keys, and
  `--test-config` reporting a bad value with its line.
- One end-to-end assertion that a long single-turn session folded at the
  threshold lands under `overhead + keep_tail_tokens` (estimate units, scaled by
  `providerScale`, as the fold already does).

## 6. Documentation

- `docs/features/compaction.md`: the fold section, the in-turn section, the
  result-eviction section and the reference table, with the numbers above; the
  matching `docs/ru/features/compaction.md` in the same pull request, restamped
  with `make docs-stamp`.
- `internal/config/config.schema.json` descriptions are the source of the
  generated field tables; `make docs` afterwards.
- Screenshot: no UI layout changes, so none.

## 7. Rollback

Both keys are additive. `keep_tail_tokens: 0` and `start_tokens: 0` restore the
previous arithmetic exactly, without a code revert.

## Remaining parts

Ordered as agreed; each gets its own record and plan.

1. **P1, this one** - the landing.
2. **P2** - the summarizer degradation ladder (tool outputs clipped, then
   removed, then user text only, then omitted) instead of another whole pass.
   Lowest value of the six: the multi-pass fold and the dedupe already bound a
   summarizer request.
3. **P3** - cache-aware compaction: a model-level cache TTL, folding only while
   the prefix is still cached, and a summary request shaped to reuse it.
   Depends on P1: there is no point protecting a cache for a fold that lands at
   419k.
4. **P4** - provider-native compaction for OpenAI Responses
   (`context_management`, `compact_threshold`), with the returned compaction
   item replayed like a native window. Provider-specific, no effect on the
   DeepSeek and mimo rows this record came from.
5. **P5** - token accounting: `stats.json` carries cached and reasoning tokens,
   and `TokenUsageByTurn` stops mixing one step's input with the turn's running
   total (`internal/agent/react.go:1693`).
6. **P6** - `DESIGN.md` leaves the standing layer: `preambleFiles` keeps
   `AGENTS.md`, and a `DESIGN.md` is read on demand from the folders a tool
   enters, like a nested `AGENTS.md`.

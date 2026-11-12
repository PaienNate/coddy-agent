package config

import (
	"fmt"
	"strings"
)

// Defaults for the compaction section when YAML omits values.
const (
	// CompactionDefaultThresholdPercent triggers auto-compaction when the estimated
	// context reaches this percent of the model's context window.
	CompactionDefaultThresholdPercent = 80
	// CompactionDefaultKeepRecentTurns is how many recent user turns stay verbatim.
	CompactionDefaultKeepRecentTurns = 2
	// CompactionDefaultKeepTailTokens is the absolute ceiling on the tail a
	// compaction keeps verbatim, whatever the model's window is. A share of a
	// wide window leaves a tail as wide, and every step after the fold pays for
	// it again; 15000 is small enough that a fold reclaims the window and large
	// enough that the exchange being answered stays in reach. 0 turns the
	// absolute cap off and leaves keep_recent_turns alone to decide.
	CompactionDefaultKeepTailTokens = 15000
)

// compactionBudgetCeiling bounds the absolute token budgets of the compaction
// section: a value above it is a typo, not a request.
const compactionBudgetCeiling = 10_000_000

// Defaults for the compaction.result_eviction subsection.
const (
	// ResultEvictionDefaultKeepRecent is how many most recent evictable read and
	// grep results (read pages, grep dumps) stay intact as a working window.
	// 2, not 1: a window of 1 cannot hold a read plus a grep at the same time, so
	// a model comparing the two keeps re-fetching whichever the other just evicted
	// (the loop guard does not catch it, because alternating calls are not
	// identical). 2 keeps the common working pair live for one extra result.
	ResultEvictionDefaultKeepRecent = 2
	// ResultEvictionDefaultMinResultBytes is the size at or below which a tool
	// result is never evicted (too small to be worth a placeholder).
	ResultEvictionDefaultMinResultBytes = 2000
	// ResultEvictionDefaultStartPercent is the share of the model's context
	// window the conversation must reach before eviction starts rewriting it.
	// Below it the history is sent untouched, because every placeholder that
	// appears mid-history is a byte the provider's prompt cache keyed the rest
	// of the conversation on: a sliding window would reprocess the whole
	// transcript on almost every step to save a few thousand tokens nobody was
	// short of yet.
	ResultEvictionDefaultStartPercent = 50
	// ResultEvictionDefaultStartTokens is the absolute ceiling on the eviction
	// gate: eviction starts at the smaller of start_percent of the window and
	// this many conversation tokens, so a window wide enough that half of it is
	// far past the point where dead results matter does not wait for that half.
	// 0 turns the absolute gate off and leaves start_percent alone.
	ResultEvictionDefaultStartTokens = 60000
)

// Compaction is the YAML compaction section (key compaction): summarizing older
// conversation history so long sessions keep fitting the model context window.
type Compaction struct {
	// Enabled toggles compaction (the manual command and the automatic trigger).
	// A nil pointer means the default (true).
	Enabled *bool `yaml:"enable"`
	// AutoEnabled turns off only the threshold trigger; manual compaction stays available.
	AutoEnabled *bool `yaml:"auto_enable"`
	// ThresholdPercent fires auto-compaction when the estimated context usage
	// reaches this percent of the effective model's context window (default
	// 80, valid 1..100): its max_context_tokens, else the window its
	// provider's model listing reports, else DefaultContextWindowTokens.
	ThresholdPercent int `yaml:"threshold_percent"`
	// KeepRecentTurns is how many most recent user turns (each with the agent
	// replies and tool activity after it) stay verbatim; only history before
	// that boundary is summarized. A nil pointer means the default (2); an
	// explicit 0 summarizes the whole window. When the window holds no more
	// user turns than this, a compaction keeps fewer: the automatic trigger
	// down to the prompt being answered, the manual command down to none.
	KeepRecentTurns *int `yaml:"keep_recent_turns"`
	// KeepTailTokens is the absolute ceiling on the tail a compaction keeps
	// verbatim, in estimated tokens: the newest turns that fit it stay, never
	// more than KeepRecentTurns and never fewer than one (none for a manual
	// compaction that folds everything). It bounds what a wide window leaves
	// behind, which is what the next steps would otherwise pay for again. A nil
	// pointer means the default (15000); 0 turns the absolute cap off, so
	// KeepRecentTurns alone decides, which is how Coddy behaved before.
	KeepTailTokens *int `yaml:"keep_tail_tokens"`
	// Model optionally selects the models[].model used for the summarization
	// call. Empty means the session's effective model.
	Model string `yaml:"model"`
	// FallbackModels are the models[].model ids tried, in order, when the
	// summarizer above them fails. A compaction is what a session out of room
	// has left, so one unreachable or overloaded model must not be the end of
	// it; the session's own model is always the last resort, whether or not it
	// is listed here (issue #247).
	FallbackModels []string `yaml:"fallback_models"`
	// ResultEviction controls pruning of superseded tool results (read pages,
	// grep dumps, and the listings of glob, print_tree, websearch and webfetch)
	// from the LLM projection (the persisted transcript is never rewritten).
	ResultEviction ResultEviction `yaml:"result_eviction"`
}

// ResultEviction is the YAML compaction.result_eviction section: collapsing
// superseded tool results to short placeholders when building the LLM request,
// so paging a large file, a wide search or a fan-out of directory listings cannot
// pin dead lines in every later turn. Unmarked read and grep results go by a
// window of results (KeepRecent); the listings of glob, print_tree, websearch and
// webfetch go by a fixed window of whole steps (internal/agent).
type ResultEviction struct {
	// Enabled toggles the projection. A nil pointer means the default (true).
	Enabled *bool `yaml:"enable"`
	// KeepRecent is how many most recent evictable read and grep results stay
	// intact as a working window. A nil pointer means the default (2); 0 keeps
	// none. Listing results do not count toward it.
	KeepRecent *int `yaml:"keep_recent"`
	// MinResultBytes is the size at or below which a result is never evicted.
	// A nil pointer means the default (2000); 0 makes every result a candidate.
	MinResultBytes *int `yaml:"min_result_bytes"`
	// StartPercent is the share of the model's context window the estimated
	// context must reach before eviction starts (default 50, valid 0..100). 0
	// evicts from the first result, which is what the projection did before
	// prompt caching was accounted for. Read and grep results of a model without
	// max_context_tokens are evicted from the start; listing results measure
	// against the window its provider reports, else the default.
	StartPercent *int `yaml:"start_percent"`
	// StartTokens is the absolute ceiling on the eviction gate: eviction starts
	// once the conversation, measured without the request's own overhead (the
	// system prompt, the tool definitions, the rules), holds at least this many
	// tokens, or start_percent of the window when that is smaller. The overhead
	// is left out because nothing can evict it and a large AGENTS.md would
	// otherwise open the gate at the first result. A nil pointer means the
	// default (60000); 0 turns the absolute gate off and leaves start_percent
	// alone.
	StartTokens *int `yaml:"start_tokens"`
}

// IsEnabled reports whether result eviction is active. Defaults to true when unset.
func (r *ResultEviction) IsEnabled() bool {
	return r.Enabled == nil || *r.Enabled
}

// EffectiveKeepRecent returns keep_recent with the default applied.
func (r *ResultEviction) EffectiveKeepRecent() int {
	if r.KeepRecent == nil {
		return ResultEvictionDefaultKeepRecent
	}
	return *r.KeepRecent
}

// EffectiveMinResultBytes returns min_result_bytes with the default applied.
func (r *ResultEviction) EffectiveMinResultBytes() int {
	if r.MinResultBytes == nil {
		return ResultEvictionDefaultMinResultBytes
	}
	return *r.MinResultBytes
}

// EffectiveStartPercent returns start_percent with the default applied.
func (r *ResultEviction) EffectiveStartPercent() int {
	if r.StartPercent == nil {
		return ResultEvictionDefaultStartPercent
	}
	return *r.StartPercent
}

// EffectiveStartTokens returns start_tokens with the default applied. 0 turns
// the absolute gate off.
func (r *ResultEviction) EffectiveStartTokens() int {
	if r.StartTokens == nil {
		return ResultEvictionDefaultStartTokens
	}
	return *r.StartTokens
}

// Validate checks bounds on explicitly set fields.
func (r *ResultEviction) Validate() error {
	if r.KeepRecent != nil && *r.KeepRecent < 0 {
		return fmt.Errorf("compaction.result_eviction.keep_recent: must be >= 0")
	}
	if r.MinResultBytes != nil && *r.MinResultBytes < 0 {
		return fmt.Errorf("compaction.result_eviction.min_result_bytes: must be >= 0")
	}
	if r.StartPercent != nil && (*r.StartPercent < 0 || *r.StartPercent > 100) {
		return fmt.Errorf("compaction.result_eviction.start_percent: must be between 0 and 100")
	}
	if r.StartTokens != nil && (*r.StartTokens < 0 || *r.StartTokens > compactionBudgetCeiling) {
		return fmt.Errorf("compaction.result_eviction.start_tokens: must be between 0 and %d", compactionBudgetCeiling)
	}
	return nil
}

// IsEnabled reports whether compaction is active. Defaults to true when unset.
func (c *Compaction) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

// IsAutoEnabled reports whether the automatic trigger may run.
func (c *Compaction) IsAutoEnabled() bool {
	return c.IsEnabled() && (c.AutoEnabled == nil || *c.AutoEnabled)
}

// EffectiveThresholdPercent returns threshold_percent with the default applied
// (covers configs constructed without ApplyDefaults).
func (c *Compaction) EffectiveThresholdPercent() int {
	if c.ThresholdPercent <= 0 {
		return CompactionDefaultThresholdPercent
	}
	return c.ThresholdPercent
}

// EffectiveKeepRecentTurns returns keep_recent_turns with the default applied.
func (c *Compaction) EffectiveKeepRecentTurns() int {
	if c.KeepRecentTurns == nil {
		return CompactionDefaultKeepRecentTurns
	}
	return *c.KeepRecentTurns
}

// EffectiveKeepTailTokens returns keep_tail_tokens with the default applied. 0
// turns the absolute cap off.
func (c *Compaction) EffectiveKeepTailTokens() int {
	if c.KeepTailTokens == nil {
		return CompactionDefaultKeepTailTokens
	}
	return *c.KeepTailTokens
}

// Normalize trims string fields in place.
func (c *Compaction) Normalize() {
	c.Model = strings.TrimSpace(c.Model)
}

// ApplyDefaults sets ThresholdPercent when it is zero.
func (c *Compaction) ApplyDefaults() {
	if c.ThresholdPercent == 0 {
		c.ThresholdPercent = CompactionDefaultThresholdPercent
	}
}

// Validate checks bounds after defaults.
func (c *Compaction) Validate() error {
	if c.ThresholdPercent < 1 || c.ThresholdPercent > 100 {
		return fmt.Errorf("compaction.threshold_percent: must be within 1..100")
	}
	if c.KeepRecentTurns != nil && *c.KeepRecentTurns < 0 {
		return fmt.Errorf("compaction.keep_recent_turns: must be >= 0")
	}
	if c.KeepTailTokens != nil && (*c.KeepTailTokens < 0 || *c.KeepTailTokens > compactionBudgetCeiling) {
		return fmt.Errorf("compaction.keep_tail_tokens: must be between 0 and %d", compactionBudgetCeiling)
	}
	if err := c.ResultEviction.Validate(); err != nil {
		return err
	}
	return nil
}

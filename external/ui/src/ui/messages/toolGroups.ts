import type { TranscriptItem } from "../chat/types";

/** One tool call row, with the index it has in the whole transcript. */
export type ToolCallItem = Extract<TranscriptItem, { type: "tool_call" }>;
export type IndexedToolCall = { item: ToolCallItem; index: number };

/** A run of adjacent tool calls, folded into one transcript row. */
export type ToolGroupRow = {
  kind: "group";
  key: string;
  items: IndexedToolCall[];
};

export type ToolRenderRow =
  | { kind: "item"; item: TranscriptItem; index: number }
  | ToolGroupRow;

export type ToolGroupCategory =
  | "shell"
  | "read"
  | "search"
  | "edit"
  | "web"
  | "agent"
  | "other";

/**
 * Groups adjacent tool calls into one display row. A single call stays as it is;
 * text, thinking, user messages and the interactive cards break a run, because
 * the caller refuses them from `canGroup` and the loop flushes the current run.
 *
 * The result keeps each item's absolute index: the transcript uses it for the
 * assistant artifact markers and for the last-row notice rules, and a group must
 * not make a row forget where it came from.
 */
export function groupToolCalls(
  items: readonly TranscriptItem[],
  startIndex: number,
  canGroup: (item: ToolCallItem) => boolean,
): ToolRenderRow[] {
  const out: ToolRenderRow[] = [];
  let run: IndexedToolCall[] = [];

  const flush = () => {
    if (run.length === 0) return;
    if (run.length === 1) {
      const only = run[0]!;
      out.push({ kind: "item", item: only.item, index: only.index });
    } else {
      out.push({
        kind: "group",
        key: `tool_group:${run[0]!.item.id}`,
        items: run,
      });
    }
    run = [];
  };

  items.forEach((item, offset) => {
    if (item.type === "tool_call" && canGroup(item)) {
      run.push({ item, index: startIndex + offset });
      return;
    }
    flush();
    out.push({ kind: "item", item, index: startIndex + offset });
  });
  flush();
  return out;
}

/** The category a group summary counts this call under. */
export function toolGroupCategory(item: ToolCallItem): ToolGroupCategory {
  const name = (item.title || item.kind || "").trim().toLowerCase();
  switch (name) {
    case "bash":
    case "shell":
    case "run_command":
    case "ssh_run_command":
      return "shell";
    case "read":
    case "read_file":
      return "read";
    case "grep":
    case "glob":
    case "print_tree":
    case "list":
    case "list_dir":
      return "search";
    case "edit":
    case "write":
    case "write_file":
    case "apply_patch":
    case "patch":
      return "edit";
    case "websearch":
    case "webfetch":
    case "http_request":
      return "web";
    case "spawn_agent":
    case "task":
    case "subagent":
      return "agent";
    default:
      return "other";
  }
}

/** Counts in the order the summary reads: Shell, Read, Search, Edit, Web, Agent, Other. */
export function toolGroupCounts(
  items: readonly ToolCallItem[],
): { category: ToolGroupCategory; count: number }[] {
  const counts = new Map<ToolGroupCategory, number>();
  for (const item of items) {
    const category = toolGroupCategory(item);
    counts.set(category, (counts.get(category) || 0) + 1);
  }
  const order: ToolGroupCategory[] = [
    "shell",
    "read",
    "search",
    "edit",
    "web",
    "agent",
    "other",
  ];
  return order
    .map((category) => ({ category, count: counts.get(category) || 0 }))
    .filter((row) => row.count > 0);
}

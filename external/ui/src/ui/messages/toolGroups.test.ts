import { expect, test } from "vitest";
import type { TranscriptItem } from "../chat/types";
import {
  groupToolCalls,
  toolGroupCategory,
  toolGroupCounts,
  type ToolCallItem,
} from "./toolGroups";

function tool(
  id: string,
  name: string,
  status: ToolCallItem["status"] = "completed",
): ToolCallItem {
  return { id: `tc_${id}`, type: "tool_call", toolCallId: id, title: name, status };
}

function text(id: string): TranscriptItem {
  return { id, type: "assistant_message", content: "done" };
}

test("groups adjacent tool calls and breaks the run at text", () => {
  const items: TranscriptItem[] = [
    tool("a", "run_command"),
    tool("b", "edit"),
    text("answer"),
    tool("c", "read"),
    tool("d", "read"),
  ];
  const rows = groupToolCalls(items, 7, () => true);
  expect(rows.map((row) => row.kind)).toEqual(["group", "item", "group"]);
  if (rows[0]?.kind !== "group" || rows[2]?.kind !== "group") {
    throw new Error("expected groups");
  }
  expect(rows[0].items.map((row) => row.index)).toEqual([7, 8]);
  expect(rows[1]).toMatchObject({ kind: "item", index: 9 });
  expect(rows[2].items.map((row) => row.index)).toEqual([10, 11]);
});

test("a single tool call is not wrapped in a group", () => {
  const rows = groupToolCalls([tool("a", "read")], 0, () => true);
  expect(rows).toHaveLength(1);
  expect(rows[0]).toMatchObject({ kind: "item", item: { id: "tc_a" } });
});

test("canGroup splits a run, so an interactive card stays on its own", () => {
  const items: TranscriptItem[] = [
    tool("a", "read"),
    tool("b", "question"),
    tool("c", "read"),
  ];
  const rows = groupToolCalls(items, 0, (item) => item.title !== "question");
  expect(rows.map((row) => row.kind)).toEqual(["item", "item", "item"]);
});

test("categories count shell, reads, edits and other tools", () => {
  const items = [
    tool("a", "run_command"),
    tool("b", "ssh_run_command"),
    tool("c", "read"),
    tool("d", "grep"),
    tool("e", "edit"),
    tool("f", "write"),
    tool("g", "webfetch"),
    tool("h", "spawn_agent"),
    tool("i", "coddy_todo_plan_read"),
  ];
  expect(toolGroupCounts(items)).toEqual([
    { category: "shell", count: 2 },
    { category: "read", count: 1 },
    { category: "search", count: 1 },
    { category: "edit", count: 2 },
    { category: "web", count: 1 },
    { category: "agent", count: 1 },
    { category: "other", count: 1 },
  ]);
  expect(toolGroupCategory(items[0]!)).toBe("shell");
  expect(toolGroupCategory(items[8]!)).toBe("other");
});

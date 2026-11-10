import { beforeEach, expect, test } from "vitest";
import type { SessionRow } from "../sessions/types";
import {
  closeLeft,
  closeOthers,
  closeRight,
  closeTabs,
  nextActiveAfterClose,
  openTab,
  orderedTabs,
  readProjectTabMemory,
  reconcileTabMemory,
  reorderTabs,
  togglePinned,
  visibleTabs,
  writeProjectTabMemory,
  type ProjectTabMemory,
} from "./projectTabs";

const row = (id: string, updatedAt = "2026-01-01T00:00:00Z"): SessionRow => ({
  id,
  title: id,
  updatedAt,
});

const memory = (
  order: string[],
  pinned: string[] = [],
  closed: string[] = [],
): ProjectTabMemory => ({
  order,
  pinned,
  closed,
});

beforeEach(() => {
  localStorage.clear();
});

test("reconcile puts a session the strip has not seen at the front and keeps named order", () => {
  const next = reconcileTabMemory(
    [row("new"), row("a"), row("b")],
    memory(["a", "gone", "b"]),
  );
  expect(next.order).toEqual(["new", "a", "gone", "b"]);
});

test("an empty listing keeps the remembered pins and order while rows load", () => {
  const held = memory(["b", "a"], ["b"], ["c"]);
  expect(reconcileTabMemory([], held)).toEqual(held);
});

test("orderedTabs keeps pinned tabs first and then the operator's order", () => {
  const rows = [row("a"), row("b"), row("c")];
  expect(
    orderedTabs(rows, memory(["c", "b", "a"], ["a", "c"])).map((r) => r.id),
  ).toEqual(["c", "a", "b"]);
  expect(
    visibleTabs(rows, memory(["a", "b", "c"], [], ["b"])).map((r) => r.id),
  ).toEqual(["a", "c"]);
});

test("pin moves a tab to the front; unpin keeps its place", () => {
  const pinned = togglePinned(memory(["a", "b", "c"]), "c");
  expect(pinned.pinned).toEqual(["c"]);
  expect(pinned.order).toEqual(["c", "a", "b"]);
  const unpinned = togglePinned(pinned, "c");
  expect(unpinned.pinned).toEqual([]);
  expect(unpinned.order).toEqual(["c", "a", "b"]);
});

test("opening a closed tab brings it back into view", () => {
  const next = openTab(memory(["a", "b"], [], ["a"]), "a");
  expect(next.closed).toEqual([]);
  expect(next.order).toEqual(["a", "b"]);
});

test("close others keeps the target and every pinned tab", () => {
  const next = closeOthers(memory(["a", "b", "c", "d"], ["c"]), "b", [
    "a",
    "b",
    "c",
    "d",
  ]);
  expect(next.closed).toEqual(["a", "d"]);
});

test("close left and right protect pinned tabs", () => {
  const base = memory(["a", "b", "c", "d"], ["b"]);
  expect(closeLeft(base, "d", ["a", "b", "c", "d"]).closed).toEqual(["a", "c"]);
  expect(closeRight(base, "a", ["a", "b", "c", "d"]).closed).toEqual([
    "c",
    "d",
  ]);
});

test("reorder moves a tab to the drop target and keeps hidden tabs remembered", () => {
  const next = reorderTabs(
    memory(["a", "b", "c", "hidden"], [], ["hidden"]),
    ["a", "b", "c"],
    "a",
    "c",
  );
  expect(next.order).toEqual(["b", "c", "a", "hidden"]);
});

test("closing the active tab picks the one to its right, then the one to its left", () => {
  expect(nextActiveAfterClose(["a", "b", "c"], ["b"], "b")).toBe("c");
  expect(nextActiveAfterClose(["a", "b", "c"], ["c"], "c")).toBe("b");
  expect(nextActiveAfterClose(["a"], ["a"], "a")).toBe("");
  expect(nextActiveAfterClose(["a", "b"], ["a"], "b")).toBe("b");
});

test("closing many tabs settles on the nearest survivor", () => {
  expect(nextActiveAfterClose(["a", "b", "c", "d"], ["a", "c", "d"], "c")).toBe(
    "b",
  );
  expect(closeTabs(memory(["a", "b"]), ["a", "a", ""]).closed).toEqual(["a"]);
});

test("the tab memory round-trips through storage", () => {
  writeProjectTabMemory("prj_1", memory(["b", "a"], ["a"], ["c"]));
  expect(readProjectTabMemory("prj_1")).toEqual({
    order: ["b", "a"],
    pinned: ["a"],
    closed: ["c"],
  });
  localStorage.setItem("coddy.projectTabs.v1.bad", "{");
  expect(readProjectTabMemory("bad")).toEqual({
    order: [],
    pinned: [],
    closed: [],
  });
});

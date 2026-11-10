import { expect, test } from "vitest";
import { projectLastAccessAgo, projectLastAccessFull } from "./lastAccess";

const now = Date.parse("2026-11-10T12:00:00Z");

test("recent project opens read as a relative time", () => {
  expect(projectLastAccessAgo("2026-11-10T11:59:30Z", now, "en-US")).toBe(
    "30 sec. ago",
  );
  expect(projectLastAccessAgo("2026-11-08T12:00:00Z", now, "en-US")).toBe(
    "2 days ago",
  );
});

test("older project opens step up to weeks, months and years", () => {
  expect(projectLastAccessAgo("2026-10-21T12:00:00Z", now, "en-US")).toBe(
    "2 wk. ago",
  );
  expect(projectLastAccessAgo("2026-09-10T12:00:00Z", now, "en-US")).toBe(
    "2 mo. ago",
  );
  expect(projectLastAccessAgo("2024-10-10T12:00:00Z", now, "en-US")).toBe(
    "2 yr. ago",
  );
});

test("a missing stamp says nothing and a clock-skewed one reads as now", () => {
  expect(projectLastAccessAgo("", now, "en-US")).toBe("");
  expect(projectLastAccessAgo("not a date", now, "en-US")).toBe("");
  expect(projectLastAccessAgo("2026-11-10T12:05:00Z", now, "en-US")).toBe(
    "now",
  );
});

test("the tooltip carries the exact local instant", () => {
  expect(projectLastAccessFull("2026-11-10T10:00:00Z", "en-US")).toContain(
    "2026",
  );
  expect(projectLastAccessFull("", "en-US")).toBe("");
});

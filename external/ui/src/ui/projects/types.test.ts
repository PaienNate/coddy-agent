import { beforeEach, expect, test } from "vitest";
import { markProjectOpened, readProjectLastOpened } from "./types";

beforeEach(() => {
  localStorage.clear();
});

test("an unopened browser has no project access stamps", () => {
  expect(readProjectLastOpened()).toEqual({});
});

test("opening a project records the instant and keeps the other projects", () => {
  markProjectOpened("one", "2026-11-10T10:00:00Z");
  const next = markProjectOpened("two", "2026-11-10T11:00:00Z");
  expect(next).toEqual({
    one: "2026-11-10T10:00:00Z",
    two: "2026-11-10T11:00:00Z",
  });
  expect(readProjectLastOpened()).toEqual(next);
});

test("a broken store is ignored instead of breaking the Projects page", () => {
  localStorage.setItem("coddy.projects.lastOpened.v1", "not json");
  expect(readProjectLastOpened()).toEqual({});
});

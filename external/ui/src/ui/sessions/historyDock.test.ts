import { beforeEach, expect, test } from "vitest";
import {
  readHistoryDockPreference,
  writeHistoryDockPreference,
} from "./historyDock";

beforeEach(() => {
  localStorage.clear();
});

test("the desktop History column is on until the operator closes it", () => {
  expect(readHistoryDockPreference()).toBe(true);
  writeHistoryDockPreference(false);
  expect(readHistoryDockPreference()).toBe(false);
  writeHistoryDockPreference(true);
  expect(readHistoryDockPreference()).toBe(true);
});

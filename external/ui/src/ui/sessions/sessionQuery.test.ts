import { expect, test } from "vitest";
import {
  SESSION_ORIGIN_FILTERS,
  defaultSortOrder,
  isHistorySortKey,
  isSessionOriginFilter,
} from "./sessionQuery";
import { messagesEn } from "../i18n/messages/en";
import { messagesRu } from "../i18n/messages/ru";

test("print runs are an origin the History filter can pick", () => {
  expect(SESSION_ORIGIN_FILTERS).toEqual(["", "local", "gateway", "print"]);
  expect(isSessionOriginFilter("print")).toBe(true);
  expect(isSessionOriginFilter("cli")).toBe(false);
});

test("the History row for print runs is named in every language", () => {
  expect(messagesEn["sessions.filter.env.print"]).toBe("CLI runs");
  expect(messagesRu["sessions.filter.env.print"]).toBe("Запуски CLI");
});

// Feature 1: "recently used" sorts the projects sidebar. The History filter
// offers the column, and a date ordering starts at its newest by default.
test("last_used is a History sort key the server accepts", () => {
  expect(isHistorySortKey("last_used")).toBe(true);
  expect(defaultSortOrder("last_used")).toBe("desc");
  expect(messagesEn["sessions.sort.lastUsed"]).toBe("Last used");
  expect(messagesRu["sessions.sort.lastUsed"]).toBe("Последнего использования");
});

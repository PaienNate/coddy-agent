import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, test } from "vitest";
import { initLocale } from "../i18n/i18n";
import type { TranscriptItem } from "../chat/types";
import { MessageList } from "./MessageList";

function tool(
  id: string,
  name: string,
  status: "completed" | "in_progress",
): TranscriptItem {
  return {
    id: `tc_${id}`,
    type: "tool_call",
    toolCallId: id,
    title: name,
    status,
  };
}

beforeEach(() => initLocale("en"));
afterEach(() => cleanup());

test("a completed run of tool calls is one folded summary", () => {
  const items = [
    tool("a", "read", "completed"),
    tool("b", "run_command", "completed"),
  ];
  const { container } = render(<MessageList items={items} />);
  const group = screen.getByTestId("tool-group");
  const summary = screen.getByTestId("tool-group-summary");
  expect(group).toHaveAttribute("data-status", "completed");
  expect(summary).toHaveTextContent("Used 2 tools");
  expect(summary).toHaveTextContent("Read 1");
  expect(summary).toHaveTextContent("Shell 1");
  expect(container.querySelector('[data-testid="tool-details-a"]')).toBeNull();

  const details = group.querySelector("details")!;
  details.open = true;
  fireEvent(details, new Event("toggle"));
  expect(
    container.querySelector('[data-testid="tool-details-a"]'),
  ).not.toBeNull();
  expect(
    container.querySelector('[data-testid="tool-details-b"]'),
  ).not.toBeNull();
});

test("a running group starts open, and one call stays a row", () => {
  const running = [
    tool("a", "read", "in_progress"),
    tool("b", "read", "completed"),
  ];
  const { container, unmount } = render(<MessageList items={running} />);
  expect(screen.getByTestId("tool-group")).toHaveAttribute(
    "data-status",
    "running",
  );
  expect(
    container.querySelector('[data-testid="tool-details-a"]'),
  ).not.toBeNull();
  unmount();

  render(<MessageList items={[tool("single", "read", "completed")]} />);
  expect(screen.queryByTestId("tool-group")).toBeNull();
});

test("an interactive question is not folded away", () => {
  const items = [
    tool("a", "read", "completed"),
    tool("b", "question", "completed"),
    tool("c", "read", "completed"),
  ];
  const { container } = render(<MessageList items={items} />);
  expect(screen.queryByTestId("tool-group")).toBeNull();
  expect(
    container.querySelectorAll('[data-testid^="tool-details-"]').length,
  ).toBe(3);
});

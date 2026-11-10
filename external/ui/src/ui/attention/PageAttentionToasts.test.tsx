import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import {
  PageAttentionToasts,
  type PageAttentionNotice,
} from "./PageAttentionToasts";

afterEach(() => cleanup());

const notices: PageAttentionNotice[] = [
  {
    id: "permission:s1",
    sessionId: "s1",
    kind: "permission",
    title: "Fix the build",
    body: "A tool needs your permission",
  },
  {
    id: "question:s2",
    sessionId: "s2",
    kind: "question",
    title: "Plan the migration",
    body: "The agent is waiting for your answer",
  },
];

test("waiting work appears as clickable page notices", () => {
  const onOpen = vi.fn();
  render(
    <PageAttentionToasts
      notices={notices}
      onOpen={onOpen}
      onDismiss={() => {}}
    />,
  );
  expect(screen.getByTestId("page-attention")).toHaveTextContent(
    "Fix the build",
  );
  fireEvent.click(screen.getByTestId("page-attention-permission-s1"));
  expect(onOpen).toHaveBeenCalledWith("s1");
});

test("a notice can be put away without opening its conversation", () => {
  const onOpen = vi.fn();
  const onDismiss = vi.fn();
  render(
    <PageAttentionToasts
      notices={notices}
      onOpen={onOpen}
      onDismiss={onDismiss}
    />,
  );
  fireEvent.click(screen.getAllByLabelText("Dismiss notice")[1]!);
  expect(onDismiss).toHaveBeenCalledWith("question:s2");
  expect(onOpen).not.toHaveBeenCalled();
});

test("no notices render no stack", () => {
  render(
    <PageAttentionToasts notices={[]} onOpen={() => {}} onDismiss={() => {}} />,
  );
  expect(screen.queryByTestId("page-attention")).toBeNull();
});

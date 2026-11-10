import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import type { SessionRow } from "../sessions/types";
import type { Project } from "./types";

const listWorkspaceSessions = vi.fn<() => Promise<SessionRow[]>>();
vi.mock("./api", () => ({
  listWorkspaceSessions: () => listWorkspaceSessions(),
}));

import { ProjectTabs } from "./ProjectTabs";
import { readProjectTabMemory } from "./projectTabs";

const project: Project = {
  id: "prj_1",
  name: "coddy",
  path: "/work/coddy",
  order: 0,
  createdAt: "",
  updatedAt: "",
};
const projects = [project];

const rows: SessionRow[] = [
  { id: "a", title: "Alpha", updatedAt: "2026-01-03T00:00:00Z" },
  { id: "b", title: "Beta", updatedAt: "2026-01-02T00:00:00Z" },
  { id: "c", title: "Gamma", updatedAt: "2026-01-01T00:00:00Z" },
];

function renderTabs(activeSessionId = "b") {
  const onSelectSession = vi.fn();
  const onNewChat = vi.fn();
  const onSwitchProject = vi.fn();
  const onOpenProjects = vi.fn();
  const onHome = vi.fn();
  const view = render(
    <ProjectTabs
      project={project}
      projects={projects}
      activeSessionId={activeSessionId}
      activeSessionTitle="Beta"
      onSelectSession={onSelectSession}
      onNewChat={onNewChat}
      onSwitchProject={onSwitchProject}
      onOpenProjects={onOpenProjects}
      onHome={onHome}
    />,
  );
  return {
    ...view,
    onSelectSession,
    onNewChat,
    onSwitchProject,
    onOpenProjects,
    onHome,
  };
}

beforeEach(() => {
  localStorage.clear();
  listWorkspaceSessions.mockReset();
  listWorkspaceSessions.mockResolvedValue(rows);
  Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
    configurable: true,
    value: vi.fn(),
  });
});

afterEach(() => {
  cleanup();
});

test("a child transcript can send its already-active parent tab back to the parent", async () => {
  const view = renderTabs("b");
  await screen.findByText("Beta");
  fireEvent.click(screen.getByText("Beta"));
  expect(view.onSelectSession).not.toHaveBeenCalled();

  view.rerender(
    <ProjectTabs
      project={project}
      projects={projects}
      activeSessionId="b"
      activeSessionTitle="Beta"
      reselectActive
      onSelectSession={view.onSelectSession}
      onNewChat={view.onNewChat}
      onSwitchProject={view.onSwitchProject}
      onOpenProjects={view.onOpenProjects}
      onHome={view.onHome}
    />,
  );
  fireEvent.click(screen.getByText("Beta"));
  expect(view.onSelectSession).toHaveBeenCalledWith("b");
});

test("a project renders one tab per conversation and the active one is selected", async () => {
  renderTabs();
  expect(await screen.findByText("Alpha")).toBeTruthy();
  expect(screen.getByText("Gamma")).toBeTruthy();
  const active = document.querySelector('[data-session-tab="b"]');
  expect(active?.getAttribute("aria-selected")).toBe("true");
});

test("a tab picks its conversation and its own close keeps the nearest survivor", async () => {
  const view = renderTabs();
  await screen.findByText("Alpha");
  const order = [...document.querySelectorAll("[data-session-tab]")].map((el) =>
    el.getAttribute("data-session-tab"),
  );
  expect(order).toEqual(["a", "b", "c"]);
  fireEvent.click(screen.getByText("Alpha"));
  expect(view.onSelectSession).toHaveBeenCalledWith("a");

  fireEvent.click(screen.getByLabelText("Close Beta"));
  expect(view.onSelectSession).toHaveBeenCalledWith("c");
});

test("a vertical wheel over an overflowing strip scrolls it sideways", async () => {
  renderTabs();
  await screen.findByText("Alpha");
  const strip = screen.getByRole("tablist");
  Object.defineProperty(strip, "scrollWidth", {
    configurable: true,
    value: 600,
  });
  Object.defineProperty(strip, "clientWidth", {
    configurable: true,
    value: 200,
  });
  strip.scrollLeft = 0;
  const wheel = new WheelEvent("wheel", {
    deltaY: 120,
    bubbles: true,
    cancelable: true,
  });
  strip.dispatchEvent(wheel);
  expect(wheel.defaultPrevented).toBe(true);
  expect(strip.scrollLeft).toBe(120);

  const trackpad = new WheelEvent("wheel", {
    deltaX: 120,
    bubbles: true,
    cancelable: true,
  });
  strip.dispatchEvent(trackpad);
  expect(trackpad.defaultPrevented).toBe(false);
});

test("a pinned tab moves to the front and the choice survives a remount", async () => {
  const first = renderTabs();
  await screen.findByText("Beta");
  fireEvent.click(screen.getByLabelText("Tab actions for Beta"));
  fireEvent.click(await screen.findByTestId("project-tab-pin"));
  await waitFor(() => {
    const beta = document.querySelector('[data-session-tab="b"]');
    expect(beta?.className).toContain("is-pinned");
    expect(
      document
        .querySelectorAll("[data-session-tab]")[0]
        ?.getAttribute("data-session-tab"),
    ).toBe("b");
  });
  await waitFor(() =>
    expect(
      JSON.parse(localStorage.getItem("coddy.projectTabs.v1.prj_1") || "{}")
        .pinned,
    ).toEqual(["b"]),
  );
  expect(readProjectTabMemory("prj_1").pinned).toEqual(["b"]);
  first.unmount();

  renderTabs();
  await screen.findByText("Beta");
  await waitFor(() =>
    expect(
      document
        .querySelectorAll("[data-session-tab]")[0]
        ?.getAttribute("data-session-tab"),
    ).toBe("b"),
  );
});

test("closing the other tabs keeps the active conversation", async () => {
  const view = renderTabs();
  await screen.findByText("Alpha");
  fireEvent.click(screen.getByLabelText("Tab actions for Beta"));
  fireEvent.click(await screen.findByTestId("project-tab-close-others"));
  await waitFor(() => expect(screen.queryByText("Alpha")).toBeNull());
  expect(screen.queryByText("Gamma")).toBeNull();
  expect(screen.getByText("Beta")).toBeTruthy();
  expect(view.onSelectSession).not.toHaveBeenCalled();
});

test("closing left and right tabs follows the menu", async () => {
  renderTabs();
  await screen.findByText("Alpha");
  fireEvent.click(screen.getByLabelText("Tab actions for Beta"));
  fireEvent.click(await screen.findByTestId("project-tab-close-left"));
  await waitFor(() => expect(screen.queryByText("Alpha")).toBeNull());
  fireEvent.click(screen.getByLabelText("Tab actions for Beta"));
  fireEvent.click(await screen.findByTestId("project-tab-close-right"));
  await waitFor(() => expect(screen.queryByText("Gamma")).toBeNull());
  expect(screen.getByText("Beta")).toBeTruthy();
});

test("a stored pin is honoured on the first render", async () => {
  localStorage.setItem(
    "coddy.projectTabs.v1.prj_1",
    JSON.stringify({ order: ["b", "a", "c"], pinned: ["b"], closed: [] }),
  );
  expect(readProjectTabMemory("prj_1").pinned).toEqual(["b"]);
  renderTabs();
  await screen.findByText("Alpha");
  await waitFor(() =>
    expect(
      document
        .querySelectorAll("[data-session-tab]")[0]
        ?.getAttribute("data-session-tab"),
    ).toBe("b"),
  );
});

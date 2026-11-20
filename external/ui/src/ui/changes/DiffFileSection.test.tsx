import { afterEach, expect, test, vi } from "vitest";
import { act, cleanup, fireEvent, render } from "@testing-library/react";
import { StrictMode } from "react";
import * as clipboard from "../clipboard";
import { DiffFileSection } from "./DiffFileSection";
import type { DiffView } from "./diffViewPrefs";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

// A removal followed by an addition, around numbers that grow from two
// digits to three: the rows whose code used to start left of the others.
const PATCH = [
  "--- a/README.md",
  "+++ b/README.md",
  "@@ -98,3 +99,3 @@",
  " keep",
  "-old",
  "+new",
  " tail",
].join("\n");

function renderView(view: DiffView) {
  return render(
    <StrictMode>
      <DiffFileSection
        file={{
          path: "README.md",
          status: "modified",
          additions: 1,
          deletions: 1,
          binary: false,
          truncated: false,
        }}
        patch={PATCH}
        loading={false}
        error=""
        collapsed={false}
        view={view}
        onToggle={() => {}}
        registerRef={() => {}}
      />
    </StrictMode>,
  );
}

// Every number cell takes its width from the file, so the code of a removed
// or an added line (one cell blank) starts where the context's does. Where it
// lands is measured in a real engine by scripts/diff-gutter-check.mjs.
test("every number cell of a diff takes its width from the file's widest number", () => {
  for (const view of ["unified", "split"] as const) {
    const { container, unmount } = renderView(view);
    const diff = container.querySelector<HTMLElement>(".dv-diff");
    expect(diff, view).not.toBeNull();
    // The new column reaches 101 while the old one stops at 100.
    expect(diff!.style.getPropertyValue("--dv-num-digits"), view).toBe("3");
    expect(container.querySelectorAll(".dv-num").length, view).toBeGreaterThan(
      0,
    );
    unmount();
  }
});

test("cancels copy feedback timer on unmount", async () => {
  vi.useFakeTimers();
  vi.spyOn(clipboard, "copyText").mockResolvedValue(true);
  const { getByTestId, unmount } = renderView("unified");
  await act(async () => {
    fireEvent.click(getByTestId("dv-copy-README.md"));
  });
  expect(vi.getTimerCount()).toBe(1);
  unmount();
  expect(vi.getTimerCount()).toBe(0);
});

test("does not schedule feedback when copying finishes after unmount", async () => {
  vi.useFakeTimers();
  let resolve!: (ok: boolean) => void;
  vi.spyOn(clipboard, "copyText").mockReturnValue(
    new Promise((r) => {
      resolve = r;
    }),
  );
  const timeout = vi.spyOn(window, "setTimeout");
  const { getByTestId, unmount } = renderView("unified");
  fireEvent.click(getByTestId("dv-copy-README.md"));
  unmount();
  await act(async () => {
    resolve(true);
  });
  expect(timeout).not.toHaveBeenCalled();
  expect(vi.getTimerCount()).toBe(0);
});

test("repeated copies reset the 1200 ms feedback timer", async () => {
  vi.useFakeTimers();
  vi.spyOn(clipboard, "copyText").mockResolvedValue(true);
  const { getByTestId } = renderView("unified");
  const button = getByTestId("dv-copy-README.md");
  await act(async () => {
    fireEvent.click(button);
  });
  act(() => {
    vi.advanceTimersByTime(800);
  });
  await act(async () => {
    fireEvent.click(button);
  });
  expect(vi.getTimerCount()).toBe(1);
  act(() => {
    vi.advanceTimersByTime(1199);
  });
  expect(button.textContent).toBe("✓");
  act(() => {
    vi.advanceTimersByTime(1);
  });
  expect(button.textContent).toBe("⧉");
});

// A file with two changes far apart, so both gaps and both file ends have
// hidden context the reader can open.
const FAR_APART_BEFORE = Array.from(
  { length: 40 },
  (_, i) => `line ${i + 1}`,
).join("\n");
const FAR_APART_AFTER = [
  "line 1",
  "changed 2",
  ...Array.from({ length: 37 }, (_, i) => `line ${i + 3}`),
].join("\n");
const FAR_APART_PATCH = [
  "--- a/far.txt",
  "+++ b/far.txt",
  "@@ -2,1 +2,1 @@",
  "-line 2",
  "+changed 2",
  "@@ -30,1 +30,1 @@",
  "-line 30",
  "+line 30 changed",
].join("\n");

function renderExpandable() {
  return render(
    <DiffFileSection
      file={{
        path: "far.txt",
        status: "modified",
        additions: 1,
        deletions: 1,
        binary: false,
        truncated: false,
        before: FAR_APART_BEFORE,
        after: FAR_APART_AFTER,
      }}
      patch={FAR_APART_PATCH}
      loading={false}
      error=""
      collapsed={false}
      view="unified"
      onToggle={() => {}}
      registerRef={() => {}}
    />,
  );
}

test("hidden context opens ten lines at a time and closes again", () => {
  const { container, getAllByLabelText, getByTestId } = renderExpandable();
  const count = () => container.querySelectorAll(".dv-line--ctx").length;
  // The patch carries no context lines of its own, so every unchanged line on
  // screen was opened by the reader.
  expect(count()).toBe(0);

  // The gap between the two hunks opens ten lines each way at once.
  fireEvent.click(getAllByLabelText("Show 10 lines in both directions")[0]!);
  expect(count()).toBe(20);

  // One control at the head opens every gap, and the same control puts the
  // file back as git wrote it.
  const toggle = getByTestId("dv-expand-lines-far.txt");
  expect(toggle.getAttribute("aria-pressed")).toBe("false");
  fireEvent.click(toggle);
  expect(toggle.getAttribute("aria-pressed")).toBe("true");
  expect(count()).toBeGreaterThan(20);

  fireEvent.click(toggle);
  expect(toggle.getAttribute("aria-pressed")).toBe("false");
  expect(count()).toBe(0);
});

test("a file without both sides offers no context controls", () => {
  const { queryAllByLabelText, queryByTestId } = renderView("unified");
  expect(queryByTestId("dv-expand-lines-README.md")).toBeNull();
  expect(queryAllByLabelText("Show 10 lines above")).toHaveLength(0);
});

test("a Markdown change offers the rich diff beside its source", () => {
  const { getByLabelText, container } = render(
    <DiffFileSection
      file={{
        path: "README.md",
        status: "modified",
        additions: 1,
        deletions: 1,
        binary: false,
        truncated: false,
        before: "# Title\n\nold\n",
        after: "# Title\n\nnew\n",
      }}
      patch={PATCH}
      loading={false}
      error=""
      collapsed={false}
      view="unified"
      onToggle={() => {}}
      registerRef={() => {}}
    />,
  );

  // The source diff is the default view; the rich one renders the Markdown.
  expect(container.querySelector(".dv-diff")).not.toBeNull();
  fireEvent.click(getByLabelText("Display the rich diff"));
  expect(container.querySelector(".dv-diff")).toBeNull();
  expect(container.querySelector(".dv-rich--markdown")).not.toBeNull();

  fireEvent.click(getByLabelText("Display the source diff"));
  expect(container.querySelector(".dv-rich")).toBeNull();
  expect(container.querySelector(".dv-diff")).not.toBeNull();
});

test("a plain text change offers no rich view", () => {
  const { queryByLabelText } = renderView("unified");
  expect(queryByLabelText("Display the rich diff")).toBeNull();
});

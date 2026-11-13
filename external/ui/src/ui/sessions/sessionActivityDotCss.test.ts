/**
 * Contract: a session whose turn is running carries a loading icon - a ring
 * spinner in the theme's accent - so a running conversation reads as working.
 * The unread dot a finished background turn leaves behind stays a dot, so
 * "working" and "done, not read yet" still read apart.
 */
import { expect, test } from "vitest";
import { stylesSource } from "../../stylesSource";

const css = stylesSource();

function rule(selector: string): string {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return (
    new RegExp(`(^|\\n)${escaped}\\s*\\{[^}]*\\}`, "s").exec(css)?.[0] ?? ""
  );
}

test("the running turn carries a ring spinner, not the unread dot's fill", () => {
  const spinner = rule(".session-activity-spinner");
  const unread = rule(".session-unread-dot");
  expect(spinner).not.toBe("");
  expect(spinner).toMatch(/animation:\s*session-spin\s/);
  // A ring: transparent sides, one accent segment, and no solid fill.
  expect(spinner).not.toMatch(/background:\s*var\(--accent\);/);
  expect(unread).toMatch(/background:\s*var\(--accent\);/);
});

test("the activity spinner holds still for reduced motion", () => {
  expect(css).toMatch(/@keyframes session-spin\s*\{/);
  expect(css).toMatch(
    /@media \(prefers-reduced-motion: reduce\)\s*\{\s*\.session-activity-spinner\s*\{\s*animation:\s*none;/,
  );
  expect(css).not.toMatch(/@keyframes session-activity-pulse\s*\{/);
});

test("a question marker stays still while it waits for the user", () => {
  expect(rule(".session-question-icon")).not.toMatch(/animation:/);
  expect(css).not.toMatch(/@keyframes session-question-pulse\s*\{/);
});

test("finished and error dots keep the shared circle geometry", () => {
  const idle = rule(".session-idle-dot");
  const error = rule(".session-error-dot");
  const unread = rule(".session-unread-dot");
  for (const dot of [idle, error, unread]) {
    expect(dot).not.toBe("");
    for (const prop of ["width", "height", "border-radius"]) {
      const re = new RegExp(`${prop}:\\s*([^;]+);`);
      expect(re.exec(dot)?.[1]).toBe(re.exec(unread)?.[1]);
    }
  }
  expect(idle).toMatch(/background:\s*transparent;/);
  expect(error).toMatch(/background:\s*var\(--coddy-danger\);/);
  expect(rule(".session-error-dot\.is-seen")).toMatch(
    /background:\s*transparent;/,
  );
  expect(rule(".session-error-dot\.is-seen")).toMatch(/box-shadow:\s*inset/);
});

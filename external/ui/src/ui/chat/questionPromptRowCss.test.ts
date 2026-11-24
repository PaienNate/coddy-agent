import { expect, test } from "vitest";
import { stylesSource } from "../../stylesSource";

function ruleBodies(css: string, selector: string): string[] {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const re = new RegExp(`(?:^|\\n)\\s*${escaped}\\s*\\{([^}]*)\\}`, "g");
  return [...css.matchAll(re)].map((m) => m[1] ?? "");
}

// The row's visually hidden radio/checkbox is `position: absolute` (`.sr-only`).
// Without a positioned ancestor its containing block is the initial one, so the
// input escaped the transcript's scroller and the body's `overflow: hidden`: its
// static position sat thousands of pixels down the scrolled content, which
// inflated the document, and focusing it (a click on the row or its letter
// bubble) scrolled the whole window into that blank tail - the page went white
// while every element stayed in the DOM.
test("the question row is the containing block of its hidden input", () => {
  const css = stylesSource();
  const bodies = ruleBodies(css, "label.question-prompt-row");
  expect(bodies.length).toBeGreaterThan(0);
  expect(bodies.some((body) => /position:\s*relative/.test(body))).toBe(true);
});

// The question card is a prompt in the transcript like the permission card, and
// both rows are flush with the transcript column: on a phone the card's edges meet
// the composer's under it on both sides. The question row used to carry a
// padding-left of its own and no padding on the right, which set the card in from
// the left edge only.
test("the question row is flush with the column on both sides, like the permission row", () => {
  const css = stylesSource();
  for (const selector of [".message-row-question", ".message-row-permission"]) {
    const bodies = ruleBodies(css, selector);
    expect(bodies.length, selector).toBeGreaterThan(0);
    for (const body of bodies) {
      expect(body, selector).not.toMatch(
        /padding(-left|-right|-inline[a-z-]*)?\s*:/,
      );
      expect(body, selector).not.toMatch(
        /margin(-left|-right|-inline[a-z-]*)\s*:/,
      );
      const margin = body.match(/(?:^|;|\s)margin\s*:\s*([^;]+);/);
      if (margin) {
        // margin: <top> 0 <bottom> - the sides stay at zero.
        const parts = margin[1]!.trim().split(/\s+/);
        expect(parts[1] ?? parts[0], selector).toBe("0");
      }
    }
  }
});

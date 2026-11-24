import { describe, expect, test } from "vitest";

import { viewedRevision } from "./viewedRevision";
import type { ChangedFile } from "./types";

const base: ChangedFile = {
  path: "src/a.ts",
  status: "modified",
  additions: 1,
  deletions: 1,
  binary: false,
  truncated: false,
};

describe("the revision a Viewed mark binds to", () => {
  test("is the server's digest when there is one", () => {
    expect(viewedRevision({ ...base, digest: "abc" })).toBe("abc");
  });

  test("falls back to the line counts, never to the patch", () => {
    // An older server sends no digest at all.
    const beforePatch = viewedRevision(base);
    const afterPatch = viewedRevision({ ...base, patch: "-a\n+b\n" });

    expect(beforePatch).toBe(afterPatch);
    expect(beforePatch).toBe("modified:1:1");
  });

  test("a rewrite under equal counts still moves it, through the digest", () => {
    expect(viewedRevision({ ...base, digest: "one" })).not.toBe(
      viewedRevision({ ...base, digest: "two" }),
    );
  });
});

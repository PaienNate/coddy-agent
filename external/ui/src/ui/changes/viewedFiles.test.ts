import { beforeEach, describe, expect, test } from "vitest";

import {
  VIEWED_FILES_STORAGE_KEY,
  isFileViewed,
  setFileViewed,
} from "./viewedFiles";

beforeEach(() => {
  window.localStorage.clear();
});

describe("viewed files", () => {
  test("a mark holds for the digest it was taken against", () => {
    setFileViewed("/repo", "src/a.ts", "digest-1", true);

    expect(isFileViewed("/repo", "src/a.ts", "digest-1")).toBe(true);
  });

  test("a rewrite of the same path is not viewed", () => {
    setFileViewed("/repo", "src/a.ts", "digest-1", true);

    expect(isFileViewed("/repo", "src/a.ts", "digest-2")).toBe(false);
  });

  test("unmarking forgets the file", () => {
    setFileViewed("/repo", "src/a.ts", "digest-1", true);
    setFileViewed("/repo", "src/a.ts", "digest-1", false);

    expect(isFileViewed("/repo", "src/a.ts", "digest-1")).toBe(false);
  });

  test("one workspace does not mark another", () => {
    setFileViewed("/repo", "src/a.ts", "digest-1", true);

    expect(isFileViewed("/other", "src/a.ts", "digest-1")).toBe(false);
  });

  test("an empty argument marks nothing", () => {
    setFileViewed("", "src/a.ts", "digest-1", true);
    setFileViewed("/repo", "", "digest-1", true);
    setFileViewed("/repo", "src/a.ts", "", true);

    expect(window.localStorage.getItem(VIEWED_FILES_STORAGE_KEY)).toBeNull();
    expect(isFileViewed("", "src/a.ts", "digest-1")).toBe(false);
    expect(isFileViewed("/repo", "src/a.ts", "")).toBe(false);
  });

  test("a corrupt store reads as nothing marked", () => {
    window.localStorage.setItem(VIEWED_FILES_STORAGE_KEY, "{not json");

    expect(isFileViewed("/repo", "src/a.ts", "digest-1")).toBe(false);
    // A write over it starts a fresh list rather than throwing.
    setFileViewed("/repo", "src/a.ts", "digest-1", true);
    expect(isFileViewed("/repo", "src/a.ts", "digest-1")).toBe(true);
  });
});

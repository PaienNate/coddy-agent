import type { ChangedFile } from "./types";

/**
 * What a Viewed mark is bound to. The server's digest is the answer whenever
 * there is one; without it (an older server) the line counts stand in, and the
 * patch never does: it arrives later than the mark, can be hundreds of
 * kilobytes, and would un-press a mark the reader already made.
 */
export function viewedRevision(file: ChangedFile): string {
  return file.digest ?? `${file.status}:${file.additions}:${file.deletions}`;
}

/**
 * Whether the desktop History sidebar is shown. It is on by default, the way a
 * chat app's conversation list is; the close control in its head hides it and
 * the rail's History button brings it back.
 */
const KEY = "coddy.historyDock.v1";

export function readHistoryDockPreference(): boolean {
  try {
    return localStorage.getItem(KEY) !== "0";
  } catch {
    return true;
  }
}

export function writeHistoryDockPreference(open: boolean): void {
  try {
    localStorage.setItem(KEY, open ? "1" : "0");
  } catch {
    // A private window may refuse storage; the in-memory state still works.
  }
}

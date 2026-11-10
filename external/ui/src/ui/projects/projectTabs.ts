import type { SessionRow } from "../sessions/types";

/** What this browser remembers about one project's tab strip. */
export type ProjectTabMemory = {
  /** Every tab in the operator's order, pinned ones included. */
  order: string[];
  /** Tabs held at the front of the strip. */
  pinned: string[];
  /** Tabs the operator closed. They stay closed until reopened. */
  closed: string[];
};

const EMPTY: ProjectTabMemory = { order: [], pinned: [], closed: [] };
const KEY_PREFIX = "coddy.projectTabs.v1.";

function uniqueStrings(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  const out: string[] = [];
  const seen = new Set<string>();
  for (const raw of value) {
    const id = typeof raw === "string" ? raw.trim() : "";
    if (!id || seen.has(id)) {
      continue;
    }
    seen.add(id);
    out.push(id);
  }
  return out;
}

export function readProjectTabMemory(projectId: string): ProjectTabMemory {
  const key = (projectId || "").trim();
  if (!key) {
    return { ...EMPTY };
  }
  try {
    const raw = localStorage.getItem(KEY_PREFIX + key);
    if (!raw) {
      return { ...EMPTY };
    }
    const parsed = JSON.parse(raw) as Partial<ProjectTabMemory> | null;
    return {
      order: uniqueStrings(parsed?.order),
      pinned: uniqueStrings(parsed?.pinned),
      closed: uniqueStrings(parsed?.closed),
    };
  } catch {
    return { ...EMPTY };
  }
}

export function writeProjectTabMemory(
  projectId: string,
  memory: ProjectTabMemory,
): void {
  const key = (projectId || "").trim();
  if (!key) {
    return;
  }
  try {
    localStorage.setItem(
      KEY_PREFIX + key,
      JSON.stringify({
        order: uniqueStrings(memory.order),
        pinned: uniqueStrings(memory.pinned),
        closed: uniqueStrings(memory.closed),
      }),
    );
  } catch {
    // A private window may refuse storage; the strip still works this session.
  }
}

/**
 * Adds sessions the strip has not seen. A new session goes to the front of the
 * remembered order - it is the newest conversation - while the operator's
 * order of old tabs is left alone. Ids the listing no longer carries stay
 * remembered: the listing loads asynchronously, and pruning before its first
 * page arrives would forget every pin and dragged position.
 */
export function reconcileTabMemory(
  rows: readonly SessionRow[],
  memory: ProjectTabMemory,
): ProjectTabMemory {
  const ordered = new Set(memory.order);
  const fresh = rows.filter((row) => !ordered.has(row.id)).map((row) => row.id);
  if (fresh.length === 0) {
    return memory;
  }
  return {
    ...memory,
    order: [...fresh, ...memory.order],
  };
}

/** The tabs in display order: pinned first, then the remembered order. */
export function orderedTabs(
  rows: readonly SessionRow[],
  memory: ProjectTabMemory,
): SessionRow[] {
  const pinned = new Set(memory.pinned);
  const rank = new Map(memory.order.map((id, i) => [id, i]));
  return [...rows].sort((a, b) => {
    const pa = pinned.has(a.id) ? 0 : 1;
    const pb = pinned.has(b.id) ? 0 : 1;
    if (pa !== pb) {
      return pa - pb;
    }
    const ra = rank.get(a.id) ?? Number.MAX_SAFE_INTEGER;
    const rb = rank.get(b.id) ?? Number.MAX_SAFE_INTEGER;
    if (ra !== rb) {
      return ra - rb;
    }
    return (b.updatedAt || "").localeCompare(a.updatedAt || "");
  });
}

export function visibleTabs(
  rows: readonly SessionRow[],
  memory: ProjectTabMemory,
): SessionRow[] {
  const closed = new Set(memory.closed);
  return orderedTabs(rows, memory).filter((row) => !closed.has(row.id));
}

/** Makes a session visible again (a History pick, a new chat, a reload). */
export function openTab(
  memory: ProjectTabMemory,
  id: string,
): ProjectTabMemory {
  const tab = (id || "").trim();
  if (!tab) {
    return memory;
  }
  return {
    ...memory,
    order: memory.order.includes(tab) ? memory.order : [tab, ...memory.order],
    closed: memory.closed.filter((x) => x !== tab),
  };
}

/** Pins a tab to the front, or lets it back into order where it stands. */
export function togglePinned(
  memory: ProjectTabMemory,
  id: string,
): ProjectTabMemory {
  const tab = (id || "").trim();
  if (!tab) {
    return memory;
  }
  const pinned = new Set(memory.pinned);
  if (pinned.has(tab)) {
    pinned.delete(tab);
    return { ...memory, pinned: [...pinned] };
  }
  pinned.add(tab);
  return {
    ...memory,
    pinned: [...pinned],
    order: [tab, ...memory.order.filter((x) => x !== tab)],
  };
}

export function closeTabs(
  memory: ProjectTabMemory,
  ids: readonly string[],
): ProjectTabMemory {
  const closed = new Set(memory.closed);
  for (const raw of ids) {
    const id = (raw || "").trim();
    if (id) {
      closed.add(id);
    }
  }
  return { ...memory, closed: [...closed] };
}

export function closeOthers(
  memory: ProjectTabMemory,
  keepId: string,
  allIds: readonly string[],
): ProjectTabMemory {
  const keep = (keepId || "").trim();
  const pinned = new Set(memory.pinned);
  return closeTabs(
    memory,
    allIds.filter((id) => id !== keep && !pinned.has(id)),
  );
}

export function closeLeft(
  memory: ProjectTabMemory,
  id: string,
  visibleIds: readonly string[],
): ProjectTabMemory {
  const index = visibleIds.indexOf((id || "").trim());
  if (index <= 0) {
    return memory;
  }
  const pinned = new Set(memory.pinned);
  return closeTabs(
    memory,
    visibleIds.slice(0, index).filter((x) => !pinned.has(x)),
  );
}

export function closeRight(
  memory: ProjectTabMemory,
  id: string,
  visibleIds: readonly string[],
): ProjectTabMemory {
  const index = visibleIds.indexOf((id || "").trim());
  if (index < 0 || index >= visibleIds.length - 1) {
    return memory;
  }
  const pinned = new Set(memory.pinned);
  return closeTabs(
    memory,
    visibleIds.slice(index + 1).filter((x) => !pinned.has(x)),
  );
}

/**
 * Reorders the visible strip after a drop. Hidden tabs keep their remembered
 * place at the end; the next reconcile puts any session the server has not
 * listed yet back at the front.
 */
export function reorderTabs(
  memory: ProjectTabMemory,
  visibleIds: readonly string[],
  fromId: string,
  toId: string,
): ProjectTabMemory {
  const from = (fromId || "").trim();
  const to = (toId || "").trim();
  const next = [...visibleIds];
  const src = next.indexOf(from);
  const dst = next.indexOf(to);
  if (src < 0 || dst < 0 || src === dst) {
    return memory;
  }
  next.splice(src, 1);
  next.splice(dst, 0, from);
  const shown = new Set(next);
  return {
    ...memory,
    order: [...next, ...memory.order.filter((id) => !shown.has(id))],
  };
}

/**
 * Which tab takes the screen after closing `closingIds`, or "" when none is
 * left. The next tab to the right wins; the last one to the left is the
 * fallback, so closing the last tab goes back to the project home.
 */
export function nextActiveAfterClose(
  visibleIds: readonly string[],
  closingIds: readonly string[],
  activeId: string,
): string {
  const closing = new Set(closingIds);
  const active = (activeId || "").trim();
  if (!closing.has(active)) {
    return active;
  }
  const index = visibleIds.indexOf(active);
  if (index < 0) {
    return "";
  }
  for (let i = index + 1; i < visibleIds.length; i++) {
    const id = visibleIds[i];
    if (id && !closing.has(id)) {
      return id;
    }
  }
  for (let i = index - 1; i >= 0; i--) {
    const id = visibleIds[i];
    if (id && !closing.has(id)) {
      return id;
    }
  }
  return "";
}

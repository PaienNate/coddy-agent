/** One row of GET /coddy/projects. */
export type Project = {
  id: string;
  name: string;
  path: string;
  color?: string;
  pinned?: boolean;
  order: number;
  createdAt: string;
  updatedAt: string;
};

const ACTIVE_PROJECT_KEY = "coddy.projects.active.v1";
const LAST_OPENED_KEY = "coddy.projects.lastOpened.v1";

/** Paths compare the way the server's workspace filter does, as far as a
 *  browser can tell: separators, trailing slashes and a Windows drive letter
 *  are spelling, not identity. Symlinks stay distinct here; the session row's
 *  repoRoot is the main checkout, which is the path a project is registered
 *  with. */
export function projectPathKey(path: string): string {
  let p = (path || "").trim();
  if (!p) {
    return "";
  }
  p = p.replace(/[\\/]+/g, "/").replace(/\/+$/, "");
  if (!p) {
    return "/";
  }
  if (/^[a-zA-Z]:\//.test(p) || (path || "").includes("\\")) {
    // A Windows drive or UNC path: the filesystem folds case.
    p = p.toLowerCase();
  }
  return p;
}

export function sameProjectPath(a: string, b: string): boolean {
  const ka = projectPathKey(a);
  return ka !== "" && ka === projectPathKey(b);
}

export function projectPathName(path: string): string {
  const p = (path || "").replace(/[\\/]+$/, "");
  const cut = Math.max(p.lastIndexOf("/"), p.lastIndexOf("\\"));
  return (cut >= 0 ? p.slice(cut + 1) : p) || path;
}

export function readActiveProjectId(): string {
  try {
    return (localStorage.getItem(ACTIVE_PROJECT_KEY) || "").trim();
  } catch {
    return "";
  }
}

export function writeActiveProjectId(id: string): void {
  try {
    const next = (id || "").trim();
    if (next) {
      localStorage.setItem(ACTIVE_PROJECT_KEY, next);
    } else {
      localStorage.removeItem(ACTIVE_PROJECT_KEY);
    }
  } catch {
    // A private window may refuse storage; the in-memory state still works.
  }
}

/** Project id -> the last instant this browser opened that project. */
export type ProjectLastOpened = Record<string, string>;

export function readProjectLastOpened(): ProjectLastOpened {
  try {
    const raw = localStorage.getItem(LAST_OPENED_KEY);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    const out: ProjectLastOpened = {};
    for (const [id, at] of Object.entries(parsed)) {
      if (
        id.trim() &&
        typeof at === "string" &&
        Number.isFinite(Date.parse(at))
      ) {
        out[id] = at;
      }
    }
    return out;
  } catch {
    return {};
  }
}

/** Records a project open and returns the whole map for React state. */
export function markProjectOpened(
  id: string,
  at: string = new Date().toISOString(),
): ProjectLastOpened {
  const next = { ...readProjectLastOpened() };
  if (id.trim()) {
    next[id] = at;
    try {
      localStorage.setItem(LAST_OPENED_KEY, JSON.stringify(next));
    } catch {
      // A private window may refuse storage; the card still has its fallback.
    }
  }
  return next;
}

import type { SessionRow } from "../sessions/types";
import { sameProjectPath, type Project } from "./types";

async function projectRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  const data = (await res.json().catch(() => null)) as
    | (T & { error?: { message?: string } | string })
    | null;
  if (!res.ok) {
    const error = data?.error;
    const message =
      typeof error === "string"
        ? error
        : error?.message || `Request failed (${res.status})`;
    throw new Error(message);
  }
  return data as T;
}

export async function listProjects(): Promise<Project[]> {
  const data = await projectRequest<{ projects?: Project[] }>(
    "/coddy/projects",
  );
  return Array.isArray(data?.projects) ? data.projects : [];
}

export async function createProject(
  path: string,
  name?: string,
): Promise<Project> {
  const data = await projectRequest<{ project: Project }>("/coddy/projects", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, name: name || "" }),
  });
  return data.project;
}

export async function updateProject(
  id: string,
  patch: Partial<Pick<Project, "name" | "path" | "color" | "pinned" | "order">>,
): Promise<Project> {
  const data = await projectRequest<{ project: Project }>(
    `/coddy/projects/${encodeURIComponent(id)}`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(patch),
    },
  );
  return data.project;
}

export async function deleteProject(id: string): Promise<void> {
  await projectRequest<{ ok?: boolean }>(
    `/coddy/projects/${encodeURIComponent(id)}`,
    { method: "DELETE" },
  );
}

/**
 * The sessions of a project workspace. A project is a folder; a worktree has
 * its own cwd, so the caller can pass those too. The per-folder queries find
 * old conversations in the project itself; one recent global page adds its
 * worktrees, whose stored cwd names the linked checkout but whose repoRoot
 * names the project. Responses are merged by id, newest first.
 */
export async function listWorkspaceSessions(
  paths: readonly string[],
): Promise<SessionRow[]> {
  const unique = [...new Set(paths.map((p) => p.trim()).filter(Boolean))];
  if (unique.length === 0) {
    return [];
  }
  const page = async (params: URLSearchParams): Promise<SessionRow[]> => {
    const res = await fetch(`/coddy/sessions?${params.toString()}`);
    if (!res.ok) {
      return [];
    }
    const data = (await res.json().catch(() => null)) as {
      sessions?: SessionRow[];
    } | null;
    return Array.isArray(data?.sessions) ? data.sessions : [];
  };
  const recentParams = new URLSearchParams({
    limit: "100",
    sort: "updated",
    order: "desc",
    include_activity: "true",
  });
  const [pages, recent] = await Promise.all([
    Promise.all(
      unique.map((path) =>
        page(
          new URLSearchParams({
            cwd: path,
            limit: "100",
            sort: "updated",
            order: "desc",
            include_activity: "true",
          }),
        ),
      ),
    ),
    page(recentParams),
  ]);
  const belongsToProject = (row: SessionRow) => {
    const stored = (row.repoRoot || row.cwd || "").trim();
    return unique.some((path) => sameProjectPath(stored, path));
  };
  const byId = new Map<string, SessionRow>();
  for (const rows of [...pages, recent.filter(belongsToProject)]) {
    for (const row of rows) {
      if (!row?.id || byId.has(row.id)) {
        continue;
      }
      byId.set(row.id, row);
    }
  }
  return [...byId.values()].sort((a, b) => {
    const ta = Date.parse(a.updatedAt || "") || 0;
    const tb = Date.parse(b.updatedAt || "") || 0;
    return tb - ta;
  });
}

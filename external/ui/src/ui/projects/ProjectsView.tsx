import { useEffect, useMemo, useState } from "react";
import { useT } from "../i18n/I18nProvider";
import { useConfirm } from "../components/useConfirm";
import { WorkspaceFolderModal } from "../chat/WorkspaceFolderModal";
import type { SessionRow } from "../sessions/types";
import {
  projectPathKey,
  projectPathName,
  sameProjectPath,
  type Project,
} from "./types";

type Props = {
  projects: readonly Project[];
  loading: boolean;
  error: string;
  onClose: () => void;
  onOpenProject: (project: Project) => void;
  onCreateProject: (path: string) => Promise<Project>;
  onDeleteProject: (project: Project) => Promise<void>;
  onRefresh: () => void;
};

type ProjectCount = { count: number; updatedAt: string; path: string };

function IconFolderLarge() {
  return (
    <svg
      width="22"
      height="22"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <path d="M3.5 6.5h5.2l1.8 2h9.9v9.2a1.8 1.8 0 0 1-1.8 1.8H5.3a1.8 1.8 0 0 1-1.8-1.8V6.5Z" />
    </svg>
  );
}

function IconPlus() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      aria-hidden
    >
      <path d="M12 5v14M5 12h14" />
    </svg>
  );
}

function IconTrash() {
  return (
    <svg
      width="15"
      height="15"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <path d="M4 7h16" />
      <path d="M9 7V4.8h6V7" />
      <path d="M6.5 7l.8 12.2h9.4L17.5 7" />
      <path d="M10 11v5M14 11v5" />
    </svg>
  );
}

function IconOpen() {
  return (
    <svg
      width="15"
      height="15"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.9"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <path d="M5 12h14" />
      <path d="m13 6 6 6-6 6" />
    </svg>
  );
}

async function fetchSessionCounts(): Promise<Map<string, ProjectCount>> {
  const out = new Map<string, ProjectCount>();
  try {
    const q = new URLSearchParams({
      limit: "100",
      sort: "updated",
      order: "desc",
    });
    const res = await fetch(`/coddy/sessions?${q.toString()}`);
    if (!res.ok) {
      return out;
    }
    const data = (await res.json()) as { sessions?: SessionRow[] };
    for (const row of data.sessions || []) {
      const path = (row.repoRoot || row.cwd || "").trim();
      const key = projectPathKey(path);
      if (!key) continue;
      const updated = row.updatedAt || "";
      const cur = out.get(key);
      if (!cur) {
        out.set(key, { count: 1, updatedAt: updated, path });
      } else {
        cur.count += 1;
        if (updated > cur.updatedAt) cur.updatedAt = updated;
      }
    }
  } catch {
    // A project page still works without the counts.
  }
  return out;
}

function countFor(
  counts: Map<string, ProjectCount>,
  project: Project,
): ProjectCount | undefined {
  return (
    counts.get(projectPathKey(project.path)) ||
    [...counts.entries()].find(([key]) =>
      sameProjectPath(key, project.path),
    )?.[1]
  );
}

/**
 * The Projects page: the folders this Coddy home knows, each opening its own
 * workspace and tab strip. Adding a project registers a folder; it never
 * copies or moves the conversations already stored for it.
 */
export function ProjectsView(props: Props) {
  const { t, tp } = useT();
  const confirm = useConfirm();
  const [folderOpen, setFolderOpen] = useState(false);
  const [busy, setBusy] = useState("");
  const [localError, setLocalError] = useState("");
  const [counts, setCounts] = useState<Map<string, ProjectCount>>(new Map());

  useEffect(() => {
    let live = true;
    void fetchSessionCounts().then((next) => {
      if (live) setCounts(next);
    });
    return () => {
      live = false;
    };
  }, [props.projects.length]);

  const sorted = useMemo(
    () =>
      [...props.projects].sort((a, b) => {
        if (!!a.pinned !== !!b.pinned) return a.pinned ? -1 : 1;
        if (a.order !== b.order) return a.order - b.order;
        return a.name.localeCompare(b.name);
      }),
    [props.projects],
  );

  // Workspaces History already holds but no project names yet. They are shown
  // as a discovery strip rather than written into the registry behind the
  // operator's back: one click makes one a project.
  const discovered = useMemo(() => {
    const rows = [...counts.values()].filter(
      (row) =>
        row.path &&
        !props.projects.some((project) =>
          sameProjectPath(project.path, row.path),
        ),
    );
    rows.sort(
      (a, b) => b.count - a.count || b.updatedAt.localeCompare(a.updatedAt),
    );
    return rows;
  }, [counts, props.projects]);

  const addFolder = async (path: string) => {
    setBusy(path);
    setLocalError("");
    try {
      await props.onCreateProject(path);
      setFolderOpen(false);
      props.onRefresh();
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy("");
    }
  };

  const remove = async (project: Project) => {
    const ok = await confirm({
      title: t("projects.delete.title", { name: project.name }),
      message: t("projects.delete.message", { name: project.name }),
      confirmLabel: t("common.delete"),
      variant: "danger",
    });
    if (!ok) return;
    setBusy(project.id);
    setLocalError("");
    try {
      await props.onDeleteProject(project);
      props.onRefresh();
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy("");
    }
  };

  return (
    <section
      className="projects-view"
      aria-label={t("projects.title")}
      data-testid="projects-view"
    >
      <header className="projects-head">
        <div className="projects-head-copy">
          <h1>{t("projects.title")}</h1>
          <p>{t("projects.subtitle")}</p>
        </div>
        <div className="projects-head-actions">
          <button
            type="button"
            className="projects-add"
            onClick={() => setFolderOpen(true)}
            disabled={busy !== ""}
          >
            <IconPlus />
            <span>{t("projects.add")}</span>
          </button>
          <button
            type="button"
            className="projects-close"
            aria-label={t("projects.close")}
            onClick={props.onClose}
          >
            ×
          </button>
        </div>
      </header>

      {localError || props.error ? (
        <div className="projects-error" role="alert">
          {localError || props.error}
        </div>
      ) : null}

      {props.loading && sorted.length === 0 && discovered.length === 0 ? (
        <div className="projects-empty">{t("projects.loading")}</div>
      ) : null}
      {!props.loading && sorted.length === 0 && discovered.length === 0 ? (
        <div className="projects-empty projects-empty--first">
          <span className="projects-empty-icon">
            <IconFolderLarge />
          </span>
          <strong>{t("projects.empty.title")}</strong>
          <span>{t("projects.empty.body")}</span>
          <button
            type="button"
            className="projects-add"
            onClick={() => setFolderOpen(true)}
          >
            <IconPlus />
            <span>{t("projects.add")}</span>
          </button>
        </div>
      ) : null}
      {sorted.length > 0 ? (
        <div className="projects-grid">
          {sorted.map((project) => {
            const stat = countFor(counts, project);
            return (
              <div
                key={project.id}
                className="projects-card"
                role="button"
                tabIndex={0}
                data-testid="project-card"
                onClick={() => props.onOpenProject(project)}
                onKeyDown={(ev) => {
                  if (ev.target !== ev.currentTarget) return;
                  if (ev.key === "Enter" || ev.key === " ") {
                    ev.preventDefault();
                    props.onOpenProject(project);
                  }
                }}
              >
                <div className="projects-card-top">
                  <span className="projects-card-icon">
                    <IconFolderLarge />
                  </span>
                  <div className="projects-card-title">
                    <strong>{project.name}</strong>
                    <span title={project.path}>{project.path}</span>
                  </div>
                  <button
                    type="button"
                    className="projects-card-delete"
                    aria-label={t("projects.delete.aria", {
                      name: project.name,
                    })}
                    title={t("projects.delete.aria", { name: project.name })}
                    disabled={busy === project.id}
                    onClick={(ev) => {
                      ev.stopPropagation();
                      void remove(project);
                    }}
                  >
                    <IconTrash />
                  </button>
                </div>
                <div className="projects-card-foot">
                  <span>{tp("projects.card.sessions", stat?.count ?? 0)}</span>
                  <span className="projects-card-open">
                    {t("projects.card.open")}
                    <IconOpen />
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      ) : null}
      {discovered.length > 0 ? (
        <section className="projects-discovered">
          <div className="projects-subhead">
            <h2>{t("projects.discovered.title")}</h2>
            <p>{t("projects.discovered.body")}</p>
          </div>
          <div className="projects-grid">
            {discovered.map((row) => {
              const name = projectPathName(row.path) || row.path;
              return (
                <div
                  key={row.path}
                  className="projects-card projects-card--discovered"
                >
                  <div className="projects-card-top">
                    <span className="projects-card-icon">
                      <IconFolderLarge />
                    </span>
                    <div className="projects-card-title">
                      <strong>{name}</strong>
                      <span title={row.path}>{row.path}</span>
                    </div>
                    <button
                      type="button"
                      className="projects-card-add"
                      aria-label={t("projects.discovered.addAria", { name })}
                      title={t("projects.discovered.addAria", { name })}
                      disabled={busy === row.path}
                      onClick={() => void addFolder(row.path)}
                    >
                      <IconPlus />
                    </button>
                  </div>
                  <div className="projects-card-foot">
                    <span>{tp("projects.card.sessions", row.count)}</span>
                    <span className="projects-card-discovered">
                      {t("projects.discovered.add")}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </section>
      ) : null}
      <WorkspaceFolderModal
        open={folderOpen}
        startPath=""
        onClose={() => setFolderOpen(false)}
        onPick={(path) => void addFolder(path)}
      />
    </section>
  );
}

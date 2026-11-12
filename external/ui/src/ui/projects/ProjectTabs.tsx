import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type DragEvent,
  type MouseEvent,
} from "react";
import { useT } from "../i18n/I18nProvider";
import {
  SessionRowMenu,
  type SessionRowMenuItem,
} from "../sessions/SessionRowMenu";
import type { SessionRow } from "../sessions/types";
import { listWorkspaceSessions } from "./api";
import {
  closeLeft,
  closeOthers,
  closeRight,
  closeTabs,
  nextActiveAfterClose,
  openTab,
  readProjectTabMemory,
  reconcileTabMemory,
  reorderTabs,
  togglePinned,
  visibleTabs,
  writeProjectTabMemory,
  type ProjectTabMemory,
} from "./projectTabMemory";
import type { Project } from "./types";

type Props = {
  project: Project;
  projects: readonly Project[];
  activeSessionId: string;
  activeSessionTitle?: string;
  /** The cwd of the session on screen when it is a worktree of the project. */
  workspacePath?: string;
  onSelectSession: (id: string) => void;
  onNewChat: (project: Project) => void;
  onSwitchProject: (project: Project) => void;
  onOpenProjects: () => void;
  /** Back to the project's start screen when closing its last tab. */
  onHome: (project: Project) => void;
  /** Bumped by the shell when a new session may have appeared. */
  refreshEpoch?: number;
  /** A child transcript is on screen under its parent tab: selecting that
   *  already-active tab still has to return to the parent chat. */
  reselectActive?: boolean;
};

function IconFolder() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <path d="M3.5 6.5h5.2l1.8 2h9.9v9.2a1.8 1.8 0 0 1-1.8 1.8H5.3a1.8 1.8 0 0 1-1.8-1.8V6.5Z" />
    </svg>
  );
}

function IconPin(props: { filled?: boolean }) {
  return (
    <svg
      width="13"
      height="13"
      viewBox="0 0 24 24"
      fill={props.filled ? "currentColor" : "none"}
      stroke="currentColor"
      strokeWidth="1.9"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <path d="M9 3h6l-.8 5.2 3.3 3.3H6.5l3.3-3.3L9 3Z" />
      <path d="M12 11.5V21" />
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

function sameMemory(a: ProjectTabMemory, b: ProjectTabMemory): boolean {
  return (
    a.order.length === b.order.length &&
    a.pinned.length === b.pinned.length &&
    a.closed.length === b.closed.length &&
    a.order.every((x, i) => x === b.order[i]) &&
    a.pinned.every((x, i) => x === b.pinned[i]) &&
    a.closed.every((x, i) => x === b.closed[i])
  );
}

function tabTitle(row: SessionRow, untitled: string): string {
  return (row.title || "").trim() || untitled;
}

/**
 * The tab strip of a project workspace. It is the project's own history: every
 * conversation in the folder is a tab, the operator's order is kept in this
 * browser, and closing a tab puts the conversation away without deleting it.
 */
export function ProjectTabs(props: Props) {
  const { t } = useT();
  const [rows, setRows] = useState<SessionRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [memory, setMemory] = useState<ProjectTabMemory>(() =>
    readProjectTabMemory(props.project.id),
  );
  const [dragId, setDragId] = useState("");
  const [menu, setMenu] = useState<{ id: string; anchor: DOMRect } | null>(
    null,
  );
  const stripRef = useRef<HTMLDivElement | null>(null);

  // A mouse wheel over a long tab strip moves it sideways. A trackpad sends a
  // horizontal delta of its own, so only a vertical wheel is translated; the
  // native listener is non-passive because the strip must keep the page still
  // while it scrolls.
  useEffect(() => {
    const el = stripRef.current;
    if (!el) return;
    const onWheel = (ev: WheelEvent) => {
      if (el.scrollWidth <= el.clientWidth || ev.ctrlKey) return;
      if (Math.abs(ev.deltaY) <= Math.abs(ev.deltaX)) return;
      const unit =
        ev.deltaMode === WheelEvent.DOM_DELTA_LINE
          ? 16
          : ev.deltaMode === WheelEvent.DOM_DELTA_PAGE
            ? el.clientWidth
            : 1;
      ev.preventDefault();
      el.scrollLeft += ev.deltaY * unit;
    };
    el.addEventListener("wheel", onWheel, { passive: false });
    return () => el.removeEventListener("wheel", onWheel);
  }, []);

  useEffect(() => {
    setMemory(readProjectTabMemory(props.project.id));
  }, [props.project.id]);

  useEffect(() => {
    writeProjectTabMemory(props.project.id, memory);
  }, [props.project.id, memory]);

  useEffect(() => {
    let live = true;
    let timer: number | undefined;
    const load = () => {
      setLoading(true);
      setError("");
      void listWorkspaceSessions([
        props.project.path,
        props.workspacePath || "",
      ])
        .then((next) => {
          if (live) {
            setRows(next);
            setError("");
          }
        })
        .catch(() => {
          if (live) setError(t("projects.tabs.loadError"));
        })
        .finally(() => {
          if (live) setLoading(false);
        });
    };
    load();
    timer = window.setInterval(load, 20000);
    return () => {
      live = false;
      if (timer !== undefined) window.clearInterval(timer);
    };
  }, [
    props.project.id,
    props.project.path,
    props.workspacePath,
    props.refreshEpoch,
    t,
  ]);

  // A just-created chat may be on screen before the listing catches up. It is
  // shown as a tab but kept out of the remembered order: when the server lists
  // it, its place comes from the project's own order (and it is newest first).
  const visible = useMemo(() => {
    const shown = visibleTabs(rows, memory);
    const sid = props.activeSessionId.trim();
    if (!sid || shown.some((row) => row.id === sid)) {
      return shown;
    }
    if (memory.closed.includes(sid)) {
      return shown;
    }
    return [
      {
        id: sid,
        title: props.activeSessionTitle || "",
        updatedAt: new Date().toISOString(),
      },
      ...shown,
    ];
  }, [rows, memory, props.activeSessionId, props.activeSessionTitle]);

  useEffect(() => {
    setMemory((prev) => {
      const next = reconcileTabMemory(rows, prev);
      return sameMemory(prev, next) ? prev : next;
    });
  }, [rows]);

  useEffect(() => {
    const sid = props.activeSessionId.trim();
    if (!sid || !rows.some((row) => row.id === sid)) return;
    setMemory((prev) => {
      const next = openTab(prev, sid);
      return sameMemory(prev, next) ? prev : next;
    });
  }, [props.activeSessionId, rows]);
  const visibleIds = useMemo(() => visible.map((row) => row.id), [visible]);
  const pinned = useMemo(() => new Set(memory.pinned), [memory.pinned]);

  // A closed active tab can be reached through History or a new chat: bringing
  // it back into view is the same rule as opening it.
  useEffect(() => {
    const sid = props.activeSessionId.trim();
    if (!sid || !visible.some((row) => row.id === sid)) {
      return;
    }
    const nodes =
      stripRef.current?.querySelectorAll<HTMLElement>("[data-session-tab]");
    if (!nodes) {
      return;
    }
    for (const node of nodes) {
      if (node.dataset.sessionTab === sid) {
        node.scrollIntoView({ block: "nearest", inline: "nearest" });
        break;
      }
    }
  }, [props.activeSessionId, visible]);

  const write = (next: ProjectTabMemory) => {
    setMemory(next);
  };

  const activate = (id: string) => {
    if (id && (id !== props.activeSessionId || props.reselectActive)) {
      props.onSelectSession(id);
    }
  };

  const closeOne = (id: string) => {
    if (!visibleIds.includes(id)) {
      write(closeTabs(memory, [id]));
      return;
    }
    const next = closeTabs(memory, [id]);
    const nextActive = nextActiveAfterClose(
      visibleIds,
      [id],
      props.activeSessionId,
    );
    write(next);
    if (id === props.activeSessionId) {
      if (nextActive) props.onSelectSession(nextActive);
      else props.onHome(props.project);
    }
  };

  const closeMany = (ids: string[]) => {
    if (ids.length === 0) return;
    const next = closeTabs(memory, ids);
    const nextActive = nextActiveAfterClose(
      visibleIds,
      ids,
      props.activeSessionId,
    );
    write(next);
    if (ids.includes(props.activeSessionId)) {
      if (nextActive) props.onSelectSession(nextActive);
      else props.onHome(props.project);
    }
  };

  const onPin = (id: string) => {
    write(togglePinned(memory, id));
  };

  const onDrop = (ev: DragEvent<HTMLDivElement>, toId: string) => {
    ev.preventDefault();
    const from = dragId;
    setDragId("");
    if (!from || from === toId) {
      return;
    }
    write(reorderTabs(memory, visibleIds, from, toId));
  };

  const openMenu = (id: string, anchor: DOMRect) => {
    setMenu({ id, anchor });
  };

  const menuItems = (id: string): SessionRowMenuItem[] => {
    const index = visibleIds.indexOf(id);
    const isPinned = pinned.has(id);
    const otherClosable = visibleIds.some((x) => x !== id && !pinned.has(x));
    const leftClosable = visibleIds
      .slice(0, Math.max(0, index))
      .some((x) => !pinned.has(x));
    const rightClosable = visibleIds
      .slice(index + 1)
      .some((x) => !pinned.has(x));
    const items: SessionRowMenuItem[] = [
      {
        key: "pin",
        label: isPinned ? t("projects.tabs.unpin") : t("projects.tabs.pin"),
        testId: "project-tab-pin",
        onPick: () => onPin(id),
      },
      {
        key: "close",
        label: t("projects.tabs.close"),
        testId: "project-tab-close",
        onPick: () => closeOne(id),
      },
    ];
    if (otherClosable) {
      items.push({
        key: "closeOthers",
        label: t("projects.tabs.closeOthers"),
        testId: "project-tab-close-others",
        startsGroup: true,
        onPick: () => {
          const next = closeOthers(memory, id, visibleIds);
          write(next);
          if (
            props.activeSessionId !== id &&
            !pinned.has(props.activeSessionId)
          ) {
            activate(id);
          }
        },
      });
    }
    if (leftClosable) {
      items.push({
        key: "closeLeft",
        label: t("projects.tabs.closeLeft"),
        testId: "project-tab-close-left",
        onPick: () => {
          const closing = visibleIds
            .slice(0, Math.max(0, index))
            .filter((x) => !pinned.has(x));
          closeMany(closing);
        },
      });
    }
    if (rightClosable) {
      items.push({
        key: "closeRight",
        label: t("projects.tabs.closeRight"),
        testId: "project-tab-close-right",
        onPick: () => {
          const closing = visibleIds
            .slice(index + 1)
            .filter((x) => !pinned.has(x));
          closeMany(closing);
        },
      });
    }
    return items;
  };

  return (
    <div className="project-tabs" data-testid="project-tabs">
      <div className="project-tabs-head">
        <span className="project-tabs-folder" aria-hidden>
          <IconFolder />
        </span>
        <div className="project-tabs-picker">
          <select
            className="project-tabs-select"
            aria-label={t("projects.tabs.switch")}
            value={props.project.id}
            onChange={(ev) => {
              const next = props.projects.find((p) => p.id === ev.target.value);
              if (next) props.onSwitchProject(next);
            }}
          >
            {props.projects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </div>
        <button
          type="button"
          className="project-tabs-all"
          onClick={props.onOpenProjects}
          title={t("projects.tabs.allProjects")}
        >
          {t("projects.tabs.allProjects")}
        </button>
        <span className="project-tabs-spacer" />
        <span
          className="project-tabs-count"
          aria-label={t("projects.tabs.count", { count: visible.length })}
        >
          {visible.length}
        </span>
      </div>

      <div
        ref={stripRef}
        className="project-tabs-strip"
        role="tablist"
        aria-label={t("projects.tabs.aria")}
      >
        {visible.length === 0 && !loading ? (
          <span className="project-tabs-empty">{t("projects.tabs.empty")}</span>
        ) : null}
        {visible.map((row) => {
          const active = row.id === props.activeSessionId;
          const isPinned = pinned.has(row.id);
          return (
            <div
              key={row.id}
              data-session-tab={row.id}
              role="tab"
              aria-selected={active}
              tabIndex={active ? 0 : -1}
              className={[
                "project-tab",
                active ? "is-active" : "",
                isPinned ? "is-pinned" : "",
                dragId === row.id ? "is-dragging" : "",
              ]
                .filter(Boolean)
                .join(" ")}
              draggable
              onDragStart={(ev) => {
                setDragId(row.id);
                ev.dataTransfer.effectAllowed = "move";
                ev.dataTransfer.setData("text/plain", row.id);
              }}
              onDragEnd={() => setDragId("")}
              onDragOver={(ev) => {
                ev.preventDefault();
                ev.dataTransfer.dropEffect = "move";
              }}
              onDrop={(ev) => onDrop(ev, row.id)}
              onClick={() => activate(row.id)}
              onKeyDown={(ev) => {
                if (ev.target !== ev.currentTarget) return;
                if (ev.key === "Enter" || ev.key === " ") {
                  ev.preventDefault();
                  activate(row.id);
                }
              }}
              onContextMenu={(ev: MouseEvent<HTMLDivElement>) => {
                ev.preventDefault();
                openMenu(row.id, ev.currentTarget.getBoundingClientRect());
              }}
              title={row.cwd || row.repoRoot || ""}
            >
              {isPinned ? (
                <span className="project-tab-pin" aria-hidden>
                  <IconPin filled />
                </span>
              ) : null}
              {row.turnActive ? (
                <span className="project-tab-live" aria-hidden />
              ) : null}
              <span className="project-tab-title">
                {tabTitle(row, t("projects.tabs.untitled"))}
              </span>
              <button
                type="button"
                className="project-tab-menu"
                aria-label={t("projects.tabs.menu", {
                  title: tabTitle(row, t("projects.tabs.untitled")),
                })}
                onClick={(ev) => {
                  ev.stopPropagation();
                  openMenu(row.id, ev.currentTarget.getBoundingClientRect());
                }}
              >
                ⋯
              </button>
              <button
                type="button"
                className="project-tab-close"
                aria-label={t("projects.tabs.closeOne", {
                  title: tabTitle(row, t("projects.tabs.untitled")),
                })}
                onClick={(ev) => {
                  ev.stopPropagation();
                  closeOne(row.id);
                }}
              >
                ×
              </button>
            </div>
          );
        })}
        <button
          type="button"
          className="project-tabs-new"
          onClick={() => props.onNewChat(props.project)}
          title={t("projects.tabs.newChat")}
          aria-label={t("projects.tabs.newChat")}
        >
          <IconPlus />
        </button>
      </div>

      {error ? (
        <span className="project-tabs-error" role="status">
          {error}
        </span>
      ) : null}

      <SessionRowMenu
        open={menu !== null}
        anchor={menu?.anchor ?? null}
        onClose={() => setMenu(null)}
        items={menu ? menuItems(menu.id) : []}
        ariaLabel={t("projects.tabs.menuAria")}
      />
    </div>
  );
}

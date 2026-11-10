import { useEffect, useRef, useState } from "react";
import { useT } from "../i18n/I18nProvider";
import { countSubagents, countTasks } from "../tasks/taskStatus";
import type { BackgroundTask } from "../tasks/types";
import { filesShortcutLabel } from "../files/filesHotkey";

export function ChatHeader(props: {
  title: string;
  /**
   * The name is still being worked out (the describe call of a new chat): a
   * shimmering bar stands where the title will be.
   */
  titlePending?: boolean;
  editable?: boolean;
  onTitleSave?: (title: string) => void;
  /** Background tasks of this chat, counted on the control `onOpenTasks` puts at the right edge. */
  tasks?: BackgroundTask[];
  /** Subagent runs behind the subagent control; defaults to `tasks`. */
  subagentTasks?: BackgroundTask[];
  /** Opens the background tasks beside the chat, or puts them away when they show. */
  onOpenTasks?: () => void;
  /** The Tasks panel is showing. */
  tasksOpen?: boolean;
  /** Opens the subagent runs of this chat. */
  onOpenSubagents?: () => void;
  /** The Subagents panel is showing. */
  subagentsOpen?: boolean;
  /** Opens the window of the session's files. */
  onOpenFiles?: () => void;
  /** The files window is open. */
  filesOpen?: boolean;
}) {
  const { t } = useT();
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(props.title || "");
  const inputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (!editing) {
      setValue(props.title || "");
    }
  }, [props.title, editing]);

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus();
      inputRef.current?.select();
    }
  }, [editing]);

  const canEdit = !!props.editable && !!props.onTitleSave;

  return (
    <header className="chat-header">
      <div className="chat-title" id="chat-title">
        {canEdit && editing ? (
          <input
            ref={inputRef}
            className="chat-title-input"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onBlur={() => {
              setEditing(false);
              const t = value.trim();
              if (t && t !== (props.title || "").trim()) {
                props.onTitleSave?.(t);
              }
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                (e.target as HTMLInputElement).blur();
              }
              if (e.key === "Escape") {
                // Leaving the title is this Escape's step: a drawer open
                // beside the chat stays (nav/railEscape.ts).
                e.preventDefault();
                setValue(props.title || "");
                setEditing(false);
              }
            }}
          />
        ) : (
          <button
            type="button"
            className={`chat-title-btn ${canEdit ? "is-editable" : ""}`}
            onClick={() => {
              if (canEdit) {
                setEditing(true);
              }
            }}
            aria-label={
              props.titlePending
                ? t("chat.namingChat")
                : t("chat.chatTitleAriaLabel")
            }
            aria-busy={props.titlePending ? true : undefined}
          >
            {props.titlePending ? (
              <span
                className="chat-title-skeleton naming-skeleton-bar"
                data-testid="chat-title-pending"
                aria-hidden="true"
              />
            ) : (
              props.title || t("chat.newChat")
            )}
          </button>
        )}
      </div>
      {props.onOpenTasks ? (
        <HeaderViews
          tasks={props.tasks ?? []}
          {...(props.subagentTasks
            ? { subagentTasks: props.subagentTasks }
            : {})}
          onOpenTasks={props.onOpenTasks}
          tasksOpen={props.tasksOpen === true}
          {...(props.onOpenSubagents
            ? {
                onOpenSubagents: props.onOpenSubagents,
                subagentsOpen: props.subagentsOpen === true,
              }
            : {})}
          {...(props.onOpenFiles ? { onOpenFiles: props.onOpenFiles } : {})}
          filesOpen={props.filesOpen === true}
        />
      ) : null}
    </header>
  );
}

type ViewId = "tasks" | "files" | "subagents";

/**
 * The views of a chat in its header - its files, its background tasks - as a
 * row of buttons at the right of the sticky header, the way the views of a
 * session sit at the top of Claude's app. They are icons at every width, as a
 * phone always showed them: Files is its icon alone, the size of the rail's, in
 * a square of the top bar's size. Background tasks come last, at the right
 * edge: the dot and the running / total count of the control the header always
 * had, the dot in the accent while work is in flight, and the word only while
 * there are no counts yet. The edits open from git's count in the bar over the
 * composer, not from here. The full name is in the tooltip everywhere. A button opens its
 * view and, pressed again, puts it away; the button of a view on show is
 * pressed (aria-pressed: brighter text, border and ground, the way the Tasks
 * control always marked its open panel).
 */
function HeaderViews(props: {
  tasks: BackgroundTask[];
  subagentTasks?: BackgroundTask[];
  onOpenTasks: () => void;
  tasksOpen: boolean;
  onOpenSubagents?: () => void;
  subagentsOpen?: boolean;
  onOpenFiles?: () => void;
  filesOpen: boolean;
}) {
  const { t } = useT();
  const { running, total } = countTasks(props.tasks);
  const subagents = countSubagents(props.subagentTasks ?? props.tasks);
  const live = running > 0;
  const tasksName =
    total === 0
      ? t("tasks.header.ariaEmpty")
      : t("tasks.header.aria", { running, total });

  const views: {
    id: ViewId;
    name: string;
    pressed: boolean;
    run: () => void;
  }[] = [];
  if (props.onOpenFiles) {
    views.push({
      id: "files",
      name: t("chat.views.filesTitle", { key: filesShortcutLabel() }),
      pressed: props.filesOpen,
      run: props.onOpenFiles,
    });
  }
  if (props.onOpenSubagents) {
    views.push({
      id: "subagents",
      name: t("subagents.header.aria", { count: subagents }),
      pressed: props.subagentsOpen === true,
      run: props.onOpenSubagents,
    });
  }
  views.push({
    id: "tasks",
    name: tasksName,
    pressed: props.tasksOpen,
    run: props.onOpenTasks,
  });

  return (
    <div
      className="chat-views"
      role="toolbar"
      aria-label={t("chat.views.label")}
      data-testid="chat-views"
    >
      {views.map((view) => (
        <span key={view.id} className="chat-view-tip-host">
          <button
            type="button"
            className={[
              "chat-view-btn",
              view.id === "tasks" ? "chat-view-btn--tasks" : "",
              view.id === "subagents" ? "chat-view-btn--subagents" : "",
              view.id === "files" ||
              (view.id === "tasks" && total === 0) ||
              (view.id === "subagents" && subagents === 0)
                ? "chat-view-btn--icon"
                : "",
              view.pressed ? "is-active" : "",
              view.id === "tasks" && live ? "is-running" : "",
            ]
              .filter(Boolean)
              .join(" ")}
            data-testid={`chat-views-${view.id}`}
            aria-pressed={view.pressed}
            aria-label={view.name}
            onClick={view.run}
          >
            {/* A square with its icon or dot alone until there are counts, then
                one line of text: the dot sits on it with vertical-align:
                middle. The names are in the tooltip. */}
            <span className="chat-view-face">
              {view.id === "tasks" ? (
                <span
                  className={`bgtask-dot ${live ? "bgtask-dot--running" : "bgtask-dot--muted"}`}
                  aria-hidden="true"
                />
              ) : view.id === "subagents" ? (
                <SubagentIcon />
              ) : (
                <FilesIcon />
              )}
              {view.id === "tasks" && total > 0 ? (
                <span
                  className="chat-view-count"
                  data-testid="chat-views-tasks-count"
                  aria-hidden="true"
                >
                  {t("tasks.header.counts", { running, total })}
                </span>
              ) : null}
              {view.id === "subagents" && subagents > 0 ? (
                <span
                  className="chat-view-count"
                  data-testid="chat-views-subagents-count"
                  aria-hidden="true"
                >
                  {subagents}
                </span>
              ) : null}
            </span>
          </button>
          <span className="chat-view-tip" role="tooltip">
            {view.name}
          </span>
        </span>
      ))}
    </div>
  );
}

/** The Subagents view: two branches of one conversation. */
function SubagentIcon() {
  return (
    <svg
      className="chat-view-icon"
      width={18}
      height={18}
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.3}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <circle cx="4" cy="4" r="1.8" />
      <circle cx="12" cy="8" r="1.8" />
      <circle cx="4" cy="12" r="1.8" />
      <path d="M5.8 4.8l4.4 2.4M5.8 11.2l4.4-2.4" />
    </svg>
  );
}

/** The Files view's 18px line icon, the size of the rail's, in the button's colour. */
function FilesIcon() {
  const common = {
    className: "chat-view-icon",
    width: 18,
    height: 18,
    viewBox: "0 0 16 16",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 1.3,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    "aria-hidden": true,
  };
  return (
    <svg {...common}>
      <path d="M1.75 4.25c0-.83.67-1.5 1.5-1.5h3l1.5 1.75h5c.83 0 1.5.67 1.5 1.5v6.25c0 .83-.67 1.5-1.5 1.5h-9.5c-.83 0-1.5-.67-1.5-1.5z" />
    </svg>
  );
}

import { useT } from "../i18n/I18nProvider";
import { projectPathName, type Project } from "./types";

function projectMonogram(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  if (words.length > 1) {
    return (
      Array.from(words[0]!)[0]! + Array.from(words[1]!)[0]!
    ).toUpperCase();
  }
  return Array.from(name.trim() || "?")
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

function IconGrid() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <rect x="2" y="2" width="4.5" height="4.5" rx="1" />
      <rect x="9.5" y="2" width="4.5" height="4.5" rx="1" />
      <rect x="2" y="9.5" width="4.5" height="4.5" rx="1" />
      <rect x="9.5" y="9.5" width="4.5" height="4.5" rx="1" />
    </svg>
  );
}

/**
 * The project switcher beside the nav rail. It is the project workspace's own
 * list: every project is one click away, without leaving the conversation to
 * the Projects page first.
 */
export function ProjectRail(props: {
  projects: readonly Project[];
  activeProjectId: string;
  onSelect: (project: Project) => void;
  onOpenProjects: () => void;
}) {
  const { t } = useT();
  return (
    <aside
      className="project-rail"
      aria-label={t("projects.rail.aria")}
      data-testid="project-rail"
    >
      <button
        type="button"
        className="project-rail-all"
        title={t("projects.rail.all")}
        aria-label={t("projects.rail.all")}
        onClick={props.onOpenProjects}
      >
        <IconGrid />
      </button>
      <div className="project-rail-list">
        {props.projects.map((project) => {
          const active = project.id === props.activeProjectId;
          return (
            <button
              key={project.id}
              type="button"
              className={`project-rail-item${active ? " is-active" : ""}`}
              aria-current={active ? "page" : undefined}
              aria-label={t("projects.rail.switch", { name: project.name })}
              title={`${project.name}\n${project.path}`}
              onClick={() => {
                if (!active) props.onSelect(project);
              }}
            >
              <span className="project-rail-avatar" aria-hidden>
                {projectMonogram(project.name || projectPathName(project.path))}
              </span>
              <span className="project-rail-name">{project.name}</span>
            </button>
          );
        })}
      </div>
    </aside>
  );
}

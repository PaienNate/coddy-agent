import type { KeyboardEvent, MouseEvent } from "react";
import { useT } from "../i18n/I18nProvider";

export type PageAttentionKind = "permission" | "question" | "subagent";

export type PageAttentionNotice = {
  id: string;
  sessionId: string;
  kind: PageAttentionKind;
  title: string;
  body: string;
};

function AttentionIcon(props: { kind: PageAttentionKind }) {
  if (props.kind === "question") {
    return (
      <svg
        width="18"
        height="18"
        viewBox="0 0 18 18"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        aria-hidden
      >
        <circle cx="9" cy="9" r="7" />
        <path d="M7.2 6.8a2 2 0 1 1 3.4 1.4c-.9.6-1.6 1-1.6 2.1" />
        <path d="M9 13.2h.01" />
      </svg>
    );
  }
  if (props.kind === "subagent") {
    return (
      <svg
        width="18"
        height="18"
        viewBox="0 0 18 18"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden
      >
        <circle cx="5" cy="4.5" r="1.7" />
        <circle cx="13" cy="9" r="1.7" />
        <circle cx="5" cy="13.5" r="1.7" />
        <path d="M6.7 5.3l4.6 2.7M6.7 12.7l4.6-2.7" />
      </svg>
    );
  }
  return (
    <svg
      width="18"
      height="18"
      viewBox="0 0 18 18"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <path d="M9 2.5 16 15H2L9 2.5Z" />
      <path d="M9 6.4v4.1M9 13.2h.01" />
    </svg>
  );
}

/**
 * Notices for work that is paused until a person acts. They live in the page,
 * unlike the browser notification a hidden tab receives, and one click opens
 * the chat that is waiting.
 */
export function PageAttentionToasts(props: {
  notices: readonly PageAttentionNotice[];
  onOpen: (sessionId: string) => void;
  onDismiss: (id: string) => void;
}) {
  const { t } = useT();
  if (props.notices.length === 0) return null;

  const open = (notice: PageAttentionNotice) => {
    props.onOpen(notice.sessionId);
  };
  const onKeyDown = (
    ev: KeyboardEvent<HTMLDivElement>,
    notice: PageAttentionNotice,
  ) => {
    if (ev.target !== ev.currentTarget) return;
    if (ev.key === "Enter" || ev.key === " ") {
      ev.preventDefault();
      open(notice);
    }
  };
  const dismiss = (ev: MouseEvent<HTMLButtonElement>, id: string) => {
    ev.stopPropagation();
    props.onDismiss(id);
  };

  return (
    <section
      className="page-attention-stack"
      aria-label={t("attention.page.aria")}
      aria-live="polite"
      data-testid="page-attention"
    >
      {props.notices.map((notice) => (
        <div
          key={notice.id}
          className={`page-attention-card is-${notice.kind}`}
          role="button"
          tabIndex={0}
          data-testid={`page-attention-${notice.kind}-${notice.sessionId}`}
          onClick={() => open(notice)}
          onKeyDown={(ev) => onKeyDown(ev, notice)}
        >
          <span className="page-attention-icon" aria-hidden>
            <AttentionIcon kind={notice.kind} />
          </span>
          <span className="page-attention-copy">
            <span className="page-attention-title">{notice.title}</span>
            <span className="page-attention-body">{notice.body}</span>
            <span className="page-attention-hint">
              {t("attention.page.open")}
            </span>
          </span>
          <button
            type="button"
            className="page-attention-close"
            aria-label={t("attention.page.dismiss")}
            onClick={(ev) => dismiss(ev, notice.id)}
          >
            ×
          </button>
        </div>
      ))}
    </section>
  );
}

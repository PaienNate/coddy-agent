import { memo, useEffect, useMemo, useState, type ReactNode } from "react";

import { Chevron } from "../components/Chevron";
import { useT } from "../i18n/I18nProvider";
import {
  toolGroupCounts,
  type IndexedToolCall,
  type ToolCallItem,
  type ToolGroupCategory,
} from "./toolGroups";

const categoryLabelKey: Record<ToolGroupCategory, string> = {
  shell: "messages.toolGroupShell",
  read: "messages.toolGroupRead",
  search: "messages.toolGroupSearch",
  edit: "messages.toolGroupEdit",
  web: "messages.toolGroupWeb",
  agent: "messages.toolGroupAgent",
  other: "messages.toolGroupOther",
};

export type ToolGroupMessageProps = {
  items: IndexedToolCall[];
  renderItem: (item: ToolCallItem) => ReactNode;
};

/**
 * One folded run of adjacent tool calls: the "Used N tools" line in the
 * transcript. The individual rows are only mounted while the group is open, so
 * a long turn costs one summary row instead of one row per call.
 */
export const ToolGroupMessage = memo(function ToolGroupMessage(
  props: ToolGroupMessageProps,
) {
  const { t } = useT();
  const pending = props.items.some(({ item }) =>
    ["pending", "in_progress"].includes((item.status || "").toLowerCase()),
  );
  const failed = props.items.filter(
    ({ item }) => (item.status || "").toLowerCase() === "failed",
  ).length;
  const [open, setOpen] = useState(pending || failed > 0);

  // A group that was complete when it first appeared and then receives another
  // call is live again; open it once, without overriding a reader who collapses
  // it while the call is still running.
  useEffect(() => {
    if (pending || failed > 0) setOpen(true);
  }, [pending, failed]);

  const counts = useMemo(
    () => toolGroupCounts(props.items.map(({ item }) => item)),
    [props.items],
  );
  const status = pending ? "running" : failed > 0 ? "failed" : "completed";

  return (
    <div
      className="thinking-row tool-group-row"
      data-testid="tool-group"
      data-status={status}
    >
      <details
        className="thinking-details"
        open={open}
        onToggle={(event) => setOpen(event.currentTarget.open)}
      >
        <summary
          className="thinking-summary"
          data-testid="tool-group-summary"
          aria-label={t("messages.toolGroupAriaLabel", {
            count: props.items.length,
          })}
        >
          <span className="thinking-left">
            <Chevron className="thinking-chevron" />
            <span className="thinking-head">
              <span className="thinking-label">
                {t("messages.toolGroupSummary", {
                  count: props.items.length,
                })}
              </span>
              <span className="thinking-trail tool-group-counts">
                {counts.map(({ category, count }, index) => (
                  <span className="tool-group-count" key={category}>
                    {index > 0 ? "· " : ""}
                    {t(categoryLabelKey[category])} {count}
                  </span>
                ))}
                {failed > 0 ? (
                  <span className="tool-failed-marker">
                    {t("messages.toolGroupFailed", { count: failed })}
                  </span>
                ) : null}
              </span>
            </span>
          </span>
        </summary>
        {open ? (
          <div className="tool-group-list">
            {props.items.map(({ item }) => (
              <div key={item.id} data-tool-group-item={item.toolCallId}>
                {props.renderItem(item)}
              </div>
            ))}
          </div>
        ) : null}
      </details>
    </div>
  );
});

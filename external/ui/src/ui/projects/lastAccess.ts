import { getLocale } from "../i18n/i18n";

/**
 * How long ago the Projects page says a project was last opened. Recent visits
 * keep the granularity of a relative time; beyond a month a month or a year is
 * enough, and the exact instant stays in the card's tooltip.
 */
export function projectLastAccessAgo(
  iso: string | undefined,
  now: number = Date.now(),
  locale: string = getLocale(),
): string {
  const ms = Date.parse(String(iso ?? ""));
  if (!Number.isFinite(ms)) {
    return "";
  }
  const seconds = Math.max(0, Math.floor((now - ms) / 1000));
  const rtf = new Intl.RelativeTimeFormat(locale, {
    numeric: "auto",
    style: "short",
  });
  if (seconds < 60) {
    return rtf.format(-seconds, "second");
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return rtf.format(-minutes, "minute");
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return rtf.format(-hours, "hour");
  }
  const days = Math.floor(hours / 24);
  if (days < 7) {
    return rtf.format(-days, "day");
  }
  if (days < 30) {
    return rtf.format(-Math.floor(days / 7), "week");
  }
  if (days < 365) {
    return rtf.format(-Math.floor(days / 30), "month");
  }
  return rtf.format(-Math.floor(days / 365), "year");
}

/** The exact local instant behind a project card's relative label. */
export function projectLastAccessFull(
  iso: string | undefined,
  locale: string = getLocale(),
): string {
  const ms = Date.parse(String(iso ?? ""));
  if (!Number.isFinite(ms)) {
    return "";
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(ms));
}

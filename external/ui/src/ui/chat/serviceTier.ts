export type ServiceTierControl = {
  value: string;
  choices: string[];
  onChange: (value: string) => void;
};

const COOKIE = "coddy_service_tier";

export function readServiceTierCookie(): string {
  if (typeof document === "undefined") return "";
  const part = document.cookie
    .split(";")
    .find((part) => part.trim().startsWith(COOKIE + "="));
  try {
    return part ? decodeURIComponent(part.trim().slice(COOKIE.length + 1)) : "";
  } catch {
    return "";
  }
}

export function writeServiceTierCookie(value: string): void {
  const secure = window.location.protocol === "https:" ? "; Secure" : "";
  document.cookie = `${COOKIE}=${encodeURIComponent(value)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
}

export function pickServiceTier(
  value: string,
  choices: readonly string[],
): string {
  return choices.includes(value) ? value : "";
}

export function serviceTierLabelKey(value: string): string {
  return "composer.serviceTier." + (value || "default");
}

/**
 * The tiers a session may pick, or `fallback` (the model row's own list) when
 * the snapshot carried none.
 *
 * A settings snapshot is not guaranteed to carry the list: an event from an
 * older server, or one written before the field existed, leaves it undefined.
 * The composer narrowed the result with `.filter`, so a missing list did not
 * merely drop the chip, it threw during render and blanked the whole page on a
 * model switch. Anything that is not an array falls back instead.
 */
export function sessionTierChoiceList(
  choices: readonly unknown[] | undefined,
  fallback: readonly string[],
): string[] {
  if (!Array.isArray(choices)) return [...fallback];
  return choices.filter(
    (choice): choice is string => typeof choice === "string",
  );
}

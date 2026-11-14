/** The raw request/response viewer's own API: GET reads the captured
 * exchanges, DELETE clears them. A server that has httpserver.raw_capture off
 * answers {enabled:false}, and the UI shows nothing then. */
import type { RawCaptureDocument } from "./types";

export type RawCaptureApiResult<T> =
  | { ok: true; data: T }
  | { ok: false; status: number; message: string };

async function readErrorMessage(res: Response): Promise<string> {
  try {
    const j = (await res.json()) as { error?: { message?: string } };
    const m = j?.error?.message;
    if (typeof m === "string" && m.trim()) return m.trim();
  } catch {
    /* ignore */
  }
  return `HTTP ${res.status}`;
}

async function parseJson<T>(res: Response): Promise<RawCaptureApiResult<T>> {
  if (!res.ok) {
    return {
      ok: false,
      status: res.status,
      message: await readErrorMessage(res),
    };
  }
  return { ok: true, data: (await res.json()) as T };
}

async function request(
  path: string,
  init?: RequestInit,
): Promise<Response | null> {
  try {
    return await fetch(path, init);
  } catch {
    return null;
  }
}

export async function readRawCapture(): Promise<
  RawCaptureApiResult<RawCaptureDocument>
> {
  const res = await request("/coddy/raw-capture");
  return res
    ? parseJson<RawCaptureDocument>(res)
    : { ok: false, status: 0, message: "HTTP 0" };
}

export async function clearRawCapture(): Promise<
  RawCaptureApiResult<RawCaptureDocument>
> {
  const res = await request("/coddy/raw-capture", { method: "DELETE" });
  return res
    ? parseJson<RawCaptureDocument>(res)
    : { ok: false, status: 0, message: "HTTP 0" };
}

/** The SSH credentials the settings page keeps for ssh_run_command. The
 * server never returns a password: it reports has_password instead, and an
 * empty password on save keeps the stored one. */
import type { RawCaptureApiResult } from "../rawCapture/api";

export type SSHHost = {
  host: string;
  user: string;
  port?: number;
  has_password: boolean;
  password_command?: string;
  key_path?: string;
};

export type SSHHostWrite = {
  host: string;
  user: string;
  port?: number;
  password?: string;
  password_command?: string;
  key_path?: string;
};

export type SSHHostsDocument = {
  object: string;
  hosts: SSHHost[];
};

export type SSHTestResult = {
  object: string;
  ok: boolean;
  message?: string;
  latency_ms?: number;
};

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

export async function readSSHHosts(): Promise<
  RawCaptureApiResult<SSHHostsDocument>
> {
  const res = await request("/coddy/ssh/hosts");
  return res
    ? parseJson<SSHHostsDocument>(res)
    : { ok: false, status: 0, message: "HTTP 0" };
}

export async function saveSSHHost(
  body: SSHHostWrite,
): Promise<RawCaptureApiResult<{ object: string; host: SSHHost }>> {
  const res = await request("/coddy/ssh/hosts", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return res
    ? parseJson<{ object: string; host: SSHHost }>(res)
    : { ok: false, status: 0, message: "HTTP 0" };
}

export async function deleteSSHHost(
  host: string,
  user: string,
  port?: number,
): Promise<RawCaptureApiResult<{ object: string; deleted: boolean }>> {
  const q = new URLSearchParams({ host, user });
  if (port && port > 0) q.set("port", String(port));
  const res = await request(`/coddy/ssh/hosts?${q.toString()}`, {
    method: "DELETE",
  });
  return res
    ? parseJson<{ object: string; deleted: boolean }>(res)
    : { ok: false, status: 0, message: "HTTP 0" };
}

export async function testSSHHost(
  body: SSHHostWrite,
): Promise<RawCaptureApiResult<SSHTestResult>> {
  const res = await request("/coddy/ssh/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return res
    ? parseJson<SSHTestResult>(res)
    : { ok: false, status: 0, message: "HTTP 0" };
}

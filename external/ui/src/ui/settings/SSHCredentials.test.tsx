import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { I18nProvider } from "../i18n/I18nProvider";
import { initLocale } from "../i18n/i18n";
import { SSHCredentials } from "./SSHCredentials";

type Reply = { status: number; body: unknown };

function jsonResponse({ status, body }: Reply): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function renderSection() {
  return render(
    <I18nProvider>
      <SSHCredentials />
    </I18nProvider>,
  );
}

describe("SSH credentials", () => {
  const calls: Array<{ method: string; url: string; body?: unknown }> = [];
  let hosts: unknown[] = [];

  beforeEach(() => {
    initLocale("en");
    calls.length = 0;
    hosts = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (path: string, init?: RequestInit) => {
        const method = init?.method ?? "GET";
        const body = init?.body ? JSON.parse(String(init.body)) : undefined;
        calls.push({ method, url: path, ...(body ? { body } : {}) });
        if (path === "/coddy/ssh/hosts" && method === "GET") {
          return jsonResponse({ status: 200, body: { object: "coddy.ssh_hosts", hosts } });
        }
        if (path === "/coddy/ssh/hosts" && method === "PUT") {
          hosts = [
            {
              host: body.host,
              user: body.user,
              port: body.port,
              has_password: Boolean(body.password),
            },
          ];
          return jsonResponse({ status: 200, body: { object: "coddy.ssh_host", host: hosts[0] } });
        }
        if (path === "/coddy/ssh/hosts" && method === "DELETE") {
          hosts = [];
          return jsonResponse({ status: 200, body: { object: "coddy.ssh_host", deleted: true } });
        }
        if (path === "/coddy/ssh/test" && method === "POST") {
          return jsonResponse({
            status: 200,
            body: { object: "coddy.ssh_test", ok: true, latency_ms: 7 },
          });
        }
        return jsonResponse({ status: 404, body: { error: { message: "no route" } } });
      }),
    );
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("starts empty and saves a host through the API", async () => {
    renderSection();
    await waitFor(() => expect(screen.getByText(/No host added yet/)).toBeTruthy());

    const inputs = screen.getAllByRole("textbox") as HTMLInputElement[];
    fireEvent.change(inputs[0]!, { target: { value: "nas.local" } });
    fireEvent.change(inputs[1]!, { target: { value: "admin" } });
    const password = document.querySelector('input[type="password"]') as HTMLInputElement;
    fireEvent.change(password, { target: { value: "s3cret" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(calls.some((c) => c.method === "PUT")).toBe(true),
    );
    const put = calls.find((c) => c.method === "PUT")!;
    expect(put.body).toMatchObject({
      host: "nas.local",
      user: "admin",
      password: "s3cret",
    });
    // The list is read again, so the saved host is drawn.
    await waitFor(() => expect(screen.getByText("admin@nas.local:22")).toBeTruthy());
  });

  it("tests the typed credentials and shows the outcome", async () => {
    renderSection();
    await waitFor(() => expect(screen.getByText(/No host added yet/)).toBeTruthy());
    const inputs = screen.getAllByRole("textbox") as HTMLInputElement[];
    fireEvent.change(inputs[0]!, { target: { value: "nas.local" } });
    fireEvent.change(inputs[1]!, { target: { value: "admin" } });
    fireEvent.click(screen.getByRole("button", { name: "Test" }));

    await waitFor(() => expect(screen.getByText(/Connected in 7 ms/)).toBeTruthy());
    const test = calls.find((c) => c.method === "POST")!;
    expect(test.url).toBe("/coddy/ssh/test");
    expect(test.body).toMatchObject({ host: "nas.local", user: "admin" });
  });

  it("shows why a test failed instead of a bare no", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (path: string, init?: RequestInit) => {
        if (path === "/coddy/ssh/hosts" && (init?.method ?? "GET") === "GET") {
          return jsonResponse({ status: 200, body: { object: "coddy.ssh_hosts", hosts: [] } });
        }
        return jsonResponse({
          status: 200,
          body: {
            object: "coddy.ssh_test",
            ok: false,
            message: "connect to nas.local:22: connection refused",
          },
        });
      }),
    );
    renderSection();
    await waitFor(() => expect(screen.getByText(/No host added yet/)).toBeTruthy());
    const inputs = screen.getAllByRole("textbox") as HTMLInputElement[];
    fireEvent.change(inputs[0]!, { target: { value: "nas.local" } });
    fireEvent.change(inputs[1]!, { target: { value: "admin" } });
    fireEvent.click(screen.getByRole("button", { name: "Test" }));
    await waitFor(() =>
      expect(screen.getByText(/connection refused/)).toBeTruthy(),
    );
  });
});

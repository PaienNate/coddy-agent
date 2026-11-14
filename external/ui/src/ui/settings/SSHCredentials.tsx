import { useCallback, useEffect, useState } from "react";
import { useT } from "../i18n/I18nProvider";
import {
  deleteSSHHost,
  readSSHHosts,
  saveSSHHost,
  testSSHHost,
  type SSHHost,
  type SSHHostWrite,
} from "./sshCredentials";

/** What the form holds. Port stays a string while typing so it can be empty;
 * the password is never prefilled - an empty one keeps the stored secret. */
type Draft = {
  host: string;
  user: string;
  port: string;
  password: string;
  password_command: string;
  key_path: string;
};

function emptyDraft(): Draft {
  return {
    host: "",
    user: "",
    port: "",
    password: "",
    password_command: "",
    key_path: "",
  };
}

function draftOf(h: SSHHost): Draft {
  return {
    host: h.host,
    user: h.user,
    port: h.port ? String(h.port) : "",
    password: "",
    password_command: h.password_command ?? "",
    key_path: h.key_path ?? "",
  };
}

function bodyOf(d: Draft): SSHHostWrite {
  const port = d.port.trim() === "" ? undefined : Number(d.port);
  return {
    host: d.host.trim(),
    user: d.user.trim(),
    ...(port && Number.isFinite(port) ? { port } : {}),
    ...(d.password ? { password: d.password } : {}),
    password_command: d.password_command.trim(),
    key_path: d.key_path.trim(),
  };
}

function hostKey(h: SSHHost): string {
  return `${h.user}@${h.host}:${h.port ?? 22}`;
}

/**
 * The SSH credentials the agent's ssh_run_command tool uses: hosts with a
 * password, a credential helper or a chosen key, and a Test button that dials
 * the host without running anything. The server never returns a password, so
 * the form leaves the field empty and an empty one keeps the stored secret.
 */
export function SSHCredentials() {
  const { t } = useT();
  const [hosts, setHosts] = useState<SSHHost[]>([]);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [test, setTest] = useState<{ ok: boolean; text: string } | null>(null);

  const load = useCallback(async () => {
    const result = await readSSHHosts();
    if (result.ok) {
      setHosts(result.data.hosts);
      setError(null);
    } else {
      setError(result.message);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const set = (field: keyof Draft) => (value: string) =>
    setDraft((d) => ({ ...d, [field]: value }));

  const save = useCallback(async () => {
    setBusy(true);
    setNotice(null);
    const result = await saveSSHHost(bodyOf(draft));
    setBusy(false);
    if (!result.ok) {
      setError(result.message);
      return;
    }
    setError(null);
    setNotice(t("ssh.saved", { host: `${result.data.host.user}@${result.data.host.host}` }));
    setDraft(emptyDraft());
    await load();
  }, [draft, load, t]);

  const runTest = useCallback(async () => {
    setBusy(true);
    setTest(null);
    const result = await testSSHHost(bodyOf(draft));
    setBusy(false);
    if (!result.ok) {
      setError(result.message);
      return;
    }
    setError(null);
    setTest(
      result.data.ok
        ? { ok: true, text: t("ssh.testOk", { ms: result.data.latency_ms ?? 0 }) }
        : {
            ok: false,
            text: t("ssh.testFailed", {
              message: result.data.message ?? t("ssh.testFailedUnknown"),
            }),
          },
    );
  }, [draft, t]);

  const remove = useCallback(
    async (h: SSHHost) => {
      setBusy(true);
      const result = await deleteSSHHost(h.host, h.user, h.port);
      setBusy(false);
      if (!result.ok) {
        setError(result.message);
        return;
      }
      setError(null);
      await load();
    },
    [load],
  );

  return (
    <section className="system-block">
      <h3 className="system-h" tabIndex={-1}>
        {t("ssh.title")}
      </h3>
      <p className="settings-field-desc">{t("ssh.hint")}</p>
      {error ? <p className="settings-error">{error}</p> : null}
      {notice ? <p className="settings-muted">{notice}</p> : null}

      {hosts.length === 0 ? (
        <p className="settings-muted">{t("ssh.empty")}</p>
      ) : (
        <ul className="ssh-hosts">
          {hosts.map((h) => (
            <li key={hostKey(h)} className="settings-row ssh-host">
              <span className="ssh-host-name">{hostKey(h)}</span>
              <span className="settings-muted">
                {h.has_password
                  ? t("ssh.passwordSet")
                  : h.key_path || h.password_command || t("ssh.noCredential")}
              </span>
              <span className="ssh-host-actions">
                <button
                  type="button"
                  className="settings-btn"
                  disabled={busy}
                  onClick={() => {
                    setDraft(draftOf(h));
                    setTest(null);
                    setNotice(null);
                  }}
                >
                  {t("ssh.edit")}
                </button>
                <button
                  type="button"
                  className="settings-btn settings-btn-danger"
                  disabled={busy}
                  onClick={() => void remove(h)}
                >
                  {t("ssh.delete")}
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}

      <div className="ssh-form">
        <label className="settings-row">
          <span>{t("ssh.host")}</span>
          <input
            className="settings-input"
            value={draft.host}
            disabled={busy}
            onChange={(e) => set("host")(e.target.value)}
            placeholder="nas.local"
            autoComplete="off"
            spellCheck={false}
          />
        </label>
        <label className="settings-row">
          <span>{t("ssh.user")}</span>
          <input
            className="settings-input"
            value={draft.user}
            disabled={busy}
            onChange={(e) => set("user")(e.target.value)}
            placeholder="admin"
            autoComplete="off"
            spellCheck={false}
          />
        </label>
        <label className="settings-row">
          <span>{t("ssh.port")}</span>
          <input
            className="settings-input"
            value={draft.port}
            disabled={busy}
            onChange={(e) => set("port")(e.target.value)}
            placeholder="22"
            inputMode="numeric"
            autoComplete="off"
          />
        </label>
        <label className="settings-row">
          <span>{t("ssh.password")}</span>
          <input
            className="settings-input"
            type="password"
            value={draft.password}
            disabled={busy}
            onChange={(e) => set("password")(e.target.value)}
            placeholder={t("ssh.passwordPlaceholder")}
            autoComplete="new-password"
          />
        </label>
        <label className="settings-row">
          <span>{t("ssh.passwordCommand")}</span>
          <input
            className="settings-input"
            value={draft.password_command}
            disabled={busy}
            onChange={(e) => set("password_command")(e.target.value)}
            placeholder="pass show nas"
            autoComplete="off"
            spellCheck={false}
          />
        </label>
        <label className="settings-row">
          <span>{t("ssh.keyPath")}</span>
          <input
            className="settings-input"
            value={draft.key_path}
            disabled={busy}
            onChange={(e) => set("key_path")(e.target.value)}
            placeholder="~/.ssh/id_ed25519"
            autoComplete="off"
            spellCheck={false}
          />
        </label>
        <div className="ssh-actions">
          <button
            type="button"
            className="settings-btn settings-btn-primary"
            disabled={busy || !draft.host.trim() || !draft.user.trim()}
            onClick={() => void save()}
          >
            {t("ssh.save")}
          </button>
          <button
            type="button"
            className="settings-btn"
            disabled={busy || !draft.host.trim() || !draft.user.trim()}
            onClick={() => void runTest()}
          >
            {t("ssh.test")}
          </button>
          <button
            type="button"
            className="settings-btn"
            disabled={busy}
            onClick={() => {
              setDraft(emptyDraft());
              setTest(null);
              setNotice(null);
              setError(null);
            }}
          >
            {t("ssh.clear")}
          </button>
        </div>
        {test ? (
          <p className={test.ok ? "ssh-test-ok" : "settings-error"}>
            {test.text}
          </p>
        ) : null}
      </div>
    </section>
  );
}

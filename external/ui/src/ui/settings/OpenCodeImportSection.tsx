import { useCallback, useEffect, useRef, useState } from "react";
import { responseError } from "./apiError";
import { useT } from "../i18n/I18nProvider";
import { FieldLabel } from "./FieldHint";

type Conn = {
  url: string;
  username: string;
  password_configured: boolean;
};

type TestResult = {
  version: string;
  projects: number;
  sessions: number;
};

type ImportJob = {
  id: string;
  started: string;
  done: boolean;
  error?: string;
  done_sessions: number;
  total_sessions: number;
  current_session?: string;
  result?: {
    projects: number;
    sessions: number;
    imported: number;
    skipped: number;
    failed: number;
    read_only: number;
    errors?: string[];
  };
};

const EMPTY: Conn = {
  url: "",
  username: "opencode",
  password_configured: false,
};

/**
 * Settings -> OpenCode: the explicit connection used by the one-shot v2
 * migration. The password is write-only; an empty field keeps the saved one.
 */
export function OpenCodeImportSection() {
  const { t } = useT();
  const [conn, setConn] = useState<Conn>(EMPTY);
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState<"load" | "save" | "test" | "import" | "">(
    "",
  );
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [test, setTest] = useState<TestResult | null>(null);
  const [job, setJob] = useState<ImportJob | null>(null);
  const poll = useRef<number | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    setBusy("load");
    fetch("/coddy/opencode", { signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error(await responseError(response));
        return (await response.json()) as Conn;
      })
      .then((next) => {
        if (!controller.signal.aborted) setConn(next);
      })
      .catch((err: unknown) => {
        if (!controller.signal.aborted) setError(String(err));
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy("");
      });
    return () => {
      controller.abort();
      if (poll.current !== null) window.clearTimeout(poll.current);
    };
  }, []);

  const save = useCallback(async () => {
    setBusy("save");
    setError("");
    setNotice("");
    try {
      const response = await fetch("/coddy/opencode", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          url: conn.url,
          username: conn.username,
          password,
        }),
      });
      if (!response.ok) throw new Error(await responseError(response));
      const next = (await response.json()) as Conn;
      setConn(next);
      setPassword("");
      setNotice(t("settings.opencode.saved"));
    } catch (err: unknown) {
      setError(String(err));
    } finally {
      setBusy("");
    }
  }, [conn.url, conn.username, password, t]);

  const runTest = useCallback(async () => {
    setBusy("test");
    setError("");
    setNotice("");
    setTest(null);
    try {
      const response = await fetch("/coddy/opencode/test", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          url: conn.url,
          username: conn.username,
          password,
        }),
      });
      if (!response.ok) throw new Error(await responseError(response));
      setTest((await response.json()) as TestResult);
    } catch (err: unknown) {
      setError(String(err));
    } finally {
      setBusy("");
    }
  }, [conn.url, conn.username, password]);

  const pollJob = useCallback(
    async (id: string) => {
      const response = await fetch(
        `/coddy/opencode/import/${encodeURIComponent(id)}`,
      );
      if (!response.ok) throw new Error(await responseError(response));
      const next = (await response.json()) as ImportJob;
      setJob(next);
      if (!next.done) {
        poll.current = window.setTimeout(() => {
          void pollJob(id).catch((err: unknown) => {
            setError(String(err));
            setBusy("");
          });
        }, 1000);
      } else {
        setBusy("");
        if (next.error) setError(next.error);
        else setNotice(t("settings.opencode.importDone"));
      }
    },
    [t],
  );

  const startImport = useCallback(async () => {
    setBusy("import");
    setError("");
    setNotice("");
    setJob(null);
    try {
      const response = await fetch("/coddy/opencode/import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          url: conn.url,
          username: conn.username,
          password,
          dry_run: false,
          skip_archived: false,
        }),
      });
      if (!response.ok) throw new Error(await responseError(response));
      const first = (await response.json()) as ImportJob;
      setJob(first);
      if (first.done) {
        setBusy("");
        if (first.error) setError(first.error);
        else setNotice(t("settings.opencode.importDone"));
        return;
      }
      poll.current = window.setTimeout(() => {
        void pollJob(first.id).catch((err: unknown) => {
          setError(String(err));
          setBusy("");
        });
      }, 1000);
    } catch (err: unknown) {
      setError(String(err));
      setBusy("");
    }
  }, [conn.url, conn.username, password, pollJob, t]);

  const disabled = busy !== "";

  return (
    <div className="settings-opencode-import" data-testid="opencode-import">
      <p className="settings-help">{t("settings.opencode.lead")}</p>
      <div className="settings-field">
        <FieldLabel label={t("settings.opencode.url")} />
        <input
          className="settings-input"
          data-testid="opencode-url"
          value={conn.url}
          placeholder="https://opencode.example.com:4096"
          onChange={(e) => setConn({ ...conn, url: e.target.value })}
          disabled={disabled}
        />
      </div>
      <div className="settings-field">
        <FieldLabel label={t("settings.opencode.username")} />
        <input
          className="settings-input"
          data-testid="opencode-username"
          value={conn.username}
          onChange={(e) => setConn({ ...conn, username: e.target.value })}
          disabled={disabled}
        />
      </div>
      <div className="settings-field">
        <FieldLabel label={t("settings.opencode.password")} />
        <input
          className="settings-input"
          data-testid="opencode-password"
          type="password"
          value={password}
          placeholder={
            conn.password_configured
              ? t("settings.opencode.passwordSet")
              : t("settings.opencode.passwordEmpty")
          }
          onChange={(e) => setPassword(e.target.value)}
          disabled={disabled}
        />
      </div>
      <div className="settings-section-actions">
        <button
          type="button"
          className="settings-btn"
          data-testid="opencode-save"
          onClick={() => void save()}
          disabled={disabled}
        >
          {t("settings.opencode.save")}
        </button>
        <button
          type="button"
          className="settings-btn"
          data-testid="opencode-test"
          onClick={() => void runTest()}
          disabled={disabled}
        >
          {t("settings.opencode.test")}
        </button>
        <button
          type="button"
          className="settings-btn settings-btn-primary"
          data-testid="opencode-import"
          onClick={() => void startImport()}
          disabled={disabled}
        >
          {t("settings.opencode.import")}
        </button>
      </div>
      {test ? (
        <p className="settings-help" data-testid="opencode-test-result">
          {t("settings.opencode.testOk", {
            version: test.version,
            projects: test.projects,
            sessions: test.sessions,
          })}
        </p>
      ) : null}
      {job ? (
        <p className="settings-help" data-testid="opencode-import-progress">
          {job.done
            ? t("settings.opencode.importSummary", {
                imported: job.result?.imported ?? 0,
                skipped: job.result?.skipped ?? 0,
                failed: job.result?.failed ?? 0,
              })
            : t("settings.opencode.importProgress", {
                done: job.done_sessions,
                total: job.total_sessions,
              })}
        </p>
      ) : null}
      {notice ? <p className="settings-help">{notice}</p> : null}
      {error ? <p className="settings-error">{error}</p> : null}
    </div>
  );
}

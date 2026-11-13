import { useCallback, useEffect, useState } from "react";
import { useT } from "../i18n/I18nProvider";
import {
  clearRawCapture,
  readRawCapture,
  type RawCaptureApiResult,
} from "./api";
import type { RawCaptureDocument, RawCaptureEntry } from "./types";

/**
 * The raw-capture debug viewer: what actually went over the wire to a
 * provider. It renders nothing while httpserver.raw_capture is off (the server
 * answers {enabled:false}); with it on it lists each captured exchange and lets
 * the operator clear the ring. The values are memory only, credentials already
 * stripped by the server.
 */
export function RawCaptureView() {
  const { t } = useT();
  const [doc, setDoc] = useState<RawCaptureDocument | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [clearing, setClearing] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    const result: RawCaptureApiResult<RawCaptureDocument> =
      await readRawCapture();
    setLoading(false);
    if (result.ok) {
      setDoc(result.data);
      setError(null);
    } else {
      setDoc(null);
      setError(result.message);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const clear = useCallback(async () => {
    setClearing(true);
    const result = await clearRawCapture();
    setClearing(false);
    if (result.ok) {
      setDoc(result.data);
      setError(null);
    } else {
      setError(result.message);
    }
  }, []);

  // Off: the server reports capture is not enabled, so the viewer stays hidden.
  if (!doc || !doc.enabled) return null;

  return (
    <section className="system-block">
      <h3 className="system-h" tabIndex={-1}>
        {t("rawCapture.title")}
      </h3>
      <div className="raw-capture">
        <div className="raw-capture-toolbar">
          <span className="raw-capture-count">
            {t("rawCapture.count", { count: doc.entries.length })}
          </span>
          <button
            type="button"
            className="raw-capture-clear"
            onClick={() => void clear()}
            disabled={clearing || doc.entries.length === 0}
          >
            {t("rawCapture.clear")}
          </button>
        </div>
        {error ? <p className="raw-capture-error">{error}</p> : null}
        {loading ? (
          <p className="raw-capture-muted">{t("rawCapture.loading")}</p>
        ) : doc.entries.length === 0 ? (
          <p className="raw-capture-muted">{t("rawCapture.empty")}</p>
        ) : (
          <ul className="raw-capture-list">
            {doc.entries.map((e) => (
              <li key={e.id} className="raw-capture-item">
                <RawCaptureExchange entry={e} />
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}

function RawCaptureExchange({ entry }: { entry: RawCaptureEntry }) {
  const { t } = useT();
  return (
    <>
      <div className="raw-capture-line">
        <span className="raw-capture-method">{entry.method}</span>
        <span className="raw-capture-url">{entry.url}</span>
        {entry.status ? (
          <span
            className="raw-capture-status"
            data-tone={entry.status >= 400 ? "error" : "ok"}
          >
            {entry.status}
          </span>
        ) : null}
        {entry.err ? (
          <span className="raw-capture-err">{entry.err}</span>
        ) : null}
      </div>
      <Pre
        label={t("rawCapture.request")}
        {...(entry.reqHeader ? { header: entry.reqHeader } : {})}
        {...(entry.reqBody !== undefined ? { body: entry.reqBody } : {})}
      />
      <Pre
        label={t("rawCapture.response")}
        {...(entry.respHeader ? { header: entry.respHeader } : {})}
        {...(entry.respBody !== undefined ? { body: entry.respBody } : {})}
      />
    </>
  );
}

function Pre({
  label,
  header,
  body,
}: {
  label: string;
  header?: Record<string, string[]>;
  body?: string;
}) {
  return (
    <details className="raw-capture-pre">
      <summary>{label}</summary>
      {header ? (
        <pre className="raw-capture-header">
          {Object.entries(header)
            .map(([k, v]) => `${k}: ${v.join(", ")}`)
            .join("\n")}
        </pre>
      ) : null}
      {body ? <pre className="raw-capture-body">{body}</pre> : null}
    </details>
  );
}

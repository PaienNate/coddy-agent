export interface RawCaptureEntry {
  id: string;
  at: string;
  method: string;
  url: string;
  status?: number;
  reqHeader?: Record<string, string[]>;
  reqBody?: string;
  respHeader?: Record<string, string[]>;
  respBody?: string;
  err?: string;
}

export interface RawCaptureDocument {
  object: "coddy.raw_capture";
  enabled: boolean;
  entries: RawCaptureEntry[];
}

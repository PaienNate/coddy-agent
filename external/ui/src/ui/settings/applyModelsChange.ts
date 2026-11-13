/**
 * applyModelsChange replaces the `models` array in the settings doc and keeps the
 * default-model references in sync. When a logical model id is renamed in place
 * (the model list keeps the same length, one entry's `model` string changes),
 * every reference that used the old id — `agent.model`, `memory.model` and
 * `compaction.model` / `compaction.fallback_models` — is rewritten to the new id.
 * When a model is removed, a reference that named it is cleared (`model` becomes
 * an empty string; a `fallback_models` entry is dropped), so a config the
 * operator saved from the Settings page never points at a model that is gone,
 * which the server rejects with "not found in models list".
 *
 * A model added leaves the references alone. A rename and a removal that happen
 * in one edit are reconciled together.
 */
export function applyModelsChange(
  doc: Record<string, unknown>,
  nextModels: unknown[],
): Record<string, unknown> {
  const base: Record<string, unknown> = { ...doc, models: nextModels };

  const prevIds = modelIds(doc.models);
  const nextIds = modelIds(nextModels);

  // A rename is only meaningful positionally when the list keeps its length;
  // on add/remove the indices no longer line up with the entry they held.
  const renames = new Map<string, string>();
  if (prevIds.length === nextIds.length) {
    for (let i = 0; i < prevIds.length; i++) {
      const from = prevIds[i]!;
      const to = nextIds[i]!;
      if (from !== "" && from !== to) {
        renames.set(from, to);
      }
    }
  }

  // A removed id is one present before and absent now, always detected, so a
  // deletion on its own (and a rename mixed into it) reconciles the references.
  const nextSet = new Set(nextIds);
  const removed = new Set(
    prevIds.filter((id) => id !== "" && !nextSet.has(id)),
  );

  if (renames.size === 0 && removed.size === 0) {
    return base;
  }

  base.agent = renameModelRef(base.agent, renames, removed);
  base.memory = renameModelRef(base.memory, renames, removed);
  base.compaction = renameCompaction(base.compaction, renames, removed);
  return base;
}

function renameModelRef(
  section: unknown,
  renames: Map<string, string>,
  removed: Set<string>,
): unknown {
  if (!section || typeof section !== "object" || Array.isArray(section)) {
    return section;
  }
  const obj = section as Record<string, unknown>;
  const current = obj.model;
  if (typeof current !== "string") {
    return section;
  }
  const renamed = renames.get(current);
  if (renamed !== undefined) {
    return { ...obj, model: renamed };
  }
  if (removed.has(current)) {
    return { ...obj, model: "" };
  }
  return section;
}

function renameCompaction(
  section: unknown,
  renames: Map<string, string>,
  removed: Set<string>,
): unknown {
  if (!section || typeof section !== "object" || Array.isArray(section)) {
    return section;
  }
  const obj = section as Record<string, unknown>;
  let out: Record<string, unknown> | undefined;
  const current = obj.model;
  if (typeof current === "string") {
    const renamed = renames.get(current);
    if (renamed !== undefined) {
      out = { ...obj, model: renamed };
    } else if (removed.has(current)) {
      out = { ...obj, model: "" };
    }
  }
  if (Array.isArray(obj.fallback_models)) {
    const src = obj.fallback_models as unknown[];
    const fallback: unknown[] = [];
    let changed = false;
    for (const m of src) {
      if (typeof m !== "string") {
        fallback.push(m);
        continue;
      }
      const renamed = renames.get(m);
      if (renamed !== undefined) {
        fallback.push(renamed);
        changed = true;
      } else if (removed.has(m)) {
        changed = true;
      } else {
        fallback.push(m);
      }
    }
    if (changed) {
      out = { ...(out ?? obj), fallback_models: fallback };
    }
  }
  return out ?? section;
}

/**
 * applyProvidersChange replaces the `providers` array in the settings doc and
 * keeps the model ids in step. A provider name is the prefix of every model id
 * it serves (`provider/id`), so renaming a provider's `name` in place renames
 * every `models[].model` under it and rewrites the references that named such
 * a model. A provider added or removed leaves the references alone.
 */
export function applyProvidersChange(
  doc: Record<string, unknown>,
  nextProviders: unknown[],
): Record<string, unknown> {
  const base: Record<string, unknown> = { ...doc, providers: nextProviders };

  const prevNames = rowNames(doc.providers);
  const nextNames = rowNames(nextProviders);
  if (prevNames.length !== nextNames.length) {
    return base;
  }
  const renames = new Map<string, string>();
  for (let i = 0; i < prevNames.length; i++) {
    const from = prevNames[i]!;
    const to = nextNames[i]!;
    if (from !== "" && from !== to) {
      renames.set(from, to);
    }
  }
  if (renames.size === 0) {
    return base;
  }

  if (Array.isArray(doc.models)) {
    base.models = doc.models.map((row) => renameModelPrefix(row, renames));
  }
  base.agent = renameRefPrefix(base.agent, renames);
  base.memory = renameRefPrefix(base.memory, renames);
  base.compaction = renameCompactionPrefix(base.compaction, renames);
  return base;
}

function renameModelPrefix(
  row: unknown,
  renames: Map<string, string>,
): unknown {
  if (!row || typeof row !== "object" || Array.isArray(row)) {
    return row;
  }
  const obj = row as Record<string, unknown>;
  const model = obj.model;
  if (typeof model !== "string") {
    return row;
  }
  const copy = replaceProviderPrefix(model, renames);
  return copy === model ? row : { ...obj, model: copy };
}

function renameRefPrefix(
  section: unknown,
  renames: Map<string, string>,
): unknown {
  if (!section || typeof section !== "object" || Array.isArray(section)) {
    return section;
  }
  const obj = section as Record<string, unknown>;
  const model = obj.model;
  if (typeof model !== "string") {
    return section;
  }
  const copy = replaceProviderPrefix(model, renames);
  return copy === model ? section : { ...obj, model: copy };
}

function renameCompactionPrefix(
  section: unknown,
  renames: Map<string, string>,
): unknown {
  if (!section || typeof section !== "object" || Array.isArray(section)) {
    return section;
  }
  const obj = section as Record<string, unknown>;
  let out: Record<string, unknown> | undefined;
  const model = obj.model;
  if (typeof model === "string") {
    const copy = replaceProviderPrefix(model, renames);
    if (copy !== model) {
      out = { ...obj, model: copy };
    }
  }
  if (Array.isArray(obj.fallback_models)) {
    const fallback = (obj.fallback_models as unknown[]).map((m) =>
      typeof m === "string" ? replaceProviderPrefix(m, renames) : m,
    );
    const changed = fallback.some(
      (m, i) => m !== (obj.fallback_models as unknown[])[i],
    );
    if (changed) {
      out = { ...(out ?? obj), fallback_models: fallback };
    }
  }
  return out ?? section;
}

/** Replaces the `provider/` prefix of a model id with the renamed provider. */
function replaceProviderPrefix(
  model: string,
  renames: Map<string, string>,
): string {
  const slash = model.indexOf("/");
  if (slash <= 0) {
    return model;
  }
  const prefix = model.slice(0, slash);
  const renamed = renames.get(prefix);
  return renamed === undefined ? model : renamed + model.slice(slash);
}

function modelIds(models: unknown): string[] {
  if (!Array.isArray(models)) {
    return [];
  }
  return models.map((row) => {
    if (row && typeof row === "object" && !Array.isArray(row)) {
      const cell = (row as Record<string, unknown>).model;
      return cell === undefined || cell === null ? "" : String(cell);
    }
    return "";
  });
}

/**
 * applyToolsetsChange replaces the `toolsets` array in the settings doc and,
 * the way applyModelsChange keeps the model references, rewrites the
 * `models[].toolset` that named a preset renamed in place, so the saved
 * configuration does not name a toolset that is gone. A preset added or
 * removed leaves the references alone.
 */
export function applyToolsetsChange(
  doc: Record<string, unknown>,
  nextToolsets: unknown[],
): Record<string, unknown> {
  const base: Record<string, unknown> = { ...doc, toolsets: nextToolsets };
  const prev = rowNames(doc.toolsets);
  const next = rowNames(nextToolsets);
  if (prev.length !== next.length || !Array.isArray(doc.models)) {
    return base;
  }
  const renames = new Map<string, string>();
  for (let i = 0; i < prev.length; i++) {
    if (prev[i] !== "" && prev[i] !== next[i]) {
      renames.set(prev[i]!, next[i]!);
    }
  }
  if (renames.size === 0) {
    return base;
  }
  base.models = doc.models.map((row) => {
    if (!row || typeof row !== "object" || Array.isArray(row)) {
      return row;
    }
    const obj = row as Record<string, unknown>;
    const renamed =
      typeof obj.toolset === "string" ? renames.get(obj.toolset) : undefined;
    return renamed === undefined ? row : { ...obj, toolset: renamed };
  });
  return base;
}

function rowNames(rows: unknown): string[] {
  if (!Array.isArray(rows)) {
    return [];
  }
  return rows.map((row) => {
    if (row && typeof row === "object" && !Array.isArray(row)) {
      const cell = (row as Record<string, unknown>).name;
      return cell === undefined || cell === null ? "" : String(cell);
    }
    return "";
  });
}

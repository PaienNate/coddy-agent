import { describe, expect, it } from "vitest";
import {
  applyModelsChange,
  applyProvidersChange,
  applyToolsetsChange,
} from "./applyModelsChange";

function baseDoc(
  models: Array<Record<string, unknown>>,
  extra?: Record<string, unknown>,
): Record<string, unknown> {
  return { providers: [], models, ...extra };
}

describe("applyModelsChange", () => {
  it("updates agent.model when the sole model id is renamed", () => {
    const doc = baseDoc([{ model: "neuraldeep/gpt-120b-oss" }], {
      agent: { model: "neuraldeep/gpt-120b-oss", max_turns: 20 },
    });
    const next = applyModelsChange(doc, [{ model: "neuraldeep/qwen-3.6" }]);
    expect((next.agent as Record<string, unknown>).model).toBe(
      "neuraldeep/qwen-3.6",
    );
    // Unrelated agent fields preserved.
    expect((next.agent as Record<string, unknown>).max_turns).toBe(20);
    // Models replaced with the new array.
    expect(next.models).toEqual([{ model: "neuraldeep/qwen-3.6" }]);
  });

  it("updates memory.model referencing the same old id", () => {
    const doc = baseDoc([{ model: "neuraldeep/gpt-120b-oss" }], {
      agent: { model: "neuraldeep/gpt-120b-oss" },
      memory: { enable: true, model: "neuraldeep/gpt-120b-oss" },
    });
    const next = applyModelsChange(doc, [{ model: "neuraldeep/qwen-3.6" }]);
    expect((next.memory as Record<string, unknown>).model).toBe(
      "neuraldeep/qwen-3.6",
    );
    expect((next.memory as Record<string, unknown>).enable).toBe(true);
  });

  it("leaves memory.model pointing at a different id untouched", () => {
    const doc = baseDoc(
      [{ model: "neuraldeep/gpt-120b-oss" }, { model: "openai/gpt-4o" }],
      {
        agent: { model: "neuraldeep/gpt-120b-oss" },
        memory: { model: "openai/gpt-4o" },
      },
    );
    const next = applyModelsChange(doc, [
      { model: "neuraldeep/qwen-3.6" },
      { model: "openai/gpt-4o" },
    ]);
    expect((next.agent as Record<string, unknown>).model).toBe(
      "neuraldeep/qwen-3.6",
    );
    expect((next.memory as Record<string, unknown>).model).toBe(
      "openai/gpt-4o",
    );
  });

  it("leaves agent.model pointing at an unchanged model untouched", () => {
    const doc = baseDoc(
      [{ model: "neuraldeep/gpt-120b-oss" }, { model: "openai/gpt-4o" }],
      { agent: { model: "openai/gpt-4o" } },
    );
    const next = applyModelsChange(doc, [
      { model: "neuraldeep/qwen-3.6" },
      { model: "openai/gpt-4o" },
    ]);
    expect((next.agent as Record<string, unknown>).model).toBe("openai/gpt-4o");
  });

  it("treats a same-length delete-and-add as a removal, not a rename", () => {
    // [a, b] -> [b, c]: b survives, a is gone, c is new. A positional diff
    // would read a->b and b->c and repoint references to the wrong ids.
    const doc = baseDoc([{ model: "local/a" }, { model: "local/b" }], {
      agent: { model: "local/a" },
      memory: { model: "local/b" },
    });
    const next = applyModelsChange(doc, [
      { model: "local/b" },
      { model: "local/c" },
    ]);
    // The reference to the removed model is cleared, never repointed.
    expect((next.agent as Record<string, unknown>).model).toBe("");
    // The reference to the surviving model is untouched.
    expect((next.memory as Record<string, unknown>).model).toBe("local/b");
  });

  it("does not repoint references when a model is added (length grows)", () => {
    const doc = baseDoc([{ model: "neuraldeep/gpt-120b-oss" }], {
      agent: { model: "neuraldeep/gpt-120b-oss" },
    });
    const next = applyModelsChange(doc, [
      { model: "neuraldeep/gpt-120b-oss" },
      { model: "openai/gpt-4o" },
    ]);
    expect((next.agent as Record<string, unknown>).model).toBe(
      "neuraldeep/gpt-120b-oss",
    );
  });

  it("clears a reference when its model is removed (length shrinks)", () => {
    const doc = baseDoc(
      [{ model: "neuraldeep/gpt-120b-oss" }, { model: "openai/gpt-4o" }],
      {
        agent: { model: "neuraldeep/gpt-120b-oss" },
        memory: { model: "neuraldeep/gpt-120b-oss" },
      },
    );
    const next = applyModelsChange(doc, [{ model: "openai/gpt-4o" }]);
    // The removed model's references cannot stay: a config naming a model that
    // is gone fails validation ("not found in models list").
    expect((next.agent as Record<string, unknown>).model).toBe("");
    expect((next.memory as Record<string, unknown>).model).toBe("");
  });

  it("clears only the reference of a removed model, leaving others", () => {
    const doc = baseDoc(
      [{ model: "neuraldeep/gpt-120b-oss" }, { model: "openai/gpt-4o" }],
      {
        agent: { model: "openai/gpt-4o" },
        memory: { model: "neuraldeep/gpt-120b-oss" },
      },
    );
    const next = applyModelsChange(doc, [{ model: "openai/gpt-4o" }]);
    expect((next.agent as Record<string, unknown>).model).toBe("openai/gpt-4o");
    expect((next.memory as Record<string, unknown>).model).toBe("");
  });

  it("renames only the reference matching the changed entry", () => {
    const doc = baseDoc(
      [{ model: "neuraldeep/gpt-120b-oss" }, { model: "openai/gpt-4o" }],
      {
        agent: { model: "openai/gpt-4o" },
        memory: { model: "neuraldeep/gpt-120b-oss" },
      },
    );
    const next = applyModelsChange(doc, [
      { model: "neuraldeep/gpt-120b-oss" },
      { model: "openai/gpt-4o-mini" },
    ]);
    // agent referenced the renamed one -> updated.
    expect((next.agent as Record<string, unknown>).model).toBe(
      "openai/gpt-4o-mini",
    );
    // memory referenced the untouched one -> unchanged.
    expect((next.memory as Record<string, unknown>).model).toBe(
      "neuraldeep/gpt-120b-oss",
    );
  });

  it("ignores a rename from an empty id", () => {
    const doc = baseDoc([{ model: "" }], {
      agent: { model: "" },
    });
    const next = applyModelsChange(doc, [{ model: "neuraldeep/qwen-3.6" }]);
    // Empty ids are not treated as a rename source.
    expect((next.agent as Record<string, unknown>).model).toBe("");
  });

  it("returns the doc unchanged when there are no model references", () => {
    const doc = baseDoc([{ model: "neuraldeep/gpt-120b-oss" }]);
    const next = applyModelsChange(doc, [{ model: "neuraldeep/qwen-3.6" }]);
    expect(next.models).toEqual([{ model: "neuraldeep/qwen-3.6" }]);
    expect(next.agent).toBeUndefined();
  });

  it("follows compaction.model through a rename", () => {
    const doc = baseDoc([{ model: "neuraldeep/gpt-120b-oss" }], {
      compaction: {
        enable: true,
        model: "neuraldeep/gpt-120b-oss",
        fallback_models: ["neuraldeep/gpt-120b-oss"],
      },
    });
    const next = applyModelsChange(doc, [{ model: "neuraldeep/qwen-3.6" }]);
    expect((next.compaction as Record<string, unknown>).model).toBe(
      "neuraldeep/qwen-3.6",
    );
    expect(
      (next.compaction as Record<string, unknown>).fallback_models,
    ).toEqual(["neuraldeep/qwen-3.6"]);
    expect((next.compaction as Record<string, unknown>).enable).toBe(true);
  });

  it("clears compaction.model and drops its fallback entry on removal", () => {
    const doc = baseDoc(
      [{ model: "neuraldeep/gpt-120b-oss" }, { model: "openai/gpt-4o" }],
      {
        compaction: {
          model: "neuraldeep/gpt-120b-oss",
          fallback_models: ["neuraldeep/gpt-120b-oss", "openai/gpt-4o"],
        },
      },
    );
    const next = applyModelsChange(doc, [{ model: "openai/gpt-4o" }]);
    expect((next.compaction as Record<string, unknown>).model).toBe("");
    expect(
      (next.compaction as Record<string, unknown>).fallback_models,
    ).toEqual(["openai/gpt-4o"]);
  });
});

describe("applyProvidersChange", () => {
  it("renames every model id under a provider whose name changed, and the references", () => {
    const doc = baseDoc(
      [
        { model: "neuraldeep/gpt-120b-oss" },
        { model: "neuraldeep/gpt-4o" },
        { model: "openai/gpt-4o" },
      ],
      { agents: undefined },
    );
    doc.providers = [
      { name: "neuraldeep", type: "openai" },
      { name: "openai", type: "openai" },
    ];
    doc.agent = { model: "neuraldeep/gpt-120b-oss" };
    doc.memory = { model: "neuraldeep/gpt-4o" };
    doc.compaction = {
      model: "neuraldeep/gpt-120b-oss",
      fallback_models: ["neuraldeep/gpt-4o"],
    };
    const next = applyProvidersChange(doc, [
      { name: "deepseek", type: "openai" },
      { name: "openai", type: "openai" },
    ]);
    expect(next.providers).toEqual([
      { name: "deepseek", type: "openai" },
      { name: "openai", type: "openai" },
    ]);
    expect(next.models).toEqual([
      { model: "deepseek/gpt-120b-oss" },
      { model: "deepseek/gpt-4o" },
      { model: "openai/gpt-4o" },
    ]);
    expect((next.agent as Record<string, unknown>).model).toBe(
      "deepseek/gpt-120b-oss",
    );
    expect((next.memory as Record<string, unknown>).model).toBe(
      "deepseek/gpt-4o",
    );
    expect((next.compaction as Record<string, unknown>).model).toBe(
      "deepseek/gpt-120b-oss",
    );
    expect(
      (next.compaction as Record<string, unknown>).fallback_models,
    ).toEqual(["deepseek/gpt-4o"]);
  });

  it("leaves everything alone when a provider is added or removed", () => {
    const doc = baseDoc([{ model: "neuraldeep/gpt-120b-oss" }]);
    doc.providers = [{ name: "neuraldeep", type: "openai" }];
    doc.agent = { model: "neuraldeep/gpt-120b-oss" };
    expect(applyProvidersChange(doc, []).models).toBe(doc.models);
    expect(
      (applyProvidersChange(doc, []).agent as Record<string, unknown>).model,
    ).toBe("neuraldeep/gpt-120b-oss");
  });
});

describe("applyToolsetsChange", () => {
  it("renames the toolset of every model that named a preset renamed in place", () => {
    const doc = baseDoc(
      [
        { model: "local/qwen", toolset: "coding" },
        { model: "local/plain" },
        { model: "local/other", toolset: "no-net" },
      ],
      { toolsets: [{ name: "coding" }, { name: "no-net" }] },
    );
    const next = applyToolsetsChange(doc, [
      { name: "lean" },
      { name: "no-net" },
    ]);
    expect(next.toolsets).toEqual([{ name: "lean" }, { name: "no-net" }]);
    expect(next.models).toEqual([
      { model: "local/qwen", toolset: "lean" },
      { model: "local/plain" },
      { model: "local/other", toolset: "no-net" },
    ]);
  });

  it("leaves the references alone when a preset is added or removed", () => {
    const models = [{ model: "local/qwen", toolset: "coding" }];
    const doc = baseDoc(models, { toolsets: [{ name: "coding" }] });
    expect(applyToolsetsChange(doc, []).models).toBe(models);
    expect(
      applyToolsetsChange(doc, [{ name: "coding" }, { name: "x" }]).models,
    ).toBe(models);
  });
});

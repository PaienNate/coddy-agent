import React from "react";
import { afterEach, expect, test, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { ModelField } from "./ModelField";
import type { ProviderRow } from "./useProviderModels";
import type { OverrideTexts } from "./settingsSections";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

const PROVIDERS: ProviderRow[] = [
  { name: "openai", type: "openai", api_key: "sk-a" },
  { name: "hub", type: "neuraldeep" },
];

const TEXTS: OverrideTexts = {
  primary: "Model id",
  second: "Provider",
};

function Harness(props: { initial?: string }) {
  const [val, setVal] = React.useState(props.initial ?? "");
  return (
    <>
      <ModelField
        value={val}
        onChange={setVal}
        providers={PROVIDERS}
        texts={TEXTS}
      />
      <span data-testid="val">{val}</span>
    </>
  );
}

test("the provider combobox lists the document's provider names", () => {
  render(<Harness />);

  fireEvent.focus(screen.getByTestId("model-field-provider"));

  expect(screen.getByText("openai")).toBeTruthy();
  expect(screen.getByText("hub")).toBeTruthy();
});

test("picking a provider and typing the id composes provider/id", () => {
  render(<Harness />);

  fireEvent.focus(screen.getByTestId("model-field-provider"));
  fireEvent.mouseDown(screen.getByText("openai"));
  fireEvent.change(screen.getByTestId("model-field-model"), {
    target: { value: "gpt-4o-mini" },
  });

  expect(screen.getByTestId("val").textContent).toBe("openai/gpt-4o-mini");
});

test("a provider the document does not list can be typed", () => {
  render(<Harness />);

  fireEvent.change(screen.getByTestId("model-field-provider"), {
    target: { value: "custom" },
  });
  fireEvent.change(screen.getByTestId("model-field-model"), {
    target: { value: "my-model" },
  });

  expect(screen.getByTestId("val").textContent).toBe("custom/my-model");
});

test("an existing value splits into provider and id on the first slash", () => {
  render(<Harness initial="openrouter/meta/llama-3" />);

  expect(
    (screen.getByTestId("model-field-provider") as HTMLInputElement).value,
  ).toBe("openrouter");
  expect(
    (screen.getByTestId("model-field-model") as HTMLInputElement).value,
  ).toBe("meta/llama-3");
});

// The model id is a plain field: the provider form lists what a provider
// advertises, so this one neither fetches nor offers a list of its own.
test("the model id is a plain field that fetches nothing", () => {
  const fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
  render(<Harness initial="openai/gpt-4o" />);

  const id = screen.getByTestId("model-field-model");
  expect(id.tagName).toBe("INPUT");
  expect(id.getAttribute("role")).toBeNull();
  fireEvent.focus(id);

  expect(fetchMock).not.toHaveBeenCalled();
  expect(document.querySelector(".settings-combobox-list")).toBeNull();
  expect(
    document.querySelector(
      '[data-testid="model-field"] ~ .settings-row .settings-field-desc',
    ),
  ).toBeNull();
});

test("a provider's display name labels its option; the value stays the id", () => {
  function LabelHarness() {
    const [val, setVal] = React.useState("");
    return (
      <>
        <ModelField
          value={val}
          onChange={setVal}
          providers={[
            { name: "openai", type: "openai", display_name: "OpenAI (work)" },
            { name: "hub", type: "neuraldeep" },
          ]}
          texts={TEXTS}
        />
        <span data-testid="val">{val}</span>
      </>
    );
  }
  render(<LabelHarness />);

  fireEvent.focus(screen.getByTestId("model-field-provider"));

  // The option shows the label the row carries; a row without one keeps its id.
  expect(screen.getByText("OpenAI (work)")).toBeTruthy();
  expect(screen.getByText("hub")).toBeTruthy();

  fireEvent.mouseDown(screen.getByText("OpenAI (work)"));
  fireEvent.change(screen.getByTestId("model-field-model"), {
    target: { value: "gpt-4o" },
  });
  expect(screen.getByTestId("val").textContent).toBe("openai/gpt-4o");
});

test("the names are the texts it is given: the id's first, the provider's second", () => {
  render(
    <ModelField
      value="openai/gpt-4o"
      onChange={() => {}}
      providers={PROVIDERS}
      texts={{ primary: "Идентификатор модели", second: "Провайдер" }}
    />,
  );

  expect(screen.getByLabelText("Идентификатор модели")).toBe(
    screen.getByTestId("model-field-model"),
  );
  expect(screen.getByLabelText("Провайдер")).toBe(
    screen.getByTestId("model-field-provider"),
  );
  expect(screen.getByText("Идентификатор модели")).toBeTruthy();
  expect(screen.getByText("Провайдер")).toBeTruthy();
});

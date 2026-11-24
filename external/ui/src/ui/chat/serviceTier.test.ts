import { describe, expect, it } from "vitest";
import {
  pickServiceTier,
  readServiceTierCookie,
  serviceTierLabelKey,
  sessionTierChoiceList,
  writeServiceTierCookie,
} from "./serviceTier";

describe("service tier preference", () => {
  it("ignores a malformed stored value", () => {
    document.cookie = "coddy_service_tier=%; Path=/";
    expect(readServiceTierCookie()).toBe("");
    writeServiceTierCookie("");
  });
  it("keeps an offered choice and drops an incompatible one", () => {
    expect(pickServiceTier("priority", ["", "default", "priority"])).toBe(
      "priority",
    );
    expect(pickServiceTier("ultrafast", ["", "default"])).toBe("");
    expect(pickServiceTier("priority", [])).toBe("");
  });
  it("remembers an explicit reset as well as Standard", () => {
    writeServiceTierCookie("default");
    expect(readServiceTierCookie()).toBe("default");
    writeServiceTierCookie("");
    expect(readServiceTierCookie()).toBe("");
  });
  it("shows an unset session as Standard", () => {
    expect(serviceTierLabelKey("")).toBe("composer.serviceTier.default");
    expect(serviceTierLabelKey("priority")).toBe(
      "composer.serviceTier.priority",
    );
  });
  // A snapshot that names no list must not blank the page: the composer handed
  // the value straight to `.filter` (issue: white screen on a model switch).
  it("falls back to the model's tiers when a snapshot names none", () => {
    expect(sessionTierChoiceList(undefined, ["", "priority"])).toEqual([
      "",
      "priority",
    ]);
    expect(sessionTierChoiceList(null as never, ["default"])).toEqual([
      "default",
    ]);
    expect(sessionTierChoiceList("priority" as never, [])).toEqual([]);
  });
  it("keeps the session's own list, empty and unclean included", () => {
    expect(sessionTierChoiceList(["", "default"], ["priority"])).toEqual([
      "",
      "default",
    ]);
    // An empty list is the session's answer ("this model offers no tier"), not
    // a missing one, so the model row does not take the chip back.
    expect(sessionTierChoiceList([], ["priority"])).toEqual([]);
    expect(sessionTierChoiceList(["", 7 as never, "priority"], [])).toEqual([
      "",
      "priority",
    ]);
  });
});

import { vi } from "vitest";
import { afterEach, beforeEach } from "vitest";
import { describe, expect, test } from "vitest";

const readRawCapture = vi.fn();
const clearRawCapture = vi.fn();

vi.mock("./api", () => ({
  readRawCapture: (...args: unknown[]) => readRawCapture(...args),
  clearRawCapture: (...args: unknown[]) => clearRawCapture(...args),
}));

import {
  cleanup,
  render,
  screen,
  fireEvent,
  waitFor,
} from "@testing-library/react";
import { RawCaptureView } from "./RawCaptureView";

describe("RawCaptureView", () => {
  beforeEach(() => {
    readRawCapture.mockReset();
    clearRawCapture.mockReset();
  });
  afterEach(() => cleanup());

  test("renders nothing while the server reports capture off", async () => {
    readRawCapture.mockResolvedValue({
      ok: true,
      data: { object: "coddy.raw_capture", enabled: false, entries: [] },
    });
    const { container } = render(<RawCaptureView />);
    await waitFor(() => expect(readRawCapture).toHaveBeenCalled());
    expect(container.firstChild).toBeNull();
  });

  test("lists captured exchanges once enabled", async () => {
    readRawCapture.mockResolvedValue({
      ok: true,
      data: {
        object: "coddy.raw_capture",
        enabled: true,
        entries: [
          {
            id: "raw_1",
            at: "2026-01-01T00:00:00Z",
            method: "POST",
            url: "https://api.example.com/v1/chat",
            status: 200,
            reqHeader: { "Content-Type": ["application/json"] },
            reqBody: '{"hello":"world"}',
            respHeader: { "Content-Type": ["application/json"] },
            respBody: '{"ok":true}',
          },
        ],
      },
    });
    render(<RawCaptureView />);
    await waitFor(() =>
      expect(
        screen.getByText("https://api.example.com/v1/chat"),
      ).toBeInTheDocument(),
    );
    expect(screen.getByText("POST")).toBeInTheDocument();
    expect(screen.getByText("200")).toBeInTheDocument();
    expect(screen.getByText('{"hello":"world"}')).toBeInTheDocument();
    expect(screen.getByText('{"ok":true}')).toBeInTheDocument();
  });

  test("an empty capture shows the empty note and a disabled clear", async () => {
    readRawCapture.mockResolvedValue({
      ok: true,
      data: { object: "coddy.raw_capture", enabled: true, entries: [] },
    });
    render(<RawCaptureView />);
    await waitFor(() =>
      expect(screen.getByText(/Nothing captured yet/i)).toBeInTheDocument(),
    );
    expect(screen.getByRole("button", { name: /clear/i })).toBeDisabled();
  });

  test("clear asks the server and refreshes the list", async () => {
    readRawCapture
      .mockResolvedValueOnce({
        ok: true,
        data: {
          object: "coddy.raw_capture",
          enabled: true,
          entries: [
            {
              id: "raw_1",
              at: "2026-01-01T00:00:00Z",
              method: "POST",
              url: "https://api.example.com/v1",
              status: 200,
            },
          ],
        },
      })
      .mockResolvedValueOnce({
        ok: true,
        data: { object: "coddy.raw_capture", enabled: true, entries: [] },
      });
    clearRawCapture.mockResolvedValue({
      ok: true,
      data: { object: "coddy.raw_capture", enabled: true, entries: [] },
    });
    render(<RawCaptureView />);
    await waitFor(() =>
      expect(
        screen.getByText("https://api.example.com/v1"),
      ).toBeInTheDocument(),
    );
    fireEvent.click(screen.getByRole("button", { name: /clear/i }));
    await waitFor(() => expect(clearRawCapture).toHaveBeenCalled());
    await waitFor(() =>
      expect(screen.getByText(/Nothing captured yet/i)).toBeInTheDocument(),
    );
  });
});

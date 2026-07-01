import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  CODE_SEARCH_DEBOUNCE_MS,
  useCodeIntelFileContent,
  useCodeIntelStatus,
  useCodeSearch,
} from "./useCodeIntel";

const fetchMock = vi.fn();

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? "OK" : "Error",
    json: async () => body,
  } as unknown as Response;
}

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("useCodeIntelStatus", () => {
  it("fetches the session index status", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        repo_root: "/repo",
        indexed: true,
        status: "ready",
        nodes: 10,
        edges: 20,
        head: { branch: "main", head_sha: "abc", base_sha: "def" },
        project: "proj",
        message: null,
      }),
    );

    const { result } = renderHook(() => useCodeIntelStatus("conv1"), { wrapper });

    await waitFor(() => expect(result.current.data?.indexed).toBe(true));
    expect(result.current.data?.status).toBe("ready");
    const url = fetchMock.mock.calls[0][0] as string;
    expect(url).toContain("/v1/sessions/conv1/code-intel/status");
  });

  it("stays idle when sessionId is null", () => {
    renderHook(() => useCodeIntelStatus(null), { wrapper });
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

describe("useCodeSearch", () => {
  it("does not fetch for queries shorter than the minimum", () => {
    renderHook(() => useCodeSearch("conv1", "a"), { wrapper });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("fetches results for a long-enough query", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        repo_root: "/repo",
        query: "widget",
        status: "ok",
        total: 1,
        results: [
          {
            name: "widget",
            qualified_name: "pkg.widget",
            label: "Function",
            file: "pkg/mod.py",
            signature: "()",
            return_type: "int",
          },
        ],
        message: null,
      }),
    );

    const { result } = renderHook(() => useCodeSearch("conv1", "widget"), { wrapper });

    await waitFor(() => expect(result.current.data?.total).toBe(1), { timeout: 2000 });
    expect(result.current.data?.results[0].qualified_name).toBe("pkg.widget");
    const url = fetchMock.mock.calls[0][0] as string;
    expect(url).toContain("/code-intel/search?");
    expect(new URL(url, "http://test").searchParams.get("q")).toBe("widget");
  });

  it("debounces rapid query changes", async () => {
    vi.useFakeTimers();
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        repo_root: "/repo",
        query: "widget",
        status: "ok",
        total: 0,
        results: [],
        message: null,
      }),
    );

    const { rerender } = renderHook(({ query }) => useCodeSearch("conv1", query), {
      initialProps: { query: "wi" },
      wrapper,
    });
    rerender({ query: "wid" });
    rerender({ query: "widget" });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(CODE_SEARCH_DEBOUNCE_MS - 1);
    });
    expect(fetchMock).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    vi.useRealTimers();
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const url = fetchMock.mock.calls[0][0] as string;
    expect(new URL(url, "http://test").searchParams.get("q")).toBe("widget");
  });
});

describe("useCodeIntelFileContent", () => {
  it("fetches repo-relative file content", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        repo_root: "/repo",
        path: "pkg/mod.py",
        size_bytes: 12,
        truncated: false,
        content: "print('ok')\n",
      }),
    );

    const { result } = renderHook(() => useCodeIntelFileContent("conv1", "pkg/mod.py"), {
      wrapper,
    });

    await waitFor(() => expect(result.current.data?.content).toContain("print"));
    const url = fetchMock.mock.calls[0][0] as string;
    expect(url).toContain("/v1/sessions/conv1/code-intel/files/pkg/mod.py");
  });

  it("stays idle when path is null", () => {
    renderHook(() => useCodeIntelFileContent("conv1", null), { wrapper });
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import { CodeIntelPanel } from "./CodeIntelPanel";

const fetchMock = vi.fn();

function jsonResponse(body: unknown): Response {
  return {
    ok: true,
    status: 200,
    statusText: "OK",
    json: async () => body,
  } as unknown as Response;
}

function renderPanel(): void {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const ui: ReactNode = (
    <QueryClientProvider client={qc}>
      <CodeIntelPanel conversationId="conv1" />
    </QueryClientProvider>
  );
  render(ui);
}

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

it("renders a ready status with counts", async () => {
  fetchMock.mockResolvedValue(
    jsonResponse({
      repo_root: "/repo",
      indexed: true,
      status: "ready",
      nodes: 1234,
      edges: 5678,
      head: { branch: "main", head_sha: "abcdef0123", base_sha: null },
      project: "proj",
      message: null,
    }),
  );

  renderPanel();

  await waitFor(() => expect(screen.getByText("Ready")).toBeInTheDocument());
  expect(screen.getByText(/nodes/)).toBeInTheDocument();
  expect(screen.getByText("main")).toBeInTheDocument();
  expect(screen.getByPlaceholderText("Search symbols…")).toBeInTheDocument();
});

it("renders the not-indexed state", async () => {
  fetchMock.mockResolvedValue(
    jsonResponse({
      repo_root: "/repo",
      indexed: false,
      status: "not_indexed",
      nodes: null,
      edges: null,
      head: null,
      project: null,
      message: null,
    }),
  );

  renderPanel();

  await waitFor(() => expect(screen.getByText("Not indexed")).toBeInTheDocument());
});

it("searches symbols and previews selected source", async () => {
  fetchMock.mockImplementation((url: string) => {
    if (url.includes("/status")) {
      return Promise.resolve(
        jsonResponse({
          repo_root: "/repo",
          indexed: true,
          status: "ready",
          nodes: 10,
          edges: 20,
          head: { branch: "main", head_sha: "abcdef", base_sha: null },
          project: "proj",
          message: null,
        }),
      );
    }
    if (url.includes("/search?")) {
      return Promise.resolve(
        jsonResponse({
          repo_root: "/repo",
          query: "CodeIntelClient",
          status: "ok",
          total: 1,
          results: [
            {
              name: "CodeIntelClient",
              qualified_name: "goalrail.code_intel.client.CodeIntelClient",
              label: "Class",
              file: "goalrail/code_intel/client.py",
              signature: null,
              return_type: null,
            },
          ],
          message: null,
        }),
      );
    }
    if (url.includes("/files/goalrail/code_intel/client.py")) {
      return Promise.resolve(
        jsonResponse({
          repo_root: "/repo",
          path: "goalrail/code_intel/client.py",
          size_bytes: 24,
          truncated: false,
          content: "class CodeIntelClient:\n    pass\n",
        }),
      );
    }
    throw new Error(`unexpected fetch ${url}`);
  });

  renderPanel();

  await waitFor(() => expect(screen.getByText("Ready")).toBeInTheDocument());
  fireEvent.change(screen.getByPlaceholderText("Search symbols…"), {
    target: { value: "CodeIntelClient" },
  });
  await waitFor(() => expect(screen.getByText("CodeIntelClient")).toBeInTheDocument());
  fireEvent.click(screen.getByRole("button", { name: /ClassCodeIntelClient/ }));

  await waitFor(() => expect(screen.getByText(/class CodeIntelClient/)).toBeInTheDocument());
  expect(
    fetchMock.mock.calls.some(([url]) =>
      String(url).includes("/files/goalrail/code_intel/client.py"),
    ),
  ).toBe(true);
});

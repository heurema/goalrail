// TanStack Query wrappers for the session-scoped code-intelligence API.
//
// The repository is resolved server-side from the session workspace, so
// these hooks only need the session id — never a filesystem path. They
// back the right-rail "Code" tab:
//   - `useCodeIntelStatus` -> GET /v1/sessions/{id}/code-intel/status
//   - `useCodeSearch`      -> GET /v1/sessions/{id}/code-intel/search?q=
//   - `useCodeIntelFileContent` -> GET /v1/sessions/{id}/code-intel/files/{path}

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { authenticatedFetch } from "@/lib/identity";

/** Head/freshness info from the index status (git block). */
export interface CodeIntelHead {
  branch: string | null;
  head_sha: string | null;
  base_sha: string | null;
}

/** Wire shape of `GET .../code-intel/status`. */
export interface CodeIntelStatus {
  repo_root: string;
  indexed: boolean;
  /** "ready" | "not_indexed" | "engine_unavailable" | engine-specific. */
  status: string;
  nodes: number | null;
  edges: number | null;
  head: CodeIntelHead | null;
  project: string | null;
  message: string | null;
}

/** One symbol hit from `GET .../code-intel/search`. */
export interface CodeSearchHit {
  name: string;
  qualified_name: string;
  label: string;
  file: string;
  signature: string | null;
  return_type: string | null;
}

/** Wire shape of `GET .../code-intel/search`. */
export interface CodeSearchResponse {
  repo_root: string;
  query: string;
  /** "ok" | "not_indexed" | "engine_unavailable". */
  status: string;
  total: number;
  results: CodeSearchHit[];
  message: string | null;
}

/** Wire shape of `GET .../code-intel/files/{path}`. */
export interface CodeIntelFileContent {
  repo_root: string;
  path: string;
  size_bytes: number;
  truncated: boolean;
  content: string;
}

/** Minimum query length before a search request is issued. */
export const CODE_SEARCH_MIN_QUERY = 2;
export const CODE_SEARCH_DEBOUNCE_MS = 250;

async function fetchStatus(sessionId: string): Promise<CodeIntelStatus> {
  const res = await authenticatedFetch(
    `/v1/sessions/${encodeURIComponent(sessionId)}/code-intel/status`,
  );
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return (await res.json()) as CodeIntelStatus;
}

/**
 * Fetch the code-intel index status for a session's repository.
 *
 * @param sessionId The owning session id, or `null` to stay idle.
 */
export function useCodeIntelStatus(sessionId: string | null) {
  return useQuery({
    queryKey: ["code-intel-status", sessionId],
    queryFn: () => fetchStatus(sessionId as string),
    enabled: sessionId !== null,
  });
}

async function fetchSearch(
  sessionId: string,
  query: string,
  limit: number,
  signal?: AbortSignal,
): Promise<CodeSearchResponse> {
  const params = new URLSearchParams({ q: query, limit: String(limit) });
  const res = await authenticatedFetch(
    `/v1/sessions/${encodeURIComponent(sessionId)}/code-intel/search?${params.toString()}`,
    { signal },
  );
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return (await res.json()) as CodeSearchResponse;
}

/**
 * Search the session repo's knowledge graph for symbols by name.
 *
 * Stays idle until `query` is at least {@link CODE_SEARCH_MIN_QUERY}
 * characters, so partial typing doesn't spam the engine.
 *
 * @param sessionId The owning session id, or `null` to stay idle.
 * @param query The symbol name pattern (trimmed by the caller).
 * @param limit Max hits to request.
 */
export function useCodeSearch(sessionId: string | null, query: string, limit = 20) {
  const trimmed = query.trim();
  const [debounced, setDebounced] = useState("");
  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(trimmed), CODE_SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [trimmed]);
  const enabled = sessionId !== null && debounced.length >= CODE_SEARCH_MIN_QUERY;
  return useQuery({
    queryKey: ["code-intel-search", sessionId, debounced, limit],
    queryFn: ({ signal }) => fetchSearch(sessionId as string, debounced, limit, signal),
    enabled,
  });
}

async function fetchFileContent(sessionId: string, path: string): Promise<CodeIntelFileContent> {
  const encodedPath = path
    .split("/")
    .map((part) => encodeURIComponent(part))
    .join("/");
  const res = await authenticatedFetch(
    `/v1/sessions/${encodeURIComponent(sessionId)}/code-intel/files/${encodedPath}`,
  );
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return (await res.json()) as CodeIntelFileContent;
}

/**
 * Read source text for a repo-relative path through the code-intel API.
 *
 * This intentionally does not use the general session filesystem resource API:
 * the Code tab is backed by the session workspace and can work even when the
 * session is not currently bound to a runner.
 */
export function useCodeIntelFileContent(sessionId: string | null, path: string | null) {
  return useQuery({
    queryKey: ["code-intel-file", sessionId, path],
    queryFn: () => fetchFileContent(sessionId as string, path as string),
    enabled: sessionId !== null && path !== null,
  });
}

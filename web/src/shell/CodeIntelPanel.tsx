// "Code" tab content for the right-side rail. Shows the code-intel
// index status (ready / not indexed / indexing / engine unavailable),
// head/branch freshness, and a symbol search whose results link into
// an inline read-only preview. The repository is resolved server-side from the
// session workspace — this component only knows the conversation id.
//
// Graph visualization is intentionally out of scope for this slice: a
// stable status + search contract comes first.

import { useState } from "react";
import { Code2Icon, GitBranchIcon, SearchIcon, XIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  CODE_SEARCH_MIN_QUERY,
  type CodeSearchHit,
  type CodeIntelStatus,
  useCodeIntelFileContent,
  useCodeIntelStatus,
  useCodeSearch,
} from "@/hooks/useCodeIntel";

interface CodeIntelPanelProps {
  /** Active conversation id (the session whose repo is indexed). */
  conversationId: string;
}

interface StatusBadge {
  label: string;
  className: string;
}

/** Map an index-status string to a human label + tone classes. */
function statusBadge(status: CodeIntelStatus): StatusBadge {
  switch (status.status) {
    case "ready":
      return { label: "Ready", className: "bg-success/15 text-success" };
    case "indexing":
      return { label: "Indexing", className: "bg-info/15 text-info" };
    case "stale":
      return { label: "Stale", className: "bg-warning/15 text-warning" };
    case "not_indexed":
      return { label: "Not indexed", className: "bg-muted text-muted-foreground" };
    case "engine_unavailable":
      return {
        label: "Engine unavailable",
        className: "bg-destructive/15 text-destructive",
      };
    case "host_unsupported":
      return { label: "Host workspace", className: "bg-muted text-muted-foreground" };
    default:
      return { label: status.status, className: "bg-muted text-muted-foreground" };
  }
}

function StatusHeader({ conversationId }: { conversationId: string }) {
  const { data, isLoading, isError } = useCodeIntelStatus(conversationId);

  if (isLoading) {
    return <div className="px-3 py-2 text-[12px] text-muted-foreground">Checking index…</div>;
  }
  if (isError || !data) {
    return (
      <div className="px-3 py-2 text-[12px] text-destructive">Failed to load index status.</div>
    );
  }

  const badge = statusBadge(data);
  return (
    <div className="flex flex-col gap-1 border-b border-border px-3 py-2">
      <div className="flex items-center gap-2">
        <span
          className={cn(
            "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium",
            badge.className,
          )}
        >
          {badge.label}
        </span>
        {data.indexed && data.nodes !== null && (
          <span className="text-[11px] tabular-nums text-muted-foreground">
            {data.nodes.toLocaleString()} nodes · {(data.edges ?? 0).toLocaleString()} edges
          </span>
        )}
      </div>
      {data.head?.branch && (
        <div className="flex items-center gap-1 text-[11px] text-muted-foreground">
          <GitBranchIcon className="size-3" />
          <span className="truncate">{data.head.branch}</span>
          {data.head.head_sha && (
            <span className="tabular-nums">@ {data.head.head_sha.slice(0, 8)}</span>
          )}
        </div>
      )}
      {data.message && <div className="text-[11px] text-muted-foreground">{data.message}</div>}
    </div>
  );
}

function SearchSection({ conversationId }: { conversationId: string }) {
  const [query, setQuery] = useState("");
  const [selectedHit, setSelectedHit] = useState<CodeSearchHit | null>(null);
  const trimmed = query.trim();
  const { data, isFetching, isError } = useCodeSearch(conversationId, trimmed);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="border-b border-border px-3 py-2">
        <div className="flex items-center gap-2 rounded-md border border-border bg-background px-2">
          <SearchIcon className="size-4 shrink-0 text-muted-foreground" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search symbols…"
            spellCheck={false}
            className="h-8 w-full bg-transparent text-[13px] outline-none placeholder:text-muted-foreground"
          />
        </div>
      </div>

      <FilePreview
        conversationId={conversationId}
        selectedHit={selectedHit}
        onClose={() => setSelectedHit(null)}
      />

      <div className="min-h-0 flex-1 overflow-y-auto">
        {trimmed.length < CODE_SEARCH_MIN_QUERY ? (
          <p className="px-3 py-2 text-[12px] text-muted-foreground">
            Type at least {CODE_SEARCH_MIN_QUERY} characters to search.
          </p>
        ) : isError ? (
          <p className="px-3 py-2 text-[12px] text-destructive">Search failed.</p>
        ) : isFetching && !data ? (
          <p className="px-3 py-2 text-[12px] text-muted-foreground">Searching…</p>
        ) : data && data.status === "not_indexed" ? (
          <p className="px-3 py-2 text-[12px] text-muted-foreground">
            Repository is not indexed yet.
          </p>
        ) : data && data.status === "engine_unavailable" ? (
          <p className="px-3 py-2 text-[12px] text-destructive">
            Code-intel engine is unavailable.
          </p>
        ) : data && data.status === "host_unsupported" ? (
          <p className="px-3 py-2 text-[12px] text-muted-foreground">
            Code intelligence is not available for host workspaces yet.
          </p>
        ) : data && data.results.length === 0 ? (
          <p className="px-3 py-2 text-[12px] text-muted-foreground">No matches.</p>
        ) : (
          <ul>
            {data?.results.map((hit) => (
              <li key={hit.qualified_name}>
                <button
                  type="button"
                  onClick={() => setSelectedHit(hit)}
                  className={cn(
                    "flex w-full flex-col items-start gap-0.5 border-b border-border px-3 py-2 text-left hover:bg-muted/50",
                    selectedHit?.qualified_name === hit.qualified_name && "bg-muted/50",
                  )}
                >
                  <span className="flex items-center gap-2">
                    <span
                      className={cn(
                        "inline-flex shrink-0 items-center rounded px-1.5 py-px text-[10px]",
                        "bg-muted text-muted-foreground",
                      )}
                    >
                      {hit.label}
                    </span>
                    <span className="truncate text-[13px] font-medium">{hit.name}</span>
                  </span>
                  <span className="truncate text-[11px] text-muted-foreground">{hit.file}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function FilePreview({
  conversationId,
  selectedHit,
  onClose,
}: {
  conversationId: string;
  selectedHit: CodeSearchHit | null;
  onClose: () => void;
}) {
  const { data, isLoading, isError } = useCodeIntelFileContent(
    conversationId,
    selectedHit?.file ?? null,
  );

  if (selectedHit === null) return null;

  return (
    <div className="flex max-h-[45%] min-h-[160px] flex-col border-b border-border bg-card">
      <div className="flex items-center gap-2 border-b border-border px-3 py-2">
        <div className="min-w-0 flex-1">
          <div className="truncate text-[12px] font-medium">{selectedHit.name}</div>
          <div className="truncate text-[11px] text-muted-foreground">{selectedHit.file}</div>
        </div>
        <button
          type="button"
          aria-label="Close code preview"
          onClick={onClose}
          className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          <XIcon className="size-4" />
        </button>
      </div>
      {isLoading ? (
        <p className="px-3 py-2 text-[12px] text-muted-foreground">Loading file…</p>
      ) : isError || !data ? (
        <p className="px-3 py-2 text-[12px] text-destructive">Failed to load file.</p>
      ) : (
        <div className="min-h-0 flex-1 overflow-auto">
          {data.truncated && (
            <div className="border-b border-border px-3 py-1 text-[11px] text-warning">
              Preview truncated at 256 KiB.
            </div>
          )}
          <pre className="whitespace-pre-wrap px-3 py-2 font-mono text-[11px] leading-5 text-foreground">
            {data.content}
          </pre>
        </div>
      )}
    </div>
  );
}

/** Right-rail "Code" tab: index status + symbol search. */
export function CodeIntelPanel({ conversationId }: CodeIntelPanelProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex items-center gap-2 px-3 py-2 text-[12px] font-medium text-muted-foreground">
        <Code2Icon className="size-4" />
        Code intelligence
      </div>
      <StatusHeader conversationId={conversationId} />
      <SearchSection conversationId={conversationId} />
    </div>
  );
}

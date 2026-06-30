import {
  AlertTriangleIcon,
  ArrowLeftIcon,
  BarChart3Icon,
  BotIcon,
  CalendarDaysIcon,
  CheckCircle2Icon,
  CircleHelpIcon,
  Clock3Icon,
  Code2Icon,
  DollarSignIcon,
  FileTextIcon,
  GitBranchIcon,
  GitPullRequestIcon,
  MessageSquareTextIcon,
  SearchIcon,
  ShieldCheckIcon,
  SparklesIcon,
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import type { ComponentType } from "react";

import { PageScroll } from "@/components/PageScroll";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { listProjects } from "@/lib/projectsApi";
import type { ProjectSummary } from "@/lib/projectsApi";
import { relativeTime } from "@/lib/relativeTime";
import { Link, useParams } from "@/lib/routing";
import { cn } from "@/lib/utils";

type Tone = "success" | "warning" | "info" | "muted" | "danger";

interface BriefItem {
  title: string;
  evidence: string;
  tone: Tone;
}

interface ProjectSession {
  title: string;
  owner: string;
  agent: string;
  status: string;
  lastActivity: string;
  cost: string;
  evidence: string;
}

interface ProjectMock {
  id: string;
  name: string;
  workspace: string;
  description: string;
  lastActivity: string;
  sessionsThisWeek: number;
  contributors: number;
  costTotalUsd: number | null;
  sessionsWithUsage: number;
  costSessionsTotal: number;
  awaitingInput: number;
  failedSessions: number;
  branch: string;
  observed: BriefItem[];
  risks: BriefItem[];
  questions: BriefItem[];
  sessions: ProjectSession[];
}

const projectDetails: ProjectMock[] = [
  {
    id: "web-app",
    name: "web-app",
    workspace: "/Users/limerc/repos/personal/goalrail/ap-web",
    description: "Primary product shell, chat workspace, file viewer, and collaboration UI.",
    lastActivity: "14m ago",
    sessionsThisWeek: 21,
    contributors: 4,
    costTotalUsd: 31.7,
    sessionsWithUsage: 18,
    costSessionsTotal: 21,
    awaitingInput: 2,
    failedSessions: 0,
    branch: "main",
    observed: [
      {
        title: "Code rail work is visible across app shell sessions",
        evidence: "6 sessions, 18 file references",
        tone: "success",
      },
      {
        title: "Project navigation and route planning appeared in design notes",
        evidence: "2 notes, 4 linked sessions",
        tone: "info",
      },
      {
        title: "Theme rebuild changed the shipped bundle",
        evidence: "build log, static asset diff",
        tone: "success",
      },
    ],
    risks: [
      {
        title: "Project identity is still path-based",
        evidence: "no normalized remote stored yet",
        tone: "warning",
      },
      {
        title: "Five code-changing sessions need explicit follow-up",
        evidence: "session metadata only",
        tone: "warning",
      },
    ],
    questions: [
      {
        title: "Should project pages live beside Sessions or under Settings?",
        evidence: "open product decision",
        tone: "info",
      },
      {
        title: "What evidence qualifies an item as resolved?",
        evidence: "needs rules before LLM briefs",
        tone: "muted",
      },
    ],
    sessions: [
      {
        title: "Add Code rail tab with status search",
        owner: "Vitaly",
        agent: "Claude Code",
        status: "merged upstream",
        lastActivity: "2h ago",
        cost: "$8.42",
        evidence: "tests + openapi",
      },
      {
        title: "Sketch Project Intelligence MVP",
        owner: "limerc",
        agent: "Codex",
        status: "in discussion",
        lastActivity: "18m ago",
        cost: "$2.10",
        evidence: "design notes",
      },
      {
        title: "Rebuild web UI assets",
        owner: "limerc",
        agent: "Codex",
        status: "verified locally",
        lastActivity: "7m ago",
        cost: "$0.34",
        evidence: "bundle contains Code",
      },
    ],
  },
  {
    id: "api-server",
    name: "api-server",
    workspace: "/Users/limerc/repos/personal/goalrail/goalrail/server",
    description: "FastAPI routes, session stores, permissions, host presence, and OpenAPI.",
    lastActivity: "2h ago",
    sessionsThisWeek: 12,
    contributors: 3,
    costTotalUsd: 18.4,
    sessionsWithUsage: 9,
    costSessionsTotal: 12,
    awaitingInput: 3,
    failedSessions: 1,
    branch: "main",
    observed: [
      {
        title: "Code-intel API routes were added for session scoped search",
        evidence: "3 routes, tests present",
        tone: "success",
      },
      {
        title: "Host-bound workspaces report unsupported state instead of failing",
        evidence: "route tests, UI state",
        tone: "success",
      },
      {
        title: "OpenAPI schema changed for code-intel endpoints",
        evidence: "openapi diff",
        tone: "info",
      },
    ],
    risks: [
      {
        title: "Host-side code-intel execution is not implemented",
        evidence: "explicit host_unsupported state",
        tone: "warning",
      },
      {
        title: "Project grouping needs repo remote normalization",
        evidence: "current MVP uses workspace path",
        tone: "warning",
      },
      {
        title: "No durable project table in first slice",
        evidence: "aggregation-only proposal",
        tone: "muted",
      },
    ],
    questions: [
      {
        title: "Should projects expose permissions separately from sessions?",
        evidence: "project membership not scoped yet",
        tone: "info",
      },
      {
        title: "How should archived sessions affect weekly activity?",
        evidence: "needs product rule",
        tone: "muted",
      },
    ],
    sessions: [
      {
        title: "Report host unsupported workspaces",
        owner: "Vitaly",
        agent: "Claude Code",
        status: "merged upstream",
        lastActivity: "2h ago",
        cost: "$4.90",
        evidence: "route tests",
      },
      {
        title: "Add native status and search tools",
        owner: "Vitaly",
        agent: "Claude Code",
        status: "merged upstream",
        lastActivity: "1d ago",
        cost: "$10.02",
        evidence: "tool tests",
      },
      {
        title: "Plan project aggregation endpoints",
        owner: "limerc",
        agent: "Codex",
        status: "proposed",
        lastActivity: "34m ago",
        cost: "$1.16",
        evidence: "MVP scope",
      },
    ],
  },
  {
    id: "desktop-shell",
    name: "desktop-shell",
    workspace: "/Users/limerc/repos/personal/goalrail/app-desktop",
    description: "Desktop packaging, icons, setup window, and native app release path.",
    lastActivity: "yesterday",
    sessionsThisWeek: 8,
    contributors: 2,
    costTotalUsd: null,
    sessionsWithUsage: 0,
    costSessionsTotal: 8,
    awaitingInput: 0,
    failedSessions: 0,
    branch: "release-prep",
    observed: [
      {
        title: "Goalrail icon assets were replaced across desktop and web",
        evidence: "asset diff",
        tone: "success",
      },
      {
        title: "Setup and find windows moved to the Dracula palette",
        evidence: "CSS + HTML diff",
        tone: "info",
      },
      {
        title: "Release packaging instructions were updated",
        evidence: "release docs",
        tone: "info",
      },
    ],
    risks: [
      {
        title: "Visual snapshot coverage changed with icon and theme updates",
        evidence: "snapshot diff",
        tone: "warning",
      },
    ],
    questions: [
      {
        title: "Do desktop screenshots need a separate review pass?",
        evidence: "not visible from server UI",
        tone: "muted",
      },
    ],
    sessions: [
      {
        title: "Replace Goalrail app icon assets",
        owner: "Vitaly",
        agent: "Claude Code",
        status: "merged upstream",
        lastActivity: "1d ago",
        cost: "$2.18",
        evidence: "asset update",
      },
      {
        title: "Apply Dracula app theme",
        owner: "Vitaly",
        agent: "Claude Code",
        status: "merged upstream",
        lastActivity: "1d ago",
        cost: "$6.92",
        evidence: "UI tests",
      },
    ],
  },
];

function toneClass(tone: Tone): string {
  switch (tone) {
    case "success":
      return "border-success/30 bg-success/10 text-success";
    case "warning":
      return "border-warning/35 bg-warning/10 text-warning";
    case "info":
      return "border-info/30 bg-info/10 text-info";
    case "danger":
      return "border-destructive/35 bg-destructive/10 text-destructive";
    default:
      return "border-border bg-muted text-muted-foreground";
  }
}

function formatCost(amount: number | null): string {
  if (amount === null) return "-";
  return `$${amount.toFixed(2)}`;
}

function costSubtext(project: ProjectMock): string {
  if (project.costTotalUsd === null) return "no usage data";
  return `total - ${project.sessionsWithUsage} of ${project.costSessionsTotal} sessions`;
}

function coverageTone(project: ProjectMock): Tone {
  if (project.costTotalUsd === null || project.sessionsWithUsage === 0) return "warning";
  if (project.sessionsWithUsage < project.costSessionsTotal) return "info";
  return "success";
}

function formatLastActivity(lastActivityAt: number): string {
  const value = relativeTime(lastActivityAt * 1000);
  return value === "now" ? "now" : `${value} ago`;
}

function Metric({
  icon: Icon,
  label,
  value,
  tone = "muted",
}: {
  icon: ComponentType<{ className?: string }>;
  label: string;
  value: string;
  tone?: Tone;
}) {
  return (
    <div className="min-w-0 border-r border-border px-4 py-3 last:border-r-0">
      <div className="mb-1 flex items-center gap-2 text-[11px] uppercase tracking-normal text-muted-foreground">
        <Icon className={cn("size-3.5", tone !== "muted" && toneClass(tone).split(" ").at(-1))} />
        {label}
      </div>
      <div className="truncate text-lg font-semibold tabular-nums text-foreground">{value}</div>
    </div>
  );
}

function ProjectsPageHeader({
  search,
  onSearchChange,
}: {
  search: string;
  onSearchChange: (value: string) => void;
}) {
  return (
    <div className="mb-8 flex flex-col gap-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <h1 className="text-3xl font-semibold tracking-tight text-foreground">Projects</h1>
        <Badge variant="outline" className="w-fit bg-muted/40 text-muted-foreground">
          Sort by Last updated
        </Badge>
      </div>
      <label className="flex min-h-12 items-center gap-3 rounded-lg border border-border bg-muted/40 px-4 text-muted-foreground focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/30">
        <SearchIcon className="size-4 shrink-0" />
        <Input
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder="Search projects..."
          aria-label="Search projects"
          className="h-11 border-0 bg-transparent px-0 text-sm shadow-none focus-visible:ring-0"
        />
      </label>
    </div>
  );
}

function ProjectsOverviewPage() {
  const [search, setSearch] = useState("");
  const {
    data: projects = [],
    isError,
    isLoading,
  } = useQuery({
    queryKey: ["projects"],
    queryFn: listProjects,
  });
  const sortedProjects = useMemo(
    () =>
      [...projects].sort(
        (a, b) => b.lastActivityAt - a.lastActivityAt || a.name.localeCompare(b.name),
      ),
    [projects],
  );
  const normalizedSearch = search.trim().toLowerCase();
  const visibleProjects =
    normalizedSearch.length === 0
      ? sortedProjects
      : sortedProjects.filter(
          (project) =>
            project.name.toLowerCase().includes(normalizedSearch) ||
            project.workspace.toLowerCase().includes(normalizedSearch),
        );

  return (
    <PageScroll maxWidthClassName="max-w-5xl" contentClassName="px-4 md:px-6">
      <ProjectsPageHeader search={search} onSearchChange={setSearch} />

      {isLoading ? (
        <div className="rounded-lg border border-border bg-card px-5 py-8 text-sm text-muted-foreground">
          Loading projects...
        </div>
      ) : isError ? (
        <div className="rounded-lg border border-border bg-card px-5 py-8 text-sm text-muted-foreground">
          Projects could not be loaded.
        </div>
      ) : sortedProjects.length === 0 ? (
        <div className="rounded-lg border border-border bg-card px-5 py-8 text-sm text-muted-foreground">
          Projects appear here automatically as your team runs AI sessions in a workspace.
        </div>
      ) : visibleProjects.length === 0 ? (
        <div className="rounded-lg border border-border bg-card px-5 py-8 text-sm text-muted-foreground">
          No projects match your search.
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {visibleProjects.map((project) => (
            <Link
              key={project.id}
              to={`/projects/${project.id}`}
              className="block rounded-lg border border-border bg-card px-5 py-5 transition-colors hover:bg-muted/30"
            >
              <ProjectCard project={project} />
            </Link>
          ))}
        </div>
      )}
    </PageScroll>
  );
}

function ProjectCard({ project }: { project: ProjectSummary }) {
  return (
    <div className="min-w-0 space-y-4">
      <h2 className="truncate text-base font-semibold text-foreground">{project.name}</h2>
      <p className="truncate text-sm text-muted-foreground">{project.workspace}</p>
      <p className="text-xs font-medium text-muted-foreground">
        Updated {formatLastActivity(project.lastActivityAt)}
      </p>
    </div>
  );
}

function DetailHeader({ project }: { project: ProjectMock }) {
  return (
    <div className="mb-5 border-b border-border pb-4">
      <Button asChild variant="ghost" size="sm" className="-ml-2 mb-3">
        <Link to="/projects">
          <ArrowLeftIcon className="size-4" />
          Projects
        </Link>
      </Button>
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div className="min-w-0">
          <div className="mb-2 flex flex-wrap items-center gap-2">
            <h1 className="truncate text-2xl font-semibold tracking-tight text-foreground">
              {project.name}
            </h1>
            {project.awaitingInput > 0 && (
              <Badge variant="outline" className={toneClass("warning")}>
                {project.awaitingInput} awaiting input
              </Badge>
            )}
          </div>
          <p className="max-w-3xl text-sm text-muted-foreground">{project.description}</p>
          <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <GitBranchIcon className="size-3.5" />
              {project.branch}
            </span>
            <span className="flex items-center gap-1">
              <Clock3Icon className="size-3.5" />
              {project.lastActivity}
            </span>
            <span className="flex items-center gap-1">
              <Code2Icon className="size-3.5" />
              {project.workspace}
            </span>
          </div>
        </div>
        <div className="grid grid-cols-3 overflow-hidden rounded-lg border border-border bg-card lg:w-[420px]">
          <Metric
            icon={MessageSquareTextIcon}
            label="Sessions"
            value={String(project.sessionsThisWeek)}
          />
          <Metric
            icon={DollarSignIcon}
            label="AI cost total"
            value={formatCost(project.costTotalUsd)}
          />
          <Metric
            icon={AlertTriangleIcon}
            label="Awaiting input"
            value={String(project.awaitingInput)}
            tone={project.awaitingInput > 0 ? "warning" : "muted"}
          />
        </div>
      </div>
    </div>
  );
}

function BriefList({
  title,
  icon: Icon,
  items,
}: {
  title: string;
  icon: ComponentType<{ className?: string }>;
  items: BriefItem[];
}) {
  return (
    <section className="min-w-0 rounded-lg border border-border bg-card">
      <div className="flex items-center gap-2 border-b border-border px-4 py-3">
        <Icon className="size-4 text-muted-foreground" />
        <h2 className="text-sm font-semibold text-foreground">{title}</h2>
      </div>
      <div className="divide-y divide-border">
        {items.map((item) => (
          <div key={item.title} className="px-4 py-3">
            <div className="mb-1 flex items-start gap-2">
              <span className={cn("mt-1 size-2 rounded-full border", toneClass(item.tone))} />
              <p className="min-w-0 flex-1 text-sm text-foreground">{item.title}</p>
            </div>
            <p className="pl-4 text-xs text-muted-foreground">{item.evidence}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

function EvidenceRail({ project }: { project: ProjectMock }) {
  const stats = [
    {
      label: "Sessions with usage",
      value: `${project.sessionsWithUsage}/${project.costSessionsTotal}`,
    },
    { label: "Awaiting input", value: String(project.awaitingInput) },
    { label: "Failed sessions", value: String(project.failedSessions) },
    { label: "Comments linked", value: project.id === "web-app" ? "9" : "4" },
  ];

  return (
    <aside className="rounded-lg border border-border bg-card">
      <div className="border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <ShieldCheckIcon className="size-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold text-foreground">Source coverage</h2>
        </div>
        <p className="mt-1 text-xs text-muted-foreground">
          Project facts here are derived from sessions and usage reports.
        </p>
      </div>
      <div className="divide-y divide-border">
        {stats.map((stat) => (
          <div key={stat.label} className="flex items-center justify-between px-4 py-3 text-sm">
            <span className="text-muted-foreground">{stat.label}</span>
            <span className="font-medium tabular-nums text-foreground">{stat.value}</span>
          </div>
        ))}
      </div>
      <div className="border-t border-border px-4 py-3">
        <Badge variant="outline" className={toneClass(coverageTone(project))}>
          {project.costTotalUsd === null ? "No usage data" : "Usage coverage"}
        </Badge>
        <p className="mt-2 text-xs leading-5 text-muted-foreground">
          Cost coverage is shown as a fact, not as a project-quality score.
        </p>
      </div>
    </aside>
  );
}

function SessionsTable({ project }: { project: ProjectMock }) {
  return (
    <section className="mt-5 overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <CalendarDaysIcon className="size-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold text-foreground">Recent project sessions</h2>
        </div>
        <Badge variant="outline" className="bg-background text-muted-foreground">
          mock data
        </Badge>
      </div>
      <div className="grid grid-cols-[minmax(0,1.5fr)_0.7fr_0.75fr_0.65fr_0.8fr] border-b border-border px-4 py-2 text-[11px] uppercase tracking-normal text-muted-foreground max-lg:hidden">
        <div>Session</div>
        <div>Agent</div>
        <div>Status</div>
        <div>Cost</div>
        <div>Evidence</div>
      </div>
      {project.sessions.map((session) => (
        <div
          key={session.title}
          className="grid gap-2 border-b border-border px-4 py-3 last:border-b-0 lg:grid-cols-[minmax(0,1.5fr)_0.7fr_0.75fr_0.65fr_0.8fr] lg:items-center"
        >
          <div className="min-w-0">
            <div className="truncate text-sm font-medium text-foreground">{session.title}</div>
            <div className="mt-1 text-xs text-muted-foreground">
              {session.owner} - {session.lastActivity}
            </div>
          </div>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <BotIcon className="size-3.5" />
            {session.agent}
          </div>
          <div className="text-sm text-foreground">{session.status}</div>
          <div className="text-sm tabular-nums text-foreground">{session.cost}</div>
          <div className="text-sm text-muted-foreground">{session.evidence}</div>
        </div>
      ))}
    </section>
  );
}

function ProjectDetailPage({ project }: { project: ProjectMock }) {
  return (
    <PageScroll maxWidthClassName="max-w-7xl" contentClassName="px-4 md:px-6">
      <DetailHeader project={project} />

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_320px]">
        <div className="min-w-0">
          <section className="mb-5 rounded-lg border border-border bg-card">
            <div className="border-b border-border px-4 py-3">
              <div className="flex items-center gap-2">
                <FileTextIcon className="size-4 text-muted-foreground" />
                <h2 className="text-sm font-semibold text-foreground">Weekly project brief</h2>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">
                The first version reads as an evidence-backed operating brief, not a completion
                report.
              </p>
            </div>
            <div className="grid gap-0 md:grid-cols-3">
              <Metric
                icon={CheckCircle2Icon}
                label="Observed activity"
                value={String(project.observed.length)}
                tone="success"
              />
              <Metric
                icon={AlertTriangleIcon}
                label="Attention items"
                value={String(project.risks.length)}
                tone="warning"
              />
              <Metric
                icon={CircleHelpIcon}
                label="Open questions"
                value={String(project.questions.length)}
                tone="info"
              />
            </div>
          </section>

          <div className="grid gap-5 lg:grid-cols-3">
            <BriefList title="Observed activity" icon={SparklesIcon} items={project.observed} />
            <BriefList title="Attention and gaps" icon={AlertTriangleIcon} items={project.risks} />
            <BriefList title="Open questions" icon={CircleHelpIcon} items={project.questions} />
          </div>

          <SessionsTable project={project} />
        </div>

        <div className="space-y-5">
          <EvidenceRail project={project} />
          <section className="rounded-lg border border-border bg-card">
            <div className="border-b border-border px-4 py-3">
              <div className="flex items-center gap-2">
                <BarChart3Icon className="size-4 text-muted-foreground" />
                <h2 className="text-sm font-semibold text-foreground">AI usage slice</h2>
              </div>
            </div>
            <div className="space-y-3 px-4 py-3">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Sessions this week</span>
                <span className="font-medium tabular-nums text-foreground">
                  {project.sessionsThisWeek}
                </span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">AI cost total</span>
                <span className="font-medium tabular-nums text-foreground">
                  {formatCost(project.costTotalUsd)}
                </span>
              </div>
              <div className="flex items-center justify-between gap-4 text-sm">
                <span className="text-muted-foreground">Usage coverage</span>
                <span className="min-w-0 truncate font-medium text-foreground">
                  {costSubtext(project)}
                </span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Primary branch</span>
                <span className="font-medium text-foreground">{project.branch}</span>
              </div>
            </div>
          </section>
          <section className="rounded-lg border border-border bg-card">
            <div className="border-b border-border px-4 py-3">
              <div className="flex items-center gap-2">
                <GitPullRequestIcon className="size-4 text-muted-foreground" />
                <h2 className="text-sm font-semibold text-foreground">Suggested next actions</h2>
              </div>
            </div>
            <div className="divide-y divide-border text-sm">
              <div className="px-4 py-3 text-foreground">
                Normalize repo identity beyond workspace path.
              </div>
              <div className="px-4 py-3 text-foreground">
                Mark sessions that have command/test verification.
              </div>
              <div className="px-4 py-3 text-foreground">
                Attach brief items to session evidence links.
              </div>
            </div>
          </section>
        </div>
      </div>
    </PageScroll>
  );
}

export function ProjectsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const project = projectId ? projectDetails.find((item) => item.id === projectId) : null;

  if (!projectId) return <ProjectsOverviewPage />;
  if (project) return <ProjectDetailPage project={project} />;

  return (
    <PageScroll maxWidthClassName="max-w-3xl" contentClassName="px-6">
      <div className="rounded-lg border border-border bg-card p-6">
        <h1 className="text-xl font-semibold text-foreground">Project not found</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          This mockup only includes a few seeded project examples.
        </p>
        <Button asChild className="mt-4" variant="outline">
          <Link to="/projects">
            <ArrowLeftIcon className="size-4" />
            Back to projects
          </Link>
        </Button>
      </div>
    </PageScroll>
  );
}

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ProjectsPage } from "./ProjectsPage";
import type { ProjectSummary } from "@/lib/projectsApi";
import * as projectsApi from "@/lib/projectsApi";

vi.mock("@/lib/projectsApi", () => ({
  listProjects: vi.fn(),
}));

function project(overrides: Partial<ProjectSummary> = {}): ProjectSummary {
  return {
    id: "proj_web",
    name: "web-app",
    workspace: "/Users/me/goalrail/ap-web",
    lastActivityAt: 1_704_067_200,
    ...overrides,
  };
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/projects"]}>
        <ProjectsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.setSystemTime(new Date(1_704_067_200_000 + 5 * 60_000));
  vi.mocked(projectsApi.listProjects).mockResolvedValue([]);
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.clearAllMocks();
});

describe("ProjectsPage overview states", () => {
  it("shows a loading state while projects load", () => {
    vi.mocked(projectsApi.listProjects).mockReturnValue(new Promise(() => {}));
    renderPage();
    expect(screen.getByText("Loading projects...")).toBeInTheDocument();
  });

  it("shows an empty state once no projects are returned", async () => {
    renderPage();
    expect(
      await screen.findByText(
        "Projects appear here automatically as your team runs AI sessions in a workspace.",
      ),
    ).toBeInTheDocument();
  });

  it("shows an error state when the API request fails", async () => {
    vi.mocked(projectsApi.listProjects).mockRejectedValue(new Error("boom"));
    renderPage();
    expect(await screen.findByText("Projects could not be loaded.")).toBeInTheDocument();
  });
});

describe("ProjectsPage overview cards", () => {
  it("renders minimal project cards and no PM metrics on the first screen", async () => {
    vi.mocked(projectsApi.listProjects).mockResolvedValue([
      project({ name: "api-server", workspace: "/Users/me/goalrail/goalrail/server" }),
      project({ id: "proj_web", name: "web-app", workspace: "/Users/me/goalrail/ap-web" }),
    ]);

    renderPage();

    expect(await screen.findByRole("heading", { name: "web-app" })).toBeInTheDocument();
    expect(screen.getByText("/Users/me/goalrail/ap-web")).toBeInTheDocument();
    expect(screen.getAllByText("Updated 5m ago")).toHaveLength(2);

    for (const forbidden of [
      "People",
      "Sessions",
      "AI cost",
      "AI cost total",
      "Awaiting input",
      "Contributors",
      "Risks",
      "Verification",
      "Source coverage",
    ]) {
      expect(screen.queryByText(forbidden)).not.toBeInTheDocument();
    }
  });

  it("filters projects by name and workspace", async () => {
    vi.mocked(projectsApi.listProjects).mockResolvedValue([
      project({ id: "proj_web", name: "web-app", workspace: "/repo/ap-web" }),
      project({ id: "proj_api", name: "api-server", workspace: "/repo/server" }),
    ]);

    renderPage();
    await screen.findByRole("heading", { name: "web-app" });

    fireEvent.change(screen.getByLabelText("Search projects"), { target: { value: "server" } });

    await waitFor(() =>
      expect(screen.queryByRole("heading", { name: "web-app" })).not.toBeInTheDocument(),
    );
    expect(screen.getByRole("heading", { name: "api-server" })).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Search projects"), { target: { value: "missing" } });
    expect(screen.getByText("No projects match your search.")).toBeInTheDocument();
  });
});

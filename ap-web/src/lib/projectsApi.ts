import { authenticatedFetch } from "./identity";

interface ProjectWire {
  id: string;
  object?: "project";
  name: string;
  workspace: string;
  last_activity_at: number;
}

interface ProjectListWire {
  object?: "list";
  data?: ProjectWire[];
}

export interface ProjectSummary {
  id: string;
  name: string;
  workspace: string;
  lastActivityAt: number;
}

export function projectFromWire(wire: ProjectWire): ProjectSummary {
  return {
    id: wire.id,
    name: wire.name,
    workspace: wire.workspace,
    lastActivityAt: wire.last_activity_at,
  };
}

export async function listProjects(): Promise<ProjectSummary[]> {
  const res = await authenticatedFetch("/v1/projects");
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  const wire = (await res.json()) as ProjectListWire;
  return (wire.data ?? []).map(projectFromWire);
}

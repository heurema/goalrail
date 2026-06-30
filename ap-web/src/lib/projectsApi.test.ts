import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { listProjects, projectFromWire } from "./projectsApi";

function mockJsonResponse(body: unknown, init?: { ok?: boolean; status?: number }): Response {
  return {
    ok: init?.ok ?? true,
    status: init?.status ?? 200,
    statusText: init?.ok === false ? "Not Found" : "OK",
    json: async () => body,
  } as unknown as Response;
}

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("projectFromWire", () => {
  it("maps snake_case API fields to the frontend project shape", () => {
    expect(
      projectFromWire({
        id: "proj_123",
        object: "project",
        name: "goalrail",
        workspace: "/Users/me/goalrail",
        last_activity_at: 1_704_067_200,
      }),
    ).toEqual({
      id: "proj_123",
      name: "goalrail",
      workspace: "/Users/me/goalrail",
      lastActivityAt: 1_704_067_200,
    });
  });
});

describe("listProjects", () => {
  it("GETs /v1/projects and parses project rows", async () => {
    fetchMock.mockResolvedValueOnce(
      mockJsonResponse({
        object: "list",
        data: [
          {
            id: "proj_123",
            object: "project",
            name: "web-app",
            workspace: "/repo/ap-web",
            last_activity_at: 1_704_067_200,
          },
        ],
      }),
    );

    const projects = await listProjects();

    expect(fetchMock).toHaveBeenCalledOnce();
    expect(fetchMock.mock.calls[0][0]).toBe("/v1/projects");
    expect(projects).toEqual([
      {
        id: "proj_123",
        name: "web-app",
        workspace: "/repo/ap-web",
        lastActivityAt: 1_704_067_200,
      },
    ]);
  });

  it("throws when the response is not ok", async () => {
    fetchMock.mockResolvedValueOnce(mockJsonResponse({}, { ok: false, status: 500 }));
    await expect(listProjects()).rejects.toThrow(/500/);
  });
});

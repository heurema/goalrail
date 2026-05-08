import { afterEach, describe, expect, it, vi } from 'vitest';

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function qualificationFeedResponse() {
  return {
    items: [
      {
        intake_id: '018f0000-0000-7000-8000-000000000201',
        goal_id: '018f0000-0000-7000-8000-000000000202',
        organization_id: '018f0000-0000-7000-8000-000000000002',
        project_id: '018f0000-0000-7000-8000-000000000003',
        repo_binding_id: '018f0000-0000-7000-8000-000000000004',
        repository_full_name: 'heurema/goalrail',
        title: 'Improve billing error handling',
        lane: 'qualification',
        intake_state: 'received',
        goal_state: 'created',
        readiness: {
          ready: false,
          reason_codes: ['missing_scope_hint'],
          source: 'goal_snapshot',
        },
        next_action: {
          kind: 'continue_goal',
          available: true,
          blocking: false,
        },
        created_at: '2026-05-08T10:00:00Z',
      },
    ],
  };
}

describe('qualificationFeedClient API base URL', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it('calls same-origin qualification feed with bearer auth and query filters by default', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(qualificationFeedResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { listQualificationFeed } = await import('./qualificationFeedClient');
    const response = await listQualificationFeed({
      accessToken: 'access-token',
      repoBindingId: '018f0000-0000-7000-8000-000000000004',
      goalState: 'needs_clarification',
      limit: 50,
    });

    expect(response.items).toHaveLength(1);
    expect(fetchMock.mock.calls[0][0]).toBe(
      '/v1/qualification-feed?repo_binding_id=018f0000-0000-7000-8000-000000000004&goal_state=needs_clarification&limit=50'
    );
    expect(fetchMock.mock.calls[0][1]).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
  });

  it('calls configured API base URL without a double slash', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', ' https://api.goalrail.dev/ ');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(qualificationFeedResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { listQualificationFeed } = await import('./qualificationFeedClient');
    await listQualificationFeed({
      accessToken: 'access-token',
      limit: 50,
    });

    expect(fetchMock.mock.calls[0][0]).toBe('https://api.goalrail.dev/v1/qualification-feed?limit=50');
  });
});

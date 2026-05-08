import { afterEach, describe, expect, it, vi } from 'vitest';

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function contractListResponse() {
  return {
    contracts: [
      {
        id: '018f0000-0000-7000-8000-000000000101',
        repo_binding_id: '018f0000-0000-7000-8000-000000000004',
        goal_id: '018f0000-0000-7000-8000-000000000102',
        state: 'draft',
        current_seed_id: '018f0000-0000-7000-8000-000000000103',
        current_draft_id: '018f0000-0000-7000-8000-000000000104',
        created_at: '2026-05-08T09:00:00Z',
        updated_at: '2026-05-08T10:00:00Z',
      },
    ],
    limit: 50,
  };
}

describe('contractListClient API base URL', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it('calls same-origin contract discovery with bearer auth and query filters by default', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(contractListResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { listContracts } = await import('./contractListClient');
    const response = await listContracts({
      accessToken: 'access-token',
      projectId: '018f0000-0000-7000-8000-000000000003',
      repoBindingId: '018f0000-0000-7000-8000-000000000004',
      goalId: '018f0000-0000-7000-8000-000000000102',
      state: 'draft',
      limit: 50,
    });

    expect(response.contracts).toHaveLength(1);
    expect(response.limit).toBe(50);
    expect(fetchMock.mock.calls[0][0]).toBe(
      '/v1/contracts?project_id=018f0000-0000-7000-8000-000000000003&repo_binding_id=018f0000-0000-7000-8000-000000000004&goal_id=018f0000-0000-7000-8000-000000000102&state=draft&limit=50'
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
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(contractListResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { listContracts } = await import('./contractListClient');
    await listContracts({
      accessToken: 'access-token',
      limit: 50,
    });

    expect(fetchMock.mock.calls[0][0]).toBe('https://api.goalrail.dev/v1/contracts?limit=50');
  });

  it('maps backend errors into typed request errors', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: { code: 'unauthorized', message: 'nope' } }, 401));
    vi.stubGlobal('fetch', fetchMock);

    const { isContractListClientError, listContracts } = await import('./contractListClient');

    try {
      await listContracts({ accessToken: 'bad-token', limit: 50 });
      throw new Error('expected listContracts to fail');
    } catch (error) {
      expect(isContractListClientError(error)).toBe(true);
      expect(error).toMatchObject({
        code: 'unauthorized',
        status: 401,
      });
    }
  });
});

import { afterEach, describe, expect, it, vi } from 'vitest';

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}

function repositoryContextResponse() {
  return {
    organization: {
      id: '018f0000-0000-7000-8000-000000000002',
      slug: 'goalrail-dev',
      display_name: 'Goalrail Dev',
      state: 'active',
    },
    contexts: [],
  };
}

describe('repositoryContextClient API base URL', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it('calls same-origin organization repository context by default', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(repositoryContextResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { getOrganizationRepositoryContext } = await import('./repositoryContextClient');
    await getOrganizationRepositoryContext({
      accessToken: 'access-token',
      organizationId: '018f0000-0000-7000-8000-000000000002',
    });

    expect(fetchMock.mock.calls[0][0]).toBe('/v1/organizations/018f0000-0000-7000-8000-000000000002/repository-context');
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
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(repositoryContextResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { getOrganizationRepositoryContext } = await import('./repositoryContextClient');
    await getOrganizationRepositoryContext({
      accessToken: 'access-token',
      organizationId: '018f0000-0000-7000-8000-000000000002',
    });

    expect(fetchMock.mock.calls[0][0]).toBe('https://api.goalrail.dev/v1/organizations/018f0000-0000-7000-8000-000000000002/repository-context');
  });
});

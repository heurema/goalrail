import { afterEach, describe, expect, it, vi } from 'vitest';

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function contractResponse() {
  return {
    id: '018f0000-0000-7000-8000-000000000101',
    repo_binding_id: '018f0000-0000-7000-8000-000000000004',
    goal_id: '018f0000-0000-7000-8000-000000000102',
    state: 'draft',
    current_seed_id: '018f0000-0000-7000-8000-000000000103',
    current_draft_id: '018f0000-0000-7000-8000-000000000104',
    created_at: '2026-05-08T09:00:00Z',
    updated_at: '2026-05-08T10:00:00Z',
  };
}

function errorEnvelope(code: string, message = 'error') {
  return { error: { code, message } };
}

describe('contractDetailClient API base URL', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it('calls same-origin contract detail with bearer auth and safe contract id encoding by default', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(contractResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { getContractDetail } = await import('./contractDetailClient');
    const response = await getContractDetail({
      accessToken: 'access-token',
      contractId: 'contract/with space',
    });

    expect(response.id).toBe('018f0000-0000-7000-8000-000000000101');
    expect(fetchMock.mock.calls[0][0]).toBe('/v1/contracts/contract%2Fwith%20space');
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
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(contractResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { getContractDetail } = await import('./contractDetailClient');
    await getContractDetail({
      accessToken: 'access-token',
      contractId: '018f0000-0000-7000-8000-000000000101',
    });

    expect(fetchMock.mock.calls[0][0]).toBe('https://api.goalrail.dev/v1/contracts/018f0000-0000-7000-8000-000000000101');
  });

  it.each([
    ['not_found', 404],
    ['forbidden', 403],
    ['membership_required', 403],
    ['unauthorized', 401],
    ['database_not_configured', 503],
    ['server_error', 502],
  ])('maps %s backend errors into typed request errors', async (code, status) => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(errorEnvelope(code), status));
    vi.stubGlobal('fetch', fetchMock);

    const { getContractDetail, isContractDetailClientError } = await import('./contractDetailClient');

    try {
      await getContractDetail({
        accessToken: 'access-token',
        contractId: '018f0000-0000-7000-8000-000000000101',
      });
      throw new Error('expected getContractDetail to fail');
    } catch (error) {
      expect(isContractDetailClientError(error)).toBe(true);
      expect(error).toMatchObject({ code, status });
    }
  });

  it('maps unknown backend codes to server_error with the response status', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(errorEnvelope('unexpected_failure'), 418));
    vi.stubGlobal('fetch', fetchMock);

    const { getContractDetail } = await import('./contractDetailClient');

    await expect(getContractDetail({
      accessToken: 'access-token',
      contractId: 'contract-id',
    })).rejects.toMatchObject({ code: 'server_error', status: 418 });
  });

  it('maps fetch failures and invalid JSON into typed client errors', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new TypeError('network failed'))
      .mockResolvedValueOnce(new Response('not json', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    const { getContractDetail, isContractDetailClientError } = await import('./contractDetailClient');

    await expect(getContractDetail({
      accessToken: 'access-token',
      contractId: 'contract-id',
    })).rejects.toMatchObject({ code: 'network_error' });

    try {
      await getContractDetail({
        accessToken: 'access-token',
        contractId: 'contract-id',
      });
      throw new Error('expected invalid JSON to fail');
    } catch (error) {
      expect(isContractDetailClientError(error)).toBe(true);
      expect(error).toMatchObject({ code: 'response_parse_error', status: 200 });
    }
  });
});

import { afterEach, describe, expect, it, vi } from 'vitest';

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function contractDraftResponse() {
  return {
    id: '018f0000-0000-7000-8000-000000000104',
    contract_id: '018f0000-0000-7000-8000-000000000101',
    contract_seed_id: '018f0000-0000-7000-8000-000000000103',
    goal_id: '018f0000-0000-7000-8000-000000000102',
    repo_binding_id: '018f0000-0000-7000-8000-000000000004',
    title: 'Render contract draft details',
    intent_summary: 'Show the current draft body in the read-only Console.',
    proposed_scope: ['Render draft body'],
    proposed_non_goals: ['No lifecycle mutations'],
    proposed_constraints: ['Read-only Console'],
    proposed_acceptance_criteria: ['Draft fields are visible'],
    proposed_expected_checks: ['npm test'],
    proposed_proof_expectations: ['Validation commands pass'],
    risk_hints: ['Keep aggregate detail intact'],
    source_refs: [{ kind: 'contract_seed', id: '018f0000-0000-7000-8000-000000000103' }],
    state: 'draft',
    created_at: '2026-05-08T10:00:00Z',
  };
}

function errorEnvelope(code: string, message = 'error') {
  return { error: { code, message } };
}

describe('contractDraftClient API base URL', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it('calls same-origin current draft detail with bearer auth and safe contract id encoding by default', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(contractDraftResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { getCurrentContractDraft } = await import('./contractDraftClient');
    const response = await getCurrentContractDraft({
      accessToken: 'access-token',
      contractId: 'contract/with space',
    });

    expect(response.id).toBe('018f0000-0000-7000-8000-000000000104');
    expect(response.title).toBe('Render contract draft details');
    expect(fetchMock.mock.calls[0][0]).toBe('/v1/contracts/contract%2Fwith%20space/current-draft');
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
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(contractDraftResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { getCurrentContractDraft } = await import('./contractDraftClient');
    await getCurrentContractDraft({
      accessToken: 'access-token',
      contractId: '018f0000-0000-7000-8000-000000000101',
    });

    expect(fetchMock.mock.calls[0][0]).toBe('https://api.goalrail.dev/v1/contracts/018f0000-0000-7000-8000-000000000101/current-draft');
  });

  it.each([
    ['not_found', 404],
    ['invalid_state', 409],
    ['forbidden', 403],
    ['membership_required', 403],
    ['unauthorized', 401],
    ['database_not_configured', 503],
  ])('maps %s backend errors into typed request errors', async (code, status) => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(errorEnvelope(code), status));
    vi.stubGlobal('fetch', fetchMock);

    const { getCurrentContractDraft, isContractDraftClientError } = await import('./contractDraftClient');

    try {
      await getCurrentContractDraft({
        accessToken: 'access-token',
        contractId: '018f0000-0000-7000-8000-000000000101',
      });
      throw new Error('expected getCurrentContractDraft to fail');
    } catch (error) {
      expect(isContractDraftClientError(error)).toBe(true);
      expect(error).toMatchObject({ code, status });
    }
  });

  it('maps fetch failures and invalid JSON into typed client errors', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new TypeError('network failed'))
      .mockResolvedValueOnce(new Response('not json', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    const { getCurrentContractDraft, isContractDraftClientError } = await import('./contractDraftClient');

    await expect(getCurrentContractDraft({
      accessToken: 'access-token',
      contractId: 'contract-id',
    })).rejects.toMatchObject({ code: 'network_error' });

    try {
      await getCurrentContractDraft({
        accessToken: 'access-token',
        contractId: 'contract-id',
      });
      throw new Error('expected invalid JSON to fail');
    } catch (error) {
      expect(isContractDraftClientError(error)).toBe(true);
      expect(error).toMatchObject({ code: 'response_parse_error', status: 200 });
    }
  });
});

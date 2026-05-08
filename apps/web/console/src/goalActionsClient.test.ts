import { afterEach, describe, expect, it, vi } from 'vitest';

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function goalContinuationResponse() {
  return {
    goal_id: '018f0000-0000-7000-8000-000000000202',
    state: 'needs_clarification',
    readiness: {
      ready: false,
      reason_codes: ['missing_scope_hint'],
    },
  };
}

function contractResponse() {
  return {
    id: '018f0000-0000-7000-8000-000000000240',
    repo_binding_id: '018f0000-0000-7000-8000-000000000004',
    goal_id: '018f0000-0000-7000-8000-000000000202',
    state: 'draft',
    current_seed_id: '018f0000-0000-7000-8000-000000000241',
    current_draft_id: '018f0000-0000-7000-8000-000000000242',
    created_at: '2026-05-08T10:00:00Z',
    updated_at: '2026-05-08T10:00:00Z',
  };
}

describe('goalActionsClient', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it('posts goal continuation with bearer auth', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(goalContinuationResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { continueGoal } = await import('./goalActionsClient');
    const response = await continueGoal({
      accessToken: 'access-token',
      goalId: '018f0000-0000-7000-8000-000000000202',
    });

    expect(response.state).toBe('needs_clarification');
    expect(fetchMock.mock.calls[0][0]).toBe('/v1/goals/018f0000-0000-7000-8000-000000000202/continuation');
    expect(fetchMock.mock.calls[0][1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(fetchMock.mock.calls[0][1]?.body).toBeUndefined();
  });

  it('calls configured API base URL without a double slash', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', ' https://api.goalrail.dev/ ');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(goalContinuationResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { continueGoal } = await import('./goalActionsClient');
    await continueGoal({
      accessToken: 'access-token',
      goalId: 'goal-1',
    });

    expect(fetchMock.mock.calls[0][0]).toBe('https://api.goalrail.dev/v1/goals/goal-1/continuation');
  });

  it('posts clarification answers with structured question_id values', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({
      ...goalContinuationResponse(),
      state: 'ready_for_contract_seed',
    }));
    vi.stubGlobal('fetch', fetchMock);

    const { answerClarification } = await import('./goalActionsClient');
    const response = await answerClarification({
      accessToken: 'access-token',
      clarificationRequestId: '018f0000-0000-7000-8000-000000000220',
      answers: [
        {
          question_id: '018f0000-0000-7000-8000-000000000221',
          value: 'Scope is billing API retry behavior.',
        },
      ],
    });

    expect(response.state).toBe('ready_for_contract_seed');
    expect(fetchMock.mock.calls[0][0]).toBe('/v1/clarifications/018f0000-0000-7000-8000-000000000220/answers/continuation');
    expect(fetchMock.mock.calls[0][1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        credentials: 'omit',
        headers: {
          Authorization: 'Bearer access-token',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          answers: [
            {
              question_id: '018f0000-0000-7000-8000-000000000221',
              value: 'Scope is billing API retry behavior.',
            },
          ],
        }),
      })
    );
  });

  it('posts contract draft creation with goal_id only', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(contractResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { draftContract } = await import('./goalActionsClient');
    const response = await draftContract({
      accessToken: 'access-token',
      goalId: '018f0000-0000-7000-8000-000000000202',
    });

    expect(response.state).toBe('draft');
    expect(fetchMock.mock.calls[0][0]).toBe('/v1/contracts');
    expect(fetchMock.mock.calls[0][1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        credentials: 'omit',
        headers: {
          Authorization: 'Bearer access-token',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          goal_id: '018f0000-0000-7000-8000-000000000202',
        }),
      })
    );
  });
});

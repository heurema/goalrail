import { afterEach, describe, expect, it, vi } from 'vitest';

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}

function loginResponse() {
  return {
    user_id: '018f0000-0000-7000-8000-000000000001',
    access_token: 'access-token',
    access_token_expires_at: '2026-05-04T12:15:00Z',
    token_type: 'Bearer',
    refresh_token: 'refresh-token',
    refresh_token_expires_at: '2026-06-03T12:00:00Z',
    must_change_password: false,
  };
}

describe('authClient API base URL', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it('calls same-origin /v1 endpoints by default', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(loginResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { login } = await import('./authClient');
    await login({ email: 'owner@example.com', password: 'password' });

    expect(fetchMock.mock.calls[0][0]).toBe('/v1/auth/login');
  });

  it('calls configured API base URL without a double slash', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', ' https://api.goalrail.dev/ ');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(loginResponse()));
    vi.stubGlobal('fetch', fetchMock);

    const { me } = await import('./authClient');
    await me('access-token');

    expect(fetchMock.mock.calls[0][0]).toBe('https://api.goalrail.dev/v1/me');
  });
});

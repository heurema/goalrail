import { afterEach, describe, expect, it, vi } from 'vitest';

describe('API URL config', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('uses same-origin /v1 paths when the API base URL is empty', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', '');

    const { apiBaseURL, buildAPIURL } = await import('./config');

    expect(apiBaseURL).toBe('');
    expect(buildAPIURL('/v1/auth/login')).toBe('/v1/auth/login');
  });

  it('trims whitespace and removes trailing slashes from configured API base URL', async () => {
    vi.resetModules();
    vi.stubEnv('VITE_GOALRAIL_API_BASE_URL', ' https://api.goalrail.dev/ ');

    const { apiBaseURL, buildAPIURL } = await import('./config');

    expect(apiBaseURL).toBe('https://api.goalrail.dev');
    expect(buildAPIURL('/v1/me')).toBe('https://api.goalrail.dev/v1/me');
  });
});

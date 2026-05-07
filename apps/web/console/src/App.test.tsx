import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Mock } from 'vitest';

import App from './App';
import i18n from './i18n';
import { render } from '../test-utils';

const THEME_STORAGE_KEY = 'goalrail.console.theme';

let fetchMock: ReturnType<typeof vi.fn>;

function asMock(value: unknown) {
  return value as Mock;
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function deferredResponse() {
  let resolve!: (response: Response) => void;
  const promise = new Promise<Response>((next) => {
    resolve = next;
  });

  return { promise, resolve };
}

function loginResponse(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    user_id: '018f0000-0000-7000-8000-000000000001',
    access_token: 'access-token',
    access_token_expires_at: '2026-05-04T12:15:00Z',
    token_type: 'Bearer',
    refresh_token: 'refresh-token',
    refresh_token_expires_at: '2026-06-03T12:00:00Z',
    must_change_password: false,
    ...overrides,
  };
}

function meResponse(overrides: { role?: string } = {}) {
  return {
    user: {
      id: '018f0000-0000-7000-8000-000000000001',
      display_name: 'Owner',
      email: 'owner@example.com',
      state: 'active',
    },
    organization_membership: {
      id: '018f0000-0000-7000-8000-000000000301',
      organization_id: '018f0000-0000-7000-8000-000000000002',
      user_id: '018f0000-0000-7000-8000-000000000001',
      role: overrides.role ?? 'owner',
      state: 'active',
    },
  };
}

function organizationUserRecord(overrides: Partial<Record<string, unknown>> = {}) {
  const userId = String(overrides.userId ?? '018f0000-0000-7000-8000-000000000010');
  const displayName = String(overrides.displayName ?? 'Dev User');
  const email = overrides.email === null ? undefined : String(overrides.email ?? 'dev@example.com');
  const role = String(overrides.role ?? 'member');
  const state = String(overrides.state ?? 'active');
  const mustChangePassword = Boolean(overrides.mustChangePassword ?? false);

  return {
    user: {
      id: userId,
      display_name: displayName,
      ...(email === undefined ? {} : { email }),
      state: String(overrides.userState ?? state),
      created_at: '2026-05-06T10:00:00Z',
      updated_at: '2026-05-06T10:15:00Z',
    },
    organization_membership: {
      id: String(overrides.membershipId ?? '018f0000-0000-7000-8000-000000000310'),
      organization_id: '018f0000-0000-7000-8000-000000000002',
      user_id: userId,
      role,
      state,
      created_at: '2026-05-06T10:00:00Z',
      updated_at: '2026-05-06T10:15:00Z',
    },
    credential: {
      must_change_password: mustChangePassword,
      password_changed_at: mustChangePassword ? null : '2026-05-06T10:20:00Z',
    },
  };
}

function organizationUsersResponse(users = [organizationUserRecord()]) {
  return { users };
}

function contractResponse(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: '018f0000-0000-7000-8000-000000000101',
    repo_binding_id: '018f0000-0000-7000-8000-000000000004',
    goal_id: '018f0000-0000-7000-8000-000000000102',
    state: 'draft',
    current_seed_id: '018f0000-0000-7000-8000-000000000103',
    current_draft_id: '018f0000-0000-7000-8000-000000000104',
    created_at: '2026-05-06T10:00:00Z',
    updated_at: '2026-05-06T10:15:00Z',
    ...overrides,
  };
}

function repositoryContextResponse(contexts: unknown[] = [repositoryContextRecord()]) {
  return {
    organization: {
      id: '018f0000-0000-7000-8000-000000000002',
      slug: 'goalrail-dev',
      display_name: 'Goalrail Dev',
      state: 'active',
    },
    contexts,
  };
}

function repositoryContextRecord(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    project: {
      id: String(overrides.projectId ?? '018f0000-0000-7000-8000-000000000003'),
      slug: String(overrides.projectSlug ?? 'github-heurema-goalrail'),
      display_name: String(overrides.projectDisplayName ?? 'heurema/goalrail'),
      state: String(overrides.projectState ?? 'active'),
      created_at: '2026-05-07T10:00:00Z',
      updated_at: '2026-05-07T10:15:00Z',
    },
    repo_binding: {
      id: String(overrides.repoBindingId ?? '018f0000-0000-7000-8000-000000000004'),
      provider: String(overrides.provider ?? 'github'),
      repository_full_name: String(overrides.repositoryFullName ?? 'heurema/goalrail'),
      repository_url: String(overrides.repositoryUrl ?? 'git@github.com:heurema/goalrail.git'),
      default_branch: String(overrides.defaultBranch ?? 'main'),
      workflow_base_branch: String(overrides.workflowBaseBranch ?? 'main'),
      path_scope: String(overrides.pathScope ?? '.'),
      access_mode: String(overrides.accessMode ?? 'metadata_only'),
      state: String(overrides.repoBindingState ?? 'active'),
      created_at: '2026-05-07T10:00:00Z',
      updated_at: '2026-05-07T10:15:00Z',
    },
  };
}

function errorEnvelope(code: string, message = 'error') {
  return {
    error: { code, message },
  };
}

async function setLocale(locale: 'en' | 'ru') {
  await i18n.changeLanguage(locale);
  document.documentElement.lang = locale;
}

async function loginSuccessfully(locale: 'en' | 'ru' = 'en', membershipRole = 'owner') {
  await setLocale(locale);
  fetchMock.mockResolvedValueOnce(jsonResponse(loginResponse()));
  fetchMock.mockResolvedValueOnce(jsonResponse(meResponse({ role: membershipRole })));
  render(<App />);

  fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
  fireEvent.change(screen.getByLabelText(locale === 'ru' ? /^Пароль$/i : /^Password$/i), {
    target: { value: 'password' },
  });
  fireEvent.click(screen.getByRole('button', { name: locale === 'ru' ? /войти/i : /sign in/i }));

  await screen.findByRole('navigation', { name: locale === 'ru' ? /разделы продукта/i : /product surfaces/i });
}

async function startPasswordChangeLogin(locale: 'en' | 'ru' = 'en') {
  await setLocale(locale);
  fetchMock.mockResolvedValueOnce(jsonResponse(loginResponse({ must_change_password: true })));
  render(<App />);

  fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
  fireEvent.change(screen.getByLabelText(locale === 'ru' ? /^Пароль$/i : /^Password$/i), {
    target: { value: 'temporary-password' },
  });
  fireEvent.click(screen.getByRole('button', { name: locale === 'ru' ? /войти/i : /sign in/i }));

  await screen.findByRole('form', { name: locale === 'ru' ? /смена временного пароля/i : /temporary password change/i });
}

describe('App', () => {
  beforeEach(async () => {
    fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    window.history.replaceState(null, '', '/');
    window.localStorage.clear();
    window.sessionStorage.clear();
    asMock(window.localStorage.clear).mockClear();
    asMock(window.localStorage.getItem).mockClear();
    asMock(window.localStorage.setItem).mockClear();
    asMock(window.sessionStorage.clear).mockClear();
    asMock(window.sessionStorage.getItem).mockClear();
    asMock(window.sessionStorage.setItem).mockClear();
    await setLocale('en');
  });

  it('renders EN and RU login screens without registration, SSO, or password reset', async () => {
    const firstRender = render(<App />);

    expect(screen.getByLabelText(/goalrail console/i)).toHaveTextContent(/^GOALRAIL$/);
    expect(screen.getByLabelText(/^Email$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Password$/i)).toBeInTheDocument();
    expect(screen.queryByText(/registration|register|sign up|sso|reset|forgot|сброс|восстанов/i)).not.toBeInTheDocument();

    firstRender.unmount();
    await setLocale('ru');
    render(<App />);

    expect(screen.getByLabelText(/консоль goalrail/i)).toHaveTextContent(/^GOALRAIL$/);
    expect(screen.getByLabelText(/^Email$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Пароль$/i)).toBeInTheDocument();
    expect(screen.queryByText(/регистрация|зарегистрироваться|sign up|sso|reset|forgot|сброс|восстанов/i)).not.toBeInTheDocument();
  });

  it('keeps empty login fields client-side and does not call the API', () => {
    render(<App />);

    fireEvent.click(screen.getByRole('button', { name: /sign in/i }));

    expect(fetchMock).not.toHaveBeenCalled();
    expect(screen.getByRole('alert')).toHaveTextContent('Enter email and password to continue.');
  });

  it('successful login calls auth login, then /v1/me, then renders the console', async () => {
    await loginSuccessfully();

    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock.mock.calls[0][0]).toBe('/v1/auth/login');
    expect(fetchMock.mock.calls[0][1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        credentials: 'omit',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: 'owner@example.com', password: 'password' }),
      })
    );
    expect(fetchMock.mock.calls[1][0]).toBe('/v1/me');
    expect(fetchMock.mock.calls[1][1]).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(screen.getByLabelText(/current user/i)).toHaveTextContent('Owner');
    expect(screen.getByLabelText(/current user/i)).toHaveTextContent('owner@example.com');
  });

  it('renders an EN localized server membership admin role', async () => {
    await loginSuccessfully('en', 'admin');

    expect(screen.getByLabelText(/current user/i)).toHaveTextContent('role · Admin');
  });

  it.each([
    ['en', /^Password$/i, /sign in/i, 'Invalid email or password.'],
    ['ru', /^Пароль$/i, /войти/i, 'Неверный email или пароль.'],
  ] as const)('invalid credentials show a localized %s error and do not render the console', async (locale, passwordLabel, buttonName, message) => {
    await setLocale(locale);
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('invalid_credentials'), 401));
    render(<App />);

    fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
    fireEvent.change(screen.getByLabelText(passwordLabel), { target: { value: 'wrong' } });
    fireEvent.click(screen.getByRole('button', { name: buttonName }));

    expect(await screen.findByRole('alert')).toHaveTextContent(message);
    expect(screen.queryByRole('navigation', { name: locale === 'ru' ? /разделы продукта/i : /product surfaces/i })).not.toBeInTheDocument();
  });

  it.each([
    ['en', 'database_not_configured', 'Goalrail server is not ready yet.'],
    ['ru', 'auth_not_configured', 'Авторизация на сервере пока не готова.'],
  ] as const)('%s operational backend errors stay honest and do not render the console', async (locale, code, message) => {
    await setLocale(locale);
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope(code), 503));
    render(<App />);

    fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
    fireEvent.change(screen.getByLabelText(locale === 'ru' ? /^Пароль$/i : /^Password$/i), {
      target: { value: 'password' },
    });
    fireEvent.click(screen.getByRole('button', { name: locale === 'ru' ? /войти/i : /sign in/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent(message);
    expect(screen.queryByRole('navigation', { name: locale === 'ru' ? /разделы продукта/i : /product surfaces/i })).not.toBeInTheDocument();
  });

  it('must_change_password shows a password-change form before the console', async () => {
    await startPasswordChangeLogin();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(screen.getByLabelText(/^Current password$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^New password$/i)).toBeInTheDocument();
    expect(screen.queryByRole('navigation', { name: /product surfaces/i })).not.toBeInTheDocument();
  });

  it('password change calls the backend, then /v1/me, then renders the console', async () => {
    await startPasswordChangeLogin('ru');
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        user_id: '018f0000-0000-7000-8000-000000000001',
        must_change_password: false,
        password_changed_at: '2026-05-04T12:01:00Z',
      })
    );
    fetchMock.mockResolvedValueOnce(jsonResponse(meResponse()));

    fireEvent.change(screen.getByLabelText(/^Текущий пароль$/i), { target: { value: 'temporary-password' } });
    fireEvent.change(screen.getByLabelText(/^Новый пароль$/i), { target: { value: 'new-password' } });
    fireEvent.click(screen.getByRole('button', { name: /сменить пароль/i }));

    await screen.findByRole('navigation', { name: /разделы продукта/i });

    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls[1][0]).toBe('/v1/auth/change-password');
    expect(fetchMock.mock.calls[1][1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        credentials: 'omit',
        headers: {
          'Content-Type': 'application/json',
          Authorization: 'Bearer access-token',
        },
        body: JSON.stringify({
          current_password: 'temporary-password',
          new_password: 'new-password',
        }),
      })
    );
    expect(fetchMock.mock.calls[2][0]).toBe('/v1/me');
  });

  it('invalid current password stays on the password-change form', async () => {
    await startPasswordChangeLogin();
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('invalid_current_password'), 401));

    fireEvent.change(screen.getByLabelText(/^Current password$/i), { target: { value: 'wrong-current' } });
    fireEvent.change(screen.getByLabelText(/^New password$/i), { target: { value: 'new-password' } });
    fireEvent.click(screen.getByRole('button', { name: /change password/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Current password is invalid.');
    expect(screen.getByRole('form', { name: /temporary password change/i })).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('logout calls the backend, clears auth state, and returns to login', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse({ revoked: true }));

    fireEvent.click(screen.getByRole('button', { name: /log out/i }));

    await screen.findByLabelText(/^Email$/i);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls[2][0]).toBe('/v1/auth/logout');
    expect(fetchMock.mock.calls[2][1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(screen.queryByRole('navigation', { name: /product surfaces/i })).not.toBeInTheDocument();
  });

  it('swallows failed logout requests, clears auth state, and keeps auth storage empty', async () => {
    await loginSuccessfully();
    fetchMock.mockRejectedValueOnce(new TypeError('network failure'));

    fireEvent.click(screen.getByRole('button', { name: /log out/i }));

    await screen.findByLabelText(/^Email$/i);
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual(['/v1/auth/login', '/v1/me', '/v1/auth/logout']);
    expect(fetchMock.mock.calls[2][1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(screen.queryByRole('navigation', { name: /product surfaces/i })).not.toBeInTheDocument();
    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(window.localStorage.length).toBe(0);
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
    expect(window.sessionStorage.length).toBe(0);
    expect(document.body).not.toHaveTextContent(
      /registration|register|sign up|sso|invite|reset password|password reset|analytics|chat|file upload|model selector|organization creation|repo integration|runner|gate queue/i
    );
  });

  it('does not write access tokens, refresh tokens, profile data, or auth state to browser storage', async () => {
    await loginSuccessfully();

    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(window.localStorage.length).toBe(0);
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
    expect(window.sessionStorage.length).toBe(0);
  });

  it('keeps exactly three product surfaces and renders contracts from real API lookup only', async () => {
    await loginSuccessfully();

    const navigation = screen.getByRole('navigation', { name: /product surfaces/i });
    const productButtons = within(navigation).getAllByRole('button');

    expect(productButtons).toHaveLength(3);
    expect(screen.getByRole('button', { name: /^Contracts$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: /^Delivery Readiness$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Proof$/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/contract workspace/i)).toBeInTheDocument();
    expect(screen.getAllByText('Not selected').length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Select a contract to review its state and delivery scope/i).length).toBeGreaterThan(0);
    expect(document.body).not.toHaveTextContent(
      /trialops-demo|C-0147|fake queue|fake pass|fake fail|backend available|endpoint|GET \/v1\/contracts|prefilled/i
    );

    fetchMock.mockResolvedValueOnce(jsonResponse(contractResponse()));
    fireEvent.change(screen.getByLabelText(/^Contract ID$/i), {
      target: { value: '018f0000-0000-7000-8000-000000000101' },
    });
    fireEvent.click(screen.getByRole('button', { name: /load contract/i }));

    await screen.findByText('018f0000-0000-7000-8000-000000000104');
    expect(fetchMock.mock.calls[2][0]).toBe('/v1/contracts/018f0000-0000-7000-8000-000000000101');
    expect(fetchMock.mock.calls[2][1]).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    expect(screen.getByText(/enough context to become a delivery contract/i)).toBeInTheDocument();
    expect(screen.getByText('NOT CHECKED')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Русский$/i }));

    expect(await screen.findByRole('navigation', { name: /разделы продукта/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /^Оценка готовности$/i }));
    expect(screen.getByText(/хватает ли контекста/i)).toBeInTheDocument();
    expect(screen.getByText('НЕ ПРОВЕРЯЛОСЬ')).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent(
      /trialops-demo|C-0147|readiness score|\/100|\bscan\b|proof queue|fake queue|fake pass|fake fail|pass\/fail/i
    );
  });

  it('shows not_found for missing contract IDs without seeding demo data', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('not_found'), 404));

    fireEvent.change(screen.getByLabelText(/^Contract ID$/i), {
      target: { value: 'missing-contract' },
    });
    fireEvent.click(screen.getByRole('button', { name: /load contract/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Contract was not found.');
    expect(screen.getAllByText('Check the ID and try again.').length).toBeGreaterThan(0);
    expect(document.body).not.toHaveTextContent(/C-0147|trialops-demo|billing-api|frontend-console/i);
  });

  it('keeps the latest contract lookup when an earlier response resolves last', async () => {
    await loginSuccessfully();
    const firstLookup = deferredResponse();
    const secondLookup = deferredResponse();
    fetchMock.mockImplementationOnce(() => firstLookup.promise);
    fetchMock.mockImplementationOnce(() => secondLookup.promise);
    const contractIdInput = screen.getByLabelText(/^Contract ID$/i);
    const contractForm = contractIdInput.closest('form');

    expect(contractForm).not.toBeNull();

    fireEvent.change(contractIdInput, {
      target: { value: 'contract-A' },
    });
    fireEvent.submit(contractForm as HTMLFormElement);

    fireEvent.change(contractIdInput, {
      target: { value: 'contract-B' },
    });
    fireEvent.submit(contractForm as HTMLFormElement);

    await act(async () => {
      secondLookup.resolve(jsonResponse(contractResponse({ id: 'contract-B', current_draft_id: 'draft-B' })));
    });

    await screen.findByText('draft-B');

    await act(async () => {
      firstLookup.resolve(jsonResponse(contractResponse({ id: 'contract-A', current_draft_id: 'draft-A' })));
    });

    await waitFor(() => {
      expect(screen.getByText('draft-B')).toBeInTheDocument();
      expect(screen.queryByText('draft-A')).not.toBeInTheDocument();
    });
    expect(fetchMock.mock.calls[2][0]).toBe('/v1/contracts/contract-A');
    expect(fetchMock.mock.calls[3][0]).toBe('/v1/contracts/contract-B');
  });

  it('theme storage remains exactly goalrail.console.theme', async () => {
    await loginSuccessfully('ru');

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'goalrail-default');

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));
    fireEvent.click(screen.getByRole('button', { name: /Nord/i }));

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'nord');
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe('nord');
    expect(asMock(window.localStorage.setItem).mock.calls).toEqual([[THEME_STORAGE_KEY, 'nord']]);
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
  });

  it('initializes with a stored valid theme and falls back for invalid values', async () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, 'solarized-dark');
    asMock(window.localStorage.setItem).mockClear();

    await loginSuccessfully();

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'solarized-dark');
  });

  it('falls back to the default theme when stored theme is invalid', async () => {
    window.localStorage.clear();
    window.localStorage.setItem(THEME_STORAGE_KEY, 'unknown-theme');
    asMock(window.localStorage.setItem).mockClear();

    await loginSuccessfully();

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'goalrail-default');
  });

  it('language switching changes visible labels without losing the authenticated session', async () => {
    await loginSuccessfully();

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    expect(screen.getByRole('heading', { name: /^Appearance$/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Русский$/i }));

    await screen.findByRole('heading', { name: /^Оформление$/i });
    expect(screen.getByLabelText(/текущий пользователь/i)).toHaveTextContent('Owner');
    expect(screen.getByRole('button', { name: /^Контракты$/i })).toBeInTheDocument();
    expect(document.documentElement.lang).toBe('ru');
    expect(window.location.search).toBe('?lng=ru');
    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
  });

  it('localizes server membership admin role after switching to RU without losing the authenticated session', async () => {
    await loginSuccessfully('en', 'admin');

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Русский$/i }));

    await screen.findByRole('heading', { name: /^Оформление$/i });
    expect(screen.getByLabelText(/текущий пользователь/i)).toHaveTextContent('Owner');
    expect(screen.getByLabelText(/текущий пользователь/i)).toHaveTextContent('role · Администратор');
    expect(screen.getByRole('button', { name: /^Контракты$/i })).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
  });

  it('users screen calls GET with the organization_id from /v1/me', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([
      organizationUserRecord({ displayName: 'Owner', email: 'owner@example.com', role: 'owner', userId: '018f0000-0000-7000-8000-000000000001' }),
    ])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));

    expect(await screen.findByRole('table', { name: /workspace users/i })).toHaveTextContent('Owner');
    expect(fetchMock.mock.calls[2][0]).toBe('/v1/organizations/018f0000-0000-7000-8000-000000000002/users');
    expect(fetchMock.mock.calls[2][1]).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(screen.queryByText('Product Lead')).not.toBeInTheDocument();
    expect(screen.queryByText('Reviewer')).not.toBeInTheDocument();
    expect(screen.queryByText('qa@example.com')).not.toBeInTheDocument();
  });

  it('repository settings calls GET with the organization_id from /v1/me and renders metadata only', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(repositoryContextResponse()));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Repository$/i }));

    expect(await screen.findByRole('heading', { name: /^Repository$/i })).toBeInTheDocument();
    expect(await screen.findByText('Goalrail Dev')).toBeInTheDocument();
    expect(screen.getAllByText('heurema/goalrail').length).toBeGreaterThan(0);
    expect(screen.getByText('metadata_only')).toBeInTheDocument();
    expect(screen.getByText(/does not prove checkout permission or provider authorization/i)).toBeInTheDocument();
    expect(fetchMock.mock.calls[2][0]).toBe('/v1/organizations/018f0000-0000-7000-8000-000000000002/repository-context');
    expect(fetchMock.mock.calls[2][1]).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(document.body).not.toHaveTextContent(/readiness score|proof status|checkout ready|provider connected|token|credential|run receipt|gate decision/i);
    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
  });

  it('repository settings shows an empty state for no active contexts', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(repositoryContextResponse([])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Repository$/i }));

    expect(await screen.findByText(/No repository context initialized yet/i)).toBeInTheDocument();
    expect(screen.getByText(/read-only Console surface/i)).toBeInTheDocument();
  });

  it('repository settings shows loading and error states', async () => {
    await loginSuccessfully();
    const repositoryLookup = deferredResponse();
    fetchMock.mockImplementationOnce(() => repositoryLookup.promise);

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Repository$/i }));

    expect(await screen.findByRole('status')).toHaveTextContent('Loading repository context...');

    await act(async () => {
      repositoryLookup.resolve(jsonResponse(errorEnvelope('forbidden'), 403));
    });

    expect(await screen.findByRole('alert')).toHaveTextContent('You do not have access to this Organization repository context.');
  });

  it('ignores stale user-list responses after auth state changes', async () => {
    await loginSuccessfully();
    const usersLookup = deferredResponse();
    fetchMock.mockImplementationOnce(() => usersLookup.promise);

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await screen.findByRole('status');

    fetchMock.mockResolvedValueOnce(jsonResponse({ revoked: true }));
    fireEvent.click(screen.getByRole('button', { name: /log out/i }));
    await screen.findByLabelText(/^Email$/i);

    await act(async () => {
      usersLookup.resolve(jsonResponse(organizationUsersResponse([
        organizationUserRecord({ displayName: 'Stale User', email: 'stale@example.com' }),
      ])));
    });

    await waitFor(() => {
      expect(screen.queryByText('Stale User')).not.toBeInTheDocument();
      expect(screen.getByLabelText(/^Email$/i)).toBeInTheDocument();
    });
  });

  it('ignores stale user mutation responses after auth state changes', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await screen.findByText(/No organization users returned/i);

    const createUser = deferredResponse();
    fetchMock.mockImplementationOnce(() => createUser.promise);
    fireEvent.click(screen.getByRole('button', { name: /^Add/i }));
    const drawer = screen.getByLabelText(/add user/i);
    fireEvent.change(within(drawer).getByLabelText(/^Name$/i), { target: { value: 'Stale Created User' } });
    fireEvent.change(within(drawer).getByLabelText(/^Email$/i), { target: { value: 'stale-created@example.com' } });
    fireEvent.click(within(drawer).getByRole('button', { name: /^Save$/i }));

    fetchMock.mockResolvedValueOnce(jsonResponse({ revoked: true }));
    fireEvent.click(screen.getByRole('button', { name: /log out/i }));
    await screen.findByLabelText(/^Email$/i);

    await act(async () => {
      createUser.resolve(
        jsonResponse({
          ...organizationUserRecord({ displayName: 'Stale Created User', email: 'stale-created@example.com' }),
          temporary_password: 'stale-created-secret',
        }, 201)
      );
    });

    await waitFor(() => {
      expect(screen.queryByText('Stale Created User')).not.toBeInTheDocument();
      expect(screen.queryByText('stale-created-secret')).not.toBeInTheDocument();
      expect(screen.getByLabelText(/^Email$/i)).toBeInTheDocument();
    });
  });

  it('keeps an in-flight users list load after a failed create', async () => {
    await loginSuccessfully();
    const usersLookup = deferredResponse();
    fetchMock.mockImplementationOnce(() => usersLookup.promise);

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await screen.findByRole('status');

    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('organization_user_exists', 'organization user already exists'), 409));
    fireEvent.click(screen.getByRole('button', { name: /^Add/i }));
    const drawer = screen.getByLabelText(/add user/i);
    fireEvent.change(within(drawer).getByLabelText(/^Name$/i), { target: { value: 'Existing User' } });
    fireEvent.change(within(drawer).getByLabelText(/^Email$/i), { target: { value: 'existing@example.com' } });
    fireEvent.click(within(drawer).getByRole('button', { name: /^Save$/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('This organization user already exists.');

    await act(async () => {
      usersLookup.resolve(jsonResponse(organizationUsersResponse([
        organizationUserRecord({ displayName: 'Loaded User', email: 'loaded@example.com' }),
      ])));
    });

    await waitFor(() => {
      expect(screen.getByRole('table', { name: /workspace users/i })).toHaveTextContent('Loaded User');
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
    });
  });

  it('prevents stale in-flight users list responses from clobbering successful creates', async () => {
    await loginSuccessfully();
    const usersLookup = deferredResponse();
    fetchMock.mockImplementationOnce(() => usersLookup.promise);

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await screen.findByRole('status');

    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        ...organizationUserRecord({ displayName: 'Created User', email: 'created@example.com', userId: '018f0000-0000-7000-8000-000000000012' }),
        temporary_password: 'created-secret',
      }, 201)
    );
    fireEvent.click(screen.getByRole('button', { name: /^Add/i }));
    const drawer = screen.getByLabelText(/add user/i);
    fireEvent.change(within(drawer).getByLabelText(/^Name$/i), { target: { value: 'Created User' } });
    fireEvent.change(within(drawer).getByLabelText(/^Email$/i), { target: { value: 'created@example.com' } });
    fireEvent.click(within(drawer).getByRole('button', { name: /^Save$/i }));

    expect(await screen.findByText('Created User')).toBeInTheDocument();

    await act(async () => {
      usersLookup.resolve(jsonResponse(organizationUsersResponse([
        organizationUserRecord({ displayName: 'Stale List User', email: 'stale-list@example.com' }),
      ])));
    });

    await waitFor(() => {
      expect(screen.getByRole('table', { name: /workspace users/i })).toHaveTextContent('Created User');
      expect(screen.getByRole('table', { name: /workspace users/i })).not.toHaveTextContent('Stale List User');
    });
  });

  it('role options are owner, admin, member, and viewer only', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse()));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await screen.findByText('Dev User');

    const roleFilter = screen.getByLabelText(/filter by role/i) as HTMLSelectElement;
    expect(Array.from(roleFilter.options).map((option) => option.value)).toEqual(['all', 'owner', 'admin', 'member', 'viewer']);

    fireEvent.click(screen.getByRole('button', { name: /^Add/i }));
    const drawer = screen.getByLabelText(/add user/i);
    const roleSelect = within(drawer).getByLabelText(/^Role$/i) as HTMLSelectElement;
    expect(Array.from(roleSelect.options).map((option) => option.value)).toEqual(['owner', 'admin', 'member', 'viewer']);
    expect(document.body).not.toHaveTextContent(/Observer/);
    expect(screen.queryByRole('option', { name: /observer/i })).not.toBeInTheDocument();
  });

  it('add user calls POST, displays a one-time temporary password, and does not persist it', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await screen.findByText(/No organization users returned/i);

    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        ...organizationUserRecord({ displayName: 'New Admin', email: 'new@example.com', role: 'admin', mustChangePassword: true }),
        temporary_password: 'shown-once-secret',
      }, 201)
    );

    fireEvent.click(screen.getByRole('button', { name: /^Add/i }));
    const drawer = screen.getByLabelText(/add user/i);
    fireEvent.change(within(drawer).getByLabelText(/^Name$/i), { target: { value: 'New Admin' } });
    fireEvent.change(within(drawer).getByLabelText(/^Email$/i), { target: { value: 'new@example.com' } });
    fireEvent.change(within(drawer).getByLabelText(/^Role$/i), { target: { value: 'admin' } });
    fireEvent.click(within(drawer).getByRole('button', { name: /^Save$/i }));

    expect(await screen.findByText('shown-once-secret')).toBeInTheDocument();
    expect(screen.getByText(/shown once/i)).toBeInTheDocument();
    expect(fetchMock.mock.calls[3][0]).toBe('/v1/organizations/018f0000-0000-7000-8000-000000000002/users');
    expect(fetchMock.mock.calls[3][1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        credentials: 'omit',
        headers: {
          'Content-Type': 'application/json',
          Authorization: 'Bearer access-token',
        },
        body: JSON.stringify({
          email: 'new@example.com',
          display_name: 'New Admin',
          role: 'admin',
        }),
      })
    );
    expect(screen.getByRole('table', { name: /workspace users/i })).toHaveTextContent('New Admin');
    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(window.localStorage.length).toBe(0);
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
    expect(window.sessionStorage.length).toBe(0);

    fireEvent.click(screen.getByRole('button', { name: /dismiss/i }));
    expect(screen.queryByText('shown-once-secret')).not.toBeInTheDocument();
    expect(screen.getByRole('table', { name: /workspace users/i })).not.toHaveTextContent('shown-once-secret');
  });

  it('conflict create shows the conflict error and does not display a temporary password', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await screen.findByText(/No organization users returned/i);
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('organization_user_exists', 'organization user already exists'), 409));

    fireEvent.click(screen.getByRole('button', { name: /^Add/i }));
    const drawer = screen.getByLabelText(/add user/i);
    fireEvent.change(within(drawer).getByLabelText(/^Name$/i), { target: { value: 'Existing User' } });
    fireEvent.change(within(drawer).getByLabelText(/^Email$/i), { target: { value: 'existing@example.com' } });
    fireEvent.click(within(drawer).getByRole('button', { name: /^Save$/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('This organization user already exists.');
    expect(screen.queryByText(/Temporary password shown once/i)).not.toBeInTheDocument();
    expect(document.body).not.toHaveTextContent(/shown-once-secret/);
  });

  it('edit user calls PATCH with backend role and state vocabulary', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([
      organizationUserRecord({ displayName: 'Dev User', email: 'dev@example.com', role: 'member', userId: '018f0000-0000-7000-8000-000000000010' }),
    ])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await screen.findByText('Dev User');
    fetchMock.mockResolvedValueOnce(
      jsonResponse(organizationUserRecord({ displayName: 'Dev Lead', email: 'dev@example.com', role: 'viewer', state: 'inactive', userId: '018f0000-0000-7000-8000-000000000010' }))
    );

    fireEvent.click(screen.getByRole('button', { name: /edit dev user/i }));
    const drawer = screen.getByLabelText(/edit user/i);
    fireEvent.change(within(drawer).getByLabelText(/^Name$/i), { target: { value: 'Dev Lead' } });
    fireEvent.change(within(drawer).getByLabelText(/^Role$/i), { target: { value: 'viewer' } });
    fireEvent.change(within(drawer).getByLabelText(/^Status$/i), { target: { value: 'inactive' } });
    fireEvent.click(within(drawer).getByRole('button', { name: /^Save$/i }));

    await screen.findByText('Dev Lead');
    expect(fetchMock.mock.calls[3][0]).toBe(
      '/v1/organizations/018f0000-0000-7000-8000-000000000002/users/018f0000-0000-7000-8000-000000000010'
    );
    expect(fetchMock.mock.calls[3][1]).toEqual(
      expect.objectContaining({
        method: 'PATCH',
        credentials: 'omit',
        headers: {
          'Content-Type': 'application/json',
          Authorization: 'Bearer access-token',
        },
        body: JSON.stringify({
          display_name: 'Dev Lead',
          role: 'viewer',
          state: 'inactive',
        }),
      })
    );
    expect(screen.getByRole('table', { name: /workspace users/i })).toHaveTextContent('Inactive');
  });

  it('resets an existing user temporary password after confirmation and displays it once', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([
      organizationUserRecord({ displayName: 'Dev User', email: 'dev@example.com', role: 'member', userId: '018f0000-0000-7000-8000-000000000010' }),
    ])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await screen.findByText('Dev User');

    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        ...organizationUserRecord({
          displayName: 'Dev User',
          email: 'dev@example.com',
          role: 'member',
          mustChangePassword: true,
          userId: '018f0000-0000-7000-8000-000000000010',
        }),
        temporary_password: 'rotated-once-secret',
      }, 201)
    );

    fireEvent.click(screen.getByRole('button', { name: /edit dev user/i }));
    const drawer = screen.getByLabelText(/edit user/i);
    fireEvent.click(within(drawer).getByRole('button', { name: /^Reset temporary password$/i }));
    const confirmDialog = screen.getByRole('dialog', { name: /confirm temporary password reset/i });
    fireEvent.click(within(confirmDialog).getByRole('button', { name: /^Reset temporary password$/i }));

    expect(await screen.findByText('rotated-once-secret')).toBeInTheDocument();
    expect(fetchMock.mock.calls[3][0]).toBe(
      '/v1/organizations/018f0000-0000-7000-8000-000000000002/users/018f0000-0000-7000-8000-000000000010/temporary-password-resets'
    );
    expect(fetchMock.mock.calls[3][1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        credentials: 'omit',
        headers: {
          Authorization: 'Bearer access-token',
        },
      })
    );
    expect(fetchMock.mock.calls[3][1]?.body).toBeUndefined();
    expect(screen.getByRole('table', { name: /workspace users/i })).toHaveTextContent('Must change password');
    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(window.localStorage.length).toBe(0);
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
    expect(window.sessionStorage.length).toBe(0);

    fireEvent.click(screen.getByRole('button', { name: /dismiss/i }));
    expect(screen.queryByText('rotated-once-secret')).not.toBeInTheDocument();
    expect(screen.getByRole('table', { name: /workspace users/i })).not.toHaveTextContent('rotated-once-secret');
  });

  it('blocks self temporary password reset before calling the API', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([
      organizationUserRecord({ displayName: 'Owner', email: 'owner@example.com', role: 'owner', userId: '018f0000-0000-7000-8000-000000000001' }),
    ])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await waitFor(() => {
      expect(screen.getByRole('table', { name: /workspace users/i })).toHaveTextContent('owner@example.com');
    });

    fireEvent.click(screen.getByRole('button', { name: /edit owner/i }));
    const drawer = screen.getByLabelText(/edit user/i);
    expect(within(drawer).getAllByText(/Use Change password for your own password/i).length).toBeGreaterThan(0);
    const resetButton = within(drawer).getByRole('button', { name: /^Reset temporary password$/i });
    expect(resetButton).toBeDisabled();
    fireEvent.click(resetButton);

    expect(screen.queryByRole('dialog', { name: /confirm temporary password reset/i })).not.toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('blocks self owner downgrade before calling PATCH', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([
      organizationUserRecord({ displayName: 'Owner', email: 'owner@example.com', role: 'owner', userId: '018f0000-0000-7000-8000-000000000001' }),
      organizationUserRecord({ displayName: 'Second Owner', email: 'second@example.com', role: 'owner', userId: '018f0000-0000-7000-8000-000000000020' }),
    ])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await waitFor(() => {
      expect(screen.getByRole('table', { name: /workspace users/i })).toHaveTextContent('owner@example.com');
    });

    fireEvent.click(screen.getByRole('button', { name: /edit owner/i }));
    const drawer = screen.getByLabelText(/edit user/i);
    fireEvent.change(within(drawer).getByLabelText(/^Role$/i), { target: { value: 'admin' } });
    fireEvent.click(within(drawer).getByRole('button', { name: /^Save$/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('You cannot remove your own owner access');
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('blocks self membership deactivation before calling PATCH', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([
      organizationUserRecord({ displayName: 'Owner', email: 'owner@example.com', role: 'owner', userId: '018f0000-0000-7000-8000-000000000001' }),
      organizationUserRecord({ displayName: 'Second Owner', email: 'second@example.com', role: 'owner', userId: '018f0000-0000-7000-8000-000000000020' }),
    ])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await waitFor(() => {
      expect(screen.getByRole('table', { name: /workspace users/i })).toHaveTextContent('owner@example.com');
    });

    fireEvent.click(screen.getByRole('button', { name: /edit owner/i }));
    const drawer = screen.getByLabelText(/edit user/i);
    fireEvent.change(within(drawer).getByLabelText(/^Status$/i), { target: { value: 'inactive' } });
    fireEvent.click(within(drawer).getByRole('button', { name: /^Save$/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('You cannot deactivate your own organization membership');
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('allows self display name edit while preserving owner role and active state', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([
      organizationUserRecord({ displayName: 'Owner', email: 'owner@example.com', role: 'owner', userId: '018f0000-0000-7000-8000-000000000001' }),
    ])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await waitFor(() => {
      expect(screen.getByRole('table', { name: /workspace users/i })).toHaveTextContent('owner@example.com');
    });
    fetchMock.mockResolvedValueOnce(
      jsonResponse(organizationUserRecord({ displayName: 'Owner Updated', email: 'owner@example.com', role: 'owner', userId: '018f0000-0000-7000-8000-000000000001' }))
    );

    fireEvent.click(screen.getByRole('button', { name: /edit owner/i }));
    const drawer = screen.getByLabelText(/edit user/i);
    fireEvent.change(within(drawer).getByLabelText(/^Name$/i), { target: { value: 'Owner Updated' } });
    fireEvent.click(within(drawer).getByRole('button', { name: /^Save$/i }));

    await screen.findByText('Owner Updated');
    expect(fetchMock.mock.calls[3][0]).toBe(
      '/v1/organizations/018f0000-0000-7000-8000-000000000002/users/018f0000-0000-7000-8000-000000000001'
    );
    expect(fetchMock.mock.calls[3][1]).toEqual(
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({
          display_name: 'Owner Updated',
          role: 'owner',
          state: 'active',
        }),
      })
    );
  });

  it('edit user does not require email because PATCH does not update email', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(organizationUsersResponse([
      organizationUserRecord({ displayName: 'No Email User', email: null, role: 'member', userId: '018f0000-0000-7000-8000-000000000011' }),
    ])));

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    await screen.findByText('No Email User');
    fetchMock.mockResolvedValueOnce(
      jsonResponse(organizationUserRecord({ displayName: 'No Email User', email: null, role: 'admin', userId: '018f0000-0000-7000-8000-000000000011' }))
    );

    fireEvent.click(screen.getByRole('button', { name: /edit no email user/i }));
    const drawer = screen.getByLabelText(/edit user/i);
    expect(within(drawer).getByLabelText(/^Email$/i)).toHaveValue('');
    fireEvent.change(within(drawer).getByLabelText(/^Role$/i), { target: { value: 'admin' } });
    fireEvent.click(within(drawer).getByRole('button', { name: /^Save$/i }));

    await waitFor(() => {
      expect(screen.getByRole('table', { name: /workspace users/i })).toHaveTextContent('Admin');
    });
    expect(fetchMock.mock.calls[3][0]).toBe(
      '/v1/organizations/018f0000-0000-7000-8000-000000000002/users/018f0000-0000-7000-8000-000000000011'
    );
    expect(fetchMock.mock.calls[3][1]).toEqual(
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({
          display_name: 'No Email User',
          role: 'admin',
          state: 'active',
        }),
      })
    );
  });

  it('does not render disallowed product or platform features', async () => {
    await loginSuccessfully();

    expect(document.body).not.toHaveTextContent(
      /registration|register|sign up|sso|invite|reset password|password reset|analytics|chat|file upload|model selector|organization creation|repo integration|runner|gate queue/i
    );
  });
});

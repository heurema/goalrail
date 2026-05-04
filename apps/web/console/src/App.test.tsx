import { fireEvent, screen, waitFor, within } from '@testing-library/react';
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
    ['en', 'database_not_configured', 'Goalrail server is not connected to a database yet. Check backend configuration.'],
    ['ru', 'auth_not_configured', 'На сервере не настроена авторизация. Проверьте GOALRAIL_AUTH_JWT_SECRET.'],
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

  it('does not write access tokens, refresh tokens, profile data, or auth state to browser storage', async () => {
    await loginSuccessfully();

    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(window.localStorage.length).toBe(0);
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
    expect(window.sessionStorage.length).toBe(0);
  });

  it('keeps exactly three product surfaces with honest EN and RU structured empty states', async () => {
    await loginSuccessfully();

    const navigation = screen.getByRole('navigation', { name: /product surfaces/i });
    const productButtons = within(navigation).getAllByRole('button');

    expect(productButtons).toHaveLength(3);
    expect(screen.getByRole('button', { name: /^Contracts$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: /^Delivery Readiness$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Proof$/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/contracts: structured empty state/i)).toBeInTheDocument();
    expect(screen.getByText('Goal → Contract → Task → Proof')).toBeInTheDocument();

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

  it('keeps users component-state-only with neutral role and status values', async () => {
    await loginSuccessfully();
    const callsBeforeUsersEdit = fetchMock.mock.calls.length;

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Users$/i }));
    fireEvent.click(screen.getByRole('button', { name: /add user/i }));

    const drawer = screen.getByRole('complementary', { name: /add user/i });
    expect(drawer).toBeInTheDocument();

    fireEvent.change(within(drawer).getByLabelText(/^Name$/i), { target: { value: 'QA Lead' } });
    fireEvent.change(within(drawer).getByLabelText(/^Email$/i), { target: { value: 'qa@example.com' } });
    fireEvent.change(within(drawer).getByLabelText(/^Role$/i), { target: { value: 'observer' } });
    fireEvent.change(within(drawer).getByLabelText(/^Status$/i), { target: { value: 'disabled' } });
    fireEvent.click(screen.getByRole('button', { name: /^Save$/i }));

    expect(screen.getByText('QA Lead')).toBeInTheDocument();
    expect(screen.getAllByText('Observer').length).toBeGreaterThan(1);
    expect(screen.getAllByText('Disabled').length).toBeGreaterThan(1);
    expect(fetchMock).toHaveBeenCalledTimes(callsBeforeUsersEdit);

    fireEvent.change(screen.getByLabelText(/filter by role/i), { target: { value: 'observer' } });
    fireEvent.change(screen.getByLabelText(/filter by status/i), { target: { value: 'active' } });

    await waitFor(() => expect(screen.queryByText('QA Lead')).not.toBeInTheDocument());
  });

  it('does not render disallowed product or platform features', async () => {
    await loginSuccessfully();

    expect(document.body).not.toHaveTextContent(
      /registration|register|sign up|sso|invite|reset password|password reset|analytics|chat|file upload|model selector|organization creation|repo integration|runner|gate queue/i
    );
  });
});

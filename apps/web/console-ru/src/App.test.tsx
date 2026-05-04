import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Mock } from 'vitest';

import App from './App';
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

function meResponse() {
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
      role: 'owner',
      state: 'active',
    },
  };
}

function errorEnvelope(code: string, message = 'error') {
  return {
    error: { code, message },
  };
}

async function loginSuccessfully() {
  fetchMock.mockResolvedValueOnce(jsonResponse(loginResponse()));
  fetchMock.mockResolvedValueOnce(jsonResponse(meResponse()));
  render(<App />);

  fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
  fireEvent.change(screen.getByLabelText(/^Пароль$/i), { target: { value: 'password' } });
  fireEvent.click(screen.getByRole('button', { name: /войти/i }));

  await screen.findByRole('navigation', { name: /разделы продукта/i });
}

async function startPasswordChangeLogin() {
  fetchMock.mockResolvedValueOnce(jsonResponse(loginResponse({ must_change_password: true })));
  render(<App />);

  fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
  fireEvent.change(screen.getByLabelText(/^Пароль$/i), { target: { value: 'temporary-password' } });
  fireEvent.click(screen.getByRole('button', { name: /войти/i }));

  await screen.findByRole('form', { name: /смена временного пароля/i });
}

describe('App', () => {
  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    window.localStorage.clear();
    window.sessionStorage.clear();
    asMock(window.localStorage.clear).mockClear();
    asMock(window.localStorage.getItem).mockClear();
    asMock(window.localStorage.setItem).mockClear();
    asMock(window.sessionStorage.clear).mockClear();
    asMock(window.sessionStorage.getItem).mockClear();
    asMock(window.sessionStorage.setItem).mockClear();
  });

  it('renders a login-only entry screen without registration, SSO, or password reset', () => {
    render(<App />);

    const brand = screen.getByLabelText(/консоль goalrail/i);

    expect(brand.tagName).toBe('DIV');
    expect(brand).toHaveTextContent(/^GOALRAIL$/);
    expect(brand.querySelector('svg.brandMark')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /консоль goalrail/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /^GoalRail Console$/ })).not.toBeInTheDocument();
    expect(screen.queryByText(/вход в рабочее пространство/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/доступ выдает администратор/i)).not.toBeInTheDocument();
    expect(screen.getByLabelText(/^Email$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Пароль$/i)).toBeInTheDocument();
    expect(screen.queryByText(/регистрация|зарегистрироваться|sign up|sso|reset|forgot|сброс|восстанов/i)).not.toBeInTheDocument();
  });

  it('keeps empty login fields client-side and does not call the API', () => {
    render(<App />);

    fireEvent.click(screen.getByRole('button', { name: /войти/i }));

    expect(fetchMock).not.toHaveBeenCalled();
    expect(screen.getByRole('alert')).toHaveTextContent('Введите email и пароль для продолжения.');
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
    expect(screen.getByLabelText(/текущий пользователь/i)).toHaveTextContent('Owner');
    expect(screen.getByLabelText(/текущий пользователь/i)).toHaveTextContent('owner');
  });

  it('invalid credentials show a Russian error and do not render the console', async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('invalid_credentials'), 401));
    render(<App />);

    fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
    fireEvent.change(screen.getByLabelText(/^Пароль$/i), { target: { value: 'wrong' } });
    fireEvent.click(screen.getByRole('button', { name: /войти/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Неверный email или пароль.');
    expect(screen.queryByRole('navigation', { name: /разделы продукта/i })).not.toBeInTheDocument();
  });

  it.each([
    ['database_not_configured', 'Сервер Goalrail пока не подключен к базе данных. Проверьте конфигурацию backend.'],
    ['auth_not_configured', 'На сервере не настроена авторизация. Проверьте GOALRAIL_AUTH_JWT_SECRET.'],
  ])('%s shows an honest operational error and does not render the console', async (code, message) => {
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope(code), 503));
    render(<App />);

    fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
    fireEvent.change(screen.getByLabelText(/^Пароль$/i), { target: { value: 'password' } });
    fireEvent.click(screen.getByRole('button', { name: /войти/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent(message);
    expect(screen.queryByRole('navigation', { name: /разделы продукта/i })).not.toBeInTheDocument();
  });

  it('must_change_password shows a password-change form before the console', async () => {
    await startPasswordChangeLogin();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(screen.getByLabelText(/^Текущий пароль$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Новый пароль$/i)).toBeInTheDocument();
    expect(screen.queryByRole('navigation', { name: /разделы продукта/i })).not.toBeInTheDocument();
  });

  it('password change calls the backend, then /v1/me, then renders the console', async () => {
    await startPasswordChangeLogin();
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

    fireEvent.change(screen.getByLabelText(/^Текущий пароль$/i), { target: { value: 'wrong-current' } });
    fireEvent.change(screen.getByLabelText(/^Новый пароль$/i), { target: { value: 'new-password' } });
    fireEvent.click(screen.getByRole('button', { name: /сменить пароль/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Текущий пароль неверен.');
    expect(screen.getByRole('form', { name: /смена временного пароля/i })).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('logout calls the backend, clears auth state, and returns to login', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse({ revoked: true }));

    fireEvent.click(screen.getByRole('button', { name: /выйти/i }));

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
    expect(screen.queryByRole('navigation', { name: /разделы продукта/i })).not.toBeInTheDocument();
  });

  it('does not write access tokens, refresh tokens, or profile data to browser storage', async () => {
    await loginSuccessfully();

    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(window.localStorage.length).toBe(0);
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
    expect(window.sessionStorage.length).toBe(0);
  });

  it('keeps exactly three product surfaces with honest structured empty states', async () => {
    await loginSuccessfully();

    const navigation = screen.getByRole('navigation', { name: /разделы продукта/i });
    const productButtons = within(navigation).getAllByRole('button');

    expect(navigation).toBeInTheDocument();
    expect(productButtons).toHaveLength(3);
    expect(screen.getByRole('button', { name: /^Контракты$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: /^Оценка готовности$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Проверка результата$/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/контракты: structured empty state/i)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Контракты$/i })).toBeInTheDocument();
    expect(screen.getByText('Goal → Contract → Task → Proof')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Оценка готовности$/i }));

    expect(screen.getByText(/хватает ли контекста/i)).toBeInTheDocument();
    expect(screen.getByText('НЕ ПРОВЕРЯЛОСЬ')).toBeInTheDocument();
    expect(screen.getByText(/Открытые вопросы, которые блокируют уверенный handoff/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Проверка результата$/i }));

    expect(screen.getByText(/проверки execution evidence через gate/i)).toBeInTheDocument();
    expect(screen.getByText('ОЖИДАЕТ VERIFIED EVIDENCE')).toBeInTheDocument();
    expect(screen.getByText(/Сохранили ли проверки и evidence доверие/i)).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent(
      /trialops-demo|C-0147|readiness score|\/100|\bscan\b|proof queue|fake queue|fake pass|fake fail|pass\/fail/i
    );
  });

  it('opens appearance settings by default without making it a product surface', async () => {
    await loginSuccessfully();

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));

    expect(screen.getByLabelText(/настройки: оформление/i)).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: /разделы продукта/i })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: /разделы настроек/i })).toBeInTheDocument();
    expect(screen.queryByText(/^Раздел$/i)).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Оформление$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Контракты$/i })).not.toHaveAttribute('aria-current', 'page');
    expect(screen.queryByText(/preview|local-only|local UI|sessions|cookies|будущ/i)).not.toBeInTheDocument();
  });

  it('renders all theme presets and applies the selected theme to the shell', async () => {
    await loginSuccessfully();

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'goalrail-default');

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));

    expect(screen.getByRole('button', { name: /Goalrail Default/i })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('button', { name: /Catppuccin Mocha/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Dracula/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Nord/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Solarized Dark/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Gruvbox Dark/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Nord/i }));

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'nord');
    expect(screen.getByRole('button', { name: /Nord/i })).toHaveAttribute('aria-pressed', 'true');
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe('nord');
    expect(asMock(window.localStorage.setItem).mock.calls).toEqual([[THEME_STORAGE_KEY, 'nord']]);
  });

  it('initializes with a stored valid theme', async () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, 'solarized-dark');

    await loginSuccessfully();

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'solarized-dark');

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));

    expect(screen.getByRole('button', { name: /Solarized Dark/i })).toHaveAttribute('aria-pressed', 'true');
  });

  it('falls back to the default theme when stored theme is invalid', async () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, 'unknown-theme');

    await loginSuccessfully();

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'goalrail-default');

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));

    expect(screen.getByRole('button', { name: /Goalrail Default/i })).toHaveAttribute('aria-pressed', 'true');
  });

  it('opens users inside settings after theme switching', async () => {
    await loginSuccessfully();

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));
    fireEvent.click(screen.getByRole('button', { name: /Nord/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Пользователи$/i }));

    expect(screen.getByLabelText(/настройки: пользователи/i)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Пользователи$/i })).toBeInTheDocument();
    expect(screen.getByRole('table', { name: /пользователи рабочего пространства/i })).toBeInTheDocument();
  });

  it('adds and edits users in the settings drawer with component state only', async () => {
    await loginSuccessfully();
    const callsBeforeUsersEdit = fetchMock.mock.calls.length;

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Пользователи$/i }));
    fireEvent.click(screen.getByRole('button', { name: /добавить пользователя/i }));

    expect(screen.getByRole('complementary', { name: /добавить пользователя/i })).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/^Имя$/i), { target: { value: 'QA Lead' } });
    fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'qa@example.com' } });
    fireEvent.click(screen.getByRole('button', { name: /^Сохранить$/i }));

    expect(screen.getByText('QA Lead')).toBeInTheDocument();
    expect(screen.getByText('qa@example.com')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /редактировать QA Lead/i }));
    fireEvent.change(screen.getByLabelText(/^Имя$/i), { target: { value: 'QA Owner' } });
    fireEvent.click(screen.getByRole('button', { name: /^Сохранить$/i }));

    expect(screen.getByText('QA Owner')).toBeInTheDocument();
    expect(screen.queryByText('QA Lead')).not.toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(callsBeforeUsersEdit);
  });

  it('filters users by search, role, and status', async () => {
    await loginSuccessfully();

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Пользователи$/i }));
    expect(screen.getByRole('table', { name: /пользователи рабочего пространства/i })).toBeInTheDocument();
    expect(screen.getByRole('searchbox', { name: /поиск пользователей/i })).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText(/поиск по имени или email/i), {
      target: { value: 'reviewer' },
    });

    expect(screen.getByText('Reviewer')).toBeInTheDocument();
    expect(screen.queryByText('Product Lead')).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText(/поиск по имени или email/i), {
      target: { value: '' },
    });
    fireEvent.change(screen.getByLabelText(/фильтр по роли/i), {
      target: { value: 'Участник' },
    });

    expect(screen.getByText('Product Lead')).toBeInTheDocument();
    expect(screen.queryByText('Reviewer')).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/фильтр по статусу/i), {
      target: { value: 'Активен' },
    });

    await waitFor(() => expect(screen.getByText(/нет пользователей/i)).toBeInTheDocument());
  });
});

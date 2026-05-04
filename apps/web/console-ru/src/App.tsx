import { FormEvent, useMemo, useState } from 'react';

import {
  changePassword,
  isAuthClientError,
  login as loginWithPassword,
  logout as logoutSession,
  me as fetchCurrentProfile,
} from './authClient';
import type { AuthClientError, LoginResponse, MeResponse } from './authClient';

import './App.css';

type SurfaceId = 'contracts' | 'delivery-readiness' | 'proof';
type ScreenId = 'console' | 'settings-appearance' | 'settings-users';
type ThemeId = 'goalrail-default' | 'catppuccin-mocha' | 'dracula' | 'nord' | 'solarized-dark' | 'gruvbox-dark';
type UserStatus = 'Активен' | 'Ожидает' | 'Отключен';
type UserRole = 'Владелец' | 'Участник' | 'Наблюдатель';
type RoleFilter = UserRole | 'all';
type StatusFilter = UserStatus | 'all';
type AuthStatus =
  | 'unauthenticated'
  | 'logging_in'
  | 'password_change_required'
  | 'changing_password'
  | 'authenticated'
  | 'logging_out';

interface TokenState {
  userId: string;
  accessToken: string;
  accessTokenExpiresAt: string;
  tokenType: 'Bearer';
  refreshToken: string;
  refreshTokenExpiresAt: string;
}

interface SurfaceItem {
  id: SurfaceId;
  label: string;
}

interface ConsoleUser {
  id: string;
  name: string;
  email: string;
  role: UserRole;
  status: UserStatus;
}

interface ThemePreset {
  id: ThemeId;
  label: string;
  swatches: string[];
}

interface SurfaceLane {
  title: string;
  body: string;
}

interface SurfaceEmptyState {
  label: string;
  kicker: string;
  copy: string;
  lanes: SurfaceLane[];
  status?: string;
  footer?: string;
}

const SURFACES: SurfaceItem[] = [
  { id: 'contracts', label: 'Контракты' },
  { id: 'delivery-readiness', label: 'Оценка готовности' },
  { id: 'proof', label: 'Проверка результата' },
];

const THEMES: ThemePreset[] = [
  { id: 'goalrail-default', label: 'Goalrail Default', swatches: ['#201f1d', '#2d2b28', '#e8e0d2', '#c783a8', '#92b66f'] },
  { id: 'catppuccin-mocha', label: 'Catppuccin Mocha', swatches: ['#1e1e2e', '#313244', '#cdd6f4', '#cba6f7', '#a6e3a1'] },
  { id: 'dracula', label: 'Dracula', swatches: ['#282a36', '#44475a', '#f8f8f2', '#bd93f9', '#50fa7b'] },
  { id: 'nord', label: 'Nord', swatches: ['#2e3440', '#3b4252', '#eceff4', '#88c0d0', '#a3be8c'] },
  { id: 'solarized-dark', label: 'Solarized Dark', swatches: ['#002b36', '#073642', '#eee8d5', '#268bd2', '#859900'] },
  { id: 'gruvbox-dark', label: 'Gruvbox Dark', swatches: ['#282828', '#3c3836', '#ebdbb2', '#fe8019', '#b8bb26'] },
];

const THEME_STORAGE_KEY = 'goalrail.console.theme';

const AUTH_ERROR_MESSAGES: Record<string, string> = {
  invalid_credentials: 'Неверный email или пароль.',
  invalid_current_password: 'Текущий пароль неверен.',
  database_not_configured: 'Сервер Goalrail пока не подключен к базе данных. Проверьте конфигурацию backend.',
  auth_not_configured: 'На сервере не настроена авторизация. Проверьте GOALRAIL_AUTH_JWT_SECRET.',
  membership_required: 'Для этого пользователя нет активного доступа к организации.',
  inactive_user: 'Пользователь отключен.',
  unauthorized: 'Сессия недействительна. Войдите заново.',
  network_error: 'Не удалось связаться с сервером Goalrail.',
  response_parse_error: 'Сервер Goalrail вернул нераспознаваемый ответ.',
};

const SURFACE_EMPTY_STATES: Record<SurfaceId, SurfaceEmptyState> = {
  contracts: {
    label: 'Контракты',
    kicker: 'contract contour',
    copy:
      'Контракты появятся здесь после подключения server-backed flow. Каждый контракт фиксирует границу между бизнес-целью и delivery-работой.',
    lanes: [
      { title: 'Намерение', body: 'Нормализованная цель и контекст владельца.' },
      { title: 'Scope', body: 'Что входит, что не входит, ограничения.' },
      { title: 'Приемка', body: 'Критерии, без которых работа не должна двигаться дальше.' },
      { title: 'Handoff', body: 'Ограниченный task packet для delivery.' },
    ],
    footer: 'Goal → Contract → Task → Proof',
  },
  'delivery-readiness': {
    label: 'Оценка готовности',
    kicker: 'readiness contour',
    copy: 'Здесь будет видно, хватает ли контекста, чтобы превратить цель в delivery contract.',
    lanes: [
      { title: 'Контекст', body: 'Что известно о цели и владельце.' },
      { title: 'Ограничения', body: 'Лимиты, non-goals и policy boundaries.' },
      { title: 'Приемка', body: 'Ожидаемый результат и proof expectations.' },
      { title: 'Риски', body: 'Открытые вопросы, которые блокируют уверенный handoff.' },
    ],
    status: 'НЕ ПРОВЕРЯЛОСЬ',
  },
  proof: {
    label: 'Проверка результата',
    kicker: 'verification contour',
    copy: 'Proof появится здесь после проверки execution evidence через gate.',
    lanes: [
      { title: 'Scope', body: 'Осталась ли работа внутри утвержденного контракта?' },
      { title: 'Integrity', body: 'Сохранили ли проверки и evidence доверие к результату?' },
      { title: 'Policy', body: 'Соблюдены ли заданные boundaries?' },
      { title: 'Target', body: 'Достигнут ли ожидаемый outcome?' },
    ],
    status: 'ОЖИДАЕТ VERIFIED EVIDENCE',
  },
};

const INITIAL_USERS: ConsoleUser[] = [
  { id: 'u1', name: 'Owner', email: 'owner@example.com', role: 'Владелец', status: 'Активен' },
  { id: 'u2', name: 'Product Lead', email: 'product@example.com', role: 'Участник', status: 'Ожидает' },
  { id: 'u3', name: 'Reviewer', email: 'reviewer@example.com', role: 'Наблюдатель', status: 'Активен' },
];

const EMPTY_DRAFT: Omit<ConsoleUser, 'id'> = {
  name: '',
  email: '',
  role: 'Участник',
  status: 'Ожидает',
};

function initials(name: string) {
  return name
    .split(' ')
    .filter(Boolean)
    .map((part) => part[0])
    .slice(0, 2)
    .join('')
    .toUpperCase();
}

function statusClass(status: UserStatus) {
  if (status === 'Активен') {
    return 'statusActive';
  }

  if (status === 'Ожидает') {
    return 'statusPending';
  }

  return 'statusDisabled';
}

function isThemeId(value: string | null): value is ThemeId {
  return THEMES.some((theme) => theme.id === value);
}

function readStoredTheme(): ThemeId {
  try {
    if (typeof window === 'undefined') {
      return 'goalrail-default';
    }

    const storedTheme = window.localStorage.getItem(THEME_STORAGE_KEY);
    return isThemeId(storedTheme) ? storedTheme : 'goalrail-default';
  } catch {
    return 'goalrail-default';
  }
}

function persistTheme(themeId: ThemeId) {
  try {
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(THEME_STORAGE_KEY, themeId);
    }
  } catch {
    // localStorage can be unavailable in restricted browser contexts.
  }
}

function tokenStateFromLogin(result: LoginResponse): TokenState {
  return {
    userId: result.user_id,
    accessToken: result.access_token,
    accessTokenExpiresAt: result.access_token_expires_at,
    tokenType: result.token_type,
    refreshToken: result.refresh_token,
    refreshTokenExpiresAt: result.refresh_token_expires_at,
  };
}

function authErrorMessage(error: unknown) {
  if (isAuthClientError(error)) {
    return AUTH_ERROR_MESSAGES[error.code] ?? operationalErrorMessage(error);
  }

  return 'Сервер Goalrail вернул ошибку. Проверьте backend и повторите вход.';
}

function operationalErrorMessage(error: AuthClientError) {
  const suffix = error.status ? ` Код: ${error.status}.` : '';
  return `Сервер Goalrail вернул ошибку.${suffix}`;
}

function App() {
  const [authStatus, setAuthStatus] = useState<AuthStatus>('unauthenticated');
  const [authError, setAuthError] = useState('');
  const [passwordChangeError, setPasswordChangeError] = useState('');
  const [tokens, setTokens] = useState<TokenState | null>(null);
  const [profile, setProfile] = useState<MeResponse | null>(null);
  const [activeSurface, setActiveSurface] = useState<SurfaceId>('contracts');
  const [screen, setScreen] = useState<ScreenId>('console');
  const [activeTheme, setActiveTheme] = useState<ThemeId>(() => readStoredTheme());
  const [users, setUsers] = useState<ConsoleUser[]>(INITIAL_USERS);
  const [searchQuery, setSearchQuery] = useState('');
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState<Omit<ConsoleUser, 'id'>>(EMPTY_DRAFT);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  const activeLabel = SURFACES.find((surface) => surface.id === activeSurface)?.label ?? 'Контракты';
  const activeEmptyState = SURFACE_EMPTY_STATES[activeSurface];
  const drawerTitle = editingId ? 'Редактировать пользователя' : 'Добавить пользователя';
  const isLoginPending = authStatus === 'logging_in';
  const isPasswordChangePending = authStatus === 'changing_password';
  const isLoggingOut = authStatus === 'logging_out';
  const sessionDisplayName = profile?.user.display_name.trim() || profile?.user.email || 'Пользователь';
  const sessionEmail = profile?.user.email;
  const sessionRole = profile?.organization_membership.role ?? 'member';

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const email = String(form.get('email') ?? '').trim();
    const password = String(form.get('password') ?? '').trim();

    if (!email || !password) {
      setAuthError('Введите email и пароль для продолжения.');
      return;
    }

    setAuthError('');
    setPasswordChangeError('');
    setAuthStatus('logging_in');

    try {
      const loginResult = await loginWithPassword({ email, password });
      const nextTokens = tokenStateFromLogin(loginResult);
      setTokens(nextTokens);

      if (loginResult.must_change_password) {
        setAuthStatus('password_change_required');
        return;
      }

      const currentProfile = await fetchCurrentProfile(loginResult.access_token);
      setProfile(currentProfile);
      setAuthStatus('authenticated');
    } catch (error) {
      setTokens(null);
      setProfile(null);
      setAuthStatus('unauthenticated');
      setAuthError(authErrorMessage(error));
    }
  }

  async function handlePasswordChange(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const currentPassword = String(form.get('currentPassword') ?? '');
    const newPassword = String(form.get('newPassword') ?? '');

    if (!currentPassword || !newPassword.trim()) {
      setPasswordChangeError('Введите текущий и новый пароль.');
      return;
    }

    if (!tokens) {
      resetAuthState();
      setAuthError('Сессия недействительна. Войдите заново.');
      return;
    }

    setPasswordChangeError('');
    setAuthStatus('changing_password');

    try {
      await changePassword({
        accessToken: tokens.accessToken,
        currentPassword,
        newPassword,
      });
      const currentProfile = await fetchCurrentProfile(tokens.accessToken);
      setProfile(currentProfile);
      setAuthStatus('authenticated');
    } catch (error) {
      if (isAuthClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(authErrorMessage(error));
        return;
      }

      setAuthStatus('password_change_required');
      setPasswordChangeError(authErrorMessage(error));
    }
  }

  async function handleLogout() {
    const accessToken = tokens?.accessToken;
    if (!accessToken) {
      resetAuthState();
      return;
    }

    setAuthStatus('logging_out');
    try {
      await logoutSession(accessToken);
    } finally {
      resetAuthState();
    }
  }

  function resetAuthState() {
    setTokens(null);
    setProfile(null);
    setAuthStatus('unauthenticated');
    setAuthError('');
    setPasswordChangeError('');
    setScreen('console');
    setIsDrawerOpen(false);
  }

  function openNewUser() {
    setEditingId(null);
    setDraft(EMPTY_DRAFT);
    setIsDrawerOpen(true);
  }

  function openExistingUser(user: ConsoleUser) {
    setEditingId(user.id);
    setDraft({
      name: user.name,
      email: user.email,
      role: user.role,
      status: user.status,
    });
    setIsDrawerOpen(true);
  }

  function closeDrawer() {
    setIsDrawerOpen(false);
  }

  function saveUser() {
    const nextDraft = {
      ...draft,
      name: draft.name.trim(),
      email: draft.email.trim(),
    };

    if (!nextDraft.name || !nextDraft.email) {
      return;
    }

    if (editingId) {
      setUsers((currentUsers) =>
        currentUsers.map((user) => (user.id === editingId ? { ...user, ...nextDraft } : user))
      );
    } else {
      setUsers((currentUsers) => [
        ...currentUsers,
        {
          id: `u${currentUsers.length + 1}`,
          ...nextDraft,
        },
      ]);
    }

    setIsDrawerOpen(false);
  }

  function updateTheme(themeId: ThemeId) {
    setActiveTheme(themeId);
    persistTheme(themeId);
  }

  const visibleUsers = useMemo(() => {
    const normalizedQuery = searchQuery.trim().toLowerCase();

    return users.filter((user) => {
      const matchesQuery =
        !normalizedQuery ||
        user.name.toLowerCase().includes(normalizedQuery) ||
        user.email.toLowerCase().includes(normalizedQuery);
      const matchesRole = roleFilter === 'all' || user.role === roleFilter;
      const matchesStatus = statusFilter === 'all' || user.status === statusFilter;

      return matchesQuery && matchesRole && matchesStatus;
    });
  }, [roleFilter, searchQuery, statusFilter, users]);

  const userRows = useMemo(
    () =>
      visibleUsers.map((user) => (
        <tr className="userRow" key={user.id}>
          <td>
            <div className="userName">
              <span className="avatar" aria-hidden="true">
                {initials(user.name)}
              </span>
              <span>{user.name}</span>
            </div>
          </td>
          <td className="userEmail">{user.email}</td>
          <td>
            <span className={user.role === 'Владелец' ? 'pill roleOwner' : 'pill'}>{user.role}</span>
          </td>
          <td>
            <span className={`pill ${statusClass(user.status)}`}>{user.status}</span>
          </td>
          <td>
            <div className="userActions">
              <button className="iconButton" onClick={() => openExistingUser(user)} type="button">
                <span aria-hidden="true">✎</span>
                <span className="srOnly">Редактировать {user.name}</span>
              </button>
            </div>
          </td>
        </tr>
      )),
    [visibleUsers]
  );

  if (authStatus === 'unauthenticated' || authStatus === 'logging_in') {
    return (
      <main
        className="loginScreen"
        data-deployment-target="console.goalrail.ru"
        data-goalrail-theme={activeTheme}
      >
        <div className="loginRails" aria-hidden="true" />
        <form className="loginCard" onSubmit={handleLogin}>
          <Brand />

          <label className="field">
            <span>Email</span>
            <input autoComplete="email" disabled={isLoginPending} name="email" placeholder="name@example.com" type="email" />
          </label>

          <label className={authError ? 'field fieldError' : 'field'}>
            <span>Пароль</span>
            <input autoComplete="current-password" disabled={isLoginPending} name="password" type="password" />
          </label>

          {authError ? <p className="fieldMessage" role="alert">{authError}</p> : null}

          <button className="primaryButton fullWidth" disabled={isLoginPending} type="submit">
            {isLoginPending ? 'Входим...' : 'Войти'}
            <span aria-hidden="true">→</span>
          </button>

        </form>
      </main>
    );
  }

  if (authStatus === 'password_change_required' || authStatus === 'changing_password') {
    return (
      <main
        className="loginScreen"
        data-deployment-target="console.goalrail.ru"
        data-goalrail-theme={activeTheme}
      >
        <div className="loginRails" aria-hidden="true" />
        <form className="loginCard passwordChangeCard" onSubmit={handlePasswordChange} aria-label="Смена временного пароля">
          <Brand />

          <div className="authStateBlock">
            <p className="authStateLabel">Требуется смена пароля</p>
          </div>

          <label className={passwordChangeError ? 'field fieldError' : 'field'}>
            <span>Текущий пароль</span>
            <input autoComplete="current-password" disabled={isPasswordChangePending} name="currentPassword" type="password" />
          </label>

          <label className={passwordChangeError ? 'field fieldError' : 'field'}>
            <span>Новый пароль</span>
            <input autoComplete="new-password" disabled={isPasswordChangePending} name="newPassword" type="password" />
          </label>

          {passwordChangeError ? <p className="fieldMessage" role="alert">{passwordChangeError}</p> : null}

          <button className="primaryButton fullWidth" disabled={isPasswordChangePending} type="submit">
            {isPasswordChangePending ? 'Сохраняем...' : 'Сменить пароль'}
            <span aria-hidden="true">→</span>
          </button>
        </form>
      </main>
    );
  }

  return (
    <main
      className="consoleShell"
      data-deployment-target="console.goalrail.ru"
      data-goalrail-theme={activeTheme}
    >
      <aside className="sidebar" aria-label="Навигация консоли Goalrail">
        <Brand />

        <nav className="surfaceNav" aria-label="Разделы продукта">
          {SURFACES.map((surface) => (
            <button
              aria-current={screen === 'console' && activeSurface === surface.id ? 'page' : undefined}
              className={screen === 'console' && activeSurface === surface.id ? 'surfaceButton active' : 'surfaceButton'}
              key={surface.id}
              onClick={() => {
                setActiveSurface(surface.id);
                setScreen('console');
              }}
              type="button"
            >
              {surface.label}
            </button>
          ))}
        </nav>

        <div className="sidebarSpacer" />

        <section className="sessionPanel" aria-label="Текущий пользователь">
          <div>
            <p className="sessionName">{sessionDisplayName}</p>
            {sessionEmail ? <p className="sessionEmail">{sessionEmail}</p> : null}
            <p className="sessionRole">role · {sessionRole}</p>
          </div>
          <button className="ghostButton logoutButton" disabled={isLoggingOut} onClick={handleLogout} type="button">
            {isLoggingOut ? 'Выходим...' : 'Выйти'}
          </button>
        </section>

        <div className="settingsDock">
          <button
            aria-current={screen.startsWith('settings-') ? 'page' : undefined}
            className={screen.startsWith('settings-') ? 'settingsButton active' : 'settingsButton'}
            onClick={() => setScreen('settings-appearance')}
            type="button"
          >
            <span aria-hidden="true">⚙</span>
            <span>Настройки</span>
          </button>
        </div>
      </aside>

      {screen === 'console' ? (
        <SurfaceEmptyStatePanel state={activeEmptyState} label={activeLabel} />
      ) : (
        <section
          className="settingsSurface"
          aria-label={screen === 'settings-appearance' ? 'Настройки: оформление' : 'Настройки: пользователи'}
        >
          <header className="surfaceHeader">
            <div>
              <p className="kicker">{screen === 'settings-appearance' ? 'settings · appearance' : 'settings · users'}</p>
              <h2>Настройки</h2>
            </div>
            <p className="metaText">{screen === 'settings-appearance' ? `${THEMES.length} presets` : `${visibleUsers.length} записи`}</p>
          </header>

          <nav className="settingsSectionNav" aria-label="Разделы настроек">
            <button
              aria-current={screen === 'settings-appearance' ? 'page' : undefined}
              className={screen === 'settings-appearance' ? 'sectionButton active' : 'sectionButton'}
              onClick={() => setScreen('settings-appearance')}
              type="button"
            >
              Оформление
            </button>
            <button
              aria-current={screen === 'settings-users' ? 'page' : undefined}
              className={screen === 'settings-users' ? 'sectionButton active' : 'sectionButton'}
              onClick={() => setScreen('settings-users')}
              type="button"
            >
              Пользователи
            </button>
          </nav>

          <div className="settingsContent">
            {screen === 'settings-appearance' ? (
              <AppearanceSettings activeTheme={activeTheme} onThemeChange={updateTheme} />
            ) : (
              <>
                <div className="usersHeader">
                  <div>
                    <h3>Пользователи</h3>
                    <p>Управление доступом к рабочему пространству.</p>
                  </div>
                  <button aria-label="Добавить пользователя" className="primaryButton" onClick={openNewUser} type="button">
                    <span aria-hidden="true">+</span>
                    <span>Добавить</span>
                  </button>
                </div>

                <div className="usersToolbar">
                  <label className="searchBox">
                    <span aria-hidden="true">⌕</span>
                    <input
                      aria-label="Поиск пользователей"
                      onChange={(event) => setSearchQuery(event.target.value)}
                      placeholder="Поиск по имени или email"
                      type="search"
                      value={searchQuery}
                    />
                  </label>
                  <label className="filterBox">
                    <span>Роль</span>
                    <select
                      aria-label="Фильтр по роли"
                      onChange={(event) => setRoleFilter(event.target.value as RoleFilter)}
                      value={roleFilter}
                    >
                      <option value="all">Все роли</option>
                      <option value="Владелец">Владелец</option>
                      <option value="Участник">Участник</option>
                      <option value="Наблюдатель">Наблюдатель</option>
                    </select>
                  </label>
                  <label className="filterBox">
                    <span>Статус</span>
                    <select
                      aria-label="Фильтр по статусу"
                      onChange={(event) => setStatusFilter(event.target.value as StatusFilter)}
                      value={statusFilter}
                    >
                      <option value="all">Все статусы</option>
                      <option value="Активен">Активен</option>
                      <option value="Ожидает">Ожидает</option>
                      <option value="Отключен">Отключен</option>
                    </select>
                  </label>
                </div>

                <div className="usersTableFrame">
                  <table className="usersTable" aria-label="Пользователи рабочего пространства">
                    <thead>
                      <tr className="userRow userHead">
                        <th scope="col">Имя</th>
                        <th scope="col">Email</th>
                        <th scope="col">Роль</th>
                        <th scope="col">Статус</th>
                        <th scope="col" aria-label="Действия" />
                      </tr>
                    </thead>
                    <tbody>
                      {userRows}
                      {visibleUsers.length === 0 ? (
                        <tr>
                          <td className="emptyUsers" colSpan={5}>
                            Нет пользователей по выбранным условиям.
                          </td>
                        </tr>
                      ) : null}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </div>
        </section>
      )}

      {isDrawerOpen ? (
        <>
          <button aria-label="Закрыть форму пользователя" className="drawerScrim" onClick={closeDrawer} type="button" />
          <aside className="drawer" aria-label={drawerTitle}>
            <header className="drawerHeader">
              <div>
                <p className="kicker">{editingId ? 'access record' : 'workspace user'}</p>
                <h3>{drawerTitle}</h3>
              </div>
              <button className="iconButton" onClick={closeDrawer} type="button">
                <span aria-hidden="true">×</span>
                <span className="srOnly">Закрыть</span>
              </button>
            </header>

            <div className="drawerBody">
              <label className="field">
                <span>Имя</span>
                <input
                  onChange={(event) => setDraft((currentDraft) => ({ ...currentDraft, name: event.target.value }))}
                  placeholder="Имя пользователя"
                  value={draft.name}
                />
              </label>

              <label className="field">
                <span>Email</span>
                <input
                  onChange={(event) => setDraft((currentDraft) => ({ ...currentDraft, email: event.target.value }))}
                  placeholder="user@example.com"
                  type="email"
                  value={draft.email}
                />
              </label>

              <label className="field">
                <span>Роль</span>
                <select
                  onChange={(event) =>
                    setDraft((currentDraft) => ({ ...currentDraft, role: event.target.value as UserRole }))
                  }
                  value={draft.role}
                >
                  <option>Владелец</option>
                  <option>Участник</option>
                  <option>Наблюдатель</option>
                </select>
              </label>

              <label className="field">
                <span>Статус</span>
                <select
                  onChange={(event) =>
                    setDraft((currentDraft) => ({ ...currentDraft, status: event.target.value as UserStatus }))
                  }
                  value={draft.status}
                >
                  <option>Активен</option>
                  <option>Ожидает</option>
                  <option>Отключен</option>
                </select>
              </label>

            </div>

            <footer className="drawerFooter">
              <button className="ghostButton" onClick={closeDrawer} type="button">
                Отмена
              </button>
              <button className="primaryButton" onClick={saveUser} type="button">
                Сохранить
              </button>
            </footer>
          </aside>
        </>
      ) : null}
    </main>
  );
}

function SurfaceEmptyStatePanel({ state, label }: { state: SurfaceEmptyState; label: string }) {
  return (
    <section className="emptySurface" aria-label={`${label}: structured empty state`}>
      <div className="emptyStateShell">
        <header className="emptyStateHeader">
          <div>
            <p className="kicker">{state.kicker}</p>
            <h2>{state.label}</h2>
          </div>
          {state.status ? <span className="emptyStateStatus">{state.status}</span> : null}
        </header>

        <p className="emptyStateCopy">{state.copy}</p>

        <div className="emptyStateGrid">
          {state.lanes.map((lane) => (
            <article className="emptyStateCard" key={lane.title}>
              <h3>{lane.title}</h3>
              <p>{lane.body}</p>
            </article>
          ))}
        </div>

        {state.footer ? <p className="emptyStateFooter">{state.footer}</p> : null}
      </div>
    </section>
  );
}

function AppearanceSettings({
  activeTheme,
  onThemeChange,
}: {
  activeTheme: ThemeId;
  onThemeChange: (theme: ThemeId) => void;
}) {
  return (
    <div className="appearancePanel">
      <div className="appearanceHeader">
        <div>
          <h3>Оформление</h3>
          <p>Выберите визуальный пресет консоли. Это влияет только на интерфейс, не на delivery logic.</p>
        </div>
        <p className="themeDisclaimer">terminal-inspired visual presets · не связаны с авторами оригинальных схем</p>
      </div>

      <div className="themeGrid">
        {THEMES.map((theme) => (
          <button
            aria-pressed={activeTheme === theme.id}
            className={activeTheme === theme.id ? 'themeCard active' : 'themeCard'}
            key={theme.id}
            onClick={() => onThemeChange(theme.id)}
            type="button"
          >
            <span className="themeCardTop">
              <span>{theme.label}</span>
              <span className="themeSelected">{activeTheme === theme.id ? 'Выбрана' : 'Выбрать'}</span>
            </span>
            <span className="themeSwatches" aria-hidden="true">
              {theme.swatches.map((swatch) => (
                <span className="themeSwatch" key={swatch} style={{ background: swatch }} />
              ))}
            </span>
            <span className="themePreview" aria-hidden="true">
              <span />
              <span />
              <span />
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}

function Brand() {
  return (
    <div className="brand" aria-label="Консоль Goalrail">
      <span className="brandText">GOALRAIL</span>
    </div>
  );
}

export default App;

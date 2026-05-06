import { FormEvent, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  changePassword,
  getContract,
  isAuthClientError,
  login as loginWithPassword,
  logout as logoutSession,
  me as fetchCurrentProfile,
} from './authClient';
import type { AuthClientError, ContractResponse, LoginResponse, MeResponse } from './authClient';
import { isSupportedLocale, updateLocaleQueryParam } from './i18n/locale';
import type { ConsoleLocale } from './i18n/resources';

import './App.css';

type SurfaceId = 'contracts' | 'delivery-readiness' | 'proof';
type ScreenId = 'console' | 'settings-appearance' | 'settings-users';
type ThemeId = 'goalrail-default' | 'catppuccin-mocha' | 'dracula' | 'nord' | 'solarized-dark' | 'gruvbox-dark';
type UserStatus = 'active' | 'pending' | 'disabled';
type UserRole = 'owner' | 'member' | 'observer';
type MembershipRole = 'owner' | 'admin' | 'member' | 'viewer';
type RoleFilter = UserRole | 'all';
type StatusFilter = UserStatus | 'all';
type ContractLoadStatus = 'idle' | 'loading' | 'loaded' | 'not_found' | 'error';
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

const SURFACES: SurfaceId[] = ['contracts', 'delivery-readiness', 'proof'];
const CONSOLE_ROLES: UserRole[] = ['owner', 'member', 'observer'];
const MEMBERSHIP_ROLES: MembershipRole[] = ['owner', 'admin', 'member', 'viewer'];
const USER_STATUSES: UserStatus[] = ['active', 'pending', 'disabled'];

const SURFACE_LANES = {
  contracts: ['intent', 'scope', 'acceptance', 'handoff'],
  'delivery-readiness': ['context', 'constraints', 'acceptance', 'risks'],
  proof: ['scope', 'integrity', 'policy', 'target'],
} as const satisfies Record<SurfaceId, readonly string[]>;

const THEMES: ThemePreset[] = [
  { id: 'goalrail-default', label: 'Goalrail Default', swatches: ['#201f1d', '#2d2b28', '#e8e0d2', '#c783a8', '#92b66f'] },
  { id: 'catppuccin-mocha', label: 'Catppuccin Mocha', swatches: ['#1e1e2e', '#313244', '#cdd6f4', '#cba6f7', '#a6e3a1'] },
  { id: 'dracula', label: 'Dracula', swatches: ['#282a36', '#44475a', '#f8f8f2', '#bd93f9', '#50fa7b'] },
  { id: 'nord', label: 'Nord', swatches: ['#2e3440', '#3b4252', '#eceff4', '#88c0d0', '#a3be8c'] },
  { id: 'solarized-dark', label: 'Solarized Dark', swatches: ['#002b36', '#073642', '#eee8d5', '#268bd2', '#859900'] },
  { id: 'gruvbox-dark', label: 'Gruvbox Dark', swatches: ['#282828', '#3c3836', '#ebdbb2', '#fe8019', '#b8bb26'] },
];

const THEME_STORAGE_KEY = 'goalrail.console.theme';

const INITIAL_USERS: ConsoleUser[] = [
];

const EMPTY_DRAFT: Omit<ConsoleUser, 'id'> = {
  name: '',
  email: '',
  role: 'member',
  status: 'pending',
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
  if (status === 'active') {
    return 'statusActive';
  }

  if (status === 'pending') {
    return 'statusPending';
  }

  return 'statusDisabled';
}

function isThemeId(value: string | null): value is ThemeId {
  return THEMES.some((theme) => theme.id === value);
}

function isMembershipRole(value: string | undefined): value is MembershipRole {
  return MEMBERSHIP_ROLES.includes(value as MembershipRole);
}

function profileConsoleRole(profile: MeResponse): UserRole {
  if (profile.organization_membership.role === 'member') {
    return 'member';
  }

  if (profile.organization_membership.role === 'viewer') {
    return 'observer';
  }

  return 'owner';
}

function profileConsoleStatus(profile: MeResponse): UserStatus {
  if (profile.user.state === 'pending' || profile.organization_membership.state === 'pending') {
    return 'pending';
  }

  if (profile.user.state === 'disabled' || profile.organization_membership.state === 'disabled') {
    return 'disabled';
  }

  return 'active';
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

function operationalErrorMessage(error: AuthClientError, t: (key: string, options?: Record<string, unknown>) => string) {
  return error.status
    ? t('auth.operationalErrorWithStatus', { status: error.status })
    : t('auth.operationalError');
}

function authErrorMessage(error: unknown, t: (key: string, options?: Record<string, unknown>) => string) {
  if (isAuthClientError(error)) {
    const translated = t(`auth.errors.${error.code}`);
    return translated === `auth.errors.${error.code}` ? operationalErrorMessage(error, t) : translated;
  }

  return t('auth.fallbackError');
}

function App() {
  const { i18n, t } = useTranslation();
  const translate = t as unknown as (key: string, options?: Record<string, unknown>) => string;
  const activeLocale = isSupportedLocale(i18n.resolvedLanguage) ? i18n.resolvedLanguage : 'en';
  const [authStatus, setAuthStatus] = useState<AuthStatus>('unauthenticated');
  const [authError, setAuthError] = useState('');
  const [passwordChangeError, setPasswordChangeError] = useState('');
  const [tokens, setTokens] = useState<TokenState | null>(null);
  const [profile, setProfile] = useState<MeResponse | null>(null);
  const [activeSurface, setActiveSurface] = useState<SurfaceId>('contracts');
  const [screen, setScreen] = useState<ScreenId>('console');
  const [activeTheme, setActiveTheme] = useState<ThemeId>(() => readStoredTheme());
  const [users, setUsers] = useState<ConsoleUser[]>(INITIAL_USERS);
  const [contractIdInput, setContractIdInput] = useState('');
  const [contractLoadStatus, setContractLoadStatus] = useState<ContractLoadStatus>('idle');
  const [contractError, setContractError] = useState('');
  const [contract, setContract] = useState<ContractResponse | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState<Omit<ConsoleUser, 'id'>>(EMPTY_DRAFT);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  useEffect(() => {
    document.documentElement.lang = activeLocale;
  }, [activeLocale]);

  const activeLabel = translate(`surfaces.${activeSurface}.label`);
  const drawerTitle = editingId ? translate('users.editUser') : translate('users.addUser');
  const isLoginPending = authStatus === 'logging_in';
  const isPasswordChangePending = authStatus === 'changing_password';
  const isLoggingOut = authStatus === 'logging_out';
  const sessionDisplayName = profile?.user.display_name.trim() || profile?.user.email || translate('session.fallbackUser');
  const sessionEmail = profile?.user.email;
  const sessionRoleValue = profile?.organization_membership.role;
  const sessionRole = isMembershipRole(sessionRoleValue)
    ? translate(`membershipRoles.${sessionRoleValue}`)
    : sessionRoleValue ?? 'member';

  async function handleLanguageChange(locale: ConsoleLocale) {
    await i18n.changeLanguage(locale);
    updateLocaleQueryParam(locale);
  }

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const email = String(form.get('email') ?? '').trim();
    const password = String(form.get('password') ?? '').trim();

    if (!email || !password) {
      setAuthError(translate('auth.missingCredentials'));
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
      setAuthError(authErrorMessage(error, translate));
    }
  }

  async function handlePasswordChange(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const currentPassword = String(form.get('currentPassword') ?? '');
    const newPassword = String(form.get('newPassword') ?? '');

    if (!currentPassword || !newPassword.trim()) {
      setPasswordChangeError(translate('auth.missingPasswordChange'));
      return;
    }

    if (!tokens) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
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
        setAuthError(authErrorMessage(error, translate));
        return;
      }

      setAuthStatus('password_change_required');
      setPasswordChangeError(authErrorMessage(error, translate));
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
    } catch {
      // Logout is best-effort in the UI; local auth cleanup still happens below.
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
    setContractIdInput('');
    setContractLoadStatus('idle');
    setContractError('');
    setContract(null);
  }

  async function handleContractLookup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const contractId = contractIdInput.trim();
    const accessToken = tokens?.accessToken;

    if (!contractId) {
      setContractError(translate('surfaces.contracts.lookupMissing'));
      setContractLoadStatus('error');
      setContract(null);
      return;
    }

    if (!accessToken) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return;
    }

    setContractLoadStatus('loading');
    setContractError('');
    setContract(null);

    try {
      const found = await getContract(accessToken, contractId);
      setContract(found);
      setContractLoadStatus('loaded');
    } catch (error) {
      setContract(null);
      if (isAuthClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(authErrorMessage(error, translate));
        return;
      }
      if (isAuthClientError(error) && error.code === 'not_found') {
        setContractLoadStatus('not_found');
        setContractError(translate('surfaces.contracts.notFound'));
        return;
      }
      setContractLoadStatus('error');
      setContractError(authErrorMessage(error, translate));
    }
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

  const visibleCurrentProfile = useMemo(() => {
    if (!profile) {
      return null;
    }

    const normalizedQuery = searchQuery.trim().toLowerCase();
    const displayName = profile.user.display_name.trim() || profile.user.email || profile.user.id;
    const email = profile.user.email ?? '';
    const matchesQuery =
      !normalizedQuery ||
      displayName.toLowerCase().includes(normalizedQuery) ||
      email.toLowerCase().includes(normalizedQuery);
    const matchesRole = roleFilter === 'all' || profileConsoleRole(profile) === roleFilter;
    const matchesStatus = statusFilter === 'all' || profileConsoleStatus(profile) === statusFilter;

    return matchesQuery && matchesRole && matchesStatus ? profile : null;
  }, [profile, roleFilter, searchQuery, statusFilter]);

  const visibleUserCount = visibleUsers.length + (visibleCurrentProfile ? 1 : 0);

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
            <span className={user.role === 'owner' ? 'pill roleOwner' : 'pill'}>{translate(`roles.${user.role}`)}</span>
          </td>
          <td>
            <span className={`pill ${statusClass(user.status)}`}>{translate(`statuses.${user.status}`)}</span>
          </td>
          <td>
            <div className="userActions">
              <button className="iconButton" onClick={() => openExistingUser(user)} type="button">
                <span aria-hidden="true">✎</span>
                <span className="srOnly">{translate('users.editUserName', { name: user.name })}</span>
              </button>
            </div>
          </td>
        </tr>
      )),
    [translate, visibleUsers]
  );

  if (authStatus === 'unauthenticated' || authStatus === 'logging_in') {
    return (
      <main
        className="loginScreen"
        data-deployment-target={translate('app.deploymentTarget')}
        data-goalrail-theme={activeTheme}
      >
        <div className="loginRails" aria-hidden="true" />
        <form className="loginCard" onSubmit={handleLogin}>
          <Brand label={translate('app.brandLabel')} />

          <label className="field">
            <span>{translate('auth.email')}</span>
            <input autoComplete="email" disabled={isLoginPending} name="email" placeholder="name@example.com" type="email" />
          </label>

          <label className={authError ? 'field fieldError' : 'field'}>
            <span>{translate('auth.password')}</span>
            <input autoComplete="current-password" disabled={isLoginPending} name="password" type="password" />
          </label>

          {authError ? <p className="fieldMessage" role="alert">{authError}</p> : null}

          <button className="primaryButton fullWidth" disabled={isLoginPending} type="submit">
            {isLoginPending ? translate('auth.signingIn') : translate('auth.signIn')}
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
        data-deployment-target={translate('app.deploymentTarget')}
        data-goalrail-theme={activeTheme}
      >
        <div className="loginRails" aria-hidden="true" />
        <form className="loginCard passwordChangeCard" onSubmit={handlePasswordChange} aria-label={translate('auth.passwordChangeForm')}>
          <Brand label={translate('app.brandLabel')} />

          <div className="authStateBlock">
            <p className="authStateLabel">{translate('auth.passwordChangeRequired')}</p>
          </div>

          <label className={passwordChangeError ? 'field fieldError' : 'field'}>
            <span>{translate('auth.currentPassword')}</span>
            <input autoComplete="current-password" disabled={isPasswordChangePending} name="currentPassword" type="password" />
          </label>

          <label className={passwordChangeError ? 'field fieldError' : 'field'}>
            <span>{translate('auth.newPassword')}</span>
            <input autoComplete="new-password" disabled={isPasswordChangePending} name="newPassword" type="password" />
          </label>

          {passwordChangeError ? <p className="fieldMessage" role="alert">{passwordChangeError}</p> : null}

          <button className="primaryButton fullWidth" disabled={isPasswordChangePending} type="submit">
            {isPasswordChangePending ? translate('auth.changingPassword') : translate('auth.changePassword')}
            <span aria-hidden="true">→</span>
          </button>
        </form>
      </main>
    );
  }

  return (
    <main
      className={screen === 'console' && activeSurface === 'contracts' ? 'consoleShell consoleShellContract' : 'consoleShell'}
      data-deployment-target={translate('app.deploymentTarget')}
      data-goalrail-theme={activeTheme}
    >
      <aside className="sidebar" aria-label={translate('nav.sidebar')}>
        <Brand label={translate('app.brandLabel')} />

        <nav className="surfaceNav" aria-label={translate('nav.productSurfaces')}>
          {SURFACES.map((surface) => (
            <button
              aria-current={screen === 'console' && activeSurface === surface ? 'page' : undefined}
              className={screen === 'console' && activeSurface === surface ? 'surfaceButton active' : 'surfaceButton'}
              key={surface}
              onClick={() => {
                setActiveSurface(surface);
                setScreen('console');
              }}
              type="button"
            >
              {translate(`surfaces.${surface}.nav`)}
            </button>
          ))}
        </nav>

        {screen === 'console' && activeSurface === 'contracts' ? (
          <ContractRailPanel
            contract={contract}
            contractError={contractError}
            contractIdInput={contractIdInput}
            loadStatus={contractLoadStatus}
            onContractIdChange={setContractIdInput}
            onLookup={handleContractLookup}
            t={translate}
          />
        ) : null}

        <div className="sidebarSpacer" />

        <section className="sessionPanel" aria-label={translate('session.currentUser')}>
          <div>
            <p className="sessionName">{sessionDisplayName}</p>
            {sessionEmail ? <p className="sessionEmail">{sessionEmail}</p> : null}
            <p className="sessionRole">{translate('session.role', { role: sessionRole })}</p>
          </div>
          <button className="ghostButton logoutButton" disabled={isLoggingOut} onClick={handleLogout} type="button">
            {isLoggingOut ? translate('session.loggingOut') : translate('session.logout')}
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
            <span>{translate('nav.settings')}</span>
          </button>
        </div>
      </aside>

      {screen === 'console' ? (
        activeSurface === 'contracts' ? (
          <ContractSurfacePanel
            contract={contract}
            loadStatus={contractLoadStatus}
            t={translate}
          />
        ) : (
          <SurfaceEmptyStatePanel surface={activeSurface} label={activeLabel} t={translate} />
        )
      ) : (
        <section
          className="settingsSurface"
          aria-label={screen === 'settings-appearance' ? translate('settings.appearanceLabel') : translate('settings.usersLabel')}
        >
          <header className="surfaceHeader">
            <div>
              <p className="kicker">{screen === 'settings-appearance' ? translate('settings.appearanceKicker') : translate('settings.usersKicker')}</p>
              <h2>{translate('nav.settings')}</h2>
            </div>
            <p className="metaText">
              {screen === 'settings-appearance'
                ? translate('settings.presets', { count: THEMES.length })
                : translate('settings.records', { count: visibleUserCount })}
            </p>
          </header>

          <nav className="settingsSectionNav" aria-label={translate('settings.sections')}>
            <button
              aria-current={screen === 'settings-appearance' ? 'page' : undefined}
              className={screen === 'settings-appearance' ? 'sectionButton active' : 'sectionButton'}
              onClick={() => setScreen('settings-appearance')}
              type="button"
            >
              {translate('settings.appearance')}
            </button>
            <button
              aria-current={screen === 'settings-users' ? 'page' : undefined}
              className={screen === 'settings-users' ? 'sectionButton active' : 'sectionButton'}
              onClick={() => setScreen('settings-users')}
              type="button"
            >
              {translate('settings.users')}
            </button>
          </nav>

          <div className="settingsContent">
            {screen === 'settings-appearance' ? (
              <AppearanceSettings
                activeLocale={activeLocale}
                activeTheme={activeTheme}
                onLanguageChange={handleLanguageChange}
                onThemeChange={updateTheme}
                t={translate}
              />
            ) : (
              <>
                <div className="usersHeader">
                  <div>
                    <h3>{translate('users.title')}</h3>
                    <p>{translate('users.manage')}</p>
                  </div>
                </div>

                <div className="usersToolbar">
                  <label className="searchBox">
                    <span aria-hidden="true">⌕</span>
                    <input
                      aria-label={translate('users.searchUsers')}
                      onChange={(event) => setSearchQuery(event.target.value)}
                      placeholder={translate('users.searchPlaceholder')}
                      type="search"
                      value={searchQuery}
                    />
                  </label>
                  <label className="filterBox">
                    <span>{translate('users.role')}</span>
                    <select
                      aria-label={translate('users.filterByRole')}
                      onChange={(event) => setRoleFilter(event.target.value as RoleFilter)}
                      value={roleFilter}
                    >
                      <option value="all">{translate('users.allRoles')}</option>
                      {CONSOLE_ROLES.map((role) => (
                        <option key={role} value={role}>
                          {translate(`roles.${role}`)}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="filterBox">
                    <span>{translate('users.status')}</span>
                    <select
                      aria-label={translate('users.filterByStatus')}
                      onChange={(event) => setStatusFilter(event.target.value as StatusFilter)}
                      value={statusFilter}
                    >
                      <option value="all">{translate('users.allStatuses')}</option>
                      {USER_STATUSES.map((status) => (
                        <option key={status} value={status}>
                          {translate(`statuses.${status}`)}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>

                <div className="usersTableFrame">
                  <table className="usersTable" aria-label={translate('users.table')}>
                    <thead>
                      <tr className="userRow userHead">
                        <th scope="col">{translate('users.name')}</th>
                        <th scope="col">{translate('users.email')}</th>
                        <th scope="col">{translate('users.role')}</th>
                        <th scope="col">{translate('users.status')}</th>
                        <th scope="col" aria-label={translate('users.actions')} />
                      </tr>
                    </thead>
                    <tbody>
                      <CurrentUserRow profile={visibleCurrentProfile} t={translate} />
                      {userRows}
                      {visibleUserCount === 0 ? (
                        <tr>
                          <td className="emptyUsers" colSpan={5}>
                            {translate('users.emptyRealOnly')}
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
          <button aria-label={translate('users.closeForm')} className="drawerScrim" onClick={closeDrawer} type="button" />
          <aside className="drawer" aria-label={drawerTitle}>
            <header className="drawerHeader">
              <div>
                <p className="kicker">{editingId ? translate('users.accessRecord') : translate('users.workspaceUser')}</p>
                <h3>{drawerTitle}</h3>
              </div>
              <button className="iconButton" onClick={closeDrawer} type="button">
                <span aria-hidden="true">×</span>
                <span className="srOnly">{translate('users.close')}</span>
              </button>
            </header>

            <div className="drawerBody">
              <label className="field">
                <span>{translate('users.name')}</span>
                <input
                  onChange={(event) => setDraft((currentDraft) => ({ ...currentDraft, name: event.target.value }))}
                  placeholder={translate('users.userNamePlaceholder')}
                  value={draft.name}
                />
              </label>

              <label className="field">
                <span>{translate('users.email')}</span>
                <input
                  onChange={(event) => setDraft((currentDraft) => ({ ...currentDraft, email: event.target.value }))}
                  placeholder="user@example.com"
                  type="email"
                  value={draft.email}
                />
              </label>

              <label className="field">
                <span>{translate('users.role')}</span>
                <select
                  onChange={(event) =>
                    setDraft((currentDraft) => ({ ...currentDraft, role: event.target.value as UserRole }))
                  }
                  value={draft.role}
                >
                  {CONSOLE_ROLES.map((role) => (
                    <option key={role} value={role}>
                      {translate(`roles.${role}`)}
                    </option>
                  ))}
                </select>
              </label>

              <label className="field">
                <span>{translate('users.status')}</span>
                <select
                  onChange={(event) =>
                    setDraft((currentDraft) => ({ ...currentDraft, status: event.target.value as UserStatus }))
                  }
                  value={draft.status}
                >
                  {USER_STATUSES.map((status) => (
                    <option key={status} value={status}>
                      {translate(`statuses.${status}`)}
                    </option>
                  ))}
                </select>
              </label>
            </div>

            <footer className="drawerFooter">
              <button className="ghostButton" onClick={closeDrawer} type="button">
                {translate('users.cancel')}
              </button>
              <button className="primaryButton" onClick={saveUser} type="button">
                {translate('users.save')}
              </button>
            </footer>
          </aside>
        </>
      ) : null}
    </main>
  );
}

function ContractRailPanel({
  contract,
  contractError,
  contractIdInput,
  loadStatus,
  onContractIdChange,
  onLookup,
  t,
}: {
  contract: ContractResponse | null;
  contractError: string;
  contractIdInput: string;
  loadStatus: ContractLoadStatus;
  onContractIdChange: (value: string) => void;
  onLookup: (event: FormEvent<HTMLFormElement>) => void;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const isLoading = loadStatus === 'loading';
  const stateLabel = contract ? t(`contractStates.${contract.state}`) : t('surfaces.contracts.notSelected');
  const statusTone = contract?.state === 'approved' ? 'pass' : contract?.state === 'ready_for_approval' ? 'amber' : contract ? 'mauve' : 'muted';

  return (
    <section className="contractRailPanel" aria-label={t('surfaces.contracts.ops.surfaceContext')}>
      <div className="opsGroupLabel">{t('surfaces.contracts.ops.surfaceContext')}</div>
      <div className="opsRailSection">
        <h3>{t('surfaces.contracts.nav')}</h3>
        <p>{contract ? t('surfaces.contracts.ops.repoScoped') : t('surfaces.contracts.ops.selectById')}</p>
      </div>

      <form className="opsContractSearch" onSubmit={onLookup}>
        <label>
          <span>{t('surfaces.contracts.lookupLabel')}</span>
          <input
            autoComplete="off"
            name="contractId"
            onChange={(event) => onContractIdChange(event.target.value)}
            placeholder={t('surfaces.contracts.lookupPlaceholder')}
            value={contractIdInput}
          />
        </label>
        <button disabled={isLoading} type="submit">
          {isLoading ? t('surfaces.contracts.loading') : t('surfaces.contracts.lookupAction')}
          <span aria-hidden="true">→</span>
        </button>
      </form>
      {contractError ? <p className="fieldMessage contractMessage" role="alert">{contractError}</p> : null}

      <div className="opsGroupLabel">{t('surfaces.contracts.ops.activeContracts')}</div>
      <div className="opsFilterChips">
        <span>{t('surfaces.contracts.ops.active')}</span>
        <span>{t('surfaces.contracts.ops.executing')}</span>
        <span>{t('surfaces.contracts.ops.approval')}</span>
      </div>
      <div className="opsContractList" aria-label={t('surfaces.contracts.ops.activeContracts')}>
        {contract ? (
          <div className="opsContractRow active">
            <div>
              <span className="opsMono">{contract.id}</span>
              <b>{t('surfaces.contracts.currentRecord')}</b>
            </div>
            <span className={`opsPill ${statusTone}`}>{stateLabel}</span>
            <p>{t('surfaces.contracts.ops.loadedSummary')}</p>
            <div className="opsRowMeta">
              <span>{contract.repo_binding_id}</span>
              <span>{formatDateTime(contract.updated_at)}</span>
            </div>
          </div>
        ) : (
          <div className="opsRailEmpty">{t('surfaces.contracts.ops.noActiveContract')}</div>
        )}
      </div>
    </section>
  );
}

function ContractSurfacePanel({
  contract,
  loadStatus,
  t,
}: {
  contract: ContractResponse | null;
  loadStatus: ContractLoadStatus;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const stateLabel = contract ? t(`contractStates.${contract.state}`) : t('surfaces.contracts.notSelected');
  const sectionIds = ['intent', 'scope', 'acceptance', 'proof'] as const;
  const stageIds = ['goalIntake', 'clarification', 'workingContract', 'workItems', 'executionEvidence', 'verification', 'proof', 'approval'] as const;
  const stateStageIndex = contract?.state === 'approved'
    ? 7
    : contract?.state === 'ready_for_approval'
      ? 6
      : contract?.state === 'draft'
        ? 2
        : contract?.state === 'seeded'
          ? 2
          : -1;
  const contractPercent = contract ? Math.max(18, Math.round(((stateStageIndex + 1) / stageIds.length) * 100)) : 0;
  const executionPercent = contract?.state === 'approved' ? 72 : contract ? 18 : 0;
  const proofPercent = contract?.state === 'approved' ? 100 : contract?.state === 'ready_for_approval' ? 64 : 0;
  const statusTone = contract?.state === 'approved' ? 'pass' : contract?.state === 'ready_for_approval' ? 'amber' : contract ? 'mauve' : 'muted';
  const activeStageId = stageIds.find((_, index) => index === stateStageIndex);

  return (
    <section className="contractSurface opsContractSurface" aria-label={t('surfaces.contracts.ariaLabel')}>
      <div className="opsContractTopbar">
        <div className="opsMeters" aria-label={t('surfaces.contracts.ops.meters')}>
          <ContractMeter tone="amber" label={t('surfaces.contracts.ops.contractMeter')} value={stateLabel} percent={contractPercent} />
          <ContractMeter tone="mauve" label={t('surfaces.contracts.ops.executionMeter')} value={contract ? t('surfaces.contracts.ops.executionQueued') : t('surfaces.contracts.ops.queued')} percent={executionPercent} />
          <ContractMeter tone="pass" label={t('surfaces.contracts.ops.proofMeter')} value={contract?.state === 'approved' ? t('surfaces.contracts.ops.proofReady') : t('surfaces.contracts.ops.queued')} percent={proofPercent} />
        </div>

        <div className="opsTopbarState">
          <div className="opsStateChip">
            <span>{t('surfaces.contracts.ops.surface')}</span>
            <b>{t('surfaces.contracts.nav')}</b>
          </div>
          <div className="opsStateChip">
            <span>{t('surfaces.contracts.ops.status')}</span>
            <b>{stateLabel}</b>
          </div>
        </div>
      </div>

      <div className="opsContractLayout">
        <div className="opsContractCanvas">
          <section className="opsSpine">
            <div className="opsPanelHead">
              <div>
                <div className="opsPanelKicker">{contract ? t('surfaces.contracts.ops.contractLoaded') : t('surfaces.contracts.ops.contractNotSelected')}</div>
                <div className="opsPanelId">{contract ? contract.id : t('surfaces.contracts.emptyValue')}</div>
              </div>
              <div className="opsTags">
                <span className="opsTag mauve">{contract?.repo_binding_id ?? t('surfaces.contracts.ops.noRepo')}</span>
                <span className={`opsTag ${statusTone}`}>{stateLabel}</span>
              </div>
            </div>

            <div className="opsStageRail" aria-label={t('surfaces.contracts.lifecycle')}>
              {stageIds.map((stage, index) => {
                const stageClass = stateStageIndex < 0 ? 'queued' : index < stateStageIndex ? 'done' : index === stateStageIndex ? 'active' : 'queued';
                return (
                  <div className={`opsStage ${stageClass}`} key={stage}>
                    <span className="opsStageNode" />
                    <span className="opsStageConnector" />
                    <b>{t(`surfaces.contracts.ops.stages.${stage}`)}</b>
                    <small>{t(`surfaces.contracts.ops.stageStates.${stageClass}`)}</small>
                  </div>
                );
              })}
            </div>

            <div className="opsActiveSummary">
              <span>{t('surfaces.contracts.ops.activeStage')}</span>
              <b>{activeStageId ? t(`surfaces.contracts.ops.stages.${activeStageId}`) : t('surfaces.contracts.ops.none')}</b>
              <small>{contract ? `${t('surfaces.contracts.fields.goalId')} ${contract.goal_id}` : t('surfaces.contracts.emptyBody')}</small>
            </div>
          </section>

          <section className="opsObject">
            <div className="opsPanelHead">
              <div>
                <div className="opsPanelKicker">{t('surfaces.contracts.ops.selectedContract')}</div>
                <h2>{contract ? contract.id : t('surfaces.contracts.notSelected')}</h2>
              </div>
              <div className="opsTags">
                <span className="opsTag">{t('surfaces.contracts.ops.liveConsole')}</span>
                <span className={`opsTag ${statusTone}`}>{stateLabel}</span>
              </div>
            </div>

            <div className="opsObjectBody">
              <div className="opsSectionLead">
                <span>{t('surfaces.contracts.sectionsTitle')}</span>
                <b>{contract ? t('surfaces.contracts.calloutLoaded') : loadStatus === 'not_found' ? t('surfaces.contracts.notFoundBody') : t('surfaces.contracts.emptyBody')}</b>
              </div>
              <div className="opsDetailGrid">
                {sectionIds.map((section) => (
                  <article className="opsDetailBlock" key={section}>
                    <span>{t(`surfaces.contracts.sections.${section}.title`)}</span>
                    <h4>{t(`surfaces.contracts.sections.${section}.title`)}</h4>
                    <p>{contract ? t('surfaces.contracts.sectionPending') : t('surfaces.contracts.sectionEmpty')}</p>
                  </article>
                ))}
              </div>
            </div>
          </section>
        </div>

        <aside className="opsContractSide">
          <section className="opsSideCard">
            <div className="opsPanelHead compact">
              <div>
                <div className="opsPanelKicker">{t('surfaces.contracts.ops.projectContext')}</div>
                <div className="opsPanelId">{contract?.repo_binding_id ?? t('surfaces.contracts.emptyValue')}</div>
              </div>
            </div>
            <dl className="opsKeyGrid">
              <FieldRow label={t('surfaces.contracts.fields.repoBindingId')} value={contract?.repo_binding_id ?? t('surfaces.contracts.emptyValue')} />
              <FieldRow label={t('surfaces.contracts.fields.goalId')} value={contract?.goal_id ?? t('surfaces.contracts.emptyValue')} />
              <FieldRow label={t('surfaces.contracts.fields.seedId')} value={contract?.current_seed_id ?? t('surfaces.contracts.emptyValue')} />
              <FieldRow label={t('surfaces.contracts.fields.draftId')} value={contract?.current_draft_id ?? t('surfaces.contracts.emptyValue')} />
            </dl>
            <div className="opsSelectionStrip">
              <span>{contract?.id ?? t('surfaces.contracts.notSelected')}</span>
              <span>{stateLabel}</span>
              <span>{contract ? t('surfaces.contracts.ops.liveData') : t('surfaces.contracts.ops.noSelection')}</span>
            </div>
          </section>

          <section className="opsSideCard muted">
            <div className="opsPanelHead compact">
              <div>
                <div className="opsPanelKicker">{t('surfaces.contracts.ops.ambiguityInspector')}</div>
                <div className="opsPanelId">{t('surfaces.contracts.ops.inputs')}</div>
              </div>
            </div>
            <div className="opsInspectorList">
              {sectionIds.map((section) => (
                <div className="opsInspectorRow" key={section}>
                  <b>{t(`surfaces.contracts.sections.${section}.title`)}</b>
                  <span>{contract ? t('surfaces.contracts.sectionPending') : t('surfaces.contracts.sectionEmpty')}</span>
                  <small>{contract ? t('surfaces.contracts.ops.pending') : t('surfaces.contracts.ops.empty')}</small>
                </div>
              ))}
            </div>
          </section>
        </aside>

        <section className="opsBottomPanel">
          <article className="opsSideCard">
            <div className="opsPanelHead compact">
              <div>
                <div className="opsPanelKicker">{t('surfaces.contracts.ops.workspaceActivity')}</div>
                <div className="opsPanelId">{t('surfaces.contracts.ops.contractEventsOnly')}</div>
              </div>
            </div>
            <div className="opsActivityRow">
              <span>{contract ? formatDateTime(contract.updated_at) : t('surfaces.contracts.emptyValue')}</span>
              <div>
                <b>{contract ? t('surfaces.contracts.ops.contractLoaded') : t('surfaces.contracts.ops.waitingForContract')}</b>
                <p>{contract ? contract.id : t('surfaces.contracts.emptyBody')}</p>
              </div>
              <small>{contract ? t('surfaces.contracts.ops.event') : t('surfaces.contracts.ops.idle')}</small>
            </div>
          </article>

          <article className="opsSideCard">
            <div className="opsPanelHead compact">
              <div>
                <div className="opsPanelKicker">{t('surfaces.contracts.ops.stageControls')}</div>
                <div className="opsPanelId">{t('surfaces.contracts.ops.liveConsoleOnly')}</div>
              </div>
            </div>
            <p className="opsControlCopy">{contract ? t('surfaces.contracts.ops.stageControlLoaded') : t('surfaces.contracts.ops.stageControlEmpty')}</p>
          </article>
        </section>
      </div>
    </section>
  );
}

function ContractMeter({ label, percent, tone, value }: { label: string; percent: number; tone: 'amber' | 'mauve' | 'pass'; value: string }) {
  return (
    <div className={`opsMeter ${tone}`}>
      <div>
        <span>{label}</span>
        <b>{value}</b>
      </div>
      <i><span style={{ width: `${percent}%` }} /></i>
    </div>
  );
}

function FieldRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

function CurrentUserRow({
  profile,
  t,
}: {
  profile: MeResponse | null;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  if (!profile) {
    return null;
  }

  const displayName = profile.user.display_name.trim() || profile.user.email || profile.user.id;
  const consoleRole = profileConsoleRole(profile);
  const status = profileConsoleStatus(profile);
  const role = isMembershipRole(profile.organization_membership.role)
    ? t(`membershipRoles.${profile.organization_membership.role}`)
    : profile.organization_membership.role;

  return (
    <tr className="userRow">
      <td>
        <div className="userName">
          <span className="avatar" aria-hidden="true">
            {initials(displayName)}
          </span>
          <span>{displayName}</span>
        </div>
      </td>
      <td className="userEmail">{profile.user.email ?? t('users.emptyValue')}</td>
      <td>
        <span className={consoleRole === 'owner' ? 'pill roleOwner' : 'pill'}>{role}</span>
      </td>
      <td>
        <span className={`pill ${statusClass(status)}`}>{t(`statuses.${status}`)}</span>
      </td>
      <td />
    </tr>
  );
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toISOString();
}

function SurfaceEmptyStatePanel({
  label,
  surface,
  t,
}: {
  label: string;
  surface: SurfaceId;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const status = t(`surfaces.${surface}.status`);
  const footer = t(`surfaces.${surface}.footer`);

  return (
    <section className="emptySurface" aria-label={`${label}: structured empty state`}>
      <div className="emptyStateShell">
        <header className="emptyStateHeader">
          <div>
            <p className="kicker">{t(`surfaces.${surface}.kicker`)}</p>
            <h2>{t(`surfaces.${surface}.label`)}</h2>
          </div>
          {status ? <span className="emptyStateStatus">{status}</span> : null}
        </header>

        <p className="emptyStateCopy">{t(`surfaces.${surface}.copy`)}</p>

        <div className="emptyStateGrid">
          {SURFACE_LANES[surface].map((lane) => (
            <article className="emptyStateCard" key={lane}>
              <h3>{t(`surfaces.${surface}.lanes.${lane}.title`)}</h3>
              <p>{t(`surfaces.${surface}.lanes.${lane}.body`)}</p>
            </article>
          ))}
        </div>

        {footer ? <p className="emptyStateFooter">{footer}</p> : null}
      </div>
    </section>
  );
}

function AppearanceSettings({
  activeLocale,
  activeTheme,
  onLanguageChange,
  onThemeChange,
  t,
}: {
  activeLocale: ConsoleLocale;
  activeTheme: ThemeId;
  onLanguageChange: (locale: ConsoleLocale) => void;
  onThemeChange: (theme: ThemeId) => void;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  return (
    <div className="appearancePanel">
      <div className="appearanceHeader">
        <div>
          <h3>{t('settings.appearance')}</h3>
          <p>{t('settings.visualPresetCopy')}</p>
        </div>
        <p className="themeDisclaimer">{t('settings.themeDisclaimer')}</p>
      </div>

      <div className="languageSwitcher" aria-label={t('settings.language')}>
        <button
          aria-pressed={activeLocale === 'en'}
          className={activeLocale === 'en' ? 'languageButton active' : 'languageButton'}
          onClick={() => onLanguageChange('en')}
          type="button"
        >
          {t('settings.english')}
        </button>
        <button
          aria-pressed={activeLocale === 'ru'}
          className={activeLocale === 'ru' ? 'languageButton active' : 'languageButton'}
          onClick={() => onLanguageChange('ru')}
          type="button"
        >
          {t('settings.russian')}
        </button>
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
              <span className="themeSelected">{activeTheme === theme.id ? t('settings.selected') : t('settings.select')}</span>
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

function Brand({ label }: { label: string }) {
  return (
    <div className="brand" aria-label={label}>
      <span className="brandText">GOALRAIL</span>
    </div>
  );
}

export default App;

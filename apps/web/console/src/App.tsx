import { FormEvent, useEffect, useMemo, useRef, useState } from 'react';
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
import {
  createOrganizationUser,
  isUsersClientError,
  listOrganizationUsers,
  patchOrganizationUser,
  resetOrganizationUserTemporaryPassword,
} from './usersClient';
import {
  getOrganizationRepositoryContext,
  isRepositoryContextClientError,
} from './repositoryContextClient';
import type { OrganizationUserRecord, OrganizationUserRole, OrganizationUserState } from './usersClient';
import type { OrganizationRepositoryContextResponse, RepositoryContextRecord } from './repositoryContextClient';
import StartPage from './StartPage';

import './App.css';

type SurfaceId = 'contracts' | 'delivery-readiness' | 'proof';
type ScreenId = 'console' | 'settings-appearance' | 'settings-users' | 'settings-repository';
type ThemeId = 'goalrail-default' | 'catppuccin-mocha' | 'dracula' | 'nord' | 'solarized-dark' | 'gruvbox-dark';
type MembershipRole = 'owner' | 'admin' | 'member' | 'viewer';
type RoleFilter = OrganizationUserRole | 'all';
type StatusFilter = OrganizationUserState | 'all';
type UsersLoadStatus = 'idle' | 'loading' | 'loaded' | 'error';
type RepositoryContextLoadStatus = 'idle' | 'loading' | 'loaded' | 'error';
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

interface UserDraft {
  name: string;
  email: string;
  role: OrganizationUserRole;
  state: OrganizationUserState;
}

interface OneTimeTemporaryPassword {
  email: string;
  password: string;
}

interface ThemePreset {
  id: ThemeId;
  label: string;
  swatches: string[];
}

const SURFACES: SurfaceId[] = ['contracts', 'delivery-readiness', 'proof'];
const CONSOLE_ROLES: OrganizationUserRole[] = ['owner', 'admin', 'member', 'viewer'];
const MEMBERSHIP_ROLES: MembershipRole[] = ['owner', 'admin', 'member', 'viewer'];
const USER_STATUSES: OrganizationUserState[] = ['active', 'inactive'];

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

const EMPTY_DRAFT: UserDraft = {
  name: '',
  email: '',
  role: 'member',
  state: 'active',
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

function statusClass(status: OrganizationUserState) {
  if (status === 'active') {
    return 'statusActive';
  }

  return 'statusInactive';
}

function displayUserState(record: OrganizationUserRecord): OrganizationUserState {
  if (record.user.state === 'inactive' || record.organization_membership.state === 'inactive') {
    return 'inactive';
  }

  return 'active';
}

function userRecordWithoutSecret(record: OrganizationUserRecord): OrganizationUserRecord {
  return {
    user: record.user,
    organization_membership: record.organization_membership,
    credential: record.credential,
  };
}

function isOwnerSelfActionBlocked(profile: MeResponse | null, record: OrganizationUserRecord | null, draft: UserDraft) {
  if (!profile || !record || profile.user.id !== record.user.id || record.organization_membership.role !== 'owner') {
    return null;
  }
  if (draft.role !== 'owner') {
    return 'users.errors.selfOwnerDowngradeBlocked';
  }
  if (draft.state === 'inactive') {
    return 'users.errors.selfDeactivationBlocked';
  }
  return null;
}

function isThemeId(value: string | null): value is ThemeId {
  return THEMES.some((theme) => theme.id === value);
}

function isMembershipRole(value: string | undefined): value is MembershipRole {
  return MEMBERSHIP_ROLES.includes(value as MembershipRole);
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

function usersErrorMessage(error: unknown, t: (key: string, options?: Record<string, unknown>) => string) {
  if (isUsersClientError(error)) {
    const translated = t(`users.errors.${error.code}`);
    if (translated !== `users.errors.${error.code}`) {
      return translated;
    }

    return error.status ? t('users.errors.genericWithStatus', { status: error.status }) : t('users.errors.generic');
  }

  return t('users.errors.generic');
}

function repositoryContextErrorMessage(error: unknown, t: (key: string, options?: Record<string, unknown>) => string) {
  if (isRepositoryContextClientError(error)) {
    const translated = t(`repository.errors.${error.code}`);
    if (translated !== `repository.errors.${error.code}`) {
      return translated;
    }

    return error.status ? t('repository.errors.genericWithStatus', { status: error.status }) : t('repository.errors.generic');
  }

  return t('repository.errors.generic');
}

function normalizedPathname() {
  if (typeof window === 'undefined') {
    return '';
  }

  return window.location.pathname.replace(/\/+$/, '') || '/';
}

function isStartRoute() {
  return normalizedPathname() === '/start';
}

function isRootRoute() {
  return normalizedPathname() === '/';
}

function RootStartRedirect() {
  useEffect(() => {
    if (normalizedPathname() === '/') {
      window.history.replaceState(window.history.state, '', '/start');
    }
  }, []);

  return <StartPage />;
}

function App() {
  if (isRootRoute()) {
    return <RootStartRedirect />;
  }

  return isStartRoute() ? <StartPage /> : <ConsoleApp />;
}

function ConsoleApp() {
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
  const [users, setUsers] = useState<OrganizationUserRecord[]>([]);
  const [usersLoadStatus, setUsersLoadStatus] = useState<UsersLoadStatus>('idle');
  const [usersError, setUsersError] = useState('');
  const [repositoryContext, setRepositoryContext] = useState<OrganizationRepositoryContextResponse | null>(null);
  const [repositoryContextLoadStatus, setRepositoryContextLoadStatus] = useState<RepositoryContextLoadStatus>('idle');
  const [repositoryContextError, setRepositoryContextError] = useState('');
  const [formError, setFormError] = useState('');
  const [temporaryPassword, setTemporaryPassword] = useState<OneTimeTemporaryPassword | null>(null);
  const [contractIdInput, setContractIdInput] = useState('');
  const [contractLoadStatus, setContractLoadStatus] = useState<ContractLoadStatus>('idle');
  const [contractError, setContractError] = useState('');
  const [contract, setContract] = useState<ContractResponse | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState<UserDraft>(EMPTY_DRAFT);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [resetTarget, setResetTarget] = useState<OrganizationUserRecord | null>(null);
  const [resetError, setResetError] = useState('');
  const [resettingUserId, setResettingUserId] = useState<string | null>(null);
  const contractLookupSequence = useRef(0);
  const usersLoadSequence = useRef(0);
  const repositoryContextLoadSequence = useRef(0);
  const authSessionSequence = useRef(0);

  useEffect(() => {
    document.documentElement.lang = activeLocale;
  }, [activeLocale]);

  useEffect(() => {
    if (screen !== 'settings-users' || authStatus !== 'authenticated') {
      return;
    }

    void loadUsers();
  }, [authStatus, profile?.organization_membership.organization_id, screen, tokens?.accessToken]);

  useEffect(() => {
    if (screen !== 'settings-repository' || authStatus !== 'authenticated') {
      return;
    }

    void loadRepositoryContext();
  }, [authStatus, profile?.organization_membership.organization_id, screen, tokens?.accessToken]);

  useEffect(() => {
    if (screen !== 'settings-users') {
      setTemporaryPassword(null);
    }
  }, [screen]);

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
  const settingsKicker =
    screen === 'settings-appearance'
      ? translate('settings.appearanceKicker')
      : screen === 'settings-users'
        ? translate('settings.usersKicker')
        : translate('settings.repositoryKicker');
  const settingsLabel =
    screen === 'settings-appearance'
      ? translate('settings.appearanceLabel')
      : screen === 'settings-users'
        ? translate('settings.usersLabel')
        : translate('settings.repositoryLabel');
  const editingRecord = editingId ? users.find((record) => record.user.id === editingId) ?? null : null;
  const isEditingSelf = Boolean(editingRecord && profile?.user.id === editingRecord.user.id);
  const isEditingOtherOwner = Boolean(editingRecord && !isEditingSelf && editingRecord.organization_membership.role === 'owner');
  const isResettingTemporaryPassword = resettingUserId !== null;

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
    authSessionSequence.current += 1;
    contractLookupSequence.current += 1;
    usersLoadSequence.current += 1;
    repositoryContextLoadSequence.current += 1;
    setTokens(null);
    setProfile(null);
    setAuthStatus('unauthenticated');
    setAuthError('');
    setPasswordChangeError('');
    setScreen('console');
    setIsDrawerOpen(false);
    setUsers([]);
    setUsersLoadStatus('idle');
    setUsersError('');
    setRepositoryContext(null);
    setRepositoryContextLoadStatus('idle');
    setRepositoryContextError('');
    setFormError('');
    setTemporaryPassword(null);
    setResetTarget(null);
    setResetError('');
    setResettingUserId(null);
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
      contractLookupSequence.current += 1;
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

    const lookupSequence = contractLookupSequence.current + 1;
    contractLookupSequence.current = lookupSequence;
    setContractLoadStatus('loading');
    setContractError('');
    setContract(null);

    try {
      const found = await getContract(accessToken, contractId);
      if (contractLookupSequence.current !== lookupSequence) {
        return;
      }
      setContract(found);
      setContractLoadStatus('loaded');
    } catch (error) {
      if (contractLookupSequence.current !== lookupSequence) {
        return;
      }
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

  async function loadUsers() {
    const accessToken = tokens?.accessToken;
    const organizationId = profile?.organization_membership.organization_id;

    if (!accessToken || !organizationId) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return;
    }

    setUsersLoadStatus('loading');
    setUsersError('');
    const loadSequence = usersLoadSequence.current + 1;
    usersLoadSequence.current = loadSequence;

    try {
      const result = await listOrganizationUsers({ accessToken, organizationId });
      if (usersLoadSequence.current !== loadSequence) {
        return;
      }
      setUsers(result.users);
      setUsersLoadStatus('loaded');
    } catch (error) {
      if (usersLoadSequence.current !== loadSequence) {
        return;
      }
      if (isUsersClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(translate('auth.invalidSession'));
        return;
      }

      setUsersLoadStatus('error');
      setUsersError(usersErrorMessage(error, translate));
    }
  }

  async function loadRepositoryContext() {
    const accessToken = tokens?.accessToken;
    const organizationId = profile?.organization_membership.organization_id;

    if (!accessToken || !organizationId) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return;
    }

    setRepositoryContextLoadStatus('loading');
    setRepositoryContextError('');
    const loadSequence = repositoryContextLoadSequence.current + 1;
    repositoryContextLoadSequence.current = loadSequence;

    try {
      const result = await getOrganizationRepositoryContext({ accessToken, organizationId });
      if (repositoryContextLoadSequence.current !== loadSequence) {
        return;
      }
      setRepositoryContext(result);
      setRepositoryContextLoadStatus('loaded');
    } catch (error) {
      if (repositoryContextLoadSequence.current !== loadSequence) {
        return;
      }
      if (isRepositoryContextClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(translate('auth.invalidSession'));
        return;
      }

      setRepositoryContext(null);
      setRepositoryContextLoadStatus('error');
      setRepositoryContextError(repositoryContextErrorMessage(error, translate));
    }
  }

  function openNewUser() {
    setEditingId(null);
    setDraft({ ...EMPTY_DRAFT });
    setFormError('');
    setTemporaryPassword(null);
    setResetTarget(null);
    setResetError('');
    setIsDrawerOpen(true);
  }

  function openExistingUser(record: OrganizationUserRecord) {
    setEditingId(record.user.id);
    setDraft({
      name: record.user.display_name,
      email: record.user.email ?? '',
      role: record.organization_membership.role,
      state: displayUserState(record),
    });
    setFormError('');
    setTemporaryPassword(null);
    setResetTarget(null);
    setResetError('');
    setIsDrawerOpen(true);
  }

  function closeDrawer() {
    if (isResettingTemporaryPassword) {
      return;
    }
    setIsDrawerOpen(false);
    setResetTarget(null);
    setResetError('');
  }

  async function saveUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextDraft = {
      ...draft,
      name: draft.name.trim(),
      email: draft.email.trim(),
    };

    if (!nextDraft.name || (!editingId && !nextDraft.email)) {
      setFormError(translate('users.validation.nameAndEmail'));
      return;
    }
    const blockedSelfAction = isOwnerSelfActionBlocked(profile, editingRecord, nextDraft);
    if (blockedSelfAction) {
      setFormError(translate(blockedSelfAction));
      return;
    }

    const accessToken = tokens?.accessToken;
    const organizationId = profile?.organization_membership.organization_id;
    if (!accessToken || !organizationId) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return;
    }

    setFormError('');
    const mutationSessionSequence = authSessionSequence.current;

    try {
      if (editingId) {
        const result = await patchOrganizationUser({
          accessToken,
          organizationId,
          userId: editingId,
          displayName: nextDraft.name,
          role: nextDraft.role,
          state: nextDraft.state,
        });
        if (authSessionSequence.current !== mutationSessionSequence) {
          return;
        }
        const userRecord = userRecordWithoutSecret(result);
        usersLoadSequence.current += 1;
        setUsers((currentUsers) => currentUsers.map((record) => (record.user.id === editingId ? userRecord : record)));
      } else {
        const result = await createOrganizationUser({
          accessToken,
          organizationId,
          email: nextDraft.email,
          displayName: nextDraft.name,
          role: nextDraft.role,
        });
        if (authSessionSequence.current !== mutationSessionSequence) {
          return;
        }
        const userRecord = userRecordWithoutSecret(result);
        usersLoadSequence.current += 1;
        setUsers((currentUsers) => [...currentUsers.filter((record) => record.user.id !== result.user.id), userRecord]);
        if (result.temporary_password) {
          setTemporaryPassword({
            email: result.user.email ?? nextDraft.email,
            password: result.temporary_password,
          });
        } else {
          setTemporaryPassword(null);
        }
      }

      setUsersLoadStatus('loaded');
      setUsersError('');
      setIsDrawerOpen(false);
    } catch (error) {
      if (authSessionSequence.current !== mutationSessionSequence) {
        return;
      }
      if (isUsersClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(translate('auth.invalidSession'));
        return;
      }

      setFormError(usersErrorMessage(error, translate));
    }
  }

  function requestTemporaryPasswordReset(record: OrganizationUserRecord) {
    if (profile?.user.id === record.user.id) {
      setResetTarget(null);
      setFormError(translate('users.errors.selfResetBlocked'));
      return;
    }
    setResetTarget(record);
    setResetError('');
  }

  function cancelTemporaryPasswordReset() {
    if (isResettingTemporaryPassword) {
      return;
    }
    setResetTarget(null);
    setResetError('');
  }

  async function confirmTemporaryPasswordReset() {
    const target = resetTarget;
    const accessToken = tokens?.accessToken;
    const organizationId = profile?.organization_membership.organization_id;
    if (!target) {
      return;
    }
    if (profile?.user.id === target.user.id) {
      setResetTarget(null);
      setResetError('');
      setFormError(translate('users.errors.selfResetBlocked'));
      return;
    }
    if (!accessToken || !organizationId) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return;
    }

    setResetError('');
    setResettingUserId(target.user.id);
    const mutationSessionSequence = authSessionSequence.current;

    try {
      const result = await resetOrganizationUserTemporaryPassword({
        accessToken,
        organizationId,
        userId: target.user.id,
      });
      if (authSessionSequence.current !== mutationSessionSequence) {
        return;
      }
      const userRecord = userRecordWithoutSecret(result);
      usersLoadSequence.current += 1;
      setUsers((currentUsers) => currentUsers.map((record) => (record.user.id === target.user.id ? userRecord : record)));
      setTemporaryPassword({
        email: result.user.email ?? target.user.email ?? target.user.id,
        password: result.temporary_password,
      });
      setUsersLoadStatus('loaded');
      setUsersError('');
      setResetTarget(null);
      setResetError('');
      setIsDrawerOpen(false);
    } catch (error) {
      if (authSessionSequence.current !== mutationSessionSequence) {
        return;
      }
      if (isUsersClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(translate('auth.invalidSession'));
        return;
      }

      setResetError(usersErrorMessage(error, translate));
    } finally {
      if (authSessionSequence.current === mutationSessionSequence) {
        setResettingUserId(null);
      }
    }
  }

  function updateTheme(themeId: ThemeId) {
    setActiveTheme(themeId);
    persistTheme(themeId);
  }

  const visibleUsers = useMemo(() => {
    const normalizedQuery = searchQuery.trim().toLowerCase();

    return users.filter((record) => {
      const displayName = record.user.display_name.trim() || record.user.email || record.user.id;
      const email = record.user.email ?? '';
      const state = displayUserState(record);
      const matchesQuery =
        !normalizedQuery ||
        displayName.toLowerCase().includes(normalizedQuery) ||
        email.toLowerCase().includes(normalizedQuery);
      const matchesRole = roleFilter === 'all' || record.organization_membership.role === roleFilter;
      const matchesStatus = statusFilter === 'all' || state === statusFilter;

      return matchesQuery && matchesRole && matchesStatus;
    });
  }, [roleFilter, searchQuery, statusFilter, users]);

  const visibleUserCount = visibleUsers.length;
  const settingsMeta =
    screen === 'settings-appearance'
      ? translate('settings.presets', { count: THEMES.length })
      : screen === 'settings-users'
        ? translate('settings.records', { count: visibleUserCount })
        : translate('settings.contexts', { count: repositoryContext?.contexts.length ?? 0 });

  const userRows = useMemo(
    () =>
      visibleUsers.map((record) => {
        const displayName = record.user.display_name.trim() || record.user.email || record.user.id;
        const email = record.user.email ?? translate('users.emptyValue');
        const state = displayUserState(record);
        const role = record.organization_membership.role;
        const credential = record.credential.must_change_password
          ? translate('users.credentialMustChange')
          : translate('users.credentialReady');

        return (
          <tr className="userRow" key={record.user.id}>
            <td>
              <div className="userName">
                <span className="avatar" aria-hidden="true">
                  {initials(displayName)}
                </span>
                <span>{displayName}</span>
              </div>
            </td>
            <td className="userEmail">{email}</td>
            <td>
              <span className={role === 'owner' ? 'pill roleOwner' : 'pill'}>{translate(`roles.${role}`)}</span>
            </td>
            <td>
              <span className={`pill ${statusClass(state)}`}>{translate(`statuses.${state}`)}</span>
            </td>
            <td>
              <span className={record.credential.must_change_password ? 'pill credentialWarning' : 'pill credentialReady'}>
                {credential}
              </span>
            </td>
            <td>
              <div className="userActions">
                <button className="iconButton" onClick={() => openExistingUser(record)} type="button">
                  <span aria-hidden="true">✎</span>
                  <span className="srOnly">{translate('users.editUserName', { name: displayName })}</span>
                </button>
              </div>
            </td>
          </tr>
        );
      }),
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
          aria-label={settingsLabel}
        >
          <header className="surfaceHeader">
            <div>
              <p className="kicker">{settingsKicker}</p>
              <h2>{translate('nav.settings')}</h2>
            </div>
            <p className="metaText">{settingsMeta}</p>
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
            <button
              aria-current={screen === 'settings-repository' ? 'page' : undefined}
              className={screen === 'settings-repository' ? 'sectionButton active' : 'sectionButton'}
              onClick={() => setScreen('settings-repository')}
              type="button"
            >
              {translate('settings.repository')}
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
            ) : screen === 'settings-users' ? (
              <>
                <div className="usersHeader">
                  <div>
                    <h3>{translate('users.title')}</h3>
                    <p>{translate('users.manage')}</p>
                  </div>
                  <button className="primaryButton" onClick={openNewUser} type="button">
                    {translate('users.add')}
                    <span aria-hidden="true">＋</span>
                  </button>
                </div>

                {temporaryPassword ? (
                  <section className="temporaryPasswordPanel" aria-label={translate('users.temporaryPasswordLabel')}>
                    <div>
                      <h4>{translate('users.temporaryPasswordTitle')}</h4>
                      <p>{translate('users.temporaryPasswordCopy', { email: temporaryPassword.email })}</p>
                    </div>
                    <code>{temporaryPassword.password}</code>
                    <button className="ghostButton" onClick={() => setTemporaryPassword(null)} type="button">
                      {translate('users.dismissTemporaryPassword')}
                    </button>
                  </section>
                ) : null}

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

                {usersLoadStatus === 'loading' ? (
                  <p className="usersNotice" role="status">{translate('users.loading')}</p>
                ) : null}
                {usersLoadStatus === 'error' ? (
                  <div className="usersNotice usersNoticeError" role="alert">
                    <p>{usersError}</p>
                    <button className="ghostButton" onClick={loadUsers} type="button">
                      {translate('users.retry')}
                    </button>
                  </div>
                ) : null}

                <div className="usersTableFrame">
                  <table className="usersTable" aria-label={translate('users.table')}>
                    <thead>
                      <tr className="userRow userHead">
                        <th scope="col">{translate('users.name')}</th>
                        <th scope="col">{translate('users.email')}</th>
                        <th scope="col">{translate('users.role')}</th>
                        <th scope="col">{translate('users.status')}</th>
                        <th scope="col">{translate('users.credential')}</th>
                        <th scope="col" aria-label={translate('users.actions')} />
                      </tr>
                    </thead>
                    <tbody>
                      {userRows}
                      {usersLoadStatus === 'loaded' && visibleUserCount === 0 ? (
                        <tr>
                          <td className="emptyUsers" colSpan={6}>
                            {users.length === 0 ? translate('users.empty') : translate('users.emptyFiltered')}
                          </td>
                        </tr>
                      ) : null}
                    </tbody>
                  </table>
                </div>
              </>
            ) : (
              <RepositorySettings
                loadStatus={repositoryContextLoadStatus}
                onRetry={loadRepositoryContext}
                repositoryContext={repositoryContext}
                repositoryContextError={repositoryContextError}
                sessionRole={sessionRole}
                t={translate}
              />
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

            <form className="drawerForm" onSubmit={saveUser}>
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
                  disabled={Boolean(editingId)}
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
                    setDraft((currentDraft) => ({ ...currentDraft, role: event.target.value as OrganizationUserRole }))
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

              {editingId ? (
              <label className="field">
                <span>{translate('users.status')}</span>
                <select
                  onChange={(event) =>
                    setDraft((currentDraft) => ({ ...currentDraft, state: event.target.value as OrganizationUserState }))
                  }
                  value={draft.state}
                >
                  {USER_STATUSES.map((status) => (
                    <option key={status} value={status}>
                      {translate(`statuses.${status}`)}
                    </option>
                  ))}
                </select>
              </label>
              ) : null}

              {isEditingSelf ? <p className="fieldMessage">{translate('users.selfActionHelper')}</p> : null}
              {isEditingOtherOwner ? <p className="fieldMessage">{translate('users.otherOwnerWarning')}</p> : null}

              {editingId && editingRecord ? (
                <section className="resetCredentialPanel" aria-label={translate('users.resetTemporaryPassword')}>
                  <div>
                    <h4>{translate('users.resetTemporaryPassword')}</h4>
                    <p>{isEditingSelf ? translate('users.selfResetBlockedCopy') : translate('users.resetTemporaryPasswordCopy')}</p>
                  </div>
                  <button
                    className="ghostButton dangerButton"
                    disabled={isResettingTemporaryPassword || isEditingSelf}
                    onClick={() => requestTemporaryPasswordReset(editingRecord)}
                    type="button"
                  >
                    {translate('users.resetTemporaryPassword')}
                  </button>
                </section>
              ) : null}

              {formError ? <p className="fieldMessage" role="alert">{formError}</p> : null}
            </div>

            <footer className="drawerFooter">
              <button className="ghostButton" onClick={closeDrawer} type="button">
                {translate('users.cancel')}
              </button>
              <button className="primaryButton" type="submit">
                {translate('users.save')}
              </button>
            </footer>
            </form>
          </aside>
        </>
      ) : null}

      {resetTarget ? (
        <>
          <button
            aria-label={translate('users.cancelTemporaryPasswordReset')}
            className="confirmScrim"
            disabled={isResettingTemporaryPassword}
            onClick={cancelTemporaryPasswordReset}
            type="button"
          />
          <section
            aria-label={translate('users.confirmTemporaryPasswordResetTitle')}
            aria-modal="true"
            className="confirmPanel"
            role="dialog"
          >
            <div>
              <p className="kicker">{translate('users.confirmTemporaryPasswordResetKicker')}</p>
              <h3>{translate('users.confirmTemporaryPasswordResetTitle')}</h3>
              <p>{translate('users.confirmTemporaryPasswordResetCopy', { email: resetTarget.user.email ?? resetTarget.user.id })}</p>
            </div>
            {resetError ? <p className="fieldMessage" role="alert">{resetError}</p> : null}
            <footer className="confirmActions">
              <button className="ghostButton" disabled={isResettingTemporaryPassword} onClick={cancelTemporaryPasswordReset} type="button">
                {translate('users.cancel')}
              </button>
              <button className="primaryButton dangerButton" disabled={isResettingTemporaryPassword} onClick={confirmTemporaryPasswordReset} type="button">
                {isResettingTemporaryPassword ? translate('users.resettingTemporaryPassword') : translate('users.confirmTemporaryPasswordReset')}
              </button>
            </footer>
          </section>
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

function RepositorySettings({
  loadStatus,
  onRetry,
  repositoryContext,
  repositoryContextError,
  sessionRole,
  t,
}: {
  loadStatus: RepositoryContextLoadStatus;
  onRetry: () => void;
  repositoryContext: OrganizationRepositoryContextResponse | null;
  repositoryContextError: string;
  sessionRole: string;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const contexts = repositoryContext?.contexts ?? [];

  return (
    <div className="repositoryPanel">
      <div className="repositoryHeader">
        <div>
          <h3>{t('repository.title')}</h3>
          <p>{t('repository.copy')}</p>
        </div>
        <p className="themeDisclaimer">{t('repository.metadataOnly')}</p>
      </div>

      {loadStatus === 'loading' ? <p className="usersNotice" role="status">{t('repository.loading')}</p> : null}

      {loadStatus === 'error' ? (
        <div className="usersNotice usersNoticeError" role="alert">
          <p>{repositoryContextError}</p>
          <button className="ghostButton" onClick={onRetry} type="button">
            {t('repository.retry')}
          </button>
        </div>
      ) : null}

      {repositoryContext ? (
        <section className="repositoryOrgPanel" aria-label={t('repository.organization')}>
          <div>
            <span>{t('repository.organization')}</span>
            <h4>{repositoryContext.organization.display_name || repositoryContext.organization.slug || repositoryContext.organization.id}</h4>
          </div>
          <dl className="repositoryKeyGrid">
            <FieldRow label={t('repository.fields.organizationId')} value={repositoryContext.organization.id} />
            <FieldRow label={t('repository.fields.organizationSlug')} value={repositoryContext.organization.slug || t('repository.emptyValue')} />
            <FieldRow label={t('repository.fields.organizationState')} value={repositoryContext.organization.state || t('repository.emptyValue')} />
            <FieldRow label={t('repository.fields.currentRole')} value={sessionRole} />
          </dl>
        </section>
      ) : null}

      {loadStatus === 'loaded' && contexts.length === 0 ? (
        <section className="repositoryEmpty" aria-label={t('repository.emptyTitle')}>
          <h4>{t('repository.emptyTitle')}</h4>
          <p>{t('repository.emptyCopy')}</p>
        </section>
      ) : null}

      {contexts.length > 0 ? (
        <div className="repositoryContextGrid" aria-label={t('repository.contexts')}>
          {contexts.map((context) => (
            <RepositoryContextCard context={context} key={`${context.project.id}-${context.repo_binding.id}`} t={t} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function RepositoryContextCard({
  context,
  t,
}: {
  context: RepositoryContextRecord;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  return (
    <article className="repositoryCard">
      <header>
        <div>
          <span>{t('repository.project')}</span>
          <h4>{context.project.display_name || context.project.slug || context.project.id}</h4>
        </div>
        <span className="pill">{context.repo_binding.access_mode || t('repository.emptyValue')}</span>
      </header>

      <dl className="repositoryKeyGrid">
        <FieldRow label={t('repository.fields.projectId')} value={context.project.id} />
        <FieldRow label={t('repository.fields.projectSlug')} value={context.project.slug || t('repository.emptyValue')} />
        <FieldRow label={t('repository.fields.projectState')} value={context.project.state || t('repository.emptyValue')} />
        <FieldRow label={t('repository.fields.repoBindingId')} value={context.repo_binding.id} />
        <FieldRow label={t('repository.fields.provider')} value={context.repo_binding.provider || t('repository.emptyValue')} />
        <FieldRow label={t('repository.fields.repositoryFullName')} value={context.repo_binding.repository_full_name || t('repository.emptyValue')} />
        <FieldRow label={t('repository.fields.repositoryUrl')} value={context.repo_binding.repository_url || t('repository.emptyValue')} />
        <FieldRow label={t('repository.fields.defaultBranch')} value={context.repo_binding.default_branch || t('repository.emptyValue')} />
        <FieldRow label={t('repository.fields.workflowBaseBranch')} value={context.repo_binding.workflow_base_branch || t('repository.emptyValue')} />
        <FieldRow label={t('repository.fields.pathScope')} value={context.repo_binding.path_scope || t('repository.emptyValue')} />
        <FieldRow label={t('repository.fields.repoBindingState')} value={context.repo_binding.state || t('repository.emptyValue')} />
        <FieldRow label={t('repository.fields.updatedAt')} value={formatDateTime(context.repo_binding.updated_at)} />
      </dl>

      <p className="repositoryFootnote">{t('repository.accessModeFootnote')}</p>
    </article>
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

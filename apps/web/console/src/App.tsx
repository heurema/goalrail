import { FormEvent, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  changePassword,
  isAuthClientError,
  login as loginWithPassword,
  logout as logoutSession,
  me as fetchCurrentProfile,
} from './authClient';
import type { AuthClientError, LoginResponse, MeResponse } from './authClient';
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
import {
  isQualificationFeedClientError,
  listQualificationFeed,
} from './qualificationFeedClient';
import {
  isContractListClientError,
  listContracts,
} from './contractListClient';
import {
  getContractDetail,
  isContractDetailClientError,
} from './contractDetailClient';
import {
  getCurrentContractDraft,
  isContractDraftClientError,
} from './contractDraftClient';
import type { OrganizationUserRecord, OrganizationUserRole, OrganizationUserState } from './usersClient';
import type { OrganizationRepositoryContextResponse, RepositoryContextRecord } from './repositoryContextClient';
import type {
  QualificationFeedItem,
  QualificationFeedResponse,
} from './qualificationFeedClient';
import type { ContractListResponse, ContractStateFilter } from './contractListClient';
import type { ContractResponse } from './contractDetailClient';
import type { ContractDraftResponse, ContractDraftSourceRef } from './contractDraftClient';
import { READINESS_DISPLAY_LANES, projectReadinessDisplay, sortReadinessItems } from './readinessDisplay';
import { formatCalmTimestamp } from './uiTime';
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
type ContractListLoadStatus = 'idle' | 'loading' | 'loaded' | 'error';
type ContractListStateFilter = ContractStateFilter | 'all';
type ContractListRepoBindingFilter = string;
type ContractSelectionSource = 'auto' | 'list' | 'manual' | 'linked' | null;
type ContractDraftLoadStatus = 'idle' | 'loading' | 'loaded' | 'no_draft' | 'unavailable' | 'error';
type QualificationFeedLoadStatus = 'idle' | 'loading' | 'loaded' | 'error';
type ReadOnlyLoadResult = 'loaded' | 'error' | 'not_found' | 'skipped' | 'stale' | 'unauthorized';
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
const CONTRACT_LIST_LIMIT = 50;
const CONTRACT_STATE_FILTERS: ContractListStateFilter[] = ['all', 'draft', 'ready_for_approval', 'approved', 'seeded'];
const ALL_REPOSITORIES_FILTER = 'all';
const CONTRACT_REFRESH_INTERVAL_MS = 5000;
const QUALIFICATION_FEED_LIMIT = 50;
const QUALIFICATION_FEED_POLL_INTERVAL_MS = 5000;
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

function qualificationFeedErrorMessage(error: unknown, t: (key: string, options?: Record<string, unknown>) => string) {
  if (isQualificationFeedClientError(error)) {
    const translated = t(`qualificationFeed.errors.${error.code}`);
    if (translated !== `qualificationFeed.errors.${error.code}`) {
      return translated;
    }

    return error.status ? t('qualificationFeed.errors.genericWithStatus', { status: error.status }) : t('qualificationFeed.errors.generic');
  }

  return t('qualificationFeed.errors.generic');
}

function contractListErrorMessage(error: unknown, t: (key: string, options?: Record<string, unknown>) => string) {
  if (isContractListClientError(error)) {
    const translated = t(`surfaces.contracts.listErrors.${error.code}`);
    if (translated !== `surfaces.contracts.listErrors.${error.code}`) {
      return translated;
    }

    return error.status
      ? t('surfaces.contracts.listErrors.genericWithStatus', { status: error.status })
      : t('surfaces.contracts.listErrors.generic');
  }

  return t('surfaces.contracts.listErrors.generic');
}

function contractDetailErrorMessage(error: unknown, t: (key: string, options?: Record<string, unknown>) => string) {
  if (isContractDetailClientError(error)) {
    const translated = t(`surfaces.contracts.detailErrors.${error.code}`);
    if (translated !== `surfaces.contracts.detailErrors.${error.code}`) {
      return translated;
    }

    return error.status
      ? t('surfaces.contracts.detailErrors.genericWithStatus', { status: error.status })
      : t('surfaces.contracts.detailErrors.generic');
  }

  return t('surfaces.contracts.detailErrors.generic');
}

function isSilentPreservableContractDetailError(error: unknown) {
  return isContractDetailClientError(error)
    && (
      error.code === 'network_error'
      || error.code === 'response_parse_error'
      || error.code === 'server_error'
    );
}

function contractDraftErrorMessage(error: unknown, t: (key: string, options?: Record<string, unknown>) => string) {
  if (isContractDraftClientError(error)) {
    const translated = t(`surfaces.contracts.draftErrors.${error.code}`);
    if (translated !== `surfaces.contracts.draftErrors.${error.code}`) {
      return translated;
    }

    return error.status
      ? t('surfaces.contracts.draftErrors.genericWithStatus', { status: error.status })
      : t('surfaces.contracts.draftErrors.generic');
  }

  return t('surfaces.contracts.draftErrors.generic');
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
      const target = `/start${window.location.search}${window.location.hash}`;
      window.history.replaceState(window.history.state, '', target);
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
  const [qualificationFeed, setQualificationFeed] = useState<QualificationFeedResponse>({ items: [] });
  const [qualificationFeedLoadStatus, setQualificationFeedLoadStatus] = useState<QualificationFeedLoadStatus>('idle');
  const [qualificationFeedError, setQualificationFeedError] = useState('');
  const [formError, setFormError] = useState('');
  const [temporaryPassword, setTemporaryPassword] = useState<OneTimeTemporaryPassword | null>(null);
  const [contractIdInput, setContractIdInput] = useState('');
  const [contractLoadStatus, setContractLoadStatus] = useState<ContractLoadStatus>('idle');
  const [contractError, setContractError] = useState('');
  const [contract, setContract] = useState<ContractResponse | null>(null);
  const [contractList, setContractList] = useState<ContractListResponse>({ contracts: [], limit: CONTRACT_LIST_LIMIT });
  const [contractListLoadStatus, setContractListLoadStatus] = useState<ContractListLoadStatus>('idle');
  const [contractListError, setContractListError] = useState('');
  const [contractListStateFilter, setContractListStateFilter] = useState<ContractListStateFilter>('all');
  const [contractListRepoBindingFilter, setContractListRepoBindingFilter] =
    useState<ContractListRepoBindingFilter>(ALL_REPOSITORIES_FILTER);
  const [contractDraft, setContractDraft] = useState<ContractDraftResponse | null>(null);
  const [contractDraftLoadStatus, setContractDraftLoadStatus] = useState<ContractDraftLoadStatus>('idle');
  const [contractDraftError, setContractDraftError] = useState('');
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
  const contractListLoadSequence = useRef(0);
  const contractDraftLoadSequence = useRef(0);
  const selectedContractId = useRef<string | null>(null);
  const selectedContractSnapshot = useRef<ContractResponse | null>(null);
  const selectedContractDraftSnapshot = useRef<ContractDraftResponse | null>(null);
  const contractSelectionSource = useRef<ContractSelectionSource>(null);
  const contractListRowsCount = useRef(0);
  const contractSurfacePollInFlight = useRef(false);
  const contractDetailRefreshInFlight = useRef(false);
  const contractDraftRefreshInFlight = useRef(false);
  const usersLoadSequence = useRef(0);
  const repositoryContextLoadSequence = useRef(0);
  const qualificationFeedLoadSequence = useRef(0);
  const authSessionSequence = useRef(0);

  useEffect(() => {
    document.documentElement.lang = activeLocale;
  }, [activeLocale]);

  useEffect(() => {
    selectedContractId.current = contract?.id ?? null;
    selectedContractSnapshot.current = contract;
  }, [contract]);

  useEffect(() => {
    contractListRowsCount.current = contractList.contracts.length;
  }, [contractList.contracts.length]);

  useEffect(() => {
    if (screen !== 'console' || activeSurface !== 'contracts' || authStatus !== 'authenticated') {
      return;
    }

    const accessToken = tokens?.accessToken ?? '';
    if (!accessToken) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return;
    }

    let cancelled = false;
    let timeoutId: ReturnType<typeof window.setTimeout> | undefined;

    function clearScheduledPoll() {
      if (timeoutId !== undefined) {
        window.clearTimeout(timeoutId);
        timeoutId = undefined;
      }
    }

    function scheduleNextPoll() {
      clearScheduledPoll();
      timeoutId = window.setTimeout(() => {
        void poll(false);
      }, CONTRACT_REFRESH_INTERVAL_MS);
    }

    async function poll(showLoading: boolean) {
      if (cancelled || contractSurfacePollInFlight.current) {
        return;
      }
      if (document.hidden) {
        scheduleNextPoll();
        return;
      }

      contractSurfacePollInFlight.current = true;
      try {
        const listResult = await refreshContractsSurface(showLoading, {
          silentTransientErrors: !showLoading,
        });
        if (cancelled || listResult === 'unauthorized') {
          return;
        }
      } finally {
        contractSurfacePollInFlight.current = false;
        if (!cancelled) {
          scheduleNextPoll();
        }
      }
    }

    function handleVisibilityChange() {
      if (cancelled || document.hidden) {
        return;
      }
      clearScheduledPoll();
      void poll(false);
    }

    document.addEventListener('visibilitychange', handleVisibilityChange);
    void poll(contractListLoadStatus === 'idle');

    return () => {
      cancelled = true;
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      contractSurfacePollInFlight.current = false;
      contractListLoadSequence.current += 1;
      contractLookupSequence.current += 1;
      contractDraftLoadSequence.current += 1;
      contractDraftRefreshInFlight.current = false;
      clearScheduledPoll();
    };
  }, [activeSurface, authStatus, contractListRepoBindingFilter, contractListStateFilter, screen, tokens?.accessToken]);

  useEffect(() => {
    if (screen !== 'console' || activeSurface !== 'contracts' || authStatus !== 'authenticated') {
      return;
    }
    if (repositoryContextLoadStatus !== 'idle') {
      return;
    }

    void loadRepositoryContext();
  }, [activeSurface, authStatus, profile?.organization_membership.organization_id, repositoryContextLoadStatus, screen, tokens?.accessToken]);

  useEffect(() => {
    if (contractListRepoBindingFilter === ALL_REPOSITORIES_FILTER || repositoryContextLoadStatus === 'loading') {
      return;
    }
    const contexts = repositoryContext?.contexts ?? [];
    if (!contexts.some((context) => context.repo_binding.id === contractListRepoBindingFilter)) {
      setContractListRepoBindingFilter(ALL_REPOSITORIES_FILTER);
    }
  }, [contractListRepoBindingFilter, repositoryContext, repositoryContextLoadStatus]);

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
    if (screen !== 'console' || activeSurface !== 'delivery-readiness' || authStatus !== 'authenticated') {
      return;
    }

    const accessToken = tokens?.accessToken ?? '';
    if (!accessToken) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return;
    }

    let cancelled = false;
    let timeoutId: ReturnType<typeof window.setTimeout> | undefined;

    function scheduleNextPoll() {
      timeoutId = window.setTimeout(() => {
        void poll(false);
      }, QUALIFICATION_FEED_POLL_INTERVAL_MS);
    }

    async function poll(showLoading: boolean) {
      if (cancelled) {
        return;
      }
      if (document.hidden) {
        scheduleNextPoll();
        return;
      }
      const loadSequence = qualificationFeedLoadSequence.current + 1;
      qualificationFeedLoadSequence.current = loadSequence;
      if (showLoading) {
        setQualificationFeedLoadStatus('loading');
      }
      setQualificationFeedError('');

      try {
        const result = await listQualificationFeed({
          accessToken,
          limit: QUALIFICATION_FEED_LIMIT,
        });
        if (cancelled || qualificationFeedLoadSequence.current !== loadSequence) {
          return;
        }
        setQualificationFeed(result);
        setQualificationFeedLoadStatus('loaded');
      } catch (error) {
        if (cancelled || qualificationFeedLoadSequence.current !== loadSequence) {
          return;
        }
        if (isQualificationFeedClientError(error) && error.code === 'unauthorized') {
          cancelled = true;
          resetAuthState();
          setAuthError(translate('auth.invalidSession'));
          return;
        }
        setQualificationFeedLoadStatus('error');
        setQualificationFeedError(qualificationFeedErrorMessage(error, translate));
      } finally {
        if (!cancelled) {
          scheduleNextPoll();
        }
      }
    }

    function handleVisibilityChange() {
      if (cancelled || document.hidden) {
        return;
      }
      if (timeoutId !== undefined) {
        window.clearTimeout(timeoutId);
        timeoutId = undefined;
      }
      void poll(false);
    }

    document.addEventListener('visibilitychange', handleVisibilityChange);
    void poll(qualificationFeedLoadStatus === 'idle');

    return () => {
      cancelled = true;
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      qualificationFeedLoadSequence.current += 1;
      if (timeoutId !== undefined) {
        window.clearTimeout(timeoutId);
      }
    };
  }, [activeSurface, authStatus, screen, tokens?.accessToken]);

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
    contractListLoadSequence.current += 1;
    contractDraftLoadSequence.current += 1;
    usersLoadSequence.current += 1;
    repositoryContextLoadSequence.current += 1;
    qualificationFeedLoadSequence.current += 1;
    selectedContractId.current = null;
    selectedContractSnapshot.current = null;
    selectedContractDraftSnapshot.current = null;
    contractSelectionSource.current = null;
    contractSurfacePollInFlight.current = false;
    contractDetailRefreshInFlight.current = false;
    contractDraftRefreshInFlight.current = false;
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
    setQualificationFeed({ items: [] });
    setQualificationFeedLoadStatus('idle');
    setQualificationFeedError('');
    setFormError('');
    setTemporaryPassword(null);
    setResetTarget(null);
    setResetError('');
    setResettingUserId(null);
    setContractIdInput('');
    setContractLoadStatus('idle');
    setContractError('');
    setContract(null);
    setContractList({ contracts: [], limit: CONTRACT_LIST_LIMIT });
    setContractListLoadStatus('idle');
    setContractListError('');
    setContractListStateFilter('all');
    setContractListRepoBindingFilter(ALL_REPOSITORIES_FILTER);
    setContractDraft(null);
    setContractDraftLoadStatus('idle');
    setContractDraftError('');
  }

  async function handleContractLookup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await loadContractById(contractIdInput, 'manual');
  }

  function clearContractDraftState(status: ContractDraftLoadStatus = 'idle', message = '') {
    contractDraftLoadSequence.current += 1;
    contractDraftRefreshInFlight.current = false;
    selectedContractDraftSnapshot.current = null;
    setContractDraft(null);
    setContractDraftLoadStatus(status);
    setContractDraftError(message);
  }

  async function loadContractDraftForContract(
    contractRecord: ContractResponse,
    options: { silentTransientErrors?: boolean } = {}
  ): Promise<ReadOnlyLoadResult> {
    const accessToken = tokens?.accessToken;
    const contractId = contractRecord.id;
    const draftId = contractRecord.current_draft_id;
    const loadSequence = contractDraftLoadSequence.current + 1;
    contractDraftLoadSequence.current = loadSequence;

    if (!draftId) {
      selectedContractDraftSnapshot.current = null;
      setContractDraft(null);
      setContractDraftLoadStatus('no_draft');
      setContractDraftError('');
      return 'skipped';
    }

    if (!accessToken) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return 'unauthorized';
    }

    contractDraftRefreshInFlight.current = true;
    if (!options.silentTransientErrors) {
      selectedContractDraftSnapshot.current = null;
      setContractDraft(null);
      setContractDraftLoadStatus('loading');
      setContractDraftError('');
    }

    try {
      const draft = await getCurrentContractDraft({
        accessToken,
        contractId,
      });
      if (contractDraftLoadSequence.current !== loadSequence || selectedContractId.current !== contractId) {
        return 'stale';
      }
      selectedContractDraftSnapshot.current = draft;
      setContractDraft(draft);
      setContractDraftLoadStatus('loaded');
      setContractDraftError('');
      return 'loaded';
    } catch (error) {
      if (contractDraftLoadSequence.current !== loadSequence || selectedContractId.current !== contractId) {
        return 'stale';
      }
      if (isContractDraftClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(translate('auth.invalidSession'));
        return 'unauthorized';
      }
      if (isContractDraftClientError(error) && (error.code === 'not_found' || error.code === 'invalid_state')) {
        selectedContractDraftSnapshot.current = null;
        setContractDraft(null);
        setContractDraftLoadStatus('unavailable');
        setContractDraftError(contractDraftErrorMessage(error, translate));
        return 'not_found';
      }

      if (
        options.silentTransientErrors
        && selectedContractDraftSnapshot.current?.contract_id === contractId
        && selectedContractDraftSnapshot.current?.id === draftId
      ) {
        setContractDraftLoadStatus('loaded');
        setContractDraftError('');
        return 'error';
      }

      selectedContractDraftSnapshot.current = null;
      setContractDraft(null);
      setContractDraftLoadStatus('error');
      setContractDraftError(contractDraftErrorMessage(error, translate));
      return 'error';
    } finally {
      if (contractDraftLoadSequence.current === loadSequence) {
        contractDraftRefreshInFlight.current = false;
      }
    }
  }

  async function refreshSelectedContractDraft(
    options: { silentTransientErrors?: boolean } = {}
  ): Promise<ReadOnlyLoadResult> {
    const selected = selectedContractSnapshot.current;
    if (!selected || contractDraftRefreshInFlight.current) {
      return 'skipped';
    }

    return loadContractDraftForContract(selected, options);
  }

  async function loadContractById(contractIdValue: string, selectionSource: Exclude<ContractSelectionSource, 'auto'> = 'manual') {
    const contractId = contractIdValue.trim();
    const accessToken = tokens?.accessToken;

    if (!contractId) {
      contractLookupSequence.current += 1;
      contractSelectionSource.current = null;
      setContractError(translate('surfaces.contracts.lookupMissing'));
      setContractLoadStatus('error');
      selectedContractId.current = null;
      selectedContractSnapshot.current = null;
      setContract(null);
      clearContractDraftState();
      return;
    }

    if (!accessToken) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return;
    }

    const lookupSequence = contractLookupSequence.current + 1;
    contractLookupSequence.current = lookupSequence;
    contractSelectionSource.current = selectionSource;
    setContractLoadStatus('loading');
    setContractError('');
    selectedContractId.current = null;
    selectedContractSnapshot.current = null;
    setContract(null);
    clearContractDraftState();

    try {
      const found = await getContractDetail({ accessToken, contractId });
      if (contractLookupSequence.current !== lookupSequence) {
        return;
      }
      selectedContractId.current = found.id;
      selectedContractSnapshot.current = found;
      setContract(found);
      setContractIdInput(found.id);
      setContractLoadStatus('loaded');
      await loadContractDraftForContract(found);
    } catch (error) {
      if (contractLookupSequence.current !== lookupSequence) {
        return;
      }
      selectedContractId.current = null;
      selectedContractSnapshot.current = null;
      setContract(null);
      clearContractDraftState();
      if (isContractDetailClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(translate('auth.invalidSession'));
        return;
      }
      if (isContractDetailClientError(error) && error.code === 'not_found') {
        setContractLoadStatus('not_found');
        setContractError(translate('surfaces.contracts.notFound'));
        return;
      }
      setContractLoadStatus('error');
      setContractError(contractDetailErrorMessage(error, translate));
    }
  }

  async function loadContractList(
    showLoading = true,
    options: { silentTransientErrors?: boolean } = {}
  ): Promise<ReadOnlyLoadResult> {
    const accessToken = tokens?.accessToken;
    if (!accessToken) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return 'unauthorized';
    }

    const loadSequence = contractListLoadSequence.current + 1;
    contractListLoadSequence.current = loadSequence;
    if (showLoading) {
      setContractListLoadStatus('loading');
    }
    if (!options.silentTransientErrors) {
      setContractListError('');
    }

    try {
      const result = await listContracts({
        accessToken,
        repoBindingId: contractListRepoBindingFilter === ALL_REPOSITORIES_FILTER ? undefined : contractListRepoBindingFilter,
        state: contractListStateFilter === 'all' ? undefined : contractListStateFilter,
        limit: CONTRACT_LIST_LIMIT,
      });
      if (contractListLoadSequence.current !== loadSequence) {
        return 'stale';
      }
      setContractList(result);
      setContractListLoadStatus('loaded');
      setContractListError('');
      reconcileContractListSelection(result.contracts);
      return 'loaded';
    } catch (error) {
      if (contractListLoadSequence.current !== loadSequence) {
        return 'stale';
      }
      if (isContractListClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(translate('auth.invalidSession'));
        return 'unauthorized';
      }

      if (options.silentTransientErrors && contractListRowsCount.current > 0) {
        return 'error';
      }
      setContractListLoadStatus('error');
      setContractListError(contractListErrorMessage(error, translate));
      return 'error';
    }
  }

  async function refreshContractsSurface(
    showLoading = true,
    options: { silentTransientErrors?: boolean } = {}
  ): Promise<ReadOnlyLoadResult> {
    const selectedDetailAtRefreshStart = selectedContractId.current;
    const listResult = await loadContractList(showLoading, options);
    if (listResult !== 'loaded') {
      return listResult;
    }
    if (selectedDetailAtRefreshStart && selectedContractId.current === selectedDetailAtRefreshStart) {
      return refreshSelectedContractDetail(options);
    }
    if (selectedContractId.current) {
      return refreshSelectedContractDraft(options);
    }

    return listResult;
  }

  async function refreshSelectedContractDetail(
    options: { silentTransientErrors?: boolean } = {}
  ): Promise<ReadOnlyLoadResult> {
    const accessToken = tokens?.accessToken;
    const contractId = selectedContractId.current;
    if (!accessToken) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return 'unauthorized';
    }
    if (!contractId || contractDetailRefreshInFlight.current) {
      return 'skipped';
    }

    contractDetailRefreshInFlight.current = true;
    const lookupSequence = contractLookupSequence.current;
    try {
      const found = await getContractDetail({ accessToken, contractId });
      if (contractLookupSequence.current !== lookupSequence || selectedContractId.current !== contractId) {
        return 'stale';
      }
      selectedContractSnapshot.current = found;
      setContract(found);
      setContractIdInput(found.id);
      setContractLoadStatus('loaded');
      setContractError('');
      const draftResult = await loadContractDraftForContract(found, options);
      if (draftResult === 'unauthorized') {
        return 'unauthorized';
      }
      return 'loaded';
    } catch (error) {
      if (contractLookupSequence.current !== lookupSequence || selectedContractId.current !== contractId) {
        return 'stale';
      }
      if (isContractDetailClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(translate('auth.invalidSession'));
        return 'unauthorized';
      }
      if (isContractDetailClientError(error) && error.code === 'not_found') {
        selectedContractId.current = null;
        selectedContractSnapshot.current = null;
        setContract(null);
        clearContractDraftState();
        setContractLoadStatus('not_found');
        setContractError(translate('surfaces.contracts.notFound'));
        return 'not_found';
      }

      if (
        options.silentTransientErrors
        && selectedContractSnapshot.current?.id === contractId
        && isSilentPreservableContractDetailError(error)
      ) {
        setContractLoadStatus('loaded');
        return 'error';
      }
      selectedContractId.current = null;
      selectedContractSnapshot.current = null;
      setContract(null);
      clearContractDraftState();
      setContractLoadStatus('error');
      setContractError(contractDetailErrorMessage(error, translate));
      return 'error';
    } finally {
      contractDetailRefreshInFlight.current = false;
    }
  }

  function reconcileContractListSelection(contracts: ContractResponse[]) {
    const selectedId = selectedContractId.current;
    const selectionSource = contractSelectionSource.current;

    if (selectedId) {
      const selectedSummary = contracts.find((record) => record.id === selectedId);
      if (selectedSummary) {
        const nextSnapshot = selectedContractSnapshot.current?.id === selectedSummary.id
          ? { ...selectedContractSnapshot.current, ...selectedSummary }
          : selectedSummary;
        selectedContractSnapshot.current = nextSnapshot;
        setContract((current) => (current?.id === selectedSummary.id ? { ...current, ...selectedSummary } : current));
        return;
      }
      return;
    }

    if (!selectionSource && contracts.length > 0) {
      selectContractSummary(contracts[0], 'auto');
    }
  }

  function selectContractSummary(nextContract: ContractResponse, selectionSource: ContractSelectionSource) {
    contractSelectionSource.current = selectionSource;
    selectedContractId.current = nextContract.id;
    selectedContractSnapshot.current = nextContract;
    setContractIdInput(nextContract.id);
    setContract(nextContract);
    setContractLoadStatus('loaded');
    setContractError('');
    void loadContractDraftForContract(nextContract);
  }

  function handleContractListSelection(nextContract: ContractResponse) {
    setContractIdInput(nextContract.id);
    void loadContractById(nextContract.id, 'list');
  }

  async function openLinkedContract(contractId: string) {
    const nextContractId = contractId.trim();
    if (!nextContractId) {
      return;
    }
    setScreen('console');
    setActiveSurface('contracts');
    setContractIdInput(nextContractId);
    await loadContractById(nextContractId, 'linked');
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

  async function loadQualificationFeedOnce() {
    const accessToken = tokens?.accessToken;
    if (!accessToken) {
      resetAuthState();
      setAuthError(translate('auth.invalidSession'));
      return;
    }

    const loadSequence = qualificationFeedLoadSequence.current + 1;
    qualificationFeedLoadSequence.current = loadSequence;
    setQualificationFeedLoadStatus('loading');
    setQualificationFeedError('');

    try {
      const result = await listQualificationFeed({
        accessToken,
        limit: QUALIFICATION_FEED_LIMIT,
      });
      if (qualificationFeedLoadSequence.current !== loadSequence) {
        return;
      }
      setQualificationFeed(result);
      setQualificationFeedLoadStatus('loaded');
    } catch (error) {
      if (qualificationFeedLoadSequence.current !== loadSequence) {
        return;
      }
      if (isQualificationFeedClientError(error) && error.code === 'unauthorized') {
        resetAuthState();
        setAuthError(translate('auth.invalidSession'));
        return;
      }

      setQualificationFeedLoadStatus('error');
      setQualificationFeedError(qualificationFeedErrorMessage(error, translate));
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
            activeLocale={activeLocale}
            contract={contract}
            contractError={contractError}
            contractIdInput={contractIdInput}
            contractList={contractList}
            contractListError={contractListError}
            contractListLoadStatus={contractListLoadStatus}
            contractListRepoBindingFilter={contractListRepoBindingFilter}
            contractListStateFilter={contractListStateFilter}
            loadStatus={contractLoadStatus}
            onContractIdChange={setContractIdInput}
            onContractListRepoBindingFilterChange={setContractListRepoBindingFilter}
            onContractListStateFilterChange={setContractListStateFilter}
            onContractSelect={handleContractListSelection}
            onLookup={handleContractLookup}
            onRefresh={() => void refreshContractsSurface(true)}
            repositoryContext={repositoryContext}
            repositoryContextLoadStatus={repositoryContextLoadStatus}
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
            activeLocale={activeLocale}
            contract={contract}
            contractDraft={contractDraft}
            contractDraftError={contractDraftError}
            contractDraftLoadStatus={contractDraftLoadStatus}
            loadStatus={contractLoadStatus}
            repositoryContext={repositoryContext}
            repositoryContextError={repositoryContextError}
            repositoryContextLoadStatus={repositoryContextLoadStatus}
            t={translate}
          />
        ) : activeSurface === 'delivery-readiness' ? (
          <QualificationFeedSurfacePanel
            feed={qualificationFeed}
            loadStatus={qualificationFeedLoadStatus}
            locale={activeLocale}
            onOpenContract={openLinkedContract}
            onRetry={loadQualificationFeedOnce}
            qualificationFeedError={qualificationFeedError}
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
  activeLocale,
  contract,
  contractError,
  contractIdInput,
  contractList,
  contractListError,
  contractListLoadStatus,
  contractListRepoBindingFilter,
  contractListStateFilter,
  loadStatus,
  onContractIdChange,
  onContractListRepoBindingFilterChange,
  onContractListStateFilterChange,
  onContractSelect,
  onLookup,
  onRefresh,
  repositoryContext,
  repositoryContextLoadStatus,
  t,
}: {
  activeLocale: ConsoleLocale;
  contract: ContractResponse | null;
  contractError: string;
  contractIdInput: string;
  contractList: ContractListResponse;
  contractListError: string;
  contractListLoadStatus: ContractListLoadStatus;
  contractListRepoBindingFilter: ContractListRepoBindingFilter;
  contractListStateFilter: ContractListStateFilter;
  loadStatus: ContractLoadStatus;
  onContractIdChange: (value: string) => void;
  onContractListRepoBindingFilterChange: (value: ContractListRepoBindingFilter) => void;
  onContractListStateFilterChange: (value: ContractListStateFilter) => void;
  onContractSelect: (contract: ContractResponse) => void;
  onLookup: (event: FormEvent<HTMLFormElement>) => void;
  onRefresh: () => void;
  repositoryContext: OrganizationRepositoryContextResponse | null;
  repositoryContextLoadStatus: RepositoryContextLoadStatus;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const isLookupLoading = loadStatus === 'loading';
  const isListLoading = contractListLoadStatus === 'loading';
  const contracts = contractList.contracts;
  const repositoryOptions = repositoryContext?.contexts ?? [];
  const repositoryFilterDisabled = repositoryOptions.length === 0;
  const repositoryFilterHintVisible = repositoryFilterDisabled && repositoryContextLoadStatus !== 'loading';

  return (
    <section className="contractRailPanel" aria-label={t('surfaces.contracts.ops.surfaceContext')}>
      <div className="opsGroupLabel">{t('surfaces.contracts.ops.surfaceContext')}</div>
      <div className="opsRailSection">
        <h3>{t('surfaces.contracts.nav')}</h3>
        <p>{t('surfaces.contracts.ops.discoveryCopy', { limit: contractList.limit })}</p>
      </div>

      <div className="opsContractToolbar">
        <label>
          <span>{t('surfaces.contracts.stateFilterLabel')}</span>
          <select
            aria-label={t('surfaces.contracts.stateFilterLabel')}
            onChange={(event) => onContractListStateFilterChange(event.target.value as ContractListStateFilter)}
            value={contractListStateFilter}
          >
            {CONTRACT_STATE_FILTERS.map((state) => (
              <option key={state} value={state}>
                {state === 'all' ? t('surfaces.contracts.allStates') : t(`contractStates.${state}`)}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>{t('surfaces.contracts.repositoryFilterLabel')}</span>
          <select
            aria-label={t('surfaces.contracts.repositoryFilterLabel')}
            disabled={repositoryFilterDisabled}
            onChange={(event) => onContractListRepoBindingFilterChange(event.target.value)}
            value={repositoryFilterDisabled ? ALL_REPOSITORIES_FILTER : contractListRepoBindingFilter}
          >
            <option value={ALL_REPOSITORIES_FILTER}>{t('surfaces.contracts.allRepositories')}</option>
            {repositoryOptions.map((context) => (
              <option key={context.repo_binding.id} value={context.repo_binding.id}>
                {repositoryFilterOptionLabel(context, t)}
              </option>
            ))}
          </select>
        </label>
        {repositoryFilterHintVisible ? (
          <p className="opsFilterNote">{t('surfaces.contracts.repositoryFilterUnavailable')}</p>
        ) : null}
        <button className="ghostButton" disabled={isListLoading} onClick={onRefresh} type="button">
          {isListLoading ? t('surfaces.contracts.refreshing') : t('surfaces.contracts.refresh')}
        </button>
      </div>

      {contractListError ? (
        <p className="fieldMessage contractMessage contractListMessage" role="alert">
          {contractListError}
        </p>
      ) : null}

      <div className="opsGroupLabel">{t('surfaces.contracts.ops.activeContracts')}</div>
      <div className="opsContractList" aria-label={t('surfaces.contracts.contractListLabel')}>
        {contracts.map((listedContract) => {
          const stateLabel = t(`contractStates.${listedContract.state}`);
          const statusTone = contractStateTone(listedContract.state);
          const isSelected = contract?.id === listedContract.id;
          return (
            <button
              aria-current={isSelected ? 'true' : undefined}
              className={isSelected ? 'opsContractRow active' : 'opsContractRow'}
              key={listedContract.id}
              onClick={() => onContractSelect(listedContract)}
              type="button"
            >
              <div>
                <span className="opsMono">{listedContract.id}</span>
                <b>{stateLabel}</b>
              </div>
              <span className={`opsPill ${statusTone}`}>{stateLabel}</span>
              <dl className="opsContractRowFields">
                <FieldRow label={t('surfaces.contracts.fields.goalId')} value={listedContract.goal_id} />
                <FieldRow label={t('surfaces.contracts.fields.repoBindingId')} value={listedContract.repo_binding_id} />
              </dl>
              <div className="opsRowMeta">
                <span>{t('surfaces.contracts.fields.updatedAt')}</span>
                <span>{formatCalmTimestamp(listedContract.updated_at, { locale: activeLocale })}</span>
              </div>
            </button>
          );
        })}
        {isListLoading && contracts.length === 0 ? (
          <div className="opsRailEmpty" role="status">{t('surfaces.contracts.listLoading')}</div>
        ) : null}
        {contractListLoadStatus === 'loaded' && contracts.length === 0 ? (
          <div className="opsRailEmpty">{t('surfaces.contracts.listEmpty')}</div>
        ) : null}
        {contractListLoadStatus === 'error' && contracts.length === 0 ? (
          <div className="opsRailEmpty">{t('surfaces.contracts.listUnavailable')}</div>
        ) : null}
      </div>

      <div className="opsGroupLabel">{t('surfaces.contracts.manualLookup')}</div>
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
        <button disabled={isLookupLoading} type="submit">
          {isLookupLoading ? t('surfaces.contracts.loading') : t('surfaces.contracts.lookupAction')}
          <span aria-hidden="true">→</span>
        </button>
      </form>
      {contractError ? <p className="fieldMessage contractMessage" role="alert">{contractError}</p> : null}
    </section>
  );
}

function repositoryFilterOptionLabel(
  context: RepositoryContextRecord,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  const emptyValue = t('repository.emptyValue');
  const repositoryName = context.repo_binding.repository_full_name || emptyValue;
  const repoBindingId = context.repo_binding.id || emptyValue;
  const projectName = context.project.display_name || context.project.slug;
  return projectName && projectName !== repositoryName
    ? `${repositoryName} · ${repoBindingId} · ${projectName}`
    : `${repositoryName} · ${repoBindingId}`;
}

function contractStateTone(state: ContractResponse['state']) {
  return state === 'approved' ? 'pass' : state === 'ready_for_approval' ? 'amber' : 'mauve';
}

const DRAFT_ARRAY_FIELDS = [
  { key: 'proposed_scope', labelKey: 'surfaces.contracts.draftFields.proposedScope' },
  { key: 'proposed_non_goals', labelKey: 'surfaces.contracts.draftFields.proposedNonGoals' },
  { key: 'proposed_constraints', labelKey: 'surfaces.contracts.draftFields.proposedConstraints' },
  { key: 'proposed_acceptance_criteria', labelKey: 'surfaces.contracts.draftFields.proposedAcceptanceCriteria' },
  { key: 'proposed_expected_checks', labelKey: 'surfaces.contracts.draftFields.proposedExpectedChecks' },
  { key: 'proposed_proof_expectations', labelKey: 'surfaces.contracts.draftFields.proposedProofExpectations' },
  { key: 'risk_hints', labelKey: 'surfaces.contracts.draftFields.riskHints' },
] as const satisfies ReadonlyArray<{
  key: keyof Pick<
    ContractDraftResponse,
    | 'proposed_scope'
    | 'proposed_non_goals'
    | 'proposed_constraints'
    | 'proposed_acceptance_criteria'
    | 'proposed_expected_checks'
    | 'proposed_proof_expectations'
    | 'risk_hints'
  >;
  labelKey: string;
}>;

function ContractSurfacePanel({
  activeLocale,
  contract,
  contractDraft,
  contractDraftError,
  contractDraftLoadStatus,
  loadStatus,
  repositoryContext,
  repositoryContextError,
  repositoryContextLoadStatus,
  t,
}: {
  activeLocale: ConsoleLocale;
  contract: ContractResponse | null;
  contractDraft: ContractDraftResponse | null;
  contractDraftError: string;
  contractDraftLoadStatus: ContractDraftLoadStatus;
  loadStatus: ContractLoadStatus;
  repositoryContext: OrganizationRepositoryContextResponse | null;
  repositoryContextError: string;
  repositoryContextLoadStatus: RepositoryContextLoadStatus;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const stateLabel = contract ? t(`contractStates.${contract.state}`) : t('surfaces.contracts.notSelected');
  const statusTone = contract?.state === 'approved' ? 'pass' : contract?.state === 'ready_for_approval' ? 'amber' : contract ? 'mauve' : 'muted';
  const summaryRows: Array<{ label: string; value: string; title?: string }> = contract ? [
    { label: t('surfaces.contracts.fields.id'), value: contract.id },
    { label: t('surfaces.contracts.fields.goalId'), value: contract.goal_id },
    { label: t('surfaces.contracts.fields.repoBindingId'), value: contract.repo_binding_id },
    {
      label: t('surfaces.contracts.fields.createdAt'),
      value: formatCalmTimestamp(contract.created_at, { locale: activeLocale }),
      title: contract.created_at,
    },
    {
      label: t('surfaces.contracts.fields.updatedAt'),
      value: formatCalmTimestamp(contract.updated_at, { locale: activeLocale }),
      title: contract.updated_at,
    },
    ...(contract.current_seed_id ? [{ label: t('surfaces.contracts.fields.seedId'), value: contract.current_seed_id }] : []),
    ...(contract.current_draft_id ? [{ label: t('surfaces.contracts.fields.draftId'), value: contract.current_draft_id }] : []),
    ...(contract.approved_snapshot_id ? [{ label: t('surfaces.contracts.fields.approvedSnapshotId'), value: contract.approved_snapshot_id }] : []),
  ] : [];

  return (
    <section className="contractSurface opsContractSurface" aria-label={t('surfaces.contracts.ariaLabel')}>
      <div className="opsContractTopbar">
        <div className="opsContractHeading">
          <span>{t('surfaces.contracts.ops.surface')}</span>
          <b>{t('surfaces.contracts.nav')}</b>
        </div>
        <div className="opsContractHeading secondary">
          <span>{t('surfaces.contracts.ops.detailMode')}</span>
          <b>{t('surfaces.contracts.ops.readOnlyDetail')}</b>
        </div>
      </div>

      <div className="opsContractLayout opsContractDetailLayout">
        <div className="opsContractCanvas">
          <section className="opsObject opsContractAggregate" aria-label={t('surfaces.contracts.selectedDetailAriaLabel')}>
            <div className="opsPanelHead">
              <div>
                <div className="opsPanelKicker">{t('surfaces.contracts.ops.selectedContract')}</div>
                <h2>{contract ? contract.id : t('surfaces.contracts.notSelected')}</h2>
              </div>
              {contract ? (
                <div className="opsPrimaryStatus" aria-label={t('surfaces.contracts.primaryStatusLabel')}>
                  <span>{t('surfaces.contracts.primaryStatusLabel')}</span>
                  <b className={statusTone}>{stateLabel}</b>
                </div>
              ) : null}
            </div>

            <div className="opsObjectBody">
              <div className="opsSectionLead">
                <span>{t('surfaces.contracts.aggregateTitle')}</span>
                <b>
                  {contract
                    ? t('surfaces.contracts.aggregateLoaded')
                    : loadStatus === 'not_found'
                      ? t('surfaces.contracts.notFoundBody')
                      : t('surfaces.contracts.emptyBody')}
                </b>
              </div>
              {contract ? (
                <dl className="opsAggregateGrid">
                  {summaryRows.map((row) => (
                    <div key={row.label}>
                      <dt>{row.label}</dt>
                      <dd title={row.title}>{row.value}</dd>
                    </div>
                  ))}
                </dl>
              ) : null}
            </div>
          </section>

          <ContractDraftDetail
            activeLocale={activeLocale}
            contract={contract}
            contractDraft={contractDraft}
            contractDraftError={contractDraftError}
            contractDraftLoadStatus={contractDraftLoadStatus}
            loadStatus={loadStatus}
            t={t}
          />
        </div>

        <aside className="opsContractSide">
          <ContractRepositoryContextPanel
            activeLocale={activeLocale}
            contract={contract}
            loadStatus={repositoryContextLoadStatus}
            repositoryContext={repositoryContext}
            repositoryContextError={repositoryContextError}
            t={t}
          />

          <section className="opsSideCard">
            <div className="opsPanelHead compact">
              <div>
                <div className="opsPanelKicker">{t('surfaces.contracts.unavailableTitle')}</div>
                <div className="opsPanelId">{t('surfaces.contracts.ops.downstreamDeferred')}</div>
              </div>
            </div>
            <div className="opsUnavailableList">
              <p>{t('surfaces.contracts.downstreamUnavailable')}</p>
            </div>
          </section>
        </aside>
      </div>
    </section>
  );
}

function ContractRepositoryContextPanel({
  activeLocale,
  contract,
  loadStatus,
  repositoryContext,
  repositoryContextError,
  t,
}: {
  activeLocale: ConsoleLocale;
  contract: ContractResponse | null;
  loadStatus: RepositoryContextLoadStatus;
  repositoryContext: OrganizationRepositoryContextResponse | null;
  repositoryContextError: string;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const emptyValue = t('repository.emptyValue');
  const contexts = repositoryContext?.contexts ?? [];
  const selectedContext = selectRepositoryContext(repositoryContext, contract);
  const selectedBindingIsMissing = Boolean(contract && repositoryContext && contexts.length > 0 && !selectedContext);
  const organizationRows: Array<{ label: string; value: string }> = repositoryContext ? [
    {
      label: t('repository.fields.organizationDisplayName'),
      value: displayMetadataValue(repositoryContext.organization.display_name, emptyValue),
    },
    {
      label: t('repository.fields.organizationSlug'),
      value: displayMetadataValue(repositoryContext.organization.slug, emptyValue),
    },
  ] : [];
  const contextRows: Array<{ label: string; value: string }> = selectedContext ? [
    {
      label: t('repository.fields.projectDisplayName'),
      value: displayMetadataValue(selectedContext.project.display_name, emptyValue),
    },
    {
      label: t('repository.fields.projectSlug'),
      value: displayMetadataValue(selectedContext.project.slug, emptyValue),
    },
    {
      label: t('repository.fields.projectState'),
      value: displayMetadataValue(selectedContext.project.state, emptyValue),
    },
    {
      label: t('repository.fields.repoBindingId'),
      value: displayMetadataValue(selectedContext.repo_binding.id, emptyValue),
    },
    {
      label: t('repository.fields.repositoryFullName'),
      value: displayMetadataValue(selectedContext.repo_binding.repository_full_name, emptyValue),
    },
    {
      label: t('repository.fields.provider'),
      value: displayMetadataValue(selectedContext.repo_binding.provider, emptyValue),
    },
    {
      label: t('repository.fields.defaultBranch'),
      value: displayMetadataValue(selectedContext.repo_binding.default_branch, emptyValue),
    },
    {
      label: t('repository.fields.workflowBaseBranch'),
      value: displayMetadataValue(selectedContext.repo_binding.workflow_base_branch, emptyValue),
    },
    {
      label: t('repository.fields.pathScope'),
      value: displayMetadataValue(selectedContext.repo_binding.path_scope, emptyValue),
    },
    {
      label: t('repository.fields.accessMode'),
      value: displayMetadataValue(selectedContext.repo_binding.access_mode, emptyValue),
    },
    {
      label: t('repository.fields.repoBindingState'),
      value: displayMetadataValue(selectedContext.repo_binding.state, emptyValue),
    },
    {
      label: t('repository.fields.updatedAt'),
      value: formatCalmTimestamp(selectedContext.repo_binding.updated_at, { locale: activeLocale }),
    },
  ] : [];
  const selectionCopy = selectedContext
    ? contract
      ? t('surfaces.contracts.repositoryContext.matchedSelection')
      : t('surfaces.contracts.repositoryContext.firstSelection')
    : '';

  return (
    <section className="opsSideCard opsRepositoryContext" aria-label={t('surfaces.contracts.repositoryContext.ariaLabel')}>
      <div className="opsPanelHead compact">
        <div>
          <div className="opsPanelKicker">{t('surfaces.contracts.repositoryContext.kicker')}</div>
          <div className="opsPanelId">{t('surfaces.contracts.repositoryContext.title')}</div>
        </div>
        <span className="opsPill muted">{t('surfaces.contracts.repositoryContext.metadataBadge')}</span>
      </div>

      <p className="opsContextDisclaimer">{t('surfaces.contracts.repositoryContext.metadataOnly')}</p>

      {loadStatus === 'loading' && !repositoryContext ? (
        <div className="opsContextEmpty" role="status">
          <span>{t('surfaces.contracts.repositoryContext.loadingTitle')}</span>
          <p>{t('repository.loading')}</p>
        </div>
      ) : null}

      {loadStatus === 'error' && !repositoryContext ? (
        <div className="opsContextEmpty" role="alert">
          <span>{t('surfaces.contracts.repositoryContext.errorTitle')}</span>
          <p>{repositoryContextError || t('repository.errors.generic')}</p>
        </div>
      ) : null}

      {repositoryContext ? (
        <div className="opsContextBlock">
          <span>{t('repository.organization')}</span>
          <dl className="opsKeyGrid">
            {organizationRows.map((row) => (
              <FieldRow key={row.label} label={row.label} value={row.value} />
            ))}
          </dl>
        </div>
      ) : null}

      {loadStatus === 'loaded' && contexts.length === 0 ? (
        <div className="opsContextEmpty">
          <span>{t('surfaces.contracts.repositoryContext.emptyTitle')}</span>
          <p>{t('surfaces.contracts.repositoryContext.emptyCopy')}</p>
        </div>
      ) : null}

      {selectedBindingIsMissing ? (
        <div className="opsContextEmpty">
          <span>{t('surfaces.contracts.repositoryContext.missingBindingTitle')}</span>
          <p>{t('surfaces.contracts.repositoryContext.missingBindingCopy', { repoBindingId: contract?.repo_binding_id })}</p>
        </div>
      ) : null}

      {selectedContext ? (
        <div className="opsContextBlock">
          <span>{t('surfaces.contracts.repositoryContext.currentContext')}</span>
          <p className="opsContextSelection">{selectionCopy}</p>
          <dl className="opsKeyGrid">
            {contextRows.map((row) => (
              <FieldRow key={row.label} label={row.label} value={row.value} />
            ))}
          </dl>
        </div>
      ) : null}
    </section>
  );
}

function selectRepositoryContext(
  repositoryContext: OrganizationRepositoryContextResponse | null,
  contract: ContractResponse | null
) {
  const contexts = repositoryContext?.contexts ?? [];
  if (contexts.length === 0) {
    return null;
  }
  if (!contract) {
    return contexts[0];
  }

  return contexts.find((context) => context.repo_binding.id === contract.repo_binding_id) ?? null;
}

function displayMetadataValue(value: string | undefined, emptyValue: string) {
  const trimmed = value?.trim();
  return trimmed ? trimmed : emptyValue;
}

function ContractDraftDetail({
  activeLocale,
  contract,
  contractDraft,
  contractDraftError,
  contractDraftLoadStatus,
  loadStatus,
  t,
}: {
  activeLocale: ConsoleLocale;
  contract: ContractResponse | null;
  contractDraft: ContractDraftResponse | null;
  contractDraftError: string;
  contractDraftLoadStatus: ContractDraftLoadStatus;
  loadStatus: ContractLoadStatus;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const draftStateLabel = contractDraft ? draftStateDisplay(contractDraft.state, t) : t('surfaces.contracts.ops.none');
  const draftRows: Array<{ label: string; value: string }> = contractDraft ? [
    { label: t('surfaces.contracts.fields.draftId'), value: contractDraft.id },
    { label: t('surfaces.contracts.fields.state'), value: draftStateLabel },
    { label: t('surfaces.contracts.fields.createdAt'), value: formatCalmTimestamp(contractDraft.created_at, { locale: activeLocale }) },
  ] : [];

  return (
    <section className="opsObject opsContractDraft" aria-label={t('surfaces.contracts.currentDraftAriaLabel')}>
      <div className="opsPanelHead">
        <div>
          <div className="opsPanelKicker">{t('surfaces.contracts.currentDraftTitle')}</div>
          <h2>{contractDraft?.title || t('surfaces.contracts.currentDraftFallbackTitle')}</h2>
        </div>
        {contractDraft ? (
          <div className="opsPrimaryStatus" aria-label={t('surfaces.contracts.draftStateLabel')}>
            <span>{t('surfaces.contracts.draftStateLabel')}</span>
            <b className={contractDraft.state === 'ready_for_approval' ? 'amber' : 'mauve'}>{draftStateLabel}</b>
          </div>
        ) : null}
      </div>

      <div className="opsObjectBody">
        {contractDraft ? (
          <>
            <div className="opsSectionLead">
              <span>{t('surfaces.contracts.draftFields.intentSummary')}</span>
              <b>{contractDraft.intent_summary || t('surfaces.contracts.sectionPending')}</b>
            </div>

            <dl className="opsAggregateGrid opsDraftMetadata">
              {draftRows.map((row) => (
                <div key={row.label}>
                  <dt>{row.label}</dt>
                  <dd>{row.value}</dd>
                </div>
              ))}
            </dl>

            <div className="opsDraftSectionGrid">
              {DRAFT_ARRAY_FIELDS.map((field) => (
                <DraftArraySection
                  emptyLabel={t('surfaces.contracts.sectionPending')}
                  key={field.key}
                  title={t(field.labelKey)}
                  values={contractDraft[field.key]}
                />
              ))}
              <DraftSourceRefsSection
                emptyLabel={t('surfaces.contracts.sectionPending')}
                refs={contractDraft.source_refs}
                title={t('surfaces.contracts.draftFields.sourceRefs')}
              />
            </div>
          </>
        ) : (
          <ContractDraftEmptyState
            contract={contract}
            contractDraftError={contractDraftError}
            contractDraftLoadStatus={contractDraftLoadStatus}
            loadStatus={loadStatus}
            t={t}
          />
        )}
      </div>
    </section>
  );
}

function ContractDraftEmptyState({
  contract,
  contractDraftError,
  contractDraftLoadStatus,
  loadStatus,
  t,
}: {
  contract: ContractResponse | null;
  contractDraftError: string;
  contractDraftLoadStatus: ContractDraftLoadStatus;
  loadStatus: ContractLoadStatus;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const message =
    !contract
      ? loadStatus === 'not_found'
        ? t('surfaces.contracts.notFoundBody')
        : t('surfaces.contracts.emptyBody')
      : contractDraftLoadStatus === 'loading'
        ? t('surfaces.contracts.draftLoading')
        : contractDraftLoadStatus === 'no_draft'
          ? t('surfaces.contracts.noCurrentDraft')
          : contractDraftLoadStatus === 'unavailable'
            ? contractDraftError || t('surfaces.contracts.draftUnavailable')
            : contractDraftLoadStatus === 'error'
              ? contractDraftError
              : t('surfaces.contracts.draftWaiting');

  return (
    <div className="opsDraftEmpty" role={contractDraftLoadStatus === 'loading' ? 'status' : undefined}>
      <span>{t('surfaces.contracts.currentDraftTitle')}</span>
      <p>{message}</p>
    </div>
  );
}

function DraftArraySection({
  emptyLabel,
  title,
  values,
}: {
  emptyLabel: string;
  title: string;
  values: string[];
}) {
  const visibleValues = Array.isArray(values) ? values : [];
  return (
    <section className="opsDraftSection">
      <h3>{title}</h3>
      {visibleValues.length > 0 ? (
        <ul>
          {visibleValues.map((value, index) => (
            <li key={`${title}-${index}`}>{value}</li>
          ))}
        </ul>
      ) : (
        <p>{emptyLabel}</p>
      )}
    </section>
  );
}

function DraftSourceRefsSection({
  emptyLabel,
  refs,
  title,
}: {
  emptyLabel: string;
  refs: ContractDraftSourceRef[];
  title: string;
}) {
  const visibleRefs = Array.isArray(refs) ? refs : [];
  return (
    <section className="opsDraftSection">
      <h3>{title}</h3>
      {visibleRefs.length > 0 ? (
        <ul>
          {visibleRefs.map((ref, index) => (
            <li key={`${ref.kind}-${ref.id}-${index}`}>
              <span className="opsSourceRef">
                <b>{ref.kind}</b>
                <code>{ref.id}</code>
                {sourceRefExtras(ref).map(([key, value]) => (
                  <small key={key}>{key}: {String(value)}</small>
                ))}
              </span>
            </li>
          ))}
        </ul>
      ) : (
        <p>{emptyLabel}</p>
      )}
    </section>
  );
}

function sourceRefExtras(ref: ContractDraftSourceRef) {
  return Object.entries(ref).filter(([key, value]) => key !== 'kind' && key !== 'id' && value !== undefined && value !== null);
}

function draftStateDisplay(state: string, t: (key: string, options?: Record<string, unknown>) => string) {
  const translated = t(`contractStates.${state}`);
  return translated === `contractStates.${state}` ? state : translated;
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

function QualificationFeedSurfacePanel({
  feed,
  loadStatus,
  locale,
  onOpenContract,
  onRetry,
  qualificationFeedError,
  t,
}: {
  feed: QualificationFeedResponse;
  loadStatus: QualificationFeedLoadStatus;
  locale: ConsoleLocale;
  onOpenContract: (contractId: string) => void;
  onRetry: () => void;
  qualificationFeedError: string;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const items = feed.items;
  const itemsByLane = READINESS_DISPLAY_LANES.map((lane) => ({
    lane,
    items: sortReadinessItems(items.filter((item) => qualificationBoardLane(item) === lane)),
  }));

  return (
    <section className="qualificationSurface" aria-label={t('qualificationFeed.ariaLabel')}>
      <header className="qualificationHeader">
        <div>
          <p className="kicker">{t('qualificationFeed.kicker')}</p>
          <h2>{t('qualificationFeed.title')}</h2>
          <p>{t('qualificationFeed.copy')}</p>
        </div>
        <div className="qualificationStats" aria-label={t('qualificationFeed.stats')}>
          {itemsByLane.map(({ lane, items: laneItems }) => (
            <span key={lane}>
              <b>{laneItems.length}</b>
              {t(`qualificationFeed.lanes.${lane}`)}
            </span>
          ))}
          <button className="ghostButton" onClick={onRetry} type="button">
            {t('qualificationFeed.refresh')}
          </button>
        </div>
      </header>

      {loadStatus === 'loading' ? <p className="usersNotice" role="status">{t('qualificationFeed.loading')}</p> : null}

      {loadStatus === 'error' ? (
        <div className="usersNotice usersNoticeError" role="alert">
          <p>{qualificationFeedError}</p>
          <button className="ghostButton" onClick={onRetry} type="button">
            {t('qualificationFeed.retry')}
          </button>
        </div>
      ) : null}

      {loadStatus === 'loaded' && items.length === 0 ? (
        <section className="qualificationEmpty" aria-label={t('qualificationFeed.emptyTitle')}>
          <h3>{t('qualificationFeed.emptyTitle')}</h3>
          <p>{t('qualificationFeed.emptyCopy')}</p>
        </section>
      ) : null}

      <div className="qualificationLaneBoard" aria-label={t('qualificationFeed.boardLabel')}>
        {itemsByLane.map(({ lane, items: laneItems }) => (
          <section className="qualificationLane" aria-label={t(`qualificationFeed.laneLabels.${lane}`)} key={lane}>
            <header>
              <h3>{t(`qualificationFeed.lanes.${lane}`)}</h3>
              <span>{laneItems.length}</span>
            </header>
            <div className="qualificationCards">
              {laneItems.map((item) => (
                <QualificationFeedCard
                  item={item}
                  key={`${item.intake_id}-${item.goal_id}`}
                  locale={locale}
                  onOpenContract={onOpenContract}
                  t={t}
                />
              ))}
              {loadStatus === 'loaded' && laneItems.length === 0 ? (
                <div className="qualificationLaneEmpty">{t('qualificationFeed.emptyLane')}</div>
              ) : null}
            </div>
          </section>
        ))}
      </div>
    </section>
  );
}

function QualificationFeedCard({
  item,
  locale,
  onOpenContract,
  t,
}: {
  item: QualificationFeedItem;
  locale: ConsoleLocale;
  onOpenContract: (contractId: string) => void;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const display = projectReadinessDisplay(item);
  const questionCount = item.open_clarification_request?.questions.length ?? 0;
  const readinessReasonCodes = item.readiness.reason_codes ?? [];
  const contractState = item.linked_contract ? t(`contractStates.${item.linked_contract.state}`) : t('qualificationFeed.noContract');
  const contractValue = item.linked_contract
    ? `${contractState} · ${item.linked_contract.id}`
    : contractState;
  const createdAtLabel = formatCalmTimestamp(item.created_at, { locale });

  return (
    <article className={`qualificationCard ${display.tone}`}>
      <header>
        <div>
          <span className="opsMono">{item.goal_id}</span>
          <h4>{item.title}</h4>
        </div>
        <span className={`opsPill ${display.tone}`}>{t(`qualificationFeed.primaryStatuses.${display.primaryStatusKey}`)}</span>
      </header>

      <dl className="qualificationCardFields">
        <FieldRow label={t('qualificationFeed.fields.repository')} value={item.repository_full_name || t('qualificationFeed.emptyValue')} />
        <FieldRow label={t('qualificationFeed.fields.repoBinding')} value={item.repo_binding_id} />
        <FieldRow label={t('qualificationFeed.fields.goalState')} value={t(`qualificationFeed.goalStates.${item.goal_state}`)} />
        <FieldRow label={t('qualificationFeed.fields.feedLane')} value={t(`qualificationFeed.itemLanes.${item.lane}`)} />
        <FieldRow label={t('qualificationFeed.fields.contract')} value={contractValue} />
      </dl>

      <div className="qualificationReasonList" aria-label={t('qualificationFeed.reasonCodes')}>
        {readinessReasonCodes.length > 0 ? (
          readinessReasonCodes.map((reasonCode) => <span key={reasonCode}>{reasonCode}</span>)
        ) : (
          <span>{t('qualificationFeed.noReasonCodes')}</span>
        )}
      </div>

      <footer className="qualificationCardFooter">
        <span>{t('qualificationFeed.questionsCount', { count: questionCount })}</span>
        <span title={item.created_at}>{createdAtLabel}</span>
      </footer>

      {item.open_clarification_request ? (
        <div className="qualificationQuestionList" aria-label={t('qualificationFeed.questions')}>
          <span>{t('qualificationFeed.questions')}</span>
          {item.open_clarification_request.questions.map((question) => (
            <div className="qualificationQuestion" key={question.id}>
              <b>{question.text}</b>
              <small>{question.why_needed || question.maps_to}</small>
            </div>
          ))}
        </div>
      ) : null}

      {item.linked_contract ? (
        <button
          className="qualificationNavigationButton"
          onClick={() => onOpenContract(item.linked_contract?.id ?? '')}
          type="button"
        >
          {t('qualificationFeed.openContract')}
        </button>
      ) : null}
    </article>
  );
}

function qualificationBoardLane(item: QualificationFeedItem): (typeof READINESS_DISPLAY_LANES)[number] {
  return item.lane;
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

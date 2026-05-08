import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
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

function contractDraftResponse(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: '018f0000-0000-7000-8000-000000000104',
    contract_id: '018f0000-0000-7000-8000-000000000101',
    contract_seed_id: '018f0000-0000-7000-8000-000000000103',
    goal_id: '018f0000-0000-7000-8000-000000000102',
    repo_binding_id: '018f0000-0000-7000-8000-000000000004',
    title: 'Render selected contract current draft',
    intent_summary: 'Show the draft body from the read-only current draft endpoint.',
    proposed_scope: ['Render draft title and body'],
    proposed_non_goals: ['Do not add lifecycle mutation controls'],
    proposed_constraints: ['Keep the Console read-only'],
    proposed_acceptance_criteria: ['Draft fields render in selected detail'],
    proposed_expected_checks: ['npm --prefix apps/web/console run test'],
    proposed_proof_expectations: ['Validation commands pass'],
    risk_hints: ['Keep aggregate detail visible on draft errors'],
    source_refs: [{ kind: 'contract_seed', id: '018f0000-0000-7000-8000-000000000103' }],
    state: 'draft',
    created_at: '2026-05-08T10:00:00Z',
    ...overrides,
  };
}

function draftResponseForContract(contractRecord: unknown, overrides: Partial<Record<string, unknown>> = {}) {
  const record = contractRecord as Record<string, unknown>;
  return contractDraftResponse({
    id: String(record.current_draft_id ?? '018f0000-0000-7000-8000-000000000104'),
    contract_id: String(record.id ?? '018f0000-0000-7000-8000-000000000101'),
    contract_seed_id: String(record.current_seed_id ?? '018f0000-0000-7000-8000-000000000103'),
    goal_id: String(record.goal_id ?? '018f0000-0000-7000-8000-000000000102'),
    repo_binding_id: String(record.repo_binding_id ?? '018f0000-0000-7000-8000-000000000004'),
    ...overrides,
  });
}

function contractListResponse(contracts: unknown[] = []) {
  return {
    contracts,
    limit: 50,
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

function qualificationFeedItem(overrides: Partial<Record<string, unknown>> = {}) {
  const goalId = String(overrides.goalId ?? '018f0000-0000-7000-8000-000000000202');
  const lane = String(overrides.lane ?? 'qualification');
  const goalState = String(overrides.goalState ?? 'created');
  const ready = Boolean(overrides.ready ?? goalState === 'ready_for_contract_seed');
  const linkedContract = overrides.linkedContract;
  const openClarificationRequest = overrides.openClarificationRequest;

  return {
    intake_id: String(overrides.intakeId ?? '018f0000-0000-7000-8000-000000000201'),
    goal_id: goalId,
    organization_id: '018f0000-0000-7000-8000-000000000002',
    project_id: String(overrides.projectId ?? '018f0000-0000-7000-8000-000000000003'),
    repo_binding_id: String(overrides.repoBindingId ?? '018f0000-0000-7000-8000-000000000004'),
    repository_full_name: String(overrides.repositoryFullName ?? 'heurema/goalrail'),
    title: String(overrides.title ?? 'Improve billing error handling'),
    lane,
    intake_state: 'received',
    goal_state: goalState,
    readiness: {
      ready,
      reason_codes: (overrides.reasonCodes as string[] | undefined) ?? (ready ? [] : ['missing_scope_hint']),
      source: 'goal_snapshot',
    },
    ...(openClarificationRequest === undefined ? {} : { open_clarification_request: openClarificationRequest }),
    ...(linkedContract === undefined ? {} : { linked_contract: linkedContract }),
    next_action: {
      kind: String(overrides.nextAction ?? 'continue_goal'),
      available: Boolean(overrides.nextActionAvailable ?? true),
      blocking: Boolean(overrides.nextActionBlocking ?? false),
    },
    created_at: String(overrides.createdAt ?? '2026-05-08T10:00:00Z'),
  };
}

function qualificationFeedResponse(items: unknown[] = [qualificationFeedItem()]) {
  return { items };
}

function openClarificationRequest(questionText = 'What is the intended scope at a high level?') {
  return {
    id: '018f0000-0000-7000-8000-000000000220',
    state: 'open',
    questions: [
      {
        id: '018f0000-0000-7000-8000-000000000221',
        text: questionText,
        why_needed: 'A scope hint is required before contract seed readiness.',
        answer_type: 'text',
        maps_to: 'goal.scope_hint',
      },
    ],
  };
}

function openIntentOwnerClarificationRequest() {
  return {
    id: '018f0000-0000-7000-8000-000000000230',
    state: 'open',
    questions: [
      {
        id: '018f0000-0000-7000-8000-000000000231',
        text: 'Who owns this intent?',
        why_needed: 'An intent owner is required before contract seed readiness.',
        answer_type: 'text',
        maps_to: 'goal.intent_owner',
      },
    ],
  };
}

function errorEnvelope(code: string, message = 'error') {
  return {
    error: { code, message },
  };
}

function fetchRequestCalls() {
  return fetchMock.mock.calls.map(([url, options]) => ({
    url: String(url),
    method: String((options as RequestInit | undefined)?.method ?? 'GET'),
    options: options as RequestInit | undefined,
  }));
}

function findFetchRequest(url: string, method = 'GET') {
  return fetchRequestCalls().find((call) => call.url === url && call.method === method);
}

function expectNoWorkflowMutationRequests() {
  const workflowMutationPatterns = [
    /\/v1\/goals\/[^/]+\/continuation$/,
    /\/v1\/clarifications\/[^/]+\/answers\/continuation$/,
    /^\/v1\/contracts$/,
    /^\/v1\/contracts\/[^/]+$/,
    /^\/v1\/contracts\/[^/]+\/(submissions|approvals|plans)$/,
    /^\/v1\/plans(?:\/|$)/,
    /^\/v1\/proposals(?:\/|$)/,
    /^\/v1\/tasks(?:\/|$)/,
    /^\/v1\/runs(?:\/|$)/,
    /^\/v1\/proofs?(?:\/|$)/,
  ];
  const calls = fetchRequestCalls();

  expect(calls.some((call) => (
    call.method !== 'GET' && workflowMutationPatterns.some((pattern) => pattern.test(call.url))
  ))).toBe(false);
}

async function setLocale(locale: 'en' | 'ru') {
  await i18n.changeLanguage(locale);
  document.documentElement.lang = locale;
}

async function loginSuccessfully(
  locale: 'en' | 'ru' = 'en',
  membershipRole = 'owner',
  contracts: unknown[] = [],
  initialDraftOverrides: Partial<Record<string, unknown>> | null = {},
  repositoryContexts: unknown[] = [repositoryContextRecord()]
) {
  await setLocale(locale);
  fetchMock.mockResolvedValueOnce(jsonResponse(loginResponse()));
  fetchMock.mockResolvedValueOnce(jsonResponse(meResponse({ role: membershipRole })));
  fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse(contracts)));
  fetchMock.mockResolvedValueOnce(jsonResponse(repositoryContextResponse(repositoryContexts)));
  const firstContract = contracts[0] as Record<string, unknown> | undefined;
  if (firstContract?.current_draft_id && initialDraftOverrides !== null) {
    fetchMock.mockResolvedValueOnce(jsonResponse(draftResponseForContract(firstContract, initialDraftOverrides)));
  }
  render(<App />);

  fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
  fireEvent.change(screen.getByLabelText(locale === 'ru' ? /^Пароль$/i : /^Password$/i), {
    target: { value: 'password' },
  });
  fireEvent.click(screen.getByRole('button', { name: locale === 'ru' ? /войти/i : /sign in/i }));

  await screen.findByRole('navigation', { name: locale === 'ru' ? /разделы продукта/i : /product surfaces/i });
  await waitFor(() => {
    expect(fetchMock.mock.calls.map(([url]) => String(url))).toContain('/v1/contracts?limit=50');
    expect(fetchMock.mock.calls.map(([url]) => String(url))).toContain('/v1/organizations/018f0000-0000-7000-8000-000000000002/repository-context');
  });
}

async function flushAsyncWork() {
  await act(async () => {
    for (let index = 0; index < 5; index += 1) {
      await Promise.resolve();
    }
  });
}

async function restartContractsPollingWithFakeTimers(
  contracts: unknown[] = [],
  detail?: unknown,
  draftOverrides: Partial<Record<string, unknown>> | null = {}
) {
  fireEvent.click(screen.getByRole('button', { name: /settings/i }));
  vi.useFakeTimers();
  fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse(contracts)));
  if (detail) {
    fetchMock.mockResolvedValueOnce(jsonResponse(detail));
    const detailRecord = detail as Record<string, unknown>;
    if (detailRecord.current_draft_id && draftOverrides !== null) {
      fetchMock.mockResolvedValueOnce(jsonResponse(draftResponseForContract(detailRecord, draftOverrides)));
    }
  }
  fireEvent.click(screen.getByRole('button', { name: /^Contracts$/i }));
  await flushAsyncWork();
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
    window.history.replaceState(null, '', '/console');
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: false,
    });
    document.title = 'Goalrail';
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

  afterEach(() => {
    vi.useRealTimers();
  });

  it('redirects the public root route to /start without backend calls or browser storage writes', async () => {
    window.history.replaceState(null, '', '/?utm_source=linkedin#ask');

    render(<App />);

    expect(screen.getByRole('heading', { name: /ask goalrail about ai-assisted delivery/i })).toBeInTheDocument();
    expect(screen.getByText('From business goal to verified code change.')).toBeInTheDocument();
    await waitFor(() =>
      expect(`${window.location.pathname}${window.location.search}${window.location.hash}`).toBe(
        '/start?utm_source=linkedin#ask'
      )
    );
    expect(fetchMock).not.toHaveBeenCalled();
    expect(window.localStorage.getItem).not.toHaveBeenCalled();
    expect(window.localStorage.setItem).not.toHaveBeenCalled();
    expect(window.sessionStorage.getItem).not.toHaveBeenCalled();
    expect(window.sessionStorage.setItem).not.toHaveBeenCalled();
  });

  it('renders the /start page without initial backend calls or browser storage writes', () => {
    window.history.replaceState(null, '', '/start');

    render(<App />);

    expect(screen.getByRole('heading', { name: /ask goalrail about ai-assisted delivery/i })).toBeInTheDocument();
    expect(screen.getByText('From business goal to verified code change.')).toBeInTheDocument();
    expect(screen.getByText(/Goalrail is a control layer for teams using AI coding tools/i)).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /ask goalrail/i })).not.toBeDisabled();
    expect(screen.getByRole('button', { name: /^Ask$/i })).toBeDisabled();
    expect(screen.getByText(/Answers use public Goalrail materials only/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'What is Goalrail?' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Is my repo ready for coding agents?' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'What is contract-first execution?' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'What does proof before approval mean?' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'How is Goalrail different from an AI IDE?' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'What would a pilot fit check look like?' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'What is AI delivery drift?' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'How should a team review AI-generated changes?' })).toBeInTheDocument();
    expect(screen.getByText('Contract-first execution')).toBeInTheDocument();
    expect(screen.getByText('Proof before approval')).toBeInTheDocument();
    expect(screen.getByText('Repo readiness')).toBeInTheDocument();
    expect(screen.getByText('AI delivery drift')).toBeInTheDocument();
    expect(screen.getByText('This page answers from public Goalrail materials only.')).toBeInTheDocument();
    expect(screen.getByText('It cannot scan your repository.')).toBeInTheDocument();
    expect(screen.getByText('It does not execute code.')).toBeInTheDocument();
    expect(screen.getByText('Do not paste secrets, private code, or customer data.')).toBeInTheDocument();
    expect(screen.getByText(/Have a real workflow where AI is making your team faster but harder to control/i)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /request a pilot fit check/i }).getAttribute('href')).toContain('mailto:');
    expect(screen.getByRole('link', { name: /view github/i })).toHaveAttribute('href', 'https://github.com/heurema/goalrail');
    expect(screen.getByRole('link', { name: /view artifacts/i })).toHaveAttribute('href', '#artifacts');
    expect(fetchMock).not.toHaveBeenCalled();
    expect(asMock(window.localStorage.getItem)).not.toHaveBeenCalled();
    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(asMock(window.sessionStorage.getItem)).not.toHaveBeenCalled();
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
  });

  it('submits a /start question to the assistant endpoint and renders sources', async () => {
    window.history.replaceState(null, '', '/start');
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        answer: 'Goalrail keeps AI-assisted delivery tied to contract, checks, proof, and approval.',
        sources: [
          {
            title: 'Goalrail Global Start Assistant',
            path: 'docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md',
            section: 'Assistant behavior',
          },
        ],
        suggested_questions: ['What is proof before approval?'],
        knowledge: {
          updated_at: '2026-05-07T12:00:00Z',
          commit_sha: 'abc123',
        },
        disclaimer: 'Answers use public Goalrail materials. This page cannot scan repos or execute code.',
      })
    );

    render(<App />);

    fireEvent.change(screen.getByRole('textbox', { name: /ask goalrail/i }), {
      target: { value: 'What is Goalrail?' },
    });
    fireEvent.click(screen.getByRole('button', { name: /^Ask$/i }));

    await screen.findByRole('heading', { name: 'Source-grounded assistant response' });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/start-chat',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question: 'What is Goalrail?' }),
      })
    );
    expect(screen.getByText(/keeps AI-assisted delivery tied to contract/i)).toBeInTheDocument();
    expect(screen.getByText('Goalrail Global Start Assistant / Assistant behavior')).toBeInTheDocument();
    expect(screen.getByText('Updated 2026-05-07T12:00:00Z')).toBeInTheDocument();
    expect(screen.getByText('Revision abc123')).toBeInTheDocument();
    expect(screen.getByText(/cannot scan repos or execute code/i)).toBeInTheDocument();
    expect(asMock(window.localStorage.setItem)).not.toHaveBeenCalled();
    expect(asMock(window.sessionStorage.setItem)).not.toHaveBeenCalled();
  });

  it('keeps /start static fallback visible when assistant endpoint is unavailable', async () => {
    window.history.replaceState(null, '', '/start');
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        error: 'assistant_unavailable',
        message: 'The public Goalrail assistant is temporarily unavailable. Static overview and artifacts are still available.',
      }, 503)
    );

    render(<App />);

    fireEvent.change(screen.getByRole('textbox', { name: /ask goalrail/i }), {
      target: { value: 'What is Goalrail?' },
    });
    fireEvent.click(screen.getByRole('button', { name: /^Ask$/i }));

    await screen.findByText(/temporarily unavailable/i);
    expect(screen.getByRole('heading', { name: /Goalrail is a control layer/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'What is contract-first execution?' })).toBeInTheDocument();
  });

  it('keeps /start fallback message safe when assistant endpoint returns non-JSON', async () => {
    window.history.replaceState(null, '', '/start');
    fetchMock.mockResolvedValueOnce(
      new Response('<html>bad gateway</html>', {
        status: 502,
        headers: { 'Content-Type': 'text/html' },
      })
    );

    render(<App />);

    fireEvent.change(screen.getByRole('textbox', { name: /ask goalrail/i }), {
      target: { value: 'What is Goalrail?' },
    });
    fireEvent.click(screen.getByRole('button', { name: /^Ask$/i }));

    await screen.findByText(/temporarily unavailable/i);
    expect(screen.queryByText(/Unexpected token/i)).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /Goalrail is a control layer/i })).toBeInTheDocument();
  });

  it('updates the /start answer panel from local static question data', () => {
    window.history.replaceState(null, '', '/start');

    render(<App />);

    expect(screen.getByRole('heading', { name: /Goalrail is a control layer/i })).toBeInTheDocument();

    const proofQuestion = screen.getByRole('button', { name: 'What does proof before approval mean?' });
    fireEvent.click(proofQuestion);

    expect(proofQuestion).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('textbox', { name: /ask goalrail/i })).toHaveValue('What does proof before approval mean?');
    expect(screen.getByRole('heading', { name: 'Output is not proof.' })).toBeInTheDocument();
    expect(screen.getByText(/They should compare contract, diff, checks, artifacts, and remaining risk/i)).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('sets /start metadata for title, description, and Open Graph', () => {
    window.history.replaceState(null, '', '/start');

    render(<App />);

    expect(document.title).toBe('Goalrail - AI-assisted delivery without losing control');
    expect(document.head.querySelector('meta[name="description"]')).toHaveAttribute(
      'content',
      'Goalrail is a control layer for AI-assisted software delivery: from business goal to verified code change with contracts, proof, and human approval.'
    );
    expect(document.head.querySelector('meta[property="og:title"]')).toHaveAttribute(
      'content',
      'Ask Goalrail about AI-assisted delivery'
    );
    expect(document.head.querySelector('meta[property="og:description"]')).toHaveAttribute(
      'content',
      'From business goal to verified code change.'
    );
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

    expect(fetchMock).toHaveBeenCalledTimes(4);
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
    expect(fetchMock.mock.calls[2][0]).toBe('/v1/contracts?limit=50');
    expect(fetchMock.mock.calls[2][1]).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(fetchMock.mock.calls[3][0]).toBe('/v1/organizations/018f0000-0000-7000-8000-000000000002/repository-context');
    expect(fetchMock.mock.calls[3][1]).toEqual(
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
    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse()));
    fetchMock.mockResolvedValueOnce(jsonResponse(repositoryContextResponse()));

    fireEvent.change(screen.getByLabelText(/^Текущий пароль$/i), { target: { value: 'temporary-password' } });
    fireEvent.change(screen.getByLabelText(/^Новый пароль$/i), { target: { value: 'new-password' } });
    fireEvent.click(screen.getByRole('button', { name: /сменить пароль/i }));

    await screen.findByRole('navigation', { name: /разделы продукта/i });
    await waitFor(() => {
      expect(fetchMock.mock.calls.map(([url]) => String(url))).toContain('/v1/contracts?limit=50');
      expect(fetchMock.mock.calls.map(([url]) => String(url))).toContain('/v1/organizations/018f0000-0000-7000-8000-000000000002/repository-context');
    });

    expect(fetchMock).toHaveBeenCalledTimes(5);
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
    expect(fetchMock.mock.calls[3][0]).toBe('/v1/contracts?limit=50');
    expect(fetchMock.mock.calls[4][0]).toBe('/v1/organizations/018f0000-0000-7000-8000-000000000002/repository-context');
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
    expect(fetchMock).toHaveBeenCalledTimes(5);
    expect(fetchMock.mock.calls[4][0]).toBe('/v1/auth/logout');
    expect(fetchMock.mock.calls[4][1]).toEqual(
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
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      '/v1/auth/login',
      '/v1/me',
      '/v1/contracts?limit=50',
      '/v1/organizations/018f0000-0000-7000-8000-000000000002/repository-context',
      '/v1/auth/logout',
    ]);
    expect(fetchMock.mock.calls[4][1]).toEqual(
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
    expect(screen.getAllByText(/Select a contract to review the public aggregate returned by the API/i).length).toBeGreaterThan(0);
    expect(document.body).not.toHaveTextContent(
      /trialops-demo|C-0147|fake queue|fake pass|fake fail|backend available|endpoint|GET \/v1\/contracts|prefilled/i
    );

    fetchMock.mockResolvedValueOnce(jsonResponse(contractResponse()));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractDraftResponse()));
    fireEvent.change(screen.getByLabelText(/^Contract ID$/i), {
      target: { value: '018f0000-0000-7000-8000-000000000101' },
    });
    fireEvent.click(screen.getByRole('button', { name: /load contract/i }));

    await screen.findByText('Render selected contract current draft');
    const contractLookupCall = findFetchRequest('/v1/contracts/018f0000-0000-7000-8000-000000000101');
    expect(contractLookupCall?.options).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );

    fetchMock.mockResolvedValueOnce(jsonResponse(qualificationFeedResponse([])));
    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    expect(await screen.findByText('No active qualification items yet.')).toBeInTheDocument();
    expect(screen.getByLabelText(/qualification feed lane view/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Русский$/i }));

    expect(await screen.findByRole('navigation', { name: /разделы продукта/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /^Оценка готовности$/i }));
    expect(screen.getByText('No active qualification items yet.')).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent(
      /trialops-demo|C-0147|readiness score|\/100|\bscan\b|proof queue|fake queue|fake pass|fake fail|pass\/fail/i
    );
  });

  it('loads the contract list on authenticated Contracts entry and renders calm list rows', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-list-draft',
        goal_id: 'goal-list-draft',
        repo_binding_id: 'repo-list-draft',
        state: 'draft',
        current_draft_id: 'draft-list-draft',
        updated_at: '2026-05-08T10:00:00Z',
      }),
      contractResponse({
        id: 'contract-list-approved',
        goal_id: 'goal-list-approved',
        repo_binding_id: 'repo-list-approved',
        state: 'approved',
        current_draft_id: 'draft-list-approved',
        updated_at: '2026-05-07T09:10:00Z',
      }),
    ]);

    const contractList = screen.getByLabelText(/contracts list/i);
    expect(contractList).toHaveTextContent('contract-list-draft');
    expect(contractList).toHaveTextContent('goal-list-draft');
    expect(contractList).toHaveTextContent('repo-list-draft');
    expect(contractList).toHaveTextContent('Draft');
    expect(contractList).toHaveTextContent('contract-list-approved');
    expect(contractList).toHaveTextContent('Approved');
    expect(contractList).toHaveTextContent(/May|Today|Yesterday|ago|just now/);
    expect(contractList).not.toHaveTextContent(/2026-05-08T10:00:00Z|2026-05-07T09:10:00Z/);
    expect(screen.getAllByText('draft-list-draft').length).toBeGreaterThan(0);
    expect(fetchMock.mock.calls[2][0]).toBe('/v1/contracts?limit=50');
  });

  it('loads and renders repository context metadata on authenticated Contracts entry', async () => {
    await loginSuccessfully('en', 'owner', [], {}, [
      repositoryContextRecord({
        projectDisplayName: 'Goalrail Console',
        projectSlug: 'goalrail-console',
        projectState: 'active',
        repoBindingId: 'repo-context-primary',
        repositoryFullName: 'heurema/goalrail',
        provider: 'github',
        defaultBranch: 'main',
        workflowBaseBranch: 'release-base',
        pathScope: 'apps/web/console',
        accessMode: 'metadata_only',
        repoBindingState: 'active',
      }),
    ]);

    const panel = screen.getByLabelText(/repository context metadata/i);
    const repositoryContextCall = findFetchRequest('/v1/organizations/018f0000-0000-7000-8000-000000000002/repository-context');

    expect(repositoryContextCall?.options).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(panel).toHaveTextContent('Goalrail Dev');
    expect(panel).toHaveTextContent('goalrail-dev');
    expect(panel).toHaveTextContent('Goalrail Console');
    expect(panel).toHaveTextContent('goalrail-console');
    expect(panel).toHaveTextContent('active');
    expect(panel).toHaveTextContent('repo-context-primary');
    expect(panel).toHaveTextContent('heurema/goalrail');
    expect(panel).toHaveTextContent('github');
    expect(panel).toHaveTextContent('main');
    expect(panel).toHaveTextContent('release-base');
    expect(panel).toHaveTextContent('apps/web/console');
    expect(panel).toHaveTextContent('metadata_only');
    expect(panel).toHaveTextContent('No Contract is selected; showing the first repository context for this Organization.');
    expect(panel).not.toHaveTextContent('2026-05-07T10:15:00Z');
    expect(panel).toHaveTextContent('Metadata only. This does not prove provider authorization, checkout permission, readiness/proof status, execution status, or runner state.');
    expect(panel).not.toHaveTextContent(/provider connected|checkout ready|readiness score|proof ready|execution running|runner online|gate decision|task plan/i);
    expectNoWorkflowMutationRequests();
  });

  it('shows the repository context matching the selected Contract repo_binding_id', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-context-match',
        goal_id: 'goal-context-match',
        repo_binding_id: 'repo-binding-match',
        current_draft_id: undefined,
      }),
    ], {}, [
      repositoryContextRecord({
        repoBindingId: 'repo-binding-other',
        projectDisplayName: 'Other Project',
        repositoryFullName: 'heurema/other',
      }),
      repositoryContextRecord({
        repoBindingId: 'repo-binding-match',
        projectDisplayName: 'Matched Project',
        repositoryFullName: 'heurema/goalrail-console',
        workflowBaseBranch: 'contracts-base',
        pathScope: 'apps/web/console',
      }),
    ]);

    const panel = screen.getByLabelText(/repository context metadata/i);

    expect(panel).toHaveTextContent('Matched to the selected Contract repo_binding_id.');
    expect(panel).toHaveTextContent('Matched Project');
    expect(panel).toHaveTextContent('repo-binding-match');
    expect(panel).toHaveTextContent('heurema/goalrail-console');
    expect(panel).toHaveTextContent('contracts-base');
    expect(panel).not.toHaveTextContent('Other Project');
    expect(panel).not.toHaveTextContent('heurema/other');
    expect(screen.getByLabelText(/selected contract detail/i)).toHaveTextContent('contract-context-match');
    expectNoWorkflowMutationRequests();
  });

  it('keeps the selected Contract visible when its repo binding has no repository context metadata', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-context-missing',
        goal_id: 'goal-context-missing',
        repo_binding_id: 'repo-binding-missing',
        current_draft_id: undefined,
      }),
    ], {}, [
      repositoryContextRecord({
        repoBindingId: 'repo-binding-present',
        projectDisplayName: 'Present Project',
        repositoryFullName: 'heurema/present',
      }),
    ]);

    const panel = screen.getByLabelText(/repository context metadata/i);
    const detail = screen.getByLabelText(/selected contract detail/i);

    expect(detail).toHaveTextContent('contract-context-missing');
    expect(detail).toHaveTextContent('repo-binding-missing');
    expect(panel).toHaveTextContent('No repository context metadata for this repo binding');
    expect(panel).toHaveTextContent('repo_binding_id repo-binding-missing is not present');
    expect(panel).not.toHaveTextContent('Present Project');
    expect(panel).not.toHaveTextContent('heurema/present');
    expectNoWorkflowMutationRequests();
  });

  it('shows an honest Contracts repository context empty state when no contexts are returned', async () => {
    await loginSuccessfully('en', 'owner', [], {}, []);

    const panel = screen.getByLabelText(/repository context metadata/i);

    expect(panel).toHaveTextContent('No repository context metadata yet');
    expect(panel).toHaveTextContent('This Organization has no active Project / RepoBinding metadata from the repository context endpoint.');
    expect(panel).toHaveTextContent('Metadata only.');
    expectNoWorkflowMutationRequests();
  });

  it('periodically refreshes the active Contracts list through the read-only discovery endpoint', async () => {
    await loginSuccessfully();
    await restartContractsPollingWithFakeTimers();
    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse([
      contractResponse({
        id: 'contract-refreshed',
        goal_id: 'goal-refreshed',
        repo_binding_id: 'repo-refreshed',
        current_draft_id: 'draft-refreshed',
      }),
    ])));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractDraftResponse({
      id: 'draft-refreshed',
      contract_id: 'contract-refreshed',
      goal_id: 'goal-refreshed',
    })));

    await act(async () => {
      vi.advanceTimersByTime(5000);
      await Promise.resolve();
    });
    await flushAsyncWork();

    expect(screen.getAllByText('contract-refreshed').length).toBeGreaterThan(0);
    expect(fetchMock.mock.calls.map(([url]) => String(url)).filter((url) => url === '/v1/contracts?limit=50')).toHaveLength(3);
    expectNoWorkflowMutationRequests();
  });

  it('includes the active state filter in scheduled Contracts refreshes', async () => {
    await loginSuccessfully();
    await restartContractsPollingWithFakeTimers();
    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse([])));

    fireEvent.change(screen.getByLabelText(/^State$/i), {
      target: { value: 'ready_for_approval' },
    });

    await flushAsyncWork();
    expect(fetchMock.mock.calls.map(([url]) => String(url))).toContain('/v1/contracts?state=ready_for_approval&limit=50');

    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse([
      contractResponse({
        id: 'contract-filter-refresh',
        state: 'ready_for_approval',
        current_draft_id: 'draft-filter-refresh',
      }),
    ])));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractDraftResponse({
      id: 'draft-filter-refresh',
      contract_id: 'contract-filter-refresh',
    })));

    await act(async () => {
      vi.advanceTimersByTime(5000);
      await Promise.resolve();
    });
    await flushAsyncWork();

    expect(screen.getAllByText('contract-filter-refresh').length).toBeGreaterThan(0);
    expect(
      fetchMock.mock.calls
        .map(([url]) => String(url))
        .filter((url) => url === '/v1/contracts?state=ready_for_approval&limit=50')
    ).toHaveLength(2);
    expectNoWorkflowMutationRequests();
  });

  it('skips scheduled Contracts refresh while hidden and refreshes once when visible again', async () => {
    await loginSuccessfully();
    await restartContractsPollingWithFakeTimers();

    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: true,
    });
    await act(async () => {
      vi.advanceTimersByTime(5000);
      await Promise.resolve();
    });

    expect(fetchMock.mock.calls.map(([url]) => String(url)).filter((url) => url === '/v1/contracts?limit=50')).toHaveLength(2);

    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse([
      contractResponse({
        id: 'contract-visible-again',
        goal_id: 'goal-visible-again',
        current_draft_id: 'draft-visible-again',
      }),
    ])));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractDraftResponse({
      id: 'draft-visible-again',
      contract_id: 'contract-visible-again',
      goal_id: 'goal-visible-again',
    })));
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: false,
    });
    document.dispatchEvent(new Event('visibilitychange'));

    await act(async () => {
      await Promise.resolve();
    });
    await flushAsyncWork();

    expect(screen.getAllByText('contract-visible-again').length).toBeGreaterThan(0);
    expect(fetchMock.mock.calls.map(([url]) => String(url)).filter((url) => url === '/v1/contracts?limit=50')).toHaveLength(3);
    expectNoWorkflowMutationRequests();
  });

  it('renders selected contract detail with the current draft body as read-only content', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-ready-detail',
        goal_id: 'goal-ready-detail',
        repo_binding_id: 'repo-ready-detail',
        state: 'ready_for_approval',
        current_seed_id: 'seed-ready-detail',
        current_draft_id: 'draft-ready-detail',
        approved_snapshot_id: 'approved-ready-detail',
        created_at: '2024-01-02T03:04:05Z',
        updated_at: '2024-01-03T04:05:06Z',
      }),
    ], {
      id: 'draft-ready-detail',
      contract_id: 'contract-ready-detail',
      contract_seed_id: 'seed-ready-detail',
      goal_id: 'goal-ready-detail',
      repo_binding_id: 'repo-ready-detail',
      title: 'Draft title for ready detail',
      intent_summary: 'Intent summary for the read-only selected draft.',
      proposed_scope: ['Render proposed scope item'],
      proposed_non_goals: ['Do not add approve controls'],
      proposed_constraints: ['Keep browser state read-only'],
      proposed_acceptance_criteria: ['Acceptance criterion is visible'],
      proposed_expected_checks: ['Expected check is visible'],
      proposed_proof_expectations: ['Proof expectation is visible'],
      risk_hints: ['Risk hint is visible'],
      source_refs: [{ kind: 'contract_seed', id: 'seed-ready-detail' }],
      state: 'ready_for_approval',
      created_at: '2024-01-04T05:06:07Z',
    });

    const detail = screen.getByLabelText(/selected contract detail/i);
    const workspace = screen.getByLabelText(/contract workspace/i);
    const primaryStatus = within(detail).getByLabelText(/lifecycle status/i);
    await screen.findByText('Draft title for ready detail');
    const draftDetail = screen.getByLabelText(/current draft detail/i);

    expect(primaryStatus).toHaveTextContent('Ready for approval');
    expect(within(detail).getAllByText('Ready for approval')).toHaveLength(1);
    expect(detail).toHaveTextContent('contract-ready-detail');
    expect(detail).toHaveTextContent('goal-ready-detail');
    expect(detail).toHaveTextContent('repo-ready-detail');
    expect(detail).toHaveTextContent('seed-ready-detail');
    expect(detail).toHaveTextContent('draft-ready-detail');
    expect(detail).toHaveTextContent('approved-ready-detail');
    expect(detail).toHaveTextContent('2 Jan');
    expect(detail).toHaveTextContent('3 Jan');
    expect(detail).not.toHaveTextContent(/2024-01-02T03:04:05Z|2024-01-03T04:05:06Z|03:04:05|04:05:06/);
    expect(draftDetail).toHaveTextContent('Draft title for ready detail');
    expect(draftDetail).toHaveTextContent('Intent summary for the read-only selected draft.');
    expect(draftDetail).toHaveTextContent('Render proposed scope item');
    expect(draftDetail).toHaveTextContent('Do not add approve controls');
    expect(draftDetail).toHaveTextContent('Keep browser state read-only');
    expect(draftDetail).toHaveTextContent('Acceptance criterion is visible');
    expect(draftDetail).toHaveTextContent('Expected check is visible');
    expect(draftDetail).toHaveTextContent('Proof expectation is visible');
    expect(draftDetail).toHaveTextContent('Risk hint is visible');
    expect(draftDetail).toHaveTextContent('contract_seed');
    expect(draftDetail).toHaveTextContent('seed-ready-detail');
    expect(draftDetail).toHaveTextContent('4 Jan');
    expect(draftDetail).not.toHaveTextContent(/2024-01-04T05:06:07Z|05:06:07/);
    expect(fetchMock.mock.calls.map(([url]) => String(url))).toContain('/v1/contracts/contract-ready-detail/current-draft');
    expect(workspace).toHaveTextContent('Task, execution, gate, and proof data are not available in this Console view yet.');
    expect(workspace).not.toHaveTextContent(
      /Execution evidence|Work items|Stage controls|Record activity|Active stage|Queued proof|Active execution|Runner online|Runner active|Task plan|Gate decision|Contract state meters/i
    );
  });

  it('does not call current draft detail when the selected Contract has no current_draft_id', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-without-draft',
        goal_id: 'goal-without-draft',
        current_draft_id: undefined,
      }),
    ]);

    expect(await screen.findByText('No current draft is linked yet.')).toBeInTheDocument();
    expect(screen.getAllByText('contract-without-draft').length).toBeGreaterThan(0);
    expect(fetchMock.mock.calls.map(([url]) => String(url)).some((url) => url.includes('/current-draft'))).toBe(false);
  });

  it('selecting a contract row loads selected detail through the read-only contract endpoint', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-list-draft',
        goal_id: 'goal-list-draft',
        current_draft_id: 'draft-list-draft',
      }),
      contractResponse({
        id: 'contract-list-approved',
        goal_id: 'goal-list-approved',
        state: 'approved',
        current_draft_id: 'draft-list-approved-summary',
      }),
    ]);
    fetchMock.mockResolvedValueOnce(jsonResponse(contractResponse({
      id: 'contract-list-approved',
      goal_id: 'goal-list-approved',
      state: 'approved',
      current_draft_id: 'draft-list-approved-detail',
    })));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractDraftResponse({
      id: 'draft-list-approved-detail',
      contract_id: 'contract-list-approved',
      goal_id: 'goal-list-approved',
      title: 'Approved row draft title',
      intent_summary: 'Approved row draft intent summary.',
    })));

    const contractList = screen.getByLabelText(/contracts list/i);
    fireEvent.click(within(contractList).getByRole('button', { name: /contract-list-approved/i }));

    expect(await screen.findByText('Approved row draft title')).toBeInTheDocument();
    const selectedDetailCall = fetchMock.mock.calls.find(([url]) => String(url) === '/v1/contracts/contract-list-approved');
    const selectedDraftCall = fetchMock.mock.calls.find(([url]) => String(url) === '/v1/contracts/contract-list-approved/current-draft');
    expect(selectedDetailCall).toBeDefined();
    expect(selectedDetailCall?.[1]).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(selectedDraftCall).toBeDefined();
    expect(fetchMock.mock.calls.map(([url]) => String(url)).some((url) => /\/v1\/contracts\/contract-list-approved\/(submissions|approvals|plans)/.test(url))).toBe(false);
  });

  it('row selection shows contract-specific access errors without calling mutation endpoints', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-list-draft',
        current_draft_id: undefined,
      }),
      contractResponse({
        id: 'contract-list-forbidden',
        state: 'approved',
        current_draft_id: undefined,
      }),
    ]);
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('forbidden'), 403));

    const contractList = screen.getByLabelText(/contracts list/i);
    fireEvent.click(within(contractList).getByRole('button', { name: /contract-list-forbidden/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('You do not have access to this contract.');
    expect(screen.queryByText('Contract was not found.')).not.toBeInTheDocument();
    expectNoWorkflowMutationRequests();
  });

  it.each([
    ['invalid_state', 409],
    ['not_found', 404],
  ])('shows an honest unavailable draft message for %s without clearing aggregate detail', async (code, status) => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(contractResponse({
      id: `contract-draft-${code}`,
      goal_id: `goal-draft-${code}`,
      repo_binding_id: `repo-draft-${code}`,
      current_draft_id: `draft-${code}`,
    })));
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope(code), status));

    fireEvent.change(screen.getByLabelText(/^Contract ID$/i), {
      target: { value: `contract-draft-${code}` },
    });
    fireEvent.click(screen.getByRole('button', { name: /load contract/i }));

    expect(await screen.findByText('Current draft is unavailable for this Contract.')).toBeInTheDocument();
    const detail = screen.getByLabelText(/selected contract detail/i);
    expect(detail).toHaveTextContent(`contract-draft-${code}`);
    expect(detail).toHaveTextContent(`goal-draft-${code}`);
    expect(detail).toHaveTextContent(`repo-draft-${code}`);
  });

  it('periodically refreshes selected Contract detail through the read-only detail endpoint', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-selected-refresh',
        goal_id: 'goal-selected-refresh',
        current_draft_id: 'draft-before-refresh',
      }),
    ]);
    await restartContractsPollingWithFakeTimers([
      contractResponse({
        id: 'contract-selected-refresh',
        goal_id: 'goal-selected-refresh',
        current_draft_id: 'draft-before-refresh',
      }),
    ], contractResponse({
      id: 'contract-selected-refresh',
      goal_id: 'goal-selected-refresh',
      current_draft_id: 'draft-before-refresh',
    }));
    expect(screen.getAllByText('draft-before-refresh').length).toBeGreaterThan(0);
    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse([
      contractResponse({
        id: 'contract-selected-refresh',
        goal_id: 'goal-selected-refresh',
        current_draft_id: 'draft-list-refresh',
      }),
    ])));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractResponse({
      id: 'contract-selected-refresh',
      goal_id: 'goal-selected-refresh',
      current_draft_id: 'draft-after-refresh',
    })));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractDraftResponse({
      id: 'draft-after-refresh',
      contract_id: 'contract-selected-refresh',
      goal_id: 'goal-selected-refresh',
      title: 'Draft after scheduled refresh',
      intent_summary: 'Scheduled refresh fetched the current draft body.',
    })));

    await act(async () => {
      vi.advanceTimersByTime(5000);
      await Promise.resolve();
    });
    await flushAsyncWork();

    expect(screen.getByText('Draft after scheduled refresh')).toBeInTheDocument();
    expect(fetchMock.mock.calls.map(([url]) => String(url))).toContain('/v1/contracts/contract-selected-refresh');
    expect(fetchMock.mock.calls.map(([url]) => String(url))).toContain('/v1/contracts/contract-selected-refresh/current-draft');
    expectNoWorkflowMutationRequests();
  });

  it('keeps selected aggregate detail on transient scheduled detail errors', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-detail-stable',
        goal_id: 'goal-detail-stable',
        current_draft_id: 'draft-detail-stable',
      }),
    ]);
    await restartContractsPollingWithFakeTimers([
      contractResponse({
        id: 'contract-detail-stable',
        goal_id: 'goal-detail-stable',
        current_draft_id: 'draft-detail-stable',
      }),
    ], contractResponse({
      id: 'contract-detail-stable',
      goal_id: 'goal-detail-stable',
      current_draft_id: 'draft-detail-stable',
    }));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse([
      contractResponse({
        id: 'contract-detail-stable',
        goal_id: 'goal-detail-stable',
        current_draft_id: 'draft-detail-stable',
      }),
    ])));
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('server_error'), 503));

    await act(async () => {
      vi.advanceTimersByTime(5000);
      await Promise.resolve();
    });
    await flushAsyncWork();

    expect(screen.getAllByText('draft-detail-stable').length).toBeGreaterThan(0);
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expectNoWorkflowMutationRequests();
  });

  it('shows access-specific errors when scheduled selected detail refresh loses organization access', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-detail-membership',
        goal_id: 'goal-detail-membership',
        current_draft_id: 'draft-detail-membership',
      }),
    ]);
    await restartContractsPollingWithFakeTimers([
      contractResponse({
        id: 'contract-detail-membership',
        goal_id: 'goal-detail-membership',
        current_draft_id: 'draft-detail-membership',
      }),
    ], contractResponse({
      id: 'contract-detail-membership',
      goal_id: 'goal-detail-membership',
      current_draft_id: 'draft-detail-membership',
    }));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse([
      contractResponse({
        id: 'contract-detail-membership',
        goal_id: 'goal-detail-membership',
        current_draft_id: 'draft-detail-membership',
      }),
    ])));
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('membership_required'), 403));

    await act(async () => {
      vi.advanceTimersByTime(5000);
      await Promise.resolve();
    });
    await flushAsyncWork();

    expect(screen.getByRole('alert')).toHaveTextContent('This contract requires active organization access.');
    expect(screen.queryByText('Contract was not found.')).not.toBeInTheDocument();
    expectNoWorkflowMutationRequests();
  });

  it('keeps the visible current draft body on transient scheduled draft refresh errors', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-draft-stable',
        goal_id: 'goal-draft-stable',
        current_draft_id: 'draft-body-stable',
      }),
    ], {
      id: 'draft-body-stable',
      contract_id: 'contract-draft-stable',
      goal_id: 'goal-draft-stable',
      title: 'Stable draft body title',
      intent_summary: 'Stable draft body intent summary.',
    });
    await restartContractsPollingWithFakeTimers([
      contractResponse({
        id: 'contract-draft-stable',
        goal_id: 'goal-draft-stable',
        current_draft_id: 'draft-body-stable',
      }),
    ], contractResponse({
      id: 'contract-draft-stable',
      goal_id: 'goal-draft-stable',
      current_draft_id: 'draft-body-stable',
    }), {
      id: 'draft-body-stable',
      contract_id: 'contract-draft-stable',
      goal_id: 'goal-draft-stable',
      title: 'Stable draft body title',
      intent_summary: 'Stable draft body intent summary.',
    });
    expect(screen.getByText('Stable draft body title')).toBeInTheDocument();
    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse([
      contractResponse({
        id: 'contract-draft-stable',
        goal_id: 'goal-draft-stable',
        current_draft_id: 'draft-body-stable',
      }),
    ])));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractResponse({
      id: 'contract-draft-stable',
      goal_id: 'goal-draft-stable',
      current_draft_id: 'draft-body-stable',
    })));
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('server_error'), 503));

    await act(async () => {
      vi.advanceTimersByTime(5000);
      await Promise.resolve();
    });
    await flushAsyncWork();

    expect(screen.getByText('Stable draft body title')).toBeInTheDocument();
    expect(screen.getByText('Stable draft body intent summary.')).toBeInTheDocument();
    expect(screen.queryByText('Goalrail server returned an error. Code: 503.')).not.toBeInTheDocument();
    expectNoWorkflowMutationRequests();
  });

  it('keeps not_found behavior when a scheduled selected detail refresh returns not_found', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-detail-missing',
        goal_id: 'goal-detail-missing',
        current_draft_id: 'draft-detail-missing',
      }),
    ]);
    await restartContractsPollingWithFakeTimers([
      contractResponse({
        id: 'contract-detail-missing',
        goal_id: 'goal-detail-missing',
        current_draft_id: 'draft-detail-missing',
      }),
    ], contractResponse({
      id: 'contract-detail-missing',
      goal_id: 'goal-detail-missing',
      current_draft_id: 'draft-detail-missing',
    }));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse([
      contractResponse({
        id: 'contract-detail-missing',
        goal_id: 'goal-detail-missing',
        current_draft_id: 'draft-detail-missing',
      }),
    ])));
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('not_found'), 404));

    await act(async () => {
      vi.advanceTimersByTime(5000);
      await Promise.resolve();
    });
    await flushAsyncWork();

    expect(screen.getByRole('alert')).toHaveTextContent('Contract was not found.');
    expect(screen.getAllByText('Check the ID and try again.').length).toBeGreaterThan(0);
    expectNoWorkflowMutationRequests();
  });

  it('filters the contract list by state without calling mutation endpoints', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({ id: 'contract-list-draft', state: 'draft' }),
    ]);
    fetchMock.mockResolvedValueOnce(jsonResponse(contractListResponse([
      contractResponse({
        id: 'contract-ready',
        goal_id: 'goal-ready',
        state: 'ready_for_approval',
        current_draft_id: 'draft-ready',
      }),
    ])));
    fetchMock.mockResolvedValueOnce(jsonResponse(contractDraftResponse({
      id: 'draft-ready',
      contract_id: 'contract-ready',
      goal_id: 'goal-ready',
      title: 'Filtered contract draft title',
    })));

    fireEvent.change(screen.getByLabelText(/^State$/i), {
      target: { value: 'ready_for_approval' },
    });

    await waitFor(() => {
      expect(fetchMock.mock.calls.map(([url]) => String(url))).toContain('/v1/contracts?state=ready_for_approval&limit=50');
    });
    expect(within(screen.getByLabelText(/contracts list/i)).getByText('contract-ready')).toBeInTheDocument();
    expect(screen.getByText('Filtered contract draft title')).toBeInTheDocument();
    const requestURLs = fetchMock.mock.calls.map(([url]) => String(url));
    expect(requestURLs.some((url) => /\/v1\/contracts$/.test(url))).toBe(false);
    expect(requestURLs.some((url) => /\/v1\/contracts\/.*\/(submissions|approvals|plans)/.test(url))).toBe(false);
  });

  it('keeps visible contract rows on transient list refresh errors', async () => {
    await loginSuccessfully('en', 'owner', [
      contractResponse({
        id: 'contract-stable',
        goal_id: 'goal-stable',
        repo_binding_id: 'repo-stable',
        current_draft_id: 'draft-stable',
      }),
    ]);
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('server_error'), 503));

    fireEvent.click(screen.getByRole('button', { name: /^Refresh$/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Goalrail server returned an error. Code: 503.');
    const contractList = screen.getByLabelText(/contracts list/i);
    expect(contractList).toHaveTextContent('contract-stable');
    expect(contractList).toHaveTextContent('goal-stable');
    expect(screen.getAllByText('draft-stable').length).toBeGreaterThan(0);
  });

  it('loads the qualification feed with one primary status, calm timestamps, and no action calls', async () => {
    await loginSuccessfully();
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-05-08T10:00:30Z'));
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-created',
            intakeId: 'intake-created',
            title: 'Created qualification goal',
            lane: 'qualification',
            goalState: 'created',
            reasonCodes: ['missing_scope_hint'],
            nextAction: 'continue_goal',
          }),
          qualificationFeedItem({
            goalId: 'goal-clarification',
            intakeId: 'intake-clarification',
            title: 'Clarification goal',
            lane: 'clarification',
            goalState: 'needs_clarification',
            reasonCodes: ['missing_acceptance_hint'],
            openClarificationRequest: openClarificationRequest(),
            nextAction: 'answer_clarification',
            nextActionBlocking: true,
          }),
          qualificationFeedItem({
            goalId: 'goal-ready',
            intakeId: 'intake-ready',
            title: 'Ready for contract seed goal',
            lane: 'qualification',
            goalState: 'ready_for_contract_seed',
            ready: true,
            reasonCodes: [],
            nextAction: 'draft_contract',
          }),
          qualificationFeedItem({
            goalId: 'goal-draft',
            intakeId: 'intake-draft',
            title: 'Contract draft goal',
            lane: 'contract',
            goalState: 'ready_for_contract_seed',
            ready: true,
            linkedContract: { id: 'contract-draft', state: 'draft' },
            nextAction: 'update_contract',
          }),
          qualificationFeedItem({
            goalId: 'goal-approved',
            intakeId: 'intake-approved',
            title: 'Approved contract goal',
            lane: 'contract',
            goalState: 'ready_for_contract_seed',
            ready: true,
            linkedContract: { id: 'contract-approved', state: 'approved' },
            nextAction: 'plan_work',
          }),
          qualificationFeedItem({
            goalId: 'goal-rejected',
            intakeId: 'intake-rejected',
            title: 'Rejected qualification goal',
            lane: 'blocked',
            goalState: 'rejected',
            reasonCodes: ['missing_scope_hint'],
            nextAction: 'blocked',
            nextActionAvailable: false,
            nextActionBlocking: true,
          }),
        ])
      )
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    await act(async () => {
      await Promise.resolve();
    });

    const board = screen.getByLabelText(/qualification feed lane view/i);
    const qualificationLane = within(board).getByLabelText('Qualification lane');
    const clarificationLane = within(board).getByLabelText('Clarification lane');
    const contractLane = within(board).getByLabelText('Contract lane');
    const blockedLane = within(board).getByLabelText('Blocked lane');

    const qualificationFeedCall = findFetchRequest('/v1/qualification-feed?limit=50');
    expect(qualificationFeedCall?.options).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(qualificationLane).toHaveTextContent('Created qualification goal');
    expect(qualificationLane).toHaveTextContent('Ready for contract seed goal');
    expect(qualificationLane).toHaveTextContent('Needs qualification');
    expect(qualificationLane).toHaveTextContent('Ready for contract');
    const qualificationCards = Array.from(qualificationLane.querySelectorAll('.qualificationCard')).map((card) => card.textContent ?? '');
    expect(qualificationCards[0]).toContain('Ready for contract seed goal');
    expect(qualificationCards[1]).toContain('Created qualification goal');
    expect(clarificationLane).toHaveTextContent('Clarification goal');
    expect(clarificationLane).toHaveTextContent('1 open questions');
    expect(clarificationLane).toHaveTextContent('What is the intended scope at a high level?');
    expect(clarificationLane).toHaveTextContent('A scope hint is required before contract seed readiness.');
    expect(clarificationLane).toHaveTextContent('Needs answer');
    expect(within(clarificationLane).queryByRole('textbox')).not.toBeInTheDocument();
    expect(within(clarificationLane).queryByRole('button', { name: /^Answer questions$/i })).not.toBeInTheDocument();
    expect(contractLane).toHaveTextContent('Contract draft goal');
    expect(contractLane).toHaveTextContent('Draft');
    expect(contractLane).toHaveTextContent('Contract linked');
    expect(within(contractLane).getAllByRole('button', { name: /^Open contract$/i })).toHaveLength(2);
    expect(contractLane).toHaveTextContent('Approved contract goal');
    expect(contractLane).toHaveTextContent('Approved');
    expect(blockedLane).toHaveTextContent('Rejected qualification goal');
    expect(blockedLane).toHaveTextContent('Rejected');
    const cards = Array.from(board.querySelectorAll('.qualificationCard'));
    expect(cards).toHaveLength(6);
    cards.forEach((card) => {
      const statusPills = Array.from(card.querySelectorAll('.opsPill'));
      expect(statusPills).toHaveLength(1);
      expect(statusPills[0]).toHaveTextContent(/^(Needs answer|Ready for contract|Needs qualification|Contract linked|Blocked)$/);
      expect(within(card as HTMLElement).queryByText(/^Ready$/)).not.toBeInTheDocument();
      expect(within(card as HTMLElement).queryByText(/^Not ready$/)).not.toBeInTheDocument();
    });
    expect(within(board).getAllByText('just now')).toHaveLength(6);
    expect(board).not.toHaveTextContent(/2026-05-08T10:00:00Z/);
    expect(board).not.toHaveTextContent(/continue_goal|answer_clarification|draft_contract/i);
    expect(board).not.toHaveTextContent(/Continue \/ Recheck|Answer questions|Draft contract|^Approve$|Plan work|Working\.\.\./i);
    const requestURLs = fetchMock.mock.calls.map(([url]) => String(url));
    expect(requestURLs.some((url) => /\/v1\/goals\/.*\/continuation/.test(url))).toBe(false);
    expect(requestURLs.some((url) => /\/v1\/clarifications\/.*\/answers\/continuation/.test(url))).toBe(false);
    expect(requestURLs.some((url) => /\/v1\/contracts$/.test(url))).toBe(false);
  });

  it('polls qualification feed into linked-contract backend state without UI mutation', async () => {
    await loginSuccessfully();
    vi.useFakeTimers();
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-cli-created',
            intakeId: 'intake-cli-created',
            title: 'CLI-created goal awaiting contract',
            lane: 'qualification',
            goalState: 'ready_for_contract_seed',
            ready: true,
            reasonCodes: [],
            nextAction: 'draft_contract',
          }),
        ])
      )
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    await act(async () => {
      await Promise.resolve();
    });
    expect(screen.getByText('CLI-created goal awaiting contract')).toBeInTheDocument();
    expect(screen.getByText('Ready for contract')).toBeInTheDocument();
    expect(screen.getByText('No linked contract')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Draft contract$/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Open contract$/i })).not.toBeInTheDocument();

    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-cli-created',
            intakeId: 'intake-cli-created',
            title: 'CLI-created goal linked by backend state',
            lane: 'contract',
            goalState: 'ready_for_contract_seed',
            ready: true,
            reasonCodes: [],
            linkedContract: { id: 'contract-cli-created', state: 'draft' },
            nextAction: 'update_contract',
          }),
        ])
      )
    );

    await act(async () => {
      vi.advanceTimersByTime(5000);
      await Promise.resolve();
    });

    const board = screen.getByLabelText(/qualification feed lane view/i);
    const contractLane = within(board).getByLabelText('Contract lane');
    expect(contractLane).toHaveTextContent('CLI-created goal linked by backend state');
    expect(contractLane).toHaveTextContent('Contract linked');
    expect(contractLane).toHaveTextContent('Draft · contract-cli-created');
    expect(within(contractLane).getByRole('button', { name: /^Open contract$/i })).toBeInTheDocument();
    expect(screen.queryByText('CLI-created goal awaiting contract')).not.toBeInTheDocument();
    expect(fetchMock.mock.calls.map(([url]) => String(url)).filter((url) => url === '/v1/qualification-feed?limit=50')).toHaveLength(2);
    expectNoWorkflowMutationRequests();
  });

  it('skips scheduled qualification feed polling while the document is hidden', async () => {
    await loginSuccessfully();
    vi.useFakeTimers();
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-visible',
            intakeId: 'intake-visible',
            title: 'Visible polling goal',
          }),
        ])
      )
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    await act(async () => {
      await Promise.resolve();
    });
    expect(screen.getByText('Visible polling goal')).toBeInTheDocument();

    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: true,
    });
    await act(async () => {
      vi.advanceTimersByTime(5000);
      await Promise.resolve();
    });

    expect(fetchMock.mock.calls.map(([url]) => String(url)).filter((url) => url === '/v1/qualification-feed?limit=50')).toHaveLength(1);

    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-visible-again',
            intakeId: 'intake-visible-again',
            title: 'Visible again polling goal',
          }),
        ])
      )
    );
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: false,
    });
    document.dispatchEvent(new Event('visibilitychange'));
    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.getByText('Visible again polling goal')).toBeInTheDocument();
    expect(fetchMock.mock.calls.map(([url]) => String(url)).filter((url) => url === '/v1/qualification-feed?limit=50')).toHaveLength(2);
  });

  it('shows qualification items as read-only and does not expose continue controls', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-created',
            intakeId: 'intake-created',
            title: 'Created qualification goal',
            lane: 'qualification',
            goalState: 'created',
            reasonCodes: ['missing_scope_hint'],
            nextAction: 'continue_goal',
          }),
        ])
      )
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    expect(await screen.findByText('Created qualification goal')).toBeInTheDocument();
    expect(screen.getByText('Needs qualification')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Continue \/ Recheck$/i })).not.toBeInTheDocument();
    expect(fetchMock.mock.calls.map(([url]) => String(url)).some((url) => /\/v1\/goals\/goal-created\/continuation/.test(url))).toBe(false);
    expect(fetchMock.mock.calls.map(([url]) => String(url)).some((url) => /\/v1\/contracts$/.test(url))).toBe(false);
  });

  it('shows ready-for-contract feed items without a draft contract control', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-ready',
            intakeId: 'intake-ready',
            title: 'Ready for contract seed goal',
            lane: 'qualification',
            goalState: 'ready_for_contract_seed',
            ready: true,
            reasonCodes: [],
            nextAction: 'draft_contract',
          }),
        ])
      )
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    expect(await screen.findByText('Ready for contract seed goal')).toBeInTheDocument();
    expect(screen.getByText('Ready for contract')).toBeInTheDocument();
    expect(screen.getByText('No linked contract')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Draft contract$/i })).not.toBeInTheDocument();
    expect(fetchMock.mock.calls.map(([url]) => String(url)).filter((url) => url === '/v1/contracts')).toHaveLength(0);
  });

  it('keeps clarification questions read-only and does not expose answer submission controls', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-clarification',
            intakeId: 'intake-clarification',
            title: 'Clarification goal',
            lane: 'clarification',
            goalState: 'needs_clarification',
            reasonCodes: ['missing_scope_hint'],
            openClarificationRequest: openClarificationRequest(),
            nextAction: 'answer_clarification',
            nextActionBlocking: true,
          }),
        ])
      )
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    const board = await screen.findByLabelText(/qualification feed lane view/i);
    const clarificationLane = within(board).getByLabelText('Clarification lane');
    expect(clarificationLane).toHaveTextContent('Clarification goal');
    expect(clarificationLane).toHaveTextContent('Needs answer');
    expect(clarificationLane).toHaveTextContent('Questions');
    expect(clarificationLane).toHaveTextContent('What is the intended scope at a high level?');
    expect(clarificationLane).toHaveTextContent('A scope hint is required before contract seed readiness.');
    expect(within(clarificationLane).queryByRole('textbox')).not.toBeInTheDocument();
    expect(within(clarificationLane).queryByRole('button', { name: /^Answer questions$/i })).not.toBeInTheDocument();
    expect(fetchMock.mock.calls.map(([url]) => String(url)).some((url) => /\/answers\/continuation/.test(url))).toBe(false);
    expect(fetchMock.mock.calls.map(([url]) => String(url)).some((url) => /\/v1\/goals\/.*\/continuation/.test(url))).toBe(false);
    expect(fetchMock.mock.calls.map(([url]) => String(url)).some((url) => /\/v1\/contracts$/.test(url))).toBe(false);
  });

  it('shows actor-mapped clarification questions as read-only information', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-intent-owner',
            intakeId: 'intake-intent-owner',
            title: 'Intent owner clarification',
            lane: 'clarification',
            goalState: 'needs_clarification',
            reasonCodes: ['missing_intent_owner'],
            openClarificationRequest: openIntentOwnerClarificationRequest(),
            nextAction: 'answer_clarification',
            nextActionBlocking: true,
          }),
        ])
      )
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    expect(await screen.findByText('Intent owner clarification')).toBeInTheDocument();
    expect(screen.getByText('Who owns this intent?')).toBeInTheDocument();
    expect(screen.getByText('An intent owner is required before contract seed readiness.')).toBeInTheDocument();
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Answer questions$/i })).not.toBeInTheDocument();

    const requestURLs = fetchMock.mock.calls.map(([url]) => String(url));
    expect(requestURLs.some((url) => /\/v1\/clarifications\/.*\/answers\/continuation/.test(url))).toBe(false);
    expect(requestURLs.some((url) => /\/v1\/goals\/.*\/continuation/.test(url))).toBe(false);
    expect(requestURLs.some((url) => /\/v1\/contracts$/.test(url))).toBe(false);
  });

  it('opens linked contracts through the existing read-only contract detail endpoint', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-linked',
            intakeId: 'intake-linked',
            title: 'Linked contract goal',
            lane: 'contract',
            goalState: 'ready_for_contract_seed',
            ready: true,
            reasonCodes: [],
            linkedContract: { id: 'contract-linked', state: 'draft' },
            nextAction: 'update_contract',
          }),
        ])
      )
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    const board = await screen.findByLabelText(/qualification feed lane view/i);
    const contractLane = within(board).getByLabelText('Contract lane');
    expect(contractLane).toHaveTextContent('Linked contract goal');
    expect(contractLane).toHaveTextContent('Draft');
    expect(contractLane).toHaveTextContent('Contract linked');

    fetchMock.mockImplementation((url: string | URL | Request) => {
      const requestURL = String(url);
      if (requestURL === '/v1/contracts/contract-linked') {
        return Promise.resolve(jsonResponse(contractResponse({
          id: 'contract-linked',
          goal_id: 'goal-linked',
          current_draft_id: 'draft-linked',
        })));
      }
      if (requestURL === '/v1/contracts/contract-linked/current-draft') {
        return Promise.resolve(jsonResponse(contractDraftResponse({
          id: 'draft-linked',
          contract_id: 'contract-linked',
          goal_id: 'goal-linked',
          title: 'Linked contract draft title',
          intent_summary: 'Linked contract draft intent summary.',
        })));
      }
      if (requestURL === '/v1/contracts?limit=50') {
        return Promise.resolve(jsonResponse(contractListResponse([
          contractResponse({
            id: 'contract-linked',
            goal_id: 'goal-linked',
            current_draft_id: 'draft-linked',
          }),
        ])));
      }

      return Promise.resolve(jsonResponse(contractListResponse([])));
    });
    fireEvent.click(within(contractLane).getByRole('button', { name: /^Open contract$/i }));

    expect(await screen.findByText('Linked contract draft title')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Contracts$/i })).toHaveAttribute('aria-current', 'page');
    const linkedContractDetailCall = fetchMock.mock.calls.find(([url]) => String(url) === '/v1/contracts/contract-linked');
    expect(linkedContractDetailCall).toBeDefined();
    expect(linkedContractDetailCall?.[1]).toEqual(
      expect.objectContaining({
        method: 'GET',
        credentials: 'omit',
        headers: { Authorization: 'Bearer access-token' },
      })
    );
    expect(fetchMock.mock.calls.map(([url]) => String(url)).filter((url) => url === '/v1/contracts')).toHaveLength(0);
    expect(fetchMock.mock.calls.map(([url]) => String(url)).some((url) => /\/v1\/goals\/.*\/continuation/.test(url))).toBe(false);
    expect(fetchMock.mock.calls.map(([url]) => String(url)).some((url) => /\/v1\/clarifications\/.*\/answers\/continuation/.test(url))).toBe(false);
  });

  it('smokes the read-only Delivery Readiness to current draft Contract handoff', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-handoff',
            intakeId: 'intake-handoff',
            repoBindingId: 'repo-handoff',
            repositoryFullName: 'heurema/goalrail',
            title: 'Console goal contract handoff',
            lane: 'contract',
            goalState: 'ready_for_contract_seed',
            ready: true,
            reasonCodes: [],
            linkedContract: { id: 'contract-handoff', state: 'draft' },
            nextAction: 'update_contract',
            createdAt: '2026-05-08T09:50:00Z',
          }),
        ])
      )
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    const board = await screen.findByLabelText(/qualification feed lane view/i);
    const contractLane = within(board).getByLabelText('Contract lane');
    const openContractButton = within(contractLane).getByRole('button', { name: /^Open contract$/i });
    const handoffCard = openContractButton.closest('article');

    expect(handoffCard).not.toBeNull();
    expect(handoffCard).toHaveTextContent('Console goal contract handoff');
    expect(handoffCard).toHaveTextContent('goal-handoff');
    expect(handoffCard).toHaveTextContent('heurema/goalrail');
    expect(handoffCard).toHaveTextContent('repo-handoff');
    expect(handoffCard).toHaveTextContent('Ready for contract seed');
    expect(handoffCard).toHaveTextContent('Contract linked');
    expect(handoffCard).toHaveTextContent('Draft · contract-handoff');
    expect(within(handoffCard as HTMLElement).getAllByRole('button')).toHaveLength(1);
    expect(within(handoffCard as HTMLElement).queryByRole('button', { name: /^Draft contract$/i })).not.toBeInTheDocument();
    expect(within(handoffCard as HTMLElement).queryByRole('button', { name: /^Answer questions$/i })).not.toBeInTheDocument();
    expect(within(handoffCard as HTMLElement).queryByRole('button', { name: /^Continue \/ Recheck$/i })).not.toBeInTheDocument();

    fetchMock.mockImplementation((url: string | URL | Request) => {
      const requestURL = String(url);
      if (requestURL === '/v1/contracts?limit=50') {
        return Promise.resolve(jsonResponse(contractListResponse([
          contractResponse({
            id: 'contract-handoff',
            goal_id: 'goal-handoff',
            repo_binding_id: 'repo-handoff',
            current_seed_id: 'seed-handoff',
            current_draft_id: 'draft-handoff',
            state: 'draft',
          }),
        ])));
      }
      if (requestURL === '/v1/contracts/contract-handoff') {
        return Promise.resolve(jsonResponse(contractResponse({
          id: 'contract-handoff',
          goal_id: 'goal-handoff',
          repo_binding_id: 'repo-handoff',
          current_seed_id: 'seed-handoff',
          current_draft_id: 'draft-handoff',
          state: 'draft',
        })));
      }
      if (requestURL === '/v1/contracts/contract-handoff/current-draft') {
        return Promise.resolve(jsonResponse(contractDraftResponse({
          id: 'draft-handoff',
          contract_id: 'contract-handoff',
          contract_seed_id: 'seed-handoff',
          goal_id: 'goal-handoff',
          repo_binding_id: 'repo-handoff',
          title: 'Console handoff current draft',
          intent_summary: 'Pin the read-only goal to contract handoff.',
          proposed_scope: ['Open the linked Contract from Delivery Readiness'],
          proposed_non_goals: ['Do not add browser lifecycle controls'],
          proposed_constraints: ['Use read-only selected Contract endpoints'],
          proposed_acceptance_criteria: ['Current draft body renders after Open contract'],
          proposed_expected_checks: ['npm --prefix apps/web/console run test'],
          proposed_proof_expectations: ['Regression smoke verifies no workflow mutations'],
          risk_hints: ['Keep CLI-owned lifecycle outside the Console'],
          source_refs: [{ kind: 'contract_seed', id: 'seed-handoff' }],
        })));
      }
      if (requestURL === '/v1/qualification-feed?limit=50') {
        return Promise.resolve(jsonResponse(qualificationFeedResponse([])));
      }

      return Promise.resolve(jsonResponse(contractListResponse([])));
    });

    fireEvent.click(openContractButton);

    expect(await screen.findByText('Console handoff current draft')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Contracts$/i })).toHaveAttribute('aria-current', 'page');

    const contractList = screen.getByLabelText(/contracts list/i);
    expect(contractList).toHaveTextContent('contract-handoff');
    expect(contractList).toHaveTextContent('goal-handoff');
    expect(contractList).toHaveTextContent('repo-handoff');

    const selectedDetail = screen.getByLabelText(/selected contract detail/i);
    expect(selectedDetail).toHaveTextContent('contract-handoff');
    expect(selectedDetail).toHaveTextContent('goal-handoff');
    expect(selectedDetail).toHaveTextContent('repo-handoff');
    expect(selectedDetail).toHaveTextContent('seed-handoff');
    expect(selectedDetail).toHaveTextContent('draft-handoff');
    expect(selectedDetail).toHaveTextContent('Draft');

    const currentDraft = screen.getByLabelText(/current draft detail/i);
    expect(currentDraft).toHaveTextContent('Console handoff current draft');
    expect(currentDraft).toHaveTextContent('Pin the read-only goal to contract handoff.');
    expect(currentDraft).toHaveTextContent('Open the linked Contract from Delivery Readiness');
    expect(currentDraft).toHaveTextContent('Current draft body renders after Open contract');
    expect(currentDraft).toHaveTextContent('Do not add browser lifecycle controls');
    expect(currentDraft).toHaveTextContent('Use read-only selected Contract endpoints');
    expect(currentDraft).toHaveTextContent('Regression smoke verifies no workflow mutations');
    expect(screen.getByText('Task, execution, gate, and proof data are not available in this Console view yet.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Submit$/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Approve$/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Plan work$/i })).not.toBeInTheDocument();

    const qualificationFeedCall = findFetchRequest('/v1/qualification-feed?limit=50');
    const contractListCall = findFetchRequest('/v1/contracts?limit=50');
    const contractDetailCall = findFetchRequest('/v1/contracts/contract-handoff');
    const currentDraftCall = findFetchRequest('/v1/contracts/contract-handoff/current-draft');
    expect(qualificationFeedCall?.options).toEqual(expect.objectContaining({
      method: 'GET',
      credentials: 'omit',
      headers: { Authorization: 'Bearer access-token' },
    }));
    expect(contractListCall?.options).toEqual(expect.objectContaining({
      method: 'GET',
      credentials: 'omit',
      headers: { Authorization: 'Bearer access-token' },
    }));
    expect(contractDetailCall?.options).toEqual(expect.objectContaining({
      method: 'GET',
      credentials: 'omit',
      headers: { Authorization: 'Bearer access-token' },
    }));
    expect(currentDraftCall?.options).toEqual(expect.objectContaining({
      method: 'GET',
      credentials: 'omit',
      headers: { Authorization: 'Bearer access-token' },
    }));
    expectNoWorkflowMutationRequests();
  });

  it('shows contract-specific access errors from Delivery Readiness linked-contract navigation', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        qualificationFeedResponse([
          qualificationFeedItem({
            goalId: 'goal-linked-forbidden',
            intakeId: 'intake-linked-forbidden',
            title: 'Linked forbidden contract goal',
            lane: 'contract',
            goalState: 'ready_for_contract_seed',
            ready: true,
            reasonCodes: [],
            linkedContract: { id: 'contract-linked-forbidden', state: 'draft' },
            nextAction: 'update_contract',
          }),
        ])
      )
    );

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    const board = await screen.findByLabelText(/qualification feed lane view/i);
    const contractLane = within(board).getByLabelText('Contract lane');

    fetchMock.mockImplementation((url: string | URL | Request) => {
      const requestURL = String(url);
      if (requestURL === '/v1/contracts/contract-linked-forbidden') {
        return Promise.resolve(jsonResponse(errorEnvelope('forbidden'), 403));
      }
      if (requestURL === '/v1/contracts?limit=50') {
        return Promise.resolve(jsonResponse(contractListResponse([])));
      }

      return Promise.resolve(jsonResponse(contractListResponse([])));
    });
    fireEvent.click(within(contractLane).getByRole('button', { name: /^Open contract$/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('You do not have access to this contract.');
    expect(screen.getByRole('button', { name: /^Contracts$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.queryByText('Contract was not found.')).not.toBeInTheDocument();
    expectNoWorkflowMutationRequests();
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

  it.each([
    ['forbidden', 403, 'You do not have access to this contract.'],
    ['membership_required', 403, 'This contract requires active organization access.'],
    ['database_not_configured', 503, 'Goalrail server is not ready yet.'],
    ['server_error', 502, 'Goalrail server returned a contract detail error. Code: 502.'],
  ] as const)('manual Contract ID lookup shows a contract-specific %s error', async (code, status, message) => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope(code), status));

    fireEvent.change(screen.getByLabelText(/^Contract ID$/i), {
      target: { value: `contract-${code}` },
    });
    fireEvent.click(screen.getByRole('button', { name: /load contract/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent(message);
    expect(screen.queryByText('Contract was not found.')).not.toBeInTheDocument();
    expect(screen.queryByText(`contract-${code}`)).not.toBeInTheDocument();
    expectNoWorkflowMutationRequests();
  });

  it('manual Contract ID lookup shows clear transient detail errors without creating fake data', async () => {
    await loginSuccessfully();
    fetchMock.mockRejectedValueOnce(new TypeError('network failed'));

    fireEvent.change(screen.getByLabelText(/^Contract ID$/i), {
      target: { value: 'contract-network-error' },
    });
    fireEvent.click(screen.getByRole('button', { name: /load contract/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not reach the Goalrail server while loading this contract.');
    expect(screen.queryByText('contract-network-error')).not.toBeInTheDocument();

    fetchMock.mockResolvedValueOnce(new Response('not json', { status: 200 }));
    fireEvent.change(screen.getByLabelText(/^Contract ID$/i), {
      target: { value: 'contract-parse-error' },
    });
    fireEvent.click(screen.getByRole('button', { name: /load contract/i }));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Goalrail server returned an unrecognized contract detail response.');
    });
    expect(screen.queryByText('contract-parse-error')).not.toBeInTheDocument();
    expectNoWorkflowMutationRequests();
  });

  it('unauthorized detail errors reset auth state consistently', async () => {
    await loginSuccessfully();
    fetchMock.mockResolvedValueOnce(jsonResponse(errorEnvelope('unauthorized'), 401));

    fireEvent.change(screen.getByLabelText(/^Contract ID$/i), {
      target: { value: 'contract-unauthorized' },
    });
    fireEvent.click(screen.getByRole('button', { name: /load contract/i }));

    expect(await screen.findByLabelText(/^Email$/i)).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent('Session is invalid. Sign in again.');
    expect(screen.queryByRole('navigation', { name: /product surfaces/i })).not.toBeInTheDocument();
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
    expect(fetchMock.mock.calls[4][0]).toBe('/v1/contracts/contract-A');
    expect(fetchMock.mock.calls[5][0]).toBe('/v1/contracts/contract-B');
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
    expect(fetchMock).toHaveBeenCalledTimes(4);
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
    expect(fetchMock.mock.calls[4][0]).toBe('/v1/organizations/018f0000-0000-7000-8000-000000000002/users');
    expect(fetchMock.mock.calls[4][1]).toEqual(
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
    expect(fetchMock.mock.calls[4][0]).toBe('/v1/organizations/018f0000-0000-7000-8000-000000000002/repository-context');
    expect(fetchMock.mock.calls[4][1]).toEqual(
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

  it('ignores stale repository-context responses after auth state changes', async () => {
    await loginSuccessfully();
    const repositoryLookup = deferredResponse();
    fetchMock.mockImplementationOnce(() => repositoryLookup.promise);

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Repository$/i }));
    await screen.findByRole('status');

    fetchMock.mockResolvedValueOnce(jsonResponse({ revoked: true }));
    fireEvent.click(screen.getByRole('button', { name: /log out/i }));
    await screen.findByLabelText(/^Email$/i);

    await act(async () => {
      repositoryLookup.resolve(jsonResponse(repositoryContextResponse([
        repositoryContextRecord({ displayName: 'Stale Repository Context' }),
      ])));
    });

    await waitFor(() => {
      expect(screen.queryByText('Stale Repository Context')).not.toBeInTheDocument();
      expect(screen.getByLabelText(/^Email$/i)).toBeInTheDocument();
    });
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
    expect(fetchMock.mock.calls[5][0]).toBe('/v1/organizations/018f0000-0000-7000-8000-000000000002/users');
    expect(fetchMock.mock.calls[5][1]).toEqual(
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
    expect(fetchMock.mock.calls[5][0]).toBe(
      '/v1/organizations/018f0000-0000-7000-8000-000000000002/users/018f0000-0000-7000-8000-000000000010'
    );
    expect(fetchMock.mock.calls[5][1]).toEqual(
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
    expect(fetchMock.mock.calls[5][0]).toBe(
      '/v1/organizations/018f0000-0000-7000-8000-000000000002/users/018f0000-0000-7000-8000-000000000010/temporary-password-resets'
    );
    expect(fetchMock.mock.calls[5][1]).toEqual(
      expect.objectContaining({
        method: 'POST',
        credentials: 'omit',
        headers: {
          Authorization: 'Bearer access-token',
        },
      })
    );
    expect(fetchMock.mock.calls[5][1]?.body).toBeUndefined();
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
    expect(fetchMock).toHaveBeenCalledTimes(5);
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
    expect(fetchMock).toHaveBeenCalledTimes(5);
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
    expect(fetchMock).toHaveBeenCalledTimes(5);
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
    expect(fetchMock.mock.calls[5][0]).toBe(
      '/v1/organizations/018f0000-0000-7000-8000-000000000002/users/018f0000-0000-7000-8000-000000000001'
    );
    expect(fetchMock.mock.calls[5][1]).toEqual(
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
    expect(fetchMock.mock.calls[5][0]).toBe(
      '/v1/organizations/018f0000-0000-7000-8000-000000000002/users/018f0000-0000-7000-8000-000000000011'
    );
    expect(fetchMock.mock.calls[5][1]).toEqual(
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
      /registration|register|sign up|sso|invite|reset password|password reset|analytics|chat|file upload|model selector|organization creation|repo integration|runner online|runner active|gate queue/i
    );
  });
});

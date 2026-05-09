import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

import type { ContractDraftResponse } from './contractDraftClient';
import type { ContractResponse } from './contractDetailClient';
import type { OrganizationRepositoryContextResponse, RepositoryContextRecord } from './repositoryContextClient';
import demoContractsCss from './DemoContractsPage.css?raw';

const demoContractsShadowCss = demoContractsCss
  .replace(':root {', ':host {')
  .replace('html,\nbody,\n#root {', ':host,\n#root {')
  .replace('html,\nbody {', ':host {')
  .replace('body {', ':host {');

type StepIndex = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7;
type ActiveSurface = 'contracts' | 'readiness' | 'proof';
type RepoId = 'trialops-demo' | 'billing-api' | 'frontend-console';
type ContractRepoId = string;
type RepoFilter = string;
type ApprovalState = 'pending' | 'accepted' | 'rework' | 'blocked';
type Tone = 'mauve' | 'amber' | 'pass' | 'block';

interface Stage {
  id: string;
  name: string;
}

interface RepoContext {
  repo: string;
  bound: 'да' | 'нет';
  init: string;
  docsIndexed: number;
  readiness: number;
  scanStatus: string;
  testsStatus: string;
  ciStatus: string;
  ownersRulesStatus: string;
  proofSurfaceStatus: string;
  recommendedMode: string;
  checklist: Array<{ label: string; value: string; tone: Tone }>;
  runtimePolicy: string;
  runtimes: string[];
}

interface ClarificationCard {
  ref: string;
  question: string;
  answer: string;
  note: string;
}

interface WorkItem {
  id: string;
  title: string;
  lane: 'внешний рантайм' | 'ручной шаг' | 'проверка/подтверждение';
  scope: string;
  status: string;
  proofObligation: string;
}

interface EvidenceItem {
  label: string;
  value: string;
  tone: Tone;
}

interface VerificationRow {
  criterion: string;
  support: string;
  outcome: string;
}

interface ContractRecord {
  id: string;
  title: string;
  repo: ContractRepoId;
  repoFilterValue?: string;
  backendState?: ContractResponse['state'];
  currentDraftId?: string;
  updatedAt?: string;
  backendContract?: ContractResponse;
  owner: string;
  scopeSurface: string;
  summary: string;
  defaultStep: StepIndex;
  goal: string;
  intakeNotes: string[];
  inScope: string[];
  outOfScope: string[];
  acceptance: string[];
  proofExpectations: string[];
  policyNote: string;
  clarifications: ClarificationCard[];
  workItems: WorkItem[];
  evidence: EvidenceItem[];
  verification: VerificationRow[];
  changed: string[];
  unchanged: string[];
  trust: string[];
  howToVerify: string[];
  activity: Record<number, Array<{ kind: string; note: string; tone: Tone }>>;
}

type LiveContractListLoadStatus = 'idle' | 'loading' | 'loaded' | 'error';
type LiveContractLoadStatus = 'idle' | 'loading' | 'loaded' | 'not_found' | 'error';
type LiveContractDraftLoadStatus = 'idle' | 'loading' | 'loaded' | 'no_draft' | 'unavailable' | 'error';
type LiveRepositoryContextLoadStatus = 'idle' | 'loading' | 'loaded' | 'error';

export interface DemoContractsLiveData {
  contracts: ContractResponse[];
  selectedContract: ContractResponse | null;
  selectedDraft: ContractDraftResponse | null;
  contractListLoadStatus: LiveContractListLoadStatus;
  contractListError: string;
  contractLoadStatus: LiveContractLoadStatus;
  contractError: string;
  contractDraftLoadStatus: LiveContractDraftLoadStatus;
  contractDraftError: string;
  repositoryContext: OrganizationRepositoryContextResponse | null;
  repositoryContextLoadStatus: LiveRepositoryContextLoadStatus;
  repositoryContextError: string;
  repoBindingFilter: string;
  stateFilter: ContractResponse['state'] | 'all';
  onContractSelect: (contract: ContractResponse) => void;
  onRefresh: () => void;
  onRepoBindingFilterChange: (repoBindingId: string) => void;
  onStateFilterChange: (state: ContractResponse['state'] | 'all') => void;
}

interface ProofFeedItem {
  id: string;
  contractId: string;
  repo: ContractRepoId;
  proofStatus: string;
  decisionStatus: string;
  humanApproval: string;
  linkedEvidence: string;
  criteriaCoverage: string;
  summary: string;
  tone: Tone;
  changed: string[];
  unchanged: string[];
  verified: string[];
  decisionTrail: string[];
  archiveLine: string;
}

interface MobileContractQueueItem {
  id: string;
  title: string;
  status: string;
  tone: Tone;
  stage: string;
  stageProgress: string;
  policy: string;
  humanDecision: string;
  repo: ContractRepoId;
  detail: {
    changePacket: string;
    evidence: string;
    projectContext: string;
    decisionTrail: string;
  };
}

interface MobileRepoQueueItem {
  repo: RepoId;
  readiness: string;
  status: string;
  tone: Tone;
}

interface MobileProofQueueItem {
  id: string;
  contractId: string;
  status: string;
  coverage: string;
  tone: Tone;
}

const STAGES: Stage[] = [
  { id: 'goal-intake', name: 'Входящий запрос' },
  { id: 'clarification', name: 'Уточнения' },
  { id: 'working-contract', name: 'Рабочий контракт' },
  { id: 'work-items', name: 'Задачи' },
  { id: 'execution-evidence', name: 'Артефакты выполнения' },
  { id: 'verification', name: 'Проверка' },
  { id: 'proof', name: 'Пакет подтверждения' },
  { id: 'решение', name: 'Решение' },
];

const REPO_OPTIONS: Array<{ value: RepoFilter; label: string }> = [
  { value: 'trialops-demo', label: 'trialops-demo' },
  { value: 'billing-api', label: 'billing-api' },
  { value: 'all', label: 'Все репозитории' },
];

const WORKSPACE_REPOS: RepoId[] = ['trialops-demo', 'billing-api', 'frontend-console'];

const REPO_CONTEXTS: Record<RepoId, RepoContext> = {
  'trialops-demo': {
    repo: 'trialops-demo',
    bound: 'да',
    init: 'завершено',
    docsIndexed: 12,
    readiness: 72,
    scanStatus: 'контекст просканирован',
    testsStatus: 'тесты найдены',
    ciStatus: 'подключено',
    ownersRulesStatus: 'правила найдены',
    proofSurfaceStatus: 'доступно',
    recommendedMode: 'локальное ограниченное выполнение',
    checklist: [
      { label: 'Тесты', value: 'найдены', tone: 'pass' },
      { label: 'CI', value: 'подключено', tone: 'mauve' },
      { label: 'Правила агента', value: 'есть', tone: 'pass' },
    ],
    runtimePolicy: 'только локально',
    runtimes: ['Codex CLI', 'Claude Code', 'ручной шаг'],
  },
  'billing-api': {
    repo: 'billing-api',
    bound: 'да',
    init: 'завершено',
    docsIndexed: 7,
    readiness: 58,
    scanStatus: 'контекст просканирован частично',
    testsStatus: 'тесты частично найдены',
    ciStatus: 'подключено',
    ownersRulesStatus: 'владельцы не найдены',
    proofSurfaceStatus: 'доступно после согласования',
    recommendedMode: 'нужно ручное согласование',
    checklist: [
      { label: 'Тесты', value: 'частично', tone: 'amber' },
      { label: 'CI', value: 'подключено', tone: 'pass' },
      { label: 'Владельцы/правила', value: 'нет', tone: 'amber' },
    ],
    runtimePolicy: 'нужно ручное согласование',
    runtimes: ['Codex CLI', 'ручной шаг', 'канал проверки без записи'],
  },
  'frontend-console': {
    repo: 'frontend-console',
    bound: 'нет',
    init: 'ожидает',
    docsIndexed: 0,
    readiness: 41,
    scanStatus: 'контекст еще не сканировали',
    testsStatus: 'неизвестно',
    ciStatus: 'неизвестно',
    ownersRulesStatus: 'правила не настроены',
    proofSurfaceStatus: 'не готово',
    recommendedMode: 'нужна настройка',
    checklist: [
      { label: 'Тесты', value: 'неизвестно', tone: 'amber' },
      { label: 'CI', value: 'неизвестно', tone: 'amber' },
      { label: 'Владельцы/правила', value: 'ожидает', tone: 'amber' },
    ],
    runtimePolicy: 'нужна настройка',
    runtimes: ['ручная настройка', 'нужна инициализация'],
  },
};

const LIVE_ALL_REPOSITORIES_FILTER = 'all';
const LIVE_CONTRACT_STATE_OPTIONS: Array<{ value: ContractResponse['state'] | 'all'; label: string }> = [
  { value: 'all', label: 'Все состояния' },
  { value: 'seeded', label: 'Seeded' },
  { value: 'draft', label: 'Draft' },
  { value: 'ready_for_approval', label: 'Ready for approval' },
  { value: 'approved', label: 'Approved' },
];

const EMPTY_LIVE_CONTRACT: ContractRecord = {
  id: 'Нет контрактов',
  title: 'Backend не вернул контракты',
  repo: 'backend',
  repoFilterValue: LIVE_ALL_REPOSITORIES_FILTER,
  owner: 'read-only backend',
  scopeSurface: 'GET /v1/contracts',
  summary: 'Контракты появятся здесь после создания через существующий CLI/API flow.',
  defaultStep: 0,
  goal: 'Показать честное пустое состояние backend-backed Contracts страницы.',
  intakeNotes: ['GET /v1/contracts вернул пустой список.', 'Локальные demo-контракты скрыты, чтобы не выдавать мок-данные за backend state.'],
  inScope: ['Read-only discovery', 'Repository context metadata', 'Current draft detail when linked'],
  outOfScope: ['Workflow mutation controls', 'Execution', 'Gate', 'Proof generation'],
  acceptance: ['Пустое состояние видно без подмены backend данных моками.'],
  proofExpectations: ['Проверить backend response и отсутствие mutation calls.'],
  policyNote: 'Console остается read-only для Contract workflow.',
  clarifications: [
    {
      ref: 'backend-state',
      question: 'Почему нет строк?',
      answer: 'Backend не вернул Contract aggregate для текущей Organization/filter.',
      note: 'Создание контрактов остается за CLI/API workflow, не за этой страницей.',
    },
  ],
  workItems: [],
  evidence: [{ label: 'Источник', value: 'GET /v1/contracts?limit=50', tone: 'mauve' }],
  verification: [{ criterion: 'Не показывать мок как real data', support: 'Пустое состояние backend list', outcome: 'Покрыто' }],
  changed: ['Текущая Contracts страница подключена к read-only backend discovery.'],
  unchanged: ['Workflow mutation controls не добавлены.', 'Runner/gate/proof по-прежнему вне этой страницы.'],
  trust: ['Данные берутся из authenticated backend endpoints.', 'Пустой backend список остается пустым на UI.'],
  howToVerify: ['Проверить network calls `/v1/contracts` и `/repository-context`.'],
  activity: {
    0: [{ kind: 'contract.discovery.empty', note: 'Backend discovery завершился без Contract rows', tone: 'amber' }],
  },
};

function makeLiveRepoOptions(repositoryContext: OrganizationRepositoryContextResponse | null) {
  const contexts = repositoryContext?.contexts ?? [];
  return [
    { value: LIVE_ALL_REPOSITORIES_FILTER, label: 'Все репозитории' },
    ...contexts.map((context) => ({
      value: context.repo_binding.id,
      label: context.repo_binding.repository_full_name || context.repo_binding.id,
    })),
  ];
}

function makeLiveRepoContexts(repositoryContext: OrganizationRepositoryContextResponse | null) {
  const contexts = repositoryContext?.contexts ?? [];
  return contexts.reduce<Record<string, RepoContext>>((acc, context) => {
    acc[context.repo_binding.id] = mapRepositoryContext(context);
    return acc;
  }, {});
}

function mapRepositoryContext(context: RepositoryContextRecord): RepoContext {
  return {
    repo: context.repo_binding.repository_full_name || context.repo_binding.id,
    bound: context.repo_binding.state === 'active' ? 'да' : 'нет',
    init: context.repo_binding.state,
    docsIndexed: 0,
    readiness: context.repo_binding.state === 'active' ? 64 : 32,
    scanStatus: `provider: ${context.repo_binding.provider || 'unknown'}`,
    testsStatus: 'metadata-only',
    ciStatus: 'metadata-only',
    ownersRulesStatus: 'metadata-only',
    proofSurfaceStatus: 'недоступно в Console view',
    recommendedMode: context.repo_binding.access_mode || 'metadata_only',
    checklist: [
      { label: 'Project', value: context.project.display_name || context.project.slug || context.project.id, tone: 'mauve' },
      { label: 'RepoBinding', value: context.repo_binding.state || 'unknown', tone: context.repo_binding.state === 'active' ? 'pass' : 'amber' },
      { label: 'Access mode', value: context.repo_binding.access_mode || 'metadata_only', tone: 'amber' },
    ],
    runtimePolicy: 'metadata-only · без provider authorization, checkout, execution, gate или proof',
    runtimes: [
      context.repo_binding.default_branch ? `default: ${context.repo_binding.default_branch}` : 'default branch unknown',
      context.repo_binding.workflow_base_branch ? `workflow base: ${context.repo_binding.workflow_base_branch}` : 'workflow base unknown',
      context.repo_binding.path_scope ? `path: ${context.repo_binding.path_scope}` : 'path scope unknown',
    ],
  };
}

function liveContractRepoLabel(contract: ContractResponse, repositoryContext: OrganizationRepositoryContextResponse | null) {
  const match = repositoryContext?.contexts.find((context) => context.repo_binding.id === contract.repo_binding_id);
  return match?.repo_binding.repository_full_name || contract.repo_binding_id;
}

function mapLiveContract(
  contract: ContractResponse,
  repositoryContext: OrganizationRepositoryContextResponse | null,
  selectedDraft: ContractDraftResponse | null
): ContractRecord {
  const draft = selectedDraft?.contract_id === contract.id ? selectedDraft : null;
  const stateLabel = contractStateLabel(contract.state);
  const defaultStep = stepForContractState(contract.state);
  const repo = liveContractRepoLabel(contract, repositoryContext);
  const title = draft?.title || `Контракт ${shortContractId(contract.id)}`;
  const intent = draft?.intent_summary || `Backend Contract aggregate для Goal ${contract.goal_id}.`;
  const proposedScope = normalizeLiveList(draft?.proposed_scope, [`RepoBinding ${contract.repo_binding_id}`, `Goal ${contract.goal_id}`]);
  const proposedNonGoals = normalizeLiveList(draft?.proposed_non_goals, ['Execution, gate и proof не отображаются этим read-only view.']);
  const proposedAcceptance = normalizeLiveList(draft?.proposed_acceptance_criteria, ['Публичный Contract aggregate загружен из backend.']);
  const proposedProof = normalizeLiveList(draft?.proposed_proof_expectations, ['Проверить read-only API response и UI state.']);
  const expectedChecks = normalizeLiveList(draft?.proposed_expected_checks, ['Console typecheck/test/build']);

  return {
    id: contract.id,
    title,
    repo,
    repoFilterValue: contract.repo_binding_id,
    backendState: contract.state,
    currentDraftId: contract.current_draft_id,
    updatedAt: contract.updated_at,
    backendContract: contract,
    owner: `backend · ${stateLabel}`,
    scopeSurface: draft ? 'current draft · read-only' : 'contract aggregate · read-only',
    summary: intent,
    defaultStep,
    goal: intent,
    intakeNotes: [
      `Contract state: ${stateLabel}`,
      `Goal: ${contract.goal_id}`,
      `RepoBinding: ${contract.repo_binding_id}`,
    ],
    inScope: proposedScope,
    outOfScope: proposedNonGoals,
    acceptance: proposedAcceptance,
    proofExpectations: proposedProof,
    policyNote: 'Данные загружены через authenticated read-only Console endpoints. Workflow mutation controls не включены.',
    clarifications: [
      {
        ref: 'contract-state',
        question: 'Какой backend state у выбранного Contract?',
        answer: stateLabel,
        note: `updated_at: ${formatLiveDate(contract.updated_at)}`,
      },
      {
        ref: 'repo-binding',
        question: 'К какому repository context привязан Contract?',
        answer: repo,
        note: contract.repo_binding_id,
      },
      {
        ref: 'current-draft',
        question: 'Есть ли current draft?',
        answer: contract.current_draft_id ? contract.current_draft_id : 'Нет linked current_draft_id',
        note: draft ? 'Draft body загружен из /current-draft.' : 'Draft detail не загружен или отсутствует.',
      },
    ],
    workItems: [
      {
        id: 'READ-01',
        title: 'Read-only Contract aggregate',
        lane: 'проверка/подтверждение',
        scope: `GET /v1/contracts/${contract.id}`,
        status: 'Загружено',
        proofObligation: 'Не выполнять workflow mutation из Console.',
      },
      {
        id: 'READ-02',
        title: 'Current draft detail',
        lane: 'проверка/подтверждение',
        scope: contract.current_draft_id ? `GET /v1/contracts/${contract.id}/current-draft` : 'current_draft_id absent',
        status: draft ? 'Загружено' : 'Недоступно',
        proofObligation: 'Показывать только read-only draft fields.',
      },
    ],
    evidence: [
      { label: 'Contract ID', value: contract.id, tone: 'mauve' },
      { label: 'State', value: stateLabel, tone: contractStateTone(contract.state) },
      { label: 'Updated', value: formatLiveDate(contract.updated_at), tone: 'pass' },
      { label: 'Current draft', value: contract.current_draft_id || 'нет', tone: contract.current_draft_id ? 'pass' : 'amber' },
    ],
    verification: expectedChecks.map((check) => ({
      criterion: check,
      support: draft ? 'proposed_expected_checks из current draft' : 'fallback Console verification',
      outcome: 'Read-only',
    })),
    changed: proposedScope,
    unchanged: proposedNonGoals,
    trust: [
      'Bearer token хранится только в React memory.',
      'Страница читает backend Contract state и не пишет lifecycle state.',
      'Repository context metadata не означает provider authorization или checkout readiness.',
    ],
    howToVerify: [
      'Проверить `/v1/contracts?limit=50` после login.',
      'Выбрать Contract row и проверить `/v1/contracts/{id}`.',
      'Если есть current_draft_id, проверить `/v1/contracts/{id}/current-draft`.',
    ],
    activity: makeLiveActivity(contract, draft),
  };
}

function normalizeLiveList(values: string[] | undefined, fallback: string[]) {
  return Array.isArray(values) && values.length > 0 ? values : fallback;
}

function contractStateLabel(state: ContractResponse['state']) {
  return state === 'ready_for_approval'
    ? 'Ready for approval'
    : state.charAt(0).toUpperCase() + state.slice(1);
}

function contractStateTone(state: ContractResponse['state']): Tone {
  return state === 'approved' ? 'pass' : state === 'ready_for_approval' ? 'amber' : 'mauve';
}

function stepForContractState(state: ContractResponse['state']): StepIndex {
  return state === 'seeded' ? 1 : state === 'draft' ? 2 : state === 'ready_for_approval' ? 6 : 7;
}

function shortContractId(id: string) {
  return id.length > 12 ? `${id.slice(0, 8)}…${id.slice(-4)}` : id;
}

function formatLiveDate(value: string | undefined) {
  if (!value) {
    return 'unknown';
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString('ru-RU', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function makeLiveActivity(contract: ContractResponse, draft: ContractDraftResponse | null) {
  const stateLabel = contractStateLabel(contract.state);
  return {
    0: [{ kind: 'contract.loaded', note: `Contract ${contract.id} загружен из backend`, tone: 'mauve' as Tone }],
    1: [{ kind: 'contract.state', note: `Lifecycle state: ${stateLabel}`, tone: contractStateTone(contract.state) }],
    2: [{ kind: 'contract.detail', note: `Goal ${contract.goal_id} · RepoBinding ${contract.repo_binding_id}`, tone: 'mauve' as Tone }],
    3: [{ kind: 'contract.readonly', note: 'Workflow mutation controls в Console не включены', tone: 'amber' as Tone }],
    4: [{ kind: 'contract.updated', note: `Updated ${formatLiveDate(contract.updated_at)}`, tone: 'pass' as Tone }],
    5: [{ kind: 'contract.checks', note: draft ? 'Expected checks получены из current draft' : 'Current draft недоступен', tone: draft ? 'pass' as Tone : 'amber' as Tone }],
    6: [{ kind: 'contract.proof', note: draft ? 'Proof expectations получены из current draft' : 'Proof/gate остаются недоступны', tone: draft ? 'pass' as Tone : 'amber' as Tone }],
    7: [{ kind: 'contract.decision', note: contract.state === 'approved' ? 'Contract approved в backend' : 'Human approval через Console не выполняется', tone: contract.state === 'approved' ? 'pass' as Tone : 'amber' as Tone }],
  };
}

function mapMobileContractQueueItem(contract: ContractRecord): MobileContractQueueItem {
  const stateLabel = contract.backendState ? contractStateLabel(contract.backendState) : getStatus(contract.defaultStep, 'pending');
  const stage = STAGES[contract.defaultStep]?.name ?? 'Контракт';

  return {
    id: contract.id,
    title: contract.title,
    status: stateLabel,
    tone: contract.backendState ? contractStateTone(contract.backendState) : getStatusTone(stateLabel),
    stage,
    stageProgress: contract.currentDraftId ? 'current draft linked' : `${contract.defaultStep + 1}/${STAGES.length}`,
    policy: contract.policyNote,
    humanDecision: contract.backendState === 'approved' ? 'Approved in backend' : 'Read-only in Console',
    repo: contract.repo,
    detail: {
      changePacket: contract.summary,
      evidence: contract.evidence.map((item) => `${item.label}: ${item.value}`).join(' · ') || 'Backend evidence unavailable',
      projectContext: contract.goal,
      decisionTrail: contract.trust.join(' · '),
    },
  };
}

const CONTRACTS: ContractRecord[] = [
  {
    id: 'C-0147',
    title: 'Ручное решение',
    repo: 'trialops-demo',
    owner: 'Vitaly · продукт и поставка',
    scopeSurface: 'демо-оболочка · пакет изменения',
    summary: 'Показать детали контракта по выбранному репозиторию и отдельный ручной шаг решения, не расширяя демо-оболочку.',
    defaultStep: 0,
    goal:
      'Показать, как один запрос по репозиторию проходит путь от формулировки до пакета подтверждения и ручного решения без бэкенд, роутинг и реальных интеграций.',
    intakeNotes: [
      'Репозиторий задает контекст, но главным рабочим объектом остается контракт.',
      'Пакет изменения остается понятным человеку и привязан к одному выбранному контракту.',
      'Контекст проекта должен быть отдельно от цепочки контракта.',
    ],
    inScope: [
      'Переключатель репозиториев: trialops-demo, billing-api и режим Все репозитории.',
      'Список контрактов с бейджами репозиториев и детальной карточкой выбранного контракта.',
      'Явные стадии: рабочий контракт, задачи, артефакты выполнения, проверка, пакет подтверждения и решение.',
      'Отдельный блок контекста проекта: привязка репозитория, готовность, политика и рантаймы.',
    ],
    outOfScope: [
      'Бэкенд, API-вызовы, авторизация, роутинг, хранение и серверная логика.',
      'Реальное сканирование репозитория, выполнение рантайма или генерация подтверждения.',
      'Отдельный агрегированный дашборд или чат-ориентированная рабочая область.',
    ],
    acceptance: [
      'Выбранный репозиторий по умолчанию фильтрует список контрактов; Все репозитории работают только как обзор.',
      'Карточка выбранного контракта всегда показывает связанный репозиторий.',
      'Готовность проекта видна вне цепочки контракта.',
      'Финальная стадия требует явного решения человека перед статусом Принято.',
    ],
    proofExpectations: [
      'Показать, какие критерии покрыты и какими демо-артефактами.',
      'Показать, что изменилось и что осталось без изменений в демо-оболочке.',
      'Оставить сценарий проверяемым, но не выдавать его за реальное выполнение.',
    ],
    policyNote:
      'Унаследованный контекст проекта: правила найдены, политика только локально, весь сценарий остается внутри демо-оболочки.',
    clarifications: [
      {
        ref: 'переключатель репозитория',
        question: 'Какие представления репозитория нужны в демо-оболочке?',
        answer: 'trialops-demo, billing-api и Все репозитории',
        note: 'По умолчанию открыт trialops-demo; Все репозитории — обзорный режим.',
      },
      {
        ref: 'главный объект',
        question: 'Что является главным рабочим объектом?',
        answer: 'Контракт, а не общий пакет изменения без привязки к репозиторию',
        note: 'Детальная карточка может оставаться пакетом изменения для выбранного контракта.',
      },
      {
        ref: 'контекст проекта',
        question: 'Где живут привязка репозитория и готовность?',
        answer: 'Вне цепочки контракта, в постоянном боковом блоке',
        note: 'Готовность к поставке уходит из верхней панели в контекст репозитория.',
      },
      {
        ref: 'задачи',
        question: 'Как показать ограниченные задачи?',
        answer: 'Как задачи с типом, областью, статусом и обязанностью по подтверждению',
        note: 'Минимум одна задача должна быть явно ручной.',
      },
      {
        ref: 'ручное решение',
        question: 'Что нужно перед финальным результатом?',
        answer: 'Явное решение человека: принять, отправить на доработку или заблокировать',
        note: 'Ожидание решения и Принято должны визуально отличаться.',
      },
    ],
    workItems: [
      {
        id: 'WI-01',
        title: 'Привязка оболочки к репозиторию',
        lane: 'внешний рантайм',
        scope: 'src/App.tsx · верхняя панель, левый список, карточка выбранного контракта',
        status: 'В области',
        proofObligation: 'Заметка по затронутой поверхности и отметка состояния контракта в центральной панели',
      },
      {
        id: 'WI-02',
        title: 'Проверка текста ручного решения',
        lane: 'ручной шаг',
        scope: 'Текст решения и блок проверки',
        status: 'Только вручную',
        proofObligation: 'Ручной шаг отмечен выполненным в пакете артефактов',
      },
      {
        id: 'WI-03',
        title: 'Пересборка проверки и подтверждения',
        lane: 'проверка/подтверждение',
        scope: 'Матрица покрытия критериев и сводка подтверждения',
        status: 'В очереди',
        proofObligation: 'Матрица покрытия критериев артефактами',
      },
    ],
    evidence: [
      { label: 'Исполнитель', value: 'Codex CLI в локальной рабочей области вне Goalrail', tone: 'mauve' },
      { label: 'Чекпоинт синхронизирован', value: 'Контракт C-0147 · пакет v3 синхронизирован обратно в демо-оболочку', tone: 'pass' },
      { label: 'Измененные файлы / область', value: 'src/App.tsx, src/App.css · только демо-оболочка', tone: 'pass' },
      { label: 'Отметки', value: 'Снимки состояний + отметки привязки критериев', tone: 'mauve' },
      { label: 'Ручной шаг выполнен', value: 'Текст ручного решения проверен оператором', tone: 'amber' },
      { label: 'Артефакт приложен', value: 'Сводка изменения · заметки подтверждения · инструкции повтора', tone: 'pass' },
    ],
    verification: [
      {
        criterion: 'Переключатель репозитория фильтрует контракты без отдельного дашборд',
        support: 'Состояние фильтра и карточка выбранного контракта сохраняются в центральной панели',
        outcome: 'Покрыто',
      },
      {
        criterion: 'Готовность проекта находится вне цепочки контракта',
        support: 'Постоянная боковая панель с контекстом проекта, метрика готовности и чеклист',
        outcome: 'Покрыто',
      },
      {
        criterion: 'Задачи показывают тип, область, статус и обязанность по подтверждению',
        support: 'Три задачи, включая ручной шаг',
        outcome: 'Покрыто',
      },
      {
        criterion: 'Решение человека явно требуется перед финальным решением',
        support: 'Стадия решения с действиями принять, вернуть на доработку и заблокировать',
        outcome: 'Покрыто',
      },
    ],
    changed: [
      'В оболочку добавлены переключатель репозиториев и список контрактов с привязкой к репозиторию.',
      'Один выбранный контракт управляет центральной карточкой и пакетом изменения.',
      'Контекст проекта вынесен из потока и теперь содержит готовность к поставке.',
      'Финальное решение теперь требует видимого ручного шага.',
    ],
    unchanged: [
      'Нет бэкенд, API-вызовы, роутинг, авторизация, хранение или серверная логика.',
      'Нет реального сканирования репозитория, выполнение рантайма или синхронизация интеграций.',
      'Нет редизайна сверх ограниченной перестройки оболочки и текстовых правок.',
    ],
    trust: [
      'Все данные остаются в локальных мок-константах и состоянии интерфейса.',
      'Артефакты выполнения описаны как результат внешнего рантайма, а не как выполнение внутри Goalrail.',
      'Подтверждение показывает измененную и нетронутую область, чтобы не было расползания области.',
      'Финальный результат по-прежнему ждет решения человека.',
    ],
    howToVerify: [
      'Проверить сводку изменений в карточке выбранного контракта.',
      'Проверить список затронутых файлов в артефактах выполнения.',
      'Пройти состояние интерфейса через каждую стадию сценария.',
      'Подтвердить каждый критерий приемки на стадиях проверки и подтверждения.',
    ],
    activity: {
      0: [
        { kind: 'goal.intake', note: 'Контракт создан для trialops-demo', tone: 'mauve' },
        { kind: 'repo.bound', note: 'Контекст репозитория закреплен за trialops-demo', tone: 'pass' },
      ],
      1: [
        { kind: 'clarification.answered', note: '5 уточнений свернуты во входные данные контракта', tone: 'mauve' },
      ],
      2: [
        { kind: 'contract.drafted', note: 'Цель, область, критерии и ожидания по подтверждению собраны', tone: 'mauve' },
      ],
      3: [
        { kind: 'work-items.ready', note: 'Внешний рантайм, ручной шаг и каналы проверки и подтверждения объявлены', tone: 'pass' },
      ],
      4: [
        { kind: 'evidence.synced', note: 'Отметки внешнего рантайма прикреплены к пакету контракта', tone: 'pass' },
      ],
      5: [
        { kind: 'verification.covered', note: 'Критерии сопоставлены с именованными отметками артефактов', tone: 'pass' },
      ],
      6: [
        { kind: 'proof.ready', note: 'Измененная и неизменная область собраны для проверки', tone: 'pass' },
      ],
      7: [
        { kind: 'decision.pending', note: 'Решение человека требуется перед статусом Принято', tone: 'amber' },
      ],
    },
  },
  {
    id: 'C-0148',
    title: 'Фильтры CSV-экспорта',
    repo: 'trialops-demo',
    owner: 'Masha · руководитель поставки',
    scopeSurface: 'окно экспорта · текст чипов фильтра',
    summary: 'Выполнение идет, каналы подтверждения еще собирают отметки.',
    defaultStep: 4,
    goal: 'Сделать фильтры CSV-экспорта явными в демо-оболочке без изменения транспорта экспорта или хранения.',
    intakeNotes: ['Контракт остается привязанным к trialops-demo.', 'Текст экспорта требует ручной проверки.'],
    inScope: ['Чипы фильтров', 'Сводка выбора', 'Заметка о готовности экспорта'],
    outOfScope: ['Генерация CSV', 'Хранилище', 'Фоновые задачи'],
    acceptance: ['Выбранные фильтры можно проверить', 'Заметка по ручной проверке названия видна'],
    proofExpectations: ['Отметка для измененной области', 'Ручная проверка текста'],
    policyNote: 'Унаследованный контекст проекта: локально, только демо без реального рантайма экспорта.',
    clarifications: [
      {
        ref: 'filters',
        question: 'Какие фильтры важны в демо?',
        answer: 'Владелец, период и состояние',
        note: 'В области только поверхность интерфейса.',
      },
      {
        ref: 'naming',
        question: 'Кто согласует текст экспорта?',
        answer: 'Ручной проверяющий',
        note: 'Название остается ручным шагом.',
      },
    ],
    workItems: [
      {
        id: 'WI-11',
        title: 'Оболочка фильтров CSV-экспорта',
        lane: 'внешний рантайм',
        scope: 'Строка чипы фильтра и полоса сводки',
        status: 'В работе',
        proofObligation: 'Снимок состояния',
      },
      {
        id: 'WI-12',
        title: 'Согласование текста',
        lane: 'ручной шаг',
        scope: 'Понятный человеку лейбл экспорта',
        status: 'Только вручную',
        proofObligation: 'Заметка о согласовании оператора',
      },
    ],
    evidence: [
      { label: 'Исполнитель', value: 'Codex CLI', tone: 'mauve' },
      { label: 'Чекпоинт синхронизирован', value: 'Фикстура состояния фильтров экспорта', tone: 'pass' },
    ],
    verification: [
      { criterion: 'Фильтры остаются проверяемыми', support: 'Сводка фильтров в интерфейсе', outcome: 'Частично' },
    ],
    changed: ['Оформление фильтров в оболочке'],
    unchanged: ['Нет экспорт бэкенд'],
    trust: ['Только мок-поверхность'],
    howToVerify: ['Проверить сводку фильтров'],
    activity: {
      0: [{ kind: 'goal.intake', note: 'Запрос по фильтрам CSV-экспорта записан', tone: 'mauve' }],
      4: [{ kind: 'execution.running', note: 'Отметки еще прикрепляются', tone: 'amber' }],
    },
  },
  {
    id: 'C-0151',
    title: 'Правка цены переключателя',
    repo: 'trialops-demo',
    owner: 'Nika · продуктовые операции',
    scopeSurface: 'текст панели цены',
    summary: 'Пакет подтверждения готов и ждет решения человека.',
    defaultStep: 7,
    goal: 'Поправить текст цены переключателя в демо-оболочке и удержать релиз до ручного решения.',
    intakeNotes: ['Демо-обновление только в интерфейсе.'],
    inScope: ['Текст панели цены'],
    outOfScope: ['Логика биллинга'],
    acceptance: ['Текст можно проверить'],
    proofExpectations: ['Заметка о решении'],
    policyNote: 'Ручное решение остается обязательным перед принятием результата.',
    clarifications: [
      {
        ref: 'текст',
        question: 'Кто согласует формулировку?',
        answer: 'Ручной проверяющий',
        note: 'Нет автоматического принятия.',
      },
    ],
    workItems: [
      {
        id: 'WI-21',
        title: 'Правка текста',
        lane: 'ручной шаг',
        scope: 'Лейблы цены',
        status: 'Ждет решения',
        proofObligation: 'Решение проверяющего',
      },
    ],
    evidence: [{ label: 'Артефакт приложен', value: 'Заметка проверки текста', tone: 'pass' }],
    verification: [{ criterion: 'Текст изменен только в области', support: 'Заметка проверки', outcome: 'Покрыто' }],
    changed: ['Только текст цены'],
    unchanged: ['Поведение биллинга'],
    trust: ['Ручная точка принятия остается активной'],
    howToVerify: ['Прочитать изменение текста'],
    activity: {
      0: [{ kind: 'goal.intake', note: 'Запрос по тексту цены поставлен в очередь', tone: 'mauve' }],
      7: [{ kind: 'decision.pending', note: 'Решение проверяющего еще открыто', tone: 'amber' }],
    },
  },
  {
    id: 'C-0149',
    title: 'Усиление журнала аудита',
    repo: 'billing-api',
    owner: 'Roma · платформа',
    scopeSurface: 'рамка отметки аудита',
    summary: 'Пакет подтверждения собран и готов к проверке.',
    defaultStep: 6,
    goal: 'Показать журнал подтверждения аудита для контракта billing-api без намека на реальные серверные записи.',
    intakeNotes: ['Контекст репозитория — billing-api.'],
    inScope: ['Сводка подтверждения аудита', 'Лейблы отметки'],
    outOfScope: ['Записи в базе данных'],
    acceptance: ['Пакет подтверждения показывает, что изменилось и что не изменилось'],
    proofExpectations: ['Объяснение доверия'],
    policyNote: 'В этом демо billing-api сохраняет ту же политику локального рантайма.',
    clarifications: [
      {
        ref: 'аудит',
        question: 'Что должно показать подтверждение?',
        answer: 'Измененная область, неизменная область и причины доверия',
        note: 'В демо нет живого потока аудита.',
      },
    ],
    workItems: [
      {
        id: 'WI-31',
        title: 'Сводка подтверждения аудита',
        lane: 'проверка/подтверждение',
        scope: 'Текст пакета подтверждения',
        status: 'Подтверждение готово',
        proofObligation: 'Именованные причины доверия',
      },
    ],
    evidence: [{ label: 'Отметки', value: 'Карта области аудита + заметка подтверждения', tone: 'pass' }],
    verification: [{ criterion: 'Подтверждение можно проверить', support: 'Сводка подтверждения', outcome: 'Покрыто' }],
    changed: ['Оформление подтверждения аудита'],
    unchanged: ['Нет платежного рантайма'],
    trust: ['Подтверждение называет неизменную область'],
    howToVerify: ['Проверить сводку подтверждения'],
    activity: {
      0: [{ kind: 'goal.intake', note: 'Запрос на усиление журнала аудита записан', tone: 'mauve' }],
      6: [{ kind: 'proof.ready', note: 'Пакет подтверждения готов к проверке', tone: 'pass' }],
    },
  },
  {
    id: 'C-0150',
    title: 'Правка синхронизации лидов',
    repo: 'billing-api',
    owner: 'Ira · операции роста',
    scopeSurface: 'лейблы статуса синхронизации',
    summary: 'Контракт активен, задачи ограничены, но выполнение еще не начато.',
    defaultStep: 3,
    goal: 'Уточнить и ограничить правку синхронизации лидов в billing-api демо-канале без создания реальной интеграции синхронизации.',
    intakeNotes: ['Контракт все еще активен.'],
    inScope: ['Лейблы статуса', 'Текст ручной проверки'],
    outOfScope: ['Реальная синхронизация CRM'],
    acceptance: ['Лейблы статуса понятны'],
    proofExpectations: ['Чеклист проверки'],
    policyNote: 'В этом демо нет интеграционного рантайма.',
    clarifications: [
      {
        ref: 'синхронизация',
        question: 'Ожидается ли реальная интеграция?',
        answer: 'Нет, только мок-обновление оболочки',
        note: 'Выполнение остается только локально.',
      },
    ],
    workItems: [
      {
        id: 'WI-41',
        title: 'Уточнение области',
        lane: 'внешний рантайм',
        scope: 'Оболочка лейблов статуса',
        status: 'Активно',
        proofObligation: 'Сводка затронутой области',
      },
    ],
    evidence: [{ label: 'Чекпоинт синхронизирован', value: 'Только план задач', tone: 'amber' }],
    verification: [{ criterion: 'Область остается ограниченной', support: 'План задач', outcome: 'Частично' }],
    changed: ['Только план задач'],
    unchanged: ['Нет синхронизации CRM'],
    trust: ['Область контракта явно задана'],
    howToVerify: ['Проверить задачи'],
    activity: {
      0: [{ kind: 'goal.intake', note: 'Запрос на правку синхронизации лидов записан', tone: 'mauve' }],
      3: [{ kind: 'work-items.ready', note: 'Каналы выполнения подготовлены', tone: 'pass' }],
    },
  },
];

const PROOF_FEED: ProofFeedItem[] = [
  {
    id: 'PF-0147',
    contractId: 'C-0147',
    repo: 'trialops-demo',
    proofStatus: 'Ждет решения',
    decisionStatus: 'Точка принятия готова',
    humanApproval: 'Ждет решения человека',
    linkedEvidence: '5 связанных проверок',
    criteriaCoverage: '5/5 критериев покрыты',
    summary: 'Контрактная оболочка готов к финальному решению оператора.',
    tone: 'amber',
    changed: [
      'Список контрактов и выбранный пакет изменения привязаны к репозиторию.',
      'Контекст проекта остается вне цепочки контракта.',
      'Действие решение явно стоит перед финальным приемка.',
    ],
    unchanged: ['Нет бэкенд, роутинг, авторизация, хранение или реальной интеграционной работы.', 'Привязка репозитория не перенесена в цепочки контракта.'],
    verified: ['Критерии приемки сопоставлены с отметками артефактов.', 'Затронутая область остается ограниченной демо-оболочкой.', 'Решение человека остается в ожидании.'],
    decisionTrail: ['contract.drafted', 'evidence.synced', 'verification.covered', 'decision.pending'],
    archiveLine: 'архив://мок/C-0147 · хеш gr_pf_0147_a91c',
  },
  {
    id: 'PF-0148',
    contractId: 'C-0148',
    repo: 'trialops-demo',
    proofStatus: 'Сбор артефактов',
    decisionStatus: 'Вердикта пока нет',
    humanApproval: 'Не готово',
    linkedEvidence: '2 связанные проверки',
    criteriaCoverage: '2/5 критериев покрыты',
    summary: 'Поверхность фильтров CSV-экспорта имеет частичные отметки и открытую ручную проверку текста.',
    tone: 'mauve',
    changed: ['Оформление чип фильтра проверяется.', 'Отметка сводки выбора есть.'],
    unchanged: ['Нет генерации CSV, хранения или фоновых задач.'],
    verified: ['Заметка по области есть.', 'Ручная проверка названия еще открыта.'],
    decisionTrail: ['contract.active', 'execution.running', 'evidence.partial'],
    archiveLine: 'архив://мок/C-0148 · черновик хеша ожидает',
  },
  {
    id: 'PF-0082',
    contractId: 'C-0082',
    repo: 'billing-api',
    proofStatus: 'Принято',
    decisionStatus: 'Принято',
    humanApproval: 'Принято',
    linkedEvidence: '6 связанных проверок',
    criteriaCoverage: 'в архиве',
    summary: 'Текст отметки аудита биллинга принят и архивирован без изменения области рантайма.',
    tone: 'pass',
    changed: ['Формулировки отметки аудита и лейблы архив подтверждения уточнены.'],
    unchanged: ['Платежный рантайм, база данных и поведение платежей не менялись.'],
    verified: ['Архив подтверждения приложен.', 'Проверки области и целостности прошли.', 'Решение человека записано.'],
    decisionTrail: ['verification.covered', 'proof.archived', 'decision.accepted'],
    archiveLine: 'архив://мок/C-0082 · хеш gr_pf_0082_f43b',
  },
  {
    id: 'PF-0091',
    contractId: 'C-0091',
    repo: 'billing-api',
    proofStatus: 'Заблокировано',
    decisionStatus: 'Ошибка целостности',
    humanApproval: 'Проверяющий заблокировал',
    linkedEvidence: '3 связанные проверки',
    criteriaCoverage: 'ошибка целостности',
    summary: 'Изменение синхронизации лидов заблокировано: пакет артефактов не совпал с заявленной областью.',
    tone: 'block',
    changed: ['Лейблы статуса синхронизации вынесены на проверку.'],
    unchanged: ['Синхронизация CRM и поведение billing-api не затронуты.'],
    verified: ['Канал целостности не пройден.', 'Проверяющий заблокировал пакет.', 'Нужна доработка до архивного подтверждения.'],
    decisionTrail: ['evidence.synced', 'integrity.failed', 'decision.blocked'],
    archiveLine: 'архив://мок/C-0091 · заблокированный хеш gr_pf_0091_b7d0',
  },
];

const MOBILE_CONTRACT_QUEUE: MobileContractQueueItem[] = [
  {
    id: 'C-0147',
    title: 'Ручное решение',
    status: 'Активно',
    tone: 'mauve',
    stage: 'Входящий запрос',
    stageProgress: '1/8',
    policy: 'только локально',
    humanDecision: 'ждет решения',
    repo: 'trialops-demo',
    detail: {
      changePacket: 'Ручное решение показано как локальный демо-сценарий.',
      evidence: 'Критерии приемки и отметки затронутой области в очереди.',
      projectContext: 'У trialops-demo есть правила и доступна поверхность подтверждения.',
      decisionTrail: 'Входящий запрос открыт. Решение человека ждет подтверждения.',
    },
  },
  {
    id: 'C-0148',
    title: 'Фильтры CSV-экспорта',
    status: 'В работе',
    tone: 'amber',
    stage: 'Артефакты выполнения',
    stageProgress: '5/8',
    policy: 'только локально',
    humanDecision: 'не готово',
    repo: 'trialops-demo',
    detail: {
      changePacket: 'Текст чип фильтра проверяется в мок-оболочка.',
      evidence: 'Есть две отметки; ручная проверка текста еще открыта.',
      projectContext: 'trialops-demo остается выбранным контекстом репозитория.',
      decisionTrail: 'Выполнение идет. Сбор артефактов не завершен.',
    },
  },
  {
    id: 'C-0151',
    title: 'Правка цены переключателя',
    status: 'Ждет решения',
    tone: 'amber',
    stage: 'Решение',
    stageProgress: '8/8',
    policy: 'только локально',
    humanDecision: 'ждет решения',
    repo: 'trialops-demo',
    detail: {
      changePacket: 'Текст цены переключателя готов к ручной проверке.',
      evidence: 'Заметка проверки текста прикреплена к пакету подтверждения.',
      projectContext: 'Поведение биллинга и интеграция не входят в область.',
      decisionTrail: 'Подтверждение готово. Финальное решение нельзя принять одним тапом на телефоне.',
    },
  },
];

const MOBILE_REPO_QUEUE: MobileRepoQueueItem[] = [
  { repo: 'trialops-demo', readiness: '72/100', status: 'Готово', tone: 'pass' },
  { repo: 'billing-api', readiness: '58/100', status: 'Частично', tone: 'amber' },
  { repo: 'frontend-console', readiness: '41/100', status: 'Настройка', tone: 'block' },
];

const MOBILE_PROOF_QUEUE: MobileProofQueueItem[] = [
  { id: 'PF-0147', contractId: 'C-0147', status: 'Ждет решения', coverage: '5/5', tone: 'amber' },
  { id: 'PF-0148', contractId: 'C-0148', status: 'Сбор артефактов', coverage: '2/5', tone: 'mauve' },
  { id: 'PF-0082', contractId: 'C-0082', status: 'Принято', coverage: 'архив', tone: 'pass' },
  { id: 'PF-0091', contractId: 'C-0091', status: 'Заблокировано', coverage: 'ошибка целостности', tone: 'block' },
];

const INITIAL_STEPS = Object.fromEntries(CONTRACTS.map((contract) => [contract.id, contract.defaultStep])) as Record<string, StepIndex>;
const INITIAL_APPROVALS = Object.fromEntries(
  CONTRACTS.map((contract) => [contract.id, contract.defaultStep >= 7 ? 'pending' : 'pending']),
) as Record<string, ApprovalState>;

const CLARIFICATION_DELAYS = [240, 520, 820, 1120, 1400] as const;
const EVIDENCE_DELAY = 220;
const VERIFICATION_DELAY = 240;

function cx(...tokens: Array<string | false | null | undefined>) {
  return tokens.filter(Boolean).join(' ');
}

function getStatus(step: StepIndex, approval: ApprovalState) {
  if (approval === 'accepted') return 'Принято';
  if (approval === 'blocked') return 'Заблокировано';
  if (approval === 'rework') return 'Нужна доработка';
  if (step >= 7) return 'Ждет решения';
  if (step >= 6) return 'Подтверждение готово';
  if (step >= 4) return 'В работе';
  return 'Активно';
}

function getStatusTone(status: string): Tone {
  if (status === 'Принято' || status === 'Подтверждение готово') return 'pass';
  if (status === 'Ждет решения' || status === 'Нужна доработка') return 'amber';
  if (status === 'Заблокировано') return 'block';
  return 'mauve';
}

function getReadinessTone(score: number): Tone {
  if (score >= 70) return 'pass';
  if (score >= 50) return 'amber';
  return 'block';
}

function getCompactOwner(owner: string) {
  return owner.split('·')[0]?.trim() ?? owner;
}

function getShelfStatus(status: string) {
  if (status === 'Ждет решения') return 'Решение';
  if (status === 'Нужна доработка') return 'Доработка';
  if (status === 'Подтверждение готово') return 'Пакет подтверждения';
  return status;
}

function getCompactReadinessSignal(value: string) {
  return value
    .replace(/^docs\/context scan /, '')
    .replace(/^context scan /, '')
    .replace(/ not started$/, ' not started');
}

const ACTIVITY_KIND_LABELS: Record<string, string> = {
  'clarification.answered': 'уточнения закрыты',
  'context.scan': 'контекст просканирован',
  'contract.active': 'контракт активен',
  'contract.drafted': 'контракт собран',
  'contract.selected': 'контракт выбран',
  'decision.accepted': 'решение принято',
  'decision.blocked': 'решение заблокировано',
  'decision.pending': 'решение ожидает',
  'decision.state': 'состояние решения',
  'evidence.partial': 'артефакты частично',
  'evidence.synced': 'артефакты синхронизированы',
  'execution.running': 'выполнение идет',
  'goal.intake': 'запрос принят',
  'integrity.failed': 'целостность нарушена',
  'mode.recommended': 'режим выбран',
  'proof.archived': 'подтверждение в архиве',
  'proof.ready': 'подтверждение готово',
  'repo.bound': 'репозиторий привязан',
  'repo.context': 'контекст репозитория',
  'repo.selected': 'репозиторий выбран',
  'verification.covered': 'проверка покрыта',
  'work-items.ready': 'задачи готовы',
};

function formatActivityKind(kind: string) {
  return ACTIVITY_KIND_LABELS[kind] ?? kind;
}

function matchesQuery(values: string[], query: string) {
  const normalizedQuery = query.trim().toLowerCase();

  if (!normalizedQuery) return true;

  return values.some((value) => value.toLowerCase().includes(normalizedQuery));
}

function getMeters(step: StepIndex, approval: ApprovalState) {
  const contractPercent = [18, 42, 68, 76, 84, 90, 96, approval === 'accepted' ? 100 : 96][step];
  const executionPercent = [0, 0, 16, 42, 76, 82, 86, 86][step];
  const proofPercent = [0, 0, 10, 18, 42, 72, 90, approval === 'accepted' ? 100 : 92][step];

  return {
    contract: { percent: contractPercent, label: STAGES[Math.min(step, 2)].name },
    execution: {
      percent: executionPercent,
      label: step < 4 ? 'В очереди' : step < 6 ? 'Отметки получены' : 'Готово к проверке',
    },
    proof: {
      percent: proofPercent,
      label: approval === 'accepted' ? 'Принято' : step >= 7 ? 'Ждет решения' : step >= 6 ? 'Подтверждение готово' : 'Черновик',
    },
  };
}

function getStepSummary(step: StepIndex) {
  return (
    [
      'Показать путь одного запроса по репозиторию через сценарий контракта.',
      'Свести открытые вопросы к ограниченным ответам внутри контракта.',
      'Зафиксировать рабочий контракт до начала прогресса по задачам.',
      'Показать тип, область, статус и обязанность по подтверждению для каждой задачи.',
      'Артефакты выполнения собираются вне Goalrail и синхронизируются обратно.',
      'Проверка связывает критерии с артефактами вместо общего текста статуса.',
      'Подтверждение показывает, что изменилось, что не изменилось и почему этому можно доверять.',
      'Решение человека определяет финальный итог для контракта.',
    ] as const
  )[step];
}

function getActivity(contract: ContractRecord, step: StepIndex, approval: ApprovalState) {
  const timeline = [
    { ts: '09:42:08', kind: 'contract.selected', note: `${contract.id} закреплен в центральной карточке`, tone: 'mauve' as Tone },
    { ts: '09:42:12', kind: 'repo.context', note: `Контекст репозитория ${contract.repo} загружен`, tone: 'pass' as Tone },
  ];

  for (let index = 0; index <= step; index += 1) {
    const stageEvents = contract.activity[index] ?? [];
    stageEvents.forEach((event, eventIndex) => {
      timeline.push({
        ts: `09:${43 + index}:${String(8 + eventIndex * 7).padStart(2, '0')}`,
        kind: event.kind,
        note: event.note,
        tone: event.tone,
      });
    });
  }

  if (step >= 7) {
    timeline.push({
      ts: '09:50:12',
      kind: 'decision.state',
      note:
        approval === 'accepted'
          ? 'Решение человека записано · результат принят'
          : approval === 'blocked'
            ? 'Проверяющий заблокировал пакет'
            : approval === 'rework'
              ? 'Проверяющий запросил доработку'
              : 'Ждет решения человека перед финальным итогом',
      tone: approval === 'accepted' ? 'pass' : approval === 'blocked' ? 'block' : 'amber',
    });
  }

  return timeline;
}

function ListBlock({ title, items }: { title: string; items: string[] }) {
  return (
    <section className="detail-block">
      <div className="detail-kicker">{title}</div>
      <ul className="bullet-list">
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </section>
  );
}

function useMobileCompanionBreakpoint() {
  const getMatches = () => (typeof window === 'undefined' ? false : window.matchMedia('(max-width: 899px)').matches);
  const [isMobileCompanion, setIsMobileCompanion] = useState(getMatches);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(max-width: 899px)');
    const handleChange = () => setIsMobileCompanion(mediaQuery.matches);

    handleChange();

    if (typeof mediaQuery.addEventListener === 'function') {
      mediaQuery.addEventListener('change', handleChange);
      return () => mediaQuery.removeEventListener('change', handleChange);
    }

    mediaQuery.addListener(handleChange);
    return () => mediaQuery.removeListener(handleChange);
  }, []);

  return isMobileCompanion;
}

function MobileStat({ label, value, tone = 'mauve' }: { label: string; value: string; tone?: Tone }) {
  return (
    <div className={cx('mobile-stat', tone)}>
      <span>{label}</span>
      <b>{value}</b>
    </div>
  );
}

function MobileDetailSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <details className="mobile-detail" open>
      <summary>{title}</summary>
      <p>{children}</p>
    </details>
  );
}

function MobileContractsSurface({
  selectedContract,
  contracts = MOBILE_CONTRACT_QUEUE,
  contractsLabel = '3 активных контракта',
  repositoryLabel = 'trialops-demo',
  modeLabel = 'демо',
  onSelectContract,
}: {
  selectedContract: MobileContractQueueItem;
  contracts?: MobileContractQueueItem[];
  contractsLabel?: string;
  repositoryLabel?: string;
  modeLabel?: string;
  onSelectContract: (contractId: string) => void;
}) {
  return (
    <div className="mobile-surface" aria-label="Мобильный раздел контрактов">
      <section className="mobile-card">
        <div className="mobile-card-kicker">Контекст</div>
        <div className="mobile-stat-grid">
          <MobileStat label="репозиторий" value={repositoryLabel} />
          <MobileStat label="контракты" value={contractsLabel} tone={contracts.length > 0 ? 'pass' : 'amber'} />
          <MobileStat label="выбранный контракт" value={selectedContract.id} />
          <MobileStat label="статус" value={selectedContract.status} tone={selectedContract.tone} />
        </div>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-head">
          <div>
            <div className="mobile-card-kicker">Очередь контрактов</div>
            <h2>Выберите для проверки</h2>
          </div>
          <span className="status-pill mauve">{modeLabel}</span>
        </div>
        <div className="mobile-queue">
          {contracts.map((contract) => (
            <button
              key={contract.id}
              className={cx('mobile-queue-row', selectedContract.id === contract.id && 'active')}
              type="button"
              onClick={() => onSelectContract(contract.id)}
            >
              <span>{contract.id} · {contract.title} · {contract.status}</span>
            </button>
          ))}
        </div>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-kicker">Выбранный контракт</div>
        <h2>{selectedContract.title}</h2>
        <dl className="mobile-facts">
          <div>
            <dt>репозиторий</dt>
            <dd>{selectedContract.repo}</dd>
          </div>
          <div>
            <dt>текущая стадия</dt>
            <dd>{selectedContract.stage}</dd>
          </div>
          <div>
            <dt>политика</dt>
            <dd>{selectedContract.policy}</dd>
          </div>
          <div>
            <dt>решение человека</dt>
            <dd>{selectedContract.humanDecision}</dd>
          </div>
        </dl>
        <div className="mobile-stage-line">Стадия: {selectedContract.stage} · {selectedContract.stageProgress}</div>
      </section>

      <section className="mobile-card mobile-detail-stack">
        <MobileDetailSection title="Пакет изменения">{selectedContract.detail.changePacket}</MobileDetailSection>
        <MobileDetailSection title="Артефакты">{selectedContract.detail.evidence}</MobileDetailSection>
        <MobileDetailSection title="Контекст проекта">{selectedContract.detail.projectContext}</MobileDetailSection>
        <MobileDetailSection title="Цепочка решений">{selectedContract.detail.decisionTrail}</MobileDetailSection>
      </section>
    </div>
  );
}

function MobileReadinessSurface({ selectedRepo, onSelectRepo }: { selectedRepo: RepoId; onSelectRepo: (repo: RepoId) => void }) {
  const context = REPO_CONTEXTS[selectedRepo];

  return (
    <div className="mobile-surface" aria-label="Мобильный раздел готовности">
      <section className="mobile-card">
        <div className="mobile-card-kicker">Готовность</div>
        <div className="mobile-stat-grid">
          <MobileStat label="репозитории" value="3 репозитория" />
          <MobileStat label="среднее" value="57/100 среднее" tone="amber" />
          <MobileStat label="настройка" value="нужна настройка 1 репозитория" tone="block" />
        </div>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-head">
          <div>
            <div className="mobile-card-kicker">Очередь репозиториев</div>
            <h2>Готовность репозитория</h2>
          </div>
          <span className="status-pill mauve">проверка</span>
        </div>
        <div className="mobile-queue">
          {MOBILE_REPO_QUEUE.map((repo) => (
            <button
              key={repo.repo}
              className={cx('mobile-queue-row', selectedRepo === repo.repo && 'active')}
              type="button"
              onClick={() => onSelectRepo(repo.repo)}
            >
              <span>{repo.repo} · {repo.readiness} · {repo.status}</span>
            </button>
          ))}
        </div>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-kicker">Детали репозитория</div>
        <h2>{context.repo}</h2>
        <dl className="mobile-facts">
          <div>
            <dt>готовность</dt>
            <dd>{context.readiness}/100</dd>
          </div>
          <div>
            <dt>инициализация</dt>
            <dd>{context.init}</dd>
          </div>
          <div>
            <dt>документов</dt>
            <dd>{context.docsIndexed}</dd>
          </div>
          <div>
            <dt>тесты</dt>
            <dd>{getCompactReadinessSignal(context.testsStatus)}</dd>
          </div>
          <div>
            <dt>CI</dt>
            <dd>{context.ciStatus}</dd>
          </div>
          <div>
            <dt>Правила агента</dt>
            <dd>{context.ownersRulesStatus.replace('Правила агента ', '')}</dd>
          </div>
          <div>
            <dt>поверхность подтверждения</dt>
            <dd>{context.proofSurfaceStatus}</dd>
          </div>
          <div>
            <dt>рекомендованный режим</dt>
            <dd>{context.recommendedMode}</dd>
          </div>
        </dl>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-kicker">Действия</div>
        <div className="mobile-action-row">
          <button className="primary-button mobile-safe-button" type="button">
            Анализировать
          </button>
          <button className="ghost-button mobile-safe-button" type="button">
            Сканировать контекст
          </button>
          <button className="ghost-button mobile-safe-button secondary" type="button">
            Добавить репозиторий
          </button>
        </div>
        <p className="mobile-action-note">Настройку лучше делать на десктопе. Демо-кнопки ничего не подключают и не меняют.</p>
      </section>
    </div>
  );
}

function MobileProofSurface({
  selectedProof,
  onSelectProof,
}: {
  selectedProof: ProofFeedItem;
  onSelectProof: (proofId: string) => void;
}) {
  const proofDecisionReady = selectedProof.contractId === 'C-0147' && selectedProof.criteriaCoverage.startsWith('5/5');
  const archivedProof = selectedProof.proofStatus === 'Принято';
  const decisionRestriction = archivedProof ? 'Архивное подтверждение: только чтение' : 'Покрытие критериев неполное';

  return (
    <div className="mobile-surface" aria-label="Мобильный раздел подтверждений">
      <section className="mobile-card">
        <div className="mobile-card-head">
          <div>
            <div className="mobile-card-kicker">Очередь пакетов подтверждения</div>
            <h2>Проверка решения</h2>
          </div>
          <span className="status-pill amber">безопасная проверка</span>
        </div>
        <div className="mobile-queue">
          {MOBILE_PROOF_QUEUE.map((proof) => (
            <button
              key={proof.id}
              className={cx('mobile-queue-row', selectedProof.id === proof.id && 'active')}
              type="button"
              onClick={() => onSelectProof(proof.id)}
            >
              <span>{proof.contractId} · {proof.status} · {proof.coverage}</span>
            </button>
          ))}
        </div>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-kicker">Выбранный пакет подтверждения</div>
        <h2>{selectedProof.contractId}</h2>
        <dl className="mobile-facts">
          <div>
            <dt>контракт</dt>
            <dd>{selectedProof.contractId}</dd>
          </div>
          <div>
            <dt>репозиторий</dt>
            <dd>{selectedProof.repo}</dd>
          </div>
          <div>
            <dt>статус подтверждения</dt>
            <dd>{selectedProof.proofStatus}</dd>
          </div>
          <div>
            <dt>покрытие критериев</dt>
            <dd>{selectedProof.criteriaCoverage}</dd>
          </div>
          <div>
            <dt>решение человека</dt>
            <dd>{selectedProof.contractId === 'C-0147' ? 'Ожидает' : selectedProof.humanApproval}</dd>
          </div>
        </dl>
      </section>

      <section className="mobile-card mobile-detail-stack">
        <MobileDetailSection title="Что изменилось">{selectedProof.changed[0]}</MobileDetailSection>
        <MobileDetailSection title="Как проверено">{selectedProof.verified[0]}</MobileDetailSection>
        <MobileDetailSection title="Отметки">{selectedProof.linkedEvidence}</MobileDetailSection>
        <MobileDetailSection title="Цепочка решений">{selectedProof.decisionTrail.slice(0, 3).map(formatActivityKind).join(' · ')}</MobileDetailSection>
        <MobileDetailSection title="Архив подтверждения / хеш">{selectedProof.archiveLine}</MobileDetailSection>
      </section>

      <section className="mobile-card mobile-decision-card">
        <div className="mobile-card-kicker">Действие</div>
        {proofDecisionReady ? (
          <>
            <button className="primary-button mobile-safe-button" type="button">
              Проверить решение
            </button>
            <p className="mobile-action-note">Только безопасная проверка. Финальное решение нельзя принять одним тапом на телефоне.</p>
          </>
        ) : (
          <>
            <span className="status-pill amber">Решение недоступно</span>
            <p className="mobile-action-note">{decisionRestriction}</p>
          </>
        )}
      </section>
    </div>
  );
}

function MobileCompanionPreview({ liveContracts }: { liveContracts?: DemoContractsLiveData }) {
  const hasLiveContracts = Boolean(liveContracts);
  const [mobileSurface, setMobileSurface] = useState<ActiveSurface>(() => (liveContracts ? 'contracts' : 'proof'));
  const [selectedMobileContractId, setSelectedMobileContractId] = useState(() => liveContracts?.selectedContract?.id ?? 'C-0147');
  const [selectedMobileRepo, setSelectedMobileRepo] = useState<RepoId>('trialops-demo');
  const [selectedMobileProofId, setSelectedMobileProofId] = useState('PF-0147');

  const liveRepoLabel = liveContracts?.repositoryContext?.contexts[0]?.repo_binding.repository_full_name ?? 'backend';
  const liveContractRecords = useMemo(() => {
    if (!liveContracts) {
      return [];
    }

    return liveContracts.contracts.map((contract) => (
      mapLiveContract(contract, liveContracts.repositoryContext, liveContracts.selectedDraft)
    ));
  }, [liveContracts]);
  const mobileContracts = hasLiveContracts
    ? (liveContractRecords.length > 0 ? liveContractRecords.map(mapMobileContractQueueItem) : [mapMobileContractQueueItem(EMPTY_LIVE_CONTRACT)])
    : MOBILE_CONTRACT_QUEUE;
  const selectedMobileContract = mobileContracts.find((contract) => contract.id === selectedMobileContractId) ?? mobileContracts[0];
  const selectedMobileProof = PROOF_FEED.find((proof) => proof.id === selectedMobileProofId) ?? PROOF_FEED[0];

  useEffect(() => {
    if (!hasLiveContracts) {
      return;
    }

    const nextContractId = liveContracts?.selectedContract?.id ?? mobileContracts[0]?.id;
    if (nextContractId && !mobileContracts.some((contract) => contract.id === selectedMobileContractId)) {
      setSelectedMobileContractId(nextContractId);
    }
  }, [hasLiveContracts, liveContracts?.selectedContract?.id, mobileContracts, selectedMobileContractId]);

  const handleMobileContractSelect = (contractId: string) => {
    setSelectedMobileContractId(contractId);
    const selectedBackendContract = liveContractRecords.find((contract) => contract.id === contractId)?.backendContract;
    if (selectedBackendContract) {
      liveContracts?.onContractSelect(selectedBackendContract);
    }
  };

  return (
    <main className="mobile-companion">
      <section className="mobile-hero">
        <div className="mobile-brand">Goalrail</div>
        <h1>Goalrail</h1>
        <p>Короткий режим проверки: контракты, готовность и подтверждения.</p>
        <span>Полная консоль оператора доступна на десктопе.</span>
      </section>

      <nav className="mobile-segmented" aria-label="Разделы мобильного режима">
        <button className={cx(mobileSurface === 'contracts' && 'active')} type="button" onClick={() => setMobileSurface('contracts')}>
          Контракты
        </button>
        <button className={cx(mobileSurface === 'readiness' && 'active')} type="button" onClick={() => setMobileSurface('readiness')}>
          Готовность
        </button>
        <button className={cx(mobileSurface === 'proof' && 'active')} type="button" onClick={() => setMobileSurface('proof')}>
          Итог
        </button>
      </nav>

      {mobileSurface === 'contracts' ? (
        <MobileContractsSurface
          contracts={mobileContracts}
          contractsLabel={hasLiveContracts ? `${liveContracts?.contracts.length ?? 0} контрактов` : '3 активных контракта'}
          modeLabel={hasLiveContracts ? 'Read-only API' : 'демо'}
          repositoryLabel={hasLiveContracts ? liveRepoLabel : 'trialops-demo'}
          selectedContract={selectedMobileContract}
          onSelectContract={handleMobileContractSelect}
        />
      ) : mobileSurface === 'readiness' ? (
        <MobileReadinessSurface selectedRepo={selectedMobileRepo} onSelectRepo={setSelectedMobileRepo} />
      ) : (
        <MobileProofSurface selectedProof={selectedMobileProof} onSelectProof={setSelectedMobileProofId} />
      )}
    </main>
  );
}

function renderStageContent({
  contract,
  projectContext,
  step,
  approval,
  visibleClarifications,
  visibleEvidence,
  visibleVerification,
  onAdvance,
  onDecision,
}: {
  contract: ContractRecord;
  projectContext: RepoContext;
  step: StepIndex;
  approval: ApprovalState;
  visibleClarifications: number;
  visibleEvidence: number;
  visibleVerification: number;
  onAdvance: () => void;
  onDecision: (decision: ApprovalState) => void;
}) {
  if (step === 0) {
    return (
      <div className="stage-content">
        <div className="compat-line">Входящий запрос</div>
        <div className="detail-grid two-up">
          <section className="detail-block">
            <div className="detail-kicker">Входящий запрос</div>
            <h2 className="stage-title">{contract.title}</h2>
            <p className="detail-copy">{contract.goal}</p>
            <ul className="bullet-list">
              {contract.intakeNotes.map((note) => (
                <li key={note}>{note}</li>
              ))}
            </ul>
          </section>

          <section className="detail-block">
            <div className="detail-kicker">Пакет запроса</div>
            <dl className="key-grid compact-grid">
              <div>
                <dt>Репозиторий</dt>
                <dd>{contract.repo}</dd>
              </div>
              <div>
                <dt>Контракт</dt>
                <dd>{contract.id}</dd>
              </div>
              <div>
                <dt>Раздел</dt>
                <dd>{contract.scopeSurface}</dd>
              </div>
              <div>
                <dt>Политика</dt>
                <dd>{projectContext.runtimePolicy}</dd>
              </div>
            </dl>
            <div className="panel-note">
              Привязка репозитория уже есть, но это <b>контекст проекта</b>, а не стадия pipeline.
            </div>
          </section>
        </div>
      </div>
    );
  }

  if (step === 1) {
    return (
      <div className="stage-content">
        <div className="compat-line">Уточнения · {contract.clarifications.length} из {contract.clarifications.length}</div>
        <div className="clarification-stack">
          {contract.clarifications.map((card, index) => {
            const pinned = index < visibleClarifications;

            return (
              <article key={`${contract.id}-${card.ref}`} className={cx('clarification-card', pinned && 'resolved')}>
                <div className="clarification-head">
                  <span>Q{index + 1}</span>
                  <span>{card.ref}</span>
                </div>
                <div className="clarification-q">{card.question}</div>
                <div className="clarification-a">{card.answer}</div>
                <div className="clarification-note">{card.note}</div>
                <div className={cx('clarification-foot', pinned && 'resolved')}>{pinned ? 'Ответ закреплен в контракте' : 'Ожидает закрепления в контракте'}</div>
              </article>
            );
          })}
        </div>
      </div>
    );
  }

  if (step === 2) {
    return (
      <div className="stage-content">
        <div className="compat-line">Рабочий контракт · черновик v3</div>
        <section className="detail-block hero-block">
          <div className="detail-kicker">Цель</div>
          <p className="detail-copy">{contract.goal}</p>
        </section>

        <div className="detail-grid two-up">
          <ListBlock title="В области" items={contract.inScope} />
          <ListBlock title="Вне области" items={contract.outOfScope} />
          <ListBlock title="Критерии приемки" items={contract.acceptance} />
          <ListBlock title="Ожидания по подтверждению" items={contract.proofExpectations} />
        </div>

        <section className="detail-block">
          <div className="detail-kicker">Контекст проекта / политика</div>
          <p className="detail-copy">{contract.policyNote}</p>
          <div className="inline-actions">
            <button className="ghost-button" type="button" onClick={onAdvance}>
              Зафиксировать контракт
            </button>
            <button className="primary-button small" type="button" onClick={onAdvance}>
              Утвердить контракт
            </button>
          </div>
        </section>
      </div>
    );
  }

  if (step === 3) {
    return (
      <div className="stage-content">
        <div className="section-tagline">Задачи</div>
        <div className="work-item-list">
          {contract.workItems.map((item) => (
            <article key={item.id} className="work-item-card">
              <div className="work-item-head">
                <div>
                  <div className="work-item-id">{item.id}</div>
                  <div className="work-item-title">{item.title}</div>
                </div>
                <div className={cx('status-pill', getStatusTone(item.status))}>{item.status}</div>
              </div>
              <dl className="key-grid work-grid">
                <div>
                  <dt>Тип</dt>
                  <dd>{item.lane}</dd>
                </div>
                <div>
                  <dt>Область / раздел</dt>
                  <dd>{item.scope}</dd>
                </div>
                <div>
                  <dt>Статус</dt>
                  <dd>{item.status}</dd>
                </div>
                <div>
                  <dt>Обязанность по подтверждению</dt>
                  <dd>{item.proofObligation}</dd>
                </div>
              </dl>
            </article>
          ))}
        </div>
      </div>
    );
  }

  if (step === 4) {
    return (
      <div className="stage-content">
        <div className="section-tagline">Артефакты выполнения</div>
        <div className="panel-note strong-note">
          Выполнение прошло <b>вне Goalrail</b>. Goalrail сохраняет синхронизированные артефакты для выбранного контракта, а не чат-лог.
        </div>
        <div className="evidence-grid">
          {contract.evidence.slice(0, visibleEvidence).map((item) => (
            <article key={`${contract.id}-${item.label}`} className="evidence-card">
              <div className="detail-kicker">{item.label}</div>
              <div className={cx('evidence-value', item.tone)}>{item.value}</div>
            </article>
          ))}
        </div>
      </div>
    );
  }

  if (step === 5) {
    return (
      <div className="stage-content">
        <div className="section-tagline">Проверка</div>
        <div className="verify-list">
          {contract.verification.slice(0, visibleVerification).map((row) => (
            <article key={`${contract.id}-${row.criterion}`} className="verify-row">
              <div className="verify-main">
                <div className="verify-criterion">{row.criterion}</div>
                <div className="verify-support">{row.support}</div>
              </div>
              <div className="verify-side">
                <div className="detail-kicker">Итог</div>
                <div className="status-pill pass">{row.outcome}</div>
              </div>
            </article>
          ))}
        </div>
      </div>
    );
  }

  if (step === 6) {
    return (
      <div className="stage-content">
        <div className="section-tagline">Пакет подтверждения</div>
        <div className="detail-grid two-up">
          <ListBlock title="Что изменилось" items={contract.changed} />
          <ListBlock title="Что не изменилось" items={contract.unchanged} />
        </div>
        <ListBlock title="Почему результату можно доверять" items={contract.trust} />
      </div>
    );
  }

  const approvalLabel =
    approval === 'accepted'
      ? 'Принято'
      : approval === 'blocked'
        ? 'Заблокировано'
        : approval === 'rework'
          ? 'Нужна доработка'
          : 'Ждет решения';

  return (
    <div className="stage-content">
      <div className="section-tagline">Решение</div>
      <div className="approval-state-row">
        <div className="detail-kicker">Состояние решения</div>
        <div className={cx('status-pill', getStatusTone(approvalLabel))}>{approvalLabel}</div>
      </div>

      <div className="detail-grid two-up">
        <ListBlock title="Что изменилось" items={contract.changed} />
        <ListBlock title="Что не изменилось" items={contract.unchanged} />
        <ListBlock title="Как проверить" items={contract.howToVerify} />
        <ListBlock title="Ожидания по подтверждению" items={contract.proofExpectations} />
      </div>

      <section className="detail-block">
        <div className="detail-kicker">Ручное решение</div>
        <div className="decision-actions">
          <button className="primary-button" type="button" onClick={() => onDecision('accepted')}>
            Принять результат
          </button>
          <button className="ghost-button" type="button" onClick={() => onDecision('rework')}>
            Вернуть на доработку
          </button>
          <button className="ghost-button danger" type="button" onClick={() => onDecision('blocked')}>
            Заблокировать
          </button>
        </div>
      </section>
    </div>
  );
}

function DeliveryReadinessSurface({ selectedRepo, onSelectRepo }: { selectedRepo: RepoId; onSelectRepo: (repo: RepoId) => void }) {
  return (
    <main className="canvas surface-canvas">
      <section className="object surface-object">
        <div className="obj-head">
          <div>
            <div className="t">Готовность</div>
            <div className="object-title">Настройка репозитория и режим работы</div>
          </div>
          <div className="tags">
            <span className="tag mauve">Раздел рабочей области</span>
            <span className="tag">Только демо</span>
          </div>
        </div>

        <div className="obj-body">
          <div className="surface-intro">
            <div className="section-tagline">Готовность · уровень репозитория и проекта</div>
            <p className="detail-copy">
              Этот раздел показывает подключенные репозитории, сигналы готовности, действия настройки и рекомендованный режим работы. Это не
              стадия цепочки контракта.
            </p>
          </div>

          <div className="repo-card-grid">
            {WORKSPACE_REPOS.map((repo) => {
              const context = REPO_CONTEXTS[repo];

              return (
                <button
                  key={repo}
                  className={cx('repo-readiness-card', selectedRepo === repo && 'active')}
                  type="button"
                  onClick={() => onSelectRepo(repo)}
                >
                  <div className="repo-card-head">
                    <div>
                      <div className="contract-id">{context.repo}</div>
                      <div className="repo-score">{context.readiness}/100 готовность</div>
                    </div>
                    <span className={cx('status-pill', getReadinessTone(context.readiness))}>{context.init}</span>
                  </div>

                  <dl className="key-grid readiness-card-grid">
                    <div>
                      <dt>Сканирование</dt>
                      <dd>{context.scanStatus}</dd>
                    </div>
                    <div>
                      <dt>Тесты</dt>
                      <dd>{context.testsStatus}</dd>
                    </div>
                    <div>
                      <dt>CI</dt>
                      <dd>{context.ciStatus}</dd>
                    </div>
                    <div>
                      <dt>Владельцы/правила</dt>
                      <dd>{context.ownersRulesStatus}</dd>
                    </div>
                    <div>
                      <dt>Поверхность подтверждения</dt>
                      <dd>{context.proofSurfaceStatus}</dd>
                    </div>
                    <div>
                      <dt>Режим</dt>
                      <dd>{context.recommendedMode}</dd>
                    </div>
                  </dl>
                </button>
              );
            })}

            <article className="repo-readiness-card add-repository-card">
              <div className="repo-card-head">
                <div>
                  <div className="contract-id">Добавить репозиторий</div>
                  <div className="repo-score">Подключить следующий репозиторий</div>
                </div>
                <span className="status-pill mauve">Действие настройки</span>
              </div>
              <p className="detail-copy">Подключить репозиторий, выполнить инициализацию, просканировать контекст и посчитать готовность.</p>
              <button className="ghost-button" type="button">
                Добавить репозиторий
              </button>
            </article>
          </div>
        </div>
      </section>
    </main>
  );
}

function DeliveryReadinessInspector({ selectedRepo }: { selectedRepo: RepoId }) {
  const context = REPO_CONTEXTS[selectedRepo];

  return (
    <aside className="sidepanel">
      <section className="panel-card">
        <div className="panel-head">
          <div className="t">Детали репозитория</div>
          <div className="id">{context.repo}</div>
        </div>

        <dl className="key-grid">
          <div>
            <dt>Готовность</dt>
            <dd>{context.readiness}/100</dd>
          </div>
          <div>
            <dt>Инициализация</dt>
            <dd>{context.init}</dd>
          </div>
          <div>
            <dt>Документов</dt>
            <dd>{context.docsIndexed}</dd>
          </div>
          <div>
            <dt>Режим</dt>
            <dd>{context.recommendedMode}</dd>
          </div>
        </dl>

        <div className="readiness-block">
          <div className="row">
            <div className="label">Готовность</div>
            <div className="val">{context.readiness}/100</div>
          </div>
          <div className={cx('bar', `bar-${getReadinessTone(context.readiness)}`)}>
            <i style={{ width: `${context.readiness}%` }} />
          </div>
        </div>

        <div className="checklist-block">
          <div className="detail-kicker">Сигналы готовности</div>
          <div className="check-row">
            <span>Сканирование контекста</span>
            <span className="check-value mauve">{getCompactReadinessSignal(context.scanStatus)}</span>
          </div>
          {context.checklist.map((item) => (
            <div key={`${context.repo}-${item.label}`} className="check-row">
              <span>{item.label}</span>
              <span className={cx('check-value', item.tone)}>{item.value}</span>
            </div>
          ))}
          <div className="check-row">
            <span>Поверхность подтверждения</span>
            <span className={cx('check-value', getReadinessTone(context.readiness))}>{context.proofSurfaceStatus}</span>
          </div>
        </div>

        <div className="detail-kicker">Демо-действия</div>
        <div className="decision-actions">
          <button className="ghost-button" type="button">
            Анализировать
          </button>
          <button className="ghost-button" type="button">
            Запустить инициализацию
          </button>
          <button className="ghost-button" type="button">
            Сканировать контекст
          </button>
        </div>
      </section>

      <section className="panel-card compact-card">
        <div className="panel-head">
          <div className="t">Граница раздела</div>
          <div className="id">Только настройка и готовность</div>
        </div>
        <p className="panel-copy">
          Добавление репозитория относится к готовности. Оно не открывает реальную интеграцию и не становится шагом потока контракта.
        </p>
      </section>
    </aside>
  );
}

function DeliveryReadinessBottomPanel({ selectedRepo }: { selectedRepo: RepoId }) {
  const context = REPO_CONTEXTS[selectedRepo];

  return (
    <section className="bottompanel">
      <section className="panel-card activity-card">
        <div className="panel-head">
          <div className="t">Активность готовности</div>
          <div className="id">Демо-события настройки</div>
        </div>
        <div className="activity-list">
          {[
            ['09:31:02', 'repo.selected', `${context.repo} выбран для деталей готовности`, 'mauve'],
            ['09:31:12', 'context.scan', context.scanStatus, getReadinessTone(context.readiness)],
            ['09:31:22', 'mode.recommended', context.recommendedMode, getReadinessTone(context.readiness)],
          ].map(([ts, kind, note, tone]) => (
            <div key={`${ts}-${kind}`} className="activity-row">
              <div className="activity-ts">{ts}</div>
              <div className="activity-body">
                <div className="activity-kind">{formatActivityKind(kind)}</div>
                <div className="activity-note">{note}</div>
              </div>
              <div className={cx('status-pill', tone as Tone)}>{tone === 'pass' ? 'готово' : tone === 'block' ? 'настройка' : 'проверка'}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="panel-card control-card">
        <div className="panel-head">
          <div className="t">Управление разделом</div>
          <div className="id">Нет реальных интеграций</div>
        </div>
        <div className="control-copy">
          Анализ, инициализация, сканирование контекста и добавление репозитория — демо-действия. Они не вызывают бэкенд и не меняют постоянное состояние.
        </div>
        <div className="control-meta">
          <span>Репозиторий: {context.repo}</span>
          <span>Готовность: {context.readiness}/100</span>
        </div>
      </section>
    </section>
  );
}

function ProofFeedSurface({ selectedProofId, onSelectProof }: { selectedProofId: string; onSelectProof: (proofId: string) => void }) {
  return (
    <main className="canvas surface-canvas">
      <section className="object surface-object">
        <div className="obj-head">
          <div>
            <div className="t">Подтверждения</div>
            <div className="object-title">Артефакты и решения по контрактам</div>
          </div>
          <div className="tags">
            <span className="tag pass">Все репозитории по умолчанию</span>
            <span className="tag">Не чат-лог</span>
          </div>
        </div>

        <div className="obj-body">
          <div className="surface-intro">
            <div className="section-tagline">Подтверждения · обзор по контрактам и репозиториям</div>
            <p className="detail-copy">
              Область по умолчанию — все репозитории. Чипы репозиториев остаются статическими демо-контролы, чтобы раздел читался как контроль подтверждений на уровне рабочей области.
            </p>
            <div className="chip-row filter-row">
              <span className="tag pass">Все репозитории</span>
              <span className="tag">trialops-demo</span>
              <span className="tag">billing-api</span>
              <span className="tag">frontend-console</span>
            </div>
          </div>

          <div className="proof-feed-list">
            {PROOF_FEED.map((item) => (
              <button
                key={item.id}
                className={cx('proof-feed-row', selectedProofId === item.id && 'active')}
                type="button"
                onClick={() => onSelectProof(item.id)}
              >
                <div className="proof-row-head">
                  <div className="proof-row-title">
                    <span className="contract-id">{item.contractId}</span>
                    <span className="repo-badge">{item.repo}</span>
                  </div>
                  <span className={cx('status-pill', item.tone)}>{item.proofStatus}</span>
                </div>
                <div className="proof-summary">{item.summary}</div>
                <dl className="key-grid proof-meta-grid">
                  <div>
                    <dt>Решение</dt>
                    <dd>{item.decisionStatus}</dd>
                  </div>
                  <div>
                    <dt>Решение человека</dt>
                    <dd>{item.humanApproval}</dd>
                  </div>
                  <div>
                    <dt>Артефакты</dt>
                    <dd>{item.linkedEvidence}</dd>
                  </div>
                  <div>
                    <dt>Покрытие</dt>
                    <dd>{item.criteriaCoverage}</dd>
                  </div>
                </dl>
              </button>
            ))}
          </div>
        </div>
      </section>
    </main>
  );
}

function ProofFeedInspector({ selectedProof }: { selectedProof: ProofFeedItem }) {
  return (
    <aside className="sidepanel">
      <section className="panel-card proof-detail-card">
        <div className="panel-head">
          <div className="t">Детали пакета подтверждения</div>
          <div className="id">{selectedProof.contractId} · {selectedProof.repo}</div>
        </div>
        <ListBlock title="Что изменилось" items={selectedProof.changed.slice(0, 2)} />
        <ListBlock title="Что не изменилось" items={selectedProof.unchanged.slice(0, 2)} />
        <ListBlock title="Как проверено" items={selectedProof.verified.slice(0, 2)} />
        <ListBlock title="Цепочка решений" items={selectedProof.decisionTrail.slice(0, 2).map(formatActivityKind)} />
        <div className="panel-note proof-archive-line">
          <b>Архив подтверждения / хеш</b>
          <br />
          {selectedProof.archiveLine}
        </div>
      </section>

      <section className="panel-card compact-card">
        <div className="panel-head">
          <div className="t">Область ленты</div>
          <div className="id">Все репозитории по умолчанию</div>
        </div>
        <p className="panel-copy">
          Эта лента — обзор уровня рабочей области по контрактам и репозиториям. Она не привязана к текущему переключателю репозитория.
        </p>
      </section>
    </aside>
  );
}

function ProofFeedBottomPanel({ selectedProof }: { selectedProof: ProofFeedItem }) {
  return (
    <section className="bottompanel">
      <section className="panel-card activity-card">
        <div className="panel-head">
          <div className="t">Активность ленты подтверждений</div>
          <div className="id">Только артефакты и решения</div>
        </div>
        <div className="activity-list">
          {selectedProof.decisionTrail.map((entry, index) => (
            <div key={`${selectedProof.id}-${entry}`} className="activity-row">
              <div className="activity-ts">10:{String(12 + index).padStart(2, '0')}:04</div>
              <div className="activity-body">
                <div className="activity-kind">{formatActivityKind(entry)}</div>
                <div className="activity-note">{selectedProof.contractId} · {selectedProof.repo}</div>
              </div>
              <div className={cx('status-pill', selectedProof.tone)}>{selectedProof.tone === 'block' ? 'блок' : selectedProof.tone === 'pass' ? 'готово' : 'событие'}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="panel-card control-card">
        <div className="panel-head">
          <div className="t">Управление лентой</div>
          <div className="id">Статическая область</div>
        </div>
        <div className="control-copy">
          Чипы репозиториев — только демо. Разбор статусов находится слева в очереди подтверждений, поэтому лента остается между контрактами и между репозиториями.
        </div>
        <div className="control-meta">
          <span>Подтверждение: {selectedProof.contractId}</span>
          <span>Область по умолчанию: все репозитории</span>
        </div>
      </section>
    </section>
  );
}

function DesktopConsole({ liveContracts }: { liveContracts?: DemoContractsLiveData }) {
  const hasLiveContracts = Boolean(liveContracts);
  const [activeSurface, setActiveSurface] = useState<ActiveSurface>('contracts');
  const [repoFilter, setRepoFilter] = useState<RepoFilter>(() => (liveContracts ? LIVE_ALL_REPOSITORIES_FILTER : 'trialops-demo'));
  const [repoSelectorOpen, setRepoSelectorOpen] = useState(false);
  const [contractSearch, setContractSearch] = useState('');
  const [repoSearch, setRepoSearch] = useState('');
  const [proofSearch, setProofSearch] = useState('');
  const [selectedContractId, setSelectedContractId] = useState('C-0147');
  const [selectedReadinessRepo, setSelectedReadinessRepo] = useState<RepoId>('trialops-demo');
  const [selectedProofId, setSelectedProofId] = useState(PROOF_FEED[0].id);
  const [contractSteps, setContractSteps] = useState<Record<string, StepIndex>>(INITIAL_STEPS);
  const [approvalStates, setApprovalStates] = useState<Record<string, ApprovalState>>(INITIAL_APPROVALS);
  const [visibleClarifications, setVisibleClarifications] = useState(0);
  const [visibleEvidence, setVisibleEvidence] = useState(0);
  const [visibleVerification, setVisibleVerification] = useState(0);

  const liveRepoOptions = useMemo(() => makeLiveRepoOptions(liveContracts?.repositoryContext ?? null), [liveContracts?.repositoryContext]);
  const liveRepoContexts = useMemo(() => makeLiveRepoContexts(liveContracts?.repositoryContext ?? null), [liveContracts?.repositoryContext]);
  const liveContractRecords = useMemo(() => {
    if (!liveContracts) {
      return [];
    }

    return liveContracts.contracts.map((contract) => (
      mapLiveContract(contract, liveContracts.repositoryContext, liveContracts.selectedDraft)
    ));
  }, [liveContracts]);
  const contractSource = hasLiveContracts ? liveContractRecords : CONTRACTS;
  const repoOptions = hasLiveContracts ? liveRepoOptions : REPO_OPTIONS;
  const selectedLiveContractId = liveContracts?.selectedContract?.id;
  const selectedLiveContract = selectedLiveContractId
    ? liveContractRecords.find((contract) => contract.id === selectedLiveContractId)
    : null;

  const repoScopedContracts = useMemo(() => {
    return repoFilter === LIVE_ALL_REPOSITORIES_FILTER
      ? contractSource
      : contractSource.filter((contract) => (contract.repoFilterValue ?? contract.repo) === repoFilter);
  }, [contractSource, repoFilter]);

  const visibleContracts = useMemo(() => {
    return repoScopedContracts.filter((contract) =>
      matchesQuery([contract.id, contract.title, contract.repo, contract.owner, contract.summary, getStatus(contractSteps[contract.id] ?? contract.defaultStep, approvalStates[contract.id] ?? 'pending')], contractSearch),
    );
  }, [approvalStates, contractSearch, contractSteps, repoScopedContracts]);

  const visibleRepos = useMemo(() => {
    return WORKSPACE_REPOS.filter((repo) => {
      const context = REPO_CONTEXTS[repo];
      return matchesQuery([context.repo, context.init, context.scanStatus, context.testsStatus, context.ciStatus, context.ownersRulesStatus, context.recommendedMode], repoSearch);
    });
  }, [repoSearch]);

  const visibleProofs = useMemo(() => {
    return PROOF_FEED.filter((item) =>
      matchesQuery([item.contractId, item.repo, item.proofStatus, item.decisionStatus, item.humanApproval, item.criteriaCoverage, item.summary], proofSearch),
    );
  }, [proofSearch]);

  useEffect(() => {
    if (hasLiveContracts) {
      if (selectedLiveContractId) {
        setSelectedContractId(selectedLiveContractId);
        return;
      }
      setSelectedContractId(repoScopedContracts[0]?.id ?? EMPTY_LIVE_CONTRACT.id);
      return;
    }

    if (!repoScopedContracts.some((contract) => contract.id === selectedContractId)) {
      const nextContract = repoScopedContracts[0];
      if (nextContract) {
        setSelectedContractId(nextContract.id);
        return;
      }

      setSelectedContractId(CONTRACTS[0].id);
    }
  }, [hasLiveContracts, repoScopedContracts, selectedContractId, selectedLiveContractId]);

  useEffect(() => {
    if (selectedLiveContractId) {
      setSelectedContractId(selectedLiveContractId);
    }
  }, [selectedLiveContractId]);

  useEffect(() => {
    if (hasLiveContracts) {
      setRepoFilter(liveContracts?.repoBindingFilter ?? LIVE_ALL_REPOSITORIES_FILTER);
    }
  }, [hasLiveContracts, liveContracts?.repoBindingFilter]);

  const selectedContract = useMemo(() => {
    if (selectedLiveContract) {
      return selectedLiveContract;
    }
    return contractSource.find((contract) => contract.id === selectedContractId)
      ?? (hasLiveContracts ? EMPTY_LIVE_CONTRACT : CONTRACTS[0]);
  }, [contractSource, hasLiveContracts, selectedContractId, selectedLiveContract]);

  const step = contractSteps[selectedContract.id] ?? selectedContract.defaultStep;
  const approval = approvalStates[selectedContract.id] ?? 'pending';
  const selectedStatus = getStatus(step, approval);
  const projectContext =
    (selectedContract.repoFilterValue ? liveRepoContexts[selectedContract.repoFilterValue] : undefined)
    ?? REPO_CONTEXTS[selectedContract.repo as RepoId]
    ?? Object.values(liveRepoContexts)[0]
    ?? REPO_CONTEXTS['trialops-demo'];
  const meters = getMeters(step, approval);
  const activity = useMemo(() => getActivity(selectedContract, step, approval), [selectedContract, step, approval]);
  const selectedProof = useMemo(() => PROOF_FEED.find((item) => item.id === selectedProofId) ?? PROOF_FEED[0], [selectedProofId]);
  const averageReadiness = Math.round(WORKSPACE_REPOS.reduce((sum, repo) => sum + REPO_CONTEXTS[repo].readiness, 0) / WORKSPACE_REPOS.length);
  const acceptedProofs = PROOF_FEED.filter((item) => item.proofStatus === 'Принято').length;
  const blockedProofs = PROOF_FEED.filter((item) => item.proofStatus === 'Заблокировано').length;
  const topbarMeters =
    activeSurface === 'contracts'
      ? [
          { tone: 'amber' as Tone, label: 'Контракт', value: meters.contract.label, percent: meters.contract.percent },
          { tone: 'mauve' as Tone, label: 'Выполнение', value: meters.execution.label, percent: meters.execution.percent },
          { tone: 'pass' as Tone, label: 'Подтверждение', value: meters.proof.label, percent: meters.proof.percent },
        ]
      : activeSurface === 'readiness'
        ? [
            { tone: 'mauve' as Tone, label: 'Рабочая область', value: `${WORKSPACE_REPOS.length} репозитория`, percent: 100 },
            { tone: 'amber' as Tone, label: 'Готовность', value: `${averageReadiness}/100 среднее`, percent: averageReadiness },
            { tone: 'pass' as Tone, label: 'Настройка', value: 'Только демо-действия', percent: 72 },
          ]
        : [
            { tone: 'mauve' as Tone, label: 'Подтверждения', value: 'Все репозитории', percent: 100 },
            { tone: 'amber' as Tone, label: 'Ожидают', value: '2 активных', percent: 50 },
            { tone: 'pass' as Tone, label: 'Решения', value: `${acceptedProofs} принято · ${blockedProofs} блок`, percent: 70 },
          ];

  useEffect(() => {
    if (step === 1) {
      setVisibleClarifications(0);
      const timers = CLARIFICATION_DELAYS.slice(0, selectedContract.clarifications.length).map((delay, index) =>
        window.setTimeout(() => {
          setVisibleClarifications(index + 1);
        }, delay),
      );

      return () => {
        timers.forEach((timer) => window.clearTimeout(timer));
      };
    }

    setVisibleClarifications(step > 1 ? selectedContract.clarifications.length : 0);

    return undefined;
  }, [selectedContract, step]);

  useEffect(() => {
    if (step === 4) {
      setVisibleEvidence(0);
      const timers = selectedContract.evidence.map((_, index) =>
        window.setTimeout(() => {
          setVisibleEvidence(index + 1);
        }, (index + 1) * EVIDENCE_DELAY),
      );

      return () => {
        timers.forEach((timer) => window.clearTimeout(timer));
      };
    }

    setVisibleEvidence(step > 4 ? selectedContract.evidence.length : 0);

    return undefined;
  }, [selectedContract, step]);

  useEffect(() => {
    if (step === 5) {
      setVisibleVerification(0);
      const timers = selectedContract.verification.map((_, index) =>
        window.setTimeout(() => {
          setVisibleVerification(index + 1);
        }, (index + 1) * VERIFICATION_DELAY),
      );

      return () => {
        timers.forEach((timer) => window.clearTimeout(timer));
      };
    }

    setVisibleVerification(step > 5 ? selectedContract.verification.length : 0);

    return undefined;
  }, [selectedContract, step]);

  const setStepForSelected = (nextStep: StepIndex) => {
    setContractSteps((current) => ({ ...current, [selectedContract.id]: nextStep }));
  };

  const goNext = () => {
    if (step < 7) {
      setStepForSelected((step + 1) as StepIndex);
    }
  };

  const goBack = () => {
    if (step > 0) {
      setStepForSelected((step - 1) as StepIndex);
    }
  };

  const resetSelected = () => {
    setContractSteps((current) => ({ ...current, [selectedContract.id]: selectedContract.defaultStep }));
    setApprovalStates((current) => ({ ...current, [selectedContract.id]: 'pending' }));
  };

  const handleDecision = (decision: ApprovalState) => {
    setApprovalStates((current) => ({ ...current, [selectedContract.id]: decision }));
    setStepForSelected(7);
  };

  const selectedRepoOption = repoOptions.find((option) => option.value === repoFilter) ?? repoOptions[0];

  const handleRepoFilterSelect = (nextRepoFilter: RepoFilter) => {
    setRepoFilter(nextRepoFilter);
    if (hasLiveContracts) {
      liveContracts?.onRepoBindingFilterChange(nextRepoFilter);
    }
    setRepoSelectorOpen(false);
  };

  const primaryActionLabel =
    step === 0
      ? 'Начать'
      : step === 1
        ? 'К контракту'
        : step === 2
          ? 'Зафиксировать контракт'
          : step === 3
            ? 'К артефактам'
            : step === 4
              ? 'К проверке'
              : step === 5
                ? 'К подтверждению'
                : step === 6
                  ? 'К решению'
                  : approval === 'accepted'
                    ? 'Повторить'
                    : null;

  const activeSurfaceLabel = activeSurface === 'contracts' ? 'Контракты' : activeSurface === 'readiness' ? 'Готовность' : 'Подтверждения';
  const topbarStateLabel = activeSurface === 'contracts' ? 'Статус' : activeSurface === 'readiness' ? 'Репозиторий' : 'Область';
  const topbarStateValue = activeSurface === 'contracts' ? selectedStatus : activeSurface === 'readiness' ? selectedReadinessRepo : 'Все репозитории';

  return (
    <div className="app-shell" data-step={step}>
      <div className="app">
        <header className="topbar">
          <div className="brand">
            <div className="mark" aria-hidden="true">
              <span />
            </div>
            <div className="name">
              Goalrail <span className="dot">·</span> <span className="brand-muted">консоль</span>
            </div>
          </div>

          <div className="meters">
            {topbarMeters.map((meter) => (
              <div key={meter.label} className={cx('meter', meter.tone)}>
                <div className="row">
                  <div className="label">{meter.label}</div>
                  <div className="val">{meter.value}</div>
                </div>
                <div className="bar">
                  <i style={{ width: `${meter.percent}%` }} />
                </div>
              </div>
            ))}
          </div>

          <div className="topbar-state">
            <div className="state-chip">
              <span className="k">Раздел</span>
              <span className="v" title={activeSurfaceLabel}>
                {activeSurfaceLabel}
              </span>
            </div>
            <div className="state-chip">
              <span className="k">{topbarStateLabel}</span>
              <span className="v" title={topbarStateValue}>
                {topbarStateValue}
              </span>
            </div>
          </div>
        </header>

        <aside className="rail">
          <div className="rail-main">
            <div className="group-label">Рабочая область</div>
            <div className="surface-switcher" aria-label="Разделы рабочей области">
              <button
                className={cx('surface-switch', activeSurface === 'contracts' && 'active')}
                type="button"
                onClick={() => setActiveSurface('contracts')}
              >
                Контракты
              </button>
              <button
                className={cx('surface-switch', activeSurface === 'readiness' && 'active')}
                type="button"
                onClick={() => setActiveSurface('readiness')}
              >
                Готовность
              </button>
              <button
                className={cx('surface-switch', activeSurface === 'proof' && 'active')}
                type="button"
                onClick={() => setActiveSurface('proof')}
              >
                Подтверждения
              </button>
            </div>

            <div className="group-label">Контекст раздела</div>
            <div className="rail-section surface-context">
              <div className="surface-context-title">
                {activeSurface === 'contracts' ? 'Контракты' : activeSurface === 'readiness' ? 'Готовность' : 'Подтверждения'}
              </div>
              <div className="rail-note">
                {activeSurface === 'contracts'
                  ? repoFilter === 'all'
                    ? 'Активная работа по всем репозиториям'
                    : 'Работа по выбранному репозиторию'
                  : activeSurface === 'readiness'
                    ? 'Настройка репозитория'
                    : 'Артефакты по репозиториям'}
              </div>

              {activeSurface === 'contracts' ? (
                <div className="surface-control">
                  <div className="select-wrap">
                    <span className="select-label">Репозиторий</span>
                    <div className="repo-select">
                      <button
                        className={cx('repo-select-trigger', repoSelectorOpen && 'open')}
                        type="button"
                        aria-haspopup="listbox"
                        aria-expanded={repoSelectorOpen}
                        onClick={() => setRepoSelectorOpen((open) => !open)}
                      >
                        <span>{selectedRepoOption.label}</span>
                        <i aria-hidden="true" />
                      </button>
                      {repoSelectorOpen ? (
                        <div className="repo-select-menu" role="listbox" aria-label="Переключатель репозитория">
                          {repoOptions.map((option) => (
                            <button
                              key={option.value}
                              className={cx('repo-select-option', option.value === repoFilter && 'active')}
                              type="button"
                              role="option"
                              aria-selected={option.value === repoFilter}
                              onClick={() => handleRepoFilterSelect(option.value)}
                            >
                              <span>{option.label}</span>
                              <b>
                                {option.value === LIVE_ALL_REPOSITORIES_FILTER
                                  ? 'все'
                                  : contractSource.filter((contract) => (contract.repoFilterValue ?? contract.repo) === option.value).length}
                              </b>
                            </button>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  </div>
                  {hasLiveContracts ? (
                    <div className="select-wrap">
                      <span className="select-label">Статус</span>
                      <select
                        aria-label="Статус контрактов"
                        className="native-select"
                        value={liveContracts?.stateFilter ?? 'all'}
                        onChange={(event) => liveContracts?.onStateFilterChange(event.target.value as ContractResponse['state'] | 'all')}
                      >
                        {LIVE_CONTRACT_STATE_OPTIONS.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </select>
                    </div>
                  ) : null}
                  {hasLiveContracts ? (
                    <button className="live-refresh-button" type="button" onClick={liveContracts?.onRefresh}>
                      {liveContracts?.contractListLoadStatus === 'loading' ? 'Обновляем' : 'Обновить'}
                    </button>
                  ) : null}
                  <div className="repo-hint-list">
                    {hasLiveContracts ? (
                      <>
                        <span>Backend discovery · {liveContracts?.contracts.length ?? 0} контрактов</span>
                        <span>{liveContracts?.repositoryContext?.contexts.length ?? 0} repository context</span>
                        <span>Read-only endpoints · без mutation controls</span>
                      </>
                    ) : (
                      <>
                        <span>trialops-demo · 3 контракта</span>
                        <span>billing-api · 2 контракта</span>
                        <span>Все репозитории доступны</span>
                      </>
                    )}
                  </div>
                  {hasLiveContracts && liveContracts?.contractListError ? (
                    <div className="live-error" role="alert">{liveContracts.contractListError}</div>
                  ) : null}
                </div>
              ) : null}
            </div>

            {activeSurface === 'contracts' ? (
              <>
                <div className="group-label">Активные контракты</div>
                <div className="shelf-tools" aria-label="Инструменты списка контрактов">
                  <input aria-label="Искать контракты" placeholder="Искать контракты" value={contractSearch} onChange={(event) => setContractSearch(event.target.value)} />
                  <div className="filter-chip-row" aria-label="Фильтры контрактов">
                    <span>Активно</span>
                    <span>В работе</span>
                    <span>Решение</span>
                  </div>
                </div>
                <div className="contract-list" aria-label="Активные контракты">
                  {visibleContracts.map((contract) => {
                    const contractStep = contractSteps[contract.id] ?? contract.defaultStep;
                    const contractApproval = approvalStates[contract.id] ?? 'pending';
                    const status = getStatus(contractStep, contractApproval);
                    const isSelected = contract.id === selectedContract.id;

                    return (
                      <button
                        key={contract.id}
                        className={cx('contract-row', isSelected && 'active', !isSelected && 'compact')}
                        type="button"
                        onClick={() => {
                          setSelectedContractId(contract.id);
                          if (hasLiveContracts && contract.backendContract) {
                            liveContracts?.onContractSelect(contract.backendContract);
                          }
                        }}
                      >
                        <div className="contract-row-top">
                          <span className="contract-id">{contract.id}</span>
                          <span className={cx('status-pill', getStatusTone(status))}>{getShelfStatus(status)}</span>
                        </div>
                        <div className="contract-title">{contract.title}</div>
                        {isSelected ? <div className="contract-summary">{contract.summary}</div> : null}
                        <div className="contract-row-meta">
                          {isSelected ? (
                            <>
                              <span className="repo-badge">{contract.repo}</span>
                              <span className="contract-owner">{contract.owner}</span>
                            </>
                          ) : (
                            <span className="contract-compact-meta">
                              {contract.repo} · {getCompactOwner(contract.owner)}
                            </span>
                          )}
                        </div>
                      </button>
                    );
                  })}
                  {visibleContracts.length === 0 ? (
                    <div className="rail-empty">
                      {hasLiveContracts && liveContracts?.contractListLoadStatus === 'loading'
                        ? 'Загружаем контракты из backend'
                        : hasLiveContracts && liveContracts?.contractListLoadStatus === 'error'
                          ? 'Backend contract discovery недоступен'
                          : hasLiveContracts
                            ? 'Backend не вернул контракты по текущему фильтру'
                            : 'Контракты не найдены'}
                    </div>
                  ) : null}
                </div>
              </>
            ) : activeSurface === 'readiness' ? (
              <>
                <div className="group-label">Репозитории</div>
                <div className="shelf-tools" aria-label="Инструменты списка репозиториев">
                  <input aria-label="Искать репозитории" placeholder="Искать репозитории" value={repoSearch} onChange={(event) => setRepoSearch(event.target.value)} />
                  <div className="filter-chip-row" aria-label="Фильтры репозиториев">
                    <span>Готово</span>
                    <span>Частично</span>
                    <span>Настройка</span>
                  </div>
                </div>
                <div className="surface-mini-list" role="group" aria-label="Список готовности репозиториев">
                  {visibleRepos.map((repo) => (
                    <button
                      key={repo}
                      className={cx('surface-mini-row', selectedReadinessRepo === repo && 'active')}
                      type="button"
                      onClick={() => setSelectedReadinessRepo(repo)}
                    >
                      <span>{repo}</span>
                      <b>{REPO_CONTEXTS[repo].readiness}/100</b>
                    </button>
                  ))}
                  {visibleRepos.length === 0 ? <div className="rail-empty">Репозитории не найдены</div> : null}
                </div>
              </>
            ) : (
              <>
                <div className="group-label">Очередь пакетов подтверждения</div>
                <div className="shelf-tools" aria-label="Инструменты ленты подтверждений">
                  <input aria-label="Искать подтверждения" placeholder="Искать подтверждения" value={proofSearch} onChange={(event) => setProofSearch(event.target.value)} />
                  <div className="filter-chip-row" aria-label="Фильтры подтверждений">
                    <span>Ожидают</span>
                    <span>Принято</span>
                    <span>Заблокировано</span>
                  </div>
                </div>
                <div className="surface-mini-list" role="group" aria-label="Очередь подтверждений">
                  {visibleProofs.map((item) => (
                    <button
                      key={item.id}
                      className={cx('surface-mini-row', selectedProof.id === item.id && 'active')}
                      type="button"
                      onClick={() => setSelectedProofId(item.id)}
                    >
                      <span>{item.contractId}</span>
                      <b>{item.proofStatus}</b>
                    </button>
                  ))}
                  {visibleProofs.length === 0 ? <div className="rail-empty">Пакеты подтверждения не найдены</div> : null}
                </div>
              </>
            )}
          </div>

          <div className="case">
            <div className="k">Режим</div>
            <div className="v">{hasLiveContracts ? 'Backend-backed Contracts' : 'Демо-разделы рабочей области'}</div>
            <div className="sub">
              {hasLiveContracts
                ? 'Read-only API · без workflow mutation controls'
                : 'Только локальное демо-состояние · без бэкенда · без роутинга'}
            </div>
          </div>
        </aside>

        {activeSurface === 'contracts' ? (
          <>
        <main className="canvas">
          <section className="spine">
            <div className="spine-head">
              <div>
                <div className="t">Контракт {selectedContract.id} · пакет изменения</div>
                <div className="id">Цепочка изменения · cp-{selectedContract.id.slice(2).toLowerCase()}</div>
              </div>
              <div className="tags">
                <span className="tag mauve">{selectedContract.repo}</span>
                <span className={cx('tag', getStatusTone(selectedStatus))}>{selectedStatus}</span>
              </div>
            </div>

            <div className="body">
              {STAGES.map((stage, index) => {
                const stateClass = step > index ? 'done' : step === index ? 'active' : '';

                return (
                  <div key={stage.id} className={cx('stage', stateClass)}>
                    <div className="node" />
                    <div className="connector" />
                    <div className="name">{stage.name}</div>
                    <div className="meta">{step > index ? 'готово' : step === index ? 'сейчас' : 'ждет'}</div>
                  </div>
                );
              })}
            </div>

            <div className="active-summary">
              <div className="marker">Активная стадия</div>
              <div className="stage-name">{STAGES[step].name}</div>
              <div className="facts">
                <div className="f">
                  Репозиторий <b>{selectedContract.repo}</b>
                </div>
                <div className="f">
                  Контракт <b>{selectedContract.id}</b>
                </div>
                <div className="f pass">
                  Статус <b>{selectedStatus}</b>
                </div>
              </div>
            </div>
          </section>

          <section className="object">
            <div className="obj-head">
              <div>
                <div className="t">Выбранный контракт</div>
                <div className="object-title">{selectedContract.id} · {selectedContract.title}</div>
              </div>
              <div className="tags">
                <span className="tag">{selectedContract.scopeSurface}</span>
                <span className="tag">{selectedContract.owner}</span>
              </div>
            </div>
            <div className="obj-body">
              {renderStageContent({
                contract: selectedContract,
                projectContext,
                step,
                approval,
                visibleClarifications,
                visibleEvidence,
                visibleVerification,
                onAdvance: goNext,
                onDecision: handleDecision,
              })}
            </div>
          </section>
        </main>

        <aside className="sidepanel">
          <section className="panel-card">
            <div className="panel-head">
              <div className="t">Контекст проекта</div>
              <div className="id">Репозиторий {projectContext.repo}</div>
            </div>
            <dl className="key-grid">
              <div>
                <dt>Репозиторий</dt>
                <dd>{projectContext.repo}</dd>
              </div>
              <div>
                <dt>Привязан</dt>
                <dd>{projectContext.bound}</dd>
              </div>
              <div>
                <dt>Инициализация</dt>
                <dd>{projectContext.init}</dd>
              </div>
              <div>
                <dt>Документов</dt>
                <dd>{projectContext.docsIndexed}</dd>
              </div>
            </dl>

            <div className="readiness-block">
              <div className="row">
                <div className="label">Готовность</div>
                <div className="val">{projectContext.readiness}/100</div>
              </div>
              <div className="bar">
                <i style={{ width: `${projectContext.readiness}%` }} />
              </div>
            </div>

            <div className="checklist-block">
              <div className="detail-kicker">Чеклист готовности</div>
              {projectContext.checklist.map((item) => (
                <div key={`${projectContext.repo}-${item.label}`} className="check-row">
                  <span>{item.label}</span>
                  <span className={cx('check-value', item.tone)}>{item.value}</span>
                </div>
              ))}
            </div>

            <div className="detail-kicker">Политика рантайма</div>
            <div className="panel-copy">{projectContext.runtimePolicy}</div>
            <div className="detail-kicker top-gap">Доступные рантаймы</div>
            <div className="chip-row">
              {projectContext.runtimes.map((runtime) => (
                <span key={runtime} className="tag">
                  {runtime}
                </span>
              ))}
            </div>

            <div className="selection-strip">
              <span>{selectedContract.id}</span>
              <span>{STAGES[step].name}</span>
              <span>{selectedStatus}</span>
            </div>
          </section>

          <section className="panel-card inspector-card">
            <div className="panel-head">
              <div className="t">Уточнения</div>
              <div className="id">Главные вводные · {selectedContract.clarifications.length} всего</div>
            </div>
            <div className="inspector-list">
              {selectedContract.clarifications.slice(0, 3).map((card, index) => {
                const resolved = step > 1 || index < visibleClarifications;
                return (
                  <div key={`${selectedContract.id}-inspector-${card.ref}`} className="inspector-row">
                    <div>
                      <div className="inspector-term">{card.ref}</div>
                      <div className="inspector-note">{card.answer}</div>
                    </div>
                    <div className={cx('status-pill', resolved ? 'pass' : 'amber')}>{resolved ? 'Закрыто' : 'Открыто'}</div>
                  </div>
                );
              })}
            </div>
            {selectedContract.clarifications.length > 3 ? (
              <div className="inspector-foot">{selectedContract.clarifications.length - 3} вводных остаются в деталях контракта.</div>
            ) : null}
          </section>
        </aside>

        <section className="bottompanel">
          <section className="panel-card activity-card">
            <div className="panel-head">
              <div className="t">Активность рабочей области</div>
              <div className="id">Не чат-лог · только события контракта</div>
            </div>
            <div className="activity-list">
              {activity.map((entry, index) => (
                <div key={`${entry.ts}-${entry.kind}-${index}`} className="activity-row">
                  <div className="activity-ts">{entry.ts}</div>
                  <div className="activity-body">
                    <div className="activity-kind">{formatActivityKind(entry.kind)}</div>
                    <div className="activity-note">{entry.note}</div>
                  </div>
                  <div className={cx('status-pill', entry.tone)}>{entry.tone === 'pass' ? 'готово' : entry.tone === 'block' ? 'блок' : entry.tone === 'amber' ? 'проверка' : 'событие'}</div>
                </div>
              ))}
            </div>
          </section>

          <section className="panel-card control-card">
            <div className="panel-head">
              <div className="t">{hasLiveContracts ? 'Backend endpoints' : 'Управление стадией'}</div>
              <div className="id">{hasLiveContracts ? 'Read-only surface' : 'Только демо-проход'}</div>
            </div>
            <div className="control-copy">
              {hasLiveContracts
                ? 'Страница читает list/detail/current-draft/repository-context endpoints и не выполняет lifecycle mutations.'
                : getStepSummary(step)}
            </div>
            <div className="control-meta">
              <span>Репозиторий: {repoFilter === 'all' ? 'Все репозитории' : repoFilter}</span>
              <span>Репозиторий карточки: {selectedContract.repo}</span>
            </div>
            <div className="control-actions">
              {hasLiveContracts ? (
                <button className="primary-button" type="button" onClick={liveContracts?.onRefresh}>
                  Обновить из backend
                </button>
              ) : (
                <>
                  <button className="ghost-button" type="button" onClick={goBack} disabled={step === 0}>
                    Назад
                  </button>
                  {primaryActionLabel ? (
                    <button className="primary-button" type="button" onClick={step === 7 ? resetSelected : goNext}>
                      {primaryActionLabel}
                    </button>
                  ) : null}
                  <button className="ghost-button" type="button" onClick={resetSelected}>
                    Сбросить
                  </button>
                </>
              )}
            </div>
          </section>
        </section>
          </>
        ) : activeSurface === 'readiness' ? (
          <>
            <DeliveryReadinessSurface selectedRepo={selectedReadinessRepo} onSelectRepo={setSelectedReadinessRepo} />
            <DeliveryReadinessInspector selectedRepo={selectedReadinessRepo} />
            <DeliveryReadinessBottomPanel selectedRepo={selectedReadinessRepo} />
          </>
        ) : (
          <>
            <ProofFeedSurface selectedProofId={selectedProof.id} onSelectProof={setSelectedProofId} />
            <ProofFeedInspector selectedProof={selectedProof} />
            <ProofFeedBottomPanel selectedProof={selectedProof} />
          </>
        )}
      </div>
    </div>
  );
}

function DemoContractsApp({ liveContracts }: { liveContracts?: DemoContractsLiveData }) {
  const isMobileCompanion = useMobileCompanionBreakpoint();

  return isMobileCompanion ? <MobileCompanionPreview liveContracts={liveContracts} /> : <DesktopConsole liveContracts={liveContracts} />;
}

export default function DemoContractsPage({ liveContracts }: { liveContracts?: DemoContractsLiveData }) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const [shadowRoot, setShadowRoot] = useState<ShadowRoot | null>(null);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) {
      return;
    }

    setShadowRoot(host.shadowRoot ?? host.attachShadow({ mode: 'open' }));
  }, []);

  return (
    <>
      <div ref={hostRef} className="demoContractsHost" style={{ display: 'block', minHeight: '100vh' }} />
      {shadowRoot
        ? createPortal(
            <>
              <style>{demoContractsShadowCss}</style>
              <div id="root">
                <DemoContractsApp liveContracts={liveContracts} />
              </div>
            </>,
            shadowRoot
          )
        : null}
    </>
  );
}

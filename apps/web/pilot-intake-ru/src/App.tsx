import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react';

import './App.css';

const MAX_LEN = 1500;
const MIN_LEN = 20;

const topChips = ['ПИЛОТ ОТКРЫТ', 'РУЧНОЙ ФОРМАТ', 'РЕАЛЬНЫЙ КЕЙС'];
const primaryNav = ['Пилот', 'Что это', 'Как работает', 'Демо'];
const contextNav = ['Небольшие команды', 'Реальная разработка', 'ИИ в разработке'];
const statusLines = ['Ручное сопровождение', 'Реальный кейс'];

const demoSteps = ['Запрос', 'Уточнения', 'Контракт', 'Проверка', 'Итог'];

const exampleChips: string[] = [
  'Добавить manual review перед proof approval',
  'Разобрать PR с неясными критериями приёмки',
  'Оценить готовность repo к AI-assisted delivery',
];

const seeItems = [
  'уточняющие вопросы по вашей задаче',
  'черновик контракта в ваших рамках',
  'критерии приёмки',
  'риски и пробелы',
  'честный следующий шаг',
];

const safeItems = [
  'код не выполняется',
  'repo не подключается',
  'данные не сохраняются как production-сущности',
  'изменений в вашей среде не происходит',
];

const clarificationItems = [
  'фиксируем границы задачи',
  'выявляем ambiguity',
  'не переходим к исполнению слишком рано',
  'готовим черновик контракта',
];

const pilotSteps = [
  'разбираем задачу',
  'формируем рабочий контракт',
  'ведём исполнение в заданных рамках',
  'проверяем результат',
  'фиксируем итог пилота',
];
const exclusions = [
  'не обещаем гарантированный результат',
  'не продаём «полный автопилот»',
  'не заменяем трекер задач или среду разработки',
  'не требуем долгого внедрения',
];

type DemoStep = 'intake' | 'clarification' | 'contract' | 'review' | 'outcome';
type Scenario = 'manual_review_gate' | 'bounded_task';

const contractPreviewItems = [
  'цель задачи',
  'scope и out of scope',
  'критерии приёмки',
  'открытые ambiguity',
  'следующий шаг',
];

const reviewPreviewItems = [
  'готовность scope',
  'decision rule',
  'failure mode',
  'риски и ambiguity',
  'следующий шаг',
];

const outcomePreviewItems = [
  'честный verdict',
  'что стало ясно',
  'что осталось открытым',
  'следующий шаг',
  'без выполнения кода',
];

const outcomeNotDone = [
  'код не выполнялся',
  'repo не подключался',
  'production-сущности не создавались',
  'результат не является выполненной задачей',
];

const learnedRG = [
  'задача относится к approval / proof flow',
  'review gate можно описать как контрактное правило',
  'failure mode должен быть явным до проверки',
  'open ambiguity влияет на готовность к пилоту',
];

const learnedBT = [
  'входной запрос можно превратить в bounded contract',
  'scope определяет пригодность кейса для пилота',
  'критерии и риски лучше фиксировать до исполнения',
  'широкий scope снижает готовность к пилоту',
];

type ReviewStatus = 'ok' | 'warning' | 'blocking';
type RiskSeverity = 'warning' | 'blocking' | 'advisory';

interface ReviewItem {
  id: string;
  label: string;
  detail: string;
  status: ReviewStatus;
}

interface RiskItem {
  id: string;
  title: string;
  detail: string;
  severity: RiskSeverity;
}

interface ReviewReport {
  readinessLabel: string;
  readinessSummary: string;
  readinessItems: ReviewItem[];
  riskItems: RiskItem[];
  ambiguityItems: OpenAmbiguity[];
  nextStep: string;
}

const STATUS_LABEL: Record<ReviewStatus, string> = {
  ok: 'ГОТОВО',
  warning: 'ОГОВОРКА',
  blocking: 'БЛОКЕР',
};

const SEVERITY_LABEL: Record<RiskSeverity, string> = {
  advisory: 'СПРАВКА',
  warning: 'ОГОВОРКА',
  blocking: 'БЛОКЕР',
};

type OutcomeTone = 'ready' | 'readyWithCaveats' | 'blocked';

interface OutcomeReport {
  tone: OutcomeTone;
  verdictLabel: string;
  verdictTitle: string;
  verdictBody: string;
  whatWeLearned: string[];
  whatRemainsOpen: string[];
  recommendedNextStep: string;
  notDone: string[];
  ctaLabel: string;
}

function deriveOutcomeTone(review: ReviewReport): OutcomeTone {
  const hasBlocking =
    review.readinessItems.some((it) => it.status === 'blocking') ||
    review.riskItems.some((r) => r.severity === 'blocking');
  if (hasBlocking) return 'blocked';
  const hasWarning =
    review.readinessItems.some((it) => it.status === 'warning') ||
    review.riskItems.some((r) => r.severity === 'warning') ||
    review.ambiguityItems.length > 0;
  if (hasWarning) return 'readyWithCaveats';
  return 'ready';
}

interface OpenAmbiguity {
  id: string;
  text: string;
}

interface ContractDraft {
  scenario: Scenario;
  title: string;
  summary: string;
  goal: string;
  scope: string;
  reviewerRule?: string;
  failureMode?: string;
  outOfScope: string[];
  acceptanceCriteria: string[];
  openAmbiguities: OpenAmbiguity[];
  nextStep: string;
}

function buildContractDraft({
  scenario,
  answers,
}: {
  scenario: Scenario;
  answers: Record<string, string>;
  intakeText: string;
}): ContractDraft {
  const questions = QUESTIONS[scenario];
  const optionLabel = (questionId: string) => {
    const q = questions.find((x) => x.id === questionId);
    const optId = answers[questionId];
    return q?.options.find((o) => o.id === optId)?.label;
  };

  const ambiguities: OpenAmbiguity[] = [];
  const pushAmbiguity = (text: string) => {
    ambiguities.push({ id: `A-${String(ambiguities.length + 1).padStart(2, '0')}`, text });
  };

  if (scenario === 'manual_review_gate') {
    const scopeOpt = answers['mrg-scope'];
    const decisionOpt = answers['mrg-decision'];
    const failOpt = answers['mrg-fail'];

    const scopeText = ((): string => {
      switch (scopeOpt) {
        case 'new-only':
          return 'Новые контракты в рамках одного demo-case.';
        case 'new-active':
          return 'Новые и активные контракты, но migration policy нужно уточнить.';
        case 'repo-scoped':
          return 'Repo-scoped контракты в рамках demo-contract.';
        case 'unsure':
          return 'Scope пока не определён.';
        default:
          return 'Scope пока не указан.';
      }
    })();

    if (scopeOpt === 'unsure') pushAmbiguity('Scope гейта пока не определён.');
    if (scopeOpt === 'new-active') pushAmbiguity('Нужна отдельная политика для уже активных контрактов.');
    if (decisionOpt === 'tbd') pushAmbiguity('Кто принимает review decision, пока не определено.');
    if (failOpt === 'manual-decision') pushAmbiguity('Нужно описать, кто принимает ручное решение и где оно фиксируется.');

    return {
      scenario,
      title: 'Ручной review gate перед proof approval',
      summary:
        'GoalRail подготовил локальный черновик контракта на основе запроса и уточнений. В реальном пилоте этот шаг проверяется оператором.',
      goal: 'Заблокировать proof approval до ручного review decision.',
      scope: scopeText,
      reviewerRule: optionLabel('mrg-decision') ?? 'не выбрано',
      failureMode: optionLabel('mrg-fail') ?? 'не выбрано',
      outOfScope: [
        'автоматическое выполнение кода',
        'подключение реального repo',
        'изменение production-политики',
        'выдача гарантированного результата',
      ],
      acceptanceCriteria: [
        'Proof нельзя утвердить без review decision.',
        'Выбранное правило reviewer видно в контракте.',
        'Failure mode фиксируется в событиях контракта.',
        'Открытые ambiguity должны быть разрешены до проверки.',
      ],
      openAmbiguities: ambiguities,
      nextStep: 'Перейти к проверке рисков и ambiguity.',
    };
  }

  const boundaryOpt = answers['bt-boundary'];
  const boundaryLabel = optionLabel('bt-boundary') ?? 'граница не указана';
  const scopeText = boundaryLabel.charAt(0).toUpperCase() + boundaryLabel.slice(1) + '.';

  if (boundaryOpt === 'unsure') pushAmbiguity('Граница задачи пока не определена.');
  if (boundaryOpt === 'multi-surface' || boundaryOpt === 'team-process') {
    pushAmbiguity('Scope может быть слишком широким для короткого пилота.');
  }

  return {
    scenario,
    title: 'Черновик рабочего контракта',
    summary:
      'GoalRail подготовил локальный черновик контракта для bounded task. Это не исполнение задачи, а рамка для проверки scope, критериев и рисков.',
    goal: 'Превратить входной запрос в ограниченный рабочий контракт с понятным следующим шагом.',
    scope: scopeText,
    outOfScope: [
      'выполнение кода',
      'подключение реального repo',
      'замена task tracker',
      'автоматическая доставка результата',
    ],
    acceptanceCriteria: [
      'Граница задачи явно зафиксирована.',
      'В контракте видны критерии приёмки.',
      'Риски и ambiguity вынесены до исполнения.',
      'Следующий шаг можно принять вручную.',
    ],
    openAmbiguities: ambiguities,
    nextStep: 'Перейти к проверке рисков и готовности.',
  };
}

function buildReviewReport({
  scenario,
  answers,
  draft,
}: {
  scenario: Scenario;
  answers: Record<string, string>;
  draft: ContractDraft;
}): ReviewReport {
  const items: ReviewItem[] = [];
  const risks: RiskItem[] = [];

  if (scenario === 'manual_review_gate') {
    const scopeOpt = answers['mrg-scope'];
    const decisionOpt = answers['mrg-decision'];
    const failOpt = answers['mrg-fail'];

    items.push({
      id: 'r-scope',
      label: 'Scope зафиксирован',
      detail: scopeOpt === 'unsure' ? 'Scope гейта пока не определён.' : 'Scope выбран в clarification.',
      status: scopeOpt === 'unsure' ? 'blocking' : 'ok',
    });
    items.push({
      id: 'r-decision',
      label: 'Review decision rule выбран',
      detail: decisionOpt === 'tbd'
        ? 'Кто принимает review decision, ещё не определено.'
        : 'Правило выбора reviewer зафиксировано.',
      status: decisionOpt === 'tbd' ? 'blocking' : 'ok',
    });
    items.push({
      id: 'r-fail',
      label: 'Failure mode определён',
      detail: failOpt === 'manual-decision'
        ? 'Ручное решение требует описания владельца.'
        : 'Failure mode выбран явно.',
      status: failOpt === 'manual-decision' ? 'warning' : 'ok',
    });
    items.push({
      id: 'r-out',
      label: 'Out of scope отделён',
      detail: 'Контракт явно исключает выполнение кода, подключение repo и production-политику.',
      status: 'ok',
    });

    if (scopeOpt === 'new-active') {
      risks.push({
        id: '',
        title: 'Нужна политика для активных контрактов',
        detail: 'Для уже активных контрактов нужна отдельная migration policy.',
        severity: 'blocking',
      });
    }
    if (decisionOpt === 'tbd') {
      risks.push({
        id: '',
        title: 'Владелец решения по review не определён',
        detail: 'Нужно определить, кто принимает review decision.',
        severity: 'blocking',
      });
    }
    if (failOpt === 'manual-decision') {
      risks.push({
        id: '',
        title: 'Путь ручного решения не описан',
        detail: 'Нужно описать, кто принимает ручное решение и где оно фиксируется.',
        severity: 'warning',
      });
    }
  } else {
    const boundaryOpt = answers['bt-boundary'];
    const visibleOpt = answers['bt-visible'];

    items.push({
      id: 'r-scope',
      label: 'Scope зафиксирован',
      detail: ((): string => {
        switch (boundaryOpt) {
          case 'one-repo':
            return 'Граница задачи зафиксирована на одном кейсе.';
          case 'multi-surface':
            return 'Несколько частей продукта — может быть слишком широко.';
          case 'team-process':
            return 'Процесс всей команды — слишком широко для пилота.';
          default:
            return 'Граница задачи пока не определена.';
        }
      })(),
      status: ((): ReviewStatus => {
        if (boundaryOpt === 'one-repo') return 'ok';
        if (boundaryOpt === 'multi-surface') return 'warning';
        return 'blocking';
      })(),
    });
    items.push({
      id: 'r-visible',
      label: 'Что должно быть видно в контракте',
      detail: visibleOpt === 'all'
        ? 'Выбор «всё перечисленное» может перегрузить контракт.'
        : 'Выбран явный фокус контракта.',
      status: visibleOpt === 'all' ? 'warning' : 'ok',
    });
    items.push({
      id: 'r-outcome',
      label: 'Честный итог выбран',
      detail: 'Ожидаемый итог пилота явно зафиксирован.',
      status: 'ok',
    });
    items.push({
      id: 'r-out',
      label: 'Out of scope отделён',
      detail: 'Контракт явно исключает выполнение кода, подключение repo и замену task tracker.',
      status: 'ok',
    });

    if (boundaryOpt === 'multi-surface') {
      risks.push({
        id: '',
        title: 'Scope может быть слишком широким',
        detail: 'Несколько частей продукта могут быть слишком широкими для короткого пилота.',
        severity: 'warning',
      });
    }
    if (boundaryOpt === 'team-process') {
      risks.push({
        id: '',
        title: 'Scope процесса команды слишком широкий',
        detail: 'Процесс всей команды лучше разбить на один repo / один кейс.',
        severity: 'blocking',
      });
    }
    if (boundaryOpt === 'unsure') {
      risks.push({
        id: '',
        title: 'Scope не определён',
        detail: 'Нужна конкретная граница задачи перед пилотом.',
        severity: 'blocking',
      });
    }
    if (visibleOpt === 'all') {
      risks.push({
        id: '',
        title: 'Контракт может быть перегружен',
        detail: 'Для первого пилота лучше выбрать главный фокус: критерии, риски или proof.',
        severity: 'warning',
      });
    }
  }

  risks.push({
    id: '',
    title: 'Выполнение в демо симулируется',
    detail: 'В этом демо код не выполняется, а проверка показывает только метод GoalRail.',
    severity: 'advisory',
  });

  risks.forEach((risk, index) => {
    risk.id = `R-${String(index + 1).padStart(2, '0')}`;
  });

  const hasBlocking =
    items.some((it) => it.status === 'blocking') || risks.some((r) => r.severity === 'blocking');
  const hasWarning =
    items.some((it) => it.status === 'warning') || risks.some((r) => r.severity === 'warning');

  let readinessLabel: string;
  let readinessSummary: string;
  let nextStep: string;
  if (hasBlocking) {
    readinessLabel = 'Нужны решения перед пилотом';
    readinessSummary = 'В черновике есть открытые вопросы, которые блокируют переход к итогу.';
    nextStep = 'Перед честным итогом нужно показать, какие решения блокируют пилот.';
  } else if (hasWarning) {
    readinessLabel = 'Готово с оговорками';
    readinessSummary = 'Черновик ограничен достаточно для обсуждения, но требует фиксации оговорок.';
    nextStep = 'Контракт можно обсуждать как пилотный кейс, но с явными оговорками.';
  } else {
    readinessLabel = 'Готово к следующему шагу';
    readinessSummary = 'Локальная проверка не выявила блокирующих вопросов.';
    nextStep = 'Черновик достаточно ограничен, чтобы перейти к честному итогу демо.';
  }

  return {
    readinessLabel,
    readinessSummary,
    readinessItems: items,
    riskItems: risks,
    ambiguityItems: draft.openAmbiguities,
    nextStep,
  };
}

function buildOutcomeReport({
  scenario,
  draft,
  review,
}: {
  scenario: Scenario;
  draft: ContractDraft;
  review: ReviewReport;
}): OutcomeReport {
  const tone = deriveOutcomeTone(review);

  let verdictLabel: string;
  let verdictTitle: string;
  let verdictBody: string;
  let recommendedNextStep: string;
  let ctaLabel: string;

  if (tone === 'ready') {
    verdictLabel = 'ГОТОВ К ПИЛОТУ';
    verdictTitle = 'Кейс подходит для короткого пилота';
    verdictBody =
      'Контракт ограничен, ключевые решения зафиксированы, а риски не блокируют следующий шаг. Это хороший кандидат для ручного пилота GoalRail на одном кейсе.';
    recommendedNextStep =
      'Обсудить реальный пилот на одном repo / одном кейсе и проверить процесс с оператором.';
    ctaLabel = 'Обсудить пилот';
  } else if (tone === 'readyWithCaveats') {
    verdictLabel = 'ПОДХОДИТ С ОГОВОРКАМИ';
    verdictTitle = 'Кейс можно брать в пилот, но с явными условиями';
    verdictBody =
      'Черновик уже показывает рабочую рамку, но перед стартом нужно явно зафиксировать открытые вопросы или риски. GoalRail полезен именно здесь: не прятать ambiguity, а вынести её до исполнения.';
    recommendedNextStep =
      'Зафиксировать оговорки и обсудить, какой bounded кейс лучше взять первым.';
    ctaLabel = 'Обсудить оговорки';
  } else {
    verdictLabel = 'СНАЧАЛА НУЖНЫ РЕШЕНИЯ';
    verdictTitle = 'Кейс пока не готов к пилоту';
    verdictBody =
      'В проверке есть блокирующие вопросы. Это не провал демо: GoalRail должен показать, где рано обещать результат. Лучше решить блокеры или выбрать более ограниченный кейс.';
    recommendedNextStep = 'Уточнить scope, decision rule или failure mode перед пилотом.';
    ctaLabel = 'Разобрать кейс';
  }

  const whatWeLearned = scenario === 'manual_review_gate' ? learnedRG : learnedBT;

  const whatRemainsOpen: string[] = [];
  for (const amb of draft.openAmbiguities) {
    whatRemainsOpen.push(amb.text);
  }
  for (const risk of review.riskItems) {
    if (risk.severity === 'blocking' || risk.severity === 'warning') {
      whatRemainsOpen.push(risk.title);
    }
  }
  if (whatRemainsOpen.length === 0) {
    whatRemainsOpen.push('критичных открытых вопросов в демо не осталось');
  }

  return {
    tone,
    verdictLabel,
    verdictTitle,
    verdictBody,
    whatWeLearned,
    whatRemainsOpen,
    recommendedNextStep,
    notDone: outcomeNotDone,
    ctaLabel,
  };
}

interface QuestionOption {
  id: string;
  label: string;
}

interface Question {
  id: string;
  prompt: string;
  options: QuestionOption[];
}

const REVIEW_GATE_SIGNALS = [
  'review',
  'reviewer',
  'approval',
  'approve',
  'proof',
  'gate',
  'manual',
  'провер',
  'ревью',
  'аппрув',
  'соглас',
  'утвержд',
];

function detectScenario(text: string): Scenario {
  const low = text.toLowerCase();
  return REVIEW_GATE_SIGNALS.some((signal) => low.includes(signal))
    ? 'manual_review_gate'
    : 'bounded_task';
}

const QUESTIONS: Record<Scenario, Question[]> = {
  manual_review_gate: [
    {
      id: 'mrg-scope',
      prompt: 'Где должен применяться review gate?',
      options: [
        { id: 'new-only', label: 'только новые контракты' },
        { id: 'new-active', label: 'новые и активные контракты' },
        { id: 'repo-scoped', label: 'только repo-scoped контракты' },
        { id: 'unsure', label: 'пока не уверен' },
      ],
    },
    {
      id: 'mrg-decision',
      prompt: 'Кто должен принимать review decision?',
      options: [
        { id: 'one-reviewer', label: 'один назначенный reviewer' },
        { id: 'any-operator', label: 'любой operator' },
        { id: 'quorum', label: 'quorum / два человека' },
        { id: 'tbd', label: 'пока не определено' },
      ],
    },
    {
      id: 'mrg-fail',
      prompt: 'Что должно произойти, если review не пройден?',
      options: [
        { id: 'block-proof', label: 'proof блокируется' },
        { id: 'back-to-clarification', label: 'контракт возвращается в clarification' },
        { id: 'halt-execution', label: 'execution останавливается' },
        { id: 'manual-decision', label: 'нужно решить вручную' },
      ],
    },
  ],
  bounded_task: [
    {
      id: 'bt-boundary',
      prompt: 'Какая граница задачи?',
      options: [
        { id: 'one-repo', label: 'один repo / один кейс' },
        { id: 'multi-surface', label: 'несколько частей продукта' },
        { id: 'team-process', label: 'процесс всей команды' },
        { id: 'unsure', label: 'пока не уверен' },
      ],
    },
    {
      id: 'bt-visible',
      prompt: 'Что должно быть видно в контракте?',
      options: [
        { id: 'criteria', label: 'критерии приёмки' },
        { id: 'risks', label: 'риски и ambiguity' },
        { id: 'proof', label: 'proof / проверка результата' },
        { id: 'all', label: 'всё перечисленное' },
      ],
    },
    {
      id: 'bt-outcome',
      prompt: 'Какой честный итог нужен?',
      options: [
        { id: 'go-pilot', label: 'можно запускать пилот' },
        { id: 'refine-scope', label: 'нужно уточнить scope' },
        { id: 'no-fit', label: 'кейс пока не подходит' },
        { id: 'see-risks', label: 'хочу увидеть риски' },
      ],
    },
  ],
};

function MenuMark() {
  return (
    <span className="menuMark" aria-hidden="true">
      <span />
      <span />
      <span />
    </span>
  );
}

function PeopleIcon() {
  return (
    <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8.5" cy="8" r="3" />
      <circle cx="16" cy="9.5" r="2.3" />
      <path d="M3 18c.8-2.8 3-4.5 5.5-4.5S13.2 15.2 14 18" />
      <path d="M14.5 18c.6-2.2 2.2-3.5 4-3.5s3.4 1.3 4 3.5" />
    </svg>
  );
}

function ClipboardIcon() {
  return (
    <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="5.5" y="4.5" width="13" height="16" rx="2" />
      <rect x="9" y="3" width="6" height="3.5" rx="1" />
      <path d="M9 11h6M9 14h6M9 17h4" />
    </svg>
  );
}

function XCircleIcon() {
  return (
    <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="8.5" />
      <path d="M9 9l6 6M15 9l-6 6" />
    </svg>
  );
}

function MailIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="3.5" y="5.5" width="17" height="13" rx="2" />
      <path d="M4 7l8 6 8-6" />
    </svg>
  );
}

function ShieldIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6l8-3z" />
      <path d="M9 12l2 2 4-4" />
    </svg>
  );
}

function FlagIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M5 21V4" />
      <path d="M5 4h11l-2 4 2 4H5" />
    </svg>
  );
}

function EyeIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

function ChecklistIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M8 9l2 2 4-4" />
      <path d="M8 14h8" />
      <path d="M8 18h6" />
    </svg>
  );
}

function DottedList({ items }: { items: string[] }) {
  return (
    <div className="dottedList">
      {items.map((item) => (
        <div className="dottedItem" key={item}>
          <span className="dot" aria-hidden="true" />
          <span>{item}</span>
        </div>
      ))}
    </div>
  );
}

function App() {
  const [intake, setIntake] = useState('');
  const [demoStep, setDemoStep] = useState<DemoStep>('intake');
  const [scenario, setScenario] = useState<Scenario | null>(null);
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const emailInputRef = useRef<HTMLInputElement | null>(null);
  const mainContentRef = useRef<HTMLElement | null>(null);
  const highlightTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const suppressNextAnimationRef = useRef(false);
  const [emailHighlighted, setEmailHighlighted] = useState(false);

  useEffect(() => {
    return () => {
      if (highlightTimeoutRef.current) {
        clearTimeout(highlightTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    suppressNextAnimationRef.current = false;
  }, [demoStep]);

  const trimmed = intake.trim();
  const length = trimmed.length;
  const isValid = length >= MIN_LEN && length <= MAX_LEN;

  const questions = scenario ? QUESTIONS[scenario] : [];
  const answeredCount = questions.filter((q) => answers[q.id]).length;
  const allAnswered = questions.length > 0 && answeredCount === questions.length;

  const contractDraft = useMemo(() => {
    if (!scenario || !allAnswered) return null;
    return buildContractDraft({ scenario, answers, intakeText: trimmed });
  }, [scenario, answers, allAnswered, trimmed]);

  const reviewReport = useMemo(() => {
    if (!scenario || !contractDraft) return null;
    return buildReviewReport({ scenario, answers, draft: contractDraft });
  }, [scenario, answers, contractDraft]);

  const outcomeReport = useMemo(() => {
    if (!scenario || !contractDraft || !reviewReport) return null;
    return buildOutcomeReport({ scenario, draft: contractDraft, review: reviewReport });
  }, [scenario, contractDraft, reviewReport]);

  const onIntakeChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    const next = event.target.value.slice(0, MAX_LEN);
    setIntake(next);
  };

  const onChip = (text: string) => {
    setIntake(text.slice(0, MAX_LEN));
  };

  const onSubmitIntake = () => {
    if (!isValid) return;
    setScenario(detectScenario(trimmed));
    setAnswers({});
    setDemoStep('clarification');
  };

  const onSelectOption = (questionId: string, optionId: string) => {
    setAnswers((prev) => ({ ...prev, [questionId]: optionId }));
  };

  const onPrepareContract = () => {
    if (!allAnswered) return;
    setDemoStep('contract');
  };

  const onBackToClarification = () => {
    suppressNextAnimationRef.current = true;
    setDemoStep('clarification');
  };

  const onAdvanceToReview = () => {
    setDemoStep('review');
  };

  const onBackToContract = () => {
    suppressNextAnimationRef.current = true;
    setDemoStep('contract');
  };

  const onPrepareOutcome = () => {
    setDemoStep('outcome');
  };

  const onBackToReview = () => {
    suppressNextAnimationRef.current = true;
    setDemoStep('review');
  };

  const onRestart = () => {
    setIntake('');
    setScenario(null);
    setAnswers({});
    setDemoStep('intake');
  };

  const onPrimaryOutcomeCta = () => {
    const target = emailInputRef.current;
    if (target) {
      target.focus();
      target.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
    if (highlightTimeoutRef.current) {
      clearTimeout(highlightTimeoutRef.current);
    }
    setEmailHighlighted(true);
    highlightTimeoutRef.current = setTimeout(() => {
      setEmailHighlighted(false);
      highlightTimeoutRef.current = null;
    }, 2400);
  };

  const onChangeIntake = () => {
    suppressNextAnimationRef.current = true;
    setDemoStep('intake');
    setScenario(null);
    setAnswers({});
  };

  const ctxRows = useMemo(() => {
    if (demoStep === 'outcome') {
      return [
        ['Кейс', 'demo-contract'],
        ['Режим', 'без подключения repo'],
        ['Формат', 'outcome'],
        ['Готовность', 'метод показан'],
      ];
    }
    if (demoStep === 'review') {
      return [
        ['Кейс', 'demo-contract'],
        ['Режим', 'без подключения repo'],
        ['Формат', 'review'],
        ['Готовность', 'локальная демо-проверка'],
      ];
    }
    if (demoStep === 'contract') {
      return [
        ['Кейс', 'demo-contract'],
        ['Режим', 'без подключения repo'],
        ['Формат', 'contract-draft'],
        ['Готовность', 'появится после проверки'],
      ];
    }
    if (demoStep === 'clarification') {
      return [
        ['Кейс', 'demo-contract'],
        ['Режим', 'без подключения repo'],
        ['Формат', 'clarification'],
        ['Готовность', 'появится после контракта'],
      ];
    }
    if (isValid) {
      return [
        ['Кейс', 'demo-contract'],
        ['Режим', 'без подключения repo'],
        ['Формат', 'ручной пилот'],
        ['Готовность', 'демо-оценка появится дальше'],
      ];
    }
    return [
      ['Кейс', 'демо-кейс'],
      ['Режим', 'без подключения repo'],
      ['Формат', 'ручной пилот'],
      ['Готовность', 'оценим после описания'],
    ];
  }, [demoStep, isValid]);

  const railState = (idx: number): 'done' | 'active' | 'muted' => {
    if (demoStep === 'intake') {
      return idx === 0 ? 'active' : 'muted';
    }
    if (demoStep === 'clarification') {
      if (idx === 0) return 'done';
      if (idx === 1) return 'active';
      return 'muted';
    }
    if (demoStep === 'contract') {
      if (idx === 0) return 'done';
      if (idx === 1) return 'done';
      if (idx === 2) return 'active';
      return 'muted';
    }
    if (demoStep === 'review') {
      if (idx <= 2) return 'done';
      if (idx === 3) return 'active';
      return 'muted';
    }
    if (idx <= 3) return 'done';
    return 'active';
  };

  return (
    <div className="landingShell">
      <a
        className="skipLink"
        href="#main-content"
        onClick={(event) => {
          event.preventDefault();
          const target = mainContentRef.current;
          if (target) {
            target.focus();
            target.scrollIntoView({ block: 'start' });
            if (typeof window !== 'undefined' && window.location) {
              window.location.hash = 'main-content';
            }
          }
        }}
      >
        К основному содержимому
      </a>
      <div className="landingFrame" data-screen-label="Goalrail Pilot Intake">
        <header className="topbar">
          <div className="topbarLeft">
            <MenuMark />
            <div className="brand mono" aria-label="Goalrail pilot">
              <span className="brandName">GOALRAIL</span>
              <span className="brandSlash">/</span>
              <span className="brandSub">ПИЛОТ</span>
            </div>
          </div>
          <div className="topChips" aria-label="Pilot status">
            {topChips.map((chip, index) => (
              <span className={index === 0 ? 'chip statusChip' : 'chip'} key={chip}>
                {chip}
              </span>
            ))}
          </div>
        </header>

        <div className="bodyGrid">
          <aside className="sidebar" aria-label="Разделы лендинга">
            <div>
              <div className="sidebarLabel">РАЗДЕЛЫ</div>
              <nav className="sideNav" aria-label="Основные разделы">
                {primaryNav.map((item, index) => (
                  <a className={index === 0 ? 'navItem active' : 'navItem'} href={`#${index === 0 ? 'pilot' : 'details'}`} key={item}>
                    <span className={index === 0 ? 'openDot' : 'spacer'} aria-hidden="true" />
                    {item}
                  </a>
                ))}
              </nav>
            </div>

            <div>
              <div className="sidebarLabel">КОНТЕКСТ</div>
              <nav className="sideNav" aria-label="Контекст пилота">
                {contextNav.map((item) => (
                  <a className="navItem" href="#fit" key={item}>
                    <span className="spacer" aria-hidden="true" />
                    {item}
                  </a>
                ))}
              </nav>
            </div>

            <section className="statusCard" aria-label="Формат пилота">
              <div className="statusHeader">
                <span className="statusTitle">ФОРМАТ</span>
                <span className="openBadge">ОТКРЫТ</span>
              </div>
              <div className="statusLines">
                {statusLines.map((line) => (
                  <div key={line}>{line}</div>
                ))}
              </div>
            </section>

            <div className="sidebarFoot">
              <div className="modeLabel">РЕЖИМ</div>
              <div className="modeValue">Публичная страница пилота</div>
            </div>
          </aside>

          <main
            className="mainGrid"
            id="main-content"
            ref={mainContentRef}
            tabIndex={-1}
            aria-label="Демо GoalRail"
          >
            <section className="card heroCard" id="pilot" aria-labelledby="hero-title">
              <div className="eyebrow">ПИЛОТНЫЙ ФОРМАТ</div>
              <h1 className="heroTitle" id="hero-title">
                Покажите задачу — мы проведём вас по контракту
              </h1>
              <p className="heroSub">
                Опишите реальную задачу, PR, изменение или кейс. Демо покажет, как GoalRail превращает её в контракт, уточняющие вопросы, критерии и честный следующий шаг — без выполнения кода и подключения репозитория.
              </p>

              <ol className="demoRail" aria-label="Шаги демо">
                {demoSteps.map((label, idx) => {
                  const state = railState(idx);
                  return (
                    <li
                      className={`demoRailItem ${state}`}
                      key={label}
                      data-step-state={state}
                      aria-current={state === 'active' ? 'step' : undefined}
                    >
                      <span className="demoRailNum">{String(idx + 1).padStart(2, '0')}</span>
                      <span className="demoRailLabel">{label}</span>
                    </li>
                  );
                })}
              </ol>

              {demoStep === 'intake' && (
                <>
                  <div className={`composer${isValid ? ' valid' : ''}`}>
                    <div className="composerHead">
                      <span className="composerLabel mono">GOAL INTAKE · Шаг 1 из 5</span>
                      <span className="composerStage mono">→ Уточнения</span>
                    </div>
                    <label className="srOnly" htmlFor="demo-intake">
                      Опишите задачу, PR, изменение или кейс
                    </label>
                    <textarea
                      id="demo-intake"
                      className="composerInput"
                      placeholder="Опишите задачу, PR, изменение или кейс…"
                      value={intake}
                      onChange={onIntakeChange}
                      maxLength={MAX_LEN}
                      rows={5}
                      aria-describedby="demo-intake-safety"
                    />
                    <div className="composerFoot">
                      <div className="composerHints">
                        <span id="demo-intake-safety" className="composerSafety">
                          Не вставляйте секреты, токены и приватные данные.
                        </span>
                        <span className="composerCount" aria-live="polite">
                          {length} / {MAX_LEN}
                        </span>
                      </div>
                      <button
                        type="button"
                        className="primaryButton"
                        onClick={onSubmitIntake}
                        disabled={!isValid}
                        aria-disabled={!isValid}
                      >
                        Запустить демо
                      </button>
                    </div>
                  </div>

                  <div className="chipRow" aria-label="Примеры задач">
                    {exampleChips.map((chip) => (
                      <button
                        key={chip}
                        type="button"
                        className="exampleChip"
                        onClick={() => onChip(chip)}
                      >
                        {chip}
                      </button>
                    ))}
                  </div>
                </>
              )}

              {demoStep === 'clarification' && (
                <div
                  className="clarificationBlock"
                  data-skip-animation={suppressNextAnimationRef.current ? 'true' : 'false'}
                >
                  <section className="intakeSummary" aria-label="Принятый запрос">
                    <span className="intakeSummaryLabel mono">ЗАПРОС ПРИНЯТ</span>
                    <p className="intakeSummaryText">{trimmed}</p>
                  </section>

                  <section className="clarificationPanel" aria-labelledby="clarification-title">
                    <div className="clarificationHead">
                      <span className="clarificationEyebrow mono">CLARIFICATION · Шаг 2 из 5</span>
                      <h2 className="clarificationTitle" id="clarification-title">
                        Уточним границы перед контрактом
                      </h2>
                      <p className="clarificationBody">
                        GoalRail не начинает execution сразу. Сначала фиксируем scope, критерии и открытые вопросы, чтобы задача стала рабочим контрактом.
                      </p>
                    </div>

                    <ol className="clarificationQuestions" aria-label="Уточняющие вопросы">
                      {questions.map((q, qi) => (
                        <li className="clarificationQuestion" key={q.id}>
                          <div className="clarificationQuestionHead">
                            <span className="clarificationQuestionNum mono">
                              Q-{String(qi + 1).padStart(2, '0')}
                            </span>
                            <span className="clarificationQuestionPrompt" id={`clar-${q.id}-label`}>
                              {q.prompt}
                            </span>
                          </div>
                          <div
                            className="clarificationOptions"
                            role="radiogroup"
                            aria-labelledby={`clar-${q.id}-label`}
                          >
                            {q.options.map((opt) => {
                              const checked = answers[q.id] === opt.id;
                              return (
                                <button
                                  type="button"
                                  role="radio"
                                  aria-checked={checked}
                                  className={`clarificationOption${checked ? ' active' : ''}`}
                                  key={opt.id}
                                  onClick={() => onSelectOption(q.id, opt.id)}
                                >
                                  <span className="clarificationOptionDot" aria-hidden="true" />
                                  <span className="clarificationOptionLabel">{opt.label}</span>
                                </button>
                              );
                            })}
                          </div>
                        </li>
                      ))}
                    </ol>

                    <div className="clarificationFoot">
                      <span className="clarificationProgress mono" aria-live="polite">
                        Ответы: {answeredCount} / {questions.length}
                      </span>
                      <div className="clarificationActions">
                        <button type="button" className="ghostButton ghostButtonInline" onClick={onChangeIntake}>
                          Изменить запрос
                        </button>
                        <button
                          type="button"
                          className="primaryButton"
                          onClick={onPrepareContract}
                          disabled={!allAnswered}
                          aria-disabled={!allAnswered}
                        >
                          Подготовить контракт
                        </button>
                      </div>
                    </div>

                  </section>
                </div>
              )}

              {demoStep === 'contract' && contractDraft && (
                <div
                  className="contractBlock"
                  data-skip-animation={suppressNextAnimationRef.current ? 'true' : 'false'}
                >
                  <section className="intakeSummary" aria-label="Принятый запрос">
                    <span className="intakeSummaryLabel mono">ЗАПРОС</span>
                    <p className="intakeSummaryText">{trimmed}</p>
                  </section>

                  <section className="contractPanel" aria-labelledby="contract-title">
                    <div className="clarificationHead">
                      <span className="clarificationEyebrow mono">CONTRACT DRAFT · Шаг 3 из 5</span>
                      <h2 className="clarificationTitle" id="contract-title">
                        Черновик контракта подготовлен
                      </h2>
                      <p className="clarificationBody">
                        Это локальная демонстрация. Код не выполняется, repo не подключается, а черновик показывает только метод GoalRail.
                      </p>
                    </div>

                    <dl className="contractCard" role="group" aria-label="Поля контракта">
                      <div className="contractRow">
                        <dt className="contractRowKey mono">Название</dt>
                        <dd className="contractRowVal">{contractDraft.title}</dd>
                      </div>
                      <div className="contractRow">
                        <dt className="contractRowKey mono">Цель</dt>
                        <dd className="contractRowVal">{contractDraft.goal}</dd>
                      </div>
                      <div className="contractRow">
                        <dt className="contractRowKey mono">Scope</dt>
                        <dd className="contractRowVal">{contractDraft.scope}</dd>
                      </div>
                      {contractDraft.reviewerRule !== undefined && (
                        <div className="contractRow">
                          <dt className="contractRowKey mono">Правило review / decision</dt>
                          <dd className="contractRowVal">{contractDraft.reviewerRule}</dd>
                        </div>
                      )}
                      {contractDraft.failureMode !== undefined && (
                        <div className="contractRow">
                          <dt className="contractRowKey mono">Failure mode</dt>
                          <dd className="contractRowVal">{contractDraft.failureMode}</dd>
                        </div>
                      )}
                      <div className="contractRow contractRowList">
                        <dt className="contractRowKey mono">Вне scope</dt>
                        <dd className="contractRowVal">
                          <ul className="contractBullets">
                            {contractDraft.outOfScope.map((item) => (
                              <li key={item}>{item}</li>
                            ))}
                          </ul>
                        </dd>
                      </div>
                      <div className="contractRow contractRowList">
                        <dt className="contractRowKey mono">Критерии приёмки</dt>
                        <dd className="contractRowVal">
                          <ol className="contractCriteria">
                            {contractDraft.acceptanceCriteria.map((criterion, idx) => (
                              <li key={criterion}>
                                <span className="contractCriteriaId mono">
                                  AC-{String(idx + 1).padStart(2, '0')}
                                </span>
                                <span className="contractCriteriaText">{criterion}</span>
                              </li>
                            ))}
                          </ol>
                        </dd>
                      </div>
                      <div className="contractRow contractRowList">
                        <dt className="contractRowKey mono">
                          Open ambiguity · {contractDraft.openAmbiguities.length}
                        </dt>
                        <dd className="contractRowVal">
                          {contractDraft.openAmbiguities.length === 0 ? (
                            <p className="contractEmpty">
                              Критичных ambiguity не найдено для демо-черновика.
                            </p>
                          ) : (
                            <ul className="contractAmbiguityList" aria-label="Открытые ambiguity">
                              {contractDraft.openAmbiguities.map((amb) => (
                                <li className="contractAmbiguity" key={amb.id}>
                                  <span className="contractAmbiguityId mono">{amb.id}</span>
                                  <span className="contractAmbiguityText">{amb.text}</span>
                                  <span className="contractAmbiguityBadge mono">требует решения</span>
                                </li>
                              ))}
                            </ul>
                          )}
                        </dd>
                      </div>
                      <div className="contractRow">
                        <dt className="contractRowKey mono">Следующий шаг</dt>
                        <dd className="contractRowVal">{contractDraft.nextStep}</dd>
                      </div>
                    </dl>

                    <div className="contractActions">
                      <button
                        type="button"
                        className="ghostButton ghostButtonInline"
                        onClick={onBackToClarification}
                      >
                        Вернуться к уточнениям
                      </button>
                      <button
                        type="button"
                        className="ghostButton ghostButtonInline"
                        onClick={onChangeIntake}
                      >
                        Изменить запрос
                      </button>
                      <button
                        type="button"
                        className="primaryButton"
                        onClick={onAdvanceToReview}
                      >
                        Перейти к проверке
                      </button>
                    </div>
                  </section>
                </div>
              )}

              {demoStep === 'review' && contractDraft && reviewReport && (
                <div
                  className="reviewBlock"
                  data-skip-animation={suppressNextAnimationRef.current ? 'true' : 'false'}
                >
                  <section className="intakeSummary" aria-label="Принятый запрос">
                    <span className="intakeSummaryLabel mono">ЗАПРОС</span>
                    <p className="intakeSummaryText">{trimmed}</p>
                  </section>

                  <section className="contractDigest" aria-label="Сводка черновика">
                    <span className="intakeSummaryLabel mono">ЧЕРНОВИК</span>
                    <p className="contractDigestTitle">{contractDraft.title}</p>
                    <p className="contractDigestSub">{contractDraft.nextStep}</p>
                  </section>

                  <section className="reviewPanel" aria-labelledby="review-title">
                    <div className="clarificationHead">
                      <span className="clarificationEyebrow mono">REVIEW · Шаг 4 из 5</span>
                      <h2 className="clarificationTitle" id="review-title">
                        Проверка рисков и готовности
                      </h2>
                      <p className="clarificationBody">
                        Это локальная демо-проверка черновика. Код не выполняется, repo не подключается, а статусы показывают, какие вопросы нужно решить до реального пилота.
                      </p>
                    </div>

                    <section className="reviewSection" aria-labelledby="review-readiness-title">
                      <h3 className="reviewSectionTitle" id="review-readiness-title">
                        Готовность к следующему шагу
                      </h3>
                      <div
                        className={
                          'reviewReadiness reviewReadiness-' +
                          (reviewReport.readinessLabel.includes('Нужны решения')
                            ? 'blocking'
                            : reviewReport.readinessLabel.includes('оговорками')
                              ? 'warning'
                              : 'ok')
                        }
                      >
                        <span className="reviewReadinessLabel">
                          {reviewReport.readinessLabel}
                        </span>
                        <span className="reviewReadinessSummary">
                          {reviewReport.readinessSummary}
                        </span>
                      </div>
                      <ul className="reviewItemsList" aria-label="Чек-лист готовности">
                        {reviewReport.readinessItems.map((item) => (
                          <li className="reviewItem" key={item.id}>
                            <span
                              className={`reviewItemBadge reviewItemBadge-${item.status} mono`}
                            >
                              {STATUS_LABEL[item.status]}
                            </span>
                            <div className="reviewItemBody">
                              <span className="reviewItemLabel">{item.label}</span>
                              <span className="reviewItemDetail">{item.detail}</span>
                            </div>
                          </li>
                        ))}
                      </ul>
                    </section>

                    <section className="reviewSection" aria-labelledby="review-risks-title">
                      <h3 className="reviewSectionTitle" id="review-risks-title">
                        Риски и ambiguity
                      </h3>
                      <ul className="reviewRiskList" aria-label="Риски">
                        {reviewReport.riskItems.map((risk) => (
                          <li className="reviewRisk" key={risk.id}>
                            <span className="reviewRiskId mono">{risk.id}</span>
                            <div className="reviewRiskBody">
                              <span className="reviewRiskTitle">{risk.title}</span>
                              <span className="reviewRiskDetail">{risk.detail}</span>
                            </div>
                            <span
                              className={`reviewRiskBadge reviewRiskBadge-${risk.severity} mono`}
                            >
                              {SEVERITY_LABEL[risk.severity]}
                            </span>
                          </li>
                        ))}
                      </ul>

                      <div className="reviewAmbiguity">
                        <span className="reviewAmbiguityHeading mono">
                          Open ambiguity · {reviewReport.ambiguityItems.length}
                        </span>
                        {reviewReport.ambiguityItems.length === 0 ? (
                          <p className="contractEmpty">
                            Критичных ambiguity не найдено для демо-проверки.
                          </p>
                        ) : (
                          <ul className="contractAmbiguityList" aria-label="Открытые ambiguity">
                            {reviewReport.ambiguityItems.map((amb) => (
                              <li className="contractAmbiguity" key={amb.id}>
                                <span className="contractAmbiguityId mono">{amb.id}</span>
                                <span className="contractAmbiguityText">{amb.text}</span>
                                <span className="contractAmbiguityBadge mono">требует решения</span>
                              </li>
                            ))}
                          </ul>
                        )}
                      </div>
                    </section>

                    <section className="reviewSection" aria-labelledby="review-conclusion-title">
                      <h3 className="reviewSectionTitle" id="review-conclusion-title">
                        Проверочный вывод
                      </h3>
                      <p className="reviewConclusion">{reviewReport.nextStep}</p>
                    </section>

                    <div className="contractActions">
                      <button
                        type="button"
                        className="ghostButton ghostButtonInline"
                        onClick={onBackToContract}
                      >
                        Вернуться к черновику
                      </button>
                      <button
                        type="button"
                        className="ghostButton ghostButtonInline"
                        onClick={onChangeIntake}
                      >
                        Изменить запрос
                      </button>
                      <button
                        type="button"
                        className="primaryButton"
                        onClick={onPrepareOutcome}
                      >
                        Подготовить итог
                      </button>
                    </div>
                  </section>
                </div>
              )}

              {demoStep === 'outcome' && contractDraft && reviewReport && outcomeReport && (
                <div
                  className="outcomeBlock"
                  data-skip-animation={suppressNextAnimationRef.current ? 'true' : 'false'}
                >
                  <section className="intakeSummary" aria-label="Принятый запрос">
                    <span className="intakeSummaryLabel mono">ЗАПРОС</span>
                    <p className="intakeSummaryText">{trimmed}</p>
                  </section>

                  <section className="contractDigest" aria-label="Сводка черновика">
                    <span className="intakeSummaryLabel mono">КОНТРАКТ</span>
                    <p className="contractDigestTitle">{contractDraft.title}</p>
                    <p className="contractDigestSub">{contractDraft.goal}</p>
                  </section>

                  <section className="reviewDigest" aria-label="Сводка проверки">
                    <span className="intakeSummaryLabel mono">ПРОВЕРКА</span>
                    <p className="contractDigestTitle">{reviewReport.readinessLabel}</p>
                    {(() => {
                      const blockerCount = reviewReport.riskItems.filter(
                        (r) => r.severity === 'blocking',
                      ).length;
                      const warningCount = reviewReport.riskItems.filter(
                        (r) => r.severity === 'warning',
                      ).length;
                      const ambiguityCount = reviewReport.ambiguityItems.length;
                      return (
                        <div className="reviewDigestCounters" aria-label="Счётчики проверки">
                          <span
                            className="reviewDigestCounter reviewDigestCounter-blocking"
                            data-count={blockerCount}
                            aria-label={`Блокеры: ${blockerCount}`}
                          >
                            <span className="reviewDigestCounterLabel mono">Блокеры</span>
                            <span className="reviewDigestCounterValue mono">{blockerCount}</span>
                          </span>
                          <span
                            className="reviewDigestCounter reviewDigestCounter-warning"
                            data-count={warningCount}
                            aria-label={`Оговорки: ${warningCount}`}
                          >
                            <span className="reviewDigestCounterLabel mono">Оговорки</span>
                            <span className="reviewDigestCounterValue mono">{warningCount}</span>
                          </span>
                          <span
                            className="reviewDigestCounter reviewDigestCounter-ambiguity"
                            data-count={ambiguityCount}
                            aria-label={`Открытых ambiguity: ${ambiguityCount}`}
                          >
                            <span className="reviewDigestCounterLabel mono">Открытых ambiguity</span>
                            <span className="reviewDigestCounterValue mono">{ambiguityCount}</span>
                          </span>
                        </div>
                      );
                    })()}
                  </section>

                  <section className="outcomePanel" aria-labelledby="outcome-title">
                    <div className="clarificationHead">
                      <span className="clarificationEyebrow mono">OUTCOME · Шаг 5 из 5</span>
                      <h2 className="clarificationTitle" id="outcome-title">
                        Честный итог демо
                      </h2>
                      <p className="clarificationBody">
                        Это финальный вывод локального walkthrough. GoalRail показывает не выполненную задачу, а метод: как запрос превращается в контракт, проверку и честный следующий шаг.
                      </p>
                    </div>

                    <div
                      className={`verdictCard verdictCard-${outcomeReport.tone}`}
                      data-outcome-tone={outcomeReport.tone}
                      aria-label="Verdict"
                    >
                      <span className="verdictLabel mono">{outcomeReport.verdictLabel}</span>
                      <h3 className="verdictTitle">{outcomeReport.verdictTitle}</h3>
                      <p className="verdictBody">{outcomeReport.verdictBody}</p>
                      <p className="verdictNextStep">
                        <span className="verdictNextStepLabel mono">Следующий шаг</span>
                        <span className="verdictNextStepText">{outcomeReport.recommendedNextStep}</span>
                      </p>
                    </div>

                    <section className="reviewSection" aria-labelledby="outcome-learned-title">
                      <h3 className="reviewSectionTitle" id="outcome-learned-title">
                        Что стало ясно
                      </h3>
                      <ul className="outcomeList" aria-label="Что стало ясно">
                        {outcomeReport.whatWeLearned.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    </section>

                    <section className="reviewSection" aria-labelledby="outcome-open-title">
                      <h3 className="reviewSectionTitle" id="outcome-open-title">
                        Что осталось открытым
                      </h3>
                      <ul className="outcomeList outcomeList-warm" aria-label="Что осталось открытым">
                        {outcomeReport.whatRemainsOpen.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    </section>

                    <section className="reviewSection" aria-labelledby="outcome-notdone-title">
                      <h3 className="reviewSectionTitle" id="outcome-notdone-title">
                        Что демо не делало
                      </h3>
                      <ul className="outcomeList" aria-label="Что демо не делало">
                        {outcomeReport.notDone.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    </section>

                    <section className="reviewSection" aria-labelledby="outcome-next-title">
                      <h3 className="reviewSectionTitle" id="outcome-next-title">
                        Следующий шаг
                      </h3>
                      <p className="reviewConclusion">{outcomeReport.recommendedNextStep}</p>
                    </section>

                    <div className="contractActions">
                      <button
                        type="button"
                        className="ghostButton ghostButtonInline"
                        onClick={onBackToReview}
                      >
                        Вернуться к проверке
                      </button>
                      <button
                        type="button"
                        className="ghostButton ghostButtonInline"
                        onClick={onRestart}
                      >
                        Начать заново
                      </button>
                      <button
                        type="button"
                        className="primaryButton"
                        onClick={onPrimaryOutcomeCta}
                      >
                        {outcomeReport.ctaLabel}
                      </button>
                    </div>
                  </section>
                </div>
              )}

              <section role="group" aria-label="Контекст демо">
                <dl className="contextStrip">
                  {ctxRows.map(([key, value]) => (
                    <div className="contextCell" key={key}>
                      <dt className="contextKey mono">{key}</dt>
                      <dd className="contextVal">{value}</dd>
                    </div>
                  ))}
                </dl>
              </section>
            </section>

            <aside className="rightColumn" aria-label="О демо">
              {demoStep === 'clarification' && (
                <section className="card sideCard">
                  <div className="sideCardHead">
                    <span className="sideIcon"><ChecklistIcon /></span>
                    <h2 className="sideCardTitle">Зачем уточнения</h2>
                  </div>
                  <DottedList items={clarificationItems} />
                </section>
              )}

              {demoStep === 'contract' && (
                <section className="card sideCard">
                  <div className="sideCardHead">
                    <span className="sideIcon"><ChecklistIcon /></span>
                    <h2 className="sideCardTitle">Что в черновике</h2>
                  </div>
                  <DottedList items={contractPreviewItems} />
                </section>
              )}

              {demoStep === 'review' && (
                <section className="card sideCard">
                  <div className="sideCardHead">
                    <span className="sideIcon"><ShieldIcon /></span>
                    <h2 className="sideCardTitle">Что проверяется</h2>
                  </div>
                  <DottedList items={reviewPreviewItems} />
                </section>
              )}

              {demoStep === 'outcome' && (
                <section className="card sideCard">
                  <div className="sideCardHead">
                    <span className="sideIcon"><FlagIcon /></span>
                    <h2 className="sideCardTitle">Что в итоге</h2>
                  </div>
                  <DottedList items={outcomePreviewItems} />
                </section>
              )}

              <section className="card sideCard">
                <div className="sideCardHead">
                  <span className="sideIcon"><EyeIcon /></span>
                  <h2 className="sideCardTitle">Что вы увидите</h2>
                </div>
                <DottedList items={seeItems} />
              </section>

              <section className="card sideCard">
                <div className="sideCardHead">
                  <span className="sideIcon"><ShieldIcon /></span>
                  <h2 className="sideCardTitle">Безопасно для демо</h2>
                </div>
                <DottedList items={safeItems} />
              </section>

              <section className="card sideCard">
                <div className="sideCardHead">
                  <span className="sideIcon"><FlagIcon /></span>
                  <h2 className="sideCardTitle">Итог пилота</h2>
                </div>
                <p className="sideCardText">
                  Вы увидите метод GoalRail: как задача превращается в контракт, где появляются ambiguity и какой следующий шаг честный.
                </p>
              </section>
            </aside>

            <section className="triplet" aria-label="Pilot explanation cards">
              <article className="card tripletCard">
                <div className="tripletIcon"><PeopleIcon /></div>
                <h2 className="tripletTitle">Когда нужен Goalrail</h2>
                <div className="tripletSeparator" />
                <p className="tripletBody">
                  Когда ИИ уже помогает писать код, но команде нужен управляемый эксперимент: с ясной постановкой, границами задачи, проверкой и понятным следующим шагом.
                </p>
              </article>

              <article className="card tripletCard">
                <div className="tripletIcon"><ClipboardIcon /></div>
                <h2 className="tripletTitle">Что делаем в пилоте</h2>
                <div className="tripletSeparator" />
                <div className="stepsList">
                  {pilotSteps.map((step) => (
                    <div className="stepItem" key={step}>{step}</div>
                  ))}
                </div>
              </article>

              <article className="card tripletCard">
                <div className="tripletIcon"><XCircleIcon /></div>
                <h2 className="tripletTitle">Что не делаем</h2>
                <div className="tripletSeparator" />
                <div className="exclusionList">
                  {exclusions.map((item) => (
                    <div className="exclusionItem" key={item}>
                      <span aria-hidden="true">×</span>
                      <span>{item}</span>
                    </div>
                  ))}
                </div>
              </article>
            </section>

            <section
              className={`card ctaCard${emailHighlighted ? ' ctaCard--highlight' : ''}`}
              aria-labelledby="cta-title"
              data-cta-highlighted={emailHighlighted ? 'true' : 'false'}
            >
              <div className="ctaIcon"><MailIcon /></div>
              <div>
                <h2 className="ctaTitle" id="cta-title">Если вам это близко, оставьте почту</h2>
                <p className="ctaSub">Без длинной анкеты. Сначала поймём, есть ли хороший кейс для пилота.</p>
              </div>
              <form className="ctaForm" action="mailto:hello@goalrail.dev" method="post" encType="text/plain" aria-label="Форма заявки на пилот">
                <label className="srOnly" htmlFor="pilot-email">Рабочая почта</label>
                <input
                  className="textInput"
                  id="pilot-email"
                  name="email"
                  type="email"
                  placeholder="ваша@компания.ru"
                  autoComplete="email"
                  ref={emailInputRef}
                />
                <button className="ghostButton" type="submit">Обсудить пилот</button>
              </form>
              <p className="ctaNote">
                Без рассылок. Только по делу. <span aria-hidden="true">·</span> Прямая почта:{' '}
                <a href="mailto:hello@goalrail.dev">hello@goalrail.dev</a>
              </p>
            </section>
          </main>
        </div>
      </div>
    </div>
  );
}

export default App;

export interface RuStartQuestion {
  id: string;
  label: string;
  answerId: string;
}

export interface RuStartAnswer {
  id: string;
  title: string;
  eyebrow: string;
  body: string[];
  sources: string[];
  nextQuestions: string[];
}

export interface RuStartArtifact {
  title: string;
  body: string;
  signal: string;
}

// Russian public /start copy derived from the English global start page data
// and the public start-assistant reference docs.
export const ruStartQuestions: RuStartQuestion[] = [
  {
    id: 'what-is-goalrail',
    label: 'Что такое GoalRail?',
    answerId: 'what-is-goalrail',
  },
  {
    id: 'repo-ready',
    label: 'Готов ли мой репозиторий к coding agents?',
    answerId: 'repo-readiness',
  },
  {
    id: 'contract-first',
    label: 'Что такое contract-first execution?',
    answerId: 'contract-first-execution',
  },
  {
    id: 'proof-before-approval',
    label: 'Что значит proof before approval?',
    answerId: 'proof-before-approval',
  },
  {
    id: 'not-ai-ide',
    label: 'Чем GoalRail отличается от AI IDE?',
    answerId: 'different-from-ai-ide',
  },
  {
    id: 'pilot-fit-check',
    label: 'Как выглядит проверка fit для пилота?',
    answerId: 'pilot-fit-check',
  },
  {
    id: 'ai-delivery-drift',
    label: 'Что такое AI delivery drift?',
    answerId: 'ai-delivery-drift',
  },
  {
    id: 'ai-review',
    label: 'Как команде ревьюить AI-generated changes?',
    answerId: 'review-ai-generated-changes',
  },
];

export const ruStartAnswers: Record<string, RuStartAnswer> = {
  'what-is-goalrail': {
    id: 'what-is-goalrail',
    title: 'GoalRail - слой контроля для AI-assisted software delivery.',
    eyebrow: 'Определение',
    body: [
      'GoalRail помогает командам пройти путь от бизнес-цели до проверенного изменения в коде: через уточнение цели, контракт, ограниченное исполнение, проверки, доказательства и human approval.',
      'Это не AI IDE, не замена Jira и не generic agent platform.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['Что такое contract-first execution?', 'Что значит proof before approval?'],
  },
  'repo-readiness': {
    id: 'repo-readiness',
    title: 'Репозиторий не становится готовым к coding agents только потому, что он собирается.',
    eyebrow: 'Repo readiness',
    body: [
      'Готовность репозитория означает, что в нем достаточно рабочих сигналов, чтобы agent мог действовать безопасно.',
      'Команда должна видеть, как запускать проект, какие проверки важны, что нельзя трогать, где границы ответственности, как доказать сохранение поведения и как откатиться, если изменение пошло не туда.',
    ],
    sources: ['START_ASSISTANT_SECURITY_AND_PRIVACY.md', 'static-answers.md'],
    nextQuestions: ['Как команде ревьюить AI-generated changes?', 'Что такое AI delivery drift?'],
  },
  'contract-first-execution': {
    id: 'contract-first-execution',
    title: 'Contract-first execution означает, что agent не начинает работу с произвольного prompt.',
    eyebrow: 'Контракт',
    body: [
      'Сначала работа ограничивается контрактом: цель, scope, non-goals, затронутые зоны, обязательные проверки, ожидаемые артефакты и критерии proof.',
      'Смысл не в лишнем процессе. Смысл в том, чтобы во время исполнения было меньше скрытых решений.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['Что значит proof before approval?', 'Чем GoalRail отличается от AI IDE?'],
  },
  'proof-before-approval': {
    id: 'proof-before-approval',
    title: 'Output - это еще не proof.',
    eyebrow: 'Proof',
    body: [
      'Proof before approval означает, что reviewers не принимают AI-generated work только потому, что diff выглядит аккуратно или summary звучит уверенно.',
      'Перед approval нужно сопоставить контракт, diff, проверки, артефакты и оставшийся риск.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['Как команде ревьюить AI-generated changes?', 'Готов ли мой репозиторий к coding agents?'],
  },
  'different-from-ai-ide': {
    id: 'different-from-ai-ide',
    title: 'GoalRail не является AI IDE.',
    eyebrow: 'Позиционирование',
    body: [
      'AI IDE помогают генерировать или редактировать код. GoalRail фокусируется на слое контроля вокруг AI-assisted delivery: intent, scope, contract, checks, proof и human approval.',
      'Он работает вокруг инструментов, которые команда уже использует.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['Что такое GoalRail?', 'Как выглядит проверка fit для пилота?'],
  },
  'pilot-fit-check': {
    id: 'pilot-fit-check',
    title: 'Pilot fit check - это короткий разговор вокруг одного реального workflow.',
    eyebrow: 'Pilot fit',
    body: [
      'Лучше всего подходят команды, которые уже используют AI coding tools и начинают чувствовать проблемы review, context, scope или proof.',
      'Первый полезный пилот - один видимый task-to-proof loop для одной команды, одного репозитория или одного workflow.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['Готов ли мой репозиторий к coding agents?', 'Что такое AI delivery drift?'],
  },
  'ai-delivery-drift': {
    id: 'ai-delivery-drift',
    title: 'AI delivery drift - это уход работы от исходного intent.',
    eyebrow: 'Drift',
    body: [
      'AI-assisted work может уходить от исходного scope, архитектуры или proof expectations даже тогда, когда код выглядит чисто.',
      'Команде потом приходится заново восстанавливать, почему изменение было сделано, осталось ли оно в границах и что действительно проверялось.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['Что такое contract-first execution?', 'Что значит proof before approval?'],
  },
  'review-ai-generated-changes': {
    id: 'review-ai-generated-changes',
    title: 'Review AI-generated changes не должен останавливаться на diff.',
    eyebrow: 'Review',
    body: [
      'Reviewer должен видеть, какой контракт исполнялся, какие проверки прошли, какие артефакты появились, что осталось непроверенным и какой риск принимается.',
      'Так human approval остается в loop, но решение становится менее расплывчатым.',
    ],
    sources: ['START_ASSISTANT_SECURITY_AND_PRIVACY.md', 'static-answers.md'],
    nextQuestions: ['Что значит proof before approval?', 'Готов ли мой репозиторий к coding agents?'],
  },
};

export const ruStartArtifacts: RuStartArtifact[] = [
  {
    title: 'Contract-first execution',
    body: 'Цель, scope, non-goals, затронутые зоны, проверки, артефакты и proof criteria до начала исполнения.',
    signal: 'Ограниченный work packet',
  },
  {
    title: 'Proof before approval',
    body: 'Review связывает контракт, diff, проверки, артефакты и оставшийся риск до human approval.',
    signal: 'Evidence over summary',
  },
  {
    title: 'Repo readiness',
    body: 'Репозиторий показывает run commands, safety boundaries, ownership, verification signals и recovery paths.',
    signal: 'Operate safely',
  },
  {
    title: 'AI delivery drift',
    body: 'Видимый способ держать AI-assisted work в границах intent, scope, architecture и proof expectations.',
    signal: 'Control drift',
  },
];

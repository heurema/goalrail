import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import App from './App';
import { render, userEvent } from '../test-utils';

const RG_INPUT = 'Добавить manual review перед proof approval';
const BT_INPUT = 'Разобрать PR с неясными критериями приёмки';

async function landOnClarification(scenarioInput: string) {
  const user = userEvent.setup();
  render(<App />);
  await user.click(screen.getByRole('button', { name: new RegExp(scenarioInput, 'i') }));
  await user.click(screen.getByRole('button', { name: /Запустить демо/i }));
  return user;
}

async function answerRGAll(
  user: ReturnType<typeof userEvent.setup>,
  picks: { scope: RegExp; decision: RegExp; fail: RegExp } = {
    scope: /только новые контракты/i,
    decision: /один назначенный reviewer/i,
    fail: /proof блокируется/i,
  },
) {
  await user.click(screen.getByRole('radio', { name: picks.scope }));
  await user.click(screen.getByRole('radio', { name: picks.decision }));
  await user.click(screen.getByRole('radio', { name: picks.fail }));
}

async function answerBTAll(
  user: ReturnType<typeof userEvent.setup>,
  picks: { boundary: RegExp; visible: RegExp; outcome: RegExp } = {
    boundary: /один repo \/ один кейс/i,
    visible: /критерии приёмки/i,
    outcome: /можно запускать пилот/i,
  },
) {
  await user.click(screen.getByRole('radio', { name: picks.boundary }));
  await user.click(screen.getByRole('radio', { name: picks.visible }));
  await user.click(screen.getByRole('radio', { name: picks.outcome }));
}

async function landOnContract(scenarioInput: string) {
  const user = await landOnClarification(scenarioInput);
  if (scenarioInput === RG_INPUT) {
    await answerRGAll(user);
  } else {
    await answerBTAll(user);
  }
  await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
  return user;
}

describe('Pilot intake landing — Phase 1 idle (still works)', () => {
  it('renders the contract-first hero, composer, and 5-step rail with only step 1 active', () => {
    render(<App />);

    expect(
      screen.getByRole('heading', { name: /Покажите задачу — мы проведём вас по контракту/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/GOAL INTAKE · Шаг 1 из 5/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Опишите задачу/i)).toBeInTheDocument();

    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items.length).toBe(5);
    expect(items[0].dataset.stepState).toBe('active');
    for (let i = 1; i < items.length; i += 1) {
      expect(items[i].dataset.stepState).toBe('muted');
    }
  });

  it('keeps the demo CTA disabled until trimmed input length reaches 20 characters', async () => {
    const user = userEvent.setup();
    render(<App />);

    const cta = screen.getByRole('button', { name: /Запустить демо/i });
    expect(cta).toBeDisabled();

    const textarea = screen.getByPlaceholderText(/Опишите задачу/i) as HTMLTextAreaElement;
    await user.type(textarea, '   short   ');
    expect(cta).toBeDisabled();

    await user.clear(textarea);
    await user.type(textarea, RG_INPUT);
    expect(cta).toBeEnabled();
  });
});

describe('Pilot intake landing — Phase 2 clarification (still works)', () => {
  it('moves from intake to clarification after clicking Запустить демо', async () => {
    await landOnClarification(RG_INPUT);

    expect(
      screen.getByRole('heading', { name: /Уточним границы перед контрактом/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/CLARIFICATION · Шаг 2 из 5/i)).toBeInTheDocument();
  });

  it('keeps "Подготовить контракт" disabled until all 3 questions are answered', async () => {
    const user = await landOnClarification(RG_INPUT);
    const cta = screen.getByRole('button', { name: /Подготовить контракт/i });
    expect(cta).toBeDisabled();

    await user.click(screen.getByRole('radio', { name: /только новые контракты/i }));
    expect(cta).toBeDisabled();
    await user.click(screen.getByRole('radio', { name: /один назначенный reviewer/i }));
    expect(cta).toBeDisabled();
    await user.click(screen.getByRole('radio', { name: /proof блокируется/i }));
    expect(cta).toBeEnabled();
  });
});

describe('Pilot intake landing — Phase 3 contract draft', () => {
  it('advances to Step 3 after clicking enabled "Подготовить контракт"', async () => {
    await landOnContract(RG_INPUT);

    expect(screen.getByText(/CONTRACT DRAFT · Шаг 3 из 5/i)).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: /Черновик контракта подготовлен/i }),
    ).toBeInTheDocument();

    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items[0].dataset.stepState).toBe('done');
    expect(items[1].dataset.stepState).toBe('done');
    expect(items[2].dataset.stepState).toBe('active');
    expect(items[2]).toHaveAttribute('aria-current', 'step');
    expect(items[3].dataset.stepState).toBe('muted');
    expect(items[4].dataset.stepState).toBe('muted');
  });

  it('renders structured manual_review_gate contract draft with reviewer rule and failure mode', async () => {
    await landOnContract(RG_INPUT);

    const draft = screen.getByRole('group', { name: /Поля контракта/i });
    expect(within(draft).getByText(/Ручной review gate перед proof approval/i)).toBeInTheDocument();
    expect(
      within(draft).getByText(/Заблокировать proof approval до ручного review decision/i),
    ).toBeInTheDocument();
    expect(within(draft).getByText(/Новые контракты в рамках одного demo-case\./i)).toBeInTheDocument();
    expect(within(draft).getByText(/Правило review \/ decision/i)).toBeInTheDocument();
    expect(within(draft).getByText(/^один назначенный reviewer$/i)).toBeInTheDocument();
    expect(within(draft).getByText(/^Failure mode$/i)).toBeInTheDocument();
    expect(within(draft).getByText(/^proof блокируется$/i)).toBeInTheDocument();
    expect(
      within(draft).getByText(/Proof нельзя утвердить без review decision\./i),
    ).toBeInTheDocument();
  });

  it('renders bounded_task contract draft without reviewer/failure rows', async () => {
    await landOnContract(BT_INPUT);

    const draft = screen.getByRole('group', { name: /Поля контракта/i });
    expect(within(draft).getByText(/Черновик рабочего контракта/i)).toBeInTheDocument();
    expect(within(draft).getByText(/Граница задачи явно зафиксирована\./i)).toBeInTheDocument();
    expect(within(draft).queryByText(/Правило review \/ decision/i)).not.toBeInTheDocument();
    expect(within(draft).queryByText(/^Failure mode$/i)).not.toBeInTheDocument();
  });

  it('shows zero-ambiguity message when manual_review_gate answers do not include unsure/tbd or risky combos', async () => {
    await landOnContract(RG_INPUT);

    const draft = screen.getByRole('group', { name: /Поля контракта/i });
    expect(within(draft).getByText(/Open ambiguity · 0/i)).toBeInTheDocument();
    expect(
      within(draft).getByText(/Критичных ambiguity не найдено для демо-черновика\./i),
    ).toBeInTheDocument();
    expect(within(draft).queryByText(/требует решения/i)).not.toBeInTheDocument();
  });

  it('renders structured ambiguity rows when user picks unsure/tbd answers', async () => {
    const user = await landOnClarification(RG_INPUT);
    await answerRGAll(user, {
      scope: /пока не уверен/i,
      decision: /пока не определено/i,
      fail: /нужно решить вручную/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));

    const ambiguityList = screen.getByRole('list', { name: /Открытые ambiguity/i });
    const items = ambiguityList.querySelectorAll('li');
    expect(items.length).toBe(3);

    const draft = screen.getByRole('group', { name: /Поля контракта/i });
    expect(within(draft).getByText(/Open ambiguity · 3/i)).toBeInTheDocument();
    expect(within(draft).getByText(/A-01/)).toBeInTheDocument();
    expect(within(draft).getByText(/Scope гейта пока не определён\./)).toBeInTheDocument();
    expect(
      within(draft).getByText(/Кто принимает review decision, пока не определено\./),
    ).toBeInTheDocument();
    expect(
      within(draft).getByText(
        /Нужно описать, кто принимает ручное решение и где оно фиксируется\./,
      ),
    ).toBeInTheDocument();
    expect(within(draft).getAllByText(/требует решения/i).length).toBe(3);
  });

  it('flags migration ambiguity for "новые и активные контракты" scope only', async () => {
    const user = await landOnClarification(RG_INPUT);
    await answerRGAll(user, {
      scope: /новые и активные контракты/i,
      decision: /один назначенный reviewer/i,
      fail: /proof блокируется/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));

    expect(
      screen.getByText(/Нужна отдельная политика для уже активных контрактов\./),
    ).toBeInTheDocument();
  });

  it('flags broad-scope ambiguity for bounded_task team-wide answer', async () => {
    const user = await landOnClarification(BT_INPUT);
    await answerBTAll(user, {
      boundary: /процесс всей команды/i,
      visible: /критерии приёмки/i,
      outcome: /можно запускать пилот/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));

    expect(
      screen.getByText(/Scope может быть слишком широким для короткого пилота\./),
    ).toBeInTheDocument();
  });

  it('updates the context strip on Step 3 to contract-draft format', async () => {
    await landOnContract(RG_INPUT);
    const ctx = screen.getByRole('group', { name: /Контекст демо/i });
    expect(ctx).toHaveTextContent(/contract-draft/);
    expect(ctx).toHaveTextContent(/появится после проверки/);
  });

  it('shows the "Что в черновике" right-side card on Step 3', async () => {
    await landOnContract(RG_INPUT);
    expect(screen.getByRole('heading', { name: /Что в черновике/i })).toBeInTheDocument();
    expect(screen.getByText(/^цель задачи$/i)).toBeInTheDocument();
    expect(screen.getByText(/^scope и out of scope$/i)).toBeInTheDocument();
    expect(screen.getByText(/^открытые ambiguity$/i)).toBeInTheDocument();
  });

  it('keeps safety messaging visible on Step 3', async () => {
    await landOnContract(RG_INPUT);
    expect(screen.getByText(/^код не выполняется$/i)).toBeInTheDocument();
    expect(screen.getByText(/^repo не подключается$/i)).toBeInTheDocument();
  });

  it('"Перейти к проверке" advances locally to Step 4 review', async () => {
    const user = await landOnContract(RG_INPUT);

    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));

    expect(screen.getByText(/REVIEW · Шаг 4 из 5/i)).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: /Проверка рисков и готовности/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('heading', { name: /Черновик контракта подготовлен/i }),
    ).not.toBeInTheDocument();

    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items[2].dataset.stepState).toBe('done');
    expect(items[3].dataset.stepState).toBe('active');
    expect(items[3]).toHaveAttribute('aria-current', 'step');
    expect(items[4].dataset.stepState).toBe('muted');
  });

  it('"Вернуться к уточнениям" returns to Step 2 and preserves selected answers', async () => {
    const user = await landOnContract(RG_INPUT);

    await user.click(screen.getByRole('button', { name: /Вернуться к уточнениям/i }));

    expect(
      screen.queryByRole('heading', { name: /Черновик контракта подготовлен/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: /Уточним границы перед контрактом/i }),
    ).toBeInTheDocument();

    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items[0].dataset.stepState).toBe('done');
    expect(items[1].dataset.stepState).toBe('active');
    expect(items[2].dataset.stepState).toBe('muted');

    expect(screen.getByRole('radio', { name: /только новые контракты/i })).toHaveAttribute(
      'aria-checked',
      'true',
    );
    expect(screen.getByRole('radio', { name: /один назначенный reviewer/i })).toHaveAttribute(
      'aria-checked',
      'true',
    );
    expect(screen.getByRole('radio', { name: /proof блокируется/i })).toHaveAttribute(
      'aria-checked',
      'true',
    );
    expect(screen.getByRole('button', { name: /Подготовить контракт/i })).toBeEnabled();
  });

  it('"Изменить запрос" from contract-step tertiary returns to Step 1, preserves intake text, clears answers', async () => {
    const user = await landOnContract(RG_INPUT);
    await user.click(screen.getAllByRole('button', { name: /Изменить запрос/i })[0]);

    const textarea = screen.getByPlaceholderText(/Опишите задачу/i) as HTMLTextAreaElement;
    expect(textarea.value).toBe(RG_INPUT);
    expect(screen.getByText(/GOAL INTAKE · Шаг 1 из 5/i)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Запустить демо/i }));
    expect(screen.getByText(/Ответы:\s*0\s*\/\s*3/i)).toBeInTheDocument();
  });
});

async function landOnReview(scenarioInput: string) {
  const user = await landOnContract(scenarioInput);
  await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
  return user;
}

describe('Pilot intake landing — Phase 4 review', () => {
  it('renders the review panel with eyebrow, title, body, and three sections', async () => {
    await landOnReview(RG_INPUT);

    expect(screen.getByText(/REVIEW · Шаг 4 из 5/i)).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: /Проверка рисков и готовности/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Это локальная демо-проверка черновика/i)).toBeInTheDocument();

    expect(
      screen.getByRole('heading', { name: /^Готовность к следующему шагу$/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Риски и ambiguity$/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Проверочный вывод$/i })).toBeInTheDocument();
  });

  it('keeps a compact intake summary and contract digest visible on Step 4', async () => {
    await landOnReview(RG_INPUT);

    const summary = screen.getByRole('region', { name: /Принятый запрос/i });
    expect(within(summary).getByText(RG_INPUT)).toBeInTheDocument();

    const digest = screen.getByRole('region', { name: /Сводка черновика/i });
    expect(
      within(digest).getByText(/Ручной review gate перед proof approval/i),
    ).toBeInTheDocument();
    expect(
      within(digest).getByText(/Перейти к проверке рисков и ambiguity/i),
    ).toBeInTheDocument();
  });

  it('renders blocking readiness items and risks for manual_review_gate with tbd/unsure answers', async () => {
    const user = await landOnClarification(RG_INPUT);
    await answerRGAll(user, {
      scope: /пока не уверен/i,
      decision: /пока не определено/i,
      fail: /нужно решить вручную/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));

    const checklist = screen.getByRole('list', { name: /Чек-лист готовности/i });
    const checklistText = checklist.textContent ?? '';
    expect(checklistText).toMatch(/Scope зафиксирован/);
    expect(checklistText).toMatch(/Review decision rule выбран/);
    expect(checklist.querySelectorAll('.reviewItemBadge-blocking').length).toBeGreaterThanOrEqual(2);

    const risks = screen.getByRole('list', { name: /^Риски$/i });
    expect(
      within(risks).getByText(/Владелец решения по review не определён/i),
    ).toBeInTheDocument();
    expect(within(risks).getByText(/Путь ручного решения не описан/i)).toBeInTheDocument();
    expect(within(risks).getByText(/Выполнение в демо симулируется/i)).toBeInTheDocument();

    expect(screen.getByText(/Нужны решения перед пилотом/i)).toBeInTheDocument();
    expect(
      screen.getByText(
        /Перед честным итогом нужно показать, какие решения блокируют пилот\./i,
      ),
    ).toBeInTheDocument();

    const ambiguityList = screen.getByRole('list', { name: /Открытые ambiguity/i });
    expect(ambiguityList.querySelectorAll('li').length).toBeGreaterThan(0);
  });

  it('renders all-ok readiness for manual_review_gate with clean answers and shows zero ambiguity', async () => {
    await landOnReview(RG_INPUT);

    expect(screen.getByText(/^Готово к следующему шагу$/i)).toBeInTheDocument();
    expect(
      screen.getByText(
        /Черновик достаточно ограничен, чтобы перейти к честному итогу демо\./i,
      ),
    ).toBeInTheDocument();

    const checklist = screen.getByRole('list', { name: /Чек-лист готовности/i });
    expect(checklist.querySelectorAll('.reviewItemBadge-ok').length).toBe(4);
    expect(checklist.querySelectorAll('.reviewItemBadge-blocking').length).toBe(0);
    expect(checklist.querySelectorAll('.reviewItemBadge-warning').length).toBe(0);

    expect(screen.getByText(/Open ambiguity · 0/i)).toBeInTheDocument();
    expect(
      screen.getByText(/Критичных ambiguity не найдено для демо-проверки\./i),
    ).toBeInTheDocument();

    const risks = screen.getByRole('list', { name: /^Риски$/i });
    const advisoryBadges = risks.querySelectorAll('.reviewRiskBadge-advisory');
    expect(advisoryBadges.length).toBe(1);
    expect(risks.querySelectorAll('.reviewRiskBadge-blocking').length).toBe(0);
  });

  it('renders blocking broad-scope risk for bounded_task team-process scope', async () => {
    const user = await landOnClarification(BT_INPUT);
    await answerBTAll(user, {
      boundary: /процесс всей команды/i,
      visible: /критерии приёмки/i,
      outcome: /можно запускать пилот/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));

    expect(
      screen.getByText(/Scope процесса команды слишком широкий/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/Нужны решения перед пилотом/i)).toBeInTheDocument();
    const risks = screen.getByRole('list', { name: /^Риски$/i });
    expect(risks.querySelectorAll('.reviewRiskBadge-blocking').length).toBeGreaterThan(0);
  });

  it('renders warning broad-scope risk for bounded_task multi-surface scope', async () => {
    const user = await landOnClarification(BT_INPUT);
    await answerBTAll(user, {
      boundary: /несколько частей продукта/i,
      visible: /критерии приёмки/i,
      outcome: /можно запускать пилот/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));

    expect(
      screen.getByText(/^Scope может быть слишком широким$/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/Готово с оговорками/i)).toBeInTheDocument();
    const risks = screen.getByRole('list', { name: /^Риски$/i });
    expect(risks.querySelectorAll('.reviewRiskBadge-warning').length).toBeGreaterThan(0);
    expect(risks.querySelectorAll('.reviewRiskBadge-blocking').length).toBe(0);
  });

  it('updates the context strip on Step 4 to review format', async () => {
    await landOnReview(RG_INPUT);
    const ctx = screen.getByRole('group', { name: /Контекст демо/i });
    expect(ctx).toHaveTextContent(/Формат/i);
    expect(ctx).toHaveTextContent(/^review$|review/i);
    expect(ctx).toHaveTextContent(/локальная демо-проверка/);
  });

  it('shows the "Что проверяется" right-side card on Step 4', async () => {
    await landOnReview(RG_INPUT);
    const heading = screen.getByRole('heading', { name: /Что проверяется/i });
    const card = heading.closest('section') as HTMLElement;
    expect(card).not.toBeNull();
    expect(within(card).getByText(/^готовность scope$/i)).toBeInTheDocument();
    expect(within(card).getByText(/^decision rule$/i)).toBeInTheDocument();
    expect(within(card).getByText(/^риски и ambiguity$/i)).toBeInTheDocument();
    expect(within(card).getByText(/^следующий шаг$/i)).toBeInTheDocument();
  });

  it('keeps safety messaging visible on Step 4', async () => {
    await landOnReview(RG_INPUT);
    expect(screen.getByText(/^код не выполняется$/i)).toBeInTheDocument();
    expect(screen.getByText(/^repo не подключается$/i)).toBeInTheDocument();
  });

  it('"Подготовить итог" advances locally to Step 5 outcome', async () => {
    const user = await landOnReview(RG_INPUT);

    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));

    expect(screen.getByText(/OUTCOME · Шаг 5 из 5/i)).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: /Честный итог демо/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('heading', { name: /Проверка рисков и готовности/i }),
    ).not.toBeInTheDocument();

    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items[3].dataset.stepState).toBe('done');
    expect(items[4].dataset.stepState).toBe('active');
    expect(items[4]).toHaveAttribute('aria-current', 'step');
  });

  it('"Вернуться к черновику" returns to Step 3 with the contract draft preserved', async () => {
    const user = await landOnReview(RG_INPUT);
    await user.click(screen.getByRole('button', { name: /Вернуться к черновику/i }));

    expect(
      screen.queryByRole('heading', { name: /Проверка рисков и готовности/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: /Черновик контракта подготовлен/i }),
    ).toBeInTheDocument();

    const draft = screen.getByRole('group', { name: /Поля контракта/i });
    expect(within(draft).getByText(/Ручной review gate перед proof approval/i)).toBeInTheDocument();
    expect(within(draft).getByText(/^один назначенный reviewer$/i)).toBeInTheDocument();

    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items[2].dataset.stepState).toBe('active');
    expect(items[3].dataset.stepState).toBe('muted');
  });
});

async function landOnOutcome(scenarioInput: string) {
  const user = await landOnReview(scenarioInput);
  await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));
  return user;
}

describe('Pilot intake landing — Phase 5 honest outcome', () => {
  it('renders the outcome panel with eyebrow, title, and four sections on Step 5', async () => {
    await landOnOutcome(RG_INPUT);

    expect(screen.getByText(/OUTCOME · Шаг 5 из 5/i)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /Честный итог демо/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Что стало ясно$/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Что осталось открытым$/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Что демо не делало$/i })).toBeInTheDocument();

    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items[0].dataset.stepState).toBe('done');
    expect(items[1].dataset.stepState).toBe('done');
    expect(items[2].dataset.stepState).toBe('done');
    expect(items[3].dataset.stepState).toBe('done');
    expect(items[4].dataset.stepState).toBe('active');
    expect(items[4]).toHaveAttribute('aria-current', 'step');
  });

  it('renders ready verdict for clean manual_review_gate flow', async () => {
    await landOnOutcome(RG_INPUT);

    const verdict = screen.getByLabelText('Verdict');
    expect(verdict).toHaveAttribute('data-outcome-tone', 'ready');
    expect(within(verdict).getByText(/^ГОТОВ К ПИЛОТУ$/i)).toBeInTheDocument();
    expect(
      within(verdict).getByText(/^Кейс подходит для короткого пилота$/i),
    ).toBeInTheDocument();
    expect(
      within(verdict).getByText(/Контракт ограничен, ключевые решения зафиксированы/i),
    ).toBeInTheDocument();

    const outcomeRegion = screen.getByRole('region', { name: /Честный итог демо/i });
    expect(
      within(outcomeRegion).getByRole('button', { name: /^Обсудить пилот$/i }),
    ).toBeInTheDocument();

    const open = screen.getByRole('list', { name: /Что осталось открытым/i });
    expect(
      within(open).getByText(/критичных открытых вопросов в демо не осталось/i),
    ).toBeInTheDocument();
  });

  it('renders readyWithCaveats verdict when manual review answer is manual-decision (warning only)', async () => {
    const user = await landOnClarification(RG_INPUT);
    await answerRGAll(user, {
      scope: /только новые контракты/i,
      decision: /один назначенный reviewer/i,
      fail: /нужно решить вручную/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));

    const verdict = screen.getByLabelText('Verdict');
    expect(verdict).toHaveAttribute('data-outcome-tone', 'readyWithCaveats');
    expect(within(verdict).getByText(/^ПОДХОДИТ С ОГОВОРКАМИ$/i)).toBeInTheDocument();
    expect(
      within(verdict).getByText(/^Кейс можно брать в пилот, но с явными условиями$/i),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Обсудить оговорки$/i })).toBeInTheDocument();

    const open = screen.getByRole('list', { name: /Что осталось открытым/i });
    expect(within(open).getByText(/Путь ручного решения не описан/i)).toBeInTheDocument();
  });

  it('renders readyWithCaveats verdict for bounded_task multi-surface scope', async () => {
    const user = await landOnClarification(BT_INPUT);
    await answerBTAll(user, {
      boundary: /несколько частей продукта/i,
      visible: /критерии приёмки/i,
      outcome: /можно запускать пилот/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));

    const verdict = screen.getByLabelText('Verdict');
    expect(verdict).toHaveAttribute('data-outcome-tone', 'readyWithCaveats');
    expect(within(verdict).getByText(/^ПОДХОДИТ С ОГОВОРКАМИ$/i)).toBeInTheDocument();
  });

  it('renders blocked verdict when manual_review_gate has unsure/tbd answers', async () => {
    const user = await landOnClarification(RG_INPUT);
    await answerRGAll(user, {
      scope: /пока не уверен/i,
      decision: /пока не определено/i,
      fail: /нужно решить вручную/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));

    const verdict = screen.getByLabelText('Verdict');
    expect(verdict).toHaveAttribute('data-outcome-tone', 'blocked');
    expect(within(verdict).getByText(/^СНАЧАЛА НУЖНЫ РЕШЕНИЯ$/i)).toBeInTheDocument();
    expect(within(verdict).getByText(/^Кейс пока не готов к пилоту$/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Разобрать кейс$/i })).toBeInTheDocument();
  });

  it('renders blocked verdict for bounded_task team-process scope', async () => {
    const user = await landOnClarification(BT_INPUT);
    await answerBTAll(user, {
      boundary: /процесс всей команды/i,
      visible: /критерии приёмки/i,
      outcome: /можно запускать пилот/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));

    const verdict = screen.getByLabelText('Verdict');
    expect(verdict).toHaveAttribute('data-outcome-tone', 'blocked');
    expect(within(verdict).getByText(/^СНАЧАЛА НУЖНЫ РЕШЕНИЯ$/i)).toBeInTheDocument();
  });

  it('"Что демо не делало" includes safety items', async () => {
    await landOnOutcome(RG_INPUT);

    const list = screen.getByRole('list', { name: /Что демо не делало/i });
    expect(within(list).getByText(/^код не выполнялся$/i)).toBeInTheDocument();
    expect(within(list).getByText(/^repo не подключался$/i)).toBeInTheDocument();
    expect(
      within(list).getByText(/^результат не является выполненной задачей$/i),
    ).toBeInTheDocument();
  });

  it('updates the context strip on Step 5 to outcome format', async () => {
    await landOnOutcome(RG_INPUT);
    const ctx = screen.getByRole('group', { name: /Контекст демо/i });
    expect(ctx).toHaveTextContent(/outcome/);
    expect(ctx).toHaveTextContent(/метод показан/);
  });

  it('shows the "Что в итоге" right-side card on Step 5', async () => {
    await landOnOutcome(RG_INPUT);

    const heading = screen.getByRole('heading', { name: /Что в итоге/i });
    const card = heading.closest('section') as HTMLElement;
    expect(within(card).getByText(/^честный verdict$/i)).toBeInTheDocument();
    expect(within(card).getByText(/^что стало ясно$/i)).toBeInTheDocument();
    expect(within(card).getByText(/^без выполнения кода$/i)).toBeInTheDocument();
  });

  it('keeps safety messaging visible on Step 5', async () => {
    await landOnOutcome(RG_INPUT);
    expect(screen.getByText(/^код не выполняется$/i)).toBeInTheDocument();
    expect(screen.getByText(/^repo не подключается$/i)).toBeInTheDocument();
  });

  it('"Вернуться к проверке" returns to Step 4 with review preserved', async () => {
    const user = await landOnOutcome(RG_INPUT);
    await user.click(screen.getByRole('button', { name: /Вернуться к проверке/i }));

    expect(
      screen.queryByRole('heading', { name: /Честный итог демо/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: /Проверка рисков и готовности/i }),
    ).toBeInTheDocument();

    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items[3].dataset.stepState).toBe('active');
    expect(items[4].dataset.stepState).toBe('muted');

    expect(
      screen.getByRole('list', { name: /Чек-лист готовности/i }),
    ).toBeInTheDocument();
  });

  it('"Начать заново" returns to Step 1, clears textarea, and disables "Запустить демо"', async () => {
    const user = await landOnOutcome(RG_INPUT);
    await user.click(screen.getByRole('button', { name: /Начать заново/i }));

    expect(screen.queryByRole('heading', { name: /Честный итог демо/i })).not.toBeInTheDocument();
    const textarea = screen.getByPlaceholderText(/Опишите задачу/i) as HTMLTextAreaElement;
    expect(textarea.value).toBe('');
    expect(screen.getByRole('button', { name: /Запустить демо/i })).toBeDisabled();

    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items[0].dataset.stepState).toBe('active');
    for (let i = 1; i < items.length; i += 1) {
      expect(items[i].dataset.stepState).toBe('muted');
    }
  });

  it('primary outcome CTA focuses the existing email input without submitting to a backend', async () => {
    const user = await landOnOutcome(RG_INPUT);
    const emailInput = document.getElementById('pilot-email') as HTMLInputElement;
    expect(emailInput).not.toBeNull();
    expect(document.activeElement).not.toBe(emailInput);

    const outcomeRegion = screen.getByRole('region', { name: /Честный итог демо/i });
    await user.click(
      within(outcomeRegion).getByRole('button', { name: /^Обсудить пилот$/i }),
    );

    expect(document.activeElement).toBe(emailInput);
    expect(emailInput.value).toBe('');
  });
});

describe('Pilot intake landing — safe text rendering', () => {
  it('renders user input containing HTML/script-like content as text without executing scripts', async () => {
    const malicious = '<script>window.__pwned=true</script> добавить manual review перед proof approval';
    const user = userEvent.setup();
    render(<App />);

    const textarea = screen.getByPlaceholderText(/Опишите задачу/i) as HTMLTextAreaElement;
    await user.click(textarea);
    await user.paste(malicious);
    await user.click(screen.getByRole('button', { name: /Запустить демо/i }));

    expect((window as unknown as { __pwned?: boolean }).__pwned).toBeUndefined();
    const summary = screen.getByRole('region', { name: /Принятый запрос/i });
    expect(summary.textContent).toContain('<script>');
    expect(summary.querySelector('script')).toBeNull();

    await answerRGAll(user);
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));

    const step3Summary = screen.getByRole('region', { name: /Принятый запрос/i });
    expect(step3Summary.textContent).toContain('<script>');
    expect(step3Summary.querySelector('script')).toBeNull();
    expect((window as unknown as { __pwned?: boolean }).__pwned).toBeUndefined();

    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    const step4Summary = screen.getByRole('region', { name: /Принятый запрос/i });
    expect(step4Summary.textContent).toContain('<script>');
    expect(step4Summary.querySelector('script')).toBeNull();
    expect((window as unknown as { __pwned?: boolean }).__pwned).toBeUndefined();

    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));
    const step5Summary = screen.getByRole('region', { name: /Принятый запрос/i });
    expect(step5Summary.textContent).toContain('<script>');
    expect(step5Summary.querySelector('script')).toBeNull();
    expect((window as unknown as { __pwned?: boolean }).__pwned).toBeUndefined();
  });
});

describe('Pilot intake landing — no chat affordances', () => {
  it('does not render chat history, model selector, assistant turns, or file upload on any step', async () => {
    const user = userEvent.setup();
    render(<App />);

    const expectNoChat = () => {
      expect(screen.queryByRole('combobox', { name: /модель|model/i })).not.toBeInTheDocument();
      expect(screen.queryByRole('img', { name: /avatar|аватар/i })).not.toBeInTheDocument();
      expect(
        screen.queryByText(/история сообщений|chat history|assistant/i),
      ).not.toBeInTheDocument();
      expect(document.querySelector('input[type="file"]')).toBeNull();
    };

    expectNoChat();

    await user.click(screen.getByRole('button', { name: new RegExp(RG_INPUT, 'i') }));
    await user.click(screen.getByRole('button', { name: /Запустить демо/i }));
    expectNoChat();

    await answerRGAll(user);
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    expectNoChat();

    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    expectNoChat();

    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));
    expectNoChat();
  });
});

describe('Pilot intake landing — Phase 6A polish / hardening', () => {
  it('renders three individually addressable review-digest counters on Step 5', async () => {
    await landOnOutcome(RG_INPUT);

    const digest = screen.getByRole('region', { name: /Сводка проверки/i });
    expect(within(digest).getByLabelText(/^Блокеры:\s*0$/i)).toBeInTheDocument();
    expect(within(digest).getByLabelText(/^Оговорки:\s*0$/i)).toBeInTheDocument();
    expect(within(digest).getByLabelText(/^Открытых ambiguity:\s*0$/i)).toBeInTheDocument();

    const blockers = within(digest).getByLabelText(/^Блокеры:\s*0$/i);
    expect(blockers).toHaveAttribute('data-count', '0');
    expect(within(blockers).getByText('Блокеры')).toBeInTheDocument();
    expect(within(blockers).getByText('0')).toBeInTheDocument();
  });

  it('reflects non-zero counters when blocking/warning/ambiguity items exist', async () => {
    const user = await landOnClarification(RG_INPUT);
    await answerRGAll(user, {
      scope: /пока не уверен/i,
      decision: /пока не определено/i,
      fail: /нужно решить вручную/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));

    const digest = screen.getByRole('region', { name: /Сводка проверки/i });
    const blockers = within(digest).getByLabelText(/^Блокеры:/i);
    expect(blockers.getAttribute('data-count')).toMatch(/^[1-9]/);
    const warnings = within(digest).getByLabelText(/^Оговорки:/i);
    expect(warnings.getAttribute('data-count')).toMatch(/^[1-9]/);
    const ambiguity = within(digest).getByLabelText(/^Открытых ambiguity:/i);
    expect(ambiguity.getAttribute('data-count')).toMatch(/^[1-9]/);
  });

  it('marks the email CTA block as highlighted after the primary outcome CTA click', async () => {
    const user = await landOnOutcome(RG_INPUT);
    const ctaCard = screen
      .getByRole('region', { name: /Если вам это близко/i });
    expect(ctaCard).toHaveAttribute('data-cta-highlighted', 'false');

    const outcomeRegion = screen.getByRole('region', { name: /Честный итог демо/i });
    await user.click(
      within(outcomeRegion).getByRole('button', { name: /^Обсудить пилот$/i }),
    );

    expect(ctaCard).toHaveAttribute('data-cta-highlighted', 'true');
    expect(ctaCard.className).toMatch(/ctaCard--highlight/);

    const emailInput = document.getElementById('pilot-email') as HTMLInputElement;
    expect(document.activeElement).toBe(emailInput);
  });

  it('keeps step rail active item exposed via aria-current="step"', async () => {
    await landOnOutcome(RG_INPUT);
    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items.length).toBe(5);
    const current = Array.from(items).find((li) => li.getAttribute('aria-current') === 'step');
    expect(current).toBeDefined();
    expect(current!.dataset.stepState).toBe('active');
  });

  it('enforces a textarea maxLength around the intended 1500 character cap', () => {
    render(<App />);
    const textarea = screen.getByPlaceholderText(/Опишите задачу/i) as HTMLTextAreaElement;
    expect(textarea.maxLength).toBeGreaterThanOrEqual(1000);
    expect(textarea.maxLength).toBeLessThanOrEqual(1500);
  });

  it('keeps safety copy visible at intake and across walkthrough steps', async () => {
    const user = userEvent.setup();
    render(<App />);

    expect(
      screen.getByText(/Не вставляйте секреты, токены и приватные данные\./i),
    ).toBeInTheDocument();
    expect(screen.getByText(/^код не выполняется$/i)).toBeInTheDocument();
    expect(screen.getByText(/^repo не подключается$/i)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: new RegExp(RG_INPUT, 'i') }));
    await user.click(screen.getByRole('button', { name: /Запустить демо/i }));
    await answerRGAll(user);
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));

    expect(screen.getByText(/^код не выполняется$/i)).toBeInTheDocument();
    expect(screen.getByText(/^repo не подключается$/i)).toBeInTheDocument();
  });
});

describe('Pilot intake landing — Phase 6B accessibility / final polish', () => {
  it('renders a skip link that targets the main content', () => {
    render(<App />);
    const link = screen.getByRole('link', { name: /К основному содержимому/i });
    expect(link).toHaveAttribute('href', '#main-content');
  });

  it('marks the main content area with id="main-content"', () => {
    render(<App />);
    const mainNodes = document.querySelectorAll('main');
    expect(mainNodes.length).toBeGreaterThanOrEqual(1);
    const mainContent = document.getElementById('main-content');
    expect(mainContent).not.toBeNull();
    expect(mainContent!.tagName.toLowerCase()).toBe('main');
  });

  it('describes the intake textarea via aria-describedby pointing at the safety help text', () => {
    render(<App />);
    const textarea = screen.getByPlaceholderText(/Опишите задачу/i) as HTMLTextAreaElement;
    const describedBy = textarea.getAttribute('aria-describedby');
    expect(describedBy).toBeTruthy();
    const help = document.getElementById(describedBy!);
    expect(help).not.toBeNull();
    expect(help!.textContent).toMatch(
      /Не вставляйте секреты, токены и приватные данные\./i,
    );
  });

  it('renders the updated risk title "Владелец решения по review не определён" for tbd reviewer', async () => {
    const user = await landOnClarification(RG_INPUT);
    await answerRGAll(user, {
      scope: /только новые контракты/i,
      decision: /пока не определено/i,
      fail: /proof блокируется/i,
    });
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));

    expect(
      screen.getByText(/Владелец решения по review не определён/i),
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/Владелец review decision не определён/i),
    ).not.toBeInTheDocument();
  });

  it('exposes exactly four notDone safety boundary items in the outcome', async () => {
    await landOnOutcome(RG_INPUT);
    const list = screen.getByRole('list', { name: /Что демо не делало/i });
    const items = list.querySelectorAll('li');
    expect(items.length).toBe(4);

    expect(within(list).getByText(/^код не выполнялся$/i)).toBeInTheDocument();
    expect(within(list).getByText(/^repo не подключался$/i)).toBeInTheDocument();
    expect(
      within(list).getByText(/^production-сущности не создавались$/i),
    ).toBeInTheDocument();
    expect(
      within(list).getByText(/^результат не является выполненной задачей$/i),
    ).toBeInTheDocument();
  });

  it('completes the full 5-step walkthrough with safety copy and rail aria-current intact', async () => {
    const user = userEvent.setup();
    render(<App />);

    // Step 1
    expect(
      screen.getByText(/Не вставляйте секреты, токены и приватные данные\./i),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: new RegExp(RG_INPUT, 'i') }));
    await user.click(screen.getByRole('button', { name: /Запустить демо/i }));

    // Step 2
    await answerRGAll(user);
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));

    // Step 3
    expect(
      screen.getByRole('heading', { name: /Черновик контракта подготовлен/i }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));

    // Step 4
    expect(
      screen.getByRole('heading', { name: /Проверка рисков и готовности/i }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));

    // Step 5
    expect(screen.getByRole('heading', { name: /Честный итог демо/i })).toBeInTheDocument();
    const rail = screen.getByRole('list', { name: /Шаги демо/i });
    const items = rail.querySelectorAll('li');
    expect(items.length).toBe(5);
    const current = Array.from(items).find(
      (li) => li.getAttribute('aria-current') === 'step',
    );
    expect(current).toBeDefined();
    expect(current!.dataset.stepState).toBe('active');

    // safety copy still visible across journey
    expect(screen.getByText(/^код не выполняется$/i)).toBeInTheDocument();
    expect(screen.getByText(/^repo не подключается$/i)).toBeInTheDocument();
  });

  it('keeps email focus and local highlight when primary outcome CTA is clicked', async () => {
    const user = await landOnOutcome(RG_INPUT);
    const ctaCard = screen.getByRole('region', { name: /Если вам это близко/i });
    expect(ctaCard).toHaveAttribute('data-cta-highlighted', 'false');

    const outcomeRegion = screen.getByRole('region', { name: /Честный итог демо/i });
    await user.click(
      within(outcomeRegion).getByRole('button', { name: /^Обсудить пилот$/i }),
    );

    expect(ctaCard).toHaveAttribute('data-cta-highlighted', 'true');
    const emailInput = document.getElementById('pilot-email') as HTMLInputElement;
    expect(document.activeElement).toBe(emailInput);
  });

  it('does not expose chat affordances after the full walkthrough', async () => {
    const user = userEvent.setup();
    render(<App />);
    await user.click(screen.getByRole('button', { name: new RegExp(RG_INPUT, 'i') }));
    await user.click(screen.getByRole('button', { name: /Запустить демо/i }));
    await answerRGAll(user);
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));

    expect(screen.queryByRole('combobox', { name: /модель|model/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('img', { name: /avatar|аватар/i })).not.toBeInTheDocument();
    expect(
      screen.queryByText(/история сообщений|chat history|assistant/i),
    ).not.toBeInTheDocument();
    expect(document.querySelector('input[type="file"]')).toBeNull();
  });
});

describe('Pilot intake landing — Phase 6C navigation hardening', () => {
  it('marks <main> with id="main-content", tabIndex=-1, and accessible name "Демо GoalRail"', () => {
    render(<App />);
    const mainContent = document.getElementById('main-content');
    expect(mainContent).not.toBeNull();
    expect(mainContent!.tagName.toLowerCase()).toBe('main');
    expect(mainContent!.getAttribute('tabindex')).toBe('-1');
    expect(mainContent!.getAttribute('aria-label')).toBe('Демо GoalRail');

    expect(screen.getByRole('main', { name: /Демо GoalRail/i })).toBe(mainContent);
  });

  it('skip-link points to #main-content and moves focus on click', async () => {
    const user = userEvent.setup();
    render(<App />);

    const link = screen.getByRole('link', { name: /К основному содержимому/i });
    expect(link).toHaveAttribute('href', '#main-content');

    const mainContent = document.getElementById('main-content');
    expect(mainContent).not.toBeNull();
    expect(document.activeElement).not.toBe(mainContent);

    await user.click(link);
    expect(document.activeElement).toBe(mainContent);
  });

  it('preserves the sidebar landmark accessible label', () => {
    render(<App />);
    expect(
      screen.getByRole('complementary', { name: /Разделы лендинга/i }),
    ).toBeInTheDocument();
  });

  it('keeps the right-column landmark accessible label after entering clarification', async () => {
    const user = userEvent.setup();
    render(<App />);

    expect(
      screen.getByRole('complementary', { name: /О демо/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: new RegExp(RG_INPUT, 'i') }));
    await user.click(screen.getByRole('button', { name: /Запустить демо/i }));

    expect(
      screen.getByRole('complementary', { name: /О демо/i }),
    ).toBeInTheDocument();
  });

  it('marks the rendered step block with data-skip-animation="true" after back-navigation', async () => {
    const user = await landOnOutcome(RG_INPUT);

    await user.click(screen.getByRole('button', { name: /Вернуться к проверке/i }));

    const reviewBlock = document.querySelector('.reviewBlock');
    expect(reviewBlock).not.toBeNull();
    expect(reviewBlock!.getAttribute('data-skip-animation')).toBe('true');
  });

  it('keeps animation enabled on forward navigation', async () => {
    const user = await landOnContract(RG_INPUT);
    const contractBlock = document.querySelector('.contractBlock');
    expect(contractBlock).not.toBeNull();
    expect(contractBlock!.getAttribute('data-skip-animation')).toBe('false');

    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    const reviewBlock = document.querySelector('.reviewBlock');
    expect(reviewBlock).not.toBeNull();
    expect(reviewBlock!.getAttribute('data-skip-animation')).toBe('false');
  });

  it('runs the full 5-step walkthrough end-to-end and keeps outcome safety boundaries explicit', async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(screen.getByRole('button', { name: new RegExp(RG_INPUT, 'i') }));
    await user.click(screen.getByRole('button', { name: /Запустить демо/i }));
    await answerRGAll(user);
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));

    expect(screen.getByRole('heading', { name: /Честный итог демо/i })).toBeInTheDocument();
    const notDoneList = screen.getByRole('list', { name: /Что демо не делало/i });
    const items = notDoneList.querySelectorAll('li');
    expect(items.length).toBe(4);
    expect(within(notDoneList).getByText(/^код не выполнялся$/i)).toBeInTheDocument();
    expect(within(notDoneList).getByText(/^repo не подключался$/i)).toBeInTheDocument();
    expect(
      within(notDoneList).getByText(/^production-сущности не создавались$/i),
    ).toBeInTheDocument();
    expect(
      within(notDoneList).getByText(/^результат не является выполненной задачей$/i),
    ).toBeInTheDocument();
  });

  it('keeps primary outcome CTA focusing email input and toggling local highlight', async () => {
    const user = await landOnOutcome(RG_INPUT);
    const ctaCard = screen.getByRole('region', { name: /Если вам это близко/i });
    expect(ctaCard).toHaveAttribute('data-cta-highlighted', 'false');

    const outcomeRegion = screen.getByRole('region', { name: /Честный итог демо/i });
    await user.click(
      within(outcomeRegion).getByRole('button', { name: /^Обсудить пилот$/i }),
    );

    expect(ctaCard).toHaveAttribute('data-cta-highlighted', 'true');
    const emailInput = document.getElementById('pilot-email') as HTMLInputElement;
    expect(document.activeElement).toBe(emailInput);
  });

  it('does not expose chat affordances at any step of the walkthrough', async () => {
    const user = userEvent.setup();
    render(<App />);

    const expectNoChat = () => {
      expect(screen.queryByRole('combobox', { name: /модель|model/i })).not.toBeInTheDocument();
      expect(screen.queryByRole('img', { name: /avatar|аватар/i })).not.toBeInTheDocument();
      expect(
        screen.queryByText(/история сообщений|chat history|assistant/i),
      ).not.toBeInTheDocument();
      expect(document.querySelector('input[type="file"]')).toBeNull();
    };

    expectNoChat();
    await user.click(screen.getByRole('button', { name: new RegExp(RG_INPUT, 'i') }));
    await user.click(screen.getByRole('button', { name: /Запустить демо/i }));
    expectNoChat();
    await answerRGAll(user);
    await user.click(screen.getByRole('button', { name: /Подготовить контракт/i }));
    expectNoChat();
    await user.click(screen.getByRole('button', { name: /Перейти к проверке/i }));
    expectNoChat();
    await user.click(screen.getByRole('button', { name: /Подготовить итог/i }));
    expectNoChat();
  });
});

describe('Pilot intake landing — lower email CTA', () => {
  it('keeps the email CTA at the bottom and visually quieter than the demo CTA', () => {
    render(<App />);

    const form = screen.getByRole('form', { name: /Форма заявки на пилот/i });
    expect(form).toHaveAttribute('action', 'mailto:hello@goalrail.dev');
    expect(screen.getByRole('button', { name: /Обсудить пилот/i })).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: /hello@goalrail\.dev/i }),
    ).toHaveAttribute('href', 'mailto:hello@goalrail.dev');
  });
});

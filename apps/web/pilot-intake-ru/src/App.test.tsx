import indexMarkup from '../index.html?raw';
import startIndexMarkup from '../start/index.html?raw';
import sidecarMainSourceText from '../server/cmd/goalrail-pilot-intake-ru/main.go?raw';
import configSourceText from '../server/internal/pilotlead/config.go?raw';
import digestSourceText from '../server/internal/pilotlead/digest.go?raw';
import handlerSourceText from '../server/internal/pilotlead/handler.go?raw';
import mailSourceText from '../server/internal/pilotlead/mail.go?raw';
import storeSourceText from '../server/internal/pilotlead/store.go?raw';
import timeSourceText from '../server/internal/pilotlead/time.go?raw';
import typesSourceText from '../server/internal/pilotlead/types.go?raw';
import appSourceText from './App.tsx?raw';
import startPageRuSourceText from './StartPageRu.tsx?raw';

import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import App from './App';
import { render, userEvent } from '../test-utils';

function appSource() {
  return appSourceText;
}

function startPageRuSource() {
  return startPageRuSourceText;
}

function indexHtml() {
  return indexMarkup;
}

function startIndexHtml() {
  return startIndexMarkup;
}

function leadEndpointSource() {
  return [
    sidecarMainSourceText,
    configSourceText,
    handlerSourceText,
    storeSourceText,
    timeSourceText,
    typesSourceText,
  ].join('\n');
}

function leadDigestSource() {
  return [
    sidecarMainSourceText,
    configSourceText,
    digestSourceText,
    storeSourceText,
    timeSourceText,
    typesSourceText,
  ].join('\n');
}

function pilotMailSource() {
  return [configSourceText, mailSourceText].join('\n');
}

afterEach(() => {
  vi.unstubAllGlobals();
  window.history.pushState({}, '', '/');
});

describe('Pilot intake RU business-first landing', () => {
  it('renders the hero title, 2-week framing, and primary CTA', () => {
    render(<App />);

    expect(screen.getByRole('heading', { name: 'ИИ-кодинг без хаоса' })).toBeInTheDocument();
    expect(screen.getByText(/2 недели · 1 команда · 1 участок продукта/i)).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'Обсудить пилот' }).length).toBeGreaterThan(0);
  });

  it('keeps the pilot landing on / and renders the RU start page only on /start', async () => {
    const { unmount } = render(<App />);

    expect(screen.getByRole('heading', { name: 'ИИ-кодинг без хаоса' })).toBeInTheDocument();

    unmount();
    window.history.pushState({}, '', '/start');
    render(<App />);

    expect(
      await screen.findByRole('heading', { name: 'Спросите GoalRail про AI-assisted delivery.' }),
    ).toBeInTheDocument();
    expect(screen.getByText('От бизнес-цели до проверенного изменения в коде.')).toBeInTheDocument();
  });

  it('renders the RU start page when static hosting serves /start/index.html directly', async () => {
    window.history.pushState({}, '', '/start/index.html');
    render(<App />);

    expect(
      await screen.findByRole('heading', { name: 'Спросите GoalRail про AI-assisted delivery.' }),
    ).toBeInTheDocument();
  });

  it('fills the RU start input from a quick question without calling the assistant', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    window.history.pushState({}, '', '/start');

    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole('button', { name: 'Что такое contract-first execution?' }));

    expect(screen.getByRole('textbox', { name: 'Спросить GoalRail' })).toHaveValue(
      'Что такое contract-first execution?',
    );
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('posts RU start questions to the same-origin start assistant endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          answer: 'GoalRail держит AI-разработку в границах контракта, проверок, proof и approval.',
          sources: [
            {
              title: 'Goalrail Global Start Assistant',
              path: 'docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md',
              section: 'Assistant behavior',
            },
          ],
          suggested_questions: ['Что значит proof before approval?'],
          knowledge: {
            updated_at: '2026-05-08T06:00:00Z',
            commit_sha: 'abc123',
          },
          disclaimer: 'Эта страница не может сканировать репозитории или выполнять код.',
        }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        },
      ),
    );
    vi.stubGlobal('fetch', fetchMock);
    window.history.pushState({}, '', '/start');

    const user = userEvent.setup();
    render(<App />);

    await user.type(await screen.findByRole('textbox', { name: 'Спросить GoalRail' }), 'Что такое GoalRail?');
    await user.click(screen.getByRole('button', { name: 'Спросить' }));

    await screen.findByRole('heading', { name: 'Ответ source-grounded ассистента' });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/start-chat',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question: 'Что такое GoalRail?' }),
      }),
    );
    expect(screen.getByText(/держит AI-разработку/i)).toBeInTheDocument();
    expect(screen.getByText('Goalrail Global Start Assistant / Assistant behavior')).toBeInTheDocument();
    expect(screen.getByText('Revision abc123')).toBeInTheDocument();
  });

  it('renders hero copy about existing AI tools and illustrative report status', () => {
    render(<App />);

    expect(
      screen.getByText(/Ваши разработчики уже используют Cursor, Claude Code, Codex или Copilot/i),
    ).toBeInTheDocument();
    expect(screen.getByText('Пример пилотного отчёта')).toBeInTheDocument();
    expect(screen.getByText(/Иллюстрация результата/i)).toBeInTheDocument();
  });

  it('renders problem and control-layer sections', () => {
    render(<App />);

    expect(screen.getByText(/ИИ ускоряет разработку/i)).toBeInTheDocument();
    expect(screen.getByText(/скрытый хаос/i)).toBeInTheDocument();
    expect(screen.getByText(/Проблема не в ИИ-инструментах/i)).toBeInTheDocument();
    expect(screen.getByText(/слоя контроля/i)).toBeInTheDocument();
  });

  it('renders the simple control-layer visual', () => {
    render(<App />);

    for (const label of ['AI-инструменты', 'GoalRail control layer', 'Безопасный пилот', 'Решение для бизнеса']) {
      expect(screen.getByRole('heading', { name: label })).toBeInTheDocument();
    }
  });

  it('renders the four GoalRail control cards', () => {
    render(<App />);

    for (const title of [
      'Аудит готовности',
      'Контекст проекта',
      'Контролируемые задачи',
      'Проверяемый результат',
    ]) {
      expect(screen.getByRole('heading', { name: title })).toBeInTheDocument();
    }
  });

  it('renders the pilot offer, repeatable process copy, and business output cards', () => {
    render(<App />);

    expect(
      screen.getByRole('heading', { name: 'Начните с безопасного пилота, а не с большого внедрения' }),
    ).toBeInTheDocument();
    expect(screen.getByText(/повторяемому процессу/i)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Готовность репозитория' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Контролируемая AI-задача' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Результат пилота' })).toBeInTheDocument();
  });

  it('renders fit, not-fit, and before/after sections', () => {
    render(<App />);

    expect(
      screen.getByRole('heading', { name: 'Для команд, которые уже пробуют ИИ-разработку' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Кому подходит' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Кому не подходит' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Что меняется после пилота' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'До пилота' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'После пилота' })).toBeInTheDocument();
  });

  it('renders FAQ with all five required questions', () => {
    render(<App />);

    expect(screen.getByRole('heading', { name: 'Частые вопросы' })).toBeInTheDocument();
    for (const question of [
      'Это заменяет разработчиков?',
      'Это альтернатива Cursor или Claude Code?',
      'Вы ускоряете разработку?',
      'Можно внедрить сразу во весь продукт?',
      'Что нужно от команды?',
    ]) {
      expect(screen.getByRole('heading', { name: question })).toBeInTheDocument();
    }
  });

  it('renders final CTA with pilot contact email and Telegram channel', () => {
    render(<App />);

    expect(
      screen.getByRole('heading', { name: 'Проверьте, готова ли ваша команда к AI-разработке' }),
    ).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'Обсудить пилот' }).length).toBeGreaterThan(0);
    expect(screen.getByText(/Без рассылок, трекинга, CRM/i)).toBeInTheDocument();
    expect(screen.getByRole('form', { name: 'Форма заявки на пилот' })).toHaveAttribute(
      'action',
      '/api/pilot-lead',
    );
    expect(screen.getByRole('link', { name: 'pilot@goalrail.dev' })).toHaveAttribute(
      'href',
      'mailto:pilot@goalrail.dev',
    );
    expect(screen.getByRole('link', { name: '@goalrail' })).toHaveAttribute(
      'href',
      'https://t.me/goalrail',
    );
  });

  it('focuses the email field from the primary CTA', async () => {
    const user = userEvent.setup();
    render(<App />);

    const hero = screen.getByRole('region', { name: 'ИИ-кодинг без хаоса' });
    await user.click(within(hero).getByRole('button', { name: 'Обсудить пилот' }));

    const emailInput = screen.getByLabelText('Рабочая почта');
    expect(document.activeElement).toBe(emailInput);
    expect(screen.getByRole('region', { name: 'Обсудить пилот' })).toHaveAttribute(
      'data-cta-highlighted',
      'true',
    );
  });

  it('posts a valid lead to the narrow lead-capture endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ ok: true }),
    });
    vi.stubGlobal('fetch', fetchMock);

    const user = userEvent.setup();
    render(<App />);

    await user.type(screen.getByLabelText('Рабочая почта'), 'buyer@example.com');
    await user.click(screen.getByRole('button', { name: 'Отправить заявку' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    expect(fetchMock).toHaveBeenCalledWith('/api/pilot-lead', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: 'buyer@example.com',
        source: 'ru-pilot',
        page: 'pilot.goalrail.ru',
        website: '',
      }),
    });
    expect(screen.getByText(/Спасибо\. Получили почту/i)).toBeInTheDocument();
  });

  it('shows fallback mail when lead submission fails', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ ok: false, error: 'mail_unavailable' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    const user = userEvent.setup();
    render(<App />);

    await user.type(screen.getByLabelText('Рабочая почта'), 'buyer@example.com');
    await user.click(screen.getByRole('button', { name: 'Отправить заявку' }));

    await screen.findByText(/Не удалось отправить заявку/i);
    expect(screen.getByText(/Напишите напрямую: pilot@goalrail.dev/i)).toBeInTheDocument();
  });

  it('renders a duplicate-state message for an already submitted email', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ ok: true, duplicate: true }),
    });
    vi.stubGlobal('fetch', fetchMock);

    const user = userEvent.setup();
    render(<App />);

    await user.type(screen.getByLabelText('Рабочая почта'), 'buyer@example.com');
    await user.click(screen.getByRole('button', { name: 'Отправить заявку' }));

    await screen.findByText(/Этот адрес уже есть в списке/i);
    expect(screen.getByText(/Повторно заявку не отправляем/i)).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });


  it('does not call the endpoint for invalid email and includes a honeypot field', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);

    const user = userEvent.setup();
    render(<App />);

    const honeypot = document.querySelector('input[name="website"]');
    expect(honeypot).toBeInTheDocument();
    expect(honeypot).toHaveAttribute('tabindex', '-1');
    expect(honeypot).toHaveAttribute('aria-hidden', 'true');
    expect(honeypot).toHaveAttribute('inert');
    expect(screen.queryByText('Не заполняйте это поле')).not.toBeInTheDocument();

    await user.type(screen.getByLabelText('Рабочая почта'), 'not-an-email');
    await user.click(screen.getByRole('button', { name: 'Отправить заявку' }));

    expect(fetchMock).not.toHaveBeenCalled();
    expect(screen.getByText('Введите рабочую почту.')).toBeInTheDocument();
  });

  it('does not render old technical-first UI terms', () => {
    render(<App />);

    for (const term of [
      'Goal Intake',
      'Contract Draft',
      'Open ambiguity',
      'Execution Evidence',
      'model selector',
      'chat history',
      'file upload',
    ]) {
      expect(screen.queryByText(new RegExp(term, 'i'))).not.toBeInTheDocument();
    }
  });

  it('keeps canonical metadata on pilot.goalrail.ru', () => {
    expect(indexHtml()).toContain('<link rel="canonical" href="https://pilot.goalrail.ru/" />');
  });

  it('serves RU start metadata from a static HTML entry', () => {
    const html = startIndexHtml();

    expect(html).toContain('<title>GoalRail - AI-разработка без потери контроля</title>');
    expect(html).toContain('<link rel="canonical" href="https://goalrail.ru/start" />');
    expect(html).toContain('<meta property="og:title" content="Спросите GoalRail про AI-assisted delivery" />');
    expect(html).toContain('<meta property="og:description" content="От бизнес-цели до проверенного изменения в коде." />');
  });

  it('keeps network behavior narrowed to the local lead endpoint only', () => {
    const source = `${appSource()}\n${indexHtml()}`;

    expect(source).toMatch(/fetch\('\/api\/pilot-lead'/);
    expect(source).not.toMatch(/fetch\s*\(\s*['"]https?:\/\//i);
    expect(source).not.toMatch(/XMLHttpRequest/i);
    expect(source).not.toMatch(/sendBeacon/i);
    expect(source).not.toMatch(/localStorage|sessionStorage|indexedDB/i);
    expect(source).not.toMatch(/gtag|googletagmanager|mixpanel|sentry|datadog/i);
    expect(source).not.toMatch(/input\s+type=["']file["']/i);
  });

  it('keeps the RU start page narrowed to the same-origin start assistant endpoint', () => {
    const source = startPageRuSource();

    expect(source).toMatch(/fetch\('\/api\/start-chat'/);
    expect(source).not.toMatch(/fetch\s*\(\s*['"]https?:\/\//i);
    expect(source).not.toMatch(/XMLHttpRequest/i);
    expect(source).not.toMatch(/sendBeacon/i);
    expect(source).not.toMatch(/localStorage|sessionStorage|indexedDB/i);
    expect(source).not.toMatch(/gtag|googletagmanager|mixpanel|sentry|datadog/i);
    expect(source).not.toMatch(/input\s+type=["']file["']/i);
  });

  it('keeps the Go sidecar endpoint narrow: local JSONL, mail notification, no external integrations', () => {
    const source = leadEndpointSource();

    expect(source).toContain('/srv/goalrail/pilot/leads/leads.jsonl');
    expect(source).toContain('/api/pilot-lead');
    expect(source).toContain('Пилот — заявка с RU лендинга');
    expect(source).toContain('MaxBodyBytes = 8192');
    expect(source).toContain('PrepareAttempt');
    expect(source).toContain('MarkNotificationResult');
    expect(source).toContain('StatusNotificationFailed');
    expect(source).toContain('StatusReceived');
    expect(source).toContain('MailRecipient');
    expect(source).toContain('SendText');
    expect(source).toContain('submitted_at_local');
    expect(source).toContain('submitted_date_local');
    expect(source).toContain('ValidateEmail');
    expect(source).not.toMatch(/google|sheets|crm|analytics|openai|anthropic|api\.github|api\.gitlab|database\/sql/i);
  });

  it('keeps the daily digest bounded to previous-day JSONL email only', () => {
    const source = leadDigestSource();

    expect(source).toContain('/srv/goalrail/pilot/leads/leads.jsonl');
    expect(source).toContain('DigestTZ           = "Europe/Moscow"');
    expect(source).toContain('GOALRAIL_DIGEST_DATE');
    expect(source).toContain('GOALRAIL_DIGEST_DRY_RUN');
    expect(source).toContain('AddDate(0, 0, -1)');
    expect(source).toContain('DigestRecords');
    expect(source).toContain('no_leads');
    expect(source).toContain('would_send');
    expect(source).toContain('SendText');
    expect(source).not.toMatch(/google|sheets|crm|analytics|openai|anthropic|api\.github|api\.gitlab|database\/sql/i);
  });


  it('keeps the Resend transport narrow and server-local', () => {
    const source = pilotMailSource();

    expect(source).toContain('https://api.resend.com/emails');
    expect(source).toContain('/srv/goalrail/pilot/backend/resend-api-key.local');
    expect(source).toContain('/srv/goalrail/pilot/backend/lead-recipient.local');
    expect(source).toContain('GoalRail Pilot <noreply@skill7.dev>');
    expect(source).toContain('pilot@goalrail.dev');
    expect(source).toContain('Authorization');
    expect(source).toContain('reply_to');
    expect(source).toContain('return "resend"');
    expect(source).toContain('return "sendmail"');
    expect(source).not.toMatch(/google|sheets|crm|analytics|openai|anthropic|api\.github|api\.gitlab|database\/sql/i);
  });
});

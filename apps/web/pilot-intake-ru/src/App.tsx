import { useEffect, useRef, useState } from 'react';

import './App.css';

type LeadSubmitState = 'idle' | 'submitting' | 'success' | 'error';

const topChips = ['2 НЕДЕЛИ', '1 КОМАНДА', '1 УЧАСТОК ПРОДУКТА'];

const readinessFields = [
  ['Готовность репозитория', '74 / 100'],
  ['Участок пилота', 'Внутренний админ-модуль'],
  ['Найдено рисков', '5'],
  ['Контролируемые задачи', 'Готово'],
];

const problemBullets = [
  'архитектура начинает разъезжаться;',
  'одинаковая логика появляется в разных местах;',
  'Пул-реквесты выглядят рабочими, но их сложнее объяснить;',
  'тесты не покрывают реальные риски;',
  'разработчики используют ИИ по-разному;',
  'бизнес видит скорость, но не видит риски.',
];


const controlLayerSteps = [
  {
    title: 'AI-инструменты',
    body: 'Cursor · Claude Code · Codex · Copilot',
  },
  {
    title: 'GoalRail control layer',
    body: 'аудит · контекст · правила задач · проверка',
  },
  {
    title: 'Безопасный пилот',
    body: 'один участок продукта · контролируемые AI-задачи · проверяемый результат',
  },
  {
    title: 'Решение для бизнеса',
    body: 'что масштабировать · что исправить · где AI пока рискован',
  },
];

const goalrailCards = [
  {
    title: 'Аудит готовности',
    body:
      'Проверяем, насколько ваш репозиторий готов к работе с ИИ-инструментами: структура, тесты, документация, контекст, риски и ограничения.',
  },
  {
    title: 'Контекст проекта',
    body:
      'Собираем базу знаний о проекте, чтобы ИИ и разработчики работали не вслепую, а с пониманием архитектуры и правил команды.',
  },
  {
    title: 'Контролируемые задачи',
    body:
      'Каждая ИИ-задача проходит через понятную рамку: цель, границы изменений, ограничения, проверки и ожидаемый результат.',
  },
  {
    title: 'Проверяемый результат',
    body:
      'Команда получает не просто сгенерированный код, а прозрачный процесс: что было сделано, почему так, как проверено и какие риски остались.',
  },
];

const pilotIncludes = [
  'выбор подходящего участка продукта;',
  'аудит репозитория;',
  'оценка готовности к ИИ-разработке;',
  'список рисков и блокеров;',
  'настройка проектного контекста;',
  'введение процесса ИИ-задач;',
  'запуск первого рабочего сценария;',
  'финальный отчет: что масштабировать, что исправить, где ИИ пока рискован.',
];

const readinessReady = ['изолированные модули', 'тесты в зоне пилота', 'понятные границы интерфейсов'];
const readinessRisks = [
  'дублирование бизнес-логики',
  'слабые интеграционные тесты',
  'неясная ответственность в модуле платежей',
];

const fits = [
  'у вас продуктовая команда от 3 до 20 разработчиков;',
  'разработчики уже используют Cursor, Claude Code, Codex, Copilot или похожие инструменты;',
  'вы хотите использовать ИИ активнее, но боитесь пускать его в критичный код;',
  'у вас есть участок продукта, на котором можно провести безопасный пилот;',
  'вы хотите не просто скорость, а контроль над качеством и архитектурой.',
];

const notFit = [
  'вы ждете полной автономной разработки без людей;',
  'вы хотите обещание десятикратной скорости;',
  'у вас огромный старый монолит и вы хотите сразу внедрить ИИ во все;',
  'команда еще вообще не использует ИИ-инструменты;',
  'вам нужен кастомный консалтинг без повторяемого процесса.',
];

const before = [
  'ИИ используется каждым разработчиком по-своему',
  'неясно, какие части кода безопасны для ИИ',
  'контекст проекта живет в головах людей',
  'задачи формулируются промптами без общих правил',
  'результат проверяется вручную и не всегда системно',
  'бизнес видит скорость, но не видит риски',
];

const after = [
  'понятно, где ИИ можно использовать безопасно',
  'есть оценка готовности репозитория',
  'есть база знаний проекта',
  'ИИ-задачи проходят через общий процесс',
  'результаты проверяются по понятным критериям',
  'бизнес видит не только скорость, но и управляемость',
];

const faqs = [
  {
    question: 'Это заменяет разработчиков?',
    answer:
      'Нет. GoalRail не заменяет разработчиков. Люди сохраняют продуктовые решения, ревью и ответственность за результат.',
  },
  {
    question: 'Это альтернатива Cursor или Claude Code?',
    answer:
      'Нет. GoalRail работает поверх Cursor, Claude Code, Codex, Copilot и похожих инструментов: добавляет общий контекст, рамки задач и проверку результата.',
  },
  {
    question: 'Вы ускоряете разработку?',
    answer:
      'Скорость может вырасти, но это не главное обещание. Главная цель — безопасная и предсказуемая ИИ-разработка без скрытого технического долга.',
  },
  {
    question: 'Можно внедрить сразу во весь продукт?',
    answer:
      'Не как первый шаг. Сначала нужен ограниченный пилот на одном участке, где риски управляемы и результат можно честно оценить.',
  },
  {
    question: 'Что нужно от команды?',
    answer:
      'Нужен инженерный владелец, доступ к репозиторию или репрезентативной кодовой базе, выбранный участок продукта и готовность проверить один реальный рабочий сценарий.',
  },
];

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function isValidEmail(value: string) {
  const email = value.trim();
  return email.length > 0 && email.length <= 254 && emailPattern.test(email);
}

function App() {
  const emailInputRef = useRef<HTMLInputElement | null>(null);
  const highlightTimerRef = useRef<number | null>(null);
  const [emailHighlighted, setEmailHighlighted] = useState(false);
  const [email, setEmail] = useState('');
  const [honeypot, setHoneypot] = useState('');
  const [leadSubmitState, setLeadSubmitState] = useState<LeadSubmitState>('idle');
  const [leadMessage, setLeadMessage] = useState('');

  useEffect(() => {
    return () => {
      if (highlightTimerRef.current !== null) {
        window.clearTimeout(highlightTimerRef.current);
      }
    };
  }, []);

  const focusContactArea = () => {
    const target = emailInputRef.current;
    if (!target) return;

    target.focus();
    target.scrollIntoView({ block: 'center', behavior: 'smooth' });
    setEmailHighlighted(true);

    if (highlightTimerRef.current !== null) {
      window.clearTimeout(highlightTimerRef.current);
    }
    highlightTimerRef.current = window.setTimeout(() => {
      setEmailHighlighted(false);
      highlightTimerRef.current = null;
    }, 2400);
  };

  const onSkipToMain = (event: React.MouseEvent<HTMLAnchorElement>) => {
    const target = document.getElementById('main-content');
    if (!target) return;

    event.preventDefault();
    target.focus();
    target.scrollIntoView({ block: 'start' });
  };

  const onLeadSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const submittedEmail = email.trim();
    if (!isValidEmail(submittedEmail)) {
      setLeadSubmitState('error');
      setLeadMessage('Введите рабочую почту.');
      emailInputRef.current?.focus();
      return;
    }

    setLeadSubmitState('submitting');
    setLeadMessage('');

    try {
      const response = await fetch('/api/pilot-lead', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: submittedEmail,
          source: 'ru-pilot',
          page: 'pilot.goalrail.ru',
          website: honeypot,
        }),
      });

      const body = (await response.json().catch(() => null)) as { duplicate?: boolean; ok?: boolean } | null;

      if (!response.ok || body?.ok !== true) {
        throw new Error('lead_submit_failed');
      }

      setLeadSubmitState('success');
      setLeadMessage(
        body.duplicate === true
          ? 'Этот адрес уже есть в списке. Повторно заявку не отправляем, чтобы не дублировать письма. Мы вернёмся с коротким следующим шагом.'
          : 'Спасибо. Получили почту — вернёмся с коротким следующим шагом.',
      );
      setEmail('');
      setHoneypot('');
    } catch {
      setLeadSubmitState('error');
      setLeadMessage('Не удалось отправить заявку. Напишите напрямую: hello@goalrail.dev');
    }
  };

  return (
    <div className="landingShell">
      <a className="skipLink" href="#main-content" onClick={onSkipToMain}>
        К основному содержимому
      </a>

      <div className="landingFrame">
        <header className="topbar" aria-label="GoalRail">
          <div className="brandBlock" aria-label="Пилот GoalRail">
            <span className="brandName mono">GoalRail</span>
            <span className="brandDivider" aria-hidden="true">/</span>
            <span className="brandSub">Пилот ИИ-разработки</span>
          </div>
          <div className="topChips" aria-label="Статус пилота">
            {topChips.map((chip) => (
              <span className="chip" key={chip}>{chip}</span>
            ))}
          </div>
        </header>

        <main id="main-content" className="page" tabIndex={-1} aria-label="Лендинг пилота ИИ-разработки">
          <section className="heroSection" aria-labelledby="hero-title">
            <div className="heroCopy">
              <p className="eyebrow mono">БЕЗОПАСНЫЙ ПИЛОТ ИИ-РАЗРАБОТКИ</p>
              <h1 id="hero-title" className="heroTitle">ИИ-кодинг без хаоса</h1>
              <p className="heroBody">
                Ваши разработчики уже используют Cursor, Claude Code, Codex или Copilot. GoalRail помогает внедрить ИИ-разработку как управляемый процесс: с аудитом репозитория, проектным контекстом, понятными задачами и проверяемым результатом.
              </p>
              <p className="heroSupport">
                Запустите безопасный 2-недельный пилот на одном участке продукта — не полную ИИ-трансформацию, а управляемую проверку качества, архитектуры и релизного контроля.
              </p>
              <p className="heroPilotMeta">2 недели · 1 команда · 1 участок продукта · без перестройки процесса</p>
              <div className="heroActions">
                <button className="primaryButton" type="button" onClick={focusContactArea}>
                  Обсудить пилот
                </button>
                <span className="heroNote">Короткая заявка без рассылок, трекинга и CRM.</span>
              </div>
            </div>

            <aside className="pilotReportCard" aria-label="Пример пилотного отчёта">
              <div className="reportHeader">
                <p className="reportKicker mono">Пример пилотного отчёта</p>
                <h2>Пилот ИИ-разработки</h2>
              </div>
              <dl className="reportFields">
                {readinessFields.map(([label, value]) => (
                  <div className="reportField" key={label}>
                    <dt>{label}</dt>
                    <dd>{value}</dd>
                  </div>
                ))}
              </dl>
              <div className="nextStepBox">
                <span className="nextStepLabel mono">Следующий шаг</span>
                <span className="nextStepValue">Запустить первый сценарий под контролем</span>
              </div>
              <p className="reportDisclaimer">
                Иллюстрация результата. Реальный аудит проводится отдельно.
              </p>
            </aside>
          </section>

          <section className="section problemSection" aria-labelledby="problem-title">
            <div className="sectionIntro">
              <p className="eyebrow mono">Проблема</p>
              <h2 id="problem-title">ИИ ускоряет разработку. Но без процесса он быстро создает скрытый хаос.</h2>
              <p>
                На старте все выглядит хорошо: разработчики быстрее пишут код, быстрее собирают прототипы и закрывают задачи. Проблема появляется позже: ИИ предлагает разные решения для похожих задач, дублирует логику, обходит архитектурные ограничения и меняет код без понятного объяснения.
              </p>
            </div>
            <ul className="bulletGrid" aria-label="Риски ИИ-разработки без процесса">
              {problemBullets.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </section>

          <section className="section controlSection" aria-labelledby="control-title">
            <div className="sectionIntro narrow">
              <p className="eyebrow mono">Слой контроля</p>
              <h2 id="control-title">Проблема не в ИИ-инструментах. Проблема в отсутствии слоя контроля.</h2>
              <p>
                Cursor, Claude Code, Codex и Copilot уже помогают разработчикам писать код. Но они не отвечают на вопросы бизнеса: готов ли репозиторий к ИИ-разработке, какие участки продукта безопасно отдавать в пилот, какие ограничения должен соблюдать ИИ, как проверить результат и как понять, что команда не накапливает скрытый технический долг.
              </p>
              <p className="closingLine">GoalRail добавляет этот слой контроля поверх ИИ-инструментов.</p>
            </div>
            <div className="controlFlow" aria-label="Как GoalRail добавляет слой контроля">
              {controlLayerSteps.map((step, index) => (
                <div className="controlFlowItem" key={step.title}>
                  <article className="controlFlowCard">
                    <h3>{step.title}</h3>
                    <p>{step.body}</p>
                  </article>
                  {index < controlLayerSteps.length - 1 ? (
                    <span className="controlFlowArrow" aria-hidden="true">↓</span>
                  ) : null}
                </div>
              ))}
            </div>
          </section>

          <section className="section" aria-labelledby="does-title">
            <div className="sectionIntro">
              <p className="eyebrow mono">Что делает GoalRail</p>
              <h2 id="does-title">GoalRail ставит ИИ-разработку на рельсы</h2>
            </div>
            <div className="featureGrid">
              {goalrailCards.map((card) => (
                <article className="card featureCard" key={card.title}>
                  <span className="cardRule" aria-hidden="true" />
                  <h3>{card.title}</h3>
                  <p>{card.body}</p>
                </article>
              ))}
            </div>
          </section>

          <section className="section offerSection" aria-labelledby="offer-title">
            <div className="sectionIntro">
              <p className="eyebrow mono">Пилотный формат</p>
              <h2 id="offer-title">Начните с безопасного пилота, а не с большого внедрения</h2>
              <p className="offerLead">
                Запустите безопасный 2-недельный пилот ИИ-разработки на одном участке продукта.
              </p>
              <p>
                Мы не предлагаем сразу перестраивать всю разработку. Пилот GoalRail запускается на одном выбранном участке продукта: модуле, внутреннем сервисе, админ-панели, интеграции или другой зоне, где можно проверить подход без риска для критичного ядра.
              </p>
              <p className="processNote">
                Пилот проводится по повторяемому процессу GoalRail, а не как разовый консалтинг-проект.
              </p>
            </div>
            <div className="offerGrid">
              <div className="card listCard">
                <h3>В пилот входит:</h3>
                <ol className="numberedList">
                  {pilotIncludes.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ol>
              </div>
              <div className="card importantCard">
                <p>
                  Цель пилота — не доказать, что ИИ может писать код. Цель — понять, как вашей команде использовать ИИ безопасно, системно и предсказуемо.
                </p>
              </div>
            </div>
          </section>

          <section className="section demoSection" aria-labelledby="demo-title">
            <div className="sectionIntro">
              <p className="eyebrow mono">Результаты пилота</p>
              <h2 id="demo-title">Как выглядит пилотный результат</h2>
              <p>Ниже примеры формата результата. Это иллюстрации, а не результаты реального сканирования.</p>
            </div>
            <div className="demoCards">
              <article className="card demoCard">
                <p className="demoLabel mono">Пример 1</p>
                <h3>Готовность репозитория</h3>
                <p className="scoreLine">Оценка готовности: <strong>74 / 100</strong></p>
                <div className="miniColumns">
                  <div>
                    <h4>Готово:</h4>
                    <ul>
                      {readinessReady.map((item) => <li key={item}>{item}</li>)}
                    </ul>
                  </div>
                  <div>
                    <h4>Риски:</h4>
                    <ul>
                      {readinessRisks.map((item) => <li key={item}>{item}</li>)}
                    </ul>
                  </div>
                </div>
              </article>

              <article className="card demoCard">
                <p className="demoLabel mono">Пример 2</p>
                <h3>Контролируемая AI-задача</h3>
                <dl className="taskSpec">
                  <div>
                    <dt>Задача:</dt>
                    <dd>Добавить экспорт отчетов в админ-панель</dd>
                  </div>
                  <div>
                    <dt>Цель:</dt>
                    <dd>Дать операторам выгружать отчеты с учетом выбранных фильтров.</dd>
                  </div>
                  <div>
                    <dt>Границы:</dt>
                    <dd>
                      <ul>
                        <li>Не менять логику платежей.</li>
                        <li>Не менять права пользователей.</li>
                        <li>Использовать существующий сервис экспорта.</li>
                      </ul>
                    </dd>
                  </div>
                  <div>
                    <dt>Проверки:</dt>
                    <dd>Модульные тесты · Интеграционный тест · Ручной чек-лист приемки</dd>
                  </div>
                </dl>
              </article>

              <article className="card demoCard">
                <p className="demoLabel mono">Пример 3</p>
                <h3>Результат пилота</h3>
                <p><strong>Результат:</strong> задача прошла через управляемый процесс ИИ-разработки</p>
                <p><strong>Подтверждения:</strong></p>
                <ul>
                  <li>измененные файлы проверены</li>
                  <li>тесты прошли</li>
                  <li>риски зафиксированы</li>
                  <li>следующие шаги созданы</li>
                </ul>
                <p className="recommendation"><strong>Рекомендация:</strong> можно безопасно запустить еще 2–3 задачи в этой зоне продукта.</p>
              </article>
            </div>
          </section>

          <section className="section fitSection" aria-labelledby="fit-title">
            <div className="sectionIntro">
              <p className="eyebrow mono">Кому подходит</p>
              <h2 id="fit-title">Для команд, которые уже пробуют ИИ-разработку</h2>
            </div>
            <div className="fitGrid">
              <article className="card fitCard">
                <h3>Кому подходит</h3>
                <ul>
                  {fits.map((item) => <li key={item}>{item}</li>)}
                </ul>
              </article>
              <article className="card fitCard mutedCard">
                <h3>Кому не подходит</h3>
                <ul>
                  {notFit.map((item) => <li key={item}>{item}</li>)}
                </ul>
              </article>
            </div>
          </section>

          <section className="section beforeAfterSection" aria-labelledby="before-after-title">
            <div className="sectionIntro">
              <p className="eyebrow mono">До и после</p>
              <h2 id="before-after-title">Что меняется после пилота</h2>
            </div>
            <div className="comparisonGrid">
              <article className="card compareCard">
                <h3>До пилота</h3>
                <ul>
                  {before.map((item) => <li key={item}>{item}</li>)}
                </ul>
              </article>
              <article className="card compareCard afterCard">
                <h3>После пилота</h3>
                <ul>
                  {after.map((item) => <li key={item}>{item}</li>)}
                </ul>
              </article>
            </div>
          </section>

          <section className="section faqSection" aria-labelledby="faq-title">
            <div className="sectionIntro">
              <p className="eyebrow mono">Вопросы</p>
              <h2 id="faq-title">Частые вопросы</h2>
            </div>
            <div className="faqList">
              {faqs.map((faq) => (
                <article className="card faqItem" key={faq.question}>
                  <h3>{faq.question}</h3>
                  <p>{faq.answer}</p>
                </article>
              ))}
            </div>
          </section>

          <section className="section finalCtaSection" aria-labelledby="final-cta-title">
            <div className="finalCtaCopy">
              <p className="eyebrow mono">Следующий шаг</p>
              <h2 id="final-cta-title">Проверьте, готова ли ваша команда к AI-разработке</h2>
              <p>
                Начните с ограниченного пилота. Мы поможем выбрать безопасный участок продукта, провести аудит и запустить первый управляемый рабочий сценарий ИИ-разработки.
              </p>
              <button className="primaryButton" type="button" onClick={focusContactArea}>
                Обсудить пилот
              </button>
            </div>
          </section>

          <section
            className={`card ctaCard${emailHighlighted ? ' ctaCard--highlight' : ''}`}
            aria-labelledby="contact-title"
            data-cta-highlighted={emailHighlighted ? 'true' : 'false'}
          >
            <div className="contactCopy">
              <p className="eyebrow mono">Ручной контакт</p>
              <h2 id="contact-title">Обсудить пилот</h2>
              <p>Оставьте рабочую почту — мы получим уведомление и ответим вручную по делу.</p>
              <p className="manualContactNote">Без рассылок, трекинга, CRM и автоматической воронки. Если форма не сработает, напишите напрямую.</p>
            </div>
            <form
              className="ctaForm"
              action="/api/pilot-lead"
              method="post"
              aria-label="Форма заявки на пилот"
              onSubmit={onLeadSubmit}
              noValidate
            >
              <label className="srOnly" htmlFor="pilot-email">Рабочая почта</label>
              <input
                ref={emailInputRef}
                className="textInput"
                id="pilot-email"
                name="email"
                type="email"
                placeholder="ваша@компания.ru"
                autoComplete="email"
                value={email}
                onChange={(event) => {
                  setEmail(event.currentTarget.value);
                  if (leadSubmitState !== 'idle') {
                    setLeadSubmitState('idle');
                    setLeadMessage('');
                  }
                }}
                required
              />
              <input
                className="honeypotField"
                id="pilot-website"
                name="website"
                type="text"
                tabIndex={-1}
                autoComplete="off"
                aria-hidden="true"
                inert
                value={honeypot}
                onChange={(event) => setHoneypot(event.currentTarget.value)}
              />
              <button className="ghostButton" type="submit" disabled={leadSubmitState === 'submitting'}>
                {leadSubmitState === 'submitting' ? 'Отправляем…' : 'Отправить заявку'}
              </button>
            </form>
            {leadMessage ? (
              <p
                className={`leadStatus leadStatus--${leadSubmitState}`}
                role={leadSubmitState === 'error' ? 'alert' : 'status'}
              >
                {leadMessage}
              </p>
            ) : null}
            <p className="ctaNote">
              Прямая почта на случай сбоя:{' '}
              <a href="mailto:hello@goalrail.dev">hello@goalrail.dev</a>
            </p>
          </section>
        </main>
      </div>
    </div>
  );
}

export default App;

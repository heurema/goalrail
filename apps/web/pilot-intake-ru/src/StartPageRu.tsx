import { FormEvent, useEffect, useMemo, useRef, useState } from 'react';

import { ruStartAnswers, ruStartArtifacts, ruStartQuestions } from './startPageRuData';
import './StartPageRu.css';

const RU_START_PAGE_TITLE = 'GoalRail - AI-разработка без потери контроля';
const RU_START_PAGE_DESCRIPTION =
  'GoalRail помогает командам использовать AI coding tools с контрактами, проверками, доказательствами и human approval.';
const RU_START_PAGE_OG_TITLE = 'Спросите GoalRail про AI-assisted delivery';
const RU_START_PAGE_OG_DESCRIPTION = 'От бизнес-цели до проверенного изменения в коде.';
const RU_START_PAGE_CANONICAL = 'https://goalrail.ru/start';
const RU_START_ASSISTANT_UNAVAILABLE =
  'Публичный ассистент GoalRail временно недоступен. Статический обзор и артефакты ниже остаются доступными.';

type LiveAssistantStatus = 'idle' | 'loading' | 'answered' | 'error';

interface LiveAssistantSource {
  title: string;
  path: string;
  section?: string | null;
}

interface LiveAssistantResponse {
  answer: string;
  sources: LiveAssistantSource[];
  suggested_questions: string[];
  knowledge?: {
    updated_at?: string | null;
    commit_sha?: string | null;
  };
  disclaimer?: string;
}

type LiveAssistantPayload = Partial<LiveAssistantResponse> & { message?: string };

async function readLiveAssistantPayload(response: Response): Promise<LiveAssistantPayload> {
  const contentType = response.headers.get('Content-Type') ?? '';

  if (!contentType.toLowerCase().includes('application/json')) {
    return {};
  }

  try {
    return (await response.json()) as LiveAssistantPayload;
  } catch {
    return {};
  }
}

function upsertNamedMeta(name: string, content: string) {
  let meta = document.head.querySelector<HTMLMetaElement>(`meta[name="${name}"]`);

  if (!meta) {
    meta = document.createElement('meta');
    meta.setAttribute('name', name);
    document.head.appendChild(meta);
  }

  meta.setAttribute('content', content);
}

function upsertPropertyMeta(property: string, content: string) {
  let meta = document.head.querySelector<HTMLMetaElement>(`meta[property="${property}"]`);

  if (!meta) {
    meta = document.createElement('meta');
    meta.setAttribute('property', property);
    document.head.appendChild(meta);
  }

  meta.setAttribute('content', content);
}

function upsertCanonicalLink(href: string) {
  let link = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]');

  if (!link) {
    link = document.createElement('link');
    link.setAttribute('rel', 'canonical');
    document.head.appendChild(link);
  }

  link.setAttribute('href', href);
}

function StartPageRu() {
  const [activeQuestionId, setActiveQuestionId] = useState(ruStartQuestions[0].id);
  const [liveQuestion, setLiveQuestion] = useState('');
  const [liveStatus, setLiveStatus] = useState<LiveAssistantStatus>('idle');
  const [liveAnswer, setLiveAnswer] = useState<LiveAssistantResponse | null>(null);
  const [liveError, setLiveError] = useState('');
  const askPanelRef = useRef<HTMLElement | null>(null);
  const liveQuestionInputRef = useRef<HTMLTextAreaElement | null>(null);
  const activeQuestion = useMemo(
    () => ruStartQuestions.find((question) => question.id === activeQuestionId) ?? ruStartQuestions[0],
    [activeQuestionId],
  );
  const activeAnswer = ruStartAnswers[activeQuestion.answerId] ?? ruStartAnswers[ruStartQuestions[0].answerId];
  const liveAnswerSources = liveAnswer
    ? liveAnswer.sources.map((source) =>
        source.section ? `${source.title} / ${source.section}` : source.title || source.path,
      )
    : [];

  useEffect(() => {
    document.title = RU_START_PAGE_TITLE;
    upsertNamedMeta('description', RU_START_PAGE_DESCRIPTION);
    upsertPropertyMeta('og:title', RU_START_PAGE_OG_TITLE);
    upsertPropertyMeta('og:description', RU_START_PAGE_OG_DESCRIPTION);
    upsertPropertyMeta('og:type', 'website');
    upsertCanonicalLink(RU_START_PAGE_CANONICAL);
  }, []);

  useEffect(() => {
    const input = liveQuestionInputRef.current;
    if (!input) return;

    input.style.height = 'auto';
    input.style.height = `${input.scrollHeight}px`;
  }, [liveQuestion]);

  async function handleLiveQuestionSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const question = liveQuestion.trim();
    if (!question || liveStatus === 'loading') {
      return;
    }

    setLiveStatus('loading');
    setLiveError('');

    try {
      const response = await fetch('/api/start-chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question }),
      });
      const payload = await readLiveAssistantPayload(response);

      if (!response.ok || typeof payload.answer !== 'string') {
        throw new Error(payload.message || RU_START_ASSISTANT_UNAVAILABLE);
      }

      setLiveAnswer({
        answer: payload.answer,
        sources: Array.isArray(payload.sources) ? payload.sources : [],
        suggested_questions: Array.isArray(payload.suggested_questions) ? payload.suggested_questions : [],
        knowledge: payload.knowledge,
        disclaimer: payload.disclaimer,
      });
      setLiveStatus('answered');
    } catch (error) {
      setLiveAnswer(null);
      setLiveStatus('error');
      setLiveError(error instanceof Error ? error.message : RU_START_ASSISTANT_UNAVAILABLE);
    }
  }

  function selectStaticQuestion(questionId: string) {
    const question = ruStartQuestions.find((item) => item.id === questionId) ?? ruStartQuestions[0];

    setActiveQuestionId(questionId);
    setLiveQuestion(question.label);
    setLiveAnswer(null);
    setLiveStatus('idle');
    setLiveError('');

    const focusPrompt = () => {
      askPanelRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' });
      liveQuestionInputRef.current?.focus({ preventScroll: true });
    };

    if (typeof window.requestAnimationFrame === 'function') {
      window.requestAnimationFrame(focusPrompt);
      return;
    }

    focusPrompt();
  }

  return (
    <main className="ruStartPage">
      <div className="ruStartRails" aria-hidden="true" />
      <div className="ruStartShell">
        <header className="ruStartTopbar">
          <div className="ruStartBrand" aria-label="GoalRail start">
            <span className="ruStartBrandText">GOALRAIL</span>
          </div>
          <nav className="ruStartTopLinks" aria-label="Навигация стартовой страницы">
            <a href="#questions">Вопросы</a>
            <a href="#artifacts">Артефакты</a>
            <a href="https://pilot.goalrail.ru/" rel="noreferrer">
              Пилот
            </a>
          </nav>
        </header>

        <section className="ruStartHero" aria-labelledby="ru-start-title">
          <div className="ruStartHeroCopy">
            <p className="ruStartKicker">Публичный старт</p>
            <h1 id="ru-start-title">Спросите GoalRail про AI-assisted delivery.</h1>
            <p className="ruStartSubtitle">От бизнес-цели до проверенного изменения в коде.</p>
            <p className="ruStartBody">
              GoalRail - слой контроля для команд, которые используют AI coding tools и не хотят терять intent,
              scope, checks, proof и approval.
            </p>
          </div>

          <section className="ruStartAskPanel" aria-label="Публичный ассистент GoalRail" ref={askPanelRef}>
            <div className="ruStartPanelHeader">
              <span>Public GoalRail KB</span>
              <span className="ruStartPanelStatus">
                {liveStatus === 'loading' ? 'отвечает' : liveStatus === 'answered' ? 'ответ' : 'live'}
              </span>
            </div>
            <form className="ruStartAskBox" onSubmit={handleLiveQuestionSubmit}>
              <textarea
                aria-label="Спросить GoalRail"
                disabled={liveStatus === 'loading'}
                onChange={(event) => setLiveQuestion(event.target.value)}
                placeholder="Спросите про готовность репозитория, контракт, proof, approval или drift AI-разработки..."
                ref={liveQuestionInputRef}
                rows={2}
                value={liveQuestion}
              />
              <button aria-label="Спросить" disabled={!liveQuestion.trim() || liveStatus === 'loading'} type="submit">
                {liveStatus === 'loading' ? (
                  <span className="ruStartAskSpinner" aria-hidden="true" />
                ) : (
                  <span aria-hidden="true">↑</span>
                )}
              </button>
            </form>
            <p className={liveStatus === 'error' ? 'ruStartAskNote error' : 'ruStartAskNote'}>
              {liveStatus === 'error'
                ? liveError
                : 'Ответы используют только публичные материалы GoalRail. Статический guide остается ниже.'}
            </p>
            {liveAnswer ? (
              <article className="ruStartLiveAnswer" aria-live="polite">
                <div className="ruStartLiveAnswerHeader">
                  <p className="ruStartKicker">Public KB answer</p>
                  <h2>Ответ source-grounded ассистента</h2>
                </div>
                <p>{liveAnswer.answer}</p>
                {liveAnswer.disclaimer ? <p className="ruStartAnswerDisclaimer">{liveAnswer.disclaimer}</p> : null}
                <div className="ruStartLiveMeta" aria-label="Источники и свежесть ответа">
                  {liveAnswer.knowledge?.updated_at || liveAnswer.knowledge?.commit_sha ? (
                    <div>
                      <h3>Knowledge</h3>
                      <ul>
                        {liveAnswer.knowledge.updated_at ? <li>Updated {liveAnswer.knowledge.updated_at}</li> : null}
                        {liveAnswer.knowledge.commit_sha ? <li>Revision {liveAnswer.knowledge.commit_sha}</li> : null}
                      </ul>
                    </div>
                  ) : null}
                  <div>
                    <h3>Sources</h3>
                    <ul>
                      {liveAnswerSources.map((source) => (
                        <li key={source}>{source}</li>
                      ))}
                    </ul>
                  </div>
                  {liveAnswer.suggested_questions.length > 0 ? (
                    <div>
                      <h3>Related</h3>
                      <ul>
                        {liveAnswer.suggested_questions.map((question) => (
                          <li key={question}>{question}</li>
                        ))}
                      </ul>
                    </div>
                  ) : null}
                </div>
              </article>
            ) : null}
            <div className="ruStartMiniArtifacts" aria-label="Delivery control path">
              <span>Цель</span>
              <span>Контракт</span>
              <span>Proof</span>
              <span>Approval</span>
            </div>
          </section>
        </section>

        <section className="ruStartQuestionSection" id="questions" aria-labelledby="ru-start-questions-title">
          <div className="ruStartSectionHeader">
            <p className="ruStartKicker">Быстрые вопросы</p>
            <h2 id="ru-start-questions-title">Выберите prompt и задайте вопрос, когда готовы.</h2>
          </div>
          <div className="ruStartQuestionGrid">
            {ruStartQuestions.map((question) => (
              <button
                aria-pressed={question.id === activeQuestionId}
                aria-label={question.label}
                className={question.id === activeQuestionId ? 'ruStartQuestionCard active' : 'ruStartQuestionCard'}
                key={question.id}
                onClick={() => selectStaticQuestion(question.id)}
                type="button"
              >
                <span className="ruStartQuestionCue" aria-hidden="true" />
                <span className="ruStartQuestionText">{question.label}</span>
                <span className="ruStartQuestionArrow" aria-hidden="true">
                  +
                </span>
              </button>
            ))}
          </div>
        </section>

        <section className="ruStartAnswerSection" aria-labelledby="ru-start-answer-title">
          <div className="ruStartAnswerPanel">
            <div className="ruStartAnswerRail" aria-hidden="true" />
            <div className="ruStartAnswerContent">
              <p className="ruStartKicker">{activeAnswer.eyebrow}</p>
              <h2 id="ru-start-answer-title">{activeAnswer.title}</h2>
              {activeAnswer.body.map((paragraph) => (
                <p key={paragraph}>{paragraph}</p>
              ))}
            </div>
            <aside className="ruStartAnswerMeta" aria-label="Источники и следующие вопросы">
              <div>
                <h3>Sources</h3>
                <ul>
                  {activeAnswer.sources.map((source) => (
                    <li key={source}>{source}</li>
                  ))}
                </ul>
              </div>
              <div>
                <h3>Related</h3>
                <ul>
                  {activeAnswer.nextQuestions.map((question) => (
                    <li key={question}>{question}</li>
                  ))}
                </ul>
              </div>
            </aside>
          </div>
        </section>

        <section className="ruStartArtifactsSection" id="artifacts" aria-labelledby="ru-start-artifacts-title">
          <div className="ruStartSectionHeader">
            <p className="ruStartKicker">Рабочие артефакты</p>
            <h2 id="ru-start-artifacts-title">Контрольные поверхности, которые GoalRail формирует первыми.</h2>
          </div>
          <div className="ruStartArtifactGrid">
            {ruStartArtifacts.map((artifact) => (
              <article className="ruStartArtifactCard" key={artifact.title}>
                <p>{artifact.signal}</p>
                <h3>{artifact.title}</h3>
                <span>{artifact.body}</span>
              </article>
            ))}
          </div>
        </section>

        <section className="ruStartBoundaryCta" aria-labelledby="ru-start-boundary-title">
          <div className="ruStartBoundaryNote">
            <p className="ruStartKicker">Границы безопасности</p>
            <h2 id="ru-start-boundary-title">Public-safe boundaries.</h2>
            <ul>
              <li>Эта страница отвечает только по публичным материалам GoalRail.</li>
              <li>Она не может сканировать ваш репозиторий.</li>
              <li>Она не выполняет код.</li>
              <li>Не вставляйте секреты, приватный код или данные клиентов.</li>
            </ul>
          </div>
          <div className="ruStartCtaPanel">
            <p className="ruStartKicker">Pilot-first</p>
            <h2>Есть реальный workflow, где AI ускоряет команду, но контроль становится сложнее?</h2>
            <p>Запросите проверку fit для пилота.</p>
            <p>Лучший формат: одна команда, один репозиторий или workflow, один видимый loop от задачи до proof.</p>
            <div className="ruStartCtaActions">
              <a className="ruStartPrimaryAction" href="mailto:pilot@goalrail.dev?subject=GoalRail%20pilot%20fit%20check">
                Запросить проверку пилота
              </a>
              <a className="ruStartSecondaryAction" href="https://github.com/heurema/goalrail" rel="noreferrer" target="_blank">
                GitHub
              </a>
              <a className="ruStartSecondaryAction" href="#artifacts">
                Артефакты
              </a>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}

export default StartPageRu;

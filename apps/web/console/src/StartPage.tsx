import { FormEvent, useEffect, useMemo, useRef, useState } from 'react';

import { startAnswers, startArtifacts, startQuestions } from './startPageData';
import './StartPage.css';

const START_PAGE_TITLE = 'Goalrail - AI-assisted delivery without losing control';
const START_PAGE_DESCRIPTION =
  'Goalrail is a control layer for AI-assisted software delivery: from business goal to verified code change with contracts, proof, and human approval.';
const START_PAGE_OG_TITLE = 'Ask Goalrail about AI-assisted delivery';
const START_PAGE_OG_DESCRIPTION = 'From business goal to verified code change.';
const START_ASSISTANT_UNAVAILABLE =
  'The public Goalrail assistant is temporarily unavailable. Static overview and artifacts are still available.';

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

function StartPage() {
  const [activeQuestionId, setActiveQuestionId] = useState(startQuestions[0].id);
  const [liveQuestion, setLiveQuestion] = useState('');
  const [liveStatus, setLiveStatus] = useState<LiveAssistantStatus>('idle');
  const [liveAnswer, setLiveAnswer] = useState<LiveAssistantResponse | null>(null);
  const [liveError, setLiveError] = useState('');
  const askPanelRef = useRef<HTMLElement | null>(null);
  const liveQuestionInputRef = useRef<HTMLTextAreaElement | null>(null);
  const activeQuestion = useMemo(
    () => startQuestions.find((question) => question.id === activeQuestionId) ?? startQuestions[0],
    [activeQuestionId]
  );
  const activeAnswer = startAnswers[activeQuestion.answerId] ?? startAnswers[startQuestions[0].answerId];
  const liveAnswerSources = liveAnswer
    ? liveAnswer.sources.map((source) =>
        source.section ? `${source.title} / ${source.section}` : source.title || source.path
      )
    : [];

  useEffect(() => {
    document.title = START_PAGE_TITLE;
    upsertNamedMeta('description', START_PAGE_DESCRIPTION);
    upsertPropertyMeta('og:title', START_PAGE_OG_TITLE);
    upsertPropertyMeta('og:description', START_PAGE_OG_DESCRIPTION);
    upsertPropertyMeta('og:type', 'website');
  }, []);

  useEffect(() => {
    const input = liveQuestionInputRef.current;
    if (!input) {
      return;
    }

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
        throw new Error(payload.message || START_ASSISTANT_UNAVAILABLE);
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
      setLiveError(error instanceof Error ? error.message : START_ASSISTANT_UNAVAILABLE);
    }
  }

  function selectStaticQuestion(questionId: string) {
    const question = startQuestions.find((item) => item.id === questionId) ?? startQuestions[0];

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
    <main className="startPage">
      <div className="startRails" aria-hidden="true" />
      <div className="startShell">
        <header className="startTopbar">
          <div className="brand" aria-label="Goalrail global start">
            <span className="brandText">GOALRAIL</span>
          </div>
          <nav className="startTopLinks" aria-label="Start page links">
            <a href="#questions">Questions</a>
            <a href="#artifacts">Artifacts</a>
            <a href="https://github.com/heurema/goalrail" rel="noreferrer" target="_blank">
              GitHub
            </a>
          </nav>
        </header>

        <section className="startHero" aria-labelledby="start-title">
          <div className="startHeroCopy">
            <p className="kicker">Global start</p>
            <h1 id="start-title">Ask Goalrail about AI-assisted delivery.</h1>
            <p className="startSubtitle">From business goal to verified code change.</p>
            <p className="startBody">
              Goalrail is a control layer for teams using AI coding tools and trying not to lose intent, scope,
              checks, proof, and approval.
            </p>
          </div>

          <section className="startAskPanel" aria-label="Public assistant entry" ref={askPanelRef}>
            <div className="startPanelHeader">
              <span>Public Goalrail KB</span>
              <span className="startPanelStatus">
                {liveStatus === 'loading' ? 'asking' : liveStatus === 'answered' ? 'answered' : 'live'}
              </span>
            </div>
            <form className="startAskBox" onSubmit={handleLiveQuestionSubmit}>
              <textarea
                aria-label="Ask Goalrail"
                disabled={liveStatus === 'loading'}
                onChange={(event) => setLiveQuestion(event.target.value)}
                placeholder="Ask about repo readiness, contracts, proof, approval, or AI delivery drift..."
                ref={liveQuestionInputRef}
                rows={2}
                value={liveQuestion}
              />
              <button aria-label="Ask" disabled={!liveQuestion.trim() || liveStatus === 'loading'} type="submit">
                {liveStatus === 'loading' ? (
                  <span className="startAskSpinner" aria-hidden="true" />
                ) : (
                  <span aria-hidden="true">↑</span>
                )}
              </button>
            </form>
            <p className={liveStatus === 'error' ? 'startAskNote error' : 'startAskNote'}>
              {liveStatus === 'error'
                ? liveError
                : 'Answers use public Goalrail materials only. Static guide remains available below.'}
            </p>
            {liveAnswer ? (
              <article className="startLiveAnswer" aria-live="polite">
                <div className="startLiveAnswerHeader">
                  <p className="kicker">Public KB answer</p>
                  <h2>Source-grounded assistant response</h2>
                </div>
                <p>{liveAnswer.answer}</p>
                {liveAnswer.disclaimer ? <p className="startAnswerDisclaimer">{liveAnswer.disclaimer}</p> : null}
                <div className="startLiveMeta" aria-label="Live answer sources and freshness">
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
            <div className="startMiniArtifacts" aria-label="Delivery control path">
              <span>Goal</span>
              <span>Contract</span>
              <span>Proof</span>
              <span>Approval</span>
            </div>
          </section>
        </section>

        <section className="startQuestionSection" id="questions" aria-labelledby="start-questions-title">
          <div className="startSectionHeader">
            <p className="kicker">Quick questions</p>
            <h2 id="start-questions-title">Pick a prompt, then ask when ready.</h2>
          </div>
          <div className="startQuestionGrid">
            {startQuestions.map((question) => (
              <button
                aria-pressed={question.id === activeQuestionId}
                aria-label={question.label}
                className={question.id === activeQuestionId ? 'startQuestionCard active' : 'startQuestionCard'}
                key={question.id}
                onClick={() => selectStaticQuestion(question.id)}
                type="button"
              >
                <span className="startQuestionCue" aria-hidden="true" />
                <span className="startQuestionText">{question.label}</span>
                <span className="startQuestionArrow" aria-hidden="true">
                  +
                </span>
              </button>
            ))}
          </div>
        </section>

        <section className="startAnswerSection" aria-labelledby="start-answer-title">
          <div className="startAnswerPanel">
            <div className="startAnswerRail" aria-hidden="true" />
            <div className="startAnswerContent">
              <p className="kicker">{activeAnswer.eyebrow}</p>
              <h2 id="start-answer-title">{activeAnswer.title}</h2>
              {activeAnswer.body.map((paragraph) => (
                <p key={paragraph}>{paragraph}</p>
              ))}
            </div>
            <aside className="startAnswerMeta" aria-label="Answer source and follow-ups">
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

        <section className="startArtifactsSection" id="artifacts" aria-labelledby="start-artifacts-title">
          <div className="startSectionHeader">
            <p className="kicker">Working artifacts</p>
            <h2 id="start-artifacts-title">The control surfaces Goalrail is shaping first.</h2>
          </div>
          <div className="startArtifactGrid">
            {startArtifacts.map((artifact) => (
              <article className="startArtifactCard" key={artifact.title}>
                <p>{artifact.signal}</p>
                <h3>{artifact.title}</h3>
                <span>{artifact.body}</span>
              </article>
            ))}
          </div>
        </section>

        <section className="startBoundaryCta" aria-labelledby="start-boundary-title">
          <div className="startBoundaryNote">
            <p className="kicker">Safety boundaries</p>
            <h2 id="start-boundary-title">Public-safe boundaries.</h2>
            <ul>
              <li>This page answers from public Goalrail materials only.</li>
              <li>It cannot scan your repository.</li>
              <li>It does not execute code.</li>
              <li>Do not paste secrets, private code, or customer data.</li>
            </ul>
          </div>
          <div className="startCtaPanel">
            <p className="kicker">Pilot-first</p>
            <h2>Have a real workflow where AI is making your team faster but harder to control?</h2>
            <p>Request a pilot fit check.</p>
            <p>Best fit: one team, one repo or workflow, one visible task-to-proof loop.</p>
            <div className="startCtaActions">
              <a className="startPrimaryAction" href="mailto:hello@goalrail.dev?subject=Goalrail%20pilot%20fit%20check">
                Request a pilot fit check
              </a>
              <a className="startSecondaryAction" href="https://github.com/heurema/goalrail" rel="noreferrer" target="_blank">
                View GitHub
              </a>
              <a className="startSecondaryAction" href="#artifacts">
                View artifacts
              </a>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}

export default StartPage;

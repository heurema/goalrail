import { useEffect, useMemo, useState } from 'react';

import { startAnswers, startArtifacts, startQuestions } from './startPageData';

const START_PAGE_TITLE = 'Goalrail - AI-assisted delivery without losing control';
const START_PAGE_DESCRIPTION =
  'Goalrail is a control layer for AI-assisted software delivery: from business goal to verified code change with contracts, proof, and human approval.';
const START_PAGE_OG_TITLE = 'Ask Goalrail about AI-assisted delivery';
const START_PAGE_OG_DESCRIPTION = 'From business goal to verified code change.';

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
  const activeQuestion = useMemo(
    () => startQuestions.find((question) => question.id === activeQuestionId) ?? startQuestions[0],
    [activeQuestionId]
  );
  const activeAnswer = startAnswers[activeQuestion.answerId] ?? startAnswers[startQuestions[0].answerId];

  useEffect(() => {
    document.title = START_PAGE_TITLE;
    upsertNamedMeta('description', START_PAGE_DESCRIPTION);
    upsertPropertyMeta('og:title', START_PAGE_OG_TITLE);
    upsertPropertyMeta('og:description', START_PAGE_OG_DESCRIPTION);
    upsertPropertyMeta('og:type', 'website');
  }, []);

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

          <aside className="startAskPanel" aria-label="Static assistant preview">
            <div className="startPanelHeader">
              <span className="startPanelDot" aria-hidden="true" />
              <span>Guided entry</span>
              <span className="startPanelStatus">static</span>
            </div>
            <div className="startAskBox">
              <textarea
                aria-label="Ask Goalrail"
                disabled
                placeholder="Ask about repo readiness, contracts, proof, approval, or AI delivery drift..."
                rows={2}
              />
              <button disabled type="button">
                Ask
              </button>
            </div>
            <p className="startAskNote">Live assistant is coming next. For now, use the guided questions below.</p>
            <div className="startMiniArtifacts" aria-label="Delivery control path">
              <span>Goal</span>
              <span>Contract</span>
              <span>Proof</span>
              <span>Approval</span>
            </div>
          </aside>
        </section>

        <section className="startQuestionSection" id="questions" aria-labelledby="start-questions-title">
          <div className="startSectionHeader">
            <p className="kicker">Quick questions</p>
            <h2 id="start-questions-title">Use the static guide while the live assistant is not connected.</h2>
          </div>
          <div className="startQuestionGrid">
            {startQuestions.map((question) => (
              <button
                aria-pressed={question.id === activeQuestionId}
                className={question.id === activeQuestionId ? 'startQuestionCard active' : 'startQuestionCard'}
                key={question.id}
                onClick={() => setActiveQuestionId(question.id)}
                type="button"
              >
                <span>{question.label}</span>
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
            <aside className="startAnswerMeta" aria-label="Static answer source and follow-ups">
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
            <h2 id="start-boundary-title">Static and public-safe.</h2>
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

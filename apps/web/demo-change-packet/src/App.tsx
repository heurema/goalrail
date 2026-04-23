import type { ReactNode } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';

import './App.css';

type StepIndex = 0 | 1 | 2 | 3 | 4 | 5 | 6;

interface Stage {
  id: string;
  name: string;
}

interface Clarification {
  id: string;
  qRefs: string[];
  ref: string;
  q: string;
  opts: string[];
  answer: number;
}

interface TaskSlice {
  n: string;
  title: string;
  surface: string;
  criteria: string[];
  risk: string;
  proof: string;
}

interface Receipt {
  ts: string;
  verdict: 'pass' | 'block';
  label: 'PASS' | 'BLOCKED';
  code: string;
  mapTo: number;
  renderText: () => ReactNode;
}

interface Criterion {
  id: number;
  name: string;
  evLabel: string;
}

interface LedgerEvent {
  ts: string;
  kind: string;
  note: string;
  tag: string;
  cls: 'amber' | 'mauve' | 'pass' | 'block';
}

interface InspectorRow {
  q: string;
  term: string;
  prov: ReactNode;
}

const STAGES: Stage[] = [
  { id: 'request', name: 'Raw request' },
  { id: 'clarify', name: 'Clarification' },
  { id: 'contract', name: 'Working contract' },
  { id: 'tasks', name: 'Bounded tasks' },
  { id: 'execution', name: 'Execution evidence' },
  { id: 'verification', name: 'Verification' },
  { id: 'proof', name: 'Proof' },
  { id: 'decision', name: 'Decision' },
];

const STEP_TO_ACTIVE = [0, 1, 2, 3, 4, 5, 7] as const;
const LIFECYCLE = ['Draft', 'Clarified', 'Contracted', 'Executed', 'Verified', 'Accepted'] as const;
const STEP_TO_LIFE = [0, 0, 1, 2, 3, 4, 5] as const;

const CLARIFICATIONS: Clarification[] = [
  {
    id: 'review-scope',
    qRefs: ['Q1'],
    ref: 'manual review',
    q: 'What triggers the manual review step?',
    opts: ['Every trial request', 'Only requests above $10k ACV', 'Only requests from regulated regions'],
    answer: 1,
  },
  {
    id: 'owner-rule',
    qRefs: ['Q2', 'Q3'],
    ref: 'reviewer · owner',
    q: 'Who can be assigned as owner?',
    opts: ['Any workspace admin', 'Users with trial_reviewer role', 'Either admin or reviewer role'],
    answer: 1,
  },
  {
    id: 'reason-rule',
    qRefs: ['Q4'],
    ref: 'decision reason',
    q: 'How is decision reason captured?',
    opts: ['Free-text, optional', 'Free-text, required, min 20 chars', 'Fixed dropdown of reason codes'],
    answer: 1,
  },
  {
    id: 'status-model',
    qRefs: ['Q5'],
    ref: 'new status',
    q: 'What is the new status model?',
    opts: [
      'Add pending_review between requested and approved',
      'Add separate flag, keep current statuses',
      'Replace approved with manual_review',
    ],
    answer: 0,
  },
  {
    id: 'audit-req',
    qRefs: ['Q6'],
    ref: 'audit log',
    q: 'What does the audit log capture?',
    opts: ['Actor + timestamp', 'Actor + timestamp + reason + previous status', 'Full diff of the request record'],
    answer: 1,
  },
];

const TASKS: TaskSlice[] = [
  {
    n: 'T-01',
    title: 'Status model',
    surface: 'schema · db · request.status enum',
    criteria: ['Manual review visible', 'Direct approval blocked'],
    risk: 'medium · migration',
    proof: 'transition matrix receipts',
  },
  {
    n: 'T-02',
    title: 'API validation',
    surface: 'POST /requests/:id/decision',
    criteria: ['Owner required', 'Reason required'],
    risk: 'low',
    proof: 'rejection receipts with error codes',
  },
  {
    n: 'T-03',
    title: 'Frontend controls',
    surface: 'views · RequestDetail, ReviewDialog',
    criteria: ['Manual review visible', 'Owner required'],
    risk: 'low',
    proof: 'component state receipts',
  },
  {
    n: 'T-04',
    title: 'Smoke / proof / docs',
    surface: 'spec · audit + runbook',
    criteria: ['Audit captures decision context'],
    risk: 'low',
    proof: 'audit event schema + sample payload',
  },
];

const RECEIPTS: Receipt[] = [
  {
    ts: 'T+0.00s',
    verdict: 'block',
    label: 'BLOCKED',
    code: 'direct_approval_forbidden',
    mapTo: 0,
    renderText: () => (
      <>
        Transition <span className="receipt-code">requested → approved</span> attempted without review
      </>
    ),
  },
  {
    ts: 'T+0.12s',
    verdict: 'pass',
    label: 'PASS',
    code: 'status.manual_review',
    mapTo: 1,
    renderText: () => (
      <>
        Transition <span className="receipt-code">requested → manual_review</span> accepted
      </>
    ),
  },
  {
    ts: 'T+0.28s',
    verdict: 'block',
    label: 'BLOCKED',
    code: 'owner_required',
    mapTo: 2,
    renderText: () => (
      <>
        Decision submitted without <span className="receipt-code">owner</span> field
      </>
    ),
  },
  {
    ts: 'T+0.41s',
    verdict: 'block',
    label: 'BLOCKED',
    code: 'reason_required',
    mapTo: 3,
    renderText: () => (
      <>
        Decision submitted without <span className="receipt-code">reason</span> field
      </>
    ),
  },
  {
    ts: 'T+0.58s',
    verdict: 'pass',
    label: 'PASS',
    code: 'audit.decision.emitted',
    mapTo: 4,
    renderText: () => (
      <>
        Audit event emitted with <span className="receipt-code">{'{actor, prev_status, reason, ts}'}</span>
      </>
    ),
  },
];

const CRITERIA: Criterion[] = [
  { id: 0, name: 'Direct approval blocked', evLabel: 'receipt · direct_approval_forbidden' },
  { id: 1, name: 'Manual review visible', evLabel: 'receipt · status.manual_review' },
  { id: 2, name: 'Owner required', evLabel: 'receipt · owner_required' },
  { id: 3, name: 'Reason required', evLabel: 'receipt · reason_required' },
  { id: 4, name: 'Audit captures decision context', evLabel: 'receipt · audit.decision.emitted' },
];

const BASE_EVENTS: LedgerEvent[] = [
  {
    ts: '09:42:08',
    kind: 'request.received',
    note: 'inbound change request · cto',
    tag: 'INBOUND',
    cls: 'mauve',
  },
  {
    ts: '09:42:09',
    kind: 'ambiguity.detected',
    note: '6 undefined terms flagged in request body',
    tag: 'AMBIG',
    cls: 'amber',
  },
  {
    ts: '09:42:10',
    kind: 'clarification.queued',
    note: '5 bounded clarifications prepared (q1–q6 → 5 decisions)',
    tag: 'QUEUED',
    cls: 'amber',
  },
];

const STEP_EVENTS: Partial<Record<StepIndex, LedgerEvent[]>> = {
  1: [
    { ts: '09:43:02', kind: 'clarification.answered', note: 'review-scope → $10k ACV', tag: 'ANS', cls: 'mauve' },
    {
      ts: '09:43:18',
      kind: 'clarification.answered',
      note: 'owner-rule → trial_reviewer role',
      tag: 'ANS',
      cls: 'mauve',
    },
    {
      ts: '09:43:34',
      kind: 'clarification.answered',
      note: 'reason-rule → required · min 20ch',
      tag: 'ANS',
      cls: 'mauve',
    },
    { ts: '09:43:48', kind: 'clarification.answered', note: 'status-model → pending_review', tag: 'ANS', cls: 'mauve' },
    {
      ts: '09:44:01',
      kind: 'clarification.answered',
      note: 'audit-req → actor+ts+reason+prev',
      tag: 'ANS',
      cls: 'mauve',
    },
  ],
  2: [
    { ts: '09:44:22', kind: 'contract.updated', note: 'Goal, in/out scope, criteria assembled', tag: 'CONTRACT', cls: 'mauve' },
    { ts: '09:44:24', kind: 'contract.updated', note: 'Out of scope: bulk approval, SSO gating', tag: 'CONTRACT', cls: 'mauve' },
  ],
  3: [
    { ts: '09:45:01', kind: 'contract.approved', note: 'Working contract locked', tag: 'OK', cls: 'pass' },
    { ts: '09:45:02', kind: 'task.created', note: 'T-01 · Status model', tag: 'TASK', cls: 'mauve' },
    { ts: '09:45:02', kind: 'task.created', note: 'T-02 · API validation', tag: 'TASK', cls: 'mauve' },
    { ts: '09:45:02', kind: 'task.created', note: 'T-03 · Frontend controls', tag: 'TASK', cls: 'mauve' },
    { ts: '09:45:02', kind: 'task.created', note: 'T-04 · Smoke · proof · docs', tag: 'TASK', cls: 'mauve' },
  ],
  4: [
    { ts: '09:45:47', kind: 'evidence.attached', note: 'direct approval transition blocked', tag: 'BLOCK', cls: 'block' },
    { ts: '09:45:59', kind: 'evidence.attached', note: 'manual_review transition accepted', tag: 'PASS', cls: 'pass' },
    { ts: '09:46:15', kind: 'evidence.attached', note: 'owner_required error returned', tag: 'BLOCK', cls: 'block' },
    { ts: '09:46:28', kind: 'evidence.attached', note: 'reason_required error returned', tag: 'BLOCK', cls: 'block' },
    { ts: '09:46:45', kind: 'evidence.attached', note: 'audit event emitted · decision context', tag: 'PASS', cls: 'pass' },
  ],
  5: [
    { ts: '09:47:02', kind: 'proof.verdict', note: 'criterion 1 · direct approval blocked', tag: 'PASS', cls: 'pass' },
    { ts: '09:47:03', kind: 'proof.verdict', note: 'criterion 2 · manual review visible', tag: 'PASS', cls: 'pass' },
    { ts: '09:47:04', kind: 'proof.verdict', note: 'criterion 3 · owner required', tag: 'PASS', cls: 'pass' },
    { ts: '09:47:04', kind: 'proof.verdict', note: 'criterion 4 · reason required', tag: 'PASS', cls: 'pass' },
    { ts: '09:47:05', kind: 'proof.verdict', note: 'criterion 5 · audit captures context', tag: 'PASS', cls: 'pass' },
  ],
  6: [{ ts: '09:47:30', kind: 'decision.unlocked', note: 'all criteria passed · 0 unresolved', tag: 'READY', cls: 'pass' }],
};

const CONTROL_MAP: Record<StepIndex, Array<{ label: string; action: 'back' | 'reset' }>> = {
  0: [{ label: 'Reset packet', action: 'reset' }],
  1: [
    { label: 'Back', action: 'back' },
    { label: 'Reset', action: 'reset' },
  ],
  2: [
    { label: 'Back', action: 'back' },
    { label: 'Reset', action: 'reset' },
  ],
  3: [
    { label: 'Back', action: 'back' },
    { label: 'Reset', action: 'reset' },
  ],
  4: [
    { label: 'Back', action: 'back' },
    { label: 'Reset', action: 'reset' },
  ],
  5: [
    { label: 'Back', action: 'back' },
    { label: 'Reset', action: 'reset' },
  ],
  6: [{ label: 'Replay state', action: 'reset' }],
};

const PRIMARY_CTA: Record<
  StepIndex,
  { label: string; title: string; sub: string; tone: 'ready' | 'pass'; disabled?: boolean }
> = {
  0: {
    label: 'next · clarify',
    title: 'Start clarification',
    sub: 'Convert 6 flagged terms into 5 bounded decisions. Intent will resolve from 42% toward 96%.',
    tone: 'ready',
  },
  1: {
    label: 'next · contract',
    title: 'Assemble working contract',
    sub: 'Fold resolved clarifications into goal, in/out scope, acceptance, and proof expectations.',
    tone: 'ready',
  },
  2: {
    label: 'next · approve',
    title: 'Approve contract v3',
    sub: 'Freeze the contract and derive bounded task slices. Each slice will declare its own proof obligation.',
    tone: 'ready',
  },
  3: {
    label: 'next · replay',
    title: 'Run execution replay',
    sub: 'Replay the deterministic fixture run. Five receipts will be emitted and bound to contract clauses.',
    tone: 'ready',
  },
  4: {
    label: 'next · verify',
    title: 'Verify against criteria',
    sub: 'Pair each acceptance criterion with the single receipt that proves it. No verdict without evidence.',
    tone: 'ready',
  },
  5: {
    label: 'next · decide',
    title: 'Open decision gate',
    sub: 'All criteria passed. The gate is unlocked for accept, rework, or block — each action archives the packet.',
    tone: 'pass',
  },
  6: {
    label: 'packet complete',
    title: 'Change packet accepted',
    sub: 'Proof archive hash pinned · contract v3 frozen · decision recorded. Replay state to run the demo again.',
    tone: 'pass',
    disabled: true,
  },
};

const INSPECTOR_ROWS: InspectorRow[] = [
  {
    q: 'Q1',
    term: 'manual review',
    prov: (
      <>
        a <b>bounded state</b> between requested and approved — trigger rule unknown
      </>
    ),
  },
  {
    q: 'Q2',
    term: 'reviewer',
    prov: (
      <>
        a <b>role</b> that can transition the request — exact role unknown
      </>
    ),
  },
  {
    q: 'Q3',
    term: 'owner',
    prov: (
      <>
        a <b>field</b> recorded on the decision — source of truth unclear
      </>
    ),
  },
  {
    q: 'Q4',
    term: 'decision reason',
    prov: (
      <>
        a <b>rationale</b> attached to the decision — required vs optional undecided
      </>
    ),
  },
  {
    q: 'Q5',
    term: 'new status',
    prov: (
      <>
        an <b>enum edge</b> on request.status — name and position unknown
      </>
    ),
  },
  {
    q: 'Q6',
    term: 'audit log',
    prov: (
      <>
        an <b>append-only record</b> of the decision — schema undefined
      </>
    ),
  },
];

const ALL_CLARIFICATION_IDS = CLARIFICATIONS.map(({ id }) => id);
const CLARIFICATION_ORDER = ['review-scope', 'owner-rule', 'reason-rule', 'status-model', 'audit-req'] as const;

function cx(...tokens: Array<string | false | null | undefined>) {
  return tokens.filter(Boolean).join(' ');
}

function buildEvents(step: StepIndex) {
  const events = [...BASE_EVENTS];

  for (let index = 1; index <= step; index += 1) {
    const stepEvents = STEP_EVENTS[index as StepIndex] ?? [];
    events.push(...stepEvents);
  }

  return events;
}

function getReadiness(step: StepIndex, answeredCount: number, visibleReceipts: number, matrixFilled: number) {
  const intentPercent =
    step === 0
      ? 42
      : step === 1
        ? 42 + Math.round((answeredCount / CLARIFICATIONS.length) * (96 - 42))
        : step === 2
          ? 96
          : 100;

  const execPercent = [0, 22, 55, 74, 92, 100, 100][step];
  const execLabel = ['Not ready', 'Gathering', 'Contract pending', 'Tasks scoped', 'Evidence flowing', 'Complete', 'Complete'][step];

  const proofPercent =
    step === 0
      ? 0
      : step === 1
        ? 12
        : step === 2
          ? 35
          : step === 3
            ? 48
            : step === 4
              ? Math.round((visibleReceipts / RECEIPTS.length) * 72)
              : step === 5
                ? Math.round(72 + (matrixFilled / CRITERIA.length) * (95 - 72))
                : 100;

  const proofLabel =
    step === 0
      ? 'Not ready'
      : step === 1
        ? 'Pending'
        : step === 2
          ? 'Criteria drafted'
          : step === 3
            ? 'Awaiting evidence'
            : step === 4
              ? visibleReceipts > 0
                ? `${visibleReceipts} of 5`
                : 'Partial'
              : step === 5
                ? matrixFilled > 0
                  ? `${matrixFilled} of 5 verified`
                  : 'Partial'
                : 'Complete';

  return {
    intentPercent,
    intentLabel: `${intentPercent}%`,
    execPercent,
    execLabel,
    proofPercent,
    proofLabel,
  };
}

function getInitialProgress(step: StepIndex) {
  if (step === 0) {
    return { answeredIds: [] as string[], visibleReceipts: 0, matrixFilled: 0 };
  }

  if (step === 1) {
    return { answeredIds: [] as string[], visibleReceipts: 0, matrixFilled: 0 };
  }

  if (step === 2 || step === 3) {
    return { answeredIds: [...ALL_CLARIFICATION_IDS], visibleReceipts: 0, matrixFilled: 0 };
  }

  if (step === 4) {
    return { answeredIds: [...ALL_CLARIFICATION_IDS], visibleReceipts: 0, matrixFilled: 0 };
  }

  if (step === 5) {
    return { answeredIds: [...ALL_CLARIFICATION_IDS], visibleReceipts: RECEIPTS.length, matrixFilled: 0 };
  }

  return { answeredIds: [...ALL_CLARIFICATION_IDS], visibleReceipts: RECEIPTS.length, matrixFilled: CRITERIA.length };
}

export default function App() {
  const [step, setStep] = useState<StepIndex>(0);
  const [answeredIds, setAnsweredIds] = useState<string[]>([]);
  const [visibleReceipts, setVisibleReceipts] = useState(0);
  const [matrixFilled, setMatrixFilled] = useState(0);
  const ledgerBodyRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const progress = getInitialProgress(step);
    setAnsweredIds(progress.answeredIds);
    setVisibleReceipts(progress.visibleReceipts);
    setMatrixFilled(progress.matrixFilled);

    const timeouts: number[] = [];

    if (step === 1) {
      CLARIFICATION_ORDER.forEach((id, index) => {
        timeouts.push(
          window.setTimeout(() => {
            setAnsweredIds((current) => (current.includes(id) ? current : [...current, id]));
          }, 350 + index * 280),
        );
      });
    }

    if (step === 4) {
      for (let index = 1; index <= RECEIPTS.length; index += 1) {
        timeouts.push(
          window.setTimeout(() => {
            setVisibleReceipts(index);
          }, 260 + index * 360),
        );
      }
    }

    if (step === 5) {
      for (let index = 1; index <= CRITERIA.length; index += 1) {
        timeouts.push(
          window.setTimeout(() => {
            setMatrixFilled(index);
          }, 220 + index * 280),
        );
      }
    }

    return () => {
      timeouts.forEach((timeoutId) => window.clearTimeout(timeoutId));
    };
  }, [step]);

  const answeredSet = useMemo(() => new Set(answeredIds), [answeredIds]);
  const activeStageIndex = STEP_TO_ACTIVE[step];
  const lifecycle = LIFECYCLE[STEP_TO_LIFE[step]];
  const events = useMemo(() => buildEvents(step), [step]);
  const readiness = getReadiness(step, answeredIds.length, visibleReceipts, matrixFilled);

  useEffect(() => {
    const ledgerBody = ledgerBodyRef.current;

    if (ledgerBody) {
      ledgerBody.scrollTop = ledgerBody.scrollHeight;
    }
  }, [events.length]);

  const primaryCta = PRIMARY_CTA[step];
  const spineSummary = getSpineSummary(step, answeredIds.length, visibleReceipts, matrixFilled);
  const contractButtonsVisible = step === 2;
  const lastEventIndex = events.length - 1;

  const goNext = () => {
    setStep((current) => (current < 6 ? ((current + 1) as StepIndex) : current));
  };

  const goBack = () => {
    setStep((current) => (current > 0 ? ((current - 1) as StepIndex) : current));
  };

  const reset = () => {
    setStep(0);
  };

  const controlAction = (action: 'back' | 'reset') => {
    if (action === 'back') {
      goBack();
      return;
    }

    reset();
  };

  return (
    <div className="app-shell" data-step={step}>
      <div className="app">
        <header className="topbar">
          <div className="brand">
            <div className="mark" aria-hidden="true">
              <span />
            </div>
            <div className="name">
              Goalrail <span className="dot">·</span> <span className="brand-muted">ops</span>
            </div>
          </div>

          <div className="meters">
            <div className="meter amber" id="meter-intent">
              <div className="row">
                <div className="label">Intent</div>
                <div className="val">{readiness.intentLabel}</div>
              </div>
              <div className="bar">
                <i style={{ width: `${readiness.intentPercent}%` }} />
              </div>
            </div>

            <div className="meter mauve" id="meter-exec">
              <div className="row">
                <div className="label">Execution</div>
                <div className="val">{readiness.execLabel}</div>
              </div>
              <div className="bar">
                <i style={{ width: `${readiness.execPercent}%` }} />
              </div>
            </div>

            <div className="meter pass" id="meter-proof">
              <div className="row">
                <div className="label">Proof</div>
                <div className="val">{readiness.proofLabel}</div>
              </div>
              <div className="bar">
                <i style={{ width: `${readiness.proofPercent}%` }} />
              </div>
            </div>

            <div className="state-chip">
              <span className="k">state</span>
              <span className="v">{lifecycle.toLowerCase()}</span>
              <span className="sep">·</span>
              <span className="derive">at {STAGES[activeStageIndex].name.toLowerCase()}</span>
            </div>
          </div>
        </header>

        <aside className="rail">
          <div className="group-label">Workspace</div>
          <div className="item active g-dot">
            <i className="g" aria-hidden="true" />
            Change packets <span className="count">1</span>
          </div>
          <div className="item g-line">
            <i className="g" aria-hidden="true" />
            Replay <span className="count">—</span>
          </div>
          <div className="item g-dot">
            <i className="g" aria-hidden="true" />
            Proof <span className="count">5</span>
          </div>
          <div className="item g-line">
            <i className="g" aria-hidden="true" />
            Demo case <span className="count">1</span>
          </div>

          <div className="group-2">
            <div className="group-label reference-label">Reference</div>
            <div className="item g-line">
              <i className="g" aria-hidden="true" />
              Contracts
            </div>
            <div className="item g-line">
              <i className="g" aria-hidden="true" />
              Evidence store
            </div>
            <div className="item g-line">
              <i className="g" aria-hidden="true" />
              Event ledger
            </div>
          </div>

          <div className="case">
            <div className="k">Demo case</div>
            <div className="v">CP-0147 · trial-review</div>
            <div className="sub">opened · cto · 14m ago</div>
          </div>
        </aside>

        <main className="canvas">
          <section className="spine">
            <div className="spine-head">
              <div className="t">Change spine · cp-0147</div>
              <div className="id">{lifecycle.toLowerCase()}</div>
            </div>

            <div className="body">
              {STAGES.map((stage, index) => {
                const cls =
                  index < activeStageIndex
                    ? 'done'
                    : index === activeStageIndex
                      ? 'active'
                      : index === 0 && step === 0
                        ? 'ambig'
                        : '';

                const stageMeta = {
                  0: step === 0 ? '6 terms flagged' : '',
                  1: step === 1 ? `${answeredIds.length}/5 answered` : step > 1 ? '5/5 answered' : '',
                  2: step >= 2 ? (step === 2 ? 'drafted' : 'approved') : '',
                  3: step >= 3 ? '4 slices' : '',
                  4: step >= 4 ? `${visibleReceipts}/5 receipts` : '',
                  5: step >= 5 ? `${matrixFilled}/5 verified` : '',
                  6: step >= 5 ? `${matrixFilled}/5 criteria` : '',
                  7: step >= 6 ? 'unlocked' : '',
                }[index] as string;

                return (
                  <div key={stage.id} className={cx('stage', cls)}>
                    <div className="node" />
                    <div className="connector" />
                    <div className="name">{stage.name}</div>
                    <div className="meta">{stageMeta}</div>
                  </div>
                );
              })}
            </div>

            <div className="active-summary">
              <span className="marker">↓ active</span>
              <span className="stage-name">{spineSummary.name}</span>
              <span className="facts">
                {spineSummary.facts.map((fact) => (
                  <span key={fact.html} className={cx('f', fact.cls)} dangerouslySetInnerHTML={{ __html: fact.html }} />
                ))}
              </span>
            </div>
          </section>

          <section className="object">
            {step === 0 ? (
              <>
                <div className="obj-head">
                  <div className="t">Raw request · inbound</div>
                  <div className="tags">
                    <span className="tag amber">6 ambiguities</span>
                  </div>
                </div>

                <div className="obj-body">
                  <div className="request-meta">
                    <span className="k">from</span>
                    <span className="v">cto@goalrail</span>
                    <span className="k request-meta-gap">received</span>
                    <span className="v">09:42:08 UTC</span>
                    <span className="k request-meta-gap">channel</span>
                    <span className="v">email → intake</span>
                  </div>

                  <div className="request">
                    Before a trial request can be approved, we need a <span className="amb">manual review<span className="qref">Q1</span></span>{' '}
                    step. The <span className="amb">reviewer<span className="qref">Q2</span></span> must assign an{' '}
                    <span className="amb">owner<span className="qref">Q3</span></span> and provide a{' '}
                    <span className="amb">decision reason<span className="qref">Q4</span></span>. The dashboard should reflect the{' '}
                    <span className="amb">new status<span className="qref">Q5</span></span>, and the{' '}
                    <span className="amb">audit log<span className="qref">Q6</span></span> should show who made the decision.
                  </div>

                  <div className="amb-inspector">
                    <div className="ai-head">
                      <div className="t">Ambiguity inspector</div>
                      <div className="c">6 terms · provisional readings · awaiting clarification</div>
                    </div>

                    {INSPECTOR_ROWS.map((row) => (
                      <div key={row.q} className="ai-row">
                        <span className="q">{row.q}</span>
                        <span>
                          <span className="term">{row.term}</span>
                        </span>
                        <span className="prov">{row.prov}</span>
                        <span className="status">flagged</span>
                      </div>
                    ))}
                  </div>
                </div>
              </>
            ) : null}

            {step === 1 ? (
              <>
                <div className="obj-head">
                  <div className="t">Clarification cards · 5 of 5</div>
                  <div className="tags">
                    <span className="tag mauve">bounded</span>
                    <span className="tag">{answeredIds.length}/5 resolved</span>
                  </div>
                </div>

                <div className="obj-body">
                  <div className="grid-cards">
                    {CLARIFICATIONS.map((clarification) => {
                      const resolved = answeredSet.has(clarification.id);

                      return (
                        <div key={clarification.id} className={cx('clar-card', resolved && 'answered')}>
                          <div className="ref">{clarification.ref}</div>
                          <div className="q">{clarification.q}</div>
                          <div className="opts">
                            {clarification.opts.map((option, optionIndex) => (
                              <div key={option} className={cx('opt', resolved && optionIndex === clarification.answer && 'selected')}>
                                <span className="radio" />
                                <span>{option}</span>
                              </div>
                            ))}
                          </div>
                          <div className="bottom">
                            <span>{resolved ? 'Answer pinned to contract' : 'Unresolved'}</span>
                            <span>{resolved ? '✓' : '—'}</span>
                          </div>
                        </div>
                      );
                    })}
                  </div>

                  <div className="object-note">
                    Each answer pins an ambiguous term to a concrete contract clause. Readiness and the change packet on the
                    right update as cards resolve.
                  </div>
                </div>
              </>
            ) : null}

            {step === 2 ? (
              <>
                <div className="obj-head">
                  <div className="t">Working contract · draft v3</div>
                  <div className="tags">
                    <span className="tag mauve">pending approval</span>
                    <span className="tag">derived from 5 clarifications</span>
                  </div>
                </div>

                <div className="obj-body">
                  <div className="contract">
                    <div className="sec full">
                      <h4>Goal</h4>
                      <div className="goal-copy">
                        Introduce a bounded <b>manual review</b> state for trial requests above <b>$10k ACV</b>, enforced at API
                        and surfaced in UI, with auditable decisions including reason and prior status.
                      </div>
                    </div>

                    <div className="sec">
                      <h4>In scope</h4>
                      <ul>
                        <li>
                          Add <code className="mono-code bright">pending_review</code> status between{' '}
                          <code className="mono-code muted-tone">requested</code> and{' '}
                          <code className="mono-code muted-tone">approved</code>
                        </li>
                        <li>
                          API rejects decision without owner (<code className="mono-code muted-tone">trial_reviewer</code>) and
                          reason (≥20ch)
                        </li>
                        <li>Dashboard exposes pending review queue + decision dialog</li>
                        <li>Audit event captures actor, timestamp, reason, previous status</li>
                      </ul>
                    </div>

                    <div className="sec out">
                      <h4>Out of scope</h4>
                      <ul>
                        <li>
                          <s>Bulk approval across multiple requests</s>
                        </li>
                        <li>
                          <s>SSO / IdP gating of reviewer role</s>
                        </li>
                        <li>
                          <s>Retroactive audit for already-approved trials</s>
                        </li>
                        <li>
                          <s>Email / Slack notifications to reviewer</s>
                        </li>
                      </ul>
                    </div>

                    <div className="sec">
                      <h4>Acceptance criteria</h4>
                      <ul>
                        <li>
                          Direct <code className="mono-code muted-tone">requested → approved</code> transition is rejected
                        </li>
                        <li>Manual review state is visible in dashboard and API</li>
                        <li>
                          Owner field is required, bound to <code className="mono-code muted-tone">trial_reviewer</code> role
                        </li>
                        <li>Reason field is required, minimum 20 characters</li>
                        <li>Audit event records full decision context</li>
                      </ul>
                    </div>

                    <div className="sec">
                      <h4>Proof expectations</h4>
                      <ul>
                        <li>Deterministic transition receipts for each enum edge</li>
                        <li>API rejection receipts with stable error codes</li>
                        <li>Sample audit payload conforming to emitted schema</li>
                      </ul>
                    </div>
                  </div>

                  <div className="contract-foot">
                    <div className="note">
                      Approving will freeze the contract and emit <span className="contract-note-emphasis">contract.approved</span>.
                    </div>
                    {contractButtonsVisible ? (
                      <div className="inline-controls">
                        <button type="button" className="inline-ctrl" onClick={goBack}>
                          Request change
                        </button>
                        <button type="button" className="inline-ctrl approve" onClick={goNext}>
                          Approve contract ▸
                        </button>
                      </div>
                    ) : null}
                  </div>
                </div>
              </>
            ) : null}

            {step === 3 ? (
              <>
                <div className="obj-head">
                  <div className="t">Task slices · derived from contract</div>
                  <div className="tags">
                    <span className="tag mauve">4 bounded</span>
                    <span className="tag">linked acceptance</span>
                  </div>
                </div>

                <div className="obj-body">
                  <div className="tasks">
                    {TASKS.map((task) => (
                      <div key={task.n} className="task">
                        <div className="row1">
                          <div>
                            <div className="num">{task.n}</div>
                            <div className="title">{task.title}</div>
                            <div className="surface">{task.surface}</div>
                          </div>
                          <span className="tag mauve task-tag">scoped</span>
                        </div>

                        <div className="grid-kv">
                          <div className="k">Criteria</div>
                          <div className="v">{task.criteria.join(' · ')}</div>
                          <div className="k">Risk</div>
                          <div className="v">{task.risk}</div>
                          <div className="k">Proof</div>
                          <div className="v">{task.proof}</div>
                        </div>
                      </div>
                    ))}
                  </div>

                  <div className="object-note">
                    Slices are derived, not invented. Each maps to one or more acceptance criteria and declares its own proof
                    obligation.
                  </div>
                </div>
              </>
            ) : null}

            {step === 4 ? (
              <>
                <div className="obj-head">
                  <div className="t">Execution replay · deterministic receipts</div>
                  <div className="tags">
                    <span className="tag mauve">{visibleReceipts}/5 receipts</span>
                    <span className="tag">simulated</span>
                  </div>
                </div>

                <div className="obj-body">
                  <div className="sim-label">
                    <span className="pulse" />
                    <span>Simulated deterministic demo run · no live runtime · receipts are fixtures derived from contract</span>
                  </div>

                  <div className="receipts">
                    {RECEIPTS.slice(0, visibleReceipts).map((receipt) => (
                      <div key={receipt.code} className="receipt">
                        <span className="ts">{receipt.ts}</span>
                        <span className="txt">
                          {receipt.renderText()} <span className="receipt-meta">· {receipt.code}</span>
                        </span>
                        <span className={cx('verdict', receipt.verdict)}>{receipt.label}</span>
                      </div>
                    ))}
                  </div>

                  {visibleReceipts < RECEIPTS.length ? (
                    <div className="streaming">streaming…</div>
                  ) : (
                    <div className="object-note">
                      All five receipts captured. Verification pass runs them against acceptance criteria in the next step.
                    </div>
                  )}
                </div>
              </>
            ) : null}

            {step === 5 ? (
              <>
                <div className="obj-head">
                  <div className="t">Proof matrix · acceptance vs evidence</div>
                  <div className="tags">
                    <span className="tag pass">{matrixFilled}/5 verified</span>
                    <span className="tag">bound to contract v3</span>
                  </div>
                </div>

                <div className="obj-body">
                  <table className="matrix">
                    <thead>
                      <tr>
                        <th className="w-40">Acceptance criterion</th>
                        <th className="w-40">Evidence</th>
                        <th className="w-20">Verdict</th>
                      </tr>
                    </thead>
                    <tbody>
                      {CRITERIA.map((criterion, index) => {
                        const filled = index < matrixFilled;

                        return (
                          <tr key={criterion.id} className={filled ? '' : 'pending'}>
                            <td>
                              <span className={cx('check', !filled && 'pending-check')} />
                              {criterion.name}
                            </td>
                            <td>
                              <span className={cx('ev', !filled && 'pending')}>{filled ? criterion.evLabel : 'awaiting…'}</span>
                            </td>
                            <td>
                              {filled ? <span className="chip pass">verified</span> : <span className="chip pending">pending</span>}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>

                  <div className="object-note">
                    Each criterion is paired with the single receipt that proves it. No criterion is verified on opinion; every
                    pass is backed by a deterministic receipt from the execution replay.
                  </div>
                </div>
              </>
            ) : null}

            {step === 6 ? (
              <>
                <div className="obj-head">
                  <div className="t">Decision gate · readiness 100%</div>
                  <div className="tags">
                    <span className="tag pass">5/5 verified</span>
                    <span className="tag">0 unresolved</span>
                    <span className="tag">contract v3 frozen</span>
                  </div>
                </div>

                <div className="obj-body">
                  <div className="decision">
                    <div className="left">
                      <h3>Why this decision is unlocked</h3>
                      <ul>
                        <li>All 5 acceptance criteria passed against evidence</li>
                        <li>Each criterion bound to a deterministic receipt</li>
                        <li>Out-of-scope items documented and acknowledged</li>
                        <li>No unresolved clarifications, no open risks</li>
                        <li>Contract v3 frozen · hash pinned to packet</li>
                      </ul>
                    </div>

                    <div className="actions">
                      <button type="button" className="btn primary">
                        <span>Accept</span>
                        <span className="sub">freeze packet · close change · archive proof</span>
                      </button>
                      <button type="button" className="btn mauve">
                        <span>Rework</span>
                        <span className="sub">return to clarification with annotations</span>
                      </button>
                      <button type="button" className="btn warn">
                        <span>Block</span>
                        <span className="sub">reject with blocking reason · notify requester</span>
                      </button>
                    </div>
                  </div>

                  <div className="cta">
                    <div>
                      <div className="t">Run one real case through this packet.</div>
                      <div className="s">Apply the decision kernel to your next live request — same spine, same proof rules.</div>
                    </div>
                    <div className="go">Open case ▸</div>
                  </div>
                </div>
              </>
            ) : null}
          </section>

          <section className={cx('primary-cta', primaryCta.tone, primaryCta.disabled && 'disabled')}>
            <div className="body-copy">
              <div className="lbl">{primaryCta.label}</div>
              <div className="ttl">{primaryCta.title}</div>
              <div className="sub">{primaryCta.sub}</div>
            </div>
            <button type="button" className="go" disabled={primaryCta.disabled} onClick={goNext}>
              {primaryCta.disabled ? 'Done' : 'Begin'} ▸<span className="kbd">↵</span>
            </button>
          </section>
        </main>

        <aside className="inspector">
          <div className="insp-head">
            <div className="id">CP-0147 · Change packet</div>
            <div className="title">Manual review step for trial approval</div>
            <div className="life">
              <span className="k">derived state</span>
              <span className="v">{lifecycle}</span>
              <span className="derive">
                · from spine at <b>{STAGES[activeStageIndex].name.toLowerCase()}</b> · intent {readiness.intentPercent}%
              </span>
            </div>
          </div>

          <div className="insp-body">
            {getInspectorSections(step, answeredSet, visibleReceipts, matrixFilled).map((section) => (
              <div key={section.key} className="insp-section">
                <div className="hd">
                  <div className="k">{section.label}</div>
                  <div className="status-row">
                    {section.count ? <div className="c">{section.count}</div> : null}
                    <span className={cx('state', section.stateClass)}>{section.state}</span>
                  </div>
                </div>
                <div className="content">{section.content}</div>
              </div>
            ))}
          </div>
        </aside>

        <footer className="ledger">
          <div className="ledger-head">
            <div className="t">
              <span className="latest-pulse" />Event ledger
            </div>
            <div className="meta">
              <b>{events.length}</b> events · append-only · seq=<b>0001</b> · cp=<b>cp-0147</b>
            </div>
            <div className="controls">
              {CONTROL_MAP[step].map((control) => (
                <button key={control.label} type="button" className="ctrl ghost" onClick={() => controlAction(control.action)}>
                  {control.label}
                </button>
              ))}
            </div>
          </div>

          <div ref={ledgerBodyRef} className="ledger-body">
            {events.map((event, index) => (
              <div key={`${event.ts}-${event.kind}`} className={cx('evt', event.cls, index === lastEventIndex && 'latest')}>
                <span className="ts">{event.ts}</span>
                <span className="kind">{event.kind}</span>
                <span className="note">{event.note}</span>
                <span className="tag-s">{event.tag}</span>
              </div>
            ))}
          </div>
        </footer>
      </div>
    </div>
  );
}

function getSpineSummary(step: StepIndex, answeredCount: number, visibleReceipts: number, matrixFilled: number) {
  const summaries = [
    {
      name: STAGES[0].name,
      facts: [
        { cls: 'amber', html: '<b>6</b> terms flagged' },
        { cls: '', html: '<b>0 / 5</b> clarifications resolved' },
        { cls: '', html: 'contract <b>not assembled</b>' },
      ],
    },
    {
      name: STAGES[1].name,
      facts: [
        { cls: '', html: `<b>${answeredCount} / 5</b> answered` },
        { cls: 'amber', html: `<b>${5 - answeredCount}</b> awaiting input` },
        { cls: '', html: 'readiness rising with each answer' },
      ],
    },
    {
      name: STAGES[2].name,
      facts: [
        { cls: 'pass', html: '<b>5</b> clauses bound from clarifications' },
        { cls: '', html: '<b>4</b> out-of-scope items declared' },
        { cls: '', html: 'awaiting approval' },
      ],
    },
    {
      name: STAGES[3].name,
      facts: [
        { cls: 'pass', html: '<b>4</b> bounded slices derived' },
        { cls: '', html: 'each slice linked to criteria' },
        { cls: '', html: 'proof obligation declared per slice' },
      ],
    },
    {
      name: STAGES[4].name,
      facts: [
        { cls: '', html: `<b>${visibleReceipts} / 5</b> receipts captured` },
        { cls: '', html: 'deterministic · no live runtime' },
        { cls: '', html: 'receipts bound to contract clauses' },
      ],
    },
    {
      name: STAGES[5].name,
      facts: [
        { cls: 'pass', html: `<b>${matrixFilled} / 5</b> criteria verified` },
        { cls: '', html: 'each verdict bound to one receipt' },
        { cls: '', html: 'no verdict without evidence' },
      ],
    },
    {
      name: STAGES[7].name,
      facts: [
        { cls: 'pass', html: '<b>5 / 5</b> verified' },
        { cls: 'pass', html: '<b>0</b> unresolved blockers' },
        { cls: '', html: 'decision unlocked' },
      ],
    },
  ] as const;

  return summaries[step];
}

function getInspectorSections(step: StepIndex, answeredSet: Set<string>, visibleReceipts: number, matrixFilled: number) {
  const intentContent =
    step >= 1 ? (
      <>
        <div className="line"><span className="dot pass" /><span>manual review confirmed required</span></div>
        <div className="line"><span className="dot pass" /><span>ownership rule confirmed</span></div>
        <div className="line"><span className="dot pass" /><span>decision reason confirmed required</span></div>
        <div className="line"><span className="dot pass" /><span>review-visible status confirmed</span></div>
        <div className="line"><span className="dot pass" /><span>audit behavior confirmed</span></div>
      </>
    ) : (
      <>
        <div className="line"><span className="dot amber" /><span>manual review is implied</span></div>
        <div className="line"><span className="dot amber" /><span>ownership rule appears required</span></div>
        <div className="line"><span className="dot amber" /><span>decision reason appears required</span></div>
        <div className="line"><span className="dot amber" /><span>a new review-visible status is implied</span></div>
        <div className="line"><span className="dot amber" /><span>audit behavior is requested</span></div>
      </>
    );

  const clarificationsContent =
    step >= 1 ? (
      <>
        <div className="line">
          <span className={cx('dot', answeredSet.has('review-scope') ? 'pass' : 'amber')} />
          <span className="qn">q1</span>
          <span>scope · {answeredSet.has('review-scope') ? 'trials above $10k ACV' : 'awaiting'}</span>
        </div>
        <div className="line">
          <span className={cx('dot', answeredSet.has('owner-rule') ? 'pass' : 'amber')} />
          <span className="qn">q2·q3</span>
          <span>owner · {answeredSet.has('owner-rule') ? 'trial_reviewer role' : 'awaiting'}</span>
        </div>
        <div className="line">
          <span className={cx('dot', answeredSet.has('reason-rule') ? 'pass' : 'amber')} />
          <span className="qn">q4</span>
          <span>reason · {answeredSet.has('reason-rule') ? 'required · ≥ 20ch' : 'awaiting'}</span>
        </div>
        <div className="line">
          <span className={cx('dot', answeredSet.has('status-model') ? 'pass' : 'amber')} />
          <span className="qn">q5</span>
          <span>status · {answeredSet.has('status-model') ? 'add pending_review' : 'awaiting'}</span>
        </div>
        <div className="line">
          <span className={cx('dot', answeredSet.has('audit-req') ? 'pass' : 'amber')} />
          <span className="qn">q6</span>
          <span>audit · {answeredSet.has('audit-req') ? 'actor+ts+reason+prev' : 'awaiting'}</span>
        </div>
      </>
    ) : (
      <>
        <div className="line"><span className="dot hollow" /><span className="qn">q1</span><span>scope · awaiting</span></div>
        <div className="line"><span className="dot hollow" /><span className="qn">q2·q3</span><span>owner · awaiting</span></div>
        <div className="line"><span className="dot hollow" /><span className="qn">q4</span><span>reason · awaiting</span></div>
        <div className="line"><span className="dot hollow" /><span className="qn">q5</span><span>status · awaiting</span></div>
        <div className="line"><span className="dot hollow" /><span className="qn">q6</span><span>audit · awaiting</span></div>
      </>
    );

  const contractContent =
    step >= 2 ? (
      <>
        <div className="line"><span className="dot pass" /><span>Goal · bounded manual review</span></div>
        <div className="line"><span className="dot pass" /><span>In scope · 4 clauses</span></div>
        <div className="line"><span className="dot pass" /><span>Out of scope · 4 clauses</span></div>
        <div className="line"><span className="dot pass" /><span>Acceptance · 5 criteria</span></div>
        <div className="line"><span className="dot pass" /><span>Proof expectations · 3 items</span></div>
      </>
    ) : (
      <div className="locked-row">
        <span className="glyph">▫</span>
        <span className="txt">locked · needs 5 clarifications resolved</span>
        <span className="dep">after · clarify</span>
      </div>
    );

  const tasksContent =
    step >= 3 ? (
      <>
        {TASKS.map((task) => (
          <div key={task.n} className="line">
            <span className="dot pass" />
            <span>
              {task.n} · {task.title}
            </span>
          </div>
        ))}
      </>
    ) : (
      <div className="locked-row">
        <span className="glyph">▫</span>
        <span className="txt">locked · derived from approved contract</span>
        <span className="dep">after · contract</span>
      </div>
    );

  const evidenceContent =
    step >= 4 && visibleReceipts > 0 ? (
      <>
        {RECEIPTS.slice(0, visibleReceipts).map((receipt) => (
          <div key={receipt.code} className="line">
            <span className={cx('dot', receipt.verdict === 'pass' ? 'pass' : 'block-dot')} />
            <span>{receipt.code}</span>
          </div>
        ))}
      </>
    ) : (
      <div className="locked-row">
        <span className="glyph">▫</span>
        <span className="txt">no receipts · waits execution replay</span>
        <span className="dep">after · tasks</span>
      </div>
    );

  const proofContent =
    step >= 5 ? (
      matrixFilled > 0 ? (
        <>
          {CRITERIA.slice(0, matrixFilled).map((criterion) => (
            <div key={criterion.id} className="line">
              <span className="dot pass" />
              <span>{criterion.name}</span>
            </div>
          ))}
        </>
      ) : (
        <div className="locked-row">
          <span className="glyph">▫</span>
          <span className="txt">awaiting verification pass</span>
          <span className="dep">now</span>
        </div>
      )
    ) : (
      <div className="locked-row">
        <span className="glyph">▫</span>
        <span className="txt">locked · waits verification pass</span>
        <span className="dep">after · evidence</span>
      </div>
    );

  const decisionContent =
    step >= 6 ? (
      <>
        <div className="line"><span className="dot pass" /><span>Accepted · contract v3 frozen</span></div>
        <div className="line"><span className="dot pass" /><span>Proof archive hash pinned</span></div>
      </>
    ) : (
      <div className="locked-row">
        <span className="glyph">▫</span>
        <span className="txt">gate closed · needs proof complete</span>
        <span className="dep">after · proof</span>
      </div>
    );

  return [
    {
      key: 'signals',
      label: 'Observed signals',
      count: null,
      state: step >= 1 ? 'Confirmed' : 'Provisional',
      stateClass: step >= 1 ? 'filled' : 'provisional',
      content: intentContent,
    },
    {
      key: 'clarifications',
      label: 'Clarifications',
      count: step >= 1 ? `${answeredSet.size}/5` : '0/5',
      state: step >= 2 ? 'Filled' : step === 1 ? 'Partial' : 'Queued',
      stateClass: step >= 2 ? 'filled' : 'partial',
      content: clarificationsContent,
    },
    {
      key: 'contract',
      label: 'Contract',
      count: step >= 2 ? 'v3' : null,
      state: step >= 3 ? 'Filled' : step === 2 ? 'Partial' : 'Locked',
      stateClass: step >= 3 ? 'filled' : step === 2 ? 'partial' : 'locked',
      content: contractContent,
    },
    {
      key: 'tasks',
      label: 'Task slices',
      count: step >= 3 ? '4' : null,
      state: step >= 3 ? 'Filled' : 'Locked',
      stateClass: step >= 3 ? 'filled' : 'locked',
      content: tasksContent,
    },
    {
      key: 'evidence',
      label: 'Evidence',
      count: step >= 4 ? `${visibleReceipts}/5` : null,
      state: step >= 4 && visibleReceipts === RECEIPTS.length ? 'Filled' : step >= 4 ? 'Partial' : 'Locked',
      stateClass: step >= 4 && visibleReceipts === RECEIPTS.length ? 'filled' : step >= 4 ? 'partial' : 'locked',
      content: evidenceContent,
    },
    {
      key: 'proof',
      label: 'Proof',
      count: step >= 5 ? `${matrixFilled}/5` : null,
      state: step >= 5 && matrixFilled === CRITERIA.length ? 'Filled' : step >= 5 ? 'Partial' : 'Locked',
      stateClass: step >= 5 && matrixFilled === CRITERIA.length ? 'filled' : step >= 5 ? 'partial' : 'locked',
      content: proofContent,
    },
    {
      key: 'decision',
      label: 'Decision',
      count: null,
      state: step >= 6 ? 'Filled' : 'Locked',
      stateClass: step >= 6 ? 'filled' : 'locked',
      content: decisionContent,
    },
  ];
}

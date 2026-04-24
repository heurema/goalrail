import { useEffect, useMemo, useState } from 'react';

import './App.css';

type StepIndex = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7;
type RepoId = 'trialops-demo' | 'billing-api';
type RepoFilter = RepoId | 'all';
type ApprovalState = 'pending' | 'accepted' | 'rework' | 'blocked';
type Tone = 'mauve' | 'amber' | 'pass' | 'block';

interface Stage {
  id: string;
  name: string;
}

interface RepoContext {
  repo: RepoId;
  bound: 'yes' | 'no';
  init: string;
  docsIndexed: number;
  readiness: number;
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
  lane: 'external runtime' | 'manual' | 'review/proof';
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
  repo: RepoId;
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

const STAGES: Stage[] = [
  { id: 'goal-intake', name: 'Goal intake' },
  { id: 'clarification', name: 'Clarification' },
  { id: 'working-contract', name: 'Working contract' },
  { id: 'work-items', name: 'Work items' },
  { id: 'execution-evidence', name: 'Execution evidence' },
  { id: 'verification', name: 'Verification' },
  { id: 'proof', name: 'Proof' },
  { id: 'approval', name: 'Approval / Decision' },
];

const REPO_OPTIONS: Array<{ value: RepoFilter; label: string }> = [
  { value: 'trialops-demo', label: 'trialops-demo' },
  { value: 'billing-api', label: 'billing-api' },
  { value: 'all', label: 'All repos' },
];

const REPO_CONTEXTS: Record<RepoId, RepoContext> = {
  'trialops-demo': {
    repo: 'trialops-demo',
    bound: 'yes',
    init: 'complete',
    docsIndexed: 12,
    readiness: 72,
    checklist: [
      { label: 'Tests', value: 'detected', tone: 'pass' },
      { label: 'CI', value: 'connected', tone: 'mauve' },
      { label: 'AGENTS/rules', value: 'present', tone: 'pass' },
    ],
    runtimePolicy: 'local-only',
    runtimes: ['Codex CLI', 'Claude Code', 'manual'],
  },
  'billing-api': {
    repo: 'billing-api',
    bound: 'yes',
    init: 'complete',
    docsIndexed: 18,
    readiness: 84,
    checklist: [
      { label: 'Tests', value: 'detected', tone: 'pass' },
      { label: 'CI', value: 'connected', tone: 'pass' },
      { label: 'AGENTS/rules', value: 'present', tone: 'pass' },
    ],
    runtimePolicy: 'local-only',
    runtimes: ['Codex CLI', 'manual', 'read-only review lane'],
  },
};

const CONTRACTS: ContractRecord[] = [
  {
    id: 'C-0147',
    title: 'Manual review gate',
    repo: 'trialops-demo',
    owner: 'Vitaly · product + delivery',
    scopeSurface: 'demo shell · contract walkthrough',
    summary: 'Add repo-aware contract detail and explicit human approval without expanding beyond the demo shell.',
    defaultStep: 0,
    goal:
      'Introduce a bounded contract-first workspace view for the demo shell so one repo-scoped contract can move from intake to proof and explicit human approval without adding backend, routing, or real integrations.',
    intakeNotes: [
      'Repo is the context container; contract is the primary working object.',
      'Change packet view stays human-facing and scoped to one selected contract.',
      'Project context must stay separate from the contract pipeline.',
    ],
    inScope: [
      'Repo selector with trialops-demo, billing-api, and All repos.',
      'Contracts list with repo badges and one selected contract in detail.',
      'Explicit working contract, work items, execution evidence, verification, proof, and approval stages.',
      'Project context block for repo binding, readiness, policy, and runtimes.',
    ],
    outOfScope: [
      'Backend, API calls, auth, routing, persistence, and server logic.',
      'Real repo scanning, runtime execution, or proof generation.',
      'A separate aggregate dashboard or chat-first workspace.',
    ],
    acceptance: [
      'Selected repo filters the contracts list by default; All repos acts as overview mode only.',
      'Selected contract detail always shows the repo that owns that contract.',
      'Project readiness is visible outside the contract flow.',
      'Final stage requires explicit human approval before Accepted.',
    ],
    proofExpectations: [
      'Show which criteria are covered and which mock evidence supports each one.',
      'State what changed and what did not change in the demo shell.',
      'Keep the walkthrough inspectable without implying real execution.',
    ],
    policyNote:
      'Inherited project context: repo rules are present, runtime policy is local-only, and this walkthrough stays fully mocked inside the demo shell.',
    clarifications: [
      {
        ref: 'repo selector',
        question: 'Which repo views must exist in the shell?',
        answer: 'trialops-demo, billing-api, and All repos',
        note: 'Default stays on trialops-demo; All repos is a secondary overview mode.',
      },
      {
        ref: 'primary object',
        question: 'What is the main working object?',
        answer: 'Contract, not repo-agnostic change packets',
        note: 'The detail view can still say Change packet view for the selected contract.',
      },
      {
        ref: 'project context',
        question: 'Where does repo binding / readiness live?',
        answer: 'Outside the contract pipeline in a persistent side block',
        note: 'Delivery readiness leaves the topbar and moves into repo context.',
      },
      {
        ref: 'work items',
        question: 'How should bounded tasks be reframed?',
        answer: 'As Work items with lane, scope, status, and proof obligation',
        note: 'At least one work item must be clearly manual-only.',
      },
      {
        ref: 'human gate',
        question: 'What must happen before final outcome?',
        answer: 'Explicit human approval with approve, rework, or block actions',
        note: 'Awaiting approval and Accepted stay visually distinct.',
      },
    ],
    workItems: [
      {
        id: 'WI-01',
        title: 'Repo-aware shell framing',
        lane: 'external runtime',
        scope: 'src/App.tsx · topbar, left rail, selected contract detail',
        status: 'Scoped',
        proofObligation: 'Touched-surface note + center-panel contract state receipt',
      },
      {
        id: 'WI-02',
        title: 'Human approval copy check',
        lane: 'manual',
        scope: 'Approval wording and verification block',
        status: 'Manual only',
        proofObligation: 'Manual step marked done in the evidence pack',
      },
      {
        id: 'WI-03',
        title: 'Verification / proof reshaping',
        lane: 'review/proof',
        scope: 'Criteria coverage matrix + proof summary',
        status: 'Queued',
        proofObligation: 'Criteria-to-evidence coverage matrix',
      },
    ],
    evidence: [
      { label: 'Runtime used', value: 'Codex CLI in a local workspace outside Goalrail', tone: 'mauve' },
      { label: 'Checkpoint synced', value: 'Contract C-0147 · packet v3 synced back into the mock shell', tone: 'pass' },
      { label: 'Touched files / changed scope', value: 'src/App.tsx, src/App.css · demo shell only', tone: 'pass' },
      { label: 'Receipts', value: 'Stage-state snapshots + criteria mapping receipts', tone: 'mauve' },
      { label: 'Manual step marked done', value: 'Human approval wording reviewed by operator', tone: 'amber' },
      { label: 'Artifact attached', value: 'Change summary packet · proof notes · replay instructions', tone: 'pass' },
    ],
    verification: [
      {
        criterion: 'Repo selector scopes contracts without creating a separate dashboard',
        support: 'Repo filter state + selected-contract detail persists in center panel',
        outcome: 'Covered',
      },
      {
        criterion: 'Project-level readiness sits outside the contract spine',
        support: 'Persistent Project context side block with readiness meter and checklist',
        outcome: 'Covered',
      },
      {
        criterion: 'Work items expose lane, scope, status, and proof obligation',
        support: 'Three work item rows including a manual-only lane',
        outcome: 'Covered',
      },
      {
        criterion: 'Human approval is explicit before final decision',
        support: 'Approval / Decision stage with Approve result, Request rework, and Block actions',
        outcome: 'Covered',
      },
    ],
    changed: [
      'Repo selector and repo-aware contracts list were added to the shell.',
      'One selected contract now owns the center detail and change packet view.',
      'Project context moved out of the flow and now carries delivery readiness.',
      'Approval / Decision now requires a visible human review step.',
    ],
    unchanged: [
      'No backend, API calls, routing, auth, persistence, or server logic.',
      'No real repo scan, runtime execution, or integration sync.',
      'No visual redesign beyond bounded shell rewiring and copy changes.',
    ],
    trust: [
      'All data stays in local mocked constants and UI state.',
      'Execution evidence is framed as external runtime output, not as Goalrail-native execution.',
      'Proof explains both changed scope and untouched scope to prevent scope drift.',
      'Final outcome still waits for a human decision.',
    ],
    howToVerify: [
      'Review the change summary in the selected contract view.',
      'Inspect touched files listed in Execution evidence.',
      'Replay the UI state through each stage of the walkthrough.',
      'Confirm each acceptance criterion in the Verification and Proof stages.',
    ],
    activity: {
      0: [
        { kind: 'goal.intake', note: 'Contract seeded for trialops-demo', tone: 'mauve' },
        { kind: 'repo.bound', note: 'Repo context pinned to trialops-demo', tone: 'pass' },
      ],
      1: [
        { kind: 'clarification.answered', note: '5 bounded questions collapsed into contract inputs', tone: 'mauve' },
      ],
      2: [
        { kind: 'contract.drafted', note: 'Goal, scope, criteria, and proof expectations assembled', tone: 'mauve' },
      ],
      3: [
        { kind: 'work-items.ready', note: 'External runtime, manual, and review/proof lanes declared', tone: 'pass' },
      ],
      4: [
        { kind: 'evidence.synced', note: 'External runtime receipts attached to the contract packet', tone: 'pass' },
      ],
      5: [
        { kind: 'verification.covered', note: 'Criteria mapped to named evidence receipts', tone: 'pass' },
      ],
      6: [
        { kind: 'proof.ready', note: 'Changed vs unchanged scope summarized for review', tone: 'pass' },
      ],
      7: [
        { kind: 'approval.pending', note: 'Human decision required before Accepted', tone: 'amber' },
      ],
    },
  },
  {
    id: 'C-0148',
    title: 'CSV export filters',
    repo: 'trialops-demo',
    owner: 'Masha · delivery lead',
    scopeSurface: 'export modal · filter chip copy',
    summary: 'Execution is in flight and proof lanes are still collecting receipts.',
    defaultStep: 4,
    goal: 'Make CSV export filters explicit in the demo shell without changing export transport or persistence.',
    intakeNotes: ['Contract stays bound to trialops-demo.', 'Manual review is needed for export naming copy.'],
    inScope: ['Filter chips', 'Selection summary', 'Export readiness note'],
    outOfScope: ['Real CSV generation', 'Storage', 'Background jobs'],
    acceptance: ['Selected filters are inspectable', 'Manual naming note is visible'],
    proofExpectations: ['Receipt for changed scope', 'Manual copy check'],
    policyNote: 'Inherited project context: local-only mock with no real export runtime.',
    clarifications: [
      {
        ref: 'filters',
        question: 'Which filters matter in the demo?',
        answer: 'Owner, date range, and state',
        note: 'Only the UI surface is in scope.',
      },
      {
        ref: 'naming',
        question: 'Who approves the export copy?',
        answer: 'Manual reviewer',
        note: 'Naming remains manual-only.',
      },
    ],
    workItems: [
      {
        id: 'WI-11',
        title: 'Export filter shell',
        lane: 'external runtime',
        scope: 'Filter chip row + summary strip',
        status: 'Executing',
        proofObligation: 'Snapshot receipt',
      },
      {
        id: 'WI-12',
        title: 'Copy signoff',
        lane: 'manual',
        scope: 'Human-readable export label',
        status: 'Manual only',
        proofObligation: 'Operator signoff note',
      },
    ],
    evidence: [
      { label: 'Runtime used', value: 'Codex CLI', tone: 'mauve' },
      { label: 'Checkpoint synced', value: 'Export filter state fixture', tone: 'pass' },
    ],
    verification: [
      { criterion: 'Filters stay inspectable', support: 'UI filter summary', outcome: 'Partial' },
    ],
    changed: ['Filter framing in the shell'],
    unchanged: ['No export backend'],
    trust: ['Mock-only surface'],
    howToVerify: ['Review filter summary'],
    activity: {
      0: [{ kind: 'goal.intake', note: 'CSV export filter request logged', tone: 'mauve' }],
      4: [{ kind: 'execution.running', note: 'Receipts are still being attached', tone: 'amber' }],
    },
  },
  {
    id: 'C-0151',
    title: 'Pricing toggle cleanup',
    repo: 'trialops-demo',
    owner: 'Nika · product ops',
    scopeSurface: 'pricing panel copy',
    summary: 'Proof is ready and waiting for a human approval decision.',
    defaultStep: 7,
    goal: 'Clean up pricing toggle language in the demo shell and hold final release until human approval.',
    intakeNotes: ['UI-only mock update.'],
    inScope: ['Pricing panel copy'],
    outOfScope: ['Billing logic'],
    acceptance: ['Copy is inspectable'],
    proofExpectations: ['Approval note'],
    policyNote: 'Human approval stays required before acceptance.',
    clarifications: [
      {
        ref: 'copy',
        question: 'Who approves the wording?',
        answer: 'Manual reviewer',
        note: 'No auto-accept.',
      },
    ],
    workItems: [
      {
        id: 'WI-21',
        title: 'Copy cleanup',
        lane: 'manual',
        scope: 'Pricing labels',
        status: 'Awaiting approval',
        proofObligation: 'Reviewer decision',
      },
    ],
    evidence: [{ label: 'Artifact attached', value: 'Copy review note', tone: 'pass' }],
    verification: [{ criterion: 'Copy changed only in scope', support: 'Review note', outcome: 'Covered' }],
    changed: ['Pricing copy only'],
    unchanged: ['Billing behavior'],
    trust: ['Manual approval gate remains active'],
    howToVerify: ['Read the copy diff'],
    activity: {
      0: [{ kind: 'goal.intake', note: 'Pricing copy request queued', tone: 'mauve' }],
      7: [{ kind: 'approval.pending', note: 'Reviewer decision still open', tone: 'amber' }],
    },
  },
  {
    id: 'C-0149',
    title: 'Audit trail hardening',
    repo: 'billing-api',
    owner: 'Roma · platform',
    scopeSurface: 'audit receipt framing',
    summary: 'Proof packet is assembled and ready to inspect.',
    defaultStep: 6,
    goal: 'Surface audit-trail proof for a billing-api contract without implying real server writes.',
    intakeNotes: ['Repo context is billing-api.'],
    inScope: ['Audit proof summary', 'Receipt labels'],
    outOfScope: ['Database writes'],
    acceptance: ['Proof packet states what changed and what did not change'],
    proofExpectations: ['Trust explanation'],
    policyNote: 'Billing-api keeps the same local-only runtime policy in this demo.',
    clarifications: [
      {
        ref: 'audit',
        question: 'What must the proof show?',
        answer: 'Changed scope, unchanged scope, and trust reasons',
        note: 'No live audit stream exists in the demo.',
      },
    ],
    workItems: [
      {
        id: 'WI-31',
        title: 'Audit proof summary',
        lane: 'review/proof',
        scope: 'Proof packet copy',
        status: 'Proof ready',
        proofObligation: 'Named trust reasons',
      },
    ],
    evidence: [{ label: 'Receipts', value: 'Audit scope map + proof note', tone: 'pass' }],
    verification: [{ criterion: 'Proof is inspectable', support: 'Proof summary', outcome: 'Covered' }],
    changed: ['Audit proof framing'],
    unchanged: ['No billing runtime'],
    trust: ['Proof names unchanged scope'],
    howToVerify: ['Inspect proof summary'],
    activity: {
      0: [{ kind: 'goal.intake', note: 'Audit hardening request logged', tone: 'mauve' }],
      6: [{ kind: 'proof.ready', note: 'Proof packet is ready for inspection', tone: 'pass' }],
    },
  },
  {
    id: 'C-0150',
    title: 'Lead sync cleanup',
    repo: 'billing-api',
    owner: 'Ira · growth ops',
    scopeSurface: 'sync status labels',
    summary: 'Contract is active and work items are bounded, but execution has not started.',
    defaultStep: 3,
    goal: 'Clarify and scope lead sync cleanup in the billing-api demo lane without building a sync integration.',
    intakeNotes: ['Contract is still active.'],
    inScope: ['Status labels', 'Human verification copy'],
    outOfScope: ['Real CRM sync'],
    acceptance: ['Status labels are clear'],
    proofExpectations: ['Verification checklist'],
    policyNote: 'No integration runtime exists in this demo.',
    clarifications: [
      {
        ref: 'sync',
        question: 'Is a real integration expected?',
        answer: 'No, mock-only shell update',
        note: 'Execution remains local-only.',
      },
    ],
    workItems: [
      {
        id: 'WI-41',
        title: 'Scope cleanup',
        lane: 'external runtime',
        scope: 'Status label shell',
        status: 'Active',
        proofObligation: 'Touched scope summary',
      },
    ],
    evidence: [{ label: 'Checkpoint synced', value: 'Task plan only', tone: 'amber' }],
    verification: [{ criterion: 'Scope stays bounded', support: 'Task plan', outcome: 'Partial' }],
    changed: ['Task plan only'],
    unchanged: ['No CRM sync'],
    trust: ['Contract scope is explicit'],
    howToVerify: ['Inspect work items'],
    activity: {
      0: [{ kind: 'goal.intake', note: 'Lead sync cleanup request logged', tone: 'mauve' }],
      3: [{ kind: 'work-items.ready', note: 'Execution lanes are prepared', tone: 'pass' }],
    },
  },
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
  if (approval === 'accepted') return 'Accepted';
  if (approval === 'blocked') return 'Blocked';
  if (approval === 'rework') return 'Needs rework';
  if (step >= 7) return 'Awaiting approval';
  if (step >= 6) return 'Proof ready';
  if (step >= 4) return 'Executing';
  return 'Active';
}

function getStatusTone(status: string): Tone {
  if (status === 'Accepted' || status === 'Proof ready') return 'pass';
  if (status === 'Awaiting approval' || status === 'Needs rework') return 'amber';
  if (status === 'Blocked') return 'block';
  return 'mauve';
}

function getMeters(step: StepIndex, approval: ApprovalState) {
  const contractPercent = [18, 42, 68, 76, 84, 90, 96, approval === 'accepted' ? 100 : 96][step];
  const executionPercent = [0, 0, 16, 42, 76, 82, 86, 86][step];
  const proofPercent = [0, 0, 10, 18, 42, 72, 90, approval === 'accepted' ? 100 : 92][step];

  return {
    contract: { percent: contractPercent, label: STAGES[Math.min(step, 2)].name },
    execution: {
      percent: executionPercent,
      label: step < 4 ? 'Queued' : step < 6 ? 'Receipts synced' : 'Ready for review',
    },
    proof: {
      percent: proofPercent,
      label: approval === 'accepted' ? 'Accepted' : step >= 7 ? 'Awaiting approval' : step >= 6 ? 'Proof ready' : 'Drafting',
    },
  };
}

function getStepSummary(step: StepIndex) {
  return (
    [
      'Turn one repo-scoped request into a contract-first walkthrough.',
      'Collapse open questions into bounded answers pinned to the contract.',
      'Freeze the working contract before any work items claim progress.',
      'Show lanes, scope, status, and proof obligation for each work item.',
      'Execution evidence is collected from outside Goalrail and synced back in.',
      'Verification maps criteria to evidence instead of generic status copy.',
      'Proof says what changed, what did not change, and why to trust it.',
      'Human approval decides the final outcome for the contract.',
    ] as const
  )[step];
}

function getActivity(contract: ContractRecord, step: StepIndex, approval: ApprovalState) {
  const timeline = [
    { ts: '09:42:08', kind: 'contract.selected', note: `${contract.id} pinned in center detail`, tone: 'mauve' as Tone },
    { ts: '09:42:12', kind: 'repo.context', note: `Repo ${contract.repo} context loaded`, tone: 'pass' as Tone },
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
          ? 'Human approval recorded · result accepted'
          : approval === 'blocked'
            ? 'Human reviewer blocked the packet'
            : approval === 'rework'
              ? 'Human reviewer requested rework'
              : 'Awaiting human approval before final outcome',
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
        <div className="compat-line">Raw request · inbound</div>
        <div className="detail-grid two-up">
          <section className="detail-block">
            <div className="detail-kicker">Goal intake</div>
            <h2 className="stage-title">{contract.title}</h2>
            <p className="detail-copy">{contract.goal}</p>
            <ul className="bullet-list">
              {contract.intakeNotes.map((note) => (
                <li key={note}>{note}</li>
              ))}
            </ul>
          </section>

          <section className="detail-block">
            <div className="detail-kicker">Request packet</div>
            <dl className="key-grid compact-grid">
              <div>
                <dt>Repo</dt>
                <dd>{contract.repo}</dd>
              </div>
              <div>
                <dt>Contract</dt>
                <dd>{contract.id}</dd>
              </div>
              <div>
                <dt>Surface</dt>
                <dd>{contract.scopeSurface}</dd>
              </div>
              <div>
                <dt>Policy</dt>
                <dd>{projectContext.runtimePolicy}</dd>
              </div>
            </dl>
            <div className="panel-note">
              Repo binding exists already, but it is <b>project context</b>, not a pipeline stage.
            </div>
          </section>
        </div>
      </div>
    );
  }

  if (step === 1) {
    return (
      <div className="stage-content">
        <div className="compat-line">Clarification cards · {contract.clarifications.length} of {contract.clarifications.length}</div>
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
                <div className={cx('clarification-foot', pinned && 'resolved')}>{pinned ? 'Answer pinned to contract' : 'Pending contract pin'}</div>
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
        <div className="compat-line">Working contract · draft v3</div>
        <section className="detail-block hero-block">
          <div className="detail-kicker">Goal</div>
          <p className="detail-copy">{contract.goal}</p>
        </section>

        <div className="detail-grid two-up">
          <ListBlock title="In scope" items={contract.inScope} />
          <ListBlock title="Out of scope" items={contract.outOfScope} />
          <ListBlock title="Acceptance criteria" items={contract.acceptance} />
          <ListBlock title="Proof expectations" items={contract.proofExpectations} />
        </div>

        <section className="detail-block">
          <div className="detail-kicker">Inherited project context / policy note</div>
          <p className="detail-copy">{contract.policyNote}</p>
          <div className="inline-actions">
            <button className="ghost-button" type="button" onClick={onAdvance}>
              Freeze contract
            </button>
            <button className="primary-button small" type="button" onClick={onAdvance}>
              Approve contract
            </button>
          </div>
        </section>
      </div>
    );
  }

  if (step === 3) {
    return (
      <div className="stage-content">
        <div className="section-tagline">Work items</div>
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
                  <dt>Lane / type</dt>
                  <dd>{item.lane}</dd>
                </div>
                <div>
                  <dt>Scope / surface</dt>
                  <dd>{item.scope}</dd>
                </div>
                <div>
                  <dt>Status</dt>
                  <dd>{item.status}</dd>
                </div>
                <div>
                  <dt>Proof obligation</dt>
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
        <div className="section-tagline">Execution evidence</div>
        <div className="panel-note strong-note">
          Execution happened <b>outside Goalrail</b>. Goalrail records synced evidence for the selected contract, not a chat log.
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
        <div className="section-tagline">Verification</div>
        <div className="verify-list">
          {contract.verification.slice(0, visibleVerification).map((row) => (
            <article key={`${contract.id}-${row.criterion}`} className="verify-row">
              <div className="verify-main">
                <div className="verify-criterion">{row.criterion}</div>
                <div className="verify-support">{row.support}</div>
              </div>
              <div className="verify-side">
                <div className="detail-kicker">Outcome</div>
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
        <div className="section-tagline">Proof</div>
        <div className="detail-grid two-up">
          <ListBlock title="What changed" items={contract.changed} />
          <ListBlock title="What did not change" items={contract.unchanged} />
        </div>
        <ListBlock title="Why this result is trustworthy" items={contract.trust} />
      </div>
    );
  }

  const approvalLabel =
    approval === 'accepted'
      ? 'Accepted'
      : approval === 'blocked'
        ? 'Blocked'
        : approval === 'rework'
          ? 'Needs rework'
          : 'Awaiting approval';

  return (
    <div className="stage-content">
      <div className="section-tagline">Approval / Decision</div>
      <div className="approval-state-row">
        <div className="detail-kicker">Decision state</div>
        <div className={cx('status-pill', getStatusTone(approvalLabel))}>{approvalLabel}</div>
      </div>

      <div className="detail-grid two-up">
        <ListBlock title="What changed" items={contract.changed} />
        <ListBlock title="What did not change" items={contract.unchanged} />
        <ListBlock title="How to verify" items={contract.howToVerify} />
        <ListBlock title="Proof expectations" items={contract.proofExpectations} />
      </div>

      <section className="detail-block">
        <div className="detail-kicker">Human approval action</div>
        <div className="decision-actions">
          <button className="primary-button" type="button" onClick={() => onDecision('accepted')}>
            Approve result
          </button>
          <button className="ghost-button" type="button" onClick={() => onDecision('rework')}>
            Request rework
          </button>
          <button className="ghost-button danger" type="button" onClick={() => onDecision('blocked')}>
            Block
          </button>
        </div>
      </section>
    </div>
  );
}

export default function App() {
  const [repoFilter, setRepoFilter] = useState<RepoFilter>('trialops-demo');
  const [selectedContractId, setSelectedContractId] = useState('C-0147');
  const [contractSteps, setContractSteps] = useState<Record<string, StepIndex>>(INITIAL_STEPS);
  const [approvalStates, setApprovalStates] = useState<Record<string, ApprovalState>>(INITIAL_APPROVALS);
  const [visibleClarifications, setVisibleClarifications] = useState(0);
  const [visibleEvidence, setVisibleEvidence] = useState(0);
  const [visibleVerification, setVisibleVerification] = useState(0);

  const filteredContracts = useMemo(() => {
    return repoFilter === 'all' ? CONTRACTS : CONTRACTS.filter((contract) => contract.repo === repoFilter);
  }, [repoFilter]);

  useEffect(() => {
    if (!filteredContracts.some((contract) => contract.id === selectedContractId)) {
      setSelectedContractId(filteredContracts[0]?.id ?? CONTRACTS[0].id);
    }
  }, [filteredContracts, selectedContractId]);

  const selectedContract = useMemo(() => {
    return CONTRACTS.find((contract) => contract.id === selectedContractId) ?? CONTRACTS[0];
  }, [selectedContractId]);

  const step = contractSteps[selectedContract.id] ?? selectedContract.defaultStep;
  const approval = approvalStates[selectedContract.id] ?? 'pending';
  const selectedStatus = getStatus(step, approval);
  const projectContext = REPO_CONTEXTS[selectedContract.repo];
  const meters = getMeters(step, approval);
  const activity = useMemo(() => getActivity(selectedContract, step, approval), [selectedContract, step, approval]);

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

  const primaryActionLabel =
    step === 0
      ? 'Begin'
      : step === 1
        ? 'Begin contract'
        : step === 2
          ? 'Freeze contract'
          : step === 3
            ? 'Open execution evidence'
            : step === 4
              ? 'Open verification'
              : step === 5
                ? 'Open proof'
                : step === 6
                  ? 'Open approval'
                  : approval === 'accepted'
                    ? 'Replay state'
                    : null;

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
            <div className="meter amber">
              <div className="row">
                <div className="label">Contract</div>
                <div className="val">{meters.contract.label}</div>
              </div>
              <div className="bar">
                <i style={{ width: `${meters.contract.percent}%` }} />
              </div>
            </div>

            <div className="meter mauve">
              <div className="row">
                <div className="label">Execution</div>
                <div className="val">{meters.execution.label}</div>
              </div>
              <div className="bar">
                <i style={{ width: `${meters.execution.percent}%` }} />
              </div>
            </div>

            <div className="meter pass">
              <div className="row">
                <div className="label">Proof</div>
                <div className="val">{meters.proof.label}</div>
              </div>
              <div className="bar">
                <i style={{ width: `${meters.proof.percent}%` }} />
              </div>
            </div>
          </div>

          <div className="topbar-state">
            <div className="state-chip">
              <span className="k">Repo</span>
              <span className="v">{selectedContract.repo}</span>
            </div>
            <div className="state-chip">
              <span className="k">Status</span>
              <span className="v">{selectedStatus}</span>
            </div>
          </div>
        </header>

        <aside className="rail">
          <div className="group-label">Repo</div>
          <div className="rail-section">
            <label className="select-wrap">
              <span className="select-label">Repo selector</span>
              <select value={repoFilter} onChange={(event) => setRepoFilter(event.target.value as RepoFilter)}>
                {REPO_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
            {repoFilter === 'all' ? (
              <div className="rail-note">All repos is overview mode only. The center panel stays pinned to one selected contract.</div>
            ) : (
              <div className="rail-note">Default view stays repo-scoped. Contract detail always reflects the selected contract repo.</div>
            )}
          </div>

          <div className="group-label">Contracts</div>
          <div className="contract-list" aria-label="Contracts">
            {filteredContracts.map((contract) => {
              const contractStep = contractSteps[contract.id] ?? contract.defaultStep;
              const contractApproval = approvalStates[contract.id] ?? 'pending';
              const status = getStatus(contractStep, contractApproval);

              return (
                <button
                  key={contract.id}
                  className={cx('contract-row', contract.id === selectedContract.id && 'active')}
                  type="button"
                  onClick={() => setSelectedContractId(contract.id)}
                >
                  <div className="contract-row-top">
                    <span className="contract-id">{contract.id}</span>
                    <span className={cx('status-pill', getStatusTone(status))}>{status}</span>
                  </div>
                  <div className="contract-title">{contract.title}</div>
                  <div className="contract-summary">{contract.summary}</div>
                  <div className="contract-row-meta">
                    <span className="repo-badge">{contract.repo}</span>
                    <span className="contract-owner">{contract.owner}</span>
                  </div>
                </button>
              );
            })}
          </div>

          <div className="case">
            <div className="k">Mode</div>
            <div className="v">Contract-first demo shell</div>
            <div className="sub">Local mocked state only · no backend · no routing</div>
          </div>
        </aside>

        <main className="canvas">
          <section className="spine">
            <div className="spine-head">
              <div>
                <div className="t">Contract {selectedContract.id} · Change packet view</div>
                <div className="id">Change spine · cp-{selectedContract.id.slice(2).toLowerCase()}</div>
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
                    <div className="meta">{step > index ? 'done' : step === index ? 'current' : 'queued'}</div>
                  </div>
                );
              })}
            </div>

            <div className="active-summary">
              <div className="marker">Active stage</div>
              <div className="stage-name">{STAGES[step].name}</div>
              <div className="facts">
                <div className="f">
                  Repo <b>{selectedContract.repo}</b>
                </div>
                <div className="f">
                  Contract <b>{selectedContract.id}</b>
                </div>
                <div className="f pass">
                  Status <b>{selectedStatus}</b>
                </div>
              </div>
            </div>
          </section>

          <section className="object">
            <div className="obj-head">
              <div>
                <div className="t">Selected contract</div>
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
              <div className="t">Project context</div>
              <div className="id">Repo {projectContext.repo}</div>
            </div>
            <dl className="key-grid">
              <div>
                <dt>Repo</dt>
                <dd>{projectContext.repo}</dd>
              </div>
              <div>
                <dt>Bound</dt>
                <dd>{projectContext.bound}</dd>
              </div>
              <div>
                <dt>Init</dt>
                <dd>{projectContext.init}</dd>
              </div>
              <div>
                <dt>Docs indexed</dt>
                <dd>{projectContext.docsIndexed}</dd>
              </div>
            </dl>

            <div className="readiness-block">
              <div className="row">
                <div className="label">Delivery readiness</div>
                <div className="val">{projectContext.readiness}/100</div>
              </div>
              <div className="bar">
                <i style={{ width: `${projectContext.readiness}%` }} />
              </div>
            </div>

            <div className="checklist-block">
              <div className="detail-kicker">Delivery readiness checklist</div>
              {projectContext.checklist.map((item) => (
                <div key={`${projectContext.repo}-${item.label}`} className="check-row">
                  <span>{item.label}</span>
                  <span className={cx('check-value', item.tone)}>{item.value}</span>
                </div>
              ))}
            </div>

            <div className="detail-kicker">Runtime policy</div>
            <div className="panel-copy">{projectContext.runtimePolicy}</div>
            <div className="detail-kicker top-gap">Available runtimes</div>
            <div className="chip-row">
              {projectContext.runtimes.map((runtime) => (
                <span key={runtime} className="tag">
                  {runtime}
                </span>
              ))}
            </div>
          </section>

          <section className="panel-card">
            <div className="panel-head">
              <div className="t">Ambiguity inspector</div>
              <div className="id">Resolved into contract inputs</div>
            </div>
            <div className="inspector-list">
              {selectedContract.clarifications.map((card, index) => {
                const resolved = step > 1 || index < visibleClarifications;
                return (
                  <div key={`${selectedContract.id}-inspector-${card.ref}`} className="inspector-row">
                    <div>
                      <div className="inspector-term">{card.ref}</div>
                      <div className="inspector-note">{card.note}</div>
                    </div>
                    <div className={cx('status-pill', resolved ? 'pass' : 'amber')}>{resolved ? 'Resolved' : 'Open'}</div>
                  </div>
                );
              })}
            </div>
          </section>

          <section className="panel-card compact-card">
            <div className="panel-head">
              <div className="t">Selection</div>
              <div className="id">Current detail</div>
            </div>
            <dl className="key-grid compact-grid">
              <div>
                <dt>Contract</dt>
                <dd>{selectedContract.id}</dd>
              </div>
              <div>
                <dt>Repo</dt>
                <dd>{selectedContract.repo}</dd>
              </div>
              <div>
                <dt>Stage</dt>
                <dd>{STAGES[step].name}</dd>
              </div>
              <div>
                <dt>Status</dt>
                <dd>{selectedStatus}</dd>
              </div>
            </dl>
          </section>
        </aside>

        <section className="bottompanel">
          <section className="panel-card activity-card">
            <div className="panel-head">
              <div className="t">Workspace activity</div>
              <div className="id">No chat log · contract events only</div>
            </div>
            <div className="activity-list">
              {activity.map((entry, index) => (
                <div key={`${entry.ts}-${entry.kind}-${index}`} className="activity-row">
                  <div className="activity-ts">{entry.ts}</div>
                  <div className="activity-body">
                    <div className="activity-kind">{entry.kind}</div>
                    <div className="activity-note">{entry.note}</div>
                  </div>
                  <div className={cx('status-pill', entry.tone)}>{entry.tone === 'pass' ? 'pass' : entry.tone === 'block' ? 'block' : entry.tone === 'amber' ? 'review' : 'event'}</div>
                </div>
              ))}
            </div>
          </section>

          <section className="panel-card control-card">
            <div className="panel-head">
              <div className="t">Stage controls</div>
              <div className="id">Mock walkthrough only</div>
            </div>
            <div className="control-copy">{getStepSummary(step)}</div>
            <div className="control-meta">
              <span>Selected repo: {repoFilter === 'all' ? 'All repos' : repoFilter}</span>
              <span>Detail repo: {selectedContract.repo}</span>
            </div>
            <div className="control-actions">
              <button className="ghost-button" type="button" onClick={goBack} disabled={step === 0}>
                Back
              </button>
              {primaryActionLabel ? (
                <button className="primary-button" type="button" onClick={step === 7 ? resetSelected : goNext}>
                  {primaryActionLabel}
                </button>
              ) : null}
              <button className="ghost-button" type="button" onClick={resetSelected}>
                Reset
              </button>
            </div>
          </section>
        </section>
      </div>
    </div>
  );
}

import { useEffect, useMemo, useState, type ReactNode } from 'react';

import './App.css';

type StepIndex = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7;
type ActiveSurface = 'contracts' | 'readiness' | 'proof';
type RepoId = 'trialops-demo' | 'billing-api' | 'frontend-console';
type ContractRepoId = 'trialops-demo' | 'billing-api';
type RepoFilter = ContractRepoId | 'all';
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
  scanStatus: string;
  testsStatus: string;
  ciStatus: string;
  ownersRulesStatus: string;
  proofSurfaceStatus: string;
  recommendedMode: string;
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
  repo: ContractRepoId;
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

interface ProofFeedItem {
  id: string;
  contractId: string;
  repo: ContractRepoId;
  proofStatus: string;
  decisionStatus: string;
  humanApproval: string;
  linkedEvidence: string;
  criteriaCoverage: string;
  summary: string;
  tone: Tone;
  changed: string[];
  unchanged: string[];
  verified: string[];
  decisionTrail: string[];
  archiveLine: string;
}

interface MobileContractQueueItem {
  id: string;
  title: string;
  status: string;
  tone: Tone;
  stage: string;
  stageProgress: string;
  policy: string;
  humanDecision: string;
  repo: ContractRepoId;
  detail: {
    changePacket: string;
    evidence: string;
    projectContext: string;
    decisionTrail: string;
  };
}

interface MobileRepoQueueItem {
  repo: RepoId;
  readiness: string;
  status: string;
  tone: Tone;
}

interface MobileProofQueueItem {
  id: string;
  contractId: string;
  status: string;
  coverage: string;
  tone: Tone;
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

const WORKSPACE_REPOS: RepoId[] = ['trialops-demo', 'billing-api', 'frontend-console'];

const REPO_CONTEXTS: Record<RepoId, RepoContext> = {
  'trialops-demo': {
    repo: 'trialops-demo',
    bound: 'yes',
    init: 'complete',
    docsIndexed: 12,
    readiness: 72,
    scanStatus: 'docs/context scan complete',
    testsStatus: 'tests detected',
    ciStatus: 'connected',
    ownersRulesStatus: 'AGENTS/rules present',
    proofSurfaceStatus: 'available',
    recommendedMode: 'local-only bounded execution',
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
    docsIndexed: 7,
    readiness: 58,
    scanStatus: 'docs/context scan partial',
    testsStatus: 'tests partial',
    ciStatus: 'connected',
    ownersRulesStatus: 'owners missing',
    proofSurfaceStatus: 'available with signoff',
    recommendedMode: 'human-signoff-required',
    checklist: [
      { label: 'Tests', value: 'partial', tone: 'amber' },
      { label: 'CI', value: 'connected', tone: 'pass' },
      { label: 'Owners/rules', value: 'missing', tone: 'amber' },
    ],
    runtimePolicy: 'human-signoff-required',
    runtimes: ['Codex CLI', 'manual', 'read-only review lane'],
  },
  'frontend-console': {
    repo: 'frontend-console',
    bound: 'no',
    init: 'pending',
    docsIndexed: 0,
    readiness: 41,
    scanStatus: 'context scan not started',
    testsStatus: 'unknown',
    ciStatus: 'unknown',
    ownersRulesStatus: 'rules pending',
    proofSurfaceStatus: 'not ready',
    recommendedMode: 'setup required',
    checklist: [
      { label: 'Tests', value: 'unknown', tone: 'amber' },
      { label: 'CI', value: 'unknown', tone: 'amber' },
      { label: 'Owners/rules', value: 'pending', tone: 'amber' },
    ],
    runtimePolicy: 'setup required',
    runtimes: ['manual setup', 'init required'],
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

const PROOF_FEED: ProofFeedItem[] = [
  {
    id: 'PF-0147',
    contractId: 'C-0147',
    repo: 'trialops-demo',
    proofStatus: 'Awaiting approval',
    decisionStatus: 'Gate ready',
    humanApproval: 'Pending human approval',
    linkedEvidence: '5 linked checks',
    criteriaCoverage: '5/5 criteria covered',
    summary: 'Repo-aware contract shell is ready for final operator decision.',
    tone: 'amber',
    changed: [
      'Contracts list and selected change packet now stay repo-aware.',
      'Project context remains outside the contract pipeline.',
      'Approval action is explicit before final acceptance.',
    ],
    unchanged: ['No backend, routing, auth, persistence, or real integration work.', 'No repo binding moved into the contract spine.'],
    verified: ['Acceptance criteria mapped to evidence receipts.', 'Touched scope stays limited to the demo shell.', 'Human approval remains pending.'],
    decisionTrail: ['contract.drafted', 'evidence.synced', 'verification.covered', 'approval.pending'],
    archiveLine: 'proof://mock/C-0147 · hash gr_pf_0147_a91c',
  },
  {
    id: 'PF-0148',
    contractId: 'C-0148',
    repo: 'trialops-demo',
    proofStatus: 'Evidence collecting',
    decisionStatus: 'No verdict yet',
    humanApproval: 'Not ready',
    linkedEvidence: '2 linked checks',
    criteriaCoverage: '2/5 criteria covered',
    summary: 'CSV export filter surface has partial receipts and open manual copy review.',
    tone: 'mauve',
    changed: ['Filter chip framing is being inspected.', 'Selection summary receipt exists.'],
    unchanged: ['No CSV generation, storage, or background jobs.'],
    verified: ['Scope note exists.', 'Manual naming check is still open.'],
    decisionTrail: ['contract.active', 'execution.running', 'evidence.partial'],
    archiveLine: 'proof://mock/C-0148 · draft hash pending',
  },
  {
    id: 'PF-0082',
    contractId: 'C-0082',
    repo: 'billing-api',
    proofStatus: 'Accepted',
    decisionStatus: 'Accepted',
    humanApproval: 'Approved',
    linkedEvidence: '6 linked checks',
    criteriaCoverage: 'proof archived',
    summary: 'Billing audit receipt copy was accepted and archived with unchanged runtime scope.',
    tone: 'pass',
    changed: ['Audit receipt wording and proof archive labels were tightened.'],
    unchanged: ['No billing runtime, database, or payment behavior changed.'],
    verified: ['Proof archive attached.', 'Scope and integrity checks passed.', 'Human approval recorded.'],
    decisionTrail: ['verification.covered', 'proof.archived', 'decision.accepted'],
    archiveLine: 'proof://mock/C-0082 · hash gr_pf_0082_f43b',
  },
  {
    id: 'PF-0091',
    contractId: 'C-0091',
    repo: 'billing-api',
    proofStatus: 'Blocked',
    decisionStatus: 'Integrity failure',
    humanApproval: 'Blocked by reviewer',
    linkedEvidence: '3 linked checks',
    criteriaCoverage: 'integrity failure',
    summary: 'Lead sync status change was blocked because the evidence bundle did not match the declared scope.',
    tone: 'block',
    changed: ['Sync status labels were proposed for review.'],
    unchanged: ['CRM sync and billing API behavior remain untouched.'],
    verified: ['Integrity lane failed.', 'Reviewer blocked the packet.', 'Rework required before proof can archive.'],
    decisionTrail: ['evidence.synced', 'integrity.failed', 'decision.blocked'],
    archiveLine: 'proof://mock/C-0091 · blocked hash gr_pf_0091_b7d0',
  },
];

const MOBILE_CONTRACT_QUEUE: MobileContractQueueItem[] = [
  {
    id: 'C-0147',
    title: 'Manual review gate',
    status: 'Active',
    tone: 'mauve',
    stage: 'Goal intake',
    stageProgress: '1/7',
    policy: 'local-only',
    humanDecision: 'pending',
    repo: 'trialops-demo',
    detail: {
      changePacket: 'Manual review gate is scoped for a local mocked demo update.',
      evidence: 'Acceptance criteria and touched-scope receipts are queued.',
      projectContext: 'trialops-demo has rules present and proof surface available.',
      decisionTrail: 'Goal intake opened. Human decision remains pending.',
    },
  },
  {
    id: 'C-0148',
    title: 'CSV export filters',
    status: 'Executing',
    tone: 'amber',
    stage: 'Execution evidence',
    stageProgress: '5/7',
    policy: 'local-only',
    humanDecision: 'not ready',
    repo: 'trialops-demo',
    detail: {
      changePacket: 'Filter chip copy is being inspected in the mock shell.',
      evidence: 'Two receipts exist; manual copy review is still open.',
      projectContext: 'trialops-demo remains the selected repo context.',
      decisionTrail: 'Execution running. Evidence collection is incomplete.',
    },
  },
  {
    id: 'C-0151',
    title: 'Pricing toggle cleanup',
    status: 'Approval',
    tone: 'amber',
    stage: 'Approval',
    stageProgress: '7/7',
    policy: 'local-only',
    humanDecision: 'pending',
    repo: 'trialops-demo',
    detail: {
      changePacket: 'Pricing copy cleanup is ready for human review.',
      evidence: 'Copy review note is attached to the proof packet.',
      projectContext: 'No billing behavior or integration is included.',
      decisionTrail: 'Proof is ready. Final approval is not one-tap on mobile.',
    },
  },
];

const MOBILE_REPO_QUEUE: MobileRepoQueueItem[] = [
  { repo: 'trialops-demo', readiness: '72/100', status: 'Ready', tone: 'pass' },
  { repo: 'billing-api', readiness: '58/100', status: 'Partial', tone: 'amber' },
  { repo: 'frontend-console', readiness: '41/100', status: 'Setup', tone: 'block' },
];

const MOBILE_PROOF_QUEUE: MobileProofQueueItem[] = [
  { id: 'PF-0147', contractId: 'C-0147', status: 'Awaiting approval', coverage: '5/5', tone: 'amber' },
  { id: 'PF-0148', contractId: 'C-0148', status: 'Evidence collecting', coverage: '2/5', tone: 'mauve' },
  { id: 'PF-0082', contractId: 'C-0082', status: 'Accepted', coverage: 'archived', tone: 'pass' },
  { id: 'PF-0091', contractId: 'C-0091', status: 'Blocked', coverage: 'integrity failure', tone: 'block' },
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

function getReadinessTone(score: number): Tone {
  if (score >= 70) return 'pass';
  if (score >= 50) return 'amber';
  return 'block';
}

function getCompactOwner(owner: string) {
  return owner.split('·')[0]?.trim() ?? owner;
}

function getShelfStatus(status: string) {
  if (status === 'Awaiting approval') return 'Approval';
  if (status === 'Needs rework') return 'Rework';
  if (status === 'Proof ready') return 'Proof';
  return status;
}

function getCompactReadinessSignal(value: string) {
  return value
    .replace(/^docs\/context scan /, '')
    .replace(/^context scan /, '')
    .replace(/ not started$/, ' not started');
}

function matchesQuery(values: string[], query: string) {
  const normalizedQuery = query.trim().toLowerCase();

  if (!normalizedQuery) return true;

  return values.some((value) => value.toLowerCase().includes(normalizedQuery));
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

function useMobileCompanionBreakpoint() {
  const getMatches = () => (typeof window === 'undefined' ? false : window.matchMedia('(max-width: 899px)').matches);
  const [isMobileCompanion, setIsMobileCompanion] = useState(getMatches);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(max-width: 899px)');
    const handleChange = () => setIsMobileCompanion(mediaQuery.matches);

    handleChange();

    if (typeof mediaQuery.addEventListener === 'function') {
      mediaQuery.addEventListener('change', handleChange);
      return () => mediaQuery.removeEventListener('change', handleChange);
    }

    mediaQuery.addListener(handleChange);
    return () => mediaQuery.removeListener(handleChange);
  }, []);

  return isMobileCompanion;
}

function MobileStat({ label, value, tone = 'mauve' }: { label: string; value: string; tone?: Tone }) {
  return (
    <div className={cx('mobile-stat', tone)}>
      <span>{label}</span>
      <b>{value}</b>
    </div>
  );
}

function MobileDetailSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <details className="mobile-detail" open>
      <summary>{title}</summary>
      <p>{children}</p>
    </details>
  );
}

function MobileContractsSurface({
  selectedContract,
  onSelectContract,
}: {
  selectedContract: MobileContractQueueItem;
  onSelectContract: (contractId: string) => void;
}) {
  return (
    <div className="mobile-surface" aria-label="Mobile contracts surface">
      <section className="mobile-card">
        <div className="mobile-card-kicker">Context summary</div>
        <div className="mobile-stat-grid">
          <MobileStat label="selected repo" value="trialops-demo" />
          <MobileStat label="active contracts" value="3 active contracts" tone="pass" />
          <MobileStat label="selected contract" value={selectedContract.id} />
          <MobileStat label="status" value={selectedContract.status} tone={selectedContract.tone} />
        </div>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-head">
          <div>
            <div className="mobile-card-kicker">Active Contracts queue</div>
            <h2>Tap to inspect</h2>
          </div>
          <span className="status-pill mauve">mock</span>
        </div>
        <div className="mobile-queue">
          {MOBILE_CONTRACT_QUEUE.map((contract) => (
            <button
              key={contract.id}
              className={cx('mobile-queue-row', selectedContract.id === contract.id && 'active')}
              type="button"
              onClick={() => onSelectContract(contract.id)}
            >
              <span>{contract.id} · {contract.title} · {contract.status}</span>
            </button>
          ))}
        </div>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-kicker">Selected Contract summary</div>
        <h2>{selectedContract.title}</h2>
        <dl className="mobile-facts">
          <div>
            <dt>repo</dt>
            <dd>{selectedContract.repo}</dd>
          </div>
          <div>
            <dt>current stage</dt>
            <dd>{selectedContract.stage}</dd>
          </div>
          <div>
            <dt>policy</dt>
            <dd>{selectedContract.policy}</dd>
          </div>
          <div>
            <dt>human decision</dt>
            <dd>{selectedContract.humanDecision}</dd>
          </div>
        </dl>
        <div className="mobile-stage-line">Stage: {selectedContract.stage} · {selectedContract.stageProgress}</div>
      </section>

      <section className="mobile-card mobile-detail-stack">
        <MobileDetailSection title="Change packet">{selectedContract.detail.changePacket}</MobileDetailSection>
        <MobileDetailSection title="Evidence">{selectedContract.detail.evidence}</MobileDetailSection>
        <MobileDetailSection title="Project context">{selectedContract.detail.projectContext}</MobileDetailSection>
        <MobileDetailSection title="Decision trail">{selectedContract.detail.decisionTrail}</MobileDetailSection>
      </section>
    </div>
  );
}

function MobileReadinessSurface({ selectedRepo, onSelectRepo }: { selectedRepo: RepoId; onSelectRepo: (repo: RepoId) => void }) {
  const context = REPO_CONTEXTS[selectedRepo];

  return (
    <div className="mobile-surface" aria-label="Mobile readiness surface">
      <section className="mobile-card">
        <div className="mobile-card-kicker">Readiness summary</div>
        <div className="mobile-stat-grid">
          <MobileStat label="repositories" value="3 repos" />
          <MobileStat label="average" value="57/100 avg" tone="amber" />
          <MobileStat label="setup" value="1 setup required" tone="block" />
        </div>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-head">
          <div>
            <div className="mobile-card-kicker">Repository queue</div>
            <h2>Repo readiness</h2>
          </div>
          <span className="status-pill mauve">review</span>
        </div>
        <div className="mobile-queue">
          {MOBILE_REPO_QUEUE.map((repo) => (
            <button
              key={repo.repo}
              className={cx('mobile-queue-row', selectedRepo === repo.repo && 'active')}
              type="button"
              onClick={() => onSelectRepo(repo.repo)}
            >
              <span>{repo.repo} · {repo.readiness} · {repo.status}</span>
            </button>
          ))}
        </div>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-kicker">Selected repo detail</div>
        <h2>{context.repo}</h2>
        <dl className="mobile-facts">
          <div>
            <dt>readiness</dt>
            <dd>{context.readiness}/100</dd>
          </div>
          <div>
            <dt>init</dt>
            <dd>{context.init}</dd>
          </div>
          <div>
            <dt>docs indexed</dt>
            <dd>{context.docsIndexed}</dd>
          </div>
          <div>
            <dt>tests</dt>
            <dd>{getCompactReadinessSignal(context.testsStatus)}</dd>
          </div>
          <div>
            <dt>CI</dt>
            <dd>{context.ciStatus}</dd>
          </div>
          <div>
            <dt>AGENTS/rules</dt>
            <dd>{context.ownersRulesStatus.replace('AGENTS/rules ', '')}</dd>
          </div>
          <div>
            <dt>proof surface</dt>
            <dd>{context.proofSurfaceStatus}</dd>
          </div>
          <div>
            <dt>recommended mode</dt>
            <dd>{context.recommendedMode}</dd>
          </div>
        </dl>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-kicker">Actions</div>
        <div className="mobile-action-row">
          <button className="primary-button mobile-safe-button" type="button">
            Analyze
          </button>
          <button className="ghost-button mobile-safe-button" type="button">
            Scan context
          </button>
          <button className="ghost-button mobile-safe-button secondary" type="button">
            Add repository
          </button>
        </div>
        <p className="mobile-action-note">Desktop setup recommended. Mock-only buttons do not connect or mutate repos.</p>
      </section>
    </div>
  );
}

function MobileProofSurface({
  selectedProof,
  onSelectProof,
}: {
  selectedProof: ProofFeedItem;
  onSelectProof: (proofId: string) => void;
}) {
  const proofDecisionReady = selectedProof.contractId === 'C-0147' && selectedProof.criteriaCoverage.startsWith('5/5');
  const archivedProof = selectedProof.proofStatus === 'Accepted';
  const decisionRestriction = archivedProof ? 'Read-only archived proof' : 'Criteria coverage incomplete';

  return (
    <div className="mobile-surface" aria-label="Mobile proof surface">
      <section className="mobile-card">
        <div className="mobile-card-head">
          <div>
            <div className="mobile-card-kicker">Proof queue</div>
            <h2>Decision review</h2>
          </div>
          <span className="status-pill amber">safe review</span>
        </div>
        <div className="mobile-queue">
          {MOBILE_PROOF_QUEUE.map((proof) => (
            <button
              key={proof.id}
              className={cx('mobile-queue-row', selectedProof.id === proof.id && 'active')}
              type="button"
              onClick={() => onSelectProof(proof.id)}
            >
              <span>{proof.contractId} · {proof.status} · {proof.coverage}</span>
            </button>
          ))}
        </div>
      </section>

      <section className="mobile-card">
        <div className="mobile-card-kicker">Selected proof summary</div>
        <h2>{selectedProof.contractId}</h2>
        <dl className="mobile-facts">
          <div>
            <dt>contract id</dt>
            <dd>{selectedProof.contractId}</dd>
          </div>
          <div>
            <dt>repo</dt>
            <dd>{selectedProof.repo}</dd>
          </div>
          <div>
            <dt>proof status</dt>
            <dd>{selectedProof.proofStatus}</dd>
          </div>
          <div>
            <dt>criteria coverage</dt>
            <dd>{selectedProof.criteriaCoverage}</dd>
          </div>
          <div>
            <dt>human approval</dt>
            <dd>{selectedProof.contractId === 'C-0147' ? 'Pending' : selectedProof.humanApproval}</dd>
          </div>
        </dl>
      </section>

      <section className="mobile-card mobile-detail-stack">
        <MobileDetailSection title="What changed">{selectedProof.changed[0]}</MobileDetailSection>
        <MobileDetailSection title="How verified">{selectedProof.verified[0]}</MobileDetailSection>
        <MobileDetailSection title="Evidence receipts">{selectedProof.linkedEvidence}</MobileDetailSection>
        <MobileDetailSection title="Decision trail">{selectedProof.decisionTrail.slice(0, 3).join(' · ')}</MobileDetailSection>
        <MobileDetailSection title="Proof archive / hash">{selectedProof.archiveLine}</MobileDetailSection>
      </section>

      <section className="mobile-card mobile-decision-card">
        <div className="mobile-card-kicker">Decision action</div>
        {proofDecisionReady ? (
          <>
            <button className="primary-button mobile-safe-button" type="button">
              Review decision
            </button>
            <p className="mobile-action-note">Safe review only. Final approval is not available as a one-tap mobile action.</p>
          </>
        ) : (
          <>
            <span className="status-pill amber">Decision unavailable</span>
            <p className="mobile-action-note">{decisionRestriction}</p>
          </>
        )}
      </section>
    </div>
  );
}

function MobileCompanionPreview() {
  const [mobileSurface, setMobileSurface] = useState<ActiveSurface>('proof');
  const [selectedMobileContractId, setSelectedMobileContractId] = useState('C-0147');
  const [selectedMobileRepo, setSelectedMobileRepo] = useState<RepoId>('trialops-demo');
  const [selectedMobileProofId, setSelectedMobileProofId] = useState('PF-0147');

  const selectedMobileContract = MOBILE_CONTRACT_QUEUE.find((contract) => contract.id === selectedMobileContractId) ?? MOBILE_CONTRACT_QUEUE[0];
  const selectedMobileProof = PROOF_FEED.find((proof) => proof.id === selectedMobileProofId) ?? PROOF_FEED[0];

  return (
    <main className="mobile-companion">
      <section className="mobile-hero">
        <div className="mobile-brand">Goalrail</div>
        <h1>Goalrail Mobile Companion</h1>
        <p>Focused review mode for contracts, readiness, and proof decisions.</p>
        <span>Open on desktop for the full operator console.</span>
      </section>

      <nav className="mobile-segmented" aria-label="Mobile companion surfaces">
        <button className={cx(mobileSurface === 'contracts' && 'active')} type="button" onClick={() => setMobileSurface('contracts')}>
          Contracts
        </button>
        <button className={cx(mobileSurface === 'readiness' && 'active')} type="button" onClick={() => setMobileSurface('readiness')}>
          Readiness
        </button>
        <button className={cx(mobileSurface === 'proof' && 'active')} type="button" onClick={() => setMobileSurface('proof')}>
          Proof
        </button>
      </nav>

      {mobileSurface === 'contracts' ? (
        <MobileContractsSurface selectedContract={selectedMobileContract} onSelectContract={setSelectedMobileContractId} />
      ) : mobileSurface === 'readiness' ? (
        <MobileReadinessSurface selectedRepo={selectedMobileRepo} onSelectRepo={setSelectedMobileRepo} />
      ) : (
        <MobileProofSurface selectedProof={selectedMobileProof} onSelectProof={setSelectedMobileProofId} />
      )}
    </main>
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

function DeliveryReadinessSurface({ selectedRepo, onSelectRepo }: { selectedRepo: RepoId; onSelectRepo: (repo: RepoId) => void }) {
  return (
    <main className="canvas surface-canvas">
      <section className="object surface-object">
        <div className="obj-head">
          <div>
            <div className="t">Delivery Readiness</div>
            <div className="object-title">Repo-level setup and operating mode</div>
          </div>
          <div className="tags">
            <span className="tag mauve">Workspace surface</span>
            <span className="tag">Mock only</span>
          </div>
        </div>

        <div className="obj-body">
          <div className="surface-intro">
            <div className="section-tagline">Delivery Readiness · repo/project-level readiness</div>
            <p className="detail-copy">
              This surface shows connected repositories, readiness signals, setup actions, and recommended operating mode. It is not a
              contract pipeline stage.
            </p>
          </div>

          <div className="repo-card-grid">
            {WORKSPACE_REPOS.map((repo) => {
              const context = REPO_CONTEXTS[repo];

              return (
                <button
                  key={repo}
                  className={cx('repo-readiness-card', selectedRepo === repo && 'active')}
                  type="button"
                  onClick={() => onSelectRepo(repo)}
                >
                  <div className="repo-card-head">
                    <div>
                      <div className="contract-id">{context.repo}</div>
                      <div className="repo-score">{context.readiness}/100 readiness</div>
                    </div>
                    <span className={cx('status-pill', getReadinessTone(context.readiness))}>{context.init}</span>
                  </div>

                  <dl className="key-grid readiness-card-grid">
                    <div>
                      <dt>Docs/context scan</dt>
                      <dd>{context.scanStatus}</dd>
                    </div>
                    <div>
                      <dt>Tests</dt>
                      <dd>{context.testsStatus}</dd>
                    </div>
                    <div>
                      <dt>CI</dt>
                      <dd>{context.ciStatus}</dd>
                    </div>
                    <div>
                      <dt>Owners/rules</dt>
                      <dd>{context.ownersRulesStatus}</dd>
                    </div>
                    <div>
                      <dt>Proof surface</dt>
                      <dd>{context.proofSurfaceStatus}</dd>
                    </div>
                    <div>
                      <dt>Mode</dt>
                      <dd>{context.recommendedMode}</dd>
                    </div>
                  </dl>
                </button>
              );
            })}

            <article className="repo-readiness-card add-repository-card">
              <div className="repo-card-head">
                <div>
                  <div className="contract-id">Add repository</div>
                  <div className="repo-score">Connect next repo</div>
                </div>
                <span className="status-pill mauve">Setup action</span>
              </div>
              <p className="detail-copy">Connect a repo, run init, scan context, and compute delivery readiness.</p>
              <button className="ghost-button" type="button">
                Add repository
              </button>
            </article>
          </div>
        </div>
      </section>
    </main>
  );
}

function DeliveryReadinessInspector({ selectedRepo }: { selectedRepo: RepoId }) {
  const context = REPO_CONTEXTS[selectedRepo];

  return (
    <aside className="sidepanel">
      <section className="panel-card">
        <div className="panel-head">
          <div className="t">Selected repo detail</div>
          <div className="id">{context.repo}</div>
        </div>

        <dl className="key-grid">
          <div>
            <dt>Readiness</dt>
            <dd>{context.readiness}/100</dd>
          </div>
          <div>
            <dt>Init</dt>
            <dd>{context.init}</dd>
          </div>
          <div>
            <dt>Docs indexed</dt>
            <dd>{context.docsIndexed}</dd>
          </div>
          <div>
            <dt>Recommended mode</dt>
            <dd>{context.recommendedMode}</dd>
          </div>
        </dl>

        <div className="readiness-block">
          <div className="row">
            <div className="label">Delivery readiness</div>
            <div className="val">{context.readiness}/100</div>
          </div>
          <div className={cx('bar', `bar-${getReadinessTone(context.readiness)}`)}>
            <i style={{ width: `${context.readiness}%` }} />
          </div>
        </div>

        <div className="checklist-block">
          <div className="detail-kicker">Readiness signals</div>
          <div className="check-row">
            <span>Docs/context scan</span>
            <span className="check-value mauve">{getCompactReadinessSignal(context.scanStatus)}</span>
          </div>
          {context.checklist.map((item) => (
            <div key={`${context.repo}-${item.label}`} className="check-row">
              <span>{item.label}</span>
              <span className={cx('check-value', item.tone)}>{item.value}</span>
            </div>
          ))}
          <div className="check-row">
            <span>Proof surface</span>
            <span className={cx('check-value', getReadinessTone(context.readiness))}>{context.proofSurfaceStatus}</span>
          </div>
        </div>

        <div className="detail-kicker">Mock actions</div>
        <div className="decision-actions">
          <button className="ghost-button" type="button">
            Analyze
          </button>
          <button className="ghost-button" type="button">
            Run init
          </button>
          <button className="ghost-button" type="button">
            Scan context
          </button>
        </div>
      </section>

      <section className="panel-card compact-card">
        <div className="panel-head">
          <div className="t">Surface boundary</div>
          <div className="id">Setup/readiness only</div>
        </div>
        <p className="panel-copy">
          Add repository belongs in Delivery Readiness. It does not open a real integration and it does not become a contract flow step.
        </p>
      </section>
    </aside>
  );
}

function DeliveryReadinessBottomPanel({ selectedRepo }: { selectedRepo: RepoId }) {
  const context = REPO_CONTEXTS[selectedRepo];

  return (
    <section className="bottompanel">
      <section className="panel-card activity-card">
        <div className="panel-head">
          <div className="t">Readiness activity</div>
          <div className="id">Mock setup events</div>
        </div>
        <div className="activity-list">
          {[
            ['09:31:02', 'repo.selected', `${context.repo} selected for readiness detail`, 'mauve'],
            ['09:31:12', 'context.scan', context.scanStatus, getReadinessTone(context.readiness)],
            ['09:31:22', 'mode.recommended', context.recommendedMode, getReadinessTone(context.readiness)],
          ].map(([ts, kind, note, tone]) => (
            <div key={`${ts}-${kind}`} className="activity-row">
              <div className="activity-ts">{ts}</div>
              <div className="activity-body">
                <div className="activity-kind">{kind}</div>
                <div className="activity-note">{note}</div>
              </div>
              <div className={cx('status-pill', tone as Tone)}>{tone === 'pass' ? 'pass' : tone === 'block' ? 'setup' : 'review'}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="panel-card control-card">
        <div className="panel-head">
          <div className="t">Surface controls</div>
          <div className="id">No real integrations</div>
        </div>
        <div className="control-copy">
          Analyze, Run init, Scan context, and Add repository are mock setup actions. They do not call a backend or mutate persistent state.
        </div>
        <div className="control-meta">
          <span>Selected repo: {context.repo}</span>
          <span>Readiness: {context.readiness}/100</span>
        </div>
      </section>
    </section>
  );
}

function ProofFeedSurface({ selectedProofId, onSelectProof }: { selectedProofId: string; onSelectProof: (proofId: string) => void }) {
  return (
    <main className="canvas surface-canvas">
      <section className="object surface-object">
        <div className="obj-head">
          <div>
            <div className="t">Proof Feed</div>
            <div className="object-title">Cross-contract evidence and decisions</div>
          </div>
          <div className="tags">
            <span className="tag pass">All repos default</span>
            <span className="tag">Not a chat log</span>
          </div>
        </div>

        <div className="obj-body">
          <div className="surface-intro">
            <div className="section-tagline">Proof Feed · cross-contract, cross-repo overview</div>
            <p className="detail-copy">
              Default scope is all repos. Repo scope chips are static mock controls so the surface reads as workspace-level proof control.
            </p>
            <div className="chip-row filter-row">
              <span className="tag pass">All repos</span>
              <span className="tag">trialops-demo</span>
              <span className="tag">billing-api</span>
              <span className="tag">frontend-console</span>
            </div>
          </div>

          <div className="proof-feed-list">
            {PROOF_FEED.map((item) => (
              <button
                key={item.id}
                className={cx('proof-feed-row', selectedProofId === item.id && 'active')}
                type="button"
                onClick={() => onSelectProof(item.id)}
              >
                <div className="proof-row-head">
                  <div className="proof-row-title">
                    <span className="contract-id">{item.contractId}</span>
                    <span className="repo-badge">{item.repo}</span>
                  </div>
                  <span className={cx('status-pill', item.tone)}>{item.proofStatus}</span>
                </div>
                <div className="proof-summary">{item.summary}</div>
                <dl className="key-grid proof-meta-grid">
                  <div>
                    <dt>Decision</dt>
                    <dd>{item.decisionStatus}</dd>
                  </div>
                  <div>
                    <dt>Human approval</dt>
                    <dd>{item.humanApproval}</dd>
                  </div>
                  <div>
                    <dt>Evidence</dt>
                    <dd>{item.linkedEvidence}</dd>
                  </div>
                  <div>
                    <dt>Coverage</dt>
                    <dd>{item.criteriaCoverage}</dd>
                  </div>
                </dl>
              </button>
            ))}
          </div>
        </div>
      </section>
    </main>
  );
}

function ProofFeedInspector({ selectedProof }: { selectedProof: ProofFeedItem }) {
  return (
    <aside className="sidepanel">
      <section className="panel-card proof-detail-card">
        <div className="panel-head">
          <div className="t">Selected proof detail</div>
          <div className="id">{selectedProof.contractId} · {selectedProof.repo}</div>
        </div>
        <ListBlock title="What changed" items={selectedProof.changed.slice(0, 2)} />
        <ListBlock title="What did not change" items={selectedProof.unchanged.slice(0, 2)} />
        <ListBlock title="How verified" items={selectedProof.verified.slice(0, 2)} />
        <ListBlock title="Decision trail" items={selectedProof.decisionTrail.slice(0, 2)} />
        <div className="panel-note proof-archive-line">
          <b>Proof archive / hash</b>
          <br />
          {selectedProof.archiveLine}
        </div>
      </section>

      <section className="panel-card compact-card">
        <div className="panel-head">
          <div className="t">Feed scope</div>
          <div className="id">All repos by default</div>
        </div>
        <p className="panel-copy">
          This feed is a workspace-level overview across contracts and repos. It is not tied to the current repo selector.
        </p>
      </section>
    </aside>
  );
}

function ProofFeedBottomPanel({ selectedProof }: { selectedProof: ProofFeedItem }) {
  return (
    <section className="bottompanel">
      <section className="panel-card activity-card">
        <div className="panel-head">
          <div className="t">Proof feed activity</div>
          <div className="id">Evidence and decisions only</div>
        </div>
        <div className="activity-list">
          {selectedProof.decisionTrail.map((entry, index) => (
            <div key={`${selectedProof.id}-${entry}`} className="activity-row">
              <div className="activity-ts">10:{String(12 + index).padStart(2, '0')}:04</div>
              <div className="activity-body">
                <div className="activity-kind">{entry}</div>
                <div className="activity-note">{selectedProof.contractId} · {selectedProof.repo}</div>
              </div>
              <div className={cx('status-pill', selectedProof.tone)}>{selectedProof.tone === 'block' ? 'block' : selectedProof.tone === 'pass' ? 'pass' : 'event'}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="panel-card control-card">
        <div className="panel-head">
          <div className="t">Feed controls</div>
          <div className="id">Static scope</div>
        </div>
        <div className="control-copy">
          Repo scope chips are mock-only. Status triage lives in the left Proof Queue so the feed stays cross-contract and cross-repo.
        </div>
        <div className="control-meta">
          <span>Selected proof: {selectedProof.contractId}</span>
          <span>Default scope: All repos</span>
        </div>
      </section>
    </section>
  );
}

function DesktopConsole() {
  const [activeSurface, setActiveSurface] = useState<ActiveSurface>('contracts');
  const [repoFilter, setRepoFilter] = useState<RepoFilter>('trialops-demo');
  const [repoSelectorOpen, setRepoSelectorOpen] = useState(false);
  const [contractSearch, setContractSearch] = useState('');
  const [repoSearch, setRepoSearch] = useState('');
  const [proofSearch, setProofSearch] = useState('');
  const [selectedContractId, setSelectedContractId] = useState('C-0147');
  const [selectedReadinessRepo, setSelectedReadinessRepo] = useState<RepoId>('trialops-demo');
  const [selectedProofId, setSelectedProofId] = useState(PROOF_FEED[0].id);
  const [contractSteps, setContractSteps] = useState<Record<string, StepIndex>>(INITIAL_STEPS);
  const [approvalStates, setApprovalStates] = useState<Record<string, ApprovalState>>(INITIAL_APPROVALS);
  const [visibleClarifications, setVisibleClarifications] = useState(0);
  const [visibleEvidence, setVisibleEvidence] = useState(0);
  const [visibleVerification, setVisibleVerification] = useState(0);

  const repoScopedContracts = useMemo(() => {
    return repoFilter === 'all' ? CONTRACTS : CONTRACTS.filter((contract) => contract.repo === repoFilter);
  }, [repoFilter]);

  const visibleContracts = useMemo(() => {
    return repoScopedContracts.filter((contract) =>
      matchesQuery([contract.id, contract.title, contract.repo, contract.owner, contract.summary, getStatus(contractSteps[contract.id] ?? contract.defaultStep, approvalStates[contract.id] ?? 'pending')], contractSearch),
    );
  }, [approvalStates, contractSearch, contractSteps, repoScopedContracts]);

  const visibleRepos = useMemo(() => {
    return WORKSPACE_REPOS.filter((repo) => {
      const context = REPO_CONTEXTS[repo];
      return matchesQuery([context.repo, context.init, context.scanStatus, context.testsStatus, context.ciStatus, context.ownersRulesStatus, context.recommendedMode], repoSearch);
    });
  }, [repoSearch]);

  const visibleProofs = useMemo(() => {
    return PROOF_FEED.filter((item) =>
      matchesQuery([item.contractId, item.repo, item.proofStatus, item.decisionStatus, item.humanApproval, item.criteriaCoverage, item.summary], proofSearch),
    );
  }, [proofSearch]);

  useEffect(() => {
    if (!repoScopedContracts.some((contract) => contract.id === selectedContractId)) {
      setSelectedContractId(repoScopedContracts[0]?.id ?? CONTRACTS[0].id);
    }
  }, [repoScopedContracts, selectedContractId]);

  const selectedContract = useMemo(() => {
    return CONTRACTS.find((contract) => contract.id === selectedContractId) ?? CONTRACTS[0];
  }, [selectedContractId]);

  const step = contractSteps[selectedContract.id] ?? selectedContract.defaultStep;
  const approval = approvalStates[selectedContract.id] ?? 'pending';
  const selectedStatus = getStatus(step, approval);
  const projectContext = REPO_CONTEXTS[selectedContract.repo];
  const meters = getMeters(step, approval);
  const activity = useMemo(() => getActivity(selectedContract, step, approval), [selectedContract, step, approval]);
  const selectedProof = useMemo(() => PROOF_FEED.find((item) => item.id === selectedProofId) ?? PROOF_FEED[0], [selectedProofId]);
  const averageReadiness = Math.round(WORKSPACE_REPOS.reduce((sum, repo) => sum + REPO_CONTEXTS[repo].readiness, 0) / WORKSPACE_REPOS.length);
  const acceptedProofs = PROOF_FEED.filter((item) => item.proofStatus === 'Accepted').length;
  const blockedProofs = PROOF_FEED.filter((item) => item.proofStatus === 'Blocked').length;
  const topbarMeters =
    activeSurface === 'contracts'
      ? [
          { tone: 'amber' as Tone, label: 'Contract', value: meters.contract.label, percent: meters.contract.percent },
          { tone: 'mauve' as Tone, label: 'Execution', value: meters.execution.label, percent: meters.execution.percent },
          { tone: 'pass' as Tone, label: 'Proof', value: meters.proof.label, percent: meters.proof.percent },
        ]
      : activeSurface === 'readiness'
        ? [
            { tone: 'mauve' as Tone, label: 'Workspace', value: `${WORKSPACE_REPOS.length} repos`, percent: 100 },
            { tone: 'amber' as Tone, label: 'Readiness', value: `${averageReadiness}/100 avg`, percent: averageReadiness },
            { tone: 'pass' as Tone, label: 'Setup', value: 'Mock actions only', percent: 72 },
          ]
        : [
            { tone: 'mauve' as Tone, label: 'Proof feed', value: 'All repos', percent: 100 },
            { tone: 'amber' as Tone, label: 'Awaiting', value: '2 active', percent: 50 },
            { tone: 'pass' as Tone, label: 'Decisions', value: `${acceptedProofs} accepted · ${blockedProofs} blocked`, percent: 70 },
          ];

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

  const selectedRepoOption = REPO_OPTIONS.find((option) => option.value === repoFilter) ?? REPO_OPTIONS[0];

  const handleRepoFilterSelect = (nextRepoFilter: RepoFilter) => {
    setRepoFilter(nextRepoFilter);
    setRepoSelectorOpen(false);
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

  const activeSurfaceLabel = activeSurface === 'contracts' ? 'Contracts' : activeSurface === 'readiness' ? 'Delivery Readiness' : 'Proof Feed';
  const topbarStateLabel = activeSurface === 'contracts' ? 'Status' : activeSurface === 'readiness' ? 'Repo' : 'Scope';
  const topbarStateValue = activeSurface === 'contracts' ? selectedStatus : activeSurface === 'readiness' ? selectedReadinessRepo : 'All repos';

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
            {topbarMeters.map((meter) => (
              <div key={meter.label} className={cx('meter', meter.tone)}>
                <div className="row">
                  <div className="label">{meter.label}</div>
                  <div className="val">{meter.value}</div>
                </div>
                <div className="bar">
                  <i style={{ width: `${meter.percent}%` }} />
                </div>
              </div>
            ))}
          </div>

          <div className="topbar-state">
            <div className="state-chip">
              <span className="k">Surface</span>
              <span className="v" title={activeSurfaceLabel}>
                {activeSurfaceLabel}
              </span>
            </div>
            <div className="state-chip">
              <span className="k">{topbarStateLabel}</span>
              <span className="v" title={topbarStateValue}>
                {topbarStateValue}
              </span>
            </div>
          </div>
        </header>

        <aside className="rail">
          <div className="rail-main">
            <div className="group-label">Workspace</div>
            <div className="surface-switcher" aria-label="Workspace surfaces">
              <button
                className={cx('surface-switch', activeSurface === 'contracts' && 'active')}
                type="button"
                onClick={() => setActiveSurface('contracts')}
              >
                Contracts
              </button>
              <button
                className={cx('surface-switch', activeSurface === 'readiness' && 'active')}
                type="button"
                onClick={() => setActiveSurface('readiness')}
              >
                Delivery Readiness
              </button>
              <button
                className={cx('surface-switch', activeSurface === 'proof' && 'active')}
                type="button"
                onClick={() => setActiveSurface('proof')}
              >
                Proof Feed
              </button>
            </div>

            <div className="group-label">Surface context</div>
            <div className="rail-section surface-context">
              <div className="surface-context-title">
                {activeSurface === 'contracts' ? 'Contracts' : activeSurface === 'readiness' ? 'Delivery Readiness' : 'Proof Feed'}
              </div>
              <div className="rail-note">
                {activeSurface === 'contracts'
                  ? repoFilter === 'all'
                    ? 'All-repo active work'
                    : 'Repo-scoped active work'
                  : activeSurface === 'readiness'
                    ? 'Repo-level setup'
                    : 'Cross-repo evidence'}
              </div>

              {activeSurface === 'contracts' ? (
                <div className="surface-control">
                  <div className="select-wrap">
                    <span className="select-label">Repo selector</span>
                    <div className="repo-select">
                      <button
                        className={cx('repo-select-trigger', repoSelectorOpen && 'open')}
                        type="button"
                        aria-haspopup="listbox"
                        aria-expanded={repoSelectorOpen}
                        onClick={() => setRepoSelectorOpen((open) => !open)}
                      >
                        <span>{selectedRepoOption.label}</span>
                        <i aria-hidden="true" />
                      </button>
                      {repoSelectorOpen ? (
                        <div className="repo-select-menu" role="listbox" aria-label="Repo selector">
                          {REPO_OPTIONS.map((option) => (
                            <button
                              key={option.value}
                              className={cx('repo-select-option', option.value === repoFilter && 'active')}
                              type="button"
                              role="option"
                              aria-selected={option.value === repoFilter}
                              onClick={() => handleRepoFilterSelect(option.value)}
                            >
                              <span>{option.label}</span>
                              <b>{option.value === 'all' ? 'all' : CONTRACTS.filter((contract) => contract.repo === option.value).length}</b>
                            </button>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  </div>
                  <div className="repo-hint-list">
                    <span>trialops-demo · 3 contracts</span>
                    <span>billing-api · 2 contracts</span>
                    <span>All repos available</span>
                  </div>
                </div>
              ) : null}
            </div>

            {activeSurface === 'contracts' ? (
              <>
                <div className="group-label">Active contracts</div>
                <div className="shelf-tools" aria-label="Contract shelf tools">
                  <input aria-label="Search contracts" placeholder="Search contracts" value={contractSearch} onChange={(event) => setContractSearch(event.target.value)} />
                  <div className="filter-chip-row" aria-label="Contract filters">
                    <span>Active</span>
                    <span>Executing</span>
                    <span>Approval</span>
                  </div>
                </div>
                <div className="contract-list" aria-label="Active contracts">
                  {visibleContracts.map((contract) => {
                    const contractStep = contractSteps[contract.id] ?? contract.defaultStep;
                    const contractApproval = approvalStates[contract.id] ?? 'pending';
                    const status = getStatus(contractStep, contractApproval);
                    const isSelected = contract.id === selectedContract.id;

                    return (
                      <button
                        key={contract.id}
                        className={cx('contract-row', isSelected && 'active', !isSelected && 'compact')}
                        type="button"
                        onClick={() => setSelectedContractId(contract.id)}
                      >
                        <div className="contract-row-top">
                          <span className="contract-id">{contract.id}</span>
                          <span className={cx('status-pill', getStatusTone(status))}>{getShelfStatus(status)}</span>
                        </div>
                        <div className="contract-title">{contract.title}</div>
                        {isSelected ? <div className="contract-summary">{contract.summary}</div> : null}
                        <div className="contract-row-meta">
                          {isSelected ? (
                            <>
                              <span className="repo-badge">{contract.repo}</span>
                              <span className="contract-owner">{contract.owner}</span>
                            </>
                          ) : (
                            <span className="contract-compact-meta">
                              {contract.repo} · {getCompactOwner(contract.owner)}
                            </span>
                          )}
                        </div>
                      </button>
                    );
                  })}
                  {visibleContracts.length === 0 ? <div className="rail-empty">No contracts match</div> : null}
                </div>
              </>
            ) : activeSurface === 'readiness' ? (
              <>
                <div className="group-label">Repositories</div>
                <div className="shelf-tools" aria-label="Repository shelf tools">
                  <input aria-label="Search repos" placeholder="Search repos" value={repoSearch} onChange={(event) => setRepoSearch(event.target.value)} />
                  <div className="filter-chip-row" aria-label="Repository filters">
                    <span>Ready</span>
                    <span>Partial</span>
                    <span>Setup</span>
                  </div>
                </div>
                <div className="surface-mini-list" role="group" aria-label="Repo readiness shelf">
                  {visibleRepos.map((repo) => (
                    <button
                      key={repo}
                      className={cx('surface-mini-row', selectedReadinessRepo === repo && 'active')}
                      type="button"
                      onClick={() => setSelectedReadinessRepo(repo)}
                    >
                      <span>{repo}</span>
                      <b>{REPO_CONTEXTS[repo].readiness}/100</b>
                    </button>
                  ))}
                  {visibleRepos.length === 0 ? <div className="rail-empty">No repos match</div> : null}
                </div>
              </>
            ) : (
              <>
                <div className="group-label">Proof queue</div>
                <div className="shelf-tools" aria-label="Proof shelf tools">
                  <input aria-label="Search proofs" placeholder="Search proofs" value={proofSearch} onChange={(event) => setProofSearch(event.target.value)} />
                  <div className="filter-chip-row" aria-label="Proof filters">
                    <span>Awaiting</span>
                    <span>Accepted</span>
                    <span>Blocked</span>
                  </div>
                </div>
                <div className="surface-mini-list" role="group" aria-label="Proof queue shelf">
                  {visibleProofs.map((item) => (
                    <button
                      key={item.id}
                      className={cx('surface-mini-row', selectedProof.id === item.id && 'active')}
                      type="button"
                      onClick={() => setSelectedProofId(item.id)}
                    >
                      <span>{item.contractId}</span>
                      <b>{item.proofStatus}</b>
                    </button>
                  ))}
                  {visibleProofs.length === 0 ? <div className="rail-empty">No proofs match</div> : null}
                </div>
              </>
            )}
          </div>

          <div className="case">
            <div className="k">Mode</div>
            <div className="v">Workspace surfaces demo</div>
            <div className="sub">Local mocked state only · no backend · no routing</div>
          </div>
        </aside>

        {activeSurface === 'contracts' ? (
          <>
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

            <div className="selection-strip">
              <span>{selectedContract.id}</span>
              <span>{STAGES[step].name}</span>
              <span>{selectedStatus}</span>
            </div>
          </section>

          <section className="panel-card inspector-card">
            <div className="panel-head">
              <div className="t">Ambiguity inspector</div>
              <div className="id">Top inputs · {selectedContract.clarifications.length} total</div>
            </div>
            <div className="inspector-list">
              {selectedContract.clarifications.slice(0, 3).map((card, index) => {
                const resolved = step > 1 || index < visibleClarifications;
                return (
                  <div key={`${selectedContract.id}-inspector-${card.ref}`} className="inspector-row">
                    <div>
                      <div className="inspector-term">{card.ref}</div>
                      <div className="inspector-note">{card.answer}</div>
                    </div>
                    <div className={cx('status-pill', resolved ? 'pass' : 'amber')}>{resolved ? 'Resolved' : 'Open'}</div>
                  </div>
                );
              })}
            </div>
            {selectedContract.clarifications.length > 3 ? (
              <div className="inspector-foot">{selectedContract.clarifications.length - 3} more inputs remain in the contract detail.</div>
            ) : null}
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
          </>
        ) : activeSurface === 'readiness' ? (
          <>
            <DeliveryReadinessSurface selectedRepo={selectedReadinessRepo} onSelectRepo={setSelectedReadinessRepo} />
            <DeliveryReadinessInspector selectedRepo={selectedReadinessRepo} />
            <DeliveryReadinessBottomPanel selectedRepo={selectedReadinessRepo} />
          </>
        ) : (
          <>
            <ProofFeedSurface selectedProofId={selectedProof.id} onSelectProof={setSelectedProofId} />
            <ProofFeedInspector selectedProof={selectedProof} />
            <ProofFeedBottomPanel selectedProof={selectedProof} />
          </>
        )}
      </div>
    </div>
  );
}

export default function App() {
  const isMobileCompanion = useMobileCompanionBreakpoint();

  return isMobileCompanion ? <MobileCompanionPreview /> : <DesktopConsole />;
}

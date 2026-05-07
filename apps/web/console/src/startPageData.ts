export interface StartQuestion {
  id: string;
  label: string;
  answerId: string;
}

export interface StartAnswer {
  id: string;
  title: string;
  eyebrow: string;
  body: string[];
  sources: string[];
  nextQuestions: string[];
}

export interface StartArtifact {
  title: string;
  body: string;
  signal: string;
}

// Derived from docs/reference/start-assistant/quick-questions.json and
// docs/reference/start-assistant/static-answers.md for the Stage 2 static /start page.
export const startQuestions: StartQuestion[] = [
  {
    id: 'what-is-goalrail',
    label: 'What is Goalrail?',
    answerId: 'what-is-goalrail',
  },
  {
    id: 'repo-ready',
    label: 'Is my repo ready for coding agents?',
    answerId: 'repo-readiness',
  },
  {
    id: 'contract-first',
    label: 'What is contract-first execution?',
    answerId: 'contract-first-execution',
  },
  {
    id: 'proof-before-approval',
    label: 'What does proof before approval mean?',
    answerId: 'proof-before-approval',
  },
  {
    id: 'not-ai-ide',
    label: 'How is Goalrail different from an AI IDE?',
    answerId: 'different-from-ai-ide',
  },
  {
    id: 'pilot-fit-check',
    label: 'What would a pilot fit check look like?',
    answerId: 'pilot-fit-check',
  },
  {
    id: 'ai-delivery-drift',
    label: 'What is AI delivery drift?',
    answerId: 'ai-delivery-drift',
  },
  {
    id: 'ai-review',
    label: 'How should a team review AI-generated changes?',
    answerId: 'review-ai-generated-changes',
  },
];

export const startAnswers: Record<string, StartAnswer> = {
  'what-is-goalrail': {
    id: 'what-is-goalrail',
    title: 'Goalrail is a control layer for AI-assisted software delivery.',
    eyebrow: 'Definition',
    body: [
      'It helps teams move from a business goal to a verified code change through goal intake, clarification, contract, bounded execution, checks, proof, and human approval.',
      'It is not an AI IDE, not a Jira replacement, and not a generic agent platform.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['What is contract-first execution?', 'What does proof before approval mean?'],
  },
  'repo-readiness': {
    id: 'repo-readiness',
    title: 'A repo is not ready for coding agents just because it builds.',
    eyebrow: 'Repo readiness',
    body: [
      'Repo readiness means the repository exposes enough working signals for an agent to operate safely.',
      'The team should be able to see how to run it, which checks matter, what not to touch, where ownership lives, how to prove behavior stayed intact, and how to recover when a change goes wrong.',
    ],
    sources: ['START_ASSISTANT_SECURITY_AND_PRIVACY.md', 'static-answers.md'],
    nextQuestions: ['How should a team review AI-generated changes?', 'What is AI delivery drift?'],
  },
  'contract-first-execution': {
    id: 'contract-first-execution',
    title: 'Contract-first execution means the agent does not start from a free prompt.',
    eyebrow: 'Contract',
    body: [
      'The work is first bounded by a contract: goal, scope, non-goals, affected areas, required checks, expected artifacts, and proof criteria.',
      'The goal is not more process. The goal is fewer hidden decisions during execution.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['What does proof before approval mean?', 'How is Goalrail different from an AI IDE?'],
  },
  'proof-before-approval': {
    id: 'proof-before-approval',
    title: 'Output is not proof.',
    eyebrow: 'Proof',
    body: [
      'Proof before approval means reviewers should not accept AI-generated work only because the diff looks clean or the agent summary sounds confident.',
      'They should compare contract, diff, checks, artifacts, and remaining risk before approving the change.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['How should a team review AI-generated changes?', 'Is my repo ready for coding agents?'],
  },
  'different-from-ai-ide': {
    id: 'different-from-ai-ide',
    title: 'Goalrail is not an AI IDE.',
    eyebrow: 'Positioning',
    body: [
      'AI IDEs help generate or edit code. Goalrail focuses on the control layer around AI-assisted delivery: intent, scope, contract, checks, proof, and human approval.',
      'It works around the tools a team already uses.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['What is Goalrail?', 'What would a pilot fit check look like?'],
  },
  'pilot-fit-check': {
    id: 'pilot-fit-check',
    title: 'A pilot fit check is a lightweight conversation around one real workflow.',
    eyebrow: 'Pilot fit',
    body: [
      'The best fit is a small or mid-sized team already using AI coding tools and starting to feel review, context, scope, or proof problems.',
      'The first useful pilot shape is one visible task-to-proof loop for one team, repo, or workflow.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['Is my repo ready for coding agents?', 'What is AI delivery drift?'],
  },
  'ai-delivery-drift': {
    id: 'ai-delivery-drift',
    title: 'AI delivery drift is when work moves away from the original intent.',
    eyebrow: 'Drift',
    body: [
      'AI-assisted work can drift away from the original scope, architecture, or proof expectations even when the code looks clean.',
      'The team then has to reconstruct why the change was made, whether it stayed in bounds, and what was actually verified.',
    ],
    sources: ['GOALRAIL_GLOBAL_START_ASSISTANT.md', 'static-answers.md'],
    nextQuestions: ['What is contract-first execution?', 'What does proof before approval mean?'],
  },
  'review-ai-generated-changes': {
    id: 'review-ai-generated-changes',
    title: 'AI review should not stop at the diff.',
    eyebrow: 'Review',
    body: [
      'A reviewer should ask what contract was executed, which checks passed, what artifacts were produced, what remains unverified, and which risk is being accepted.',
      'That keeps human approval in the loop, but makes the decision less vague.',
    ],
    sources: ['START_ASSISTANT_SECURITY_AND_PRIVACY.md', 'static-answers.md'],
    nextQuestions: ['What does proof before approval mean?', 'Is my repo ready for coding agents?'],
  },
};

export const startArtifacts: StartArtifact[] = [
  {
    title: 'Contract-first execution',
    body: 'Goal, scope, non-goals, affected areas, checks, artifacts, and proof criteria before execution starts.',
    signal: 'Bounded work packet',
  },
  {
    title: 'Proof before approval',
    body: 'Review connects the contract, diff, checks, artifacts, and remaining risk before a human approves.',
    signal: 'Evidence over summary',
  },
  {
    title: 'Repo readiness',
    body: 'The repo exposes run commands, safety boundaries, ownership, verification signals, and recovery paths.',
    signal: 'Operate safely',
  },
  {
    title: 'AI delivery drift',
    body: 'A visible way to keep AI-assisted work aligned with intent, scope, architecture, and proof expectations.',
    signal: 'Control drift',
  },
];

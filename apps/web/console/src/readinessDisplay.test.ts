import { describe, expect, it } from 'vitest';

import { projectReadinessDisplay, sortReadinessItems } from './readinessDisplay';
import type { QualificationFeedItem } from './qualificationFeedClient';

function item(overrides: Partial<QualificationFeedItem> = {}): QualificationFeedItem {
  return {
    intake_id: 'intake-1',
    goal_id: 'goal-1',
    organization_id: 'org-1',
    project_id: 'project-1',
    repo_binding_id: 'repo-1',
    repository_full_name: 'heurema/goalrail',
    title: 'Display readiness',
    lane: 'qualification',
    intake_state: 'received',
    goal_state: 'created',
    readiness: {
      ready: false,
      reason_codes: ['missing_scope_hint'],
      source: 'goal_snapshot',
    },
    next_action: {
      kind: 'continue_goal',
      available: true,
      blocking: false,
    },
    created_at: '2026-05-08T10:00:00Z',
    ...overrides,
  };
}

describe('readinessDisplay', () => {
  it('derives each primary readiness status', () => {
    expect(projectReadinessDisplay(item({
      lane: 'clarification',
      goal_state: 'needs_clarification',
      open_clarification_request: {
        id: 'clarification-1',
        state: 'open',
        questions: [{
          id: 'question-1',
          text: 'What scope?',
          why_needed: 'Scope is needed.',
          answer_type: 'text',
          maps_to: 'goal.scope_hint',
        }],
      },
      next_action: { kind: 'answer_clarification', available: true, blocking: true },
    }))).toMatchObject({ primaryStatusKey: 'needs_answer', displayPriority: 1, tone: 'amber' });

    expect(projectReadinessDisplay(item({
      goal_state: 'ready_for_contract_seed',
      readiness: { ready: true, reason_codes: [], source: 'goal_snapshot' },
      next_action: { kind: 'draft_contract', available: true, blocking: false },
    }))).toMatchObject({ primaryStatusKey: 'ready_for_contract', displayPriority: 2, tone: 'pass' });

    expect(projectReadinessDisplay(item())).toMatchObject({
      primaryStatusKey: 'needs_qualification',
      displayPriority: 3,
      tone: 'mauve',
    });

    expect(projectReadinessDisplay(item({
      lane: 'contract',
      goal_state: 'ready_for_contract_seed',
      readiness: { ready: true, reason_codes: [], source: 'goal_snapshot' },
      linked_contract: { id: 'contract-1', state: 'draft' },
      next_action: { kind: 'update_contract', available: false, blocking: false },
    }))).toMatchObject({ primaryStatusKey: 'contract_linked', displayPriority: 4, tone: 'pass' });

    expect(projectReadinessDisplay(item({
      lane: 'blocked',
      goal_state: 'rejected',
      next_action: { kind: 'blocked', available: false, blocking: true },
    }))).toMatchObject({ primaryStatusKey: 'blocked', displayPriority: 5, tone: 'muted' });
  });

  it('sorts display items by D-0091 priority before timestamp', () => {
    const sorted = sortReadinessItems([
      item({ goal_id: 'blocked', lane: 'blocked', goal_state: 'rejected', next_action: { kind: 'blocked', available: false, blocking: true } }),
      item({ goal_id: 'linked', lane: 'contract', linked_contract: { id: 'contract-1', state: 'approved' } }),
      item({ goal_id: 'needs-qualification', created_at: '2026-05-08T12:00:00Z' }),
      item({
        goal_id: 'needs-answer',
        lane: 'clarification',
        goal_state: 'needs_clarification',
        open_clarification_request: {
          id: 'clarification-1',
          state: 'open',
          questions: [{
            id: 'question-1',
            text: 'Question?',
            why_needed: 'Needed.',
            answer_type: 'text',
            maps_to: 'goal.scope_hint',
          }],
        },
        next_action: { kind: 'answer_clarification', available: true, blocking: true },
      }),
      item({
        goal_id: 'ready',
        goal_state: 'ready_for_contract_seed',
        readiness: { ready: true, reason_codes: [], source: 'goal_snapshot' },
        next_action: { kind: 'draft_contract', available: true, blocking: false },
      }),
    ]);

    expect(sorted.map((entry) => entry.goal_id)).toEqual([
      'needs-answer',
      'ready',
      'needs-qualification',
      'linked',
      'blocked',
    ]);
  });
});

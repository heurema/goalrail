import type { QualificationFeedItem, QualificationLane } from './qualificationFeedClient';

export type ReadinessPrimaryStatusKey =
  | 'needs_answer'
  | 'ready_for_contract'
  | 'needs_qualification'
  | 'contract_linked'
  | 'blocked';

export type ReadinessTone = 'amber' | 'pass' | 'mauve' | 'muted';

export interface ReadinessDisplay {
  primaryStatusKey: ReadinessPrimaryStatusKey;
  displayPriority: number;
  tone: ReadinessTone;
}

export const READINESS_STATUS_PRIORITY: Record<ReadinessPrimaryStatusKey, number> = {
  needs_answer: 1,
  ready_for_contract: 2,
  needs_qualification: 3,
  contract_linked: 4,
  blocked: 5,
};

const READINESS_STATUS_TONE: Record<ReadinessPrimaryStatusKey, ReadinessTone> = {
  needs_answer: 'amber',
  ready_for_contract: 'pass',
  needs_qualification: 'mauve',
  contract_linked: 'pass',
  blocked: 'muted',
};

export const READINESS_DISPLAY_LANES = ['clarification', 'qualification', 'contract', 'blocked'] as const satisfies readonly QualificationLane[];

export function projectReadinessDisplay(item: QualificationFeedItem): ReadinessDisplay {
  const primaryStatusKey = derivePrimaryStatusKey(item);

  return {
    primaryStatusKey,
    displayPriority: READINESS_STATUS_PRIORITY[primaryStatusKey],
    tone: READINESS_STATUS_TONE[primaryStatusKey],
  };
}

export function sortReadinessItems(items: QualificationFeedItem[]) {
  return [...items].sort((left, right) => {
    const priorityDelta = projectReadinessDisplay(left).displayPriority - projectReadinessDisplay(right).displayPriority;
    if (priorityDelta !== 0) {
      return priorityDelta;
    }

    return parseTimestamp(right.created_at) - parseTimestamp(left.created_at);
  });
}

function derivePrimaryStatusKey(item: QualificationFeedItem): ReadinessPrimaryStatusKey {
  if (item.goal_state === 'rejected' || item.lane === 'blocked' || item.next_action.kind === 'blocked') {
    return 'blocked';
  }

  const openQuestions = item.open_clarification_request?.state === 'open'
    && (item.open_clarification_request.questions?.length ?? 0) > 0;
  if (openQuestions || item.lane === 'clarification') {
    return 'needs_answer';
  }

  if (item.linked_contract?.id) {
    return 'contract_linked';
  }

  if (item.readiness.ready || item.goal_state === 'ready_for_contract_seed') {
    return 'ready_for_contract';
  }

  return 'needs_qualification';
}

function parseTimestamp(value: string) {
  const timestamp = new Date(value).getTime();
  return Number.isNaN(timestamp) ? 0 : timestamp;
}

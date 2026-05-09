import { buildAPIURL } from './config';

export type QualificationGoalState = 'created' | 'needs_clarification' | 'ready_for_contract_seed' | 'rejected';
export type QualificationLane = 'qualification' | 'clarification' | 'contract' | 'blocked';
export type QualificationContractState = 'seeded' | 'draft' | 'ready_for_approval' | 'approved';
export type QualificationNextActionKind =
  | 'continue_goal'
  | 'answer_clarification'
  | 'draft_contract'
  | 'update_contract'
  | 'approve_contract'
  | 'plan_work'
  | 'blocked'
  | 'none';

export interface QualificationFeedQuestion {
  id: string;
  text: string;
  why_needed: string;
  answer_type: 'text' | 'choice' | 'boolean';
  maps_to: string;
}

export interface QualificationFeedItem {
  intake_id: string;
  goal_id: string;
  organization_id: string;
  project_id: string;
  repo_binding_id: string;
  repository_full_name: string;
  title: string;
  lane: QualificationLane;
  intake_state: string;
  goal_state: QualificationGoalState;
  readiness: {
    ready: boolean;
    reason_codes: string[];
    source: string;
  };
  open_clarification_request?: {
    id: string;
    state: string;
    questions: QualificationFeedQuestion[];
  };
  linked_contract?: {
    id: string;
    state: QualificationContractState;
  };
  next_action: {
    kind: QualificationNextActionKind;
    available: boolean;
    blocking: boolean;
  };
  created_at: string;
}

export interface QualificationFeedResponse {
  items: QualificationFeedItem[];
}

export interface QualificationFeedClientError {
  code: string;
  message: string;
  status?: number;
}

interface BackendErrorEnvelope {
  error?: {
    code?: unknown;
    message?: unknown;
  };
}

export class QualificationFeedClientRequestError extends Error implements QualificationFeedClientError {
  code: string;
  status?: number;

  constructor(error: QualificationFeedClientError) {
    super(error.message);
    this.name = 'QualificationFeedClientRequestError';
    this.code = error.code;
    this.status = error.status;
  }
}

export function isQualificationFeedClientError(error: unknown): error is QualificationFeedClientError {
  return error instanceof QualificationFeedClientRequestError;
}

export async function listQualificationFeed(input: {
  accessToken: string;
  repoBindingId?: string;
  goalState?: QualificationGoalState;
  limit?: number;
}): Promise<QualificationFeedResponse> {
  const query = new URLSearchParams();
  if (input.repoBindingId) {
    query.set('repo_binding_id', input.repoBindingId);
  }
  if (input.goalState) {
    query.set('goal_state', input.goalState);
  }
  if (input.limit !== undefined) {
    query.set('limit', String(input.limit));
  }
  const suffix = query.toString();
  return request<QualificationFeedResponse>(`/v1/qualification-feed${suffix ? `?${suffix}` : ''}`, input.accessToken);
}

async function request<T>(path: string, accessToken: string): Promise<T> {
  let response: Response;
  try {
    response = await fetch(buildAPIURL(path), {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
      credentials: 'omit',
    });
  } catch {
    throw new QualificationFeedClientRequestError({
      code: 'network_error',
      message: 'network error',
    });
  }

  const parsed = await parseJSON(response);
  if (!response.ok) {
    throw toResponseError(response, parsed);
  }

  return parsed as T;
}

async function parseJSON(response: Response): Promise<unknown> {
  const raw = await response.text();
  if (raw.trim() === '') {
    throw new QualificationFeedClientRequestError({
      code: 'response_parse_error',
      message: 'empty JSON response',
      status: response.status,
    });
  }

  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new QualificationFeedClientRequestError({
      code: 'response_parse_error',
      message: 'invalid JSON response',
      status: response.status,
    });
  }
}

function toResponseError(response: Response, parsed: unknown) {
  const envelope = parsed as BackendErrorEnvelope;
  const code = typeof envelope.error?.code === 'string' ? envelope.error.code : 'server_error';
  const message = typeof envelope.error?.message === 'string' ? envelope.error.message : 'server error';

  return new QualificationFeedClientRequestError({
    code,
    message,
    status: response.status,
  });
}

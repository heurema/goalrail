import { buildAPIURL } from './config';
import type { QualificationContractState, QualificationGoalState } from './qualificationFeedClient';

export interface GoalContinuationResponse {
  goal_id: string;
  state: QualificationGoalState;
  readiness?: {
    ready: boolean;
    reason_codes: string[];
  };
  goal?: unknown;
  clarification_request?: unknown;
}

export interface ClarificationAnswerInput {
  question_id: string;
  value: string;
}

export interface DraftContractResponse {
  id: string;
  repo_binding_id: string;
  goal_id: string;
  state: QualificationContractState;
  current_seed_id?: string;
  current_draft_id?: string;
  approved_snapshot_id?: string;
  created_at: string;
  updated_at: string;
}

export interface GoalActionsClientError {
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

export class GoalActionsClientRequestError extends Error implements GoalActionsClientError {
  code: string;
  status?: number;

  constructor(error: GoalActionsClientError) {
    super(error.message);
    this.name = 'GoalActionsClientRequestError';
    this.code = error.code;
    this.status = error.status;
  }
}

export function isGoalActionsClientError(error: unknown): error is GoalActionsClientError {
  return error instanceof GoalActionsClientRequestError;
}

export async function continueGoal(input: {
  accessToken: string;
  goalId: string;
}): Promise<GoalContinuationResponse> {
  return request<GoalContinuationResponse>(`/v1/goals/${encodeURIComponent(input.goalId)}/continuation`, {
    method: 'POST',
    accessToken: input.accessToken,
  });
}

export async function answerClarification(input: {
  accessToken: string;
  clarificationRequestId: string;
  answers: ClarificationAnswerInput[];
}): Promise<GoalContinuationResponse> {
  return request<GoalContinuationResponse>(
    `/v1/clarifications/${encodeURIComponent(input.clarificationRequestId)}/answers/continuation`,
    {
      method: 'POST',
      accessToken: input.accessToken,
      body: {
        answers: input.answers,
      },
    }
  );
}

export async function draftContract(input: {
  accessToken: string;
  goalId: string;
}): Promise<DraftContractResponse> {
  return request<DraftContractResponse>('/v1/contracts', {
    method: 'POST',
    accessToken: input.accessToken,
    body: {
      goal_id: input.goalId,
    },
  });
}

async function request<T>(path: string, options: { method: 'POST'; accessToken: string; body?: unknown }): Promise<T> {
  const headers: Record<string, string> = {
    Authorization: `Bearer ${options.accessToken}`,
  };
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }

  let response: Response;
  try {
    response = await fetch(buildAPIURL(path), {
      method: options.method,
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      credentials: 'omit',
    });
  } catch {
    throw new GoalActionsClientRequestError({
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
    throw new GoalActionsClientRequestError({
      code: 'response_parse_error',
      message: 'empty JSON response',
      status: response.status,
    });
  }

  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new GoalActionsClientRequestError({
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

  return new GoalActionsClientRequestError({
    code,
    message,
    status: response.status,
  });
}

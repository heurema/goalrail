import type { ContractResponse } from './contractDetailClient';
import { buildAPIURL } from './config';

export type ContractStateFilter = ContractResponse['state'];

export interface ContractListResponse {
  contracts: ContractResponse[];
  limit: number;
}

export interface ContractListClientError {
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

export class ContractListClientRequestError extends Error implements ContractListClientError {
  code: string;
  status?: number;

  constructor(error: ContractListClientError) {
    super(error.message);
    this.name = 'ContractListClientRequestError';
    this.code = error.code;
    this.status = error.status;
  }
}

export function isContractListClientError(error: unknown): error is ContractListClientError {
  return error instanceof ContractListClientRequestError;
}

export async function listContracts(input: {
  accessToken: string;
  projectId?: string;
  repoBindingId?: string;
  goalId?: string;
  state?: ContractStateFilter;
  limit?: number;
}): Promise<ContractListResponse> {
  const query = new URLSearchParams();
  if (input.projectId) {
    query.set('project_id', input.projectId);
  }
  if (input.repoBindingId) {
    query.set('repo_binding_id', input.repoBindingId);
  }
  if (input.goalId) {
    query.set('goal_id', input.goalId);
  }
  if (input.state) {
    query.set('state', input.state);
  }
  if (input.limit !== undefined) {
    query.set('limit', String(input.limit));
  }

  const suffix = query.toString();
  return request<ContractListResponse>(`/v1/contracts${suffix ? `?${suffix}` : ''}`, input.accessToken);
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
    throw new ContractListClientRequestError({
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
    throw new ContractListClientRequestError({
      code: 'response_parse_error',
      message: 'empty JSON response',
      status: response.status,
    });
  }

  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new ContractListClientRequestError({
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

  return new ContractListClientRequestError({
    code,
    message,
    status: response.status,
  });
}

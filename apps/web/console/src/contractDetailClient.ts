import { buildAPIURL } from './config';

export type ContractDetailClientErrorCode =
  | 'not_found'
  | 'forbidden'
  | 'membership_required'
  | 'unauthorized'
  | 'database_not_configured'
  | 'network_error'
  | 'response_parse_error'
  | 'server_error';

export interface ContractResponse {
  id: string;
  repo_binding_id: string;
  goal_id: string;
  state: 'seeded' | 'draft' | 'ready_for_approval' | 'approved';
  current_seed_id?: string;
  current_draft_id?: string;
  approved_snapshot_id?: string;
  created_at: string;
  updated_at: string;
}

export interface ContractDetailClientError {
  code: ContractDetailClientErrorCode;
  message: string;
  status?: number;
}

interface BackendErrorEnvelope {
  error?: {
    code?: unknown;
    message?: unknown;
  };
}

const SUPPORTED_ERROR_CODES = new Set<ContractDetailClientErrorCode>([
  'not_found',
  'forbidden',
  'membership_required',
  'unauthorized',
  'database_not_configured',
  'network_error',
  'response_parse_error',
  'server_error',
]);

export class ContractDetailClientRequestError extends Error implements ContractDetailClientError {
  code: ContractDetailClientErrorCode;
  status?: number;

  constructor(error: ContractDetailClientError) {
    super(error.message);
    this.name = 'ContractDetailClientRequestError';
    this.code = error.code;
    this.status = error.status;
  }
}

export function isContractDetailClientError(error: unknown): error is ContractDetailClientError {
  return error instanceof ContractDetailClientRequestError;
}

export async function getContractDetail(input: {
  accessToken: string;
  contractId: string;
}): Promise<ContractResponse> {
  const contractId = encodeURIComponent(input.contractId);
  return request<ContractResponse>(`/v1/contracts/${contractId}`, input.accessToken);
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
    throw new ContractDetailClientRequestError({
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
    throw new ContractDetailClientRequestError({
      code: 'response_parse_error',
      message: 'empty JSON response',
      status: response.status,
    });
  }

  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new ContractDetailClientRequestError({
      code: 'response_parse_error',
      message: 'invalid JSON response',
      status: response.status,
    });
  }
}

function toResponseError(response: Response, parsed: unknown) {
  const envelope = parsed as BackendErrorEnvelope;
  const rawCode = typeof envelope.error?.code === 'string' ? envelope.error.code : 'server_error';
  const code = SUPPORTED_ERROR_CODES.has(rawCode as ContractDetailClientErrorCode)
    ? rawCode as ContractDetailClientErrorCode
    : 'server_error';
  const message = typeof envelope.error?.message === 'string' ? envelope.error.message : 'server error';

  return new ContractDetailClientRequestError({
    code,
    message,
    status: response.status,
  });
}

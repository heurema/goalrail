import { buildAPIURL } from './config';

export type ContractDraftClientErrorCode =
  | 'not_found'
  | 'invalid_state'
  | 'forbidden'
  | 'membership_required'
  | 'unauthorized'
  | 'database_not_configured'
  | 'network_error'
  | 'response_parse_error'
  | 'server_error';

export interface ContractDraftSourceRef {
  kind: string;
  id: string;
  [key: string]: unknown;
}

export interface ContractDraftResponse {
  id: string;
  contract_id: string;
  contract_seed_id: string;
  goal_id: string;
  repo_binding_id: string;
  title: string;
  intent_summary: string;
  proposed_scope: string[];
  proposed_non_goals: string[];
  proposed_constraints: string[];
  proposed_acceptance_criteria: string[];
  proposed_expected_checks: string[];
  proposed_proof_expectations: string[];
  risk_hints: string[];
  source_refs: ContractDraftSourceRef[];
  state: 'draft' | 'ready_for_approval' | string;
  created_at: string;
}

export interface ContractDraftClientError {
  code: ContractDraftClientErrorCode;
  message: string;
  status?: number;
}

interface BackendErrorEnvelope {
  error?: {
    code?: unknown;
    message?: unknown;
  };
}

const SUPPORTED_ERROR_CODES = new Set<ContractDraftClientErrorCode>([
  'not_found',
  'invalid_state',
  'forbidden',
  'membership_required',
  'unauthorized',
  'database_not_configured',
  'network_error',
  'response_parse_error',
  'server_error',
]);

export class ContractDraftClientRequestError extends Error implements ContractDraftClientError {
  code: ContractDraftClientErrorCode;
  status?: number;

  constructor(error: ContractDraftClientError) {
    super(error.message);
    this.name = 'ContractDraftClientRequestError';
    this.code = error.code;
    this.status = error.status;
  }
}

export function isContractDraftClientError(error: unknown): error is ContractDraftClientError {
  return error instanceof ContractDraftClientRequestError;
}

export async function getCurrentContractDraft(input: {
  accessToken: string;
  contractId: string;
}): Promise<ContractDraftResponse> {
  const contractId = encodeURIComponent(input.contractId);
  return request<ContractDraftResponse>(`/v1/contracts/${contractId}/current-draft`, input.accessToken);
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
    throw new ContractDraftClientRequestError({
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
    throw new ContractDraftClientRequestError({
      code: 'response_parse_error',
      message: 'empty JSON response',
      status: response.status,
    });
  }

  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new ContractDraftClientRequestError({
      code: 'response_parse_error',
      message: 'invalid JSON response',
      status: response.status,
    });
  }
}

function toResponseError(response: Response, parsed: unknown) {
  const envelope = parsed as BackendErrorEnvelope;
  const rawCode = typeof envelope.error?.code === 'string' ? envelope.error.code : 'server_error';
  const code = SUPPORTED_ERROR_CODES.has(rawCode as ContractDraftClientErrorCode)
    ? rawCode as ContractDraftClientErrorCode
    : 'server_error';
  const message = typeof envelope.error?.message === 'string' ? envelope.error.message : 'server error';

  return new ContractDraftClientRequestError({
    code,
    message,
    status: response.status,
  });
}

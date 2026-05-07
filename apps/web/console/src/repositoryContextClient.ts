import { buildAPIURL } from './config';

export interface OrganizationSummary {
  id: string;
  slug: string;
  display_name: string;
  state: string;
}

export interface RepositoryProject {
  id: string;
  slug: string;
  display_name: string;
  state: string;
  created_at: string;
  updated_at: string;
}

export interface RepositoryBinding {
  id: string;
  provider: string;
  repository_full_name: string;
  repository_url: string;
  default_branch: string;
  workflow_base_branch: string;
  path_scope: string;
  access_mode: string;
  state: string;
  created_at: string;
  updated_at: string;
}

export interface RepositoryContextRecord {
  project: RepositoryProject;
  repo_binding: RepositoryBinding;
}

export interface OrganizationRepositoryContextResponse {
  organization: OrganizationSummary;
  contexts: RepositoryContextRecord[];
}

export interface RepositoryContextClientError {
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

export class RepositoryContextClientRequestError extends Error implements RepositoryContextClientError {
  code: string;
  status?: number;

  constructor(error: RepositoryContextClientError) {
    super(error.message);
    this.name = 'RepositoryContextClientRequestError';
    this.code = error.code;
    this.status = error.status;
  }
}

export function isRepositoryContextClientError(error: unknown): error is RepositoryContextClientError {
  return error instanceof RepositoryContextClientRequestError;
}

export async function getOrganizationRepositoryContext(input: {
  accessToken: string;
  organizationId: string;
}): Promise<OrganizationRepositoryContextResponse> {
  return request<OrganizationRepositoryContextResponse>(
    `/v1/organizations/${encodeURIComponent(input.organizationId)}/repository-context`,
    input.accessToken
  );
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
    throw new RepositoryContextClientRequestError({
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
    throw new RepositoryContextClientRequestError({
      code: 'response_parse_error',
      message: 'empty JSON response',
      status: response.status,
    });
  }

  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new RepositoryContextClientRequestError({
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

  return new RepositoryContextClientRequestError({
    code,
    message,
    status: response.status,
  });
}

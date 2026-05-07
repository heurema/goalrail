import { buildAPIURL } from './config';

export type OrganizationUserRole = 'owner' | 'admin' | 'member' | 'viewer';
export type OrganizationUserState = 'active' | 'inactive';

export interface OrganizationUser {
  id: string;
  display_name: string;
  email?: string;
  state: OrganizationUserState;
  created_at: string;
  updated_at: string;
}

export interface OrganizationMembership {
  id: string;
  organization_id: string;
  user_id: string;
  role: OrganizationUserRole;
  state: OrganizationUserState;
  created_at: string;
  updated_at: string;
}

export interface CredentialSummary {
  must_change_password: boolean;
  password_changed_at?: string | null;
}

export interface OrganizationUserRecord {
  user: OrganizationUser;
  organization_membership: OrganizationMembership;
  credential: CredentialSummary;
}

export interface ListOrganizationUsersResponse {
  users: OrganizationUserRecord[];
}

export interface CreateOrganizationUserResponse extends OrganizationUserRecord {
  temporary_password?: string;
}

export interface ResetOrganizationUserTemporaryPasswordResponse extends OrganizationUserRecord {
  temporary_password: string;
}

export type PatchOrganizationUserResponse = OrganizationUserRecord;

export interface UsersClientError {
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

interface RequestOptions {
  method: 'GET' | 'POST' | 'PATCH';
  body?: unknown;
  accessToken: string;
}

export class UsersClientRequestError extends Error implements UsersClientError {
  code: string;
  status?: number;

  constructor(error: UsersClientError) {
    super(error.message);
    this.name = 'UsersClientRequestError';
    this.code = error.code;
    this.status = error.status;
  }
}

export function isUsersClientError(error: unknown): error is UsersClientError {
  return error instanceof UsersClientRequestError;
}

export async function listOrganizationUsers(input: {
  accessToken: string;
  organizationId: string;
}): Promise<ListOrganizationUsersResponse> {
  return request<ListOrganizationUsersResponse>(organizationUsersPath(input.organizationId), {
    method: 'GET',
    accessToken: input.accessToken,
  });
}

export async function createOrganizationUser(input: {
  accessToken: string;
  organizationId: string;
  email: string;
  displayName: string;
  role: OrganizationUserRole;
}): Promise<CreateOrganizationUserResponse> {
  return request<CreateOrganizationUserResponse>(organizationUsersPath(input.organizationId), {
    method: 'POST',
    accessToken: input.accessToken,
    body: {
      email: input.email,
      display_name: input.displayName,
      role: input.role,
    },
  });
}

export async function patchOrganizationUser(input: {
  accessToken: string;
  organizationId: string;
  userId: string;
  displayName?: string;
  role?: OrganizationUserRole;
  state?: OrganizationUserState;
}): Promise<PatchOrganizationUserResponse> {
  return request<PatchOrganizationUserResponse>(
    `${organizationUsersPath(input.organizationId)}/${encodeURIComponent(input.userId)}`,
    {
      method: 'PATCH',
      accessToken: input.accessToken,
      body: {
        ...(input.displayName === undefined ? {} : { display_name: input.displayName }),
        ...(input.role === undefined ? {} : { role: input.role }),
        ...(input.state === undefined ? {} : { state: input.state }),
      },
    }
  );
}

export async function resetOrganizationUserTemporaryPassword(input: {
  accessToken: string;
  organizationId: string;
  userId: string;
}): Promise<ResetOrganizationUserTemporaryPasswordResponse> {
  return request<ResetOrganizationUserTemporaryPasswordResponse>(
    `${organizationUsersPath(input.organizationId)}/${encodeURIComponent(input.userId)}/temporary-password-resets`,
    {
      method: 'POST',
      accessToken: input.accessToken,
    }
  );
}

function organizationUsersPath(organizationId: string) {
  return `/v1/organizations/${encodeURIComponent(organizationId)}/users`;
}

async function request<T>(path: string, options: RequestOptions): Promise<T> {
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
    throw new UsersClientRequestError({
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
    throw new UsersClientRequestError({
      code: 'response_parse_error',
      message: 'empty JSON response',
      status: response.status,
    });
  }

  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new UsersClientRequestError({
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

  return new UsersClientRequestError({
    code,
    message,
    status: response.status,
  });
}

import { buildAPIURL } from './config';

export interface LoginResponse {
  user_id: string;
  access_token: string;
  access_token_expires_at: string;
  token_type: 'Bearer';
  refresh_token: string;
  refresh_token_expires_at: string;
  must_change_password: boolean;
}

export interface ChangePasswordResponse {
  user_id: string;
  must_change_password: false;
  password_changed_at: string;
}

export interface MeResponse {
  user: {
    id: string;
    display_name: string;
    email?: string;
    state: string;
  };
  organization_membership: {
    id: string;
    organization_id: string;
    user_id: string;
    role: string;
    state: string;
  };
}

export interface LogoutResponse {
  revoked: boolean;
}

export interface AuthClientError {
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
  method: 'GET' | 'POST';
  body?: unknown;
  accessToken?: string;
}

export class AuthClientRequestError extends Error implements AuthClientError {
  code: string;
  status?: number;

  constructor(error: AuthClientError) {
    super(error.message);
    this.name = 'AuthClientRequestError';
    this.code = error.code;
    this.status = error.status;
  }
}

export function isAuthClientError(error: unknown): error is AuthClientError {
  return error instanceof AuthClientRequestError;
}

export async function login(input: { email: string; password: string }): Promise<LoginResponse> {
  return request<LoginResponse>('/v1/auth/login', {
    method: 'POST',
    body: input,
  });
}

export async function changePassword(input: {
  accessToken: string;
  currentPassword: string;
  newPassword: string;
}): Promise<ChangePasswordResponse> {
  return request<ChangePasswordResponse>('/v1/auth/change-password', {
    method: 'POST',
    accessToken: input.accessToken,
    body: {
      current_password: input.currentPassword,
      new_password: input.newPassword,
    },
  });
}

export async function me(accessToken: string): Promise<MeResponse> {
  return request<MeResponse>('/v1/me', {
    method: 'GET',
    accessToken,
  });
}

export async function logout(accessToken: string): Promise<LogoutResponse> {
  return request<LogoutResponse>('/v1/auth/logout', {
    method: 'POST',
    accessToken,
  });
}

async function request<T>(path: string, options: RequestOptions): Promise<T> {
  const headers: Record<string, string> = {};
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }
  if (options.accessToken) {
    headers.Authorization = `Bearer ${options.accessToken}`;
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
    throw new AuthClientRequestError({
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
    throw new AuthClientRequestError({
      code: 'response_parse_error',
      message: 'empty JSON response',
      status: response.status,
    });
  }

  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new AuthClientRequestError({
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

  return new AuthClientRequestError({
    code,
    message,
    status: response.status,
  });
}

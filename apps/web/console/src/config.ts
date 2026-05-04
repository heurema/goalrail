export function normalizeAPIBaseURL(value: string | undefined) {
  return String(value ?? '').trim().replace(/\/+$/, '');
}

export const apiBaseURL = normalizeAPIBaseURL(import.meta.env.VITE_GOALRAIL_API_BASE_URL);

export function buildAPIURL(path: string, baseURL = apiBaseURL) {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${baseURL}${normalizedPath}`;
}

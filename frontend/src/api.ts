const BASE = '/api';

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(BASE + path, {
    method,
    credentials: 'include',
    headers: body ? { 'Content-Type': 'application/json' } : {},
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error ?? `HTTP ${res.status}`);
  return data as T;
}

export const api = {
  register: (email: string, password: string) =>
    request('POST', '/auth/register', { email, password }),

  login: (email: string, password: string) =>
    request<{ access_token?: string; mfa_required?: boolean; mfa_session_id?: string }>(
      'POST', '/auth/login', { email, password }
    ),

  verifyEmail: (token: string) =>
    request('GET', `/auth/verify-email?token=${encodeURIComponent(token)}`),

  resendVerification: (email: string) =>
    request('POST', '/auth/resend-verification', { email }),

  resetRequest: (email: string) =>
    request('POST', '/auth/password/reset-request', { email }),

  resetPassword: (token: string, password: string) =>
    request('POST', '/auth/password/reset', { token, password }),

  mfaVerify: (mfa_session_id: string, code: string) =>
    request<{ access_token: string }>('POST', '/auth/2fa/verify', { mfa_session_id, code }),

  mfaRecovery: (mfa_session_id: string, recovery_code: string) =>
    request<{ access_token: string }>('POST', '/auth/2fa/recovery', { mfa_session_id, recovery_code }),

  enroll2fa: (token: string) =>
    request<{ secret: string; otp_uri: string; recovery_codes: string[] }>(
      'POST', '/account/2fa/enroll', undefined
    ),

  confirm2fa: (token: string, code: string) =>
    request('POST', '/account/2fa/confirm', { code }),

  refresh: () =>
    request<{ access_token: string }>('POST', '/auth/token/refresh'),

  logout: () =>
    request('POST', '/auth/logout'),
};

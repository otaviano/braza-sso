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
    request<{ mfa_required?: boolean; mfa_session_id?: string }>(
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

  mfaVerify: (mfaSessionId: string, code: string) =>
    request('POST', '/auth/2fa/verify', { mfa_session_id: mfaSessionId, code }),

  mfaRecovery: (mfaSessionId: string, recoveryCode: string) =>
    request('POST', '/auth/2fa/recovery', { mfa_session_id: mfaSessionId, recovery_code: recoveryCode }),

  enroll2fa: () =>
    request<{ secret: string; otp_uri: string; recovery_codes: string[] }>(
      'POST', '/account/2fa/enroll', undefined
    ),

  confirm2fa: (code: string) =>
    request('POST', '/account/2fa/confirm', { code }),

  refresh: () =>
    request('POST', '/auth/token/refresh'),

  logout: () =>
    request('POST', '/auth/logout'),
};

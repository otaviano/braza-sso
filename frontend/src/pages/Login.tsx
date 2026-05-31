import { useState } from 'react';
import { api } from '../api';
import { card, title, form, input, btn, link, errorStyle } from '../styles';

interface Props {
  onRegister: () => void;
  onForgotPassword: () => void;
  onMFARequired: (sessionId: string) => void;
}

function getNextUrl(): string {
  return new URLSearchParams(window.location.search).get('next') ?? '/';
}

export default function Login({ onRegister, onForgotPassword, onMFARequired }: Props) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const res = await api.login(email, password);
      if (res.mfa_required && res.mfa_session_id) {
        onMFARequired(res.mfa_session_id);
      } else {
        window.location.href = getNextUrl();
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={card}>
      <div style={brand}>braza</div>
      <h1 style={title}>Sign in</h1>
      <form onSubmit={handleSubmit} style={form}>
        <input style={input} type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required />
        <div style={{ position: 'relative' }}>
          <input
            style={{ ...input, paddingRight: '2.75rem' }}
            type={showPassword ? 'text' : 'password'}
            placeholder="Password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            required
          />
          <button type="button" onClick={() => setShowPassword(v => !v)} style={eyeBtn} tabIndex={-1}>
            {showPassword ? '🙈' : '👁'}
          </button>
        </div>
        {error && <p style={errorStyle}>{error}</p>}
        <button style={btn} type="submit" disabled={loading}>{loading ? 'Signing in…' : 'Sign in'}</button>
      </form>
      <a href="/auth/federation/google" style={googleBtn}>
        <GoogleIcon /> Continue with Google
      </a>
      <div style={linksRow}>
        <button style={link} onClick={onForgotPassword}>Forgot password?</button>
        <button style={link} onClick={onRegister}>Create account</button>
      </div>
    </div>
  );
}

function GoogleIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" style={{ flexShrink: 0 }}>
      <path fill="#4285F4" d="M17.64 9.2c0-.637-.057-1.251-.164-1.84H9v3.481h4.844c-.209 1.125-.843 2.078-1.796 2.717v2.258h2.908c1.702-1.567 2.684-3.875 2.684-6.615z"/>
      <path fill="#34A853" d="M9 18c2.43 0 4.467-.806 5.956-2.18l-2.908-2.259c-.806.54-1.837.86-3.048.86-2.344 0-4.328-1.584-5.036-3.711H.957v2.332C2.438 15.983 5.482 18 9 18z"/>
      <path fill="#FBBC05" d="M3.964 10.71c-.18-.54-.282-1.117-.282-1.71s.102-1.17.282-1.71V4.958H.957C.347 6.173 0 7.548 0 9s.348 2.827.957 4.042l3.007-2.332z"/>
      <path fill="#EA4335" d="M9 3.58c1.321 0 2.508.454 3.44 1.345l2.582-2.58C13.463.891 11.426 0 9 0 5.482 0 2.438 2.017.957 4.958L3.964 7.29C4.672 5.163 6.656 3.58 9 3.58z"/>
    </svg>
  );
}

const brand: React.CSSProperties = {
  textAlign: 'center',
  fontSize: '1.5rem',
  fontWeight: 700,
  color: 'var(--accent)',
  letterSpacing: '-0.04em',
  marginBottom: '1.5rem',
};

const eyeBtn: React.CSSProperties = {
  position: 'absolute', right: '0.75rem', top: '50%', transform: 'translateY(-50%)',
  background: 'none', border: 'none', cursor: 'pointer', fontSize: '1rem', padding: 0,
  color: 'var(--text-muted)', lineHeight: 1,
};

const googleBtn: React.CSSProperties = {
  display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.625rem',
  marginTop: '0.875rem', padding: '0.75rem',
  border: '1px solid var(--border)', borderRadius: 8,
  color: 'var(--text)', textDecoration: 'none', fontSize: '0.9375rem',
  background: 'var(--surface-2)', transition: 'border-color 0.15s',
};

const linksRow: React.CSSProperties = {
  display: 'flex', justifyContent: 'space-between', marginTop: '1.25rem',
};

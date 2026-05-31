import { useState } from 'react';
import { api } from '../api';
import { card, title, form, input, btn, link, errorStyle } from '../styles';

interface Props { onLogin: () => void; }

function strengthLabel(password: string): { label: string; color: string } {
  let score = 0;
  if (password.length >= 12) score++;
  if (/[A-Z]/.test(password)) score++;
  if (/[0-9]/.test(password)) score++;
  if (/[^a-zA-Z0-9]/.test(password)) score++;
  if (score <= 1) return { label: 'Weak', color: 'var(--error)' };
  if (score === 2) return { label: 'Fair', color: '#f59e0b' };
  if (score === 3) return { label: 'Good', color: '#22c55e' };
  return { label: 'Strong', color: '#16a34a' };
}

export default function Register({ onLogin }: Props) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [done, setDone] = useState(false);
  const [loading, setLoading] = useState(false);
  const strength = strengthLabel(password);
  const confirmMismatch = confirm.length > 0 && confirm !== password;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (password !== confirm) {
      setError('Passwords do not match');
      return;
    }
    setError('');
    setLoading(true);
    try {
      await api.register(email, password);
      setDone(true);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Registration failed');
    } finally {
      setLoading(false);
    }
  }

  if (done) {
    return (
      <div style={card}>
        <div style={brand}>braza</div>
        <h1 style={title}>Check your email</h1>
        <p style={{ textAlign: 'center', color: 'var(--text-muted)', marginBottom: '1.5rem' }}>
          We sent a verification link to <strong style={{ color: 'var(--text)' }}>{email}</strong>.
        </p>
        <button style={btn} onClick={onLogin}>Back to sign in</button>
      </div>
    );
  }

  return (
    <div style={card}>
      <div style={brand}>braza</div>
      <h1 style={title}>Create account</h1>
      <form onSubmit={handleSubmit} style={form}>
        <input style={input} type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required />
        <div>
          <div style={{ position: 'relative' }}>
            <input
              style={{ ...input, paddingRight: '2.75rem' }}
              type={showPassword ? 'text' : 'password'}
              placeholder="Password (min 12 chars)"
              value={password}
              onChange={e => setPassword(e.target.value)}
              required
            />
            <button type="button" onClick={() => setShowPassword(v => !v)} style={eyeBtn} tabIndex={-1}>
              {showPassword ? '🙈' : '👁'}
            </button>
          </div>
          {password && <span style={{ fontSize: '0.78rem', color: strength.color, marginTop: '0.25rem', display: 'block' }}>{strength.label}</span>}
        </div>
        <div>
          <div style={{ position: 'relative' }}>
            <input
              style={{ ...input, paddingRight: '2.75rem', borderColor: confirmMismatch ? 'var(--error)' : undefined }}
              type={showPassword ? 'text' : 'password'}
              placeholder="Confirm password"
              value={confirm}
              onChange={e => setConfirm(e.target.value)}
              required
            />
            <button type="button" onClick={() => setShowPassword(v => !v)} style={eyeBtn} tabIndex={-1}>
              {showPassword ? '🙈' : '👁'}
            </button>
          </div>
          {confirmMismatch && <span style={{ fontSize: '0.78rem', color: 'var(--error)', marginTop: '0.25rem', display: 'block' }}>Passwords do not match</span>}
        </div>
        {error && <p style={errorStyle}>{error}</p>}
        <button style={btn} type="submit" disabled={loading}>{loading ? 'Creating…' : 'Create account'}</button>
      </form>
      <div style={{ textAlign: 'center', marginTop: '1.25rem' }}>
        <button style={link} onClick={onLogin}>Already have an account? Sign in</button>
      </div>
    </div>
  );
}

const brand: React.CSSProperties = {
  textAlign: 'center', fontSize: '1.5rem', fontWeight: 700,
  color: 'var(--accent)', letterSpacing: '-0.04em', marginBottom: '1.5rem',
};

const eyeBtn: React.CSSProperties = {
  position: 'absolute', right: '0.75rem', top: '50%', transform: 'translateY(-50%)',
  background: 'none', border: 'none', cursor: 'pointer', fontSize: '1rem', padding: 0,
  color: 'var(--text-muted)', lineHeight: 1,
};

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
  if (score <= 1) return { label: 'Weak', color: '#c00' };
  if (score === 2) return { label: 'Fair', color: '#f90' };
  if (score === 3) return { label: 'Good', color: '#090' };
  return { label: 'Strong', color: '#060' };
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
        <h1 style={title}>Check your email</h1>
        <p style={{ textAlign: 'center', color: '#555' }}>We sent a verification link to <strong>{email}</strong>.</p>
        <button style={{ ...btn, marginTop: '1.5rem' }} onClick={onLogin}>Back to sign in</button>
      </div>
    );
  }

  return (
    <div style={card}>
      <h1 style={title}>Create account</h1>
      <form onSubmit={handleSubmit} style={form}>
        <input style={input} type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required />
        <div style={{ position: 'relative' }}>
          <input
            style={{ ...input, width: '100%', boxSizing: 'border-box', paddingRight: '2.5rem' }}
            type={showPassword ? 'text' : 'password'}
            placeholder="Password (min 12 chars)"
            value={password}
            onChange={e => setPassword(e.target.value)}
            required
          />
          <button type="button" onClick={() => setShowPassword(v => !v)} style={eyeBtn} tabIndex={-1}>
            {showPassword ? '🙈' : '👁'}
          </button>
          {password && <span style={{ fontSize: '0.8rem', color: strength.color }}>{strength.label}</span>}
        </div>
        <div style={{ position: 'relative' }}>
          <input
            style={{ ...input, width: '100%', boxSizing: 'border-box', paddingRight: '2.5rem', borderColor: confirmMismatch ? '#c00' : undefined }}
            type={showPassword ? 'text' : 'password'}
            placeholder="Confirm password"
            value={confirm}
            onChange={e => setConfirm(e.target.value)}
            required
          />
          <button type="button" onClick={() => setShowPassword(v => !v)} style={eyeBtn} tabIndex={-1}>
            {showPassword ? '🙈' : '👁'}
          </button>
          {confirmMismatch && <span style={{ fontSize: '0.8rem', color: '#c00' }}>Passwords do not match</span>}
        </div>
        {error && <p style={errorStyle}>{error}</p>}
        <button style={btn} type="submit" disabled={loading}>{loading ? 'Creating…' : 'Create account'}</button>
      </form>
      <div style={{ textAlign: 'center', marginTop: '1rem' }}>
        <button style={link} onClick={onLogin}>Already have an account? Sign in</button>
      </div>
    </div>
  );
}

const eyeBtn: React.CSSProperties = {
  position: 'absolute', right: '0.5rem', top: '0.6rem',
  background: 'none', border: 'none', cursor: 'pointer', fontSize: '1rem', padding: 0,
};

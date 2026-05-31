import { useState } from 'react';
import { api } from '../api';
import { card, title, form, input, btn, errorStyle } from '../styles';

interface Props { onDone: () => void; }

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

export default function PasswordResetConfirm({ onDone }: Props) {
  const token = new URLSearchParams(window.location.search).get('token') ?? '';
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
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
      await api.resetPassword(token, password);
      onDone();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Reset failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={card}>
      <h1 style={title}>Set new password</h1>
      <form onSubmit={handleSubmit} style={form}>
        <div style={{ position: 'relative' }}>
          <input
            style={{ ...input, width: '100%', boxSizing: 'border-box', paddingRight: '2.5rem' }}
            type={showPassword ? 'text' : 'password'}
            placeholder="New password (min 12 chars)"
            value={password}
            onChange={e => setPassword(e.target.value)}
            required
          />
          <button
            type="button"
            onClick={() => setShowPassword(v => !v)}
            style={eyeBtn}
            tabIndex={-1}
          >
            {showPassword ? '🙈' : '👁'}
          </button>
          {password && <span style={{ fontSize: '0.8rem', color: strength.color }}>{strength.label}</span>}
        </div>
        <div>
          <input
            style={{ ...input, width: '100%', boxSizing: 'border-box', borderColor: confirmMismatch ? '#c00' : undefined }}
            type={showPassword ? 'text' : 'password'}
            placeholder="Confirm new password"
            value={confirm}
            onChange={e => setConfirm(e.target.value)}
            required
          />
          {confirmMismatch && <span style={{ fontSize: '0.8rem', color: '#c00' }}>Passwords do not match</span>}
        </div>
        {error && <p style={errorStyle}>{error}</p>}
        <button style={btn} type="submit" disabled={loading}>{loading ? 'Updating…' : 'Update password'}</button>
      </form>
    </div>
  );
}

const eyeBtn: React.CSSProperties = {
  position: 'absolute', right: '0.5rem', top: '0.6rem',
  background: 'none', border: 'none', cursor: 'pointer', fontSize: '1rem', padding: 0,
};

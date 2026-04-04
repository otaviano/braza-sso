import { useState } from 'react';
import { api } from '../api';

interface Props { onLogin: () => void; }

function strengthLabel(pw: string): { label: string; color: string } {
  let score = 0;
  if (pw.length >= 12) score++;
  if (/[A-Z]/.test(pw)) score++;
  if (/[0-9]/.test(pw)) score++;
  if (/[^a-zA-Z0-9]/.test(pw)) score++;
  if (score <= 1) return { label: 'Weak', color: '#c00' };
  if (score === 2) return { label: 'Fair', color: '#f90' };
  if (score === 3) return { label: 'Good', color: '#090' };
  return { label: 'Strong', color: '#060' };
}

export default function Register({ onLogin }: Props) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [done, setDone] = useState(false);
  const [loading, setLoading] = useState(false);
  const strength = strengthLabel(password);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await api.register(email, password);
      setDone(true);
    } catch (err: any) {
      setError(err.message ?? 'Registration failed');
    } finally {
      setLoading(false);
    }
  }

  if (done) return (
    <div style={card}>
      <h1 style={title}>Check your email</h1>
      <p style={{ textAlign: 'center', color: '#555' }}>We sent a verification link to <strong>{email}</strong>.</p>
      <button style={{ ...btn, marginTop: '1.5rem' }} onClick={onLogin}>Back to sign in</button>
    </div>
  );

  return (
    <div style={card}>
      <h1 style={title}>Create account</h1>
      <form onSubmit={handleSubmit} style={form}>
        <input style={input} type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required />
        <div>
          <input style={input} type="password" placeholder="Password (min 12 chars)" value={password} onChange={e => setPassword(e.target.value)} required />
          {password && <span style={{ fontSize: '0.8rem', color: strength.color }}>{strength.label}</span>}
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

const card: React.CSSProperties = { background: '#fff', borderRadius: 12, padding: '2rem', width: 360, boxShadow: '0 2px 16px rgba(0,0,0,0.1)' };
const title: React.CSSProperties = { marginBottom: '1.5rem', fontSize: '1.5rem', textAlign: 'center' };
const form: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: '0.75rem' };
const input: React.CSSProperties = { padding: '0.625rem', borderRadius: 6, border: '1px solid #ddd', fontSize: '1rem', width: '100%', boxSizing: 'border-box' };
const btn: React.CSSProperties = { padding: '0.75rem', background: '#0066ff', color: '#fff', border: 'none', borderRadius: 6, fontSize: '1rem', cursor: 'pointer' };
const link: React.CSSProperties = { background: 'none', border: 'none', color: '#0066ff', cursor: 'pointer', fontSize: '0.875rem' };
const errorStyle: React.CSSProperties = { color: '#c00', fontSize: '0.875rem' };

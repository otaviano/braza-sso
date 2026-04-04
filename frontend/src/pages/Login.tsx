import { useState } from 'react';
import { api } from '../api';

interface Props {
  onRegister: () => void;
  onForgotPassword: () => void;
  onMFARequired: (sessionId: string) => void;
}

export default function Login({ onRegister, onForgotPassword, onMFARequired }: Props) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
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
      } else if (res.access_token) {
        localStorage.setItem('access_token', res.access_token);
        window.location.href = '/';
      }
    } catch (err: any) {
      setError(err.message ?? 'Login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={card}>
      <h1 style={title}>Sign in</h1>
      <form onSubmit={handleSubmit} style={form}>
        <input style={input} type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required />
        <input style={input} type="password" placeholder="Password" value={password} onChange={e => setPassword(e.target.value)} required />
        {error && <p style={errorStyle}>{error}</p>}
        <button style={btn} type="submit" disabled={loading}>{loading ? 'Signing in…' : 'Sign in'}</button>
      </form>
      <a href="/auth/federation/google" style={googleBtn}>Continue with Google</a>
      <div style={links}>
        <button style={link} onClick={onForgotPassword}>Forgot password?</button>
        <button style={link} onClick={onRegister}>Create account</button>
      </div>
    </div>
  );
}

const card: React.CSSProperties = { background: '#fff', borderRadius: 12, padding: '2rem', width: 360, boxShadow: '0 2px 16px rgba(0,0,0,0.1)' };
const title: React.CSSProperties = { marginBottom: '1.5rem', fontSize: '1.5rem', textAlign: 'center' };
const form: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: '0.75rem' };
const input: React.CSSProperties = { padding: '0.625rem', borderRadius: 6, border: '1px solid #ddd', fontSize: '1rem' };
const btn: React.CSSProperties = { padding: '0.75rem', background: '#0066ff', color: '#fff', border: 'none', borderRadius: 6, fontSize: '1rem', cursor: 'pointer' };
const googleBtn: React.CSSProperties = { display: 'block', marginTop: '0.75rem', padding: '0.75rem', textAlign: 'center', border: '1px solid #ddd', borderRadius: 6, color: '#333', textDecoration: 'none' };
const links: React.CSSProperties = { display: 'flex', justifyContent: 'space-between', marginTop: '1rem' };
const link: React.CSSProperties = { background: 'none', border: 'none', color: '#0066ff', cursor: 'pointer', fontSize: '0.875rem' };
const errorStyle: React.CSSProperties = { color: '#c00', fontSize: '0.875rem' };

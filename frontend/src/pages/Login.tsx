import { useState } from 'react';
import { api } from '../api';
import { card, title, form, input, btn, link, errorStyle } from '../styles';

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
      } else {
        window.location.href = '/';
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Login failed');
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

const googleBtn: React.CSSProperties = { display: 'block', marginTop: '0.75rem', padding: '0.75rem', textAlign: 'center', border: '1px solid #ddd', borderRadius: 6, color: '#333', textDecoration: 'none' };
const links: React.CSSProperties = { display: 'flex', justifyContent: 'space-between', marginTop: '1rem' };

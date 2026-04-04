import { useState } from 'react';
import { api } from '../api';

interface Props { onBack: () => void; }

export default function PasswordReset({ onBack }: Props) {
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    await api.resetRequest(email).catch(() => {});
    setSent(true);
    setLoading(false);
  }

  if (sent) return (
    <div style={card}>
      <h1 style={title}>Check your email</h1>
      <p style={center}>If an account exists for <strong>{email}</strong>, you'll receive a reset link shortly.</p>
      <button style={btn} onClick={onBack}>Back to sign in</button>
    </div>
  );

  return (
    <div style={card}>
      <h1 style={title}>Reset password</h1>
      <form onSubmit={handleSubmit} style={form}>
        <input style={input} type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required />
        <button style={btn} type="submit" disabled={loading}>{loading ? 'Sending…' : 'Send reset link'}</button>
      </form>
      <div style={{ textAlign: 'center', marginTop: '1rem' }}>
        <button style={link} onClick={onBack}>Back to sign in</button>
      </div>
    </div>
  );
}

const card: React.CSSProperties = { background: '#fff', borderRadius: 12, padding: '2rem', width: 360, boxShadow: '0 2px 16px rgba(0,0,0,0.1)' };
const title: React.CSSProperties = { marginBottom: '1.5rem', fontSize: '1.5rem', textAlign: 'center' };
const form: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: '0.75rem' };
const input: React.CSSProperties = { padding: '0.625rem', borderRadius: 6, border: '1px solid #ddd', fontSize: '1rem' };
const btn: React.CSSProperties = { padding: '0.75rem', background: '#0066ff', color: '#fff', border: 'none', borderRadius: 6, fontSize: '1rem', cursor: 'pointer' };
const link: React.CSSProperties = { background: 'none', border: 'none', color: '#0066ff', cursor: 'pointer', fontSize: '0.875rem' };
const center: React.CSSProperties = { textAlign: 'center', color: '#555', marginBottom: '1rem' };

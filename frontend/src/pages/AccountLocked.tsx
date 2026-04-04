import { useState } from 'react';
import { api } from '../api';

interface Props { onBack: () => void; }

export default function AccountLocked({ onBack }: Props) {
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);

  async function handleResend(e: React.FormEvent) {
    e.preventDefault();
    await api.resendVerification(email).catch(() => {});
    setSent(true);
  }

  return (
    <div style={card}>
      <h1 style={title}>Account locked</h1>
      <p style={text}>Your account has been temporarily locked due to multiple failed login attempts. It will unlock automatically in 30 minutes.</p>
      {!sent ? (
        <form onSubmit={handleResend} style={form}>
          <input style={input} type="email" placeholder="Your email" value={email} onChange={e => setEmail(e.target.value)} required />
          <button style={btn} type="submit">Request unlock email</button>
        </form>
      ) : (
        <p style={{ ...text, color: '#090' }}>If your account exists, you'll receive an unlock email shortly.</p>
      )}
      <button style={link} onClick={onBack}>Back to sign in</button>
    </div>
  );
}

const card: React.CSSProperties = { background: '#fff', borderRadius: 12, padding: '2rem', width: 360, boxShadow: '0 2px 16px rgba(0,0,0,0.1)' };
const title: React.CSSProperties = { marginBottom: '1rem', fontSize: '1.5rem', textAlign: 'center' };
const text: React.CSSProperties = { color: '#555', fontSize: '0.9rem', marginBottom: '1rem' };
const form: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: '0.75rem', marginBottom: '1rem' };
const input: React.CSSProperties = { padding: '0.625rem', borderRadius: 6, border: '1px solid #ddd', fontSize: '1rem' };
const btn: React.CSSProperties = { padding: '0.75rem', background: '#0066ff', color: '#fff', border: 'none', borderRadius: 6, fontSize: '1rem', cursor: 'pointer' };
const link: React.CSSProperties = { display: 'block', background: 'none', border: 'none', color: '#0066ff', cursor: 'pointer', fontSize: '0.875rem', marginTop: '0.5rem' };

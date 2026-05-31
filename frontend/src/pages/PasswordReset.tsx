import { useState } from 'react';
import { api } from '../api';
import { card, title, form, input, btn, link, centerText } from '../styles';

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

  if (sent) {
    return (
      <div style={card}>
        <div style={brand}>braza</div>
        <h1 style={title}>Check your email</h1>
        <p style={{ ...centerText, marginBottom: '1.5rem' }}>
          If an account exists for <strong style={{ color: 'var(--text)' }}>{email}</strong>, you'll receive a reset link shortly.
        </p>
        <button style={btn} onClick={onBack}>Back to sign in</button>
      </div>
    );
  }

  return (
    <div style={card}>
      <div style={brand}>braza</div>
      <h1 style={title}>Reset password</h1>
      <form onSubmit={handleSubmit} style={form}>
        <input style={input} type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required />
        <button style={btn} type="submit" disabled={loading}>{loading ? 'Sending…' : 'Send reset link'}</button>
      </form>
      <div style={{ textAlign: 'center', marginTop: '1.25rem' }}>
        <button style={link} onClick={onBack}>Back to sign in</button>
      </div>
    </div>
  );
}

const brand: React.CSSProperties = {
  textAlign: 'center', fontSize: '1.5rem', fontWeight: 700,
  color: 'var(--accent)', letterSpacing: '-0.04em', marginBottom: '1.5rem',
};

import { useState } from 'react';
import { api } from '../api';
import { card, title, input, btn, link } from '../styles';

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
      <div style={brand}>braza</div>
      <h1 style={title}>Account locked</h1>
      <p style={text}>
        Your account has been temporarily locked due to multiple failed login attempts. It will unlock automatically in 30 minutes.
      </p>
      {!sent ? (
        <form onSubmit={handleResend} style={inlineForm}>
          <input style={input} type="email" placeholder="Your email" value={email} onChange={e => setEmail(e.target.value)} required />
          <button style={btn} type="submit">Request unlock email</button>
        </form>
      ) : (
        <p style={{ ...text, color: 'var(--success)' }}>If your account exists, you'll receive an unlock email shortly.</p>
      )}
      <button style={{ ...link, display: 'block', marginTop: '0.75rem' }} onClick={onBack}>Back to sign in</button>
    </div>
  );
}

const brand: React.CSSProperties = {
  textAlign: 'center', fontSize: '1.5rem', fontWeight: 700,
  color: 'var(--accent)', letterSpacing: '-0.04em', marginBottom: '1.5rem',
};
const text: React.CSSProperties = {
  color: 'var(--text-muted)', fontSize: '0.9rem', marginBottom: '1.25rem', lineHeight: 1.6,
};
const inlineForm: React.CSSProperties = {
  display: 'flex', flexDirection: 'column', gap: '0.75rem', marginBottom: '0.5rem',
};

import { useState } from 'react';
import { api } from '../api';
import { card, title, input, btn, link } from '../styles';

interface Props { onBack: () => void; }

const text: React.CSSProperties = { color: '#555', fontSize: '0.9rem', marginBottom: '1rem' };
const inlineForm: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: '0.75rem', marginBottom: '1rem' };

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
        <form onSubmit={handleResend} style={inlineForm}>
          <input style={input} type="email" placeholder="Your email" value={email} onChange={e => setEmail(e.target.value)} required />
          <button style={btn} type="submit">Request unlock email</button>
        </form>
      ) : (
        <p style={{ ...text, color: '#090' }}>If your account exists, you'll receive an unlock email shortly.</p>
      )}
      <button style={{ ...link, display: 'block', marginTop: '0.5rem' }} onClick={onBack}>Back to sign in</button>
    </div>
  );
}

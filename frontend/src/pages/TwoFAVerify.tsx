import { useState } from 'react';
import { api } from '../api';
import { card, title, form, input, btn, link, errorStyle } from '../styles';

interface Props { sessionId: string; onDone: () => void; }

export default function TwoFAVerify({ sessionId, onDone }: Props) {
  const [code, setCode] = useState('');
  const [recovery, setRecovery] = useState('');
  const [mode, setMode] = useState<'totp' | 'recovery'>('totp');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      if (mode === 'totp') {
        await api.mfaVerify(sessionId, code);
      } else {
        await api.mfaRecovery(sessionId, recovery);
      }
      window.location.href = '/';
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Verification failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={card}>
      <div style={brand}>braza</div>
      <h1 style={title}>Two-factor auth</h1>
      <form onSubmit={handleSubmit} style={form}>
        {mode === 'totp' ? (
          <input style={input} type="text" placeholder="6-digit code" inputMode="numeric" maxLength={6} value={code} onChange={e => setCode(e.target.value)} required />
        ) : (
          <input style={input} type="text" placeholder="Recovery code" value={recovery} onChange={e => setRecovery(e.target.value)} required />
        )}
        {error && <p style={errorStyle}>{error}</p>}
        <button style={btn} type="submit" disabled={loading}>{loading ? 'Verifying…' : 'Verify'}</button>
      </form>
      <div style={{ textAlign: 'center', marginTop: '1rem' }}>
        <button style={link} onClick={() => setMode(mode === 'totp' ? 'recovery' : 'totp')}>
          {mode === 'totp' ? 'Use a recovery code' : 'Use authenticator app'}
        </button>
      </div>
      <div style={{ textAlign: 'center', marginTop: '0.5rem' }}>
        <button style={link} onClick={onDone}>Back to sign in</button>
      </div>
    </div>
  );
}

const brand: React.CSSProperties = {
  textAlign: 'center', fontSize: '1.5rem', fontWeight: 700,
  color: 'var(--accent)', letterSpacing: '-0.04em', marginBottom: '1.5rem',
};

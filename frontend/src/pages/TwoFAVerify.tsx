import { useState } from 'react';
import { api } from '../api';

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
      let res;
      if (mode === 'totp') {
        res = await api.mfaVerify(sessionId, code);
      } else {
        res = await api.mfaRecovery(sessionId, recovery);
      }
      if (res.access_token) {
        localStorage.setItem('access_token', res.access_token);
        window.location.href = '/';
      }
    } catch (err: any) {
      setError(err.message ?? 'Verification failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={card}>
      <h1 style={title}>Two-factor authentication</h1>
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
    </div>
  );
}

const card: React.CSSProperties = { background: '#fff', borderRadius: 12, padding: '2rem', width: 360, boxShadow: '0 2px 16px rgba(0,0,0,0.1)' };
const title: React.CSSProperties = { marginBottom: '1.5rem', fontSize: '1.5rem', textAlign: 'center' };
const form: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: '0.75rem' };
const input: React.CSSProperties = { padding: '0.625rem', borderRadius: 6, border: '1px solid #ddd', fontSize: '1rem' };
const btn: React.CSSProperties = { padding: '0.75rem', background: '#0066ff', color: '#fff', border: 'none', borderRadius: 6, fontSize: '1rem', cursor: 'pointer' };
const link: React.CSSProperties = { background: 'none', border: 'none', color: '#0066ff', cursor: 'pointer', fontSize: '0.875rem' };
const errorStyle: React.CSSProperties = { color: '#c00', fontSize: '0.875rem' };

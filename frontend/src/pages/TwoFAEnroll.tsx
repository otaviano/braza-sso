import { useState, useEffect } from 'react';
import { api } from '../api';

interface Props { onDone: () => void; }

export default function TwoFAEnroll({ onDone }: Props) {
  const [data, setData] = useState<{ secret: string; otp_uri: string; recovery_codes: string[] } | null>(null);
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [step, setStep] = useState<'scan' | 'confirm' | 'codes'>('scan');
  const [loading, setLoading] = useState(false);
  const token = localStorage.getItem('access_token') ?? '';

  useEffect(() => {
    api.enroll2fa(token).then(setData).catch(e => setError(e.message));
  }, []);

  async function handleConfirm(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await api.confirm2fa(token, code);
      setStep('codes');
    } catch (err: any) {
      setError(err.message ?? 'Confirmation failed');
    } finally {
      setLoading(false);
    }
  }

  if (!data) return <div style={card}><p style={{ textAlign: 'center' }}>{error || 'Loading…'}</p></div>;

  if (step === 'codes') return (
    <div style={card}>
      <h1 style={title}>Save your recovery codes</h1>
      <p style={{ color: '#555', fontSize: '0.875rem', marginBottom: '1rem' }}>Store these codes safely. Each can only be used once.</p>
      <ul style={{ listStyle: 'none', padding: 0, fontFamily: 'monospace', fontSize: '0.875rem', lineHeight: 2 }}>
        {data.recovery_codes.map(c => <li key={c}>{c}</li>)}
      </ul>
      <button style={btn} onClick={onDone}>Done</button>
    </div>
  );

  if (step === 'scan') return (
    <div style={card}>
      <h1 style={title}>Set up authenticator</h1>
      <p style={{ color: '#555', fontSize: '0.875rem' }}>Scan this QR code with your authenticator app, or enter the secret manually.</p>
      <img src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(data.otp_uri)}`} alt="QR Code" style={{ display: 'block', margin: '1rem auto' }} />
      <p style={{ fontFamily: 'monospace', textAlign: 'center', fontSize: '0.8rem', wordBreak: 'break-all' }}>{data.secret}</p>
      <button style={btn} onClick={() => setStep('confirm')}>Next</button>
    </div>
  );

  return (
    <div style={card}>
      <h1 style={title}>Confirm setup</h1>
      <p style={{ color: '#555', fontSize: '0.875rem' }}>Enter the 6-digit code from your authenticator app.</p>
      <form onSubmit={handleConfirm} style={form}>
        <input style={input} type="text" placeholder="6-digit code" inputMode="numeric" maxLength={6} value={code} onChange={e => setCode(e.target.value)} required />
        {error && <p style={errorStyle}>{error}</p>}
        <button style={btn} type="submit" disabled={loading}>{loading ? 'Confirming…' : 'Confirm'}</button>
      </form>
    </div>
  );
}

const card: React.CSSProperties = { background: '#fff', borderRadius: 12, padding: '2rem', width: 380, boxShadow: '0 2px 16px rgba(0,0,0,0.1)' };
const title: React.CSSProperties = { marginBottom: '1rem', fontSize: '1.5rem', textAlign: 'center' };
const form: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: '0.75rem' };
const input: React.CSSProperties = { padding: '0.625rem', borderRadius: 6, border: '1px solid #ddd', fontSize: '1rem' };
const btn: React.CSSProperties = { padding: '0.75rem', background: '#0066ff', color: '#fff', border: 'none', borderRadius: 6, fontSize: '1rem', cursor: 'pointer', marginTop: '0.5rem' };
const errorStyle: React.CSSProperties = { color: '#c00', fontSize: '0.875rem' };

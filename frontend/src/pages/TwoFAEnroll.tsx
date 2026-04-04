import { useState, useEffect } from 'react';
import QRCode from 'qrcode';
import { api } from '../api';
import { card, title, form, input, btn, errorStyle } from '../styles';

interface Props { onDone: () => void; }

type EnrollData = { secret: string; otp_uri: string; recovery_codes: string[] };

export default function TwoFAEnroll({ onDone }: Props) {
  const [data, setData] = useState<EnrollData | null>(null);
  const [qrDataUrl, setQrDataUrl] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [step, setStep] = useState<'scan' | 'confirm' | 'codes'>('scan');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api.enroll2fa()
      .then(async (enrollData) => {
        setData(enrollData);
        const dataUrl = await QRCode.toDataURL(enrollData.otp_uri);
        setQrDataUrl(dataUrl);
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : 'Failed to start enrollment');
      });
  }, []);

  async function handleConfirm(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await api.confirm2fa(code);
      setStep('codes');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Confirmation failed');
    } finally {
      setLoading(false);
    }
  }

  if (!data) {
    return <div style={card}><p style={{ textAlign: 'center' }}>{error || 'Loading…'}</p></div>;
  }

  if (step === 'codes') {
    return (
      <div style={card}>
        <h1 style={title}>Save your recovery codes</h1>
        <p style={{ color: '#555', fontSize: '0.875rem', marginBottom: '1rem' }}>Store these codes safely. Each can only be used once.</p>
        <ul style={{ listStyle: 'none', padding: 0, fontFamily: 'monospace', fontSize: '0.875rem', lineHeight: 2 }}>
          {data.recovery_codes.map(c => <li key={c}>{c}</li>)}
        </ul>
        <button style={btn} onClick={onDone}>Done</button>
      </div>
    );
  }

  if (step === 'scan') {
    return (
      <div style={{ ...card, width: 380 }}>
        <h1 style={title}>Set up authenticator</h1>
        <p style={{ color: '#555', fontSize: '0.875rem' }}>Scan this QR code with your authenticator app, or enter the secret manually.</p>
        {qrDataUrl && <img src={qrDataUrl} alt="QR Code" style={{ display: 'block', margin: '1rem auto' }} />}
        <p style={{ fontFamily: 'monospace', textAlign: 'center', fontSize: '0.8rem', wordBreak: 'break-all' }}>{data.secret}</p>
        <button style={{ ...btn, marginTop: '0.5rem' }} onClick={() => setStep('confirm')}>Next</button>
      </div>
    );
  }

  return (
    <div style={{ ...card, width: 380 }}>
      <h1 style={title}>Confirm setup</h1>
      <p style={{ color: '#555', fontSize: '0.875rem' }}>Enter the 6-digit code from your authenticator app.</p>
      <form onSubmit={handleConfirm} style={form}>
        <input style={input} type="text" placeholder="6-digit code" inputMode="numeric" maxLength={6} value={code} onChange={e => setCode(e.target.value)} required />
        {error && <p style={errorStyle}>{error}</p>}
        <button style={{ ...btn, marginTop: '0.5rem' }} type="submit" disabled={loading}>{loading ? 'Confirming…' : 'Confirm'}</button>
      </form>
    </div>
  );
}

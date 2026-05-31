import { useEffect, useState } from 'react';
import { api } from '../api';
import { card, title, centerText, btn } from '../styles';

export default function VerifyEmail() {
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');

  useEffect(() => {
    const token = new URLSearchParams(window.location.search).get('token') ?? '';
    api.verifyEmail(token)
      .then(() => setStatus('success'))
      .catch(() => setStatus('error'));
  }, []);

  return (
    <div style={card}>
      <div style={brand}>braza</div>
      {status === 'loading' && <p style={centerText}>Verifying…</p>}
      {status === 'success' && (
        <>
          <h1 style={title}>Email verified!</h1>
          <p style={{ ...centerText, marginBottom: '1.5rem' }}>Your account is now active.</p>
          <a href="/" style={{ ...btn, display: 'block', textAlign: 'center', textDecoration: 'none' }}>Sign in</a>
        </>
      )}
      {status === 'error' && (
        <>
          <h1 style={title}>Verification failed</h1>
          <p style={{ ...centerText, marginBottom: '1.5rem' }}>This link may have expired. Request a new one on the sign-in page.</p>
          <a href="/" style={{ ...btn, display: 'block', textAlign: 'center', textDecoration: 'none' }}>Back to sign in</a>
        </>
      )}
    </div>
  );
}

const brand: React.CSSProperties = {
  textAlign: 'center', fontSize: '1.5rem', fontWeight: 700,
  color: 'var(--accent)', letterSpacing: '-0.04em', marginBottom: '1.5rem',
};

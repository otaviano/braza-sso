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
      {status === 'loading' && <p style={centerText}>Verifying…</p>}
      {status === 'success' && (
        <>
          <h1 style={title}>Email verified!</h1>
          <p style={centerText}>Your account is now active.</p>
          <a href="/" style={{ ...btn, display: 'block', marginTop: '1.5rem', textAlign: 'center', textDecoration: 'none' }}>Sign in</a>
        </>
      )}
      {status === 'error' && (
        <>
          <h1 style={title}>Verification failed</h1>
          <p style={centerText}>This link may have expired. Request a new one on the sign-in page.</p>
          <a href="/" style={{ ...btn, display: 'block', marginTop: '1.5rem', textAlign: 'center', textDecoration: 'none' }}>Back to sign in</a>
        </>
      )}
    </div>
  );
}

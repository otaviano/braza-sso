import { useEffect, useState } from 'react';
import { api } from '../api';

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
      {status === 'loading' && <p style={center}>Verifying…</p>}
      {status === 'success' && (
        <>
          <h1 style={title}>Email verified!</h1>
          <p style={center}>Your account is now active.</p>
          <a href="/" style={btn}>Sign in</a>
        </>
      )}
      {status === 'error' && (
        <>
          <h1 style={title}>Verification failed</h1>
          <p style={center}>This link may have expired. Request a new one on the sign-in page.</p>
          <a href="/" style={btn}>Back to sign in</a>
        </>
      )}
    </div>
  );
}

const card: React.CSSProperties = { background: '#fff', borderRadius: 12, padding: '2rem', width: 360, boxShadow: '0 2px 16px rgba(0,0,0,0.1)' };
const title: React.CSSProperties = { marginBottom: '1rem', fontSize: '1.5rem', textAlign: 'center' };
const center: React.CSSProperties = { textAlign: 'center', color: '#555' };
const btn: React.CSSProperties = { display: 'block', marginTop: '1.5rem', padding: '0.75rem', background: '#0066ff', color: '#fff', border: 'none', borderRadius: 6, fontSize: '1rem', textAlign: 'center', textDecoration: 'none' };

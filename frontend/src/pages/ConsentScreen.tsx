import { useState } from 'react';

export default function ConsentScreen() {
  const params = new URLSearchParams(window.location.search);
  const clientId = params.get('client_id') ?? '';
  const scope = params.get('scope') ?? 'openid email';
  const scopes = scope.split(' ');
  const [loading, setLoading] = useState(false);

  function submitConsentForm(approved: boolean) {
    setLoading(true);
    const consentForm = document.createElement('form');
    consentForm.method = 'POST';
    consentForm.action = '/api/oauth/consent';
    for (const [key, value] of params.entries()) {
      const hiddenInput = document.createElement('input');
      hiddenInput.type = 'hidden';
      hiddenInput.name = key;
      hiddenInput.value = value;
      consentForm.appendChild(hiddenInput);
    }
    const approvedInput = document.createElement('input');
    approvedInput.type = 'hidden';
    approvedInput.name = 'approved';
    approvedInput.value = String(approved);
    consentForm.appendChild(approvedInput);
    document.body.appendChild(consentForm);
    consentForm.submit();
  }

  return (
    <div style={card}>
      <h1 style={title}>Authorize access</h1>
      <p style={text}><strong>{clientId}</strong> is requesting access to your account.</p>
      <div style={{ background: '#f5f5f5', borderRadius: 8, padding: '1rem', marginBottom: '1.5rem' }}>
        <p style={{ margin: 0, fontWeight: 600, marginBottom: '0.5rem' }}>Permissions requested:</p>
        <ul style={{ margin: 0, paddingLeft: '1.25rem' }}>
          {scopes.map(s => <li key={s} style={{ fontSize: '0.875rem', color: '#333', marginBottom: '0.25rem' }}>{s}</li>)}
        </ul>
      </div>
      <div style={{ display: 'flex', gap: '0.75rem' }}>
        <button style={denyBtn} onClick={() => submitConsentForm(false)} disabled={loading}>Deny</button>
        <button style={allowBtn} onClick={() => submitConsentForm(true)} disabled={loading}>{loading ? 'Authorizing…' : 'Allow'}</button>
      </div>
    </div>
  );
}

const card: React.CSSProperties = { background: '#fff', borderRadius: 12, padding: '2rem', width: 400, boxShadow: '0 2px 16px rgba(0,0,0,0.1)' };
const title: React.CSSProperties = { marginBottom: '1rem', fontSize: '1.5rem', textAlign: 'center' };
const text: React.CSSProperties = { color: '#555', marginBottom: '1.5rem' };
const allowBtn: React.CSSProperties = { flex: 1, padding: '0.75rem', background: '#0066ff', color: '#fff', border: 'none', borderRadius: 6, fontSize: '1rem', cursor: 'pointer' };
const denyBtn: React.CSSProperties = { flex: 1, padding: '0.75rem', background: '#fff', color: '#333', border: '1px solid #ddd', borderRadius: 6, fontSize: '1rem', cursor: 'pointer' };

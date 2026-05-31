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
    consentForm.action = '/oauth/consent';
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
      <div style={brand}>braza</div>
      <h1 style={cardTitle}>Authorize access</h1>
      <p style={description}>
        <strong style={{ color: 'var(--text)' }}>{clientId}</strong> is requesting access to your account.
      </p>
      <div style={scopeBox}>
        <p style={scopeHeader}>Permissions requested:</p>
        <ul style={scopeList}>
          {scopes.map(s => <li key={s} style={scopeItem}>{s}</li>)}
        </ul>
      </div>
      <div style={actions}>
        <button style={denyBtn} onClick={() => submitConsentForm(false)} disabled={loading}>Deny</button>
        <button style={allowBtn} onClick={() => submitConsentForm(true)} disabled={loading}>
          {loading ? 'Authorizing…' : 'Allow'}
        </button>
      </div>
    </div>
  );
}

const card: React.CSSProperties = {
  background: 'var(--surface)', borderRadius: 16, padding: '2.5rem', width: 400,
  border: '1px solid var(--border)', boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
};
const brand: React.CSSProperties = {
  textAlign: 'center', fontSize: '1.5rem', fontWeight: 700,
  color: 'var(--accent)', letterSpacing: '-0.04em', marginBottom: '1.5rem',
};
const cardTitle: React.CSSProperties = {
  marginBottom: '0.75rem', fontSize: '1.375rem', fontWeight: 600,
  textAlign: 'center', color: 'var(--text)', letterSpacing: '-0.02em',
};
const description: React.CSSProperties = {
  color: 'var(--text-muted)', textAlign: 'center', marginBottom: '1.25rem', fontSize: '0.9rem',
};
const scopeBox: React.CSSProperties = {
  background: 'var(--surface-2)', borderRadius: 8, padding: '1rem',
  marginBottom: '1.5rem', border: '1px solid var(--border)',
};
const scopeHeader: React.CSSProperties = {
  margin: '0 0 0.5rem', fontWeight: 600, fontSize: '0.875rem', color: 'var(--text)',
};
const scopeList: React.CSSProperties = { margin: 0, paddingLeft: '1.25rem' };
const scopeItem: React.CSSProperties = {
  fontSize: '0.875rem', color: 'var(--text-muted)', marginBottom: '0.2rem',
};
const actions: React.CSSProperties = { display: 'flex', gap: '0.75rem' };
const denyBtn: React.CSSProperties = {
  flex: 1, padding: '0.75rem', background: 'var(--surface-2)',
  color: 'var(--text-muted)', border: '1px solid var(--border)',
  borderRadius: 8, fontSize: '0.9375rem', cursor: 'pointer',
};
const allowBtn: React.CSSProperties = {
  flex: 1, padding: '0.75rem', background: 'var(--accent)',
  color: '#fff', border: 'none', borderRadius: 8,
  fontSize: '0.9375rem', fontWeight: 500, cursor: 'pointer',
};

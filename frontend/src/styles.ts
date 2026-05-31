export const card: React.CSSProperties = {
  background: 'var(--surface)',
  borderRadius: 16,
  padding: '2.5rem',
  width: 380,
  border: '1px solid var(--border)',
  boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
};

export const title: React.CSSProperties = {
  marginBottom: '1.75rem',
  fontSize: '1.375rem',
  fontWeight: 600,
  textAlign: 'center',
  color: 'var(--text)',
  letterSpacing: '-0.02em',
};

export const form: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: '0.875rem',
};

export const input: React.CSSProperties = {
  padding: '0.75rem 1rem',
  borderRadius: 8,
  border: '1px solid var(--border)',
  background: 'var(--surface-2)',
  color: 'var(--text)',
  fontSize: '0.9375rem',
  outline: 'none',
  width: '100%',
  transition: 'border-color 0.15s',
};

export const btn: React.CSSProperties = {
  padding: '0.75rem',
  background: 'var(--accent)',
  color: '#fff',
  border: 'none',
  borderRadius: 8,
  fontSize: '0.9375rem',
  fontWeight: 500,
  cursor: 'pointer',
  width: '100%',
  transition: 'background 0.15s',
};

export const link: React.CSSProperties = {
  background: 'none',
  border: 'none',
  color: 'var(--text-muted)',
  cursor: 'pointer',
  fontSize: '0.875rem',
  textDecoration: 'none',
  transition: 'color 0.15s',
};

export const errorStyle: React.CSSProperties = {
  color: 'var(--error)',
  fontSize: '0.8125rem',
  textAlign: 'center',
};

export const centerText: React.CSSProperties = {
  textAlign: 'center',
  color: 'var(--text-muted)',
  fontSize: '0.9rem',
  lineHeight: 1.6,
};

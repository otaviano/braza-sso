import { useState } from 'react';
import { api } from '../api';
import { card, title, form, input, btn, errorStyle } from '../styles';

interface Props { onDone: () => void; }

export default function PasswordResetConfirm({ onDone }: Props) {
  const token = new URLSearchParams(window.location.search).get('token') ?? '';
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await api.resetPassword(token, password);
      onDone();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Reset failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={card}>
      <h1 style={title}>Set new password</h1>
      <form onSubmit={handleSubmit} style={form}>
        <input style={input} type="password" placeholder="New password (min 12 chars)" value={password} onChange={e => setPassword(e.target.value)} required />
        {error && <p style={errorStyle}>{error}</p>}
        <button style={btn} type="submit" disabled={loading}>{loading ? 'Updating…' : 'Update password'}</button>
      </form>
    </div>
  );
}

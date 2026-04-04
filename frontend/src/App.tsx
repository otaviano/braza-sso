import { useState } from 'react';
import Login from './pages/Login';
import Register from './pages/Register';
import VerifyEmail from './pages/VerifyEmail';
import PasswordReset from './pages/PasswordReset';
import PasswordResetConfirm from './pages/PasswordResetConfirm';
import TwoFAVerify from './pages/TwoFAVerify';
import TwoFAEnroll from './pages/TwoFAEnroll';
import AccountLocked from './pages/AccountLocked';
import ConsentScreen from './pages/ConsentScreen';

type Page =
  | 'login'
  | 'register'
  | 'verify-email'
  | 'reset-password'
  | 'reset-password-confirm'
  | '2fa-verify'
  | '2fa-enroll'
  | 'account-locked'
  | 'consent';

function getInitialPage(): Page {
  const path = window.location.pathname;
  const params = new URLSearchParams(window.location.search);
  if (path.includes('verify-email')) return 'verify-email';
  if (path.includes('reset-password') && params.has('token')) return 'reset-password-confirm';
  if (path.includes('reset-password')) return 'reset-password';
  if (path.includes('2fa-enroll')) return '2fa-enroll';
  if (path.includes('2fa-verify')) return '2fa-verify';
  if (path.includes('register')) return 'register';
  if (path.includes('locked')) return 'account-locked';
  if (path.includes('consent')) return 'consent';
  return 'login';
}

const layout: React.CSSProperties = {
  minHeight: '100vh',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  background: '#f5f5f5',
};

export default function App() {
  const [page, setPage] = useState<Page>(getInitialPage);
  const [mfaSession, setMfaSession] = useState('');

  function navigateToLogin() {
    setMfaSession('');
    setPage('login');
  }

  function handleMFARequired(sessionId: string) {
    setMfaSession(sessionId);
    setPage('2fa-verify');
  }

  return (
    <div style={layout}>
      {page === 'login' && (
        <Login
          onRegister={() => setPage('register')}
          onForgotPassword={() => setPage('reset-password')}
          onMFARequired={handleMFARequired}
        />
      )}
      {page === 'register' && <Register onLogin={navigateToLogin} />}
      {page === 'verify-email' && <VerifyEmail />}
      {page === 'reset-password' && <PasswordReset onBack={navigateToLogin} />}
      {page === 'reset-password-confirm' && <PasswordResetConfirm onDone={navigateToLogin} />}
      {page === '2fa-verify' && <TwoFAVerify sessionId={mfaSession} onDone={navigateToLogin} />}
      {page === '2fa-enroll' && <TwoFAEnroll onDone={navigateToLogin} />}
      {page === 'account-locked' && <AccountLocked onBack={navigateToLogin} />}
      {page === 'consent' && <ConsentScreen />}
    </div>
  );
}

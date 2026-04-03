package email

import (
	"fmt"
	"net/smtp"
)

// Sender is the interface for sending emails.
type Sender interface {
	SendVerification(to, verifyURL string) error
	SendPasswordReset(to, resetURL string) error
	SendAccountLocked(to, unlockURL string) error
}

// SMTPSender sends emails via SMTP.
type SMTPSender struct {
	host string
	port int
	user string
	pass string
	from string
}

func NewSMTPSender(host string, port int, user, pass, from string) *SMTPSender {
	return &SMTPSender{host: host, port: port, user: user, pass: pass, from: from}
}

func (s *SMTPSender) send(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body)
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}

func (s *SMTPSender) SendVerification(to, verifyURL string) error {
	body := fmt.Sprintf(`<p>Verify your email address by clicking the link below:</p>
<p><a href="%s">Verify Email</a></p>
<p>This link expires in 24 hours.</p>`, verifyURL)
	return s.send(to, "Verify your Braza SSO email", body)
}

func (s *SMTPSender) SendPasswordReset(to, resetURL string) error {
	body := fmt.Sprintf(`<p>Reset your password by clicking the link below:</p>
<p><a href="%s">Reset Password</a></p>
<p>This link expires in 1 hour. If you did not request a reset, ignore this email.</p>`, resetURL)
	return s.send(to, "Reset your Braza SSO password", body)
}

func (s *SMTPSender) SendAccountLocked(to, unlockURL string) error {
	body := fmt.Sprintf(`<p>Your account has been temporarily locked due to multiple failed login attempts.</p>
<p>It will unlock automatically in 30 minutes, or you can <a href="%s">request an unlock</a>.</p>`, unlockURL)
	return s.send(to, "Braza SSO account locked", body)
}

// Package mailer sends HTML emails via SMTP.
// Compatible with any SMTP provider (Yandex, Mail.ru, SMTP2Go, MailerSend, etc.)
// and with Resend's SMTP relay (smtp.resend.com:465, user=resend, pass=API_KEY).
package mailer

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// Config holds SMTP connection parameters.
type Config struct {
	Host      string // e.g. "smtp.resend.com"
	Port      string // e.g. "465" (TLS) or "587" (STARTTLS)
	Username  string
	Password  string
	FromEmail string
	FromName  string
	SiteURL   string
}

// Mailer sends transactional emails via SMTP.
type Mailer struct {
	cfg Config
}

func New(cfg Config) *Mailer {
	return &Mailer{cfg: cfg}
}

func (m *Mailer) from() string {
	return fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.FromEmail)
}

// send delivers an HTML email.
func (m *Mailer) send(to, subject, html string) error {
	msg := m.buildMessage(to, subject, html)

	addr := m.cfg.Host + ":" + m.cfg.Port
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)

	if m.cfg.Port == "465" {
		return m.sendTLS(addr, auth, to, msg)
	}
	return smtp.SendMail(addr, auth, m.cfg.FromEmail, []string{to}, []byte(msg))
}

func (m *Mailer) sendTLS(addr string, auth smtp.Auth, to, msg string) error {
	tlsCfg := &tls.Config{ServerName: m.cfg.Host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer c.Close()

	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := c.Mail(m.cfg.FromEmail); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer w.Close()
	_, err = fmt.Fprint(w, msg)
	return err
}

func (m *Mailer) buildMessage(to, subject, html string) string {
	var sb strings.Builder
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	sb.WriteString("From: " + m.from() + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(html)
	return sb.String()
}

// SendRegistration sends OTP and optional plain password to the participant.
func (m *Mailer) SendRegistration(
	to, firstName, otp string,
	password *string,
	loginURL, eventTitle string,
) error {
	name := firstName
	if name == "" {
		name = "участник"
	}

	passwordBlock := ""
	if password != nil && *password != "" {
		passwordBlock = fmt.Sprintf(
			`<p><strong>Пароль для личного кабинета:</strong> %s</p>
			 <p>Ссылка для входа: <a href="%s">%s</a></p>`,
			*password, loginURL, loginURL,
		)
	}

	html := fmt.Sprintf(`
		<p>Здравствуйте, %s!</p>
		<p>Вы начали регистрацию на мероприятие:</p>
		<p><strong>%s</strong></p>
		<p><strong>Код подтверждения (OTP):</strong> %s</p>
		%s
		<p>Если вы не запрашивали регистрацию, просто проигнорируйте это письмо.</p>
	`, name, eventTitle, otp, passwordBlock)

	return m.send(to, "Подтверждение регистрации", html)
}

// SendNewPassword mails a newly generated password.
func (m *Mailer) SendNewPassword(to, firstName, password, loginURL string) error {
	name := firstName
	if name == "" {
		name = "участник"
	}
	html := fmt.Sprintf(`
		<p>Здравствуйте, %s!</p>
		<p>Ваш новый пароль для входа в личный кабинет:</p>
		<p><strong>%s</strong></p>
		<p>Войти: <a href="%s">%s</a></p>
		<p>Если вы не запрашивали сброс пароля, проигнорируйте это письмо.</p>
	`, name, password, loginURL, loginURL)
	return m.send(to, "Новый пароль для личного кабинета", html)
}

// SendTicket sends a ticket confirmation with the QR link for offline events.
func (m *Mailer) SendTicket(to, firstName, eventTitle, ticketPageURL string) error {
	name := firstName
	if name == "" {
		name = "участник"
	}
	html := fmt.Sprintf(`
		<p>Здравствуйте, %s!</p>
		<p>Ваше участие в мероприятии <strong>%s</strong> подтверждено.</p>
		<p>Ваш билет с QR-кодом для входа:</p>
		<p><a href="%s" style="display:inline-block;padding:12px 24px;background:linear-gradient(135deg,#ff7c2c,#009b35);color:#fff;text-decoration:none;border-radius:8px;font-weight:bold">Открыть билет / QR</a></p>
		<p style="color:#888;font-size:13px">Покажите QR-код на входе в место проведения мероприятия.</p>
	`, name, eventTitle, ticketPageURL)
	return m.send(to, "Ваш билет на мероприятие: "+eventTitle, html)
}

// SendAdminOTP mails an OTP code for admin login.
func (m *Mailer) SendAdminOTP(to, name, otp string) error {
	html := fmt.Sprintf(`
		<p>Здравствуйте, %s!</p>
		<p>Код подтверждения для входа в панель администратора:</p>
		<p><strong>%s</strong></p>
		<p>Код действителен 10 минут.</p>
	`, name, otp)
	return m.send(to, "Код входа в панель администратора", html)
}

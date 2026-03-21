// Package mailer sends HTML emails via SMTP.
// Compatible with any SMTP provider (Yandex, Mail.ru, SMTP2Go, MailerSend, etc.)
// and with Resend's SMTP relay (smtp.resend.com:465, user=resend, pass=API_KEY).
//
// # Scalability design
//
// For high-concurrency registration bursts (e.g. 2000 registrations in 30 minutes)
// the Mailer runs a background worker pool so that HTTP handlers never block on SMTP.
//
//   - A buffered channel (default cap 500) decouples producers from SMTP workers.
//   - Workers (default 5) drain the channel concurrently, each opening its own TCP
//     connection to the SMTP server when needed.
//   - If the channel is full the job is dropped and the error is logged — the
//     registration is already persisted in the DB and the user can request a resend.
//   - Critical-path emails (admin OTP, password-reset) use sendSync which blocks
//     the caller until the SMTP handshake completes (these are low-volume paths).
package mailer

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"sync"
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

type mailJob struct {
	to      string
	subject string
	html    string
	// errCh is non-nil only for synchronous callers.
	errCh chan error
}

// Mailer sends transactional emails via SMTP with an async worker pool.
type Mailer struct {
	cfg    Config
	jobCh  chan mailJob
	once   sync.Once
	wg     sync.WaitGroup
	logger interface {
		Printf(format string, args ...interface{})
	}
}

// defaultLogger is used when no logger is injected.
type stdLogger struct{}

func (stdLogger) Printf(format string, args ...interface{}) {
	fmt.Printf("[mailer] "+format+"\n", args...)
}

// New creates a Mailer and starts `workers` background goroutines.
// bufferSize is the number of queued jobs before SendAsync blocks/drops.
// Typical values: workers=5, bufferSize=500.
func New(cfg Config) *Mailer {
	return NewWithPool(cfg, 5, 500)
}

// NewWithPool lets callers tune the pool size (useful for tests).
func NewWithPool(cfg Config, workers, bufferSize int) *Mailer {
	m := &Mailer{
		cfg:    cfg,
		jobCh:  make(chan mailJob, bufferSize),
		logger: stdLogger{},
	}
	m.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go m.worker()
	}
	return m
}

// IsDevMode returns true when SMTP password is not configured.
// In dev mode the caller may choose to log OTP instead of returning an error.
func (m *Mailer) IsDevMode() bool {
	return m.cfg.Password == ""
}

// Close drains remaining jobs and waits for all workers to finish.
// Call this during graceful shutdown.
func (m *Mailer) Close() {
	m.once.Do(func() { close(m.jobCh) })
	m.wg.Wait()
}

func (m *Mailer) worker() {
	defer m.wg.Done()
	for job := range m.jobCh {
		err := m.sendSync(job.to, job.subject, job.html)
		if job.errCh != nil {
			job.errCh <- err
		} else if err != nil {
			m.logger.Printf("async send failed to=%s subject=%q err=%v", job.to, job.subject, err)
		}
	}
}

// sendAsync enqueues a job. If the buffer is full the job is dropped (non-blocking).
func (m *Mailer) sendAsync(to, subject, html string) {
	select {
	case m.jobCh <- mailJob{to: to, subject: subject, html: html}:
	default:
		m.logger.Printf("email queue full, dropped to=%s subject=%q", to, subject)
	}
}

// sendBlocking enqueues a job and waits for the worker to finish it.
// Use for paths where the caller must know if delivery succeeded.
func (m *Mailer) sendBlocking(to, subject, html string) error {
	errCh := make(chan error, 1)
	select {
	case m.jobCh <- mailJob{to: to, subject: subject, html: html, errCh: errCh}:
	default:
		return fmt.Errorf("email queue full")
	}
	return <-errCh
}

func (m *Mailer) from() string {
	return fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.FromEmail)
}

// sendSync is the raw synchronous SMTP delivery used by workers.
func (m *Mailer) sendSync(to, subject, html string) error {
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

// ── Public send methods ───────────────────────────────────────────────────────

// SendRegistration sends an OTP confirmation code to a new registrant.
// Enqueued ASYNC — HTTP handler returns immediately; SMTP happens in background.
// If delivery fails it is logged but the registration is not rolled back.
func (m *Mailer) SendRegistration(to, firstName, otp, eventTitle string) error {
	name := firstName
	if name == "" {
		name = "участник"
	}

	html := fmt.Sprintf(`
		<div style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto">
		  <p>Здравствуйте, <strong>%s</strong>!</p>
		  <p>Вы начали регистрацию на мероприятие:</p>
		  <p><strong>%s</strong></p>
		  <p>Ваш код подтверждения:</p>
		  <p style="font-size:36px;font-weight:bold;letter-spacing:8px;color:#009b35;margin:16px 0">%s</p>
		  <p style="color:#888;font-size:13px">Код действителен 10 минут.</p>
		  <p style="color:#aaa;font-size:12px">Если вы не запрашивали регистрацию — просто проигнорируйте это письмо.</p>
		</div>
	`, name, eventTitle, otp)

	// Async: do not block HTTP handler on SMTP latency.
	m.sendAsync(to, "Код подтверждения регистрации", html)
	return nil
}

// SendWelcome sends credentials and participation details after successful OTP verification.
// Includes login, password, participant ID, ticket link and cabinet link. ASYNC.
func (m *Mailer) SendWelcome(to, firstName, password string, userID, regID int, cabinetURL, ticketURL string) error {
	name := firstName
	if name == "" {
		name = "участник"
	}

	ticketBlock := ""
	if ticketURL != "" {
		ticketBlock = fmt.Sprintf(`
		  <p>Ваш билет с QR-кодом:</p>
		  <p>
		    <a href="%s" style="display:inline-block;padding:12px 24px;background:linear-gradient(135deg,#ff7c2c,#009b35);color:#fff;text-decoration:none;border-radius:8px;font-weight:bold">
		      Открыть билет / QR
		    </a>
		  </p>`, ticketURL)
	}

	html := fmt.Sprintf(`
		<div style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto">
		  <p>Здравствуйте, <strong>%s</strong>!</p>
		  <p>Ваш email подтверждён. Доступ в личный кабинет активирован.</p>
		  <table style="width:100%%;border-collapse:collapse;margin:16px 0">
		    <tr><td style="padding:8px;color:#555;width:40%%">Логин (email):</td>
		        <td style="padding:8px;font-weight:bold">%s</td></tr>
		    <tr style="background:#f9f9f9"><td style="padding:8px;color:#555">Пароль:</td>
		        <td style="padding:8px"><code style="background:#f0f0f0;padding:2px 6px;border-radius:4px;font-size:15px">%s</code></td></tr>
		    <tr><td style="padding:8px;color:#555">ID участника:</td>
		        <td style="padding:8px;color:#666">#%d (рег. №%d)</td></tr>
		  </table>
		  <p>
		    <a href="%s" style="display:inline-block;padding:12px 24px;background:#009b35;color:#fff;text-decoration:none;border-radius:8px;font-weight:bold">
		      Перейти в личный кабинет
		    </a>
		  </p>
		  %s
		  <p style="color:#aaa;font-size:12px;margin-top:24px">Рекомендуем сменить пароль после первого входа.</p>
		</div>
	`, name, to, password, userID, regID, cabinetURL, ticketBlock)

	m.sendAsync(to, "Добро пожаловать! Ваши данные для входа", html)
	return nil
}

// SendNewPassword mails a newly generated password. BLOCKING — user is waiting.
func (m *Mailer) SendNewPassword(to, firstName, password, loginURL string) error {
	name := firstName
	if name == "" {
		name = "участник"
	}
	html := fmt.Sprintf(`
		<div style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto">
		  <p>Здравствуйте, <strong>%s</strong>!</p>
		  <p>Ваш новый пароль для входа в личный кабинет:</p>
		  <p><code style="font-size:20px;background:#f0f0f0;padding:4px 10px;border-radius:4px">%s</code></p>
		  <p>Войти: <a href="%s">%s</a></p>
		  <p style="color:#aaa;font-size:12px">Если вы не запрашивали сброс пароля — проигнорируйте это письмо.</p>
		</div>
	`, name, password, loginURL, loginURL)
	return m.sendBlocking(to, "Новый пароль для личного кабинета", html)
}

// SendTicket sends a ticket confirmation with the QR link. ASYNC.
func (m *Mailer) SendTicket(to, firstName, eventTitle, ticketPageURL string) error {
	name := firstName
	if name == "" {
		name = "участник"
	}
	html := fmt.Sprintf(`
		<div style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto">
		  <p>Здравствуйте, <strong>%s</strong>!</p>
		  <p>Ваше участие в мероприятии <strong>%s</strong> подтверждено.</p>
		  <p>Ваш билет с QR-кодом для входа:</p>
		  <p>
		    <a href="%s" style="display:inline-block;padding:12px 24px;background:linear-gradient(135deg,#ff7c2c,#009b35);color:#fff;text-decoration:none;border-radius:8px;font-weight:bold">
		      Открыть билет / QR
		    </a>
		  </p>
		  <p style="color:#888;font-size:13px">Покажите QR-код на входе в место проведения мероприятия.</p>
		</div>
	`, name, eventTitle, ticketPageURL)
	m.sendAsync(to, "Ваш билет на мероприятие: "+eventTitle, html)
	return nil
}

// SendAdminOTP mails an OTP code for admin login. BLOCKING — admin is waiting.
func (m *Mailer) SendAdminOTP(to, name, otp string) error {
	html := fmt.Sprintf(`
		<div style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto">
		  <p>Здравствуйте, <strong>%s</strong>!</p>
		  <p>Код подтверждения для входа в панель администратора:</p>
		  <p style="font-size:36px;font-weight:bold;letter-spacing:8px;color:#009b35;margin:16px 0">%s</p>
		  <p style="color:#888;font-size:13px">Код действителен 10 минут.</p>
		</div>
	`, name, otp)
	return m.sendBlocking(to, "Код входа в панель администратора", html)
}

// SendDataExport sends a plain-text data export to the user. BLOCKING.
func (m *Mailer) SendDataExport(to, firstName, dataJSON string) error {
	name := firstName
	if name == "" {
		name = "участник"
	}
	html := fmt.Sprintf(`
		<div style="font-family:Arial,sans-serif;max-width:560px;margin:0 auto">
		  <p>Здравствуйте, <strong>%s</strong>!</p>
		  <p>По вашему запросу (152-ФЗ ст.14 / GDPR Art.15) направляем все персональные данные, хранящиеся в нашей системе:</p>
		  <pre style="background:#f5f5f5;padding:12px;border-radius:6px;font-size:12px;overflow-x:auto">%s</pre>
		  <p style="color:#aaa;font-size:12px">Если вы не запрашивали эти данные — проигнорируйте это письмо.</p>
		</div>
	`, name, dataJSON)
	return m.sendBlocking(to, "Ваши персональные данные (экспорт)", html)
}

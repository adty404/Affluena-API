package mailer

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// Mailer defines the interface for sending emails.
type Mailer interface {
	SendEmail(ctx context.Context, to []string, subject string, htmlBody string) error
}

// SMTPMailer implements Mailer using standard net/smtp.
type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewSMTPMailer creates a new SMTP Mailer.
func NewSMTPMailer(host string, port int, username, password, from string) *SMTPMailer {
	return &SMTPMailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

// SendEmail sends an HTML email via SMTP.
func (m *SMTPMailer) SendEmail(ctx context.Context, to []string, subject string, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	var auth smtp.Auth
	if m.username != "" || m.password != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	headers := make(map[string]string)
	headers["From"] = m.from
	headers["To"] = strings.Join(to, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""

	var msgBuilder strings.Builder
	for k, v := range headers {
		msgBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(htmlBody)

	msg := []byte(msgBuilder.String())

	return smtp.SendMail(addr, auth, m.from, to, msg)
}

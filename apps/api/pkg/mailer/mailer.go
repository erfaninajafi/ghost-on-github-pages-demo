// Package mailer implements durable-outbox delivery adapters.
package mailer

import (
	"context"
	"crypto/tls"
	"expertsacademy/blog/internal/domain"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// Mock records delivery metadata, never recipients or private comment content.
type Mock struct{ Log *slog.Logger }

func (m Mock) Send(_ context.Context, e domain.Email) error {
	m.Log.Info("mock email delivered", "job_id", e.ID)
	return nil
}

type SMTP struct{ Host, Port, Username, Password, From string }

func (s SMTP) Send(ctx context.Context, e domain.Email) error {
	from, err := mail.ParseAddress(s.From)
	if err != nil {
		return err
	}
	to, err := mail.ParseAddress(e.Recipient)
	if err != nil {
		return err
	}
	if strings.ContainsAny(e.Subject+from.Address+to.Address, "\r\n") {
		return fmt.Errorf("invalid mail header")
	}
	d := net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(s.Host, s.Port))
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(20 * time.Second)
	if t, ok := ctx.Deadline(); ok {
		deadline = t
	}
	if err = conn.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server must support STARTTLS")
	}
	if err = client.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
		return err
	}
	if s.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return err
		}
	}
	if err = client.Mail(from.Address); err != nil {
		return err
	}
	if err = client.Rcpt(to.Address); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	body := strings.ReplaceAll(strings.ReplaceAll(e.Body, "\r\n", "\n"), "\n", "\r\n")
	_, err = fmt.Fprintf(w, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: <%s@%s>\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", from.String(), to.String(), mime.QEncoding.Encode("UTF-8", e.Subject), e.ID, s.Host, time.Now().Format(time.RFC1123Z), body)
	if err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

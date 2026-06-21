package mail

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/morphatrix/campmenu/internal/settings"
)

// smtpTimeout caps how long we wait on a misbehaving mail server so a test send
// (or any email) fails fast with a clear error instead of hanging the request.
const smtpTimeout = 10 * time.Second

// buildMessage assembles a 7-bit-clean MIME message: the Subject is RFC 2047
// encoded and the UTF-8 body is base64'd. This avoids raw 8-bit content, which
// makes Postfix demand SMTPUTF8 — rejected by relays like OVH that don't offer it.
func buildMessage(from, to, subject, body string) []byte {
	enc := base64.StdEncoding.EncodeToString([]byte(body))
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n")
	b.WriteString("\r\n")
	for i := 0; i < len(enc); i += 76 { // RFC 2045: wrap base64 at 76 chars
		end := i + 76
		if end > len(enc) {
			end = len(enc)
		}
		b.WriteString(enc[i:end] + "\r\n")
	}
	return []byte(b.String())
}

// Mailer sends transactional email using the live settings store. When SMTP is
// unconfigured it logs instead, so local development needs no mail server.
type Mailer struct {
	settings *settings.Store
}

func New(s *settings.Store) *Mailer {
	return &Mailer{settings: s}
}

// Send delivers a plain-text email (or logs it when SMTP is disabled).
func (m *Mailer) Send(to, subject, body string) error {
	host := m.settings.Get(settings.KeySMTPHost)
	if host == "" {
		slog.Info("email (smtp disabled, logging only)", "to", to, "subject", subject, "body", body)
		return nil
	}
	port := m.settings.Int(settings.KeySMTPPort, 587)
	from := m.settings.Get(settings.KeySMTPFrom)
	user := m.settings.Get(settings.KeySMTPUser)
	pass := m.settings.Get(settings.KeySMTPPass)

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	msg := buildMessage(from, to, subject, body)

	// Dial with a timeout (net/smtp.SendMail has none, so an unreachable host
	// would block the request indefinitely).
	conn, err := net.DialTimeout("tcp", addr, smtpTimeout)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer c.Close()
	if user != "" {
		if err := c.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		_ = wc.Close()
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// SendConfirmation emails an email-confirmation link.
func (m *Mailer) SendConfirmation(to, token string) error {
	appURL := m.settings.Get(settings.KeyAppURL)
	siteName := m.settings.Get(settings.KeySiteName)
	link := fmt.Sprintf("%s/confirm/%s", appURL, token)
	body := fmt.Sprintf("Bienvenue sur %s !\n\nConfirmez votre adresse email en cliquant sur ce lien :\n%s\n", siteName, link)
	return m.Send(to, fmt.Sprintf("[%s] Confirmez votre email", siteName), body)
}

// SendPasswordReset emails a password-reset link.
func (m *Mailer) SendPasswordReset(to, token string) error {
	appURL := m.settings.Get(settings.KeyAppURL)
	siteName := m.settings.Get(settings.KeySiteName)
	link := fmt.Sprintf("%s/reset/%s", appURL, token)
	body := fmt.Sprintf("Réinitialisation de votre mot de passe sur %s.\n\nCliquez sur ce lien (valable 1 heure) :\n%s\n\nSi vous n'êtes pas à l'origine de cette demande, ignorez cet email.\n", siteName, link)
	return m.Send(to, fmt.Sprintf("[%s] Réinitialisation du mot de passe", siteName), body)
}

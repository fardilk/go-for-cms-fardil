// Package mailer sends the notification that a new enquiry arrived.
package mailer

import (
	"fmt"
	"log"
	"mime"
	"net/smtp"
	"os"
	"strings"
)

// Config is read from the environment. When it is incomplete, sending is
// skipped rather than failing: an enquiry that is stored but not emailed is a
// missed notification, while an enquiry rejected because SMTP is misconfigured
// is a lost customer.
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	To       string
}

func FromEnv() Config {
	return Config{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		Username: os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
		To:       os.Getenv("LEAD_NOTIFY_TO"),
	}
}

// Configured reports whether enough is set to attempt a send.
func (c Config) Configured() bool {
	return c.Host != "" && c.Port != "" && c.From != "" && c.To != ""
}

// Send delivers a plain-text message. Recipients are taken from the config, not
// from user input, so a submitted field can never redirect the mail.
func Send(c Config, subject, body string) error {
	if !c.Configured() {
		return fmt.Errorf("smtp is not configured")
	}

	recipients := strings.Split(c.To, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	// Header values must not carry newlines, or a crafted subject could inject
	// extra headers.
	subject = strings.NewReplacer("\r", " ", "\n", " ").Replace(subject)

	msg := strings.Join([]string{
		"From: " + c.From,
		"To: " + strings.Join(recipients, ", "),
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := c.Host + ":" + c.Port
	var auth smtp.Auth
	if c.Username != "" {
		auth = smtp.PlainAuth("", c.Username, c.Password, c.Host)
	}

	return smtp.SendMail(addr, auth, c.From, recipients, []byte(msg))
}

// SendAsync sends in the background so a slow mail server cannot hold up the
// visitor's request. Failures are logged; the enquiry is already saved.
func SendAsync(c Config, subject, body string) {
	if !c.Configured() {
		log.Println("lead notification skipped: SMTP is not configured")
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("lead notification panicked: %v", r)
			}
		}()
		if err := Send(c, subject, body); err != nil {
			log.Printf("lead notification failed: %v", err)
		}
	}()
}

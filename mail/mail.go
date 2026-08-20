package mail

import (
	"fmt"
	"net/smtp"

	"github.com/MetroReviews/backend-v2/state"
	"go.uber.org/zap"
)

func Send(to, subject, body string) error {
	cfg := state.Config.SMTP
	if cfg.Host == "" {
		state.Logger.Warn("smtp not configured; skipping send",
			zap.String("to", to), zap.String("subject", subject))
		return nil
	}

	from := cfg.From
	if from == "" {
		from = cfg.User
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", from, to, subject, body)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var auth smtp.Auth
	if cfg.User != "" {
		auth = smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

package gmail

import (
	"flag"
	"fmt"
	"grpc-service-ref/internal/lib/logger/sl"
	"log/slog"
	"net/smtp"
	"net/textproto"
	"os"

	"github.com/jordan-wright/email"
)

const (
	smtpAuthAddress   = "smtp.gmail.com"
	smtpServerAddress = "smtp.gmail.com:587"
)

type GmailSender struct {
	log               *slog.Logger
	name              string
	fromEmailAddress  string
	fromEmailPassword string
}

func New(
	log *slog.Logger,
	name string,
	email string,
	password string) *GmailSender {
	return &GmailSender{
		log:               log,
		name:              name,
		fromEmailAddress:  email,
		fromEmailPassword: password,
	}
}

func (sender *GmailSender) SendEmail(
	subject string,
	to []string,
	content string,
	cc []string,
	bcc []string,
	atachFiles []string,
) error {
	const op = "Gmail.SendEmail"

	log := sender.log.With(
		slog.String("op", op),
	)

	log.Info("attempting to send email")

	e := &email.Email{
		To:      to,
		From:    fmt.Sprintf("%s <%s>", sender.name, sender.fromEmailAddress),
		Subject: subject,
		//Text:    []byte(body),
		HTML:    []byte(content),
		Headers: textproto.MIMEHeader{},
		Cc:      cc,
		Bcc:     bcc,
	}

	if len(atachFiles) > 0 {
		for _, f := range atachFiles {
			_, err := e.AttachFile(f)
			if err != nil {
				sender.log.Error("failed to attach file to email", sl.Err(err))
			}
		}
	}

	smtpAuth := smtp.PlainAuth("", sender.fromEmailAddress, sender.fromEmailPassword, smtpAuthAddress)
	return e.Send(smtpServerAddress, smtpAuth)

}

// fetchConfigPath fetches config path from command line flag or environment variable.
// Priority: flag > env > default.
// Default value is empty string.
func fetchSenderName() string {
	var res string

	flag.StringVar(&res, "senderName", "", "sender name")
	flag.Parse()

	if res == "" {
		res = os.Getenv("EMAIL_NAME")
	}

	return res
}

func fetchSenderEmail() string {
	var res string

	flag.StringVar(&res, "senderEmail", "", "sender email")
	flag.Parse()

	if res == "" {
		res = os.Getenv("SENDER_EMAIL")
	}

	return res
}

func fetchSenderPassword() string {
	var res string

	flag.StringVar(&res, "senderPassword", "", "sender password")
	flag.Parse()

	if res == "" {
		res = os.Getenv("SENDER_PASSWORD")
	}

	return res
}

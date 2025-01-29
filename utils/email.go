package utils

import (
	"iis-logs-parser/config"
	"net/smtp"
	"os"
)

func SendEmail(to []string, subject string, body string) error {
	fromEmail := os.Getenv("FROM_EMAIL")
	fromEmailSmtp := os.Getenv("FROM_EMAIL_SMTP")

	auth := smtp.PlainAuth(
		"",
		fromEmail,
		os.Getenv("FROM_EMAIL_PASSWORD"),
		fromEmailSmtp,
	)

	from := fromEmail
	msg := "From: " + from + "\r\n" +
		"To: " + to[0] + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n"

	return smtp.SendMail(
		os.Getenv("FROM_EMAIL_SMTP")+":"+os.Getenv("FROM_EMAIL_PORT"),
		auth,
		fromEmail,
		to,
		[]byte(msg),
	)
}

func SendVerifyUserEmail(to string, token string) error {
	subject := "Verify your email"

	body := "Click the link to verify your email: http://localhost:" + config.GetServerPortOrDefault() + "/api/v1/users/verify?token=" + token
	return SendEmail([]string{to}, subject, body)
}

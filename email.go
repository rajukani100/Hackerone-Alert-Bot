package main

import (
	"bytes"
	"fmt"
	"net/smtp"
	"os"
)

func sendEmail(list []string) {
	from := os.Getenv("FROM_EMAIL")
	password := os.Getenv("FROM_EMAIL_PASSWORD")
	to := []string{os.Getenv("TO_EMAIL")}
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	// HTML email
	subject := "Subject: 🚨 HackerOne Update Alert\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	tmpl := `
	<html>
		<body style="font-family: sans-serif;">
			<h2 style="color: #4CAF50;">HackerOne Programs Updated</h2>
			<p>Hey,</p>
			<p>The following programs had scope updates:</p>
			<ul>%s</ul>
			<p>Keep hacking 👾</p>
		</body>
	</html>
	`

	var htmlBuf bytes.Buffer

	for _, st := range list {
		htmlBuf.WriteString(fmt.Sprintf("<li>%s</li>", st))
	}

	body := fmt.Sprintf(tmpl, htmlBuf.String())

	msg := []byte(subject + mime + body)
	auth := smtp.PlainAuth("", from, password, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, msg)
	if err != nil {
		fmt.Println("Failed to send email:", err)
		return
	}
	fmt.Println("Email sent successfully.")
}

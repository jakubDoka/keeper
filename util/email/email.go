package email

import "net/smtp"

const (
	stmpHost = "smtp.gmail.com"
	stmpPort = "587"
)

type Client struct {
	Email, Password string
	Auth            smtp.Auth
}

func New(email, password string) *Client {
	auth := smtp.PlainAuth("", email, password, stmpHost)

	return &Client{
		Email:    email,
		Password: password,
		Auth:     auth,
	}
}

func (e *Client) Send(subject, body string, to ...string) error {
	return smtp.SendMail(
		stmpHost+":"+stmpPort,
		e.Auth,
		e.Email,
		to,
		[]byte("Subject: "+subject+"\r\n\r\n"+body),
	)
}

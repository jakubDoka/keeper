package auth

import (
	"github.com/jakubDoka/keeper/util/email"
	"github.com/jakubDoka/keeper/util/khtml"
)

type EmailClient struct {
	*email.Client
}

func NewEmailClient(emailAddress, password string) EmailClient {
	return EmailClient{email.New(emailAddress, password)}
}

func (c *EmailClient) Send(link, to string) error {
	var builder khtml.Html

	builder.
		Tag("h1").Text("Email Verification").
		Tag("h3").Text("Your verification link: " + link).
		Tag("p").Text("link expires after 5 minutes.")

	return c.Client.Send("Verification email from Something.", string(builder.Close(nil)), to)
}

package mailer

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"text/template"

	gomail "github.com/wneessen/go-mail"
	"gitlab.com/secban/secban-agent/ipc"
)

//go:embed body.tpl
var bodyTemplate string

var mailerInstance *Mailer //nolint: gochecknoglobals

var ErrMailerNotInitialized = errors.New("mailer not initialized")

type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	TLSPolicy gomail.TLSPolicy
}

type Mailer struct {
	Template template.Template
	From     string
	To       []string
	SMTP     SMTPConfig
}

type MailerConfig struct {
	Template template.Template
	SMTP     SMTPConfig
	From     string
	To       []string
}

func Init(config MailerConfig) {
	mailerInstance = New(config)
}

func DefaultTemplate() template.Template {
	return *template.Must(template.New("body").Parse(bodyTemplate))
}

func New(config MailerConfig) *Mailer {
	if config.Template == (template.Template{}) {
		config.Template = DefaultTemplate()
	}

	return &Mailer{
		Template: config.Template,
		From:     config.From,
		To:       config.To,
		SMTP:     config.SMTP,
	}
}

func (m *Mailer) Send(ctx context.Context, msg ipc.IPCMessage) error {
	email := gomail.NewMsg()

	fromErr := email.From(m.From)
	if fromErr != nil {
		return fromErr
	}

	toErr := email.To(m.To...)
	if toErr != nil {
		return toErr
	}

	subject := fmt.Sprintf("SecBan [%s] %s", msg.Rule, msg.IP)
	email.Subject(subject)

	bodyErr := email.SetBodyTextTemplate(&m.Template, msg)
	if bodyErr != nil {
		return bodyErr
	}

	clientOptions := []gomail.Option{
		gomail.WithTLSPolicy(m.SMTP.TLSPolicy),
	}
	if m.SMTP.Username != "" || m.SMTP.Password != "" {
		clientOptions = append(clientOptions,
			gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
			gomail.WithUsername(m.SMTP.Username),
			gomail.WithPassword(m.SMTP.Password),
		)
	}

	if m.SMTP.Port != 0 {
		clientOptions = append(clientOptions, gomail.WithPort(m.SMTP.Port))
	}

	client, clientErr := gomail.NewClient(m.SMTP.Host, clientOptions...)
	if clientErr != nil {
		return clientErr
	}

	sendErr := client.DialAndSend(email)
	if sendErr != nil {
		return sendErr
	}

	return nil
}

func Send(ctx context.Context, msg ipc.IPCMessage) error {
	if mailerInstance == nil {
		return ErrMailerNotInitialized
	}

	return mailerInstance.Send(ctx, msg)
}

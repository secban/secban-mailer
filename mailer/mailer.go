package mailer

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"text/template"

	gomail "github.com/wneessen/go-mail"
	"gitlab.com/secban/secban-agent/ipc"
)

//go:embed body.tpl
var bodyTemplate string

var mailerInstance *Mailer //nolint:gochecknoglobals

var ErrMailerNotInitialized = errors.New("mailer not initialized")

type SMTPConfig struct {
	Host      string
	Username  string
	Password  string
	Port      int
	TLSPolicy gomail.TLSPolicy
}

type Mailer struct {
	Template template.Template
	From     string
	To       []string
	SMTP     SMTPConfig
}

type Config struct {
	Template template.Template
	From     string
	To       []string
	SMTP     SMTPConfig
}

func Init(config Config) {
	mailerInstance = New(config)
}

func DefaultTemplate() template.Template {
	return *template.Must(template.New("body").Parse(bodyTemplate))
}

func New(config Config) *Mailer {
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

func (m *Mailer) Send(ctx context.Context, msg ipc.Message) error {
	return m.SendGroup(ctx, []ipc.Message{msg})
}

func (m *Mailer) SendGroup(_ context.Context, bans []ipc.Message) error {
	if len(bans) == 0 {
		return nil
	}

	email, buildErr := m.buildEmail(bans)
	if buildErr != nil {
		return buildErr
	}

	client, clientErr := m.buildClient()
	if clientErr != nil {
		return clientErr
	}

	sendErr := client.DialAndSend(email)
	if sendErr != nil {
		return sendErr
	}

	return nil
}

func (m *Mailer) buildEmail(bans []ipc.Message) (*gomail.Msg, error) {
	email := gomail.NewMsg()

	fromErr := email.From(m.From)
	if fromErr != nil {
		return nil, fromErr
	}

	toErr := email.To(m.To...)
	if toErr != nil {
		return nil, toErr
	}

	hostname := getHostname()

	templateData := struct {
		Host string
		Bans []ipc.Message
	}{
		Bans: bans,
		Host: hostname,
	}

	subject := "SecBan report from " + hostname

	email.Subject(subject)

	bodyErr := email.SetBodyTextTemplate(&m.Template, templateData)
	if bodyErr != nil {
		return nil, bodyErr
	}

	return email, nil
}

func (m *Mailer) buildClient() (*gomail.Client, error) {
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

	return gomail.NewClient(m.SMTP.Host, clientOptions...)
}

func Send(ctx context.Context, msg ipc.Message) error {
	if mailerInstance == nil {
		return ErrMailerNotInitialized
	}

	return mailerInstance.Send(ctx, msg)
}

func SendGroup(ctx context.Context, bans []ipc.Message) error {
	if mailerInstance == nil {
		return ErrMailerNotInitialized
	}

	return mailerInstance.SendGroup(ctx, bans)
}

func getHostname() string {
	hostname, hostErr := os.Hostname()
	if hostErr != nil {
		return "unknown"
	}

	return hostname
}

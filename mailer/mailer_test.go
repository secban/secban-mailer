package mailer_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gomail "github.com/wneessen/go-mail"
	"gitlab.com/secban/secban-agent/entry"
	"gitlab.com/secban/secban-agent/ipc"
	"gitlab.com/secban/secban-agent/offender"
	"gitlab.com/secban/secban-mailer/config"
	mailer "gitlab.com/secban/secban-mailer/mailer"
)

func TestBodyTemplateExecutesForIPCMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()
	msg := ipc.IPCMessage{
		IP:   "127.0.0.1",
		Rule: "dummy-rule",
		Offender: offender.Offender{
			Timestamp: now,
			Entries: []entry.Entry{
				{Timestamp: now, Message: "Dummy message"},
			},
		},
	}

	tmpl := mailer.DefaultTemplate()

	var buf bytes.Buffer

	err := tmpl.Execute(&buf, msg)
	if err != nil {
		t.Fatalf("template execution failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "127.0.0.1") {
		t.Fatalf("rendered output missing IP: %q", out)
	}

	if !strings.Contains(out, "dummy-rule") {
		t.Fatalf("rendered output missing rule: %q", out)
	}

	if !strings.Contains(out, "Dummy message") {
		t.Fatalf("rendered output missing offender message: %q", out)
	}
}

func TestSMTPHostWithPortIsSplitIntoHostAndPort(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mailer.yaml")

	configData := []byte(
		"from: from@example.com\nsmtp:\n  host: localhost:1025\n  username: tester\n  password: secret\n",
	)

	writeErr := os.WriteFile(configPath, configData, 0o600)
	if writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	cfg, err := config.LoadConfig(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.SMTP.Host != "localhost" {
		t.Fatalf("host = %q, want localhost", cfg.SMTP.Host)
	}

	if cfg.SMTP.Port != 1025 {
		t.Fatalf("port = %d, want 1025", cfg.SMTP.Port)
	}
}

func TestMailerNewUsesEmbeddedTemplateByDefault(t *testing.T) {
	t.Parallel()

	mailerInstance := mailer.New(mailer.MailerConfig{})
	if mailerInstance.Template.Name() != "body" {
		t.Fatalf("template name = %q, want body", mailerInstance.Template.Name())
	}
}

func TestLoadConfigUsesProvidedTemplatePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	templatePath := filepath.Join(dir, "custom.tpl")
	configPath := filepath.Join(dir, "mailer.yaml")

	templateData := []byte("Custom template for {{ .IP }}\n")

	writeErr := os.WriteFile(templatePath, templateData, 0o600)
	if writeErr != nil {
		t.Fatalf("write template: %v", writeErr)
	}

	configData := []byte(
		"from: from@example.com\n" +
			"template_file: custom.tpl\n" +
			"smtp:\n" +
			"  host: smtp.example.com\n" +
			"  port: 25\n\n" +
			"to:\n" +
			"  - target@example.com\n",
	)

	writeErr = os.WriteFile(configPath, configData, 0o600)
	if writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	cfg, err := config.LoadConfig(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Template.Name() != "custom.tpl" {
		t.Fatalf("template name = %q, want custom.tpl", cfg.Template.Name())
	}
}

func TestLoadConfigParsesCoreSMTPSettings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mailer.yaml")

	configData := []byte(
		"from: secban@example.com\n" +
			"smtp:\n" +
			"  host: mail.example.com:2525\n" +
			"  username: tester\n" +
			"  password: secret\n\n" +
			"to:\n" +
			"  - user@example.com\n" +
			"  - ops@example.com\n",
	)

	writeErr := os.WriteFile(configPath, configData, 0o600)
	if writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	cfg, err := config.LoadConfig(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	assertSMTPConfig(t, cfg)
}

func assertSMTPConfig(t *testing.T, cfg *config.Config) {
	t.Helper()

	if cfg.From != "secban@example.com" {
		t.Fatalf("from = %q, want secban@example.com", cfg.From)
	}

	if cfg.SMTP.Host != "mail.example.com" {
		t.Fatalf("smtp host = %q, want mail.example.com", cfg.SMTP.Host)
	}

	if cfg.SMTP.Port != 2525 {
		t.Fatalf("smtp port = %d, want 2525", cfg.SMTP.Port)
	}

	if cfg.SMTP.Username != "tester" {
		t.Fatalf("smtp username = %q, want tester", cfg.SMTP.Username)
	}

	if cfg.SMTP.Password != "secret" {
		t.Fatalf("smtp password = %q, want secret", cfg.SMTP.Password)
	}

	if len(cfg.To) != 2 || cfg.To[0] != "user@example.com" || cfg.To[1] != "ops@example.com" {
		t.Fatalf("unexpected recipients: %#v", cfg.To)
	}

	if cfg.SMTP.TLSPolicy != gomail.TLSOpportunistic {
		t.Fatalf("smtp tls policy = %v, want %v", cfg.SMTP.TLSPolicy, gomail.TLSOpportunistic)
	}
}

func TestLoadConfigParsesTLSPolicies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		yaml       string
		wantPolicy gomail.TLSPolicy
	}{
		{name: "notls", yaml: "from: a@b.com\nsmtp:\n  host: localhost\n  tls: notls\n", wantPolicy: gomail.NoTLS},
		{name: "notsl", yaml: "from: a@b.com\nsmtp:\n  host: localhost\n  tls: notsl\n", wantPolicy: gomail.NoTLS},
		{name: "mandatory", yaml: "from: a@b.com\nsmtp:\n  host: localhost\n  tls: mandatory\n", wantPolicy: gomail.TLSMandatory},
		{name: "opportunistic", yaml: "from: a@b.com\nsmtp:\n  host: localhost\n  tls: opportunistic\n", wantPolicy: gomail.TLSOpportunistic},
		{name: "empty defaults to opportunistic", yaml: "from: a@b.com\nsmtp:\n  host: localhost\n", wantPolicy: gomail.TLSOpportunistic},
		{name: "unknown defaults to opportunistic", yaml: "from: a@b.com\nsmtp:\n  host: localhost\n  tls: something_else\n", wantPolicy: gomail.TLSOpportunistic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertLoadConfigTLSPolicy(t, tt.yaml, tt.wantPolicy)
		})
	}
}

func assertLoadConfigTLSPolicy(t *testing.T, yamlContent string, wantPolicy gomail.TLSPolicy) {
	t.Helper()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mailer.yaml")

	writeErr := os.WriteFile(configPath, []byte(yamlContent), 0o600)
	if writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	cfg, loadErr := config.LoadConfig(dir)
	if loadErr != nil {
		t.Fatalf("unexpected error: %v", loadErr)
	}

	if cfg.SMTP.TLSPolicy != wantPolicy {
		t.Fatalf("TLSPolicy = %v, want %v", cfg.SMTP.TLSPolicy, wantPolicy)
	}
}

func TestMailerNewStoresTLSPolicy(t *testing.T) {
	t.Parallel()

	mailerInstance := mailer.New(mailer.MailerConfig{
		SMTP: mailer.SMTPConfig{
			TLSPolicy: gomail.TLSOpportunistic,
		},
	})
	if mailerInstance.SMTP.TLSPolicy != gomail.TLSOpportunistic {
		t.Fatalf("TLSPolicy = %v, want %v", mailerInstance.SMTP.TLSPolicy, gomail.TLSOpportunistic)
	}
}

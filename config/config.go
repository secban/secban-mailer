package config

import (
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"

	"github.com/rs/zerolog"
	gomail "github.com/wneessen/go-mail"
	"gopkg.in/yaml.v2"
)

const (
	fileName = "mailer.yaml"
)

type ParsedSMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	TLS      string `yaml:"tls"`
}

type ParsedConfig struct {
	From         string           `yaml:"from"`
	SMTP         ParsedSMTPConfig `yaml:"smtp"`
	To           []string         `yaml:"to"`
	LogLevel     string           `yaml:"log_level"`
	TemplateFile string           `yaml:"template_file"`
	SocketPath   string           `yaml:"socket_path"`
}

type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	TLS       string
	TLSPolicy gomail.TLSPolicy
}

type Config struct {
	From       string
	SMTP       SMTPConfig
	To         []string
	LogLevel   zerolog.Level
	Template   template.Template
	SocketPath string
}

func LoadConfig(configDir string) (*Config, error) {
	parsed, parseErr := parseConfig(configDir)
	if parseErr != nil {
		return nil, parseErr
	}

	smtpCfg := normalizeSMTPConfig(parsed.SMTP)
	applyEnvOverrides(&smtpCfg)

	logLevel, logLevelErr := parseLogLevel(parsed.LogLevel)
	if logLevelErr != nil {
		return nil, logLevelErr
	}

	tmpl, tmplErr := loadTemplate(configDir, parsed.TemplateFile)
	if tmplErr != nil {
		return nil, tmplErr
	}

	return &Config{
		From:       parsed.From,
		SMTP:       smtpCfg,
		To:         parsed.To,
		SocketPath: parsed.SocketPath,
		LogLevel:   logLevel,
		Template:   tmpl,
	}, nil
}

func applyEnvOverrides(cfg *SMTPConfig) {
	if username := os.Getenv("SECBAN_MAILER_SMTP_USERNAME"); username != "" {
		cfg.Username = username
	}

	if password := os.Getenv("SECBAN_MAILER_SMTP_PASSWORD"); password != "" {
		cfg.Password = password
	}
}

func loadTemplate(configDir string, templateFile string) (template.Template, error) {
	if templateFile == "" {
		return template.Template{}, nil
	}

	templatePath := path.Join(configDir, templateFile)

	tmpl, tmplErr := template.ParseFiles(templatePath)
	if tmplErr != nil {
		return template.Template{}, tmplErr
	}

	return *tmpl, nil
}

func parseConfig(configDir string) (*ParsedConfig, error) {
	mainConfigPath := path.Join(configDir, fileName)

	var parsed ParsedConfig

	configData, readErr := os.ReadFile(mainConfigPath)
	if readErr != nil {
		return nil, readErr
	}

	decodeErr := yaml.Unmarshal(configData, &parsed)
	if decodeErr != nil {
		return nil, decodeErr
	}

	return &parsed, nil
}

func normalizeSMTPConfig(parsed ParsedSMTPConfig) SMTPConfig {
	cfg := SMTPConfig{
		Host:      parsed.Host,
		Port:      parsed.Port,
		Username:  parsed.Username,
		Password:  parsed.Password,
		TLS:       parsed.TLS,
		TLSPolicy: parseTLSPolicy(parsed.TLS),
	}

	if cfg.Port != 0 || strings.TrimSpace(cfg.Host) == "" {
		return cfg
	}

	if strings.Contains(strings.TrimSpace(cfg.Host), ":") {
		host, port, splitErr := net.SplitHostPort(strings.TrimSpace(cfg.Host))
		if splitErr == nil {
			parsedPort, parsePortErr := strconv.Atoi(port)
			if parsePortErr == nil {
				cfg.Host = host
				cfg.Port = parsedPort
			}
		}
	}

	return cfg
}

func parseTLSPolicy(raw string) gomail.TLSPolicy {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "notls", "notsl":
		return gomail.NoTLS
	case "mandatory":
		return gomail.TLSMandatory
	default:
		return gomail.TLSOpportunistic
	}
}

func parseLogLevel(logLevel string) (zerolog.Level, error) {
	if logLevel == "" {
		logLevel = "info"
	}

	parsedLevel, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		return zerolog.Disabled, err
	}

	return parsedLevel, nil
}

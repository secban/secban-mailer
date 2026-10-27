package config

import (
	"errors"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog"
	gomail "github.com/wneessen/go-mail"
	"go.yaml.in/yaml/v3"
)

const (
	fileName = "mailer.yaml"
)

var (
	ErrDatabaseRequired   = errors.New("database path required when frequency or threshold is set")
	ErrNegativeFrequency  = errors.New("frequency cannot be negative")
	ErrStateFileRequired  = errors.New("state file required when frequency or threshold is set")
)

type ParsedSMTPConfig struct {
	Host     string `yaml:"host"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	TLS      string `yaml:"tls"`
	Port     int    `yaml:"port"`
}

type ParsedConfig struct {
	SMTP         ParsedSMTPConfig `yaml:"smtp"`
	From         string           `yaml:"from"`
	LogLevel     string           `yaml:"log_level"`
	TemplateFile string           `yaml:"template_file"`
	SocketPath   string           `yaml:"socket_path"`
	Database     string           `yaml:"database"`
	StateFile    string           `yaml:"state_file"`
	Frequency    string           `yaml:"frequency"`
	To           []string         `yaml:"to"`
	Threshold    int              `yaml:"threshold"`
}

type SMTPConfig struct {
	Host      string
	Username  string
	Password  string
	TLS       string
	Port      int
	TLSPolicy gomail.TLSPolicy
}

type Config struct {
	Template   template.Template
	From       string
	SocketPath string
	Database   string
	StateFile  string
	To         []string
	SMTP       SMTPConfig
	Frequency  time.Duration
	Threshold  int
	LogLevel   zerolog.Level
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

	freq, parseFreqErr := parseFrequency(parsed.Frequency)
	if parseFreqErr != nil {
		return nil, parseFreqErr
	}

	validateErr := validateReporting(parsed, freq)
	if validateErr != nil {
		return nil, validateErr
	}

	return &Config{
		From:       parsed.From,
		SMTP:       smtpCfg,
		To:         parsed.To,
		SocketPath: parsed.SocketPath,
		LogLevel:   logLevel,
		Template:   tmpl,
		Database:   parsed.Database,
		StateFile:  parsed.StateFile,
		Frequency:  freq,
		Threshold:  parsed.Threshold,
	}, nil
}

func validateReporting(parsed *ParsedConfig, freq time.Duration) error {
	if freq <= 0 && parsed.Threshold <= 0 {
		return nil
	}

	if parsed.Database == "" {
		return ErrDatabaseRequired
	}

	if parsed.StateFile == "" {
		return ErrStateFileRequired
	}

	return nil
}

func parseFrequency(raw string) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "0" {
		return 0, nil
	}

	duration, parseErr := time.ParseDuration(trimmed)
	if parseErr != nil {
		return 0, parseErr
	}

	if duration < 0 {
		return 0, ErrNegativeFrequency
	}

	return duration, nil
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

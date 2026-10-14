package config

import (
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v2"
)

const (
	fileName = "mailer.yaml"
)

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ParsedConfig struct {
	From         string     `yaml:"from"`
	SMTP         SMTPConfig `yaml:"smtp"`
	To           []string   `yaml:"to"`
	LogLevel     string     `yaml:"log_level"`
	TemplateFile string     `yaml:"template_file"`
	SocketPath   string     `yaml:"socket_path"`
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

	config := &Config{
		From:       parsed.From,
		SMTP:       normalizeSMTPConfig(parsed.SMTP),
		To:         parsed.To,
		SocketPath: parsed.SocketPath,
	}

	if os.Getenv("SECBAN_MAILER_SMTP_USERNAME") != "" {
		config.SMTP.Username = os.Getenv("SECBAN_MAILER_SMTP_USERNAME")
	}

	if os.Getenv("SECBAN_MAILER_SMTP_PASSWORD") != "" {
		config.SMTP.Password = os.Getenv("SECBAN_MAILER_SMTP_PASSWORD")
	}

	logLevel, logLevelErr := parseLogLevel(parsed.LogLevel)
	if logLevelErr != nil {
		return nil, logLevelErr
	}

	config.LogLevel = logLevel

	if parsed.TemplateFile != "" {
		templatePath := path.Join(configDir, parsed.TemplateFile)

		tmpl, tmplErr := template.ParseFiles(templatePath)
		if tmplErr != nil {
			return nil, tmplErr
		}

		config.Template = *tmpl
	}

	return config, nil
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

func normalizeSMTPConfig(cfg SMTPConfig) SMTPConfig {
	if cfg.Port != 0 || strings.TrimSpace(cfg.Host) == "" {
		return cfg
	}

	host, port, err := net.SplitHostPort(strings.TrimSpace(cfg.Host))
	if err != nil {
		return cfg
	}

	parsedPort, parsePortErr := strconv.Atoi(port)
	if parsePortErr != nil {
		return cfg
	}

	cfg.Host = host
	cfg.Port = parsedPort

	return cfg
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

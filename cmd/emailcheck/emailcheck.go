package main

import (
	"context"
	"flag"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/secban/secban-agent/entry"
	"gitlab.com/secban/secban-agent/ipc"
	"gitlab.com/secban/secban-agent/offender"
	"gitlab.com/secban/secban-mailer/config"
	"gitlab.com/secban/secban-mailer/mailer"
)

func main() {
	conf := loadConf()

	log.Info().
		Str("smtp", conf.SMTP.Host).
		Str("tls", conf.SMTP.TLSPolicy.String()).
		Msg("configuration loaded")

	mailer.Init(mailer.Config{
		Template: conf.Template,
		SMTP: mailer.SMTPConfig{
			Host:      conf.SMTP.Host,
			Port:      conf.SMTP.Port,
			Username:  conf.SMTP.Username,
			Password:  conf.SMTP.Password,
			TLSPolicy: conf.SMTP.TLSPolicy,
		},
		From: conf.From,
		To:   conf.To,
	})

	err := mailer.Send(context.Background(), dummy())
	if err != nil {
		log.Error().Err(err).Msg("failed to send test email")
	}
}

func dummy() ipc.Message {
	return ipc.Message{
		IP:   "127.0.0.1",
		Rule: "dummy-rule",
		Offender: offender.Offender{
			Timestamp: time.Now(),
			Entries: []entry.Entry{
				{
					Timestamp: time.Now(),
					Message:   "Dummy message",
				},
			},
		},
	}
}

func loadConf() *config.Config {
	var configDir string
	//nolint:revive,nolintlint // Not a real deep-exit, just extracted from main for readability.
	flag.StringVar(&configDir, "config", "/etc/secban", "path to config directory")
	//nolint:revive,nolintlint
	flag.Parse()

	conf, confErr := config.LoadConfig(configDir)
	if confErr != nil {
		log.Fatal().Err(confErr).Msg("failed to load config")
	}

	return conf
}

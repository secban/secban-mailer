package main

import (
	"context"
	"flag"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/thequietbot/secban/secban-agent/entry"
	"gitlab.com/thequietbot/secban/secban-agent/ipc"
	"gitlab.com/thequietbot/secban/secban-agent/offender"
	"gitlab.com/thequietbot/secban/secban-mailer/config"
	"gitlab.com/thequietbot/secban/secban-mailer/mailer"
)

func main() {
	conf := loadConf()

	log.Info().
		Str("smtp", conf.SMTP.Host).
		Msg("configuration loaded")

	mailer.Init(mailer.MailerConfig{
		Template: conf.Template,
		SMTP: mailer.SMTPConfig{
			Host:     conf.SMTP.Host,
			Port:     conf.SMTP.Port,
			Username: conf.SMTP.Username,
			Password: conf.SMTP.Password,
		},
		From: conf.From,
		To:   conf.To,
	})

	err := mailer.Send(context.Background(), dummy())
	if err != nil {
		log.Error().Err(err).Msg("failed to send test email")
	}
}

func dummy() ipc.IPCMessage {
	return ipc.IPCMessage{
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

	flag.StringVar(&configDir, "config", "/etc/secban", "path to config directory")
	flag.Parse()

	conf, confErr := config.LoadConfig(configDir)
	if confErr != nil {
		log.Fatal().Err(confErr).Msg("failed to load config")
	}

	return conf
}

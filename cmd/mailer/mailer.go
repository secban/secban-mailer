package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/secban/secban-agent/ipc"
	"gitlab.com/secban/secban-mailer/config"
	"gitlab.com/secban/secban-mailer/mailer"
)

func main() {
	conf := loadConf()

	zerolog.SetGlobalLevel(conf.LogLevel)

	initMailer(conf)

	ctx := context.Background()

	ipc.ListenUnixSocket(conf.SocketPath, func(msg ipc.IPCMessage) error {
		sendErr := mailer.Send(ctx, msg)
		if sendErr != nil {
			log.Error().Err(sendErr).Msg("failed to send email")
		}

		return nil
	})
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

func initMailer(conf *config.Config) {
	mailer.Init(mailer.MailerConfig{
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
}

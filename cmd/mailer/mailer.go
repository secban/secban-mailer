package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/secban/secban-agent/ipc"
	"gitlab.com/secban/secban-agent/profiler"
	"gitlab.com/secban/secban-agent/version"
	"gitlab.com/secban/secban-mailer/config"
	"gitlab.com/secban/secban-mailer/mailer"
)

func main() {
	version.CLI()

	conf := loadConf()

	zerolog.SetGlobalLevel(conf.LogLevel)

	initMailer(conf)

	ctx := context.Background()

	if conf.Profiler != "" {
		go profiler.Listen(ctx, conf.Profiler)
	}

	reporter := mailer.NewReporter(conf)

	reporterInitErr := reporter.Init(ctx)
	if reporterInitErr != nil {
		log.Fatal().Err(reporterInitErr).Msg("failed to initialize reporter")
	}

	log.Info().Msg("starting mailer")

	listenErr := ipc.ListenUnixSocket(conf.SocketPath, func(msg ipc.Message) error {
		handleErr := reporter.HandleBan(ctx, msg)
		if handleErr != nil {
			log.Error().Err(handleErr).Msg("failed to handle ban report")
		}

		return nil
	})
	if listenErr != nil {
		log.Fatal().Err(listenErr).Msg("failed to listen on unix socket")
	}
}

func loadConf() *config.Config {
	var configDir string
	//nolint:revive,nolintlint // Not a real deep-exit, extracted from main for readability.
	flag.StringVar(&configDir, "config", "/etc/secban", "path to config directory")
	//nolint:revive,nolintlint // Not a real deep-exit, extracted from main for readability.
	flag.Parse()

	conf, confErr := config.LoadConfig(configDir)
	if confErr != nil {
		log.Fatal().Err(confErr).Msg("failed to load config")
	}

	return conf
}

func initMailer(conf *config.Config) {
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
}

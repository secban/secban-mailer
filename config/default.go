package config

func defaultParsedConfig() ParsedConfig {
	return ParsedConfig{
		Profiler:   "localhost:6062",
		LogLevel:   "info",
		SocketPath: "/var/run/secban/mailer.sock",
		StateFile:  "/var/run/secban/mailer_report",
		Database:   "/var/lib/secban/agent.sqlite",
		To:         []string{},
		SMTP: ParsedSMTPConfig{
			TLS: "opportunistic",
		},
	}
}

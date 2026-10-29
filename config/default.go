package config

func DefaultParsedConfig() ParsedConfig {
	return ParsedConfig{
		Profiler:   "",
		LogLevel:   "info",
		SocketPath: "/var/run/secban/mailer.sock",
		StateFile:  "/var/run/secban/mailer_report",
		Database:   "/var/lib/secban/agent.sqlite",
		From:       "secban",
		To:         []string{},
		SMTP: ParsedSMTPConfig{
			Host: "localhost:25",
			TLS:  "opportunistic",
		},
	}
}

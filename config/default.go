package config

func defaultParsedConfig() ParsedConfig {
	return ParsedConfig{
		LogLevel: "info",
		To:       []string{},
		SMTP: ParsedSMTPConfig{
			TLS: "opportunistic",
		},
	}
}

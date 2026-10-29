package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/secban/secban-mailer/config"
)

func TestConfigLoadWithFrequencyAndThreshold(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mailer.yaml")

	configData := []byte(
		"from: test@example.com\n" +
			"database: /tmp/test.sqlite\n" +
			"state_file: /tmp/mailer.state\n" +
			"frequency: 2h\n" +
			"threshold: 10\n" +
			"smtp:\n  host: localhost\n",
	)

	writeErr := os.WriteFile(configPath, configData, 0o600)
	if writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	cfg, err := config.LoadConfig(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Database != "/tmp/test.sqlite" {
		t.Fatalf("database = %q, want /tmp/test.sqlite", cfg.Database)
	}

	if cfg.StateFile != "/tmp/mailer.state" {
		t.Fatalf("state_file = %q, want /tmp/mailer.state", cfg.StateFile)
	}

	if cfg.Frequency != 2*time.Hour {
		t.Fatalf("frequency = %v, want 2h", cfg.Frequency)
	}

	if cfg.Threshold != 10 {
		t.Fatalf("threshold = %d, want 10", cfg.Threshold)
	}
}

func TestConfigRequiresDatabaseWhenFrequencySet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mailer.yaml")

	configData := []byte(
		"from: test@example.com\n" +
			"frequency: 1h\n" +
			"state_file: /tmp/mailer.state\n" +
			"database: ''\n" +
			"smtp:\n  host: localhost\n",
	)

	writeErr := os.WriteFile(configPath, configData, 0o600)
	if writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	_, err := config.LoadConfig(dir)
	if err == nil {
		t.Fatal("expected error when frequency set without database")
	}
}

func TestConfigRequiresStateFileWhenThresholdSet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mailer.yaml")

	configData := []byte(
		"from: test@example.com\n" +
			"database: /tmp/test.sqlite\n" +
			"threshold: 5\n" +
			"state_file: ''\n" +
			"smtp:\n  host: localhost\n",
	)

	writeErr := os.WriteFile(configPath, configData, 0o600)
	if writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	_, err := config.LoadConfig(dir)
	if err == nil {
		t.Fatal("expected error when threshold set without state_file")
	}
}

func TestConfigRejectsInvalidFrequency(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mailer.yaml")

	configData := []byte(
		"from: test@example.com\n" +
			"database: /tmp/test.sqlite\n" +
			"frequency: invalid\n" +
			"smtp:\n  host: localhost\n",
	)

	writeErr := os.WriteFile(configPath, configData, 0o600)
	if writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	_, err := config.LoadConfig(dir)
	if err == nil {
		t.Fatal("expected error for invalid frequency")
	}
}

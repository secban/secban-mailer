package mailer_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.com/secban/secban-agent/banlog"
	"gitlab.com/secban/secban-agent/db"
	"gitlab.com/secban/secban-agent/ipc"
	"gitlab.com/secban/secban-agent/offender"
	"gitlab.com/secban/secban-mailer/config"
	"gitlab.com/secban/secban-mailer/mailer"
)

type mockSender struct {
	sentSingle []ipc.Message
	sentGroup  [][]ipc.Message
}

func (m *mockSender) Send(_ context.Context, msg ipc.Message) error {
	m.sentSingle = append(m.sentSingle, msg)

	return nil
}

func (m *mockSender) SendGroup(_ context.Context, bans []ipc.Message) error {
	m.sentGroup = append(m.sentGroup, bans)

	return nil
}

func readStateTime(t *testing.T, stateFile string) time.Time {
	t.Helper()

	content, readErr := os.ReadFile(stateFile)
	if readErr != nil {
		t.Fatalf("read state file failed: %v", readErr)
	}

	parsed, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(content)))
	if parseErr != nil {
		fallback, fallbackErr := time.Parse(time.RFC3339, strings.TrimSpace(string(content)))
		if fallbackErr != nil {
			t.Fatalf("parse state timestamp failed: %v", parseErr)
		}

		return fallback
	}

	return parsed
}

func writeStateTime(t *testing.T, stateFile string, ts time.Time) {
	t.Helper()

	writeErr := os.WriteFile(stateFile, []byte(ts.Format(time.RFC3339Nano)), 0o600)
	if writeErr != nil {
		t.Fatalf("write state file failed: %v", writeErr)
	}
}

func TestReporterGroupedByThreshold(t *testing.T) { //nolint:paralleltest // Shared global database state in db package.
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agent.sqlite")
	statePath := filepath.Join(dir, "mailer.state")

	initErr := db.Init(ctx, dbPath)
	if initErr != nil {
		t.Fatalf("db.Init failed: %v", initErr)
	}
	defer func() {
		_ = db.Close()
	}()

	conf := &config.Config{
		Database:  dbPath,
		StateFile: statePath,
		Threshold: 2,
	}

	sender := &mockSender{}
	reporter := mailer.NewReporterWithSender(conf, sender)

	reporterInitErr := reporter.Init(ctx)
	if reporterInitErr != nil {
		t.Fatalf("reporter.Init failed: %v", reporterInitErr)
	}

	verifyThresholdWorkflow(ctx, t, reporter, sender, statePath)
}

func verifyThresholdWorkflow(
	ctx context.Context,
	t *testing.T,
	reporter *mailer.Reporter,
	sender *mockSender,
	statePath string,
) {
	t.Helper()

	stateBefore := readStateTime(t, statePath)

	// 1 ban: below threshold
	banlog.Log(ctx, "rule-1", offender.Offender{
		IP:        "1.1.1.1",
		Timestamp: stateBefore.Add(1 * time.Second),
	})

	handleErr := reporter.HandleBan(ctx, ipc.Message{IP: "1.1.1.1", Rule: "rule-1"})
	if handleErr != nil {
		t.Fatalf("HandleBan failed: %v", handleErr)
	}

	stateMiddle := readStateTime(t, statePath)
	if !stateMiddle.Equal(stateBefore) {
		t.Fatal("state timestamp should not update before threshold hit")
	}

	// 2nd ban: threshold hit
	banlog.Log(ctx, "rule-2", offender.Offender{
		IP:        "2.2.2.2",
		Timestamp: stateBefore.Add(2 * time.Second),
	})

	handleErr2 := reporter.HandleBan(ctx, ipc.Message{IP: "2.2.2.2", Rule: "rule-2"})
	if handleErr2 != nil {
		t.Fatalf("HandleBan 2 failed: %v", handleErr2)
	}

	stateAfter := readStateTime(t, statePath)
	if !stateAfter.After(stateBefore) {
		t.Fatal("state timestamp should advance after report trigger")
	}

	if len(sender.sentGroup) != 1 {
		t.Fatalf("expected 1 sentGroup email, got %d", len(sender.sentGroup))
	}

	if len(sender.sentGroup[0]) != 2 {
		t.Fatalf("expected 2 bans in grouped email, got %d", len(sender.sentGroup[0]))
	}
}

func TestReporterGroupedByFrequency(t *testing.T) { //nolint:paralleltest // Shared global database state in db package.
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "agent.sqlite")
	statePath := filepath.Join(dir, "mailer.state")

	initErr := db.Init(ctx, dbPath)
	if initErr != nil {
		t.Fatalf("db.Init failed: %v", initErr)
	}
	defer func() {
		_ = db.Close()
	}()

	conf := &config.Config{
		Database:  dbPath,
		StateFile: statePath,
		Frequency: 1 * time.Hour,
	}

	sender := &mockSender{}
	reporter := mailer.NewReporterWithSender(conf, sender)

	reporterInitErr := reporter.Init(ctx)
	if reporterInitErr != nil {
		t.Fatalf("reporter.Init failed: %v", reporterInitErr)
	}

	oldTime := time.Now().Add(-2 * time.Hour)
	writeStateTime(t, statePath, oldTime)

	banlog.Log(ctx, "rule-old", offender.Offender{
		IP:        "3.3.3.3",
		Timestamp: oldTime.Add(10 * time.Minute),
	})

	handleErr := reporter.HandleBan(ctx, ipc.Message{IP: "3.3.3.3", Rule: "rule-old"})
	if handleErr != nil {
		t.Fatalf("HandleBan failed: %v", handleErr)
	}

	stateAfter := readStateTime(t, statePath)
	if !stateAfter.After(oldTime) {
		t.Fatal("state timestamp should advance after frequency reached")
	}

	if len(sender.sentGroup) != 1 {
		t.Fatalf("expected 1 sentGroup email, got %d", len(sender.sentGroup))
	}
}

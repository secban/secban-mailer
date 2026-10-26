package mailer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/secban/secban-agent/banlog"
	"gitlab.com/secban/secban-agent/db"
	"gitlab.com/secban/secban-agent/ipc"
	"gitlab.com/secban/secban-agent/offender"
	"gitlab.com/secban/secban-mailer/config"
)

const (
	stateDirPerm  = 0o750
	stateFilePerm = 0o600
)

type Sender interface {
	Send(ctx context.Context, msg ipc.Message) error
	SendGroup(ctx context.Context, bans []ipc.Message) error
}

type defaultSender struct{}

func (defaultSender) Send(ctx context.Context, msg ipc.Message) error {
	return Send(ctx, msg)
}

func (defaultSender) SendGroup(ctx context.Context, bans []ipc.Message) error {
	return SendGroup(ctx, bans)
}

type Reporter struct {
	config *config.Config
	sender Sender
	mu     sync.Mutex
}

func NewReporter(conf *config.Config) *Reporter {
	return &Reporter{
		config: conf,
		sender: defaultSender{},
	}
}

func NewReporterWithSender(conf *config.Config, sender Sender) *Reporter {
	return &Reporter{
		config: conf,
		sender: sender,
	}
}

func (r *Reporter) Init(ctx context.Context) error {
	if !r.isGrouped() {
		return nil
	}

	initErr := db.Init(ctx, r.config.Database)
	if initErr != nil {
		return initErr
	}

	_, readErr := r.readState()
	if readErr != nil {
		if !errors.Is(readErr, os.ErrNotExist) {
			return readErr
		}

		writeErr := r.writeState(time.Now())
		if writeErr != nil {
			return writeErr
		}
	}

	return nil
}

func (r *Reporter) HandleBan(ctx context.Context, msg ipc.Message) error {
	if !r.isGrouped() {
		return r.sender.Send(ctx, msg)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	lastReport, readErr := r.readState()
	if readErr != nil {
		return readErr
	}

	now := time.Now()

	shouldReport, checkErr := r.shouldReport(ctx, lastReport, now)
	if checkErr != nil {
		return checkErr
	}

	if !shouldReport {
		return nil
	}

	entries, getEntriesErr := banlog.GetSince(ctx, lastReport)
	if getEntriesErr != nil {
		return getEntriesErr
	}

	if len(entries) == 0 {
		return r.writeState(now)
	}

	bans := toMessages(entries)

	sendErr := r.sender.SendGroup(ctx, bans)
	if sendErr != nil {
		return sendErr
	}

	return r.writeState(now)
}

func (r *Reporter) readState() (time.Time, error) {
	content, readErr := os.ReadFile(r.config.StateFile)
	if readErr != nil {
		return time.Time{}, readErr
	}

	trimmed := strings.TrimSpace(string(content))

	parsedTime, parseErr := time.Parse(time.RFC3339Nano, trimmed)
	if parseErr != nil {
		// Fall back to standard RFC3339 if nano fails.
		fallbackTime, fallbackErr := time.Parse(time.RFC3339, trimmed)
		if fallbackErr != nil {
			return time.Time{}, parseErr
		}

		return fallbackTime, nil
	}

	return parsedTime, nil
}

func (r *Reporter) writeState(timestamp time.Time) error {
	dir := filepath.Dir(r.config.StateFile)

	mkdirErr := os.MkdirAll(dir, stateDirPerm)
	if mkdirErr != nil {
		return mkdirErr
	}

	formatted := timestamp.Format(time.RFC3339Nano)

	writeErr := os.WriteFile(r.config.StateFile, []byte(formatted), stateFilePerm)
	if writeErr != nil {
		return writeErr
	}

	return nil
}

func (r *Reporter) shouldReport(ctx context.Context, lastReport time.Time, now time.Time) (bool, error) {
	if r.config.Frequency > 0 && now.Sub(lastReport) >= r.config.Frequency {
		return true, nil
	}

	if r.config.Threshold > 0 {
		count, countErr := banlog.CountSince(ctx, lastReport)
		if countErr != nil {
			return false, countErr
		}

		if count >= r.config.Threshold {
			return true, nil
		}
	}

	return false, nil
}

func toMessages(entries []*banlog.Entry) []ipc.Message {
	bans := make([]ipc.Message, 0, len(entries))
	for _, entry := range entries {
		o := parseOffender(entry)
		bans = append(bans, ipc.Message{
			IP:       entry.IP,
			Rule:     entry.Rule,
			Offender: o,
		})
	}

	return bans
}

func parseOffender(entry *banlog.Entry) offender.Offender {
	if entry.Events == "" {
		return offender.Offender{
			IP:        entry.IP,
			Timestamp: entry.BanTime,
			BanCount:  entry.BanCount,
		}
	}

	var o offender.Offender

	unmarshalErr := json.Unmarshal([]byte(entry.Events), &o)
	if unmarshalErr != nil {
		log.Warn().Err(unmarshalErr).Str("ip", entry.IP).Msg("failed to unmarshal offender events")

		return offender.Offender{
			IP:        entry.IP,
			Timestamp: entry.BanTime,
			BanCount:  entry.BanCount,
		}
	}

	return o
}

func (r *Reporter) isGrouped() bool {
	return r.config.Frequency > 0 || r.config.Threshold > 0
}

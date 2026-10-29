package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rs/zerolog"
)

func TestZerologEmitsStructuredEvents(t *testing.T) {
	var output bytes.Buffer
	log := &zerologLog{client: zerolog.New(&output).With().Timestamp().Logger()}
	log.Info("started")
	log.Debug("debugging")
	log.Warn("warning")
	log.Error("failed", errors.New("database unavailable"))
	decoder := json.NewDecoder(&output)
	for _, tc := range []struct{ level, message, err string }{
		{"info", "started", ""},
		{"debug", "debugging", ""},
		{"warn", "warning", ""},
		{"error", "failed", "database unavailable"},
	} {
		var event map[string]any
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		if event["level"] != tc.level || event["message"] != tc.message || event["time"] == nil {
			t.Fatalf("unexpected event: %#v", event)
		}
		if tc.err != "" && event["error"] != tc.err {
			t.Fatalf("missing error: %#v", event)
		}
	}
}

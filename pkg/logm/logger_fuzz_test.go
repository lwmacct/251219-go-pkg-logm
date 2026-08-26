package logm

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func FuzzJSONHandlerNeverEmitsInvalidJSON(f *testing.F) {
	f.Add("hello", "key", "value")
	f.Add("quote\"newline\n", "weird\\key", "x")
	f.Fuzz(func(t *testing.T, msg, key, value string) {
		var buf bytes.Buffer
		l, err := New(Config{Outputs: []Output{{Writer: &buf, Format: FormatJSON}}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if closeErr := l.Close(); closeErr != nil {
				t.Errorf("close logger: %v", closeErr)
			}
		})
		l.Info(msg, slog.String(key, value))
		var decoded map[string]any
		if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &decoded); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
	})
}

package logm

import (
	"io"
	"log/slog"
	"testing"
)

func BenchmarkJSONLogger(b *testing.B) {
	l, err := New(Config{Level: slog.LevelInfo, Outputs: []Output{{Writer: io.Discard, Format: FormatJSON}}})
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		l.Info("benchmark", "ok", true)
	}
}

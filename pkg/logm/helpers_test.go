package logm

import (
	"bytes"
	"context"
	"log/slog"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogWithPC(t *testing.T) {
	var buf bytes.Buffer

	handler := NewHandler(&HandlerConfig{
		LevelVar:   &slog.LevelVar{},
		Format:     FormatText,
		Output:     &testWriter{buf: &buf},
		AddSource:  true,
		TimeFormat: "time",
	})

	slog.SetDefault(slog.New(handler))

	var pcs [1]uintptr
	runtime.Callers(1, pcs[:])
	pc := pcs[0]

	LogWithPC(context.Background(), slog.LevelInfo, pc, "test message", slog.String("key", "value"))

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=value")
	assert.Contains(t, output, "helpers_test.go")
}

func TestLogWithPC_ZeroPC(t *testing.T) {
	var buf bytes.Buffer

	handler := NewHandler(&HandlerConfig{
		LevelVar:  &slog.LevelVar{},
		Format:    FormatText,
		Output:    &testWriter{buf: &buf},
		AddSource: true,
	})

	slog.SetDefault(slog.New(handler))
	LogWithPC(context.Background(), slog.LevelInfo, 0, "test message")

	assert.Contains(t, buf.String(), "test message")
}

func TestLogWithPC_LevelDisabled(t *testing.T) {
	var buf bytes.Buffer
	levelVar := &slog.LevelVar{}
	levelVar.Set(slog.LevelWarn)

	handler := NewHandler(&HandlerConfig{
		LevelVar: levelVar,
		Format:   FormatText,
		Output:   &testWriter{buf: &buf},
	})

	slog.SetDefault(slog.New(handler))
	LogWithPC(context.Background(), slog.LevelInfo, 0, "should not appear")

	assert.NotContains(t, buf.String(), "should not appear")
}

func TestLogWithPC_WithCallerPC(t *testing.T) {
	var buf bytes.Buffer

	handler := NewHandler(&HandlerConfig{
		LevelVar:  &slog.LevelVar{},
		Format:    FormatText,
		Output:    &testWriter{buf: &buf},
		AddSource: true,
	})

	slog.SetDefault(slog.New(handler))
	LogWithPC(context.Background(), slog.LevelInfo, CallerPC(), "combined test")

	output := buf.String()
	assert.Contains(t, output, "combined test")
	assert.Contains(t, output, "helpers_test.go")
}

package logm

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_Default(t *testing.T) {
	err := Init()
	require.NoError(t, err)
	defer func() { _ = Close() }()

	slog.Info("test message")
}

func TestNew_WithJSONFormat(t *testing.T) {
	var buf bytes.Buffer

	log := New(Config{
		Level:  "DEBUG",
		Format: FormatJSON,
		Output: &testWriter{buf: &buf},
	})
	require.NotNil(t, log)

	log.Info("hello", "user", "alice")

	output := buf.String()
	assert.Contains(t, output, `"msg":"hello"`)
	assert.Contains(t, output, `"user":"alice"`)
	assert.Contains(t, output, `"level":"INFO"`)
}

func TestNew_WithSlogHandler(t *testing.T) {
	var textBuf bytes.Buffer
	var jsonBuf bytes.Buffer

	log := New(Config{
		Format:       FormatText,
		Output:       &testWriter{buf: &textBuf},
		SlogHandlers: []slog.Handler{slog.NewJSONHandler(&jsonBuf, nil)},
	})
	require.NotNil(t, log)

	log.Info("hello", "user", "alice")

	assert.Contains(t, textBuf.String(), "msg=hello")
	assert.Contains(t, textBuf.String(), "user=alice")
	assert.Contains(t, jsonBuf.String(), `"msg":"hello"`)
	assert.Contains(t, jsonBuf.String(), `"user":"alice"`)
}

func TestNew_WithLevelVar(t *testing.T) {
	var buf bytes.Buffer
	levelVar := &slog.LevelVar{}
	levelVar.Set(slog.LevelError)

	log := New(Config{
		LevelVar: levelVar,
		Output:   &testWriter{buf: &buf},
	})
	require.NotNil(t, log)

	log.Info("filtered")
	assert.NotContains(t, buf.String(), "filtered")

	levelVar.Set(slog.LevelInfo)
	log.Info("visible")
	assert.Contains(t, buf.String(), "visible")
}

func TestHandler_ErrorSourceTimeAndExpandJSON(t *testing.T) {
	var buf bytes.Buffer

	handler := NewHandler(&HandlerConfig{
		Format:     FormatText,
		Output:     &testWriter{buf: &buf},
		AddSource:  true,
		TimeFormat: "time",
		ExpandJSON: true,
	})

	logger := slog.New(handler)
	logger.Error("boom", "error", errors.New("explode"), "payload", `{"count":2,"user":"alice"}`)

	output := buf.String()
	assert.Contains(t, output, "error=explode")
	assert.Contains(t, output, "payload.count=2")
	assert.Contains(t, output, "payload.user=alice")
	assert.Contains(t, output, "source=")
	assert.Contains(t, output, ".go:")
	assert.Regexp(t, `time=\d{2}:\d{2}:\d{2}`, output)
}

func TestHandler_ColorOutput(t *testing.T) {
	var buf bytes.Buffer

	handler := NewHandler(&HandlerConfig{
		Format: FormatText,
		Output: &testWriter{buf: &buf},
		Color:  true,
	})

	slog.New(handler).Error("boom")
	assert.Contains(t, buf.String(), "\x1b[31m")
}

func TestHandler_WithAttrsAndGroup(t *testing.T) {
	var buf bytes.Buffer

	handler := NewHandler(&HandlerConfig{
		Format: FormatText,
		Output: &testWriter{buf: &buf},
	})

	logger := slog.New(handler).With("service", "api").WithGroup("request")
	logger.Info("started", "method", "POST")

	output := buf.String()
	assert.Contains(t, output, "service=api")
	assert.Contains(t, output, "request.method=POST")
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	err := Init(Config{Level: "INFO", Output: &testWriter{buf: &buf}})
	require.NoError(t, err)
	defer func() { _ = Close() }()

	SetLevel("DEBUG")
	slog.Debug("debug message")
	assert.Contains(t, buf.String(), "debug message")
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"UNKNOWN", slog.LevelInfo},
		{"debug", slog.LevelDebug},
		{"Info", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseLevel(tt.input))
		})
	}
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()
	logger := FromContext(ctx)
	assert.NotNil(t, logger)

	customLogger := slog.Default().With("custom", "value")
	ctxWithLogger := WithLogger(ctx, customLogger)
	assert.Equal(t, customLogger, FromContext(ctxWithLogger))
}

func TestWithRequestID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "test-request-123")
	assert.NotNil(t, FromContext(ctx))
}

func TestFormatBytes(t *testing.T) {
	assert.Equal(t, "0 B", FormatBytes(0))
	assert.Equal(t, "1.0 KB", FormatBytes(1024))
	assert.Equal(t, "1.5 MB", FormatBytes(1536*1024))
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	err := LogError(context.Background(), "operation failed", context.DeadlineExceeded, "user_id", "123")
	assert.Equal(t, context.DeadlineExceeded, err)

	output := buf.String()
	assert.Contains(t, output, "operation failed")
	assert.Contains(t, output, "user_id=123")
	assert.Contains(t, output, "error=\"context deadline exceeded\"")
}

func TestLogAndWrap(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	wrappedErr := LogAndWrap("fetch failed", context.DeadlineExceeded, "url", "http://example.com")
	assert.Contains(t, wrappedErr.Error(), "fetch failed")
	assert.Contains(t, buf.String(), "fetch failed")
}

func TestClipWorkspacePath(t *testing.T) {
	assert.Equal(t, "main.go:146", clipWorkspacePath("/workspace/251127-ai-agent-hatch/main.go:146"))
	assert.Equal(t, "pkg/logm/logm.go:100", clipWorkspacePath("/apps/data/workspace/251219-go-pkg-logm/pkg/logm/logm.go:100"))
	assert.Empty(t, clipWorkspacePath(""))
}

func TestDebugInfoWarnError(t *testing.T) {
	var buf bytes.Buffer
	err := Init(Config{Level: "DEBUG", Output: &testWriter{buf: &buf}})
	require.NoError(t, err)
	defer func() { _ = Close() }()

	Debug("debug message", "key", "value")
	Info("info message", "key", "value")
	Warn("warn message", "key", "value")
	Error("error message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestWithAndWithGroup(t *testing.T) {
	err := Init(Config{Level: "INFO", Output: &testWriter{buf: &bytes.Buffer{}}})
	require.NoError(t, err)
	defer func() { _ = Close() }()

	assert.NotNil(t, With("module", "test"))
	assert.NotNil(t, WithGroup("request"))
}

func TestClose_Multiple(t *testing.T) {
	err := Init()
	require.NoError(t, err)

	require.NoError(t, Close())
	require.NoError(t, Close())
}

func TestClose_RestoresPreviousDefault(t *testing.T) {
	previous := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	slog.SetDefault(previous)

	err := Init(Config{Output: &testWriter{buf: &bytes.Buffer{}}})
	require.NoError(t, err)

	require.NoError(t, Close())
	assert.Same(t, previous, slog.Default())
}

func TestInit_WithManagedSlogHandler(t *testing.T) {
	managed := &managedTestHandler{}

	err := Init(Config{
		Output:       &testWriter{buf: &bytes.Buffer{}},
		SlogHandlers: []slog.Handler{managed},
	})
	require.NoError(t, err)

	require.NoError(t, Sync())
	assert.Equal(t, 1, managed.syncCalls)

	require.NoError(t, Close())
	assert.Equal(t, 1, managed.closeCalls)
}

// testWriter 是一个简单的 Writer 实现用于测试。
type testWriter struct {
	buf *bytes.Buffer
}

func (w *testWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *testWriter) Close() error {
	return nil
}

func (w *testWriter) Sync() error {
	return nil
}

type managedTestHandler struct {
	syncCalls  int
	closeCalls int
}

func (h *managedTestHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *managedTestHandler) Handle(context.Context, slog.Record) error {
	return nil
}

func (h *managedTestHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *managedTestHandler) WithGroup(string) slog.Handler {
	return h
}

func (h *managedTestHandler) Sync() error {
	h.syncCalls++
	return nil
}

func (h *managedTestHandler) Close() error {
	h.closeCalls++
	return nil
}

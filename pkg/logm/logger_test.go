package logm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestNewJSONUsesStandardSemantics(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Config{
		Level:     slog.LevelDebug,
		Outputs:   []Output{{Writer: &buf, Format: FormatJSON}},
		AddSource: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := l.Close(); closeErr != nil {
			t.Errorf("close logger: %v", closeErr)
		}
	})
	l.Info("hello", slog.Group("request", slog.String("path", "/")), slog.Any("err", errors.New("bad")))
	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &got); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, buf.String())
	}
	if got["msg"] != "hello" {
		t.Fatalf("msg = %#v", got["msg"])
	}
	request, ok := got["request"].(map[string]any)
	if !ok || request["path"] != "/" {
		t.Fatalf("request = %#v", got["request"])
	}
	if got["err"] != "bad" {
		t.Fatalf("err = %#v", got["err"])
	}
}

func TestNewDefaultConfig(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Config{Outputs: []Output{{Writer: &buf, Format: FormatText}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := l.Close(); closeErr != nil {
			t.Errorf("close logger: %v", closeErr)
		}
	})
	l.Info("default")
	if !strings.Contains(buf.String(), "msg=default") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestPrettyOutputColorsLevel(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Config{Outputs: []Output{{Writer: &buf, Format: FormatPretty}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := l.Close(); closeErr != nil {
			t.Errorf("close logger: %v", closeErr)
		}
	})
	l.Warn("warning")
	if !strings.Contains(buf.String(), "\x1b[33mWARN") {
		t.Fatalf("pretty output = %q", buf.String())
	}
}

func TestMiddlewareCanDropAndRewrite(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Config{
		Outputs: []Output{{Writer: &buf, Format: FormatText}},
		Middleware: []Middleware{func(_ context.Context, r slog.Record) (slog.Record, bool) {
			r.Message = strings.ToUpper(r.Message)
			return r, r.Message != "DROP"
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := l.Close(); closeErr != nil {
			t.Errorf("close logger: %v", closeErr)
		}
	})
	l.Info("hello")
	l.Info("drop")
	if !strings.Contains(buf.String(), "msg=HELLO") || strings.Contains(buf.String(), "DROP") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestRequestID(t *testing.T) {
	ctx := WithNewRequestID(context.Background())
	if id := RequestID(ctx); len(id) != 36 {
		t.Fatalf("request id = %q", id)
	}
	if FromContext(ctx) == nil {
		t.Fatal("missing logger")
	}
}

func TestParseLevelStrict(t *testing.T) {
	if _, err := ParseLevel("unknown"); err == nil {
		t.Fatal("expected invalid level error")
	}
	level, err := ParseLevel(" warning ")
	if err != nil || level != slog.LevelWarn {
		t.Fatalf("ParseLevel = %v, %v", level, err)
	}
}

func TestInitCloseLifecycleAndLevel(t *testing.T) {
	previous := slog.Default()
	var buf bytes.Buffer
	if err := Init(Config{Level: slog.LevelError, Outputs: []Output{{Writer: &buf, Format: FormatText}}}); err != nil {
		t.Fatal(err)
	}
	slog.Info("hidden")
	slog.Error("shown")
	if strings.Contains(buf.String(), "hidden") || !strings.Contains(buf.String(), "shown") {
		t.Fatalf("level filtering output=%q", buf.String())
	}
	if err := Close(); err != nil {
		t.Fatal(err)
	}
	if slog.Default() != previous {
		t.Fatal("Close did not restore previous slog default")
	}
}

func TestSetLevelAffectsGlobalPreset(t *testing.T) {
	var buf bytes.Buffer
	cfg := PresetProd()
	cfg.Outputs = []Output{{Writer: &buf, Format: FormatText}}
	if err := Init(cfg); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Close(); SetLevelValue(slog.LevelInfo) }()
	if err := SetLevel("DEBUG"); err != nil {
		t.Fatal(err)
	}
	slog.Debug("visible")
	if !strings.Contains(buf.String(), "visible") {
		t.Fatal("SetLevel did not update global handler")
	}
}

func TestCustomLevelerRemainsDynamic(t *testing.T) {
	var buf bytes.Buffer
	levels := new(slog.LevelVar)
	levels.Set(slog.LevelInfo)
	l, err := New(Config{Level: levels, Outputs: []Output{{Writer: &buf, Format: FormatText}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := l.Close(); closeErr != nil {
			t.Errorf("close logger: %v", closeErr)
		}
	})
	l.Debug("hidden")
	levels.Set(slog.LevelDebug)
	l.Debug("visible")
	if strings.Contains(buf.String(), "hidden") || !strings.Contains(buf.String(), "visible") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestOnErrorReceivesSinkFailure(t *testing.T) {
	want := errors.New("sink failed")
	var got error
	l, err := New(Config{
		Outputs: []Output{{Writer: failingWriter{err: want}, Format: FormatText}},
		OnError: func(err error) { got = err },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := l.Close(); closeErr != nil {
			t.Errorf("close logger: %v", closeErr)
		}
	})
	l.Info("hello")
	if !errors.Is(got, want) {
		t.Fatalf("OnError = %v", got)
	}
}

type testLogValuer struct{}

func (testLogValuer) LogValue() slog.Value { return slog.StringValue("resolved") }

func TestReplaceAttrReceivesResolvedValue(t *testing.T) {
	var buf bytes.Buffer
	var kind slog.Kind
	l, err := New(Config{
		Outputs: []Output{{Writer: &buf, Format: FormatJSON}},
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == "value" {
				kind = a.Value.Kind()
			}
			return a
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := l.Close(); closeErr != nil {
			t.Errorf("close logger: %v", closeErr)
		}
	})
	l.Info("x", "value", testLogValuer{})
	if kind != slog.KindString {
		t.Fatalf("ReplaceAttr kind = %v", kind)
	}
}

func TestLogWithPCUsesProvidedProgramCounter(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Config{Outputs: []Output{{Writer: &buf, Format: FormatJSON}}, AddSource: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := l.Close(); closeErr != nil {
			t.Errorf("close logger: %v", closeErr)
		}
	})
	ctx := WithLogger(context.Background(), l.Logger)
	pc := CallerPC("logm.TestLogWithPCUsesProvidedProgramCounter")
	LogWithPC(ctx, slog.LevelInfo, pc, "pc")
	if !strings.Contains(buf.String(), `"source"`) {
		t.Fatalf("source missing: %s", buf.String())
	}
}

func TestClipSourcePath(t *testing.T) {
	tests := []struct {
		path, prefix string
		depth        int
		want         string
	}{
		{"/workspace/project/pkg/logm/file.go", "/workspace/", 3, "pkg/logm/file.go"},
		{"/a/b/c/d.go", "", 2, "c/d.go"},
	}
	for _, tt := range tests {
		if got := clipSourcePath(tt.path, tt.prefix, tt.depth); got != tt.want {
			t.Errorf("clipSourcePath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestReplaceAttrAndSourceClip(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Config{
		Outputs:    []Output{{Writer: &buf, Format: FormatJSON}},
		AddSource:  true,
		SourceClip: "/workspace/",
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "secret" {
				return slog.String("secret", "redacted")
			}
			return a
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := l.Close(); closeErr != nil {
			t.Errorf("close logger: %v", closeErr)
		}
	})
	l.Info("hello", "secret", "value")
	if !strings.Contains(buf.String(), `"secret":"redacted"`) {
		t.Fatalf("replace attr failed: %s", buf.String())
	}
}

type closeBuffer struct {
	bytes.Buffer
	closed bool
}

func (w *closeBuffer) Close() error { w.closed = true; return nil }

func TestNewOwnOutputCloses(t *testing.T) {
	w := new(closeBuffer)
	l, err := New(Config{Outputs: []Output{{Writer: w, Format: FormatText, Own: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if !w.closed {
		t.Fatal("owned output was not closed")
	}
}

func TestLoadConfigFromEnvStrictAndFormatOrder(t *testing.T) {
	t.Setenv("LOGM_ENV", "prod")
	t.Setenv("LOGM_OUTPUT", "stderr")
	t.Setenv("LOGM_FORMAT", "json")
	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Outputs) != 1 || cfg.Outputs[0].Format != FormatJSON {
		t.Fatalf("outputs = %#v", cfg.Outputs)
	}
	t.Setenv("LOGM_LEVEL", "bogus")
	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("expected invalid level")
	}
	t.Setenv("LOGM_LEVEL", "INFO")
	t.Setenv("LOGM_ENV", "staging")
	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("expected invalid environment")
	}
}

package logm_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm"
)

//nolint:testableexamples
func Example() {
	_ = logm.Init(logm.Config{
		Level:  "DEBUG",
		Format: logm.FormatText,
	})
	defer func() { _ = logm.Close() }()
}

func Example_slogCompatible() {
	var buf bytes.Buffer

	_ = logm.Init(logm.Config{
		Format: logm.FormatJSON,
		Output: &testBufWriter{buf: &buf},
		Level:  "INFO",
	})
	defer func() { _ = logm.Close() }()

	slog.Info("hello from slog", "user", "alice")

	output := buf.String()
	if strings.Contains(output, `"msg":"hello from slog"`) &&
		strings.Contains(output, `"user":"alice"`) &&
		strings.Contains(output, `"level":"INFO"`) {
		fmt.Println("slog uses logm handler")
	}
	// Output: slog uses logm handler
}

//nolint:testableexamples
func Example_development() {
	_ = logm.Init(logm.PresetDev())
	defer func() { _ = logm.Close() }()
}

//nolint:testableexamples
func Example_production() {
	_ = logm.Init(logm.PresetProd())
	defer func() { _ = logm.Close() }()
}

//nolint:testableexamples
func Example_fromEnv() {
	_ = logm.Init(logm.PresetFromEnv())
	defer func() { _ = logm.Close() }()
}

//nolint:testableexamples
func Example_withRequestID() {
	ctx := context.Background()
	ctx = logm.WithRequestID(ctx, "req-12345")
	_ = logm.FromContext(ctx)
}

//nolint:testableexamples
func Example_new() {
	_ = logm.New(logm.Config{
		Level:     "INFO",
		Format:    logm.FormatJSON,
		AddSource: true,
	})
}

func Example_multiHandler() {
	var textBuf bytes.Buffer
	var jsonBuf bytes.Buffer

	log := logm.New(logm.Config{
		Format:       logm.FormatText,
		Output:       &testBufWriter{buf: &textBuf},
		SlogHandlers: []slog.Handler{slog.NewJSONHandler(&jsonBuf, nil)},
	})

	log.Info("hello", "user", "alice")

	if strings.Contains(textBuf.String(), "hello") &&
		strings.Contains(jsonBuf.String(), `"user":"alice"`) {
		fmt.Println("multi handler works")
	}
	// Output: multi handler works
}

func ExampleFormatBytes() {
	fmt.Println(logm.FormatBytes(0))
	fmt.Println(logm.FormatBytes(1024))
	fmt.Println(logm.FormatBytes(1024 * 1024))
	fmt.Println(logm.FormatBytes(1536 * 1024 * 1024))
	// Output:
	// 0 B
	// 1.0 KB
	// 1.0 MB
	// 1.5 GB
}

func Example_with() {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	moduleLog := logm.With("module", "worker", "version", "2.0")
	moduleLog.Info("任务完成", "count", 42)

	output := buf.String()
	if strings.Contains(output, "module=worker") &&
		strings.Contains(output, "count=42") {
		fmt.Println("with works")
	}
	// Output: with works
}

func ExampleLogError() {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	returnedErr := logm.LogError(context.Background(), "操作超时", context.DeadlineExceeded, "timeout", "5s")
	if errors.Is(returnedErr, context.DeadlineExceeded) {
		fmt.Println("original error returned")
	}
	// Output: original error returned
}

func ExampleWithGroup() {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	reqLog := logm.WithGroup("request")
	reqLog.Info("处理请求", "method", "GET", "path", "/api/users")

	output := buf.String()
	if strings.Contains(output, `"request":{`) {
		fmt.Println("group works")
	}
	// Output: group works
}

//nolint:testableexamples
func ExampleFromContext() {
	ctx := context.Background()
	_ = logm.FromContext(ctx)

	customLog := slog.Default().With("service", "api")
	ctx = logm.WithLogger(ctx, customLog)
	_ = logm.FromContext(ctx)
}

//nolint:testableexamples
func Example_coloredOutput() {
	_ = logm.Init(logm.Config{
		Level:      "DEBUG",
		Format:     logm.FormatText,
		Color:      true,
		TimeFormat: "time",
		AddSource:  true,
	})
	defer func() { _ = logm.Close() }()
}

//nolint:testableexamples
func Example_jsonOutput() {
	_ = logm.Init(logm.Config{
		Level:      "INFO",
		Format:     logm.FormatJSON,
		TimeFormat: "rfc3339ms",
	})
	defer func() { _ = logm.Close() }()
}

//nolint:testableexamples
func Example_dynamicLevel() {
	_ = logm.Init(logm.Config{Level: "INFO"})
	defer func() { _ = logm.Close() }()

	logm.SetLevel("DEBUG")
	logm.SetLevel("INFO")
}

func Example_slogGroup() {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	slog.Info("HTTP 请求",
		slog.Group("request",
			"method", "GET",
			"path", "/api/users",
		),
		slog.Group("response",
			"status", 200,
			"size", 1024,
		),
	)

	output := buf.String()
	if strings.Contains(output, `"request":{`) &&
		strings.Contains(output, `"response":{`) {
		fmt.Println("slog.Group works")
	}
	// Output: slog.Group works
}

func ExampleParseLevel() {
	fmt.Println(logm.ParseLevel("DEBUG"))
	fmt.Println(logm.ParseLevel("INFO"))
	fmt.Println(logm.ParseLevel("WARN"))
	fmt.Println(logm.ParseLevel("ERROR"))
	fmt.Println(logm.ParseLevel("unknown"))
	// Output:
	// DEBUG
	// INFO
	// WARN
	// ERROR
	// INFO
}

type testBufWriter struct {
	buf *bytes.Buffer
}

func (w *testBufWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *testBufWriter) Close() error {
	return nil
}

func (w *testBufWriter) Sync() error {
	return nil
}

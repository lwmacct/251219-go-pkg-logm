package logm

import (
	"bytes"
	"context"
	"log/slog"
	"runtime"
	"strings"
	"testing"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter"
	"github.com/stretchr/testify/assert"
)

func TestLogWithPC(t *testing.T) {
	// 创建一个捕获输出的 buffer
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	// 创建 handler，启用 source
	handler := NewHandler(&HandlerConfig{
		LevelVar:   &slog.LevelVar{},
		Formatter:  formatter.Text(),
		Writers:    []Writer{stdoutWriter},
		AddSource:  true,
		TimeFormat: "15:04:05",
	})

	// 设置为默认 logger
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// 获取当前位置的 PC
	var pcs [1]uintptr
	runtime.Callers(1, pcs[:])
	pc := pcs[0]

	// 使用 LogWithPC 记录日志
	ctx := context.Background()
	LogWithPC(ctx, slog.LevelInfo, pc, "test message", slog.String("key", "value"))

	output := buf.String()

	// 验证输出包含消息
	assert.Contains(t, output, "test message")

	// 验证输出包含属性
	assert.Contains(t, output, "key=value")

	// 验证输出包含正确的 source（应该指向测试文件）
	assert.Contains(t, output, "helpers_test.go")
}

func TestLogWithPC_ZeroPC(t *testing.T) {
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	handler := NewHandler(&HandlerConfig{
		LevelVar:   &slog.LevelVar{},
		Formatter:  formatter.Text(),
		Writers:    []Writer{stdoutWriter},
		AddSource:  true,
		TimeFormat: "15:04:05",
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	ctx := context.Background()
	// 传入 0 作为 PC
	LogWithPC(ctx, slog.LevelInfo, 0, "test message")

	output := buf.String()

	// 验证输出包含消息
	assert.Contains(t, output, "test message")
}

func TestLogWithPC_LevelDisabled(t *testing.T) {
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	levelVar := &slog.LevelVar{}
	levelVar.Set(slog.LevelWarn) // 只启用 WARN 及以上

	handler := NewHandler(&HandlerConfig{
		LevelVar:   levelVar,
		Formatter:  formatter.Text(),
		Writers:    []Writer{stdoutWriter},
		AddSource:  true,
		TimeFormat: "15:04:05",
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	ctx := context.Background()
	// 尝试记录 INFO 级别日志（应该被过滤）
	LogWithPC(ctx, slog.LevelInfo, 0, "should not appear")

	output := buf.String()

	// 验证输出为空
	if strings.Contains(output, "should not appear") {
		t.Errorf("INFO log should be filtered when level is WARN, got: %s", output)
	}
}

func TestLogWithPC_WithCallerPC(t *testing.T) {
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	handler := NewHandler(&HandlerConfig{
		LevelVar:   &slog.LevelVar{},
		Formatter:  formatter.Text(),
		Writers:    []Writer{stdoutWriter},
		AddSource:  true,
		TimeFormat: "15:04:05",
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// 使用 CallerPC + LogWithPC 组合
	ctx := context.Background()
	pc := CallerPC()
	LogWithPC(ctx, slog.LevelInfo, pc, "combined test")

	output := buf.String()

	// 验证输出包含消息
	assert.Contains(t, output, "combined test")

	// 验证 source 指向当前测试文件
	assert.Contains(t, output, "helpers_test.go")
}

// Package logger 提供统一的结构化日志系统
//
// 基于 Go 1.21+ 的 log/slog 包，提供：
// 1. 统一的日志配置和初始化
// 2. 支持多种输出格式（JSON、文本）
// 3. 灵活的日志级别控制
// 4. 结构化日志的最佳实践
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config 日志配置
type Config struct {
	// Level 日志级别: DEBUG, INFO, WARN, ERROR
	Level string
	// Format 输出格式: json, text
	Format string
	// Output 输出目标: stdout, stderr, 或文件路径
	Output string
	// AddSource 是否添加源代码位置信息
	AddSource bool
	// TimeFormat 时间格式: datetime (默认), rfc3339, rfc3339ms, unix, unixms
	TimeFormat string
	// Timezone 时区名称，例如 "Asia/Shanghai"，默认为 "Asia/Shanghai"
	Timezone string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Level:      "INFO",
		Format:     "text",
		Output:     "stdout",
		AddSource:  false,
		TimeFormat: "datetime", // 默认格式: 2006-01-02 15:04:05
		Timezone:   "Asia/Shanghai",
	}
}

// validFormats 有效的输出格式
var validFormats = map[string]bool{
	"json": true, "text": true, "color": true, "colored": true,
}

// validLevels 有效的日志级别
var validLevels = map[string]bool{
	"DEBUG": true, "INFO": true, "WARN": true, "WARNING": true, "ERROR": true,
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	if c.Format != "" && !validFormats[c.Format] {
		return fmt.Errorf("invalid log format: %q, valid options: json, text, color", c.Format)
	}

	level := strings.ToUpper(c.Level)
	if c.Level != "" && !validLevels[level] {
		return fmt.Errorf("invalid log level: %q, valid options: DEBUG, INFO, WARN, ERROR", c.Level)
	}

	return nil
}

// globalCloser 保存全局 logger 的可关闭资源
var globalCloser io.Closer

// Init 初始化全局日志系统
//
// 这个函数应该在应用启动时调用一次，用于配置全局的 slog.Default() logger
// 如果输出到文件，应在程序退出时调用 Close() 关闭文件
func Init(cfg *Config) error {
	logger, closer, err := NewWithCloser(cfg)
	if err != nil {
		return err
	}
	// 关闭之前的 closer（忽略错误，因为我们正在替换它）
	if globalCloser != nil {
		_ = globalCloser.Close()
	}
	globalCloser = closer
	slog.SetDefault(logger)
	return nil
}

// Close 关闭全局 logger 的资源（如文件）
//
// 应在程序退出时调用，确保日志文件正确关闭
func Close() error {
	if globalCloser != nil {
		err := globalCloser.Close()
		globalCloser = nil
		return err
	}
	return nil
}

// New 创建新的 logger 实例
//
// 用于需要独立配置的场景，例如为特定模块创建专用 logger
// 注意：如果输出到文件，调用者需要使用 NewWithCloser 来获取 closer 并在适当时候关闭
func New(cfg *Config) (*slog.Logger, error) {
	logger, _, err := NewWithCloser(cfg)
	return logger, err
}

// NewWithCloser 创建新的 logger 实例并返回 closer
//
// 如果输出到文件，closer 不为 nil，应在不再使用时调用 closer.Close()
// 如果输出到 stdout/stderr，closer 为 nil
func NewWithCloser(cfg *Config) (*slog.Logger, io.Closer, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}

	writer, closer, err := getWriter(cfg.Output)
	if err != nil {
		return nil, nil, err
	}

	handler := createHandler(cfg, writer)
	return slog.New(handler), closer, nil
}

// createHandler 根据配置创建 slog.Handler
func createHandler(cfg *Config, writer io.Writer) slog.Handler {
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	switch cfg.Format {
	case "json":
		return newJSONHandler(writer, opts, cfg.TimeFormat, cfg.Timezone)
	case "color", "colored":
		colorConfig := &ColoredHandlerConfig{
			Level:        level,
			AddSource:    cfg.AddSource,
			EnableColor:  true,
			CallerClip:   "",
			PriorityKeys: []string{"time", "level", "msg"},
			TrailingKeys: []string{"source"},
			TimeFormat:   cfg.TimeFormat,
			Timezone:     cfg.Timezone,
		}
		return NewColoredHandler(writer, colorConfig)
	default: // text
		return slog.NewTextHandler(writer, opts)
	}
}

// parseLevel 解析日志级别字符串（大小写不敏感）
func parseLevel(levelStr string) slog.Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// getWriter 获取输出写入器
// 返回 writer 和 closer（如果是文件则 closer 不为 nil）
func getWriter(output string) (io.Writer, io.Closer, error) {
	switch output {
	case "stdout", "":
		return os.Stdout, nil, nil
	case "stderr":
		return os.Stderr, nil, nil
	default:
		// 文件路径
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, nil, err
		}
		return file, file, nil
	}
}

// WithAttrs 创建带有额外属性的 logger
//
// 用于为特定上下文添加固定的日志字段，例如：
//
//	logger := logger.WithAttrs("module", "worker", "node_id", nodeID)
func WithAttrs(attrs ...any) *slog.Logger {
	return slog.Default().With(attrs...)
}

// WithGroup 创建带有分组的 logger
//
// 用于将日志字段分组，使日志更有结构
func WithGroup(name string) *slog.Logger {
	return slog.Default().WithGroup(name)
}

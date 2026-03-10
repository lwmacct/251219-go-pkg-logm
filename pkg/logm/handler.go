package logm

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"time"

	writerpkg "github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

// Handler 统一的 slog.Handler 实现。
//
// 将格式化（Formatter）和输出（Writer）分离，
// 支持单输出 sink 和拦截器链。
type Handler struct {
	levelVar     *slog.LevelVar
	formatter    Formatter
	output       Writer
	interceptors []Interceptor
	addSource    bool
	timeFormat   string
	location     *time.Location

	// 继承的分组和属性
	groups []string
	attrs  []slog.Attr

	mu sync.Mutex
}

// HandlerConfig Handler 配置
type HandlerConfig struct {
	LevelVar     *slog.LevelVar
	Formatter    Formatter
	Output       Writer
	Interceptors []Interceptor
	AddSource    bool
	TimeFormat   string
	Location     *time.Location
}

// NewHandler 创建新的 Handler。
func NewHandler(cfg *HandlerConfig) *Handler {
	if cfg == nil {
		cfg = &HandlerConfig{}
	}

	h := &Handler{
		levelVar:     cfg.LevelVar,
		formatter:    cfg.Formatter,
		output:       cfg.Output,
		interceptors: cfg.Interceptors,
		addSource:    cfg.AddSource,
		timeFormat:   cfg.TimeFormat,
		location:     cfg.Location,
	}

	if h.levelVar == nil {
		h.levelVar = &slog.LevelVar{}
		h.levelVar.Set(slog.LevelInfo)
	}

	if h.location == nil {
		h.location = time.Local
	}

	if h.output == nil {
		h.output = writerpkg.Stdout()
	}

	return h
}

// Enabled 实现 slog.Handler 接口。
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.levelVar.Level()
}

// Handle 实现 slog.Handler 接口。
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// 转换为 Record
	rec := h.toRecord(r)

	// 应用拦截器
	for _, interceptor := range h.interceptors {
		rec = interceptor(ctx, rec)
		if rec == nil {
			return nil // 日志被过滤
		}
	}

	// 格式化
	if h.formatter == nil {
		return nil
	}

	data, err := h.formatter.Format(rec)
	if err != nil {
		return err
	}

	// 写入输出目标
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.output == nil {
		return nil
	}

	_, err = h.output.Write(data)
	return err
}

// WithAttrs 实现 slog.Handler 接口。
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	clone := h.clone()
	clone.attrs = append(clone.attrs, attrs...)
	return clone
}

// WithGroup 实现 slog.Handler 接口。
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	clone := h.clone()
	clone.groups = append(clone.groups, name)
	return clone
}

// clone 创建 Handler 的浅拷贝
func (h *Handler) clone() *Handler {
	return &Handler{
		levelVar:     h.levelVar,
		formatter:    h.formatter,
		output:       h.output,
		interceptors: h.interceptors,
		addSource:    h.addSource,
		timeFormat:   h.timeFormat,
		location:     h.location,
		groups:       append([]string{}, h.groups...),
		attrs:        append([]slog.Attr{}, h.attrs...),
	}
}

// toRecord 将 slog.Record 转换为 Record
func (h *Handler) toRecord(r slog.Record) *Record {
	rec := &Record{
		Time:    r.Time.In(h.location),
		Level:   r.Level,
		Message: r.Message,
		Groups:  h.groups,
	}

	// 添加继承的属性
	rec.Attrs = append(rec.Attrs, h.attrs...)

	// 添加当前记录的属性
	r.Attrs(func(a slog.Attr) bool {
		rec.Attrs = append(rec.Attrs, a)
		return true
	})

	// 提取源代码位置
	if h.addSource && r.PC != 0 {
		rec.Source = h.source(r.PC)
	}

	return rec
}

// source 从 PC 获取源代码位置
func (h *Handler) source(pc uintptr) *slog.Source {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	return &slog.Source{
		Function: f.Function,
		File:     f.File,
		Line:     f.Line,
	}
}

// Close 关闭输出 Writer
func (h *Handler) Close() error {
	if h.output != nil {
		return h.output.Close()
	}
	return nil
}

// Sync 刷新输出 Writer 缓冲区
func (h *Handler) Sync() error {
	if h.output != nil {
		return h.output.Sync()
	}
	return nil
}

// SetLevel 动态设置日志级别
func (h *Handler) SetLevel(level slog.Level) {
	h.levelVar.Set(level)
}

// Level 获取当前日志级别
func (h *Handler) Level() slog.Level {
	return h.levelVar.Level()
}

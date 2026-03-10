package logm

import (
	"log/slog"
	"sync"
)

var (
	globalRuntime *runtimeState
	// globalMu 保护全局状态
	globalMu sync.RWMutex
)

type managedHandler interface {
	Close() error
	Sync() error
}

type builtHandler struct {
	handler slog.Handler
	managed []managedHandler
}

type runtimeState struct {
	managed         []managedHandler
	previousDefault *slog.Logger
}

// Init 初始化全局日志系统。
//
// 使用 Config 配置：
//
//	logm.Init(Config{
//	    Level:     "DEBUG",
//	    Formatter: formatter.ColorText(),
//	    Output:    writer.Stdout(),
//	})
//
// 也可使用预设配置：
//
//	logm.Init(logm.PresetDev())
//	logm.Init(logm.PresetProd())
func Init(configs ...Config) error {
	built := buildHandlerWithLevelVar(globalLevelVar, firstConfig(configs...))
	logger := slog.New(built.handler)

	globalMu.Lock()
	previousDefault := slog.Default()
	if globalRuntime != nil && globalRuntime.previousDefault != nil {
		previousDefault = globalRuntime.previousDefault
	}
	oldRuntime := globalRuntime
	globalRuntime = &runtimeState{
		managed:         built.managed,
		previousDefault: previousDefault,
	}
	slog.SetDefault(logger)
	globalMu.Unlock()

	if oldRuntime != nil {
		_ = closeManagedHandlers(oldRuntime.managed)
	}

	return nil
}

// MustInit 初始化全局日志系统，失败时 panic。
//
// 适用于程序启动阶段，日志系统初始化失败通常意味着程序无法正常运行：
//
//	func main() {
//	    logm.MustInit(logm.PresetDev())
//	    defer logm.Close()
//	    // ...
//	}
func MustInit(configs ...Config) {
	if err := Init(configs...); err != nil {
		panic("logm: init failed: " + err.Error())
	}
}

// New 创建独立的 logger 实例。
//
// 返回的 logger 独立于全局配置，适用于模块专用日志。
func New(configs ...Config) *slog.Logger {
	built := buildHandlerWithLevelVar(nil, firstConfig(configs...))

	return slog.New(built.handler)
}

func firstConfig(configs ...Config) Config {
	if len(configs) == 0 {
		return Config{}
	}
	return configs[0]
}

// Close 关闭全局日志系统，释放资源。
func Close() error {
	globalMu.Lock()
	runtime := globalRuntime
	globalRuntime = nil
	if runtime != nil && runtime.previousDefault != nil {
		slog.SetDefault(runtime.previousDefault)
	}
	globalMu.Unlock()

	if runtime != nil {
		return closeManagedHandlers(runtime.managed)
	}
	return nil
}

// Sync 刷新全局日志缓冲区。
func Sync() error {
	globalMu.RLock()
	var handlers []managedHandler
	if globalRuntime != nil {
		handlers = append([]managedHandler(nil), globalRuntime.managed...)
	}
	globalMu.RUnlock()

	if len(handlers) > 0 {
		return syncManagedHandlers(handlers)
	}
	return nil
}

func closeManagedHandlers(handlers []managedHandler) error {
	var firstErr error
	for _, h := range handlers {
		if err := h.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func syncManagedHandlers(handlers []managedHandler) error {
	var firstErr error
	for _, h := range handlers {
		if err := h.Sync(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Default 返回全局默认 logger。
func Default() *slog.Logger {
	return slog.Default()
}

// 便捷日志函数

// Debug 记录调试级别日志。
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Info 记录信息级别日志。
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Warn 记录警告级别日志。
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error 记录错误级别日志。
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// With 返回带有额外属性的 logger。
func With(args ...any) *slog.Logger {
	return slog.Default().With(args...)
}

// WithGroup 返回带有分组的 logger。
func WithGroup(name string) *slog.Logger {
	return slog.Default().WithGroup(name)
}

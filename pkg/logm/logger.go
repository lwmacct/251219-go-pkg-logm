package logm

import (
	"log/slog"
	"sync"
)

// Logger owns the resources configured for its handlers. Embed *slog.Logger
// so existing slog call sites keep the full standard API while Close and Sync
// provide deterministic lifecycle management.
type Logger struct {
	*slog.Logger
	managed   []managed
	syncers   []syncer
	closeOnce sync.Once
	closeErr  error
}

type runtimeState struct {
	logger          *Logger
	previousDefault *slog.Logger
}

var (
	globalMu      sync.RWMutex
	globalRuntime *runtimeState
)

// New creates an independent logger. The caller owns it and should call Close
// when it is no longer needed.
func New(configs ...Config) (*Logger, error) {
	built, err := buildLogger(firstConfig(configs...), nil)
	if err != nil {
		return nil, err
	}
	return &Logger{Logger: slog.New(built.handler), managed: built.managed, syncers: built.syncers}, nil
}

// Init installs a process-wide default logger and closes the previous logger
// after the new one is visible. If building fails, the existing logger remains
// untouched.
func Init(configs ...Config) error {
	cfg := firstConfig(configs...)
	// Concrete preset levels initialize the process-wide LevelVar so SetLevel
	// continues to affect the installed logger. Callers that provide their own
	// Leveler retain full control over its dynamic behavior.
	if level, ok := cfg.Level.(slog.Level); ok {
		globalLevelVar.Set(level)
		cfg.Level = globalLevelVar
	}
	built, err := buildLogger(cfg, globalLevelVar)
	if err != nil {
		return err
	}
	logger := &Logger{Logger: slog.New(built.handler), managed: built.managed, syncers: built.syncers}

	globalMu.Lock()
	previousDefault := slog.Default()
	if globalRuntime != nil && globalRuntime.previousDefault != nil {
		previousDefault = globalRuntime.previousDefault
	}
	old := globalRuntime
	globalRuntime = &runtimeState{logger: logger, previousDefault: previousDefault}
	slog.SetDefault(logger.Logger)
	globalMu.Unlock()

	if old != nil && old.logger != nil {
		return old.logger.Close()
	}
	return nil
}

// MustInit panics when Init cannot validate or construct the logger.
func MustInit(configs ...Config) {
	if err := Init(configs...); err != nil {
		panic("logm: init failed: " + err.Error())
	}
}

// Close closes the global logger and restores the slog default that was active
// before the first Init call. It is safe to call repeatedly.
func Close() error {
	globalMu.Lock()
	state := globalRuntime
	globalRuntime = nil
	if state != nil && state.previousDefault != nil {
		slog.SetDefault(state.previousDefault)
	}
	globalMu.Unlock()
	if state == nil || state.logger == nil {
		return nil
	}
	return state.logger.Close()
}

// Sync flushes all global logger sinks without closing them.
func Sync() error {
	globalMu.RLock()
	state := globalRuntime
	globalMu.RUnlock()
	if state == nil || state.logger == nil {
		return nil
	}
	return state.logger.Sync()
}

// Close releases resources owned by this logger. The result is stable across
// repeated calls and includes all close errors.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.closeOnce.Do(func() { l.closeErr = closeManaged(l.managed) })
	return l.closeErr
}

// Sync flushes all sinks owned by this logger.
func (l *Logger) Sync() error {
	if l == nil {
		return nil
	}
	return syncAll(l.syncers)
}

// Default returns the standard global logger. Use Logger methods when you need
// to close an independently-created instance.
func Default() *slog.Logger { return slog.Default() }

func firstConfig(configs ...Config) Config {
	if len(configs) == 0 {
		return Config{}
	}
	return configs[0]
}

// convenience functions intentionally delegate to standard slog.Default.
func Debug(msg string, args ...any)      { slog.Debug(msg, args...) }
func Info(msg string, args ...any)       { slog.Info(msg, args...) }
func Warn(msg string, args ...any)       { slog.Warn(msg, args...) }
func Error(msg string, args ...any)      { slog.Error(msg, args...) }
func With(args ...any) *slog.Logger      { return slog.Default().With(args...) }
func WithGroup(name string) *slog.Logger { return slog.Default().WithGroup(name) }

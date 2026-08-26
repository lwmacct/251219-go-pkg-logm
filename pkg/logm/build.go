package logm

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
)

type builtLogger struct {
	handler slog.Handler
	managed []managed
	syncers []syncer
}

func buildLogger(cfg Config, fallback *slog.LevelVar) (*builtLogger, error) {
	cfg, leveler, err := normalizeConfig(cfg, fallback)
	if err != nil {
		return nil, err
	}
	replace := buildReplaceAttr(cfg)
	var handlers []slog.Handler
	var managedResources []managed
	var syncers []syncer

	for _, output := range cfg.Outputs {
		if output.Format == "" {
			output.Format = FormatText
		}
		target, owned, outputSyncers, err := makeOutput(output)
		if err != nil {
			_ = closeManaged(managedResources)
			return nil, err
		}
		managedResources = append(managedResources, owned...)
		syncers = append(syncers, outputSyncers...)
		handler, err := makeSlogHandler(output.Format, target, &slog.HandlerOptions{
			Level:       leveler,
			AddSource:   cfg.AddSource,
			ReplaceAttr: replace,
		})
		if err != nil {
			_ = closeManaged(managedResources)
			return nil, err
		}
		handlers = append(handlers, handler)
	}
	handlers = append(handlers, cfg.Handlers...)
	if len(handlers) == 0 {
		return nil, fmt.Errorf("logm: no handlers configured")
	}
	var handler slog.Handler
	if len(handlers) == 1 {
		handler = handlers[0]
	} else {
		handler = slog.NewMultiHandler(handlers...)
	}
	if len(cfg.Middleware) != 0 || cfg.OnError != nil {
		handler = &middlewareHandler{next: handler, middleware: cfg.Middleware, onError: cfg.OnError}
	}
	for _, h := range cfg.Handlers {
		if m, ok := h.(managed); ok {
			managedResources = append(managedResources, m)
		}
		if s, ok := h.(syncer); ok {
			syncers = append(syncers, s)
		}
	}
	return &builtLogger{handler: handler, managed: managedResources, syncers: syncers}, nil
}

func makeSlogHandler(format Format, target io.Writer, options *slog.HandlerOptions) (slog.Handler, error) {
	switch format {
	case FormatText:
		return slog.NewTextHandler(target, options), nil
	case FormatJSON:
		return slog.NewJSONHandler(target, options), nil
	case FormatPretty:
		return slog.NewTextHandler(&prettyWriter{dst: target}, options), nil
	default:
		return nil, fmt.Errorf("logm: unsupported output format %q", format)
	}
}

func buildReplaceAttr(cfg Config) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, attr slog.Attr) slog.Attr {
		attr.Value = attr.Value.Resolve()
		if cfg.ReplaceAttr != nil {
			attr = cfg.ReplaceAttr(groups, attr)
			if attr.Key == "" {
				return attr
			}
			attr.Value = attr.Value.Resolve()
		}
		switch attr.Key {
		case "time":
			if attr.Value.Kind() == slog.KindTime {
				t := attr.Value.Time()
				if cfg.Location != nil {
					t = t.In(cfg.Location)
				}
				return slog.String(attr.Key, formatTime(t, cfg.TimeFormat))
			}
		case "file":
			if attr.Value.Kind() == slog.KindString {
				return slog.String(attr.Key, clipSourcePath(attr.Value.String(), cfg.SourceClip, cfg.SourceDepth))
			}
		}
		return attr
	}
}

func clipSourcePath(path, prefix string, depth int) string {
	path = filepath.ToSlash(path)
	prefix = filepath.ToSlash(prefix)
	if prefix != "" {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			path = strings.TrimPrefix(path, string(filepath.Separator))
			if strings.HasSuffix(prefix, "/") {
				if slash := strings.IndexByte(path, '/'); slash >= 0 {
					path = path[slash+1:]
				}
			}
		}
	}
	if strings.Contains(path, "/workspace/") {
		parts := strings.SplitN(path, "/workspace/", 2)
		path = parts[1]
		if slash := strings.IndexByte(path, '/'); slash >= 0 {
			path = path[slash+1:]
		}
	}
	if depth <= 0 || !strings.HasPrefix(path, "/") {
		return path
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) <= depth {
		return path
	}
	return strings.Join(parts[len(parts)-depth:], "/")
}

// prettyWriter colors only the level token of standard TextHandler output.
// It deliberately remains a writer wrapper, so all serialization still comes
// from the standard library and remains race-safe and parseable when color is
// disabled by selecting FormatText or FormatJSON.
type prettyWriter struct {
	mu  sync.Mutex
	dst io.Writer
}

func (w *prettyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	line := string(p)
	for _, level := range []string{"DEBUG", "INFO", "WARN", "ERROR"} {
		needle := "level=" + level
		if strings.Contains(line, needle) {
			line = strings.Replace(line, needle, "level="+levelColor(level)+level+ansiReset, 1)
			break
		}
	}
	n, err := io.WriteString(w.dst, line)
	if err != nil {
		return 0, err
	}
	if n != len(line) {
		return 0, io.ErrShortWrite
	}
	return len(p), nil
}

const ansiReset = "\x1b[0m"

func levelColor(level string) string {
	switch level {
	case "DEBUG":
		return "\x1b[36m"
	case "INFO":
		return "\x1b[32m"
	case "WARN":
		return "\x1b[33m"
	case "ERROR":
		return "\x1b[31m"
	default:
		return ""
	}
}

func closeManaged(resources []managed) error {
	var errs []error
	for i := len(resources) - 1; i >= 0; i-- {
		if err := resources[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return joinErrors(errs)
}

func syncAll(syncers []syncer) error {
	var errs []error
	for _, s := range syncers {
		if err := s.Sync(); err != nil {
			errs = append(errs, err)
		}
	}
	return joinErrors(errs)
}

func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return errorsJoin(errs...)
}

// Keep the errors.Join call in one small helper to make the error policy easy
// to audit and to avoid exposing implementation details in the rest of build.
func errorsJoin(errs ...error) error {
	var nonNil []error
	for _, err := range errs {
		if err != nil {
			nonNil = append(nonNil, err)
		}
	}
	if len(nonNil) == 0 {
		return nil
	}
	return errors.Join(nonNil...)
}

package logm

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"strings"
	"time"

	writerpkg "github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

// Handler 仅负责包装标准 slog handler，并托管底层输出资源。
type Handler struct {
	next    slog.Handler
	managed []managedHandler
}

// HandlerConfig Handler 配置。
type HandlerConfig struct {
	LevelVar    *slog.LevelVar
	Format      Format
	Output      Writer
	AddSource   bool
	TimeFormat  string
	Location    *time.Location
	Color       bool
	ExpandJSON  bool
	ReplaceAttr func(groups []string, attr slog.Attr) slog.Attr
}

// NewHandler 创建新的 Handler。
func NewHandler(cfg *HandlerConfig) *Handler {
	if cfg == nil {
		cfg = &HandlerConfig{}
	}

	levelVar := cfg.LevelVar
	if levelVar == nil {
		levelVar = &slog.LevelVar{}
		levelVar.Set(slog.LevelInfo)
	}

	location := cfg.Location
	if location == nil {
		location = time.Local
	}

	output := cfg.Output
	if output == nil {
		output = writerpkg.Stdout()
	}

	baseWriter := output
	if cfg.Color {
		baseWriter = &colorWriter{next: output}
	}

	options := &slog.HandlerOptions{
		AddSource:   cfg.AddSource,
		Level:       levelVar,
		ReplaceAttr: buildReplaceAttr(location, cfg.TimeFormat, cfg.ExpandJSON, cfg.ReplaceAttr),
	}

	var next slog.Handler
	switch normalizeFormat(cfg.Format) {
	case FormatJSON:
		next = slog.NewJSONHandler(baseWriter, options)
	default:
		next = slog.NewTextHandler(baseWriter, options)
	}

	return &Handler{
		next:    next,
		managed: []managedHandler{baseWriter},
	}
}

// Enabled 实现 slog.Handler 接口。
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle 实现 slog.Handler 接口。
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	return h.next.Handle(ctx, r)
}

// WithAttrs 实现 slog.Handler 接口。
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return &Handler{
		next:    h.next.WithAttrs(attrs),
		managed: h.managed,
	}
}

// WithGroup 实现 slog.Handler 接口。
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &Handler{
		next:    h.next.WithGroup(name),
		managed: h.managed,
	}
}

// Close 关闭输出 Writer。
func (h *Handler) Close() error {
	return closeManagedHandlers(h.managed)
}

// Sync 刷新输出 Writer 缓冲区。
func (h *Handler) Sync() error {
	return syncManagedHandlers(h.managed)
}

func buildReplaceAttr(
	location *time.Location,
	timeFormat string,
	expandJSON bool,
	userReplace func(groups []string, attr slog.Attr) slog.Attr,
) func(groups []string, attr slog.Attr) slog.Attr {
	return func(groups []string, attr slog.Attr) slog.Attr {
		switch attr.Key {
		case slog.TimeKey:
			attr = slog.String(attr.Key, formatTimeValue(resolveTimeAttr(attr, location), timeFormat))
		case slog.SourceKey:
			if src, ok := attr.Value.Any().(*slog.Source); ok && src != nil {
				clipped := *src
				clipped.File = clipWorkspacePath(clipped.File)
				attr = slog.Any(attr.Key, &clipped)
			}
		default:
			if err, ok := attr.Value.Any().(error); ok && err != nil {
				attr = slog.String(attr.Key, err.Error())
			}
			if expandJSON && attr.Value.Kind() == slog.KindString {
				if expanded, ok := expandJSONStringAttr(attr.Key, attr.Value.String()); ok {
					attr = expanded
				}
			}
		}

		if userReplace != nil {
			attr = userReplace(groups, attr)
		}
		return attr
	}
}

func resolveTimeAttr(attr slog.Attr, location *time.Location) time.Time {
	t := attr.Value.Time()
	if location != nil {
		return t.In(location)
	}
	return t
}

func formatTimeValue(t time.Time, format string) string {
	switch format {
	case "time":
		return t.Format("15:04:05")
	case "timems":
		return t.Format("15:04:05.000")
	case "datetime":
		return t.Format("2006-01-02 15:04:05")
	case "rfc3339":
		return t.Format(time.RFC3339)
	case "rfc3339ms":
		return t.Format("2006-01-02T15:04:05.000Z07:00")
	default:
		if format == "" {
			return t.Format("2006-01-02 15:04:05")
		}
		return t.Format(format)
	}
}

func expandJSONStringAttr(key, raw string) (slog.Attr, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return slog.Attr{}, false
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return slog.Attr{}, false
	}

	return slog.Group(key, mapToAttrs(payload)...), true
}

func mapToAttrs(payload map[string]any) []any {
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	attrs := make([]any, 0, len(keys))
	for _, key := range keys {
		attrs = append(attrs, jsonValueToAttr(key, payload[key]))
	}
	return attrs
}

func jsonValueToAttr(key string, value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return slog.Group(key, mapToAttrs(typed)...)
	case string:
		return slog.String(key, typed)
	case bool:
		return slog.Bool(key, typed)
	case float64:
		if typed == float64(int64(typed)) {
			return slog.Int64(key, int64(typed))
		}
		return slog.Float64(key, typed)
	case nil:
		return slog.Any(key, nil)
	default:
		return slog.Any(key, typed)
	}
}

type colorWriter struct {
	next Writer
}

func (w *colorWriter) Write(p []byte) (int, error) {
	return w.next.Write(colorizeLine(p))
}

func (w *colorWriter) Close() error {
	return w.next.Close()
}

func (w *colorWriter) Sync() error {
	return w.next.Sync()
}

func colorizeLine(p []byte) []byte {
	line := bytes.TrimSuffix(p, []byte{'\n'})
	color := detectLevelColor(line)
	if color == "" {
		return append([]byte(nil), p...)
	}

	var out bytes.Buffer
	out.WriteString(color)
	out.Write(line)
	out.WriteString("\x1b[0m")
	if len(line) != len(p) {
		out.WriteByte('\n')
	}
	return out.Bytes()
}

func detectLevelColor(line []byte) string {
	text := string(line)
	switch {
	case strings.Contains(text, ` level=DEBUG`) || strings.Contains(text, `"level":"DEBUG"`):
		return "\x1b[36m"
	case strings.Contains(text, ` level=INFO`) || strings.Contains(text, `"level":"INFO"`):
		return "\x1b[32m"
	case strings.Contains(text, ` level=WARN`) || strings.Contains(text, `"level":"WARN"`):
		return "\x1b[33m"
	case strings.Contains(text, ` level=ERROR`) || strings.Contains(text, `"level":"ERROR"`):
		return "\x1b[31m"
	default:
		return ""
	}
}

var _ slog.Handler = (*Handler)(nil)

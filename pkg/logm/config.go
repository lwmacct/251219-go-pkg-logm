package logm

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

// Config configures a logger. Outputs and Handlers are combined with
// slog.NewMultiHandler; when both are empty a text handler writing to stdout
// is installed.
type Config struct {
	Level       slog.Leveler
	Outputs     []Output
	Handlers    []slog.Handler
	AddSource   bool
	TimeFormat  string
	Location    *time.Location
	SourceClip  string
	SourceDepth int
	ReplaceAttr func([]string, slog.Attr) slog.Attr
	Middleware  []Middleware
	OnError     func(error)
}

// PresetDefault returns a conservative text configuration suitable for local
// command-line programs.
func PresetDefault() Config {
	return Config{
		Level:       slog.LevelInfo,
		TimeFormat:  "datetime",
		Location:    time.Local,
		SourceDepth: 3,
	}
}

// ParseOutput parses stdout, stderr, or a comma-separated list of file paths.
// File outputs are owned by the resulting configuration and are closed by
// Logger.Close. Empty entries are ignored; an entirely empty value is an
// error, which prevents silently losing logs due to a typo.
func ParseOutput(raw string) ([]Output, error) {
	var outputs []Output
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		switch strings.ToLower(name) {
		case "stdout":
			outputs = append(outputs, Output{Name: name, Writer: writer.Stdout(), Format: FormatText})
		case "stderr":
			outputs = append(outputs, Output{Name: name, Writer: writer.Stderr(), Format: FormatText})
		default:
			outputs = append(outputs, Output{Name: name, Writer: writer.File(name), Format: FormatJSON, Own: true})
		}
	}
	if len(outputs) == 0 {
		return nil, errors.New("logm: output is empty")
	}
	return outputs, nil
}

func defaultOutput() Output {
	return Output{Name: "stdout", Writer: writer.Stdout(), Format: FormatText}
}

func normalizeConfig(cfg Config, fallback *slog.LevelVar) (Config, slog.Leveler, error) {
	defaults := PresetDefault()
	if cfg.Level == nil {
		if fallback != nil {
			cfg.Level = fallback
		} else {
			cfg.Level = defaults.Level
		}
	}
	leveler := resolveLeveler(cfg.Level, fallback)
	if cfg.Location == nil {
		cfg.Location = defaults.Location
	}
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = defaults.TimeFormat
	}
	if cfg.SourceDepth == 0 {
		cfg.SourceDepth = defaults.SourceDepth
	}
	if cfg.SourceDepth < 0 {
		return Config{}, nil, errors.New("logm: source depth cannot be negative")
	}
	if len(cfg.Outputs) == 0 && len(cfg.Handlers) == 0 {
		cfg.Outputs = []Output{defaultOutput()}
	}
	cfg.Outputs = append([]Output(nil), cfg.Outputs...)
	cfg.Handlers = append([]slog.Handler(nil), cfg.Handlers...)
	cfg.Middleware = append([]Middleware(nil), cfg.Middleware...)
	return cfg, leveler, nil
}

func resolveLeveler(leveler slog.Leveler, fallback *slog.LevelVar) slog.Leveler {
	if leveler != nil {
		return leveler
	}
	if fallback != nil {
		return fallback
	}
	v := new(slog.LevelVar)
	v.Set(slog.LevelInfo)
	return v
}

func makeOutput(o Output) (io.Writer, []managed, []syncer, error) {
	if o.Async.Overflow > writer.OverflowFail {
		return nil, nil, nil, fmt.Errorf("logm: invalid async overflow policy %d", o.Async.Overflow)
	}
	target := o.Writer
	if target == nil {
		switch o.Name {
		case "", "stdout":
			target = writer.Stdout()
		case "stderr":
			target = writer.Stderr()
		default:
			target = writer.File(o.Name)
			o.Own = true
		}
	}
	var owned []managed
	var syncers []syncer
	if o.Async.Capacity != 0 || o.Async.Overflow != writer.OverflowBlock {
		aw := writer.NewAsync(target, o.Async)
		target = aw
		owned = append(owned, aw)
		syncers = append(syncers, aw)
		return target, owned, syncers, nil
	}
	if s, ok := target.(syncer); ok {
		syncers = append(syncers, s)
	}
	if o.Own {
		if c, ok := target.(managed); ok {
			owned = append(owned, c)
		} else {
			return nil, nil, nil, fmt.Errorf("logm: owned output %q does not implement io.Closer", o.Name)
		}
	}
	return target, owned, syncers, nil
}

func formatTime(t time.Time, format string) string {
	switch format {
	case "", "datetime":
		return t.Format("2006-01-02 15:04:05")
	case "time":
		return t.Format("15:04:05")
	case "timems":
		return t.Format("15:04:05.000")
	case "rfc3339":
		return t.Format(time.RFC3339)
	case "rfc3339ms":
		return t.Format("2006-01-02T15:04:05.000Z07:00")
	default:
		return t.Format(format)
	}
}

func timezone(name string) (*time.Location, error) {
	if name == "" {
		return time.Local, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("logm: invalid timezone %q: %w", name, err)
	}
	return loc, nil
}

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q", raw)
	}
}

// LoadConfigFromEnv loads LOGM_* settings and validates every supplied value.
// Supported variables are LOGM_LEVEL, LOGM_FORMAT, LOGM_OUTPUT,
// LOGM_SOURCE, LOGM_TIME_FORMAT, LOGM_TIMEZONE, and LOGM_ENV.
func LoadConfigFromEnv() (Config, error) {
	cfg := PresetProd()
	var formatOverride Format
	if raw := strings.TrimSpace(os.Getenv("LOGM_ENV")); raw != "" {
		switch strings.ToLower(raw) {
		case "dev", "development":
			cfg = PresetDev()
		case "prod", "production":
			cfg = PresetProd()
		default:
			return Config{}, fmt.Errorf("logm: invalid environment %q", raw)
		}
	}
	if raw := os.Getenv("LOGM_LEVEL"); raw != "" {
		level, err := ParseLevel(raw)
		if err != nil {
			return Config{}, err
		}
		cfg.Level = level
	}
	if raw := os.Getenv("LOGM_FORMAT"); raw != "" {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "text":
			formatOverride = FormatText
		case "json":
			formatOverride = FormatJSON
		case "pretty", "color_text", "color-text":
			formatOverride = FormatPretty
		default:
			return Config{}, fmt.Errorf("logm: invalid format %q", raw)
		}
	}
	if raw := os.Getenv("LOGM_OUTPUT"); raw != "" {
		outputs, err := ParseOutput(raw)
		if err != nil {
			return Config{}, err
		}
		cfg.Outputs = outputs
	}
	if raw := os.Getenv("LOGM_SOURCE"); raw != "" {
		value, err := parseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("logm: LOGM_SOURCE: %w", err)
		}
		cfg.AddSource = value
	}
	if raw := os.Getenv("LOGM_TIME_FORMAT"); raw != "" {
		cfg.TimeFormat = raw
	}
	if raw := os.Getenv("LOGM_TIMEZONE"); raw != "" {
		loc, err := timezone(raw)
		if err != nil {
			return Config{}, err
		}
		cfg.Location = loc
	}
	if formatOverride != "" {
		setOutputFormats(&cfg, formatOverride)
	}
	return cfg, nil
}

func setOutputFormats(cfg *Config, format Format) {
	for i := range cfg.Outputs {
		cfg.Outputs[i].Format = format
	}
}

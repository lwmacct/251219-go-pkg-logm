package logm

import (
	"log/slog"
	"time"

	formatterpkg "github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter"
	writerpkg "github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

type resolvedOptions struct {
	levelVar     *slog.LevelVar
	formatter    Formatter
	output       Writer
	slogHandlers []slog.Handler
	interceptors []Interceptor
	addSource    bool
	timeFormat   string
	location     *time.Location
}

func buildHandlerWithLevelVar(fallbackLevelVar *slog.LevelVar, cfg Config) *builtHandler {
	return buildHandler(normalizeConfig(cfg, fallbackLevelVar))
}

func normalizeConfig(cfg Config, fallbackLevelVar *slog.LevelVar) *resolvedOptions {
	defaults := PresetDefault()

	if cfg.TimeFormat == "" {
		cfg.TimeFormat = defaults.TimeFormat
	}
	if cfg.Timezone == "" && cfg.Location == nil {
		cfg.Timezone = defaults.Timezone
	}
	if cfg.Level == "" && cfg.LevelVar == nil {
		cfg.Level = defaults.Level
	}

	location := resolveLocation(cfg.Location, cfg.Timezone)
	levelVar := resolveLevelVar(cfg.LevelVar, fallbackLevelVar)
	if cfg.Level != "" || cfg.LevelVar == nil {
		levelVar.Set(ParseLevel(cfg.Level))
	}

	return &resolvedOptions{
		levelVar:     levelVar,
		formatter:    resolveFormatter(cfg.Formatter, cfg.TimeFormat, cfg.Timezone),
		output:       resolveOutput(cfg.Output),
		slogHandlers: append([]slog.Handler(nil), cfg.SlogHandlers...),
		interceptors: append([]Interceptor(nil), cfg.Interceptors...),
		addSource:    cfg.AddSource,
		timeFormat:   cfg.TimeFormat,
		location:     location,
	}
}

func resolveLevelVar(requested *slog.LevelVar, fallback *slog.LevelVar) *slog.LevelVar {
	switch {
	case requested != nil:
		return requested
	case fallback != nil:
		return fallback
	default:
		return &slog.LevelVar{}
	}
}

func resolveLocation(location *time.Location, timezone string) *time.Location {
	if location != nil {
		return location
	}
	if resolved := loadTimezone(timezone); resolved != nil {
		return resolved
	}
	return time.Local
}

func resolveFormatter(current Formatter, timeFormat, timezone string) Formatter {
	if current != nil {
		return current
	}
	return formatterpkg.Text(
		formatterpkg.WithTimeFormat(timeFormat),
		formatterpkg.WithTimezone(timezone),
	)
}

func resolveOutput(output Writer) Writer {
	if output == nil {
		return writerpkg.Stdout()
	}
	return output
}

func loadTimezone(tz string) *time.Location {
	if tz == "" {
		return nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil
	}
	return loc
}

func buildHandler(resolved *resolvedOptions) *builtHandler {
	base := NewHandler(&HandlerConfig{
		LevelVar:     resolved.levelVar,
		Formatter:    resolved.formatter,
		Output:       resolved.output,
		Interceptors: resolved.interceptors,
		AddSource:    resolved.addSource,
		TimeFormat:   resolved.timeFormat,
		Location:     resolved.location,
	})

	handlers := make([]slog.Handler, 0, 1+len(resolved.slogHandlers))
	handlers = append(handlers, base)

	managed := []managedHandler{base}
	for _, h := range resolved.slogHandlers {
		handlers = append(handlers, h)
		if mh, ok := h.(managedHandler); ok {
			managed = append(managed, mh)
		}
	}

	combined := slog.Handler(base)
	if len(handlers) > 1 {
		combined = slog.NewMultiHandler(handlers...)
	}

	return &builtHandler{
		handler: combined,
		managed: managed,
	}
}

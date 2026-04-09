package logm

import (
	"log/slog"
	"time"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

type resolvedOptions struct {
	levelVar     *slog.LevelVar
	format       Format
	output       Writer
	slogHandlers []slog.Handler
	addSource    bool
	timeFormat   string
	location     *time.Location
	color        bool
	expandJSON   bool
	replaceAttr  func(groups []string, attr slog.Attr) slog.Attr
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
		format:       normalizeFormat(cfg.Format),
		output:       resolveOutput(cfg.Output),
		slogHandlers: append([]slog.Handler(nil), cfg.SlogHandlers...),
		addSource:    cfg.AddSource,
		timeFormat:   cfg.TimeFormat,
		location:     location,
		color:        cfg.Color,
		expandJSON:   cfg.ExpandJSON,
		replaceAttr:  cfg.ReplaceAttr,
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

func resolveOutput(output Writer) Writer {
	if output == nil {
		return writer.Stdout()
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
		LevelVar:    resolved.levelVar,
		Format:      resolved.format,
		Output:      resolved.output,
		AddSource:   resolved.addSource,
		TimeFormat:  resolved.timeFormat,
		Location:    resolved.location,
		Color:       resolved.color,
		ExpandJSON:  resolved.expandJSON,
		ReplaceAttr: resolved.replaceAttr,
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

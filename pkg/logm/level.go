package logm

import (
	"fmt"
	"log/slog"
	"strings"
)

var globalLevelVar = new(slog.LevelVar)

func init() { globalLevelVar.Set(slog.LevelInfo) }

// ParseLevel strictly parses a level name. Unknown values are errors rather
// than silently becoming INFO, because an unnoticed production typo is unsafe.
func ParseLevel(raw string) (slog.Level, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN", "WARNING":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("logm: invalid level %q", raw)
	}
}

// SetLevel changes the process-wide minimum level and returns an error for an
// invalid name.
func SetLevel(raw string) error {
	level, err := ParseLevel(raw)
	if err != nil {
		return err
	}
	globalLevelVar.Set(level)
	return nil
}

// SetLevelValue is the typed variant for callers that already have a slog
// level.
func SetLevelValue(level slog.Level) { globalLevelVar.Set(level) }

func GetLevel() slog.Level { return globalLevelVar.Level() }

func GetLevelVar() *slog.LevelVar { return globalLevelVar }

func LevelString(level slog.Level) string { return level.String() }

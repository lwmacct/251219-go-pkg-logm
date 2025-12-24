package formatter

import "log/slog"

// LevelInfo 级别信息
type LevelInfo struct {
	Name  string
	Color string
}

// DefaultLevelInfo 返回级别的默认信息
func DefaultLevelInfo(level slog.Level) LevelInfo {
	switch {
	case level < slog.LevelInfo:
		return LevelInfo{Name: "DEBUG", Color: ColorCyan}
	case level < slog.LevelWarn:
		return LevelInfo{Name: "INFO", Color: ColorGreen}
	case level < slog.LevelError:
		return LevelInfo{Name: "WARN", Color: ColorYellow}
	default:
		return LevelInfo{Name: "ERROR", Color: ColorRed}
	}
}

// LevelName 返回级别名称
func LevelName(level slog.Level) string {
	return DefaultLevelInfo(level).Name
}

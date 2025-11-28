package logger

import (
	"io"
	"log/slog"
	"time"
)

// newTextHandler 创建自定义 Text handler，支持灵活的时间格式
func newTextHandler(w io.Writer, opts *slog.HandlerOptions, timeFormat string, timezone string) *slog.TextHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	if timeFormat == "" {
		timeFormat = "datetime"
	}

	// 加载时区，默认使用上海时区 (UTC+8)
	loc := loadTimezone(timezone)

	// 使用 ReplaceAttr 来自定义时间格式
	originalReplace := opts.ReplaceAttr
	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		// 先执行原有的 ReplaceAttr（如果有）
		if originalReplace != nil {
			a = originalReplace(groups, a)
		}

		// 只处理顶级的 time 字段
		if len(groups) == 0 && a.Key == slog.TimeKey {
			if t, ok := a.Value.Any().(time.Time); ok {
				// 转换到指定时区
				if loc != nil {
					t = t.In(loc)
				}
				// 格式化时间
				formatted := formatTimeString(t, timeFormat)
				return slog.String(slog.TimeKey, formatted)
			}
		}
		return a
	}

	return slog.NewTextHandler(w, opts)
}

// formatTimeString 根据配置格式化时间为字符串
func formatTimeString(t time.Time, timeFormat string) string {
	switch timeFormat {
	case "rfc3339":
		return t.Format(time.RFC3339)
	case "rfc3339ms":
		return t.Format("2006-01-02T15:04:05.000Z07:00")
	case "time":
		return t.Format("15:04:05")
	case "timems":
		return t.Format("15:04:05.000")
	case "datetime", "":
		// 默认格式：日期时间（秒精度）
		return t.Format("2006-01-02 15:04:05")
	default:
		// 自定义格式
		return t.Format(timeFormat)
	}
}

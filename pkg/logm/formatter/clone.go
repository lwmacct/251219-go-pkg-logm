package formatter

import "maps"

// CloneWithTimeOptions 克隆内置 formatter，并覆写时间格式和时区。
//
// 对非内置 formatter，直接返回原值。
func CloneWithTimeOptions(f Formatter, timeFormat, timezone string) Formatter {
	switch typed := f.(type) {
	case *JSONFormatter:
		clone := *typed
		clone.opts = cloneOptions(typed.opts)
		applyTimeOptions(clone.opts, timeFormat, timezone)
		return &clone
	case *TextFormatter:
		clone := *typed
		clone.opts = cloneOptions(typed.opts)
		applyTimeOptions(clone.opts, timeFormat, timezone)
		return &clone
	case *ColorTextFormatter:
		clone := *typed
		clone.opts = cloneOptions(typed.opts)
		applyTimeOptions(clone.opts, timeFormat, timezone)
		clone.priorityKeys = append([]string(nil), typed.priorityKeys...)
		clone.trailingKeys = append([]string(nil), typed.trailingKeys...)
		return &clone
	case *ColorJSONFormatter:
		clone := *typed
		clone.opts = cloneOptions(typed.opts)
		applyTimeOptions(clone.opts, timeFormat, timezone)
		return &clone
	default:
		return f
	}
}

func cloneOptions(src *Options) *Options {
	if src == nil {
		return defaultOptions()
	}

	clone := *src
	if src.RawFields != nil {
		clone.RawFields = make(map[string]bool, len(src.RawFields))
		maps.Copy(clone.RawFields, src.RawFields)
	}
	return &clone
}

func applyTimeOptions(opts *Options, timeFormat, timezone string) {
	if opts == nil {
		return
	}
	if timeFormat != "" {
		opts.TimeFormat = timeFormat
	}
	opts.Location = loadTimezone(timezone)
}

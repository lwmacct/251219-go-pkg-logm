package formatter

import "log/slog"

// normalizeAttrs 递归展开空 key group，保持与标准 slog 的内联语义一致。
func normalizeAttrs(attrs []slog.Attr) []slog.Attr {
	if len(attrs) == 0 {
		return nil
	}

	normalized := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		appendNormalizedAttr(&normalized, attr)
	}
	return normalized
}

func appendNormalizedAttr(dst *[]slog.Attr, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()

	if attr.Key == "" {
		if attr.Value.Kind() == slog.KindGroup {
			for _, child := range attr.Value.Group() {
				appendNormalizedAttr(dst, child)
			}
		}
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		attr.Value = slog.GroupValue(normalizeAttrs(attr.Value.Group())...)
	}

	*dst = append(*dst, attr)
}

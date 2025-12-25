package formatter

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
)

// ColorTextFormatter 彩色格式化器。
//
// 输出带 ANSI 颜色的日志，适合终端开发调试。
// 支持 JSON 字符串自动展开和嵌套结构平铺。
type ColorTextFormatter struct {
	opts         *Options
	flattenJSON  bool
	priorityKeys []string
	trailingKeys []string
}

// ColorText 创建彩色格式化器。
func ColorText(opts ...Option) *ColorTextFormatter {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return &ColorTextFormatter{
		opts:         o,
		flattenJSON:  true,
		priorityKeys: []string{"time", "level", "msg"},
		trailingKeys: []string{"source"},
	}
}

// Format 实现 Formatter 接口。
func (f *ColorTextFormatter) Format(r *Record) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	// 时间
	t := r.Time
	if f.opts.Location != nil {
		t = t.In(f.opts.Location)
	}
	f.writeColored(buf, f.opts.ColorScheme.Time, formatTime(t, f.opts.TimeFormat))
	buf.WriteByte(' ')

	// 级别（带颜色）
	f.writeLevel(buf, r.Level)
	buf.WriteByte(' ')

	// 消息（无色）
	buf.WriteString(r.Message)

	// 属性
	f.writeAttrs(buf, r.Attrs, r.Groups)

	// 源代码位置
	if r.Source != nil {
		buf.WriteByte(' ')
		f.writeColored(buf, f.opts.ColorScheme.Source, FormatSource(r.Source, f.opts))
	}

	buf.WriteByte('\n')

	return copyBytes(buf.Bytes()), nil
}

// writeLevel 写入级别（带颜色）
func (f *ColorTextFormatter) writeLevel(buf *bytes.Buffer, level slog.Level) {
	info := DefaultLevelInfo(level)
	color := f.opts.ColorScheme.LevelColor(level)

	if f.opts.EnableColor {
		buf.WriteString(color)
		buf.WriteString(ColorBold)
	}
	buf.WriteString(info.Name)
	if f.opts.EnableColor {
		buf.WriteString(ColorReset)
	}
}

// writeColored 写入带颜色的文本
func (f *ColorTextFormatter) writeColored(buf *bytes.Buffer, color, text string) {
	if f.opts.EnableColor {
		buf.WriteString(color)
	}
	buf.WriteString(text)
	if f.opts.EnableColor {
		buf.WriteString(ColorReset)
	}
}

// writeAttrs 写入属性
func (f *ColorTextFormatter) writeAttrs(buf *bytes.Buffer, attrs []slog.Attr, groups []string) {
	prefix := ""
	var prefixSb140 strings.Builder
	for _, g := range groups {
		prefixSb140.WriteString(g + ".")
	}
	prefix += prefixSb140.String()

	for _, attr := range attrs {
		if attr.Key == "" {
			continue
		}
		buf.WriteByte(' ')
		f.writeAttr(buf, attr, prefix)
	}
}

// writeAttr 写入单个属性
func (f *ColorTextFormatter) writeAttr(buf *bytes.Buffer, attr slog.Attr, prefix string) {
	key := prefix + attr.Key
	f.writeColored(buf, f.opts.ColorScheme.Key, key)
	buf.WriteByte('=')

	// 检查是否为 raw 字段（不加引号直接输出）
	if f.opts.RawFields[attr.Key] {
		buf.WriteString(attr.Value.Resolve().String())
		return
	}

	f.writeValue(buf, attr.Value, key)
}

// writeValue 写入值
func (f *ColorTextFormatter) writeValue(buf *bytes.Buffer, v slog.Value, keyPath string) {
	v = v.Resolve()

	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		// 尝试展开 JSON 字符串
		if f.flattenJSON && len(s) > 0 && (s[0] == '{' || s[0] == '[') {
			if expanded := f.tryFlattenJSON(s, keyPath); expanded != "" {
				buf.WriteString(expanded)
				return
			}
		}
		f.writeColored(buf, f.opts.ColorScheme.String, strconv.Quote(s))

	case slog.KindInt64:
		f.writeColored(buf, f.opts.ColorScheme.Number, strconv.FormatInt(v.Int64(), 10))

	case slog.KindUint64:
		f.writeColored(buf, f.opts.ColorScheme.Number, strconv.FormatUint(v.Uint64(), 10))

	case slog.KindFloat64:
		f.writeColored(buf, f.opts.ColorScheme.Number, strconv.FormatFloat(v.Float64(), 'f', -1, 64))

	case slog.KindBool:
		if v.Bool() {
			f.writeColored(buf, f.opts.ColorScheme.Number, "true")
		} else {
			f.writeColored(buf, f.opts.ColorScheme.Number, "false")
		}

	case slog.KindDuration:
		f.writeColored(buf, f.opts.ColorScheme.Number, v.Duration().String())

	case slog.KindTime:
		t := v.Time()
		if f.opts.Location != nil {
			t = t.In(f.opts.Location)
		}
		f.writeColored(buf, f.opts.ColorScheme.String, strconv.Quote(formatTime(t, f.opts.TimeFormat)))

	case slog.KindGroup:
		// 展开分组为平铺格式
		attrs := v.Group()
		for i, attr := range attrs {
			if i > 0 {
				buf.WriteByte(' ')
			}
			f.writeAttr(buf, attr, keyPath+".")
		}

	case slog.KindAny:
		f.writeAny(buf, v.Any(), keyPath)

	default:
		f.writeColored(buf, f.opts.ColorScheme.String, strconv.Quote(v.String()))
	}
}

// writeAny 写入任意类型
func (f *ColorTextFormatter) writeAny(buf *bytes.Buffer, v any, keyPath string) {
	if v == nil {
		f.writeColored(buf, f.opts.ColorScheme.Null, "null")
		return
	}

	// 尝试 JSON 序列化后平铺
	if f.flattenJSON {
		data, err := json.Marshal(v)
		if err == nil && len(data) > 0 {
			if expanded := f.tryFlattenJSON(string(data), keyPath); expanded != "" {
				buf.WriteString(expanded)
				return
			}
		}
	}

	// 回退到简单字符串
	data, err := json.Marshal(v)
	if err != nil {
		f.writeColored(buf, f.opts.ColorScheme.String, "<error>")
		return
	}
	f.writeColored(buf, f.opts.ColorScheme.String, string(data))
}

// tryFlattenJSON 尝试展开 JSON 为平铺格式
func (f *ColorTextFormatter) tryFlattenJSON(s string, keyPath string) string {
	var data any
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return ""
	}

	var parts []string
	f.flattenValue(data, keyPath, &parts)
	return strings.Join(parts, " ")
}

// flattenValue 递归展开值
func (f *ColorTextFormatter) flattenValue(v any, path string, parts *[]string) {
	switch val := v.(type) {
	case map[string]any:
		for k, v := range val {
			f.flattenValue(v, path+"."+k, parts)
		}
	case []any:
		for i, v := range val {
			f.flattenValue(v, path+"["+strconv.Itoa(i)+"]", parts)
		}
	case string:
		*parts = append(*parts, f.coloredKV(path, strconv.Quote(val)))
	case float64:
		*parts = append(*parts, f.coloredKV(path, strconv.FormatFloat(val, 'f', -1, 64)))
	case bool:
		*parts = append(*parts, f.coloredKV(path, strconv.FormatBool(val)))
	case nil:
		*parts = append(*parts, f.coloredKV(path, "null"))
	default:
		data, err := json.Marshal(val)
		if err != nil {
			*parts = append(*parts, f.coloredKV(path, "<error>"))
			return
		}
		*parts = append(*parts, f.coloredKV(path, string(data)))
	}
}

// coloredKV 生成带颜色的 key=value
func (f *ColorTextFormatter) coloredKV(key, value string) string {
	if f.opts.EnableColor {
		return f.opts.ColorScheme.Key + key + ColorReset + "=" + f.opts.ColorScheme.String + value + ColorReset
	}
	return key + "=" + value
}

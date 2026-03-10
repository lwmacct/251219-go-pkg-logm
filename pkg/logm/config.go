package logm

import (
	"log/slog"
	"strings"
	"time"

	writerpkg "github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

// Config 定义 logger 的完整配置。
type Config struct {
	Level        string
	LevelVar     *slog.LevelVar
	Formatter    Formatter
	Output       Writer
	SlogHandlers []slog.Handler
	AddSource    bool
	TimeFormat   string
	Timezone     string
	Location     *time.Location
	Interceptors []Interceptor
}

// PresetDefault 返回默认配置。
func PresetDefault() Config {
	return Config{
		Level:      "INFO",
		TimeFormat: "datetime",
		Timezone:   "Asia/Shanghai",
	}
}

// ParseOutput 将字符串输出配置解析为 Writer。
//
// 支持:
//   - "stdout"
//   - "stderr"
//   - 文件路径
//   - 多个输出使用逗号分隔，如 "stdout,/tmp/app.log"
func ParseOutput(raw string) Writer {
	parts := strings.Split(raw, ",")
	writers := make([]Writer, 0, len(parts))
	for _, part := range parts {
		output := strings.TrimSpace(part)
		if output == "" {
			continue
		}
		if resolved := outputWriter(output); resolved != nil {
			writers = append(writers, resolved)
		}
	}

	switch len(writers) {
	case 0:
		return nil
	case 1:
		return writers[0]
	default:
		return writerpkg.Multi(writers...)
	}
}

func outputWriter(output string) Writer {
	switch output {
	case "", "stdout":
		return writerpkg.Stdout()
	case "stderr":
		return writerpkg.Stderr()
	default:
		return writerpkg.File(output)
	}
}

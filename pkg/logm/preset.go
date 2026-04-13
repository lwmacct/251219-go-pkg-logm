package logm

import (
	"os"
	"strings"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter"
	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

// PresetDev 返回开发环境预设配置。
//
// 特点：
//   - 彩色输出到 stdout
//   - DEBUG 级别
//   - 显示源代码位置
//   - 简洁时间格式 (15:04:05)
//   - sql/query 字段不加引号（方便阅读 SQL）
func PresetDev() Config {
	return Config{
		Level: "DEBUG",
		Formatter: formatter.ColorText(
			formatter.WithTimeFormat("time"),
			formatter.WithRawFields("sql", "query"),
		),
		Output:     writer.Stdout(),
		AddSource:  true,
		TimeFormat: "time",
		Timezone:   "Asia/Shanghai",
	}
}

// PresetProd 返回生产环境预设配置。
//
// 特点：
//   - JSON 格式输出
//   - INFO 级别
//   - 显示源代码位置
//   - RFC3339 时间格式
func PresetProd() Config {
	return Config{
		Level: "INFO",
		Formatter: formatter.JSON(
			formatter.WithTimeFormat("rfc3339ms"),
		),
		Output:     writer.Stdout(),
		AddSource:  true,
		TimeFormat: "rfc3339ms",
		Timezone:   "Asia/Shanghai",
	}
}

// PresetAuto 自动检测环境并返回相应配置。
//
// 检测逻辑：
//   - VSCODE_INJECTION=1 → 开发环境
//   - 否则 → 生产环境
func PresetAuto() Config {
	if os.Getenv("VSCODE_INJECTION") == "1" {
		return PresetDev()
	}
	return PresetProd()
}

// PresetFromEnv 根据环境变量返回配置。
//
// 支持的环境变量：
//   - LOGM_ENV: dev 使用开发配置，prod 使用生产配置（默认）
//   - LOGM_LEVEL: DEBUG, INFO, WARN, ERROR
//   - LOGM_FORMAT: json, text, color_text, color_json
//   - LOGM_OUTPUT: stdout, stderr, 或文件路径
//   - LOGM_SOURCE: true, false
//   - LOGM_TIME_FORMAT: time, datetime, rfc3339, rfc3339ms
func PresetFromEnv() Config {
	var cfg Config
	if isDevEnv() {
		cfg = PresetDev()
	} else {
		cfg = PresetProd()
	}

	if level := os.Getenv("LOGM_LEVEL"); level != "" {
		cfg.Level = level
	}

	if format := os.Getenv("LOGM_FORMAT"); format != "" {
		var f Formatter
		switch strings.ToLower(format) {
		case "json":
			f = formatter.JSON()
		case "text":
			f = formatter.Text()
		case "color_text":
			f = formatter.ColorText()
		case "color_json":
			f = formatter.ColorJSON()
		}
		if f != nil {
			cfg.Formatter = f
		}
	}

	if output := os.Getenv("LOGM_OUTPUT"); output != "" {
		if resolved := ParseOutput(output); resolved != nil {
			cfg.Output = resolved
		}
	}

	if source := os.Getenv("LOGM_SOURCE"); source != "" {
		cfg.AddSource = strings.ToLower(source) == "true" || source == "1"
	}

	if timeFormat := os.Getenv("LOGM_TIME_FORMAT"); timeFormat != "" {
		cfg.TimeFormat = timeFormat
	}

	return cfg
}

// isDevEnv 检测是否为开发环境
func isDevEnv() bool {
	env := strings.ToLower(os.Getenv("LOGM_ENV"))
	return env == "dev" || env == "development"
}

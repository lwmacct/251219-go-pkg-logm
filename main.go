package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/lwmacct/251125-go-mod-logger/pkg/logger"
)

func main() {
	// 示例 1: 使用默认配置（datetime 格式 + Asia/Shanghai 时区）
	demoDefaultConfig()

	// 示例 2: 使用彩色输出
	demoColoredOutput()

	// 示例 3: 使用 JSON 输出
	demoJSONOutput()

	// 示例 4: 自定义时间格式和时区
	demoCustomTimeFormat()

	// 示例 5: Context 集成
	demoContextIntegration()

	// 示例 6: 结构化日志与数据平铺
	demoStructuredLogging()
}

// demoDefaultConfig 演示默认配置
func demoDefaultConfig() {
	printSection("默认配置 (datetime + Asia/Shanghai)")

	// 默认配置：TimeFormat="datetime", Timezone="Asia/Shanghai"
	if err := logger.Init(nil); err != nil {
		panic(err)
	}

	logger.Info("服务启动", "port", 8080)
	logger.Debug("调试信息（不会显示，默认级别为 INFO）")
	logger.Warn("警告信息", "remaining", 10)
}

// demoColoredOutput 演示彩色输出
func demoColoredOutput() {
	printSection("彩色输出 (color)")

	cfg := &logger.Config{
		Level:      "DEBUG",
		Format:     "color",
		TimeFormat: "datetime",
		Timezone:   "Asia/Shanghai",
	}
	if err := logger.Init(cfg); err != nil {
		panic(err)
	}

	logger.Debug("调试信息", "module", "auth")
	logger.Info("用户登录", "user_id", 12345, "ip", "192.168.1.100")
	logger.Warn("连接池即将耗尽", "active", 95, "max", 100)
	logger.Error("数据库连接失败", "error", "connection refused", "host", "db.example.com")
}

// demoJSONOutput 演示 JSON 输出
func demoJSONOutput() {
	printSection("JSON 输出")

	cfg := &logger.Config{
		Level:      "INFO",
		Format:     "json",
		TimeFormat: "datetime",
		Timezone:   "Asia/Shanghai",
	}
	if err := logger.Init(cfg); err != nil {
		panic(err)
	}

	logger.Info("API 请求", "method", "POST", "path", "/api/users", "duration_ms", 42)
}

// demoCustomTimeFormat 演示自定义时间格式
func demoCustomTimeFormat() {
	printSection("自定义时间格式")

	// 使用 RFC3339 格式
	cfg := &logger.Config{
		Level:      "INFO",
		Format:     "color",
		TimeFormat: "rfc3339",
		Timezone:   "Asia/Shanghai",
	}
	if err := logger.Init(cfg); err != nil {
		panic(err)
	}
	logger.Info("RFC3339 格式")

	// 使用自定义 Go 时间格式字符串
	cfg.TimeFormat = "2006/01/02 15:04:05"
	if err := logger.Init(cfg); err != nil {
		panic(err)
	}
	logger.Info("自定义格式 (yyyy/mm/dd)")

	// 使用固定偏移时区
	cfg.TimeFormat = "datetime"
	cfg.Timezone = "+08:00"
	if err := logger.Init(cfg); err != nil {
		panic(err)
	}
	logger.Info("固定偏移时区 (+08:00)")
}

// demoContextIntegration 演示 Context 集成
func demoContextIntegration() {
	printSection("Context 集成")

	cfg := &logger.Config{
		Level:  "INFO",
		Format: "color",
	}
	if err := logger.Init(cfg); err != nil {
		panic(err)
	}

	// 创建带 request_id 的 context
	ctx := context.Background()
	ctx = logger.WithRequestID(ctx, "req-abc123")

	// 从 context 获取 logger
	log := logger.FromContext(ctx)
	log.Info("处理请求", "action", "create_user")
}

// demoStructuredLogging 演示结构化日志与数据平铺
func demoStructuredLogging() {
	printSection("结构化日志与数据平铺")

	cfg := &logger.Config{
		Level:  "INFO",
		Format: "color",
	}
	if err := logger.Init(cfg); err != nil {
		panic(err)
	}

	// JSON 字符串自动平铺
	logger.Info("收到请求体", "body", `{"user":"alice","age":30}`)

	// map 自动平铺
	logger.Info("配置信息", "config", map[string]any{
		"debug":   true,
		"timeout": 30,
	})

	// slog.Group 分组
	logger.Info("HTTP 请求",
		slog.Group("request",
			"method", "GET",
			"path", "/api/users",
		),
		slog.Group("response",
			"status", 200,
			"size", 1024,
		),
	)

	// 带属性的 logger
	serviceLogger := logger.WithAttrs("service", "payment", "version", "2.0")
	serviceLogger.Info("支付成功", "order_id", "ORD-001", "amount", 99.99)
}

// printSection 打印分隔标题
func printSection(title string) {
	os.Stdout.WriteString("\n")
	os.Stdout.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	os.Stdout.WriteString("  " + title + "\n")
	os.Stdout.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

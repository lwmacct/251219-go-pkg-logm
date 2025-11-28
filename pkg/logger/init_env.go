package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// globalCloser 保存全局 logger 的可关闭资源
var globalCloser io.Closer

// InitCfg 使用配置初始化全局日志系统
//
// 这个函数应该在应用启动时调用一次，用于配置全局的 slog.Default() logger
// 如果输出到文件，应在程序退出时调用 Close() 关闭文件
//
// 使用示例：
//
//	func main() {
//	    if err := logger.InitCfg(&logger.Config{
//	        Level:  "DEBUG",
//	        Format: "color",
//	    }); err != nil {
//	        log.Fatalf("初始化日志失败: %v", err)
//	    }
//	    defer logger.Close()
//	    // ...
//	}
func InitCfg(cfg *Config) error {
	logger, closer, err := NewWithCloser(cfg)
	if err != nil {
		return err
	}
	// 关闭之前的 closer（忽略错误，因为我们正在替换它）
	if globalCloser != nil {
		_ = globalCloser.Close()
	}
	globalCloser = closer
	slog.SetDefault(logger)
	return nil
}

// InitEnv 从环境变量初始化全局日志系统
//
// 这是初始化日志的便捷方式，适用于通过环境变量配置应用的场景（如容器化部署）。
//
// 支持的环境变量：
//   - LOG_LEVEL: 日志级别 (DEBUG, INFO, WARN, ERROR)，默认 INFO
//   - LOG_FORMAT: 输出格式 (json, text, color/colored)，默认 color
//   - LOG_OUTPUT: 输出目标 (stdout, stderr, 或文件路径)，默认 stdout
//   - LOG_ADD_SOURCE: 是否添加源代码位置 (true, false)，默认 true
//   - LOG_TIME_FORMAT: 时间格式，默认 datetime
//
// 时间格式可选值：datetime, rfc3339, rfc3339ms, time, timems, 或自定义格式
//
// 使用示例：
//
//	func main() {
//	    if err := logger.InitEnv(); err != nil {
//	        log.Fatalf("初始化日志失败: %v", err)
//	    }
//	    defer logger.Close()
//	    // ...
//	}
func InitEnv() error {
	cfg := &Config{
		Level:      getEnv("LOG_LEVEL", "INFO"),
		Format:     getEnv("LOG_FORMAT", "color"),
		Output:     getEnv("LOG_OUTPUT", "stdout"),
		AddSource:  getEnvBool("LOG_ADD_SOURCE", true),
		TimeFormat: getEnv("LOG_TIME_FORMAT", "datetime"),
	}

	return InitCfg(cfg)
}

// InitAuto 根据环境自动选择配置初始化日志系统
//
// 通过 IS_SANDBOX 环境变量自动检测运行环境，并选择合适的默认配置：
//   - 开发环境 (IS_SANDBOX=1): 彩色输出、DEBUG 级别、显示源码位置、简洁时间格式
//   - 生产环境: JSON 格式、INFO 级别、无源码位置、完整时间格式
//
// 所有配置仍可通过对应的环境变量覆盖。
//
// 默认值对比：
//
//	| 配置项         | 开发环境 (IS_SANDBOX=1) | 生产环境       |
//	|----------------|-------------------------|----------------|
//	| LOG_LEVEL      | DEBUG                   | INFO           |
//	| LOG_FORMAT     | color                   | json           |
//	| LOG_ADD_SOURCE | true                    | false          |
//	| LOG_TIME_FORMAT| time (15:04:05)         | datetime       |
//
// 使用示例：
//
//	func main() {
//	    if err := logger.InitAuto(); err != nil {
//	        log.Fatalf("初始化日志失败: %v", err)
//	    }
//	    defer logger.Close()
//	    // ...
//	}
func InitAuto() error {
	isSandbox := isSandboxEnv()

	// 根据环境选择默认值
	defaultLevel := "INFO"
	defaultFormat := "json"
	defaultAddSource := false
	defaultTimeFormat := "datetime"

	if isSandbox {
		defaultLevel = "DEBUG"
		defaultFormat = "color"
		defaultAddSource = true
		defaultTimeFormat = "time"
	}

	cfg := &Config{
		Level:      getEnv("LOG_LEVEL", defaultLevel),
		Format:     getEnv("LOG_FORMAT", defaultFormat),
		Output:     getEnv("LOG_OUTPUT", "stdout"),
		AddSource:  getEnvBool("LOG_ADD_SOURCE", defaultAddSource),
		TimeFormat: getEnv("LOG_TIME_FORMAT", defaultTimeFormat),
	}

	return InitCfg(cfg)
}

// Close 关闭全局 logger 的资源（如文件）
//
// 应在程序退出时调用，确保日志文件正确关闭
func Close() error {
	if globalCloser != nil {
		err := globalCloser.Close()
		globalCloser = nil
		return err
	}
	return nil
}

// isSandboxEnv 检测是否为沙盒/开发环境
func isSandboxEnv() bool {
	value := os.Getenv("IS_SANDBOX")
	return value == "1" || strings.ToLower(value) == "true"
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.ToLower(value) == "true" || value == "1"
}

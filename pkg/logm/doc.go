// Package logm 提供统一的结构化日志系统。
//
// 基于 Go 1.26+ 的 log/slog 包构建，采用显式 Config 结构配置，
// 支持多种输出格式、日志轮转、异步写入和动态级别调整。
//
// # Architecture
//
// logm 包采用标准 slog.Handler + Writer 架构：
//   - Handler: 基于标准库 TextHandler / JSONHandler 组装
//   - ReplaceAttr: 统一处理时间、source、error 和 JSON 展开
//   - Writer: 输出目标接口，支持多种输出（Stdout/File/Async/Multi）
//
// # Quick Start
//
// 最简单的使用方式是使用预设配置：
//
//	func main() {
//	    // 开发环境：彩色输出 + DEBUG + 源代码位置
//	    logm.MustInit(logm.PresetDev())
//	    defer logm.Close()
//
//	    slog.Info("应用启动", "version", "1.0.0")
//	}
//
// 生产环境：
//
//	logm.MustInit(logm.PresetProd())
//
// 从环境变量读取配置：
//
//	logm.MustInit(logm.PresetFromEnv())
//
// # Config
//
// 使用 Config 进行精确配置：
//
//	logm.Init(logm.Config{
//	    Level:  "DEBUG",
//	    Format: logm.FormatText,
//	    Color:  true,
//	    Output: writer.Multi(
//	        writer.Stdout(),
//	        writer.File("/var/log/app.log", writer.WithRotation(100, 7)),
//	    ),
//	    AddSource: true,
//	})
//
// Go 1.26 起也可混用多个 slog.Handler：
//
//	logm.Init(logm.Config{
//	    Output: writer.Stdout(),
//	    SlogHandlers: []slog.Handler{
//	        slog.NewJSONHandler(file, nil),
//	    },
//	})
//
// logm 会自动用 slog.NewMultiHandler 同时分发到两路 Handler。
//
// # Writers
//
// writer 子包提供输出目标实现：
//
//	import "github.com/.../logm/writer"
//
//	writer.Stdout()                          // 标准输出
//	writer.File(path, writer.WithRotation(100, 7))  // 带轮转的文件
//	writer.Async(w, 1000)                    // 异步写入
//	writer.Multi(w1, w2)                     // 多目标输出
//
// # Dynamic Level
//
// 支持运行时动态调整日志级别：
//
//	logm.SetLevel("DEBUG")  // 开启调试日志
//	logm.SetLevel("ERROR")  // 只显示错误
//
// # ReplaceAttr
//
// 使用 ReplaceAttr 统一改写结构化字段：
//
//	cfg := logm.PresetDefault()
//	cfg.ReplaceAttr = func(groups []string, attr slog.Attr) slog.Attr {
//	    if attr.Key == "trace_id" && attr.Value.String() == "" {
//	        return slog.String("trace_id", "missing")
//	    }
//	    return attr
//	}
//	logm.Init(cfg)
//
// # Context Integration
//
// 在 HTTP 请求等场景中，可将 logger 存入 context 实现请求追踪：
//
//	func Handler(w http.ResponseWriter, r *http.Request) {
//	    ctx := logm.WithRequestID(r.Context(), r.Header.Get("X-Request-ID"))
//	    log := logm.FromContext(ctx)
//	    log.Info("处理请求", "path", r.URL.Path)
//	}
//
// # Thread Safety
//
// 本包所有导出函数都是并发安全的。全局 logger 可在多个 goroutine 中安全使用。
// [slog.Logger] 实例也是并发安全的，可以在 context 中自由传递。
// 动态级别调整（SetLevel）也是线程安全的。
package logm

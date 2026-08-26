package logm

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// FormatBytes 将字节数格式化为人类可读的字符串（如 "1.5 KB"、"2.3 MB"）。
//
// 使用 1024 为单位换算，支持 B、KB、MB、GB、TB、PB、EB。
// 常用于日志中输出文件大小、网络传输量等信息。
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// LogError 记录错误日志并返回原始错误，适用于同时需要记录和返回错误的场景。
//
// logger 从 ctx 提取，错误会作为 "error" 字段自动添加到日志属性中。
//
// 示例：
//
//	return logm.LogError(ctx, "数据库查询失败", err, "table", "users")
func LogError(ctx context.Context, msg string, err error, attrs ...any) error {
	logger := FromContext(ctx)

	// 合并错误到属性中
	allAttrs := append([]any{"error", err}, attrs...)
	logger.Error(msg, allAttrs...)

	return err
}

// LogAndWrap 记录错误日志并返回带有上下文信息的包装错误。
//
// 与 [LogError] 不同，该函数使用 fmt.Errorf 的 %w 动词包装原始错误，
// 使得错误链可以通过 [errors.Is] 和 [errors.As] 追溯。
//
// 示例：
//
//	return logm.LogAndWrap("保存配置失败", err, "path", configPath)
//	// 返回错误: "保存配置失败: original error"
func LogAndWrap(msg string, err error, attrs ...any) error {
	allAttrs := append([]any{"error", err}, attrs...)
	slog.Error(msg, allAttrs...)
	return fmt.Errorf("%s: %w", msg, err)
}

// LogWithPC 使用指定的 PC（程序计数器）记录日志。
//
// 配合 [CallerPC] 使用，可以在日志封装场景中正确显示调用源位置。
// 当 pc 为 0 时，日志不会包含 source 信息。
//
// 示例：
//
//	pc := logm.CallerPC("gorm.io/gorm", "myapp/database")
//	logm.LogWithPC(ctx, slog.LevelDebug, pc, "query executed",
//	    slog.Duration("elapsed", elapsed),
//	    slog.String("sql", sql),
//	)
func LogWithPC(ctx context.Context, level slog.Level, pc uintptr, msg string, attrs ...slog.Attr) {
	if ctx == nil {
		ctx = context.Background()
	}
	logger := FromContext(ctx)
	if !logger.Enabled(ctx, level) {
		return
	}

	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.AddAttrs(attrs...)
	_ = logger.Handler().Handle(ctx, r)
}

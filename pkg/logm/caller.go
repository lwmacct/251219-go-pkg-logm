package logm

import (
	"runtime"
	"strings"
)

// CallerPC 获取第一个不在指定包路径中的调用者 PC。
//
// 遍历调用栈，跳过函数名包含 skipPkgs 中任意字符串的栈帧，
// 返回第一个"外部"调用者的 PC。适用于日志封装场景，
// 可以跳过日志库和中间件的内部调用，定位到业务代码。
//
// 示例：
//
//	// 在 GORM logger 中使用
//	pc := logm.CallerPC("gorm.io/gorm", "infrastructure/database")
//	logm.LogWithPC(ctx, slog.LevelDebug, pc, "GORM query", attrs...)
func CallerPC(skipPkgs ...string) uintptr {
	var pcs [32]uintptr
	n := runtime.Callers(2, pcs[:]) // 跳过 runtime.Callers 和 CallerPC
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		skip := false
		for _, pkg := range skipPkgs {
			if strings.Contains(frame.Function, pkg) {
				skip = true
				break
			}
		}
		if !skip {
			return frame.PC
		}
		if !more {
			break
		}
	}
	return 0
}

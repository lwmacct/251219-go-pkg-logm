package logm

import (
	"runtime"
	"strings"
	"testing"
)

func TestCallerPC(t *testing.T) {
	// 直接调用，应该返回当前测试函数的 PC
	pc := CallerPC()
	if pc == 0 {
		t.Error("CallerPC() should return non-zero PC")
	}

	// 验证返回的 PC 指向测试函数
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	if !strings.Contains(frame.Function, "TestCallerPC") {
		t.Errorf("expected function to contain 'TestCallerPC', got %s", frame.Function)
	}
}

func TestCallerPC_SkipPackages(t *testing.T) {
	// 通过中间函数调用，测试跳过功能
	pc := wrapperFunc()
	if pc == 0 {
		t.Error("CallerPC() should return non-zero PC")
	}

	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()

	// 应该跳过 wrapperFunc，返回测试函数
	if strings.Contains(frame.Function, "wrapperFunc") {
		t.Errorf("should skip wrapperFunc, got %s", frame.Function)
	}
	if !strings.Contains(frame.Function, "TestCallerPC_SkipPackages") {
		t.Errorf("expected function to contain 'TestCallerPC_SkipPackages', got %s", frame.Function)
	}
}

// wrapperFunc 模拟中间封装层
func wrapperFunc() uintptr {
	return CallerPC("logm.wrapperFunc")
}

func TestCallerPC_MultipleSkipPackages(t *testing.T) {
	// 测试多层嵌套跳过
	pc := outerWrapper()
	if pc == 0 {
		t.Error("CallerPC() should return non-zero PC")
	}

	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()

	// 应该跳过 outerWrapper 和 innerWrapper
	if strings.Contains(frame.Function, "Wrapper") {
		t.Errorf("should skip all wrappers, got %s", frame.Function)
	}
}

func outerWrapper() uintptr {
	return innerWrapper()
}

func innerWrapper() uintptr {
	return CallerPC("logm.outerWrapper", "logm.innerWrapper")
}

func TestCallerPC_NoMatch(t *testing.T) {
	// 当没有匹配的跳过规则时，应返回调用者
	pc := CallerPC("nonexistent.package")
	if pc == 0 {
		t.Error("CallerPC() should return non-zero PC")
	}

	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	if !strings.Contains(frame.Function, "TestCallerPC_NoMatch") {
		t.Errorf("expected function to contain 'TestCallerPC_NoMatch', got %s", frame.Function)
	}
}

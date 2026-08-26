package logm

import (
	"runtime"
	"strings"
	"testing"
)

func TestCallerPCFindsExternalFrame(t *testing.T) {
	pc := callerWrapper()
	if pc == 0 {
		t.Fatal("CallerPC returned zero")
	}
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	if strings.Contains(frame.Function, "callerWrapper") {
		t.Fatalf("frame = %s", frame.Function)
	}
}

func callerWrapper() uintptr { return CallerPC("logm.callerWrapper") }

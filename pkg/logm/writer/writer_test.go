package writer

import (
	"bytes"
	"errors"
	"io"
	"runtime/pprof"
	"sync"
	"testing"
	"testing/synctest"
)

type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }

type syncWriter struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	err  error
	sync int
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return 0, w.err
	}
	return w.buf.Write(p)
}
func (w *syncWriter) Sync() error { w.sync++; return w.err }

func TestMultiPropagatesErrors(t *testing.T) {
	want := errors.New("boom")
	m := Multi(&bytes.Buffer{}, errWriter{err: want})
	n, err := m.Write([]byte("x"))
	if n != 0 || !errors.Is(err, want) {
		t.Fatalf("Write() = %d, %v", n, err)
	}
}

func TestAsyncCloseDrainsAndPropagates(t *testing.T) {
	inner := &syncWriter{}
	a := NewAsync(inner, AsyncConfig{Capacity: 2, Overflow: OverflowBlock})
	if _, err := a.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if got := inner.buf.String(); got != "hello" {
		t.Fatalf("buffer = %q", got)
	}
	if _, err := a.Write([]byte("x")); !errors.Is(err, ErrAsyncWriterClosed) {
		t.Fatalf("write after close = %v", err)
	}
}

func TestAsyncCloseDoesNotLeakGoroutine(t *testing.T) {
	before := pprof.Lookup("goroutineleak").Count()
	a := NewAsync(io.Discard, AsyncConfig{Capacity: 8})
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if after := pprof.Lookup("goroutineleak").Count(); after > before {
		t.Fatalf("goroutine leaks increased: before=%d after=%d", before, after)
	}
}

func TestAsyncWriteError(t *testing.T) {
	want := errors.New("write failed")
	a := NewAsync(errWriter{err: want}, AsyncConfig{Capacity: 1})
	if _, err := a.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); !errors.Is(err, want) {
		t.Fatalf("Close() = %v", err)
	}
}

func TestAsyncConcurrentClose(t *testing.T) {
	a := NewAsync(io.Discard, AsyncConfig{Capacity: 8})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = a.Close() }()
	}
	wg.Wait()
}

func TestAsyncSynctestDrain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := NewAsync(io.Discard, AsyncConfig{Capacity: 2})
		if _, err := a.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
		if err := a.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAsyncOverflowFail(t *testing.T) {
	block := make(chan struct{})
	inner := blockingWriter{release: block}
	a := NewAsync(inner, AsyncConfig{Capacity: 1, Overflow: OverflowFail})
	defer func() { close(block); _ = a.Close() }()
	var full bool
	for i := 0; i < 10; i++ {
		_, err := a.Write([]byte("x"))
		if errors.Is(err, ErrAsyncWriterFull) {
			full = true
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if !full {
		t.Fatal("queue never became full")
	}
}

type blockingWriter struct{ release chan struct{} }

func (w blockingWriter) Write(p []byte) (int, error) {
	<-w.release
	return len(p), nil
}

var _ io.Writer = (*AsyncWriter)(nil)

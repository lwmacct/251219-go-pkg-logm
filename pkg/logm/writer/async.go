package writer

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

var (
	ErrAsyncWriterClosed = errors.New("async writer is closed")
	ErrAsyncWriterFull   = errors.New("async writer buffer is full")
)

// OverflowPolicy controls behavior when the asynchronous queue is full.
type OverflowPolicy uint8

const (
	OverflowBlock OverflowPolicy = iota
	OverflowDropNewest
	OverflowDropOldest
	OverflowFail
)

// AsyncConfig configures NewAsync. Capacity defaults to 1024. OverflowBlock
// preserves every record and applies backpressure; the other policies favor
// bounded latency at the cost of dropped records.
type AsyncConfig struct {
	Capacity int
	Overflow OverflowPolicy
}

type asyncRequest struct {
	data  []byte
	flush chan error
	close chan error
}

// AsyncWriter is a bounded, lifecycle-aware asynchronous io.Writer. Write
// errors are retained and returned by Sync, Close, and Err.
type AsyncWriter struct {
	writer   io.Writer
	requests chan asyncRequest
	policy   OverflowPolicy
	wg       sync.WaitGroup

	mu       sync.Mutex
	errMu    sync.Mutex
	closed   bool
	closeErr error
	writeErr error
	dropped  atomic.Uint64
}

// Async is a convenience constructor using OverflowBlock for compatibility
// with the common Async(w, capacity) form.
func Async(w io.Writer, capacity int) *AsyncWriter {
	return NewAsync(w, AsyncConfig{Capacity: capacity, Overflow: OverflowBlock})
}

func NewAsync(w io.Writer, cfg AsyncConfig) *AsyncWriter {
	if w == nil {
		w = io.Discard
	}
	if cfg.Capacity <= 0 {
		cfg.Capacity = 1024
	}
	a := &AsyncWriter{writer: w, requests: make(chan asyncRequest, cfg.Capacity), policy: cfg.Overflow}
	a.wg.Add(1)
	go a.run()
	return a
}

func (a *AsyncWriter) run() {
	defer a.wg.Done()
	for req := range a.requests {
		switch {
		case req.flush != nil:
			err := a.flushUnderlying()
			req.flush <- err
		case req.close != nil:
			err := a.flushUnderlying()
			if c, ok := a.writer.(io.Closer); ok {
				if closeErr := c.Close(); err == nil {
					err = closeErr
				} else if closeErr != nil {
					err = errors.Join(err, closeErr)
				}
			}
			a.errMu.Lock()
			a.closeErr = errors.Join(err, a.writeErr)
			closeErr := a.closeErr
			a.errMu.Unlock()
			req.close <- closeErr
			return
		default:
			n, err := a.writer.Write(req.data)
			if err == nil && n != len(req.data) {
				err = io.ErrShortWrite
			}
			if err != nil {
				a.errMu.Lock()
				if a.writeErr == nil {
					a.writeErr = err
				} else {
					a.writeErr = errors.Join(a.writeErr, err)
				}
				a.errMu.Unlock()
			}
		}
	}
}

func (a *AsyncWriter) flushUnderlying() error {
	if s, ok := a.writer.(Syncer); ok {
		if err := s.Sync(); err != nil {
			a.errMu.Lock()
			if a.writeErr == nil {
				a.writeErr = err
			} else {
				a.writeErr = errors.Join(a.writeErr, err)
			}
			a.errMu.Unlock()
			return err
		}
	}
	a.errMu.Lock()
	err := a.writeErr
	a.errMu.Unlock()
	return err
}

func (a *AsyncWriter) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return 0, ErrAsyncWriterClosed
	}
	data := append([]byte(nil), p...)
	req := asyncRequest{data: data}
	switch a.policy {
	case OverflowDropNewest:
		select {
		case a.requests <- req:
			return len(p), nil
		default:
			a.dropped.Add(1)
			return len(p), nil
		}
	case OverflowDropOldest:
		select {
		case a.requests <- req:
			return len(p), nil
		default:
			select {
			case <-a.requests:
			default:
			}
			select {
			case a.requests <- req:
				a.dropped.Add(1)
				return len(p), nil
			default:
				a.dropped.Add(1)
				return len(p), nil
			}
		}
	case OverflowFail:
		select {
		case a.requests <- req:
			return len(p), nil
		default:
			return 0, ErrAsyncWriterFull
		}
	default:
		// Blocking while holding the state mutex prevents Close from racing a
		// send and guarantees that a successful Write is drained before Close.
		a.requests <- req
		return len(p), nil
	}
}

func (a *AsyncWriter) enqueueControl(req asyncRequest) {
	a.requests <- req
}

func (a *AsyncWriter) Close() error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		a.wg.Wait()
		a.errMu.Lock()
		err := a.closeErr
		a.errMu.Unlock()
		return err
	}
	a.closed = true
	done := make(chan error, 1)
	a.enqueueControl(asyncRequest{close: done})
	a.mu.Unlock()
	err := <-done
	a.wg.Wait()
	return err
}

func (a *AsyncWriter) Sync() error {
	a.mu.Lock()
	if a.closed {
		a.errMu.Lock()
		err := errors.Join(a.writeErr, a.closeErr)
		a.errMu.Unlock()
		a.mu.Unlock()
		return err
	}
	done := make(chan error, 1)
	a.enqueueControl(asyncRequest{flush: done})
	a.mu.Unlock()
	return <-done
}

func (a *AsyncWriter) Err() error {
	a.errMu.Lock()
	writeErr := a.writeErr
	a.errMu.Unlock()
	a.errMu.Lock()
	closeErr := a.closeErr
	a.errMu.Unlock()
	return errors.Join(writeErr, closeErr)
}

func (a *AsyncWriter) Dropped() uint64 { return a.dropped.Load() }

func (a *AsyncWriter) Capacity() int { return cap(a.requests) }

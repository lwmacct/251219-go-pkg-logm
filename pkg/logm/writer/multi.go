package writer

import (
	"errors"
	"io"
	"sync"
)

// MultiWriter fans each record out to every sink. All sinks are attempted even
// when one fails; errors are joined so a transient secondary sink cannot hide
// the primary failure.
type MultiWriter struct {
	mu      sync.RWMutex
	writers []io.Writer
}

func Multi(writers ...io.Writer) *MultiWriter {
	filtered := make([]io.Writer, 0, len(writers))
	for _, w := range writers {
		if w != nil {
			filtered = append(filtered, w)
		}
	}
	return &MultiWriter{writers: filtered}
}

func (m *MultiWriter) Write(p []byte) (int, error) {
	m.mu.RLock()
	writers := append([]io.Writer(nil), m.writers...)
	m.mu.RUnlock()
	if len(writers) == 0 {
		return 0, io.ErrClosedPipe
	}
	minN := len(p)
	var errs []error
	for _, w := range writers {
		n, err := w.Write(p)
		if n < minN {
			minN = n
		}
		if err == nil && n != len(p) {
			err = io.ErrShortWrite
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return minN, errors.Join(errs...)
	}
	return len(p), nil
}

func (m *MultiWriter) Close() error {
	m.mu.RLock()
	writers := append([]io.Writer(nil), m.writers...)
	m.mu.RUnlock()
	var errs []error
	for _, w := range writers {
		if c, ok := w.(io.Closer); ok {
			if err := c.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (m *MultiWriter) Sync() error {
	m.mu.RLock()
	writers := append([]io.Writer(nil), m.writers...)
	m.mu.RUnlock()
	var errs []error
	for _, w := range writers {
		if s, ok := w.(Syncer); ok {
			if err := s.Sync(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (m *MultiWriter) Add(w io.Writer) {
	if w == nil {
		return
	}
	m.mu.Lock()
	m.writers = append(m.writers, w)
	m.mu.Unlock()
}

func (m *MultiWriter) Writers() []io.Writer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]io.Writer(nil), m.writers...)
}

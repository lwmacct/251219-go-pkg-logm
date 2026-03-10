package writer

import (
	"sync"
)

type asyncRequest struct {
	data  []byte
	flush chan error
	close chan error
}

// AsyncWriter 异步 Writer。
//
// 使用缓冲通道异步写入，提升高并发场景下的性能。
// 调用 Close 时会等待所有缓冲数据写入完成。
type AsyncWriter struct {
	writer   Writer
	requests chan asyncRequest
	wg       sync.WaitGroup
	closed   bool
	mu       sync.Mutex
}

// Async 创建异步 Writer。
//
// bufferSize 指定缓冲通道大小，建议值 1000-10000。
func Async(w Writer, bufferSize int) *AsyncWriter {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	aw := &AsyncWriter{
		writer:   w,
		requests: make(chan asyncRequest, bufferSize),
	}

	aw.wg.Add(1)
	go aw.run()

	return aw
}

// run 后台写入协程
func (a *AsyncWriter) run() {
	defer a.wg.Done()
	for req := range a.requests {
		switch {
		case req.flush != nil:
			req.flush <- a.writer.Sync()
		case req.close != nil:
			err := a.writer.Sync()
			if closeErr := a.writer.Close(); err == nil {
				err = closeErr
			}
			req.close <- err
			return
		default:
			_, _ = a.writer.Write(req.data)
		}
	}
}

// Write 实现 io.Writer。
//
// 将数据复制后放入缓冲通道，非阻塞（除非缓冲区满）。
func (a *AsyncWriter) Write(p []byte) (n int, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return 0, nil
	}

	// 复制数据避免竞态
	data := make([]byte, len(p))
	copy(data, p)

	select {
	case a.requests <- asyncRequest{data: data}:
		return len(p), nil
	default:
		// 缓冲区满，丢弃日志（或可选择阻塞）
		return len(p), nil
	}
}

// Close 实现 io.Closer。
//
// 等待所有已入队数据写入完成，再关闭底层 Writer。
func (a *AsyncWriter) Close() error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	a.closed = true
	done := make(chan error, 1)
	a.requests <- asyncRequest{close: done}
	a.mu.Unlock()

	err := <-done
	a.wg.Wait()
	return err
}

// Sync 实现 Writer.Sync。
//
// 等待当前缓冲区数据写入完成。
func (a *AsyncWriter) Sync() error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	done := make(chan error, 1)
	a.requests <- asyncRequest{flush: done}
	a.mu.Unlock()

	return <-done
}

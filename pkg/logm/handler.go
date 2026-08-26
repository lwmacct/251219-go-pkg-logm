package logm

import (
	"context"
	"log/slog"
)

// middlewareHandler adapts record middleware to slog.Handler while preserving
// slog's WithAttrs and WithGroup semantics.
type middlewareHandler struct {
	next       slog.Handler
	middleware []Middleware
	onError    func(error)
}

func (h *middlewareHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *middlewareHandler) Handle(ctx context.Context, record slog.Record) error {
	record = record.Clone()
	for _, middleware := range h.middleware {
		if middleware == nil {
			continue
		}
		var keep bool
		record, keep = middleware(ctx, record)
		if !keep {
			return nil
		}
	}
	err := h.next.Handle(ctx, record)
	if err != nil && h.onError != nil {
		h.onError(err)
	}
	return err
}

func (h *middlewareHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &middlewareHandler{next: h.next.WithAttrs(attrs), middleware: h.middleware, onError: h.onError}
}

func (h *middlewareHandler) WithGroup(name string) slog.Handler {
	return &middlewareHandler{next: h.next.WithGroup(name), middleware: h.middleware, onError: h.onError}
}

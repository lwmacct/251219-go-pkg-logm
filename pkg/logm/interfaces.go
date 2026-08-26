package logm

import (
	"context"
	"io"
	"log/slog"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

// Format selects the standard slog handler used by an output.
type Format string

const (
	FormatText   Format = "text"
	FormatJSON   Format = "json"
	FormatPretty Format = "pretty"
)

// Output describes one log destination. Writer can be any io.Writer; it is
// closed by Logger.Close only when Own is true. A nil Writer is replaced with
// stdout. Async enables the bounded asynchronous sink in the writer package.
type Output struct {
	Name   string
	Writer io.Writer
	Format Format
	Async  writer.AsyncConfig
	Own    bool
}

// Middleware can enrich, rewrite, or drop a record before it reaches the
// configured slog handlers. The record is cloned by logm before middleware is
// invoked, so mutating it does not affect another handler or caller.
type Middleware func(context.Context, slog.Record) (slog.Record, bool)

// managed is implemented by resources owned by a Logger.
type managed interface {
	Close() error
}

type syncer interface {
	Sync() error
}

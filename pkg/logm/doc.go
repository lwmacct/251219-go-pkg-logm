// Package logm provides a small lifecycle-aware facade around log/slog.
//
// Log records are serialized exclusively by the standard library's
// slog.TextHandler and slog.JSONHandler. Config supports multiple outputs,
// dynamic levels, source rewriting, middleware, asynchronous sinks, and
// deterministic Close/Sync operations. The API intentionally follows Go 1.27
// conventions and does not retain the former formatter.Record abstraction.
package logm

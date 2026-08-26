// Package writer provides lifecycle-aware io.Writer sinks for logm.
package writer

import "io"

// Writer is the minimal sink contract accepted by logm.
type Writer interface{ io.Writer }

// Syncer optionally flushes buffered data.
type Syncer interface{ Sync() error }

var (
	_ io.Writer = (*StdWriter)(nil)
	_ io.Writer = (*FileWriter)(nil)
	_ io.Writer = (*AsyncWriter)(nil)
	_ io.Writer = (*MultiWriter)(nil)
)

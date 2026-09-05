package logm

import (
	"log/slog"
	"os"
	"time"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

// PresetDev is a colorful, source-enabled text configuration for local work.
func PresetDev() Config {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.Local
	}
	return Config{
		Level:      slog.LevelDebug,
		Outputs:    []Output{{Name: "stdout", Writer: writer.Stdout(), Format: FormatPretty}},
		AddSource:  true,
		TimeFormat: "time",
		Location:   loc,
	}
}

// PresetProd is a structured JSON configuration suitable for ingestion by
// collectors. Source is kept as a structured object by slog.JSONHandler.
func PresetProd() Config {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	return Config{
		Level:      slog.LevelInfo,
		Outputs:    []Output{{Name: "stdout", Writer: writer.Stdout(), Format: FormatJSON}},
		AddSource:  true,
		TimeFormat: "rfc3339ms",
		Location:   loc,
	}
}

// PresetAuto selects development output when VSCODE_INJECTION=1, otherwise
// production output.
func PresetAuto() Config {
	if os.Getenv("VSCODE_INJECTION") == "1" {
		return PresetDev()
	}
	return PresetProd()
}

// PresetCLI returns the automatic development or production format while
// reserving stdout for a CLI command's machine-readable results.
func PresetCLI() Config {
	cfg := PresetAuto()
	for i := range cfg.Outputs {
		cfg.Outputs[i].Name = "stderr"
		cfg.Outputs[i].Writer = writer.Stderr()
	}
	return cfg
}

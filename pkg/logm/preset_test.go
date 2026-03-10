package logm

import (
	"testing"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
	"github.com/stretchr/testify/assert"
)

func TestPresetFromEnv_OutputOverridesPreset(t *testing.T) {
	t.Setenv("LOGM_ENV", "prod")
	t.Setenv("LOGM_OUTPUT", "stderr")

	resolved := normalizeConfig(PresetFromEnv(), nil)
	assert.IsType(t, writer.Stderr(), resolved.output)
}

func TestPresetFromEnv_SupportsMultipleOutputs(t *testing.T) {
	t.Setenv("LOGM_ENV", "prod")
	t.Setenv("LOGM_OUTPUT", "stdout, stderr")

	resolved := normalizeConfig(PresetFromEnv(), nil)
	multi, ok := resolved.output.(*writer.MultiWriter)
	assert.True(t, ok)
	if ok {
		members := multi.Writers()
		assert.Len(t, members, 2)
		assert.IsType(t, writer.Stdout(), members[0])
		assert.IsType(t, writer.Stderr(), members[1])
	}
}

func TestPresetFromEnv_DefaultPresetStillUsesSingleOutput(t *testing.T) {
	t.Setenv("LOGM_ENV", "dev")

	resolved := normalizeConfig(PresetFromEnv(), nil)
	assert.IsType(t, writer.Stdout(), resolved.output)
}

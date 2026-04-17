package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRegistryURLRejectsPath pins the documented limitation: a -registry
// URL with a path component (https://example.com/docker, common behind a
// reverse proxy) is rejected at flag-parse time with a clear error. The
// tool does NOT attempt to rewrite requests or inject a path prefix;
// go-containerregistry hardcodes /v2/ at the host root and does not expose
// a knob to change that, so supporting path-prefixed registries would
// require patching go-containerregistry itself (see the parseRegistryHost
// docstring in main.go).
//
// This test is the contract that tells future contributors "no, we
// deliberately removed the workaround transport"; a green test here with
// NO error message means someone re-introduced a silent path-drop
// regression.
func TestRegistryURLRejectsPath(t *testing.T) {
	output := runDeckschrubberWithFlags(t, false,
		"-registry", "https://example.com/docker",
		"-repos", "1",
		"-dry",
	)

	require.Contains(t, output, "path-prefixed registries are not supported",
		"expected a clear error rejecting path-prefixed -registry URLs; output:\n%s", output)
	require.Contains(t, output, `/docker`,
		"error should quote the offending path so the operator sees what they passed; output:\n%s", output)
}

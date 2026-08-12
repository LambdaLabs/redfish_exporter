package collector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// loadTestData loads a JSON fixture from testdata directory
func loadTestData(t *testing.T, filename string) map[string]interface{} {
	t.Helper()

	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path) //nolint:gosec // Test fixtures are controlled by test code, not user input
	require.NoError(t, err, "Failed to read test data file: %s", filename)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err, "Failed to parse test data file: %s", filename)

	return result
}

// loadTestDataInto decodes a JSON fixture into target, for tests that want a typed
// gofish resource rather than a generic map and do not need an HTTP server.
func loadTestDataInto(t *testing.T, filename string, target any) {
	t.Helper()

	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path) //nolint:gosec // Test fixtures are controlled by test code, not user input
	require.NoError(t, err, "Failed to read test data file: %s", filename)

	require.NoError(t, json.Unmarshal(data, target), "Failed to parse test data file: %s", filename)
}

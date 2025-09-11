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


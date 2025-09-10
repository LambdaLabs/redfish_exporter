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
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read test data file: %s", filename)
	
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err, "Failed to parse test data file: %s", filename)
	
	return result
}

// loadSchemaTestData loads a JSON fixture from testdata/schemas directory
func loadSchemaTestData(t *testing.T, version, resourceType string) map[string]interface{} {
	t.Helper()
	
	filename := filepath.Join("schemas", version+"_"+resourceType+".json")
	return loadTestData(t, filename)
}

// TestDataCatalog provides easy access to common test fixtures
type TestDataCatalog struct {
	t *testing.T
}

// NewTestDataCatalog creates a new test data catalog
func NewTestDataCatalog(t *testing.T) *TestDataCatalog {
	return &TestDataCatalog{t: t}
}

// ProcessorWithMetrics returns a processor with metrics link
func (c *TestDataCatalog) ProcessorWithMetrics() map[string]interface{} {
	return loadTestData(c.t, "processor_with_metrics.json")
}

// ProcessorMetricsFull returns full processor metrics with all fields
func (c *TestDataCatalog) ProcessorMetricsFull() map[string]interface{} {
	return loadTestData(c.t, "processor_metrics_full.json")
}

// ProcessorV100 returns a v1.0.0 processor (no metrics)
func (c *TestDataCatalog) ProcessorV100() map[string]interface{} {
	return loadSchemaTestData(c.t, "v1_0_0", "processor")
}

// ProcessorMetricsV100 returns v1.0.0 processor metrics (no PCIe errors)
func (c *TestDataCatalog) ProcessorMetricsV100() map[string]interface{} {
	return loadSchemaTestData(c.t, "v1_4_0", "processor_metrics")
}
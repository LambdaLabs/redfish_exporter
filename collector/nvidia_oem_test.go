package collector

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"log/slog"

	"github.com/stmcginnis/gofish/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMemoryOEMMetrics_EmptyOem(t *testing.T) {
	memoryJSON := `{
		"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0",
		"Oem": {}
	}`

	client := &common.TestClient{}
	client.CustomReturnForActions = map[string][]interface{}{
		http.MethodGet: {
			&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(memoryJSON)),
			},
		},
	}

	logger := slog.Default()
	oemClient := NewNvidiaOEMClient(client, logger)

	metrics, err := oemClient.GetMemoryOEMMetrics("/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0")
	require.NoError(t, err)

	// Should return zero values when OEM data is missing
	assert.False(t, metrics.RowRemappingFailed, "expected RowRemappingFailed to be false for empty OEM")
	assert.False(t, metrics.RowRemappingPending, "expected RowRemappingPending to be false for empty OEM")
}

func TestGetMemoryOEMMetrics_ErrorHandling(t *testing.T) {
	// Test with invalid JSON
	client := &common.TestClient{}
	client.CustomReturnForActions = map[string][]interface{}{
		http.MethodGet: {
			&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("invalid json")),
			},
		},
	}

	logger := slog.Default()
	oemClient := NewNvidiaOEMClient(client, logger)

	_, err := oemClient.GetMemoryOEMMetrics("/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0")
	assert.Error(t, err, "expected error for invalid JSON")
}

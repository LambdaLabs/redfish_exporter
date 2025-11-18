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

func TestGetPortMetricsOEMData(t *testing.T) {
	metricsJSON := `{
		"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/Ports/NVLink_0/Metrics",
		"Oem": {
			"Nvidia": {
				"@odata.type": "#NvidiaPortMetrics.v1_0_0.NvidiaPortMetrics",
				"NVLinkErrors": {
					"RuntimeError": true,
					"TrainingError": false
				},
				"LinkErrorRecoveryCount": 10,
				"LinkDownedCount": 2,
				"SymbolErrors": 100,
				"MalformedPackets": 5,
				"BitErrorRate": 0.001,
				"EffectiveBER": 0.0005
			}
		}
	}`

	client := &common.TestClient{}
	client.CustomReturnForActions = map[string][]interface{}{
		http.MethodGet: {
			&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(metricsJSON)),
			},
		},
	}

	logger := slog.Default()
	oemClient := NewNvidiaOEMClient(client, logger)

	metrics, err := oemClient.GetPortMetricsOEMData("/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/Ports/NVLink_0/Metrics")
	require.NoError(t, err)

	assert.True(t, metrics.NVLinkErrors.RuntimeError, "expected NVLinkErrors.RuntimeError to be true")
	assert.False(t, metrics.NVLinkErrors.TrainingError, "expected NVLinkErrors.TrainingError to be false")
	assert.Equal(t, int64(10), metrics.LinkErrorRecoveryCount, "expected LinkErrorRecoveryCount to be 10")
}

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

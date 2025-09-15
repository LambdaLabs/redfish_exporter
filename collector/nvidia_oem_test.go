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

func TestGetMemoryOEMMetrics(t *testing.T) {
	memoryJSON := `{
		"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0",
		"Oem": {
			"Nvidia": {
				"@odata.type": "#NvidiaMemory.v1_0_0.NvidiaMemory",
				"RowRemappingFailed": true,
				"RowRemappingPending": false
			}
		}
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

	assert.True(t, metrics.RowRemappingFailed, "expected RowRemappingFailed to be true")
	assert.False(t, metrics.RowRemappingPending, "expected RowRemappingPending to be false")
}

func TestGetMemoryMetricsOEMData(t *testing.T) {
	metricsJSON := `{
		"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0/MemoryMetrics",
		"Oem": {
			"Nvidia": {
				"@odata.type": "#NvidiaMemoryMetrics.v1_2_0.NvidiaGPUMemoryMetrics",
				"RowRemapping": {
					"CorrectableRowRemappingCount": 5,
					"HighAvailabilityBankCount": 10,
					"LowAvailabilityBankCount": 2,
					"MaxAvailabilityBankCount": 100,
					"NoAvailabilityBankCount": 0,
					"PartialAvailabilityBankCount": 3,
					"UncorrectableRowRemappingCount": 1
				}
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

	metrics, err := oemClient.GetMemoryMetricsOEMData("/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0/MemoryMetrics")
	require.NoError(t, err)

	assert.Equal(t, int64(5), metrics.CorrectableRowRemappingCount, "expected CorrectableRowRemappingCount to be 5")
	assert.Equal(t, int64(1), metrics.UncorrectableRowRemappingCount, "expected UncorrectableRowRemappingCount to be 1")
}

func TestGetProcessorMetricsOEMData(t *testing.T) {
	metricsJSON := `{
		"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/ProcessorMetrics",
		"Oem": {
			"Nvidia": {
				"@odata.type": "#NvidiaProcessorMetrics.v1_0_0.NvidiaProcessorMetrics",
				"SMUtilizationPercent": 75.5,
				"SMActivityPercent": 80.2,
				"SMOccupancyPercent": 65.0,
				"TensorCoreActivityPercent": 45.5,
				"FP16ActivityPercent": 30.0,
				"FP32ActivityPercent": 25.5,
				"FP64ActivityPercent": 10.0,
				"IntegerActivityUtilizationPercent": 15.5,
				"SRAMECCErrorThresholdExceeded": true,
				"NVLinkDataRxBandwidthGbps": 100.5,
				"NVLinkDataTxBandwidthGbps": 95.3,
				"PCIeRXBytes": 1024000,
				"PCIeTXBytes": 2048000,
				"ThrottleReasons": ["ThermalLimit", "PowerLimit"]
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

	metrics, err := oemClient.GetProcessorMetricsOEMData("/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/ProcessorMetrics")
	require.NoError(t, err)

	assert.Equal(t, 75.5, metrics.SMUtilizationPercent, "expected SMUtilizationPercent to be 75.5")
	assert.True(t, metrics.SRAMECCErrorThresholdExceeded, "expected SRAMECCErrorThresholdExceeded to be true")
	assert.Len(t, metrics.ThrottleReasons, 2, "expected 2 throttle reasons")
}

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

	assert.True(t, metrics.NVLinkErrorsRuntimeError, "expected NVLinkErrorsRuntimeError to be true")
	assert.False(t, metrics.NVLinkErrorsTrainingError, "expected NVLinkErrorsTrainingError to be false")
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
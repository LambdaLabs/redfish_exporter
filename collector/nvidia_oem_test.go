package collector

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"log/slog"

	"github.com/stmcginnis/gofish/common"
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !metrics.RowRemappingFailed {
		t.Errorf("expected RowRemappingFailed to be true")
	}

	if metrics.RowRemappingPending {
		t.Errorf("expected RowRemappingPending to be false")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.CorrectableRowRemappingCount != 5 {
		t.Errorf("expected CorrectableRowRemappingCount to be 5, got %d", metrics.CorrectableRowRemappingCount)
	}

	if metrics.UncorrectableRowRemappingCount != 1 {
		t.Errorf("expected UncorrectableRowRemappingCount to be 1, got %d", metrics.UncorrectableRowRemappingCount)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.SMUtilizationPercent != 75.5 {
		t.Errorf("expected SMUtilizationPercent to be 75.5, got %f", metrics.SMUtilizationPercent)
	}

	if !metrics.SRAMECCErrorThresholdExceeded {
		t.Errorf("expected SRAMECCErrorThresholdExceeded to be true")
	}

	if len(metrics.ThrottleReasons) != 2 {
		t.Errorf("expected 2 throttle reasons, got %d", len(metrics.ThrottleReasons))
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !metrics.NVLinkErrorsRuntimeError {
		t.Errorf("expected NVLinkErrorsRuntimeError to be true")
	}

	if metrics.NVLinkErrorsTrainingError {
		t.Errorf("expected NVLinkErrorsTrainingError to be false")
	}

	if metrics.LinkErrorRecoveryCount != 10 {
		t.Errorf("expected LinkErrorRecoveryCount to be 10, got %d", metrics.LinkErrorRecoveryCount)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return zero values when OEM data is missing
	if metrics.RowRemappingFailed {
		t.Errorf("expected RowRemappingFailed to be false for empty OEM")
	}

	if metrics.RowRemappingPending {
		t.Errorf("expected RowRemappingPending to be false for empty OEM")
	}
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
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
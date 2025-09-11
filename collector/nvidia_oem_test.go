package collector

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"log/slog"
)

// MockClient implements common.Client for testing
type MockClient struct {
	responses map[string]string
}

func NewMockClient(responses map[string]string) *MockClient {
	return &MockClient{responses: responses}
}

func (m *MockClient) Get(url string) (*http.Response, error) {
	body, ok := m.responses[url]
	if !ok {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
	
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func (m *MockClient) GetWithHeaders(url string, headers map[string]string) (*http.Response, error) {
	return m.Get(url)
}

func (m *MockClient) Post(url string, payload interface{}) (*http.Response, error) {
	return nil, nil
}

func (m *MockClient) PostWithHeaders(url string, payload interface{}, headers map[string]string) (*http.Response, error) {
	return nil, nil
}

func (m *MockClient) PostWithContentType(url string, contentType string, payload interface{}) (*http.Response, error) {
	return nil, nil
}

func (m *MockClient) Put(url string, payload interface{}) (*http.Response, error) {
	return nil, nil
}

func (m *MockClient) PutWithHeaders(url string, payload interface{}, headers map[string]string) (*http.Response, error) {
	return nil, nil
}

func (m *MockClient) Patch(url string, payload interface{}) (*http.Response, error) {
	return nil, nil
}

func (m *MockClient) PatchWithHeaders(url string, payload interface{}, headers map[string]string) (*http.Response, error) {
	return nil, nil
}

func (m *MockClient) Delete(url string) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func (m *MockClient) DeleteWithHeaders(url string, headers map[string]string) (*http.Response, error) {
	return m.Delete(url)
}

func (m *MockClient) DeleteWithPayload(url string, payload interface{}) (*http.Response, error) {
	return nil, nil
}

func (m *MockClient) DeleteWithContentType(url string, contentType string, payload interface{}) (*http.Response, error) {
	return nil, nil
}

func (m *MockClient) PostMultipart(url string, payload map[string]io.Reader) (*http.Response, error) {
	return nil, nil
}

func (m *MockClient) PostMultipartWithHeaders(url string, payload map[string]io.Reader, headers map[string]string) (*http.Response, error) {
	return nil, nil
}

func TestIsNvidiaGPUMemory(t *testing.T) {
	tests := []struct {
		name     string
		memoryID string
		expected bool
	}{
		{"GPU memory", "GPU_0_DRAM_0", true},
		{"GPU memory variant", "GPU_1_DRAM_1", true},
		{"Regular memory", "DIMM_A1", false},
		{"CPU memory", "CPU_0_DIMM_0", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNvidiaGPUMemory(tt.memoryID)
			if result != tt.expected {
				t.Errorf("IsNvidiaGPUMemory(%s) = %v, expected %v", tt.memoryID, result, tt.expected)
			}
		})
	}
}

func TestIsNvidiaGPU(t *testing.T) {
	tests := []struct {
		name        string
		processorID string
		expected    bool
	}{
		{"GPU processor", "GPU_0", true},
		{"GPU processor variant", "GPU_7", true},
		{"CPU processor", "CPU_0", false},
		{"Generic processor", "Processor1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNvidiaGPU(tt.processorID)
			if result != tt.expected {
				t.Errorf("IsNvidiaGPU(%s) = %v, expected %v", tt.processorID, result, tt.expected)
			}
		})
	}
}

func TestIsNVLinkPort(t *testing.T) {
	tests := []struct {
		name     string
		portID   string
		expected bool
	}{
		{"NVLink port", "NVLink_0", true},
		{"NVLink port high number", "NVLink_17", true},
		{"PCIe port", "PCIe_0", false},
		{"Generic port", "Port1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNVLinkPort(tt.portID)
			if result != tt.expected {
				t.Errorf("IsNVLinkPort(%s) = %v, expected %v", tt.portID, result, tt.expected)
			}
		})
	}
}

func TestGetMemoryOEMMetrics(t *testing.T) {
	mockResponses := map[string]string{
		"/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0": `{
			"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0",
			"Oem": {
				"Nvidia": {
					"@odata.type": "#NvidiaMemory.v1_0_0.NvidiaMemory",
					"RowRemappingFailed": true,
					"RowRemappingPending": false
				}
			}
		}`,
		"/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_1_DRAM_0": `{
			"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_1_DRAM_0",
			"Oem": {}
		}`,
	}

	client := NewMockClient(mockResponses)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	helper := NewNvidiaOEMHelper(client, logger)

	t.Run("Memory with Nvidia OEM", func(t *testing.T) {
		metrics, err := helper.GetMemoryOEMMetrics("/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !metrics.RowRemappingFailed {
			t.Error("expected RowRemappingFailed to be true")
		}
		if metrics.RowRemappingPending {
			t.Error("expected RowRemappingPending to be false")
		}
	})

	t.Run("Memory without Nvidia OEM", func(t *testing.T) {
		metrics, err := helper.GetMemoryOEMMetrics("/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_1_DRAM_0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if metrics.RowRemappingFailed {
			t.Error("expected RowRemappingFailed to be false (default)")
		}
		if metrics.RowRemappingPending {
			t.Error("expected RowRemappingPending to be false (default)")
		}
	})
}

func TestGetMemoryMetricsOEMData(t *testing.T) {
	mockResponses := map[string]string{
		"/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0/MemoryMetrics": `{
			"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0/MemoryMetrics",
			"Oem": {
				"Nvidia": {
					"@odata.type": "#NvidiaMemoryMetrics.v1_2_0.NvidiaGPUMemoryMetrics",
					"RowRemapping": {
						"CorrectableRowRemappingCount": 5,
						"HighAvailabilityBankCount": 5950,
						"LowAvailabilityBankCount": 1,
						"MaxAvailabilityBankCount": 5952,
						"NoAvailabilityBankCount": 1,
						"PartialAvailabilityBankCount": 0,
						"UncorrectableRowRemappingCount": 2
					}
				}
			}
		}`,
	}

	client := NewMockClient(mockResponses)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	helper := NewNvidiaOEMHelper(client, logger)

	metrics, err := helper.GetMemoryMetricsOEMData("/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0/MemoryMetrics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.CorrectableRowRemappingCount != 5 {
		t.Errorf("expected CorrectableRowRemappingCount = 5, got %d", metrics.CorrectableRowRemappingCount)
	}
	if metrics.UncorrectableRowRemappingCount != 2 {
		t.Errorf("expected UncorrectableRowRemappingCount = 2, got %d", metrics.UncorrectableRowRemappingCount)
	}
	if metrics.MaxAvailabilityBankCount != 5952 {
		t.Errorf("expected MaxAvailabilityBankCount = 5952, got %d", metrics.MaxAvailabilityBankCount)
	}
	if metrics.NoAvailabilityBankCount != 1 {
		t.Errorf("expected NoAvailabilityBankCount = 1, got %d", metrics.NoAvailabilityBankCount)
	}
}

func TestGetProcessorMetricsOEMData(t *testing.T) {
	mockResponses := map[string]string{
		"/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/ProcessorMetrics": `{
			"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/ProcessorMetrics",
			"Oem": {
				"Nvidia": {
					"@odata.type": "#NvidiaProcessorMetrics.v1_4_0.NvidiaGPUProcessorMetrics",
					"SMUtilizationPercent": 85.5,
					"SMActivityPercent": 90.2,
					"TensorCoreActivityPercent": 75.3,
					"FP32ActivityPercent": 60.0,
					"SRAMECCErrorThresholdExceeded": true,
					"NVLinkDataRxBandwidthGbps": 125.5,
					"NVLinkDataTxBandwidthGbps": 130.2,
					"PCIeRXBytes": 1000000,
					"PCIeTXBytes": 2000000,
					"ThrottleReasons": ["Thermal", "Power"]
				}
			}
		}`,
	}

	client := NewMockClient(mockResponses)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	helper := NewNvidiaOEMHelper(client, logger)

	metrics, err := helper.GetProcessorMetricsOEMData("/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/ProcessorMetrics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.SMUtilizationPercent != 85.5 {
		t.Errorf("expected SMUtilizationPercent = 85.5, got %f", metrics.SMUtilizationPercent)
	}
	if metrics.TensorCoreActivityPercent != 75.3 {
		t.Errorf("expected TensorCoreActivityPercent = 75.3, got %f", metrics.TensorCoreActivityPercent)
	}
	if !metrics.SRAMECCErrorThresholdExceeded {
		t.Error("expected SRAMECCErrorThresholdExceeded to be true")
	}
	if metrics.PCIeRXBytes != 1000000 {
		t.Errorf("expected PCIeRXBytes = 1000000, got %d", metrics.PCIeRXBytes)
	}
	if len(metrics.ThrottleReasons) != 2 || metrics.ThrottleReasons[0] != "Thermal" {
		t.Errorf("unexpected ThrottleReasons: %v", metrics.ThrottleReasons)
	}
}

func TestGetPortMetricsOEMData(t *testing.T) {
	mockResponses := map[string]string{
		"/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/Ports/NVLink_0/Metrics": `{
			"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/Ports/NVLink_0/Metrics",
			"Oem": {
				"Nvidia": {
					"@odata.type": "#NvidiaPortMetrics.v1_6_0.NvidiaNVLinkPortMetrics",
					"NVLinkErrors": {
						"RuntimeError": true,
						"TrainingError": false
					},
					"LinkErrorRecoveryCount": 10,
					"LinkDownedCount": 2,
					"SymbolErrors": 5,
					"MalformedPackets": 3,
					"BitErrorRate": 1.5e-10,
					"EffectiveBER": 1.2e-10
				}
			}
		}`,
	}

	client := NewMockClient(mockResponses)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	helper := NewNvidiaOEMHelper(client, logger)

	metrics, err := helper.GetPortMetricsOEMData("/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/Ports/NVLink_0/Metrics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !metrics.NVLinkErrorsRuntimeError {
		t.Error("expected NVLinkErrorsRuntimeError to be true")
	}
	if metrics.NVLinkErrorsTrainingError {
		t.Error("expected NVLinkErrorsTrainingError to be false")
	}
	if metrics.LinkErrorRecoveryCount != 10 {
		t.Errorf("expected LinkErrorRecoveryCount = 10, got %d", metrics.LinkErrorRecoveryCount)
	}
	if metrics.SymbolErrors != 5 {
		t.Errorf("expected SymbolErrors = 5, got %d", metrics.SymbolErrors)
	}
	if metrics.BitErrorRate != 1.5e-10 {
		t.Errorf("expected BitErrorRate = 1.5e-10, got %e", metrics.BitErrorRate)
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getFloat64", func(t *testing.T) {
		data := map[string]interface{}{
			"float":   123.45,
			"int":     100,
			"int64":   int64(200),
			"string":  "not a number",
			"missing": nil,
		}

		if v := getFloat64(data, "float"); v != 123.45 {
			t.Errorf("expected 123.45, got %f", v)
		}
		if v := getFloat64(data, "int"); v != 100.0 {
			t.Errorf("expected 100.0, got %f", v)
		}
		if v := getFloat64(data, "int64"); v != 200.0 {
			t.Errorf("expected 200.0, got %f", v)
		}
		if v := getFloat64(data, "string"); v != 0.0 {
			t.Errorf("expected 0.0 for invalid type, got %f", v)
		}
		if v := getFloat64(data, "missing"); v != 0.0 {
			t.Errorf("expected 0.0 for missing key, got %f", v)
		}
	})

	t.Run("getInt64", func(t *testing.T) {
		data := map[string]interface{}{
			"float":   123.45,
			"int":     100,
			"int64":   int64(200),
			"string":  "not a number",
			"missing": nil,
		}

		if v := getInt64(data, "float"); v != 123 {
			t.Errorf("expected 123, got %d", v)
		}
		if v := getInt64(data, "int"); v != 100 {
			t.Errorf("expected 100, got %d", v)
		}
		if v := getInt64(data, "int64"); v != 200 {
			t.Errorf("expected 200, got %d", v)
		}
		if v := getInt64(data, "string"); v != 0 {
			t.Errorf("expected 0 for invalid type, got %d", v)
		}
		if v := getInt64(data, "missing"); v != 0 {
			t.Errorf("expected 0 for missing key, got %d", v)
		}
	})
}
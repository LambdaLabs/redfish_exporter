package collector

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
)

// TestGPUCollector_NVLinkExpandFallback covers a BMC (Supermicro X14 in front of the HGX B300)
// that answers 404 to the expanded Ports collection query but expands the processor itself.
// The mock mux ignores query strings, so the processor route serves the expanded body for
// both the plain processor read and the fallback read.
func TestGPUCollector_NVLinkExpandFallback(t *testing.T) {
	server := newTestRedfishServer(t)
	systemPath := "/redfish/v1/Systems/System1"
	gpuPath := systemPath + "/Processors/GPU_0"
	server.addRoute(systemPath, map[string]any{
		"@odata.id":   systemPath,
		"@odata.type": "#ComputerSystem.v1_20_0.ComputerSystem",
		"Id":          "System1",
		"Name":        "HGX_Baseboard_0",
		"Status":      map[string]any{"State": "Enabled", "Health": "OK"},
		"Processors":  map[string]any{"@odata.id": systemPath + "/Processors"},
	})
	server.addRoute(systemPath+"/Processors", map[string]any{
		"@odata.type": "#ProcessorCollection.ProcessorCollection",
		"Members":     []map[string]any{{"@odata.id": gpuPath}},
	})
	server.addRoute(gpuPath, map[string]any{
		"@odata.id":     gpuPath,
		"@odata.type":   "#Processor.v1_20_0.Processor",
		"Id":            "GPU_0",
		"Name":          "GPU_0",
		"ProcessorType": "GPU",
		"Status":        map[string]any{"State": "Enabled", "Health": "OK"},
		"Metrics":       map[string]any{"@odata.id": gpuPath + "/ProcessorMetrics"},
		"Ports": map[string]any{
			"@odata.id":   gpuPath + "/Ports",
			"@odata.type": "#PortCollection.PortCollection",
			"Members": []map[string]any{
				{
					"@odata.id":    gpuPath + "/Ports/NVLink_0",
					"Id":           "NVLink_0",
					"PortProtocol": "NVLink",
					"PortType":     "BidirectionalPort",
					"LinkStatus":   "LinkUp",
					"Status":       map[string]any{"State": "Enabled", "Health": "OK"},
					"Metrics": map[string]any{
						"@odata.id": gpuPath + "/Ports/NVLink_0/Metrics",
						"Oem": map[string]any{
							"Nvidia": map[string]any{
								"BitErrorRate":           1e-12,
								"LinkDownedCount":        1,
								"LinkErrorRecoveryCount": 2,
								"SymbolErrors":           3,
								"NVLinkErrors":           map[string]any{"RuntimeError": false, "TrainingError": true},
							},
						},
					},
				},
				{
					"@odata.id":    gpuPath + "/Ports/PCIe_0",
					"Id":           "PCIe_0",
					"PortProtocol": "PCIe",
					"PortType":     "BidirectionalPort",
					"LinkStatus":   "LinkUp",
					"Status":       map[string]any{"State": "Enabled", "Health": "OK"},
				},
			},
		},
	})
	// No context utilization duration, so every emitted sample is a gauge for drainMetrics.
	server.addRoute(gpuPath+"/ProcessorMetrics", map[string]any{
		"@odata.id":   gpuPath + "/ProcessorMetrics",
		"@odata.type": "#ProcessorMetrics.v1_6_0.ProcessorMetrics",
		"Id":          "ProcessorMetrics",
		"Oem":         map[string]any{"Nvidia": map[string]any{"SRAMECCErrorThresholdExceeded": false}},
	})
	server.addErrorRoute(gpuPath+"/Ports", http.StatusNotFound)

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	collector, err := NewGPUCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.DefaultGPUCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 256)
	collector.CollectWithContext(context.Background(), ch)
	metrics := drainMetrics(t, ch)

	for _, name := range []string{
		"redfish_gpu_nvlink_state",
		"redfish_gpu_nvlink_health",
		"redfish_gpu_nvlink_link_status",
		"redfish_gpu_nvlink_runtime_error",
		"redfish_gpu_nvlink_training_error",
		"redfish_gpu_nvlink_link_error_recovery_count",
		"redfish_gpu_nvlink_link_downed_count",
		"redfish_gpu_nvlink_symbol_errors",
		"redfish_gpu_nvlink_bit_error_rate",
	} {
		sample := requireMetric(t, metrics, name)
		require.Equal(t, "NVLink_0", sample.labels["port_id"], name)
		require.Equal(t, "GPU_0", sample.labels["gpu_id"], name)
		require.Equal(t, "HGX_Baseboard_0", sample.labels["system_id"], name)
	}
	require.InDelta(t, 0, requireMetric(t, metrics, "redfish_gpu_nvlink_runtime_error").value, 0.01)
	require.InDelta(t, 1, requireMetric(t, metrics, "redfish_gpu_nvlink_training_error").value, 0.01)
	require.InDelta(t, 1, requireMetric(t, metrics, "redfish_gpu_nvlink_link_downed_count").value, 0.01)
	require.InDelta(t, 2, requireMetric(t, metrics, "redfish_gpu_nvlink_link_error_recovery_count").value, 0.01)
	require.InDelta(t, 3, requireMetric(t, metrics, "redfish_gpu_nvlink_symbol_errors").value, 0.01)
	require.InDelta(t, 1e-12, requireMetric(t, metrics, "redfish_gpu_nvlink_bit_error_rate").value, 1e-15)

	// The Ports collection query was attempted first and refused, then the processor expand served the data.
	require.Contains(t, server.requests, "GET "+gpuPath+"/Ports")
}

package collector

import (
	"log/slog"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/config"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGPUCollectorWithNvidiaGPU tests the GPU collector with Nvidia GPU hardware
// Note: GPU temperature and memory power metrics are now collected via TelemetryService (HGX_PlatformEnvironmentMetrics_0)
func TestGPUCollectorWithNvidiaGPU(t *testing.T) {
	t.Fail()
}

// TestGPUContextUtilization tests the collection of GPU context utilization duration metric
func TestGPUContextUtilization(t *testing.T) {
	t.Fail()
}

// TestGPUContextUtilizationWithDifferentOEMLocations tests finding the duration in different OEM locations
func TestGPUContextUtilizationWithDifferentOEMLocations(t *testing.T) {
	t.Fail()
}

// TestGPUTemperatureSensorEdgeCases tests edge cases for GPU temperature collection
// Note: GPU temperature collection is now done via TelemetryService (HGX_PlatformEnvironmentMetrics_0)
func TestGPUTemperatureSensorEdgeCases(t *testing.T) {
	t.Fail()
}

// TestCollectGPUProcessorMetrics tests collection of GPU processor metrics with various health states
func TestCollectGPUProcessorMetrics(t *testing.T) {
	t.Fail()
}

// TestGPUSerialNumberAndUUIDMetrics tests that GPU serial number and UUID info metrics are collected correctly
func TestGPUSerialNumberAndUUIDMetrics(t *testing.T) {
	t.Fail()
}

// TestGPUMetricsWithMissingSerialOrUUID tests handling of GPUs with missing serial number or UUID
func TestGPUMetricsWithMissingSerialOrUUID(t *testing.T) {
	t.Fail()
}

func Test_filterGPUs(t *testing.T) {
	tT := map[string]struct {
		processors []*redfish.Processor
		want       []*redfish.Processor
	}{
		"happy path, CPUs filtered": {
			processors: []*redfish.Processor{
				{
					ProcessorType: redfish.CPUProcessorType,
				},
				{
					ProcessorType: redfish.GPUProcessorType,
					Description:   "want this one",
				},
			},
			want: []*redfish.Processor{
				{
					ProcessorType: redfish.GPUProcessorType,
					Description:   "want this one",
				},
			},
		},
		"happy path, no GPUs returned": {
			processors: []*redfish.Processor{
				{
					ProcessorType: redfish.CPUProcessorType,
				},
				{
					ProcessorType: redfish.CPUProcessorType,
				},
			},
			want: []*redfish.Processor{},
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			got := filterGPUs(test.processors)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGPUCollector_gatherGPUs(t *testing.T) {
	tests := map[string]struct {
		want         []SystemGPU
		wantErr      bool
		wantErrMsg   string
		rfRoutes     map[string]any
		rfFileRoutes map[string]string
	}{
		"happy path": {
			want: []SystemGPU{
				{
					SystemID:   "HGX_Baseboard_0",
					SystemName: "HGX_Baseboard_0",
					Processor: &redfish.Processor{
						Entity: common.Entity{
							ID:      "GPU_0",
							ODataID: "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0",
						},
						ProcessorType: redfish.GPUProcessorType,
					},
				},
			},
			wantErr:    false,
			wantErrMsg: "",
			rfRoutes: map[string]any{
				"/redfish/v1/Systems": map[string]any{
					"@odata.type": "#ComputerSystemCollection.ComputerSystemCollection",
					"Members": []map[string]string{
						{"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0"},
					},
				},
				"/redfish/v1/Systems/HGX_Baseboard_0": map[string]any{
					"@odata.id":   "/redfish/v1/Systems",
					"@odata.type": "#ComputerSystem.v1_22_0.ComputerSystem",
					"Id":          "HGX_Baseboard_0",
					"Name":        "HGX_Baseboard_0",
					"Processors": map[string]string{
						"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors",
					},
				},
				"/redfish/v1/Systems/HGX_Baseboard_0/Processors": map[string]any{
					"@odata.id":   "/redfish/v1/Systems/HGX_Baseboard_0/Processors",
					"@odata.type": "#ProcessorCollection.ProcessorCollection",
					"Members": []map[string]string{
						{
							"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0",
						},
						{
							"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors/CPU_1",
						},
					},
				},
				"/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0": map[string]any{
					"@odata.id":     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0",
					"Id":            "GPU_0",
					"ProcessorType": "GPU",
				},
				"/redfish/v1/Systems/HGX_Baseboard_0/Processors/CPU_0": map[string]any{
					"@odata.id":     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/CPU_0",
					"Id":            "CPU_0",
					"ProcessorType": "CPU",
				},
			},
			rfFileRoutes: map[string]string{
				"/redfish/v1/": "service_root.json",
			},
		},
	}
	for tName, test := range tests {
		t.Run(tName, func(t *testing.T) {
			_, client := testServer(t, test.rfRoutes, test.rfFileRoutes)
			collector, err := NewGPUCollector(t.Name(), client, slog.Default(), config.DefaultGPUCollector)
			require.NoError(t, err)
			got, gotErr := collector.gatherGPUs(t.Context())
			if test.wantErr {
				require.Error(t, gotErr)
				require.Contains(t, gotErr.Error(), test.wantErrMsg)
			} else {
				require.NoError(t, gotErr)
			}
			assert.Empty(t, cmp.Diff(test.want, got,
				cmpopts.IgnoreUnexported(common.Entity{}),
				cmpopts.IgnoreUnexported(redfish.Processor{}),
				cmpopts.IgnoreUnexported(redfish.MemorySummary{}),
			))
		})
	}
}

}

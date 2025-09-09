package collector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectSystemGPUMetrics(t *testing.T) {
	tT := map[string]struct {
		gpu           *redfish.Processor
		systemName    string
		expectMetric  bool
		expectedValue float64
	}{
		"happy path, gpu healthy": {
			gpu: &redfish.Processor{
				Entity: common.Entity{
					ID:   "GPU_0",
					Name: "testgpu",
				},
				Model:         "NVIDIA GB300",
				ProcessorType: redfish.GPUProcessorType,
				Status: common.Status{
					Health:       "OK",
					HealthRollup: "OK",
					State:        "Enabled",
				},
			},
			systemName:    "test",
			expectMetric:  true,
			expectedValue: 1,
		},
		"gpu with critical status": {
			gpu: &redfish.Processor{
				Entity: common.Entity{
					ID:   "GPU_1",
					Name: "testgpu1",
				},
				Model:         "NVIDIA GB300",
				ProcessorType: redfish.GPUProcessorType,
				Status: common.Status{
					Health:       "Critical",
					HealthRollup: "Critical",
					State:        "Enabled",
				},
			},
			systemName:    "test",
			expectMetric:  true,
			expectedValue: 3,
		},
		"cpu and should not emit metrics": {
			gpu: &redfish.Processor{
				ProcessorType: redfish.CPUProcessorType,
			},
			systemName:   "test",
			expectMetric: false,
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			outCh := make(chan prometheus.Metric, 1)
			emitGPUHealth(outCh, test.gpu, []string{test.gpu.Name, test.systemName, test.gpu.ID})

			select {
			case metric := <-outCh:
				assert.True(t, test.expectMetric, "found metric when not expecting one")

				dtoMetric := &dto.Metric{}
				require.NoError(t, metric.Write(dtoMetric), "unexpected error writing DTO metric")
				requireGaugeWithValue(t, dtoMetric, test.expectedValue)
				requireMetricDescContains(t, metric, "gpu_health")
			default:
				if test.expectMetric {
					t.Errorf("Expected metric to be emitted, but none was received")
				}
			}
		})
	}
}

func requireGaugeWithValue(t *testing.T, metric *dto.Metric, expected float64) {
	t.Helper()
	require.NotNil(t, metric.Gauge, "required a gauge")
	require.Equal(t, expected, *metric.Gauge.Value)
}

func requireMetricDescContains(t *testing.T, m prometheus.Metric, contains string) {
	t.Helper()
	desc := m.Desc()
	assert.NotNil(t, desc)
	require.Contains(t, desc.String(), contains)
}

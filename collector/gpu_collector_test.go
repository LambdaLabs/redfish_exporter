package collector

import (
	"errors"
	"log/slog"
	"strings"
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

// mockMemoryWithMetrics implements MemoryWithMetrics interface for testing
type mockMemoryWithMetrics struct {
	id                         string
	correctableECCErrorCount   int
	uncorrectableECCErrorCount int
	shouldError                bool
}

func (m *mockMemoryWithMetrics) GetID() string {
	return m.id
}

func (m *mockMemoryWithMetrics) Metrics() (*redfish.MemoryMetrics, error) {
	if m.shouldError {
		return nil, errors.New("metrics retrieval failed")
	}

	return &redfish.MemoryMetrics{
		CurrentPeriod: redfish.CurrentPeriod{
			CorrectableECCErrorCount:   m.correctableECCErrorCount,
			UncorrectableECCErrorCount: m.uncorrectableECCErrorCount,
		},
	}, nil
}

func TestEmitGPUECCMetrics(t *testing.T) {
	tT := map[string]struct {
		memories             []MemoryWithMetrics
		systemName           string
		expectedMetricCount  int
		expectedMetricChecks []struct {
			nameContains  string
			expectedValue float64
		}
	}{
		"happy path": {
			memories: []MemoryWithMetrics{
				&mockMemoryWithMetrics{
					id:                         "mockMem",
					correctableECCErrorCount:   100,
					uncorrectableECCErrorCount: 0,
				},
			},
			systemName:          "test",
			expectedMetricCount: 2,
			expectedMetricChecks: []struct {
				nameContains  string
				expectedValue float64
			}{
				{nameContains: "memory_ecc_correctable", expectedValue: 100},
				{nameContains: "memory_ecc_uncorrectable", expectedValue: 0},
			},
		},
		// should simply error from within functon-under-test, no returned metrics
		"no memory metrics": {
			memories: []MemoryWithMetrics{
				&mockMemoryWithMetrics{
					id:          "mockMem",
					shouldError: true,
				},
			},
			systemName:          "test",
			expectedMetricCount: 0,
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			outCh := make(chan prometheus.Metric, 100)
			go func() {
				emitGPUECCMetrics(outCh, test.memories, slog.Default(), []string{"testGPU", test.systemName, "testGPUId"})
				close(outCh)
			}()

			var metrics []prometheus.Metric
			for metric := range outCh {
				metrics = append(metrics, metric)
			}

			require.Equal(t, test.expectedMetricCount, len(metrics), "unexpected number of metrics")

			for _, check := range test.expectedMetricChecks {
				found := false
				for _, metric := range metrics {
					if descContains(metric, check.nameContains) {
						found = true
						dtoMetric := &dto.Metric{}
						require.NoError(t, metric.Write(dtoMetric), "unexpected error writing DTO metric")
						requireCounterWithValue(t, dtoMetric, check.expectedValue)
						break
					}
				}
				assert.True(t, found, "expected metric containing '%s' not found", check.nameContains)
			}
		})
	}
}

func requireGaugeWithValue(t *testing.T, metric *dto.Metric, expected float64) {
	t.Helper()
	require.NotNil(t, metric.Gauge, "required a gauge")
	require.Equal(t, expected, *metric.Gauge.Value)
}

func requireCounterWithValue(t *testing.T, metric *dto.Metric, expected float64) {
	t.Helper()
	require.NotNil(t, metric.Counter, "required a counter")
	require.Equal(t, expected, *metric.Counter.Value)
}

func requireMetricDescContains(t *testing.T, m prometheus.Metric, contains string) {
	t.Helper()
	desc := m.Desc()
	assert.NotNil(t, desc)
	require.Contains(t, desc.String(), contains)
}

func descContains(m prometheus.Metric, contains string) bool {
	desc := m.Desc()
	if desc == nil {
		return false
	}
	return strings.Contains(desc.String(), contains)
}

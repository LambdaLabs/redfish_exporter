package collector

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
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
			emitGPUHealth(outCh, test.systemName, test.gpu)

			select {
			case metric := <-outCh:
				if !test.expectMetric {
					t.Errorf("Expected no metric, but got one")
				}
				dtoMetric := &dto.Metric{}
				err := metric.Write(dtoMetric)
				if err != nil {
					t.Fatalf("Failed to write metric to DTO: %v", err)
				}
				if dtoMetric.Gauge == nil {
					t.Errorf("Expected gauge metric, got nil")
				} else if *dtoMetric.Gauge.Value != test.expectedValue {
					t.Errorf("Expected value %f, got %f", test.expectedValue, *dtoMetric.Gauge.Value)
				}
				desc := metric.Desc()
				if desc != nil {
					descString := desc.String()
					if !strings.Contains(descString, "gpu_health") {
						t.Errorf("Expected metric name to contain 'gpu_health', got: %s", descString)
					}
				}
			default:
				if test.expectMetric {
					t.Errorf("Expected metric to be emitted, but none was received")
				}
			}
		})
	}
}

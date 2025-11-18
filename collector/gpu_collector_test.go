package collector

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/config"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"
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
		want        []SystemGPU
		wantErr     bool
		wantErrMsg  string
		testdataDir string
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
						ODataType:     "#Processor.v1_20_0.Processor",
						ProcessorType: redfish.GPUProcessorType,
						UUID:          "uuid-here",
					},
				},
			},
			wantErr:     false,
			wantErrMsg:  "",
			testdataDir: "testdata/gathergpus_happypath",
		},
	}
	for tName, test := range tests {
		t.Run(tName, func(t *testing.T) {
			_, client := setupTestServerClient(t, test.testdataDir)
			logger := NewTestLogger(t, slog.LevelDebug)
			collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
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

func TestGPUCollector_emitGPUMemoryMetrics(t *testing.T) {
	tT := map[string]struct {
		testdataPath     string
		seriesToCheck    []string
		testLogLevel     slog.Level
		wantSeriesCount  int
		wantSeriesString string
	}{
		"redfish_gpu_memory_state": {
			testdataPath:    "testdata/gb300_happypath",
			seriesToCheck:   []string{"redfish_gpu_memory_state"},
			testLogLevel:    slog.LevelInfo,
			wantSeriesCount: 4,
			wantSeriesString: `
# HELP redfish_gpu_memory_state GPU memory state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_gpu_memory_state gauge
redfish_gpu_memory_state{gpu_id="GPU_0",memory_id="GPU_0_DRAM_0",system_id="HGX_Baseboard_0"} 1
redfish_gpu_memory_state{gpu_id="GPU_1",memory_id="GPU_1_DRAM_0",system_id="HGX_Baseboard_0"} 1
redfish_gpu_memory_state{gpu_id="GPU_2",memory_id="GPU_2_DRAM_0",system_id="HGX_Baseboard_0"} 1
redfish_gpu_memory_state{gpu_id="GPU_3",memory_id="GPU_3_DRAM_0",system_id="HGX_Baseboard_0"} 1
`,
		},
		"redfish_gpu_memory_uncorrectable_row_remapping_count": {
			testdataPath:    "testdata/gb300_happypath",
			seriesToCheck:   []string{"redfish_gpu_memory_uncorrectable_row_remapping_count"},
			testLogLevel:    slog.LevelDebug,
			wantSeriesCount: 4,
			wantSeriesString: `
# HELP redfish_gpu_memory_uncorrectable_row_remapping_count GPU memory uncorrectable row remapping count
# TYPE redfish_gpu_memory_uncorrectable_row_remapping_count gauge
redfish_gpu_memory_uncorrectable_row_remapping_count{gpu_id="GPU_0",memory_id="GPU_0_DRAM_0",system_id="HGX_Baseboard_0"} 1
redfish_gpu_memory_uncorrectable_row_remapping_count{gpu_id="GPU_1",memory_id="GPU_1_DRAM_0",system_id="HGX_Baseboard_0"} 0
redfish_gpu_memory_uncorrectable_row_remapping_count{gpu_id="GPU_2",memory_id="GPU_2_DRAM_0",system_id="HGX_Baseboard_0"} 0
redfish_gpu_memory_uncorrectable_row_remapping_count{gpu_id="GPU_3",memory_id="GPU_3_DRAM_0",system_id="HGX_Baseboard_0"} 0
`,
		},
		"redfish_gpu_memory_row_remapping_pending": {
			testdataPath:    "testdata/gb300_happypath",
			seriesToCheck:   []string{"redfish_gpu_memory_row_remapping_pending"},
			testLogLevel:    slog.LevelDebug,
			wantSeriesCount: 4,
			wantSeriesString: `
# TYPE redfish_gpu_memory_row_remapping_pending gauge
# HELP redfish_gpu_memory_row_remapping_pending GPU memory row remapping pending status (1 if pending)
redfish_gpu_memory_row_remapping_pending{gpu_id="GPU_0",memory_id="GPU_0_DRAM_0",system_id="HGX_Baseboard_0"} 0
redfish_gpu_memory_row_remapping_pending{gpu_id="GPU_1",memory_id="GPU_1_DRAM_0",system_id="HGX_Baseboard_0"} 1
redfish_gpu_memory_row_remapping_pending{gpu_id="GPU_2",memory_id="GPU_2_DRAM_0",system_id="HGX_Baseboard_0"} 0
redfish_gpu_memory_row_remapping_pending{gpu_id="GPU_3",memory_id="GPU_3_DRAM_0",system_id="HGX_Baseboard_0"} 0
`,
		},
		"redfish_gpu_memory_ecc_correctable": {
			testdataPath:    "testdata/gb300_happypath",
			seriesToCheck:   []string{"redfish_gpu_memory_ecc_correctable"},
			testLogLevel:    slog.LevelDebug,
			wantSeriesCount: 4,
			wantSeriesString: `
# HELP redfish_gpu_memory_ecc_correctable current correctable memory ecc errors reported on the gpu
# TYPE redfish_gpu_memory_ecc_correctable counter
redfish_gpu_memory_ecc_correctable{gpu_id="GPU_0",memory_id="GPU_0_DRAM_0",system_id="HGX_Baseboard_0"} 0
redfish_gpu_memory_ecc_correctable{gpu_id="GPU_1",memory_id="GPU_1_DRAM_0",system_id="HGX_Baseboard_0"} 100
redfish_gpu_memory_ecc_correctable{gpu_id="GPU_2",memory_id="GPU_2_DRAM_0",system_id="HGX_Baseboard_0"} 0
redfish_gpu_memory_ecc_correctable{gpu_id="GPU_3",memory_id="GPU_3_DRAM_0",system_id="HGX_Baseboard_0"} 0
`,
		},
	}
	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			_, client := setupTestServerClient(t, test.testdataPath)
			logger := NewTestLogger(t, test.testLogLevel)
			collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
			require.NoError(t, err)
			assert.Equal(t, test.wantSeriesCount, testutil.CollectAndCount(collector, test.seriesToCheck...))
			wanted := strings.NewReader(test.wantSeriesString)
			if test.wantSeriesString != "" {
				assert.NoError(t, testutil.CollectAndCompare(collector, wanted, test.seriesToCheck...))
			}
		})
	}
}

func TestGPUCollector_emitHealthInfo(t *testing.T) {
	tT := map[string]struct {
		testdataPath     string
		seriesToCheck    []string
		testLogLevel     slog.Level
		wantSeriesCount  int
		wantSeriesString string
	}{
		"gpu info/health/state": {
			testdataPath:    "testdata/gb300_happypath",
			seriesToCheck:   []string{"redfish_gpu_info", "redfish_gpu_health", "redfish_gpu_state"},
			testLogLevel:    slog.LevelDebug,
			wantSeriesCount: 12,
			wantSeriesString: `
# HELP redfish_gpu_health GPU processor health,1(OK),2(Warning),3(Critical)
# TYPE redfish_gpu_health gauge
redfish_gpu_health{gpu_id="GPU_0",system_id="HGX_Baseboard_0"} 1
redfish_gpu_health{gpu_id="GPU_1",system_id="HGX_Baseboard_0"} 1
redfish_gpu_health{gpu_id="GPU_2",system_id="HGX_Baseboard_0"} 1
redfish_gpu_health{gpu_id="GPU_3",system_id="HGX_Baseboard_0"} 2
# HELP redfish_gpu_info GPU information with serial number and UUID
# TYPE redfish_gpu_info gauge
redfish_gpu_info{gpu_id="GPU_0",serial_number="123456",system_id="HGX_Baseboard_0",uuid="gpu-0-uuid"} 1
redfish_gpu_info{gpu_id="GPU_1",serial_number="234567",system_id="HGX_Baseboard_0",uuid="gpu-1-uuid"} 1
redfish_gpu_info{gpu_id="GPU_2",serial_number="345678",system_id="HGX_Baseboard_0",uuid="gpu-2-uuid"} 1
redfish_gpu_info{gpu_id="GPU_3",serial_number="456789",system_id="HGX_Baseboard_0",uuid="gpu-3-uuid"} 1
# HELP redfish_gpu_state GPU processor state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_gpu_state gauge
redfish_gpu_state{gpu_id="GPU_0",system_id="HGX_Baseboard_0"} 1
redfish_gpu_state{gpu_id="GPU_1",system_id="HGX_Baseboard_0"} 1
redfish_gpu_state{gpu_id="GPU_2",system_id="HGX_Baseboard_0"} 1
redfish_gpu_state{gpu_id="GPU_3",system_id="HGX_Baseboard_0"} 5
`,
		},
		"special case for gpu info, unknown everything": {
			testdataPath:    "testdata/gpu_info_unknown_multi",
			seriesToCheck:   []string{"redfish_gpu_info"},
			testLogLevel:    9, // Disable even error logging for this test, the testdata dir cuts many corners
			wantSeriesCount: 2,
			wantSeriesString: `
# HELP redfish_gpu_info GPU information with serial number and UUID
# TYPE redfish_gpu_info gauge
redfish_gpu_info{gpu_id="GPU_SXM_1",serial_number="unknown",system_id="HGX_Baseboard_0",uuid="unknown"} 1
redfish_gpu_info{gpu_id="GPU_SXM_2",serial_number="unknown",system_id="HGX_Baseboard_0",uuid="unknown"} 1
`,
		},
	}
	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			_, client := setupTestServerClient(t, test.testdataPath)
			logger := NewTestLogger(t, test.testLogLevel)
			collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
			require.NoError(t, err)
			assert.Equal(t, test.wantSeriesCount, testutil.CollectAndCount(collector, test.seriesToCheck...))
			wanted := strings.NewReader(test.wantSeriesString)
			if test.wantSeriesString != "" {
				assert.NoError(t, testutil.CollectAndCompare(collector, wanted, test.seriesToCheck...))
			}
		})
	}
}

func TestGPUCollector_emitGPUOem(t *testing.T) {
	tT := map[string]struct {
		testdataPath     string
		seriesToCheck    []string
		testLogLevel     slog.Level
		wantSeriesCount  int
		wantSeriesString string
	}{
		"happy path": {
			testdataPath:    "testdata/gb300_happypath",
			seriesToCheck:   []string{"redfish_gpu_context_utilization_seconds_total", "redfish_gpu_sram_ecc_error_threshold_exceeded"},
			testLogLevel:    slog.LevelDebug,
			wantSeriesCount: 8,
			wantSeriesString: `
# HELP redfish_gpu_context_utilization_seconds_total Accumulated GPU context utilization duration in seconds
# TYPE redfish_gpu_context_utilization_seconds_total counter
redfish_gpu_context_utilization_seconds_total{gpu_id="GPU_0",system_id="HGX_Baseboard_0"} 60
redfish_gpu_context_utilization_seconds_total{gpu_id="GPU_1",system_id="HGX_Baseboard_0"} 0
redfish_gpu_context_utilization_seconds_total{gpu_id="GPU_2",system_id="HGX_Baseboard_0"} 0
redfish_gpu_context_utilization_seconds_total{gpu_id="GPU_3",system_id="HGX_Baseboard_0"} 0
# HELP redfish_gpu_sram_ecc_error_threshold_exceeded GPU SRAM ECC error threshold exceeded (1 if exceeded)
# TYPE redfish_gpu_sram_ecc_error_threshold_exceeded gauge
redfish_gpu_sram_ecc_error_threshold_exceeded{gpu_id="GPU_0",system_id="HGX_Baseboard_0"} 0
redfish_gpu_sram_ecc_error_threshold_exceeded{gpu_id="GPU_1",system_id="HGX_Baseboard_0"} 0
redfish_gpu_sram_ecc_error_threshold_exceeded{gpu_id="GPU_2",system_id="HGX_Baseboard_0"} 1
redfish_gpu_sram_ecc_error_threshold_exceeded{gpu_id="GPU_3",system_id="HGX_Baseboard_0"} 0
`,
		},
	}
	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			_, client := setupTestServerClient(t, test.testdataPath)
			logger := NewTestLogger(t, test.testLogLevel)
			collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
			require.NoError(t, err)
			assert.Equal(t, test.wantSeriesCount, testutil.CollectAndCount(collector, test.seriesToCheck...))
			wanted := strings.NewReader(test.wantSeriesString)
			if test.wantSeriesString != "" {
				assert.NoError(t, testutil.CollectAndCompare(collector, wanted, test.seriesToCheck...))
			}
		})
	}
}

func TestGPUCollector_emitGPUNVLinkTelemetry(t *testing.T) {
	tT := map[string]struct {
		testdataPath         string
		seriesToCheck        []string
		testLogLevel         slog.Level
		wantSeriesCount      int
		wantSeriesGoldenPath string // NVLink telemetry is a lot, so use golden files instead
	}{
		"happy path": {
			testdataPath: "testdata/gb300_happypath",
			seriesToCheck: []string{"redfish_gpu_nvlink_state",
				"redfish_gpu_nvlink_health",
				"redfish_gpu_nvlink_runtime_error",
				"redfish_gpu_nvlink_training_error",
				"redfish_gpu_nvlink_link_error_recovery_count",
				"redfish_gpu_nvlink_link_downed_count",
				"redfish_gpu_nvlink_symbol_errors",
				"redfish_gpu_nvlink_bit_error_rate",
			},
			testLogLevel:         slog.LevelDebug,
			wantSeriesCount:      576,
			wantSeriesGoldenPath: "golden/gb300_nvlink_happy.golden",
		},
	}
	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			_, client := setupTestServerClient(t, test.testdataPath)
			logger := NewTestLogger(t, test.testLogLevel)
			collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
			require.NoError(t, err)
			assert.Equal(t, test.wantSeriesCount, testutil.CollectAndCount(collector, test.seriesToCheck...))
			if test.wantSeriesGoldenPath != "" {
				wanted := golden.Get(t, test.wantSeriesGoldenPath)
				assert.NoError(t, testutil.CollectAndCompare(collector, bytes.NewReader(wanted), test.seriesToCheck...))
			}
		})
	}
}

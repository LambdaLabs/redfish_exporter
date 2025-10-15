package collector

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	isoDuration "github.com/sosodev/duration"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

const TelemetrySubsystem = "telemetry"

var (
	telemetryBaseLabels       = []string{"hostname", "system_id", "gpu_id"}
	telemetryMemoryLabels     = []string{"hostname", "system_id", "gpu_id", "memory_id"}
	telemetryPortLabels       = []string{"hostname", "system_id", "gpu_id", "port_id"}
	telemetryMetrics          = createTelemetryMetricMap()
)

func createTelemetryMetricMap() map[string]Metric {
	metrics := make(map[string]Metric)

	// Cache ECC errors (L2/SRAM) - different from memory DRAM ECC
	addToMetricMap(metrics, TelemetrySubsystem, "cache_ecc_correctable_total", "Total correctable ECC errors in GPU cache (L2/SRAM)", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "cache_ecc_uncorrectable_total", "Total uncorrectable ECC errors in GPU cache (L2/SRAM)", telemetryBaseLabels)

	// PCIe error counters
	addToMetricMap(metrics, TelemetrySubsystem, "pcie_correctable_errors_total", "Total PCIe correctable errors", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "pcie_nonfatal_errors_total", "Total PCIe non-fatal errors", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "pcie_fatal_errors_total", "Total PCIe fatal errors", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "pcie_l0_to_recovery_total", "Total PCIe L0 to recovery transitions", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "pcie_replay_total", "Total PCIe replay events", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "pcie_replay_rollover_total", "Total PCIe replay rollover events", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "pcie_nak_sent_total", "Total PCIe NAK sent", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "pcie_nak_received_total", "Total PCIe NAK received", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "pcie_unsupported_request_total", "Total PCIe unsupported requests", telemetryBaseLabels)

	// Throttling durations
	addToMetricMap(metrics, TelemetrySubsystem, "power_throttle_duration_seconds_total", "Total time GPU was throttled due to power limits", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "thermal_throttle_duration_seconds_total", "Total time GPU was throttled due to thermal limits", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "hardware_violation_throttle_duration_seconds_total", "Total time GPU was throttled due to hardware violations", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "software_violation_throttle_duration_seconds_total", "Total time GPU was throttled due to software violations", telemetryBaseLabels)

	// Memory metrics from HGX_MemoryMetrics_0
	addToMetricMap(metrics, TelemetrySubsystem, "memory_ecc_correctable_lifetime_total", "Lifetime correctable DRAM ECC errors", telemetryMemoryLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "memory_ecc_uncorrectable_lifetime_total", "Lifetime uncorrectable DRAM ECC errors", telemetryMemoryLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "memory_bandwidth_percent", "Memory bandwidth utilization percentage", telemetryMemoryLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "memory_capacity_utilization_percent", "Memory capacity utilization percentage", telemetryMemoryLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "memory_operating_speed_mhz", "Memory operating speed in MHz", telemetryMemoryLabels)

	// Reset metrics from HGX_ProcessorResetMetrics_0
	addToMetricMap(metrics, TelemetrySubsystem, "conventional_reset_entry_total", "Total conventional reset entry events", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "conventional_reset_exit_total", "Total conventional reset exit events", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "fundamental_reset_entry_total", "Total fundamental reset entry events", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "fundamental_reset_exit_total", "Total fundamental reset exit events", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "irot_reset_exit_total", "Total IRoT (Internal Root of Trust) reset exit events", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "pf_flr_reset_entry_total", "Total PF FLR (Physical Function Function-Level Reset) entry events", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "pf_flr_reset_exit_total", "Total PF FLR (Physical Function Function-Level Reset) exit events", telemetryBaseLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "last_reset_type_info", "Last reset type (1=Conventional, 2=Fundamental, 3=IRoT, 4=PF_FLR)", telemetryBaseLabels)

	// Port metrics from HGX_ProcessorPortMetrics_0 - NVLink and PCIe
	// Link state and speed
	addToMetricMap(metrics, TelemetrySubsystem, "port_current_speed_gbps", "Current port link speed in Gbps", telemetryPortLabels)

	// Standard networking metrics
	addToMetricMap(metrics, TelemetrySubsystem, "port_rx_bytes_total", "Total bytes received on port", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_tx_bytes_total", "Total bytes transmitted on port", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_rx_errors_total", "Total receive errors on port", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_rx_frames_total", "Total frames received on port", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_tx_frames_total", "Total frames transmitted on port", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_tx_discards_total", "Total transmit discards on port", telemetryPortLabels)

	// OEM metrics - link reliability
	addToMetricMap(metrics, TelemetrySubsystem, "port_intentional_link_down_count_total", "Total intentional link down events", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_unintentional_link_down_count_total", "Total unintentional link down events", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_link_down_reason_code", "Last link down reason code", telemetryPortLabels)

	// OEM metrics - error counters
	addToMetricMap(metrics, TelemetrySubsystem, "port_neighbor_mtu_discards_total", "Total neighbor MTU discards", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_qp1_dropped_total", "Total QP1 packets dropped", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_rx_remote_physical_errors_total", "Total RX remote physical errors", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_rx_switch_relay_errors_total", "Total RX switch relay errors", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_vl15_dropped_total", "Total VL15 packets dropped", telemetryPortLabels)

	// OEM metrics - no-protocol bytes (non-data traffic)
	addToMetricMap(metrics, TelemetrySubsystem, "port_rx_no_protocol_bytes_total", "Total RX bytes without protocol", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_tx_no_protocol_bytes_total", "Total TX bytes without protocol", telemetryPortLabels)

	// OEM metrics - VL15 (management traffic)
	addToMetricMap(metrics, TelemetrySubsystem, "port_vl15_tx_bytes_total", "Total VL15 bytes transmitted", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_vl15_tx_packets_total", "Total VL15 packets transmitted", telemetryPortLabels)

	// OEM metrics - link width and wait
	addToMetricMap(metrics, TelemetrySubsystem, "port_rx_width", "Current receive link width", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_tx_width", "Current transmit link width", telemetryPortLabels)
	addToMetricMap(metrics, TelemetrySubsystem, "port_tx_wait_total", "Total TX wait time", telemetryPortLabels)

	// OEM metrics - bit error rate
	addToMetricMap(metrics, TelemetrySubsystem, "port_total_raw_ber", "Total raw bit error rate", telemetryPortLabels)

	return metrics
}

// TelemetryCollector collects metrics from Redfish TelemetryService
type TelemetryCollector struct {
	redfishClient         *gofish.APIClient
	metrics               map[string]Metric
	logger                *slog.Logger
	collectorScrapeStatus *prometheus.GaugeVec
}

// NewTelemetryCollector creates a new TelemetryService collector
func NewTelemetryCollector(redfishClient *gofish.APIClient, logger *slog.Logger) *TelemetryCollector {
	return &TelemetryCollector{
		redfishClient: redfishClient,
		metrics:       telemetryMetrics,
		logger:        logger.With(slog.String("collector", "TelemetryCollector")),
		collectorScrapeStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "collector_scrape_status",
				Help:      "collector_scrape_status",
			},
			[]string{"collector"},
		),
	}
}

// Describe implements prometheus.Collector
func (t *TelemetryCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range t.metrics {
		ch <- metric.desc
	}
	t.collectorScrapeStatus.Describe(ch)
}

// Collect implements prometheus.Collector
func (t *TelemetryCollector) Collect(ch chan<- prometheus.Metric) {
	t.collectorScrapeStatus.WithLabelValues("telemetry").Set(float64(0))

	service := t.redfishClient.Service

	// Get TelemetryService
	telemetryService, err := service.TelemetryService()
	if err != nil {
		t.logger.Debug("failed to get telemetry service - may not be supported",
			slog.Any("error", err),
		)
		return
	}

	// Get all metric reports
	metricReports, err := telemetryService.MetricReports()
	if err != nil {
		// MetricReports may not be available on all TelemetryService implementations
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			t.logger.Debug("metric reports not available - may not be supported",
				slog.Any("error", err),
			)
		} else {
			t.logger.Error("failed to get metric reports",
				slog.Any("error", err),
			)
		}
		return
	}

	// Get systems for hostname mapping
	systems, err := service.Systems()
	if err != nil {
		t.logger.Error("failed to get systems",
			slog.Any("error", err),
		)
		return
	}

	// Create system ID to name mapping
	systemMap := make(map[string]string)
	for _, sys := range systems {
		systemMap[sys.ID] = sys.Name
		if systemMap[sys.ID] == "" {
			systemMap[sys.ID] = sys.ID
		}
	}

	// Process each metric report
	wg := &sync.WaitGroup{}
	for _, report := range metricReports {
		if strings.Contains(report.ID, "HGX_ProcessorMetrics") && !strings.Contains(report.ID, "HGX_ProcessorResetMetrics") {
			wg.Add(1)
			go t.collectProcessorMetrics(ch, report, systemMap, wg)
		} else if strings.Contains(report.ID, "HGX_MemoryMetrics") {
			wg.Add(1)
			go t.collectMemoryMetrics(ch, report, systemMap, wg)
		} else if strings.Contains(report.ID, "HGX_ProcessorResetMetrics") {
			wg.Add(1)
			go t.collectResetMetrics(ch, report, systemMap, wg)
		} else if strings.Contains(report.ID, "HGX_ProcessorPortMetrics") {
			wg.Add(1)
			go t.collectPortMetrics(ch, report, systemMap, wg)
		}
	}

	wg.Wait()
	t.collectorScrapeStatus.WithLabelValues("telemetry").Set(float64(1))
}

// collectProcessorMetrics processes a single HGX_ProcessorMetrics report
func (t *TelemetryCollector) collectProcessorMetrics(ch chan<- prometheus.Metric, report *redfish.MetricReport, systemMap map[string]string, wg *sync.WaitGroup) {
	defer wg.Done()

	t.logger.Debug("processing metric report",
		slog.String("report_id", report.ID),
		slog.Int("metric_count", len(report.MetricValues)),
	)

	// Group metrics by GPU ID
	metricsByGPU := make(map[string]map[string]float64)

	for _, metricValue := range report.MetricValues {
		// Extract GPU ID and metric name from MetricProperty
		// Format: /redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_1/ProcessorMetrics#/...
		gpuID, metricName := parseMetricProperty(metricValue.MetricProperty)
		if gpuID == "" || metricName == "" {
			continue
		}

		// Parse metric value
		value, err := parseMetricValue(metricValue.MetricValue)
		if err != nil {
			t.logger.Debug("failed to parse metric value",
				slog.String("metric", metricName),
				slog.String("value", metricValue.MetricValue),
				slog.Any("error", err),
			)
			continue
		}

		// Initialize GPU map if needed
		if metricsByGPU[gpuID] == nil {
			metricsByGPU[gpuID] = make(map[string]float64)
		}
		metricsByGPU[gpuID][metricName] = value
	}

	// Extract system ID from metric properties (all should be same system)
	systemID := extractSystemIDFromReport(report)
	systemName := systemMap[systemID]
	if systemName == "" {
		systemName = systemID
	}

	// Emit metrics for each GPU
	for gpuID, metrics := range metricsByGPU {
		labels := []string{systemName, systemID, gpuID}
		t.emitGPUMetrics(ch, labels, metrics)
	}
}

// parseMetricProperty extracts GPU ID and metric name from a MetricProperty path
// Example: /redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_1/ProcessorMetrics#/CacheMetricsTotal/LifeTime/CorrectableECCErrorCount
// Returns: "GPU_SXM_1", "CacheMetricsTotal/LifeTime/CorrectableECCErrorCount"
func parseMetricProperty(property string) (gpuID string, metricName string) {
	// Split on '#' to separate resource path from property path
	parts := strings.Split(property, "#/")
	if len(parts) != 2 {
		return "", ""
	}

	resourcePath := parts[0]
	metricName = parts[1]

	// Extract GPU ID from resource path
	// Look for pattern: /Processors/GPU_SXM_X/
	if idx := strings.Index(resourcePath, "/Processors/"); idx != -1 {
		remainder := resourcePath[idx+len("/Processors/"):]
		if endIdx := strings.Index(remainder, "/"); endIdx != -1 {
			gpuID = remainder[:endIdx]
		}
	}

	return gpuID, metricName
}

// parseMetricValue converts a metric value string to float64
// Handles booleans, numeric values, and ISO 8601 durations
func parseMetricValue(value string) (float64, error) {
	// Handle boolean values
	if value == "true" {
		return 1.0, nil
	}
	if value == "false" {
		return 0.0, nil
	}

	// Check if it's an ISO 8601 duration (starts with "P")
	if strings.HasPrefix(value, "P") || strings.HasPrefix(value, "p") {
		duration, err := isoDuration.Parse(value)
		if err != nil {
			// If duration parsing fails, try as float
			return strconv.ParseFloat(value, 64)
		}
		// Convert to seconds
		return duration.ToTimeDuration().Seconds(), nil
	}

	// Parse as float
	return strconv.ParseFloat(value, 64)
}

// extractSystemIDFromReport extracts the system ID from the first metric property in a report
func extractSystemIDFromReport(report *redfish.MetricReport) string {
	if len(report.MetricValues) == 0 {
		return ""
	}

	// Example: /redfish/v1/Systems/HGX_Baseboard_0/Processors/...
	property := report.MetricValues[0].MetricProperty
	if idx := strings.Index(property, "/Systems/"); idx != -1 {
		remainder := property[idx+len("/Systems/"):]
		if endIdx := strings.Index(remainder, "/"); endIdx != -1 {
			return remainder[:endIdx]
		}
	}

	return ""
}

// emitGPUMetrics emits Prometheus metrics for a single GPU
func (t *TelemetryCollector) emitGPUMetrics(ch chan<- prometheus.Metric, labels []string, metrics map[string]float64) {
	// Cache ECC errors
	if val, ok := metrics["CacheMetricsTotal/LifeTime/CorrectableECCErrorCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_cache_ecc_correctable_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["CacheMetricsTotal/LifeTime/UncorrectableECCErrorCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_cache_ecc_uncorrectable_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// PCIe errors
	if val, ok := metrics["PCIeErrors/CorrectableErrorCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pcie_correctable_errors_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["PCIeErrors/NonFatalErrorCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pcie_nonfatal_errors_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["PCIeErrors/FatalErrorCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pcie_fatal_errors_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["PCIeErrors/L0ToRecoveryCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pcie_l0_to_recovery_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["PCIeErrors/ReplayCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pcie_replay_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["PCIeErrors/ReplayRolloverCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pcie_replay_rollover_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["PCIeErrors/NAKSentCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pcie_nak_sent_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["PCIeErrors/NAKReceivedCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pcie_nak_received_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["PCIeErrors/UnsupportedRequestCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pcie_unsupported_request_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// Throttling durations - parseMetricValue handles ISO 8601 duration conversion
	if val, ok := metrics["PowerLimitThrottleDuration"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_power_throttle_duration_seconds_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["ThermalLimitThrottleDuration"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_thermal_throttle_duration_seconds_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Oem/Nvidia/HardwareViolationThrottleDuration"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_hardware_violation_throttle_duration_seconds_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Oem/Nvidia/GlobalSoftwareViolationThrottleDuration"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_software_violation_throttle_duration_seconds_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
}

// collectMemoryMetrics processes a single HGX_MemoryMetrics report
func (t *TelemetryCollector) collectMemoryMetrics(ch chan<- prometheus.Metric, report *redfish.MetricReport, systemMap map[string]string, wg *sync.WaitGroup) {
	defer wg.Done()

	t.logger.Debug("processing memory metric report",
		slog.String("report_id", report.ID),
		slog.Int("metric_count", len(report.MetricValues)),
	)

	// Group metrics by GPU ID and Memory ID
	// MetricProperty format: /redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_SXM_1_DRAM_0/MemoryMetrics#/...
	metricsByMemory := make(map[string]map[string]map[string]float64) // systemID -> memoryID -> metricName -> value

	for _, metricValue := range report.MetricValues {
		// Parse the metric property to extract system ID, memory ID, and metric name
		systemID, memoryID, metricName := parseMemoryMetricProperty(metricValue.MetricProperty)
		if systemID == "" || memoryID == "" || metricName == "" {
			continue
		}

		// Parse metric value
		value, err := parseMetricValue(metricValue.MetricValue)
		if err != nil {
			t.logger.Debug("failed to parse metric value",
				slog.String("metric", metricName),
				slog.String("value", metricValue.MetricValue),
				slog.Any("error", err),
			)
			continue
		}

		// Initialize maps if needed
		if metricsByMemory[systemID] == nil {
			metricsByMemory[systemID] = make(map[string]map[string]float64)
		}
		if metricsByMemory[systemID][memoryID] == nil {
			metricsByMemory[systemID][memoryID] = make(map[string]float64)
		}
		metricsByMemory[systemID][memoryID][metricName] = value
	}

	// Emit metrics for each memory module
	for systemID, memoryMap := range metricsByMemory {
		systemName := systemMap[systemID]
		if systemName == "" {
			systemName = systemID
		}

		for memoryID, metrics := range memoryMap {
			// Extract GPU ID from memory ID (e.g., "GPU_SXM_1_DRAM_0" -> "GPU_SXM_1")
			gpuID := extractGPUIDFromMemoryID(memoryID)
			labels := []string{systemName, systemID, gpuID, memoryID}
			t.emitMemoryMetrics(ch, labels, metrics)
		}
	}
}

// parseMemoryMetricProperty extracts system ID, memory ID, and metric name from a MetricProperty path
// Example: /redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_SXM_1_DRAM_0/MemoryMetrics#/LifeTime/CorrectableECCErrorCount
// Returns: "HGX_Baseboard_0", "GPU_SXM_1_DRAM_0", "LifeTime/CorrectableECCErrorCount"
func parseMemoryMetricProperty(property string) (systemID string, memoryID string, metricName string) {
	// Split on '#' to separate resource path from property path
	parts := strings.Split(property, "#/")
	if len(parts) != 2 {
		return "", "", ""
	}

	resourcePath := parts[0]
	metricName = parts[1]

	// Extract system ID: /Systems/SYSTEM_ID/
	if idx := strings.Index(resourcePath, "/Systems/"); idx != -1 {
		remainder := resourcePath[idx+len("/Systems/"):]
		if endIdx := strings.Index(remainder, "/"); endIdx != -1 {
			systemID = remainder[:endIdx]
		}
	}

	// Extract memory ID: /Memory/MEMORY_ID/
	if idx := strings.Index(resourcePath, "/Memory/"); idx != -1 {
		remainder := resourcePath[idx+len("/Memory/"):]
		if endIdx := strings.Index(remainder, "/"); endIdx != -1 {
			memoryID = remainder[:endIdx]
		}
	}

	return systemID, memoryID, metricName
}

// extractGPUIDFromMemoryID extracts GPU ID from memory ID
// Example: "GPU_SXM_1_DRAM_0" -> "GPU_SXM_1"
func extractGPUIDFromMemoryID(memoryID string) string {
	// Look for pattern GPU_SXM_N or GPU_N
	parts := strings.Split(memoryID, "_")
	if len(parts) >= 2 && parts[0] == "GPU" {
		if parts[1] == "SXM" && len(parts) >= 3 {
			// GPU_SXM_N_...
			return fmt.Sprintf("GPU_SXM_%s", parts[2])
		}
		// GPU_N_...
		return fmt.Sprintf("GPU_%s", parts[1])
	}
	return memoryID
}

// emitMemoryMetrics emits Prometheus metrics for a single memory module
func (t *TelemetryCollector) emitMemoryMetrics(ch chan<- prometheus.Metric, labels []string, metrics map[string]float64) {
	// Lifetime ECC errors
	if val, ok := metrics["LifeTime/CorrectableECCErrorCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_memory_ecc_correctable_lifetime_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["LifeTime/UncorrectableECCErrorCount"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_memory_ecc_uncorrectable_lifetime_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// Bandwidth and utilization
	if val, ok := metrics["BandwidthPercent"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_memory_bandwidth_percent"].desc,
			prometheus.GaugeValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["CapacityUtilizationPercent"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_memory_capacity_utilization_percent"].desc,
			prometheus.GaugeValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["OperatingSpeedMHz"]; ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_memory_operating_speed_mhz"].desc,
			prometheus.GaugeValue,
			val,
			labels...,
		)
	}
}

// collectResetMetrics processes a single HGX_ProcessorResetMetrics report
func (t *TelemetryCollector) collectResetMetrics(ch chan<- prometheus.Metric, report *redfish.MetricReport, systemMap map[string]string, wg *sync.WaitGroup) {
	defer wg.Done()

	t.logger.Debug("processing reset metric report",
		slog.String("report_id", report.ID),
		slog.Int("metric_count", len(report.MetricValues)),
	)

	// Group metrics by GPU ID
	metricsByGPU := make(map[string]map[string]interface{}) // interface{} to handle both float64 and string

	for _, metricValue := range report.MetricValues {
		// Extract GPU ID and metric name from MetricProperty
		// Format: /redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_1/Oem/Nvidia/ProcessorResetMetrics#/...
		gpuID, metricName := parseResetMetricProperty(metricValue.MetricProperty)
		if gpuID == "" || metricName == "" {
			continue
		}

		// For LastResetType, store the string value; for others, parse as float
		if metricName == "LastResetType" {
			// Initialize GPU map if needed
			if metricsByGPU[gpuID] == nil {
				metricsByGPU[gpuID] = make(map[string]interface{})
			}
			metricsByGPU[gpuID][metricName] = metricValue.MetricValue
		} else {
			// Parse numeric metric value
			value, err := parseMetricValue(metricValue.MetricValue)
			if err != nil {
				t.logger.Debug("failed to parse reset metric value",
					slog.String("metric", metricName),
					slog.String("value", metricValue.MetricValue),
					slog.Any("error", err),
				)
				continue
			}

			// Initialize GPU map if needed
			if metricsByGPU[gpuID] == nil {
				metricsByGPU[gpuID] = make(map[string]interface{})
			}
			metricsByGPU[gpuID][metricName] = value
		}
	}

	// Extract system ID from metric properties (all should be same system)
	systemID := extractSystemIDFromReport(report)
	systemName := systemMap[systemID]
	if systemName == "" {
		systemName = systemID
	}

	// Emit metrics for each GPU
	for gpuID, metrics := range metricsByGPU {
		labels := []string{systemName, systemID, gpuID}
		t.emitResetMetrics(ch, labels, metrics)
	}
}

// parseResetMetricProperty extracts GPU ID and metric name from a reset MetricProperty path
// Example: /redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_1/Oem/Nvidia/ProcessorResetMetrics#/ConventionalResetEntryCount
// Returns: "GPU_SXM_1", "ConventionalResetEntryCount"
func parseResetMetricProperty(property string) (gpuID string, metricName string) {
	// Split on '#' to separate resource path from property path
	parts := strings.Split(property, "#/")
	if len(parts) != 2 {
		return "", ""
	}

	resourcePath := parts[0]
	metricName = parts[1]

	// Extract GPU ID from resource path
	// Look for pattern: /Processors/GPU_SXM_X/
	if idx := strings.Index(resourcePath, "/Processors/"); idx != -1 {
		remainder := resourcePath[idx+len("/Processors/"):]
		if endIdx := strings.Index(remainder, "/"); endIdx != -1 {
			gpuID = remainder[:endIdx]
		}
	}

	return gpuID, metricName
}

// emitResetMetrics emits Prometheus metrics for GPU reset events
func (t *TelemetryCollector) emitResetMetrics(ch chan<- prometheus.Metric, labels []string, metrics map[string]interface{}) {
	// Conventional reset counters
	if val, ok := metrics["ConventionalResetEntryCount"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_conventional_reset_entry_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["ConventionalResetExitCount"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_conventional_reset_exit_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// Fundamental reset counters
	if val, ok := metrics["FundamentalResetEntryCount"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_fundamental_reset_entry_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["FundamentalResetExitCount"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_fundamental_reset_exit_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// IRoT reset counter
	if val, ok := metrics["IRoTResetExitCount"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_irot_reset_exit_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// PF_FLR reset counters
	if val, ok := metrics["PF_FLR_ResetEntryCount"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pf_flr_reset_entry_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["PF_FLR_ResetExitCount"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_pf_flr_reset_exit_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// LastResetType - convert string to numeric value
	if val, ok := metrics["LastResetType"].(string); ok {
		var resetTypeValue float64
		switch val {
		case "Conventional":
			resetTypeValue = 1.0
		case "Fundamental":
			resetTypeValue = 2.0
		case "IRoT":
			resetTypeValue = 3.0
		case "PF_FLR":
			resetTypeValue = 4.0
		default:
			// Unknown reset type, skip emitting
			return
		}

		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_last_reset_type_info"].desc,
			prometheus.GaugeValue,
			resetTypeValue,
			labels...,
		)
	}
}

// collectPortMetrics processes a single HGX_ProcessorPortMetrics report
func (t *TelemetryCollector) collectPortMetrics(ch chan<- prometheus.Metric, report *redfish.MetricReport, systemMap map[string]string, wg *sync.WaitGroup) {
	defer wg.Done()

	t.logger.Debug("processing port metric report",
		slog.String("report_id", report.ID),
		slog.Int("metric_count", len(report.MetricValues)),
	)

	// Group metrics by GPU ID and Port ID
	// MetricProperty format: /redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_0/Ports/NVLink_0/Metrics#/RXBytes
	// or /redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_0/Ports/PCIe/Metrics#/CurrentSpeedGbps
	metricsByPort := make(map[string]map[string]map[string]interface{}) // gpuID -> portID -> metricName -> value

	for _, metricValue := range report.MetricValues {
		// Extract GPU ID, port ID, and metric name from MetricProperty
		gpuID, portID, metricName := parsePortMetricProperty(metricValue.MetricProperty)
		if gpuID == "" || portID == "" || metricName == "" {
			continue
		}

		// For LinkDownReasonCode, store the string value; for others, parse as float
		if metricName == "Oem/Nvidia/LinkDownReasonCode" {
			// Initialize maps if needed
			if metricsByPort[gpuID] == nil {
				metricsByPort[gpuID] = make(map[string]map[string]interface{})
			}
			if metricsByPort[gpuID][portID] == nil {
				metricsByPort[gpuID][portID] = make(map[string]interface{})
			}
			metricsByPort[gpuID][portID][metricName] = metricValue.MetricValue
		} else {
			// Parse numeric metric value
			value, err := parseMetricValue(metricValue.MetricValue)
			if err != nil {
				t.logger.Debug("failed to parse port metric value",
					slog.String("metric", metricName),
					slog.String("value", metricValue.MetricValue),
					slog.Any("error", err),
				)
				continue
			}

			// Initialize maps if needed
			if metricsByPort[gpuID] == nil {
				metricsByPort[gpuID] = make(map[string]map[string]interface{})
			}
			if metricsByPort[gpuID][portID] == nil {
				metricsByPort[gpuID][portID] = make(map[string]interface{})
			}
			metricsByPort[gpuID][portID][metricName] = value
		}
	}

	// Extract system ID from metric properties (all should be same system)
	systemID := extractSystemIDFromReport(report)
	systemName := systemMap[systemID]
	if systemName == "" {
		systemName = systemID
	}

	// Emit metrics for each port
	for gpuID, ports := range metricsByPort {
		for portID, metrics := range ports {
			labels := []string{systemName, systemID, gpuID, portID}
			t.emitPortMetrics(ch, labels, metrics)
		}
	}
}

// parsePortMetricProperty extracts GPU ID, port ID, and metric name from a port MetricProperty path
// Example: /redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_0/Ports/NVLink_0/Metrics#/RXBytes
// Returns: "GPU_SXM_0", "NVLink_0", "RXBytes"
func parsePortMetricProperty(property string) (gpuID string, portID string, metricName string) {
	// Split on '#' to separate resource path from property path
	parts := strings.Split(property, "#/")
	if len(parts) != 2 {
		return "", "", ""
	}

	resourcePath := parts[0]
	metricName = parts[1]

	// Extract GPU ID from resource path: /Processors/GPU_SXM_X/
	if idx := strings.Index(resourcePath, "/Processors/"); idx != -1 {
		remainder := resourcePath[idx+len("/Processors/"):]
		if endIdx := strings.Index(remainder, "/"); endIdx != -1 {
			gpuID = remainder[:endIdx]
		}
	}

	// Extract Port ID from resource path: /Ports/PORT_ID/
	if idx := strings.Index(resourcePath, "/Ports/"); idx != -1 {
		remainder := resourcePath[idx+len("/Ports/"):]
		if endIdx := strings.Index(remainder, "/"); endIdx != -1 {
			portID = remainder[:endIdx]
		}
	}

	return gpuID, portID, metricName
}

// emitPortMetrics emits Prometheus metrics for a single port
func (t *TelemetryCollector) emitPortMetrics(ch chan<- prometheus.Metric, labels []string, metrics map[string]interface{}) {
	// Link speed
	if val, ok := metrics["CurrentSpeedGbps"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_current_speed_gbps"].desc,
			prometheus.GaugeValue,
			val,
			labels...,
		)
	}

	// Standard networking metrics - cumulative counters
	if val, ok := metrics["RXBytes"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_rx_bytes_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["TXBytes"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_tx_bytes_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["RXErrors"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_rx_errors_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Networking/RXFrames"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_rx_frames_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Networking/TXFrames"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_tx_frames_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Networking/TXDiscards"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_tx_discards_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// OEM metrics - link reliability counters
	if val, ok := metrics["Oem/Nvidia/IntentionalLinkDownCount"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_intentional_link_down_count_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Oem/Nvidia/UnintentionalLinkDownCount"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_unintentional_link_down_count_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// LinkDownReasonCode - convert string to numeric value if it exists
	if val, ok := metrics["Oem/Nvidia/LinkDownReasonCode"].(string); ok {
		// Parse as integer if possible, otherwise skip
		if reasonCode, err := strconv.ParseFloat(val, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(
				t.metrics["telemetry_port_link_down_reason_code"].desc,
				prometheus.GaugeValue,
				reasonCode,
				labels...,
			)
		}
	}

	// OEM metrics - error counters
	if val, ok := metrics["Oem/Nvidia/NeighborMTUDiscards"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_neighbor_mtu_discards_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Oem/Nvidia/QP1Dropped"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_qp1_dropped_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Oem/Nvidia/RXRemotePhysicalErrors"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_rx_remote_physical_errors_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Oem/Nvidia/RXSwitchRelayErrors"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_rx_switch_relay_errors_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Oem/Nvidia/VL15Dropped"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_vl15_dropped_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// OEM metrics - no-protocol bytes
	if val, ok := metrics["Oem/Nvidia/RXNoProtocolBytes"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_rx_no_protocol_bytes_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Oem/Nvidia/TXNoProtocolBytes"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_tx_no_protocol_bytes_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// OEM metrics - VL15 (management traffic)
	if val, ok := metrics["Oem/Nvidia/VL15TXBytes"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_vl15_tx_bytes_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Oem/Nvidia/VL15TXPackets"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_vl15_tx_packets_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// OEM metrics - link width (current state)
	if val, ok := metrics["Oem/Nvidia/RXWidth"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_rx_width"].desc,
			prometheus.GaugeValue,
			val,
			labels...,
		)
	}
	if val, ok := metrics["Oem/Nvidia/TXWidth"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_tx_width"].desc,
			prometheus.GaugeValue,
			val,
			labels...,
		)
	}

	// OEM metrics - TX wait (cumulative)
	if val, ok := metrics["Oem/Nvidia/TXWait"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_tx_wait_total"].desc,
			prometheus.CounterValue,
			val,
			labels...,
		)
	}

	// OEM metrics - bit error rate (current state)
	if val, ok := metrics["Oem/Nvidia/TotalRawBER"].(float64); ok {
		ch <- prometheus.MustNewConstMetric(
			t.metrics["telemetry_port_total_raw_ber"].desc,
			prometheus.GaugeValue,
			val,
			labels...,
		)
	}
}

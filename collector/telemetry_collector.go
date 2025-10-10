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

	return metrics
}

// TelemetryCollector collects metrics from Redfish TelemetryService
type TelemetryCollector struct {
	redfishClient         *gofish.APIClient
	metrics               map[string]Metric
	logger                *slog.Logger
	collectorScrapeStatus *prometheus.GaugeVec
	capabilities          *TelemetryCapabilities
}

// NewTelemetryCollector creates a new TelemetryService collector
func NewTelemetryCollector(redfishClient *gofish.APIClient, logger *slog.Logger, capabilities *TelemetryCapabilities) *TelemetryCollector {
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
		capabilities: capabilities,
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

	// Reset capabilities at the start of each scrape
	if t.capabilities != nil {
		t.capabilities.Reset()
	}

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
		if strings.Contains(report.ID, "HGX_ProcessorMetrics") {
			wg.Add(1)
			go t.collectProcessorMetrics(ch, report, systemMap, wg)
		} else if strings.Contains(report.ID, "HGX_MemoryMetrics") {
			wg.Add(1)
			go t.collectMemoryMetrics(ch, report, systemMap, wg)
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

	// Register that processor metrics were collected (once per report, not per GPU)
	if t.capabilities != nil && len(metricsByGPU) > 0 {
		t.capabilities.Register(CapabilityProcessorMetrics)
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

	// Register that memory metrics were collected
	if t.capabilities != nil {
		t.capabilities.Register(CapabilityMemoryMetrics)
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

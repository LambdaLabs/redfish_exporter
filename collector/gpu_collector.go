package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	isoDuration "github.com/sosodev/duration"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

// GPUSubsystem is the GPU subsystem name
const GPUSubsystem = "gpu"

// GPU metric label names
var (
	// Base labels using the main branch pattern but with system_id instead of system
	gpuBaseLabels      = []string{"hostname", "system_id", "gpu_id"}
	gpuMemoryLabels    = baseWithExtraLabels([]string{"memory_id"})
	gpuProcessorLabels = baseWithExtraLabels([]string{"processor_name"})
	gpuPortLabels      = baseWithExtraLabels([]string{"port_id", "port_type", "port_protocol"})

	gpuMetrics     = createGPUMetricMap()
	gpuMemoryTypes = createGPUMemoryTypeSet()
)

func baseWithExtraLabels(extra []string) []string {
	gpuBaseLabelsCopy := make([]string, len(gpuBaseLabels))
	copy(gpuBaseLabelsCopy, gpuBaseLabels)
	return append(gpuBaseLabelsCopy, extra...)
}

// createGPUMemoryTypeSet creates a set of GPU memory types for efficient lookup
func createGPUMemoryTypeSet() map[redfish.MemoryDeviceType]bool {
	return map[redfish.MemoryDeviceType]bool{
		redfish.HBMMemoryDeviceType:    true,
		redfish.HBM2MemoryDeviceType:   true,
		redfish.HBM2EMemoryDeviceType:  true,
		redfish.HBM3MemoryDeviceType:   true,
		redfish.GDDRMemoryDeviceType:   true,
		redfish.GDDR2MemoryDeviceType:  true,
		redfish.GDDR3MemoryDeviceType:  true,
		redfish.GDDR4MemoryDeviceType:  true,
		redfish.GDDR5MemoryDeviceType:  true,
		redfish.GDDR5XMemoryDeviceType: true,
		redfish.GDDR6MemoryDeviceType:  true,
	}
}

// isGPUMemory checks if the memory device type indicates GPU memory
func isGPUMemory(deviceType redfish.MemoryDeviceType) bool {
	return gpuMemoryTypes[deviceType]
}

// GPUCollector collects GPU-specific metrics including Nvidia OEM fields
type GPUCollector struct {
	redfishClient         *gofish.APIClient
	metrics               map[string]Metric
	logger                *slog.Logger
	collectorScrapeStatus *prometheus.GaugeVec
	oemClient             *NvidiaOEMClient
}

// systemGPUInfo holds GPU information for a system
type systemGPUInfo struct {
	systemName string
	systemID   string
	gpus       []*redfish.Processor
}

func createGPUMetricMap() map[string]Metric {
	gpuMetrics := make(map[string]Metric)

	// Basic GPU metrics from main branch
	addToMetricMap(gpuMetrics, GPUSubsystem, "health", "health of gpu reported by system,1(OK),2(Warning),3(Critical)", gpuBaseLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_ecc_correctable", "current correctable memory ecc errors reported on the gpu", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_ecc_uncorrectable", "current uncorrectable memory ecc errors reported on the gpu", gpuMemoryLabels)

	// GPU Memory metrics
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_capacity_mib", "GPU memory capacity in MiB", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_state", fmt.Sprintf("GPU memory state,%s", CommonStateHelp), gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_health", fmt.Sprintf("GPU memory health,%s", CommonHealthHelp), gpuMemoryLabels)

	// Nvidia GPU Memory OEM metrics
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_row_remapping_failed", "GPU memory row remapping failed status (1 if failed)", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_row_remapping_pending", "GPU memory row remapping pending status (1 if pending)", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_correctable_row_remapping_count", "GPU memory correctable row remapping count", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_uncorrectable_row_remapping_count", "GPU memory uncorrectable row remapping count", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_high_availability_bank_count", "GPU memory high availability bank count", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_low_availability_bank_count", "GPU memory low availability bank count", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_no_availability_bank_count", "GPU memory no availability bank count", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_partial_availability_bank_count", "GPU memory partial availability bank count", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_max_availability_bank_count", "GPU memory max availability bank count", gpuMemoryLabels)

	// GPU Processor metrics
	addToMetricMap(gpuMetrics, GPUSubsystem, "processor_state", fmt.Sprintf("GPU processor state,%s", CommonStateHelp), gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "processor_health", fmt.Sprintf("GPU processor health,%s", CommonHealthHelp), gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "processor_total_cores", "GPU processor total cores", gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "processor_total_threads", "GPU processor total threads", gpuProcessorLabels)

	// Nvidia GPU Processor OEM metrics
	// Note: SM utilization, activity, occupancy, tensor/FP activity, and PCIe bandwidth metrics
	// are now collected via TelemetryService (HGX_ProcessorGPMMetrics_0) for better performance
	addToMetricMap(gpuMetrics, GPUSubsystem, "sram_ecc_error_threshold_exceeded", "GPU SRAM ECC error threshold exceeded (1 if exceeded)", gpuProcessorLabels)

	// NVLink Port metrics
	addToMetricMap(gpuMetrics, GPUSubsystem, "nvlink_state", fmt.Sprintf("NVLink port state,%s", CommonStateHelp), gpuPortLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "nvlink_health", fmt.Sprintf("NVLink port health,%s", CommonHealthHelp), gpuPortLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "nvlink_runtime_error", "NVLink runtime error status (1 if error)", gpuPortLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "nvlink_training_error", "NVLink training error status (1 if error)", gpuPortLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "nvlink_link_error_recovery_count", "NVLink error recovery count", gpuPortLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "nvlink_link_downed_count", "NVLink link downed count", gpuPortLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "nvlink_symbol_errors", "NVLink symbol error count", gpuPortLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "nvlink_bit_error_rate", "NVLink bit error rate", gpuPortLabels)

	// GPU Temperature sensor metrics
	addToMetricMap(gpuMetrics, GPUSubsystem, "temperature_tlimit_celsius", "GPU TLIMIT temperature headroom in Celsius", gpuBaseLabels)

	// GPU Memory Power metrics
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_power_watts", "GPU memory (DRAM) power consumption in watts", gpuMemoryLabels)

	// GPU Context Utilization metrics
	addToMetricMap(gpuMetrics, GPUSubsystem, "context_utilization_seconds_total", "Accumulated GPU context utilization duration in seconds", gpuBaseLabels)

	return gpuMetrics
}

// NewGPUCollector creates a new GPU collector
func NewGPUCollector(redfishClient *gofish.APIClient, logger *slog.Logger) *GPUCollector {
	return &GPUCollector{
		redfishClient: redfishClient,
		metrics:       gpuMetrics,
		logger:        logger.With(slog.String("collector", "GPUCollector")),
		collectorScrapeStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "collector_scrape_status",
				Help:      "collector_scrape_status",
			},
			[]string{"collector"},
		),
		oemClient: NewNvidiaOEMClient(redfishClient.GetService().GetClient(), logger),
	}
}

// Describe implements prometheus.Collector
func (g *GPUCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range g.metrics {
		ch <- metric.desc
	}
	g.collectorScrapeStatus.Describe(ch)
}

// Collect implements prometheus.Collector
func (g *GPUCollector) Collect(ch chan<- prometheus.Metric) {
	g.collectorScrapeStatus.WithLabelValues("gpu").Set(float64(0))

	service := g.redfishClient.Service
	systems, err := service.Systems()
	if err != nil {
		g.logger.Error("failed getting systems",
			slog.Any("error", err),
			slog.String("operation", "service.Systems()"),
		)
		return
	}

	// Channel to collect GPU info from each system
	gpuInfoChan := make(chan systemGPUInfo, len(systems))
	wg := &sync.WaitGroup{}
	for _, system := range systems {
		wg.Add(1)
		go g.collectSystemGPUs(ch, system, wg, gpuInfoChan)
	}

	// Close channel after all systems are processed
	go func() {
		wg.Wait()
		close(gpuInfoChan)
	}()

	// Gather all GPU info while systems are being processed
	var allSystemGPUs []systemGPUInfo
	for info := range gpuInfoChan {
		if len(info.gpus) > 0 {
			allSystemGPUs = append(allSystemGPUs, info)
			g.logger.Debug("Collected GPU info",
				slog.String("system", info.systemName),
				slog.Int("gpu_count", len(info.gpus)))
		}
	}

	// Collect GPU temperature sensors using gathered GPU info
	g.collectGPUTemperatureSensors(ch, allSystemGPUs)

	// Collect GPU memory power sensors using gathered GPU info
	g.collectGPUMemoryPowerSensors(ch, allSystemGPUs)

	// Collect GPU context utilization metrics
	g.collectGPUContextUtilization(ch, allSystemGPUs)

	g.collectorScrapeStatus.WithLabelValues("gpu").Set(float64(1))
}

// collectSystemGPUs collects all GPU-related metrics for a system
func (g *GPUCollector) collectSystemGPUs(ch chan<- prometheus.Metric, system *redfish.ComputerSystem, wg *sync.WaitGroup, gpuInfoChan chan<- systemGPUInfo) {
	defer wg.Done()

	systemID := system.ID
	systemName := system.Name
	if systemName == "" {
		systemName = systemID
	}

	// Get processors first for GPU-specific metrics
	processors, err := system.Processors()
	if err != nil {
		g.logger.Error("failed to get processors for system",
			slog.String("system_id", systemID),
			slog.Any("error", err),
		)
		return
	}

	// Filter for GPU processors and collect basic metrics (from main branch)
	gpus := filterGPUs(processors)

	// Send GPU info back through channel
	if len(gpus) > 0 {
		gpuInfoChan <- systemGPUInfo{
			systemName: systemName,
			systemID:   systemID,
			gpus:       gpus,
		}
	}

	for _, gpu := range gpus {
		if gpu.Name == "" {
			gpu.Name = gpu.ID
		}
		commonGPULabels := []string{gpu.Name, systemName, gpu.ID}
		emitGPUHealth(ch, gpu, commonGPULabels, g.metrics)

		// Get GPU-specific memory for ECC metrics
		gpuMem, err := gpu.Memory()
		if err != nil {
			g.logger.Error("error getting gpu memory", slog.Any("error", err))
			continue
		}
		memWithMetrics := make([]MemoryWithMetrics, len(gpuMem))
		for i, mem := range gpuMem {
			memWithMetrics[i] = &redfishMemoryAdapter{Memory: mem}
		}
		emitGPUECCMetrics(ch, memWithMetrics, g.logger, commonGPULabels, g.metrics)
	}

	// Collect detailed OEM metrics
	// Note: TelemetryService provides aggregated memory statistics (ECC totals, bandwidth, etc.)
	// while this collects structural health indicators (remapping, banks, state, capacity).
	// These are complementary, not overlapping, so we always collect both.
	wgMemory := &sync.WaitGroup{}
	semMemory := make(chan struct{}, 5)

	if memories, err := system.Memory(); err != nil {
		g.logger.Error("failed to get memory for system",
			slog.String("system_id", systemID),
			slog.Any("error", err),
		)
	} else {
		for _, memory := range memories {
			// Collect metrics for GPU memory (HBM and GDDR types)
			if isGPUMemory(memory.MemoryDeviceType) {
				wgMemory.Add(1)
				mem := memory // Capture loop variable
				go func() {
					semMemory <- struct{}{}        // Acquire semaphore
					defer func() { <-semMemory }() // Release semaphore
					g.collectGPUMemory(ch, systemName, systemID, mem, wgMemory)
				}()
			}
		}
	}

	// Collect GPU processor metrics (reusing processors)
	wgProcessor := &sync.WaitGroup{}
	// Use semaphore to limit concurrent goroutines
	semProcessor := make(chan struct{}, 5)

	for _, processor := range processors {
		// Collect metrics for any GPU processor
		if processor.ProcessorType == redfish.GPUProcessorType {
			wgProcessor.Add(1)
			proc := processor // Capture loop variable
			go func() {
				semProcessor <- struct{}{}        // Acquire semaphore
				defer func() { <-semProcessor }() // Release semaphore
				g.collectGPUProcessor(ch, systemName, systemID, proc, wgProcessor)
			}()
		}
	}

	// Wait for both memory and processor collection to complete
	// This allows them to run in parallel for maximum efficiency
	wgMemory.Wait()
	wgProcessor.Wait()
}

// collectGPUMemory collects GPU memory metrics including OEM fields
func (g *GPUCollector) collectGPUMemory(ch chan<- prometheus.Metric, systemName, systemID string, memory *redfish.Memory, wg *sync.WaitGroup) {
	defer wg.Done()

	memoryID := memory.ID
	// Extract GPU ID from memory ID (e.g., "GPU_0_DRAM_0" -> "GPU_0")
	gpuID := extractGPUID(memoryID)

	labels := []string{systemName, systemID, gpuID, memoryID}

	// Basic memory metrics
	ch <- prometheus.MustNewConstMetric(
		g.metrics["gpu_memory_capacity_mib"].desc,
		prometheus.GaugeValue,
		float64(memory.CapacityMiB),
		labels...,
	)

	if stateValue, ok := parseCommonStatusState(memory.Status.State); ok {
		ch <- prometheus.MustNewConstMetric(
			g.metrics["gpu_memory_state"].desc,
			prometheus.GaugeValue,
			stateValue,
			labels...,
		)
	}

	if healthValue, ok := parseCommonStatusHealth(memory.Status.Health); ok {
		ch <- prometheus.MustNewConstMetric(
			g.metrics["gpu_memory_health"].desc,
			prometheus.GaugeValue,
			healthValue,
			labels...,
		)
	}

	// Get Memory OEM metrics
	if memOEM, err := g.oemClient.GetMemoryOEMMetrics(memory.ODataID); err == nil {
		ch <- prometheus.MustNewConstMetric(
			g.metrics["gpu_memory_row_remapping_failed"].desc,
			prometheus.GaugeValue,
			boolToFloat64(memOEM.RowRemappingFailed),
			labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			g.metrics["gpu_memory_row_remapping_pending"].desc,
			prometheus.GaugeValue,
			boolToFloat64(memOEM.RowRemappingPending),
			labels...,
		)
	} else {
		g.logger.Error("failed to get Memory OEM metrics",
			slog.String("memory_id", memoryID),
			slog.Any("error", err),
		)
	}

	// Get MemoryMetrics OEM data
	if metrics, err := memory.Metrics(); err == nil && metrics != nil {
		if metricsOEM, err := g.oemClient.GetMemoryMetricsOEMData(metrics.ODataID); err == nil {
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_memory_correctable_row_remapping_count"].desc,
				prometheus.GaugeValue,
				float64(metricsOEM.CorrectableRowRemappingCount),
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_memory_uncorrectable_row_remapping_count"].desc,
				prometheus.GaugeValue,
				float64(metricsOEM.UncorrectableRowRemappingCount),
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_memory_high_availability_bank_count"].desc,
				prometheus.GaugeValue,
				float64(metricsOEM.HighAvailabilityBankCount),
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_memory_low_availability_bank_count"].desc,
				prometheus.GaugeValue,
				float64(metricsOEM.LowAvailabilityBankCount),
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_memory_no_availability_bank_count"].desc,
				prometheus.GaugeValue,
				float64(metricsOEM.NoAvailabilityBankCount),
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_memory_partial_availability_bank_count"].desc,
				prometheus.GaugeValue,
				float64(metricsOEM.PartialAvailabilityBankCount),
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_memory_max_availability_bank_count"].desc,
				prometheus.GaugeValue,
				float64(metricsOEM.MaxAvailabilityBankCount),
				labels...,
			)
		} else {
			g.logger.Error("failed to get MemoryMetrics OEM data",
				slog.String("memory_id", memoryID),
				slog.Any("error", err),
			)
		}
	}
}

// collectGPUProcessor collects GPU processor metrics including OEM fields
func (g *GPUCollector) collectGPUProcessor(ch chan<- prometheus.Metric, systemName, systemID string, processor *redfish.Processor, wg *sync.WaitGroup) {
	defer wg.Done()

	processorID := processor.ID
	processorName := processor.Name
	if processorName == "" {
		processorName = processorID
	}

	labels := []string{systemName, systemID, processorID, processorName}

	// Basic processor metrics
	if stateValue, ok := parseCommonStatusState(processor.Status.State); ok {
		ch <- prometheus.MustNewConstMetric(
			g.metrics["gpu_processor_state"].desc,
			prometheus.GaugeValue,
			stateValue,
			labels...,
		)
	}

	if healthValue, ok := parseCommonStatusHealth(processor.Status.Health); ok {
		ch <- prometheus.MustNewConstMetric(
			g.metrics["gpu_processor_health"].desc,
			prometheus.GaugeValue,
			healthValue,
			labels...,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		g.metrics["gpu_processor_total_cores"].desc,
		prometheus.GaugeValue,
		float64(processor.TotalCores),
		labels...,
	)

	ch <- prometheus.MustNewConstMetric(
		g.metrics["gpu_processor_total_threads"].desc,
		prometheus.GaugeValue,
		float64(processor.TotalThreads),
		labels...,
	)

	// Get ProcessorMetrics OEM data
	// Note: Most GPU performance metrics (SM activity, tensor cores, FP operations, PCIe bandwidth)
	// are now collected via TelemetryService (HGX_ProcessorGPMMetrics_0) for better performance.
	// We only collect SRAM ECC threshold here as it's a health indicator not in GPM.
	if metrics, err := processor.Metrics(); err == nil && metrics != nil {
		if metricsOEM, err := g.oemClient.GetProcessorMetricsOEMData(metrics.ODataID); err == nil {
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_sram_ecc_error_threshold_exceeded"].desc,
				prometheus.GaugeValue,
				boolToFloat64(metricsOEM.SRAMECCErrorThresholdExceeded),
				labels...,
			)
		} else {
			g.logger.Debug("failed to get ProcessorMetrics OEM data",
				slog.String("processor_id", processorID),
				slog.Any("error", err),
			)
		}
	}

	// Collect NVLink port metrics
	g.collectNVLinkPorts(ch, systemName, systemID, processorID, processor)
}

// collectNVLinkPorts collects NVLink port metrics for a GPU processor
func (g *GPUCollector) collectNVLinkPorts(ch chan<- prometheus.Metric, systemName, systemID, gpuID string, processor *redfish.Processor) {
	ports, err := processor.Ports()
	if err != nil {
		g.logger.Error("failed to get port data")
	}

	for _, port := range ports {
		if port.PortProtocol != redfish.NVLinkPortProtocol && !strings.Contains(port.ID, "NVLink_") {
			continue
		}

		portID := port.ID
		portType := string(port.PortType)
		portProtocol := string(port.PortProtocol)
		labels := []string{systemName, systemID, gpuID, portID, portType, portProtocol}

		// Basic port metrics
		if stateValue, ok := parseCommonStatusState(port.Status.State); ok {
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_nvlink_state"].desc,
				prometheus.GaugeValue,
				stateValue,
				labels...,
			)
		}

		if healthValue, ok := parseCommonStatusHealth(port.Status.Health); ok {
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_nvlink_health"].desc,
				prometheus.GaugeValue,
				healthValue,
				labels...,
			)
		}

		// Get PortMetrics OEM data
		if metrics, err := port.Metrics(); err == nil && metrics != nil {
			if metricsOEM, err := g.oemClient.GetPortMetricsOEMData(metrics.ODataID); err == nil {
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_nvlink_runtime_error"].desc,
					prometheus.GaugeValue,
					boolToFloat64(metricsOEM.NVLinkErrors.RuntimeError),
					labels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_nvlink_training_error"].desc,
					prometheus.GaugeValue,
					boolToFloat64(metricsOEM.NVLinkErrors.TrainingError),
					labels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_nvlink_link_error_recovery_count"].desc,
					prometheus.GaugeValue,
					float64(metricsOEM.LinkErrorRecoveryCount),
					labels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_nvlink_link_downed_count"].desc,
					prometheus.GaugeValue,
					float64(metricsOEM.LinkDownedCount),
					labels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_nvlink_symbol_errors"].desc,
					prometheus.GaugeValue,
					float64(metricsOEM.SymbolErrors),
					labels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_nvlink_bit_error_rate"].desc,
					prometheus.GaugeValue,
					metricsOEM.BitErrorRate,
					labels...,
				)
			} else {
				g.logger.Error("failed to get PortMetrics OEM data",
					slog.String("port_id", portID),
					slog.Any("error", err),
				)
			}
		}
	}
}

// collectGPUTemperatureSensors collects GPU temperature sensor readings from Chassis
func (g *GPUCollector) collectGPUTemperatureSensors(ch chan<- prometheus.Metric, allSystemGPUs []systemGPUInfo) {
	// Use a wait group to collect temperatures concurrently
	wg := &sync.WaitGroup{}
	// Limit concurrent requests to avoid overwhelming the server
	sem := make(chan struct{}, 5)

	for _, sysInfo := range allSystemGPUs {
		for _, gpu := range sysInfo.gpus {
			wg.Add(1)
			go func(gpuID, systemName, systemID string) {
				defer wg.Done()
				sem <- struct{}{}        // Acquire semaphore
				defer func() { <-sem }() // Release semaphore

				g.collectSingleGPUTemperature(ch, gpuID, systemName, systemID)
			}(gpu.ID, sysInfo.systemName, sysInfo.systemID)
		}
	}

	wg.Wait()
}

// collectSingleGPUTemperature collects temperature for a single GPU
func (g *GPUCollector) collectSingleGPUTemperature(ch chan<- prometheus.Metric, gpuID, systemName, systemID string) {
	// Construct the chassis sensor path for this GPU
	// Pattern: /redfish/v1/Chassis/HGX_{gpuID}/Sensors/HGX_{gpuID}_TEMP_1
	chassisID := fmt.Sprintf("HGX_%s", gpuID)
	sensorPath := fmt.Sprintf("/redfish/v1/Chassis/%s/Sensors/%s_TEMP_1", chassisID, chassisID)

	// Use gofish to get the sensor
	sensor, err := redfish.GetSensor(g.redfishClient, sensorPath)
	if err != nil {
		g.logger.Debug("failed to fetch GPU temperature sensor",
			slog.String("gpu_id", gpuID),
			slog.String("path", sensorPath),
			slog.Any("error", err),
		)
		return
	}

	// Verify it's a temperature sensor (defensive check)
	if sensor.ReadingType != redfish.TemperatureReadingType {
		g.logger.Warn("sensor is not a temperature type",
			slog.String("gpu_id", gpuID),
			slog.String("actual_type", string(sensor.ReadingType)),
		)
		return
	}

	// Log additional sensor info if in debug mode
	g.logger.Debug("collected GPU temperature sensor",
		slog.String("gpu_id", gpuID),
		slog.Float64("reading", float64(sensor.Reading)),
		slog.String("reading_basis", string(sensor.ReadingBasis)),
		slog.String("status", string(sensor.Status.Health)),
	)

	// Emit the temperature metric
	labels := []string{systemName, systemID, gpuID}

	ch <- prometheus.MustNewConstMetric(
		g.metrics["gpu_temperature_tlimit_celsius"].desc,
		prometheus.GaugeValue,
		float64(sensor.Reading),
		labels...,
	)
}

// collectGPUMemoryPowerSensors collects GPU memory power consumption from Chassis sensors
func (g *GPUCollector) collectGPUMemoryPowerSensors(ch chan<- prometheus.Metric, allSystemGPUs []systemGPUInfo) {
	wg := &sync.WaitGroup{}
	// Limit concurrent requests to avoid overwhelming the server
	sem := make(chan struct{}, 5)

	for _, sysInfo := range allSystemGPUs {
		for _, gpu := range sysInfo.gpus {
			// Assuming each GPU has DRAM_0, we can extend this if there are multiple DRAMs
			// Pattern: HGX_GPU_{gpu_id}_DRAM_0_Power_0
			wg.Add(1)
			go func(gpuID, systemName, systemID string) {
				defer wg.Done()
				sem <- struct{}{}        // Acquire semaphore
				defer func() { <-sem }() // Release semaphore

				// For now, assume DRAM_0. Could be extended to discover all DRAMs
				memoryID := fmt.Sprintf("%s_DRAM_0", gpuID)
				g.collectSingleMemoryPower(ch, gpuID, memoryID, systemName, systemID)
			}(gpu.ID, sysInfo.systemName, sysInfo.systemID)
		}
	}

	wg.Wait()
}

// collectSingleMemoryPower collects power for a single GPU memory module
func (g *GPUCollector) collectSingleMemoryPower(ch chan<- prometheus.Metric, gpuID, memoryID, systemName, systemID string) {
	// Construct the chassis sensor path for this GPU memory power
	// Pattern: /redfish/v1/Chassis/HGX_{gpuID}/Sensors/HGX_{memoryID}_Power_0
	chassisID := fmt.Sprintf("HGX_%s", gpuID)
	sensorPath := fmt.Sprintf("/redfish/v1/Chassis/%s/Sensors/HGX_%s_Power_0", chassisID, memoryID)

	// Use gofish to get the sensor
	sensor, err := redfish.GetSensor(g.redfishClient, sensorPath)
	if err != nil {
		g.logger.Debug("failed to fetch GPU memory power sensor",
			slog.String("gpu_id", gpuID),
			slog.String("memory_id", memoryID),
			slog.String("path", sensorPath),
			slog.Any("error", err),
		)
		return
	}

	// Verify it's a power sensor (defensive check)
	if sensor.ReadingType != redfish.PowerReadingType {
		g.logger.Warn("sensor is not a power type",
			slog.String("gpu_id", gpuID),
			slog.String("memory_id", memoryID),
			slog.String("actual_type", string(sensor.ReadingType)),
		)
		return
	}

	// Log additional sensor info if in debug mode
	g.logger.Debug("collected GPU memory power sensor",
		slog.String("gpu_id", gpuID),
		slog.String("memory_id", memoryID),
		slog.Float64("reading", float64(sensor.Reading)),
		slog.String("units", string(sensor.ReadingUnits)),
		slog.String("status", string(sensor.Status.Health)),
	)

	// Emit the power metric
	labels := []string{systemName, systemID, gpuID, memoryID}

	ch <- prometheus.MustNewConstMetric(
		g.metrics["gpu_memory_power_watts"].desc,
		prometheus.GaugeValue,
		float64(sensor.Reading),
		labels...,
	)
}

// collectGPUContextUtilization collects accumulated GPU context utilization duration from ProcessorMetrics
func (g *GPUCollector) collectGPUContextUtilization(ch chan<- prometheus.Metric, allSystemGPUs []systemGPUInfo) {
	wg := &sync.WaitGroup{}
	// Limit concurrent requests
	sem := make(chan struct{}, 5)

	for _, sysInfo := range allSystemGPUs {
		for _, gpu := range sysInfo.gpus {
			wg.Add(1)
			go func(gpuProcessor *redfish.Processor, systemName, systemID string) {
				defer wg.Done()
				sem <- struct{}{}        // Acquire semaphore
				defer func() { <-sem }() // Release semaphore

				g.collectSingleGPUContextUtilization(ch, gpuProcessor, systemName, systemID)
			}(gpu, sysInfo.systemName, sysInfo.systemID)
		}
	}

	wg.Wait()
}

// collectSingleGPUContextUtilization collects context utilization for a single GPU
func (g *GPUCollector) collectSingleGPUContextUtilization(ch chan<- prometheus.Metric, gpuProcessor *redfish.Processor, systemName, systemID string) {
	// Use gofish's built-in method to get ProcessorMetrics
	metrics, err := gpuProcessor.Metrics()
	if err != nil {
		g.logger.Debug("failed to fetch GPU processor metrics",
			slog.String("gpu_id", gpuProcessor.ID),
			slog.Any("error", err),
		)
		return
	}

	if metrics == nil {
		g.logger.Debug("no processor metrics available",
			slog.String("gpu_id", gpuProcessor.ID),
		)
		return
	}

	// Parse the OEM field to look for NVIDIA-specific data
	if len(metrics.OEM) == 0 {
		g.logger.Debug("no OEM data in processor metrics",
			slog.String("gpu_id", gpuProcessor.ID),
		)
		return
	}

	var oemData map[string]interface{}
	if err := json.Unmarshal(metrics.OEM, &oemData); err != nil {
		g.logger.Warn("failed to parse OEM data",
			slog.String("gpu_id", gpuProcessor.ID),
			slog.Any("error", err),
		)
		return
	}

	// Look for AccumulatedGPUContextUtilizationDuration
	// It could be at top level of OEM or under a vendor key like "Nvidia"
	var durationStr string

	// Check if it's directly in OEM
	if val, ok := oemData["AccumulatedGPUContextUtilizationDuration"].(string); ok {
		durationStr = val
	} else {
		// Check under vendor keys (e.g., "Nvidia", "NVIDIA", etc.)
		for vendorKey, vendorData := range oemData {
			if vendorMap, ok := vendorData.(map[string]interface{}); ok {
				if val, ok := vendorMap["AccumulatedGPUContextUtilizationDuration"].(string); ok {
					durationStr = val
					g.logger.Debug("found GPU context utilization in OEM vendor section",
						slog.String("vendor", vendorKey),
						slog.String("gpu_id", gpuProcessor.ID),
					)
					break
				}
			}
		}
	}

	if durationStr == "" {
		g.logger.Debug("AccumulatedGPUContextUtilizationDuration not found in OEM data",
			slog.String("gpu_id", gpuProcessor.ID),
		)
		return
	}

	// Parse ISO 8601 duration using sosodev/duration library
	duration, err := isoDuration.Parse(durationStr)
	if err != nil {
		g.logger.Warn("failed to parse GPU context utilization duration",
			slog.String("gpu_id", gpuProcessor.ID),
			slog.String("duration", durationStr),
			slog.Any("error", err),
		)
		return
	}

	// Convert to seconds (as float64 for Prometheus)
	seconds := duration.ToTimeDuration().Seconds()

	g.logger.Debug("collected GPU context utilization",
		slog.String("gpu_id", gpuProcessor.ID),
		slog.String("duration_str", durationStr),
		slog.Float64("seconds", seconds),
	)

	// Emit the metric as a counter (since it's accumulated time)
	labels := []string{systemName, systemID, gpuProcessor.ID}

	ch <- prometheus.MustNewConstMetric(
		g.metrics["gpu_context_utilization_seconds_total"].desc,
		prometheus.CounterValue,
		seconds,
		labels...,
	)
}

// extractGPUID extracts the GPU ID from a memory or other component ID
// e.g., "GPU_0_DRAM_0" -> "GPU_0"
func extractGPUID(componentID string) string {
	parts := strings.Split(componentID, "_")
	if len(parts) >= 2 && parts[0] == "GPU" {
		return fmt.Sprintf("GPU_%s", parts[1])
	}
	return componentID
}

// Helper types and functions from main branch

// MemoryWithMetrics interface for accessing memory metrics
type MemoryWithMetrics interface {
	Metrics() (*redfish.MemoryMetrics, error)
	GetID() string
}

// redfishMemoryAdapter adapts redfish.Memory to MemoryWithMetrics interface
type redfishMemoryAdapter struct {
	*redfish.Memory
}

func (r *redfishMemoryAdapter) GetID() string {
	return r.ID
}

// filterGPUs filters processors to return only GPU processors
func filterGPUs(cpus []*redfish.Processor) []*redfish.Processor {
	gpus := []*redfish.Processor{}
	for _, cpu := range cpus {
		if cpu.ProcessorType == redfish.GPUProcessorType {
			gpus = append(gpus, cpu)
		}
	}
	return gpus
}

// emitGPUHealth emits GPU health metrics
func emitGPUHealth(ch chan<- prometheus.Metric, gpu *redfish.Processor, commonLabels []string, metrics map[string]Metric) {
	if gpuStatusHealthValue, ok := parseCommonStatusHealth(gpu.Status.Health); ok {
		ch <- prometheus.MustNewConstMetric(
			metrics["gpu_health"].desc,
			prometheus.GaugeValue,
			gpuStatusHealthValue,
			commonLabels...)
	}
}

// emitGPUECCMetrics emits GPU ECC memory error metrics
func emitGPUECCMetrics(ch chan<- prometheus.Metric, mem []MemoryWithMetrics, logger *slog.Logger, commonLabels []string, metrics map[string]Metric) {
	for _, m := range mem {
		memMetric, err := m.Metrics()
		if err != nil {
			logger.Error("error getting gpu memory metrics", slog.Any("error", err))
			continue
		}
		metricLabels := append(commonLabels, m.GetID())

		ch <- prometheus.MustNewConstMetric(
			metrics["gpu_memory_ecc_correctable"].desc,
			prometheus.CounterValue,
			float64(memMetric.CurrentPeriod.CorrectableECCErrorCount),
			metricLabels...)
		ch <- prometheus.MustNewConstMetric(
			metrics["gpu_memory_ecc_uncorrectable"].desc,
			prometheus.CounterValue,
			float64(memMetric.CurrentPeriod.UncorrectableECCErrorCount),
			metricLabels...)
	}
}

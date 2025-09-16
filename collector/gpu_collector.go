package collector

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
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
	addToMetricMap(gpuMetrics, GPUSubsystem, "sm_utilization_percent", "GPU SM utilization percentage", gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "sm_activity_percent", "GPU SM activity percentage", gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "sm_occupancy_percent", "GPU SM occupancy percentage", gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "tensor_core_activity_percent", "GPU tensor core activity percentage", gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "fp16_activity_percent", "GPU FP16 activity percentage", gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "fp32_activity_percent", "GPU FP32 activity percentage", gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "fp64_activity_percent", "GPU FP64 activity percentage", gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "sram_ecc_error_threshold_exceeded", "GPU SRAM ECC error threshold exceeded (1 if exceeded)", gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "pcie_rx_bytes", "GPU PCIe receive bytes", gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "pcie_tx_bytes", "GPU PCIe transmit bytes", gpuProcessorLabels)

	// NVLink Port metrics - only the two requested metrics
	addToMetricMap(gpuMetrics, GPUSubsystem, "nvlink_runtime_error", "NVLink runtime error status (1 if error)", gpuPortLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "nvlink_training_error", "NVLink training error status (1 if error)", gpuPortLabels)

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

	wg := &sync.WaitGroup{}
	for _, system := range systems {
		wg.Add(1)
		go g.collectSystemGPUs(ch, system, wg)
	}
	wg.Wait()

	g.collectorScrapeStatus.WithLabelValues("gpu").Set(float64(1))
}

// collectSystemGPUs collects all GPU-related metrics for a system
func (g *GPUCollector) collectSystemGPUs(ch chan<- prometheus.Metric, system *redfish.ComputerSystem, wg *sync.WaitGroup) {
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

	// Collect detailed OEM metrics (from HEAD branch)
	// Collect GPU memory metrics from system level
	wgMemory := &sync.WaitGroup{}
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
				go g.collectGPUMemory(ch, systemName, systemID, memory, wgMemory)
			}
		}
	}

	// Collect GPU processor metrics (reusing processors from line 182)
	wgProcessor := &sync.WaitGroup{}
	for _, processor := range processors {
		// Collect metrics for any GPU processor
		if processor.ProcessorType == redfish.GPUProcessorType {
			wgProcessor.Add(1)
			go g.collectGPUProcessor(ch, systemName, systemID, processor, wgProcessor)
		}
	}

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
	if metrics, err := processor.Metrics(); err == nil && metrics != nil {
		if metricsOEM, err := g.oemClient.GetProcessorMetricsOEMData(metrics.ODataID); err == nil {
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_sm_utilization_percent"].desc,
				prometheus.GaugeValue,
				metricsOEM.SMUtilizationPercent,
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_sm_activity_percent"].desc,
				prometheus.GaugeValue,
				metricsOEM.SMActivityPercent,
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_sm_occupancy_percent"].desc,
				prometheus.GaugeValue,
				metricsOEM.SMOccupancyPercent,
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_tensor_core_activity_percent"].desc,
				prometheus.GaugeValue,
				metricsOEM.TensorCoreActivityPercent,
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_fp16_activity_percent"].desc,
				prometheus.GaugeValue,
				metricsOEM.FP16ActivityPercent,
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_fp32_activity_percent"].desc,
				prometheus.GaugeValue,
				metricsOEM.FP32ActivityPercent,
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_fp64_activity_percent"].desc,
				prometheus.GaugeValue,
				metricsOEM.FP64ActivityPercent,
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_sram_ecc_error_threshold_exceeded"].desc,
				prometheus.GaugeValue,
				boolToFloat64(metricsOEM.SRAMECCErrorThresholdExceeded),
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_pcie_rx_bytes"].desc,
				prometheus.GaugeValue,
				float64(metricsOEM.PCIeRXBytes),
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_pcie_tx_bytes"].desc,
				prometheus.GaugeValue,
				float64(metricsOEM.PCIeTXBytes),
				labels...,
			)
		} else {
			g.logger.Error("failed to get ProcessorMetrics OEM data",
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
	// Directly fetch NVLink_10 metrics as requested
	// You specified GPU_0/Ports/NVLink_10 and GPU_1/Ports/NVLink_10
	portID := "NVLink_10"
	metricsURL := fmt.Sprintf("%s/Ports/%s/Metrics", processor.ODataID, portID)

	metricsOEM, err := g.oemClient.GetPortMetricsOEMData(metricsURL)
	if err != nil {
		g.logger.Debug("no metrics found for NVLink_10",
			slog.String("processor_id", processor.ID),
			slog.String("url", metricsURL),
		)
		return
	}

	// Port exists and has metrics, emit only the two requested error metrics
	portType := "NVLink"
	portProtocol := "NVLink"
	labels := []string{systemName, systemID, gpuID, portID, portType, portProtocol}

	// Emit only the two requested NVLink error metrics
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


package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/LambdaLabs/redfish_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	isoDuration "github.com/sosodev/duration"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/common"
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
	gpuInfoLabels      = baseWithExtraLabels([]string{"serial_number", "uuid"})

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
	config                config.GPUCollectorConfig
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
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_ecc_correctable", "current correctable memory ecc errors reported on the gpu", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "memory_ecc_uncorrectable", "current uncorrectable memory ecc errors reported on the gpu", gpuMemoryLabels)

	// GPU info metric
	addToMetricMap(gpuMetrics, GPUSubsystem, "info", "GPU information with serial number and UUID", gpuInfoLabels)

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
	addToMetricMap(gpuMetrics, GPUSubsystem, "state", fmt.Sprintf("GPU processor state,%s", CommonStateHelp), gpuProcessorLabels)
	addToMetricMap(gpuMetrics, GPUSubsystem, "health", fmt.Sprintf("GPU processor health,%s", CommonHealthHelp), gpuProcessorLabels)
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

	// GPU Context Utilization metrics
	// Note: GPU temperature and memory power metrics are now collected via TelemetryService (HGX_PlatformEnvironmentMetrics_0)
	addToMetricMap(gpuMetrics, GPUSubsystem, "context_utilization_seconds_total", "Accumulated GPU context utilization duration in seconds", gpuBaseLabels)

	return gpuMetrics
}

// NewGPUCollector creates a new GPU collector
func NewGPUCollector(collectorName string, redfishClient *gofish.APIClient, logger *slog.Logger, config config.GPUCollectorConfig) (*GPUCollector, error) {
	return &GPUCollector{
		redfishClient: redfishClient,
		metrics:       gpuMetrics,
		config:        config,
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
	}, nil
}

// Describe implements prometheus.Collector
func (g *GPUCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range g.metrics {
		ch <- metric.desc
	}
	g.collectorScrapeStatus.Describe(ch)
}

func (g *GPUCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
	g.collect(ctx, ch)
}

// Collect implements prometheus.Collector
func (g *GPUCollector) Collect(ch chan<- prometheus.Metric) {
	g.collect(context.TODO(), ch)
}

func (g *GPUCollector) collect(ctx context.Context, ch chan<- prometheus.Metric) {
	if ctx.Err() != nil {
		g.logger.With("error", ctx.Err().Error()).Debug("skipping gpu collection")
		return
	}
	g.collectorScrapeStatus.WithLabelValues("gpu").Set(float64(0))

	//	service := g.redfishClient.Service
	//	systems, err := service.Systems()
	//	if err != nil {
	//		g.logger.Error("failed getting systems",
	//			slog.Any("error", err),
	//			slog.String("operation", "service.Systems()"),
	//		)
	//		return
	//	}

	gpus, err := g.gatherGPUs(ctx)
	if err != nil {
		g.logger.With("error", err, "operation", "gatherGPUs").Error("unable to gather gpus")
		return
	}

	// Channel to collect GPU info from each system
	//	gpuInfoChan := make(chan systemGPUInfo, len(systems))
	//	wg := &sync.WaitGroup{}
	//
	//	// Track UUIDs to detect duplicates
	//	uuidTracker := &sync.Map{} // map[uuid]gpuID
	//
	//	for _, system := range systems {
	//		if ctx.Err() != nil {
	//			g.logger.With("error", ctx.Err().Error()).Debug("skipping gpu collection as context is errored")
	//			continue
	//		}
	//		wg.Add(1)
	//		go g.collectSystemGPUs(ctx, ch, system, wg, gpuInfoChan, uuidTracker)
	//	}

	for systemName, systemGPUs := range gpus {
		for _, gpu := range systemGPUs {
			if ctx.Err() != nil {
				g.logger.With("error", ctx.Err().Error()).Debug("skipping further gpu collection")
				return
			}
			commonLabels := []string{gpu.Name, systemName, gpu.ID}
			// NOTE(mfuller): NOT emitting gpu_health as it is duplicated with gpu_processor_* metrics further below
			// emitGPUECCMetrics
			gpuMems, err := gpu.Memory()
			if err != nil {
				g.logger.With("error", err, "gpu_id", gpu.ID, "system_name", systemName).Error("failed obtaining gpu memory, skipping")
				continue
			}
			for _, mem := range gpuMems {
				memMetric, err := mem.Metrics()
				if err != nil {
					g.logger.With("error", err, "gpu_id", gpu.ID, "memory_id", mem.ID, "system_name", systemName).Error("failed obtaining gpu memory metrics, skipping")
					continue
				}
				memLabels := make([]string, len(commonLabels))
				copy(memLabels, commonLabels)
				memLabels = append(memLabels, mem.ID)

				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_memory_ecc_correctable"].desc,
					prometheus.CounterValue,
					float64(memMetric.CurrentPeriod.CorrectableECCErrorCount),
					memLabels...)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_memory_ecc_uncorrectable"].desc,
					prometheus.CounterValue,
					float64(memMetric.CurrentPeriod.UncorrectableECCErrorCount),
					memLabels...)
				// collectGPUMemory
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_memory_capacity_mib"].desc,
					prometheus.GaugeValue,
					float64(mem.CapacityMiB),
					[]string{systemName, "", gpu.ID, mem.ID}...,
				)
				if stateValue, ok := parseCommonStatusState(mem.Status.State); ok {
					ch <- prometheus.MustNewConstMetric(
						g.metrics["gpu_memory_state"].desc,
						prometheus.GaugeValue,
						stateValue,
						memLabels...,
					)
				}
				if healthValue, ok := parseCommonStatusHealth(mem.Status.Health); ok {
					ch <- prometheus.MustNewConstMetric(
						g.metrics["gpu_memory_health"].desc,
						prometheus.GaugeValue,
						healthValue,
						memLabels...,
					)
				}
				// collectGPUMemory->OEM Metrics
				var oemMem MemoryMetricsOEMData
				if err := json.Unmarshal(memMetric.OEM, &oemMem); err != nil {
					g.logger.With("error", err).Debug("unable to unmarshal OEM memory")
					continue
				}
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_memory_correctable_row_remapping_count"].desc,
					prometheus.GaugeValue,
					float64(oemMem.CorrectableRowRemappingCount),
					memLabels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_memory_uncorrectable_row_remapping_count"].desc,
					prometheus.GaugeValue,
					float64(oemMem.UncorrectableRowRemappingCount),
					memLabels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_memory_high_availability_bank_count"].desc,
					prometheus.GaugeValue,
					float64(oemMem.HighAvailabilityBankCount),
					memLabels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_memory_low_availability_bank_count"].desc,
					prometheus.GaugeValue,
					float64(oemMem.LowAvailabilityBankCount),
					memLabels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_memory_no_availability_bank_count"].desc,
					prometheus.GaugeValue,
					float64(oemMem.NoAvailabilityBankCount),
					memLabels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_memory_partial_availability_bank_count"].desc,
					prometheus.GaugeValue,
					float64(oemMem.PartialAvailabilityBankCount),
					memLabels...,
				)
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_memory_max_availability_bank_count"].desc,
					prometheus.GaugeValue,
					float64(oemMem.MaxAvailabilityBankCount),
					memLabels...,
				)
			}
			// collectGPUProcessor
			// TODO(mfuller): Should drop the last label here, processor ID and Name are repetitive
			procBaseLabels := []string{systemName, "FIXME", gpu.ID, gpu.ID}
			// NOTE (0, mfuller): No longer emitting cores/threads series, GB200/GB300, B200 do not support it.
			// In fact, no system at Lambda seems to emit this as a nonzero value.
			// NOTE (1, mfuller): Consolidating and renaming the `gpu_health` and `gpu_processor_foo` series to just the two here.
			if stateValue, ok := parseCommonStatusState(gpu.Status.State); ok {
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_state"].desc,
					prometheus.GaugeValue,
					stateValue,
					procBaseLabels...,
				)
			}
			if healthValue, ok := parseCommonStatusHealth(gpu.Status.Health); ok {
				ch <- prometheus.MustNewConstMetric(
					g.metrics["gpu_health"].desc,
					prometheus.GaugeValue,
					healthValue,
					procBaseLabels...,
				)
			}
			// NOTE(mfuller): Always emit this, if value are unknown then just say that
			var gpuSerial, gpuUUID string
			if gpuSerial = gpu.SerialNumber; gpuSerial == "" {
				gpuSerial = "unknown"
			}
			if gpuUUID = gpu.UUID; gpuUUID == "" {
				gpuUUID = "unknown"
			}
			infoLabels := []string{systemName, "", gpu.ID, gpuSerial, gpuUUID}
			ch <- prometheus.MustNewConstMetric(
				g.metrics["gpu_info"].desc,
				prometheus.GaugeValue,
				1,
				infoLabels...,
			)

			// TODO: collectGPUProcessor -> OEM
			gpuOEMMetrics, err := gpu.Metrics()
			if err != nil {
				g.logger.With("error", err, "gpu_id", gpu.ID, "system_name", systemName).Error("failed obtaining gpu processor metrics, skipping")
			} else {
				var gpuOEM struct {
					Nvidia ProcessorMetricsOEMData `json:"Nvidia"`
				}
				if err := json.Unmarshal(gpuOEMMetrics.OEM, &gpuOEM); err != nil {
					g.logger.With("error", err, "gpu_id", gpu.ID, "system_name", systemName).Error("failed unmarshaling gpu processor metrics, skipping")
				} else {
					ch <- prometheus.MustNewConstMetric(
						g.metrics["gpu_sram_ecc_error_threshold_exceeded"].desc,
						prometheus.GaugeValue,
						boolToFloat64(gpuOEM.Nvidia.SRAMECCErrorThresholdExceeded),
						procBaseLabels...,
					)
					// NOTE(mfuller): GPU context utilization
					if gpuOEM.Nvidia.AccumulatedGPUContextUtilizationDuration != "" {
						duration, err := isoDuration.Parse(gpuOEM.Nvidia.AccumulatedGPUContextUtilizationDuration)
						if err != nil {
							g.logger.With("error", err, "gpu_id", gpu.ID, "raw_duration", duration).Warn("unable to parse gpu context duration, setting to zero")
							duration = &isoDuration.Duration{
								Seconds: 0,
							}
						}
						labels := []string{systemName, "foo", gpu.ID}
						ch <- prometheus.MustNewConstMetric(
							g.metrics["gpu_context_utilization_seconds_total"].desc,
							prometheus.CounterValue,
							duration.Seconds,
							labels...,
						)
					}
				}
			}
			//   TODO: collectGPUProcessor -> collectNVLinkPorts
			// NOTE(mfuller): Instead of for every port calling RF API, drop down
			// to a direct client and use expansion.
			rfClient := g.redfishClient.WithContext(ctx)
			rfPath := fmt.Sprintf(`%s/Ports?$expand=.($levels=2)`, gpu.ODataID)
			response, err := rfClient.Get(rfPath)
			if err != nil {

				g.logger.With("error", err, "gpu_id", gpu.ID, "system_name", systemName).Error("unable to gather NVLink data, skipping")
			} else {
				type aggregateNVLinkData struct {
					ODataID   string `json:"@odata.id"`
					ODataType string `json:"@odata.type"`
					Members   []struct {
						ID      string `json:"Id"`
						Metrics struct {
							Oem struct {
								NVidiaOEM struct {
									OdataType string `json:"@odata.type,omitempty"`
									// PCIe-specific fields
									RXErrorsPerLane []int `json:"RXErrorsPerLane,omitempty"`
									// NVLink-specific fields
									BitErrorRate              float64 `json:"BitErrorRate,omitempty"`
									EffectiveBER              float64 `json:"EffectiveBER,omitempty"`
									EffectiveError            int     `json:"EffectiveError,omitempty"`
									IntentionalLinkDownCount  int     `json:"IntentionalLinkDownCount,omitempty"`
									LinkDownReasonCode        string  `json:"LinkDownReasonCode,omitempty"`
									LinkDownedCount           int     `json:"LinkDownedCount,omitempty"`
									LinkErrorRecoveryCount    int     `json:"LinkErrorRecoveryCount,omitempty"`
									MalformedPackets          int     `json:"MalformedPackets,omitempty"`
									NVLinkDataRxBandwidthGbps float64 `json:"NVLinkDataRxBandwidthGbps,omitempty"`
									NVLinkDataTxBandwidthGbps float64 `json:"NVLinkDataTxBandwidthGbps,omitempty"`
									NVLinkErrors              struct {
										RuntimeError  bool `json:"RuntimeError"`
										TrainingError bool `json:"TrainingError"`
									} `json:"NVLinkErrors,omitempty"`
									NVLinkRawRxBandwidthGbps   float64 `json:"NVLinkRawRxBandwidthGbps,omitempty"`
									NVLinkRawTxBandwidthGbps   float64 `json:"NVLinkRawTxBandwidthGbps,omitempty"`
									RXNoProtocolBytes          int64   `json:"RXNoProtocolBytes,omitempty"`
									SymbolErrors               int     `json:"SymbolErrors,omitempty"`
									TXNoProtocolBytes          int64   `json:"TXNoProtocolBytes,omitempty"`
									TXWait                     int     `json:"TXWait,omitempty"`
									TotalRawBER                float64 `json:"TotalRawBER,omitempty"`
									TotalRawError              int     `json:"TotalRawError,omitempty"`
									UnintentionalLinkDownCount int     `json:"UnintentionalLinkDownCount,omitempty"`
									VL15Dropped                int     `json:"VL15Dropped,omitempty"`
									VL15TXBytes                int     `json:"VL15TXBytes,omitempty"`
									VL15TXPackets              int     `json:"VL15TXPackets,omitempty"`
								} `json:"Nvidia,omittempty"`
							} `json:"Oem"`
						} `json:"Metrics"`
						PortType     string               `json:"PortType"`
						PortProtocol redfish.PortProtocol `json:"PortProtocol"`
						Status       struct {
							Health common.Health `json:"Health"`
							State  common.State  `json:"State"`
						} `json:"Status"`
					} `json:"Members"`
				}
				body, err := io.ReadAll(response.Body)
				if err != nil {
					g.logger.With("error", err, "gpu_id", gpu.ID, "system_name", systemName).Error("unable to read in NVLink data, skipping")
				} else {
					agg := &aggregateNVLinkData{}
					if err := json.Unmarshal(body, agg); err != nil {
						g.logger.With("error", err, "gpu_id", gpu.ID, "system_name", systemName).Error("unable to unmarshal NVLink data, skipping")
					} else {
						for _, port := range agg.Members {
							if port.PortProtocol != redfish.NVLinkPortProtocol ||
								!strings.Contains(port.ID, "NVLink_") {
								continue
							}
							strProto := string(port.PortProtocol)
							portLabels := []string{systemName, "FIXME", gpu.ID, port.ID, port.PortType, strProto}
							// Common
							if stateValue, ok := parseCommonStatusState(port.Status.State); ok {
								ch <- prometheus.MustNewConstMetric(
									g.metrics["gpu_nvlink_state"].desc,
									prometheus.GaugeValue,
									stateValue,
									portLabels...,
								)
							}
							if healthValue, ok := parseCommonStatusHealth(port.Status.Health); ok {
								ch <- prometheus.MustNewConstMetric(
									g.metrics["gpu_nvlink_health"].desc,
									prometheus.GaugeValue,
									healthValue,
									portLabels...,
								)
							}

							// Get PortMetrics OEM data
							ch <- prometheus.MustNewConstMetric(
								g.metrics["gpu_nvlink_runtime_error"].desc,
								prometheus.GaugeValue,
								boolToFloat64(port.Metrics.Oem.NVidiaOEM.NVLinkErrors.RuntimeError),
								portLabels...,
							)
							ch <- prometheus.MustNewConstMetric(
								g.metrics["gpu_nvlink_training_error"].desc,
								prometheus.GaugeValue,
								boolToFloat64(port.Metrics.Oem.NVidiaOEM.NVLinkErrors.TrainingError),
								portLabels...,
							)
							ch <- prometheus.MustNewConstMetric(
								g.metrics["gpu_nvlink_link_error_recovery_count"].desc,
								prometheus.GaugeValue,
								float64(port.Metrics.Oem.NVidiaOEM.LinkErrorRecoveryCount),
								portLabels...,
							)
							ch <- prometheus.MustNewConstMetric(
								g.metrics["gpu_nvlink_link_downed_count"].desc,
								prometheus.GaugeValue,
								float64(port.Metrics.Oem.NVidiaOEM.LinkDownedCount),
								portLabels...,
							)
							ch <- prometheus.MustNewConstMetric(
								g.metrics["gpu_nvlink_symbol_errors"].desc,
								prometheus.GaugeValue,
								float64(port.Metrics.Oem.NVidiaOEM.SymbolErrors),
								portLabels...,
							)
							ch <- prometheus.MustNewConstMetric(
								g.metrics["gpu_nvlink_bit_error_rate"].desc,
								prometheus.GaugeValue,
								port.Metrics.Oem.NVidiaOEM.BitErrorRate,
								portLabels...,
							)

						}
					}
				}
			}
		}

	}

	//	// Close channel after all systems are processed
	//	go func() {
	//		wg.Wait()
	//		close(gpuInfoChan)
	//	}()
	//
	//	if ctx.Err() != nil {
	//		g.logger.With("error", ctx.Err().Error()).Debug("skipping gpu collection as context is errored")
	//		return
	//	}
	// Gather all GPU info while systems are being processed
	//	var allSystemGPUs []systemGPUInfo
	//	for info := range gpuInfoChan {
	//		if len(info.gpus) > 0 {
	//			allSystemGPUs = append(allSystemGPUs, info)
	//			g.logger.Debug("Collected GPU info",
	//				slog.String("system", info.systemName),
	//				slog.Int("gpu_count", len(info.gpus)))
	//		}
	//	}

	// Collect GPU context utilization metrics
	// Note: GPU temperature and memory power are now collected via TelemetryService (HGX_PlatformEnvironmentMetrics_0)
	//g.collectGPUContextUtilization(ctx, ch, allSystemGPUs)

	// TODO: GPU Context Utilization

	g.collectorScrapeStatus.WithLabelValues("gpu").Set(float64(1))
}

func (g *GPUCollector) gatherGPUs(ctx context.Context) (map[string][]*redfish.Processor, error) {
	ret := make(map[string][]*redfish.Processor)
	if ctx.Err() != nil {
		return ret, ctx.Err()
	}
	systems, err := g.redfishClient.Service.Systems()
	if err != nil {
		return ret, fmt.Errorf("unable to obtain systems data: %w", err)
	}

	for _, sys := range systems {
		if strings.Contains(sys.Name, "HGX_") {
			procs, err := sys.Processors()
			if err != nil {
				return ret, fmt.Errorf("unable to obtain system processors: %w", err)
			}
			gpus := filterGPUs(procs)
			ret[sys.Name] = gpus
		}
	}

	return ret, nil
}

// collectSystemGPUs collects all GPU-related metrics for a system
func (g *GPUCollector) collectSystemGPUs(ctx context.Context, ch chan<- prometheus.Metric, system *redfish.ComputerSystem, wg *sync.WaitGroup, gpuInfoChan chan<- systemGPUInfo, uuidTracker *sync.Map) {
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
				g.collectGPUProcessor(ctx, ch, systemName, systemID, proc, wgProcessor, uuidTracker)
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
func (g *GPUCollector) collectGPUProcessor(ctx context.Context, ch chan<- prometheus.Metric, systemName, systemID string, processor *redfish.Processor, wg *sync.WaitGroup, uuidTracker *sync.Map) {
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

	// Collect GPU info metric with serial number and UUID
	serialNumber := processor.SerialNumber
	uuid := processor.UUID

	// Log if either is missing
	if serialNumber == "" {
		g.logger.Debug("GPU has no serial number",
			slog.String("processor_id", processorID),
			slog.String("system_id", systemID))
	}
	if uuid == "" {
		g.logger.Warn("GPU has no UUID",
			slog.String("processor_id", processorID),
			slog.String("system_id", systemID))
	}

	// Only emit metric if at least one identifier is present
	if serialNumber != "" || uuid != "" {
		// Check for UUID uniqueness - UUIDs must always be unique
		if uuid != "" {
			if existingGPUID, exists := uuidTracker.LoadOrStore(uuid, processorID); exists {
				g.logger.Error("duplicate GPU UUID detected - UUIDs must be unique",
					slog.String("uuid", uuid),
					slog.String("processor_id", processorID),
					slog.String("duplicate_processor_id", existingGPUID.(string)),
					slog.String("system_id", systemID))
			}
		}

		infoLabels := []string{systemName, systemID, processorID, serialNumber, uuid}
		ch <- prometheus.MustNewConstMetric(
			g.metrics["gpu_info"].desc,
			prometheus.GaugeValue,
			1, // Info metrics always have value 1
			infoLabels...,
		)
	}

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
	g.collectNVLinkPorts(ctx, ch, systemName, systemID, processorID, processor)

}

// collectNVLinkPorts collects NVLink port metrics for a GPU processor
func (g *GPUCollector) collectNVLinkPorts(ctx context.Context, ch chan<- prometheus.Metric, systemName, systemID, gpuID string, processor *redfish.Processor) {
	if ctx.Err() != nil {
		return
	}

	ports, err := processor.Ports()
	if err != nil {
		g.logger.Error("failed to get ports",
			slog.String("gpu_id", gpuID),
			slog.String("system_id", systemID),
			slog.Any("error", err))
		return
	}

	for _, port := range ports {
		if ctx.Err() != nil {
			continue
		}
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

// collectGPUContextUtilization collects accumulated GPU context utilization duration from ProcessorMetrics
func (g *GPUCollector) collectGPUContextUtilization(ctx context.Context, ch chan<- prometheus.Metric, allSystemGPUs []systemGPUInfo) {
	wg := &sync.WaitGroup{}
	// Limit concurrent requests
	sem := make(chan struct{}, 5)

	for _, sysInfo := range allSystemGPUs {
		if ctx.Err() != nil {
			g.logger.With("error", ctx.Err().Error()).Debug("skipping gpu collection as context is errored")
			return
		}
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

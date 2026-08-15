package collector

import (
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish/schemas"
)

// This file implements the GPU collector's enable_gpu_module_telemetry mode: emitting the
// redfish_telemetry_* metric families from the per-GPU ProcessorMetrics and
// MemoryMetrics resources the collector already fetches, instead of the
// NVIDIA-format HGX_* MetricReports the telemetry module consumes. It exists
// for platforms whose BMCs do not publish those reports (e.g. Dell XE9780
// iDRAC proxying the HGX tray) yet expose the identical data on the
// per-resource endpoints. The metric contract is shared with the
// TelemetryCollector (telemetryMetrics and its emit helpers), so series are
// indistinguishable regardless of which collection path ran.

// maxGPMInstanceID mirrors the telemetry_collector's report-path validation,
// which only accepts NVDec/NVJpg instance IDs 0-7.
const maxGPMInstanceID = 7

type gpuTelemetryProcessorOEMNvidia struct {
	SMActivityPercent                       *float64   `json:"SMActivityPercent"`
	SMOccupancyPercent                      *float64   `json:"SMOccupancyPercent"`
	TensorCoreActivityPercent               *float64   `json:"TensorCoreActivityPercent"`
	DMMAUtilizationPercent                  *float64   `json:"DMMAUtilizationPercent"`
	HMMAUtilizationPercent                  *float64   `json:"HMMAUtilizationPercent"`
	IMMAUtilizationPercent                  *float64   `json:"IMMAUtilizationPercent"`
	IntegerActivityUtilizationPercent       *float64   `json:"IntegerActivityUtilizationPercent"`
	FP16ActivityPercent                     *float64   `json:"FP16ActivityPercent"`
	FP32ActivityPercent                     *float64   `json:"FP32ActivityPercent"`
	FP64ActivityPercent                     *float64   `json:"FP64ActivityPercent"`
	GraphicsEngineActivityPercent           *float64   `json:"GraphicsEngineActivityPercent"`
	NVDecUtilizationPercent                 *float64   `json:"NVDecUtilizationPercent"`
	NVJpgUtilizationPercent                 *float64   `json:"NVJpgUtilizationPercent"`
	NVOfaUtilizationPercent                 *float64   `json:"NVOfaUtilizationPercent"`
	NVDecInstanceUtilizationPercent         []*float64 `json:"NVDecInstanceUtilizationPercent"`
	NVJpgInstanceUtilizationPercent         []*float64 `json:"NVJpgInstanceUtilizationPercent"`
	NVLinkDataRxBandwidthGbps               *float64   `json:"NVLinkDataRxBandwidthGbps"`
	NVLinkDataTxBandwidthGbps               *float64   `json:"NVLinkDataTxBandwidthGbps"`
	NVLinkRawRxBandwidthGbps                *float64   `json:"NVLinkRawRxBandwidthGbps"`
	NVLinkRawTxBandwidthGbps                *float64   `json:"NVLinkRawTxBandwidthGbps"`
	PCIeRawRxBandwidthGbps                  *float64   `json:"PCIeRawRxBandwidthGbps"`
	PCIeRawTxBandwidthGbps                  *float64   `json:"PCIeRawTxBandwidthGbps"`
	HardwareViolationThrottleDuration       string     `json:"HardwareViolationThrottleDuration"`
	GlobalSoftwareViolationThrottleDuration string     `json:"GlobalSoftwareViolationThrottleDuration"`
}

// gpuTelemetryProcessorOEM is the OEM envelope of a ProcessorMetrics resource.
type gpuTelemetryProcessorOEM struct {
	Nvidia gpuTelemetryProcessorOEMNvidia `json:"Nvidia"`
}

// emitTelemetryFromProcessorMetrics emits the redfish_telemetry_* processor families
// (cache ECC, PCIe errors, etc.) from a GPU's already-fetched ProcessorMetrics resource.
func (g *GPUCollector) emitTelemetryFromProcessorMetrics(ch chan<- prometheus.Metric, gpu SystemGPU, pm *schemas.ProcessorMetrics) {
	if pm == nil {
		return
	}
	var oem gpuTelemetryProcessorOEM
	if len(pm.OEM) > 0 {
		if err := json.Unmarshal(pm.OEM, &oem); err != nil {
			g.logger.With("error", err, "gpu_id", gpu.ID, "system_id", gpu.SystemID).Debug("unable to unmarshal processor metrics OEM data")
		}
	}

	labels := []string{gpu.SystemID, gpu.ID}
	emitGPUMetrics(ch, telemetryMetrics, labels, g.processorTelemetryValues(pm, &oem.Nvidia))
	emitGPMMetrics(ch, telemetryMetrics, labels, gpmTelemetryValues(&oem.Nvidia))
	g.emitGPMInstanceTelemetry(ch, gpu.SystemID, gpu.ID, "NVDecInstanceUtilizationPercent", oem.Nvidia.NVDecInstanceUtilizationPercent)
	g.emitGPMInstanceTelemetry(ch, gpu.SystemID, gpu.ID, "NVJpgInstanceUtilizationPercent", oem.Nvidia.NVJpgInstanceUtilizationPercent)
}

func (g *GPUCollector) processorTelemetryValues(pm *schemas.ProcessorMetrics, oem *gpuTelemetryProcessorOEMNvidia) map[string]float64 {
	values := make(map[string]float64)

	addInt := func(key string, v *int) {
		if v != nil {
			values[key] = float64(*v)
		}
	}
	addInt("CacheMetricsTotal/LifeTime/CorrectableECCErrorCount", pm.CacheMetricsTotal.LifeTime.CorrectableECCErrorCount)
	addInt("CacheMetricsTotal/LifeTime/UncorrectableECCErrorCount", pm.CacheMetricsTotal.LifeTime.UncorrectableECCErrorCount)

	addInt("PCIeErrors/CorrectableErrorCount", pm.PCIeErrors.CorrectableErrorCount)
	addInt("PCIeErrors/NonFatalErrorCount", pm.PCIeErrors.NonFatalErrorCount)
	addInt("PCIeErrors/FatalErrorCount", pm.PCIeErrors.FatalErrorCount)
	addInt("PCIeErrors/L0ToRecoveryCount", pm.PCIeErrors.L0ToRecoveryCount)
	addInt("PCIeErrors/ReplayCount", pm.PCIeErrors.ReplayCount)
	addInt("PCIeErrors/ReplayRolloverCount", pm.PCIeErrors.ReplayRolloverCount)
	addInt("PCIeErrors/NAKSentCount", pm.PCIeErrors.NAKSentCount)
	addInt("PCIeErrors/NAKReceivedCount", pm.PCIeErrors.NAKReceivedCount)
	addInt("PCIeErrors/UnsupportedRequestCount", pm.PCIeErrors.UnsupportedRequestCount)

	addDuration := func(key, raw string) {
		if raw == "" {
			return
		}
		seconds, err := parseMetricValue(raw)
		if err != nil {
			g.logger.Debug("failed to parse throttle duration",
				slog.String("metric", key),
				slog.String("value", raw),
				slog.Any("error", err),
			)
			return
		}
		values[key] = seconds
	}
	addDuration("PowerLimitThrottleDuration", pm.PowerLimitThrottleDuration)
	addDuration("ThermalLimitThrottleDuration", pm.ThermalLimitThrottleDuration)
	addDuration("Oem/Nvidia/HardwareViolationThrottleDuration", oem.HardwareViolationThrottleDuration)
	addDuration("Oem/Nvidia/GlobalSoftwareViolationThrottleDuration", oem.GlobalSoftwareViolationThrottleDuration)

	return values
}

func gpmTelemetryValues(oem *gpuTelemetryProcessorOEMNvidia) map[string]float64 {
	values := make(map[string]float64)
	for key, v := range map[string]*float64{
		"SMActivityPercent":                 oem.SMActivityPercent,
		"SMOccupancyPercent":                oem.SMOccupancyPercent,
		"TensorCoreActivityPercent":         oem.TensorCoreActivityPercent,
		"DMMAUtilizationPercent":            oem.DMMAUtilizationPercent,
		"HMMAUtilizationPercent":            oem.HMMAUtilizationPercent,
		"IMMAUtilizationPercent":            oem.IMMAUtilizationPercent,
		"IntegerActivityUtilizationPercent": oem.IntegerActivityUtilizationPercent,
		"FP16ActivityPercent":               oem.FP16ActivityPercent,
		"FP32ActivityPercent":               oem.FP32ActivityPercent,
		"FP64ActivityPercent":               oem.FP64ActivityPercent,
		"GraphicsEngineActivityPercent":     oem.GraphicsEngineActivityPercent,
		"NVDecUtilizationPercent":           oem.NVDecUtilizationPercent,
		"NVJpgUtilizationPercent":           oem.NVJpgUtilizationPercent,
		"NVOfaUtilizationPercent":           oem.NVOfaUtilizationPercent,
		"NVLinkDataRxBandwidthGbps":         oem.NVLinkDataRxBandwidthGbps,
		"NVLinkDataTxBandwidthGbps":         oem.NVLinkDataTxBandwidthGbps,
		"NVLinkRawRxBandwidthGbps":          oem.NVLinkRawRxBandwidthGbps,
		"NVLinkRawTxBandwidthGbps":          oem.NVLinkRawTxBandwidthGbps,
		"PCIeRawRxBandwidthGbps":            oem.PCIeRawRxBandwidthGbps,
		"PCIeRawTxBandwidthGbps":            oem.PCIeRawTxBandwidthGbps,
	} {
		if v != nil {
			values[key] = *v
		}
	}
	return values
}

func (g *GPUCollector) emitGPMInstanceTelemetry(ch chan<- prometheus.Metric, systemID, gpuID, metricName string, instances []*float64) {
	for i, v := range instances {
		if i > maxGPMInstanceID {
			g.logger.Debug("invalid instance ID",
				slog.String("metric", metricName),
				slog.String("gpu_id", gpuID),
				slog.Int("instance_id", i),
			)
			continue
		}
		if v == nil {
			continue
		}
		labels := []string{systemID, gpuID, strconv.Itoa(i)}
		emitGPMInstanceMetrics(ch, telemetryMetrics, labels, map[string]float64{metricName: *v})
	}
}

func memoryTelemetryValues(mm *schemas.MemoryMetrics) map[string]float64 {
	values := make(map[string]float64)
	if v := mm.LifeTime.CorrectableECCErrorCount; v != nil {
		values["LifeTime/CorrectableECCErrorCount"] = float64(*v)
	}
	if v := mm.LifeTime.UncorrectableECCErrorCount; v != nil {
		values["LifeTime/UncorrectableECCErrorCount"] = float64(*v)
	}
	if v := mm.BandwidthPercent; v != nil {
		values["BandwidthPercent"] = *v
	}
	if v := mm.CapacityUtilizationPercent; v != nil {
		values["CapacityUtilizationPercent"] = *v
	}
	if v := mm.OperatingSpeedMHz; v != nil {
		values["OperatingSpeedMHz"] = float64(*v)
	}
	return values
}

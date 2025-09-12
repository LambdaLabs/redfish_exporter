package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/stmcginnis/gofish/common"
)

// NvidiaOEMHelper provides methods to extract Nvidia OEM fields from Redfish responses
type NvidiaOEMHelper struct {
	client common.Client
	logger *slog.Logger
}

// NewNvidiaOEMHelper creates a new helper for extracting Nvidia OEM fields
func NewNvidiaOEMHelper(client common.Client, logger *slog.Logger) *NvidiaOEMHelper {
	return &NvidiaOEMHelper{
		client: client,
		logger: logger.With(slog.String("component", "nvidia_oem")),
	}
}

// MemoryOEMMetrics represents Nvidia OEM fields from Memory endpoint
type MemoryOEMMetrics struct {
	RowRemappingFailed  bool
	RowRemappingPending bool
}

// MemoryMetricsOEMData represents Nvidia OEM fields from MemoryMetrics endpoint
type MemoryMetricsOEMData struct {
	CorrectableRowRemappingCount   int64
	HighAvailabilityBankCount      int64
	LowAvailabilityBankCount       int64
	MaxAvailabilityBankCount       int64
	NoAvailabilityBankCount        int64
	PartialAvailabilityBankCount   int64
	UncorrectableRowRemappingCount int64
}

// ProcessorMetricsOEMData represents Nvidia OEM fields from ProcessorMetrics endpoint
type ProcessorMetricsOEMData struct {
	SMUtilizationPercent              float64
	SMActivityPercent                 float64
	SMOccupancyPercent                float64
	TensorCoreActivityPercent         float64
	FP16ActivityPercent               float64
	FP32ActivityPercent               float64
	FP64ActivityPercent               float64
	IntegerActivityUtilizationPercent float64
	SRAMECCErrorThresholdExceeded     bool
	NVLinkDataRxBandwidthGbps         float64
	NVLinkDataTxBandwidthGbps         float64
	PCIeRXBytes                       int64
	PCIeTXBytes                       int64
	ThrottleReasons                   []string
}

// PortMetricsOEMData represents Nvidia OEM fields from PortMetrics endpoint
type PortMetricsOEMData struct {
	NVLinkErrorsRuntimeError   bool
	NVLinkErrorsTrainingError  bool
	LinkErrorRecoveryCount     int64
	LinkDownedCount            int64
	SymbolErrors               int64
	MalformedPackets           int64
	BitErrorRate               float64
	EffectiveBER               float64
}

// IsNvidiaGPUMemory checks if a memory ID indicates GPU memory
func IsNvidiaGPUMemory(memoryID string) bool {
	// Check if memory ID contains GPU pattern
	return strings.Contains(memoryID, "GPU_") && strings.Contains(memoryID, "_DRAM_")
}

// IsNvidiaGPU checks if a processor ID indicates an Nvidia GPU
func IsNvidiaGPU(processorID string) bool {
	return strings.Contains(processorID, "GPU_")
}

// IsNVLinkPort checks if a port ID indicates an NVLink port
func IsNVLinkPort(portID string) bool {
	return strings.Contains(portID, "NVLink_")
}

// fetchJSON fetches JSON data from the given endpoint
func (h *NvidiaOEMHelper) fetchJSON(endpoint string) (map[string]interface{}, error) {
	resp, err := h.client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", endpoint, err)
	}
	defer resp.Body.Close() // nolint:errcheck // Close() errors on read are not actionable

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode JSON from %s: %w", endpoint, err)
	}

	return data, nil
}

// parseMemoryOEMMetrics parses Nvidia OEM fields from Memory JSON data
func parseMemoryOEMMetrics(data map[string]interface{}) *MemoryOEMMetrics {
	metrics := &MemoryOEMMetrics{}
	
	// Navigate to Oem.Nvidia fields
	if oemData, ok := data["Oem"].(map[string]interface{}); ok {
		if nvidiaData, ok := oemData["Nvidia"].(map[string]interface{}); ok {
			if val, ok := nvidiaData["RowRemappingFailed"].(bool); ok {
				metrics.RowRemappingFailed = val
			}
			if val, ok := nvidiaData["RowRemappingPending"].(bool); ok {
				metrics.RowRemappingPending = val
			}
		}
	}
	
	return metrics
}

// GetMemoryOEMMetrics fetches and parses Nvidia OEM fields from Memory endpoint
func (h *NvidiaOEMHelper) GetMemoryOEMMetrics(odataID string) (*MemoryOEMMetrics, error) {
	data, err := h.fetchJSON(odataID)
	if err != nil {
		return nil, err
	}
	
	metrics := parseMemoryOEMMetrics(data)
	
	h.logger.Debug("extracted Memory OEM metrics",
		slog.String("odataID", odataID),
		slog.Bool("row_remapping_failed", metrics.RowRemappingFailed),
		slog.Bool("row_remapping_pending", metrics.RowRemappingPending))
	
	return metrics, nil
}

// parseMemoryMetricsOEMData parses Nvidia OEM fields from MemoryMetrics JSON data
func parseMemoryMetricsOEMData(data map[string]interface{}) *MemoryMetricsOEMData {
	metrics := &MemoryMetricsOEMData{}
	
	// Navigate to Oem.Nvidia.RowRemapping fields
	if oemData, ok := data["Oem"].(map[string]interface{}); ok {
		if nvidiaData, ok := oemData["Nvidia"].(map[string]interface{}); ok {
			if rowRemapping, ok := nvidiaData["RowRemapping"].(map[string]interface{}); ok {
				metrics.CorrectableRowRemappingCount = getInt64(rowRemapping, "CorrectableRowRemappingCount")
				metrics.HighAvailabilityBankCount = getInt64(rowRemapping, "HighAvailabilityBankCount")
				metrics.LowAvailabilityBankCount = getInt64(rowRemapping, "LowAvailabilityBankCount")
				metrics.MaxAvailabilityBankCount = getInt64(rowRemapping, "MaxAvailabilityBankCount")
				metrics.NoAvailabilityBankCount = getInt64(rowRemapping, "NoAvailabilityBankCount")
				metrics.PartialAvailabilityBankCount = getInt64(rowRemapping, "PartialAvailabilityBankCount")
				metrics.UncorrectableRowRemappingCount = getInt64(rowRemapping, "UncorrectableRowRemappingCount")
			}
		}
	}
	
	return metrics
}

// GetMemoryMetricsOEMData fetches and parses Nvidia OEM fields from MemoryMetrics endpoint
func (h *NvidiaOEMHelper) GetMemoryMetricsOEMData(metricsEndpoint string) (*MemoryMetricsOEMData, error) {
	data, err := h.fetchJSON(metricsEndpoint)
	if err != nil {
		return nil, err
	}
	
	metrics := parseMemoryMetricsOEMData(data)
	
	h.logger.Debug("extracted MemoryMetrics OEM data",
		slog.String("endpoint", metricsEndpoint),
		slog.Int64("correctable_count", metrics.CorrectableRowRemappingCount),
		slog.Int64("uncorrectable_count", metrics.UncorrectableRowRemappingCount))
	
	return metrics, nil
}

// parseProcessorMetricsOEMData parses Nvidia OEM fields from ProcessorMetrics JSON data
func parseProcessorMetricsOEMData(data map[string]interface{}) *ProcessorMetricsOEMData {
	metrics := &ProcessorMetricsOEMData{}
	
	// Navigate to Oem.Nvidia fields
	if oemData, ok := data["Oem"].(map[string]interface{}); ok {
		if nvidiaData, ok := oemData["Nvidia"].(map[string]interface{}); ok {
			metrics.SMUtilizationPercent = getFloat64(nvidiaData, "SMUtilizationPercent")
			metrics.SMActivityPercent = getFloat64(nvidiaData, "SMActivityPercent")
			metrics.SMOccupancyPercent = getFloat64(nvidiaData, "SMOccupancyPercent")
			metrics.TensorCoreActivityPercent = getFloat64(nvidiaData, "TensorCoreActivityPercent")
			metrics.FP16ActivityPercent = getFloat64(nvidiaData, "FP16ActivityPercent")
			metrics.FP32ActivityPercent = getFloat64(nvidiaData, "FP32ActivityPercent")
			metrics.FP64ActivityPercent = getFloat64(nvidiaData, "FP64ActivityPercent")
			metrics.IntegerActivityUtilizationPercent = getFloat64(nvidiaData, "IntegerActivityUtilizationPercent")
			metrics.NVLinkDataRxBandwidthGbps = getFloat64(nvidiaData, "NVLinkDataRxBandwidthGbps")
			metrics.NVLinkDataTxBandwidthGbps = getFloat64(nvidiaData, "NVLinkDataTxBandwidthGbps")
			metrics.PCIeRXBytes = getInt64(nvidiaData, "PCIeRXBytes")
			metrics.PCIeTXBytes = getInt64(nvidiaData, "PCIeTXBytes")
			
			if val, ok := nvidiaData["SRAMECCErrorThresholdExceeded"].(bool); ok {
				metrics.SRAMECCErrorThresholdExceeded = val
			}
			
			if reasons, ok := nvidiaData["ThrottleReasons"].([]interface{}); ok {
				for _, r := range reasons {
					if str, ok := r.(string); ok {
						metrics.ThrottleReasons = append(metrics.ThrottleReasons, str)
					}
				}
			}
		}
	}
	
	return metrics
}

// GetProcessorMetricsOEMData fetches and parses Nvidia OEM fields from ProcessorMetrics endpoint
func (h *NvidiaOEMHelper) GetProcessorMetricsOEMData(metricsEndpoint string) (*ProcessorMetricsOEMData, error) {
	data, err := h.fetchJSON(metricsEndpoint)
	if err != nil {
		return nil, err
	}
	
	metrics := parseProcessorMetricsOEMData(data)
	
	h.logger.Debug("extracted ProcessorMetrics OEM data",
		slog.String("endpoint", metricsEndpoint),
		slog.Float64("sm_utilization", metrics.SMUtilizationPercent),
		slog.Float64("tensor_core_activity", metrics.TensorCoreActivityPercent))
	
	return metrics, nil
}

// parsePortMetricsOEMData parses Nvidia OEM fields from PortMetrics JSON data
func parsePortMetricsOEMData(data map[string]interface{}) *PortMetricsOEMData {
	metrics := &PortMetricsOEMData{}
	
	// Navigate to Oem.Nvidia fields
	if oemData, ok := data["Oem"].(map[string]interface{}); ok {
		if nvidiaData, ok := oemData["Nvidia"].(map[string]interface{}); ok {
			// Extract NVLinkErrors
			if nvlinkErrors, ok := nvidiaData["NVLinkErrors"].(map[string]interface{}); ok {
				if val, ok := nvlinkErrors["RuntimeError"].(bool); ok {
					metrics.NVLinkErrorsRuntimeError = val
				}
				if val, ok := nvlinkErrors["TrainingError"].(bool); ok {
					metrics.NVLinkErrorsTrainingError = val
				}
			}
			
			metrics.LinkErrorRecoveryCount = getInt64(nvidiaData, "LinkErrorRecoveryCount")
			metrics.LinkDownedCount = getInt64(nvidiaData, "LinkDownedCount")
			metrics.SymbolErrors = getInt64(nvidiaData, "SymbolErrors")
			metrics.MalformedPackets = getInt64(nvidiaData, "MalformedPackets")
			metrics.BitErrorRate = getFloat64(nvidiaData, "BitErrorRate")
			metrics.EffectiveBER = getFloat64(nvidiaData, "EffectiveBER")
		}
	}
	
	return metrics
}

// GetPortMetricsOEMData fetches and parses Nvidia OEM fields from PortMetrics endpoint
func (h *NvidiaOEMHelper) GetPortMetricsOEMData(metricsEndpoint string) (*PortMetricsOEMData, error) {
	data, err := h.fetchJSON(metricsEndpoint)
	if err != nil {
		return nil, err
	}
	
	metrics := parsePortMetricsOEMData(data)
	
	h.logger.Debug("extracted PortMetrics OEM data",
		slog.String("endpoint", metricsEndpoint),
		slog.Bool("runtime_error", metrics.NVLinkErrorsRuntimeError),
		slog.Bool("training_error", metrics.NVLinkErrorsTrainingError),
		slog.Int64("link_recovery_count", metrics.LinkErrorRecoveryCount))
	
	return metrics, nil
}

// Helper functions to safely extract values from maps
func getFloat64(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok {
		return val
	}
	// Handle JSON numbers that might be integers
	if val, ok := data[key].(int); ok {
		return float64(val)
	}
	if val, ok := data[key].(int64); ok {
		return float64(val)
	}
	return 0
}

func getInt64(data map[string]interface{}, key string) int64 {
	if val, ok := data[key].(float64); ok {
		return int64(val)
	}
	if val, ok := data[key].(int64); ok {
		return val
	}
	if val, ok := data[key].(int); ok {
		return int64(val)
	}
	return 0
}
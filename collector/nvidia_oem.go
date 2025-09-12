package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/stmcginnis/gofish/common"
)

// NvidiaOEMClient provides methods to extract Nvidia OEM fields from Redfish responses
type NvidiaOEMClient struct {
	client common.Client
	logger *slog.Logger
}

// NewNvidiaOEMClient creates a new client for extracting Nvidia OEM fields
func NewNvidiaOEMClient(client common.Client, logger *slog.Logger) *NvidiaOEMClient {
	return &NvidiaOEMClient{
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

// memoryResponse represents the JSON structure from Memory endpoint
type memoryResponse struct {
	Oem struct {
		Nvidia struct {
			RowRemappingFailed  bool `json:"RowRemappingFailed"`
			RowRemappingPending bool `json:"RowRemappingPending"`
		} `json:"Nvidia"`
	} `json:"Oem"`
}

// memoryMetricsResponse represents the JSON structure from MemoryMetrics endpoint
type memoryMetricsResponse struct {
	Oem struct {
		Nvidia struct {
			RowRemapping struct {
				CorrectableRowRemappingCount   int64 `json:"CorrectableRowRemappingCount"`
				HighAvailabilityBankCount      int64 `json:"HighAvailabilityBankCount"`
				LowAvailabilityBankCount       int64 `json:"LowAvailabilityBankCount"`
				MaxAvailabilityBankCount       int64 `json:"MaxAvailabilityBankCount"`
				NoAvailabilityBankCount        int64 `json:"NoAvailabilityBankCount"`
				PartialAvailabilityBankCount   int64 `json:"PartialAvailabilityBankCount"`
				UncorrectableRowRemappingCount int64 `json:"UncorrectableRowRemappingCount"`
			} `json:"RowRemapping"`
		} `json:"Nvidia"`
	} `json:"Oem"`
}

// processorMetricsResponse represents the JSON structure from ProcessorMetrics endpoint
type processorMetricsResponse struct {
	Oem struct {
		Nvidia struct {
			SMUtilizationPercent              float64  `json:"SMUtilizationPercent"`
			SMActivityPercent                 float64  `json:"SMActivityPercent"`
			SMOccupancyPercent                float64  `json:"SMOccupancyPercent"`
			TensorCoreActivityPercent         float64  `json:"TensorCoreActivityPercent"`
			FP16ActivityPercent               float64  `json:"FP16ActivityPercent"`
			FP32ActivityPercent               float64  `json:"FP32ActivityPercent"`
			FP64ActivityPercent               float64  `json:"FP64ActivityPercent"`
			IntegerActivityUtilizationPercent float64  `json:"IntegerActivityUtilizationPercent"`
			SRAMECCErrorThresholdExceeded     bool     `json:"SRAMECCErrorThresholdExceeded"`
			NVLinkDataRxBandwidthGbps         float64  `json:"NVLinkDataRxBandwidthGbps"`
			NVLinkDataTxBandwidthGbps         float64  `json:"NVLinkDataTxBandwidthGbps"`
			PCIeRXBytes                       int64    `json:"PCIeRXBytes"`
			PCIeTXBytes                       int64    `json:"PCIeTXBytes"`
			ThrottleReasons                   []string `json:"ThrottleReasons"`
		} `json:"Nvidia"`
	} `json:"Oem"`
}

// portMetricsResponse represents the JSON structure from PortMetrics endpoint
type portMetricsResponse struct {
	Oem struct {
		Nvidia struct {
			NVLinkErrors struct {
				RuntimeError  bool `json:"RuntimeError"`
				TrainingError bool `json:"TrainingError"`
			} `json:"NVLinkErrors"`
			LinkErrorRecoveryCount int64   `json:"LinkErrorRecoveryCount"`
			LinkDownedCount        int64   `json:"LinkDownedCount"`
			SymbolErrors           int64   `json:"SymbolErrors"`
			MalformedPackets       int64   `json:"MalformedPackets"`
			BitErrorRate           float64 `json:"BitErrorRate"`
			EffectiveBER           float64 `json:"EffectiveBER"`
		} `json:"Nvidia"`
	} `json:"Oem"`
}

// GetMemoryOEMMetrics fetches and parses Nvidia OEM fields from Memory endpoint
func (c *NvidiaOEMClient) GetMemoryOEMMetrics(odataID string) (*MemoryOEMMetrics, error) {
	resp, err := c.client.Get(odataID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", odataID, err)
	}
	defer resp.Body.Close() // nolint:errcheck // Close() errors on read are not actionable

	var response memoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode JSON from %s: %w", odataID, err)
	}

	metrics := &MemoryOEMMetrics{
		RowRemappingFailed:  response.Oem.Nvidia.RowRemappingFailed,
		RowRemappingPending: response.Oem.Nvidia.RowRemappingPending,
	}

	c.logger.Debug("extracted Memory OEM metrics",
		slog.String("odataID", odataID),
		slog.Bool("row_remapping_failed", metrics.RowRemappingFailed),
		slog.Bool("row_remapping_pending", metrics.RowRemappingPending))

	return metrics, nil
}

// GetMemoryMetricsOEMData fetches and parses Nvidia OEM fields from MemoryMetrics endpoint
func (c *NvidiaOEMClient) GetMemoryMetricsOEMData(metricsEndpoint string) (*MemoryMetricsOEMData, error) {
	resp, err := c.client.Get(metricsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", metricsEndpoint, err)
	}
	defer resp.Body.Close() // nolint:errcheck // Close() errors on read are not actionable

	var response memoryMetricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode JSON from %s: %w", metricsEndpoint, err)
	}

	metrics := &MemoryMetricsOEMData{
		CorrectableRowRemappingCount:   response.Oem.Nvidia.RowRemapping.CorrectableRowRemappingCount,
		HighAvailabilityBankCount:      response.Oem.Nvidia.RowRemapping.HighAvailabilityBankCount,
		LowAvailabilityBankCount:       response.Oem.Nvidia.RowRemapping.LowAvailabilityBankCount,
		MaxAvailabilityBankCount:       response.Oem.Nvidia.RowRemapping.MaxAvailabilityBankCount,
		NoAvailabilityBankCount:        response.Oem.Nvidia.RowRemapping.NoAvailabilityBankCount,
		PartialAvailabilityBankCount:   response.Oem.Nvidia.RowRemapping.PartialAvailabilityBankCount,
		UncorrectableRowRemappingCount: response.Oem.Nvidia.RowRemapping.UncorrectableRowRemappingCount,
	}

	c.logger.Debug("extracted MemoryMetrics OEM data",
		slog.String("endpoint", metricsEndpoint),
		slog.Int64("correctable_count", metrics.CorrectableRowRemappingCount),
		slog.Int64("uncorrectable_count", metrics.UncorrectableRowRemappingCount))

	return metrics, nil
}

// GetProcessorMetricsOEMData fetches and parses Nvidia OEM fields from ProcessorMetrics endpoint
func (c *NvidiaOEMClient) GetProcessorMetricsOEMData(metricsEndpoint string) (*ProcessorMetricsOEMData, error) {
	resp, err := c.client.Get(metricsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", metricsEndpoint, err)
	}
	defer resp.Body.Close() // nolint:errcheck // Close() errors on read are not actionable

	var response processorMetricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode JSON from %s: %w", metricsEndpoint, err)
	}

	metrics := &ProcessorMetricsOEMData{
		SMUtilizationPercent:              response.Oem.Nvidia.SMUtilizationPercent,
		SMActivityPercent:                 response.Oem.Nvidia.SMActivityPercent,
		SMOccupancyPercent:                response.Oem.Nvidia.SMOccupancyPercent,
		TensorCoreActivityPercent:         response.Oem.Nvidia.TensorCoreActivityPercent,
		FP16ActivityPercent:               response.Oem.Nvidia.FP16ActivityPercent,
		FP32ActivityPercent:               response.Oem.Nvidia.FP32ActivityPercent,
		FP64ActivityPercent:               response.Oem.Nvidia.FP64ActivityPercent,
		IntegerActivityUtilizationPercent: response.Oem.Nvidia.IntegerActivityUtilizationPercent,
		SRAMECCErrorThresholdExceeded:     response.Oem.Nvidia.SRAMECCErrorThresholdExceeded,
		NVLinkDataRxBandwidthGbps:         response.Oem.Nvidia.NVLinkDataRxBandwidthGbps,
		NVLinkDataTxBandwidthGbps:         response.Oem.Nvidia.NVLinkDataTxBandwidthGbps,
		PCIeRXBytes:                       response.Oem.Nvidia.PCIeRXBytes,
		PCIeTXBytes:                       response.Oem.Nvidia.PCIeTXBytes,
		ThrottleReasons:                   response.Oem.Nvidia.ThrottleReasons,
	}

	c.logger.Debug("extracted ProcessorMetrics OEM data",
		slog.String("endpoint", metricsEndpoint),
		slog.Float64("sm_utilization", metrics.SMUtilizationPercent),
		slog.Float64("tensor_core_activity", metrics.TensorCoreActivityPercent))

	return metrics, nil
}

// GetPortMetricsOEMData fetches and parses Nvidia OEM fields from PortMetrics endpoint
func (c *NvidiaOEMClient) GetPortMetricsOEMData(metricsEndpoint string) (*PortMetricsOEMData, error) {
	resp, err := c.client.Get(metricsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", metricsEndpoint, err)
	}
	defer resp.Body.Close() // nolint:errcheck // Close() errors on read are not actionable

	var response portMetricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode JSON from %s: %w", metricsEndpoint, err)
	}

	metrics := &PortMetricsOEMData{
		NVLinkErrorsRuntimeError:  response.Oem.Nvidia.NVLinkErrors.RuntimeError,
		NVLinkErrorsTrainingError: response.Oem.Nvidia.NVLinkErrors.TrainingError,
		LinkErrorRecoveryCount:    response.Oem.Nvidia.LinkErrorRecoveryCount,
		LinkDownedCount:           response.Oem.Nvidia.LinkDownedCount,
		SymbolErrors:              response.Oem.Nvidia.SymbolErrors,
		MalformedPackets:          response.Oem.Nvidia.MalformedPackets,
		BitErrorRate:              response.Oem.Nvidia.BitErrorRate,
		EffectiveBER:              response.Oem.Nvidia.EffectiveBER,
	}

	c.logger.Debug("extracted PortMetrics OEM data",
		slog.String("endpoint", metricsEndpoint),
		slog.Bool("runtime_error", metrics.NVLinkErrorsRuntimeError),
		slog.Bool("training_error", metrics.NVLinkErrorsTrainingError),
		slog.Int64("link_recovery_count", metrics.LinkErrorRecoveryCount))

	return metrics, nil
}
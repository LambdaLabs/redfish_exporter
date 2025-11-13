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
	RowRemappingFailed  bool `json:"RowRemappingFailed"`
	RowRemappingPending bool `json:"RowRemappingPending"`
}

// MemoryMetricsOEMData represents Nvidia OEM fields from MemoryMetrics endpoint
type MemoryMetricsOEMData struct {
	CorrectableRowRemappingCount   int64 `json:"CorrectableRowRemappingCount"`
	HighAvailabilityBankCount      int64 `json:"HighAvailabilityBankCount"`
	LowAvailabilityBankCount       int64 `json:"LowAvailabilityBankCount"`
	MaxAvailabilityBankCount       int64 `json:"MaxAvailabilityBankCount"`
	NoAvailabilityBankCount        int64 `json:"NoAvailabilityBankCount"`
	PartialAvailabilityBankCount   int64 `json:"PartialAvailabilityBankCount"`
	UncorrectableRowRemappingCount int64 `json:"UncorrectableRowRemappingCount"`
}

// ProcessorMetricsOEMData represents Nvidia OEM fields from ProcessorMetrics endpoint
type ProcessorMetricsOEMData struct {
	AccumulatedGPUContextUtilizationDuration string   `json:"AccumulatedGPUContextUtilizationDuration"`
	SMUtilizationPercent                     float64  `json:"SMUtilizationPercent"`
	SMActivityPercent                        float64  `json:"SMActivityPercent"`
	SMOccupancyPercent                       float64  `json:"SMOccupancyPercent"`
	TensorCoreActivityPercent                float64  `json:"TensorCoreActivityPercent"`
	FP16ActivityPercent                      float64  `json:"FP16ActivityPercent"`
	FP32ActivityPercent                      float64  `json:"FP32ActivityPercent"`
	FP64ActivityPercent                      float64  `json:"FP64ActivityPercent"`
	IntegerActivityUtilizationPercent        float64  `json:"IntegerActivityUtilizationPercent"`
	SRAMECCErrorThresholdExceeded            bool     `json:"SRAMECCErrorThresholdExceeded"`
	NVLinkDataRxBandwidthGbps                float64  `json:"NVLinkDataRxBandwidthGbps"`
	NVLinkDataTxBandwidthGbps                float64  `json:"NVLinkDataTxBandwidthGbps"`
	PCIeRXBytes                              int64    `json:"PCIeRXBytes"`
	PCIeTXBytes                              int64    `json:"PCIeTXBytes"`
	ThrottleReasons                          []string `json:"ThrottleReasons"`
}

// NVLinkErrors represents NVLink error states
type NVLinkErrors struct {
	RuntimeError  bool `json:"RuntimeError"`
	TrainingError bool `json:"TrainingError"`
}

// PortMetricsOEMData represents Nvidia OEM fields from PortMetrics endpoint
type PortMetricsOEMData struct {
	NVLinkErrors           NVLinkErrors `json:"NVLinkErrors"`
	LinkErrorRecoveryCount int64        `json:"LinkErrorRecoveryCount"`
	LinkDownedCount        int64        `json:"LinkDownedCount"`
	SymbolErrors           int64        `json:"SymbolErrors"`
	MalformedPackets       int64        `json:"MalformedPackets"`
	BitErrorRate           float64      `json:"BitErrorRate"`
	EffectiveBER           float64      `json:"EffectiveBER"`
}

// memoryResponse represents the JSON structure from Memory endpoint
type memoryResponse struct {
	Oem struct {
		Nvidia MemoryOEMMetrics `json:"Nvidia"`
	} `json:"Oem"`
}

// memoryMetricsResponse represents the JSON structure from MemoryMetrics endpoint
type memoryMetricsResponse struct {
	Oem struct {
		Nvidia struct {
			RowRemapping MemoryMetricsOEMData `json:"RowRemapping"`
		} `json:"Nvidia"`
	} `json:"Oem"`
}

// processorMetricsResponse represents the JSON structure from ProcessorMetrics endpoint
type processorMetricsResponse struct {
	Oem struct {
		Nvidia ProcessorMetricsOEMData `json:"Nvidia"`
	} `json:"Oem"`
}

// portMetricsResponse represents the JSON structure from PortMetrics endpoint
type portMetricsResponse struct {
	Oem struct {
		Nvidia PortMetricsOEMData `json:"Nvidia"`
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

	metrics := &response.Oem.Nvidia

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

	metrics := &response.Oem.Nvidia.RowRemapping

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

	metrics := &response.Oem.Nvidia

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

	metrics := &response.Oem.Nvidia

	c.logger.Debug("extracted PortMetrics OEM data",
		slog.String("endpoint", metricsEndpoint),
		slog.Bool("runtime_error", metrics.NVLinkErrors.RuntimeError),
		slog.Bool("training_error", metrics.NVLinkErrors.TrainingError),
		slog.Int64("link_recovery_count", metrics.LinkErrorRecoveryCount))

	return metrics, nil
}

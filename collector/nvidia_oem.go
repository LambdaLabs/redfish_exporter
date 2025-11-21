package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
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

// MemoryMetricsResponse represents the JSON structure from MemoryMetrics endpoint
type MemoryMetricsResponse struct {
	Nvidia struct {
		RowRemapping MemoryMetricsOEMData `json:"RowRemapping"`
	} `json:"Nvidia"`
}

// ProcessorMetricsOEMResponse represents the JSON structure from ProcessorMetrics OEM data
type ProcessorMetricsOEMResponse struct {
	Nvidia ProcessorMetricsOEMData `json:"Nvidia"`
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

	return metrics, nil
}

type GPUNVLinkCollection struct {
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
					BitErrorRate               float64      `json:"BitErrorRate,omitempty"`
					EffectiveBER               float64      `json:"EffectiveBER,omitempty"`
					EffectiveError             int          `json:"EffectiveError,omitempty"`
					IntentionalLinkDownCount   int          `json:"IntentionalLinkDownCount,omitempty"`
					LinkDownReasonCode         string       `json:"LinkDownReasonCode,omitempty"`
					LinkDownedCount            int          `json:"LinkDownedCount,omitempty"`
					LinkErrorRecoveryCount     int          `json:"LinkErrorRecoveryCount,omitempty"`
					MalformedPackets           int          `json:"MalformedPackets,omitempty"`
					NVLinkDataRxBandwidthGbps  float64      `json:"NVLinkDataRxBandwidthGbps,omitempty"`
					NVLinkDataTxBandwidthGbps  float64      `json:"NVLinkDataTxBandwidthGbps,omitempty"`
					NVLinkErrors               NVLinkErrors `json:"NVLinkErrors,omitempty"`
					NVLinkRawRxBandwidthGbps   float64      `json:"NVLinkRawRxBandwidthGbps,omitempty"`
					NVLinkRawTxBandwidthGbps   float64      `json:"NVLinkRawTxBandwidthGbps,omitempty"`
					RXNoProtocolBytes          int64        `json:"RXNoProtocolBytes,omitempty"`
					SymbolErrors               int          `json:"SymbolErrors,omitempty"`
					TXNoProtocolBytes          int64        `json:"TXNoProtocolBytes,omitempty"`
					TXWait                     int          `json:"TXWait,omitempty"`
					TotalRawBER                float64      `json:"TotalRawBER,omitempty"`
					TotalRawError              int          `json:"TotalRawError,omitempty"`
					UnintentionalLinkDownCount int          `json:"UnintentionalLinkDownCount,omitempty"`
					VL15Dropped                int          `json:"VL15Dropped,omitempty"`
					VL15TXBytes                int          `json:"VL15TXBytes,omitempty"`
					VL15TXPackets              int          `json:"VL15TXPackets,omitempty"`
				} `json:"Nvidia,omitempty"`
			} `json:"Oem"`
		} `json:"Metrics"`
		PortType     string               `json:"PortType"`
		PortProtocol redfish.PortProtocol `json:"PortProtocol"`
		Status       common.Status        `json:"Status"`
	} `json:"Members"`
}

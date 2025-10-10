package collector

import (
	"sync"
)

// TelemetryCapability represents a capability that TelemetryService provides
type TelemetryCapability string

const (
	// CapabilityMemoryMetrics indicates TelemetryService provides HGX_MemoryMetrics_0
	CapabilityMemoryMetrics TelemetryCapability = "memory_metrics"
	// CapabilityProcessorMetrics indicates TelemetryService provides HGX_ProcessorMetrics_0
	CapabilityProcessorMetrics TelemetryCapability = "processor_metrics"
	// CapabilityHealthMetrics indicates TelemetryService provides HGX_HealthMetrics_0
	CapabilityHealthMetrics TelemetryCapability = "health_metrics"
	// CapabilityProcessorGPMMetrics indicates TelemetryService provides HGX_ProcessorGPMMetrics_0
	CapabilityProcessorGPMMetrics TelemetryCapability = "processor_gpm_metrics"
)

// TelemetryCapabilities tracks which capabilities TelemetryService provides
// This allows gpu_collector to skip redundant API calls if telemetry already collected the data
type TelemetryCapabilities struct {
	mu           sync.RWMutex
	capabilities map[TelemetryCapability]bool
}

// NewTelemetryCapabilities creates a new capabilities registry
func NewTelemetryCapabilities() *TelemetryCapabilities {
	return &TelemetryCapabilities{
		capabilities: make(map[TelemetryCapability]bool),
	}
}

// Register marks a capability as available
func (tc *TelemetryCapabilities) Register(capability TelemetryCapability) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.capabilities[capability] = true
}

// Has checks if a capability is available
func (tc *TelemetryCapabilities) Has(capability TelemetryCapability) bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.capabilities[capability]
}

// Reset clears all capabilities (called at the start of each scrape)
func (tc *TelemetryCapabilities) Reset() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.capabilities = make(map[TelemetryCapability]bool)
}

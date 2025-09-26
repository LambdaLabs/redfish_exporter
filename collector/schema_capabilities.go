package collector

import (
	"strconv"
	"strings"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

// RedfishVersion represents a parsed Redfish version
type RedfishVersion struct {
	Major int
	Minor int
	Patch int
}

// ParseRedfishVersion parses a version string like "1.14.0" into a struct
func ParseRedfishVersion(version string) RedfishVersion {
	rv := RedfishVersion{}
	parts := strings.Split(version, ".")
	if len(parts) >= 1 {
		rv.Major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		rv.Minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		rv.Patch, _ = strconv.Atoi(parts[2])
	}
	return rv
}

// IsAtLeast checks if this version is at least the specified version
func (v RedfishVersion) IsAtLeast(major, minor, patch int) bool {
	if v.Major > major {
		return true
	}
	if v.Major < major {
		return false
	}
	// Major versions are equal
	if v.Minor > minor {
		return true
	}
	if v.Minor < minor {
		return false
	}
	// Major and minor are equal
	return v.Patch >= patch
}

// SchemaCapabilities provides methods to check what metrics are supported
// based on the Redfish service version.
type SchemaCapabilities struct {
	version RedfishVersion
}

// NewSchemaCapabilities creates a new capability checker using the service version
func NewSchemaCapabilities(service *gofish.Service) *SchemaCapabilities {
	version := ParseRedfishVersion(service.RedfishVersion)
	return &SchemaCapabilities{
		version: version,
	}
}

// NewSchemaCapabilitiesWithVersion creates a capability checker with a specific version
// This is useful for testing or when you already have the version string
func NewSchemaCapabilitiesWithVersion(versionStr string) *SchemaCapabilities {
	version := ParseRedfishVersion(versionStr)
	return &SchemaCapabilities{
		version: version,
	}
}

// HasPCIeErrors checks if the Redfish version supports PCIe error metrics
// PCIeErrors was added in Redfish 1.14.0 (ProcessorMetrics v1.3.0)
func (s *SchemaCapabilities) HasPCIeErrors(pm *redfish.ProcessorMetrics) bool {
	if pm == nil {
		return false
	}

	// PCIeErrors was added in Redfish 1.14.0
	// This corresponds to ProcessorMetrics schema v1.3.0
	return s.version.IsAtLeast(1, 14, 0)
}

// HasCacheMetrics checks if the Redfish version supports cache metrics
// CacheMetrics was added in Redfish 1.11.0 (ProcessorMetrics v1.0.0)
func (s *SchemaCapabilities) HasCacheMetrics(pm *redfish.ProcessorMetrics) bool {
	if pm == nil {
		return false
	}

	// ProcessorMetrics (including CacheMetrics) was introduced in Redfish 1.11.0
	return s.version.IsAtLeast(1, 11, 0)
}

// HasCoreMetrics checks if the Redfish version supports per-core metrics
// CoreMetrics was added in Redfish 1.15.1 (ProcessorMetrics v1.4.0)
func (s *SchemaCapabilities) HasCoreMetrics(pm *redfish.ProcessorMetrics) bool {
	if pm == nil {
		return false
	}

	// CoreMetrics was added in Redfish 1.15.1
	return s.version.IsAtLeast(1, 15, 1)
}

// HasAcceleratorMetrics checks if the Redfish version supports accelerator/GPU metrics
// AcceleratorMetrics was added in Redfish 1.15.1 (ProcessorMetrics v1.4.0)
func (s *SchemaCapabilities) HasAcceleratorMetrics(pm *redfish.ProcessorMetrics) bool {
	if pm == nil {
		return false
	}

	// AcceleratorMetrics was added in Redfish 1.15.1
	return s.version.IsAtLeast(1, 15, 1)
}

// HasMemoryMetrics checks if the Redfish version supports memory metrics
// Memory.Metrics link was added in Memory v1.6.0 (around Redfish 1.8.0)
func (s *SchemaCapabilities) HasMemoryMetrics(mem *redfish.Memory) bool {
	if mem == nil {
		return false
	}

	// Memory metrics have been available since early versions
	// We'll check if the actual link works rather than version
	metrics, err := mem.Metrics()
	return err == nil && metrics != nil
}

// HasThermalSubsystem checks if the Redfish version supports ThermalSubsystem
// ThermalSubsystem was added in Redfish 1.15.1
func (s *SchemaCapabilities) HasThermalSubsystem(chassis *redfish.Chassis) bool {
	if chassis == nil {
		return false
	}

	// ThermalSubsystem was introduced in Redfish 1.15.1
	// But we should still try to access it in case it's available
	if s.version.IsAtLeast(1, 15, 1) {
		thermalSubsystem, err := chassis.ThermalSubsystem()
		return err == nil && thermalSubsystem != nil
	}
	return false
}

// HasPowerSubsystem checks if the Redfish version supports PowerSubsystem
// PowerSubsystem was added in Redfish 1.15.1
func (s *SchemaCapabilities) HasPowerSubsystem(chassis *redfish.Chassis) bool {
	if chassis == nil {
		return false
	}

	// PowerSubsystem was introduced in Redfish 1.15.1
	if s.version.IsAtLeast(1, 15, 1) {
		powerSubsystem, err := chassis.PowerSubsystem()
		return err == nil && powerSubsystem != nil
	}
	return false
}

// HasLeakDetection checks if the Redfish version supports leak detection
// LeakDetection was added in Redfish 1.15.1
func (s *SchemaCapabilities) HasLeakDetection(thermalSubsystem *redfish.ThermalSubsystem) bool {
	if thermalSubsystem == nil {
		return false
	}

	// LeakDetection was added in Redfish 1.15.1
	if !s.version.IsAtLeast(1, 15, 1) {
		return false
	}

	// Check if leak detection is actually configured
	leakDetection, err := thermalSubsystem.LeakDetection()
	if err != nil || leakDetection == nil {
		return false
	}

	// Check if we actually have leak detectors
	for _, ld := range leakDetection {
		if ld != nil {
			detectors, err := ld.LeakDetectors()
			if err == nil && len(detectors) > 0 {
				return true
			}
		}
	}

	return false
}

// HasNetworkPortStatistics checks if the Redfish version supports network port statistics
// Enhanced port statistics were added in Redfish 1.9.0
func (s *SchemaCapabilities) HasNetworkPortStatistics(port *redfish.NetworkPort) bool {
	if port == nil {
		return false
	}

	// Network port statistics were enhanced in Redfish 1.9.0
	return s.version.IsAtLeast(1, 9, 0)
}

// HasControls checks if the Redfish version supports Controls endpoint
// Controls was added in Redfish 1.17.0
func (s *SchemaCapabilities) HasControls(chassis *redfish.Chassis) bool {
	if chassis == nil {
		return false
	}

	// Controls endpoint was introduced in Redfish 1.17.0
	return s.version.IsAtLeast(1, 17, 0)
}

// HasNetworkAdapters checks if the Redfish version supports NetworkAdapters
// NetworkAdapters was available from early versions (1.8.0+)
func (s *SchemaCapabilities) HasNetworkAdapters(chassis *redfish.Chassis) bool {
	if chassis == nil {
		return false
	}

	// NetworkAdapters have been available since early versions
	return true
}

// HasPhysicalSecurity checks if the Redfish version supports physical security sensors
// PhysicalSecurity was available from early versions
func (s *SchemaCapabilities) HasPhysicalSecurity(chassis *redfish.Chassis) bool {
	if chassis == nil {
		return false
	}

	// PhysicalSecurity has been available since early versions
	return true
}

// HasStorage checks if the Redfish version supports Storage endpoint
// Storage was available from early versions (1.0.0+)
func (s *SchemaCapabilities) HasStorage(system *redfish.ComputerSystem) bool {
	if system == nil {
		return false
	}
	// Storage has been available since early versions
	return true
}

// HasPCIeDevices checks if the Redfish version supports PCIeDevices
// PCIeDevices was added in Redfish 1.1.0
func (s *SchemaCapabilities) HasPCIeDevices(system *redfish.ComputerSystem) bool {
	if system == nil {
		return false
	}
	// PCIeDevices was added in Redfish 1.1.0
	return s.version.IsAtLeast(1, 1, 0)
}

// HasPCIeFunctions checks if the Redfish version supports PCIeFunctions
// PCIeFunctions was added in Redfish 1.2.0
func (s *SchemaCapabilities) HasPCIeFunctions(system *redfish.ComputerSystem) bool {
	if system == nil {
		return false
	}
	// PCIeFunctions was added in Redfish 1.2.0
	return s.version.IsAtLeast(1, 2, 0)
}

// HasNetworkInterfaces checks if the Redfish version supports NetworkInterfaces
// NetworkInterfaces was added in Redfish 1.5.0
func (s *SchemaCapabilities) HasNetworkInterfaces(system *redfish.ComputerSystem) bool {
	if system == nil {
		return false
	}
	// NetworkInterfaces was added in Redfish 1.5.0
	return s.version.IsAtLeast(1, 5, 0)
}

// HasEthernetInterfaces checks if the Redfish version supports EthernetInterfaces
// EthernetInterfaces was available from early versions (1.0.0+)
func (s *SchemaCapabilities) HasEthernetInterfaces(system *redfish.ComputerSystem) bool {
	if system == nil {
		return false
	}
	// EthernetInterfaces has been available since early versions
	return true
}

// HasLogServices checks if the Redfish version supports LogServices
// LogServices was available from early versions (1.0.0+)
func (s *SchemaCapabilities) HasLogServices(resource interface{}) bool {
	if resource == nil {
		return false
	}
	// LogServices has been available since early versions
	return true
}

// HasMemory checks if the Redfish version supports Memory endpoint
// Memory was available from early versions (1.0.0+)
func (s *SchemaCapabilities) HasMemory(system *redfish.ComputerSystem) bool {
	if system == nil {
		return false
	}
	// Memory has been available since early versions
	return true
}

// HasProcessors checks if the Redfish version supports Processors endpoint
// Processors was available from early versions (1.0.0+)
func (s *SchemaCapabilities) HasProcessors(system *redfish.ComputerSystem) bool {
	if system == nil {
		return false
	}
	// Processors has been available since early versions
	return true
}

// HasThermal checks if the Redfish version supports legacy Thermal endpoint
// Thermal was available from early versions but deprecated in favor of ThermalSubsystem
func (s *SchemaCapabilities) HasThermal(chassis *redfish.Chassis) bool {
	if chassis == nil {
		return false
	}
	// Thermal has been available since early versions
	// Note: Deprecated in favor of ThermalSubsystem in v1.15.1
	return true
}

// HasPower checks if the Redfish version supports legacy Power endpoint
// Power was available from early versions but deprecated in favor of PowerSubsystem
func (s *SchemaCapabilities) HasPower(chassis *redfish.Chassis) bool {
	if chassis == nil {
		return false
	}
	// Power has been available since early versions
	// Note: Deprecated in favor of PowerSubsystem in v1.15.1
	return true
}

// HasVolumes checks if the Storage version supports Volumes
// Volumes was added in Storage v1.1.0 (around Redfish 1.1.0)
func (s *SchemaCapabilities) HasVolumes(storage *redfish.Storage) bool {
	if storage == nil {
		return false
	}
	// Volumes was added in Storage v1.1.0
	return s.version.IsAtLeast(1, 1, 0)
}

// HasDrives checks if the Storage version supports Drives
// Drives was available from early versions of Storage
func (s *SchemaCapabilities) HasDrives(storage *redfish.Storage) bool {
	if storage == nil {
		return false
	}
	// Drives has been available since early Storage versions
	return true
}

// HasStorageControllers checks if the Storage version supports StorageControllers
// StorageControllers was added in Storage v1.4.0 (around Redfish 1.7.0)
func (s *SchemaCapabilities) HasStorageControllers(storage *redfish.Storage) bool {
	if storage == nil {
		return false
	}
	// StorageControllers was added in Storage v1.4.0
	return s.version.IsAtLeast(1, 7, 0)
}

// PreferThermalSubsystem checks if we should prefer ThermalSubsystem over legacy Thermal
// ThermalSubsystem is preferred starting from Redfish 1.15.1
func (s *SchemaCapabilities) PreferThermalSubsystem() bool {
	return s.version.IsAtLeast(1, 15, 1)
}

// PreferPowerSubsystem checks if we should prefer PowerSubsystem over legacy Power
// PowerSubsystem is preferred starting from Redfish 1.15.1
func (s *SchemaCapabilities) PreferPowerSubsystem() bool {
	return s.version.IsAtLeast(1, 15, 1)
}

// HasPortMetrics checks if the Redfish version supports PortMetrics
// PortMetrics was added in Redfish 1.13.0 (Port.v1_3_0)
func (s *SchemaCapabilities) HasPortMetrics(port *redfish.Port) bool {
	if port == nil {
		return false
	}

	// PortMetrics was added in Redfish 1.13.0
	return s.version.IsAtLeast(1, 13, 0)
}

// HasGPUMemoryMetrics checks if the Redfish version supports GPU memory metrics
// MemoryMetrics was added in Redfish 1.8.0 (Memory.v1_6_0)
func (s *SchemaCapabilities) HasGPUMemoryMetrics(memory *redfish.Memory) bool {
	if memory == nil {
		return false
	}

	// MemoryMetrics has been available since Redfish 1.8.0
	// We check the actual link instead of just version
	metrics, err := memory.Metrics()
	return err == nil && metrics != nil
}
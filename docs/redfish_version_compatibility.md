# Redfish Version Compatibility Documentation

## Overview
This document tracks schema changes across Redfish versions 1.8.0 to 1.18.0 that affect the redfish_exporter metrics collection.

## Version Timeline and Key Changes

### Redfish 1.8.0 (2019-12)
**Baseline version - Core features**
- **ComputerSystem**: Basic properties (Health, State, ProcessorSummary, MemorySummary)
- **Processor**: Basic properties (Model, TotalCores, Status)
  - NO ProcessorMetrics link
- **Memory**: Basic properties (CapacityMiB, Status)
- **Chassis**:
  - Thermal endpoint (`/redfish/v1/Chassis/{id}/Thermal`)
  - Power endpoint (`/redfish/v1/Chassis/{id}/Power`)
  - NO ThermalSubsystem
- **Manager**: Basic properties (Status, FirmwareVersion)
- **Storage**: Basic storage and drives
- **NetworkAdapter**: Basic network adapter support

### Redfish 1.9.0 (2020-05)
**Incremental improvements**
- **ComputerSystem**: Added BootProgress properties
- **Processor**: Added FPGA and accelerator types
- **Memory**: Enhanced error reporting
- **NetworkAdapter**: Added port statistics
- **PCIeDevice**: Enhanced PCIe device modeling
- **Storage**: NVMe specific properties

### Redfish 1.11.0 (2020-12)
**ProcessorMetrics introduction**
- **Processor**:
  - Added `Metrics` link (points to ProcessorMetrics)
  - ProcessorMetrics v1.0.0 introduced
    - CacheMetrics (CorrectedCacheErrors, UncorrectableCacheErrors)
    - NO PCIeErrors yet
- **Chassis**: Enhanced sensor collections
- **Memory**: Persistent memory regions

### Redfish 1.14.0 (2021-10)
**PCIe errors and enhanced metrics**
- **ProcessorMetrics**: v1.3.0
  - Added PCIeErrors object:
    - CorrectableErrorCount
    - FatalErrorCount
    - L0ToRecoveryCount
    - NAKReceivedCount
    - NAKSentCount
    - ReplayCount
    - ReplayRolloverCount
- **ComputerSystem**: Graphics controller support
- **Chassis**: Environmental metrics enhanced

### Redfish 1.15.1 (2022-03)
**Thermal subsystem refactoring**
- **Chassis**:
  - NEW: ThermalSubsystem endpoint (`/redfish/v1/Chassis/{id}/ThermalSubsystem`)
    - Fans moved to ThermalSubsystem/Fans
    - ThermalMetrics added
    - LeakDetection endpoint added
  - DEPRECATED: Thermal endpoint (but still supported)
- **PowerSubsystem**: New power subsystem model
  - PowerSupplies collection
  - Batteries collection
- **ProcessorMetrics**: v1.4.0
  - CoreMetrics added
  - AcceleratorMetrics for GPUs

### Redfish 1.17.0 (2022-11)
**Enhanced monitoring capabilities**
- **ProcessorMetrics**: v1.5.0
  - Enhanced PCIeErrors
  - Bandwidth utilization metrics
  - Thermal throttling metrics
- **Memory**: Predictive failure analysis
- **Chassis**:
  - Leak detection enhancements
  - Coolant connectors
- **Manager**: ComponentIntegrity collection

### Redfish 1.18.0 (2023-04)
**Latest features**
- **ProcessorMetrics**: v1.6.0
  - CXL metrics
  - Enhanced cache hierarchy metrics
- **ComputerSystem**: Composition service updates
- **ThermalSubsystem**: Coolant loop modeling
- **PowerSubsystem**: Enhanced efficiency metrics
- **ServiceRoot**: Expanded service capabilities

## Collector Impact Analysis

### System Collector
| Feature | 1.8.0 | 1.9.0 | 1.11.0 | 1.14.0 | 1.15.1 | 1.17.0 | 1.18.0 |
|---------|-------|-------|--------|--------|--------|--------|--------|
| Basic System Info | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| ProcessorSummary | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| MemorySummary | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Storage | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| PCIeDevices | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Processor.Metrics link | No | No | Yes | Yes | Yes | Yes | Yes |
| ProcessorMetrics.CacheMetrics | No | No | Yes | Yes | Yes | Yes | Yes |
| ProcessorMetrics.PCIeErrors | No | No | No | Yes | Yes | Yes | Yes |
| ProcessorMetrics.CoreMetrics | No | No | No | No | Yes | Yes | Yes |

### Chassis Collector
| Feature | 1.8.0 | 1.9.0 | 1.11.0 | 1.14.0 | 1.15.1 | 1.17.0 | 1.18.0 |
|---------|-------|-------|--------|--------|--------|--------|--------|
| Thermal endpoint | Yes | Yes | Yes | Yes | Deprecated | Deprecated | Deprecated |
| Power endpoint | Yes | Yes | Yes | Yes | Deprecated | Deprecated | Deprecated |
| ThermalSubsystem | No | No | No | No | Yes | Yes | Yes |
| ThermalSubsystem.Fans | No | No | No | No | Yes | Yes | Yes |
| ThermalSubsystem.LeakDetection | No | No | No | No | Yes | Yes | Yes |
| PowerSubsystem | No | No | No | No | Yes | Yes | Yes |
| NetworkAdapters | Yes | Yes | Yes | Yes | Yes | Yes | Yes |

Note: Deprecated = Still supported but not preferred

### Manager Collector
| Feature | 1.8.0 | 1.9.0 | 1.11.0 | 1.14.0 | 1.15.1 | 1.17.0 | 1.18.0 |
|---------|-------|-------|--------|--------|--------|--------|--------|
| Manager Status | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| FirmwareVersion | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| LogServices | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| NetworkProtocol | Yes | Yes | Yes | Yes | Yes | Yes | Yes |

### GPU Collector (NVIDIA OEM)
| Feature | 1.8.0 | 1.9.0 | 1.11.0 | 1.14.0 | 1.15.1 | 1.17.0 | 1.18.0 |
|---------|-------|-------|--------|--------|--------|--------|--------|
| OEM GPU extensions | No | No | No | Yes | Yes | Yes | Yes |
| GPU Memory metrics | No | No | No | Yes | Yes | Yes | Yes |
| GPU Thermal metrics | No | No | No | Yes | Yes | Yes | Yes |

## Testing Strategy

### For Each Version
1. **Service Root**: Must return appropriate RedfishVersion
2. **Required Endpoints**: Test presence/absence based on version
3. **Schema Compliance**: Responses match version-specific schemas
4. **Graceful Degradation**: Missing features don't cause failures

### Expected Behaviors

#### Version 1.8.0-1.9.0
- No ProcessorMetrics endpoints (404 expected)
- Use legacy Thermal/Power endpoints only
- Basic metrics only

#### Version 1.11.0-1.14.0
- ProcessorMetrics available
- CacheMetrics in 1.11.0+
- PCIeErrors in 1.14.0+
- Still using legacy Thermal/Power

#### Version 1.15.1+
- ThermalSubsystem preferred over Thermal
- PowerSubsystem preferred over Power
- LeakDetection endpoints available
- Fall back to legacy if new endpoints missing

## Metrics Availability Matrix

| Metric | 1.8.0 | 1.9.0 | 1.11.0 | 1.14.0 | 1.15.1 | 1.17.0 | 1.18.0 |
|--------|-------|-------|--------|--------|--------|--------|--------|
| `redfish_up` | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `redfish_system_health_state` | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `redfish_system_processor_state` | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `redfish_system_processor_cache_lifetime_correctable_ecc_error_count` | No | No | Yes | Yes | Yes | Yes | Yes |
| `redfish_system_processor_cache_lifetime_uncorrectable_ecc_error_count` | No | No | Yes | Yes | Yes | Yes | Yes |
| `redfish_system_processor_pcie_errors_correctable_count` | No | No | No | Yes | Yes | Yes | Yes |
| `redfish_system_processor_pcie_errors_fatal_count` | No | No | No | Yes | Yes | Yes | Yes |
| `redfish_system_processor_pcie_errors_l0_to_recovery_count` | No | No | No | Yes | Yes | Yes | Yes |
| `redfish_chassis_fan_rpm` | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `redfish_chassis_temperature_celsius` | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `redfish_chassis_power_watts` | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `redfish_chassis_leak_detector_state` | No | No | No | No | Yes | Yes | Yes |

## Implementation Notes

1. **Endpoint Discovery**: Always check for newer endpoints first, fall back to legacy
2. **Error Handling**: 404s are expected for newer features on older versions
3. **Null Checks**: Older versions may have null or missing properties
4. **OEM Extensions**: May not be present in all versions or vendors
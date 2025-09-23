# Redfish Version Test Fixtures

## Purpose
These fixtures provide minimal, schema-compliant Redfish responses for testing backwards compatibility across versions 1.8.0 to 1.18.0.

## Directory Structure
```
redfish_v1_8_0/   - Baseline version, no ProcessorMetrics
redfish_v1_9_0/   - Incremental improvements, enhanced network stats
redfish_v1_11_0/  - ProcessorMetrics introduced (cache only)
redfish_v1_14_0/  - PCIe errors added to ProcessorMetrics
redfish_v1_15_1/  - ThermalSubsystem/PowerSubsystem refactoring
redfish_v1_17_0/  - Enhanced metrics
redfish_v1_18_0/  - Latest features
```

## Key Test Points by Version

### v1.8.0 - Baseline
- Basic system, processor, chassis endpoints
- Legacy Thermal/Power endpoints only
- No ProcessorMetrics
- No ThermalSubsystem
- No LeakDetection

### v1.9.0 - Incremental Improvements
- Enhanced network port statistics
- FPGA and accelerator processor types
- NVMe specific properties
- Still no ProcessorMetrics
- Legacy Thermal/Power endpoints only

### v1.11.0 - ProcessorMetrics Introduction
- Processor.Metrics link added
- ProcessorMetrics with CacheMetrics
- No PCIe errors yet
- Still using legacy Thermal/Power

### v1.14.0 - PCIe Errors
- ProcessorMetrics.PCIeErrors added
- Full cache and PCIe metrics
- Still using legacy Thermal/Power

### v1.15.1 - Major Refactoring
- ThermalSubsystem endpoint
- PowerSubsystem endpoint
- LeakDetection endpoint
- Legacy endpoints still work (deprecated)
- Exporter should prefer new endpoints

### v1.17.0 - Enhanced Monitoring
- ProcessorMetrics v1.5.0 with bandwidth utilization
- Thermal throttling metrics
- Enhanced cache hierarchy metrics
- Predictive memory failure analysis

### v1.18.0 - Latest Features
- ProcessorMetrics v1.6.0 with CXL metrics
- Enhanced cache hierarchy with L1I/L1D split
- Coolant loop modeling
- Enhanced power efficiency metrics

## Testing Checklist

For each version, verify:

1. **No Crashes**: Exporter handles missing endpoints gracefully
2. **Metric Collection**: Appropriate metrics collected based on version
3. **Error Handling**: 404s for newer features on older versions
4. **Fallback Logic**: Use legacy endpoints when new ones missing
5. **Logging**: Appropriate INFO/WARN messages for missing features

## Expected 404 Responses

| Version | Expected 404s |
|---------|--------------|
| 1.8.0   | ProcessorMetrics, ThermalSubsystem, LeakDetection |
| 1.9.0   | ProcessorMetrics, ThermalSubsystem, LeakDetection |
| 1.11.0  | ThermalSubsystem, LeakDetection |
| 1.14.0  | ThermalSubsystem, LeakDetection |
| 1.15.1  | None (all endpoints available) |
| 1.17.0  | None (all endpoints available) |
| 1.18.0  | None (all endpoints available) |

## Usage in Tests

```go
// Example test setup for v1.8.0
func TestRedfishV1_8_0(t *testing.T) {
    server := newTestRedfishServer(t)
    server.addRouteFromFixture("/redfish/v1/",
        "schemas/redfish_v1_8_0/service_root.json")
    // ... add other endpoints

    // Verify no panic on missing ProcessorMetrics
    // Verify basic metrics collected
    // Verify appropriate error logs
}
```
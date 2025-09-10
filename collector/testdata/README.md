# Test Data Directory

This directory contains JSON fixtures for integration and unit tests. All test responses are stored as JSON files for consistency and transparency.

## Structure

```
testdata/
├── README.md                          # This file
├── service_root.json                  # Redfish service root response
├── systems_collection.json            # Systems collection response
├── system1.json                       # Basic system response
├── processors_collection.json         # Processor collection template
├── processor_with_metrics.json        # Processor v1.20.0 with Metrics link
├── processor_metrics_full.json        # ProcessorMetrics v1.6.1 with all fields
├── processor_zero_errors.json         # Processor with metrics link (zero errors)
├── processor_metrics_zero.json        # ProcessorMetrics with all zero counts
└── schemas/                           # Version-specific test data
    ├── v1_0_0_processor.json          # Schema v1.0.0 - no Metrics support
    ├── v1_4_0_processor.json          # Schema v1.4.0 - Metrics link added
    └── v1_4_0_processor_metrics.json  # ProcessorMetrics v1.0.0 - cache only, no PCIe
```

## Usage in Tests

### Loading Fixtures

```go
// In test setup, use fixture loading methods
setupMock: func(m *testRedfishServer) {
    // Set up basic system structure
    m.setupSystemWithProcessor("System1", "GPU_2")
    
    // Load fixtures for specific routes
    m.addRouteFromFixture("/path/to/resource", "fixture_file.json")
}

// Or load fixture data directly
data := loadTestData(t, "processor_with_metrics.json")
```

### Adding New Test Data

1. Capture real Redfish responses or create minimal valid JSON
2. Save to appropriate file in this directory
3. Add a helper method to `TestDataCatalog` if frequently used
4. Document the schema version and important fields

## File Naming Convention

- `{resource}_with_{feature}.json` - Resources with specific features
- `{resource}_without_{feature}.json` - Resources missing features
- `schemas/v{version}_{resource}.json` - Version-specific schemas

## Schema Versions

| Version | ProcessorMetrics | PCIeErrors | CacheMetrics |
|---------|-----------------|------------|--------------|
| v1.0.0  | ❌              | ❌         | ❌           |
| v1.4.0  | ✅              | ❌         | ✅           |
| v1.7.0  | ✅              | ✅         | ✅           |
| v1.20.0 | ✅              | ✅         | ✅           |

## Design Philosophy

**All test responses are JSON fixtures** - This ensures:
- Consistency across all tests
- Transparent test data that's reviewable in PRs
- Easy to capture real hardware responses
- Clear separation between test logic and test data

## Best Practices

1. **Keep files minimal** - Include only fields being tested
2. **Add comments** - Use `"_comment"` fields to document test scenarios
3. **Version accuracy** - Ensure @odata.type matches actual schema versions
4. **Name clearly** - File names should indicate the test scenario
5. **Real data** - Base fixtures on actual hardware responses when possible
6. **Organize by purpose** - Use `schemas/` for version-specific fixtures
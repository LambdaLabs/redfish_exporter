# Test Data Directory

This directory contains test fixtures for integration and unit tests.

## Structure

```
testdata/
├── README.md                          # This file
├── processor_with_metrics.json        # Processor v1.20.0 with Metrics link
├── processor_metrics_full.json        # ProcessorMetrics v1.6.1 with all fields
└── schemas/                           # Version-specific test data
    ├── v1_0_0_processor.json          # Schema v1.0.0 - no Metrics support
    └── v1_4_0_processor_metrics.json  # Schema v1.0.0 - cache only, no PCIe
```

## Usage in Tests

### Loading Test Data

```go
// Use the test data catalog
catalog := NewTestDataCatalog(t)
processor := catalog.ProcessorWithMetrics()
metrics := catalog.ProcessorMetricsFull()

// Or load directly
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

## Best Practices

1. **Keep files minimal** - Include only fields being tested
2. **Add comments** - Use `"_comment"` fields to document test scenarios
3. **Version accuracy** - Ensure @odata.type matches actual schema versions
4. **Reuse fixtures** - Use the catalog for common test data
5. **Real data** - Base fixtures on actual hardware responses when possible
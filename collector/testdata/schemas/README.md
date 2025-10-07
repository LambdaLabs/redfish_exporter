# Test Fixture Organization

This directory contains Redfish schema test fixtures organized by version and usage pattern.

## Directory Structure

```
schemas/
├── common/                        # Shared fixtures across multiple Redfish versions
│   └── processors_collection_with_cpu1.json  # Processor collection (unchanged across versions)
│
├── redfish_v1_8_0/               # Redfish 1.8.0 specific fixtures
├── redfish_v1_9_0/               # Redfish 1.9.0 specific fixtures
├── redfish_v1_11_0/              # Redfish 1.11.0 specific fixtures
│   ├── processor.json            # Full v1.11.0 processor fixture
│   ├── processor_v1_4_0.json     # Minimal schema test (Processor.v1_4_0)
│   └── processor_metrics_v1_0_0.json  # Schema test (ProcessorMetrics.v1_0_0)
├── redfish_v1_14_0/              # Redfish 1.14.0 specific fixtures
├── redfish_v1_15_1/              # Redfish 1.15.1 specific fixtures
├── redfish_v1_17_0/              # Redfish 1.17.0 specific fixtures
└── redfish_v1_18_0/              # Redfish 1.18.0 specific fixtures

testdata/                          # Parent directory
├── schemas/                      # Version-specific test fixtures (this directory)
├── *.json                        # General test fixtures used across multiple tests
│   ├── chassis_collection.json   # Used by multiple test files
│   ├── systems_collection.json   # Used by multiple test files
│   └── ...                       # Other test-specific fixtures
└── README.md                     # Test data documentation
```

## Fixture Types

### 1. Version-Specific Fixtures
Located in `redfish_vX_Y_Z/` directories. These represent how a specific Redfish version responds:
- Complete fixtures with realistic data for that version
- Include version-appropriate features (e.g., ProcessorMetrics only in v1.11.0+)

### 2. Schema-Specific Test Fixtures
Named with schema version (e.g., `processor_v1_4_0.json`):
- Minimal fixtures for testing specific schema versions
- Used to test backwards compatibility of individual schemas
- May appear in the Redfish version directory where they were introduced

### 3. Common Fixtures
Located in `common/` directory:
- Fixtures that don't change across Redfish versions
- Currently includes processor collections

### 4. General Test Fixtures
Located in parent `testdata/` directory:
- Used by multiple test files beyond version compatibility tests
- Include collections, GPU fixtures, leak detection fixtures, etc.

## Usage Guidelines

1. **Adding a new Redfish version**: Create a new `redfish_vX_Y_Z/` directory
2. **Testing schema compatibility**: Use or create minimal schema-specific fixtures
3. **Shared across versions**: Place in `common/` if only used by version tests
4. **Used by other tests**: Keep in parent `testdata/` directory

## See Also
- `VERSION_TEST_FIXTURES.md` - Detailed documentation of version-specific test expectations
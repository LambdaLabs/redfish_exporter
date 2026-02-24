# Test Data Directory

This directory contains JSON fixtures for integration and unit tests.
It is highly recommended to structure a Redfish API representing a System Under Test (SUT)
first via some top-level name (perhaps reflecting the hardware reivison of a SUT or a specific test name), like so:

## Structure

``` text
.
├── gb300_happypath -> GB300 as a SUT, "happy path" as it contains data representing output the exporter is expected to parse correctly
│   ├── hashedresponses -> See below about the `hashedresponses` directory
│   │   ├── (truncated)
│   │   └── c047e385872467fa29fe50fb04a832d0.json
│   └── redfish -> Beginning of a Redfish API (e.g. /redfish)
│       └── v1 -> e.g. `/redfish/v1`
│           ├── index.json -> See below about index.json files
│           └── Systems -> e.g. `/redfish/v1/Systems`
│               ├── HGX_Baseboard_0
│               │   ├── index.json
│               │   ├── Memory
│               │   │   ├── GPU_0_DRAM_0
│               │   │   │   ├── index.json
│               │   │   │   └── MemoryMetrics
│               │   │   │       └── index.json
│               │   │   ├── GPU_1_DRAM_0
│               │   │   │   ├── index.json
│               │   │   │   └── MemoryMetrics
│               │   │   │       └── index.json
│               │   │   ├── GPU_2_DRAM_0
│               │   │   │   ├── index.json
│               │   │   │   └── MemoryMetrics
│               │   │   │       └── index.json
│               │   │   └── GPU_3_DRAM_0
│               │   │       ├── index.json
│               │   │       └── MemoryMetrics
│               │   │           └── index.json
│               │   └── Processors -> e.g. `/redfish/v1/Systems/HGX_Baseboard_0/Processors`
(truncated)
├── golden -> See below about the `golden` directory
│   └── gb300_nvlink_happy.golden
├── gpu_info_unknown_multi -> Another unit test
├── README.md -> This file
(truncated)
```

Via test helpers, a given `testdata/` prefix following this pattern may be exposed as an HTTP server for easily accommodating and iterating as a 'fake' Redfish-capable server.

### `hashedresponses` directories

In some cases, the exporter MAY need to gather data directly from a Redfish system, using [query parameters](https://redfish.dmtf.org/schemas/DSP0266_1.15.0.html#query-parameters).

These URLs are very shell-unfriendly, so to prevent accidental damage to a local developer system our test helper will first check if a requested URL query path+params matches an MD5 sum of any file found under `hashedresponses/`. If found - that hashed file is read and returned, and if not, the test helper falls back on trying to load and respond with the file as requested.

Generating an MD5 for a given Redfish API follows this format:

``` shell
> echo -n '/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_3/Ports?$expand=.($levels=2)' | md5sum
c047e385872467fa29fe50fb04a832d0  -
```

### `index.json` files

Redfish returns JSON responses for resources at unnamed endpoints.
For instance, `/redfish/v1/Systems` will return a JSON response typically for all Systems that the BMC is aware of.
To return these "unnamed" endpoints' data, the response MUST be represented in `testdata/` directories as a file `index.json`.
The test helpers will read and respond accordingly.

### `golden` directories

Golden files generally allow for representing expected string outputs of a test, for comparison.
In most test cases, it's fine to model the expected output directly in the test.
For some tests (like NVLink telemetry where we want to assert an entire output - over 700 metrics!), it helps to use a golden file to represent the expected output and compare so as to keep the test more focused.

## Usage in Tests

### Spawning a test server and Redfish client

A test which follows the above pattern can spin up and use a mocked Redfish API server and a matching gofish API client as follows, using the `setupTestServerclient` helper:

``` go
// assuming a test table where test.testdataPath = "testdata/gb300_happypath"
_, client := setupTestServerClient(t, test.testdataPath) // Server unused here, but available
logger := NewTestLogger(t, test.testLogLevel) // Another test helper, see below
collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector) // Uses the returned client to create the collector
// TODO: Test something
```

In cases where added middleware is necessary (like to add artifical latency on calls), a `newTestServer` test helper can instead be used and passed a variadic set of middleware. In most cases, it's recommended to just use `setupTestServerClient` :)

### Adding New Test Data

1. (**OPTIONAL**) Copy an existing `testdata/` directory for a new SUT/test case
2. Create the necessary filesystem structure mapping to the necessary Redfish API paths for test
3. Capture real Redfish responses or create minimal valid JSON
4. Save to appropriate files under paths created above in step 1

## Best Practices

- **Use real data** - Capture real Redfish responses when possible. If capturing from a Linux system, and assuming availability of `xsel`, something like:
``` shell
curl -sk "https://<user>:<password>@<redfish-ip>/some/path/here" | jq | xsel -ib
```
will format the response and copy it to the system clipboard for pasting.
- **Naming `testdata/` directories** - Again, model these for either a SUT or a particular test case
- **Remove UUIDs and Serial Numbers** - This is the most painful bit. We'd prefer to anonymize both UUIDs and Serial Numbers of hardware, so take a cursory look and replace with some placeholder for those fields. If the behavior being tested does not need them, consider even removing the fields entirely.

## Other Test Helpers

### `NewTestLogger`

`NewTestLogger` is an `slog.Logger` which writes data using a `t.Log()`.
Doing this lets tests more naturally inject a logger (as most functions require an `slog.Logger`). Many collectors emit detailed data at a debug level, and this makes it very easy to emit those logs during a test. For convenience, the test name is also set as an emitted field on this logger.

Example usage:

``` go
// Assuming a test table where test.testLogLevel = slog.LevelDebug
logger := NewTestLogger(t, test.testLogLevel)
collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
// The collector will now use this logger which writes to t.Log() instead of stdout, etc.
```

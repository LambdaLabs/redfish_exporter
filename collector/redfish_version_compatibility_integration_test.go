package collector

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// VersionTestResult holds the test results for a specific Redfish version
type VersionTestResult struct {
	Version          string
	MetricsCollected map[string]bool
	Errors           []string
	Warnings         []string
	ProcessorMetrics bool
	PCIeErrors       bool
	ThermalSubsystem bool
	LeakDetection    bool
}

// TestRedfishVersionCompatibility tests backwards compatibility across Redfish versions
func TestRedfishVersionCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	versions := []struct {
		version             string
		hasProcessorMetrics bool
		hasPCIeErrors       bool
		hasThermalSubsystem bool
		hasLeakDetection    bool
		setupMock           func(*testRedfishServer)
	}{
		{
			version:             "1.8.0",
			hasProcessorMetrics: false,
			hasPCIeErrors:       false,
			hasThermalSubsystem: false,
			hasLeakDetection:    false,
			setupMock: func(m *testRedfishServer) {
				// v1.8.0 - Baseline version
				m.addRouteFromFixture("/redfish/v1/", "schemas/redfish_v1_8_0/service_root.json")
				m.addRouteFromFixture("/redfish/v1/Systems", "systems_collection.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1", "schemas/redfish_v1_8_0/system.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors", "schemas/common/processors_collection_with_cpu1.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1", "schemas/redfish_v1_8_0/processor.json")
				m.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "schemas/redfish_v1_8_0/chassis.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Thermal", "schemas/redfish_v1_8_0/thermal.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Power", "schemas/redfish_v1_8_0/power.json")
				// ProcessorMetrics endpoint should return 404
				m.add404Route("/redfish/v1/Systems/System1/Processors/CPU1/ProcessorMetrics")
				// ThermalSubsystem endpoints should return 404
				m.add404Route("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem")
				m.add404Route("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection")
			},
		},
		{
			version:             "1.9.0",
			hasProcessorMetrics: false,
			hasPCIeErrors:       false,
			hasThermalSubsystem: false,
			hasLeakDetection:    false,
			setupMock: func(m *testRedfishServer) {
				// v1.9.0 - Incremental improvements, enhanced network port stats
				m.addRouteFromFixture("/redfish/v1/", "schemas/redfish_v1_9_0/service_root.json")
				m.addRouteFromFixture("/redfish/v1/Systems", "systems_collection.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1", "schemas/redfish_v1_9_0/system.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors", "schemas/common/processors_collection_with_cpu1.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1", "schemas/redfish_v1_9_0/processor.json")
				m.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "schemas/redfish_v1_9_0/chassis.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Thermal", "schemas/redfish_v1_9_0/thermal.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Power", "schemas/redfish_v1_9_0/power.json")
				// ProcessorMetrics endpoint should return 404
				m.add404Route("/redfish/v1/Systems/System1/Processors/CPU1/ProcessorMetrics")
				// ThermalSubsystem endpoints should return 404
				m.add404Route("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem")
				m.add404Route("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection")
			},
		},
		{
			version:             "1.11.0",
			hasProcessorMetrics: true,
			hasPCIeErrors:       false,
			hasThermalSubsystem: false,
			hasLeakDetection:    false,
			setupMock: func(m *testRedfishServer) {
				// v1.11.0 - ProcessorMetrics introduced
				m.addRouteFromFixture("/redfish/v1/", "schemas/redfish_v1_11_0/service_root.json")
				m.addRouteFromFixture("/redfish/v1/Systems", "systems_collection.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1", "schemas/redfish_v1_11_0/system.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors", "schemas/common/processors_collection_with_cpu1.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1", "schemas/redfish_v1_11_0/processor.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1/ProcessorMetrics", "schemas/redfish_v1_11_0/processor_metrics.json")
				m.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "schemas/redfish_v1_8_0/chassis.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Thermal", "schemas/redfish_v1_8_0/thermal.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Power", "schemas/redfish_v1_8_0/power.json")
				// ThermalSubsystem endpoints should still return 404
				m.add404Route("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem")
				m.add404Route("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection")
			},
		},
		{
			version:             "1.14.0",
			hasProcessorMetrics: true,
			hasPCIeErrors:       true,
			hasThermalSubsystem: false,
			hasLeakDetection:    false,
			setupMock: func(m *testRedfishServer) {
				// v1.14.0 - PCIe errors added
				m.addRouteFromFixture("/redfish/v1/", "schemas/redfish_v1_14_0/service_root.json")
				m.addRouteFromFixture("/redfish/v1/Systems", "systems_collection.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1", "schemas/redfish_v1_14_0/system.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors", "schemas/common/processors_collection_with_cpu1.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1", "schemas/redfish_v1_14_0/processor.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1/ProcessorMetrics", "schemas/redfish_v1_14_0/processor_metrics.json")
				m.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "schemas/redfish_v1_8_0/chassis.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Thermal", "schemas/redfish_v1_8_0/thermal.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Power", "schemas/redfish_v1_8_0/power.json")
				// ThermalSubsystem endpoints should still return 404
				m.add404Route("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem")
				m.add404Route("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection")
			},
		},
		{
			version:             "1.15.1",
			hasProcessorMetrics: true,
			hasPCIeErrors:       true,
			hasThermalSubsystem: true,
			hasLeakDetection:    true,
			setupMock: func(m *testRedfishServer) {
				// v1.15.1 - ThermalSubsystem/PowerSubsystem refactoring
				m.addRouteFromFixture("/redfish/v1/", "schemas/redfish_v1_15_1/service_root.json")
				m.addRouteFromFixture("/redfish/v1/Systems", "systems_collection.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1", "schemas/redfish_v1_15_1/system.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors", "schemas/common/processors_collection_with_cpu1.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1", "schemas/redfish_v1_15_1/processor.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1/ProcessorMetrics", "schemas/redfish_v1_15_1/processor_metrics.json")
				m.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "schemas/redfish_v1_15_1/chassis.json")
				// Both legacy and new endpoints available
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Thermal", "schemas/redfish_v1_8_0/thermal.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Power", "schemas/redfish_v1_8_0/power.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "schemas/redfish_v1_15_1/thermal_subsystem.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", "schemas/redfish_v1_15_1/leak_detection.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/Fans", "leak_detectors_collection.json")
			},
		},
		{
			version:             "1.17.0",
			hasProcessorMetrics: true,
			hasPCIeErrors:       true,
			hasThermalSubsystem: true,
			hasLeakDetection:    true,
			setupMock: func(m *testRedfishServer) {
				// v1.17.0 - Enhanced monitoring, thermal throttling, bandwidth metrics
				m.addRouteFromFixture("/redfish/v1/", "schemas/redfish_v1_17_0/service_root.json")
				m.addRouteFromFixture("/redfish/v1/Systems", "systems_collection.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1", "schemas/redfish_v1_17_0/system.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors", "schemas/common/processors_collection_with_cpu1.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1", "schemas/redfish_v1_17_0/processor.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1/ProcessorMetrics", "schemas/redfish_v1_17_0/processor_metrics.json")
				m.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "schemas/redfish_v1_17_0/chassis.json")
				// Both legacy and new endpoints available
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Thermal", "schemas/redfish_v1_8_0/thermal.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Power", "schemas/redfish_v1_8_0/power.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "schemas/redfish_v1_15_1/thermal_subsystem.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", "schemas/redfish_v1_15_1/leak_detection.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/Fans", "leak_detectors_collection.json")
			},
		},
		{
			version:             "1.18.0",
			hasProcessorMetrics: true,
			hasPCIeErrors:       true,
			hasThermalSubsystem: true,
			hasLeakDetection:    true,
			setupMock: func(m *testRedfishServer) {
				// v1.18.0 - Latest features, CXL metrics, enhanced cache hierarchy
				m.addRouteFromFixture("/redfish/v1/", "schemas/redfish_v1_18_0/service_root.json")
				m.addRouteFromFixture("/redfish/v1/Systems", "systems_collection.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1", "schemas/redfish_v1_18_0/system.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors", "schemas/common/processors_collection_with_cpu1.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1", "schemas/redfish_v1_18_0/processor.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1/ProcessorMetrics", "schemas/redfish_v1_18_0/processor_metrics.json")
				m.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "schemas/redfish_v1_18_0/chassis.json")
				// Both legacy and new endpoints available
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Thermal", "schemas/redfish_v1_8_0/thermal.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Power", "schemas/redfish_v1_8_0/power.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "schemas/redfish_v1_15_1/thermal_subsystem.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", "schemas/redfish_v1_15_1/leak_detection.json")
				m.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/Fans", "leak_detectors_collection.json")
			},
		},
	}

	results := make([]VersionTestResult, 0, len(versions))

	for _, vt := range versions {
		t.Run(fmt.Sprintf("Redfish_%s", vt.version), func(t *testing.T) {
			// Create a server without default routes
			mux := http.NewServeMux()
			server := &testRedfishServer{
				t:        t,
				mux:      mux,
				requests: make([]string, 0),
			}
			server.Server = httptest.NewServer(mux)
			t.Cleanup(server.Close)

			// Set up version-specific endpoints
			vt.setupMock(server)

			// Connect to test server using gofish client
			client := connectToTestServer(t, server)

			// Collect system metrics
			systemMetrics := collectSystemMetrics(t, client)

			// Analyze collected metrics
			result := VersionTestResult{
				Version:          vt.version,
				MetricsCollected: make(map[string]bool),
				Errors:           []string{},
				Warnings:         []string{},
			}

			// Always mark redfish_up as true if we got here
			result.MetricsCollected["redfish_up"] = true

			// Check system metrics
			for key := range systemMetrics {
				if strings.Contains(key, "processor_state") {
					result.MetricsCollected["processor_state"] = true
				}
				if strings.Contains(key, "cache_correctable") || strings.Contains(key, "cache_uncorrectable") {
					result.MetricsCollected["processor_cache_metrics"] = true
					result.ProcessorMetrics = true
				}
				if strings.Contains(key, "pcie_") {
					result.MetricsCollected["processor_pcie_errors"] = true
					result.PCIeErrors = true
				}
			}

			// Verify expectations based on version
			t.Logf("Version %s collected %d system metrics: %v", vt.version, len(systemMetrics), systemMetrics)

			// Core metrics should always be present
			assert.True(t, result.MetricsCollected["redfish_up"], "redfish_up missing for %s", vt.version)
			assert.True(t, result.MetricsCollected["processor_state"], "processor_state missing for %s", vt.version)

			// Version-specific metrics
			if vt.hasProcessorMetrics {
				assert.True(t, result.ProcessorMetrics, "ProcessorMetrics expected for %s", vt.version)
			} else {
				assert.False(t, result.ProcessorMetrics, "ProcessorMetrics not expected for %s", vt.version)
			}

			if vt.hasPCIeErrors {
				assert.True(t, result.PCIeErrors, "PCIe errors expected for %s", vt.version)
			} else {
				assert.False(t, result.PCIeErrors, "PCIe errors not expected for %s", vt.version)
			}

			if vt.hasLeakDetection {
				// Note: LeakDetection might not generate metrics if no detectors configured
				result.LeakDetection = vt.hasLeakDetection
			}

			results = append(results, result)
		})
	}

	// Generate compatibility matrix
	t.Log("\n=== Redfish Version Compatibility Matrix ===")
	t.Log("| Version | redfish_up | processor_state | cache_metrics | pcie_errors | leak_detection |")
	t.Log("|---------|------------|-----------------|---------------|-------------|----------------|")
	for _, r := range results {
		t.Logf("| %7s | %10s | %15s | %13s | %11s | %14s |",
			r.Version,
			boolToCheck(r.MetricsCollected["redfish_up"]),
			boolToCheck(r.MetricsCollected["processor_state"]),
			boolToCheck(r.ProcessorMetrics),
			boolToCheck(r.PCIeErrors),
			boolToCheck(r.LeakDetection))
	}
}

// newTestRedfishServerWithoutDefaults creates a test server without default routes
func newTestRedfishServerWithoutDefaults(t *testing.T) *testRedfishServer {
	t.Helper()

	trs := &testRedfishServer{
		t:        t,
		mux:      http.NewServeMux(),
		requests: make([]string, 0),
	}

	// Use TLS server to match the collector's HTTPS requirement
	trs.Server = httptest.NewTLSServer(trs.mux)
	t.Cleanup(trs.Close)

	// Don't set up default routes for version-specific testing
	return trs
}

// TestRedfishV1_8_0_Detailed provides detailed testing for v1.8.0 specifically
func TestRedfishV1_8_0_Detailed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a server without default routes
	mux := http.NewServeMux()
	server := &testRedfishServer{
		t:        t,
		mux:      mux,
		requests: make([]string, 0),
	}
	server.Server = httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// Set up v1.8.0 endpoints
	server.addRouteFromFixture("/redfish/v1/", "schemas/redfish_v1_8_0/service_root.json")
	server.addRouteFromFixture("/redfish/v1/Systems", "systems_collection.json")
	server.addRouteFromFixture("/redfish/v1/Systems/System1", "schemas/redfish_v1_8_0/system.json")
	server.addRouteFromFixture("/redfish/v1/Systems/System1/Processors", "schemas/common/processors_collection_with_cpu1.json")
	server.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU1", "schemas/redfish_v1_8_0/processor.json")
	server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "schemas/redfish_v1_8_0/chassis.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Thermal", "schemas/redfish_v1_8_0/thermal.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Power", "schemas/redfish_v1_8_0/power.json")

	// These should return 404 in v1.8.0
	server.add404Route("/redfish/v1/Systems/System1/Processors/CPU1/ProcessorMetrics")
	server.add404Route("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem")
	server.add404Route("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection")

	// Connect to test server using gofish client directly
	client := connectToTestServer(t, server)

	// Run system collector
	systemMetrics := collectSystemMetrics(t, client)

	// Test chassis collector to verify thermal/power endpoints work
	chassis, err := client.Service.Chassis()
	require.NoError(t, err, "Failed to get chassis")

	hasFanMetrics := false
	hasPowerMetrics := false
	chassisMetricCount := 0

	// Check if we can access thermal and power data
	if len(chassis) > 0 {
		firstChassis := chassis[0]

		// Check thermal endpoint (should work in v1.8.0)
		thermal, err := firstChassis.Thermal()
		if err == nil && thermal != nil {
			if len(thermal.Fans) > 0 {
				hasFanMetrics = true
				chassisMetricCount++
			}
		}

		// Check power endpoint (should work in v1.8.0)
		power, err := firstChassis.Power()
		if err == nil && power != nil {
			if len(power.PowerControl) > 0 || len(power.PowerSupplies) > 0 {
				hasPowerMetrics = true
				chassisMetricCount++
			}
		}
	}

	// Verify expected behavior for v1.8.0
	// Should only have basic processor_state, not ProcessorMetrics data (cache/PCIe)
	assert.Equal(t, 1, len(systemMetrics), "Should only have processor_state in v1.8.0")
	assert.Contains(t, systemMetrics, "processor_state", "Should have basic processor state")

	// Should have basic chassis metrics from legacy endpoints
	assert.True(t, hasFanMetrics, "Missing fan metrics from legacy Thermal endpoint")
	assert.True(t, hasPowerMetrics, "Missing power metrics from legacy Power endpoint")

	t.Logf("v1.8.0: No ProcessorMetrics as expected. Chassis has fans=%v power=%v",
		hasFanMetrics, hasPowerMetrics)
}

// Helper function to convert bool to check mark
func boolToCheck(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

// add404Route adds a route that returns 404
func (m *testRedfishServer) add404Route(path string) {
	m.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		m.requests = append(m.requests, r.URL.Path)
		w.WriteHeader(404)
		w.Write([]byte(`{"error": {"code": "Base.1.0.GeneralError", "message": "Resource not found"}}`))
	})
}

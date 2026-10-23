package collector

import (
	"context"
	"log/slog"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
)

// TestCollectPoweredBy covers a chassis that serves no Power resource (Supermicro X14) but
// links its power supplies through Links.PoweredBy: the legacy power supply families and the
// consumed-watts family are emitted from PowerSupply, PowerSupplyMetrics and EnvironmentMetrics.
func TestCollectPoweredBy(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
	server.addRoute("/redfish/v1/Chassis/Chassis_0", map[string]any{
		"@odata.id":   "/redfish/v1/Chassis/Chassis_0",
		"@odata.type": "#Chassis.v1_22_0.Chassis",
		"Id":          "Chassis_0",
		"Name":        "Chassis_0",
		"ChassisType": "RackMount",
		"Status":      map[string]any{"Health": "OK", "HealthRollup": "OK", "State": "Enabled"},
		"PowerSubsystem": map[string]any{
			"@odata.id": "/redfish/v1/Chassis/Chassis_0/PowerSubsystem",
		},
		"EnvironmentMetrics": map[string]any{
			"@odata.id": "/redfish/v1/Chassis/Chassis_0/EnvironmentMetrics",
		},
		"Links": map[string]any{
			"PoweredBy": []map[string]any{
				{"@odata.id": "/redfish/v1/Chassis/Chassis_0/PowerSubsystem/PowerSupplies/1"},
				{"@odata.id": "/redfish/v1/Chassis/Chassis_0/PowerSubsystem/PowerSupplies/2"},
			},
		},
	})
	server.addRoute("/redfish/v1/Chassis/Chassis_0/PowerSubsystem/PowerSupplies/1", map[string]any{
		"@odata.id":          "/redfish/v1/Chassis/Chassis_0/PowerSubsystem/PowerSupplies/1",
		"@odata.type":        "#PowerSupply.v1_5_0.PowerSupply",
		"Id":                 "1",
		"Name":               "Power Supply Bay 1",
		"PowerCapacityWatts": 6600,
		"EfficiencyRatings":  []map[string]any{{"EfficiencyPercent": 94, "LoadPercent": 50}},
		"Status":             map[string]any{"Health": "OK", "State": "Enabled"},
		"Metrics": map[string]any{
			"@odata.id": "/redfish/v1/Chassis/Chassis_0/PowerSubsystem/PowerSupplies/1/Metrics",
		},
	})
	server.addRoute("/redfish/v1/Chassis/Chassis_0/PowerSubsystem/PowerSupplies/1/Metrics", map[string]any{
		"@odata.id":        "/redfish/v1/Chassis/Chassis_0/PowerSubsystem/PowerSupplies/1/Metrics",
		"@odata.type":      "#PowerSupplyMetrics.v1_1_2.PowerSupplyMetrics",
		"Id":               "Metrics",
		"InputPowerWatts":  map[string]any{"Reading": 685},
		"OutputPowerWatts": map[string]any{"Reading": 567},
	})
	// An absent supply: no capacity, no readings, so only its state and health are reported.
	server.addRoute("/redfish/v1/Chassis/Chassis_0/PowerSubsystem/PowerSupplies/2", map[string]any{
		"@odata.id":   "/redfish/v1/Chassis/Chassis_0/PowerSubsystem/PowerSupplies/2",
		"@odata.type": "#PowerSupply.v1_5_0.PowerSupply",
		"Id":          "2",
		"Name":        "Power Supply Bay 2",
		"Status":      map[string]any{"Health": "Critical", "State": "Absent"},
	})
	server.addRoute("/redfish/v1/Chassis/Chassis_0/EnvironmentMetrics", map[string]any{
		"@odata.id":   "/redfish/v1/Chassis/Chassis_0/EnvironmentMetrics",
		"@odata.type": "#EnvironmentMetrics.v1_3_0.EnvironmentMetrics",
		"Id":          "EnvironmentMetrics",
		"Name":        "Chassis Environment Metrics",
		"PowerWatts":  map[string]any{"Reading": 3855},
	})

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.DefaultChassisCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 256)
	collector.CollectWithContext(context.Background(), ch)
	metrics := drainMetrics(t, ch)

	for name, want := range map[string]int{
		"redfish_chassis_power_powersupply_state":                       2,
		"redfish_chassis_power_powersupply_health":                      2,
		"redfish_chassis_power_powersupply_power_capacity_watts":        1,
		"redfish_chassis_power_powersupply_power_efficiency_percentage": 1,
		"redfish_chassis_power_powersupply_power_input_watts":           1,
		"redfish_chassis_power_powersupply_power_output_watts":          1,
		"redfish_chassis_power_powersupply_last_power_output_watts":     1,
		"redfish_chassis_power_average_consumed_watts":                  1,
	} {
		require.Len(t, metrics[name], want, "unexpected sample count for %s", name)
	}

	input := requireMetric(t, metrics, "redfish_chassis_power_powersupply_power_input_watts")
	require.Equal(t, "power_supply", input.labels["resource"])
	require.Equal(t, "Chassis_0", input.labels["chassis_id"])
	require.Equal(t, "Power Supply Bay 1", input.labels["power_supply"])
	require.Equal(t, "1", input.labels["power_supply_id"])
	require.InDelta(t, 685, input.value, 0.01)
	require.InDelta(t, 567, requireMetric(t, metrics, "redfish_chassis_power_powersupply_power_output_watts").value, 0.01)
	require.InDelta(t, 567, requireMetric(t, metrics, "redfish_chassis_power_powersupply_last_power_output_watts").value, 0.01)
	require.InDelta(t, 6600, requireMetric(t, metrics, "redfish_chassis_power_powersupply_power_capacity_watts").value, 0.01)
	require.InDelta(t, 94, requireMetric(t, metrics, "redfish_chassis_power_powersupply_power_efficiency_percentage").value, 0.01)

	states := metrics["redfish_chassis_power_powersupply_state"]
	require.InDelta(t, 1, requireSample(t, states, "power_supply_id", "1").value, 0.01)
	require.InDelta(t, 1, requireSample(t, metrics["redfish_chassis_power_powersupply_health"], "power_supply_id", "1").value, 0.01)

	consumed := requireMetric(t, metrics, "redfish_chassis_power_average_consumed_watts")
	require.Equal(t, "power_wattage", consumed.labels["resource"])
	require.Equal(t, "Chassis Environment Metrics", consumed.labels["power_voltage"])
	require.Equal(t, "EnvironmentMetrics", consumed.labels["power_voltage_id"])
	require.InDelta(t, 3855, consumed.value, 0.01)

	// The PowerSubsystem resource itself is never read: the PoweredBy links are enough.
	for _, request := range server.requests {
		require.NotEqual(t, "GET /redfish/v1/Chassis/Chassis_0/PowerSubsystem", request)
	}
}

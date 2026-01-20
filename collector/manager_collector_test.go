package collector

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestManagerCollectorFirmwareVersionLabel(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRouteFromFixture("/redfish/v1/Managers", "manager_collection.json")
	server.addRouteFromFixture("/redfish/v1/Managers/BMC", "manager.json")

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewManagerCollector(t.Name(), client, logger, config.DefaultManagerCollector)
	require.NoError(t, err)

	// Collect metrics
	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	// Check for manager metrics with firmware_version label
	foundManagerMetric := false
	var firmwareVersion string

	for metric := range ch {
		dto := &dto.Metric{}
		require.NoError(t, metric.Write(dto))

		desc := metric.Desc()
		descString := desc.String()

		if strings.Contains(descString, "manager_state") ||
			strings.Contains(descString, "manager_health_state") ||
			strings.Contains(descString, "manager_power_state") {
			foundManagerMetric = true

			// Check labels
			labelMap := make(map[string]string)
			for _, label := range dto.Label {
				labelMap[label.GetName()] = label.GetValue()
			}

			require.Equal(t, "BMC", labelMap["manager_id"], "Expected manager_id label")
			require.Equal(t, "Manager", labelMap["name"], "Expected name label")
			require.Equal(t, "Lenovo XClarity Controller", labelMap["model"], "Expected model label")
			require.Equal(t, "BMC", labelMap["type"], "Expected type label")
			require.Contains(t, labelMap, "firmware_version", "Expected firmware_version label")
			firmwareVersion = labelMap["firmware_version"]
			break
		}
	}

	require.True(t, foundManagerMetric, "Expected to find manager metric")
	require.Equal(t, "2.10.0", firmwareVersion, "Expected firmware_version to be '2.10.0'")
}

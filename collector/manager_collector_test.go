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
	_, client := setupTestServerClient(t, "testdata/manager_happypath")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewManagerCollector(t.Name(), client, logger, config.DefaultManagerCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

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

			labelMap := make(map[string]string)
			for _, label := range dto.Label {
				labelMap[label.GetName()] = label.GetValue()
			}

			require.Equal(t, "BMC", labelMap["manager_id"])
			require.Equal(t, "Manager", labelMap["name"])
			require.Equal(t, "Lenovo XClarity Controller", labelMap["model"])
			require.Equal(t, "BMC", labelMap["type"])
			require.Contains(t, labelMap, "firmware_version")
			firmwareVersion = labelMap["firmware_version"]
			break
		}
	}

	require.True(t, foundManagerMetric, "Expected to find manager metric")
	require.Equal(t, "2.10.0", firmwareVersion)
}

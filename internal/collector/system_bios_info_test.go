package collector

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// TestSystemCollectorBiosInfo verifies that the system collector emits
// redfish_system_bios_info for the host system (with bios_version + model
// labels) and skips members that report no BiosVersion (e.g. HGX baseboards).
func TestSystemCollectorBiosInfo(t *testing.T) {
	_, client := setupTestServerClient(t, "testdata/gpu_info_unknown_multi")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewSystemCollector(t.Name(), client, logger, config.DefaultSystemCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 1000)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	var biosInfoLabels []map[string]string
	for metric := range ch {
		if !strings.Contains(metric.Desc().String(), "system_bios_info") {
			continue
		}
		d := &dto.Metric{}
		require.NoError(t, metric.Write(d))
		require.Equal(t, float64(1), d.GetGauge().GetValue())
		labelMap := make(map[string]string)
		for _, label := range d.Label {
			labelMap[label.GetName()] = label.GetValue()
		}
		biosInfoLabels = append(biosInfoLabels, labelMap)
	}

	// Exactly one series: the host system. HGX_Baseboard_0 has no BiosVersion
	// and must be filtered out.
	require.Len(t, biosInfoLabels, 1, "expected exactly one host bios_info series")
	require.Equal(t, "2.4b", biosInfoLabels[0]["bios_version"])
	require.Equal(t, "SYS-A21GE-NBRT-01-LL014", biosInfoLabels[0]["model"])
}

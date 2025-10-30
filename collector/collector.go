package collector

import (
	"fmt"
	"log/slog"

	"github.com/LambdaLabs/redfish_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
)

// Collector creates and returns a prometheus.Collector from the Module, based on the Prober.
// Both redfish client and logger are common dependencies for any redfish_exporter collector,
// and must be provided as inputs.
func NewCollectorFromModule(m *config.Module, rfClient *gofish.APIClient, logger *slog.Logger) (prometheus.Collector, error) {
	switch m.Prober {
	case "chassis_collector":
		return NewChassisCollector(rfClient, logger, &m.ChassisCollector)
	case "gpu_collector":
		return NewGPUCollector(rfClient, logger, &m.GPUCollector)
	case "json_collector":
		return NewJSONCollector(rfClient, logger, &m.JSONCollector)
	case "manager_collector":
		return NewManagerCollector(rfClient, logger, &m.ManagerCollector)
	case "system_collector":
		return NewSystemCollector(rfClient, logger, &m.SystemCollector)
	case "telemetry_collector":
		return NewTelemetryCollector(rfClient, logger, &m.TelemetryCollector)
	default:
	}
	return nil, fmt.Errorf("prober type %s is not known to redfish_exporter", m.Prober)
}

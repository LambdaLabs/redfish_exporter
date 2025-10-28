package collector

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/LambdaLabs/redfish_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
)

// ManagerSubmanager is the manager subsystem
var (
	ManagerSubmanager = "manager"
	ManagerLabelNames = []string{"manager_id", "name", "model", "type"}

	ManagerLogServiceLabelNames = []string{"manager_id", "log_service", "log_service_id", "log_service_enabled", "log_service_overwrite_policy"}

	managerMetrics = createManagerMetricMap()
)

// ManagerCollector implements the prometheus.Collector.
type ManagerCollector struct {
	redfishClient         *gofish.APIClient
	config                *config.ManagerCollectorConfig
	metrics               map[string]Metric
	logger                *slog.Logger
	collectorScrapeStatus *prometheus.GaugeVec
}

func createManagerMetricMap() map[string]Metric {
	managerMetrics := make(map[string]Metric)
	addToMetricMap(managerMetrics, ManagerSubmanager, "state", fmt.Sprintf("manager state,%s", CommonStateHelp), ManagerLabelNames)
	addToMetricMap(managerMetrics, ManagerSubmanager, "health_state", fmt.Sprintf("manager health,%s", CommonHealthHelp), ManagerLabelNames)
	addToMetricMap(managerMetrics, ManagerSubmanager, "power_state", "manager power state", ManagerLabelNames)

	addToMetricMap(managerMetrics, ManagerSubmanager, "log_service_state", fmt.Sprintf("manager log service state,%s", CommonStateHelp), ManagerLogServiceLabelNames)
	addToMetricMap(managerMetrics, ManagerSubmanager, "log_service_health_state", fmt.Sprintf("manager log service health state,%s", CommonHealthHelp), ManagerLogServiceLabelNames)

	return managerMetrics
}

// NewManagerCollector returns a collector that collecting memory statistics
func NewManagerCollector(redfishClient *gofish.APIClient, logger *slog.Logger, config *config.ManagerCollectorConfig) *ManagerCollector {
	return &ManagerCollector{
		redfishClient: redfishClient,
		config:        config,
		metrics:       managerMetrics,
		logger:        logger,
		collectorScrapeStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "collector_scrape_status",
				Help:      "collector_scrape_status",
			},
			[]string{"collector"},
		),
	}
}

// Describe implemented prometheus.Collector
func (m *ManagerCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range m.metrics {
		ch <- metric.desc
	}
	m.collectorScrapeStatus.Describe(ch)

}

// Collect implemented prometheus.Collector
func (m *ManagerCollector) Collect(ch chan<- prometheus.Metric) {

	logger := m.logger.With(slog.String("collector", "ManagerCollector"))
	service := m.redfishClient.Service

	// get a list of managers from service
	if managers, err := service.Managers(); err != nil {
		logger.Error("error getting managers from service", slog.String("operation", "service.Managers()"), slog.Any("error", err))
	} else {
		for _, manager := range managers {
			managerLogger := logger.With(slog.String("manager", manager.ID))
			managerLogger.Info("collector scrape started")

			// overall manager metrics
			ManagerID := manager.ID
			managerName := manager.Name
			managerModel := manager.Model
			managerType := fmt.Sprint(manager.ManagerType)
			managerPowerState := manager.PowerState
			managerState := manager.Status.State
			managerHealthState := manager.Status.Health

			ManagerLabelValues := []string{ManagerID, managerName, managerModel, managerType}

			if managerHealthStateValue, ok := parseCommonStatusHealth(managerHealthState); ok {
				ch <- prometheus.MustNewConstMetric(m.metrics["manager_health_state"].desc, prometheus.GaugeValue, managerHealthStateValue, ManagerLabelValues...)
			}
			if managerStateValue, ok := parseCommonStatusState(managerState); ok {
				ch <- prometheus.MustNewConstMetric(m.metrics["manager_state"].desc, prometheus.GaugeValue, managerStateValue, ManagerLabelValues...)
			}
			if managerPowerStateValue, ok := parseCommonPowerState(managerPowerState); ok {
				ch <- prometheus.MustNewConstMetric(m.metrics["manager_power_state"].desc, prometheus.GaugeValue, managerPowerStateValue, ManagerLabelValues...)
			}

			// process log services
			logServices, err := manager.LogServices()
			if err != nil {
				managerLogger.Error("error getting log services from manager", slog.Any("error", err), slog.String("operation", "manager.LogServices()"))
			} else if logServices == nil {
				managerLogger.Info("no log services found", slog.String("operation", "manager.LogServices()"))
			} else {
				wg := &sync.WaitGroup{}
				wg.Add(len(logServices))

				for _, logService := range logServices {
					if err = parseLogService(ch, managerMetrics, ManagerSubmanager, ManagerID, logService, wg); err != nil {
						managerLogger.Error("error getting log entries from log service", slog.String("operation", "manager.LogServices()"))
					}
				}
			}
			managerLogger.Info("collector scrape completed")
		}
		m.collectorScrapeStatus.WithLabelValues("manager").Set(float64(1))
	}
}

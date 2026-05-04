package collector

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
)

// FirmwareInventorySubsystem is the firmware inventory subsystem name.
var (
	FirmwareInventorySubsystem = "firmware_inventory"

	FirmwareInventoryLabelNames     = []string{"component_id"}
	FirmwareInventoryInfoLabelNames = []string{"component_id", "name", "version", "manufacturer"}

	// Id-prefix exclusions: inactive recovery / standby / staged-update slots
	// that would generate false drift alerts if surfaced as fleet metrics.
	firmwareInventoryExcludedIDPrefixes = []string{"Backup_", "Golden_", "Capsule_"}

	firmwareInventoryMetrics = createFirmwareInventoryMetricMap()
)

var _ ContextAwareCollector = (*FirmwareInventoryCollector)(nil)

// FirmwareInventoryCollector implements the prometheus.Collector for firmware inventory metrics.
type FirmwareInventoryCollector struct {
	redfishClient         *gofish.APIClient
	config                config.FirmwareInventoryCollectorConfig
	metrics               map[string]Metric
	logger                *slog.Logger
	collectorScrapeStatus *prometheus.GaugeVec
}

func createFirmwareInventoryMetricMap() map[string]Metric {
	metrics := make(map[string]Metric)
	addToMetricMap(metrics, FirmwareInventorySubsystem, "state", fmt.Sprintf("firmware inventory state,%s", CommonStateHelp), FirmwareInventoryLabelNames)
	addToMetricMap(metrics, FirmwareInventorySubsystem, "health", fmt.Sprintf("firmware inventory health,%s", CommonHealthHelp), FirmwareInventoryLabelNames)
	addToMetricMap(metrics, FirmwareInventorySubsystem, "write_protected", "firmware inventory write protection (1 if write-protected, 0 if not write-protected or field absent from BMC response)", FirmwareInventoryLabelNames)
	addToMetricMap(metrics, FirmwareInventorySubsystem, "info", "firmware inventory metadata; always 1. The version label on this metric is the fleet drift signal.", FirmwareInventoryInfoLabelNames)
	return metrics
}

// NewFirmwareInventoryCollector returns a collector that exposes firmware version and status
// for entries under /redfish/v1/UpdateService/FirmwareInventory.
func NewFirmwareInventoryCollector(moduleName string, redfishClient *gofish.APIClient, logger *slog.Logger, config config.FirmwareInventoryCollectorConfig) (*FirmwareInventoryCollector, error) {
	return &FirmwareInventoryCollector{
		redfishClient: redfishClient,
		config:        config,
		metrics:       firmwareInventoryMetrics,
		logger:        logger,
		collectorScrapeStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "collector_scrape_status",
				Help:      "collector_scrape_status",
			},
			[]string{"collector"},
		),
	}, nil
}

// Describe implements prometheus.Collector.
func (f *FirmwareInventoryCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range f.metrics {
		ch <- metric.desc
	}
	f.collectorScrapeStatus.Describe(ch)
}

// Collect implements prometheus.Collector.
func (f *FirmwareInventoryCollector) Collect(ch chan<- prometheus.Metric) {
	f.collect(context.TODO(), ch)
}

// CollectWithContext implements ContextAwareCollector.
func (f *FirmwareInventoryCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
	f.collect(ctx, ch)
}

func (f *FirmwareInventoryCollector) collect(ctx context.Context, ch chan<- prometheus.Metric) {
	if ctx.Err() != nil {
		f.logger.With("error", ctx.Err(), "collector", "firmware_inventory").Debug("skipping collection")
		return
	}
	logger := f.logger.With(slog.String("collector", "FirmwareInventoryCollector"))
	service := f.redfishClient.WithContext(ctx).Service

	updateService, err := service.UpdateService()
	if err != nil {
		logger.Error("error getting update service", slog.String("operation", "service.UpdateService()"), slog.Any("error", err))
		return
	}
	if updateService == nil {
		logger.Info("no update service found", slog.String("operation", "service.UpdateService()"))
		return
	}

	inventories, err := updateService.FirmwareInventory()
	if err != nil {
		logger.Error("error getting firmware inventory", slog.String("operation", "updateService.FirmwareInventory()"), slog.Any("error", err))
		return
	}

	for _, item := range inventories {
		if ctx.Err() != nil {
			logger.With("error", ctx.Err()).Debug("skipping further collection")
			return
		}
		if item == nil || isExcludedFirmwareInventoryID(item.ID) {
			continue
		}

		labelValues := []string{item.ID}

		if v, ok := parseCommonStatusState(item.Status.State); ok {
			ch <- prometheus.MustNewConstMetric(f.metrics["firmware_inventory_state"].desc, prometheus.GaugeValue, v, labelValues...)
		}
		if v, ok := parseCommonStatusHealth(item.Status.Health); ok {
			ch <- prometheus.MustNewConstMetric(f.metrics["firmware_inventory_health"].desc, prometheus.GaugeValue, v, labelValues...)
		}
		ch <- prometheus.MustNewConstMetric(f.metrics["firmware_inventory_write_protected"].desc, prometheus.GaugeValue, boolToFloat64(item.WriteProtected), labelValues...)

		infoLabelValues := []string{item.ID, item.Name, item.Version, item.Manufacturer}
		ch <- prometheus.MustNewConstMetric(f.metrics["firmware_inventory_info"].desc, prometheus.GaugeValue, 1, infoLabelValues...)
	}

	f.collectorScrapeStatus.WithLabelValues("firmware_inventory").Set(float64(1))
}

func isExcludedFirmwareInventoryID(id string) bool {
	for _, prefix := range firmwareInventoryExcludedIDPrefixes {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}

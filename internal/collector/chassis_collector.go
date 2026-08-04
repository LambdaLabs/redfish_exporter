package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/schemas"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
)

// ChassisSubsystem is the chassis subsystem
var (
	ChassisSubsystem                  = "chassis"
	ChassisLabelNames                 = []string{"resource", "chassis_id"}
	ChassisModel                      = []string{"resource", "chassis_id", "manufacturer", "model", "part_number", "sku"}
	ChassisTemperatureLabelNames      = []string{"resource", "chassis_id", "sensor", "sensor_id"}
	ChassisFanLabelNames              = []string{"resource", "chassis_id", "fan", "fan_id", "fan_unit"}
	ChassisPowerVoltageLabelNames     = []string{"resource", "chassis_id", "power_voltage", "power_voltage_id"}
	ChassisPowerSupplyLabelNames      = []string{"resource", "chassis_id", "power_supply", "power_supply_id"}
	ChassisNetworkAdapterLabelNames   = []string{"resource", "chassis_id", "network_adapter", "network_adapter_id"}
	ChassisNetworkPortLabelNames      = []string{"resource", "chassis_id", "network_adapter", "network_adapter_id", "network_port", "network_port_id", "network_port_type", "network_port_speed", "network_port_connectiont_type", "network_physical_port_number"}
	ChassisPhysicalSecurityLabelNames = []string{"resource", "chassis_id", "intrusion_sensor_number", "intrusion_sensor_rearm"}
	ChassisLeakDetectorLabelNames     = []string{"resource", "chassis_id", "leak_detection_id", "leak_detector_id"}
	ChassisLeakDetectionLabelNames    = []string{"resource", "chassis_id", "leak_detection_id"}
	ChassisSensorLabelNames           = []string{"resource", "chassis_id", "sensor", "sensor_id", "sensor_units", "physical_context"}
	ChassisLeakDetectorInfoLabelNames = []string{"resource", "chassis_id", "leak_detection_id", "leak_detector_id", "leak_detector_type", "physical_context", "physical_sub_context"}

	ChassisLogServiceLabelNames = []string{"chassis_id", "log_service", "log_service_id", "log_service_enabled", "log_service_overwrite_policy"}

	chassisMetrics = createChassisMetricMap()
)

// ChassisCollector implements the prometheus.Collector.
type ChassisCollector struct {
	redfishClient         *gofish.APIClient
	config                config.ChassisCollectorConfig
	chassisInclude        *regexp.Regexp
	chassisExclude        *regexp.Regexp
	metrics               map[string]Metric
	logger                *slog.Logger
	collectorScrapeStatus *prometheus.GaugeVec
}

// skipChassis reports whether a chassis Id is filtered out by the configured
// include/exclude patterns. An unset pattern never filters.
func (c *ChassisCollector) skipChassis(chassisID string) bool {
	if c.chassisInclude != nil && !c.chassisInclude.MatchString(chassisID) {
		return true
	}
	if c.chassisExclude != nil && c.chassisExclude.MatchString(chassisID) {
		return true
	}
	return false
}

func createChassisMetricMap() map[string]Metric {
	chassisMetrics := make(map[string]Metric)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "health", fmt.Sprintf("health of chassis,%s", CommonHealthHelp), ChassisLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "health_rollup", fmt.Sprintf("health rollup of chassis,%s", CommonHealthHelp), ChassisLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "state", fmt.Sprintf("state of chassis,%s", CommonStateHelp), ChassisLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "model_info", "organization responsible for producing the chassis, the name by which the manufacturer generally refers to the chassis, and a part number and sku assigned by the organization that is responsible for producing or manufacturing the chassis", ChassisModel)

	addToMetricMap(chassisMetrics, ChassisSubsystem, "temperature_sensor_state", fmt.Sprintf("status state of temperature on this chassis component,%s", CommonStateHelp), ChassisTemperatureLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "temperature_sensor_health", fmt.Sprintf("status health of temperature on this chassis component,%s", CommonStateHelp), ChassisTemperatureLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "temperature_celsius", "celsius of temperature on this chassis component", ChassisTemperatureLabelNames)

	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_health", fmt.Sprintf("fan health on this chassis component,%s", CommonHealthHelp), ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_state", fmt.Sprintf("fan state on this chassis component,%s", CommonStateHelp), ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_rpm", "fan RPM or percentage on this chassis component", ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_rpm_percentage", "fan RPM, as a percentage of the min-max RPMs possible, on this chassis component", ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_rpm_min", "lowest possible fan RPM or percentage, on this chassis component", ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_rpm_max", "highest possible fan RPM or percentage, on this chassis component", ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_rpm_lower_threshold_critical", "threshold below the normal range fan RPM or percentage, but not fatal, on this chassis component", ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_rpm_lower_threshold_non_critical", "threshold below the normal range fan RPM or percentage, but not critical, on this chassis component", ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_rpm_lower_threshold_fatal", "threshold below the normal range fan RPM or percentage, and is fatal, on this chassis component", ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_rpm_upper_threshold_critical", "threshold above the normal range fan RPM or percentage, but not fatal, on this chassis component", ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_rpm_upper_threshold_non_critical", "threshold above the normal range fan RPM or percentage, but not critical, on this chassis component", ChassisFanLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "fan_rpm_upper_threshold_fatal", "threshold above the normal range fan RPM or percentage, and is fatal, on this chassis component", ChassisFanLabelNames)

	addToMetricMap(chassisMetrics, ChassisSubsystem, "power_voltage_state", fmt.Sprintf("power voltage state of chassis component,%s", CommonStateHelp), ChassisPowerVoltageLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "power_voltage_volts", "power voltage volts number of chassis component", ChassisPowerVoltageLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "power_average_consumed_watts", "power wattage watts number of chassis component", ChassisPowerVoltageLabelNames)

	addToMetricMap(chassisMetrics, ChassisSubsystem, "power_powersupply_state", fmt.Sprintf("powersupply state of chassis component,%s", CommonStateHelp), ChassisPowerSupplyLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "power_powersupply_health", fmt.Sprintf("powersupply health of chassis component,%s", CommonHealthHelp), ChassisPowerSupplyLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "power_powersupply_power_efficiency_percentage", "rated efficiency, as a percentage, of the associated power supply on this chassis", ChassisPowerSupplyLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "power_powersupply_last_power_output_watts", "average power output, measured in Watts, of the associated power supply on this chassis", ChassisPowerSupplyLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "power_powersupply_power_input_watts", "measured input power, in Watts, of powersupply on this chassis", ChassisPowerSupplyLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "power_powersupply_power_output_watts", "measured output power, in Watts, of powersupply on this chassis", ChassisPowerSupplyLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "power_powersupply_power_capacity_watts", "power_capacity_watts of powersupply on this chassis", ChassisPowerSupplyLabelNames)

	addToMetricMap(chassisMetrics, ChassisSubsystem, "network_adapter_state", fmt.Sprintf("chassis network adapter state,%s", CommonStateHelp), ChassisNetworkAdapterLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "network_adapter_health_state", fmt.Sprintf("chassis network adapter health state,%s", CommonHealthHelp), ChassisNetworkAdapterLabelNames)

	addToMetricMap(chassisMetrics, ChassisSubsystem, "network_port_state", fmt.Sprintf("chassis network port state,%s", CommonStateHelp), ChassisNetworkPortLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "network_port_health_state", fmt.Sprintf("chassis network port health state,%s", CommonHealthHelp), ChassisNetworkPortLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "network_port_link_state", fmt.Sprintf("chassis network port link state state,%s", CommonPortLinkHelp), ChassisNetworkPortLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "physical_security_sensor_state", fmt.Sprintf("indicates the known state of the physical security sensor, such as if it is hardware intrusion detected,%s", CommonIntrusionSensorHelp), ChassisPhysicalSecurityLabelNames)

	addToMetricMap(chassisMetrics, ChassisSubsystem, "log_service_state", fmt.Sprintf("chassis log service state,%s", CommonStateHelp), ChassisLogServiceLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "log_service_health_state", fmt.Sprintf("chassis log service health state,%s", CommonHealthHelp), ChassisLogServiceLabelNames)

	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_health", fmt.Sprintf("chassis leak detector health state,%s", CommonHealthHelp), ChassisLeakDetectorLabelNames)
	// leak_detector_state is the actual leak signal. leak_detector_health above describes
	// the health of the detector device itself, which can remain OK while a leak is
	// reported, so it must not be used as a leak indicator on its own.
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_state", fmt.Sprintf("chassis leak detector state; this is the leak signal,%s", CommonDetectorStateHelp), ChassisLeakDetectorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_enabled", "whether this chassis leak detector is enabled, 1(enabled),0(disabled); a disabled detector reports Unavailable state and does not trigger events", ChassisLeakDetectorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_info", "chassis leak detector type and physical location, always 1", ChassisLeakDetectorInfoLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detection_health", fmt.Sprintf("health of the chassis leak detection subsystem as a whole,%s", CommonHealthHelp), ChassisLeakDetectionLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detection_state", fmt.Sprintf("state of the chassis leak detection subsystem as a whole,%s", CommonStateHelp), ChassisLeakDetectionLabelNames)
	// Analog side of the leak detectors, read from the companion Sensor resources. These
	// are resistive moisture ropes on a voltage divider: dry reads high and water pulls
	// the voltage down, so the alarm is a LOWER critical crossing.
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_volts", "chassis leak detector reading in volts; falls toward the lower critical threshold as moisture is detected", ChassisLeakDetectorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_volts_lower_threshold_critical", "voltage at or below which this chassis leak detector reports a critical leak", ChassisLeakDetectorLabelNames)

	// Catch-all metrics for Sensors whose ReadingType has no exact equivalent among the
	// curated families above, so that a reading is never silently dropped. Sensors whose
	// meaning is unambiguous (Temperature, Rotational, Voltage) are folded into the
	// existing chassis families instead, which keeps one metric name per concept
	// fleet-wide regardless of whether a platform implements Thermal/Power or Sensors.
	addToMetricMap(chassisMetrics, ChassisSubsystem, "sensor_watts", "chassis sensor reading in watts", ChassisSensorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "sensor_amperes", "chassis sensor reading in amperes", ChassisSensorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "sensor_joules", "chassis sensor reading in joules", ChassisSensorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "sensor_hertz", "chassis sensor reading in hertz", ChassisSensorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "sensor_percent", "chassis sensor reading as a percentage", ChassisSensorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "sensor_reading", fmt.Sprintf("chassis sensor reading for a reading type with no dedicated metric; the units are in the %q label", "sensor_units"), ChassisSensorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "sensor_health", fmt.Sprintf("chassis sensor health,%s", CommonHealthHelp), ChassisSensorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "sensor_state", fmt.Sprintf("chassis sensor state,%s", CommonStateHelp), ChassisSensorLabelNames)

	// Note: chassis_gpu_total_power_watts is now collected via TelemetryService (HGX_PlatformEnvironmentMetrics_0)

	return chassisMetrics
}

// NewChassisCollector returns a collector that collecting chassis statistics
func NewChassisCollector(collectorName string, redfishClient *gofish.APIClient, logger *slog.Logger, config config.ChassisCollectorConfig) (*ChassisCollector, error) {
	// get service from redfish client

	// Compile the chassis filters up front so a bad pattern is reported at collector
	// construction rather than silently filtering nothing on every scrape.
	var chassisInclude, chassisExclude *regexp.Regexp
	if config.ChassisInclude != "" {
		re, err := regexp.Compile(config.ChassisInclude)
		if err != nil {
			return nil, fmt.Errorf("invalid chassis_include pattern %q: %w", config.ChassisInclude, err)
		}
		chassisInclude = re
	}
	if config.ChassisExclude != "" {
		re, err := regexp.Compile(config.ChassisExclude)
		if err != nil {
			return nil, fmt.Errorf("invalid chassis_exclude pattern %q: %w", config.ChassisExclude, err)
		}
		chassisExclude = re
	}

	return &ChassisCollector{
		redfishClient:  redfishClient,
		metrics:        chassisMetrics,
		config:         config,
		chassisInclude: chassisInclude,
		chassisExclude: chassisExclude,
		logger:         logger,
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

func (c *ChassisCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
	c.collect(ctx, ch)
}

// Collect implemented prometheus.Collector
func (c *ChassisCollector) Collect(ch chan<- prometheus.Metric) {
	c.collect(context.TODO(), ch)
}

func (c *ChassisCollector) collect(ctx context.Context, ch chan<- prometheus.Metric) {
	if ctx.Err() != nil {
		c.logger.With("error", ctx.Err(), "collector", "chassis").Debug("skipping collection")
		return
	}
	logger := c.logger.With(slog.String("collector", "ChassisCollector"))
	service := c.redfishClient.Service

	if ctx.Err() != nil {
		c.logger.With("error", ctx.Err(), "collector", "chassis").Debug("skipping collection")
		return
	}
	// get a list of chassis from service
	if chassises, err := service.Chassis(); err != nil {
		logger.Error("error getting chassis from service", slog.String("operation", "service.Chassis()"), slog.Any("error", err))
	} else {
		// process the chassises
		for _, chassis := range chassises {
			if ctx.Err() != nil {
				c.logger.With("error", ctx.Err()).Warn("skipping further collection")
				continue
			}
			if c.skipChassis(chassis.ID) {
				logger.Debug("chassis filtered out by configuration", slog.String("Chassis", chassis.ID))
				continue
			}
			chassisLogger := logger.With(slog.String("Chassis", chassis.ID))
			chassisLogger.Info("collector scrape started")
			chassisID := chassis.ID
			chassisStatus := chassis.Status
			chassisStatusState := chassisStatus.State
			chassisStatusHealth := chassisStatus.Health
			chassisStatusHealthRollup := chassisStatus.HealthRollup
			ChassisLabelValues := []string{"chassis", chassisID}
			if chassisStatusHealthValue, ok := parseCommonStatusHealth(chassisStatusHealth); ok {
				ch <- prometheus.MustNewConstMetric(c.metrics["chassis_health"].desc, prometheus.GaugeValue, chassisStatusHealthValue, ChassisLabelValues...)
			}
			if chassisStatusHealthRollupValue, ok := parseCommonStatusHealth(chassisStatusHealthRollup); ok {
				ch <- prometheus.MustNewConstMetric(c.metrics["chassis_health_rollup"].desc, prometheus.GaugeValue, chassisStatusHealthRollupValue, ChassisLabelValues...)
			}
			if chassisStatusStateValue, ok := parseCommonStatusState(chassisStatusState); ok {
				ch <- prometheus.MustNewConstMetric(c.metrics["chassis_state"].desc, prometheus.GaugeValue, chassisStatusStateValue, ChassisLabelValues...)
			}

			chassisManufacturer := chassis.Manufacturer
			chassisModel := chassis.Model
			chassisPartNumber := chassis.PartNumber
			chassisSKU := chassis.SKU
			ChassisModelLabelValues := []string{"chassis", chassisID, chassisManufacturer, chassisModel, chassisPartNumber, chassisSKU}
			ch <- prometheus.MustNewConstMetric(c.metrics["chassis_model_info"].desc, prometheus.GaugeValue, 1, ChassisModelLabelValues...)

			// Whether this chassis advertises either of the deprecated Thermal/Power
			// schemas. Newer platforms (for example an NVL72 tray, where no chassis
			// implements either) express the same readings through Sensors instead, which is
			// collected below only when these are absent so the two paths never emit
			// duplicate series.
			//
			// Read from the advertised links rather than from whether a fetch returned data,
			// so that disabling Thermal or Power by configuration does not make every
			// chassis look like a modern one and pull in its whole Sensors collection.
			legacyThermalOrPower := chassisAdvertisesThermalOrPower(chassis)

			chassisThermal, err := c.chassisThermal(chassis)
			if err != nil {
				chassisLogger.Error("error getting thermal data from chassis", slog.String("operation", "chassis.Thermal()"), slog.Any("error", err))
			} else if chassisThermal == nil {
				chassisLogger.Debug("no thermal data found", slog.String("operation", "chassis.Thermal()"))
			} else {
				// process temperature and fans
				chassisTemperatures := chassisThermal.Temperatures
				chassisFans := chassisThermal.Fans
				eg := newRecoverGroup(ctx)
				for _, chassisTemperature := range chassisTemperatures {
					eg.Go(func() error {
						parseChassisTemperature(ch, chassisID, chassisTemperature)
						return nil
					})
				}
				for _, chassisFan := range chassisFans {
					eg.Go(func() error {
						parseChassisFan(ch, chassisID, chassisFan)
						return nil
					})
				}
				if err := eg.Wait(); err != nil {
					chassisLogger.Error("goroutine error", slog.Any("error", err))
				}
			}
			// leakDetectorIDs maps the Id of each leak detector on this chassis to its
			// parent LeakDetection Id. Each detector is dual-surfaced: the LeakDetector
			// resource carries the discrete DetectorState, while a Sensor of the same Id
			// carries the analog voltage and its threshold. This lets the Sensors pass
			// below recognise those readings instead of treating them as ordinary voltages.
			leakDetectorIDs := map[string]string{}

			chassisThermalSubsystem, err := c.chassisThermalSubsystem(chassis)
			if err != nil {
				chassisLogger.Error("error getting thermal subsystem from chassis", slog.String("operation", "chassis.ThermalSubsystem()"), slog.Any("error", err))
			} else if chassisThermalSubsystem == nil {
				chassisLogger.Debug("no thermal subsystem found", slog.String("operation", "chassis.ThermalSubsystem()"))
			} else {
				leakDetection, leakDetectors := c.getLeakDetection(chassisThermalSubsystem, chassisLogger)

				leakDetectionID := "LeakDetection"
				if leakDetection != nil {
					if leakDetection.ID != "" {
						leakDetectionID = leakDetection.ID
					}
					parseLeakDetection(ch, chassisID, leakDetectionID, leakDetection)
				}

				for _, ld := range leakDetectors {
					leakDetectorIDs[ld.ID] = leakDetectionID
				}

				if len(leakDetectors) > 0 {
					egLD := newRecoverGroup(ctx)
					for _, ld := range leakDetectors {
						egLD.Go(func() error {
							parseLeakDetector(ch, chassisID, leakDetectionID, ld)
							return nil
						})
					}
					if err := egLD.Wait(); err != nil {
						chassisLogger.Error("goroutine error", slog.Any("error", err))
					}
				} else {
					chassisLogger.Debug("no leak detectors found")
				}
			}

			chassisPowerInfo, err := c.chassisPower(chassis)
			if err != nil {
				chassisLogger.Error("error getting power data from chassis", slog.String("operation", "chassis.Power()"), slog.Any("error", err))
			} else if chassisPowerInfo == nil {
				chassisLogger.Debug("no power data found", slog.String("operation", "chassis.Power()"))
			} else {
				egPower := newRecoverGroup(ctx)

				// power voltages
				for _, chassisPowerInfoVoltage := range chassisPowerInfo.Voltages {
					egPower.Go(func() error {
						parseChassisPowerInfoVoltage(ch, chassisID, chassisPowerInfoVoltage)
						return nil
					})
				}

				// power control
				for _, chassisPowerInfoPowerControl := range chassisPowerInfo.PowerControl {
					egPower.Go(func() error {
						parseChassisPowerInfoPowerControl(ch, chassisID, chassisPowerInfoPowerControl)
						return nil
					})
				}

				// powerSupply
				for _, chassisPowerInfoPowerSupply := range chassisPowerInfo.PowerSupplies {
					egPower.Go(func() error {
						parseChassisPowerInfoPowerSupply(ch, chassisID, chassisPowerInfoPowerSupply)
						return nil
					})
				}
				if err := egPower.Wait(); err != nil {
					chassisLogger.Error("goroutine error", slog.Any("error", err))
				}
			}

			// Sensors carry the readings that Thermal/Power would have provided on older
			// platforms, plus the analog side of the leak detectors. Consulted when the
			// legacy schemas are absent, so this is a no-op on existing hardware, and
			// additionally on any chassis carrying leak detectors so their voltages and
			// thresholds are picked up even where Thermal/Power do exist.
			if !c.config.DisableSensors && (!legacyThermalOrPower || len(leakDetectorIDs) > 0) {
				sensors, err := c.getChassisSensors(ctx, chassis, leakDetectorIDs, chassisLogger)
				if err != nil {
					chassisLogger.Error("error getting sensors from chassis", slog.String("operation", "chassis.Sensors()"), slog.Any("error", err))
				} else if len(sensors) == 0 {
					chassisLogger.Debug("no sensors found", slog.String("operation", "chassis.Sensors()"))
				} else {
					egSensors := newRecoverGroup(ctx)
					for _, sensor := range sensors {
						egSensors.Go(func() error {
							parseChassisSensor(ch, chassisID, sensor, leakDetectorIDs)
							return nil
						})
					}
					if err := egSensors.Wait(); err != nil {
						chassisLogger.Error("goroutine error", slog.Any("error", err))
					}
				}
			}

			// process NetworkAdapter
			networkAdapters, err := c.chassisNetworkAdapters(chassis)
			if err != nil {
				chassisLogger.Error("error getting network adapters data from chassis", slog.String("operation", "chassis.NetworkAdapters()"), slog.Any("error", err))
			} else if networkAdapters == nil {
				chassisLogger.Info("no network adapters data found", slog.String("operation", "chassis.NetworkAdapters()"))
			} else {
				egNA := newRecoverGroup(ctx)
				for _, networkAdapter := range networkAdapters {
					egNA.Go(func() error {
						return parseNetworkAdapter(ctx, ch, chassisID, networkAdapter)
					})
				}
				if err := egNA.Wait(); err != nil {
					chassisLogger.Error("error getting network ports from network adapter", slog.String("operation", "chassis.NetworkAdapters()"), slog.Any("error", err))
				}
			}

			physicalSecurity := chassis.PhysicalSecurity
			if physicalSecurity != (schemas.PhysicalSecurity{}) {
				physicalSecurityIntrusionSensor := physicalSecurity.IntrusionSensor
				physicalSecurityIntrusionSensorNumber := fmt.Sprint(physicalSecurity.IntrusionSensorNumber) //nolint:staticcheck
				physicalSecurityIntrusionSensorReArmMethod := string(physicalSecurity.IntrusionSensorReArm)

				if phySecIntrusionSensor, ok := parsePhySecIntrusionSensor(physicalSecurityIntrusionSensor); ok {
					ChassisPhysicalSecurityLabelValues := []string{"physical_security", chassisID, physicalSecurityIntrusionSensorNumber, physicalSecurityIntrusionSensorReArmMethod}
					ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_physical_security_sensor_state"].desc, prometheus.GaugeValue, phySecIntrusionSensor, ChassisPhysicalSecurityLabelValues...)
				}
			}

			chassisLogger.Info("collector scrape completed")
		}
	}

	c.collectorScrapeStatus.WithLabelValues("chassis").Set(float64(1))
}

// Describe implemented prometheus.Collector
func (c *ChassisCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metrics {
		ch <- metric.desc
	}
	c.collectorScrapeStatus.Describe(ch)

}

// chassisThermal returns the deprecated Thermal resource, or (nil, nil) when the
// subsystem is disabled by configuration or not implemented by the chassis.
func (c *ChassisCollector) chassisThermal(chassis *schemas.Chassis) (*schemas.Thermal, error) {
	if c.config.DisableThermal {
		return nil, nil
	}
	return chassis.Thermal()
}

// chassisPower returns the deprecated Power resource, or (nil, nil) when the subsystem
// is disabled by configuration or not implemented by the chassis.
func (c *ChassisCollector) chassisPower(chassis *schemas.Chassis) (*schemas.Power, error) {
	if c.config.DisablePower {
		return nil, nil
	}
	return chassis.Power()
}

// chassisThermalSubsystem returns the ThermalSubsystem resource, or (nil, nil) when the
// subsystem is disabled by configuration or not implemented by the chassis.
func (c *ChassisCollector) chassisThermalSubsystem(chassis *schemas.Chassis) (*schemas.ThermalSubsystem, error) {
	if c.config.DisableThermalSubsystem {
		return nil, nil
	}
	return chassis.ThermalSubsystem()
}

// chassisNetworkAdapters returns the chassis network adapters, or (nil, nil) when the
// subsystem is disabled by configuration or not implemented by the chassis.
func (c *ChassisCollector) chassisNetworkAdapters(chassis *schemas.Chassis) ([]*schemas.NetworkAdapter, error) {
	if c.config.DisableNetworkAdapters {
		return nil, nil
	}
	return chassis.NetworkAdapters()
}

// sensorCollection is the shape of an expanded Sensors collection response. Members are
// full Sensor bodies rather than links, so one request replaces one-per-sensor.
type sensorCollection struct {
	Members []*schemas.Sensor `json:"Members"`
}

// chassisAdvertisesThermalOrPower reports whether the chassis links either of the
// deprecated Thermal or Power resources.
//
// This reads the advertised links from the chassis payload rather than inferring from a
// fetch, so the answer does not change when a subsystem is disabled by configuration. A
// chassis whose payload cannot be inspected is treated as advertising them, which is the
// conservative answer: it suppresses the Sensors fallback rather than risking an
// unnecessary request per chassis.
func chassisAdvertisesThermalOrPower(chassis *schemas.Chassis) bool {
	if len(chassis.RawData) == 0 {
		return true
	}
	var raw struct {
		Thermal *struct {
			ODataID string `json:"@odata.id"`
		} `json:"Thermal"`
		Power *struct {
			ODataID string `json:"@odata.id"`
		} `json:"Power"`
	}
	if err := json.Unmarshal(chassis.RawData, &raw); err != nil {
		return true
	}
	return (raw.Thermal != nil && raw.Thermal.ODataID != "") || (raw.Power != nil && raw.Power.ODataID != "")
}

// getChassisSensors returns sensors for a chassis in a single request.
//
// The whole collection is requested with $expand so the BMC inlines every member body. This
// matters for request load: a single NVL72 tray carries a few hundred sensors across its
// chassis, and gofish's typed accessor issues one request per member, which with
// max_concurrent_requests defaulting to 1 serialises into a very long scrape.
//
// If the BMC does not honour $expand there is deliberately no fan-out to one request per
// sensor. Doing so would multiply this collector's request count against the very BMCs least
// able to absorb it. Instead only the named sensors are fetched individually — the leak
// detectors, whose readings are safety relevant and few in number — and the bulk sensor
// telemetry is skipped with a warning.
func (c *ChassisCollector) getChassisSensors(ctx context.Context, chassis *schemas.Chassis, required map[string]string, logger *slog.Logger) ([]*schemas.Sensor, error) {
	rfClient := c.redfishClient.WithContext(ctx)
	sensorsPath := chassis.ODataID + "/Sensors"

	response, err := rfClient.Get(sensorsPath + `?$expand=.($levels=1)`)
	if err != nil {
		logger.Debug("expanded Sensors request failed", slog.Any("error", err))
		return c.getNamedSensors(ctx, sensorsPath, required, logger), nil
	}
	defer response.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var agg sensorCollection
	if err := json.Unmarshal(body, &agg); err != nil {
		return nil, err
	}
	// A BMC that ignores $expand answers with link-only members, which unmarshal into
	// Sensors carrying no Id.
	if len(agg.Members) > 0 && agg.Members[0] != nil && agg.Members[0].ID != "" {
		return agg.Members, nil
	}
	if len(agg.Members) > 0 {
		logger.Warn("BMC did not honour $expand on Sensors; collecting required sensors only",
			slog.String("sensors", sensorsPath),
			slog.Int("skipped", len(agg.Members)),
		)
	}
	return c.getNamedSensors(ctx, sensorsPath, required, logger), nil
}

// getNamedSensors fetches individually named sensors, one request each. Callers must keep
// the set small.
func (c *ChassisCollector) getNamedSensors(ctx context.Context, sensorsPath string, required map[string]string, logger *slog.Logger) []*schemas.Sensor {
	if len(required) == 0 {
		return nil
	}
	client := c.redfishClient.WithContext(ctx).GetService().GetClient()
	sensors := make([]*schemas.Sensor, 0, len(required))
	for sensorID := range required {
		sensor, err := schemas.GetSensor(client, sensorsPath+"/"+sensorID)
		if err != nil {
			logger.Debug("could not get named sensor", slog.String("sensor_id", sensorID), slog.Any("error", err))
			continue
		}
		sensors = append(sensors, sensor)
	}
	return sensors
}

// getLeakDetection returns the chassis LeakDetection resource and its detectors.
//
// Both return values may be nil: leak detection is optional, and some BMCs advertise a
// LeakDetection link that then 404s (observed on SYS-A21GE-NBRT), which is reported at
// debug level rather than as a collection error.
func (c *ChassisCollector) getLeakDetection(thermalSubsystem *schemas.ThermalSubsystem, logger *slog.Logger) (*schemas.LeakDetection, []*schemas.LeakDetector) {
	leakDetection, err := thermalSubsystem.LeakDetection()
	if err != nil {
		// An advertised-but-absent LeakDetection is a known OEM quirk, not an error.
		logger.Debug("could not get LeakDetection from ThermalSubsystem", slog.Any("error", err))
		return nil, nil
	}
	if leakDetection == nil {
		return nil, nil
	}

	detectors, err := leakDetection.LeakDetectors()
	if err != nil {
		logger.Error("failed obtaining LeakDetectors", slog.String("leak_detection", leakDetection.ODataID), slog.Any("error", err))
		return leakDetection, nil
	}

	return leakDetection, detectors
}

func parseChassisTemperature(ch chan<- prometheus.Metric, chassisID string, chassisTemperature schemas.Temperature) {
	chassisTemperatureSensorName := chassisTemperature.Name
	chassisTemperatureSensorID := chassisTemperature.MemberID
	chassisTemperatureStatus := chassisTemperature.Status
	chassisTemperatureLabelvalues := []string{"temperature", chassisID, chassisTemperatureSensorName, chassisTemperatureSensorID}

	chassisTemperatureStatusHealth := chassisTemperatureStatus.Health
	if chassisTemperatureStatusHealthValue, ok := parseCommonStatusHealth(chassisTemperatureStatusHealth); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_temperature_sensor_health"].desc, prometheus.GaugeValue, chassisTemperatureStatusHealthValue, chassisTemperatureLabelvalues...)
	}

	chassisTemperatureStatusState := chassisTemperatureStatus.State
	//			chassisTemperatureStatusLabelNames :=[]string{BaseLabelNames,"temperature_sensor_name","temperature_sensor_member_id")
	//		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_temperature_status_health"].desc, prometheus.GaugeValue, parseCommonStatusHealth(chassisTemperatureStatusHealth), chassisTemperatureLabelvalues...)
	if chassisTemperatureStatusStateValue, ok := parseCommonStatusState(chassisTemperatureStatusState); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_temperature_sensor_state"].desc, prometheus.GaugeValue, chassisTemperatureStatusStateValue, chassisTemperatureLabelvalues...)
	}

	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_temperature_celsius"].desc, prometheus.GaugeValue, gofish.Deref(chassisTemperature.ReadingCelsius), chassisTemperatureLabelvalues...)
}

func parseChassisFan(ch chan<- prometheus.Metric, chassisID string, chassisFan schemas.ThermalFan) {
	chassisFanID := chassisFan.MemberID
	chassisFanName := chassisFan.Name
	chassisFanStaus := chassisFan.Status
	chassisFanStausHealth := chassisFanStaus.Health
	chassisFanStausState := chassisFanStaus.State
	chassisFanRPM := intPtrToFloat64(chassisFan.Reading)
	chassisFanUnit := chassisFan.ReadingUnits
	chassisFanRPMLowerCriticalThreshold := intPtrToFloat64(chassisFan.LowerThresholdCritical)
	chassisFanRPMUpperCriticalThreshold := intPtrToFloat64(chassisFan.UpperThresholdCritical)
	chassisFanRPMLowerFatalThreshold := intPtrToFloat64(chassisFan.LowerThresholdFatal)
	chassisFanRPMUpperFatalThreshold := intPtrToFloat64(chassisFan.UpperThresholdFatal)
	chassisFanRPMMin := intPtrToFloat64(chassisFan.MinReadingRange)
	chassisFanRPMMax := intPtrToFloat64(chassisFan.MaxReadingRange)

	chassisFanPercentage := chassisFanRPM
	if chassisFanUnit != schemas.PercentReadingUnits {
		// Some vendors (e.g. PowerEdge C6420) report null RPMs for Min/Max, as well as Lower/UpperFatal,
		// but provide Lower/UpperCritical, so use largest non-null for max. However, we can't know if
		// min is null (reported as zero by gofish) or just zero, so we'll have to assume a min of zero
		// if Min is not reported...
		min := chassisFanRPMMin
		max := math.Max(math.Max(chassisFanRPMMax, chassisFanRPMUpperFatalThreshold), chassisFanRPMUpperCriticalThreshold)
		chassisFanPercentage = 0
		if max != 0 {
			chassisFanPercentage = float64((chassisFanRPM+min)/max) * 100
		}
	}

	//			chassisFanStatusLabelNames :=[]string{BaseLabelNames,"fan_name","fan_member_id")
	chassisFanLabelvalues := []string{"fan", chassisID, chassisFanName, chassisFanID, strings.ToLower(string(chassisFanUnit))} // e.g. RPM -> rpm, Percentage -> percentage

	if chassisFanStausHealthValue, ok := parseCommonStatusHealth(chassisFanStausHealth); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_health"].desc, prometheus.GaugeValue, chassisFanStausHealthValue, chassisFanLabelvalues...)
	}

	if chassisFanStausStateValue, ok := parseCommonStatusState(chassisFanStausState); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_state"].desc, prometheus.GaugeValue, chassisFanStausStateValue, chassisFanLabelvalues...)
	}
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm"].desc, prometheus.GaugeValue, chassisFanRPM, chassisFanLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_min"].desc, prometheus.GaugeValue, chassisFanRPMMin, chassisFanLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_max"].desc, prometheus.GaugeValue, chassisFanRPMMax, chassisFanLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_percentage"].desc, prometheus.GaugeValue, chassisFanPercentage, chassisFanLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_lower_threshold_critical"].desc, prometheus.GaugeValue, chassisFanRPMLowerCriticalThreshold, chassisFanLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_upper_threshold_critical"].desc, prometheus.GaugeValue, chassisFanRPMUpperCriticalThreshold, chassisFanLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_lower_threshold_fatal"].desc, prometheus.GaugeValue, chassisFanRPMLowerFatalThreshold, chassisFanLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_upper_threshold_fatal"].desc, prometheus.GaugeValue, chassisFanRPMUpperFatalThreshold, chassisFanLabelvalues...)
}

// thresholdReading flattens an optional Redfish threshold to a float.
func thresholdReading(t schemas.Threshold) float64 {
	return gofish.Deref(t.Reading)
}

// parseChassisSensor emits metrics for a single Sensor resource.
//
// Sensors are the modern replacement for the deprecated Thermal and Power schemas. Where a
// ReadingType maps unambiguously onto an existing chassis metric family it is folded into
// that family, so a platform that only implements Sensors produces the same series names as
// one that implements Thermal/Power. Anything else lands in a sensor_* catch-all rather
// than being guessed at or dropped.
//
// Readings are deliberately not inferred from sensor naming. On an NVL72 tray, fan PWM
// duty cycle and CPU core utilisation are both ReadingType "Percent" and neither carries a
// distinguishing PhysicalContext, so treating "Percent" as a fan speed would mislabel over
// a hundred sensors per tray.
func parseChassisSensor(ch chan<- prometheus.Metric, chassisID string, sensor *schemas.Sensor, leakDetectorIDs map[string]string) {
	if sensor == nil || sensor.ID == "" {
		return
	}
	reading := gofish.Deref(sensor.Reading)

	// The analog half of a leak detector, correlated by Id with the LeakDetector
	// resources enumerated from ThermalSubsystem.
	if leakDetectionID, ok := leakDetectorIDs[sensor.ID]; ok {
		labelValues := []string{"leak_detector", chassisID, leakDetectionID, sensor.ID}
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detector_volts"].desc, prometheus.GaugeValue, reading, labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detector_volts_lower_threshold_critical"].desc, prometheus.GaugeValue, thresholdReading(sensor.Thresholds.LowerCritical), labelValues...)
		return
	}

	switch sensor.ReadingType {
	case schemas.TemperatureReadingType:
		labelValues := []string{"temperature", chassisID, sensor.Name, sensor.ID}
		if health, ok := parseCommonStatusHealth(sensor.Status.Health); ok {
			ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_temperature_sensor_health"].desc, prometheus.GaugeValue, health, labelValues...)
		}
		if state, ok := parseCommonStatusState(sensor.Status.State); ok {
			ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_temperature_sensor_state"].desc, prometheus.GaugeValue, state, labelValues...)
		}
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_temperature_celsius"].desc, prometheus.GaugeValue, reading, labelValues...)

	case schemas.RotationalReadingType:
		// A rotational reading is a fan tachometer. Percentage is derived the same way
		// parseChassisFan derives it, so the two paths agree.
		labelValues := []string{"fan", chassisID, sensor.Name, sensor.ID, strings.ToLower(sensor.ReadingUnits)}
		rpmMin := gofish.Deref(sensor.ReadingRangeMin)
		rpmMax := gofish.Deref(sensor.ReadingRangeMax)
		percentage := float64(0)
		if rpmMax != 0 {
			percentage = ((reading + rpmMin) / rpmMax) * 100
		}
		if health, ok := parseCommonStatusHealth(sensor.Status.Health); ok {
			ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_health"].desc, prometheus.GaugeValue, health, labelValues...)
		}
		if state, ok := parseCommonStatusState(sensor.Status.State); ok {
			ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_state"].desc, prometheus.GaugeValue, state, labelValues...)
		}
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm"].desc, prometheus.GaugeValue, reading, labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_min"].desc, prometheus.GaugeValue, rpmMin, labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_max"].desc, prometheus.GaugeValue, rpmMax, labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_percentage"].desc, prometheus.GaugeValue, percentage, labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_lower_threshold_critical"].desc, prometheus.GaugeValue, thresholdReading(sensor.Thresholds.LowerCritical), labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_upper_threshold_critical"].desc, prometheus.GaugeValue, thresholdReading(sensor.Thresholds.UpperCritical), labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_lower_threshold_fatal"].desc, prometheus.GaugeValue, thresholdReading(sensor.Thresholds.LowerFatal), labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_upper_threshold_fatal"].desc, prometheus.GaugeValue, thresholdReading(sensor.Thresholds.UpperFatal), labelValues...)

	case schemas.VoltageReadingType:
		labelValues := []string{"power_voltage", chassisID, sensor.Name, sensor.ID}
		if state, ok := parseCommonStatusState(sensor.Status.State); ok {
			ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_voltage_state"].desc, prometheus.GaugeValue, state, labelValues...)
		}
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_voltage_volts"].desc, prometheus.GaugeValue, reading, labelValues...)

	default:
		parseChassisSensorCatchall(ch, chassisID, sensor, reading)
	}
}

// parseChassisSensorCatchall emits a sensor whose ReadingType has no curated equivalent.
func parseChassisSensorCatchall(ch chan<- prometheus.Metric, chassisID string, sensor *schemas.Sensor, reading float64) {
	labelValues := []string{"sensor", chassisID, sensor.Name, sensor.ID, sensor.ReadingUnits, string(sensor.PhysicalContext)}

	metricKey := "chassis_sensor_reading"
	switch sensor.ReadingType {
	case schemas.PowerReadingType:
		metricKey = "chassis_sensor_watts"
	case schemas.CurrentReadingType:
		metricKey = "chassis_sensor_amperes"
	case schemas.EnergyJoulesReadingType:
		metricKey = "chassis_sensor_joules"
	case schemas.FrequencyReadingType:
		metricKey = "chassis_sensor_hertz"
	case schemas.PercentReadingType:
		metricKey = "chassis_sensor_percent"
	}

	if health, ok := parseCommonStatusHealth(sensor.Status.Health); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_sensor_health"].desc, prometheus.GaugeValue, health, labelValues...)
	}
	if state, ok := parseCommonStatusState(sensor.Status.State); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_sensor_state"].desc, prometheus.GaugeValue, state, labelValues...)
	}
	ch <- prometheus.MustNewConstMetric(chassisMetrics[metricKey].desc, prometheus.GaugeValue, reading, labelValues...)
}

// leakDetectorEnabled reports the detector's Enabled property, and whether the firmware
// reported it at all. Redfish adds Enabled in LeakDetector v1.3.0; when a detector really
// is disabled the spec also requires DetectorState to be Unavailable, so a missing
// property is safe to omit rather than assume.
func leakDetectorEnabled(ld *schemas.LeakDetector) (enabled bool, reported bool) {
	if len(ld.RawData) == 0 {
		return false, false
	}
	var raw struct {
		Enabled *bool `json:"Enabled"`
	}
	if err := json.Unmarshal(ld.RawData, &raw); err != nil || raw.Enabled == nil {
		return false, false
	}
	return *raw.Enabled, true
}

// parseLeakDetection emits the rollup status of the leak detection subsystem itself.
// A detector reporting a leak does not necessarily degrade this rollup, so it is a
// supplement to the per-detector state rather than a substitute for it.
func parseLeakDetection(ch chan<- prometheus.Metric, chassisID, leakDetectionID string, ldn *schemas.LeakDetection) {
	labelValues := []string{"leak_detection", chassisID, leakDetectionID}

	if statusHealth, ok := parseCommonStatusHealth(ldn.Status.Health); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detection_health"].desc, prometheus.GaugeValue, statusHealth, labelValues...)
	}
	if statusState, ok := parseCommonStatusState(ldn.Status.State); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detection_state"].desc, prometheus.GaugeValue, statusState, labelValues...)
	}
}

func parseLeakDetector(ch chan<- prometheus.Metric, chassisID, leakDetectionID string, ld *schemas.LeakDetector) {
	ldID := ld.ID
	labelValues := []string{"leak_detector", chassisID, leakDetectionID, ldID}

	if statusHealth, ok := parseCommonStatusHealth(ld.Status.Health); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detector_health"].desc, prometheus.GaugeValue, statusHealth, labelValues...)
	}

	// DetectorState, not Status.Health, is the leak signal.
	if detectorState, ok := parseDetectorState(ld.DetectorState); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detector_state"].desc, prometheus.GaugeValue, detectorState, labelValues...)
	}

	// Enabled was added in LeakDetector v1.3.0 and is absent on older firmware (including
	// the GB300 trays, whose detector bodies carry only DetectorState, type and status).
	// gofish types it as a plain bool, so an absent property is indistinguishable from an
	// explicit false and would report every healthy detector as disabled. Consult the raw
	// payload instead and stay silent when the property was not reported.
	if enabled, ok := leakDetectorEnabled(ld); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detector_enabled"].desc, prometheus.GaugeValue, boolToFloat64(enabled), labelValues...)
	}

	infoLabelValues := []string{"leak_detector", chassisID, leakDetectionID, ldID, string(ld.LeakDetectorType), string(ld.PhysicalContext), string(ld.PhysicalSubContext)}
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detector_info"].desc, prometheus.GaugeValue, 1, infoLabelValues...)
}

func parseChassisPowerInfoVoltage(ch chan<- prometheus.Metric, chassisID string, chassisPowerInfoVoltage schemas.Voltage) {
	chassisPowerInfoVoltageName := chassisPowerInfoVoltage.Name
	chassisPowerInfoVoltageID := chassisPowerInfoVoltage.MemberID
	chassisPowerInfoVoltageNameReadingVolts := chassisPowerInfoVoltage.ReadingVolts
	chassisPowerInfoVoltageState := chassisPowerInfoVoltage.Status.State
	chassisPowerVoltageLabelvalues := []string{"power_voltage", chassisID, chassisPowerInfoVoltageName, chassisPowerInfoVoltageID}
	if chassisPowerInfoVoltageStateValue, ok := parseCommonStatusState(chassisPowerInfoVoltageState); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_voltage_state"].desc, prometheus.GaugeValue, chassisPowerInfoVoltageStateValue, chassisPowerVoltageLabelvalues...)
	}
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_voltage_volts"].desc, prometheus.GaugeValue, float32PtrToFloat64(chassisPowerInfoVoltageNameReadingVolts), chassisPowerVoltageLabelvalues...)
}

func parseChassisPowerInfoPowerControl(ch chan<- prometheus.Metric, chassisID string, chassisPowerInfoPowerControl schemas.PowerControl) {
	name := chassisPowerInfoPowerControl.Name
	id := chassisPowerInfoPowerControl.MemberID
	pm := chassisPowerInfoPowerControl.PowerMetrics
	chassisPowerVoltageLabelvalues := []string{"power_wattage", chassisID, name, id}
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_average_consumed_watts"].desc, prometheus.GaugeValue, float32PtrToFloat64(pm.AverageConsumedWatts), chassisPowerVoltageLabelvalues...)
}

func parseChassisPowerInfoPowerSupply(ch chan<- prometheus.Metric, chassisID string, chassisPowerInfoPowerSupply schemas.PowerSupply) {
	chassisPowerInfoPowerSupplyName := chassisPowerInfoPowerSupply.Name
	// This is optional in some devices causing duplicate metrics
	chassisPowerInfoPowerSupplyID := chassisPowerInfoPowerSupply.MemberID
	if chassisPowerInfoPowerSupplyID == "" {
		slog.Debug("PowerSupply ID is empty, using serial number as ID")
		chassisPowerInfoPowerSupplyID = chassisPowerInfoPowerSupply.SerialNumber
		if chassisPowerInfoPowerSupplyID == "" {
			slog.Error("PowerSupply ID and serial number empty - skipping power supply")
			return
		}
	}
	chassisPowerInfoPowerSupplyEfficiencyPercent := gofish.Deref(chassisPowerInfoPowerSupply.EfficiencyPercent)
	chassisPowerInfoPowerSupplyPowerCapacityWatts := chassisPowerInfoPowerSupply.PowerCapacityWatts
	chassisPowerInfoPowerSupplyPowerInputWatts := chassisPowerInfoPowerSupply.PowerInputWatts
	chassisPowerInfoPowerSupplyPowerOutputWatts := chassisPowerInfoPowerSupply.PowerOutputWatts
	chassisPowerInfoPowerSupplyLastPowerOutputWatts := chassisPowerInfoPowerSupply.LastPowerOutputWatts

	chassisPowerInfoPowerSupplyState := chassisPowerInfoPowerSupply.Status.State
	chassisPowerInfoPowerSupplyHealth := chassisPowerInfoPowerSupply.Status.Health
	chassisPowerSupplyLabelvalues := []string{"power_supply", chassisID, chassisPowerInfoPowerSupplyName, chassisPowerInfoPowerSupplyID}
	if chassisPowerInfoPowerSupplyStateValue, ok := parseCommonStatusState(chassisPowerInfoPowerSupplyState); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_powersupply_state"].desc, prometheus.GaugeValue, chassisPowerInfoPowerSupplyStateValue, chassisPowerSupplyLabelvalues...)
	}
	if chassisPowerInfoPowerSupplyHealthValue, ok := parseCommonStatusHealth(chassisPowerInfoPowerSupplyHealth); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_powersupply_health"].desc, prometheus.GaugeValue, chassisPowerInfoPowerSupplyHealthValue, chassisPowerSupplyLabelvalues...)
	}
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_powersupply_power_efficiency_percentage"].desc, prometheus.GaugeValue, chassisPowerInfoPowerSupplyEfficiencyPercent, chassisPowerSupplyLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_powersupply_last_power_output_watts"].desc, prometheus.GaugeValue, float32PtrToFloat64(chassisPowerInfoPowerSupplyLastPowerOutputWatts), chassisPowerSupplyLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_powersupply_power_capacity_watts"].desc, prometheus.GaugeValue, float32PtrToFloat64(chassisPowerInfoPowerSupplyPowerCapacityWatts), chassisPowerSupplyLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_powersupply_power_input_watts"].desc, prometheus.GaugeValue, float32PtrToFloat64(chassisPowerInfoPowerSupplyPowerInputWatts), chassisPowerSupplyLabelvalues...)
	ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_power_powersupply_power_output_watts"].desc, prometheus.GaugeValue, float32PtrToFloat64(chassisPowerInfoPowerSupplyPowerOutputWatts), chassisPowerSupplyLabelvalues...)
}

func parseNetworkAdapter(ctx context.Context, ch chan<- prometheus.Metric, chassisID string, networkAdapter *schemas.NetworkAdapter) error {
	networkAdapterName := networkAdapter.Name
	networkAdapterID := networkAdapter.ID
	networkAdapterState := networkAdapter.Status.State
	networkAdapterHealthState := networkAdapter.Status.Health
	chassisNetworkAdapterLabelValues := []string{"network_adapter", chassisID, networkAdapterName, networkAdapterID}
	if networkAdapterStateValue, ok := parseCommonStatusState(networkAdapterState); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_network_adapter_state"].desc, prometheus.GaugeValue, networkAdapterStateValue, chassisNetworkAdapterLabelValues...)
	}
	if networkAdapterHealthStateValue, ok := parseCommonStatusHealth(networkAdapterHealthState); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_network_adapter_health_state"].desc, prometheus.GaugeValue, networkAdapterHealthStateValue, chassisNetworkAdapterLabelValues...)
	}

	networkPorts, err := networkAdapter.NetworkPorts()
	if err != nil {
		return err
	}
	egPort := newRecoverGroup(ctx)
	for _, networkPort := range networkPorts {
		egPort.Go(func() error {
			parseNetworkPort(ch, chassisID, networkPort, networkAdapterName, networkAdapterID)
			return nil
		})
	}
	if err := egPort.Wait(); err != nil {
		return err
	}
	return nil
}

func parseNetworkPort(ch chan<- prometheus.Metric, chassisID string, networkPort *schemas.NetworkPort, networkAdapterName string, networkAdapterID string) {
	networkPortName := networkPort.Name
	networkPortID := networkPort.ID
	networkPortState := networkPort.Status.State
	networkLinkStatus := networkPort.LinkStatus
	networkPortLinkType := networkPort.ActiveLinkTechnology
	networkPortLinkSpeed := fmt.Sprintf("%d Mbps", networkPort.CurrentLinkSpeedMbps)
	networkPortHealthState := networkPort.Status.Health
	networkPortConnectionType := networkPort.FCPortConnectionType
	networkPhysicalPortNumber := networkPort.PhysicalPortNumber
	chassisNetworkPortLabelValues := []string{"network_port", chassisID, networkAdapterName, networkAdapterID, networkPortName, networkPortID, string(networkPortLinkType), networkPortLinkSpeed, string(networkPortConnectionType), networkPhysicalPortNumber}

	if networkLinkStatusValue, ok := parsePortLinkStatus(schemas.PortLinkStatus(networkLinkStatus)); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_network_port_link_state"].desc, prometheus.GaugeValue, networkLinkStatusValue, chassisNetworkPortLabelValues...)
	}

	if networkPortStateValue, ok := parseCommonStatusState(networkPortState); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_network_port_state"].desc, prometheus.GaugeValue, networkPortStateValue, chassisNetworkPortLabelValues...)
	}
	if networkPortHealthStateValue, ok := parseCommonStatusHealth(networkPortHealthState); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_network_port_health_state"].desc, prometheus.GaugeValue, networkPortHealthStateValue, chassisNetworkPortLabelValues...)
	}
}

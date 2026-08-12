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
	sensorExclude         *regexp.Regexp
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

// skipSensor reports whether a Sensor Id is filtered out by the configured pattern. An
// unset pattern never filters. Callers must exempt leak detector sensors before consulting
// this; see config.ChassisCollectorConfig.SensorExclude.
func (c *ChassisCollector) skipSensor(sensorID string) bool {
	return c.sensorExclude != nil && c.sensorExclude.MatchString(sensorID)
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
	// leak_detector_state and leak_detector_health carry the same value on every detector we
	// have captured, including three GB300 trays with a tripped detector. That is not a
	// coincidence: all of them are LeakDetector v1_1_0, where DetectorState is a $ref to
	// Resource.Health, so the two share one enum of OK/Warning/Critical. Either metric
	// detects a trip today, which is how leak detection already works on NVL72.
	//
	// They are both emitted because the schema separates them later. From v1_3_0
	// DetectorState only "should equate" to Health, and v1_6_0 narrows that to when the
	// detector is enabled and functional -- a detector fault is where they part company,
	// with Status.Conditions carrying the fault type. Health is the rollup that goes
	// non-OK for a leak *or* a fault; DetectorState is the leak-specific reading.
	//
	// Neither distinction is observable on our hardware yet: Conditions needs Resource
	// v1_11_0 and appears in no captured payload. So a trip and a faulted detector are
	// currently indistinguishable from these two metrics alone -- see leak_detector_volts
	// below, which is the only fault check we have, and only where a companion sensor exists.
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_state", fmt.Sprintf("chassis leak detector state; this is the signal to alert on, and a Critical state is a detector trip that the companion voltage classifies as wet or as contamination,%s", CommonDetectorStateHelp), ChassisLeakDetectorLabelNames)
	// Enabled arrives in LeakDetector v1_3_0, so this is absent on all current hardware.
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_enabled", "whether this chassis leak detector is enabled, 1(enabled),0(disabled); a disabled detector does not trigger events", ChassisLeakDetectorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_info", "chassis leak detector type and physical location, always 1", ChassisLeakDetectorInfoLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detection_health", fmt.Sprintf("health of the chassis leak detection subsystem as a whole,%s", CommonHealthHelp), ChassisLeakDetectionLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detection_state", fmt.Sprintf("state of the chassis leak detection subsystem as a whole,%s", CommonStateHelp), ChassisLeakDetectionLabelNames)
	// Analog side of the leak detectors, read from the companion Sensor resources. These
	// are resistive moisture ropes on a voltage divider: dry reads high and water pulls
	// the voltage down, so the alarm is a LOWER critical crossing.
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_volts", "chassis leak detector reading in volts; falls toward the lower critical threshold as moisture is detected", ChassisLeakDetectorLabelNames)
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_volts_lower_threshold_critical", "voltage at or below which this chassis leak detector reports a critical leak", ChassisLeakDetectorLabelNames)
	// The upper threshold is not a wetter-still reading: a rope that is disconnected or
	// shorted to the rail reads high, so crossing it means the detector has stopped being
	// able to see a leak rather than that it has seen one.
	addToMetricMap(chassisMetrics, ChassisSubsystem, "leak_detector_volts_upper_threshold_critical", "voltage at or above which this chassis leak detector is considered faulty, typically an open or shorted sense line rather than a leak", ChassisLeakDetectorLabelNames)

	// Catch-all for Sensors whose ReadingType has no equivalent among the families above,
	// so a reading is never silently dropped. Unambiguous ones fold into those families
	// instead; see parseChassisSensor.
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

// compileFilter compiles an optional filter pattern, naming it in any error so a bad
// pattern points at the configuration key that carries it. An empty pattern yields nil,
// which every filter treats as "does not filter".
func compileFilter(name, pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid %s pattern %q: %w", name, pattern, err)
	}
	return re, nil
}

// NewChassisCollector returns a collector that collecting chassis statistics
func NewChassisCollector(collectorName string, redfishClient *gofish.APIClient, logger *slog.Logger, config config.ChassisCollectorConfig) (*ChassisCollector, error) {
	// get service from redfish client

	// Compile the filters up front so a bad pattern is reported at collector construction
	// rather than silently filtering nothing on every scrape.
	chassisInclude, err := compileFilter("chassis_include", config.ChassisInclude)
	if err != nil {
		return nil, err
	}
	chassisExclude, err := compileFilter("chassis_exclude", config.ChassisExclude)
	if err != nil {
		return nil, err
	}
	sensorExclude, err := compileFilter("sensor_exclude", config.SensorExcludePattern())
	if err != nil {
		return nil, err
	}

	return &ChassisCollector{
		redfishClient:  redfishClient,
		metrics:        chassisMetrics,
		config:         config,
		chassisInclude: chassisInclude,
		chassisExclude: chassisExclude,
		sensorExclude:  sensorExclude,
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

	if ctx.Err() != nil {
		c.logger.With("error", ctx.Err(), "collector", "chassis").Debug("skipping collection")
		return
	}
	// get a list of chassis from service
	chassises, err := c.listChassis(ctx, logger)
	if err != nil {
		// A collection error means some member failed, not that none of them arrived:
		// gofish still returns the chassis it did fetch. Discarding those made one flaky
		// chassis out of forty silently zero the whole chassis scrape, which reads as a
		// host with no chassis rather than as a host with a problem.
		logger.Error("error listing chassis", slog.String("operation", "listChassis()"), slog.Any("error", err))
	}

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

		// The subordinate resources this chassis actually advertises. Newer platforms
		// (for example an NVL72 tray, where no chassis implements Thermal or Power)
		// express the same readings through Sensors instead, which is collected below
		// only when the legacy schemas are absent so the two paths never emit duplicate
		// series.
		links := chassisAdvertisedLinks(chassis)

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

		// bulkSensors distinguishes the two reasons to consult the collection: standing in
		// for absent Thermal/Power, which emits everything, or reaching the leak detectors
		// alone, which emits only those. Emitting everything in the second case would
		// publish a chassis's temperatures twice under one series name, failing the scrape
		// at registration.
		optedOutOfBulk := c.config.DisableThermal && c.config.DisablePower
		bulkSensors := !links.legacyThermalOrPower() && !optedOutOfBulk
		if !c.config.DisableSensors && (bulkSensors || len(leakDetectorIDs) > 0) {
			sensorsPath := links.sensorsPath(chassis)
			sensors, err := c.getChassisSensors(ctx, sensorsPath, leakDetectorIDs, chassisLogger)
			if err != nil {
				chassisLogger.Error("error getting sensors from chassis", slog.String("operation", "chassis.Sensors()"), slog.Any("error", err))
			} else if len(sensors) == 0 {
				chassisLogger.Debug("no sensors found", slog.String("operation", "chassis.Sensors()"))
			} else {
				// Parsing a sensor is a handful of channel sends, so this stays a plain
				// loop; a goroutine per sensor would be several hundred per tray to save
				// nothing.
				for _, sensor := range sensors {
					if sensor == nil {
						continue
					}
					_, isLeakDetector := leakDetectorIDs[sensor.ID]
					if !isLeakDetector && (!bulkSensors || c.skipSensor(sensor.ID)) {
						continue
					}
					parseChassisSensor(ch, chassisID, sensor, leakDetectorIDs)
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

	c.collectorScrapeStatus.WithLabelValues("chassis").Set(float64(1))
}

// Describe implemented prometheus.Collector
func (c *ChassisCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metrics {
		ch <- metric.desc
	}
	c.collectorScrapeStatus.Describe(ch)

}

// listChassis returns the chassis this collector should walk.
//
// With no include/exclude pattern this is gofish's own listing, unchanged and one request
// cheaper than the filtered path.
//
// With a pattern, the member links are read first and only matching members are fetched.
// skipChassis alone filters after every body has been paid for, which on a module such as
// leak_detection is the entire per-scrape cost: forty-two chassis fetched to look at one.
//
// Members are matched on the trailing URI segment, a convention rather than a guarantee, so
// callers still apply skipChassis to the fetched Id.
func (c *ChassisCollector) listChassis(ctx context.Context, logger *slog.Logger) ([]*schemas.Chassis, error) {
	service := c.redfishClient.Service
	if c.chassisInclude == nil && c.chassisExclude == nil {
		return service.Chassis()
	}

	client := c.redfishClient.WithContext(ctx)
	collectionURI, err := c.chassisCollectionURI(client)
	if err != nil {
		logger.Debug("could not read the chassis collection link, falling back to the full listing", slog.Any("error", err))
		return service.Chassis()
	}

	memberURIs, err := collectionMemberURIs(client, collectionURI)
	if err != nil {
		logger.Debug("could not read chassis collection members, falling back to the full listing", slog.Any("error", err))
		return service.Chassis()
	}

	chassises := make([]*schemas.Chassis, 0, len(memberURIs))
	for _, uri := range memberURIs {
		if c.skipChassis(resourceIDFromURI(uri)) {
			continue
		}
		chassis, err := schemas.GetChassis(client.GetService().GetClient(), uri)
		if err != nil {
			logger.Error("error getting chassis", slog.String("chassis", uri), slog.Any("error", err))
			continue
		}
		chassises = append(chassises, chassis)
	}
	return chassises, nil
}

// chassisCollectionURI returns the ChassisCollection URI advertised by the service root.
//
// gofish parses the same link but keeps it unexported, so it is re-read from the service
// root here. Synthesising "/redfish/v1/Chassis" would work on every BMC we have captured,
// but a wrong guess here silently collects nothing at all, which is not a failure mode worth
// trading one request for.
func (c *ChassisCollector) chassisCollectionURI(client *gofish.APIClient) (string, error) {
	response, err := client.Get(c.redfishClient.Service.ODataID)
	if err != nil {
		return "", err
	}
	defer response.Body.Close() //nolint:errcheck

	var root struct {
		Chassis odataLink `json:"Chassis"`
	}
	if err := json.NewDecoder(response.Body).Decode(&root); err != nil {
		return "", err
	}
	if root.Chassis.ODataID == "" {
		return "", fmt.Errorf("service root advertises no Chassis collection")
	}
	return root.Chassis.ODataID, nil
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

// chassisLinks are the subordinate resource URIs advertised by a chassis payload.
//
// gofish parses the same links but keeps them unexported, so they are re-read from the raw
// payload here. Reading the advertised links rather than inferring from a fetch means the
// answers do not change when a subsystem is disabled by configuration, and that a resource
// the chassis never advertises is never requested.
type chassisLinks struct {
	thermal string
	power   string
	sensors string
	// opaque records that the payload could not be inspected at all, which every field
	// being empty would otherwise be indistinguishable from.
	opaque bool
}

// chassisAdvertisedLinks extracts the links this collector decides on from a chassis.
func chassisAdvertisedLinks(chassis *schemas.Chassis) chassisLinks {
	if len(chassis.RawData) == 0 {
		return chassisLinks{opaque: true}
	}
	var raw struct {
		Thermal odataLink `json:"Thermal"`
		Power   odataLink `json:"Power"`
		Sensors odataLink `json:"Sensors"`
	}
	if err := json.Unmarshal(chassis.RawData, &raw); err != nil {
		return chassisLinks{opaque: true}
	}
	return chassisLinks{
		thermal: raw.Thermal.ODataID,
		power:   raw.Power.ODataID,
		sensors: raw.Sensors.ODataID,
	}
}

// odataLink is a Redfish reference object, which is absent, empty, or carries a URI.
type odataLink struct {
	ODataID string `json:"@odata.id"`
}

// legacyThermalOrPower reports whether the chassis links either of the deprecated Thermal
// or Power resources.
//
// An uninspectable payload counts as advertising them, suppressing the Sensors fallback
// rather than risking a wasted request per chassis.
//
// Either link suppresses it, not both: a chassis implementing exactly one and expressing the
// other half only through Sensors would lose that half. None does in any captured payload,
// and the alternative risks duplicate series, which fails the scrape rather than thinning it.
func (l chassisLinks) legacyThermalOrPower() bool {
	return l.opaque || l.thermal != "" || l.power != ""
}

// sensorsPath returns the advertised Sensors collection URI, or "" when this chassis has
// no Sensors resource to request.
//
// Synthesising "<chassis>/Sensors" instead would 404 once per scrape on every chassis that
// has none, which is roughly a third of an NVL72 tray or HGX baseboard (the ERoT/IRoT
// roots). An opaque payload falls back to the conventional path so a leak detector's
// companion sensor stays reachable in the degraded case.
func (l chassisLinks) sensorsPath(chassis *schemas.Chassis) string {
	if l.opaque {
		return chassis.ODataID + "/Sensors"
	}
	return l.sensors
}

// getChassisSensors returns sensors for a chassis in a single request, using $expand so the
// BMC inlines every member body rather than gofish's one request per member.
//
// If the BMC does not honour $expand there is deliberately no fan-out to one request per
// sensor: that would multiply request count against the BMCs least able to absorb it. Only
// the named sensors are fetched individually — the leak detectors, safety relevant and few
// — and the bulk sensor telemetry is skipped with a warning.
func (c *ChassisCollector) getChassisSensors(ctx context.Context, sensorsPath string, required map[string]string, logger *slog.Logger) ([]*schemas.Sensor, error) {
	if sensorsPath == "" {
		return nil, nil
	}
	rfClient := c.redfishClient.WithContext(ctx)

	response, err := rfClient.Get(sensorsPath + `?$expand=.($levels=1)`)
	if err != nil {
		logger.Debug("expanded Sensors request failed", slog.Any("error", err))
		return c.getNamedSensors(ctx, sensorsPath, required, nil, logger), nil
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
	// The link-only members are the collection's own account of what exists, so hand them to
	// the fallback rather than letting it synthesise URIs.
	return c.getNamedSensors(ctx, sensorsPath, required, sensorMemberURIs(agg.Members), logger), nil
}

// sensorMemberURIs returns the URIs of an unexpanded Sensors collection's members.
func sensorMemberURIs(members []*schemas.Sensor) []string {
	uris := make([]string, 0, len(members))
	for _, member := range members {
		if member != nil && member.ODataID != "" {
			uris = append(uris, member.ODataID)
		}
	}
	return uris
}

// getNamedSensors fetches individually named sensors, one request each. Callers must keep
// the set small.
//
// available is the collection's member URIs when known, so only required sensors that
// actually exist are fetched: the MGX NVLink switch tray has seven leak detectors and one
// sensor, so synthesising a URI per detector is seven guaranteed 404s per scrape. A nil
// available means the collection was unreadable, leaving the conventional path.
func (c *ChassisCollector) getNamedSensors(ctx context.Context, sensorsPath string, required map[string]string, available []string, logger *slog.Logger) []*schemas.Sensor {
	if len(required) == 0 {
		return nil
	}

	targets := make([]string, 0, len(required))
	if available == nil {
		for sensorID := range required {
			targets = append(targets, sensorsPath+"/"+sensorID)
		}
	} else {
		for _, uri := range available {
			if _, ok := required[resourceIDFromURI(uri)]; ok {
				targets = append(targets, uri)
			}
		}
	}

	client := c.redfishClient.WithContext(ctx).GetService().GetClient()
	sensors := make([]*schemas.Sensor, 0, len(targets))
	for _, uri := range targets {
		sensor, err := schemas.GetSensor(client, uri)
		if err != nil {
			logger.Debug("could not get named sensor", slog.String("sensor", uri), slog.Any("error", err))
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

// thresholdReading flattens an optional Redfish threshold, reporting whether the firmware
// supplied one at all.
func thresholdReading(t schemas.Threshold) (float64, bool) {
	if t.Reading == nil {
		return 0, false
	}
	return *t.Reading, true
}

// thresholdReadingOrZero is thresholdReading for the fan families, where the Thermal path
// has always emitted an unreported threshold as zero and the existing alerts compare the
// two series directly. A Sensors-only platform must produce the same shape.
func thresholdReadingOrZero(t schemas.Threshold) float64 {
	value, _ := thresholdReading(t)
	return value
}

// parseChassisSensor emits metrics for a single Sensor resource.
//
// An unambiguous ReadingType folds into the existing chassis metric family so that a
// Sensors-only platform produces the same series names as a Thermal/Power one; anything
// else lands in a sensor_* catch-all rather than being guessed at or dropped.
//
// Readings are deliberately not inferred from sensor naming. On an NVL72 tray, fan PWM duty
// cycle and CPU core utilisation are both ReadingType "Percent" and neither carries a
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
		// An unreported threshold is left absent rather than defaulted to zero. These
		// exist to be compared against the reading, and a fabricated 0 would make
		// "reading <= lower" a comparison that can never fire — a blind spot that looks
		// like a healthy alert, where a missing series is visibly missing.
		if lower, ok := thresholdReading(sensor.Thresholds.LowerCritical); ok {
			ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detector_volts_lower_threshold_critical"].desc, prometheus.GaugeValue, lower, labelValues...)
		}
		if upper, ok := thresholdReading(sensor.Thresholds.UpperCritical); ok {
			ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detector_volts_upper_threshold_critical"].desc, prometheus.GaugeValue, upper, labelValues...)
		}
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
		// parseChassisFan derives it — including its quirks: the reading is already a
		// percentage when that is the unit, the upper thresholds stand in for a max the
		// firmware did not report, and the formula is (reading + min) / max rather than
		// the usual (reading - min) / (max - min). Reproducing it exactly is the point,
		// so a platform that moved from Thermal to Sensors does not shift the series.
		labelValues := []string{"fan", chassisID, sensor.Name, sensor.ID, strings.ToLower(sensor.ReadingUnits)}
		rpmMin := gofish.Deref(sensor.ReadingRangeMin)
		rpmMax := gofish.Deref(sensor.ReadingRangeMax)
		// Thermal spells this unit "Percent" and Sensors spells it "%"; accept either, so
		// that a firmware using the Thermal spelling in a Sensor is not scaled twice.
		percentage := reading
		if !strings.EqualFold(sensor.ReadingUnits, "%") && !strings.EqualFold(sensor.ReadingUnits, string(schemas.PercentReadingUnits)) {
			upper := math.Max(thresholdReadingOrZero(sensor.Thresholds.UpperFatal), thresholdReadingOrZero(sensor.Thresholds.UpperCritical))
			max := math.Max(rpmMax, upper)
			percentage = 0
			if max != 0 {
				percentage = ((reading + rpmMin) / max) * 100
			}
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
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_lower_threshold_critical"].desc, prometheus.GaugeValue, thresholdReadingOrZero(sensor.Thresholds.LowerCritical), labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_upper_threshold_critical"].desc, prometheus.GaugeValue, thresholdReadingOrZero(sensor.Thresholds.UpperCritical), labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_lower_threshold_fatal"].desc, prometheus.GaugeValue, thresholdReadingOrZero(sensor.Thresholds.LowerFatal), labelValues...)
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_fan_rpm_upper_threshold_fatal"].desc, prometheus.GaugeValue, thresholdReadingOrZero(sensor.Thresholds.UpperFatal), labelValues...)

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

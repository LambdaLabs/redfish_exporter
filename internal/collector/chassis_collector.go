package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
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
	ChassisSensorLabelNames           = []string{"resource", "chassis_id", "sensor", "sensor_id", "sensor_units", "physical_context"}

	ChassisLogServiceLabelNames = []string{"chassis_id", "log_service", "log_service_id", "log_service_enabled", "log_service_overwrite_policy"}

	chassisMetrics = createChassisMetricMap()
)

// ChassisCollector implements the prometheus.Collector.
type ChassisCollector struct {
	redfishClient         *gofish.APIClient
	config                config.ChassisCollectorConfig
	metrics               map[string]Metric
	logger                *slog.Logger
	collectorScrapeStatus *prometheus.GaugeVec
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

// NewChassisCollector returns a collector that collecting chassis statistics
func NewChassisCollector(collectorName string, redfishClient *gofish.APIClient, logger *slog.Logger, config config.ChassisCollectorConfig) (*ChassisCollector, error) {
	// get service from redfish client

	return &ChassisCollector{
		redfishClient: redfishClient,
		metrics:       chassisMetrics,
		config:        config,
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
	chassises, err := service.Chassis()
	if err != nil {
		// A collection error means some member failed, not that none of them arrived:
		// gofish still returns the chassis it did fetch. Discarding those made one flaky
		// chassis out of forty silently zero the whole chassis scrape, which reads as a
		// host with no chassis rather than as a host with a problem.
		logger.Error("error getting chassis from service", slog.String("operation", "service.Chassis()"), slog.Any("error", err))
	}

	// process the chassises
	for _, chassis := range chassises {
		if ctx.Err() != nil {
			c.logger.With("error", ctx.Err()).Warn("skipping further collection")
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

		chassisThermal, err := chassis.Thermal()
		if err != nil {
			chassisLogger.Error("error getting thermal data from chassis", slog.String("operation", "chassis.Thermal()"), slog.Any("error", err))
		} else if chassisThermal == nil {
			chassisLogger.Info("no thermal data found", slog.String("operation", "chassis.Thermal()"))
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
		// leakDetectorIDs is the Id of every leak detector on this chassis. Each detector is
		// dual-surfaced: the LeakDetector resource carries the discrete state, while a
		// Sensor of the same Id carries an analog voltage. The Sensors pass below uses this
		// to leave those readings alone rather than publishing them as ordinary chassis
		// voltages, which is what they would otherwise be mistaken for.
		leakDetectorIDs := map[string]struct{}{}

		chassisThermalSubsystem, err := chassis.ThermalSubsystem()
		if err != nil {
			chassisLogger.Error("error getting thermal subsystem from chassis", slog.String("operation", "chassis.ThermalSubsystem()"), slog.Any("error", err))
		} else if chassisThermalSubsystem == nil {
			chassisLogger.Info("no thermal subsystem found", slog.String("operation", "chassis.ThermalSubsystem()"))
		} else {
			// NOTE: Handles some odd (maybe even buggy) OEM implementations of LeakDeteactor
			leakDetectors := c.getLeakDetectors(chassisThermalSubsystem, chassisLogger)

			for _, ld := range leakDetectors {
				leakDetectorIDs[ld.ID] = struct{}{}
			}

			if len(leakDetectors) > 0 {
				egLD := newRecoverGroup(ctx)
				for _, ld := range leakDetectors {
					egLD.Go(func() error {
						parseLeakDetector(ch, chassisID, ld)
						return nil
					})
				}
				if err := egLD.Wait(); err != nil {
					chassisLogger.Error("goroutine error", slog.Any("error", err))
				}
			} else {
				chassisLogger.Info("no leak detectors found")
			}
		}

		chassisPowerInfo, err := chassis.Power()
		if err != nil {
			chassisLogger.Error("error getting power data from chassis", slog.String("operation", "chassis.Power()"), slog.Any("error", err))
		} else if chassisPowerInfo == nil {
			chassisLogger.Info("no power data found", slog.String("operation", "chassis.Power()"))
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

		// The Sensors collection stands in for Thermal and Power on platforms that
		// implement neither, so it is consulted only when both are absent. Collecting it
		// alongside them would publish a chassis's temperatures twice under one series
		// name, failing the scrape at registration.
		if !links.legacyThermalOrPower() {
			sensorsPath := links.sensorsPath(chassis)
			// getChassisSensors warns about anything it could not collect, so an empty
			// answer here only means this chassis published none.
			sensors := c.getChassisSensors(ctx, sensorsPath, chassisLogger)
			if len(sensors) == 0 {
				chassisLogger.Info("no sensors found", slog.String("operation", "GET "+sensorsPath))
			} else {
				// Parsing a sensor is a handful of channel sends, so this stays a plain
				// loop; a goroutine per sensor would be several hundred per tray to save
				// nothing. A panic in one is still contained, by the recover group the
				// whole chassis collection runs under.
				for _, sensor := range sensors {
					// A leak detector's companion sensor is a leak signal, not a chassis
					// voltage, and publishing it as one would misreport it. It is left to
					// the leak detection path, which is the only thing able to interpret
					// it against its thresholds.
					if _, isLeakDetector := leakDetectorIDs[sensor.ID]; isLeakDetector {
						continue
					}
					parseChassisSensor(ch, chassisID, sensor)
				}
			}
		}

		// process NetworkAdapter
		networkAdapters, err := chassis.NetworkAdapters()
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

// sensorCollection is the shape of an expanded Sensors collection response. Members are
// full Sensor bodies rather than links, so one request replaces one-per-sensor.
type sensorCollection struct {
	Members []*schemas.Sensor `json:"Members"`
	// NextLink is set when the BMC paginated the collection, which this collector reports
	// rather than follows.
	NextLink string `json:"Members@odata.nextLink"`
}

// chassisLinks are the subordinate resource URIs advertised by a chassis payload.
//
// gofish parses the same links but keeps them unexported, and its Chassis.Sensors() fans out
// one request per member, so the URIs are re-read from the raw payload here. Reading what the
// chassis advertises rather than inferring it from a fetch means a resource the chassis never
// advertises is never requested.
type chassisLinks struct {
	thermal string
	power   string
	sensors string
	// opaque records that the payload could not be inspected at all, which every field
	// being empty would otherwise be indistinguishable from.
	opaque bool
}

// chassisAdvertisedLinks extracts the links this collector decides on from a chassis.
//
// The references are read through schemas.Link, the same type gofish parses them with, so
// the two cannot disagree about whether a chassis advertises Thermal. A reference object
// may spell its URI "href" rather than "@odata.id", and reading only the latter would leave
// gofish collecting a chassis's Thermal while this concluded it had none — publishing that
// chassis's temperatures from both paths at once, which fails the scrape at registration.
func chassisAdvertisedLinks(chassis *schemas.Chassis) chassisLinks {
	if len(chassis.RawData) == 0 {
		return chassisLinks{opaque: true}
	}
	var raw struct {
		Thermal schemas.Link `json:"Thermal"`
		Power   schemas.Link `json:"Power"`
		Sensors schemas.Link `json:"Sensors"`
	}
	if err := json.Unmarshal(chassis.RawData, &raw); err != nil {
		return chassisLinks{opaque: true}
	}
	return chassisLinks{
		thermal: raw.Thermal.String(),
		power:   raw.Power.String(),
		sensors: raw.Sensors.String(),
	}
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
// roots). An opaque payload falls back to the conventional path.
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
// sensor: that would multiply request count against the BMCs least able to absorb it. The
// bulk sensor telemetry is skipped instead.
//
// Nothing here fails the chassis: every way this can go wrong is the BMC answering for its
// own Sensors collection badly, and the answer is always the same — collect what arrived and
// warn about what did not, naming the collection. The caller has no decision left to make,
// which is why no error is returned.
func (c *ChassisCollector) getChassisSensors(ctx context.Context, sensorsPath string, logger *slog.Logger) []*schemas.Sensor {
	if sensorsPath == "" {
		return nil
	}
	rfClient := c.redfishClient.WithContext(ctx)

	// The path came from the chassis's own advertisement, so a failure here is the BMC
	// contradicting itself.
	response, err := rfClient.Get(sensorsPath + `?$expand=.($levels=1)`)
	if err != nil {
		logger.Warn("expanded Sensors request failed; skipping sensor collection",
			slog.String("sensors", sensorsPath),
			slog.Any("error", err),
		)
		return nil
	}
	defer response.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Warn("could not read the Sensors collection; skipping sensor collection",
			slog.String("sensors", sensorsPath),
			slog.Any("error", err),
		)
		return nil
	}

	var agg sensorCollection
	if err := json.Unmarshal(body, &agg); err != nil {
		logger.Warn("could not parse the Sensors collection; skipping sensor collection",
			slog.String("sensors", sensorsPath),
			slog.Any("error", err),
		)
		return nil
	}
	if agg.NextLink != "" {
		// Following the pages would be a request apiece, which is the fan-out this collector
		// exists to avoid, so the tail is dropped — but never silently.
		logger.Warn("Sensors collection is paginated; collecting the first page only",
			slog.String("sensors", sensorsPath),
			slog.String("next", agg.NextLink),
			slog.Int("collected", len(agg.Members)),
		)
	}

	// A BMC that ignores $expand answers with link-only members, which unmarshal into Sensors
	// carrying no Id. A partially expanded answer is possible too, so every member is checked
	// rather than only the first.
	expanded := make([]*schemas.Sensor, 0, len(agg.Members))
	for _, member := range agg.Members {
		if member != nil && member.ID != "" {
			expanded = append(expanded, member)
		}
	}
	if skipped := len(agg.Members) - len(expanded); skipped > 0 {
		logger.Warn("BMC did not honour $expand on Sensors; skipping the unexpanded members",
			slog.String("sensors", sensorsPath),
			slog.Int("skipped", skipped),
			slog.Int("collected", len(expanded)),
		)
	}
	return expanded
}

// getLeakDetectors works around an unfortunate fact that the LeakDetection schema is not yet standard, and some OEMs return
// a single LeakDetection object from their ThermalSubsystem, instead of a gofish-expected collection.
func (c *ChassisCollector) getLeakDetectors(thermalSubsystem *schemas.ThermalSubsystem, logger *slog.Logger) []*schemas.LeakDetector {
	var allDetectors []*schemas.LeakDetector

	// Standard gofish approach, for starters
	leakDetectionCollection, err := thermalSubsystem.LeakDetection()
	if err != nil {
		logger.Debug("standard LeakDetection() call failed, will try fallback", slog.Any("error", err))
		if leakDetectionCollection != nil {
			detectors, err := leakDetectionCollection.LeakDetectors()
			if err != nil {
				logger.Error("failed obtaining LeakDetectors at all", slog.Any("error", err))
				return nil
			}
			allDetectors = append(allDetectors, detectors...)
		}
		return allDetectors
	}

	// ...otherwise, try a fallback to handle buggy OEM implementations.
	leakDetectionURL := thermalSubsystem.ODataID + "/LeakDetection"
	leakDetection, err := schemas.GetLeakDetection(c.redfishClient.Service.GetClient(), leakDetectionURL)
	if err != nil {
		logger.Debug("fallback GetLeakDetection failed", slog.Any("error", err))
		return allDetectors
	}

	if leakDetection != nil {
		detectors, err := leakDetection.LeakDetectors()
		if err != nil {
			logger.Debug("error fetching leak detectors via fallback method", slog.Any("error", err))
			return nil
		}

		if len(detectors) > 0 {
			allDetectors = append(allDetectors, detectors...)
		}
	}
	return allDetectors
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

// fanUnitLabel normalises a fan's reading unit into the fan_unit label, shared by the Thermal
// and the Sensors path.
//
// The two schemas spell the same unit differently — Thermal's enum gives "RPM" and "Percent",
// a Sensor gives "%" for the same duty cycle — and fan_unit is part of the fan series
// identity, so leaving each path to lowercase its own spelling would split one fan's series
// in two across the platforms that report it either way.
func fanUnitLabel(units string) string {
	switch strings.ToLower(units) {
	case "%", "percent", "percentage":
		return "percent"
	case "rpm", "{rev}/min":
		// Redfish prefers "{rev}/min" for Rotational and deprecates "RPM"; no captured
		// platform uses it yet, but it must not arrive as a second fan_unit spelling.
		return "rpm"
	}
	return strings.ToLower(units)
}

// fanPercentage derives a fan's duty cycle from a tachometer reading, shared by the Thermal
// and the Sensors path so the two cannot drift apart: a platform moving between them must
// not shift the series under the existing alerts.
//
// alreadyPercent covers a fan reporting a duty cycle directly, which needs no derivation.
//
// Otherwise the quirks are deliberate. Some vendors (e.g. PowerEdge C6420) report null
// Min/Max and Lower/UpperFatal but do provide Lower/UpperCritical, so the largest non-null
// upper bound stands in for a max the firmware did not report. A null min is indistinguishable
// from a real zero once gofish has flattened it, so it is taken at face value — which is why
// the formula is (reading + min) / max rather than the usual (reading - min) / (max - min).
func fanPercentage(reading, rangeMin, rangeMax, upperFatal, upperCritical float64, alreadyPercent bool) float64 {
	if alreadyPercent {
		return reading
	}
	ceiling := math.Max(math.Max(rangeMax, upperFatal), upperCritical)
	if ceiling == 0 {
		return 0
	}
	return ((reading + rangeMin) / ceiling) * 100
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

	chassisFanPercentage := fanPercentage(
		chassisFanRPM,
		chassisFanRPMMin,
		chassisFanRPMMax,
		chassisFanRPMUpperFatalThreshold,
		chassisFanRPMUpperCriticalThreshold,
		chassisFanUnit == schemas.PercentReadingUnits,
	)

	//			chassisFanStatusLabelNames :=[]string{BaseLabelNames,"fan_name","fan_member_id")
	chassisFanLabelvalues := []string{"fan", chassisID, chassisFanName, chassisFanID, fanUnitLabel(string(chassisFanUnit))} // e.g. RPM -> rpm, Percent -> percent

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

func parseLeakDetector(ch chan<- prometheus.Metric, chassisID string, ld *schemas.LeakDetector) {
	ldID := ld.ID
	labelValues := []string{"leak_detector", chassisID, "LeakDetection", ldID}

	if statusHealth, ok := parseCommonStatusHealth(ld.Status.Health); ok {
		ch <- prometheus.MustNewConstMetric(chassisMetrics["chassis_leak_detector_health"].desc, prometheus.GaugeValue, statusHealth, labelValues...)
	}
}

// thresholdReadingOrZero flattens an optional Redfish threshold, reporting a threshold the
// firmware did not supply as zero rather than omitting the series: the Thermal path has
// always emitted it that way and the existing alerts compare the two series directly, so a
// Sensors-only platform has to produce the same shape.
func thresholdReadingOrZero(t schemas.Threshold) float64 {
	if t.Reading == nil {
		return 0
	}
	return *t.Reading
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
func parseChassisSensor(ch chan<- prometheus.Metric, chassisID string, sensor *schemas.Sensor) {
	if sensor == nil || sensor.ID == "" {
		return
	}
	reading := gofish.Deref(sensor.Reading)

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
		// A rotational reading is a fan tachometer, so it goes through the same derivation
		// the Thermal path uses, quirks and all — see fanPercentage.
		labelValues := []string{"fan", chassisID, sensor.Name, sensor.ID, fanUnitLabel(sensor.ReadingUnits)}
		rpmMin := gofish.Deref(sensor.ReadingRangeMin)
		rpmMax := gofish.Deref(sensor.ReadingRangeMax)
		// Thermal spells this unit "Percent" and Sensors spells it "%"; fanUnitLabel accepts
		// either, so a firmware using the Thermal spelling in a Sensor is not scaled twice.
		alreadyPercent := fanUnitLabel(sensor.ReadingUnits) == "percent"
		percentage := fanPercentage(
			reading,
			rpmMin,
			rpmMax,
			thresholdReadingOrZero(sensor.Thresholds.UpperFatal),
			thresholdReadingOrZero(sensor.Thresholds.UpperCritical),
			alreadyPercent,
		)
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

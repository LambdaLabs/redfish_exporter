package collector

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/schemas"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
)

var (
	powershelfMetrics      = createPowershelfMetricsMap()
	PowershelfPSULabels    = []string{"power_supply_id"}
	PowershelfSensorLabels = []string{"sensor_id"}
	PowershelfSubsystem    = "powershelf"
)

// PowershelfCollector implements the prometheus.Collector.
type PowershelfCollector struct {
	redfishClient         *gofish.APIClient
	config                config.PowershelfCollectorConfig
	metrics               map[string]Metric
	logger                *slog.Logger
	collectorScrapeStatus *prometheus.GaugeVec
	adapters              []vendorAdapter
}

type Sample struct {
	Name        string // metric <name>, e.g. "input_voltage"
	PowerSupply string // "ps1".."ps6"; "" for shelf-level
	Value       float64
	SensorID    string // set only for catch-all metrics; carries the raw sensor id as a label
}

type vendorAdapter interface {
	name() string
	// collect returns this vendor's samples. matched=false means "not my device, skip me"
	collect(ctx context.Context, client *gofish.APIClient) (samples []Sample, matched bool, err error)
}

func createPowershelfMetricsMap() map[string]Metric {
	powershelfMetrics := make(map[string]Metric)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "input_voltage", "PSU AC input voltage, volts", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "psu_health", fmt.Sprintf("PSU health,%s", CommonHealthHelp), PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "psu_state", fmt.Sprintf("PSU state,%s", CommonStateHelp), PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "output_voltage", "PSU DC output (rail) voltage, volts", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "input_current", "PSU input current, amperes", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "output_current", "PSU output current, amperes", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "input_power", "PSU input power, watts", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "output_power", "PSU output power, watts", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "standby_output_voltage", "PSU standby rail voltage, volts", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "standby_output_current", "PSU standby rail current, amperes", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "standby_output_power", "PSU standby rail power, watts", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "powerfactor", "PSU power factor, percent (0-100)", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "energy_in", "PSU input energy (per-interval gauge)", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "energy_out", "PSU output energy (per-interval gauge)", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "total_energy_in", "PSU accumulated input energy (counter)", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "fan1", "PSU fan 1 speed, RPM", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "fan2", "PSU fan 2 speed, RPM", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "temp_input", "PSU intake (ambient) temperature, celsius", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "temp_output", "PSU exhaust temperature, celsius", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "temp_hotspot", "PSU hotspot temperature, celsius", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "temp_clip_plus", "PSU DC clip+ temperature, celsius", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "temp_clip_minus", "PSU DC clip- temperature, celsius", PowershelfPSULabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "input_frequency", "PSU AC input frequency, hertz", PowershelfPSULabels)

	// per-ReadingType catch-all so no sensor is dropped; labeled by raw sensor id
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "sensor_volts", "uncurated sensor reading, volts", PowershelfSensorLabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "sensor_amperes", "uncurated sensor reading, amperes", PowershelfSensorLabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "sensor_watts", "uncurated sensor reading, watts", PowershelfSensorLabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "sensor_celsius", "uncurated sensor reading, celsius", PowershelfSensorLabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "sensor_percent", "uncurated sensor reading, percent", PowershelfSensorLabels)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "sensor_rpm", "uncurated sensor reading, rpm", PowershelfSensorLabels)

	// shelf-level metrics (have no power_supply_id, so pass nil label)
	// we expect these to be common across all powershelves
	// totals / aggregate output
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "total_power_in", "shelf total input power, watts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "total_power_in_A", "shelf total input power on phase A, watts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "total_power_out", "shelf total output power, watts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "total_current_out", "shelf total output current, amperes", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "average_current_out", "shelf average per-PSU output current, amperes", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "max_voltage_out", "shelf maximum output (rail) voltage, volts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "power_load", "shelf load, percent (0-100)", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "total_efficiency", "shelf conversion efficiency, percent (0-100)", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "current_share", "shelf load-share current, amperes", nil)

	// 3-phase AC input (Mains)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "voltage_in_A_A", "shelf AC input voltage, phase A, volts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "voltage_in_A_B", "shelf AC input voltage, phase B, volts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "voltage_in_A_C", "shelf AC input voltage, phase C, volts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "current_in_A_A", "shelf AC input current, phase A, amperes", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "current_in_A_B", "shelf AC input current, phase B, amperes", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "current_in_A_C", "shelf AC input current, phase C, amperes", nil)

	// BMC / controller rails
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "bmc_12v", "shelf BMC 12V rail, volts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "bmc_3v3", "shelf BMC 3.3V rail, volts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "bmc_temp", "shelf BMC temperature, celsius", nil)

	// hotswap / ORing controller
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "hotswap_input_current", "shelf hotswap input current, amperes", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "hotswap_input_power", "shelf hotswap input power, watts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "hotswap_input_voltage", "shelf hotswap input voltage, volts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "hotswap_output_voltage", "shelf hotswap output voltage, volts", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "hotswap_temp", "shelf hotswap temperature, celsius", nil)

	// Misc shelf-level metrics
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "total_current_in", "shelf total input current, amperes", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "temp_shelf", "shelf ambient temperature, celsius", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "dc_temp_plus", "shelf DC bus (+) temperature, celsius", nil)
	addToMetricMap(powershelfMetrics, PowershelfSubsystem, "dc_temp_minus", "shelf DC bus (-) temperature, celsius", nil)

	return powershelfMetrics
}

// NewPowershelfCollector returns a collector that collecting chassis statistics
func NewPowershelfCollector(collectorName string, redfishClient *gofish.APIClient, logger *slog.Logger, config config.PowershelfCollectorConfig) (*PowershelfCollector, error) {
	// get service from redfish client

	return &PowershelfCollector{
		redfishClient: redfishClient,
		metrics:       powershelfMetrics,
		config:        config,
		logger:        logger,
		adapters:      []vendorAdapter{&liteonAdapter{}, &deltaAdapter{}},
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

func (c *PowershelfCollector) CollectWithContext(ctx context.Context, ch chan<- prometheus.Metric) {
	c.collect(ctx, ch)
}

// Collect implemented prometheus.Collector
func (c *PowershelfCollector) Collect(ch chan<- prometheus.Metric) {
	c.collect(context.TODO(), ch)
}

func (c *PowershelfCollector) collect(ctx context.Context, ch chan<- prometheus.Metric) {
	for _, a := range c.adapters {
		samples, matched, err := a.collect(ctx, c.redfishClient)
		if err != nil {
			c.logger.Error("powershelf adapter error", slog.String("adapter", a.name()), slog.Any("error", err))
			continue
		}
		if !matched {
			continue
		}
		for _, s := range samples {
			metric, ok := c.metrics["powershelf_"+s.Name]
			if !ok {
				c.logger.Warn("no descriptor for powershelf sample", slog.String("name", s.Name))
				continue
			}
			vt := prometheus.GaugeValue
			if s.Name == "total_energy_in" {
				vt = prometheus.CounterValue
			}
			switch {
			case s.SensorID != "":
				ch <- prometheus.MustNewConstMetric(metric.desc, vt, s.Value, s.SensorID)
			case s.PowerSupply != "":
				ch <- prometheus.MustNewConstMetric(metric.desc, vt, s.Value, s.PowerSupply)
			default:
				ch <- prometheus.MustNewConstMetric(metric.desc, vt, s.Value)
			}
		}
		c.collectorScrapeStatus.WithLabelValues("powershelf").Set(1)
		c.collectorScrapeStatus.Collect(ch)
		return
	}
	// nothing recognized the device
	c.collectorScrapeStatus.WithLabelValues("powershelf").Set(0)
	c.collectorScrapeStatus.Collect(ch)
}

// Describe implemented prometheus.Collector
func (c *PowershelfCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metrics {
		ch <- metric.desc
	}
	c.collectorScrapeStatus.Describe(ch)

}

// findPowershelfChassis returns the first Chassis whose Manufacturer contains vendorMatch
// (compared upper-cased) and has a non-empty Sensors collection — the data-bearing
// powershelf chassis for that vendor. Returns (nil, nil) if none match.
func findPowershelfChassis(client *gofish.APIClient, vendorMatch string) (*schemas.Chassis, error) {
	chassises, err := client.Service.Chassis()
	if err != nil {
		return nil, err
	}
	for _, ch := range chassises {
		if !strings.Contains(strings.ToUpper(ch.Manufacturer), vendorMatch) {
			continue
		}
		sensors, err := ch.Sensors()
		if err != nil || len(sensors) == 0 {
			continue
		}
		return ch, nil
	}
	return nil, nil
}

// collectPSUStatus emits psu_health/psu_state from the chassis PowerSubsystem.
// psid maps a PowerSupply Id to the canonical power_supply_id ("" => skip that PSU).
func collectPSUStatus(shelf *schemas.Chassis, psid func(string) string) ([]Sample, error) {
	psub, err := shelf.PowerSubsystem()
	if err != nil || psub == nil {
		return nil, err
	}
	supplies, err := psub.PowerSupplies()
	if err != nil {
		return nil, err
	}
	var out []Sample
	for _, psu := range supplies {
		id := psid(psu.ID)
		if id == "" {
			continue
		}
		if v, ok := parseCommonStatusHealth(psu.Status.Health); ok {
			out = append(out, Sample{Name: "psu_health", PowerSupply: id, Value: v})
		}
		if v, ok := parseCommonStatusState(psu.Status.State); ok {
			out = append(out, Sample{Name: "psu_state", PowerSupply: id, Value: v})
		}
	}
	return out, nil
}

// powershelfCatchall maps a Redfish ReadingType to a generic catch-all metric, so any
// sensor we don't curate is emitted (labeled by raw sensor id) rather than dropped.
var powershelfCatchall = map[string]string{
	"Voltage": "sensor_volts", "Current": "sensor_amperes", "Power": "sensor_watts",
	"Temperature": "sensor_celsius", "Percent": "sensor_percent", "Rotational": "sensor_rpm",
}

func catchallSample(id, readingType string, val float64) (Sample, bool) {
	name, ok := powershelfCatchall[readingType]
	if !ok {
		return Sample{}, false
	}
	return Sample{Name: name, SensorID: id, Value: val}, true
}

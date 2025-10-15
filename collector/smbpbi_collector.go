package collector

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
)

const (
	SMBPBISubsystem = "smbpbi"
	NumGPUs         = 8
)

var _ prometheus.Collector = &smbpbiCollector{}

type smbpbiCollector struct {
	client  *gofish.APIClient
	metrics MetricMap
	logger  *slog.Logger
}

var (
	OpcodeNoop                     = 0x00
	OpcodeGetCapabilities          = 0x01
	OpcodeGetTemperature           = 0x02
	OpcodeGetTemperatureExtended   = 0x03
	OpcodeGetPower                 = 0x04
	OpcodeGPUPerformanceMonitoring = 0x28
)

// Opcode: 0x28 - GPU Performance Monitoring
var (
	GPUPerformanceMonitoringMetricGraphicsEngine = 0x00
	GPUPerformanceMonitoringSMActivity           = 0x01
	GPUPerformanceMonitoringSMOccupancy          = 0x02
	GPUPerformanceMonitoringTensorCoreActivity   = 0x03
)

type status int

var (
	StatusRegisterNULL           status = 0x00
	StatusRegisterErrRequest     status = 0x01
	StatusRegisterErrOpcode      status = 0x02
	StatusRegisterErrArg1        status = 0x03
	StatusRegisterErrArg2        status = 0x04
	StatusRegisterData           status = 0x05
	StatusRegisterMisc           status = 0x06
	StatusRegisterI2CAccess      status = 0x07
	StatusRegisterNotSupported   status = 0x08
	StatusRegisterNotAvailable   status = 0x09
	StatusRegisterBusy           status = 0x0A
	StatusRegisterAgain          status = 0x0B
	StatusRegisterSensorData     status = 0x0C
	StatusRegisterDisposition    status = 0x0D
	StatusRegisterPartialFailure status = 0x1B
	StatusRegisterAccepted       status = 0x1C
	StatusRegisterInactive       status = 0x1D
	StatusRegisterReady          status = 0x1E
	StatusRegisterSuccess        status = 0x1F
)
var statusRegisterValues = map[status]string{
	StatusRegisterNULL:           "Invalid status",
	StatusRegisterErrRequest:     "Error in request",
	StatusRegisterErrOpcode:      "Error in opcode",
	StatusRegisterErrArg1:        "Error in arg1",
	StatusRegisterErrArg2:        "Error in arg2",
	StatusRegisterData:           "Data error",
	StatusRegisterMisc:           "Misc error",
	StatusRegisterI2CAccess:      "I2C access error",
	StatusRegisterNotSupported:   "Not supported",
	StatusRegisterNotAvailable:   "Not available",
	StatusRegisterBusy:           "Busy",
	StatusRegisterAgain:          "Try again",
	StatusRegisterSensorData:     "Sensor data error",
	StatusRegisterDisposition:    "Disposition error",
	StatusRegisterPartialFailure: "Partial failure",
	StatusRegisterAccepted:       "Accepted",
	StatusRegisterInactive:       "Inactive",
	StatusRegisterReady:          "Ready",
	StatusRegisterSuccess:        "Success",
}

func (s status) String() string {
	if str, ok := statusRegisterValues[s]; ok {
		return str
	}
	return "Unknown status"
}

func (s status) IsError() bool {
	return int(s) >= 0x00 && int(s) <= 0x1B
}

type StatusRegister []string

// IsError checks if any of the status register values indicate an error.
// If an error is found, it returns true along with the corresponding status and nil error.
// If no errors are found, it returns false, 0, and nil error.
func (s StatusRegister) IsError() (bool, status, error) {
	if len(s) < 4 {
		return false, 0, fmt.Errorf("status register must have at least 4 elements, got %d", len(s))
	}
	statusValue := s[3]
	asInt := hexToInt(statusValue)
	status := status(asInt)
	if status.IsError() {
		return true, status, nil
	}
	return false, 0, nil
}

type NvidiaManagerResponse struct {
	OdataType      string         `json:"@odata.type"`
	DataOut        []string       `json:"DataOut"`
	ExtDataOut     []string       `json:"ExtDataOut"`
	StatusRegister StatusRegister `json:"StatusRegister"`
}

func smbpbiPostBody(gpuIDX int, opcode int, arg1 int, arg2 int) (map[string]any, error) {
	m := map[string]any{
		"TargetType":       "GPU",
		"TargetInstanceId": gpuIDX,
		"Opcode":           fmt.Sprintf("%02X", opcode),
		"Arg1":             fmt.Sprintf("%02X", arg1),
		"Arg2":             fmt.Sprintf("%02X", arg2),
	}
	return m, nil
}

// NewSmbpbiCollector returns a new instance of smbpbiCollector
func NewSMBPBICollector(client *gofish.APIClient, logger *slog.Logger) *smbpbiCollector {
	return &smbpbiCollector{
		client:  client,
		metrics: createSMBPBIMetricMap(),
		logger:  logger,
	}
}

type MetricMap map[string]Metric

func (m MetricMap) GetMetric(name string) (Metric, error) {
	metric, ok := m[fmt.Sprintf("%s_%s", SMBPBISubsystem, name)]
	if !ok {
		return Metric{}, fmt.Errorf("metric %s not found", name)
	}
	return metric, nil
}

func createSMBPBIMetricMap() MetricMap {
	smbpbiMetrics := make(map[string]Metric)
	addToMetricMap(smbpbiMetrics, SMBPBISubsystem, "temperature", "SMBPBI temperature in Celsius", []string{"gpu_idx"})
	addToMetricMap(smbpbiMetrics, SMBPBISubsystem, "sm_activity", "Streaming Multiprocessor activity active cycles / total cycles as a percentage.", []string{"gpu_idx"})
	return smbpbiMetrics
}

func hexToInt(s string) int {
	i, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		panic(err)
	}
	return int(i)
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector.
func (c *smbpbiCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metrics {
		ch <- metric.desc
	}
}

func (c *smbpbiCollector) sendSMBPBICommand(jsonBody map[string]any) (*NvidiaManagerResponse, error) {
	resp, err := c.client.Post("/redfish/v1/Oem/Supermicro/HGX_H100/Managers/HGX_BMC_0/Actions/Oem/NvidiaManager.SyncOOBRawCommand", jsonBody)
	if err != nil {
		return nil, fmt.Errorf("failed to send smbpbi post request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("smbpbi post request returned non-200 status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var response NvidiaManagerResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode smbpbi post response: %w", err)
	}
	isError, status, err := response.StatusRegister.IsError()
	if err != nil {
		return nil, fmt.Errorf("failed to check smbpbi status register for errors: %w", err)
	}
	if isError {
		return nil, fmt.Errorf("smbpbi status register indicates an error: %s", status.String())
	}
	return &response, nil
}

func getCelciusFromFixedPointSignedInteger(dataOut []string) (int, error) {
	if len(dataOut) < 4 {
		return 0, fmt.Errorf("dataOut must have at least 4 elements, got %d", len(dataOut))
	}

	// From docs:
	// Fixed-point signed integer with 8 fractional bits (7:0 all zero) representing
	// the source temperature in Celsius. The temperature is stored in 31:8 and
	// must be right-shifted by 8 bits by the initiator to get the correct
	// representation.

	// This is a fixed-point signed integer with 8 fractional bits (7:0)
	tempUnshifted := (hexToInt(dataOut[0]) << 24) |
		(hexToInt(dataOut[1]) << 16) |
		(hexToInt(dataOut[2]) << 8) |
		(hexToInt(dataOut[3]) << 0)

	// Discard the fractional bits by right-shifting 8 bits
	temp := tempUnshifted >> 8
	// We need to again shift right by 8 bits according to the docs.
	temp = temp >> 8

	return temp, nil
}

func (c *smbpbiCollector) collectSMBPBITemperature(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	defer wg.Wait()

	for i := 1; i <= NumGPUs; i++ {
		jsonBody, err := smbpbiPostBody(i, OpcodeGetTemperature, 0x00, 0x00)
		if err != nil {
			c.logger.Error("failed to create smbpbi post body", slog.Any("error", err))
			continue
		}

		wg.Add(1)
		go func(gpuIDX int, jsonBody map[string]any) {
			defer wg.Done()
			response, err := c.sendSMBPBICommand(jsonBody)
			if err != nil {
				c.logger.Error("failed to send smbpbi command", slog.Any("error", err), slog.Int("gpu_idx", gpuIDX))
				return
			}
			c.logger.Info(fmt.Sprintf("dataout: %v", response.DataOut))
			tempCelcius, err := getCelciusFromFixedPointSignedInteger(response.DataOut)
			if err != nil {
				c.logger.Error("failed to parse temperature from smbpbi response", slog.Any("error", err), slog.Int("gpu_idx", gpuIDX))
				return
			}

			metric, err := c.metrics.GetMetric("temperature")
			if err != nil {
				c.logger.Error("failed to get temperature metric", slog.Any("error", err))
				return
			}
			c.logger.Info(fmt.Sprintf("temperature as int: %d, temperature as float: %f", tempCelcius, float64(tempCelcius)))
			ch <- prometheus.MustNewConstMetric(
				metric.desc,
				prometheus.GaugeValue,
				float64(tempCelcius),
				fmt.Sprintf("%d", gpuIDX),
			)
		}(i, jsonBody)
	}
}

func getFloatFromFixedPointSignedInteger(dataOut []string, fractionalBits uint) float64 {
	fixedPointInt := (hexToInt(dataOut[0]) << 24) |
		(hexToInt(dataOut[1]) << 16) |
		(hexToInt(dataOut[2]) << 8) |
		(hexToInt(dataOut[3]) << 0)
	scalingFactor := 1 << fractionalBits
	return float64(fixedPointInt) / float64(scalingFactor)
}

func (c *smbpbiCollector) collectSMBStreamingMultiprocessorActivity(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	defer wg.Wait()

	// arg1: Bits 7:6 - 0x00 (Return instantaneous metric value)
	//       Bits 5:0 - Metric type
	arg1 := (0x00 << 6) | (GPUPerformanceMonitoringSMActivity)
	// arg2: Select device-level or partition-level
	//		 0xff: Device-level
	arg2 := 0xff

	for i := 1; i <= NumGPUs; i++ {
		jsonBody, err := smbpbiPostBody(i, OpcodeGPUPerformanceMonitoring, arg1, arg2)
		if err != nil {
			c.logger.Error("failed to create smbpbi post body", slog.Any("error", err))
			continue
		}
		wg.Add(1)
		go func(gpuIDX int, jsonBody map[string]any) {
			defer wg.Done()
			response, err := c.sendSMBPBICommand(jsonBody)
			if err != nil {
				c.logger.Error("failed to send smbpbi command", slog.Any("error", err), slog.Int("gpu_idx", gpuIDX))
				return
			}
			c.logger.Info(fmt.Sprintf("dataout: %v", response.DataOut))
			percentage := getFloatFromFixedPointSignedInteger(response.DataOut, 8)

			metric, err := c.metrics.GetMetric("sm_activity")
			if err != nil {
				c.logger.Error("failed to get sm_activity metric", slog.Any("error", err))
				return
			}
			c.logger.Info(fmt.Sprintf("sm_activity as float: %f", percentage))
			ch <- prometheus.MustNewConstMetric(
				metric.desc,
				prometheus.GaugeValue,
				percentage,
				fmt.Sprintf("%d", gpuIDX),
			)
		}(i, jsonBody)
	}
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *smbpbiCollector) Collect(ch chan<- prometheus.Metric) {
	c.collectSMBPBITemperature(ch)
	c.collectSMBStreamingMultiprocessorActivity(ch)
}

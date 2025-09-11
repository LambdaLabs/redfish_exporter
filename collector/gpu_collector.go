package collector

import (
	"log/slog"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

var (
	gpuSubsystem    = "gpu"
	gpuBaseLabels   = []string{"resource", "system", "gpu_id"}
	gpuMemoryLabels = baseWithExtraLabels([]string{"memory_id"})
	gpuMetrics      = createGPUMetricMap()
)

func baseWithExtraLabels(extra []string) []string {
	gpuBaseLabelsCopy := make([]string, len(gpuBaseLabels))
	copy(gpuBaseLabelsCopy, gpuBaseLabels)
	return append(gpuBaseLabelsCopy, extra...)
}

type GPUCollector struct {
	rfClient              *gofish.APIClient
	metrics               map[string]Metric
	logger                *slog.Logger
	collectorScrapeStatus *prometheus.GaugeVec
}

func createGPUMetricMap() map[string]Metric {
	gpuMetrics := make(map[string]Metric)
	addToMetricMap(gpuMetrics, gpuSubsystem, "health", "health of gpu reported by system,1(OK),2(Warning),3(Critical)", gpuBaseLabels)
	addToMetricMap(gpuMetrics, gpuSubsystem, "memory_ecc_correctable", "current correctable memory ecc errors reported on the gpu", gpuMemoryLabels)
	addToMetricMap(gpuMetrics, gpuSubsystem, "memory_ecc_uncorrectable", "current uncorrectable memory ecc errors reported on the gpu", gpuMemoryLabels)
	return gpuMetrics
}

func NewGPUCollector(rfClient *gofish.APIClient, logger *slog.Logger) *GPUCollector {
	return &GPUCollector{
		rfClient: rfClient,
		metrics:  gpuMetrics,
		logger:   logger.With(slog.String("collector", "GPUCollector")),
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

func (g *GPUCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range g.metrics {
		ch <- metric.desc
	}
	g.collectorScrapeStatus.Describe(ch)
}

func (g *GPUCollector) Collect(ch chan<- prometheus.Metric) {
	service := g.rfClient.Service
	systems, err := service.Systems()
	if err != nil {
		g.logger.Error("failed getting systems",
			slog.Any("error", err.Error()),
			slog.String("operation", "service.Systems()"),
		)
		g.collectorScrapeStatus.WithLabelValues("gpu").Set(float64(0))
		return
	}
	wg := &sync.WaitGroup{}
	for _, system := range systems {
		wg.Add(1)
		// Gather GPU metrics per-discovered-system
		go collectSystemGPUMetrics(ch, system, wg, g.logger)
	}
	wg.Wait()
	g.collectorScrapeStatus.WithLabelValues("gpu").Set(float64(1))
}

func collectSystemGPUMetrics(ch chan<- prometheus.Metric, system *redfish.ComputerSystem, wg *sync.WaitGroup, logger *slog.Logger) {
	defer wg.Done()
	cpus, err := system.Processors()
	if err != nil {
		logger.Error("failed getting gpus",
			slog.Any("error", err.Error()),
			slog.String("operation", "system.Processors()"),
		)
		return
	}

	if system.Name == "" {
		system.Name = system.ID
	}

	for _, gpu := range filterGPUs(cpus) {
		if gpu.Name == "" {
			gpu.Name = gpu.ID
		}
		commonGPULabels := []string{gpu.Name, system.Name, gpu.ID}
		emitGPUHealth(ch, gpu, commonGPULabels)
		gpuMem, err := gpu.Memory()
		if err != nil {
			logger.Error("error getting gpu memory", slog.Any("error", err))
			continue
		}
		memWithMetrics := make([]MemoryWithMetrics, len(gpuMem))
		for i, mem := range gpuMem {
			memWithMetrics[i] = &redfishMemoryAdapter{Memory: mem}
		}
		emitGPUECCMetrics(ch, memWithMetrics, logger, commonGPULabels)
	}
}

type MemoryWithMetrics interface {
	Metrics() (*redfish.MemoryMetrics, error)
	GetID() string
}

type redfishMemoryAdapter struct {
	*redfish.Memory
}

func (r *redfishMemoryAdapter) GetID() string {
	return r.ID
}

func emitGPUECCMetrics(ch chan<- prometheus.Metric, mem []MemoryWithMetrics, logger *slog.Logger, commonLabels []string) {
	for _, m := range mem {
		memMetric, err := m.Metrics()
		if err != nil {
			logger.Error("error getting gpu memory metrics", slog.Any("error", err))
			continue
		}
		metricLabels := append(commonLabels, m.GetID())

		ch <- prometheus.MustNewConstMetric(
			gpuMetrics["gpu_memory_ecc_correctable"].desc,
			prometheus.CounterValue,
			float64(memMetric.CurrentPeriod.CorrectableECCErrorCount),
			metricLabels...)
		ch <- prometheus.MustNewConstMetric(
			gpuMetrics["gpu_memory_ecc_uncorrectable"].desc,
			prometheus.CounterValue,
			float64(memMetric.CurrentPeriod.UncorrectableECCErrorCount),
			metricLabels...)
	}
}

func emitGPUHealth(ch chan<- prometheus.Metric, gpu *redfish.Processor, commonLabels []string) {
	if gpuStatusHealthValue, ok := parseCommonStatusHealth(gpu.Status.Health); ok {
		ch <- prometheus.MustNewConstMetric(
			gpuMetrics["gpu_health"].desc,
			prometheus.GaugeValue,
			gpuStatusHealthValue,
			commonLabels...)
	}
}

func filterGPUs(cpus []*redfish.Processor) []*redfish.Processor {
	gpus := []*redfish.Processor{}

	for _, cpu := range cpus {
		if cpu.ProcessorType == redfish.GPUProcessorType {
			gpus = append(gpus, cpu)
		}
	}
	return gpus
}

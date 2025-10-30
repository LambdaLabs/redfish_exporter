package collector

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"slices"

	"github.com/LambdaLabs/redfish_exporter/config"
	"github.com/itchyny/gojq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
)

type DescWithLabels struct {
	SortedLabels []string
	Desc         *prometheus.Desc
}

// JSONYieldedMetric is a metric-like struct built from the output of applying a JQ filter
// to some collected Redfish JSON output (like OEM data).
type JSONYieldedMetric struct {
	Name   string
	Help   string
	Value  float64
	Labels map[string]string
}

// JSONCollector is a collector which probes a particular Redfish path, applies a given JQ filter, and returns metrics accordingly.
type JSONCollector struct {
	redfishClient  *gofish.APIClient
	config         *config.JSONCollectorConfig
	jqQuery        *gojq.Query
	logger         *slog.Logger
	cachedResponse []byte
	metricsDescs   map[string]DescWithLabels
}

// NewJSONCollector yields a JSON collector.
// During creation, the collector will probe the configured endpoint, caching the response
// for followup processing.
func NewJSONCollector(redfishClient *gofish.APIClient, logger *slog.Logger, config *config.JSONCollectorConfig) (*JSONCollector, error) {
	query, err := gojq.Parse(config.JQFilter)
	if err != nil {
		return nil, fmt.Errorf("jq parse error in collector creation: %w", err)
	}
	rawClient, err := redfishClient.Get(config.RedfishRoot)
	if err != nil {
		return nil, fmt.Errorf("json collector could not perform lookup against %s: %w", config.RedfishRoot, err)
	}
	body, err := io.ReadAll(rawClient.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to cache json response: %w", err)
	}

	return &JSONCollector{
		redfishClient:  redfishClient,
		config:         config,
		jqQuery:        query,
		logger:         logger,
		cachedResponse: body,
		metricsDescs:   map[string]DescWithLabels{},
	}, nil
}

// Describe implements prometheus.Collector
func (j *JSONCollector) Describe(ch chan<- *prometheus.Desc) {
	metrics, err := metricsFromBody(j.jqQuery, j.cachedResponse)
	if err != nil {
		j.logger.Error("failed to convert collected data to a metrics description", slog.Any("error", err))
		return
	}
	for _, metric := range metrics {
		labelNames := maps.Keys(metric.Labels)
		sorted := slices.Sorted(labelNames)
		d := prometheus.NewDesc(metric.Name, metric.Help, sorted, nil)
		j.metricsDescs[metric.Name] = DescWithLabels{
			SortedLabels: sorted,
			Desc:         d,
		}
		ch <- d
	}
}

// Collect implements prometheus.Collector
func (j *JSONCollector) Collect(ch chan<- prometheus.Metric) {
	metrics, err := metricsFromBody(j.jqQuery, j.cachedResponse)
	if err != nil {
		j.logger.Error("failed to convert collected data to metrics", slog.Any("error", err))
		return
	}
	for _, metric := range metrics {
		labels := []string{}
		for _, desiredLabel := range j.metricsDescs[metric.Name].SortedLabels {
			if lVal, ok := metric.Labels[desiredLabel]; ok {
				labels = append(labels, lVal)
			} else {
				labels = append(labels, "")
			}
		}
		ch <- prometheus.MustNewConstMetric(
			j.metricsDescs[metric.Name].Desc,
			prometheus.GaugeValue,
			metric.Value,
			labels...,
		)
	}
}

// metricsFromBody applies the given gojq.Query to a Redfish response body.
// It is expected that the result of JQ application yields data in a format which may further be
// converted to a typed struct. A slice of JSONYieldedMetric is returned then for all data which meets
// this expectation.
// An error during JQ parsing results in returning a nil slice and the encountered error.
func metricsFromBody(query *gojq.Query, jsonBody []byte) ([]JSONYieldedMetric, error) {
	var yielded []JSONYieldedMetric
	var intermediary map[string]any

	if err := json.Unmarshal(jsonBody, &intermediary); err != nil {
		return yielded, err
	}
	iter := query.Run(intermediary)

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
				break
			}
			return nil, err
		}
		if container, ok := v.([]any); ok {
			for _, items := range container {
				if item, ok := items.(map[string]any); ok {
					yielded = append(yielded, convertToMetric(item))
				}
			}
		}
	}
	return yielded, nil
}

// converttoMetric yields a typed struct generated through type assertions
// on an item, where the item was generated elsewhere by applying JQ expressions against Redfish
// data. The exact desired input format is documented more completely
// in the exporter's user documentation.
func convertToMetric(item map[string]any) JSONYieldedMetric {
	ret := JSONYieldedMetric{
		Labels: map[string]string{},
	}
	if iName, ok := item["name"]; ok {
		if strName, ok := iName.(string); ok {
			ret.Name = strName
		}
	}
	if iVal, ok := item["value"]; ok {
		if floatVal, ok := iVal.(float64); ok {
			ret.Value = floatVal
		}
	}
	if iHelp, ok := item["help"]; ok {
		if strHelp, ok := iHelp.(string); ok {
			ret.Help = strHelp
		}
	}
	if iLabels, ok := item["labels"]; ok {
		if mapLabels, ok := iLabels.(map[string]any); ok {
			for lName, lVal := range mapLabels {
				if valStr, ok := lVal.(string); ok {
					ret.Labels[lName] = valStr
				}
			}
		}
	}
	return ret
}

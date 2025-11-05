package collector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"slices"
	"sync"

	"github.com/LambdaLabs/redfish_exporter/config"
	"github.com/itchyny/gojq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
)

// DescWithLabels wraps prometheus.Desc with a sorted slice of label names, allowing
// emitted metric labels to also be sorted for consistency.
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
	redfishClient     *gofish.APIClient
	config            *config.JSONCollectorConfig
	jqQuery           *gojq.Query
	logger            *slog.Logger
	once              *sync.Once
	cacheCollectError error
	cache             []byte
	metricsDescs      map[string]DescWithLabels
}

// NewJSONCollector yields a JSON collector.
func NewJSONCollector(redfishClient *gofish.APIClient, logger *slog.Logger, config *config.JSONCollectorConfig) (*JSONCollector, error) {
	query, err := gojq.Parse(config.JQFilter)
	if err != nil {
		return nil, fmt.Errorf("jq parse error in collector creation: %w", err)
	}

	return &JSONCollector{
		redfishClient:     redfishClient,
		config:            config,
		jqQuery:           query,
		logger:            logger,
		once:              &sync.Once{},
		cacheCollectError: nil,
		cache:             []byte{}, //newCache(),
		metricsDescs: map[string]DescWithLabels{
			"redfish_collector_json_parse_success": {
				SortedLabels: []string{},
				Desc: prometheus.NewDesc("redfish_collector_json_parse_failure_last",
					"Indicates if JSON parsing of the redfish endpoint was successful (0=no,1=yes)", nil, nil),
			},
		},
	}, nil
}

// redfishResponse queries the Redfish API configured for a JSONCollector.
// Using sync.Once, data will be saved to the JSONCollector cache for consistency
// in reuse by j.Describe and j.Collect
func (j *JSONCollector) redfishResponse() ([]byte, error) {
	j.once.Do(func() {
		rawClient, err := j.redfishClient.Get(j.config.RedfishRoot)
		if err != nil {
			j.cacheCollectError = fmt.Errorf("json collector could not perform initial lookup against %s: %w", j.config.RedfishRoot, err)
		}
		body, err := io.ReadAll(rawClient.Body)
		if err != nil {
			j.cacheCollectError = fmt.Errorf("json collector could not cache initial lookup against %s: %w", j.config.RedfishRoot, err)
		}
		j.cache = body
	})

	return j.cache, j.cacheCollectError
}

// Describe implements prometheus.Collector.
// Describe is called during Collector registration, and due to the dynamic
// nature of this collector, it is impossible to know all labels or timeseries descriptions
// at compile time.
// As a result, Describe will check for the existence of cached Redfish API data.
// If no cache is populated, Describe will collect and set the JSONCollector's cached Redfish API data,
// so that followup calls (e.g. to Collect) operate on a consistent set of data.
func (j *JSONCollector) Describe(ch chan<- *prometheus.Desc) {
	ctx, cancel := context.WithTimeout(context.Background(), j.config.Timeout)
	defer cancel()
	body, err := j.redfishResponse()
	if err != nil {
		j.logger.Warn("skipping Describe() as Redfish data was unavailable")
		return
	}

	metrics, err := metricsFromBody(ctx, j.jqQuery, body)
	if err != nil {
		j.logger.Error("failed to convert collected data to a metrics description",
			slog.Any("error", err),
			slog.Any("response_body", string(body)))
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

// Collect implements prometheus.Collector.
// Collect should generally be called after j.Describe, but in the chance
// that this changes in the future, Collect will first check for cached Redfish response
// data.
// If no cache is populated, Collect will fetch and cache the Redfish API data before
// processing and emitting timeseries.
func (j *JSONCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), j.config.Timeout)
	defer cancel()
	body, err := j.redfishResponse()
	if err != nil {
		j.logger.Warn("skipping Collect() as Redfish data was unavailable")
		return
	}

	metrics, err := metricsFromBody(ctx, j.jqQuery, body)
	if err != nil {
		j.logger.Error("failed to convert collected data to metrics",
			slog.Any("error", err),
			slog.Any("response_body", string(body)))
		ch <- prometheus.MustNewConstMetric(
			j.metricsDescs["redfish_collector_json_parse_success"].Desc,
			prometheus.GaugeValue, 0)
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
	ch <- prometheus.MustNewConstMetric(
		j.metricsDescs["redfish_collector_json_parse_success"].Desc,
		prometheus.GaugeValue, 1)
}

// metricsFromBody applies the given gojq.Query to a Redfish response body.
// It is expected that the result of JQ application yields data in a format which may further be
// converted to a typed struct. A slice of JSONYieldedMetric is returned then for all data which meets
// this expectation.
// An error during JQ parsing results skips the item, and errors encountered in this way
// are joined together and returned as a bundle.
func metricsFromBody(ctx context.Context, query *gojq.Query, jsonBody []byte) ([]JSONYieldedMetric, error) {
	var yielded []JSONYieldedMetric
	var parseErrors []error
	var intermediary map[string]any

	if err := json.Unmarshal(jsonBody, &intermediary); err != nil {
		return yielded, err
	}
	iter := query.RunWithContext(ctx, intermediary)

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
				break
			}
			return []JSONYieldedMetric{}, err
		}
		if container, ok := v.([]any); ok {
			for idx, items := range container {
				if item, ok := items.(map[string]any); ok {
					yieldedMetric, err := convertToMetric(item)
					if err != nil {
						parseErrors = append(parseErrors, fmt.Errorf("item %d from response failed to parse: %w", idx, err))
						continue
					}
					yielded = append(yielded, yieldedMetric)
				}
			}
		}
	}
	return yielded, errors.Join(parseErrors...)
}

// converttoMetric yields a typed struct generated through type assertions
// on an item, where the item was generated elsewhere by applying JQ expressions against Redfish
// data. The exact desired input format is documented more completely
// in the exporter's user documentation.
func convertToMetric(item map[string]any) (JSONYieldedMetric, error) {
	ret := JSONYieldedMetric{
		Labels: map[string]string{},
	}
	var convertErrors []error
	keys := slices.Sorted(maps.Keys(item))

	if iName, ok := item["name"]; ok {
		if strName, ok := iName.(string); ok {
			ret.Name = strName
		} else {
			convertErrors = append(convertErrors, fmt.Errorf("item contained a non-string name"))
		}
	} else {
		convertErrors = append(convertErrors, fmt.Errorf("item missing name, provided keys: %s", keys))
	}

	if iVal, ok := item["value"]; ok {
		if floatVal, ok := iVal.(float64); ok {
			ret.Value = floatVal
		} else {
			convertErrors = append(convertErrors, fmt.Errorf("item contained a non-float value"))
		}
	} else {
		convertErrors = append(convertErrors, fmt.Errorf("item missing value, provided keys: %s", keys))
	}
	if iHelp, ok := item["help"]; ok {
		if strHelp, ok := iHelp.(string); ok {
			ret.Help = strHelp
		} else {
			convertErrors = append(convertErrors, fmt.Errorf("item contained a non-string help"))
		}
	} else {
		convertErrors = append(convertErrors, fmt.Errorf("item missing help, provided keys: %s", keys))
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

	return ret, errors.Join(convertErrors...)
}

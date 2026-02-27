package main

import (
	"fmt"

	promexporter "go.opentelemetry.io/otel/exporters/prometheus"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type meterProviderOption func(*meterProviderConfig)

type meterProviderConfig struct {
	reader sdkmetric.Reader
}

func withReader(r sdkmetric.Reader) meterProviderOption {
	return func(c *meterProviderConfig) {
		c.reader = r
	}
}

func initOTelMeterProvider(opts ...meterProviderOption) (*sdkmetric.MeterProvider, error) {
	cfg := &meterProviderConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	reader := cfg.reader
	if reader == nil {
		promExporter, err := promexporter.New()
		if err != nil {
			return nil, fmt.Errorf("creating prometheus exporter: %w", err)
		}
		reader = promExporter
	}

	durationView := sdkmetric.NewView(
		sdkmetric.Instrument{Name: "http.client.request.duration"},
		sdkmetric.Stream{
			Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
			},
			// server.address and server.port are high-cardinality (one value per BMC target IP)
			// and must be excluded to keep the metric manageable.
			AttributeFilter: attribute.NewDenyKeysFilter(
				attribute.Key("server.address"),
				attribute.Key("server.port"),
			),
		},
	)

	dropRequestBodySizeView := sdkmetric.NewView(
		sdkmetric.Instrument{Name: "http.client.request.body.size"},
		sdkmetric.Stream{
			Aggregation: sdkmetric.AggregationDrop{},
		},
	)

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithView(durationView),
		sdkmetric.WithView(dropRequestBodySizeView),
	)
	otel.SetMeterProvider(provider)
	return provider, nil
}

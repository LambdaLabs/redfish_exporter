package main

import (
	"fmt"

	"go.opentelemetry.io/otel"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func initOTelMeterProvider(readers ...sdkmetric.Reader) (*sdkmetric.MeterProvider, error) {
	var reader sdkmetric.Reader
	if len(readers) == 0 {
		promExporter, err := promexporter.New()
		if err != nil {
			return nil, fmt.Errorf("creating prometheus exporter: %w", err)
		}
		reader = promExporter
	} else {
		reader = readers[0]
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

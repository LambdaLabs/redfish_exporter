package main

import (
	"fmt"

	"go.opentelemetry.io/otel"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func initOTelMeterProvider() (*sdkmetric.MeterProvider, error) {
	promExporter, err := promexporter.New()
	if err != nil {
		return nil, fmt.Errorf("creating prometheus exporter: %w", err)
	}

	durationView := sdkmetric.NewView(
		sdkmetric.Instrument{Name: "http.client.request.duration"},
		sdkmetric.Stream{
			Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
			},
		},
	)

	dropRequestBodySizeView := sdkmetric.NewView(
		sdkmetric.Instrument{Name: "http.client.request.body.size"},
		sdkmetric.Stream{
			Aggregation: sdkmetric.AggregationDrop{},
		},
	)

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithView(durationView),
		sdkmetric.WithView(dropRequestBodySizeView),
	)
	otel.SetMeterProvider(provider)
	return provider, nil
}

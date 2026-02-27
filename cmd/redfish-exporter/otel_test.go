package main

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// newTestMeterProvider creates a MeterProvider with a ManualReader and the same
// view configurations as initOTelMeterProvider, but without touching the global
// Prometheus registry or the global OTel meter provider.
func newTestMeterProvider(t *testing.T) (*sdkmetric.MeterProvider, *sdkmetric.ManualReader) {
	t.Helper()

	reader := sdkmetric.NewManualReader()

	durationView := sdkmetric.NewView(
		sdkmetric.Instrument{Name: "http.client.request.duration"},
		sdkmetric.Stream{
			Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
			},
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
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
	})

	return provider, reader
}

// TestOTelAttributeFilter verifies that the attribute filter drops server.address
// and server.port from http.client.request.duration data points while keeping
// other attributes such as module.
func TestOTelAttributeFilter(t *testing.T) {
	provider, reader := newTestMeterProvider(t)

	meter := provider.Meter("test")
	histogram, err := meter.Float64Histogram("http.client.request.duration")
	if err != nil {
		t.Fatalf("creating histogram: %v", err)
	}

	ctx := context.Background()
	histogram.Record(ctx, 0.5,
		metric.WithAttributes(
			attribute.String("server.address", "192.168.1.1"),
			attribute.Int("server.port", 443),
			attribute.String("module", "chassis_collector"),
		),
	)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("collecting metrics: %v", err)
	}

	// Locate the http.client.request.duration metric.
	var found *metricdata.Metrics
	for i := range rm.ScopeMetrics {
		for j := range rm.ScopeMetrics[i].Metrics {
			m := &rm.ScopeMetrics[i].Metrics[j]
			if m.Name == "http.client.request.duration" {
				found = m
				break
			}
		}
	}

	if found == nil {
		t.Fatal("http.client.request.duration metric not found in collected data")
	}

	hist, ok := found.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected Histogram[float64], got %T", found.Data)
	}

	if len(hist.DataPoints) == 0 {
		t.Fatal("expected at least one data point, got none")
	}

	dp := hist.DataPoints[0]
	attrs := dp.Attributes

	// server.address and server.port must be absent.
	if _, exists := attrs.Value(attribute.Key("server.address")); exists {
		t.Error("server.address attribute should have been filtered out but was present")
	}
	if _, exists := attrs.Value(attribute.Key("server.port")); exists {
		t.Error("server.port attribute should have been filtered out but was present")
	}

	// module=chassis_collector must be present.
	moduleVal, exists := attrs.Value(attribute.Key("module"))
	if !exists {
		t.Error("module attribute should be present but was absent")
	} else if moduleVal.AsString() != "chassis_collector" {
		t.Errorf("module attribute: got %q, want %q", moduleVal.AsString(), "chassis_collector")
	}
}

// TestOTelRequestBodySizeDropped verifies that http.client.request.body.size
// measurements are completely dropped and produce no data points.
func TestOTelRequestBodySizeDropped(t *testing.T) {
	provider, reader := newTestMeterProvider(t)

	meter := provider.Meter("test")
	histogram, err := meter.Int64Histogram("http.client.request.body.size")
	if err != nil {
		t.Fatalf("creating histogram: %v", err)
	}

	ctx := context.Background()
	histogram.Record(ctx, 1024,
		metric.WithAttributes(
			attribute.String("server.address", "192.168.1.1"),
			attribute.Int("server.port", 443),
		),
	)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("collecting metrics: %v", err)
	}

	// The metric should not appear at all in the collected data.
	for i := range rm.ScopeMetrics {
		for j := range rm.ScopeMetrics[i].Metrics {
			if rm.ScopeMetrics[i].Metrics[j].Name == "http.client.request.body.size" {
				t.Error("http.client.request.body.size should have been dropped but was collected")
			}
		}
	}
}

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestOTelMeterProviderViews verifies the views configured on the OTel meter provider:
//   - server.address and server.port are absent from http.client.request.duration data points
//   - module attribute is present with the correct value
//   - http.client.request.body.size produces no data points
func TestOTelMeterProviderViews(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := initOTelMeterProvider(withReader(reader))
	require.NoError(t, err, "creating meter provider")
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
	})

	ctx := context.Background()
	meter := provider.Meter("test")

	// Record a http.client.request.duration observation with high-cardinality attributes.
	durationHist, err := meter.Float64Histogram("http.client.request.duration")
	require.NoError(t, err, "creating http.client.request.duration histogram")
	durationHist.Record(ctx, 0.5,
		metric.WithAttributes(
			attribute.String("server.address", "192.168.1.1"),
			attribute.Int("server.port", 443),
			attribute.String("module", "chassis_collector"),
		),
	)

	// Record a http.client.request.body.size observation that should be dropped entirely.
	bodySizeHist, err := meter.Int64Histogram("http.client.request.body.size")
	require.NoError(t, err, "creating http.client.request.body.size histogram")
	bodySizeHist.Record(ctx, 1024,
		metric.WithAttributes(
			attribute.String("server.address", "192.168.1.1"),
			attribute.Int("server.port", 443),
		),
	)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm), "collecting metrics")

	// Locate http.client.request.duration in the collected data.
	var durationMetric *metricdata.Metrics
	for i := range rm.ScopeMetrics {
		for j := range rm.ScopeMetrics[i].Metrics {
			m := &rm.ScopeMetrics[i].Metrics[j]
			switch m.Name {
			case "http.client.request.duration":
				durationMetric = m
			case "http.client.request.body.size":
				assert.Fail(t, "http.client.request.body.size should have been dropped but was collected")
			}
		}
	}

	require.NotNil(t, durationMetric, "http.client.request.duration metric not found in collected data")

	hist, ok := durationMetric.Data.(metricdata.Histogram[float64])
	require.True(t, ok, "expected Histogram[float64], got %T", durationMetric.Data)
	require.NotEmpty(t, hist.DataPoints, "expected at least one data point, got none")

	attrs := hist.DataPoints[0].Attributes

	// server.address and server.port must be absent.
	_, exists := attrs.Value(attribute.Key("server.address"))
	assert.False(t, exists, "server.address attribute should have been filtered out but was present")

	_, exists = attrs.Value(attribute.Key("server.port"))
	assert.False(t, exists, "server.port attribute should have been filtered out but was present")

	// module=chassis_collector must be present.
	moduleVal, exists := attrs.Value(attribute.Key("module"))
	if assert.True(t, exists, "module attribute should be present but was absent") {
		assert.Equal(t, "chassis_collector", moduleVal.AsString(), "module attribute value mismatch")
	}
}

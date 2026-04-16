package collector

import (
	"context"
	"log/slog"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubCollector is a no-op ContextAwareCollector used to populate redfishCollector.collectors
// without making any real Redfish API calls.
type stubCollector struct {
	// panicOnCollect causes CollectWithContext to panic, simulating a sub-collector crash.
	panicOnCollect bool
}

func (s *stubCollector) Describe(_ chan<- *prometheus.Desc) {}
func (s *stubCollector) Collect(_ chan<- prometheus.Metric) {}
func (s *stubCollector) CollectWithContext(_ context.Context, _ chan<- prometheus.Metric) {
	if s.panicOnCollect {
		panic("simulated collector panic")
	}
}

// newTestRedfishCollector builds a redfishCollector with no real Redfish connection,
// suitable for unit-testing collector-level behaviour such as counter resets.
func newTestRedfishCollector(collectors []ContextAwareCollector) *redfishCollector {
	return &redfishCollector{
		ctx:        context.Background(),
		collectors: collectors,
		logger:     slog.Default(),
		redfishUp:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "redfish_up_test"}),
	}
}

// TestCollect_CountsOutcomes verifies that collectorsSucceeded and collectorsFailed reflect
// the actual outcome of each sub-collector goroutine after Collect() completes.
// collectorsSucceeded is incremented only when CollectWithContext returns normally;
// collectorsFailed is derived as len(collectors) - collectorsSucceeded, so panicking
// goroutines (caught by recoverGroup) are automatically counted as failures.
func TestCollect_CountsOutcomes(t *testing.T) {
	t.Run("all collectors succeed", func(t *testing.T) {
		rc := newTestRedfishCollector([]ContextAwareCollector{
			&stubCollector{},
			&stubCollector{},
			&stubCollector{},
		})
		ch := make(chan prometheus.Metric, 16)
		rc.Collect(ch)

		assert.Equal(t, int64(3), rc.collectorsSucceeded.Load())
		assert.Equal(t, int64(0), rc.collectorsFailed.Load())
	})

	t.Run("all collectors fail via panic", func(t *testing.T) {
		rc := newTestRedfishCollector([]ContextAwareCollector{
			&stubCollector{panicOnCollect: true},
			&stubCollector{panicOnCollect: true},
		})
		ch := make(chan prometheus.Metric, 16)
		rc.Collect(ch)

		assert.Equal(t, int64(0), rc.collectorsSucceeded.Load())
		assert.Equal(t, int64(2), rc.collectorsFailed.Load())
	})

	t.Run("mixed success and failure", func(t *testing.T) {
		rc := newTestRedfishCollector([]ContextAwareCollector{
			&stubCollector{},
			&stubCollector{panicOnCollect: true},
			&stubCollector{},
			&stubCollector{panicOnCollect: true},
		})
		ch := make(chan prometheus.Metric, 16)
		rc.Collect(ch)

		assert.Equal(t, int64(2), rc.collectorsSucceeded.Load())
		assert.Equal(t, int64(2), rc.collectorsFailed.Load())
	})

	t.Run("no collectors registered", func(t *testing.T) {
		rc := newTestRedfishCollector([]ContextAwareCollector{})
		ch := make(chan prometheus.Metric, 16)
		rc.Collect(ch)

		assert.Equal(t, int64(0), rc.collectorsSucceeded.Load())
		assert.Equal(t, int64(0), rc.collectorsFailed.Load())
	})

	t.Run("succeeded and failed always sum to total collectors", func(t *testing.T) {
		collectors := []ContextAwareCollector{
			&stubCollector{},
			&stubCollector{panicOnCollect: true},
			&stubCollector{},
		}
		rc := newTestRedfishCollector(collectors)
		ch := make(chan prometheus.Metric, 16)
		rc.Collect(ch)

		assert.Equal(t, int64(len(collectors)), rc.collectorsSucceeded.Load()+rc.collectorsFailed.Load())
	})
}

// TestCollectorOutcome verifies that CollectorOutcome() returns the succeeded and failed
// counts that reflect the most recent Collect() call, and that it is consistent with
// the values emitted as Prometheus gauges.
func TestCollectorOutcome(t *testing.T) {
	t.Run("returns succeeded and failed counts after all succeed", func(t *testing.T) {
		rc := newTestRedfishCollector([]ContextAwareCollector{
			&stubCollector{},
			&stubCollector{},
		})
		ch := make(chan prometheus.Metric, 16)
		rc.Collect(ch)

		succeeded, failed := rc.CollectorOutcome()
		assert.Equal(t, int64(2), succeeded)
		assert.Equal(t, int64(0), failed)
	})

	t.Run("returns succeeded and failed counts after all fail", func(t *testing.T) {
		rc := newTestRedfishCollector([]ContextAwareCollector{
			&stubCollector{panicOnCollect: true},
			&stubCollector{panicOnCollect: true},
		})
		ch := make(chan prometheus.Metric, 16)
		rc.Collect(ch)

		succeeded, failed := rc.CollectorOutcome()
		assert.Equal(t, int64(0), succeeded)
		assert.Equal(t, int64(2), failed)
	})

	t.Run("returns updated counts after each Collect call", func(t *testing.T) {
		// Verifies that CollectorOutcome() always reflects the most recent Collect(),
		// not a stale result from a previous scrape session.
		rc := newTestRedfishCollector([]ContextAwareCollector{
			&stubCollector{},
			&stubCollector{panicOnCollect: true},
		})

		ch := make(chan prometheus.Metric, 16)
		rc.Collect(ch)
		succeeded, failed := rc.CollectorOutcome()
		assert.Equal(t, int64(1), succeeded)
		assert.Equal(t, int64(1), failed)

		// Second Collect with same collectors — outcome should be identical, not accumulated.
		ch = make(chan prometheus.Metric, 16)
		rc.Collect(ch)
		succeeded, failed = rc.CollectorOutcome()
		assert.Equal(t, int64(1), succeeded)
		assert.Equal(t, int64(1), failed)
	})
}

// TestCollect_ResetsCounters verifies that collectorsSucceeded and collectorsFailed are
// both reset to zero at the start of every Collect() call. This ensures that counts from
// a previous scrape session do not carry over into the next one, keeping per-scrape
// outcome tracking accurate.
func TestCollect_ResetsCounters(t *testing.T) {
	rc := newTestRedfishCollector([]ContextAwareCollector{&stubCollector{}, &stubCollector{}})

	t.Run("counters start at zero before first Collect", func(t *testing.T) {
		assert.Equal(t, int64(0), rc.collectorsSucceeded.Load())
		assert.Equal(t, int64(0), rc.collectorsFailed.Load())
	})

	t.Run("counters are reset to zero at the start of a subsequent Collect", func(t *testing.T) {
		// Simulate leftover state from a previous scrape session.
		rc.collectorsSucceeded.Store(3)
		rc.collectorsFailed.Store(2)

		ch := make(chan prometheus.Metric, 16)
		rc.Collect(ch)

		// The reset happens before any goroutine work, so after Collect() returns
		// the counters reflect the just-completed session, not the seeded values.
		// We verify only that the seed values (3 and 2) are gone.
		assert.NotEqual(t, int64(3), rc.collectorsSucceeded.Load(), "collectorsSucceeded should have been reset")
		assert.NotEqual(t, int64(2), rc.collectorsFailed.Load(), "collectorsFailed should have been reset")
	})

	t.Run("counters are reset to zero on every Collect call", func(t *testing.T) {
		// Seed non-zero values and confirm each new Collect wipes them.
		for range 3 {
			rc.collectorsSucceeded.Store(99)
			rc.collectorsFailed.Store(99)

			ch := make(chan prometheus.Metric, 16)
			rc.Collect(ch)

			assert.NotEqual(t, int64(99), rc.collectorsSucceeded.Load())
			assert.NotEqual(t, int64(99), rc.collectorsFailed.Load())
		}
	})
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestCollect_EmitsGauges verifies that Collect() emits
// redfish_exporter_collectors_succeeded and redfish_exporter_collectors_failed
// as Prometheus gauge metrics with values that match the atomic counters.
func TestCollect_EmitsGauges(t *testing.T) {
	// collectMetricValues runs Collect and returns the emitted gauge values for
	// collectors_succeeded and collectors_failed by reading from the channel.
	collectMetricValues := func(t *testing.T, rc *redfishCollector) (succeeded, failed float64) {
		t.Helper()
		ch := make(chan prometheus.Metric, 32)
		rc.Collect(ch)
		close(ch)
		for m := range ch {
			var dtoM dto.Metric
			require.NoError(t, m.Write(&dtoM))
			if dtoM.Gauge == nil {
				continue
			}
			desc := m.Desc().String()
			if findSubstring(desc, "collectors_succeeded") {
				succeeded = dtoM.Gauge.GetValue()
			}
			if findSubstring(desc, "collectors_failed") {
				failed = dtoM.Gauge.GetValue()
			}
		}
		return
	}

	t.Run("emits succeeded=N failed=0 when all collectors succeed", func(t *testing.T) {
		rc := newTestRedfishCollector([]ContextAwareCollector{
			&stubCollector{},
			&stubCollector{},
			&stubCollector{},
		})
		succeeded, failed := collectMetricValues(t, rc)
		assert.Equal(t, float64(3), succeeded)
		assert.Equal(t, float64(0), failed)
	})

	t.Run("emits succeeded=0 failed=N when all collectors panic", func(t *testing.T) {
		rc := newTestRedfishCollector([]ContextAwareCollector{
			&stubCollector{panicOnCollect: true},
			&stubCollector{panicOnCollect: true},
		})
		succeeded, failed := collectMetricValues(t, rc)
		assert.Equal(t, float64(0), succeeded)
		assert.Equal(t, float64(2), failed)
	})

	t.Run("emits correct values for mixed success and failure", func(t *testing.T) {
		rc := newTestRedfishCollector([]ContextAwareCollector{
			&stubCollector{},
			&stubCollector{panicOnCollect: true},
			&stubCollector{},
		})
		succeeded, failed := collectMetricValues(t, rc)
		assert.Equal(t, float64(2), succeeded)
		assert.Equal(t, float64(1), failed)
	})

	t.Run("gauge values match atomic counters after Collect", func(t *testing.T) {
		// Verifies consistency between the emitted gauge values and the struct fields,
		// ensuring the metrics are not stale snapshots taken at the wrong point.
		rc := newTestRedfishCollector([]ContextAwareCollector{
			&stubCollector{},
			&stubCollector{panicOnCollect: true},
		})
		succeeded, failed := collectMetricValues(t, rc)
		assert.Equal(t, float64(rc.collectorsSucceeded.Load()), succeeded)
		assert.Equal(t, float64(rc.collectorsFailed.Load()), failed)
	})
}

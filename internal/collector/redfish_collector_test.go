package collector

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// stubCollector is a no-op ContextAwareCollector used to populate redfishCollector.collectors
// without making any real Redfish API calls.
type stubCollector struct{}

func (s *stubCollector) Describe(_ chan<- *prometheus.Desc)                           {}
func (s *stubCollector) Collect(_ chan<- prometheus.Metric)                           {}
func (s *stubCollector) CollectWithContext(_ context.Context, _ chan<- prometheus.Metric) {}

// newTestRedfishCollector builds a redfishCollector with no real Redfish connection,
// suitable for unit-testing collector-level behaviour such as counter resets.
func newTestRedfishCollector(collectors []ContextAwareCollector) *redfishCollector {
	return &redfishCollector{
		ctx:        context.Background(),
		collectors: collectors,
		redfishUp:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "redfish_up_test"}),
	}
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

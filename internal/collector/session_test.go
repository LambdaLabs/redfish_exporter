package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
)

// sessionBMC is a fake Redfish endpoint implementing the session service, which tracks how
// many session slots are currently occupied. Slot accounting is the assertion that matters
// for session-lifecycle tests: a BMC caps concurrent sessions and refuses every new one once
// the cap is reached, so the question is always "did the slot come back".
type sessionBMC struct {
	*httptest.Server

	mu      sync.Mutex
	open    map[string]bool
	created int
	deleted int

	// refuseDeletes makes the session service answer a DELETE with 503, as a BMC at its
	// session cap does. The slot stays occupied.
	refuseDeletes bool

	// refuseDeleteAttempts makes the session service answer that many DELETEs with 503
	// before behaving normally, reproducing a teardown that only a retry gets through.
	refuseDeleteAttempts int

	// refuseTokenDeletes makes the session service answer 401 to a DELETE authenticated with
	// the session token, while still honouring one authenticated with credentials. Real BMCs
	// do this once they have invalidated a token whose slot they are still holding — the case
	// gofish cannot recover from, since it will only ever send the token.
	refuseTokenDeletes bool

	// authSeen records the authentication of every DELETE the session service received, in
	// order, as "token", "basic" or "none".
	authSeen []string

	// refuseSessionList makes a GET of the session collection answer 403, as a BMC does for
	// an account behind a forced password change.
	refuseSessionList bool

	// stall, when non-nil, holds a handler until the channel is closed, after that handler
	// has already applied its side effect. It reproduces a BMC that accepts a request and
	// then never answers it — the failure mode both bounds in this change exist for.
	stall chan struct{}
}

// stallHandlers makes every handler block until test cleanup, so the BMC accepts requests
// and returns no response headers.
func (b *sessionBMC) stallHandlers(t *testing.T) {
	t.Helper()
	stall := make(chan struct{})
	b.mu.Lock()
	b.stall = stall
	b.mu.Unlock()
	// Released at cleanup so the test server can shut down instead of waiting on the handler.
	t.Cleanup(func() { close(stall) })
}

func (b *sessionBMC) waitIfStalled() {
	b.mu.Lock()
	stall := b.stall
	b.mu.Unlock()
	if stall != nil {
		<-stall
	}
}

func newSessionBMC(t *testing.T) *sessionBMC {
	t.Helper()

	b := &sessionBMC{open: make(map[string]bool)}
	writeJSON := func(w http.ResponseWriter, body any) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(body))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/redfish/v1/", func(w http.ResponseWriter, _ *http.Request) {
		b.waitIfStalled()
		writeJSON(w, map[string]any{
			"@odata.id":      "/redfish/v1/",
			"@odata.type":    "#ServiceRoot.v1_15_0.ServiceRoot",
			"Id":             "RootService",
			"RedfishVersion": "1.15.0",
			"Links": map[string]any{
				"Sessions": map[string]string{"@odata.id": "/redfish/v1/SessionService/Sessions"},
			},
		})
	})

	mux.HandleFunc("/redfish/v1/SessionService/Sessions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// The session collection, as the exporter counts open slots from.
			b.mu.Lock()
			refused := b.refuseSessionList
			members := make([]map[string]string, 0, len(b.open))
			for uri := range b.open {
				members = append(members, map[string]string{"@odata.id": uri})
			}
			b.mu.Unlock()

			b.waitIfStalled()
			if refused {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			writeJSON(w, map[string]any{
				"@odata.id":           "/redfish/v1/SessionService/Sessions",
				"Members":             members,
				"Members@odata.count": len(members),
			})
		case http.MethodPost:
			b.mu.Lock()
			b.created++
			uri := fmt.Sprintf("/redfish/v1/SessionService/Sessions/%d", b.created)
			b.open[uri] = true
			b.mu.Unlock()

			w.Header().Set("Location", uri)
			w.Header().Set("X-Auth-Token", "token")
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/redfish/v1/SessionService/Sessions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		_, _, hasBasic := r.BasicAuth()
		auth := "none"
		switch {
		case hasBasic:
			auth = "basic"
		case r.Header.Get("X-Auth-Token") != "":
			auth = "token"
		}

		b.mu.Lock()
		b.deleted++
		b.authSeen = append(b.authSeen, auth)
		status := http.StatusNoContent
		switch {
		case b.refuseDeletes:
			status = http.StatusServiceUnavailable
		case b.refuseDeleteAttempts > 0:
			b.refuseDeleteAttempts--
			status = http.StatusServiceUnavailable
		case b.refuseTokenDeletes && auth != "basic":
			status = http.StatusUnauthorized
		}
		if status == http.StatusNoContent {
			delete(b.open, r.URL.Path)
		}
		b.mu.Unlock()

		b.waitIfStalled()
		w.WriteHeader(status)
	})

	// TLS, so tests exercise the real newRedfishClient path, which builds an https://
	// endpoint and relies on the client's Insecure setting for the BMCs' self-signed certs.
	b.Server = httptest.NewTLSServer(mux)
	t.Cleanup(b.Close)
	return b
}

func (b *sessionBMC) counts() (openSlots, created, deleted int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.open), b.created, b.deleted
}

// deleteAuth reports how each DELETE the session service received was authenticated.
func (b *sessionBMC) deleteAuth() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.authSeen...)
}

func (b *sessionBMC) host(t *testing.T) string {
	t.Helper()
	u, err := url.Parse(b.URL)
	require.NoError(t, err)
	return u.Host
}

func testRedfishConfig() config.RedfishClientConfig {
	return config.RedfishClientConfig{
		MaxConcurrentRequests: 1,
		DialTimeout:           2 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		LogoutTimeout:         2 * time.Second,
	}
}

// newBMCCollector builds a redfishCollector against the fake BMC via the exporter's own
// constructor, so the session is created exactly as it is in production.
func newBMCCollector(t *testing.T, ctx context.Context, b *sessionBMC) *redfishCollector {
	t.Helper()
	return newBMCCollectorWithConfig(t, ctx, b, testRedfishConfig())
}

func newBMCCollectorWithConfig(t *testing.T, ctx context.Context, b *sessionBMC, cfg config.RedfishClientConfig) *redfishCollector {
	t.Helper()
	rc, err := NewRedfishCollector(ctx, NewTestLogger(t, 0), b.host(t), "user", "pass", cfg)
	require.NoError(t, err)
	return rc
}

// TestClose_ReleasesSessionWhenScrapeContextCancelled is the leak this change targets.
// The session is opened against the inbound request's context, so when Prometheus abandons a
// slow scrape that context is already cancelled by the time collection would start. Collect()
// then takes its early-return branch, and teardown used to live only in the other branch —
// so the slot stayed occupied until the BMC's own idle timeout reclaimed it.
func TestClose_ReleasesSessionWhenScrapeContextCancelled(t *testing.T) {
	bmc := newSessionBMC(t)
	ctx, cancel := context.WithCancel(context.Background())

	rc := newBMCCollector(t, ctx, bmc)

	openSlots, created, _ := bmc.counts()
	require.Equal(t, 1, created, "constructing the collector opens exactly one session")
	require.Equal(t, 1, openSlots)

	// The scrape context dies before collection begins.
	cancel()

	ch := make(chan prometheus.Metric, 32)
	rc.Collect(ch)

	openSlots, _, deleted := bmc.counts()
	require.Equal(t, 0, deleted, "Collect() no longer owns teardown")
	require.Equal(t, 1, openSlots)

	rc.Close()

	openSlots, created, deleted = bmc.counts()
	assert.Equal(t, 0, openSlots, "the slot must be returned to the BMC")
	assert.Equal(t, 1, created)
	assert.Equal(t, 1, deleted)
}

// TestClose_ReleasesSessionAfterCollect covers the ordinary path, confirming teardown still
// happens now that it has moved out of Collect().
func TestClose_ReleasesSessionAfterCollect(t *testing.T) {
	bmc := newSessionBMC(t)
	rc := newBMCCollector(t, context.Background(), bmc)
	rc.WithCollectors([]ContextAwareCollector{&stubCollector{}})

	ch := make(chan prometheus.Metric, 32)
	rc.Collect(ch)
	rc.Close()

	openSlots, created, deleted := bmc.counts()
	assert.Equal(t, 0, openSlots)
	assert.Equal(t, 1, created)
	assert.Equal(t, 1, deleted)
}

// TestClose_ReleasesSessionWhenCollectPanics is the other path the old placement missed.
// A deferred Close() runs while the panic unwinds, which is why teardown belongs to the
// caller rather than to Collect().
func TestClose_ReleasesSessionWhenCollectPanics(t *testing.T) {
	bmc := newSessionBMC(t)
	rc := newBMCCollector(t, context.Background(), bmc)

	func() {
		defer func() {
			_ = recover()
		}()
		defer rc.Close()
		panic("simulated failure between construction and collection")
	}()

	openSlots, created, deleted := bmc.counts()
	assert.Equal(t, 0, openSlots, "a panic must not strand the session")
	assert.Equal(t, 1, created)
	assert.Equal(t, 1, deleted)
}

// TestClose_Idempotent matters because a second DELETE would target a slot the BMC may have
// already reissued to another client.
func TestClose_Idempotent(t *testing.T) {
	bmc := newSessionBMC(t)
	rc := newBMCCollector(t, context.Background(), bmc)

	rc.Close()
	rc.Close()
	rc.Close()

	openSlots, _, deleted := bmc.counts()
	assert.Equal(t, 0, openSlots)
	assert.Equal(t, 1, deleted, "only the first Close() issues a DELETE")
}

// TestClose_NoClient guards the path where the collector never got a working client, so a
// deferred Close() cannot turn a failed scrape into a panic.
func TestClose_NoClient(t *testing.T) {
	rc := newTestRedfishCollector(nil)
	assert.NotPanics(t, rc.Close)
}

// TestClose_BoundedWhenBMCGoesSilent is the second half of the leak. Teardown deliberately
// runs on a context detached from the scrape's, and gofish's Logout() substitutes
// context.Background() for a cancelled one, so without a deadline of its own the delete has
// no limit at all: a BMC that accepts the DELETE and never answers would park the handler
// goroutine and hold the session indefinitely, one goroutine per scrape, without bound.
func TestClose_BoundedWhenBMCGoesSilent(t *testing.T) {
	bmc := newSessionBMC(t)
	cfg := testRedfishConfig()
	cfg.LogoutTimeout = 250 * time.Millisecond

	rc := newBMCCollectorWithConfig(t, context.Background(), bmc, cfg)

	// Only stall once the session exists, so construction succeeds and just the DELETE hangs.
	bmc.stallHandlers(t)

	done := make(chan time.Duration, 1)
	go func() {
		start := time.Now()
		rc.Close()
		done <- time.Since(start)
	}()

	select {
	case elapsed := <-done:
		assert.Less(t, elapsed, 5*time.Second, "Close() must return on its own deadline")
	case <-time.After(10 * time.Second):
		t.Fatal("Close() blocked indefinitely on a silent BMC")
	}
}

// TestClose_BoundedWhenScrapeContextCancelled pins the interaction between the two mechanisms:
// the detached context is what lets the delete run at all after the scrape is cancelled, and
// the deadline is what stops that detached request from running forever.
func TestClose_BoundedWhenScrapeContextCancelled(t *testing.T) {
	bmc := newSessionBMC(t)
	cfg := testRedfishConfig()
	cfg.LogoutTimeout = 250 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	rc := newBMCCollectorWithConfig(t, ctx, bmc, cfg)
	bmc.stallHandlers(t)
	cancel()

	start := time.Now()
	rc.Close()
	elapsed := time.Since(start)

	assert.Greater(t, elapsed, 100*time.Millisecond,
		"a cancelled scrape context must not short-circuit the delete; it should have been attempted")
	assert.Less(t, elapsed, 5*time.Second, "and it must still be bounded by LogoutTimeout")

	// Not an exact count: a refused teardown is retried, and how many of those retries reach
	// the network before the shared fallback budget runs out is a matter of timing. What this
	// test is about is that the cancelled scrape context did not stop the delete being sent.
	_, _, deleted := bmc.counts()
	assert.GreaterOrEqual(t, deleted, 1, "the BMC should have received the DELETE")
}

// TestNewRedfishCollector_ZeroTimeoutsFallBackToBounded confirms a config predating these
// fields cannot mean "no limit". A zero must resolve to the package default, not to infinity.
func TestNewRedfishCollector_ZeroTimeoutsFallBackToBounded(t *testing.T) {
	assert.Positive(t, config.DefaultRedfishConfig.LogoutTimeout)
	assert.Positive(t, config.DefaultRedfishConfig.ResponseHeaderTimeout)

	bmc := newSessionBMC(t)
	rc := newBMCCollectorWithConfig(t, context.Background(), bmc, config.RedfishClientConfig{
		MaxConcurrentRequests: 1,
		DialTimeout:           2 * time.Second,
	})

	assert.Equal(t, config.DefaultRedfishConfig.LogoutTimeout, rc.logoutTimeout)

	rc.Close()
	openSlots, _, _ := bmc.counts()
	assert.Equal(t, 0, openSlots)
}

// TestResponseHeaderTimeout_BoundsASilentBMC covers the other bound. The scrape context is
// normally what limits a data request, but it only fires when Prometheus gives up or the
// connection drops; a BMC that completes TCP and TLS and then says nothing needs a limit at
// the transport. Construction is the cheapest place to observe it, since the service-root GET
// is the first request the client makes.
func TestResponseHeaderTimeout_BoundsASilentBMC(t *testing.T) {
	bmc := newSessionBMC(t)
	bmc.stallHandlers(t)

	cfg := testRedfishConfig()
	cfg.ResponseHeaderTimeout = 250 * time.Millisecond

	start := time.Now()
	_, err := NewRedfishCollector(context.Background(), NewTestLogger(t, 0), bmc.host(t),
		"user", "pass", cfg)
	elapsed := time.Since(start)

	require.Error(t, err, "a BMC that never sends headers must not be waited on forever")
	assert.Less(t, elapsed, 5*time.Second, "the transport bound should have fired")

	// Assert which bound fired. Without this the test would also pass on a dial or TLS
	// failure, which would not demonstrate anything about ResponseHeaderTimeout.
	var netErr net.Error
	require.ErrorAs(t, err, &netErr)
	assert.True(t, netErr.Timeout(), "expected a timeout, got %v", err)
	assert.Contains(t, err.Error(), "timeout awaiting response headers",
		"expected the transport's response-header bound specifically, got %v", err)
}

// logoutCount reads the counter for one target and outcome. Each test gets its own fake BMC
// on a fresh port, so the target label isolates tests from each other on this package-level
// metric.
func logoutCount(t *testing.T, target, result string) float64 {
	t.Helper()
	return testutil.ToFloat64(sessionLogoutsTotal.WithLabelValues(target, result))
}

// TestClose_CountsSuccessfulTeardown gives the failure rate below a denominator; an
// abandoned-session count is not interpretable without the total.
func TestClose_CountsSuccessfulTeardown(t *testing.T) {
	bmc := newSessionBMC(t)
	target := bmc.host(t)
	rc := newBMCCollector(t, context.Background(), bmc)

	before := logoutCount(t, target, resultSuccess)
	rc.Close()

	assert.Equal(t, before+1, logoutCount(t, target, resultSuccess))
	assert.Equal(t, float64(0), logoutCount(t, target, resultFailure))
}

// constMetricValue drains ch and returns the value of the metric whose descriptor names
// fqName, and whether it was emitted at all. Absence is meaningful for redfish_sessions_open:
// it is simply not reported when the BMC declines to list its sessions.
func constMetricValue(t *testing.T, ch chan prometheus.Metric, fqName string) (float64, bool) {
	t.Helper()
	want := fmt.Sprintf("fqName: %q", fqName)
	for {
		select {
		case m := <-ch:
			if !strings.Contains(m.Desc().String(), want) {
				continue
			}
			var pb dto.Metric
			require.NoError(t, m.Write(&pb))
			require.NotNil(t, pb.Gauge, "%s should be a gauge", fqName)
			return pb.Gauge.GetValue(), true
		default:
			return 0, false
		}
	}
}

// TestCollect_ReportsOpenSessionCount is the point of the metric: it counts what the device
// holds, not what this process opened. The second collector stands in for a slot held by
// anything else — an earlier scrape that leaked, a previous build, another client — and that
// slot consumes the cap just the same, so a count that missed it would miss the accumulation
// the metric exists to show.
func TestCollect_ReportsOpenSessionCount(t *testing.T) {
	bmc := newSessionBMC(t)

	other := newBMCCollector(t, context.Background(), bmc)
	defer other.Close()

	rc := newBMCCollector(t, context.Background(), bmc)
	defer rc.Close()

	openSlots, _, _ := bmc.counts()
	require.Equal(t, 2, openSlots)

	ch := make(chan prometheus.Metric, 32)
	rc.Collect(ch)

	open, found := constMetricValue(t, ch, "redfish_sessions_open")
	require.True(t, found, "redfish_sessions_open should have been emitted")
	assert.Equal(t, float64(2), open, "both open slots must be counted, not just our own")
}

// TestCollect_OmitsOpenSessionCountWhenRefused covers the live case that prompted this metric:
// an account behind a forced password change gets 403 on the session collection. That must not
// fail the scrape — the metric goes absent, which a query can see, and everything else in the
// scrape still reports.
func TestCollect_OmitsOpenSessionCountWhenRefused(t *testing.T) {
	bmc := newSessionBMC(t)
	bmc.mu.Lock()
	bmc.refuseSessionList = true
	bmc.mu.Unlock()

	rc := newBMCCollector(t, context.Background(), bmc)
	defer rc.Close()

	ch := make(chan prometheus.Metric, 32)
	assert.NotPanics(t, func() { rc.Collect(ch) })

	_, found := constMetricValue(t, ch, "redfish_sessions_open")
	assert.False(t, found, "a refused session listing must omit the metric, not report a wrong number")

	succeeded, failed := rc.CollectorOutcome()
	assert.Equal(t, int64(0), succeeded)
	assert.Equal(t, int64(0), failed, "the refusal is not a collector failure")
}

// TestSessionsCollectionOf pins the derivation the count depends on. The collection is the
// parent of our own session's URI, which is more reliable than a hardcoded path — but only
// while degenerate inputs stay degenerate rather than becoming "." or "/".
func TestSessionsCollectionOf(t *testing.T) {
	assert.Equal(t, "/redfish/v1/SessionService/Sessions",
		sessionsCollectionOf("/redfish/v1/SessionService/Sessions/42"))
	assert.Equal(t, "", sessionsCollectionOf(""))
	assert.Equal(t, "", sessionsCollectionOf("42"))
	assert.Equal(t, "", sessionsCollectionOf("/42"))
}

// TestClose_CountsRefusedTeardown is the case gofish's Logout() hid entirely: it discards the
// error from DeleteSession, so a BMC refusing the delete looked exactly like success. That
// refusal is the interesting event, because the slot stays occupied.
func TestClose_CountsRefusedTeardown(t *testing.T) {
	bmc := newSessionBMC(t)
	bmc.mu.Lock()
	bmc.refuseDeletes = true
	bmc.mu.Unlock()

	target := bmc.host(t)
	rc := newBMCCollector(t, context.Background(), bmc)

	before := logoutCount(t, target, resultFailure)
	rc.Close()

	assert.Equal(t, before+1, logoutCount(t, target, resultFailure),
		"a refused teardown must be attributed to the target BMC")
	assert.Equal(t, float64(0), logoutCount(t, target, resultSuccess))
	assert.Equal(t, float64(0), logoutCount(t, target, resultRecovered),
		"nothing recovered the slot, so the outcome is a plain failure")

	openSlots, _, deleted := bmc.counts()
	assert.Equal(t, 3, deleted, "gofish's delete plus both direct attempts were made")
	assert.Equal(t, 1, openSlots, "and the BMC kept the slot")
}

// TestClose_CountsTimedOutTeardown checks the bound from Change 2 is also reported, rather
// than being silently dropped as a non-event.
func TestClose_CountsTimedOutTeardown(t *testing.T) {
	bmc := newSessionBMC(t)
	cfg := testRedfishConfig()
	cfg.LogoutTimeout = 250 * time.Millisecond

	target := bmc.host(t)
	rc := newBMCCollectorWithConfig(t, context.Background(), bmc, cfg)
	bmc.stallHandlers(t)

	before := logoutCount(t, target, resultFailure)
	rc.Close()

	assert.Equal(t, before+1, logoutCount(t, target, resultFailure))
}

// TestClose_DirectDeleteRecoversTransientRefusal is the plainest reason to retry: a 503 says
// the BMC would not release the slot now, not that it cannot. gofish's delete is a single
// attempt, so before this the slot was written off on the first refusal.
func TestClose_DirectDeleteRecoversTransientRefusal(t *testing.T) {
	bmc := newSessionBMC(t)
	bmc.mu.Lock()
	bmc.refuseDeleteAttempts = 1
	bmc.mu.Unlock()

	target := bmc.host(t)
	rc := newBMCCollector(t, context.Background(), bmc)

	rc.Close()

	openSlots, _, deleted := bmc.counts()
	assert.Equal(t, 0, openSlots, "the retry must return the slot to the BMC")
	assert.Equal(t, 2, deleted, "the refused delete, then the direct one")

	assert.Equal(t, float64(1), logoutCount(t, target, resultRecovered),
		"a slot the fallback rescued is neither a clean success nor a leak")
	assert.Equal(t, float64(0), logoutCount(t, target, resultFailure))
	assert.Equal(t, float64(0), logoutCount(t, target, resultSuccess))
}

// TestClose_DirectDeleteRecoversRejectedToken is the failure gofish cannot recover from at
// all. A BMC that has invalidated the session token while still holding its slot answers 401
// to the only request gofish can make, and the slot is releasable the whole time — by asking
// again with credentials, which is why the fallback issues the DELETE itself.
func TestClose_DirectDeleteRecoversRejectedToken(t *testing.T) {
	bmc := newSessionBMC(t)
	bmc.mu.Lock()
	bmc.refuseTokenDeletes = true
	bmc.mu.Unlock()

	target := bmc.host(t)
	rc := newBMCCollector(t, context.Background(), bmc)

	rc.Close()

	openSlots, _, _ := bmc.counts()
	assert.Equal(t, 0, openSlots, "the credentialed delete must return the slot")
	assert.Equal(t, []string{"token", "token", "basic"}, bmc.deleteAuth(),
		"the fallback should escalate from the token to credentials, not repeat the token forever")

	assert.Equal(t, float64(1), logoutCount(t, target, resultRecovered))
	assert.Equal(t, float64(0), logoutCount(t, target, resultFailure))
}

// TestClose_NoDirectDeleteOnSuccess keeps the fallback off the ordinary path. A second DELETE
// after a delete the BMC accepted would target a slot it may have already reissued.
func TestClose_NoDirectDeleteOnSuccess(t *testing.T) {
	bmc := newSessionBMC(t)
	rc := newBMCCollector(t, context.Background(), bmc)

	rc.Close()

	_, _, deleted := bmc.counts()
	assert.Equal(t, 1, deleted, "a confirmed teardown must not be retried")
}

// TestClose_CountsNothingWithoutASession keeps every recorded value meaningful: no client
// means no session was ever created, so there is no teardown to report either way.
func TestClose_CountsNothingWithoutASession(t *testing.T) {
	rc := newTestRedfishCollector(nil)
	rc.host = "no-client.test"

	rc.Close()

	assert.Equal(t, float64(0), logoutCount(t, "no-client.test", resultSuccess))
	assert.Equal(t, float64(0), logoutCount(t, "no-client.test", resultFailure))
}

// TestSessionIDOf covers the reduction from the session URI gofish reports to the identifier
// the BMC's own session list shows, which is the value that ends up in a metric label.
func TestSessionIDOf(t *testing.T) {
	assert.Equal(t, "42", sessionIDOf("/redfish/v1/SessionService/Sessions/42"))
	assert.Equal(t, "42", sessionIDOf("42"))
	// An unset URI must not become a "." label, which path.Base would otherwise produce.
	assert.Equal(t, "", sessionIDOf(""))
}

// TestSessionMetrics_Registers guards against a malformed or duplicate metric definition,
// since registration in main() is fatal.
func TestSessionMetrics_Registers(t *testing.T) {
	require.NotEmpty(t, SessionMetrics())
	reg := prometheus.NewRegistry()
	for _, c := range SessionMetrics() {
		assert.NoError(t, reg.Register(c))
	}
}

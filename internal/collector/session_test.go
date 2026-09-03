package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
)

// fakeBMC is a Redfish service root that implements the session service and, crucially,
// keeps a count of how many sessions are currently open. The BMCs this exporter talks to
// cap concurrent sessions and refuse every new session once the table is full, so the
// assertion that matters for session-lifecycle tests is "did the slot come back".
type fakeBMC struct {
	*httptest.Server

	mu       sync.Mutex
	open     map[string]bool
	created  int
	deleted  int
	deleteOK bool

	// blockDeletes, when non-nil, holds every DELETE until the channel is closed. It
	// reproduces a BMC that completes TCP and TLS and then declines to answer.
	blockDeletes chan struct{}
}

func newFakeBMC(t *testing.T) *fakeBMC {
	t.Helper()

	b := &fakeBMC{
		open:     make(map[string]bool),
		deleteOK: true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/redfish/v1/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"@odata.id":      "/redfish/v1/",
			"@odata.type":    "#ServiceRoot.v1_15_0.ServiceRoot",
			"Id":             "RootService",
			"Name":           "Root Service",
			"RedfishVersion": "1.15.0",
			"Links": map[string]any{
				"Sessions": map[string]any{"@odata.id": "/redfish/v1/SessionService/Sessions"},
			},
		}))
	})

	mux.HandleFunc("/redfish/v1/SessionService/Sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		b.mu.Lock()
		b.created++
		id := fmt.Sprintf("%d", b.created)
		sessionURI := "/redfish/v1/SessionService/Sessions/" + id
		b.open[sessionURI] = true
		b.mu.Unlock()

		w.Header().Set("Location", sessionURI)
		w.Header().Set("X-Auth-Token", "token-"+id)
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("/redfish/v1/SessionService/Sessions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if blk := b.blocker(); blk != nil {
			<-blk
		}
		b.mu.Lock()
		b.deleted++
		ok := b.deleteOK
		if ok {
			delete(b.open, r.URL.Path)
		}
		b.mu.Unlock()

		if !ok {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// TLS, so tests drive the real newRedfishClient path (which builds an https:// endpoint
	// and relies on the client's Insecure setting for the BMCs' self-signed certs).
	b.Server = httptest.NewTLSServer(mux)
	t.Cleanup(b.Close)
	return b
}

func (b *fakeBMC) blocker() chan struct{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.blockDeletes
}

// host returns the "host:port" form the exporter's config uses as a target.
func (b *fakeBMC) host(t *testing.T) string {
	t.Helper()
	u, err := url.Parse(b.URL)
	require.NoError(t, err)
	return u.Host
}

func (b *fakeBMC) counts() (openSessions, created, deleted int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.open), b.created, b.deleted
}

// newBMCCollector builds a redfishCollector against a fakeBMC using the exporter's own
// client constructor, so the transport settings and auth mode under test are the real ones.
func newBMCCollector(t *testing.T, ctx context.Context, b *fakeBMC, rfConfig config.RedfishClientConfig) *redfishCollector {
	t.Helper()

	host := b.host(t)
	client, err := newRedfishClient(ctx, host, "user", "pass", rfConfig)
	require.NoError(t, err)

	logoutTimeout := rfConfig.LogoutTimeout
	if logoutTimeout <= 0 {
		logoutTimeout = config.DefaultRedfishConfig.LogoutTimeout
	}

	return &redfishCollector{
		ctx:           ctx,
		logger:        NewTestLogger(t, 0),
		redfishClient: client,
		host:          host,
		logoutTimeout: logoutTimeout,
		redfishUp:     prometheus.NewGauge(prometheus.GaugeOpts{Name: "redfish_up_test"}),
	}
}

func testRedfishConfig() config.RedfishClientConfig {
	return config.RedfishClientConfig{
		MaxConcurrentRequests: 1,
		DialTimeout:           2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
		LogoutTimeout:         2 * time.Second,
	}
}

// TestClose_ReleasesSessionWhenScrapeContextCancelled covers the leak this work targets.
// The session is opened against the inbound HTTP request's context, so when Prometheus
// gives up on a slow scrape that context is cancelled before collection starts. Collect()
// then declines to do any work, and before Close() existed nothing deleted the session:
// the slot stayed occupied on the BMC until its own idle timeout reclaimed it.
func TestClose_ReleasesSessionWhenScrapeContextCancelled(t *testing.T) {
	bmc := newFakeBMC(t)
	ctx, cancel := context.WithCancel(context.Background())

	rc := newBMCCollector(t, ctx, bmc, testRedfishConfig())

	openSessions, created, _ := bmc.counts()
	require.Equal(t, 1, created, "constructing the collector should open exactly one session")
	require.Equal(t, 1, openSessions)

	// The scrape context dies before collection begins.
	cancel()

	ch := make(chan prometheus.Metric, 32)
	rc.Collect(ch)

	// Collect() took the early-return branch, so it did not release anything.
	openSessions, _, deleted := bmc.counts()
	require.Equal(t, 0, deleted, "Collect() no longer owns session teardown")
	require.Equal(t, 1, openSessions)

	rc.Close()

	openSessions, created, deleted = bmc.counts()
	assert.Equal(t, 0, openSessions, "the session slot must be returned to the BMC")
	assert.Equal(t, 1, created)
	assert.Equal(t, 1, deleted)
}

// TestClose_ReleasesSessionAfterSuccessfulCollect verifies the ordinary path still tears the
// session down now that teardown moved out of Collect() and into Close().
func TestClose_ReleasesSessionAfterSuccessfulCollect(t *testing.T) {
	bmc := newFakeBMC(t)
	rc := newBMCCollector(t, context.Background(), bmc, testRedfishConfig())
	rc.collectors = []ContextAwareCollector{&stubCollector{}}

	ch := make(chan prometheus.Metric, 32)
	rc.Collect(ch)
	rc.Close()

	openSessions, created, deleted := bmc.counts()
	assert.Equal(t, 0, openSessions)
	assert.Equal(t, 1, created)
	assert.Equal(t, 1, deleted)
}

// TestClose_Idempotent matters because Close() is deferred by the handler while the
// collector may be closed elsewhere; a second DELETE against a released session would
// either error noisily or, on some BMCs, delete a slot that has since been reissued.
func TestClose_Idempotent(t *testing.T) {
	bmc := newFakeBMC(t)
	rc := newBMCCollector(t, context.Background(), bmc, testRedfishConfig())

	rc.Close()
	rc.Close()
	rc.Close()

	openSessions, _, deleted := bmc.counts()
	assert.Equal(t, 0, openSessions)
	assert.Equal(t, 1, deleted, "only the first Close() should issue a DELETE")
}

// TestClose_NoClient guards the path where the collector never got a working client, so
// that a deferred Close() cannot turn a failed scrape into a panic.
func TestClose_NoClient(t *testing.T) {
	rc := newTestRedfishCollector(nil)
	rc.host = "no-client.invalid"
	rc.logoutTimeout = time.Second

	assert.NotPanics(t, rc.Close)
}

// TestClose_SkipsWhenNoSessionExists covers basic-auth clients, which authenticate per
// request and never hold a session. There is nothing to delete and Close() must not
// attempt a DELETE against an empty session URL.
func TestClose_SkipsWhenNoSessionExists(t *testing.T) {
	bmc := newFakeBMC(t)
	rfConfig := testRedfishConfig()
	rfConfig.BasicAuth = true

	rc := newBMCCollector(t, context.Background(), bmc, rfConfig)

	openSessions, created, _ := bmc.counts()
	require.Equal(t, 0, created, "basic auth must not create a session")
	require.Equal(t, 0, openSessions)

	rc.Close()

	_, _, deleted := bmc.counts()
	assert.Equal(t, 0, deleted, "there is no session to delete")
}

// TestBasicAuth_ConsumesNoSessionSlots is the direct statement of why basic auth is worth
// having: a BMC pinned at its session cap refuses new sessions, and a client that never
// opens one is immune to that. Several scrapes in a row must leave the session table empty.
func TestBasicAuth_ConsumesNoSessionSlots(t *testing.T) {
	bmc := newFakeBMC(t)
	rfConfig := testRedfishConfig()
	rfConfig.BasicAuth = true

	for range 5 {
		rc := newBMCCollector(t, context.Background(), bmc, rfConfig)
		ch := make(chan prometheus.Metric, 32)
		rc.Collect(ch)
		rc.Close()
	}

	openSessions, created, deleted := bmc.counts()
	assert.Equal(t, 0, created, "no sessions should ever be created under basic auth")
	assert.Equal(t, 0, openSessions)
	assert.Equal(t, 0, deleted)
}

// TestSessionAuth_ConsumesOneSlotPerScrape documents the churn that basic auth removes:
// with session auth every scrape opens and closes a slot, and each of those is a chance to
// leak if the teardown is skipped or fails.
func TestSessionAuth_ConsumesOneSlotPerScrape(t *testing.T) {
	bmc := newFakeBMC(t)

	for range 5 {
		rc := newBMCCollector(t, context.Background(), bmc, testRedfishConfig())
		ch := make(chan prometheus.Metric, 32)
		rc.Collect(ch)
		rc.Close()
	}

	openSessions, created, deleted := bmc.counts()
	assert.Equal(t, 5, created)
	assert.Equal(t, 5, deleted)
	assert.Equal(t, 0, openSessions, "every slot should come back")
}

// TestClose_BoundedBySilentBMC is the second half of the leak. Once the teardown runs on a
// context detached from the cancelled scrape context, nothing else bounds it: the HTTP
// client has no overall timeout, so a BMC that accepts the DELETE and never answers would
// park the handler goroutine indefinitely — one stuck goroutine and one held session per
// scrape, growing without limit. Close() must return on its own timeout instead.
func TestClose_BoundedBySilentBMC(t *testing.T) {
	bmc := newFakeBMC(t)
	block := make(chan struct{})
	bmc.mu.Lock()
	bmc.blockDeletes = block
	bmc.mu.Unlock()
	// Release the parked handler so the test server can shut down.
	t.Cleanup(func() { close(block) })

	rfConfig := testRedfishConfig()
	rfConfig.LogoutTimeout = 250 * time.Millisecond

	rc := newBMCCollector(t, context.Background(), bmc, rfConfig)

	done := make(chan time.Duration, 1)
	go func() {
		start := time.Now()
		rc.Close()
		done <- time.Since(start)
	}()

	select {
	case elapsed := <-done:
		assert.Less(t, elapsed, 5*time.Second, "Close() must be bounded by logoutTimeout")
	case <-time.After(10 * time.Second):
		t.Fatal("Close() blocked indefinitely on a silent BMC")
	}
}

// TestClose_RecordsAbandonedSession verifies a failed teardown is counted against the BMC
// address. The pre-existing HTTP-level DELETE metric labels the exporter instance, so it
// cannot answer "which device is leaking", which is the only form of the question that
// helps when a device refuses all new sessions.
func TestClose_RecordsAbandonedSession(t *testing.T) {
	bmc := newFakeBMC(t)
	bmc.mu.Lock()
	bmc.deleteOK = false
	bmc.mu.Unlock()

	rc := newBMCCollector(t, context.Background(), bmc, testRedfishConfig())
	target := bmc.host(t)

	before := testutil.ToFloat64(sessionsAbandonedTotal.WithLabelValues(target))
	rc.Close()
	after := testutil.ToFloat64(sessionsAbandonedTotal.WithLabelValues(target))

	assert.Equal(t, float64(1), after-before, "a failed teardown must be attributed to the target")

	openSessions, _, deleted := bmc.counts()
	assert.Equal(t, 1, deleted, "the DELETE was attempted")
	assert.Equal(t, 1, openSessions, "and the BMC kept the slot")
}

// TestClose_RecordsSuccessfulLogout checks the success counter, since an abandoned-session
// rate is only interpretable next to the total.
func TestClose_RecordsSuccessfulLogout(t *testing.T) {
	bmc := newFakeBMC(t)
	rc := newBMCCollector(t, context.Background(), bmc, testRedfishConfig())
	target := bmc.host(t)

	before := testutil.ToFloat64(sessionLogoutsTotal.WithLabelValues(target, "success"))
	rc.Close()
	after := testutil.ToFloat64(sessionLogoutsTotal.WithLabelValues(target, "success"))

	assert.Equal(t, float64(1), after-before)
}

// TestSessionMetrics_Registerable guards against a duplicate or malformed metric definition
// reaching main(), where registration is fatal.
func TestSessionMetrics_Registerable(t *testing.T) {
	reg := prometheus.NewRegistry()
	assert.NotEmpty(t, SessionMetrics())
	assert.NoError(t, reg.Register(prometheus.NewGauge(prometheus.GaugeOpts{Name: "placeholder"})))
	for _, c := range SessionMetrics() {
		assert.NoError(t, reg.Register(c))
	}
}

// TestRedfishClientDefaults_BoundUnsetTimeouts confirms a config that predates the new
// timeout fields still gets bounded behaviour rather than an unlimited wait.
func TestRedfishClientDefaults_BoundUnsetTimeouts(t *testing.T) {
	assert.Positive(t, config.DefaultRedfishConfig.ResponseHeaderTimeout)
	assert.Positive(t, config.DefaultRedfishConfig.LogoutTimeout)

	bmc := newFakeBMC(t)
	// Zero timeouts, as an older config file would produce.
	rc := newBMCCollector(t, context.Background(), bmc, config.RedfishClientConfig{
		MaxConcurrentRequests: 1,
		DialTimeout:           2 * time.Second,
	})
	assert.Equal(t, config.DefaultRedfishConfig.LogoutTimeout, rc.logoutTimeout)

	rc.Close()
	openSessions, _, _ := bmc.counts()
	assert.Equal(t, 0, openSessions)
}

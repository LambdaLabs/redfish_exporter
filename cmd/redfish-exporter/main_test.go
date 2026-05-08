package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stmcginnis/gofish"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLogLevel(t *testing.T) {
	testCases := []struct {
		level    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo}, // default level
	}

	for _, tc := range testCases {
		actual := parseLogLevel(tc.level)
		assert.Equal(t, tc.expected, actual, fmt.Sprintf("Unexpected log level parsed for infot %s", tc.level))
	}
}

// newTestGofishClient creates a gofish APIClient backed by a minimal httptest server.
// The server returns valid-enough JSON for gofish to connect and for collectors to be
// constructed (no real Redfish calls happen at construction time).
func newTestGofishClient(t *testing.T) *gofish.APIClient {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"@odata.id": r.URL.Path,
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(func() {
		server.Close()
	})

	client, err := gofish.ConnectContext(context.Background(), gofish.ClientConfig{
		Endpoint:   server.URL,
		HTTPClient: server.Client(),
		Insecure:   true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		client.Logout()
	})

	return client
}

// blockUntilCtxDone is a server function that blocks until ctx is cancelled,
// then returns http.ErrServerClosed — mirroring web.ListenAndServe behaviour
// when it handles the signal via its own internal handler.
func blockUntilCtxDone(ctx context.Context) error {
	<-ctx.Done()
	return http.ErrServerClosed
}

// listenAndReturnOnSignal is a server function that registers its own signal
// handler (simulating web.ListenAndServe's internal signal handling) and
// returns ErrServerClosed as soon as the signal fires, potentially before
// signal.NotifyContext's ctx.Done() is processed.
func listenAndReturnOnSignal(sig syscall.Signal) func(context.Context) error {
	return func(_ context.Context) error {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, sig)
		defer signal.Stop(ch)
		<-ch
		return http.ErrServerClosed
	}
}

// TestRunServer covers the behaviour of runServer across no-signal and
// signal-triggered scenarios.
func TestRunServer(t *testing.T) {
	ptr := func(d time.Duration) *time.Duration { return &d }
	sigptr := func(s syscall.Signal) *syscall.Signal { return &s }

	// noShutdownFactory returns a shutdown func that must NOT be called, plus a no-op check.
	noShutdownFactory := func(t *testing.T) ([]func(context.Context) error, func()) {
		t.Helper()
		f := func(_ context.Context) error {
			t.Error("shutdown func must not be called")
			return nil
		}
		return []func(context.Context) error{f}, func() {}
	}

	// trackShutdownFactory returns a shutdown func that records whether it was called,
	// plus a check that asserts it was called iff wantShutdown is true.
	trackShutdownFactory := func(t *testing.T, wantShutdown bool) ([]func(context.Context) error, func()) {
		t.Helper()
		shutdownCalled := false
		f := func(_ context.Context) error { shutdownCalled = true; return nil }
		check := func() {
			assert.Equal(t, wantShutdown, shutdownCalled, "unexpected shutdown func invocation state")
		}
		return []func(context.Context) error{f}, check
	}

	// deadlineFactory returns a shutdown func that captures deadline info,
	// plus a check that asserts the deadline is approximately wantDeadline away.
	deadlineFactory := func(t *testing.T, wantDeadline time.Duration) ([]func(context.Context) error, func()) {
		t.Helper()
		var deadlineSet bool
		var remaining time.Duration
		f := func(ctx context.Context) error {
			dl, ok := ctx.Deadline()
			deadlineSet = ok
			if ok {
				remaining = time.Until(dl)
			}
			return nil
		}
		check := func() {
			assert.True(t, deadlineSet, "shutdown context should have a deadline")
			assert.InDelta(t, wantDeadline.Seconds(), remaining.Seconds(), 0.5,
				"shutdown context deadline should be ~%s from now", wantDeadline)
		}
		return []func(context.Context) error{f}, check
	}

	tests := []struct {
		name          string
		serverFunc    func(context.Context) error
		shutdownFuncs func(t *testing.T) ([]func(context.Context) error, func())
		sendSig       *syscall.Signal
		wantErr       string
		wantShutdown  bool
		wantDeadline  *time.Duration
	}{
		{
			name:          "server error is returned without calling shutdown funcs",
			serverFunc:    func(_ context.Context) error { return errors.New("boom") },
			shutdownFuncs: noShutdownFactory,
			sendSig:       nil,
			wantErr:       "boom",
			wantShutdown:  false,
			wantDeadline:  nil,
		},
		{
			name:          "clean nil exit does not call shutdown funcs",
			serverFunc:    func(_ context.Context) error { return nil },
			shutdownFuncs: noShutdownFactory,
			sendSig:       nil,
			wantErr:       "",
			wantShutdown:  false,
			wantDeadline:  nil,
		},
		{
			name:          "ErrServerClosed without signal does not call shutdown funcs",
			serverFunc:    func(_ context.Context) error { return http.ErrServerClosed },
			shutdownFuncs: noShutdownFactory,
			sendSig:       nil,
			wantErr:       "",
			wantShutdown:  false,
			wantDeadline:  nil,
		},
		{
			name:       "SIGTERM via ctx.Done path",
			serverFunc: blockUntilCtxDone,
			shutdownFuncs: func(t *testing.T) ([]func(context.Context) error, func()) {
				return trackShutdownFactory(t, true)
			},
			sendSig:      sigptr(syscall.SIGTERM),
			wantErr:      "",
			wantShutdown: true,
			wantDeadline: nil,
		},
		{
			name:       "SIGINT via ctx.Done path",
			serverFunc: blockUntilCtxDone,
			shutdownFuncs: func(t *testing.T) ([]func(context.Context) error, func()) {
				return trackShutdownFactory(t, true)
			},
			sendSig:      sigptr(syscall.SIGINT),
			wantErr:      "",
			wantShutdown: true,
			wantDeadline: nil,
		},
		{
			name:       "SIGTERM via serverErr path (web.ListenAndServe behaviour)",
			serverFunc: listenAndReturnOnSignal(syscall.SIGTERM),
			shutdownFuncs: func(t *testing.T) ([]func(context.Context) error, func()) {
				return trackShutdownFactory(t, true)
			},
			sendSig:      sigptr(syscall.SIGTERM),
			wantErr:      "",
			wantShutdown: true,
			wantDeadline: nil,
		},
		{
			name:       "shutdown funcs receive context with configured deadline",
			serverFunc: blockUntilCtxDone,
			shutdownFuncs: func(t *testing.T) ([]func(context.Context) error, func()) {
				return deadlineFactory(t, 3*time.Second)
			},
			sendSig:      sigptr(syscall.SIGTERM),
			wantErr:      "",
			wantShutdown: true,
			wantDeadline: ptr(3 * time.Second),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			shutdownFuncs, check := tc.shutdownFuncs(t)

			const timeout = 5 * time.Second
			wantTimeout := timeout
			if tc.wantDeadline != nil {
				wantTimeout = *tc.wantDeadline
			}

			var err error
			if tc.sendSig != nil {
				done := make(chan error, 1)
				go func() {
					done <- runServer(tc.serverFunc, shutdownFuncs, wantTimeout)
				}()
				time.Sleep(50 * time.Millisecond)
				require.NoError(t, syscall.Kill(os.Getpid(), *tc.sendSig))
				select {
				case err = <-done:
				case <-time.After(10 * time.Second):
					t.Fatalf("runServer did not return within timeout after %s", *tc.sendSig)
				}
			} else {
				err = runServer(tc.serverFunc, shutdownFuncs, wantTimeout)
			}

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}

			check()
		})
	}
}

// newScrapeRequest builds an *http.Request with a scrapeRequest injected into its context,
// pointing at the given target. hostConfig may be nil to use empty credentials.
func newScrapeRequest(target string, hostConfig *config.HostConfig) *http.Request {
	if hostConfig == nil {
		hostConfig = &config.HostConfig{}
	}
	sr := &scrapeRequest{
		Target:     target,
		Modules:    []string{"rf_exporter_default"},
		HostConfig: hostConfig,
	}
	r := httptest.NewRequest(http.MethodGet, "/redfish?target="+target, nil)
	ctx := context.WithValue(r.Context(), scrapeRequestCtxKey, sr)
	return r.WithContext(ctx)
}

// TestScrapeRequestsTotal verifies that redfish_exporter_scrape_requests_total is incremented
// correctly across different scrape outcomes. The counter is intended to always reflect the
// number of scrape attempts per target, regardless of whether the scrape succeeded or failed,
// so that operators can detect missing or failing endpoints even when no other metrics are emitted.
func TestScrapeRequestsTotal(t *testing.T) {
	// Use a minimal safeConfig so metricsHandler can call RedfishClientConfig().
	safeConfig = config.NewSafeConfig("")

	// Each sub-test uses a unique target address so its counter label starts at zero,
	// allowing absolute value assertions without resetting shared state.

	t.Run("increments on failed connection", func(t *testing.T) {
		// Verifies the counter is incremented even when the Redfish connection fails entirely.
		// This is the primary motivation for the metric: scrape attempts that produce no metrics
		// at all (e.g. host unreachable, TLS error) must still be counted so gaps in other
		// metrics can be attributed to collection failure rather than a healthy absence of data.
		target := "unreachable-failed.test:1"
		before := testutil.ToFloat64(scrapeCountersFor(target).requests)

		w := httptest.NewRecorder()
		metricsHandler()(w, newScrapeRequest(target, nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, before+1, testutil.ToFloat64(scrapeCountersFor(target).requests))
	})

	t.Run("increments on successful scrape", func(t *testing.T) {
		// Verifies the counter is also incremented on the happy path, where the Redfish
		// endpoint is reachable and the handler proceeds to collect metrics.
		// newRedfishClient always dials https://, so a TLS server is required here.
		// Insecure: true is hardcoded in newRedfishClient, so the self-signed test cert is accepted.
		rfServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"@odata.id": r.URL.Path})
		}))
		t.Cleanup(rfServer.Close)

		target := rfServer.Listener.Addr().String()
		before := testutil.ToFloat64(scrapeCountersFor(target).requests)

		w := httptest.NewRecorder()
		metricsHandler()(w, newScrapeRequest(target, nil))

		assert.Equal(t, before+1, testutil.ToFloat64(scrapeCountersFor(target).requests))
	})

	t.Run("counts are per target", func(t *testing.T) {
		// Verifies that the counter uses the target address as a label, so each host's
		// scrape count is tracked independently and requests to one target do not affect another.
		targetA := "per-target-a.test:1"
		targetB := "per-target-b.test:1"
		beforeA := testutil.ToFloat64(scrapeCountersFor(targetA).requests)
		beforeB := testutil.ToFloat64(scrapeCountersFor(targetB).requests)

		metricsHandler()(httptest.NewRecorder(), newScrapeRequest(targetA, nil))
		metricsHandler()(httptest.NewRecorder(), newScrapeRequest(targetA, nil))
		metricsHandler()(httptest.NewRecorder(), newScrapeRequest(targetB, nil))

		assert.Equal(t, beforeA+2, testutil.ToFloat64(scrapeCountersFor(targetA).requests))
		assert.Equal(t, beforeB+1, testutil.ToFloat64(scrapeCountersFor(targetB).requests))
	})

	t.Run("does not increment when scrape request missing from context", func(t *testing.T) {
		// Verifies the counter is NOT incremented for malformed requests where the scrape
		// request was never injected into the context (i.e. the mustScrapeRequest middleware
		// was bypassed). These are not genuine scrape attempts and should not be counted.
		target := "no-context.test:1"
		before := testutil.ToFloat64(scrapeCountersFor(target).requests)

		r := httptest.NewRequest(http.MethodGet, "/redfish", nil)
		w := httptest.NewRecorder()
		metricsHandler()(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, before, testutil.ToFloat64(scrapeCountersFor(target).requests))
	})
}

// TestScrapeSuccessesTotal verifies that redfish_exporter_scrape_successes_total is incremented
// only when all sub-collectors in the scrape session completed without failure.
func TestScrapeSuccessesTotal(t *testing.T) {
	safeConfig = config.NewSafeConfig("")

	t.Run("increments when all collectors succeed", func(t *testing.T) {
		// A TLS server that responds with minimal valid JSON allows NewRedfishCollector
		// to connect and all (stub) collectors to complete without panicking.
		rfServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"@odata.id": r.URL.Path})
		}))
		t.Cleanup(rfServer.Close)

		target := rfServer.Listener.Addr().String()
		before := testutil.ToFloat64(scrapeCountersFor(target).successes)

		metricsHandler()(httptest.NewRecorder(), newScrapeRequest(target, nil))

		assert.Equal(t, before+1, testutil.ToFloat64(scrapeCountersFor(target).successes))
	})

	t.Run("does not increment when connection fails", func(t *testing.T) {
		// An unreachable target means NewRedfishCollector fails and no collectors run,
		// so CollectorOutcome() returns failed > 0 and the success counter must not increment.
		target := "unreachable-success.test:1"
		before := testutil.ToFloat64(scrapeCountersFor(target).successes)

		metricsHandler()(httptest.NewRecorder(), newScrapeRequest(target, nil))

		assert.Equal(t, before, testutil.ToFloat64(scrapeCountersFor(target).successes))
	})

	t.Run("requests and successes counters are independent", func(t *testing.T) {
		// Verifies that a failed scrape increments requests but not successes,
		// so the difference between the two counters reflects the failure count.
		target := "independent-counters.test:1"
		beforeReqs := testutil.ToFloat64(scrapeCountersFor(target).requests)
		beforeSucc := testutil.ToFloat64(scrapeCountersFor(target).successes)

		metricsHandler()(httptest.NewRecorder(), newScrapeRequest(target, nil))

		assert.Equal(t, beforeReqs+1, testutil.ToFloat64(scrapeCountersFor(target).requests))
		assert.Equal(t, beforeSucc, testutil.ToFloat64(scrapeCountersFor(target).successes))
	})
}

func TestBuildCollectorsFor(t *testing.T) {
	moduleConfig := map[string]config.Module{
		"chassis_collector": {Prober: "chassis_collector"},
		"system_collector":  {Prober: "system_collector"},
	}

	testCases := []struct {
		name         string
		modules      []string
		moduleConfig map[string]config.Module
		expected     int
	}{
		{
			name:         "known modules",
			modules:      []string{"chassis_collector", "system_collector"},
			moduleConfig: moduleConfig,
			expected:     2,
		},
		{
			name:         "unknown module skipped",
			modules:      []string{"nonexistent"},
			moduleConfig: moduleConfig,
			expected:     0,
		},
		{
			name:         "mixed known and unknown",
			modules:      []string{"chassis_collector", "nonexistent"},
			moduleConfig: moduleConfig,
			expected:     1,
		},
		{
			name:         "default bundle",
			modules:      []string{"rf_exporter_default"},
			moduleConfig: moduleConfig, // will be overridden internally
			expected:     5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rfClient := newTestGofishClient(t)
			collectors := buildCollectorsFor(context.Background(), tc.modules, tc.moduleConfig, rfClient, slog.Default())
			assert.Equal(t, tc.expected, len(collectors), "unexpected number of collectors")
		})
	}
}

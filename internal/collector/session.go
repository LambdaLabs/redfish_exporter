package collector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/schemas"
)

// Values of the result label on sessionLogoutsTotal.
const (
	// resultSuccess: the delete gofish issued was accepted, so the slot is back.
	resultSuccess = "success"
	// resultRecovered: that delete failed and the direct DELETE fallback released the slot.
	resultRecovered = "recovered"
	// resultFailure: every attempt failed, so the slot may stay occupied on the BMC.
	resultFailure = "failure"
)

// How one direct DELETE attempt authenticated itself, reported in the auth log field.
const (
	authSessionToken = "x-auth-token"
	authBasic        = "basic"
)

// maxErrorBody caps how much of a refusal body is quoted into a log line.
const maxErrorBody = 1024

// sessionLogoutsTotal counts session teardown attempts. It is labelled by BMC address
// because a session cap is a per-device limit, so "which device is holding sessions we
// failed to release" is the only useful form of the question. The HTTP-level metrics from
// otelhttp cannot answer it: they label the exporter instance, not the target.
//
// It is deliberately NOT labelled by session id. BMCs that mint a random identifier per
// session would make every scrape a new series here, in Prometheus and in this process's own
// memory, since a CounterVec never releases a child. The identifier belongs in the logs,
// where it costs nothing to keep.
//
// Teardown happens after the scrape response has already been gathered and written, so this
// cannot live on the per-scrape registry — it is registered on the default registerer and
// exposed on /metrics.
var sessionLogoutsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: prometheus.BuildFQName(namespace, exporter, "session_logouts_total"),
	Help: "Redfish session teardown attempts by target and outcome. result=success is a delete the BMC accepted, result=recovered means that delete failed and the direct DELETE fallback released the slot, and result=failure means every attempt failed and the session may stay occupied on the BMC until its own idle timeout reclaims it.",
}, []string{"target", "result"})

// SessionMetrics returns the session-lifecycle metrics for the caller to register.
func SessionMetrics() []prometheus.Collector {
	return []prometheus.Collector{sessionLogoutsTotal}
}

// sessionsOpenDesc reports how many sessions the BMC is holding, this scrape's own included.
//
// It is the only session signal that describes the device rather than the exporter. Our own
// counters can only see what this process did: a slot stranded by an earlier scrape, by a
// previous build, or by another client entirely occupies the cap just the same and is
// invisible to them. This is what shows the slot budget being consumed — and, once a fix is
// deployed, what shows it being given back.
//
// Unlike sessionLogoutsTotal it lives on the per-scrape registry, so it carries no labels at
// all: Prometheus attributes it to the target it scraped, and there is nothing else to say.
var sessionsOpenDesc = prometheus.NewDesc(
	prometheus.BuildFQName(namespace, "", "sessions_open"),
	"Sessions currently open on the BMC, including the one this scrape opened.",
	nil, nil,
)

// sessionsCollectionOf derives the session collection URI from the URI of a session in it.
// The Location the BMC returned when it created our session is by definition a member of the
// collection we want to count, which makes its parent a better answer than either a hardcoded
// path or a second discovery request through SessionService.
func sessionsCollectionOf(sessionURI string) string {
	if sessionURI == "" {
		return ""
	}
	dir := path.Dir(sessionURI)
	if dir == "." || dir == "/" {
		return ""
	}
	return dir
}

// openSessionCount asks the BMC how many sessions it is currently holding.
//
// It reads the collection document alone rather than going through gofish's
// SessionService.Sessions(), which fetches every member individually — that would turn one
// request per scrape into N+1 against exactly the devices whose slowness and slot exhaustion
// are the reason for counting in the first place. The cost of this metric is one GET.
func (r *redfishCollector) openSessionCount() (int, error) {
	if r.redfishClient == nil {
		return 0, errors.New("no redfish client")
	}
	session, err := r.redfishClient.GetSession()
	if err != nil {
		return 0, err
	}
	uri := sessionsCollectionOf(session.ID)
	if uri == "" {
		return 0, fmt.Errorf("cannot derive the session collection from session URI %q", session.ID)
	}

	// Tagged like a sub-collector so the request is attributed on the http.client.* metrics
	// rather than showing up as uncategorised traffic.
	api := r.redfishClient.WithContext(ContextWithCollector(r.ctx, "sessions"))
	resp, err := api.Get(uri)
	if err != nil {
		return 0, err
	}
	defer schemas.DeferredCleanupHTTPResponse(resp)

	var collection struct {
		Count   *int `json:"Members@odata.count"`
		Members []struct {
			ODataID string `json:"@odata.id"`
		} `json:"Members"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return 0, err
	}

	// The count annotation is optional in practice; fall back to the array it annotates.
	if collection.Count != nil {
		return *collection.Count, nil
	}
	return len(collection.Members), nil
}

// sessionIDOf reduces the session URI gofish reports — the Location header from session
// creation, e.g. /redfish/v1/SessionService/Sessions/42 — to the identifier the BMC's own
// session list shows, which is what someone reading a log line or a metric label can match
// against the device.
func sessionIDOf(sessionURI string) string {
	if sessionURI == "" {
		return ""
	}
	return path.Base(sessionURI)
}

// sessionLogger attaches the identity of a session to a logger. Both forms are recorded: the
// id is what the BMC's session list and the metric label carry, the URI is what was actually
// requested, and they diverge on BMCs that answer creation with a Location the session
// collection does not repeat.
func sessionLogger(logger *slog.Logger, sessionURI string) *slog.Logger {
	return logger.With(
		slog.String("session_id", sessionIDOf(sessionURI)),
		slog.String("session_uri", sessionURI),
	)
}

// httpStatusOf reports the HTTP status a gofish error carries, since gofish wraps a refusal
// from the BMC in *schemas.Error. It returns 0 when the request failed before any response
// arrived — a dial failure, a cancelled context, a timeout — which is itself the distinction
// worth having in a log line: a 4xx or 5xx means the BMC decided something, a 0 means it never
// answered and the state of the slot is unknown.
func httpStatusOf(err error) int {
	var rfErr *schemas.Error
	if errors.As(err, &rfErr) {
		return rfErr.HTTPReturnedStatusCode
	}
	return 0
}

// Close releases the Redfish session opened by NewRedfishCollector. It is idempotent and
// safe to call on a collector whose Collect() never ran.
//
// Session teardown belongs to whoever caused the session to be created, which is the scrape
// handler, not Collect(). BMCs cap concurrent sessions and refuse every new one once the cap
// is reached, so a session that outlives its scrape denies service to later scrapes until the
// BMC's own idle timeout reclaims the slot. Callers should defer Close() as soon as the
// collector is constructed: a deferred call also runs while a panic unwinds, so it covers the
// paths where Collect() is never reached or does no work.
//
// The delete runs on a context detached from the scrape's. By the time Close() is reached the
// scrape context is frequently already cancelled — that is precisely the case that leaks — and
// a cancelled context cannot carry the request that cleans up after it. Detaching keeps the
// scrape's context values, so the collector attribute on the HTTP metrics survives, while
// dropping its cancellation. The replacement carries its own deadline: gofish's Logout()
// substitutes context.Background() for a cancelled context, which would otherwise leave this
// request with no limit at all and park the handler goroutine on a silent BMC.
func (r *redfishCollector) Close() {
	// A double DELETE would target a slot the BMC may have already reissued, hence the once.
	r.closeOnce.Do(func() {
		if r.redfishClient == nil {
			r.logger.Debug("no redfish client, so no session to release")
			return
		}
		session, err := r.redfishClient.GetSession()
		if err != nil {
			// No session was established — a basic-auth client holds none — so there is
			// nothing to release and no outcome to report.
			r.logger.Debug("no redfish session to release", slog.Any("error", err))
			return
		}

		logger := sessionLogger(r.logger, session.ID)

		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.ctx), r.logoutTimeout)
		defer cancel()

		// Operate on a WithContext copy rather than on the client itself: the copy shares the
		// auth pointer, so the session is still identified correctly, but no field of a client
		// that other goroutines may still be reading gets mutated.
		api := r.redfishClient.WithContext(ctx)
		// gofish's Logout() does this for us; since we bypass it, close the idle connections
		// the keepalive settings would otherwise hold for up to a minute after the scrape.
		defer api.HTTPClient.CloseIdleConnections()

		logger.Debug("deleting redfish session", slog.Duration("timeout", r.logoutTimeout))
		start := time.Now()

		// DeleteSession is called directly rather than through gofish's Logout(), which
		// discards the returned error and so makes a failed teardown indistinguishable from
		// a successful one.
		if err := api.Service.DeleteSession(session.ID); err != nil {
			logger.Error("failed to delete redfish session, falling back to a direct DELETE",
				slog.Duration("elapsed", time.Since(start)),
				slog.Int("status_code", httpStatusOf(err)),
				slog.Any("error", err))

			if r.forceDeleteSession(logger, session) {
				sessionLogoutsTotal.WithLabelValues(r.host, resultRecovered).Inc()
				return
			}

			sessionLogoutsTotal.WithLabelValues(r.host, resultFailure).Inc()
			logger.Error("every attempt to delete the redfish session failed; the slot may stay occupied on the BMC until its own idle timeout reclaims it")
			return
		}

		sessionLogoutsTotal.WithLabelValues(r.host, resultSuccess).Inc()
		logger.Debug("redfish session deleted", slog.Duration("elapsed", time.Since(start)))
	})
}

// forceDeleteSession is the fallback for a teardown the BMC did not confirm: it issues the
// session DELETE itself, once per authentication scheme it has the material for, and reports
// whether any attempt was accepted.
//
// A second request is worth making because the common ways the first delete fails are ones
// that asking again answers. A timeout or a transient 5xx says nothing about whether the slot
// can be released, only that this attempt did not release it. And gofish will only ever
// present the session token, so a BMC that has already invalidated that token answers 401
// while still holding the slot — there the basic-auth attempt is the only one that can
// succeed.
//
// The fallback takes a deadline of its own rather than sharing the first attempt's, because a
// timeout is one of the failures it exists to recover from: on the expired context of the
// attempt that just timed out, every retry would fail before reaching the network. Its own
// attempts do share that one deadline between them, which caps teardown at roughly twice
// logout_timeout on a silent BMC — and costs nothing where escalating actually helps, since a
// BMC that rejects the token answers immediately and leaves the budget intact.
func (r *redfishCollector) forceDeleteSession(logger *slog.Logger, session *gofish.Session) bool {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.ctx), r.logoutTimeout)
	defer cancel()

	attempts := []struct {
		auth      string
		available bool
	}{
		{auth: authSessionToken, available: session.Token != ""},
		{auth: authBasic, available: r.username != "" && r.password != ""},
	}

	for _, attempt := range attempts {
		if !attempt.available {
			logger.Debug("skipping direct DELETE, no credentials for it", slog.String("auth", attempt.auth))
			continue
		}

		attemptLogger := logger.With(slog.String("auth", attempt.auth))
		attemptLogger.Debug("issuing a direct DELETE for the redfish session")

		status, err := r.deleteSessionDirect(ctx, session, attempt.auth)
		attemptLogger = attemptLogger.With(slog.Int("status_code", status))
		if err != nil {
			attemptLogger.Error("direct DELETE for the redfish session failed", slog.Any("error", err))
			continue
		}

		attemptLogger.Warn("direct DELETE released the redfish session after gofish teardown failed")
		return true
	}

	return false
}

// deleteSessionDirect issues one DELETE against the session URI, returning the status the BMC
// answered with (0 if it answered nothing) alongside any error.
//
// It goes through the client's own *http.Client, so the request keeps the transport, TLS
// settings and instrumentation of every other request to this BMC, but bypasses gofish itself:
// gofish attaches the session token whenever it holds one and offers no way to send anything
// else, which rules out the auth scheme this fallback exists to try. It therefore also bypasses
// gofish's concurrency semaphore, which costs nothing here — teardown runs after the scrape's
// requests are done.
func (r *redfishCollector) deleteSessionDirect(ctx context.Context, session *gofish.Session, auth string) (int, error) {
	// The session URI is the path from the Location header the BMC returned when it created
	// the session; gofish joins it to the endpoint the same way.
	endpoint := fmt.Sprintf("https://%s%s", r.host, session.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")

	switch auth {
	case authSessionToken:
		req.Header.Set("X-Auth-Token", session.Token)
	case authBasic:
		req.SetBasicAuth(r.username, r.password)
	default:
		return 0, fmt.Errorf("unknown auth scheme %q", auth)
	}

	resp, err := r.redfishClient.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer schemas.DeferredCleanupHTTPResponse(resp)

	// gofish treats 200, 201, 202 and 204 as success; every other code is the BMC declining.
	if resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return resp.StatusCode, fmt.Errorf("DELETE %s: %s: %s",
			session.ID, resp.Status, strings.TrimSpace(string(body)))
	}

	return resp.StatusCode, nil
}

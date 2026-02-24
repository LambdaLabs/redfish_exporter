package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
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

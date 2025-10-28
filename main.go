package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"log/slog"

	"github.com/LambdaLabs/redfish_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/stmcginnis/gofish"
)

var (
	webConfig     = flag.String("web.config-file", "", "Path to web configuration file.")
	configFile    = flag.String("config.file", "config.yml", "Path to configuration file.")
	pprofEnabled  = flag.Bool("pprof.enabled", false, "Enable pprof handler at /pprof")
	listenAddress = flag.String(
		"web.listen-address",
		":9610",
		"Address to listen on for web interface and telemetry.",
	)
	config = &SafeConfig{
		Config: &Config{},
	}
	reloadCh chan chan error
)

func reloadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Method == "PUT" {
			slog.Info("Triggered configuration reload from /-/reload HTTP endpoint")
			err := config.ReloadConfig(*configFile)
			if err != nil {
				slog.Error("failed to reload config file", slog.Any("error", err))
				http.Error(w, "failed to reload config file", http.StatusInternalServerError)
			}
			slog.Info("config file reloaded", slog.String("operation", "sc.ReloadConfig"))

			w.WriteHeader(http.StatusOK)
			_, err = io.WriteString(w, "Configuration reloaded successfully!")
			if err != nil {
				slog.Warn("failed to send configuration reload status message")
			}
		} else {
			http.Error(w, "Only PUT and POST methods are allowed", http.StatusBadRequest)
		}
	}
}

// metricsHandler provides the client interface for the redfish_exporter.
// Clients (like Prometheus) MUST provide a target (FQDN or IP)
// and SHOULD provide a 'module' param.
func metricsHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()
		target := r.URL.Query().Get("target")
		if target == "" {
			http.Error(w, "'target' parameter must be specified", 400)
			return
		}

		logger.Debug("Scraping target host", slog.String("target", target))

		var (
			hostConfig *HostConfig
			err        error
			ok         bool
			group      []string
		)

		group, ok = r.URL.Query()["group"]

		if ok && len(group[0]) >= 1 {
			// Trying to get hostConfig from group.
			if hostConfig, err = config.HostConfigForGroup(group[0]); err != nil {
				logger.Error("error getting credentials", slog.Any("error", err))
				return
			}
		}

		// Always falling back to single host config when group config failed.
		if hostConfig == nil {
			if hostConfig, err = config.HostConfigForTarget(target); err != nil {
				logger.Error("error getting credentials", slog.Any("error", err))
				return
			}
		}

		collector := collector.NewRedfishCollector(target, hostConfig.Username, hostConfig.Password)
		registry.MustRegister(collector)
		gatherers := prometheus.Gatherers{
			prometheus.DefaultGatherer,
			registry,
		}
		// Delegate http serving to Prometheus client library, which will call collector.Collect.
		h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

// Parse the log leven from input
func parseLogLevel(level string) slog.Level {
	ret := slog.LevelInfo
	switch level {
	case "debug":
		ret = slog.LevelDebug
	case "info":
		ret = slog.LevelInfo
	case "warn":
		ret = slog.LevelWarn
	case "error":
		ret = slog.LevelError
	default:
		slog.Warn("Invalid loglevel provided. Fallback to default")
	}

	return ret
}

func main() {
	slog.Info("Starting redfish_exporter")
	flag.Parse()

	// load config  first time
	if err := config.ReloadConfig(*configFile); err != nil {
		slog.Error("Error parsing config file", slog.Any("error", err))
		os.Exit(1)
	}

	// Setup dinal logger from config
	opts := &slog.HandlerOptions{
		Level: parseLogLevel(config.Config.Loglevel),
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	slog.Info("Config successfully parsed", slog.String("loglevel", opts.Level.Level().String()))

	// load config in background to watch for config changes
	hup := make(chan os.Signal, 1)
	reloadCh = make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)

	go func() {
		for {
			select {
			case <-hup:
				if err := config.ReloadConfig(*configFile); err != nil {
					slog.Error("failed to reload config file", slog.Any("error", err))
					break
				}
				slog.Info("config file reload", slog.String("operation", "sc.ReloadConfig"))
			case rc := <-reloadCh:
				if err := config.ReloadConfig(*configFile); err != nil {
					slog.Error("failed to reload config file", slog.Any("error", err))
					rc <- err
					break
				}
				slog.Info("config file reloaded", slog.String("operation", "sc.ReloadConfig"))
				rc <- nil
			}
		}
	}()

	mux := http.NewServeMux()

	mux.Handle("/redfish", metricsHandler(logger)) // Regular metrics endpoint for local Redfish metrics.
	mux.Handle("/-/reload", reloadHandler())       // HTTP endpoint for triggering configuration reload
	mux.Handle("/metrics", promhttp.Handler())

	if *pprofEnabled {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		mux.Handle("/debug/pprof/block", pprof.Handler("block"))
		mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

		slog.Info("pprof endpoints enabled", slog.Any("endpoint", "/debug/pprof/"))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// nolint
		w.Write([]byte(`<html>
            <head>
            <title>Redfish Exporter</title>
            </head>
						<body>
            <h1>redfish Exporter</h1>
            <form action="/redfish">
            <label>Target:</label> <input type="text" name="target" placeholder="X.X.X.X" value="1.2.3.4"><br>
            <label>Group:</label> <input type="text" name="group" placeholder="group (optional)" value=""><br>
            <input type="submit" value="Submit">
						</form>
						<p><a href="/metrics">Local metrics</a></p>
            </body>
            </html>`))
	})

	exporterToolkitConf := web.FlagConfig{
		WebListenAddresses: &([]string{*listenAddress}),
		WebConfigFile:      webConfig,
	}
	slog.Info("Exporter started", slog.String("listenAddress", *listenAddress))
	srv := &http.Server{
		Handler: mux,
	}
	err := web.ListenAndServe(srv, &exporterToolkitConf, logger)
	if err != nil {
		log.Fatal(err)
	}
}

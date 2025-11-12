package main

import (
	"flag"
	"io"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/LambdaLabs/redfish_exporter/collector"
	"github.com/LambdaLabs/redfish_exporter/config"
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
	safeConfig = &config.SafeConfig{
		Config: &config.Config{},
	}
	reloadCh chan chan error
)

// 'meta' metrics, which the collector itself should emit
var (
	// collectorLastStatus tracks the last configuration staus of a collector
	collectorLastStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "redfish_exporter_config_collector_last_status",
		Help: "The last known configuration state of a collector. 0=unconfigured/unhealthy, 1=configured/healthy",
	},
		[]string{"collector_name"},
	)
	// collectorModuleUnknown tracks count of requests which asked for &module=foo, but where module foo is not
	// in the exporter configuration file
	collectorModuleUnknown = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "redfish_exporter_unknown_modules_requested_total",
		Help: "Count of requests specifying a module name not known to this exporter",
	})
)

func reloadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Method == "PUT" {
			slog.Info("Triggered configuration reload from /-/reload HTTP endpoint")
			err := safeConfig.ReloadConfig(*configFile)
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

// registerMetaMetrics registers 'meta' metrics for the redfish_exporter.
// 'meta' metrics include details like "which collectors errored during creation?"
func registerMetaMetrics() {
	custom := []prometheus.Collector{
		collectorLastStatus,
		collectorModuleUnknown,
	}

	prometheus.DefaultRegisterer.MustRegister(custom...)
}

// metricsHandler provides the client interface for the redfish_exporter.
// Clients (like Prometheus) MUST provide a target (FQDN or IP)
// and SHOULD provide a 'module' param.
func metricsHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()
		urlQuery, urlErr := url.ParseQuery(r.URL.RawQuery)
		if urlErr != nil {
			logger.Error("request query was not parsed", slog.Any("error", urlErr))
			http.Error(w, "failed to successfully parse incoming query", 400)
			return
		}
		target := urlQuery.Get("target")
		if target == "" {
			http.Error(w, "'target' parameter must be specified", 400)
			return
		}
		modules, modulesDefined := urlQuery["module"]
		if !modulesDefined || len(modules) == 0 {
			logger.Debug("incoming request included no modules. using default modules, and a future release will result in failed collection without one or more modules configured and provided")
			modules = []string{"rf_exporter_default"}
		}

		logger.Debug("Scraping target host", slog.String("target", target))

		var (
			hostConfig *config.HostConfig
			err        error
			ok         bool
			group      []string
		)

		group, ok = urlQuery["group"]

		if ok && len(group[0]) >= 1 {
			// Trying to get hostConfig from group.
			if hostConfig, err = safeConfig.HostConfigForGroup(group[0]); err != nil {
				logger.Error("error getting credentials", slog.Any("error", err))
				return
			}
		}

		// Always falling back to single host config when group config failed.
		if hostConfig == nil {
			if hostConfig, err = safeConfig.HostConfigForTarget(target); err != nil {
				logger.Error("error getting credentials", slog.Any("error", err))
				return
			}
		}

		aggregateCollector, err := collector.NewRedfishCollector(target,
			hostConfig.Username,
			hostConfig.Password)

		if err != nil {
			logger.Error("unable to create redfish client, bailing", slog.Any("error", err))
			http.Error(w, "unable to construct redfish client", 500)
			return
		}

		collectors := buildCollectorsFor(modules, safeConfig.GetModules(), aggregateCollector.Client(), logger)
		aggregateCollector.WithCollectors(collectors)
		registry.MustRegister(aggregateCollector)
		gatherers := prometheus.Gatherers{
			registry,
		}
		// Delegate http serving to Prometheus client library, which will call collector.Collect.
		h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

// buildCollectorsFor accepts a list of modules and map of moduleConfig, yielding a slice of
// prometheus.Collector built from the module config.
// It silently discards modules for which there is no module config.
// For ease onboarding from existing redfish_exporter deployments,
// a modules[0] == "rf_exporter_default" will yield a []prometheus.Collector defined
// in this function. Future relases will remove this behavior and require user input.
func buildCollectorsFor(modules []string, moduleConfig map[string]config.Module, rfClient *gofish.APIClient, logger *slog.Logger) []prometheus.Collector {
	if modules[0] == "rf_exporter_default" {
		logger.Warn("Using default collector bundle. In a future release, the exporter will require configuration of one or more modules.")
		return buildCollectorsFor([]string{
			"gpu_collector",
			"chassis_collector",
			"manager_collector",
			"system_collector",
			"telemetry_collector",
		}, config.DefaultModuleConfig, rfClient, logger)
	}
	c := []prometheus.Collector{}
	for _, module := range modules {
		if modConfig, found := moduleConfig[module]; found {
			collector, err := collector.NewCollectorFromModule(module, &modConfig, rfClient, logger)
			if err != nil {
				logger.Error("unable to create collector", slog.Any("error", err))
				collectorLastStatus.WithLabelValues(module).Set(0)
				continue
			}
			collectorLastStatus.WithLabelValues(module).Set(1)
			c = append(c, collector)
		} else {
			collectorModuleUnknown.Inc()
		}
	}
	return c
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

	// load config first time
	if err := safeConfig.ReloadConfig(*configFile); err != nil {
		slog.Error("Error parsing config file", slog.Any("error", err))
		os.Exit(1)
	}

	registerMetaMetrics()
	// Setup dinal logger from config
	opts := &slog.HandlerOptions{
		Level: parseLogLevel(safeConfig.Config.Loglevel),
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
				if err := safeConfig.ReloadConfig(*configFile); err != nil {
					slog.Error("failed to reload config file", slog.Any("error", err))
					break
				}
				slog.Info("config file reload", slog.String("operation", "sc.ReloadConfig"))
			case rc := <-reloadCh:
				if err := safeConfig.ReloadConfig(*configFile); err != nil {
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
            <label>Module:</label> <input type="text" name="module" placeholder="module (optional)" value=""><br>
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
		slog.With("error", err).Error("exiting on ListenAndServe error")
		os.Exit(1)
	}
}

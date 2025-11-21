package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/prometheus/common/expfmt"
)

// ScrapeResult holds the results of a scrape
type ScrapeResult struct {
	Target         string
	Duration       time.Duration
	MetricsCount   int
	Error          error
	CollectorTimes map[string]time.Duration // From redfish_exporter_collector_duration_seconds
	RedfishUp      float64                  // Value of redfish_up metric (0 = failed, 1 = success)
	ScrapeSuccess  bool                     // Whether this was a successful hardware scrape
}

// Config holds the configuration for the analysis
type Config struct {
	Target   string
	Endpoint string // redfish_exporter endpoint
	Runs     int
	Verbose  bool
	Format   string // "text" or "json"

	// Comparison mode
	CompareMode  bool
	MockTarget   string
	MockEndpoint string
	LiveTarget   string
	LiveEndpoint string
}

func main() {
	var config Config

	// Single target mode
	flag.StringVar(&config.Target, "target", "", "Target to scrape (IP or hostname)")
	flag.StringVar(&config.Endpoint, "endpoint", "http://localhost:9610", "Redfish exporter endpoint")
	flag.IntVar(&config.Runs, "runs", 1, "Number of scrape runs to perform")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.StringVar(&config.Format, "format", "text", "Output format (text or json)")

	// Comparison mode
	flag.BoolVar(&config.CompareMode, "compare", false, "Enable comparison mode")
	flag.StringVar(&config.MockTarget, "mock", "localhost:8443", "Mock server target (for comparison)")
	flag.StringVar(&config.MockEndpoint, "mock-endpoint", "http://localhost:9610", "Mock exporter endpoint")
	flag.StringVar(&config.LiveTarget, "live", "", "Live system target (for comparison)")
	flag.StringVar(&config.LiveEndpoint, "live-endpoint", "http://localhost:9611", "Live exporter endpoint")
	flag.Parse()

	if config.CompareMode {
		// Comparison mode
		if config.LiveTarget == "" {
			fmt.Fprintf(os.Stderr, "Error: -live target is required in comparison mode\n")
			flag.Usage()
			os.Exit(1)
		}
		runComparison(config)
	} else {
		// Single target mode
		if config.Target == "" {
			fmt.Fprintf(os.Stderr, "Error: -target is required\n")
			flag.Usage()
			os.Exit(1)
		}
		runAnalysis(config)
	}

}

func runAnalysis(config Config) {
	results := make([]ScrapeResult, config.Runs)

	fmt.Printf("Analyzing performance for target: %s\n", config.Target)
	fmt.Printf("Using exporter at: %s\n", config.Endpoint)
	fmt.Printf("Running %d scrape(s)...\n\n", config.Runs)

	for i := 0; i < config.Runs; i++ {
		if config.Runs > 1 {
			fmt.Printf("Run %d/%d: ", i+1, config.Runs)
		}

		result := performScrape(config)
		results[i] = result

		if result.Error != nil {
			fmt.Printf("ERROR: %v\n", result.Error)
		} else if !result.ScrapeSuccess {
			fmt.Printf("FAILED: Duration: %v, redfish_up=%v (only %d exporter metrics)\n",
				result.Duration, result.RedfishUp, result.MetricsCount)
		} else {
			fmt.Printf("Duration: %v, Metrics: %d\n", result.Duration, result.MetricsCount)
		}

		if config.Verbose && result.CollectorTimes != nil {
			fmt.Println("  Collector timings:")
			for collector, duration := range result.CollectorTimes {
				fmt.Printf("    %s: %v\n", collector, duration)
			}
		}
	}

	// Print summary
	printSummary(results, config)
}

func runComparison(config Config) {
	fmt.Println("=== Redfish Performance Comparison ===")
	fmt.Printf("Mock target: %s\n", config.MockTarget)
	fmt.Printf("Live target: %s\n", config.LiveTarget)
	fmt.Printf("Running %d scrapes for each target...\n\n", config.Runs)

	// Test mock server
	fmt.Println("Testing MOCK server...")
	mockResults := make([]ScrapeResult, config.Runs)
	for i := 0; i < config.Runs; i++ {
		fmt.Printf("  Run %d/%d: ", i+1, config.Runs)
		result := performScrape(Config{
			Target:   config.MockTarget,
			Endpoint: config.MockEndpoint,
		})
		mockResults[i] = result

		if result.Error != nil {
			fmt.Printf("ERROR: %v\n", result.Error)
		} else if !result.ScrapeSuccess {
			fmt.Printf("FAILED: %v (redfish_up=%v, only %d metrics)\n",
				result.Duration, result.RedfishUp, result.MetricsCount)
		} else {
			fmt.Printf("%v (%d metrics)\n", result.Duration, result.MetricsCount)
		}
	}

	// Test live system
	fmt.Println("\nTesting LIVE system...")
	liveResults := make([]ScrapeResult, config.Runs)
	for i := 0; i < config.Runs; i++ {
		fmt.Printf("  Run %d/%d: ", i+1, config.Runs)
		result := performScrape(Config{
			Target:   config.LiveTarget,
			Endpoint: config.LiveEndpoint,
		})
		liveResults[i] = result

		if result.Error != nil {
			fmt.Printf("ERROR: %v\n", result.Error)
		} else if !result.ScrapeSuccess {
			fmt.Printf("FAILED: %v (redfish_up=%v, only %d metrics)\n",
				result.Duration, result.RedfishUp, result.MetricsCount)
		} else {
			fmt.Printf("%v (%d metrics)\n", result.Duration, result.MetricsCount)
		}
	}

	// Analyze and compare
	printComparisonSummary(mockResults, liveResults, config)
}

func performScrape(config Config) ScrapeResult {
	result := ScrapeResult{
		Target:         config.Target,
		CollectorTimes: make(map[string]time.Duration),
	}

	// Build the URL
	url := fmt.Sprintf("%s/redfish?target=%s", config.Endpoint, config.Target)

	// Perform the scrape
	start := time.Now()
	resp, err := http.Get(url)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = fmt.Errorf("failed to scrape: %w", err)
		return result
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			result.Error = fmt.Errorf("error closing response body: %w", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Errorf("scrape failed with status %d: %s", resp.StatusCode, string(body))
		return result
	}

	// Parse the metrics
	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse metrics: %w", err)
		return result
	}

	// Count metrics and extract important indicators
	result.RedfishUp = 0 // Default to failed
	for name, mf := range metricFamilies {
		result.MetricsCount += len(mf.Metric)

		// Check if scrape was successful
		if name == "redfish_up" {
			for _, metric := range mf.Metric {
				if metric.Gauge != nil && metric.Gauge.Value != nil {
					result.RedfishUp = *metric.Gauge.Value
				}
			}
		}

		// Extract collector duration metrics
		if name == "redfish_exporter_collector_duration_seconds" {
			for _, metric := range mf.Metric {
				if metric.Gauge != nil && metric.Gauge.Value != nil {
					// The metric doesn't have labels in the current implementation,
					// but we track the total duration
					result.CollectorTimes["total"] = time.Duration(*metric.Gauge.Value * float64(time.Second))
				}
			}
		}

		// Extract per-collector scrape durations if available
		if strings.HasSuffix(name, "_scrape_duration_seconds") {
			for _, metric := range mf.Metric {
				if metric.Gauge != nil && metric.Gauge.Value != nil {
					collectorName := strings.TrimSuffix(name, "_scrape_duration_seconds")
					result.CollectorTimes[collectorName] = time.Duration(*metric.Gauge.Value * float64(time.Second))
				}
			}
		}
	}

	// Determine if this was a successful hardware scrape
	// Failed scrapes typically return ~30-40 exporter-only metrics
	result.ScrapeSuccess = result.RedfishUp == 1 && result.MetricsCount > 50

	return result
}

func printSummary(results []ScrapeResult, config Config) {
	if len(results) == 0 {
		return
	}

	fmt.Println("\n=== Performance Analysis Summary ===")

	// Calculate statistics
	var totalDuration time.Duration
	var totalMetrics int
	var successCount int
	durations := make([]time.Duration, 0, len(results))

	var failedScrapes int
	for _, r := range results {
		if r.Error == nil && r.ScrapeSuccess {
			totalDuration += r.Duration
			totalMetrics += r.MetricsCount
			successCount++
			durations = append(durations, r.Duration)
		} else if r.Error == nil && !r.ScrapeSuccess {
			failedScrapes++
		}
	}

	if successCount == 0 {
		fmt.Println("\n✗ No successful scrapes")
		if failedScrapes > 0 {
			fmt.Printf("All %d scrapes failed (redfish_up=0)\n", failedScrapes)
			fmt.Println("\nLikely causes:")
			fmt.Println("  • Authentication failed (check username/password)")
			fmt.Println("  • Target is not reachable (check network/firewall)")
			fmt.Println("  • Target is not a valid Redfish endpoint")
			fmt.Println("  • TLS certificate issues (if using HTTPS)")
			fmt.Println("\nDebug steps:")
			fmt.Println("  1. Check exporter logs: journalctl -u redfish_exporter -n 50")
			fmt.Println("  2. Test connectivity: curl -k https://<target>/redfish/v1/")
			fmt.Println("  3. Verify credentials in config file")
		}
		return
	}

	// Calculate average
	avgDuration := totalDuration / time.Duration(successCount)
	avgMetrics := totalMetrics / successCount

	fmt.Printf("Successful scrapes: %d/%d", successCount, len(results))
	if failedScrapes > 0 {
		fmt.Printf(" (%d failed with redfish_up=0)", failedScrapes)
	}
	fmt.Println()
	fmt.Printf("Average duration: %v\n", avgDuration)
	fmt.Printf("Average metrics: %d\n", avgMetrics)

	if len(durations) > 1 {
		// Sort durations for percentiles
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})

		fmt.Printf("Min duration: %v\n", durations[0])
		fmt.Printf("Max duration: %v\n", durations[len(durations)-1])

		// Calculate percentiles
		p50 := durations[len(durations)*50/100]
		p95 := durations[len(durations)*95/100]
		p99 := durations[len(durations)*99/100]

		fmt.Printf("P50 duration: %v\n", p50)
		fmt.Printf("P95 duration: %v\n", p95)
		fmt.Printf("P99 duration: %v\n", p99)
	}

	// Analyze performance and identify bottlenecks
	fmt.Println("\n=== Performance Analysis ===")

	// Determine performance category based on average duration
	if avgDuration < 100*time.Millisecond {
		fmt.Printf("✓ EXCELLENT: Average scrape time is %v (< 100ms)\n", avgDuration)
	} else if avgDuration < 1*time.Second {
		fmt.Printf("✓ GOOD: Average scrape time is %v (< 1s)\n", avgDuration)
	} else if avgDuration < 10*time.Second {
		fmt.Printf("⚠ SLOW: Average scrape time is %v (1s - 10s)\n", avgDuration)
		fmt.Println("  This may cause issues with Prometheus scrape intervals")
	} else if avgDuration < 30*time.Second {
		fmt.Printf("⚠ VERY SLOW: Average scrape time is %v (10s - 30s)\n", avgDuration)
		fmt.Println("  This will likely cause Prometheus timeouts with default settings")
	} else {
		fmt.Printf("✗ CRITICAL: Average scrape time is %v (> 30s)\n", avgDuration)
		fmt.Println("  Scrapes are taking too long and will timeout in most configurations")
	}

	// Analyze variance to identify consistency issues
	if len(durations) > 1 {
		variance := durations[len(durations)-1] - durations[0]
		if variance > avgDuration/2 {
			fmt.Printf("\n⚠ HIGH VARIANCE: Response times vary by %v\n", variance)
			fmt.Println("  This suggests intermittent performance issues")
		}
	}

	// Identify likely bottlenecks based on duration patterns
	fmt.Println("\n=== Likely Bottlenecks ===")

	if avgDuration > 30*time.Second {
		fmt.Println("With scrape times > 30 seconds, likely causes:")
		fmt.Println("  1. Network latency or packet loss to BMC")
		fmt.Println("  2. BMC is overloaded or slow to respond")
		fmt.Println("  3. Too many resources being enumerated (many chassis/systems)")
		fmt.Println("  4. Authentication delays or session management issues")
		fmt.Println("\nRecommendations:")
		fmt.Println("  • Check network connectivity to BMC (ping, traceroute)")
		fmt.Println("  • Verify BMC firmware is up to date")
		fmt.Println("  • Consider reducing scrape frequency")
		fmt.Println("  • Check if BMC has rate limiting enabled")
	} else if avgDuration > 10*time.Second {
		fmt.Println("With scrape times 10-30 seconds, likely causes:")
		fmt.Println("  1. Large number of components being scraped")
		fmt.Println("  2. Slow BMC response times")
		fmt.Println("  3. Sequential processing of many resources")
		fmt.Println("\nRecommendations:")
		fmt.Println("  • Review which collectors are needed")
		fmt.Println("  • Consider filtering unnecessary metrics")
		fmt.Println("  • Check BMC performance and load")
	} else if avgDuration > 1*time.Second {
		fmt.Println("With scrape times 1-10 seconds:")
		fmt.Println("  • Performance is acceptable but could be improved")
		fmt.Println("  • Consider caching if scraping frequently")
		fmt.Println("  • Monitor for degradation over time")
	}

	// Compare mock vs live performance if both available
	if config.Target != "localhost:8000" && config.Target != "localhost" {
		fmt.Println("\n=== Performance Comparison ===")
		fmt.Println("To identify if delays are network/BMC related, compare with mock:")
		fmt.Println("  make perf-mock  # Test with mock server")
		fmt.Printf("  Current target (%s): %v average\n", config.Target, avgDuration)
		fmt.Println("\nIf mock is significantly faster (< 100ms), the issue is:")
		fmt.Println("  • Network latency to BMC")
		fmt.Println("  • BMC processing speed")
		fmt.Println("  • Number of resources on actual hardware")
	}

	// Metrics analysis
	if avgMetrics > 0 {
		metricsPerSecond := float64(avgMetrics) / avgDuration.Seconds()
		fmt.Printf("\n=== Metrics Efficiency ===\n")
		fmt.Printf("Metrics collected: %d\n", avgMetrics)
		fmt.Printf("Metrics per second: %.1f\n", metricsPerSecond)

		if metricsPerSecond < 10 {
			fmt.Println("⚠ LOW THROUGHPUT: Less than 10 metrics/second")
			fmt.Println("  This indicates significant overhead per metric")
		} else if metricsPerSecond < 100 {
			fmt.Println("⚠ MODERATE THROUGHPUT: 10-100 metrics/second")
		} else {
			fmt.Printf("✓ GOOD THROUGHPUT: %.1f metrics/second\n", metricsPerSecond)
		}
	}

	// Output JSON if requested
	if config.Format == "json" {
		fmt.Println("\n=== JSON Output ===")
		output := map[string]interface{}{
			"target":       config.Target,
			"runs":         len(results),
			"successful":   successCount,
			"avg_duration": avgDuration.Seconds(),
			"avg_metrics":  avgMetrics,
			"results":      results,
		}

		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}

func printComparisonSummary(mockResults, liveResults []ScrapeResult, config Config) {
	// Calculate statistics
	mockStats := calculateStats(mockResults)
	liveStats := calculateStats(liveResults)

	if mockStats.count == 0 || liveStats.count == 0 {
		fmt.Println("\n✗ Unable to compare - scrapes failed")
		if mockStats.count == 0 {
			fmt.Println("  Mock server scrapes failed (redfish_up=0)")
			fmt.Println("  Check that mock server is running on localhost:8000")
			fmt.Println("  Check exporter config has correct credentials for mock")
		}
		if liveStats.count == 0 {
			fmt.Println("  Live system scrapes failed (redfish_up=0)")
			fmt.Println("  Check credentials (REDFISH_USER/REDFISH_PASS)")
			fmt.Println("  Verify network connectivity to target")
			fmt.Println("  Check if target has valid Redfish API")
		}
		return
	}

	// Calculate comparison metrics
	speedupFactor := float64(liveStats.avg) / float64(mockStats.avg)
	networkOverhead := liveStats.avg - mockStats.avg
	overheadPercent := (float64(networkOverhead) / float64(liveStats.avg)) * 100

	fmt.Println("\n=== Performance Summary ===")
	fmt.Printf("%-20s %15s %15s\n", "Target", "Mock", "Live")
	fmt.Printf("%-20s %15s %15s\n", "─────────────────", "───────────", "───────────")
	fmt.Printf("%-20s %15v %15v\n", "Average Duration", mockStats.avg, liveStats.avg)
	fmt.Printf("%-20s %15v %15v\n", "Min Duration", mockStats.min, liveStats.min)
	fmt.Printf("%-20s %15v %15v\n", "Max Duration", mockStats.max, liveStats.max)
	fmt.Printf("%-20s %15d %15d\n", "Avg Metrics", mockStats.metrics, liveStats.metrics)
	fmt.Printf("%-20s %15.1f %15.1f\n", "Metrics/sec", mockStats.throughput, liveStats.throughput)

	fmt.Println("\n=== Performance Comparison ===")
	fmt.Printf("Live system is %.1fx slower than mock\n", speedupFactor)
	fmt.Printf("Network/BMC overhead: %v (%.1f%% of total time)\n",
		networkOverhead, overheadPercent)

	// Identify bottleneck
	var bottleneck string
	if speedupFactor < 2 {
		bottleneck = "Processing/Code (similar performance between mock and live)"
	} else if speedupFactor < 10 {
		bottleneck = "Mixed (both network and processing contribute)"
	} else if speedupFactor < 100 {
		bottleneck = "Network latency or BMC response time"
	} else {
		bottleneck = "Severe network/BMC issues or resource enumeration"
	}

	fmt.Println("\n=== Bottleneck Analysis ===")
	fmt.Printf("Primary bottleneck: %s\n", bottleneck)

	// Provide recommendations
	fmt.Println("\n=== Recommendations ===")
	if speedupFactor > 100 {
		fmt.Println("✗ CRITICAL PERFORMANCE GAP")
		fmt.Println("  The live system is >100x slower than mock, indicating:")
		fmt.Println("  • Severe network issues (check connectivity, packet loss)")
		fmt.Println("  • BMC is overloaded or unresponsive")
		fmt.Println("  • Possible authentication/session issues")
		fmt.Println("\nImmediate actions:")
		fmt.Println("  1. Check network path to BMC (ping, traceroute)")
		fmt.Println("  2. Verify BMC is not rate-limiting requests")
		fmt.Println("  3. Check BMC CPU/memory usage")
		fmt.Println("  4. Review BMC logs for errors")
	} else if speedupFactor > 10 {
		fmt.Println("⚠ SIGNIFICANT PERFORMANCE GAP")
		fmt.Println("  The live system is 10-100x slower, indicating:")
		fmt.Println("  • High network latency to BMC")
		fmt.Println("  • Slow BMC response times")
		fmt.Println("  • Large number of hardware components")
		fmt.Println("\nRecommended actions:")
		fmt.Println("  1. Optimize network path to BMC")
		fmt.Println("  2. Consider BMC firmware updates")
		fmt.Println("  3. Reduce scrape frequency if possible")
	} else if speedupFactor > 2 {
		fmt.Println("⚠ MODERATE PERFORMANCE GAP")
		fmt.Println("  The live system is 2-10x slower")
		fmt.Println("  This is expected for remote BMC access")
		fmt.Println("\nOptimizations to consider:")
		fmt.Println("  1. Implement caching if appropriate")
		fmt.Println("  2. Filter unnecessary metrics")
		fmt.Println("  3. Monitor for degradation over time")
	} else {
		fmt.Println("✓ MINIMAL PERFORMANCE GAP")
		fmt.Println("  Mock and live performance are similar")
		fmt.Println("  This suggests the bottleneck is in the exporter code itself")
		fmt.Println("\nCode optimizations needed:")
		fmt.Println("  1. Profile the exporter code")
		fmt.Println("  2. Optimize parallel processing")
		fmt.Println("  3. Review data parsing efficiency")
	}

	// Component breakdown if very slow
	if liveStats.avg > 30*time.Second {
		fmt.Println("\n=== Component Analysis ===")
		fmt.Println("With scrape times >30s on live system:")
		fmt.Printf("Processing rate: %.2f metrics/second\n", liveStats.throughput)

		if mockStats.metrics > 0 && liveStats.metrics > mockStats.metrics {
			componentRatio := float64(liveStats.metrics) / float64(mockStats.metrics)
			fmt.Printf("Live system has ~%.1fx more metrics than mock\n", componentRatio)
			fmt.Println("Consider if all components need to be scraped")
		}
	}

	if config.Verbose {
		fmt.Println("\n=== Detailed Metrics ===")
		fmt.Printf("Mock: successful=%d, failed=%d\n",
			mockStats.count, len(mockResults)-mockStats.count)
		fmt.Printf("Live: successful=%d, failed=%d\n",
			liveStats.count, len(liveResults)-liveStats.count)
	}
}

type stats struct {
	avg        time.Duration
	min        time.Duration
	max        time.Duration
	metrics    int
	throughput float64
	count      int
}

func calculateStats(results []ScrapeResult) stats {
	var total time.Duration
	var count, metrics int
	var min, max time.Duration

	for _, r := range results {
		if r.Error == nil && r.ScrapeSuccess {
			total += r.Duration
			metrics += r.MetricsCount
			count++

			if min == 0 || r.Duration < min {
				min = r.Duration
			}
			if r.Duration > max {
				max = r.Duration
			}
		}
	}

	if count == 0 {
		return stats{}
	}

	avg := total / time.Duration(count)
	avgMetrics := metrics / count
	throughput := float64(avgMetrics) / avg.Seconds()

	return stats{
		avg:        avg,
		min:        min,
		max:        max,
		metrics:    avgMetrics,
		throughput: throughput,
		count:      count,
	}
}

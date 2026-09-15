---
name: go-conventions
description: Applies whenever writing, editing or reviewing Go code in this repository — a collector, a parse helper, a test, anything under internal/ or cmd/. Covers when a goroutine is justified, how a panic is kept from taking the exporter down, and why a missing Redfish subresource is not a scrape failure.
---

# Go in this repository

## A goroutine is justified by a Redfish call, nothing else

A scrape is slow because of HTTP round trips to the BMC, never because of CPU.
Spawn a goroutine only when the work inside it performs a **Redfish GET**.

Decide by asking what the callee does:

| Work | Concurrency |
|---|---|
| `chassis.ThermalSubsystem()`, `thermalSubsystem.ThermalMetrics()`, `parseNetworkAdapter` (fetches `NetworkPorts`) — a gofish accessor method, or a helper that calls one | goroutine |
| `parseChassisFan`, `parseChassisTemperature`, `parseLeakDetector` — reads fields off an already-fetched struct and sends on `ch` | plain `for` loop |

A gofish **method** with an `error` return (`chassis.Power()`, `psu.Metrics()`)
is a GET. A gofish **field** (`thermal.Fans`, `chassis.PhysicalSecurity`) is
already in memory: the parent fetch paid for it.

Fanning out over an in-memory slice costs goroutine setup, errgroup
bookkeeping and scheduler jitter per element, per chassis, per scrape, and buys
nothing — the metric channel has no ordering guarantee either way, so the loop
is also the simpler read.

`collectThermalSubsystem` (`internal/collector/chassis_collector.go`) is the
shape to copy: three goroutines because `Fans`, `ThermalMetrics` and
`LeakDetection` are three separate GETs, and a plain loop inside each one over
the members that GET returned.

## Every goroutine goes through `newRecoverGroup`

`internal/collector/errgroup.go`. Never a bare `go func()`, never
`errgroup.Group` directly.

```go
eg := newRecoverGroup(ctx)
eg.Go(func() error {
    // ... one Redfish call and the parsing of its result
    return nil
})
if err := eg.Wait(); err != nil {
    logger.Error("goroutine error", slog.Any("error", err))
}
```

A panic in a bare goroutine — a nil pointer on a field a BMC omitted, an index
off a short array — is unrecoverable and kills the whole exporter process, so
one malformed response on one host stops every scrape for every host.
`recoverGroup.Go` recovers, logs it with the context logger and turns it into
an error on `Wait()`.

Test helpers draining a metric channel are the one place a bare `go func()` is
fine; they are not collecting.

## A missing subresource is normal, not a scrape failure

BMCs implement a subset of Redfish and vendors disagree about which subset.
Log what is absent and keep collecting everything else — never return early and
never propagate the error up to fail the scrape. `collect` distinguishes three
cases per resource, and the third is not an error:

```go
thermal, err := chassis.Thermal()
if err != nil {
    logger.Error("error getting thermal data from chassis", slog.String("operation", "chassis.Thermal()"), slog.Any("error", err))
} else if thermal == nil {
    logger.Info("no thermal data found", slog.String("operation", "chassis.Thermal()"))
} else {
    collectThermal(ch, chassisID, thermal)
}
```

Same rule one level up: one unreachable chassis must not cost you the chassis
that did answer (`TestCollectSurvivesOneUnreachableChassis`).

Inside a `recoverGroup`, return `nil` for a failure you have already logged so
the sibling goroutines still finish.

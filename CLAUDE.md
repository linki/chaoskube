# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`chaoskube` is a Kubernetes chaos engineering tool that periodically kills random pods in a cluster. It runs as a long-lived process (or Kubernetes deployment) and terminates pods at a configurable interval, with optional filtering by labels, annotations, namespaces, pod name patterns, owner kinds, and time-based exclusions (weekdays, times of day, days of year).

## Commands

```bash
# Build
go build -o bin/chaoskube -v

# Run all tests (with race detector and coverage)
GODEBUG=randseednop=0 go test ./... -race -cover

# Run a single package's tests
GODEBUG=randseednop=0 go test ./chaoskube/... -race -cover

# Run a single test by name
GODEBUG=randseednop=0 go test ./chaoskube/... -run TestSuite/TestFilterByNamespaces -v
```

`GODEBUG=randseednop=0` is required so that random selection in tests is deterministic.

Tests use `testify/suite` — individual test names follow the pattern `TestSuite/TestFunctionName`.

## Architecture

The code is organized around four packages plus `main.go`:

**`chaoskube/` — core logic**
The `Chaoskube` struct holds all configuration (selectors, filters, timezone, terminator, notifier). Its main entry points are:
- `Run(ctx, ticker)` — loop that calls `TerminateVictims` on each tick
- `TerminateVictims(ctx)` — checks time-based exclusions, then calls `Victims` → `DeletePod`
- `Candidates(ctx)` — queries Kubernetes API and applies all pod filters in sequence
- `filterBy*` functions — pure functions that filter `[]v1.Pod` slices; these are where new filter types are added

**`terminator/` — pod deletion**
The `Terminator` interface has a single method `Terminate(ctx, pod)`. The only production implementation is `DeletePodTerminator` in `terminator/delete_pod.go`, which calls the Kubernetes API. This interface exists to allow testing without a real cluster.

**`notifier/` — post-termination notifications**
The `Notifier` interface has `NotifyPodTermination(pod)`. `Notifiers` is a composite that fans out to multiple implementations. Currently only Slack is implemented (`slack.go`). The `Noop` implementation is used in tests.

**`util/` — time parsing helpers**
Parses weekday lists, time-of-day periods (`HH:MM-HH:MM`), and day-of-year strings (`Apr1`). Also contains `RandomPodSubSlice`.

**`metrics/` — Prometheus metrics**
Registers counters/histograms for terminated pods, errors, and interval counts.

**`main.go`** — CLI flag parsing (via `kingpin`), client construction, wiring everything together. All flags also accept `CHAOSKUBE_<UPPERCASE_FLAG>` env vars.

## Testing approach

Tests use `k8s.io/client-go/kubernetes/fake` for a fake Kubernetes client — no real cluster needed. The `internal/testutil` package provides a base `TestSuite` (embedding `testify/suite.Suite`) with helpers `AssertPod`/`AssertPods`/`AssertLog`. All test files embed this suite and register with `suite.Run(t, new(Suite))`.

When adding new filter types, follow the pattern in `chaoskube.go`: add a field to `Chaoskube`, add a `filterBy*` function (pure, takes `[]v1.Pod` + selector, returns `[]v1.Pod`), and call it in `Candidates`.

# k8s-tools

Update Helm chart dependencies in a directory and its child directories.

```sh
k8s-tools update ./charts
k8s-tools -threads 4 --command-timeout 2m update ./charts
k8s-tools --version
```

The update command runs `helm dependency update` for each discovered chart. Helm must be available on PATH. Without a directory argument, the command searches `$HOME/repos`. Explicit directory arguments are used literally; the shell handles any desired variable expansion. Only one directory argument is accepted. Extra arguments (including flags placed after the directory) fail before discovery or Helm execution; global flags must precede `update`.

Individual chart failures are logged, and independent charts continue processing. The command exits with status 0 only when every discovered chart succeeds (including when no charts are found). It exits with status 1 if any chart fails or chart discovery fails; the final error lists every failed chart path. `--threads` must be greater than zero and must precede `update`; invalid concurrency exits with status 1 before chart discovery or Helm execution.

Each Helm invocation has a five-minute deadline, configurable with the global `--command-timeout` duration flag (for example `30s` or `2m`). The value must be positive and the flag must precede `update`. A timeout fails that chart and lets independent charts continue. SIGINT (Ctrl-C) or SIGTERM cancels discovery and scheduling, stops running Helm process groups on Linux/macOS, waits for workers, and exits with status 1. Previously recorded failures remain in the final error. Descendants that deliberately create a new process group/session are outside group cleanup; output-pipe waits are bounded to one additional second. Discovery checks cancellation between filesystem operations; an individual blocked filesystem call cannot be interrupted.

Failure diagnostics retain the last 64 KiB of combined stdout/stderr per command and mark truncated output. These diagnostics appear in error-level logs without requiring verbose mode.

Discovery updates root charts only: once a chart is found, its directory is not traversed, so vendored subcharts remain owned by their parent chart.

Discovery treats a missing `Chart.yaml` as a non-chart. Other filesystem errors (such as a symlink loop or inaccessible chart path) abort discovery with the affected path and exit status 1 before any Helm updates start. A directory named `Chart.yaml` is not a chart file.

## Build and verify

Use the Go version in `go.mod`. Run `make check` for formatting, vet, race tests, and a local binary. Run `make release-check` with GoReleaser 2.18.2 to create Linux and macOS archives for amd64 and arm64 plus SHA-256 checksums without publishing.

`VERSION` records the application version. Keep it aligned with the default version in `main.go`, the release snapshot version, and the Dockerfile default. Release tags use `v<VERSION>`.

## Release

The release workflow checks a version tag before publishing CLI archives and checksums to GitHub. Pull request checks use read-only repository access and no secrets. Only the tag-triggered publication job receives contents:write, using the job's short-lived `GITHUB_TOKEN`. No repository secrets are required.

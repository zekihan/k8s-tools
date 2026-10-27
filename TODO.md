# OPS-119 follow-up

- [ ] Review the draft PR and merge only after separate authorization.
- [ ] Review GitHub's automatically triggered PR checks when available. Private Actions budget may be exhausted; no workflow retries, billing changes, or weakened checks were attempted. Local validation is the evidence for this change.

## Local validation

- `GOCACHE=/tmp/ops119-go-cache make check`: passed (formatting, vet, full race suite, binary build), using Go 1.27.3 on macOS arm64. Go emitted a non-fatal module stat-cache write warning under the sandbox; the build exited successfully.
- `GOCACHE=/tmp/ops119-go-cache go test -v -run 'Test(IsHelmDir|Discovery)' ./...`: passed, including the permission-error fixture without skipping.
- `goreleaser check`: passed.
- `git diff --check`: passed.

All fixtures use temporary local files and fake Helm commands. No live Kubernetes/Helm acceptance is needed for this filesystem error-handling fix. No deployment or release was performed.

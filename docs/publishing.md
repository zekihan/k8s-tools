# Publishing

Pull requests and the tagged publication workflow run `make check` and
`goreleaser check` on the checked-out commit. The publication workflow then
creates the release archives.

Stable releases start from an existing signed `vX.Y.Z` tag whose commit belongs to
main and whose version matches `VERSION`. Prepare the version locally, update any
version adapters, run checks, commit and sign the tag. Push only when ready:

```sh
git push --atomic origin main vX.Y.Z
```

The tagged workflow validates the captured commit and creates its GitHub release.

Publishing uses full-SHA pins to `zekihan/release-workflows`. Review pin updates;
the shared workflow embeds its helpers rather than fetching executable scripts from
a moving branch. Repo-specific build commands run without publication credentials.

Registry updates are not atomic. If publication stops partway, use **Re-run failed
jobs** within seven days to reuse the prepared artifacts. Re-running every job can
produce different bytes and correctly fail the immutability check. Never retag or
overwrite an existing release to recover. After artifact expiry, use a new version
if the original content cannot be recovered.

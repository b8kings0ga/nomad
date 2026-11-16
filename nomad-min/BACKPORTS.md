# Upstream backports

## 2026-11-16: consolidated fork release

The release version is `2.0.7+mimc.20261116`. The upstream base remains Nomad
2.0.7, with the 2.1.x fixes listed below. Source version metadata and
`UPSTREAM_VERSION` agree; packaging refuses inconsistent version overrides.

This release also incorporates the remaining changes from PRs #8 and #13 and
the October hoststats/stateio prereleases:

- `a9fb96a9ca`: guarded zero-disk mimc sandbox compatibility and bounded health pings.
- `5cb440775a`, `b9da240eda`: healthy heartbeat TTLs, no-leader backoff, canceled
  RPC stream opening, pool shutdown handling, remaining read cancellation,
  removed minimal routes, and ordinary-build Vault task cancellation.
- `5140d4f810`: serialize concurrent initial host statistics collection.
- `bd1247185d`: keep configuration readers live during durable metadata writes,
  retaining the main branch's identical-update suppression.

The verification gate covers these regressions, including pool, hoststats, and
metadata race checks. Release packages are checked against their manifests,
SHA256 files, embedded metadata, and actual version/profile/revision output on
both architectures; arm64 execution uses QEMU on the amd64 runner. Historical
immutable releases and prereleases remain available for rollback.

## 2026-11-14: fixes from Nomad 2.1.x

The latest stable upstream release checked was `v2.1.1` (2026-10-29), commit
`d0f2104299`. These fixes are cherry-picked onto fork commit `ef18a280da`.
This remains a `2.0.7`-based fork; `UPSTREAM_VERSION` and `UPSTREAM_COMMIT`
continue to identify that base, rather than claiming a complete 2.1.1 upgrade.
Each backport commit records its upstream source with `cherry-pick -x`.

| Upstream commit | Included change |
| --- | --- |
| `862114e0ec` | Restrict node heartbeats to the same node or the leader. |
| `04af70ce00` | Avoid a `var put` panic on invalid backslash input. |
| `84559c2feb` | Report only supported CSI plugin capabilities. |
| `4cf0f23713` | Fix Apple M4 fingerprinting with go-m1cpu 0.2.3. |
| `0c8fe412fa` | Reject negative Apple Silicon core counts before indexing. |
| `b1e9965dd2` | Avoid client-wide serialization of task process cleanup. |
| `9d4576f1dd` | Refresh optimistic scheduling snapshots under sustained load. |
| `6ba4975265` | Correct job-status pagination when ModifyIndex values repeat, including the matching UI changes. |
| `8246ad1ff2` | Restrict namespace reads and secure task directory migration, dispatch payloads, log files, and related task files. |
| `7bbc27c2a4` | Update go-jose/v4 to 4.1.6. |

The 2.1 batch queue feature and its queue-specific timeout fix are excluded
because this fork does not include that feature. The server-join changes,
deprecated allocation field removal, TLS refactor, broad dependency/toolchain
updates, and unrelated UI changes are also excluded from this fix-only update.
The existing nomad_min profile, mimc driver support, admission checks, and local
RPC/pool fixes are retained.

### Validation

`CI=true GOMAXPROCS=2 GOFLAGS='-p=2' scripts/nomad-min/verify.sh` passed on
darwin/arm64 with Go 1.26.7. This covers ordinary and minimal compilation,
targeted heartbeat/namespace/pagination/planner tests, dispatch and logmon hooks,
directory migration, log rotation, dependency exclusions, both Linux architecture
builds, and the minimal runtime command/configuration checks.

A broader macOS test run also found failures in topology detection, allocation
directory mounting/secret reads, symlink path checks, logmon restart, and the
client reconnect integration test. All failure groups reproduced on the unchanged
`ef18a280da` baseline; reconnect failed with the same zero-allocation timeout.
The verification script selects the backport-specific cases rather than claiming
that these broader suites pass. Linux-only mount/cgroup runtime tests and UI
browser tests were not run locally. No release or cluster rollout was performed.

### Fork CI

Only `.github/workflows/nomad-min-upstream.yml` is retained as an executable
workflow in this fork. Pull requests targeting `nomad-min`, pushes to that branch,
and manual `verify_only` runs execute `scripts/nomad-min/verify.sh` on a hosted
Linux runner, then build and verify both architecture packages. These events do
not publish releases. Scheduled and manual publication use the reviewed default
branch and publish the same validated artifacts. Release tags include the fork
version and source revision; existing assets are never overwritten. The workflow
does not replay upstream during publication. Upstream changes must be reviewed
and merged separately. Inherited HashiCorp workflows remain disabled and removed.

To validate the current branch without publishing, dispatch the workflow with
`verify_only=true`. To publish the reviewed default branch, dispatch it on
`nomad-min` with `verify_only=false`. Only the publication job has write access.

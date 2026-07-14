# nomad_min runtime benchmark

The runtime profile is measured on the same idle Mimir client, with the same
Nomad v2.0.4 source, configuration, data directory, server, and external driver
plugins. Each result is a 15-second warmup followed by 45 seconds of Linux
`/proc` sampling at one-second intervals.

Run the sampler on a Linux node with:

```sh
scripts/nomad-min/bench-linux.sh --unit nomad --warmup 15 --duration 45
```

## 2026-07-14 Orb VM result

VM: `mimir-uslax-ac-vanat`, Linux arm64, idle registered client.

| Build | Run | CPU | RSS avg | RSS max | PSS avg | Threads at end |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| v2.0.4 before runtime pruning | 1 | 0.878% | 35,591 KiB | 48,992 KiB | 35,587 KiB | 16 |
| v2.0.4 before runtime pruning | 2 | 0.607% | 37,083 KiB | 38,896 KiB | 37,079 KiB | 15 |
| v2.0.4 runtime-pruned | 1 | 0.636% | 36,918 KiB | 49,084 KiB | 36,914 KiB | 13 |
| v2.0.4 runtime-pruned | 2 | 0.461% | 38,237 KiB | 41,632 KiB | 38,233 KiB | 15 |

The two-run mean CPU usage fell from 0.743% to 0.549%, a 26.1% reduction.
RSS did not show a repeatable improvement: mean RSS increased by 3.4% and the
largest observed RSS was effectively unchanged. Treat this change as a CPU and
background-work optimization, not as a demonstrated memory reduction.

The runtime-pruned profile removes server metrics publisher goroutines, skips
client metric conversion and blackhole emission, disables allocation-hook
metrics, and limits required host-stat collection to no more than once every
10 seconds. Host statistics collection itself remains enabled because client
resource accounting depends on it.

The arm64 stripped binary changed from 43,843,746 to 43,778,210 bytes
(-65,536 bytes, -0.15%). Package size is not the purpose of this change.

## 2026-07-14 event and scheduler pruning

VM: `mimc-memtest`, Linux arm64. The baseline and candidate were built from
the same v2.0.4 commit and ran concurrently as isolated idle servers with
separate data directories and ports. The candidate disables the event broker,
does not register the HTTP or streaming RPC event endpoints, defaults to one
scheduler worker, and omits the per-task metrics stats hook.

| Build | Run | CPU | RSS avg | RSS max | PSS avg | Threads at end |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| v2.0.4 baseline | 1 | 0.338% | 42,349 KiB | 42,564 KiB | 42,345 KiB | 12 |
| v2.0.4 baseline | 2 | 0.341% | 45,207 KiB | 45,220 KiB | 45,203 KiB | 14 |
| v2.0.4 event/scheduler-pruned | 1 | 0.270% | 42,048 KiB | 42,184 KiB | 42,044 KiB | 11 |
| v2.0.4 event/scheduler-pruned | 2 | 0.409% | 44,724 KiB | 44,804 KiB | 44,720 KiB | 15 |

The two-run mean RSS/PSS fell by about 392 KiB, or 0.9%. CPU and thread count
did not improve repeatably, so they must not be presented as a measured gain.
The stripped arm64 binary remained exactly 43,778,210 bytes; this is a runtime
memory reduction rather than a link-size reduction.

Both isolated servers elected themselves leader. The candidate returned HTTP
404 for `/v1/event/stream`, and explicitly enabling the removed broker failed
with `nomad_min: unsupported feature event stream`. The scheduler count remains
operator-configurable for installations that need more evaluation throughput.

## 2026-07-14 agent-only runtime entrypoint

The `nomad_min` executable now exposes only `nomad agent` and `nomad version`.
Executor, logmon, Docker logger, artifact getter, and template renderer
subprocess modes remain linked because the agent invokes them internally. The
ordinary build retains the complete operator CLI.

| Architecture | Previous binary | Agent-only binary | Reduction | zstd -22 reduction |
| --- | ---: | ---: | ---: | ---: |
| linux/amd64 | 47,157,410 B | 43,249,826 B | 3,907,584 B (8.29%) | 937,256 B (7.18%) |
| linux/arm64 | 43,778,210 B | 40,173,730 B | 3,604,480 B (8.23%) | 817,404 B (7.10%) |

The compressed comparison uses the stripped executable directly with
`zstd --ultra -22 -T1`; release archives also contain the small version marker
and tar metadata.

Concurrent idle Linux arm64 A/B runs did not demonstrate a runtime memory or
CPU improvement:

| Build | Run | CPU | RSS avg | PSS avg | Threads at end |
| --- | ---: | ---: | ---: | ---: | ---: |
| complete minimal CLI | 1 | 0.430% | 46,933 KiB | 46,929 KiB | 14 |
| complete minimal CLI | 2 | 0.495% | 47,496 KiB | 47,492 KiB | 14 |
| agent-only | 1 | 0.430% | 47,920 KiB | 47,916 KiB | 15 |
| agent-only | 2 | 0.528% | 47,997 KiB | 47,993 KiB | 15 |

The agent-only mean RSS was 744 KiB higher in this sample. Treat this change as
a binary-size, attack-surface, and startup-logic reduction, not as a proven
runtime memory optimization.

The agent-only binary elected a leader, registered its client, and completed a
batch allocation using a `template` stanza plus `raw_exec`. A complete external
Nomad CLI remained able to operate the agent over HTTP, while invoking
`nomad job` on the runtime executable failed explicitly with
`nomad_min: unsupported command job`.

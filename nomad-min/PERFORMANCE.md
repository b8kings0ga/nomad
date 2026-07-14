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

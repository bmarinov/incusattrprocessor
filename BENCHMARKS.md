# Benchmarks

Linux 7.2.11-200.fc44.x86_64 / go version go1.27.3 linux/amd64

```
goos: linux
goarch: amd64
pkg: github.com/bmarinov/incusattrprocessor
cpu: AMD Ryzen 7 7840U w/ Radeon  780M Graphics     
                                                 │  bench.out  │
                                                 │   sec/op    │
ProcessProfiles/real_proc-16                       894.3µ ± 6%
ProcessProfiles/page-cached/1/all-container-16     7.308µ ± 1%
ProcessProfiles/page-cached/1/mixed-50-16          7.545µ ± 1%
ProcessProfiles/page-cached/1/all-host-16          7.642µ ± 4%
ProcessProfiles/page-cached/10/all-container-16    74.51µ ± 3%
ProcessProfiles/page-cached/10/mixed-50-16         74.36µ ± 1%
ProcessProfiles/page-cached/10/all-host-16         75.35µ ± 3%
ProcessProfiles/page-cached/100/all-container-16   756.8µ ± 2%
ProcessProfiles/page-cached/100/mixed-50-16        758.9µ ± 2%
ProcessProfiles/page-cached/100/all-host-16        794.0µ ± 7%
geomean                                            96.73µ

                                                 │  bench.out  │
                                                 │ profiles/s  │
ProcessProfiles/real_proc-16                       111.8k ± 5%
ProcessProfiles/page-cached/1/all-container-16     136.8k ± 1%
ProcessProfiles/page-cached/1/mixed-50-16          132.5k ± 1%
ProcessProfiles/page-cached/1/all-host-16          130.9k ± 3%
ProcessProfiles/page-cached/10/all-container-16    134.2k ± 3%
ProcessProfiles/page-cached/10/mixed-50-16         134.5k ± 1%
ProcessProfiles/page-cached/10/all-host-16         132.7k ± 2%
ProcessProfiles/page-cached/100/all-container-16   132.1k ± 2%
ProcessProfiles/page-cached/100/mixed-50-16        131.8k ± 2%
ProcessProfiles/page-cached/100/all-host-16        126.0k ± 7%
geomean                                            130.1k

                                                 │  bench.out   │
                                                 │     B/op     │
ProcessProfiles/real_proc-16                       453.7Ki ± 0%
ProcessProfiles/page-cached/1/all-container-16     4.559Ki ± 0%
ProcessProfiles/page-cached/1/mixed-50-16          4.547Ki ± 0%
ProcessProfiles/page-cached/1/all-host-16          4.547Ki ± 0%
ProcessProfiles/page-cached/10/all-container-16    44.62Ki ± 0%
ProcessProfiles/page-cached/10/mixed-50-16         44.34Ki ± 0%
ProcessProfiles/page-cached/10/all-host-16         44.34Ki ± 0%
ProcessProfiles/page-cached/100/all-container-16   445.1Ki ± 0%
ProcessProfiles/page-cached/100/mixed-50-16        442.3Ki ± 0%
ProcessProfiles/page-cached/100/all-host-16        442.3Ki ± 0%
geomean                                            56.42Ki

                                                 │  bench.out  │
                                                 │  allocs/op  │
ProcessProfiles/real_proc-16                       1.101k ± 0%
ProcessProfiles/page-cached/1/all-container-16      9.000 ± 0%
ProcessProfiles/page-cached/1/mixed-50-16           11.00 ± 0%
ProcessProfiles/page-cached/1/all-host-16           11.00 ± 0%
ProcessProfiles/page-cached/10/all-container-16     81.00 ± 0%
ProcessProfiles/page-cached/10/mixed-50-16          91.00 ± 0%
ProcessProfiles/page-cached/10/all-host-16          101.0 ± 0%
ProcessProfiles/page-cached/100/all-container-16    801.0 ± 0%
ProcessProfiles/page-cached/100/mixed-50-16         902.0 ± 0%
ProcessProfiles/page-cached/100/all-host-16        1.002k ± 0%
geomean                                             120.5

pkg: github.com/bmarinov/incusattrprocessor/internal/cgroup
                  │  bench.out  │
                  │   sec/op    │
Read/proc-self-16   8.670µ ± 1%
Read/temp-file-16   6.702µ ± 1%
ParseLXC-16         7.744n ± 1%
geomean             766.3n

                  │   bench.out    │
                  │      B/op      │
Read/proc-self-16   4.242Ki ± 0%
Read/temp-file-16   4.258Ki ± 0%
ParseLXC-16           0.000 ± 0%
geomean                          ¹
¹ summaries must be >0 to compute geomean

                  │  bench.out   │
                  │  allocs/op   │
Read/proc-self-16   6.000 ± 0%
Read/temp-file-16   6.000 ± 0%
ParseLXC-16         0.000 ± 0%
geomean                        ¹
¹ summaries must be >0 to compute geomean

pkg: github.com/bmarinov/incusattrprocessor/internal/metadata
                                      │  bench.out  │
                                      │   sec/op    │
CacheGetInstance/warm_hit-16            26.50n ± 3%
CacheGetInstance/warm_hit_parallel-16   28.93n ± 1%
CacheGetInstance/miss-16                3.395µ ± 9%
geomean                                 137.5n

                                      │   bench.out   │
                                      │     B/op      │
CacheGetInstance/warm_hit-16            0.000 ±  0%
CacheGetInstance/warm_hit_parallel-16   0.000 ±  0%
CacheGetInstance/miss-16                698.0 ± 13%
geomean                                             ¹
¹ summaries must be >0 to compute geomean

                                      │  bench.out   │
                                      │  allocs/op   │
CacheGetInstance/warm_hit-16            0.000 ± 0%
CacheGetInstance/warm_hit_parallel-16   0.000 ± 0%
CacheGetInstance/miss-16                6.000 ± 0%
geomean                                            ¹
¹ summaries must be >0 to compute geomean
```

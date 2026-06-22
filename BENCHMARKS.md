# Benchmarks

Linux 7.0.10-201.fc44.x86_64 / go version go1.26.3 linux/amd64

```
goos: linux
goarch: amd64
pkg: github.com/bmarinov/incusattrprocessor
cpu: AMD Ryzen 7 7840U w/ Radeon  780M Graphics     
                                                 │  bench.out  │
                                                 │   sec/op    │
ProcessProfiles/real_proc-16                       952.3µ ± 5%
ProcessProfiles/page-cached/1/all-container-16     7.335µ ± 1%
ProcessProfiles/page-cached/1/mixed-50-16          7.591µ ± 4%
ProcessProfiles/page-cached/1/all-host-16          7.713µ ± 2%
ProcessProfiles/page-cached/10/all-container-16    72.04µ ± 4%
ProcessProfiles/page-cached/10/mixed-50-16         74.82µ ± 5%
ProcessProfiles/page-cached/10/all-host-16         73.77µ ± 1%
ProcessProfiles/page-cached/100/all-container-16   732.3µ ± 2%
ProcessProfiles/page-cached/100/mixed-50-16        749.3µ ± 1%
ProcessProfiles/page-cached/100/all-host-16        761.3µ ± 3%
geomean                                            96.21µ

                                                 │  bench.out  │
                                                 │ profiles/s  │
ProcessProfiles/real_proc-16                       105.0k ± 4%
ProcessProfiles/page-cached/1/all-container-16     136.3k ± 1%
ProcessProfiles/page-cached/1/mixed-50-16          131.7k ± 4%
ProcessProfiles/page-cached/1/all-host-16          129.7k ± 2%
ProcessProfiles/page-cached/10/all-container-16    138.8k ± 4%
ProcessProfiles/page-cached/10/mixed-50-16         133.7k ± 4%
ProcessProfiles/page-cached/10/all-host-16         135.6k ± 1%
ProcessProfiles/page-cached/100/all-container-16   136.6k ± 2%
ProcessProfiles/page-cached/100/mixed-50-16        133.5k ± 1%
ProcessProfiles/page-cached/100/all-host-16        131.4k ± 3%
geomean                                            130.9k

                                                 │  bench.out   │
                                                 │     B/op     │
ProcessProfiles/real_proc-16                       455.2Ki ± 0%
ProcessProfiles/page-cached/1/all-container-16     4.559Ki ± 0%
ProcessProfiles/page-cached/1/mixed-50-16          4.547Ki ± 0%
ProcessProfiles/page-cached/1/all-host-16          4.547Ki ± 0%
ProcessProfiles/page-cached/10/all-container-16    44.62Ki ± 0%
ProcessProfiles/page-cached/10/mixed-50-16         44.34Ki ± 0%
ProcessProfiles/page-cached/10/all-host-16         44.34Ki ± 0%
ProcessProfiles/page-cached/100/all-container-16   445.1Ki ± 0%
ProcessProfiles/page-cached/100/mixed-50-16        442.3Ki ± 0%
ProcessProfiles/page-cached/100/all-host-16        442.3Ki ± 0%
geomean                                            56.43Ki

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
ProcessProfiles/page-cached/100/mixed-50-16         901.0 ± 0%
ProcessProfiles/page-cached/100/all-host-16        1.001k ± 0%
geomean                                             120.5

pkg: github.com/bmarinov/incusattrprocessor/internal/cgroup
                  │  bench.out  │
                  │   sec/op    │
Read/proc-self-16   8.396µ ± 1%
Read/temp-file-16   6.517µ ± 1%
ParseLXC-16         7.190n ± 1%
geomean             732.7n

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
CacheGetInstance/warm_hit-16            27.03n ± 2%
CacheGetInstance/warm_hit_parallel-16   28.11n ± 1%
CacheGetInstance/miss-16                3.291µ ± 1%
geomean                                 135.7n

                                      │  bench.out   │
                                      │     B/op     │
CacheGetInstance/warm_hit-16            0.000 ± 0%
CacheGetInstance/warm_hit_parallel-16   0.000 ± 0%
CacheGetInstance/miss-16                613.0 ± 2%
geomean                                            ¹
¹ summaries must be >0 to compute geomean

                                      │  bench.out   │
                                      │  allocs/op   │
CacheGetInstance/warm_hit-16            0.000 ± 0%
CacheGetInstance/warm_hit_parallel-16   0.000 ± 0%
CacheGetInstance/miss-16                6.000 ± 0%
geomean                                            ¹
¹ summaries must be >0 to compute geomean
```

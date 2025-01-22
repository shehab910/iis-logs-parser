# MVP (Naive)
I have created multiple combiners to test multiple methods which are:
1. no db used at all just parsing
1. inserting sequentially to db
1. inserting in db by batches

Note: smallest batch size used is 1000 which is less than the lines in the mini file

results are as follows:

## Using the mini file (1.7MB) file
```txt
goos: linux
goarch: amd64
pkg: iis-logs-parser/tests
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
                  │ mini_file-1.7MB-no-db.txt │      mini_file-1.7MB-batch-db.txt       │     mini_file-1.7MB-sequential-db.txt     │
                  │          sec/op           │     sec/op      vs base                 │     sec/op       vs base                  │
ProcessLogFile-12                95.18µ ± 46%   1722.88µ ± 35%  +1710.05% (p=0.001 n=7)   18307.50µ ± 56%  +19133.80% (p=0.001 n=7)

                  │ mini_file-1.7MB-no-db.txt │     mini_file-1.7MB-batch-db.txt     │  mini_file-1.7MB-sequential-db.txt   │
                  │           B/op            │     B/op      vs base                │     B/op      vs base                │
ProcessLogFile-12                18.24Ki ± 0%   67.36Ki ± 0%  +269.24% (p=0.001 n=7)   92.81Ki ± 0%  +408.75% (p=0.001 n=7)

                  │ mini_file-1.7MB-no-db.txt │    mini_file-1.7MB-batch-db.txt    │  mini_file-1.7MB-sequential-db.txt  │
                  │         allocs/op         │ allocs/op   vs base                │  allocs/op   vs base                │
ProcessLogFile-12                  241.0 ± 0%   569.0 ± 0%  +136.10% (p=0.001 n=7)   1110.0 ± 1%  +360.58% (p=0.001 n=7)

```

Here we can see that that the sequential method is comically slow compared to the batch method and the no db is just used as a base here.

## using the medium file (29MB) file
The sequential method is really bad it was taking too long to complete the benchmark so I removed it.
Here is the results for the first two iterations of the sequential method:
```txt
goos: linux
goarch: amd64
pkg: iis-logs-parser/tests
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
BenchmarkProcessLogFile-12    	       1	227066168627 ns/op	1677991616 B/op	20051051 allocs/op
BenchmarkProcessLogFile-12    	       1	239557656588 ns/op	1677897648 B/op	20050776 allocs/op
*** Test killed with quit: ran too long (11m0s).
exit status 2
FAIL	iis-logs-parser/tests	660.007s

```

We will continue with the batch method and no db method only. The purpose of keeping no db method is to measure future performance improvement away from the db as it can alter the results.

# Effect of scaling on the batch method (mini-batch vs. medium-batch)
```txt
goos: linux
goarch: amd64
pkg: iis-logs-parser/tests
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
                  │ mini_file-1.7MB-batch-db.txt │       medium_file-29MB-batch-db.txt        │
                  │            sec/op            │     sec/op       vs base                   │
ProcessLogFile-12                   1.723m ± 35%   3317.141m ± 28%  +192434.43% (p=0.001 n=7)

                  │ mini_file-1.7MB-batch-db.txt │         medium_file-29MB-batch-db.txt         │
                  │             B/op             │       B/op         vs base                    │
ProcessLogFile-12                   67.36Ki ± 0%   1211768.12Ki ± 0%  +1798807.71% (p=0.001 n=7)

                  │ mini_file-1.7MB-batch-db.txt │       medium_file-29MB-batch-db.txt        │
                  │          allocs/op           │   allocs/op     vs base                    │
ProcessLogFile-12                     569.0 ± 0%   9716963.0 ± 0%  +1707626.36% (p=0.001 n=7)
```

It's clear that the batch method is scaling very badly.

# Effect of scaling on the no db method (mini-no-db vs. medium-no-db vs. large-no-db)
To help identify the bottleneck, I did the same test using no db method.
```txt
goos: linux
goarch: amd64
pkg: iis-logs-parser/tests
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
                                         │ all-no-db.txt │
                                         │    sec/op     │
ProcessLogFile/mini_file-1.7MB-no-db-12    197.2µ ± 103%
ProcessLogFile/medium_file-29MB-no-db-12   762.0m ±  11%
ProcessLogFile/large_file-1.7GB-no-db-12    47.94 ±   3%
geomean                                    193.1m

                                         │ all-no-db.txt │
                                         │     B/op      │
ProcessLogFile/mini_file-1.7MB-no-db-12     17.96Ki ± 0%
ProcessLogFile/medium_file-29MB-no-db-12    236.4Mi ± 0%
ProcessLogFile/large_file-1.7GB-no-db-12    13.86Gi ± 0%
geomean                                     38.90Mi

                                         │ all-no-db.txt │
                                         │   allocs/op   │
ProcessLogFile/mini_file-1.7MB-no-db-12       238.0 ± 0%
ProcessLogFile/medium_file-29MB-no-db-12     3.884M ± 0%
ProcessLogFile/large_file-1.7GB-no-db-12     233.1M ± 0%
geomean                                      599.5k
```

These results made me think there is a lot to optimize outside looking in db optimization yet.

# Non-DB optimizations
### Writing parsed logs using bufio.Writer instead of fmt.Fprintln
```txt
goos: linux
goarch: amd64
pkg: iis-logs-parser/tests
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
                                         │ all-no-db-8w-step1.txt │       all-no-db-8w-bufio.txt        │
                                         │         sec/op         │    sec/op     vs base               │
ProcessLogFile/mini_file-1.7MB-no-db-12              171.0µ ± 49%   166.4µ ± 43%        ~ (p=0.710 n=7)
ProcessLogFile/medium_file-29MB-no-db-12             675.2m ± 15%   543.8m ± 13%  -19.47% (p=0.001 n=7)
ProcessLogFile/large_file-1.7GB-no-db-12              41.41 ±  6%    34.28 ±  3%  -17.21% (p=0.001 n=7)
geomean                                              168.5m         145.9m        -13.43%

                                         │ all-no-db-8w-step1.txt │       all-no-db-8w-bufio.txt        │
                                         │          B/op          │     B/op      vs base               │
ProcessLogFile/mini_file-1.7MB-no-db-12              18.25Ki ± 0%   22.15Ki ± 0%  +21.36% (p=0.001 n=7)
ProcessLogFile/medium_file-29MB-no-db-12             236.6Mi ± 0%   234.7Mi ± 0%   -0.80% (p=0.001 n=7)
ProcessLogFile/large_file-1.7GB-no-db-12             13.86Gi ± 0%   13.75Gi ± 0%   -0.80% (p=0.001 n=7)
geomean                                              39.11Mi        41.50Mi        +6.10%

                                         │ all-no-db-8w-step1.txt │      all-no-db-8w-bufio.txt       │
                                         │       allocs/op        │  allocs/op   vs base              │
ProcessLogFile/mini_file-1.7MB-no-db-12                246.0 ± 0%    240.0 ± 0%  -2.44% (p=0.001 n=7)
ProcessLogFile/medium_file-29MB-no-db-12              3.884M ± 0%   3.755M ± 0%  -3.32% (p=0.001 n=7)
ProcessLogFile/large_file-1.7GB-no-db-12              233.0M ± 0%   225.3M ± 0%  -3.32% (p=0.001 n=7)
geomean                                               606.1k        587.8k       -3.03%
```
### Using multiple combiners
Below are the results of using 1 vs many (workers/2) combiners

```txt
goos: linux
goarch: amd64
pkg: iis-logs-parser/tests
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
                                               │ all-1-combiner.txt │           all-n-combiners.txt           │
                                               │       sec/op       │     sec/op      vs base                 │
ProcessLogFile/below_md_file-17MB-no-db-12             271.0m ± 11%   226.6m ± 28%    -16.39% (p=0.026 n=7)
ProcessLogFile/below_md_file-17MB-batch-db-12          906.7m ± 15%   590.9m ± 21%    -34.83% (p=0.001 n=7)
ProcessLogFile/medium_file-29MB-no-db-12               448.5m ± 20%   367.6m ± 21%    -18.04% (p=0.001 n=7)
ProcessLogFile/medium_file-29MB-batch-db-12             1.726 ±  8%    1.123 ± 15%    -34.95% (p=0.001 n=7)
ProcessLogFile/below_lg_file-433MB-no-db-12             6.792 ±  4%    5.885 ±  7%    -13.36% (p=0.001 n=7)
ProcessLogFile/below_lg_file-433MB-batch-db-12          25.46 ±  3%    17.68 ±   ∞ ¹  -30.54% (p=0.003 n=7+5)
geomean                                                 1.790          1.339          -25.22%
¹ need >= 6 samples for confidence interval at level 0.95

                                               │ all-1-combiner.txt │          all-n-combiners.txt           │
                                               │        B/op        │      B/op       vs base                │
ProcessLogFile/below_md_file-17MB-no-db-12             116.8Mi ± 0%   117.0Mi ± 0%    +0.20% (p=0.001 n=7)
ProcessLogFile/below_md_file-17MB-batch-db-12          622.9Mi ± 0%   592.4Mi ± 0%    -4.90% (p=0.001 n=7)
ProcessLogFile/medium_file-29MB-no-db-12               233.7Mi ± 0%   233.8Mi ± 0%    +0.06% (p=0.001 n=7)
ProcessLogFile/medium_file-29MB-batch-db-12            1.220Gi ± 0%   1.160Gi ± 0%    -4.95% (p=0.001 n=7)
ProcessLogFile/below_lg_file-433MB-no-db-12            3.423Gi ± 0%   3.424Gi ± 0%    +0.04% (p=0.001 n=7)
ProcessLogFile/below_lg_file-433MB-batch-db-12         18.33Gi ± 0%   17.37Gi ±  ∞ ¹  -5.23% (p=0.003 n=7+5)
geomean                                                1.033Gi        1.007Gi         -2.50%
¹ need >= 6 samples for confidence interval at level 0.95

                                               │ all-1-combiner.txt │          all-n-combiners.txt          │
                                               │     allocs/op      │   allocs/op    vs base                │
ProcessLogFile/below_md_file-17MB-no-db-12              1.877M ± 0%   1.878M ± 0%    +0.06% (p=0.001 n=7)
ProcessLogFile/below_md_file-17MB-batch-db-12           4.789M ± 0%   4.796M ± 0%    +0.16% (p=0.001 n=7)
ProcessLogFile/medium_file-29MB-no-db-12                3.752M ± 0%   3.752M ± 0%    +0.02% (p=0.001 n=7)
ProcessLogFile/medium_file-29MB-batch-db-12             9.577M ± 0%   9.591M ± 0%    +0.15% (p=0.001 n=7)
ProcessLogFile/below_lg_file-433MB-no-db-12             56.27M ± 0%   56.28M ± 0%    +0.01% (p=0.001 n=7)
ProcessLogFile/below_lg_file-433MB-batch-db-12          143.6M ± 0%   143.9M ±  ∞ ¹       ~ (p=0.639 n=7+5)
geomean                                                 11.73M        11.74M         +0.09%
¹ need >= 6 samples for confidence interval at level 0.95
```
Note: My laptop is struggling with the benchmarks so samples are less than 6 for the last test only

We can see a small bump in performance with the memory practically the same. So, I will go with multiple combiners

### Tuning the batch size
It was surprising to see that increasing the batch size is worse for performance.
```txt
goos: linux
goarch: amd64
pkg: iis-logs-parser/tests
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
                                               │ all-n-combiners-bs1k.txt │        all-n-combiners-bs10k.txt        │       all-n-combiners-bs5k.txt        │          all-n-combiners.txt           │
                                               │          sec/op          │     sec/op      vs base                 │    sec/op     vs base                 │     sec/op      vs base                │
ProcessLogFile/below_md_file-17MB-no-db-12                 232.4m ± 96%     252.4m ± 16%          ~ (p=0.620 n=7)     229.4m ± 20%        ~ (p=0.383 n=7)     226.6m ± 28%         ~ (p=0.259 n=7)
ProcessLogFile/below_md_file-17MB-batch-db-12              599.7m ± 16%     862.0m ± 11%    +43.73% (p=0.001 n=7)     817.7m ± 10%  +36.35% (p=0.001 n=7)     590.9m ± 21%         ~ (p=0.805 n=7)
ProcessLogFile/medium_file-29MB-no-db-12                   378.4m ± 12%     356.1m ± 22%          ~ (p=0.209 n=7)     362.5m ± 21%        ~ (p=0.209 n=7)     367.6m ± 21%         ~ (p=0.535 n=7)
ProcessLogFile/medium_file-29MB-batch-db-12                 1.140 ± 12%      1.661 ± 12%    +45.74% (p=0.001 n=7)      1.485 ±  6%  +30.26% (p=0.001 n=7)      1.123 ± 15%         ~ (p=0.805 n=7)
ProcessLogFile/below_lg_file-433MB-no-db-12                 5.775 ±  6%      5.828 ±  5%          ~ (p=0.165 n=7)      5.691 ± 10%        ~ (p=1.000 n=7)      5.885 ±  7%         ~ (p=0.128 n=7)
ProcessLogFile/below_lg_file-433MB-batch-db-12              15.80 ±   ∞ ¹    23.76 ±   ∞ ¹        ~ (p=0.556 n=4+5)    23.24 ± 65%        ~ (p=0.352 n=4+6)    17.68 ±   ∞ ¹       ~ (p=0.905 n=4+5)
geomean                                                     1.328            1.616          +21.70%                    1.540        +15.98%                    1.339          +0.79%
¹ need >= 6 samples for confidence interval at level 0.95

                                               │ all-n-combiners-bs1k.txt │        all-n-combiners-bs10k.txt        │        all-n-combiners-bs5k.txt        │          all-n-combiners.txt           │
                                               │           B/op           │      B/op       vs base                 │     B/op       vs base                 │      B/op       vs base                │
ProcessLogFile/below_md_file-17MB-no-db-12                 117.1Mi ± 0%     117.1Mi ± 0%          ~ (p=0.318 n=7)     117.0Mi ±  0%        ~ (p=0.620 n=7)     117.0Mi ± 0%         ~ (p=0.710 n=7)
ProcessLogFile/below_md_file-17MB-batch-db-12              592.5Mi ± 0%     805.3Mi ± 1%    +35.91% (p=0.001 n=7)     792.6Mi ±  1%  +33.77% (p=0.001 n=7)     592.4Mi ± 0%         ~ (p=0.902 n=7)
ProcessLogFile/medium_file-29MB-no-db-12                   233.8Mi ± 0%     233.7Mi ± 0%     -0.05% (p=0.001 n=7)     233.8Mi ±  0%   -0.02% (p=0.001 n=7)     233.8Mi ± 0%         ~ (p=0.053 n=7)
ProcessLogFile/medium_file-29MB-batch-db-12                1.158Gi ± 0%     1.599Gi ± 1%    +38.07% (p=0.001 n=7)     1.564Gi ±  1%  +35.00% (p=0.001 n=7)     1.160Gi ± 0%         ~ (p=0.805 n=7)
ProcessLogFile/below_lg_file-433MB-no-db-12                3.424Gi ± 0%     3.423Gi ± 0%     -0.04% (p=0.001 n=7)     3.423Gi ±  0%   -0.02% (p=0.001 n=7)     3.424Gi ± 0%         ~ (p=0.805 n=7)
ProcessLogFile/below_lg_file-433MB-batch-db-12             14.81Gi ±  ∞ ¹   24.11Gi ±  ∞ ¹        ~ (p=0.556 n=4+5)   23.50Gi ± 75%        ~ (p=0.257 n=4+6)   17.37Gi ±  ∞ ¹       ~ (p=0.286 n=4+5)
geomean                                                   1003.8Mi          1.181Gi         +20.43%                   1.168Gi        +19.17%                   1.007Gi         +2.71%
¹ need >= 6 samples for confidence interval at level 0.95

                                               │ all-n-combiners-bs1k.txt │        all-n-combiners-bs10k.txt        │        all-n-combiners-bs5k.txt        │          all-n-combiners.txt          │
                                               │        allocs/op         │   allocs/op     vs base                 │   allocs/op    vs base                 │   allocs/op    vs base                │
ProcessLogFile/below_md_file-17MB-no-db-12                  1.878M ± 0%      1.878M ± 0%          ~ (p=0.318 n=7)      1.878M ±  0%        ~ (p=0.646 n=7)     1.878M ± 0%         ~ (p=0.710 n=7)
ProcessLogFile/below_md_file-17MB-batch-db-12               4.796M ± 0%      6.758M ± 1%    +40.91% (p=0.001 n=7)      6.785M ±  1%  +41.46% (p=0.001 n=7)     4.796M ± 0%         ~ (p=0.710 n=7)
ProcessLogFile/medium_file-29MB-no-db-12                    3.752M ± 0%      3.752M ± 0%     -0.01% (p=0.001 n=7)      3.752M ±  0%   -0.01% (p=0.001 n=7)     3.752M ± 0%         ~ (p=0.053 n=7)
ProcessLogFile/medium_file-29MB-batch-db-12                 9.591M ± 0%     13.716M ± 1%    +43.01% (p=0.001 n=7)     13.718M ±  1%  +43.04% (p=0.001 n=7)     9.591M ± 0%         ~ (p=0.805 n=7)
ProcessLogFile/below_lg_file-433MB-no-db-12                 56.28M ± 0%      56.27M ± 0%     -0.01% (p=0.001 n=7)      56.27M ±  0%   -0.00% (p=0.001 n=7)     56.28M ± 0%         ~ (p=1.000 n=7)
ProcessLogFile/below_lg_file-433MB-batch-db-12              127.8M ±  ∞ ¹    205.7M ±  ∞ ¹        ~ (p=0.413 n=4+5)    205.7M ± 64%        ~ (p=0.257 n=4+6)   143.9M ±  ∞ ¹       ~ (p=0.556 n=4+5)
geomean                                                     11.52M           14.01M         +21.65%                    14.02M        +21.74%                   11.74M         +1.99%
¹ need >= 6 samples for confidence interval at level 0.95
```



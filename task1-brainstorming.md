# Details & Thoughts

## How It Works

We can divide the parser into 3 main parts:

1. Reading the file and splitting it into lines. (reader)
2. Parsing each line into a struct. (parser)
3. Doing something with the parsed struct. (currently counts unique status codes) (processor)

- The reader reads the file line by line and sends each line to the parser channel.
- Depending on the number of workers n, n parsers will receive the lines and parse them concurrently and send the parsed results to the processor channel.
- The processor (acts as combiner) will receive the parsed results and count unique status codes.

## Performance Concerns

The current implementation depends heavily on go channels which produces many memory allocations and put a big pressure on the garbage collector when working with large files.

Below is the results of `benchstat` on two benchmark results (each ran 10 times) for a 1.7GB log file and a 29MB log file.

```log
pkg: iis-logs-parser/complete
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
                    │ complete/bigger.txt │         complete/smaller.txt         │
                    │       sec/op        │    sec/op     vs base                │
ProcessLogLgFile-12          8067.3m ± 4%   151.7m ± 28%  -98.12% (p=0.000 n=10)

                    │ complete/bigger.txt │         complete/smaller.txt         │
                    │        B/op         │     B/op      vs base                │
ProcessLogLgFile-12        5109.12Mi ± 0%   85.24Mi ± 0%  -98.33% (p=0.000 n=10)

                    │ complete/bigger.txt │        complete/smaller.txt         │
                    │      allocs/op      │  allocs/op   vs base                │
ProcessLogLgFile-12         23339.9k ± 0%   391.1k ± 0%  -98.32% (p=0.000 n=10)
```

## Trails & Errors

### Trying To Divide The File Into Chunks

I knew that processing large files would be an issue from the beginning so I tried to implement a solution that would be able to handle large files.

So, I tried to implement a solution that would take large chunks of the file (and respect the starting and ending of each individual log line) and process them concurrently.

You can find some remaining code from that trial in [./load_file_chunk/load_file_chunk.go](https://github.com/shehab910/iis-logs-parser/blob/master/load_file_chunk/load_file_chunk.go)

I discovered that `os.File.Seek` doesn't affect `bufio.Reader` and `bufio` doesn't implement the seek interface as mentioned in this [stack overflow question](https://stackoverflow.com/questions/20441966/seek-in-buffered-reader#:~:text=Unfortunately%20bufio.Reader%20doesn%27t%20support%20seeking.).

The reason I didn't continue with this approach is that I will not be reusing memory which really defeats the reason i wanted to go with at the first place.

Below is the benchmarks for reusing memory vs not reusing memory.
Note: the below benchmarks for earlier implementation that doesn't respect the ending of each log line, meaning the ending of the chunk could contain half a log line which is not a valid log line.

```go
// Using a ~25MB file With 4 Bytes chunk size
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
BenchmarkChunkingWithChangingBuf-12    	       1	2615647991 ns/op	130858680 B/op	13084734 allocs/op
PASS
ok  	iis-logs-parser	2.620s
//-------------------
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
BenchmarkChunkingWithoutChangingBuf-12    	       1	1330407576 ns/op	     168 B/op	       6 allocs/op
PASS
ok  	iis-logs-parser	1.334s

/==========================/

// Using a 1GB file with 50 MB chunk size
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
BenchmarkChunkingWithChangingBuf-12    	       3	 454040818 ns/op	2306867658 B/op	      49 allocs/op
PASS
ok  	iis-logs-parser	2.744s
//-------------------
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
BenchmarkChunkingWithoutChangingBuf-12    	       2	 543538542 ns/op	52431408 B/op	       5 allocs/op
PASS
ok  	iis-logs-parser	1.726s

/==========================/

// Using 1GB file with 250MB chunk size
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
BenchmarkChunkingWithChangingBuf-12    	       3	 523363010 ns/op	3145728042 B/op	      13 allocs/op
PASS
ok  	iis-logs-parser	3.244s
//-------------------
cpu: 13th Gen Intel(R) Core(TM) i7-1355U
BenchmarkChunkingWithoutChangingBuf-12    	       3	 440493776 ns/op	174764378 B/op	       3 allocs/op
PASS
ok  	iis-logs-parser	1.610s
```

### Trying To Use A Pool Of Workers

I'm currently learning how/if I can use `sync.pools` to improve the performance of the parser.

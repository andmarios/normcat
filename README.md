# normcat

**normcat** will read a file (or stdin) and print it to stdout with
a variable rate following a normal distribution (configurable).

When it reads from a file, it can cycle, so when it reaches the end
of the file, it will start over again, for almost infinite output.

It can understand files compressed with xz, gzip and lz4 and
uncompresses automatically on the fly.

Why? Because I needed to create a couple demos with streaming data.
For example in the graph below, there are 3 normcat instances
piping data to a kafka producer:

![normcat_use](https://user-images.githubusercontent.com/1239679/29490368-2b4aad4a-8542-11e7-973a-223f66087353.png)

Normcat memory usage depends on the average line size. By default normcat will
try to buffer 64*1024 entries (lines of input) into memory. If you prefer
lower memory (buffer just 1024 entries) build with:

    go build -tags lowmem

---

GPLv3 License.

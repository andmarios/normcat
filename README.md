# normcat

**normcat** will read a file (or stdin) and print it to stdout with
a variable rate following a normal distribution (configurable).

When it reads from a file, it can cycle, so when it reaches the end
of the file, it will start over again, for almost infinite output.

It can understand files compressed with xz, gzip and lz4 and
uncompresses automatically on the fly.

Why? Because I needed to create a couple demos with streaming data.


---

GPLv3 License.

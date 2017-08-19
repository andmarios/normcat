// Copyright (c) 2017 Marios Andreopoulos.
//
// This file is part of normcat.
//
// 	Normcat is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// 	Normcat is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// 	You should have received a copy of the GNU General Public License
// along with normat.  If not, see <http://www.gnu.org/licenses/>.

/*
A program to cycle-read lines from a source file (or stdin) and write
them to stdout with a custom rate and jitter following a normal
distribution.
*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

//go:generate go run version-generate/main.go

var (
	cycle      = flag.Bool("cycle", false, "whether to read the file again if we reached the end but haven't read as many lines as requested")
	verbose    = flag.Bool("verbose", false, "print status updates in stderr")
	lines      = flag.Int64("lines", 9223372036854775807, "number of lines to print")
	printers   = flag.Int("workers", 1, "number of workers to print data to stdout, multiple workers are faster but print messages out of order")
	rateLimit  = flag.Int("rate", 1000, "base print rate per sec, should be > 20")
	jitter     = flag.Float64("jitter", 200, "if not 0, print rate will follow a normal distribution with mean=rate and stddev=jitter")
	rateUpdate = flag.Duration("period", 10*time.Second, "time (time.Duration) between rate adjustments if jitter > 0")
	version    = flag.Bool("version", false, "print version and exit")
)

// init here set shorthandles for flags, sets help text, parses the
// filename (dataFile var) and if verbose == false, disables logging.
func init() {
	// Add shorthandles to flags:
	flag.BoolVar(cycle, "c", *cycle, "")
	flag.BoolVar(verbose, "v", *verbose, "")
	flag.Int64Var(lines, "n", *lines, "")
	flag.IntVar(printers, "w", *printers, "")
	flag.IntVar(rateLimit, "r", *rateLimit, "")
	flag.Float64Var(jitter, "j", *jitter, "")
	flag.DurationVar(rateUpdate, "p", *rateUpdate, "")
	flag.BoolVar(version, "V", *version, "")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage: normcat [OPTION]... [FILE]
Read lines from [FILE] or stdin and print to stdout with a rate that follows a
normal distribution. If the file is compressed, normcat will try to decompress
it (formats supported: xz, gz, lz4).

  -n, -lines
        Number of lines to print (default %d).
  -c, -cycle
        If set, normcat will cycle the file until the number of requested lines
        is met. Doesn't work when reading from stdin.
  -r, -rate
        Print lines with this mean rate (default %d).
  -j, -jitter
        Print rate follows a normal distribution with mean=rate, stddev=jitter.
        If jitter is set to 0, then the rate is stable (default %d).
  -w, -workers
        Number of workers to use for printing to stdout. Multiple workers can
        speed things up but lines will be printed out of order (default %d).
  -p, -period
        How often the real rate (not the mean) is updated with a random value.
        (default %s)
  -v, -verbose
        Print information and progress updates to stderr.
  -V, -version
        Print version information.

Report bugs at <https://github.com/andmarios/normcat>
`, *lines, *rateLimit, *jitter, *printers, *rateUpdate)

	}
	flag.Parse()

	if *version {
		fmt.Fprintf(os.Stderr, "normcat %s\n", vgVersion)
		os.Exit(2)
	}

	dataFile = strings.Join(flag.Args(), " ")

	if !*verbose {
		log.SetOutput(ioutil.Discard)
	}
}

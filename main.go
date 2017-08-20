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
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/pierrec/lz4"
	"github.com/ulikunitz/xz"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	filetype "gopkg.in/h2non/filetype.v1"
)

// Flag vars used below:
// cycle bool
// lines int64
// printers int
// rateLimit int
// jitter float
// rateUpdate duration

func init() {
	// Add LZ4 type detection
	var lz4Type = filetype.NewType("lz4", "application/lz4")
	var lz4Func = func(buf []byte) bool {
		return len(buf) > 3 &&
			buf[3] == 0x18 && buf[2] == 0x4D && buf[1] == 0x22 && buf[0] == 0x04
	}
	// If we ever want to add snappy, this is the magic byte
	// 	return len(buf) > 10 && string(buf[0:9]) == "\xff\x06\x00\x00"+"sNaPpY"
	filetype.AddMatcher(lz4Type, lz4Func)
}

var workerWg sync.WaitGroup
var limiter *rate.Limiter
var rateDivider = 1
var dataFile string // set in init() of flags.go

func main() {
	// The dispatchBus will deliver the decoded messages to the workers.
	dispatchBus := make(chan string, 1024*64)

	// Spawn our workers.
	workerWg.Add(*printers)
	for i := 1; i <= *printers; i++ {
		go worker(dispatchBus)
	}

	// We periodically check if we are within our set rate. Doing it for
	// every line we print, takes too much CPU and makes it impossible to
	// get high rates. So instead, as the requested rate gets higher, we
	// check every n messages, where n an arbitrary number set below.
	// E.g for a rate of 70000 lines/sec we check every 11 lines, which will
	// lead to a tiny error but will let us reach the desired rate.
	switch {
	case 5000 <= *rateLimit && *rateLimit < 10000:
		rateDivider = 3
	case 10000 <= *rateLimit && *rateLimit < 50000:
		rateDivider = 7
	case 50000 <= *rateLimit && *rateLimit < 100000:
		rateDivider = 11
	case 100000 <= *rateLimit && *rateLimit < 500000:
		rateDivider = 19
	case 500000 <= *rateLimit && *rateLimit < 1000000:
		rateDivider = 41
	case 1000000 <= *rateLimit:
		rateDivider = 97
	}
	// Here we set rate and jitter according to the chosen divider.
	// Think of a rate of 1,000 msg/sec as 1,000 checks/sec. If we
	// want to check every 5 messages, then for 1,000 msg/sec we
	// should perform 200 checks/sec.
	rateBase := rate.Limit(*rateLimit / rateDivider)
	*jitter = *jitter / float64(rateDivider)
	limiter = rate.NewLimiter(rateBase, (*rateLimit/rateDivider)>>1)

	// To keep track of messages decoded and send to workers.
	numMessages := int64(0)

	// This function sets the rate every period.
	go func() {
		lastMessages := int64(0)
		curMessages := int64(0)
		var jitterLimit rate.Limit
		ticker := time.NewTicker(*rateUpdate)
		rand.Seed(time.Now().UTC().UnixNano())
		for range ticker.C {
			// Set jitter (disabled if jitter == 0)
			jitterLimit = rateBase + rate.Limit(rand.NormFloat64()**jitter)
			if jitterLimit < 0 {
				jitterLimit = 1
			}
			limiter.SetLimit(jitterLimit)

			// no lock for numMessages but meh
			// Even if we read an outdated or future dated value, it isn't
			// very important because it can't affect the rate that much.
			curMessages = numMessages
			log.Printf("Messages sent - %s / total, new rate set at: %d / %d, %.2f msg/sec\n",
				*rateUpdate, curMessages-lastMessages, curMessages, jitterLimit*rate.Limit(rateDivider))
			lastMessages = curMessages
		}
	}()

	// This function reads lines from the source (file, stdin) and dispatches
	// them to workers. It keeps tracks of messages, cycling the source, etc.
	go func() {
		// TODO: we could move this inside the for loop  and always re-opening
		// the file. This would let us cycle over bzip and normcat multiple files
		// but I have to test if it will change the memory profile (or find a way
		// to make it safe). Also not sure what will happen with small files where
		// they may be overhead re-opening them.
		data, reset := streamHandle()

	readLoop:
		for {
			in := bufio.NewScanner(data)
			in.Split(bufio.ScanLines)
			for in.Scan() {
				dispatchBus <- in.Text()
				numMessages++
				if numMessages == *lines {
					break readLoop
				}
			}
			if *cycle != true {
				*lines = numMessages
				//wait <- true
				break readLoop
			}
			// Roll to the start of the file
			reset()

		}
	}()

	for i := int64(0); i < *lines; i++ {
		<-wait
	}
	log.Printf("Finished processing input. %d lines printed.\n", numMessages)

}

var wait = make(chan bool, 1024*60)

// worker just print messages from the dispatch channel to stdout
// checking periodically the rate and waiting accordingly
func worker(msgBus chan string) {
	defer workerWg.Done()

	numMessages := 0
	ctx := context.TODO()
	for msg := range msgBus {
		fmt.Println(msg)
		wait <- true
		if numMessages%rateDivider == 0 {
			limiter.Wait(ctx)
		}
		numMessages++
	}
}

// streamHandle decides whether to read stdin or a file.
// If it's  file, it tests for known compressed file types.
// It returns an io.Reader which can be used to read data
// from whichever source we set. It also returns the reset
// method, which can be used to cycle to the start of the
// file if needed.
func streamHandle() (data io.Reader, reset func()) {
	var source *os.File
	var err error

	// In this case we read from stdin.
	if dataFile == "" {
		source = os.Stdin
		*cycle = false
		log.Println("Starting to process messages from stdin.")
		data, reset = source, func() {}
		return
	}

	// This is the case where we read from a file
	source, err = os.Open(dataFile)
	if err != nil {
		log.Fatalln(err)
	}
	// streamHandle will exit but normcat will continue, so we can't defer close()
	// here. We wouldn't want anyway. As long as normcat runs, source should stay open.
	// defer source.Close()
	log.Printf("Starting to process messages from %s.\n", dataFile)

	// Code to read first 261 bytes of source and detect filetype.
	var header [261]byte
	_, err = source.Read(header[0:261])
	if err != nil {
		log.Println(err)
	}

	t, err := filetype.Get(header[0:261])
	if err != nil {
		log.Println(err)
	}

	// Return to the start of the file
	source.Seek(0, 0)

	// Depending on extension, set proper data and reset functions.
	switch t.Extension {
	case "xz":
		log.Println("Using xz")
		data, err = xz.NewReader(source)
		reset = func() { source.Seek(0, 0) }
	// bzip2 is disabled because I couldn't find a way to reset the reader
	// case "bz2":
	// 	log.Println("Using bz2")
	// 	data = bzip2.NewReader(source)
	// 	reset = func() { source.Seek(0, 0) }
	case "gz":
		log.Println("Using gz")
		data, err = gzip.NewReader(source)
		reset = func() {
			source.Seek(0, 0)
			d, _ := data.(*gzip.Reader)
			d.Reset(source)
		}
	case "lz4":
		log.Println("Using lz4")
		data = lz4.NewReader(source)
		reset = func() {
			source.Seek(0, 0)
			d, _ := data.(*lz4.Reader)
			d.Reset(source)
		}
	// case "zlib":
	// 	log.Println("Using zlib")
	// 	data, err = zlib.NewReader(source)
	// 	reset = func() {
	// 		source.Seek(0, 0)
	// 		d, _ := data.(zlib.Resetter)
	// 		d.Reset(source, nil)
	// 	}
	default:
		log.Println("Uncompressed stream")
		err = nil
		data = source
		reset = func() { source.Seek(0, 0) }
	}
	if err != nil {
		log.Println(err)
	}

	return
}

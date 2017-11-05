// +build lowmem

package main

// With only 1024 entries buffer, normcat is a bit slower but, depending on
// your average line size may use much less memory.
// How much is the difference? For a file with an average line size of 640 bytes,
// normcat was 10% slower but used 20% of RAM (about 10MB in total).
var chanBuffer = 1024 * 1

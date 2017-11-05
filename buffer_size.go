// +build !lowmem

package main

// The default buffer size gives good performance but, depending on
// your average line size may use more memory.
// How much is the difference? For a file with an average line size of 640 bytes,
// normcat was 10% faster but used 500% RAM (about 50MB in total).
var chanBuffer = 1024 * 60

#!/bin/env bash

function verify() {
    RES="$(./normcat ${@:2} | tr '\n' ' ')"
    [[ "$RES" == "$1" ]] && echo "  SUCCESS." || echo "  FAIL (got '$RES' instead of '$1')"
}
go build

echo -n "Test normcat printing whole file:"
verify "1 2 3 " tests/test.txt

echo -n "Test normcat printing part of file:"
verify "1 2 " -n 2 tests/test.txt

echo -n "Test normcat cycling file:"
verify "1 2 3 1 2 " -n 5 -c tests/test.txt

echo -n "Test normcat cycling xz file:"
verify "1 2 3 1 2 " -n 5 -c tests/test.txt.xz

echo -n "Test normcat cycling gzip file:"
verify "1 2 3 1 2 " -n 5 -c tests/test.txt.xz

echo -n "Test normcat cycling lz4 file:"
verify "1 2 3 1 2 " -n 5 -c tests/test.txt.xz

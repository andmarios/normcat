#!/bin/env bash
set -e
set -u

VERSION="$(git describe --tags)"
EMAIL="$(git config --global user.email)"

mkdir -p _release
rm -rf _release/*

go get -v
go generate
GOOS=windows GOARCH=amd64 go build
zip _release/normcat-$VERSION-windows-amd64.zip normcat.exe
GOOS=windows GOARCH=amd64 go build -tags lowmem
zip _release/normcat-$VERSION-windows-amd64-lowmem.zip normcat.exe

GOOS=darwin GOARCH=amd64 go build
zip _release/normcat-$VERSION-darwin-amd64.zip normcat
GOOS=darwin GOARCH=amd64 go build -tags lowmem
zip _release/normcat-$VERSION-darwin-amd64-lowmem.zip normcat

GOOS=linux GOARCH=amd64 go build
tar czf _release/normcat-$VERSION-linux-amd64.tar.gz normcat
GOOS=linux GOARCH=amd64 go build -tags lowmem
tar czf _release/normcat-$VERSION-linux-amd64-lowmem.tar.gz normcat

cd _release/
gpg -u "$EMAIL" --armor --detach-sign normcat-$VERSION-darwin-amd64.zip
gpg -u "$EMAIL" --armor --detach-sign normcat-$VERSION-darwin-amd64-lowmem.zip
gpg -u "$EMAIL" --armor --detach-sign normcat-$VERSION-linux-amd64.tar.gz
gpg -u "$EMAIL" --armor --detach-sign normcat-$VERSION-linux-amd64-lowmem.tar.gz
gpg -u "$EMAIL" --armor --detach-sign normcat-$VERSION-windows-amd64.zip
gpg -u "$EMAIL" --armor --detach-sign normcat-$VERSION-windows-amd64-lowmem.zip
cd ../

rm normcat.exe normcat

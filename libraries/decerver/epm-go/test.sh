#!/bin/sh
cd $GOPATH/src/github.com/eris-ltd/epm-go

# run the go unit tests
cd epm && go test -v ./... -race
cd ../chains && go test -v ./... -race
cd ../utils && go test -v ./... -race
cd ../cmd/epm && go test -v ./... -race # these don't exist yet

# run the base pdx deploy test
cd ../tests && go test -v ./... -race

# test suite of eris-std-lib deploys
cd $GOPATH/src/github.com/eris-ltd/eris-std-lib/DTT/tests
./test.sh


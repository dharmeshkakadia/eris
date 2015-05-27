#!/bin/sh
cd $GOPATH/src/github.com/eris-ltd/thelonious
go test -v ./... -race

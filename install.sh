#!/bin/bash

CURDIR=`pwd`
GOPATH=${CURDIR}
export GOPATH
go version
go build -x -o gearmand ./src/gearman/main.go
rm -rf ${CURDIR}/bin/gearmand
mv gearmand ./bin

echo 'finished'
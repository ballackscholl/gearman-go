#!/bin/bash

if [ ! -f install.sh ]; then
    echo 'install must be run within its container folder' 1>&2
    exit 1
fi

CURDIR=`pwd`
GOPATH=${CURDIR}
export GOPATH
go build -x -o gearmand ./src/gearman/main.go
rm -rf ${CURDIR}/bin/gearmand
mv gearmand ./bin

echo 'finished'
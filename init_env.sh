#!/usr/bin/env bash

echo Setting GOPATH ...
DIR=`pwd`
if [ ${DIR} != ${GOPATH} ]
then
    export OLDGOPATH=$GOPATH
    export GOPATH=$GOPATH:$DIR
    echo "Set GOPATH=${GOPATH}"
fi
echo Finished!
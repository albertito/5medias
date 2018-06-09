#!/bin/bash

set -e
source "$(dirname ${0})/common.sh"

if [ "${RACE}" == "1" ]; then
	export GOFLAGS="$GOFLAGS -race"
fi

( cd ../; go build $GOFLAGS -tags="$GOTAGS" )

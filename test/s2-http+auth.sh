#!/bin/bash

set -e
source "$(dirname ${0})/common.sh"

busybox httpd -f -p 9001 > .http.log 2>&1 &
../5medias --allow_loopback --username=user --password=test 2> .5medias.log &

wait_until_ready 9001
wait_until_ready 1080

curl -s -S --preproxy "socks5://user:test@localhost:1080/" \
		http://localhost:9001/.random \
		> .curl.out

if ! diff -q .random .curl.out; then
	echo proxied content differs
	exit 1
fi

if curl -s --preproxy "socks5://localhost:1080/" \
		http://localhost:9001/.random \
		> /dev/null
then
	echo proxy worked without auth
	exit 1
fi

echo success

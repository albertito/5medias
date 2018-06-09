#!/bin/bash
set -e

cd "$(realpath `dirname ${0}`)"

echo build
setsid -w ./build.sh
echo

for i in s*.sh; do
	echo $i
	if ! setsid -w ./$i; then
		echo "FAILED"
		exit 1
	fi
	echo
done

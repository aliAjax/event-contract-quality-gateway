#!/bin/sh
set -eu
find . -name '*.go' -not -name '*_test.go' -not -path './vendor/*' -print0 | xargs -0 wc -l | tail -n 1

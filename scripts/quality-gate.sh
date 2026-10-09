#!/usr/bin/env bash
set -euo pipefail

make validate
if [[ "$(uname -s)" == "Linux" ]]; then
    xvfb-run -a make test-render
else
    make test-render
fi
make web-build
make web-test-browser

if [[ "$(uname -s)" == "Linux" ]]; then
    xvfb-run -a go test -count=1 -run=. -skip='^$' -list= -bench= -fuzz= -short=false -p=1 -parallel=1 -race ./internal/debugging/...
else
    go test -count=1 -run=. -skip='^$' -list= -bench= -fuzz= -short=false -p=1 -parallel=1 -race ./internal/debugging/...
fi

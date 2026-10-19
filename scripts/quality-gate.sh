#!/usr/bin/env bash
set -euo pipefail

stage_name=
stage_started=0

finish_stage() {
    local status=$1
    if [[ -n "$stage_name" ]]; then
        printf 'STAGE END %s status=%s DURATION=%ss\n' \
            "$stage_name" "$status" "$((SECONDS - stage_started))" >&2
        stage_name=
    fi
}

trap 'finish_stage "$?"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

run_stage() {
    stage_name=$1
    shift
    stage_started=$SECONDS
    printf 'STAGE START %s\n' "$stage_name" >&2
    "$@"
    finish_stage 0
}

# Keep the canonical validation graph in one Make invocation. Stage reporting
# wraps existing commands and never reruns a check to obtain its timing.
run_stage validate make validate
if [[ "$(uname -s)" == "Linux" ]]; then
    run_stage native xvfb-run -a make test-render
else
    run_stage native make test-render
fi
run_stage frontend make web-build
run_stage browser make web-test-browser

if [[ "$(uname -s)" == "Linux" ]]; then
    run_stage race xvfb-run -a go test -json -count=1 -run=. -skip='^$' -list= -bench= -fuzz= -short=false -p=1 -parallel=1 -race ./internal/debugging/... ./scripts/check-audio-module
else
    run_stage race go test -json -count=1 -run=. -skip='^$' -list= -bench= -fuzz= -short=false -p=1 -parallel=1 -race ./internal/debugging/... ./scripts/check-audio-module
fi

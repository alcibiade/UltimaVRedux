#!/usr/bin/env bash
# Drive Ultima V Redux headlessly: Xvfb display + sandboxed HOME + xdotool keys + ImageMagick screenshots.
# Run from the repository root. See SKILL.md next to this file.
set -euo pipefail

REPO=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
STATE=${URDX_STATE:-${XDG_RUNTIME_DIR:-/tmp}/ultimavredux-driver}
DISPNUM=${URDX_DISPLAY:-99}
DATA=${U5_DATA:-$HOME/UltimaV}
SANDBOX_HOME="$STATE/home"
SHOTS="$STATE/shots"
HOLD=${URDX_HOLD:-0.12}   # seconds a key is held; Ebiten polls once per frame, instant press+release gets dropped
GAP=${URDX_GAP:-0.35}     # pause after each key

export DISPLAY=":$DISPNUM"
unset WAYLAND_DISPLAY

die() { echo "driver: $*" >&2; exit 1; }
pidof_file() { [ -f "$STATE/$1.pid" ] && kill -0 "$(cat "$STATE/$1.pid")" 2>/dev/null && cat "$STATE/$1.pid"; }
window() {
    local w
    w=$(xdotool search --name '^Ultima V Redux$' 2>/dev/null | head -1) || true
    [ -n "$w" ] || die "no game window on $DISPLAY (is it started? see '$0 log')"
    echo "$w"
}

press() {
    local w; w=$(window)
    for k in "$@"; do
        xdotool keydown --window "$w" "$k"
        sleep "$HOLD"
        xdotool keyup --window "$w" "$k"
        sleep "$GAP"
    done
}

shot() {
    local name=${1:-shot-$(date +%H%M%S)} w
    w=$(window)
    mkdir -p "$SHOTS"
    import -window "$w" "$SHOTS/$name.png"
    echo "$SHOTS/$name.png"
}

start() {
    local fresh=0
    if [ "${1:-}" = "--fresh" ]; then fresh=1; shift; fi
    [ "${1:-}" = "--" ] && shift

    [ -f "$DATA/BRIT.DAT" ] || die "no original DOS game files in $DATA (set U5_DATA; copy Original-Release/IBM-PC/ultima5 there)"
    [ -x "$REPO/bin/ultimav" ] || { echo "driver: building bin/ultimav"; (cd "$REPO" && make build); }
    [ -z "$(pidof_file game || true)" ] || die "already running (pid $(cat "$STATE/game.pid")); '$0 stop' first"

    [ "$fresh" = 1 ] && rm -rf "$SANDBOX_HOME"
    mkdir -p "$SANDBOX_HOME/.ultima_v_redux" "$SHOTS"
    # A config with only datafilepath skips the first-run "Game File Setup" screens.
    [ -f "$SANDBOX_HOME/.ultima_v_redux/config.yaml" ] ||
        printf 'datafilepath: %s\n' "$DATA" > "$SANDBOX_HOME/.ultima_v_redux/config.yaml"

    if [ -z "$(pidof_file xvfb || true)" ]; then
        [ -e "/tmp/.X11-unix/X$DISPNUM" ] && die "display :$DISPNUM is taken by something else; set URDX_DISPLAY"
        setsid Xvfb ":$DISPNUM" -screen 0 1280x960x24 -nolisten tcp > "$STATE/xvfb.log" 2>&1 &
        echo $! > "$STATE/xvfb.pid"
        for _ in $(seq 50); do [ -e "/tmp/.X11-unix/X$DISPNUM" ] && break; sleep 0.1; done
    fi

    (cd "$REPO" && HOME="$SANDBOX_HOME" setsid ./bin/ultimav -no-sound "$@" > "$STATE/game.log" 2>&1 &
     echo $! > "$STATE/game.pid")
    for _ in $(seq 100); do
        xdotool search --name '^Ultima V Redux$' >/dev/null 2>&1 && { sleep 2; echo "driver: game up on $DISPLAY, state in $STATE"; return; }
        kill -0 "$(cat "$STATE/game.pid")" 2>/dev/null || { tail -20 "$STATE/game.log"; die "game exited during startup"; }
        sleep 0.2
    done
    die "game window never appeared; see $STATE/game.log"
}

# The title menu's blue frame covers this pixel; it is black during the intro and in gameplay.
on_title_menu() {
    [ "$(import -window "$(window)" -format '%[pixel:p{34,560}]' info: 2>/dev/null)" = "srgb(0,10,173)" ]
}

# Title screen -> gameplay. Imports the data dir's SAVED.GAM into slot 1 the first time; afterwards "Journey Onward" loads it.
enter_game() {
    # Only Enter skips the intro (REDUX DEVIATION INTRO-0003); other keys are ignored until the menu shows.
    for _ in $(seq 20); do
        on_title_menu && break
        press Return
        sleep 1
    done
    on_title_menu || die "title menu never appeared; '$0 shot' to see where it is"
    if ls "$SANDBOX_HOME/.ultima_v_redux/saves/"slot-*.json >/dev/null 2>&1; then
        press Return        # Journey Onward (newest save) is selected by default once a save exists
    else
        [ -f "$DATA/SAVED.GAM" ] || die "no save in sandbox and no $DATA/SAVED.GAM to import"
        press Down Return   # Import from Legacy
        sleep 1
        press Return        # target slot 1
        sleep 0.5
        press Return        # empty description -> import and start playing
    fi
    sleep 3
}

stop() {
    for p in game xvfb; do
        local pid; pid=$(pidof_file "$p" || true)
        [ -n "$pid" ] && kill "$pid" && echo "driver: stopped $p ($pid)"
        rm -f "$STATE/$p.pid"
    done
}

cmd=${1:-help}; shift || true
mkdir -p "$STATE"
case "$cmd" in
    start)      start "$@" ;;
    enter-game) enter_game ;;
    key)        press "$@" ;;
    type)       xdotool type --window "$(window)" --delay 120 "$*"; sleep "$GAP" ;;
    wait)       sleep "${1:-1}" ;;
    shot)       shot "${1:-}" ;;
    log)        tail -n "${1:-40}" "$STATE/game.log"; echo "--- redux.log"; tail -n "${1:-40}" "$SANDBOX_HOME/.ultima_v_redux/redux.log" 2>/dev/null || true ;;
    status)     echo "game: $(pidof_file game || echo down)  xvfb: $(pidof_file xvfb || echo down)  display: $DISPLAY  state: $STATE" ;;
    stop)       stop ;;
    *)          sed -n '2,3p' "$0"; echo "usage: $0 start [--fresh] [-- game-args] | enter-game | key K... | type TEXT | wait S | shot [NAME] | log [N] | status | stop"; exit 2 ;;
esac

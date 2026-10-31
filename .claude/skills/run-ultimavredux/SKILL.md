---
name: run-ultimavredux
description: Build, run, launch, start, play, drive or screenshot Ultima V Redux (the Go + Ebiten game in this repo) headlessly - send keys to the real game window, reach gameplay from a save, capture PNG screenshots, or open the debug workbench. Use when asked to run the game, see a change working in-game, or take a screenshot.
---

# Run Ultima V Redux

The game is a desktop Ebiten window. Agents drive it with `driver.sh` (next to this file): it runs the game on a
private Xvfb display with a sandboxed `HOME`, sends held keys with `xdotool`, and saves screenshots with ImageMagick.
Nothing appears on the user's desktop and the user's `~/.ultima_v_redux` (config, saves) is never touched.

All paths below are relative to the repository root. `D=.claude/skills/run-ultimavredux/driver.sh`.

## Prerequisites (Manjaro/Arch)

```bash
sudo pacman -S --needed go xorg-server-xvfb xdotool imagemagick
```

`go.mod` wants Go 1.27 (`toolchain go1.27.1`); the `extra` repo ships 1.27.1.

The game needs the original **MS-DOS** Ultima V files (`BRIT.DAT`, `SAVED.GAM`, ...). The driver reads them from
`$U5_DATA` (default `~/UltimaV`). In this workspace they come from the knowledge base's untouchable
`Original-Release/` - copy, never point at it:

```bash
cp -a ../Original-Release/IBM-PC/ultima5 ~/UltimaV && chmod -R u+w ~/UltimaV
```

## Build

```bash
make build          # -> bin/ultimav (~18 s cold)
./bin/ultimav --version
```

`driver.sh start` builds automatically if `bin/ultimav` is missing, but **not** if it is stale - rebuild after edits.

## Run (agent path)

```bash
D=.claude/skills/run-ultimavredux/driver.sh
$D start --fresh            # Xvfb :99 + game; --fresh wipes the sandbox (saves, config)
$D enter-game               # intro -> title menu -> gameplay (see below)
$D shot 01-game             # prints /run/user/1000/ultimavredux-driver/shots/01-game.png
$D key Down Down Right      # each key held 120 ms; xdotool keysym names (Return, Escape, Up, a, ctrl+u ...)
$D type 'some text'         # text fields (save descriptions, conversation input)
$D wait 2
$D shot 02-moved            # then Read the PNG and LOOK at it
$D log                      # game stdout + sandbox redux.log
$D status
$D stop
```

`enter-game`: presses Enter until the title menu shows (detected by a blue pixel), then

- first run in a sandbox: **Import from Legacy** -> slot 1 -> empty description. Imports `$U5_DATA/SAVED.GAM`;
  the stock save starts in **Iolo's Hut** with Elwood, Shamino and Iolo.
- later runs (`start` without `--fresh`): **Journey Onward** loads slot 1.

Extra game flags go after `--`, e.g. the browser debug workbench (state inspection/commands over HTTP + WebSocket):

```bash
$D start -- -debug-workbench
curl -s http://127.0.0.1:8585/healthz      # {"status":"ok"}
```

Knobs (env): `U5_DATA`, `URDX_DISPLAY` (default 99), `URDX_STATE` (default `$XDG_RUNTIME_DIR/ultimavredux-driver`),
`URDX_HOLD` / `URDX_GAP` (key hold / pause, seconds).

## Run (human path)

```bash
make dev            # build + open the window on the real desktop
make run            # run the existing bin/ultimav
```

Uses the real `~/.ultima_v_redux/config.yaml`; a first run with no `datafilepath` shows "Game File Setup" and then an
"Ultima V Found" confirmation that a human must accept with Enter.

## Test

```bash
go test ./cmd/ultimav/ -run 'IntroMenu' -count=1    # ~11 s; title/intro scene tests
make test                                           # whole suite (long)
```

## Gotchas

- **Only Enter skips the intro.** Escape, arrows and letters are ignored until the title menu appears
  (`cmd/ultimav/intro_menu_scene.go`, REDUX DEVIATION INTRO-0003). Pressing Down "too early" silently does nothing,
  and the next Enter then skips the intro instead of choosing a menu item - the menu ends up one step off.
- **`xdotool key` gets dropped.** Ebiten polls the keyboard once per frame; an instant press+release often falls
  between frames. The driver uses `keydown` / 120 ms / `keyup`. Raise `URDX_HOLD` if keys still go missing.
- **The title menu goes to an attract view after 10 s idle** ("Return to the View"). Don't park on the menu.
- **First-run setup screens** appear whenever `config.yaml` has no `datafilepath`. The driver writes a one-line
  `config.yaml` into its sandbox `HOME` (`datafilepath: $U5_DATA`), which skips them entirely.
- The game resolves its config via `os.UserHomeDir()`, so `HOME=<sandbox>` isolates config, saves, logs and cache.
- The driver unsets `WAYLAND_DISPLAY` and sets `DISPLAY=:99` so the window cannot land on the user's desktop.
- Window size under Xvfb is 1152x720; screenshots are of the game window only (`import -window <id>`).
- A legacy import logs `Legacy save import completed with 7 conversion warnings` - normal for the stock save.
- Stop processes with `$D stop` (pid files). `pkill -f '<pattern>'` from an agent shell also matches the shell's own
  command line and kills it (exit 144).

## Troubleshooting

| Symptom | Fix |
|---|---|
| `make: go: No such file or directory` | `sudo pacman -S go` |
| Screenshot shows "Game File Setup" / "No complete Ultima V installation" | `$U5_DATA` lacks the DOS files, or the config lacks `datafilepath`; copy the files (Prerequisites) and `start --fresh` |
| Screenshot shows "Select Save Game" with all slots Empty after `enter-game` | A key was swallowed by the intro; `$D key Escape`, then `$D key Down Return Return Return` |
| `bin/ultimav: permission denied` | Exec bit lost (ownCloud sync); `make build` or `chmod +x bin/ultimav` |
| `driver: display :99 is taken` | Another X server owns :99; `URDX_DISPLAY=98 $D start` |

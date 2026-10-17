# Arch Linux / GNOME notes

Changes in this fork compared to upstream:

- `ui/main.cpp`: when the compositor does not offer layer-shell (e.g. GNOME),
  the app no longer exits with an error. It falls back to a borderless,
  always-on-top window running through XWayland.
- `go.mod`: a2kit bumped to v0.4.0. With the previous pinned commit, combat
  events were not decoded in my setup.

## Usage

```bash
scripts/setup-arch.sh   # installs dependencies and builds
scripts/run.sh          # starts SlopMeter
```

`setup-arch.sh` adds your user to the `wireshark` group so `dumpcap` can run
without root. Log out and back in afterwards for the group to apply.

## Start-up order

1. Close the game and its launcher.
2. Start SlopMeter.
3. Start the launcher, go through the server list and character selection,
   and enter the world.

If the capture starts after the session is already established, the decoder
cannot pick it up and the meter stays empty.

## Known limitations

- Automatic game-display detection relies on KWin and does not work on GNOME.
  Choose the display from the tray menu instead.
- GNOME needs an AppIndicator extension to show the tray icon.
- Under XWayland the overlay may be hidden if the game runs in exclusive
  fullscreen. Use windowed/borderless mode.
- Tested only on one Arch + GNOME (Wayland) machine.

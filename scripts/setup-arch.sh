#!/usr/bin/env bash
# Arch Linux setup: installs dependencies, joins the wireshark group and builds.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> Installing dependencies (pacman)"
sudo pacman -S --needed base-devel git go libpcap cmake pkgconf \
  qt6-base qt6-declarative qt6-svg qt6-imageformats qt6-wayland layer-shell-qt wayland \
  nodejs python wireshark-cli polkit xorg-xwayland

if [[ "${XDG_CURRENT_DESKTOP:-}" == *GNOME* ]]; then
  echo "==> GNOME: installing the AppIndicator extension (tray icon)"
  sudo pacman -S --needed gnome-shell-extension-appindicator || true
  gnome-extensions enable appindicatorsupport@rgcjonas.gmail.com 2>/dev/null \
    || echo "   Enable it after logging out and back in."
fi

NEED_RELOGIN=0
if ! getent group wireshark | grep -qE "[:,]${USER}(,|$)"; then
  echo "==> Adding $USER to the wireshark group (lets dumpcap run without root)"
  sudo usermod -aG wireshark "$USER"
  NEED_RELOGIN=1
elif ! id -nG | tr ' ' '\n' | grep -qx wireshark; then
  NEED_RELOGIN=1
fi

echo "==> Building"
bash scripts/build.sh

echo
echo "Done. Run: scripts/run.sh"
if [[ $NEED_RELOGIN -eq 1 ]]; then
  echo "IMPORTANT: log out and back in so the wireshark group takes effect."
fi

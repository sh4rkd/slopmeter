#!/usr/bin/env bash
# Setup de SlopMeter en Arch Linux, incluido GNOME/Wayland sin layer-shell
# (el overlay usa el modo de respaldo con XWayland).
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> Instalando dependencias (pacman)"
sudo pacman -S --needed base-devel git go libpcap cmake pkgconf \
  qt6-base qt6-declarative qt6-svg qt6-imageformats qt6-wayland layer-shell-qt wayland \
  nodejs python wireshark-cli polkit xorg-xwayland

if [[ "${XDG_CURRENT_DESKTOP:-}" == *GNOME* ]]; then
  echo "==> GNOME: extensión de bandeja (AppIndicator)"
  sudo pacman -S --needed gnome-shell-extension-appindicator || true
  gnome-extensions enable appindicatorsupport@rgcjonas.gmail.com 2>/dev/null \
    || echo "   Actívala después de reiniciar sesión."
fi

NEED_RELOGIN=0
if ! getent group wireshark | grep -qE "[:,]${USER}(,|$)"; then
  echo "==> Agregando $USER al grupo wireshark (permite capturar sin root)"
  sudo usermod -aG wireshark "$USER"
  NEED_RELOGIN=1
elif ! id -nG | tr ' ' '\n' | grep -qx wireshark; then
  NEED_RELOGIN=1
fi

echo "==> Compilando"
bash scripts/build.sh

echo
echo "Listo. Ejecuta: scripts/run.sh"
if [[ $NEED_RELOGIN -eq 1 ]]; then
  echo "IMPORTANTE: cierra sesión y vuelve a entrar para que el grupo wireshark se aplique."
fi

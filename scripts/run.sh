#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

if ! id -nG | tr ' ' '\n' | grep -qx wireshark; then
  echo "Your session is not in the wireshark group: capture will not work."
  echo "Log out and back in (or run scripts/setup-arch.sh)."
  exit 1
fi
if [[ ! -x build/ui/slopmeter || ! -x build/slopmeter-capture ]]; then
  echo "Not built yet: run scripts/setup-arch.sh"
  exit 1
fi

# Allow the XWayland fallback on compositors without layer-shell (e.g. GNOME).
export SLOPMETER_X11_FALLBACK="${SLOPMETER_X11_FALLBACK:-1}"

echo "Reminder: start SlopMeter BEFORE the game launcher, then go through the"
echo "server list and character selection so your session is detected."
exec ./build/ui/slopmeter "$@"

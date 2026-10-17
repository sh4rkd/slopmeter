#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

if ! id -nG | tr ' ' '\n' | grep -qx wireshark; then
  echo "Tu sesión no tiene el grupo wireshark: la captura no va a funcionar."
  echo "Cierra sesión y vuelve a entrar (o ejecuta scripts/setup-arch.sh)."
  exit 1
fi
if [[ ! -x build/ui/slopmeter || ! -x build/slopmeter-capture ]]; then
  echo "Falta compilar: ejecuta scripts/setup-arch.sh"
  exit 1
fi
echo "Recuerda: abre SlopMeter ANTES del launcher del juego y pasa por la"
echo "lista de servidores y selección de personaje para que detecte tu sesión."
exec ./build/ui/slopmeter "$@"

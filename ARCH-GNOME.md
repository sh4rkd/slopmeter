# Notas para Arch Linux / GNOME

Cambios de este fork respecto al original:

- `ui/main.cpp`: si el compositor no ofrece layer-shell (GNOME), en vez de
  salir con error usa una ventana sin bordes y siempre encima vía XWayland.
- `go.mod`: a2kit actualizado a v0.4.0. Con el commit anterior no se
  decodificaban eventos de combate.

Uso: `scripts/setup-arch.sh` y luego `scripts/run.sh`.

Orden de arranque: abrir SlopMeter, luego el launcher, y pasar por la lista de
servidores y la selección de personaje (si no, no detecta la sesión).

Limitaciones: la detección automática del monitor del juego usa KWin y no
funciona en GNOME (elige el monitor desde la bandeja). En XWayland el overlay
puede quedar tapado si el juego está en pantalla completa exclusiva.

#!/usr/bin/env bash
# Instala o retira el servicio de usuario que procesa las peticiones de
# actualización del host. Es idempotente y no ejecuta sudo.
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
REPO=$(cd "$SCRIPT_DIR/.." && pwd -P)
UNIT_SRC="$REPO/deploy/systemd"
UNIT_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
SYSTEMCTL_BIN=${SYSTEMCTL_BIN:-systemctl}
LOGINCTL_BIN=${LOGINCTL_BIN:-loginctl}
RUN_DIR=${RUN_DIR:-$REPO/run}
BACKUP_DIR=${BACKUP_DIR:-$HOME/cinecat-backups}
CACHE_DIR=${CACHE_DIR:-$HOME/.cache/cinecat-tmp}
USER_NAME=${USER:-$(id -un)}

usage() {
	printf 'Uso: %s [--desinstalar]\n' "$(basename "$0")"
}

install_units() {
	mkdir -p "$UNIT_DIR" "$RUN_DIR" "$BACKUP_DIR" "$CACHE_DIR"

	local unit
	for unit in cinecat-update.service cinecat-update.path; do
		local source="$UNIT_SRC/$unit"
		local target="$UNIT_DIR/$unit"
		if [ ! -f "$source" ]; then
			printf 'Falta la plantilla %s\n' "$source" >&2
			exit 1
		fi
		local content
		content=$(cat "$source")
		content=${content//@REPO@/$REPO}
		printf '%s\n' "$content" >"$target"
	done
	chmod +x "$REPO/scripts/cinecat-updater.sh"

	if ! "$SYSTEMCTL_BIN" --user daemon-reload; then
		printf 'No se pudo recargar systemd; ejecuta systemctl --user daemon-reload cuando esté disponible.\n' >&2
	fi
	if ! "$SYSTEMCTL_BIN" --user enable --now cinecat-update.path; then
		printf 'No se pudo activar cinecat-update.path; revisa el estado con systemctl --user status cinecat-update.path.\n' >&2
	fi

	local linger
	linger=$("$LOGINCTL_BIN" show-user "$USER_NAME" -p Linger 2>/dev/null || true)
	if [[ "$linger" != *"Linger=yes"* ]]; then
		if ! "$LOGINCTL_BIN" enable-linger "$USER_NAME" >/dev/null 2>&1; then
			printf 'Para que el servicio funcione sin sesión abierta, ejecuta:\n  sudo loginctl enable-linger %s\n' "$USER_NAME"
		fi
	fi

	printf '\nActualizador instalado.\n'
	printf '  Repositorio: %s\n' "$REPO"
	printf '  Unidades:    %s\n' "$UNIT_DIR"
	printf '  Peticiones:  %s\n' "$RUN_DIR"
	printf '  Copias:      %s\n' "$BACKUP_DIR"
	printf '  Caché:       %s\n\n' "$CACHE_DIR"
	local path_state service_state
	path_state=$("$SYSTEMCTL_BIN" --user is-active cinecat-update.path 2>/dev/null || true)
	service_state=$("$SYSTEMCTL_BIN" --user is-active cinecat-update.service 2>/dev/null || true)
	printf 'Estado: path=%s, service=%s\n\n' "${path_state:-desconocido}" "${service_state:-desconocido}"
	printf '  %s --user status cinecat-update.path\n  %s --user status cinecat-update.service\n  journalctl --user -u cinecat-update\n' \
		"$SYSTEMCTL_BIN" "$SYSTEMCTL_BIN"
}

uninstall_units() {
	"$SYSTEMCTL_BIN" --user disable --now cinecat-update.path >/dev/null 2>&1 || true
	"$SYSTEMCTL_BIN" --user stop cinecat-update.service >/dev/null 2>&1 || true
	rm -f "$UNIT_DIR/cinecat-update.path" "$UNIT_DIR/cinecat-update.service"
	"$SYSTEMCTL_BIN" --user daemon-reload >/dev/null 2>&1 || true
	printf 'Actualizador desinstalado. Las copias y el registro se conservan en %s.\n' "$BACKUP_DIR"
}

case "${1:-}" in
'')
	install_units
	;;
--desinstalar)
	uninstall_units
	;;
-h | --help)
	usage
	;;
*)
	usage >&2
	exit 2
	;;
esac

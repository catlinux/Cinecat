#!/usr/bin/env bash
# Pruebas del actualizador del host con repositorios y comandos falsos.
# No usa Docker, red ni el sistema real.
set -euo pipefail

TEST_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
REPO_ROOT=$(cd "$TEST_DIR/../.." && pwd -P)
UPDATER="$REPO_ROOT/scripts/cinecat-updater.sh"
GIT_BIN=${GIT_BIN:-git}
if ! command -v "$GIT_BIN" >/dev/null 2>&1; then
	# En WSL puede faltar git aunque exista el Git de Windows. Se vuelve a
	# lanzar la prueba con su bash, que sí lo tiene.
	GIT_BASH="/mnt/c/Program Files/Git/bin/bash.exe"
	if [ -x "$GIT_BASH" ] && command -v wslpath >/dev/null 2>&1; then
		windows_path=$(wslpath -w "$REPO_ROOT/scripts/tests/updater_test.sh")
		git_path=$(printf '%s' "$windows_path" | sed 's|\\|/|g; s|^C:|/c|')
		exec "$GIT_BASH" "$git_path" "$@"
	fi
	printf 'No se encontró git; no se pueden ejecutar las pruebas.\n' >&2
	exit 1
fi
if ! command -v flock >/dev/null 2>&1; then
	printf 'Aviso: flock no está disponible; se probará el candado de respaldo.\n' >&2
fi

TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/cinecat-updater-test.XXXXXX")
FAKE_BIN="$TEST_ROOT/bin"
mkdir -p "$FAKE_BIN"
PASS=0
FAIL=0
CURRENT_CASE=

cleanup() {
	if [ -n "${CURRENT_CASE:-}" ] && [ -d "$CURRENT_CASE" ]; then
		rm -rf "$CURRENT_CASE"
	fi
	rm -rf "$TEST_ROOT"
}
trap cleanup EXIT

ok() {
	PASS=$((PASS + 1))
	printf '  ok: %s\n' "$1"
}

bad() {
	FAIL=$((FAIL + 1))
	printf '  FALLO: %s\n' "$1" >&2
}

assert_eq() {
	local want=$1
	local got=$2
	local label=$3
	if [ "$want" = "$got" ]; then
		ok "$label"
	else
		bad "$label (se esperaba %q, se obtuvo %q)" "$want" "$got"
	fi
}

assert_contains() {
	local needle=$1
	local file=$2
	local label=$3
	if [ -f "$file" ] && grep -Fq "$needle" "$file"; then
		ok "$label"
	else
		bad "$label"
	fi
}

assert_file() {
	if [ -f "$1" ]; then
		ok "$2"
	else
		bad "$2"
	fi
}

assert_no_file() {
	if [ ! -e "$1" ]; then
		ok "$2"
	else
		bad "$2"
	fi
}

cat >"$FAKE_BIN/docker" <<'FAKE'
#!/usr/bin/env bash
set -u
printf 'docker %s\n' "$*" >>"${FAKE_CALLS:?}"
case "${1:-}" in
info)
	sleep "${FAKE_DOCKER_INFO_DELAY:-0}"
	printf '%s\n' "${FAKE_DOCKER_ROOT:?}"
	;;
cp)
	source_path=${2#*:}
	relative=${source_path#/data/}
	source_file="${FAKE_CONTAINER_DATA:?}/$relative"
	[ -f "$source_file" ] || exit 1
	mkdir -p "$(dirname "$3")"
	cp "$source_file" "$3"
	;;
compose)
	case "${2:-}" in
	build)
		sleep "${FAKE_DOCKER_DELAY:-0}"
		[ "${FAKE_DOCKER_FAIL_BUILD:-0}" = "1" ] && exit 1
		;;
	up)
		sleep "${FAKE_DOCKER_DELAY:-0}"
		if [ "${FAKE_DOCKER_KEEP_VERSION:-0}" != "1" ] && [ -n "${FAKE_VERSION_FILE:-}" ]; then
			tr -d '[:space:]' <"${CINECAT_DIR:?}/VERSION" >"$FAKE_VERSION_FILE"
		fi
		;;
	esac
	;;
inspect)
	printf '%s\n' "${FAKE_SERVER_IMAGE:-cinecat-servidor}"
	;;
run)
	backup=
	for argument in "$@"; do
		case "$argument" in
		*:/backup.db:ro)
			backup=${argument%:/backup.db:ro}
			;;
		esac
	done
	if [ -n "$backup" ] && [ -f "$backup" ]; then
		rm -f "${FAKE_CONTAINER_DATA:?}/cinecat.db" "${FAKE_CONTAINER_DATA:?}/cinecat.db-wal" "${FAKE_CONTAINER_DATA:?}/cinecat.db-shm"
		cp "$backup" "${FAKE_CONTAINER_DATA:?}/cinecat.db"
	fi
	;;
esac
exit 0
FAKE

cat >"$FAKE_BIN/sqlite3" <<'FAKE'
#!/usr/bin/env bash
set -u
printf 'sqlite3 %s\n' "$*" >>"${FAKE_CALLS:?}"
source_db=$1
destination=$(printf '%s' "$2" | sed -n "s/.*VACUUM INTO '\([^']*\)'.*/\1/p")
[ -n "$destination" ] || exit 1
mkdir -p "$(dirname "$destination")"
cp "$source_db" "$destination"
FAKE

cat >"$FAKE_BIN/df" <<'FAKE'
#!/usr/bin/env bash
printf 'Filesystem 1024-blocks Used Available Capacity Mounted on\n'
printf 'fake 9999999 1 %s 1%% /\n' "${FAKE_DF_AVAILABLE_KB:-999999999}"
FAKE

cat >"$FAKE_BIN/curl" <<'FAKE'
#!/usr/bin/env bash
set -u
printf 'curl %s\n' "$*" >>"${FAKE_CALLS:?}"
if [ "${FAKE_CURL_FAIL:-0}" = "1" ]; then
	exit 1
fi
url=${!#}
case "$url" in
*/api/v1/system/version)
	version=$(cat "${FAKE_VERSION_FILE:?}" 2>/dev/null || printf '0.1.0')
	printf '{"server":"%s","commit":"fake","built_at":"fake"}\n' "$version"
	;;
*)
	printf 'ok\n'
	;;
esac
FAKE

chmod +x "$FAKE_BIN/docker" "$FAKE_BIN/sqlite3" "$FAKE_BIN/df" "$FAKE_BIN/curl"

new_repo() {
	local case_dir=$1
	local origin="$case_dir/origin.git"
	local seed="$case_dir/seed"
	local repo="$case_dir/repo"

	"$GIT_BIN" init -q --bare "$origin"
	"$GIT_BIN" init -q -b main "$seed"
	"$GIT_BIN" -C "$seed" config user.name "Pruebas"
	"$GIT_BIN" -C "$seed" config user.email "pruebas@example.invalid"
	"$GIT_BIN" -C "$seed" config core.autocrlf false
	printf '0.1.0\n' >"$seed/VERSION"
	printf 'Prueba\n' >"$seed/README.md"
	"$GIT_BIN" -C "$seed" add VERSION README.md
	"$GIT_BIN" -C "$seed" commit -qm "v0.1.0"
	"$GIT_BIN" -C "$seed" tag v0.1.0
	printf '0.2.0\n' >"$seed/VERSION"
	"$GIT_BIN" -C "$seed" add VERSION
	"$GIT_BIN" -C "$seed" commit -qm "v0.2.0"
	"$GIT_BIN" -C "$seed" tag v0.2.0
	"$GIT_BIN" -C "$seed" remote add origin "$origin"
	"$GIT_BIN" -C "$seed" push -q origin main --tags
	"$GIT_BIN" -C "$origin" symbolic-ref HEAD refs/heads/main

	"$GIT_BIN" clone -q -c core.autocrlf=false -b main "$origin" "$repo"
	"$GIT_BIN" -C "$repo" checkout -q v0.1.0
	"$GIT_BIN" -C "$repo" config user.name "Pruebas"
	"$GIT_BIN" -C "$repo" config user.email "pruebas@example.invalid"
	"$GIT_BIN" -C "$repo" config core.autocrlf false
}

new_case() {
	local name=$1
	CURRENT_CASE=$(mktemp -d "$TEST_ROOT/caso.XXXXXX")
	REPO="$CURRENT_CASE/repo"
	RUN="$REPO/run"
	BACKUP="$CURRENT_CASE/backups"
	CACHE="$CURRENT_CASE/cache"
	DATA="$CURRENT_CASE/container-data"
	CALLS="$CURRENT_CASE/calls.log"
	VERSION_FILE="$CURRENT_CASE/server-version"
	mkdir -p "$BACKUP" "$CACHE" "$DATA" "$CURRENT_CASE/docker-root"
	printf 'OLD-DB\n' >"$DATA/cinecat.db"
	printf 'wal\n' >"$DATA/cinecat.db-wal"
	printf 'shm\n' >"$DATA/cinecat.db-shm"
	new_repo "$CURRENT_CASE"
	mkdir -p "$RUN"
	printf '0.1.0\n' >"$VERSION_FILE"
	export CINECAT_DIR="$REPO"
	export RUN_DIR="$RUN"
	export BACKUP_DIR="$BACKUP"
	export CACHE_DIR="$CACHE"
	export FAKE_CALLS="$CALLS"
	export FAKE_CONTAINER_DATA="$DATA"
	export FAKE_DOCKER_ROOT="$CURRENT_CASE/docker-root"
	export FAKE_VERSION_FILE="$VERSION_FILE"
	export FAKE_TARGET_VERSION="0.2.0"
	export GIT_BIN
	export DOCKER_BIN="$FAKE_BIN/docker"
	export SQLITE3_BIN="$FAKE_BIN/sqlite3"
	export CURL_BIN="$FAKE_BIN/curl"
	export DF_BIN="$FAKE_BIN/df"
	export CINECAT_HEALTH_TIMEOUT=2
	export CINECAT_HEALTH_INTERVAL=0.05
	unset FAKE_DF_AVAILABLE_KB FAKE_CURL_FAIL FAKE_DOCKER_INFO_DELAY FAKE_DOCKER_DELAY FAKE_DOCKER_FAIL_BUILD FAKE_DOCKER_KEEP_VERSION 2>/dev/null || true
	printf '\n== %s ==\n' "$name"
}

write_request() {
	local raw=$1
	printf '%s\n' "$raw" >"$RUN/update-request.json"
}

write_tag_request() {
	local tag=$1
	write_request "{\"tag\":\"$tag\",\"requested_by\":\"pruebas\",\"requested_at\":\"2026-10-27T00:00:00Z\"}"
}

run_updater() {
	env \
		PATH="$FAKE_BIN:$PATH" \
		CINECAT_DIR="$REPO" \
		RUN_DIR="$RUN" \
		BACKUP_DIR="$BACKUP" \
		CACHE_DIR="$CACHE" \
		DOCKER_BIN="$DOCKER_BIN" \
		SQLITE3_BIN="$SQLITE3_BIN" \
		CURL_BIN="$CURL_BIN" \
		DF_BIN="$DF_BIN" \
		GIT_BIN="$GIT_BIN" \
		CINECAT_HEALTH_TIMEOUT="$CINECAT_HEALTH_TIMEOUT" \
		CINECAT_HEALTH_INTERVAL="$CINECAT_HEALTH_INTERVAL" \
		FAKE_CALLS="$FAKE_CALLS" \
		FAKE_CONTAINER_DATA="$FAKE_CONTAINER_DATA" \
		FAKE_DOCKER_ROOT="$FAKE_DOCKER_ROOT" \
		FAKE_VERSION_FILE="$FAKE_VERSION_FILE" \
		FAKE_DF_AVAILABLE_KB="${FAKE_DF_AVAILABLE_KB:-999999999}" \
		FAKE_CURL_FAIL="${FAKE_CURL_FAIL:-0}" \
		FAKE_DOCKER_INFO_DELAY="${FAKE_DOCKER_INFO_DELAY:-0}" \
		FAKE_DOCKER_DELAY="${FAKE_DOCKER_DELAY:-0}" \
		FAKE_DOCKER_FAIL_BUILD="${FAKE_DOCKER_FAIL_BUILD:-0}" \
		FAKE_DOCKER_KEEP_VERSION="${FAKE_DOCKER_KEEP_VERSION:-0}" \
		bash "$UPDATER" >"$CURRENT_CASE/stdout.log" 2>&1
}

status_result() {
	sed -n 's/.*"result":"\([^"]*\)".*/\1/p' "$RUN/update-status.json"
}

status_message() {
	sed -n 's/.*"message":"\([^"]*\)".*/\1/p' "$RUN/update-status.json"
}

request_removed() {
	assert_no_file "$RUN/update-request.json" "la petición se borra al empezar"
}

# Etiqueta no válida.
new_case "etiqueta no válida"
write_tag_request "v1.2"
if run_updater; then bad "una etiqueta inválida debe fallar"; else ok "una etiqueta inválida falla"; fi
assert_eq "failed" "$(status_result)" "estado failed"
request_removed
assert_eq "0.1.0" "$(tr -d '[:space:]' <"$REPO/VERSION")" "VERSION no cambia"
assert_eq "" "$(find "$BACKUP" -name '*-pre-*.db' -print -quit)" "no se crea copia"

# Etiqueta no mayor.
new_case "etiqueta no mayor"
write_tag_request "v0.1.0"
if run_updater; then bad "una etiqueta igual debe fallar"; else ok "una etiqueta igual falla"; fi
assert_eq "failed" "$(status_result)" "estado failed"
assert_contains "no es mayor" "$RUN/update-status.json" "mensaje de versión no mayor"
request_removed

# Etiqueta inexistente.
new_case "etiqueta inexistente"
write_tag_request "v9.9.9"
if run_updater; then bad "una etiqueta inexistente debe fallar"; else ok "una etiqueta inexistente falla"; fi
assert_eq "rolled_back" "$(status_result)" "estado rolled_back"
assert_eq "0.1.0" "$(tr -d '[:space:]' <"$REPO/VERSION")" "VERSION sigue en la anterior"
assert_file "$BACKUP/updater.log" "se escribe el registro"

# Espacio insuficiente.
new_case "espacio insuficiente"
FAKE_DF_AVAILABLE_KB=1000
write_tag_request "v0.2.0"
if run_updater; then bad "sin espacio debe fallar"; else ok "sin espacio falla"; fi
assert_eq "failed" "$(status_result)" "estado failed"
assert_contains "Espacio insuficiente" "$RUN/update-status.json" "mensaje de espacio"
if grep -Fq 'docker cp' "$CALLS"; then bad "no debe copiarse la base sin espacio"; else ok "no se copia la base sin espacio"; fi

# Caso feliz.
new_case "actualización correcta"
write_tag_request "v0.2.0"
if run_updater; then ok "la actualización termina bien"; else bad "la actualización debía terminar bien"; fi
assert_eq "ok" "$(status_result)" "estado ok"
assert_eq "0.2.0" "$(tr -d '[:space:]' <"$REPO/VERSION")" "VERSION nueva"
request_removed
assert_eq "1" "$(find "$BACKUP" -name '*-pre-0.1.0.db' | wc -l | tr -d ' ')" "se conserva una copia previa"
assert_contains 'docker compose build' "$CALLS" "se compila"
assert_contains 'docker compose up' "$CALLS" "se levanta"
assert_contains '"phase":"Terminado"' "$RUN/update-status.json" "fase final"

# Fallo de salud y rollback.
new_case "fallo de salud y rollback"
FAKE_DOCKER_KEEP_VERSION=1
OLD_COMMIT=$("$GIT_BIN" -C "$REPO" rev-parse HEAD)
write_tag_request "v0.2.0"
if run_updater; then bad "un fallo de salud debe fallar"; else ok "un fallo de salud falla"; fi
assert_eq "rolled_back" "$(status_result)" "estado rolled_back"
assert_eq "0.1.0" "$(tr -d '[:space:]' <"$REPO/VERSION")" "se restaura VERSION"
assert_eq "$OLD_COMMIT" "$("$GIT_BIN" -C "$REPO" rev-parse HEAD)" "se restaura el commit"
assert_contains 'docker run --rm' "$CALLS" "se restaura la base con un contenedor temporal"
assert_eq "OLD-DB" "$(tr -d '[:space:]' <"$DATA/cinecat.db")" "la base vuelve a la copia"
request_removed

# Fallo antes de levantar: el servidor anterior sigue vivo y no se toca su base.
new_case "fallo de compilación sin reinicio"
FAKE_DOCKER_FAIL_BUILD=1
FAKE_DOCKER_KEEP_VERSION=1
OLD_COMMIT=$("$GIT_BIN" -C "$REPO" rev-parse HEAD)
printf 'DB-CON-DATOS-NUEVOS
' >"$DATA/cinecat.db"
write_tag_request "v0.2.0"
if run_updater; then bad "un fallo de compilación debe fallar"; else ok "un fallo de compilación falla"; fi
assert_eq "rolled_back" "$(status_result)" "estado rolled_back tras fallo de compilación"
assert_eq "$OLD_COMMIT" "$("$GIT_BIN" -C "$REPO" rev-parse HEAD)" "se restaura el commit tras fallo de compilación"
if grep -Fq 'docker run --rm' "$CALLS"; then bad "no debe restaurarse la base si no se reinició"; else ok "no se restaura la base si no se reinició"; fi
if grep -Fq 'compose stop' "$CALLS"; then bad "no debe pararse el servidor si no se reinició"; else ok "no se para el servidor si no se reinició"; fi
assert_eq "DB-CON-DATOS-NUEVOS" "$(tr -d '[:space:]' <"$DATA/cinecat.db")" "la base conserva lo guardado durante la compilación"
request_removed

# Contenido malicioso.
new_case "contenido malicioso"
PWNED="$CURRENT_CASE/pwned"
write_request "{\"tag\":\"v0.2.0\\\"; touch $PWNED; #\",\"requested_by\":\"x\",\"requested_at\":\"2026-10-27T00:00:00Z\"}"
if run_updater; then bad "la inyección debe rechazarse"; else ok "la inyección se rechaza"; fi
assert_eq "failed" "$(status_result)" "estado failed"
assert_no_file "$PWNED" "no se ejecuta el contenido de la petición"
assert_eq "0.1.0" "$(tr -d '[:space:]' <"$REPO/VERSION")" "VERSION intacta"

# Dos ejecuciones simultáneas.
new_case "ejecuciones simultáneas"
write_tag_request "v0.2.0"
export FAKE_DOCKER_INFO_DELAY=1
run_updater &
FIRST_PID=$!
sleep 0.2
if run_updater; then ok "la segunda ejecución sale sin hacer cambios"; else bad "la segunda ejecución no debía fallar"; fi
if wait "$FIRST_PID"; then ok "la primera ejecución termina"; else bad "la primera ejecución debía terminar"; fi
assert_eq "ok" "$(status_result)" "solo queda un estado final ok"
assert_contains "Ya hay otra actualización en curso" "$BACKUP/updater.log" "se detecta el candado"
assert_eq "1" "$(find "$BACKUP" -name '*-pre-0.1.0.db' | wc -l | tr -d ' ')" "solo se crea una copia"
unset FAKE_DOCKER_INFO_DELAY

printf '\n%d pruebas correctas, %d fallos.\n' "$PASS" "$FAIL"
if [ "$FAIL" -ne 0 ]; then
	exit 1
fi

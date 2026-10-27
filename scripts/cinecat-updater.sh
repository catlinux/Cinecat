#!/usr/bin/env bash
# Recoge la petición escrita por el servidor, actualiza Cinecat desde una
# etiqueta de origin y vuelve atrás si algo falla. Lo ejecuta systemd como
# usuario, nunca desde el contenedor.
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
CINECAT_DIR=${CINECAT_DIR:-$(cd "$SCRIPT_DIR/.." && pwd -P)}
RUN_DIR=${RUN_DIR:-$CINECAT_DIR/run}
BACKUP_DIR=${BACKUP_DIR:-$HOME/cinecat-backups}
CACHE_DIR=${CACHE_DIR:-$HOME/.cache/cinecat-tmp}
REQUEST_FILE=${REQUEST_FILE:-$RUN_DIR/update-request.json}
STATUS_FILE=${STATUS_FILE:-$RUN_DIR/update-status.json}
LOCK_FILE=${LOCK_FILE:-$RUN_DIR/update.lock}
LOG_FILE=${LOG_FILE:-$BACKUP_DIR/updater.log}

DOCKER_BIN=${DOCKER_BIN:-docker}
GIT_BIN=${GIT_BIN:-git}
SQLITE3_BIN=${SQLITE3_BIN:-sqlite3}
CURL_BIN=${CURL_BIN:-curl}
DF_BIN=${DF_BIN:-df}
FLOCK_BIN=${FLOCK_BIN:-flock}
PYTHON_BIN=${PYTHON_BIN:-python3}
SERVER_CONTAINER=${SERVER_CONTAINER:-cinecat-servidor-1}
WEB_CONTAINER=${WEB_CONTAINER:-cinecat-web-1}
DATA_VOLUME=${DATA_VOLUME:-cinecat_cinecat_data}
SERVER_IMAGE_FALLBACK=${SERVER_IMAGE_FALLBACK:-cinecat-servidor}
HEALTH_TIMEOUT=${CINECAT_HEALTH_TIMEOUT:-90}
HEALTH_INTERVAL=${CINECAT_HEALTH_INTERVAL:-2}
REQUIRED_FREE_KB=${CINECAT_REQUIRED_FREE_KB:-3145728}

STARTED_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
TAG=
TARGET_VERSION=
CURRENT_VERSION=
OLD_COMMIT=
BACKUP_PATH=
DB_TMP=
FAILURE_REASON=
ROLLBACK_IMAGE=
UPDATE_APPLIED=0
NEW_STARTED=0
STATUS_FINAL=0
IN_ROLLBACK=0
LOCK_HELD=0
API_PORT=8080
WEB_PORT=8081

mkdir -p "$RUN_DIR" "$BACKUP_DIR" "$CACHE_DIR"

log() {
	local message=$1
	printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$message" | tee -a "$LOG_FILE" >&2
}

rotate_log() {
	if [ -f "$LOG_FILE" ]; then
		local tmp="$LOG_FILE.tmp.$$"
		tail -n 2000 "$LOG_FILE" >"$tmp" || true
		mv -f "$tmp" "$LOG_FILE" || true
	fi
}

json_escape() {
	local value=${1-}
	value=${value//\\/\\\\}
	value=${value//\"/\\\"}
	value=${value//$'\n'/\\n}
	value=${value//$'\r'/}
	value=${value//$'\t'/\\t}
	printf '%s' "$value"
}

write_status() {
	local phase=$1
	local message=$2
	local result=${3:-}
	local finished_at=
	if [ -n "$result" ]; then
		finished_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	fi
	local tmp="$STATUS_FILE.tmp.$$"
	printf '{"phase":"%s","tag":"%s","message":"%s","started_at":"%s","finished_at":"%s","result":"%s"}\n' \
		"$(json_escape "$phase")" \
		"$(json_escape "$TAG")" \
		"$(json_escape "$message")" \
		"$STARTED_AT" \
		"$finished_at" \
		"$(json_escape "$result")" >"$tmp"
	mv -f "$tmp" "$STATUS_FILE"
}

finish_status() {
	local phase=$1
	local message=$2
	local result=$3
	write_status "$phase" "$message" "$result"
	STATUS_FINAL=1
	log "$message"
	rotate_log
}

fail() {
	finish_status "Terminado" "$1" "failed"
	exit 1
}

cleanup() {
	if [ -n "$DB_TMP" ] && [ -d "$DB_TMP" ]; then
		rm -rf "$DB_TMP"
	fi
	if [ "$LOCK_HELD" -eq 1 ]; then
		rmdir "$LOCK_FILE.d" 2>/dev/null || true
	fi
}

rollback_failure_message() {
	local image=${ROLLBACK_IMAGE:-<imagen>}
	local restore_command='rm -f /data/cinecat.db /data/cinecat.db-wal /data/cinecat.db-shm && cp /backup.db /data/cinecat.db'
	printf "No se pudo revertir automáticamente. Copia: %s. Commit anterior: %s. Recuperación manual: cd %s && git reset --hard %s && docker compose build && docker compose stop servidor && docker run --rm -v %s:/data -v %s:/backup.db:ro --entrypoint sh %s -c '%s' && docker compose up -d" \
		"${BACKUP_PATH:-sin copia}" "${OLD_COMMIT:-desconocido}" "$CINECAT_DIR" \
		"${OLD_COMMIT:-desconocido}" "$DATA_VOLUME" "${BACKUP_PATH:-sin copia}" \
		"$image" "$restore_command"
}

on_exit() {
	local rc=$?
	cleanup
	if [ "$STATUS_FINAL" -eq 1 ]; then
		return
	fi
	if [ "$UPDATE_APPLIED" -eq 1 ] && [ "$IN_ROLLBACK" -eq 0 ]; then
		log "Fallo inesperado; se intenta revertir."
		if do_rollback; then
			finish_status "Revertido" "La actualización falló y se revirtió a la versión anterior." "rolled_back"
		else
			finish_status "Terminado" "$(rollback_failure_message)" "failed"
		fi
	else
		finish_status "Terminado" "Error inesperado durante la actualización; revisa el registro." "failed"
	fi
	exit "$rc"
}
trap on_exit EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP

version_gt() {
	local IFS=.
	local a b c x y z
	read -r a b c <<<"${1#v}"
	read -r x y z <<<"${2#v}"
	a=${a:-0}
	b=${b:-0}
	c=${c:-0}
	x=${x:-0}
	y=${y:-0}
	z=${z:-0}
	((10#$a > 10#$x)) && return 0
	((10#$a < 10#$x)) && return 1
	((10#$b > 10#$y)) && return 0
	((10#$b < 10#$y)) && return 1
	((10#$c > 10#$z))
}

env_value() {
	local key=$1
	local value=
	if [ -f "$CINECAT_DIR/.env" ]; then
		value=$(sed -n "s/^[[:space:]]*${key}[[:space:]]*=[[:space:]]*//p" "$CINECAT_DIR/.env" | tail -n 1)
	fi
	value=${value%$'\r'}
	value=${value%\"}
	value=${value#\"}
	printf '%s' "$value"
}

extract_tag() {
	if command -v "$PYTHON_BIN" >/dev/null 2>&1; then
		printf '%s' "$REQUEST_CONTENT" | "$PYTHON_BIN" -c '
import json
import sys

try:
    data = json.load(sys.stdin)
except Exception:
    sys.exit(1)
if not isinstance(data, dict):
    sys.exit(1)
tag = data.get("tag")
if not isinstance(tag, str):
    sys.exit(1)
print(tag)
' 2>/dev/null || true
	else
		# Sin python3, solo se acepta la estructura exacta que escribe el
		# servidor; el campo tag no admite comillas ni barras escapadas.
		local compact=${REQUEST_CONTENT//$'\n'/}
		compact=${compact//$'\r'/}
		if [[ "$compact" =~ ^[[:space:]]*\{[[:space:]]*\"tag\"[[:space:]]*:[[:space:]]*\"([^\"\\]*)\"[[:space:]]*,[[:space:]]*\"requested_by\"[[:space:]]*:[[:space:]]*\"([^\"\\]|\\.)*\"[[:space:]]*,[[:space:]]*\"requested_at\"[[:space:]]*:[[:space:]]*\"([^\"\\]|\\.)*\"[[:space:]]*\}[[:space:]]*$ ]]; then
			printf '%s' "${BASH_REMATCH[1]}"
		fi
	fi
}

check_space() {
	local docker_root
	docker_root=$("$DOCKER_BIN" info -f '{{.DockerRootDir}}' 2>/dev/null) ||
		fail "No se pudo consultar el directorio de Docker: el servicio no puede usar Docker. Si el usuario entró en el grupo docker después de arrancar el servidor, reinicia el gestor de usuario con «sudo systemctl restart user@$(id -u).service» o reinicia el servidor."
	local available
	available=$("$DF_BIN" -Pk "$docker_root" 2>/dev/null | awk 'NR==2 {print $4}')
	if [ -z "$available" ] || [ "$available" -lt "$REQUIRED_FREE_KB" ]; then
		fail "Espacio insuficiente en el sistema de archivos de Docker."
	fi
	available=$("$DF_BIN" -Pk "$CACHE_DIR" 2>/dev/null | awk 'NR==2 {print $4}')
	if [ -z "$available" ] || [ "$available" -lt "$REQUIRED_FREE_KB" ]; then
		fail "Espacio insuficiente en el directorio temporal de compilación."
	fi
}

backup_database() {
	local suffix
	DB_TMP=$(mktemp -d "$CACHE_DIR/db.XXXXXX") || return 1
	if ! "$DOCKER_BIN" cp "$SERVER_CONTAINER:/data/cinecat.db" "$DB_TMP/cinecat.db"; then
		return 1
	fi
	for suffix in -wal -shm; do
		"$DOCKER_BIN" cp "$SERVER_CONTAINER:/data/cinecat.db$suffix" "$DB_TMP/cinecat.db$suffix" >/dev/null 2>&1 || true
	done
	BACKUP_PATH="$BACKUP_DIR/cinecat-$(date +%Y%m%d-%H%M%S)-pre-${CURRENT_VERSION}.db"
	if ! "$SQLITE3_BIN" "$DB_TMP/cinecat.db" "VACUUM INTO '$BACKUP_PATH'"; then
		return 1
	fi
	ls -1t "$BACKUP_DIR"/cinecat-*-pre-*.db 2>/dev/null | tail -n +11 |
		while IFS= read -r old; do
			rm -f "$old"
		done
	return 0
}

wait_for_health() {
	local version=$1
	local deadline=$((SECONDS + HEALTH_TIMEOUT))
	local version_json=
	while [ "$SECONDS" -lt "$deadline" ]; do
		if "$CURL_BIN" -fsS "http://127.0.0.1:${API_PORT}/healthz" >/dev/null 2>&1; then
			version_json=$("$CURL_BIN" -fsS "http://127.0.0.1:${API_PORT}/api/v1/system/version" 2>/dev/null || true)
			if printf '%s' "$version_json" |
				grep -Eq "\"server\"[[:space:]]*:[[:space:]]*\"${version}\"" &&
				"$CURL_BIN" -fsS "http://127.0.0.1:${WEB_PORT}/" >/dev/null 2>&1; then
				return 0
			fi
		fi
		sleep "$HEALTH_INTERVAL"
	done
	return 1
}

apply_update() {
	write_status "Descargando la versión" "Preparando la etiqueta $TAG." ""
	OLD_COMMIT=$("$GIT_BIN" -C "$CINECAT_DIR" rev-parse HEAD) || {
		FAILURE_REASON="No se pudo leer el commit actual."
		return 1
	}
	if ! "$GIT_BIN" -C "$CINECAT_DIR" fetch --tags origin; then
		FAILURE_REASON="No se pudo descargar la etiqueta desde origin."
		return 1
	fi
	if ! "$GIT_BIN" -C "$CINECAT_DIR" ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null 2>&1; then
		FAILURE_REASON="La etiqueta $TAG no existe en origin."
		return 1
	fi
	local new_commit
	new_commit=$("$GIT_BIN" -C "$CINECAT_DIR" rev-parse --verify "$TAG^{commit}") || {
		FAILURE_REASON="La etiqueta $TAG no apunta a un commit válido."
		return 1
	}
	if ! "$GIT_BIN" -C "$CINECAT_DIR" merge-base --is-ancestor "$OLD_COMMIT" "$new_commit"; then
		FAILURE_REASON="La etiqueta $TAG no desciende del commit actual."
		return 1
	fi
	if ! "$GIT_BIN" -C "$CINECAT_DIR" merge --ff-only "$TAG"; then
		FAILURE_REASON="No se pudo aplicar la etiqueta $TAG."
		return 1
	fi
	UPDATE_APPLIED=1

	write_status "Compilando" "Compilando y levantando $TAG." ""
	local commit built_at
	commit=$("$GIT_BIN" -C "$CINECAT_DIR" rev-parse --short HEAD) || {
		FAILURE_REASON="No se pudo leer el commit nuevo."
		return 1
	}
	built_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	export COMMIT="$commit" BUILT_AT="$built_at" TMPDIR="$CACHE_DIR"
	if ! (cd "$CINECAT_DIR" && "$DOCKER_BIN" compose build); then
		FAILURE_REASON="Falló la compilación."
		return 1
	fi
	write_status "Reiniciando" "Levantando los contenedores de $TAG." ""
	NEW_STARTED=1
	if ! (cd "$CINECAT_DIR" && "$DOCKER_BIN" compose up -d); then
		FAILURE_REASON="No se pudieron levantar los contenedores."
		return 1
	fi
	write_status "Comprobando que arranca" "Esperando a que responda la versión $TARGET_VERSION." ""
	if ! wait_for_health "$TARGET_VERSION"; then
		FAILURE_REASON="La versión nueva no respondió a tiempo."
		return 1
	fi
	return 0
}

do_rollback() {
	IN_ROLLBACK=1
	write_status "Reiniciando" "Revirtiendo al commit $OLD_COMMIT." ""
	if [ -z "$OLD_COMMIT" ]; then
		return 1
	fi
	if ! "$GIT_BIN" -C "$CINECAT_DIR" reset --hard "$OLD_COMMIT"; then
		return 1
	fi
	# Si aún no se habían recreado los contenedores, el servidor anterior sigue
	# en marcha con su base de datos al día: restaurar la copia o reiniciarlo
	# solo haría perder lo guardado durante la compilación.
	if [ "$NEW_STARTED" -eq 0 ]; then
		wait_for_health "$CURRENT_VERSION"
		return
	fi
	local commit built_at
	commit=$("$GIT_BIN" -C "$CINECAT_DIR" rev-parse --short HEAD) || return 1
	built_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	export COMMIT="$commit" BUILT_AT="$built_at" TMPDIR="$CACHE_DIR"
	if ! (cd "$CINECAT_DIR" && "$DOCKER_BIN" compose stop servidor); then
		return 1
	fi
	local image
	image=$("$DOCKER_BIN" inspect -f '{{.Config.Image}}' "$SERVER_CONTAINER" 2>/dev/null || true)
	if [ -z "$image" ]; then
		image=$SERVER_IMAGE_FALLBACK
	fi
	ROLLBACK_IMAGE=$image
	if [ -z "$BACKUP_PATH" ] || [ ! -f "$BACKUP_PATH" ]; then
		return 1
	fi
	if ! "$DOCKER_BIN" run --rm \
		-v "$DATA_VOLUME:/data" \
		-v "$BACKUP_PATH:/backup.db:ro" \
		--entrypoint sh "$image" -c \
		'rm -f /data/cinecat.db /data/cinecat.db-wal /data/cinecat.db-shm && cp /backup.db /data/cinecat.db'; then
		return 1
	fi
	if ! (cd "$CINECAT_DIR" && "$DOCKER_BIN" compose build); then
		return 1
	fi
	if ! (cd "$CINECAT_DIR" && "$DOCKER_BIN" compose up -d); then
		return 1
	fi
	wait_for_health "$CURRENT_VERSION"
}

if command -v "$FLOCK_BIN" >/dev/null 2>&1; then
	exec 9>"$LOCK_FILE"
	if ! "$FLOCK_BIN" -n 9; then
		log "Ya hay otra actualización en curso; no se hace nada."
		STATUS_FINAL=1
		exit 0
	fi
else
	# Respaldo para entornos de prueba sin flock; en el host se usa flock.
	if ! mkdir "$LOCK_FILE.d" 2>/dev/null; then
		log "Ya hay otra actualización en curso; no se hace nada."
		STATUS_FINAL=1
		exit 0
	fi
	LOCK_HELD=1
fi

if [ ! -f "$REQUEST_FILE" ]; then
	log "No hay petición de actualización."
	STATUS_FINAL=1
	exit 0
fi

REQUEST_CONTENT=$(cat "$REQUEST_FILE")
rm -f "$REQUEST_FILE"

TAG=$(extract_tag)
write_status "Validando" "Comprobando la petición." ""

if ! [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	fail "La etiqueta de la petición no es válida."
fi
TARGET_VERSION=${TAG#v}
if [ ! -f "$CINECAT_DIR/VERSION" ]; then
	fail "No se encontró el archivo VERSION del repositorio."
fi
CURRENT_VERSION=$(tr -d '[:space:]' <"$CINECAT_DIR/VERSION")
if ! version_gt "$TARGET_VERSION" "$CURRENT_VERSION"; then
	fail "La versión $TARGET_VERSION no es mayor que la actual ($CURRENT_VERSION)."
fi

API_PORT=$(env_value CINECAT_API_PORT)
API_PORT=${API_PORT:-8080}
WEB_PORT=$(env_value CINECAT_WEB_PORT)
WEB_PORT=${WEB_PORT:-8081}

write_status "Comprobando espacio" "Comprobando espacio libre en Docker y en la caché." ""
check_space

write_status "Copia de la base de datos" "Guardando una copia de la base de datos." ""
if ! backup_database; then
	fail "No se pudo crear la copia de la base de datos."
fi

write_status "Descargando la versión" "Actualizando el repositorio a $TAG." ""
if apply_update; then
	finish_status "Terminado" "Actualizado correctamente a $TAG." "ok"
	exit 0
fi

log "La actualización a $TAG falló; se intenta revertir."
reason=${FAILURE_REASON:-motivo desconocido}
if do_rollback; then
	finish_status "Revertido" "La actualización a $TAG falló: $reason Se revirtió a $CURRENT_VERSION." "rolled_back"
else
	finish_status "Terminado" "$reason $(rollback_failure_message)" "failed"
fi
exit 1

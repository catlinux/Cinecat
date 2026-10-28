#!/usr/bin/env bash
# Mantiene al día un registro DynHost de OVH cuando cambia la IP pública.
#
# Lo ejecuta el temporizador de usuario cinecat-dynhost cada pocos minutos. Solo
# llama a OVH si la IP ha cambiado desde la última vez, o si hace más de 24 h
# que no se confirma (por si alguien tocó el registro a mano en el panel).
#
# La configuración está en ~/.config/cinecat/dynhost.env (permisos 0600):
#   DYNHOST_HOSTNAME  nombre completo del registro, p. ej. media.techmarc.es
#   DYNHOST_USER      identificador DynHost creado en el panel de OVH
#   DYNHOST_PASSWORD  su contraseña (no la de la cuenta de OVH)
#
# Variables opcionales de entorno, útiles para probar:
#   DYNHOST_CONFIG       otra ruta de configuración
#   DYNHOST_URL          otro endpoint (por defecto, el oficial de OVH)
#   DYNHOST_IP_SERVICES  servicios que devuelven la IP pública, separados por espacios
#   DYNHOST_DRY_RUN=1    no llama a OVH ni guarda estado; solo muestra lo que haría
set -euo pipefail

CONFIG=${DYNHOST_CONFIG:-${XDG_CONFIG_HOME:-$HOME/.config}/cinecat/dynhost.env}
STATE_DIR=${XDG_STATE_HOME:-$HOME/.local/state}/cinecat
STATE_FILE=$STATE_DIR/dynhost.ip
OVH_URL=${DYNHOST_URL:-https://dns.eu.ovhapis.com/nic/update}
IP_SERVICES=${DYNHOST_IP_SERVICES:-"https://api.ipify.org https://ifconfig.me/ip https://icanhazip.com"}
DRY_RUN=${DYNHOST_DRY_RUN:-0}
REFRESH_MINUTES=1440

log() { printf '%s\n' "$*"; }
fail() { printf 'Error: %s\n' "$*" >&2; exit 1; }

command -v curl >/dev/null 2>&1 || fail "falta curl"
[ -f "$CONFIG" ] || fail "no existe $CONFIG (copia deploy/dynhost/dynhost.env.example)"

# La contraseña no debe poder leerla otro usuario del equipo.
mode=$(stat -c '%a' "$CONFIG")
if (( (8#$mode & 8#077) != 0 )); then
	fail "$CONFIG tiene permisos $mode; ejecuta: chmod 600 $CONFIG"
fi

DYNHOST_HOSTNAME= DYNHOST_USER= DYNHOST_PASSWORD=
# El archivo se lee como datos (CLAVE=valor), no como código: una contraseña
# con &, $ o comillas no debe interpretarse. Se admiten comillas envolventes.
while IFS= read -r line || [ -n "$line" ]; do
	line=${line%$'\r'}
	case $line in '' | \#*) continue ;; esac
	key=${line%%=*}
	value=${line#*=}
	if [ "${#value}" -ge 2 ] && { [[ $value == \"*\" ]] || [[ $value == \'*\' ]]; }; then
		value=${value:1:${#value}-2}
	fi
	case $key in
		DYNHOST_HOSTNAME) DYNHOST_HOSTNAME=$value ;;
		DYNHOST_USER) DYNHOST_USER=$value ;;
		DYNHOST_PASSWORD) DYNHOST_PASSWORD=$value ;;
	esac
done < "$CONFIG"
[ -n "$DYNHOST_HOSTNAME" ] || fail "falta DYNHOST_HOSTNAME en $CONFIG"
[ -n "$DYNHOST_USER" ] || fail "falta DYNHOST_USER en $CONFIG"
[ -n "$DYNHOST_PASSWORD" ] || fail "falta DYNHOST_PASSWORD en $CONFIG"
[[ $DYNHOST_HOSTNAME =~ ^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$ ]] || fail "DYNHOST_HOSTNAME no es un nombre válido"

valid_ipv4() {
	local ip=$1 octet
	[[ $ip =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] || return 1
	local IFS=.
	for octet in $ip; do
		(( 10#$octet <= 255 )) || return 1
	done
}

public_ip() {
	local service ip
	for service in $IP_SERVICES; do
		ip=$(curl -fsS -4 -m 10 "$service" 2>/dev/null | tr -d '[:space:]') || continue
		if valid_ipv4 "$ip"; then
			printf '%s' "$ip"
			return 0
		fi
	done
	return 1
}

ip=$(public_ip) || fail "no se pudo averiguar la IP pública (¿sin conexión?)"

last=
[ -f "$STATE_FILE" ] && last=$(tr -d '[:space:]' < "$STATE_FILE")
stale=0
if [ -f "$STATE_FILE" ] && [ -n "$(find "$STATE_FILE" -mmin +"$REFRESH_MINUTES" 2>/dev/null)" ]; then
	stale=1
fi

if [ "$ip" = "$last" ] && [ "$stale" = 0 ]; then
	log "La IP sigue siendo $ip; no hay nada que actualizar."
	exit 0
fi

if [ "$DRY_RUN" = 1 ]; then
	log "Simulación: se actualizaría $DYNHOST_HOSTNAME a $ip (última conocida: ${last:-ninguna})."
	exit 0
fi

# El usuario y la contraseña viajan por la entrada estándar de curl para que no
# aparezcan en la lista de procesos.
escape() { printf '%s' "$1" | sed 's/[\\"]/\\&/g'; }
response=$(printf 'user = "%s:%s"\n' "$(escape "$DYNHOST_USER")" "$(escape "$DYNHOST_PASSWORD")" |
	curl -sS -m 30 -K - -G \
		--data-urlencode "system=dyndns" \
		--data-urlencode "hostname=$DYNHOST_HOSTNAME" \
		--data-urlencode "myip=$ip" \
		-w '\n%{http_code}' "$OVH_URL") || fail "no se pudo contactar con OVH"

status=${response##*$'\n'}
body=${response%$'\n'*}
if [ "$status" = 200 ] && [[ $body =~ ^(good|nochg)([[:space:]]|$) ]]; then
	mkdir -p "$STATE_DIR"
	printf '%s\n' "$ip" > "$STATE_FILE"
	log "OVH confirmó $DYNHOST_HOSTNAME → $ip (${body%%[[:space:]]*})."
	exit 0
fi

hint=
if [ "$status" = 401 ] || [[ $body == badauth* ]]; then
	hint=" Revisa DYNHOST_USER y DYNHOST_PASSWORD: deben ser los del identificador DynHost del panel de OVH."
fi
fail "OVH respondió HTTP $status: ${body:-sin cuerpo}.$hint"

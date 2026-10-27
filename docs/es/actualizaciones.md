# Actualizaciones automáticas desde el host

Cinecat no se actualiza desde dentro del contenedor. El servidor deja una
petición en `run/update-request.json`; un servicio de systemd del usuario la
recoge, hace una copia de la base de datos, actualiza el repositorio, reconstruye
los contenedores y comprueba que la versión nueva responde. Si algo falla,
vuelve al commit y a la base de datos anteriores.

## Requisitos

- Docker con Compose y acceso del usuario al grupo `docker`.
- `git`, `sqlite3`, `curl`, `flock`, `python3` y `bash`.
- El repositorio desplegado en `/opt/cinecat` (o la ruta que se use).
- Linger activo para el usuario, para que el servicio funcione sin sesión
  abierta.

Todo se ejecuta como el usuario propietario del repositorio, nunca como root.
El directorio `run/` está ignorado por Git y se comparte con el contenedor
`servidor` mediante el montaje `./run:/run/cinecat-update`.

## Instalación

```bash
cd /opt/cinecat
mkdir -p run
docker compose up -d
bash scripts/instalar-actualizador.sh
```

El instalador primero **verifica los requisitos** (programas, acceso a Docker y
que `git` llegue a `origin` sin pedir contraseña ni frase de paso, porque el
servicio no tiene terminal). Después crea las unidades en
`~/.config/systemd/user/`, sustituye la ruta real del repositorio, activa
`cinecat-update.path` y prepara `run/`, `~/cinecat-backups` y
`~/.cache/cinecat-tmp`. Si no puede activar linger, muestra el comando exacto que
hay que ejecutar con `sudo`; no lo ejecuta. Si algo falla termina con
«Instalación INCOMPLETA» y explica qué corregir; al acabar bien, el estado debe
ser `path=active, service=inactive` (el servicio solo arranca cuando hay una
petición).

Si entras al servidor con `su` o con una shell sin sesión de login, el instalador
define por su cuenta `XDG_RUNTIME_DIR` y `DBUS_SESSION_BUS_ADDRESS`. Para usar
`systemctl --user` a mano en esa situación:

```bash
export XDG_RUNTIME_DIR=/run/user/$(id -u)
export DBUS_SESSION_BUS_ADDRESS=unix:path=$XDG_RUNTIME_DIR/bus
```

### Servidor nuevo o cambio de dominio

El actualizador no depende del dominio: habla con los contenedores por
`127.0.0.1` (puertos `CINECAT_API_PORT` y `CINECAT_WEB_PORT` de `.env`) y con
GitHub por `origin`. En un servidor nuevo, o tras cambiar de dominio:

1. Clona el repositorio en `/opt/cinecat` con el usuario que lo va a gestionar y
   crea `.env` (ver `docs/es/`).
2. Deja el proxy inverso del dominio nuevo apuntando a `127.0.0.1:8080` (`/api/`
   y `/healthz`) y a `127.0.0.1:8081` (resto); ejemplo en `deploy/`.
3. `mkdir -p run` y `docker compose up -d` **antes** de instalar, para que `run/`
   pertenezca al usuario y no a root (si Docker lo crea, el actualizador no podrá
   borrar las peticiones).
4. `bash scripts/instalar-actualizador.sh` y corrige lo que avise.
5. Prueba el ciclo completo con una versión pequeña antes de fiarte de él: desde
   Administración → Sistema, «Buscar actualizaciones» y «Actualizar ahora».
6. Si el servidor está en otra distribución o no tiene `flock` o `python3`,
   instálalos (el instalador lo detecta).

Para retirarlo:

```bash
bash scripts/instalar-actualizador.sh --desinstalar
```

Las copias y el registro se conservan.

## Flujo del actualizador

1. Lee la petición y la borra inmediatamente; valida que la etiqueta tenga el
   formato `vX.Y.Z` y sea mayor que `VERSION`.
2. Comprueba que quedan al menos 3 GB libres en el sistema de Docker y en
   `~/.cache/cinecat-tmp`.
3. Copia `cinecat.db`, `cinecat.db-wal` y `cinecat.db-shm` desde el contenedor y
   genera una copia consistente con `sqlite3 ... "VACUUM INTO ..."` en
   `~/cinecat-backups/cinecat-<fecha>-pre-<versión>.db`. Conserva las 10 últimas.
4. Hace `git fetch --tags`, comprueba que la etiqueta existe en `origin` y que
   desciende del commit actual, y aplica un `merge --ff-only`.
5. Compila y levanta con `COMMIT`, `BUILT_AT` y `TMPDIR` apuntando a
   `~/.cache/cinecat-tmp`.
6. Espera hasta 90 segundos a `/healthz`, a que `/api/v1/system/version`
   devuelva la versión nueva y a que responda la web.
7. Si falla cualquiera de los pasos anteriores desde la descarga: vuelve al
   commit guardado y deja el resultado en `update-status.json`. Si todavía no se
   habían recreado los contenedores (falló la descarga o la compilación), el
   servidor anterior sigue en marcha y no se toca su base de datos. Si ya se
   habían recreado, restaura también la copia de la base de datos y reconstruye la
   versión anterior.

El estado se escribe de forma atómica en `run/update-status.json`. Las fases son
«Validando», «Comprobando espacio», «Copia de la base de datos», «Descargando la
versión», «Compilando», «Reiniciando», «Comprobando que arranca», «Terminado» y
«Revertido». El resultado final es `ok`, `failed` o `rolled_back`.

## Problemas conocidos

- **«No se pudo consultar el directorio de Docker»**: los servicios de usuario
  heredan los grupos del gestor `systemd --user`, que con linger puede llevar
  meses vivo. Si el usuario entró en el grupo `docker` después de que arrancase,
  el servicio no puede usar Docker aunque tu sesión sí. Solución, una vez:
  `sudo systemctl restart user@$(id -u).service` (o reiniciar el servidor). El
  instalador lo detecta y lo avisa.
- **`systemctl --user` dice que no encuentra el bus**: ver arriba, variables
  `XDG_RUNTIME_DIR` y `DBUS_SESSION_BUS_ADDRESS`.

## Comprobar el estado

```bash
systemctl --user status cinecat-update.path
systemctl --user status cinecat-update.service
journalctl --user -u cinecat-update -f
cat /opt/cinecat/run/update-status.json
tail -n 50 ~/cinecat-backups/updater.log
```

El registro también se envía a journald y se recorta a las 2000 últimas líneas
en `~/cinecat-backups/updater.log`. Las copias anteriores están en
`~/cinecat-backups/`.

## Revertir a mano

Si el estado queda en `failed`, el mensaje incluye la copia y el commit que hay
que recuperar. El procedimiento es:

```bash
cd /opt/cinecat
git reset --hard <commit-anterior>
export COMMIT=$(git rev-parse --short HEAD)
export BUILT_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
export TMPDIR=$HOME/.cache/cinecat-tmp
docker compose build
docker compose stop servidor
IMAGEN=$(docker inspect -f '{{.Config.Image}}' cinecat-servidor-1)
docker run --rm \
  -v cinecat_cinecat_data:/data \
  -v "$HOME/cinecat-backups/<copia>.db:/backup.db:ro" \
  --entrypoint sh "$IMAGEN" -c \
  'rm -f /data/cinecat.db /data/cinecat.db-wal /data/cinecat.db-shm && cp /backup.db /data/cinecat.db'
docker compose up -d
```

No copies solo el archivo `.db`: hay que retirar también `-wal` y `-shm` y
restaurar la copia completa para no reutilizar un WAL antiguo. Después comprueba
`/healthz` y `/api/v1/system/version`.

## Seguridad

El actualizador solo acepta etiquetas con el formato estricto `vX.Y.Z`, no
interpreta el resto de la petición y comprueba que la etiqueta exista en
`origin`. Nunca ejecuta contenido escrito por el servidor. Las peticiones y el
estado no contienen secretos.

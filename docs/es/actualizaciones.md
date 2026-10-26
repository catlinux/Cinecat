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

El instalador crea las unidades en `~/.config/systemd/user/`, sustituye la ruta
real del repositorio, activa `cinecat-update.path` y prepara `run/`,
`~/cinecat-backups` y `~/.cache/cinecat-tmp`. Si no puede activar linger,
muestra el comando exacto que hay que ejecutar con `sudo`; no lo ejecuta.

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
   commit guardado, restaura la copia de la base de datos, reconstruye la versión
   anterior y deja el resultado en `update-status.json`.

El estado se escribe de forma atómica en `run/update-status.json`. Las fases son
«Validando», «Comprobando espacio», «Copia de la base de datos», «Descargando la
versión», «Compilando», «Reiniciando», «Comprobando que arranca», «Terminado» y
«Revertido». El resultado final es `ok`, `failed` o `rolled_back`.

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

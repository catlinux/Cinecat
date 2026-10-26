# Automatic updates from the host

Cinecat does not update itself from inside the container. The server writes a
request to `run/update-request.json`; a per-user systemd service picks it up,
backs up the database, updates the repository, rebuilds the containers and
checks that the new version responds. If anything fails, it restores the
previous commit and database copy.

## Requirements

- Docker with Compose and the user in the `docker` group.
- `git`, `sqlite3`, `curl`, `flock`, `python3` and `bash`.
- The repository deployed at `/opt/cinecat` (or the path actually used).
- Linger enabled for the user, so the service works without an open session.

Everything runs as the repository owner, never as root. `run/` is ignored by Git
and shared with the `servidor` container through `./run:/run/cinecat-update`.

## Installation

```bash
cd /opt/cinecat
mkdir -p run
docker compose up -d
bash scripts/instalar-actualizador.sh
```

The installer creates the units in `~/.config/systemd/user/`, replaces the real
repository path, enables `cinecat-update.path` and prepares `run/`,
`~/cinecat-backups` and `~/.cache/cinecat-tmp`. If it cannot enable linger, it
prints the exact command to run with `sudo`; it does not run it.

To remove it:

```bash
bash scripts/instalar-actualizador.sh --desinstalar
```

Backups and the log are kept.

## Update flow

1. Reads and immediately deletes the request; validates the `vX.Y.Z` tag and
   that it is newer than `VERSION`.
2. Checks for at least 3 GB free in Docker's filesystem and in
   `~/.cache/cinecat-tmp`.
3. Copies `cinecat.db`, `cinecat.db-wal` and `cinecat.db-shm` from the container
   and creates a consistent copy with `sqlite3 ... "VACUUM INTO ..."` at
   `~/cinecat-backups/cinecat-<date>-pre-<version>.db`. Keeps the last 10.
4. Runs `git fetch --tags`, checks that the tag exists in `origin` and descends
   from the current commit, then applies a `merge --ff-only`.
5. Builds and starts with `COMMIT`, `BUILT_AT` and `TMPDIR` pointing to
   `~/.cache/cinecat-tmp`.
6. Waits up to 90 seconds for `/healthz`, for `/api/v1/system/version` to return
   the new version and for the web container to respond.
7. If any step after the download fails: restores the saved commit and database
   copy, rebuilds the previous version and records the result in
   `update-status.json`.

Status is written atomically to `run/update-status.json`. Phases are
“Validando”, “Comprobando espacio”, “Copia de la base de datos”, “Descargando la
versión”, “Compilando”, “Reiniciando”, “Comprobando que arranca”, “Terminado” and
“Revertido”. The final result is `ok`, `failed` or `rolled_back`.

## Checking the status

```bash
systemctl --user status cinecat-update.path
systemctl --user status cinecat-update.service
journalctl --user -u cinecat-update -f
cat /opt/cinecat/run/update-status.json
tail -n 50 ~/cinecat-backups/updater.log
```

The log is also sent to journald and trimmed to the last 2000 lines in
`~/cinecat-backups/updater.log`. Previous copies are kept in
`~/cinecat-backups/`.

## Manual rollback

If the status ends in `failed`, its message includes the backup and commit to
recover. The procedure is:

```bash
cd /opt/cinecat
git reset --hard <previous-commit>
export COMMIT=$(git rev-parse --short HEAD)
export BUILT_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
export TMPDIR=$HOME/.cache/cinecat-tmp
docker compose build
docker compose stop servidor
IMAGE=$(docker inspect -f '{{.Config.Image}}' cinecat-servidor-1)
docker run --rm \
  -v cinecat_cinecat_data:/data \
  -v "$HOME/cinecat-backups/<backup>.db:/backup.db:ro" \
  --entrypoint sh "$IMAGE" -c \
  'rm -f /data/cinecat.db /data/cinecat.db-wal /data/cinecat.db-shm && cp /backup.db /data/cinecat.db'
docker compose up -d
```

Do not copy only the `.db` file: remove the `-wal` and `-shm` files and restore
the full copy so an old WAL is not reused. Then check `/healthz` and
`/api/v1/system/version`.

## Security

The updater only accepts tags matching the strict `vX.Y.Z` format, does not
interpret the rest of the request and checks that the tag exists in `origin`. It
never executes content written by the server. Requests and status contain no
secrets.

# Docker deployment

[Español](../es/despliegue.md) · [Back to README](../../README.en.md) · [Updates](updates.md)

Cinecat is deployed with Docker Compose: one container for the Go server and one for the web client, published only on `127.0.0.1`. The Debian host's Apache acts as reverse proxy, handles HTTPS and routes `/api/` and `/healthz` to the server.

## Requirements

- Docker with the `docker compose` plugin.
- Git and repository access.
- Apache with `mod_proxy`, `mod_proxy_http` and `mod_headers`.
- For the automatic updater: `sqlite3`, `curl`, `python3`, `flock`, the user in the `docker` group and linger enabled.

## New server

```sh
sudo mkdir -p /opt/cinecat
sudo chown "$USER":"$USER" /opt/cinecat
git clone https://github.com/catlinux/Cinecat.git /opt/cinecat
cd /opt/cinecat
cp .env.deploy.example .env
```

Edit `.env`:

- `CINECAT_MEDIA_PATH`: host path mounted whole and read-only at `/media`.
- `CINECAT_MEDIA_ROOTS`: roots visible inside the container for choosing library folders; `/media` by default.
- `CINECAT_TMDB_API_KEY`: TMDB key for complete metadata; without it, OMDb can be used for movie synopsis and poster only.
- `CINECAT_OMDB_API_KEY`: optional, fallback without TMDB.
- `CINECAT_SCAN_INTERVAL`: automatic interval; `0` disables it.
- `CINECAT_ADMIN_PASSWORD`: password for `admin` on first start.
- `CINECAT_API_PORT` and `CINECAT_WEB_PORT`: only if 8080/8081 are taken.

```sh
mkdir -p run
docker compose up -d --build
docker compose logs servidor
```

The first start creates the `admin` user. If you did not set a password, it appears once in the log: copy it before it is lost. The server listens on `http://127.0.0.1:8080` and the web app on `http://127.0.0.1:8081`, reachable only from the Debian machine.

## Apache and HTTPS

```sh
sudo cp deploy/video.warcrafted.com.conf /etc/apache2/sites-available/
sudo a2ensite video.warcrafted.com.conf
sudo apache2ctl configtest
sudo systemctl reload apache2
sudo certbot --apache -d video.warcrafted.com
```

Certbot copies the `ProxyPass` directives into the `:443` block but does not change `X-Forwarded-Proto`. Inside the HTTPS block, keep:

```apache
RequestHeader set X-Forwarded-Proto "https"
```

Reload Apache. Without that header the session cookie is not marked `Secure`.

## Libraries and first scan

Open the web app, sign in and go to **Administración → Bibliotecas**. Create one library per real folder inside the mount:

- `/media/Multimedia/Pelis` for movies;
- `/media/Multimedia/Series` for series.

Press **Escanear todas las bibliotecas**. The scan continues in the background and the panel shows progress; you can close the page. The container cannot delete or upload files because `/media` is mounted `:ro`.

## Updating

Manual:

```sh
cd /opt/cinecat
./scripts/actualizar.sh
```

It runs `git pull --ff-only`, rebuilds the images and starts them; the database and libraries are kept. The first time you want to update from Administration → System, install the host service:

```sh
cd /opt/cinecat
./scripts/instalar-actualizador.sh
```

The installer checks requirements, creates the user systemd units and warns if linger or Docker access is missing. Details, backups and recovery are in [updates](updates.md).

## Backups

The SQLite database lives in the `cinecat_data` volume. A hot copy:

```sh
mkdir -p "$HOME/cinecat-backups"
docker cp cinecat-servidor-1:/data/cinecat.db "$HOME/cinecat-backups/cinecat.db"
docker cp cinecat-servidor-1:/data/cinecat.db-wal "$HOME/cinecat-backups/cinecat.db-wal" 2>/dev/null || true
docker cp cinecat-servidor-1:/data/cinecat.db-shm "$HOME/cinecat-backups/cinecat.db-shm" 2>/dev/null || true
sqlite3 "$HOME/cinecat-backups/cinecat.db" "VACUUM INTO '$HOME/cinecat-backups/cinecat-copy.db'"
```

The automatic updater already makes a copy before each update and keeps the last 10 in `~/cinecat-backups/`.

## Changing domain or server

1. Copy the repository and `.env`; keep the `cinecat_data` volume or restore a database copy.
2. Copy the vhost, change `ServerName` and run certbot for the new domain.
3. Set `CINECAT_MEDIA_PATH` to the new host's real path.
4. Start the containers and check `/healthz`, `/api/v1/system/version` and the web app.

## Deployment limits

- The Go server listens on plain HTTP on localhost; Apache provides TLS.
- There is no CORS: the web app and API must share an origin through the proxy.
- The automatic updater needs the host service; without it, the web app reports that it is not installed.
- TMDB/OMDb quotas and the TMDB cache TTL should be reviewed before heavy use.

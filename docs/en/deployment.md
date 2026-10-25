# Docker deployment

[Versión en español](../es/despliegue.md) · [Back to README](../../README.md)

Cinecat is deployed with Docker Compose: three containers (Go server, web client, Caddy proxy) behind a single entry point, so session cookies and the HLS player work without manual CORS configuration.

## Requirements

- Docker and Docker Compose (the `docker compose` plugin, not the old standalone `docker-compose`).
- Your video library folder accessible on the machine running Docker.

## First deployment

```sh
git clone git@github-catlinux:catlinux/Cinecat.git
cd Cinecat
cp .env.deploy.example .env
```

Edit `.env` and set at least `CINECAT_MEDIA_PATH` (the real path to your library). The other values are optional at first.

```sh
docker compose up -d --build
docker compose logs servidor
```

The first startup creates the `admin` user. If you didn't set `CINECAT_ADMIN_PASSWORD` in `.env`, the generated password appears **once** in that log: copy it before it scrolls away.

Open `http://<Debian's IP>/` (or `http://localhost/` if you're on the same machine). Log in as `admin`, create a library pointing to `/media` (the container-internal path: Docker maps it to `CINECAT_MEDIA_PATH` for you) and trigger a scan from the interface itself or with:

```sh
curl -X POST http://localhost/api/v1/libraries/scan -H "Authorization: Bearer <your token>"
```

## Applying a new version

Whenever there are changes to try out:

```sh
./scripts/actualizar.sh
```

It runs `git pull`, rebuilds the images and restarts the containers. The database (`cinecat_data` volume) and Caddy's certificates (`caddy_data`/`caddy_config` volumes) are untouched: your library, users and registered libraries survive across updates.

## Custom domain and HTTPS

By default (`CINECAT_DOMAIN=:80`) Caddy serves plain HTTP with no certificate. If you point a domain at the Debian's IP, set that domain in `CINECAT_DOMAIN` (e.g. `cinecat.yourdomain.com`) and Caddy will obtain an automatic HTTPS certificate from Let's Encrypt the first time someone accesses it from outside. Make sure ports 80 and 443 reach the Debian box (router port forwarding) before enabling this.

## What this deployment doesn't cover yet

- **TMDB metadata**: `internal/metadata` isn't wired into the server yet; you'll see titles parsed from filenames, with no posters or synopses.
- **Per-IP login lockout**: the failed-attempt limit is keyed by username, not IP (see `TODO.md`). Fine on a local network; review before exposing the domain to the internet without further protection.
- **Backups**: the `cinecat_data` volume holds the SQLite database; back it up periodically (`docker run --rm -v cinecat_data:/data -v $(pwd):/backup debian tar czf /backup/cinecat-db.tar.gz /data`) if losing registered users and libraries would matter to you.

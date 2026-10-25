# Docker deployment

[Versión en español](../es/despliegue.md) · [Back to README](../../README.md)

Cinecat is deployed with Docker Compose: two containers (Go server, web client), each published only on `127.0.0.1` of the Debian box. The Apache already installed on that machine acts as the reverse proxy for both and handles HTTPS, just like its other vhosts.

## Requirements

- Docker and Docker Compose (the `docker compose` plugin, not the old standalone `docker-compose`).
- Apache with `mod_proxy`, `mod_proxy_http` and `mod_headers` enabled (`a2enmod proxy proxy_http headers`), already running with other vhosts.
- Your video library folder accessible on the machine running Docker.

## 1. Bring up the containers

```sh
git clone git@github-catlinux:catlinux/Cinecat.git
cd Cinecat
cp .env.deploy.example .env
```

Edit `.env` and set at least `CINECAT_MEDIA_PATH` (the real path to your library). The `CINECAT_API_PORT`/`CINECAT_WEB_PORT` ports (8080/8081 by default) only need changing if you already use them for something else on that Debian box.

```sh
docker compose up -d --build
docker compose logs servidor
```

The first startup creates the `admin` user. If you didn't set `CINECAT_ADMIN_PASSWORD` in `.env`, the generated password appears **once** in that log: copy it before it scrolls away.

At this point the server already answers at `http://127.0.0.1:8080` and the web client at `http://127.0.0.1:8081`, but only from the machine itself: the next step is the Apache vhost that makes them reachable from outside.

## 2. Apache vhost

The repository's `deploy/video.warcrafted.com.conf` already has the configuration for `video.warcrafted.com`. Copy it to `sites-available` and enable it:

```sh
sudo cp deploy/video.warcrafted.com.conf /etc/apache2/sites-available/
sudo a2ensite video.warcrafted.com.conf
sudo apache2ctl configtest
sudo systemctl reload apache2
```

For HTTPS, use certbot just like with your other vhosts (it adds the `:443` block and the redirect automatically):

```sh
sudo certbot --apache -d video.warcrafted.com
```

Certbot copies the `ProxyPass` directives into the new `:443` block, but doesn't update the `X-Forwarded-Proto` header. Edit `/etc/apache2/sites-available/video.warcrafted.com-le-ssl.conf` (or whatever file certbot created) and change, inside `<VirtualHost *:443>`:

```apache
RequestHeader set X-Forwarded-Proto "https"
```

then reload Apache (`sudo systemctl reload apache2`). If you skip this, the session cookie won't be marked `Secure` over HTTPS.

Open `https://video.warcrafted.com/`. Log in as `admin`, create a library pointing to `/media` (the container-internal path: Docker maps it to `CINECAT_MEDIA_PATH` for you) and trigger a scan from the interface itself or with:

```sh
curl -X POST https://video.warcrafted.com/api/v1/libraries/scan -H "Authorization: Bearer <your token>"
```

## Applying a new version

Whenever there are changes to try out:

```sh
./scripts/actualizar.sh
```

It runs `git pull`, rebuilds the images and restarts the containers. The database (`cinecat_data` volume) is untouched: your library, users and registered libraries survive across updates. The Apache vhost doesn't change unless `deploy/video.warcrafted.com.conf` itself changes in a commit (then it needs to be copied over and Apache reloaded by hand again).

## What this deployment doesn't cover yet

- **TMDB metadata**: `internal/metadata` isn't wired into the server yet; you'll see titles parsed from filenames, with no posters or synopses.
- **Per-IP login lockout**: the failed-attempt limit is keyed by username, not IP (see `TODO.md`). Review before relying on this alone for security with the domain already exposed to the internet.
- **Backups**: the `cinecat_data` volume holds the SQLite database; back it up periodically (`docker run --rm -v cinecat_data:/data -v $(pwd):/backup debian tar czf /backup/cinecat-db.tar.gz /data`) if losing registered users and libraries would matter to you.

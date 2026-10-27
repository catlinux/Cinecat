# Cinecat

> Own server and clients (web, Android and Android TV) for your media library: Spanish-language metadata, direct play or HLS, and an interface with a personality of its own.

**Languages / Idiomas:** [Español](README.md) · English

**Status:** server and web client implemented; Android and Android TV clients not started. Version `0.18.0`.

Cinecat is for people who keep their own video library at home and want a carefully designed interface, control over playback and Spanish-language metadata without depending on an external service. It is not a clone: it reuses what is legal from other projects, with credits, and writes the rest from scratch.

## What already exists

- **Library and scanning:** local libraries, filename parsing for movies, series, seasons and episodes, manual and automatic background scanning, and technical file information through `ffprobe`.
- **Identification and detail pages:** TMDB in Spanish; OMDb as a fallback when there is no TMDB key; automatic, manual and "not this one" states; search, filters by letter, genre and watched state, and sorting; cast and crew with photos, synopsis, posters, backdrops and IMDb/TMDB links.
- **Series:** one detail page per folder, seasons and episodes with title, synopsis and image, next episode, watched state per episode, season or series, and manual correction.
- **Collections:** automatic TMDB sagas and manual collections, with editing, ordering, poster and backdrop.
- **Playback:** direct play for MP4/M4V; HLS for the rest, re-encoded to H.264 + stereo AAC, with seeking, audio tracks, text subtitles as WebVTT, quality choices and per-user, per-movie preferences.
- **Web interface:** portal with "Continue watching", "Next episode" and "Recently added"; movie, series and collection lists with search and filters; full detail pages; custom player with keyboard shortcuts; light, dark or system theme; side or top navigation.
- **Administration:** summary, libraries, scan progress, hot settings, metadata correction, images and updates from System.
- **Updates:** the server checks tags and release notes; "Update now" hands the request to a host systemd service that backs up the database, updates, checks and rolls back on failure. See [updates](docs/en/updates.md).

What is still missing, in detail and in order, is in [status](docs/en/status.md).

## Requirements

- Docker and Docker Compose.
- The server container already includes FFmpeg; it does not need to be installed on the host.
- A TMDB key is optional, but without it there will be no complete TMDB metadata. If TMDB is absent, OMDb can be used for movie synopsis and poster only.
- For deployment with a domain, Apache with `mod_proxy`, `mod_proxy_http` and `mod_headers`.

Server variables (all optional): `CINECAT_CONFIG`, `CINECAT_LISTEN_ADDR` (`:8080`), `CINECAT_DB_PATH` (`cinecat.db`), `CINECAT_TMDB_API_KEY`, `CINECAT_OMDB_API_KEY`, `CINECAT_SCAN_INTERVAL` (`6h`), `CINECAT_MEDIA_ROOTS` (`/media`), `CINECAT_ADMIN_PASSWORD` and `CINECAT_RUN_DIR` (`/run/cinecat-update`). Docker additionally uses `CINECAT_MEDIA_PATH`, `CINECAT_API_PORT` and `CINECAT_WEB_PORT`.

## Installation and first start

```sh
git clone https://github.com/catlinux/Cinecat.git
cd Cinecat
cp .env.deploy.example .env
# Edit .env: at least CINECAT_MEDIA_PATH and, if you want, the TMDB/OMDb keys.
docker compose up -d --build
docker compose logs servidor
```

The first start creates the `admin` user. If you did not set `CINECAT_ADMIN_PASSWORD`, the generated password appears once in that log. The containers listen only on `127.0.0.1`; the Apache vhost and HTTPS setup are explained in [deployment](docs/en/deployment.md).

## Updating

Manual update:

```sh
./scripts/actualizar.sh
```

Update from Administration → System, with a database backup and automatic rollback if the new version does not start; install the host service once:

```sh
./scripts/instalar-actualizador.sh
```

Details, systemd/linger requirements, backups and manual recovery are in [updates](docs/en/updates.md).

## Visual direction

**"Marquesina"**: classic cinema with a modern, bright feel. Maroon and light cream, gold and orange accents, Fraunces serif for titles and illustrated posters of our own; never third-party photos. The web app bundles its typefaces locally and offers light, dark or system themes.

## Screenshots

<!-- capturas: pendiente de D2 -->

## Documentation

| Topic | Español | English |
|---|---|---|
| Status and what is missing | [estado](docs/es/estado.md) | [status](docs/en/status.md) |
| User guide | [guía de usuario](docs/es/guia-de-usuario.md) | [user guide](docs/en/user-guide.md) |
| Administration | [administración](docs/es/administracion.md) | [administration](docs/en/administration.md) |
| Vision and decisions | [visión](docs/es/vision.md) | [vision](docs/en/vision.md) |
| Architecture and security | [arquitectura](docs/es/arquitectura.md) | [architecture](docs/en/architecture.md) |
| Playback and codecs | [reproducción](docs/es/reproduccion.md) | [playback](docs/en/playback.md) |
| Metadata | [metadatos](docs/es/metadatos.md) | [metadata](docs/en/metadata.md) |
| Updates | [actualizaciones](docs/es/actualizaciones.md) | [updates](docs/en/updates.md) |
| Docker deployment | [despliegue](docs/es/despliegue.md) | [deployment](docs/en/deployment.md) |
| Roadmap | [hoja de ruta](docs/es/hoja-de-ruta.md) | [roadmap](docs/en/roadmap.md) |
| Licenses and credits | [licencias y créditos](docs/es/licencias-y-creditos.md) | [licenses and credits](docs/en/licenses-and-credits.md) |

Other: [CHANGELOG](CHANGELOG.en.md) · [Open tasks](TODO.en.md) · [Contributing](CONTRIBUTING.en.md) · [Security](SECURITY.md) · [Credits](CREDITS.md) · [Authors](AUTHORS)

## License

**GPL-3.0**, with an additional term (permitted under its section 7) requiring Catlinux's authorship notices to be preserved in any copy or modified version. See [LICENSE](LICENSE) and [licenses and credits](docs/en/licenses-and-credits.md).

## Authorship

Author and maintainer: **Catlinux** ([github.com/catlinux](https://github.com/catlinux)).

## Data attribution

This product uses the TMDB API but is not endorsed or certified by TMDB.
(Once clients exist, this notice and the TMDB logo will appear on the "About" screen.)

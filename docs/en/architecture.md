# Architecture and security

[Español](../es/arquitectura.md) · [Back to README](../../README.en.md) · [Status](status.md)

Cinecat has a Go server, a SvelteKit web client and a Docker deployment with Apache in front. Android and Android TV apps are part of the vision but do not exist yet. This page describes what is built and clearly separates what is not.

## Overview

```
 Browser / future clients
            │ HTTPS
            ▼
 ┌─────────────────────┐
 │ Host Apache         │  /api/ and /healthz → server
 │ (vhost + TLS)       │  everything else   → web
 └───────┬─────────────┘
         │ 127.0.0.1
   ┌─────▼──────┐      ┌──────────────┐
   │  Go server │      │ web container│
   │  :8080     │      │ :3000        │
   └──┬───┬───┬─┘      └──────────────┘
      │   │   └── FFmpeg/ffprobe (server image)
      │   └────── SQLite + TMDB cache (/data)
      └────────── video library (/media, read-only)
```

The updater does not live inside the containers: the Go server writes `run/update-request.json` and a host systemd service does the work. See [updates](updates.md).

## Components

- **Go server (`cmd/cinecatd`):** REST API, authentication, scanning, identification, metadata, collections, series, progress, settings and playback preparation.
- **Web client (`web/`):** SvelteKit + TypeScript + Svelte 5, with the session in an `httpOnly` cookie; it never stores the API token in JavaScript. It uses hls.js for HLS and native `<video>` for direct play.
- **SQLite database:** embedded migrations, `STRICT`, WAL, foreign keys and `busy_timeout`. It lives in the `cinecat_data` volume.
- **TMDB cache:** next to the database (`/data/cache/tmdb` in Docker), with raw responses and a 7-day TTL.
- **FFmpeg/ffprobe:** external tools installed in the server image, not distributed in the repository. They are always invoked with argument lists, never through a shell.
- **Host Apache:** reverse proxy and TLS. It routes `/api/` and `/healthz` to the server and everything else to the web client, so browser and API share an origin.

## Data flow

1. A library is created with a container path (`/media/...`).
2. The scan walks video files, parses names and folders and upserts them into `media_items`.
3. Movies and series are identified with TMDB (or OMDb as a movie fallback), and metadata, cast, images and collections are stored.
4. The web app reads the API with the session cookie; the server turns the cookie into an internal Bearer token.
5. When playing, the server chooses direct play or HLS and returns a signed URL. The browser requests the file or the playlist and segments.
6. Progress, watched state and preferences are stored per user in SQLite.

## Main data model

| Table | Content |
|---|---|
| `users`, `sessions` | Users, roles and sessions; only the SHA-256 hash of the token is stored. |
| `libraries` | Name and path of each library. |
| `media_items` | Movies and episodes, path, title, metadata, TMDB, match state, images, credits and series/season/episode. |
| `series`, `series_seasons` | Series and seasons; episodes are rows in `media_items`. |
| `collections`, `collection_items` | Automatic/manual collections, members, order and exclusions. |
| `watch_state`, `playback_progress` | Watched state and position per user. |
| `item_preferences` | Audio, subtitle and quality per user and movie. |
| `settings` | Typed persisted settings; effective values are held in memory. |

## API and contract

The API is versioned under `/api/v1/` and documented in [api/openapi.yaml](../../api/openapi.yaml). The main groups are authentication, libraries, items, series, collections, settings, system/update and signed streaming. `GET /healthz` also exists as a health check.

Error responses use `{error}` or `{message}` depending on the endpoint; common codes are 401 without a session, 403 without the role, 400 validation, 404, 409 conflict, 415 unsupported container, 502 external dependency and 503 FFmpeg or update unavailable.

## Playback

- **Direct:** `.mp4` and `.m4v` are served with `http.ServeContent`, including range support.
- **HLS:** other extensions are re-encoded to H.264 + stereo AAC and served in 6 s MPEG-TS segments. The playlist is VOD when the duration is known; otherwise FFmpeg's `event` playlist is used.
- **Control:** maximum simultaneous transcodes, low-priority processes, idle stop and a limit on segments generated ahead. See [playback](playback.md).

## Security

### Implemented

- **Argon2id** passwords; 24 h sessions with sliding renewal; 32-byte token with only its hash stored.
- `admin` and `user` roles; admin endpoints check the role on the server.
- Session cookie `httpOnly`, `SameSite=Lax` and `Secure` when received over HTTPS.
- Lockout after five failed attempts per user+IP pair, with an additional limit of 20 failures per IP, and login messages that do not reveal whether the user exists. The IP comes from `RemoteAddr`; `X-Forwarded-For` is only accepted from the ranges in `CINECAT_TRUSTED_PROXIES`.
- Playback URLs signed with HMAC-SHA256, 8-hour TTL and a per-process key.
- Path validation inside the library and scanning without following symlinks.
- No telemetry. Outgoing connections go to TMDB, OMDb and GitHub only when relevant.

### Known limits

- Login lockout lives in process memory and is not shared across instances; if the proxy does not forward the IP or the trusted ranges are wrong, it falls back to the direct origin.
- There is no CORS: the deployment relies on Apache unifying the origin.
- There are no per-library permissions: any authenticated user sees the whole catalogue.
- There is no user creation or session revocation from the web app.
- `/media` is mounted read-only; there is no file upload or deletion.
- Playback URLs are not bound to the user and stop working after a server restart.

Details and the improvement order are in [status](status.md) and [open tasks](../../TODO.en.md).

## Technology stack

| Part | Choice | State |
|---|---|---|
| Server | Go, SQLite, external FFmpeg | Implemented. |
| Web | TypeScript + SvelteKit + hls.js | Implemented. |
| Android / Android TV | Kotlin + Jetpack Compose + Media3/ExoPlayer, optional libmpv | Decided, no code. |
| Deployment | Docker Compose + host Apache | Implemented. |

## Security decisions

The platform may be exposed to the internet, so HTTPS, strong passwords and a minimal surface are requirements. PIN profiles and per-library permissions remain design goals; they should not be confused with features that are already available.

See also [SECURITY.md](../../SECURITY.md) and [licenses and credits](licenses-and-credits.md).

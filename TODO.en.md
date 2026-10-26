# TODO

[Versión en español](TODO.md)

Priorities: 🔴 blocking · 🟠 next phase · 🟢 later.

## Before writing code
- 🟠 Enable private vulnerability reporting on GitHub.
- 🔴 The login failed-attempt lockout (`internal/auth`) is keyed by username, not IP: anyone can lock out another legitimate user's account by sending 5 failed logins with their `username`. Review before exposing the server to the internet (e.g. combine username+IP or add a per-IP limit).
- 🟠 There is no way yet to create users (no endpoint or command); testing login requires inserting the user and their argon2id hash directly into the database.
- 🟠 The API scan (`POST /api/v1/libraries/scan`) is synchronous: with large libraries the request can take a long time. The asynchronous version with progress is still pending.
- 🟠 Series and episodes still have no plot or poster: OMDb is only queried for movies and the TMDB client (`internal/metadata/tmdb.go`) remains unwired for the future.
- The TMDB cache TTL (7 days, `internal/metadata`) is a starting value not checked against the API's current terms: review before production.
- 🟠 The `GET /api/v1/libraries|items` endpoints do not filter by role or by library: any authenticated user sees everything. The architecture plans per-library permissions (Phase 3+); implement before real multi-user use.
- 🟠 The free OMDb key allows 1000 requests/day: a first full scan of a large library may exhaust the quota. Items without metadata are retried on later scans (there is a negative disk cache).
- `internal/metadata` (TMDB) is still not wired into the endpoints or the scanner; `media_items.tmdb_id` is only filled in by hand for now.
- Small duplication: `bearerToken()` is repeated in `internal/auth/http.go` and `internal/library/http.go`; could be extracted to a shared spot once there's a third consumer.
- 🔴 HLS remux/transcode tests are skipped with `t.Skip()` when `ffmpeg`/`ffprobe` are not in PATH. Run the full suite on the production/CI machine with FFmpeg before relying on playback of `.mkv`/`.avi`/`.mov`/`.ts`/`.m2ts`.
- 🟠 HLS generation is progressive: Cinecat writes a complete VOD playlist with `ENDLIST` from the start and serves each segment as FFmpeg produces it. If the container duration cannot be read (some old AVI files), it falls back to FFmpeg's `event` playlist and the client treats it as live; an alternative way to compute duration in that case is still missing.
- 🟠 Transcoding (every `.avi`, plus any other container when the keyframe interval requires it) uses software encoding (libx264 `veryfast`) without hardware acceleration: it may be slow on modest servers. Hardware acceleration is left for a later Phase 3 step.
- The keyframe interval analysis (`internal/playback/probe.go`) only decides for containers other than `.avi`; it hasn't been checked whether other old containers (`.mov`/`.ts`/`.m2ts` of dubious origin) have the same `N/A` timestamp problem that forced always transcoding `.avi`. Revisit if another real case shows up.
- 🟠 `/api/v1/items/{id}/play` still returns `mode: "remux"` even when the server transcodes the video for HLS; from the client's point of view it is still HLS. Exposing a `transcode` mode is a pending improvement.
- 🟠 The remux only copies video and audio; subtitles are dropped (to be addressed in Phase 3, per the playback design).
- The web client (`web/`) decides the `Secure` cookie flag from `event.url.protocol`; the Docker deployment (`docker-compose.yml`, see [deployment](docs/en/deployment.md)) already passes `PROTOCOL_HEADER=x-forwarded-proto` to the `web` container, and the Apache vhost (`deploy/video.warcrafted.com.conf`) sets that same header. Solved for this deployment; revisit if it's ever served without that proxy.
- The web player uses hls.js for remux, which fetches the playlist and segments via XHR against `PUBLIC_API_URL`; the Docker deployment avoids the CORS problem by serving the server and client under the same origin (the Debian's Apache routes `/api/*` to the Go server and everything else to the web client, with `PUBLIC_API_URL` empty). The Go server still doesn't send CORS headers: if it's ever deployed without that proxy unifying the origin, they'll need to be added.

## Known, accepted risks
- There is an inactive Android app from 2015 called "CineCat" (F-Droid, Catalan film catalog using Generalitat open data; no trademark registration). Same name, different scope. Keeping the name Cinecat; minor confusion risk, no action pending.

## Product
- 🟠 Minimal server: library, movie and series identification, `es-ES` metadata.
- 🟠 Playback decision engine (direct / remux / transcoding).
- 🟠 Web client with login.
- 🟠 Android and Android TV clients.
- 🟢 Photos and slideshows.
- 🟢 Internet content and live TV (only if decided).
- 🟢 Intro/credits segments, offline downloads, cross-device playback sync.
- 🟢 Other platforms (Apple, Samsung/LG TVs).

## Documentation
- 🟠 User guide and plain-language glossary once there is something to try.
- 🟠 Code of conduct.
- 🟢 Web client screenshots.

## Document maintenance
- Keep the Spanish and English versions of every document in sync.

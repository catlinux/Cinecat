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
- 🟠 HLS tests are skipped with `t.Skip()` without `ffmpeg`/`ffprobe` in PATH (as on the Windows PC). They can be run in the production container by building the test binary for Linux (`GOOS=linux go test -c ./internal/playback`) and running it next to the package `.go` files. Automating this in CI is pending.
- 🟠 HLS generation is progressive: Cinecat writes a complete VOD playlist with `ENDLIST` from the start and restarts FFmpeg at the requested segment on seek. If the container duration cannot be read (some old AVI files), it falls back to FFmpeg's `event` playlist and the client treats it as live; an alternative way to compute duration in that case is still missing.
- 🟠 All HLS is software re-encoded (libx264 `veryfast`), even H.264 MKV files that could be copied: on SD it runs at about 20x real time on the Xeon E3-1225 v5, but 1080p/HEVC may be tight. Pending: copy the video when it is already 8-bit H.264 (requires cutting segments at its real keyframes) and use Quick Sync/VAAPI (the server has `/dev/dri`, but it is not passed to the container).
- 🟠 FFmpeg is not throttled: after a seek it encodes up to the end of the video even if nobody asks (on SD, ~1 GB in the container `/tmp` per movie, deleted after 1 h unused). Jellyfin throttles it when far ahead; assess whether it is needed.
- 🟢 With two audio tracks (for example Spanish and English) the first one is always used; track selection is missing.
- The keyframe analysis in `internal/playback/probe.go` is no longer used (all HLS is re-encoded); it is kept for when remux without re-encoding comes back.
- 🟠 `/api/v1/items/{id}/play` still returns `mode: "remux"` even when the server transcodes the video for HLS; from the client's point of view it is still HLS. Exposing a `transcode` mode is a pending improvement.
- 🟠 HLS only carries video and audio; subtitles are dropped (to be addressed in Phase 3, per the playback design).
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

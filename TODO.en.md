# TODO

[Versión en español](TODO.md)

Priorities: 🔴 blocking · 🟠 next phase · 🟢 later.

## Before writing code
- 🟠 Enable private vulnerability reporting on GitHub.
- 🔴 The login failed-attempt lockout (`internal/auth`) is keyed by username, not IP: anyone can lock out another legitimate user's account by sending 5 failed logins with their `username`. Review before exposing the server to the internet (e.g. combine username+IP or add a per-IP limit).
- 🟠 There is no way yet to create users (no endpoint or command); testing login requires inserting the user and their argon2id hash directly into the database.
- 🟠 The API scan (`POST /api/v1/libraries/scan`) is synchronous: with large libraries the request can take a long time. The asynchronous version with progress is still pending.
- 🟠 The metadata client (`internal/metadata`) only searches movies (`/search/movie`); the TMDB series/episode endpoint is still missing to identify `media_items.kind = 'series'/'episode'`. Also not yet wired into the scanner or the server.
- The TMDB cache TTL (7 days, `internal/metadata`) is a starting value not checked against the API's current terms: review before production.
- 🟠 The `GET /api/v1/libraries|items` endpoints do not filter by role or by library: any authenticated user sees everything. The architecture plans per-library permissions (Phase 3+); implement before real multi-user use.
- `internal/metadata` (TMDB) is still not wired into the endpoints or the scanner; `media_items.tmdb_id` is only filled in by hand for now.
- Small duplication: `bearerToken()` is repeated in `internal/auth/http.go` and `internal/library/http.go`; could be extracted to a shared spot once there's a third consumer.
- 🔴 HLS remuxing (`internal/playback`) has never been tested with real FFmpeg: the development machine used in Phase 1 has no FFmpeg installed and those tests were always skipped with `t.Skip()`. Run the full suite on a machine with FFmpeg before relying on playback of `.mkv`/`.avi`/`.mov`/`.ts`/`.m2ts`.
- 🟠 The remux is generated fully and synchronously on the first playlist request: a long file takes a while to start playing. On-demand segmentation is left for later.
- 🟠 The remux only copies video and audio; subtitles are dropped (to be addressed in Phase 3, per the playback design).

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
- 🟢 Screenshots and Docker deployment guide.

## Document maintenance
- Keep the Spanish and English versions of every document in sync.

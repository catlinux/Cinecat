# TODO

[Versión en español](TODO.md)

Priorities: 🔴 blocking · 🟠 next phase · 🟢 later.

## Before writing code
- 🟠 Enable private vulnerability reporting on GitHub.
- 🔴 The login failed-attempt lockout (`internal/auth`) is keyed by username, not IP: anyone can lock out another legitimate user's account by sending 5 failed logins with their `username`. Review before exposing the server to the internet (e.g. combine username+IP or add a per-IP limit).
- 🟠 There is no way yet to create users (no endpoint or command); testing login requires inserting the user and their argon2id hash directly into the database.
- 🟠 There are no endpoints or CLI yet to register libraries (`libraries`) or to trigger a scan (`internal/library`); the package is ready to be wired up in Step 6 (API) or earlier.

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

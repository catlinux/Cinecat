# TODO

[Versión en español](TODO.md)

Priorities: 🔴 blocking · 🟠 next phase · 🟢 later.

## Before writing code
- 🔴 Define the technology stack for web and Android (proposals in [architecture](docs/en/architecture.md); server already decided: Go).

## Known, accepted risks
- There is an inactive Android app from 2015 called "CineCat" (F-Droid, Catalan film catalog using Generalitat open data; no trademark registration). Same name, different scope. Keeping the name Cinecat; minor confusion risk, no action pending.
- 🟠 Create `CLAUDE.md` with the project rules.
- 🟠 Enable private vulnerability reporting on GitHub.

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

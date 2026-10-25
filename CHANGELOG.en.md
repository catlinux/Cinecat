# Changelog

All notable changes to this project are documented here.
Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versioned according to [Semantic Versioning](https://semver.org/) (`x.y.z`).

[Versión en español](CHANGELOG.md)

## [Unreleased]

## [0.3.2] - 2026-10-25

### Fixed
- The web client rejected every API request with "PUBLIC_API_URL is not configured on the server", including login, when `PUBLIC_API_URL` was deliberately left empty to use relative paths behind the Docker deployment's proxy (see `0.3.1`). `createApiClient` (`web/src/lib/api-factory.ts`) no longer treats an empty string as a configuration error; `web/src/routes/login/+page.server.ts` now uses that same function instead of duplicating the logic with its own check.

## [0.3.1] - 2026-10-25

### Changed
- The Docker deployment no longer ships its own Caddy proxy: the Apache that already manages other vhosts on the Debian box acts as the reverse proxy for the server and web client (sample vhost in `deploy/video.warcrafted.com.conf`). `docker-compose.yml` publishes both containers only on `127.0.0.1` (`CINECAT_API_PORT`/`CINECAT_WEB_PORT` ports, never exposed directly to the network). `.env.deploy.example` and `docs/es/despliegue.md`/`docs/en/deployment.md` updated with the two steps (containers + Apache vhost).

## [0.3.0] - 2026-10-25

### Added
- Docker Compose deployment: Go server `Dockerfile` (with FFmpeg installed in the image), web client `web/Dockerfile`, `docker-compose.yml` with Caddy as the single proxy in front of both (same origin, no CORS needed), `scripts/actualizar.sh` script to apply new versions without losing data, and a deployment guide at `docs/en/deployment.md`.
- `CINECAT_ADMIN_PASSWORD` support documented in the deployment flow to set the administrator's password on first startup without relying on the log.
- Real version embedded in the server binary via the `Dockerfile` (previously showed "unknown" in a binary built without `-ldflags`).

### Changed
- `docker-compose.yml` passes `PROTOCOL_HEADER=x-forwarded-proto` to the web client container (a variable already supported by `@sveltejs/adapter-node`), so the session cookie is correctly marked `Secure` behind Caddy.

## [0.2.0] - 2026-10-25

Closes Phase 2: functional web client with login, library, item pages and player.

### Added
- Switchable light/dark/system theme persisted in `localStorage`, applied before first paint to avoid flashes. Dark "Marquesina" palette consistent with the light one, reviewed text contrast (≥ 4.5:1), unified visible focus for links, buttons, fields and video, and an accessible switcher on every page.

## [0.2.0-dev.4] - 2026-10-25

### Added
- Real web player on the playback route: native `<video>` with controls for `mode: "direct"` and hls.js 1.7.3 (Apache-2.0) for `mode: "remux"`, with native HLS on Safari/iOS, a clear message when the browser cannot play HLS, `hls.destroy()` cleanup on unmount, and dark cinema styling around the video.

## [0.2.0-dev.3] - 2026-10-25

### Added
- Library screen on the landing page (grid of `PosterCard` with a typographic fallback poster and a stable colour derived from the title) and item detail page with metadata and a "Play" button leading to a provisional playback route that shows the mode and the signed URL. Protected pages redirect to `/login` without a session.

## [0.2.0-dev.2] - 2026-10-25

### Added
- Typed API client and session management for the web client: login and logout through SvelteKit form actions, token stored in an httpOnly cookie with SameSite=Lax and Secure over HTTPS, server URL via `PUBLIC_API_URL`, and a "Marquesina" login page with messages for wrong credentials and temporary lockout.

## [0.2.0-dev.1] - 2026-10-25

### Added
- Web client skeleton in `web/`: SvelteKit + TypeScript with `@sveltejs/adapter-node`, Fraunces and Inter typefaces bundled locally (OFL-1.1), and the "Marquesina" visual direction on the landing page. The server base URL is configured with `PUBLIC_API_URL` (see `web/.env.example`).

## [0.1.3] - 2026-10-25

### Added
- `POST /api/v1/libraries/scan`, restricted to administrators: triggers a synchronous scan of all libraries and returns the number of libraries walked, files found and entries updated, along with warnings for unreadable paths.

## [0.1.2] - 2026-10-25

### Added
- `POST /api/v1/libraries` and `DELETE /api/v1/libraries/{id}`, restricted to admin-role users. Validates that the path exists and is a directory; 409 if a library already uses that path; deletion cascades to remove its `media_items`.

### Changed
- `requireSession` can now also require a specific role (`requireAdmin`), without changing the behavior of existing endpoints.

### Added
- Automatic creation of the first administrator user when the server starts against a database with no users: password from `CINECAT_ADMIN_PASSWORD` or randomly generated (144 bits) and shown once in the log.

## [0.1.0] - 2026-10-25

Closes Phase 1: functional minimal server (no clients yet).

### Added
- Go server skeleton (`cinecatd`): module, file/env configuration, `--version` flag and `GET /healthz` endpoint (Phase 1, step 1).
- SQLite database schema (`users`, `sessions`, `libraries`, `media_items`, `playback_state`) and embedded, idempotent migrations (Phase 1, step 2). Driver: `modernc.org/sqlite` (BSD-3-Clause).
- Authentication with argon2id, 24h renewable sessions (sliding expiry) and temporary lockout after 5 failed attempts; `POST /api/v1/auth/login` and `POST /api/v1/auth/logout` endpoints (Phase 1, step 3). Dependency: `golang.org/x/crypto` (BSD-3-Clause).
- Library scanning and a custom file/folder name parser (title, year, season and episode), with upsert into `media_items` that avoids duplicates on rescans (Phase 1, step 4). `season`/`episode` columns added to the schema.
- TMDB metadata client (es-ES): movie search, poster selection by language priority (es-ES → generic Spanish → English → no language) and disk cache with configurable TTL for searches and images (Phase 1, step 5). Not yet wired into the scanner or the endpoints; only searches movies, series are left for later.
- OpenAPI contract (`api/openapi.yaml`) and `GET /api/v1/libraries`, `GET /api/v1/items` (with pagination) and `GET /api/v1/items/{id}` endpoints, protected with Bearer session auth (Phase 1, step 6).
- Direct playback (`http.ServeContent`, with HTTP range support) for `.mp4`/`.m4v` and remux to fMP4 HLS without re-encoding (codec copy) for `.mkv`/`.avi`/`.mov`/`.ts`/`.m2ts`, invoking FFmpeg as an external process with arguments always passed as a list. Playback URLs signed with HMAC-SHA256 (process key, never the session key) and a 15-minute expiry. `GET /api/v1/items/{id}/play`, `GET /api/v1/stream/{id}` and `GET /api/v1/stream/{id}/{file}` endpoints (Phase 1, step 7).

### Verification note
- The development environment used for this phase does not have FFmpeg installed: the tests exercising real remuxing (`TestReproduccionDirectaMP4`, `TestRemuxMKVGeneraHLS`) were skipped with `t.Skip()` in every review of this phase and **have never actually run**. The rest of the playback tests (signing, expiry, mode decision, absence of a shell in the code) did run and pass. Before relying on the remux in production, run the full suite on a machine with FFmpeg installed.

## [0.0.5] - 2026-10-25

### Added
- Phase 0 technology stack completed: web in **TypeScript + SvelteKit** + hls.js; Android/Android TV in **Kotlin + Jetpack Compose** + Media3/ExoPlayer (optional libmpv).

## [0.0.4] - 2026-10-24

### Added
- Project `CLAUDE.md` with the decisions already made, the third-party license table and what not to do.
- "Cinecat" name check: no trademark registration and no conflicting repository; documented as an accepted risk that an Android app inactive since 2015 uses the same name.

## [0.0.3] - 2026-10-24

### Added
- **License decided: GPL-3.0**, with the additional section 7 term requiring Catlinux's authorship notices to be preserved. `LICENSE` file at the repository root.
- Table of verified licenses (checked on each project's official repository) for the reuse candidates: Jellyfin and its clients (GPL-2.0), Findroid and Intro Skipper (GPL-3.0), Streamyfin (MPL-2.0), Media3/hls.js/Shaka (Apache-2.0), Vidstack (MIT).
- Server technology stack decided: **Go**.

### Changed
- README, `docs/es` and `docs/en` (licenses-and-credits, vision, architecture, roadmap) and `TODO` updated to reflect the now-decided license and server stack.

## [0.0.2] - 2026-10-24

### Added
- Chosen visual direction for the interface: "Marquesina" (1950s-60s neighborhood cinema, Fraunces serif, our own illustrated posters), with a link to the reference mockup.
- `VERSION` file.

## [0.0.1] - 2026-10-24

### Added
- Initial bilingual documentation (Spanish and English): vision, architecture, playback and codecs, metadata, licenses and credits, and roadmap.
- `README`, `TODO`, `CONTRIBUTING`, `SECURITY`, `AUTHORS` and `CREDITS`.
- Starting decisions: own server and clients; web, Android and Android TV; H.265 preferred with H.264 as a fallback; Spanish (Spain) metadata from existing databases.

[Unreleased]: https://github.com/catlinux/cinecat/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/catlinux/cinecat/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/catlinux/cinecat/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/catlinux/cinecat/compare/v0.0.5...v0.1.0
[0.0.5]: https://github.com/catlinux/cinecat/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/catlinux/cinecat/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/catlinux/cinecat/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/catlinux/cinecat/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/catlinux/cinecat/releases/tag/v0.0.1

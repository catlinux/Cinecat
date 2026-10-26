# Changelog

All notable changes to this project are documented here.
Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versioned according to [Semantic Versioning](https://semver.org/) (`x.y.z`).

[Versión en español](CHANGELOG.md)

## [Unreleased]

## [0.16.1] - 2026-10-27

### Fixed

- The updater installer sets the systemd user bus variables when they are missing (entering with `su` or without a login session) and reports an incomplete installation if it cannot enable the units, instead of claiming success. It also checks the requirements first (programs, Docker access and that `git` can reach `origin` without asking for a password) and the documentation now includes a guide for a new server or a domain change.

## [0.16.0] - 2026-10-27

### Added

- Administration → System: "Check for updates" asks the server and shows the new version's notes; "Update now" (with confirmation) asks the host service to update and follows its progress by phases, including the restart, until "Updated to X" or a notice that it rolled back to the previous version.
- Host updater service (`scripts/cinecat-updater.sh`, systemd user units and `scripts/instalar-actualizador.sh`): picks up the server's request, backs up the database, updates to the requested tag, checks it starts and, on failure, rolls back the code and, if the containers had already been restarted, the database too. Documented in `docs/en/updates.md`.

### Changed

- The web's update check no longer reads `VERSION` from GitHub nor asks to run a command by hand: it uses the server's.

## [0.15.0] - 2026-10-27

### Changed

- Cast and crew: actors first, then directing, writing, production, etc.
- "More like this" moves to the bottom of the page (after seasons and episodes on series).
- Narrower side menu pinned to the edge, wider content with smaller margins; admin cards aligned and equal in size.
- "List columns" (`ui.grid_columns`) and "Rows per page" (`ui.grid_rows`) replace `ui.page_size`: results per page are columns × rows. New `ui.row_columns` for the posters visible per portal row (6 by default).

### Added

- Season posters in the season selector and in each season header on the series page (series poster as fallback); the next scan fills them in without re-fetching episodes.
- Portal episode cards with a still (or the series poster) and the series name.

## [0.14.0] - 2026-10-26

### Added

- Update check: the server queries GitHub `vX.Y.Z` tags (opt-in via `system.check_updates`, or on demand) and gives administrators `GET /api/v1/system/update`, `POST …/check` and `POST …/apply`; the latter drops the order in a directory shared with the host (`CINECAT_RUN_DIR`). The host service that carries it out ships in a later version.
- Rows with ‹ › arrows and no scrollbar; the portal no longer overflows horizontally (responsive category grid).
- Cast and crew with photos (initials fallback portrait) and a "More like this" row on movie and series pages.
- Left-hand navigation menu (`ui.nav_layout` setting: side or top); the web now applies the public `ui.*` settings (name, default theme, accent, show collections and series).
- Tagline, portal title and intro editable from the admin panel (`ui.tagline`, `ui.portal_title`, `ui.portal_intro`).

### Changed

- Lighter look (near-white cream, no beige veil); the "neighborhood cinema" wording is gone: new tagline "Sesión continua" and title "Tu cine en casa".

## [0.13.1] - 2026-10-26

### Fixed

- The web failed to build in the container: episode navigation was exported from `+page.server.ts`, which SvelteKit does not allow. It now lives in `$lib/episode-navigation.ts`.

## [0.13.0] - 2026-10-26

### Added
- Player with its own controls: play/pause, ±10 s, progress bar with time on hover, volume and mute, speed from 0.5× to 2×, picture-in-picture, fullscreen and "Next episode" with a countdown. Keyboard shortcuts (Space/K, J/L, arrows, M, F, C, 0-9). Without JavaScript it keeps the native controls.
- Audio track, subtitle and quality menus in the player (changing audio or quality reloads at the current second) and Audio, Subtitles and Quality selectors on the movie page, with the preference saved per user and per movie.
- Playback with selectable audio track and quality (`GET /api/v1/items/{id}/play?audio=&quality=&subtitle=`): the response lists the audio tracks, the available qualities (never above the original resolution) and the subtitles. Embedded subtitles and `.srt`/`.vtt`/`.ass` files next to the video, served as WebVTT. Audio, subtitle and quality preferences saved per user and per movie (`/api/v1/items/{id}/preferences`).

## [0.12.1] - 2026-10-26

### Fixed
- Cast photos and characters, automatic collections and series credits are now actually filled in: the TMDB cache stored only the fields the program knew when it was created, so fields added later (photo, character, collection) were missing. It now stores the full original response, old entries are discarded and the next scan rebuilds the records.
- Credits of already identified series are filled in even when their seasons are complete.

## [0.12.0] - 2026-10-26

### Fixed
- Video no longer freezes when seeking after several playbacks: FFmpeg stops after 90 s without requests, never runs more than 20 segments ahead of the requested point, there is a cap on simultaneous transcodes (`playback.max_transcodes` setting, 2 by default) and processes run at low priority with limited threads. Before, every abandoned playback kept transcoding the whole movie.

### Added
- Cast and crew with photo and character (`credits` on movies and series; saved records are filled from the TMDB cache) and "More like this" computed locally: `GET /api/v1/items/{id}/similar` and `GET /api/v1/series/{id}/similar`.
- Admin panel on the web with a side menu: Overview (versions, last scan, counts and items pending identification), Libraries, and settings forms per section (Scan, Metadata, Interface, Playback, System) with defaults, per-field validation and "Reset". Optional update check, only if enabled and the button is pressed.
- Settings apply live, without restarting: automatic scan interval and startup scan, TMDB language, identification threshold, transcoding quality (maximum height, CRF and preset) and minimum movies per automatic collection.
- Server settings stored in the database with a typed, validated registry (scan, metadata, interface, playback and system): `GET`/`PATCH /api/v1/settings`, `DELETE /api/v1/settings/{key}` and `GET /api/v1/settings/public`. Environment variables still provide the initial value.

## [0.11.0] - 2026-10-26

### Changed
- TMDB becomes the main source of records (in Spanish); OMDb stays as a fallback when there is no TMDB key.
- The movie listing moves from `/` to `/movies`; `/` is now the portal.

### Fixed
- "Back to automatic" re-parses the file name from the library root, not from the absolute path.

### Added
- Library scans run in the background: `POST /api/v1/libraries/scan` returns 202 (or 409 if one is already running) and `GET /api/v1/libraries/scan` reports the status and the last result.
- Automatic scan one minute after startup and then every `CINECAT_SCAN_INTERVAL` (default `6h`; `0` disables it).
- The libraries panel starts the scan in the background and shows "Escaneo en curso…" polling every 3 s; when done, it shows files, identified, unidentified and collapsible warnings.
- `GET /api/v1/system/version` (no session) with the version, commit and build date; the web footer shows the server version and warns if web and server versions differ. The web version comes from the root `VERSION` (the web image is now built from the repository root).
- Full movie record in the API (Spanish title, original title, tagline, genres, runtime, rating, directors, cast, backdrop, identification status) and, in `GET /api/v1/items`, the `kind`, `q`, `unmatched` filters and the `sort` ordering. Manual correction for admins: `GET /items/{id}/candidates`, `PUT` and `DELETE /items/{id}/match`.
- The scan fills in the Spanish title of records saved before it existed.
- Search by title, cast, director or everything (`field`), filter by initial letter (`letter`, ignoring articles and accents: "El Padrino" goes under P) and by genre (`genre`), genre list (`GET /api/v1/genres`) and ordering by normalised title.
- Per-user watched state: `PUT`/`DELETE /api/v1/items/{id}/watched`, `watched` field on items and a `watched` filter.
- Home portal at `/` with one entry per category (Movies; Collections and Series "coming soon") and a "Recently added" row. The movie listing moves to `/movies`, paginated (48 per page), with search by title, cast or director, filters by letter and genre, sorting, watched/unwatched and, for admins, "Unidentified". "Watched" mark on posters.
- `GET /api/v1/items/{id}/file-info`: technical file information via ffprobe (container, size, video, audio and subtitle tracks with codecs, HDR, quality) and how it will be played.
- Customisable images (admin): `GET /api/v1/items/{id}/images` offers TMDB posters and backdrops; `PUT`/`DELETE /api/v1/items/{id}/images/{poster|backdrop}` set a custom URL or restore the TMDB one. Scans do not overwrite pinned images.
- Full movie page (original title, tagline, runtime, genres, rating, directors, cast, IMDb and TMDB links, backdrop), watched/unwatched button and, for admins, "Fix record" with TMDB candidate search, "None of these" and "Back to automatic".
- Per-user playback progress (`PUT`/`DELETE /api/v1/items/{id}/progress`; at 90 % it is marked as watched) and `GET /api/v1/home` with "continue watching", recently added items and the portal counters.
- "File" panel on the movie page (quality, container, size, video, audio and subtitle tracks, how it will play) and, for admins, an "Image" panel to pick the poster and backdrop from TMDB or set a custom URL.
- The player saves progress every 15 s and on exit, and resumes where you left off ("Continue · 1 h 12 min" on the movie page, "Start from the beginning"). The portal shows "Continue watching" and "Recently added".
- Automatic collections: movies that belong to a TMDB saga are grouped into a collection automatically; a collection deleted or a movie removed by the user does not reappear in later scans. Already identified movies join their collection on the first scan.
- Collections API: paginated list and detail, and admin management (create, edit, delete, add and remove movies, reorder). Movies list the collections they belong to and can be filtered by `collection_id`.
- Collections on the web: active category in the portal and header, paginated list with search and sorting, a page per collection with its movies and, for admins, management (create, edit, change poster and backdrop, add and remove movies, reorder, delete). The movie page links to its collections.
- The name parser takes the series from the top-level folder and recognises episodes in formats such as `ChicagoFire720p4x17` or `MR_Robot_305` inside a season folder (they used to be taken for movies).
- Series: each series folder is grouped into its own record, identified with TMDB in Spanish (same rules as movies: automatic, manual or "none of these") and the scan fills in seasons and episodes (title, overview, still) with one request per season. Scan status includes `series_identified` and `series_unidentified`.
- Series API: paginated list with search, letter and genre filters, record with seasons and next episode, episodes per season, mark series or season as watched, identification correction and images for admins. `GET /api/v1/home` fills in "next episode".
- Series on the web: active category in the portal and header, paginated list with search, letter and genre filters, and a series page with season selector, episodes with title, overview and still, "Continue" and marking an episode, season or series as watched. The portal shows "Next episode".
- Fix a series record (admins): search TMDB candidates, "This one", "None of these" and "Back to automatic", and change its poster and backdrop.
- Episode page with breadcrumbs (series › season › episode), episode still and overview, and previous/next navigation.
- The server shuts down gracefully on SIGINT/SIGTERM, cancelling any running scan.

## [0.10.0] - 2026-10-26

### Fixed
- Movies with AC-3 5.1 audio (almost the whole real library) kept loading forever: audio was converted to 5.1 AAC, which browser players do not accept. It is now always downmixed to stereo AAC.
- Seeking to a far point no longer leaves the video "thinking": if the requested segment is far from what FFmpeg has produced (or is a gap behind it), the server stops FFmpeg and restarts it at that segment (`-ss` + `-start_number`, the same rule Jellyfin uses). A seek takes under a second on SD video, instead of waiting up to 5 minutes and failing.
- Playback no longer stops after 15 minutes: HLS segments and seeks in direct play reuse the signature of the initial URL, which expired after 15 minutes. The default URL lifetime is now 8 hours.
- The last playlist segment can no longer be a few-millisecond remainder that FFmpeg never produces (it failed at the end of the video).
- The web player uses hls.js whenever the browser supports it, instead of the native HLS that Chrome on Android and desktop advertise; it retries on network or decoding errors instead of giving up on the first one, and waits up to 60 s for the first byte of a segment.

### Changed
- HLS uses MPEG-TS segments (`segment_XXXXX.ts`) instead of fMP4 (`init.mp4` + `.m4s`): each segment carries absolute timestamps, so segments produced by different FFmpeg processes fit the same timeline. Output timestamps carry a fixed 10 s offset so the process starting at 0 has no negative DTS (which FFmpeg fixed by shifting a few frames).
- Everything served over HLS is re-encoded to H.264 (libx264 `veryfast`, CRF 20, up to 1080p with even dimensions) + stereo AAC. Remux without re-encoding is removed: the real library is almost all DivX/MPEG-4 with AC-3 or MP3, which browsers cannot play. FFmpeg no longer starts when the playlist is requested but on the first segment request, at the requested point.
- The API still reports `mode: "remux"` for HLS.

## [0.9.1] - 2026-10-26

### Fixed
- The served HLS playlist is now a complete VOD, not a live event: Cinecat obtains the duration with `ffprobe` and writes `playlist.m3u8` itself before starting FFmpeg, listing every expected segment with `EXT-X-PLAYLIST-TYPE:VOD` and `EXT-X-ENDLIST` from the first request. This lets hls.js know the total duration and seek normally. FFmpeg writes its own playlist separately (`ffmpeg.m3u8`) so it cannot overwrite it, and segments are still served as soon as they exist. If the duration cannot be read (some old AVI files), the fallback to FFmpeg's `event` playlist remains.

## [0.9.0] - 2026-10-26

### Fixed
- HLS generation no longer blocks until the whole video is processed: `Service.prepare()` starts FFmpeg in the background and returns as soon as the initial playlist exists, and segments are served as soon as FFmpeg writes them (with a bounded wait and a check that the segment is complete). So the playlist exists from the first segment, it now uses `-hls_playlist_type event` instead of `vod` (`vod` only wrote it at the end); `ENDLIST` is added on completion and it remains a normal VOD. The process is tied to the service lifecycle, not to the HTTP request that starts it, and two simultaneous requests for the same item reuse the same process. Previously a one-hour movie could take over 25 minutes to start.

## [0.8.1] - 2026-10-26

### Fixed
- 0.8.0 didn't fix every real case: a production `.avi` with 2804 well-spread keyframes (below the detection threshold) still produced a single HLS segment with the whole movie, because `-c:v copy` needs reliable packet timestamps to segment, and in AVI/DivX many packets carry `pts_time` as `N/A`. The keyframe interval can be fine and the container still not be segmentable. Now any `.avi` always transcodes, without going through the keyframe analysis, which is kept for other containers (`.mkv`, `.mov`, `.ts`, `.m2ts`).

## [0.8.0] - 2026-10-26

### Added
- Software video transcoding when the keyframe interval prevents a reasonable HLS remux: instead of failing, only the video is re-encoded with libx264 `veryfast`/CRF 20, AAC 160k audio and keyframes forced every 6 s, keeping the rest of the HLS flow. This fixes playback of `.avi`/DivX files with sparse keyframes such as the real production case; it does not implement the full device-profile engine or hardware acceleration yet, and the API still reports `mode: "remux"` (the client receives HLS either way).

## [0.7.3] - 2026-10-26

### Fixed
- The HLS remux now checks with `ffprobe`, before launching FFmpeg, that the maximum keyframe interval is viable (≤ 15 s). For videos with widely spaced keyframes (common in old `.avi`/DivX files) it returns a clear error instead of producing a single huge segment that the browser cannot start playing. **It does not fix playback of those files**: real transcoding is left for the next Phase 3 step.

## [0.7.2] - 2026-10-26

### Added
- `GET /api/v1/items` accepts an optional `library_id` to filter by library (needed on the web client side, since with 1652 real items in production the flat, unfiltered, unpaginated list was unmanageable).

## [0.7.1] - 2026-10-26

### Fixed
- `docker-compose.yml` wasn't passing `CINECAT_OMDB_API_KEY` to the server container (support was added in the code in 0.6.0, but forgotten in the compose file): even with the key in `.env`, the server never saw it and the scan kept running without metadata. Also added to `.env.deploy.example`.

## [0.7.0] - 2026-10-26

### Added
- The library shows the real OMDb poster when available and keeps the typographic fallback poster when it is missing. The item page also shows the real poster and adds the plot (`overview`) when available, omitting the section when there is no data.

## [0.6.0] - 2026-10-25

### Added
- OMDb metadata for movies during scanning (`CINECAT_OMDB_API_KEY`): plot and poster URL, with an on-disk search cache, avoiding repeat queries for items that already have data and for series/episodes. Exposed as `overview` and `poster_url` in `GET /api/v1/items` and `GET /api/v1/items/{id}`. OMDb misses or failures are reported as scan warnings.

## [0.5.0] - 2026-10-25

### Added
- `/admin/libraries` panel for admin-role users: lists libraries, creates a new one (name and path), deletes it with confirmation and runs the full scan showing libraries, files, updated entries and warnings. "Bibliotecas" link in the header only for administrators, with the role obtained from `GET /api/v1/auth/me`; users without the role get a clear 403.

## [0.4.1] - 2026-10-25

### Added
- `GET /api/v1/auth/me`: returns `{username, role}` for the current Bearer session user, with 401 if the token is missing or invalid.

## [0.4.0] - 2026-10-25

### Added
- "My account" screen in the web client to change the password: form with current, new and repeated password, local validation of matching values and minimum length (8 characters), clear API error messages and a success confirmation without closing the current session. "Mi cuenta" link in the header only when a session exists.

## [0.3.5] - 2026-10-25

### Added
- `POST /api/v1/auth/password`: lets an authenticated user change their own password by checking the current one (401 if it does not match, 400 if the new one is shorter than 8 characters), without invalidating the current session or any other live sessions.

## [0.3.4] - 2026-10-25

### Changed
- `docs/es/despliegue.md`/`docs/en/deployment.md` and `.env.deploy.example` explain how to register several libraries (e.g. movies and series) when `CINECAT_MEDIA_PATH` points to a disk or folder that contains all of them as subfolders, instead of assuming a single library at the mounted root.

## [0.3.3] - 2026-10-25

### Added
- `cinecatd --hash-password <password>` flag: prints the argon2id hash in PHC format, to manually set `password_hash` in the database (e.g. with `sqlite3` inside the container) while there's still no endpoint or screen to change the password from the interface.

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

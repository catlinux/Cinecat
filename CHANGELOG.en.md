# Changelog

All notable changes to this project are documented here.
Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versioned according to [Semantic Versioning](https://semver.org/) (`x.y.z`).

[Versión en español](CHANGELOG.md)

## [Unreleased]

### Fixed

- Android app: the portal, lists and detail pages did not load from a real server, because lists omit cast and credits and the app required them. All library fields are now optional and a `null` list is read as empty.
- Android app on TV: primary button text was dark on maroon and barely readable.
- Android app: large titles (app name and TV screen titles) did not use Fraunces.
- Android app: the server address and local IP fields drop the spaces some keyboards add after each dot.
- Android app on TV: the remote could not leave a text field with the up and down arrows; they now move to the previous or next field.

## [0.33.0] - 2026-10-29

### Added

- Library in the Android app for phones and tablets: portal with the same rows and order as the web client, bottom navigation (side rail on tablets), movies and series in a paged grid with sort and genre, collections, movie and series search, and movie, series (seasons and episodes with progress) and collection pages with favourite, "to watch" and watched. Posters are loaded from TMDB at screen size and cached. Playback is not available yet; the dedicated TV interface will come later.

## [0.32.0] - 2026-10-29

### Added

- The Android and Android TV app now connects to the server: a screen for the server address (HTTPS only, with version and compatibility check), sign-in with username and password on phones and pairing by code with a QR code on TV (also available on phones). The device session is stored encrypted with the Android Keystore and the app is excluded from Android backups. After signing in there is a provisional home screen; the library and playback will arrive in the next versions.
- "Server local IP" in the app: when set (private IPv4 or local IPv6) and the device is on Wi-Fi or Ethernet, the app reaches the server over the local network using the same domain and its certificate. It is the equivalent of editing `hosts` on the PC, which a TV does not allow. Details in [docs/en/android.md](docs/en/android.md#server-local-ip).

## [0.31.0] - 2026-10-29

### Added

- Skeleton of the app for Android, tablet and Android TV in `android/` (Kotlin + Jetpack Compose): Gradle project with its modules, "Marquesina" theme and a test home screen that differs for phone and TV. It does not connect to the server or have features yet; the app version comes from the `VERSION` file. Documentation in [docs/en/android.md](docs/en/android.md), dependencies and licences in `android/DEPENDENCIES.md` and `CREDITS.md`, and a GitHub Actions workflow that builds the app when `android/` changes.

## [0.30.0] - 2026-10-29

### Added

- Stable API contract for native clients: `GET /api/v1/system/version` now returns `api_level` (currently 1) and `min_client` (minimum supported app version). The OpenAPI now covers all 109 routes (users, trash, backups, uploads, people, delete item and `/healthz` were documented) and a new test (`internal/apicontract`) fails if a server route is missing from the OpenAPI or vice versa.
- API documentation for client developers: [docs/en/api.md](docs/en/api.md) (stability, authentication, pairing by code, playback with `caps`, errors and a `curl` example).

## [0.29.0] - 2026-10-28

### Added

- Native app capabilities in the playback decision: with the new `caps` tokens (containers `mkv`/`mp4`, video `h264`, `h264hi10`, `hevc`, `hevc10`, `av1`, `vp9`, audio `aac`, `mp3`, `ac3`, `eac3`, `dts`, `truehd`, `opus`, `flac` and `mch`), an MKV or MP4 is served directly if the device plays its video and the chosen audio track; otherwise HLS copying HEVC. `/play` returns `reason` (`container`, `video`, `audio`). The web does not send these tokens and its behaviour is unchanged (a test walks its whole matrix).

## [0.28.0] - 2026-10-28

### Added

- Configurable portal: **Administration → Settings → Interface → Portal rows** sets the order of the home page rows, which ones are shown and their title and kicker (`ui.portal_rows` setting). The category strip stays fixed at the top.
- New **"Vistas hace poco"** (recently watched) row with the latest movies marked as watched (`recently_watched` in `GET /api/v1/home`).

### Changed

- Default portal order: recently added movies, continue watching movies, movies to watch, recently watched, recently added series, continue watching series, next episode, series to watch and favourites. "Próximas a ver" is split into movies and series; some rows are renamed.

## [0.27.0] - 2026-10-28

### Added

- Pair a device by code: an Android or Android TV app requests a code (`POST /api/v1/auth/pair`), shows it as `XXXX-XXXX` with a QR code, and the user approves it on the web in the new **/vincular** page (also linked from Your account) after seeing the device name and an anti-scam warning; the device then collects its 90-day session (`POST /api/v1/auth/pair/token`). The code expires after 10 minutes and is single-use; the token is only handed to whoever holds the device secret, which is never shown; code and secret are stored only as hashes, with attempt and polling limits. Migration 0022.
- Login returns to the page you came from (`?next=`), only when it is a path on the same site.

## [0.26.0] - 2026-10-28

### Added

- Device sessions: each session stores the device name (the web derives it from the browser and system, e.g. "Chrome · Windows") and its type (`web`, `android` or `androidtv`). **Your account** lists "Open sessions" with the last activity and a button to close each one, and in **Administration → Users** each card shows and closes that user's sessions. Native client sessions will last 90 days (renewable); web sessions stay at 24 h. API: `GET/DELETE /api/v1/auth/sessions` and, for administrators, `GET/DELETE /api/v1/users/{id}/sessions`; login accepts `device_name` and `client`. Migration 0021. This is the foundation for the Android and Android TV apps.

## [0.25.0] - 2026-10-28

### Added

- Daily database backup: the server creates a consistent copy (`VACUUM INTO`) every 24 hours in `/data/backups`, next to the database, and keeps 7 days (adjustable from 1 to 365; the most recent one is always kept). Settings `backup.enabled` and `backup.retention_days`. Administration → Settings → System has a panel listing the copies with a "Back up now" button (API `GET/POST /api/v1/system/backups`, administrators only, no disk paths exposed). Copies live in the same volume as the database: they do not protect against losing the disk.

## [0.24.1] - 2026-10-28

### Fixed

- The portal's "Continue watching" no longer mixes movies and episodes in one row: there are now two, "Continue watching movies" and "Continue watching series".

## [0.24.0] - 2026-10-28

### Security

- Per-library access: each non-administrator user only sees the libraries assigned to them (or all of them, including future ones, with that option). It applies to every listing, search, the home page, series, collections, people, music, the user's lists and to each item, its playback, thumbnail, progress, watched state and preferences. Anything hidden answers as if it did not exist (404) and cannot be written to. Collections and people only count what is visible. An automated leak test walks every read route with a marked hidden library and fails if any response gives it away or if a new route appears without coverage. Known limitation: an already issued playback URL stays valid until it expires (8 h).
- User management on the server (administrators only): `GET/POST /api/v1/users`, `PATCH /api/v1/users/{id}` (role, disable, access to all libraries), `PUT /api/v1/users/{id}/libraries`, `POST /api/v1/users/{id}/password` and `DELETE /api/v1/users/{id}`. Resetting the password, disabling, deleting or changing the role closes all of that account's sessions; a disabled account cannot sign in or use an earlier session; nobody can delete, disable or demote themselves, and at least one active administrator always remains. Migration 0020 (`disabled`, `all_libraries`, `user_libraries`). The screen and per-library content filtering arrive in later versions.
- Changing the password closes that user's other open sessions (the current one is kept).
- JSON API requests are limited to 1 MiB (413 response when exceeded); file uploads do not have that limit.
- File and library folder paths (on detail pages, listings, libraries and the file technical details) are only sent to administrators; other users no longer see how the server disk is organised.
- Security headers on the web app and the API: the web app can no longer be embedded in another page (`X-Frame-Options: DENY` and `frame-ancestors 'none'`), `Strict-Transport-Security` over HTTPS, `X-Content-Type-Options: nosniff`, referrer and permissions policies. The CSP is only the minimal one; a full one is left for later. `ServerTokens Prod` is recommended for Apache (see deployment).
- The server is built with Go 1.26 (before 1.22.12, which no longer receives patches and carried 43 known standard library vulnerabilities: `net/http`, `crypto/tls`, `crypto/x509`, `mime/multipart`…).

### Added

- Administration → Users: the administrator creates accounts ("User" role by default), chooses which libraries each one sees (or "all, including future ones"), resets passwords, disables or enables accounts, changes the role and deletes with confirmation. Your own row does not let you disable or delete yourself. A user who is not an administrator only sees in the menu the sections they have a library for, and if they have none the home page explains it.
- Hardware acceleration (Intel VA-API) when re-encoding: when `docker-compose.gpu.yml` is enabled, the integrated graphics encodes the video to H.264 and, when no scaling is needed, also decodes it (8-bit H.264 and H.265), with much lower CPU use. If there is no graphics device, the "Hardware acceleration" setting is "Off" or something fails, it re-encodes with `libx264` as before. The file details show "Re-encode (graphics)". The image includes the `intel-media-va-driver` driver. It is optional: without `COMPOSE_FILE=docker-compose.yml:docker-compose.gpu.yml` in `.env` nothing changes.
- Copy mode in playback: when the browser supports the codec (8-bit H.264, or H.265 when the browser plays it through MSE), MKV and MP4 files are repackaged to HLS copying the video without re-encoding it; only the audio is converted to AAC. It uses almost no CPU and loses no quality. Segments are cut at the real keyframes, read from the MKV internal index in milliseconds or, for other files, with ffprobe in the background the first time (stored in `cache/keyframes`); meanwhile, and on any failure, it re-encodes as before. AVI and TS files are always re-encoded. Copies do not count towards "Simultaneous transcodes". The file details show the method: direct, copy or re-encode.
- The web app declares whether the browser plays H.265 through MSE (`hevcmse`, `hevc10mse` in the `cinecat_caps` cookie).

### Changed

- The example Apache vhost is now `deploy/media.techmarc.es.conf` and the documentation uses `media.techmarc.es` as the domain. When you change domain you have to sign in again (the cookie belongs to the domain); the documentation explains it.
- Listings (movies, series, home, lists, similar) no longer include each title's full cast, which cards do not use: a page of 48 movies goes from about 320 KB to a fraction of that and loads faster on mobile and slow connections. The detail page still shows the full cast.
- "Upload movie": "Create a folder for each movie" is unchecked by default.

### Fixed

- "Sign out" is now in the menu (and in My account); before it only existed inside the sign-in page and was hard to find. My account now explains what happens to sessions when you change the password: those on other devices are closed and the current one is kept.
- Playback in progress is no longer cut when the server is updated or restarted: the key used to sign playback URLs is stored next to the database (`signing.key`, permissions 0600) instead of being generated anew on every start.
- An H.265 MP4 tagged `hev1` (most downloaded ones) is no longer sent directly to a browser that does not accept it (Safari and iOS only accept `hvc1`): the web app also checks that tag and, if not, plays it by copy or re-encode.
- The collections list is much faster (the count uses `JOIN`).
- Long uploads no longer fail with `fetch failed`: the web app forwarded the file with a `fetch` that gives up after 300 s without a response, and the server only answers once it has received it. It is now streamed without a time limit and straight to the server over the Docker network (`INTERNAL_API_URL`, already in `docker-compose.yml`), without going out to the internet or through Apache again.
- An MP4 whose default audio track is AAC or MP3 with more than two channels (e.g. AAC 5.1) is no longer played directly but through HLS with the audio downmixed to stereo: Opera produced no sound from multichannel AAC in direct play.

## [0.22.1] - 2026-10-28

### Fixed

- The collection indicator on posters now shows the collection name (with "+N" when there are several) and its tooltip works: before, the icon blocked the mouse and it was not clear which collection it was.

## [0.22.0] - 2026-10-27

### Added

- Person page and trailers on the web: clicking an actor, director or crew member opens their page (`/people/{id}`) with their profile, what you have in the library and, separately, what they have done that you do not have. "Ver tráiler" button (YouTube, in Spanish when available, otherwise in English) on movie and series pages.
- People and trailers on the server: credits store each person's TMDB id, `GET /api/v1/people/{id}` returns their profile with what is in the library and what is not (separately), and movies and series carry `trailer_url` (YouTube, in Spanish when available, otherwise in English). The series list includes the rating. Migration 0019: the next scan asks TMDB again for the entries to fill this in (it will take longer than usual). The screen arrives in the next version.
- TMDB rating in a corner of movie and series posters (for example "7,4"), only when available.
- "Upload movie" in Movies (administrators only): uploads one or more movies, with their subtitles, to the movies folder without having to choose it; by default it creates one folder per movie.
- Guided series uploads on the web (administrators only): "Create new series" button in Series, "New season" on the series page and "Upload episode" on each season. The web app detects season and episode in the file name (`S01E05`, `1x05`, "Capítulo 5"…) and lets you correct them before uploading, with a progress bar per file.
- Guided uploads on the server: `GET /api/v1/uploads/targets` reports the movie and series folders of each library, and `PUT /api/v1/series/{id}/files?season=&episode=&name=` uploads an episode with the server choosing the folder (the season's folder if it exists, "Temporada N" if the series uses season folders) and the name (`<series> S01E05.ext`, keeping the subtitle language). The screen arrives in the next version.
- Collections: a discreet icon on the poster of movies that belong to a collection, a "Collection: In one / None" filter in Movies and, on the movie page, "Manage collections" (administrators only) to add the movie to an existing collection, create a new one with it or remove it; removing it from a TMDB collection excludes it so the scan does not add it back. `GET /api/v1/collections?all=1` (administrators) also lists hidden or empty collections and `GET /api/v1/items` accepts `collection=any|none`.

### Fixed

- Playback is decided by the real codec and what the browser supports, not by the extension: an H.265 MP4 (or 10-bit H.264, or with AC-3 audio) is no longer sent directly to a browser that cannot play it; it is converted. The web app detects once whether the browser supports H.265 and remembers it in the `cinecat_caps` cookie.
- The `.cinecat-papelera` folder is created with 0775 permissions, like the rest of the library, instead of 0755 from the container umask.

## [0.21.0] - 2026-10-27

### Added

- Upload files from the web: Administration → Libraries → "Upload files" on video and clip libraries (destination folder with the folder picker, optional new folder, several files with a progress bar and cancel). The web streams each file to the server.
- Trash on the web: "Move to trash" button on the item page (administrators only, with confirmation) and an Administration → Trash page to restore, delete permanently or empty expired entries. New Administration → Library section with "Allow file changes" and "Days in the trash".

- Video trash on the server (administrators only, with the "Allow file changes" setting): `DELETE /api/v1/items/{id}` moves the movie, episode or clip (with its subtitles, or its folder when it holds no other videos) to `.cinecat-papelera` on the same disk, without deleting anything; `GET /api/v1/trash`, `POST /api/v1/trash/{id}/restore` (brings back the file and the entry with watched state, progress, lists, collections and preferences), `DELETE /api/v1/trash/{id}` and `POST /api/v1/trash/purge`. Daily purge according to "Days in the trash" (14 by default). Migration 0018.
- File upload on the server: `PUT /api/v1/libraries/{id}/files?dir=…&name=…[&subdir=…]` streams a video or subtitle into a folder of a video library (never overwrites; writes a `.part` file and renames it when done), with a "Maximum upload size (GB)" limit (50 by default), a free-space check and a scan afterwards. `CINECAT_FILE_UID`/`CINECAT_FILE_GID` set the owner of what the server creates.

### Changed

- Deployment: the `/media` mount depends on `CINECAT_MEDIA_MODE` (`ro` by default, `rw` to upload and use the trash); new `CINECAT_FILE_UID`/`CINECAT_FILE_GID` variables, and the web container accepts large bodies (`BODY_SIZE_LIMIT=Infinity`). Documented under Administration → "Uploading files and trash".
- Administration → System: "Update now" now opens a confirmation dialog (with "Cancel" and "Update now") instead of a collapsible with a second button. Without JavaScript the two-step fallback is kept.

## [0.20.0] - 2026-10-27

### Added

- Favorites and "Watch next" on the web: heart and bookmark buttons on movie, video, series and collection pages and on cards (they also work without JavaScript), "Favorites" and "Watch next" pages with sections and pagination, menu entries (when signed in) and "Watch next" and "Your favorites" rows on the portal.
- Favorites and "Watch next" on the server: each user marks and unmarks movies, videos, series and collections (`PUT`/`DELETE /api/v1/{items|series|collections}/{id}/lists/{list}`) and lists them with `GET /api/v1/lists/{list}`. Migration 0017. DTOs carry `favorite` and `watchlist`, and the home carries `watchlist` and `favorites`. Marking something as watched removes it from "Watch next" automatically (and a series once it is fully watched). The screen arrives in the next version.

## [0.19.1] - 2026-10-27

### Security

- Failed-login lockout can no longer be used to lock someone else's account: attempts are counted per user + client IP pair, with a separate limit of 20 failures per IP. The web passes the real IP to the API (`X-Forwarded-For`, with `ADDRESS_HEADER` and `XFF_DEPTH=1` on the web container) and the API only trusts that header when the direct connection comes from a trusted proxy (`CINECAT_TRUSTED_PROXIES`, loopback and private networks by default). The lockout answers 429 with `Retry-After`, does not reveal whether the user exists and the in-memory map is capped.

## [0.19.0] - 2026-10-27

### Added

- "Music" section on the web: albums (with cover, sorting and letter filter), artists and track search, album and artist pages, a menu entry and a portal category. The "Music" type can be chosen when creating a library.
- Persistent music player: a fixed bottom bar that keeps playing while you browse, with a queue (add, play next, remove, clear), shuffle, repeat, volume, media keys and system metadata (Media Session). It pauses when a video plays and the queue is remembered between visits (restored paused).
- "Music" library type on the server: the scan reads tags with `ffprobe` (artist, album, track, disc, year, genre), with no external services, only re-inspects new or changed files, and keeps entries if a folder is unavailable. Migration 0016 (`music_artists`, `music_albums`, `music_tracks`). Covers from the folder or embedded (cached in `cache/covers`).
- Music API: `GET /api/v1/music/artists`, `/artists/{id}`, `/albums`, `/albums/{id}`, `/albums/{id}/cover`, `/tracks` (search) and `/tracks/{id}/play`, and the signed streaming endpoint with ranges `GET /api/v1/stream/music/{id}`. `GET /api/v1/home` adds `has_music`.

### Changed

- Signed playback URLs now carry a namespace (`video` or `music`) so a track id cannot validate as a movie or vice versa.

## [0.18.0] - 2026-10-27

### Added

- Video thumbnails taken from a frame of the file itself, generated with FFmpeg the first time they are viewed and cached (`cache/thumbs`); each folder uses one of its videos as its cover (`cover_item_id`). Without FFmpeg, or on failure, the card keeps its gradient. Endpoint `GET /api/v1/items/{id}/thumbnail`.
- "Videos" (`clips`) library type, meant for music videos and concerts: scanned without identification or outbound connections (the title comes from the file name), kept apart from movies, search, collections and counters, and browsed by folders with `GET /api/v1/libraries/{id}/browse`. Items expose `library_kind` and the home exposes `has_clips`.
- "Videos" section on the web: menu entry (when there are videos), folder browsing with breadcrumbs and pagination, cards with progress and watched mark, a minimal detail page (no cast or metadata correction) and a portal category. The "Videos" type can be chosen when creating a library.

### Fixed

- Playback, file information and thumbnails failed ("file not found") for files in the secondary folders of a multi-folder library, because only the main folder was checked.

## [0.17.0] - 2026-10-27

### Added

- Administration → Libraries: redesigned screen with one card per library, several folders per library (add, remove with confirmation), rename, scan and delete, and a server folder browser to pick them. The "Videos" and "Music" types show as "Coming soon".
- Libraries with several folders (for example on different disks) and a type (`video`, `clips`, `music`): `POST /api/v1/libraries` accepts `paths`, plus new `PATCH /api/v1/libraries/{id}`, `POST /api/v1/libraries/{id}/paths` and `DELETE /api/v1/libraries/{id}/paths/{pathId}`. Migration 0015 (`library_paths`, `libraries.kind`); the existing library is migrated with its folder. The `clips` and `music` types are accepted but not scanned yet.
- Server folder browser `GET /api/v1/fs` (administrators only), confined to the roots in `CINECAT_MEDIA_ROOTS` (`/media` by default).
- Portal: "Recently added series" row (`recent_series` in `GET /api/v1/home`) linking to the series list sorted by latest added. The movies row is now titled "Recently added movies".

### Changed

- The scan only prunes entries from folders it traversed successfully: if a folder is unavailable or shows up empty while having entries (unmounted disk), it keeps the entries with their watched state and progress and reports it.

## [0.16.3] - 2026-10-27

### Changed

- Documentation brought up to date in Spanish and English: rewritten README, new status page (what is done and what is missing), user guide and administration guide (with the full settings table), and architecture, playback, metadata, deployment, roadmap and TODO reviewed against the code.

## [0.16.2] - 2026-10-27

### Fixed

- The updater installer detects that the `systemd --user` manager started before the user joined the `docker` group (the service could not use Docker) and explains how to fix it; the updater gives that same message instead of a bare "No se pudo consultar el directorio de Docker".

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

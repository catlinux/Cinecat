# TODO

[Versión en español](TODO.md) · [Project status](docs/en/status.md)

Priorities: 🔴 blocking · 🟠 next phase · 🟢 later.

This file contains only work that is still open. Published work is in the [CHANGELOG](CHANGELOG.en.md); the full status is in [status](docs/en/status.md).

## 🔴 Blocking for multi-user use

- **Login lockout by username.** `internal/auth` counts failures per `username`; five failures lock that account for 15 minutes. Anyone who knows a username can lock it. Combine username+IP or add a per-IP limit.
- **Users cannot be created.** There is no endpoint or screen to create, list, change role, reset password or delete users. Only the initial `admin` and `--hash-password` exist.
- **No per-library permissions.** Any authenticated user sees every library, item, series and collection. The `libraries` table has no owner or ACL.

## 🟠 Next phase

- **Favorites and watchlist.** Tables and endpoints, heart/bookmark on cards and detail pages, dedicated pages and rows on the portal.
- **Collections from a movie page** (add, remove and create without leaving the page).
- **Identify collections on TMDB:** manual search, synopsis and "missing from your library".
- **Libraries with multiple folders** and a **folder browser** to choose paths from the web.
- **File upload and deletion with a trash bin:** retention, restore, write permissions and proxy size warnings.
- **Users and profile:** user CRUD, own profile, portal customisation, open sessions and history.
- **Sessions, devices, activity and backups** in administration.
- **User-uploaded subtitles:** needs a design first; today only embedded text and `.srt`/`.vtt`/`.ass` files next to the video are read.
- **Advanced playback:** copy video when it is already 8-bit H.264, Quick Sync/VAAPI, per-device profiles, HDR/10-bit, deinterlacing, image subtitles and unsupported audio.
- **Real HLS tests and CI:** tests are skipped without `ffmpeg`/`ffprobe`; no automated Linux job runs them.
- **Review the 7-day TMDB TTL** against the current terms before accepting the production cache.
- **Expose a `transcode` playback mode** in the API; re-encoded HLS still reports `remux`.

## 🟢 Later

- 🟢 **Purge thumbnails for items that no longer exist.**
- **Plugins**, still at the design stage.
- **DLNA/UPnP**, still at the design stage.
- **Library types** other than video (music, photos or others) when the model is addressed.
- **Android and Android TV clients.** No code or detailed plan yet; the web client is the only finished one.
- Photos and slideshows, live TV and internet content, intro/credits segments, offline downloads and cross-device sync.
- Other platforms (Apple, Samsung/LG TVs).

## Discarded and why

- **Caddy as a bundled proxy:** discarded in 0.3.1; the host Apache already manages vhosts and HTTPS.
- **Remux without re-encoding and fMP4 segments:** discarded; real containers and timestamps made reliable remux unviable. All HLS is re-encoded to H.264 + stereo AAC.
- **OMDb as the main source:** TMDB became the main source in 0.11.0; OMDb is a fallback when there is no TMDB key.
- **The container updating itself:** it has no access to Docker; a host systemd service runs the request.
- **FilmAffinity and scraping:** no official API; not used.
- **Segmenting AVI by copying the codec:** discarded because timestamps are unreliable; AVI files are always transcoded.
- **`ui.page_size`:** replaced by columns × rows in 0.15.0.

## Accepted risks

- An inactive Android app from 2015 called "CineCat" exists (F-Droid, Catalan film catalogue; no trademark). The Cinecat name is kept; minor confusion risk.
- **No CORS:** the server sends no `Access-Control-*` headers; the deployment relies on Apache serving the API and web app under the same origin.
- **Read-only media:** `/media` is mounted `:ro`; the application cannot upload, delete or move files.
- **Software only:** transcoding uses libx264; there is no hardware acceleration or H.264 copy.
- **Cache and quotas:** the TMDB TTL is a starting value and OMDb has a daily quota; TMDB requires attribution (the "About" screen is missing).
- **Manually corrected series:** scans do not touch them; missing seasons or episodes are not filled automatically.
- **Playback URLs:** valid for 8 hours, not bound to the user, and invalid after a server restart.
- **Automatic updater:** only works when the host service is installed; the repository cannot verify its state on a given server.

## Open documentation

- Code of conduct.
- Enable private vulnerability reporting on GitHub; `SECURITY.md` already offers it as a channel.
- Web client screenshots and replacement of the private mockup link; task D2 will produce them.

## Document maintenance

- Keep the Spanish and English versions of every document in sync.

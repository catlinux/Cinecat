# TODO

[Versión en español](TODO.md) · [Project status](docs/en/status.md)

Priorities: 🔴 blocking · 🟠 next phase · 🟢 later.

This file contains only work that is still open. Published work is in the [CHANGELOG](CHANGELOG.en.md); the full status is in [status](docs/en/status.md).

## 🟠 Next phase

- **User profile:** portal customisation and history (user management and open sessions already exist).
- **Copy backups outside the volume** (another disk or machine): today they live in `/data/backups`, next to the database.
- **User-uploaded subtitles:** needs a design first; today only embedded text and `.srt`/`.vtt`/`.ass` files next to the video are read.
- **Advanced playback:** per-device profiles, HDR/10-bit, deinterlacing, image subtitles and unsupported audio.
- **Real HLS tests and CI:** tests are skipped without `ffmpeg`/`ffprobe`; no automated Linux job runs them.
- **Review the 7-day TMDB TTL** against the current terms before accepting the production cache.
- **Expose a `transcode` playback mode** in the API; copied or re-encoded HLS still reports `remux`.
- **Android and Android TV app** (plan in `.agents/plans/apps-android/apps-android.PLAN.md`): test AVI over HLS on the TV; HLS with AAC 5.1 when the device supports it (today AC-3 is converted to stereo AAC).

## 🟢 Later

- 🟢 **Hide Movies and Collections from users without movies:** a user with only a series library still sees those tabs in the app (and on the web), with "No movie matches". There is no data leak, but it is confusing and it was decided to leave it for now; the server needs to say which sections the user has, because movie and series libraries are the same type.
- 🟢 **Share login lockout between several server instances** (today it is process memory).
- 🟢 **Notify the administrator after many failed attempts.**
- 🟢 **Purge thumbnails for items that no longer exist.**
- **Plugins**, still at the design stage.
- **DLNA/UPnP**, still at the design stage.
- **Library types** other than video (music, photos or others) when the model is addressed.
- **Android and Android TV clients.** No code yet; plan in `.agents/plans/apps-android/apps-android.PLAN.md` (first device sessions, pairing by code, native capabilities and a stable API contract on the server).
- Photos and slideshows, live TV and internet content, intro/credits segments, offline downloads and cross-device sync.
- Other platforms (Apple, Samsung/LG TVs).

## Discarded and why

- **Caddy as a bundled proxy:** discarded in 0.3.1; the host Apache already manages vhosts and HTTPS.
- **fMP4 segments for copy mode:** discarded; real containers and timestamps made reliable fMP4 unviable. Copying uses MPEG-TS with cuts at real keyframes; re-encoding still uses H.264.
- **OMDb as the main source:** TMDB became the main source in 0.11.0; OMDb is a fallback when there is no TMDB key.
- **The container updating itself:** it has no access to Docker; a host systemd service runs the request.
- **FilmAffinity and scraping:** no official API; not used.
- **Segmenting AVI by copying the codec:** discarded because timestamps are unreliable; AVI files are always transcoded.
- **`ui.page_size`:** replaced by columns × rows in 0.15.0.

## Accepted risks

- An inactive Android app from 2015 called "CineCat" exists (F-Droid, Catalan film catalogue; no trademark). The Cinecat name is kept; minor confusion risk.
- **No CORS:** the server sends no `Access-Control-*` headers; the deployment relies on Apache serving the API and web app under the same origin.
- **Writing to disks:** uploads and the trash require `CINECAT_MEDIA_MODE=rw`; music cannot be uploaded or moved to the trash.
- **Re-encoding:** uses VA-API when available and libx264 otherwise. Copying video does not re-encode and uses almost no CPU.
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

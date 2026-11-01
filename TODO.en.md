# TODO

[Versión en español](TODO.md) · [Project status](docs/en/status.md)

Priorities: 🔴 blocking · 🟠 next phase · 🟢 later.

This file contains only work that is still open. Published work is in the [CHANGELOG](CHANGELOG.en.md); the full status is in [status](docs/en/status.md).

## 🟠 Next phase

- **Advanced playback:** device profiles, image subtitles and unsupported audio. Test the 0.55.0 deinterlacing and SDR tone mapping with real interlaced and HDR files.
- **Test the 0.56.0 5.1 audio on the TV** with a real 5.1 file (5.1 AAC over HLS) and check it sounds right.
- **Verify the "Servidor y web" GitHub Actions workflow** (`.github/workflows/server-web.yml`, runs the Go tests with FFmpeg and the web tests): check its first run is green.

## 🟢 Later

- 🟢 **Copy backups outside the volume** (another disk or machine): today they live in `/data/backups`, next to the database. The user does not want it for now.
- **Plugins**, still at the design stage.
- **DLNA/UPnP**, still at the design stage.
- **Library types** other than video (music, photos or others) when the model is addressed.
- **Android and Android TV clients.** No code yet; plan in `.agents/plans/apps-android/apps-android.PLAN.md` (first device sessions, pairing by code, native capabilities and a stable API contract on the server).
- Photos and slideshows, live TV and internet content, intro/credits segments, offline downloads and cross-device sync.
- Other platforms (Apple, Samsung/LG TVs).

## Discarded and why

- **`transcode` mode in the API:** `mode` in `GET /api/v1/items/{id}/play` stays `remux` for copied or re-encoded HLS. Changing it would break already installed apps, which compare against `remux`; `reason` already says why it is not direct.
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

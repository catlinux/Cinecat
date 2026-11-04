# TODO

[Versión en español](TODO.md) · [Project status](docs/en/status.md)

Priorities: 🔴 blocking · 🟠 next phase · 🟢 later.

This file contains only work that is still open. Published work is in the [CHANGELOG](CHANGELOG.en.md); the full status is in [status](docs/en/status.md).

## 🟠 Next phase

- **Test the 0.59.0 device profiles on a device** (Account → "This device" on the TV and phone; the profile applies when playing a film with no saved choice).
- **Advanced playback:** DTS and TrueHD are still converted to AAC in HLS (AC-3/E-AC-3 is copied since 0.60.0 for apps that declare it; still to be tested on an AC-3 device). The 0.58.0 image subtitles were tested on 2026-11-02 on the TV with a test VobSub (choosing one reloads over HLS with the subtitle burned in); still to see with a real PGS and on the phone. The 0.55.0 deinterlacing and SDR tone mapping are already tested with real FFmpeg in `hls_real_test.go`; still to see on an HDR/interlaced file from the library.
- **Hear the 0.56.0 5.1 audio on a 5.1 system:** tested on 2026-11-02 on the TV (Xiaomi MiTV-MSSP3) with a test AC-3 5.1 file: the HLS arrives as 6-channel AAC, ExoPlayer opens a 6-channel track (0x3f) and the TV downmixes it to stereo on its speakers. Still to hear on a real 5.1 system.
- **Test the 0.62.0 skip intro and credits on a device** (an episode with «Intro»/«Credits» chapters or manual marks, on the web, the phone and the TV).
- **Test the 0.63.0 and 0.65.0 offline downloads on the phone (MKV included)** (download, watch in airplane mode, delete; check that progress reaches the server when the connection returns).
- **Test the 0.64.0 per-series autoplay on a device** (switch on the series page, countdown when an episode ends, «Next episode» in the credits, on the web, the phone and the TV).
- **Pending improvement:** autoplay switch inside the apps' player (needs `/play` to return the series id).
- **Verify the "Servidor y web" GitHub Actions workflow** (`.github/workflows/server-web.yml`, runs the Go tests with FFmpeg and the web tests): check its first run is green.

## 🟢 Later

- 🟢 **Copy backups outside the volume** (another disk or machine): today they live in `/data/backups`, next to the database. The user does not want it for now.
- **Plugins (parked by the user on 2026-11-03):** there is a proposal in [plugins](docs/en/plugins.md); it will be picked up later, with nothing implemented until then.
- **Photo libraries (parked by the user on 2026-11-03):** implemented since 0.61.0 but unused and not tested on a device; not extended until the topic is picked up again.
- **DLNA/UPnP**, still at the design stage.
- **Android and Android TV clients.** No code yet; plan in `.agents/plans/apps-android/apps-android.PLAN.md` (first device sessions, pairing by code, native capabilities and a stable API contract on the server).
- Live TV and internet content.
- Other platforms (Apple, Samsung/LG TVs).

## Discarded and why

- **Cross-device sync** (sending playback to another device or watching together): not needed for now (decided on 2026-11-03); progress and watched state are already shared through the server.
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
- **Cache and quotas:** the TMDB TTL is a starting value and OMDb has a daily quota; TMDB requires attribution (shown since 0.57.0 in the web footer and on the apps' account screen).
- **Manually corrected series:** scans do not touch them; missing seasons or episodes are not filled automatically.
- **Playback URLs:** valid for 8 hours, not bound to the user, and invalid after a server restart.
- **Automatic updater:** only works when the host service is installed; the repository cannot verify its state on a given server.

## Open documentation

- Enable private vulnerability reporting on GitHub; `SECURITY.md` already offers it as a channel.

## Document maintenance

- Keep the Spanish and English versions of every document in sync.

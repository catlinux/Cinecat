# Status: what exists and what is missing

[Español](../es/estado.md) · [Back to README](../../README.en.md)

Current version: **0.65.1**. This page describes the real state of the code, not the original plan. Versions come from the [CHANGELOG](../../CHANGELOG.en.md).

Legend: ✅ done · 🟨 awaiting testing · ⬜ pending.

## What already exists

| Area | Feature | Status | Notes |
|---|---|---|---|
| Server and library | Go server, SQLite, migrations and versioned REST API | ✅ 0.1.0 | `cinecatd` binary; WAL and foreign keys. |
| Server and library | Local libraries, create/delete | ✅ 0.1.2 | Since 0.17.0 they support several folders and a type; deleting a library removes its items. |
| Server and library | Manual and automatic background scans | ✅ 0.11.0 | Scan returns 202/409 and reports progress; scheduled by `scan.interval` and at startup. |
| Server and library | Technical file details through `ffprobe` | ✅ 0.11.0 | Container, size, tracks, subtitles and playback mode. |
| Server and library | Multiple folders per library and folder browser | ✅ 0.17.0 | video, clips, music and photos types |
| Server and library | Vídeos library (unidentified), folder browsing | ✅ 0.18.0 | with frame thumbnails |
| Server and library | Music library: scan, API, screen and player | ✅ 0.19.0 | persistent player with queue |
| Server and library | Photos library: scan, API, thumbnails and web/Android viewers | 🟨 0.61.0 | JPEG, PNG, WebP and GIF; EXIF date on JPEG; 400 px JPEG thumbnails in `cache/photos`; folders, sorting and viewer on the web app and apps. Migration 0028. Covered by tests; not yet tried on a real device. |
| Music | Playlists | ⬜ | upcoming music step |
| Music | Music favourites | ⬜ | will reuse the favourites system |
| Music | Unsupported-format conversion | ⬜ | FFmpeg transcoding |
| Music | Gapless playback | ⬜ | seamless track changes |
| Server and library | Upload or delete files with a trash bin | ✅ 0.21.0 | Video only; disabled by default (`rw` mount and setting). |
| Identification and metadata | TMDB as the main Spanish-language source, with cache | ✅ 0.11.0 | OMDb is a fallback when TMDB is absent; OMDb only fills movie synopsis and poster. |
| Identification and metadata | Periodic refresh of TMDB entries | ✅ 0.50.0 | Daily maintenance fetches entries older than 90 days again, up to 40 movies and 20 series per day; manually corrected entries are left alone. Migration 0023 (`metadata_refreshed_at`). |
| Identification and metadata | Automatic identification, manual correction and "not this one" | ✅ 0.11.0 | `match_status`: `auto`, `manual`, `none` or empty; configurable threshold. |
| Identification and metadata | Cast and crew with photo and character | ✅ 0.12.0 | Movies and series; old records are filled from the TMDB cache. |
| Identification and metadata | Age certifications, TheTVDB, Fanart.tv or TVmaze | ⬜ | Not integrated. |
| Detail pages | Complete movie page with search, filters and links | ✅ 0.11.0 | Original title, synopsis, genres, runtime, rating, cast, backdrop and collections. |
| Detail pages | "More like this" | ✅ 0.12.0 / 0.15.0 | Computed locally; placed at the end of the page. |
| Detail pages | Correct metadata and change poster/backdrop | ✅ 0.11.0 | Administrators only; manual images survive scans. |
| Series | Series, seasons and episodes through TMDB | ✅ 0.11.0 | One series per top-level folder; title, synopsis, images and cast. |
| Series | Season selector and season posters | ✅ 0.15.0 | Falls back to the series poster. |
| Series | Next episode, continue and watched state per episode, season or series | ✅ 0.11.0 | Portal shows "Next episode". |
| Series | Correct series metadata or images | ✅ 0.11.0 | Administrators only. |
| Collections | Automatic collections from TMDB sagas | ✅ 0.11.0 | Automatic collection shown only when it reaches `ui.collection_min_items`. |
| Collections | Create, edit, delete, order and choose poster/backdrop | ✅ 0.11.0 | Manual collections and user edits preserved on automatic collections. |
| Collections | Manage collections from a movie page | ✅ 0.22.0 | Add, create and remove; poster icon and "No collection" filter. |
| Collections | TMDB identification and missing films | ✅ 0.41.0 / 0.42.0 | Search, link or unlink manual collections; the page shows the incomplete saga. |
| Web interface | TMDB rating on posters | ✅ 0.22.0 | Movies and series. |
| Web interface | Person page and "Ver tráiler" button | ✅ 0.22.0 | What you have and what you do not; YouTube trailer in Spanish when available. |
| Server and library | Guided uploads of series, seasons, episodes and movies | ✅ 0.22.0 | Administrators only; same requirements as the generic upload. |
| Playback | Decision by real codec and browser capabilities | ✅ 0.22.0 | H.265 direct only when the browser supports it; copying also depends on the keyframe index and the MSE tokens. |
| Playback | Direct play for MP4/M4V | ✅ 0.1.0 | Serves the file with HTTP range support. |
| Playback | HLS for MKV, AVI, MOV, TS and M2TS | ✅ 0.12.0 / 0.24.0 | Copies the video when the container, codec, height and keyframe index allow it; otherwise re-encodes to H.264 + stereo AAC, or 5.1 for TV apps with `mch`. MPEG-TS segments, seeking and process limits. |
| Playback | Copy mode without re-encoding | ✅ 0.24.0 | Cuts at real keyframes; H.265 through MSE. |
| Playback | Intel hardware acceleration (VA-API) | ✅ 0.24.0 | H.264 VA-API with CPU or GPU decoding; automatic fallback to libx264. |
| Playback | Deinterlacing and HDR to SDR while re-encoding | 🟨 0.55.0 | Re-encoding only; uses `yadif` and `zscale`/`tonemap` when FFmpeg has them. Covered by tests, not yet tried on real files. |
| Playback | 5.1 AAC over HLS for TV | 🟨 0.56.0 | Apps with `mch` and a track of 6 or more channels receive 5.1 AAC; browser and phone stay in stereo. Not yet tried on a real TV. |
| Playback | Audio, text subtitles and quality | ✅ 0.13.0 | Embedded text and `.srt`/`.vtt`/`.ass` as WebVTT; per-user, per-movie preferences. |
| Playback | Embedded image subtitles (PGS, VobSub, DVB) | 🟨 0.58.0 | They can be selected in the web app and the apps with the " · imagen" mark; the server burns them into the video when re-encoding for HLS, never in copy mode or with VA-API. Covered by the server's real FFmpeg test and unit tests for web and Android; not yet tried on a real device. |
| Playback | Per-device profiles (session) | 🟨 0.59.0 | Maximum quality and preferred audio and subtitle languages per session; edited in the web app and the apps and applied with explicit > item > profile > default precedence. They never pick an image track, and the quality cap only affects HLS. Covered by tests; not yet tried on a real device. |
| Playback | Skip intro and credits from chapters or manual marks | 🟨 0.62.0 | Buttons in the web app, Android and Android TV; administrators edit manual segments and apply them to the season. Detection by chapter titles, with no audio fingerprinting. Migration 0029. Covered by tests; not yet tried on a real device. |
| Playback | Custom player with controls, shortcuts and next episode | ✅ 0.13.0 | hls.js for HLS and native `<video>` for direct play. |
| Playback | 10-bit and ABR | ⬜ | Copy mode exists; re-encoding uses VA-API or software and H.264 + stereo or 5.1 AAC. |
| Web interface | Login, httpOnly session, logout and password change | ✅ 0.2.0 / 0.3.5 | Logout is on `/login`. |
| Web interface | Portal, movies, series and collections with search, filters, sorting and pagination | ✅ 0.11.0 / 0.15.0 | Columns × rows per page; letter, genre and watched filters on movies. |
| Web interface | Light/dark/system themes, side or top navigation and responsive layout | ✅ 0.2.0 / 0.14.0 | Theme preference stored in the browser. |
| Web interface | Favorites and watchlist | ✅ 0.20.0 | Cards, pages and portal rows. |
| Web interface | Playback history | ✅ 0.52.0 | `/history` page in the menu; shows the latest played or watched items and allows removing entries. Accessible content only. |
| Web interface | Custom home portal rows | ✅ 0.53.0 | Under My account → "Portal de inicio" each user hides their own rows; other accounts are unaffected. |
| Web interface | Global search | ⬜ | Not implemented. |
| Web interface | Navigation based on content | ✅ 0.47.0 | Hides Movies, Series and Collections when empty; if the request fails, it shows all of them. |
| Administration | Summary, libraries, scan, settings, correction and images | ✅ 0.11.0 / 0.12.0 | Settings apply without a restart. |
| Administration | Daily database backup | ✅ 0.25.0 | `VACUUM INTO` in `/data/backups` with configurable retention; list and "back up now" in Settings → System. |
| Administration | User management and library access | ✅ 0.24.0 | Create, role, disable, reset password and delete; each account sees its libraries. |
| Administration | Device sessions (view and close) | ✅ 0.26.0 | In Your account and per user; native apps use 90-day sessions. |
| Administration | Activity | ✅ 0.44.0 | Latest playbacks from all users, with a filter; `GET /api/v1/activity`. Since 0.52.0 it honours per-library access. |
| Administration | Security card and persistent lockouts | ✅ 0.51.0 | The overview shows lockouts from the last 7 days and the latest 5; attempts and lockouts are stored in SQLite (migration 0024). `GET /api/v1/system/security-events`. |
| Administration | Uploaded external subtitles | ✅ 0.54.0 | Administrators only; `.srt`, `.vtt` or `.ass` up to 2 MB in `subtitles/` next to the database, with `u<N>` ids. `POST`/`DELETE /api/v1/items/{id}/subtitles`. |
| Updates | Check tags and release notes from the server | ✅ 0.14.0 | GitHub `vX.Y.Z` tags; `system.check_updates` is opt-in. |
| Updates | Check and update from Administration → System | ✅ 0.16.0 | Phase tracking with restart, success, failure or rollback. |
| Updates | Host service with database backup and rollback | ✅ 0.16.0 | Bash + user systemd; the container does not run it. |
| Security | Argon2id, 24 h sessions and admin/user roles | ✅ 0.1.0 | Only the token hash is stored in the database. |
| Security | Per-library and per-user permissions | ✅ 0.24.0 | Each account only sees its libraries; hidden content returns 404. |
| Quality | "Servidor y web" GitHub Actions workflow | 🟨 | `.github/workflows/server-web.yml` runs the Go tests with FFmpeg and the web tests; its first green run still needs checking. |
| Android / Android TV clients | Android and Android TV | ✅ 0.59.0 | One app for phones, tablets and TV: session or pairing, library, Media3 player with preferences, A–Z index on TV, picture-in-picture on phones, automatic updates and update notices. Since 0.49.0 it offers "From the start" and "Continue", since 0.49.1 it names untitled tracks, since 0.56.0 the TV declares `mch`, since 0.58.0 it shows and selects image subtitles and since 0.59.0 it edits the "Este dispositivo" profile. Hides empty sections. See [Android](android.md). |
| Android / Android TV clients | Offline downloads on phones and tablets | 🟨 0.65.0 | Button on the movie page and every episode if the device plays the original's video and audio (MKV included), with the reason if not; DownloadManager into the private folder, Descargas screen, local playback with embedded subtitles and pending progress sync. TV does not show them. Covered by tests; not yet tried on a real device. |
| Playback | Automatic next-episode playback per series | 🟨 0.64.0 | Per-user, per-series setting (no row means enabled); cancellable 8-second countdown in the web app, Android and Android TV, manual credits button and `next_episode` in `/play`. Migration 0030. Covered by tests; not yet tried on a real device. |

## What comes next, in order

The order comes from the plans in `.agents/plans` and the open tasks; no dates or committed versions.

1. **Advanced playback**: 10-bit and DTS/TrueHD without re-encoding in HLS. Test 0.55.0 deinterlacing and HDR with real files, and 0.56.0 5.1 AAC on a TV.
2. **Verify the "Servidor y web" GitHub Actions workflow** and its first green run.
3. **Plugins**: design proposal in [plugins](plugins.md), awaiting approval.
4. **DLNA/UPnP**, still at the design stage.

## Known limitations and risks

- **Login lockout:** failures are counted per user+IP pair, with an additional per-IP limit; it depends on the proxy forwarding `X-Forwarded-For` correctly and on `CINECAT_TRUSTED_PROXIES`. Attempts and lockouts persist in SQLite (migration 0024).
- **No CORS:** the server sends no `Access-Control-*` headers. The intended deployment works because Apache serves `/api/` and the web app under the same origin.
- **Manually corrected series:** scans do not touch `manual` or `none` series; missing seasons or episodes are not filled automatically.
- **HLS:** re-encoding uses VA-API when `/dev/dri` is mounted and the check succeeds; otherwise it uses libx264 (software). Since 0.55.0 it deinterlaces and converts HDR to SDR when FFmpeg has the filters, but this has not yet been tried on real files. ABR is still missing. Image subtitles are burned while re-encoding on the CPU, never in copy mode or with VA-API. Copying video does not re-encode and uses almost no CPU.
- **The API still calls HLS `remux`** even though it copies or re-encodes; the client receives HLS in both cases.
- **If duration cannot be read**, FFmpeg's `event` playlist is used and long seeks are not fast.
- **Subtitles:** text subtitles are converted to WebVTT; embedded image subtitles (PGS, VobSub, DVB) can be selected and are burned into the video when re-encoding for HLS, never in copy mode or with VA-API. Administrators can upload `.srt`, `.vtt` or `.ass` files up to 2 MB (stored in `subtitles/`, next to the database) and they are served as WebVTT with `u<N>` ids. HLS has no separate subtitle rendition.
- **Intro and credits segments:** they are detected only from chapter titles or manual marks; there is no audio-fingerprint detection. An intro must last less than 10 minutes, and manual segments replace the chapter segment of the same type.
- **Real HLS tests:** skipped without `ffmpeg`/`ffprobe` in PATH; the "Servidor y web" GitHub Actions workflow installs FFmpeg and runs the Go and web tests.
- **TMDB:** daily maintenance fetches entries older than 90 days again (up to 40 movies and 20 series per day) so none exceeds the 6 months its terms allow; manually corrected entries are left alone. The 7-day response cache and quotas should be reviewed before heavy use. OMDb is only a fallback and has a daily quota.
- **Changes on disk:** uploading and moving to the trash require mounting `/media` read-write; music cannot be uploaded or moved.
- **Playback URLs:** valid for 8 hours and not bound to the user. The HMAC key is stored next to the database, so they survive server restarts.
- **Automatic updater:** only works when the host service has been installed; the repository cannot verify whether it is active on a given server.

## How it is updated

Manual: `./scripts/actualizar.sh`. From the web: Administration → System → "Check for updates" and "Update now". Service installation, backups, status and manual recovery are in [updates](updates.md).

[Español](../es/estado.md) · [Back to README](../../README.en.md)

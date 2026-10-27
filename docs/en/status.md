# Status: what exists and what is missing

[Español](../es/estado.md) · [Back to README](../../README.en.md)

Current version: **0.19.0**. This page describes the real state of the code, not the original plan. Versions come from the [CHANGELOG](../../CHANGELOG.en.md).

## What already exists

| Area | Feature | Status | Notes |
|---|---|---|---|
| Server and library | Go server, SQLite, migrations and versioned REST API | ✅ 0.1.0 | `cinecatd` binary; WAL and foreign keys. |
| Server and library | Local libraries, create/delete | ✅ 0.1.2 | One path per library; deleting a library removes its items. |
| Server and library | Manual and automatic background scans | ✅ 0.11.0 | Scan returns 202/409 and reports progress; scheduled by `scan.interval` and at startup. |
| Server and library | Technical file details through `ffprobe` | ✅ 0.11.0 | Container, size, tracks, subtitles and playback mode. |
| Server and library | Multiple folders per library and folder browser | ✅ 0.17.0 | video, clips and music types |
| Server and library | Vídeos library (unidentified), folder browsing | ✅ 0.18.0 | with frame thumbnails |
| Server and library | Music library: scan, API, screen and player | ✅ 0.19.0 | persistent player with queue |
| Music | Playlists | ⬜ | upcoming music step |
| Music | Music favourites | ⬜ | will reuse the favourites system |
| Music | Unsupported-format conversion | ⬜ | FFmpeg transcoding |
| Music | Gapless playback | ⬜ | seamless track changes |
| Server and library | Upload or delete files with a trash bin | ⬜ | `/media` is mounted read-only. |
| Identification and metadata | TMDB as the main Spanish-language source, with cache | ✅ 0.11.0 | OMDb is a fallback when TMDB is absent; OMDb only fills movie synopsis and poster. |
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
| Collections | Manage collections from a movie page | ⬜ | Managed from the collection page today. |
| Playback | Direct play for MP4/M4V | ✅ 0.1.0 | Serves the file with HTTP range support. |
| Playback | HLS for MKV, AVI, MOV, TS and M2TS | ✅ 0.12.0 | Re-encoded to H.264 + stereo AAC, 6 s MPEG-TS segments, seeking and process limits. |
| Playback | Audio, text subtitles and quality | ✅ 0.13.0 | Embedded text and `.srt`/`.vtt`/`.ass` as WebVTT; per-user, per-movie preferences. |
| Playback | Custom player with controls, shortcuts and next episode | ✅ 0.13.0 | hls.js for HLS and native `<video>` for direct play. |
| Playback | Per-device profiles, H.265, hardware, HDR, image subtitles and ABR | ⬜ | Current HLS is always software H.264 + stereo AAC. |
| Web interface | Login, httpOnly session, logout and password change | ✅ 0.2.0 / 0.3.5 | Logout is on `/login`. |
| Web interface | Portal, movies, series and collections with search, filters, sorting and pagination | ✅ 0.11.0 / 0.15.0 | Columns × rows per page; letter, genre and watched filters on movies. |
| Web interface | Light/dark/system themes, side or top navigation and responsive layout | ✅ 0.2.0 / 0.14.0 | Theme preference stored in the browser. |
| Web interface | Favorites, watchlist, global search and profiles | ⬜ | Not implemented. |
| Administration | Summary, libraries, scan, settings, correction and images | ✅ 0.11.0 / 0.12.0 | Settings apply without a restart. |
| Administration | Users, sessions, devices, activity and backups | ⬜ | "Users" appears as "Soon"; there is no user CRUD. |
| Updates | Check tags and release notes from the server | ✅ 0.14.0 | GitHub `vX.Y.Z` tags; `system.check_updates` is opt-in. |
| Updates | Check and update from Administration → System | ✅ 0.16.0 | Phase tracking with restart, success, failure or rollback. |
| Updates | Host service with database backup and rollback | ✅ 0.16.0 | Bash + user systemd; the container does not run it. |
| Security | Argon2id, 24 h sessions and admin/user roles | ✅ 0.1.0 | Only the token hash is stored in the database. |
| Security | Per-library and per-user permissions | ⬜ | Any authenticated user sees the whole library. |
| Android / Android TV clients | Android and Android TV | ⬜ | No code or detailed plan yet. The web client is the only finished one. |

## What comes next, in order

The order comes from the plans in `.agents/plans`; no dates or committed versions.

1. **Favorites and watchlist:** API, cards and pages.
2. **Collections from a movie page** and **manual TMDB collection search/identification**, with "missing from your library".
3. **File upload and deletion with a trash bin**, with retention and restore.
4. **Users** (create, role, reset password, delete) and a **profile** with portal customisation.
5. **Sessions/devices, activity and backups** from administration.
6. **User-uploaded subtitles** and a design for the subtitle types not supported today.
7. **Plugins**, still at the design stage.
8. **DLNA/UPnP**, still at the design stage.
9. **Library types** (for example music or photos) when the model is addressed.
10. **Android and Android TV clients**.

## Known limitations and risks

- **Login lockout by username:** five failures lock that account for 15 minutes; the counter is in memory and there is no per-IP limit. Someone can temporarily lock another known account.
- **No CORS:** the server sends no `Access-Control-*` headers. The intended deployment works because Apache serves `/api/` and the web app under the same origin.
- **No per-library permissions:** any authenticated user can see every library, item, series and collection. This is a risk for multi-user use.
- **No user creation:** only the initial `admin` and the `--hash-password` flag exist; there is no user screen or endpoint.
- **Manually corrected series:** scans do not touch `manual` or `none` series; missing seasons or episodes are not filled automatically.
- **HLS is always software:** no H.264 copy, Quick Sync/VAAPI, HDR tone mapping, deinterlacing, image subtitles or ABR. 1080p/HEVC may be tight on the CPU.
- **The API still calls HLS `remux`** even though it re-encodes; the client receives HLS in both cases.
- **If duration cannot be read**, FFmpeg's `event` playlist is used and long seeks are not fast.
- **Subtitles:** text only; PGS/VobSub are ignored. HLS has no separate subtitle rendition; they are served as WebVTT.
- **Real HLS tests:** skipped without `ffmpeg`/`ffprobe` in PATH, and there is no CI to run them.
- **TMDB TTL:** the 7-day cache is a starting value not checked against current terms. OMDb is only a fallback and has a daily quota.
- **Read-only media:** files cannot be uploaded, moved or deleted from Cinecat.
- **Password change:** it does not invalidate other open sessions.
- **Playback URLs:** valid for 8 hours, not bound to the user, and invalid after a server restart because the HMAC key is per process.
- **Automatic updater:** only works when the host service has been installed; the repository cannot verify whether it is active on a given server.

## How it is updated

Manual: `./scripts/actualizar.sh`. From the web: Administration → System → "Check for updates" and "Update now". Service installation, backups, status and manual recovery are in [updates](updates.md).

[Español](../es/estado.md) · [Back to README](../../README.en.md)

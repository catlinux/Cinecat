# Playback and codecs

[Español](../es/reproduccion.md) · [Back to README](../../README.en.md) · [Status](status.md)

Playback is implemented for the web with two paths: **direct** for MP4/M4V and **HLS** for everything else. Per-device capability decisions, H.265 and hardware acceleration are still future design.

## Current decision

| Situation | Mode |
|---|---|
| `.mp4` or `.m4v` | Direct play: the file is served with HTTP range support. |
| `.mkv`, `.avi`, `.mov`, `.ts`, `.m2ts` | HLS: re-encoded to H.264 + stereo AAC and served in segments. |
| Any other extension | 415; the container is not supported. |

The decision is based on the extension. The file's real codecs and the browser's capabilities are not checked before choosing. The API still calls the HLS mode `remux` even though it re-encodes.

## HLS: how it works

- Video: `libx264` with configurable preset and CRF, `yuv420p`, scaled up to the configured maximum height.
- Audio: one track, stereo AAC at 160 kbps.
- Forced keyframes every 6 s and 6 s MPEG-TS segments.
- A fixed 10 s `-output_ts_offset` so segments from different processes fit together.
- The playlist is a complete VOD when `ffprobe` knows the duration. Otherwise FFmpeg's `event` playlist is used and long seeks are not fast.
- FFmpeg does not start when the playlist is requested: it starts on the first segment request, at the requested point.
- If a segment more than 24 s ahead of what has been generated is requested, or a gap behind it, the server restarts FFmpeg at that segment (`-ss` + `-start_number`).
- A segment is complete when it appears in its process playlist; on restart, the half-written one and those about to be rewritten are discarded.

## Process limits

| Limit | Value |
|---|---|
| Initial playlist wait | 20 s |
| Segment wait | 2 min |
| Maximum process lifetime | 4 h |
| Idle stop | 90 s without requests |
| Segments generated ahead | 20 (~2 min) |
| Simultaneous transcodes | `playback.max_transcodes`, 2 by default |
| HLS job cleanup | 1 h unused, checked every 15 min |

When the transcode limit is reached, the job with the longest idle time is stopped. Processes run at low priority with a limited number of threads.

## Audio, subtitles and quality

- **Audio:** chosen by position in the `ffprobe` track list. The user's and movie's preference is stored in `item_preferences`.
- **Subtitles:** embedded text tracks and external `.srt`, `.vtt` or `.ass` files next to the video, with language and `forced` suffixes. They are converted to WebVTT. Image tracks (PGS/VobSub) are not supported and HLS has no separate subtitle rendition.
- **Quality:** HLS offers `auto`, 480p, 720p, 1080p (never above the original height or `playback.max_height`) and `original`. `original` does not copy the original file: it limits re-encoding to the original height. Direct play has only `original`.
- **Preferences:** audio, subtitle and quality per user and movie; the player saves them and reloads at the current second when changed.

## Web player

- Native `<video>` for direct play; hls.js 1.7.3 for HLS.
- Custom controls: play/pause, ±10 s, progress bar, volume, mute, speed (0.5×–2×), audio, subtitles, quality, picture-in-picture, fullscreen and next episode.
- Shortcuts: `Space`/`K`, `J`/`L` or arrows, `M`, `F`, `C`, `,`/`.`, `0`–`9` and `Esc`. Details are in the [user guide](user-guide.md).
- Progress is sent every 15 s and on exit; at 90 % it is marked watched and progress is cleared. If 15 s or less remain, it does not resume.
- When an episode ends and there is a next one, an 8-second countdown appears.

## Signed URLs

- HMAC-SHA256 with a random 32-byte key per process.
- Default TTL: 8 hours.
- The signature covers the identifier and expiry; it is not bound to the user. Anyone with the URL can use it until it expires.
- Restarting the server changes the key, so previous URLs stop working.

## What advanced playback still needs

- Per-device capability profiles and a choice between direct play, copy and transcoding.
- Copying video when it is already 8-bit H.264, cutting at real keyframes.
- Hardware acceleration (Quick Sync/VAAPI); the server has `/dev/dri`, but it is not passed to the container.
- H.265, HDR/10-bit with tone mapping, deinterlacing, multichannel audio or passthrough, and image subtitles.
- ABR and a master playlist with several qualities.
- An explicit `transcode` API mode.

The improvement order is in [status](status.md). The Reduvid relationship still stands: converting files beforehand with [Reduvid](https://github.com/catlinux/reduvid) reduces the need for server transcoding.

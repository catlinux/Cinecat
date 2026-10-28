# Playback and codecs

[Español](../es/reproduccion.md) · [Back to README](../../README.en.md) · [Status](status.md)

Cinecat chooses between three paths for each file: **direct**, **copy** and **re-encode**. The decision uses the real `ffprobe` details (codec, profile, bit depth, height and tracks) and what the browser declares in the `cinecat_caps` cookie. The API still calls the HLS mode `remux`, whether it copies or re-encodes; the file details distinguish the method with `direct`, `copy` or `transcode`.

## Current decision

| Case | Mode |
|---|---|
| MP4/M4V with 8-bit H.264 (`yuv420p`/`yuvj420p`, not High 10, 4:2:2 or 4:4:4) and a default audio track that is AAC, MP3, or no audio | Direct: the file is served with HTTP range support. |
| MP4/M4V with H.265 that the browser plays directly: 8-bit with `hevc`, 10-bit with `hevc10`, and AAC, MP3 or no audio | Direct. |
| MP4/M4V with H.265 that is not played directly but is supported through MSE: 8-bit with `hevcmse`, 10-bit with `hevc10mse`; height within the maximum and keyframe index ready | HLS copy. |
| MKV or MOV with 8-bit H.264, height within the maximum and keyframe index ready | HLS copy. |
| MKV, MOV, MP4 or M4V with 8-bit H.265 (`hevcmse`) or 10-bit H.265 (`hevc10mse`), height within the maximum and keyframe index ready | HLS copy. |
| MP4 or M4V with 8-bit H.264 but unsupported audio (AC-3, DTS, etc.), height within the maximum and keyframe index ready | HLS copy. |
| AVI, TS or M2TS | Re-encode to H.264 + stereo AAC. |
| MPEG-4 Part 2 (DivX/Xvid), AV1, 10-bit H.264 or High 10/4:2:2/4:4:4 profiles | Re-encode to H.264 + stereo AAC. |
| Copy candidate above the maximum height, or with a pending or failed keyframe index | Re-encode while copying is not possible; the index stays queued for next time. |
| Any other extension | 415; the container is not supported. |

The **maximum height** is that of the selected quality: with `auto` or `original`, the `playback.max_height` setting (never above the original height). Direct MP4/M4V playback does not apply `playback.max_height`: if it is served directly, the original resolution is shown. The height limit only decides whether the file can be **copied** when it goes through HLS; if its height exceeds the selected quality, it is re-encoded. The `ffprobe` details decide codec, audio and pixel format; the extension only separates MP4/M4V from the rest.

An H.265 MP4 with unsupported audio is not served directly: it is copied if the browser declares MSE and copying is possible, otherwise it is re-encoded.

## How copying is decided

Copying needs the file's keyframe index:

- **MKV:** the Cues in the container are read, usually in milliseconds. It is detected by the EBML signature, not by the extension; a file named `.mkv` that is really MP4 still takes the `ffprobe` path.
- **MP4 and other containers:** the first time, packets are read with `ffprobe` in the background, one task at a time. While it finishes, that playback re-encodes; the next one can copy. Reading the packets of a large file can take a while (on the author's server, about 31 s cold for 5 GB).
- If the index has fewer than two keyframes, or the first one is not near the start, it is discarded and `ffprobe` is tried.

The index is stored in `cache/keyframes`, next to the database, under a key made from the path, size and date. It is invalidated only when the file changes and can be deleted safely: it will be calculated again. Reading the Cues of an MKV takes 0.05–0.26 s per file in the author's server measurements.

Once the index is ready, the server calculates the segment boundaries: it starts at 0 and the next cut is the first keyframe at least 6 s after the previous one, with a 0.25 s margin so the seek does not land on another keyframe. The VOD playlist uses the real durations of those cuts.

If copying fails before generating the requested segment, the request returns the error, the file is marked in memory as unsuitable for copying (path, size and date) and the next playback re-encodes. The mark disappears when the server restarts.

## HLS: copy and re-encode

Both modes share MPEG-TS segments, seeking and process limits; the VOD playlist is used when the duration is known:

| | Copy | Re-encode |
|---|---|---|
| Video | `-c:v copy`, without scaling or quality loss. | `libx264` or VA-API, with configurable preset/CRF or `qp` and maximum height. |
| Audio | One track to stereo AAC at 160 kbps. | One track to stereo AAC at 160 kbps. |
| Cuts | Real keyframes, with `-copyts`, a 10 s `-output_ts_offset` and `-segment_times`; the playlist uses real durations. | Keyframes forced every 6 s and 6 s segments. |
| When | Supported codec, height within the maximum and ready index. | Every other case. |

Copying is only offered when the video does not need scaling. Choosing a quality below the original resolution forces a re-encode; `original` keeps the original height as long as it does not exceed `playback.max_height`.

The rest of the flow is shared:

- With its own VOD playlist, FFmpeg does not start when the playlist is requested: it starts on the first segment request, at the requested point. If there is no duration (only possible when re-encoding), it starts while preparing the `event` playlist.
- The playlist is a complete VOD when the duration is known. Copying always needs it; if it cannot be determined, the file is re-encoded. Re-encoding uses FFmpeg's `event` playlist when there is no duration, and then long seeks are not fast.
- If a segment more than 24 s ahead of what has been generated is requested, or a gap behind it, the server restarts FFmpeg at that segment. For copying, the seek is 0.2 s after the keyframe so it does not land on the previous one.
- A segment is complete when it appears in its process playlist; on restart, the half-written one and those about to be rewritten are discarded.
- Processes run at low priority with a limited number of threads.

On the author's server, copying runs at about 37× real time and uses almost no CPU.

## Hardware acceleration

Re-encoding can use the integrated Intel GPU through **VA-API**. It does not change copying, which still leaves the video untouched, or the direct/HLS decision.

Requirements:

- The image includes `intel-media-va-driver` and sets `LIBVA_DRIVER_NAME=iHD`.
- The host exposes `/dev/dri/renderD128` (it can be changed with `CINECAT_VAAPI_DEVICE`) and `docker-compose.gpu.yml` mounts it into the `servidor` container. Without that file, `docker-compose.yml` still starts and everything runs in software.
- The deployment's `.env` enables `COMPOSE_FILE=docker-compose.yml:docker-compose.gpu.yml` (the updater respects it).
- The `playback.hardware` setting is `auto` (the default). `off` always disables it. It is read when preparing each playback, so changes apply to new playbacks.

The first time it is needed, the server checks VA-API with a one-second synthetic video and a 15 s limit. The result is remembered until restart and logged once, whether it succeeds or not.

There are two routes:

- **CPU+GPU:** the CPU decodes and scales; the GPU encodes with `h264_vaapi`. This is the general route, including DivX/Xvid, 10-bit H.264, interlaced video, 10-bit H.265 or when the height must be reduced.
- **Full GPU:** the GPU decodes and encodes, with no scaling filter. It is only used for 8-bit H.264 (`yuv420p`/`yuvj420p`, not High 10/4:2:2/4:4:4) or 8-bit H.265 Main, progressive or with unknown field order, and when the source height does not exceed the output height.

The GPU's scaling unit is not used: `scale_vaapi` and `scale_qsv` are unavailable on this GPU and are never invoked. Encoding uses `qp = CRF + 2`, `-g 250` and the same forced cuts every 6 s; audio and the rest of the HLS flow are unchanged.

On the author's server, 60 s of 1080p→720p movie goes from 44 s of CPU with libx264 (HEVC source) to 19 s with CPU+GPU, and from 39 s to 18 s with H.264; with everything on the GPU and no scaling, 0.6 s (about 14×). These are measurements from that server, not a promise for other hardware.

If the hardware process fails before generating the requested segment, it is logged, the file (path, size and date) is marked as unsuitable for hardware and the same segment is retried in software without the client seeing an error. If the general check fails, hardware stays disabled until restart. Hardware re-encodes count towards "Simultaneous transcodes" like the others.

## Process limits

| Limit | Value |
|---|---|
| Initial playlist wait | 20 s |
| Segment wait | 2 min |
| Maximum process lifetime | 4 h |
| Idle stop | 90 s without requests |
| Segments generated ahead | 20 (~2 min) |
| Simultaneous transcodes | `playback.max_transcodes`, 2 by default |
| Simultaneous copies | 6 |
| HLS job cleanup | 1 h unused, checked every 15 min |

Copies **do not count** towards "Simultaneous transcodes". If more than 6 copies are requested, the oldest copy without recent requests is stopped. Idle and look-ahead stops apply to both copies and re-encodes.

## What the user sees

The file details show the planned method in `playback.method`:

- `direct`: direct playback.
- `copy`: "Copia (sin recodificar)".
- `transcode`: re-encode to H.264 + stereo AAC.

The details are calculated from the cached `ffprobe` record, the browser capabilities and the maximum height setting. The keyframe index is checked when preparing playback: that is why the first time an MP4 is watched it may show copy and still re-encode while the index is calculated in the background.

The player offers `auto`, 480p, 720p, 1080p (never above the original height or `playback.max_height`) and `original`. Direct playback only has `original`. In HLS, copying does not change the resolution; lower qualities are re-encoded.

## Audio, subtitles and quality

- **Direct audio:** the original file is kept. It is served directly if the default track is AAC or MP3 (or there is no audio).
- **HLS audio:** one track is chosen by its position in the `ffprobe` list and converted to stereo AAC at 160 kbps. The user's and movie's preference is stored in `item_preferences`.
- **Subtitles:** embedded text tracks and external `.srt`, `.vtt` or `.ass` files next to the video, with language and `forced` suffixes. They are converted to WebVTT. Image tracks (PGS/VobSub) are not supported and HLS has no separate subtitle rendition.
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

## What format to store

| Recommendation | Why |
|---|---|
| **8-bit H.265**, in MKV or MP4 | Best balance: it is copied when the browser supports it through MSE and takes less space than H.264. Roughly 30–50 % smaller than H.264 at similar quality, with no guaranteed figures. |
| **8-bit H.264**, in MP4 or MKV | Still very compatible: MP4 with AAC/MP3 is served directly and MKV/MOV is copied without re-encoding. |
| Avoid **10-bit H.265** unless every device needs it | Many browsers and devices cannot play it; the server GPU does not fully decode 10-bit HEVC. It is only copied if the browser declares `hevc10mse`. |
| **Original audio** (AAC, MP3, AC-3…) | Direct playback keeps it; in HLS only the audio is re-encoded to stereo AAC, at a small cost. |
| **Text subtitles** embedded in the MKV or in `.srt` next to the video | They are converted to WebVTT; image subtitles (PGS/VobSub) cannot be shown. |
| **Keyframes every 2–10 s** | This is common and allows reasonable segments and seeks when copying. Very long intervals make segments longer and seeks less precise. |
| Avoid **AVI and TS/M2TS** in the library | They are always re-encoded; converting to MKV/MP4 preserves the codec better and saves server work. |

Re-encoding uses VA-API when available and libx264 when not. Copying uses almost no CPU because it neither decodes nor encodes the video.

## What advanced playback still needs

- Per-device profiles and automatic quality selection.
- HDR with tone mapping and 10-bit handling beyond the H.265 that the browser already supports through MSE.
- Deinterlacing.
- Image subtitles and unsupported, multichannel or passthrough audio.
- ABR and a master playlist with several qualities.
- An explicit `transcode` API mode separate from `remux`.

The improvement order is in [status](status.md). The Reduvid relationship still stands: converting files beforehand with [Reduvid](https://github.com/catlinux/reduvid) reduces the need for server transcoding.

# Playback and codecs

[Español](../es/reproduccion.md) · [Back to README](../../README.en.md)

> Status: **design**. Figures are indicative and must be measured in practice.

## Core idea

Each device **declares what it can do** and the server **picks the cheapest way** to play each video without losing quality or smoothness.

## Client capability profile

On connecting, the client sends a profile including, among others:

- video codecs it decodes (H.264, H.265, VP9, AV1…), profiles and levels, 10-bit, HDR;
- audio codecs (AAC, AC3, EAC3, Opus, DTS…) and channels;
- containers and subtitle formats it supports;
- maximum resolution and available bandwidth;
- **decoding preference:** device hardware, device software, or delegated to the server.

How to obtain it: on Android, by querying the system codecs; in the browser, with the media capabilities APIs and format checks.

## Decision chain

1. **Direct play:** the client understands video, audio, container and subtitles. The server just sends the file (with seek support). Almost no server cost.
2. **Remux:** the video is compatible but the container is not (for example MKV in a browser). The server repackages on the fly to HLS without re-encoding video. *Not used today: see below.*
3. **Partial transcoding:** the video is copied and only the audio is converted (for example DTS to AAC/AC3).
4. **Full transcoding:** the server re-encodes the video (with hardware acceleration if available) and adapts resolution and bitrate.

**Implemented today:** MP4/M4V are served directly; everything else (MKV, AVI, MOV, TS, M2TS) is served over HLS, **always re-encoded** to H.264 (libx264 `veryfast`, CRF 20, up to 1080p) and **stereo** AAC, with keyframes every 6 s and 6 s MPEG-TS segments. The real library is almost all DivX/MPEG-4 with AC-3 5.1 or MP3 audio, which no browser plays through MSE, and 5.1 AAC is not accepted either, hence the stereo downmix. The rest of the device-profile chain described in this document (remux, partial transcoding, hardware, subtitles) is still future design.

**Seeking.** The playlist is a complete VOD from the first request. FFmpeg starts on the first segment request, at that segment. If a segment more than 4 segments (24 s) ahead of what has been produced is requested later, or a gap behind it, the server stops FFmpeg and restarts it at the requested segment (`-ss` + `-start_number`), as Jellyfin does. So that segments from different processes fit together, all output timestamps carry a fixed 10 s offset (`-output_ts_offset`): segment N always starts at the same instant, whichever process produced it. A segment is considered complete once it appears in its process playlist (`ffmpeg.m3u8`); on restart, the half-written one and those the new process will rewrite are deleted.

The server **prefers H.265** when the client supports it (less space and bandwidth) and **falls back to H.264** when it does not.

## Who decodes

| Situation | Decision |
|---|---|
| Powerful phone or tablet with a hardware H.265 decoder | Direct play; the device itself decodes. |
| Weak TV or no codec support | The server transcodes (H.264) and the TV just displays. |
| Capable client whose hardware decoding fails | The user can force software on the client (for example libmpv) or delegate to the server. |
| Limited bandwidth (away from home) | Transcoding with an adapted bitrate. |

The preference can be set per device and per user.

## Browsers and H.265

It cannot be taken for granted. As of 2026-10-24, according to the sources consulted:

- Chrome supports H.265 with **hardware decoding**; on Linux it depends on the GPU and setup.
- Firefox supports it with hardware on Windows (since 134), macOS (136) and Linux (137), with initial container limitations.
- Real-world results vary between machines.

That is why the web client **checks** whether it can play the video and the server **falls back to H.264** when it cannot. Until the client-capability decision exists, HLS is always re-encoded to H.264.

## Cases that force transcoding

- 10-bit video on devices that only accept 8-bit.
- HDR / Dolby Vision on SDR screens (conversion to SDR with tone mapping if FFmpeg allows it).
- Image subtitles (PGS/VobSub) on clients that cannot show them (burned into the video).
- Unsupported audio (DTS, TrueHD).

## Subtitles, audio and chapters

- Text subtitles: sent as text and rendered by the client (with style support when possible).
- Image subtitles: shown if the client can, or burned into the video.
- Audio and subtitle track selection by language, remembering the choice.
- Chapters and timeline markers.

## Sessions

- Control of **simultaneous transcodes** so the server is not overloaded.
- Cleanup of temporary files.
- Log of why each mode was chosen (useful for debugging and for showing it to the user).

## Relationship with Reduvid

Converting videos beforehand with [Reduvid](https://github.com/catlinux/reduvid) to a compatible format reduces the need for transcoding on the server.

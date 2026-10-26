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
2. **Remux:** the video is compatible but the container is not (for example MKV in a browser). The server repackages on the fly (HLS with fMP4) without re-encoding video.
3. **Partial transcoding:** the video is copied and only the audio is converted (for example DTS to AAC/AC3).
4. **Full transcoding:** the server re-encodes the video (with hardware acceleration if available) and adapts resolution and bitrate.

Before remuxing, the server checks the keyframe interval of the first video stream with `ffprobe`. If it exceeds **15 s** (common in old AVI/DivX files), it does not attempt the remux without re-encoding: the first HLS segment could be huge and slow to start. It returns a clear error instead. Real transcoding for those files is left for Step 2 of Phase 3.

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

That is why the web client **checks** whether it can play the video and the server **falls back to H.264** when it cannot. MKV files are served through HLS with fMP4 segments, which do support H.265.

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

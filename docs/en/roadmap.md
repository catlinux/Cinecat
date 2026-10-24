# Roadmap

[Español](../es/hoja-de-ruta.md) · [Back to README](../../README.en.md)

> Indicative plan. Each phase ends with a review before moving to the next. Versioning `x.y.z`.

## Phase 0: preparation (`0.0.x`)

- Initial bilingual documentation (**done in `0.0.1`**).
- Check the name, decide the technology stack, commit convention and design direction. License already decided (GPL-3.0 with attribution).
- Create `CLAUDE.md` and the code repository structure.

## Phase 1: minimal server (`0.1.0`)

- Library scanning and filename parsing.
- `es-ES` metadata (TMDB) and image cache.
- Users, sessions and a versioned API with an OpenAPI contract.
- Direct play and remux (HLS fMP4).
- Automated tests with test videos generated with FFmpeg.

## Phase 2: web client (`0.2.0`)

- Login, PIN profiles, library, movie or series page.
- Player with subtitles, audio selection and chapters.
- Own design, light and dark, adaptable to phone and tablet.
- Secure access from outside the home (HTTPS and VPN guide).

## Phase 3: advanced playback decision (`0.3.0`)

- Per-device capability profiles and decoding preference.
- Partial and full transcoding, with optional hardware acceleration.
- Special cases: 10-bit, HDR, image subtitles, unsupported audio.

## Phase 4: Android and Android TV (`0.4.0`)

- Android app with native player and software fallback.
- TV interface designed for the remote.
- Offline download (if decided).

## Phase 5: polish and first stable release (`1.0.0`)

- Security and license review.
- User documentation and plain-language glossary.
- Packaging and deployment (Docker, Debian package).

## Afterwards

- Photos and slideshows.
- Intro/credits segments.
- Internet content and live TV (only if decided).
- Other platforms.

Task details in [TODO.en.md](../../TODO.en.md).

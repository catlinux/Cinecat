# Architecture and security

[Español](../es/arquitectura.md) · [Back to README](../../README.en.md)

> Status: **design proposal**. None of this is implemented.

## Overview

```
 ┌────────────┐   ┌──────────────┐   ┌──────────────────┐
 │ Web client │   │ Android app  │   │ Android TV app   │
 └─────┬──────┘   └──────┬───────┘   └────────┬─────────┘
       │      HTTPS · versioned REST API        │
       └──────────────────┬─────────────────────┘
                   ┌──────▼───────┐
                   │   Cinecat    │── FFmpeg (external)
                   │   server     │── Local database
                   └──────┬───────┘── Image cache
                          │
                 Video library (disk)
                 Metadata sources (TMDB, ...)
```

## Components

- **Server:** scans the library, identifies titles, downloads and caches metadata, manages users and sessions, decides how to play and serves the video. It orchestrates **FFmpeg** as an external tool (not distributed in the repository).
- **Web client:** PC, phone and tablet. Plays with HLS and browser APIs.
- **Android and Android TV app:** native player with hardware decoding when available, and a software fallback. The TV interface is designed for the remote (focus navigation).
- **API:** REST, versioned and documented with OpenAPI (contract first). Clients are written against the contract.
- **Database:** local and simple at first (for example SQLite).

## Technology stack (decided)

| Part | Choice | Note |
|---|---|---|
| Server | **Go** | Single binary, easy deployment on Debian and Docker, native concurrency for managing FFmpeg processes, comfortable one-person maintenance. |
| Web | **TypeScript + SvelteKit** + hls.js | Less client-side JavaScript than a heavier framework (important for TV and modest phones), simple routing and SSR. hls.js already verified as Apache-2.0. Own "Marquesina" design, no generic templates. |
| Android / Android TV | **Kotlin + Jetpack Compose** (with Compose for TV) + **Media3/ExoPlayer**, optional libmpv as a software-decoding fallback | A single Compose codebase for Android and Android TV. Media3 already verified as Apache-2.0; check libmpv's license depending on the build before integrating it. |

## Security

The platform may be exposed to the internet, so it is designed with these rules:

1. **HTTPS required** outside the local network. Using a reverse proxy (for example Caddy) is documented, along with a VPN alternative (Tailscale/WireGuard) for personal use.
2. **Authentication:** username and password with a strong hash (argon2id or similar), short-lived renewable session tokens, and logout that invalidates the token.
3. **Brute-force protection:** attempt limits, temporary lockout and alerts.
4. **PIN:** a short PIN is **not** enough by itself for remote access. It is used to switch profiles inside an already authenticated device (for example, the TV). For remote access: strong password and, later, a second factor (TOTP or passkey).
5. **Signed, short-lived playback URLs.** Never passwords or permanent tokens in the URL.
6. **Roles:** administrator and users; per-library permissions.
7. **Audit log** of sign-ins and administrative actions.
8. **No telemetry.** Outgoing connections only to the configured metadata sources.
9. **Safe external processes:** FFmpeg is invoked with an argument list, never by string concatenation or through a shell.
10. **Dependencies:** license review and check for known vulnerabilities.

See also [SECURITY.md](../../SECURITY.md).

## Planned deployment

- Server on Debian (native or Docker container).
- FFmpeg installed on the system.
- Server hardware acceleration (VAAPI, NVENC, QSV) as an advanced option.
- Container image: if it ever bundles FFmpeg, review the license obligations.

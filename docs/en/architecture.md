# Architecture and security

[Español](../es/arquitectura.md) · [Back to README](../../README.en.md)

> Status: **design proposal**. None of this is implemented. The technology stack is decided in Phase 0.

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

## Technology stack (proposals, to be decided in Phase 0)

| Part | Options | Note |
|---|---|---|
| Server | Go, Rust, Kotlin (Ktor) or TypeScript | Weigh easy deployment on Debian and Docker, concurrency and one-person maintainability. |
| Web | TypeScript with a modern framework + hls.js or Shaka | Own design, no generic templates. |
| Android / TV | Kotlin + Jetpack Compose (+ TV libraries) + Media3/ExoPlayer, optional libmpv | Check licenses before using each piece. |

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

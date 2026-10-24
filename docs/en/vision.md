# Vision and decisions

[Español](../es/vision.md) · [Back to README](../../README.en.md)

## The problem

People who keep their own video library at home usually end up using someone else's media server. It works, but:

- the interface is not always comfortable or pleasant;
- playback fails or stutters on some devices (for example, a weak TV that cannot decode H.265);
- the user has little control over **who decodes**: the device itself or the server.

## The proposal

**Cinecat**: an own server and own clients, with an interface that has personality, that automatically pick the best way to play each video on each device, and that fetch covers and information in Spanish from existing databases.

## Decisions made (2026-10-24)

| # | Decision |
|---|---|
| 1 | **Own server and clients**, with an own name (Cinecat). |
| 2 | We **reuse everything that is legal** from other free projects, with credits. The rest is written from scratch. |
| 3 | Platforms: **web** (PC, phone, tablet; reachable from outside the home) and **Android / Android TV**. Nothing else for now. |
| 4 | **Optimal playback**: H.265 preferred, H.264 as a fallback; the device declares what it can do and the server decides. |
| 5 | **Metadata from existing databases**, in Spanish (Spain). We do not build our own. |
| 6 | **Protected access** with username and password (and PIN for profiles). |
| 7 | **Own, distinctive interface**, unlike what is usual in AI-generated projects. |
| 8 | Documentation in **Spanish and English**, commits in **Spanish**, versions **x.y.z**. |
| 9 | Single author and maintainer: **Catlinux**. |
| 10 | License **compatible with the GPL code** that gets reused, preserving authorship and credits. Chosen before publishing code. |

## Out of scope for now

- Live TV and internet content (maybe later).
- Photos and slideshows (later versions).
- Platforms other than web, Android and Android TV.

## Principles

1. **Easy by default, powerful if you want it.** Options are explained in plain language.
2. **Never lose or damage the originals.**
3. **Privacy:** no telemetry and no unnecessary connections.
4. **Always credit:** whatever is reused is cited and respected.
5. **Technical honesty:** if something cannot be guaranteed (for example H.265 in a browser), we say so and offer an alternative.

## Related projects

- **Reduvid** (sister project by the same author): a video compressor to get files ready for Jellyfin and for Cinecat.

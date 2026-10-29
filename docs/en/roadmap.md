# Roadmap

[Español](../es/hoja-de-ruta.md) · [Back to README](../../README.en.md) · [Status](status.md)

The current version is **0.34.1**. Feature details, versions and limits are in [status](status.md); this page only summarises the phases.

## Phase 0: preparation (`0.0.x`) — done

Bilingual documentation, name, technology stack, GPL-3.0 license with attribution, and the "Marquesina" visual direction.

## Phase 1: minimal server (`0.1.0`) — done

Go server, SQLite and migrations, argon2id authentication, scanning, filename parser, OpenAPI contract and initial direct/HLS playback.

## Phase 2: web client (`0.2.0`) — done, with later changes

Login and session, library, detail page, player, light/dark themes and accessibility. PIN profiles were not included; the current account screen changes the password, it does not create profiles.

## Phase 3: advanced playback (`0.8.0`–`0.13.0`) — partial

Fallback transcoding, progressive segmentation, seeking, FFmpeg supervision, custom controls, audio, text subtitles and preferences were implemented. Per-device capability decisions, H.265, hardware, HDR, image subtitles and ABR remain outside.

## Phase 3b: metadata, series and administration (`0.11.0`–`0.16.0`) — done

TMDB, full detail pages, collections, series, hot settings, administration panel, web updates and the host updater service.

## Next batches

The agreed order is: favorites and watchlist; collections from the movie page and collection identification; libraries with multiple folders and a browser; upload/deletion with a trash bin; users and profile; sessions, devices, activity and backups; user-uploaded subtitles; plugins; DLNA/UPnP; library types; Android and Android TV clients. Details are in [status](status.md).

## Phase 4: Android and Android TV — in progress

Kotlin + Jetpack Compose + Media3/ExoPlayer, with optional libmpv. Done: the server (device sessions, TV pairing by code, native capabilities in playback, a stable API contract and TMDB images served by the server itself) and a single APK for phone and TV that signs in, browses the library and plays. Still missing: the TV's own interface, background playback and publishing.

## Phase 5: first stable release (`1.0.0`) — future

Security and license review, user documentation, packaging and reproducible deployment. No date.

## Out of scope for now

Live TV and internet content; photos and slideshows; platforms other than web, Android and Android TV. They will be considered later.

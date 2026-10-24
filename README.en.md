# Cinecat

> Own server and clients (web, Android and Android TV) for your media library: pick the best codec (H.265/H.264), direct play or transcoding, and Spanish-language metadata.

**Languages / Idiomas:** [Español](README.md) · English

**Status:** design phase. Version `0.0.2` (documentation only, no code yet).

---

## What is Cinecat?

Cinecat is a project to play **your own videos** (the ones on your home server) on your TV, computer, phone and tablet, with a custom, carefully designed and practical interface, and a server that decides by itself the best way to play each video on each device.

It exists because current solutions work, but their interface and some playback decisions don't fit what we want. Cinecat is not meant to be a clone: it makes use of the best existing free software (always with credits and respecting licenses) and builds something of its own on top.

## Goals

- **Own server** that indexes your library, fetches covers and information from open databases and serves the video.
- **Own clients:** web (PC, phone and tablet, also from outside your home) and **Android** and **Android TV** apps.
- **Optimal playback:** each device declares what it can decode and the server picks between direct play, container remux or transcoding, preferring **H.265** with **H.264** as a fallback. See [playback](docs/en/playback.md).
- **Spanish (Spain) metadata** from existing databases (we do not build our own). See [metadata](docs/en/metadata.md).
- **Secure access** with user accounts and passwords, and PIN for profiles. See [architecture](docs/en/architecture.md).
- **An interface with its own personality**, no generic templates.

## Out of scope for now

Live TV and internet content. Photos and slideshows will come in later versions. Platforms other than web, Android and Android TV also come later.

## Visual direction

**Chosen: "Marquesina".** 1950s-60s neighborhood cinema: warm maroon and cream tones, gold and orange accents, a characterful serif typeface (Fraunces) for titles. Illustrated posters of our own (none is a real photo) with varied scenes and lighting to give the library some color.

Reference mockup: ["Marquesina" artboard on Claude](https://claude.ai/artifact/J2o3AdQxPbN9woGjUfWQFk) (maintainer's private link; will be replaced with screenshots in `docs/` once a real interface exists).

## Documentation

| Topic | Español | English |
|---|---|---|
| Vision and decisions | [vision](docs/es/vision.md) | [vision](docs/en/vision.md) |
| Architecture and security | [arquitectura](docs/es/arquitectura.md) | [architecture](docs/en/architecture.md) |
| Playback and codecs | [reproduccion](docs/es/reproduccion.md) | [playback](docs/en/playback.md) |
| Metadata | [metadatos](docs/es/metadatos.md) | [metadata](docs/en/metadata.md) |
| Licenses and credits | [licencias-y-creditos](docs/es/licencias-y-creditos.md) | [licenses-and-credits](docs/en/licenses-and-credits.md) |
| Roadmap | [hoja-de-ruta](docs/es/hoja-de-ruta.md) | [roadmap](docs/en/roadmap.md) |

Other: [CHANGELOG](CHANGELOG.en.md) · [TODO](TODO.en.md) · [Contributing](CONTRIBUTING.en.md) · [Security](SECURITY.md) · [Credits](CREDITS.md) · [Authors](AUTHORS)

## License

**To be decided.** Since we want to reuse free code from other projects (several GPL-licensed), the final license will be one compatible with that code that always preserves authorship and credits. It will be applied before the first line of code is published. See [licenses and credits](docs/en/licenses-and-credits.md).

## Authorship

Author and maintainer: **Catlinux** ([github.com/catlinux](https://github.com/catlinux)).

## Data attribution

This product uses the TMDB API but is not endorsed or certified by TMDB.
(Once clients exist, this notice and the TMDB logo will appear on the "About" screen.)

# Metadata

[Español](../es/metadatos.md) · [Back to README](../../README.en.md)

> Status: **design**. The terms of each source were checked on 2026-10-24 and **must be reviewed** before implementation, because they may change.

## Principle

We do not create our own database: we use existing ones, in **Spanish (Spain, `es-ES`)**.

## Sources

| Source | Use | Relevant terms |
|---|---|---|
| **TMDB** | Main source for movies and series: titles, synopses, cast, posters, backdrops, certifications by country. | Free for **non-commercial** use, with attribution (logo and notice) and a time limit on keeping cached data. Commercial use needs a license. |
| **TheTVDB** | Supplement for series (episodes, translations). | Free for small projects with attribution, or through a per-user subscription. It reserves the right to change terms. |
| **Fanart.tv** | Extra artwork (logos, backdrops). | Requires a project key; the personal key is optional. |
| **TVmaze** | Optional for series schedules. | Data under CC BY-SA 4.0 (share-alike). |
| **OMDb** | Optional for ratings. | Daily limit on the free plan. |

FilmAffinity is **not** used: it has no official API and could only be obtained by scraping the website.

## File identification

1. Try to read **local metadata** if present (for example `.nfo` files and artwork next to the video).
2. Parse the **file and folder names** to extract title, year, season and episode (with a name-parsing library, chosen in Phase 0 by license and quality).
3. Search TMDB with that title and year, in `es-ES`.
4. If in doubt, offer the user a **manual fix** (pick the right result) and remember it.

## Language and image strategy

A known problem on other servers is that, when an image is missing in the main language, the title ends up with no poster or one without text. In Cinecat the server picks the priority **itself**:

1. Poster in `es-ES`.
2. Poster in generic Spanish.
3. Poster in English.
4. Poster with no language.

Images are requested from the source in several languages at once and this order is applied; the user will be able to change it.

## Cache

- Needed data and images are stored locally to work offline.
- They are **refreshed** periodically, respecting each source's retention limit (TMDB: check the current period before implementing).

## Attribution

- "About" screen with the official TMDB logo, unaltered, and the notice: *This product uses the TMDB API but is not endorsed or certified by TMDB.*
- The TMDB logo must be less prominent than Cinecat's.
- Credits for the other sources in [CREDITS.md](../../CREDITS.md).

## Age ratings

**Spain's** certifications will be used when the source provides them, to filter by profile (for example, a children's profile).

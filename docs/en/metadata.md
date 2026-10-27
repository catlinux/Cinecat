# Metadata

[Español](../es/metadatos.md) · [Back to README](../../README.en.md) · [Status](status.md)

Cinecat does not build its own database: it uses TMDB as the main source and, when there is no TMDB key, OMDb as a movie fallback. Interface text and proper names remain in Spanish.

## Sources

| Source | Actual use | Notes |
|---|---|---|
| **TMDB** | Movies and series: title, original title, synopsis, cast, crew, genres, rating, posters, backdrops, stills, seasons, episodes and collections. | Main source since 0.11.0. The key is set with `CINECAT_TMDB_API_KEY`. |
| **OMDb** | Movies only: synopsis and poster. | Fallback when there is no TMDB key. It has a daily quota. Not used for series or identification. |
| TheTVDB, Fanart.tv, TVmaze | Not integrated. | They were documented as possible sources, but no code uses them. |
| FilmAffinity | Not used. | No official API, and no scraping. |

## Identification

- **Movies:** variants of the file and folder title are generated, searched on TMDB and scored by word overlap, year and saga rules. The default threshold is `0.75`, configured through `metadata.match_threshold`.
- **Series:** the series is the top-level folder. It is identified with TMDB, then seasons and episodes are requested. A folder that is directly the library root does not create a series, because there is no top-level folder.
- **States:** `auto` (identified by the scan), `manual` (corrected by an administrator), `none` ("not this one") or empty (unidentified and retried). The scan does not touch `manual` or `none`.
- **Correction:** an administrator can search candidates, pick one, mark "not this one" or return to automatic. For series, correction requests seasons and episodes again.

Local `.nfo` files are not read and age certifications are not used: those features are not implemented.

## Language

`metadata.language` accepts `es-ES`, `es-MX` and `en-US`, with `es-ES` by default. It affects new requests; it does not rewrite saved records.

For synopses, if the preferred language has no text, variants are tried: for Spanish, the other variant, `es` and `en-US`; for English, `en-US`; for others, the base language and `en-US`. The web app shows "Sinopsis no disponible en castellano" when the stored synopsis does not start with `es`.

## Images

- Poster priority: `es-ES` → generic Spanish → English → no language; within each language, the highest vote.
- Common sizes: poster `w500`, lists `w342`/`w780`, backdrops `w780`/`w1280`, stills `w300` and season poster `w342`.
- Manual images are locked against scans. If the `tmdb_id` changes, they are unlocked.
- If a season was queried and TMDB had no poster, an empty marker is stored to avoid repeating the request.

## Cast and crew

TMDB returns cast and crew; up to 30 cast credits and up to 12 crew credits are stored in a fixed order. The page shows character, photo and name; when the photo is missing, initials are used. Series also store credits.

## Cache

- **TMDB:** raw responses and images in `<database directory>/cache/tmdb`; 7-day TTL. 404 responses are cached too so failed searches are not repeated. The cache key excludes `api_key`.
- **OMDb:** 30-day disk cache in the user's cache directory; failed searches are also stored.
- The 7-day TMDB TTL is a starting value and has not been checked against current terms; review it before production.

## Collections

TMDB provides `belongs_to_collection`. Cinecat creates or updates the collection, respects collections the user deleted or edited, and only shows automatic collections that reach `ui.collection_min_items` (2 by default). Manual collections are not touched by TMDB.

## Attribution

This product uses the TMDB API but is not endorsed or certified by TMDB.

Attribution and the logo should appear on the "About" screen when it exists; today the web app has no "About" screen. Source and dependency credits are in [CREDITS.md](../../CREDITS.md).

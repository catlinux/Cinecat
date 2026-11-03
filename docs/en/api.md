# API for native clients

[Español](../es/api.md) · [Back to README](../../README.en.md) · [Architecture and security](architecture.md)

Cinecat exposes an HTTP API so Android and Android TV apps can sign in, browse the library and play content without relying on the web client. The native apps use this contract. This page summarises its rules and main operations; the exact details of every operation are in the OpenAPI document.

## The contract

The contract is [api/openapi.yaml](../../api/openapi.yaml), an OpenAPI 3.0.3 document written in Spanish. It covers every API route registered by the Go server, including methods, parameters, request bodies, responses and session requirements.

The API lives under `/api/v1/`. `GET /healthz` is the only health route outside that prefix: it requires no session and returns `ok` as plain text. The `internal/apicontract` test compares the routes registered in the Go code with those documented in the OpenAPI file; if a route appears on one side but not the other, the suite fails and prints the difference.

## Stability

The `/api/v1/` contract is stable for apps:

- Inside `v1`, only routes and fields are added.
- Clients must ignore fields they do not know.
- Fields are never removed and their meaning never changes.
- An incompatible change would require a new version, `/api/v2`.

`GET /api/v1/system/version` requires no session and returns:

| Field | Meaning |
|---|---|
| `server` | Server version. |
| `commit` | Build commit; it may be empty. |
| `built_at` | Build date in RFC 3339; it may be empty. |
| `api_level` | Level of the `/api/v1` contract; currently `1`. |
| `min_client` | Minimum native app version accepted by this server; currently `0.0.0`. |

The app must compare its own version with `min_client` and warn the user when it is lower. `api_level` only increases when the contract changes in an incompatible way.

## Android app updates

`GET /api/v1/system/client-update?platform=android&version_code=N` lets the Android app check whether the server version has a newer published APK. It requires a valid session, but not the admin role.

When an update is available it returns `200` with `version`, `version_code`, `url`, `sha256` and `notes`. The URL points to the server release APK and `sha256` is its lowercase SHA-256 checksum; `notes` contains the CHANGELOG section when it is available in the checker cache.

It returns `204` when update checks are disabled, `platform` is not `android`, the server version is invalid, the client is already up to date or the checksum cannot be fetched. `version_code` is required, must be a non-negative integer and a missing or invalid value returns `400`.

## Authentication

`POST /api/v1/auth/login` accepts the credentials and device details:

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "ana",
  "password": "…",
  "device_name": "Living room TV",
  "client": "androidtv"
}
```

`client` accepts `web`, `android` and `androidtv`; when omitted, it is treated as `web`. The response contains `token`, `expires_at` and `session_id`.

Except for login, `POST /api/v1/auth/pair`, `POST /api/v1/auth/pair/token`, `/healthz`, `/api/v1/system/version` and signed playback URLs, requests carry this header:

```http
Authorization: Bearer <token>
```

Web sessions last 24 hours; `android` and `androidtv` sessions last 90 days. They are renewed on use when less than half of their lifetime remains. If a request returns `401`, the session is missing, invalid or expired: sign in again.

The current session and the user's other sessions are listed by `GET /api/v1/auth/sessions`; `DELETE /api/v1/auth/sessions/{id}` closes a specific session. Each list entry includes `profile`, the profile for that device. Administrators can list and revoke an account's sessions through the `/api/v1/users/{id}/sessions` routes.

Login limits failed attempts per user and IP. When the limit is exceeded it returns `429` with a `Retry-After` header; the app must wait that long before retrying.

## Device profile

Each active session has its own profile that is applied during playback. It is read with `GET /api/v1/auth/sessions/{id}/profile` and replaced as a whole with `PUT` on the same route. `{id}` accepts a numeric identifier or `current` for the session making the request; only the user's own sessions can be read or edited, and a session belonging to someone else, expired or missing returns `404`. The body and response contain the three `SessionProfile` fields:

- `max_quality`: `""` (no limit), `480`, `720` or `1080`.
- `audio_language`: `""` (no preference) or a 2- or 3-letter lowercase code.
- `subtitle_language`: `""` (no preference), `off` (disabled) or a 2- or 3-letter lowercase code.

An invalid value returns `400`. The profile belongs to the session: closing it or pairing the device again discards it and leaves it empty.

## Pairing a device

For a TV or a device without a comfortable keyboard:

1. The app calls `POST /api/v1/auth/pair` with `device_name` and `client` (`android` or `androidtv`). The response includes `pair_id`, `secret`, `code`, `verification_path`, `expires_in` and `interval`.
2. The app shows the `XXXX-XXXX` code and a QR code pointing to `<server>/vincular?codigo=XXXX-XXXX`. The `secret` is never shown on screen or included in the QR code.
3. The user opens the URL, signs in to the web app and enters or approves the code. The web app uses `POST /api/v1/auth/pair/approve`: with `confirm: false` it gets a device preview, and with `confirm: true` it confirms access.
4. The app polls `POST /api/v1/auth/pair/token` with `pair_id` and `secret` every 5 seconds. Polling too quickly is rejected before 4 seconds. The final response contains `token`, `expires_at`, `session_id` and `user` with `username` and `role`.

Polling status codes:

| Code | Meaning |
|---|---|
| `200` | Pairing approved; the session is returned. |
| `428` | Still waiting for approval; wait for `Retry-After`. |
| `429` | Polling too fast or too many attempts. |
| `410` | Pairing expired or already used. |
| `404` | Invalid identifier or secret. |

## Available sections

`GET /api/v1/sections` tells which sections have content for the session user:

```json
{"movies": true, "series": false, "collections": true}
```

`movies` is true when there is at least one accessible movie, `series` when there is at least one accessible series and `collections` when there is at least one visible collection with an accessible movie. Administrators always receive all three as true. The web app and the apps use this response to hide empty sections; if the call fails or an older server returns `404`, they must show all of them.

## Photos

`GET /api/v1/photos` returns the direct subfolders and photos of a `photos` library. It requires `library_id` and accepts `folder` (path relative to the library root; empty for the root), `sort` (`date` or `name`, default `date`), `limit` (1–200; 50 by default) and `offset` (0 by default). `date` sorts by `taken_at` and, when it is missing, by the file modification date. Only a photos library the user can access can be queried; when it does not exist or is not visible, the route returns `404`.

The response has this shape:

```json
{
  "folders": ["2024", "Trips"],
  "items": [
    {"id": 17, "name": "IMG_0017.jpg", "width": 4032, "height": 3024, "taken_at": 1600000000, "size": 2456789}
  ],
  "total": 42
}
```

`folders` are the names of the direct subfolders of `folder`; `items` are the photos in that folder, paginated; `total` is the total number of photos in that folder. `taken_at` is the EXIF capture date in Unix seconds or `null` when it does not exist. JPEG, PNG, WebP and GIF are indexed; HEIC and RAW are not supported.

`GET /api/v1/photos/{id}/thumb` returns a JPEG thumbnail with a 400 px longest side, generated with FFmpeg and cached; it supports `ETag` and returns `304` when unchanged. `GET /api/v1/photos/{id}/original` serves the original with its `Content-Type` and HTTP range support (`206`). Both require a session and access to the library; when the photo does not exist or falls outside its library, they return `404`.

## Activity (administrators only)

`GET /api/v1/activity` returns the latest playbacks from all users. It combines progress and watched marks into one entry per user and item, using the most recent date, and includes user, title, series, season, episode, position, duration and watched state. It accepts `limit` (maximum 200; 50 by default) and `user_id` to filter. It requires the `admin` role.

## Security (administrators only)

`GET /api/v1/system/security-events` returns the latest 50 login lockouts and the total for the last 7 days (`lockouts` and `total_7d`). It includes no passwords or tokens. It requires the `admin` role.

## History and portal

`GET /api/v1/history?limit=N` returns the session user's history, newest first. `limit` defaults to 50 and is capped at 200. Each entry includes `item_id`, `title`, `kind`, series and episode data, `poster_url`, position, duration, `watched` and `updated_at`, and only content from accessible libraries appears. `DELETE /api/v1/history/{itemId}` removes that user's progress and watched mark for the item and returns `204`; when the item does not exist or is not accessible it returns `404`.

`GET /api/v1/home/preferences` returns every possible portal row with `id`, `title` and `hidden`. `PUT /api/v1/home/preferences` accepts `{"hidden":["id",…]}` and replaces the user's hidden set; absent rows become visible again and an unknown identifier returns `400`. `GET /api/v1/home` applies the preference.

## Playback

`GET /api/v1/items/{id}/play` prepares playback and accepts these query parameters:

| Parameter | Use |
|---|---|
| `caps` | Device capabilities, comma-separated. |
| `audio` | Index of the selected audio track. |
| `quality` | `auto`, `original`, `480`, `720` or `1080`. |
| `subtitle` | Subtitle identifier; empty to disable it. If it is an embedded image track (`e<N>`), playback switches to HLS and the subtitle is burned into the video. |

The apps declare in `caps` what their device plays: containers `mkv` and `mp4`; video `h264`, `h264hi10`, `hevc`, `hevc10`, `av1` and `vp9`; audio `aac`, `mp3`, `ac3`, `eac3`, `dts`, `truehd`, `opus` and `flac`; and `mch` if it decodes multichannel AAC. Each token means the device can decode that container, codec or format; `hevc10` implies `hevc`, and `mch` is only needed for AAC with more than two channels. If a native app declares `mch` and the chosen track has 6 or more channels, HLS delivers 5.1 AAC; otherwise it delivers stereo. For the server to treat the request as a native app, `caps` must include at least `mkv` or `mp4`. The full list and its effect are in [playback](playback.md#native-apps-android-and-android-tv).

The response includes `url`, `mode`, `expires_at`, `audio_tracks`, `selected_audio`, `qualities`, `selected_quality`, `subtitles`, `selected_subtitle`, `segments`, `autoplay_next` and `next_episode`. `mode` is `direct` or `remux`; when it is not direct, `reason` explains why (`container`, `video`, `audio` or `subtitle`).

Each entry in `subtitles` is a `SubtitleOption`: `burn_in` reports whether it is an embedded image track (PGS, VobSub or DVB); in that case `url` is empty and the client must not load it as WebVTT. The rest keep their signed WebVTT URL.

The `/stream` URL is relative to the server, signed and valid for 8 hours. It does not need `Authorization`: prepend the server base and pass it unchanged to the player, such as ExoPlayer. Do not add custom headers or modify the signature.

When an image track is chosen with `subtitle=e<N>`, `mode` is `remux`, `reason` is `subtitle` and the signed HLS URLs under `/stream` include `burnsub=<N>` (the track position in the file, the same `N` as in `e<N>`). Both `/stream/{id}` and `/stream/{id}/{file}` accept `burnsub` and force CPU HLS, never copy or VA-API. The WebVTT endpoint for an image track (`/api/v1/stream/{id}/subtitles/<sid>.vtt`) returns `404`.

Progress is stored with `PUT /api/v1/items/{id}/progress` and a body such as `{"position_seconds": 300, "duration_seconds": 1800}`; `DELETE` on the same route clears it. At 90 % of the duration the server marks the item as watched and removes progress. Watched state is controlled with `PUT` and `DELETE /api/v1/items/{id}/watched`.

Audio, subtitle and quality preferences are read with `GET /api/v1/items/{id}/preferences`, which returns `audio_index`, `subtitle_id` and `quality` (or `null` when there is no preference). They are saved with `PUT` on the same route and a partial body: absent fields are left untouched, `null` clears them and an empty `subtitle_id` means "no subtitles". `DELETE` removes all the user's preferences for that item. The apps restore them when the item is reopened.

`GET /play` applies this precedence: explicit parameter > item preference > session profile > default. The audio profile picks the first track in that language; if none exists, normal behaviour applies. The subtitle profile picks the first text track (never an image track) in that language, preferring non-forced tracks; `off` shows no subtitles by default. Maximum quality only acts as a cap for HLS: it trims the offered qualities and lowers `auto`, `original` and heights above the limit; direct playback is unchanged.

## Automatic next-episode playback

The setting is per user and series and is enabled when no row has been saved. It is read with `GET /api/v1/series/{id}/autoplay` and saved with `PUT` on the same route and a `{"enabled": true}` or `{"enabled": false}` body; both responses are `{"enabled": ...}`. They require a session and a visible series: a missing series, a hidden one or a non-series identifier returns `404`, and an invalid body returns `400`.

`GET /api/v1/series/{id}` includes `autoplay_next` with that setting. `GET /api/v1/items/{id}/play` also includes it: for an episode it reflects its series setting, and for any other item it is `true`. In addition, `next_episode` is `null` or `{id, title, season, episode}` with the immediately following visible episode, ordered by season and number; if there is no other episode or the item is not an episode, it is `null`. A next episode hidden by library permissions is not revealed.

## Downloading for offline viewing

`GET /api/v1/items/{id}/download` returns the original file of a movie or an episode. It requires a valid session and access to the item's library; a missing or hidden item, an item of a different kind, or a file-less item returns `404`.

The response uses `http.ServeContent`: it supports ranges (`206` and `If-Range`), sends `Last-Modified`, `Accept-Ranges: bytes`, `Content-Disposition` with `filename*=UTF-8''...` and the `Content-Type` for the extension (`.mkv` is `video/x-matroska`, `.mp4` and `.m4v` are `video/mp4`, everything else `application/octet-stream`). It does not re-encode or modify the file. A missing or invalid session returns `401`, a range outside the file `416` and an internal error `500`.

The Android app only uses this route when `/play` returns `mode: "direct"` with the device's capabilities. If it returns `remux`, it does not download and warns that the file needs conversion; no re-encoded versions are generated.

## Intro and credits segments

`segments` is a possibly empty list of `{kind, start, end, source}` objects. `kind` is `intro` or `credits`, `start` and `end` are in seconds and `source` is `chapter` or `manual`. Manual segments replace detected segments of the same type. Detection uses the technical file's chapters and their normalised titles (case, accents and extra spaces ignored); segments can also be marked by hand on the item page. There is no audio detection. If the file details cannot be read, the list is empty and playback continues unchanged.

The editing routes require the `admin` role:

- `GET /api/v1/items/{id}/segments` returns `{"manual":[…],"chapters":[…]}`. Each item has `kind`, `start` and `end`; `manual` contains the saved ones and `chapters` the ones detected in the file.
- `PUT /api/v1/items/{id}/segments` accepts `{"segments":[{"kind":"intro","start":10,"end":85}]}` and replaces every manual segment for the item. An empty list deletes them. It returns `204`; an invalid value (unknown or repeated type, negative `start` or `end` not greater than `start`) returns `400` with a Spanish message.
- `POST /api/v1/items/{id}/segments/apply-season` copies the episode's manual segments to the other episodes in the same series and season, replacing theirs, and returns `{"updated":n}`. If the item is not an episode, it returns `400`.

A missing or invisible item returns `404`; without the administrator role, `403`.

## Uploaded subtitles (administrators only)

`POST /api/v1/items/{id}/subtitles` accepts `multipart/form-data` with a required `file` and optional `language` and `label`. It supports `.srt`, `.vtt` and `.ass` up to 2 MB of UTF-8 or Latin-1/Windows-1252 text and rejects binaries. It returns `201` with `id` (`u<N>`), `language`, `label` and `codec`; the subtitle appears in `/play` `subtitles` with `source: "uploaded"` and its signed WebVTT URL. `DELETE /api/v1/items/{id}/subtitles/{subId}` accepts `u<N>` or the row number, deletes the row and its file and returns `204`. Files live in `subtitles/` next to the database.

## Images

On item pages, `poster_url` and `backdrop_url` are paths relative to the server (`/api/v1/images/tmdb/...`), not `image.tmdb.org` URLs. The image is served by the server itself and requires the session, like the rest of the authenticated routes.

Clip thumbnails and music covers are also served by the server and require the session:

- `GET /api/v1/items/{id}/thumbnail` returns the clip thumbnail.
- `GET /api/v1/music/albums/{id}/cover` returns the album cover.

Both responses are images and support caching and `ETag`.

## Errors

API errors use this shape:

```json
{"error": "…"}
```

Common codes:

| Code | Meaning |
|---|---|
| `400` | Invalid parameters or request body. |
| `401` | Missing, invalid or expired session. |
| `403` | The account does not have the required role. |
| `404` | The resource does not exist or is not visible to the account. |
| `409` | Conflict with the current state. |
| `429` | Too many attempts; respect `Retry-After`. |
| `502` | An external dependency, such as TMDB, failed. |
| `503` | FFmpeg or another required service is unavailable. |

An unsupported playback container returns `415`.

## Library access

The server already filters by the libraries assigned to the account and by its role. The app does not need to send library identifiers to hide content: anything not allowed returns `404`, just like a missing resource. This also applies to playback, downloads and thumbnails.

## curl example

Replace `https://cinecat.example`, the username and the password with your server's values.

```bash
BASE=https://cinecat.example

TOKEN=$(curl -sS "$BASE/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"ana","password":"…","device_name":"Living room TV","client":"androidtv"}' \
  | jq -r .token)

curl -sS "$BASE/api/v1/items?kind=movie&sort=title&limit=20" \
  -H "Authorization: Bearer $TOKEN"

curl -sS "$BASE/api/v1/items/42/play?caps=mkv,mp4,h264,aac,ac3" \
  -H "Authorization: Bearer $TOKEN"
```

The last response contains a `url` such as `/api/v1/stream/42?exp=…&sig=…`. Pass it to the player with the server base in front; the player must not send the `Bearer` token.

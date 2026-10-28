# API for native clients

[Español](../es/api.md) · [Back to README](../../README.en.md) · [Architecture and security](architecture.md)

Cinecat exposes an HTTP API so Android and Android TV apps can sign in, browse the library and play content without relying on the web client. The native apps are not in the repository yet; this is the contract they must use when implemented. This page summarises the contract and the rules native clients must follow; the exact details of every operation are in the OpenAPI document.

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

The current session and the user's other sessions are listed by `GET /api/v1/auth/sessions`; `DELETE /api/v1/auth/sessions/{id}` closes a specific session. Administrators can list and revoke an account's sessions through the `/api/v1/users/{id}/sessions` routes.

Login limits failed attempts per user and IP. When the limit is exceeded it returns `429` with a `Retry-After` header; the app must wait that long before retrying.

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

## Playback

`GET /api/v1/items/{id}/play` prepares playback and accepts these query parameters:

| Parameter | Use |
|---|---|
| `caps` | Device capabilities, comma-separated. |
| `audio` | Index of the selected audio track. |
| `quality` | `auto`, `original`, `480`, `720` or `1080`. |
| `subtitle` | Subtitle identifier; empty to disable it. |

The apps declare in `caps` what their device plays: containers `mkv` and `mp4`; video `h264`, `h264hi10`, `hevc`, `hevc10`, `av1` and `vp9`; audio `aac`, `mp3`, `ac3`, `eac3`, `dts`, `truehd`, `opus` and `flac`; and `mch` if it decodes multichannel AAC. Each token means the device can decode that container, codec or format; `hevc10` implies `hevc`, and `mch` is only needed for AAC with more than two channels. For the server to treat the request as a native app, `caps` must include at least `mkv` or `mp4`. The full list and its effect are in [playback](playback.md#native-apps-android-and-android-tv).

The response includes `url`, `mode`, `expires_at`, `audio_tracks`, `selected_audio`, `qualities`, `selected_quality`, `subtitles` and `selected_subtitle`. `mode` is `direct` or `remux`; when it is not direct, `reason` explains why (`container`, `video` or `audio`).

The `/stream` URL is relative to the server, signed and valid for 8 hours. It does not need `Authorization`: prepend the server base and pass it unchanged to the player, such as ExoPlayer. Do not add custom headers or modify the signature.

Progress is stored with `PUT /api/v1/items/{id}/progress` and a body such as `{"position_seconds": 300, "duration_seconds": 1800}`; `DELETE` on the same route clears it. At 90 % of the duration the server marks the item as watched and removes progress. Watched state is controlled with `PUT` and `DELETE /api/v1/items/{id}/watched`.

## Images

On item pages, `poster_url` and `backdrop_url` are TMDB URLs that the app can load directly, without going through the API.

Clip thumbnails and music covers are served by the server and require the session:

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

The server already filters by the libraries assigned to the account and by its role. The app does not need to send library identifiers to hide content: anything not allowed returns `404`, just like a missing resource. This also applies to playback and thumbnails.

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

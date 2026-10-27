# Administration

[Español](../es/administracion.md) · [Back to README](../../README.en.md) · [User guide](user-guide.md)

Only users with the `admin` role can open `/admin`. The server checks the role on every request; the web app is not what protects the data.

## Screens

| Route | Screen | What it does |
|---|---|---|
| `/admin` | Summary | Server and web versions, last scan, movie/series/collection counts, and links to unidentified titles. |
| `/admin/libraries` | Libraries | Create and delete libraries, start a scan of all of them and follow its background progress. |
| `/admin/settings/scan` | Scan | Interval and scan at startup. |
| `/admin/settings/metadata` | Metadata | TMDB language and identification threshold. |
| `/admin/settings/ui` | Interface | Name, texts, menu, grid, order, collections, series, theme and accent. |
| `/admin/settings/playback` | Playback | Maximum height, CRF, preset and simultaneous transcodes. |
| `/admin/settings/system` | System | Update check, versions and the "Update now" panel. |
| `/admin/update` | — | JSON endpoint used by the System panel to follow progress; not a screen. |
| `/admin/users` | — | Does not exist yet. Navigation shows "Usuarios · Pronto". |

## Creating a library

1. Open **Administración → Bibliotecas**.
2. Under "Nueva biblioteca", enter a **Nombre** and the **Ruta en el servidor**.
3. Press **Crear biblioteca**. The path must exist and be a directory inside the container, for example `/media/Multimedia/Pelis`.
4. Press **Escanear todas las bibliotecas**. The scan continues in the background even if you close the page.

### Multiple folders per library

A library can group several folders, even on different disks. The first folder is kept as the main folder for compatibility; if it is removed, the next oldest becomes the main one. A library can be `video` (movies and series), `clips` (unidentified videos) or `music`; for now only `video` is scanned, and the other two types are skipped with a warning.

The server can add and remove folders and exposes a confined browser at `GET /api/v1/fs` (administrators only). The web screen for that browser will arrive in the next task. Until then, the path is typed by hand; the container mounts `CINECAT_MEDIA_PATH` whole and read-only at `/media`.

Folders are validated against `CINECAT_MEDIA_ROOTS`: a list of roots separated by `:` on Linux and `;` on Windows (default `/media`). A path outside those roots, missing, not a directory, repeated or overlapping another one is rejected. Removing a folder deletes its records from the database but does not touch files on disk.

**Protection against unmounted disks.** A scan only deletes entries from folders it traversed successfully. If a folder is unavailable, or is traversed without finding a single file although the database has entries for it (which is what an unmounted disk looks like, its mount point left as an empty folder), it keeps the entries, with their watched state and progress, and reports it in the scan result. Consequence: if you really empty a whole folder, its entries are not deleted automatically; remove the folder from the library to delete them.

## Scanning and identification

- The scan walks the libraries, ignores hidden files and folders and does not follow symlinks.
- It recognises `S01E02`, `1x02`, "Temporada N", "Episodio N" and formats such as `ChicagoFire720p4x17` or `MR_Robot_305` inside a season folder.
- Movies are identified with TMDB; without a TMDB key, OMDb is used only for movie synopsis and poster. With no key, files are indexed but not identified.
- `match_status`: `auto` (identified by the scan), `manual` (corrected by an administrator), `none` ("not this one") or empty (unidentified and retried).
- Series are grouped by their top-level folder. A manually corrected series is not touched by later scans; missing seasons or episodes are not filled automatically.
- Scan warnings appear in a collapsible list when it ends. A season error does not abort the rest.

## Correcting metadata

On a movie or series page, as an administrator, open **Corregir ficha**:

1. Adjust the **Título** and, optionally, the **Año**.
2. Press **Buscar en TMDB** and choose **Es esta**.
3. **No es ninguna** leaves the record empty and stops retrying.
4. **Volver a automático** clears the correction and retries identification from the filename.

Under **Cambiar imagen** you can set a poster or backdrop URL, choose one TMDB offers, or **Restaurar la de TMDB**. Manually set images are not overwritten by scans.

## Settings

Settings are stored in the database with per-type validation. They apply without a server restart, although some affect only the next operation: the interval affects the next schedule, the language affects new requests and quality affects new transcodes. Each setting shows its default value and can be reset.

| Key | Name | Description | Type and range / options | Default | Application |
|---|---|---|---|---|---|
| `scan.interval` | Intervalo de escaneo | How often the library scans itself. Use "0" to disable it; the minimum is 1 minute. | Go duration; `0` or minimum `1m`; no maximum | `6h` | Hot; reschedules the next scan. |
| `scan.on_startup` | Escanear al arrancar | Runs a scan one minute after server startup. | Boolean | `true` | Hot; only when the interval is not `0`. |
| `metadata.language` | Idioma de los metadatos | Preferred language for new TMDB synopses and records; it does not rewrite saved records. | `es-ES`, `es-MX`, `en-US` | `es-ES` | Hot; only affects new requests. |
| `metadata.match_threshold` | Umbral de identificación | Minimum confidence to accept a match automatically. | Decimal `0.6`–`0.95` | `0.75` | Hot; next identification. |
| `ui.server_name` | Nombre del servidor | Name shown in the interface. | Text 1–40 | `Cinecat` | Hot; next page. |
| `ui.tagline` | Eslogan | Short phrase next to the name in the header. Empty to hide it. | Text 0–60 | `Sesión continua` | Hot; next page. |
| `ui.portal_title` | Título del portal | Main title of the home page. | Text 1–60 | `Tu cine en casa` | Hot; next page. |
| `ui.portal_intro` | Texto del portal | Paragraph below the home page title. Empty to hide it. | Text 0–240 | `Una biblioteca personal con olor a palomitas, carteles pintados a mano y la sesión de siempre esperándote.` | Hot; next page. |
| `ui.nav_layout` | Menú de navegación | Position of the main menu: side on desktop or top. | `side` (left), `top` (top) | `side` | Hot; next page. |
| `ui.grid_columns` | Columnas de los listados | Posters per row in movies, series and collections. | Integer 4–8 | `6` | Hot; next page. |
| `ui.grid_rows` | Filas por página | Poster rows per page; the result count is columns × rows. | Integer 2–12 | `8` | Hot; next page. |
| `ui.row_columns` | Carátulas por fila en el portal | How many posters are visible at once in scrollable rows. | Integer 3–8 | `6` | Hot; next page. |
| `ui.default_sort` | Orden por defecto | Initial sort order for movies. | `title`, `year`, `added` | `title` | Hot; the web app still starts with Título. |
| `ui.show_collections` | Mostrar colecciones | Shows the collections section in the interface. | Boolean | `true` | Hot; hides the menu link. |
| `ui.show_series` | Mostrar series | Shows the series section in the interface. | Boolean | `true` | Hot; hides the menu link. |
| `ui.collection_min_items` | Mínimo para colecciones | Minimum number of movies to show an automatic collection. | Integer 1–10 | `2` | Hot; next automatic collection listing. |
| `ui.theme_default` | Tema por defecto | Initial theme when there is no saved preference. | `system`, `light`, `dark` | `system` | Hot; used when the browser has no preference. |
| `ui.accent` | Color de acento | Interface accent colour. | `granate`, `naranja`, `dorado` | `granate` | Hot; next page. |
| `playback.max_height` | Altura máxima | Maximum resolution for HLS output. | `480`, `720`, `1080` | `1080` | Hot; new transcodes. |
| `playback.crf` | Calidad (CRF) | Compression quality for transcoding; a lower value means more quality and size. | Integer 18–28 | `20` | Hot; new transcodes. |
| `playback.preset` | Velocidad de codificación | x264 preset used for transcoding. | `veryfast`, `faster`, `fast` | `veryfast` | Hot; new transcodes. |
| `playback.max_transcodes` | Transcodificaciones simultáneas | How many movies can be transcoded at once. More than the CPU can handle causes stalls when seeking. | Integer 1–8 | `2` | Hot; next process. |
| `system.check_updates` | Buscar actualizaciones | Checks for new versions; it implies an optional outgoing connection that the administrator must enable. | Boolean | `false` | Hot; enables periodic checking and the immediate check when saved. |

`scan.interval` may start with a different value when `CINECAT_SCAN_INTERVAL` is defined; the value stored in the database takes precedence over the environment default.

## Environment variables

Read by the server:

| Variable | Default | Purpose |
|---|---|---|
| `CINECAT_CONFIG` | empty | Path to a JSON configuration file. |
| `CINECAT_LISTEN_ADDR` | `:8080` | HTTP address and port. |
| `CINECAT_DB_PATH` | `cinecat.db` | SQLite file. |
| `CINECAT_TMDB_API_KEY` | empty | Enables TMDB; takes precedence over OMDb. |
| `CINECAT_OMDB_API_KEY` | empty | Enables OMDb only when TMDB is absent. |
| `CINECAT_SCAN_INTERVAL` | `6h` | Initial default for `scan.interval`. |
| `CINECAT_MEDIA_ROOTS` | `/media` | Allowed roots for library folders and the browser; separated by `:` on Linux and `;` on Windows. |
| `CINECAT_ADMIN_PASSWORD` | empty | Password for the `admin` created on first start; if absent, one is generated and printed once. |
| `CINECAT_RUN_DIR` | `/run/cinecat-update` | Shared directory for update requests. |

Docker also uses `CINECAT_MEDIA_PATH` (host path mounted at `/media`), `CINECAT_MEDIA_ROOTS` (roots visible inside the container), `CINECAT_API_PORT` (`8080`) and `CINECAT_WEB_PORT` (`8081`). `PUBLIC_API_URL` and `PROTOCOL_HEADER` belong to the web container. There are no variables for the FFmpeg path, CORS, users or HLS limits.

## Updating the server

- **Manual:** `./scripts/actualizar.sh` at the repository root. It runs `git pull --ff-only`, rebuilds and starts; the database and libraries are kept.
- **From the web:** Administration → System. "Buscar actualizaciones" asks the server; "Actualizar ahora" asks the host service to back up the database, update to the tag, check that it starts and roll back on failure.
- Install the service once with `./scripts/instalar-actualizador.sh`; it needs Docker accessible to the user, `git`, `sqlite3`, `curl`, `python3`, `flock` and linger. Details are in [updates](updates.md).
- The panel shows the phase, elapsed time and result; during the restart it is normal for the web app to be unavailable for a moment.

## Security limitations

- Login lockout is keyed by username, not IP: a known account can be temporarily locked.
- There are no per-library permissions: any authenticated user sees the whole catalogue.
- There is no screen to create users or revoke sessions.
- The server sends no CORS headers; the Apache deployment works by sharing the origin.
- `/media` is read-only: the application does not upload, move or delete files.

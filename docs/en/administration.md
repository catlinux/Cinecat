# Administration

[Español](../es/administracion.md) · [Back to README](../../README.en.md) · [User guide](user-guide.md)

Only users with the `admin` role can open `/admin`. The server checks the role on every request; the web app is not what protects the data.

## Screens

| Route | Screen | What it does |
|---|---|---|
| `/admin` | Summary | Server and web versions, last scan, movie/series/collection counts, and links to unidentified titles. |
| `/admin/libraries` | Libraries | Create libraries with a name, type and several folders, choose folders on the server, rename, add or remove folders, scan, delete and upload files. |
| `/admin/trash` | Trash | Restore or permanently delete what was moved to the trash, and empty expired entries. |
| `/admin/settings/scan` | Scan | Interval and scan at startup. |
| `/admin/settings/metadata` | Metadata | TMDB language and identification threshold. |
| `/admin/settings/ui` | Interface | Name, texts, menu, grid, order, collections, series, theme and accent. |
| `/admin/settings/library` | Library | Allow file changes, days in the trash and maximum upload size. |
| `/admin/settings/playback` | Playback | Maximum height, CRF, preset and simultaneous transcodes. |
| `/admin/settings/system` | System | Update check, versions and the "Update now" panel. |
| `/admin/update` | — | JSON endpoint used by the System panel to follow progress; not a screen. |
| `/admin/users` | — | Does not exist yet. Navigation shows "Usuarios · Pronto". |

## Creating a library

1. Open **Administración → Bibliotecas**.
2. Under "Nueva biblioteca", enter a **Nombre** and choose the **Tipo**: "Películas y series", "Vídeos (videoclips, conciertos…)" or "Música".
3. Type the first folder path or press **Elegir…** to browse the server folders. With **Añadir otra carpeta** you can add more; all of them are stored in the same library.
4. Press **Crear biblioteca**. The paths must exist and be directories inside the allowed roots; for example `/media/Multimedia/Pelis`.
5. On each library card you can **Renombrar**, **Añadir carpeta**, **Quitar** a folder, **Escanear** or **Borrar biblioteca**. To delete one you must type its name to confirm.

The scan continues in the background even if you close the page. The **Escanear** button on a card starts the same scan of all libraries.

### Multiple folders per library

A library can group several folders, even on different disks. The first folder is kept as the main folder for compatibility; if it is removed, the next oldest becomes the main one. A library can be `video` (movies and series), `clips` (unidentified videos) or `music`. All three types are scanned; Music tracks use their own tables and are never mixed with movies, series or videos.

### Vídeos libraries (unidentified)

A `clips` library walks its folders like a movie library, but identifies nothing: it does not query TMDB or OMDb, does not infer a year, does not create series or collections and makes no external metadata requests. The title comes from the file name, replacing `_` and `.` with spaces; for example, `2019-08-15_Cumple.de.Ana.mp4` is shown as "2019-08-15 Cumple de Ana".

Videos from these libraries are not mixed with movies in listings, searches, filters, recommendations, collections, portal counts or the administration summary. They keep watched state, progress, preferences and playback, and can appear in "Continue watching". The **Vídeos** section of the web app lets you browse their folders and subfolders, open a minimal detail page and play them. The Music type has its own scan and API; the Music web screen will arrive in a later task.

### Music libraries

A `music` library walks all its folders and reads audio tags with `ffprobe`: title, artist, album artist, album, track, disc, year and genre. The recommended structure is `Artist/Album/Track`; when tags are missing, the title comes from the file name (without a prefix such as `01 - `), the album from the folder and the artist from the parent folder. Compilations tagged "Various Artists", "VV.AA." or "Varios artistas" are unified as "Varios artistas".

The supported formats are MP3, FLAC, M4A, AAC, OGG, OGA, OPUS and WAV; WMA, APE and other formats are not indexed. No external service is queried: no MusicBrainz, no downloaded covers and no lyrics. A track that fails in `ffprobe` is still indexed with fallback values and the scan reports a warning.

The album cover comes from the first `cover`, `folder`, `front` or `album` file with a JPG, JPEG, PNG or WEBP extension in the track folder, case-insensitively. If there is none, the embedded image of the first track that has one is extracted with FFmpeg and stored in `cache/covers`, next to the database. Covers and thumbnails can be deleted safely because they are generated again. FFmpeg and ffprobe must be installed in the container.

The scan does not inspect a track again when its size and modification time have not changed, and limits concurrent `ffprobe` reads to four. Removing a folder or a library also removes its tracks, albums and artists that are left empty, but never the files on disk.

Thumbnails are generated the first time they are viewed and stored in `cache/thumbs`, next to the database and inside the persistent volume. They can be deleted without risk: they will be generated again. The server needs FFmpeg and ffprobe installed in the container.

The **Elegir…** button opens a server folder browser. It shows the allowed roots, lets you go down into subfolders, go up with **Subir**, move through the breadcrumbs and confirm with **Elegir esta carpeta**. It only lists directories, does not follow links that leave the roots and limits the listing to 1000 entries. You can also type the path by hand without JavaScript. The container mounts `CINECAT_MEDIA_PATH` whole at `/media`, read-only unless uploads are enabled (see "Uploading files and trash").

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
| `library.allow_file_changes` | Permitir cambiar archivos | Allows uploading files and moving them to the trash; requires mounting the library read-write. | Boolean | `false` | Hot. |
| `trash.retention_days` | Días en la papelera | How long entries are kept before being purged for good. | Integer 1–90 | `14` | Hot; next daily purge. |
| `library.upload_max_gb` | Tamaño máximo de subida (GB) | Maximum size of each uploaded file. | Integer 1–500 | `50` | Hot; next upload. |
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
| `CINECAT_TRUSTED_PROXIES` | loopback and private networks | Networks from which `X-Forwarded-For` is accepted; comma-separated CIDR or IP list. |
| `CINECAT_FILE_UID` / `CINECAT_FILE_GID` | empty | Numeric owner of what the server creates on the disks (uploads, folders, trash). |

Docker also uses `CINECAT_MEDIA_PATH` (host path mounted at `/media`), `CINECAT_MEDIA_ROOTS` (roots visible inside the container), `CINECAT_API_PORT` (`8080`), `CINECAT_WEB_PORT` (`8081`) and `CINECAT_MEDIA_MODE` (`ro`; `rw` to upload and use the trash). The web container uses `ADDRESS_HEADER=X-Forwarded-For` and `XFF_DEPTH=1` (number of trusted proxies in front of the web app) and `BODY_SIZE_LIMIT=Infinity` (for uploads); `PUBLIC_API_URL`, `PROTOCOL_HEADER` and `INTERNAL_API_URL` (the server's internal address, `http://servidor:8080`, used to forward uploads without going through the internet) also belong to the web container. There are no variables for the FFmpeg path, CORS, users or HLS limits.

Each user has their own **Favorites** and **Watchlist**; they are not shared between accounts.

## Uploading files and trash

Disabled by default. Two things are needed: mount the library read-write (`CINECAT_MEDIA_MODE=rw` in the Debian `.env`, then bring the containers up again) and enable **Permitir cambiar archivos** in Administration → Library. With either one off, the web app shows the reason and does not touch the disk. With `rw`, any Cinecat administrator can write to your disks: protect that account well.

- **Upload:** Administration → Libraries → **Subir archivos** on a movies-and-series or videos library. Choose the destination folder (and optionally a new folder inside it), select one or more videos or subtitles (`.srt`, `.vtt`, `.ass`, `.ssa`, `.sub`, `.idx`) and press **Subir**. Each file is first written as `.name.part` and renamed when done; an existing file is never overwritten. A scan starts afterwards. Limit: **Tamaño máximo de subida (GB)** (50 by default); the upload is refused when there is no room for the file plus a 1 GB margin. Do not close the page until it finishes.
- **Move to trash:** on the item page, **Mover a la papelera**. Nothing is deleted: the video and its subtitles (or its folder, for a movie or video whose folder holds no other videos) are moved to `.cinecat-papelera`, at the root of the library folder and on the same disk. The entry disappears from listings.
- **Trash:** Administration → Trash. **Restaurar** puts the files back and restores the entry with its watched state, progress, lists, collections and preferences (if the original path is taken, it does nothing and says so). **Eliminar definitivamente** deletes from disk. Anything older than **Días en la papelera** (14 by default) is deleted automatically every day.
- **Owner:** the container runs as root. Set `CINECAT_FILE_UID` and `CINECAT_FILE_GID` (the ones shown by `ls -ln /media`) so uploads and the trash belong to your user and group.
- Video only: music cannot be uploaded or moved to the trash from Cinecat.

## Updating the server

- **Manual:** `./scripts/actualizar.sh` at the repository root. It runs `git pull --ff-only`, rebuilds and starts; the database and libraries are kept.
- **From the web:** Administration → System. "Buscar actualizaciones" asks the server; "Actualizar ahora" asks the host service to back up the database, update to the tag, check that it starts and roll back on failure.
- Install the service once with `./scripts/instalar-actualizador.sh`; it needs Docker accessible to the user, `git`, `sqlite3`, `curl`, `python3`, `flock` and linger. Details are in [updates](updates.md).
- The panel shows the phase, elapsed time and result; during the restart it is normal for the web app to be unavailable for a moment.

## Security limitations

- Login lockout counts failures per user+IP pair and adds a per-IP limit, but it still lives in process memory and depends on Apache and the web app forwarding the real IP correctly.
- There are no per-library permissions: any authenticated user sees the whole catalogue.
- There is no screen to create users or revoke sessions.
- The server sends no CORS headers; the Apache deployment works by sharing the origin.
- With `CINECAT_MEDIA_MODE=rw`, an administrator can upload and move to the trash; permanent deletion only affects `.cinecat-papelera`.

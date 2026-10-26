# Fichas en castellano, colecciones, series y navegación de la biblioteca

Estado (2026-10-26): T1, T2 y T3a hechas (en local, sin push). **Siguiente: T3b.**

Orden: Hito 1 (T3b → T3c → T3d → ▶ → T4 → T5 → T6 → D1) → Hito 1b (A1 → A2 → A3 → ▶ → A4 → ▶) → Hito 2 (T7 → T8 → T9 → ▶) → Hito 3 (T10 → T11 → T12 → T13 → D2).

## Cómo ejecutar este plan (sesión orquestadora, Sonnet)

- Cada tarea `Tn` es una ejecución de AgentRelay (salvo las marcadas «sesión»). Copia de la tarea: objetivo, archivos, contratos y criterios; añade a `context` las convenciones de abajo. Una tarea = un commit tuyo tras revisarla.
- Antes de cada tarea: `git status --short` vacío. JSON de la tarea en un archivo **fuera del repo** (`$env:TEMP\tarea.json`) y `agentrelay.cmd run $env:TEMP\tarea.json` (en segundo plano: tarda 5-15 min).
- **Validación de AgentRelay:** su ejecutor de validación usa `cmd.exe`. Pon solo órdenes válidas en cmd: `C:\Users\msanc\tools\go1.26.9\bin\go.exe vet ./... && C:\Users\msanc\tools\go1.26.9\bin\go.exe test ./...` y, en tareas web, `cd web && npm run check && npx vitest run`. Nada de `export` ni `$env:`.
- **Revisión (siempre tú):** `agentrelay show <id>`, diff completo, y en Git Bash: `export PATH="/c/Users/msanc/tools/go1.26.9/bin:$PATH"; gofmt -l internal cmd` (vacío), `go vet ./...`, `go test ./...`; web: `cd web && npm run check && npx vitest run`. El ejecutor a veces estropea acentos en comentarios: `grep -n "[a-zA-Z]?[a-zA-Z]" <archivos>`. Código de salida 2 de `agentrelay run` = pendiente de revisión, no fallo. Si solo falla su validación y la tuya pasa: `agentrelay review <id> --decision accept --force`.
- Documentación en el mismo commit que el cambio: `api/openapi.yaml` en toda tarea que toque la API; `docs/es` y `docs/en` sincronizados; `CHANGELOG.md` y `CHANGELOG.en.md` en `[Sin publicar]`; `TODO.md`/`TODO.en.md`. **Versión** (`VERSION`) solo en la tarea que cierra cada hito (indicado abajo), moviendo `[Sin publicar]` a la versión con fecha.
- Si una tarea topa con una decisión de diseño no prevista aquí: para, anótala en este archivo bajo la tarea y deja que la resuelva una sesión con Opus.
- Marca cada tarea como ✅ en este archivo al hacer su commit.
- **Desplegar a menudo:** el usuario quiere ver las mejoras en la web conforme llegan. En cada punto `▶ Desplegar` (abajo), y en general tras cada tarea que cambie algo visible, propón push + despliegue; hazlo solo con su aprobación explícita para ese push (regla global). Despliegue: `ssh 192.168.1.150 'cd /opt/cinecat && ./scripts/actualizar.sh'`, y comprueba después en `https://video.warcrafted.com` que la versión mostrada es la nueva y que lo cambiado funciona (curl con login, como en `docs`/handoff) antes de decirle que está.

Convenciones para el `context` de cada tarea: Go 1.22+ (módulo `github.com/catlinux/cinecat`), SQLite con migraciones `.sql` en `internal/store/migrations` (tablas STRICT); web SvelteKit + TypeScript en `web/`, dirección visual «Marquesina» (README: cine de barrio, serif Fraunces, nada de estética genérica de IA); toda la interfaz y los comentarios en castellano; comentarios solo para el porqué no obvio; sin dependencias nuevas salvo que la tarea lo diga.

## Contexto y decisiones ya tomadas

- Producción (copia de la BD del 2026-10-26): 886 películas (38 % con cartel, fichas de OMDb en inglés) y 766 episodios de 26 series sin metadatos. Los archivos tienen títulos en castellano y casi nunca año.
- Fuente: **TMDB** en `es-ES` (clave v3 ya en el `.env` del Debian; `docker-compose.yml` ya la pasa). OMDb solo como respaldo sin clave de TMDB.
- La identificación es **automática** en cada escaneo; la corrección manual es para el resto. `match_status`: `NULL` sin identificar (se reintenta en cada escaneo), `'auto'`, `'manual'` (corrección del usuario: ningún escaneo la toca) y `'none'` («no es ninguna»: no se reintenta).
- Navegación: tres categorías en la cabecera, **Películas · Colecciones · Series**, todas paginadas (48 por página, `?page=N` en la URL).
- Colecciones: se crean solas a partir de las sagas de TMDB (`belongs_to_collection`) y el usuario puede crear, renombrar, editar, añadir o quitar películas, reordenar y borrar. Una colección automática solo se muestra si tiene ≥ 2 películas en la biblioteca. Los cambios del usuario sobre una colección automática se respetan en los escaneos siguientes.
- Series: una ficha por carpeta de serie, sus temporadas y sus episodios, con títulos y sinopsis en castellano.

## Hecho

- ✅ **T1** — Cliente TMDB (`internal/metadata/tmdb.go`): `SearchMovies`, `MovieDetails`, `MovieCollection`, `SearchTV`, `TVDetails`, `EpisodeDetails`; limitador a 20 peticiones/s; reintento tras 429; caché en disco por petición; respaldo de sinopsis es-ES → es-MX → es → en-US con `OverviewLanguage`.
- ✅ **T2** — Identificación (`internal/library/match.go`): `IdentifyMovie(ctx, MovieSearcher, title, year) (Candidate, score, ok, err)`, umbral `MatchThreshold` = 0.75. Ajustada con las 870 películas reales: 854 identificadas (98 %), aproximadas revisadas a mano. Test con respuestas reales grabadas (`internal/library/testdata/match_fixture.json`, 53 casos): **no cambiar `match.go` sin regenerar y revisar el fixture**. Fallos conocidos (corrección manual): «MEGAN», «Desafío total - Remake», Star Wars IV–VI, erratas en el nombre.
- ✅ **T3a** — Migración `0004_media_item_tmdb.sql` (columnas `original_title, imdb_id, tagline, genres, runtime_minutes, rating, directors, cast_names, backdrop_url, overview_language, metadata_source, match_status, match_score`); escaneo con TMDB (`Scanner.WithTMDB`, `enrichMovieTMDB`); `ApplyMovieDetails(ctx, db, itemID, *Details, status, score)` y `ClearMovieMetadata(ctx, db, itemID, status)` exportadas; caché de TMDB en `<dir de la BD>/cache/tmdb` (volumen `/data`).

## Hito 1 — Fichas de películas, corrección y navegación (→ 0.11.0)

### T3b — Escaneo en segundo plano, con estado y automático
- Por qué: el primer escaneo con TMDB hace ~5000 peticiones (4-5 min); hoy va ligado a la petición HTTP y Apache lo cortaría a medias. Y el usuario quiere que las películas nuevas se identifiquen solas.
- Archivos: `internal/library/scanner.go` (o nuevo `internal/library/scanjob.go`), `internal/library/http.go`, `internal/library/http_test.go`, `cmd/cinecatd/main.go`, `internal/config/config.go`, `docker-compose.yml`, `.env.deploy.example`, `api/openapi.yaml`.
- Contrato:
  - Tipo `ScanJob` con mutex: `Start(ctx) bool` lanza `Scan` en una goroutine con un contexto propio del servicio (no el de la petición) y devuelve false si ya hay uno en marcha; `Status() ScanStatus`.
  - `ScanStatus{Running bool, StartedAt, FinishedAt *time.Time, Files, Upserted, Identified, Unidentified int, Warnings []string, Error string}` (en JSON `snake_case`); guarda el último resultado.
  - `POST /api/v1/libraries/scan` (admin) → `202 {"started": true}` o `409 {"error": "ya hay un escaneo en curso"}`. `GET /api/v1/libraries/scan` (admin) → `ScanStatus`.
  - Escaneo automático: variable `CINECAT_SCAN_INTERVAL` (duración Go, por defecto `6h`; `0` lo desactiva) en `config.go`, `docker-compose.yml` y `.env.deploy.example`. Un escaneo 1 minuto después de arrancar y luego cada intervalo, usando el mismo `ScanJob` (si ya hay uno en marcha, se salta).
- Criterios: tests de `ScanJob` (dos `Start` seguidos → el segundo false; `Status` refleja el resultado) y de los dos endpoints; `go test ./...` en verde.
- Esfuerzo: alto.

### T3d — Versión visible en servidor y web
- Archivos: `cmd/cinecatd/main.go` (ya hay una versión: se registra «cinecatd 0.9.1 escuchando» — ver cómo se inyecta), `internal/library/http.go` o nuevo `internal/system/`, `api/openapi.yaml`, `web/src/routes/+layout.svelte` (pie), `web/vite.config.ts` o equivalente, `web/Dockerfile`/`Dockerfile`, tests.
- `GET /api/v1/system/version` (sin sesión) → `{"server": "0.11.0", "commit": "<hash corto o vacío>", "built_at": "..."}`. La versión sale de `VERSION` en tiempo de compilación (`-ldflags -X`), nunca escrita a mano en el código.
- La web muestra en el pie «Cinecat 0.11.0» (versión del servidor) y, si la de su propio build (también desde `VERSION`, inyectada en la compilación de Vite) no coincide con la del servidor, un aviso «La web y el servidor tienen versiones distintas: recarga o actualiza». Resolver cómo llega `VERSION` al build de la web en Docker (contexto de build; si no lo incluye, `ARG`).
- `web/package.json` deja de llevar su propia versión independiente: alinearla con `VERSION` o documentar que la autoridad es `VERSION`.
- Comprobación de actualizaciones: se hace en A3 (panel), no aquí.
- Esfuerzo: medio.

▶ **Desplegar** tras T3b+T3c+T3d: el usuario ya verá carteles y sinopsis en castellano en la cartelera actual, el escaneo con progreso y la versión en el pie.

### T3c — Panel de bibliotecas con el escaneo en segundo plano
- Archivos: `web/src/routes/admin/libraries/+page.svelte`, `+page.server.ts`, `admin-libraries.test.ts`, `web/src/lib/api.ts`.
- El botón «Escanear» llama al POST y muestra «Escaneo en curso…» consultando `GET /api/v1/libraries/scan` cada 3 s hasta `running=false`; entonces muestra archivos, identificadas, sin identificar y avisos (los avisos en una lista plegable). Con 409, mostrar que ya hay uno en curso y seguir consultando.
- Esfuerzo: medio.

### T4 — API: ficha completa, filtros, búsqueda y corrección de películas
- Archivos: `internal/library/http.go`, `internal/library/http_test.go`, `api/openapi.yaml`.
- `mediaItemDTO` con los campos nuevos: `original_title, imdb_id, tagline, genres[], runtime_minutes, rating, directors[], cast[]` (la columna es `cast_names`), `backdrop_url, overview_language, match_status, match_score` (arrays vacíos si NULL).
- `GET /api/v1/items`: nuevos parámetros `kind` (`movie`|`episode`), `q` (título analizado u `original_title`, sin distinguir mayúsculas, `LIKE`), `unmatched=true` (`kind='movie' AND (match_status IS NULL OR match_status='none')`), `sort` (`title` por defecto — `COALESCE` del título de TMDB si lo hay… ver nota —, `year`, `added`=id desc). Sigue devolviendo `total`.
  - Nota: hoy `title` es el título analizado del archivo. Añadir en la migración `0005` (en esta misma tarea) la columna `display_title TEXT` que `ApplyMovieDetails` rellena con `Details.Title` (el título en castellano de TMDB) y `ClearMovieMetadata` vacía; el DTO expone `title` = `COALESCE(display_title, title)` y `file_title` = el analizado. Actualizar `ApplyMovieDetails`/`ClearMovieMetadata` y sus tests.
- `GET /api/v1/items/{id}/candidates?q=&year=` (admin): `q` por defecto = título analizado; llama a `SearchMovies` y devuelve `Candidate[]` (con `poster_url` w185).
- `PUT /api/v1/items/{id}/match` (admin): `{"tmdb_id": 123}` → `MovieDetails` + `ApplyMovieDetails(..., "manual", 1)`; `{"tmdb_id": null}` → `ClearMovieMetadata(..., "none")`. `DELETE /api/v1/items/{id}/match` → `ClearMovieMetadata(..., nil)` y reidentificar en el momento con `IdentifyMovie` (devuelve el ítem resultante).
- La API necesita el cliente TMDB: `NewAPI` recibe la interfaz `TMDBClient` (puede ser nil: entonces los endpoints de corrección responden 503 «TMDB no está configurado»).
- Criterios: tests con TMDB falso para cada endpoint y filtro; un usuario no admin recibe 403 en los de corrección.
- Esfuerzo: alto.

### T5 — Web: cabecera con categorías, cartelera paginada y búsqueda
- Archivos: `web/src/routes/+layout.svelte`, `web/src/routes/+page.svelte`, `web/src/routes/+page.server.ts`, `web/src/lib/components/Pagination.svelte` (nuevo), `web/src/lib/api.ts`, tests.
- Cabecera: **Películas** (`/`), **Colecciones** (`/collections`), **Series** (`/series`) y los enlaces de admin ya existentes; la categoría activa marcada. Colecciones y Series pueden enlazar a páginas «próximamente» hasta T7/T9.
- `/`: solo `kind=movie`, 48 por página, `?page=N` (1 por defecto; fuera de rango → última), buscador por título (`?q=`), orden (Título · Año · Recientes, `?sort=`), y para admin el filtro «Sin identificar (N)» (`?unmatched=1`, N con una petición `limit=1`). `Pagination.svelte`: anterior/siguiente, primera, última y números alrededor de la actual; enlaces normales (funciona sin JavaScript).
- Criterios: tests del cálculo de páginas y de los parámetros de la petición; `npm run check` sin errores.
- Esfuerzo: medio.

### T6 — Web: ficha completa y «Corregir ficha»
- Archivos: `web/src/routes/items/[id]/+page.svelte`, `+page.server.ts`, `web/src/lib/components/MatchDialog.svelte` (nuevo), `web/src/lib/api.ts`, tests.
- Ficha: título en castellano, título original si difiere, año · duración (h min) · géneros · nota (una cifra decimal), lema, sinopsis, dirección, reparto, fondo `backdrop_url` discreto tras la cabecera. Aviso discreto «Sinopsis no disponible en castellano» si `overview_language` no empieza por `es`. Si la película pertenece a una colección (cuando exista T7), enlace a ella.
- Admin: «Corregir ficha» abre `MatchDialog`: buscador (título precargado con `file_title`, año opcional), lista de candidatos (cartel, título, título original, año, primera línea de sinopsis) → «Es esta»; botones «No es ninguna» y «Volver a automático». Tras cada acción, recargar la ficha. Mostrar el nombre del archivo (`file_title`) para que se vea qué se está corrigiendo.
- Criterios: tests de las acciones del servidor (form actions) con la API simulada.
- Esfuerzo: alto. Referencias: README (dirección visual).
- **Cierra el hito 1:** `VERSION` 0.11.0 y CHANGELOG (Añadido: fichas TMDB en castellano, corrección manual, sin identificar, escaneo en segundo plano y automático, cartelera paginada con búsqueda. Cambiado: OMDb pasa a respaldo).

### D1 — Despliegue y medición del hito 1 (sesión)
- Con aprobación del usuario: push y `actualizar.sh`. Lanzar el escaneo desde el panel y esperar a que acabe.
- Medir con una copia de la BD (`docker cp cinecat-servidor-1:/data/cinecat.db /tmp/x.db` y `sqlite3`): % de películas con `match_status='auto'`, lista de las sin identificar para el usuario. Revisar 40 al azar (`display_title` frente a la ruta): ninguna con la ficha de otra película. Esperado ≈ 98 % (lo medido en T2).

## Hito 1b — Panel de administración (→ 0.11.x/0.12.0 según encaje)

El usuario quiere un panel de administración bien estructurado para configurar todo lo posible de la interfaz y las opciones. Hoy hay `/admin/libraries` y «Mi cuenta» sueltos.

### A1 — Ajustes guardados en la base de datos
- Archivos: `internal/store/migrations/0005b_settings.sql` (numerar tras la última existente), `internal/settings/settings.go` (nuevo, +test), `internal/settings/http.go` (+test), `cmd/cinecatd/main.go`, `api/openapi.yaml`.
- Tabla `settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)` (valor JSON). Registro tipado en Go de cada ajuste: clave, tipo, valor por defecto, validación (rango/opciones), sección y descripción en castellano. Las variables de entorno siguen existiendo como valor por defecto inicial; lo guardado en el panel manda.
- Ajustes iniciales (cada uno lo consume la parte correspondiente, leyendo en caliente sin reiniciar):
  - Escaneo: `scan.interval` (sustituye a `CINECAT_SCAN_INTERVAL` como valor efectivo), `scan.on_startup`.
  - Metadatos: `metadata.language` (`es-ES` por defecto; `es-MX`, `en-US`), `metadata.match_threshold` (0.6–0.95, por defecto 0.75).
  - Interfaz: `ui.page_size` (24/48/96), `ui.default_sort` (título/año/recientes), `ui.show_collections`, `ui.show_series`, `ui.collection_min_items` (por defecto 2), `ui.server_name` (título en la cabecera, por defecto «Cinecat»).
  - Reproducción: `playback.max_height` (480/720/1080), `playback.crf` (18–28, por defecto 20), `playback.preset` (veryfast/faster/fast).
  - Sistema: `system.check_updates` (por defecto **false**: es una conexión saliente y las reglas del proyecto las prohíben salvo a las fuentes de metadatos; solo con el usuario activándola).
- API: `GET /api/v1/settings` (admin) → secciones con sus ajustes (valor, defecto, tipo, opciones, descripción); `PATCH /api/v1/settings {clave: valor, ...}` con validación (400 con el error por clave); `GET /api/v1/settings/public` (con sesión) → solo los `ui.*` que necesita la web.
- Esfuerzo: alto.

### A2 — Conectar los ajustes a cada parte
- Archivos: `internal/library/scanner.go`/`scanjob.go`, `internal/library/match.go` (**solo** leer el umbral de un parámetro con el valor actual por defecto; el fixture de T2 debe seguir pasando con 0.75), `internal/playback/hls.go` (CRF, preset y altura máxima en `hlsTranscodeArgs`: el filtro `scale` pasa a `min(<altura>, ih)` manteniendo dimensiones pares), `internal/metadata/tmdb.go` (idioma).
- Criterios: tests que cambian un ajuste y comprueban el efecto (argumentos de FFmpeg, intervalo, idioma de la petición).
- Esfuerzo: alto. **Ojo con `internal/playback`:** los tests con FFmpeg real se ejecutan en el contenedor (ver TODO.md); no aceptar sin pasarlos allí.

### A3 — Web: panel de administración por secciones
- Archivos: `web/src/routes/admin/+layout.svelte` (nuevo: menú lateral), `web/src/routes/admin/+page.svelte` (resumen), `web/src/routes/admin/settings/[section]/+page.svelte|.server.ts`, mover `/admin/libraries` dentro del menú, tests.
- Secciones del menú: **Resumen** (versión del servidor y de la web, estado del último escaneo, nº de películas/series/colecciones, sin identificar con enlace al filtro, espacio de la caché de TMDB y de HLS si se expone), **Bibliotecas** (lo actual + escaneo con progreso de T3c), **Escaneo**, **Metadatos**, **Interfaz**, **Reproducción**, **Usuarios** (A4), **Sistema** (versión; «Comprobar actualizaciones» si está activado: compara con `https://raw.githubusercontent.com/catlinux/Cinecat/main/VERSION` y muestra «Estás al día» o «Hay una versión nueva: X.Y.Z», más el comando de actualización `./scripts/actualizar.sh`).
- Los formularios se generan a partir de la descripción de `GET /api/v1/settings` (tipo → control), con «Restablecer valor por defecto» por ajuste.
- La cabecera y la cartelera usan `GET /api/v1/settings/public` (nombre, tamaño de página, orden, categorías visibles).
- Esfuerzo: alto. Referencias: README (dirección visual).

### A4 — Usuarios
- Hoy no hay forma de crear usuarios (ver TODO.md). Archivos: `internal/auth/http.go` (+test), `api/openapi.yaml`, `web/src/routes/admin/users/...`.
- Admin: listar, crear (nombre, contraseña inicial, rol), cambiar rol, restablecer contraseña, borrar (no a sí mismo ni al último admin).
- Esfuerzo: medio.

▶ **Desplegar** tras A3 y tras A4.

## Hito 2 — Colecciones (→ 0.12.0)

### T7 — Modelo y creación automática de colecciones
- Archivos: `internal/store/migrations/0006_collections.sql`, `internal/metadata/tmdb.go` (+test), `internal/library/collections.go` (nuevo, +test), `internal/library/scanner.go`.
- `MovieDetails` añade `Collection *CollectionRef{TMDBID int, Name, PosterURL, BackdropURL string}` desde `belongs_to_collection` (ya viene en la respuesta de `/movie/{id}`; el nombre llega en es-ES, p. ej. «Harry Potter - Colección»).
- Tablas:
  - `collections (id INTEGER PK, name TEXT NOT NULL, overview TEXT, poster_url TEXT, backdrop_url TEXT, tmdb_collection_id INTEGER UNIQUE, source TEXT NOT NULL CHECK (source IN ('tmdb','manual')), edited INTEGER NOT NULL DEFAULT 0, deleted INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT (...))`.
  - `collection_items (collection_id INTEGER REFERENCES collections ON DELETE CASCADE, media_item_id INTEGER REFERENCES media_items ON DELETE CASCADE, position INTEGER, excluded INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (collection_id, media_item_id))`.
- Reglas (en `collections.go`, llamadas desde `ApplyMovieDetails` cuando `Details.Collection != nil`):
  - Buscar la colección por `tmdb_collection_id`; si no existe, crearla (`source='tmdb'`). Si existe con `deleted=1` (el usuario la borró), no hacer nada. Si `edited=0`, actualizar nombre, cartel y fondo desde TMDB.
  - Añadir la película (`INSERT OR IGNORE`); si ya existe con `excluded=1` (el usuario la quitó), no tocarla.
  - Orden en colecciones de TMDB: por año de la película (posición NULL). En las manuales, `position`.
  - Al cambiar la identificación de una película (corrección manual o «no es ninguna»), quitarla de las colecciones `source='tmdb'` a las que ya no pertenece.
- Rellenar las películas ya identificadas: las fichas están en caché, así que basta con que el primer escaneo tras la migración vuelva a llamar a `MovieDetails` para las `auto`/`manual` sin colección comprobada. Añadir la columna `media_items.collection_checked INTEGER DEFAULT 0` y procesarlas en el escaneo.
- Criterios: tests de cada regla (creación, colección borrada que no reaparece, película excluida que no vuelve, `edited` que conserva el nombre).
- Esfuerzo: alto.

### T8 — API de colecciones
- Archivos: `internal/library/collections_http.go` (nuevo), `internal/library/collections_http_test.go`, `internal/library/http.go` (registro de rutas), `api/openapi.yaml`.
- `GET /api/v1/collections?page_size&offset&q=` → `{collections: [{id, name, poster_url, item_count, source}], total}`; solo `deleted=0` y (`source='manual'` o ≥ 2 películas no excluidas). Cartel: el de la colección o, si no tiene, el de su primera película.
- `GET /api/v1/collections/{id}` → colección + películas en orden (DTO de ítem).
- Admin: `POST /api/v1/collections {name, overview?}` (manual); `PATCH /api/v1/collections/{id} {name?, overview?, poster_url?}` (pone `edited=1`); `DELETE /api/v1/collections/{id}` (manual → borrar; tmdb → `deleted=1`); `POST /api/v1/collections/{id}/items {media_item_id}`; `DELETE /api/v1/collections/{id}/items/{itemId}` (manual → borrar fila; tmdb → `excluded=1`); `PUT /api/v1/collections/{id}/order {media_item_ids: [...]}` (pasa a usar `position`).
- `GET /api/v1/items/{id}` incluye `collections: [{id, name}]`.
- Esfuerzo: medio.

### T9 — Web de colecciones
- Archivos: `web/src/routes/collections/+page.svelte|.server.ts`, `web/src/routes/collections/[id]/+page.svelte|.server.ts`, componentes que hagan falta, `web/src/lib/api.ts`, tests.
- `/collections`: rejilla de carteles paginada (48) con nombre y número de películas; buscador. Admin: «Nueva colección».
- `/collections/[id]`: cabecera (fondo, nombre, descripción) y películas en orden. Admin: editar nombre, descripción y cartel (elegir entre los carteles de sus películas o pegar una URL), añadir película (buscador sobre `GET /api/v1/items?kind=movie&q=`), quitar, subir o bajar en el orden, y borrar la colección con confirmación.
- La ficha de película (T6) enlaza a sus colecciones.
- **Cierra el hito 2:** `VERSION` 0.12.0 y CHANGELOG (o la siguiente menor si el hito 1b ya la usó).
- Esfuerzo: alto.

▶ **Desplegar** tras T9.

## Hito 3 — Series (→ 0.13.0)

### T10 — Analizador: serie por carpeta y episodios mal reconocidos
- Archivos: `internal/library/nameparser.go`, `internal/library/nameparser_test.go`.
- En una biblioteca de series, el nombre de la serie es la carpeta de primer nivel (`folderTitle`), no el nombre del archivo: «The Big Bang theory/Temporada 3/Big Bang - 3x12.avi» → serie «The Big Bang theory». Exponer también esa carpeta (`Media.SeriesFolder`).
- Dentro de una carpeta de temporada, reconocer `NxMM` pegado a otras letras («ChicagoFire720p4x17») y el número de 3 cifras cuya primera cifra es la temporada de la carpeta («MR_Robot_305_DivxTotaL» en «Temporada 3» → 3x05). Así los 16 archivos de `Series/` que hoy son películas pasan a episodios.
- Tests con estas rutas reales: `Chicago fire/Temporada 4/ChicagoFire720p4x17 [www.torrentlocura.com].mkv`, `Mr Robot/Temporada 3/MR_Robot_305_DivxTotaL.avi`, `The Big Bang theory/Temporada 3/Big Bang - 3x12.avi`, `Juego de tronos/Temporada 1/...` (las variantes de mayúsculas acaban en la misma carpeta).
- Esfuerzo: medio.

### T11 — Modelo, identificación y escaneo de series
- Archivos: `internal/store/migrations/0007_series.sql`, `internal/library/series.go` (nuevo, +test), `internal/library/match.go` (solo añadir `IdentifySeries`; **no tocar** `IdentifyMovie`), `internal/library/scanner.go`.
- Tablas: `series (id, library_id REFERENCES libraries ON DELETE CASCADE, folder TEXT NOT NULL, title TEXT NOT NULL, display_title, original_title, year, overview, overview_language, poster_url, backdrop_url, genres, rating, tmdb_id, metadata_source, match_status, match_score, UNIQUE (library_id, folder))`; `series_seasons (series_id REFERENCES series ON DELETE CASCADE, season INTEGER, name, overview, poster_url, PRIMARY KEY (series_id, season))`; en `media_items`: `series_id INTEGER REFERENCES series ON DELETE SET NULL`, `episode_title TEXT`, `still_url TEXT`.
- `IdentifySeries(ctx, TVSearcher, title, year)`: misma puntuación que las películas, pero con `SearchTV` y sin la regla de colecciones (reutilizar `matchVariants`, `scoreCandidate`, `poolWeights`).
- Escaneo: cada episodio se asocia a su fila de `series` por (`library_id`, `SeriesFolder`). Una identificación por serie (mismas reglas de `match_status` que las películas) con `TVDetails`; por temporada, `/tv/{id}/season/{n}` en es-ES (añadir `SeasonDetails` al cliente si hace falta, en `tmdb.go`) para nombre, sinopsis y cartel; por episodio `EpisodeDetails` → `episode_title`, `overview`, `still_url`. Solo se consulta lo que falta.
- Esfuerzo: alto.

### T12 — API de series
- Archivos: `internal/library/series_http.go` (nuevo, +test), `internal/library/http.go`, `api/openapi.yaml`.
- `GET /api/v1/series?limit&offset&q&unmatched` → `{series: [{id, title, poster_url, year, season_count, episode_count, match_status}], total}`.
- `GET /api/v1/series/{id}` → ficha de la serie + `seasons: [{season, name, overview, poster_url, episode_count}]`.
- `GET /api/v1/series/{id}/seasons/{n}` → episodios ordenados (DTO de ítem con `episode_title`, `still_url`).
- Corrección (admin): `GET /api/v1/series/{id}/candidates`, `PUT /api/v1/series/{id}/match`, `DELETE /api/v1/series/{id}/match`, con la misma semántica que las películas.
- Esfuerzo: medio.

### T13 — Web de series
- Archivos: `web/src/routes/series/+page.svelte|.server.ts`, `web/src/routes/series/[id]/+page.svelte|.server.ts`, ficha de episodio en `web/src/routes/items/[id]/+page.svelte`, `MatchDialog` reutilizado, tests.
- `/series`: rejilla paginada (48) con buscador y, para admin, «Sin identificar».
- `/series/[id]`: cabecera de la serie (fondo, título, años, géneros, sinopsis), pestañas o selector de temporada (`?season=N`), y la lista de episodios de la temporada (número, título del episodio, imagen, sinopsis corta, botón reproducir). Admin: «Corregir ficha» a nivel de serie.
- Ficha de episodio: serie › temporada › episodio, con enlace de vuelta.
- **Cierra el hito 3:** `VERSION` 0.13.0 y CHANGELOG.
- Esfuerzo: alto.

### D2 — Despliegue y medición de los hitos 2 y 3 (sesión)
- Igual que D1: las 26 series identificadas (corregir las que no), episodios con título en castellano, colecciones creadas (Harry Potter, Marvel…), y revisión con el usuario.

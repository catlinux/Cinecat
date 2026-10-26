# Fichas en castellano con TMDB, corrección manual y no identificadas

Estado: pasos 1 y 2 hechos (2026-10-26). Siguiente: paso 3.

## Problema (medido en producción, copia de la BD del 2026-10-26)

- 886 películas: solo 339 con cartel (38 %) y 308 con sinopsis; todo en inglés (OMDb).
- Causa: los archivos tienen **títulos en castellano y casi nunca año** («La lista de Schindler.avi», «Malditos bastardos.avi»), y OMDb solo busca por título original en inglés. Además, cuando OMDb acierta a ciegas con un título ambiguo, la ficha es de otra película.
- No se puede corregir una ficha mal identificada ni identificar a mano una desconocida.
- Series: 766 episodios, 26 series, sin metadatos. El nombre de la serie sale del archivo, no de la carpeta: «Big Bang» (carpeta «The Big Bang theory»), 3 variantes de «Juego de tronos», «Chicago PD»/«Chicago P D». 16 archivos de `Series/` quedan como película («MR_Robot_305_DivxTotaL.avi», «ChicagoFire720p4x17 [...]»).
- Decisión del usuario: **fuente TMDB** (clave v3 en `CINECAT_TMDB_API_KEY`, ya pasada por `docker-compose.yml`). OMDb se queda solo como respaldo si no hay clave de TMDB.

## Decisiones de diseño

- Idioma: todo se pide con `language=es-ES`. Si la sinopsis viene vacía, se reintenta con `es-MX` y, como último recurso, `en-US` (guardando `overview_language` para que la UI pueda avisar).
- Una ficha corregida a mano (`match_status = 'manual'`) **nunca** la toca un escaneo. `'none'` = el usuario dijo «no es ninguna»: tampoco se reintenta. `'auto'` = identificada sola; se refresca. `NULL` = sin identificar; se reintenta en cada escaneo.
- Identificación automática con puntuación (paso 2), no «primer resultado»: más vale dejar una película «sin identificar» que ponerle la ficha de otra.
- Respetar el límite de TMDB: limitador propio a ~20 peticiones/s en el cliente; caché en disco como la actual.
- Carteles: URLs de `image.tmdb.org` (como hace hoy OMDb con Amazon); sin descarga local en este plan.
- Sin cambios en reproducción (`internal/playback`).

## Pasos

Convenciones de todos los pasos: Go con `export PATH="/c/Users/msanc/tools/go1.26.9/bin:$PATH"`; validar con `gofmt -l internal cmd` (vacío), `go vet ./...`, `go test ./...`; web con `cd web && npm run check && npx vitest run`. Commits en castellano. Documentación es/en sincronizada. Archivos del repo en CRLF.

### Paso 1 — Cliente TMDB completo (películas y series) ✅ hecho
- Hecho además: `Candidate.VoteCount` y `MovieCollection(ctx, movieID)` (partes de la saga ordenadas por estreno), que usa el paso 2.
- Archivos: `internal/metadata/tmdb.go`, `internal/metadata/tmdb_test.go`.
- Añadir (manteniendo `Search` para compatibilidad):
  - `SearchMovies(ctx, query string, year int) ([]Candidate, error)`: `/search/movie` es-ES, hasta 10 resultados. `Candidate{TMDBID, Title, OriginalTitle, Year, Overview, PosterURL (w185), Popularity}`.
  - `MovieDetails(ctx, id int) (*Details, error)`: `/movie/{id}?append_to_response=credits,images&include_image_language=es,null,en`. `Details{TMDBID, IMDbID, Title, OriginalTitle, Year, Tagline, Overview, OverviewLanguage, Genres []string, RuntimeMinutes, Rating float64 (vote_average), Directors []string (crew job=Director), Cast []string (8 primeros), PosterURL (pickPoster), BackdropURL (w1280)}` con el respaldo de idioma de la sinopsis descrito arriba.
  - `SearchTV(ctx, query, year)` → `[]Candidate` (`name`, `original_name`, `first_air_date`), `TVDetails(ctx, id)` → `Details` (sin runtime; `Directors` = `created_by`), `EpisodeDetails(ctx, tvID, season, episode)` → `{Name, Overview, AirYear, StillURL}`.
  - Limitador de peticiones (máx. 20/s) en `getJSON`; reintento único tras 2 s si TMDB responde 429.
  - Caché en disco por petición (clave = ruta + parámetros sin `api_key`), TTL actual.
- Tests con `httptest` (sin red): idioma, respaldo de sinopsis, director/reparto, cartel, 429.
- Terminado: tests pasan; ningún uso de red real en tests.
- Ejecutor: AgentRelay, esfuerzo medio.
- Versión: 0.11.0 lo agrupa con el paso 2–3 (no publicar entre medias); CHANGELOG en «Sin publicar».

### Paso 2 — Identificación con puntuación ✅ hecho
- Implementado en `internal/library/match.go`: `IdentifyMovie(ctx, MovieSearcher, title, year) (Candidate, score, ok, err)`, umbral `MatchThreshold` = 0.75. Busca SIN año; el año solo puntúa.
- Ajustado con las 870 películas reales contra TMDB (prototipo en Python, fuera del repo): 854 identificadas (98 %), revisadas a mano todas las aproximadas. Reglas finales: variantes solo para buscar y puntuación siempre contra el título completo; F1 ponderado por IDF entre candidatos; contención 0.85; números de secuela (arábigos/romanos) suman o restan; saga + número sin subtítulo → parte N de la colección de TMDB; homónima con < 200 votos cede ante otra ≥ 20× más votada; «Título (Original)» como título alternativo; «Dr» → «Doctor».
- Test con respuestas reales grabadas: `internal/library/testdata/match_fixture.json` (53 casos). Si el código cambia las búsquedas que hace, el test falla.
- Fallos conocidos (corrección manual): «MEGAN» (M3GAN), «Desafío total - Remake» (elige 1990), Star Wars IV–VI (TMDB no lleva el subtítulo), erratas.
- Diseño original del paso (referencia):
- Archivos: `internal/library/match.go`, `internal/library/match_test.go`.
- `QueryVariants(title string, folder string) []string`, en orden: título limpio; sin número de saga final cuando la carpeta es una colección («Harry Potter Y El Prisionero De Azkaban 3» → sin « 3»; «Logan 8» → «Logan»); partes separadas por « - » («El Señor de los Anillos - 3 - El retorno del Rey» → «El Señor de los Anillos El retorno del Rey», «El retorno del Rey»); «HarryPotter» → «Harry Potter» (separar camelCase); «S A» → «S.A.».
- `BestMatch(parsedTitle string, year int, candidates []metadata.Candidate) (metadata.Candidate, float64, bool)`: similitud normalizada (minúsculas, sin acentos ni puntuación, sin artículos iniciales) contra `Title` y `OriginalTitle` (la mayor); +0.15 si el año coincide (±1), −0.3 si hay año y difiere más de 1; desempate por popularidad. Aceptar si ≥ 0.75.
- Tests con los títulos reales de la sección «Problema» (lista en `match_test.go`), con candidatos fijos sacados de TMDB al implementar.
- Ejecutor: la sesión (Opus); necesita la clave de TMDB para extraer candidatos reales.

### Paso 3 — Esquema y escaneo con TMDB
- Archivos: `internal/store/migrations/0004_media_item_tmdb.sql`, `internal/library/scanner.go`, `internal/library/scanner_metadata_test.go`, `cmd/cinecatd/main.go`.
- Migración: columnas `original_title, imdb_id, tagline, genres (JSON), runtime_minutes, rating REAL, directors (JSON), cast (JSON), backdrop_url, overview_language, metadata_source ('tmdb'|'omdb'), match_status ('auto'|'manual'|'none'|NULL)`; `tmdb_id` ya existe. Tabla `series (id, library_id, folder, title, tmdb_id, original_title, year, overview, poster_url, backdrop_url, genres, match_status, UNIQUE(library_id, folder))` y en `media_items` la columna `series_id` (FK) y `episode_title`.
- Escaneo: si hay cliente TMDB, para películas con `match_status` NULL o `'auto'` con `metadata_source != 'tmdb'`: variantes del paso 2 → `SearchMovies` → `BestMatch` → `MovieDetails` → guardar todo con `match_status='auto'`, `metadata_source='tmdb'`. Sin coincidencia: dejar NULL y limpiar los datos de OMDb (eran de otra película o en inglés). OMDb solo si no hay clave TMDB (comportamiento actual).
- `main.go`: construir el cliente TMDB si `cfg.TMDBAPIKey != ""`.
- Terminado: test con TMDB falso (`httptest`) que cubre auto, sin coincidencia y que `'manual'`/`'none'` no se tocan.
- Ejecutor: AgentRelay, esfuerzo alto.

### Paso 4 — API de corrección
- Archivos: `internal/library/http.go`, `internal/library/http_test.go`, `api/openapi.yaml`.
- DTO del ítem con los campos nuevos (`genres`, `directors`, `cast` como arrays; `match_status`).
- `GET /api/v1/items?unmatched=true` (sin identificar o `'none'`) y el total en la respuesta.
- `GET /api/v1/items/{id}/candidates?q=&year=` (solo admin): `q` por defecto = título analizado; devuelve `Candidate[]`.
- `PUT /api/v1/items/{id}/match` (admin) `{ "tmdb_id": 123 }` → `MovieDetails` y guardar con `'manual'`. `{ "tmdb_id": null }` → «no es ninguna»: limpiar metadatos, `'none'`. `DELETE /api/v1/items/{id}/match` → volver a automático (NULL) y reidentificar al momento.
- Ejecutor: AgentRelay, esfuerzo medio.

### Paso 5 — Web: ficha completa, corregir y «Sin identificar»
- Archivos: `web/src/routes/items/[id]/+page.svelte`, `+page.server.ts`, `web/src/lib/api.ts`, `web/src/routes/+page.svelte` (y componente nuevo `web/src/lib/components/MatchDialog.svelte`).
- Ficha: título, título original si difiere, año · duración · géneros · nota, lema, sinopsis, dirección, reparto, fondo (`backdrop_url`) discreto. Aviso «Sinopsis no disponible en castellano» si `overview_language` no empieza por `es`.
- Admin: botón «Corregir ficha» → diálogo con buscador (título + año opcional), lista de candidatos (cartel pequeño, título, título original, año, primera línea de sinopsis) → elegir; «No es ninguna de estas»; «Volver a automático».
- Cartelera: filtro «Sin identificar (N)» visible para admin.
- Dirección visual «Marquesina» (ver README); nada de estética genérica.
- Ejecutor: AgentRelay, esfuerzo alto; referencias: README (dirección visual).

### Paso 6 — Series
- Archivos: `internal/library/nameparser.go` (+test), `internal/library/scanner.go`, `internal/library/http.go`, ficha web de episodio.
- Nombre de la serie = carpeta de primer nivel bajo la biblioteca (`folderTitle`), no el archivo. Reconocer «305» → 3x05 y «4x17» pegado («ChicagoFire720p4x17») dentro de una carpeta de temporada.
- Una identificación por carpeta de serie (`SearchTV` + `BestMatch`), corrección manual a nivel de serie (mismos endpoints con `/api/v1/series/{id}/...`), y por episodio `EpisodeDetails` (título y sinopsis del episodio en castellano; cartel = el de la serie).
- Ejecutor: AgentRelay en dos tareas (analizador; escaneo+API), esfuerzo medio/alto; ficha web aparte.
- Versión: 0.12.0.

### Paso 7 — Despliegue y verificación con datos reales
- Sesión (Opus/Sonnet): push con aprobación, `./scripts/actualizar.sh`, escaneo, y medir con la copia de la BD: % de películas identificadas, revisar a mano 40 al azar (acierto real), lista de no identificadas para el usuario.
- Criterio: ≥ 85 % identificadas y ninguna de las 40 revisadas con ficha de otra película; las que queden se corrigen desde la web.

# Plan: bibliotecas con varias carpetas, vídeos sin identificar y música

Petición del usuario: crear bibliotecas con **varias carpetas** (en discos distintos), elegir la carpeta **navegando por el servidor**, y añadir **música** y **vídeos** (no películas ni series) como tipos de biblioteca.

## Punto de partida (verificado en el código)

- `libraries(id, path UNIQUE, name)`: una sola carpeta por biblioteca y sin tipo.
- `media_items.kind` solo admite `movie | series | episode` (CHECK) y `path` es absoluto.
- El escaneo recorre `libraries.path` con `filepath.WalkDir` (`internal/library/scanner.go`) y deduce películas y series por el nombre del archivo.
- Rutas HTTP en `internal/library/http.go`: `GET/POST /api/v1/libraries`, `DELETE /api/v1/libraries/{id}`, escaneo en `/libraries/scan`.
- El contenedor ve la biblioteca en `/media` (solo lectura hoy; ver `gestion-archivos.PLAN.md` para escritura).

## Decisiones

- **Tipos de biblioteca** (`libraries.kind`): `video` (películas y series, como hoy; por defecto y para las existentes), `clips` («Vídeos»: sin identificar, sin TMDB), `music` («Música»).
- **Vídeos (`clips`)** reutilizan `media_items` con `kind = 'movie'`, `metadata_source` NULL y `match_status = 'none'` para no tocar el CHECK de la tabla ni duplicar reproducción, progreso, visto y preferencias. El título sale del nombre del archivo; se navegan **por carpetas**.
- **Música** usa tablas propias (`music_artists`, `music_albums`, `music_tracks`), porque una pista no es una película. Etiquetas leídas con `ffprobe` (`format.tags`: title, artist, album_artist, album, track, disc, date/year, genre); carátula de `folder.jpg|cover.jpg|front.jpg` o la incrustada (extraída con FFmpeg a la caché). Sin servicios externos (sin telemetría).
- **Varias carpetas**: tabla `library_paths(library_id, path UNIQUE)`; `libraries.path` queda como la primera para compatibilidad y se retira más adelante. Quitar una carpeta pide confirmación y borra sus ítems de la base (no los archivos).
- **Navegador de carpetas del servidor**: `GET /api/v1/fs?path=` solo para administradores, **confinado** a las raíces `CINECAT_MEDIA_ROOTS` (por defecto `/media`), solo directorios, sin seguir enlaces simbólicos que escapen, con límite de entradas.
- **Reproducción de audio**: directa (rango de bytes) para mp3, aac/m4a, ogg/opus y flac; si el navegador no lo reproduce, transcodificación a AAC con FFmpeg (fase posterior, no bloquea).
- **Reproductor de música persistente**: un `<audio>` en el layout para que siga sonando al navegar, con cola y mini reproductor fijo abajo.

## Pasos (un ejecutor cada uno, en este orden)

1. **T18 — «Añadido recientemente» de series en el portal** (pequeña). Ejecutor: AgentRelay, esfuerzo medio. Versión patch/minor con la siguiente.
2. **L1 — Varias carpetas y navegador de carpetas (servidor).** Migración `library_paths` + `libraries.kind`; API de crear/editar bibliotecas con varias carpetas y tipo; `GET /api/v1/fs`; escaneo recorre todas las carpetas; quitar una carpeta limpia sus ítems; tests (confinamiento de rutas, enlaces simbólicos, carpeta repetida entre bibliotecas, compatibilidad con bibliotecas existentes). Ejecutor: AgentRelay, esfuerzo alto, revisión mía de la parte de seguridad de `/fs`.
3. **L2 — Pantalla de bibliotecas (web).** Crear/editar: nombre, tipo, lista de carpetas con navegador de carpetas del servidor, añadir y quitar carpetas, escanear una biblioteca. Ejecutor: AgentRelay, esfuerzo alto.
4. **V1 — Biblioteca de vídeos (`clips`).** Escáner sin identificación, títulos por nombre y carpeta; web `/videos` con navegación por carpetas (migas de pan), tarjetas con carátula generada y la ficha mínima reutilizando la reproducción. Ejecutor: AgentRelay, esfuerzo alto.
5. **M1 — Música: servidor.** Migración de tablas, escáner con `ffprobe`, carátulas, API (`/music/artists`, `/music/albums`, `/music/albums/{id}`, `/music/tracks/{id}/stream` firmada, búsqueda), reproducción directa con rangos. Ejecutor: AgentRelay, esfuerzo alto.
6. **M2 — Música: web.** `/music` con Artistas y Álbumes, página de álbum, búsqueda, reproductor persistente con cola, controles y atajos. Ejecutor: AgentRelay, esfuerzo alto.
7. **M3 — Música: extras.** Transcodificación de formatos no compatibles, listas de reproducción, favoritos (reutiliza el sistema de favoritos F1), orden aleatorio.

## Después de cada paso

Pruebas Go y web verdes, `npm run build`, CHANGELOG, commit y, al cerrar versión, etiqueta y aviso para subir y desplegar con el botón de actualización.

## Lo que no se hace en este plan

Metadatos externos de música (MusicBrainz, letras), podcasts, radio, ni imágenes/fotos como tipo (queda en el plan de tipos de biblioteca posterior).

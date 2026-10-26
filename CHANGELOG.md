# Registro de cambios

Todos los cambios relevantes de este proyecto se documentan aquí.
Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y versionado según [Versionado Semántico](https://semver.org/lang/es/) (`x.y.z`).

[English version](CHANGELOG.en.md)

## [Sin publicar]

### Añadido
- El escaneo de bibliotecas se ejecuta en segundo plano: `POST /api/v1/libraries/scan` responde 202 (o 409 si ya hay uno en marcha) y `GET /api/v1/libraries/scan` informa del estado y del último resultado.
- Escaneo automático un minuto después de arrancar y después cada `CINECAT_SCAN_INTERVAL` (por defecto `6h`; `0` lo desactiva).
- El panel de bibliotecas lanza el escaneo en segundo plano y muestra «Escaneo en curso…» con sondeo cada 3 s; al acabar, archivos, identificadas, sin identificar y avisos plegables.
- `GET /api/v1/system/version` (sin sesión) con la versión, el commit y la fecha de compilación; el pie de la web muestra la versión del servidor y avisa si la web y el servidor tienen versiones distintas. La versión de la web sale del `VERSION` de la raíz (la imagen de la web se construye ahora desde la raíz del repositorio).
- El servidor se apaga de forma ordenada ante SIGINT/SIGTERM, cancelando el escaneo en curso.

## [0.10.0] - 2026-10-26

### Corregido
- Las películas con audio AC-3 5.1 (casi toda la biblioteca real) se quedaban cargando: el audio se convertía a AAC 5.1, que los reproductores del navegador no aceptan. Ahora se baja siempre a AAC estéreo.
- Saltar a un punto avanzado ya no deja el vídeo «pensando»: si el segmento pedido queda lejos de lo que FFmpeg ha generado (o es un hueco hacia atrás), el servidor detiene FFmpeg y lo relanza en ese segmento (`-ss` + `-start_number`, el mismo criterio que Jellyfin). Un salto tarda menos de un segundo en vídeo SD, en vez de esperar hasta 5 minutos y fallar.
- La reproducción ya no se corta a los 15 minutos: los segmentos HLS y los saltos en reproducción directa reutilizan la firma de la URL inicial, que caducaba a los 15 minutos. La vida por defecto de la URL pasa a 8 horas.
- El último segmento de la playlist ya no puede ser un resto de milisegundos que FFmpeg no llega a generar (fallaba al final del vídeo).
- El reproductor web usa hls.js siempre que el navegador lo admite, en vez del HLS nativo que anuncian Chrome en Android y en escritorio; reintenta ante errores de red o de descodificación en lugar de rendirse al primero, y espera hasta 60 s al primer byte de un segmento.

### Cambiado
- HLS usa segmentos MPEG-TS (`segment_XXXXX.ts`) en vez de fMP4 (`init.mp4` + `.m4s`): cada segmento lleva sus marcas de tiempo absolutas, así que los generados por procesos FFmpeg distintos encajan en la misma línea de tiempo. Las marcas de salida llevan un desplazamiento fijo de 10 s para que el proceso que empieza en 0 no tenga DTS negativos (que FFmpeg corregía desplazando unos fotogramas).
- Todo lo que se sirve por HLS se recodifica a H.264 (libx264 `veryfast`, CRF 20, hasta 1080p y dimensiones pares) + AAC estéreo. Se elimina el remux sin recodificar: la biblioteca real es casi toda DivX/MPEG-4 con AC-3 o MP3, que el navegador no reproduce. FFmpeg ya no arranca al pedir la playlist, sino con la primera petición de segmento, en el punto pedido.
- La API sigue anunciando `mode: "remux"` para HLS.

## [0.9.1] - 2026-10-26

### Corregido
- La playlist HLS que se sirve ya es un VOD completo, no un evento en directo: Cinecat calcula la duración con `ffprobe` y escribe él mismo `playlist.m3u8` antes de arrancar FFmpeg, con todos los segmentos esperados, `EXT-X-PLAYLIST-TYPE:VOD` y `EXT-X-ENDLIST` desde la primera petición. Así hls.js conoce la duración total y permite avanzar/retroceder. FFmpeg escribe su propia playlist aparte (`ffmpeg.m3u8`) para no pisarla, y los segmentos se siguen sirviendo en cuanto existen. Si no se puede leer la duración (algunos AVI antiguos), se mantiene el fallback a la playlist `event` de FFmpeg.

## [0.9.0] - 2026-10-26

### Corregido
- La generación HLS ya no bloquea hasta procesar el vídeo completo: `Service.prepare()` arranca FFmpeg en segundo plano y devuelve en cuanto existe la playlist inicial, y los segmentos se sirven en cuanto FFmpeg los genera (con espera acotada y comprobación de que el segmento está completo). Para que la playlist exista desde el primer segmento se usa `-hls_playlist_type event` en vez de `vod` (con `vod` FFmpeg solo la escribía al terminar); al acabar se añade `ENDLIST` y sigue siendo un VOD normal. El proceso queda ligado al ciclo de vida del servicio, no a la petición HTTP que lo lanza, y dos peticiones simultáneas del mismo ítem reutilizan el mismo proceso. Antes, una película de una hora podía tardar más de 25 minutos en empezar.

## [0.8.1] - 2026-10-26

### Corregido
- La 0.8.0 no arreglaba todos los casos reales: un `.avi` de producción con 2804 keyframes bien repartidos (por debajo del umbral de la detección) seguía generando un único segmento HLS con la película entera, porque `-c:v copy` necesita timestamps de paquete fiables para segmentar, y en AVI/DivX muchos paquetes llevan `pts_time` como `N/A`. El intervalo entre keyframes puede ser correcto y el contenedor no ser segmentable igualmente. Ahora cualquier `.avi` se transcodifica siempre, sin pasar por el análisis de keyframes, que se mantiene para el resto de contenedores (`.mkv`, `.mov`, `.ts`, `.m2ts`).

## [0.8.0] - 2026-10-26

### Añadido
- Transcodificación de vídeo por software cuando el intervalo entre keyframes impide un remux HLS razonable: en vez de fallar, se recodifica solo el vídeo con libx264 `veryfast`/CRF 20, audio AAC 160k y keyframes forzados cada 6 s, manteniendo el resto del flujo HLS. Esto resuelve la reproducción de `.avi`/DivX con keyframes dispersos como el caso real de producción; no implementa todavía el motor completo de perfiles ni aceleración por hardware, y la API sigue anunciando `mode: "remux"` (el cliente recibe HLS en ambos casos).

## [0.7.3] - 2026-10-26

### Corregido
- El remux HLS comprueba con `ffprobe`, antes de lanzar FFmpeg, que el intervalo máximo entre keyframes sea viable (≤ 15 s). En vídeos con keyframes muy separados (típico en `.avi`/DivX antiguos) devuelve un error claro en vez de generar un único segmento gigante que el navegador no puede empezar a reproducir. **No repara la reproducción de esos archivos**: la transcodificación real queda para el siguiente paso de la Fase 3.

## [0.7.2] - 2026-10-26

### Añadido
- `GET /api/v1/items` acepta `library_id` opcional para filtrar por biblioteca (necesario de cara al cliente web, con 1652 ítems reales en producción la lista plana sin filtro ni paginación visible resultaba inmanejable).

## [0.7.1] - 2026-10-26

### Corregido
- `docker-compose.yml` no pasaba `CINECAT_OMDB_API_KEY` al contenedor del servidor (se añadió el soporte en el código en la 0.6.0, pero se olvidó en el compose): aunque la clave estuviera en `.env`, el servidor nunca la veía y el escaneo seguía sin metadatos. Añadida también a `.env.deploy.example`.

## [0.7.0] - 2026-10-26

### Añadido
- La biblioteca muestra el cartel real de OMDb cuando existe y mantiene el cartel tipográfico de respaldo cuando falta. La ficha de ítem muestra también el cartel real y añade la sinopsis (`overview`) cuando está disponible, omitiendo la sección si no hay datos.

## [0.6.0] - 2026-10-25

### Añadido
- Metadatos de OMDb para películas durante el escaneo (`CINECAT_OMDB_API_KEY`): sinopsis y URL del cartel, con caché en disco de las búsquedas, sin repetir consultas para ítems que ya tienen datos ni para series/episodios. Se exponen como `overview` y `poster_url` en `GET /api/v1/items` y `GET /api/v1/items/{id}`. Los fallos o ausencias de OMDb quedan como avisos del escaneo.

## [0.5.0] - 2026-10-25

### Añadido
- Panel `/admin/libraries` para usuarios con rol admin: lista las bibliotecas, permite crear una nueva (nombre y ruta), borrarla con confirmación y lanzar el escaneo completo mostrando bibliotecas, archivos, entradas actualizadas y avisos. Enlace «Bibliotecas» en la cabecera solo para administradores, con el rol obtenido de `GET /api/v1/auth/me`; los usuarios sin rol reciben un 403 claro.

## [0.4.1] - 2026-10-25

### Añadido
- `GET /api/v1/auth/me`: devuelve `{username, role}` del usuario de la sesión Bearer actual, con 401 si falta el token o no es válido.

## [0.4.0] - 2026-10-25

### Añadido
- Pantalla «Mi cuenta» en el cliente web para cambiar la contraseña: formulario con contraseña actual, nueva y repetición, validación local de coincidencia y longitud mínima (8 caracteres), mensajes claros para errores de la API y confirmación de éxito sin cerrar la sesión actual. Enlace «Mi cuenta» en la cabecera solo cuando hay sesión.

## [0.3.5] - 2026-10-25

### Añadido
- `POST /api/v1/auth/password`: permite al usuario autenticado cambiar su propia contraseña comprobando la actual (401 si no coincide, 400 si la nueva tiene menos de 8 caracteres) y sin invalidar la sesión ni el resto de sesiones vigentes.

## [0.3.4] - 2026-10-25

### Cambiado
- `docs/es/despliegue.md`/`docs/en/deployment.md` y `.env.deploy.example` explican cómo dar de alta varias bibliotecas (p. ej. películas y series) cuando `CINECAT_MEDIA_PATH` apunta a un disco o carpeta que las contiene a todas como subcarpetas, en vez de asumir una sola biblioteca en la raíz montada.

## [0.3.3] - 2026-10-25

### Añadido
- Flag `cinecatd --hash-password <contraseña>`: genera el hash argon2id en formato PHC y lo imprime, para fijar manualmente `password_hash` en la base de datos (p. ej. con `sqlite3` dentro del contenedor) mientras no existe todavía un endpoint ni pantalla para cambiar la contraseña desde la interfaz.

## [0.3.2] - 2026-10-25

### Corregido
- El cliente web rechazaba toda petición a la API con «Falta configurar PUBLIC_API_URL en el servidor», incluido el login, cuando `PUBLIC_API_URL` se dejaba vacío a propósito para usar rutas relativas detrás del proxy del despliegue Docker (ver `0.3.1`). `createApiClient` (`web/src/lib/api-factory.ts`) ya no trata la cadena vacía como error de configuración; `web/src/routes/login/+page.server.ts` usa ahora esa misma función en vez de duplicar la lógica con su propio chequeo.

## [0.3.1] - 2026-10-25

### Cambiado
- El despliegue Docker ya no lleva Caddy como proxy propio: el Apache que ya gestiona otros vhosts del Debian hace de proxy inverso hacia el servidor y el cliente web (vhost de ejemplo en `deploy/video.warcrafted.com.conf`). `docker-compose.yml` publica ambos contenedores solo en `127.0.0.1` (puertos `CINECAT_API_PORT`/`CINECAT_WEB_PORT`, nunca expuestos directamente a la red). `.env.deploy.example` y `docs/es/despliegue.md`/`docs/en/deployment.md` actualizados con los dos pasos (contenedores + vhost de Apache).

## [0.3.0] - 2026-10-25

### Añadido
- Despliegue con Docker Compose: `Dockerfile` del servidor Go (con FFmpeg instalado en la imagen), `web/Dockerfile` del cliente web, `docker-compose.yml` con Caddy como proxy único delante de ambos (mismo origen, sin necesidad de CORS), script `scripts/actualizar.sh` para aplicar versiones nuevas sin perder datos, y guía de despliegue en `docs/es/despliegue.md`.
- Soporte de `CINECAT_ADMIN_PASSWORD` documentado en el flujo de despliegue para fijar la contraseña del administrador en el primer arranque sin depender del log.
- Versión real incrustada en el binario del servidor en el `Dockerfile` (antes mostraba «desconocida» en un binario compilado sin `-ldflags`).

### Cambiado
- `docker-compose.yml` pasa `PROTOCOL_HEADER=x-forwarded-proto` al contenedor del cliente web (variable que ya soporta `@sveltejs/adapter-node`), de forma que la cookie de sesión se marca `Secure` correctamente detrás de Caddy.

## [0.2.0] - 2026-10-25

Cierra la Fase 2: cliente web funcional con inicio de sesión, biblioteca, fichas y reproductor.

### Añadido
- Modo claro/oscuro/sistema conmutable y persistido en `localStorage`, con aplicación antes del primer pintado para evitar parpadeos. Paleta oscura «Marquesina» coherente con la clara, contraste de texto revisado (≥ 4.5:1), foco visible unificado para enlaces, botones, campos y vídeo, y conmutador accesible en todas las páginas.

## [0.2.0-dev.4] - 2026-10-25

### Añadido
- Reproductor web real en la ruta de reproducción: `<video>` nativo con controles para `mode: "direct"` y hls.js 1.7.3 (Apache-2.0) para `mode: "remux"`, con HLS nativo en Safari/iOS, mensaje claro si el navegador no soporta HLS, limpieza de `hls.destroy()` al salir y estilos de sala oscura alrededor del vídeo.

## [0.2.0-dev.3] - 2026-10-25

### Añadido
- Pantalla de biblioteca en la portada (grid de `PosterCard` con cartel de respaldo tipografiado y color estable derivado del título) y ficha de ítem con metadatos y botón «Reproducir» hacia una ruta de reproducción provisional que muestra el modo y la URL firmada. Las páginas protegidas redirigen a `/login` sin sesión.

## [0.2.0-dev.2] - 2026-10-25

### Añadido
- Cliente de API tipado y gestión de sesión del cliente web: login y logout mediante form actions de SvelteKit, token guardado en cookie httpOnly con SameSite=Lax y Secure bajo HTTPS, URL del servidor por `PUBLIC_API_URL` y página de login «Marquesina» con mensajes para credenciales incorrectas y bloqueo temporal.

## [0.2.0-dev.1] - 2026-10-25

### Añadido
- Esqueleto del cliente web en `web/`: SvelteKit + TypeScript con `@sveltejs/adapter-node`, tipografías Fraunces e Inter empaquetadas localmente (OFL-1.1) y dirección visual «Marquesina» en la portada. La URL base del servidor se configura con `PUBLIC_API_URL` (ver `web/.env.example`).

## [0.1.3] - 2026-10-25

### Añadido
- `POST /api/v1/libraries/scan`, restringido a administradores: dispara un escaneo síncrono de todas las bibliotecas y devuelve el número de bibliotecas recorridas, archivos encontrados y entradas actualizadas, junto con los avisos de rutas ilegibles.

## [0.1.2] - 2026-10-25

### Añadido
- `POST /api/v1/libraries` y `DELETE /api/v1/libraries/{id}`, restringidos a usuarios con rol admin. Valida que la ruta exista y sea un directorio; 409 si ya hay una biblioteca con esa ruta; el borrado limpia en cascada sus `media_items`.

### Cambiado
- `requireSession` ahora permite exigir también un rol concreto (`requireAdmin`), sin cambiar el comportamiento de los endpoints existentes.

### Añadido
- Alta automática del primer usuario administrador al arrancar el servidor contra una base de datos sin usuarios: contraseña por `CINECAT_ADMIN_PASSWORD` o generada aleatoriamente (144 bits) y mostrada una sola vez en el log.

## [0.1.0] - 2026-10-25

Cierra la Fase 1: servidor mínimo funcional (sin clientes todavía).

### Añadido
- Esqueleto del servidor Go (`cinecatd`): módulo, configuración por archivo/entorno, flag `--version` y endpoint `GET /healthz` (Fase 1, paso 1).
- Esquema de base de datos SQLite (`users`, `sessions`, `libraries`, `media_items`, `playback_state`) y migraciones embebidas e idempotentes (Fase 1, paso 2). Driver: `modernc.org/sqlite` (BSD-3-Clause).
- Autenticación con argon2id, sesiones de 24h renovables (caducidad deslizante) y bloqueo temporal tras 5 intentos fallidos; endpoints `POST /api/v1/auth/login` y `POST /api/v1/auth/logout` (Fase 1, paso 3). Dependencia: `golang.org/x/crypto` (BSD-3-Clause).
- Escaneo de biblioteca y analizador propio de nombres de archivo y carpeta (título, año, temporada y episodio), con upsert en `media_items` sin duplicar en re-escaneos (Fase 1, paso 4). Columnas `season`/`episode` añadidas al esquema.
- Cliente de metadatos TMDB (es-ES): búsqueda de películas, selección de cartel por prioridad de idioma (es-ES → español genérico → inglés → sin idioma) y caché en disco con TTL configurable para búsquedas e imágenes (Fase 1, paso 5). Sin conectar todavía al escaneo ni a los endpoints; solo busca películas, series quedan para más adelante.
- Contrato OpenAPI (`api/openapi.yaml`) y endpoints `GET /api/v1/libraries`, `GET /api/v1/items` (con paginación) y `GET /api/v1/items/{id}`, protegidos con sesión Bearer (Fase 1, paso 6).
- Reproducción directa (`http.ServeContent`, con rangos HTTP) para `.mp4`/`.m4v` y remux a HLS fMP4 sin reencodar (copia de códecs) para `.mkv`/`.avi`/`.mov`/`.ts`/`.m2ts`, invocando FFmpeg como proceso externo con argumentos siempre en lista. URLs de reproducción firmadas con HMAC-SHA256 (clave de proceso, nunca la de sesión) y caducidad de 15 minutos. Endpoints `GET /api/v1/items/{id}/play`, `GET /api/v1/stream/{id}` y `GET /api/v1/stream/{id}/{file}` (Fase 1, paso 7).

### Nota de verificación
- El entorno de desarrollo usado para esta fase no tiene FFmpeg instalado: los tests que ejercitan el remux real (`TestReproduccionDirectaMP4`, `TestRemuxMKVGeneraHLS`) se han saltado con `t.Skip()` en todas las revisiones de esta fase y **nunca se han ejecutado de verdad**. El resto de pruebas de reproducción (firma, caducidad, decisión de modo, ausencia de shell en el código) sí se han ejecutado y pasan. Antes de confiar en el remux en producción, ejecutar la suite completa en una máquina con FFmpeg instalado.

## [0.0.5] - 2026-10-25

### Añadido
- Pila tecnológica de la Fase 0 completada: web en **TypeScript + SvelteKit** + hls.js; Android/Android TV en **Kotlin + Jetpack Compose** + Media3/ExoPlayer (libmpv opcional).

## [0.0.4] - 2026-10-24

### Añadido
- `CLAUDE.md` del proyecto con las decisiones ya tomadas, la tabla de licencias de terceros y qué no hacer.
- Comprobación del nombre «Cinecat»: sin marca registrada ni repositorio en conflicto; se documenta como riesgo aceptado la existencia de una app Android inactiva desde 2015 con el mismo nombre.

## [0.0.3] - 2026-10-24

### Añadido
- **Licencia decidida: GPL-3.0**, con la condición adicional del artículo 7 que exige conservar los avisos de autoría de Catlinux. Archivo `LICENSE` en la raíz.
- Tabla de licencias verificadas (en el repositorio oficial de cada proyecto) de los candidatos a reutilización: Jellyfin y clientes (GPL-2.0), Findroid e Intro Skipper (GPL-3.0), Streamyfin (MPL-2.0), Media3/hls.js/Shaka (Apache-2.0), Vidstack (MIT).
- Pila tecnológica del servidor decidida: **Go**.

### Cambiado
- README, `docs/es` y `docs/en` (licencias-y-créditos, visión, arquitectura, hoja de ruta) y `TODO` actualizados para reflejar la licencia y la pila del servidor ya decididas.

## [0.0.2] - 2026-10-24

### Añadido
- Dirección visual elegida para la interfaz: «Marquesina» (cine de barrio de los 50-60, serif Fraunces, carteles ilustrados propios), con enlace a la maqueta de referencia.
- Archivo `VERSION`.

## [0.0.1] - 2026-10-24

### Añadido
- Documentación inicial bilingüe (castellano e inglés): visión, arquitectura, reproducción y códecs, metadatos, licencias y créditos, y hoja de ruta.
- `README`, `TODO`, `CONTRIBUTING`, `SECURITY`, `AUTHORS` y `CREDITS`.
- Decisiones de partida: servidor y clientes propios; web, Android y Android TV; H.265 preferido con H.264 de reserva; metadatos en español de España a partir de bases de datos existentes.

[Sin publicar]: https://github.com/catlinux/cinecat/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/catlinux/cinecat/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/catlinux/cinecat/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/catlinux/cinecat/compare/v0.0.5...v0.1.0
[0.0.5]: https://github.com/catlinux/cinecat/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/catlinux/cinecat/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/catlinux/cinecat/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/catlinux/cinecat/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/catlinux/cinecat/releases/tag/v0.0.1

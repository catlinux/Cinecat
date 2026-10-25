# Registro de cambios

Todos los cambios relevantes de este proyecto se documentan aquí.
Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y versionado según [Versionado Semántico](https://semver.org/lang/es/) (`x.y.z`).

[English version](CHANGELOG.en.md)

## [Sin publicar]

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

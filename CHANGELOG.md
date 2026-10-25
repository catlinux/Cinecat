# Registro de cambios

Todos los cambios relevantes de este proyecto se documentan aquí.
Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y versionado según [Versionado Semántico](https://semver.org/lang/es/) (`x.y.z`).

[English version](CHANGELOG.en.md)

## [Sin publicar]

### Añadido
- Esqueleto del servidor Go (`cinecatd`): módulo, configuración por archivo/entorno, flag `--version` y endpoint `GET /healthz` (Fase 1, paso 1).
- Esquema de base de datos SQLite (`users`, `sessions`, `libraries`, `media_items`, `playback_state`) y migraciones embebidas e idempotentes (Fase 1, paso 2). Driver: `modernc.org/sqlite` (BSD-3-Clause).
- Autenticación con argon2id, sesiones de 24h renovables (caducidad deslizante) y bloqueo temporal tras 5 intentos fallidos; endpoints `POST /api/v1/auth/login` y `POST /api/v1/auth/logout` (Fase 1, paso 3). Dependencia: `golang.org/x/crypto` (BSD-3-Clause).
- Escaneo de biblioteca y analizador propio de nombres de archivo y carpeta (título, año, temporada y episodio), con upsert en `media_items` sin duplicar en re-escaneos (Fase 1, paso 4). Columnas `season`/`episode` añadidas al esquema.
- Cliente de metadatos TMDB (es-ES): búsqueda de películas, selección de cartel por prioridad de idioma (es-ES → español genérico → inglés → sin idioma) y caché en disco con TTL configurable para búsquedas e imágenes (Fase 1, paso 5). Pendiente de conectar al servidor (pasos 6-7); solo busca películas, series quedan para más adelante.
- Contrato OpenAPI (`api/openapi.yaml`) y endpoints `GET /api/v1/libraries`, `GET /api/v1/items` (con paginación) y `GET /api/v1/items/{id}`, protegidos con sesión Bearer (Fase 1, paso 6).

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

[Sin publicar]: https://github.com/catlinux/cinecat/compare/v0.0.5...HEAD
[0.0.5]: https://github.com/catlinux/cinecat/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/catlinux/cinecat/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/catlinux/cinecat/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/catlinux/cinecat/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/catlinux/cinecat/releases/tag/v0.0.1

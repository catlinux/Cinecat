# Fase 1: servidor mínimo (0.1.0)

Hoja de ruta para la Fase 1 de Cinecat (ver `docs/es/hoja-de-ruta.md`). Objetivo: escaneo de biblioteca, análisis de nombres, metadatos `es-ES` con caché de imágenes, usuarios y sesiones, API versionada con contrato OpenAPI, reproducción directa y remux (HLS fMP4), con pruebas desde el principio.

Decisiones ya fijadas (no las recuestiones): servidor en **Go**, licencia **GPL-3.0 + atribución**, sin FFmpeg en el repo (se detecta en el sistema), sin telemetría, contraseñas con argon2id, sesiones de vida corta, URLs de reproducción firmadas y de corta duración (ver `docs/es/arquitectura.md`, `docs/es/reproduccion.md`, `docs/es/metadatos.md`).

Cada paso es autocontenido: un ejecutor sin memoria de esta conversación debe poder abordarlo leyendo solo este documento y los archivos que indica. Al completar un paso, el ejecutor lo marca `[x]` aquí mismo y añade su entrada en `CHANGELOG.md`/`CHANGELOG.en.md` (`## [Sin publicar]`) y el `VERSION` correspondiente, en el mismo commit.

Convención de commits: `tipo: descripción en castellano` (`feat`, `fix`, `docs`, `test`, `chore`, `refactor`). Nunca `go get` de una dependencia sin comprobar su licencia primero (ver tabla en `CLAUDE.md`).

## Paso 1 — Esqueleto del módulo Go y configuración [x] hecho (ejecución 20261025-020450-b014)

- **Archivos:** `go.mod` (módulo `github.com/catlinux/cinecat`), `cmd/cinecatd/main.go`, `internal/config/config.go`, `.gitignore` (añadir binarios Go y `*.db`), `Makefile` o `justfile` sencillo con `build`, `test`, `run`.
- **Qué hace:** arranca un binario `cinecatd` que lee configuración desde variables de entorno o un archivo fuera de control de versiones (nunca claves de API en el repo — ver `docs/es/licencias-y-creditos.md` y la regla de seguridad global), expone un flag `--version` que imprime el contenido de `VERSION`, y un servidor HTTP mínimo con un único endpoint `GET /healthz` que responde `200 OK`.
- **Criterio de terminado:** `go build ./...` y `go vet ./...` sin errores; `go run ./cmd/cinecatd --version` imprime la versión; el servidor arranca y `curl localhost:PUERTO/healthz` responde `200`.
- **Ejecutor:** AgentRelay, esfuerzo bajo.
- **Versión/CHANGELOG:** `0.1.0-dev.1` (parche sobre la base 0.0.5; sigue sin ser una versión publicada) — Añadido: esqueleto del servidor Go, configuración y `/healthz`.

## Paso 2 — Esquema de base de datos y migraciones [x] hecho (ejecución 20261025-021856-35ab)

- **Archivos:** `internal/store/migrations/0001_init.sql` (o la convención que fije el ejecutor con una librería de migraciones ligera para SQLite, p. ej. `golang-migrate` o `goose` — comprobar licencia, ambas MIT/Apache-2.0), `internal/store/store.go`.
- **Qué hace:** define las tablas mínimas: `users` (id, username, password_hash argon2id, role, created_at), `sessions` (id, user_id, token_hash, expires_at), `libraries` (id, path, name), `media_items` (id, library_id, path, title, year, kind: película/serie/episodio, tmdb_id nullable), `playback_state` (user_id, media_item_id, position_seconds, updated_at). Nunca guardar contraseñas ni tokens en claro.
- **Criterio de terminado:** las migraciones se aplican sobre una base SQLite vacía sin error; test que arranca la base en memoria, aplica migraciones y comprueba que las tablas existen.
- **Ejecutor:** AgentRelay, esfuerzo medio (diseño de esquema con cuidado de los índices y claves foráneas).
- **Dependencias:** Paso 1.
- **Versión/CHANGELOG:** `0.1.0-dev.2` — Añadido: esquema de base de datos y migraciones iniciales.

## Paso 3 — Usuarios, contraseñas y sesiones [x] hecho (ejecución 20261025-023620-a67a; riesgo pendiente: bloqueo por usuario, no por IP — ver TODO.md)

- **Archivos:** `internal/auth/auth.go`, `internal/auth/auth_test.go`, endpoints HTTP `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`.
- **Qué hace:** hash de contraseña con argon2id (librería `golang.org/x/crypto/argon2`, BSD-3-Clause, compatible), generación de tokens de sesión aleatorios de vida corta (p. ej. 24h, renovables), invalidación en logout, límite de intentos fallidos con bloqueo temporal (ver `docs/es/arquitectura.md`, punto 3). Sin PIN todavía (es para perfiles, Fase 2).
- **Criterio de terminado:** tests que cubren login correcto, contraseña incorrecta, bloqueo tras N intentos, logout invalida el token, token expirado se rechaza. `go test ./internal/auth/...` en verde.
- **Ejecutor:** AgentRelay, esfuerzo medio (seguridad: revisar con cuidado antes de aceptar).
- **Dependencias:** Paso 2.
- **Versión/CHANGELOG:** `0.1.0-dev.3` — Añadido: autenticación con argon2id y sesiones de vida corta.

## Paso 4 — Escaneo de biblioteca y análisis de nombres [x] hecho (ejecución 20261025-024356-06eb; añadió la migración 0002 con las columnas `season`/`episode` en `media_items`, no prevista en el Paso 2)

- **Archivos:** `internal/library/scanner.go`, `internal/library/nameparser.go`, tests con nombres de archivo variados (incluyendo rutas con caracteres especiales, espacios, `-` inicial — ver riesgo de inyección de argumentos en `docs/es/arquitectura.md` punto 9, aplica también aquí por si se reusa el parser para construir comandos).
- **Qué hace:** recorre una o más rutas configuradas como `libraries`, detecta archivos de vídeo por extensión, extrae título/año/temporada/episodio del nombre de archivo y carpeta. Decidir aquí (y documentarlo en el propio código, una línea) si se usa una librería de terceros para el análisis de nombres o se escribe propia; si es de terceros, comprobar su licencia exacta antes de añadirla (ver tabla de `CLAUDE.md`: MIT y Apache-2.0 son compatibles, LGPL-3.0 como `guessit` necesita revisión aparte porque es un enlace a biblioteca, no texto copiado — está en Python, así que en Go la alternativa real es escribir el analizador propio).
- **Criterio de terminado:** test con al menos 15 nombres de archivo reales variados (castellano e inglés, con y sin año, con y sin SxxEyy) que verifica el resultado esperado; el escáner puebla `media_items` sin duplicar entradas en un segundo escaneo.
- **Ejecutor:** AgentRelay, esfuerzo medio.
- **Dependencias:** Paso 2.
- **Versión/CHANGELOG:** `0.1.0-dev.4` — Añadido: escaneo de biblioteca y análisis de nombres de archivo.

## Paso 5 — Cliente de metadatos TMDB (es-ES)

- **Archivos:** `internal/metadata/tmdb.go`, `internal/metadata/tmdb_test.go` (con un servidor HTTP de prueba simulando la API, nunca llamadas reales en los tests).
- **Qué hace:** busca por título/año en `es-ES`, pide carteles en varios idiomas a la vez y aplica la prioridad `es-ES → español genérico → inglés → sin idioma` (ver `docs/es/metadatos.md`), cachea resultados e imágenes en disco respetando el límite de conservación de TMDB (revisar los términos vigentes antes de implementar el TTL de caché — si hay duda, parar y preguntar, no inventar un plazo). La clave de API se lee de configuración (Paso 1), nunca en el repositorio.
- **Criterio de terminado:** tests contra el servidor simulado cubren: búsqueda con resultado, sin resultado, selección de cartel por prioridad de idioma, caché que evita una segunda llamada para el mismo ítem dentro del TTL.
- **Ejecutor:** AgentRelay, esfuerzo medio.
- **Dependencias:** Paso 4.
- **Versión/CHANGELOG:** `0.1.0-dev.5` — Añadido: cliente de metadatos TMDB con caché y prioridad de idioma.

## Paso 6 — Contrato OpenAPI y API versionada

- **Archivos:** `api/openapi.yaml`, handlers HTTP bajo `/api/v1/` que implementan el contrato (biblioteca: listar, ficha de un ítem; auth del Paso 3).
- **Qué hace:** define el contrato primero (OpenAPI 3.x) para los endpoints ya implementados (`/auth/login`, `/auth/logout`, `GET /api/v1/libraries`, `GET /api/v1/items`, `GET /api/v1/items/{id}`) y los handlers que lo cumplen. Versión de la API en la ruta (`/api/v1/`), coherente con `VERSION` del proyecto pero independiente de ella.
- **Criterio de terminado:** el YAML valida con una herramienta estándar (p. ej. `swagger-cli validate` o equivalente); tests de integración HTTP que llaman a cada endpoint y comprueban el esquema de respuesta.
- **Ejecutor:** AgentRelay, esfuerzo medio.
- **Dependencias:** Pasos 3, 4, 5.
- **Versión/CHANGELOG:** `0.1.0-dev.6` — Añadido: contrato OpenAPI y API REST versionada.

## Paso 7 — Reproducción directa y remux HLS fMP4

- **Archivos:** `internal/playback/decision.go`, `internal/playback/hls.go`, endpoint `GET /api/v1/items/{id}/play`.
- **Qué hace:** implementa solo los dos primeros escalones de la cadena de decisión de `docs/es/reproduccion.md`: reproducción directa (servir el archivo tal cual, con soporte de rangos HTTP) y remux a HLS con segmentos fMP4 cuando el contenedor no es compatible (invocando FFmpeg como proceso externo, **argumentos siempre en lista, nunca por concatenación de texto** — ver `docs/es/arquitectura.md` punto 9). Genera URLs de reproducción firmadas y de vida corta (no tokens permanentes en la URL). Transcodificación parcial/completa queda para la Fase 3: aquí, si ninguno de los dos primeros escalones vale, se devuelve un error claro, no se intenta transcodificar.
- **Criterio de terminado:** test que, con un vídeo de prueba generado por el propio test con FFmpeg (nunca un binario subido al repo — ver `docs/es/arquitectura.md`), verifica reproducción directa cuando el contenedor es compatible y remux cuando no; test que verifica que una URL de reproducción expirada se rechaza; test que verifica que los argumentos a FFmpeg se pasan como lista (no hay construcción de comando por concatenación en ningún punto del código).
- **Ejecutor:** AgentRelay, esfuerzo alto (seguridad de procesos externos: revisar con cuidado antes de aceptar).
- **Dependencias:** Paso 6.
- **Versión/CHANGELOG:** `0.1.0` — Añadido: reproducción directa y remux HLS fMP4. Esto cierra la Fase 1: versión pública `0.1.0`, con entrada en `CHANGELOG.md` que resume todo lo añadido en los pasos 1-7.

## Al cerrar la fase

- El paso 7 sube la versión a `0.1.0` (menor: primera funcionalidad real, compatible con la base 0.0.x de solo documentación) y consolida el CHANGELOG.
- Revisar que `README.md`/`README.en.md` ya no diga «solo documentación, todavía no hay código».
- Parar y pedir el visto bueno del usuario antes de empezar la Fase 2 (cliente web), como pide el prompt de arranque del proyecto.

## Cómo seguir en barato

Tras revisar y aceptar este plan: `/clear` → `/model sonnet` (ya es el modelo activo) → «ejecuta el paso 1 de fase-1-servidor-minimo.PLAN.md con AgentRelay». Cada paso siguiente se lanza igual, en orden, revisando el diff antes de aceptar cada uno.

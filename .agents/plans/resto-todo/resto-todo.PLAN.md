# Resto del TODO (salvo DLNA)

Encargo del usuario (2026-11-03): «Menos DLNA por ahora puedes hacer todo el resto ya». Orden aceptado: audio, duplicado de subtítulo en TV, fotos, intro/créditos, descargas, sincronización, complementos.

**Fuera de este plan (decidido):** DLNA (lo pospone el usuario); otras plataformas (Apple, Samsung/LG), porque `CLAUDE.md` fija «web, Android, Android TV, nada más por ahora»; televisión en directo y contenido de internet (producto aparte, decisión del usuario).

## Reglas para cada paso

- Ejecución secuencial en el mismo árbol. Delegar con AgentRelay (`agentrelay.cmd run %TEMP%\<tarea>.json`; la validación del ejecutor con `./gradlew` falla en Windows: validar a mano con `cd android && cmd /c gradlew.bat testDebugUnitTest assembleDebug` y aceptar con `--force` si el resto es correcto).
- Tras cada paso: revisar el diff, ejecutar `go vet ./internal/...`, `go test ./internal/... -count=1` (FFmpeg en el PATH), `cd web && npm run check && npm test`, y Gradle si se toca Android. Commit por nombre de archivo, **sin push**.
- La sesión (no el ejecutor) sube `VERSION`, `web/package.json`, `web/package-lock.json`, CHANGELOG es/en, TODO es/en, `docs/es/estado.md` y `docs/en/status.md` en el commit que cierra el paso, y crea la etiqueta local `vX.Y.Z`. Docs de usuario/API es/en: en la misma tarea del ejecutor o en una tarea de docs aparte.
- Archivos en CRLF (salvo README/TODO): no cambiar finales de línea. Comentarios en castellano y solo con el porqué.
- No afirmar en docs que algo está probado en un dispositivo si no lo está.
- Ante un imprevisto de diseño: parar y anotarlo aquí.

## Pasos

### 1. [x] Audio AC-3/E-AC-3 copiado en el HLS — 0.60.0 (menor)

Hoy (`internal/playback/hls.go`, `transcodeAudioArgs`) el HLS siempre recodifica el audio a AAC 2.0 o 5.1. Diseño:
- Si el cliente es nativo (`caps.Native()`) y declara el códec de la pista elegida (`ac3` o `eac3` en `internal/playback/native.go`), el HLS usa `-c:a copy` para esa pista. Nunca para DTS/TrueHD (MPEG-TS/HLS y ExoPlayer no lo admiten bien), ni para la web, ni si la pista no es AC-3/E-AC-3.
- Va en `TranscodeSettings` como un campo nuevo (p. ej. `AudioCopy bool`) que entra en `streamKey` (sufijo propio) para no mezclar cachés; se decide junto a `AudioChannelsFor`.
- Tests: argumentos de FFmpeg con y sin copia, clave de stream distinta, y un test real en `hls_real_test.go` (AC-3 5.1 de prueba generado con FFmpeg → el primer segmento conserva `ac3` de 6 canales).
- Android: comprobar qué tokens declara la app (buscar `ac3` en `android/`); solo declara `ac3`/`eac3` si `AudioCapabilities` del equipo los anuncia (la TV de casa no los anuncia: seguirá recibiendo AAC). No cambiar eso si ya es así.
- Docs: `docs/es/reproduccion.md`/`docs/en/playback.md` (quitar «passthrough sin recodificar» de lo que falta, explicar cuándo se copia).
- Ejecutor: AgentRelay, esfuerzo alto. CHANGELOG: «Servidor: el HLS copia el audio AC-3/E-AC-3 sin recodificar para las apps que lo declaran.»

### 2. [x] TV: no duplicar el subtítulo de imagen en reproducción directa — 0.60.1 (parche)

En directa, ExoPlayer ya expone los VobSub/PGS del MKV como pistas de texto nativas y `TrackDialog` (`android/app/src/main/java/.../PlayerScreen.kt`) añade además la sección «Subtítulos de imagen» (que recarga por HLS quemando). Diseño: en modo directo, ocultar en «Subtítulos de imagen» cada pista de imagen del servidor para la que ExoPlayer ya ofrece una pista de texto con un MIME de imagen (`application/vobsub`, `application/pgs`, `application/dvbsubs`) y el mismo idioma; si no hay coincidencia, mostrarla como hoy. En HLS no cambia nada. Lógica pura en una función testeable (`PlayerViewModel` o archivo propio) con tests unitarios.
- Ejecutor: AgentRelay, esfuerzo medio. CHANGELOG: «Android TV: en reproducción directa ya no aparece dos veces el mismo subtítulo de imagen.»

### 3. [x] Bibliotecas de fotos — 0.61.0 (menor)

Ya existen los tipos `video`, `clips` y `music` (`libraries.kind`, `internal/library/access.go`). Nuevo tipo `photos`, siguiendo el patrón de `music*.go` y `clips*.go`:
- **3a. Servidor** (AgentRelay, alto): migración nueva (tabla `photos`: id, library_id, path, relpath de carpeta, nombre, tamaño, mtime, ancho, alto, fecha de captura si se puede leer del EXIF con código propio mínimo y sin dependencias nuevas, si no mtime); escaneo de JPEG, PNG, WebP y GIF (sin HEIC/RAW: necesitarían dependencias o FFmpeg; anotarlo en docs); miniaturas generadas con FFmpeg (ya es dependencia del sistema) y cacheadas en el directorio de caché existente; API `GET /api/v1/photos` (por carpeta, paginada, ordenable por fecha/nombre), `GET /api/v1/photos/{id}/thumb`, `GET /api/v1/photos/{id}/original`; permisos por biblioteca como los demás tipos; crear biblioteca de tipo fotos en administración; `api/openapi.yaml` y `apicontract`.
- **3b. Web** (AgentRelay, medio): entrada «Fotos» en el menú si hay alguna biblioteca de fotos; vista por carpetas con cuadrícula de miniaturas y visor a pantalla completa con flechas/teclado y presentación automática; selector de tipo en administración. Estilo «Marquesina».
- **3c. Android/TV** (AgentRelay, medio): sección Fotos con cuadrícula y visor (mando en TV), solo si el servidor tiene bibliotecas de fotos.
- **3d. Docs** es/en (guía de usuario, administración, API, android). Cierra 0.61.0.

### 4. [x] Saltar intro y créditos — 0.62.0 (menor)

Diseño sin detección por huella de audio (cara y frágil):
- Servidor: tabla de segmentos por ítem (`item_segments`: item_id, tipo `intro`|`credits`, inicio, fin). Fuentes: (1) capítulos del MKV cuyo título coincide (sin mayúsculas ni acentos) con intro/opening/apertura/cabecera o credits/créditos/ending/final, leídos con `ffprobe -show_chapters` (ya se piden en `fileinfo.go`; hoy solo se cuentan); (2) edición manual del administrador. API: `segments` en la respuesta de `/play` y `PUT /api/v1/items/{id}/segments` (solo admin). Para episodios, un botón de administración «aplicar a toda la temporada» copia la marca de un episodio a los demás con los mismos tiempos.
- Web y Android/TV: botón «Saltar intro» durante el segmento intro y «Siguiente episodio» (o «Saltar créditos» en películas) en el segmento de créditos; en TV, el botón recibe el foco al aparecer.
- Pasos: 4a servidor (alto), 4b web (medio), 4c Android (medio), 4d docs.

### 5. [x] Descargas sin conexión en Android — 0.63.0 (menor)

- Solo archivos que la app reproduciría en directo (mismo criterio que `/play` con las capacidades de la app). Si el archivo necesita HLS, el botón explica que no se puede descargar (no se genera una versión recodificada en esta versión).
- Descarga con el `DownloadManager` del sistema (sin dependencias nuevas; decidido en vez de `WorkManager`) a la carpeta privada de la app (`getExternalFilesDir`) usando la ruta autenticada `GET /api/v1/items/{id}/download` (paso 5a), con progreso, cancelar y borrar; pantalla «Descargas» en el móvil (no en TV). Reproducción local con el mismo reproductor; el progreso se guarda en local y se envía al servidor al volver la conexión.
- Servidor: si hace falta, una ruta de descarga autenticada con soporte de rangos (revisar si la URL de directa ya sirve; dura 8 h).
- Pasos: 5a servidor si procede (medio), 5b Android (alto), 5c docs.

### 6. [-] Sincronización entre dispositivos — descartada por ahora (el usuario, 2026-11-03)

El progreso y el estado de visto ya viven en el servidor y se comparten entre dispositivos. Antes de implementar, preguntar al usuario qué quiere decir: (a) mandar la reproducción de un dispositivo a otro («reproducir en la TV» desde el móvil) o (b) ver lo mismo a la vez en varios equipos. No implementar sin respuesta.

### 7. [x] Complementos — diseño (Opus) — sin versión

Solo documento de diseño en `docs/es/complementos.md` y `docs/en/plugins.md`, que escribe un modelo potente: qué puede extender un complemento (fuentes de metadatos, acciones tras escanear, notificaciones), cómo se aísla (proceso externo con protocolo JSON por stdin/stdout, sin cargar código en el servidor, sin red salvo lo declarado), permisos y riesgos. Implementar solo tras la aprobación del usuario.

Fotos y complementos quedan aparcados por el usuario (2026-11-03): no ampliar ni implementar sin hablarlo antes.

Imprevistos de diseño:

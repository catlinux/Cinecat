# Descargas sin conexión a partir del HLS del servidor

Encargo del usuario (2026-11-04): las descargas del archivo original fallan por compatibilidad (MKV con AC-3/DTS, etc.). Si el archivo se ve en streaming, debe poder verse descargado con el mismo reproductor. Decisión: **descargar el mismo HLS que se reproduce en streaming** (Media3 offline) en lugar del archivo original. Así lo que se ve en streaming se ve sin conexión, sin lógica de compatibilidad propia.

Metodología habitual del proyecto: AgentRelay (o la propia sesión si los ejecutores no tienen cuota), revisión del diff, tests (`go vet`/`gofmt`/`go test ./internal/...` con FFmpeg en el PATH; `cd android && cmd /c gradlew.bat testDebugUnitTest assembleDebug -q`), commit por nombre de archivo, versión + CHANGELOG/TODO/docs es/en en el commit que cierra cada paso, etiqueta local, **sin push sin aprobación**. Archivos en CRLF salvo los `.go` del árbol de trabajo (LF; `gofmt -l` debe salir vacío).

## Hechos comprobados (no volver a investigar)

- `GET /api/v1/items/{id}/play` devuelve `url` firmada (`exp`, `sig`; TTL `DefaultURLTTL` = 8 h en `internal/playback/service.go`). En modo `remux` la URL es `/api/v1/stream/{id}/playlist.m3u8?exp&sig&caps&audio&q[&burnsub]` (`internal/library/http.go`, `handlePlay`).
- Con la duración conocida, la playlist es VOD completa (`writeVODPlaylist`, `#EXT-X-ENDLIST`) y cada segmento (`.ts`) se genera bajo demanda (`waitForSegment`/`checkSegmentLocked` en `internal/playback/hls.go`). `signPlaylist` añade la misma query firmada a cada segmento.
- Un proceso de transcodificación ocupa un cupo de `playback.max_transcodes`; `reserveTranscodeSlotLocked` expulsa al proceso con la petición más antigua. Las copias tienen su propio cupo (`hlsMaxCopyProcesses`). `monitorLocked` para procesos inactivos 90 s o adelantados más de 20 segmentos.
- Los subtítulos de texto se sirven como WebVTT firmados con la misma firma (`handleStreamSubtitle`); la lista viene en `subtitles` de `/play`. Los de imagen solo se ven quemados (`burnsub`), y entonces ya van dentro del HLS.
- Android ya usa `media3-exoplayer`, `media3-exoplayer-hls` y `media3-datasource-okhttp` 1.9.4 (`android/gradle/libs.versions.toml`). Las descargas actuales (0.63.0–0.65.2) usan el `DownloadManager` del sistema sobre el archivo original: `DownloadRepository.kt`, `DownloadRegistry`, `DownloadFormatting.kt`, `DownloadButton.kt`, `DownloadsScreen.kt`, reproducción local con `Player(itemId, startSeconds, localUri)` en `MainShell.kt`. En el móvil del usuario no se ha completado ninguna descarga con ese sistema.

## Diseño (decidido)

**Servidor**
- `/play?download=1`: fuerza el modo `remux` (HLS; copia de vídeo cuando se pueda, igual que hoy) aunque el archivo admita directo, firma con un TTL de descarga largo (nuevo ajuste `playback.download_url_ttl`, por defecto 72 h) y añade `dl=1` a la query firmada que se propaga a los segmentos (`signedQuery` en `ServeHLSFile`).
- Prioridad: un proceso arrancado por una petición con `dl=1` es de descarga. Nunca expulsa a un proceso de reproducción; los de reproducción expulsan primero a los de descarga. Como máximo **1** proceso de transcodificación de descarga a la vez (las copias no cuentan). Si no hay cupo, el segmento responde **503 con `Retry-After: 30`** (Media3 reintenta). La entrada HLS (caché de segmentos) se comparte con la reproducción normal si coinciden ítem, audio, calidad y capacidades.
- Lo demás de `/play` no cambia. `download_direct`/`download_blocked` y `GET /items/{id}/download` se conservan en la API (informativos; las apps dejan de usarlos).

**Android (solo móvil y tableta; la TV sigue sin descargas)**
- Media3 offline: `SimpleCache` propio en `filesDir/offline-hls` con `NoOpCacheEvictor` y `StandaloneDatabaseProvider`; `androidx.media3.exoplayer.offline.DownloadManager` con `OkHttpDataSource.Factory` sobre el cliente OkHttp de la app; `CinecatDownloadService : DownloadService` con notificación (`DownloadNotificationHelper`), `foregroundServiceType="dataSync"` y los permisos `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_DATA_SYNC` y `POST_NOTIFICATIONS` (pedido al iniciar la primera descarga en Android 13+; sin él la descarga sigue igualmente).
- **Clave de caché sin firma**: un `CacheKeyFactory` que use la URL sin `exp` ni `sig` (resto de parámetros ordenados). Así una URL re-firmada reutiliza lo ya descargado.
- Descargar: `/play?download=1` con las capacidades y el perfil del equipo (misma pista de audio y calidad que elegiría al reproducir) → `DownloadRequest(id = itemId, uri = playlist absoluta, mimeType = APPLICATION_M3U8, data = metadatos JSON: título, etiqueta de serie, subtítulos)` → `DownloadService.sendAddDownload`. Los subtítulos de texto de `/play` se bajan aparte con OkHttp a `filesDir/offline-subs/<itemId>/` (pequeños).
- Firma caducada: si una descarga falla con 403 o se reanuda tras caducar, pedir de nuevo `/play?download=1` y volver a añadir la petición con el **mismo id** y la URL nueva. Verificar en el código de Media3 1.9.4 que `DownloadRequest.copyWithMergedRequest` adopta la URI nueva; **si no la adopta, parar y anotarlo en «Imprevistos»** (la alternativa sería un `DataSource` que re-firme, a decidir por un modelo potente).
- Reproducir sin conexión: `MediaItem` de `download.request.toMediaItem()` con `CacheDataSource.Factory` de solo lectura sobre la misma caché y la misma `CacheKeyFactory`, más `SubtitleConfiguration` de los VTT locales. Se mantienen el guardado de progreso pendiente y la sincronización al volver la conexión.
- UI: mismos textos y estados que hoy (Descargar, Preparando, Descargando n %, Descargado, Error), más «En espera del servidor» cuando Media3 está reintentando. Desaparece el aviso de incompatibilidad. Pantalla Descargas: lista desde el índice de Media3 con tamaño descargado, reproducir y borrar.
- Migración: se elimina el sistema anterior (registro JSON, `DownloadManager` del sistema, archivos originales). Al primer arranque de la versión nueva se borran sus archivos y su registro.

## Pasos

1. [x] **Servidor: `/play?download=1`, TTL de descarga y prioridad baja.** Archivos: `internal/library/http.go`, `internal/playback/hls.go`, `internal/playback/service.go`, ajuste nuevo en `internal/config` (y donde se cargan los ajustes de reproducción), `api/openapi.yaml`, tests en `internal/playback` e `internal/library`. Terminado: tests de `download=1` (fuerza remux, TTL largo, `dl=1` en playlist y segmentos), de la expulsión (reproducción expulsa a descarga y no al revés), del máximo de 1 transcodificación de descarga y del 503 con `Retry-After`; `go vet`, `gofmt -l` y `go test ./internal/...` en verde. Ejecutor: AgentRelay o la sesión, Sonnet, esfuerzo alto. Versión **0.66.0** (menor: parámetro y ajuste nuevos). CHANGELOG «Añadido» + `docs/es/api.md`/`docs/en/api.md`, `docs/*/reproduccion|playback.md` (prioridad de descargas), configuración documentada donde estén los demás ajustes de `playback`.
2. [x] **Android: infraestructura Media3 offline.** Archivos: nuevo `OfflineDownloads.kt` (caché, `DownloadManager`, `CacheKeyFactory`), `CinecatDownloadService.kt`, `AppContainer.kt`, `AndroidManifest.xml`, `DownloadRepository.kt` reescrito (iniciar, re-firmar, borrar, flujo de estados), `PlaybackModels.kt`/`CinecatService.kt` si hace falta el parámetro `download`, tests unitarios de la clave de caché, del mapeo de estados de Media3 a la UI y de los metadatos de la petición. Solo se añaden módulos Media3 de la misma versión 1.9.4 si hacen falta (p. ej. `media3-database`). Terminado: compila, pasan los tests y la verificación de `copyWithMergedRequest` queda anotada en el informe. Ejecutor: Sonnet, esfuerzo alto. Sin versión (se cierra en el paso 3).
3. [x] **Android: UI, reproducción sin conexión y migración.** Archivos: `DownloadButton.kt`, `DownloadsScreen.kt`, `DownloadFormatting.kt` (quitar lo que ya no se use), `MainShell.kt` (ruta de reproducción local por id de descarga), `PlayerScreen.kt`/`PlayerViewModel.kt` (fuente desde la caché + subtítulos locales), `strings.xml`, migración del sistema anterior; tests. Docs es/en: `android.md`, `guia-de-usuario.md`/`user-guide.md`, `estado.md`/`status.md`, `api.md` (marcar `download_direct` como informativo). Terminado: compila y pasan los tests; texto de ayuda explica que la descarga es la misma versión que se ve en streaming y que el servidor la prepara (puede tardar si hay que convertirla). Ejecutor: Sonnet, esfuerzo alto. Versión **0.67.0** (menor).
4. [x] **Prueba en el Pixel** (pedir permiso al usuario antes de usar el móvil): descargar «El médico» (MKV con audio que el móvil no decodifica), ver progreso y notificación, reproducir en modo avión con audio y subtítulos, cerrar la app a mitad de descarga y reanudar, borrar. Comprobar en el servidor que una reproducción simultánea no se corta. Corregir lo que falle y cerrar **0.67.x**. Ejecutor: la propia sesión, Sonnet, esfuerzo medio.

## Fuera de este plan

- Elegir calidad o pista de audio antes de descargar (se usa la que se elegiría al reproducir). Anotar en TODO.
- Descargas en la web o en la TV.
- Límite de espacio en el móvil (lo gestiona el usuario borrando).

## Imprevistos de diseño

### Informe del paso 2 (2026-11-04)

- `DownloadRequest.copyWithMergedRequest` (bytecode de Media3 1.9.4) adopta la URI de la petición **nueva**; `DownloadRequest` no tiene `buildUpon`, así que la re-firma construye una petición nueva con el mismo id, tipo, claves, `customCacheKey` y `data`.
- `DownloadRepository` conserva la API que usa la UI actual (`start`, `status`, `delete`, `hasLocalFile`, `localUri`…) para que el paso 3 la adapte. `localUri` devuelve ahora la URI del HLS de la petición (no el archivo local) y `DownloadStartResult.NotDownloadable` ya no se produce: el paso 3 debe reemplazar la reproducción local por la fuente de caché y quitar ambos.
- Re-firma automática: al abrir la app (descargas sin terminar) y una vez por sesión cuando una descarga falla.
- Pendiente para el paso 3: pedir `POST_NOTIFICATIONS` al iniciar la primera descarga (Android 13+); estado «En espera del servidor» (Media3 no lo expone; hay que deducirlo de «descargando» sin avance); migración de `downloads.json` antiguo y archivos del `DownloadManager` del sistema; cadena `download_channel_name` solo existe en castellano como el resto de `strings.xml`.
- El servicio usa `getScheduler() = null`: si el sistema mata el proceso, las descargas se reanudan al abrir la app.

### Informe del paso 3 (2026-11-04)

- Hecho por la propia sesión (Sonnet). Versión 0.67.0. La ruta `Player` conserva `localUri` (ahora la URI del HLS de la petición) y el VM recibe además `localSubtitles`; el reproductor usa `OfflineDownloads.playbackDataSourceFactory` (caché de solo lectura con la clave sin firma y `DefaultDataSource` como origen, para los VTT `file://`) cuando es local.
- `DownloadRepository.statuses` (StateFlow compartido) sustituye al sondeo por botón; «En espera del servidor» = fase Running sin avance de bytes durante 5 sondeos (`isWaitingForServer`).
- Migración: los registros con `fileName` (sistema anterior) se cancelan en el `DownloadManager` del sistema, se borra su archivo y se quita el registro, tras volcar el progreso pendiente. `DownloadRecord.downloadId`/`fileName` se conservan solo para esto.
- Pendiente (paso 4): probar en el Pixel; no hay prueba automática de la reproducción desde caché ni de la migración.

### Informe del paso 4 (2026-11-05)

- Prueba en el Pixel 10 Pro con la 0.67.2 y el servidor ya en 0.67.2 (con el servidor en 0.65.2 la descarga falla con «Input does not start with the #EXTM3U header», porque ignora `download=1`).
- «El médico» (1,5 GB): descarga con progreso en el botón y en la notificación, cerrar la app a mitad (force-stop) y reanudar al reabrir sin empezar de cero, «Descargado», reproducción **en modo avión** desde Descargas (vídeo avanza, `AudioTrack` activo) y borrado.
- Una descarga que había fallado con el servidor antiguo se reanudó sola con el servidor nuevo (re-firma) y terminó.
- Fallo corregido en 0.67.3: tras aceptar el permiso de notificaciones la descarga no arrancaba (la ficha se recarga al volver del diálogo y el ámbito de la composición cancelaba el inicio); ahora se lanza en el ámbito de la app (`DownloadRepository.startAsync`).
- No probado: subtítulos de texto sin conexión, reproducción simultánea con descarga en el servidor.

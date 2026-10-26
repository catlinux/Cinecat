# Segmentación progresiva de HLS (corrige el diseño del Paso 2 de la Fase 3)

## Por qué existe este plan

El Paso 2 de la Fase 3 (commit `e07f5d5`) y su fix posterior (`10f4e15`) arreglaron **qué** vídeos necesitan transcodificarse, pero no **cómo** se sirven: `Service.prepare()` (`internal/playback/hls.go`) lanza `cmd.Run()` de forma **bloqueante** y espera a que FFmpeg procese el archivo **completo** antes de devolver la playlist. Para una película de ~1h con `libx264 veryfast` en software, esto tarda demasiado para ser usable — confirmado en producción con el usuario esperando 25+ minutos sin que terminara.

Investigado el código real de Jellyfin (`jellyfin/jellyfin`, `Jellyfin.Api/Controllers/DynamicHlsController.cs`, método `GetDynamicSegment` y `GetSegmentResult`, revisado línea a línea desde GitHub el 2026-10-26) para no reinventar algo ya resuelto en producción por un proyecto maduro. Su patrón real:

1. La playlist servida al cliente es tipo **`event`** (crece progresivamente), no `vod` completo de golpe, mientras FFmpeg sigue trabajando.
2. FFmpeg arranca con `-start_number <segmentId>` apuntando al segmento que el cliente pidió (permite saltos/seeks sin regenerar desde el principio).
3. Cada petición de segmento (`GetDynamicSegment`) primero comprueba si el archivo ya existe en disco: si existe, lo sirve sin tocar FFmpeg.
4. Si no existe, arranca o mantiene vivo un proceso FFmpeg y **espera activamente en un bucle corto (100ms)** a que el archivo del segmento concreto aparezca — nunca espera al archivo completo.
5. Un segmento se considera "listo para servir" cuando existe Y (FFmpeg ya terminó O el siguiente segmento también existe ya) — evita servir un `.m4s` a medio escribir.
6. Hay un `TranscodeManager`/lock por playlist para que peticiones simultáneas de segmentos del mismo ítem no disparen FFmpeg dos veces ni se pisen.

Cinecat es mucho más pequeño en alcance (sin perfiles de dispositivo, sin live TV) así que no hace falta portar todo Jellyfin — solo el patrón productor/consumidor de la sección 3-5, adaptado a lo que ya existe en `internal/playback`.

## Decisiones ya fijadas

- FFmpeg como proceso externo, argumentos siempre en lista (nunca shell) — ya establecido en todo `internal/playback/hls.go`, mantener.
- `-hls_time 6` ya es el valor usado en remux y transcodificación — mantener.
- El caché de `entries` por `itemID` ya existe (`Service.entries`, protegido por `s.mu`) — se amplía, no se sustituye.
- El fix de 0.8.1 (forzar transcodificación en `.avi` siempre) sigue siendo correcto y no se toca en este plan: el problema de fondo es el modo de *servir*, no de *decidir* cuándo transcodificar.

## Paso 1 — Arrancar FFmpeg en segundo plano y servir segmentos según se generan

- **Archivos:** `internal/playback/hls.go`, `internal/playback/service.go`, tests correspondientes.
- **Qué hace:** sustituye `cmd.Run()` (bloqueante) por `cmd.Start()` + una goroutine que espera el resultado y marca el job como terminado (guardar el `*exec.Cmd` y un canal/flag de "ha terminado" en `hlsEntry`, con el `stderr` capturado igual que ahora para diagnosticar fallos). El handler HTTP que sirve un segmento concreto (`serveSigned`/`handleStreamFile` en `internal/library/http.go`, revisar sin tocar más de lo necesario) debe, tras obtener la `hlsEntry`, esperar en un bucle corto (p. ej. cada 100-200ms, con un timeout razonable total, p. ej. 2-3 minutos) a que el archivo del segmento pedido exista en disco, igual que hace Jellyfin, en vez de asumir que ya existe porque `prepare()` ya retornó.
- **Por qué:** es el cambio mínimo que hace que el primer segmento esté disponible en segundos (lo que tarde FFmpeg en generar los primeros 6s de vídeo) en vez de esperar a la película entera.
- **Criterio de terminado:** test de integración: arrancar `prepare()` para un archivo que tarda en transcodificar (usar un archivo de prueba largo generado con FFmpeg), y comprobar que el primer segmento (`segment_00000.m4s`) está disponible mucho antes de que el proceso FFmpeg completo termine (medir con un mock o con tiempos reales si hay FFmpeg disponible). Un segundo ítem distinto pedido en paralelo no debe bloquearse por el primero.
- **Ejecutor:** AgentRelay, esfuerzo alto (concurrencia, hay que acertar el ciclo de vida del proceso FFmpeg en segundo plano sin fugas ni condiciones de carrera).
- **Versión/CHANGELOG:** menor — Corregido: la transcodificación ya no bloquea hasta generar el vídeo completo; el primer segmento se sirve en cuanto existe.

## Paso 2 — Playlist que crece mientras FFmpeg trabaja (opcional, evaluar necesidad real)

- **Qué hace:** si tras el Paso 1 el cliente web (hls.js) sigue sin poder arrancar por pedir la playlist completa antes de que exista el primer segmento, cambiar `-hls_playlist_type vod` a sin especificar tipo (o gestionar manualmente una playlist `event` que se reescribe según aparecen segmentos), siguiendo el patrón de Jellyfin. **Evaluar primero si el Paso 1 ya basta**: como Cinecat sirve la playlist completa (conocemos la duración total del vídeo de antemano, a diferencia de live TV), puede bastar con que la playlist liste todos los segmentos esperados desde el principio (duración total conocida) y que cada segmento individual tarde en aparecer; no necesariamente hace falta una playlist `event` creciente. Decidir con una prueba real antes de construir algo más complejo de lo necesario.
- **Ejecutor:** AgentRelay, esfuerzo medio — solo si el Paso 1 no resuelve el problema por sí solo.
- **Dependencias:** Paso 1, y confirmación de que hace falta.

## Al cerrar

Probar en real contra el caso que falló (`1408.avi`, item id 2 en producción) y confirmar con el usuario que el tiempo hasta el primer fotograma es razonable (segundos, no minutos). Actualizar `docs/es/reproduccion.md`/`docs/en/playback.md` con el diseño real de servido progresivo. Actualizar `TODO.md`/`TODO.en.md` quitando cualquier referencia a que la transcodificación es bloqueante.

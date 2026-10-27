# Reproducción y códecs

[English](../en/playback.md) · [Volver al README](../../README.md) · [Estado](estado.md)

La reproducción está implementada para la web con dos caminos: **directa** para MP4/M4V y **HLS** para el resto. La decisión por capacidades del dispositivo, H.265 y la aceleración por hardware siguen siendo diseño futuro.

## Decisión actual

| Situación | Modo |
|---|---|
| `.mp4` o `.m4v` | Reproducción directa: se sirve el archivo con soporte de rangos HTTP. |
| `.mkv`, `.avi`, `.mov`, `.ts`, `.m2ts` | HLS: se recodifica a H.264 + AAC estéreo y se sirve por segmentos. |
| Cualquier otra extensión | 415; el contenedor no se admite. |

La decisión se basa en la extensión. No se comprueban los códecs reales del archivo ni las capacidades del navegador antes de elegir. La API sigue llamando `remux` al modo HLS aunque recodifique.

## HLS: cómo funciona

- Vídeo: `libx264` con preset y CRF configurables, `yuv420p`, escala hasta la altura máxima configurada.
- Audio: una pista, AAC estéreo a 160 kbps.
- Keyframes forzados cada 6 s y segmentos MPEG-TS de 6 s.
- `-output_ts_offset` fijo de 10 s para que los segmentos de procesos distintos encajen.
- La playlist es un VOD completo si `ffprobe` conoce la duración. Si no, se usa la playlist `event` de FFmpeg y los saltos lejanos no son rápidos.
- FFmpeg no arranca al pedir la playlist: arranca con la primera petición de segmento, en el punto pedido.
- Si se pide un segmento más de 24 s por delante de lo generado, o un hueco hacia atrás, el servidor reinicia FFmpeg en ese segmento (`-ss` + `-start_number`).
- Un segmento se considera completo cuando aparece en la playlist del proceso; al reiniciar se descartan el que quedó a medias y los que se van a reescribir.

## Límites de procesos

| Límite | Valor |
|---|---|
| Espera de la playlist inicial | 20 s |
| Espera de un segmento | 2 min |
| Vida máxima de un proceso | 4 h |
| Parada por inactividad | 90 s sin peticiones |
| Segmentos generados por delante | 20 (~2 min) |
| Transcodificaciones simultáneas | `playback.max_transcodes`, 2 por defecto |
| Limpieza de trabajos HLS | 1 h sin uso, revisión cada 15 min |

Si se alcanza el máximo de transcodificaciones, se detiene el trabajo con más tiempo sin peticiones. Los procesos van con prioridad baja y número de hilos limitado.

## Audio, subtítulos y calidad

- **Audio:** se elige por posición en la lista de pistas de `ffprobe`. La preferencia del usuario y de la película se guarda en `item_preferences`.
- **Subtítulos:** se leen pistas de texto incrustadas y archivos externos `.srt`, `.vtt` o `.ass` junto al vídeo, con sufijos de idioma y `forced`. Se convierten a WebVTT. Las pistas de imagen (PGS/VobSub) no se admiten y el HLS no lleva una representación separada de subtítulos.
- **Calidad:** en HLS se ofrecen `auto`, 480p, 720p, 1080p (sin superar la altura original ni `playback.max_height`) y `original`. `original` no copia el archivo original: limita la recodificación a la altura original. En directa solo hay `original`.
- **Preferencias:** audio, subtítulo y calidad por usuario y película; el reproductor las guarda y recarga en el segundo actual al cambiarlas.

## Reproductor web

- `<video>` nativo para reproducción directa; hls.js 1.7.3 para HLS.
- Controles propios: play/pausa, ±10 s, barra de progreso, volumen, silencio, velocidad (0,5×–2×), audio, subtítulos, calidad, imagen en imagen, pantalla completa y siguiente episodio.
- Atajos: `Espacio`/`K`, `J`/`L` o flechas, `M`, `F`, `C`, `,`/`.`, `0`–`9` y `Esc`. El detalle está en la [guía de usuario](guia-de-usuario.md).
- El progreso se envía cada 15 s y al salir; al 90 % se marca como visto y se borra el progreso. Si quedan 15 s o menos, no se reanuda.
- Al terminar un episodio con siguiente, aparece una cuenta atrás de 8 s.

## URLs firmadas

- HMAC-SHA256 con una clave aleatoria de 32 bytes por proceso.
- TTL por defecto: 8 horas.
- La firma cubre el identificador y la caducidad; no está ligada al usuario. Cualquiera con la URL puede usarla hasta que expire.
- Al reiniciar el servidor cambia la clave y las URLs anteriores dejan de valer.

## Qué falta para la reproducción avanzada

- Perfiles de capacidades por dispositivo y elección entre directa, copia y transcodificación.
- Copiar el vídeo cuando ya es H.264 8 bits, con corte en keyframes reales.
- Aceleración por hardware (Quick Sync/VAAPI); el servidor tiene `/dev/dri`, pero no se pasa al contenedor.
- H.265, HDR/10 bits con tone mapping, desentrelazado, audio multicanal o passthrough y subtítulos de imagen.
- ABR y playlist maestra con varias calidades.
- Un modo de API `transcode` explícito.

El orden de estas mejoras está en [estado](estado.md). La relación con Reduvid se mantiene: convertir antes los archivos con [Reduvid](https://github.com/catlinux/reduvid) reduce la necesidad de transcodificar.

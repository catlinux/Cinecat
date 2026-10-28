# Reproducción y códecs

[English](../en/playback.md) · [Volver al README](../../README.md) · [Estado](estado.md)

Cinecat elige entre tres caminos para cada archivo: **directa**, **copia** y **recodificación**. La decisión usa la ficha real de `ffprobe` (códec, perfil, profundidad de color, altura y pistas) y lo que declara el navegador en la cookie `cinecat_caps`. La API sigue llamando `remux` al modo HLS, tanto si copia como si recodifica; la ficha del archivo distingue el método con `direct`, `copy` o `transcode`.

## Decisión actual

| Caso | Modo |
|---|---|
| MP4/M4V con H.264 de 8 bits (`yuv420p`/`yuvj420p`, sin High 10, 4:2:2 ni 4:4:4) y pista de audio por defecto AAC, MP3 o sin audio | Directa: se sirve el archivo con soporte de rangos HTTP. |
| MP4/M4V con H.265 que el navegador reproduce directamente: 8 bits con `hevc`, 10 bits con `hevc10`, y audio AAC, MP3 o sin audio | Directa. |
| MP4/M4V con H.265 que no se reproduce directo, pero el navegador lo admite por MSE: 8 bits con `hevcmse`, 10 bits con `hevc10mse`; altura dentro del máximo e índice de claves listo | Copia por HLS. |
| MKV o MOV con H.264 de 8 bits, altura dentro del máximo e índice de claves listo | Copia por HLS. |
| MKV, MOV, MP4 o M4V con H.265 de 8 bits (`hevcmse`) o 10 bits (`hevc10mse`), altura dentro del máximo e índice de claves listo | Copia por HLS. |
| MP4 o M4V con H.264 de 8 bits pero audio no compatible (AC-3, DTS, etc.), altura dentro del máximo e índice de claves listo | Copia por HLS. |
| AVI, TS o M2TS | Recodificación a H.264 + AAC estéreo. |
| MPEG-4 parte 2 (DivX/Xvid), AV1, H.264 de 10 bits o con perfil High 10/4:2:2/4:4:4 | Recodificación a H.264 + AAC estéreo. |
| Candidato a copia por encima de la altura máxima, o con el índice de claves pendiente o fallido | Recodificación mientras no se pueda copiar; el índice queda encolado para la próxima vez. |
| Cualquier otra extensión | 415; el contenedor no se admite. |

La **altura máxima** es la de la calidad elegida: con `auto` u `original`, la de `playback.max_height` (sin superar la altura original). La directa de MP4/M4V no aplica `playback.max_height`: si se sirve directa, se ve la resolución original. El máximo de altura solo decide si se puede **copiar** cuando el archivo va por HLS; si la altura supera la de la calidad elegida, se recodifica. Para códecs, audio y píxeles manda la ficha de `ffprobe`; la extensión solo separa MP4/M4V del resto.

Un MP4 de H.265 con audio incompatible no se sirve directo: se copia si el navegador declara MSE y se puede copiar, o se recodifica si no.

## Cómo se decide la copia

La copia necesita el índice de fotogramas clave del archivo:

- **MKV:** se leen los Cues del contenedor, que suelen estar en milisegundos. Se detecta por la firma EBML, no por la extensión; un archivo llamado `.mkv` que en realidad sea MP4 sigue el camino de `ffprobe`.
- **MP4 y otros contenedores:** la primera vez se leen los paquetes con `ffprobe` en segundo plano, una sola tarea a la vez. Mientras termina, esa reproducción recodifica; la siguiente ya puede copiar. Leer los paquetes de un archivo grande puede tardar (en el servidor del autor, unos 31 s en frío para 5 GB).
- Si el índice no tiene al menos dos claves y la primera no está al principio, se descarta y se prueba con `ffprobe`.

El índice se guarda en `cache/keyframes`, junto a la base de datos, con una clave formada por ruta, tamaño y fecha. Se invalida solo cuando cambia el archivo y se puede borrar sin riesgo: se volverá a calcular. Leer los Cues de un MKV tarda entre 0,05 y 0,26 s por archivo en las medidas del servidor del autor.

Con el índice listo, el servidor calcula los límites de los segmentos: empieza en 0 y el siguiente corte es la primera clave que queda al menos a 6 s del anterior, con un margen de 0,25 s para que la búsqueda no caiga en otra clave. La playlist VOD usa las duraciones reales de esos cortes.

Si la copia falla antes de generar el segmento pedido, la petición devuelve el error, el archivo queda marcado en memoria como no apto para copia (ruta, tamaño y fecha) y la siguiente reproducción recodifica. La marca desaparece al reiniciar el servidor.

## HLS: copia y recodificación

Los dos modos comparten los segmentos MPEG-TS, los saltos y los límites de procesos; la playlist VOD se usa cuando se conoce la duración:

| | Copia | Recodificación |
|---|---|---|
| Vídeo | `-c:v copy`, sin escalar ni perder calidad. | `libx264` con preset, CRF y altura máxima configurables, `yuv420p`. |
| Audio | Una pista a AAC estéreo de 160 kbps. | Una pista a AAC estéreo de 160 kbps. |
| Cortes | Fotogramas clave reales, con `-copyts`, `-output_ts_offset` de 10 s y `-segment_times`; la playlist usa duraciones reales. | Keyframes forzados cada 6 s y segmentos de 6 s. |
| Cuándo | Códec admitido, altura dentro del máximo e índice listo. | En el resto de casos. |

La copia solo se ofrece si el vídeo no hay que escalarlo. Elegir una calidad inferior a la resolución original obliga a recodificar; `original` conserva la altura original siempre que no supere `playback.max_height`.

El resto del flujo es común:

- Con playlist VOD propia, FFmpeg no arranca al pedir la playlist: arranca con la primera petición de segmento, en el punto pedido. Si no hay duración (solo puede pasar al recodificar), arranca al preparar la playlist `event`.
- La playlist es un VOD completo si se conoce la duración. La copia siempre la necesita; si no se puede conocer, se recodifica. La recodificación usa la playlist `event` de FFmpeg cuando no hay duración y entonces los saltos lejanos no son rápidos.
- Si se pide un segmento más de 24 s por delante de lo generado, o un hueco hacia atrás, el servidor reinicia FFmpeg en ese segmento. En la copia, la búsqueda se hace 0,2 s después de la clave para no caer en la anterior.
- Un segmento se considera completo cuando aparece en la playlist del proceso; al reiniciar se descartan el que quedó a medias y los que se van a reescribir.
- Los procesos van con prioridad baja y número de hilos limitado.

En el servidor del autor, la copia va a unas 37× tiempo real y casi no usa CPU.

## Límites de procesos

| Límite | Valor |
|---|---|
| Espera de la playlist inicial | 20 s |
| Espera de un segmento | 2 min |
| Vida máxima de un proceso | 4 h |
| Parada por inactividad | 90 s sin peticiones |
| Segmentos generados por delante | 20 (~2 min) |
| Transcodificaciones simultáneas | `playback.max_transcodes`, 2 por defecto |
| Copias simultáneas | 6 |
| Limpieza de trabajos HLS | 1 h sin uso, revisión cada 15 min |

Las copias **no cuentan** para «Transcodificaciones simultáneas». Si se superan las 6 copias, se para la copia más antigua sin peticiones recientes. La parada por inactividad y por adelanto se aplica igual a copias y recodificaciones.

## Qué ve el usuario

La ficha del archivo indica el método previsto en `playback.method`:

- `direct`: reproducción directa.
- `copy`: «Copia (sin recodificar)».
- `transcode`: recodificación a H.264 + AAC estéreo.

La ficha se calcula con los datos cacheados de `ffprobe`, las capacidades del navegador y la altura máxima del ajuste. El índice de claves se comprueba al preparar la reproducción: por eso la primera vez que se ve un MP4 puede indicar copia y aun así recodificar mientras el índice se calcula en segundo plano.

El reproductor ofrece `auto`, 480p, 720p, 1080p (sin superar la altura original ni `playback.max_height`) y `original`. En directa solo hay `original`. En HLS, la copia no cambia la resolución; las calidades inferiores se recodifican.

## Audio, subtítulos y calidad

- **Audio directo:** se conserva el archivo original. Se sirve directo si la pista por defecto es AAC o MP3 (o no hay audio).
- **Audio HLS:** se elige una pista por posición en la lista de `ffprobe` y se convierte a AAC estéreo de 160 kbps. La preferencia del usuario y de la película se guarda en `item_preferences`.
- **Subtítulos:** se leen pistas de texto incrustadas y archivos externos `.srt`, `.vtt` o `.ass` junto al vídeo, con sufijos de idioma y `forced`. Se convierten a WebVTT. Las pistas de imagen (PGS/VobSub) no se admiten y el HLS no lleva una representación separada de subtítulos.
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

## Qué formato conviene guardar

| Recomendación | Por qué |
|---|---|
| **H.265 de 8 bits**, en MKV o MP4 | Es el mejor equilibrio: se copia si el navegador lo admite por MSE y ocupa menos que H.264. Ahorro orientativo del 30–50 % frente a H.264 con calidad parecida, sin cifras garantizadas. |
| **H.264 de 8 bits**, en MP4 o MKV | Sigue siendo muy compatible: MP4 con AAC/MP3 se sirve directo y MKV/MOV se copia sin recodificar. |
| Evitar **H.265 de 10 bits** salvo que todos los dispositivos lo necesiten | Muchos navegadores y equipos no lo reproducen; en el servidor no hay decodificación por hardware. Solo se copia si el navegador declara `hevc10mse`. |
| **Audio original** (AAC, MP3, AC-3…) | En directa se conserva; en HLS solo se recodifica el audio a AAC estéreo, un coste pequeño. |
| **Subtítulos de texto** incrustados en el MKV o en `.srt` junto al vídeo | Se convierten a WebVTT; los de imagen (PGS/VobSub) no se pueden mostrar. |
| **Fotogramas clave cada 2–10 s** | Es lo habitual y permite segmentos y saltos razonables en la copia. Intervalos muy largos hacen segmentos más largos y saltos menos precisos. |
| Evitar **AVI y TS/M2TS** para la biblioteca | Se recodifican siempre; convertir a MKV/MP4 conserva mejor el códec y evita trabajo al servidor. |

No hay aceleración por hardware todavía: la recodificación usa la CPU. La copia apenas consume CPU porque no decodifica ni codifica el vídeo.

## Qué falta para la reproducción avanzada

- Aceleración por hardware (Quick Sync/VA-API). El servidor tiene `/dev/dri`, pero no se pasa al contenedor ni está implementada.
- Perfiles por dispositivo y elección automática entre calidades.
- HDR con tone mapping y tratamiento de 10 bits más allá del H.265 que el navegador ya admita por MSE.
- Desentrelazado.
- Subtítulos de imagen y audio no soportado o multicanal/passthrough.
- ABR y playlist maestra con varias calidades.
- Un modo de API `transcode` explícito, separado de `remux`.

El orden de estas mejoras está en [estado](estado.md). La relación con Reduvid se mantiene: convertir antes los archivos con [Reduvid](https://github.com/catlinux/reduvid) reduce la necesidad de transcodificar.

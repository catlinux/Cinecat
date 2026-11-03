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
| AVI, TS o M2TS | Recodificación a H.264 + AAC: estéreo, o 5.1 para apps de TV que declaran `mch`. |
| MPEG-4 parte 2 (DivX/Xvid), AV1, H.264 de 10 bits o con perfil High 10/4:2:2/4:4:4 | Recodificación a H.264 + AAC: estéreo, o 5.1 para apps de TV que declaran `mch`. |
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
| Vídeo | `-c:v copy`, sin escalar ni perder calidad. | `libx264` o VA-API, con preset/CRF o `qp` y altura máxima configurables. |
| Audio | Una pista a AAC estéreo de 160 kbps, o AAC 5.1 de 384 kbps para apps de TV con `mch` y una pista de 6 o más canales. | Una pista a AAC estéreo de 160 kbps, o AAC 5.1 de 384 kbps para apps de TV con `mch` y una pista de 6 o más canales. |
| Cortes | Fotogramas clave reales, con `-copyts`, `-output_ts_offset` de 10 s y `-segment_times`; la playlist usa duraciones reales. | Keyframes forzados cada 6 s y segmentos de 6 s. |
| Cuándo | Códec admitido, altura dentro del máximo e índice listo. | En el resto de casos. |

La copia solo se ofrece si el vídeo no hay que escalarlo. Elegir una calidad inferior a la resolución original obliga a recodificar; `original` conserva la altura original siempre que no supere `playback.max_height`.

Al recodificar, si `ffprobe` marca el vídeo como entrelazado (`field_order` `tt`, `bb`, `tb` o `bt`) se antepone `yadif` a la cadena de filtros. Si `color_transfer` es `smpte2084` (HDR10) o `arib-std-b67` (HLG) se aplica un mapeo de tonos a SDR con `zscale` y `tonemap` antes del escalado, siempre que FFmpeg incluya ambos filtros; si faltan, se registra un aviso una sola vez y se recodifica sin el mapeo. La copia de vídeo nunca aplica filtros. Los tests cubren los argumentos, pero 0.55.0 aún no se ha probado con archivos entrelazados o HDR reales; el AAC 5.1 de 0.56.0 aún no se ha probado en una TV real.

El resto del flujo es común:

- Con playlist VOD propia, FFmpeg no arranca al pedir la playlist: arranca con la primera petición de segmento, en el punto pedido. Si no hay duración (solo puede pasar al recodificar), arranca al preparar la playlist `event`.
- La playlist es un VOD completo si se conoce la duración. La copia siempre la necesita; si no se puede conocer, se recodifica. La recodificación usa la playlist `event` de FFmpeg cuando no hay duración y entonces los saltos lejanos no son rápidos.
- Si se pide un segmento más de 24 s por delante de lo generado, o un hueco hacia atrás, el servidor reinicia FFmpeg en ese segmento. En la copia, la búsqueda se hace 0,2 s después de la clave para no caer en la anterior.
- Un segmento se considera completo cuando aparece en la playlist del proceso; al reiniciar se descartan el que quedó a medias y los que se van a reescribir.
- Los procesos van con prioridad baja y número de hilos limitado.

En el servidor del autor, la copia va a unas 37× tiempo real y casi no usa CPU.

## Aceleración por hardware

La recodificación puede usar la gráfica Intel integrada mediante **VA-API**. No cambia la copia, que sigue sin tocar el vídeo, ni la decisión directo/HLS.

Requisitos:

- La imagen incluye `intel-media-va-driver` y fija `LIBVA_DRIVER_NAME=iHD`.
- El host expone `/dev/dri/renderD128` (se puede cambiar con `CINECAT_VAAPI_DEVICE`) y se usa `docker-compose.gpu.yml` para montarlo en el contenedor `servidor`. Sin ese archivo, `docker-compose.yml` sigue arrancando y todo funciona por software.
- En el `.env` del despliegue se activa `COMPOSE_FILE=docker-compose.yml:docker-compose.gpu.yml` (el actualizador lo respeta).
- El ajuste `playback.hardware` está en `auto` (valor por defecto). Con `off` se desactiva siempre. Se lee al preparar cada reproducción, así que el cambio se aplica a las nuevas.

La primera vez que hace falta, el servidor comprueba VA-API con un vídeo sintético de un segundo y un límite de 15 s. El resultado se recuerda hasta reiniciar y se registra una vez en el log, tanto si se activa como si no.

Hay dos rutas:

- **CPU+GPU:** la CPU decodifica y escala; la gráfica codifica con `h264_vaapi`. Es la ruta general, también para DivX/Xvid, H.264 de 10 bits, entrelazados, H.265 de 10 bits o cuando hay que reducir la altura.
- **GPU completa:** la gráfica decodifica y codifica, sin filtro de escala. Solo se usa con H.264 de 8 bits (`yuv420p`/`yuvj420p`, perfiles no High 10/4:2:2/4:4:4) o H.265 Main de 8 bits, progresivo o con orden de campo desconocido, sin conversión HDR a SDR pendiente, y cuando la altura de origen no supera la de salida.

No se usa la unidad de escalado de la gráfica: `scale_vaapi` y `scale_qsv` no están disponibles en esta gráfica y nunca se invocan. La codificación usa `qp = CRF + 2`, `-g 250` y los mismos cortes forzados cada 6 s; el audio y el resto del flujo HLS no cambian.

En el servidor del autor, 60 s de película 1080p→720p pasan de 44 s de CPU con libx264 (HEVC de origen) a 19 s con CPU+GPU, y de 39 s a 18 s con H.264; con todo en la gráfica y sin escalar, 0,6 s (unas 14×). Son medidas de ese servidor, no una promesa para otro hardware.

Si el proceso con hardware falla antes de generar el segmento pedido, se registra, se marca el archivo (ruta, tamaño y fecha) como no apto para hardware y se reintenta el mismo segmento por software sin que el cliente vea el error. Si falla la comprobación general, el hardware queda desactivado hasta reiniciar. Las recodificaciones con hardware cuentan para «Transcodificaciones simultáneas» como las demás.

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
- `transcode`: recodificación a H.264 + AAC (estéreo, o 5.1 para apps de TV con `mch`).

La ficha se calcula con los datos cacheados de `ffprobe`, las capacidades del navegador y la altura máxima del ajuste. El índice de claves se comprueba al preparar la reproducción: por eso la primera vez que se ve un MP4 puede indicar copia y aun así recodificar mientras el índice se calcula en segundo plano.

El reproductor ofrece `auto`, 480p, 720p, 1080p (sin superar la altura original ni `playback.max_height`) y `original`. En directa solo hay `original`. En HLS, la copia no cambia la resolución; las calidades inferiores se recodifican.

## Intro y créditos

`/play` incluye `segments`, una lista posiblemente vacía de `{kind, start, end, source}`. `kind` es `intro` o `credits`, los tiempos están en segundos y `source` indica `chapter` o `manual`. La ficha técnica expone los capítulos en `chapter_list` (inicio, fin y título).

El servidor reconoce un capítulo como intro si su título, sin mayúsculas, acentos ni espacios sobrantes, es o empieza por `intro`, `introduccion`, `opening`, `apertura`, `cabecera`, `cortinilla`, `generique` o `generique de debut`, y como créditos si empieza por `credits`, `creditos`, `ending`, `end credits`, `final credits`, `generique de fin` o `titulos de credito`. Se toma el primero de cada tipo; debe cumplir `0 <= inicio < fin <= duración` y la intro no puede durar 10 minutos o más. Los créditos no tienen límite de duración. No hay detección por audio.

Los segmentos manuales se guardan en `item_segments` y sustituyen al capítulo del mismo tipo. En la ficha web, un administrador puede escribirlos, borrarlos dejando la fila vacía y, en episodios, aplicarlos a toda la temporada. La web muestra **Saltar intro** o **Saltar créditos** y, en episodios con siguiente, **Siguiente episodio** en los créditos; Android y Android TV también muestran **Siguiente episodio** cuando hay uno.

## Reproducción automática del siguiente episodio

El ajuste vive en la tabla `series_autoplay` (usuario y serie; sin fila está activado) y `/play` lo devuelve como `autoplay_next`. `/play` incluye además `next_episode`, que es `null` o `{id, title, season, episode}` con el episodio inmediatamente posterior visible por temporada y número; un episodio siguiente oculto por permisos de biblioteca no aparece. La web y las apps usan estos campos para la cuenta atrás. Con el ajuste desactivado no hay salto automático, pero el botón manual **Siguiente episodio** de los créditos sigue disponible.

## Audio, subtítulos y calidad

- **Audio directo:** se conserva el archivo original. Se sirve directo si la pista por defecto es AAC o MP3 (o no hay audio).
- **Audio HLS:** se elige una pista por posición en la lista de `ffprobe` y se convierte a AAC estéreo de 160 kbps. Si la app es nativa, declara `mch` y la pista elegida tiene 6 o más canales, se conserva como AAC 5.1 de 384 kbps con `-channel_layout 5.1`. La excepción es una app nativa que declara el mismo códec de la pista elegida y esta es AC-3 o E-AC-3: el HLS copia el audio tal cual, tanto si el vídeo se copia como si se recodifica. Nunca se copian DTS ni TrueHD, y la web siempre recibe AAC. La preferencia del usuario y de la película se guarda en `item_preferences`.
- **Otra pista en MP4:** si en la web eliges una pista distinta de la que el navegador reproduciría sola en un MP4 compatible, el servidor lo sirve por HLS con esa pista (desde 0.48.3).
- **Subtítulos:** los de texto incrustados y los archivos externos `.srt`, `.vtt` o `.ass` junto al vídeo, con sufijos de idioma y `forced`, se convierten a WebVTT. Las pistas de imagen incrustadas (PGS, VobSub y DVB) aparecen en `/play` con `burn_in: true` y `url` vacía; al elegirlas, el servidor pasa a HLS y las quema en el vídeo, así que hay que recodificar (nunca en copia ni con VA-API). El HLS no lleva una representación separada de subtítulos: el texto se sirve aparte y la imagen forma parte del vídeo.
- **Preferencias:** audio, subtítulo y calidad por usuario y película; el reproductor las guarda y recarga en el segundo actual al cambiarlas.
- **Perfil del dispositivo:** cada sesión puede guardar una calidad máxima y un idioma de audio y de subtítulo preferidos. Se aplican al reproducir solo si no hay parámetro explícito ni preferencia guardada para esa película (explícito > ítem > perfil > defecto). El subtítulo del perfil nunca elige una pista de imagen; `off` desactiva los subtítulos por defecto. La calidad máxima solo limita HLS: recorta las calidades ofrecidas y baja `auto`, `original` y las superiores al tope; la reproducción directa no se toca.

## Apps nativas (Android y Android TV)

Las apps declaran en `caps` lo que su equipo reproduce: contenedores (`mkv`, `mp4`), vídeo (`h264`, `h264hi10`, `hevc`, `hevc10`, `av1`, `vp9`), audio (`aac`, `mp3`, `ac3`, `eac3`, `dts`, `truehd`, `opus`, `flac`) y `mch` si decodifica AAC multicanal. Con eso, un MKV o MP4 cuyo vídeo y pista de audio elegida estén declarados se sirve **en directo**, sin tocar el archivo (AC-3, E-AC-3 o DTS llegan tal cual para pasarlos por HDMI). Si no, va por HLS: el HEVC se copia sin recodificar y el audio se convierte a AAC estéreo, salvo que la app sea de TV y declare `mch` con una pista de 6 o más canales, en cuyo caso recibe AAC 5.1. El móvil no declara `mch`; la web tampoco envía esos tokens, así que sigue en estéreo. Si la app declara `ac3` o `eac3` y la pista elegida usa ese mismo códec, el HLS copia el audio; DTS y TrueHD nunca se copian. `/play` devuelve `reason` (`container`, `video`, `audio` o `subtitle`) para saber por qué no fue en directo; una pista de imagen fuerza HLS y ese último motivo.

## Reproductor web

- `<video>` nativo para reproducción directa; hls.js 1.7.3 para HLS.
- Controles propios: play/pausa, ±10 s, barra de progreso, volumen, silencio, velocidad (0,5×–2×), audio, subtítulos, calidad, imagen en imagen, pantalla completa y siguiente episodio.
- Atajos: `Espacio`/`K`, `J`/`L` o flechas, `M`, `F`, `C`, `,`/`.`, `0`–`9` y `Esc`. El detalle está en la [guía de usuario](guia-de-usuario.md).
- El progreso se envía cada 15 s y al salir; al 90 % se marca como visto y se borra el progreso. Si quedan 15 s o menos, no se reanuda.
- Con la reproducción automática activa y un siguiente episodio, al terminar aparece una cuenta atrás de 8 s; el interruptor **Reproducción automática** de los controles permite cambiarla durante la reproducción.

## URLs firmadas

- HMAC-SHA256 con una clave aleatoria de 32 bytes guardada junto a la base de datos.
- TTL por defecto: 8 horas.
- La firma cubre el identificador y la caducidad; no está ligada al usuario. Cualquiera con la URL puede usarla hasta que expire.
- La clave se conserva entre reinicios, así que las URLs anteriores siguen valiendo hasta su caducidad.

## Qué formato conviene guardar

| Recomendación | Por qué |
|---|---|
| **H.265 de 8 bits**, en MKV o MP4 | Es el mejor equilibrio: se copia si el navegador lo admite por MSE y ocupa menos que H.264. Ahorro orientativo del 30–50 % frente a H.264 con calidad parecida, sin cifras garantizadas. |
| **H.264 de 8 bits**, en MP4 o MKV | Sigue siendo muy compatible: MP4 con AAC/MP3 se sirve directo y MKV/MOV se copia sin recodificar. |
| Evitar **H.265 de 10 bits** salvo que todos los dispositivos lo necesiten | Muchos navegadores y equipos no lo reproducen; la gráfica del servidor no decodifica HEVC de 10 bits entero. Solo se copia si el navegador declara `hevc10mse`. |
| **Audio original** (AAC, MP3, AC-3…) | En directa se conserva; en HLS solo se recodifica el audio a AAC estéreo o 5.1 según el cliente, un coste pequeño. |
| **Subtítulos de texto** incrustados en el MKV o en `.srt` junto al vídeo | Se convierten a WebVTT. |
| **Subtítulos de imagen** (PGS, VobSub, DVB) incrustados | Se eligen y se queman en el vídeo al recodificar por HLS; solo se pueden usar al recodificar, nunca en copia ni con VA-API. |
| **Fotogramas clave cada 2–10 s** | Es lo habitual y permite segmentos y saltos razonables en la copia. Intervalos muy largos hacen segmentos más largos y saltos menos precisos. |
| Evitar **AVI y TS/M2TS** para la biblioteca | Se recodifican siempre; convertir a MKV/MP4 conserva mejor el códec y evita trabajo al servidor. |

La recodificación usa VA-API cuando está disponible y libx264 cuando no. Quemar un subtítulo de imagen fuerza la ruta por CPU (libx264), sin VA-API. La copia apenas consume CPU porque no decodifica ni codifica el vídeo.

## Qué falta para la reproducción avanzada

- Elección automática entre calidades.
- Tratamiento de 10 bits más allá del H.265 que el navegador ya admita por MSE.
- DTS y TrueHD sin recodificar en HLS (hoy se convierten a AAC).
- ABR y playlist maestra con varias calidades.
- Un modo de API `transcode` explícito, separado de `remux`.

El orden de estas mejoras está en [estado](estado.md). La relación con Reduvid se mantiene: convertir antes los archivos con [Reduvid](https://github.com/catlinux/reduvid) reduce la necesidad de transcodificar.

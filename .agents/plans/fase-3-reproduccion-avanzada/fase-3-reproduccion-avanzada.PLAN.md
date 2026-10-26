# Fase 3: decisión de reproducción avanzada (`0.8.0`+)

Arranca por un bug real encontrado en producción (2026-10-26): `Prison Break 4x22.avi` remuxea en un único segmento HLS de 362 MB / 1504 s porque el `.avi` tiene muy pocos keyframes y `-c:v copy` solo puede cortar en keyframes. El navegador tiene que descargar el segmento casi entero antes de poder reproducir — en la práctica, no se reproduce. Causa raíz: el remux sin recodificar (Fase 1/2) no cubre vídeos con intervalos de keyframe grandes, y hoy no hay nada que lo detecte ni caiga a transcodificar.

Decisión con el usuario (2026-10-26): en vez de un parche puntual, se aborda ya la Fase 3 del roadmap (`docs/es/reproduccion.md`), pero en pasos pequeños que primero resuelven el caso real y van ampliando hacia el diseño completo (perfiles de dispositivo, H.265/H.264, aceleración por hardware). No construir todo el motor de golpe.

Decisiones ya fijadas: FFmpeg como herramienta externa (ya en el `Dockerfile` del despliegue, confirmado `7.1.5` funcionando en el Debian real), argumentos en lista nunca por shell (`internal/playback/hls.go`), licencia GPL-3.0. Ver `docs/es/reproduccion.md` para la visión completa de la Fase 3 (perfiles de capacidades, 10 bits, HDR, subtítulos quemados — fuera de alcance de los pasos de abajo, vienen después).

Cada paso es autocontenido. Al completarlo, el ejecutor lo marca `[x]` aquí y añade su entrada en `CHANGELOG.md`/`CHANGELOG.en.md` y `VERSION`, en el mismo commit. Commits: `tipo: descripción en castellano`.

## Paso 1 — Detectar intervalos de keyframe problemáticos antes de remuxear

- **Archivos:** `internal/playback/probe.go` (nuevo), `internal/playback/hls.go`, `internal/playback/decision.go`, tests correspondientes.
- **Qué hace:** antes de lanzar el remux, ejecuta `ffprobe` (ya viene con FFmpeg) sobre el archivo para listar los tiempos de los keyframes del flujo de vídeo (`ffprobe -select_streams v:0 -show_entries frame=pict_type,pkt_pts_time -of csv` o equivalente; investigar la invocación más fiable y barata, p. ej. limitarla a leer solo cabeceras sin decodificar todo el archivo si es posible). Si el intervalo máximo entre keyframes consecutivos supera un umbral razonable (a decidir y documentar, por ejemplo 15-20 s), el remux sin recodificar no es viable: en ese caso `Decide` (o una función nueva) debe indicar que hace falta transcodificar el vídeo en vez de copiarlo, en lugar de intentar el remux `copy` y producir un segmento gigante como ahora.
- **Por qué este paso primero:** es la pieza que falta para no repetir el problema real ya visto; no necesita todavía implementar la transcodificación en sí (eso es el Paso 2), solo detectar el caso y no mentir con un remux que no sirve.
- **Criterio de terminado:** test con un archivo de prueba generado con FFmpeg que tenga un intervalo de keyframe grande (por ejemplo `-x264-params keyint=9999` o equivalente) y otro con keyframes frecuentes (por defecto); la función de detección distingue ambos casos correctamente. Documentar en el propio código el umbral elegido y por qué.
- **Ejecutor:** AgentRelay, esfuerzo alto (lógica de `ffprobe` delicada, hay que acertar la invocación y el parseo).
- **Versión/CHANGELOG:** parche o menor según alcance final — Añadido: detección de intervalos de keyframe que hacen inviable el remux sin recodificar.

## Paso 2 — Transcodificación de vídeo cuando el remux puro no basta

- **Archivos:** `internal/playback/hls.go`, `internal/playback/service.go`, `internal/playback/decision.go`.
- **Qué hace:** cuando el Paso 1 detecta que hace falta transcodificar, añade un nuevo `Mode` (p. ej. `ModeTranscode`) que invoca FFmpeg con `-c:v libx264` (u otro códec soportado ampliamente, a decidir) en vez de `-c:v copy`, manteniendo `-c:a copy` si el audio ya es compatible (AAC/AC3/MP3; si no, aplicar también `-c:a aac` — puede ser parte de este paso o uno posterior, decidir según cuánto crezca). Sin aceleración por hardware todavía (software puro); eso es un paso posterior si hace falta por rendimiento. Reutiliza el resto de la infraestructura ya existente (firmas, caché de workDir, servir los segmentos) sin tocarla si no hace falta.
- **Criterio de terminado:** test de integración con el archivo de keyframes dispersos del Paso 1: el remux produce varios segmentos de duración razonable (próxima a `hls_time`), no uno solo gigante. Prueba manual documentada contra el episodio real que falló (o uno con características similares) si el ejecutor tiene FFmpeg disponible.
- **Ejecutor:** AgentRelay, esfuerzo alto (tocar la invocación de FFmpeg es sensible; revisar con cuidado que sigue sin pasar por shell).
- **Dependencias:** Paso 1.
- **Versión/CHANGELOG:** menor — Añadido: transcodificación de vídeo por software cuando el remux sin recodificar no es viable.

## Paso 3 — Mostrar el modo elegido y progreso básico en el cliente web

- **Archivos:** `api/openapi.yaml` (si `PlayResponse.mode` necesita un valor nuevo), `web/src/lib/api.ts`, `web/src/lib/components/Player.svelte`, `web/src/routes/items/[id]/play/`.
- **Qué hace:** el cliente web ya distingue `direct`/`remux` en `Player.svelte`; si se añade `transcode` como modo nuevo, debe seguir funcionando igual del lado del reproductor (sigue siendo HLS), pero puede mostrar un aviso breve de que el vídeo se está preparando (la transcodificación tarda más que un remux puro) en vez de dejar la rueda de carga sin contexto.
- **Criterio de terminado:** prueba manual: reproducir un vídeo que requiera transcodificación desde el cliente web real y confirmar que arranca en un tiempo razonable y sin mensajes confusos.
- **Ejecutor:** AgentRelay, esfuerzo medio.
- **Dependencias:** Paso 2.
- **Versión/CHANGELOG:** menor — Añadido: aviso de preparación en el reproductor web cuando el vídeo necesita transcodificar.

## Después de estos tres pasos

Parar y revisar con el usuario qué de la Fase 3 completa (perfiles de capacidades por dispositivo, H.265 preferente con fallback a H.264, aceleración por hardware, 10 bits/HDR, subtítulos quemados, audio no soportado) hace falta de verdad para su biblioteca real, en vez de construir todo lo que describe `docs/es/reproduccion.md` sin necesidad concreta. Probablemente el caso de Prison Break y similares (series antiguas con codecs/contenedores problemáticos) sea el grueso del problema real; priorizar según lo que seguya fallando tras estos tres pasos.

## Al cerrar

Actualizar `docs/es/reproduccion.md` para reflejar el estado real (quitar "Estado: diseño" si ya hay partes implementadas) y `TODO.md`/`TODO.en.md` para los huecos que queden (aceleración por hardware, HDR, etc., si no se abordan en esta tanda).

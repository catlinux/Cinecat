# TODO

[English version](TODO.en.md)

Prioridades: 🔴 bloqueante · 🟠 próxima fase · 🟢 más adelante.

## Antes de escribir código
- 🟠 Activar en GitHub los avisos de seguridad privados (private vulnerability reporting).
- 🔴 El bloqueo por intentos fallidos de login (`internal/auth`) se indexa por nombre de usuario, no por IP: cualquiera puede bloquear la cuenta de otro usuario legítimo enviando 5 logins fallidos con su `username`. Revisar antes de exponer el servidor a internet (por ejemplo, combinando usuario+IP o añadiendo límite por IP).
- 🟠 No hay todavía forma de crear usuarios (ni endpoint ni comando); para probar el login hay que insertar el usuario y su hash argon2id directamente en la base de datos.
- 🟠 El escaneo por API (`POST /api/v1/libraries/scan`) es síncrono: con bibliotecas grandes la petición puede tardar mucho. La versión asíncrona con progreso queda pendiente.
- 🟠 Series y episodios siguen sin sinopsis ni cartel: OMDb se consulta solo para películas y el cliente TMDB (`internal/metadata/tmdb.go`) queda sin conectar para el futuro.
- El TTL de caché de TMDB (7 días, `internal/metadata`) es un valor de partida sin verificar contra los términos vigentes de la API: revisar antes de producción.
- 🟠 Los endpoints `GET /api/v1/libraries|items` no filtran por rol ni por biblioteca: cualquier usuario autenticado ve todo. La arquitectura prevé permisos por biblioteca (Fase 3+); implementar antes de un uso multiusuario real.
- 🟠 La clave gratuita de OMDb permite 1000 peticiones/día: el primer escaneo completo de una biblioteca grande puede agotar la cuota. Los ítems sin metadatos se reintentan en escaneos posteriores (hay caché negativa en disco).
- `internal/metadata` (TMDB) sigue sin conectarse a los endpoints ni al escaneo; de momento `media_items.tmdb_id` solo se rellena a mano.
- Pequeña duplicación: `bearerToken()` está repetida en `internal/auth/http.go` e `internal/library/http.go`; se podría extraer a un sitio común cuando haya un tercer consumidor.
- 🟠 Los tests HLS se saltan con `t.Skip()` sin `ffmpeg`/`ffprobe` en PATH (como en el PC Windows). Se pueden ejecutar en el contenedor de producción compilando el binario de tests para Linux (`GOOS=linux go test -c ./internal/playback`) y lanzándolo junto a los `.go` del paquete. Falta automatizarlo en CI.
- 🟠 La generación HLS es progresiva: Cinecat escribe una playlist VOD completa con `ENDLIST` desde el principio y relanza FFmpeg en el segmento pedido al saltar. Si no se puede leer la duración del contenedor (algunos AVI antiguos), se cae a la playlist `event` de FFmpeg y el cliente la trata como contenido en directo; falta un método alternativo para calcular la duración en ese caso.
- 🟠 Todo el HLS se recodifica por software (libx264 `veryfast`), incluso los MKV H.264 que podrían copiarse: en SD va a unas 20 veces tiempo real en el Xeon E3-1225 v5, pero en 1080p/HEVC puede ir justo. Pendiente: copiar el vídeo cuando ya sea H.264 8 bits (exige cortar los segmentos en sus keyframes reales) y usar Quick Sync/VAAPI (el servidor tiene `/dev/dri`, pero no se pasa al contenedor).
- 🟠 FFmpeg no se frena: tras un salto genera hasta el final del vídeo aunque nadie lo pida (en SD, ~1 GB en `/tmp` del contenedor por película, que se borra tras 1 h sin uso). Jellyfin lo frena cuando va muy por delante; valorar si hace falta.
- 🟢 Con dos audios (por ejemplo castellano e inglés) se usa siempre el primero; falta poder elegir pista.
- El análisis de keyframes de `internal/playback/probe.go` ya no se usa (todo el HLS se recodifica); se conserva para cuando vuelva el remux sin recodificar.
- 🟠 La API de `/api/v1/items/{id}/play` sigue devolviendo `mode: "remux"` aunque el servidor transcodifique el vídeo para HLS; para el cliente el resultado sigue siendo HLS. Exponer un modo `transcode` es una mejora pendiente.
- 🟠 El HLS solo lleva vídeo y audio; los subtítulos se descartan (se abordarán en la Fase 3, según el diseño de reproducción).
- El cliente web (`web/`) decide la cookie `Secure` según `event.url.protocol`; el despliegue Docker (`docker-compose.yml`, ver [despliegue](docs/es/despliegue.md)) ya pasa `PROTOCOL_HEADER=x-forwarded-proto` al contenedor `web`, y el vhost de Apache (`deploy/video.warcrafted.com.conf`) fija esa misma cabecera. Resuelto para este despliegue; revisar si algún día se sirve sin ese proxy.
- El reproductor web usa hls.js para el remux, que descarga la playlist y los segmentos por XHR contra `PUBLIC_API_URL`; el despliegue Docker evita el problema de CORS sirviendo servidor y cliente bajo el mismo origen (el Apache del Debian enruta `/api/*` al servidor Go y el resto al cliente web, con `PUBLIC_API_URL` vacío). El servidor Go sigue sin enviar cabeceras CORS: si algún día se despliega sin ese proxy unificando el origen, habrá que añadirlas.

## Riesgos conocidos, aceptados
- Existe una app Android inactiva desde 2015 llamada «CineCat» (F-Droid, catálogo de cine catalán con datos de la Generalitat; sin registro de marca). Mismo nombre, ámbito distinto. Se mantiene el nombre Cinecat; riesgo de confusión menor, sin acción pendiente.

## Producto
- 🟠 Servidor mínimo: biblioteca, identificación de películas y series, metadatos en `es-ES`.
- 🟠 Motor de decisión de reproducción (directa / remux / transcodificación).
- 🟠 Cliente web con inicio de sesión.
- 🟠 Clientes Android y Android TV.
- 🟢 Fotos y presentaciones.
- 🟢 Contenido de internet y televisión en directo (solo si se decide).
- 🟢 Segmentos de intro/créditos, descargas sin conexión, sincronizar reproducción entre dispositivos.
- 🟢 Otras plataformas (Apple, televisores Samsung/LG).

## Documentación
- 🟠 Guía de usuario y glosario «para tontos» cuando exista algo que probar.
- 🟠 Código de conducta.
- 🟢 Capturas de pantalla del cliente web.

## Mantenimiento de documentos
- Mantener sincronizadas las versiones en español e inglés de cada documento.

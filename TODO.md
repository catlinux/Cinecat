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
- 🔴 El remux a HLS (`internal/playback`) nunca se ha probado con FFmpeg real: el equipo de desarrollo usado en la Fase 1 no tiene FFmpeg instalado y esos tests se saltaron siempre con `t.Skip()`. Ejecutar la suite completa en una máquina con FFmpeg antes de confiar en la reproducción de `.mkv`/`.avi`/`.mov`/`.ts`/`.m2ts`.
- 🟠 El remux se genera completo y de forma síncrona en la primera petición de la playlist: un archivo largo tarda en empezar a reproducirse. La segmentación bajo demanda queda para más adelante.
- 🟠 La transcodificación (todo `.avi`, y el resto de contenedores cuando el intervalo de keyframes lo exige) es por software (libx264 `veryfast`) sin aceleración por hardware: puede tardar en servidores modestos. La aceleración por hardware queda para un paso posterior del plan de la Fase 3.
- El análisis de intervalo de keyframes (`internal/playback/probe.go`) solo decide para contenedores distintos de `.avi`; no se ha verificado si otros contenedores antiguos (`.mov`/`.ts`/`.m2ts` de origen dudoso) tienen el mismo problema de timestamps `N/A` que obligó a forzar siempre la transcodificación en `.avi`. Revisar si aparece otro caso real.
- 🟠 La API de `/api/v1/items/{id}/play` sigue devolviendo `mode: "remux"` aunque el servidor transcodifique el vídeo para HLS; para el cliente el resultado sigue siendo HLS. Exponer un modo `transcode` es una mejora pendiente.
- 🟠 El remux copia solo vídeo y audio; los subtítulos se descartan (se abordarán en la Fase 3, según el diseño de reproducción).
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

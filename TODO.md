# TODO

[English version](TODO.en.md)

Prioridades: 🔴 bloqueante · 🟠 próxima fase · 🟢 más adelante.

## Antes de escribir código
- 🟠 Activar en GitHub los avisos de seguridad privados (private vulnerability reporting).
- 🔴 El bloqueo por intentos fallidos de login (`internal/auth`) se indexa por nombre de usuario, no por IP: cualquiera puede bloquear la cuenta de otro usuario legítimo enviando 5 logins fallidos con su `username`. Revisar antes de exponer el servidor a internet (por ejemplo, combinando usuario+IP o añadiendo límite por IP).
- 🟠 No hay todavía forma de crear usuarios (ni endpoint ni comando); para probar el login hay que insertar el usuario y su hash argon2id directamente en la base de datos.
- 🟠 El escaneo por API (`POST /api/v1/libraries/scan`) es síncrono: con bibliotecas grandes la petición puede tardar mucho. La versión asíncrona con progreso queda pendiente.
- 🟠 El cliente de metadatos (`internal/metadata`) solo busca películas (`/search/movie`); falta el endpoint de series/episodios de TMDB para identificar `media_items.kind = 'series'/'episode'`. Tampoco está conectado todavía al escaneo ni al servidor.
- El TTL de caché de TMDB (7 días, `internal/metadata`) es un valor de partida sin verificar contra los términos vigentes de la API: revisar antes de producción.
- 🟠 Los endpoints `GET /api/v1/libraries|items` no filtran por rol ni por biblioteca: cualquier usuario autenticado ve todo. La arquitectura prevé permisos por biblioteca (Fase 3+); implementar antes de un uso multiusuario real.
- `internal/metadata` (TMDB) sigue sin conectarse a los endpoints ni al escaneo; de momento `media_items.tmdb_id` solo se rellena a mano.
- Pequeña duplicación: `bearerToken()` está repetida en `internal/auth/http.go` e `internal/library/http.go`; se podría extraer a un sitio común cuando haya un tercer consumidor.
- 🔴 El remux a HLS (`internal/playback`) nunca se ha probado con FFmpeg real: el equipo de desarrollo usado en la Fase 1 no tiene FFmpeg instalado y esos tests se saltaron siempre con `t.Skip()`. Ejecutar la suite completa en una máquina con FFmpeg antes de confiar en la reproducción de `.mkv`/`.avi`/`.mov`/`.ts`/`.m2ts`.
- 🟠 El remux se genera completo y de forma síncrona en la primera petición de la playlist: un archivo largo tarda en empezar a reproducirse. La segmentación bajo demanda queda para más adelante.
- 🟠 El remux copia solo vídeo y audio; los subtítulos se descartan (se abordarán en la Fase 3, según el diseño de reproducción).

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
- 🟢 Capturas de pantalla y guía de despliegue con Docker.

## Mantenimiento de documentos
- Mantener sincronizadas las versiones en español e inglés de cada documento.

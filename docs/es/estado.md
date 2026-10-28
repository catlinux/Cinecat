# Estado: qué hay hecho y qué falta

[English](../en/status.md) · [Volver al README](../../README.md)

Versión actual: **0.20.0**. Esta página resume el estado real del código, no el plan inicial. Las versiones salen del [CHANGELOG](../../CHANGELOG.md).

## Qué hay hecho

| Área | Función | Estado | Notas |
|---|---|---|---|
| Servidor y biblioteca | Servidor Go, SQLite, migraciones y API REST versionada | ✅ 0.1.0 | Binario `cinecatd`; esquema con WAL y claves foráneas. |
| Servidor y biblioteca | Bibliotecas locales, alta y borrado | ✅ 0.1.2 | Una ruta por biblioteca; el borrado limpia sus ítems. |
| Servidor y biblioteca | Escaneo manual y automático en segundo plano | ✅ 0.11.0 | El escaneo responde 202/409 y muestra progreso; se programa cada `scan.interval` y al arrancar. |
| Servidor y biblioteca | Ficha técnica del archivo con `ffprobe` | ✅ 0.11.0 | Contenedor, tamaño, pistas, subtítulos y modo de reproducción. |
| Servidor y biblioteca | Varias carpetas por biblioteca y navegador de carpetas | ✅ 0.17.0 | tipos video, clips y music |
| Servidor y biblioteca | Biblioteca de Vídeos (sin identificar), navegación por carpetas | ✅ 0.18.0 | con miniaturas de fotograma |
| Servidor y biblioteca | Biblioteca de Música: escaneo, API, pantalla y reproductor | ✅ 0.19.0 | reproductor persistente con cola |
| Música | Listas de reproducción | ⬜ | siguiente paso de música |
| Música | Favoritos de música | ⬜ | reutilizará el sistema de favoritos |
| Música | Conversión de formatos no compatibles | ⬜ | transcodificación con FFmpeg |
| Música | Reproducción sin cortes entre pistas | ⬜ | gapless |
| Servidor y biblioteca | Subir o borrar archivos con papelera | ✅ 0.21.0 | Solo vídeo; desactivado por defecto (montaje `rw` y ajuste). |
| Identificación y metadatos | TMDB como fuente principal en castellano, con caché | ✅ 0.11.0 | OMDb queda como respaldo si no hay TMDB; OMDb solo rellena sinopsis y cartel de películas. |
| Identificación y metadatos | Identificación automática, corrección manual y «no es ninguna» | ✅ 0.11.0 | `match_status`: `auto`, `manual`, `none` o vacío; umbral configurable. |
| Identificación y metadatos | Reparto y equipo con foto y personaje | ✅ 0.12.0 | Películas y series; las fichas antiguas se rellenan desde la caché de TMDB. |
| Identificación y metadatos | Certificaciones por edad, TheTVDB, Fanart.tv o TVmaze | ⬜ | No están integrados. |
| Fichas | Ficha completa de película con búsqueda, filtros y enlaces | ✅ 0.11.0 | Incluye título original, sinopsis, géneros, duración, nota, reparto, fondo y colecciones. |
| Fichas | «Más como este» | ✅ 0.12.0 / 0.15.0 | Calculado en local; al final de la ficha. |
| Fichas | Corregir ficha y cambiar cartel/fondo | ✅ 0.11.0 | Solo administradores; las imágenes manuales no las pisa el escaneo. |
| Series | Series, temporadas y episodios con TMDB | ✅ 0.11.0 | Una serie por carpeta de primer nivel; título, sinopsis, imágenes y reparto. |
| Series | Selector de temporadas y carátulas de temporada | ✅ 0.15.0 | Con la carátula de la serie como respaldo. |
| Series | Siguiente episodio, continuar y visionado por episodio, temporada o serie | ✅ 0.11.0 | El portal muestra «Siguiente episodio». |
| Series | Corregir ficha o imágenes de una serie | ✅ 0.11.0 | Solo administradores. |
| Colecciones | Colecciones automáticas de sagas de TMDB | ✅ 0.11.0 | Una colección automática solo se muestra si alcanza `ui.collection_min_items`. |
| Colecciones | Crear, editar, borrar, ordenar y elegir cartel/fondo | ✅ 0.11.0 | Colecciones manuales y cambios respetados en colecciones automáticas. |
| Colecciones | Gestionar colecciones desde la ficha de una película | ✅ 0.22.0 | Añadir, crear y quitar; icono en la carátula y filtro «Sin colección». |
| Interfaz web | Puntuación de TMDB en las carátulas | ✅ 0.22.0 | Película y serie. |
| Interfaz web | Página de persona y botón «Ver tráiler» | ✅ 0.22.0 | Lo que tienes y lo que no; tráiler de YouTube en castellano si existe. |
| Servidor y biblioteca | Subidas guiadas de series, temporadas, capítulos y películas | ✅ 0.22.0 | Solo administradores; mismos requisitos que la subida genérica. |
| Reproducción | Decisión por códec real y capacidades del navegador | ✅ 0.22.0 | H.265 directo solo si el navegador lo admite; la copia depende además del índice de claves y de los tokens MSE. |
| Reproducción | Reproducción directa de MP4/M4V | ✅ 0.1.0 | Sirve el archivo con soporte de rangos HTTP. |
| Reproducción | HLS para MKV, AVI, MOV, TS y M2TS | ✅ 0.12.0 / 0.24.0 | Copia el vídeo si el contenedor, el códec, la altura y el índice de claves lo permiten; si no, recodifica a H.264 + AAC estéreo. Segmentos MPEG-TS, saltos y límites de procesos. |
| Reproducción | Modo copia sin recodificar | ✅ 0.24.0 | Cortes en fotogramas clave reales; H.265 por MSE. |
| Reproducción | Aceleración por hardware Intel (VA-API) | ✅ 0.24.0 | H.264 VA-API con decodificación CPU o GPU; vuelta automática a libx264. |
| Reproducción | Audio, subtítulos de texto y calidad | ✅ 0.13.0 | Subtítulos incrustados de texto y `.srt`/`.vtt`/`.ass` como WebVTT; preferencias por usuario y película. |
| Reproducción | Reproductor propio con controles, atajos y siguiente episodio | ✅ 0.13.0 | hls.js para HLS y `<video>` nativo para directa. |
| Reproducción | Perfiles por dispositivo, HDR/10 bits, subtítulos de imagen y ABR | ⬜ | La copia existe; la recodificación usa VA-API o software, todavía H.264 + AAC estéreo. |
| Interfaz web | Inicio de sesión, sesión en cookie httpOnly, cierre y cambio de contraseña | ✅ 0.2.0 / 0.3.5 | El cierre de sesión está en `/login`. |
| Interfaz web | Portal, cartelera, series y colecciones con búsqueda, filtros, orden y paginación | ✅ 0.11.0 / 0.15.0 | Columnas × filas por página; filtros por letra, género y visto en películas. |
| Interfaz web | Temas claro/oscuro/sistema, menú lateral o superior y diseño adaptable | ✅ 0.2.0 / 0.14.0 | La preferencia de tema se guarda en el navegador. |
| Interfaz web | Favoritos, lista de próximas, búsqueda global y perfiles | ⬜ | No implementados. |
| Administración | Resumen, bibliotecas, escaneo, ajustes, corrección e imágenes | ✅ 0.11.0 / 0.12.0 | Los ajustes se aplican sin reiniciar. |
| Administración | Copia diaria de la base de datos | ✅ 0.25.0 | `VACUUM INTO` en `/data/backups` con retención configurable; lista y «hacer copia ahora» en Ajustes → Sistema. |
| Administración | Sesiones de dispositivo (ver y cerrar) | ✅ 0.26.0 | En Tu cuenta y por usuario; las apps nativas usarán sesiones de 90 días. |
| Administración | Actividad | ⬜ | Historial de reproducciones por usuario; sin hacer. |
| Actualizaciones | Comprobar etiquetas y notas desde el servidor | ✅ 0.14.0 | Etiquetas `vX.Y.Z` de GitHub; `system.check_updates` es opt-in. |
| Actualizaciones | Comprobar y actualizar desde Administración → Sistema | ✅ 0.16.0 | Seguimiento por fases con reinicio, éxito, fallo o vuelta atrás. |
| Actualizaciones | Servicio del host con copia de la base y rollback | ✅ 0.16.0 | Bash + systemd de usuario; no lo ejecuta el contenedor. |
| Seguridad | Argon2id, sesiones de 24 h y roles admin/usuario | ✅ 0.1.0 | El token solo se guarda como hash en la base. |
| Seguridad | Permisos por biblioteca y por usuario | ⬜ | Cualquier usuario autenticado ve toda la biblioteca. |
| Clientes Android / Android TV | Android y Android TV | ⬜ | Sin código ni plan detallado todavía. El cliente web es el único terminado. |

## Lo que viene, en orden

El orden sale de los planes de `.agents/plans`; no hay fechas ni versiones comprometidas.

1. **Búsqueda/identificación manual de colecciones en TMDB**, con «faltan en tu biblioteca».
2. **Usuarios** (crear, rol, restablecer contraseña, borrar) y **perfil** con personalización del portal.
3. **Actividad** desde administración.
4. **Subtítulos subidos por el usuario** y diseño de subtítulos que hoy no se admiten.
5. **Complementos**, todavía en fase de diseño.
6. **DLNA/UPnP**, todavía en fase de diseño.
7. **Tipos de biblioteca** (por ejemplo, música o fotos) cuando se aborde el modelo.
8. **Clientes Android y Android TV**.

## Limitaciones y riesgos conocidos

- **Bloqueo de login por nombre de usuario:** cinco fallos bloquean esa cuenta 15 minutos; el contador está en memoria y no hay límite por IP. Se puede bloquear temporalmente a otra persona conocida.
- **No hay CORS:** el servidor no envía cabeceras `Access-Control-*`. El despliegue previsto funciona porque Apache sirve `/api/` y la web bajo el mismo origen.
- **Sin permisos por biblioteca:** cualquier usuario autenticado puede ver todas las bibliotecas, ítems, series y colecciones. Es un riesgo para uso multiusuario.
- **Sin creación de usuarios:** solo existe el `admin` inicial y el flag `--hash-password`; no hay pantalla ni endpoint de usuarios.
- **Series corregidas a mano:** el escaneo no vuelve a tocar las series `manual` ni `none`; si faltan temporadas o episodios, no se rellenan solos.
- **HLS:** la recodificación usa VA-API si se monta `/dev/dri` y la comprobación funciona; si no, usa libx264 (software). No hay tone mapping HDR, desentrelazado, subtítulos de imagen ni ABR. La copia de vídeo no recodifica y apenas usa CPU.
- **La API sigue llamando `remux` al HLS** aunque copie o recodifique; el cliente recibe HLS en ambos casos.
- **Si no se puede leer la duración**, se usa la playlist `event` de FFmpeg y los saltos lejanos no son rápidos.
- **Subtítulos:** solo de texto; PGS/VobSub se ignoran. El HLS no lleva una pista de subtítulos separada, se sirven como WebVTT.
- **Tests HLS reales:** se saltan sin `ffmpeg`/`ffprobe` en PATH y no hay CI que los ejecute.
- **TTL de TMDB:** la caché de 7 días es un valor de partida no verificado contra los términos vigentes. OMDb solo se usa como respaldo y tiene cuota diaria.
- **Cambios en los discos:** subir y mover a la papelera exige montar `/media` en lectura y escritura; la música no se puede subir ni mover.
- **Cambio de contraseña:** no invalida las demás sesiones abiertas.
- **URLs de reproducción:** duran 8 horas, no están ligadas al usuario y dejan de valer al reiniciar el servidor porque la clave HMAC es por proceso.
- **Actualizador automático:** solo funciona si se instaló el servicio del host; el repositorio no puede comprobar si está activo en un servidor concreto.

## Cómo se actualiza

Manual: `./scripts/actualizar.sh`. Desde la web: Administración → Sistema → «Buscar actualizaciones» y «Actualizar ahora». La instalación del servicio, las copias, el estado y la recuperación manual están en [actualizaciones](actualizaciones.md).

[English](../en/status.md) · [Volver al README](../../README.md)

# Estado: qué hay hecho y qué falta

[English](../en/status.md) · [Volver al README](../../README.md)

Versión actual: **0.63.0**. Esta página resume el estado real del código, no el plan inicial. Las versiones salen del [CHANGELOG](../../CHANGELOG.md).

Leyenda: ✅ hecho · 🟨 pendiente de probar · ⬜ pendiente.

## Qué hay hecho

| Área | Función | Estado | Notas |
|---|---|---|---|
| Servidor y biblioteca | Servidor Go, SQLite, migraciones y API REST versionada | ✅ 0.1.0 | Binario `cinecatd`; esquema con WAL y claves foráneas. |
| Servidor y biblioteca | Bibliotecas locales, alta y borrado | ✅ 0.1.2 | Desde 0.17.0 admiten varias carpetas y tipo; el borrado limpia sus ítems. |
| Servidor y biblioteca | Escaneo manual y automático en segundo plano | ✅ 0.11.0 | El escaneo responde 202/409 y muestra progreso; se programa cada `scan.interval` y al arrancar. |
| Servidor y biblioteca | Ficha técnica del archivo con `ffprobe` | ✅ 0.11.0 | Contenedor, tamaño, pistas, subtítulos y modo de reproducción. |
| Servidor y biblioteca | Varias carpetas por biblioteca y navegador de carpetas | ✅ 0.17.0 | tipos video, clips, music y photos |
| Servidor y biblioteca | Biblioteca de Vídeos (sin identificar), navegación por carpetas | ✅ 0.18.0 | con miniaturas de fotograma |
| Servidor y biblioteca | Biblioteca de Música: escaneo, API, pantalla y reproductor | ✅ 0.19.0 | reproductor persistente con cola |
| Servidor y biblioteca | Biblioteca de Fotos: escaneo, API, miniaturas y visores web y Android | 🟨 0.61.0 | JPEG, PNG, WebP y GIF; fecha EXIF en JPEG; miniaturas JPEG de 400 px en `cache/photos`; carpetas, orden y visor en web y apps. Migración 0028. Cubierto por tests; aún sin probar en un dispositivo real. |
| Música | Listas de reproducción | ⬜ | siguiente paso de música |
| Música | Favoritos de música | ⬜ | reutilizará el sistema de favoritos |
| Música | Conversión de formatos no compatibles | ⬜ | transcodificación con FFmpeg |
| Música | Reproducción sin cortes entre pistas | ⬜ | gapless |
| Servidor y biblioteca | Subir o borrar archivos con papelera | ✅ 0.21.0 | Solo vídeo; desactivado por defecto (montaje `rw` y ajuste). |
| Identificación y metadatos | TMDB como fuente principal en castellano, con caché | ✅ 0.11.0 | OMDb queda como respaldo si no hay TMDB; OMDb solo rellena sinopsis y cartel de películas. |
| Identificación y metadatos | Refresco periódico de fichas de TMDB | ✅ 0.50.0 | En el mantenimiento diario se vuelven a pedir las fichas con más de 90 días, hasta 40 películas y 20 series al día; no toca las corregidas a mano. Migración 0023 (`metadata_refreshed_at`). |
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
| Colecciones | Identificación con TMDB y películas que faltan | ✅ 0.41.0 / 0.42.0 | Buscar, enlazar o desvincular colecciones manuales; la ficha muestra la saga incompleta. |
| Interfaz web | Puntuación de TMDB en las carátulas | ✅ 0.22.0 | Película y serie. |
| Interfaz web | Página de persona y botón «Ver tráiler» | ✅ 0.22.0 | Lo que tienes y lo que no; tráiler de YouTube en castellano si existe. |
| Servidor y biblioteca | Subidas guiadas de series, temporadas, capítulos y películas | ✅ 0.22.0 | Solo administradores; mismos requisitos que la subida genérica. |
| Reproducción | Decisión por códec real y capacidades del navegador | ✅ 0.22.0 | H.265 directo solo si el navegador lo admite; la copia depende además del índice de claves y de los tokens MSE. |
| Reproducción | Reproducción directa de MP4/M4V | ✅ 0.1.0 | Sirve el archivo con soporte de rangos HTTP. |
| Reproducción | HLS para MKV, AVI, MOV, TS y M2TS | ✅ 0.12.0 / 0.24.0 | Copia el vídeo si el contenedor, el códec, la altura y el índice de claves lo permiten; si no, recodifica a H.264 + AAC estéreo, o 5.1 para apps de TV con `mch`. Segmentos MPEG-TS, saltos y límites de procesos. |
| Reproducción | Modo copia sin recodificar | ✅ 0.24.0 | Cortes en fotogramas clave reales; H.265 por MSE. |
| Reproducción | Aceleración por hardware Intel (VA-API) | ✅ 0.24.0 | H.264 VA-API con decodificación CPU o GPU; vuelta automática a libx264. |
| Reproducción | Desentrelazado y HDR a SDR al recodificar | 🟨 0.55.0 | Solo al recodificar; usa `yadif` y `zscale`/`tonemap` si FFmpeg los tiene. Cubierto por tests, aún sin probar con archivos reales. |
| Reproducción | AAC 5.1 por HLS para la TV | 🟨 0.56.0 | Apps con `mch` y pista de 6 o más canales reciben AAC 5.1; navegador y móvil siguen en estéreo. Aún sin probar en una TV real. |
| Reproducción | Audio, subtítulos de texto y calidad | ✅ 0.13.0 | Subtítulos incrustados de texto y `.srt`/`.vtt`/`.ass` como WebVTT; preferencias por usuario y película. |
| Reproducción | Subtítulos de imagen incrustados (PGS, VobSub, DVB) | 🟨 0.58.0 | Se eligen en la web y las apps con la marca « · imagen»; el servidor los quema en el vídeo al recodificar por HLS, nunca en copia ni con VA-API. Cubierto por la prueba real del servidor con FFmpeg y tests unitarios de web y Android; sin probar en un dispositivo real. |
| Reproducción | Perfiles por dispositivo (sesión) | 🟨 0.59.0 | Calidad máxima, idioma de audio y de subtítulo por sesión; se editan en la web y las apps y se aplican con la precedencia explícito > ítem > perfil > defecto. Nunca eligen una pista de imagen y el tope de calidad solo afecta a HLS. Cubierto por tests; sin probar en un dispositivo real. |
| Reproducción | Saltar intro y créditos desde capítulos o marcas manuales | 🟨 0.62.0 | Botones en la web, Android y Android TV; el administrador edita los manuales y los aplica a la temporada. Detección por títulos de capítulo, sin huella de audio. Migración 0029. Cubierto por tests; aún sin probar en un dispositivo real. |
| Reproducción | Reproductor propio con controles, atajos y siguiente episodio | ✅ 0.13.0 | hls.js para HLS y `<video>` nativo para directa. |
| Reproducción | 10 bits y ABR | ⬜ | La copia existe; la recodificación usa VA-API o software y H.264 + AAC estéreo o 5.1. |
| Interfaz web | Inicio de sesión, sesión en cookie httpOnly, cierre y cambio de contraseña | ✅ 0.2.0 / 0.3.5 | El cierre de sesión está en `/login`. |
| Interfaz web | Portal, cartelera, series y colecciones con búsqueda, filtros, orden y paginación | ✅ 0.11.0 / 0.15.0 | Columnas × filas por página; filtros por letra, género y visto en películas. |
| Interfaz web | Temas claro/oscuro/sistema, menú lateral o superior y diseño adaptable | ✅ 0.2.0 / 0.14.0 | La preferencia de tema se guarda en el navegador. |
| Interfaz web | Favoritos y lista de próximas | ✅ 0.20.0 | Tarjetas, páginas y filas del portal. |
| Interfaz web | Historial de reproducciones | ✅ 0.52.0 | Página `/history` en el menú; muestra lo último reproducido o visto y permite quitar entradas. Solo contenido accesible. |
| Interfaz web | Filas del portal personalizables | ✅ 0.53.0 | En Mi cuenta → «Portal de inicio» cada usuario oculta sus propias filas; no afecta a las demás cuentas. |
| Interfaz web | Búsqueda global | ⬜ | No implementada. |
| Interfaz web | Navegación según el contenido | ✅ 0.47.0 | Oculta Películas, Series y Colecciones sin contenido; si falla la consulta, las muestra todas. |
| Administración | Resumen, bibliotecas, escaneo, ajustes, corrección e imágenes | ✅ 0.11.0 / 0.12.0 | Los ajustes se aplican sin reiniciar. |
| Administración | Copia diaria de la base de datos | ✅ 0.25.0 | `VACUUM INTO` en `/data/backups` con retención configurable; lista y «hacer copia ahora» en Ajustes → Sistema. |
| Administración | Gestión de usuarios y acceso a bibliotecas | ✅ 0.24.0 | Crear, rol, desactivar, restablecer contraseña y borrar; cada cuenta ve sus bibliotecas. |
| Administración | Sesiones de dispositivo (ver y cerrar) | ✅ 0.26.0 | En Tu cuenta y por usuario; las apps nativas usan sesiones de 90 días. |
| Administración | Actividad | ✅ 0.44.0 | Últimas reproducciones de todos los usuarios, con filtro; `GET /api/v1/activity`. Desde 0.52.0 respeta el acceso por biblioteca. |
| Administración | Tarjeta de Seguridad y bloqueos persistentes | ✅ 0.51.0 | El Resumen muestra los bloqueos de los últimos 7 días y los 5 últimos; intentos y bloqueos se guardan en SQLite (migración 0024). `GET /api/v1/system/security-events`. |
| Administración | Subtítulos externos subidos | ✅ 0.54.0 | Solo administradores; `.srt`, `.vtt` o `.ass` de hasta 2 MB en `subtitles/` junto a la base, con ids `u<N>`. `POST`/`DELETE /api/v1/items/{id}/subtitles`. |
| Actualizaciones | Comprobar etiquetas y notas desde el servidor | ✅ 0.14.0 | Etiquetas `vX.Y.Z` de GitHub; `system.check_updates` es opt-in. |
| Actualizaciones | Comprobar y actualizar desde Administración → Sistema | ✅ 0.16.0 | Seguimiento por fases con reinicio, éxito, fallo o vuelta atrás. |
| Actualizaciones | Servicio del host con copia de la base y rollback | ✅ 0.16.0 | Bash + systemd de usuario; no lo ejecuta el contenedor. |
| Seguridad | Argon2id, sesiones de 24 h y roles admin/usuario | ✅ 0.1.0 | El token solo se guarda como hash en la base. |
| Seguridad | Permisos por biblioteca y por usuario | ✅ 0.24.0 | Cada cuenta solo ve sus bibliotecas; lo oculto responde 404. |
| Calidad | Flujo «Servidor y web» de GitHub Actions | 🟨 | `.github/workflows/server-web.yml` ejecuta los tests de Go con FFmpeg y los de la web; falta comprobar su primera ejecución en verde. |
| Clientes Android / Android TV | Android y Android TV | ✅ 0.59.0 | Una app para móvil, tableta y TV: sesión o vinculación, biblioteca, reproductor Media3 con preferencias, índice A–Z en TV, imagen en imagen en móvil, actualización automática y avisos de actualización. Desde 0.49.0 ofrece «Desde el principio» y «Continuar», desde 0.49.1 nombra las pistas sin título, desde 0.56.0 la TV declara `mch`, desde 0.58.0 muestra y elige los subtítulos de imagen y desde 0.59.0 edita el perfil de «Este dispositivo». Oculta las secciones vacías. Ver [Android](android.md). |
| Clientes Android / Android TV | Descargas sin conexión en móvil y tableta | 🟨 0.63.0 | Botón en la ficha y en cada episodio solo con reproducción directa; DownloadManager a la carpeta privada, pantalla Descargas, reproducción local con subtítulos incrustados y sincronización pendiente de progreso. La TV no las muestra. Cubierto por tests; aún sin probar en un dispositivo real. |

## Lo que viene, en orden

El orden sale de los planes de `.agents/plans` y de las tareas abiertas; no hay fechas ni versiones comprometidas.

1. **Reproducción avanzada**: 10 bits y DTS/TrueHD sin recodificar en HLS. Probar con archivos reales el desentrelazado y el HDR de 0.55.0, y el AAC 5.1 de 0.56.0 en una TV.
2. **Verificar el flujo «Servidor y web»** de GitHub Actions y su primera ejecución en verde.
3. **Complementos**, todavía en fase de diseño.
4. **DLNA/UPnP**, todavía en fase de diseño.

## Limitaciones y riesgos conocidos

- **Bloqueo de login:** los fallos se cuentan por pareja usuario+IP y hay un límite adicional por IP. Depende de que el proxy reenvíe bien `X-Forwarded-For` y de `CINECAT_TRUSTED_PROXIES`; intentos y bloqueos persisten en SQLite (migración 0024).
- **No hay CORS:** el servidor no envía cabeceras `Access-Control-*`. El despliegue previsto funciona porque Apache sirve `/api/` y la web bajo el mismo origen.
- **Series corregidas a mano:** el escaneo no vuelve a tocar las series `manual` ni `none`; si faltan temporadas o episodios, no se rellenan solos.
- **HLS:** la recodificación usa VA-API si se monta `/dev/dri` y la comprobación funciona; si no, usa libx264 (software). Desde 0.55.0 desentrelaza y convierte HDR a SDR si FFmpeg tiene los filtros, pero aún no se ha probado con archivos reales. Sigue faltando ABR. Las pistas de imagen se queman al recodificar por CPU, nunca en copia ni con VA-API. La copia de vídeo no recodifica y apenas usa CPU.
- **La API sigue llamando `remux` al HLS** aunque copie o recodifique; el cliente recibe HLS en ambos casos.
- **Si no se puede leer la duración**, se usa la playlist `event` de FFmpeg y los saltos lejanos no son rápidos.
- **Subtítulos:** los de texto se convierten a WebVTT; los de imagen incrustados (PGS, VobSub, DVB) se pueden elegir y se queman en el vídeo al recodificar por HLS, nunca en copia ni con VA-API. Los administradores pueden subir `.srt`, `.vtt` o `.ass` de hasta 2 MB (se guardan en `subtitles/`, junto a la base) y se sirven como WebVTT con ids `u<N>`. El HLS no lleva una pista de subtítulos separada.
- **Segmentos de intro y créditos:** se detectan solo por el título de los capítulos o por marcas manuales; no hay detección por huella de audio. La intro debe durar menos de 10 minutos y los manuales sustituyen al capítulo del mismo tipo.
- **Tests HLS reales:** se saltan sin `ffmpeg`/`ffprobe` en PATH; el flujo «Servidor y web» de GitHub Actions instala FFmpeg y ejecuta las pruebas de Go y de la web.
- **TMDB:** el mantenimiento diario vuelve a pedir las fichas con más de 90 días (máximo 40 películas y 20 series al día) para no superar los 6 meses de sus términos; no toca las corregidas a mano. La caché de respuestas de 7 días y las cuotas deben revisarse antes de un uso intensivo. OMDb solo se usa como respaldo y tiene cuota diaria.
- **Cambios en los discos:** subir y mover a la papelera exige montar `/media` en lectura y escritura; la música no se puede subir ni mover.
- **URLs de reproducción:** duran 8 horas y no están ligadas al usuario. La clave HMAC se guarda junto a la base, así que sobreviven a los reinicios del servidor.
- **Actualizador automático:** solo funciona si se instaló el servicio del host; el repositorio no puede comprobar si está activo en un servidor concreto.

## Cómo se actualiza

Manual: `./scripts/actualizar.sh`. Desde la web: Administración → Sistema → «Buscar actualizaciones» y «Actualizar ahora». La instalación del servicio, las copias, el estado y la recuperación manual están en [actualizaciones](actualizaciones.md).

[English](../en/status.md) · [Volver al README](../../README.md)

# TODO

[English version](TODO.en.md) · [Estado del proyecto](docs/es/estado.md)

Prioridades: 🔴 bloqueante · 🟠 próxima fase · 🟢 más adelante.

Este archivo contiene solo trabajo que sigue abierto. Lo ya publicado está en el [CHANGELOG](CHANGELOG.md); el estado completo, en [estado](docs/es/estado.md).

## 🔴 Antes de abrir la instancia a otros usuarios

- **Verificar el acceso por biblioteca con un navegador real** (U4 del [plan](.agents/plans/usuarios-permisos/usuarios-permisos.PLAN.md)) y revisión del conjunto de U2: un olvido de filtro sería una fuga de contenido. La prueba automática de fugas y el recorrido con la base real ya pasaron.

## 🟠 Próxima fase

- **Identificar colecciones en TMDB:** búsqueda manual, sinopsis y «faltan en tu biblioteca».
- **Perfil de usuario:** personalización del portal e historial (la gestión de usuarios y las sesiones abiertas ya existen).
- **Actividad** (historial de reproducciones) en administración.
- **Copiar las copias de seguridad fuera del volumen** (otro disco o equipo): hoy están en `/data/backups`, junto a la base.
- **Subtítulos subidos por el usuario:** requiere diseño previo; hoy solo se leen incrustados de texto o archivos `.srt`/`.vtt`/`.ass` junto al vídeo.
- **Reproducción avanzada:** perfiles por dispositivo, HDR/10 bits, desentrelazado, subtítulos de imagen y audio no soportado.
- **Tests HLS reales y CI:** los tests se saltan sin `ffmpeg`/`ffprobe`; no hay flujo automático que los ejecute en Linux.
- **Revisar el TTL de 7 días de TMDB** contra los términos vigentes antes de dar por buena la caché de producción.
- **Exponer un modo de reproducción `transcode`** en la API; hoy el HLS copiado o recodificado sigue anunciándose como `remux`.
- **App Android y Android TV** (plan en `.agents/plans/apps-android/apps-android.PLAN.md`): Películas, Series, Colecciones y Buscar con diseño de TV (solo la ficha y Cuenta lo tienen); en el móvil, carteles más pequeños, títulos sin cortes y «Colecciones» en una línea; PiP en el móvil; guardar en el servidor la pista elegida; probar AVI por HLS en la TV; HLS con AAC 5.1 cuando el equipo lo admita (hoy el AC-3 se convierte a AAC estéreo); valorar `compileSdk 36`.

## 🟢 Más adelante

- 🟢 **Bloqueo de login compartido entre varias instancias del servidor** (hoy es memoria del proceso).
- 🟢 **Aviso al administrador tras muchos intentos fallidos.**
- 🟢 **Purgar miniaturas de ítems que ya no existen.**
- **Complementos**, todavía en diseño.
- **DLNA/UPnP**, todavía en diseño.
- **Tipos de biblioteca** distintos de vídeo (música, fotos u otros) cuando se aborde el modelo.
- Fotos y presentaciones, televisión en directo y contenido de internet, segmentos de intro/créditos, descargas sin conexión y sincronización entre dispositivos.
- Otras plataformas (Apple, televisores Samsung/LG).

## Descartado y por qué

- **Caddy como proxy propio:** se descartó en 0.3.1; el Apache del host ya gestiona los vhosts y HTTPS.
- **Segmentos fMP4 para la copia:** se descartaron; los contenedores reales y los timestamps hacían inviable un fMP4 fiable. La copia usa MPEG-TS con cortes en fotogramas clave reales; la recodificación sigue usando H.264.
- **OMDb como fuente principal:** TMDB pasó a ser la fuente principal en 0.11.0; OMDb queda como respaldo si no hay clave de TMDB.
- **Que el contenedor se actualice a sí mismo:** no tiene acceso a Docker; la orden la ejecuta un servicio de systemd del host.
- **FilmAffinity y scraping:** no hay API oficial; no se usa.
- **Segmentar AVI copiando el códec:** descartado por timestamps poco fiables; los AVI se transcodifican siempre.
- **`ui.page_size`:** sustituido por columnas × filas en 0.15.0.

## Riesgos aceptados

- Existe una app Android inactiva desde 2015 llamada «CineCat» (F-Droid, catálogo de cine catalán; sin marca registrada). Se mantiene el nombre Cinecat; riesgo de confusión menor.
- **Sin CORS:** el servidor no envía cabeceras `Access-Control-*`; el despliegue depende de que Apache sirva la API y la web bajo el mismo origen.
- **Escritura en los discos:** subir y la papelera exigen `CINECAT_MEDIA_MODE=rw`; la música no se puede subir ni mover a la papelera.
- **Recodificación:** usa VA-API si está disponible y libx264 en caso contrario. La copia de vídeo no recodifica y apenas usa CPU.
- **Caché y cuotas:** el TTL de TMDB es un valor de partida y OMDb tiene cuota diaria; TMDB exige atribución (falta la pantalla «Acerca de»).
- **Series corregidas a mano:** el escaneo no las vuelve a tocar; si faltan temporadas o episodios, no se rellenan solos.
- **URLs de reproducción:** duran 8 horas, no están ligadas al usuario y dejan de valer al reiniciar el servidor.
- **Actualizador automático:** solo funciona si el servicio del host está instalado; el repositorio no puede comprobar su estado en un servidor concreto.

## Documentación abierta

- Código de conducta.
- Activar los avisos privados de seguridad en GitHub; `SECURITY.md` ya los ofrece como canal.
- Capturas de pantalla del cliente web y sustitución del enlace privado a la maqueta; las hará la tarea D2.

## Mantenimiento de documentos

- Mantener sincronizadas las versiones en español e inglés de cada documento.

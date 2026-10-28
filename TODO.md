# TODO

[English version](TODO.en.md) · [Estado del proyecto](docs/es/estado.md)

Prioridades: 🔴 bloqueante · 🟠 próxima fase · 🟢 más adelante.

Este archivo contiene solo trabajo que sigue abierto. Lo ya publicado está en el [CHANGELOG](CHANGELOG.md); el estado completo, en [estado](docs/es/estado.md).

## 🔴 Bloqueante para uso multiusuario

- **No se pueden crear usuarios.** No hay endpoint ni pantalla para crear, listar, cambiar rol, restablecer contraseña o borrar usuarios. Solo existe el `admin` inicial y `--hash-password`.
- **No hay permisos por biblioteca.** Cualquier usuario autenticado ve todas las bibliotecas, ítems, series y colecciones. La tabla `libraries` no tiene propietario ni ACL.

## 🟠 Próxima fase

- **Identificar colecciones en TMDB:** búsqueda manual, sinopsis y «faltan en tu biblioteca».
- **Usuarios y perfil:** CRUD de usuarios, perfil propio, personalización del portal, sesiones abiertas e historial.
- **Sesiones, dispositivos y actividad** en administración.
- **Copiar las copias de seguridad fuera del volumen** (otro disco o equipo): hoy están en `/data/backups`, junto a la base.
- **Subtítulos subidos por el usuario:** requiere diseño previo; hoy solo se leen incrustados de texto o archivos `.srt`/`.vtt`/`.ass` junto al vídeo.
- **Reproducción avanzada:** perfiles por dispositivo, HDR/10 bits, desentrelazado, subtítulos de imagen y audio no soportado.
- **Tests HLS reales y CI:** los tests se saltan sin `ffmpeg`/`ffprobe`; no hay flujo automático que los ejecute en Linux.
- **Revisar el TTL de 7 días de TMDB** contra los términos vigentes antes de dar por buena la caché de producción.
- **Exponer un modo de reproducción `transcode`** en la API; hoy el HLS copiado o recodificado sigue anunciándose como `remux`.

## 🟢 Más adelante

- 🟢 **Bloqueo de login compartido entre varias instancias del servidor** (hoy es memoria del proceso).
- 🟢 **Aviso al administrador tras muchos intentos fallidos.**
- 🟢 **Purgar miniaturas de ítems que ya no existen.**
- **Complementos**, todavía en diseño.
- **DLNA/UPnP**, todavía en diseño.
- **Tipos de biblioteca** distintos de vídeo (música, fotos u otros) cuando se aborde el modelo.
- **Clientes Android y Android TV.** Sin código; plan en `.agents/plans/apps-android/apps-android.PLAN.md` (primero sesiones de dispositivo, vinculación por código, capacidades nativas y contrato estable de la API en el servidor).
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

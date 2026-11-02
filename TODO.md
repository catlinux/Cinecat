# TODO

[English version](TODO.en.md) · [Estado del proyecto](docs/es/estado.md)

Prioridades: 🔴 bloqueante · 🟠 próxima fase · 🟢 más adelante.

Este archivo contiene solo trabajo que sigue abierto. Lo ya publicado está en el [CHANGELOG](CHANGELOG.md); el estado completo, en [estado](docs/es/estado.md).

## 🟠 Próxima fase

- **Reproducción avanzada:** perfiles por dispositivo, subtítulos de imagen y audio no soportado. El desentrelazado y el mapeo a SDR de 0.55.0 ya están probados con FFmpeg real en `hls_real_test.go`; falta verlos en un archivo HDR/entrelazado de la biblioteca.
- **Comprobar en la TV que el audio 5.1 de 0.56.0 se oye bien:** el servidor ya está probado con FFmpeg real (AC-3, E-AC-3 y AAC 5.1, también 5.1(side)); falta el oído en el equipo de sonido.
- **Verificar el flujo «Servidor y web» de GitHub Actions** (`.github/workflows/server-web.yml`, ejecuta los tests de Go con FFmpeg y los de la web): comprobar su primera ejecución en verde.

## 🟢 Más adelante

- 🟢 **Copiar las copias de seguridad fuera del volumen** (otro disco o equipo): hoy están en `/data/backups`, junto a la base. El usuario no lo quiere por ahora.
- **Complementos**, todavía en diseño.
- **DLNA/UPnP**, todavía en diseño.
- **Tipos de biblioteca** distintos de vídeo (música, fotos u otros) cuando se aborde el modelo.
- Fotos y presentaciones, televisión en directo y contenido de internet, segmentos de intro/créditos, descargas sin conexión y sincronización entre dispositivos.
- Otras plataformas (Apple, televisores Samsung/LG).

## Descartado y por qué

- **Modo `transcode` en la API:** el valor `mode` de `GET /api/v1/items/{id}/play` sigue siendo `remux` para el HLS copiado o recodificado. Cambiarlo rompería a las apps ya instaladas, que comparan con `remux`; `reason` ya indica por qué no va en directo.
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
- **Caché y cuotas:** el TTL de TMDB es un valor de partida y OMDb tiene cuota diaria; TMDB exige atribución (visible desde 0.57.0 en el pie de la web y en la cuenta de las apps).
- **Series corregidas a mano:** el escaneo no las vuelve a tocar; si faltan temporadas o episodios, no se rellenan solos.
- **URLs de reproducción:** duran 8 horas, no están ligadas al usuario y dejan de valer al reiniciar el servidor.
- **Actualizador automático:** solo funciona si el servicio del host está instalado; el repositorio no puede comprobar su estado en un servidor concreto.

## Documentación abierta

- Código de conducta.
- Activar los avisos privados de seguridad en GitHub; `SECURITY.md` ya los ofrece como canal.
- Capturas de pantalla del cliente web y sustitución del enlace privado a la maqueta; las hará la tarea D2.

## Mantenimiento de documentos

- Mantener sincronizadas las versiones en español e inglés de cada documento.

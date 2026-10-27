# Cinecat

> Servidor y clientes propios (web, Android y Android TV) para tu biblioteca multimedia: fichas en castellano, reproducción directa o HLS y una interfaz con personalidad propia.

**Idiomas / Languages:** Español · [English](README.en.md)

**Estado:** servidor y cliente web implementados; clientes Android y Android TV sin empezar. Versión `0.22.1`.

Cinecat es para quien guarda su propia biblioteca de vídeo en casa y quiere una interfaz cuidada, control sobre la reproducción y metadatos en español sin depender de un servicio externo. No es un clon: reutiliza lo que es legal de otros proyectos, con créditos, y escribe lo demás desde cero.

## Qué hay ya

- **Biblioteca y escaneo:** bibliotecas locales, análisis de nombres de películas, series, temporadas y episodios, escaneo manual y automático en segundo plano, información técnica del archivo con `ffprobe`.
- **Identificación y fichas:** TMDB en castellano; OMDb como respaldo si no hay clave de TMDB; estados automático, manual y «no es ninguna»; búsqueda, filtros por letra, género, visionado y orden; reparto y equipo con fotos, sinopsis, carteles, fondos y enlaces a IMDb/TMDB.
- **Series:** ficha por carpeta, temporadas y episodios con título, sinopsis e imagen, siguiente episodio, visionado por episodio, temporada o serie, y corrección manual.
- **Colecciones:** sagas automáticas de TMDB y colecciones manuales, con edición, orden, cartel y fondo.
- **Reproducción:** directa para MP4/M4V; HLS para el resto, recodificado a H.264 + AAC estéreo, con saltos, pistas de audio, subtítulos de texto como WebVTT, calidades y preferencias por usuario y película.
- **Interfaz web:** portal con «Continuar viendo», «Siguiente episodio» y «Añadido recientemente», cartelera, series y colecciones con búsqueda y filtros, fichas completas, reproductor propio con atajos, temas claro/oscuro/sistema y menú lateral o superior.
- **Administración:** resumen, bibliotecas, escaneo con progreso, ajustes en caliente, corrección de fichas, imágenes y actualizaciones desde Sistema.
- **Actualizaciones:** el servidor comprueba etiquetas y notas; «Actualizar ahora» deja la orden a un servicio de systemd del host que copia la base, actualiza, comprueba y revierte si falla. Ver [actualizaciones](docs/es/actualizaciones.md).

Lo que queda por hacer, con detalle y orden, está en [estado](docs/es/estado.md).

## Requisitos

- Docker y Docker Compose.
- El contenedor del servidor ya incluye FFmpeg; no hace falta instalarlo en el host.
- Una clave de TMDB es opcional, pero sin ella no habrá fichas completas de TMDB. Si no hay TMDB, se puede usar OMDb solo para sinopsis y cartel de películas.
- Para el despliegue con dominio, Apache con `mod_proxy`, `mod_proxy_http` y `mod_headers`.

Variables del servidor (todas opcionales): `CINECAT_CONFIG`, `CINECAT_LISTEN_ADDR` (`:8080`), `CINECAT_DB_PATH` (`cinecat.db`), `CINECAT_TMDB_API_KEY`, `CINECAT_OMDB_API_KEY`, `CINECAT_SCAN_INTERVAL` (`6h`), `CINECAT_MEDIA_ROOTS` (`/media`), `CINECAT_ADMIN_PASSWORD` y `CINECAT_RUN_DIR` (`/run/cinecat-update`). En Docker, además, se usan `CINECAT_MEDIA_PATH`, `CINECAT_API_PORT` y `CINECAT_WEB_PORT`.

## Instalación y puesta en marcha

```sh
git clone https://github.com/catlinux/Cinecat.git
cd Cinecat
cp .env.deploy.example .env
# Edita .env: al menos CINECAT_MEDIA_PATH y, si quieres, las claves de TMDB/OMDb.
docker compose up -d --build
docker compose logs servidor
```

El primer arranque crea el usuario `admin`. Si no fijaste `CINECAT_ADMIN_PASSWORD`, la contraseña generada aparece una sola vez en ese log. Los contenedores escuchan solo en `127.0.0.1`; el vhost de Apache y HTTPS se explican en [despliegue](docs/es/despliegue.md).

## Actualizar

Actualización manual:

```sh
./scripts/actualizar.sh
```

Actualización desde Administración → Sistema, con copia previa de la base y vuelta atrás automática si la versión nueva no arranca; requiere instalar una vez el servicio del host:

```sh
./scripts/instalar-actualizador.sh
```

Los detalles, requisitos de systemd/linger, copias y recuperación manual están en [actualizaciones](docs/es/actualizaciones.md).

## Dirección visual

**«Marquesina»**: cine clásico con aire actual y luminoso. Granate y crema claro, acentos dorados y naranjas, serif Fraunces para los títulos y carteles ilustrados propios; nunca fotos de terceros. La web usa tipografías empaquetadas localmente y temas claro, oscuro o según el sistema.

## Capturas

<!-- capturas: pendiente de D2 -->

## Documentación

| Tema | Español | English |
|---|---|---|
| Estado y lo que falta | [estado](docs/es/estado.md) | [status](docs/en/status.md) |
| Guía de usuario | [guía de usuario](docs/es/guia-de-usuario.md) | [user guide](docs/en/user-guide.md) |
| Administración | [administración](docs/es/administracion.md) | [administration](docs/en/administration.md) |
| Visión y decisiones | [visión](docs/es/vision.md) | [vision](docs/en/vision.md) |
| Arquitectura y seguridad | [arquitectura](docs/es/arquitectura.md) | [architecture](docs/en/architecture.md) |
| Reproducción y códecs | [reproducción](docs/es/reproduccion.md) | [playback](docs/en/playback.md) |
| Metadatos | [metadatos](docs/es/metadatos.md) | [metadata](docs/en/metadata.md) |
| Actualizaciones | [actualizaciones](docs/es/actualizaciones.md) | [updates](docs/en/updates.md) |
| Despliegue con Docker | [despliegue](docs/es/despliegue.md) | [deployment](docs/en/deployment.md) |
| Hoja de ruta | [hoja de ruta](docs/es/hoja-de-ruta.md) | [roadmap](docs/en/roadmap.md) |
| Licencias y créditos | [licencias y créditos](docs/es/licencias-y-creditos.md) | [licenses and credits](docs/en/licenses-and-credits.md) |

Otros: [CHANGELOG](CHANGELOG.md) · [Tareas abiertas](TODO.md) · [Cómo contribuir](CONTRIBUTING.md) · [Seguridad](SECURITY.md) · [Créditos](CREDITS.md) · [Autores](AUTHORS)

## Licencia

**GPL-3.0**, con una condición adicional (permitida por su artículo 7) que obliga a conservar los avisos de autoría de Catlinux en cualquier copia o versión modificada. Ver [LICENSE](LICENSE) y [licencias y créditos](docs/es/licencias-y-creditos.md).

## Autoría

Autor y mantenedor: **Catlinux** ([github.com/catlinux](https://github.com/catlinux)).

## Atribución de datos

This product uses the TMDB API but is not endorsed or certified by TMDB.
(Cuando existan clientes, esta nota y el logotipo de TMDB aparecerán en la pantalla «Acerca de».)

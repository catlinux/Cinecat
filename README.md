# Cinecat

> Servidor y clientes propios (web, Android y Android TV) para tu biblioteca multimedia: elige el mejor códec (H.265/H.264), reproducción directa o transcodificación, y metadatos en español.

**Idiomas / Languages:** Español · [English](README.en.md)

**Estado:** fase de diseño. Versión `0.0.1` (solo documentación, todavía no hay código).

---

## ¿Qué es Cinecat?

Cinecat es un proyecto para reproducir **tus propios vídeos** (los de tu servidor doméstico) desde la televisión, el ordenador, el móvil y la tableta, con una interfaz propia, cuidada y práctica, y con un servidor que decide por sí solo la mejor forma de reproducir cada vídeo en cada dispositivo.

Nace porque las soluciones existentes funcionan, pero su interfaz y algunas decisiones de reproducción no encajan con lo que queremos. Cinecat no pretende ser un clon: aprovecha lo mejor del software libre que ya existe (siempre con sus créditos y respetando sus licencias) y construye encima algo propio.

## Objetivos

- **Servidor propio** que indexa tu biblioteca, obtiene portadas e información de bases de datos abiertas y sirve el vídeo.
- **Clientes propios:** web (PC, móvil y tableta, también desde fuera de casa) y apps para **Android** y **Android TV**.
- **Reproducción óptima:** cada dispositivo declara qué sabe descodificar y el servidor elige entre reproducción directa, cambio de contenedor o transcodificación, prefiriendo **H.265** y con **H.264** como alternativa. Ver [reproducción](docs/es/reproduccion.md).
- **Metadatos en español de España** a partir de bases de datos ya existentes (no creamos la nuestra). Ver [metadatos](docs/es/metadatos.md).
- **Acceso seguro** con usuarios y contraseña, y PIN para perfiles. Ver [arquitectura](docs/es/arquitectura.md).
- **Interfaz con personalidad propia**, nada de plantillas genéricas.

## Fuera de alcance por ahora

Televisión en directo y contenido de internet. Fotos y presentaciones llegarán en versiones posteriores. Otras plataformas distintas de web, Android y Android TV, también más adelante.

## Documentación

| Tema | Español | English |
|---|---|---|
| Visión y decisiones | [vision](docs/es/vision.md) | [vision](docs/en/vision.md) |
| Arquitectura y seguridad | [arquitectura](docs/es/arquitectura.md) | [architecture](docs/en/architecture.md) |
| Reproducción y códecs | [reproduccion](docs/es/reproduccion.md) | [playback](docs/en/playback.md) |
| Metadatos | [metadatos](docs/es/metadatos.md) | [metadata](docs/en/metadata.md) |
| Licencias y créditos | [licencias-y-creditos](docs/es/licencias-y-creditos.md) | [licenses-and-credits](docs/en/licenses-and-credits.md) |
| Hoja de ruta | [hoja-de-ruta](docs/es/hoja-de-ruta.md) | [roadmap](docs/en/roadmap.md) |

Otros: [CHANGELOG](CHANGELOG.md) · [TODO](TODO.md) · [Cómo contribuir](CONTRIBUTING.md) · [Seguridad](SECURITY.md) · [Créditos](CREDITS.md) · [Autores](AUTHORS)

## Licencia

**Pendiente de elegir.** Como queremos aprovechar código libre de otros proyectos (varios con licencia GPL), la licencia final será una compatible con ese código y que conserve siempre la autoría y los créditos. Se aplicará antes de publicar la primera línea de código. Ver [licencias y créditos](docs/es/licencias-y-creditos.md).

## Autoría

Autor y mantenedor: **Catlinux** ([github.com/catlinux](https://github.com/catlinux)).

## Atribución de datos

This product uses the TMDB API but is not endorsed or certified by TMDB.
(Cuando existan clientes, esta nota y el logotipo de TMDB aparecerán en la pantalla «Acerca de».)

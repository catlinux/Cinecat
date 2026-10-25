# Cinecat

> Servidor y clientes propios (web, Android y Android TV) para tu biblioteca multimedia: elige el mejor códec (H.265/H.264), reproducción directa o transcodificación, y metadatos en español.

**Idiomas / Languages:** Español · [English](README.en.md)

**Estado:** servidor mínimo funcional (Fase 1 completa), sin clientes todavía. Versión `0.1.0`.

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

## Dirección visual

**Elegida: «Marquesina».** Cine de barrio de los años 50-60: rojo granate y crema cálidos, acentos dorados y naranjas, tipografía serif de carácter (Fraunces) para los títulos. Carteles propios ilustrados (ninguno es una foto real) con escenas e iluminación variadas para dar colorido a la biblioteca.

Maqueta de referencia: [artboard "Marquesina" en Claude](https://claude.ai/artifact/J2o3AdQxPbN9woGjUfWQFk) (enlace privado del mantenedor; se sustituirá por capturas en `docs/` cuando haya interfaz real).

## Documentación

| Tema | Español | English |
|---|---|---|
| Visión y decisiones | [vision](docs/es/vision.md) | [vision](docs/en/vision.md) |
| Arquitectura y seguridad | [arquitectura](docs/es/arquitectura.md) | [architecture](docs/en/architecture.md) |
| Reproducción y códecs | [reproduccion](docs/es/reproduccion.md) | [playback](docs/en/playback.md) |
| Metadatos | [metadatos](docs/es/metadatos.md) | [metadata](docs/en/metadata.md) |
| Licencias y créditos | [licencias-y-creditos](docs/es/licencias-y-creditos.md) | [licenses-and-credits](docs/en/licenses-and-credits.md) |
| Hoja de ruta | [hoja-de-ruta](docs/es/hoja-de-ruta.md) | [roadmap](docs/en/roadmap.md) |
| Despliegue con Docker | [despliegue](docs/es/despliegue.md) | [deployment](docs/en/deployment.md) |

Otros: [CHANGELOG](CHANGELOG.md) · [TODO](TODO.md) · [Cómo contribuir](CONTRIBUTING.md) · [Seguridad](SECURITY.md) · [Créditos](CREDITS.md) · [Autores](AUTHORS)

## Licencia

**GPL-3.0**, con una condición adicional (permitida por su artículo 7) que obliga a conservar los avisos de autoría de Catlinux en cualquier copia o versión modificada. Ver [LICENSE](LICENSE) y [licencias y créditos](docs/es/licencias-y-creditos.md).

## Autoría

Autor y mantenedor: **Catlinux** ([github.com/catlinux](https://github.com/catlinux)).

## Atribución de datos

This product uses the TMDB API but is not endorsed or certified by TMDB.
(Cuando existan clientes, esta nota y el logotipo de TMDB aparecerán en la pantalla «Acerca de».)

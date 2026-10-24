# Licencias y créditos

[English](../en/licenses-and-credits.md) · [Volver al README](../../README.md)

> Esto es documentación de trabajo, **no asesoramiento legal**. Antes de la primera versión pública conviene una revisión de licencias.

## Estado

- **Licencia del proyecto: GPL-3.0**, con una condición adicional del artículo 7 que exige conservar los avisos de autoría de Catlinux. Ver [LICENSE](../../LICENSE).
- A fecha de esta decisión **no se ha copiado ningún código** de otros proyectos. Solo se puede copiar código literal de proyectos con licencia GPL-3.0 (o posterior) o MPL-2.0 (compatibles); el código GPL-2.0-only (como Jellyfin y sus clientes oficiales, salvo que un archivo concreto indique "or later") solo puede usarse como **inspiración**, reescribiendo el código desde cero.

## Qué queremos

1. Poder **aprovechar todo lo que sea legal** de otros proyectos libres, citando siempre su origen.
2. Que nuestro código pueda **compartirse, modificarse y estudiarse**, **conservando siempre la autoría (Catlinux) y los créditos**, sin que se puedan borrar.
3. Una licencia que no nos obligue a nada que no podamos cumplir.

## Qué implica reutilizar código GPL

Varios proyectos de referencia (Jellyfin y sus clientes oficiales, Wholphin, Findroid) usan licencias **GPL**. Si se copia código GPL, el resultado combinado debe quedar también bajo GPL. Consecuencias prácticas:

- La licencia de Cinecat será **GPL o compatible con GPL**.
- **GPL-2.0 y GPL-3.0 no siempre se pueden mezclar**: si un archivo es «solo 2.0» no puede combinarse con código «solo 3.0». Hay que mirar la etiqueta de licencia de **cada archivo** (por ejemplo la línea SPDX) antes de copiar.
- Si se reutiliza código de licencias **permisivas o de copyleft débil** (Apache-2.0, MIT, MPL-2.0), se conservan sus avisos y condiciones, y son compatibles con una licencia GPL.
- Lo que no se copie sino que solo se **tome como idea** se reescribe desde cero y se cita como inspiración.

## Créditos que no se pueden borrar

- Los avisos de copyright y licencia de los proyectos reutilizados **se conservan** en cada archivo y en [CREDITS.md](../../CREDITS.md).
- Para el código propio, la GPLv3 permite añadir como condición adicional la **conservación de atribuciones de autor** en los avisos legales que muestran las aplicaciones. Es una de las opciones a valorar al elegir licencia.
- Las apps tendrán una pantalla **«Acerca de»** con autor, versión, licencia y créditos de terceros (incluida TMDB).

## Decisión (2026-10-24)

**GPL-3.0, con la condición adicional del artículo 7** que exige conservar los avisos de autoría de Catlinux en cualquier copia o versión modificada (pantallas «Acerca de», créditos, avisos legales). Texto completo en [LICENSE](../../LICENSE).

Licencias verificadas de los proyectos candidatos a reutilización (en su repositorio oficial, no de memoria):

| Proyecto | Licencia confirmada | Qué implica para Cinecat |
|---|---|---|
| Jellyfin (servidor) | GPL-2.0 | Solo inspiración; no copiar código literal sin comprobar archivo a archivo si está marcado «or later» |
| Jellyfin Android TV | GPL-2.0 | Solo inspiración |
| Wholphin | GPL-2.0 | Solo inspiración |
| Findroid | GPL-3.0 | Se puede copiar literal, con créditos y avisos de copyright |
| Intro Skipper | GPL-3.0 | Se puede copiar literal, con créditos y avisos de copyright |
| Streamyfin | MPL-2.0 | Compatible; se puede copiar literal respetando MPL-2.0 en los archivos afectados |
| Media3/ExoPlayer, hls.js, Shaka Player | Apache-2.0 | Compatible; se puede copiar literal con sus avisos |
| Vidstack | MIT | Compatible; se puede copiar literal con sus avisos |

Se aplica con `LICENSE` en la raíz, cabeceras SPDX (`SPDX-License-Identifier: GPL-3.0-or-later`) en los archivos de código propio, y campo de licencia en los metadatos de los paquetes cuando corresponda.

## Reglas para reutilizar código de terceros

1. Verificar la licencia **en la versión exacta** que se copia (no en la web del proyecto).
2. Anotar en `CREDITS.md` y `THIRD_PARTY_NOTICES.md`: proyecto, URL, versión o *commit*, licencia, archivos afectados.
3. Conservar los avisos de copyright originales en los archivos.
4. No hacer *fork* ni importar el historial de otros repositorios: el código se incorpora en *commits* nuevos del repositorio (el historial de Cinecat solo muestra a Catlinux).
5. Lo que no se pueda reutilizar legalmente se escribe de nuevo (sin copiar su estructura).

## Fuentes de datos y marcas

- **TMDB**, **TheTVDB**, **Fanart.tv**: uso sujeto a sus términos. Se muestra la atribución exigida.
- Las marcas y logotipos de terceros pertenecen a sus propietarios.
- **FFmpeg** se usa como herramienta externa y no se distribuye en el repositorio.

## Lista de proyectos

Ver [CREDITS.md](../../CREDITS.md).

# Licencias y créditos

[English](../en/licenses-and-credits.md) · [Volver al README](../../README.md)

> Esto es documentación de trabajo, **no asesoramiento legal**. Antes de la primera versión pública conviene una revisión de licencias.

## Estado

- **Licencia del proyecto: pendiente de elegir.** Se decide **antes** de publicar la primera línea de código.
- A fecha 0.0.1 **no se ha copiado ningún código** de otros proyectos.

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

## Opciones de licencia a valorar (Fase 0)

| Opción | Ventajas | A tener en cuenta |
|---|---|---|
| GPL-3.0 (con atribuciones adicionales) | Copyleft fuerte; permite condición de conservar autoría. | No mezclable con código «solo GPL-2.0». |
| GPL-2.0 | Compatible con código de Jellyfin si es «solo 2.0». | Menos modernizada; sin cláusulas de patentes de la 3.0. |
| AGPL-3.0 | Cierra el hueco de servicios en red (útil para servidor). | Mismas limitaciones de mezcla que GPL-3.0. |

La decisión se toma tras listar **qué código concreto** se reutilizará. Se aplica con `LICENSE`, cabeceras SPDX en los archivos y campo de licencia en los metadatos de los paquetes.

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

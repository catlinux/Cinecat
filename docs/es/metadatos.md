# Metadatos

[English](../en/metadata.md) · [Volver al README](../../README.md) · [Estado](estado.md)

Cinecat no crea una base de datos propia: usa TMDB como fuente principal y, si no hay clave de TMDB, OMDb como respaldo para películas. La interfaz y los nombres propios se mantienen en castellano.

## Fuentes

| Fuente | Uso real | Notas |
|---|---|---|
| **TMDB** | Películas y series: título, título original, sinopsis, reparto, equipo, géneros, nota, carteles, fondos, stills, temporadas, episodios y colecciones. | Fuente principal desde 0.11.0. La clave se configura con `CINECAT_TMDB_API_KEY`. |
| **OMDb** | Solo películas: sinopsis y cartel. | Respaldo cuando no hay clave de TMDB. Tiene cuota diaria. No se usa para series ni para identificar. |
| TheTVDB, Fanart.tv, TVmaze | No integrados. | Se documentaron como posibles fuentes, pero no hay código que los use. |
| FilmAffinity | No se usa. | No tiene API oficial y no se hace scraping. |

## Identificación

- **Películas:** se generan variantes del título del archivo y de la carpeta, se buscan en TMDB y se puntúan por coincidencia de palabras, año y reglas de sagas. El umbral por defecto es `0.75` y se configura en `metadata.match_threshold`.
- **Series:** la serie es la carpeta de primer nivel. Se identifica con TMDB y después se piden temporadas y episodios. Una carpeta que sea directamente la raíz de una biblioteca no crea una serie, porque no hay carpeta de primer nivel.
- **Estados:** `auto` (identificada por el escaneo), `manual` (corregida por un administrador), `none` («no es ninguna») o vacío (sin identificar, se reintenta). El escaneo no toca `manual` ni `none`.
- **Corrección:** el administrador puede buscar candidatos, elegir uno, marcar «no es ninguna» o volver a automático. En series, la corrección vuelve a pedir temporadas y episodios.

No se leen archivos `.nfo` locales ni se usan certificaciones por edad: esas funciones no están implementadas.

## Idioma

`metadata.language` acepta `es-ES`, `es-MX` y `en-US`, con `es-ES` por defecto. Afecta a las peticiones nuevas; no reescribe fichas ya guardadas.

Para las sinopsis, si el idioma preferido no tiene texto se prueban variantes: en español, la otra variante, `es` y `en-US`; en inglés, `en-US`; en el resto, el idioma base y `en-US`. La web muestra «Sinopsis no disponible en castellano» si la sinopsis guardada no empieza por `es`.

## Imágenes

- Prioridad de cartel: `es-ES` → español genérico → inglés → sin idioma; dentro de cada idioma, mayor votación.
- Tamaños habituales: póster `w500`, listas `w342`/`w780`, fondos `w780`/`w1280`, stills `w300` y póster de temporada `w342`.
- Las imágenes manuales quedan bloqueadas frente al escaneo. Si cambia el `tmdb_id`, se desbloquean.
- Si una temporada ya se consultó y TMDB no tenía póster, se guarda una marca vacía para no repetir la consulta.

## Reparto y equipo

TMDB devuelve reparto y equipo; se guardan hasta 30 créditos de reparto y hasta 12 de equipo en un orden fijo. La ficha muestra el personaje, la foto y el nombre; si falta la foto, se usan iniciales. Las series también guardan créditos.

## Caché

- **TMDB:** respuestas crudas e imágenes en `<directorio de la base>/cache/tmdb`; TTL de 7 días. Los 404 también se cachean para no repetir búsquedas fallidas. La clave de caché no incluye `api_key`.
- **OMDb:** caché en disco de 30 días en el directorio de caché del usuario; las búsquedas no encontradas también se guardan.
- El TTL de 7 días de TMDB es un valor de partida y no se ha verificado contra los términos vigentes; conviene revisarlo antes de producción.

## Colecciones

TMDB aporta `belongs_to_collection`. Cinecat crea o actualiza la colección, respeta las que el usuario haya borrado o editado y solo muestra las automáticas que alcanzan `ui.collection_min_items` (2 por defecto). Las colecciones manuales no se tocan desde TMDB.

## Atribución

This product uses the TMDB API but is not endorsed or certified by TMDB.

La atribución y el logotipo deben aparecer en la pantalla «Acerca de» cuando exista; hoy no hay una pantalla «Acerca de» en la web. Los créditos de las fuentes y dependencias están en [CREDITS.md](../../CREDITS.md).

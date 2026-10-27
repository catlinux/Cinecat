# Administración

[English](../en/administration.md) · [Volver al README](../../README.md) · [Guía de usuario](guia-de-usuario.md)

Solo los usuarios con rol `admin` pueden entrar en `/admin`. El servidor comprueba el rol en cada petición; la web no es la que protege los datos.

## Pantallas

| Ruta | Pantalla | Qué permite |
|---|---|---|
| `/admin` | Resumen | Versiones del servidor y de la web, último escaneo, recuentos de películas, series y colecciones, y enlaces a los títulos sin identificar. |
| `/admin/libraries` | Bibliotecas | Crear bibliotecas con nombre, tipo y varias carpetas, elegir carpetas en el servidor, renombrar, añadir o quitar carpetas, escanear y borrar. |
| `/admin/settings/scan` | Escaneo | Intervalo y escaneo al arrancar. |
| `/admin/settings/metadata` | Metadatos | Idioma de TMDB y umbral de identificación. |
| `/admin/settings/ui` | Interfaz | Nombre, textos, menú, rejilla, orden, colecciones, series, tema y acento. |
| `/admin/settings/playback` | Reproducción | Altura máxima, CRF, preset y transcodificaciones simultáneas. |
| `/admin/settings/system` | Sistema | Búsqueda de actualizaciones, versiones y panel «Actualizar ahora». |
| `/admin/update` | — | Endpoint JSON que usa el panel de Sistema para seguir el estado; no es una pantalla. |
| `/admin/users` | — | No existe todavía. La navegación muestra «Usuarios · Pronto». |

## Crear una biblioteca

1. Abre **Administración → Bibliotecas**.
2. En «Nueva biblioteca», escribe un **Nombre** y elige el **Tipo**: «Películas y series» o «Vídeos (videoclips, conciertos…)». «Música» sigue pendiente.
3. Escribe la ruta de la primera carpeta o pulsa **Elegir…** para navegar por las carpetas del servidor. Con **Añadir otra carpeta** puedes añadir más; todas se guardan en la misma biblioteca.
4. Pulsa **Crear biblioteca**. Las rutas deben existir y ser directorios dentro de las raíces permitidas; por ejemplo `/media/Multimedia/Pelis`.
5. En la tarjeta de cada biblioteca puedes **Renombrar**, **Añadir carpeta**, **Quitar** una carpeta, **Escanear** o **Borrar biblioteca**. Al borrar hay que escribir el nombre para confirmar.

El escaneo sigue en segundo plano aunque cierres la página. El botón **Escanear** de una tarjeta inicia el mismo escaneo de todas las bibliotecas.

### Varias carpetas por biblioteca

Una biblioteca puede agrupar varias carpetas, incluso de discos distintos. La primera carpeta se conserva como carpeta principal por compatibilidad; al quitar la principal, pasa a serlo la siguiente más antigua. Una biblioteca puede ser de tipo `video` (películas y series), `clips` (vídeos sin identificar) o `music` (música). Los tipos `video` y `clips` se escanean; `music` sigue pendiente y se omite con un aviso.

### Bibliotecas de Vídeos (sin identificar)

Una biblioteca de tipo `clips` recorre sus carpetas igual que una de películas, pero no identifica nada: no consulta TMDB ni OMDb, no deduce año, no crea series ni colecciones y no pide metadatos externos. El título sale del nombre del archivo, cambiando `_` y `.` por espacios; por ejemplo, `2019-08-15_Cumple.de.Ana.mp4` se muestra como «2019-08-15 Cumple de Ana».

Los vídeos de estas bibliotecas no se mezclan con las películas en listados, búsquedas, filtros, recomendaciones, colecciones, recuentos del portal ni del resumen de administración. Sí conservan visto, progreso, preferencias y reproducción, y pueden aparecer en «Continuar viendo». La sección **Vídeos** de la web permite navegar por sus carpetas y subcarpetas, abrir una ficha mínima y reproducirlos. El tipo Música sigue pendiente.

El botón **Elegir…** abre un navegador de carpetas del servidor. Muestra las raíces permitidas, permite bajar por las subcarpetas, subir con **Subir**, moverse con las migas de pan y confirmar con **Elegir esta carpeta**. Solo lista directorios, no sigue enlaces que salgan de las raíces y limita el listado a 1000 entradas. También puedes escribir la ruta a mano sin usar JavaScript. El contenedor monta `CINECAT_MEDIA_PATH` completo y en solo lectura en `/media`.

Las carpetas se validan contra `CINECAT_MEDIA_ROOTS`: una lista de raíces separadas por `:` en Linux y `;` en Windows (por defecto `/media`). Una ruta fuera de esas raíces, inexistente, que no sea un directorio, repetida o solapada con otra se rechaza. Quitar una carpeta borra sus fichas de la base de datos, pero no toca los archivos del disco.

**Protección contra discos sin montar.** El escaneo solo borra fichas de las carpetas que ha recorrido con éxito. Si una carpeta no está disponible, o se recorre sin encontrar ni un archivo aunque la base tiene fichas suyas (lo que pasa con un disco sin montar, cuyo punto de montaje queda como carpeta vacía), conserva las fichas, con su visto y progreso, y lo avisa en el resultado del escaneo. Consecuencia: si vacías de verdad una carpeta entera, sus fichas no se borran solas; quita la carpeta de la biblioteca para eliminarlas.

## Escaneo e identificación

- El escaneo recorre las bibliotecas, ignora archivos y carpetas ocultas y no sigue enlaces simbólicos.
- Reconoce `S01E02`, `1x02`, «Temporada N», «Episodio N» y formatos como `ChicagoFire720p4x17` o `MR_Robot_305` dentro de una carpeta de temporada.
- Las películas se identifican con TMDB; si no hay clave de TMDB se usa OMDb solo para sinopsis y cartel de películas. Sin clave, los archivos se indexan pero no se identifican.
- `match_status`: `auto` (identificada por el escaneo), `manual` (corregida por un administrador), `none` («no es ninguna») o vacío (sin identificar y se reintenta).
- Las series se agrupan por la carpeta de primer nivel. Una serie corregida a mano no se vuelve a tocar en escaneos posteriores; si faltan temporadas o episodios, no se rellenan solos.
- Los avisos del escaneo aparecen en un desplegable al terminar. Los errores de una temporada no abortan el resto.

## Corregir una ficha

En la ficha de una película o serie, como administrador, abre **Corregir ficha**:

1. Ajusta el **Título** y, si quieres, el **Año**.
2. Pulsa **Buscar en TMDB** y elige **Es esta**.
3. **No es ninguna** deja la ficha vacía y no se reintenta.
4. **Volver a automático** limpia la corrección y reintenta la identificación por el nombre del archivo.

En **Cambiar imagen** puedes fijar una URL de cartel o fondo, elegir una de las que ofrece TMDB o **Restaurar la de TMDB**. Las imágenes fijadas a mano no las pisa el escaneo.

## Ajustes

Los ajustes se guardan en la base de datos con validación por tipo. Se aplican sin reiniciar el servidor, aunque algunos solo afectan a la siguiente operación: el intervalo al siguiente ciclo, el idioma a las peticiones nuevas y la calidad a las transcodificaciones nuevas. Cada ajuste muestra su valor por defecto y se puede restablecer.

| Clave | Nombre | Descripción | Tipo y rango / opciones | Defecto | Aplicación |
|---|---|---|---|---|---|
| `scan.interval` | Intervalo de escaneo | Cada cuánto se escanea sola la biblioteca. Usa «0» para desactivarlo; el mínimo es 1 minuto. | Duración Go; `0` o mínimo `1m`; sin máximo | `6h` | En caliente; reprograma el siguiente escaneo. |
| `scan.on_startup` | Escanear al arrancar | Lanza un escaneo un minuto después de iniciar el servidor. | Booleano | `true` | En caliente; solo si el intervalo no es `0`. |
| `metadata.language` | Idioma de los metadatos | Idioma preferido para las sinopsis y fichas nuevas de TMDB; no reescribe las fichas ya guardadas. | `es-ES`, `es-MX`, `en-US` | `es-ES` | En caliente; solo afecta a peticiones nuevas. |
| `metadata.match_threshold` | Umbral de identificación | Confianza mínima para aceptar automáticamente una coincidencia. | Decimal `0.6`–`0.95` | `0.75` | En caliente; siguiente identificación. |
| `ui.server_name` | Nombre del servidor | Nombre que aparece en la interfaz. | Texto 1–40 | `Cinecat` | En caliente; siguiente página. |
| `ui.tagline` | Eslogan | Frase corta junto al nombre en la cabecera. Vacía para ocultarla. | Texto 0–60 | `Sesión continua` | En caliente; siguiente página. |
| `ui.portal_title` | Título del portal | Título principal de la página de inicio. | Texto 1–60 | `Tu cine en casa` | En caliente; siguiente página. |
| `ui.portal_intro` | Texto del portal | Párrafo bajo el título de la página de inicio. Vacío para ocultarlo. | Texto 0–240 | `Una biblioteca personal con olor a palomitas, carteles pintados a mano y la sesión de siempre esperándote.` | En caliente; siguiente página. |
| `ui.nav_layout` | Menú de navegación | Posición del menú principal: lateral en escritorio o superior. | `side` (lateral), `top` (superior) | `side` | En caliente; siguiente página. |
| `ui.grid_columns` | Columnas de los listados | Carátulas por fila en películas, series y colecciones. | Entero 4–8 | `6` | En caliente; siguiente página. |
| `ui.grid_rows` | Filas por página | Filas de carátulas por página; el número de resultados es columnas × filas. | Entero 2–12 | `8` | En caliente; siguiente página. |
| `ui.row_columns` | Carátulas por fila en el portal | Cuántas carátulas se ven a la vez en las filas desplazables. | Entero 3–8 | `6` | En caliente; siguiente página. |
| `ui.default_sort` | Orden por defecto | Criterio de orden inicial de las películas. | `title`, `year`, `added` | `title` | En caliente; la web actual aún usa Título como orden inicial. |
| `ui.show_collections` | Mostrar colecciones | Muestra la sección de colecciones en la interfaz. | Booleano | `true` | En caliente; oculta el enlace del menú. |
| `ui.show_series` | Mostrar series | Muestra la sección de series en la interfaz. | Booleano | `true` | En caliente; oculta el enlace del menú. |
| `ui.collection_min_items` | Mínimo para colecciones | Número mínimo de películas para mostrar una colección automática. | Entero 1–10 | `2` | En caliente; siguiente listado de colecciones automáticas. |
| `ui.theme_default` | Tema por defecto | Tema inicial cuando no hay una preferencia guardada. | `system`, `light`, `dark` | `system` | En caliente; se usa si el navegador no tiene preferencia. |
| `ui.accent` | Color de acento | Color de acento de la interfaz. | `granate`, `naranja`, `dorado` | `granate` | En caliente; siguiente página. |
| `playback.max_height` | Altura máxima | Resolución máxima de los remux. | `480`, `720`, `1080` | `1080` | En caliente; nuevas transcodificaciones. |
| `playback.crf` | Calidad (CRF) | Calidad de compresión del remux; un valor menor da más calidad y tamaño. | Entero 18–28 | `20` | En caliente; nuevas transcodificaciones. |
| `playback.preset` | Velocidad de codificación | Preset de x264 usado al remuxear. | `veryfast`, `faster`, `fast` | `veryfast` | En caliente; nuevas transcodificaciones. |
| `playback.max_transcodes` | Transcodificaciones simultáneas | Cuántas películas se pueden recodificar a la vez. Más de las que aguante el procesador provoca cortes al avanzar. | Entero 1–8 | `2` | En caliente; siguiente proceso. |
| `system.check_updates` | Buscar actualizaciones | Comprueba si hay versiones nuevas; implica una conexión saliente opcional que debe activar el administrador. | Booleano | `false` | En caliente; activa la comprobación periódica y la inmediata al guardar. |

`scan.interval` puede tener un valor inicial distinto si se define `CINECAT_SCAN_INTERVAL`; el valor guardado en la base manda sobre el defecto de entorno.

## Variables de entorno

Leídas por el servidor:

| Variable | Defecto | Uso |
|---|---|---|
| `CINECAT_CONFIG` | vacío | Ruta a un archivo JSON de configuración. |
| `CINECAT_LISTEN_ADDR` | `:8080` | Dirección y puerto HTTP. |
| `CINECAT_DB_PATH` | `cinecat.db` | Archivo SQLite. |
| `CINECAT_TMDB_API_KEY` | vacío | Activa TMDB; tiene prioridad sobre OMDb. |
| `CINECAT_OMDB_API_KEY` | vacío | Activa OMDb solo si no hay TMDB. |
| `CINECAT_SCAN_INTERVAL` | `6h` | Defecto inicial de `scan.interval`. |
| `CINECAT_MEDIA_ROOTS` | `/media` | Raíces permitidas para las carpetas de biblioteca y el navegador; separadas por `:` en Linux y `;` en Windows. |
| `CINECAT_ADMIN_PASSWORD` | vacío | Contraseña del `admin` creado en el primer arranque; si falta, se genera y se imprime una vez. |
| `CINECAT_RUN_DIR` | `/run/cinecat-update` | Directorio compartido para las peticiones de actualización. |

En Docker, además: `CINECAT_MEDIA_PATH` (ruta del host montada en `/media`), `CINECAT_MEDIA_ROOTS` (raíces visibles dentro del contenedor), `CINECAT_API_PORT` (`8080`) y `CINECAT_WEB_PORT` (`8081`). `PUBLIC_API_URL` y `PROTOCOL_HEADER` son del contenedor web. No hay variables para la ruta de FFmpeg, CORS, usuarios ni límites de HLS.

## Actualizar el servidor

- **Manual:** `./scripts/actualizar.sh` en la raíz del repositorio. Hace `git pull --ff-only`, reconstruye y levanta; la base y las bibliotecas se conservan.
- **Desde la web:** Administración → Sistema. «Buscar actualizaciones» consulta al servidor; «Actualizar ahora» pide al servicio del host que haga copia de la base, actualice a la etiqueta, compruebe que arranca y revierta si falla.
- La instalación del servicio se hace una vez con `./scripts/instalar-actualizador.sh` y requiere Docker accesible por el usuario, `git`, `sqlite3`, `curl`, `python3`, `flock` y linger. Los detalles están en [actualizaciones](actualizaciones.md).
- El panel muestra la fase, el tiempo transcurrido y el resultado; durante el reinicio es normal que la web no responda unos instantes.

## Limitaciones de seguridad

- El bloqueo de login es por nombre de usuario, no por IP: se puede bloquear temporalmente una cuenta conocida.
- No hay permisos por biblioteca: cualquier usuario autenticado ve todo el catálogo.
- No hay pantalla para crear usuarios ni para revocar sesiones.
- El servidor no envía CORS; el despliegue con Apache funciona por compartir origen.
- `/media` está en solo lectura: la aplicación no sube, mueve ni borra archivos.

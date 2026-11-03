# Guía de usuario

[English](../en/user-guide.md) · [Volver al README](../../README.md) · [Estado del proyecto](estado.md)

Esta guía explica cómo usar la web de Cinecat. Está pensada para una persona que solo quiere ver su biblioteca; la administración se explica en [administración](administracion.md).

## Entrar y salir

1. Abre la dirección de tu Cinecat (por ejemplo, `https://media.techmarc.es`).
2. En la pantalla «Entrar», escribe tu **Usuario** y **Contraseña** y pulsa «Entrar en la sala».
3. Si los datos son correctos, llegarás al portal. Si fallas varias veces, el servidor puede bloquear temporalmente el intento: espera unos minutos y vuelve a probar.

Para cerrar sesión, entra en `/login` y pulsa «Cerrar sesión». La sesión se guarda en una cookie `httpOnly`; el navegador no guarda el token en `localStorage`.

## El portal

El portal es la página de inicio. Arriba verás el nombre del servidor, el eslogan y el texto de bienvenida que haya configurado el administrador. Después aparecen los accesos a **Películas**, **Colecciones**, **Series** y, si existen bibliotecas de Vídeos o Música con contenido, **Vídeos** y **Música**. Películas, Colecciones y Series se ocultan cuando tu cuenta no tiene contenido en ellas; si la consulta falla, se muestran todas.

Debajo se muestran, solo si tienen contenido:

- **Continuar viendo**: lo que dejaste a medias.
- **Siguiente episodio**: el próximo capítulo de tus series.
- **Añadido recientemente**: las películas más nuevas de la biblioteca.

Las filas se pueden desplazar con las flechas `‹` y `›`; con el dedo o el trackpad también puedes deslizar. Desde **Mi cuenta → Portal de inicio** puedes ocultar las filas que no quieras ver; la elección es solo tuya y no cambia el portal de las demás cuentas.

## Navegar por la biblioteca

El menú puede estar a la izquierda (**lateral**) o arriba (**superior**), según lo configure el administrador. Contiene **Inicio**, **Películas**, **Colecciones**, **Series**, **Vídeos** y **Música** cuando tu cuenta tiene acceso a una biblioteca de esos tipos, **Fotos** cuando hay alguna biblioteca de fotos visible, **Historial**, **Mi cuenta** y, si eres administrador, **Administración**.

### Películas

La pantalla «Películas» muestra la cartelera paginada. Puedes:

- buscar por **Todo**, **Título**, **Reparto** o **Dirección**;
- filtrar por **Género**;
- elegir el **Orden**: Título, Año o Recientes;
- filtrar por **Visionado**: Todas, Sin ver o Vistas;
- pulsar una letra (`#` o `A`–`Z`) para filtrar por inicial;
- quitar un filtro con su `×` o con «Limpiar filtros».

El número de resultados por página depende de **columnas × filas** configurado en administración (por defecto 6 × 8 = 48). La paginación aparece cuando hay más de una página.

Si eres administrador verás además «Sin identificar (N)», que muestra las películas que el escaneo no ha podido identificar.

### Colecciones

«Colecciones» agrupa sagas y ciclos. Puedes buscar por nombre y ordenar por **Nombre**, **Añadidas** o **Tamaño**. Al abrir una colección verás su descripción, sus películas y un botón para reproducir la primera sin ver (o la primera, si ya están todas vistas).

Si eres administrador, dentro de una colección aparece «Gestionar colección» para cambiar nombre, descripción, cartel o fondo, añadir o quitar películas, reordenarlas y borrar la colección. También incluye **Identificación en TMDB**: puedes buscar la colección por nombre, vincularla con un candidato o desvincular una colección manual. Si está identificada, la ficha muestra las películas de la saga que faltan en tu biblioteca visible, con cartel y año.

### Series

«Series» muestra las series paginadas. Puedes buscar, filtrar por género, elegir orden y filtrar por letra. Al abrir una serie verás su ficha y el selector de temporadas; cada temporada muestra sus episodios con título, sinopsis, imagen y duración.

Los botones de visionado permiten marcar un episodio, una temporada o la serie completa como vistos. El botón principal de la serie cambia según tu progreso: «Empezar», «Continuar» o «Volver a ver». Si eres administrador, al final de la ficha aparecen **Cambiar imagen** y **Corregir ficha**.

### Vídeos

La sección **Vídeos** aparece en el menú cuando existe una biblioteca de tipo Vídeos con contenido. Para crearla, entra en **Administración → Bibliotecas**, elige el tipo «Vídeos (videoclips, conciertos…)», añade sus carpetas y lanza el escaneo.

Este tipo no identifica los archivos ni descarga carátulas de TMDB u OMDb: el título sale del nombre del archivo. Al abrir la sección verás las carpetas de la biblioteca, con el número de vídeos que contienen (incluidas sus subcarpetas). Dentro de una carpeta, las subcarpetas aparecen primero y después los vídeos en tarjetas apaisadas. Las miniaturas se generan al verlas por primera vez y se guardan en caché; si aún no hay miniatura, se usa una portada de respaldo generada. Las migas de pan permiten volver a cualquier nivel. Puedes reproducir o continuar un vídeo, marcarlo como visto y cambiar audio, subtítulos o calidad.

### Música

La sección **Música** aparece cuando hay una biblioteca de ese tipo con pistas. Para crearla, entra en **Administración → Bibliotecas**, elige el tipo «Música», añade sus carpetas y lanza el escaneo. La estructura recomendada es `Artista/Álbum/Pista`: el servidor lee con `ffprobe` el título, artista, artista del álbum, álbum, pista, disco, año y género. Si faltan etiquetas usa el nombre del archivo y de las carpetas. Se admiten MP3, FLAC, M4A, AAC, OGG, OGA, OPUS y WAV; no se indexan WMA, APE ni otros formatos. La carátula sale de `cover`, `folder`, `front` o `album` en la carpeta del álbum o, si no hay, de la imagen incrustada en la primera pista.

En **Música** puedes cambiar entre **Álbumes**, **Artistas** y **Pistas**, buscar por texto, filtrar por letra y ordenar los álbumes por título, artista, año o añadidos. La vista de pistas solo muestra resultados cuando escribes una búsqueda. Al abrir un álbum verás su carátula, año, número de pistas, duración total y la lista de pistas; con **Reproducir**, **Aleatorio** o **Añadir a la cola** puedes empezar a escuchar. En la página de un artista, **Reproducir todo** encola sus álbumes en orden.

El reproductor de música es una barra fija en la parte inferior que sigue sonando al navegar por la web. Muestra la portada, el título y el artista, permite pausar, avanzar o retroceder, mover el progreso, cambiar el volumen o silenciar, activar el aleatorio y repetir (no, todas o una), y abrir la **Cola** para quitar pistas o vaciarla. La cola, el volumen, el aleatorio y el repetir se recuerdan al recargar; la reproducción se restaura en pausa. Si una pista falla o ha caducado su enlace, se reintenta una vez y, si sigue fallando, se salta con un aviso. Al reproducir un vídeo, la música se pausa automáticamente y no se reanuda sola. No hay atajos nuevos para la barra: usa sus botones, los controles multimedia del sistema y los botones de reproducción de las pistas.

### Fotos

La sección **Fotos** aparece en el menú cuando tu cuenta tiene alguna biblioteca de tipo Fotos visible. Para crearla, entra en **Administración → Bibliotecas**, elige el tipo «Fotos», añade sus carpetas y lanza el escaneo.

Si solo hay una biblioteca de fotos, la sección abre directamente su raíz; si hay varias, muestra una lista con su nombre y el número de fotos. Dentro de una biblioteca, las **migas de pan** permiten volver a cualquier nivel. Primero aparecen las subcarpetas y después las fotos en una cuadrícula paginada; puedes cambiar el **Orden** entre Fecha y Nombre. Las miniaturas se cargan al acercarte a ellas.

Pulsa una foto para abrir el visor a pantalla completa. Las flechas `←` y `→` cambian de foto y `Escape` cierra el visor; también puedes usar los botones **Anterior** y **Siguiente**. El visor muestra el contador y el nombre y la fecha de captura cuando existe, y **Presentación automática** avanza cada 5 segundos; **Abrir original en otra pestaña** muestra el archivo original.

## Favoritos y próximas a ver

Cada usuario tiene sus propias listas de **Favoritos** y **Próximas a ver**. Puedes marcar películas, vídeos, series y colecciones desde sus fichas, con los botones **Favorito** y **Próximas a ver**, o desde el corazón de sus tarjetas. En las tarjetas el corazón aparece al pasar el ratón, al enfocar la tarjeta o siempre en pantallas táctiles.

Las dos listas están en el menú, debajo de las categorías. **Favoritos** reúne lo que has guardado; **Próximas a ver** es tu lista para más tarde. Cada página separa Películas, Vídeos, Series y Colecciones, y permite paginar cada bloque. En el portal aparecen las filas **Próximas a ver** y **Tus favoritos** cuando tienen contenido.

Las listas no se comparten entre usuarios. Cuando marcas una película o un vídeo como visto, el servidor lo quita automáticamente de **Próximas a ver** (no de Favoritos); cuando terminas una serie completa, también desaparece de esa lista. Volver a marcarlo como no visto no lo añade de nuevo: tendrás que marcarlo manualmente si quieres recuperarlo.

## Historial

**Historial** reúne lo último que has reproducido o marcado como visto, con la fecha más reciente. Cada entrada muestra la carátula, el título (y la serie, temporada y episodio si es un episodio), el progreso o la marca **Visto**, y un enlace a su ficha. **Quitar del historial** borra tu progreso y tu marca de visto de ese elemento; no afecta a las demás cuentas. Solo aparecen contenidos de las bibliotecas a las que tienes acceso. Si todavía no has visto nada, la página lo indica.

## Personas, puntuaciones y tráileres

En las carátulas de películas y series, una insignia en la esquina superior derecha muestra la puntuación de TMDB (por ejemplo, «7,4»). Un icono en la esquina inferior izquierda indica que la película está en una colección; en Películas puedes filtrar por **En una colección** o **Sin colección** para encontrar las que no se han agrupado solas.

En la ficha, el reparto y el equipo son enlaces: al pulsar a una persona se abre su página con **En tu biblioteca** (sus películas y series que tienes, con su papel) y, aparte, **No la tienes** (el resto de su filmografía, solo informativa). Si la ficha tiene tráiler, aparece **Ver tráiler**, que abre YouTube en otra pestaña: en castellano si existe y, si no, en inglés. Las fichas anteriores a esta versión no tienen ni enlaces ni tráiler hasta que el escaneo las vuelve a pedir a TMDB.

## Subir contenido (administradores)

Con la subida activada (ver [administración](administracion.md#subir-archivos-y-papelera)), los administradores ven botones discretos: **Crear nueva serie** en Series, **Nueva temporada** en la ficha de una serie, **Subir capítulo** en cada temporada y **Subir película** en Películas. No hay que elegir carpetas: Cinecat guarda cada cosa en su sitio y nombra los capítulos como `Serie S01E05`. La web detecta temporada y episodio en el nombre del archivo y puedes corregirlos antes de subir. Desde la ficha de una película, **Gestionar colecciones** permite añadirla a una colección, crear una nueva con ella o quitarla. En el panel **Archivo** de una película o episodio, los administradores pueden subir subtítulos externos `.srt`, `.vtt` o `.ass` de hasta 2 MB, indicar idioma y etiqueta, y eliminarlos; esto no requiere activar la subida general y aparecen en la reproducción con un identificador `u<N>`.

## La ficha de una película

La ficha muestra el cartel, el título (y el original si es distinto), año, duración, géneros, nota, lema, sinopsis, reparto y equipo, enlaces a IMDb/TMDB y las colecciones a las que pertenece. Al final está **Más como este**.

- **Reproducir** empieza desde el principio; **Continuar** retoma donde lo dejaste. **Desde el principio** ignora el progreso guardado.
- **Marcar como vista** / **Desmarcar** cambia el estado de visionado y borra el progreso.
- El panel **Reproducción** permite elegir **Audio**, **Subtítulos** y **Calidad**. La elección se guarda para esa película y ese usuario. Si eliges una pista de audio distinta de la predeterminada en un MP4 que se reproduciría directo, Cinecat lo sirve por HLS para respetarla.
- El panel **Archivo** muestra los datos técnicos y cómo se reproducirá.
- Si eres administrador, al final aparecen **Cambiar imagen** y **Corregir ficha**.

## La ficha de un episodio

La ficha del episodio muestra la miga de pan `Series › Serie › Temporada N › Episodio N`, el título, la sinopsis, la duración, el reparto y los datos del archivo. Puedes marcar el episodio como visto, continuar donde lo dejaste o empezar desde el principio, y pasar al **Episodio anterior** o **Siguiente episodio**.

## La ficha de un vídeo

La ficha de un vídeo de la sección Vídeos es mínima: el título, **Volver a Vídeos**, reproducir o continuar, marcar como visto, las opciones de reproducción y el panel del archivo. No muestra sinopsis, reparto, colecciones, enlaces a IMDb/TMDB, «Más como este», «Corregir ficha» ni «Cambiar imagen», porque estos vídeos no se identifican.

## Reproducir

El reproductor tiene controles propios. Estos son los atajos de teclado:

| Tecla | Acción |
|---|---|
| `Espacio` o `K` | Reproducir o pausar |
| `←` o `J` | Retroceder 10 s (`Mayús`: 30 s) |
| `→` o `L` | Avanzar 10 s (`Mayús`: 30 s) |
| `↑` / `↓` | Subir o bajar el volumen |
| `M` | Silenciar o activar sonido |
| `F` | Pantalla completa |
| `C` | Activar o desactivar subtítulos |
| `,` / `.` | Fotograma anterior / siguiente, solo en pausa |
| `0`–`9` | Saltar al 0 %, 10 %, …, 90 % |
| `Esc` | Cerrar menús o cancelar la cuenta atrás |

Los botones visibles permiten reproducir/pausar, retroceder o avanzar 10 s, cambiar el volumen, elegir velocidad (0,5× a 2×), audio, subtítulos y calidad, activar imagen en imagen, pantalla completa y pasar al siguiente episodio. El clic en el vídeo reproduce o pausa; el doble clic alterna pantalla completa.

Si el archivo tiene segmentos, durante la intro aparece solo un botón **Saltar intro** sobre los controles y salta al final del segmento. En los créditos aparece **Siguiente episodio** cuando hay uno y lo abre; si no lo hay, aparece **Saltar créditos** y salta al final. El botón cambia solo al entrar o salir del segmento y no te quita el foco del teclado. Los administradores pueden revisar o marcar estos segmentos desde la ficha del ítem (ver [administración](administracion.md#intro-y-créditos)).

En el menú de subtítulos, las pistas de imagen incrustadas (PGS, VobSub y DVB) aparecen con la marca « · imagen». Al elegirlas, la reproducción se recarga en el segundo actual con el subtítulo quemado; al quitarlas o pasar a texto desde una de ellas, se recarga sin el quemado.

El progreso se guarda cada 15 s y al salir. Si quedan 15 s o menos, no se reanuda: se considera terminado. Al acabar un episodio con siguiente, aparece una cuenta atrás de 8 s para reproducirlo.

## Temas, menú y cuenta

El selector de tema tiene **Auto**, **Claro** y **Oscuro**. Auto sigue el tema del sistema. La preferencia se guarda en ese navegador; si no hay ninguna, se usa el tema por defecto del servidor.

En **Mi cuenta → Cambiar mi contraseña** puedes cambiar tu contraseña: pide la actual, la nueva y la repetición; la nueva debe tener al menos 8 caracteres. La sesión abierta en ese dispositivo sigue funcionando. En **Mi cuenta → Portal de inicio** eliges qué filas ocultar en tu portal. Como usuario solo ves las bibliotecas que el administrador te haya asignado.

En **Mi cuenta → Sesiones abiertas**, cada dispositivo de la lista tiene **Ajustes**. Allí puedes fijar su calidad máxima (sin límite, 480p, 720p o 1080p), su idioma de audio preferido (sin preferencia o un idioma) y sus subtítulos por defecto (sin preferencia, desactivados o un idioma). Se aplican al reproducir en ese dispositivo cuando no hayas elegido otra cosa para la película; el subtítulo del perfil nunca elige una pista de imagen y el tope de calidad solo afecta a HLS, no a la reproducción directa. El perfil pertenece a la sesión: al cerrarla o volver a vincular el dispositivo se pierde y vuelve a estar vacío.

## Glosario

- **Biblioteca:** una carpeta del servidor que Cinecat recorre para encontrar contenido.
- **Escaneo:** el proceso que busca archivos, analiza sus nombres y rellena fichas.
- **Ficha:** la página con los datos de una película, serie o episodio.
- **Identificar:** encontrar la ficha correcta en TMDB (o OMDb).
- **Transcodificación:** recodificar el vídeo o el audio para que el reproductor pueda entenderlo.
- **HLS:** formato de reproducción por segmentos que usa el reproductor web cuando el archivo original no se puede servir tal cual.
- **Reproducción directa:** enviar el archivo original sin recodificar.
- **WebVTT:** formato de subtítulos que el navegador puede mostrar.

## Si algo no funciona

- Si la página vuelve a «Entrar», la sesión ha caducado: inicia sesión otra vez.
- Si una película no aparece, comprueba con un administrador que la biblioteca existe y que el escaneo ha terminado.
- Si el vídeo no empieza, prueba otra calidad o avisa al administrador: el servidor puede no tener FFmpeg o el archivo puede usar un formato no admitido.
- Si ves el aviso de versiones distintas, recarga la página; si persiste, el administrador debe actualizar la web o el servidor.

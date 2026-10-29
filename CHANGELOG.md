# Registro de cambios

Todos los cambios relevantes de este proyecto se documentan aquí.
Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y versionado según [Versionado Semántico](https://semver.org/lang/es/) (`x.y.z`).

[English version](CHANGELOG.en.md)

## [Sin publicar]

### Añadido

- Servidor: las imágenes de TMDB se sirven desde el propio servidor (`/api/v1/images/tmdb/<tamaño>/<archivo>`) con una caché en disco en `cache/images`, y la API devuelve esas rutas relativas. Los carteles ya no dependen de que el DNS de cada aparato resuelva el CDN de TMDB, que algunos DNS bloquean.
- App Android: carga las imágenes desde el servidor con sus rutas relativas.
- App Android: reproductor con Media3 (ExoPlayer). Reproduce en directo lo que el aparato decodifica (detecta sus códecs) y, si falla, vuelve a pedir la reproducción por HLS. Reanuda donde se dejó, guarda el progreso cada 10 s, al pausar y al salir, permite elegir audio y subtítulos («Pistas») y pasa al siguiente episodio en las series.
- App Android en TV: control con el mando en el reproductor: centro pausa o reanuda, izquierda y derecha retroceden o avanzan 10 s, abajo muestra los controles y arriba o Menú abre «Pistas».

### Corregido

- App Android: el portal, los listados y las fichas no cargaban con un servidor real, porque los listados no envían el reparto ni los créditos y la app los exigía. Ahora todos los campos de la biblioteca son opcionales y un `null` de una lista se toma como lista vacía.
- App Android en TV: el texto de los botones principales salía oscuro sobre granate y casi no se leía.
- App Android: los títulos grandes (nombre de la app y títulos de pantalla en TV) no usaban Fraunces.
- App Android: los campos de dirección del servidor e IP local descartan los espacios que algunos teclados añaden tras cada punto.
- App Android en TV: la pantalla de vinculación no cabía en 1080p (el QR se cortaba) y el salvapantallas la tapaba antes de aprobar. Ahora va en dos columnas y mantiene la pantalla encendida.
- App Android en TV: con el mando no se podía salir de un campo de texto con las flechas arriba y abajo; ahora pasan al campo anterior o siguiente.
- App Android en TV: en las fichas de película y serie el mando solo llegaba a «Atrás»; ahora la cabecera toma el foco al entrar y abajo baja hasta Reproducir.

## [0.33.0] - 2026-10-29

### Añadido

- Biblioteca en la app Android para móvil y tableta: portal con las mismas filas y orden que la web, navegación inferior (riel lateral en tableta), películas y series en cuadrícula paginada con orden y género, colecciones, búsqueda de películas y series, y fichas de película, serie (temporadas y episodios con progreso) y colección con favorito, «por ver» y visto. Los carteles se cargan de TMDB a tamaño de pantalla y se guardan en caché. Aún no se reproduce; la interfaz propia de la TV llegará más adelante.

## [0.32.0] - 2026-10-29

### Añadido

- La app Android y Android TV ya se conecta con el servidor: pantalla para la dirección del servidor (solo HTTPS, con comprobación de versión y compatibilidad), inicio de sesión con usuario y contraseña en el móvil y vinculación por código con QR en la TV (también disponible en el móvil). La sesión del dispositivo se guarda cifrada con el Android Keystore y la app no entra en las copias de seguridad de Android. Tras iniciar sesión hay una pantalla de inicio provisional; la biblioteca y la reproducción llegarán en las próximas versiones.
- «IP local del servidor» en la app: si se indica (IPv4 privada o IPv6 local) y el equipo está en Wi-Fi o Ethernet, la app conecta con el servidor por la red local usando el mismo dominio y su certificado. Es el equivalente de editar `hosts` en el PC, que una TV no permite. Detalles en [docs/es/android.md](docs/es/android.md#ip-local-del-servidor).

## [0.31.0] - 2026-10-29

### Añadido

- Esqueleto de la app para Android, tableta y Android TV en `android/` (Kotlin + Jetpack Compose): proyecto Gradle con sus módulos, tema «Marquesina» y una pantalla de inicio de prueba distinta para móvil y para TV. Aún no se conecta al servidor ni tiene funciones; la versión de la app se toma del archivo `VERSION`. Documentación en [docs/es/android.md](docs/es/android.md), dependencias y licencias en `android/DEPENDENCIES.md` y `CREDITS.md`, y un flujo de GitHub Actions que compila la app cuando cambia `android/`.

## [0.30.0] - 2026-10-29

### Añadido

- Contrato de la API estable para clientes nativos: `GET /api/v1/system/version` devuelve ahora `api_level` (hoy 1) y `min_client` (versión mínima de app admitida). El OpenAPI incluye ya las 109 rutas (se documentaron usuarios, papelera, copias de seguridad, subidas, personas, borrar ítem y `/healthz`) y una prueba nueva (`internal/apicontract`) falla si una ruta del servidor no está en el OpenAPI o al revés.
- Documentación de la API para quien desarrolle clientes: [docs/es/api.md](docs/es/api.md) (estabilidad, autenticación, vinculación por código, reproducción con `caps`, errores y un ejemplo con `curl`).

## [0.29.0] - 2026-10-28

### Añadido

- Capacidades de las apps nativas en la decisión de reproducción: con los tokens nuevos de `caps` (contenedores `mkv`/`mp4`, vídeo `h264`, `h264hi10`, `hevc`, `hevc10`, `av1`, `vp9`, audio `aac`, `mp3`, `ac3`, `eac3`, `dts`, `truehd`, `opus`, `flac` y `mch`), un MKV o MP4 se sirve en directo si el equipo reproduce su vídeo y la pista de audio elegida; si no, HLS copiando el HEVC. `/play` devuelve `reason` (`container`, `video`, `audio`). La web no envía esos tokens y su comportamiento no cambia (lo comprueba una prueba que recorre su matriz).

## [0.28.0] - 2026-10-28

### Añadido

- Portal configurable: en **Administración → Ajustes → Interfaz → Filas del portal** se elige el orden de las filas de la página de inicio, cuáles se muestran y su título y antetítulo (ajuste `ui.portal_rows`). La tira de categorías sigue fija arriba.
- Fila nueva **«Vistas hace poco»** con las últimas películas marcadas como vistas (`recently_watched` en `GET /api/v1/home`).

### Cambiado

- Orden por defecto del portal: películas recién añadidas, continuar viendo películas, películas por ver, vistas hace poco, series recién añadidas, continuar viendo series, siguiente episodio, series por ver y favoritos. «Próximas a ver» se separa en «Películas por ver» y «Series por ver»; algunas filas cambian de nombre («Películas recién añadidas», «Series recién añadidas»).

## [0.27.0] - 2026-10-28

### Añadido

- Vincular un dispositivo por código: una app de Android o Android TV pide un código (`POST /api/v1/auth/pair`), lo muestra como `XXXX-XXXX` con un QR, y el usuario lo aprueba en la web en la nueva página **/vincular** (enlace también en Tu cuenta) tras ver el nombre del dispositivo y un aviso contra engaños; el dispositivo recoge entonces su sesión de 90 días (`POST /api/v1/auth/pair/token`). El código caduca a los 10 minutos y es de un solo uso; el token solo se entrega a quien tiene el secreto del dispositivo, que nunca se muestra; código y secreto se guardan solo como hash, con límites de intentos y de sondeo. Migración 0022.
- El login vuelve a la página de la que venías (`?next=`), solo si es una ruta del propio sitio.

## [0.26.0] - 2026-10-28

### Añadido

- Sesiones de dispositivo: cada sesión guarda el nombre del dispositivo (la web lo deduce del navegador y del sistema, p. ej. «Chrome · Windows») y su tipo (`web`, `android` o `androidtv`). En **Tu cuenta** hay una lista de «Sesiones abiertas» con la última actividad y un botón para cerrar cada una, y en **Administración → Usuarios** cada tarjeta muestra y cierra las sesiones de ese usuario. Las sesiones de los clientes nativos durarán 90 días (renovables); las de la web siguen en 24 h. API: `GET/DELETE /api/v1/auth/sessions` y, para administradores, `GET/DELETE /api/v1/users/{id}/sessions`; el login acepta `device_name` y `client`. Migración 0021. Es la base de las apps Android y Android TV.

## [0.25.0] - 2026-10-28

### Añadido

- Copia de seguridad diaria de la base de datos: el servidor crea una copia consistente (`VACUUM INTO`) cada 24 horas en `/data/backups`, junto a la base, y conserva 7 días (ajustable de 1 a 365; la más reciente se conserva siempre). Ajustes `backup.enabled` y `backup.retention_days`. En Administración → Ajustes → Sistema hay un panel con la lista de copias y el botón «Hacer copia ahora» (API `GET/POST /api/v1/system/backups`, solo administradores, sin exponer rutas del disco). Las copias están en el mismo volumen que la base: no protegen de perder el disco.

## [0.24.1] - 2026-10-28

### Corregido

- «Continuar viendo» del portal ya no mezcla películas y episodios en una sola fila: ahora hay dos, «Continuar viendo películas» y «Continuar viendo series».

## [0.24.0] - 2026-10-28

### Seguridad

- Acceso por biblioteca: cada usuario que no es administrador solo ve las bibliotecas que se le asignan (o todas, incluidas las futuras, si tiene esa opción). Se aplica en todos los listados, la búsqueda, el portal, series, colecciones, personas, música, las listas del usuario y en cada ítem, su reproducción, miniatura, progreso, visto y preferencias. Lo oculto responde como si no existiera (404) y no se puede escribir en ello. Las colecciones y las personas solo cuentan lo visible. Una prueba automática de fugas recorre todas las rutas de lectura con una biblioteca oculta marcada y falla si alguna respuesta la delata o si aparece una ruta nueva sin cubrir. Limitación conocida: una URL de reproducción ya emitida sigue valiendo hasta que caduca (8 h).
- Gestión de usuarios en el servidor (solo administradores): `GET/POST /api/v1/users`, `PATCH /api/v1/users/{id}` (rol, desactivar, acceso a todas las bibliotecas), `PUT /api/v1/users/{id}/libraries`, `POST /api/v1/users/{id}/password` y `DELETE /api/v1/users/{id}`. Restablecer la contraseña, desactivar, borrar o cambiar el rol cierra todas las sesiones de esa cuenta; una cuenta desactivada no puede entrar ni usar una sesión anterior; nadie puede borrarse, desactivarse ni quitarse el rol a sí mismo, y siempre queda al menos un administrador activo. Migración 0020 (`disabled`, `all_libraries`, `user_libraries`). La pantalla y el filtrado del contenido por biblioteca llegan en las siguientes versiones.
- Cambiar la contraseña cierra las demás sesiones abiertas de ese usuario (la actual se mantiene).
- Las peticiones JSON de la API están limitadas a 1 MiB (respuesta 413 si se supera); la subida de archivos no tiene ese límite.
- Las rutas de los archivos y de las carpetas de las bibliotecas (en fichas, listados, bibliotecas y la ficha técnica del archivo) solo las reciben los administradores; el resto de usuarios ya no ve cómo está organizado el disco del servidor.
- Cabeceras de seguridad en la web y en la API: la web ya no se puede incrustar en otra página (`X-Frame-Options: DENY` y `frame-ancestors 'none'`), `Strict-Transport-Security` por HTTPS, `X-Content-Type-Options: nosniff`, política de referencia y de permisos. La CSP es solo la mínima; una completa queda para más adelante. Recomendación de `ServerTokens Prod` en Apache (ver despliegue).
- El servidor se compila con Go 1.26 (antes 1.22.12, que ya no recibe parches y arrastraba 43 vulnerabilidades conocidas de la biblioteca estándar: `net/http`, `crypto/tls`, `crypto/x509`, `mime/multipart`…).

### Añadido

- Administración → Usuarios: el administrador crea cuentas (rol «Usuario» por defecto), elige las bibliotecas que cada una ve (o «todas, también las futuras»), restablece contraseñas, desactiva o activa cuentas, cambia el rol y borra con confirmación. Tu propia fila no permite desactivarte ni borrarte. Un usuario que no es administrador solo ve en el menú las secciones para las que tiene biblioteca, y si no tiene ninguna, el portal se lo explica.
- Aceleración por hardware (Intel VA-API) en la recodificación: cuando se activa `docker-compose.gpu.yml`, la gráfica integrada codifica el vídeo a H.264 y, si no hay que escalar, también lo decodifica (H.264 y H.265 de 8 bits), con un uso de CPU mucho menor. Si no hay gráfica, el ajuste «Aceleración por hardware» está en «Desactivada» o algo falla, se recodifica con `libx264` como antes. La ficha del archivo indica «Recodificación (gráfica)». La imagen incluye el controlador `intel-media-va-driver`. Es opcional: sin `COMPOSE_FILE=docker-compose.yml:docker-compose.gpu.yml` en el `.env` no cambia nada.
- Modo copia en la reproducción: si el navegador admite el códec (H.264 de 8 bits, o H.265 cuando el navegador lo reproduce por MSE), los MKV y MP4 se reempaquetan a HLS copiando el vídeo sin recodificarlo; solo se convierte el audio a AAC. Casi no consume CPU y no pierde calidad. Los segmentos se cortan en los fotogramas clave reales, que se leen del índice interno de los MKV en milisegundos o, en el resto, con ffprobe en segundo plano la primera vez (se guardan en `cache/keyframes`); mientras tanto, y ante cualquier fallo, se recodifica como antes. Los AVI y TS siempre se recodifican. Las copias no cuentan para «Transcodificaciones simultáneas». La ficha del archivo indica el método: directo, copia o recodificación.
- La web declara si el navegador reproduce H.265 por MSE (`hevcmse`, `hevc10mse` en la cookie `cinecat_caps`).

### Cambiado

- El vhost de Apache de ejemplo pasa a llamarse `deploy/media.techmarc.es.conf` y la documentación usa `media.techmarc.es` como dominio. Al cambiar de dominio hay que volver a iniciar sesión (la cookie es del dominio); la documentación lo explica.
- Los listados (películas, series, portal, listas, similares) ya no incluyen el reparto completo de cada título, que las tarjetas no usan: una página de 48 películas pasa de unos 320 KB a una fracción, y carga más rápido en móvil y en conexiones lentas. La ficha sigue mostrando el reparto completo.
- «Subir película»: «Crear una carpeta para cada película» viene desmarcado por defecto.

### Corregido

- «Cerrar sesión» está ahora en el menú (y en Mi cuenta); antes solo existía dentro de la página de inicio de sesión y no se encontraba. Mi cuenta explica ya qué pasa con las sesiones al cambiar la contraseña: se cierran las de otros dispositivos y se mantiene la actual.
- Las reproducciones en curso ya no se cortan al actualizar o reiniciar el servidor: la clave con la que se firman las URLs de reproducción se guarda junto a la base de datos (`signing.key`, permisos 0600) en lugar de generarse nueva en cada arranque.
- Un MP4 con H.265 etiquetado `hev1` (la mayoría de los descargados) ya no se envía en directo a un navegador que no lo acepta (Safari y iOS solo aceptan `hvc1`): la web comprueba también esa etiqueta y, si no, se reproduce por copia o recodificación.
- El listado de colecciones es mucho más rápido (el recuento usa `JOIN`).
- Las subidas largas ya no fallan con `fetch failed`: la web reenviaba el archivo con un `fetch` que corta a los 300 s sin respuesta, y el servidor solo responde al terminar de recibirlo. Ahora se reenvía en streaming sin límite de tiempo y directo al servidor por la red de Docker (`INTERNAL_API_URL`, ya en `docker-compose.yml`), sin salir a internet ni pasar otra vez por Apache.
- Un MP4 cuya pista de audio por defecto es AAC o MP3 con más de dos canales (p. ej. AAC 5.1) ya no se reproduce en directo, sino por HLS con el audio bajado a estéreo: Opera no sacaba sonido de AAC multicanal en directo.

## [0.22.1] - 2026-10-28

### Corregido

- El indicador de colección de las carátulas ahora muestra el nombre de la colección (con «+N» si hay varias) y su tooltip funciona: antes el icono bloqueaba el ratón y no se veía de qué colección era.

## [0.22.0] - 2026-10-27

### Añadido

- Página de persona y tráileres en la web: al pulsar un actor, director o miembro del equipo se abre su página (`/people/{id}`) con su ficha, lo que tienes en la biblioteca y, aparte, lo que ha hecho y no tienes. Botón «Ver tráiler» (YouTube, en castellano si existe; si no, en inglés) en las fichas de película y serie.
- Personas y tráileres en el servidor: los créditos guardan el id de TMDB de cada persona, `GET /api/v1/people/{id}` devuelve su ficha con lo que hay en la biblioteca y lo que no (por separado), y películas y series traen `trailer_url` (YouTube, en castellano si existe; si no, en inglés). La lista de series incluye la puntuación. Migración 0019: el siguiente escaneo vuelve a pedir a TMDB las fichas para rellenarlo (tardará más de lo normal). La pantalla llega en la siguiente versión.
- Puntuación de TMDB en una esquina de las carátulas de películas y series (por ejemplo «7,4»), solo cuando existe.
- «Subir película» en Películas (solo administradores): sube una o varias películas, con sus subtítulos, a la carpeta de películas sin tener que elegirla; por defecto crea una carpeta por película.
- Subidas guiadas de series en la web (solo administradores): botón «Crear nueva serie» en Series, «Nueva temporada» en la ficha de la serie y «Subir capítulo» en cada temporada. La web detecta temporada y episodio en el nombre del archivo (`S01E05`, `1x05`, «Capítulo 5»…) y deja corregirlos antes de subir, con barra de progreso por archivo.
- Subidas guiadas en el servidor: `GET /api/v1/uploads/targets` indica las carpetas de películas y series de cada biblioteca, y `PUT /api/v1/series/{id}/files?season=&episode=&name=` sube un capítulo eligiendo el servidor la carpeta (la de la temporada si existe, «Temporada N» si la serie usa carpetas por temporada) y el nombre (`<serie> S01E05.ext`, conservando el idioma de los subtítulos). La pantalla llega en la siguiente versión.
- Colecciones: icono discreto en la carátula de las películas que están en una colección, filtro «Colección: En una / Sin colección» en Películas y, en la ficha, «Gestionar colecciones» (solo administradores) para añadir la película a una colección existente, crear una nueva con ella o quitarla; quitarla de una colección de TMDB la excluye para que el escaneo no la vuelva a añadir. `GET /api/v1/collections?all=1` (administradores) lista también las colecciones ocultas o vacías y `GET /api/v1/items` admite `collection=any|none`.

### Corregido

- Reproducción según el códec real y lo que admite el navegador, no según la extensión: un MP4 en H.265 (o en H.264 de 10 bits, o con audio AC-3) ya no se envía directo a un navegador que no puede reproducirlo; se convierte. La web detecta una vez si el navegador admite H.265 y lo recuerda en la cookie `cinecat_caps`.
- La carpeta `.cinecat-papelera` se crea con permisos 0775, como el resto de la biblioteca, en lugar de 0755 por la umask del contenedor.

## [0.21.0] - 2026-10-27

### Añadido

- Subir archivos desde la web: Administración → Bibliotecas → «Subir archivos» en las bibliotecas de vídeo y clips (carpeta de destino con el selector de carpetas, carpeta nueva opcional, varios archivos con barra de progreso y cancelación). La web reenvía cada archivo en streaming al servidor.
- Papelera en la web: botón «Mover a la papelera» en la ficha (solo administradores, con confirmación) y página Administración → Papelera para restaurar, eliminar definitivamente o vaciar lo caducado. Nueva sección Administración → Biblioteca con «Permitir cambiar archivos» y «Días en la papelera».

- Papelera de vídeos en el servidor (solo administradores y con el ajuste «Permitir cambiar archivos»): `DELETE /api/v1/items/{id}` mueve la película, el episodio o el clip (con sus subtítulos, o su carpeta si no contiene otros vídeos) a `.cinecat-papelera` en el mismo disco, sin borrar nada; `GET /api/v1/trash`, `POST /api/v1/trash/{id}/restore` (devuelve el archivo y la ficha con visto, progreso, listas, colecciones y preferencias), `DELETE /api/v1/trash/{id}` y `POST /api/v1/trash/purge`. Purga diaria según «Días en la papelera» (14 por defecto). Migración 0018.
- Subida de archivos en el servidor: `PUT /api/v1/libraries/{id}/files?dir=…&name=…[&subdir=…]` guarda en streaming un vídeo o subtítulo en una carpeta de una biblioteca de vídeo (nunca sobrescribe; escribe en un `.part` y lo renombra al terminar), con límite «Tamaño máximo de subida (GB)» (50 por defecto), comprobación de espacio libre y escaneo posterior. `CINECAT_FILE_UID`/`CINECAT_FILE_GID` fijan el propietario de lo que crea el servidor.

### Cambiado

- Despliegue: el montaje de `/media` depende de `CINECAT_MEDIA_MODE` (`ro` por defecto, `rw` para subir y usar la papelera); nuevas variables `CINECAT_FILE_UID`/`CINECAT_FILE_GID`, y el contenedor web acepta cuerpos grandes (`BODY_SIZE_LIMIT=Infinity`). Documentado en Administración → «Subir archivos y papelera».
- Administración → Sistema: «Actualizar ahora» abre ahora un cuadro de confirmación (con «Cancelar» y «Actualizar ahora») en lugar de un desplegable con un segundo botón. Sin JavaScript se conserva el respaldo de dos pasos.

## [0.20.0] - 2026-10-27

### Añadido

- Favoritos y «Próximas a ver» en la web: botones de corazón y marcador en las fichas de película, vídeo, serie y colección y en las tarjetas (funcionan también sin JavaScript), páginas «Favoritos» y «Próximas a ver» con secciones y paginación, entradas en el menú (con sesión) y filas «Próximas a ver» y «Tus favoritos» en el portal.
- Favoritos y «Próximas a ver» en el servidor: cada usuario marca y desmarca películas, vídeos, series y colecciones (`PUT`/`DELETE /api/v1/{items|series|collections}/{id}/lists/{list}`) y los lista con `GET /api/v1/lists/{list}`. Migración 0017. Los DTO traen `favorite` y `watchlist`, y el portal `watchlist` y `favorites`. Al marcar algo como visto sale solo de «Próximas a ver» (y la serie, cuando queda completamente vista). La pantalla llega en la siguiente versión.

## [0.19.1] - 2026-10-27

### Seguridad

- El bloqueo de login por intentos fallidos ya no permite bloquear la cuenta de otra persona: se cuenta por pareja usuario + IP del cliente, con un límite aparte de 20 fallos por IP. La web pasa la IP real a la API (`X-Forwarded-For`, con `ADDRESS_HEADER` y `XFF_DEPTH=1` en el contenedor web) y la API solo se fía de esa cabecera si la conexión directa viene de un proxy de confianza (`CINECAT_TRUSTED_PROXIES`, por defecto loopback y redes privadas). El bloqueo responde 429 con `Retry-After`, no revela si el usuario existe y el mapa en memoria tiene tope.

## [0.19.0] - 2026-10-27

### Añadido

- Sección «Música» en la web: álbumes (con carátula, orden y filtro por letra), artistas y búsqueda de pistas, página de álbum y de artista, entrada en el menú y categoría en el portal. El tipo «Música» se puede elegir al crear una biblioteca.
- Reproductor de música persistente: una barra fija inferior que sigue sonando mientras se navega, con cola (añadir, reproducir a continuación, quitar, vaciar), aleatorio, repetir, volumen, teclas multimedia y metadatos del sistema (Media Session). Se pausa al reproducir un vídeo y la cola se recuerda entre visitas (se restaura en pausa).
- Bibliotecas de tipo «Música» en el servidor: el escaneo lee las etiquetas con `ffprobe` (artista, álbum, pista, disco, año, género), sin servicios externos, solo reinspecciona los archivos nuevos o cambiados, y conserva las fichas si una carpeta no está disponible. Migración 0016 (`music_artists`, `music_albums`, `music_tracks`). Carátulas de la carpeta o incrustadas (en caché `cache/covers`).
- API de música: `GET /api/v1/music/artists`, `/artists/{id}`, `/albums`, `/albums/{id}`, `/albums/{id}/cover`, `/tracks` (búsqueda) y `/tracks/{id}/play`, y flujo firmado con rangos `GET /api/v1/stream/music/{id}`. `GET /api/v1/home` añade `has_music`.

### Cambiado

- Las URL firmadas de reproducción llevan ahora un espacio de nombres (`video` o `music`) para que un id de pista no valide como película ni al revés.

## [0.18.0] - 2026-10-27

### Añadido

- Miniaturas de los vídeos con un fotograma del propio archivo, generadas con FFmpeg la primera vez que se ven y guardadas en caché (`cache/thumbs`); cada carpeta usa la de uno de sus vídeos como portada (`cover_item_id`). Si no hay FFmpeg o falla, la tarjeta conserva su degradado. Endpoint `GET /api/v1/items/{id}/thumbnail`.
- Bibliotecas de tipo «Vídeos» (`clips`), pensadas para videoclips y conciertos: se escanean sin identificación ni conexiones externas (el título sale del nombre del archivo), no se mezclan con películas, búsqueda, colecciones ni contadores, y se navegan por carpetas con `GET /api/v1/libraries/{id}/browse`. El ítem expone `library_kind` y el portal `has_clips`.
- Sección «Vídeos» en la web: entrada en el menú (si hay vídeos), navegación por carpetas con migas de pan y paginación, tarjetas con progreso y marca de visto, ficha mínima (sin reparto ni corrección de ficha) y categoría en el portal. El tipo «Vídeos» se puede elegir al crear una biblioteca.

### Corregido

- Reproducción, información del archivo y miniaturas fallaban («archivo no encontrado») para los archivos de las carpetas secundarias de una biblioteca con varias carpetas, porque se comprobaba solo contra la carpeta principal.

## [0.17.0] - 2026-10-27

### Añadido

- Administración → Bibliotecas: pantalla renovada con tarjetas por biblioteca, varias carpetas por biblioteca (añadir, quitar con confirmación), renombrar, escanear y borrar, y un navegador de carpetas del servidor para elegirlas. Los tipos «Vídeos» y «Música» aparecen como «Próximamente».
- Bibliotecas con varias carpetas (por ejemplo, en discos distintos) y con tipo (`video`, `clips`, `music`): `POST /api/v1/libraries` acepta `paths`, y nuevos `PATCH /api/v1/libraries/{id}`, `POST /api/v1/libraries/{id}/paths` y `DELETE /api/v1/libraries/{id}/paths/{pathId}`. Migración 0015 (`library_paths`, `libraries.kind`); la biblioteca existente se migra con su carpeta. Los tipos `clips` y `music` se aceptan pero aún no se escanean.
- Navegador de carpetas del servidor `GET /api/v1/fs` (solo administradores), confinado a las raíces de `CINECAT_MEDIA_ROOTS` (por defecto `/media`).
- Portal: fila «Series añadidas recientemente» (`recent_series` en `GET /api/v1/home`) con enlace a la lista de series ordenada por las últimas añadidas. La fila de películas pasa a titularse «Películas añadidas recientemente».

### Cambiado

- El escaneo solo poda fichas de las carpetas que recorrió con éxito: si una carpeta no está disponible o aparece vacía teniendo fichas (disco sin montar), conserva las fichas con su visto y progreso y lo avisa.

## [0.16.3] - 2026-10-27

### Cambiado

- Documentación puesta al día en castellano e inglés: README reescrito, nuevas páginas de estado (qué hay hecho y qué falta), guía de usuario y administración (con la tabla completa de ajustes), y revisión de arquitectura, reproducción, metadatos, despliegue, hoja de ruta y TODO contra el código.

## [0.16.2] - 2026-10-27

### Corregido

- El instalador del actualizador detecta que el gestor `systemd --user` arrancó antes de que el usuario entrase en el grupo `docker` (el servicio no podría usar Docker) y explica cómo corregirlo; el actualizador da ese mismo mensaje en lugar de «No se pudo consultar el directorio de Docker» a secas.

## [0.16.1] - 2026-10-27

### Corregido

- El instalador del actualizador define las variables del bus de usuario de systemd cuando faltan (entrada con `su` o sin sesión de login) y avisa de que la instalación está incompleta si no puede activar las unidades, en lugar de decir que terminó bien. Además verifica antes los requisitos (programas, acceso a Docker y que `git` llegue a `origin` sin pedir contraseña) y la documentación incluye una guía para servidor nuevo o cambio de dominio.

## [0.16.0] - 2026-10-27

### Añadido

- Administración → Sistema: «Buscar actualizaciones» consulta al servidor y muestra las notas de la versión nueva; «Actualizar ahora» (con confirmación) pide la actualización al servicio del host y sigue su progreso por fases, incluido el reinicio, hasta «Actualizado a X» o el aviso de que se volvió a la versión anterior.
- Servicio actualizador para el host (`scripts/cinecat-updater.sh`, unidades de systemd de usuario e `scripts/instalar-actualizador.sh`): recoge la petición del servidor, hace copia de la base de datos, actualiza a la etiqueta pedida, comprueba que arranca y, si falla, vuelve al código y, si ya se habían reiniciado los contenedores, también a la base de datos anteriores. Documentado en `docs/es/actualizaciones.md`.

### Cambiado

- La comprobación de actualizaciones de la web ya no lee `VERSION` de GitHub ni pide ejecutar un comando a mano: usa la del servidor.

## [0.15.0] - 2026-10-27

### Cambiado

- Reparto y equipo: primero los actores y después dirección, guion, producción, etc.
- «Más como este» pasa al final de la ficha (tras temporadas y episodios en series).
- Menú lateral más estrecho y pegado al borde, contenido a más ancho y con menos margen; tarjetas de la administración alineadas y de igual tamaño.
- Los ajustes «Columnas de los listados» (`ui.grid_columns`) y «Filas por página» (`ui.grid_rows`) sustituyen a `ui.page_size`: el número de resultados por página es columnas × filas. Nuevo `ui.row_columns` para las carátulas visibles por fila en el portal (6 por defecto).

### Añadido

- Carátulas de temporada en el selector y la cabecera de cada temporada de la ficha de serie (con la de la serie como respaldo); el siguiente escaneo las rellena sin reconsultar episodios.
- Tarjetas de episodio del portal con fotograma (o cartel de la serie) y el nombre de la serie.

## [0.14.0] - 2026-10-26

### Añadido

- Comprobación de actualizaciones: el servidor consulta las etiquetas `vX.Y.Z` de GitHub (opt-in con `system.check_updates`, o a petición) y ofrece a los administradores `GET /api/v1/system/update`, `POST …/check` y `POST …/apply`; esta última deja la orden en un directorio compartido con el host (`CINECAT_RUN_DIR`). El servicio del host que la ejecuta llega en una versión posterior.
- Filas con flechas ‹ › sin barra de desplazamiento; el portal ya no desborda horizontalmente (categorías en rejilla adaptable).
- Reparto y equipo con fotos (retrato de respaldo con iniciales) y fila «Más como este» en las fichas de película y de serie.
- Menú de navegación lateral a la izquierda (ajuste `ui.nav_layout`: lateral o superior); la web aplica ahora los ajustes públicos `ui.*` (nombre, tema por defecto, acento, mostrar colecciones y series).
- Eslogan, título e introducción del portal editables desde la administración (`ui.tagline`, `ui.portal_title`, `ui.portal_intro`).

### Cambiado

- Aspecto más claro (crema casi blanco, sin velo beige); fuera la referencia a «cine de barrio»: nuevo eslogan «Sesión continua» y título «Tu cine en casa».

## [0.13.1] - 2026-10-26

### Corregido

- La web no compilaba en el contenedor: la navegación entre episodios se exportaba desde `+page.server.ts`, algo que SvelteKit no admite. Pasa a `$lib/episode-navigation.ts`.

## [0.13.0] - 2026-10-26

### Añadido
- Reproductor con controles propios: reproducir/pausa, ±10 s, barra de progreso con el tiempo al pasar el ratón, volumen y silencio, velocidad de 0,5× a 2×, imagen en imagen, pantalla completa y «Siguiente episodio» con cuenta atrás. Atajos de teclado (Espacio/K, J/L, flechas, M, F, C, 0-9). Sin JavaScript conserva los controles nativos.
- Menús de pista de audio, subtítulos y calidad en el reproductor (cambiar audio o calidad recarga en el segundo actual) y selectores de Audio, Subtítulos y Calidad en la ficha, con la preferencia guardada por usuario y por película.
- Reproducción con pista de audio y calidad a elegir (`GET /api/v1/items/{id}/play?audio=&quality=&subtitle=`): la respuesta lista las pistas de audio, las calidades disponibles (sin subir de la resolución original) y los subtítulos. Subtítulos incrustados y archivos `.srt`/`.vtt`/`.ass` junto al vídeo, servidos como WebVTT. Preferencias de audio, subtítulos y calidad guardadas por usuario y por película (`/api/v1/items/{id}/preferences`).

## [0.12.1] - 2026-10-26

### Corregido
- El reparto con fotos y personajes, las colecciones automáticas y los créditos de las series se rellenan de verdad: la caché de TMDB guardaba solo los campos que el programa conocía al crearla, y los campos añadidos después (foto, personaje, colección) faltaban. Ahora guarda la respuesta original completa, las entradas antiguas se descartan y el siguiente escaneo rehace las fichas.
- El relleno de los créditos de las series ya identificadas se ejecuta aunque sus temporadas estén completas.

## [0.12.0] - 2026-10-26

### Corregido
- El vídeo ya no se cuelga al avanzar tras varias reproducciones: FFmpeg se detiene a los 90 s sin peticiones, no se adelanta más de 20 segmentos al punto pedido, hay un tope de transcodificaciones simultáneas (ajuste `playback.max_transcodes`, 2 por defecto) y los procesos van con prioridad baja y hilos limitados. Antes cada reproducción abandonada seguía transcodificando la película entera.

### Añadido
- Reparto y equipo con foto y personaje (`credits` en películas y series; las fichas ya guardadas se rellenan desde la caché de TMDB) y «Más como este» calculado en local: `GET /api/v1/items/{id}/similar` y `GET /api/v1/series/{id}/similar`.
- Panel de administración en la web con menú lateral: Resumen (versiones, último escaneo, recuentos y pendientes de identificar), Bibliotecas, y formularios de ajustes por sección (Escaneo, Metadatos, Interfaz, Reproducción, Sistema) con valores por defecto, validación por campo y «Restablecer». Comprobación opcional de actualizaciones, solo si se activa y se pulsa el botón.
- Los ajustes se aplican en caliente, sin reiniciar: intervalo y arranque del escaneo automático, idioma de TMDB, umbral de identificación, calidad de la transcodificación (altura máxima, CRF y preset) y mínimo de películas por colección automática.
- Ajustes del servidor guardados en la base de datos con registro tipado y validación (escaneo, metadatos, interfaz, reproducción y sistema): `GET`/`PATCH /api/v1/settings`, `DELETE /api/v1/settings/{clave}` y `GET /api/v1/settings/public`. Las variables de entorno siguen dando el valor inicial.

## [0.11.0] - 2026-10-26

### Cambiado
- TMDB pasa a ser la fuente principal de fichas (en castellano); OMDb queda como respaldo si no hay clave de TMDB.
- La cartelera pasa de `/` a `/movies`; `/` es ahora el portal.

### Corregido
- «Volver a automático» reanaliza el nombre del archivo desde la raíz de su biblioteca, no desde la ruta absoluta.

### Añadido
- El escaneo de bibliotecas se ejecuta en segundo plano: `POST /api/v1/libraries/scan` responde 202 (o 409 si ya hay uno en marcha) y `GET /api/v1/libraries/scan` informa del estado y del último resultado.
- Escaneo automático un minuto después de arrancar y después cada `CINECAT_SCAN_INTERVAL` (por defecto `6h`; `0` lo desactiva).
- El panel de bibliotecas lanza el escaneo en segundo plano y muestra «Escaneo en curso…» con sondeo cada 3 s; al acabar, archivos, identificadas, sin identificar y avisos plegables.
- `GET /api/v1/system/version` (sin sesión) con la versión, el commit y la fecha de compilación; el pie de la web muestra la versión del servidor y avisa si la web y el servidor tienen versiones distintas. La versión de la web sale del `VERSION` de la raíz (la imagen de la web se construye ahora desde la raíz del repositorio).
- Ficha completa de película en la API (título en castellano, título original, lema, géneros, duración, nota, dirección, reparto, fondo, estado de la identificación) y, en `GET /api/v1/items`, los filtros `kind`, `q`, `unmatched` y la ordenación `sort`. Corrección manual para administradores: `GET /items/{id}/candidates`, `PUT` y `DELETE /items/{id}/match`.
- El escaneo rellena el título en castellano de las fichas guardadas antes de que existiera.
- Búsqueda por título, reparto, dirección o todo (`field`), filtro por inicial (`letter`, ignorando artículos y tildes: «El Padrino» va en la P) y por género (`genre`), lista de géneros (`GET /api/v1/genres`) y ordenación por título normalizado.
- Estado de visto por usuario: `PUT`/`DELETE /api/v1/items/{id}/watched`, campo `watched` en los ítems y filtro `watched`.
- Portal inicial en `/` con una entrada por categoría (Películas; Colecciones y Series «Próximamente») y la fila «Añadido recientemente». La cartelera pasa a `/movies`, paginada (48 por página), con búsqueda por título, reparto o dirección, filtro por letra y por género, orden, visto/sin ver y, para administradores, «Sin identificar». Marca «Visto» en los carteles.
- `GET /api/v1/items/{id}/file-info`: información técnica del archivo con ffprobe (contenedor, tamaño, pistas de vídeo, audio y subtítulos con sus códecs, HDR, calidad) y cómo se reproducirá.
- Imágenes personalizables (administración): `GET /api/v1/items/{id}/images` ofrece los carteles y fondos de TMDB; `PUT`/`DELETE /api/v1/items/{id}/images/{poster|backdrop}` fijan una URL propia o restauran la de TMDB. El escaneo no pisa las imágenes fijadas.
- Ficha de película completa (título original, lema, duración, géneros, nota, dirección, reparto, enlaces a IMDb y TMDB, fondo), botón de visto/no visto y, para administradores, «Corregir ficha» con búsqueda de candidatos en TMDB, «No es ninguna» y «Volver a automático».
- Progreso de reproducción por usuario (`PUT`/`DELETE /api/v1/items/{id}/progress`; al 90 % se marca como visto) y `GET /api/v1/home` con «continuar viendo», lo añadido recientemente y los contadores del portal.
- Panel «Archivo» en la ficha (calidad, contenedor, tamaño, pistas de vídeo, audio y subtítulos, cómo se reproducirá) y, para administradores, panel «Imagen» para elegir el cartel y el fondo entre los de TMDB o fijar una URL propia.
- El reproductor guarda el progreso cada 15 s y al salir, y permite continuar donde se dejó («Continuar · 1 h 12 min» en la ficha, «Empezar desde el principio»). El portal muestra «Continuar viendo» y «Añadido recientemente».
- Colecciones automáticas: las películas que pertenecen a una saga de TMDB se agrupan solas en una colección; una colección borrada o una película quitada por el usuario no reaparecen en los escaneos siguientes. Las películas ya identificadas se añaden a su colección en el primer escaneo.
- API de colecciones: listado paginado y detalle, y gestión para administradores (crear, editar, borrar, añadir y quitar películas, reordenar). Las películas indican a qué colecciones pertenecen y se pueden filtrar por `collection_id`.
- Colecciones en la web: categoría activa en el portal y la cabecera, listado paginado con búsqueda y orden, página de cada colección con sus películas y, para administradores, gestión (crear, editar, cambiar cartel y fondo, añadir y quitar películas, reordenar, borrar). La ficha de película enlaza a sus colecciones.
- El analizador de nombres toma la serie de la carpeta de primer nivel y reconoce episodios con formatos como `ChicagoFire720p4x17` o `MR_Robot_305` dentro de una carpeta de temporada (antes se tomaban por películas).
- Series: cada carpeta de serie se agrupa en una ficha propia, se identifica con TMDB en castellano (mismas reglas que las películas: automática, manual o «no es ninguna») y el escaneo rellena temporadas y episodios (título, sinopsis, imagen) con una petición por temporada. El estado del escaneo incluye `series_identified` y `series_unidentified`.
- API de series: listado paginado con búsqueda, letra y género, ficha con temporadas y siguiente episodio, episodios por temporada, marcar serie o temporada como vista, corrección de la identificación e imágenes para administradores. `GET /api/v1/home` rellena «siguiente episodio».
- Series en la web: categoría activa en el portal y la cabecera, listado paginado con búsqueda, letra y género, y ficha de serie con selector de temporadas, episodios con título, sinopsis e imagen, «Continuar» y marcar episodio, temporada o serie como vistos. El portal muestra «Siguiente episodio».
- Corregir la ficha de una serie (administradores): buscar candidatos en TMDB, «Es esta», «No es ninguna» y «Volver a automático», y cambiar su cartel y fondo.
- Ficha de episodio con migas de pan (serie › temporada › episodio), imagen y sinopsis del episodio y navegación al anterior y al siguiente.
- El servidor se apaga de forma ordenada ante SIGINT/SIGTERM, cancelando el escaneo en curso.

## [0.10.0] - 2026-10-26

### Corregido
- Las películas con audio AC-3 5.1 (casi toda la biblioteca real) se quedaban cargando: el audio se convertía a AAC 5.1, que los reproductores del navegador no aceptan. Ahora se baja siempre a AAC estéreo.
- Saltar a un punto avanzado ya no deja el vídeo «pensando»: si el segmento pedido queda lejos de lo que FFmpeg ha generado (o es un hueco hacia atrás), el servidor detiene FFmpeg y lo relanza en ese segmento (`-ss` + `-start_number`, el mismo criterio que Jellyfin). Un salto tarda menos de un segundo en vídeo SD, en vez de esperar hasta 5 minutos y fallar.
- La reproducción ya no se corta a los 15 minutos: los segmentos HLS y los saltos en reproducción directa reutilizan la firma de la URL inicial, que caducaba a los 15 minutos. La vida por defecto de la URL pasa a 8 horas.
- El último segmento de la playlist ya no puede ser un resto de milisegundos que FFmpeg no llega a generar (fallaba al final del vídeo).
- El reproductor web usa hls.js siempre que el navegador lo admite, en vez del HLS nativo que anuncian Chrome en Android y en escritorio; reintenta ante errores de red o de descodificación en lugar de rendirse al primero, y espera hasta 60 s al primer byte de un segmento.

### Cambiado
- HLS usa segmentos MPEG-TS (`segment_XXXXX.ts`) en vez de fMP4 (`init.mp4` + `.m4s`): cada segmento lleva sus marcas de tiempo absolutas, así que los generados por procesos FFmpeg distintos encajan en la misma línea de tiempo. Las marcas de salida llevan un desplazamiento fijo de 10 s para que el proceso que empieza en 0 no tenga DTS negativos (que FFmpeg corregía desplazando unos fotogramas).
- Todo lo que se sirve por HLS se recodifica a H.264 (libx264 `veryfast`, CRF 20, hasta 1080p y dimensiones pares) + AAC estéreo. Se elimina el remux sin recodificar: la biblioteca real es casi toda DivX/MPEG-4 con AC-3 o MP3, que el navegador no reproduce. FFmpeg ya no arranca al pedir la playlist, sino con la primera petición de segmento, en el punto pedido.
- La API sigue anunciando `mode: "remux"` para HLS.

## [0.9.1] - 2026-10-26

### Corregido
- La playlist HLS que se sirve ya es un VOD completo, no un evento en directo: Cinecat calcula la duración con `ffprobe` y escribe él mismo `playlist.m3u8` antes de arrancar FFmpeg, con todos los segmentos esperados, `EXT-X-PLAYLIST-TYPE:VOD` y `EXT-X-ENDLIST` desde la primera petición. Así hls.js conoce la duración total y permite avanzar/retroceder. FFmpeg escribe su propia playlist aparte (`ffmpeg.m3u8`) para no pisarla, y los segmentos se siguen sirviendo en cuanto existen. Si no se puede leer la duración (algunos AVI antiguos), se mantiene el fallback a la playlist `event` de FFmpeg.

## [0.9.0] - 2026-10-26

### Corregido
- La generación HLS ya no bloquea hasta procesar el vídeo completo: `Service.prepare()` arranca FFmpeg en segundo plano y devuelve en cuanto existe la playlist inicial, y los segmentos se sirven en cuanto FFmpeg los genera (con espera acotada y comprobación de que el segmento está completo). Para que la playlist exista desde el primer segmento se usa `-hls_playlist_type event` en vez de `vod` (con `vod` FFmpeg solo la escribía al terminar); al acabar se añade `ENDLIST` y sigue siendo un VOD normal. El proceso queda ligado al ciclo de vida del servicio, no a la petición HTTP que lo lanza, y dos peticiones simultáneas del mismo ítem reutilizan el mismo proceso. Antes, una película de una hora podía tardar más de 25 minutos en empezar.

## [0.8.1] - 2026-10-26

### Corregido
- La 0.8.0 no arreglaba todos los casos reales: un `.avi` de producción con 2804 keyframes bien repartidos (por debajo del umbral de la detección) seguía generando un único segmento HLS con la película entera, porque `-c:v copy` necesita timestamps de paquete fiables para segmentar, y en AVI/DivX muchos paquetes llevan `pts_time` como `N/A`. El intervalo entre keyframes puede ser correcto y el contenedor no ser segmentable igualmente. Ahora cualquier `.avi` se transcodifica siempre, sin pasar por el análisis de keyframes, que se mantiene para el resto de contenedores (`.mkv`, `.mov`, `.ts`, `.m2ts`).

## [0.8.0] - 2026-10-26

### Añadido
- Transcodificación de vídeo por software cuando el intervalo entre keyframes impide un remux HLS razonable: en vez de fallar, se recodifica solo el vídeo con libx264 `veryfast`/CRF 20, audio AAC 160k y keyframes forzados cada 6 s, manteniendo el resto del flujo HLS. Esto resuelve la reproducción de `.avi`/DivX con keyframes dispersos como el caso real de producción; no implementa todavía el motor completo de perfiles ni aceleración por hardware, y la API sigue anunciando `mode: "remux"` (el cliente recibe HLS en ambos casos).

## [0.7.3] - 2026-10-26

### Corregido
- El remux HLS comprueba con `ffprobe`, antes de lanzar FFmpeg, que el intervalo máximo entre keyframes sea viable (≤ 15 s). En vídeos con keyframes muy separados (típico en `.avi`/DivX antiguos) devuelve un error claro en vez de generar un único segmento gigante que el navegador no puede empezar a reproducir. **No repara la reproducción de esos archivos**: la transcodificación real queda para el siguiente paso de la Fase 3.

## [0.7.2] - 2026-10-26

### Añadido
- `GET /api/v1/items` acepta `library_id` opcional para filtrar por biblioteca (necesario de cara al cliente web, con 1652 ítems reales en producción la lista plana sin filtro ni paginación visible resultaba inmanejable).

## [0.7.1] - 2026-10-26

### Corregido
- `docker-compose.yml` no pasaba `CINECAT_OMDB_API_KEY` al contenedor del servidor (se añadió el soporte en el código en la 0.6.0, pero se olvidó en el compose): aunque la clave estuviera en `.env`, el servidor nunca la veía y el escaneo seguía sin metadatos. Añadida también a `.env.deploy.example`.

## [0.7.0] - 2026-10-26

### Añadido
- La biblioteca muestra el cartel real de OMDb cuando existe y mantiene el cartel tipográfico de respaldo cuando falta. La ficha de ítem muestra también el cartel real y añade la sinopsis (`overview`) cuando está disponible, omitiendo la sección si no hay datos.

## [0.6.0] - 2026-10-25

### Añadido
- Metadatos de OMDb para películas durante el escaneo (`CINECAT_OMDB_API_KEY`): sinopsis y URL del cartel, con caché en disco de las búsquedas, sin repetir consultas para ítems que ya tienen datos ni para series/episodios. Se exponen como `overview` y `poster_url` en `GET /api/v1/items` y `GET /api/v1/items/{id}`. Los fallos o ausencias de OMDb quedan como avisos del escaneo.

## [0.5.0] - 2026-10-25

### Añadido
- Panel `/admin/libraries` para usuarios con rol admin: lista las bibliotecas, permite crear una nueva (nombre y ruta), borrarla con confirmación y lanzar el escaneo completo mostrando bibliotecas, archivos, entradas actualizadas y avisos. Enlace «Bibliotecas» en la cabecera solo para administradores, con el rol obtenido de `GET /api/v1/auth/me`; los usuarios sin rol reciben un 403 claro.

## [0.4.1] - 2026-10-25

### Añadido
- `GET /api/v1/auth/me`: devuelve `{username, role}` del usuario de la sesión Bearer actual, con 401 si falta el token o no es válido.

## [0.4.0] - 2026-10-25

### Añadido
- Pantalla «Mi cuenta» en el cliente web para cambiar la contraseña: formulario con contraseña actual, nueva y repetición, validación local de coincidencia y longitud mínima (8 caracteres), mensajes claros para errores de la API y confirmación de éxito sin cerrar la sesión actual. Enlace «Mi cuenta» en la cabecera solo cuando hay sesión.

## [0.3.5] - 2026-10-25

### Añadido
- `POST /api/v1/auth/password`: permite al usuario autenticado cambiar su propia contraseña comprobando la actual (401 si no coincide, 400 si la nueva tiene menos de 8 caracteres) y sin invalidar la sesión ni el resto de sesiones vigentes.

## [0.3.4] - 2026-10-25

### Cambiado
- `docs/es/despliegue.md`/`docs/en/deployment.md` y `.env.deploy.example` explican cómo dar de alta varias bibliotecas (p. ej. películas y series) cuando `CINECAT_MEDIA_PATH` apunta a un disco o carpeta que las contiene a todas como subcarpetas, en vez de asumir una sola biblioteca en la raíz montada.

## [0.3.3] - 2026-10-25

### Añadido
- Flag `cinecatd --hash-password <contraseña>`: genera el hash argon2id en formato PHC y lo imprime, para fijar manualmente `password_hash` en la base de datos (p. ej. con `sqlite3` dentro del contenedor) mientras no existe todavía un endpoint ni pantalla para cambiar la contraseña desde la interfaz.

## [0.3.2] - 2026-10-25

### Corregido
- El cliente web rechazaba toda petición a la API con «Falta configurar PUBLIC_API_URL en el servidor», incluido el login, cuando `PUBLIC_API_URL` se dejaba vacío a propósito para usar rutas relativas detrás del proxy del despliegue Docker (ver `0.3.1`). `createApiClient` (`web/src/lib/api-factory.ts`) ya no trata la cadena vacía como error de configuración; `web/src/routes/login/+page.server.ts` usa ahora esa misma función en vez de duplicar la lógica con su propio chequeo.

## [0.3.1] - 2026-10-25

### Cambiado
- El despliegue Docker ya no lleva Caddy como proxy propio: el Apache que ya gestiona otros vhosts del Debian hace de proxy inverso hacia el servidor y el cliente web (vhost de ejemplo en `deploy/video.warcrafted.com.conf`). `docker-compose.yml` publica ambos contenedores solo en `127.0.0.1` (puertos `CINECAT_API_PORT`/`CINECAT_WEB_PORT`, nunca expuestos directamente a la red). `.env.deploy.example` y `docs/es/despliegue.md`/`docs/en/deployment.md` actualizados con los dos pasos (contenedores + vhost de Apache).

## [0.3.0] - 2026-10-25

### Añadido
- Despliegue con Docker Compose: `Dockerfile` del servidor Go (con FFmpeg instalado en la imagen), `web/Dockerfile` del cliente web, `docker-compose.yml` con Caddy como proxy único delante de ambos (mismo origen, sin necesidad de CORS), script `scripts/actualizar.sh` para aplicar versiones nuevas sin perder datos, y guía de despliegue en `docs/es/despliegue.md`.
- Soporte de `CINECAT_ADMIN_PASSWORD` documentado en el flujo de despliegue para fijar la contraseña del administrador en el primer arranque sin depender del log.
- Versión real incrustada en el binario del servidor en el `Dockerfile` (antes mostraba «desconocida» en un binario compilado sin `-ldflags`).

### Cambiado
- `docker-compose.yml` pasa `PROTOCOL_HEADER=x-forwarded-proto` al contenedor del cliente web (variable que ya soporta `@sveltejs/adapter-node`), de forma que la cookie de sesión se marca `Secure` correctamente detrás de Caddy.

## [0.2.0] - 2026-10-25

Cierra la Fase 2: cliente web funcional con inicio de sesión, biblioteca, fichas y reproductor.

### Añadido
- Modo claro/oscuro/sistema conmutable y persistido en `localStorage`, con aplicación antes del primer pintado para evitar parpadeos. Paleta oscura «Marquesina» coherente con la clara, contraste de texto revisado (≥ 4.5:1), foco visible unificado para enlaces, botones, campos y vídeo, y conmutador accesible en todas las páginas.

## [0.2.0-dev.4] - 2026-10-25

### Añadido
- Reproductor web real en la ruta de reproducción: `<video>` nativo con controles para `mode: "direct"` y hls.js 1.7.3 (Apache-2.0) para `mode: "remux"`, con HLS nativo en Safari/iOS, mensaje claro si el navegador no soporta HLS, limpieza de `hls.destroy()` al salir y estilos de sala oscura alrededor del vídeo.

## [0.2.0-dev.3] - 2026-10-25

### Añadido
- Pantalla de biblioteca en la portada (grid de `PosterCard` con cartel de respaldo tipografiado y color estable derivado del título) y ficha de ítem con metadatos y botón «Reproducir» hacia una ruta de reproducción provisional que muestra el modo y la URL firmada. Las páginas protegidas redirigen a `/login` sin sesión.

## [0.2.0-dev.2] - 2026-10-25

### Añadido
- Cliente de API tipado y gestión de sesión del cliente web: login y logout mediante form actions de SvelteKit, token guardado en cookie httpOnly con SameSite=Lax y Secure bajo HTTPS, URL del servidor por `PUBLIC_API_URL` y página de login «Marquesina» con mensajes para credenciales incorrectas y bloqueo temporal.

## [0.2.0-dev.1] - 2026-10-25

### Añadido
- Esqueleto del cliente web en `web/`: SvelteKit + TypeScript con `@sveltejs/adapter-node`, tipografías Fraunces e Inter empaquetadas localmente (OFL-1.1) y dirección visual «Marquesina» en la portada. La URL base del servidor se configura con `PUBLIC_API_URL` (ver `web/.env.example`).

## [0.1.3] - 2026-10-25

### Añadido
- `POST /api/v1/libraries/scan`, restringido a administradores: dispara un escaneo síncrono de todas las bibliotecas y devuelve el número de bibliotecas recorridas, archivos encontrados y entradas actualizadas, junto con los avisos de rutas ilegibles.

## [0.1.2] - 2026-10-25

### Añadido
- `POST /api/v1/libraries` y `DELETE /api/v1/libraries/{id}`, restringidos a usuarios con rol admin. Valida que la ruta exista y sea un directorio; 409 si ya hay una biblioteca con esa ruta; el borrado limpia en cascada sus `media_items`.

### Cambiado
- `requireSession` ahora permite exigir también un rol concreto (`requireAdmin`), sin cambiar el comportamiento de los endpoints existentes.

### Añadido
- Alta automática del primer usuario administrador al arrancar el servidor contra una base de datos sin usuarios: contraseña por `CINECAT_ADMIN_PASSWORD` o generada aleatoriamente (144 bits) y mostrada una sola vez en el log.

## [0.1.0] - 2026-10-25

Cierra la Fase 1: servidor mínimo funcional (sin clientes todavía).

### Añadido
- Esqueleto del servidor Go (`cinecatd`): módulo, configuración por archivo/entorno, flag `--version` y endpoint `GET /healthz` (Fase 1, paso 1).
- Esquema de base de datos SQLite (`users`, `sessions`, `libraries`, `media_items`, `playback_state`) y migraciones embebidas e idempotentes (Fase 1, paso 2). Driver: `modernc.org/sqlite` (BSD-3-Clause).
- Autenticación con argon2id, sesiones de 24h renovables (caducidad deslizante) y bloqueo temporal tras 5 intentos fallidos; endpoints `POST /api/v1/auth/login` y `POST /api/v1/auth/logout` (Fase 1, paso 3). Dependencia: `golang.org/x/crypto` (BSD-3-Clause).
- Escaneo de biblioteca y analizador propio de nombres de archivo y carpeta (título, año, temporada y episodio), con upsert en `media_items` sin duplicar en re-escaneos (Fase 1, paso 4). Columnas `season`/`episode` añadidas al esquema.
- Cliente de metadatos TMDB (es-ES): búsqueda de películas, selección de cartel por prioridad de idioma (es-ES → español genérico → inglés → sin idioma) y caché en disco con TTL configurable para búsquedas e imágenes (Fase 1, paso 5). Sin conectar todavía al escaneo ni a los endpoints; solo busca películas, series quedan para más adelante.
- Contrato OpenAPI (`api/openapi.yaml`) y endpoints `GET /api/v1/libraries`, `GET /api/v1/items` (con paginación) y `GET /api/v1/items/{id}`, protegidos con sesión Bearer (Fase 1, paso 6).
- Reproducción directa (`http.ServeContent`, con rangos HTTP) para `.mp4`/`.m4v` y remux a HLS fMP4 sin reencodar (copia de códecs) para `.mkv`/`.avi`/`.mov`/`.ts`/`.m2ts`, invocando FFmpeg como proceso externo con argumentos siempre en lista. URLs de reproducción firmadas con HMAC-SHA256 (clave de proceso, nunca la de sesión) y caducidad de 15 minutos. Endpoints `GET /api/v1/items/{id}/play`, `GET /api/v1/stream/{id}` y `GET /api/v1/stream/{id}/{file}` (Fase 1, paso 7).

### Nota de verificación
- El entorno de desarrollo usado para esta fase no tiene FFmpeg instalado: los tests que ejercitan el remux real (`TestReproduccionDirectaMP4`, `TestRemuxMKVGeneraHLS`) se han saltado con `t.Skip()` en todas las revisiones de esta fase y **nunca se han ejecutado de verdad**. El resto de pruebas de reproducción (firma, caducidad, decisión de modo, ausencia de shell en el código) sí se han ejecutado y pasan. Antes de confiar en el remux en producción, ejecutar la suite completa en una máquina con FFmpeg instalado.

## [0.0.5] - 2026-10-25

### Añadido
- Pila tecnológica de la Fase 0 completada: web en **TypeScript + SvelteKit** + hls.js; Android/Android TV en **Kotlin + Jetpack Compose** + Media3/ExoPlayer (libmpv opcional).

## [0.0.4] - 2026-10-24

### Añadido
- `CLAUDE.md` del proyecto con las decisiones ya tomadas, la tabla de licencias de terceros y qué no hacer.
- Comprobación del nombre «Cinecat»: sin marca registrada ni repositorio en conflicto; se documenta como riesgo aceptado la existencia de una app Android inactiva desde 2015 con el mismo nombre.

## [0.0.3] - 2026-10-24

### Añadido
- **Licencia decidida: GPL-3.0**, con la condición adicional del artículo 7 que exige conservar los avisos de autoría de Catlinux. Archivo `LICENSE` en la raíz.
- Tabla de licencias verificadas (en el repositorio oficial de cada proyecto) de los candidatos a reutilización: Jellyfin y clientes (GPL-2.0), Findroid e Intro Skipper (GPL-3.0), Streamyfin (MPL-2.0), Media3/hls.js/Shaka (Apache-2.0), Vidstack (MIT).
- Pila tecnológica del servidor decidida: **Go**.

### Cambiado
- README, `docs/es` y `docs/en` (licencias-y-créditos, visión, arquitectura, hoja de ruta) y `TODO` actualizados para reflejar la licencia y la pila del servidor ya decididas.

## [0.0.2] - 2026-10-24

### Añadido
- Dirección visual elegida para la interfaz: «Marquesina» (cine de barrio de los 50-60, serif Fraunces, carteles ilustrados propios), con enlace a la maqueta de referencia.
- Archivo `VERSION`.

## [0.0.1] - 2026-10-24

### Añadido
- Documentación inicial bilingüe (castellano e inglés): visión, arquitectura, reproducción y códecs, metadatos, licencias y créditos, y hoja de ruta.
- `README`, `TODO`, `CONTRIBUTING`, `SECURITY`, `AUTHORS` y `CREDITS`.
- Decisiones de partida: servidor y clientes propios; web, Android y Android TV; H.265 preferido con H.264 de reserva; metadatos en español de España a partir de bases de datos existentes.

[Sin publicar]: https://github.com/catlinux/cinecat/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/catlinux/cinecat/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/catlinux/cinecat/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/catlinux/cinecat/compare/v0.0.5...v0.1.0
[0.0.5]: https://github.com/catlinux/cinecat/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/catlinux/cinecat/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/catlinux/cinecat/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/catlinux/cinecat/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/catlinux/cinecat/releases/tag/v0.0.1

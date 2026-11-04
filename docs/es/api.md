# API para clientes nativos

[English](../en/api.md) · [Volver al README](../../README.md) · [Arquitectura y seguridad](arquitectura.md)

Cinecat expone una API HTTP para que las apps Android y Android TV puedan autenticarse, consultar la biblioteca y reproducir contenido sin depender del cliente web. Las apps nativas usan este contrato. Esta página resume sus reglas y las operaciones principales; el detalle exacto de cada operación está en el OpenAPI.

## El contrato

El contrato es [api/openapi.yaml](../../api/openapi.yaml), un documento OpenAPI 3.0.3 en castellano. Incluye todas las rutas de la API registradas por el servidor Go, con sus métodos, parámetros, cuerpos, respuestas y requisitos de sesión.

La API vive bajo `/api/v1/`. `GET /healthz` es la única ruta de salud fuera de ese prefijo: no exige sesión y devuelve `ok` en texto plano. La prueba `internal/apicontract` compara las rutas registradas en el código Go con las documentadas en el OpenAPI; si una aparece en un lado y no en el otro, la suite falla y muestra la diferencia.

## Estabilidad

El contrato de `/api/v1/` es estable para las apps:

- Dentro de `v1` solo se añaden rutas y campos.
- Los clientes deben ignorar los campos que no conozcan.
- No se eliminan campos ni se les cambia el significado.
- Un cambio incompatible exigiría una nueva versión, `/api/v2`.

`GET /api/v1/system/version` no exige sesión y devuelve:

| Campo | Significado |
|---|---|
| `server` | Versión del servidor. |
| `commit` | Commit de la compilación; puede estar vacío. |
| `built_at` | Fecha de compilación en RFC 3339; puede estar vacía. |
| `api_level` | Nivel del contrato de `/api/v1`; hoy vale `1`. |
| `min_client` | Versión mínima de app nativa que este servidor admite; hoy vale `0.0.0`. |

La app debe comparar su propia versión con `min_client` y avisar al usuario si es menor. `api_level` solo sube cuando cambia el contrato de forma incompatible.

## Actualización de la app Android

`GET /api/v1/system/client-update?platform=android&version_code=N` permite a la app Android comprobar si la versión del servidor tiene un APK publicado más nuevo. Requiere una sesión válida, pero no rol de administrador.

Si hay una actualización, responde `200` con `version`, `version_code`, `url`, `sha256` y `notes`. La URL apunta al APK de la release del servidor y `sha256` es su suma SHA-256 en minúsculas; `notes` contiene la sección del CHANGELOG si está disponible en la caché del comprobador.

Responde `204` si la comprobación está desactivada, si `platform` no es `android`, si la versión del servidor no es válida, si el cliente ya está al día o si no se pudo obtener la suma. `version_code` es obligatorio, debe ser un entero no negativo y su ausencia o formato inválido devuelve `400`.

## Autenticación

`POST /api/v1/auth/login` recibe las credenciales y los datos del dispositivo:

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "ana",
  "password": "…",
  "device_name": "TV del salón",
  "client": "androidtv"
}
```

`client` admite `web`, `android` y `androidtv`; si se omite, se trata como `web`. La respuesta contiene `token`, `expires_at` y `session_id`.

Salvo el login, `POST /api/v1/auth/pair`, `POST /api/v1/auth/pair/token`, `/healthz`, `/api/v1/system/version` y las URLs firmadas de reproducción, las peticiones llevan la cabecera:

```http
Authorization: Bearer <token>
```

Las sesiones web duran 24 horas y las de `android`/`androidtv`, 90 días. Se renuevan al usarlas cuando les queda menos de la mitad de su vida. Si una petición devuelve `401`, la sesión no es válida o ha caducado: hay que volver a iniciar sesión.

La sesión actual y las demás sesiones propias se consultan con `GET /api/v1/auth/sessions`; `DELETE /api/v1/auth/sessions/{id}` cierra una sesión concreta. Cada elemento de la lista incluye `profile`, el perfil de ese dispositivo. Los administradores pueden consultar y revocar las sesiones de una cuenta con las rutas `/api/v1/users/{id}/sessions`.

El login limita los intentos fallidos por usuario e IP. Al superar el límite responde `429` con la cabecera `Retry-After`; la app debe esperar ese tiempo antes de reintentar.

## Perfil del dispositivo

Cada sesión activa tiene un perfil propio que se aplica al reproducir. Se lee con `GET /api/v1/auth/sessions/{id}/profile` y se sustituye entero con `PUT` sobre la misma ruta. `{id}` admite un identificador numérico o `current` para la sesión que hace la petición; solo se pueden consultar y editar sesiones del propio usuario, y una sesión ajena, caducada o inexistente responde `404`. El cuerpo y la respuesta tienen los tres campos de `SessionProfile`:

- `max_quality`: `""` (sin límite), `480`, `720` o `1080`.
- `audio_language`: `""` (sin preferencia) o un código de 2 o 3 letras en minúsculas.
- `subtitle_language`: `""` (sin preferencia), `off` (desactivados) o un código de 2 o 3 letras en minúsculas.

Un valor no válido responde `400`. El perfil pertenece a la sesión: al cerrarla o volver a vincular el dispositivo se pierde y vuelve a estar vacío.

## Vincular un dispositivo

Para una TV o un dispositivo sin teclado cómodo:

1. La app llama a `POST /api/v1/auth/pair` con `device_name` y `client` (`android` o `androidtv`). La respuesta incluye `pair_id`, `secret`, `code`, `verification_path`, `expires_in` e `interval`.
2. La app muestra el código `XXXX-XXXX` y un QR que apunta a `<servidor>/vincular?codigo=XXXX-XXXX`. El `secret` no se muestra en pantalla ni se incluye en el QR.
3. El usuario abre la URL, inicia sesión en la web e introduce o aprueba el código. La web usa `POST /api/v1/auth/pair/approve`: con `confirm: false` obtiene una vista previa del dispositivo y con `confirm: true` confirma el acceso.
4. La app consulta `POST /api/v1/auth/pair/token` con `pair_id` y `secret` cada 5 segundos. El sondeo demasiado rápido se rechaza antes de 4 segundos. La respuesta final contiene `token`, `expires_at`, `session_id` y `user` con `username` y `role`.

Estados del sondeo:

| Código | Significado |
|---|---|
| `200` | Vinculación aprobada; se entrega la sesión. |
| `428` | Todavía pendiente de aprobación; esperar `Retry-After`. |
| `429` | Sondeo demasiado rápido o demasiados intentos. |
| `410` | Vinculación caducada o ya usada. |
| `404` | Identificador o secreto no válidos. |

## Secciones disponibles

`GET /api/v1/sections` indica qué secciones tienen contenido para el usuario de la sesión:

```json
{"movies": true, "series": false, "collections": true}
```

`movies` es verdadero si hay alguna película accesible, `series` si hay alguna serie accesible y `collections` si hay alguna colección visible con al menos una película accesible. Los administradores reciben siempre los tres valores en verdadero. La web y las apps usan esta respuesta para ocultar las secciones vacías; si la llamada falla o un servidor antiguo responde `404`, deben mostrar todas.

## Fotos

`GET /api/v1/photos` devuelve las subcarpetas directas y las fotos de una biblioteca de tipo `photos`. Requiere `library_id` y acepta `folder` (ruta relativa a la raíz; vacía para la raíz), `sort` (`date` o `name`, por defecto `date`), `limit` (1–200; 50 por defecto) y `offset` (0 por defecto). `date` ordena por `taken_at` y, si falta, por la fecha de modificación. Solo se puede consultar una biblioteca de fotos a la que el usuario tenga acceso; si no existe o no es visible, responde `404`.

La respuesta tiene la forma:

```json
{
  "folders": ["2024", "Viajes"],
  "items": [
    {"id": 17, "name": "IMG_0017.jpg", "width": 4032, "height": 3024, "taken_at": 1600000000, "size": 2456789}
  ],
  "total": 42
}
```

`folders` son los nombres de las subcarpetas directas de `folder`; `items` son las fotos de esa carpeta, paginadas; `total` es el número total de fotos de esa carpeta. `taken_at` es la fecha de captura EXIF en segundos Unix o `null` si no existe. Se indexan JPEG, PNG, WebP y GIF; HEIC y RAW no se admiten.

`GET /api/v1/photos/{id}/thumb` devuelve una miniatura JPEG de 400 px de lado mayor, generada con FFmpeg y cacheada; admite `ETag` y responde `304` si no ha cambiado. `GET /api/v1/photos/{id}/original` sirve el original con su `Content-Type` y soporte de rangos HTTP (`206`). Ambas requieren sesión y acceso a la biblioteca; si la foto no existe o queda fuera de su biblioteca, responden `404`.

## Actividad (solo administradores)

`GET /api/v1/activity` devuelve las últimas reproducciones de todos los usuarios. Combina el progreso y las marcas de visto en una entrada por usuario e ítem, con la fecha más reciente, e incluye usuario, título, serie, temporada, episodio, posición, duración y estado de visto. Acepta `limit` (máximo 200; por defecto 50) y `user_id` para filtrar. Requiere rol `admin`.

## Seguridad (solo administradores)

`GET /api/v1/system/security-events` devuelve los 50 bloqueos de inicio de sesión más recientes y el total de los últimos 7 días (`lockouts` y `total_7d`). No incluye contraseñas ni tokens. Requiere rol `admin`.

## Historial y portal

`GET /api/v1/history?limit=N` devuelve el historial del usuario de la sesión, de más reciente a más antiguo. `limit` vale 50 por defecto y se recorta a 200. Cada entrada incluye `item_id`, `title`, `kind`, datos de serie y episodio, `poster_url`, posición, duración, `watched` y `updated_at`, y solo aparece contenido de bibliotecas accesibles. `DELETE /api/v1/history/{itemId}` borra el progreso y la marca de visto de ese usuario para el ítem y responde `204`; si el ítem no existe o no es accesible, `404`.

`GET /api/v1/home/preferences` devuelve todas las filas posibles del portal con `id`, `title` y `hidden`. `PUT /api/v1/home/preferences` recibe `{"hidden":["id",…]}` y sustituye el conjunto de filas ocultas del usuario; las ausentes vuelven a mostrarse y un identificador desconocido devuelve `400`. `GET /api/v1/home` aplica la preferencia.

## Reproducción

`GET /api/v1/items/{id}/play` prepara la reproducción y acepta estos parámetros de consulta:

| Parámetro | Uso |
|---|---|
| `caps` | Capacidades del dispositivo, separadas por comas. |
| `audio` | Índice de la pista de audio elegida. |
| `quality` | `auto`, `original`, `480`, `720` o `1080`. |
| `subtitle` | Identificador de subtítulo; vacío para desactivarlo. Si es una pista de imagen incrustada (`e<N>`), pasa a HLS y se quema en el vídeo. |
| `download` | `1` para preparar una descarga sin conexión; ver «Descarga para ver sin conexión». |

Las apps declaran en `caps` lo que su equipo reproduce: contenedores `mkv` y `mp4`; vídeo `h264`, `h264hi10`, `hevc`, `hevc10`, `av1` y `vp9`; audio `aac`, `mp3`, `ac3`, `eac3`, `dts`, `truehd`, `opus` y `flac`; y `mch` si decodifica AAC multicanal. Cada token indica que el equipo puede decodificar ese contenedor, códec o formato; `hevc10` implica `hevc`, y `mch` solo hace falta para AAC de más de dos canales. Si una app nativa declara `mch` y la pista elegida tiene 6 o más canales, el HLS entrega AAC 5.1; en los demás casos entrega estéreo. Para que el servidor trate la petición como app nativa, `caps` debe incluir al menos `mkv` o `mp4`. La lista completa y su efecto están en [reproducción](reproduccion.md#apps-nativas-android-y-android-tv).

La respuesta incluye `url`, `mode`, `expires_at`, `audio_tracks`, `selected_audio`, `qualities`, `selected_quality`, `subtitles`, `selected_subtitle`, `segments`, `autoplay_next` y `next_episode`. `mode` es `direct` o `remux`; si no es directo, `reason` explica el motivo (`container`, `video`, `audio` o `subtitle`).

Cada elemento de `subtitles` es un `SubtitleOption`: `burn_in` indica si es una imagen incrustada (PGS, VobSub o DVB); en ese caso `url` va vacía y el cliente no debe cargarla como WebVTT. El resto conserva su URL WebVTT firmada.

La URL de `/stream` es relativa al servidor, está firmada y vale 8 horas. No necesita `Authorization`: antepón la base del servidor y pásala tal cual al reproductor, por ejemplo ExoPlayer. No añadas cabeceras propias ni modifiques la firma.

Si se elige una pista de imagen con `subtitle=e<N>`, `mode` es `remux`, `reason` es `subtitle` y las URLs HLS firmadas de `/stream` incluyen `burnsub=<N>` (la posición de la pista en el archivo, el mismo `N` de `e<N>`). Tanto `/stream/{id}` como `/stream/{id}/{file}` aceptan `burnsub` y fuerzan HLS por CPU, nunca copia ni VA-API. El endpoint WebVTT de una pista de imagen (`/api/v1/stream/{id}/subtitles/<sid>.vtt`) responde `404`.

El progreso se guarda con `PUT /api/v1/items/{id}/progress` y un cuerpo como `{"position_seconds": 300, "duration_seconds": 1800}`; `DELETE` sobre la misma ruta lo borra. Al llegar al 90 % de la duración, el servidor marca el ítem como visto y elimina el progreso. El visionado se controla con `PUT` y `DELETE /api/v1/items/{id}/watched`.

Las preferencias de audio, subtítulos y calidad se leen con `GET /api/v1/items/{id}/preferences`, que devuelve `audio_index`, `subtitle_id` y `quality` (o `null` si no hay preferencia). Se guardan con `PUT` sobre la misma ruta y un cuerpo parcial: los campos ausentes no se tocan, `null` los borra y `subtitle_id` vacío significa «sin subtítulos». `DELETE` elimina todas las preferencias del usuario para ese ítem. Las apps las restauran al reabrir el elemento.

`GET /play` aplica esta precedencia: parámetro explícito > preferencia del ítem > perfil de la sesión > valor por defecto. El perfil de audio elige la primera pista de ese idioma; si no hay ninguna, se usa el comportamiento normal. El perfil de subtítulo elige la primera pista de texto (nunca una de imagen) de ese idioma, prefiriendo las no forzadas; `off` no muestra subtítulos por defecto. La calidad máxima solo actúa como tope en HLS: recorta las calidades ofrecidas y baja `auto`, `original` y las alturas superiores al límite; la reproducción directa no cambia.

## Reproducción automática del siguiente episodio

El ajuste es por usuario y serie y, si no hay fila guardada, está activado. Se lee con `GET /api/v1/series/{id}/autoplay` y se guarda con `PUT` sobre la misma ruta y un cuerpo `{"enabled": true}` o `{"enabled": false}`; ambas respuestas son `{"enabled": ...}`. Requieren sesión y una serie visible: una serie inexistente, oculta o que no sea serie responde `404`, y un cuerpo inválido `400`.

`GET /api/v1/series/{id}` incluye `autoplay_next` con ese ajuste. `GET /api/v1/items/{id}/play` también lo incluye: para un episodio refleja el de su serie y para cualquier otro ítem es `true`. Además, `next_episode` es `null` o `{id, title, season, episode}` con el episodio inmediatamente posterior visible, por temporada y número; si no hay otro o el ítem no es un episodio, es `null`. Un siguiente episodio oculto por los permisos de biblioteca no se revela.

## Descarga para ver sin conexión

`GET /api/v1/items/{id}/download` devuelve el archivo original de una película o un episodio. Requiere una sesión válida y acceso a la biblioteca del ítem; un ítem inexistente, oculto, de otro tipo o sin archivo responde `404`.

La respuesta usa `http.ServeContent`: admite rangos (`206` e `If-Range`), envía `Last-Modified`, `Accept-Ranges: bytes`, `Content-Disposition` con `filename*=UTF-8''...` y el `Content-Type` según la extensión (`.mkv` es `video/x-matroska`, `.mp4` y `.m4v` son `video/mp4`, el resto `application/octet-stream`). No recodifica ni modifica el archivo. Una sesión ausente o no válida responde `401`, un rango fuera del archivo `416` y un error interno `500`.

`GET /api/v1/items/{id}/play?download=1` prepara la descarga de la misma versión que se ve en streaming (Media3 offline en Android): fuerza `remux` aunque el archivo admitiera reproducción directa, con la pista de audio y la calidad que se elegirían al reproducir. La URL firmada vale `playback.download_url_ttl` horas (72 por defecto, en lugar de las 8 de la reproducción) y lleva `dl=1`, que se propaga a la playlist y a los segmentos. El servidor genera los segmentos bajo demanda y con prioridad baja (ver [reproducción](reproduccion.md)); si no hay cupo, el segmento responde `503` con `Retry-After: 30` y el cliente debe reintentar. Con una firma caducada, el cliente vuelve a pedir `/play?download=1` y sigue con la URL nueva. `download_direct` y `download_blocked` pasan a ser informativos.

La ruta `/download` anterior se conserva. La app Android ya no la usa; antes solo la usaba cuando `/play`, con las capacidades del equipo, devuelve `download_direct: true`: el contenedor da igual (`.mp4`, `.m4v`, `.mkv`, `.mov`) siempre que el dispositivo reproduzca el vídeo y el audio. Si es `false`, `download_blocked` dice por qué (`container`, `video`, `audio` o `client`) y no se descarga; no se generan versiones recodificadas.

## Segmentos de intro y créditos

`segments` es una lista, posiblemente vacía, de objetos `{kind, start, end, source}`. `kind` es `intro` o `credits`, `start` y `end` están en segundos y `source` es `chapter` o `manual`. Los segmentos manuales sustituyen a los detectados del mismo tipo. La detección usa los capítulos de la ficha técnica y los títulos normalizados (sin mayúsculas, acentos ni espacios sobrantes); también se pueden marcar a mano desde la ficha. No hay detección por audio. Si la ficha no se puede leer, la lista va vacía y la reproducción sigue igual.

Las rutas de edición requieren rol `admin`:

- `GET /api/v1/items/{id}/segments` devuelve `{"manual":[…],"chapters":[…]}`. Cada elemento tiene `kind`, `start` y `end`; `manual` son los guardados y `chapters` los detectados en el archivo.
- `PUT /api/v1/items/{id}/segments` recibe `{"segments":[{"kind":"intro","start":10,"end":85}]}` y sustituye todos los manuales del ítem. Una lista vacía los borra. Responde `204`; un valor no válido (tipo desconocido o repetido, `start` negativo o `end` no mayor que `start`) devuelve `400` con un mensaje en castellano.
- `POST /api/v1/items/{id}/segments/apply-season` copia los manuales del episodio a los demás episodios de la misma serie y temporada, sustituyendo los suyos, y devuelve `{"updated":n}`. Si el ítem no es un episodio, responde `400`.

Un ítem inexistente o no visible responde `404`; sin rol de administrador, `403`.

## Subtítulos subidos (solo administradores)

`POST /api/v1/items/{id}/subtitles` recibe `multipart/form-data` con `file` obligatorio y `language` y `label` opcionales. Admite `.srt`, `.vtt` y `.ass` de hasta 2 MB de texto UTF-8 o Latin-1/Windows-1252, y rechaza binarios. Responde `201` con `id` (`u<N>`), `language`, `label` y `codec`; el subtítulo aparece en `subtitles` de `/play` con `source: "uploaded"` y su URL WebVTT firmada. `DELETE /api/v1/items/{id}/subtitles/{subId}` acepta `u<N>` o el número de fila, borra la fila y su archivo y responde `204`. Los archivos viven en `subtitles/` junto a la base de datos.

## Imágenes

En las fichas, `poster_url` y `backdrop_url` son rutas relativas al servidor (`/api/v1/images/tmdb/...`), no URLs de `image.tmdb.org`. La imagen se sirve desde el propio servidor y requiere la sesión, igual que el resto de rutas autenticadas.

Las miniaturas de clips y las carátulas de música también las sirve el servidor y requieren la sesión:

- `GET /api/v1/items/{id}/thumbnail` devuelve la miniatura del clip.
- `GET /api/v1/music/albums/{id}/cover` devuelve la carátula del álbum.

Ambas respuestas son imágenes y admiten caché y `ETag`.

## Errores

Los errores de la API usan la forma:

```json
{"error": "…"}
```

Códigos habituales:

| Código | Significado |
|---|---|
| `400` | Parámetros o cuerpo no válidos. |
| `401` | Sesión ausente, no válida o caducada. |
| `403` | La cuenta no tiene el rol necesario. |
| `404` | El recurso no existe o no es visible para la cuenta. |
| `409` | Conflicto con el estado actual. |
| `429` | Demasiados intentos; respetar `Retry-After`. |
| `502` | Fallo de una dependencia externa, como TMDB. |
| `503` | FFmpeg u otro servicio necesario no está disponible. |

Un contenedor no admitido en la reproducción responde `415`.

## Acceso por biblioteca

El servidor ya filtra por las bibliotecas asignadas a la cuenta y por su rol. La app no necesita enviar identificadores de biblioteca para ocultar contenido: lo que no está permitido responde `404`, igual que un recurso inexistente. Esto se aplica también a la reproducción, la descarga y las miniaturas.

## Ejemplo con curl

Sustituye `https://cinecat.example`, el usuario y la contraseña por los de tu servidor.

```bash
BASE=https://cinecat.example

TOKEN=$(curl -sS "$BASE/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"ana","password":"…","device_name":"TV del salón","client":"androidtv"}' \
  | jq -r .token)

curl -sS "$BASE/api/v1/items?kind=movie&sort=title&limit=20" \
  -H "Authorization: Bearer $TOKEN"

curl -sS "$BASE/api/v1/items/42/play?caps=mkv,mp4,h264,aac,ac3" \
  -H "Authorization: Bearer $TOKEN"
```

La última respuesta contiene una `url` como `/api/v1/stream/42?exp=…&sig=…`. Pásala al reproductor con la base del servidor delante; el reproductor no debe enviar el `Bearer`.

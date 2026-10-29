# API para clientes nativos

[English](../en/api.md) · [Volver al README](../../README.md) · [Arquitectura y seguridad](arquitectura.md)

Cinecat expone una API HTTP para que las apps Android y Android TV puedan autenticarse, consultar la biblioteca y reproducir contenido sin depender del cliente web. Las apps nativas aún no están en el repositorio; este es el contrato que deberán usar cuando se implementen. Esta página resume el contrato y las reglas que deben respetar los clientes nativos; el detalle exacto de cada operación está en el OpenAPI.

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

La sesión actual y las demás sesiones propias se consultan con `GET /api/v1/auth/sessions`; `DELETE /api/v1/auth/sessions/{id}` cierra una sesión concreta. Los administradores pueden consultar y revocar las sesiones de una cuenta con las rutas `/api/v1/users/{id}/sessions`.

El login limita los intentos fallidos por usuario e IP. Al superar el límite responde `429` con la cabecera `Retry-After`; la app debe esperar ese tiempo antes de reintentar.

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

## Reproducción

`GET /api/v1/items/{id}/play` prepara la reproducción y acepta estos parámetros de consulta:

| Parámetro | Uso |
|---|---|
| `caps` | Capacidades del dispositivo, separadas por comas. |
| `audio` | Índice de la pista de audio elegida. |
| `quality` | `auto`, `original`, `480`, `720` o `1080`. |
| `subtitle` | Identificador de subtítulo; vacío para desactivarlo. |

Las apps declaran en `caps` lo que su equipo reproduce: contenedores `mkv` y `mp4`; vídeo `h264`, `h264hi10`, `hevc`, `hevc10`, `av1` y `vp9`; audio `aac`, `mp3`, `ac3`, `eac3`, `dts`, `truehd`, `opus` y `flac`; y `mch` si decodifica AAC multicanal. Cada token indica que el equipo puede decodificar ese contenedor, códec o formato; `hevc10` implica `hevc`, y `mch` solo hace falta para AAC de más de dos canales. Para que el servidor trate la petición como app nativa, `caps` debe incluir al menos `mkv` o `mp4`. La lista completa y su efecto están en [reproducción](reproduccion.md#apps-nativas-android-y-android-tv).

La respuesta incluye `url`, `mode`, `expires_at`, `audio_tracks`, `selected_audio`, `qualities`, `selected_quality`, `subtitles` y `selected_subtitle`. `mode` es `direct` o `remux`; si no es directo, `reason` explica el motivo (`container`, `video` o `audio`).

La URL de `/stream` es relativa al servidor, está firmada y vale 8 horas. No necesita `Authorization`: antepón la base del servidor y pásala tal cual al reproductor, por ejemplo ExoPlayer. No añadas cabeceras propias ni modifiques la firma.

El progreso se guarda con `PUT /api/v1/items/{id}/progress` y un cuerpo como `{"position_seconds": 300, "duration_seconds": 1800}`; `DELETE` sobre la misma ruta lo borra. Al llegar al 90 % de la duración, el servidor marca el ítem como visto y elimina el progreso. El visionado se controla con `PUT` y `DELETE /api/v1/items/{id}/watched`.

## Imágenes

En las fichas, `poster_url` y `backdrop_url` son URLs de TMDB que la app puede cargar directamente, sin pasar por la API.

Las miniaturas de clips y las carátulas de música sí las sirve el servidor y requieren la sesión:

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

El servidor ya filtra por las bibliotecas asignadas a la cuenta y por su rol. La app no necesita enviar identificadores de biblioteca para ocultar contenido: lo que no está permitido responde `404`, igual que un recurso inexistente. Esto se aplica también a la reproducción y a las miniaturas.

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

# Arquitectura y seguridad

[English](../en/architecture.md) · [Volver al README](../../README.md) · [Estado](estado.md)

Cinecat tiene un servidor Go, un cliente web SvelteKit y un despliegue Docker con Apache delante. Las apps Android y Android TV forman parte de la visión, pero todavía no existen. Esta página describe lo que hay construido y separa con claridad lo que aún no está.

## Vista general

```
 Navegador / clientes futuros
            │ HTTPS
            ▼
 ┌─────────────────────┐
 │ Apache del host     │  /api/ y /healthz → servidor
 │ (vhost + TLS)       │  resto            → web
 └───────┬─────────────┘
         │ 127.0.0.1
   ┌─────▼──────┐      ┌──────────────┐
   │  servidor  │      │ contenedor   │
   │  Go :8080  │      │ web :3000    │
   └──┬───┬───┬─┘      └──────────────┘
      │   │   └── FFmpeg/ffprobe (imagen del servidor)
      │   └────── SQLite + caché TMDB (/data)
      └────────── biblioteca de vídeo (/media, solo lectura)
```

El servidor de actualizaciones no vive dentro de los contenedores: el servidor Go escribe `run/update-request.json` y un servicio de systemd del host ejecuta el trabajo. Ver [actualizaciones](actualizaciones.md).

## Componentes

- **Servidor Go (`cmd/cinecatd`):** API REST, autenticación, escaneo, identificación, metadatos, colecciones, series, progreso, ajustes y preparación de la reproducción.
- **Cliente web (`web/`):** SvelteKit + TypeScript + Svelte 5, con sesión en cookie `httpOnly`; nunca guarda el token de la API en JavaScript. Usa hls.js para HLS y `<video>` nativo para reproducción directa.
- **Base de datos SQLite:** migraciones embebidas, `STRICT`, WAL, claves foráneas y `busy_timeout`. Vive en el volumen `cinecat_data`.
- **Caché de TMDB:** junto a la base de datos (`/data/cache/tmdb` en Docker), con respuestas crudas y TTL de 7 días.
- **FFmpeg/ffprobe:** herramienta externa instalada en la imagen del servidor; no se distribuye en el repositorio. Se invoca siempre con listas de argumentos, sin shell.
- **Apache del host:** proxy inverso y TLS. Enruta `/api/` y `/healthz` al servidor y el resto al cliente web, de modo que navegador y API comparten origen.

## Flujo de datos

1. Se crea una biblioteca con una ruta del contenedor (`/media/...`).
2. El escaneo recorre los archivos de vídeo, analiza nombres y carpetas y hace *upsert* en `media_items`.
3. Las películas y series se identifican con TMDB (o OMDb como respaldo de películas) y se guardan fichas, reparto, imágenes y colecciones.
4. La web lee la API con la cookie de sesión; el servidor traduce la cookie a un token Bearer interno.
5. Al reproducir, el servidor decide reproducción directa o HLS y devuelve una URL firmada. El navegador pide el archivo o la playlist y los segmentos.
6. El progreso, el visionado y las preferencias se guardan por usuario en SQLite.

## Modelo de datos principal

| Tabla | Contenido |
|---|---|
| `users`, `sessions` | Usuarios, roles y sesiones; solo se guarda el hash SHA-256 del token. |
| `libraries` | Nombre y ruta de cada biblioteca. |
| `media_items` | Películas y episodios, ruta, título, metadatos, TMDB, estado de identificación, imágenes, créditos y serie/temporada/episodio. |
| `series`, `series_seasons` | Series y temporadas; los episodios son filas de `media_items`. |
| `collections`, `collection_items` | Colecciones automáticas/manuales, miembros, orden y exclusiones. |
| `watch_state`, `playback_progress` | Visto y posición por usuario. |
| `item_preferences` | Audio, subtítulo y calidad por usuario y película. |
| `settings` | Ajustes tipados persistidos; los valores efectivos están en memoria. |

## API y contrato

La API está versionada bajo `/api/v1/` y documentada en [api/openapi.yaml](../../api/openapi.yaml). Los grupos principales son autenticación, bibliotecas, ítems, series, colecciones, ajustes, sistema/actualización y streaming firmado. `GET /healthz` existe además como comprobación de salud.

Las respuestas de error usan `{error}` o `{message}` según el endpoint; los códigos habituales son 401 sin sesión, 403 sin rol, 400 de validación, 404, 409 de conflicto, 415 de contenedor no admitido, 502 de dependencia externa y 503 de FFmpeg o actualización no disponible.

## Reproducción

- **Directa:** `.mp4` y `.m4v` se sirven con `http.ServeContent`, con soporte de rangos.
- **HLS:** el resto de extensiones se recodifica a H.264 + AAC estéreo y se sirve por segmentos MPEG-TS de 6 s. La playlist es VOD si se conoce la duración; si no, se usa la playlist `event` de FFmpeg.
- **Control:** máximo de transcodificaciones simultáneas, procesos con prioridad baja, parada por inactividad y límite de segmentos por delante. Ver [reproducción](reproduccion.md).

## Seguridad

### Implementado

- Contraseñas con **argon2id**; sesiones de 24 h con renovación deslizante; token de 32 bytes y solo su hash en la base.
- Roles `admin` y `user`; los endpoints de administración comprueban el rol en el servidor.
- Cookie de sesión `httpOnly`, `SameSite=Lax` y `Secure` cuando llega por HTTPS.
- Bloqueo tras cinco intentos fallidos y mensajes de login que no distinguen usuario inexistente de contraseña incorrecta.
- URLs de reproducción firmadas con HMAC-SHA256, TTL de 8 horas y clave por proceso.
- Validación de rutas dentro de la biblioteca y escaneo sin seguir enlaces simbólicos.
- Sin telemetría. Las conexiones salientes son a TMDB, OMDb y GitHub solo cuando corresponde.

### Límites conocidos

- El bloqueo de login se indexa por usuario, no por IP; puede usarse para bloquear temporalmente una cuenta.
- No hay CORS: el despliegue depende de que Apache una el origen.
- No hay permisos por biblioteca: cualquier usuario autenticado ve todo el catálogo.
- No hay creación de usuarios ni revocación de sesiones desde la web.
- `/media` se monta en solo lectura; no hay subida ni borrado de archivos.
- Las URLs de reproducción no están ligadas al usuario y dejan de valer al reiniciar el servidor.

El detalle y el orden de mejora están en [estado](estado.md) y [tareas abiertas](../../TODO.md).

## Pila tecnológica

| Parte | Elección | Estado |
|---|---|---|
| Servidor | Go, SQLite, FFmpeg externo | Implementado. |
| Web | TypeScript + SvelteKit + hls.js | Implementado. |
| Android / Android TV | Kotlin + Jetpack Compose + Media3/ExoPlayer, libmpv opcional | Decidido, sin código. |
| Despliegue | Docker Compose + Apache del host | Implementado. |

## Decisiones de seguridad

La plataforma puede exponerse a internet, así que HTTPS, contraseñas fuertes y una superficie mínima son requisitos. El PIN de perfiles y los permisos por biblioteca siguen siendo objetivos de diseño; no deben confundirse con funciones ya disponibles.

Ver también [SECURITY.md](../../SECURITY.md) y [licencias y créditos](licencias-y-creditos.md).

# Auditoría de Cinecat (2026-10-28)

Alcance: seguridad, rendimiento, funcionamiento y compatibilidad del código en `main` (`cb1e372`, 0.22.1 + [Sin publicar]) y de la instalación de producción (Debian, 0.22.1). Hecha por el orquestador (Opus) con pruebas reales: `govulncheck` sobre el binario de producción, `npm audit`, cabeceras de `https://video.warcrafted.com`, registros de producción y medidas de la API en la instancia de prueba con una copia de la base real (1701 vídeos, 17 series, 193 colecciones). No se ha cambiado código ni producción.

## Resumen

| Prioridad | Hallazgo | Arreglo |
|---|---|---|
| 🔴 Crítico | El servidor se compila con **Go 1.22.12**: el binario de producción tiene **43 vulnerabilidades conocidas** de la biblioteca estándar (`net/http`, `crypto/tls`, `crypto/x509`, `mime/multipart`, `net/url`…). Con Go 1.26.9 queda 1 (`crypto/x509`, corregida en 1.26.10). | `Dockerfile`: `FROM golang:1.26-bookworm` (sigue los parches de la 1.26 en cada compilación) y `go.mod` `go 1.26`. |
| 🟠 Alto | **Safari/iOS no reproducirán 30 de las 33 películas H.265**: son MP4 con etiqueta `hev1` (solo 3 son `hvc1`); desde H1, si Safari declara H.265 el servidor los manda en directo y Safari solo acepta `hvc1`. | Directo HEVC solo si `codec_tag_string == hvc1`; si no, copia (TS no depende de la etiqueta) o recodificación. Reduvid/Reduvid-Granja: `-tag:v hvc1` al generar MP4 HEVC (en MKV no aplica). |
| 🟠 Alto | **Sin cabeceras de seguridad** en producción: ni HSTS, ni `X-Frame-Options`/`frame-ancestors` (las páginas de administración se pueden incrustar en otra web: *clickjacking*), ni `X-Content-Type-Options`, ni `Referrer-Policy`. Solo `Server: Apache/2.4.69 (Debian)`. | Cabeceras en el vhost de Apache (`deploy/video.warcrafted.com.conf`) o en `hooks.server.ts`: `Strict-Transport-Security: max-age=31536000`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin`, CSP mínima (`frame-ancestors 'none'; object-src 'none'; base-uri 'self'`). `ServerTokens Prod`. |
| 🟡 Medio | **Listados pesados**: cada página de 48 películas pesa ~320 KB porque cada ítem incluye el reparto completo (`credits`, ~6 KB, el 90 %) que las carátulas no usan. Afecta a móvil, TV y conexiones lentas. | DTO ligero en listados (sin `credits`, `overview` recortado o fuera); los créditos solo en la ficha. |
| 🟡 Medio | **Fuga de información**: los listados y fichas envían `path` (ruta absoluta del archivo en el servidor) a cualquier usuario con sesión. | `path` solo para administradores. |
| 🟡 Medio | **Listado de colecciones lento** (257 ms con 193 colecciones): el recuento usa `IN (SELECT … media_items JOIN libraries)` correlacionado, hasta 3 veces por colección. Con `JOIN` el recuento pasa de 40 ms a 2 ms (medido con SQLite sobre la copia). | Reescribir `collectionItemCountExpr` con `JOIN`. |
| 🟡 Medio | **Memoria de la web**: 572 MB tras 8 h en producción frente a 17 MB recién arrancada; coincide con las dos subidas fallidas por el `fetch` antiguo, que retenía el cuerpo. Ya hay arreglo (streaming, en [Sin publicar]). | Verificar tras publicar 0.23.0; si sigue creciendo, perfilar. Opcional: `NODE_OPTIONS=--max-old-space-size=512`. |
| 🟡 Medio | **Sin copias de seguridad periódicas**: la base solo se copia antes de cada actualización (`~/cinecat-backups`). Un fallo de disco pierde visto, progreso, listas, colecciones y correcciones. | Copia diaria con `sqlite3 .backup` (o desde el servidor con `VACUUM INTO`) y retención de N días; ya figura en el TODO como «copias de seguridad en administración». |
| 🟢 Bajo | **La clave de firma de las URLs de reproducción es aleatoria en cada arranque**: una actualización corta las reproducciones en curso hasta recargar. | Guardar la clave en `/data` (generada una vez, permisos 0600). |
| 🟢 Bajo | Cambiar la contraseña no cierra las demás sesiones (ya en el TODO). | Borrar las otras sesiones del usuario al cambiarla. |
| 🟢 Bajo | El contenedor del servidor corre como root con `/media` en lectura y escritura y FFmpeg analizando archivos (solo los suben administradores). | Aceptable hoy; reconstruir la imagen periódicamente para recoger los parches de FFmpeg de Debian; más adelante, `user:` con la base migrada de propietario. |
| 🟢 Bajo | `http.Server` solo tiene `ReadHeaderTimeout` (ni límites de cuerpo en los JSON); Apache delante lo mitiga y todo requiere sesión. | `http.MaxBytesReader` (p. ej. 1 MB) en los endpoints JSON. |
| 🟢 Bajo | npm: 0 vulnerabilidades en dependencias de producción; 7 de severidad baja en herramientas de desarrollo (`cookie` < 0.7 transitivo). | Actualizar cuando lo haga SvelteKit; no llega a producción. |

## Lo que está bien (comprobado)

- **Inyección SQL:** todos los valores van como parámetros; los nombres de columna concatenados salen de listas fijas (`imageColumns`, órdenes por `switch`, tablas fijas en la papelera).
- **XSS:** ningún `{@html}`/`innerHTML`; lo inyectado en `<html>` son enums validados.
- **CSRF:** comprobación de origen de SvelteKit activa, `SameSite=Lax` y los `PUT`/JSON exigirían *preflight* que el servidor no autoriza.
- **Sesión:** cookie `httpOnly` + `Secure` + `SameSite=Lax`, token aleatorio guardado como hash SHA-256, caducidad 24 h, Argon2id, bloqueo por usuario+IP con límite por IP (verificado en 0.19.1).
- **URLs de reproducción:** HMAC-SHA256 por tipo de recurso + ítem + caducidad (8 h), comparación en tiempo constante.
- **Actualizador:** solo etiquetas `^v\d+\.\d+\.\d+$` existentes en GitHub y que desciendan del commit actual (`--ff-only`): una web comprometida no puede ejecutar nada en el host salvo actualizar a una versión publicada. El riesgo real es la cuenta de GitHub.
- **Rutas de archivo:** subida, papelera, navegador de carpetas y reproducción confinados a las raíces de la biblioteca, sin seguir enlaces simbólicos (revisado en FA/FB/SG1).
- **Exposición:** servidor y web solo en `127.0.0.1`; Apache con redirección HTTP→HTTPS; `/api` sin sesión → 401.
- **Rendimiento:** con la base real casi todo responde en < 40 ms (portal 38 ms, listados 15-22 ms, fichas ~2 ms, similares 79 ms); servidor 110 MB de RAM; índices adecuados. Copia ~37× tiempo real y casi sin CPU; recodificación por gráfica ≈ 11 % de CPU frente a ≈ 303 % por software (H4).
- **Registros de producción:** el servidor no registra errores; en la web solo las 2 subidas fallidas conocidas y un 500 aislado durante un reinicio.

## Compatibilidad

| Ámbito | Estado | Observación |
|---|---|---|
| Chrome / Edge (escritorio y Android) | ✅ | hls.js; H.265 directo o copia si el equipo lo decodifica. |
| Firefox | ✅ | Sin H.265 en general → copia H.264 o recodificación (gráfica tras H4). |
| Safari macOS / iPadOS / iOS ≥ 17.1 | ⚠️ | hls.js con ManagedMediaSource; **MP4 `hev1` fallarían en directo** (hallazgo alto). |
| iOS < 17.1 (HLS nativo) | ✅ | TS H.264; la copia HEVC no se ofrece (bien, el nativo no admite HEVC en TS). |
| Smart TV (navegador del televisor) | ⚠️ | Funciona si tiene MSE; sin navegación por mando ni pantalla de TV. |
| Audio | ⚠️ | En HLS todo pasa a AAC estéreo: se pierde el 5.1 (AC-3/E-AC-3/DTS). |
| Subtítulos de texto (SRT, ASS, VTT, mov_text) | ✅ | A WebVTT; ASS pierde estilos y posiciones. |
| Subtítulos de imagen (PGS de Blu-ray, VobSub de DVD) | ❌ | Se listan pero no se pueden mostrar (ni quemar en la imagen). Ej.: `Deep Impact.mkv` tiene 3 PGS. |
| HDR / 10 bits | ⚠️ | Sin *tone mapping*: un HDR recodificado sale lavado. Hoy hay 0 archivos de 10 bits. |
| Entrelazado (DVD, grabaciones TV) | ⚠️ | Sin desentrelazar: «peines» en movimiento. |
| AV1 | ⚠️ | 1 archivo; siempre se recodifica aunque muchos navegadores lo decodifiquen. |
| Chromecast / AirPlay | ❌ | No hay envío a TV desde el móvil. |
| Instalable (PWA) | ❌ | Sin `manifest` ni iconos de pantalla de inicio. |
| Velocidad adaptativa (ABR) | ❌ | Solo selección manual 480/720/1080. |

## Plan de arreglos (ejecutable por una sesión nueva)

Versión sugerida: **0.23.1** (parche: correcciones de seguridad y rendimiento sin funciones nuevas), después de publicar 0.23.0.

1. **A1 — Go actual.** `Dockerfile` (`golang:1.26-bookworm`) y `go.mod` (`go 1.26`); `go mod tidy`; pruebas; `govulncheck ./...` sin vulnerabilidades de la biblioteca estándar alcanzables. Ejecutor: la propia sesión (trivial). Verificar en la instancia de prueba que compila y arranca.
2. **A2 — HEVC `hev1`.** `internal/playback/fileinfo.go` (leer `codec_tag_string`) y `decision.go` (directo HEVC solo con `hvc1`); tests. AgentRelay, esfuerzo medio.
3. **A3 — Cabeceras de seguridad.** En `hooks.server.ts` (viajan con la web) y `ServerTokens Prod` en el vhost; comprobar que hls.js, los carteles de TMDB (`image.tmdb.org`) y las fuentes siguen cargando con la CSP. AgentRelay, esfuerzo medio; el cambio de Apache lo aplica el orquestador con aprobación.
4. **A4 — Listados ligeros y `path` solo para admin.** DTO de listado sin `credits`; `path` omitido para no administradores; la web no debe depender de esos campos en tarjetas. AgentRelay, esfuerzo medio.
5. **A5 — Recuento de colecciones con JOIN.** `internal/library/collections_http.go`. AgentRelay, esfuerzo bajo.
6. **A6 — Clave de firma persistente y cierre de sesiones al cambiar contraseña.** AgentRelay, esfuerzo medio.
7. **A7 — Copia de seguridad diaria de la base** (nueva función, versión menor aparte): `VACUUM INTO` a `/data/backups` con retención configurable y sección en Administración. AgentRelay, esfuerzo medio.

Compatibilidad que conviene abordar después (versiones menores): PWA (`manifest`, iconos), subtítulos de imagen quemados al recodificar (opción por pista), desentrelazado automático (`bwdif` si `field_order` es entrelazado), AV1 directo/copia con capacidades, conservar 5.1 en AAC multicanal cuando el cliente lo admita (aplicaciones nativas).

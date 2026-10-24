# Arquitectura y seguridad

[English](../en/architecture.md) · [Volver al README](../../README.md)

> Estado: **propuesta de diseño**. Nada de esto está implementado. Servidor: **Go** (decidido). Web y Android: a decidir en la Fase 0.

## Visión general

```
 ┌────────────┐   ┌──────────────┐   ┌──────────────────┐
 │ Cliente web│   │ App Android │   │ App Android TV   │
 └─────┬──────┘   └──────┬───────┘   └────────┬─────────┘
       │      HTTPS · API REST versionada       │
       └──────────────────┬─────────────────────┘
                   ┌──────▼───────┐
                   │   Servidor   │── FFmpeg (externo)
                   │   Cinecat    │── Base de datos local
                   └──────┬───────┘── Caché de imágenes
                          │
                 Biblioteca de vídeo (disco)
                 Fuentes de metadatos (TMDB, ...)
```

## Componentes

- **Servidor:** escanea la biblioteca, identifica títulos, descarga y cachea metadatos, gestiona usuarios y sesiones, decide la forma de reproducción y sirve el vídeo. Orquesta **FFmpeg** como herramienta externa (no se distribuye en el repositorio).
- **Cliente web:** PC, móvil y tableta. Reproduce con HLS y APIs del navegador.
- **App Android y Android TV:** reproductor nativo con descodificación por hardware cuando exista, y alternativa por software. La interfaz de televisión se diseña para el mando (navegación por foco).
- **API:** REST, versionada y documentada con OpenAPI (contrato primero). Los clientes se escriben contra el contrato.
- **Base de datos:** local y simple al principio (por ejemplo SQLite).

## Pila tecnológica

| Parte | Elección | Nota |
|---|---|---|
| Servidor | **Go** (decidido) | Binario único, despliegue sencillo en Debian y Docker, concurrencia nativa para gestionar procesos FFmpeg, mantenimiento cómodo en solitario. |
| Web | TypeScript con un framework moderno + hls.js o Shaka (a decidir en Fase 0) | Diseño propio, sin plantillas genéricas. |
| Android / TV | Kotlin + Jetpack Compose (+ librerías para TV) + Media3/ExoPlayer, libmpv opcional (a decidir en Fase 0) | Comprobar licencias antes de usar cada pieza. |

## Seguridad

La plataforma puede quedar expuesta a internet, así que se diseña con estas reglas:

1. **HTTPS obligatorio** fuera de la red local. Se documenta el uso de un proxy inverso (por ejemplo Caddy) y la alternativa de una VPN (Tailscale/WireGuard) para uso personal.
2. **Autenticación:** usuario y contraseña con *hash* robusto (argon2id o similar), sesiones con *tokens* de vida corta y renovables, y cierre de sesión que invalida el *token*.
3. **Protección frente a fuerza bruta:** límite de intentos, bloqueo temporal y avisos.
4. **PIN:** un PIN corto **no** es suficiente por sí solo para el acceso remoto. Se usa para cambiar de perfil dentro de un dispositivo ya autenticado (por ejemplo, el televisor). Para el acceso remoto, contraseña fuerte y, más adelante, segundo factor (TOTP o passkey).
5. **URLs de reproducción firmadas** y de corta duración. Nunca contraseñas ni *tokens* permanentes en la URL.
6. **Roles:** administrador y usuarios; permisos por biblioteca.
7. **Registro de auditoría** de inicios de sesión y acciones administrativas.
8. **Sin telemetría.** Conexiones salientes solo a las fuentes de metadatos configuradas.
9. **Procesos externos seguros:** FFmpeg se invoca con argumentos en lista, nunca por concatenación de texto ni con intérprete de órdenes.
10. **Dependencias:** revisión de licencias y de vulnerabilidades conocidas.

Ver también [SECURITY.md](../../SECURITY.md).

## Despliegue previsto

- Servidor en Debian (nativo o contenedor Docker).
- FFmpeg instalado en el sistema.
- Aceleración por hardware del servidor (VAAPI, NVENC, QSV) como opción avanzada.
- Imagen de contenedor: si algún día incluye FFmpeg, revisar las obligaciones de licencia.

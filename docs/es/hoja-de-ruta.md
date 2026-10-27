# Hoja de ruta

[English](../en/roadmap.md) · [Volver al README](../../README.md) · [Estado](estado.md)

La versión actual es **0.20.0**. El detalle de funciones, versiones y límites está en [estado](estado.md); esta página solo resume las fases.

## Fase 0: preparación (`0.0.x`) — hecha

Documentación bilingüe, nombre, pila tecnológica, licencia GPL-3.0 con atribución y dirección visual «Marquesina».

## Fase 1: servidor mínimo (`0.1.0`) — hecha

Servidor Go, SQLite y migraciones, autenticación con argon2id, escaneo, analizador de nombres, contrato OpenAPI y reproducción directa/HLS inicial.

## Fase 2: cliente web (`0.2.0`) — hecha, con ajustes posteriores

Login y sesión, biblioteca, ficha, reproductor, temas claro/oscuro y accesibilidad. Los perfiles con PIN no se incluyeron; la cuenta actual permite cambiar la contraseña, no crear perfiles.

## Fase 3: reproducción avanzada (`0.8.0`–`0.13.0`) — parcial

Se implementaron la transcodificación de respaldo, la segmentación progresiva, los saltos, la supervisión de FFmpeg, los controles propios, audio, subtítulos de texto y preferencias. La decisión por capacidades del dispositivo, H.265, hardware, HDR, subtítulos de imagen y ABR siguen fuera.

## Fase 3b: fichas, series y administración (`0.11.0`–`0.16.0`) — hecha

TMDB, fichas completas, colecciones, series, ajustes en caliente, panel de administración, actualizaciones desde la web y servicio actualizador del host.

## Próximas tandas

El orden acordado es: favoritos y próximas a ver; colecciones desde la ficha e identificación de colecciones; bibliotecas con varias carpetas y navegador; subida/borrado con papelera; usuarios y perfil; sesiones, dispositivos, actividad y copias; subtítulos subidos; complementos; DLNA/UPnP; tipos de biblioteca; clientes Android y Android TV. El detalle está en [estado](estado.md).

## Fase 4: Android y Android TV — sin empezar

Kotlin + Jetpack Compose + Media3/ExoPlayer, con libmpv opcional. No hay código ni plan detallado. La web es el único cliente terminado.

## Fase 5: primera versión estable (`1.0.0`) — futura

Revisión de seguridad y licencias, documentación de usuario, empaquetado y despliegue reproducible. No hay fecha.

## Fuera de alcance por ahora

Televisión en directo y contenido de internet; fotos y presentaciones; plataformas distintas de web, Android y Android TV. Se valorarán más adelante.

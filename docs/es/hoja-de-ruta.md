# Hoja de ruta

[English](../en/roadmap.md) · [Volver al README](../../README.md) · [Estado](estado.md)

La versión actual es **0.48.0**. El detalle de funciones, versiones y límites está en [estado](estado.md); esta página solo resume las fases.

## Fase 0: preparación (`0.0.x`) — hecha

Documentación bilingüe, nombre, pila tecnológica, licencia GPL-3.0 con atribución y dirección visual «Marquesina».

## Fase 1: servidor mínimo (`0.1.0`) — hecha

Servidor Go, SQLite y migraciones, autenticación con argon2id, escaneo, analizador de nombres, contrato OpenAPI y reproducción directa/HLS inicial.

## Fase 2: cliente web (`0.2.0`) — hecha, con ajustes posteriores

Login y sesión, biblioteca, ficha, reproductor, temas claro/oscuro y accesibilidad. Los perfiles con PIN no se incluyeron; la cuenta actual permite cambiar la contraseña, no crear perfiles.

## Fase 3: reproducción avanzada (`0.8.0`–`0.13.0`) — parcial

Se implementaron la transcodificación de respaldo, la segmentación progresiva, los saltos, la supervisión de FFmpeg, los controles propios, audio, subtítulos de texto y preferencias. Después llegaron la decisión por capacidades del dispositivo, H.265, la copia sin recodificar y la aceleración por hardware. HDR, desentrelazado, subtítulos de imagen y ABR siguen fuera.

## Fase 3b: fichas, series y administración (`0.11.0`–`0.16.0`) — hecha

TMDB, fichas completas, colecciones, series, ajustes en caliente, panel de administración, actualizaciones desde la web y servicio actualizador del host.

## Próximas tandas

Ya están hechos los favoritos y próximas a ver, las colecciones y su identificación en TMDB, las bibliotecas con varias carpetas, la subida y papelera, los usuarios, las sesiones, la actividad, las copias y los clientes Android y Android TV. Quedan el perfil de usuario, los subtítulos subidos, la reproducción avanzada (HDR, subtítulos de imagen, ABR y audio), los complementos, DLNA/UPnP y otros tipos de biblioteca. El detalle está en [estado](estado.md).

## Fase 4: Android y Android TV — hecha, con mejoras pendientes

Kotlin + Jetpack Compose + Media3/ExoPlayer, con libmpv opcional. Hecho: el servidor (sesiones de dispositivo, vinculación de la TV por código, capacidades nativas en la reproducción, contrato estable de la API e imágenes de TMDB servidas por el propio servidor) y un solo APK para móvil, tableta y TV que inicia sesión, recorre la biblioteca y reproduce. Desde la 0.35.0 la app se publica firmada en GitHub y se actualiza sola. Desde la 0.36.0 la TV se maneja con el mando con foco visible; desde la 0.40.0 tiene índice A–Z, desde la 0.43.0 imagen en imagen en el móvil, y desde la 0.46.0 recuerda audio, subtítulos y calidad. Falta la reproducción en segundo plano y HLS con AAC 5.1 en los equipos que lo admitan.

## Fase 5: primera versión estable (`1.0.0`) — futura

Revisión de seguridad y licencias, documentación de usuario, empaquetado y despliegue reproducible. No hay fecha.

## Fuera de alcance por ahora

Televisión en directo y contenido de internet; fotos y presentaciones; plataformas distintas de web, Android y Android TV. Se valorarán más adelante.

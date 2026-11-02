# Hoja de ruta

[English](../en/roadmap.md) · [Volver al README](../../README.md) · [Estado](estado.md)

La versión actual es **0.58.0**. El detalle de funciones, versiones y límites está en [estado](estado.md); esta página solo resume las fases.

## Fase 0: preparación (`0.0.x`) — hecha

Documentación bilingüe, nombre, pila tecnológica, licencia GPL-3.0 con atribución y dirección visual «Marquesina».

## Fase 1: servidor mínimo (`0.1.0`) — hecha

Servidor Go, SQLite y migraciones, autenticación con argon2id, escaneo, analizador de nombres, contrato OpenAPI y reproducción directa/HLS inicial.

## Fase 2: cliente web (`0.2.0`) — hecha, con ajustes posteriores

Login y sesión, biblioteca, ficha, reproductor, temas claro/oscuro y accesibilidad. Los perfiles con PIN no se incluyeron; la cuenta actual permite cambiar la contraseña, no crear perfiles.

## Fase 3: reproducción avanzada (`0.8.0`–`0.13.0`) — parcial

Se implementaron la transcodificación de respaldo, la segmentación progresiva, los saltos, la supervisión de FFmpeg, los controles propios, audio, subtítulos de texto y preferencias. Después llegaron la decisión por capacidades del dispositivo, H.265, la copia sin recodificar, la aceleración por hardware, el desentrelazado y el mapeo HDR a SDR al recodificar (0.55.0, pendientes de probar con archivos reales) y AAC 5.1 para la TV (0.56.0, pendiente de probar en la TV). En 0.58.0 llegaron los subtítulos de imagen incrustados (PGS, VobSub y DVB): se eligen y se queman en el vídeo al recodificar por HLS, nunca en copia ni con VA-API. Siguen fuera ABR y los perfiles por dispositivo.

## Fase 3b: fichas, series y administración (`0.11.0`–`0.16.0`) — hecha

TMDB, fichas completas, colecciones, series, ajustes en caliente, panel de administración, actualizaciones desde la web y servicio actualizador del host.

## Próximas tandas

Ya están hechos los favoritos y próximas a ver, las colecciones y su identificación en TMDB, las bibliotecas con varias carpetas, la subida y papelera, los usuarios, las sesiones, la actividad, las copias, el historial, las filas del portal personalizables, los subtítulos subidos por administradores y los de imagen incrustados, el desentrelazado y HDR a SDR, el AAC 5.1 para la TV y los clientes Android y Android TV. Quedan la reproducción avanzada (perfiles por dispositivo, ABR y audio no soportado), los complementos, DLNA/UPnP, otros tipos de biblioteca y verificar el flujo de GitHub Actions. El detalle está en [estado](estado.md).

## Fase 4: Android y Android TV — hecha, con mejoras pendientes

Kotlin + Jetpack Compose + Media3/ExoPlayer, con libmpv opcional. Hecho: el servidor (sesiones de dispositivo, vinculación de la TV por código, capacidades nativas en la reproducción, contrato estable de la API e imágenes de TMDB servidas por el propio servidor) y un solo APK para móvil, tableta y TV que inicia sesión, recorre la biblioteca y reproduce. Desde la 0.35.0 la app se publica firmada en GitHub y se actualiza sola. Desde la 0.36.0 la TV se maneja con el mando con foco visible; desde la 0.40.0 tiene índice A–Z, desde la 0.43.0 imagen en imagen en el móvil, desde la 0.46.0 recuerda audio, subtítulos y calidad, desde la 0.49.0 ofrece «Desde el principio» y desde la 0.49.1 nombra las pistas sin título. La 0.56.0 añade AAC 5.1 en la TV, pendiente de probar con un equipo real, y la 0.58.0 los subtítulos de imagen. Falta la reproducción en segundo plano.

## Fase 5: primera versión estable (`1.0.0`) — futura

Revisión de seguridad y licencias, documentación de usuario, empaquetado y despliegue reproducible. No hay fecha.

## Fuera de alcance por ahora

Televisión en directo y contenido de internet; fotos y presentaciones; plataformas distintas de web, Android y Android TV. Se valorarán más adelante.

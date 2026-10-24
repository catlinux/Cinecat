# Hoja de ruta

[English](../en/roadmap.md) · [Volver al README](../../README.md)

> Plan orientativo. Cada fase termina con una revisión antes de pasar a la siguiente. Versionado `x.y.z`.

## Fase 0: preparación (`0.0.x`)

- Documentación inicial bilingüe (**hecho en `0.0.1`**).
- Comprobar el nombre, decidir la pila tecnológica, convención de commits y dirección de diseño. Licencia ya decidida (GPL-3.0 con atribución).
- Crear `CLAUDE.md` y la estructura del repositorio de código.

## Fase 1: servidor mínimo (`0.1.0`)

- Escaneo de biblioteca y análisis de nombres de archivo.
- Metadatos en `es-ES` (TMDB) y caché de imágenes.
- Usuarios, sesiones y API versionada con contrato OpenAPI.
- Reproducción directa y cambio de contenedor (HLS fMP4).
- Pruebas automáticas con vídeos de prueba generados con FFmpeg.

## Fase 2: cliente web (`0.2.0`)

- Inicio de sesión, perfiles con PIN, biblioteca, ficha de película o serie.
- Reproductor con subtítulos, selección de audio y capítulos.
- Diseño propio, claro y oscuro, adaptable a móvil y tableta.
- Acceso seguro desde fuera de casa (guía de HTTPS y VPN).

## Fase 3: decisión de reproducción avanzada (`0.3.0`)

- Perfiles de capacidades por dispositivo y preferencia de descodificación.
- Transcodificación parcial y completa, con aceleración por hardware opcional.
- Casos especiales: 10 bits, HDR, subtítulos de imagen, audio no soportado.

## Fase 4: Android y Android TV (`0.4.0`)

- App Android con reproductor nativo y alternativa por software.
- Interfaz de televisión pensada para el mando.
- Descarga sin conexión (si se decide).

## Fase 5: pulido y primera versión estable (`1.0.0`)

- Revisión de seguridad y de licencias.
- Documentación de usuario y glosario «para tontos».
- Empaquetado y despliegue (Docker, paquete Debian).

## Después

- Fotos y presentaciones.
- Segmentos de intro/créditos.
- Contenido de internet y televisión en directo (solo si se decide).
- Otras plataformas.

Detalle de tareas en [TODO.md](../../TODO.md).

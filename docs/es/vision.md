# Visión y decisiones

[English](../en/vision.md) · [Volver al README](../../README.md) · [Estado](estado.md)

## El problema

Quien guarda su propia biblioteca de vídeo en casa suele acabar usando un servidor multimedia ajeno. Funciona, pero la interfaz no siempre es cómoda, la reproducción falla en algunos dispositivos y el usuario no controla bien quién descodifica: el aparato o el servidor.

## La propuesta

**Cinecat**: un servidor y unos clientes propios, con una interfaz con personalidad, que eligen la mejor forma de reproducir cada vídeo y que obtienen portadas e información en español de bases de datos existentes. El servidor y el cliente web ya están implementados; Android y Android TV siguen en el horizonte.

## Decisiones tomadas (2026-10-24)

| # | Decisión |
|---|---|
| 1 | **Servidor y clientes propios**, con nombre propio (Cinecat). |
| 2 | Se **aprovecha todo lo que sea legal** de otros proyectos libres, con sus créditos. Lo demás se escribe de nuevo. |
| 3 | Plataformas: **web** (PC, móvil, tableta; accesible desde fuera de casa) y **Android / Android TV**. Nada más de momento. |
| 4 | **Reproducción óptima**: H.265 preferido, H.264 de reserva; el dispositivo declara qué sabe hacer y el servidor decide. La implementación actual usa H.264 por software; H.265 y los perfiles por dispositivo son diseño futuro. |
| 5 | **Metadatos de bases de datos existentes**, en español de España. No creamos la nuestra. |
| 6 | **Acceso protegido** con usuario y contraseña; el PIN para perfiles es un objetivo de diseño, no una función actual. |
| 7 | **Interfaz propia y distinta** a lo habitual en proyectos generados con IA. |
| 8 | Documentación en **castellano e inglés**, commits en **castellano**, versiones **x.y.z**. |
| 9 | Autor y mantenedor único: **Catlinux**. |
| 10 | Licencia: **GPL-3.0 con atribución** (artículo 7), preservando autoría y créditos. |

## Estado y prioridades

El servidor, la web, las fichas, las series, las colecciones, la administración y las actualizaciones están implementados. La siguiente prioridad es el uso multiusuario seguro y, después, favoritos, colecciones desde la ficha, bibliotecas con varias carpetas y gestión de archivos. El orden completo, con lo que ya existe y lo que no, está en [estado](estado.md).

## Fuera de alcance por ahora

- Televisión en directo y contenido de internet.
- Fotos y presentaciones.
- Plataformas distintas de web, Android y Android TV.

## Principios

1. **Fácil por defecto, potente si lo quieres.** Las opciones se explican en lenguaje llano.
2. **Nunca perder ni estropear los originales.**
3. **Privacidad:** sin telemetría ni conexiones innecesarias.
4. **Créditos siempre:** lo que se reutilice se cita y se respeta.
5. **Honestidad técnica:** si algo no se puede garantizar (por ejemplo H.265 en un navegador), se dice y se ofrece alternativa.

## Proyectos relacionados

- **Reduvid** (proyecto hermano del mismo autor): compresor de vídeo para dejar los archivos listos para Jellyfin y para Cinecat.

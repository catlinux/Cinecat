# Cinecat

Servidor y clientes propios (web, Android, Android TV) para reproducir una biblioteca de vídeo personal. Ver [README.md](README.md) y `docs/es/` para la visión completa; este archivo solo recoge lo que no esté ya en las reglas globales del usuario.

## Decisiones ya tomadas (no las cuestiones sin motivo nuevo)

- Autor y mantenedor único: **Catlinux**. Nunca otro usuario como autor/coautor en el historial.
- Licencia: **GPL-3.0** con la condición adicional del artículo 7 (conservar atribución de Catlinux). Ver [LICENSE](LICENSE) y [docs/es/licencias-y-creditos.md](docs/es/licencias-y-creditos.md).
- Pila: servidor en **Go**; web en **TypeScript + SvelteKit** + hls.js; Android/Android TV en **Kotlin + Jetpack Compose** + Media3/ExoPlayer (libmpv opcional). Ver `docs/es/arquitectura.md`.
- Dirección visual de la interfaz: **«Marquesina»** (cine de barrio de los 50-60, serif Fraunces, carteles ilustrados propios, nunca fotos de terceros). Ver la sección «Dirección visual» del README.
- Documentación en castellano e inglés, siempre sincronizada. Commits en castellano.
- Plataformas: web, Android, Android TV. Nada más por ahora (ver «Fuera de alcance» en el README).

## Reutilización de código de terceros

Antes de copiar código de otro proyecto, comprobar su licencia **en el archivo o commit exacto**, nunca de memoria ni de esta lista:

| Licencia confirmada | Qué permite |
|---|---|
| GPL-3.0, MPL-2.0, Apache-2.0, MIT | Copiar literal, conservando avisos de copyright y créditos en `CREDITS.md` / `THIRD_PARTY_NOTICES.md` |
| GPL-2.0-only (Jellyfin y sus clientes oficiales, Wholphin) | Solo como inspiración; reescribir desde cero, sin copiar estructura ni texto |

Nunca hacer fork ni importar historial ajeno: todo entra en commits nuevos de este repositorio.

## Qué no hacer

- No incluir FFmpeg en el repositorio (herramienta externa, se detecta en el sistema).
- No telemetría ni conexiones salientes salvo a las fuentes de metadatos configuradas.
- No interfaz con aspecto genérico de IA (gradientes morados, tarjetas con sombra por defecto, iconos sin criterio): seguir la dirección «Marquesina».
- No copiar código GPL-2.0-only literal, ni mezclarlo con código GPL-3.0-only propio.

## Pruebas

Separar el motor (decisión de reproducción, análisis de nombres, metadatos) de la interfaz. Generar los vídeos de prueba con FFmpeg en el momento de probar, nunca subir binarios de vídeo al repositorio.

@AGENTS.md

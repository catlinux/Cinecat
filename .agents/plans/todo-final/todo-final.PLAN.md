# Cierre del TODO (0.50.x)

Orden de ejecución, una tarea de AgentRelay por paso (el árbol de trabajo es único: no lanzar dos a la vez). Tras cada una: revisar diff, `go test ./internal/...` / `npm test` / `gradlew.bat testDebugUnitTest assembleDebug` según toque, subir VERSION + CHANGELOG es/en + TODO, commit con archivos por nombre, etiqueta y push (autorizado en la sesión del 2026-10-31).

Decisiones de diseño tomadas (no reabrir sin motivo):
- Subtítulos subidos: solo administradores, `.srt`/`.vtt`/`.ass` de hasta 2 MB, guardados junto a la base en `subtitles/<id>/`, ids `u<N>`; compartidos por todos los usuarios con acceso al ítem.
- Perfil: historial propio de reproducciones (solo lectura, con «quitar») y filas del portal que el usuario puede ocultar; el servidor aplica la preferencia en `/api/v1/home`, así que web y apps la respetan.
- Avisos de login: eventos de bloqueo en tabla SQLite (compartida entre instancias y persistente), visibles para el administrador en Resumen y en el registro.
- Reproducción avanzada: solo desentrelazado (yadif al recodificar) y HDR→SDR (tonemap al recodificar). Fuera de alcance: subtítulos de imagen quemados y perfiles por dispositivo (anotados como pendientes).
- AAC 5.1 por HLS cuando el cliente declara `mch` y no es navegador.
- Fuera: copias de seguridad externas (el usuario no las quiere), Complementos, DLNA, otras plataformas.

## Pasos

1. [ ] **TMDB: refresco periódico** de metadatos guardados (< 6 meses). Servidor. 0.50.0.
2. [ ] **Login: persistencia y aviso al administrador.** Servidor + web (Resumen). 0.51.0.
3. [ ] **Historial del usuario** (`GET/DELETE /api/v1/history`) + página web. 0.52.0.
4. [ ] **Filas del portal personalizables** por usuario. Servidor + web. 0.53.0.
5. [ ] **Subtítulos subidos** (admin): servidor + web. 0.54.0.
6. [ ] **Desentrelazado y HDR→SDR** al recodificar. Servidor. 0.55.0.
7. [ ] **AAC 5.1 por HLS** para apps. Servidor (+ app si hace falta declarar `mch`). 0.56.0.
8. [ ] Docs es/en al día y TODO final.

Imprevistos de diseño: parar y anotarlos aquí.

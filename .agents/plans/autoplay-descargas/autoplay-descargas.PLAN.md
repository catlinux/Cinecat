# Reproducción automática por serie, intro/créditos y descargas

Encargo del usuario (2026-11-03): terminar intro, créditos y «siguiente episodio»; en cada serie debe poder activarse o desactivarse la reproducción automática de forma fácil y visible; seguir con las descargas móviles.

Ejecución secuencial. Misma metodología que `resto-todo.PLAN.md`: AgentRelay, revisión del diff, tests (`go test ./internal/...` con FFmpeg en el PATH, `npm run check` y `npm test`, `gradlew.bat testDebugUnitTest assembleDebug`), commit por nombre de archivo, versión y CHANGELOG/TODO/docs en el commit que cierra cada paso, etiqueta local, **sin push sin aprobación**.

## Diseño (decidido)

- Ajuste por usuario y serie: `series_autoplay` (sin fila = activada). `autoplay_next` en `GET /api/v1/series/{id}` y en `GET /api/v1/items/{id}/play` (true para ítems que no son episodio). `PUT/GET /api/v1/series/{id}/autoplay`.
- Con la reproducción automática **desactivada**, no hay cuenta atrás ni salto automático al terminar un episodio; el botón manual «Siguiente episodio» en los créditos sigue disponible.
- Con ella **activada**, al terminar el episodio (o al llegar a los créditos con el botón pulsado) pasa al siguiente tras una cuenta atrás de 8 s cancelable. Sin siguiente episodio no hace nada.
- Web: interruptor «Reproducir el siguiente episodio automáticamente» bien visible en la ficha de la serie y en el reproductor (junto a la cuenta atrás). Android/TV: interruptor equivalente en la ficha de la serie y en la cuenta atrás; «Siguiente episodio» en el reproductor.

## Pasos

1. [x] **Servidor**: migración 0030, rutas, `autoplay_next`. AgentRelay, medio. Versión 0.64.0 (menor) se asigna al cerrar el paso 3.
2. [x] **Web**: interruptor en la ficha de serie y en el reproductor; el reproductor respeta `autoplay_next`. AgentRelay, medio.
3. [x] **Android/TV**: navegación al siguiente episodio desde el reproductor con cuenta atrás cancelable, botón «Siguiente episodio» en créditos, interruptor en la ficha de serie. Docs es/en. Cierra **0.64.0**. AgentRelay, medio.
4. [x] **Probar las descargas en el móvil** (Pixel por adb), corregir lo que falle y cerrar **0.64.1** o lo que toque.
   - Prueba en el Pixel (app 0.64.0, servidor 0.64.0): Logan, El médico y Proyecto Power se rechazan con «necesita conversión» (solo `direct` = MP4/M4V compatible se puede descargar). No se ha visto aún una descarga real; la biblioteca parece ser casi toda MKV o incompatible. Pendiente decidir: descarga convertida a MP4 desde el servidor.

Imprevistos de diseño:
- Las descargas solo servían para MP4: se corrigió en 0.65.0 (`download_direct` en `/play`, MKV incluido). Falta ver una descarga real en el Pixel.

# SubtÃ­tulos de imagen quemados (PGS, VobSub, DVB)

Hoy `DiscoverSubtitles` omite las pistas incrustadas que no son texto (`SubtitleStream.Text == false`). Objetivo: que se puedan elegir y se vean, quemÃ¡ndolas en el vÃ­deo al recodificar por HLS.

EjecuciÃ³n secuencial (Ã¡rbol Ãºnico). Tras cada paso: revisar diff, `go test ./internal/...` / `npm test` / `gradlew.bat testDebugUnitTest assembleDebug`, commit por nombre de archivo, **sin push**. La versiÃ³n (0.58.0) y el CHANGELOG se suben en el Ãºltimo paso.

## DiseÃ±o (decidido, no reabrir)

- Las pistas de imagen incrustadas entran en `DiscoverSubtitles` con id `e<N>` (N = posiciÃ³n en `info.Subtitles`, igual que las de texto) y un campo nuevo `Image bool` en `SubtitleTrack`. CÃ³decs: `hdmv_pgs_subtitle`, `dvd_subtitle`, `dvb_subtitle`.
- `subtitleDTO` gana `burn_in bool` (`json:"burn_in"`). Las de imagen llevan `burn_in: true` y `url: ""`; `GET /api/v1/stream/{id}/subtitles/{file}` para una de ellas responde 404.
- Seleccionar una pista de imagen = pedir `/play?subtitle=e<N>`. Si `selected_subtitle` es de imagen: el modo pasa a HLS recodificado (nunca copia, nunca VA-API: se fuerza la ruta por CPU), `reason` pasa a `subtitle` y la URL de HLS lleva `burnsub=<N>`. Las URLs de segmentos conservan `burnsub`, y la clave de cachÃ© HLS (`streamKey`) lo distingue.
- Filtro FFmpeg: `-filter_complex "[0:V:0]<desentrelazado/HDR si toca>[b];[b][0:s:N]overlay[v];[v]scale=...[vout]"` y `-map "[vout]"`; sin filtros previos, `[0:V:0][0:s:N]overlay`. Los filtros de entrelazado y HDR se aplican antes del overlay.
- Un id de imagen en `/stream/...` (clientes nativos) sin `burnsub` no se acepta como subtÃ­tulo de texto: nunca se intenta convertir a WebVTT.
- Preferencias guardadas (`subtitle_id`) que apunten a una pista de imagen funcionan igual que las de texto.
- No se tocan los subtÃ­tulos de texto, externos ni subidos.

## Pasos

1. [x] **Servidor** (`internal/playback`, `internal/library`, `api/openapi.yaml`, tests). Ejecutor: AgentRelay, esfuerzo alto. Terminado: tests de Go pasan (incluye una prueba real con FFmpeg que genera un MKV con subtÃ­tulo PGS o dvd_subtitle â€”por ejemplo `-c:s dvdsub` desde un `.srt` convertidoâ€” y comprueba que el segmento resultante se genera y que cambia respecto a la versiÃ³n sin quemar), `apicontract` verde, `burn_in`/`reason: subtitle`/`burnsub` documentados en `api/openapi.yaml`.
2. [x] **Web** (`web/src/lib/components/Player.svelte` y componentes de menÃº): las pistas con `burn_in` aparecen en el menÃº de subtÃ­tulos con una marca (Â«imagenÂ»); elegirlas recarga la reproducciÃ³n con `subtitle=<id>` conservando la posiciÃ³n (como ya hace el audio); no se crea `<track>` para ellas. Tests con vitest. Ejecutor: AgentRelay, esfuerzo medio.
3. [x] **Android y Android TV** (`PlayerViewModel.kt`, `PlayerScreen.kt`, `api` DTO `burnIn`): mismo comportamiento; `PlayerSubtitle` ya recarga con `subtitleId`. Tests unitarios. Ejecutor: AgentRelay, esfuerzo medio.
4. [ ] **VersiÃ³n 0.58.0**, CHANGELOG es/en, TODO es/en (quitar Â«subtÃ­tulos de imagenÂ»), docs es/en (`reproduccion.md`/`playback.md`, `api`, `estado`/`status`, `android`). Ejecutor: AgentRelay esfuerzo bajo; la prueba en la TV con un vÃ­deo generado la hace la sesiÃ³n.

Imprevistos de diseÃ±o: parar y anotarlos aquÃ­.

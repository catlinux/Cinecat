# Perfiles por dispositivo

Cada dispositivo (sesiÃ³n) guarda ajustes propios que sirven de valor por defecto al reproducir: calidad mÃ¡xima, idioma de audio preferido e idioma de subtÃ­tulo preferido. Decidido con el usuario: Â«se guarda por dispositivoÂ».

EjecuciÃ³n secuencial (Ã¡rbol Ãºnico). Tras cada paso: revisar diff, tests (`go test ./internal/...` con FFmpeg en PATH, `npm run check` y `npm test`, `gradlew.bat testDebugUnitTest assembleDebug`), commit por nombre de archivo, **sin push**. VersiÃ³n 0.59.0 y CHANGELOG en el Ãºltimo paso.

## DiseÃ±o (decidido, no reabrir)

- Un dispositivo es una **sesiÃ³n**: el perfil vive en una tabla nueva `session_profiles(session_id PK â†’ sessions ON DELETE CASCADE, max_quality TEXT NOT NULL DEFAULT '', audio_language TEXT NOT NULL DEFAULT '', subtitle_language TEXT NOT NULL DEFAULT '')` (migraciÃ³n 0027). Si la sesiÃ³n se cierra o se vuelve a iniciar, el perfil vuelve al valor vacÃ­o: es aceptable y se documenta.
- Valores: `max_quality` âˆˆ `""` (sin lÃ­mite), `"480"`, `"720"`, `"1080"`. `audio_language`: `""` (sin preferencia) o cÃ³digo de idioma de 2 o 3 letras en minÃºsculas (`es`, `spa`â€¦; se comparan normalizados con la misma lÃ³gica que ya usa `languageName`). `subtitle_language`: `""`, `"off"` (sin subtÃ­tulos por defecto) o cÃ³digo de idioma.
- API: `GET` y `PUT /api/v1/auth/sessions/{id}/profile`, solo sobre sesiones del propio usuario; `{id}` admite tambiÃ©n `current` (la sesiÃ³n que hace la peticiÃ³n). `PUT` sustituye el perfil entero. ValidaciÃ³n estricta, 400 con mensaje en castellano. Cada elemento de `GET /api/v1/auth/sessions` incluye `profile` (objeto con los tres campos).
- AplicaciÃ³n en `GET /api/v1/items/{id}/play` (y en el `/stream` que vuelve a decidir, si procede) con esta **precedencia**: parÃ¡metro explÃ­cito de la peticiÃ³n > preferencia por Ã­tem del usuario (`item preferences`) > perfil del dispositivo > valores por defecto actuales.
  - Audio: sin explÃ­cito ni preferencia de Ã­tem, se elige la primera pista cuyo idioma coincide con `audio_language`; si ninguna coincide, el comportamiento actual.
  - SubtÃ­tulo: sin explÃ­cito ni preferencia de Ã­tem: `off` â†’ ninguno; un idioma â†’ la primera pista **de texto** (no `burn_in`) que coincida, prefiriendo las no forzadas; nunca se elige por defecto una pista de imagen (obligarÃ­a a recodificar); si ninguna coincide, el comportamiento actual.
  - Calidad: `max_quality` es un **tope** solo para HLS (remux): limita las calidades ofrecidas y `effective height`; si el valor pedido o preferido (incluido `auto`) supera el tope, se usa el tope. La reproducciÃ³n directa no se toca (el tope no obliga a recodificar).
- Clientes: web (Mi cuenta â†’ lista de dispositivos: botÃ³n Â«AjustesÂ» por dispositivo con tres selectores) y apps Android/TV (Cuenta â†’ secciÃ³n Â«Este dispositivoÂ» con tres selectores simples). No hay perfiles por usuario ni administrador que los fuerce.

## Pasos

1. [x] **Servidor** (migraciÃ³n 0027, `internal/auth` o `internal/library` segÃºn encaje, `api/openapi.yaml`, tests). AgentRelay, esfuerzo alto. Terminado: tests de Go pasan; `apicontract` verde; pruebas que cubren precedencia, tope de calidad, idiomas, sesiÃ³n ajena (404), `current`.
2. [ ] **Web**. AgentRelay, esfuerzo medio. `web/src/lib/api.ts`, `SessionList.svelte` y la pÃ¡gina de cuenta; vitest.
3. [ ] **Android y Android TV**. AgentRelay, esfuerzo medio. Modelos de `core/api`, repositorio, `AccountScreen.kt` y viewmodel; tests unitarios.
4. [ ] **VersiÃ³n 0.59.0**, CHANGELOG es/en, TODO es/en, docs es/en (api, administraciÃ³n/guÃ­a de usuario, android, estado, hoja de ruta). AgentRelay bajo los docs; versiÃ³n y changelog los hace la sesiÃ³n.

Imprevistos de diseÃ±o: parar y anotarlos aquÃ­.

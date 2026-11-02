# Perfiles por dispositivo

Cada dispositivo (sesión) guarda ajustes propios que sirven de valor por defecto al reproducir: calidad máxima, idioma de audio preferido e idioma de subtítulo preferido. Decidido con el usuario: «se guarda por dispositivo».

Ejecución secuencial (árbol único). Tras cada paso: revisar diff, tests (`go test ./internal/...` con FFmpeg en PATH, `npm run check` y `npm test`, `gradlew.bat testDebugUnitTest assembleDebug`), commit por nombre de archivo, **sin push**. Versión 0.59.0 y CHANGELOG en el último paso.

## Diseño (decidido, no reabrir)

- Un dispositivo es una **sesión**: el perfil vive en una tabla nueva `session_profiles(session_id PK → sessions ON DELETE CASCADE, max_quality TEXT NOT NULL DEFAULT '', audio_language TEXT NOT NULL DEFAULT '', subtitle_language TEXT NOT NULL DEFAULT '')` (migración 0027). Si la sesión se cierra o se vuelve a iniciar, el perfil vuelve al valor vacío: es aceptable y se documenta.
- Valores: `max_quality` ∈ `""` (sin límite), `"480"`, `"720"`, `"1080"`. `audio_language`: `""` (sin preferencia) o código de idioma de 2 o 3 letras en minúsculas (`es`, `spa`…; se comparan normalizados con la misma lógica que ya usa `languageName`). `subtitle_language`: `""`, `"off"` (sin subtítulos por defecto) o código de idioma.
- API: `GET` y `PUT /api/v1/auth/sessions/{id}/profile`, solo sobre sesiones del propio usuario; `{id}` admite también `current` (la sesión que hace la petición). `PUT` sustituye el perfil entero. Validación estricta, 400 con mensaje en castellano. Cada elemento de `GET /api/v1/auth/sessions` incluye `profile` (objeto con los tres campos).
- Aplicación en `GET /api/v1/items/{id}/play` (y en el `/stream` que vuelve a decidir, si procede) con esta **precedencia**: parámetro explícito de la petición > preferencia por ítem del usuario (`item preferences`) > perfil del dispositivo > valores por defecto actuales.
  - Audio: sin explícito ni preferencia de ítem, se elige la primera pista cuyo idioma coincide con `audio_language`; si ninguna coincide, el comportamiento actual.
  - Subtítulo: sin explícito ni preferencia de ítem: `off` → ninguno; un idioma → la primera pista **de texto** (no `burn_in`) que coincida, prefiriendo las no forzadas; nunca se elige por defecto una pista de imagen (obligaría a recodificar); si ninguna coincide, el comportamiento actual.
  - Calidad: `max_quality` es un **tope** solo para HLS (remux): limita las calidades ofrecidas y `effective height`; si el valor pedido o preferido (incluido `auto`) supera el tope, se usa el tope. La reproducción directa no se toca (el tope no obliga a recodificar).
- Clientes: web (Mi cuenta → lista de dispositivos: botón «Ajustes» por dispositivo con tres selectores) y apps Android/TV (Cuenta → sección «Este dispositivo» con tres selectores simples). No hay perfiles por usuario ni administrador que los fuerce.

## Pasos

1. [x] **Servidor** (migración 0027, `internal/auth` o `internal/library` según encaje, `api/openapi.yaml`, tests). AgentRelay, esfuerzo alto. Terminado: tests de Go pasan; `apicontract` verde; pruebas que cubren precedencia, tope de calidad, idiomas, sesión ajena (404), `current`.
2. [x] **Web**. AgentRelay, esfuerzo medio. `web/src/lib/api.ts`, `SessionList.svelte` y la página de cuenta; vitest.
3. [ ] **Android y Android TV**. AgentRelay, esfuerzo medio. Modelos de `core/api`, repositorio, `AccountScreen.kt` y viewmodel; tests unitarios.
4. [ ] **Versión 0.59.0**, CHANGELOG es/en, TODO es/en, docs es/en (api, administración/guía de usuario, android, estado, hoja de ruta). AgentRelay bajo los docs; versión y changelog los hace la sesión.

Imprevistos de diseño: parar y anotarlos aquí.

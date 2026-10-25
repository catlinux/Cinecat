# Fase 2: cliente web (0.2.0)

Requiere el plan `.agents/plans/servidor-administracion/servidor-administracion.PLAN.md` cerrado primero (endpoints de administración, usuario admin, bibliotecas, escaneo).

Objetivo de la fase (`docs/es/hoja-de-ruta.md`): inicio de sesión, perfiles con PIN, biblioteca y fichas, reproductor con subtítulos/audio/capítulos, diseño propio claro/oscuro, guía de acceso remoto seguro.

Decisiones ya fijadas: **TypeScript + SvelteKit** + hls.js (ver `docs/es/arquitectura.md`), dirección visual **«Marquesina»** (cine de barrio 50-60, serif Fraunces, carteles ilustrados propios — ver sección «Dirección visual» del README y la maqueta enlazada), API REST ya documentada en `api/openapi.yaml`. Licencia GPL-3.0+atribución: cualquier librería de terceros que se añada (SvelteKit, hls.js, fuentes) debe verificarse y anotarse en `CREDITS.md` antes de usarse.

Cada paso es autocontenido. Al completarlo, el ejecutor lo marca `[x]` aquí y añade su entrada en `CHANGELOG.md`/`CHANGELOG.en.md` y `VERSION`, en el mismo commit. Commits: `tipo: descripción en castellano`.

## [x] Paso 1 — Esqueleto del proyecto SvelteKit

- **Archivos:** nuevo directorio `web/` en la raíz (`web/package.json`, `web/svelte.config.js`, `web/src/app.html`, `web/src/routes/+layout.svelte`, `web/.gitignore` para `node_modules`/`build`).
- **Qué hace:** inicializa un proyecto SvelteKit en TypeScript (adaptador estático o node, a decidir por el ejecutor según el despliegue previsto en `docs/es/arquitectura.md` — Debian/Docker, detrás de un proxy inverso; documentar la elección), con las fuentes de **Fraunces** e **Inter** servidas localmente (nunca cargadas desde Google Fonts en tiempo de ejecución, para cumplir «sin telemetría ni conexiones salientes» del README) y comprobadas sus licencias (ambas son OFL, verificar en el repositorio real antes de incluirlas, anotar en `CREDITS.md`). Variable de entorno o archivo de configuración para apuntar a la URL base del servidor Cinecat (nunca hardcodeada).
- **Criterio de terminado:** `npm run build` genera la app sin errores; `npm run dev` arranca y sirve una página con el texto «Cinecat» en la tipografía Fraunces.
- **Ejecutor:** AgentRelay, esfuerzo bajo.
- **Versión/CHANGELOG:** `0.2.0-dev.1` — Añadido: esqueleto del cliente web SvelteKit.
- **Hecho:** ejecución `20261025-153448-d856` (2026-10-25).

## [x] Paso 2 — Cliente de API y gestión de sesión

- **Archivos:** `web/src/lib/api.ts`, `web/src/lib/session.ts`, `web/src/routes/login/+page.svelte`.
- **Qué hace:** cliente TypeScript tipado contra `api/openapi.yaml` (generado o escrito a mano, a decidir por el ejecutor; si genera tipos, documentar la herramienta y su licencia) para `POST /auth/login`, `POST /auth/logout` y las llamadas autenticadas (cabecera `Authorization: Bearer`). El token de sesión se guarda en una cookie `httpOnly` gestionada por el propio SvelteKit (nunca en `localStorage`, para reducir el riesgo de robo de token por XSS) con el flag `Secure` cuando se sirve por HTTPS. Página de login con el estilo «Marquesina»: hero cálido, formulario simple, mensaje de error claro en credenciales incorrectas o bloqueo temporal (429).
- **Criterio de terminado:** test de componente o e2e (a elegir por el ejecutor, Vitest/Playwright, licencias ya permisivas MIT/Apache verificar igualmente) que simula login correcto e incorrecto contra un servidor de API simulado; prueba manual contra el servidor real documentada en el informe.
- **Ejecutor:** AgentRelay, esfuerzo medio.
- **Dependencias:** Paso 1, y el plan de administración del servidor cerrado (para tener un usuario real).
- **Versión/CHANGELOG:** `0.2.0-dev.2` — Añadido: inicio de sesión y gestión de sesión del cliente web.
- **Hecho:** ejecución `20261025-155339-3ad6` (2026-10-25).

## Paso 3 — Biblioteca y fichas

- **Archivos:** `web/src/routes/+page.svelte` (biblioteca/inicio), `web/src/routes/items/[id]/+page.svelte` (ficha), `web/src/lib/components/PosterCard.svelte`.
- **Qué hace:** pantalla de biblioteca con el layout ya prototipado en la maqueta «Marquesina» (hero de «continuar viendo», carruseles de carteles). Como `internal/metadata` (TMDB) todavía no está conectado al servidor, los carteles reales pueden no existir aún: diseña el `PosterCard` para mostrar el cartel si `media_items` llega a tener uno asociado, y un estado de respaldo con el título tipografiado sobre un fondo de color derivado del propio título (no una ilustración SVG fija como la maqueta, que era solo de referencia visual) cuando no hay cartel. Ficha de ítem: título, año, tipo, y botón «Reproducir» que llama a `GET /items/{id}/play` y redirige según el `mode` devuelto (`direct` o `remux`).
- **Criterio de terminado:** prueba e2e o manual documentada que, contra el servidor real con al menos un ítem en la base de datos, muestra la biblioteca y permite abrir una ficha.
- **Ejecutor:** AgentRelay, esfuerzo medio.
- **Dependencias:** Paso 2.
- **Versión/CHANGELOG:** `0.2.0-dev.3` — Añadido: pantalla de biblioteca y ficha de ítem.

## Paso 4 — Reproductor con hls.js

- **Archivos:** `web/src/lib/components/Player.svelte`, `web/src/routes/items/[id]/play/+page.svelte`.
- **Qué hace:** reproductor que usa el elemento `<video>` nativo para `mode: "direct"` y hls.js para `mode: "remux"` (comprobando antes `Hls.isSupported()`; en Safari/iOS, que soportan HLS nativo, usa el `<video>` directo con la URL de la playlist). Controles básicos: play/pausa, barra de progreso, volumen, pantalla completa. Guarda la posición de reproducción llamando a una actualización periódica (si no existe aún el endpoint para `playback_state`, el ejecutor debe señalarlo como incidencia y dejarlo para un paso posterior, sin bloquear el resto).
- **Criterio de terminado:** prueba manual documentada: un archivo `.mp4` se reproduce con el `<video>` nativo; si hay FFmpeg disponible en el entorno de prueba, un `.mkv` se reproduce vía hls.js (si no hay FFmpeg, señalarlo igual que se hizo en la Fase 1 y no bloquear el paso).
- **Ejecutor:** AgentRelay, esfuerzo medio.
- **Dependencias:** Paso 3.
- **Versión/CHANGELOG:** `0.2.0-dev.4` — Añadido: reproductor web con hls.js para remux y `<video>` nativo para directa.

## Paso 5 — Modo claro/oscuro y accesibilidad básica

- **Archivos:** `web/src/lib/theme.ts`, hojas de estilo ya creadas en los pasos anteriores.
- **Qué hace:** variante oscura de la paleta «Marquesina» (coherente con los tonos ya usados en la maqueta: granate/crema con dorado en claro, análogo más oscuro para el modo noche), conmutable y persistida en `localStorage` (uso legítimo de almacenamiento del navegador para preferencia de interfaz, no para datos de sesión). Revisión de contraste de texto (mínimo 4.5:1) y que los controles del reproductor sean operables por teclado.
- **Criterio de terminado:** capturas de pantalla en claro y oscuro documentadas en el informe (o adjuntas si el ejecutor puede generarlas); verificación manual de contraste con una herramienta accesible desde el ejecutor (o cálculo manual documentado si no hay herramienta disponible).
- **Ejecutor:** AgentRelay, esfuerzo bajo.
- **Dependencias:** Paso 3.
- **Versión/CHANGELOG:** `0.2.0` — cierra la Fase 2. Añadido: modo claro/oscuro y accesibilidad básica del cliente web.

## Fuera de alcance de esta fase (van después, según el roadmap)

Perfiles con PIN (necesita diseño de perfiles por usuario, no solo login — proponer en un plan aparte cuando llegue el momento), guía de acceso remoto seguro con HTTPS/proxy inverso (documentación, no código del cliente), subtítulos y selección de pista de audio en el reproductor (el servidor aún no los expone — Fase 3).

## Al cerrar

Parar y pedir el visto bueno del usuario antes de la Fase 3 (decisión de reproducción avanzada), como pide el prompt de arranque del proyecto.

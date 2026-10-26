# Plan: actualizaciones desde la propia app

Objetivo: en el panel de administración, un botón «Buscar actualizaciones» y, si hay una versión nueva en GitHub, un botón «Actualizar ahora». Sin tocar el contenedor a mano.

## Decisiones ya tomadas

- **Qué es una versión nueva:** una etiqueta `vX.Y.Z` del repositorio `catlinux/Cinecat` (público; se consulta `GET https://api.github.com/repos/catlinux/Cinecat/tags` sin credenciales) mayor que la versión en ejecución. Las etiquetas solo las crea el mantenedor para cambios importantes.
- **Notas de la versión:** sección de esa versión en `CHANGELOG.md` leída de `https://raw.githubusercontent.com/catlinux/Cinecat/<etiqueta>/CHANGELOG.md`.
- **Privacidad:** la consulta es opt-in (`system.check_updates`, apagado por defecto, ya existe) y solo se hace a GitHub. El botón manual funciona aunque el ajuste esté apagado (lo pulsa un administrador).
- **El contenedor no se actualiza a sí mismo.** Solo escribe una petición en un directorio compartido con el host (`./run`, bind mount); un servicio de systemd del host hace el trabajo. La app no tiene acceso a Docker.
- **Seguridad del host:** el actualizador solo acepta etiquetas con formato `vX.Y.Z` y que existan en `origin`; nunca ejecuta nada que venga de la petición.

## Pasos

### U1 — Servidor: comprobar y pedir la actualización (Go)

Ejecutor: AgentRelay, Sonnet, esfuerzo alto. Versión 0.14.0 (menor).

- Paquete `internal/update`: `Checker` (cliente HTTP con timeout 10 s, `User-Agent: cinecat/<versión>`, caché de 1 h salvo comprobación manual, comparación SemVer sin dependencias, ignora etiquetas que no sean `vX.Y.Z` estricto), lector de notas del CHANGELOG, y `Requester` que escribe `run/update-request.json` (`{"tag","requested_by","requested_at"}`, escritura atómica) y lee `run/update-status.json`.
- Configuración: variable `CINECAT_RUN_DIR` (por defecto `/run/cinecat-update`; si no existe el directorio, `apply` responde 503 «actualización automática no disponible en esta instalación»). Repositorio fijado en una constante, no configurable por la API.
- Endpoints de administrador (los demás usuarios 403): `GET /api/v1/system/update` → `{current, latest, available, checked_at, notes, apply_supported, status}`; `POST /api/v1/system/update/check` (fuerza la consulta; 502 si GitHub no responde); `POST /api/v1/system/update/apply` con `{tag}` (solo la `latest` conocida; 409 si ya hay una en curso).
- Si `system.check_updates` está activo, un comprobador en segundo plano consulta cada 6 h.
- Tests con servidor HTTP falso y directorio temporal; `api/openapi.yaml`.
- Terminado: `go vet`, `go test`, `gofmt` limpios.

### U2 — Host: servicio actualizador (Bash + systemd)

Ejecutor: AgentRelay, Sonnet, esfuerzo alto, revisión mía línea a línea. Versión 0.14.1 si va aparte, o dentro de 0.14.0.

- `docker-compose.yml`: montar `./run:/run/cinecat-update` en `servidor`; `run/` en `.gitignore`.
- `scripts/cinecat-updater.sh`, lanzado por `deploy/systemd/cinecat-update.service` (usuario `stark`, `Type=oneshot`) y disparado por `cinecat-update.path` (`PathExists=/opt/cinecat/run/update-request.json`). Pasos, escribiendo `update-status.json` (`phase`, `tag`, `message`, `started_at`, `finished_at`, `result`: `ok|failed|rolled_back`) en cada fase:
  1. validar la etiqueta (`^v[0-9]+\.[0-9]+\.[0-9]+$`) y que sea mayor que la actual;
  2. comprobar espacio libre (≥ 3 GB en la raíz de Docker y en el `TMPDIR` propio `$HOME/.cache/cinecat-tmp`);
  3. copia de la base de datos antes de tocar nada (`docker cp` de db, wal y shm → `sqlite3 VACUUM INTO ~/cinecat-backups/cinecat-<fecha>-pre-<versión>.db`), conservando solo las 10 últimas;
  4. `git fetch --tags` y `git merge --ff-only <etiqueta>`; construir y levantar con `COMMIT`/`BUILT_AT`;
  5. comprobación de salud: `GET /api/v1/system/version` debe devolver la versión nueva en 90 s;
  6. si falla cualquier paso desde el 4: volver al commit anterior, reconstruir, restaurar la copia de la base de datos y marcar `rolled_back`.
  - Borra `update-request.json` al empezar; un candado (`flock`) evita dos ejecuciones a la vez.
- `scripts/instalar-actualizador.sh` (idempotente): copia las unidades a `~/.config/systemd/user/` y las activa (`systemctl --user enable --now`, con `loginctl enable-linger`).
- `docs/es/actualizaciones.md` y `docs/en/updates.md`: cómo funciona, cómo instalarlo, cómo revertir a mano, dónde están las copias.
- Terminado: el script pasa `shellcheck` si está disponible; prueba en seco en el Debian con una etiqueta falsa (rechazada) y con la actual (sin cambios).

### U3 — Web: panel «Actualizaciones»

Ejecutor: AgentRelay, Sonnet, esfuerzo medio. Se publica en 0.14.0.

- Nueva sección en `/admin` (`admin/updates`): versión actual y comprobación, botón «Buscar actualizaciones», si hay nueva: versión, notas y «Actualizar ahora» con confirmación; durante la actualización, estado por fases (consulta periódica de `GET /system/update`, tolerando los cortes mientras el servidor se reinicia) y mensaje final (correcta, fallida o revertida). Interruptor de `system.check_updates` en la misma pantalla. Estilo «Marquesina» como el resto del panel.
- Todo por load y acciones de SvelteKit (cookie httpOnly); el sondeo del estado, por un endpoint `+server.ts`.
- Tests de load, acciones y componente; `npm run check`, `vitest` y **`npm run build`** (los tests no detectan exportaciones inválidas de `+page.server.ts`).

## Cierre

1. Revisar cada paso (diff + tests) y commit por paso.
2. Publicar una versión de prueba para ejercitar de verdad el ciclo: el Debian estará en 0.13.1 y se actualiza a 0.14.0 desde el botón. Esa etiqueta solo se sube con aprobación explícita del usuario, y la primera instalación del actualizador (U2) se hace a mano una vez.
3. Actualizar README, `TODO.md` y la documentación en `docs/es` y `docs/en`.

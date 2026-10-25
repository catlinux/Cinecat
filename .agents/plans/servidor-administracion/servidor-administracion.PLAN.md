# Administración mínima del servidor (previo a la Fase 2)

Hueco detectado al cerrar la Fase 1: el servidor no tiene forma de crear el primer usuario, dar de alta bibliotecas ni lanzar un escaneo, salvo editando la base de datos a mano. El cliente web (Fase 2) necesita estos endpoints para tener algo real con qué trabajar. Decidido con el usuario (2026-10-25): cerrar este hueco antes de empezar el cliente web, nunca haciendo que el cliente hable directo con la base de datos.

Decisiones ya fijadas: licencia GPL-3.0+atribución, servidor en Go, API REST versionada bajo `/api/v1/` con contrato OpenAPI (`api/openapi.yaml`), autenticación con `internal/auth` (argon2id + sesiones Bearer), base de datos en `internal/store`. Ver `CLAUDE.md` y `docs/es/arquitectura.md`.

Cada paso es autocontenido. Al completarlo, el ejecutor lo marca `[x]` aquí y añade su entrada en `CHANGELOG.md`/`CHANGELOG.en.md` y `VERSION`, en el mismo commit. Commits: `tipo: descripción en castellano`.

## Paso A — Alta del primer usuario administrador [x] hecho (ejecución 20261025-063041-6b23)

- **Archivos:** `internal/auth/http.go` o un nuevo `internal/auth/bootstrap.go`, `cmd/cinecatd/main.go`.
- **Qué hace:** si la tabla `users` está vacía al arrancar el servidor, genera un usuario `admin` con una contraseña aleatoria (nunca fija ni derivada de nada predecible) y la imprime **una sola vez** en el log de arranque con un aviso claro de que debe cambiarse; si ya hay usuarios, no hace nada. Alternativa a valorar por el ejecutor y documentar la elegida: un flag `--create-admin` que pide la contraseña por variable de entorno `CINECAT_ADMIN_PASSWORD` en el primer arranque, en vez de generarla sola. Decide una de las dos (o ambas) y justifícalo en el informe; no expongas esto como endpoint HTTP público sin autenticar.
- **Criterio de terminado:** test que arranca el servidor contra una base vacía y comprueba que aparece exactamente un usuario con rol `admin`; test que comprueba que un segundo arranque no crea un segundo usuario.
- **Ejecutor:** AgentRelay, esfuerzo medio (decisión de diseño pequeña incluida).
- **Versión/CHANGELOG:** parche — Añadido: alta del primer usuario administrador al arrancar el servidor.

## Paso B — Gestión de bibliotecas por API (solo admin) [x] hecho (ejecución 20261025-063520-7cae)

- **Archivos:** `internal/library/http.go`, `api/openapi.yaml`.
- **Qué hace:** añade `POST /api/v1/libraries` (crea una biblioteca: `name`, `path`; valida que el path exista y sea un directorio) y `DELETE /api/v1/libraries/{id}` (borra la biblioteca y, en cascada, sus `media_items`, ya previsto por el esquema). Ambos exigen sesión válida **y** rol `admin` (nuevo chequeo: hasta ahora `requireSession` no miraba el rol). `GET /api/v1/libraries` ya existe y no cambia.
- **Criterio de terminado:** tests de integración HTTP: crear biblioteca con admin → 201; crear biblioteca con usuario no admin → 403; crear biblioteca con un path que no existe → 400; borrar biblioteca existente → 204 y sus `media_items` desaparecen; borrar con usuario no admin → 403.
- **Ejecutor:** AgentRelay, esfuerzo medio.
- **Dependencias:** Paso A (para poder probar con un admin real).
- **Versión/CHANGELOG:** parche — Añadido: `POST`/`DELETE /api/v1/libraries`, con comprobación de rol admin.

## Paso C — Disparar un escaneo por API

- **Archivos:** `internal/library/http.go`, `api/openapi.yaml`, `cmd/cinecatd/main.go` (inyectar `library.Scanner` en `library.API`, como ya se hizo con `playback.Service`).
- **Qué hace:** añade `POST /api/v1/libraries/scan` (admin únicamente) que ejecuta `Scanner.Scan(ctx)` de forma síncrona y devuelve el `Result` (bibliotecas, archivos, advertencias) como JSON. Si tarda mucho con bibliotecas grandes es un problema conocido y aceptado para esta versión (documentarlo en TODO.md); la versión asíncrona con progreso queda para más adelante.
- **Criterio de terminado:** test de integración HTTP: con una biblioteca ya creada y una carpeta con un par de archivos de prueba, `POST /api/v1/libraries/scan` con admin devuelve 200 y el recuento esperado de archivos; con usuario no admin, 403.
- **Ejecutor:** AgentRelay, esfuerzo bajo.
- **Dependencias:** Pasos A y B.
- **Versión/CHANGELOG:** parche — Añadido: `POST /api/v1/libraries/scan`.

## Al cerrar

Con los pasos A-C hechos, el cliente web puede: iniciar sesión como admin recién creado, crear una biblioteca apuntando a una carpeta real, escanearla y listar/reproducir sus archivos — todo por API, sin tocar la base de datos a mano. Revisar que `docs/es/metadatos.md`/`arquitectura.md` sigan siendo coherentes (no deberían necesitar cambios). Marcar este plan como cerrado y pasar al de la Fase 2.

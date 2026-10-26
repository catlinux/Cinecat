# Plan: subir y borrar archivos de la biblioteca (con papelera)

Solo administradores. Hoy la biblioteca se monta de solo lectura (`${CINECAT_MEDIA_PATH}:/media:ro`): este plan exige montarla en lectura y escritura, y por eso se activa con un ajuste explícito (`library.allow_file_changes`, apagado por defecto) además del montaje.

## Decisiones

- **Papelera:** borrar nunca elimina: mueve el archivo (y sus subtítulos externos y carpeta de serie vacía) a `<raíz de la carpeta de la biblioteca>/.cinecat-papelera/<aaaammdd-hhmmss>-<nombre>`, en el mismo disco (renombrado instantáneo, sin copiar). Tabla `trash_items` (migración 0016): ruta original, ruta en papelera, biblioteca, usuario, fecha, instantánea de la ficha (título, año, tmdb_id, tipo) y estado. Retención por ajuste `trash.retention_days` (defecto 14, 1-90); un trabajo diario purga lo caducado (esto sí borra de verdad, con registro en el log).
- **Restaurar:** devuelve el archivo a su ruta original (si está ocupada, sufijo « (restaurado)»), lanza un escaneo de esa carpeta y recupera la ficha, y las preferencias y el visto se conservan porque la fila del ítem se marca `trashed_at` en vez de borrarse.
- **Escáner:** ignora `.cinecat-papelera` y trata los ítems con `trashed_at` como ausentes de los listados.
- **Subida:** el navegador no habla con la API con token (cookie httpOnly): sube por un endpoint de SvelteKit que reenvía en streaming a `PUT /api/v1/libraries/{id}/files?path=…`. Escribe en `<nombre>.part` y renombra al terminar; solo extensiones de vídeo y subtítulo conocidas; límite configurable; comprueba espacio libre antes. Tras subir, escaneo de esa biblioteca.
- **Seguridad:** toda ruta se resuelve contra las carpetas de la biblioteca y se rechazan `..`, enlaces simbólicos que salgan de ella y rutas absolutas; solo rol admin; el borrado pide confirmación escribiendo el título; cada operación se registra (usuario, ruta, hora).
- **Permisos en disco:** el contenedor corre como root y crearía archivos de root en tus discos. Solución: variables `CINECAT_UID`/`CINECAT_GID` en el compose (`user:`) con los del usuario/grupo `media` del Debian, y `chmod`/`chown` de los archivos subidos a 0664.
- **Límite conocido:** si el dominio pasa por un proxy con tope de tamaño de cuerpo (Cloudflare gratuito: 100 MB), las subidas grandes fallarán por ahí; se pueden hacer por la red local. Se documenta.

## Pasos

1. **FA — Papelera en el servidor (Go).** Migración 0016 (`trash_items`, `media_items.trashed_at`), `internal/library/trash.go`: mover a papelera, listar, restaurar, purgar caducados, ajustes `library.allow_file_changes`, `trash.retention_days`; endpoints admin `DELETE /api/v1/items/{id}` (a papelera), `GET /api/v1/trash`, `POST /api/v1/trash/{id}/restore`, `DELETE /api/v1/trash/{id}` (vaciar uno) y `POST /api/v1/trash/purge`. Tests con directorios temporales (confinamiento de rutas, enlaces simbólicos, colisiones, restauración, caducidad con reloj inyectado). Ejecutor: AgentRelay, esfuerzo alto. Versión menor.
2. **FB — Subida en el servidor (Go).** `PUT /api/v1/libraries/{id}/files`, confinamiento, `.part`, extensiones, espacio libre, límite, escaneo posterior. Tests. Ejecutor: AgentRelay, esfuerzo alto.
3. **FC — Interfaz web.** Ficha: botón «Mover a la papelera» (solo admin, con confirmación). Administración → «Papelera»: lista con fecha de borrado y días restantes, Restaurar y Eliminar definitivamente. Administración → Bibliotecas: «Subir archivos» (elegir carpeta con el navegador de carpetas existente, varios archivos, barra de progreso). Ajustes en Sistema. Ejecutor: AgentRelay, esfuerzo medio-alto.
4. **FD — Despliegue.** Compose con montaje rw y `user:`, documentación `docs/es/archivos.md` y `docs/en/files.md`, aviso de seguridad. Lo aplico yo en el Debian con aprobación del usuario.
   Además: el contenedor web debe llevar `BODY_SIZE_LIMIT` alto (por defecto adapter-node rechaza cuerpos de más de 512 KB). El Apache de delante (`video.warcrafted.com-le-ssl.conf`) no limita el cuerpo (`LimitRequestBody` ausente) y su `Timeout 300` no corta una subida en curso; el dominio no pasa por Cloudflare (comprobado el 2026-10-26).

Orden: después de T17b, U2 y U3. Versión 0.16.0.

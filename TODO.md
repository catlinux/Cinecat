# TODO

[English version](TODO.en.md)

Prioridades: 🔴 bloqueante · 🟠 próxima fase · 🟢 más adelante.

## Antes de escribir código
- 🟠 Activar en GitHub los avisos de seguridad privados (private vulnerability reporting).
- 🔴 El bloqueo por intentos fallidos de login (`internal/auth`) se indexa por nombre de usuario, no por IP: cualquiera puede bloquear la cuenta de otro usuario legítimo enviando 5 logins fallidos con su `username`. Revisar antes de exponer el servidor a internet (por ejemplo, combinando usuario+IP o añadiendo límite por IP).
- 🟠 No hay todavía forma de crear usuarios (ni endpoint ni comando); para probar el login hay que insertar el usuario y su hash argon2id directamente en la base de datos.
- 🟠 No hay todavía endpoints ni CLI para dar de alta bibliotecas (`libraries`) ni para lanzar el escaneo (`internal/library`); el paquete queda listo para conectarlo en el Paso 6 (API) o antes.
- 🟠 El cliente de metadatos (`internal/metadata`) solo busca películas (`/search/movie`); falta el endpoint de series/episodios de TMDB para identificar `media_items.kind = 'series'/'episode'`. Tampoco está conectado todavía al escaneo ni al servidor.
- El TTL de caché de TMDB (7 días, `internal/metadata`) es un valor de partida sin verificar contra los términos vigentes de la API: revisar antes de producción.

## Riesgos conocidos, aceptados
- Existe una app Android inactiva desde 2015 llamada «CineCat» (F-Droid, catálogo de cine catalán con datos de la Generalitat; sin registro de marca). Mismo nombre, ámbito distinto. Se mantiene el nombre Cinecat; riesgo de confusión menor, sin acción pendiente.

## Producto
- 🟠 Servidor mínimo: biblioteca, identificación de películas y series, metadatos en `es-ES`.
- 🟠 Motor de decisión de reproducción (directa / remux / transcodificación).
- 🟠 Cliente web con inicio de sesión.
- 🟠 Clientes Android y Android TV.
- 🟢 Fotos y presentaciones.
- 🟢 Contenido de internet y televisión en directo (solo si se decide).
- 🟢 Segmentos de intro/créditos, descargas sin conexión, sincronizar reproducción entre dispositivos.
- 🟢 Otras plataformas (Apple, televisores Samsung/LG).

## Documentación
- 🟠 Guía de usuario y glosario «para tontos» cuando exista algo que probar.
- 🟠 Código de conducta.
- 🟢 Capturas de pantalla y guía de despliegue con Docker.

## Mantenimiento de documentos
- Mantener sincronizadas las versiones en español e inglés de cada documento.

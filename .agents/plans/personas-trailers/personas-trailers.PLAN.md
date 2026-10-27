# Plan: puntuación en las carátulas, páginas de persona y tráileres

Pedido el 2026-10-27. Todo sale de TMDB (la fuente de metadatos ya configurada); no hay conexiones nuevas a otros servicios. El tráiler es un enlace a YouTube que abre el usuario: no se incrusta ni se descarga nada.

## Decisiones

- **Puntuación:** `rating` ya existe en `media_items` y `series` y llega a la web. Insignia discreta en una esquina de la carátula (película y serie), con un decimal (`7,4`), solo si hay valor; no en clips ni en música.
- **Personas:** hoy los créditos se guardan como JSON sin el id de TMDB de la persona. Se añade `id` (TMDB) a cada crédito de reparto y equipo. La migración 0019 vacía `credits` de las fichas de TMDB (como hizo la 0013) para que el siguiente escaneo las vuelva a pedir con ids y tráiler. Las fichas sin id siguen funcionando sin enlace.
- **Página de persona** `/people/{tmdbId}`: nombre, foto, fechas y biografía (TMDB `/person/{id}` en el idioma configurado, con `append_to_response=combined_credits`, cacheado como el resto). Dos secciones separadas: **En tu biblioteca** (películas y series cuyos créditos contienen ese id, con su papel o puesto) y **No la tienes** (resto de `combined_credits`, sin las que ya están por `tmdb_id`, ordenadas por fecha descendente, sin enlace de reproducción, con carátula de TMDB).
- **Tráiler:** `append_to_response` añade `videos` con `include_video_language=es,en,null`. Elección: YouTube, tipo Trailer en castellano (oficial primero), luego Teaser en castellano, luego Trailer en inglés. Se guarda `trailer_url` y `trailer_language` en `media_items` y `series`. En la ficha, botón «Ver tráiler» (y «(en inglés)» si no hay en castellano) que abre YouTube en otra pestaña.

- **Subidas guiadas (pedido después, va antes de PT2):** el usuario no quiere elegir carpetas a mano. Botones discretos solo para administradores: «Crear nueva serie» en `/series`, «Nueva temporada» en la ficha de la serie, «Subir capítulo» en cada temporada y «Subir película» junto a «Sin identificar» en `/movies`. El servidor decide carpeta y nombre: una serie es una carpeta de primer nivel de una raíz de biblioteca de vídeo, y los capítulos se guardan como `<carpeta de la serie> S01E05.ext` (en la carpeta de la temporada si la serie ya usa carpetas por temporada; si no, en la de la serie). La web detecta temporada y episodio en el nombre del archivo (`S01E05`, `1x05`, número suelto) y deja corregirlos antes de subir.

## Pasos

1. **R — Puntuación en las carátulas (web).** `PosterCard.svelte`, `SeriesCard.svelte` y sus tests. AgentRelay, esfuerzo medio. Parche.
2. **PT1 — Servidor (Go).** Migración `0019_people_trailers.sql`; ids de persona en créditos; vídeos en `MovieDetails`/`TVDetails` y elección del tráiler; guardar en escaneo y en corrección manual; DTO de ítem y serie con `trailer_url`, `trailer_language` y `id` en cada crédito; `GET /api/v1/people/{id}` (sesión) con `person`, `in_library` y `not_in_library`. Tests con servidor TMDB falso. AgentRelay, esfuerzo alto.
3. **SG1 — Subidas guiadas en el servidor (Go).** `GET /api/v1/uploads/targets` y `PUT /api/v1/series/{id}/files?season=&episode=&name=`. AgentRelay, esfuerzo alto.
4. **SG2 — Subidas guiadas en la web (series).** Componente de cola de subida con temporada/episodio por archivo; botones en `/series` y `/series/{id}`. AgentRelay, esfuerzo alto.
5. **SG3 — «Subir película» en `/movies`.** AgentRelay, esfuerzo medio.
6. **PT2 — Web.** Créditos enlazados a `/people/{id}`; página de persona con las dos secciones; botón de tráiler en las fichas de película y serie. AgentRelay, esfuerzo alto. Cierra versión menor.

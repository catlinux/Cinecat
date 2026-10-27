# User guide

[Español](../es/guia-de-usuario.md) · [Back to README](../../README.en.md) · [Project status](status.md)

This guide explains how to use the Cinecat web app. It is meant for someone who just wants to watch their library; administration is covered in [administration](administration.md).

## Signing in and out

1. Open your Cinecat address (for example, `https://video.warcrafted.com`).
2. On the "Entrar" screen, enter your **Usuario** and **Contraseña** and press "Entrar en la sala".
3. If the details are correct, you reach the portal. After several failures the server may temporarily block the attempt: wait a few minutes and try again.

To sign out, open `/login` and press "Cerrar sesión". The session is stored in an `httpOnly` cookie; the browser never keeps the token in `localStorage`.

## The portal

The portal is the home page. At the top you see the server name, tagline and welcome text configured by the administrator. Then come three entry points: **Películas**, **Colecciones** and **Series**, with the number of titles in each.

Below, only when they have content:

- **Continuar viendo**: what you left unfinished.
- **Siguiente episodio**: the next episode of your series.
- **Añadido recientemente**: the newest movies in the library.

Rows can be scrolled with the `‹` and `›` arrows; you can also swipe with a finger or trackpad.

## Browsing the library

The menu can be on the left (**lateral**) or at the top (**superior**), depending on the administrator's setting. It contains **Inicio**, **Películas**, **Colecciones**, **Series**, **Mi cuenta** and, for administrators, **Administración**.

### Movies

The "Películas" screen shows the paginated movie list. You can:

- search across **Todo**, **Título**, **Reparto** or **Dirección**;
- filter by **Género**;
- choose the **Orden**: Título, Año or Recientes;
- filter by **Visionado**: Todas, Sin ver or Vistas;
- press a letter (`#` or `A`–`Z`) to filter by initial;
- remove a filter with its `×` or with "Limpiar filtros".

The number of results per page is **columns × rows** as configured in administration (6 × 8 = 48 by default). Pagination appears when there is more than one page.

Administrators also see "Sin identificar (N)", which shows movies the scanner could not identify.

### Collections

"Colecciones" groups sagas and cycles. You can search by name and sort by **Nombre**, **Añadidas** or **Tamaño**. Opening a collection shows its description, its movies and a button to play the first unwatched one (or the first one, if all are watched).

Administrators see "Gestionar colección" inside a collection to change its name, description, poster or backdrop, add or remove movies, reorder them and delete the collection.

### Series

"Series" shows the paginated series list. You can search, filter by genre, choose the order and filter by letter. Opening a series shows its page and the season selector; each season shows its episodes with title, synopsis, image and runtime.

The watched buttons mark an episode, a season or the whole series as watched. The main series button changes with your progress: "Empezar", "Continuar" or "Volver a ver". Administrators also see **Cambiar imagen** and **Corregir ficha** at the end of the series page.

## Movie detail page

The page shows the poster, title (and original title if different), year, runtime, genres, rating, tagline, synopsis, cast and crew, IMDb/TMDB links and the collections it belongs to. "Más como este" is at the end.

- **Reproducir** starts from the beginning; **Continuar** resumes where you left off. **Desde el principio** ignores saved progress.
- **Marcar como vista** / **Desmarcar** changes the watched state and clears progress.
- The **Reproducción** panel lets you choose **Audio**, **Subtítulos** and **Calidad**. The choice is saved for that movie and user.
- The **Archivo** panel shows technical details and how it will be played.
- Administrators additionally see **Cambiar imagen** and **Corregir ficha** at the end.

## Episode detail page

The episode page shows the breadcrumb `Series › Serie › Temporada N › Episodio N`, the title, synopsis, runtime, cast and file details. You can mark the episode as watched, continue where you left off and go to the **Episodio anterior** or **Siguiente episodio**.

## Playback

The player has custom controls. These are the keyboard shortcuts:

| Key | Action |
|---|---|
| `Space` or `K` | Play or pause |
| `←` or `J` | Rewind 10 s (`Shift`: 30 s) |
| `→` or `L` | Forward 10 s (`Shift`: 30 s) |
| `↑` / `↓` | Volume up or down |
| `M` | Mute or unmute |
| `F` | Fullscreen |
| `C` | Enable or disable subtitles |
| `,` / `.` | Previous / next frame, paused only |
| `0`–`9` | Jump to 0 %, 10 %, …, 90 % |
| `Esc` | Close menus or cancel the countdown |

The visible buttons let you play/pause, rewind or forward 10 s, change volume, choose speed (0.5× to 2×), audio, subtitles and quality, enable picture-in-picture, fullscreen and play the next episode. Clicking the video plays or pauses; double-clicking toggles fullscreen.

Progress is saved every 15 s and on exit. If 15 s or less remain, playback does not resume: it is treated as finished. When an episode ends and there is a next one, an 8-second countdown starts.

## Themes, menu and account

The theme selector offers **Auto**, **Claro** and **Oscuro**. Auto follows the system theme. The preference is stored in that browser; if there is none, the server default is used.

In **Mi cuenta** you can change your password: enter the current one, the new one and the confirmation; the new one must be at least 8 characters. The current session on that device stays active.

## Glossary

- **Biblioteca:** a server folder that Cinecat scans for videos.
- **Escaneo:** the process that finds files, parses their names and fills in metadata.
- **Ficha:** the page with the data for a movie, series or episode.
- **Identificar:** find the correct TMDB (or OMDb) record.
- **Transcodificación:** re-encode video or audio so the player can understand it.
- **HLS:** segmented playback format used by the web player when the original file cannot be served as is.
- **Reproducción directa:** send the original file without re-encoding.
- **WebVTT:** subtitle format the browser can display.

## If something does not work

- If the page returns to "Entrar", the session expired: sign in again.
- If a movie is missing, ask an administrator to check that the library exists and the scan has finished.
- If a video does not start, try another quality or tell the administrator: the server may lack FFmpeg or the file may use an unsupported format.
- If you see the version mismatch notice, reload the page; if it persists, the administrator needs to update the web app or the server.

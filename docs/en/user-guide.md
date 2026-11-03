# User guide

[Español](../es/guia-de-usuario.md) · [Back to README](../../README.en.md) · [Project status](status.md)

This guide explains how to use the Cinecat web app. It is meant for someone who just wants to watch their library; administration is covered in [administration](administration.md).

## Signing in and out

1. Open your Cinecat address (for example, `https://media.techmarc.es`).
2. On the "Entrar" screen, enter your **Usuario** and **Contraseña** and press "Entrar en la sala".
3. If the details are correct, you reach the portal. After several failures the server may temporarily block the attempt: wait a few minutes and try again.

To sign out, open `/login` and press "Cerrar sesión". The session is stored in an `httpOnly` cookie; the browser never keeps the token in `localStorage`.

## The portal

The portal is the home page. At the top you see the server name, tagline and welcome text configured by the administrator. Then come the entry points for **Películas**, **Colecciones**, **Series** and, when Vídeos or Música libraries have content, **Vídeos** and **Música**. Movies, Collections and Series are hidden when your account has no content in them; if the request fails, all of them are shown.

Below, only when they have content:

- **Continuar viendo**: what you left unfinished.
- **Siguiente episodio**: the next episode of your series.
- **Añadido recientemente**: the newest movies in the library.

Rows can be scrolled with the `‹` and `›` arrows; you can also swipe with a finger or trackpad. Under **Mi cuenta → Portal de inicio** you can hide the rows you do not want to see; the choice is yours alone and does not change other accounts' portal.

## Browsing the library

The menu can be on the left (**lateral**) or at the top (**superior**), depending on the administrator's setting. It contains **Inicio**, **Películas**, **Colecciones**, **Series**, **Vídeos** and **Música** when your account has access to a library of those types, **Fotos** when there is at least one visible photo library, **Historial**, **Mi cuenta** and, for administrators, **Administración**.

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

Administrators see "Gestionar colección" inside a collection to change its name, description, poster or backdrop, add or remove movies, reorder them and delete the collection. It also includes **Identificación en TMDB**: you can search for the collection by name, link it to a candidate or unlink a manual collection. When it is identified, the page shows the films in the saga that are missing from your visible library, with their poster and year.

### Series

"Series" shows the paginated series list. You can search, filter by genre, choose the order and filter by letter. Opening a series shows its page and the season selector; each season shows its episodes with title, synopsis, image and runtime.

The watched buttons mark an episode, a season or the whole series as watched. The main series button changes with your progress: "Empezar", "Continuar" or "Volver a ver". Administrators also see **Cambiar imagen** and **Corregir ficha** at the end of the series page.

### Vídeos

The **Vídeos** section appears in the menu when a Vídeos library has content. To create one, open **Administración → Bibliotecas**, choose the "Vídeos (videoclips, conciertos…)" type, add its folders and run a scan.

This type does not identify files or download posters from TMDB or OMDb: the title comes from the file name. Opening the section shows the library folders with the number of videos they contain (including subfolders). Inside a folder, subfolders come first and then the videos as landscape cards. Thumbnails are generated the first time they are viewed and stored in a cache; if no thumbnail exists yet, a generated fallback cover is used. The breadcrumbs let you go back to any level. You can play or continue a video, mark it as watched and change audio, subtitles or quality.

### Music

The **Música** section appears when a library of that type has tracks. To create one, open **Administración → Bibliotecas**, choose the "Música" type, add its folders and run a scan. The recommended structure is `Artist/Album/Track`: the server reads title, artist, album artist, album, track, disc, year and genre with `ffprobe`. When tags are missing it uses the file and folder names. MP3, FLAC, M4A, AAC, OGG, OGA, OPUS and WAV are supported; WMA, APE and other formats are not indexed. The cover comes from `cover`, `folder`, `front` or `album` in the album folder or, if there is none, from the embedded image of the first track.

In **Música** you can switch between **Álbumes**, **Artistas** and **Pistas**, search by text, filter by letter and sort albums by title, artist, year or recently added. The tracks view only shows results after you type a search. Opening an album shows its cover, year, track count, total duration and track list; **Reproducir**, **Aleatorio** or **Añadir a la cola** start listening. On an artist page, **Reproducir todo** queues its albums in order.

The music player is a fixed bar at the bottom that keeps playing while you browse the site. It shows the cover, title and artist, and lets you pause, skip, seek, change or mute the volume, enable shuffle and repeat (off, all or one), and open the **Cola** to remove tracks or clear it. The queue, volume, shuffle and repeat settings are remembered after a reload; playback is restored paused. If a track fails or its link has expired, it is retried once and then skipped with a notice. Playing a video pauses the music automatically, and it does not resume on its own. There are no new keyboard shortcuts for the bar: use its buttons, the system media controls and the playback buttons on each track.

### Photos

The **Fotos** section appears in the menu when your account has at least one visible Photos library. To create one, open **Administración → Bibliotecas**, choose the "Fotos" type, add its folders and run a scan.

If there is only one photo library, the section opens its root directly; if there are several, it shows a list with their names and photo counts. Inside a library, the **breadcrumbs** let you go back to any level. Subfolders come first, followed by the photos in a paginated grid; you can change the **Orden** between Fecha and Nombre. Thumbnails load as you approach them.

Press a photo to open the full-screen viewer. The `←` and `→` arrows change photo and `Escape` closes the viewer; you can also use the **Anterior** and **Siguiente** buttons. The viewer shows the counter and the name and capture date when available, and **Presentación automática** advances every 5 seconds; **Abrir original en otra pestaña** shows the original file.

## Favorites and watchlist

Each user has their own **Favorites** and **Watchlist**. You can mark movies, videos, series and collections from their pages with the **Favorito** and **Próximas a ver** buttons, or from the heart on their cards. On cards the heart appears on hover, on keyboard focus, or always on touch screens.

Both lists live in the menu, below the categories. **Favoritos** keeps what you saved; **Próximas a ver** is your watch-later list. Each page separates Movies, Videos, Series and Collections and lets you page through each block. The portal shows the **Próximas a ver** and **Tus favoritos** rows when they have content.

Lists are not shared between users. Marking a movie or video as watched removes it from **Próximas a ver** automatically (not from Favorites); finishing a whole series also removes it from that list. Marking it unwatched does not add it back: you have to add it again manually if you want it.

## History

**Historial** gathers what you last played or marked as watched, with the most recent date. Each entry shows the poster, title (and series, season and episode for an episode), progress or the **Visto** mark, and a link to its page. **Quitar del historial** removes your progress and watched mark for that item; other accounts are unaffected. Only content from libraries you can access appears. If you have not watched anything yet, the page says so.

## People, ratings and trailers

On movie and series posters, a badge in the top right corner shows the TMDB rating (for example, "7,4"). An icon in the bottom left corner marks that the movie is in a collection; in Movies you can filter by **In a collection** or **No collection** to find the ones that were not grouped automatically.

On the detail page, cast and crew are links: clicking a person opens their page with **En tu biblioteca** (their movies and series you have, with their role) and, separately, **No la tienes** (the rest of their filmography, informational only). If the entry has a trailer, **Ver tráiler** appears and opens YouTube in another tab: in Spanish when it exists, otherwise in English. Entries from before this version have neither links nor trailer until the scan requests them again from TMDB.

## Uploading content (administrators)

With uploads enabled (see [administration](administration.md#uploading-files-and-trash)), administrators see discreet buttons: **Crear nueva serie** in Series, **Nueva temporada** on a series page, **Subir capítulo** on each season and **Subir película** in Movies. You do not choose folders: Cinecat stores everything in the right place and names episodes like `Series S01E05`. The web app detects season and episode in the file name and you can correct them before uploading. From a movie page, **Gestionar colecciones** lets you add it to a collection, create a new one with it or remove it. In the **Archivo** panel of a movie or episode, administrators can upload external `.srt`, `.vtt` or `.ass` subtitles up to 2 MB, set a language and label, and delete them; this does not require enabling general uploads, and they appear in playback with a `u<N>` identifier.

## Movie detail page

The page shows the poster, title (and original title if different), year, runtime, genres, rating, tagline, synopsis, cast and crew, IMDb/TMDB links and the collections it belongs to. "Más como este" is at the end.

- **Reproducir** starts from the beginning; **Continuar** resumes where you left off. **Desde el principio** ignores saved progress.
- **Marcar como vista** / **Desmarcar** changes the watched state and clears progress.
- The **Reproducción** panel lets you choose **Audio**, **Subtítulos** and **Calidad**. The choice is saved for that movie and user. If you choose a non-default audio track on an MP4 that would otherwise play directly, Cinecat serves it through HLS to honour it.
- The **Archivo** panel shows technical details and how it will be played.
- Administrators additionally see **Cambiar imagen** and **Corregir ficha** at the end.

## Episode detail page

The episode page shows the breadcrumb `Series › Serie › Temporada N › Episodio N`, the title, synopsis, runtime, cast and file details. You can mark the episode as watched, continue where you left off or start from the beginning, and go to the **Episodio anterior** or **Siguiente episodio**.

## Video detail page

The page for a video in the Vídeos section is minimal: title, **Volver a Vídeos**, play or continue, mark as watched, playback options and the file panel. It does not show synopsis, cast, collections, IMDb/TMDB links, "Más como este", "Corregir ficha" or "Cambiar imagen", because these videos are not identified.

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

In the subtitle menu, embedded image tracks (PGS, VobSub and DVB) carry the " · imagen" mark. Choosing one reloads playback at the current second with the subtitle burned in; removing it or switching to text from one reloads without the burn.

Progress is saved every 15 s and on exit. If 15 s or less remain, playback does not resume: it is treated as finished. When an episode ends and there is a next one, an 8-second countdown starts.

## Themes, menu and account

The theme selector offers **Auto**, **Claro** and **Oscuro**. Auto follows the system theme. The preference is stored in that browser; if there is none, the server default is used.

In **Mi cuenta → Change my password** you can change your password: enter the current one, the new one and the confirmation; the new one must be at least 8 characters. The current session on that device stays active. Under **Mi cuenta → Portal de inicio** you choose which rows to hide in your portal. As a user you only see the libraries assigned to you by the administrator.

Under **Mi cuenta → Open sessions**, each device in the list has **Ajustes**. There you can set its maximum quality (no limit, 480p, 720p or 1080p), preferred audio language (no preference or a language) and default subtitles (no preference, disabled or a language). They apply when playing on that device unless you have chosen something else for the movie; the profile subtitle never picks an image track, and the quality cap only affects HLS, not direct playback. The profile belongs to the session: closing it or pairing the device again discards it and leaves it empty.

## Glossary

- **Biblioteca:** a server folder that Cinecat scans for content.
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

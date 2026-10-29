# Créditos / Credits

## Español

Cinecat se apoya en el trabajo de otras personas y proyectos. Aquí se listan **todos** los que han servido de inspiración o de los que se reutilice código, datos o recursos. Esta lista se mantiene al día y se mostrará también en la pantalla «Acerca de» de las apps.

**Estado a 0.2.0-dev.1:** no se ha copiado código de otros proyectos. Se usan las dependencias y tipografías anotadas en la tabla; de las fuentes se incluyen los archivos OFL junto a los TTF en `web/static/fonts/`. Cuando se reutilice código se anotará aquí el origen, la versión o commit, la licencia y los archivos afectados, y se conservarán sus avisos de copyright.

## English

Cinecat builds on the work of other people and projects. This list includes **all** projects that have inspired it or whose code, data or resources are reused. It is kept up to date and will also appear on the "About" screen of the apps.

**Status at 0.2.0-dev.1:** no code has been copied from other projects. The dependencies and typefaces listed in the table are in use; the OFL files are shipped next to the TTF files in `web/static/fonts/`. When code is reused, the origin, version or commit, license and affected files will be recorded here, and their copyright notices preserved.

## Tabla / Table

| Proyecto / Project | Uso previsto / Planned use | Licencia / License (verificar / to verify) |
|---|---|---|
| [Jellyfin](https://jellyfin.org) | Inspiración: modelo de API, perfiles de dispositivo, decisiones de reproducción / Inspiration: API model, device profiles, playback decisions | GPL-2.0 |
| [Jellyfin Android TV](https://github.com/jellyfin/jellyfin-androidtv) | Referencia de navegación con mando y perfil de reproducción directa / Remote navigation and direct-play profile reference | GPL-2.0 |
| [Wholphin](https://github.com/damontecres/Wholphin) | Referencia de interfaz para Android TV, ExoPlayer + MPV, perfiles con PIN / Android TV UI reference, ExoPlayer + MPV, PIN profiles | GPL-2.0 |
| [Findroid](https://github.com/jarnedemeulemeester/findroid) | Referencia: Compose, ExoPlayer + mpv, descargas, capítulos / Reference: Compose, ExoPlayer + mpv, downloads, chapters | GPL-3.0 |
| [Streamyfin](https://github.com/streamyfin/streamyfin) | Candidato a reutilización de código / Candidate for code reuse | MPL-2.0 |
| [Media3 / ExoPlayer](https://github.com/androidx/media) | Reproductor de Android / Android player | Apache-2.0 |
| [Jetpack Compose](https://developer.android.com/compose) BOM 2026.06.00 (UI 1.11.3, Material3 1.4.0) | Interfaz de la app Android / Android app UI | Apache-2.0 |
| [AndroidX Activity](https://developer.android.com/jetpack/androidx/releases/activity) 1.10.1 | Actividad Compose de la app Android / Android app Compose activity | Apache-2.0 |
| [AndroidX Core](https://developer.android.com/jetpack/androidx/releases/core) 1.16.0 | Utilidades base de la app Android / Android app core utilities | Apache-2.0 |
| [AndroidX Lifecycle](https://developer.android.com/jetpack/androidx/releases/lifecycle) 2.10.0 | Ciclo de vida de la app Android / Android app lifecycle | Apache-2.0 |
| [androidx.tv:tv-material](https://developer.android.com/jetpack/androidx/releases/tv) 1.1.0 | Componentes de interfaz para Android TV / Android TV UI components | Apache-2.0 |
| [Kotlin](https://kotlinlang.org) 2.4.21 | Lenguaje y compilador de la app Android / Android app language and compiler | Apache-2.0 |
| [OkHttp](https://square.github.io/okhttp/) 5.3.2 | Cliente HTTP y DNS de dirección local de la app Android / Android app HTTP client and local-address DNS | Apache-2.0 |
| [Retrofit](https://square.github.io/retrofit/) 3.0.0 | Cliente de la API de la app Android / Android app API client | Apache-2.0 |
| [kotlinx.serialization](https://github.com/Kotlin/kotlinx.serialization) 1.12.0 | JSON de la app Android / Android app JSON | Apache-2.0 |
| [kotlinx.coroutines](https://github.com/Kotlin/kotlinx.coroutines) 1.11.0 | Concurrencia de la app Android / Android app concurrency | Apache-2.0 |
| [AndroidX DataStore](https://developer.android.com/jetpack/androidx/releases/datastore) 1.3.0 | Ajustes y sesión cifrada de la app Android / Android app settings and encrypted session | Apache-2.0 |
| [Coil](https://github.com/coil-kt/coil) 3.4.0 | Carga de carteles e imágenes en la app Android / Poster and image loading in the Android app | Apache-2.0 |
| [AndroidX Navigation](https://developer.android.com/jetpack/androidx/releases/navigation) 2.9.8 | Navegación de la app Android / Android app navigation | Apache-2.0 |
| [ZXing core](https://github.com/zxing/zxing) 3.5.4 | Código QR de vinculación en la app Android / Pairing QR code in the Android app | Apache-2.0 |
| [Android Gradle Plugin](https://developer.android.com/build) 9.5.0 | Herramienta de compilación de la app Android / Android app build tool | Apache-2.0 |
| [Gradle](https://gradle.org) 9.8.2 | Herramienta de compilación de la app Android / Android app build tool | Apache-2.0 |
| [JUnit](https://junit.org/junit4/) 4.13.2 | Pruebas unitarias JVM de la app Android / Android app JVM unit tests | EPL-1.0 |
| [mpv](https://mpv.io) / libmpv | Reproductor alternativo en Android / Alternative Android player | por verificar según compilación / to verify per build |
| [hls.js](https://github.com/video-dev/hls.js) 1.7.3 | Reproducción HLS en el reproductor web (en uso) / HLS playback in the web player (in use) | Apache-2.0 |
| [Shaka Player](https://github.com/shaka-project/shaka-player) | Alternativa para navegador / Browser alternative | Apache-2.0 |
| [Vidstack](https://vidstack.io) | Interfaz del reproductor web / Web player UI | MIT |
| [FFmpeg](https://ffmpeg.org) | Herramienta externa de conversión (no se distribuye en el repositorio) / External conversion tool (not distributed in the repository) | LGPL/GPL según compilación / depending on build |
| [parse-torrent-title](https://pypi.org/project/parse-torrent-title/) | Identificar títulos a partir del nombre de archivo / Identify titles from filenames | MIT |
| [hunch](https://docs.rs/crate/hunch/latest) | Alternativa en Rust para el mismo fin / Rust alternative for the same purpose | MIT |
| [guessit](https://github.com/guessit-io/guessit) | Alternativa para identificar nombres / Alternative for name parsing | LGPL-3.0 |
| [Intro Skipper](https://github.com/intro-skipper) | Idea: detección de intros por audio / Idea: audio-based intro detection | GPL-3.0 |
| [TMDB](https://www.themoviedb.org) | Metadatos e imágenes (cliente preparado, sin conectar) / Metadata and images (client prepared, not wired) | Términos de uso de la API / API terms of use |
| [OMDb API](https://www.omdbapi.com) | Metadatos de películas: sinopsis y URL de cartel / Movie metadata: plot and poster URL | Términos de uso de la API; datos derivados de IMDb, restringidos a uso personal y no comercial / API terms of use; data derived from IMDb, restricted to personal, non-commercial use |
| [TheTVDB](https://thetvdb.com) | Metadatos de series (complemento) / Series metadata (supplement) | Términos de la API / API terms |
| [Fanart.tv](https://fanart.tv) | Arte adicional / Extra artwork | Términos de la API / API terms |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | Driver SQLite en Go puro usado por el servidor / Pure-Go SQLite driver used by the server | BSD-3-Clause |
| [golang.org/x/crypto](https://github.com/golang/crypto) | Hash de contraseñas con argon2id / Password hashing with argon2id | BSD-3-Clause |
| [Fraunces](https://github.com/undercasetype/Fraunces) variable | Tipografía de títulos del cliente web, empaquetada en `web/static/fonts/Fraunces-Variable.ttf` / Web client display typeface, bundled | OFL-1.1 (`web/static/fonts/OFL-Fraunces.txt`) |
| [Fraunces](https://github.com/undercasetype/Fraunces) variable (app Android) | Tipografía de títulos de la app Android, empaquetada en `android/ui/common/src/main/res/font/fraunces_variable.ttf` / Android app display typeface, bundled | OFL-1.1 (`android/ui/common/src/main/res/raw/ofl_fraunces.txt`) |
| [Inter](https://github.com/rsms/inter) variable | Tipografía de texto del cliente web, empaquetada en `web/static/fonts/Inter-Variable.ttf` / Web client body typeface, bundled | OFL-1.1 (`web/static/fonts/OFL-Inter.txt`) |
| [SvelteKit](https://svelte.dev) 2.70.3 | Framework del cliente web / Web client framework | MIT |
| [Svelte](https://svelte.dev) 5.57.2 | Componentes del cliente web / Web components | MIT |
| [Vite](https://vite.dev) 7.3.7 | Empaquetador y servidor de desarrollo del cliente web / Bundler and dev server | MIT |
| [@sveltejs/adapter-node](https://github.com/sveltejs/kit/tree/main/packages/adapter-node) 5.5.7 | Despliegue del cliente web en Node / Node deployment for the web client | MIT |
| [@sveltejs/vite-plugin-svelte](https://github.com/sveltejs/vite-plugin-svelte) 6.2.4 | Integración de Svelte con Vite / Svelte–Vite integration | MIT |
| [TypeScript](https://www.typescriptlang.org) 5.9.3 | Lenguaje del cliente web / Web client language | Apache-2.0 |
| [svelte-check](https://github.com/sveltejs/language-tools) 4.7.7 | Comprobación de tipos de Svelte / Svelte type checking | MIT |
| [Vitest](https://vitest.dev) 5.0.5 | Pruebas unitarias del cliente web / Web client unit tests | MIT |

**TMDB:** This product uses the TMDB API but is not endorsed or certified by TMDB.

Las marcas y logotipos pertenecen a sus propietarios. / Trademarks and logos belong to their owners.

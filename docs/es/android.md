# App Android

[English](../en/android.md) · [Volver al README](../../README.md) · [Arquitectura y seguridad](arquitectura.md)

Cinecat incluye una única app Android para móvil, tableta y Android TV. Inicia sesión, permite recorrer la biblioteca y reproduce en los tres; la TV tiene una interfaz propia pensada para el mando.

## Estado actual

Al abrirla por primera vez pide la dirección del servidor (solo HTTPS) y la comprueba con `/healthz` y `/api/v1/system/version`: avisa si el servidor es anterior a la 0.30.0 o si la app es más antigua que la `min_client` del servidor. Después:

- **Móvil y tableta:** inicio de sesión con usuario y contraseña, o con un código de vinculación.
- **TV:** vinculación por código. La TV muestra un código `XXXX-XXXX` y un QR; se abre `<servidor>/vincular` en el móvil o el ordenador, se inicia sesión y se aprueba el código (ver [API: vincular un dispositivo](api.md#vincular-un-dispositivo)).

La sesión del dispositivo dura 90 días y se renueva con el uso; aparece en «Mi cuenta → Sesiones abiertas» de la web, donde se puede cerrar. El token se guarda cifrado con una clave del Android Keystore y la app no entra en las copias de seguridad de Android. Tras iniciar sesión, el móvil muestra el portal con las mismas filas y el mismo orden que la web (configurables en Administración), y una barra inferior (riel lateral en tableta y TV) con Inicio, Películas, Series, Colecciones, Fotos y Buscar. **Películas**, **Series** y **Colecciones** desaparecen de la barra cuando el usuario no tiene contenido en ellas; si la consulta de secciones falla o el servidor es anterior a la 0.47.0, se muestran todas. **Fotos** aparece solo cuando el servidor tiene alguna biblioteca de fotos visible; si falla la consulta de bibliotecas, no se muestra. Inicio, Buscar y Cuenta siguen siempre disponibles. Películas y series se ven en cuadrícula con orden y género; las fichas de película, serie (con temporadas y episodios) y colección permiten marcar favorito, «por ver» y visto. La cuenta (usuario, servidor, cerrar sesión, cambiar de servidor) está en el riel de TV y tableta y en el icono del portal en el móvil.

Desde la 0.59.0, Cuenta incluye **Este dispositivo** para editar la calidad máxima y los idiomas de audio y subtítulo preferidos. Cada ajuste es un botón que cambia a la siguiente opción y guarda al pulsar; el foco del mando se ve como en el resto de botones. Se aplican al reproducir en este dispositivo cuando no hayas elegido otra cosa para la película, y el perfil se pierde si se cierra la sesión o se vuelve a vincular. Con un servidor anterior, la sección no aparece.

En la TV, Películas y Series tienen una columna de iniciales (**A–Z** y **#**) para saltar al primer título de cada letra cuando el orden es por título; se probó con más de 600 películas. La interfaz cambia entre la disposición móvil y la de TV según el tipo de dispositivo.

Desde la 0.61.0, la barra incluye **Fotos** cuando el servidor tiene alguna biblioteca de fotos visible. Si solo hay una, se abre directamente; si hay varias, muestra una lista. Dentro de una biblioteca se navega por carpetas con **Subir** y una cuadrícula paginada de miniaturas, con orden Fecha o Nombre. Al abrir una foto, el visor ocupa la pantalla: en el móvil se cambia con un deslizamiento y en la TV con las flechas izquierda y derecha del mando; la tecla central inicia o detiene una presentación de 5 segundos y Atrás cierra. Muestra el nombre, la fecha cuando existe y el contador.

### IP local del servidor

Si el dominio del servidor apunta a la IP pública y el router no tiene DNS local, ir desde casa por Internet puede ser mucho más lento que por la red local, y una TV no permite editar `hosts`. En la pantalla del servidor se puede indicar opcionalmente la **IP local** del servidor (una IPv4 privada como `192.168.1.150`, o una IPv6 local). Cuando el dispositivo está en Wi-Fi o Ethernet y esa IP responde en el puerto del servidor en menos de 1,5 s, la app conecta con ella usando el mismo dominio, así que HTTPS sigue validando el certificado de siempre. Si no responde, o el dispositivo está en datos móviles, usa el DNS normal. El resultado se recuerda 60 s y se vuelve a comprobar al cambiar de red.

## Requisitos

- Android 8.0 o superior (`minSdk 26`).
- Servidor Cinecat 0.30.0 o superior.
- Para compilar: JDK 17 y Android SDK con la plataforma 36.

## Compilación

Desde la carpeta `android/`:

```bash
./gradlew assembleDebug testDebugUnitTest lint
```

En Windows PowerShell:

```powershell
.\gradlew.bat assembleDebug testDebugUnitTest lint
```

El APK de depuración queda en `app/build/outputs/apk/debug/app-debug.apk`.

`local.properties` contiene la ruta del SDK (`sdk.dir`) y no se versiona. Lo crea Android Studio o se escribe a mano con la ruta del SDK instalado.

## Instalación

Cada versión publica en la [página de versiones de GitHub](https://github.com/catlinux/Cinecat/releases/latest) el APK firmado `cinecat-X.Y.Z.apk` y su suma `cinecat-X.Y.Z.apk.sha256`. Se instala una sola vez a mano; después la app se actualiza sola.

- **Móvil:** abre el enlace de la versión en el navegador, descarga el APK, ábrelo y permite al navegador instalar aplicaciones cuando lo pida.
- **TV:** con `adb` desde el ordenador (`adb connect IP:5555` y `adb install cinecat-X.Y.Z.apk`), o con una app de descargas para TV (por ejemplo «Downloader») apuntando a la URL del APK de la versión.

Si ya tenías instalada una versión anterior a la 0.35.0, desinstálala antes: estaba firmada con una clave de pruebas y Android no permite actualizarla con el APK firmado. Habrá que volver a iniciar sesión o a vincular la TV.

### Actualización automática

Al arrancar y cada 12 horas, la app pregunta al servidor si hay un APK más nuevo: el de la versión del servidor, si su `versionCode` es mayor que el instalado. Si «Actualizar automáticamente» está activo (en Cuenta → Actualizaciones), lo descarga de la *release* de GitHub y ofrece «Instalar». Antes comprueba el SHA-256, que el APK sea de Cinecat, de una versión mayor y firmado con la misma clave que la app instalada. Mientras se instala se muestra una pantalla completa **«Actualizando Cinecat»**; al volver a abrir la app aparece una vez el aviso **«Cinecat actualizado»** con la versión instalada. El aviso se decide comparando la versión instalada con la última que la app recuerda haber visto, sin depender de lo guardado justo antes de instalar. En Android 9 y anteriores el sistema siempre pide confirmar la instalación; la primera vez hay que permitir a Cinecat instalar aplicaciones (Ajustes → Seguridad → Orígenes desconocidos en la TV). El servidor solo consulta GitHub si la comprobación de actualizaciones está activada en Administración.

### Versión de pruebas

El APK de depuración (`assembleDebug`) se llama «Cinecat (pruebas)» y usa el identificador `io.github.catlinux.cinecat.debug`, así que convive con la versión publicada. No se actualiza solo:

```bash
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

### Clave de firma

Los APK publicados se firman en GitHub Actions (`.github/workflows/android-release.yml`) con un almacén PKCS12 guardado en los secretos del repositorio `ANDROID_KEYSTORE_BASE64`, `ANDROID_KEYSTORE_PASSWORD` y `ANDROID_KEY_ALIAS`. Para firmar en local, `assembleRelease` lee `CINECAT_KEYSTORE_FILE`, `CINECAT_KEYSTORE_PASSWORD` y `CINECAT_KEY_ALIAS`; sin ellas genera un APK sin firmar. Guarda una copia de la clave y su contraseña fuera del equipo: si se pierde, las apps instaladas no podrán actualizarse sin desinstalarlas.

## Estructura de módulos

- `app`: actividad, contenedor de dependencias, selección entre interfaz móvil y de TV y pantallas de servidor, inicio de sesión, vinculación e inicio.
- `ui/common`: tema Marquesina de Compose, colores y tipografía Fraunces.
- `core/api`: cliente de la API (Retrofit, OkHttp y kotlinx.serialization), cabecera de sesión solo hacia el servidor y DNS de IP local.
- `core/data`: dirección del servidor, sesión cifrada (DataStore y Android Keystore), comprobación de compatibilidad, inicio de sesión y vinculación.
- `core/player`: Media3 (ExoPlayer) y detección de los códecs del equipo.

## Versión e identificador

El `versionName` es el contenido del archivo `VERSION` de la raíz del repositorio, recortado. El `versionCode` se calcula en Gradle como:

```text
major * 10000 + minor * 100 + patch
```

Así, `0.32.0` produce el código `3200`. El identificador de la app es `io.github.catlinux.cinecat`.

## Reproducción

La app no incluye FFmpeg ni libmpv. Reproduce con Media3 (ExoPlayer): al abrir una película o episodio envía al servidor los códecs que el equipo decodifica (y los de audio que admite por HDMI en passthrough), y el servidor decide entre reproducción directa y HLS, recodificando lo que el equipo no pueda reproducir. Si la reproducción directa falla, la app pide HLS una vez. Reanuda donde se dejó y guarda el progreso cada 10 s, al pausar y al salir, compartido con la web.

El reproductor recuerda en el servidor el audio, los subtítulos y la calidad elegidos para cada elemento y los restaura al reabrirlo, siempre que sigan existiendo; si ya no están, ignora esa preferencia. En el móvil, al salir de la app mientras se reproduce un vídeo, el reproductor pasa a una ventana de imagen en imagen sin los controles de la app; al cerrar la ventana se detiene la reproducción y se guarda el progreso.

Desde la 0.58.0, las pistas de imagen incrustadas (PGS, VobSub y DVB) aparecen en «Pistas» con la marca « · imagen». No se cargan en ExoPlayer: al elegirlas, la app guarda la preferencia y recarga la reproducción con `subtitle=<id>` en el segundo actual; al quitarlas o pasar a texto desde una quemada recarga sin el subtítulo quemado. El servidor las quema al recodificar por HLS.

En la TV, con el mando: centro pausa o reanuda, izquierda y derecha retroceden o avanzan 10 s, abajo muestra los controles, arriba o Menú abre «Pistas» (audio y subtítulos) y Atrás sale. Las pistas sin título se nombran por su idioma o como «Audio 1», «Audio 2», «Subtítulo 1»… (0.49.1). En el móvil la reproducción va a pantalla completa en horizontal. En las series, la ficha ofrece el siguiente episodio.

Desde la 0.62.0, si el vídeo tiene segmentos, durante la intro aparece **Saltar intro** y durante los créditos **Saltar créditos**; el botón salta al final del segmento y, si este llega al final del vídeo, la reproducción termina como siempre. En la TV recibe el foco al aparecer y se activa con la tecla central; al desaparecer, el foco vuelve al reproductor, y las flechas izquierda y derecha siguen controlando el avance cuando el botón no está. En el móvil es un botón táctil normal. Sin segmentos, el reproductor se comporta igual que antes.

Desde la 0.64.0, la ficha de la serie tiene el interruptor **Reproducir el siguiente episodio automáticamente**, con estado **Activado** o **Desactivado**, y lo guarda en el servidor por usuario y serie; en la TV es un botón enfocable con el foco visible de siempre. Si el ajuste está activo y hay un siguiente episodio, al terminar aparece el panel **Siguiente episodio** con una cuenta atrás de 8 s, **Reproducir ya** y **Cancelar**; en la TV, Reproducir ya recibe el foco y Atrás cancela. El botón de los créditos dice **Siguiente episodio** cuando hay uno y lo abre. En el reproductor de las apps no hay interruptor: se cambia desde la ficha de la serie. En las reproducciones locales de descargas no hay siguiente episodio automático.

Desde la 0.49.0, la ficha de una película a medias muestra **Desde el principio** junto a **Continuar**, y el botón del siguiente episodio dice **Continuar S01E01** cuando hay progreso. La TV declara `mch` y recibe AAC 5.1 por HLS cuando la pista elegida tiene 6 o más canales; el móvil no declara `mch` y sigue recibiendo estéreo (0.56.0, pendiente de probar en una TV real).

## Descargas sin conexión

Solo están en móvil y tableta; la TV no muestra el botón ni la pantalla de Descargas. En la ficha de una película y en cada episodio, **Descargar** consulta `/play` con las capacidades del equipo. Descarga el archivo original, sea MKV, MP4 o MOV, solo si la respuesta trae `download_direct: true`, es decir, si el dispositivo declara que reproduce en directo su contenedor, su vídeo y su audio (los mismos criterios que la reproducción directa nativa). Si no, explica el motivo (`download_blocked`: audio, vídeo o formato) y no genera ninguna versión recodificada.

La descarga la hace el DownloadManager del sistema desde `GET /api/v1/items/{id}/download`, con la sesión de la app y a la carpeta privada de la app (`Environment.DIRECTORY_MOVIES`), con el nombre `<id>.<extensión>`. No pide permisos de almacenamiento y el archivo se borra al desinstalar la app. La notificación del sistema muestra el título de la película o `Serie · TxE · título` durante la descarga y al terminar. Mientras baja, el botón muestra **Descargando n %** y ofrece cancelar; al terminar muestra **Descargado** y ofrece reproducir o borrar.

En **Cuenta › Descargas** ves el cartel, el título, el tamaño y el progreso de cada descarga, con acciones para reproducir, cancelar y borrar. La reproducción es local con Media3 desde la URI `file://`, sin HLS ni subtítulos del servidor; las pistas incrustadas del archivo siguen apareciendo en **Pistas** y se empieza en la posición guardada. El progreso se guarda en el registro local y se intenta enviar con `PUT /api/v1/items/{id}/progress`; si falla, queda pendiente y se reintenta al arrancar la app o al abrir Descargas. Si al arrancar no hay conexión y hay descargas completadas, la pantalla de error ofrece **Ver descargas** y se pueden reproducir sin servidor.

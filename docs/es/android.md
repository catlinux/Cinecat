# App Android

[English](../en/android.md) · [Volver al README](../../README.md) · [Arquitectura y seguridad](arquitectura.md)

Cinecat incluye una única app Android para móvil, tableta y Android TV. Ahora mismo inicia sesión, permite recorrer la biblioteca y reproduce en el móvil, la tableta y la TV; la TV aún no tiene su interfaz propia.

## Estado actual

Al abrirla por primera vez pide la dirección del servidor (solo HTTPS) y la comprueba con `/healthz` y `/api/v1/system/version`: avisa si el servidor es anterior a la 0.30.0 o si la app es más antigua que la `min_client` del servidor. Después:

- **Móvil y tableta:** inicio de sesión con usuario y contraseña, o con un código de vinculación.
- **TV:** vinculación por código. La TV muestra un código `XXXX-XXXX` y un QR; se abre `<servidor>/vincular` en el móvil o el ordenador, se inicia sesión y se aprueba el código (ver [API: vincular un dispositivo](api.md#vincular-un-dispositivo)).

La sesión del dispositivo dura 90 días y se renueva con el uso; aparece en «Mi cuenta → Sesiones abiertas» de la web, donde se puede cerrar. El token se guarda cifrado con una clave del Android Keystore y la app no entra en las copias de seguridad de Android. Tras iniciar sesión, el móvil muestra el portal con las mismas filas y el mismo orden que la web (configurables en Administración), y una barra inferior (riel lateral en tableta) con Películas, Series, Colecciones y Buscar. Películas y series se ven en cuadrícula con orden y género; las fichas de película, serie (con temporadas y episodios) y colección permiten marcar favorito, «por ver» y visto. La cuenta (usuario, servidor, cerrar sesión, cambiar de servidor) está en el icono del portal. En la TV, por ahora, se ve la misma interfaz que en la tableta, manejable con el mando. La interfaz cambia entre la disposición móvil y la de TV según el tipo de dispositivo.

### IP local del servidor

Si el dominio del servidor apunta a la IP pública y el router no tiene DNS local, ir desde casa por Internet puede ser mucho más lento que por la red local, y una TV no permite editar `hosts`. En la pantalla del servidor se puede indicar opcionalmente la **IP local** del servidor (una IPv4 privada como `192.168.1.150`, o una IPv6 local). Cuando el dispositivo está en Wi-Fi o Ethernet y esa IP responde en el puerto del servidor en menos de 1,5 s, la app conecta con ella usando el mismo dominio, así que HTTPS sigue validando el certificado de siempre. Si no responde, o el dispositivo está en datos móviles, usa el DNS normal. El resultado se recuerda 60 s y se vuelve a comprobar al cambiar de red.

## Requisitos

- Android 8.0 o superior (`minSdk 26`).
- Servidor Cinecat 0.30.0 o superior.
- Para compilar: JDK 17 y Android SDK con la plataforma 35.

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

Al arrancar y cada 12 horas, la app pregunta al servidor si hay un APK más nuevo: el de la versión del servidor, si su `versionCode` es mayor que el instalado. Si «Actualizar automáticamente» está activo (en Cuenta → Actualizaciones), lo descarga de la *release* de GitHub y ofrece «Instalar». Antes comprueba el SHA-256, que el APK sea de Cinecat, de una versión mayor y firmado con la misma clave que la app instalada. En Android 9 y anteriores el sistema siempre pide confirmar la instalación; la primera vez hay que permitir a Cinecat instalar aplicaciones (Ajustes → Seguridad → Orígenes desconocidos en la TV). El servidor solo consulta GitHub si la comprobación de actualizaciones está activada en Administración.

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

En la TV, con el mando: centro pausa o reanuda, izquierda y derecha retroceden o avanzan 10 s, abajo muestra los controles, arriba o Menú abre «Pistas» (audio y subtítulos) y Atrás sale. En el móvil la reproducción va a pantalla completa en horizontal. En las series, la ficha ofrece el siguiente episodio.

# App Android

[English](../en/android.md) · [Volver al README](../../README.md) · [Arquitectura y seguridad](arquitectura.md)

Cinecat incluye una única app Android para móvil, tableta y Android TV. Ahora mismo es un **esqueleto**: abre una pantalla de presentación con el tema Marquesina y todavía no se conecta al servidor. La primera versión utilizable llegará con los siguientes pasos del plan de apps.

## Estado actual

La app compila, pero todavía no permite iniciar sesión, navegar por la biblioteca ni reproducir contenido. La pantalla de inicio muestra el nombre de la app, su versión y un aviso de que aún no hay conexión con el servidor. La interfaz cambia entre la disposición móvil y la de TV según el tipo de dispositivo.

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

Con un dispositivo conectado por USB:

```bash
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

En una TV con depuración por red, conéctala antes:

```bash
adb connect IP:5555
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

## Estructura de módulos

- `app`: actividad, selección entre interfaz móvil y de TV y pantallas de presentación.
- `ui/common`: tema Marquesina de Compose, colores y tipografía Fraunces.
- `core/api`: reservado para el cliente de la API; vacío por ahora.
- `core/data`: reservado para sesión, preferencias y repositorios; vacío por ahora.
- `core/player`: reservado para el reproductor; vacío por ahora.

## Versión e identificador

El `versionName` es el contenido del archivo `VERSION` de la raíz del repositorio, recortado. El `versionCode` se calcula en Gradle como:

```text
major * 10000 + minor * 100 + patch
```

Así, `0.31.0` produce el código `3100`. El identificador de la app es `io.github.catlinux.cinecat`.

## Reproducción

La app no incluye FFmpeg ni libmpv. Cuando se añada la reproducción, el servidor seguirá decidiendo entre reproducción directa y HLS y recodificará lo que el equipo no pueda reproducir; la app solo recibirá la URL firmada y la pasará al reproductor.

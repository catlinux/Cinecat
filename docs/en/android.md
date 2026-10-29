# Android app

[Español](../es/android.md) · [Back to README](../../README.en.md) · [Architecture and security](architecture.md)

Cinecat includes a single Android app for phones, tablets and Android TV. It is currently a **skeleton**: it opens a welcome screen with the Marquesina theme and does not connect to the server yet. The first usable version will arrive with the following steps of the app plan.

## Current state

The app builds, but it cannot sign in, browse the library or play content yet. The welcome screen shows the app name, its version and a notice that there is no server connection yet. The interface switches between the mobile and TV layouts depending on the device type.

## Requirements

- Android 8.0 or later (`minSdk 26`).
- Cinecat server 0.30.0 or later.
- To build: JDK 17 and Android SDK with platform 35.

## Build

From the `android/` directory:

```bash
./gradlew assembleDebug testDebugUnitTest lint
```

On Windows PowerShell:

```powershell
.\gradlew.bat assembleDebug testDebugUnitTest lint
```

The debug APK is written to `app/build/outputs/apk/debug/app-debug.apk`.

`local.properties` contains the SDK path (`sdk.dir`) and is not versioned. Android Studio creates it or it can be written by hand with the path of the installed SDK.

## Installation

With a USB-connected device:

```bash
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

On a TV with network debugging, connect it first:

```bash
adb connect IP:5555
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

## Module structure

- `app`: activity, mobile/TV interface selection and welcome screens.
- `ui/common`: Compose Marquesina theme, colours and Fraunces typography.
- `core/api`: reserved for the API client; empty for now.
- `core/data`: reserved for session, preferences and repositories; empty for now.
- `core/player`: reserved for the player; empty for now.

## Version and identifier

The `versionName` is the contents of the `VERSION` file at the repository root, trimmed. The `versionCode` is calculated in Gradle as:

```text
major * 10000 + minor * 100 + patch
```

For example, `0.31.0` produces code `3100`. The app identifier is `io.github.catlinux.cinecat`.

## Playback

The app does not include FFmpeg or libmpv. When playback is added, the server will keep choosing between direct play and HLS and will re-encode what the device cannot play; the app will only receive the signed URL and pass it to the player.

# Android app

[Español](../es/android.md) · [Back to README](../../README.en.md) · [Architecture and security](architecture.md)

Cinecat includes a single Android app for phones, tablets and Android TV. It currently connects to the server and signs in, but does not show the library or play anything yet. The first usable version will arrive with the following steps of the app plan.

## Current state

On first launch it asks for the server address (HTTPS only) and checks it with `/healthz` and `/api/v1/system/version`: it warns if the server is older than 0.30.0 or if the app is older than the server's `min_client`. Then:

- **Phone and tablet:** sign in with username and password, or with a pairing code.
- **TV:** pairing by code. The TV shows an `XXXX-XXXX` code and a QR code; open `<server>/vincular` on a phone or computer, sign in and approve the code (see [API: pairing a device](api.md#pairing-a-device)).

The device session lasts 90 days and is renewed with use; it appears under "My account → Open sessions" in the web client, where it can be closed. The token is stored encrypted with an Android Keystore key and the app is excluded from Android backups. After signing in, the home screen is provisional (user, server, sign out and change server): browsing the library and playback are not available yet. The interface switches between the mobile and TV layouts depending on the device type.

### Server local IP

If the server domain points to the public IP and the router has no local DNS, going through the Internet from home can be much slower than the local network, and a TV does not let you edit `hosts`. On the server screen you can optionally enter the server's **local IP** (a private IPv4 such as `192.168.1.150`, or a local IPv6). When the device is on Wi-Fi or Ethernet and that IP answers on the server port within 1.5 s, the app connects to it using the same domain, so HTTPS still validates the usual certificate. If it does not answer, or the device is on mobile data, normal DNS is used. The result is remembered for 60 s and checked again when the network changes.

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

- `app`: activity, dependency container, mobile/TV interface selection and the server, sign-in, pairing and home screens.
- `ui/common`: Compose Marquesina theme, colours and Fraunces typography.
- `core/api`: API client (Retrofit, OkHttp and kotlinx.serialization), session header sent only to the server, and local-IP DNS.
- `core/data`: server address, encrypted session (DataStore and Android Keystore), compatibility check, sign-in and pairing.
- `core/player`: reserved for the player; empty for now.

## Version and identifier

The `versionName` is the contents of the `VERSION` file at the repository root, trimmed. The `versionCode` is calculated in Gradle as:

```text
major * 10000 + minor * 100 + patch
```

For example, `0.32.0` produces code `3200`. The app identifier is `io.github.catlinux.cinecat`.

## Playback

The app does not include FFmpeg or libmpv. When playback is added, the server will keep choosing between direct play and HLS and will re-encode what the device cannot play; the app will only receive the signed URL and pass it to the player.

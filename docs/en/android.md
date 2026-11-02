# Android app

[Español](../es/android.md) · [Back to README](../../README.en.md) · [Architecture and security](architecture.md)

Cinecat includes a single Android app for phones, tablets and Android TV. It signs in, lets you browse the library and plays on all three; the TV has its own interface designed for the remote.

## Current state

On first launch it asks for the server address (HTTPS only) and checks it with `/healthz` and `/api/v1/system/version`: it warns if the server is older than 0.30.0 or if the app is older than the server's `min_client`. Then:

- **Phone and tablet:** sign in with username and password, or with a pairing code.
- **TV:** pairing by code. The TV shows an `XXXX-XXXX` code and a QR code; open `<server>/vincular` on a phone or computer, sign in and approve the code (see [API: pairing a device](api.md#pairing-a-device)).

The device session lasts 90 days and is renewed with use; it appears under "My account → Open sessions" in the web client, where it can be closed. The token is stored encrypted with an Android Keystore key and the app is excluded from Android backups. After signing in, phones show the portal with the same rows and order as the web client (configurable in Administration), and a bottom bar (side rail on tablets and TV) with Home, Movies, Series, Collections and Search. **Movies**, **Series** and **Collections** disappear from the bar when the user has no content in them; if the sections request fails or the server is older than 0.47.0, all of them are shown. Home, Search and Account remain available at all times. Movies and series are shown in a grid with sort and genre; the movie, series (with seasons and episodes) and collection pages let you mark favourite, "to watch" and watched. The account (user, server, sign out, change server) is in the TV/tablet rail and behind the portal icon on phones.

On TV, Movies and Series have a column of initials (**A–Z** and **#**) to jump to the first title of each letter when sorted by title; it was tested with over 600 movies. The interface switches between the mobile and TV layouts depending on the device type.

### Server local IP

If the server domain points to the public IP and the router has no local DNS, going through the Internet from home can be much slower than the local network, and a TV does not let you edit `hosts`. On the server screen you can optionally enter the server's **local IP** (a private IPv4 such as `192.168.1.150`, or a local IPv6). When the device is on Wi-Fi or Ethernet and that IP answers on the server port within 1.5 s, the app connects to it using the same domain, so HTTPS still validates the usual certificate. If it does not answer, or the device is on mobile data, normal DNS is used. The result is remembered for 60 s and checked again when the network changes.

## Requirements

- Android 8.0 or later (`minSdk 26`).
- Cinecat server 0.30.0 or later.
- To build: JDK 17 and Android SDK with platform 36.

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

Each version publishes the signed APK `cinecat-X.Y.Z.apk` and its checksum `cinecat-X.Y.Z.apk.sha256` on the [GitHub releases page](https://github.com/catlinux/Cinecat/releases/latest). It is installed by hand once; after that the app updates itself.

- **Phone:** open the release link in the browser, download the APK, open it and allow the browser to install apps when asked.
- **TV:** with `adb` from the computer (`adb connect IP:5555` and `adb install cinecat-X.Y.Z.apk`), or with a TV download app (for example "Downloader") pointed at the release APK URL.

If you had a version older than 0.35.0 installed, uninstall it first: it was signed with a test key and Android does not allow updating it with the signed APK. You will need to sign in or pair the TV again.

### Automatic updates

On start and every 12 hours, the app asks the server whether there is a newer APK: the one for the server's version, if its `versionCode` is higher than the installed one. If "Update automatically" is on (Account → Updates), it downloads it from the GitHub release and offers "Install". Before that it checks the SHA-256, that the APK is Cinecat's, a higher version and signed with the same key as the installed app. While it installs, a full-screen **"Updating Cinecat"** screen is shown; when the app is opened again, a one-time **"Cinecat updated"** notice shows the installed version. The notice is decided by comparing the installed version with the last one the app remembers seeing, instead of relying on what was saved just before installing. On Android 9 and earlier the system always asks to confirm the installation; the first time you must allow Cinecat to install apps (Settings → Security → Unknown sources on the TV). The server only contacts GitHub if update checking is enabled in Administration.

### Testing build

The debug APK (`assembleDebug`) is called "Cinecat (pruebas)" and uses the identifier `io.github.catlinux.cinecat.debug`, so it coexists with the published version. It does not update itself:

```bash
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

### Signing key

Published APKs are signed in GitHub Actions (`.github/workflows/android-release.yml`) with a PKCS12 keystore stored in the repository secrets `ANDROID_KEYSTORE_BASE64`, `ANDROID_KEYSTORE_PASSWORD` and `ANDROID_KEY_ALIAS`. To sign locally, `assembleRelease` reads `CINECAT_KEYSTORE_FILE`, `CINECAT_KEYSTORE_PASSWORD` and `CINECAT_KEY_ALIAS`; without them it produces an unsigned APK. Keep a copy of the key and its password off the machine: if it is lost, installed apps cannot be updated without uninstalling them.

## Module structure

- `app`: activity, dependency container, mobile/TV interface selection and the server, sign-in, pairing and home screens.
- `ui/common`: Compose Marquesina theme, colours and Fraunces typography.
- `core/api`: API client (Retrofit, OkHttp and kotlinx.serialization), session header sent only to the server, and local-IP DNS.
- `core/data`: server address, encrypted session (DataStore and Android Keystore), compatibility check, sign-in and pairing.
- `core/player`: Media3 (ExoPlayer) and detection of the device's codecs.

## Version and identifier

The `versionName` is the contents of the `VERSION` file at the repository root, trimmed. The `versionCode` is calculated in Gradle as:

```text
major * 10000 + minor * 100 + patch
```

For example, `0.32.0` produces code `3200`. The app identifier is `io.github.catlinux.cinecat`.

## Playback

The app does not include FFmpeg or libmpv. It plays with Media3 (ExoPlayer): when a movie or episode is opened it sends the server the codecs the device decodes (and the audio formats it supports as HDMI passthrough), and the server chooses between direct play and HLS, re-encoding what the device cannot play. If direct play fails, the app requests HLS once. It resumes where you left off and saves progress every 10 s, on pause and on exit, shared with the web client.

The player remembers the audio, subtitles and quality chosen for each item on the server and restores them when it is reopened, as long as they still exist; otherwise that preference is ignored. On phones, leaving the app while a video is playing moves the player to a picture-in-picture window without the app controls; closing the window stops playback and saves progress.

Since 0.58.0, embedded image tracks (PGS, VobSub and DVB) appear under "Tracks" with the " · imagen" mark. They are not loaded into ExoPlayer: choosing one saves the preference and reloads playback with `subtitle=<id>` at the current second; removing it or switching to text from one reloads without the burned subtitle. The server burns them while re-encoding for HLS.

On TV, with the remote: centre pauses or resumes, left and right go back or forward 10 s, down shows the controls, up or Menu opens "Tracks" (audio and subtitles) and Back exits. Untitled tracks are named by language or as "Audio 1", "Audio 2", "Subtitle 1"… (0.49.1). On phones playback goes full screen in landscape. In series, the page offers the next episode.

Since 0.49.0, a partly watched movie page shows **Desde el principio** next to **Continuar**, and the next-episode button reads **Continuar S01E01** when there is progress. The TV declares `mch` and receives 5.1 AAC over HLS when the chosen track has 6 or more channels; the phone does not declare `mch` and stays in stereo (0.56.0, not yet tried on a real TV).

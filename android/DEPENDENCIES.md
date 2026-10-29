# Dependencias de Android

Versiones elegidas el 2026-10-29. Se comprobó la licencia en el POM o repositorio de cada versión exacta.

| Dependencia | Versión | Licencia | Comprobación |
|---|---|---|---|
| Gradle | 9.8.2 | Apache-2.0 | https://github.com/gradle/gradle/blob/v9.8.2/LICENSE |
| Android Gradle Plugin | 9.5.0 | Apache-2.0 | https://dl.google.com/dl/android/maven2/com/android/tools/build/gradle/9.5.0/gradle-9.5.0.pom |
| Kotlin Gradle Plugin | 2.4.21 | Apache-2.0 | https://repo1.maven.org/maven2/org/jetbrains/kotlin/kotlin-gradle-plugin/2.4.21/kotlin-gradle-plugin-2.4.21.pom |
| Kotlin stdlib | 2.4.21 | Apache-2.0 | https://repo1.maven.org/maven2/org/jetbrains/kotlin/kotlin-stdlib/2.4.21/kotlin-stdlib-2.4.21.pom |
| Kotlin Compose compiler plugin | 2.4.21 | Apache-2.0 | https://repo1.maven.org/maven2/org/jetbrains/kotlin/compose-compiler-gradle-plugin/2.4.21/compose-compiler-gradle-plugin-2.4.21.pom |
| Compose BOM | 2026.06.00 | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/compose/compose-bom/2026.06.00/compose-bom-2026.06.00.pom |
| androidx.compose.ui:ui | 1.11.3 (BOM) | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/compose/ui/ui/1.11.3/ui-1.11.3.pom |
| androidx.compose.foundation:foundation | 1.11.3 (BOM) | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/compose/foundation/foundation/1.11.3/foundation-1.11.3.pom |
| androidx.compose.material3:material3 | 1.4.0 (BOM) | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/compose/material3/material3/1.4.0/material3-1.4.0.pom |
| androidx.compose.ui:ui-tooling-preview | 1.11.3 (BOM) | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/compose/ui/ui-tooling-preview/1.11.3/ui-tooling-preview-1.11.3.pom |
| androidx.compose.ui:ui-tooling | 1.11.3 (BOM, debug) | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/compose/ui/ui-tooling/1.11.3/ui-tooling-1.11.3.pom |
| androidx.activity:activity-compose | 1.10.1 | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/activity/activity-compose/1.10.1/activity-compose-1.10.1.pom |
| androidx.core:core-ktx | 1.16.0 | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/core/core-ktx/1.16.0/core-ktx-1.16.0.pom |
| androidx.lifecycle:lifecycle-runtime-ktx | 2.10.0 | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/lifecycle/lifecycle-runtime-ktx/2.10.0/lifecycle-runtime-ktx-2.10.0.pom |
| androidx.tv:tv-material | 1.1.0 | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/tv/tv-material/1.1.0/tv-material-1.1.0.pom |
| OkHttp | 5.3.2 | Apache-2.0 | https://repo1.maven.org/maven2/com/squareup/okhttp3/okhttp/5.3.2/okhttp-5.3.2.pom |
| Retrofit | 3.0.0 | Apache-2.0 | https://repo1.maven.org/maven2/com/squareup/retrofit2/retrofit/3.0.0/retrofit-3.0.0.pom |
| Retrofit converter-kotlinx-serialization | 3.0.0 | Apache-2.0 | https://repo1.maven.org/maven2/com/squareup/retrofit2/converter-kotlinx-serialization/3.0.0/converter-kotlinx-serialization-3.0.0.pom |
| kotlinx-serialization-json | 1.12.0 | Apache-2.0 | https://repo1.maven.org/maven2/org/jetbrains/kotlinx/kotlinx-serialization-json/1.12.0/kotlinx-serialization-json-1.12.0.pom |
| Kotlin serialization plugin | 2.4.21 | Apache-2.0 | https://repo1.maven.org/maven2/org/jetbrains/kotlin/kotlin-serialization/2.4.21/kotlin-serialization-2.4.21.pom |
| kotlinx-coroutines-core | 1.11.0 | Apache-2.0 | https://repo1.maven.org/maven2/org/jetbrains/kotlinx/kotlinx-coroutines-core/1.11.0/kotlinx-coroutines-core-1.11.0.pom |
| kotlinx-coroutines-test | 1.11.0 (pruebas) | Apache-2.0 | https://repo1.maven.org/maven2/org/jetbrains/kotlinx/kotlinx-coroutines-test/1.11.0/kotlinx-coroutines-test-1.11.0.pom |
| OkHttp mockwebserver | 5.3.2 (pruebas) | Apache-2.0 | https://repo1.maven.org/maven2/com/squareup/okhttp3/mockwebserver/5.3.2/mockwebserver-5.3.2.pom |
| androidx.datastore:datastore-preferences | 1.3.0 | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/datastore/datastore-preferences/1.3.0/datastore-preferences-1.3.0.pom |
| androidx.datastore:datastore-preferences-core | 1.3.0 (pruebas) | Apache-2.0 | https://dl.google.com/dl/android/maven2/androidx/datastore/datastore-preferences-core/1.3.0/datastore-preferences-core-1.3.0.pom |
| JUnit | 4.13.2 | EPL-1.0 | https://repo1.maven.org/maven2/junit/junit/4.13.2/junit-4.13.2.pom |
| Fraunces Variable | archivo local | OFL-1.1 | web/static/fonts/OFL-Fraunces.txt |

JUnit es la única dependencia con una licencia distinta de Apache-2.0, MIT o BSD: se usa solo en las pruebas unitarias de JVM y conserva su licencia EPL-1.0.

## Criterio de versiones

- **Gradle 9.8.2**: versión estable actual de https://services.gradle.org/versions/current.
- **AGP 9.5.0**: última estable de Google Maven; sus notas exigen Gradle 9.6.0 o superior y JDK 17.
- **Kotlin 2.4.21**: última estable de Maven Central; AGP 9.5 incluye KGP 2.2.10 y el proyecto lo eleva a 2.4.21 con el `buildscript` de la raíz, como documenta Android para usar una KGP superior.
- **Compose BOM 2026.06.00**: última cuyo `ui` (1.11.3) declara `minCompileSdk=35`. Los BOM 2026.08.00 y posteriores ya exigen compileSdk 37.
- **activity-compose 1.10.1**: última con `minCompileSdk=35`; desde 1.11.0 exige 36.
- **core-ktx 1.16.0**: última con `minCompileSdk=35`; desde 1.17.0 exige 36.
- **lifecycle-runtime-ktx 2.10.0**: última cuyo `lifecycle-runtime-compose` declara `minCompileSdk=35`; 2.11.0 exige 37.
- **tv-material 1.1.0**: última estable y compatible con compileSdk 35.
- **JUnit 4.13.2**: versión estable usada por las pruebas JVM.
- **DataStore 1.3.0**: última estable de Google Maven; `datastore-preferences-core` se usa además en las pruebas JVM de DataStore.

El proyecto fija `compileSdk = 35` y `targetSdk = 35` por requisito de este paso; por eso no se usan las versiones más nuevas de AndroidX que exigirían compileSdk 36 o 37.

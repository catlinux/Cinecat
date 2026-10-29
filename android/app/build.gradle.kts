plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
}

val versionFile = rootProject.file("../VERSION")
check(versionFile.isFile) {
    "No se encontró el archivo VERSION en la raíz del repositorio: ${versionFile.absolutePath}"
}
val projectVersionName = versionFile.readText().trim()
val versionMatch = Regex("""^(\d+)\.(\d+)\.(\d+)$""").matchEntire(projectVersionName)
    ?: error("VERSION debe tener el formato x.y.z; valor recibido: \"$projectVersionName\"")
val (major, minor, patch) = versionMatch.destructured
val projectVersionCode = major.toInt() * 10_000 + minor.toInt() * 100 + patch.toInt()

android {
    namespace = "io.github.catlinux.cinecat.app"
    compileSdk = 35

    defaultConfig {
        applicationId = "io.github.catlinux.cinecat"
        minSdk = 26
        targetSdk = 35
        versionCode = projectVersionCode
        versionName = projectVersionName
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    lint {
        abortOnError = true
    }
}

dependencies {
    implementation(project(":core:data"))
    implementation(project(":ui:common"))
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.androidx.datastore.preferences)
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.foundation)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.ui.tooling.preview)
    implementation(libs.androidx.tv.material)
    implementation(libs.zxing.core)
    debugImplementation(libs.androidx.compose.ui.tooling)

    testImplementation(libs.junit)
    testImplementation(libs.kotlinx.coroutines.test)
}

package io.github.catlinux.cinecat.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp

@Composable
fun DeviceProfileSection(
    viewModel: DeviceProfileViewModel,
    isTelevision: Boolean
) {
    val state by viewModel.state.collectAsState()
    when (val current = state) {
        DeviceProfileUiState.Loading, DeviceProfileUiState.Unsupported -> Unit
        is DeviceProfileUiState.Error -> Column(
            modifier = Modifier.fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(
                text = stringResource(R.string.device_profile_section),
                style = MaterialTheme.typography.titleLarge,
                color = MaterialTheme.colorScheme.primary
            )
            Text(
                text = stringResource(R.string.device_profile_load_error),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.error
            )
        }
        is DeviceProfileUiState.Ready -> Column(
            modifier = Modifier.fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(
                text = stringResource(R.string.device_profile_section),
                style = MaterialTheme.typography.titleLarge,
                color = MaterialTheme.colorScheme.primary
            )
            DeviceProfileRow(
                title = stringResource(R.string.device_profile_quality),
                value = qualityLabel(current.profile.maxQuality),
                isTelevision = isTelevision,
                onCycle = viewModel::cycleMaxQuality
            )
            DeviceProfileRow(
                title = stringResource(R.string.device_profile_audio_language),
                value = languageLabel(current.profile.audioLanguage, subtitle = false),
                isTelevision = isTelevision,
                onCycle = viewModel::cycleAudioLanguage
            )
            DeviceProfileRow(
                title = stringResource(R.string.device_profile_subtitle_language),
                value = languageLabel(current.profile.subtitleLanguage, subtitle = true),
                isTelevision = isTelevision,
                onCycle = viewModel::cycleSubtitleLanguage
            )
            when {
                current.saving -> Text(
                    text = stringResource(R.string.device_profile_saving),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                current.saveResult == DeviceProfileSaveResult.Saved -> Text(
                    text = stringResource(R.string.device_profile_saved),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.primary
                )
                current.saveResult == DeviceProfileSaveResult.Failed -> Text(
                    text = stringResource(R.string.device_profile_save_failed),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error
                )
            }
        }
    }
}

@Composable
private fun DeviceProfileRow(
    title: String,
    value: String,
    isTelevision: Boolean,
    onCycle: () -> Unit
) {
    SecondaryActionButton(
        isTelevision = isTelevision,
        label = stringResource(R.string.device_profile_row, title, value),
        fillWidth = !isTelevision,
        onClick = onCycle
    )
}

@Composable
private fun qualityLabel(value: String): String = when (value) {
    "" -> stringResource(R.string.device_profile_quality_unlimited)
    "480" -> stringResource(R.string.device_profile_quality_480)
    "720" -> stringResource(R.string.device_profile_quality_720)
    "1080" -> stringResource(R.string.device_profile_quality_1080)
    else -> value
}

@Composable
private fun languageLabel(value: String, subtitle: Boolean): String = when (value) {
    "" -> stringResource(R.string.device_profile_language_none)
    "off" -> if (subtitle) {
        stringResource(R.string.device_profile_subtitles_off)
    } else {
        value
    }
    "es" -> stringResource(R.string.device_profile_language_es)
    "ca" -> stringResource(R.string.device_profile_language_ca)
    "en" -> stringResource(R.string.device_profile_language_en)
    "fr" -> stringResource(R.string.device_profile_language_fr)
    "de" -> stringResource(R.string.device_profile_language_de)
    "it" -> stringResource(R.string.device_profile_language_it)
    "pt" -> stringResource(R.string.device_profile_language_pt)
    "ja" -> stringResource(R.string.device_profile_language_ja)
    else -> value
}

package io.github.catlinux.cinecat.app

import androidx.compose.foundation.layout.Box
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.res.stringResource

data class FilterOption(
    val value: String?,
    val label: String
)

@Composable
fun FilterMenu(
    label: String,
    selectedLabel: String,
    options: List<FilterOption>,
    onSelected: (String?) -> Unit
) {
    var expanded by remember { mutableStateOf(false) }
    Box {
        FilterChip(
            selected = true,
            onClick = { expanded = true },
            label = {
                Text(stringResource(R.string.filter_chip_format, label, selectedLabel))
            }
        )
        DropdownMenu(
            expanded = expanded,
            onDismissRequest = { expanded = false }
        ) {
            options.forEach { option ->
                DropdownMenuItem(
                    text = { Text(option.label) },
                    onClick = {
                        expanded = false
                        onSelected(option.value)
                    }
                )
            }
        }
    }
}

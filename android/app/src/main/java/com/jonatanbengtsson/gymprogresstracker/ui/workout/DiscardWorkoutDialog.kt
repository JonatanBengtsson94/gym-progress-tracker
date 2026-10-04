package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import com.jonatanbengtsson.gymprogresstracker.R

/** Asks before throwing away the workout in progress. */
@Composable
fun DiscardWorkoutDialog(onConfirm: () -> Unit, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.discard_workout_title)) },
        text = { Text(stringResource(R.string.discard_workout_text)) },
        confirmButton = {
            TextButton(onClick = onConfirm) {
                Text(stringResource(R.string.discard_workout_confirm))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.discard_workout_cancel))
            }
        }
    )
}

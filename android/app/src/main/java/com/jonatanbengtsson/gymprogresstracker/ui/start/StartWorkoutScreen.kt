package com.jonatanbengtsson.gymprogresstracker.ui.start

import androidx.annotation.StringRes
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.selection.toggleable
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.LatestWorkout
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExercise
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutSet
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import com.jonatanbengtsson.gymprogresstracker.ui.workout.DiscardWorkoutDialog
import com.jonatanbengtsson.gymprogresstracker.ui.workout.WorkoutTimer
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle
import kotlin.uuid.Uuid

@Composable
fun StartWorkoutScreen(
    onStartNewWorkout: () -> Unit,
    onContinueWorkout: () -> Unit,
    onStartFromTemplate: (WorkoutTemplate) -> Unit,
    modifier: Modifier = Modifier,
    viewModel: StartWorkoutViewModel = viewModel(factory = StartWorkoutViewModel.Factory)
) {
    StartWorkoutContent(
        uiState = viewModel.uiState,
        onRetry = viewModel::loadTemplates,
        onStartNewWorkout = {
            viewModel.startNewWorkout()
            onStartNewWorkout()
        },
        onContinueWorkout = onContinueWorkout,
        onDiscardWorkout = viewModel::discardWorkout,
        onStartFromTemplate = onStartFromTemplate,
        modifier = modifier
    )
}

@Composable
fun StartWorkoutContent(
    uiState: StartWorkoutUiState,
    onRetry: () -> Unit,
    onStartNewWorkout: () -> Unit,
    onContinueWorkout: () -> Unit,
    onDiscardWorkout: () -> Unit,
    onStartFromTemplate: (WorkoutTemplate) -> Unit,
    modifier: Modifier = Modifier
) {
    var confirmDiscard by rememberSaveable { mutableStateOf(false) }

    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(24.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        item {
            Text(
                text = stringResource(R.string.start_workout_title),
                style = MaterialTheme.typography.headlineMedium
            )
        }
        if (uiState.workoutInProgress) {
            item {
                Card(modifier = Modifier.fillMaxWidth()) {
                    Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = stringResource(R.string.start_workout_in_progress),
                                    style = MaterialTheme.typography.labelMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                                Text(
                                    text = uiState.workoutName.ifEmpty { stringResource(R.string.workout_name_placeholder) },
                                    style = MaterialTheme.typography.titleMedium,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis
                                )
                            }
                            uiState.workoutStartedAt?.let {
                                WorkoutTimer(startedAt = it, style = MaterialTheme.typography.titleMedium)
                            }
                        }
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            TextButton(
                                onClick = { confirmDiscard = true },
                                modifier = Modifier.weight(1f),
                                colors = ButtonDefaults.textButtonColors(contentColor = MaterialTheme.colorScheme.error)
                            ) {
                                Text(stringResource(R.string.discard_workout))
                            }
                            Button(onClick = onContinueWorkout, modifier = Modifier.weight(1f)) {
                                Text(stringResource(R.string.start_workout_continue))
                            }
                        }
                    }
                }
            }
        } else {
            item {
                Button(onClick = onStartNewWorkout, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.start_workout_new))
                }
            }
        }
        if (uiState.pendingWorkouts > 0) {
            item {
                Text(
                    text = pluralStringResource(R.plurals.start_workout_pending, uiState.pendingWorkouts, uiState.pendingWorkouts),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
        item {
            Text(
                text = stringResource(R.string.start_workout_from_template),
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(top = 12.dp)
            )
        }

        // Keys must be saveable in instance state, which a Uuid isn't.
        when {
            uiState.templates.isNotEmpty() -> itemsIndexed(uiState.templates, key = { _, template -> template.id.toString() }) { index, template ->
                Column {
                    if (index > 0) HorizontalDivider(modifier = Modifier.padding(bottom = 12.dp))
                    TemplateRow(template = template, onClick = { onStartFromTemplate(template) })
                }
            }
            uiState.isLoading -> item {
                Box(Modifier.fillMaxWidth().padding(24.dp), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            }
            uiState.errorMessage != null -> item {
                Column(
                    modifier = Modifier.fillMaxWidth(),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Text(
                        text = stringResource(uiState.errorMessage),
                        color = MaterialTheme.colorScheme.error,
                        style = MaterialTheme.typography.bodyMedium
                    )
                    OutlinedButton(onClick = onRetry) {
                        Text(stringResource(R.string.start_workout_retry))
                    }
                }
            }
            else -> item {
                Text(
                    text = stringResource(R.string.start_workout_no_templates),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }

    if (confirmDiscard) {
        DiscardWorkoutDialog(
            onConfirm = {
                confirmDiscard = false
                onDiscardWorkout()
            },
            onDismiss = { confirmDiscard = false }
        )
    }
}

@Composable
private fun TemplateRow(template: WorkoutTemplate, onClick: () -> Unit) {
    val dateFormatter = remember {
        DateTimeFormatter.ofLocalizedDate(FormatStyle.MEDIUM).withZone(ZoneId.systemDefault())
    }
    val latestWorkout = template.latestWorkout
    val exercises = latestWorkout?.exercises.orEmpty()
    var expanded by rememberSaveable { mutableStateOf(false) }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 8.dp)
    ) {
        Text(text = template.name, style = MaterialTheme.typography.titleMedium)
        Text(
            text = if (latestWorkout == null) {
                stringResource(R.string.start_workout_never_performed)
            } else {
                stringResource(
                    R.string.start_workout_last_performed,
                    dateFormatter.format(latestWorkout.completedAt)
                )
            },
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        if (exercises.isNotEmpty()) {
            Row(
                modifier = Modifier
                    .toggleable(value = expanded, onValueChange = { expanded = it })
                    .padding(top = 4.dp, bottom = 4.dp, end = 8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                val chevronRotation by animateFloatAsState(if (expanded) 90f else 0f)
                Icon(
                    imageVector = Icons.AutoMirrored.Filled.KeyboardArrowRight,
                    contentDescription = null,
                    modifier = Modifier.size(20.dp).rotate(chevronRotation),
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Text(
                    text = pluralStringResource(
                        R.plurals.start_workout_exercise_count,
                        exercises.size,
                        exercises.size
                    ),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            AnimatedVisibility(visible = expanded) {
                Column(
                    modifier = Modifier.padding(start = 20.dp),
                    verticalArrangement = Arrangement.spacedBy(2.dp)
                ) {
                    exercises.forEach { exercise ->
                        Text(text = exercise.name, style = MaterialTheme.typography.bodyMedium)
                    }
                }
            }
        }
    }
}

private val previewTemplates = listOf(
    WorkoutTemplate(
        id = Uuid.fromLongs(0, 1),
        name = "Push Day",
        latestWorkout = LatestWorkout(
            workoutId = Uuid.fromLongs(0, 7),
            startedAt = Instant.parse("2026-09-30T17:00:00Z"),
            completedAt = Instant.parse("2026-09-30T18:00:00Z"),
            exercises = listOf(
                WorkoutExercise(Uuid.fromLongs(0, 1), "Bench Press (Barbell)", listOf(WorkoutSet(8, 60000))),
                WorkoutExercise(Uuid.fromLongs(0, 2), "Overhead Press (Barbell)", listOf(WorkoutSet(10, 30000)))
            )
        )
    ),
    WorkoutTemplate(id = Uuid.fromLongs(0, 2), name = "Leg Day", latestWorkout = null)
)

@Preview(showBackground = true)
@Composable
fun StartWorkoutContentPreview() {
    GymProgressTrackerTheme {
        StartWorkoutContent(
            uiState = StartWorkoutUiState(templates = previewTemplates),
            onRetry = {},
            onStartNewWorkout = {},
            onContinueWorkout = {},
            onDiscardWorkout = {},
            onStartFromTemplate = {}
        )
    }
}

@Preview(showBackground = true)
@Composable
fun StartWorkoutContentInProgressPreview() {
    GymProgressTrackerTheme {
        StartWorkoutContent(
            uiState = StartWorkoutUiState(
                templates = previewTemplates,
                workoutInProgress = true,
                workoutStartedAt = Instant.now().minusSeconds(1234),
                workoutName = "Push day",
                pendingWorkouts = 2
            ),
            onRetry = {},
            onStartNewWorkout = {},
            onContinueWorkout = {},
            onDiscardWorkout = {},
            onStartFromTemplate = {}
        )
    }
}

@Preview(showBackground = true)
@Composable
fun StartWorkoutContentErrorPreview() {
    GymProgressTrackerTheme {
        StartWorkoutContent(
            uiState = StartWorkoutUiState(errorMessage = R.string.start_workout_error_network),
            onRetry = {},
            onStartNewWorkout = {},
            onContinueWorkout = {},
            onDiscardWorkout = {},
            onStartFromTemplate = {}
        )
    }
}

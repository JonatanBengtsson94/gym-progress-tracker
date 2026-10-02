package com.jonatanbengtsson.gymprogresstracker.ui.start

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import com.jonatanbengtsson.gymprogresstracker.BuildConfig
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.HttpTemplatesApi
import com.jonatanbengtsson.gymprogresstracker.data.LatestWorkout
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExercise
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutSet
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

@Composable
fun StartWorkoutScreen(
    sessionId: String,
    onSessionExpired: () -> Unit,
    onStartNewWorkout: () -> Unit,
    onStartFromTemplate: (WorkoutTemplate) -> Unit,
    modifier: Modifier = Modifier,
    // Keyed by session so logging in again doesn't reuse the previous session's view model.
    viewModel: StartWorkoutViewModel = viewModel(
        key = sessionId,
        factory = viewModelFactory {
            initializer { StartWorkoutViewModel(HttpTemplatesApi(BuildConfig.BASE_URL), sessionId) }
        }
    )
) {
    val uiState = viewModel.uiState

    LaunchedEffect(uiState.sessionExpired) {
        if (uiState.sessionExpired) onSessionExpired()
    }

    StartWorkoutContent(
        uiState = uiState,
        onRetry = viewModel::loadTemplates,
        onStartNewWorkout = onStartNewWorkout,
        onStartFromTemplate = onStartFromTemplate,
        modifier = modifier
    )
}

@Composable
fun StartWorkoutContent(
    uiState: StartWorkoutUiState,
    onRetry: () -> Unit,
    onStartNewWorkout: () -> Unit,
    onStartFromTemplate: (WorkoutTemplate) -> Unit,
    modifier: Modifier = Modifier
) {
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
        item {
            Button(onClick = onStartNewWorkout, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.start_workout_new))
            }
        }
        item {
            Text(
                text = stringResource(R.string.start_workout_from_template),
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(top = 12.dp)
            )
        }

        when {
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
            uiState.templates.isEmpty() -> item {
                Text(
                    text = stringResource(R.string.start_workout_no_templates),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            else -> items(uiState.templates, key = { it.id }) { template ->
                TemplateCard(template = template, onClick = { onStartFromTemplate(template) })
            }
        }
    }
}

@Composable
private fun TemplateCard(template: WorkoutTemplate, onClick: () -> Unit) {
    val dateFormatter = remember {
        DateTimeFormatter.ofLocalizedDate(FormatStyle.MEDIUM).withZone(ZoneId.systemDefault())
    }
    val latestWorkout = template.latestWorkout

    Card(onClick = onClick, modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp)
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
                style = MaterialTheme.typography.bodyMedium
            )
            if (latestWorkout != null && latestWorkout.exercises.isNotEmpty()) {
                Text(
                    text = latestWorkout.exercises.joinToString(" · ") { it.name },
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
            }
        }
    }
}

private val previewTemplates = listOf(
    WorkoutTemplate(
        id = 1,
        name = "Push Day",
        latestWorkout = LatestWorkout(
            workoutId = 7,
            startedAt = Instant.parse("2026-09-30T17:00:00Z"),
            completedAt = Instant.parse("2026-09-30T18:00:00Z"),
            exercises = listOf(
                WorkoutExercise(1, "Bench Press (Barbell)", listOf(WorkoutSet(8, 60000))),
                WorkoutExercise(2, "Overhead Press (Barbell)", listOf(WorkoutSet(10, 30000)))
            )
        )
    ),
    WorkoutTemplate(id = 2, name = "Leg Day", latestWorkout = null)
)

@Preview(showBackground = true)
@Composable
fun StartWorkoutContentPreview() {
    GymProgressTrackerTheme {
        StartWorkoutContent(
            uiState = StartWorkoutUiState(templates = previewTemplates),
            onRetry = {},
            onStartNewWorkout = {},
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
            onStartFromTemplate = {}
        )
    }
}

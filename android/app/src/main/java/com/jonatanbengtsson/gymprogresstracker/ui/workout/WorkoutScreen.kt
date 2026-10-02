package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.activity.compose.BackHandler
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
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Check
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import com.jonatanbengtsson.gymprogresstracker.BuildConfig
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.HttpExercisesApi
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme

@Composable
fun WorkoutScreen(
    sessionId: String,
    onSessionExpired: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
    // Keyed by session so logging in again doesn't reuse the previous session's view model.
    viewModel: WorkoutViewModel = viewModel(
        key = sessionId,
        factory = viewModelFactory {
            initializer { WorkoutViewModel(HttpExercisesApi(BuildConfig.BASE_URL), sessionId) }
        }
    )
) {
    BackHandler(onBack = onBack)

    val uiState = viewModel.uiState

    LaunchedEffect(uiState.sessionExpired) {
        if (uiState.sessionExpired) onSessionExpired()
    }

    WorkoutContent(
        uiState = uiState,
        onRetryExercises = viewModel::loadExercises,
        onExerciseSelected = viewModel::addExercise,
        modifier = modifier
    )
}

@Composable
fun WorkoutContent(
    uiState: WorkoutUiState,
    onRetryExercises: () -> Unit,
    onExerciseSelected: (Exercise) -> Unit,
    modifier: Modifier = Modifier
) {
    var showExercisePicker by rememberSaveable { mutableStateOf(false) }

    if (showExercisePicker) {
        // Composed after WorkoutScreen's BackHandler, so system back closes the picker first.
        BackHandler { showExercisePicker = false }
        ExercisePicker(
            uiState = uiState,
            onRetry = onRetryExercises,
            onExerciseSelected = {
                showExercisePicker = false
                onExerciseSelected(it)
            },
            onBack = { showExercisePicker = false },
            modifier = modifier
        )
        return
    }

    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(24.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        item {
            Text(
                text = stringResource(R.string.workout_title),
                style = MaterialTheme.typography.headlineMedium
            )
        }
        items(uiState.workoutExercises, key = { it.id }) { exercise ->
            Card(modifier = Modifier.fillMaxWidth()) {
                Text(
                    text = exercise.name,
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.padding(16.dp)
                )
            }
        }
        item {
            Button(onClick = { showExercisePicker = true }, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.workout_add_exercise))
            }
        }
    }
}

@Composable
private fun ExercisePicker(
    uiState: WorkoutUiState,
    onRetry: () -> Unit,
    onExerciseSelected: (Exercise) -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 4.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            IconButton(onClick = onBack) {
                Icon(
                    imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                    contentDescription = stringResource(R.string.workout_picker_back)
                )
            }
            Text(
                text = stringResource(R.string.workout_choose_exercise),
                style = MaterialTheme.typography.titleLarge
            )
        }
        HorizontalDivider()

        LazyColumn(
            modifier = Modifier.weight(1f),
            contentPadding = PaddingValues(vertical = 8.dp)
        ) {
            when {
                uiState.isLoadingExercises -> item {
                    Box(Modifier.fillMaxWidth().padding(24.dp), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator()
                    }
                }
                uiState.exercisesErrorMessage != null -> item {
                    Column(
                        modifier = Modifier.fillMaxWidth().padding(24.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(
                            text = stringResource(uiState.exercisesErrorMessage),
                            color = MaterialTheme.colorScheme.error,
                            style = MaterialTheme.typography.bodyMedium
                        )
                        OutlinedButton(onClick = onRetry) {
                            Text(stringResource(R.string.workout_exercises_retry))
                        }
                    }
                }
                else -> items(uiState.exercises, key = { it.id }) { exercise ->
                    ExercisePickerRow(
                        exercise = exercise,
                        added = uiState.workoutExercises.any { it.id == exercise.id },
                        onClick = { onExerciseSelected(exercise) }
                    )
                }
            }
        }
    }
}

/** Exercises already in the workout stay in place but are greyed out and can't be picked again. */
@Composable
private fun ExercisePickerRow(exercise: Exercise, added: Boolean, onClick: () -> Unit) {
    // Material's standard opacity for disabled content.
    val contentColor = if (added) {
        MaterialTheme.colorScheme.onSurface.copy(alpha = 0.38f)
    } else {
        MaterialTheme.colorScheme.onSurface
    }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(enabled = !added, onClick = onClick)
            .padding(horizontal = 24.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = exercise.name,
            style = MaterialTheme.typography.bodyLarge,
            color = contentColor,
            modifier = Modifier.weight(1f)
        )
        if (added) {
            Icon(
                imageVector = Icons.Filled.Check,
                contentDescription = stringResource(R.string.workout_exercise_added),
                tint = contentColor,
                modifier = Modifier.size(20.dp)
            )
        }
    }
}

@Preview(showBackground = true)
@Composable
fun WorkoutContentPreview() {
    GymProgressTrackerTheme {
        WorkoutContent(
            uiState = WorkoutUiState(
                workoutExercises = listOf(Exercise(1, "Bench Press (Barbell)")),
                exercises = listOf(Exercise(1, "Bench Press (Barbell)"), Exercise(2, "Squat (Barbell)"))
            ),
            onRetryExercises = {},
            onExerciseSelected = {}
        )
    }
}

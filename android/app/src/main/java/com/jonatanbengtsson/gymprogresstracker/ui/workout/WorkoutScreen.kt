package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
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
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
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
        // TODO: add the chosen exercise to the workout.
        onExerciseSelected = {},
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

    Column(
        modifier = modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        Text(
            text = stringResource(R.string.workout_title),
            style = MaterialTheme.typography.headlineMedium
        )
        Button(onClick = { showExercisePicker = true }, modifier = Modifier.fillMaxWidth()) {
            Text(stringResource(R.string.workout_add_exercise))
        }
    }

    if (showExercisePicker) {
        ExercisePickerSheet(
            uiState = uiState,
            onRetry = onRetryExercises,
            onExerciseSelected = {
                showExercisePicker = false
                onExerciseSelected(it)
            },
            onDismiss = { showExercisePicker = false }
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ExercisePickerSheet(
    uiState: WorkoutUiState,
    onRetry: () -> Unit,
    onExerciseSelected: (Exercise) -> Unit,
    onDismiss: () -> Unit
) {
    ModalBottomSheet(onDismissRequest = onDismiss) {
        LazyColumn(contentPadding = PaddingValues(bottom = 24.dp)) {
            item {
                Text(
                    text = stringResource(R.string.workout_choose_exercise),
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.padding(horizontal = 24.dp, vertical = 8.dp)
                )
            }

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
                    Text(
                        text = exercise.name,
                        style = MaterialTheme.typography.bodyLarge,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onExerciseSelected(exercise) }
                            .padding(horizontal = 24.dp, vertical = 12.dp)
                    )
                }
            }
        }
    }
}

@Preview(showBackground = true)
@Composable
fun WorkoutContentPreview() {
    GymProgressTrackerTheme {
        WorkoutContent(
            uiState = WorkoutUiState(
                exercises = listOf(Exercise(1, "Bench Press (Barbell)"), Exercise(2, "Squat (Barbell)"))
            ),
            onRetryExercises = {},
            onExerciseSelected = {}
        )
    }
}

package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.activity.compose.BackHandler
import androidx.annotation.StringRes
import androidx.compose.animation.animateColorAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsFocusedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.wrapContentWidth
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Clear
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.IconButtonDefaults
import androidx.compose.material3.IconToggleButton
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.SetEntry
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExerciseEntry
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import java.time.Instant
import kotlin.uuid.Uuid

@Composable
fun WorkoutScreen(
    onWorkoutSaved: () -> Unit,
    onWorkoutDiscarded: () -> Unit,
    modifier: Modifier = Modifier,
    viewModel: WorkoutViewModel = viewModel(factory = WorkoutViewModel.Factory)
) {
    LaunchedEffect(viewModel.uiState.isSaved) {
        if (viewModel.uiState.isSaved) onWorkoutSaved()
    }

    WorkoutContent(
        uiState = viewModel.uiState,
        onNameChange = viewModel::updateName,
        onExerciseSelected = viewModel::addExercise,
        onRemoveExercise = viewModel::removeExercise,
        onAddSet = viewModel::addSet,
        onRemoveSet = viewModel::removeSet,
        onToggleSetCompleted = viewModel::toggleSetCompleted,
        onWeightChange = viewModel::updateWeight,
        onRepsChange = viewModel::updateReps,
        onSave = viewModel::saveWorkout,
        onDiscard = {
            viewModel.discardWorkout()
            onWorkoutDiscarded()
        },
        modifier = modifier
    )
}

@Composable
fun WorkoutContent(
    uiState: WorkoutUiState,
    onNameChange: (String) -> Unit,
    onExerciseSelected: (Exercise) -> Unit,
    onRemoveExercise: (exerciseId: Uuid) -> Unit,
    onAddSet: (exerciseId: Uuid) -> Unit,
    onRemoveSet: (exerciseId: Uuid, setIndex: Int) -> Unit,
    onToggleSetCompleted: (exerciseId: Uuid, setIndex: Int) -> Unit,
    onWeightChange: (exerciseId: Uuid, setIndex: Int, weightKg: String) -> Unit,
    onRepsChange: (exerciseId: Uuid, setIndex: Int, reps: String) -> Unit,
    onSave: () -> Unit,
    onDiscard: () -> Unit,
    modifier: Modifier = Modifier
) {
    var showExercisePicker by rememberSaveable { mutableStateOf(false) }
    var confirmDiscard by rememberSaveable { mutableStateOf(false) }
    var confirmSave by rememberSaveable { mutableStateOf(false) }
    // Kept as text, since a Uuid can't be saved in instance state.
    var exerciseIdToConfirmRemoval by rememberSaveable { mutableStateOf<String?>(null) }

    if (showExercisePicker) {
        // Added after the navigation's back handling, so system back closes the picker before the screen.
        BackHandler { showExercisePicker = false }
        ExercisePicker(
            uiState = uiState,
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
        modifier = modifier.fillMaxSize().imePadding(),
        contentPadding = PaddingValues(24.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        item {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                WorkoutNameField(name = uiState.name, onNameChange = onNameChange, modifier = Modifier.weight(1f))
                uiState.startedAt?.let {
                    WorkoutTimer(startedAt = it, style = MaterialTheme.typography.titleLarge)
                }
            }
        }
        if (uiState.isLoadingWorkout) {
            item {
                Box(Modifier.fillMaxWidth().padding(24.dp), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            }
        } else {
            // Keys must be saveable in instance state, which a Uuid isn't.
            items(uiState.workoutExercises, key = { it.exercise.id.toString() }) { entry ->
                val exerciseId = entry.exercise.id
                WorkoutExerciseCard(
                    entry = entry,
                    onRemove = {
                        if (entry.hasEnteredSets) exerciseIdToConfirmRemoval = exerciseId.toString() else onRemoveExercise(exerciseId)
                    },
                    onAddSet = { onAddSet(exerciseId) },
                    onRemoveSet = { onRemoveSet(exerciseId, it) },
                    onToggleCompleted = { onToggleSetCompleted(exerciseId, it) },
                    onWeightChange = { setIndex, weightKg -> onWeightChange(exerciseId, setIndex, weightKg) },
                    onRepsChange = { setIndex, reps -> onRepsChange(exerciseId, setIndex, reps) }
                )
            }
            item {
                Button(onClick = { showExercisePicker = true }, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.workout_add_exercise))
                }
            }
            if (uiState.workoutExercises.isNotEmpty()) {
                item {
                    WorkoutActions(
                        isSaving = uiState.isSaving,
                        errorMessage = uiState.saveErrorMessage,
                        onSave = { if (uiState.uncompletedSetsLeftOut > 0) confirmSave = true else onSave() },
                        onDiscard = { confirmDiscard = true }
                    )
                }
            }
        }
    }

    if (confirmSave) {
        val leftOut = uiState.uncompletedSetsLeftOut
        AlertDialog(
            onDismissRequest = { confirmSave = false },
            title = { Text(stringResource(R.string.workout_save_incomplete_title)) },
            text = { Text(pluralStringResource(R.plurals.workout_save_incomplete_text, leftOut, leftOut)) },
            confirmButton = {
                TextButton(onClick = {
                    confirmSave = false
                    onSave()
                }) {
                    Text(stringResource(R.string.workout_save_incomplete_confirm))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmSave = false }) {
                    Text(stringResource(R.string.workout_save_incomplete_cancel))
                }
            }
        )
    }

    if (confirmDiscard) {
        DiscardWorkoutDialog(
            onConfirm = {
                confirmDiscard = false
                onDiscard()
            },
            onDismiss = { confirmDiscard = false }
        )
    }

    uiState.workoutExercises.find { it.exercise.id.toString() == exerciseIdToConfirmRemoval }?.let { entry ->
        AlertDialog(
            onDismissRequest = { exerciseIdToConfirmRemoval = null },
            title = { Text(stringResource(R.string.workout_remove_exercise_title, entry.exercise.name)) },
            text = { Text(stringResource(R.string.workout_remove_exercise_text)) },
            confirmButton = {
                TextButton(onClick = {
                    exerciseIdToConfirmRemoval = null
                    onRemoveExercise(entry.exercise.id)
                }) {
                    Text(stringResource(R.string.workout_remove_exercise_confirm))
                }
            },
            dismissButton = {
                TextButton(onClick = { exerciseIdToConfirmRemoval = null }) {
                    Text(stringResource(R.string.workout_remove_exercise_cancel))
                }
            }
        )
    }
}

@Composable
private fun WorkoutActions(isSaving: Boolean, @StringRes errorMessage: Int?, onSave: () -> Unit, onDiscard: () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            TextButton(
                onClick = onDiscard,
                enabled = !isSaving,
                modifier = Modifier.weight(1f),
                colors = ButtonDefaults.textButtonColors(contentColor = MaterialTheme.colorScheme.error)
            ) {
                Text(stringResource(R.string.discard_workout))
            }
            Button(onClick = onSave, enabled = !isSaving, modifier = Modifier.weight(1f)) {
                if (isSaving) {
                    CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp, color = LocalContentColor.current)
                } else {
                    Text(stringResource(R.string.workout_save))
                }
            }
        }
        if (errorMessage != null) {
            Text(
                text = stringResource(errorMessage),
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodyMedium,
                textAlign = TextAlign.Center
            )
        }
    }
}

@Composable
private fun WorkoutNameField(name: String, onNameChange: (String) -> Unit, modifier: Modifier) {
    val focusManager = LocalFocusManager.current
    val description = stringResource(R.string.workout_name_description)
    val style = MaterialTheme.typography.headlineMedium
    val interactionSource = remember { MutableInteractionSource() }
    val focused by interactionSource.collectIsFocusedAsState()
    val underlineColor = if (focused) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline
    val underlineWidth = if (focused) 2.dp else 1.dp

    BasicTextField(
        value = name,
        onValueChange = onNameChange,
        modifier = modifier.semantics { contentDescription = description },
        interactionSource = interactionSource,
        textStyle = style.copy(color = MaterialTheme.colorScheme.onSurface),
        keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences, imeAction = ImeAction.Done),
        keyboardActions = KeyboardActions(onDone = { focusManager.clearFocus() }),
        singleLine = true,
        cursorBrush = SolidColor(MaterialTheme.colorScheme.primary),
        decorationBox = { innerTextField ->
            Row(
                modifier = Modifier
                    .wrapContentWidth(Alignment.Start)
                    .drawBehind {
                        val y = size.height - underlineWidth.toPx() / 2
                        drawLine(underlineColor, Offset(0f, y), Offset(size.width, y), underlineWidth.toPx())
                    }
                    .padding(bottom = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                Box(modifier = Modifier.weight(1f, fill = false)) {
                    if (name.isEmpty()) {
                        Text(
                            text = stringResource(R.string.workout_name_placeholder),
                            style = style,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    innerTextField()
                }
                Icon(
                    imageVector = Icons.Filled.Edit,
                    contentDescription = null,
                    modifier = Modifier.size(20.dp),
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    )
}

@Composable
private fun WorkoutExerciseCard(
    entry: WorkoutExerciseEntry,
    onRemove: () -> Unit,
    onAddSet: () -> Unit,
    onRemoveSet: (setIndex: Int) -> Unit,
    onToggleCompleted: (setIndex: Int) -> Unit,
    onWeightChange: (setIndex: Int, weightKg: String) -> Unit,
    onRepsChange: (setIndex: Int, reps: String) -> Unit
) {
    val defaultColors = CardDefaults.cardColors()
    val containerColor by animateColorAsState(
        if (entry.allSetsCompleted) MaterialTheme.colorScheme.primaryContainer else defaultColors.containerColor
    )
    val contentColor by animateColorAsState(
        if (entry.allSetsCompleted) MaterialTheme.colorScheme.onPrimaryContainer else defaultColors.contentColor
    )

    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = containerColor, contentColor = contentColor)
    ) {
        val rowPadding = PaddingValues(start = 16.dp, end = 4.dp)
        Column(modifier = Modifier.padding(vertical = 4.dp)) {
            Row(modifier = Modifier.padding(rowPadding), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = entry.exercise.name,
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.weight(1f)
                )
                IconButton(onClick = onRemove) {
                    Icon(
                        imageVector = Icons.Filled.Close,
                        contentDescription = stringResource(R.string.workout_remove_exercise, entry.exercise.name)
                    )
                }
            }
            if (entry.sets.isNotEmpty()) {
                Row(
                    modifier = Modifier.padding(rowPadding).padding(top = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    SetColumnHeader(stringResource(R.string.workout_set_number), Modifier.width(SET_NUMBER_WIDTH))
                    SetColumnHeader(stringResource(R.string.workout_set_weight), Modifier.weight(1f))
                    SetColumnHeader(stringResource(R.string.workout_set_reps), Modifier.weight(1f))
                    Spacer(Modifier.width(96.dp))
                }
            }
            entry.sets.forEachIndexed { index, set ->
                key(set.id) {
                    val setNumber = index + 1
                    val completedColor = MaterialTheme.colorScheme.primaryContainer
                    val rowColor by animateColorAsState(
                        if (set.completed) completedColor else completedColor.copy(alpha = 0f)
                    )
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        color = rowColor,
                        contentColor = if (set.completed) MaterialTheme.colorScheme.onPrimaryContainer else LocalContentColor.current
                    ) {
                        Row(
                            modifier = Modifier.padding(rowPadding),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            Text(
                                text = "$setNumber",
                                style = MaterialTheme.typography.titleSmall,
                                textAlign = TextAlign.Center,
                                modifier = Modifier.width(SET_NUMBER_WIDTH)
                            )
                            SetField(
                                value = set.weightKg,
                                onValueChange = { onWeightChange(index, it) },
                                keyboardType = KeyboardType.Decimal,
                                description = stringResource(R.string.workout_set_weight_description, setNumber),
                                completed = set.completed,
                                modifier = Modifier.weight(1f)
                            )
                            SetField(
                                value = set.reps,
                                onValueChange = { onRepsChange(index, it) },
                                keyboardType = KeyboardType.Number,
                                description = stringResource(R.string.workout_set_reps_description, setNumber),
                                completed = set.completed,
                                modifier = Modifier.weight(1f)
                            )
                            Row {
                                IconToggleButton(
                                    checked = set.completed,
                                    onCheckedChange = { onToggleCompleted(index) },
                                    enabled = set.completed || set.canComplete,
                                    colors = IconButtonDefaults.iconToggleButtonColors(
                                        checkedContainerColor = Color.Transparent,
                                        checkedContentColor = MaterialTheme.colorScheme.onPrimaryContainer
                                    )
                                ) {
                                    Icon(
                                        imageVector = Icons.Filled.Check,
                                        contentDescription = stringResource(R.string.workout_complete_set, setNumber),
                                        modifier = Modifier.size(20.dp)
                                    )
                                }
                                IconButton(onClick = { onRemoveSet(index) }) {
                                    Icon(
                                        imageVector = Icons.Filled.Close,
                                        contentDescription = stringResource(R.string.workout_remove_set, setNumber),
                                        modifier = Modifier.size(20.dp)
                                    )
                                }
                            }
                        }
                    }
                }
            }
            TextButton(
                onClick = onAddSet,
                modifier = Modifier.align(Alignment.CenterHorizontally)
            ) {
                Icon(
                    imageVector = Icons.Filled.Add,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp)
                )
                Text(
                    text = stringResource(R.string.workout_add_set),
                    modifier = Modifier.padding(start = 8.dp)
                )
            }
        }
    }
}

private val SET_NUMBER_WIDTH = 28.dp

@Composable
private fun SetColumnHeader(text: String, modifier: Modifier) {
    Text(
        text = text,
        style = MaterialTheme.typography.labelMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        textAlign = TextAlign.Center,
        modifier = modifier
    )
}

@Composable
private fun SetField(
    value: String,
    onValueChange: (String) -> Unit,
    keyboardType: KeyboardType,
    description: String,
    completed: Boolean,
    modifier: Modifier
) {
    val surface = MaterialTheme.colorScheme.surface
    val backgroundColor by animateColorAsState(if (completed) surface.copy(alpha = 0f) else surface)
    val textColor by animateColorAsState(
        if (completed) LocalContentColor.current else MaterialTheme.colorScheme.onSurface
    )

    BasicTextField(
        value = value,
        onValueChange = onValueChange,
        modifier = modifier
            .height(40.dp)
            .background(backgroundColor, MaterialTheme.shapes.small)
            .semantics { contentDescription = description },
        textStyle = MaterialTheme.typography.bodyLarge.copy(
            color = textColor,
            textAlign = TextAlign.Center
        ),
        keyboardOptions = KeyboardOptions(keyboardType = keyboardType, imeAction = ImeAction.Next),
        singleLine = true,
        cursorBrush = SolidColor(MaterialTheme.colorScheme.primary),
        decorationBox = { innerTextField ->
            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { innerTextField() }
        }
    )
}

@Composable
private fun ExercisePicker(
    uiState: WorkoutUiState,
    onExerciseSelected: (Exercise) -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    var query by rememberSaveable { mutableStateOf("") }
    val matchingExercises = remember(uiState.exercises, query) {
        uiState.exercises.filter { it.matches(query) }
    }
    val keyboardController = LocalSoftwareKeyboardController.current

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
        OutlinedTextField(
            value = query,
            onValueChange = { query = it },
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp).padding(bottom = 8.dp),
            placeholder = { Text(stringResource(R.string.workout_search_exercises)) },
            leadingIcon = { Icon(imageVector = Icons.Filled.Search, contentDescription = null) },
            trailingIcon = {
                if (query.isNotEmpty()) {
                    IconButton(onClick = { query = "" }) {
                        Icon(
                            imageVector = Icons.Filled.Clear,
                            contentDescription = stringResource(R.string.workout_search_clear)
                        )
                    }
                }
            },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
            keyboardActions = KeyboardActions(onSearch = { keyboardController?.hide() })
        )
        HorizontalDivider()

        LazyColumn(
            modifier = Modifier.weight(1f).imePadding(),
            contentPadding = PaddingValues(vertical = 8.dp)
        ) {
                when {
                uiState.exercises.isEmpty() -> item {
                    Text(
                        text = stringResource(R.string.workout_no_exercises),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(24.dp)
                    )
                }
                matchingExercises.isEmpty() && query.isNotBlank() -> item {
                    Text(
                        text = stringResource(R.string.workout_search_no_results, query.trim()),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(24.dp)
                    )
                }
                else -> items(matchingExercises, key = { it.id.toString() }) { exercise ->
                    ExercisePickerRow(
                        exercise = exercise,
                        added = uiState.workoutExercises.any { it.exercise.id == exercise.id },
                        onClick = { onExerciseSelected(exercise) }
                    )
                }
            }
        }
    }
}

/** True when every word of [query] appears in the name, ignoring case and word order. */
internal fun Exercise.matches(query: String): Boolean =
    query.split(' ').filter { it.isNotBlank() }.all { name.contains(it, ignoreCase = true) }

/** An exercise already in the workout is greyed out and can't be picked again. */
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
                startedAt = Instant.now().minusSeconds(1234),
                name = "Push day",
                workoutExercises = listOf(
                    WorkoutExerciseEntry(
                        exercise = Exercise(Uuid.fromLongs(0, 1), "Bench Press (Barbell)"),
                        sets = listOf(
                            SetEntry(weightKg = "60", reps = "8", completed = true),
                            SetEntry(weightKg = "62,5", reps = "6", id = 1)
                        )
                    )
                ),
                exercises = listOf(Exercise(Uuid.fromLongs(0, 1), "Bench Press (Barbell)"), Exercise(Uuid.fromLongs(0, 2), "Squat (Barbell)"))
            ),
            onNameChange = {},
            onExerciseSelected = {},
            onRemoveExercise = {},
            onAddSet = {},
            onRemoveSet = { _, _ -> },
            onToggleSetCompleted = { _, _ -> },
            onWeightChange = { _, _, _ -> },
            onRepsChange = { _, _, _ -> },
            onSave = {},
            onDiscard = {}
        )
    }
}

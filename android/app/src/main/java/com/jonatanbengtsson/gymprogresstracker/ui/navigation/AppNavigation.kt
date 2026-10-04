package com.jonatanbengtsson.gymprogresstracker.ui.navigation

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.lifecycle.viewmodel.navigation3.rememberViewModelStoreNavEntryDecorator
import androidx.navigation3.runtime.NavKey
import androidx.navigation3.runtime.entryProvider
import androidx.navigation3.runtime.rememberNavBackStack
import androidx.navigation3.runtime.rememberSaveableStateHolderNavEntryDecorator
import androidx.navigation3.ui.NavDisplay
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutScreen
import com.jonatanbengtsson.gymprogresstracker.ui.workout.WorkoutScreen
import kotlinx.serialization.Serializable

/** The screens of a logged-in session. */
sealed interface Screen : NavKey {
    @Serializable
    data object StartWorkout : Screen

    @Serializable
    data object Workout : Screen
}

/**
 * Navigates between the screens of a logged-in session, starting at [Screen.StartWorkout]. Each screen
 * gets its own view models, cleared when the screen is popped or this leaves the composition.
 */
@Composable
fun AppNavigation(modifier: Modifier = Modifier) {
    val backStack = rememberNavBackStack(Screen.StartWorkout)

    // A second tap can land while the screen it opens is still animating in.
    fun navigateTo(screen: Screen) {
        if (backStack.lastOrNull() != screen) backStack.add(screen)
    }

    NavDisplay(
        backStack = backStack,
        modifier = modifier,
        entryDecorators = listOf(
            rememberSaveableStateHolderNavEntryDecorator(),
            rememberViewModelStoreNavEntryDecorator()
        ),
        entryProvider = entryProvider {
            entry<Screen.StartWorkout> {
                StartWorkoutScreen(
                    onStartNewWorkout = { navigateTo(Screen.Workout) },
                    onContinueWorkout = { navigateTo(Screen.Workout) },
                    // TODO: open the workout screen prefilled from the template.
                    onStartFromTemplate = {}
                )
            }
            entry<Screen.Workout> {
                WorkoutScreen()
            }
        }
    )
}

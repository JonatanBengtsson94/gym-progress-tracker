package com.jonatanbengtsson.gymprogresstracker.ui.navigation

import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.lifecycle.viewmodel.navigation3.rememberViewModelStoreNavEntryDecorator
import androidx.navigation3.runtime.NavKey
import androidx.navigation3.runtime.entryProvider
import androidx.navigation3.runtime.rememberNavBackStack
import androidx.navigation3.runtime.rememberSaveableStateHolderNavEntryDecorator
import androidx.navigation3.ui.NavDisplay
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.ui.login.LoginScreen
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutScreen
import com.jonatanbengtsson.gymprogresstracker.ui.user.UserScreen
import com.jonatanbengtsson.gymprogresstracker.ui.workout.WorkoutScreen
import kotlinx.serialization.Serializable

/** The screens of the app. */
sealed interface Screen : NavKey {
    @Serializable
    data object StartWorkout : Screen

    @Serializable
    data object Workout : Screen

    @Serializable
    data object User : Screen

    @Serializable
    data object Login : Screen
}

/**
 * Navigates between the app's screens, starting at [Screen.StartWorkout]. A navigation bar switches
 * between the start screen and [Screen.User], and is hidden on the other screens. [Screen.Login] opens
 * over the screen that needs a session and closes once the user has logged in. Each screen gets its own
 * view models, cleared when the screen is popped or this leaves the composition.
 */
@Composable
fun AppNavigation(modifier: Modifier = Modifier) {
    val backStack = rememberNavBackStack(Screen.StartWorkout)
    val currentScreen = backStack.last()

    // A second tap can land while the screen it opens is still animating in.
    fun navigateTo(screen: Screen) {
        if (backStack.lastOrNull() != screen) backStack.add(screen)
    }

    // The start screen stays at the bottom of the back stack, so back from any other tab returns to it.
    fun selectTab(screen: Screen) {
        if (currentScreen == screen) return
        while (backStack.size > 1) backStack.removeAt(backStack.lastIndex)
        if (screen != Screen.StartWorkout) backStack.add(screen)
    }

    Scaffold(
        modifier = modifier,
        bottomBar = {
            if (currentScreen == Screen.StartWorkout || currentScreen == Screen.User) {
                NavigationBar {
                    NavigationBarItem(
                        selected = currentScreen == Screen.StartWorkout,
                        onClick = { selectTab(Screen.StartWorkout) },
                        icon = { Icon(painterResource(R.drawable.ic_fitness_center), contentDescription = null) },
                        label = { Text(stringResource(R.string.navigation_workout)) }
                    )
                    NavigationBarItem(
                        selected = currentScreen == Screen.User,
                        onClick = { selectTab(Screen.User) },
                        icon = { Icon(Icons.Filled.Person, contentDescription = null) },
                        label = { Text(stringResource(R.string.navigation_user)) }
                    )
                }
            }
        }
    ) { innerPadding ->
        NavDisplay(
            backStack = backStack,
            modifier = Modifier.padding(innerPadding).consumeWindowInsets(innerPadding),
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
                    WorkoutScreen(
                        onWorkoutSaved = { backStack.remove(Screen.Workout) },
                        onWorkoutDiscarded = { backStack.remove(Screen.Workout) }
                    )
                }
                entry<Screen.User> {
                    UserScreen(onLogIn = { navigateTo(Screen.Login) })
                }
                entry<Screen.Login> {
                    LoginScreen(onLoggedIn = { backStack.remove(Screen.Login) })
                }
            }
        )
    }
}

package com.jonatanbengtsson.gymprogresstracker.ui.navigation

import androidx.navigation3.runtime.NavKey

/** True when the screen on top is one the navigation bar switches between. */
internal val List<NavKey>.showsNavigationBar: Boolean
    get() = lastOrNull() == Screen.StartWorkout || lastOrNull() == Screen.User

/** Opens [screen] on top. Ignored if it's already on top, as when a second tap lands while it animates in. */
internal fun MutableList<NavKey>.navigateTo(screen: Screen) {
    if (lastOrNull() != screen) add(screen)
}

/**
 * Switches to the navigation bar's [screen]. The start screen stays at the bottom of the back stack,
 * so back from any other tab returns to it.
 */
internal fun MutableList<NavKey>.selectTab(screen: Screen) {
    if (lastOrNull() == screen) return
    while (size > 1) removeAt(lastIndex)
    if (screen != Screen.StartWorkout) add(screen)
}

/** Closes [Screen.Login] once the user has logged in. A first login is the only screen, so the start screen takes its place. */
internal fun MutableList<NavKey>.closeLogin() {
    if (size == 1) set(0, Screen.StartWorkout) else remove(Screen.Login)
}

package com.jonatanbengtsson.gymprogresstracker.ui.navigation

import androidx.navigation3.runtime.NavKey
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class BackStackTest {

    private fun backStack(vararg screens: Screen) = mutableListOf<NavKey>(*screens)

    @Test
    fun `the navigation bar shows on the screens it switches between`() {
        assertTrue(backStack(Screen.StartWorkout).showsNavigationBar)
        assertTrue(backStack(Screen.StartWorkout, Screen.User).showsNavigationBar)
    }

    @Test
    fun `the navigation bar hides during a workout and on the login`() {
        assertFalse(backStack(Screen.StartWorkout, Screen.Workout).showsNavigationBar)
        assertFalse(backStack(Screen.StartWorkout, Screen.User, Screen.Login).showsNavigationBar)
        assertFalse(backStack(Screen.Login).showsNavigationBar)
    }

    @Test
    fun `navigating opens the screen on top`() {
        val backStack = backStack(Screen.StartWorkout)

        backStack.navigateTo(Screen.Workout)

        assertEquals(backStack(Screen.StartWorkout, Screen.Workout), backStack)
    }

    @Test
    fun `navigating to the screen already on top is ignored`() {
        val backStack = backStack(Screen.StartWorkout, Screen.Workout)

        backStack.navigateTo(Screen.Workout)

        assertEquals(backStack(Screen.StartWorkout, Screen.Workout), backStack)
    }

    @Test
    fun `the user tab opens over the start screen, so back returns to it`() {
        val backStack = backStack(Screen.StartWorkout)

        backStack.selectTab(Screen.User)

        assertEquals(backStack(Screen.StartWorkout, Screen.User), backStack)
    }

    @Test
    fun `the workout tab returns to the start screen`() {
        val backStack = backStack(Screen.StartWorkout, Screen.User)

        backStack.selectTab(Screen.StartWorkout)

        assertEquals(backStack(Screen.StartWorkout), backStack)
    }

    @Test
    fun `selecting the tab already shown is ignored`() {
        val backStack = backStack(Screen.StartWorkout, Screen.User)

        backStack.selectTab(Screen.User)

        assertEquals(backStack(Screen.StartWorkout, Screen.User), backStack)
    }

    @Test
    fun `a login opened over another screen closes back to it`() {
        val backStack = backStack(Screen.StartWorkout, Screen.User, Screen.Login)

        backStack.closeLogin()

        assertEquals(backStack(Screen.StartWorkout, Screen.User), backStack)
    }

    @Test
    fun `the first login is replaced by the start screen`() {
        val backStack = backStack(Screen.Login)

        backStack.closeLogin()

        assertEquals(backStack(Screen.StartWorkout), backStack)
    }
}

package com.jonatanbengtsson.gymprogresstracker.ui.workout

import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ExerciseMatchesTest {

    private val inclineDumbbell = Exercise(1, "Incline Bench Press (Dumbbell)")

    @Test
    fun `blank query matches everything`() {
        assertTrue(inclineDumbbell.matches(""))
        assertTrue(inclineDumbbell.matches("   "))
    }

    @Test
    fun `matching ignores case`() {
        assertTrue(inclineDumbbell.matches("BENCH"))
    }

    @Test
    fun `partial words match`() {
        assertTrue(inclineDumbbell.matches("incl"))
    }

    @Test
    fun `every word must appear, in any order`() {
        assertTrue(inclineDumbbell.matches("dumbbell incline"))
        assertTrue(inclineDumbbell.matches("  incline   dumb "))
        assertFalse(inclineDumbbell.matches("incline barbell"))
    }
}

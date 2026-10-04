package com.jonatanbengtsson.gymprogresstracker.ui.workout

import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.Duration

class FormatElapsedTest {

    @Test
    fun `under an hour shows minutes and seconds`() {
        assertEquals("0:00", formatElapsed(Duration.ZERO))
        assertEquals("0:07", formatElapsed(Duration.ofSeconds(7)))
        assertEquals("59:59", formatElapsed(Duration.ofSeconds(59 * 60 + 59)))
    }

    @Test
    fun `from an hour on shows hours too`() {
        assertEquals("1:00:00", formatElapsed(Duration.ofHours(1)))
        assertEquals("1:05:09", formatElapsed(Duration.ofSeconds(3600 + 5 * 60 + 9)))
        assertEquals("25:00:00", formatElapsed(Duration.ofHours(25)))
    }

    @Test
    fun `parts of a second are dropped`() {
        assertEquals("0:01", formatElapsed(Duration.ofMillis(1999)))
    }
}

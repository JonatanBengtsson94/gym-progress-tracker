package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.compose.material3.LocalTextStyle
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.produceState
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.TextStyle
import kotlinx.coroutines.delay
import java.time.Duration
import java.time.Instant

/** The time since [startedAt], ticking each second. */
@Composable
fun WorkoutTimer(startedAt: Instant, modifier: Modifier = Modifier, style: TextStyle = LocalTextStyle.current) {
    val elapsed by produceState(elapsedSince(startedAt), startedAt) {
        while (true) {
            value = elapsedSince(startedAt)
            // Waits for the next whole second, so the timer ticks in step with the clock.
            delay(1000 - value.toMillis() % 1000)
        }
    }
    // Tabular figures keep the text from shifting as the digits change.
    Text(text = formatElapsed(elapsed), modifier = modifier, style = style.copy(fontFeatureSettings = "tnum"))
}

/** Never negative, in case the device's clock was turned back since the workout started. */
private fun elapsedSince(startedAt: Instant): Duration =
    Duration.between(startedAt, Instant.now()).coerceAtLeast(Duration.ZERO)

/** Formats [elapsed] as m:ss, or h:mm:ss from an hour on. */
fun formatElapsed(elapsed: Duration): String {
    val seconds = elapsed.toSeconds()
    val hours = seconds / 3600
    val minutes = seconds % 3600 / 60
    return if (hours > 0) {
        "%d:%02d:%02d".format(hours, minutes, seconds % 60)
    } else {
        "%d:%02d".format(minutes, seconds % 60)
    }
}

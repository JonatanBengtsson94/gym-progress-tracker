package com.jonatanbengtsson.gymprogresstracker

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.jonatanbengtsson.gymprogresstracker.data.SessionState
import com.jonatanbengtsson.gymprogresstracker.ui.login.LoginScreen
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutScreen
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import com.jonatanbengtsson.gymprogresstracker.ui.workout.WorkoutScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        val sessionRepository = (application as GymProgressTrackerApplication).container.sessionRepository
        setContent {
            GymProgressTrackerTheme {
                val session by sessionRepository.session.collectAsStateWithLifecycle()
                var inWorkout by rememberSaveable { mutableStateOf(false) }

                LaunchedEffect(session) {
                    if (session == SessionState.LoggedOut) inWorkout = false
                }

                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    when (val currentSession = session) {
                        // Brief enough that the window background is all that shows.
                        SessionState.Loading -> {}
                        SessionState.LoggedOut -> LoginScreen(modifier = Modifier.padding(innerPadding))
                        is SessionState.LoggedIn -> if (inWorkout) {
                            WorkoutScreen(
                                sessionId = currentSession.sessionId,
                                onBack = { inWorkout = false },
                                modifier = Modifier.padding(innerPadding)
                            )
                        } else {
                            StartWorkoutScreen(
                                sessionId = currentSession.sessionId,
                                onStartNewWorkout = { inWorkout = true },
                                onContinueWorkout = { inWorkout = true },
                                // TODO: open the workout screen prefilled from the template.
                                onStartFromTemplate = {},
                                modifier = Modifier.padding(innerPadding)
                            )
                        }
                    }
                }
            }
        }
    }
}

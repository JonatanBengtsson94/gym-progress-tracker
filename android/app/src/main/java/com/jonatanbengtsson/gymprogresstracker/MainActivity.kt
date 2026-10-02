package com.jonatanbengtsson.gymprogresstracker

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.jonatanbengtsson.gymprogresstracker.ui.login.LoginScreen
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutScreen
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import com.jonatanbengtsson.gymprogresstracker.ui.workout.WorkoutScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            GymProgressTrackerTheme {
                var sessionId by rememberSaveable { mutableStateOf<String?>(null) }
                var inWorkout by rememberSaveable { mutableStateOf(false) }

                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    val currentSessionId = sessionId
                    if (currentSessionId == null) {
                        LoginScreen(
                            onLoggedIn = { sessionId = it },
                            modifier = Modifier.padding(innerPadding)
                        )
                    } else if (inWorkout) {
                        WorkoutScreen(
                            onBack = { inWorkout = false },
                            modifier = Modifier.padding(innerPadding)
                        )
                    } else {
                        StartWorkoutScreen(
                            sessionId = currentSessionId,
                            onSessionExpired = { sessionId = null },
                            onStartNewWorkout = { inWorkout = true },
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

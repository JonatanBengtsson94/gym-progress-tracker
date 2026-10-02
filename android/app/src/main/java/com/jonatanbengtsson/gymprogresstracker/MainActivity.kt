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

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            GymProgressTrackerTheme {
                var sessionId by rememberSaveable { mutableStateOf<String?>(null) }

                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    val currentSessionId = sessionId
                    if (currentSessionId == null) {
                        LoginScreen(
                            onLoggedIn = { sessionId = it },
                            modifier = Modifier.padding(innerPadding)
                        )
                    } else {
                        StartWorkoutScreen(
                            sessionId = currentSessionId,
                            onSessionExpired = { sessionId = null },
                            // TODO: open the workout screen once it exists.
                            onStartNewWorkout = {},
                            onStartFromTemplate = {},
                            modifier = Modifier.padding(innerPadding)
                        )
                    }
                }
            }
        }
    }
}

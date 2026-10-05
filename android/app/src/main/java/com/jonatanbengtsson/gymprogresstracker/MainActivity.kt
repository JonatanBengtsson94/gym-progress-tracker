package com.jonatanbengtsson.gymprogresstracker

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.jonatanbengtsson.gymprogresstracker.data.SessionState
import com.jonatanbengtsson.gymprogresstracker.data.owner
import com.jonatanbengtsson.gymprogresstracker.ui.navigation.AppNavigation
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        val sessionRepository = (application as GymProgressTrackerApplication).container.sessionRepository
        setContent {
            GymProgressTrackerTheme {
                val session by sessionRepository.session.collectAsStateWithLifecycle()

                // Waits for the saved session, to know whether the device has an owner yet.
                if (session != SessionState.Loading) {
                    AppNavigation(startAtLogin = session.owner == null, modifier = Modifier.fillMaxSize())
                }
            }
        }
    }
}

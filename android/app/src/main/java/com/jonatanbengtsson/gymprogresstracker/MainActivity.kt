package com.jonatanbengtsson.gymprogresstracker

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.LocalViewModelStoreOwner
import androidx.lifecycle.viewmodel.compose.rememberViewModelStoreOwner
import com.jonatanbengtsson.gymprogresstracker.data.SessionState
import com.jonatanbengtsson.gymprogresstracker.data.owner
import com.jonatanbengtsson.gymprogresstracker.ui.login.LoginScreen
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

                when {
                    // Waits for the saved session, so the first requests are sent with it.
                    session == SessionState.Loading -> {}
                    // Scoped to the first login, so its view model is cleared once the device has an owner.
                    session.owner == null -> CompositionLocalProvider(
                        LocalViewModelStoreOwner provides rememberViewModelStoreOwner()
                    ) {
                        Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                            // The app leaves this screen by itself once the session is saved.
                            LoginScreen(onLoggedIn = {}, modifier = Modifier.padding(innerPadding))
                        }
                    }
                    else -> AppNavigation(modifier = Modifier.fillMaxSize())
                }
            }
        }
    }
}

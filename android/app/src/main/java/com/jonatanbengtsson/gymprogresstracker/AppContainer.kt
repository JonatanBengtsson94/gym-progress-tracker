package com.jonatanbengtsson.gymprogresstracker

import android.content.Context
import androidx.lifecycle.ViewModelProvider.AndroidViewModelFactory.Companion.APPLICATION_KEY
import androidx.lifecycle.viewmodel.CreationExtras
import androidx.room.Room
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.RoomActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.local.GymDatabase
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob

/** The objects that live as long as the app, shared by the view models that need them. */
class AppContainer(context: Context) {

    private val applicationScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    private val database = Room.databaseBuilder(context, GymDatabase::class.java, "gym-progress-tracker.db").build()

    // Created with the app, so the saved workout is loading before any screen asks for it.
    val activeWorkoutRepository: ActiveWorkoutRepository =
        RoomActiveWorkoutRepository(database.activeWorkoutDao(), applicationScope)
}

/** The app's [AppContainer], for view model factories. */
val CreationExtras.appContainer: AppContainer
    get() = (checkNotNull(this[APPLICATION_KEY]) as GymProgressTrackerApplication).container

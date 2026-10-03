package com.jonatanbengtsson.gymprogresstracker

import android.content.Context
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import androidx.datastore.preferences.preferencesDataStoreFile
import androidx.lifecycle.ViewModelProvider.AndroidViewModelFactory.Companion.APPLICATION_KEY
import androidx.lifecycle.viewmodel.CreationExtras
import androidx.room.Room
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.AuthApi
import com.jonatanbengtsson.gymprogresstracker.data.DataStoreSessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesRepository
import com.jonatanbengtsson.gymprogresstracker.data.HttpAuthApi
import com.jonatanbengtsson.gymprogresstracker.data.HttpExercisesApi
import com.jonatanbengtsson.gymprogresstracker.data.HttpTemplatesApi
import com.jonatanbengtsson.gymprogresstracker.data.RoomActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.RoomExercisesRepository
import com.jonatanbengtsson.gymprogresstracker.data.RoomTemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.SessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.local.GymDatabase
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob

/** The objects that live as long as the app, shared by the view models that need them. */
class AppContainer(context: Context) {

    private val applicationScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    private val database = Room.databaseBuilder(context, GymDatabase::class.java, "gym-progress-tracker.db")
        // Version 1 identified exercises by numbers the server no longer uses, so its data can't be kept.
        .fallbackToDestructiveMigrationFrom(dropAllTables = true, 1)
        .build()

    val authApi: AuthApi = HttpAuthApi(BuildConfig.BASE_URL)

    // Created with the app, so the saved workout is loading before any screen asks for it.
    val activeWorkoutRepository: ActiveWorkoutRepository =
        RoomActiveWorkoutRepository(database.activeWorkoutDao(), applicationScope)

    val sessionRepository: SessionRepository = DataStoreSessionRepository(
        PreferenceDataStoreFactory.create { context.preferencesDataStoreFile("session") },
        activeWorkoutRepository,
        applicationScope
    )

    val exercisesRepository: ExercisesRepository =
        RoomExercisesRepository(HttpExercisesApi(BuildConfig.BASE_URL), database.exerciseDao())

    val templatesRepository: TemplatesRepository =
        RoomTemplatesRepository(HttpTemplatesApi(BuildConfig.BASE_URL), database.templateDao())
}

/** The app's [AppContainer], for view model factories. */
val CreationExtras.appContainer: AppContainer
    get() = (checkNotNull(this[APPLICATION_KEY]) as GymProgressTrackerApplication).container

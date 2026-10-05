package com.jonatanbengtsson.gymprogresstracker

import android.content.Context
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import androidx.datastore.preferences.preferencesDataStoreFile
import androidx.lifecycle.ViewModelProvider.AndroidViewModelFactory.Companion.APPLICATION_KEY
import androidx.lifecycle.viewmodel.CreationExtras
import androidx.room.Room
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.ApiWorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.data.ApiClient
import com.jonatanbengtsson.gymprogresstracker.data.AuthApi
import com.jonatanbengtsson.gymprogresstracker.data.DataStoreSessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.DefaultSyncRepository
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesRepository
import com.jonatanbengtsson.gymprogresstracker.data.HttpAuthApi
import com.jonatanbengtsson.gymprogresstracker.data.HttpExercisesApi
import com.jonatanbengtsson.gymprogresstracker.data.HttpTemplatesApi
import com.jonatanbengtsson.gymprogresstracker.data.HttpWorkoutsApi
import com.jonatanbengtsson.gymprogresstracker.data.PendingWorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.data.RoomActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.RoomExercisesRepository
import com.jonatanbengtsson.gymprogresstracker.data.RoomPendingWorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.data.RoomTemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.SessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.SyncRepository
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesApi
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.data.local.GymDatabase
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob

/** The objects that live as long as the app, shared by the view models that need them. */
class AppContainer(context: Context) {

    private val applicationScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    private val database = Room.databaseBuilder(context, GymDatabase::class.java, "gym-progress-tracker.db").build()

    val activeWorkoutRepository: ActiveWorkoutRepository =
        RoomActiveWorkoutRepository(database.activeWorkoutDao(), applicationScope)

    private val pendingWorkoutsRepository: PendingWorkoutsRepository =
        RoomPendingWorkoutsRepository(database.pendingWorkoutDao())

    val sessionRepository: SessionRepository = DataStoreSessionRepository(
        PreferenceDataStoreFactory.create { context.preferencesDataStoreFile("session") },
        applicationScope
    )

    private val apiClient = ApiClient(BuildConfig.BASE_URL, sessionRepository)

    val authApi: AuthApi = HttpAuthApi(apiClient)

    private val templatesApi: TemplatesApi = HttpTemplatesApi(apiClient)

    val exercisesRepository: ExercisesRepository = RoomExercisesRepository(HttpExercisesApi(apiClient), database.exerciseDao())

    val templatesRepository: TemplatesRepository = RoomTemplatesRepository(templatesApi, database.templateDao(), pendingWorkoutsRepository)

    val workoutsRepository: WorkoutsRepository =
        ApiWorkoutsRepository(HttpWorkoutsApi(apiClient), templatesApi, pendingWorkoutsRepository, templatesRepository)

    val syncRepository: SyncRepository = DefaultSyncRepository(workoutsRepository, templatesRepository, exercisesRepository)
}

/** The app's [AppContainer], for view model factories. */
val CreationExtras.appContainer: AppContainer
    get() = (checkNotNull(this[APPLICATION_KEY]) as GymProgressTrackerApplication).container

package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.Query
import androidx.room.Transaction
import kotlinx.coroutines.flow.Flow
import kotlin.uuid.Uuid

@Dao
interface PendingWorkoutDao {

    /** Oldest first, by when they were completed. */
    @Transaction
    @Query("SELECT * FROM pending_workout ORDER BY completed_at")
    fun observeWorkouts(): Flow<List<PendingWorkoutWithSets>>

    /** Stores the workout and its sets, replacing any stored under the same id. */
    @Transaction
    suspend fun replaceWorkout(workout: PendingWorkoutEntity, sets: List<PendingWorkoutSetEntity>) {
        deleteWorkout(workout.workoutId)
        insertWorkout(workout)
        insertSets(sets)
    }

    /** Deletes its sets too. */
    @Query("DELETE FROM pending_workout WHERE workout_id = :workoutId")
    suspend fun deleteWorkout(workoutId: Uuid)

    /** Deletes their sets too. */
    @Query("DELETE FROM pending_workout")
    suspend fun deleteWorkouts()

    @Insert
    suspend fun insertWorkout(workout: PendingWorkoutEntity)

    @Insert
    suspend fun insertSets(sets: List<PendingWorkoutSetEntity>)
}

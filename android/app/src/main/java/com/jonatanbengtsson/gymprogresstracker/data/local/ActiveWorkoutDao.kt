package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.Query
import androidx.room.Transaction

@Dao
interface ActiveWorkoutDao {

    @Transaction
    @Query("SELECT * FROM active_workout_exercise ORDER BY position")
    suspend fun getWorkout(): List<ActiveWorkoutExerciseWithSets>

    @Transaction
    suspend fun replaceWorkout(exercises: List<ActiveWorkoutExerciseEntity>, sets: List<ActiveWorkoutSetEntity>) {
        deleteExercises()
        insertExercises(exercises)
        insertSets(sets)
    }

    /** Deletes their sets too. */
    @Query("DELETE FROM active_workout_exercise")
    suspend fun deleteExercises()

    @Insert
    suspend fun insertExercises(exercises: List<ActiveWorkoutExerciseEntity>)

    @Insert
    suspend fun insertSets(sets: List<ActiveWorkoutSetEntity>)
}

package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import androidx.room.Transaction

@Dao
interface ActiveWorkoutDao {

    /** Null until a workout has been saved. */
    @Query("SELECT * FROM active_workout")
    suspend fun getWorkout(): ActiveWorkoutEntity?

    @Transaction
    @Query("SELECT * FROM active_workout_exercise ORDER BY position")
    suspend fun getExercises(): List<ActiveWorkoutExerciseWithSets>

    @Transaction
    suspend fun replaceWorkout(
        workout: ActiveWorkoutEntity,
        exercises: List<ActiveWorkoutExerciseEntity>,
        sets: List<ActiveWorkoutSetEntity>
    ) {
        insertWorkout(workout)
        deleteExercises()
        insertExercises(exercises)
        insertSets(sets)
    }

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertWorkout(workout: ActiveWorkoutEntity)

    /** Deletes their sets too. */
    @Query("DELETE FROM active_workout_exercise")
    suspend fun deleteExercises()

    @Insert
    suspend fun insertExercises(exercises: List<ActiveWorkoutExerciseEntity>)

    @Insert
    suspend fun insertSets(sets: List<ActiveWorkoutSetEntity>)
}

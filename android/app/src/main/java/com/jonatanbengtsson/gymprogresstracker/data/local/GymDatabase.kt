package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Database
import androidx.room.RoomDatabase

@Database(entities = [ActiveWorkoutExerciseEntity::class, ActiveWorkoutSetEntity::class], version = 1)
abstract class GymDatabase : RoomDatabase() {
    abstract fun activeWorkoutDao(): ActiveWorkoutDao
}

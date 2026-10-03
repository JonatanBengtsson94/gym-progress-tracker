package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Database
import androidx.room.RoomDatabase
import androidx.room.TypeConverters

@Database(
    entities = [
        ActiveWorkoutExerciseEntity::class,
        ActiveWorkoutSetEntity::class,
        ExerciseEntity::class,
        TemplateEntity::class,
        TemplateSetEntity::class
    ],
    version = 2
)
@TypeConverters(Converters::class)
abstract class GymDatabase : RoomDatabase() {
    abstract fun activeWorkoutDao(): ActiveWorkoutDao
    abstract fun exerciseDao(): ExerciseDao
    abstract fun templateDao(): TemplateDao
}

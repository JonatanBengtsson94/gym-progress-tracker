package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.TypeConverter
import java.time.Instant
import kotlin.uuid.Uuid

class Converters {
    @TypeConverter
    fun uuidToString(uuid: Uuid): String = uuid.toString()

    @TypeConverter
    fun stringToUuid(value: String): Uuid = Uuid.parse(value)

    @TypeConverter
    fun instantToEpochMillis(instant: Instant): Long = instant.toEpochMilli()

    @TypeConverter
    fun epochMillisToInstant(millis: Long): Instant = Instant.ofEpochMilli(millis)
}

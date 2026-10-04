package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.json.JSONArray
import org.json.JSONObject
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Before
import org.junit.Test
import java.time.Instant
import kotlin.uuid.Uuid

class HttpWorkoutsApiTest {

    private lateinit var server: MockWebServer
    private lateinit var api: HttpWorkoutsApi

    private val workoutId = Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d20")
    private val workout = FinishedWorkout(
        templateId = Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d10"),
        templateName = "Push Day",
        startedAt = Instant.parse("2026-09-30T17:00:00Z"),
        completedAt = Instant.parse("2026-09-30T18:00:00Z"),
        exercises = listOf(
            FinishedExercise(Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d01"), listOf(WorkoutSet(8, 60000), WorkoutSet(6, 62500)))
        )
    )

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        val client = ApiClient(server.url("").toString().removeSuffix("/"), FakeSessionRepository(SessionState.LoggedIn("session-123")))
        api = HttpWorkoutsApi(client)
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `putWorkout puts a workout under its template to its id`() = runTest {
        server.enqueue(MockResponse().setResponseCode(201).setBody(contract("workout.response.json")))

        assertEquals(ApiResult.Success(Unit), api.putWorkout(workoutId, workout))

        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/workouts/$workoutId", request.path)
        assertEquals(JSONObject(contract("put_workout.request.json")).toKotlin(), JSONObject(request.body.readUtf8()).toKotlin())
    }

    @Test
    fun `a workout without a template id names a new template`() = runTest {
        server.enqueue(MockResponse().setResponseCode(201).setBody(contract("workout.response.json")))

        api.putWorkout(workoutId, workout.copy(templateId = null))

        val body = JSONObject(server.takeRequest().body.readUtf8())
        assertEquals("Push Day", body.getString("template_name"))
        assertFalse(body.has("template_id"))
    }

    @Test
    fun `a rejected workout is a server error`() = runTest {
        server.enqueue(MockResponse().setResponseCode(400).setBody("Invalid request body"))

        assertEquals(ApiResult.ServerError, api.putWorkout(workoutId, workout))
    }

    /** The JSON as maps and lists, which compare by content regardless of key order. */
    private fun Any.toKotlin(): Any = when (this) {
        is JSONObject -> keys().asSequence().associateWith { get(it).toKotlin() }
        is JSONArray -> List(length()) { get(it).toKotlin() }
        else -> this
    }
}

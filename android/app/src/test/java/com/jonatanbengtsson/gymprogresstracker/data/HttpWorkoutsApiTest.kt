package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.json.JSONObject
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import java.time.Instant
import kotlin.uuid.Uuid

class HttpWorkoutsApiTest {

    private lateinit var server: MockWebServer
    private lateinit var api: HttpWorkoutsApi

    private val workoutId = Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d20")
    private val templateId = Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d10")
    private val workout = FinishedWorkout(
        name = "Push Day",
        startedAt = Instant.parse("2026-09-30T17:00:00Z"),
        completedAt = Instant.parse("2026-09-30T18:00:00Z"),
        exercises = listOf(
            WorkoutExercise(
                Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d01"),
                "Bench Press (Barbell)",
                listOf(WorkoutSet(8, 60000), WorkoutSet(6, 62500))
            )
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

        assertEquals(ApiResult.Success(Unit), api.putWorkout(workoutId, templateId, workout))

        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/workouts/$workoutId", request.path)
        assertEquals(JSONObject(contract("put_workout.request.json")).toKotlin(), JSONObject(request.body.readUtf8()).toKotlin())
    }

    @Test
    fun `a rejected workout is a server error`() = runTest {
        server.enqueue(MockResponse().setResponseCode(400).setBody("Invalid request body"))

        assertEquals(ApiResult.ServerError, api.putWorkout(workoutId, templateId, workout))
    }
}

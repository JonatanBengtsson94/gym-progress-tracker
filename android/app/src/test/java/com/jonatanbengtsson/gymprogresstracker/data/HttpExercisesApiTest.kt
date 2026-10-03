package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import kotlin.uuid.Uuid

class HttpExercisesApiTest {

    private lateinit var server: MockWebServer
    private lateinit var api: HttpExercisesApi

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        api = HttpExercisesApi(server.url("").toString().removeSuffix("/"))
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `getExercises sends the session as a bearer token`() = runTest {
        server.enqueue(MockResponse().setBody("""{"exercises":[]}"""))

        api.getExercises("session-123")

        val request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/exercises", request.path)
        assertEquals("Bearer session-123", request.getHeader("Authorization"))
    }

    @Test
    fun `200 parses exercises in response order`() = runTest {
        server.enqueue(MockResponse().setBody(contract("get_exercises.response.json")))

        val expected = ExercisesResult.Success(
            listOf(
                Exercise(Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d01"), "Bench Press (Barbell)"),
                Exercise(Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d02"), "Zercher Squat")
            )
        )
        assertEquals(expected, api.getExercises("session-123"))
    }

    @Test
    fun `200 with an exercise id that isn't a UUID returns ServerError`() = runTest {
        server.enqueue(MockResponse().setBody("""{"exercises": [{"exercise_id": 1, "exercise_name": "Squat (Barbell)"}]}"""))

        assertEquals(ExercisesResult.ServerError, api.getExercises("session-123"))
    }

    @Test
    fun `401 returns SessionExpired`() = runTest {
        server.enqueue(MockResponse().setResponseCode(401))

        assertEquals(ExercisesResult.SessionExpired, api.getExercises("expired"))
    }

    @Test
    fun `500 returns ServerError`() = runTest {
        server.enqueue(MockResponse().setResponseCode(500))

        assertEquals(ExercisesResult.ServerError, api.getExercises("session-123"))
    }

    @Test
    fun `200 with malformed JSON returns ServerError`() = runTest {
        server.enqueue(MockResponse().setBody("not json"))

        assertEquals(ExercisesResult.ServerError, api.getExercises("session-123"))
    }

    @Test
    fun `200 without exercises returns ServerError`() = runTest {
        server.enqueue(MockResponse().setBody("""{"templates":[]}"""))

        assertEquals(ExercisesResult.ServerError, api.getExercises("session-123"))
    }

    @Test
    fun `unreachable server returns NetworkError`() = runTest {
        server.shutdown()

        assertEquals(ExercisesResult.NetworkError, api.getExercises("session-123"))
    }
}

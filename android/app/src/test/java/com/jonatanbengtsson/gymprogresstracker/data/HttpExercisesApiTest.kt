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
        val client = ApiClient(server.url("").toString().removeSuffix("/"), FakeSessionRepository(SessionState.LoggedIn("session-123")))
        api = HttpExercisesApi(client)
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `getExercises gets the exercises`() = runTest {
        server.enqueue(MockResponse().setBody("""{"exercises":[]}"""))

        api.getExercises()

        val request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/exercises", request.path)
    }

    @Test
    fun `exercises are parsed in response order`() = runTest {
        server.enqueue(MockResponse().setBody(contract("get_exercises.response.json")))

        val expected = ApiResult.Success(
            listOf(
                Exercise(Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d01"), "Bench Press (Barbell)"),
                Exercise(Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d02"), "Zercher Squat")
            )
        )
        assertEquals(expected, api.getExercises())
    }

    @Test
    fun `an exercise id that isn't a UUID is a server error`() = runTest {
        server.enqueue(MockResponse().setBody("""{"exercises": [{"exercise_id": 1, "exercise_name": "Squat (Barbell)"}]}"""))

        assertEquals(ApiResult.ServerError, api.getExercises())
    }

    @Test
    fun `a response without exercises is a server error`() = runTest {
        server.enqueue(MockResponse().setBody("""{"templates":[]}"""))

        assertEquals(ApiResult.ServerError, api.getExercises())
    }
}

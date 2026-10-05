package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import kotlin.uuid.Uuid

class HttpExercisesApiTest {

    private lateinit var server: MockWebServer
    private lateinit var api: HttpExercisesApi
    private val sessionRepository = FakeSessionRepository(SessionState.LoggedIn("session-123", "alice"))

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        val client = ApiClient(server.url("").toString().removeSuffix("/"), sessionRepository)
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
                Exercise(Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d04"), "Landmine Press"),
                Exercise(Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d02"), "Zercher Squat")
            )
        )
        assertEquals(expected, api.getExercises())
    }

    @Test
    fun `getGlobalExercises gets them without a session`() = runTest {
        sessionRepository.session.value = SessionState.LoggedOut(null)
        server.enqueue(MockResponse().setBody("""{"exercises":[]}"""))

        api.getGlobalExercises()

        val request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/global-exercises", request.path)
        assertNull(request.getHeader("Authorization"))
    }

    @Test
    fun `getGlobalExercises sends no credentials even when logged in`() = runTest {
        server.enqueue(MockResponse().setBody("""{"exercises":[]}"""))

        api.getGlobalExercises()

        assertNull(server.takeRequest().getHeader("Authorization"))
    }

    @Test
    fun `global exercises are parsed in response order`() = runTest {
        server.enqueue(MockResponse().setBody(contract("get_global_exercises.response.json")))

        val expected = ApiResult.Success(
            listOf(
                Exercise(Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d01"), "Bench Press (Barbell)"),
                Exercise(Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d03"), "Squat (Barbell)")
            )
        )
        assertEquals(expected, api.getGlobalExercises())
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

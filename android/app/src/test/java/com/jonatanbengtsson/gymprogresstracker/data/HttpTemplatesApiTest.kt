package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import java.time.Instant
import kotlin.uuid.Uuid

class HttpTemplatesApiTest {

    private lateinit var server: MockWebServer
    private lateinit var api: HttpTemplatesApi

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        val client = ApiClient(server.url("").toString().removeSuffix("/"), FakeSessionRepository(SessionState.LoggedIn("session-123")))
        api = HttpTemplatesApi(client)
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `getTemplates gets the templates`() = runTest {
        server.enqueue(MockResponse().setBody("""{"templates":[]}"""))

        api.getTemplates()

        val request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/templates", request.path)
    }

    @Test
    fun `templates with and without a latest workout are parsed in order`() = runTest {
        server.enqueue(MockResponse().setBody(contract("get_templates.response.json")))

        val benchPressId = Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d01")
        val expected = ApiResult.Success(
            listOf(
                WorkoutTemplate(
                    id = Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d10"),
                    name = "Push Day",
                    latestWorkout = LatestWorkout(
                        workoutId = Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d20"),
                        startedAt = Instant.parse("2026-09-30T17:00:00Z"),
                        completedAt = Instant.parse("2026-09-30T18:00:00Z"),
                        exercises = listOf(
                            WorkoutExercise(benchPressId, "Bench Press (Barbell)", listOf(WorkoutSet(8, 60000), WorkoutSet(6, 62500)))
                        )
                    )
                ),
                WorkoutTemplate(id = Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d11"), name = "Leg Day", latestWorkout = null)
            )
        )
        assertEquals(expected, api.getTemplates())
    }

    @Test
    fun `timestamps with fractional seconds and offsets are parsed`() = runTest {
        server.enqueue(
            MockResponse().setBody(
                """
                {"templates": [{"template_id": "0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d10", "template_name": "Push Day",
                  "latest_workout": {"workout_id": "0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d20",
                    "started_at": "2024-05-08T09:00:00.123456789Z",
                    "completed_at": "2024-05-08T12:00:00+02:00",
                    "exercises": []}}]}
                """.trimIndent()
            )
        )

        val result = api.getTemplates() as ApiResult.Success
        val latest = result.value.single().latestWorkout!!

        assertEquals(Instant.parse("2024-05-08T09:00:00.123456789Z"), latest.startedAt)
        assertEquals(Instant.parse("2024-05-08T10:00:00Z"), latest.completedAt)
    }

    @Test
    fun `an id that isn't a UUID is a server error`() = runTest {
        server.enqueue(MockResponse().setBody("""{"templates": [{"template_id": 1, "template_name": "Push Day", "latest_workout": null}]}"""))

        assertEquals(ApiResult.ServerError, api.getTemplates())
    }

    @Test
    fun `an invalid timestamp is a server error`() = runTest {
        server.enqueue(
            MockResponse().setBody(
                """
                {"templates": [{"template_id": "0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d10", "template_name": "Push Day",
                  "latest_workout": {"workout_id": "0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d20", "started_at": "yesterday",
                    "completed_at": "2024-05-08T10:00:00Z", "exercises": []}}]}
                """.trimIndent()
            )
        )

        assertEquals(ApiResult.ServerError, api.getTemplates())
    }

    @Test
    fun `a response without templates is a server error`() = runTest {
        server.enqueue(MockResponse().setBody("""{"workouts":[]}"""))

        assertEquals(ApiResult.ServerError, api.getTemplates())
    }
}

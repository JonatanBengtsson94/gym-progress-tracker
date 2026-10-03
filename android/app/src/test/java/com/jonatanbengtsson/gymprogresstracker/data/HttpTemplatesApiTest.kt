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
        api = HttpTemplatesApi(server.url("").toString().removeSuffix("/"))
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `getTemplates sends the session as a bearer token`() = runTest {
        server.enqueue(MockResponse().setBody("""{"templates":[]}"""))

        api.getTemplates("session-123")

        val request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/templates", request.path)
        assertEquals("Bearer session-123", request.getHeader("Authorization"))
    }

    @Test
    fun `200 parses templates with and without a latest workout in order`() = runTest {
        server.enqueue(MockResponse().setBody(contract("get_templates.response.json")))

        val benchPressId = Uuid.parse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d01")
        val expected = TemplatesResult.Success(
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
        assertEquals(expected, api.getTemplates("session-123"))
    }

    @Test
    fun `200 with an id that isn't a UUID returns ServerError`() = runTest {
        server.enqueue(MockResponse().setBody("""{"templates": [{"template_id": 1, "template_name": "Push Day", "latest_workout": null}]}"""))

        assertEquals(TemplatesResult.ServerError, api.getTemplates("session-123"))
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

        val result = api.getTemplates("session-123") as TemplatesResult.Success
        val latest = result.templates.single().latestWorkout!!

        assertEquals(Instant.parse("2024-05-08T09:00:00.123456789Z"), latest.startedAt)
        assertEquals(Instant.parse("2024-05-08T10:00:00Z"), latest.completedAt)
    }

    @Test
    fun `401 returns SessionExpired`() = runTest {
        server.enqueue(MockResponse().setResponseCode(401))

        assertEquals(TemplatesResult.SessionExpired, api.getTemplates("expired"))
    }

    @Test
    fun `500 returns ServerError`() = runTest {
        server.enqueue(MockResponse().setResponseCode(500))

        assertEquals(TemplatesResult.ServerError, api.getTemplates("session-123"))
    }

    @Test
    fun `200 with malformed JSON returns ServerError`() = runTest {
        server.enqueue(MockResponse().setBody("not json"))

        assertEquals(TemplatesResult.ServerError, api.getTemplates("session-123"))
    }

    @Test
    fun `200 without templates returns ServerError`() = runTest {
        server.enqueue(MockResponse().setBody("""{"workouts":[]}"""))

        assertEquals(TemplatesResult.ServerError, api.getTemplates("session-123"))
    }

    @Test
    fun `200 with an invalid timestamp returns ServerError`() = runTest {
        server.enqueue(
            MockResponse().setBody(
                """
                {"templates": [{"template_id": "0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d10", "template_name": "Push Day",
                  "latest_workout": {"workout_id": "0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d20", "started_at": "yesterday",
                    "completed_at": "2024-05-08T10:00:00Z", "exercises": []}}]}
                """.trimIndent()
            )
        )

        assertEquals(TemplatesResult.ServerError, api.getTemplates("session-123"))
    }

    @Test
    fun `unreachable server returns NetworkError`() = runTest {
        server.shutdown()

        assertEquals(TemplatesResult.NetworkError, api.getTemplates("session-123"))
    }
}

package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import org.json.JSONObject
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import kotlin.uuid.Uuid

class ApiClientTest {

    private lateinit var server: MockWebServer
    private val sessionRepository = FakeSessionRepository(SessionState.LoggedIn("session-123", "alice"))
    private lateinit var client: ApiClient

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        client = ApiClient(server.url("").toString().removeSuffix("/"), sessionRepository)
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    private suspend fun getName() = client.send("GET", "/things") { it.getString("name") }

    @Test
    fun `a request is sent as the logged-in user`() = runTest {
        server.enqueue(MockResponse().setBody("""{"name":"Bench"}"""))

        assertEquals(ApiResult.Success("Bench"), getName())

        val request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/things", request.path)
        assertEquals("Bearer session-123", request.getHeader("Authorization"))
    }

    @Test
    fun `a body is sent as JSON`() = runTest {
        server.enqueue(MockResponse().setResponseCode(201).setBody("""{"name":"Bench"}"""))

        val result = client.send("POST", "/things", JSONObject().put("name", "Bench")) { it.getString("name") }

        assertEquals(ApiResult.Success("Bench"), result)
        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("application/json", request.getHeader("Content-Type"))
        assertEquals("Bench", JSONObject(request.body.readUtf8()).getString("name"))
    }

    @Test
    fun `without a session nothing is sent`() = runTest {
        sessionRepository.session.value = SessionState.LoggedOut("alice")

        assertEquals(ApiResult.Unauthorized, getName())
        assertEquals(0, server.requestCount)
    }

    @Test
    fun `401 ends the session the request was sent with`() = runTest {
        server.enqueue(MockResponse().setResponseCode(401))

        assertEquals(ApiResult.Unauthorized, getName())
        assertEquals(SessionState.LoggedOut("alice"), sessionRepository.session.value)
    }

    @Test
    fun `401 for a session that has been replaced keeps the new one`() = runTest {
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                // The user logs in again while the request is on its way.
                sessionRepository.session.value = SessionState.LoggedIn("session-456", "alice")
                return MockResponse().setResponseCode(401)
            }
        }

        assertEquals(ApiResult.Unauthorized, getName())
        assertEquals(SessionState.LoggedIn("session-456", "alice"), sessionRepository.session.value)
    }

    @Test
    fun `other error statuses are server errors and keep the session`() = runTest {
        for (status in listOf(400, 404, 409, 500)) {
            server.enqueue(MockResponse().setResponseCode(status))

            assertEquals(ApiResult.ServerError, getName())
        }
        assertEquals(SessionState.LoggedIn("session-123", "alice"), sessionRepository.session.value)
    }

    @Test
    fun `a body that isn't JSON is a server error`() = runTest {
        server.enqueue(MockResponse().setBody("not json"))

        assertEquals(ApiResult.ServerError, getName())
    }

    @Test
    fun `a body parse can't read is a server error`() = runTest {
        server.enqueue(MockResponse().setBody("""{"id":"not a uuid"}"""))

        assertEquals(ApiResult.ServerError, client.send("GET", "/things") { Uuid.parse(it.getString("id")) })
    }

    @Test
    fun `unreachable server is a network error`() = runTest {
        server.shutdown()

        assertEquals(ApiResult.NetworkError, getName())
    }

    @Test
    fun `a request without a session carries no credentials and a 401 ends nothing`() = runTest {
        server.enqueue(MockResponse().setResponseCode(401))

        assertEquals(ApiResult.Unauthorized, client.sendWithoutSession("POST", "/login", JSONObject()) { it.getString("session_id") })
        assertNull(server.takeRequest().getHeader("Authorization"))
        assertEquals(SessionState.LoggedIn("session-123", "alice"), sessionRepository.session.value)
    }
}

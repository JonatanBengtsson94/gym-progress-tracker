package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.json.JSONObject
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test

class HttpAuthApiTest {

    private lateinit var server: MockWebServer
    private lateinit var api: HttpAuthApi

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        api = HttpAuthApi(ApiClient(server.url("").toString().removeSuffix("/"), FakeSessionRepository()))
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `login posts credentials as JSON to the login endpoint`() = runTest {
        server.enqueue(MockResponse().setBody("""{"session_id":"abc"}"""))

        api.login("alice", "s3cret")

        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/login", request.path)
        assertEquals("application/json", request.getHeader("Content-Type"))
        val body = JSONObject(request.body.readUtf8())
        assertEquals("alice", body.getString("username"))
        assertEquals("s3cret", body.getString("password"))
    }

    @Test
    fun `credentials with special characters are JSON-escaped`() = runTest {
        server.enqueue(MockResponse().setBody("""{"session_id":"abc"}"""))

        api.login("al\"ice", "p\\ass\nwörd")

        val body = JSONObject(server.takeRequest().body.readUtf8())
        assertEquals("al\"ice", body.getString("username"))
        assertEquals("p\\ass\nwörd", body.getString("password"))
    }

    @Test
    fun `200 with session id returns Success`() = runTest {
        server.enqueue(MockResponse().setBody("""{"session_id":"session-123"}"""))

        assertEquals(LoginResult.Success("session-123"), api.login("alice", "pw"))
    }

    @Test
    fun `401 returns InvalidCredentials`() = runTest {
        server.enqueue(MockResponse().setResponseCode(401))

        assertEquals(LoginResult.InvalidCredentials, api.login("alice", "wrong"))
    }

    @Test
    fun `500 returns ServerError`() = runTest {
        server.enqueue(MockResponse().setResponseCode(500))

        assertEquals(LoginResult.ServerError, api.login("alice", "pw"))
    }

    @Test
    fun `other client errors return ServerError`() = runTest {
        server.enqueue(MockResponse().setResponseCode(400))

        assertEquals(LoginResult.ServerError, api.login("alice", "pw"))
    }

    @Test
    fun `200 with malformed JSON returns ServerError`() = runTest {
        server.enqueue(MockResponse().setBody("not json"))

        assertEquals(LoginResult.ServerError, api.login("alice", "pw"))
    }

    @Test
    fun `200 without session id returns ServerError`() = runTest {
        server.enqueue(MockResponse().setBody("""{"token":"abc"}"""))

        assertEquals(LoginResult.ServerError, api.login("alice", "pw"))
    }

    @Test
    fun `unreachable server returns NetworkError`() = runTest {
        server.shutdown()

        assertEquals(LoginResult.NetworkError, api.login("alice", "pw"))
    }
}

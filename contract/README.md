# API contract

Example requests and responses for the endpoints the app uses. The backend's
tests check that its handlers produce and accept exactly these, and the app's
tests should parse and send them, so the two can't drift apart.

Change a file here when the API changes, and both sides' tests will say what
needs updating.

package handlers_test

/*
func TestPing(t *testing.T) {
	cfg := &config.Config{
		AppName: "lema",
	}
	logger := zap.NewNop()

	h := handlers.New(cfg, logger)

	// Create an HTTP GET request.
	req := httptest.NewRequest("GET", "/ping", nil)
	rr := httptest.NewRecorder()

	h.Ping(rr, req)

	// Assert that the HTTP status code is 200 as well as the header.
	assert.Equal(t, http.StatusOK, rr.Code, "expected status code 200")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "expected Content-Type application/json")

	// Unmarshal the JSON response.
	var resp handlers.ResponseFormat
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to unmarshal response body")

	expectedMessage := "welcome to lema"
	assert.Equal(t, expectedMessage, resp.Message, "unexpected response message")

	assert.True(t, resp.Status, "expected response status to be true")
}
*/
